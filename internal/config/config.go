package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDir  = "imagenodes"
	configFile = "config.json"
)

// Config holds CLI configuration
type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`

	// APIKeySource and BaseURLSource say where each value came from, such as
	// "--api-key flag" or "config file /path/config.json". Not saved.
	APIKeySource  string `json:"-"`
	BaseURLSource string `json:"-"`
}

// Load reads config with priority: flags > env vars > config file
func Load(flagAPIKey, flagBaseURL string) (*Config, error) {
	cfg := &Config{APIKeySource: "not set"}

	// Start with config file
	fileCfg, _ := loadFromFile()
	if fileCfg != nil {
		fileSource := "config file"
		if path, err := FilePath(); err == nil {
			fileSource += " " + path
		}
		if fileCfg.APIKey != "" {
			cfg.APIKey, cfg.APIKeySource = fileCfg.APIKey, fileSource
		}
		if fileCfg.BaseURL != "" {
			cfg.BaseURL, cfg.BaseURLSource = fileCfg.BaseURL, fileSource
		}
	}

	// Override with env vars
	if v := os.Getenv("INODES_API_KEY"); v != "" {
		cfg.APIKey, cfg.APIKeySource = v, "INODES_API_KEY environment variable"
	}
	if v := os.Getenv("INODES_BASE_URL"); v != "" {
		cfg.BaseURL, cfg.BaseURLSource = v, "INODES_BASE_URL environment variable"
	}

	// Override with flags
	if flagAPIKey != "" {
		cfg.APIKey, cfg.APIKeySource = flagAPIKey, "--api-key flag"
	}
	if flagBaseURL != "" {
		cfg.BaseURL, cfg.BaseURLSource = flagBaseURL, "--base-url flag"
	}

	// Default base URL
	if cfg.BaseURL == "" {
		cfg.BaseURL, cfg.BaseURLSource = "https://imagenodes.com", "default"
	}

	return cfg, nil
}

// Save persists config to disk
func Save(cfg *Config) error {
	dir, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, configFile), data, 0600)
}

// RequireAPIKey validates that API key is present
func (c *Config) RequireAPIKey() error {
	if c.APIKey == "" {
		return errors.New("API key not configured. Run 'inodes configure' or set INODES_API_KEY")
	}
	return nil
}

func loadFromFile() (*Config, error) {
	dir, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func configPath() (string, error) {
	home, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDir), nil
}

// FilePath returns the path of the config file, whether or not it exists.
func FilePath() (string, error) {
	dir, err := configPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

// MaskKey hides all but the first and last four characters of an API key.
func MaskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
