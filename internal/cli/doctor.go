package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/cliapp"
	"github.com/redboard/mintlify-search-cli/internal/config"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
	"github.com/redboard/mintlify-search-cli/internal/render"
)

func newDoctorCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration and MCP connectivity",
		Long: `doctor runs five diagnostic checks:
  1. Config file loaded and parsed
  2. mcp_url shape is valid
  3. initialize succeeds (reports latency)
  4. tools/list advertises a search and a fs tool
  5. a dummy search call returns non-empty content

The tools-discovery cache is invalidated before running so a stale cache
cannot mask server-side changes.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON array of checks")
	return cmd
}

func runDoctor(cmd *cobra.Command, asJSON bool) error {
	format := render.FormatText
	if asJSON {
		format = render.FormatJSON
	}
	payload := render.DoctorPayload{OK: true}
	addCheck := func(c render.DoctorCheck) {
		payload.Checks = append(payload.Checks, c)
		if c.Status != "pass" {
			payload.OK = false
		}
	}

	// Check 1: config file
	cfgPath, _ := config.ConfigFilePath()
	app, appErr := cliapp.FromCmd(cmd, format)
	if appErr != nil {
		addCheck(render.DoctorCheck{Name: "Config file", Status: "fail", Detail: appErr.Error()})
		// Render the partial report without an app (direct renderer).
		_ = render.New(format).Render(os.Stdout, payload)
		return cliapp.Newf(cliapp.ExitConfig, "config invalid")
	}
	addCheck(render.DoctorCheck{Name: "Config file", Status: "pass", Detail: cfgPath})

	// Check 2: MCP URL
	if err := config.ValidateMCPURL(app.Cfg.MCPURL); err != nil {
		addCheck(render.DoctorCheck{Name: "MCP URL", Status: "fail", Detail: err.Error()})
		return finishDoctor(app, payload)
	}
	addCheck(render.DoctorCheck{Name: "MCP URL", Status: "pass", Detail: app.Cfg.MCPURL})

	// Force re-discovery of tools.
	_ = mcp.InvalidateToolsCache(app.CacheDir, app.Cfg.MCPURL)
	app.NoCache = true

	ctx, cancel := context.WithTimeout(context.Background(), app.Timeout)
	defer cancel()

	// Check 3: initialize
	init, dur, err := app.Client.Initialize(ctx)
	if err != nil {
		addCheck(render.DoctorCheck{Name: "Initialize", Status: "fail", Detail: err.Error()})
		return finishDoctor(app, payload)
	}
	addCheck(render.DoctorCheck{Name: "Initialize", Status: "pass",
		Detail:     fmt.Sprintf("%s %s (protocol %s)", init.ServerInfo.Name, init.ServerInfo.Version, init.ProtocolVersion),
		DurationMs: dur.Milliseconds(),
	})

	// Check 4: tools discovered
	searchTool, err := app.ResolveSearchTool(ctx)
	if err != nil {
		addCheck(render.DoctorCheck{Name: "Search tool", Status: "fail", Detail: err.Error()})
		return finishDoctor(app, payload)
	}
	addCheck(render.DoctorCheck{Name: "Search tool", Status: "pass", Detail: searchTool})

	fsTool, err := app.ResolveFSTool(ctx)
	if err != nil {
		addCheck(render.DoctorCheck{Name: "FS tool", Status: "fail", Detail: err.Error()})
		return finishDoctor(app, payload)
	}
	addCheck(render.DoctorCheck{Name: "FS tool", Status: "pass", Detail: fsTool})

	// Check 5: dummy search call
	start := time.Now()
	resp, err := app.Client.CallTool(ctx, searchTool, map[string]any{"query": "overview"})
	if err != nil {
		addCheck(render.DoctorCheck{Name: "Search call", Status: "fail", Detail: err.Error()})
		return finishDoctor(app, payload)
	}
	blocks, err := mcp.ParseSearchBlocks(resp)
	if err != nil {
		addCheck(render.DoctorCheck{Name: "Search call", Status: "fail", Detail: err.Error()})
		return finishDoctor(app, payload)
	}
	addCheck(render.DoctorCheck{Name: "Search call", Status: "pass",
		Detail:     fmt.Sprintf("%d result(s)", len(blocks)),
		DurationMs: time.Since(start).Milliseconds(),
	})

	return finishDoctor(app, payload)
}

func finishDoctor(app *cliapp.App, payload render.DoctorPayload) error {
	if err := app.Renderer.Render(os.Stdout, payload); err != nil {
		return err
	}
	if !payload.OK {
		return cliapp.Newf(cliapp.ExitRuntime, "doctor checks failed")
	}
	return nil
}
