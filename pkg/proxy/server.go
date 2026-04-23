package proxy

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/klauspost/compress/zstd"
	"github.com/lkarlslund/tokenrouter/pkg/cache"
	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/conversations"
	"github.com/lkarlslund/tokenrouter/pkg/logstore"
	"github.com/lkarlslund/tokenrouter/pkg/logutil"
	"github.com/lkarlslund/tokenrouter/pkg/pricing"
	"github.com/lkarlslund/tokenrouter/pkg/version"
	"golang.org/x/crypto/acme/autocert"
)

type Server struct {
	store                 *config.ServerConfigStore
	resolver              *ProviderResolver
	stats                 *StatsStore
	pricing               *pricing.Manager
	conversations         *conversations.Store
	logs                  *logstore.Store
	providerHealthChecker *ProviderHealthChecker
	adminHandler          *AdminHandler
	httpServer            *http.Server
	listenerMu            sync.Mutex
	httpListeners         map[string]net.Listener
	modelsCachePath       string
	modelsCached          atomic.Pointer[[]ModelCard]
	activeProxyRequests   atomic.Int64
	draining              atomic.Bool
	modelRefreshMu        sync.Mutex
	modelRefreshLast      map[string]time.Time
	modelRefreshRunning   map[string]bool
}

const accessTokenCleanupInterval = 1 * time.Minute
const modelNotFoundRefreshDebounce = 2 * time.Minute
const defaultUpstreamRetryCount = 2
const defaultUpstreamRetryBackoff = 250 * time.Millisecond

func NewServer(configPath string, cfg *config.ServerConfig) (*Server, error) {
	store := config.NewServerConfigStore(configPath, cfg)
	resolver := NewProviderResolver(store)
	stats := NewPersistentStatsStore(10000, config.DefaultUsageStatsPath())

	s := &Server{
		store:           store,
		resolver:        resolver,
		stats:           stats,
		httpListeners:   map[string]net.Listener{},
		modelsCachePath: config.DefaultModelsCachePath(),
		conversations: conversations.NewStore(config.DefaultConversationsPath(), conversations.Settings{
			Enabled:    cfg.Conversations.Enabled,
			MaxItems:   cfg.Conversations.MaxItems,
			MaxAgeDays: cfg.Conversations.MaxAgeDays,
		}),
		logs: logstore.NewStore(config.DefaultLogsPath(), logstore.Settings{
			MaxLines: cfg.Logs.MaxLines,
		}),
		modelRefreshLast:    map[string]time.Time{},
		modelRefreshRunning: map[string]bool{},
	}
	logutil.SetOutputTee(s.logs.Writer())
	s.loadModelsCacheFromDisk()
	pricingMgr, err := pricing.NewManager(config.DefaultPricingCachePath())
	if err != nil {
		return nil, fmt.Errorf("init pricing manager: %w", err)
	}
	pricingMgr.SetProviders(resolver.ListProviders())
	s.pricing = pricingMgr
	s.providerHealthChecker = NewProviderHealthChecker(resolver, providerHealthCheckInterval)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.proxyRequestLifecycleMiddleware)
	r.Use(requestDebugLogMiddleware)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.With(s.authAPIMiddleware).Get("/v1", s.handleV1Root)

	r.Route("/v1", func(v1 chi.Router) {
		v1.Use(s.authAPIMiddleware)
		v1.Get("/", s.handleV1Root)
		v1.Get("/models", s.handleModels)
		v1.Post("/chat/completions", s.proxyHandler)
		v1.Post("/completions", s.proxyHandler)
		v1.Post("/embeddings", s.proxyHandler)
		v1.Post("/responses", s.proxyHandler)
	})
	r.With(s.authAnyTokenMiddleware).Get("/v1/status", s.handleStatus)

	instanceID := fmt.Sprintf("%d-%d", time.Now().UTC().UnixNano(), os.Getpid())
	adminHandler := NewAdminHandler(store, stats, resolver, pricingMgr, s.providerHealthChecker, instanceID)
	adminHandler.conversations = s.conversations
	adminHandler.logs = s.logs
	if s.logs != nil {
		s.logs.SetOnAppend(adminHandler.NotifyLogChanged)
	}
	s.adminHandler = adminHandler
	adminHandler.SetAccessTokenCleanup(s.runAccessTokenCleanupOnce)
	adminHandler.SetNetworkListenerControl(s.addHTTPListener, s.removeHTTPListener, s.listHTTPListeners)
	adminHandler.RegisterRoutes(r)

	s.httpServer = &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	defer func() {
		if s.conversations != nil {
			s.conversations.Flush()
		}
		if s.logs != nil {
			s.logs.Flush()
		}
		if s.stats != nil {
			s.stats.Flush()
		}
	}()
	cfg := s.store.Snapshot()
	errCh := make(chan error, 2)
	go s.providerHealthChecker.Run(ctx)
	go s.runMaintenanceScheduler(ctx)
	s.runAccessTokenCleanupOnce()

	if cfg.TLS.Enabled {
		httpsSrv := *s.httpServer
		tlsListenAddr := strings.TrimSpace(cfg.TLS.ListenAddr)
		if tlsListenAddr == "" {
			tlsListenAddr = ":443"
		}
		httpsSrv.Addr = tlsListenAddr
		httpMode := strings.ToLower(strings.TrimSpace(cfg.HTTPMode))
		if httpMode == "" {
			httpMode = "enabled"
		}
		var certFile string
		var keyFile string
		var httpChallenge *http.Server
		switch strings.TrimSpace(cfg.TLS.Mode) {
		case "pem":
			httpsSrv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
			cert, err := tls.X509KeyPair([]byte(cfg.TLS.CertPEM), []byte(cfg.TLS.KeyPEM))
			if err != nil {
				return fmt.Errorf("load pem certificate: %w", err)
			}
			httpsSrv.TLSConfig.Certificates = []tls.Certificate{cert}
		case "self_signed":
			httpsSrv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
			cert, err := generateSelfSignedCert(strings.TrimSpace(cfg.TLS.Domain))
			if err != nil {
				return fmt.Errorf("generate self-signed certificate: %w", err)
			}
			httpsSrv.TLSConfig.Certificates = []tls.Certificate{cert}
		default:
			mgr := &autocert.Manager{
				Cache:      autocert.DirCache(cfg.TLS.CacheDir),
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(cfg.TLS.Domain),
				Email:      cfg.TLS.Email,
			}
			httpsSrv.TLSConfig = &tls.Config{GetCertificate: mgr.GetCertificate, MinVersion: tls.VersionTLS12}
			if httpMode != "disabled" {
				httpChallenge = &http.Server{
					Addr:              ":80",
					Handler:           mgr.HTTPHandler(http.HandlerFunc(redirectHTTPS)),
					ReadHeaderTimeout: 10 * time.Second,
				}
			}
		}
		if httpMode == "enabled" {
			if err := s.addHTTPListener(cfg.ListenAddr); err != nil {
				return fmt.Errorf("http listener (tls mode): %w", err)
			}
		}

		if httpChallenge != nil {
			go func() {
				slog.Info("http challenge/redirect listening on :80")
				if err := httpChallenge.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					errCh <- fmt.Errorf("http challenge server: %w", err)
				}
			}()
		}

		go func() {
			slog.Info("proxy listening", "url", listenerURL("https", tlsListenAddr), "tls_mode", strings.TrimSpace(cfg.TLS.Mode))
			if err := httpsSrv.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("https server: %w", err)
			}
		}()

		<-ctx.Done()
		s.draining.Store(true)
		s.waitForProxyIdle(ctx)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
		if httpChallenge != nil {
			_ = httpChallenge.Shutdown(shutdownCtx)
		}
		_ = httpsSrv.Shutdown(shutdownCtx)
		return firstErr(errCh)
	}

	go func() {
		if err := s.addHTTPListener(cfg.ListenAddr); err != nil {
			errCh <- fmt.Errorf("proxy listener: %w", err)
		}
	}()

	<-ctx.Done()
	s.draining.Store(true)
	s.waitForProxyIdle(ctx)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.httpServer.Shutdown(shutdownCtx)
	return firstErr(errCh)
}

