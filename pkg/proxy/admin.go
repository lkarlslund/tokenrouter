package proxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/lkarlslund/tokenrouter/pkg/assets"
	"github.com/lkarlslund/tokenrouter/pkg/cache"
	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/conversations"
	"github.com/lkarlslund/tokenrouter/pkg/logstore"
	"github.com/lkarlslund/tokenrouter/pkg/pricing"
	"github.com/lkarlslund/tokenrouter/pkg/version"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const adminSessionCookie = "opp_admin_session"
const adminInstanceHeader = "X-OPP-Instance-ID"

type AdminHandler struct {
	store                   *config.ServerConfigStore
	stats                   *StatsStore
	resolver                *ProviderResolver
	pricing                 *pricing.Manager
	conversations           *conversations.Store
	logs                    *logstore.Store
	healthChecker           *ProviderHealthChecker
	instance                string
	oauthMu                 sync.Mutex
	oauthPending            map[string]*oauthSession
	oauthSrvMu              sync.Mutex
	oauthSrv                *http.Server
	oauthSrvAddr            string
	quotaMu                 sync.Mutex
	quotaCache              *cache.TTLMap[string, quotaCacheValue]
	quotaCachePath          string
	quotaPersistMu          sync.Mutex
	quotaPersistPending     bool
	statsCache              *cache.TTLMap[int64, StatsSummary]
	wsMu                    sync.Mutex
	wsClients               map[*adminWSClient]struct{}
	statsNotifyAt           time.Time
	logNotifyAt             time.Time
	networkMu               sync.Mutex
	networkSwitch           map[string]pendingNetworkSwitch
	addListener             func(string) error
	removeListener          func(string) error
	listListeners           func() []string
	runAccessTokenCleanup   func()
	conversationViewCacheMu sync.Mutex
	conversationViewCache   map[string]conversationViewCacheEntry
	benchmarkMu             sync.Mutex
	benchmarkRun            *benchmarkRun
}

type adminWSClient struct {
	ch          chan []byte
	updateEvery time.Duration
	lastRefresh time.Time
	realtime    bool
}

type pendingNetworkSwitch struct {
	ID       string
	OldAddr  string
	NewAddr  string
	HTTPMode string
	Created  time.Time
}

type conversationViewCacheEntry struct {
	Title     string
	Views     []conversationRecordView
	ExpiresAt time.Time
}

type adminAuthContextKey struct{}

type oauthSession struct {
	Provider     string
	Verifier     string
	RedirectURI  string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Originator   string
	CreatedAt    time.Time
	Done         bool
	Error        string
	AccessToken  string
	RefreshToken string
	ExpiresAt    string
	AccountID    string
	BaseURL      string
}

type benchmarkRun struct {
	ID               string                  `json:"id"`
	StartedAt        string                  `json:"started_at"`
	Status           string                  `json:"status"`
	Strategy         benchmarkStrategy       `json:"strategy"`
	Models           []benchmarkModelProgress `json:"models"`
	TotalPrompt      int                     `json:"total_prompt_tokens"`
	TotalCompletion  int                     `json:"total_completion_tokens"`
	TotalTokens      int                     `json:"total_tokens"`
	Error            string                  `json:"error,omitempty"`
	cancel           context.CancelFunc
}

type benchmarkStrategy struct {
	ChatsPerModel   int `json:"chats_per_model"`
	MessagesPerChat int `json:"messages_per_chat"`
	StopAfterTokens int `json:"stop_after_tokens"`
}

type benchmarkModelProgress struct {
	Model            string `json:"model"`
	Status           string `json:"status"`
	ChatsTotal       int    `json:"chats_total"`
	ChatsDone        int    `json:"chats_done"`
	MessagesTotal    int    `json:"messages_total"`
	MessagesDone     int    `json:"messages_done"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	LastError        string `json:"last_error,omitempty"`
}

type quotaCacheValue struct {
	Snapshot   ProviderQuotaSnapshot
	LastGood   ProviderQuotaSnapshot
	Refreshing bool
}

const quotaRefreshOK = 15 * time.Minute
const quotaRefreshError = 30 * time.Second
const statsRefreshInterval = 15 * time.Minute
const adminWSCheckInterval = 1 * time.Second
const adminWSRealtimeIdleRefresh = 5 * time.Minute
const quotaPersistDebounce = 250 * time.Millisecond
const conversationViewCacheTTL = 5 * time.Second

type modelListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type conversationRecordView = conversations.ConversationRecordView

type persistedQuotaCacheFile struct {
	SavedAt string                              `json:"saved_at,omitempty"`
	Entries map[string]persistedQuotaCacheEntry `json:"entries,omitempty"`
}

type persistedQuotaCacheEntry struct {
	Value     quotaCacheValue `json:"value"`
	ExpiresAt string          `json:"expires_at,omitempty"`
}

func NewAdminHandler(store *config.ServerConfigStore, stats *StatsStore, resolver *ProviderResolver, pricingMgr *pricing.Manager, healthChecker *ProviderHealthChecker, instanceID string) *AdminHandler {
	quotaCachePath := quotaCachePathForStore(store)
	h := &AdminHandler{
		store:                 store,
		stats:                 stats,
		resolver:              resolver,
		pricing:               pricingMgr,
		healthChecker:         healthChecker,
		instance:              instanceID,
		oauthPending:          map[string]*oauthSession{},
		oauthSrvAddr:          "127.0.0.1:1455",
		quotaCache:            cache.NewTTLMap[string, quotaCacheValue](),
		quotaCachePath:        quotaCachePath,
		statsCache:            cache.NewTTLMap[int64, StatsSummary](),
		wsClients:             map[*adminWSClient]struct{}{},
		networkSwitch:         map[string]pendingNetworkSwitch{},
		conversationViewCache: map[string]conversationViewCacheEntry{},
	}
	h.loadQuotaCacheFromDisk()
	go h.runWSScheduler()
	return h
}

func quotaCachePathForStore(store *config.ServerConfigStore) string {
	cacheDir := filepath.Dir(config.DefaultUsageStatsPath())
	storePath := ""
	if store != nil {
		storePath = strings.TrimSpace(store.Path())
	}
	if storePath == "" {
		return filepath.Join(cacheDir, "admin-quota-cache.json")
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(filepath.Clean(storePath)))
	return filepath.Join(cacheDir, fmt.Sprintf("admin-quota-cache-%08x.json", hasher.Sum32()))
}

func (h *AdminHandler) SetNetworkListenerControl(add func(string) error, remove func(string) error, list func() []string) {
	if h == nil {
		return
	}
	h.networkMu.Lock()
	h.addListener = add
	h.removeListener = remove
	h.listListeners = list
	h.networkMu.Unlock()
}

func (h *AdminHandler) SetAccessTokenCleanup(run func()) {
	if h == nil {
		return
	}
	h.networkMu.Lock()
	h.runAccessTokenCleanup = run
	h.networkMu.Unlock()
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.With(h.withRuntimeInstanceHeader, h.requireAdminPage).Get("/admin", h.page)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminPage).Get("/admin/", h.page)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminPage).Get("/admin/tab/{tab}", h.tabPage)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminPage).Get("/admin/ws", h.adminWebsocket)
	r.With(h.withRuntimeInstanceHeader).MethodFunc(http.MethodGet, "/admin/setup", h.legacySetupRedirect)
	r.With(h.withRuntimeInstanceHeader).MethodFunc(http.MethodPost, "/admin/setup", h.legacySetupRedirect)
	r.With(h.withRuntimeInstanceHeader).MethodFunc(http.MethodGet, "/admin/login", h.login)
	r.With(h.withRuntimeInstanceHeader).MethodFunc(http.MethodPost, "/admin/login", h.login)
	r.With(h.withRuntimeInstanceHeader).MethodFunc(http.MethodPost, "/admin/logout", h.logout)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/stats", h.statsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/settings/security", h.securitySettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Put("/admin/api/settings/security", h.securitySettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/settings/tls", h.tlsSettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Put("/admin/api/settings/tls", h.tlsSettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/settings/network", h.networkSettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/settings/network/apply", h.networkApplyAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/settings/network/confirm", h.networkConfirmAPI)
	r.With(h.withRuntimeInstanceHeader).Get("/admin/api/network/probe", h.networkProbeAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/settings/tls/test-certificate", h.tlsTestCertificateAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/settings/tls/renew", h.tlsRenewCertificateAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/version", h.versionAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireTokenRole(config.TokenRoleAdmin, config.TokenRoleKeymaster)).Get("/admin/api/access-tokens", h.accessTokensAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireTokenRole(config.TokenRoleAdmin, config.TokenRoleKeymaster)).Post("/admin/api/access-tokens", h.accessTokensAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireTokenRole(config.TokenRoleAdmin, config.TokenRoleKeymaster)).Put("/admin/api/access-tokens/{id}", h.accessTokenByIDAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireTokenRole(config.TokenRoleAdmin, config.TokenRoleKeymaster)).Delete("/admin/api/access-tokens/{id}", h.accessTokenByIDAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/pricing", h.pricingAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/pricing/refresh", h.refreshPricingAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/providers/popular", h.popularProvidersAPI)
	r.With(h.withRuntimeInstanceHeader).Get("/admin/static/*", h.adminStaticAsset)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/providers/device-code", h.providerDeviceCodeAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/providers/device-token", h.providerDeviceTokenAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/providers/oauth/start", h.providerOAuthStartAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/providers/oauth/result", h.providerOAuthResultAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/providers/refresh", h.refreshProvidersAPI)
	r.With(h.withRuntimeInstanceHeader).Get("/admin/oauth/callback", h.providerOAuthCallbackPage)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/providers", h.providersAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/providers", h.providersAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/providers/test", h.testProviderAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Put("/admin/api/providers/{name}", h.providerByNameAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Delete("/admin/api/providers/{name}", h.providerByNameAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/models/refresh", h.refreshModelsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/models", h.modelsCatalogAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/models/catalog", h.modelsCatalogAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/benchmark", h.benchmarkStatusAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/benchmark/start", h.benchmarkStartAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Post("/admin/api/benchmark/cancel", h.benchmarkCancelAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/settings/conversations", h.conversationsSettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Put("/admin/api/settings/conversations", h.conversationsSettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/settings/logs", h.logsSettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Put("/admin/api/settings/logs", h.logsSettingsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/logs", h.logsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Delete("/admin/api/logs", h.logsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/conversations", h.conversationsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Delete("/admin/api/conversations", h.conversationsAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Get("/admin/api/conversations/{id}", h.conversationByIDAPI)
	r.With(h.withRuntimeInstanceHeader, h.requireAdminAPI).Delete("/admin/api/conversations/{id}", h.conversationByIDAPI)
}

func (h *AdminHandler) runWSScheduler() {
	t := time.NewTicker(adminWSCheckInterval)
	defer t.Stop()
	for range t.C {
		h.pushScheduledRefresh()
	}
}

func (h *AdminHandler) broadcastAdminEvent(event map[string]any) {
	if event == nil {
		return
	}
	b, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.wsMu.Lock()
	defer h.wsMu.Unlock()
	isRefresh := strings.TrimSpace(fmt.Sprintf("%v", event["type"])) == "refresh"
	now := time.Now().UTC()
	for client := range h.wsClients {
		if isRefresh && !client.realtime {
			continue
		}
		select {
		case client.ch <- b:
			if isRefresh {
				client.lastRefresh = now
			}
		default:
			// Coalesce bursty updates by dropping one stale queued event,
			// then retry enqueueing the newest event.
			select {
			case <-client.ch:
			default:
			}
			select {
			case client.ch <- b:
				if isRefresh {
					client.lastRefresh = now
				}
			default:
			}
		}
	}
}

func (h *AdminHandler) RecordConversation(in conversations.CaptureInput) bool {
	if h == nil || h.conversations == nil {
		slog.Warn("conversation capture skipped", "reason", "store_unavailable", "endpoint", strings.TrimSpace(in.Endpoint), "provider", strings.TrimSpace(in.Provider), "model", strings.TrimSpace(in.Model), "status", in.StatusCode)
		return false
	}
	rec, ok := h.conversations.Add(in)
	if !ok {
		settings := h.conversations.Settings()
		reason := "rejected"
		if !settings.Enabled {
			reason = "disabled"
		}
		slog.Warn("conversation capture skipped", "reason", reason, "enabled", settings.Enabled, "endpoint", strings.TrimSpace(in.Endpoint), "provider", strings.TrimSpace(in.Provider), "model", strings.TrimSpace(in.Model), "status", in.StatusCode)
		return false
	}
	slog.Info("conversation captured", "conversation_key", rec.ConversationKey, "endpoint", strings.TrimSpace(rec.Endpoint), "provider", strings.TrimSpace(rec.Provider), "model", strings.TrimSpace(rec.Model), "status", rec.StatusCode, "latency_ms", rec.LatencyMS)
	h.invalidateConversationViewCacheForKey(rec.ConversationKey)
	h.broadcastAdminEvent(map[string]any{
		"type":             "conversation_append",
		"conversation_key": rec.ConversationKey,
		"id":               rec.ID,
		"updated_at":       rec.UpdatedAt.Format(time.RFC3339Nano),
	})
	return true
}

func (h *AdminHandler) conversationViewCacheKey(id string, includeInternal bool) string {
	return id + "|" + strconv.FormatBool(includeInternal)
}

func cloneConversationViews(in []conversationRecordView) []conversationRecordView {
	if len(in) == 0 {
		return []conversationRecordView{}
	}
	out := make([]conversationRecordView, len(in))
	copy(out, in)
	return out
}

func (h *AdminHandler) getConversationViewCache(id string, includeInternal bool) (string, []conversationRecordView, bool) {
	if h == nil {
		return "", nil, false
	}
	key := h.conversationViewCacheKey(id, includeInternal)
	now := time.Now().UTC()
	h.conversationViewCacheMu.Lock()
	defer h.conversationViewCacheMu.Unlock()
	entry, ok := h.conversationViewCache[key]
	if !ok {
		return "", nil, false
	}
	if now.After(entry.ExpiresAt) {
		delete(h.conversationViewCache, key)
		return "", nil, false
	}
	return entry.Title, cloneConversationViews(entry.Views), true
}

func (h *AdminHandler) setConversationViewCache(id string, includeInternal bool, title string, views []conversationRecordView) {
	if h == nil {
		return
	}
	key := h.conversationViewCacheKey(id, includeInternal)
	h.conversationViewCacheMu.Lock()
	h.conversationViewCache[key] = conversationViewCacheEntry{
		Title:     strings.TrimSpace(title),
		Views:     cloneConversationViews(views),
		ExpiresAt: time.Now().UTC().Add(conversationViewCacheTTL),
	}
	h.conversationViewCacheMu.Unlock()
}

func (h *AdminHandler) invalidateConversationViewCacheForKey(id string) {
	if h == nil || strings.TrimSpace(id) == "" {
		return
	}
	id = strings.TrimSpace(id)
	h.conversationViewCacheMu.Lock()
	delete(h.conversationViewCache, h.conversationViewCacheKey(id, false))
	delete(h.conversationViewCache, h.conversationViewCacheKey(id, true))
	h.conversationViewCacheMu.Unlock()
}

func (h *AdminHandler) invalidateConversationViewCacheAll() {
	if h == nil {
		return
	}
	h.conversationViewCacheMu.Lock()
	h.conversationViewCache = map[string]conversationViewCacheEntry{}
	h.conversationViewCacheMu.Unlock()
}

func (h *AdminHandler) registerWSClient(c *adminWSClient) {
	h.wsMu.Lock()
	defer h.wsMu.Unlock()
	h.wsClients[c] = struct{}{}
}

func (h *AdminHandler) unregisterWSClient(c *adminWSClient) {
	h.wsMu.Lock()
	defer h.wsMu.Unlock()
	if _, ok := h.wsClients[c]; ok {
		delete(h.wsClients, c)
		close(c.ch)
	}
}

func (h *AdminHandler) adminWebsocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(req *http.Request) bool {
			origin := strings.TrimSpace(req.Header.Get("Origin"))
			if origin == "" {
				return true
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			return strings.EqualFold(u.Host, req.Host)
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	client := &adminWSClient{
		ch:          make(chan []byte, 16),
		updateEvery: adminWSRealtimeIdleRefresh,
		realtime:    true,
	}
	h.registerWSClient(client)
	defer h.unregisterWSClient(client)
	h.sendWSRefreshToClient(client)
	pingTicker := time.NewTicker(25 * time.Second)
	defer pingTicker.Stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			h.handleWSClientMessage(client, payload)
		}
	}()
	for {
		select {
		case <-done:
			return
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return
			}
		case msg, ok := <-client.ch:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}

func (h *AdminHandler) handleWSClientMessage(client *adminWSClient, payload []byte) {
	if client == nil || len(payload) == 0 {
		return
	}
	var msg struct {
		Type        string `json:"type"`
		UpdateSpeed string `json:"update_speed"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	if strings.TrimSpace(msg.Type) != "subscribe" {
		return
	}
	h.wsMu.Lock()
	defer h.wsMu.Unlock()
	client.updateEvery, client.realtime = parseWSUpdateEvery(msg.UpdateSpeed)
}

func parseWSUpdateEvery(updateSpeed string) (time.Duration, bool) {
	switch strings.TrimSpace(strings.ToLower(updateSpeed)) {
	case "2s":
		return 2 * time.Second, false
	case "10s":
		return 10 * time.Second, false
	case "30s":
		return 30 * time.Second, false
	case "1m":
		return 1 * time.Minute, false
	case "5m":
		return 5 * time.Minute, false
	case "15m":
		return 15 * time.Minute, false
	case "disabled":
		return 0, false
	case "realtime":
		return adminWSRealtimeIdleRefresh, true
	default:
		return adminWSRealtimeIdleRefresh, true
	}
}

func (h *AdminHandler) pushScheduledRefresh() {
	now := time.Now().UTC()
	h.wsMu.Lock()
	defer h.wsMu.Unlock()
	for client := range h.wsClients {
		every := client.updateEvery
		if every <= 0 {
			continue
		}
		if !client.lastRefresh.IsZero() && now.Sub(client.lastRefresh) < every {
			continue
		}
		select {
		case client.ch <- []byte(`{"type":"refresh"}`):
			client.lastRefresh = now
		default:
		}
	}
}

func (h *AdminHandler) sendWSRefreshToClient(client *adminWSClient) {
	if client == nil {
		return
	}
	select {
	case client.ch <- []byte(`{"type":"refresh"}`):
		client.lastRefresh = time.Now().UTC()
	default:
	}
}

func (h *AdminHandler) NotifyStatsChanged() {
	if h == nil {
		return
	}
	now := time.Now().UTC()
	h.wsMu.Lock()
	if !h.statsNotifyAt.IsZero() && now.Sub(h.statsNotifyAt) < 1500*time.Millisecond {
		h.wsMu.Unlock()
		return
	}
	h.statsNotifyAt = now
	h.wsMu.Unlock()
	h.notifyAdminChanged("stats")
}

func (h *AdminHandler) NotifyLogChanged() {
	if h == nil {
		return
	}
	now := time.Now().UTC()
	h.wsMu.Lock()
	if !h.logNotifyAt.IsZero() && now.Sub(h.logNotifyAt) < 250*time.Millisecond {
		h.wsMu.Unlock()
		return
	}
	h.logNotifyAt = now
	h.wsMu.Unlock()
	h.notifyAdminChanged("log")
}

func (h *AdminHandler) notifyAdminChanged(scope string) {
	if h == nil {
		return
	}
	s := strings.TrimSpace(strings.ToLower(scope))
	if s == "" {
		s = "all"
	}
	h.broadcastAdminEvent(map[string]any{
		"type":  "changed",
		"scope": s,
	})
}

func (h *AdminHandler) withRuntimeInstanceHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.instance != "" {
			w.Header().Set(adminInstanceHeader, h.instance)
		}
		next.ServeHTTP(w, r)
	})
}

