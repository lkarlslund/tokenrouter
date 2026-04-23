package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/conversations"
)

func TestRecordUsageCountsRequestWithoutUsagePayload(t *testing.T) {
	s := &Server{stats: NewStatsStore(100)}
	s.recordUsage("openai", "openai/gpt-4.1", "gpt-4.1", 200, 120*time.Millisecond, []byte(`{"id":"resp_123","object":"response"}`))

	summary := s.stats.Summary(time.Hour)
	if summary.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", summary.Requests)
	}
	if summary.TotalTokens != 0 {
		t.Fatalf("expected 0 total tokens, got %d", summary.TotalTokens)
	}
	if got := summary.RequestsPerProvider["openai"]; got != 1 {
		t.Fatalf("expected provider request count 1, got %d", got)
	}
	if got := summary.RequestsPerModel["openai/gpt-4.1"]; got != 1 {
		t.Fatalf("expected model request count 1, got %d", got)
	}
}

func TestRecordUsageParsesResponsesUsageSchema(t *testing.T) {
	s := &Server{stats: NewStatsStore(100)}
	body := []byte(`{"usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18}}`)
	s.recordUsage("openai", "openai/gpt-4.1", "gpt-4.1", 200, 250*time.Millisecond, body)

	summary := s.stats.Summary(time.Hour)
	if summary.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", summary.Requests)
	}
	if summary.PromptTokens != 11 {
		t.Fatalf("expected 11 prompt tokens, got %d", summary.PromptTokens)
	}
	if summary.CompletionTokens != 7 {
		t.Fatalf("expected 7 completion tokens, got %d", summary.CompletionTokens)
	}
	if summary.TotalTokens != 18 {
		t.Fatalf("expected 18 total tokens, got %d", summary.TotalTokens)
	}
}

func TestRecordUsageMeasuredRecordsFailedStatuses(t *testing.T) {
	s := &Server{stats: NewStatsStore(100)}
	s.recordUsageMeasured("openai", "openai/gpt-4.1", "gpt-4.1", 500, 100*time.Millisecond, 0, 0, 0, 0, 0, 0, clientUsageMeta{})

	summary := s.stats.Summary(time.Hour)
	if summary.Requests != 1 {
		t.Fatalf("expected failed status to be tracked, got %d requests", summary.Requests)
	}
	if summary.FailedRequests != 1 {
		t.Fatalf("expected failed request count 1, got %d", summary.FailedRequests)
	}
}

func TestDedupeConversationTextPreservesSpacingAndMarkdown(t *testing.T) {
	in := "time for a conversation\n" +
		"time for a conversation\n" +
		"\n" +
		"hi hi\n" +
		"hi    hi\n" +
		"\n" +
		"again\n" +
		"again\n" +
		"again\n"

	out := dedupeConversationText(in)
	if out != strings.TrimSpace(in) {
		t.Fatalf("expected text fidelity, got: %q", out)
	}
}

func TestProxyHandlerCapturesConversation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id":"resp_conv_1",
			"usage":{"input_tokens":5,"output_tokens":3,"total_tokens":8},
			"choices":[{"message":{"role":"assistant","content":"hello"}}]
		}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
		adminHandler: &AdminHandler{
			conversations: conversations.NewStore("", conversations.Settings{Enabled: true, MaxItems: 1000, MaxAgeDays: 30}),
			wsClients:     map[*adminWSClient]struct{}{},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	threads, _, total := s.adminHandler.conversations.ListThreads(conversations.ListFilter{Limit: 100})
	if total < 1 || len(threads) < 1 {
		t.Fatalf("expected at least one conversation thread, got total=%d len=%d", total, len(threads))
	}
}