func (s *Server) addHTTPListener(addr string) error {
	listenAddr := strings.TrimSpace(addr)
	if listenAddr == "" {
		return fmt.Errorf("listen address required")
	}
	s.listenerMu.Lock()
	if _, ok := s.httpListeners[listenAddr]; ok {
		s.listenerMu.Unlock()
		return nil
	}
	s.listenerMu.Unlock()

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	s.listenerMu.Lock()
	if _, ok := s.httpListeners[listenAddr]; ok {
		s.listenerMu.Unlock()
		_ = ln.Close()
		return nil
	}
	s.httpListeners[listenAddr] = ln
	s.listenerMu.Unlock()

	slog.Info("proxy listening", "url", listenerURL("http", listenAddr))
	go func(addr string, l net.Listener) {
		err := s.httpServer.Serve(l)
		if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			slog.Warn("proxy listener stopped", "addr", addr, "error", err)
		}
		s.listenerMu.Lock()
		if cur, ok := s.httpListeners[addr]; ok && cur == l {
			delete(s.httpListeners, addr)
		}
		s.listenerMu.Unlock()
	}(listenAddr, ln)
	return nil
}

func listenerURL(scheme, addr string) string {
	listenAddr := strings.TrimSpace(addr)
	if listenAddr == "" {
		return scheme + "://"
	}
	if strings.Contains(listenAddr, "://") {
		return listenAddr
	}
	host, port, err := net.SplitHostPort(listenAddr)
	if err == nil {
		if strings.TrimSpace(host) == "" {
			host = "127.0.0.1"
		}
		return scheme + "://" + net.JoinHostPort(host, port)
	}
	return scheme + "://" + listenAddr
}

func (s *Server) removeHTTPListener(addr string) error {
	listenAddr := strings.TrimSpace(addr)
	if listenAddr == "" {
		return fmt.Errorf("listen address required")
	}
	s.listenerMu.Lock()
	ln, ok := s.httpListeners[listenAddr]
	if !ok {
		s.listenerMu.Unlock()
		return nil
	}
	if len(s.httpListeners) <= 1 {
		cfg := s.store.Snapshot()
		if !cfg.TLS.Enabled {
			s.listenerMu.Unlock()
			return fmt.Errorf("cannot remove last active listener")
		}
	}
	delete(s.httpListeners, listenAddr)
	s.listenerMu.Unlock()
	return ln.Close()
}

func (s *Server) listHTTPListeners() []string {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	out := make([]string, 0, len(s.httpListeners))
	for addr := range s.httpListeners {
		out = append(out, addr)
	}
	sort.Strings(out)
	return out
}

func (s *Server) runMaintenanceScheduler(ctx context.Context) {
	t := time.NewTicker(accessTokenCleanupInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.runAccessTokenCleanupOnce()
		}
	}
}

type accessTokenCleanupResult struct {
	ExpiredTokens    int
	EmptyQuotaTokens int
	OrphanedTokens   int
}

func (s *Server) runAccessTokenCleanupOnce() {
	if s == nil || s.store == nil {
		return
	}
	res, err := s.cleanupAccessTokens(nowUTC())
	if err != nil {
		slog.Warn("access token cleanup failed", "error", err)
		return
	}
	totalRemoved := res.ExpiredTokens + res.EmptyQuotaTokens + res.OrphanedTokens
	if totalRemoved > 0 {
		slog.Info("access token cleanup completed",
			"removed_total", totalRemoved,
			"removed_expired", res.ExpiredTokens,
			"removed_empty_non_reset_quota", res.EmptyQuotaTokens,
			"removed_orphaned", res.OrphanedTokens,
		)
	}
}

func (s *Server) cleanupAccessTokens(now time.Time) (accessTokenCleanupResult, error) {
	if s == nil || s.store == nil {
		return accessTokenCleanupResult{}, nil
	}
	cfg := s.store.Snapshot()
	enabledExpired := cfg.AutoRemoveExpiredTokens
	enabledEmptyQuota := cfg.AutoRemoveEmptyQuotaTokens
	if !enabledExpired && !enabledEmptyQuota {
		return accessTokenCleanupResult{}, nil
	}
	var result accessTokenCleanupResult
	err := s.store.Update(func(c *config.ServerConfig) error {
		removedIDs := map[string]struct{}{}
		next := make([]config.IncomingAPIToken, 0, len(c.IncomingTokens))
		for _, tok := range c.IncomingTokens {
			if enabledExpired && tokenIsExpired(tok, now) {
				result.ExpiredTokens++
				removedIDs[strings.TrimSpace(tok.ID)] = struct{}{}
				continue
			}
			if enabledEmptyQuota && tokenHasEmptyNonResetQuota(tok) {
				result.EmptyQuotaTokens++
				removedIDs[strings.TrimSpace(tok.ID)] = struct{}{}
				continue
			}
			next = append(next, tok)
		}
		if len(removedIDs) > 0 {
			filtered := next[:0]
			for _, tok := range next {
				parentID := strings.TrimSpace(tok.ParentID)
				if parentID != "" {
					if _, orphaned := removedIDs[parentID]; orphaned {
						result.OrphanedTokens++
						continue
					}
				}
				filtered = append(filtered, tok)
			}
			next = filtered
		}
		c.IncomingTokens = next
		return nil
	})
	return result, err
}

func tokenIsExpired(tok config.IncomingAPIToken, now time.Time) bool {
	expiresAt := strings.TrimSpace(tok.ExpiresAt)
	if expiresAt == "" {
		return false
	}
	ts, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return false
	}
	return !now.Before(ts)
}

func tokenHasEmptyNonResetQuota(tok config.IncomingAPIToken) bool {
	q := tok.Quota
	if q == nil {
		return false
	}
	return quotaBudgetExhaustedWithoutReset(q.Requests) || quotaBudgetExhaustedWithoutReset(q.Tokens)
}

func quotaBudgetExhaustedWithoutReset(b *config.TokenQuotaBudget) bool {
	if b == nil || b.Limit <= 0 {
		return false
	}
	if b.IntervalSeconds > 0 {
		return false
	}
	return b.Used >= b.Limit
}

func redirectHTTPS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func generateSelfSignedCert(domain string) (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	notBefore := time.Now().Add(-5 * time.Minute)
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, err
	}
	host := strings.TrimSpace(domain)
	if host == "" {
		host = "localhost"
	}
	tmpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	if host != "" && host != "localhost" {
		if ip := net.ParseIP(host); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, host)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
		Leaf:        tmpl,
	}, nil
}

func requestDebugLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		slog.Debug("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}

func (s *Server) proxyRequestLifecycleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isProxyReq := strings.HasPrefix(r.URL.Path, "/v1/")
		if isProxyReq && s.draining.Load() {
			w.Header().Set("Retry-After", "3")
			http.Error(w, "server shutting down", http.StatusServiceUnavailable)
			return
		}
		if isProxyReq {
			s.activeProxyRequests.Add(1)
			defer s.activeProxyRequests.Add(-1)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) waitForProxyIdle(ctx context.Context) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	lastLog := time.Time{}
	for {
		active := s.activeProxyRequests.Load()
		if active <= 0 {
			slog.Info("shutdown: proxy idle")
			return
		}
		if lastLog.IsZero() || time.Since(lastLog) >= time.Second {
			slog.Info("shutdown: waiting for active proxy requests", "active", active)
			lastLog = time.Now()
		}
		select {
		case <-ctx.Done():
		case <-t.C:
		}
	}
}

func (s *Server) authAPIMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := s.store.Snapshot()
		if cfg.AllowLocalhostNoAuth && requestIsTrustedNoAuth(r, cfg) {
			next.ServeHTTP(w, r)
			return
		}
		identity, ok := resolveAuthIdentity(bearerToken(r.Header), cfg)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !config.RoleAtLeast(identity.Role, config.TokenRoleInferrer) {
			http.Error(w, "forbidden: token role cannot use inference api", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(withAPIAuthIdentity(r.Context(), identity)))
	})
}

