package proxy

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/conversations"
)

func TestProxyHandlerConversationCaptureIgnoresClientAcceptEncoding(t *testing.T) {
	isolateDefaultDataPaths(t)

	var upstreamAcceptEncoding string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamAcceptEncoding = strings.TrimSpace(r.Header.Get("Accept-Encoding"))
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(strings.ToLower(upstreamAcceptEncoding), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			_, _ = gz.Write([]byte(`{"id":"chatcmpl-test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"decoded reply"}}]}`))
			_ = gz.Close()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(buf.Bytes())
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"plain reply"}}]}`))
	}))
	defer upstream.Close()

	cfg := config.NewDefaultServerConfig()
	cfg.AllowLocalhostNoAuth = true
	cfg.IncomingTokens = nil
	cfg.AutoEnablePublicFreeModels = false
	cfg.AutoDetectLocalServers = false
	cfg.Providers = []config.ProviderConfig{
		{
			Name:           "test-provider",
			BaseURL:        upstream.URL + "/v1",
			APIKey:         "test-key",
			Enabled:        true,
			TimeoutSeconds: 10,
		},
	}
	s, err := NewServer(filepath.Join(t.TempDir(), "torod.toml"), cfg)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"test-provider/test-model","messages":[{"role":"user","content":"hello"}]}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, br, zstd")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(strings.ToLower(upstreamAcceptEncoding), "br") || strings.Contains(strings.ToLower(upstreamAcceptEncoding), "zstd") {
		t.Fatalf("expected proxy to strip client compression preferences, got upstream Accept-Encoding %q", upstreamAcceptEncoding)
	}

	threads, _, _ := s.conversations.ListThreads(conversations.ListFilter{})
	if len(threads) != 1 {
		t.Fatalf("expected one captured conversation thread, got %d", len(threads))
	}
	records := s.conversations.Conversation(threads[0].ConversationKey)
	if len(records) != 1 {
		t.Fatalf("expected one captured conversation record, got %d", len(records))
	}
	views := buildConversationRecordViews(records, false)
	if len(views) != 1 {
		t.Fatalf("expected one rendered conversation view, got %d", len(views))
	}
	if views[0].RequestRenderMarkdown != "hello" {
		t.Fatalf("unexpected request render %q", views[0].RequestRenderMarkdown)
	}
	if views[0].ResponseRenderMarkdown != "decoded reply" {
		t.Fatalf("unexpected response render %q", views[0].ResponseRenderMarkdown)
	}
	if strings.Contains(views[0].ResponsePayloadRaw, "\x1f\x8b") {
		snippet := views[0].ResponsePayloadRaw
		if len(snippet) > 16 {
			snippet = snippet[:16]
		}
		t.Fatalf("expected stored response payload to be decoded JSON, got gzip header in %q", snippet)
	}
}
