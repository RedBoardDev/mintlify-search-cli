---
name: mintlify-search
description: Search Mintlify documentation from the CLI. Fast, deterministic retrieval — no LLM, no MCP.
---

# Mintlify Search CLI (msc)

## Prerequisites

Set these environment variables (e.g. in `.claude/settings.local.json` under `"env"`):

```
MSC_API_KEY=mint_dsc_xxxxx
MSC_DOMAIN=docs.example.com
```

## Commands

```bash
# Search documentation (default: 5 results, text format)
msc search "authentication"

# JSON output (minified, token-optimized for agents)
msc search "webhooks" --json --limit 3

# Open top result in browser
msc open "getting started"

# Check configuration and connectivity
msc doctor

# Manual config (alternative to env vars)
msc config set-key <api-key>
msc config set-domain <domain>
msc config show
```

## Output

- **Text mode**: Compact numbered list with title, URL, and snippet.
- **JSON mode** (`--json`): Minified flat array — minimal tokens, maximum signal.

## Tips

- Results are cached for 5 minutes (file-based, SHA256 keys).
- Use `--json` when feeding results into another tool or LLM context.
- Use `--limit 1` when you only need the most relevant hit.
- Flags `--api-key` and `--domain` override everything for one-shot usage.