func (s *Server) authAnyTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := s.store.Snapshot()
		if cfg.AllowLocalhostNoAuth && requestIsTrustedNoAuth(r, cfg) {
			identity := tokenAuthIdentity{
				Role: config.TokenRoleInferrer,
			}
			next.ServeHTTP(w, r.WithContext(withAPIAuthIdentity(r.Context(), identity)))
			return
		}
		identity, ok := resolveAuthIdentity(bearerToken(r.Header), cfg)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(withAPIAuthIdentity(r.Context(), identity)))
	})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.resolver.DiscoverModels(r.Context())
	if err != nil {
		if cached := s.modelsCached.Load(); cached != nil {
			models = *cached
		} else {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	} else {
		s.modelsCached.Store(&models)
		s.saveModelsCacheToDisk(models)
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": models})
}

func (s *Server) handleV1Root(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"object":  "tokenrouter",
		"service": "TokenRouter OpenAI-compatible API",
		"paths": []string{
			"/v1/models",
			"/v1/chat/completions",
			"/v1/completions",
			"/v1/embeddings",
			"/v1/responses",
			"/v1/status",
		},
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	period := time.Hour
	if raw := strings.TrimSpace(r.URL.Query().Get("period_seconds")); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			period = time.Duration(sec) * time.Second
		}
	}

	now := time.Now().UTC()
	summary := s.stats.Summary(period)
	providers := []config.ProviderConfig{}
	quotaProviders := []config.ProviderConfig{}
	quotas := map[string]ProviderQuotaSnapshot{}
	if s.adminHandler != nil {
		providers = s.adminHandler.catalogProviders()
		quotaProviders = s.adminHandler.quotaProviders()
		quotas = s.adminHandler.readProviderQuotas(r.Context(), quotaProviders, false)
	}
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		names = append(names, p.Name)
	}
	providersAvailable := len(names)
	providersOnline := 0
	if s.providerHealthChecker != nil {
		providersAvailable, providersOnline = s.providerHealthChecker.AvailabilitySummary(names)
	}

	v := version.Current()
	writeJSON(w, http.StatusOK, map[string]any{
		"checked_at":          now.Format(time.RFC3339),
		"period_seconds":      int64(period / time.Second),
		"version":             version.String(),
		"raw":                 v.Version,
		"commit":              v.Commit,
		"date":                v.Date,
		"dirty":               v.Dirty,
		"providers_available": providersAvailable,
		"providers_online":    providersOnline,
		"provider_quotas":     quotas,
		"requests":            summary.Requests,
		"prompt_tokens":       summary.PromptTokens,
		"completion_tokens":   summary.CompletionTokens,
		"total_tokens":        summary.TotalTokens,
		"avg_latency_ms":      summary.AvgLatencyMS,
		"avg_prompt_tps":      summary.AvgPromptTPS,
		"avg_generation_tps":  summary.AvgGenerationTPS,
	})
}

func (s *Server) loadModelsCacheFromDisk() {
	if s == nil || strings.TrimSpace(s.modelsCachePath) == "" {
		return
	}
	var models []ModelCard
	if err := cache.LoadJSON(s.modelsCachePath, &models); err != nil {
		return
	}
	if len(models) == 0 {
		return
	}
	s.modelsCached.Store(&models)
}

func (s *Server) saveModelsCacheToDisk(models []ModelCard) {
	if s == nil || strings.TrimSpace(s.modelsCachePath) == "" {
		return
	}
	cp := append([]ModelCard(nil), models...)
	if len(cp) == 0 {
		return
	}
	_ = cache.SaveJSON(s.modelsCachePath, cp)
}

