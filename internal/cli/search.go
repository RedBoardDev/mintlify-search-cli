package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/api"
	"github.com/redboard/mintlify-search-cli/internal/cache"
	"github.com/redboard/mintlify-search-cli/internal/config"
	"github.com/redboard/mintlify-search-cli/internal/output"
)

func newSearchCmd() *cobra.Command {
	var (
		jsonFlag bool
		limit    int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search documentation",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := resolveConfig(cmd)
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}

			query := strings.Join(args, " ")
			return runSearch(cmd, cfg, query, limit, jsonFlag)
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as minified JSON")
	cmd.Flags().IntVar(&limit, "limit", 5, "Max number of results")

	return cmd
}

func runSearch(cmd *cobra.Command, cfg *config.Config, query string, limit int, jsonOut bool) error {
	ctx := cmd.Context()
	cacheKey := cache.Key(cfg.Domain, query, strconv.Itoa(limit))

	cacheDir, _ := config.CacheDir()
	var c *cache.Cache
	if cacheDir != "" {
		c = cache.New(cacheDir)
		if data, err := c.Get(cacheKey); err == nil && data != nil {
			return outputCached(data, cfg.Domain, jsonOut)
		}
	}

	client := api.NewClient(cfg.APIKey, cfg.Domain)
	results, err := client.Search(ctx, query, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if c != nil {
		if data, err := json.Marshal(results); err == nil {
			_ = c.Set(cacheKey, data) // Best-effort: cache failure is non-fatal.
		}
	}

	format := output.FormatText
	if jsonOut {
		format = output.FormatJSON
	}
	return output.Render(os.Stdout, results, cfg.Domain, format)
}

func outputCached(data []byte, domain string, jsonOut bool) error {
	if jsonOut {
		if _, err := os.Stdout.Write(data); err != nil {
			return err
		}
		_, err := fmt.Fprintln(os.Stdout)
		return err
	}

	var results []api.SearchResult
	if err := json.Unmarshal(data, &results); err != nil {
		return err
	}
	return output.Render(os.Stdout, results, domain, output.FormatText)
}
