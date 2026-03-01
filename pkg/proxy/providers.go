package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/provider"
)

type ModelCard = provider.ModelCard
type ProviderHTTPError = provider.HTTPError

func NewProviderClient(p config.ProviderConfig) *provider.Client {
	return provider.NewClient(p)
}

func IsProviderAuthError(err error) bool {
	return provider.IsAuthError(err)
}

func IsProviderBlocked(err error) bool {
	return provider.IsBlocked(err)
}

func IsProviderRateLimited(err error) bool {
	return provider.IsRateLimited(err)
}

func splitModelPrefix(model string) (providerName string, stripped string, ok bool) {
	return provider.SplitModelPrefix(model)
}

func normalizeModelID(model string) string {
	return provider.NormalizeModelID(model)
}

func joinProviderPath(basePath, requestPath string) string {
	return provider.JoinProviderPath(basePath, requestPath)
}

type ProviderResolver struct {
	store *config.ServerConfigStore
}

const autoProviderProbeTTL = 20 * time.Second

var autoProviderProbeFn = probeAutoProviderOnline
var autoProviderEnvLookup = os.Getenv
var llamaProcessProbeFn = probeLlamaCPPProcesses

var autoProviderProbeState struct {
	mu    sync.Mutex
	byKey map[string]autoProviderProbeResult
}

type autoProviderProbeResult struct {
	checkedAt time.Time
	online    bool
}

type llamaProcessInfo struct {
	Host string
	Port int
}

const llamaProcessProbeTTL = 20 * time.Second

var llamaProcessProbeState struct {
	mu        sync.Mutex
	checkedAt time.Time
	processes []llamaProcessInfo
}

func NewProviderResolver(store *config.ServerConfigStore) *ProviderResolver {
	return &ProviderResolver{store: store}
}

func (r *ProviderResolver) ListProviders() []config.ProviderConfig {
	cfg := r.store.Snapshot()
	out := make([]config.ProviderConfig, 0, len(cfg.Providers))
	seen := map[string]struct{}{}
	popularByName := map[string]config.ProviderConfig{}
	if popular, err := getPopularProviders(); err == nil {
		for _, p := range popular {
			popularByName[p.Name] = p.AsProviderConfig()
		}
	}
	for _, p := range cfg.Providers {
		if p.Enabled {
			presetKey := providerTypeOrName(p)
			resolved := resolveProviderWithDefaults(p, popularByName[presetKey])
			seen[resolved.Name] = struct{}{}
			out = append(out, resolved)
		}
	}
	if cfg.AutoEnablePublicFreeModels {
		popular, err := getPopularProviders()
		if err == nil {
			for _, p := range popular {
				if !p.PublicFreeNoAuth {
					continue
				}
				// Local endpoints are handled by dedicated local auto-detection.
				if p.Name == "lmstudio" || p.Name == "ollama" {
					continue
				}
				if _, ok := seen[p.Name]; ok {
					continue
				}
				candidate := p.AsProviderConfig()
				if !autoProviderOnline(candidate) {
					// Keep auto public-free providers virtually disabled until endpoint is reachable.
					continue
				}
				out = append(out, candidate)
				seen[p.Name] = struct{}{}
			}
		}
	}
	for _, p := range autoDetectedProviders(popularByName, seen, cfg.AutoEnablePublicFreeModels, cfg.AutoDetectLocalServers, cfg.ListenAddr) {
		out = append(out, p)
	}
	return out
}

func autoDetectedProviders(popularByName map[string]config.ProviderConfig, seen map[string]struct{}, autoMergeEnabled bool, autoDetectLocal bool, listenAddr string) []config.ProviderConfig {
	out := make([]config.ProviderConfig, 0, 4)
	addIfOnline := func(p config.ProviderConfig) {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		p.Enabled = true
		if p.TimeoutSeconds <= 0 {
			p.TimeoutSeconds = 60
		}
		if !autoProviderOnline(p) {
			return
		}
		seen[name] = struct{}{}
		out = append(out, p)
	}

	if autoDetectLocal {
		if lmstudio, ok := popularByName["lmstudio"]; ok {
			addIfOnline(lmstudio)
		}
		if ollama, ok := popularByName["ollama"]; ok {
			addIfOnline(ollama)
		}
		for _, p := range autoDetectedLlamaCPPProviders(listenAddr) {
			addIfOnline(p)
		}
	}
	if autoMergeEnabled {
		if cloud, ok := popularByName["ollama-cloud"]; ok {
			if key := strings.TrimSpace(autoProviderEnvLookup("OLLAMA_API_KEY")); key != "" {
				cloud.APIKey = key
				addIfOnline(cloud)
			}
		}
	}
	return out
}

