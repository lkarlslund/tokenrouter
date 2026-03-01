package config

import (
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	defaultConfigFileName = "torod.toml"

	TokenRoleAdmin     = "admin"
	TokenRoleKeymaster = "keymaster"
	TokenRoleInferrer  = "inferrer"
)

type ProviderConfig struct {
	Name           string `toml:"name" json:"name"`
	ProviderType   string `toml:"provider_type,omitempty" json:"provider_type,omitempty"`
	BaseURL        string `toml:"base_url,omitempty" json:"base_url"`
	ModelListURL   string `toml:"model_list_url,omitempty" json:"model_list_url,omitempty"`
	APIKey         string `toml:"api_key,omitempty" json:"api_key,omitempty"`
	AuthToken      string `toml:"auth_token,omitempty" json:"auth_token,omitempty"`
	RefreshToken   string `toml:"refresh_token,omitempty" json:"refresh_token,omitempty"`
	TokenExpiresAt string `toml:"token_expires_at,omitempty" json:"token_expires_at,omitempty"`
	AccountID      string `toml:"account_id,omitempty" json:"account_id,omitempty"`
	DeviceAuthURL  string `toml:"device_auth_url,omitempty" json:"device_auth_url,omitempty"`
	Enabled        bool   `toml:"enabled,omitempty" json:"enabled,omitempty"`
	TimeoutSeconds int    `toml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
}

type TLSConfig struct {
	Enabled    bool   `toml:"enabled"`
	Mode       string `toml:"mode"`
	ListenAddr string `toml:"listen_addr"`
	Domain     string `toml:"domain"`
	Email      string `toml:"email"`
	CacheDir   string `toml:"cache_dir"`
	CertPEM    string `toml:"cert_pem,omitempty"`
	KeyPEM     string `toml:"key_pem,omitempty"`
}

type ConversationsConfig struct {
	Enabled    bool `toml:"enabled"`
	MaxItems   int  `toml:"max_items,omitempty"`
	MaxAgeDays int  `toml:"max_age_days,omitempty"`
}

type LogsConfig struct {
	MaxLines int `toml:"max_lines,omitempty"`
}

type TokenQuotaBudget struct {
	Limit           int64  `toml:"limit,omitempty" json:"limit,omitempty"`
	IntervalSeconds int64  `toml:"interval_seconds,omitempty" json:"interval_seconds,omitempty"`
	Used            int64  `toml:"used,omitempty" json:"used,omitempty"`
	WindowStartedAt string `toml:"window_started_at,omitempty" json:"window_started_at,omitempty"`
}

type TokenQuota struct {
	Requests *TokenQuotaBudget `toml:"requests,omitempty" json:"requests,omitempty"`
	Tokens   *TokenQuotaBudget `toml:"tokens,omitempty" json:"tokens,omitempty"`
}

type IncomingAPIToken struct {
	ID        string      `toml:"id"`
	Name      string      `toml:"name"`
	Role      string      `toml:"role,omitempty"`
	ParentID  string      `toml:"parent_id,omitempty"`
	Comment   string      `toml:"comment,omitempty"`
	Key       string      `toml:"key"`
	ExpiresAt string      `toml:"expires_at,omitempty"`
	CreatedAt string      `toml:"created_at,omitempty"`
	Quota     *TokenQuota `toml:"quota,omitempty" json:"quota,omitempty"`
}

type ServerConfig struct {
	ListenAddr                    string              `toml:"listen_addr"`
	HTTPMode                      string              `toml:"http_mode"`
	IncomingTokens                []IncomingAPIToken  `toml:"incoming_tokens"`
	AllowLocalhostNoAuth          bool                `toml:"allow_localhost_no_auth"`
	AllowHostDockerInternalNoAuth bool                `toml:"allow_host_docker_internal_no_auth"`
	AutoEnablePublicFreeModels    bool                `toml:"auto_enable_public_free_models"`
	AutoDetectLocalServers        bool                `toml:"auto_detect_local_servers"`
	AutoRemoveExpiredTokens       bool                `toml:"auto_remove_expired_tokens"`
	AutoRemoveEmptyQuotaTokens    bool                `toml:"auto_remove_empty_quota_tokens"`
	DefaultProvider               string              `toml:"default_provider"`
	Providers                     []ProviderConfig    `toml:"providers"`
	Conversations                 ConversationsConfig `toml:"conversations"`
	Logs                          LogsConfig          `toml:"logs"`
	TLS                           TLSConfig           `toml:"tls"`
}

type ClientConfig struct {
	ServerURL string `toml:"server_url"`
	APIKey    string `toml:"api_key,omitempty"`
}

func DefaultServerConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultConfigFileName
	}
	return filepath.Join(home, ".config", "tokenrouter", defaultConfigFileName)
}

func DefaultClientConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "toro.toml"
	}
	return filepath.Join(home, ".config", "tokenrouter", "toro.toml")
}

func DefaultPricingCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "pricing-cache.json"
	}
	return filepath.Join(home, ".cache", "tokenrouter", "pricing-cache.json")
}

func DefaultUsageStatsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "usage-stats.json"
	}
	return filepath.Join(home, ".cache", "tokenrouter", "usage-stats.json")
}

func DefaultModelsCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "models-cache.json"
	}
	return filepath.Join(home, ".cache", "tokenrouter", "models-cache.json")
}

func DefaultConversationsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "conversations.json"
	}
	return filepath.Join(home, ".cache", "tokenrouter", "conversations.json")
}

func DefaultLogsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "logs.json"
	}
	return filepath.Join(home, ".cache", "tokenrouter", "logs.json")
}

func DefaultTLSCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "tls-autocert"
	}
	return filepath.Join(home, ".cache", "tokenrouter", "tls-autocert")
}

func NewDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ListenAddr:                 "127.0.0.1:7050",
		HTTPMode:                   "enabled",
		IncomingTokens:             []IncomingAPIToken{},
		AllowLocalhostNoAuth:       true,
		DefaultProvider:            "",
		Providers:                  []ProviderConfig{},
		AutoEnablePublicFreeModels: true,
		AutoDetectLocalServers:     true,
		AutoRemoveExpiredTokens:    true,
		AutoRemoveEmptyQuotaTokens: false,
		Conversations: ConversationsConfig{
			Enabled:    true,
			MaxItems:   5000,
			MaxAgeDays: 30,
		},
		Logs: LogsConfig{
			MaxLines: 5000,
		},
		TLS: TLSConfig{
			Enabled:    false,
			Mode:       "letsencrypt",
			ListenAddr: ":443",
			Domain:     "",
			Email:      "",
			CacheDir:   DefaultTLSCacheDir(),
		},
	}
}

func HasAdminToken(tokens []IncomingAPIToken) bool {
	now := time.Now().UTC()
	for _, t := range tokens {
		if NormalizeIncomingTokenRole(t.Role) != TokenRoleAdmin {
			continue
		}
		if strings.TrimSpace(t.Key) == "" {
			continue
		}
		if exp := strings.TrimSpace(t.ExpiresAt); exp != "" {
			ts, err := time.Parse(time.RFC3339, exp)
			if err != nil || !now.Before(ts) {
				continue
			}
		}
		return true
	}
	return false
}

func NewDefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerURL: "http://127.0.0.1:7050/v1",
	}
}

func LoadClientConfig(path string) (*ClientConfig, error) {
	cfg := NewDefaultClientConfig()
	if err := load(path, cfg); err != nil {
		return nil, err
	}
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadOrCreateClientConfig(path string) (*ClientConfig, error) {
	cfg := NewDefaultClientConfig()
	if err := loadOrCreate(path, cfg); err != nil {
		return nil, err
	}
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadServerConfig(path string) (*ServerConfig, error) {
	cfg := NewDefaultServerConfig()
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := unmarshalServerConfigTOML(b, cfg); err != nil {
		return nil, err
	}
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadOrCreateServerConfig(path string) (*ServerConfig, error) {
	cfg := NewDefaultServerConfig()
	if err := loadOrCreate(path, cfg); err != nil {
		return nil, err
	}
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadOrCreate(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := writeAtomic(path, v); err != nil {
			return fmt.Errorf("write default config: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := toml.Unmarshal(b, v); err != nil {
		return fmt.Errorf("parse toml: %w", err)
	}
	return nil
}

func load(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := toml.Unmarshal(b, v); err != nil {
		return fmt.Errorf("parse toml: %w", err)
	}
	return nil
}

func unmarshalServerConfigTOML(b []byte, cfg *ServerConfig) error {
	var top map[string]any
	_ = toml.Unmarshal(b, &top)
	_, hasAutoDetectLocalServers := top["auto_detect_local_servers"]

	type legacyServerConfig struct {
		ServerConfig
		IncomingAPIKeys []string `toml:"incoming_api_keys"`
		AdminAPIKey     string   `toml:"admin_api_key"`
	}
	var raw legacyServerConfig
	if err := toml.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("parse toml: %w", err)
	}
	*cfg = raw.ServerConfig
	if !hasAutoDetectLocalServers {
		cfg.AutoDetectLocalServers = true
	}
	if len(cfg.IncomingTokens) == 0 && len(raw.IncomingAPIKeys) > 0 {
		cfg.IncomingTokens = make([]IncomingAPIToken, 0, len(raw.IncomingAPIKeys))
		for i, k := range raw.IncomingAPIKeys {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			cfg.IncomingTokens = append(cfg.IncomingTokens, IncomingAPIToken{
				ID:   tokenID(k, i),
				Name: fmt.Sprintf("Token %d", len(cfg.IncomingTokens)+1),
				Role: TokenRoleInferrer,
				Key:  k,
			})
		}
	}
	legacyAdminKey := strings.TrimSpace(raw.AdminAPIKey)
	if legacyAdminKey != "" {
		matched := false
		for i := range cfg.IncomingTokens {
			if strings.TrimSpace(cfg.IncomingTokens[i].Key) != legacyAdminKey {
				continue
			}
			cfg.IncomingTokens[i].Role = TokenRoleAdmin
			if strings.TrimSpace(cfg.IncomingTokens[i].Name) == "" {
				cfg.IncomingTokens[i].Name = "Admin"
			}
			matched = true
		}
		if !matched {
			cfg.IncomingTokens = append(cfg.IncomingTokens, IncomingAPIToken{
				ID:   tokenID(legacyAdminKey, len(cfg.IncomingTokens)),
				Name: "Admin",
				Role: TokenRoleAdmin,
				Key:  legacyAdminKey,
			})
		}
	}
	return nil
}

func Save(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return writeAtomic(path, v)
}

func writeAtomic(path string, v any) error {
	b, err := marshalTOML(v)
	if err != nil {
		return fmt.Errorf("encode toml: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func marshalTOML(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetArraysMultiline(true)
	enc.SetIndentSymbol("  ")
	enc.SetIndentTables(true)
	enc.SetTablesInline(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return out, nil
}

func (c *ServerConfig) Normalize() {
	if c.ListenAddr == "" {
		c.ListenAddr = ":7050"
	}
	c.HTTPMode = strings.ToLower(strings.TrimSpace(c.HTTPMode))
	if c.HTTPMode == "" {
		c.HTTPMode = "enabled"
	}
	if c.HTTPMode != "enabled" && c.HTTPMode != "when_required" && c.HTTPMode != "disabled" {
		c.HTTPMode = "enabled"
	}
	c.TLS.Mode = strings.ToLower(strings.TrimSpace(c.TLS.Mode))
	if c.TLS.Mode == "" {
		c.TLS.Mode = "letsencrypt"
	}
	if c.TLS.Mode != "letsencrypt" && c.TLS.Mode != "self_signed" && c.TLS.Mode != "pem" {
		c.TLS.Mode = "letsencrypt"
	}
	c.TLS.ListenAddr = strings.TrimSpace(c.TLS.ListenAddr)
	if c.TLS.ListenAddr == "" {
		c.TLS.ListenAddr = ":443"
	}
	c.TLS.Domain = strings.TrimSpace(c.TLS.Domain)
	c.TLS.Email = strings.TrimSpace(c.TLS.Email)
	c.TLS.CacheDir = strings.TrimSpace(c.TLS.CacheDir)
	c.TLS.CertPEM = strings.TrimSpace(c.TLS.CertPEM)
	c.TLS.KeyPEM = strings.TrimSpace(c.TLS.KeyPEM)
	if c.TLS.CacheDir == "" {
		c.TLS.CacheDir = DefaultTLSCacheDir()
	}
	if c.Conversations.MaxItems <= 0 {
		c.Conversations.MaxItems = 5000
	}
	if c.Conversations.MaxAgeDays <= 0 {
		c.Conversations.MaxAgeDays = 30
	}
	if c.Logs.MaxLines <= 0 {
		c.Logs.MaxLines = 5000
	}
	tokenSeen := map[string]struct{}{}
	tokens := make([]IncomingAPIToken, 0, len(c.IncomingTokens))
	for i, t := range c.IncomingTokens {
		t.ID = strings.TrimSpace(t.ID)
		t.Name = strings.TrimSpace(t.Name)
		t.Role = NormalizeIncomingTokenRole(t.Role)
		t.ParentID = strings.TrimSpace(t.ParentID)
		t.Comment = strings.TrimSpace(t.Comment)
		t.Key = strings.TrimSpace(t.Key)
		t.ExpiresAt = strings.TrimSpace(t.ExpiresAt)
		t.CreatedAt = strings.TrimSpace(t.CreatedAt)
		t.Quota = normalizeTokenQuota(t.Quota)
		if t.Key == "" {
			continue
		}
		if _, ok := tokenSeen[t.Key]; ok {
			continue
		}
		tokenSeen[t.Key] = struct{}{}
		if t.ID == "" {
			t.ID = tokenID(t.Key, i)
		}
		if t.Name == "" {
			t.Name = fmt.Sprintf("Token %d", len(tokens)+1)
		}
		tokens = append(tokens, t)
	}
	c.IncomingTokens = tokens
	for i := range c.Providers {
		c.Providers[i].Name = strings.TrimSpace(c.Providers[i].Name)
		c.Providers[i].ProviderType = strings.TrimSpace(c.Providers[i].ProviderType)
		c.Providers[i].BaseURL = strings.TrimSpace(c.Providers[i].BaseURL)
		c.Providers[i].APIKey = strings.TrimSpace(c.Providers[i].APIKey)
		c.Providers[i].AuthToken = strings.TrimSpace(c.Providers[i].AuthToken)
		c.Providers[i].RefreshToken = strings.TrimSpace(c.Providers[i].RefreshToken)
		c.Providers[i].TokenExpiresAt = strings.TrimSpace(c.Providers[i].TokenExpiresAt)
		c.Providers[i].AccountID = strings.TrimSpace(c.Providers[i].AccountID)
		c.Providers[i].DeviceAuthURL = strings.TrimSpace(c.Providers[i].DeviceAuthURL)
		if c.Providers[i].ProviderType == "" {
			c.Providers[i].ProviderType = c.Providers[i].Name
		}
	}

	sort.SliceStable(c.Providers, func(i, j int) bool { return c.Providers[i].Name < c.Providers[j].Name })
}

func (c *ServerConfig) Validate() error {
	idSeen := map[string]struct{}{}
	for _, t := range c.IncomingTokens {
		if t.ID == "" {
			return errors.New("incoming token id cannot be empty")
		}
		if _, ok := idSeen[t.ID]; ok {
			return fmt.Errorf("duplicate incoming token id %q", t.ID)
		}
		idSeen[t.ID] = struct{}{}
		if t.Name == "" {
			return fmt.Errorf("incoming token %q name cannot be empty", t.ID)
		}
		t.Role = NormalizeIncomingTokenRole(t.Role)
		if t.Role == "" {
			return fmt.Errorf("incoming token %q has invalid role", t.ID)
		}
		if t.Key == "" {
			return fmt.Errorf("incoming token %q key cannot be empty", t.ID)
		}
		if t.ExpiresAt != "" {
			if _, err := time.Parse(time.RFC3339, t.ExpiresAt); err != nil {
				return fmt.Errorf("incoming token %q has invalid expires_at (RFC3339 required)", t.ID)
			}
		}
		if t.CreatedAt != "" {
			if _, err := time.Parse(time.RFC3339, t.CreatedAt); err != nil {
				return fmt.Errorf("incoming token %q has invalid created_at (RFC3339 required)", t.ID)
			}
		}
		if t.ParentID != "" && t.Quota != nil {
			return fmt.Errorf("incoming token %q with parent_id cannot define quota", t.ID)
		}
		if t.Quota != nil {
			if err := validateTokenQuota(t.ID, t.Quota); err != nil {
				return err
			}
		}
	}
	if c.TLS.Enabled {
		switch c.TLS.Mode {
		case "letsencrypt":
			if c.TLS.Domain == "" {
				return errors.New("tls.domain is required when tls.enabled=true and tls.mode=letsencrypt")
			}
		case "pem":
			if c.TLS.CertPEM == "" || c.TLS.KeyPEM == "" {
				return errors.New("tls.cert_pem and tls.key_pem are required when tls.enabled=true and tls.mode=pem")
			}
		case "self_signed":
		default:
			return errors.New("tls.mode must be one of letsencrypt, self_signed, pem")
		}
	}
	if c.Conversations.MaxItems < 100 {
		return errors.New("conversations.max_items must be >= 100")
	}
	if c.Conversations.MaxItems > 200000 {
		return errors.New("conversations.max_items must be <= 200000")
	}
	if c.Conversations.MaxAgeDays < 1 {
		return errors.New("conversations.max_age_days must be >= 1")
	}
	if c.Logs.MaxLines < 100 {
		return errors.New("logs.max_lines must be >= 100")
	}
	if c.Logs.MaxLines > 200000 {
		return errors.New("logs.max_lines must be <= 200000")
	}
	if c.HTTPMode != "enabled" && c.HTTPMode != "when_required" && c.HTTPMode != "disabled" {
		return errors.New("http_mode must be one of enabled, when_required, disabled")
	}
	if c.TLS.Enabled && c.TLS.Mode == "letsencrypt" && c.HTTPMode == "disabled" {
		return errors.New("http_mode cannot be disabled when tls.mode=letsencrypt")
	}
	nameSeen := map[string]struct{}{}
	for _, p := range c.Providers {
		if p.Name == "" {
			return errors.New("provider name cannot be empty")
		}
		if _, ok := nameSeen[p.Name]; ok {
			return fmt.Errorf("duplicate provider name %q", p.Name)
		}
		nameSeen[p.Name] = struct{}{}
	}
	if c.DefaultProvider != "" {
		if _, ok := nameSeen[c.DefaultProvider]; !ok {
			return fmt.Errorf("default_provider %q not found", c.DefaultProvider)
		}
	}
	return nil
}

func (c *ClientConfig) Normalize() {
	c.ServerURL = strings.TrimSpace(c.ServerURL)
	c.APIKey = strings.TrimSpace(c.APIKey)
	if c.ServerURL == "" {
		c.ServerURL = "http://127.0.0.1:7050/v1"
	}
}

func (c *ClientConfig) Validate() error {
	if strings.TrimSpace(c.ServerURL) == "" {
		return errors.New("server_url cannot be empty")
	}
	return nil
}

type ServerConfigStore struct {
	mu   sync.RWMutex
	path string
	cfg  *ServerConfig
}

func NewServerConfigStore(path string, cfg *ServerConfig) *ServerConfigStore {
	return &ServerConfigStore{path: path, cfg: cfg}
}

func (s *ServerConfigStore) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

func (s *ServerConfigStore) Snapshot() ServerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := *s.cfg
	cp.IncomingTokens = cloneIncomingTokens(s.cfg.IncomingTokens)
	cp.Providers = append([]ProviderConfig(nil), s.cfg.Providers...)
	return cp
}

func (s *ServerConfigStore) Update(mutator func(*ServerConfig) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *s.cfg
	cp.IncomingTokens = cloneIncomingTokens(s.cfg.IncomingTokens)
	cp.Providers = append([]ProviderConfig(nil), s.cfg.Providers...)
	if err := mutator(&cp); err != nil {
		return err
	}
	cp.Normalize()
	if err := cp.Validate(); err != nil {
		return err
	}
	if err := Save(s.path, &cp); err != nil {
		return err
	}
	s.cfg = &cp
	return nil
}

func cloneIncomingTokens(in []IncomingAPIToken) []IncomingAPIToken {
	if len(in) == 0 {
		return nil
	}
	out := make([]IncomingAPIToken, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Quota = cloneTokenQuota(in[i].Quota)
	}
	return out
}

func cloneTokenQuota(in *TokenQuota) *TokenQuota {
	if in == nil {
		return nil
	}
	out := &TokenQuota{}
	if in.Requests != nil {
		cp := *in.Requests
		out.Requests = &cp
	}
	if in.Tokens != nil {
		cp := *in.Tokens
		out.Tokens = &cp
	}
	if out.Requests == nil && out.Tokens == nil {
		return nil
	}
	return out
}

func normalizeTokenQuota(in *TokenQuota) *TokenQuota {
	if in == nil {
		return nil
	}
	out := cloneTokenQuota(in)
	out.Requests = normalizeTokenQuotaBudget(out.Requests)
	out.Tokens = normalizeTokenQuotaBudget(out.Tokens)
	if out.Requests == nil && out.Tokens == nil {
		return nil
	}
	return out
}

func normalizeTokenQuotaBudget(b *TokenQuotaBudget) *TokenQuotaBudget {
	if b == nil {
		return nil
	}
	cp := *b
	cp.WindowStartedAt = strings.TrimSpace(cp.WindowStartedAt)
	if cp.Limit <= 0 {
		return nil
	}
	if cp.IntervalSeconds < 0 {
		cp.IntervalSeconds = 0
	}
	if cp.Used < 0 {
		cp.Used = 0
	}
	if cp.Used > cp.Limit {
		cp.Used = cp.Limit
	}
	return &cp
}

func validateTokenQuota(tokenID string, q *TokenQuota) error {
	if q == nil {
		return nil
	}
	if err := validateTokenQuotaBudget(tokenID, "requests", q.Requests); err != nil {
		return err
	}
	if err := validateTokenQuotaBudget(tokenID, "tokens", q.Tokens); err != nil {
		return err
	}
	return nil
}

func validateTokenQuotaBudget(tokenID, name string, b *TokenQuotaBudget) error {
	if b == nil {
		return nil
	}
	if b.Limit <= 0 {
		return fmt.Errorf("incoming token %q quota.%s.limit must be > 0", tokenID, name)
	}
	if b.IntervalSeconds < 0 {
		return fmt.Errorf("incoming token %q quota.%s.interval_seconds must be >= 0", tokenID, name)
	}
	if b.Used < 0 {
		return fmt.Errorf("incoming token %q quota.%s.used must be >= 0", tokenID, name)
	}
	if b.WindowStartedAt != "" {
		if _, err := time.Parse(time.RFC3339, b.WindowStartedAt); err != nil {
			return fmt.Errorf("incoming token %q quota.%s.window_started_at must be RFC3339", tokenID, name)
		}
	}
	return nil
}

func tokenID(key string, idx int) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return fmt.Sprintf("tok-%d-%x", idx+1, h.Sum64())
}

func NormalizeIncomingTokenRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "", TokenRoleInferrer:
		return TokenRoleInferrer
	case TokenRoleAdmin:
		return TokenRoleAdmin
	case TokenRoleKeymaster:
		return TokenRoleKeymaster
	default:
		return ""
	}
}

func RoleAtLeast(actualRole, requiredRole string) bool {
	actual := NormalizeIncomingTokenRole(actualRole)
	required := NormalizeIncomingTokenRole(requiredRole)
	if actual == "" || required == "" {
		return false
	}
	return roleLevel(actual) >= roleLevel(required)
}

func roleLevel(role string) int {
	switch NormalizeIncomingTokenRole(role) {
	case TokenRoleAdmin:
		return 3
	case TokenRoleKeymaster:
		return 2
	case TokenRoleInferrer:
		return 1
	default:
		return 0
	}
}
