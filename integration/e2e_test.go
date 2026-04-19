//go:build integration

// Package integration runs msc against the real production MCP. Enable with:
//   go test -tags=integration ./integration/...
// The tests are skipped by default in CI; the MSC_E2E_URL env var can override
// the default target.
package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func bin(t *testing.T) string {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ""))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	out := filepath.Join(t.TempDir(), "msc")
	build := exec.Command("go", "build", "-o", out, "./cmd/msc")
	build.Dir = repoRoot
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, b)
	}
	return out
}

func mcpURL() string {
	if v := os.Getenv("MSC_E2E_URL"); v != "" {
		return v
	}
	return "https://api-documentation.kare-app.fr/mcp"
}

func run(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin(t), append([]string{"--mcp-url", mcpURL()}, args...)...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	return so.String(), se.String(), code
}

func TestDoctor(t *testing.T) {
	stdout, stderr, code := run(t, "doctor")
	if code != 0 {
		t.Fatalf("exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "All checks passed") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestSearchJSON(t *testing.T) {
	stdout, _, code := run(t, "search", "authentication", "--limit", "3")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		Query   string `json:"query"`
		Results []struct {
			Title, URL, Page, Content string
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode json: %v\n%s", err, stdout)
	}
	if got.Query != "authentication" {
		t.Fatalf("query = %q", got.Query)
	}
	if len(got.Results) == 0 || len(got.Results) > 3 {
		t.Fatalf("results = %d", len(got.Results))
	}
	if got.Results[0].Page == "" {
		t.Fatalf("first result has empty page field")
	}
}

func TestSearchEmptyQueryExit2(t *testing.T) {
	_, stderr, code := run(t, "search", "")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "non-empty") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestFSTree(t *testing.T) {
	stdout, _, code := run(t, "fs", "tree / -L 1")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "Auth") {
		t.Fatalf("unexpected tree output: %s", stdout)
	}
}

func TestReadResolvesMdx(t *testing.T) {
	stdout, _, code := run(t, "read", "/Rule/v2/api-bo/read/list-rules-for-the-given-site-ids", "--lines", "5")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if stdout == "" {
		t.Fatal("expected markdown content")
	}
	if strings.Count(stdout, "\n") > 6 {
		t.Fatalf("expected at most 5 lines, got %d", strings.Count(stdout, "\n"))
	}
}

func TestOpenTopHit(t *testing.T) {
	stdout, _, code := run(t, "open", "list rules", "--lines", "10")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if stdout == "" {
		t.Fatal("expected markdown content on stdout")
	}
}

func TestInvalidMCPURL(t *testing.T) {
	cmd := exec.Command(bin(t), "--mcp-url", "https://invalid.example.com/mcp", "search", "x")
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("expected exit 1, got err=%v", err)
	}
	if !strings.Contains(se.String(), "msc: error:") {
		t.Fatalf("stderr missing prefix: %s", se.String())
	}
}

func TestSearchDeterminism(t *testing.T) {
	a, _, _ := run(t, "search", "rate limiting", "--limit", "5")
	b, _, _ := run(t, "search", "rate limiting", "--limit", "5")
	if a != b {
		t.Fatalf("non-deterministic output:\n a=%q\n b=%q", a, b)
	}
}

func TestCompletionBash(t *testing.T) {
	stdout, _, code := run(t, "completion", "bash")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "bash completion") {
		t.Fatalf("missing header: %s", stdout[:minInt(200, len(stdout))])
	}
}

func TestFSTruncation(t *testing.T) {
	_, stderr, code := run(t, "fs", "cat /openapi/global/openapi.yaml")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "truncated") {
		t.Fatalf("expected truncation marker on stderr, got: %s", stderr)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
