package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/llmclient"
	"github.com/lkarlslund/tokenrouter/pkg/logutil"
	"github.com/lkarlslund/tokenrouter/pkg/version"
	"github.com/pelletier/go-toml/v2"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

func main() {
	var wrapperTokenName string
	var wrapperTTL time.Duration
	root := &cobra.Command{
		Use:   "toro",
		Short: "TokenRouter client CLI",
		Long:  "Toro is the TokenRouter client CLI. It will later support wrapping other programs.",
	}
	root.SilenceUsage = true
	root.SilenceErrors = true
	var logLevel string
	clientConfigPath := config.DefaultClientConfigPath()
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return logutil.Configure(logLevel)
	}
	root.PersistentFlags().StringVar(&clientConfigPath, "config", config.DefaultClientConfigPath(), "Client config TOML path")
	root.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Log level (trace, debug, info, warn, error, fatal)")
	root.PersistentFlags().StringVar(&wrapperTokenName, "name", "", "Temporary token display name for wrapper commands (codex, opencode, wrap)")
	root.PersistentFlags().DurationVar(&wrapperTTL, "ttl", 8*time.Hour, "Temporary token expiry duration for wrapper commands (codex, opencode, wrap)")

	var connectServerConfigPath string
	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect toro to a TokenRouter server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectTUI(cmd, clientConfigPath, connectServerConfigPath)
		},
	}
	connectCmd.Flags().StringVar(&connectServerConfigPath, "server-config", "", "Server config TOML path to read defaults from (also checks common paths)")
	root.AddCommand(connectCmd)
	setKeyCmd := &cobra.Command{
		Use:   "set-key <api_key>",
		Short: "Set and save client API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetKey(cmd, clientConfigPath, args[0])
		},
	}
	root.AddCommand(setKeyCmd)

	var statusAsJSON bool
	var statusPeriodSeconds int
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show TokenRouter server health, version, and provider quotas",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, clientConfigPath, statusAsJSON, statusPeriodSeconds)
		},
	}
	statusCmd.Flags().BoolVar(&statusAsJSON, "json", false, "Output status as JSON")
	statusCmd.Flags().IntVar(&statusPeriodSeconds, "period-seconds", 3600, "Usage/quota lookback period in seconds")
	root.AddCommand(statusCmd)

	var modelsAsJSON bool
	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "List available models from TokenRouter",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModels(cmd, clientConfigPath, modelsAsJSON)
		},
	}
	modelsCmd.Flags().BoolVar(&modelsAsJSON, "json", false, "Output models as JSON")
	root.AddCommand(modelsCmd)

	var pingPongModels string
	var pingPongTurnPairs int
	var pingPongStarter string
	var pingPongReasoning string
	var pingPongMaxTokens int
	var pingPongRuns int
	var pingPongShowTokens string
	var pingPongParallel int
	pingPongCmd := &cobra.Command{
		Use:   "pingpong",
		Short: "Run a ping-pong chat loop between two models",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPingPong(cmd, clientConfigPath, pingPongModels, pingPongTurnPairs, pingPongStarter, pingPongReasoning, pingPongMaxTokens, pingPongRuns, pingPongShowTokens, pingPongParallel)
		},
	}
	pingPongCmd.Flags().StringVar(&pingPongModels, "models", "", "Model pair as model1,model2 (or one model to use the same on both sides)")
	pingPongCmd.Flags().IntVar(&pingPongTurnPairs, "turn-pairs", 10, "Number of turn pairs (each pair is one B reply and one A reply)")
	pingPongCmd.Flags().StringVar(&pingPongStarter, "starter", "", "Starter question text (skips auto-generated question)")
	pingPongCmd.Flags().StringVar(&pingPongReasoning, "reasoning", "none", "Reasoning effort: none, low, medium, high")
	pingPongCmd.Flags().IntVar(&pingPongMaxTokens, "max-tokens", 10000, "Stop conversation after this many cumulative tokens (0 disables limit)")
	pingPongCmd.Flags().IntVar(&pingPongRuns, "runs", 1, "Number of complete ping-pong runs to execute")
	pingPongCmd.Flags().StringVar(&pingPongShowTokens, "show-tokens", "dot", "Streaming display mode: words, dot, none")
	pingPongCmd.Flags().IntVar(&pingPongParallel, "parallel", 4, "Number of runs to execute in parallel")
	root.AddCommand(pingPongCmd)

	var opencodeProviderID string
	var opencodeProviderName string
	var opencodeModel string
	var opencodeDisableOtherProviders bool
	opencodeCmd := &cobra.Command{
		Use:   "opencode [wrapper_flags] [opencode_args...]",
		Short: "Launch opencode with a temporary subordinate TokenRouter key",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpencodeWrap(cmd, clientConfigPath, wrapperTokenName, opencodeProviderID, opencodeProviderName, opencodeModel, wrapperTTL, opencodeDisableOtherProviders, args)
		},
	}
	opencodeCmd.FParseErrWhitelist.UnknownFlags = true
	opencodeCmd.Flags().SetInterspersed(false)
	opencodeCmd.Flags().StringVar(&opencodeProviderID, "provider-id", "tokenrouter", "Injected opencode provider id")
	opencodeCmd.Flags().StringVar(&opencodeProviderName, "provider-name", "TokenRouter", "Injected opencode provider display name")
	opencodeCmd.Flags().StringVar(&opencodeModel, "model", "", "Optional model to select (bare model id or provider/model)")
	opencodeCmd.Flags().BoolVar(&opencodeDisableOtherProviders, "disable-other-providers", true, "Disable all other opencode providers while wrapped")
	root.AddCommand(opencodeCmd)

	var codexModel string
	codexCmd := &cobra.Command{
		Use:   "codex [wrapper_flags] [codex_args...]",
		Short: "Launch codex-cli with a temporary subordinate TokenRouter key",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCodexWrap(cmd, clientConfigPath, wrapperTokenName, codexModel, wrapperTTL, args)
		},
	}
	codexCmd.FParseErrWhitelist.UnknownFlags = true
	codexCmd.Flags().SetInterspersed(false)
	codexCmd.Flags().StringVar(&codexModel, "model", "", "Optional model override passed via OPENAI_MODEL")
	root.AddCommand(codexCmd)

	var wrapURLEnv string
	var wrapKeyEnv string
	wrapCmd := &cobra.Command{
		Use:   "wrap [wrapper_flags] <command> [args...]",
		Short: "Run any command with a temporary subordinate TokenRouter key",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenericWrap(cmd, clientConfigPath, wrapperTokenName, wrapperTTL, wrapURLEnv, wrapKeyEnv, args)
		},
	}
	wrapCmd.Flags().SetInterspersed(false)
	wrapCmd.Flags().StringVar(&wrapURLEnv, "url-env", "OPENAI_BASE_URL", "Environment variable name for TokenRouter /v1 URL")
	wrapCmd.Flags().StringVar(&wrapKeyEnv, "key-env", "OPENAI_API_KEY", "Environment variable name for API key")
	root.AddCommand(wrapCmd)

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print toro version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version.Detailed("toro"))
		},
	})

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runConnectTUI(cmd *cobra.Command, path, explicitServerConfigPath string) error {
	cfg, err := config.LoadOrCreateClientConfig(path)
	if err != nil {
		return fmt.Errorf("load client config: %w", err)
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	out := cmd.OutOrStdout()
	hints := discoverServerConnectHints(explicitServerConfigPath)

	fmt.Fprintf(out, "Toro connect\nClient config: %s\n", path)
	if strings.TrimSpace(hints.SourcePath) != "" {
		fmt.Fprintf(out, "Using defaults from server config: %s\n", hints.SourcePath)
	}
	fmt.Fprintln(out, "Press Enter to keep current/default value.")
	fmt.Fprintln(out, "Enter '-' for API key to clear it.")

	defaultURL := strings.TrimSpace(cfg.ServerURL)
	if shouldUseHintURL(defaultURL) {
		if strings.TrimSpace(hints.ServerURL) != "" {
			defaultURL = strings.TrimSpace(hints.ServerURL)
		} else {
			defaultURL = "http://localhost:7050"
		}
	}

	serverURL, err := promptLine(reader, out, fmt.Sprintf("Remote server URL [%s]: ", defaultURL))
	if err != nil {
		return err
	}
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		cfg.ServerURL = defaultURL
	} else {
		cfg.ServerURL = strings.TrimSpace(serverURL)
	}

	apiKeyPrompt := "API key [not set]: "
	defaultKey := strings.TrimSpace(cfg.APIKey)
	if defaultKey == "" {
		defaultKey = strings.TrimSpace(hints.APIKey)
	}
	if defaultKey != "" {
		redacted := defaultKey
		if len(redacted) <= 4 {
			redacted = strings.Repeat("*", len(redacted))
		} else {
			redacted = redacted[:4] + strings.Repeat("*", len(redacted)-4)
		}
		apiKeyPrompt = fmt.Sprintf("API key [%s]: ", redacted)
	}
	apiKeyInput, err := promptLine(reader, out, apiKeyPrompt)
	if err != nil {
		return err
	}
	apiKeyInput = strings.TrimSpace(apiKeyInput)
	switch apiKeyInput {
	case "":
		cfg.APIKey = defaultKey
	case "-":
		cfg.APIKey = ""
	default:
		cfg.APIKey = apiKeyInput
	}

	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("save client config: %w", err)
	}
	fmt.Fprintln(out, "Saved.")
	if err := checkToroConnection(cfg); err != nil {
		fmt.Fprintf(out, "Connect check: failed (%v)\n", err)
	} else {
		fmt.Fprintln(out, "Connect check: OK")
	}
	return nil
}

