package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultServerConfigPathUsesConfigToml(t *testing.T) {
	if got := filepath.Base(DefaultServerConfigPath()); got != defaultConfigFileName {
		t.Fatalf("expected default config file %q, got %q", defaultConfigFileName, got)
	}
}

func TestDefaultClientConfigPathUsesToroToml(t *testing.T) {
	if got := filepath.Base(DefaultClientConfigPath()); got != "toro.toml" {
		t.Fatalf("expected default client config file %q, got %q", "toro.toml", got)
	}
}

func TestLoadServerConfigMigratesLegacyIncomingAPIKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, defaultConfigFileName)
	legacyToml := `
listen_addr = ":7050"
incoming_api_keys = ["k1", "k2", "k1", ""]
`
	if err := os.WriteFile(path, []byte(strings.TrimSpace(legacyToml)+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadServerConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.IncomingTokens) != 2 {
		t.Fatalf("expected 2 migrated tokens, got %d", len(cfg.IncomingTokens))
	}
	if cfg.IncomingTokens[0].Key != "k1" || cfg.IncomingTokens[1].Key != "k2" {
		t.Fatalf("unexpected migrated keys: %+v", cfg.IncomingTokens)
	}
}

func TestLoadServerConfigMigratesLegacyAdminAPIKeyToIncomingAdminToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, defaultConfigFileName)
	legacyToml := `
listen_addr = ":7050"
admin_api_key = "legacy-admin-secret"
`
	if err := os.WriteFile(path, []byte(strings.TrimSpace(legacyToml)+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadServerConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.IncomingTokens) != 1 {
		t.Fatalf("expected 1 migrated admin token, got %d", len(cfg.IncomingTokens))
	}
	if cfg.IncomingTokens[0].Role != TokenRoleAdmin {
		t.Fatalf("expected migrated token role admin, got %q", cfg.IncomingTokens[0].Role)
	}
	if cfg.IncomingTokens[0].Key != "legacy-admin-secret" {
		t.Fatalf("unexpected migrated admin key: %q", cfg.IncomingTokens[0].Key)
	}
}