func autoDetectedLlamaCPPProviders(listenAddr string) []config.ProviderConfig {
	processes := cachedLlamaCPPProcesses(listenAddr)
	if len(processes) == 0 {
		return nil
	}
	out := make([]config.ProviderConfig, 0, len(processes))
	for _, proc := range processes {
		port := proc.Port
		if port <= 0 || port > 65535 {
			continue
		}
		host := normalizeLocalProbeHost(proc.Host)
		baseURL := "http://" + host + ":" + strconv.Itoa(port) + "/v1"
		name := "llama-cpp-local-" + strconv.Itoa(port)
		out = append(out, config.ProviderConfig{
			Name:           name,
			ProviderType:   "llama-cpp-local",
			BaseURL:        baseURL,
			Enabled:        true,
			TimeoutSeconds: 60,
		})
	}
	return out
}

func cachedLlamaCPPProcesses(listenAddr string) []llamaProcessInfo {
	now := time.Now().UTC()
	llamaProcessProbeState.mu.Lock()
	if !llamaProcessProbeState.checkedAt.IsZero() && now.Sub(llamaProcessProbeState.checkedAt) < llamaProcessProbeTTL {
		cached := append([]llamaProcessInfo(nil), llamaProcessProbeState.processes...)
		llamaProcessProbeState.mu.Unlock()
		return cached
	}
	llamaProcessProbeState.mu.Unlock()

	found := []llamaProcessInfo{}
	if llamaProcessProbeFn != nil {
		found = llamaProcessProbeFn()
	}
	if selfPort, ok := localSelfProbePort(listenAddr); ok {
		filtered := make([]llamaProcessInfo, 0, len(found))
		for _, proc := range found {
			if proc.Port == selfPort {
				continue
			}
			filtered = append(filtered, proc)
		}
		found = filtered
	}
	found = dedupeAndSortLlamaProcesses(found)

	llamaProcessProbeState.mu.Lock()
	llamaProcessProbeState.checkedAt = now
	llamaProcessProbeState.processes = append([]llamaProcessInfo(nil), found...)
	llamaProcessProbeState.mu.Unlock()
	return found
}

func localSelfProbePort(listenAddr string) (int, bool) {
	host, port := splitHostPortLoose(strings.TrimSpace(listenAddr))
	if port <= 0 || port > 65535 {
		return 0, false
	}
	h := strings.ToLower(strings.TrimSpace(host))
	switch h {
	case "", "0.0.0.0", "::", "::0", "*", "localhost", "127.0.0.1", "::1":
		return port, true
	default:
		return 0, false
	}
}

func dedupeAndSortLlamaProcesses(in []llamaProcessInfo) []llamaProcessInfo {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]llamaProcessInfo, 0, len(in))
	for _, p := range in {
		host := normalizeLocalProbeHost(p.Host)
		if p.Port <= 0 || p.Port > 65535 {
			continue
		}
		key := host + ":" + strconv.Itoa(p.Port)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, llamaProcessInfo{Host: host, Port: p.Port})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Port == out[j].Port {
			return out[i].Host < out[j].Host
		}
		return out[i].Port < out[j].Port
	})
	return out
}

func normalizeLocalProbeHost(host string) string {
	h := strings.TrimSpace(host)
	if h == "" {
		return "127.0.0.1"
	}
	h = strings.Trim(h, "[]")
	switch h {
	case "0.0.0.0", "::", "::0", "::1", "*":
		return "127.0.0.1"
	}
	return h
}