func checkToroConnection(cfg *config.ClientConfig) error {
	if cfg == nil {
		return fmt.Errorf("empty config")
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	if base == "" {
		return fmt.Errorf("missing server URL")
	}
	u, err := neturl.Parse(base)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/v1/models"
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return nil
}

type connectHints struct {
	SourcePath string
	ServerURL  string
	APIKey     string
}

func discoverServerConnectHints(explicitPath string) connectHints {
	seen := map[string]struct{}{}
	paths := make([]string, 0, 4)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	add(explicitPath)
	add(config.DefaultServerConfigPath())
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		add(filepath.Join(home, ".config", "tokenrouter", "server.toml"))
	}
	add("server.toml")

	for _, path := range paths {
		h, err := readServerConnectHints(path)
		if err != nil {
			continue
		}
		if strings.TrimSpace(h.ServerURL) == "" && strings.TrimSpace(h.APIKey) == "" {
			continue
		}
		h.SourcePath = path
		return h
	}
	return connectHints{}
}

func readServerConnectHints(path string) (connectHints, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return connectHints{}, err
	}
	var raw struct {
		ListenAddr string `toml:"listen_addr"`
		TLS        struct {
			Enabled    bool   `toml:"enabled"`
			ListenAddr string `toml:"listen_addr"`
		} `toml:"tls"`
		IncomingTokens []struct {
			Role      string `toml:"role"`
			Key       string `toml:"key"`
			ExpiresAt string `toml:"expires_at"`
		} `toml:"incoming_tokens"`
	}
	if err := toml.Unmarshal(b, &raw); err != nil {
		return connectHints{}, err
	}
	return connectHints{
		ServerURL: connectURLFromServerConfig(raw.ListenAddr, raw.TLS.Enabled, raw.TLS.ListenAddr),
		APIKey:    bestTokenKey(raw.IncomingTokens),
	}, nil
}

func connectURLFromServerConfig(httpListen string, tlsEnabled bool, tlsListen string) string {
	scheme := "http"
	addr := strings.TrimSpace(httpListen)
	defaultPort := "7050"
	if tlsEnabled {
		scheme = "https"
		if strings.TrimSpace(tlsListen) != "" {
			addr = strings.TrimSpace(tlsListen)
		}
		defaultPort = "443"
	}
	host, port := connectHostPort(addr)
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = defaultPort
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return scheme + "://" + host + ":" + port
}

func shouldUseHintURL(serverURL string) bool {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return true
	}
	defaultURL := strings.TrimSpace(config.NewDefaultClientConfig().ServerURL)
	if strings.EqualFold(serverURL, defaultURL) {
		return true
	}
	u, err := neturl.Parse(serverURL)
	if err != nil || strings.TrimSpace(u.Host) == "" {
		return false
	}
	host, port := connectHostPort(u.Host)
	host = strings.ToLower(strings.TrimSpace(host))
	path := strings.TrimSpace(strings.TrimSuffix(u.Path, "/"))
	if host != "localhost" && host != "127.0.0.1" {
		return false
	}
	if port != "" && port != "7050" {
		return false
	}
	if path != "" && path != "/v1" {
		return false
	}
	return true
}

func connectHostPort(addr string) (string, string) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "localhost", ""
	}
	if strings.Contains(addr, "://") {
		if u, err := neturl.Parse(addr); err == nil {
			return connectHostPort(u.Host)
		}
	}
	if strings.HasPrefix(addr, ":") {
		return "localhost", strings.TrimPrefix(addr, ":")
	}
	if host, port, err := net.SplitHostPort(addr); err == nil {
		return normalizeConnectHost(host), strings.TrimSpace(port)
	}
	// Fallback for simple host or host:port without strict net.SplitHostPort support.
	if strings.Count(addr, ":") == 1 {
		parts := strings.SplitN(addr, ":", 2)
		return normalizeConnectHost(parts[0]), strings.TrimSpace(parts[1])
	}
	return normalizeConnectHost(addr), ""
}

func normalizeConnectHost(host string) string {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	switch strings.ToLower(host) {
	case "", "0.0.0.0", "::", "::0", "*":
		return "localhost"
	default:
		return host
	}
}

func bestTokenKey(tokens []struct {
	Role      string `toml:"role"`
	Key       string `toml:"key"`
	ExpiresAt string `toml:"expires_at"`
}) string {
	now := time.Now().UTC()
	bestPriority := 99
	best := ""
	for _, t := range tokens {
		key := strings.TrimSpace(t.Key)
		if key == "" {
			continue
		}
		exp := strings.TrimSpace(t.ExpiresAt)
		if exp != "" {
			ts, err := time.Parse(time.RFC3339, exp)
			if err == nil && !now.Before(ts) {
				continue
			}
		}
		p := rolePriority(strings.TrimSpace(t.Role))
		if p < bestPriority {
			bestPriority = p
			best = key
		}
	}
	return best
}

func rolePriority(role string) int {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return 0
	case "keymaster":
		return 1
	case "", "inferrer":
		return 2
	default:
		return 3
	}
}