func (h *AdminHandler) login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if h.isAuthenticated(r) {
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		next := "/admin"
		if q := r.URL.Query().Get("next"); strings.HasPrefix(q, "/admin") {
			next = q
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t, err := getTemplates()
		if err != nil {
			http.Error(w, "failed to render login page", http.StatusInternalServerError)
			return
		}
		_ = t.ExecuteTemplate(w, "login.html", struct {
			Next  string
			Error string
		}{Next: next})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		key := strings.TrimSpace(r.FormValue("api_key"))
		next := "/admin"
		if q := r.FormValue("next"); strings.HasPrefix(q, "/admin") {
			next = q
		}
		identity, ok := resolveAuthIdentity(key, h.store.Snapshot())
		if !ok || identity.Role != config.TokenRoleAdmin {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			t, err := getTemplates()
			if err != nil {
				http.Error(w, "failed to render login page", http.StatusInternalServerError)
				return
			}
			_ = t.ExecuteTemplate(w, "login.html", struct {
				Next  string
				Error string
			}{Next: next, Error: "Invalid API key"})
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     adminSessionCookie,
			Value:    key,
			Path:     "/admin",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
			MaxAge:   86400,
		})
		http.Redirect(w, r, next, http.StatusFound)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) legacySetupRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (h *AdminHandler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func (h *AdminHandler) requireAdminPage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := h.store.Snapshot()
		if cfg.AllowLocalhostNoAuth && requestIsTrustedNoAuth(r, cfg) {
			next.ServeHTTP(w, r)
			return
		}
		if !h.isAuthenticated(r) {
			http.Redirect(w, r, "/admin/login?next="+url.QueryEscape(r.URL.Path), http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *AdminHandler) requireAdminAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := h.store.Snapshot()
		if cfg.AllowLocalhostNoAuth && requestIsTrustedNoAuth(r, cfg) {
			identity := tokenAuthIdentity{
				Role:    config.TokenRoleAdmin,
				IsAdmin: true,
			}
			ctx := context.WithValue(r.Context(), adminAuthContextKey{}, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if !config.HasAdminToken(cfg.IncomingTokens) {
			http.Error(w, "admin setup required", http.StatusServiceUnavailable)
			return
		}
		identity, ok := h.requestIdentity(r)
		if !ok || identity.Role != config.TokenRoleAdmin {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), adminAuthContextKey{}, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *AdminHandler) requireTokenRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, role := range allowedRoles {
		role = config.NormalizeIncomingTokenRole(role)
		if role == "" {
			continue
		}
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := h.store.Snapshot()
			if cfg.AllowLocalhostNoAuth && requestIsTrustedNoAuth(r, cfg) {
				identity := tokenAuthIdentity{
					Role:    config.TokenRoleAdmin,
					IsAdmin: true,
				}
				ctx := context.WithValue(r.Context(), adminAuthContextKey{}, identity)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if !config.HasAdminToken(cfg.IncomingTokens) {
				http.Error(w, "admin setup required", http.StatusServiceUnavailable)
				return
			}
			identity, ok := h.requestIdentity(r)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			allowedByHierarchy := false
			for required := range allowed {
				if config.RoleAtLeast(identity.Role, required) {
					allowedByHierarchy = true
					break
				}
			}
			if !allowedByHierarchy {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), adminAuthContextKey{}, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *AdminHandler) isAuthenticated(r *http.Request) bool {
	identity, ok := h.requestIdentity(r)
	return ok && identity.Role == config.TokenRoleAdmin
}

func (h *AdminHandler) adminTokensFromRequest(r *http.Request) []string {
	tokens := make([]string, 0, 3)
	if tok := bearerToken(r.Header); tok != "" {
		tokens = append(tokens, tok)
	}
	if tok := strings.TrimSpace(r.URL.Query().Get("key")); tok != "" {
		tokens = append(tokens, tok)
	}
	if c, err := r.Cookie(adminSessionCookie); err == nil && strings.TrimSpace(c.Value) != "" {
		tokens = append(tokens, strings.TrimSpace(c.Value))
	}
	return tokens
}

func (h *AdminHandler) requestIdentity(r *http.Request) (tokenAuthIdentity, bool) {
	cfg := h.store.Snapshot()
	for _, token := range h.adminTokensFromRequest(r) {
		if identity, ok := resolveAuthIdentity(token, cfg); ok {
			return identity, true
		}
	}
	return tokenAuthIdentity{}, false
}

func adminIdentityFromContext(ctx context.Context) (tokenAuthIdentity, bool) {
	raw := ctx.Value(adminAuthContextKey{})
	identity, ok := raw.(tokenAuthIdentity)
	return identity, ok
}

func safeEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (h *AdminHandler) page(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/" {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := getTemplates()
	if err != nil {
		http.Error(w, "failed to load admin template", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "admin.html", struct{}{}); err != nil {
		http.Error(w, "failed to render admin page", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) tabPage(w http.ResponseWriter, r *http.Request) {
	tab := strings.TrimSpace(chi.URLParam(r, "tab"))
	if tab == "" {
		http.NotFound(w, r)
		return
	}
	tabTemplates := map[string]string{
		"status":        "admin_tab_status",
		"conversations": "admin_tab_conversations",
		"log":           "admin_tab_log",
		"quota":         "admin_tab_quota",
		"providers":     "admin_tab_providers",
		"access":        "admin_tab_access",
		"network":       "admin_tab_network",
		"models":        "admin_tab_models",
		"performance":   "admin_tab_performance",
		"benchmark":     "admin_tab_benchmark",
	}
	name, ok := tabTemplates[tab]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := getTemplates()
	if err != nil {
		http.Error(w, "failed to load admin template", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, struct{}{}); err != nil {
		http.Error(w, "failed to render admin tab", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) statsAPI(w http.ResponseWriter, r *http.Request) {
	period := time.Hour
	if raw := r.URL.Query().Get("period_seconds"); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			period = time.Duration(sec) * time.Second
		}
	}
	force := r.URL.Query().Get("force") == "1"
	periodKey := int64(period / time.Second)
	now := time.Now().UTC()
	if !force {
		if summary, ok := h.statsCache.GetFresh(periodKey, now); ok {
			providers := h.catalogProviders()
			quotaProviders := h.quotaProviders()
			names := make([]string, 0, len(providers))
			for _, p := range providers {
				names = append(names, p.Name)
			}
			if h.healthChecker != nil {
				summary.ProvidersAvailable, summary.ProvidersOnline = h.healthChecker.AvailabilitySummary(names)
			} else {
				summary.ProvidersAvailable = len(names)
			}
			summary.ProviderQuotas = h.readProviderQuotas(r.Context(), quotaProviders, false)
			writeJSON(w, http.StatusOK, summary)
			return
		}
	}

	summary := h.stats.Summary(period)
	providers := h.catalogProviders()
	quotaProviders := h.quotaProviders()
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		names = append(names, p.Name)
	}
	if h.healthChecker != nil {
		summary.ProvidersAvailable, summary.ProvidersOnline = h.healthChecker.AvailabilitySummary(names)
	} else {
		summary.ProvidersAvailable = len(names)
	}
	summary.ProviderQuotas = h.readProviderQuotas(r.Context(), quotaProviders, force)
	h.statsCache.SetWithTTL(periodKey, summary, now, statsRefreshInterval)
	writeJSON(w, http.StatusOK, summary)
}

func (h *AdminHandler) readProviderQuotas(ctx context.Context, providers []config.ProviderConfig, forceRefresh bool) map[string]ProviderQuotaSnapshot {
	byName, err := popularProvidersByName()
	if err != nil {
		return nil
	}
	out := map[string]ProviderQuotaSnapshot{}
	for _, p := range providers {
		providerType := providerTypeOrName(p)
		preset, ok := byName[providerType]
		if !ok {
			preset = assets.PopularProvider{
				ProviderConfig: config.ProviderConfig{Name: providerType},
				DisplayName:    p.Name,
			}
		}
		if !p.Enabled {
			snap := newProviderQuotaSnapshot(time.Now().UTC(), p, preset, quotaReaderFromPreset(preset))
			snap.Status = "disabled"
			snap.Error = "provider disabled"
			out[p.Name] = snap
			continue
		}
		snap := h.readProviderQuotaCached(ctx, p, preset, forceRefresh)
		out[p.Name] = snap
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func quotaReaderFromPreset(preset assets.PopularProvider) string {
	reader := strings.TrimSpace(preset.QuotaReader)
	if reader == "" {
		return "header_auto"
	}
	return reader
}

func popularProvidersByName() (map[string]assets.PopularProvider, error) {
	popular, err := getPopularProviders()
	if err != nil {
		return nil, err
	}
	byName := make(map[string]assets.PopularProvider, len(popular))
	for _, p := range popular {
		byName[p.Name] = p
	}
	return byName, nil
}

func presetForProvider(p config.ProviderConfig) (assets.PopularProvider, bool) {
	byName, err := popularProvidersByName()
	if err != nil {
		return assets.PopularProvider{}, false
	}
	preset, ok := byName[providerTypeOrName(p)]
	return preset, ok
}

func providerBaseURLOrPreset(p config.ProviderConfig, preset assets.PopularProvider) string {
	baseURL := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if baseURL != "" {
		return baseURL
	}
	return strings.TrimRight(strings.TrimSpace(preset.BaseURL), "/")
}

func (h *AdminHandler) loadQuotaCacheFromDisk() {
	if h == nil || strings.TrimSpace(h.quotaCachePath) == "" || h.quotaCache == nil {
		return
	}
	var persisted persistedQuotaCacheFile
	if err := cache.LoadJSON(h.quotaCachePath, &persisted); err != nil {
		if !errors.Is(err, cache.ErrNotFound) {
			slog.Warn("failed to load admin quota cache", "path", h.quotaCachePath, "err", err)
		}
		return
	}
	now := time.Now().UTC()
	for provider, entry := range persisted.Entries {
		name := strings.TrimSpace(provider)
		if name == "" {
			continue
		}
		val := entry.Value
		val.Refreshing = false
		if val.Snapshot.Provider == "" {
			continue
		}
		if val.LastGood.Provider == "" && strings.EqualFold(strings.TrimSpace(val.Snapshot.Status), "ok") {
			val.LastGood = val.Snapshot
		}
		expiresAt := now
		if ts := strings.TrimSpace(entry.ExpiresAt); ts != "" {
			if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
				expiresAt = parsed.UTC()
			}
		}
		h.quotaCache.SetWithExpiry(name, val, expiresAt)
	}
}

func (h *AdminHandler) persistQuotaCacheToDisk() {
	if h == nil || strings.TrimSpace(h.quotaCachePath) == "" || h.quotaCache == nil {
		return
	}
	entries := h.quotaCache.Entries()
	out := persistedQuotaCacheFile{
		SavedAt: time.Now().UTC().Format(time.RFC3339),
		Entries: make(map[string]persistedQuotaCacheEntry, len(entries)),
	}
	for provider, entry := range entries {
		name := strings.TrimSpace(provider)
		if name == "" {
			continue
		}
		val := entry.Value
		val.Refreshing = false
		if val.Snapshot.Provider == "" {
			continue
		}
		persisted := persistedQuotaCacheEntry{Value: val}
		if !entry.ExpiresAt.IsZero() {
			persisted.ExpiresAt = entry.ExpiresAt.UTC().Format(time.RFC3339)
		}
		out.Entries[name] = persisted
	}
	if len(out.Entries) == 0 {
		return
	}
	if err := cache.SaveJSON(h.quotaCachePath, out); err != nil {
		slog.Warn("failed to persist admin quota cache", "path", h.quotaCachePath, "err", err)
	}
}

func (h *AdminHandler) scheduleQuotaCachePersist() {
	if h == nil || strings.TrimSpace(h.quotaCachePath) == "" {
		return
	}
	h.quotaPersistMu.Lock()
	if h.quotaPersistPending {
		h.quotaPersistMu.Unlock()
		return
	}
	h.quotaPersistPending = true
	h.quotaPersistMu.Unlock()
	go func() {
		time.Sleep(quotaPersistDebounce)
		h.persistQuotaCacheToDisk()
		h.quotaPersistMu.Lock()
		h.quotaPersistPending = false
		h.quotaPersistMu.Unlock()
	}()
}

func (h *AdminHandler) setQuotaCacheWithExpiry(provider string, val quotaCacheValue, expiresAt time.Time) {
	if h == nil || h.quotaCache == nil {
		return
	}
	h.quotaCache.SetWithExpiry(provider, val, expiresAt)
	h.scheduleQuotaCachePersist()
}

func (h *AdminHandler) setQuotaCacheWithTTL(provider string, val quotaCacheValue, now time.Time, ttl time.Duration) {
	if h == nil || h.quotaCache == nil {
		return
	}
	h.quotaCache.SetWithTTL(provider, val, now, ttl)
	h.scheduleQuotaCachePersist()
}

func quotaProbeModelCandidates(preset assets.PopularProvider, modelsBody []byte, fallbacks ...string) []string {
	allFallbacks := make([]string, 0, len(preset.QuotaProbeModels)+len(fallbacks))
	for _, id := range preset.QuotaProbeModels {
		allFallbacks = append(allFallbacks, strings.TrimSpace(id))
	}
	allFallbacks = append(allFallbacks, fallbacks...)
	return modelIDsFromModelsBody(modelsBody, allFallbacks...)
}

func (h *AdminHandler) runQuotaReader(ctx context.Context, p config.ProviderConfig, preset assets.PopularProvider, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	switch quotaReaderFromPreset(preset) {
	case "openai_codex":
		return h.readOpenAICodexQuota(ctx, p, snap)
	case "google_antigravity":
		return h.readGoogleAntigravityQuota(ctx, p, snap)
	case "openrouter_key":
		return h.readOpenRouterQuota(ctx, p, snap)
	case "nvidia_unknown":
		return h.readNVIDIAUnknownQuota(ctx, p, snap)
	case "header_auto":
		return h.readAutoHeaderQuota(ctx, p, snap)
	case "groq_headers":
		return h.readGroqQuota(ctx, p, snap)
	case "mistral_headers":
		return h.readMistralQuota(ctx, p, snap)
	case "huggingface_headers":
		return h.readHuggingFaceQuota(ctx, p, snap)
	case "cerebras_headers":
		return h.readCerebrasQuota(ctx, p, snap)
	default:
		return h.readAutoHeaderQuota(ctx, p, snap)
	}
}

func (h *AdminHandler) readAutoHeaderQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	preset, _ := presetForProvider(p)
	token := providerTokenPreferAPIKey(p)
	allowNoAuth := token == "" && preset.PublicFreeNoAuth
	if token == "" && !allowNoAuth {
		snap.Error = "missing api key"
		return snap
	}
	baseURL := providerBaseURLOrPreset(p, preset)
	if baseURL == "" {
		snap.Error = "base_url is required"
		return snap
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		snap.Error = "invalid base_url"
		return snap
	}
	u.Path = joinProviderPath(u.Path, "/v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		snap.Error = "failed to build request"
		return snap
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		if allowNoAuth {
			snap.Status = "ok"
			snap.Error = ""
			snap.PlanType = "public"
			snap.LeftPercent = 100
			return snap
		}
		snap.Error = err.Error()
		return snap
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if allowNoAuth {
			snap.Status = "ok"
			snap.Error = ""
			snap.PlanType = "public"
			snap.LeftPercent = 100
			return snap
		}
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		snap.Error = msg
		return snap
	}

	metrics := autoQuotaMetricsFromHeaders(resp.Header)
	if len(metrics) == 0 {
		candidates := quotaProbeModelCandidates(preset, body)
		if len(candidates) > 0 {
			probeResp, probeErr := sendTinyChatProbe(ctx, baseURL, token, candidates[0])
			if probeErr == nil {
				_, _ = io.ReadAll(io.LimitReader(probeResp.Body, 8*1024))
				metrics = autoQuotaMetricsFromHeaders(probeResp.Header)
				probeResp.Body.Close()
			}
		}
	}
	if len(metrics) == 0 {
		if allowNoAuth {
			snap.Status = "ok"
			snap.Error = ""
			snap.PlanType = "public"
			snap.LeftPercent = 100
			return snap
		}
		snap.Error = "quota headers unavailable"
		return snap
	}
	best := metrics[0]
	for _, m := range metrics {
		if strings.Contains(strings.ToLower(m.MeteredFeature), "request") {
			best = m
			break
		}
	}
	snap.Status = "ok"
	snap.Error = ""
	snap.Metrics = metrics
	snap.LeftPercent = best.LeftPercent
	snap.ResetAt = best.ResetAt
	snap.PlanType = "auto"
	return snap
}

func (h *AdminHandler) readOpenRouterQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	token := providerTokenPreferAPIKey(p)
	if token == "" {
		snap.Error = "missing api key"
		return snap
	}
	preset, _ := presetForProvider(p)
	baseURL := providerBaseURLOrPreset(p, preset)
	if baseURL == "" {
		snap.Error = "base_url is required"
		return snap
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		snap.Error = "invalid base_url"
		return snap
	}
	u.Path = joinProviderPath(u.Path, "/key")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		snap.Error = "failed to build request"
		return snap
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		snap.Error = msg
		return snap
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		snap.Error = "invalid quota payload"
		return snap
	}
	data := asMap(payload["data"])
	if len(data) == 0 {
		data = payload
	}

	now := time.Now().UTC()
	metrics := make([]ProviderQuotaMetric, 0, 2)
	planType := "paid"
	if b, ok := data["is_free_tier"].(bool); ok && b {
		planType = "free"
	}

	limit, hasLimit := asFloat(firstMapValue(data, "limit"))
	remaining, hasRemaining := asFloat(firstMapValue(data, "limit_remaining", "remaining"))
	if !hasRemaining && hasLimit {
		if usage, ok := asFloat(firstMapValue(data, "usage")); ok {
			remaining = limit - usage
			hasRemaining = true
		}
	}
	resetAt := parseRateLimitResetAt(now, asString(firstMapValue(data, "limit_reset", "reset_at", "resetAt")))
	if resetAt == "" {
		resetAt = parseOpenRouterResetAt(now, asString(firstMapValue(data, "limit_reset", "reset_at", "resetAt")))
	}
	if hasLimit && limit > 0 && hasRemaining {
		if remaining < 0 {
			remaining = 0
		}
		if remaining > limit {
			remaining = limit
		}
		used := limit - remaining
		if used < 0 {
			used = 0
		}
		leftPercent := (remaining / limit) * 100
		metrics = append(metrics, ProviderQuotaMetric{
			Key:            "openrouter:spend",
			MeteredFeature: "credits",
			Window:         "quota",
			LeftPercent:    leftPercent,
			UsedValue:      used,
			RemainingValue: remaining,
			LimitValue:     limit,
			Unit:           "usd",
			ResetAt:        resetAt,
		})
		snap.LeftPercent = leftPercent
		snap.ResetAt = resetAt
	} else {
		// OpenRouter allows unbounded keys; avoid presenting this as an error.
		snap.LeftPercent = 100
		snap.ResetAt = resetAt
	}

	if rate := asMap(data["rate_limit"]); len(rate) > 0 {
		reqLimit, hasReqLimit := asFloat(firstMapValue(rate, "requests", "limit"))
		if hasReqLimit && reqLimit > 0 {
			interval := asString(firstMapValue(rate, "interval", "window"))
			var windowSeconds int64
			if d, ok := parseDurationLike(interval); ok && d > 0 {
				windowSeconds = int64(d / time.Second)
			}
			window := normalizeQuotaWindowLabel(interval, windowSeconds)
			metrics = append(metrics, ProviderQuotaMetric{
				Key:            "openrouter:requests-rate",
				MeteredFeature: "requests",
				Window:         window,
				WindowSeconds:  windowSeconds,
				LeftPercent:    100,
				UsedValue:      0,
				RemainingValue: reqLimit,
				LimitValue:     reqLimit,
				Unit:           "requests",
			})
		}
	}

	snap.Status = "ok"
	snap.Error = ""
	snap.Metrics = metrics
	snap.PlanType = planType
	return snap
}

func (h *AdminHandler) readNVIDIAUnknownQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	_ = ctx
	token := providerTokenPreferAPIKey(p)
	if token == "" {
		snap.Error = "missing api key"
		return snap
	}
	// NVIDIA requires an API key, but available quota/reset values are not reliably exposed.
	snap.Status = "ok"
	snap.Error = ""
	snap.PlanType = "unknown"
	snap.LeftPercent = 100
	snap.ResetAt = ""
	snap.Metrics = nil
	return snap
}

func parseOpenRouterResetAt(now time.Time, raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	base := now.UTC()
	switch s {
	case "hour", "hourly":
		return base.Truncate(time.Hour).Add(time.Hour).Format(time.RFC3339)
	case "day", "daily":
		start := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.UTC)
		return start.Add(24 * time.Hour).Format(time.RFC3339)
	case "week", "weekly":
		// Normalize to next Monday 00:00 UTC.
		start := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.UTC)
		weekday := int(start.Weekday())
		// Go weekday: Sunday=0. Convert to ISO weekday with Monday=1..Sunday=7.
		isoWeekday := weekday
		if isoWeekday == 0 {
			isoWeekday = 7
		}
		daysUntilMonday := 8 - isoWeekday
		return start.Add(time.Duration(daysUntilMonday) * 24 * time.Hour).Format(time.RFC3339)
	case "month", "monthly":
		start := time.Date(base.Year(), base.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start.AddDate(0, 1, 0).Format(time.RFC3339)
	case "year", "yearly", "annual", "annually":
		start := time.Date(base.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return start.AddDate(1, 0, 0).Format(time.RFC3339)
	default:
		return ""
	}
}

func (h *AdminHandler) readProviderQuotaCached(ctx context.Context, p config.ProviderConfig, preset assets.PopularProvider, forceRefresh bool) ProviderQuotaSnapshot {
	if forceRefresh {
		return h.computeProviderQuotaAndStore(p, preset)
	}
	_ = ctx
	now := time.Now().UTC()
	reader := quotaReaderFromPreset(preset)
	const refreshStaleGrace = 45 * time.Second
	h.quotaMu.Lock()
	if val, expiry, ok := h.quotaCache.Get(p.Name); ok {
		// Recover from stuck refresh workers: if entry is expired and still marked
		// refreshing long after expiry, allow a fresh background refresh.
		if val.Refreshing && !expiry.IsZero() && !now.Before(expiry.Add(refreshStaleGrace)) {
			val.Refreshing = false
			h.setQuotaCacheWithExpiry(p.Name, val, expiry)
		}
		snap := val.Snapshot
		if snap.Provider == "" {
			snap = newProviderQuotaSnapshot(now, p, preset, reader)
		}
		if strings.ToLower(strings.TrimSpace(snap.Status)) != "ok" && val.LastGood.Provider != "" {
			snap = val.LastGood
		}
		if !val.Refreshing && !expiry.IsZero() && !now.Before(expiry) {
			val.Refreshing = true
			h.setQuotaCacheWithExpiry(p.Name, val, expiry)
			go h.refreshProviderQuota(p, preset)
		}
		h.quotaMu.Unlock()
		return snap
	}
	snap := newProviderQuotaSnapshot(now, p, preset, reader)
	snap.Status = "loading"
	snap.Error = ""
	h.setQuotaCacheWithTTL(p.Name, quotaCacheValue{
		Snapshot:   snap,
		LastGood:   ProviderQuotaSnapshot{},
		Refreshing: true,
	}, now, quotaRefreshError)
	h.quotaMu.Unlock()
	go h.refreshProviderQuota(p, preset)
	return snap
}

func newProviderQuotaSnapshot(now time.Time, p config.ProviderConfig, preset assets.PopularProvider, reader string) ProviderQuotaSnapshot {
	snap := ProviderQuotaSnapshot{
		Provider:     p.Name,
		ProviderType: providerTypeOrName(p),
		DisplayName:  p.Name,
		Reader:       reader,
		Status:       "error",
		CheckedAt:    now.Format(time.RFC3339),
	}
	if snap.ProviderType == p.Name && strings.TrimSpace(preset.DisplayName) != "" {
		snap.DisplayName = strings.TrimSpace(preset.DisplayName)
	} else if strings.TrimSpace(preset.DisplayName) != "" {
		snap.DisplayName = p.Name + " (" + strings.TrimSpace(preset.DisplayName) + ")"
	}
	return snap
}

func (h *AdminHandler) refreshProviderQuota(p config.ProviderConfig, preset assets.PopularProvider) {
	defer func() {
		if rec := recover(); rec != nil {
			now := time.Now().UTC()
			reader := quotaReaderFromPreset(preset)
			snap := newProviderQuotaSnapshot(now, p, preset, reader)
			snap.Status = "error"
			snap.Error = "quota refresh panic"
			h.quotaMu.Lock()
			lastGood := ProviderQuotaSnapshot{}
			if prev, _, ok := h.quotaCache.Get(p.Name); ok {
				lastGood = prev.LastGood
			}
			storeSnap := snap
			if lastGood.Provider != "" {
				storeSnap = lastGood
			}
			h.setQuotaCacheWithTTL(p.Name, quotaCacheValue{
				Snapshot:   storeSnap,
				LastGood:   lastGood,
				Refreshing: false,
			}, now, quotaRefreshError)
			h.quotaMu.Unlock()
			slog.Warn("quota refresh panic recovered", "provider", strings.TrimSpace(p.Name))
		}
	}()
	h.computeProviderQuotaAndStore(p, preset)
}

func (h *AdminHandler) computeProviderQuotaAndStore(p config.ProviderConfig, preset assets.PopularProvider) ProviderQuotaSnapshot {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	p = h.ensureProviderOAuthTokenFresh(ctx, p, preset)
	now := time.Now().UTC()
	reader := quotaReaderFromPreset(preset)
	shouldLogRefresh := !isInlineQuotaHeaderReader(reader)
	if shouldLogRefresh {
		slog.Info("refreshing provider quota", "provider", strings.TrimSpace(p.Name), "reader", reader)
	}
	snap := newProviderQuotaSnapshot(now, p, preset, reader)
	snap = h.runQuotaReader(ctx, p, preset, snap)

	nextDelay := quotaRefreshError
	storeSnap := snap
	lastGood := ProviderQuotaSnapshot{}
	if prev, _, ok := h.quotaCache.Get(p.Name); ok {
		lastGood = prev.LastGood
		if lastGood.Provider == "" && strings.EqualFold(strings.TrimSpace(prev.Snapshot.Status), "ok") {
			lastGood = prev.Snapshot
		}
	}
	if snap.Status != "ok" {
		if lastGood.Provider != "" {
			// Preserve last known-good quota snapshot and retry in background.
			storeSnap = lastGood
		}
	} else {
		lastGood = snap
	}
	if snap.Status == "ok" {
		nextDelay = quotaRefreshOK
	}
	h.quotaMu.Lock()
	h.setQuotaCacheWithTTL(p.Name, quotaCacheValue{
		Snapshot:   storeSnap,
		LastGood:   lastGood,
		Refreshing: false,
	}, now, nextDelay)
	h.quotaMu.Unlock()
	if shouldLogRefresh {
		slog.Info(
			"provider quota refresh result",
			"provider", strings.TrimSpace(p.Name),
			"reader", reader,
			"status", strings.TrimSpace(snap.Status),
			"plan", strings.TrimSpace(snap.PlanType),
			"metrics", len(snap.Metrics),
			"left_percent", snap.LeftPercent,
			"reset_at", strings.TrimSpace(snap.ResetAt),
			"error", strings.TrimSpace(snap.Error),
		)
	}
	return storeSnap
}

func isInlineQuotaHeaderReader(reader string) bool {
	switch strings.TrimSpace(strings.ToLower(reader)) {
	case "header_auto", "groq_headers", "mistral_headers", "huggingface_headers", "cerebras_headers":
		return true
	default:
		return false
	}
}

func (h *AdminHandler) RecordQuotaFromResponse(p config.ProviderConfig, header http.Header) {
	if h == nil || len(header) == 0 {
		return
	}
	byName, err := popularProvidersByName()
	if err != nil {
		return
	}
	preset, ok := byName[providerTypeOrName(p)]
	if !ok {
		preset = assets.PopularProvider{
			ProviderConfig: config.ProviderConfig{Name: providerTypeOrName(p)},
			DisplayName:    p.Name,
		}
	}
	reader := quotaReaderFromPreset(preset)
	now := time.Now().UTC()
	snap := newProviderQuotaSnapshot(now, p, preset, reader)
	var metrics []ProviderQuotaMetric
	switch reader {
	case "groq_headers":
		if m, ok := groqMetricFromHeaders(header, "requests", "requests/day"); ok {
			metrics = append(metrics, m)
		}
		if m, ok := groqMetricFromHeaders(header, "tokens", "tokens/min"); ok {
			metrics = append(metrics, m)
		}
		snap.PlanType = "groq"
	case "mistral_headers":
		metrics = mistralMetricsFromHeaders(header)
		snap.PlanType = "mistral"
	case "huggingface_headers":
		metrics = huggingFaceMetricsFromHeaders(header)
		snap.PlanType = "huggingface"
	case "cerebras_headers":
		metrics = cerebrasMetricsFromHeaders(header)
		snap.PlanType = "cerebras"
	default:
		metrics = autoQuotaMetricsFromHeaders(header)
		snap.PlanType = "auto"
		snap.Reader = "header_auto"
	}
	if len(metrics) == 0 {
		metrics = autoQuotaMetricsFromHeaders(header)
		if len(metrics) > 0 {
			if strings.TrimSpace(snap.PlanType) == "" {
				snap.PlanType = "auto"
			}
			if strings.TrimSpace(snap.Reader) == "" {
				snap.Reader = "header_auto"
			}
		}
	}
	if len(metrics) == 0 {
		return
	}
	best := metrics[0]
	for _, m := range metrics {
		if strings.Contains(strings.ToLower(m.MeteredFeature), "request") {
			best = m
			break
		}
	}
	snap.Status = "ok"
	snap.Error = ""
	snap.Metrics = metrics
	snap.LeftPercent = best.LeftPercent
	snap.ResetAt = best.ResetAt
	h.quotaMu.Lock()
	h.setQuotaCacheWithTTL(p.Name, quotaCacheValue{
		Snapshot:   snap,
		LastGood:   snap,
		Refreshing: false,
	}, now, quotaRefreshOK)
	h.quotaMu.Unlock()
}

func autoQuotaMetricsFromHeaders(h http.Header) []ProviderQuotaMetric {
	return rateLimitMetricsFromHeaders(h, "auto", autoQuotaFeatureWindowFromSuffix)
}

func autoQuotaFeatureWindowFromSuffix(suffix string) (string, string) {
	s := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(suffix, "_", "-")))
	feature := "requests"
	if strings.Contains(s, "token") {
		feature = "tokens"
	}
	switch {
	case strings.Contains(s, "minute"), strings.HasSuffix(s, "-min"), strings.HasSuffix(s, "-m"):
		return feature, "1m"
	case strings.Contains(s, "hour"), strings.HasSuffix(s, "-h"):
		return feature, "1h"
	case strings.Contains(s, "day"), strings.HasSuffix(s, "-d"):
		return feature, "1d"
	case strings.Contains(s, "week"), strings.HasSuffix(s, "-w"):
		return feature, "7d"
	case strings.Contains(s, "month"):
		return feature, "30d"
	case s == "requests" || s == "tokens":
		return feature, "quota"
	default:
		return feature, strings.ReplaceAll(s, "-", "/")
	}
}

func (h *AdminHandler) ensureProviderOAuthTokenFresh(ctx context.Context, p config.ProviderConfig, preset assets.PopularProvider) config.ProviderConfig {
	if strings.TrimSpace(p.RefreshToken) == "" {
		return p
	}
	expiresAt := strings.TrimSpace(p.TokenExpiresAt)
	token := strings.TrimSpace(p.AuthToken)
	if expiresAt != "" && token != "" {
		if ts, err := time.Parse(time.RFC3339, expiresAt); err == nil && time.Until(ts) > 60*time.Second {
			return p
		}
	}

	tokenURL := strings.TrimSpace(preset.OAuthTokenURL)
	clientID := strings.TrimSpace(preset.OAuthClientID)
	clientSecret := strings.TrimSpace(preset.OAuthClientSecret)
	if tokenURL == "" || clientID == "" {
		return p
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(p.RefreshToken))
	form.Set("client_id", clientID)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return p
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return p
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return p
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return p
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return p
	}
	p.AuthToken = strings.TrimSpace(out.AccessToken)
	if strings.TrimSpace(out.RefreshToken) != "" {
		p.RefreshToken = strings.TrimSpace(out.RefreshToken)
	}
	if out.ExpiresIn > 0 {
		p.TokenExpiresAt = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
	}
	if h.store != nil {
		_ = h.store.Update(func(c *config.ServerConfig) error {
			for i := range c.Providers {
				if c.Providers[i].Name != p.Name {
					continue
				}
				c.Providers[i].AuthToken = p.AuthToken
				c.Providers[i].RefreshToken = p.RefreshToken
				c.Providers[i].TokenExpiresAt = p.TokenExpiresAt
				break
			}
			return nil
		})
	}
	return p
}

func (h *AdminHandler) readOpenAICodexQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	token, tokenFromAuth := providerTokenPreferAuth(p)
	if token == "" {
		snap.Error = "missing auth token"
		return snap
	}
	preset, _ := presetForProvider(p)
	baseURL := providerBaseURLOrPreset(p, preset)
	oauthBaseURL := strings.TrimRight(strings.TrimSpace(preset.OAuthBaseURL), "/")
	if tokenFromAuth && (baseURL == "" || strings.Contains(strings.ToLower(baseURL), "api.openai.com")) && oauthBaseURL != "" {
		baseURL = oauthBaseURL
	}
	if baseURL == "" && oauthBaseURL != "" {
		baseURL = oauthBaseURL
	}
	if baseURL == "" {
		baseURL = "https://chatgpt.com/backend-api"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		snap.Error = "invalid base_url"
		return snap
	}
	u.Path = joinProviderPath(u.Path, "/wham/usage")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		snap.Error = "failed to build request"
		return snap
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(p.AccountID) != "" {
		req.Header.Set("ChatGPT-Account-Id", strings.TrimSpace(p.AccountID))
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		snap.Error = msg
		return snap
	}
	planType, metrics, err := parseOpenAICodexQuotaMetrics(body)
	if err != nil {
		snap.Error = "invalid quota payload"
		return snap
	}
	if len(metrics) == 0 {
		snap.Error = "quota payload missing rate limits"
		return snap
	}
	best := pickPreferredQuotaMetric(metrics)
	snap.Status = "ok"
	snap.Error = ""
	snap.PlanType = strings.TrimSpace(planType)
	snap.LeftPercent = best.LeftPercent
	snap.ResetAt = best.ResetAt
	snap.Metrics = metrics
	return snap
}

