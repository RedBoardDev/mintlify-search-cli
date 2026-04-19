package mcp

import (
	"testing"
	"time"
)

func TestFindSearchTool(t *testing.T) {
	tools := []ToolDefinition{
		{Name: "query_docs_filesystem_x", InputSchema: InputSchema{
			Required:   []string{"command"},
			Properties: map[string]SchemaProperty{"command": {Type: "string"}},
		}},
		{Name: "search_x", InputSchema: InputSchema{
			Required:   []string{"query"},
			Properties: map[string]SchemaProperty{"query": {Type: "string"}},
		}},
	}
	tool, err := FindSearchTool(tools)
	if err != nil {
		t.Fatalf("FindSearchTool: %v", err)
	}
	if tool.Name != "search_x" {
		t.Fatalf("got %q, want search_x", tool.Name)
	}
}

func TestFindSearchTool_Single(t *testing.T) {
	tools := []ToolDefinition{{Name: "anything"}}
	tool, err := FindSearchTool(tools)
	if err != nil || tool.Name != "anything" {
		t.Fatalf("single-tool fallback failed: tool=%+v err=%v", tool, err)
	}
}

func TestFindSearchTool_NoMatch(t *testing.T) {
	tools := []ToolDefinition{
		{Name: "foo"},
		{Name: "bar"},
	}
	if _, err := FindSearchTool(tools); err == nil {
		t.Fatalf("expected error when no search tool present")
	}
}

func TestFindFSTool(t *testing.T) {
	tools := []ToolDefinition{
		{Name: "search_x", InputSchema: InputSchema{
			Required:   []string{"query"},
			Properties: map[string]SchemaProperty{"query": {Type: "string"}},
		}},
		{Name: "query_docs_filesystem_x", InputSchema: InputSchema{
			Required:   []string{"command"},
			Properties: map[string]SchemaProperty{"command": {Type: "string"}},
		}},
	}
	tool, err := FindFSTool(tools)
	if err != nil {
		t.Fatalf("FindFSTool: %v", err)
	}
	if tool.Name != "query_docs_filesystem_x" {
		t.Fatalf("got %q", tool.Name)
	}
}

func TestToolsCacheRoundtrip(t *testing.T) {
	dir := t.TempDir()
	mcpURL := "https://example.com/mcp"
	entry := ToolsCache{Search: "search_x", FS: "query_docs_filesystem_x"}

	if err := SaveToolsCache(dir, mcpURL, entry); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, ok := LoadToolsCache(dir, mcpURL, time.Hour)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if got.Search != "search_x" || got.FS != "query_docs_filesystem_x" {
		t.Fatalf("unexpected cache: %+v", got)
	}
}

func TestToolsCache_ExpiredMiss(t *testing.T) {
	dir := t.TempDir()
	mcpURL := "https://example.com/mcp"
	if err := SaveToolsCache(dir, mcpURL, ToolsCache{Search: "s", FS: "f"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	// TTL 0 ⇒ always expired.
	if _, ok := LoadToolsCache(dir, mcpURL, 0); ok {
		t.Fatalf("expected cache miss when TTL exceeded")
	}
}

func TestToolsCache_Invalidate(t *testing.T) {
	dir := t.TempDir()
	mcpURL := "https://example.com/mcp"
	if err := SaveToolsCache(dir, mcpURL, ToolsCache{Search: "s", FS: "f"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := InvalidateToolsCache(dir, mcpURL); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if _, ok := LoadToolsCache(dir, mcpURL, time.Hour); ok {
		t.Fatalf("expected miss after invalidate")
	}
}