func TestProxyHandlerCapturesSeparateConversationsByHeader(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"resp_conv_h","usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
		adminHandler: &AdminHandler{
			conversations: conversations.NewStore("", conversations.Settings{Enabled: true, MaxItems: 1000, MaxAgeDays: 30}),
			wsClients:     map[*adminWSClient]struct{}{},
		},
	}

	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-Conversation-ID", "conv-A")
	w1 := httptest.NewRecorder()
	s.proxyHandler(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d body=%s", w1.Code, w1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hello"}]}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Conversation-ID", "conv-B")
	w2 := httptest.NewRecorder()
	s.proxyHandler(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected second request 200, got %d body=%s", w2.Code, w2.Body.String())
	}

	threads, _, total := s.adminHandler.conversations.ListThreads(conversations.ListFilter{Limit: 100})
	if total != 2 || len(threads) != 2 {
		t.Fatalf("expected exactly 2 conversation threads, got total=%d len=%d", total, len(threads))
	}
	keys := map[string]bool{}
	for _, th := range threads {
		keys[th.ConversationKey] = true
	}
	if !keys["cid:conv-A"] || !keys["cid:conv-B"] {
		t.Fatalf("expected cid:conv-A and cid:conv-B, got %#v", keys)
	}
}

func TestProxyHandlerRetriesTransientBadGateway(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls < 3 {
			http.Error(w, `{"error":"temporary upstream failure"}`, http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"resp_retry_ok","choices":[{"message":{"role":"assistant","content":"retried ok"}}]}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after retries, got %d body=%s", w.Code, w.Body.String())
	}
	if calls != 3 {
		t.Fatalf("expected 3 upstream attempts, got %d", calls)
	}
}

func TestProxyHandlerDoesNotRetryUnauthorized(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without retry, got %d body=%s", w.Code, w.Body.String())
	}
	if calls != 1 {
		t.Fatalf("expected 1 upstream attempt, got %d", calls)
	}
}

func TestParseConversationRequestIDsPrefersHeaderAndReadsMetadata(t *testing.T) {
	h := http.Header{}
	h.Set("X-Conversation-ID", "header-conv")
	h.Set("X-Previous-Response-ID", "prev-header")
	body := []byte(`{
		"conversation_id":"payload-conv",
		"previous_response_id":"prev-payload",
		"metadata":{"conversation_id":"meta-conv"}
	}`)
	cid, prev := parseConversationRequestIDs(h, body)
	if cid != "header-conv" {
		t.Fatalf("expected header conversation id, got %q", cid)
	}
	if prev != "prev-header" {
		t.Fatalf("expected header previous response id, got %q", prev)
	}

	cid2, prev2 := parseConversationRequestIDs(nil, []byte(`{
		"metadata":{"conversation_id":"meta-only"},
		"previous_response_id":"prev-only"
	}`))
	if cid2 != "meta-only" {
		t.Fatalf("expected metadata conversation id, got %q", cid2)
	}
	if prev2 != "prev-only" {
		t.Fatalf("expected payload previous response id, got %q", prev2)
	}
}

func TestProxyHandlerStreamingRecordsUsageRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":4,\"output_tokens\":3,\"total_tokens\":7}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)

	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	summary := s.stats.Summary(time.Hour)
	if summary.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", summary.Requests)
	}
	if summary.TotalTokens != 7 {
		t.Fatalf("expected 7 total tokens, got %d", summary.TotalTokens)
	}
	if summary.PromptTokens != 4 {
		t.Fatalf("expected 4 prompt tokens, got %d", summary.PromptTokens)
	}
	if summary.CompletionTokens != 3 {
		t.Fatalf("expected 3 completion tokens, got %d", summary.CompletionTokens)
	}
}

func TestParseUsageTokensFindsNestedUsage(t *testing.T) {
	body := []byte(`{"type":"response.completed","response":{"usage":{"input_tokens":2,"output_tokens":5,"total_tokens":7}}}`)
	p, c, total := parseUsageTokens(body)
	if p != 2 || c != 5 || total != 7 {
		t.Fatalf("unexpected usage parse result p=%d c=%d t=%d", p, c, total)
	}
}