func (h *AdminHandler) readGoogleAntigravityQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	token := strings.TrimSpace(p.AuthToken)
	if token == "" {
		snap.Error = "missing auth token"
		return snap
	}

	projectID := strings.TrimSpace(p.AccountID)
	endpoints := antigravityEndpointsForProvider(p)
	var lastErr string
	var retrieveErr string
	for _, endpoint := range endpoints {
		u, err := url.Parse(strings.TrimRight(endpoint, "/"))
		if err != nil {
			lastErr = "invalid antigravity endpoint"
			continue
		}
		if projectID != "" {
			planType, metrics, rerr := fetchGoogleRetrieveUserQuota(ctx, u, token, projectID)
			if rerr == nil && len(metrics) > 0 {
				best := pickPreferredQuotaMetric(metrics)
				snap.Status = "ok"
				snap.Error = ""
				snap.PlanType = planType
				snap.Metrics = metrics
				snap.LeftPercent = best.LeftPercent
				snap.ResetAt = best.ResetAt
				return snap
			}
			if rerr != nil {
				lastErr = rerr.Error()
				retrieveErr = rerr.Error()
			}
		}

		planType, loadMetrics, loadProject, loadErr := fetchGoogleLoadCodeAssistQuota(ctx, u, token)
		if loadErr != nil {
			lastErr = loadErr.Error()
			continue
		}

		if projectID == "" && loadProject != "" {
			projectID = loadProject
			planType2, metrics2, rerr2 := fetchGoogleRetrieveUserQuota(ctx, u, token, projectID)
			if rerr2 == nil && len(metrics2) > 0 {
				best := pickPreferredQuotaMetric(metrics2)
				snap.Status = "ok"
				snap.Error = ""
				if strings.TrimSpace(planType2) != "" {
					snap.PlanType = planType2
				} else {
					snap.PlanType = planType
				}
				snap.Metrics = metrics2
				snap.LeftPercent = best.LeftPercent
				snap.ResetAt = best.ResetAt
				return snap
			}
			if rerr2 != nil {
				lastErr = rerr2.Error()
				retrieveErr = rerr2.Error()
			}
		}

		// Fallback to any quota-like fields present in loadCodeAssist payload.
		if len(loadMetrics) > 0 {
			best := pickPreferredQuotaMetric(loadMetrics)
			snap.Status = "ok"
			snap.Error = ""
			snap.PlanType = planType
			snap.Metrics = loadMetrics
			snap.LeftPercent = best.LeftPercent
			snap.ResetAt = best.ResetAt
			return snap
		}
		if retrieveErr != "" {
			snap.Error = retrieveErr
		} else {
			snap.Error = "quota fields unavailable in antigravity response"
		}
		return snap
	}

	if lastErr == "" {
		lastErr = "antigravity quota request failed"
	}
	snap.Error = lastErr
	return snap
}

func fetchGoogleRetrieveUserQuota(ctx context.Context, baseURL *url.URL, token string, projectID string) (string, []ProviderQuotaMetric, error) {
	u := *baseURL
	u.Path = joinProviderPath(u.Path, "/v1internal:retrieveUserQuota")
	reqBody := map[string]any{
		"project": strings.TrimSpace(projectID),
	}
	rawBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(rawBody))
	if err != nil {
		return "", nil, fmt.Errorf("failed to build retrieveUserQuota request")
	}
	setGoogleAntigravityHeaders(req, token)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return "", nil, errors.New(msg)
	}
	planType, metrics, err := parseGoogleRetrieveUserQuota(body)
	if err != nil {
		return "", nil, err
	}
	return planType, metrics, nil
}

func fetchGoogleLoadCodeAssistQuota(ctx context.Context, baseURL *url.URL, token string) (string, []ProviderQuotaMetric, string, error) {
	u := *baseURL
	u.Path = joinProviderPath(u.Path, "/v1internal:loadCodeAssist")
	reqBody := map[string]any{
		"metadata": map[string]any{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}
	rawBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(rawBody))
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to build antigravity request")
	}
	setGoogleAntigravityHeaders(req, token)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", nil, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return "", nil, "", errors.New(msg)
	}
	planType, metrics, parseErr := parseGoogleAntigravityQuota(body)
	if parseErr != nil {
		return "", nil, "", parseErr
	}
	project := extractAntigravityProjectID(body)
	return planType, metrics, project, nil
}

func setGoogleAntigravityHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Client", "google-cloud-sdk vscode_cloudshelleditor/0.1")
	req.Header.Set("Client-Metadata", `{"ideType":"IDE_UNSPECIFIED","platform":"PLATFORM_UNSPECIFIED","pluginType":"GEMINI"}`)
}

func parseGoogleRetrieveUserQuota(body []byte) (string, []ProviderQuotaMetric, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, err
	}
	planType := inferAntigravityPlanType(payload)
	buckets := asSlice(payload["buckets"])
	metrics := make([]ProviderQuotaMetric, 0, len(buckets))
	for _, b := range buckets {
		m := asMap(b)
		if len(m) == 0 {
			continue
		}
		fraction, ok := asFloat(m["remainingFraction"])
		if !ok {
			continue
		}
		if fraction < 0 {
			fraction = 0
		}
		if fraction > 1 {
			fraction = 1
		}
		leftPercent := fraction * 100
		modelID := strings.TrimSpace(asString(m["modelId"]))
		tokenType := strings.TrimSpace(strings.ToLower(asString(m["tokenType"])))
		if modelID == "" {
			modelID = "gemini"
		}
		window := "quota"
		if tokenType != "" {
			window = strings.ReplaceAll(tokenType, "_", "/")
		}
		resetAt := ""
		if ts := strings.TrimSpace(asString(m["resetTime"])); ts != "" {
			if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
				resetAt = parsed.UTC().Format(time.RFC3339)
			}
		}
		key := strings.ToLower(modelID + ":" + window)
		metrics = append(metrics, ProviderQuotaMetric{
			Key:            key,
			MeteredFeature: modelID,
			Window:         window,
			LeftPercent:    leftPercent,
			UsedValue:      100 - leftPercent,
			RemainingValue: leftPercent,
			LimitValue:     100,
			Unit:           "percent",
			ResetAt:        resetAt,
		})
	}
	return planType, metrics, nil
}

func extractAntigravityProjectID(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	project := strings.TrimSpace(asString(firstMapValue(
		payload,
		"cloudaicompanionProject",
		"cloudaiCompanionProject",
		"duetProject",
		"project",
	)))
	return project
}

func (h *AdminHandler) readGroqQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	token := providerTokenPreferAPIKey(p)
	if token == "" {
		snap.Error = "missing api key"
		return snap
	}
	preset, _ := presetForProvider(p)
	baseURL := providerBaseURLOrPreset(p, preset)
	if baseURL == "" {
		snap.Error = "base_url is required"
		return snap
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		snap.Error = "invalid base_url"
		return snap
	}
	u.Path = joinProviderPath(u.Path, "/v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		snap.Error = "failed to build request"
		return snap
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		snap.Error = msg
		return snap
	}

	metrics := make([]ProviderQuotaMetric, 0, 4)
	if m, ok := groqMetricFromHeaders(resp.Header, "requests", "requests/day"); ok {
		metrics = append(metrics, m)
	}
	if m, ok := groqMetricFromHeaders(resp.Header, "tokens", "tokens/min"); ok {
		metrics = append(metrics, m)
	}
	if len(metrics) == 0 {
		chatMetrics, chatErr := h.readGroqQuotaFromTinyChat(ctx, p, preset, baseURL, token, body)
		if chatErr == nil && len(chatMetrics) > 0 {
			metrics = chatMetrics
		}
	}
	if len(metrics) == 0 {
		snap.Error = "quota headers unavailable"
		return snap
	}
	best := metrics[0]
	if len(metrics) > 1 && strings.Contains(strings.ToLower(metrics[1].MeteredFeature), "requests") {
		best = metrics[1]
	}
	snap.Status = "ok"
	snap.Error = ""
	snap.Metrics = metrics
	snap.LeftPercent = best.LeftPercent
	snap.ResetAt = best.ResetAt
	snap.PlanType = "groq"
	return snap
}

func (h *AdminHandler) readMistralQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	token := providerTokenPreferAPIKey(p)
	if token == "" {
		snap.Error = "missing api key"
		return snap
	}
	preset, _ := presetForProvider(p)
	baseURL := providerBaseURLOrPreset(p, preset)
	if baseURL == "" {
		snap.Error = "base_url is required"
		return snap
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		snap.Error = "invalid base_url"
		return snap
	}
	u.Path = joinProviderPath(u.Path, "/v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		snap.Error = "failed to build request"
		return snap
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		snap.Error = msg
		return snap
	}
	metrics := mistralMetricsFromHeaders(resp.Header)
	if len(metrics) == 0 {
		chatMetrics, chatErr := h.readMistralQuotaFromTinyChat(ctx, p, preset, baseURL, token, body)
		if chatErr == nil && len(chatMetrics) > 0 {
			metrics = chatMetrics
		}
	}
	if len(metrics) == 0 {
		snap.Error = "quota headers unavailable"
		return snap
	}
	best := metrics[0]
	for _, m := range metrics {
		if strings.Contains(strings.ToLower(m.MeteredFeature), "request") {
			best = m
			break
		}
	}
	snap.Status = "ok"
	snap.Error = ""
	snap.Metrics = metrics
	snap.LeftPercent = best.LeftPercent
	snap.ResetAt = best.ResetAt
	snap.PlanType = "mistral"
	return snap
}

func (h *AdminHandler) readHuggingFaceQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	token := providerTokenPreferAPIKey(p)
	if token == "" {
		snap.Error = "missing api key"
		return snap
	}
	preset, _ := presetForProvider(p)
	baseURL := providerBaseURLOrPreset(p, preset)
	if baseURL == "" {
		snap.Error = "base_url is required"
		return snap
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		snap.Error = "invalid base_url"
		return snap
	}
	u.Path = joinProviderPath(u.Path, "/v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		snap.Error = "failed to build request"
		return snap
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		snap.Error = msg
		return snap
	}

	metrics := huggingFaceMetricsFromHeaders(resp.Header)
	if len(metrics) == 0 {
		chatMetrics, chatErr := h.readHuggingFaceQuotaFromTinyChat(ctx, p, preset, baseURL, token, body)
		if chatErr == nil && len(chatMetrics) > 0 {
			metrics = chatMetrics
		}
	}
	if len(metrics) == 0 {
		snap.Error = "quota headers unavailable"
		return snap
	}
	best := metrics[0]
	for _, m := range metrics {
		if strings.Contains(strings.ToLower(m.MeteredFeature), "request") {
			best = m
			break
		}
	}
	snap.Status = "ok"
	snap.Error = ""
	snap.Metrics = metrics
	snap.LeftPercent = best.LeftPercent
	snap.ResetAt = best.ResetAt
	snap.PlanType = "huggingface"
	return snap
}

func (h *AdminHandler) readCerebrasQuota(ctx context.Context, p config.ProviderConfig, snap ProviderQuotaSnapshot) ProviderQuotaSnapshot {
	token := providerTokenPreferAPIKey(p)
	if token == "" {
		snap.Error = "missing api key"
		return snap
	}
	preset, _ := presetForProvider(p)
	baseURL := providerBaseURLOrPreset(p, preset)
	if baseURL == "" {
		snap.Error = "base_url is required"
		return snap
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		snap.Error = "invalid base_url"
		return snap
	}
	u.Path = joinProviderPath(u.Path, "/v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		snap.Error = "failed to build request"
		return snap
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		snap.Error = msg
		return snap
	}

	metrics := cerebrasMetricsFromHeaders(resp.Header)
	if len(metrics) == 0 {
		chatMetrics, chatErr := h.readCerebrasQuotaFromTinyChat(ctx, p, preset, baseURL, token, body)
		if chatErr == nil && len(chatMetrics) > 0 {
			metrics = chatMetrics
		}
	}
	if len(metrics) == 0 {
		snap.Error = "quota headers unavailable"
		return snap
	}
	best := metrics[0]
	for _, m := range metrics {
		if strings.Contains(strings.ToLower(m.MeteredFeature), "request") {
			best = m
			break
		}
	}
	snap.Status = "ok"
	snap.Error = ""
	snap.Metrics = metrics
	snap.LeftPercent = best.LeftPercent
	snap.ResetAt = best.ResetAt
	snap.PlanType = "cerebras"
	return snap
}

