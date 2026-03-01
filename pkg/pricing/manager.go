package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lkarlslund/tokenrouter/pkg/assets"
	"github.com/lkarlslund/tokenrouter/pkg/cache"
	"github.com/lkarlslund/tokenrouter/pkg/config"
)

type ModelPricing struct {
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Currency     string    `json:"currency"`
	InputPer1M   float64   `json:"input_per_1m"`
	OutputPer1M  float64   `json:"output_per_1m"`
	DiscoveredAt time.Time `json:"discovered_at"`
	Source       string    `json:"source"`
}

type ProviderState struct {
	LastAttempt time.Time `json:"last_attempt"`
	LastUpdate  time.Time `json:"last_update"`
	LastError   string    `json:"last_error,omitempty"`
}

type Cache struct {
	UpdatedAt      time.Time                `json:"updated_at"`
	ProviderStates map[string]ProviderState `json:"provider_states"`
	Entries        map[string]ModelPricing  `json:"entries"`
}

type Manager struct {
	mu        sync.RWMutex
	path      string
	cache     Cache
	providers []config.ProviderConfig

	active     bool
	stopCh     chan struct{}
	refreshing bool

	sources []ProviderPricingSource
}

const pricingRefreshInterval = 12 * time.Hour

const defaultCerebrasPricingModelsURL = "https://api.cerebras.ai/public/v1/models"
const defaultGoogleGeminiPricingURL = "https://ai.google.dev/gemini-api/docs/pricing"
const defaultOpenCodeZenPricingURL = "https://opencode.ai/docs/zen/"

var (
	popularProvidersOnce sync.Once
	popularProvidersByID map[string]assets.PopularProvider
)

func NewManager(path string) (*Manager, error) {
	m := &Manager{path: path}
	if err := m.load(); err != nil {
		return nil, err
	}
	m.sources = []ProviderPricingSource{
		&AllFreePricingSource{},
		&OpenCodeZenPricingSource{},
		&GoogleGeminiDocsPricingSource{},
		&NvidiaNIMPricingSource{},
		&CerebrasPublicPricingSource{},
		&ModelsEndpointPricingSource{},
	}
	return m, nil
}

func (m *Manager) SetProviders(providers []config.ProviderConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers = append([]config.ProviderConfig(nil), providers...)
	if !m.active {
		m.active = true
		m.stopCh = make(chan struct{})
		go m.loop(m.stopCh)
	}
}

func (m *Manager) EnsureFreshAsync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		_ = m.Refresh(ctx)
	}()
}

func (m *Manager) Refresh(ctx context.Context) error {
	return m.refreshInternal(ctx, false)
}

func (m *Manager) ForceRefresh(ctx context.Context) error {
	return m.refreshInternal(ctx, true)
}

func (m *Manager) refreshInternal(ctx context.Context, force bool) error {
	m.mu.Lock()
	if m.refreshing {
		m.mu.Unlock()
		return nil
	}
	m.refreshing = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.refreshing = false
		m.mu.Unlock()
	}()

	m.mu.RLock()
	providers := append([]config.ProviderConfig(nil), m.providers...)
	m.mu.RUnlock()
	if len(providers) == 0 {
		return nil
	}

	now := time.Now()
	m.mu.Lock()
	if m.cache.ProviderStates == nil {
		m.cache.ProviderStates = map[string]ProviderState{}
	}
	if m.cache.Entries == nil {
		m.cache.Entries = map[string]ModelPricing{}
	}
	m.mu.Unlock()

	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		if !force && !m.providerNeedsRefreshLocked(p.Name, now) {
			continue
		}
		entries, source, err := m.fetchProviderPricing(ctx, p)
		m.mu.Lock()
		state := m.cache.ProviderStates[p.Name]
		state.LastAttempt = now
		if err != nil {
			state.LastError = err.Error()
			m.cache.ProviderStates[p.Name] = state
			m.mu.Unlock()
			continue
		}
		state.LastError = ""
		state.LastUpdate = now
		m.cache.ProviderStates[p.Name] = state
		prefix := p.Name + "/"
		for k := range m.cache.Entries {
			if strings.HasPrefix(k, prefix) {
				delete(m.cache.Entries, k)
			}
		}
		for _, e := range entries {
			e.Provider = p.Name
			e.DiscoveredAt = now
			e.Source = source
			m.cache.Entries[p.Name+"/"+e.Model] = e
		}
		m.cache.UpdatedAt = now
		m.mu.Unlock()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) providerNeedsRefreshLocked(provider string, now time.Time) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.cache.ProviderStates[provider]
	if !ok {
		return true
	}
	last := state.LastUpdate
	if state.LastAttempt.After(last) {
		last = state.LastAttempt
	}
	if last.IsZero() {
		return true
	}
	return now.Sub(last) >= pricingRefreshInterval
}