func TestParseProviderUsageMetricsExtractsTimeInfoAndCachedTokens(t *testing.T) {
	body := []byte(`{
		"usage":{
			"prompt_tokens":36,
			"completion_tokens":1,
			"total_tokens":37,
			"prompt_tokens_details":{"cached_tokens":12}
		},
		"time_info":{"prompt_time":0.002,"completion_time":0.001}
	}`)
	m := parseProviderUsageMetrics(body)
	if m.PromptCachedTokens != 12 {
		t.Fatalf("expected cached tokens 12, got %d", m.PromptCachedTokens)
	}
	if !m.HasPromptTPS || !m.HasGenTPS {
		t.Fatalf("expected both tps flags true, got prompt=%v gen=%v", m.HasPromptTPS, m.HasGenTPS)
	}
	if m.PromptTPS < 17999 || m.PromptTPS > 18001 {
		t.Fatalf("unexpected prompt tps: %f", m.PromptTPS)
	}
	if m.GenTPS < 999 || m.GenTPS > 1001 {
		t.Fatalf("unexpected gen tps: %f", m.GenTPS)
	}
}

func TestParseProviderUsageMetricsExtractsUsageTimingFields(t *testing.T) {
	body := []byte(`{
		"usage":{
			"prompt_tokens":20,
			"completion_tokens":10,
			"total_tokens":30,
			"queue_time":0.2,
			"prompt_time":0.5,
			"completion_time":0.25
		}
	}`)
	m := parseProviderUsageMetrics(body)
	if !m.HasPromptTPS || !m.HasGenTPS {
		t.Fatalf("expected prompt/gen tps from usage timings, got prompt=%v gen=%v", m.HasPromptTPS, m.HasGenTPS)
	}
	if m.PromptTPS < 39.9 || m.PromptTPS > 40.1 {
		t.Fatalf("unexpected prompt tps: %f", m.PromptTPS)
	}
	if m.GenTPS < 39.9 || m.GenTPS > 40.1 {
		t.Fatalf("unexpected gen tps: %f", m.GenTPS)
	}
	if !m.HasTotalSeconds {
		t.Fatalf("expected total seconds from usage timings")
	}
	if m.TotalSeconds < 0.949 || m.TotalSeconds > 0.951 {
		t.Fatalf("expected total seconds 0.95, got %f", m.TotalSeconds)
	}
	if m.QueueSeconds < 0.199 || m.QueueSeconds > 0.201 {
		t.Fatalf("expected queue seconds 0.2, got %f", m.QueueSeconds)
	}
}

func TestProxyHandlerUsesProviderReportedTimeInfoAndCachedTokens(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-1",
			"usage":{
				"prompt_tokens":36,
				"completion_tokens":1,
				"total_tokens":37,
				"prompt_tokens_details":{"cached_tokens":10}
			},
			"time_info":{"prompt_time":0.002,"completion_time":0.001}
		}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	summary := s.stats.Summary(time.Hour)
	if summary.PromptCachedTokens != 10 {
		t.Fatalf("expected cached prompt tokens 10, got %d", summary.PromptCachedTokens)
	}
	if summary.AvgPromptTPS < 1999 || summary.AvgPromptTPS > 2001 {
		t.Fatalf("expected prompt tps to use provider time_info then clamp to 2000, got %f", summary.AvgPromptTPS)
	}
	if summary.AvgGenerationTPS < 999 || summary.AvgGenerationTPS > 1001 {
		t.Fatalf("expected gen tps from provider time_info, got %f", summary.AvgGenerationTPS)
	}
}

func TestProxyHandlerUsesProviderTotalTimeForLatency(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-1",
			"usage":{
				"prompt_tokens":20,
				"completion_tokens":10,
				"total_tokens":30,
				"queue_time":0.2,
				"prompt_time":0.5,
				"completion_time":0.25
			}
		}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	summary := s.stats.Summary(time.Hour)
	if summary.AvgLatencyMS < 949 || summary.AvgLatencyMS > 951 {
		t.Fatalf("expected avg latency from provider total_time (950ms), got %f", summary.AvgLatencyMS)
	}
}

