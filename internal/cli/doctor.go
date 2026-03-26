package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/config"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration and MCP connectivity",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd)
		},
	}
}

func runDoctor(cmd *cobra.Command) error {
	allPassed := true

	cfg, err := resolveConfig(cmd)
	if err != nil {
		printCheck(false, "Config", err.Error())
		return nil
	}

	path, _ := config.ConfigFilePath()
	printCheck(true, "Config file", path)
	printSource("mcp_url", cfg.MCPURL, config.EnvMCPURL, "mcp-url", cmd)

	if cfg.MCPURL == "" {
		printCheck(false, "MCP URL", "not set")
		allPassed = false
	} else if err := cfg.Validate(); err != nil {
		printCheck(false, "MCP URL", err.Error())
		allPassed = false
	} else {
		printCheck(true, "MCP URL", cfg.MCPURL)
	}

	if cfg.MCPURL == "" {
		printCheck(false, "Discovery", "skipped (incomplete config)")
		printCheck(false, "Initialize", "skipped (incomplete config)")
		printCheck(false, "Search tool", "skipped (incomplete config)")
		printCheck(false, "Search call", "skipped (incomplete config)")
		allPassed = false
	} else {
		client := mcp.NewClient(cfg.MCPURL)

		discovery, err := client.Discover(cmd.Context())
		if err != nil {
			printCheck(false, "Discovery", doctorHint(cfg.MCPURL, err))
			printCheck(false, "Initialize", "skipped (discovery failed)")
			printCheck(false, "Search tool", "skipped (discovery failed)")
			printCheck(false, "Search call", "skipped (discovery failed)")
			allPassed = false
		} else {
			printCheck(true, "Discovery", fmt.Sprintf("%s (%s)", discovery.Server.Name, discovery.Server.Transport))

			_, latency, err := client.Initialize(cmd.Context())
			if err != nil {
				printCheck(false, "Initialize", doctorHint(cfg.MCPURL, err))
				allPassed = false
			} else {
				printCheck(true, "Initialize", fmt.Sprintf("OK (%s)", latency.Round(time.Millisecond)))
			}

			tools, err := client.ListTools(cmd.Context())
			if err != nil {
				printCheck(false, "Search tool", err.Error())
				printCheck(false, "Search call", "skipped (tool listing failed)")
				allPassed = false
			} else {
				tool, err := mcp.FindSearchToolFromList(tools)
				if err != nil {
					printCheck(false, "Search tool", err.Error())
					printCheck(false, "Search call", "skipped (search tool missing)")
					allPassed = false
				} else {
					printCheck(true, "Search tool", tool.Name)

					rawResp, err := client.CallTool(cmd.Context(), tool.Name, map[string]any{"query": "test"})
					if err != nil {
						printCheck(false, "Search call", doctorHint(cfg.MCPURL, err))
						allPassed = false
					} else {
						call, err := mcp.ParseToolCallResult(rawResp)
						if err != nil {
							printCheck(false, "Search call", err.Error())
							allPassed = false
						} else {
							printCheck(true, "Search call", fmt.Sprintf("%d content block(s)", len(call.Content)))
						}
					}
				}
			}
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("All checks passed.")
	} else {
		fmt.Println("Some checks failed. Fix the issues above and re-run: msc doctor")
	}
	return nil
}

func printCheck(ok bool, name, detail string) {
	icon := "PASS"
	if !ok {
		icon = "FAIL"
	}
	fmt.Printf("  [%s] %s: %s\n", icon, name, detail)
}

func printSource(label, value, envVar, flagName string, cmd *cobra.Command) {
	if value == "" {
		return
	}

	source := "config file"
	if flagVal, _ := cmd.Flags().GetString(flagName); flagVal != "" {
		source = "flag --" + flagName
	} else if os.Getenv(envVar) != "" {
		source = "env " + envVar
	}

	fmt.Printf("         %s via %s\n", label, source)
}

func doctorHint(mcpURL string, err error) string {
	msg := err.Error()
	u, parseErr := url.Parse(mcpURL)
	if parseErr == nil && u.Path == "/authed/mcp" {
		return msg + " (authenticated MCP may require OAuth and configured redirect domains)"
	}
	if strings.Contains(msg, "status 500") {
		return msg + " (server-side MCP setup may be incomplete)"
	}
	return msg
}