func (m *Manager) loop(stop <-chan struct{}) {
	t := time.NewTicker(pricingRefreshInterval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			_ = m.Refresh(ctx)
			cancel()
		}
	}
}

func (m *Manager) Snapshot() Cache {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := Cache{
		UpdatedAt:      m.cache.UpdatedAt,
		ProviderStates: map[string]ProviderState{},
		Entries:        map[string]ModelPricing{},
	}
	for k, v := range m.cache.ProviderStates {
		cp.ProviderStates[k] = v
	}
	for k, v := range m.cache.Entries {
		cp.Entries[k] = v
	}
	return cp
}

func (m *Manager) TouchProviderFresh(provider string, at time.Time) error {
	name := strings.TrimSpace(provider)
	if name == "" {
		return nil
	}
	when := at.UTC()
	if when.IsZero() {
		when = time.Now().UTC()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cache.ProviderStates == nil {
		m.cache.ProviderStates = map[string]ProviderState{}
	}
	state := m.cache.ProviderStates[name]
	if state.LastAttempt.Before(when) {
		state.LastAttempt = when
	}
	if state.LastUpdate.Before(when) {
		state.LastUpdate = when
	}
	state.LastError = ""
	m.cache.ProviderStates[name] = state
	if m.cache.UpdatedAt.Before(when) {
		m.cache.UpdatedAt = when
	}
	return m.saveLocked()
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = Cache{ProviderStates: map[string]ProviderState{}, Entries: map[string]ModelPricing{}}
	err := cache.LoadJSON(m.path, &m.cache)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("load pricing cache: %w", err)
	}
	if m.cache.ProviderStates == nil {
		m.cache.ProviderStates = map[string]ProviderState{}
	}
	if m.cache.Entries == nil {
		m.cache.Entries = map[string]ModelPricing{}
	}
	return nil
}

func (m *Manager) saveLocked() error {
	if err := cache.SaveJSON(m.path, m.cache); err != nil {
		return fmt.Errorf("save pricing cache: %w", err)
	}
	return nil
}

func (m *Manager) fetchProviderPricing(ctx context.Context, p config.ProviderConfig) ([]ModelPricing, string, error) {
	var errs []string
	for _, src := range m.sources {
		if !src.Match(p) {
			continue
		}
		entries, source, err := src.Fetch(ctx, p)
		if err != nil {
			errs = append(errs, src.Name()+": "+err.Error())
			continue
		}
		return entries, source, nil
	}
	if len(errs) > 0 {
		return nil, "", errors.New(strings.Join(errs, "; "))
	}
	return nil, "", fmt.Errorf("no pricing source matched provider %q", p.Name)
}

func providerTypeOrName(p config.ProviderConfig) string {
	// Backward compatibility: older presets tagged local "ollama" as
	// provider_type "ollama-cloud". Keep local ollama behavior separated from
	// cloud even when stale config still carries that type.
	if strings.EqualFold(strings.TrimSpace(p.Name), "ollama") {
		return "ollama"
	}
	if v := strings.TrimSpace(p.ProviderType); v != "" {
		return strings.ToLower(v)
	}
	return strings.ToLower(strings.TrimSpace(p.Name))
}