func TestComputePromptAndGenerationTPSUsesPhaseTimings(t *testing.T) {
	promptTPS, genTPS := computePromptAndGenerationTPS(200, 100, 2*time.Second, 12*time.Second)
	if promptTPS != 100 {
		t.Fatalf("expected prompt tps 100, got %f", promptTPS)
	}
	if genTPS != 10 {
		t.Fatalf("expected generation tps 10, got %f", genTPS)
	}
}

func TestProxyHandlerFallbackEstimatesTokensWithoutUsageNonStreaming(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"resp_1","output_text":"short reply"}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"test-provider/test-model","input":"hello world"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	summary := s.stats.Summary(time.Hour)
	if summary.TotalTokens <= 0 {
		t.Fatalf("expected fallback token estimate > 0, got %d", summary.TotalTokens)
	}
	if summary.PromptTokens <= 0 {
		t.Fatalf("expected fallback prompt token estimate > 0, got %d", summary.PromptTokens)
	}
	if summary.CompletionTokens <= 0 {
		t.Fatalf("expected fallback completion token estimate > 0, got %d", summary.CompletionTokens)
	}
}

func TestProxyHandlerFallbackEstimatesTokensWithoutUsageStreaming(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"streamed reply\"}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"test-provider/test-model","stream":true,"input":"hello world"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	summary := s.stats.Summary(time.Hour)
	if summary.TotalTokens <= 0 {
		t.Fatalf("expected fallback token estimate > 0, got %d", summary.TotalTokens)
	}
	if summary.PromptTokens <= 0 {
		t.Fatalf("expected fallback prompt token estimate > 0, got %d", summary.PromptTokens)
	}
	if summary.CompletionTokens <= 0 {
		t.Fatalf("expected fallback completion token estimate > 0, got %d", summary.CompletionTokens)
	}
}

func TestProxyHandlerAcceptsGzipRequestBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("upstream decode failed: %v", err)
		}
		if got := strings.TrimSpace(anyToString(payload["model"])); got != "test-model" {
			t.Fatalf("expected upstream model test-model, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	raw := []byte(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`)
	var compressed bytes.Buffer
	gzw := gzip.NewWriter(&compressed)
	if _, err := gzw.Write(raw); err != nil {
		t.Fatalf("gzip write failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(compressed.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestProxyHandlerAcceptsPlainJSONWithIncorrectGzipHeader(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("upstream decode failed: %v", err)
		}
		if got := strings.TrimSpace(anyToString(payload["model"])); got != "test-model" {
			t.Fatalf("expected upstream model test-model, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestProxyHandlerAcceptsZstdRequestBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("upstream decode failed: %v", err)
		}
		if got := strings.TrimSpace(anyToString(payload["model"])); got != "test-model" {
			t.Fatalf("expected upstream model test-model, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	raw := []byte(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`)
	var compressed bytes.Buffer
	zw, err := zstd.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("zstd writer failed: %v", err)
	}
	if _, err := zw.Write(raw); err != nil {
		t.Fatalf("zstd write failed: %v", err)
	}
	zw.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(compressed.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "zstd")
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestProxyHandlerAcceptsZstdRequestBodyWithoutEncodingHeader(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("upstream decode failed: %v", err)
		}
		if got := strings.TrimSpace(anyToString(payload["model"])); got != "test-model" {
			t.Fatalf("expected upstream model test-model, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	store := config.NewServerConfigStore(filepath.Join(t.TempDir(), "config.toml"), cfg)
	resolver := NewProviderResolver(store)
	s := &Server{
		store:                 store,
		resolver:              resolver,
		stats:                 NewStatsStore(100),
		providerHealthChecker: NewProviderHealthChecker(resolver, providerHealthCheckInterval),
	}

	raw := []byte(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hi"}]}`)
	var compressed bytes.Buffer
	zw, err := zstd.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("zstd writer failed: %v", err)
	}
	if _, err := zw.Write(raw); err != nil {
		t.Fatalf("zstd write failed: %v", err)
	}
	zw.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(compressed.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	// Intentionally omit Content-Encoding to match observed Codex traffic.
	w := httptest.NewRecorder()
	s.proxyHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func anyToString(v any) string {
	s, _ := v.(string)
	return s
}
