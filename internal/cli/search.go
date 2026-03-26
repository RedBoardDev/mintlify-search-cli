package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/cache"
	"github.com/redboard/mintlify-search-cli/internal/config"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
	"github.com/redboard/mintlify-search-cli/internal/output"
)

func newSearchCmd() *cobra.Command {
	var (
		jsonFlag bool
		rawFlag  bool
		limit    int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search documentation via Mintlify MCP",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonFlag && rawFlag {
				return fmt.Errorf("--json and --raw cannot be used together")
			}
			if rawFlag && cmd.Flags().Changed("limit") {
				return fmt.Errorf("--raw and --limit cannot be used together")
			}

			cfg, err := resolveConfig(cmd)
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}

			query := strings.Join(args, " ")
			mode := searchModeText
			if jsonFlag {
				mode = searchModeJSON
			}
			if rawFlag {
				mode = searchModeRaw
			}

			return runSearch(cmd, cfg, query, limit, mode)
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output normalized minified JSON")
	cmd.Flags().BoolVar(&rawFlag, "raw", false, "Output raw MCP JSON-RPC payload")
	cmd.Flags().IntVar(&limit, "limit", 5, "Max number of results")

	return cmd
}

type searchMode int

const (
	searchModeText searchMode = iota
	searchModeJSON
	searchModeRaw
)

func runSearch(cmd *cobra.Command, cfg *config.Config, query string, limit int, mode searchMode) error {
	if mode == searchModeRaw {
		return runRawSearch(cmd, cfg, query)
	}

	results, err := loadCachedResults(cfg.MCPURL, query, limit)
	if err != nil {
		return err
	}
	if results == nil {
		results, err = fetchNormalizedResults(cmd, cfg, query)
		if err != nil {
			return err
		}
		results = applyLimit(results, limit)
		_ = saveCachedResults(cfg.MCPURL, query, limit, results)
	}

	format := output.FormatText
	if mode == searchModeJSON {
		format = output.FormatJSON
	}
	return output.Render(os.Stdout, results, format)
}

func runRawSearch(cmd *cobra.Command, cfg *config.Config, query string) error {
	client := mcp.NewClient(cfg.MCPURL)
	discovery, err := client.Discover(cmd.Context())
	if err != nil {
		return fmt.Errorf("discovering mcp server: %w", err)
	}

	tool, err := mcp.FindSearchTool(discovery)
	if err != nil {
		return fmt.Errorf("finding search tool: %w", err)
	}

	rawResp, err := client.CallTool(cmd.Context(), tool.Name, map[string]any{"query": query})
	if err != nil {
		return fmt.Errorf("calling search tool: %w", err)
	}

	data, err := mcp.MarshalMinified(rawResp)
	if err != nil {
		return err
	}
	if _, err := os.Stdout.Write(data); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout)
	return err
}

func fetchNormalizedResults(cmd *cobra.Command, cfg *config.Config, query string) ([]mcp.SearchResult, error) {
	client := mcp.NewClient(cfg.MCPURL)
	discovery, err := client.Discover(cmd.Context())
	if err != nil {
		return nil, fmt.Errorf("discovering mcp server: %w", err)
	}

	tool, err := mcp.FindSearchTool(discovery)
	if err != nil {
		return nil, fmt.Errorf("finding search tool: %w", err)
	}

	rawResp, err := client.CallTool(cmd.Context(), tool.Name, map[string]any{"query": query})
	if err != nil {
		return nil, fmt.Errorf("calling search tool: %w", err)
	}

	call, err := mcp.ParseToolCallResult(rawResp)
	if err != nil {
		return nil, fmt.Errorf("decoding search results: %w", err)
	}

	return mcp.NormalizeSearchResults(call), nil
}

func applyLimit(results []mcp.SearchResult, limit int) []mcp.SearchResult {
	if limit <= 0 || len(results) <= limit {
		return results
	}
	return results[:limit]
}

func loadCachedResults(mcpURL, query string, limit int) ([]mcp.SearchResult, error) {
	cacheDir, _ := config.CacheDir()
	if cacheDir == "" {
		return nil, nil
	}

	c := cache.New(cacheDir)
	key := cache.Key(mcpURL, query, strconv.Itoa(limit))
	data, err := c.Get(key)
	if err != nil || data == nil {
		return nil, err
	}

	var results []mcp.SearchResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func saveCachedResults(mcpURL, query string, limit int, results []mcp.SearchResult) error {
	cacheDir, _ := config.CacheDir()
	if cacheDir == "" {
		return nil
	}

	data, err := json.Marshal(results)
	if err != nil {
		return nil
	}

	c := cache.New(cacheDir)
	return c.Set(cache.Key(mcpURL, query, strconv.Itoa(limit)), data)
}