func loadPopularProvidersByID() map[string]assets.PopularProvider {
	popularProvidersOnce.Do(func() {
		popularProvidersByID = map[string]assets.PopularProvider{}
		popular, err := assets.LoadPopularProviders()
		if err != nil {
			return
		}
		for _, p := range popular {
			id := strings.ToLower(strings.TrimSpace(p.Name))
			if id == "" {
				continue
			}
			popularProvidersByID[id] = p
		}
	})
	return popularProvidersByID
}

func popularProviderFor(p config.ProviderConfig) (assets.PopularProvider, bool) {
	byID := loadPopularProvidersByID()
	if len(byID) == 0 {
		return assets.PopularProvider{}, false
	}
	pp, ok := byID[providerTypeOrName(p)]
	return pp, ok
}

func pricingDocsURLFor(p config.ProviderConfig, fallback string) string {
	if preset, ok := popularProviderFor(p); ok {
		if v := strings.TrimSpace(preset.PricingURL); v != "" {
			return v
		}
	}
	return fallback
}

func pricingModelsURLFor(p config.ProviderConfig, fallback string) string {
	if v := strings.TrimSpace(p.ModelListURL); v != "" {
		return v
	}
	if preset, ok := popularProviderFor(p); ok {
		if v := strings.TrimSpace(preset.PricingModelsURL); v != "" {
			return v
		}
	}
	return fallback
}

func pricingGathererFor(p config.ProviderConfig) string {
	if preset, ok := popularProviderFor(p); ok {
		return strings.ToLower(strings.TrimSpace(preset.PricingGatherer))
	}
	return ""
}

type ProviderPricingSource interface {
	Name() string
	Match(config.ProviderConfig) bool
	Fetch(context.Context, config.ProviderConfig) ([]ModelPricing, string, error)
}

type AllFreePricingSource struct{}

func (s *AllFreePricingSource) Name() string { return "allfree" }

func (s *AllFreePricingSource) Match(p config.ProviderConfig) bool {
	if pricingGathererFor(p) == "allfree" {
		return true
	}
	switch providerTypeOrName(p) {
	case "ollama", "lmstudio":
		return true
	default:
		return false
	}
}

func (s *AllFreePricingSource) Fetch(ctx context.Context, p config.ProviderConfig) ([]ModelPricing, string, error) {
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	cli := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	modelIDs, err := fetchModelIDs(ctx, cli, p)
	if err != nil {
		return nil, "", err
	}
	if len(modelIDs) == 0 {
		return nil, "", fmt.Errorf("no models returned")
	}
	out := make([]ModelPricing, 0, len(modelIDs))
	for _, id := range modelIDs {
		out = append(out, ModelPricing{
			Model:       id,
			Currency:    "USD",
			InputPer1M:  0,
			OutputPer1M: 0,
		})
	}
	return out, "allfree", nil
}

type ModelsEndpointPricingSource struct{}

func (s *ModelsEndpointPricingSource) Name() string { return "models-endpoint" }

func (s *ModelsEndpointPricingSource) Match(_ config.ProviderConfig) bool { return true }

func (s *ModelsEndpointPricingSource) Fetch(ctx context.Context, p config.ProviderConfig) ([]ModelPricing, string, error) {
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	cli := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	u, err := url.Parse(strings.TrimRight(p.BaseURL, "/"))
	if err != nil {
		return nil, "", err
	}
	u.Path = joinProviderPath(u.Path, "/v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, "", err
	}
	out := make([]ModelPricing, 0)
	multiplier := pricingMultiplierForModelsEndpointProvider(p)
	for _, item := range payload.Data {
		modelID, _ := item["id"].(string)
		if modelID == "" {
			continue
		}
		in, outp, ok := parsePricingFieldsWithMultiplier(item, multiplier)
		if !ok {
			continue
		}
		out = append(out, ModelPricing{
			Model:       modelID,
			Currency:    "USD",
			InputPer1M:  in,
			OutputPer1M: outp,
		})
	}
	return out, "v1/models", nil
}

