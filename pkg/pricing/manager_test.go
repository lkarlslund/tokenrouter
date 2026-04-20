package pricing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lkarlslund/tokenrouter/pkg/config"
)

func TestExtractZenModelIDs(t *testing.T) {
	doc := `<h3 id="models"><a href="#models">Models</a></h3><table><thead><tr><th>Model</th><th>Model ID</th><th>Endpoint</th><th>Package</th></tr></thead><tbody>` +
		`<tr><td>GPT 5</td><td>gpt-5</td><td><code>https://opencode.ai/zen/v1/responses</code></td><td><code>@ai-sdk/openai</code></td></tr>` +
		`<tr><td>Kimi K2.5</td><td><code>kimi-k2.5</code></td><td><code>https://opencode.ai/zen/v1/chat/completions</code></td><td><code>@ai-sdk/openai-compatible</code></td></tr>` +
		`</tbody></table>`
	ids := extractZenModelIDs(doc)
	if len(ids) != 2 {
		t.Fatalf("expected 2 model IDs, got %d (%v)", len(ids), ids)
	}
	if ids[0] != "gpt-5" || ids[1] != "kimi-k2.5" {
		t.Fatalf("unexpected model IDs: %v", ids)
	}
}

func TestNormalizePricingModelID(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider string
		in       string
		want     string
	}{
		{name: "bare", provider: "opencode-zen", in: "gpt-5-nano", want: "gpt-5-nano"},
		{name: "prefixed", provider: "opencode-zen", in: "opencode-zen/gpt-5-nano", want: "gpt-5-nano"},
		{name: "models prefix", provider: "google-gemini", in: "models/gemini-2.5-flash", want: "gemini-2.5-flash"},
		{name: "other provider prefix preserved", provider: "opencode-zen", in: "openai/gpt-5", want: "openai/gpt-5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizePricingModelID(tc.provider, tc.in); got != tc.want {
				t.Fatalf("normalizePricingModelID(%q, %q) = %q, want %q", tc.provider, tc.in, got, tc.want)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	in := []string{"a", "b", "a", "c", "b"}
	out := uniqueStrings(in)
	if len(out) != 3 {
		t.Fatalf("expected 3 unique strings, got %d (%v)", len(out), out)
	}
	if out[0] != "a" || out[1] != "b" || out[2] != "c" {
		t.Fatalf("unexpected order/content: %v", out)
	}
}

func TestParseGeminiPricingRows(t *testing.T) {
	doc := `
<h2 id="gemini-2.5-pro">Gemini 2.5 Pro</h2>
<em><code>gemini-2.5-pro</code></em>
<table><tbody>
<tr><td>Input price</td><td>Free of charge</td><td>$1.25, prompts &lt;= 200k tokens<br>$2.50, prompts &gt; 200k tokens</td></tr>
<tr><td>Output price (including thinking tokens)</td><td>Free of charge</td><td>$10.00, prompts &lt;= 200k tokens<br>$15.00, prompts &gt; 200k</td></tr>
</tbody></table>
<h2 id="gemini-2.5-flash">Gemini 2.5 Flash</h2>
<em><code>gemini-2.5-flash</code></em>
<table><tbody>
<tr><td>Input price</td><td>Free of charge</td><td>Free of charge</td></tr>
<tr><td>Output price (including thinking tokens)</td><td>Free of charge</td><td>Free of charge</td></tr>
</tbody></table>`
	rows := parseGeminiPricingRows(doc)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d (%v)", len(rows), rows)
	}
	if rows[0].Model != "gemini-2.5-pro" || rows[0].InputPer1M != 1.25 || rows[0].OutputPer1M != 10.0 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
	if rows[1].Model != "gemini-2.5-flash" || rows[1].InputPer1M != 0 || rows[1].OutputPer1M != 0 {
		t.Fatalf("unexpected second row: %+v", rows[1])
	}
}

func TestProviderNeedsRefreshUsesLastAttemptWindow(t *testing.T) {
	now := time.Now()
	m := &Manager{
		cache: Cache{
			ProviderStates: map[string]ProviderState{
				"google-gemini": {LastAttempt: now.Add(-2 * time.Hour), LastUpdate: now.Add(-2 * time.Hour)},
			},
			Entries: map[string]ModelPricing{},
		},
	}
	if m.providerNeedsRefreshLocked("google-gemini", now) {
		t.Fatal("expected provider not to need refresh before 12h window")
	}
	if !m.providerNeedsRefreshLocked("google-gemini", now.Add(13*time.Hour)) {
		t.Fatal("expected provider to need refresh after 12h window")
	}
}