var (
	llamaPortLongRE  = regexp.MustCompile(`(?:^|\s)--port(?:=|\s+)(\d{2,5})(?:\s|$)`)
	llamaPortShortRE = regexp.MustCompile(`(?:^|\s)-p\s+(\d{2,5})(?:\s|$)`)
	llamaHostLongRE  = regexp.MustCompile(`(?:^|\s)--host(?:=|\s+)([^\s]+)`)
	llamaHostShortRE = regexp.MustCompile(`(?:^|\s)-h\s+([^\s]+)`)
)

func probeLlamaCPPProcesses() []llamaProcessInfo {
	switch runtime.GOOS {
	case "linux":
		byProc := probeLlamaCPPProcessesLinux()
		byProbe := probeLlamaCPPProcessesByLocalProbe()
		return dedupeAndSortLlamaProcesses(append(byProc, byProbe...))
	case "darwin", "windows":
		return probeLlamaCPPProcessesByLocalProbe()
	default:
		return nil
	}
}

func probeLlamaCPPProcessesLinux() []llamaProcessInfo {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	found := make([]llamaProcessInfo, 0, 4)
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		pid := strings.TrimSpace(ent.Name())
		if pid == "" {
			continue
		}
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		raw, err := os.ReadFile("/proc/" + pid + "/cmdline")
		if err != nil || len(raw) == 0 {
			continue
		}
		line := strings.TrimSpace(strings.ReplaceAll(string(raw), "\x00", " "))
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		if !strings.Contains(low, "llama-server") && !strings.Contains(low, "llama_cpp.server") && !strings.Contains(low, "llama-cpp-python") {
			continue
		}
		port := 7050
		host := "127.0.0.1"
		if m := llamaPortLongRE.FindStringSubmatch(line); len(m) == 2 {
			if n, err := strconv.Atoi(m[1]); err == nil {
				port = n
			}
		} else if m := llamaPortShortRE.FindStringSubmatch(line); len(m) == 2 {
			if n, err := strconv.Atoi(m[1]); err == nil {
				port = n
			}
		}
		if m := llamaHostLongRE.FindStringSubmatch(line); len(m) == 2 {
			host = m[1]
		} else if m := llamaHostShortRE.FindStringSubmatch(line); len(m) == 2 {
			host = m[1]
		}
		found = append(found, llamaProcessInfo{Host: normalizeLocalProbeHost(host), Port: port})
	}
	return dedupeAndSortLlamaProcesses(found)
}

func probeLlamaCPPProcessesByLocalProbe() []llamaProcessInfo {
	hosts := []string{"127.0.0.1", "localhost"}
	ports := []int{7050, 8080, 8081, 8082, 8083, 8000, 1337}
	out := make([]llamaProcessInfo, 0, len(ports))
	for _, h := range hosts {
		for _, p := range ports {
			if probeLlamaCPPHostPort(h, p) {
				out = append(out, llamaProcessInfo{Host: "127.0.0.1", Port: p})
			}
		}
	}
	return dedupeAndSortLlamaProcesses(out)
}

func probeLlamaCPPHostPort(host string, port int) bool {
	if strings.TrimSpace(host) == "" || port <= 0 || port > 65535 {
		return false
	}
	base := "http://" + host + ":" + strconv.Itoa(port)
	cli := &http.Client{Timeout: 1000 * time.Millisecond}
	if probeLlamaCPPProps(cli, base) {
		return true
	}
	if probeLlamaCPPModels(cli, base) {
		return true
	}
	if probeLlamaCPPHealth(cli, base) {
		return true
	}
	return false
}

func probeLlamaCPPProps(cli *http.Client, base string) bool {
	req, err := http.NewRequest(http.MethodGet, base+"/props", nil)
	if err != nil {
		return false
	}
	resp, err := cli.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return false
	}
	if obj == nil {
		return false
	}
	if _, ok := obj["default_generation_settings"]; ok {
		return true
	}
	if _, ok := obj["model_path"]; ok {
		return true
	}
	if _, ok := obj["total_slots"]; ok {
		return true
	}
	return false
}

