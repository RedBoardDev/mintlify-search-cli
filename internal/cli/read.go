package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/cliapp"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
	"github.com/redboard/mintlify-search-cli/internal/render"
)

func newReadCmd() *cobra.Command {
	var (
		asJSON bool
		asRaw  bool
		lines  int
		from   int
		to     int
	)

	cmd := &cobra.Command{
		Use:   "read <path>",
		Short: "Read a doc page by path",
		Long: `read fetches the full markdown of a documentation page from the MCP sandbox.

The path may include or omit the extension; msc tries .mdx, .md, then /index.mdx
in order until one resolves. Path returned by "msc search --json" (field "page")
is consumable as-is.

Examples:
  msc read /Auth-Rs/v2/api-bo/authenticate-and-log-in-a-backoffice-user
  msc read /Rule/v2/api-rs/read/list-rules-for-the-given-site-ids --lines 50
  msc read /openapi/global/openapi.yaml --from 1 --to 80
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRead(cmd, args[0], readFlags{json: asJSON, raw: asRaw, lines: lines, from: from, to: to})
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "wrap result in JSON {page, resolved_path, content, truncated, bytes}")
	cmd.Flags().BoolVar(&asRaw, "raw", false, "output the raw MCP result payload")
	cmd.Flags().IntVar(&lines, "lines", 0, "equivalent to head -N")
	cmd.Flags().IntVar(&from, "from", 0, "start line (requires --to)")
	cmd.Flags().IntVar(&to, "to", 0, "end line (requires --from)")
	return cmd
}

type readFlags struct {
	json, raw       bool
	lines, from, to int
}

func runRead(cmd *cobra.Command, rawPath string, f readFlags) error {
	if f.json && f.raw {
		return cliapp.Newf(cliapp.ExitUsage, "--json and --raw are mutually exclusive")
	}
	if f.lines > 0 && (f.from > 0 || f.to > 0) {
		return cliapp.Newf(cliapp.ExitUsage, "--lines cannot be combined with --from/--to")
	}
	if (f.from > 0) != (f.to > 0) {
		return cliapp.Newf(cliapp.ExitUsage, "--from and --to must be specified together")
	}
	if f.from > 0 && f.from > f.to {
		return cliapp.Newf(cliapp.ExitUsage, "--from must be <= --to")
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

	candidates := ResolvePathCandidates(rawPath)
	var (
		chosen  string
		result  *mcp.FSResult
		rawResp []byte
	)
	for _, candidate := range candidates {
		command := buildReadCommand(candidate, f)
		resp, err := app.Client.CallTool(ctx, toolName, map[string]any{"command": command})
		if err != nil {
			return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("read: %w", err))
		}
		r, err := mcp.ParseFSResult(resp)
		if err != nil {
			return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("read: %w", err))
		}
		if r.Exit == 0 {
			chosen = candidate
			result = r
			rawResp = resp.Result
			break
		}
		if !f.raw {
			// keep trying siblings — cat returned non-zero, likely missing file
			continue
		}
		chosen = candidate
		result = r
		rawResp = resp.Result
		break
	}
	if result == nil {
		return cliapp.Newf(cliapp.ExitRuntime, "read: no matching page for %q (tried: %s)", rawPath, strings.Join(candidates, ", "))
	}

	if f.raw {
		return app.Renderer.Render(os.Stdout, render.RawPayload{Result: rawResp})
	}

	if f.json {
		return app.Renderer.Render(os.Stdout, render.ReadPayload{
			Page:         canonicalPage(chosen),
			ResolvedPath: chosen,
			Content:      result.Stdout,
			Truncated:    result.Truncated,
			Bytes:        result.Bytes,
		})
	}

	if err := app.Renderer.Render(os.Stdout, render.ReadPayload{Content: result.Stdout}); err != nil {
		return err
	}
	if result.Truncated {
		fmt.Fprintln(os.Stderr, "[truncated: output > 30 KB]")
	}
	if result.Exit != 0 {
		return cliapp.Wrap(cliapp.ExitRuntime, fmt.Errorf("read exited with code %d", result.Exit))
	}
	return nil
}

// ResolvePathCandidates returns the ordered list of paths to probe for a
// user-supplied path. If the input already has an extension, only that path
// is tried; otherwise we try .mdx, .md, /index.mdx, and finally the bare
// path (for directories someone might already expect the server to expose
// as a file).
func ResolvePathCandidates(raw string) []string {
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	if filepath.Ext(raw) != "" {
		return []string{raw}
	}
	return []string{
		raw + ".mdx",
		raw + ".md",
		strings.TrimRight(raw, "/") + "/index.mdx",
		raw,
	}
}

func buildReadCommand(path string, f readFlags) string {
	quoted := shellQuote(path)
	switch {
	case f.lines > 0:
		return fmt.Sprintf("head -n %d %s", f.lines, quoted)
	case f.from > 0 && f.to > 0:
		return fmt.Sprintf("sed -n '%d,%dp' %s", f.from, f.to, quoted)
	default:
		return fmt.Sprintf("cat %s", quoted)
	}
}

// canonicalPage strips the file extension so the returned "page" field is
// round-trippable as a search → read input.
func canonicalPage(path string) string {
	return strings.TrimPrefix(strings.TrimSuffix(path, filepath.Ext(path)), "/")
}

// shellQuote wraps s in single quotes, escaping any embedded single quote.
// Used to pass filesystem paths to the sandbox without breaking on spaces.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