func TestForceRefreshBypassesRefreshInterval(t *testing.T) {
	now := time.Now()
	cachePath := filepath.Join(t.TempDir(), "pricing-cache.json")
	m, err := NewManager(cachePath)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	m.cache = Cache{
		UpdatedAt: now,
		ProviderStates: map[string]ProviderState{
			"ollama": {LastAttempt: now, LastUpdate: now},
		},
		Entries: map[string]ModelPricing{},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "llama3.1:8b"},
			},
		})
	}))
	defer srv.Close()
	m.SetProviders([]config.ProviderConfig{
		{Name: "ollama", ProviderType: "ollama", BaseURL: srv.URL + "/v1", Enabled: true, TimeoutSeconds: 5},
	})

	if err := m.ForceRefresh(context.Background()); err != nil {
		t.Fatalf("force refresh: %v", err)
	}
	snap := m.Snapshot()
	if _, ok := snap.Entries["ollama/llama3.1:8b"]; !ok {
		t.Fatalf("expected forced refresh to populate ollama pricing entry, got %+v", snap.Entries)
	}
}

func TestProviderTypeOrNameTreatsLocalOllamaAsLocal(t *testing.T) {
	p := config.ProviderConfig{
		Name:         "ollama",
		ProviderType: "ollama-cloud",
	}
	if got := providerTypeOrName(p); got != "ollama" {
		t.Fatalf("expected local ollama provider type, got %q", got)
	}
}

func TestNvidiaNIMPricingSourceFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"meta/llama-3.1-70b-instruct"},{"id":"nvidia/llama-3.1-nemotron-ultra-253b-v1"}]}`))
	}))
	defer srv.Close()

	src := &NvidiaNIMPricingSource{}
	provider := config.ProviderConfig{
		Name:           "nvidia",
		BaseURL:        srv.URL + "/v1",
		TimeoutSeconds: 5,
	}

	entries, source, err := src.Fetch(context.Background(), provider)
	if err != nil {
		t.Fatalf("fetch pricing: %v", err)
	}
	if source != "https://build.nvidia.com/" {
		t.Fatalf("unexpected source: %q", source)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Currency != "USD" {
			t.Fatalf("unexpected currency for %s: %s", e.Model, e.Currency)
		}
		if e.InputPer1M != 0 || e.OutputPer1M != 0 {
			t.Fatalf("expected zero pricing for %s, got in=%f out=%f", e.Model, e.InputPer1M, e.OutputPer1M)
		}
	}
	if !src.Match(provider) {
		t.Fatal("expected nvidia source to match provider by name")
	}
	if !src.Match(config.ProviderConfig{Name: "custom", BaseURL: "https://integrate.api.nvidia.com/v1"}) {
		t.Fatal("expected nvidia source to match provider by base URL")
	}
	if src.Match(config.ProviderConfig{Name: "openai", BaseURL: "https://api.openai.com/v1"}) {
		t.Fatal("did not expect nvidia source to match non-nvidia provider")
	}
	models := make([]string, 0, len(entries))
	for _, e := range entries {
		models = append(models, e.Model)
	}
	got := strings.Join(models, ",")
	if !strings.Contains(got, "meta/llama-3.1-70b-instruct") || !strings.Contains(got, "nvidia/llama-3.1-nemotron-ultra-253b-v1") {
		t.Fatalf("unexpected models: %s", got)
	}
}

func TestAllFreePricingSourceFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"llama3.2"},{"id":"qwen2.5"}]}`))
	}))
	defer srv.Close()

	src := &AllFreePricingSource{}
	provider := config.ProviderConfig{
		Name:         "ollama",
		ProviderType: "ollama",
		BaseURL:      srv.URL + "/v1",
	}
	entries, source, err := src.Fetch(context.Background(), provider)
	if err != nil {
		t.Fatalf("fetch pricing: %v", err)
	}
	if source != "allfree" {
		t.Fatalf("unexpected source: %q", source)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Currency != "USD" {
			t.Fatalf("unexpected currency for %s: %s", e.Model, e.Currency)
		}
		if e.InputPer1M != 0 || e.OutputPer1M != 0 {
			t.Fatalf("expected zero pricing for %s, got in=%f out=%f", e.Model, e.InputPer1M, e.OutputPer1M)
		}
	}
	if !src.Match(provider) {
		t.Fatal("expected allfree source to match local ollama")
	}
	if !src.Match(config.ProviderConfig{Name: "lmstudio", ProviderType: "lmstudio", BaseURL: "http://127.0.0.1:1234/v1"}) {
		t.Fatal("expected allfree source to match lmstudio")
	}
	if src.Match(config.ProviderConfig{Name: "openai", ProviderType: "openai", BaseURL: "https://api.openai.com/v1"}) {
		t.Fatal("did not expect allfree source to match non-local provider")
	}
}

