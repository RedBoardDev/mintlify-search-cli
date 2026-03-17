---
name: mintlify-search
description: Search Mintlify documentation with `msc`. Prefer compact JSON output for agent workflows.
---

# Mintlify Search CLI (`msc`)

Use `msc` to query Mintlify documentation directly from the terminal.

## Setup

Provide:

```
MSC_API_KEY=mint_dsc_xxxxx
MSC_DOMAIN=docs.example.com
```

Resolution order: `flags > env vars > config file`

## Agent Usage

- Prefer `msc search "<query>" --json --limit 3`
- Start with short, specific queries
- Increase `--limit` only when needed
- Use `msc open "<query>"` only if the user explicitly wants a browser opened
- Run `msc doctor` if config or connectivity looks wrong
- Use `msc --help` or `msc <command> --help` when unsure

`--json` returns minified, flat JSON optimized for LLM consumption.

## Commands

```bash
msc search "authentication"
msc search "webhooks" --json --limit 3
msc open "getting started"
msc doctor
msc config set-key <api-key>
msc config set-domain <domain>
msc config show
```

For one-shot calls, `--api-key` and `--domain` override stored config.
