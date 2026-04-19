package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/cliapp"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
	"github.com/redboard/mintlify-search-cli/internal/render"
)

func newFsCmd() *cobra.Command {
	var (
		asJSON bool
		asRaw  bool
	)

	cmd := &cobra.Command{
		Use:   "fs <command>",
		Short: "Run a shell command in the MCP docs sandbox",
		Long: `fs sends a shell-style command to the Mintlify docs filesystem sandbox and
prints the result. Supported binaries: rg, grep, find, tree, ls, cat, head, tail,
stat, wc, sort, uniq, cut, sed, awk, jq.

Quote the command as a single argument. The sandbox is read-only, stateless, and
truncates output at ~30 KB per call.

Examples:
  msc fs "tree / -L 2"
  msc fs "rg -il 'rate limit' /"
  msc fs "cat /openapi/global/openapi.yaml | jq '.paths | keys | length'"
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFs(cmd, args, fsFlags{json: asJSON, raw: asRaw})
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "wrap result in JSON {exit_code, stdout, stderr, truncated, bytes}")
	cmd.Flags().BoolVar(&asRaw, "raw", false, "output the raw MCP result payload")
	return cmd
}

type fsFlags struct {
	json, raw bool
}

func runFs(cmd *cobra.Command, args []string, f fsFlags) error {
	if f.json && f.raw {
		return cliapp.Newf(cliapp.ExitUsage, "--json and --raw are mutually exclusive")
	}
	if len(args) > 1 {
		return cliapp.Newf(cliapp.ExitUsage, "pass the command as a single quoted argument, e.g. msc fs \"tree / -L 2\"")
	}
	command := args[0]
	if command == "" {
		return cliapp.Newf(cliapp.ExitUsage, "fs command must be non-empty")
	}

	format := render.FormatText
	if f.json {
		format = render.FormatJSON
	} else if f.raw {
		format = render.FormatRaw
	}

	app, err := cliapp.FromCmd(cmd, format)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), app.Timeout)
	defer cancel()

	toolName, err := app.ResolveFSTool(ctx)
	if err != nil {
		return err
	}
	resp, err := app.Client.CallTool(ctx, toolName, map[string]any{"command": command})
	if err != nil {
		return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("fs: %w", err))
	}

	if f.raw {
		return app.Renderer.Render(os.Stdout, render.RawPayload{Result: resp.Result})
	}

	res, err := mcp.ParseFSResult(resp)
	if err != nil {
		return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("fs: %w", err))
	}

	if f.json {
		if renderErr := app.Renderer.Render(os.Stdout, fsToPayload(res)); renderErr != nil {
			return renderErr
		}
	} else {
		// Text/default: mirror shell. Stdout → stdout, stderr → stderr.
		if res.Stdout != "" {
			if renderErr := app.Renderer.Render(os.Stdout, fsStdoutOnly(res)); renderErr != nil {
				return renderErr
			}
		}
		if res.Stderr != "" {
			fmt.Fprintln(os.Stderr, res.Stderr)
		}
		if res.Truncated {
			fmt.Fprintln(os.Stderr, "[truncated: output > 30 KB]")
		}
	}

	if res.Exit != 0 {
		return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("fs exited with code %d", res.Exit))
	}
	return nil
}

func fsToPayload(r *mcp.FSResult) render.FSPayload {
	return render.FSPayload{
		ExitCode:  r.Exit,
		Stdout:    r.Stdout,
		Stderr:    r.Stderr,
		Truncated: r.Truncated,
		Bytes:     r.Bytes,
	}
}

func fsStdoutOnly(r *mcp.FSResult) render.FSPayload {
	return render.FSPayload{Stdout: r.Stdout}
}