func runSetKey(cmd *cobra.Command, path, apiKey string) error {
	cfg, err := config.LoadOrCreateClientConfig(path)
	if err != nil {
		return fmt.Errorf("load client config: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("api key cannot be empty")
	}
	cfg.APIKey = apiKey
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("save client config: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Saved key.")
	return nil
}

type statusHealth struct {
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code,omitempty"`
	Body       string `json:"body,omitempty"`
	Error      string `json:"error,omitempty"`
}

type statusVersion struct {
	Version string `json:"version,omitempty"`
	Raw     string `json:"raw,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Date    string `json:"date,omitempty"`
	Dirty   bool   `json:"dirty,omitempty"`
	Error   string `json:"error,omitempty"`
}

type statusQuotaMetric struct {
	Key            string  `json:"key,omitempty"`
	MeteredFeature string  `json:"metered_feature,omitempty"`
	Window         string  `json:"window,omitempty"`
	LeftPercent    float64 `json:"left_percent,omitempty"`
	ResetAt        string  `json:"reset_at,omitempty"`
	Unit           string  `json:"unit,omitempty"`
}

type statusProviderQuota struct {
	Provider    string              `json:"provider"`
	Status      string              `json:"status,omitempty"`
	PlanType    string              `json:"plan_type,omitempty"`
	LeftPercent float64             `json:"left_percent,omitempty"`
	ResetAt     string              `json:"reset_at,omitempty"`
	Error       string              `json:"error,omitempty"`
	Metrics     []statusQuotaMetric `json:"metrics,omitempty"`
}

type statusStats struct {
	ProvidersAvailable int                   `json:"providers_available,omitempty"`
	ProvidersOnline    int                   `json:"providers_online,omitempty"`
	ProviderQuotas     []statusProviderQuota `json:"provider_quotas,omitempty"`
	Error              string                `json:"error,omitempty"`
}

type statusReport struct {
	CheckedAt string        `json:"checked_at"`
	ServerURL string        `json:"server_url"`
	Health    statusHealth  `json:"health"`
	Version   statusVersion `json:"version"`
	Stats     statusStats   `json:"stats"`
	Models    statusModels  `json:"models"`
}

type statusModels struct {
	Status string `json:"status"`
	Count  int    `json:"count,omitempty"`
	Error  string `json:"error,omitempty"`
}

type modelsModel struct {
	ID       string `json:"id"`
	Object   string `json:"object,omitempty"`
	Provider string `json:"provider,omitempty"`
}

type modelsReport struct {
	CheckedAt string        `json:"checked_at"`
	ServerURL string        `json:"server_url"`
	Count     int           `json:"count"`
	Models    []modelsModel `json:"models"`
}

type pingPongMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type toroClient struct {
	serverBase      string
	apiKey          string
	session         llmclient.Session
	httpClient      *http.Client
}

func newToroClient(serverBase, apiKey string, opts ...llmclient.Option) *toroClient {
	return &toroClient{
		serverBase: strings.TrimSuffix(strings.TrimSpace(serverBase), "/"),
		apiKey:     strings.TrimSpace(apiKey),
		session:    llmclient.NewSession(opts...),
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func runStatus(cmd *cobra.Command, cfgPath string, asJSON bool, periodSeconds int) error {
	cfg, err := config.LoadClientConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load client config (run `toro connect` first): %w", err)
	}
	serverBase, err := deriveServerBaseURL(cfg.ServerURL)
	if err != nil {
		return err
	}
	if periodSeconds <= 0 {
		periodSeconds = 3600
	}

	report := statusReport{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		ServerURL: strings.TrimSuffix(serverBase, "/") + "/v1",
	}
	client := newToroClient(serverBase, cfg.APIKey)

	report.Health = client.checkServerHealth()

	v, s := client.readServerStatus(periodSeconds)
	report.Version = v
	report.Stats = s
	report.Models = client.readModelsStatus()

	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	printStatusReportHuman(cmd.OutOrStdout(), report)
	return nil
}

func runModels(cmd *cobra.Command, cfgPath string, asJSON bool) error {
	cfg, err := config.LoadClientConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load client config (run `toro connect` first): %w", err)
	}
	serverBase, err := deriveServerBaseURL(cfg.ServerURL)
	if err != nil {
		return err
	}
	client := newToroClient(serverBase, cfg.APIKey)
	models, err := client.fetchModels()
	if err != nil {
		return err
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
	report := modelsReport{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		ServerURL: strings.TrimSuffix(serverBase, "/") + "/v1",
		Count:     len(models),
		Models:    models,
	}
	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	printModelsReportHuman(cmd.OutOrStdout(), report)
	return nil
}

func (c *toroClient) fetchModels() ([]modelsModel, error) {
	req, err := http.NewRequest(http.MethodGet, c.serverBase+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("models endpoint error (%d): %s", resp.StatusCode, msg)
	}
	var raw struct {
		Data []modelsModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw.Data, nil
}

func printModelsReportHuman(w io.Writer, report modelsReport) {
	fmt.Fprintf(w, "Server: %s\n", report.ServerURL)
	fmt.Fprintf(w, "Checked: %s\n", report.CheckedAt)
	fmt.Fprintf(w, "Models: %d\n", report.Count)
	for _, m := range report.Models {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		fmt.Fprintf(w, "  - %s\n", id)
	}
}

type pingPongRunStats struct {
	CompletedTurnPairs int
	UsedTokens         int
	PromptTokens       int
	CompletionTokens   int
	StoppedByMaxTokens bool
	Duration           time.Duration
}

func runPingPong(cmd *cobra.Command, cfgPath, modelsFlag string, iterations int, starter, reasoning string, maxTokens int, runs int, showTokens string, parallel int) error {
	if iterations <= 0 {
		return fmt.Errorf("iterations must be > 0")
	}
	if runs <= 0 {
		return fmt.Errorf("runs must be > 0")
	}
	if parallel <= 0 {
		return fmt.Errorf("parallel must be > 0")
	}
	if maxTokens < 0 {
		return fmt.Errorf("max-tokens must be >= 0")
	}
	reasoning = strings.ToLower(strings.TrimSpace(reasoning))
	switch reasoning {
	case "", "none":
		reasoning = "none"
	case "low", "medium", "high":
	default:
		return fmt.Errorf("invalid --reasoning %q (expected one of: none, low, medium, high)", reasoning)
	}
	showTokens = strings.ToLower(strings.TrimSpace(showTokens))
	switch showTokens {
	case "", "dot":
		showTokens = "dot"
	case "words", "none":
	default:
		return fmt.Errorf("invalid --show-tokens %q (expected one of: words, dot, none)", showTokens)
	}
	cfg, err := config.LoadClientConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load client config (run `toro connect` first): %w", err)
	}
	serverBase, err := deriveServerBaseURL(cfg.ServerURL)
	if err != nil {
		return err
	}
	client := newToroClient(serverBase, cfg.APIKey)
	modelA, modelB, err := client.selectPingPongModels(modelsFlag)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Server: %s\n", strings.TrimSuffix(serverBase, "/")+"/v1")
	fmt.Fprintf(out, "Models: A=%s, B=%s\n", modelA, modelB)
	fmt.Fprintf(out, "Turn pairs: %d\n", iterations)
	fmt.Fprintf(out, "Runs: %d\n", runs)
	fmt.Fprintf(out, "Parallel: %d\n", parallel)
	if maxTokens > 0 {
		fmt.Fprintf(out, "Max tokens: %d\n", maxTokens)
	}
	if parallel == 1 {
		for run := 1; run <= runs; run++ {
			stats, err := runPingPongOnce(cmd, client, modelA, modelB, iterations, starter, reasoning, maxTokens, showTokens, true)
			if err != nil {
				return fmt.Errorf("run %d: %w", run, err)
			}
			fmt.Fprintf(out, "Run %d/%d: turns=%d tokens=%d pp/s=%.1f tg/s=%.1f duration=%s%s\n", run, runs, stats.CompletedTurnPairs, stats.UsedTokens, ratePerSecond(stats.PromptTokens, stats.Duration), ratePerSecond(stats.CompletionTokens, stats.Duration), stats.Duration.Round(time.Millisecond), map[bool]string{true: " stop=max-tokens", false: ""}[stats.StoppedByMaxTokens])
		}
		return nil
	}
	if parallel > runs {
		parallel = runs
	}
	type runResult struct {
		index int
		stats pingPongRunStats
		err   error
	}
	jobs := make(chan int, runs)
	results := make(chan runResult, runs)
	for i := 1; i <= runs; i++ {
		jobs <- i
	}
	close(jobs)
	var wg sync.WaitGroup
	for w := 0; w < parallel; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				stats, err := runPingPongOnce(cmd, client, modelA, modelB, iterations, starter, reasoning, maxTokens, "none", false)
				results <- runResult{index: idx, stats: stats, err: err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	var firstErr error
	for rr := range results {
		if rr.err != nil && firstErr == nil {
			firstErr = fmt.Errorf("run %d: %w", rr.index, rr.err)
		}
		if rr.err != nil {
			fmt.Fprintf(out, "Run %d/%d: error=%v\n", rr.index, runs, rr.err)
			continue
		}
		fmt.Fprintf(out, "Run %d/%d: turns=%d tokens=%d pp/s=%.1f tg/s=%.1f duration=%s%s\n", rr.index, runs, rr.stats.CompletedTurnPairs, rr.stats.UsedTokens, ratePerSecond(rr.stats.PromptTokens, rr.stats.Duration), ratePerSecond(rr.stats.CompletionTokens, rr.stats.Duration), rr.stats.Duration.Round(time.Millisecond), map[bool]string{true: " stop=max-tokens", false: ""}[rr.stats.StoppedByMaxTokens])
	}
	return firstErr
}

func runPingPongOnce(cmd *cobra.Command, client *toroClient, modelA, modelB string, iterations int, starter, reasoning string, maxTokens int, showTokens string, verbose bool) (pingPongRunStats, error) {
	started := time.Now()
	out := cmd.OutOrStdout()
	conversationABID, err := randomConversationID()
	if err != nil {
		return pingPongRunStats{}, fmt.Errorf("generate A->B conversation id: %w", err)
	}
	conversationBAID, err := randomConversationID()
	if err != nil {
		return pingPongRunStats{}, fmt.Errorf("generate B->A conversation id: %w", err)
	}
	clientAB := newToroClient(client.serverBase, client.apiKey, llmclient.WithConversationID(conversationABID))
	clientBA := newToroClient(client.serverBase, client.apiKey, llmclient.WithConversationID(conversationBAID))
	usedTokens := 0
	promptTokens := 0
	completionTokens := 0
	turns := 0
	stopped := false

	seedQuestion := compactPingPongText(starter)
	if seedQuestion == "" {
		seedPrompt := "Give me a short random question. Return only the question."
		if verbose {
			fmt.Fprintf(out, "Seed (%s): ", modelA)
		}
		seedQuestion, _, err := clientAB.streamChatCompletion(modelA, []pingPongMessage{
			{Role: "system", Content: "You produce concise, random conversation starters. Do not output reasoning. Output only the final question text."},
			{Role: "user", Content: seedPrompt},
		}, 0.9, 256, reasoning, showTokens, out, cmd.ErrOrStderr())
		if verbose {
			fmt.Fprintln(out)
		}
		if err != nil {
			return pingPongRunStats{}, fmt.Errorf("seed question from %s: %w", modelA, err)
		}
		seedQuestion = compactPingPongText(seedQuestion)
		p := estimatePromptTokenCount([]pingPongMessage{
			{Role: "system", Content: "You produce concise, random conversation starters. Do not output reasoning. Output only the final question text."},
			{Role: "user", Content: seedPrompt},
		})
		c := estimateCompletionTokenCount(seedQuestion)
		promptTokens += p
		completionTokens += c
		usedTokens += p + c
	} else if verbose {
		fmt.Fprintf(out, "Starter: manual\n")
		fmt.Fprintf(out, "Seed (%s): %s\n", modelA, seedQuestion)
	}
	if seedQuestion == "" {
		return pingPongRunStats{}, fmt.Errorf("seed question from %s was empty (provider returned no final content)", modelA)
	}

	convA := []pingPongMessage{
		{Role: "system", Content: "You are participant A in a ping-pong conversation. Reply with one or two short sentences."},
	}
	convB := []pingPongMessage{
		{Role: "system", Content: "You are participant B in a ping-pong conversation. Reply with one or two short sentences."},
	}
	current := seedQuestion
	for i := 1; i <= iterations; i++ {
		if maxTokens > 0 && usedTokens >= maxTokens {
			stopped = true
			break
		}
		convB = append(convB, pingPongMessage{Role: "user", Content: current})
		if verbose {
			fmt.Fprintf(out, "B[%d] (%s): ", i, modelB)
		}
		replyB, _, err := clientAB.streamChatCompletion(modelB, convB, 0.7, 256, reasoning, showTokens, out, cmd.ErrOrStderr())
		if verbose {
			fmt.Fprintln(out)
		}
		if err != nil {
			return pingPongRunStats{}, fmt.Errorf("iteration %d model %s: %w", i, modelB, err)
		}
		replyB = compactPingPongText(replyB)
		if replyB == "" {
			return pingPongRunStats{}, fmt.Errorf("iteration %d model %s returned empty content", i, modelB)
		}
		convB = append(convB, pingPongMessage{Role: "assistant", Content: replyB})
		convB = trimPingPongConversation(convB, 14)
		p := estimatePromptTokenCount(convB[:len(convB)-1])
		c := estimateCompletionTokenCount(replyB)
		promptTokens += p
		completionTokens += c
		usedTokens += p + c

		if maxTokens > 0 && usedTokens >= maxTokens {
			stopped = true
			break
		}
		convA = append(convA, pingPongMessage{Role: "user", Content: replyB})
		if verbose {
			fmt.Fprintf(out, "A[%d] (%s): ", i, modelA)
		}
		replyA, _, err := clientBA.streamChatCompletion(modelA, convA, 0.7, 256, reasoning, showTokens, out, cmd.ErrOrStderr())
		if verbose {
			fmt.Fprintln(out)
		}
		if err != nil {
			return pingPongRunStats{}, fmt.Errorf("iteration %d model %s: %w", i, modelA, err)
		}
		replyA = compactPingPongText(replyA)
		if replyA == "" {
			return pingPongRunStats{}, fmt.Errorf("iteration %d model %s returned empty content", i, modelA)
		}
		convA = append(convA, pingPongMessage{Role: "assistant", Content: replyA})
		convA = trimPingPongConversation(convA, 14)
		p = estimatePromptTokenCount(convA[:len(convA)-1])
		c = estimateCompletionTokenCount(replyA)
		promptTokens += p
		completionTokens += c
		usedTokens += p + c
		current = replyA
		turns++
	}
	if verbose && stopped {
		fmt.Fprintf(out, "Stopped early: reached max token budget (%d/%d)\n", usedTokens, maxTokens)
	}
	return pingPongRunStats{
		CompletedTurnPairs: turns,
		UsedTokens:         usedTokens,
		PromptTokens:       promptTokens,
		CompletionTokens:   completionTokens,
		StoppedByMaxTokens: stopped,
		Duration:           time.Since(started),
	}, nil
}

func parsePingPongModelsFlag(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", nil
	}
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		items = append(items, p)
	}
	if len(items) == 0 {
		return "", "", fmt.Errorf("models flag is empty")
	}
	if len(items) > 2 {
		return "", "", fmt.Errorf("models flag supports at most two models")
	}
	if len(items) == 1 {
		return items[0], items[0], nil
	}
	return items[0], items[1], nil
}

func (c *toroClient) selectPingPongModels(modelsFlag string) (string, string, error) {
	a, b, err := parsePingPongModelsFlag(modelsFlag)
	if err != nil {
		return "", "", err
	}
	if a != "" {
		return a, b, nil
	}
	auto, err := c.discoverZeroCostModels()
	if err != nil {
		return "", "", err
	}
	if len(auto) == 0 {
		return "", "", fmt.Errorf("could not auto-detect zero-cost models; set --models model1,model2")
	}
	if len(auto) == 1 {
		return auto[0], auto[0], nil
	}
	return auto[0], auto[1], nil
}

func (c *toroClient) discoverZeroCostModels() ([]string, error) {
	models, err := c.fetchModels()
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	out := make([]string, 0, len(models))
	for _, m := range models {
		id := strings.TrimSpace(m.ID)
		if id == "" || !looksLikeFreeModel(id) {
			continue
		}
		if _, exists := set[id]; exists {
			continue
		}
		set[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func looksLikeFreeModel(modelID string) bool {
	id := strings.ToLower(strings.TrimSpace(modelID))
	if id == "" {
		return false
	}
	return strings.Contains(id, ":free") || strings.Contains(id, "/free") || strings.Contains(id, "-free") || strings.Contains(id, ".free")
}

func (c *toroClient) streamChatCompletion(model string, messages []pingPongMessage, temperature float64, maxTokens int, reasoning, showTokens string, tokenOut, spinnerOut io.Writer) (string, int, error) {
	cfg := openai.DefaultConfig(c.apiKey)
	cfg.BaseURL = c.serverBase + "/v1"
	cfg.HTTPClient = &http.Client{
		Transport: c.session.WrapRoundTripper(http.DefaultTransport),
	}
	client := openai.NewClientWithConfig(cfg)
	stopSpinner := startWaitingSpinner(spinnerOut)
	defer stopSpinner()

	reqMessages := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		reqMessages = append(reqMessages, openai.ChatCompletionMessage{
			Role:    strings.TrimSpace(m.Role),
			Content: m.Content,
		})
	}
	req := openai.ChatCompletionRequest{
		Model:       strings.TrimSpace(model),
		Messages:    reqMessages,
		Stream:      true,
		Temperature: float32(temperature),
	}
	if strings.TrimSpace(reasoning) != "" && strings.TrimSpace(reasoning) != "none" {
		req.ReasoningEffort = strings.TrimSpace(reasoning)
	}
	if maxTokens > 0 {
		req.MaxCompletionTokens = maxTokens
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return "", 0, err
	}
	defer stream.Close()

	var b strings.Builder
	lastFinish := ""
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", 0, err
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if fr := strings.TrimSpace(string(chunk.Choices[0].FinishReason)); fr != "" {
			lastFinish = fr
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		stopSpinner()
		b.WriteString(delta)
		if tokenOut != nil {
			switch showTokens {
			case "words":
				_, _ = io.WriteString(tokenOut, delta)
			case "dot":
				_, _ = io.WriteString(tokenOut, ".")
			case "none":
			default:
				_, _ = io.WriteString(tokenOut, ".")
			}
		}
	}
	out := strings.TrimSpace(b.String())
	if out != "" {
		return out, estimateTokenCount(messages, out), nil
	}

	if lastFinish != "" {
		return "", 0, fmt.Errorf("chat completion returned empty content (finish_reason=%s)", lastFinish)
	}
	return "", 0, fmt.Errorf("chat completion returned empty content")
}

func estimateTokenCount(messages []pingPongMessage, reply string) int {
	totalChars := len(strings.TrimSpace(reply))
	for _, m := range messages {
		totalChars += len(strings.TrimSpace(m.Content))
	}
	if totalChars <= 0 {
		return 0
	}
	// Rough heuristic used only for pingpong budget control.
	return (totalChars + 3) / 4
}

func estimatePromptTokenCount(messages []pingPongMessage) int {
	totalChars := 0
	for _, m := range messages {
		totalChars += len(strings.TrimSpace(m.Content))
	}
	if totalChars <= 0 {
		return 0
	}
	return (totalChars + 3) / 4
}

func estimateCompletionTokenCount(reply string) int {
	totalChars := len(strings.TrimSpace(reply))
	if totalChars <= 0 {
		return 0
	}
	return (totalChars + 3) / 4
}

func ratePerSecond(tokens int, d time.Duration) float64 {
	if tokens <= 0 || d <= 0 {
		return 0
	}
	return float64(tokens) / d.Seconds()
}

func startWaitingSpinner(w io.Writer) func() {
	if !isTerminalWriter(w) {
		return func() {}
	}

	model := newWaitSpinnerModel()
	program := tea.NewProgram(
		model,
		tea.WithOutput(w),
		tea.WithInput(os.Stdin),
		tea.WithoutSignals(),
	)

	done := make(chan struct{})
	go func() {
		_, _ = program.Run()
		close(done)
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			program.Quit()
			<-done
			_, _ = io.WriteString(w, "\r\033[2K")
		})
	}
}

type waitSpinnerModel struct {
	idx int
}

func newWaitSpinnerModel() waitSpinnerModel {
	return waitSpinnerModel{}
}

func (m waitSpinnerModel) Init() tea.Cmd {
	return tickSpinner()
}

type spinnerTickMsg struct{}

func tickSpinner() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return spinnerTickMsg{} })
}

func (m waitSpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case spinnerTickMsg:
		m.idx = (m.idx + 1) % 4
		return m, tickSpinner()
	default:
		return m, nil
	}
}

func (m waitSpinnerModel) View() tea.View {
	frames := []string{"|", "/", "-", `\`}
	return tea.NewView("\r" + frames[m.idx] + " waiting...")
}

func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func trimPingPongConversation(messages []pingPongMessage, maxTail int) []pingPongMessage {
	if maxTail <= 0 || len(messages) == 0 {
		return messages
	}
	start := 0
	head := []pingPongMessage{}
	if messages[0].Role == "system" {
		head = append(head, messages[0])
		start = 1
	}
	if len(messages)-start <= maxTail {
		return messages
	}
	tail := messages[len(messages)-maxTail:]
	out := make([]pingPongMessage, 0, len(head)+len(tail))
	out = append(out, head...)
	out = append(out, tail...)
	return out
}

func compactPingPongText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

func (c *toroClient) checkServerHealth() statusHealth {
	u := c.serverBase + "/healthz"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return statusHealth{Error: err.Error()}
	}
	client := c.httpClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return statusHealth{Error: err.Error()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	body := strings.TrimSpace(string(b))
	return statusHealth{
		OK:         resp.StatusCode == http.StatusOK,
		StatusCode: resp.StatusCode,
		Body:       body,
	}
}

func (c *toroClient) readServerStatus(periodSeconds int) (statusVersion, statusStats) {
	path := "/v1/status?period_seconds=" + strconv.Itoa(periodSeconds)
	req, err := http.NewRequest(http.MethodGet, c.serverBase+path, nil)
	if err != nil {
		msg := err.Error()
		return statusVersion{Error: msg}, statusStats{Error: msg}
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		msg := err.Error()
		return statusVersion{Error: msg}, statusStats{Error: msg}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		msg := fmt.Sprintf("status endpoint error (%d): %s", resp.StatusCode, strings.TrimSpace(string(b)))
		return statusVersion{Error: msg}, statusStats{Error: msg}
	}

	var raw struct {
		Version string `json:"version"`
		Raw     string `json:"raw"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
		Dirty   bool   `json:"dirty"`

		ProvidersAvailable int `json:"providers_available"`
		ProvidersOnline    int `json:"providers_online"`
		ProviderQuotas     map[string]struct {
			Provider    string  `json:"provider"`
			Status      string  `json:"status"`
			PlanType    string  `json:"plan_type"`
			LeftPercent float64 `json:"left_percent"`
			ResetAt     string  `json:"reset_at"`
			Error       string  `json:"error"`
			Metrics     []struct {
				Key            string  `json:"key"`
				MeteredFeature string  `json:"metered_feature"`
				Window         string  `json:"window"`
				LeftPercent    float64 `json:"left_percent"`
				ResetAt        string  `json:"reset_at"`
				Unit           string  `json:"unit"`
			} `json:"metrics"`
		} `json:"provider_quotas"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		msg := err.Error()
		return statusVersion{Error: msg}, statusStats{Error: msg}
	}

	version := statusVersion{
		Version: strings.TrimSpace(raw.Version),
		Raw:     strings.TrimSpace(raw.Raw),
		Commit:  strings.TrimSpace(raw.Commit),
		Date:    strings.TrimSpace(raw.Date),
		Dirty:   raw.Dirty,
	}
	stats := statusStats{
		ProvidersAvailable: raw.ProvidersAvailable,
		ProvidersOnline:    raw.ProvidersOnline,
		ProviderQuotas:     make([]statusProviderQuota, 0, len(raw.ProviderQuotas)),
	}
	for providerName, q := range raw.ProviderQuotas {
		item := statusProviderQuota{
			Provider:    strings.TrimSpace(providerName),
			Status:      strings.TrimSpace(q.Status),
			PlanType:    strings.TrimSpace(q.PlanType),
			LeftPercent: q.LeftPercent,
			ResetAt:     strings.TrimSpace(q.ResetAt),
			Error:       strings.TrimSpace(q.Error),
			Metrics:     make([]statusQuotaMetric, 0, len(q.Metrics)),
		}
		if strings.TrimSpace(q.Provider) != "" {
			item.Provider = strings.TrimSpace(q.Provider)
		}
		for _, m := range q.Metrics {
			item.Metrics = append(item.Metrics, statusQuotaMetric{
				Key:            strings.TrimSpace(m.Key),
				MeteredFeature: strings.TrimSpace(m.MeteredFeature),
				Window:         strings.TrimSpace(m.Window),
				LeftPercent:    m.LeftPercent,
				ResetAt:        strings.TrimSpace(m.ResetAt),
				Unit:           strings.TrimSpace(m.Unit),
			})
		}
		sort.Slice(item.Metrics, func(i, j int) bool {
			if item.Metrics[i].MeteredFeature == item.Metrics[j].MeteredFeature {
				return item.Metrics[i].Window < item.Metrics[j].Window
			}
			return item.Metrics[i].MeteredFeature < item.Metrics[j].MeteredFeature
		})
		stats.ProviderQuotas = append(stats.ProviderQuotas, item)
	}
	sort.Slice(stats.ProviderQuotas, func(i, j int) bool {
		return stats.ProviderQuotas[i].Provider < stats.ProviderQuotas[j].Provider
	})
	return version, stats
}

func printStatusReportHuman(w io.Writer, report statusReport) {
	healthText := "down"
	if report.Health.OK {
		healthText = "ok"
	}
	fmt.Fprintf(w, "Server: %s\n", report.ServerURL)
	fmt.Fprintf(w, "Checked: %s\n", report.CheckedAt)
	if strings.TrimSpace(report.Health.Error) != "" {
		fmt.Fprintf(w, "Health: %s (%s)\n", healthText, report.Health.Error)
	} else {
		body := strings.TrimSpace(report.Health.Body)
		if body != "" {
			fmt.Fprintf(w, "Health: %s (status=%d, body=%q)\n", healthText, report.Health.StatusCode, body)
		} else {
			fmt.Fprintf(w, "Health: %s (status=%d)\n", healthText, report.Health.StatusCode)
		}
	}

	if strings.TrimSpace(report.Version.Error) != "" {
		fmt.Fprintf(w, "Version: unavailable (%s)\n", report.Version.Error)
	} else {
		version := strings.TrimSpace(report.Version.Version)
		if version == "" {
			version = "unknown"
		}
		if strings.TrimSpace(report.Version.Commit) != "" {
			commit := report.Version.Commit
			if len(commit) > 12 {
				commit = commit[:12]
			}
			fmt.Fprintf(w, "Version: %s (commit=%s)\n", version, commit)
		} else {
			fmt.Fprintf(w, "Version: %s\n", version)
		}
	}

	if strings.TrimSpace(report.Stats.Error) != "" {
		fmt.Fprintf(w, "Providers: unavailable (%s)\n", report.Stats.Error)
		if strings.TrimSpace(report.Models.Error) != "" {
			fmt.Fprintf(w, "Models: unavailable (%s)\n", report.Models.Error)
		} else {
			fmt.Fprintf(w, "Models: %d online / %d available\n", report.Models.Count, report.Models.Count)
		}
		fmt.Fprintln(w, "Quota: unavailable")
	} else {
		fmt.Fprintf(w, "Providers: %d online / %d available\n", report.Stats.ProvidersOnline, report.Stats.ProvidersAvailable)
		if strings.TrimSpace(report.Models.Error) != "" {
			fmt.Fprintf(w, "Models: unavailable (%s)\n", report.Models.Error)
		} else {
			fmt.Fprintf(w, "Models: %d online / %d available\n", report.Models.Count, report.Models.Count)
		}
		if len(report.Stats.ProviderQuotas) == 0 {
			fmt.Fprintln(w, "Quota: no data")
		} else {
			fmt.Fprintln(w, "Quota:")
			for _, p := range report.Stats.ProviderQuotas {
				status := strings.TrimSpace(p.Status)
				if status == "" {
					status = "unknown"
				}
				line := fmt.Sprintf("  - %s: status=%s", p.Provider, status)
				if p.LeftPercent > 0 {
					line += fmt.Sprintf(", left=%.1f%%", p.LeftPercent)
				}
				if strings.TrimSpace(p.ResetAt) != "" {
					line += ", reset=" + p.ResetAt
				}
				if strings.TrimSpace(p.Error) != "" {
					line += ", error=" + p.Error
				}
				fmt.Fprintln(w, line)
				if len(p.Metrics) > 0 {
					for _, m := range p.Metrics {
						label := strings.TrimSpace(m.MeteredFeature)
						if label == "" {
							label = "quota"
						}
						if strings.TrimSpace(m.Window) != "" {
							label += "/" + m.Window
						}
						sub := fmt.Sprintf("      * %s: %.1f%% left", label, m.LeftPercent)
						if strings.TrimSpace(m.ResetAt) != "" {
							sub += ", reset=" + m.ResetAt
						}
						fmt.Fprintln(w, sub)
					}
				}
			}
		}
	}
}

func (c *toroClient) readModelsStatus() statusModels {
	models, err := c.fetchModels()
	if err != nil {
		return statusModels{
			Status: "unavailable",
			Error:  err.Error(),
		}
	}
	return statusModels{
		Status: "ok",
		Count:  len(models),
	}
}

func promptLine(reader *bufio.Reader, out io.Writer, prompt string) (string, error) {
	fmt.Fprint(out, prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		if len(line) == 0 {
			return "", err
		}
	}
	return strings.TrimRight(line, "\r\n"), nil
}

type accessTokenItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Role        string `json:"role,omitempty"`
	RedactedKey string `json:"redacted_key"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

func runOpencodeWrap(cmd *cobra.Command, cfgPath, tokenName, providerID, providerName, model string, ttl time.Duration, disableOtherProviders bool, opencodeArgs []string) error {
	cfg, err := config.LoadClientConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load client config (run `toro connect` first): %w", err)
	}
	serverBase, err := deriveServerBaseURL(cfg.ServerURL)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be > 0")
	}
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return fmt.Errorf("provider-id cannot be empty")
	}
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "TokenRouter"
	}
	bearer := strings.TrimSpace(cfg.APIKey)
	if bearer == "" {
		if !isLoopbackServerBase(serverBase) {
			return fmt.Errorf("client api key is required for non-localhost server (set with: toro connect)")
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "No client API key configured; using localhost no-auth admin access to create a temporary token.")
	}
	key, err := randomTemporaryKey()
	if err != nil {
		return fmt.Errorf("generate temporary key: %w", err)
	}
	if strings.TrimSpace(tokenName) == "" {
		tokenName = "toro-opencode-" + time.Now().UTC().Format("20060102-150405")
	}
	expiresAt := time.Now().UTC().Add(ttl).Format(time.RFC3339)

	before, err := fetchAccessTokens(serverBase, bearer)
	if err != nil {
		return fmt.Errorf("list access tokens before create: %w", err)
	}
	beforeIDs := map[string]struct{}{}
	for _, t := range before {
		beforeIDs[strings.TrimSpace(t.ID)] = struct{}{}
	}
	if err := createAccessToken(serverBase, bearer, tokenName, key, "inferrer", expiresAt); err != nil {
		return fmt.Errorf("create temporary access token: %w", err)
	}
	tmpID, err := findCreatedTokenID(serverBase, bearer, beforeIDs, tokenName, expiresAt)
	if err != nil {
		return fmt.Errorf("locate temporary access token id: %w", err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Created temporary token %q (id=%s, expires=%s)\n", tokenName, tmpID, expiresAt)

	cleanup := func() {
		if err := deleteAccessToken(serverBase, bearer, tmpID); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to delete temporary token %q (id=%s): %v\n", tokenName, tmpID, err)
			return
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Deleted temporary token %q (id=%s)\n", tokenName, tmpID)
	}
	defer cleanup()

	configContent, err := buildOpencodeConfigContent(serverBase, key, providerID, providerName, model)
	if err != nil {
		return err
	}
	proc := exec.Command("opencode", opencodeArgs...)
	proc.Stdin = cmd.InOrStdin()
	proc.Stdout = cmd.OutOrStdout()
	proc.Stderr = cmd.ErrOrStderr()
	env := filteredEnv([]string{"OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT"})
	if disableOtherProviders {
		isolationRoot, err := os.MkdirTemp("", "toro-opencode-home-*")
		if err != nil {
			return fmt.Errorf("create temporary opencode isolation directory: %w", err)
		}
		defer os.RemoveAll(isolationRoot)
		if err := os.MkdirAll(isolationRoot, 0o755); err != nil {
			return fmt.Errorf("prepare temporary opencode isolation directory: %w", err)
		}

		tmp, err := os.CreateTemp("", "toro-opencode-config-*.json")
		if err != nil {
			return fmt.Errorf("create temporary opencode config: %w", err)
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.WriteString(configContent); err != nil {
			tmp.Close()
			return fmt.Errorf("write temporary opencode config: %w", err)
		}
		if err := tmp.Close(); err != nil {
			return fmt.Errorf("close temporary opencode config: %w", err)
		}

		xdgConfigHome := isolationRoot + "/config"
		xdgDataHome := isolationRoot + "/data"
		xdgCacheHome := isolationRoot + "/cache"
		if err := os.MkdirAll(xdgConfigHome, 0o755); err != nil {
			return fmt.Errorf("create temporary XDG config dir: %w", err)
		}
		if err := os.MkdirAll(xdgDataHome, 0o755); err != nil {
			return fmt.Errorf("create temporary XDG data dir: %w", err)
		}
		if err := os.MkdirAll(xdgCacheHome, 0o755); err != nil {
			return fmt.Errorf("create temporary XDG cache dir: %w", err)
		}

		env = append(env, "OPENCODE_CONFIG="+tmp.Name())
		env = append(env, "OPENCODE_DISABLE_PROJECT_CONFIG=1")
		env = append(env, "XDG_CONFIG_HOME="+xdgConfigHome)
		env = append(env, "XDG_DATA_HOME="+xdgDataHome)
		env = append(env, "XDG_CACHE_HOME="+xdgCacheHome)
	} else {
		env = append(env, "OPENCODE_CONFIG_CONTENT="+configContent)
	}
	proc.Env = env
	if err := proc.Run(); err != nil {
		return err
	}
	return nil
}

func runCodexWrap(cmd *cobra.Command, cfgPath, tokenName, model string, ttl time.Duration, codexArgs []string) error {
	cfg, err := config.LoadClientConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load client config (run `toro connect` first): %w", err)
	}
	serverBase, err := deriveServerBaseURL(cfg.ServerURL)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be > 0")
	}
	bearer := strings.TrimSpace(cfg.APIKey)
	if bearer == "" {
		if !isLoopbackServerBase(serverBase) {
			return fmt.Errorf("client api key is required for non-localhost server (set with: toro connect)")
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "No client API key configured; using localhost no-auth admin access to create a temporary token.")
	}
	key, err := randomTemporaryKey()
	if err != nil {
		return fmt.Errorf("generate temporary key: %w", err)
	}
	if strings.TrimSpace(tokenName) == "" {
		tokenName = "toro-codex-" + time.Now().UTC().Format("20060102-150405")
	}
	expiresAt := time.Now().UTC().Add(ttl).Format(time.RFC3339)

	before, err := fetchAccessTokens(serverBase, bearer)
	if err != nil {
		return fmt.Errorf("list access tokens before create: %w", err)
	}
	beforeIDs := map[string]struct{}{}
	for _, t := range before {
		beforeIDs[strings.TrimSpace(t.ID)] = struct{}{}
	}
	if err := createAccessToken(serverBase, bearer, tokenName, key, "inferrer", expiresAt); err != nil {
		return fmt.Errorf("create temporary access token: %w", err)
	}
	tmpID, err := findCreatedTokenID(serverBase, bearer, beforeIDs, tokenName, expiresAt)
	if err != nil {
		return fmt.Errorf("locate temporary access token id: %w", err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Created temporary token %q (id=%s, expires=%s)\n", tokenName, tmpID, expiresAt)

	cleanup := func() {
		if err := deleteAccessToken(serverBase, bearer, tmpID); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to delete temporary token %q (id=%s): %v\n", tokenName, tmpID, err)
			return
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Deleted temporary token %q (id=%s)\n", tokenName, tmpID)
	}
	defer cleanup()

	selectedModel, selectedModelMsg, selErr := newToroClient(serverBase, cfg.APIKey).selectCodexModel(model)
	if selErr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to auto-select model: %v\n", selErr)
	}
	if strings.TrimSpace(selectedModelMsg) != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), selectedModelMsg)
	}

	launchArgs := make([]string, 0, len(codexArgs)+2)
	if strings.TrimSpace(selectedModel) != "" && !codexArgsContainModelSelection(codexArgs) {
		launchArgs = append(launchArgs, "--model", strings.TrimSpace(selectedModel))
	}
	launchArgs = append(launchArgs, codexArgs...)
	proc := exec.Command("codex", launchArgs...)
	proc.Stdin = cmd.InOrStdin()
	proc.Stdout = cmd.OutOrStdout()
	proc.Stderr = cmd.ErrOrStderr()
	env := filteredEnv([]string{
		"OPENAI_API_KEY",
		"OPENAI_BASE_URL",
		"OPENAI_API_BASE",
		"OPENAI_MODEL",
		"CODEX_API_KEY",
		"CODEX_BASE_URL",
		"CODEX_HOME",
	})
	env = append(env, "OPENAI_BASE_URL="+strings.TrimSuffix(serverBase, "/")+"/v1")
	env = append(env, "OPENAI_API_BASE="+strings.TrimSuffix(serverBase, "/")+"/v1")
	env = append(env, "CODEX_BASE_URL="+strings.TrimSuffix(serverBase, "/")+"/v1")
	env = append(env, "OPENAI_API_KEY="+key)
	env = append(env, "CODEX_API_KEY="+key)
	proc.Env = env
	if err := proc.Run(); err != nil {
		return err
	}
	return nil
}

