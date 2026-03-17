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
		newSetKeyCmd(),
		newSetDomainCmd(),
		newShowCmd(),
	)

	return cmd
}

func newSetKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-key <api-key>",
		Short: "Set the Mintlify API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.APIKey = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("API key saved.")
			return nil
		},
	}
}

func newSetDomainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-domain <domain>",
		Short: "Set the documentation domain (e.g. docs.example.com)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Domain = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("Domain saved.")
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