func (h *AdminHandler) readCerebrasQuotaFromTinyChat(ctx context.Context, p config.ProviderConfig, preset assets.PopularProvider, baseURL string, token string, modelsBody []byte) ([]ProviderQuotaMetric, error) {
	candidates := quotaProbeModelCandidates(preset, modelsBody)
	if len(candidates) == 0 {
		return nil, nil
	}
	modelID := candidates[0]
	resp, err := sendTinyChatProbe(ctx, baseURL, token, modelID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	return cerebrasMetricsFromHeaders(resp.Header), nil
}

func (h *AdminHandler) readMistralQuotaFromTinyChat(ctx context.Context, p config.ProviderConfig, preset assets.PopularProvider, baseURL string, token string, modelsBody []byte) ([]ProviderQuotaMetric, error) {
	candidates := quotaProbeModelCandidates(preset, modelsBody)
	if len(candidates) == 0 {
		return nil, nil
	}
	modelID := candidates[0]
	resp, err := sendTinyChatProbe(ctx, baseURL, token, modelID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	return mistralMetricsFromHeaders(resp.Header), nil
}

func (h *AdminHandler) readGroqQuotaFromTinyChat(ctx context.Context, p config.ProviderConfig, preset assets.PopularProvider, baseURL string, token string, modelsBody []byte) ([]ProviderQuotaMetric, error) {
	candidates := quotaProbeModelCandidates(preset, modelsBody)
	if len(candidates) == 0 {
		return nil, nil
	}
	var lastErr error
	for _, modelID := range candidates {
		resp, err := sendTinyChatProbe(ctx, baseURL, token, modelID)
		if err != nil {
			lastErr = err
			continue
		}
		_, _ = io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		metrics := make([]ProviderQuotaMetric, 0, 4)
		if m, ok := groqMetricFromHeaders(resp.Header, "requests", "requests/day"); ok {
			metrics = append(metrics, m)
		}
		if m, ok := groqMetricFromHeaders(resp.Header, "tokens", "tokens/min"); ok {
			metrics = append(metrics, m)
		}
		resp.Body.Close()
		if len(metrics) > 0 {
			return metrics, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

func (h *AdminHandler) readHuggingFaceQuotaFromTinyChat(ctx context.Context, p config.ProviderConfig, preset assets.PopularProvider, baseURL string, token string, modelsBody []byte) ([]ProviderQuotaMetric, error) {
	candidates := quotaProbeModelCandidates(preset, modelsBody)
	if len(candidates) > 12 {
		candidates = candidates[:12]
	}
	var lastErr error
	for _, modelID := range candidates {
		resp, err := sendTinyChatProbe(ctx, baseURL, token, modelID)
		if err != nil {
			lastErr = err
			continue
		}
		_, _ = io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		metrics := huggingFaceMetricsFromHeaders(resp.Header)
		resp.Body.Close()
		if len(metrics) > 0 {
			return metrics, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

func sendTinyChatProbe(ctx context.Context, baseURL string, token string, modelID string) (*http.Response, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = joinProviderPath(u.Path, "/v1/chat/completions")
	payload := map[string]any{
		"model":       strings.TrimSpace(modelID),
		"messages":    []map[string]string{{"role": "user", "content": "hi"}},
		"max_tokens":  1,
		"temperature": 0,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		_, _ = io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		return nil, fmt.Errorf("chat probe status %d", resp.StatusCode)
	}
	return resp, nil
}

func providerTokenPreferAPIKey(p config.ProviderConfig) string {
	if v := strings.TrimSpace(p.APIKey); v != "" {
		return v
	}
	return strings.TrimSpace(p.AuthToken)
}

func providerTokenPreferAuth(p config.ProviderConfig) (token string, fromAuth bool) {
	if v := strings.TrimSpace(p.AuthToken); v != "" {
		return v, true
	}
	if v := strings.TrimSpace(p.APIKey); v != "" {
		return v, false
	}
	return "", false
}

func groqMetricFromHeaders(h http.Header, feature string, _ string) (ProviderQuotaMetric, bool) {
	metrics := rateLimitMetricsFromHeaders(h, "groq", func(suffix string) (string, string) {
		s := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(suffix, "_", "-")))
		switch s {
		case "requests":
			return "requests", "requests/day"
		case "tokens":
			return "tokens", "tokens/min"
		default:
			return "", ""
		}
	})
	for _, m := range metrics {
		if strings.EqualFold(strings.TrimSpace(m.MeteredFeature), strings.TrimSpace(feature)) {
			return m, true
		}
	}
	return ProviderQuotaMetric{}, false
}

func mistralMetricsFromHeaders(h http.Header) []ProviderQuotaMetric {
	return rateLimitMetricsFromHeaders(h, "mistral", mistralFeatureWindowFromSuffix)
}

func huggingFaceMetricsFromHeaders(h http.Header) []ProviderQuotaMetric {
	metrics := rateLimitMetricsFromHeaders(h, "huggingface", mistralFeatureWindowFromSuffix)
	if len(metrics) > 0 {
		return metrics
	}
	return rateLimitStructuredMetricsFromHeaders(h, "huggingface")
}

func quotaMetricUnit(feature string) string {
	f := strings.ToLower(strings.TrimSpace(feature))
	if strings.Contains(f, "token") {
		return "tokens"
	}
	return "requests"
}

func rateLimitMetricsFromHeaders(h http.Header, keyPrefix string, classifySuffix func(string) (string, string)) []ProviderQuotaMetric {
	if classifySuffix == nil {
		return nil
	}
	type metricParts struct {
		limit     float64
		hasLimit  bool
		remaining float64
		hasRemain bool
		resetRaw  string
	}
	now := time.Now().UTC()
	parts := map[string]*metricParts{}
	globalReset := ""
	resetBySuffix := map[string]string{}
	for k, vals := range h {
		if len(vals) == 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(k))
		headerPrefix := ""
		if strings.HasPrefix(key, "x-ratelimit-") {
			headerPrefix = "x-ratelimit-"
		} else if strings.HasPrefix(key, "ratelimit-") {
			headerPrefix = "ratelimit-"
		}
		if headerPrefix == "" {
			continue
		}
		value := strings.TrimSpace(vals[0])
		switch {
		case key == headerPrefix+"limit":
			limit, err := strconv.ParseFloat(value, 64)
			if err != nil || limit <= 0 {
				continue
			}
			const suffix = "requests"
			mp := parts[suffix]
			if mp == nil {
				mp = &metricParts{}
				parts[suffix] = mp
			}
			mp.limit = limit
			mp.hasLimit = true
		case key == headerPrefix+"remaining":
			remain, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			const suffix = "requests"
			mp := parts[suffix]
			if mp == nil {
				mp = &metricParts{}
				parts[suffix] = mp
			}
			mp.remaining = remain
			mp.hasRemain = true
		case key == headerPrefix+"reset":
			const suffix = "requests"
			if globalReset == "" {
				globalReset = value
			}
			resetBySuffix[suffix] = value
			mp := parts[suffix]
			if mp == nil {
				mp = &metricParts{}
				parts[suffix] = mp
			}
			mp.resetRaw = value
		case strings.HasPrefix(key, headerPrefix+"limit-"):
			suffix := strings.TrimPrefix(key, headerPrefix+"limit-")
			if suffix == "" {
				continue
			}
			limit, err := strconv.ParseFloat(value, 64)
			if err != nil || limit <= 0 {
				continue
			}
			mp := parts[suffix]
			if mp == nil {
				mp = &metricParts{}
				parts[suffix] = mp
			}
			mp.limit = limit
			mp.hasLimit = true
		case strings.HasPrefix(key, headerPrefix+"remaining-"):
			suffix := strings.TrimPrefix(key, headerPrefix+"remaining-")
			if suffix == "" {
				continue
			}
			remain, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			mp := parts[suffix]
			if mp == nil {
				mp = &metricParts{}
				parts[suffix] = mp
			}
			mp.remaining = remain
			mp.hasRemain = true
		case strings.HasPrefix(key, headerPrefix+"reset-"):
			suffix := strings.TrimPrefix(key, headerPrefix+"reset-")
			if suffix == "" {
				if globalReset == "" {
					globalReset = value
				}
				continue
			}
			resetBySuffix[suffix] = value
			mp := parts[suffix]
			if mp == nil {
				mp = &metricParts{}
				parts[suffix] = mp
			}
			mp.resetRaw = value
		}
	}

	metrics := make([]ProviderQuotaMetric, 0, len(parts))
	for suffix, mp := range parts {
		if mp == nil || !mp.hasLimit || !mp.hasRemain || mp.limit <= 0 {
			continue
		}
		remaining := mp.remaining
		if remaining < 0 {
			remaining = 0
		}
		if remaining > mp.limit {
			remaining = mp.limit
		}
		used := mp.limit - remaining
		if used < 0 {
			used = 0
		}
		leftPercent := (remaining / mp.limit) * 100
		feature, window := classifySuffix(suffix)
		if strings.TrimSpace(feature) == "" || strings.TrimSpace(window) == "" {
			continue
		}
		resetRaw := strings.TrimSpace(mp.resetRaw)
		if resetRaw == "" {
			featurePrefix := strings.Split(strings.ToLower(suffix), "-")[0]
			for rs, rv := range resetBySuffix {
				lrs := strings.ToLower(rs)
				if lrs == featurePrefix || strings.HasPrefix(lrs, featurePrefix+"-") {
					resetRaw = strings.TrimSpace(rv)
					break
				}
			}
		}
		if resetRaw == "" {
			resetRaw = strings.TrimSpace(globalReset)
		}
		resetAt := parseRateLimitResetAt(now, resetRaw)
		metrics = append(metrics, ProviderQuotaMetric{
			Key:            keyPrefix + ":" + suffix,
			MeteredFeature: feature,
			Window:         window,
			LeftPercent:    leftPercent,
			UsedValue:      used,
			RemainingValue: remaining,
			LimitValue:     mp.limit,
			Unit:           quotaMetricUnit(feature),
			ResetAt:        resetAt,
		})
	}
	sort.SliceStable(metrics, func(i, j int) bool {
		if metrics[i].MeteredFeature != metrics[j].MeteredFeature {
			return metrics[i].MeteredFeature < metrics[j].MeteredFeature
		}
		return metrics[i].Window < metrics[j].Window
	})
	return metrics
}

func cerebrasMetricsFromHeaders(h http.Header) []ProviderQuotaMetric {
	raw := rateLimitMetricsFromHeaders(h, "cerebras", mistralFeatureWindowFromSuffix)
	if len(raw) == 0 {
		return raw
	}
	// Temporary workaround: Cerebras 1h buckets consistently report depleted quota.
	filtered := make([]ProviderQuotaMetric, 0, len(raw))
	for _, m := range raw {
		window := strings.ToLower(strings.TrimSpace(m.Window))
		if m.WindowSeconds == 3600 ||
			window == "1h" ||
			window == "hour" ||
			strings.Contains(window, "/1h") ||
			strings.Contains(window, "/hour") {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

func rateLimitStructuredMetricsFromHeaders(h http.Header, keyPrefix string) []ProviderQuotaMetric {
	now := time.Now().UTC()
	rateLimitParts := map[string]map[string]string{}
	policyParts := map[string]map[string]string{}
	globalRateLimit := map[string]string{}
	globalPolicy := map[string]string{}

	for _, raw := range h.Values("RateLimit") {
		for _, entry := range splitCSVHeader(raw) {
			label, params := parseRateLimitEntry(entry)
			if len(params) == 0 {
				continue
			}
			if label == "" {
				for k, v := range params {
					globalRateLimit[k] = v
				}
				continue
			}
			rateLimitParts[label] = params
		}
	}
	for _, raw := range h.Values("RateLimit-Policy") {
		for _, entry := range splitCSVHeader(raw) {
			label, params := parseRateLimitEntry(entry)
			if len(params) == 0 {
				continue
			}
			if label == "" {
				for k, v := range params {
					globalPolicy[k] = v
				}
				continue
			}
			policyParts[label] = params
		}
	}

	labels := make([]string, 0, len(rateLimitParts)+len(policyParts))
	seen := map[string]struct{}{}
	for k := range rateLimitParts {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			labels = append(labels, k)
		}
	}
	for k := range policyParts {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			labels = append(labels, k)
		}
	}
	sort.Strings(labels)

	metrics := make([]ProviderQuotaMetric, 0, len(labels)+1)
	appendMetric := func(label string, rl map[string]string, pol map[string]string) {
		if rl == nil {
			rl = map[string]string{}
		}
		if pol == nil {
			pol = map[string]string{}
		}
		if len(rl) == 0 && len(pol) == 0 {
			return
		}
		q, okQ := asFloat(firstMapValue(castMapStringAny(pol), "q", "limit"))
		r, okR := asFloat(firstMapValue(castMapStringAny(rl), "r", "remaining"))
		if !okQ || !okR || q <= 0 {
			return
		}
		if r < 0 {
			r = 0
		}
		if r > q {
			r = q
		}
		used := q - r
		if used < 0 {
			used = 0
		}
		leftPercent := (r / q) * 100

		resetAt := ""
		if tRaw := strings.TrimSpace(asString(firstMapValue(castMapStringAny(rl), "t", "reset"))); tRaw != "" {
			resetAt = parseRateLimitResetAt(now, tRaw)
		}

		windowSeconds, _ := asInt64(firstMapValue(castMapStringAny(pol), "w", "window"))
		window := normalizeQuotaWindowLabel("", windowSeconds)
		if window == "" {
			window = "quota"
		}
		feature := strings.TrimSpace(label)
		if feature == "" {
			feature = "requests"
		}
		suffix := strings.ToLower(strings.ReplaceAll(feature, " ", "_"))
		metrics = append(metrics, ProviderQuotaMetric{
			Key:            keyPrefix + ":" + suffix + ":" + window,
			MeteredFeature: feature,
			Window:         window,
			WindowSeconds:  windowSeconds,
			LeftPercent:    leftPercent,
			UsedValue:      used,
			RemainingValue: r,
			LimitValue:     q,
			Unit:           quotaMetricUnit(feature),
			ResetAt:        resetAt,
		})
	}

	for _, label := range labels {
		appendMetric(label, rateLimitParts[label], policyParts[label])
	}
	if len(metrics) == 0 && (len(globalRateLimit) > 0 || len(globalPolicy) > 0) {
		appendMetric("requests", globalRateLimit, globalPolicy)
	}
	sort.SliceStable(metrics, func(i, j int) bool {
		if metrics[i].MeteredFeature != metrics[j].MeteredFeature {
			return metrics[i].MeteredFeature < metrics[j].MeteredFeature
		}
		return metrics[i].Window < metrics[j].Window
	})
	return metrics
}

func splitCSVHeader(raw string) []string {
	out := []string{}
	var b strings.Builder
	inQuote := false
	for _, r := range raw {
		switch r {
		case '"':
			inQuote = !inQuote
			b.WriteRune(r)
		case ',':
			if inQuote {
				b.WriteRune(r)
				continue
			}
			part := strings.TrimSpace(b.String())
			if part != "" {
				out = append(out, part)
			}
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	last := strings.TrimSpace(b.String())
	if last != "" {
		out = append(out, last)
	}
	return out
}

func parseRateLimitEntry(entry string) (string, map[string]string) {
	params := map[string]string{}
	parts := strings.Split(entry, ";")
	if len(parts) == 0 {
		return "", params
	}
	label := strings.Trim(strings.TrimSpace(parts[0]), `"`)
	start := 1
	if strings.Contains(parts[0], "=") {
		label = ""
		start = 0
	}
	for i := start; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		if key == "" || val == "" {
			continue
		}
		params[key] = val
	}
	return label, params
}

func castMapStringAny(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func parseRateLimitResetAt(now time.Time, raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if d, ok := parseDurationLike(s); ok {
		return now.Add(d).Format(time.RFC3339)
	}
	if parsed, err := time.Parse(time.RFC3339, s); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	if unix, ok := asInt64(s); ok && unix > 0 {
		// Most rate-limit headers expose relative seconds; Unix epoch values are much larger.
		if unix < 1_000_000_000 {
			return now.Add(time.Duration(unix) * time.Second).Format(time.RFC3339)
		}
		if unix > 1_000_000_000_000 {
			unix = unix / 1000
		}
		return time.Unix(unix, 0).UTC().Format(time.RFC3339)
	}
	// Some structured headers emit plain integer seconds as string.
	if d, ok := parseDurationLike(s + "s"); ok {
		return now.Add(d).Format(time.RFC3339)
	}
	return ""
}

func mistralFeatureWindowFromSuffix(suffix string) (string, string) {
	s := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(suffix, "_", "-")))
	feature := "requests"
	if strings.Contains(s, "token") {
		feature = "tokens"
	}
	switch {
	case strings.Contains(s, "minute"), strings.HasSuffix(s, "-min"), strings.HasSuffix(s, "-m"):
		return feature, "1m"
	case strings.Contains(s, "hour"), strings.HasSuffix(s, "-h"):
		return feature, "1h"
	case strings.Contains(s, "day"), strings.HasSuffix(s, "-d"):
		return feature, "1d"
	case strings.Contains(s, "week"), strings.HasSuffix(s, "-w"):
		return feature, "7d"
	case strings.Contains(s, "month"):
		return feature, "30d"
	default:
		return feature, strings.ReplaceAll(s, "-", "/")
	}
}

func modelIDsFromModelsBody(modelsBody []byte, fallbacks ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 16)
	var parsed modelListResponse
	if err := json.Unmarshal(modelsBody, &parsed); err == nil {
		for _, m := range parsed.Data {
			id := strings.TrimSpace(m.ID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	for _, fb := range fallbacks {
		id := strings.TrimSpace(fb)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func antigravityUserAgent() string {
	platform := "linux"
	switch runtime.GOOS {
	case "darwin":
		platform = "darwin"
	case "windows":
		platform = "windows"
	}
	arch := "amd64"
	switch runtime.GOARCH {
	case "arm64":
		arch = "arm64"
	case "386":
		arch = "386"
	}
	return "antigravity/1.15.8 " + platform + "/" + arch
}

func antigravityEndpointsForProvider(p config.ProviderConfig) []string {
	fallbacks := []string{
		"https://daily-cloudcode-pa.sandbox.googleapis.com",
		"https://autopush-cloudcode-pa.sandbox.googleapis.com",
		"https://cloudcode-pa.googleapis.com",
	}
	baseURL := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if baseURL == "" {
		return fallbacks
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return fallbacks
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if strings.Contains(host, "cloudcode-pa.") || strings.Contains(baseURL, "v1internal") {
		root := u.Scheme + "://" + u.Host
		out := []string{root}
		for _, ep := range fallbacks {
			if ep != root {
				out = append(out, ep)
			}
		}
		return out
	}
	return fallbacks
}

func parseGoogleAntigravityQuota(body []byte) (string, []ProviderQuotaMetric, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, err
	}
	planType := inferAntigravityPlanType(payload)
	metrics := collectAntigravityQuotaMetrics(payload)
	return planType, metrics, nil
}

func inferAntigravityPlanType(payload map[string]any) string {
	if pt := asString(payload["plan_type"]); pt != "" {
		return pt
	}
	if paid := asMap(payload["paidTier"]); len(paid) > 0 {
		if id := asString(paid["id"]); id != "" {
			return id
		}
	}
	if cur := asMap(payload["currentTier"]); len(cur) > 0 {
		if id := asString(cur["id"]); id != "" {
			return id
		}
	}
	for _, tAny := range asSlice(payload["allowedTiers"]) {
		t := asMap(tAny)
		if len(t) == 0 {
			continue
		}
		if asString(t["isDefault"]) == "true" {
			if id := asString(t["id"]); id != "" {
				return id
			}
		}
	}
	return ""
}

func collectAntigravityQuotaMetrics(payload map[string]any) []ProviderQuotaMetric {
	metrics := make([]ProviderQuotaMetric, 0, 8)
	seen := map[string]struct{}{}
	now := time.Now().UTC()
	var walk func(any)
	walk = func(node any) {
		switch t := node.(type) {
		case map[string]any:
			if m, ok := antigravityMetricFromMap(t, now); ok {
				key := m.Key
				if key == "" {
					key = strings.ToLower(strings.TrimSpace(m.MeteredFeature)) + ":" + strings.ToLower(strings.TrimSpace(m.Window))
				}
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					metrics = append(metrics, m)
				}
			}
			for _, v := range t {
				walk(v)
			}
		case []any:
			for _, v := range t {
				walk(v)
			}
		}
	}
	walk(payload)
	sort.SliceStable(metrics, func(i, j int) bool {
		if metrics[i].MeteredFeature != metrics[j].MeteredFeature {
			return metrics[i].MeteredFeature < metrics[j].MeteredFeature
		}
		return metrics[i].Window < metrics[j].Window
	})
	return metrics
}

func antigravityMetricFromMap(m map[string]any, now time.Time) (ProviderQuotaMetric, bool) {
	used, hasUsed := asFloat(firstMapValue(m, "used_percent", "usedPercent", "usagePercent", "quotaUsedPercent"))
	left, hasLeft := asFloat(firstMapValue(m, "left_percent", "leftPercent", "remainingPercent", "quotaRemainingPercent"))
	if !hasUsed && !hasLeft {
		return ProviderQuotaMetric{}, false
	}
	if !hasUsed && hasLeft {
		used = 100 - left
		hasUsed = true
	}
	if !hasLeft && hasUsed {
		left = 100 - used
		hasLeft = true
	}
	if !hasUsed || !hasLeft {
		return ProviderQuotaMetric{}, false
	}
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	if left < 0 {
		left = 0
	}
	if left > 100 {
		left = 100
	}

	feature := asString(firstMapValue(m, "metered_feature", "meteredFeature", "model", "model_id", "name", "id"))
	if feature == "" {
		feature = "gemini"
	}
	window := asString(firstMapValue(m, "window", "windowName", "period"))
	windowSeconds, _ := asInt64(firstMapValue(m, "window_seconds", "windowSeconds", "limit_window_seconds", "limitWindowSeconds"))
	if window == "" {
		window = normalizeQuotaWindowLabel("", windowSeconds)
	}
	if window == "" {
		window = "unknown"
	}

	resetAt := ""
	if v := firstMapValue(m, "reset_at", "resetAt", "resetTime", "quotaResetTimeStamp", "quota_reset_timestamp"); v != nil {
		switch t := v.(type) {
		case string:
			if ts := strings.TrimSpace(t); ts != "" {
				if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
					resetAt = parsed.UTC().Format(time.RFC3339)
				} else if unix, ok := asInt64(ts); ok && unix > 0 {
					resetAt = time.Unix(unix, 0).UTC().Format(time.RFC3339)
				}
			}
		default:
			if unix, ok := asInt64(v); ok && unix > 0 {
				// Treat large numbers as milliseconds.
				if unix > 1_000_000_000_000 {
					unix = unix / 1000
				}
				resetAt = time.Unix(unix, 0).UTC().Format(time.RFC3339)
			}
		}
	}
	if resetAt == "" {
		if delay := asString(firstMapValue(m, "quotaResetDelay", "retryDelay")); delay != "" {
			if d, ok := parseDurationLike(delay); ok {
				resetAt = now.Add(d).UTC().Format(time.RFC3339)
			}
		}
	}

	key := strings.ToLower(strings.TrimSpace(feature)) + ":" + strings.ToLower(strings.TrimSpace(window))
	return ProviderQuotaMetric{
		Key:            key,
		MeteredFeature: feature,
		Window:         window,
		WindowSeconds:  windowSeconds,
		LeftPercent:    left,
		UsedValue:      used,
		RemainingValue: left,
		LimitValue:     100,
		Unit:           "percent",
		ResetAt:        resetAt,
	}, true
}

func firstMapValue(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}

func parseDurationLike(raw string) (time.Duration, bool) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return 0, false
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, true
	}
	// Support integer seconds as string.
	if iv, err := strconv.ParseInt(s, 10, 64); err == nil && iv >= 0 {
		return time.Duration(iv) * time.Second, true
	}
	return 0, false
}

func parseOpenAICodexQuotaMetrics(body []byte) (string, []ProviderQuotaMetric, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, err
	}
	planType := asString(payload["plan_type"])
	metrics := make([]ProviderQuotaMetric, 0, 8)
	seen := map[string]struct{}{}

	// Prefer explicit additional/model-specific limits first.
	for _, listKey := range []string{"additional_rate_limits", "model_rate_limits", "additional_model_rate_limits"} {
		items := asSlice(payload[listKey])
		for _, item := range items {
			m := asMap(item)
			if len(m) == 0 {
				continue
			}
			feature := asString(m["metered_feature"])
			if feature == "" {
				feature = asString(m["model"])
			}
			if feature == "" {
				feature = asString(m["model_id"])
			}
			if feature == "" {
				feature = asString(m["name"])
			}
			if feature == "" {
				feature = "codex"
			}
			appendRateLimitMetrics(&metrics, seen, asMap(m["rate_limit"]), feature)
		}
	}

	// Fallback root limit if additional limits are absent.
	appendRateLimitMetrics(&metrics, seen, asMap(payload["rate_limit"]), "codex")

	sort.SliceStable(metrics, func(i, j int) bool {
		if metrics[i].MeteredFeature != metrics[j].MeteredFeature {
			return metrics[i].MeteredFeature < metrics[j].MeteredFeature
		}
		if metrics[i].WindowSeconds != metrics[j].WindowSeconds {
			return metrics[i].WindowSeconds < metrics[j].WindowSeconds
		}
		return metrics[i].Window < metrics[j].Window
	})

	return planType, metrics, nil
}

func appendRateLimitMetrics(dst *[]ProviderQuotaMetric, seen map[string]struct{}, rate map[string]any, feature string) {
	if len(rate) == 0 {
		return
	}
	keys := make([]string, 0, len(rate))
	for k := range rate {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		window := asMap(rate[key])
		if len(window) == 0 {
			continue
		}
		usedPercent, ok := asFloat(window["used_percent"])
		if !ok {
			continue
		}
		windowSeconds, _ := asInt64(window["limit_window_seconds"])
		resetAt, _ := asInt64(window["reset_at"])
		if usedPercent < 0 {
			usedPercent = 0
		}
		if usedPercent > 100 {
			usedPercent = 100
		}
		windowName := normalizeQuotaWindowLabel(key, windowSeconds)
		keyID := strings.ToLower(strings.TrimSpace(feature)) + ":" + strings.ToLower(strings.TrimSpace(windowName))
		if _, exists := seen[keyID]; exists {
			continue
		}
		seen[keyID] = struct{}{}
		m := ProviderQuotaMetric{
			Key:            keyID,
			MeteredFeature: strings.TrimSpace(feature),
			Window:         windowName,
			WindowSeconds:  windowSeconds,
			LeftPercent:    100 - usedPercent,
			UsedValue:      usedPercent,
			RemainingValue: 100 - usedPercent,
			LimitValue:     100,
			Unit:           "percent",
		}
		if resetAt > 0 {
			m.ResetAt = time.Unix(resetAt, 0).UTC().Format(time.RFC3339)
		}
		*dst = append(*dst, m)
	}
}

func pickPreferredQuotaMetric(metrics []ProviderQuotaMetric) ProviderQuotaMetric {
	if len(metrics) == 0 {
		return ProviderQuotaMetric{}
	}
	contains := func(haystack, needle string) bool {
		return strings.Contains(strings.ToLower(strings.TrimSpace(haystack)), needle)
	}
	priority := []func(ProviderQuotaMetric) bool{
		func(m ProviderQuotaMetric) bool {
			return contains(m.MeteredFeature, "codex") && !contains(m.MeteredFeature, "spark") && m.WindowSeconds > 0 && m.WindowSeconds <= 6*3600
		},
		func(m ProviderQuotaMetric) bool {
			return contains(m.MeteredFeature, "codex") && !contains(m.MeteredFeature, "spark")
		},
		func(m ProviderQuotaMetric) bool {
			return contains(m.MeteredFeature, "codex") && m.WindowSeconds > 0 && m.WindowSeconds <= 6*3600
		},
	}
	for _, matcher := range priority {
		for _, m := range metrics {
			if matcher(m) {
				return m
			}
		}
	}
	return metrics[0]
}

func normalizeQuotaWindowLabel(raw string, windowSeconds int64) string {
	name := strings.TrimSpace(strings.ToLower(raw))
	if windowSeconds > 0 {
		switch {
		case windowSeconds%(7*24*3600) == 0:
			return fmt.Sprintf("%dd", windowSeconds/(24*3600))
		case windowSeconds%(24*3600) == 0:
			return fmt.Sprintf("%dd", windowSeconds/(24*3600))
		case windowSeconds%3600 == 0:
			return fmt.Sprintf("%dh", windowSeconds/3600)
		case windowSeconds%60 == 0:
			return fmt.Sprintf("%dm", windowSeconds/60)
		default:
			return fmt.Sprintf("%ds", windowSeconds)
		}
	}
	switch {
	case strings.Contains(name, "week"):
		return "7d"
	case strings.Contains(name, "day"):
		return "1d"
	case strings.Contains(name, "hour"), strings.Contains(name, "primary"):
		return "5h"
	default:
		return strings.ReplaceAll(name, "_", " ")
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func asString(v any) string {
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		if err == nil {
			return f, true
		}
	}
	raw := strings.TrimSpace(fmt.Sprintf("%v", v))
	if raw == "" || raw == "<nil>" {
		return 0, false
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func asInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int64:
		return t, true
	case float64:
		return int64(t), true
	case float32:
		return int64(t), true
	case json.Number:
		i, err := t.Int64()
		if err == nil {
			return i, true
		}
		f, ferr := t.Float64()
		if ferr == nil {
			return int64(f), true
		}
	}
	raw := strings.TrimSpace(fmt.Sprintf("%v", v))
	if raw == "" || raw == "<nil>" {
		return 0, false
	}
	i, err := strconv.ParseInt(raw, 10, 64)
	if err == nil {
		return i, true
	}
	f, ferr := strconv.ParseFloat(raw, 64)
	if ferr != nil {
		return 0, false
	}
	return int64(f), true
}

func (h *AdminHandler) securitySettingsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := h.store.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{
			"allow_localhost_no_auth":            cfg.AllowLocalhostNoAuth,
			"allow_host_docker_internal_no_auth": cfg.AllowHostDockerInternalNoAuth,
			"auto_enable_public_free_models":     cfg.AutoEnablePublicFreeModels,
			"auto_detect_local_servers":          cfg.AutoDetectLocalServers,
			"auto_remove_expired_tokens":         cfg.AutoRemoveExpiredTokens,
			"auto_remove_empty_quota_tokens":     cfg.AutoRemoveEmptyQuotaTokens,
		})
	case http.MethodPut:
		var payload struct {
			AllowLocalhostNoAuth          bool  `json:"allow_localhost_no_auth"`
			AllowHostDockerInternalNoAuth bool  `json:"allow_host_docker_internal_no_auth"`
			AutoEnablePublicFreeModels    bool  `json:"auto_enable_public_free_models"`
			AutoDetectLocalServers        *bool `json:"auto_detect_local_servers"`
			AutoRemoveExpiredTokens       bool  `json:"auto_remove_expired_tokens"`
			AutoRemoveEmptyQuotaTokens    bool  `json:"auto_remove_empty_quota_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := h.store.Update(func(c *config.ServerConfig) error {
			c.AllowLocalhostNoAuth = payload.AllowLocalhostNoAuth
			c.AllowHostDockerInternalNoAuth = payload.AllowHostDockerInternalNoAuth
			c.AutoEnablePublicFreeModels = payload.AutoEnablePublicFreeModels
			if payload.AutoDetectLocalServers != nil {
				c.AutoDetectLocalServers = *payload.AutoDetectLocalServers
			}
			c.AutoRemoveExpiredTokens = payload.AutoRemoveExpiredTokens
			c.AutoRemoveEmptyQuotaTokens = payload.AutoRemoveEmptyQuotaTokens
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cfg := h.store.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":                             "ok",
			"allow_localhost_no_auth":            cfg.AllowLocalhostNoAuth,
			"allow_host_docker_internal_no_auth": cfg.AllowHostDockerInternalNoAuth,
			"auto_enable_public_free_models":     cfg.AutoEnablePublicFreeModels,
			"auto_detect_local_servers":          cfg.AutoDetectLocalServers,
			"auto_remove_expired_tokens":         cfg.AutoRemoveExpiredTokens,
			"auto_remove_empty_quota_tokens":     cfg.AutoRemoveEmptyQuotaTokens,
		})
		if h.runAccessTokenCleanup != nil {
			h.runAccessTokenCleanup()
		}
		if h.healthChecker != nil {
			h.healthChecker.Trigger()
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) tlsSettingsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := h.store.Snapshot()
		bindMode, customBind, port := decomposeListenAddr(strings.TrimSpace(cfg.TLS.ListenAddr))
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":         cfg.TLS.Enabled,
			"mode":            strings.TrimSpace(cfg.TLS.Mode),
			"domain":          strings.TrimSpace(cfg.TLS.Domain),
			"email":           strings.TrimSpace(cfg.TLS.Email),
			"cache_dir":       strings.TrimSpace(cfg.TLS.CacheDir),
			"listen_addr":     strings.TrimSpace(cfg.TLS.ListenAddr),
			"bind_mode":       bindMode,
			"custom_bind":     customBind,
			"port":            port,
			"cert_configured": strings.TrimSpace(cfg.TLS.CertPEM) != "",
			"key_configured":  strings.TrimSpace(cfg.TLS.KeyPEM) != "",
		})
	case http.MethodPut:
		var payload struct {
			Enabled    bool   `json:"enabled"`
			Mode       string `json:"mode"`
			BindMode   string `json:"bind_mode"`
			CustomBind string `json:"custom_bind"`
			Port       int    `json:"port"`
			Domain     string `json:"domain"`
			Email      string `json:"email"`
			CertPEM    string `json:"cert_pem"`
			KeyPEM     string `json:"key_pem"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		cfg := h.store.Snapshot()
		mode := strings.ToLower(strings.TrimSpace(payload.Mode))
		if mode == "" {
			mode = "letsencrypt"
		}
		if mode != "letsencrypt" && mode != "self_signed" && mode != "pem" {
			http.Error(w, "mode must be one of letsencrypt, self_signed, pem", http.StatusBadRequest)
			return
		}
		domain := strings.TrimSpace(payload.Domain)
		email := strings.TrimSpace(payload.Email)
		listenAddr, bindMode, customBind, port, err := composeListenAddr(payload.BindMode, payload.CustomBind, payload.Port)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.TLS != nil {
			writeJSON(w, http.StatusConflict, map[string]any{
				"status":   "transition_required",
				"error":    "switch to HTTP admin before applying HTTPS listener changes",
				"http_url": redirectURLForListenerWithScheme(r, strings.TrimSpace(cfg.ListenAddr), "http"),
			})
			return
		}
		certPEM := strings.TrimSpace(payload.CertPEM)
		keyPEM := strings.TrimSpace(payload.KeyPEM)
		if payload.Enabled && mode == "letsencrypt" && domain == "" {
			http.Error(w, "domain is required when tls mode is letsencrypt", http.StatusBadRequest)
			return
		}
		if payload.Enabled && mode == "letsencrypt" && strings.ToLower(strings.TrimSpace(cfg.HTTPMode)) == "disabled" {
			http.Error(w, "http mode must be enabled or when_required when tls mode is letsencrypt", http.StatusBadRequest)
			return
		}
		if payload.Enabled && mode == "pem" && (certPEM == "" || keyPEM == "") {
			// Permit preserving already configured PEM material when fields are omitted.
			if strings.TrimSpace(cfg.TLS.CertPEM) == "" || strings.TrimSpace(cfg.TLS.KeyPEM) == "" {
				http.Error(w, "cert_pem and key_pem are required when tls mode is pem", http.StatusBadRequest)
				return
			}
		}
		if payload.Enabled && isPublicFacingListenAddr(listenAddr) {
			if cfg.AllowLocalhostNoAuth || cfg.AllowHostDockerInternalNoAuth {
				http.Error(w, "disable unauth localhost/docker access before enabling public tls bind", http.StatusBadRequest)
				return
			}
			if !config.HasAdminToken(cfg.IncomingTokens) {
				http.Error(w, "configure at least one non-expired admin token before enabling public tls bind", http.StatusBadRequest)
				return
			}
		}
		if err := h.store.Update(func(c *config.ServerConfig) error {
			c.TLS.Enabled = payload.Enabled
			c.TLS.Mode = mode
			c.TLS.ListenAddr = listenAddr
			c.TLS.Domain = domain
			c.TLS.Email = email
			if certPEM != "" {
				c.TLS.CertPEM = certPEM
			}
			if keyPEM != "" {
				c.TLS.KeyPEM = keyPEM
			}
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.notifyAdminChanged("network")
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "ok",
			"mode":        mode,
			"bind_mode":   bindMode,
			"custom_bind": customBind,
			"port":        port,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) networkSettingsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := h.store.Snapshot()
	bindMode, customBind, port := decomposeListenAddr(strings.TrimSpace(cfg.ListenAddr))
	active := []string{}
	h.networkMu.Lock()
	if h.listListeners != nil {
		active = h.listListeners()
	}
	var pending map[string]any
	for _, sw := range h.networkSwitch {
		pending = map[string]any{
			"id":         sw.ID,
			"old_addr":   sw.OldAddr,
			"new_addr":   sw.NewAddr,
			"created_at": sw.Created.Format(time.RFC3339),
		}
		break
	}
	h.networkMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"bind_mode":   bindMode,
		"custom_bind": customBind,
		"port":        port,
		"http_mode":   strings.TrimSpace(cfg.HTTPMode),
		"listen_addr": strings.TrimSpace(cfg.ListenAddr),
		"active":      active,
		"local_addrs": detectLocalBindCandidates(),
		"pending":     pending,
	})
}

func (h *AdminHandler) networkApplyAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := h.store.Snapshot()
	var payload struct {
		BindMode   string `json:"bind_mode"`
		CustomBind string `json:"custom_bind"`
		Port       int    `json:"port"`
		HTTPMode   string `json:"http_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	httpMode := strings.ToLower(strings.TrimSpace(payload.HTTPMode))
	if httpMode == "" {
		httpMode = "enabled"
	}
	if httpMode != "enabled" && httpMode != "when_required" && httpMode != "disabled" {
		http.Error(w, "http_mode must be one of enabled, when_required, disabled", http.StatusBadRequest)
		return
	}
	if httpMode == "disabled" && r.TLS == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error",
			"error":  "don't saw off your ui legs",
		})
		return
	}
	if httpMode == "disabled" && !cfg.TLS.Enabled {
		http.Error(w, "cannot disable http listener while tls is not enabled", http.StatusBadRequest)
		return
	}
	if cfg.TLS.Enabled && strings.ToLower(strings.TrimSpace(cfg.TLS.Mode)) == "letsencrypt" && httpMode == "disabled" {
		http.Error(w, "http_mode cannot be disabled while letsencrypt tls is enabled", http.StatusBadRequest)
		return
	}
	newAddr, bindMode, customBind, port, err := composeListenAddr(payload.BindMode, payload.CustomBind, payload.Port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	oldAddr := strings.TrimSpace(cfg.ListenAddr)
	if oldAddr == "" {
		oldAddr = "127.0.0.1:7050"
	}
	if newAddr == oldAddr && httpMode == strings.ToLower(strings.TrimSpace(cfg.HTTPMode)) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "ok",
			"http_mode":   httpMode,
			"bind_mode":   bindMode,
			"custom_bind": customBind,
			"port":        port,
			"new_addr":    newAddr,
			"old_addr":    oldAddr,
		})
		return
	}
	if cfg.TLS.Enabled {
		if newAddr != oldAddr {
			http.Error(w, "network bind/port apply is not supported while TLS listener mode is enabled", http.StatusBadRequest)
			return
		}
		if httpMode == "disabled" && r.TLS == nil {
			tlsAddr := strings.TrimSpace(cfg.TLS.ListenAddr)
			if tlsAddr == "" {
				tlsAddr = ":443"
			}
			writeJSON(w, http.StatusConflict, map[string]any{
				"status":    "transition_required",
				"error":     "switch admin to https before disabling http listener",
				"https_url": redirectURLForListenerWithScheme(r, tlsAddr, "https"),
			})
			return
		}
		h.networkMu.Lock()
		addFn := h.addListener
		removeFn := h.removeListener
		h.networkMu.Unlock()
		if httpMode == "enabled" {
			if addFn == nil {
				http.Error(w, "listener control unavailable", http.StatusServiceUnavailable)
				return
			}
			if err := addFn(oldAddr); err != nil {
				http.Error(w, "failed to enable http listener: "+err.Error(), http.StatusBadGateway)
				return
			}
		} else {
			if removeFn == nil {
				http.Error(w, "listener control unavailable", http.StatusServiceUnavailable)
				return
			}
			if err := removeFn(oldAddr); err != nil {
				http.Error(w, "failed to disable http listener: "+err.Error(), http.StatusBadGateway)
				return
			}
		}
		if err := h.store.Update(func(c *config.ServerConfig) error {
			c.HTTPMode = httpMode
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.notifyAdminChanged("network")
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "ok",
			"http_mode":   httpMode,
			"bind_mode":   bindMode,
			"custom_bind": customBind,
			"port":        port,
			"new_addr":    newAddr,
			"old_addr":    oldAddr,
		})
		return
	}
	if isPublicFacingListenAddr(newAddr) {
		if cfg.AllowLocalhostNoAuth || cfg.AllowHostDockerInternalNoAuth {
			http.Error(w, "disable unauth localhost/docker access before enabling public bind", http.StatusBadRequest)
			return
		}
		if !config.HasAdminToken(cfg.IncomingTokens) {
			http.Error(w, "configure at least one non-expired admin token before enabling public bind", http.StatusBadRequest)
			return
		}
	}
	h.networkMu.Lock()
	addFn := h.addListener
	h.networkMu.Unlock()
	if addFn == nil {
		http.Error(w, "listener control unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := addFn(newAddr); err != nil {
		http.Error(w, "failed to add listener: "+err.Error(), http.StatusBadGateway)
		return
	}

	id, err := randomURLSafe(12)
	if err != nil {
		http.Error(w, "failed to create switch id", http.StatusInternalServerError)
		return
	}
	sw := pendingNetworkSwitch{
		ID:       id,
		OldAddr:  oldAddr,
		NewAddr:  newAddr,
		HTTPMode: httpMode,
		Created:  time.Now().UTC(),
	}
	h.networkMu.Lock()
	h.networkSwitch = map[string]pendingNetworkSwitch{id: sw}
	h.networkMu.Unlock()
	h.notifyAdminChanged("network")
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "pending_probe",
		"switch_id":   id,
		"instance_id": h.instance,
		"http_mode":   httpMode,
		"bind_mode":   bindMode,
		"custom_bind": customBind,
		"port":        port,
		"new_addr":    newAddr,
		"old_addr":    oldAddr,
	})
}

func (h *AdminHandler) networkConfirmAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		SwitchID string `json:"switch_id"`
		ProbeOK  bool   `json:"probe_ok"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	switchID := strings.TrimSpace(payload.SwitchID)
	if switchID == "" {
		http.Error(w, "switch_id is required", http.StatusBadRequest)
		return
	}
	h.networkMu.Lock()
	sw, ok := h.networkSwitch[switchID]
	removeFn := h.removeListener
	delete(h.networkSwitch, switchID)
	h.networkMu.Unlock()
	if !ok {
		http.Error(w, "switch not found", http.StatusBadRequest)
		return
	}
	if !payload.ProbeOK {
		if removeFn != nil {
			_ = removeFn(sw.NewAddr)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled"})
		return
	}
	hardened := false
	if err := h.store.Update(func(c *config.ServerConfig) error {
		c.ListenAddr = sw.NewAddr
		c.HTTPMode = sw.HTTPMode
		if isPublicFacingListenAddr(sw.NewAddr) {
			if c.AllowLocalhostNoAuth || c.AllowHostDockerInternalNoAuth {
				hardened = true
			}
			c.AllowLocalhostNoAuth = false
			c.AllowHostDockerInternalNoAuth = false
		}
		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if removeFn != nil && strings.TrimSpace(sw.OldAddr) != "" && sw.OldAddr != sw.NewAddr {
		_ = removeFn(sw.OldAddr)
	}
	h.notifyAdminChanged("network")
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"listen_addr":  sw.NewAddr,
		"redirect_url": redirectURLForNewListener(r, sw.NewAddr),
		"hardened":     hardened,
	})
}

func (h *AdminHandler) networkProbeAPI(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"instance": h.instance,
	})
}

func decomposeListenAddr(listenAddr string) (bindMode string, customBind string, port int) {
	host, p := splitHostPortLoose(strings.TrimSpace(listenAddr))
	if p <= 0 {
		p = 7050
	}
	switch strings.TrimSpace(host) {
	case "", "0.0.0.0", "::":
		return "all", "", p
	case "127.0.0.1", "localhost":
		return "localhost", "", p
	default:
		return "custom", host, p
	}
}

func composeListenAddr(bindMode, customBind string, port int) (listenAddr string, normalizedMode string, normalizedCustom string, normalizedPort int, err error) {
	mode := strings.ToLower(strings.TrimSpace(bindMode))
	if mode == "" {
		mode = "localhost"
	}
	if port <= 0 || port > 65535 {
		return "", "", "", 0, fmt.Errorf("port must be between 1 and 65535")
	}
	host := ""
	switch mode {
	case "localhost":
		host = "127.0.0.1"
	case "all":
		host = "0.0.0.0"
	case "custom":
		host = strings.TrimSpace(customBind)
		if host == "" {
			return "", "", "", 0, fmt.Errorf("custom bind host is required")
		}
	default:
		return "", "", "", 0, fmt.Errorf("bind_mode must be localhost, all, or custom")
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), mode, host, port, nil
}

func splitHostPortLoose(addr string) (host string, port int) {
	a := strings.TrimSpace(addr)
	if a == "" {
		return "", 7050
	}
	if strings.HasPrefix(a, ":") {
		if n, err := strconv.Atoi(strings.TrimPrefix(a, ":")); err == nil {
			return "", n
		}
		return "", 7050
	}
	h, p, err := net.SplitHostPort(a)
	if err != nil {
		return a, 7050
	}
	n, _ := strconv.Atoi(strings.TrimSpace(p))
	return strings.TrimSpace(h), n
}

func redirectURLForNewListener(r *http.Request, listenAddr string) string {
	return redirectURLForListenerWithScheme(r, listenAddr, "")
}

func redirectURLForListenerWithScheme(r *http.Request, listenAddr string, forceScheme string) string {
	host, port := splitHostPortLoose(listenAddr)
	scheme := "http"
	if strings.TrimSpace(forceScheme) != "" {
		scheme = strings.TrimSpace(forceScheme)
	} else if r.TLS != nil {
		scheme = "https"
	}
	currentHost := strings.TrimSpace(r.Host)
	currentName := currentHost
	if h, _, err := net.SplitHostPort(currentHost); err == nil {
		currentName = h
	}
	targetHost := strings.TrimSpace(host)
	if targetHost == "" || targetHost == "0.0.0.0" || targetHost == "::" {
		targetHost = currentName
		if targetHost == "" {
			targetHost = "127.0.0.1"
		}
	}
	return fmt.Sprintf("%s://%s/admin", scheme, net.JoinHostPort(targetHost, strconv.Itoa(port)))
}

func isPublicFacingListenAddr(listenAddr string) bool {
	host, _ := splitHostPortLoose(listenAddr)
	h := strings.ToLower(strings.TrimSpace(host))
	switch h {
	case "", "0.0.0.0", "::":
		return true
	}
	return !isLoopbackHost(h)
}

func isLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

func detectLocalBindCandidates() []string {
	addrs := map[string]struct{}{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{}
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		iaddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range iaddrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
				continue
			}
			ip = ip.To16()
			if ip == nil {
				continue
			}
			// Prefer IPv4 candidates in UI; include global/private unicast only.
			if v4 := ip.To4(); v4 != nil {
				addrs[v4.String()] = struct{}{}
				continue
			}
			if ip.IsGlobalUnicast() {
				addrs[ip.String()] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(addrs))
	for ip := range addrs {
		out = append(out, ip)
	}
	sort.Strings(out)
	return out
}

func (h *AdminHandler) conversationsSettingsAPI(w http.ResponseWriter, r *http.Request) {
	if h.conversations == nil {
		http.Error(w, "conversations store unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg := h.store.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":      cfg.Conversations.Enabled,
			"max_items":    cfg.Conversations.MaxItems,
			"max_age_days": cfg.Conversations.MaxAgeDays,
		})
	case http.MethodPut:
		var payload struct {
			Enabled    bool `json:"enabled"`
			MaxItems   int  `json:"max_items"`
			MaxAgeDays int  `json:"max_age_days"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := h.store.Update(func(c *config.ServerConfig) error {
			c.Conversations.Enabled = payload.Enabled
			c.Conversations.MaxItems = payload.MaxItems
			c.Conversations.MaxAgeDays = payload.MaxAgeDays
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cfg := h.store.Snapshot()
		h.conversations.UpdateSettings(conversations.Settings{
			Enabled:    cfg.Conversations.Enabled,
			MaxItems:   cfg.Conversations.MaxItems,
			MaxAgeDays: cfg.Conversations.MaxAgeDays,
		})
		h.invalidateConversationViewCacheAll()
		h.notifyAdminChanged("conversations")
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) logsSettingsAPI(w http.ResponseWriter, r *http.Request) {
	if h.logs == nil {
		http.Error(w, "logs store unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg := h.store.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{
			"max_lines": cfg.Logs.MaxLines,
		})
	case http.MethodPut:
		var payload struct {
			MaxLines int `json:"max_lines"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := h.store.Update(func(c *config.ServerConfig) error {
			c.Logs.MaxLines = payload.MaxLines
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cfg := h.store.Snapshot()
		h.logs.UpdateSettings(logstore.Settings{
			MaxLines: cfg.Logs.MaxLines,
		})
		h.notifyAdminChanged("log")
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) logsAPI(w http.ResponseWriter, r *http.Request) {
	if h.logs == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": []any{},
		})
		return
	}
	if r.Method == http.MethodDelete {
		h.logs.Clear()
		h.notifyAdminChanged("log")
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filter := logstore.ListFilter{
		Level: strings.TrimSpace(r.URL.Query().Get("level")),
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
	}
	page := 1
	pageSize := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			page = n
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("page_size")); raw != "" {
		if strings.EqualFold(raw, "all") || raw == "0" {
			pageSize = 0
		} else if n, err := strconv.Atoi(raw); err == nil {
			pageSize = n
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			// Backward-compat for older clients that only set limit.
			page = 1
			pageSize = n
		}
	}
	entries, total, outPage, outPageSize := h.logs.ListPage(filter, page, pageSize)
	writeJSON(w, http.StatusOK, map[string]any{
		"entries":   entries,
		"total":     total,
		"page":      outPage,
		"page_size": outPageSize,
	})
}

func (h *AdminHandler) conversationsAPI(w http.ResponseWriter, r *http.Request) {
	if h.conversations == nil {
		writeJSON(w, http.StatusOK, map[string]any{"threads": []any{}, "total": 0, "next_before": ""})
		return
	}
	if r.Method == http.MethodDelete {
		removed := h.conversations.ClearAll()
		h.invalidateConversationViewCacheAll()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"removed": removed,
		})
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filter := conversations.ListFilter{
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
		Provider:   strings.TrimSpace(r.URL.Query().Get("provider")),
		Model:      strings.TrimSpace(r.URL.Query().Get("model")),
		APIKeyName: strings.TrimSpace(r.URL.Query().Get("api_key_name")),
		RemoteIP:   strings.TrimSpace(r.URL.Query().Get("remote_ip")),
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			filter.Limit = n
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("before")); raw != "" {
		if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			filter.Before = ts
		}
	}
	threads, nextBefore, total := h.conversations.ListThreads(filter)
	for i := range threads {
		key := strings.TrimSpace(threads[i].ConversationKey)
		if key == "" {
			continue
		}
		records := h.conversations.Conversation(key)
		threads[i].Title = conversations.ExtractConversationTitle(records)
		views := buildConversationRecordViews(records, false)
		threads[i].Count = len(views)
		var tokenCount int64
		for j := range records {
			tokenCount += conversationRecordTotalTokens(records[j])
		}
		threads[i].TokenCount = tokenCount
	}
	settings := h.conversations.Settings()
	writeJSON(w, http.StatusOK, map[string]any{
		"threads":      threads,
		"total":        total,
		"next_before":  nextBefore,
		"max_items":    settings.MaxItems,
		"max_age_days": settings.MaxAgeDays,
		"enabled":      settings.Enabled,
	})
}

func conversationRecordTotalTokens(rec conversations.Record) int64 {
	structured := rec.ToStructuredRecord()
	total := 0
	total += estimateTokensFromText(strings.TrimSpace(structured.RequestTextMarkdown))
	total += estimateTokensFromText(strings.TrimSpace(structured.ResponseTextMarkdown))
	total += estimateTokensFromText(strings.TrimSpace(structured.RequestSystemMarkdown))
	total += estimateTokensFromText(strings.TrimSpace(structured.ResponseThinkMarkdown))
	if total < 0 {
		total = 0
	}
	return int64(total)
}

func parseTotalTokensFromRawPayload(raw string) int64 {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0
	}
	if strings.HasPrefix(strings.ToLower(s), "data:") || strings.Contains(s, "\ndata:") {
		lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
		var total int64
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(strings.ToLower(line), "data:") {
				continue
			}
			payload := strings.TrimSpace(line[5:])
			if payload == "" || payload == "[DONE]" {
				continue
			}
			p, c, t := parseUsageTokens([]byte(payload))
			if t > 0 {
				total = int64(t)
				continue
			}
			if p > 0 || c > 0 {
				total = int64(p + c)
			}
		}
		return total
	}
	p, c, t := parseUsageTokens([]byte(s))
	if t > 0 {
		return int64(t)
	}
	if p > 0 || c > 0 {
		return int64(p + c)
	}
	return 0
}

