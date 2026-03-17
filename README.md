# mintlify-search-cli

**mintlify-search-cli** (`msc`) is a high-performance, deterministic retrieval engine built for developers and AI agents (Claude, Cursor, Codex). It bypasses the overhead of Model Context Protocol (MCP) and LLM-based RAG by querying the official Mintlify Discovery API directly.

> **Why this?** Most documentation tools are slow or return too much noise, MCP uses too many tokens. This CLI is optimized for **speed**, **token-efficiency**, and **machine-readability**.

---

## Features

- **Zero-Latency:** Direct API calls with 3s timeout.
- **Agent-Optimized:** Flat JSON output designed to minimize token usage in LLM context windows.
- **Deterministic:** No generative AI hallucinations — only indexed documentation.
- **Built-in Diagnostics:** The `doctor` command validates your connectivity, auth, and latency.
- **Local Cache:** Intelligent TTL-based caching (5min, SHA256 keys) to avoid redundant network overhead.

---

## Installation

### From source

```bash
git clone https://github.com/redboard/mintlify-search-cli.git
cd mintlify-search-cli
make install
```

### One-liner

```bash
curl -sSL https://raw.githubusercontent.com/redboard/mintlify-search-cli/main/install.sh | bash
```

**Requirements:** Go 1.24+

---

## Configuration

Configuration is resolved with precedence: **flags > env vars > config file**.

### Environment variables (recommended for agents)

```bash
export MSC_API_KEY="mint_dsc_xxxxx"
export MSC_DOMAIN="docs.example.com"
```

For Claude Code, add to `.claude/settings.local.json`:

```json
{
  "env": {
    "MSC_API_KEY": "mint_dsc_xxxxx",
    "MSC_DOMAIN": "docs.example.com"
  }
}
```

### Config file (recommended for humans)

```bash
msc config set-key <your-mintlify-api-key>
msc config set-domain <your-docs-domain>
msc config show
```

Stored in `~/.config/msc/config.json`.

### One-shot flags

```bash
msc search "auth" --api-key mint_dsc_xxx --domain docs.example.com
```

---

## Usage

### Search documentation

```bash
# Text output (default)
msc search "authentication"

# Limit results
msc search "webhooks" --limit 3

# JSON output (minified, token-optimized)
msc search "authentication" --json
```

### Open in browser

```bash
msc open "getting started"
```

### Diagnostics

```bash
msc doctor
```

---

## Agent Integration (Claude/Cursor)

**Output Philosophy:**
The JSON output is **minified and flat**. We strip unnecessary nesting to ensure the agent receives the maximum amount of information within its context window without wasting tokens.

```bash
# Example for agent consumption
msc search "api endpoints" --json --limit 3
```

See `skill.md` for agent skill definition.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
