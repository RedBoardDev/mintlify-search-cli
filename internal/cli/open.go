package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/cliapp"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
	"github.com/redboard/mintlify-search-cli/internal/render"
)

func newOpenCmd() *cobra.Command {
	var (
		asJSON bool
		asRaw  bool
		lines  int
	)

	cmd := &cobra.Command{
		Use:   "open <query>",
		Short: "Search + read the top result in one call",
		Long: `open runs a search, picks the top result, and prints the full markdown of its page.

Breaking change vs. v1: this no longer opens a browser — the command now produces
the page content on stdout so LLM agents can consume it.

Examples:
  msc open "list users"
  msc open "rate limit" --lines 80
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpen(cmd, args, openFlags{json: asJSON, raw: asRaw, lines: lines})
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "wrap result in JSON {title, url, page, resolved_path, content, truncated}")
	cmd.Flags().BoolVar(&asRaw, "raw", false, "output the raw MCP result payloads (search + cat concatenated)")
	cmd.Flags().IntVar(&lines, "lines", 0, "equivalent to head -N on the page")
	return cmd
}

type openFlags struct {
	json, raw bool
	lines     int
}

func runOpen(cmd *cobra.Command, args []string, f openFlags) error {
	if f.json && f.raw {
		return cliapp.Newf(cliapp.ExitUsage, "--json and --raw are mutually exclusive")
	}
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return cliapp.Newf(cliapp.ExitUsage, "open query must be non-empty")
	}

	format := render.FormatText
	if f.json {
		format = render.FormatJSON
	} else if f.raw {
		format = render.FormatRaw
	}

	app, err := cliapp.FromCmd(cmd, format)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), app.Timeout)
	defer cancel()

	searchTool, err := app.ResolveSearchTool(ctx)
	if err != nil {
		return err
	}
	fsTool, err := app.ResolveFSTool(ctx)
	if err != nil {
		return err
	}

	searchResp, err := app.Client.CallTool(ctx, searchTool, map[string]any{"query": query})
	if err != nil {
		return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("open search: %w", err))
	}
	blocks, err := mcp.ParseSearchBlocks(searchResp)
	if err != nil {
		return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("open search: %w", err))
	}
	if len(blocks) == 0 {
		return cliapp.Newf(cliapp.ExitRuntime, "no results for query %q", query)
	}
	top := blocks[0]

	// Build the read path: Page: is preferred (no extension, no leading slash);
	// fall back to URL-path stem if Page is empty.
	pagePath := top.Page
	if pagePath == "" {
		pagePath = extractPageFromURL(top.URL)
	}
	candidates := ResolvePathCandidates(pagePath)

	var (
		chosen    string
		fsResult  *mcp.FSResult
		fsRawResp json.RawMessage
	)
	for _, candidate := range candidates {
		cmdStr := buildReadCommand(candidate, readFlags{lines: f.lines})
		resp, err := app.Client.CallTool(ctx, fsTool, map[string]any{"command": cmdStr})
		if err != nil {
			return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("open read: %w", err))
		}
		r, err := mcp.ParseFSResult(resp)
		if err != nil {
			return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("open read: %w", err))
		}
		if r.Exit == 0 {
			chosen = candidate
			fsResult = r
			fsRawResp = resp.Result
			break
		}
	}
	if fsResult == nil {
		return cliapp.Newf(cliapp.ExitRuntime, "no readable page for top hit %q (page=%q)", top.Title, pagePath)
	}

	if f.raw {
		merged := struct {
			Search json.RawMessage `json:"search"`
			Read   json.RawMessage `json:"read"`
		}{Search: searchResp.Result, Read: fsRawResp}
		return app.Renderer.Render(os.Stdout, merged)
	}

	if f.json {
		return app.Renderer.Render(os.Stdout, render.OpenPayload{
			Title:        top.Title,
			URL:          top.URL,
			Page:         top.Page,
			ResolvedPath: chosen,
			Content:      fsResult.Stdout,
			Truncated:    fsResult.Truncated,
		})
	}

	if err := app.Renderer.Render(os.Stdout, render.OpenPayload{Content: fsResult.Stdout}); err != nil {
		return err
	}
	if fsResult.Truncated {
		fmt.Fprintln(os.Stderr, "[truncated: output > 30 KB]")
	}
	return nil
}

// extractPageFromURL turns a full URL like https://x.com/Auth/v1/login into
// "Auth/v1/login" when the search block's Page field is missing.
func extractPageFromURL(rawURL string) string {
	idx := strings.Index(rawURL, "://")
	if idx < 0 {
		return ""
	}
	after := rawURL[idx+3:]
	slash := strings.Index(after, "/")
	if slash < 0 {
		return ""
	}
	return strings.TrimPrefix(after[slash:], "/")
}