func pricingMultiplierForModelsEndpointProvider(p config.ProviderConfig) float64 {
	// Hugging Face router pricing in /v1/models is already quoted in USD per 1M tokens.
	if providerTypeOrName(p) == "huggingface" {
		return 1
	}
	return 1_000_000
}

type OpenCodeZenPricingSource struct{}

func (s *OpenCodeZenPricingSource) Name() string { return "opencode-zen-docs" }

func (s *OpenCodeZenPricingSource) Match(p config.ProviderConfig) bool {
	if providerTypeOrName(p) == "opencode-zen" {
		return true
	}
	base := strings.ToLower(strings.TrimSpace(p.BaseURL))
	if strings.Contains(base, "opencode.ai/zen") {
		return true
	}
	if strings.Contains(base, "opencode.ai") {
		return true
	}
	return false
}

func (s *OpenCodeZenPricingSource) Fetch(ctx context.Context, p config.ProviderConfig) ([]ModelPricing, string, error) {
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	cli := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	docsURL := pricingDocsURLFor(p, defaultOpenCodeZenPricingURL)
	docsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, docsURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := cli.Do(docsReq)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, "", fmt.Errorf("docs status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, "", err
	}
	htmlDoc := string(body)
	section := extractPricingSection(htmlDoc)
	if section == "" {
		return nil, "", fmt.Errorf("pricing section not found")
	}
	rows := parseZenPricingRows(section)
	if len(rows) == 0 {
		return nil, "", fmt.Errorf("no pricing rows parsed")
	}
	modelIDs := extractZenModelIDs(htmlDoc)
	endpointModelIDs, endpointErr := fetchModelIDs(ctx, cli, p)
	if endpointErr == nil {
		modelIDs = append(modelIDs, endpointModelIDs...)
	}
	modelIDs = uniqueStrings(modelIDs)
	if len(modelIDs) == 0 {
		if endpointErr != nil {
			return nil, "", fmt.Errorf("no model IDs from docs and models endpoint failed: %w", endpointErr)
		}
		return nil, "", fmt.Errorf("no model IDs found")
	}

	index := buildModelMatchIndex(modelIDs)
	out := make([]ModelPricing, 0)
	for _, r := range rows {
		modelID, ok := index.Match(r.ModelLabel)
		if !ok {
			continue
		}
		out = append(out, ModelPricing{
			Model:       modelID,
			Currency:    "USD",
			InputPer1M:  r.InputPer1M,
			OutputPer1M: r.OutputPer1M,
		})
	}
	if len(out) == 0 {
		return nil, "", fmt.Errorf("no pricing rows matched model IDs")
	}
	return out, docsURL, nil
}

type GoogleGeminiDocsPricingSource struct{}

func (s *GoogleGeminiDocsPricingSource) Name() string { return "google-gemini-docs" }

func (s *GoogleGeminiDocsPricingSource) Match(p config.ProviderConfig) bool {
	if providerTypeOrName(p) == "google-gemini" {
		return true
	}
	base := strings.ToLower(strings.TrimSpace(p.BaseURL))
	return strings.Contains(base, "generativelanguage.googleapis.com")
}

