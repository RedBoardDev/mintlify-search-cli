package cliapp

import (
	"context"
	"testing"
	"time"

	"github.com/redboard/mintlify-search-cli/internal/config"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
	"github.com/redboard/mintlify-search-cli/internal/render"
)

func fakeSearchTool() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name: "search_docs",
		InputSchema: mcp.InputSchema{
			Required:   []string{"query"},
			Properties: map[string]mcp.SchemaProperty{"query": {Type: "string"}},
		},
	}
}

func fakeFSTool() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name: "query_docs_filesystem_docs",
		InputSchema: mcp.InputSchema{
			Required:   []string{"command"},
			Properties: map[string]mcp.SchemaProperty{"command": {Type: "string"}},
		},
	}
}

func newApp(t *testing.T, client mcp.MCPClient) *App {
	t.Helper()
	return &App{
		Cfg: &config.Config{
			MCPURL:       "https://x.example.com/mcp",
			DefaultLimit: 5,
			Cache:        config.CacheConfig{TTLSeconds: 600, ToolsTTLSeconds: 3600},
		},
		Client:   client,
		Renderer: render.New(render.FormatJSON),
		CacheDir: t.TempDir(),
		Timeout:  5 * time.Second,
	}
}

func TestResolveSearchTool_DiscoversAndCaches(t *testing.T) {
	fake := &mcp.FakeClient{Tools: []mcp.ToolDefinition{fakeSearchTool(), fakeFSTool()}}
	app := newApp(t, fake)

	got, err := app.ResolveSearchTool(context.Background())
	if err != nil {
		t.Fatalf("ResolveSearchTool: %v", err)
	}
	if got != "search_docs" {
		t.Fatalf("got %q", got)
	}

	// Second call should hit the cache (no additional tools/list call).
	before := len(fake.CallLog)
	got2, err := app.ResolveSearchTool(context.Background())
	if err != nil {
		t.Fatalf("cache lookup: %v", err)
	}
	if got2 != "search_docs" || len(fake.CallLog) != before {
		t.Fatalf("expected cache hit, got call log grew: %d -> %d", before, len(fake.CallLog))
	}
}

func TestResolveFSTool(t *testing.T) {
	fake := &mcp.FakeClient{Tools: []mcp.ToolDefinition{fakeSearchTool(), fakeFSTool()}}
	app := newApp(t, fake)
	got, err := app.ResolveFSTool(context.Background())
	if err != nil {
		t.Fatalf("ResolveFSTool: %v", err)
	}
	if got != "query_docs_filesystem_docs" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveSearchTool_NoCacheBypassesCache(t *testing.T) {
	fake := &mcp.FakeClient{Tools: []mcp.ToolDefinition{fakeSearchTool(), fakeFSTool()}}
	app := newApp(t, fake)
	app.NoCache = true

	if _, err := app.ResolveSearchTool(context.Background()); err != nil {
		t.Fatalf("first: %v", err)
	}
	listCalls := countCalls(fake.CallLog, "tools/list")

	if _, err := app.ResolveSearchTool(context.Background()); err != nil {
		t.Fatalf("second: %v", err)
	}
	listCalls2 := countCalls(fake.CallLog, "tools/list")
	if listCalls2 != listCalls+1 {
		t.Fatalf("expected NoCache to re-query tools/list: %d -> %d", listCalls, listCalls2)
	}
}

func TestMapError(t *testing.T) {
	if MapError(nil) != nil {
		t.Fatal("nil should map to nil")
	}
	if ee := MapError(ErrUsage); ee.Code != ExitUsage {
		t.Fatalf("ErrUsage code=%d", ee.Code)
	}
	if ee := MapError(ErrConfig); ee.Code != ExitConfig {
		t.Fatalf("ErrConfig code=%d", ee.Code)
	}
	wrapped := Wrap(ExitRuntime, ErrMCPUnreachable)
	if MapError(wrapped).Code != ExitRuntime {
		t.Fatalf("pre-wrapped should keep its code")
	}
}

func countCalls(log []mcp.CallRecord, method string) int {
	n := 0
	for _, c := range log {
		if c.Method == method {
			n++
		}
	}
	return n
}
