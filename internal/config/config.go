// Package config manages on-disk configuration and environment-variable
// overrides for the msc CLI.
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

// Environment variables recognised by Load. Flags take precedence over env,
// which takes precedence over the config file.
const (
	EnvMCPURL        = "MSC_MCP_URL"
	EnvDefaultLimit  = "MSC_DEFAULT_LIMIT"
	EnvCacheEnabled  = "MSC_CACHE_ENABLED"
	EnvCacheTTL      = "MSC_CACHE_TTL_SECONDS"
	EnvToolsCacheTTL = "MSC_TOOLS_CACHE_TTL_SECONDS"
)

// Defaults applied when config fields are zero-valued.
const (
	DefaultLimit            = 5
	DefaultCacheTTLSeconds  = 600
	DefaultToolsTTLSeconds  = 86400
	MaxSearchLimit          = 20
)

var configFilePathFn = ConfigFilePath // Indirection for testing.

// Config is the on-disk representation. Fields are small and flat on purpose:
// the CLI does not need a deep tree.
type Config struct {
	MCPURL       string      `json:"mcp_url"`
	DefaultLimit int         `json:"default_limit,omitempty"`
	Cache        CacheConfig `json:"cache"`
}

// CacheConfig controls the opt-in search-result cache. The tools-discovery
// cache is always on and uses its own TTL (ToolsTTLSeconds) which is not
// persisted separately — it lives alongside cache.ttl_seconds for simplicity.
type CacheConfig struct {
	Enabled         bool `json:"enabled"`
	TTLSeconds      int  `json:"ttl_seconds,omitempty"`
	ToolsTTLSeconds int  `json:"tools_ttl_seconds,omitempty"`
}

// Load reads the config file (if present) and overlays environment variables.
// A missing config file is not an error; it yields a zero-valued Config plus
// any env var overrides.
func Load() (*Config, error) {
	cfg, err := loadFile()
	if err != nil {
		return nil, err
	}
	applyEnv(cfg)
	applyDefaults(cfg)
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

func applyEnv(cfg *Config) {
	if v := os.Getenv(EnvMCPURL); v != "" {
		cfg.MCPURL = v
	}
	if v := os.Getenv(EnvDefaultLimit); v != "" {
		if n, err := atoiPositive(v); err == nil {
			cfg.DefaultLimit = n
		}
	}
	if v := os.Getenv(EnvCacheEnabled); v != "" {
		cfg.Cache.Enabled = v == "1" || strings.EqualFold(v, "true")
	}
	if v := os.Getenv(EnvCacheTTL); v != "" {
		if n, err := atoiPositive(v); err == nil {
			cfg.Cache.TTLSeconds = n
		}
	}
	if v := os.Getenv(EnvToolsCacheTTL); v != "" {
		if n, err := atoiPositive(v); err == nil {
			cfg.Cache.ToolsTTLSeconds = n
		}
	}
}

func applyDefaults(cfg *Config) {
	if cfg.DefaultLimit <= 0 {
		cfg.DefaultLimit = DefaultLimit
	}
	if cfg.Cache.TTLSeconds <= 0 {
		cfg.Cache.TTLSeconds = DefaultCacheTTLSeconds
	}
	if cfg.Cache.ToolsTTLSeconds <= 0 {
		cfg.Cache.ToolsTTLSeconds = DefaultToolsTTLSeconds
	}
}

// Save writes the config to disk with 0600 permissions. The parent directory
// is created with 0700 if needed.
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

// Validate checks required fields and URL constraints.
func (c *Config) Validate() error {
	var errs []string
	if c.MCPURL == "" {
		errs = append(errs, "mcp_url is not set (set MSC_MCP_URL or run: msc config set mcp_url <url>)")
	} else if err := ValidateMCPURL(c.MCPURL); err != nil {
		errs = append(errs, err.Error())
	}
	if c.DefaultLimit < 1 || c.DefaultLimit > MaxSearchLimit {
		errs = append(errs, fmt.Sprintf("default_limit must be between 1 and %d", MaxSearchLimit))
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid config:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// ValidateMCPURL enforces the URL shape expected by Mintlify MCP servers.
// Exported so the `config set` handler can reject a bad URL at write time.
func ValidateMCPURL(raw string) error {
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

func atoiPositive(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not a positive integer")
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return 0, errors.New("zero is not positive")
	}
	return n, nil
}