func (s *Server) proxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := readRequestBody(r, 8<<20)
	if err != nil {
		http.Error(w, "failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	model, mutatedBody, provider, upstreamModel, stream, err := s.prepareUpstreamRequest(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	identity, hasIdentity := apiAuthIdentityFromContext(r.Context())
	quotaView := keyQuotaView{}
	hasQuota := false
	if hasIdentity {
		qv, metered, qerr := s.reserveRequestQuota(identity)
		if qerr != nil {
			if errors.Is(qerr, errQuotaExceeded) {
				writeQuotaExceededResponse(w, qv)
				return
			}
			if errors.Is(qerr, errOwnerTokenMissing) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, wrapQuotaInternalErr(qerr).Error(), http.StatusInternalServerError)
			return
		}
		hasQuota = metered && qv.HasAny()
		quotaView = qv
	}
	clientMeta := extractClientUsageMeta(r, s.store.Snapshot())
	captureEnabled := isConversationCaptureEndpoint(r.URL.Path)
	reqConversationID, reqPrevResponseID := parseConversationRequestIDs(r.Header, body)
	requestPayloadRaw := capConversationRawText(body, 128<<10)
	requestHeaders := sanitizeConversationRequestHeaders(r.Header)
	requestHeadersRaw := conversationHeadersMapToRaw(requestHeaders)

	start := time.Now()
	if stream {
		if hasQuota {
			applyQuotaHeaders(w.Header(), quotaView)
		}
		statusCode, upstreamHeader, usage, initialLatency, err := s.forwardStreamingRequest(r.Context(), provider, r.URL.Path, mutatedBody, r.Header, w)
		latency := time.Since(start)
		if err != nil {
			s.providerHealthChecker.RecordProxyResult(provider.Name, latency, statusCode, err)
			if statusCode == 0 {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
			return
		}
		s.providerHealthChecker.RecordProxyResult(provider.Name, latency, statusCode, nil)
		if statusCode >= 200 && statusCode <= 299 && s.adminHandler != nil {
			s.adminHandler.RecordQuotaFromResponse(provider, upstreamHeader)
		}
		if usage.TotalTokens == 0 {
			usage.PromptTokens = estimatePromptTokensFromRequest(body)
			if usage.CompletionTokens == 0 {
				usage.CompletionTokens = usage.EstimatedCompletionTokens
			}
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		}
		if hasIdentity && hasQuota && statusCode >= 200 && statusCode <= 299 {
			if qv, metered, qerr := s.applyTokenUsageQuota(identity, int64(usage.TotalTokens)); qerr == nil && metered && qv.HasAny() {
				quotaView = qv
			}
		}
		promptTPS := usage.PromptTPS
		genTPS := usage.GenTPS
		if !usage.HasPromptTPS || !usage.HasGenTPS {
			fallbackPromptTPS, fallbackGenTPS := computePromptAndGenerationTPS(usage.PromptTokens, usage.CompletionTokens, initialLatency, latency)
			if !usage.HasPromptTPS {
				promptTPS = fallbackPromptTPS
			}
			if !usage.HasGenTPS {
				genTPS = fallbackGenTPS
			}
		}
		usageLatency := latency
		if usage.HasProviderTotalSeconds && usage.ProviderTotalSeconds > 0 {
			usageLatency = durationFromProviderSeconds(usage.ProviderTotalSeconds)
		}
		s.recordUsageMeasured(provider.Name, model, upstreamModel, statusCode, usageLatency, usage.PromptTokens, usage.PromptCachedTokens, usage.CompletionTokens, usage.TotalTokens, promptTPS, genTPS, clientMeta)
		if captureEnabled && s.adminHandler != nil {
			responseHeaders := sanitizeConversationResponseHeaders(upstreamHeader)
			s.adminHandler.RecordConversation(conversations.CaptureInput{
				Timestamp:          time.Now().UTC(),
				Endpoint:           normalizeConversationEndpoint(r.URL.Path),
				Provider:           provider.Name,
				Model:              model,
				RemoteIP:           clientMeta.ClientIP,
				APIKeyName:         clientMeta.APIKeyName,
				RequestHeadersRaw:  requestHeadersRaw,
				ResponseHeadersRaw: conversationHeadersMapToRaw(responseHeaders),
				RequestPayloadRaw:  requestPayloadRaw,
				ResponsePayloadRaw: usage.CapturedStreamRaw,
				StatusCode:         statusCode,
				LatencyMS:          latency.Milliseconds(),
				Stream:             true,
				ProtocolIDs: conversations.ProtocolIDs{
					RequestConversationID:   reqConversationID,
					RequestPreviousResponse: reqPrevResponseID,
					ResponseID:              strings.TrimSpace(usage.ResponseID),
				},
			})
		}
		return
	}

	respBody, statusCode, header, initialLatency, err := s.forwardRequest(r.Context(), provider, r.URL.Path, mutatedBody, r.Header)
	latency := time.Since(start)
	if err != nil {
		s.providerHealthChecker.RecordProxyResult(provider.Name, latency, 0, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if isProviderModelNotFoundError(statusCode, respBody) {
		s.triggerProviderModelRefresh(provider, "model_not_found")
	}
	s.providerHealthChecker.RecordProxyResult(provider.Name, latency, statusCode, nil)
	if statusCode >= 200 && statusCode <= 299 && s.adminHandler != nil {
		s.adminHandler.RecordQuotaFromResponse(provider, header)
	}

	promptTokens, completionTokens, totalTokens := parseUsageTokens(respBody)
	metrics := parseProviderUsageMetrics(respBody)
	if totalTokens == 0 {
		promptTokens = estimatePromptTokensFromRequest(body)
		completionTokens = estimateCompletionTokensFromResponse(respBody)
		totalTokens = promptTokens + completionTokens
	}
	if hasIdentity && hasQuota && statusCode >= 200 && statusCode <= 299 {
		if qv, metered, qerr := s.applyTokenUsageQuota(identity, int64(totalTokens)); qerr == nil && metered && qv.HasAny() {
			quotaView = qv
		}
	}
	if hasQuota {
		respBody = injectQuotaIntoJSONBody(respBody, quotaView)
	}
	for k, vals := range header {
		if strings.EqualFold(k, "content-length") {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	if hasQuota {
		applyQuotaHeaders(w.Header(), quotaView)
	}
	w.WriteHeader(statusCode)
	_, _ = w.Write(respBody)

	promptTPS := metrics.PromptTPS
	genTPS := metrics.GenTPS
	if !metrics.HasPromptTPS || !metrics.HasGenTPS {
		fallbackPromptTPS, fallbackGenTPS := computePromptAndGenerationTPS(promptTokens, completionTokens, initialLatency, latency)
		if !metrics.HasPromptTPS {
			promptTPS = fallbackPromptTPS
		}
		if !metrics.HasGenTPS {
			genTPS = fallbackGenTPS
		}
	}
	usageLatency := latency
	if metrics.HasTotalSeconds && metrics.TotalSeconds > 0 {
		usageLatency = durationFromProviderSeconds(metrics.TotalSeconds)
	}
	s.recordUsageMeasured(provider.Name, model, upstreamModel, statusCode, usageLatency, promptTokens, metrics.PromptCachedTokens, completionTokens, totalTokens, promptTPS, genTPS, clientMeta)
	if captureEnabled && s.adminHandler != nil {
		responseHeaders := sanitizeConversationResponseHeaders(header)
		s.adminHandler.RecordConversation(conversations.CaptureInput{
			Timestamp:          time.Now().UTC(),
			Endpoint:           normalizeConversationEndpoint(r.URL.Path),
			Provider:           provider.Name,
			Model:              model,
			RemoteIP:           clientMeta.ClientIP,
			APIKeyName:         clientMeta.APIKeyName,
			RequestHeadersRaw:  requestHeadersRaw,
			ResponseHeadersRaw: conversationHeadersMapToRaw(responseHeaders),
			RequestPayloadRaw:  requestPayloadRaw,
			ResponsePayloadRaw: capConversationRawText(respBody, 128<<10),
			StatusCode:         statusCode,
			LatencyMS:          latency.Milliseconds(),
			Stream:             false,
			ProtocolIDs: conversations.ProtocolIDs{
				RequestConversationID:   reqConversationID,
				RequestPreviousResponse: reqPrevResponseID,
				ResponseID:              parseResponseID(respBody),
			},
		})
	}
}

func isProviderModelNotFoundError(status int, body []byte) bool {
	if status != http.StatusNotFound || len(body) == 0 {
		return false
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	return payloadContainsModelNotFound(payload)
}

func payloadContainsModelNotFound(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			key := strings.ToLower(strings.TrimSpace(k))
			if key == "code" || key == "type" || key == "error" || key == "message" {
				val := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", vv)))
				if strings.Contains(val, "model_not_found") || strings.Contains(val, "not_found_error") || strings.Contains(val, "model does not exist") {
					return true
				}
			}
			if payloadContainsModelNotFound(vv) {
				return true
			}
		}
	case []any:
		for _, vv := range x {
			if payloadContainsModelNotFound(vv) {
				return true
			}
		}
	}
	return false
}

func (s *Server) triggerProviderModelRefresh(provider config.ProviderConfig, reason string) {
	name := strings.TrimSpace(provider.Name)
	if s == nil || name == "" {
		return
	}
	now := time.Now().UTC()
	s.modelRefreshMu.Lock()
	if s.modelRefreshRunning[name] {
		s.modelRefreshMu.Unlock()
		return
	}
	if last := s.modelRefreshLast[name]; !last.IsZero() && now.Sub(last) < modelNotFoundRefreshDebounce {
		s.modelRefreshMu.Unlock()
		return
	}
	s.modelRefreshRunning[name] = true
	s.modelRefreshLast[name] = now
	s.modelRefreshMu.Unlock()

	slog.Info("triggering provider model refresh", "provider", name, "reason", strings.TrimSpace(reason))
	go func(p config.ProviderConfig) {
		defer func() {
			s.modelRefreshMu.Lock()
			s.modelRefreshRunning[name] = false
			s.modelRefreshMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cards, err := NewProviderClient(p).ListModels(ctx)
		if err != nil {
			slog.Warn("provider model refresh failed", "provider", name, "err", err)
			return
		}
		s.mergeProviderModelsCache(name, cards)
		if s.adminHandler != nil {
			s.adminHandler.notifyAdminChanged("models")
		}
		slog.Info("provider model refresh completed", "provider", name, "models", len(cards))
	}(provider)
}

func (s *Server) mergeProviderModelsCache(providerName string, cards []ModelCard) {
	if s == nil || strings.TrimSpace(providerName) == "" {
		return
	}
	existing := []ModelCard{}
	if cur := s.modelsCached.Load(); cur != nil {
		existing = append(existing, (*cur)...)
	}
	out := make([]ModelCard, 0, len(existing)+len(cards))
	for _, m := range existing {
		if strings.TrimSpace(m.Provider) == providerName {
			continue
		}
		out = append(out, m)
	}
	out = append(out, cards...)
	s.modelsCached.Store(&out)
	s.saveModelsCacheToDisk(out)
}

func dedupeConversationText(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
}

func (s *Server) prepareUpstreamRequest(body []byte) (incomingModel string, outBody []byte, provider config.ProviderConfig, upstreamModel string, stream bool, err error) {
	if len(body) == 0 {
		return "", nil, config.ProviderConfig{}, "", false, fmt.Errorf("request body required")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, config.ProviderConfig{}, "", false, fmt.Errorf("invalid json")
	}
	modelAny, ok := payload["model"]
	if !ok {
		return "", nil, config.ProviderConfig{}, "", false, fmt.Errorf("model is required")
	}
	model, ok := modelAny.(string)
	if !ok || strings.TrimSpace(model) == "" {
		return "", nil, config.ProviderConfig{}, "", false, fmt.Errorf("model must be a non-empty string")
	}
	stream, _ = payload["stream"].(bool)
	provider, upstreamModel, err = s.resolver.Resolve(model)
	if err != nil {
		return "", nil, config.ProviderConfig{}, "", false, err
	}
	payload["model"] = upstreamModel
	outBody, err = json.Marshal(payload)
	if err != nil {
		return "", nil, config.ProviderConfig{}, "", false, fmt.Errorf("encode json: %w", err)
	}
	return model, outBody, provider, upstreamModel, stream, nil
}

func (s *Server) forwardRequest(ctx context.Context, provider config.ProviderConfig, requestPath string, body []byte, clientHeaders http.Header) ([]byte, int, http.Header, time.Duration, error) {
	provider = s.ensureOAuthTokenFresh(ctx, provider)
	u, err := url.Parse(strings.TrimRight(provider.BaseURL, "/"))
	if err != nil {
		return nil, 0, nil, 0, fmt.Errorf("invalid provider base_url: %w", err)
	}
	if isOpenAICodexProvider(provider) && requestPath == "/v1/responses" {
		requestPath = "/codex/responses"
	}
	u.Path = joinProviderPath(u.Path, requestPath)
	cli := &http.Client{Timeout: time.Duration(provider.TimeoutSeconds) * time.Second}
	var initialLatency time.Duration
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
		if err != nil {
			return nil, 0, nil, 0, err
		}
		applyUpstreamProviderHeaders(req, provider, clientHeaders)

		upstreamStart := time.Now()
		resp, err := cli.Do(req)
		initialLatency = time.Since(upstreamStart)
		if err != nil {
			if attempt < defaultUpstreamRetryCount && shouldRetryUpstreamError(err) && sleepWithContext(ctx, upstreamRetryDelay(attempt)) == nil {
				continue
			}
			return nil, 0, nil, 0, err
		}
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			if attempt < defaultUpstreamRetryCount && shouldRetryUpstreamError(readErr) && sleepWithContext(ctx, upstreamRetryDelay(attempt)) == nil {
				continue
			}
			return nil, 0, nil, initialLatency, readErr
		}
		if attempt < defaultUpstreamRetryCount && shouldRetryUpstreamStatus(resp.StatusCode) && sleepWithContext(ctx, upstreamRetryDelay(attempt)) == nil {
			continue
		}
		return b, resp.StatusCode, resp.Header.Clone(), initialLatency, nil
	}
}

type usageTokenCounts struct {
	PromptTokens              int
	PromptCachedTokens        int
	CompletionTokens          int
	TotalTokens               int
	EstimatedCompletionTokens int
	PromptTPS                 float64
	GenTPS                    float64
	HasPromptTPS              bool
	HasGenTPS                 bool
	ProviderQueueSeconds      float64
	ProviderTotalSeconds      float64
	HasProviderTotalSeconds   bool
	CapturedOutput            string
	CapturedStreamRaw         string
	ResponseID                string
}

type clientUsageMeta struct {
	ClientType string
	UserAgent  string
	ClientIP   string
	APIKeyName string
}

func (s *Server) forwardStreamingRequest(ctx context.Context, provider config.ProviderConfig, requestPath string, body []byte, clientHeaders http.Header, w http.ResponseWriter) (int, http.Header, usageTokenCounts, time.Duration, error) {
	provider = s.ensureOAuthTokenFresh(ctx, provider)
	u, err := url.Parse(strings.TrimRight(provider.BaseURL, "/"))
	if err != nil {
		return 0, nil, usageTokenCounts{}, 0, fmt.Errorf("invalid provider base_url: %w", err)
	}
	if isOpenAICodexProvider(provider) && requestPath == "/v1/responses" {
		requestPath = "/codex/responses"
	}
	u.Path = joinProviderPath(u.Path, requestPath)
	cli := &http.Client{Timeout: time.Duration(provider.TimeoutSeconds) * time.Second}
	var (
		resp           *http.Response
		initialLatency time.Duration
		upstreamStart  time.Time
	)
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
		if err != nil {
			return 0, nil, usageTokenCounts{}, 0, err
		}
		applyUpstreamProviderHeaders(req, provider, clientHeaders)

		upstreamStart = time.Now()
		resp, err = cli.Do(req)
		initialLatency = time.Since(upstreamStart)
		if err != nil {
			if attempt < defaultUpstreamRetryCount && shouldRetryUpstreamError(err) && sleepWithContext(ctx, upstreamRetryDelay(attempt)) == nil {
				continue
			}
			return 0, nil, usageTokenCounts{}, 0, err
		}
		if attempt < defaultUpstreamRetryCount && shouldRetryUpstreamStatus(resp.StatusCode) {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
			_ = resp.Body.Close()
			if sleepWithContext(ctx, upstreamRetryDelay(attempt)) == nil {
				continue
			}
		}
		break
	}
	defer resp.Body.Close()
	firstChunkLatency := time.Duration(0)
	upstreamHeader := resp.Header.Clone()

	for k, vals := range resp.Header {
		if strings.EqualFold(k, "content-length") {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	parser := newSSEUsageParser()
	const maxCapturedStreamRawBytes = 128 << 10
	rawCapture := make([]byte, 0, 8<<10)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if firstChunkLatency <= 0 {
				firstChunkLatency = time.Since(upstreamStart)
			}
			parser.Consume(buf[:n])
			if len(rawCapture) < maxCapturedStreamRawBytes {
				remaining := maxCapturedStreamRawBytes - len(rawCapture)
				if remaining > n {
					remaining = n
				}
				rawCapture = append(rawCapture, buf[:remaining]...)
			}
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				if firstChunkLatency > 0 {
					initialLatency = firstChunkLatency
				}
				usage := parser.Usage()
				usage.CapturedStreamRaw = capConversationRawText(rawCapture, maxCapturedStreamRawBytes)
				return resp.StatusCode, upstreamHeader, usage, initialLatency, writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if errors.Is(readErr, io.EOF) {
			if firstChunkLatency > 0 {
				initialLatency = firstChunkLatency
			}
			usage := parser.Usage()
			usage.CapturedStreamRaw = capConversationRawText(rawCapture, maxCapturedStreamRawBytes)
			return resp.StatusCode, upstreamHeader, usage, initialLatency, nil
		}
		if readErr != nil {
			if firstChunkLatency > 0 {
				initialLatency = firstChunkLatency
			}
			usage := parser.Usage()
			usage.CapturedStreamRaw = capConversationRawText(rawCapture, maxCapturedStreamRawBytes)
			return resp.StatusCode, upstreamHeader, usage, initialLatency, readErr
		}
	}
}

func shouldRetryUpstreamStatus(status int) bool {
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func shouldRetryUpstreamError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "unexpected eof")
}

func upstreamRetryDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	return defaultUpstreamRetryBackoff * time.Duration(1<<attempt)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func readRequestBody(r *http.Request, limit int64) ([]byte, error) {
	body := r.Body
	if body == nil {
		return nil, io.EOF
	}
	defer body.Close()
	raw, err := io.ReadAll(io.LimitReader(body, limit))
	if err != nil {
		return nil, err
	}
	encoding := strings.TrimSpace(strings.ToLower(r.Header.Get("Content-Encoding")))
	if encoding == "" || encoding == "identity" {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			return raw, nil
		}
		if decoded, ok := decodeByMagic(raw, limit); ok {
			return decoded, nil
		}
		return raw, nil
	}
	decoded, err := decodeRequestBytes(raw, encoding, limit)
	if err == nil {
		return decoded, nil
	}
	// Some clients send an incorrect content-encoding header while still
	// sending plain JSON. In that case, prefer raw payload if it looks like JSON.
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return raw, nil
	}
	return nil, err
}

