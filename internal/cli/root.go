// Package cli wires Cobra commands on top of the cliapp execution context.
// Each subcommand lives in its own file; this file owns the root command,
// persistent flags, and error routing.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/cliapp"
)

// Version is overridden at build time via -ldflags.
var Version = "dev"

// NewRootCmd builds the root Cobra command. Callers typically invoke
// `cmd.Execute()` and then route the returned error through RunAndExit.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "msc",
		Short:         "Mintlify Search CLI — LLM-first MCP client",
		Long:          longRoot,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Version = Version

	pf := cmd.PersistentFlags()
	pf.String("mcp-url", "", "MCP endpoint URL (overrides MSC_MCP_URL and config)")
	pf.Int("timeout", 30, "HTTP timeout per request, in seconds")
	pf.Bool("no-cache", false, "disable tools-discovery and search caches for this invocation")
	pf.BoolP("verbose", "v", false, "trace HTTP round-trips on stderr")

	cmd.AddCommand(
		newSearchCmd(),
		newFsCmd(),
		newReadCmd(),
		newOpenCmd(),
		newDoctorCmd(),
		newConfigCmd(),
		newCompletionCmd(),
	)
	return cmd
}

// RunAndExit executes root and converts the returned error into a meaningful
// os.Exit code. The message is printed to stderr with the `msc: error: ` prefix.
func RunAndExit(root *cobra.Command) {
	err := root.Execute()
	if err == nil {
		return
	}
	ee := cliapp.MapError(err)
	printError(os.Stderr, ee.Err)
	os.Exit(ee.Code)
}

func printError(w io.Writer, err error) {
	if err == nil {
		return
	}
	// Skip the prefix for nested ExitError (already formatted).
	var ee *cliapp.ExitError
	if errors.As(err, &ee) {
		err = ee.Err
	}
	fmt.Fprintf(w, "msc: error: %s\n", err.Error())
}

const longRoot = `msc queries Mintlify-hosted MCP servers to search and read API documentation.
Designed for LLM agents: JSON-first output for structured commands, raw output for content commands.

Quickstart:
  msc config set mcp_url https://api-documentation.example.com/mcp
  msc doctor
  msc search "authentication"
  msc open "list users"
`
