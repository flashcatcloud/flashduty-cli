package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultBaseURL = "https://api.flashcat.cloud"
	configDirName  = ".flashduty"
	configFileName = "config.yaml"
)

type Config struct {
	AppKey  string `yaml:"app_key"`
	BaseURL string `yaml:"base_url"`
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configDirName)
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), configFileName)
}

// Load reads config from file, then overlays environment variables.
// Flag overrides are applied by the caller.
func Load() (*Config, error) {
	cfg := &Config{
		BaseURL: DefaultBaseURL,
	}

	data, err := os.ReadFile(ConfigPath())
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	if v := os.Getenv("FLASHDUTY_APP_KEY"); v != "" {
		cfg.AppKey = v
	}
	if v := os.Getenv("FLASHDUTY_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}

	return cfg, nil
}

// Save writes the config to disk with restricted permissions.
func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(), data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// MaskKey partially masks an app key for display.
func MaskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// ConfigSource returns the source of a config value for display.
func ConfigSource(key string) string {
	switch key {
	case "app_key":
		if os.Getenv("FLASHDUTY_APP_KEY") != "" {
			return "(from env FLASHDUTY_APP_KEY)"
		}
	case "base_url":
		if os.Getenv("FLASHDUTY_BASE_URL") != "" {
			return "(from env FLASHDUTY_BASE_URL)"
		}
	}
	if _, err := os.Stat(ConfigPath()); err == nil {
		return fmt.Sprintf("(from %s)", ConfigPath())
	}
	return "(default)"
}
