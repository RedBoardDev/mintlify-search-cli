# mintlify-search-cli

`msc` is a CLI for searching Mintlify documentation through hosted MCP endpoints.

## Install

### Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/redboarddev/mintlify-search-cli/main/install.sh | bash
```

The installer builds `msc`, installs it to `/usr/local/bin`, asks for your MCP URL, and can install the agent skill for Claude Code, Cursor, or Codex.

### From source

```bash
git clone https://github.com/redboarddev/mintlify-search-cli.git
cd mintlify-search-cli
make install
```

### Requirements

- Go 1.24+

## Configuration

Resolution order: `flags > env vars > config file`

### MCP URL

Use a full Mintlify MCP endpoint:

```bash
https://docs.example.com/mcp
https://docs.example.com/authed/mcp
```

### Environment variable

```bash
export MSC_MCP_URL="https://docs.example.com/mcp"
```

### Config file

```bash
msc config set-mcp-url https://docs.example.com/mcp
msc config show
```

Config path depends on your OS. Use `msc config show` to print the active file path.

### One-shot usage

```bash
msc search "auth" --mcp-url https://docs.example.com/mcp
```

## Commands

```bash
msc search "authentication"
msc search "authentication" --json
msc search "authentication" --raw
msc search "authentication" --limit 3
msc open "getting started"
msc doctor
```

## Output

- Default output: compact numbered text results
- `--json`: normalized minified JSON for agents
- `--raw`: raw MCP JSON-RPC payload

See [skill.md](skill.md) for the agent integration template and [MCP_ENDPOINT_RESEARCH.md](MCP_ENDPOINT_RESEARCH.md) for protocol notes.

## License

MIT. See [LICENSE](LICENSE).
