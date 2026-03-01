package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/llmclient"
	"github.com/spf13/cobra"
)

func TestIsValidEnvVarName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		ok   bool
	}{
		{name: "empty", in: "", ok: false},
		{name: "leading digit", in: "1OPENAI", ok: false},
		{name: "contains dash", in: "OPENAI-KEY", ok: false},
		{name: "contains space", in: "OPENAI KEY", ok: false},
		{name: "simple", in: "OPENAI_API_KEY", ok: true},
		{name: "lowercase", in: "openai_base_url", ok: true},
		{name: "leading underscore", in: "_TOKEN", ok: true},
		{name: "with digits", in: "API_KEY_2", ok: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isValidEnvVarName(tc.in); got != tc.ok {
				t.Fatalf("isValidEnvVarName(%q) = %v, want %v", tc.in, got, tc.ok)
			}
		})
	}
}

func TestShouldUseHintURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "empty", in: "", want: true},
		{name: "default with v1", in: "http://127.0.0.1:7050/v1", want: true},
		{name: "localhost no path", in: "http://localhost:7050", want: true},
		{name: "loopback no v1 path", in: "http://127.0.0.1:7050", want: true},
		{name: "non-default localhost port", in: "http://127.0.0.1:8080", want: false},
		{name: "remote host", in: "https://api.example.com", want: false},
		{name: "custom path", in: "http://127.0.0.1:7050/custom", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldUseHintURL(tc.in)
			if got != tc.want {
				t.Fatalf("shouldUseHintURL(%q)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestRunModelsHuman(t *testing.T) {
	const apiKey = "test-key"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer")); got != apiKey {
			t.Fatalf("authorization header mismatch: got %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"zeta/mini","object":"model"},{"id":"alpha/base","object":"model"}]}`))
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "toro.toml")
	if err := config.Save(cfgPath, &config.ClientConfig{
		ServerURL: strings.TrimRight(srv.URL, "/") + "/v1",
		APIKey:    apiKey,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runModels(cmd, cfgPath, false); err != nil {
		t.Fatalf("runModels: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Models: 2") {
		t.Fatalf("expected count in output, got:\n%s", got)
	}
	if !strings.Contains(got, "  - alpha/base") || !strings.Contains(got, "  - zeta/mini") {
		t.Fatalf("expected models in output, got:\n%s", got)
	}
	if strings.Index(got, "alpha/base") > strings.Index(got, "zeta/mini") {
		t.Fatalf("expected sorted models in output, got:\n%s", got)
	}
}

func TestRunModelsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"p/a","object":"model","provider":"p"},{"id":"p/b","object":"model","provider":"p"}]}`))
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "toro.toml")
	if err := config.Save(cfgPath, &config.ClientConfig{
		ServerURL: strings.TrimRight(srv.URL, "/"),
		APIKey:    "unused",
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runModels(cmd, cfgPath, true); err != nil {
		t.Fatalf("runModels json: %v", err)
	}
	var report modelsReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("decode json output: %v\n%s", err, out.String())
	}
	if report.Count != 2 {
		t.Fatalf("count = %d, want 2", report.Count)
	}
	if len(report.Models) != 2 || report.Models[0].ID != "p/a" || report.Models[1].ID != "p/b" {
		t.Fatalf("unexpected models: %+v", report.Models)
	}
}

func TestParsePingPongModelsFlag(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		a       string
		b       string
		wantErr bool
	}{
		{name: "empty", in: "", a: "", b: ""},
		{name: "one model", in: "provider/model", a: "provider/model", b: "provider/model"},
		{name: "two models", in: "a/m1,b/m2", a: "a/m1", b: "b/m2"},
		{name: "with spaces", in: " a/m1 , b/m2 ", a: "a/m1", b: "b/m2"},
		{name: "too many", in: "a,b,c", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, b, err := parsePingPongModelsFlag(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.in, err)
			}
			if a != tc.a || b != tc.b {
				t.Fatalf("parsePingPongModelsFlag(%q) => %q,%q want %q,%q", tc.in, a, b, tc.a, tc.b)
			}
		})
	}
}

func TestCodexArgDetection(t *testing.T) {
	if !codexArgsContainOSS([]string{"--oss"}) {
		t.Fatal("expected --oss detection")
	}
	if !codexArgsContainConfigKey([]string{"-c", `model_provider="tokenrouter"`}, "model_provider") {
		t.Fatal("expected model_provider config detection")
	}
	if !codexArgsContainForcedLoginMethod([]string{"--config", `forced_login_method="api"`}) {
		t.Fatal("expected forced_login_method config detection")
	}
	if !codexArgsContainModelSelection([]string{"--model", "gpt-5"}) {
		t.Fatal("expected --model detection")
	}
	if !codexArgsContainModelSelection([]string{"-c", `model="gpt-5"`}) {
		t.Fatal("expected -c model= detection")
	}
	if codexArgsContainConfigKey([]string{"-c", `model="gpt-5"`}, "model_provider") {
		t.Fatal("did not expect model_provider when only model is set")
	}
}

func TestDiscoverZeroCostModelsFromV1Models(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"p/alpha:free"},{"id":"p/beta"},{"id":"q/gamma-free"}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	models, err := newToroClient(srv.URL, "").discoverZeroCostModels()
	if err != nil {
		t.Fatalf("discoverZeroCostModels: %v", err)
	}
	if len(models) != 2 || models[0] != "p/alpha:free" || models[1] != "q/gamma-free" {
		t.Fatalf("unexpected models: %v", models)
	}
}

func TestRunPingPongWithProvidedModels(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	conversationIDs := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			mu.Lock()
			conversationIDs = append(conversationIDs, strings.TrimSpace(r.Header.Get("X-Conversation-ID")))
			mu.Unlock()
			var req struct {
				Model    string `json:"model"`
				Stream   bool   `json:"stream"`
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if !req.Stream {
				t.Fatal("expected stream=true request")
			}
			content := ""
			if len(req.Messages) >= 2 && strings.Contains(strings.ToLower(req.Messages[len(req.Messages)-1].Content), "short random question") {
				content = "What is your favorite season?"
			} else {
				mu.Lock()
				callCount++
				n := callCount
				mu.Unlock()
				content = req.Model + " reply " + strconv.Itoa(n)
			}
			writeChatStreamChunk(w, content)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "toro.toml")
	if err := config.Save(cfgPath, &config.ClientConfig{
		ServerURL: strings.TrimRight(srv.URL, "/") + "/v1",
		APIKey:    "test-key",
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runPingPong(cmd, cfgPath, "provider-a/m1,provider-b/m2", 2, "Manual starter?", "none", 10000, 1, "words", 1); err != nil {
		t.Fatalf("runPingPong: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Seed (provider-a/m1): Manual starter?") {
		t.Fatalf("missing seed output:\n%s", got)
	}
	if !strings.Contains(got, "B[2] (provider-b/m2):") || !strings.Contains(got, "A[2] (provider-a/m1):") {
		t.Fatalf("missing iteration output:\n%s", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(conversationIDs) == 0 {
		t.Fatal("expected at least one chat request")
	}
	first := conversationIDs[0]
	if first == "" {
		t.Fatal("expected non-empty X-Conversation-ID header")
	}
	if !strings.HasPrefix(first, "toroconv_") {
		t.Fatalf("expected toroconv_ prefix, got %q", first)
	}
	seen := map[string]struct{}{}
	for i := 0; i < len(conversationIDs); i++ {
		cid := strings.TrimSpace(conversationIDs[i])
		if cid == "" {
			t.Fatalf("expected non-empty X-Conversation-ID at request %d", i)
		}
		if !strings.HasPrefix(cid, "toroconv_") {
			t.Fatalf("expected toroconv_ prefix, got %q", cid)
		}
		seen[cid] = struct{}{}
	}
	if len(seen) != 2 {
		t.Fatalf("expected exactly 2 conversation IDs (A->B and B->A), got %d: %v", len(seen), conversationIDs)
	}
}

func TestRunPingPongWithManualStarter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !req.Stream {
			t.Fatal("expected stream=true request")
		}
		writeChatStreamChunk(w, req.Model+" ok")
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "toro.toml")
	if err := config.Save(cfgPath, &config.ClientConfig{
		ServerURL: strings.TrimRight(srv.URL, "/"),
		APIKey:    "test-key",
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runPingPong(cmd, cfgPath, "a/m,b/m", 1, "What is one tiny win today?", "none", 10000, 1, "words", 1); err != nil {
		t.Fatalf("runPingPong: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Starter: manual") {
		t.Fatalf("missing manual starter marker:\n%s", got)
	}
	if !strings.Contains(got, "Seed (a/m): What is one tiny win today?") {
		t.Fatalf("missing seed output:\n%s", got)
	}
}

func writeChatStreamChunk(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]any{
		"id":     "chatcmpl-test",
		"object": "chat.completion.chunk",
		"created": 1,
		"model":  "test-model",
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{
					"role":    "assistant",
					"content": content,
				},
			},
		},
	}))
	if flusher != nil {
		flusher.Flush()
	}
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestStreamChatCompletionErrorsWhenStreamIsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Stream bool `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !req.Stream {
			t.Fatal("expected stream request")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"length\"}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var out bytes.Buffer
	got, _, err := newToroClient(srv.URL, "k", llmclient.WithConversationID("test-conv")).streamChatCompletion("p/m", []pingPongMessage{{Role: "user", Content: "hi"}}, 0.7, 32, "none", "words", &out, nil)
	if err == nil {
		t.Fatalf("expected error, got content %q", got)
	}
	if !strings.Contains(err.Error(), "empty content") {
		t.Fatalf("unexpected error: %v", err)
	}
}
