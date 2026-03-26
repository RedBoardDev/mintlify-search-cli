package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	cfg := &Config{MCPURL: "https://docs.example.com/mcp"}

	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.MCPURL != cfg.MCPURL {
		t.Errorf("mcp_url: got %q, want %q", loaded.MCPURL, cfg.MCPURL)
	}
}

func TestLoad_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	t.Setenv(EnvMCPURL, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MCPURL != "" {
		t.Errorf("expected zero config, got %+v", cfg)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	cfg := &Config{MCPURL: "https://file.example.com/mcp"}
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	t.Setenv(EnvMCPURL, "https://env.example.com/mcp")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.MCPURL != "https://env.example.com/mcp" {
		t.Errorf("mcp_url: got %q, want %q", loaded.MCPURL, "https://env.example.com/mcp")
	}
}

func TestLoad_EnvOnlyNoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	t.Setenv(EnvMCPURL, "https://env.example.com/mcp")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.MCPURL != "https://env.example.com/mcp" {
		t.Errorf("mcp_url: got %q, want %q", loaded.MCPURL, "https://env.example.com/mcp")
	}
}

func TestLoad_IgnoresLegacyFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	if err := os.WriteFile(path, []byte(`{"api_key":"mint_x","domain":"docs.example.com"}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.MCPURL != "" {
		t.Errorf("expected empty mcp_url, got %q", loaded.MCPURL)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid public", Config{MCPURL: "https://docs.example.com/mcp"}, false},
		{"valid authed", Config{MCPURL: "https://docs.example.com/authed/mcp"}, false},
		{"missing", Config{}, true},
		{"http only", Config{MCPURL: "http://docs.example.com/mcp"}, true},
		{"missing host", Config{MCPURL: "https:///mcp"}, true},
		{"query", Config{MCPURL: "https://docs.example.com/mcp?x=1"}, true},
		{"fragment", Config{MCPURL: "https://docs.example.com/mcp#x"}, true},
		{"wrong path", Config{MCPURL: "https://docs.example.com/docs"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "deep", "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	cfg := &Config{MCPURL: "https://docs.example.com/mcp"}
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}
