package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchRejectsJSONAndRawTogether(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"search", "auth", "--json", "--raw"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--json and --raw cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchRejectsRawAndExplicitLimitTogether(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"search", "auth", "--raw", "--limit", "3"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--raw and --limit cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigSetMCPURLSavesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"config", "set-mcp-url", "https://docs.example.com/mcp"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, "Library", "Application Support", "msc", "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), `"mcp_url": "https://docs.example.com/mcp"`) {
		t.Fatalf("unexpected config file: %s", data)
	}
}

func TestConfigSetMCPURLRejectsInvalidURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"config", "set-mcp-url", "https://docs.example.com/docs"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "mcp_url must end with /mcp or /authed/mcp") {
		t.Fatalf("unexpected error: %v", err)
	}
}