func (h *AdminHandler) conversationByIDAPI(w http.ResponseWriter, r *http.Request) {
	if h.conversations == nil {
		writeJSON(w, http.StatusOK, map[string]any{"records": []any{}})
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if decoded, err := url.PathUnescape(id); err == nil && strings.TrimSpace(decoded) != "" {
		id = strings.TrimSpace(decoded)
	}
	if r.Method == http.MethodDelete {
		removed := h.conversations.DeleteConversation(id)
		h.invalidateConversationViewCacheForKey(id)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"id":      id,
			"removed": removed,
		})
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	includeInternal := false
	if raw := strings.TrimSpace(r.URL.Query().Get("include_internal")); raw != "" {
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "on":
			includeInternal = true
		}
	}
	if title, views, ok := h.getConversationViewCache(id, includeInternal); ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"id":      id,
			"title":   title,
			"records": views,
		})
		return
	}
	records := h.conversations.Conversation(id)
	title := conversations.ExtractConversationTitle(records)
	views := buildConversationRecordViews(records, includeInternal)
	h.setConversationViewCache(id, includeInternal, title, views)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"title":   title,
		"records": views,
	})
}

func buildConversationRecordViews(records []conversations.Record, includeInternal bool) []conversationRecordView {
	return conversations.BuildConversationRecordViewsWithOptions(records, conversations.ConversationViewOptions{
		IncludeInternal: includeInternal,
	})
}