func (s *GoogleGeminiDocsPricingSource) Fetch(ctx context.Context, p config.ProviderConfig) ([]ModelPricing, string, error) {
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	cli := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	docsURL := pricingDocsURLFor(p, defaultGoogleGeminiPricingURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, docsURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, "", fmt.Errorf("docs status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	if err != nil {
		return nil, "", err
	}
	rows := parseGeminiPricingRows(string(body))
	if len(rows) == 0 {
		return nil, "", fmt.Errorf("no gemini pricing rows parsed")
	}
	out := make([]ModelPricing, 0, len(rows))
	for _, r := range rows {
		out = append(out, ModelPricing{
			Model:       r.Model,
			Currency:    "USD",
			InputPer1M:  r.InputPer1M,
			OutputPer1M: r.OutputPer1M,
		})
	}
	return out, docsURL, nil
}

type NvidiaNIMPricingSource struct{}

func (s *NvidiaNIMPricingSource) Name() string { return "nvidia-nim-trial-pricing" }

func (s *NvidiaNIMPricingSource) Match(p config.ProviderConfig) bool {
	if providerTypeOrName(p) == "nvidia" {
		return true
	}
	base := strings.ToLower(strings.TrimSpace(p.BaseURL))
	return strings.Contains(base, "integrate.api.nvidia.com")
}

func (s *NvidiaNIMPricingSource) Fetch(ctx context.Context, p config.ProviderConfig) ([]ModelPricing, string, error) {
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	cli := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	modelIDs, err := fetchModelIDs(ctx, cli, p)
	if err != nil {
		return nil, "", err
	}
	if len(modelIDs) == 0 {
		return nil, "", fmt.Errorf("no models returned")
	}
	out := make([]ModelPricing, 0, len(modelIDs))
	for _, id := range modelIDs {
		out = append(out, ModelPricing{
			Model:       id,
			Currency:    "USD",
			InputPer1M:  0,
			OutputPer1M: 0,
		})
	}
	return out, "https://build.nvidia.com/", nil
}

type CerebrasPublicPricingSource struct{}

func (s *CerebrasPublicPricingSource) Name() string { return "cerebras-public-models" }

func (s *CerebrasPublicPricingSource) Match(p config.ProviderConfig) bool {
	if providerTypeOrName(p) == "cerebras" {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(p.Name))
	base := strings.ToLower(strings.TrimSpace(p.BaseURL))
	if strings.Contains(name, "cerebras") {
		return true
	}
	return strings.Contains(base, "api.cerebras.ai")
}

func (s *CerebrasPublicPricingSource) Fetch(ctx context.Context, p config.ProviderConfig) ([]ModelPricing, string, error) {
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	cli := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	modelsURL := pricingModelsURLFor(p, defaultCerebrasPricingModelsURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, "", fmt.Errorf("public models status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, "", err
	}
	out := make([]ModelPricing, 0, len(payload.Data))
	for _, item := range payload.Data {
		id, _ := item["id"].(string)
		if strings.TrimSpace(id) == "" {
			continue
		}
		in, outp, ok := parsePricingFields(item)
		if !ok {
			continue
		}
		out = append(out, ModelPricing{
			Model:       strings.TrimSpace(id),
			Currency:    "USD",
			InputPer1M:  in,
			OutputPer1M: outp,
		})
	}
	if len(out) == 0 {
		return nil, "", fmt.Errorf("no cerebras pricing rows parsed")
	}
	return out, modelsURL, nil
}

func parsePricingFields(item map[string]any) (inputPer1M, outputPer1M float64, ok bool) {
	return parsePricingFieldsWithMultiplier(item, 1_000_000)
}

func parsePricingFieldsWithMultiplier(item map[string]any, multiplier float64) (inputPer1M, outputPer1M float64, ok bool) {
	if multiplier <= 0 {
		multiplier = 1_000_000
	}
	if pricing, ok := item["pricing"].(map[string]any); ok {
		inTok, inOK := parseNumber(pricing["prompt"])
		outTok, outOK := parseNumber(pricing["completion"])
		if !inOK {
			inTok, inOK = parseNumber(pricing["input"])
		}
		if !outOK {
			outTok, outOK = parseNumber(pricing["output"])
		}
		if inOK || outOK {
			return inTok * multiplier, outTok * multiplier, true
		}
	}
	if inTok, outTok, nestedOK := parseProviderArrayPricing(item["providers"]); nestedOK {
		return inTok * multiplier, outTok * multiplier, true
	}
	inTok, inOK := parseNumber(item["input_cost_per_token"])
	outTok, outOK := parseNumber(item["output_cost_per_token"])
	if inOK || outOK {
		return inTok * multiplier, outTok * multiplier, true
	}
	return 0, 0, false
}

func parseProviderArrayPricing(v any) (inputPerToken float64, outputPerToken float64, ok bool) {
	providers, ok := v.([]any)
	if !ok || len(providers) == 0 {
		return 0, 0, false
	}
	bestSet := false
	bestScore := 0.0
	bestIn := 0.0
	bestOut := 0.0
	for _, raw := range providers {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		pricing, ok := m["pricing"].(map[string]any)
		if !ok {
			continue
		}
		in, inOK := parseNumber(pricing["prompt"])
		out, outOK := parseNumber(pricing["completion"])
		if !inOK {
			in, inOK = parseNumber(pricing["input"])
		}
		if !outOK {
			out, outOK = parseNumber(pricing["output"])
		}
		if !inOK && !outOK {
			continue
		}
		score := in + out
		if !bestSet || score < bestScore {
			bestSet = true
			bestScore = score
			bestIn = in
			bestOut = out
		}
	}
	if !bestSet {
		return 0, 0, false
	}
	return bestIn, bestOut, true
}

func fetchModelIDs(ctx context.Context, cli *http.Client, p config.ProviderConfig) ([]string, error) {
	u, err := url.Parse(strings.TrimRight(p.BaseURL, "/"))
	if err != nil {
		return nil, err
	}
	u.Path = joinProviderPath(u.Path, "/v1/models")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("models status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Data))
	for _, m := range payload.Data {
		if m.ID != "" {
			out = append(out, m.ID)
		}
	}
	return out, nil
}

var (
	pricingSectionRe  = regexp.MustCompile(`(?s)<h2 id="pricing".*?</h2>(.*?)</table>`)
	pricingRowRe      = regexp.MustCompile(`(?s)<tr><td>(.*?)</td><td>(.*?)</td><td>(.*?)</td><td>.*?</td><td>.*?</td></tr>`)
	modelsSectionRe   = regexp.MustCompile(`(?s)<h3 id="models".*?</h3>(.*?)</table>`)
	modelIDRowRe      = regexp.MustCompile(`(?s)<tr><td>.*?</td><td>(.*?)</td><td>.*?</td><td>.*?</td></tr>`)
	geminiModelCodeRe = regexp.MustCompile(`(?is)<em><code[^>]*>\s*(gemini-[^<\s]+)\s*</code></em>`)
	geminiInputRowRe  = regexp.MustCompile(`(?is)<tr[^>]*>\s*<td[^>]*>\s*Input price.*?</td>(.*?)</tr>`)
	geminiOutputRowRe = regexp.MustCompile(`(?is)<tr[^>]*>\s*<td[^>]*>\s*Output price.*?</td>(.*?)</tr>`)
	tdCellRe          = regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	dollarRe          = regexp.MustCompile(`\$([0-9]+(?:\.[0-9]+)?)`)
	tagRe             = regexp.MustCompile(`(?s)<[^>]+>`)
	parenRe           = regexp.MustCompile(`\s*\([^)]*\)`)
)

func extractPricingSection(doc string) string {
	m := pricingSectionRe.FindStringSubmatch(doc)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

type zenPricingRow struct {
	ModelLabel  string
	InputPer1M  float64
	OutputPer1M float64
}

func parseZenPricingRows(section string) []zenPricingRow {
	matches := pricingRowRe.FindAllStringSubmatch(section, -1)
	out := make([]zenPricingRow, 0, len(matches))
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		model := cleanText(m[1])
		in, okIn := parseDollarOrFree(m[2])
		outp, okOut := parseDollarOrFree(m[3])
		if model == "" || (!okIn && !okOut) {
			continue
		}
		out = append(out, zenPricingRow{ModelLabel: model, InputPer1M: in, OutputPer1M: outp})
	}
	return out
}

func extractZenModelIDs(doc string) []string {
	m := modelsSectionRe.FindStringSubmatch(doc)
	if len(m) < 2 {
		return nil
	}
	rows := modelIDRowRe.FindAllStringSubmatch(m[1], -1)
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		id := cleanText(row[1])
		if id == "" {
			continue
		}
		out = append(out, id)
	}
	return out
}

func cleanText(s string) string {
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

func parseDollarOrFree(s string) (float64, bool) {
	v := strings.TrimSpace(cleanText(s))
	if v == "" || v == "-" {
		return 0, false
	}
	if strings.EqualFold(v, "free") {
		return 0, true
	}
	v = strings.TrimPrefix(v, "$")
	v = strings.ReplaceAll(v, ",", "")
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

type geminiPricingRow struct {
	Model       string
	InputPer1M  float64
	OutputPer1M float64
}

func parseGeminiPricingRows(doc string) []geminiPricingRow {
	matches := geminiModelCodeRe.FindAllStringSubmatchIndex(doc, -1)
	out := make([]geminiPricingRow, 0, len(matches))
	for i, m := range matches {
		if len(m) < 4 {
			continue
		}
		model := strings.TrimSpace(cleanText(doc[m[2]:m[3]]))
		if model == "" {
			continue
		}
		blockStart := m[1]
		blockEnd := len(doc)
		if i+1 < len(matches) && len(matches[i+1]) >= 1 {
			blockEnd = matches[i+1][0]
		}
		block := doc[blockStart:blockEnd]
		in, okIn := parseGeminiRowPrice(geminiInputRowRe, block)
		outp, okOut := parseGeminiRowPrice(geminiOutputRowRe, block)
		if !okIn && !okOut {
			continue
		}
		out = append(out, geminiPricingRow{Model: model, InputPer1M: in, OutputPer1M: outp})
	}
	return out
}

func parseGeminiRowPrice(rowRe *regexp.Regexp, block string) (float64, bool) {
	m := rowRe.FindStringSubmatch(block)
	if len(m) < 2 {
		return 0, false
	}
	cells := tdCellRe.FindAllStringSubmatch(m[1], -1)
	if len(cells) == 0 {
		return 0, false
	}
	last := cleanText(cells[len(cells)-1][1])
	return parseGeminiPriceCell(last)
}

func parseGeminiPriceCell(cell string) (float64, bool) {
	v := strings.TrimSpace(cell)
	if v == "" || v == "-" {
		return 0, false
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, "free of charge") || strings.EqualFold(v, "free") {
		return 0, true
	}
	m := dollarRe.FindStringSubmatch(v)
	if len(m) < 2 {
		return 0, false
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

type modelMatchIndex struct {
	byNorm   map[string]string
	bySig    map[string]string
	modelIDs []string
}

func buildModelMatchIndex(modelIDs []string) modelMatchIndex {
	byNorm := map[string]string{}
	bySig := map[string]string{}
	for _, id := range modelIDs {
		n := normalizeName(id)
		if n != "" {
			byNorm[n] = id
		}
		sig := tokenSignature(id)
		if sig != "" {
			bySig[sig] = id
		}
	}
	return modelMatchIndex{byNorm: byNorm, bySig: bySig, modelIDs: modelIDs}
}

func (m modelMatchIndex) Match(label string) (string, bool) {
	label = strings.TrimSpace(parenRe.ReplaceAllString(label, ""))
	n := normalizeName(label)
	if id, ok := m.byNorm[n]; ok {
		return id, true
	}
	sig := tokenSignature(label)
	if id, ok := m.bySig[sig]; ok {
		return id, true
	}
	best := ""
	bestScore := 0
	for _, id := range m.modelIDs {
		nid := normalizeName(id)
		if strings.Contains(n, nid) || strings.Contains(nid, n) {
			score := len(n)
			if len(nid) < score {
				score = len(nid)
			}
			if score > bestScore {
				bestScore = score
				best = id
			}
		}
	}
	if best != "" {
		return best, true
	}
	return "", false
}

func normalizeName(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func tokenSignature(s string) string {
	s = strings.ToLower(s)
	toks := strings.FieldsFunc(s, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	if len(toks) == 0 {
		return ""
	}
	sort.Strings(toks)
	return strings.Join(toks, "|")
}

func parseNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		n = strings.TrimSpace(n)
		if n == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func joinProviderPath(basePath, requestPath string) string {
	base := path.Clean("/" + strings.TrimSpace(basePath))
	req := path.Clean("/" + strings.TrimSpace(requestPath))
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(req, "/v1/") {
		return path.Join(base, strings.TrimPrefix(req, "/v1/"))
	}
	return path.Join(base, req)
}