func TestParsePricingFieldsSupportsInputOutput(t *testing.T) {
	item := map[string]any{
		"id": "meta-llama/Llama-3.1-8B-Instruct",
		"pricing": map[string]any{
			"input":  0.02,
			"output": 0.05,
		},
	}
	in, out, ok := parsePricingFields(item)
	if !ok {
		t.Fatal("expected pricing to parse")
	}
	if in != 20000 || out != 50000 {
		t.Fatalf("unexpected pricing parsed: in=%f out=%f", in, out)
	}
}

func TestParsePricingFieldsFromProvidersArray(t *testing.T) {
	item := map[string]any{
		"id": "Qwen/Qwen3-Coder-Next",
		"providers": []any{
			map[string]any{
				"provider": "novita",
				"pricing":  map[string]any{"input": 0.2, "output": 1.5},
			},
			map[string]any{
				"provider": "together",
				"pricing":  map[string]any{"input": 0.3, "output": 1.8},
			},
			map[string]any{
				"provider": "featherless-ai",
			},
		},
	}
	in, out, ok := parsePricingFields(item)
	if !ok {
		t.Fatal("expected pricing from providers array to parse")
	}
	if in != 200000 || out != 1500000 {
		t.Fatalf("unexpected pricing parsed from providers array: in=%f out=%f", in, out)
	}
}

func TestCerebrasPublicPricingSourceFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"llama3.1-8b","pricing":{"prompt":"0.0000001","completion":"0.0000001"}},{"id":"gpt-oss-120b","pricing":{"prompt":"0.00000035","completion":"0.00000075"}}]}`))
	}))
	defer srv.Close()

	src := &CerebrasPublicPricingSource{}
	provider := config.ProviderConfig{
		Name:         "cerebras-test",
		ProviderType: "cerebras-test",
		BaseURL:      "https://api.cerebras.ai/v1",
		ModelListURL: srv.URL + "/public/v1/models",
	}
	entries, source, err := src.Fetch(context.Background(), provider)
	if err != nil {
		t.Fatalf("fetch pricing: %v", err)
	}
	if source != provider.ModelListURL {
		t.Fatalf("unexpected source: %q", source)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	var found bool
	for _, e := range entries {
		if e.Model == "gpt-oss-120b" {
			found = true
			if e.InputPer1M != 0.35 || e.OutputPer1M != 0.75 {
				t.Fatalf("unexpected gpt-oss-120b pricing: in=%f out=%f", e.InputPer1M, e.OutputPer1M)
			}
		}
	}
	if !found {
		t.Fatal("expected gpt-oss-120b in entries")
	}
	if !src.Match(config.ProviderConfig{Name: "cerebras", BaseURL: "https://api.cerebras.ai/v1"}) {
		t.Fatal("expected source to match cerebras provider")
	}
	if !src.Match(config.ProviderConfig{Name: "custom-cerebras", BaseURL: "https://api.cerebras.ai/v1"}) {
		t.Fatal("expected source to match cerebras base URL")
	}
	if src.Match(config.ProviderConfig{Name: "openai", BaseURL: "https://api.openai.com/v1"}) {
		t.Fatal("did not expect source to match non-cerebras provider")
	}
}

func TestModelsEndpointPricingSourceFetchHuggingFaceUsesPer1MPricing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"Qwen/Qwen2.5-7B-Instruct","pricing":{"input":0.12,"output":0.35}}]}`))
	}))
	defer srv.Close()

	src := &ModelsEndpointPricingSource{}
	provider := config.ProviderConfig{
		Name:         "hf-main",
		ProviderType: "huggingface",
		BaseURL:      srv.URL + "/v1",
	}
	entries, source, err := src.Fetch(context.Background(), provider)
	if err != nil {
		t.Fatalf("fetch pricing: %v", err)
	}
	if source != "v1/models" {
		t.Fatalf("unexpected source: %q", source)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].InputPer1M != 0.12 || entries[0].OutputPer1M != 0.35 {
		t.Fatalf("unexpected huggingface pricing: in=%f out=%f", entries[0].InputPer1M, entries[0].OutputPer1M)
	}
}
