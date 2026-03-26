package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage msc configuration",
	}

	cmd.AddCommand(
		newSetMCPURLCmd(),
		newShowCmd(),
	)

	return cmd
}

func newSetMCPURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-mcp-url <url>",
		Short: "Set the Mintlify MCP URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.MCPURL = args[0]
			if err := cfg.Validate(); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("MCP URL saved.")
			return nil
		},
	}
}

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}

			path, _ := config.ConfigFilePath()
			_, err = fmt.Fprintf(os.Stdout, "Config file: %s\n\n%s\n", path, string(data))
			return err
		},
	}
}
