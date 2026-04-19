---
name: mintlify-search
description: Search and read Mintlify API documentation with `msc`. Prefer `msc search` + `msc open` / `msc read` over direct MCP calls.
---

# Mintlify Search CLI (`msc`)

Use `msc` to query Mintlify documentation through a hosted MCP endpoint. JSON-first output for structured commands, raw markdown for content commands — both are optimized for LLM token consumption.

## Setup

Use this MCP endpoint:

```
__MSC_MCP_URL__
```

If the CLI is not configured yet, pass `--mcp-url __MSC_MCP_URL__` on every command.

## Core loop (95% of use cases)

```bash
# 1. Find relevant pages
msc search "<query>" --mcp-url __MSC_MCP_URL__ --limit 5

# 2a. Read the top hit directly
msc open "<query>" --mcp-url __MSC_MCP_URL__

# 2b. Or read a specific page by the "page" field returned by search
msc read "<page>" --mcp-url __MSC_MCP_URL__
```

Example chain:

```bash
msc search "list users" --mcp-url __MSC_MCP_URL__
# -> JSON with results[0].page = "User/v1/api-rs/list-users"
msc read "User/v1/api-rs/list-users" --mcp-url __MSC_MCP_URL__
# -> full markdown
```

## Escape hatch (rare)

```bash
msc fs "tree /Rule -L 3" --mcp-url __MSC_MCP_URL__
msc fs "rg -il 'x-rate-limit' /" --mcp-url __MSC_MCP_URL__
msc fs "cat /openapi/global/openapi.yaml | jq '.paths | keys'" --mcp-url __MSC_MCP_URL__
```

## Guidelines

- `search` default output is **minified JSON** — do not pass `--text` unless a human is reading.
- `open` outputs raw markdown on stdout; pipe/capture directly.
- `read` resolves `.mdx` automatically — pass the `page` field from search as-is.
- Exit codes: 0 = ok (even 0 results), 1 = runtime, 2 = usage (e.g. empty query), 3 = config.
- Do not call the MCP endpoint directly; go through `msc` so caching and tools discovery are shared.
- Use `msc doctor --mcp-url __MSC_MCP_URL__ --json` if something seems off.

## Commands reference

```bash
msc search "<query>" [--limit N] [--json|--text|--raw]
msc open   "<query>" [--lines N] [--json|--raw]
msc read   "<path>"  [--lines N | --from L --to M] [--json|--raw]
msc fs     "<command>" [--json|--raw]
msc doctor [--json]
msc config [list|get <key>|set <key> <value>] [--json]
```
