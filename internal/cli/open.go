package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
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
			results, err := fetchNormalizedResults(cmd, cfg, query)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			for _, result := range results {
				if result.URL == "" {
					continue
				}
				fmt.Printf("Opening: %s\n", result.URL)
				return openBrowser(result.URL)
			}

			fmt.Println("No results found.")
			return nil
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
