package cli

import (
	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/config"
)

var Version = "dev"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "msc",
		Short: "Mintlify Search CLI — fast documentation retrieval",
		Long:  "msc queries the Mintlify Discovery API to search documentation.\nDesigned for humans and AI agents alike.",
	}

	cmd.Version = Version

	cmd.PersistentFlags().String("api-key", "", "Mintlify API key (overrides MSC_API_KEY and config file)")
	cmd.PersistentFlags().String("domain", "", "Documentation domain (overrides MSC_DOMAIN and config file)")

	cmd.AddCommand(
		newSearchCmd(),
		newOpenCmd(),
		newConfigCmd(),
		newDoctorCmd(),
	)

	return cmd
}

// Precedence: flags > env > file.
func resolveConfig(cmd *cobra.Command) (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if v, _ := cmd.Flags().GetString("api-key"); v != "" {
		cfg.APIKey = v
	}
	if v, _ := cmd.Flags().GetString("domain"); v != "" {
		cfg.Domain = v
	}

	return cfg, nil
}
