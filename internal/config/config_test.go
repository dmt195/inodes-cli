package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// isolateConfig points os.UserConfigDir at a temp directory so tests never
// read or overwrite the developer's real config. XDG_CONFIG_HOME alone is not
// enough: on macOS UserConfigDir ignores it and uses $HOME/Library/Application Support.
func isolateConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("APPDATA", filepath.Join(tmp, "AppData"))

	dir, err := configPath()
	if err != nil {
		t.Fatalf("configPath: %v", err)
	}
	if !strings.HasPrefix(dir, tmp) {
		t.Fatalf("config path %q escaped the test temp dir %q", dir, tmp)
	}
	return tmp
}

func TestLoad_Defaults(t *testing.T) {
	isolateConfig(t)
	// Clear env vars so they don't interfere
	t.Setenv("INODES_API_KEY", "")
	t.Setenv("INODES_BASE_URL", "")

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.BaseURL != "https://imagenodes.com" {
		t.Errorf("expected default base URL, got %q", cfg.BaseURL)
	}
	if cfg.APIKey != "" {
		t.Errorf("expected empty API key, got %q", cfg.APIKey)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	isolateConfig(t)
	t.Setenv("INODES_API_KEY", "env-key-123")
	t.Setenv("INODES_BASE_URL", "http://localhost:8081")

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.APIKey != "env-key-123" {
		t.Errorf("expected env API key, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != "http://localhost:8081" {
		t.Errorf("expected env base URL, got %q", cfg.BaseURL)
	}
}

func TestLoad_FlagOverride(t *testing.T) {
	isolateConfig(t)
	t.Setenv("INODES_API_KEY", "env-key")

	cfg, err := Load("flag-key", "http://flag-url")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.APIKey != "flag-key" {
		t.Errorf("expected flag API key, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != "http://flag-url" {
		t.Errorf("expected flag base URL, got %q", cfg.BaseURL)
	}
}

func TestSaveAndLoad(t *testing.T) {
	isolateConfig(t)
	t.Setenv("INODES_API_KEY", "")
	t.Setenv("INODES_BASE_URL", "")

	if err := Save(&Config{APIKey: "test-key-save", BaseURL: "http://test.local"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.APIKey != "test-key-save" {
		t.Errorf("expected saved API key, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != "http://test.local" {
		t.Errorf("expected saved base URL, got %q", cfg.BaseURL)
	}
}

func TestRequireAPIKey(t *testing.T) {
	cfg := &Config{}
	if err := cfg.RequireAPIKey(); err == nil {
		t.Error("expected error for empty API key")
	}

	cfg.APIKey = "some-key"
	if err := cfg.RequireAPIKey(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