func extractConversationDelta(current, previous string) string {
	return conversations.ExtractConversationDelta(current, previous)
}

func (h *AdminHandler) tlsTestCertificateAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := h.store.Snapshot()
	if strings.TrimSpace(cfg.TLS.Mode) != "letsencrypt" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error",
			"error":  "certificate test is only available for letsencrypt mode",
		})
		return
	}
	notAfter, cacheDir, err := obtainManagedCertificate(cfg.TLS, true, false)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	msg := "test certificate obtained"
	if !notAfter.IsZero() {
		msg = "test certificate obtained; expires " + notAfter.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"message":   msg,
		"not_after": notAfter.UTC().Format(time.RFC3339),
		"cache_dir": cacheDir,
	})
}

func (h *AdminHandler) tlsRenewCertificateAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := h.store.Snapshot()
	if strings.TrimSpace(cfg.TLS.Mode) != "letsencrypt" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error",
			"error":  "certificate renew is only available for letsencrypt mode",
		})
		return
	}
	notAfter, cacheDir, err := obtainManagedCertificate(cfg.TLS, false, true)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	msg := "certificate renewed"
	if !notAfter.IsZero() {
		msg = "certificate renewed; expires " + notAfter.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"message":   msg,
		"not_after": notAfter.UTC().Format(time.RFC3339),
		"cache_dir": cacheDir,
	})
}

func (h *AdminHandler) versionAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	v := version.Current()
	writeJSON(w, http.StatusOK, map[string]any{
		"version": version.String(),
		"raw":     v.Version,
		"commit":  v.Commit,
		"date":    v.Date,
		"dirty":   v.Dirty,
	})
}

func obtainManagedCertificate(tlsCfg config.TLSConfig, useStaging bool, forceRenew bool) (time.Time, string, error) {
	domain := strings.TrimSpace(tlsCfg.Domain)
	if domain == "" {
		return time.Time{}, "", fmt.Errorf("tls domain is not configured")
	}
	cacheDir := strings.TrimSpace(tlsCfg.CacheDir)
	if cacheDir == "" {
		cacheDir = config.DefaultTLSCacheDir()
	}
	if useStaging {
		cacheDir = filepath.Join(cacheDir, "staging-test")
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return time.Time{}, cacheDir, fmt.Errorf("create cache dir: %w", err)
	}
	if forceRenew {
		purgeAutocertDomainCache(cacheDir, domain)
	}

	mgr := &autocert.Manager{
		Cache:      autocert.DirCache(cacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
		Email:      strings.TrimSpace(tlsCfg.Email),
	}
	if useStaging {
		mgr.Client = &acme.Client{DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory"}
	}

	type result struct {
		notAfter time.Time
		err      error
	}
	done := make(chan result, 1)
	go func() {
		cert, err := mgr.GetCertificate(&tls.ClientHelloInfo{
			ServerName: domain,
		})
		if err != nil {
			done <- result{err: err}
			return
		}
		if cert == nil || len(cert.Certificate) == 0 {
			done <- result{}
			return
		}
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			done <- result{}
			return
		}
		done <- result{notAfter: leaf.NotAfter.UTC()}
	}()

	select {
	case out := <-done:
		return out.notAfter, cacheDir, out.err
	case <-time.After(2 * time.Minute):
		return time.Time{}, cacheDir, fmt.Errorf("timed out waiting for certificate challenge completion")
	}
}

func purgeAutocertDomainCache(cacheDir, domain string) {
	dc := autocert.DirCache(cacheDir)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = dc.Delete(ctx, domain)
	_ = dc.Delete(ctx, domain+"+rsa")
	_ = dc.Delete(ctx, domain+"+ecdsa")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	needle := strings.ToLower(strings.TrimSpace(domain))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(entry.Name()))
		if !strings.Contains(name, needle) {
			continue
		}
		_ = os.Remove(filepath.Join(cacheDir, entry.Name()))
	}
}