func probeLlamaCPPModels(cli *http.Client, base string) bool {
	req, err := http.NewRequest(http.MethodGet, base+"/v1/models", nil)
	if err != nil {
		return false
	}
	resp, err := cli.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return false
	}
	data, ok := obj["data"]
	if !ok {
		return false
	}
	arr, ok := data.([]any)
	if !ok {
		return false
	}
	if len(arr) == 0 {
		// Empty models still indicates OpenAI-compatible server is running.
		return true
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := m["id"].(string); ok && strings.TrimSpace(id) != "" {
			return true
		}
	}
	return false
}

func probeLlamaCPPHealth(cli *http.Client, base string) bool {
	req, err := http.NewRequest(http.MethodGet, base+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := cli.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	text := strings.ToLower(strings.TrimSpace(string(body)))
	server := strings.ToLower(strings.TrimSpace(resp.Header.Get("Server")))
	poweredBy := strings.ToLower(strings.TrimSpace(resp.Header.Get("X-Powered-By")))
	if strings.Contains(text, "llama") || strings.Contains(server, "llama") || strings.Contains(poweredBy, "llama") {
		return true
	}
	// Many llama.cpp builds answer "ok"/"healthy" on /health.
	if text == "ok" || text == "healthy" {
		return true
	}
	return false
}

func autoLMStudioOnline(p config.ProviderConfig) bool {
	return autoProviderOnline(p)
}

func autoProviderOnline(p config.ProviderConfig) bool {
	now := time.Now().UTC()
	key := strings.ToLower(strings.TrimSpace(p.Name)) + "|" + strings.TrimSpace(p.BaseURL)
	autoProviderProbeState.mu.Lock()
	if autoProviderProbeState.byKey == nil {
		autoProviderProbeState.byKey = map[string]autoProviderProbeResult{}
	}
	if prev, ok := autoProviderProbeState.byKey[key]; ok && !prev.checkedAt.IsZero() && now.Sub(prev.checkedAt) < autoProviderProbeTTL {
		online := prev.online
		autoProviderProbeState.mu.Unlock()
		return online
	}
	autoProviderProbeState.mu.Unlock()

	online := false
	if autoProviderProbeFn != nil {
		online = autoProviderProbeFn(p)
	}

	autoProviderProbeState.mu.Lock()
	autoProviderProbeState.byKey[key] = autoProviderProbeResult{
		checkedAt: now,
		online:    online,
	}
	autoProviderProbeState.mu.Unlock()
	return online
}

func probeAutoProviderOnline(p config.ProviderConfig) bool {
	name := strings.ToLower(strings.TrimSpace(p.Name))
	switch name {
	case "lmstudio":
		return probeLMStudioOnline(p)
	case "ollama":
		return probeOllamaOnline(p)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	_, err := NewProviderClient(p).ListModels(ctx)
	return err == nil
}

func probeLMStudioOnline(p config.ProviderConfig) bool {
	baseURL := strings.TrimSpace(p.BaseURL)
	if baseURL == "" {
		return false
	}
	u, err := neturl.Parse(baseURL)
	if err != nil || strings.TrimSpace(u.Scheme) == "" || strings.TrimSpace(u.Host) == "" {
		return false
	}
	root := &neturl.URL{Scheme: u.Scheme, Host: u.Host}
	cli := &http.Client{Timeout: 1200 * time.Millisecond}

	// Reject known Ollama signature quickly; avoids false auto-enable on :1234.
	ollamaURL := *root
	ollamaURL.Path = "/api/tags"
	if req, err := http.NewRequest(http.MethodGet, ollamaURL.String(), nil); err == nil {
		if resp, err := cli.Do(req); err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8*1024))
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				return false
			}
		}
	}

	// Require LM Studio native endpoint to be present.
	lmsURL := *root
	lmsURL.Path = "/api/v0/models"
	req, err := http.NewRequest(http.MethodGet, lmsURL.String(), nil)
	if err != nil {
		return false
	}
	resp, err := cli.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return false
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return false
	}
	switch decoded.(type) {
	case []any, map[string]any:
		return true
	default:
		return false
	}
}