func decodeRequestBytes(raw []byte, encoding string, limit int64) ([]byte, error) {
	parts := strings.Split(encoding, ",")
	decoded := raw
	for i := len(parts) - 1; i >= 0; i-- {
		enc := strings.TrimSpace(strings.ToLower(parts[i]))
		if enc == "" || enc == "identity" {
			continue
		}
		out, err := decodeSingleEncoding(decoded, enc, limit)
		if err != nil {
			return nil, err
		}
		decoded = out
	}
	return decoded, nil
}

func decodeSingleEncoding(raw []byte, encoding string, limit int64) ([]byte, error) {
	in := bytes.NewReader(raw)
	switch encoding {
	case "gzip":
		gr, err := gzip.NewReader(in)
		if err != nil {
			return nil, fmt.Errorf("decode gzip: %w", err)
		}
		defer gr.Close()
		return io.ReadAll(io.LimitReader(gr, limit))
	case "deflate":
		zr, zErr := zlib.NewReader(in)
		if zErr == nil {
			defer zr.Close()
			return io.ReadAll(io.LimitReader(zr, limit))
		}
		fr := flate.NewReader(in)
		defer fr.Close()
		return io.ReadAll(io.LimitReader(fr, limit))
	case "zstd":
		zr, err := zstd.NewReader(in)
		if err != nil {
			return nil, fmt.Errorf("decode zstd: %w", err)
		}
		defer zr.Close()
		return io.ReadAll(io.LimitReader(zr, limit))
	default:
		return nil, fmt.Errorf("unsupported content-encoding %q", encoding)
	}
}

func decodeByMagic(raw []byte, limit int64) ([]byte, bool) {
	// zstd frame magic bytes: 28 B5 2F FD
	if len(raw) >= 4 && raw[0] == 0x28 && raw[1] == 0xB5 && raw[2] == 0x2F && raw[3] == 0xFD {
		if b, err := decodeSingleEncoding(raw, "zstd", limit); err == nil {
			trim := bytes.TrimSpace(b)
			if len(trim) > 0 && (trim[0] == '{' || trim[0] == '[') {
				return b, true
			}
		}
	}
	// gzip magic bytes: 1F 8B
	if len(raw) >= 2 && raw[0] == 0x1F && raw[1] == 0x8B {
		if b, err := decodeSingleEncoding(raw, "gzip", limit); err == nil {
			trim := bytes.TrimSpace(b)
			if len(trim) > 0 && (trim[0] == '{' || trim[0] == '[') {
				return b, true
			}
		}
	}
	// zlib usually starts with 0x78 (common: 78 9C, 78 DA, 78 01).
	if len(raw) >= 1 && raw[0] == 0x78 {
		if b, err := decodeSingleEncoding(raw, "deflate", limit); err == nil {
			trim := bytes.TrimSpace(b)
			if len(trim) > 0 && (trim[0] == '{' || trim[0] == '[') {
				return b, true
			}
		}
	}
	return nil, false
}

