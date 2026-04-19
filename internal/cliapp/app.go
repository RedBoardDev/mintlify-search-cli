package cliapp

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/redboard/mintlify-search-cli/internal/config"
	"github.com/redboard/mintlify-search-cli/internal/mcp"
	"github.com/redboard/mintlify-search-cli/internal/render"
)

// App is the per-invocation execution context. It is built once per command
// by FromCmd and passed down to handlers. All dependencies are explicit —
// no package-level globals.
type App struct {
	Cfg      *config.Config
	Client   mcp.MCPClient
	Renderer render.Renderer
	CacheDir string
	NoCache  bool
	Verbose  bool
	Timeout  time.Duration
	Format   render.FormatKind
}

// Options overrides individual defaults from FromCmd. Primarily used by tests
// to inject a FakeClient.
type Options struct {
	Client mcp.MCPClient
}

// FromCmd builds an App from the persistent flags attached to the root
// command. It loads config, applies --mcp-url overrides, validates, and
// constructs the HTTP MCP client.
func FromCmd(cmd *cobra.Command, format render.FormatKind, opts ...Options) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, Wrap(ExitConfig, err)
	}
	if v, _ := cmd.Flags().GetString("mcp-url"); v != "" {
		cfg.MCPURL = v
	}
	if err := cfg.Validate(); err != nil {
		return nil, Wrap(ExitConfig, err)
	}

	cacheDir, err := config.CacheDir()
	if err != nil {
		return nil, Wrap(ExitConfig, fmt.Errorf("resolving cache dir: %w", err))
	}

	noCache, _ := cmd.Flags().GetBool("no-cache")
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeoutSec, _ := cmd.Flags().GetInt("timeout")
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	timeout := time.Duration(timeoutSec) * time.Second

	var client mcp.MCPClient
	if len(opts) > 0 && opts[0].Client != nil {
		client = opts[0].Client
	} else {
		var clientOpts []mcp.Option
		if verbose {
			clientOpts = append(clientOpts, mcp.WithLogger(func(format string, args ...any) {
				fmt.Fprintf(os.Stderr, "msc: "+format+"\n", args...)
			}))
		}
		client = mcp.NewClient(cfg.MCPURL, clientOpts...)
	}

	return &App{
		Cfg:      cfg,
		Client:   client,
		Renderer: render.New(format),
		CacheDir: cacheDir,
		NoCache:  noCache,
		Verbose:  verbose,
		Timeout:  timeout,
		Format:   format,
	}, nil
}
