package cli

import (
	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/config"
)

var Version = "dev"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "msc",
		Short: "Mintlify Search CLI via hosted MCP",
		Long:  "msc queries hosted Mintlify MCP servers to search documentation.\nDesigned for humans and AI agents alike.",
	}

	cmd.Version = Version

	cmd.PersistentFlags().String("mcp-url", "", "MCP endpoint URL (overrides MSC_MCP_URL and config file)")

	cmd.AddCommand(
		newSearchCmd(),
		newOpenCmd(),
		newConfigCmd(),
		newDoctorCmd(),
	)

	return cmd
}

func resolveConfig(cmd *cobra.Command) (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if v, _ := cmd.Flags().GetString("mcp-url"); v != "" {
		cfg.MCPURL = v
	}

	return cfg, nil
}