func (c *toroClient) selectCodexModel(requested string) (string, string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested, "", nil
	}
	auto, err := c.discoverZeroCostModels()
	if err == nil && len(auto) > 0 {
		return auto[0], "Auto-selected model: " + auto[0], nil
	}
	models, err := c.fetchModels()
	if err != nil {
		if len(auto) == 0 {
			return "", "", err
		}
		return "", "", nil
	}
	ids := make([]string, 0, len(models))
	for _, m := range models {
		id := strings.TrimSpace(m.ID)
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "", "", nil
	}
	sort.Strings(ids)
	return ids[0], "Auto-selected model: " + ids[0], nil
}

func runGenericWrap(cmd *cobra.Command, cfgPath, tokenName string, ttl time.Duration, urlEnvName, keyEnvName string, args []string) error {
	cfg, err := config.LoadClientConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load client config (run `toro connect` first): %w", err)
	}
	serverBase, err := deriveServerBaseURL(cfg.ServerURL)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be > 0")
	}
	urlEnvName = strings.TrimSpace(urlEnvName)
	keyEnvName = strings.TrimSpace(keyEnvName)
	if !isValidEnvVarName(urlEnvName) {
		return fmt.Errorf("invalid --url-env %q", urlEnvName)
	}
	if !isValidEnvVarName(keyEnvName) {
		return fmt.Errorf("invalid --key-env %q", keyEnvName)
	}

	targetCmd := strings.TrimSpace(args[0])
	if targetCmd == "" {
		return fmt.Errorf("command cannot be empty")
	}

	bearer := strings.TrimSpace(cfg.APIKey)
	if bearer == "" {
		if !isLoopbackServerBase(serverBase) {
			return fmt.Errorf("client api key is required for non-localhost server (set with: toro connect)")
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "No client API key configured; using localhost no-auth admin access to create a temporary token.")
	}

	tmpKey, err := randomTemporaryKey()
	if err != nil {
		return fmt.Errorf("generate temporary key: %w", err)
	}
	if strings.TrimSpace(tokenName) == "" {
		tokenName = "toro-wrap-" + time.Now().UTC().Format("20060102-150405")
	}
	expiresAt := time.Now().UTC().Add(ttl).Format(time.RFC3339)

	before, err := fetchAccessTokens(serverBase, bearer)
	if err != nil {
		return fmt.Errorf("list access tokens before create: %w", err)
	}
	beforeIDs := map[string]struct{}{}
	for _, t := range before {
		beforeIDs[strings.TrimSpace(t.ID)] = struct{}{}
	}
	if err := createAccessToken(serverBase, bearer, tokenName, tmpKey, "inferrer", expiresAt); err != nil {
		return fmt.Errorf("create temporary access token: %w", err)
	}
	tmpID, err := findCreatedTokenID(serverBase, bearer, beforeIDs, tokenName, expiresAt)
	if err != nil {
		return fmt.Errorf("locate temporary access token id: %w", err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Created temporary token %q (id=%s, expires=%s)\n", tokenName, tmpID, expiresAt)

	cleanup := func() {
		if err := deleteAccessToken(serverBase, bearer, tmpID); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to delete temporary token %q (id=%s): %v\n", tokenName, tmpID, err)
			return
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Deleted temporary token %q (id=%s)\n", tokenName, tmpID)
	}
	defer cleanup()

	proc := exec.Command(targetCmd, args[1:]...)
	proc.Stdin = cmd.InOrStdin()
	proc.Stdout = cmd.OutOrStdout()
	proc.Stderr = cmd.ErrOrStderr()
	env := filteredEnv([]string{urlEnvName, keyEnvName})
	env = append(env, urlEnvName+"="+strings.TrimSuffix(serverBase, "/")+"/v1")
	env = append(env, keyEnvName+"="+tmpKey)
	proc.Env = env
	if err := proc.Run(); err != nil {
		return err
	}
	return nil
}

func codexArgsContainForcedLoginMethod(args []string) bool {
	return codexArgsContainConfigKey(args, "forced_login_method")
}

func codexArgsContainOSS(args []string) bool {
	for _, raw := range args {
		v := strings.TrimSpace(raw)
		if v == "--oss" {
			return true
		}
		if strings.HasPrefix(v, "--oss=") {
			return true
		}
	}
	return false
}

func codexArgsContainModelSelection(args []string) bool {
	for i := 0; i < len(args); i++ {
		v := strings.TrimSpace(args[i])
		if v == "" {
			continue
		}
		if v == "-m" || v == "--model" {
			return true
		}
		if strings.HasPrefix(v, "-m=") || strings.HasPrefix(v, "--model=") {
			return true
		}
	}
	return codexArgsContainConfigKey(args, "model")
}

func codexArgsContainConfigKey(args []string, key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for i := 0; i < len(args); i++ {
		v := strings.TrimSpace(args[i])
		if v == "" {
			continue
		}
		if strings.HasPrefix(v, "-c") || strings.HasPrefix(v, "--config") {
			if codexConfigFragmentContainsKey(v, key) {
				return true
			}
			if (v == "-c" || v == "--config") && i+1 < len(args) {
				if codexConfigFragmentContainsKey(args[i+1], key) {
					return true
				}
			}
		}
	}
	return false
}

func codexConfigFragmentContainsKey(fragment, key string) bool {
	s := strings.TrimSpace(fragment)
	key = strings.TrimSpace(key)
	if s == "" || key == "" {
		return false
	}
	if s == key {
		return true
	}
	return strings.HasPrefix(s, key+"=")
}

func filteredEnv(dropKeys []string) []string {
	if len(dropKeys) == 0 {
		return os.Environ()
	}
	drop := map[string]struct{}{}
	for _, k := range dropKeys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		drop[k] = struct{}{}
	}
	in := os.Environ()
	out := make([]string, 0, len(in))
	for _, e := range in {
		if i := strings.IndexByte(e, '='); i > 0 {
			if _, blocked := drop[e[:i]]; blocked {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

func isValidEnvVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' {
				continue
			}
			return false
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func deriveServerBaseURL(serverURL string) (string, error) {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return "", fmt.Errorf("server_url is empty")
	}
	u, err := neturl.Parse(serverURL)
	if err != nil {
		return "", fmt.Errorf("parse server_url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("server_url must be absolute, got %q", serverURL)
	}
	path := strings.TrimSpace(u.Path)
	path = strings.TrimSuffix(path, "/")
	if strings.HasSuffix(path, "/v1") {
		path = strings.TrimSuffix(path, "/v1")
	}
	u.Path = path
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	base := strings.TrimSuffix(u.String(), "/")
	return base, nil
}

func isLoopbackServerBase(serverBase string) bool {
	u, err := neturl.Parse(strings.TrimSpace(serverBase))
	if err != nil {
		return false
	}
	host := strings.TrimSpace(strings.Trim(u.Hostname(), "[]"))
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func randomTemporaryKey() (string, error) {
	var b [48]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString(b[:])
	return "tor_tmp_" + enc, nil
}

func randomConversationID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "toroconv_" + base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func (c *toroClient) adminAPIRequest(bearer, method, path string, body any) (*http.Response, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.serverBase+path, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

func fetchAccessTokens(serverBase, bearer string) ([]accessTokenItem, error) {
	r, err := newToroClient(serverBase, "").adminAPIRequest(bearer, http.MethodGet, "/admin/api/access-tokens", nil)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 8192))
		return nil, formatAdminTokenAPIError(r.StatusCode, b)
	}
	var items []accessTokenItem
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func createAccessToken(serverBase, bearer, name, key, role, expiresAt string) error {
	payload := map[string]any{
		"name":       strings.TrimSpace(name),
		"key":        strings.TrimSpace(key),
		"role":       strings.TrimSpace(role),
		"expires_at": strings.TrimSpace(expiresAt),
	}
	r, err := newToroClient(serverBase, "").adminAPIRequest(bearer, http.MethodPost, "/admin/api/access-tokens", payload)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 8192))
		return formatAdminTokenAPIError(r.StatusCode, b)
	}
	return nil
}

