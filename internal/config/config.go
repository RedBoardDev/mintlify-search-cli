package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvAPIKey = "MSC_API_KEY" //nolint:gosec // Env var name, not a credential.
	EnvDomain = "MSC_DOMAIN"
)

var configFilePathFn = ConfigFilePath // Indirection for testing.

type Config struct {
	APIKey string `json:"api_key"`
	Domain string `json:"domain"`
}

// Precedence: env vars > config file. Flag overrides are in the CLI layer.
func Load() (*Config, error) {
	cfg, err := loadFile()
	if err != nil {
		return nil, err
	}

	if v := os.Getenv(EnvAPIKey); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv(EnvDomain); v != "" {
		cfg.Domain = v
	}

	return cfg, nil
}

func loadFile() (*Config, error) {
	path, err := configFilePathFn()
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // Path from internal config dir, not user input.
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := configFilePathFn()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

func (c *Config) Validate() error {
	var errs []string

	if c.APIKey == "" {
		errs = append(errs, "api_key is not set (set MSC_API_KEY or run: msc config set-key <key>)")
	} else if !strings.HasPrefix(c.APIKey, "mint_") {
		errs = append(errs, "api_key does not start with 'mint_' — check your key")
	}

	if c.Domain == "" {
		errs = append(errs, "domain is not set (set MSC_DOMAIN or run: msc config set-domain <domain>)")
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid config:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