func isOpenAICodexProvider(provider config.ProviderConfig) bool {
	base := strings.TrimSpace(provider.BaseURL)
	if base == "" {
		return false
	}
	u, err := url.Parse(base)
	if err != nil {
		baseLower := strings.ToLower(base)
		return strings.Contains(baseLower, "chatgpt.com/backend-api") || strings.Contains(baseLower, "/backend-api")
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	pathLower := strings.ToLower(strings.Trim(strings.TrimSpace(u.Path), "/"))
	return host == "chatgpt.com" || strings.Contains(pathLower, "backend-api")
}

func applyUpstreamProviderHeaders(req *http.Request, provider config.ProviderConfig, clientHeaders http.Header) {
	copyForwardableRequestHeaders(req.Header, clientHeaders)
	req.Header.Set("Content-Type", "application/json")
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	} else if strings.TrimSpace(provider.AuthToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(provider.AuthToken))
	}
	if !isOpenAICodexProvider(provider) {
		return
	}
	if strings.TrimSpace(req.Header.Get("Accept")) == "" {
		req.Header.Set("Accept", "application/json")
	}
	if strings.TrimSpace(req.Header.Get("OpenAI-Beta")) == "" {
		req.Header.Set("OpenAI-Beta", "responses=experimental")
	}
	if strings.TrimSpace(req.Header.Get("originator")) == "" {
		req.Header.Set("originator", openAICodexOriginator())
	}
	if strings.TrimSpace(req.Header.Get("User-Agent")) == "" {
		req.Header.Set("User-Agent", "codex-cli/0.104.0")
	}
	if strings.TrimSpace(provider.AccountID) != "" {
		req.Header.Set("ChatGPT-Account-ID", strings.TrimSpace(provider.AccountID))
	}
}

func copyForwardableRequestHeaders(dst http.Header, src http.Header) {
	if len(src) == 0 {
		return
	}
	for k, vals := range src {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if shouldDropForwardedRequestHeader(key) {
			continue
		}
		for _, v := range vals {
			dst.Add(key, v)
		}
	}
}

func shouldDropForwardedRequestHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "host",
		"connection",
		"proxy-connection",
		"keep-alive",
		"accept-encoding",
		"content-encoding",
		"transfer-encoding",
		"upgrade",
		"te",
		"trailer",
		"proxy-authenticate",
		"proxy-authorization",
		"content-length",
		"authorization",
		"x-api-key":
		return true
	default:
		return false
	}
}

func openAICodexOriginator() string {
	originator := strings.TrimSpace(os.Getenv("CODEX_INTERNAL_ORIGINATOR_OVERRIDE"))
	if originator == "" {
		return "codex_cli_rs"
	}
	return originator
}

func (s *Server) ensureOAuthTokenFresh(ctx context.Context, provider config.ProviderConfig) config.ProviderConfig {
	return refreshOAuthTokenForProvider(ctx, s.store, provider)
}

func (s *Server) recordUsage(providerName, incomingModel, upstreamModel string, status int, latency time.Duration, responseBody []byte) {
	promptTokens, completionTokens, totalTokens := 0, 0, 0
	if status >= 200 && status <= 299 {
		promptTokens, completionTokens, totalTokens = parseUsageTokens(responseBody)
	}
	s.recordUsageWithTokens(providerName, incomingModel, upstreamModel, status, latency, promptTokens, completionTokens, totalTokens)
}

func (s *Server) recordUsageWithTokens(providerName, incomingModel, upstreamModel string, status int, latency time.Duration, promptTokens int, completionTokens int, totalTokens int) {
	promptTPS, genTPS := computePromptAndGenerationTPS(promptTokens, completionTokens, latency, latency)
	s.recordUsageMeasured(providerName, incomingModel, upstreamModel, status, latency, promptTokens, 0, completionTokens, totalTokens, promptTPS, genTPS, clientUsageMeta{})
}

func (s *Server) recordUsageMeasured(providerName, incomingModel, upstreamModel string, status int, latency time.Duration, promptTokens int, promptCachedTokens int, completionTokens int, totalTokens int, promptTPS float64, genTPS float64, meta clientUsageMeta) {
	if status < 200 || status > 299 {
		promptTokens = 0
		promptCachedTokens = 0
		completionTokens = 0
		totalTokens = 0
		promptTPS = 0
		genTPS = 0
	}
	s.stats.Add(UsageEvent{
		Timestamp:      time.Now(),
		Provider:       providerName,
		Model:          incomingModel,
		ClientType:     strings.TrimSpace(meta.ClientType),
		UserAgent:      strings.TrimSpace(meta.UserAgent),
		ClientIP:       strings.TrimSpace(meta.ClientIP),
		APIKeyName:     strings.TrimSpace(meta.APIKeyName),
		StatusCode:     status,
		PromptTokens:   promptTokens,
		PromptCached:   promptCachedTokens,
		CompletionToks: completionTokens,
		TotalTokens:    totalTokens,
		LatencyMS:      latency.Milliseconds(),
		PromptTPS:      promptTPS,
		GenTPS:         genTPS,
	})
	if s.adminHandler != nil {
		s.adminHandler.NotifyStatsChanged()
	}
	_ = upstreamModel
}

func parseUsageTokens(responseBody []byte) (promptTokens int, completionTokens int, totalTokens int) {
	if len(responseBody) == 0 {
		return 0, 0, 0
	}
	var payload any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return 0, 0, 0
	}
	return findUsageTokens(payload)
}

type providerUsageMetrics struct {
	PromptCachedTokens int
	PromptTPS          float64
	GenTPS             float64
	HasPromptTPS       bool
	HasGenTPS          bool
	QueueSeconds       float64
	TotalSeconds       float64
	HasTotalSeconds    bool
}

func parseProviderUsageMetrics(responseBody []byte) providerUsageMetrics {
	if len(responseBody) == 0 {
		return providerUsageMetrics{}
	}
	var payload any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return providerUsageMetrics{}
	}
	return findProviderUsageMetrics(payload)
}

func findProviderUsageMetrics(payload any) providerUsageMetrics {
	out := providerUsageMetrics{}
	bestCached := 0
	bestTotalSeconds := 0.0
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			cached := 0
			if usageMap, ok := x["usage"].(map[string]any); ok {
				if detailsMap, ok2 := usageMap["prompt_tokens_details"].(map[string]any); ok2 {
					cached = int(firstFloat(detailsMap, "cached_tokens"))
				}
				promptTime := firstFloat(usageMap, "prompt_time")
				completionTime := firstFloat(usageMap, "completion_time")
				queueTime := firstFloat(usageMap, "queue_time")
				totalTime := firstFloat(usageMap, "total_time")
				if ti, ok2 := x["time_info"].(map[string]any); ok2 {
					if promptTime <= 0 {
						promptTime = firstFloat(ti, "prompt_time")
					}
					if completionTime <= 0 {
						completionTime = firstFloat(ti, "completion_time")
					}
					if queueTime <= 0 {
						queueTime = firstFloat(ti, "queue_time")
					}
					if totalTime <= 0 {
						totalTime = firstFloat(ti, "total_time")
					}
				}
				promptTokens := firstFloat(usageMap, "prompt_tokens", "input_tokens")
				completionTokens := firstFloat(usageMap, "completion_tokens", "output_tokens")
				if promptTime > 0 && promptTokens > 0 {
					out.PromptTPS = promptTokens / promptTime
					out.HasPromptTPS = true
				}
				if completionTime > 0 && completionTokens > 0 {
					out.GenTPS = completionTokens / completionTime
					out.HasGenTPS = true
				}
				if totalTime <= 0 {
					totalTime = queueTime + promptTime + completionTime
				}
				if totalTime > bestTotalSeconds {
					bestTotalSeconds = totalTime
					out.QueueSeconds = queueTime
					out.TotalSeconds = totalTime
					out.HasTotalSeconds = totalTime > 0
				}
			}
			if cached > bestCached {
				bestCached = cached
				out.PromptCachedTokens = cached
			}
			for _, vv := range x {
				walk(vv)
			}
		case []any:
			for _, vv := range x {
				walk(vv)
			}
		}
	}
	walk(payload)
	return out
}

