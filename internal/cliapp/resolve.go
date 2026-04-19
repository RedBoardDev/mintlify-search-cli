package cliapp

import (
	"context"
	"fmt"
	"time"

	"github.com/redboard/mintlify-search-cli/internal/mcp"
)

// ResolveSearchTool returns the name of the search tool to call, using the
// on-disk tools cache when fresh. A cache miss triggers an initialize +
// tools/list round-trip against the MCP server.
func (a *App) ResolveSearchTool(ctx context.Context) (string, error) {
	return a.resolveTool(ctx, func(c *mcp.ToolsCache) string { return c.Search }, mcp.FindSearchTool, "search")
}

// ResolveFSTool is the fs equivalent of ResolveSearchTool.
func (a *App) ResolveFSTool(ctx context.Context) (string, error) {
	return a.resolveTool(ctx, func(c *mcp.ToolsCache) string { return c.FS }, mcp.FindFSTool, "fs")
}

type toolPicker func([]mcp.ToolDefinition) (*mcp.ToolDefinition, error)

func (a *App) resolveTool(ctx context.Context, field func(*mcp.ToolsCache) string, pick toolPicker, label string) (string, error) {
	ttl := time.Duration(a.Cfg.Cache.ToolsTTLSeconds) * time.Second
	if !a.NoCache {
		if c, ok := mcp.LoadToolsCache(a.CacheDir, a.Cfg.MCPURL, ttl); ok {
			if name := field(c); name != "" {
				return name, nil
			}
		}
	}

	name, err := a.discoverAndCache(ctx)
	if err != nil {
		return "", err
	}
	_ = name // discoverAndCache persists both names; we return the picked one
	c, _ := mcp.LoadToolsCache(a.CacheDir, a.Cfg.MCPURL, ttl)
	if c == nil {
		return "", Wrap(ExitRuntime, fmt.Errorf("tools cache missing after discovery"))
	}
	got := field(c)
	if got == "" {
		return "", Wrap(ExitRuntime, fmt.Errorf("no %s tool exposed by server", label))
	}
	return got, nil
}

// discoverAndCache runs initialize + tools/list, then persists the matched
// search and fs tool names to the on-disk cache.
func (a *App) discoverAndCache(ctx context.Context) (string, error) {
	tools, err := a.Client.ListTools(ctx)
	if err != nil {
		return "", Wrap(ExitRuntime, fmt.Errorf("listing tools: %w", err))
	}
	searchTool, serr := mcp.FindSearchTool(tools)
	fsTool, ferr := mcp.FindFSTool(tools)
	if serr != nil && ferr != nil {
		return "", Wrap(ExitRuntime, fmt.Errorf("no matching tools advertised by server"))
	}

	entry := mcp.ToolsCache{}
	if searchTool != nil {
		entry.Search = searchTool.Name
	}
	if fsTool != nil {
		entry.FS = fsTool.Name
	}
	if !a.NoCache {
		_ = mcp.SaveToolsCache(a.CacheDir, a.Cfg.MCPURL, entry)
	} else {
		// Write anyway so the subsequent field lookup finds it — the cache
		// file reflects this run and will be evicted by TTL eventually.
		_ = mcp.SaveToolsCache(a.CacheDir, a.Cfg.MCPURL, entry)
	}
	return "", nil
}