func findCreatedTokenID(serverBase, bearer string, beforeIDs map[string]struct{}, name, expiresAt string) (string, error) {
	for i := 0; i < 8; i++ {
		items, err := fetchAccessTokens(serverBase, bearer)
		if err != nil {
			return "", err
		}
		for _, t := range items {
			id := strings.TrimSpace(t.ID)
			if id == "" {
				continue
			}
			if _, existed := beforeIDs[id]; existed {
				continue
			}
			if strings.TrimSpace(t.Name) != strings.TrimSpace(name) {
				continue
			}
			if strings.TrimSpace(t.Role) != config.TokenRoleInferrer {
				continue
			}
			if strings.TrimSpace(t.ExpiresAt) != strings.TrimSpace(expiresAt) {
				continue
			}
			return id, nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return "", fmt.Errorf("temporary token not found after create")
}

func deleteAccessToken(serverBase, bearer, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("missing token id")
	}
	r, err := newToroClient(serverBase, "").adminAPIRequest(bearer, http.MethodDelete, "/admin/api/access-tokens/"+neturl.PathEscape(id), nil)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 8192))
		return formatAdminTokenAPIError(r.StatusCode, b)
	}
	return nil
}

func formatAdminTokenAPIError(status int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		if msg == "" {
			msg = "unauthorized"
		}
		return fmt.Errorf("status %d: %s (toro wrappers require an admin or keymaster token in toro connect)", status, msg)
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	return fmt.Errorf("status %d: %s", status, msg)
}

func buildOpencodeConfigContent(serverBase, tempKey, providerID, providerName, model string) (string, error) {
	injectedModel := strings.TrimSpace(model)
	if injectedModel != "" && !strings.Contains(injectedModel, "/") {
		injectedModel = providerID + "/" + injectedModel
	}
	payload := map[string]any{
		"provider": map[string]any{
			providerID: map[string]any{
				"name": providerName,
				"npm":  "@ai-sdk/openai-compatible",
				"options": map[string]any{
					"baseURL": strings.TrimSuffix(serverBase, "/") + "/v1",
					"apiKey":  tempKey,
				},
			},
		},
	}
	if injectedModel != "" {
		payload["model"] = injectedModel
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal opencode config content: %w", err)
	}
	return string(b), nil
}