func (h *AdminHandler) accessTokensAPI(w http.ResponseWriter, r *http.Request) {
	actor, ok := adminIdentityFromContext(r.Context())
	if !ok {
		actor = tokenAuthIdentity{Role: config.TokenRoleAdmin}
	}
	switch r.Method {
	case http.MethodGet:
		cfg := h.store.Snapshot()
		type tokenItem struct {
			ID        string             `json:"id"`
			Name      string             `json:"name"`
			Role      string             `json:"role,omitempty"`
			ParentID  string             `json:"parent_id,omitempty"`
			Quota     *config.TokenQuota `json:"quota,omitempty"`
			ExpiresAt string             `json:"expires_at,omitempty"`
		}
		out := make([]tokenItem, 0, len(cfg.IncomingTokens))
		for _, t := range cfg.IncomingTokens {
			if actor.Role == config.TokenRoleKeymaster && strings.TrimSpace(t.ParentID) != strings.TrimSpace(actor.Token.ID) {
				continue
			}
			out = append(out, tokenItem{
				ID:        strings.TrimSpace(t.ID),
				Name:      strings.TrimSpace(t.Name),
				Role:      config.NormalizeIncomingTokenRole(t.Role),
				ParentID:  strings.TrimSpace(t.ParentID),
				Quota:     t.Quota,
				ExpiresAt: strings.TrimSpace(t.ExpiresAt),
			})
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		var payload struct {
			Name      string `json:"name"`
			Key       string `json:"key"`
			Role      string `json:"role,omitempty"`
			ExpiresAt string `json:"expires_at,omitempty"`
			Quota     any    `json:"quota,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		key := strings.TrimSpace(payload.Key)
		if key == "" {
			http.Error(w, "key is required", http.StatusBadRequest)
			return
		}
		expiresAt := strings.TrimSpace(payload.ExpiresAt)
		if expiresAt != "" {
			if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
				http.Error(w, "expires_at must be RFC3339", http.StatusBadRequest)
				return
			}
		}
		name := strings.TrimSpace(payload.Name)
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		var quota *config.TokenQuota
		if payload.Quota != nil {
			qb, err := json.Marshal(payload.Quota)
			if err != nil {
				http.Error(w, "invalid quota", http.StatusBadRequest)
				return
			}
			var decoded config.TokenQuota
			if err := json.Unmarshal(qb, &decoded); err != nil {
				http.Error(w, "invalid quota", http.StatusBadRequest)
				return
			}
			quota = &decoded
		}
		role := config.NormalizeIncomingTokenRole(payload.Role)
		if role == "" {
			role = config.TokenRoleInferrer
		}
		parentID := ""
		if actor.Role == config.TokenRoleKeymaster {
			if role != config.TokenRoleInferrer {
				http.Error(w, "keymaster can only create inferrer tokens", http.StatusForbidden)
				return
			}
			if quota != nil {
				http.Error(w, "subordinate tokens cannot define quota", http.StatusForbidden)
				return
			}
			parentID = strings.TrimSpace(actor.Token.ID)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		if err := h.store.Update(func(c *config.ServerConfig) error {
			for _, existing := range c.IncomingTokens {
				if strings.TrimSpace(existing.Key) == key {
					return fmt.Errorf("token already exists")
				}
			}
			c.IncomingTokens = append(c.IncomingTokens, config.IncomingAPIToken{
				Name:      name,
				Key:       key,
				Role:      role,
				ParentID:  parentID,
				ExpiresAt: expiresAt,
				CreatedAt: now,
				Quota:     quota,
			})
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.notifyAdminChanged("access")
		writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) accessTokenByIDAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	actor, ok := adminIdentityFromContext(r.Context())
	if !ok {
		actor = tokenAuthIdentity{Role: config.TokenRoleAdmin}
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodPut {
		var payload struct {
			Name      string `json:"name"`
			Role      string `json:"role,omitempty"`
			ExpiresAt string `json:"expires_at,omitempty"`
			Quota     any    `json:"quota,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(payload.Name)
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		role := config.NormalizeIncomingTokenRole(payload.Role)
		if role == "" {
			http.Error(w, "role is required", http.StatusBadRequest)
			return
		}
		expiresAt := strings.TrimSpace(payload.ExpiresAt)
		if expiresAt != "" {
			if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
				http.Error(w, "expires_at must be RFC3339", http.StatusBadRequest)
				return
			}
		}
		var quota *config.TokenQuota
		if payload.Quota != nil {
			qb, err := json.Marshal(payload.Quota)
			if err != nil {
				http.Error(w, "invalid quota", http.StatusBadRequest)
				return
			}
			var decoded config.TokenQuota
			if err := json.Unmarshal(qb, &decoded); err != nil {
				http.Error(w, "invalid quota", http.StatusBadRequest)
				return
			}
			quota = &decoded
		}
		if err := h.store.Update(func(c *config.ServerConfig) error {
			for i := range c.IncomingTokens {
				t := &c.IncomingTokens[i]
				if strings.TrimSpace(t.ID) != id {
					continue
				}
				if actor.Role == config.TokenRoleKeymaster {
					if strings.TrimSpace(t.ParentID) != strings.TrimSpace(actor.Token.ID) {
						return fmt.Errorf("forbidden")
					}
					if role != config.TokenRoleInferrer {
						return fmt.Errorf("keymaster can only set role inferrer")
					}
					if quota != nil {
						return fmt.Errorf("subordinate tokens cannot define quota")
					}
				}
				t.Name = name
				t.Role = role
				t.ExpiresAt = expiresAt
				t.Quota = quota
				return nil
			}
			return fmt.Errorf("token not found")
		}); err != nil {
			if strings.EqualFold(strings.TrimSpace(err.Error()), "forbidden") {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.notifyAdminChanged("access")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if err := h.store.Update(func(c *config.ServerConfig) error {
		target := config.IncomingAPIToken{}
		for _, t := range c.IncomingTokens {
			if strings.TrimSpace(t.ID) == id {
				target = t
				break
			}
		}
		if actor.Role == config.TokenRoleKeymaster {
			if strings.TrimSpace(target.ID) == "" {
				return fmt.Errorf("token not found")
			}
			if strings.TrimSpace(target.ParentID) != strings.TrimSpace(actor.Token.ID) {
				return fmt.Errorf("forbidden")
			}
		}
		next := make([]config.IncomingAPIToken, 0, len(c.IncomingTokens))
		found := false
		for _, t := range c.IncomingTokens {
			if strings.TrimSpace(t.ID) == id {
				found = true
				continue
			}
			next = append(next, t)
		}
		if !found {
			return fmt.Errorf("token not found")
		}
		c.IncomingTokens = next
		return nil
	}); err != nil {
		if strings.EqualFold(strings.TrimSpace(err.Error()), "forbidden") {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.notifyAdminChanged("access")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminHandler) providersAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := h.store.Snapshot()
		configuredByName := make(map[string]struct{}, len(cfg.Providers))
		configuredEnabled := make(map[string]bool, len(cfg.Providers))
		for _, p := range cfg.Providers {
			configuredByName[p.Name] = struct{}{}
			configuredEnabled[p.Name] = p.Enabled
		}
		activeProviders := h.catalogProviders()
		activeByName := make(map[string]config.ProviderConfig, len(activeProviders))
		for _, p := range activeProviders {
			activeByName[p.Name] = p
		}
		providers := make([]config.ProviderConfig, 0, len(cfg.Providers)+len(activeProviders))
		seen := make(map[string]struct{}, len(cfg.Providers)+len(activeProviders))
		// Always include configured providers, including disabled ones.
		for _, p := range cfg.Providers {
			if ap, ok := activeByName[p.Name]; ok {
				providers = append(providers, ap)
			} else {
				providers = append(providers, h.applyPresetProviderDefaults(p))
			}
			seen[p.Name] = struct{}{}
		}
		// Then include auto-merged providers.
		for _, p := range activeProviders {
			if _, ok := seen[p.Name]; ok {
				continue
			}
			providers = append(providers, p)
			seen[p.Name] = struct{}{}
		}
		if h.pricing != nil {
			h.pricing.SetProviders(providers)
			h.pricing.EnsureFreshAsync()
		}
		displayNames := map[string]string{}
		if popular, err := getPopularProviders(); err == nil {
			for _, p := range popular {
				if strings.TrimSpace(p.DisplayName) != "" {
					displayNames[p.Name] = p.DisplayName
				}
			}
		}
		type providerListItem struct {
			DisplayName    string `json:"display_name"`
			Name           string `json:"name"`
			ProviderType   string `json:"provider_type,omitempty"`
			BaseURL        string `json:"base_url"`
			TimeoutSeconds int    `json:"timeout_seconds"`
			Status         string `json:"status"`
			ModelCount     int    `json:"model_count"`
			PricedModels   int    `json:"priced_models"`
			FreeModels     int    `json:"free_models"`
			PricingUpdated string `json:"pricing_last_update,omitempty"`
			ResponseMS     int64  `json:"response_ms,omitempty"`
			CheckedAt      string `json:"checked_at,omitempty"`
			Managed        bool   `json:"managed"`
		}
		var pricingSnapshot pricing.Cache
		if h.pricing != nil {
			pricingSnapshot = h.pricing.Snapshot()
		}
		out := make([]providerListItem, 0, len(providers))
		for _, p := range providers {
			providerType := providerTypeOrName(p)
			item := providerListItem{
				DisplayName:    p.Name,
				Name:           p.Name,
				ProviderType:   providerType,
				BaseURL:        p.BaseURL,
				TimeoutSeconds: p.TimeoutSeconds,
			}
			if p.Name == providerType {
				if dn, ok := displayNames[providerType]; ok {
					item.DisplayName = dn
				}
			} else if dn, ok := displayNames[p.Name]; ok {
				item.DisplayName = dn
			}
			_, item.Managed = configuredByName[p.Name]
			if item.Managed && !configuredEnabled[p.Name] {
				item.Status = "disabled"
			} else {
				item.Status = "unknown"
				if h.healthChecker != nil {
					if snap, ok := h.healthChecker.Snapshot(p.Name); ok {
						item.Status = snap.Status
						item.ModelCount = snap.ModelCount
						item.ResponseMS = snap.ResponseMS
						item.CheckedAt = snap.CheckedAt.Format(time.RFC3339)
					}
				}
			}
			if st, ok := pricingSnapshot.ProviderStates[p.Name]; ok && !st.LastUpdate.IsZero() {
				item.PricingUpdated = st.LastUpdate.Format(time.RFC3339)
			}
			for key := range pricingSnapshot.Entries {
				if strings.HasPrefix(key, p.Name+"/") {
					item.PricedModels++
					entry := pricingSnapshot.Entries[key]
					if entry.InputPer1M == 0 && entry.OutputPer1M == 0 {
						item.FreeModels++
					}
				}
			}
			// Auto-merged providers should be silent when offline.
			if !item.Managed && strings.EqualFold(strings.TrimSpace(item.Status), "offline") {
				continue
			}
			out = append(out, item)
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		var p config.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := h.validateProviderForSave(p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := h.store.Update(func(c *config.ServerConfig) error {
			for _, existing := range c.Providers {
				if existing.Name == p.Name {
					return fmt.Errorf("provider exists")
				}
			}
			c.Providers = append(c.Providers, p)
			if c.DefaultProvider == "" {
				c.DefaultProvider = p.Name
			}
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if h.healthChecker != nil {
			h.healthChecker.Trigger()
		}
		h.notifyAdminChanged("providers")
		writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) popularProvidersAPI(w http.ResponseWriter, r *http.Request) {
	providers, err := getPopularProviders()
	if err != nil {
		http.Error(w, "popular providers unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, providers)
}

func (h *AdminHandler) adminStaticAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(chi.URLParam(r, "*"))
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" || name == "." || strings.HasPrefix(name, "..") {
		http.NotFound(w, r)
		return
	}
	b, err := assets.LoadStaticAsset(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	// Admin assets change frequently during development; avoid stale UI behavior.
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	_, _ = w.Write(b)
}

func (h *AdminHandler) providerDeviceCodeAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider      string `json:"provider"`
		DeviceCodeURL string `json:"device_code_url"`
		ClientID      string `json:"client_id"`
		Scope         string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Provider = strings.TrimSpace(req.Provider)
	req.DeviceCodeURL = strings.TrimSpace(req.DeviceCodeURL)
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.Scope = strings.TrimSpace(req.Scope)

	var preset assets.PopularProvider
	var havePreset bool
	if req.Provider != "" {
		if popular, err := getPopularProviders(); err == nil {
			for _, p := range popular {
				if p.Name != req.Provider {
					continue
				}
				preset = p
				havePreset = true
				if req.DeviceCodeURL == "" {
					req.DeviceCodeURL = strings.TrimSpace(p.DeviceCodeURL)
				}
				if req.ClientID == "" {
					req.ClientID = strings.TrimSpace(p.DeviceClientID)
				}
				if req.Scope == "" {
					req.Scope = strings.TrimSpace(p.DeviceScope)
				}
				break
			}
		}
	}
	if req.DeviceCodeURL == "" {
		http.Error(w, "provider does not support device code retrieval", http.StatusBadRequest)
		return
	}
	if req.ClientID == "" {
		http.Error(w, "client_id is required for this provider", http.StatusBadRequest)
		return
	}
	if req.Scope == "" {
		req.Scope = "openid profile email"
	}
	providerName := strings.ToLower(strings.TrimSpace(req.Provider))
	u, err := url.Parse(req.DeviceCodeURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		http.Error(w, "invalid device_code_url", http.StatusBadRequest)
		return
	}
	body := ""
	contentType := "application/x-www-form-urlencoded"
	if providerName == "openai" || providerName == "github-copilot" {
		contentType = "application/json"
		payload := map[string]string{"client_id": req.ClientID}
		if req.Scope != "" {
			payload["scope"] = req.Scope
		}
		raw, _ := json.Marshal(payload)
		body = string(raw)
	} else {
		form := url.Values{}
		form.Set("client_id", req.ClientID)
		form.Set("scope", req.Scope)
		body = form.Encode()
	}
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, req.DeviceCodeURL, strings.NewReader(body))
	if err != nil {
		http.Error(w, "failed to build device code request", http.StatusBadRequest)
		return
	}
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":    false,
			"error": "device code request failed: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		errMsg := strings.TrimSpace(string(rawBody))
		var errBody map[string]any
		if json.Unmarshal(rawBody, &errBody) == nil {
			if desc := strings.TrimSpace(fmt.Sprintf("%v", errBody["error_description"])); desc != "" && desc != "<nil>" {
				errMsg = desc
			} else if code := strings.TrimSpace(fmt.Sprintf("%v", errBody["error"])); code != "" && code != "<nil>" {
				errMsg = code
			}
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":          false,
			"status_code": resp.StatusCode,
			"error":       errMsg,
		})
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(rawBody, &decoded); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "device code response was not valid JSON",
		})
		return
	}
	respPayload := map[string]any{
		"ok":   true,
		"data": decoded,
	}
	for _, key := range []string{"device_code", "user_code", "verification_uri", "verification_url", "verification_uri_complete", "expires_in", "interval", "message"} {
		if v, ok := decoded[key]; ok {
			respPayload[key] = v
		}
	}
	if providerName == "openai" {
		if v, ok := decoded["device_auth_id"]; ok {
			respPayload["device_auth_id"] = v
		}
		// OpenAI device flow uses user_code for display and device_auth_id for exchange.
		// Keep the UI binding URL available even if upstream omits verification_uri.
		if _, ok := decoded["verification_uri"]; !ok && havePreset && strings.TrimSpace(preset.DeviceBindingURL) != "" {
			respPayload["verification_uri"] = strings.TrimSpace(preset.DeviceBindingURL)
		}
	}
	writeJSON(w, http.StatusOK, respPayload)
}

func (h *AdminHandler) providerDeviceTokenAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider       string `json:"provider"`
		DeviceTokenURL string `json:"device_token_url"`
		ClientID       string `json:"client_id"`
		DeviceCode     string `json:"device_code"`
		DeviceAuthID   string `json:"device_auth_id"`
		GrantType      string `json:"grant_type"`
		OAuthTokenURL  string `json:"oauth_token_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Provider = strings.TrimSpace(req.Provider)
	req.DeviceTokenURL = strings.TrimSpace(req.DeviceTokenURL)
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.DeviceCode = strings.TrimSpace(req.DeviceCode)
	req.DeviceAuthID = strings.TrimSpace(req.DeviceAuthID)
	req.GrantType = strings.TrimSpace(req.GrantType)
	req.OAuthTokenURL = strings.TrimSpace(req.OAuthTokenURL)

	var preset assets.PopularProvider
	var havePreset bool
	if req.Provider != "" {
		if popular, err := getPopularProviders(); err == nil {
			for _, p := range popular {
				if p.Name != req.Provider {
					continue
				}
				preset = p
				havePreset = true
				if req.DeviceTokenURL == "" {
					req.DeviceTokenURL = strings.TrimSpace(p.DeviceTokenURL)
				}
				if req.ClientID == "" {
					req.ClientID = strings.TrimSpace(p.DeviceClientID)
				}
				if req.GrantType == "" {
					req.GrantType = strings.TrimSpace(p.DeviceGrantType)
				}
				break
			}
		}
	}
	if req.DeviceTokenURL == "" {
		http.Error(w, "provider does not support device token exchange", http.StatusBadRequest)
		return
	}
	if req.ClientID == "" {
		http.Error(w, "client_id is required for this provider", http.StatusBadRequest)
		return
	}
	if req.DeviceCode == "" {
		http.Error(w, "device_code is required", http.StatusBadRequest)
		return
	}
	if strings.EqualFold(req.Provider, "openai") && req.DeviceAuthID == "" {
		http.Error(w, "device_auth_id is required for openai headless flow", http.StatusBadRequest)
		return
	}
	if req.GrantType == "" {
		req.GrantType = "urn:ietf:params:oauth:grant-type:device_code"
	}
	providerName := strings.ToLower(strings.TrimSpace(req.Provider))

	body := ""
	contentType := "application/x-www-form-urlencoded"
	if providerName == "openai" {
		contentType = "application/json"
		payload := map[string]string{
			"device_auth_id": req.DeviceAuthID,
			"user_code":      req.DeviceCode,
		}
		raw, _ := json.Marshal(payload)
		body = string(raw)
	} else if providerName == "github-copilot" {
		contentType = "application/json"
		payload := map[string]string{
			"client_id":   req.ClientID,
			"device_code": req.DeviceCode,
			"grant_type":  req.GrantType,
		}
		raw, _ := json.Marshal(payload)
		body = string(raw)
	} else {
		form := url.Values{}
		form.Set("client_id", req.ClientID)
		form.Set("device_code", req.DeviceCode)
		form.Set("grant_type", req.GrantType)
		body = form.Encode()
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, req.DeviceTokenURL, strings.NewReader(body))
	if err != nil {
		http.Error(w, "failed to build device token request", http.StatusBadRequest)
		return
	}
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":    false,
			"error": "device token request failed: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	decoded := map[string]any{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "device token response was not valid JSON",
		})
		return
	}

	errCode := strings.TrimSpace(fmt.Sprintf("%v", decoded["error"]))
	if providerName == "openai" {
		if (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound) && (errCode == "" || errCode == "<nil>") {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":      true,
				"pending": true,
			})
			return
		}
		if errCode == "authorization_pending" || errCode == "slow_down" {
			out := map[string]any{
				"ok":      true,
				"pending": true,
				"error":   errCode,
			}
			if iv, ok := decoded["interval"]; ok {
				out["interval"] = iv
			}
			writeJSON(w, http.StatusOK, out)
			return
		}
		authCode := strings.TrimSpace(fmt.Sprintf("%v", decoded["authorization_code"]))
		codeVerifier := strings.TrimSpace(fmt.Sprintf("%v", decoded["code_verifier"]))
		if authCode != "" && authCode != "<nil>" && codeVerifier != "" && codeVerifier != "<nil>" {
			tokenURL := req.OAuthTokenURL
			if havePreset {
				if tokenURL == "" {
					tokenURL = strings.TrimSpace(preset.OAuthTokenURL)
				}
			}
			if tokenURL == "" {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"ok":    false,
					"error": "openai oauth token endpoint not configured",
				})
				return
			}
			redirectURI := "https://auth.openai.com/deviceauth/callback"
			form := url.Values{}
			form.Set("grant_type", "authorization_code")
			form.Set("code", authCode)
			form.Set("redirect_uri", redirectURI)
			form.Set("client_id", req.ClientID)
			form.Set("code_verifier", codeVerifier)
			exReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "failed to build oauth token exchange request"})
				return
			}
			exReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			exReq.Header.Set("Accept", "application/json")
			exResp, err := (&http.Client{Timeout: 15 * time.Second}).Do(exReq)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "oauth token exchange failed: " + err.Error()})
				return
			}
			defer exResp.Body.Close()
			exBody, _ := io.ReadAll(io.LimitReader(exResp.Body, 64*1024))
			var exDecoded map[string]any
			if err := json.Unmarshal(exBody, &exDecoded); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "oauth token response was not valid JSON"})
				return
			}
			if exResp.StatusCode < 200 || exResp.StatusCode > 299 {
				exErr := strings.TrimSpace(fmt.Sprintf("%v", exDecoded["error_description"]))
				if exErr == "" || exErr == "<nil>" {
					exErr = strings.TrimSpace(fmt.Sprintf("%v", exDecoded["error"]))
				}
				if exErr == "" || exErr == "<nil>" {
					exErr = strings.TrimSpace(string(exBody))
				}
				if exErr == "" {
					exErr = "oauth token exchange failed"
				}
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": exErr, "status_code": exResp.StatusCode})
				return
			}
			tok := strings.TrimSpace(fmt.Sprintf("%v", exDecoded["access_token"]))
			if tok == "" || tok == "<nil>" {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "oauth token exchange missing access_token"})
				return
			}
			out := map[string]any{"ok": true, "pending": false, "auth_token": tok}
			if rt := strings.TrimSpace(fmt.Sprintf("%v", exDecoded["refresh_token"])); rt != "" && rt != "<nil>" {
				out["refresh_token"] = rt
			}
			if exp, ok := exDecoded["expires_in"]; ok {
				out["expires_in"] = exp
			}
			if idToken := strings.TrimSpace(fmt.Sprintf("%v", exDecoded["id_token"])); idToken != "" && idToken != "<nil>" {
				if accountID := extractOpenAIAccountID(idToken); accountID != "" {
					out["account_id"] = accountID
				}
			}
			writeJSON(w, http.StatusOK, out)
			return
		}
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if tok := strings.TrimSpace(fmt.Sprintf("%v", decoded["access_token"])); tok != "" && tok != "<nil>" {
			out := map[string]any{
				"ok":         true,
				"pending":    false,
				"auth_token": tok,
			}
			if rt := strings.TrimSpace(fmt.Sprintf("%v", decoded["refresh_token"])); rt != "" && rt != "<nil>" {
				out["refresh_token"] = rt
			}
			if exp, ok := decoded["expires_in"]; ok {
				out["expires_in"] = exp
			}
			writeJSON(w, http.StatusOK, out)
			return
		}
		if errCode == "authorization_pending" || errCode == "slow_down" {
			out := map[string]any{
				"ok":      true,
				"pending": true,
				"error":   errCode,
			}
			if iv, ok := decoded["interval"]; ok {
				out["interval"] = iv
			}
			writeJSON(w, http.StatusOK, out)
			return
		}
	}

	msg := strings.TrimSpace(fmt.Sprintf("%v", decoded["error_description"]))
	if msg == "" || msg == "<nil>" {
		msg = errCode
	}
	if msg == "" || msg == "<nil>" {
		msg = strings.TrimSpace(string(b))
	}
	if msg == "" {
		msg = "device token exchange failed"
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"ok":          false,
		"status_code": resp.StatusCode,
		"error":       msg,
	})
}

func (h *AdminHandler) providerOAuthStartAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider          string `json:"provider"`
		OAuthAuthorizeURL string `json:"oauth_authorize_url"`
		OAuthTokenURL     string `json:"oauth_token_url"`
		OAuthClientID     string `json:"oauth_client_id"`
		OAuthClientSecret string `json:"oauth_client_secret"`
		OAuthScope        string `json:"oauth_scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Provider = strings.TrimSpace(req.Provider)
	req.OAuthAuthorizeURL = strings.TrimSpace(req.OAuthAuthorizeURL)
	req.OAuthTokenURL = strings.TrimSpace(req.OAuthTokenURL)
	req.OAuthClientID = strings.TrimSpace(req.OAuthClientID)
	req.OAuthClientSecret = strings.TrimSpace(req.OAuthClientSecret)
	req.OAuthScope = strings.TrimSpace(req.OAuthScope)
	if req.Provider == "" {
		http.Error(w, "provider is required", http.StatusBadRequest)
		return
	}
	popular, err := getPopularProviders()
	if err != nil {
		http.Error(w, "popular providers unavailable", http.StatusInternalServerError)
		return
	}
	var preset *assets.PopularProvider
	for i := range popular {
		if popular[i].Name == req.Provider {
			preset = &popular[i]
			break
		}
	}
	if preset == nil {
		http.Error(w, "unknown provider", http.StatusBadRequest)
		return
	}
	authorizeURL := strings.TrimSpace(preset.OAuthAuthorizeURL)
	tokenURL := strings.TrimSpace(preset.OAuthTokenURL)
	clientID := strings.TrimSpace(preset.OAuthClientID)
	clientSecret := strings.TrimSpace(preset.OAuthClientSecret)
	scope := strings.TrimSpace(preset.OAuthScope)
	baseURL := strings.TrimSpace(preset.OAuthBaseURL)
	originator := strings.TrimSpace(preset.OAuthOriginator)
	if req.OAuthAuthorizeURL != "" {
		authorizeURL = req.OAuthAuthorizeURL
	}
	if req.OAuthTokenURL != "" {
		tokenURL = req.OAuthTokenURL
	}
	if req.OAuthClientID != "" {
		clientID = req.OAuthClientID
	}
	if req.OAuthClientSecret != "" {
		clientSecret = req.OAuthClientSecret
	}
	if req.OAuthScope != "" {
		scope = req.OAuthScope
	}

	if authorizeURL == "" || tokenURL == "" || clientID == "" {
		http.Error(w, "provider oauth configuration incomplete (authorize_url, token_url, client_id required)", http.StatusBadRequest)
		return
	}
	if scope == "" {
		scope = "openid profile email offline_access"
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(preset.BaseURL)
	}
	providerName := strings.ToLower(strings.TrimSpace(preset.Name))
	redirectURI := h.externalAdminCallbackURL(r)
	if providerName == "openai" {
		openAIRedirect, err := h.ensureLoopbackOAuthCallback("http://localhost:1455/auth/callback")
		if err != nil {
			http.Error(w, "openai oauth callback listener failed on localhost:1455: "+err.Error(), http.StatusBadRequest)
			return
		}
		redirectURI = openAIRedirect
	} else if providerName == "google-gemini" {
		googleRedirect, err := h.ensureLoopbackOAuthCallback("http://127.0.0.1:1455/oauth2callback")
		if err != nil {
			http.Error(w, "google oauth callback listener failed on 127.0.0.1:1455: "+err.Error(), http.StatusBadRequest)
			return
		}
		redirectURI = googleRedirect
	}
	state, err := randomURLSafe(24)
	if err != nil {
		http.Error(w, "failed to create oauth state", http.StatusInternalServerError)
		return
	}
	verifier, err := randomURLSafe(64)
	if err != nil {
		http.Error(w, "failed to create oauth verifier", http.StatusInternalServerError)
		return
	}
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	u, err := url.Parse(authorizeURL)
	if err != nil {
		http.Error(w, "invalid oauth authorize url", http.StatusBadRequest)
		return
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", scope)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	if providerName == "openai" {
		if originator == "" {
			originator = "codex_cli_rs"
		}
		q.Set("originator", originator)
		q.Set("id_token_add_organizations", "true")
		q.Set("codex_cli_simplified_flow", "true")
	} else if providerName == "google-gemini" {
		q.Set("access_type", "offline")
		q.Set("prompt", "consent")
		q.Set("include_granted_scopes", "true")
	}
	u.RawQuery = q.Encode()

	h.oauthMu.Lock()
	h.pruneOAuthLocked(time.Now())
	h.oauthPending[state] = &oauthSession{
		Provider:     req.Provider,
		Verifier:     verifier,
		RedirectURI:  redirectURI,
		TokenURL:     tokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Originator:   originator,
		CreatedAt:    time.Now(),
		BaseURL:      baseURL,
	}
	h.oauthMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"state":    state,
		"auth_url": u.String(),
	})
}

func (h *AdminHandler) ensureLoopbackOAuthCallback(callbackURL string) (string, error) {
	h.oauthSrvMu.Lock()
	defer h.oauthSrvMu.Unlock()
	if h.oauthSrv != nil {
		return callbackURL, nil
	}
	ln, err := net.Listen("tcp", h.oauthSrvAddr)
	if err != nil {
		return "", err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", h.providerOAuthCallbackPage)
	mux.HandleFunc("/oauth2callback", h.providerOAuthCallbackPage)
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	h.oauthSrv = srv
	go func() {
		_ = srv.Serve(ln)
	}()
	return callbackURL, nil
}

func (h *AdminHandler) providerOAuthResultAPI(w http.ResponseWriter, r *http.Request) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		http.Error(w, "state is required", http.StatusBadRequest)
		return
	}
	h.oauthMu.Lock()
	defer h.oauthMu.Unlock()
	sess, ok := h.oauthPending[state]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "oauth session not found"})
		return
	}
	if !sess.Done {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pending": true})
		return
	}
	if sess.Error != "" {
		delete(h.oauthPending, state)
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": sess.Error})
		return
	}
	resp := map[string]any{
		"ok":               true,
		"pending":          false,
		"auth_token":       sess.AccessToken,
		"refresh_token":    sess.RefreshToken,
		"token_expires_at": sess.ExpiresAt,
		"account_id":       sess.AccountID,
		"base_url":         sess.BaseURL,
	}
	delete(h.oauthPending, state)
	writeJSON(w, http.StatusOK, resp)
}

func (h *AdminHandler) providerOAuthCallbackPage(w http.ResponseWriter, r *http.Request) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	errCode := strings.TrimSpace(r.URL.Query().Get("error"))
	errDesc := strings.TrimSpace(r.URL.Query().Get("error_description"))

	h.oauthMu.Lock()
	sess, ok := h.oauthPending[state]
	if !ok {
		h.oauthMu.Unlock()
		http.Error(w, "oauth session not found", http.StatusBadRequest)
		return
	}
	if errCode != "" {
		sess.Done = true
		if errDesc != "" {
			sess.Error = errCode + ": " + errDesc
		} else {
			sess.Error = errCode
		}
		h.oauthMu.Unlock()
		h.writeOAuthCallbackHTML(w, false, sess.Error)
		return
	}
	if code == "" {
		sess.Done = true
		sess.Error = "missing authorization code"
		h.oauthMu.Unlock()
		h.writeOAuthCallbackHTML(w, false, sess.Error)
		return
	}
	tokenURL := sess.TokenURL
	clientID := sess.ClientID
	clientSecret := sess.ClientSecret
	verifier := sess.Verifier
	redirectURI := sess.RedirectURI
	h.oauthMu.Unlock()

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", redirectURI)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		h.oauthMu.Lock()
		if cur := h.oauthPending[state]; cur != nil {
			cur.Done = true
			cur.Error = "failed to build token request"
		}
		h.oauthMu.Unlock()
		h.writeOAuthCallbackHTML(w, false, "failed to build token request")
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		h.oauthMu.Lock()
		if cur := h.oauthPending[state]; cur != nil {
			cur.Done = true
			cur.Error = "oauth token request failed: " + err.Error()
		}
		h.oauthMu.Unlock()
		h.writeOAuthCallbackHTML(w, false, "oauth token request failed")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		var parsed map[string]any
		if json.Unmarshal(body, &parsed) == nil {
			if v := strings.TrimSpace(fmt.Sprintf("%v", parsed["error_description"])); v != "" && v != "<nil>" {
				msg = v
			}
		}
		h.oauthMu.Lock()
		if cur := h.oauthPending[state]; cur != nil {
			cur.Done = true
			cur.Error = msg
		}
		h.oauthMu.Unlock()
		h.writeOAuthCallbackHTML(w, false, msg)
		return
	}
	var token struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &token); err != nil || token.AccessToken == "" {
		h.oauthMu.Lock()
		if cur := h.oauthPending[state]; cur != nil {
			cur.Done = true
			cur.Error = "invalid oauth token response"
		}
		h.oauthMu.Unlock()
		h.writeOAuthCallbackHTML(w, false, "invalid oauth token response")
		return
	}
	expiresAt := ""
	if token.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
	}
	accountID := extractOpenAIAccountID(token.AccessToken)
	h.oauthMu.Lock()
	if cur := h.oauthPending[state]; cur != nil {
		cur.Done = true
		cur.AccessToken = token.AccessToken
		cur.RefreshToken = token.RefreshToken
		cur.ExpiresAt = expiresAt
		cur.AccountID = accountID
	}
	h.oauthMu.Unlock()
	h.writeOAuthCallbackHTML(w, true, "")
}

func (h *AdminHandler) writeOAuthCallbackHTML(w http.ResponseWriter, ok bool, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	status := "Authentication completed. You can close this window."
	if !ok {
		status = "Authentication failed: " + msg
	}
	escapedStatus := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;").Replace(status)
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>OAuth</title></head><body><div style="font-family:system-ui;padding:20px;">` + escapedStatus + `</div><script>setTimeout(function(){window.close();},800);</script></body></html>`))
}

func (h *AdminHandler) externalAdminCallbackURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xf := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); xf != "" {
		parts := strings.Split(xf, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			scheme = strings.TrimSpace(parts[0])
		}
	}
	host := strings.TrimSpace(r.Host)
	if xfh := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); xfh != "" {
		parts := strings.Split(xfh, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			host = strings.TrimSpace(parts[0])
		}
	}
	return scheme + "://" + host + "/admin/oauth/callback"
}

func (h *AdminHandler) pruneOAuthLocked(now time.Time) {
	for state, sess := range h.oauthPending {
		if now.Sub(sess.CreatedAt) > 15*time.Minute {
			delete(h.oauthPending, state)
		}
	}
}

func randomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func extractOpenAIAccountID(accessToken string) string {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return ""
	}
	claimAny, ok := payload["https://api.openai.com/auth"]
	if !ok {
		return ""
	}
	claim, ok := claimAny.(map[string]any)
	if !ok {
		return ""
	}
	accountID := strings.TrimSpace(fmt.Sprintf("%v", claim["chatgpt_account_id"]))
	if accountID == "<nil>" {
		return ""
	}
	return accountID
}

func (h *AdminHandler) testProviderAPI(w http.ResponseWriter, r *http.Request) {
	var p config.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		p.Name = "test"
	}
	p = h.applyPresetProviderDefaults(p)
	if strings.TrimSpace(p.BaseURL) == "" {
		http.Error(w, "base_url is required", http.StatusBadRequest)
		return
	}
	models, err := NewProviderClient(p).ListModels(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"model_count": len(models),
	})
}

func (h *AdminHandler) providerByNameAPI(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		http.Error(w, "provider name required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var p config.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		p.Name = name
		err := h.store.Update(func(c *config.ServerConfig) error {
			for i := range c.Providers {
				if c.Providers[i].Name == name {
					cur := c.Providers[i]
					next := p
					next.Name = name
					if strings.TrimSpace(next.ProviderType) == "" {
						next.ProviderType = cur.ProviderType
					}
					if strings.TrimSpace(next.BaseURL) == "" {
						next.BaseURL = cur.BaseURL
					}
					if next.TimeoutSeconds <= 0 {
						next.TimeoutSeconds = cur.TimeoutSeconds
					}
					// Preserve existing credentials unless explicitly replaced.
					if strings.TrimSpace(next.APIKey) == "" {
						next.APIKey = cur.APIKey
					}
					if strings.TrimSpace(next.AuthToken) == "" {
						next.AuthToken = cur.AuthToken
					}
					if strings.TrimSpace(next.RefreshToken) == "" {
						next.RefreshToken = cur.RefreshToken
					}
					if strings.TrimSpace(next.TokenExpiresAt) == "" {
						next.TokenExpiresAt = cur.TokenExpiresAt
					}
					if strings.TrimSpace(next.AccountID) == "" {
						next.AccountID = cur.AccountID
					}
					if strings.TrimSpace(next.DeviceAuthURL) == "" {
						next.DeviceAuthURL = cur.DeviceAuthURL
					}
					if err := h.validateProviderForSave(next); err != nil {
						return err
					}
					c.Providers[i] = next
					return nil
				}
			}
			return fmt.Errorf("provider not found")
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if h.healthChecker != nil {
			h.healthChecker.Trigger()
		}
		h.notifyAdminChanged("providers")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case http.MethodDelete:
		err := h.store.Update(func(c *config.ServerConfig) error {
			next := make([]config.ProviderConfig, 0, len(c.Providers))
			found := false
			for _, p := range c.Providers {
				if p.Name == name {
					found = true
					continue
				}
				next = append(next, p)
			}
			if !found {
				return fmt.Errorf("provider not found")
			}
			c.Providers = next
			if c.DefaultProvider == name {
				c.DefaultProvider = ""
				if len(c.Providers) > 0 {
					c.DefaultProvider = c.Providers[0].Name
				}
			}
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if h.healthChecker != nil {
			h.healthChecker.Trigger()
		}
		h.notifyAdminChanged("providers")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) refreshModelsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	models, err := h.resolver.DiscoverModels(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": models})
}

func (h *AdminHandler) refreshProvidersAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	providers := h.catalogProviders()
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	if h.healthChecker != nil {
		h.healthChecker.checkOnce(ctx, true)
	}
	pricingFreshTouched := 0
	if h.pricing != nil {
		for _, p := range providers {
			pp := refreshOAuthTokenForProvider(ctx, h.store, p)
			hasPricing := probeModelsEndpointHasPricing(ctx, pp)
			if !hasPricing {
				continue
			}
			if err := h.pricing.TouchProviderFresh(pp.Name, time.Now().UTC()); err == nil {
				pricingFreshTouched++
			}
		}
	}
	if h.healthChecker != nil {
		h.healthChecker.Trigger()
	}
	h.notifyAdminChanged("providers")
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                "ok",
		"providers":             len(providers),
		"pricing_fresh_touched": pricingFreshTouched,
	})
}

func probeModelsEndpointHasPricing(ctx context.Context, p config.ProviderConfig) bool {
	base := strings.TrimSpace(p.BaseURL)
	if base == "" {
		return false
	}
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return false
	}
	u.Path = joinProviderPath(u.Path, "/v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false
	}
	applyUpstreamProviderHeaders(req, p, http.Header{})
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	cli := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return false
	}
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return false
	}
	for _, item := range payload.Data {
		if modelsJSONIncludesPricing(item) {
			return true
		}
	}
	return false
}

func modelsJSONIncludesPricing(item map[string]any) bool {
	if item == nil {
		return false
	}
	if pricing, ok := item["pricing"].(map[string]any); ok {
		for _, k := range []string{"prompt", "completion", "input", "output"} {
			if _, ok := pricing[k]; ok {
				return true
			}
		}
	}
	if _, ok := item["input_cost_per_token"]; ok {
		return true
	}
	if _, ok := item["output_cost_per_token"]; ok {
		return true
	}
	if providers, ok := item["providers"].([]any); ok {
		for _, raw := range providers {
			pm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			pricing, ok := pm["pricing"].(map[string]any)
			if !ok {
				continue
			}
			for _, k := range []string{"prompt", "completion", "input", "output"} {
				if _, ok := pricing[k]; ok {
					return true
				}
			}
		}
	}
	return false
}

func (h *AdminHandler) benchmarkStatusAPI(w http.ResponseWriter, r *http.Request) {
	h.benchmarkMu.Lock()
	defer h.benchmarkMu.Unlock()
	if h.benchmarkRun == nil {
		writeJSON(w, http.StatusOK, map[string]any{"running": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"running": h.benchmarkRun.Status == "running", "run": h.benchmarkRun})
}

func (h *AdminHandler) benchmarkStartAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChatsPerModel   int      `json:"chats_per_model"`
		MessagesPerChat int      `json:"messages_per_chat"`
		StopAfterTokens int      `json:"stop_after_tokens"`
		Models          []string `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ChatsPerModel <= 0 {
		req.ChatsPerModel = 1
	}
	if req.MessagesPerChat <= 0 {
		req.MessagesPerChat = 1
	}
	if req.MessagesPerChat > 20 {
		req.MessagesPerChat = 20
	}
	if req.ChatsPerModel > 200 {
		req.ChatsPerModel = 200
	}
	models := make([]string, 0, len(req.Models))
	seen := map[string]struct{}{}
	for _, m := range req.Models {
		x := strings.TrimSpace(m)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		models = append(models, x)
	}
	if len(models) == 0 {
		http.Error(w, "at least one model is required", http.StatusBadRequest)
		return
	}

	h.benchmarkMu.Lock()
	if h.benchmarkRun != nil && h.benchmarkRun.Status == "running" {
		h.benchmarkMu.Unlock()
		http.Error(w, "benchmark already running", http.StatusConflict)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &benchmarkRun{
		ID:        fmt.Sprintf("bench-%d", time.Now().UTC().UnixNano()),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    "running",
		Strategy: benchmarkStrategy{
			ChatsPerModel:   req.ChatsPerModel,
			MessagesPerChat: req.MessagesPerChat,
			StopAfterTokens: req.StopAfterTokens,
		},
		Models: make([]benchmarkModelProgress, 0, len(models)),
		cancel: cancel,
	}
	for _, m := range models {
		run.Models = append(run.Models, benchmarkModelProgress{
			Model:         m,
			Status:        "queued",
			ChatsTotal:    req.ChatsPerModel,
			MessagesTotal: req.ChatsPerModel * req.MessagesPerChat,
		})
	}
	h.benchmarkRun = run
	h.benchmarkMu.Unlock()

	go h.runBenchmark(ctx, run)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "run": run})
}

