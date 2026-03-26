---
name: mintlify-search
description: Search Mintlify documentation with `msc` through a hosted MCP endpoint. Prefer normalized JSON for agent workflows.
---

# Mintlify Search CLI (`msc`)

Use `msc` to query Mintlify documentation through a hosted MCP endpoint.

## Setup

Use this MCP endpoint:

```bash
__MSC_MCP_URL__
```

If the CLI is not configured yet, use `--mcp-url` directly in commands.

## Agent Usage

- Prefer `msc search "<query>" --mcp-url __MSC_MCP_URL__ --json`
- Use short, specific queries first
- Use `msc search "<query>" --mcp-url __MSC_MCP_URL__ --raw` only if raw MCP output is explicitly needed
- Use `msc open "<query>" --mcp-url __MSC_MCP_URL__` only if the user explicitly wants a browser opened
- Run `msc doctor --mcp-url __MSC_MCP_URL__` if config or connectivity looks wrong
- Use `msc --help` or `msc <command> --help` when unsure

`--json` returns normalized minified JSON optimized for LLM consumption.

## Commands

```bash
msc search "authentication" --mcp-url __MSC_MCP_URL__ --json
msc search "authentication" --mcp-url __MSC_MCP_URL__ --raw
msc search "authentication" --mcp-url __MSC_MCP_URL__ --limit 3
msc open "getting started" --mcp-url __MSC_MCP_URL__
msc doctor --mcp-url __MSC_MCP_URL__
msc config set-mcp-url __MSC_MCP_URL__
msc config show
```