type sseUsageParser struct {
	pending []byte
	usage   usageTokenCounts
}

func newSSEUsageParser() *sseUsageParser {
	return &sseUsageParser{pending: make([]byte, 0, 1024)}
}

func (p *sseUsageParser) Consume(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	p.pending = append(p.pending, chunk...)
	for {
		idx := bytes.IndexByte(p.pending, '\n')
		if idx < 0 {
			return
		}
		line := strings.TrimSpace(string(p.pending[:idx]))
		p.pending = p.pending[idx+1:]
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		p.captureStreamFields([]byte(data))
		prompt, completion, total := parseUsageTokens([]byte(data))
		metrics := parseProviderUsageMetrics([]byte(data))
		p.merge(prompt, metrics.PromptCachedTokens, completion, total, metrics)
		if p.usage.TotalTokens == 0 {
			p.usage.EstimatedCompletionTokens += estimateCompletionTokensFromResponse([]byte(data))
		}
	}
}

func (p *sseUsageParser) captureStreamFields(data []byte) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}
	if p.usage.ResponseID == "" {
		if rid := strings.TrimSpace(asString(payload["id"])); rid != "" && rid != "<nil>" {
			p.usage.ResponseID = rid
		}
	}
	if p.usage.ResponseID == "" {
		if resp, ok := payload["response"].(map[string]any); ok {
			if rid := strings.TrimSpace(asString(resp["id"])); rid != "" && rid != "<nil>" {
				p.usage.ResponseID = rid
			}
		}
	}
	chunkText := extractCompletionText(payload)
	if strings.TrimSpace(chunkText) == "" {
		return
	}
	if p.usage.CapturedOutput == "" {
		p.usage.CapturedOutput = chunkText
		return
	}
	p.usage.CapturedOutput += chunkText
}

func (p *sseUsageParser) Usage() usageTokenCounts {
	return p.usage
}

func (p *sseUsageParser) merge(prompt int, promptCached int, completion int, total int, metrics providerUsageMetrics) {
	if total > p.usage.TotalTokens {
		p.usage.PromptTokens = prompt
		p.usage.PromptCachedTokens = promptCached
		p.usage.CompletionTokens = completion
		p.usage.TotalTokens = total
	} else if p.usage.TotalTokens == 0 && (prompt > 0 || completion > 0) {
		p.usage.PromptTokens = prompt
		p.usage.PromptCachedTokens = promptCached
		p.usage.CompletionTokens = completion
		p.usage.TotalTokens = prompt + completion
	}
	if promptCached > p.usage.PromptCachedTokens {
		p.usage.PromptCachedTokens = promptCached
	}
	if metrics.HasPromptTPS {
		p.usage.PromptTPS = metrics.PromptTPS
		p.usage.HasPromptTPS = true
	}
	if metrics.HasGenTPS {
		p.usage.GenTPS = metrics.GenTPS
		p.usage.HasGenTPS = true
	}
	if metrics.HasTotalSeconds && metrics.TotalSeconds > p.usage.ProviderTotalSeconds {
		p.usage.ProviderQueueSeconds = metrics.QueueSeconds
		p.usage.ProviderTotalSeconds = metrics.TotalSeconds
		p.usage.HasProviderTotalSeconds = true
	}
}

func findUsageTokens(payload any) (promptTokens int, completionTokens int, totalTokens int) {
	bestTotal := 0
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			p, c, t, ok := parseUsageObject(x)
			if ok {
				score := t
				if score == 0 {
					score = p + c
				}
				if score > bestTotal {
					promptTokens, completionTokens, totalTokens = p, c, t
					bestTotal = score
				}
			}
			for _, vv := range x {
				walk(vv)
			}
		case []any:
			for _, vv := range x {
				walk(vv)
			}
		}
	}
	walk(payload)
	if totalTokens == 0 {
		totalTokens = promptTokens + completionTokens
	}
	return promptTokens, completionTokens, totalTokens
}

func parseUsageObject(m map[string]any) (prompt int, completion int, total int, ok bool) {
	usageRaw, hasUsage := m["usage"]
	if hasUsage {
		if usageMap, mapOK := usageRaw.(map[string]any); mapOK {
			return parseUsageFields(usageMap)
		}
	}
	return parseUsageFields(m)
}

func parseUsageFields(m map[string]any) (prompt int, completion int, total int, ok bool) {
	prompt = int(firstFloat(m, "prompt_tokens", "input_tokens"))
	completion = int(firstFloat(m, "completion_tokens", "output_tokens"))
	total = int(firstFloat(m, "total_tokens"))
	if prompt == 0 && completion == 0 && total == 0 {
		return 0, 0, 0, false
	}
	if total == 0 {
		total = prompt + completion
	}
	return prompt, completion, total, true
}

func firstFloat(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case json.Number:
			if f, err := n.Float64(); err == nil {
				return f
			}
		}
	}
	return 0
}

func extractClientUsageMeta(r *http.Request, cfg config.ServerConfig) clientUsageMeta {
	token := strings.TrimSpace(bearerToken(r.Header))
	uaRaw := strings.TrimSpace(r.Header.Get("User-Agent"))
	return clientUsageMeta{
		ClientType: classifyClientType(uaRaw),
		UserAgent:  normalizeUserAgent(uaRaw),
		ClientIP:   requestClientIP(r),
		APIKeyName: resolveIncomingTokenName(token, cfg),
	}
}

func normalizeUserAgent(userAgent string) string {
	ua := strings.TrimSpace(userAgent)
	if ua == "" {
		return "unknown"
	}
	const maxLen = 200
	if len(ua) <= maxLen {
		return ua
	}
	return strings.TrimSpace(ua[:maxLen]) + "..."
}

func requestClientIP(r *http.Request) string {
	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return ""
	}
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		return strings.TrimSpace(parsed)
	}
	return host
}

func classifyClientType(userAgent string) string {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return "unknown"
	}
	switch {
	case strings.Contains(ua, "openai-python"):
		return "openai-python"
	case strings.Contains(ua, "openai-node"):
		return "openai-node"
	case strings.Contains(ua, "openai-go"):
		return "openai-go"
	case strings.Contains(ua, "curl/"):
		return "curl"
	case strings.Contains(ua, "python-requests"), strings.Contains(ua, "python-httpx"), strings.Contains(ua, "python"):
		return "python"
	case strings.Contains(ua, "node"), strings.Contains(ua, "undici"), strings.Contains(ua, "axios"):
		return "nodejs"
	case strings.Contains(ua, "go-http-client"):
		return "go-http-client"
	case strings.Contains(ua, "mozilla/"):
		return "browser"
	default:
		return "other"
	}
}

func resolveIncomingTokenName(token string, cfg config.ServerConfig) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	for _, t := range cfg.IncomingTokens {
		if token != strings.TrimSpace(t.Key) {
			continue
		}
		expiresAt := strings.TrimSpace(t.ExpiresAt)
		if expiresAt != "" {
			ts, err := parseRFC3339(expiresAt)
			if err != nil || !nowUTC().Before(ts) {
				return ""
			}
		}
		name := strings.TrimSpace(t.Name)
		if name != "" {
			return name
		}
		return strings.TrimSpace(t.ID)
	}
	return ""
}

func computePromptAndGenerationTPS(promptTokens int, completionTokens int, initialLatency time.Duration, totalLatency time.Duration) (float64, float64) {
	const (
		minPhaseDuration = 250 * time.Millisecond
		maxMeasuredTPS   = 2000.0
	)
	if totalLatency <= 0 {
		totalLatency = time.Millisecond
	}
	if initialLatency <= 0 || initialLatency > totalLatency {
		initialLatency = totalLatency
	}
	promptSeconds := initialLatency.Seconds()
	if promptSeconds <= 0 {
		promptSeconds = 0.001
	}
	genDuration := totalLatency - initialLatency
	if genDuration <= 0 {
		genDuration = totalLatency
	}
	genSeconds := genDuration.Seconds()
	if genSeconds <= 0 {
		genSeconds = 0.001
	}
	promptTPS := 0.0
	genTPS := 0.0
	if promptTokens > 0 {
		promptTPS = float64(promptTokens) / promptSeconds
	}
	if promptTPS > maxMeasuredTPS {
		promptTPS = maxMeasuredTPS
	}
	if completionTokens > 0 {
		if genDuration < minPhaseDuration {
			genDuration = totalLatency
			if genDuration < minPhaseDuration {
				genDuration = minPhaseDuration
			}
			genSeconds = genDuration.Seconds()
		}
		genTPS = float64(completionTokens) / genSeconds
	}
	if genTPS > maxMeasuredTPS {
		genTPS = maxMeasuredTPS
	}
	return promptTPS, genTPS
}

