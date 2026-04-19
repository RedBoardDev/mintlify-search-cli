# mintlify-search-cli

`msc` is a CLI client for Mintlify-hosted Model Context Protocol (MCP) servers that expose company API documentation. It is designed for **LLM agents** (Claude Code, Cursor, and similar) that need to search and read the docs during a development workflow.

## What it does

- `msc search <query>` — semantic search, returns structured JSON
- `msc open <query>` — search + full markdown of the top result, in one call
- `msc read <path>` — fetch a specific page by path
- `msc fs <command>` — escape hatch: run `rg`, `tree`, `cat`, `jq`, etc. against the docs sandbox
- `msc doctor` — diagnose configuration and MCP connectivity
- `msc config` / `msc completion` — setup plumbing

## Install

### Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/redboarddev/mintlify-search-cli/main/install.sh | bash
```

The installer builds `msc`, installs it to `/usr/local/bin`, asks for your MCP URL, and can install the agent skill for Claude Code / Cursor / Codex.

Non-interactive:

```bash
MSC_MCP_URL="https://docs.example.com/mcp" NONINTERACTIVE=1 \
  curl -fsSL https://raw.githubusercontent.com/redboarddev/mintlify-search-cli/main/install.sh | bash
```

### From source

```bash
git clone https://github.com/redboarddev/mintlify-search-cli.git
cd mintlify-search-cli
make install
```

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/redboarddev/mintlify-search-cli/main/uninstall.sh | bash
```

Requirements: Go 1.24+ to build.

## Configuration

Resolution order: **flags > env vars > config file > defaults**.

```bash
msc config set mcp_url https://docs.example.com/mcp
msc config set default_limit 5
msc config list
```

Supported keys: `mcp_url`, `default_limit`, `cache.enabled`, `cache.ttl_seconds`, `cache.tools_ttl_seconds`.

Environment variables: `MSC_MCP_URL`, `MSC_DEFAULT_LIMIT`, `MSC_CACHE_ENABLED`, `MSC_CACHE_TTL_SECONDS`, `MSC_TOOLS_CACHE_TTL_SECONDS`.

Config path is `~/.config/msc/config.json` on Linux and `~/Library/Application Support/msc/config.json` on macOS.

## Commands

### `search`

```bash
msc search "authentication"              # minified JSON (default)
msc search "authentication" --limit 3
msc search "rate limiting" --text        # human-readable
msc search "..." --raw                   # raw MCP payload
```

JSON schema:

```json
{
  "query": "authentication",
  "results": [
    {
      "title": "Authenticate and log in a backoffice user.",
      "url": "https://docs.example.com/Auth/v1/login",
      "page": "Auth/v1/login",
      "content": "..."
    }
  ]
}
```

`page` is the path consumable by `msc read` and `msc fs "cat /<page>.mdx"`.

### `open`

```bash
msc open "list rules"                # prints full markdown of top hit
msc open "rate limit" --lines 80     # first 80 lines only
msc open "..." --json                # {title, url, page, resolved_path, content, truncated}
```

**Breaking change vs v1**: `open` no longer launches a browser. It now prints the markdown on stdout so agents can consume it.

### `read`

```bash
msc read /Rule/v2/api-rs/read/list-rules-for-the-given-site-ids
msc read Rule/v2/api-rs/read/list-rules-for-the-given-site-ids      # .mdx auto-resolved
msc read /overview --lines 50
msc read /openapi/global/openapi.yaml --from 1 --to 80
```

Extension resolution tries `.mdx`, `.md`, `/index.mdx`, then the bare path.

### `fs`

```bash
msc fs "tree / -L 2"
msc fs "rg -il 'rate limit' /"
msc fs "cat /openapi/global/openapi.yaml | jq '.paths | keys | length'"
```

Quote the command as a single argument. Supported binaries: `rg`, `grep`, `find`, `tree`, `ls`, `cat`, `head`, `tail`, `stat`, `wc`, `sort`, `uniq`, `cut`, `sed`, `awk`, `jq`. Output is truncated at ~30 KB per call.

### `doctor`

```bash
msc doctor
msc doctor --json
```

Checks: config loaded, URL valid, `initialize` succeeds, search+fs tools advertised, a dummy search call returns content.

## Output formats

| Command | Default | `--json` | `--text` | `--raw` |
|---|---|---|---|---|
| `search` | JSON (token-lean) | same | numbered listing | raw MCP payload |
| `fs` | raw stdout | wrap metadata | — | raw MCP payload |
| `read` | markdown | wrap metadata | — | raw MCP payload |
| `open` | markdown | wrap metadata | — | both MCP payloads |
| `doctor` | human | array of checks | — | — |
| `config list` | human | JSON dump | — | — |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success (even 0 results) |
| 1 | runtime error: MCP unreachable, tool call failed, page not found |
| 2 | usage error: bad flags, missing args, empty query |
| 3 | config error: file unreadable, URL invalid |

## Global flags

- `--mcp-url <url>` — override endpoint
- `--timeout <sec>` — HTTP timeout (default 30)
- `--no-cache` — bypass tools-discovery cache
- `--verbose` / `-v` — trace HTTP on stderr

## Testing

```bash
go test -race ./...                                        # unit
go test -tags=integration ./integration/...                # E2E against prod MCP
```

See [skill.md](skill.md) for the agent skill template rendered by the installer.

## License

MIT. See [LICENSE](LICENSE).