func probeOllamaOnline(p config.ProviderConfig) bool {
	baseURL := strings.TrimSpace(p.BaseURL)
	if baseURL == "" {
		return false
	}
	u, err := neturl.Parse(baseURL)
	if err != nil || strings.TrimSpace(u.Scheme) == "" || strings.TrimSpace(u.Host) == "" {
		return false
	}
	root := &neturl.URL{Scheme: u.Scheme, Host: u.Host}
	cli := &http.Client{Timeout: 1200 * time.Millisecond}
	tagsURL := *root
	tagsURL.Path = "/api/tags"
	req, err := http.NewRequest(http.MethodGet, tagsURL.String(), nil)
	if err != nil {
		return false
	}
	resp, err := cli.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	var out struct {
		Models []map[string]any `json:"models"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return false
	}
	return out.Models != nil
}

func resolveProviderWithDefaults(p config.ProviderConfig, preset config.ProviderConfig) config.ProviderConfig {
	if strings.TrimSpace(p.BaseURL) == "" {
		p.BaseURL = preset.BaseURL
	}
	if strings.TrimSpace(p.ModelListURL) == "" {
		p.ModelListURL = preset.ModelListURL
	}
	if strings.TrimSpace(p.DeviceAuthURL) == "" {
		p.DeviceAuthURL = preset.DeviceAuthURL
	}
	if p.TimeoutSeconds <= 0 {
		if preset.TimeoutSeconds > 0 {
			p.TimeoutSeconds = preset.TimeoutSeconds
		} else {
			p.TimeoutSeconds = 60
		}
	}
	return p
}

func providerTypeOrName(p config.ProviderConfig) string {
	// Backward compatibility: older presets tagged local "ollama" as
	// provider_type "ollama-cloud". Keep local ollama behavior separated from
	// cloud even when stale config still carries that type.
	if strings.EqualFold(strings.TrimSpace(p.Name), "ollama") {
		return "ollama"
	}
	if strings.TrimSpace(p.ProviderType) != "" {
		return strings.TrimSpace(p.ProviderType)
	}
	return strings.TrimSpace(p.Name)
}

func (r *ProviderResolver) GetProviderByName(name string) (config.ProviderConfig, bool) {
	for _, p := range r.ListProviders() {
		if p.Name == name {
			return p, true
		}
	}
	return config.ProviderConfig{}, false
}

func (r *ProviderResolver) Resolve(model string) (config.ProviderConfig, string, error) {
	providers := r.ListProviders()
	normalizedModel := normalizeModelID(model)
	if model != "" {
		if providerName, stripped, ok := splitModelPrefix(model); ok {
			stripped = normalizeModelID(stripped)
			for _, p := range providers {
				if p.Name == providerName {
					return p, stripped, nil
				}
			}
		}
	}
	if preferredProvider := preferredProviderForUnqualifiedModel(normalizedModel, providers); preferredProvider != nil {
		return *preferredProvider, normalizedModel, nil
	}
	cfg := r.store.Snapshot()
	if cfg.DefaultProvider != "" {
		for _, p := range providers {
			if p.Name == cfg.DefaultProvider {
				return p, model, nil
			}
		}
	}
	for _, p := range providers {
		return p, normalizedModel, nil
	}
	return config.ProviderConfig{}, "", fmt.Errorf("no enabled providers configured")
}

func preferredProviderForUnqualifiedModel(model string, providers []config.ProviderConfig) *config.ProviderConfig {
	model = strings.TrimSpace(strings.ToLower(model))
	if model == "" {
		return nil
	}
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o") {
		for i := range providers {
			p := providers[i]
			name := strings.ToLower(strings.TrimSpace(p.Name))
			providerType := strings.ToLower(strings.TrimSpace(p.ProviderType))
			if name == "openai" || providerType == "openai" {
				return &providers[i]
			}
		}
	}
	return nil
}

func (r *ProviderResolver) DiscoverModels(ctx context.Context) ([]ModelCard, error) {
	providers := r.ListProviders()
	models := make([]ModelCard, 0)
	for _, p := range providers {
		cards, err := NewProviderClient(p).ListModels(ctx)
		if err != nil {
			continue
		}
		models = append(models, cards...)
	}
	return models, nil
}
