package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/redboard/mintlify-search-cli/internal/mcp"
)

var testResults = []mcp.SearchResult{
	{Title: "Authentication", Content: "How to set up auth", URL: "https://docs.example.com/auth"},
	{Title: "OAuth Setup", Content: "OAuth flow guide", URL: "https://docs.example.com/oauth"},
}

func TestRenderText(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, testResults, FormatText); err != nil {
		t.Fatalf("render: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "[1] Authentication") {
		t.Error("missing result 1 title")
	}
	if !strings.Contains(out, "https://docs.example.com/auth") {
		t.Error("missing result 1 URL")
	}
	if !strings.Contains(out, "[2] OAuth Setup") {
		t.Error("missing result 2 title")
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, testResults, FormatJSON); err != nil {
		t.Fatalf("render: %v", err)
	}

	out := buf.String()

	// Should be minified (single line).
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (minified), got %d", len(lines))
	}

	if !strings.Contains(out, `"title":"Authentication"`) {
		t.Error("missing title in JSON output")
	}
}

func TestRenderEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, nil, FormatText); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "No results found") {
		t.Error("expected 'No results found' message")
	}
}
