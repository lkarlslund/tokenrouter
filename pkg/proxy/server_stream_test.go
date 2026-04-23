package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lkarlslund/tokenrouter/pkg/config"
)

func TestForwardStreamingRequestRelaysSSEBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization: %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"id\":\"1\"}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	s := &Server{}
	w := httptest.NewRecorder()
	provider := config.ProviderConfig{
		Name:           "test-provider",
		BaseURL:        upstream.URL + "/v1",
		APIKey:         "test-key",
		TimeoutSeconds: 10,
	}
	status, _, usage, initialLatency, err := s.forwardStreamingRequest(context.Background(), provider, "/v1/chat/completions", []byte(`{"model":"x","stream":true}`), http.Header{}, w)
	if err != nil {
		t.Fatalf("forwardStreamingRequest returned error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("unexpected status: %d", status)
	}
	if usage.TotalTokens != 0 {
		t.Fatalf("expected zero usage in simple sse stream, got %+v", usage)
	}
	if initialLatency <= 0 {
		t.Fatalf("expected positive initial latency, got %v", initialLatency)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected recorder status: %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("unexpected content-type: %q", got)
	}
	wantBody := "data: {\"id\":\"1\"}\n\ndata: [DONE]\n\n"
	if got := w.Body.String(); got != wantBody {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestForwardStreamingRequestRetriesTransientBadGatewayBeforeStreamingStarts(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			http.Error(w, `{"error":"temporary upstream failure"}`, http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"id\":\"1\"}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	s := &Server{}
	w := httptest.NewRecorder()
	provider := config.ProviderConfig{
		Name:           "test-provider",
		BaseURL:        upstream.URL + "/v1",
		APIKey:         "test-key",
		TimeoutSeconds: 10,
	}
	status, _, _, _, err := s.forwardStreamingRequest(context.Background(), provider, "/v1/chat/completions", []byte(`{"model":"x","stream":true}`), http.Header{}, w)
	if err != nil {
		t.Fatalf("forwardStreamingRequest returned error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("unexpected status: %d", status)
	}
	if calls != 3 {
		t.Fatalf("expected 3 upstream attempts, got %d", calls)
	}
	wantBody := "data: {\"id\":\"1\"}\n\ndata: [DONE]\n\n"
	if got := w.Body.String(); got != wantBody {
		t.Fatalf("unexpected body: %q", got)
	}
}
