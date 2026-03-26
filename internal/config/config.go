package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const EnvMCPURL = "MSC_MCP_URL"

var configFilePathFn = ConfigFilePath // Indirection for testing.

type Config struct {
	MCPURL string `json:"mcp_url"`
}

func Load() (*Config, error) {
	cfg, err := loadFile()
	if err != nil {
		return nil, err
	}

	if v := os.Getenv(EnvMCPURL); v != "" {
		cfg.MCPURL = v
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

	if c.MCPURL == "" {
		errs = append(errs, "mcp_url is not set (set MSC_MCP_URL or run: msc config set-mcp-url <url>)")
	} else if err := validateMCPURL(c.MCPURL); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid config:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func validateMCPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("mcp_url is invalid: %w", err)
	}

	switch {
	case u.Scheme != "https":
		return errors.New("mcp_url must use https")
	case u.Host == "":
		return errors.New("mcp_url must include a host")
	case u.RawQuery != "":
		return errors.New("mcp_url must not include a query string")
	case u.Fragment != "":
		return errors.New("mcp_url must not include a fragment")
	case u.Path != "/mcp" && u.Path != "/authed/mcp":
		return errors.New("mcp_url must end with /mcp or /authed/mcp")
	default:
		return nil
	}
}