func durationFromProviderSeconds(seconds float64) time.Duration {
	if seconds <= 0 {
		return 0
	}
	d := time.Duration(seconds * float64(time.Second))
	if d <= 0 {
		return time.Millisecond
	}
	return d
}

func estimatePromptTokensFromRequest(requestBody []byte) int {
	if len(requestBody) == 0 {
		return 0
	}
	var payload any
	if err := json.Unmarshal(requestBody, &payload); err != nil {
		return 0
	}
	text := extractPromptText(payload)
	return estimateTokensFromText(text)
}

func estimateCompletionTokensFromResponse(responseBody []byte) int {
	if len(responseBody) == 0 {
		return 0
	}
	var payload any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return 0
	}
	text := extractCompletionText(payload)
	return estimateTokensFromText(text)
}

func estimateTokensFromText(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	runes := utf8.RuneCountInString(text)
	if runes <= 0 {
		return 0
	}
	return (runes + 3) / 4
}

func extractPromptText(payload any) string {
	parts := make([]string, 0, 8)
	var walk func(v any, key string)
	walk = func(v any, key string) {
		switch x := v.(type) {
		case map[string]any:
			for k, vv := range x {
				lk := strings.ToLower(strings.TrimSpace(k))
				walk(vv, lk)
			}
		case []any:
			for _, vv := range x {
				walk(vv, key)
			}
		case string:
			if key == "prompt" || key == "input" || key == "instructions" || key == "system" || key == "content" || key == "text" {
				parts = append(parts, x)
			}
		}
	}
	walk(payload, "")
	return strings.Join(parts, "\n")
}

func extractCompletionText(payload any) string {
	parts := make([]string, 0, 8)
	var walk func(v any, key string)
	walk = func(v any, key string) {
		switch x := v.(type) {
		case map[string]any:
			for k, vv := range x {
				lk := strings.ToLower(strings.TrimSpace(k))
				switch lk {
				case "output_text", "delta":
					if s, ok := vv.(string); ok {
						parts = append(parts, s)
					}
				case "text":
					if s, ok := vv.(string); ok && key != "input" && key != "prompt" {
						parts = append(parts, s)
					}
				case "content":
					if s, ok := vv.(string); ok && (key == "message" || key == "delta" || key == "choice" || key == "output") {
						parts = append(parts, s)
					}
				}
				walk(vv, lk)
			}
		case []any:
			for _, vv := range x {
				walk(vv, key)
			}
		}
	}
	walk(payload, "")
	return strings.Join(parts, "\n")
}

func firstErr(ch <-chan error) error {
	select {
	case err := <-ch:
		return err
	default:
		return nil
	}
}

func isConversationCaptureEndpoint(path string) bool {
	switch strings.TrimSpace(path) {
	case "/v1/chat/completions", "/v1/completions", "/v1/embeddings", "/v1/responses":
		return true
	default:
		return false
	}
}

func normalizeConversationEndpoint(path string) string {
	switch strings.TrimSpace(path) {
	case "/v1/chat/completions":
		return "chat.completions"
	case "/v1/completions":
		return "completions"
	case "/v1/embeddings":
		return "embeddings"
	case "/v1/responses":
		return "responses"
	default:
		return strings.TrimSpace(path)
	}
}

func parseConversationRequestIDs(headers http.Header, body []byte) (conversationID string, previousResponseID string) {
	conversationID = strings.TrimSpace(firstHeaderValue(headers,
		"X-Conversation-ID",
		"Conversation-ID",
		"X-Session-ID",
	))
	previousResponseID = strings.TrimSpace(firstHeaderValue(headers,
		"X-Previous-Response-ID",
		"Previous-Response-ID",
	))
	if len(body) == 0 {
		return conversationID, previousResponseID
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return conversationID, previousResponseID
	}
	if conversationID == "" {
		conversationID = strings.TrimSpace(asStringAny(firstMapValueAny(
			payload,
			"conversation_id",
			"conversationId",
			"thread_id",
			"threadId",
		)))
		if conversationID == "" {
			if metadata, ok := payload["metadata"].(map[string]any); ok {
				conversationID = strings.TrimSpace(asStringAny(firstMapValueAny(
					metadata,
					"conversation_id",
					"conversationId",
					"thread_id",
					"threadId",
				)))
			}
		}
	}
	if previousResponseID == "" {
		previousResponseID = strings.TrimSpace(asStringAny(firstMapValueAny(payload, "previous_response_id", "previousResponseId")))
	}
	return conversationID, previousResponseID
}

func parseResponseID(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if rid := strings.TrimSpace(asStringAny(payload["id"])); rid != "" && rid != "<nil>" {
		return rid
	}
	if resp, ok := payload["response"].(map[string]any); ok {
		if rid := strings.TrimSpace(asStringAny(resp["id"])); rid != "" && rid != "<nil>" {
			return rid
		}
	}
	return ""
}

func extractPromptTextFromBody(body []byte) string {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(extractPromptText(payload))
}

func extractCompletionTextFromBody(body []byte) string {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(extractCompletionText(payload))
}

func capJSONBytes(in []byte, max int) []byte {
	if len(in) == 0 {
		return nil
	}
	normalize := func(b []byte) []byte {
		if json.Valid(b) {
			out := make([]byte, len(b))
			copy(out, b)
			return out
		}
		encoded, _ := json.Marshal(string(b))
		return encoded
	}
	if max <= 0 || len(in) <= max {
		return normalize(in)
	}
	trimmed := append([]byte(nil), in[:max]...)
	ellipsis := []byte(`"...(truncated)"`)
	if len(trimmed) > len(ellipsis) {
		copy(trimmed[len(trimmed)-len(ellipsis):], ellipsis)
	}
	return normalize(trimmed)
}

func capConversationRawText(in []byte, max int) string {
	if len(in) == 0 {
		return ""
	}
	if max <= 0 || len(in) <= max {
		return strings.ToValidUTF8(string(in), "\uFFFD")
	}
	const marker = "\n...(truncated)"
	trimmed := strings.ToValidUTF8(string(in[:max]), "\uFFFD")
	return strings.TrimSpace(trimmed) + marker
}

func conversationHeadersMapToRaw(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		val := strings.TrimSpace(headers[k])
		lines = append(lines, k+": "+val)
	}
	return strings.Join(lines, "\n")
}

func sanitizeConversationRequestHeaders(h http.Header) map[string]string {
	return sanitizeConversationHeaders(h, true)
}

func sanitizeConversationResponseHeaders(h http.Header) map[string]string {
	return sanitizeConversationHeaders(h, false)
}

func sanitizeConversationHeaders(h http.Header, request bool) map[string]string {
	if len(h) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, vals := range h {
		key := strings.ToLower(strings.TrimSpace(k))
		if key == "" {
			continue
		}
		if isSensitiveHeader(key) {
			out[key] = "[redacted]"
			continue
		}
		if !isAllowedConversationHeader(key, request) {
			continue
		}
		out[key] = strings.TrimSpace(strings.Join(vals, ", "))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isSensitiveHeader(key string) bool {
	return key == "authorization" || key == "cookie" || key == "set-cookie" ||
		strings.Contains(key, "token") || strings.Contains(key, "secret") ||
		strings.Contains(key, "api-key") || strings.Contains(key, "apikey") ||
		strings.Contains(key, "auth")
}

func isAllowedConversationHeader(key string, request bool) bool {
	if strings.HasPrefix(key, "x-") {
		return true
	}
	if strings.HasPrefix(key, "ratelimit-") {
		return true
	}
	if strings.HasPrefix(key, "openai-") {
		return true
	}
	if request {
		switch key {
		case "user-agent", "content-type", "accept", "origin", "referer":
			return true
		}
		return false
	}
	switch key {
	case "content-type", "retry-after":
		return true
	}
	return false
}

func firstMapValueAny(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func firstHeaderValue(h http.Header, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(h.Get(k)); v != "" {
			return v
		}
	}
	return ""
}

func asStringAny(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
