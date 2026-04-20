package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/lkarlslund/tokenrouter/pkg/cache"
	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/pricing"
)

func TestAdminModelsCatalogDedupesProviderQualifiedLiveModelsAgainstPricingCache(t *testing.T) {
	isolateDefaultDataPaths(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			http.NotFound(w, r)
		case "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:8b"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.AllowLocalhostNoAuth = true
	cfg.IncomingTokens = nil
	cfg.AutoEnablePublicFreeModels = false
	cfg.AutoDetectLocalServers = false
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "ollama",
			ProviderType:   "ollama",
			BaseURL:        upstream.URL + "/v1",
			Enabled:        true,
			TimeoutSeconds: 2,
		},
	}
	s, err := NewServer(filepath.Join(t.TempDir(), "torod.toml"), cfg)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	pricingPath := filepath.Join(t.TempDir(), "pricing-cache.json")
	pricingCache := pricing.Cache{
		UpdatedAt:      time.Now().UTC(),
		ProviderStates: map[string]pricing.ProviderState{},
		Entries: map[string]pricing.ModelPricing{
			"ollama/qwen3:8b": {
				Provider:     "ollama",
				Model:        "qwen3:8b",
				Currency:     "USD",
				InputPer1M:   0,
				OutputPer1M:  0,
				DiscoveredAt: time.Now().UTC(),
				Source:       "test",
			},
		},
	}
	if err := cache.SaveJSON(pricingPath, pricingCache); err != nil {
		t.Fatalf("save pricing cache: %v", err)
	}
	pricingMgr, err := pricing.NewManager(pricingPath)
	if err != nil {
		t.Fatalf("new pricing manager: %v", err)
	}
	pricingMgr.SetProviders(cfg.Providers)
	s.pricing = pricingMgr
	s.adminHandler.pricing = pricingMgr

	req := httptest.NewRequest(http.MethodGet, "/admin/api/models", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /admin/api/models, got %d body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Data []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("expected exactly one model row, got %d body=%s", len(body.Data), w.Body.String())
	}
	if body.Data[0].Provider != "ollama" {
		t.Fatalf("expected provider ollama, got %+v", body.Data[0])
	}
	if body.Data[0].Model != "qwen3:8b" {
		t.Fatalf("expected unqualified model id qwen3:8b, got %+v", body.Data[0])
	}
}

func TestAdminModelsCatalogUsesLegacyDoublePrefixedPricingCacheEntry(t *testing.T) {
	isolateDefaultDataPaths(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"opencode-zen/gpt-5-nano"}]}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.AllowLocalhostNoAuth = true
	cfg.IncomingTokens = nil
	cfg.AutoEnablePublicFreeModels = false
	cfg.AutoDetectLocalServers = false
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "opencode-zen",
			ProviderType:   "opencode-zen",
			BaseURL:        upstream.URL + "/v1",
			Enabled:        true,
			TimeoutSeconds: 2,
		},
	}
	s, err := NewServer(filepath.Join(t.TempDir(), "torod.toml"), cfg)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	pricingPath := filepath.Join(t.TempDir(), "pricing-cache.json")
	pricingCache := pricing.Cache{
		UpdatedAt:      time.Now().UTC(),
		ProviderStates: map[string]pricing.ProviderState{},
		Entries: map[string]pricing.ModelPricing{
			"opencode-zen/opencode-zen/gpt-5-nano": {
				Provider:     "opencode-zen",
				Model:        "opencode-zen/gpt-5-nano",
				Currency:     "USD",
				InputPer1M:   0,
				OutputPer1M:  0,
				DiscoveredAt: time.Now().UTC(),
				Source:       "legacy-test",
			},
		},
	}
	if err := cache.SaveJSON(pricingPath, pricingCache); err != nil {
		t.Fatalf("save pricing cache: %v", err)
	}
	pricingMgr, err := pricing.NewManager(pricingPath)
	if err != nil {
		t.Fatalf("new pricing manager: %v", err)
	}
	pricingMgr.SetProviders(cfg.Providers)
	s.pricing = pricingMgr
	s.adminHandler.pricing = pricingMgr

	req := httptest.NewRequest(http.MethodGet, "/admin/api/models", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /admin/api/models, got %d body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Data []struct {
			Provider   string   `json:"provider"`
			Model      string   `json:"model"`
			InputPer1M *float64 `json:"input_per_1m"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("expected exactly one model row, got %d body=%s", len(body.Data), w.Body.String())
	}
	if body.Data[0].Provider != "opencode-zen" || body.Data[0].Model != "gpt-5-nano" {
		t.Fatalf("unexpected model row: %+v", body.Data[0])
	}
	if body.Data[0].InputPer1M == nil || *body.Data[0].InputPer1M != 0 {
		t.Fatalf("expected free pricing from legacy cache entry, got %+v", body.Data[0])
	}
}
