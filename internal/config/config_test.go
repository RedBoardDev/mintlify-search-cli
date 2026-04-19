package config

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	prev := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	t.Cleanup(func() { configFilePathFn = prev })
	for _, k := range []string{EnvMCPURL, EnvDefaultLimit, EnvCacheEnabled, EnvCacheTTL, EnvToolsCacheTTL} {
		t.Setenv(k, "")
	}
	return path
}

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	withTempConfig(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultLimit != DefaultLimit {
		t.Errorf("DefaultLimit = %d, want %d", cfg.DefaultLimit, DefaultLimit)
	}
	if cfg.Cache.TTLSeconds != DefaultCacheTTLSeconds {
		t.Errorf("Cache.TTLSeconds = %d, want %d", cfg.Cache.TTLSeconds, DefaultCacheTTLSeconds)
	}
	if cfg.Cache.ToolsTTLSeconds != DefaultToolsTTLSeconds {
		t.Errorf("Cache.ToolsTTLSeconds = %d", cfg.Cache.ToolsTTLSeconds)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	withTempConfig(t)
	in := &Config{
		MCPURL:       "https://api-documentation.kare-app.fr/mcp",
		DefaultLimit: 10,
		Cache: CacheConfig{
			Enabled:         true,
			TTLSeconds:      1200,
			ToolsTTLSeconds: 3600,
		},
	}
	if err := Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.MCPURL != in.MCPURL || out.DefaultLimit != 10 || !out.Cache.Enabled || out.Cache.TTLSeconds != 1200 || out.Cache.ToolsTTLSeconds != 3600 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	path := withTempConfig(t)
	_ = os.WriteFile(path, []byte(`{"mcp_url":"https://file.example.com/mcp"}`), 0o600)

	t.Setenv(EnvMCPURL, "https://env.example.com/mcp")
	t.Setenv(EnvDefaultLimit, "7")
	t.Setenv(EnvCacheEnabled, "1")
	t.Setenv(EnvCacheTTL, "900")
	t.Setenv(EnvToolsCacheTTL, "7200")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MCPURL != "https://env.example.com/mcp" {
		t.Errorf("MCPURL = %q", cfg.MCPURL)
	}
	if cfg.DefaultLimit != 7 {
		t.Errorf("DefaultLimit = %d", cfg.DefaultLimit)
	}
	if !cfg.Cache.Enabled {
		t.Errorf("expected cache enabled from env")
	}
	if cfg.Cache.TTLSeconds != 900 {
		t.Errorf("TTLSeconds = %d", cfg.Cache.TTLSeconds)
	}
	if cfg.Cache.ToolsTTLSeconds != 7200 {
		t.Errorf("ToolsTTLSeconds = %d", cfg.Cache.ToolsTTLSeconds)
	}
}

func TestLoad_IgnoresUnknownFields(t *testing.T) {
	path := withTempConfig(t)
	_ = os.WriteFile(path, []byte(`{"api_key":"x","domain":"y","mcp_url":"https://docs.example.com/mcp"}`), 0o600)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MCPURL != "https://docs.example.com/mcp" {
		t.Errorf("MCPURL = %q", cfg.MCPURL)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid public", Config{MCPURL: "https://docs.example.com/mcp", DefaultLimit: 5, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, false},
		{"valid authed", Config{MCPURL: "https://docs.example.com/authed/mcp", DefaultLimit: 10, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, false},
		{"missing url", Config{DefaultLimit: 5, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, true},
		{"http only", Config{MCPURL: "http://x.com/mcp", DefaultLimit: 5, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, true},
		{"query string", Config{MCPURL: "https://x.com/mcp?a=b", DefaultLimit: 5, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, true},
		{"fragment", Config{MCPURL: "https://x.com/mcp#f", DefaultLimit: 5, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, true},
		{"wrong path", Config{MCPURL: "https://x.com/docs", DefaultLimit: 5, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, true},
		{"limit too high", Config{MCPURL: "https://x.com/mcp", DefaultLimit: 100, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, true},
		{"limit zero", Config{MCPURL: "https://x.com/mcp", DefaultLimit: 0, Cache: CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 86400}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "deep", "config.json")
	prev := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	t.Cleanup(func() { configFilePathFn = prev })

	cfg := &Config{MCPURL: "https://docs.example.com/mcp", DefaultLimit: 5}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}
