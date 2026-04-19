package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/cliapp"
	"github.com/redboard/mintlify-search-cli/internal/config"
	"github.com/redboard/mintlify-search-cli/internal/render"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage persistent configuration",
		Long: `config reads and writes ~/.config/msc/config.json (macOS: Library/Application Support/msc).

Supported keys:
  mcp_url              Mintlify MCP endpoint (required)
  default_limit        default --limit for search (1..20)
  cache.enabled        reserved for future opt-in search cache
  cache.ttl_seconds    reserved for future opt-in search cache
  cache.tools_ttl_seconds  TTL of the tools-discovery cache (default 86400)
`,
	}
	cmd.AddCommand(newConfigGetCmd(), newConfigSetCmd(), newConfigListCmd())
	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print the value of a config key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return cliapp.Wrap(cliapp.ExitConfig, err)
			}
			v, err := getConfigField(cfg, args[0])
			if err != nil {
				return cliapp.Wrap(cliapp.ExitUsage, err)
			}
			fmt.Fprintln(os.Stdout, v)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Persist a config key/value pair",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return cliapp.Wrap(cliapp.ExitConfig, err)
			}
			if err := setConfigField(cfg, args[0], args[1]); err != nil {
				// Unknown key → usage (the user typed a wrong key name);
				// any other error (type conversion, URL validation) → config.
				if strings.HasPrefix(err.Error(), "unknown config key") {
					return cliapp.Wrap(cliapp.ExitUsage, err)
				}
				return cliapp.Wrap(cliapp.ExitConfig, err)
			}
			if err := cfg.Validate(); err != nil {
				return cliapp.Wrap(cliapp.ExitConfig, err)
			}
			if err := config.Save(cfg); err != nil {
				return cliapp.Wrap(cliapp.ExitConfig, err)
			}
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the effective configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return cliapp.Wrap(cliapp.ExitConfig, err)
			}
			path, _ := config.ConfigFilePath()
			payload := render.ConfigPayload{
				MCPURL:       cfg.MCPURL,
				DefaultLimit: cfg.DefaultLimit,
				Cache: render.CachePayload{
					Enabled:         cfg.Cache.Enabled,
					TTLSeconds:      cfg.Cache.TTLSeconds,
					ToolsTTLSeconds: cfg.Cache.ToolsTTLSeconds,
				},
				Path: path,
			}
			format := render.FormatText
			if asJSON {
				format = render.FormatJSON
			}
			return render.New(format).Render(os.Stdout, payload)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	return cmd
}

func getConfigField(cfg *config.Config, key string) (string, error) {
	switch strings.ToLower(key) {
	case "mcp_url":
		return cfg.MCPURL, nil
	case "default_limit":
		return strconv.Itoa(cfg.DefaultLimit), nil
	case "cache.enabled":
		return strconv.FormatBool(cfg.Cache.Enabled), nil
	case "cache.ttl_seconds":
		return strconv.Itoa(cfg.Cache.TTLSeconds), nil
	case "cache.tools_ttl_seconds":
		return strconv.Itoa(cfg.Cache.ToolsTTLSeconds), nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func setConfigField(cfg *config.Config, key, value string) error {
	switch strings.ToLower(key) {
	case "mcp_url":
		if err := config.ValidateMCPURL(value); err != nil {
			return err
		}
		cfg.MCPURL = value
	case "default_limit":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("default_limit: %w", err)
		}
		cfg.DefaultLimit = n
	case "cache.enabled":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("cache.enabled: %w", err)
		}
		cfg.Cache.Enabled = b
	case "cache.ttl_seconds":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("cache.ttl_seconds: %w", err)
		}
		cfg.Cache.TTLSeconds = n
	case "cache.tools_ttl_seconds":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("cache.tools_ttl_seconds: %w", err)
		}
		cfg.Cache.ToolsTTLSeconds = n
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}
