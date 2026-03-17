package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Override config path for testing.
	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	cfg := &Config{
		APIKey: "mint_dsc_test123",
		Domain: "docs.example.com",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.APIKey != cfg.APIKey {
		t.Errorf("api_key: got %q, want %q", loaded.APIKey, cfg.APIKey)
	}
	if loaded.Domain != cfg.Domain {
		t.Errorf("domain: got %q, want %q", loaded.Domain, cfg.Domain)
	}
}

func TestLoad_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	// Clear env vars to avoid interference.
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvDomain, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "" || cfg.Domain != "" {
		t.Errorf("expected zero config, got %+v", cfg)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	// Save a file-based config.
	cfg := &Config{APIKey: "mint_file_key", Domain: "file.example.com"}
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Set env vars — should override file values.
	t.Setenv(EnvAPIKey, "mint_env_key")
	t.Setenv(EnvDomain, "env.example.com")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.APIKey != "mint_env_key" {
		t.Errorf("api_key: got %q, want %q", loaded.APIKey, "mint_env_key")
	}
	if loaded.Domain != "env.example.com" {
		t.Errorf("domain: got %q, want %q", loaded.Domain, "env.example.com")
	}
}

func TestLoad_EnvPartialOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	cfg := &Config{APIKey: "mint_file_key", Domain: "file.example.com"}
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Only override domain via env.
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvDomain, "env.example.com")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.APIKey != "mint_file_key" {
		t.Errorf("api_key should come from file: got %q", loaded.APIKey)
	}
	if loaded.Domain != "env.example.com" {
		t.Errorf("domain should come from env: got %q", loaded.Domain)
	}
}

func TestLoad_EnvOnlyNoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "config.json")

	origFn := configFilePathFn
	configFilePathFn = func() (string, error) { return path, nil }
	defer func() { configFilePathFn = origFn }()

	t.Setenv(EnvAPIKey, "mint_env_only")
	t.Setenv(EnvDomain, "env.example.com")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.APIKey != "mint_env_only" {
		t.Errorf("api_key: got %q, want %q", loaded.APIKey, "mint_env_only")
	}
	if loaded.Domain != "env.example.com" {
		t.Errorf("domain: got %q, want %q", loaded.Domain, "env.example.com")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid", Config{APIKey: "mint_dsc_abc", Domain: "docs.example.com"}, false},
		{"missing key", Config{Domain: "docs.example.com"}, true},
		{"bad prefix", Config{APIKey: "sk_abc", Domain: "docs.example.com"}, true},
		{"missing domain", Config{APIKey: "mint_dsc_abc"}, true},
		{"empty", Config{}, true},
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

	cfg := &Config{APIKey: "mint_test", Domain: "example.com"}
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}
