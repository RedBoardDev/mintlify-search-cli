package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/cliapp"
	"github.com/redboard/mintlify-search-cli/internal/config"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
	"github.com/redboard/mintlify-search-cli/internal/render"
)

func newSearchCmd() *cobra.Command {
	var (
		asJSON bool
		asText bool
		asRaw  bool
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Semantic search of the documentation",
		Long: `search runs a semantic query against the Mintlify MCP server and prints the top results.

Default output is minified JSON (LLM-friendly). Use --text for a human listing, or --raw
to inspect the unparsed MCP payload.

Examples:
  msc search "rate limiting"
  msc search "create customer" --limit 3
  msc search "auth" --text
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd, args, searchFlags{json: asJSON, text: asText, raw: asRaw, limit: limit})
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output minified JSON (default)")
	cmd.Flags().BoolVar(&asText, "text", false, "output human-readable listing")
	cmd.Flags().BoolVar(&asRaw, "raw", false, "output the raw MCP result payload")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "cap the number of results (default from config, max 20)")
	return cmd
}

type searchFlags struct {
	json, text, raw bool
	limit           int
}

func runSearch(cmd *cobra.Command, args []string, f searchFlags) error {
	if err := exclusiveFormat(f.json, f.text, f.raw); err != nil {
		return err
	}
	format := pickFormat(f.json, f.text, f.raw)
	if f.raw && f.limit > 0 {
		return cliapp.Newf(cliapp.ExitUsage, "--raw is incompatible with --limit")
	}

	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return cliapp.Newf(cliapp.ExitUsage, "search query must be non-empty")
	}

	app, err := cliapp.FromCmd(cmd, format)
	if err != nil {
		return err
	}

	limit := f.limit
	if limit <= 0 {
		limit = app.Cfg.DefaultLimit
	}
	if limit > config.MaxSearchLimit {
		limit = config.MaxSearchLimit
	}

	ctx, cancel := context.WithTimeout(context.Background(), app.Timeout)
	defer cancel()

	toolName, err := app.ResolveSearchTool(ctx)
	if err != nil {
		return err
	}
	resp, err := app.Client.CallTool(ctx, toolName, map[string]any{"query": query})
	if err != nil {
		return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("search: %w", err))
	}

	if f.raw {
		return app.Renderer.Render(os.Stdout, render.RawPayload{Result: resp.Result})
	}

	blocks, err := mcp.ParseSearchBlocks(resp)
	if err != nil {
		return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("search: %w", err))
	}
	if len(blocks) > limit {
		blocks = blocks[:limit]
	}
	payload := render.SearchPayload{Query: query, Results: make([]render.SearchEntry, 0, len(blocks))}
	for _, b := range blocks {
		payload.Results = append(payload.Results, render.SearchEntry{
			Title:   b.Title,
			URL:     b.URL,
			Page:    b.Page,
			Content: b.Content,
		})
	}
	return app.Renderer.Render(os.Stdout, payload)
}

func exclusiveFormat(json, text, raw bool) error {
	count := 0
	for _, b := range []bool{json, text, raw} {
		if b {
			count++
		}
	}
	if count > 1 {
		return cliapp.Newf(cliapp.ExitUsage, "--json, --text and --raw are mutually exclusive")
	}
	return nil
}

func pickFormat(asJSON, asText, asRaw bool) render.FormatKind {
	switch {
	case asRaw:
		return render.FormatRaw
	case asText:
		return render.FormatText
	default:
		return render.FormatJSON
	}
}

