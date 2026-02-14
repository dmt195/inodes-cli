package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	configDir  = "imagenodes"
	configFile = "config.json"
)

// Config holds CLI configuration
type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

// Load reads config with priority: flags > env vars > config file
func Load(flagAPIKey, flagBaseURL string) (*Config, error) {
	cfg := &Config{}

	// Start with config file
	fileCfg, _ := loadFromFile()
	if fileCfg != nil {
		*cfg = *fileCfg
	}

	// Override with env vars
	if v := os.Getenv("INODES_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("INODES_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	// Override with flags
	if flagAPIKey != "" {
		cfg.APIKey = flagAPIKey
	}
	if flagBaseURL != "" {
		cfg.BaseURL = flagBaseURL
	}

	// Default base URL
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://imagenodes.com"
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
