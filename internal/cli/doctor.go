package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/api"
	"github.com/redboard/mintlify-search-cli/internal/config"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration and connectivity",
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
	printSource("api_key", cfg.APIKey, config.EnvAPIKey, "api-key", cmd)
	printSource("domain", cfg.Domain, config.EnvDomain, "domain", cmd)

	switch {
	case cfg.APIKey == "":
		printCheck(false, "API key", "not set")
		allPassed = false
	case !strings.HasPrefix(cfg.APIKey, "mint_"):
		printCheck(false, "API key", "does not start with 'mint_'")
		allPassed = false
	default:
		masked := cfg.APIKey[:8] + "..." + cfg.APIKey[len(cfg.APIKey)-4:]
		printCheck(true, "API key", masked)
	}

	if cfg.Domain == "" {
		printCheck(false, "Domain", "not set")
		allPassed = false
	} else {
		printCheck(true, "Domain", cfg.Domain)
	}

	if cfg.APIKey != "" && cfg.Domain != "" {
		client := api.NewClient(cfg.APIKey, cfg.Domain)
		latency, err := client.Ping(cmd.Context())
		if err != nil {
			printCheck(false, "Connectivity", err.Error())
			allPassed = false
		} else {
			printCheck(true, "Connectivity", fmt.Sprintf("OK (%s)", latency.Round(time.Millisecond)))
		}
	} else {
		printCheck(false, "Connectivity", "skipped (incomplete config)")
		allPassed = false
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
