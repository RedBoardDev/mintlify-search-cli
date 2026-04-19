package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// searchToolPrefix and fsToolPrefix are the name prefixes used by the
// Mintlify MCP server to identify its two tools. The suffix varies per
// customer (e.g. search_kare_api_documentation), so we match on prefix.
const (
	searchToolPrefix = "search_"
	fsToolPrefix     = "query_docs_filesystem_"
)

// FindSearchTool selects the semantic search tool from a tools/list response.
// It prefers tools whose name starts with `search_` and require a `query`
// string input; if exactly one tool is advertised, it is returned as-is.
func FindSearchTool(tools []ToolDefinition) (*ToolDefinition, error) {
	if len(tools) == 1 {
		t := tools[0]
		return &t, nil
	}
	for _, tool := range tools {
		if strings.HasPrefix(tool.Name, searchToolPrefix) && requiresStringParam(tool.InputSchema, "query") {
			t := tool
			return &t, nil
		}
	}
	return nil, errors.New("no search tool with required query input found")
}

// FindFSTool selects the docs filesystem tool from a tools/list response by
// matching the `query_docs_filesystem_` prefix and requiring a `command`
// string input.
func FindFSTool(tools []ToolDefinition) (*ToolDefinition, error) {
	for _, tool := range tools {
		if strings.HasPrefix(tool.Name, fsToolPrefix) && requiresStringParam(tool.InputSchema, "command") {
			t := tool
			return &t, nil
		}
	}
	return nil, errors.New("no docs filesystem tool with required command input found")
}

func requiresStringParam(schema InputSchema, name string) bool {
	prop, ok := schema.Properties[name]
	if !ok || prop.Type != "string" {
		return false
	}
	return slices.Contains(schema.Required, name)
}

// ToolsCache is the persisted record of the tool names discovered for a
// given MCP endpoint. Cached on disk to avoid paying the ~1s initialize RTT
// on every CLI invocation.
type ToolsCache struct {
	Search  string    `json:"search"`
	FS      string    `json:"fs"`
	SavedAt time.Time `json:"saved_at"`
}

// LoadToolsCache reads the cache entry for the given MCP URL. Returns
// (nil, false) on miss or when the cached entry is older than ttl.
func LoadToolsCache(cacheDir, mcpURL string, ttl time.Duration) (*ToolsCache, bool) {
	path := toolsCachePath(cacheDir, mcpURL)
	data, err := os.ReadFile(path) //nolint:gosec // Path built from trusted cache dir + SHA256.
	if err != nil {
		return nil, false
	}
	var c ToolsCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, false
	}
	if time.Since(c.SavedAt) > ttl {
		return nil, false
	}
	if c.Search == "" || c.FS == "" {
		return nil, false
	}
	return &c, true
}

// SaveToolsCache writes a cache entry for the given MCP URL. Best-effort: on
// any IO error, returns the error but the caller is free to ignore it.
func SaveToolsCache(cacheDir, mcpURL string, c ToolsCache) error {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	c.SavedAt = time.Now().UTC()
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(toolsCachePath(cacheDir, mcpURL), data, 0o600)
}

// InvalidateToolsCache removes the cache entry for the given MCP URL.
// Used by `msc doctor` to force re-discovery.
func InvalidateToolsCache(cacheDir, mcpURL string) error {
	err := os.Remove(toolsCachePath(cacheDir, mcpURL))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func toolsCachePath(cacheDir, mcpURL string) string {
	h := sha256.Sum256([]byte(mcpURL))
	name := "tools." + hex.EncodeToString(h[:])[:16] + ".json"
	return filepath.Join(cacheDir, name)
}
