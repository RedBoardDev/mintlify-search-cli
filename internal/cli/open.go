package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/api"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <query>",
		Short: "Search and open the top result in a browser",
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

			client := api.NewClient(cfg.APIKey, cfg.Domain)
			results, err := client.Search(cmd.Context(), query, 1)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}
			if len(results) == 0 {
				fmt.Println("No results found.")
				return nil
			}

			url := "https://" + cfg.Domain + results[0].Path
			fmt.Printf("Opening: %s\n", url)
			return openBrowser(url)
		},
	}
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start() //nolint:gosec // Args from controlled switch, not user input.
}