func (h *AdminHandler) benchmarkCancelAPI(w http.ResponseWriter, r *http.Request) {
	h.benchmarkMu.Lock()
	defer h.benchmarkMu.Unlock()
	if h.benchmarkRun == nil || h.benchmarkRun.Status != "running" {
		writeJSON(w, http.StatusOK, map[string]any{"status": "idle"})
		return
	}
	h.benchmarkRun.Status = "cancelling"
	if h.benchmarkRun.cancel != nil {
		h.benchmarkRun.cancel()
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *AdminHandler) runBenchmark(ctx context.Context, run *benchmarkRun) {
	stopAfter := run.Strategy.StopAfterTokens
	for i := range run.Models {
		select {
		case <-ctx.Done():
			h.finishBenchmark("cancelled", "")
			return
		default:
		}
		h.benchmarkMu.Lock()
		run.Models[i].Status = "running"
		h.benchmarkMu.Unlock()

		for c := 0; c < run.Strategy.ChatsPerModel; c++ {
			for m := 0; m < run.Strategy.MessagesPerChat; m++ {
				select {
				case <-ctx.Done():
					h.finishBenchmark("cancelled", "")
					return
				default:
				}
				p, ct, tt, err := h.runBenchmarkMessage(ctx, run.Models[i].Model, c, m)
				h.benchmarkMu.Lock()
				run.Models[i].MessagesDone++
				if (m + 1) == run.Strategy.MessagesPerChat {
					run.Models[i].ChatsDone++
				}
				run.Models[i].PromptTokens += p
				run.Models[i].CompletionTokens += ct
				run.Models[i].TotalTokens += tt
				run.TotalPrompt += p
				run.TotalCompletion += ct
				run.TotalTokens += tt
				if err != nil {
					run.Models[i].Status = "error"
					run.Models[i].LastError = err.Error()
				}
				h.benchmarkMu.Unlock()
				h.notifyAdminChanged("benchmark")
				if stopAfter > 0 {
					h.benchmarkMu.Lock()
					stop := run.TotalTokens >= stopAfter
					h.benchmarkMu.Unlock()
					if stop {
						h.finishBenchmark("completed", "")
						return
					}
				}
			}
		}
		h.benchmarkMu.Lock()
		if run.Models[i].Status == "running" {
			run.Models[i].Status = "done"
		}
		h.benchmarkMu.Unlock()
		h.notifyAdminChanged("benchmark")
	}
	h.finishBenchmark("completed", "")
}

func (h *AdminHandler) finishBenchmark(status, errMsg string) {
	h.benchmarkMu.Lock()
	defer h.benchmarkMu.Unlock()
	if h.benchmarkRun == nil {
		return
	}
	h.benchmarkRun.Status = strings.TrimSpace(status)
	h.benchmarkRun.Error = strings.TrimSpace(errMsg)
	if h.benchmarkRun.cancel != nil {
		h.benchmarkRun.cancel()
		h.benchmarkRun.cancel = nil
	}
	h.notifyAdminChanged("benchmark")
}

func (h *AdminHandler) runBenchmarkMessage(ctx context.Context, model string, chatIdx, msgIdx int) (int, int, int, error) {
	provider, upstreamModel, err := h.resolver.Resolve(model)
	if err != nil {
		return 0, 0, 0, err
	}
	provider = refreshOAuthTokenForProvider(ctx, h.store, provider)
	payload := map[string]any{
		"model":       upstreamModel,
		"temperature": 0.2,
		"messages": []map[string]string{
			{"role": "system", "content": "Be concise."},
			{"role": "user", "content": fmt.Sprintf("Benchmark chat %d message %d. Reply with one short sentence.", chatIdx+1, msgIdx+1)},
		},
	}
	body, _ := json.Marshal(payload)
	u, err := url.Parse(strings.TrimRight(provider.BaseURL, "/"))
	if err != nil {
		return 0, 0, 0, err
	}
	u.Path = joinProviderPath(u.Path, "/v1/chat/completions")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return 0, 0, 0, err
	}
	applyUpstreamProviderHeaders(req, provider, http.Header{})
	cli := &http.Client{Timeout: time.Duration(max(10, provider.TimeoutSeconds)) * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return 0, 0, 0, fmt.Errorf("status %d: %s", resp.StatusCode, msg)
	}
	p, c, t := parseUsageTokens(respBody)
	return p, c, t, nil
}

func (h *AdminHandler) modelsCatalogAPI(w http.ResponseWriter, r *http.Request) {
	providers := h.catalogProviders()
	cfg := h.store.Snapshot()
	configuredByName := make(map[string]struct{}, len(cfg.Providers))
	for _, p := range cfg.Providers {
		configuredByName[p.Name] = struct{}{}
	}
	quotaSnapshots := h.readProviderQuotas(r.Context(), providers, false)
	popularByType := map[string]assets.PopularProvider{}
	if popular, err := getPopularProviders(); err == nil {
		for _, p := range popular {
			popularByType[p.Name] = p
		}
	}
	cache := pricing.Cache{}
	if h.pricing != nil {
		h.pricing.SetProviders(providers)
		if r.URL.Query().Get("refresh") == "1" {
			ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
			defer cancel()
			_ = h.pricing.Refresh(ctx)
		} else {
			h.pricing.EnsureFreshAsync()
		}
		cache = h.pricing.Snapshot()
	}
	type row struct {
		Provider            string   `json:"provider"`
		ProviderType        string   `json:"provider_type,omitempty"`
		ProviderDisplayName string   `json:"provider_display_name"`
		Model               string   `json:"model"`
		Status              string   `json:"status"`
		ResponseMS          int64    `json:"response_ms,omitempty"`
		CheckedAt           string   `json:"checked_at,omitempty"`
		InputPer1M          *float64 `json:"input_per_1m,omitempty"`
		OutputPer1M         *float64 `json:"output_per_1m,omitempty"`
		Currency            string   `json:"currency,omitempty"`
		PerfRequests        int      `json:"perf_requests,omitempty"`
		PromptTokens        int      `json:"prompt_tokens,omitempty"`
		CompletionTokens    int      `json:"completion_tokens,omitempty"`
		FailureRate         float64  `json:"failure_rate,omitempty"`
		AvgPromptTPS        float64  `json:"avg_prompt_tps,omitempty"`
		AvgGenerationTPS    float64  `json:"avg_generation_tps,omitempty"`
	}
	period := time.Hour
	if raw := strings.TrimSpace(r.URL.Query().Get("period_seconds")); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			period = time.Duration(sec) * time.Second
		}
	}
	type usageAgg struct {
		requests int
		failed   int
		prompt   int
		gen      int
		ppSum    float64
		tgSum    float64
	}
	usageKey := func(providerName, modelName string) string {
		return strings.TrimSpace(providerName) + "\x00" + strings.TrimSpace(modelName)
	}
	usageByModel := map[string]usageAgg{}
	if h.stats != nil {
		summary := h.stats.Summary(period)
		for _, b := range summary.Buckets {
			p := strings.TrimSpace(b.Provider)
			m := strings.TrimSpace(b.Model)
			if p == "" || m == "" || b.Requests <= 0 {
				continue
			}
			key := usageKey(p, m)
			a := usageByModel[key]
			a.requests += b.Requests
			a.failed += b.FailedRequests
			a.prompt += b.PromptTokens
			a.gen += b.CompletionTokens
			a.ppSum += b.PromptTPSSum
			a.tgSum += b.GenerationTPSSum
			usageByModel[key] = a
		}
	}
	displayNames := map[string]string{}
	if popular, err := getPopularProviders(); err == nil {
		for _, p := range popular {
			if strings.TrimSpace(p.DisplayName) != "" {
				displayNames[p.Name] = p.DisplayName
			}
		}
	}
	out := make([]row, 0)
	seenModel := map[string]struct{}{}
	providerStatus := map[string]string{}
	providerTypeByName := map[string]string{}
	for _, p := range providers {
		providerTypeByName[p.Name] = providerTypeOrName(p)
	}
	for _, p := range providers {
		providerType := providerTypeOrName(p)
		_, managed := configuredByName[p.Name]
		status := "online"
		cards, err := NewProviderClient(p).ListModels(r.Context())
		if err != nil {
			status = "offline"
			if IsProviderAuthError(err) {
				status = "auth problem"
			}
		}
		if !managed && strings.EqualFold(strings.TrimSpace(status), "offline") {
			// Auto-merged providers should be silent when offline.
			continue
		}
		providerStatus[p.Name] = status
		var providerResponseMS int64
		var providerCheckedAt string
		if h.healthChecker != nil {
			if snap, ok := h.healthChecker.Snapshot(p.Name); ok {
				providerResponseMS = snap.ResponseMS
				providerCheckedAt = snap.CheckedAt.Format(time.RFC3339)
			}
		}
		for _, m := range cards {
			provider := strings.TrimSpace(m.Provider)
			if provider == "" {
				provider = strings.TrimSpace(p.Name)
			}
			modelID := strings.TrimSpace(m.ID)
			key := usageKey(provider, modelID)
			seenModel[key] = struct{}{}
			entry, ok := cache.Entries[provider+"/"+modelID]
			item := row{
				Provider:            provider,
				ProviderType:        providerTypeByName[provider],
				ProviderDisplayName: provider,
				Model:               modelID,
				Status:              status,
				ResponseMS:          providerResponseMS,
				CheckedAt:           providerCheckedAt,
			}
			if item.ProviderType == "" {
				item.ProviderType = provider
			}
			if provider == item.ProviderType {
				if dn, ok := displayNames[item.ProviderType]; ok {
					item.ProviderDisplayName = dn
				}
			} else if dn, ok := displayNames[provider]; ok {
				item.ProviderDisplayName = dn
			}
			if ok {
				item.Currency = entry.Currency
				in := entry.InputPer1M
				outp := entry.OutputPer1M
				item.InputPer1M = &in
				item.OutputPer1M = &outp
			}
			if quotaModelIsIncluded(provider, item.ProviderType, modelID, quotaSnapshots, popularByType) {
				in := 0.0
				outp := 0.0
				item.InputPer1M = &in
				item.OutputPer1M = &outp
				if item.Currency == "" {
					item.Currency = "USD"
				}
			}
			if agg, ok := usageByModel[usageKey(provider, modelID)]; ok && agg.requests > 0 {
				item.PerfRequests = agg.requests
				item.PromptTokens = agg.prompt
				item.CompletionTokens = agg.gen
				item.FailureRate = float64(agg.failed) * 100.0 / float64(agg.requests)
				item.AvgPromptTPS = agg.ppSum / float64(agg.requests)
				item.AvgGenerationTPS = agg.tgSum / float64(agg.requests)
			}
			out = append(out, item)
		}
		if len(cards) == 0 {
			item := row{
				Provider:            p.Name,
				ProviderType:        providerType,
				ProviderDisplayName: p.Name,
				Model:               "",
				Status:              status,
				ResponseMS:          providerResponseMS,
				CheckedAt:           providerCheckedAt,
			}
			if p.Name == providerType {
				if dn, ok := displayNames[providerType]; ok {
					item.ProviderDisplayName = dn
				}
			} else if dn, ok := displayNames[p.Name]; ok {
				item.ProviderDisplayName = dn
			}
			out = append(out, item)
		}
	}
	for key, entry := range cache.Entries {
		pn, modelID, ok := splitModelPrefix(key)
		if !ok {
			continue
		}
		if _, exists := seenModel[usageKey(pn, modelID)]; exists {
			continue
		}
		status := providerStatus[pn]
		if status == "" {
			status = "cached"
		}
		if _, managed := configuredByName[pn]; !managed && strings.EqualFold(strings.TrimSpace(status), "offline") {
			continue
		}
		item := row{
			Provider:            pn,
			ProviderType:        providerTypeByName[pn],
			ProviderDisplayName: pn,
			Model:               modelID,
			Status:              status,
		}
		if h.healthChecker != nil {
			if snap, ok := h.healthChecker.Snapshot(pn); ok {
				item.ResponseMS = snap.ResponseMS
				item.CheckedAt = snap.CheckedAt.Format(time.RFC3339)
			}
		}
		if item.ProviderType == "" {
			item.ProviderType = pn
		}
		if pn == item.ProviderType {
			if dn, ok := displayNames[item.ProviderType]; ok {
				item.ProviderDisplayName = dn
			}
		} else if dn, ok := displayNames[pn]; ok {
			item.ProviderDisplayName = dn
		}
		in := entry.InputPer1M
		outp := entry.OutputPer1M
		item.InputPer1M = &in
		item.OutputPer1M = &outp
		item.Currency = entry.Currency
		if quotaModelIsIncluded(pn, item.ProviderType, modelID, quotaSnapshots, popularByType) {
			in2 := 0.0
			out2 := 0.0
			item.InputPer1M = &in2
			item.OutputPer1M = &out2
			if item.Currency == "" {
				item.Currency = "USD"
			}
		}
		if agg, ok := usageByModel[usageKey(pn, modelID)]; ok && agg.requests > 0 {
			item.PerfRequests = agg.requests
			item.PromptTokens = agg.prompt
			item.CompletionTokens = agg.gen
			item.FailureRate = float64(agg.failed) * 100.0 / float64(agg.requests)
			item.AvgPromptTPS = agg.ppSum / float64(agg.requests)
			item.AvgGenerationTPS = agg.tgSum / float64(agg.requests)
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":                     out,
		"fetched_at":               time.Now().Format(time.RFC3339),
		"pricing_cache_updated_at": cache.UpdatedAt.Format(time.RFC3339),
	})
}

func quotaModelIsIncluded(providerName, providerType, modelID string, quota map[string]ProviderQuotaSnapshot, popular map[string]assets.PopularProvider) bool {
	snap, ok := quota[providerName]
	if !ok || strings.TrimSpace(strings.ToLower(snap.Status)) != "ok" {
		return false
	}
	preset, hasPreset := popular[strings.TrimSpace(providerType)]
	normModel := strings.ToLower(strings.TrimSpace(normalizeModelID(modelID)))
	if normModel == "" {
		return false
	}
	// If configured, treat model-specific quota buckets as included models.
	if hasPreset && preset.QuotaIncludedByMetric {
		ignore := make(map[string]struct{}, len(preset.QuotaMetricFeatureIgnore))
		for _, v := range preset.QuotaMetricFeatureIgnore {
			key := strings.ToLower(strings.TrimSpace(v))
			if key == "" {
				continue
			}
			ignore[key] = struct{}{}
		}
		for _, m := range snap.Metrics {
			feature := strings.ToLower(strings.TrimSpace(normalizeModelID(m.MeteredFeature)))
			if feature == "" {
				continue
			}
			if _, blocked := ignore[feature]; blocked {
				continue
			}
			if feature == normModel {
				return true
			}
		}
	}
	if !hasPreset || len(preset.QuotaFreeByPlan) == 0 {
		return false
	}
	plan := strings.ToLower(strings.TrimSpace(snap.PlanType))
	patterns := make([]string, 0)
	if any, ok := preset.QuotaFreeByPlan["*"]; ok {
		patterns = append(patterns, any...)
	}
	if plan != "" {
		if exact, ok := preset.QuotaFreeByPlan[plan]; ok {
			patterns = append(patterns, exact...)
		}
	}
	for _, p := range patterns {
		pat := strings.ToLower(strings.TrimSpace(p))
		if pat == "" {
			continue
		}
		if strings.Contains(pat, "*") {
			if matched, _ := path.Match(pat, normModel); matched {
				return true
			}
			continue
		}
		if pat == normModel {
			return true
		}
	}
	return false
}

func (h *AdminHandler) catalogProviders() []config.ProviderConfig {
	return h.activeProviders()
}

func (h *AdminHandler) quotaProviders() []config.ProviderConfig {
	return h.activeProviders()
}

func (h *AdminHandler) activeProviders() []config.ProviderConfig {
	if h == nil || h.resolver == nil {
		return nil
	}
	active := h.resolver.ListProviders()
	out := make([]config.ProviderConfig, 0, len(active))
	seen := map[string]struct{}{}
	for _, p := range active {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, p)
	}
	return out
}

func (h *AdminHandler) applyPresetProviderDefaults(p config.ProviderConfig) config.ProviderConfig {
	p.Name = strings.TrimSpace(p.Name)
	p.ProviderType = strings.TrimSpace(p.ProviderType)
	if p.ProviderType == "" {
		p.ProviderType = p.Name
	}
	p.BaseURL = strings.TrimSpace(p.BaseURL)
	p.AuthToken = strings.TrimSpace(p.AuthToken)
	p.APIKey = strings.TrimSpace(p.APIKey)
	p.DeviceAuthURL = strings.TrimSpace(p.DeviceAuthURL)
	if p.TimeoutSeconds > 0 && p.BaseURL != "" {
		return p
	}
	popular, err := getPopularProviders()
	if err != nil {
		if p.TimeoutSeconds <= 0 {
			p.TimeoutSeconds = 60
		}
		return p
	}
	presetKey := providerTypeOrName(p)
	for _, pr := range popular {
		if pr.Name != presetKey {
			continue
		}
		p = resolveProviderWithDefaults(p, pr.AsProviderConfig())
		if strings.TrimSpace(p.AuthToken) != "" && strings.TrimSpace(p.BaseURL) == "" && strings.TrimSpace(pr.OAuthBaseURL) != "" {
			p.BaseURL = strings.TrimSpace(pr.OAuthBaseURL)
		}
		if p.DeviceAuthURL == "" {
			p.DeviceAuthURL = strings.TrimSpace(pr.DeviceBindingURL)
		}
		return p
	}
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = 60
	}
	return p
}

func (h *AdminHandler) validateProviderForSave(p config.ProviderConfig) error {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return fmt.Errorf("provider name required")
	}
	p.ProviderType = strings.TrimSpace(p.ProviderType)
	if p.ProviderType == "" {
		p.ProviderType = name
	}
	if strings.TrimSpace(p.BaseURL) != "" {
		return nil
	}
	presetKey := providerTypeOrName(p)
	popular, err := getPopularProviders()
	if err == nil {
		for _, pr := range popular {
			if pr.Name == presetKey {
				if strings.TrimSpace(pr.BaseURLTemplate) != "" {
					base := strings.TrimSpace(p.BaseURL)
					if base == "" {
						return fmt.Errorf("base_url is required for %s", presetKey)
					}
					if strings.Contains(base, "{") || strings.Contains(base, "}") {
						return fmt.Errorf("base_url still contains template placeholders")
					}
				}
				return nil
			}
		}
	}
	return fmt.Errorf("base_url is required for custom providers")
}

func (h *AdminHandler) pricingAPI(w http.ResponseWriter, r *http.Request) {
	if h.pricing == nil {
		http.Error(w, "pricing manager unavailable", http.StatusInternalServerError)
		return
	}
	h.pricing.SetProviders(h.catalogProviders())
	h.pricing.EnsureFreshAsync()
	cache := h.pricing.Snapshot()
	provider := strings.TrimSpace(r.URL.Query().Get("provider"))
	if provider == "" {
		writeJSON(w, http.StatusOK, cache)
		return
	}
	filtered := pricing.Cache{
		UpdatedAt:      cache.UpdatedAt,
		ProviderStates: map[string]pricing.ProviderState{},
		Entries:        map[string]pricing.ModelPricing{},
	}
	if st, ok := cache.ProviderStates[provider]; ok {
		filtered.ProviderStates[provider] = st
	}
	for k, v := range cache.Entries {
		if strings.HasPrefix(k, provider+"/") {
			filtered.Entries[k] = v
		}
	}
	writeJSON(w, http.StatusOK, filtered)
}

func (h *AdminHandler) refreshPricingAPI(w http.ResponseWriter, r *http.Request) {
	if h.pricing == nil {
		http.Error(w, "pricing manager unavailable", http.StatusInternalServerError)
		return
	}
	h.pricing.SetProviders(h.catalogProviders())
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	if err := h.pricing.ForceRefresh(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "cache": h.pricing.Snapshot()})
}
