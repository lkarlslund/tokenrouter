package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lkarlslund/tokenrouter/pkg/config"
)

func TestListProvidersIncludesPublicFreeWhenEnabled(t *testing.T) {
	prevProbeFn := autoProviderProbeFn
	autoProviderProbeFn = func(config.ProviderConfig) bool { return true }
	autoProviderProbeState.mu.Lock()
	autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
	autoProviderProbeState.mu.Unlock()
	defer func() {
		autoProviderProbeFn = prevProbeFn
		autoProviderProbeState.mu.Lock()
		autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
		autoProviderProbeState.mu.Unlock()
	}()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = nil
	cfg.AutoEnablePublicFreeModels = true
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	providers := r.ListProviders()
	if len(providers) == 0 {
		t.Fatal("expected at least one provider when auto public free models is enabled")
	}
	found := false
	for _, p := range providers {
		if p.Name == "opencode-zen" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected opencode-zen to be included in auto-enabled public free providers")
	}
	for _, p := range providers {
		if p.Name == "nvidia" {
			t.Fatal("did not expect nvidia to be auto-enabled because it requires auth")
		}
	}
}

func TestResolveWithAutoPublicFreeProvider(t *testing.T) {
	prevProbeFn := autoProviderProbeFn
	autoProviderProbeFn = func(config.ProviderConfig) bool { return true }
	autoProviderProbeState.mu.Lock()
	autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
	autoProviderProbeState.mu.Unlock()
	defer func() {
		autoProviderProbeFn = prevProbeFn
		autoProviderProbeState.mu.Lock()
		autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
		autoProviderProbeState.mu.Unlock()
	}()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = nil
	cfg.AutoEnablePublicFreeModels = true
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	p, model, err := r.Resolve("opencode-zen/gpt-5-nano")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if p.Name != "opencode-zen" {
		t.Fatalf("expected opencode-zen provider, got %q", p.Name)
	}
	if model != "gpt-5-nano" {
		t.Fatalf("expected stripped model gpt-5-nano, got %q", model)
	}
}

func TestListProvidersResolvesConfiguredPresetDefaults(t *testing.T) {
	cfg := config.NewDefaultServerConfig()
	cfg.AutoEnablePublicFreeModels = false
	cfg.AutoDetectLocalServers = false
	cfg.Providers = []config.ProviderConfig{
		{
			Name:    "opencode-zen",
			APIKey:  "x",
			Enabled: true,
		},
	}
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	providers := r.ListProviders()
	if len(providers) != 1 {
		t.Fatalf("expected one provider, got %d", len(providers))
	}
	if providers[0].Name != "opencode-zen" {
		t.Fatalf("expected opencode-zen, got %q", providers[0].Name)
	}
	if providers[0].BaseURL == "" {
		t.Fatal("expected base_url to be resolved from preset defaults")
	}
	if providers[0].TimeoutSeconds <= 0 {
		t.Fatal("expected timeout_seconds to be defaulted")
	}
}

func TestProviderClientUsesAuthTokenWhenAPIKeyEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer device-token" {
			t.Fatalf("expected bearer device token, got %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"m"}]}`))
	}))
	defer srv.Close()
	p := config.ProviderConfig{Name: "x", BaseURL: srv.URL, AuthToken: "device-token", Enabled: true, TimeoutSeconds: 2}
	models, err := NewProviderClient(p).ListModels(context.Background())
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected one model, got %d", len(models))
	}
	if models[0].ID != "x/m" {
		t.Fatalf("expected model ID x/m, got %q", models[0].ID)
	}
}

func TestProviderClientReturnsStaticModelsForCodexProvider(t *testing.T) {
	p := config.ProviderConfig{
		Name:      "openai",
		BaseURL:   "https://chatgpt.com/backend-api",
		AuthToken: "oauth-token",
		Enabled:   true,
	}
	models, err := NewProviderClient(p).ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) < 5 {
		t.Fatalf("expected at least 5 static codex models, got %d", len(models))
	}
	if models[0].Provider != "openai" {
		t.Fatalf("expected provider openai, got %q", models[0].Provider)
	}
}

func TestIsProviderBlockedCloudflareChallenge(t *testing.T) {
	err := &ProviderHTTPError{
		Provider:   "openai",
		StatusCode: http.StatusForbidden,
		Body:       "<html><title>Just a moment...</title>__cf_chl_tk=abc</html>",
	}
	if !IsProviderBlocked(err) {
		t.Fatal("expected cloudflare challenge to be classified as blocked")
	}
	if IsProviderAuthError(err) {
		t.Fatal("expected blocked challenge not to be classified as auth problem")
	}
}

func TestProviderClientNormalizesModelsPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"models/gemini-2.5-flash"}]}`))
	}))
	defer srv.Close()
	p := config.ProviderConfig{Name: "google-gemini", BaseURL: srv.URL, Enabled: true, TimeoutSeconds: 2}
	models, err := NewProviderClient(p).ListModels(context.Background())
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected one model, got %d", len(models))
	}
	if models[0].ID != "google-gemini/gemini-2.5-flash" {
		t.Fatalf("expected normalized model ID, got %q", models[0].ID)
	}
}

func TestResolveNormalizesModelsPrefix(t *testing.T) {
	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{Name: "google-gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", APIKey: "x", Enabled: true},
	}
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	p, model, err := r.Resolve("google-gemini/models/gemini-2.5-flash")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if p.Name != "google-gemini" {
		t.Fatalf("expected google-gemini provider, got %q", p.Name)
	}
	if model != "gemini-2.5-flash" {
		t.Fatalf("expected normalized model, got %q", model)
	}
}

func TestListProvidersResolvesPresetDefaultsUsingProviderType(t *testing.T) {
	cfg := config.NewDefaultServerConfig()
	cfg.AutoEnablePublicFreeModels = false
	cfg.AutoDetectLocalServers = false
	cfg.Providers = []config.ProviderConfig{
		{
			Name:         "openai-work",
			ProviderType: "openai",
			APIKey:       "x",
			Enabled:      true,
		},
	}
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	providers := r.ListProviders()
	if len(providers) != 1 {
		t.Fatalf("expected one provider, got %d", len(providers))
	}
	if providers[0].Name != "openai-work" {
		t.Fatalf("expected openai-work, got %q", providers[0].Name)
	}
	if providers[0].ProviderType != "openai" {
		t.Fatalf("expected provider_type openai, got %q", providers[0].ProviderType)
	}
	if providers[0].BaseURL == "" {
		t.Fatal("expected base_url to be resolved from openai preset via provider_type")
	}
}

func TestResolvePrefersOpenAIProviderForUnqualifiedGPTModel(t *testing.T) {
	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{Name: "groq", BaseURL: "https://api.groq.com/openai/v1", APIKey: "x", Enabled: true},
		{Name: "openai", BaseURL: "https://chatgpt.com/backend-api", AuthToken: "tok", Enabled: true},
	}
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	p, model, err := r.Resolve("gpt-5.3-codex")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if p.Name != "openai" {
		t.Fatalf("expected openai provider for unqualified gpt model, got %q", p.Name)
	}
	if model != "gpt-5.3-codex" {
		t.Fatalf("expected model unchanged, got %q", model)
	}
}

func TestProbeLMStudioOnlineRejectsOllamaSignature(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"models":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	if probeLMStudioOnline(config.ProviderConfig{Name: "lmstudio", BaseURL: srv.URL + "/v1"}) {
		t.Fatal("expected lmstudio probe to reject ollama signature")
	}
}

func TestProbeLMStudioOnlineAcceptsNativeAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			http.NotFound(w, r)
		case "/api/v0/models":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"qwen2.5-7b-instruct"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if !probeLMStudioOnline(config.ProviderConfig{Name: "lmstudio", BaseURL: srv.URL + "/v1"}) {
		t.Fatal("expected lmstudio probe to accept native api signature")
	}
}

func TestProbeOllamaOnlineAcceptsTagsEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"models":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	if !probeOllamaOnline(config.ProviderConfig{Name: "ollama", BaseURL: srv.URL + "/v1"}) {
		t.Fatal("expected ollama probe to accept /api/tags signature")
	}
}

func TestListProvidersAutoDetectsLocalOllama(t *testing.T) {
	prevProbeFn := autoProviderProbeFn
	prevEnv := autoProviderEnvLookup
	prevLlamaProbe := llamaProcessProbeFn
	autoProviderProbeFn = func(p config.ProviderConfig) bool { return strings.TrimSpace(p.Name) == "ollama" }
	autoProviderEnvLookup = func(string) string { return "" }
	llamaProcessProbeFn = func() []llamaProcessInfo { return nil }
	autoProviderProbeState.mu.Lock()
	autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
	autoProviderProbeState.mu.Unlock()
	llamaProcessProbeState.mu.Lock()
	llamaProcessProbeState.checkedAt = time.Time{}
	llamaProcessProbeState.processes = nil
	llamaProcessProbeState.mu.Unlock()
	defer func() {
		autoProviderProbeFn = prevProbeFn
		autoProviderEnvLookup = prevEnv
		llamaProcessProbeFn = prevLlamaProbe
		autoProviderProbeState.mu.Lock()
		autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
		autoProviderProbeState.mu.Unlock()
		llamaProcessProbeState.mu.Lock()
		llamaProcessProbeState.checkedAt = time.Time{}
		llamaProcessProbeState.processes = nil
		llamaProcessProbeState.mu.Unlock()
	}()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = nil
	cfg.AutoEnablePublicFreeModels = false
	cfg.AutoDetectLocalServers = true
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	providers := r.ListProviders()
	found := false
	for _, p := range providers {
		if p.Name == "ollama" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected local ollama to be auto-detected")
	}
}

func TestListProvidersSkipsLocalAutoDetectionWhenDisabled(t *testing.T) {
	prevProbeFn := autoProviderProbeFn
	prevEnv := autoProviderEnvLookup
	prevLlamaProbe := llamaProcessProbeFn
	autoProviderProbeFn = func(config.ProviderConfig) bool { return true }
	autoProviderEnvLookup = func(string) string { return "" }
	llamaProcessProbeFn = func() []llamaProcessInfo {
		return []llamaProcessInfo{{Host: "127.0.0.1", Port: 8081}}
	}
	autoProviderProbeState.mu.Lock()
	autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
	autoProviderProbeState.mu.Unlock()
	llamaProcessProbeState.mu.Lock()
	llamaProcessProbeState.checkedAt = time.Time{}
	llamaProcessProbeState.processes = nil
	llamaProcessProbeState.mu.Unlock()
	defer func() {
		autoProviderProbeFn = prevProbeFn
		autoProviderEnvLookup = prevEnv
		llamaProcessProbeFn = prevLlamaProbe
		autoProviderProbeState.mu.Lock()
		autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
		autoProviderProbeState.mu.Unlock()
		llamaProcessProbeState.mu.Lock()
		llamaProcessProbeState.checkedAt = time.Time{}
		llamaProcessProbeState.processes = nil
		llamaProcessProbeState.mu.Unlock()
	}()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = nil
	cfg.AutoEnablePublicFreeModels = true
	cfg.AutoDetectLocalServers = false
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	providers := r.ListProviders()
	for _, p := range providers {
		if p.Name == "ollama" || p.Name == "lmstudio" || strings.HasPrefix(p.Name, "llama-cpp-local-") {
			t.Fatalf("did not expect local auto-detected provider %q when local detection is disabled", p.Name)
		}
	}
}

func TestListProvidersAutoDetectsOllamaCloudFromEnv(t *testing.T) {
	prevProbeFn := autoProviderProbeFn
	prevEnv := autoProviderEnvLookup
	prevLlamaProbe := llamaProcessProbeFn
	autoProviderProbeFn = func(p config.ProviderConfig) bool { return strings.TrimSpace(p.Name) == "ollama-cloud" }
	autoProviderEnvLookup = func(key string) string {
		if key == "OLLAMA_API_KEY" {
			return "test-key"
		}
		return ""
	}
	llamaProcessProbeFn = func() []llamaProcessInfo { return nil }
	autoProviderProbeState.mu.Lock()
	autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
	autoProviderProbeState.mu.Unlock()
	llamaProcessProbeState.mu.Lock()
	llamaProcessProbeState.checkedAt = time.Time{}
	llamaProcessProbeState.processes = nil
	llamaProcessProbeState.mu.Unlock()
	defer func() {
		autoProviderProbeFn = prevProbeFn
		autoProviderEnvLookup = prevEnv
		llamaProcessProbeFn = prevLlamaProbe
		autoProviderProbeState.mu.Lock()
		autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
		autoProviderProbeState.mu.Unlock()
		llamaProcessProbeState.mu.Lock()
		llamaProcessProbeState.checkedAt = time.Time{}
		llamaProcessProbeState.processes = nil
		llamaProcessProbeState.mu.Unlock()
	}()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = nil
	cfg.AutoEnablePublicFreeModels = true
	cfg.AutoDetectLocalServers = true
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	providers := r.ListProviders()
	found := false
	for _, p := range providers {
		if p.Name == "ollama-cloud" {
			found = true
			if strings.TrimSpace(p.APIKey) != "test-key" {
				t.Fatalf("expected auto-detected ollama-cloud to use env api key, got %q", p.APIKey)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected ollama-cloud to be auto-detected when OLLAMA_API_KEY is set")
	}
}

func TestListProvidersAutoDetectsLlamaCPPProcesses(t *testing.T) {
	prevProbeFn := autoProviderProbeFn
	prevEnv := autoProviderEnvLookup
	prevLlamaProbe := llamaProcessProbeFn
	autoProviderProbeFn = func(p config.ProviderConfig) bool {
		return strings.HasPrefix(strings.TrimSpace(p.Name), "llama-cpp-local-")
	}
	autoProviderEnvLookup = func(string) string { return "" }
	llamaProcessProbeFn = func() []llamaProcessInfo {
		return []llamaProcessInfo{
			{Host: "0.0.0.0", Port: 8081},
			{Host: "127.0.0.1", Port: 8082},
		}
	}
	autoProviderProbeState.mu.Lock()
	autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
	autoProviderProbeState.mu.Unlock()
	llamaProcessProbeState.mu.Lock()
	llamaProcessProbeState.checkedAt = time.Time{}
	llamaProcessProbeState.processes = nil
	llamaProcessProbeState.mu.Unlock()
	defer func() {
		autoProviderProbeFn = prevProbeFn
		autoProviderEnvLookup = prevEnv
		llamaProcessProbeFn = prevLlamaProbe
		autoProviderProbeState.mu.Lock()
		autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
		autoProviderProbeState.mu.Unlock()
		llamaProcessProbeState.mu.Lock()
		llamaProcessProbeState.checkedAt = time.Time{}
		llamaProcessProbeState.processes = nil
		llamaProcessProbeState.mu.Unlock()
	}()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = nil
	cfg.AutoEnablePublicFreeModels = false
	cfg.AutoDetectLocalServers = true
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	providers := r.ListProviders()
	want := map[string]string{
		"llama-cpp-local-8081": "http://127.0.0.1:8081/v1",
		"llama-cpp-local-8082": "http://127.0.0.1:8082/v1",
	}
	for _, p := range providers {
		if u, ok := want[p.Name]; ok {
			if p.BaseURL != u {
				t.Fatalf("expected %s base_url %q, got %q", p.Name, u, p.BaseURL)
			}
			delete(want, p.Name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing auto-detected llama.cpp providers: %+v", want)
	}
}

func TestListProvidersSkipsSelfListenPortForLocalAutoDetect(t *testing.T) {
	prevProbeFn := autoProviderProbeFn
	prevEnv := autoProviderEnvLookup
	prevLlamaProbe := llamaProcessProbeFn
	autoProviderProbeFn = func(p config.ProviderConfig) bool {
		return strings.HasPrefix(strings.TrimSpace(p.Name), "llama-cpp-local-")
	}
	autoProviderEnvLookup = func(string) string { return "" }
	llamaProcessProbeFn = func() []llamaProcessInfo {
		return []llamaProcessInfo{
			{Host: "127.0.0.1", Port: 8080},
			{Host: "127.0.0.1", Port: 8081},
		}
	}
	autoProviderProbeState.mu.Lock()
	autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
	autoProviderProbeState.mu.Unlock()
	llamaProcessProbeState.mu.Lock()
	llamaProcessProbeState.checkedAt = time.Time{}
	llamaProcessProbeState.processes = nil
	llamaProcessProbeState.mu.Unlock()
	defer func() {
		autoProviderProbeFn = prevProbeFn
		autoProviderEnvLookup = prevEnv
		llamaProcessProbeFn = prevLlamaProbe
		autoProviderProbeState.mu.Lock()
		autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
		autoProviderProbeState.mu.Unlock()
		llamaProcessProbeState.mu.Lock()
		llamaProcessProbeState.checkedAt = time.Time{}
		llamaProcessProbeState.processes = nil
		llamaProcessProbeState.mu.Unlock()
	}()

	cfg := config.NewDefaultServerConfig()
	cfg.ListenAddr = "0.0.0.0:8080"
	cfg.Providers = nil
	cfg.AutoEnablePublicFreeModels = false
	cfg.AutoDetectLocalServers = true
	store := config.NewServerConfigStore("/tmp/non-persistent.toml", cfg)
	r := NewProviderResolver(store)

	providers := r.ListProviders()
	for _, p := range providers {
		if p.Name == "llama-cpp-local-8080" {
			t.Fatalf("did not expect self-listen address to be auto-detected: %+v", p)
		}
	}
	found8081 := false
	for _, p := range providers {
		if p.Name == "llama-cpp-local-8081" {
			found8081 = true
			break
		}
	}
	if !found8081 {
		t.Fatalf("expected non-self local server to remain detectable, got %+v", providers)
	}
}

func TestProbeLlamaCPPHostPortAcceptsV1Models(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/props":
			http.NotFound(w, r)
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"qwen2.5-7b-instruct"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	u := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, ok := strings.Cut(u, ":")
	if !ok {
		t.Fatalf("unexpected test server url %q", srv.URL)
	}
	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	if !probeLlamaCPPHostPort(host, port) {
		t.Fatal("expected llama.cpp probe to accept /v1/models signature")
	}
}

func TestProbeLlamaCPPHostPortAcceptsHealthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/props", "/v1/models":
			http.NotFound(w, r)
		case "/health":
			w.Header().Set("Server", "llama.cpp")
			_, _ = w.Write([]byte("ok"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	u := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, ok := strings.Cut(u, ":")
	if !ok {
		t.Fatalf("unexpected test server url %q", srv.URL)
	}
	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	if !probeLlamaCPPHostPort(host, port) {
		t.Fatal("expected llama.cpp probe to accept /health signature")
	}
}
