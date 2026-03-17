# 🍃 mintlify-search-cli

**mintlify-search-cli** is a high-performance, deterministic retrieval engine built for developers and AI agents (Claude, Cursor, Codex). It bypasses the overhead of Model Context Protocol (MCP) and LLM-based RAG by querying the official Mintlify Discovery API directly.

> **Why this?** Most documentation tools are slow or return too much noise, MCP are using too much tokens. This CLI is optimized for **speed**, **token-efficiency**, and **machine-readability**.

-----

## Features

  * **Zero-Latency:** Direct API calls.
  * **Agent-Optimized:** Flat JSON output designed to minimize token usage in LLM context windows.
  * **Deterministic:** No generative AI "hallucinations"—only indexed documentation.
  * **Built-in Diagnostics:** The `doctor` command validates your connectivity, auth, and latency.
  * **Local Cache:** Intelligent TTL-based caching to avoid redundant network overhead.

-----

## Installation

-----

## Configuration

-----

## Usage

-----

## Agent Integration (Claude/Cursor)

**Output Philosophy:**
The JSON output is **minified and flat**. We strip unnecessary nesting to ensure the agent receives the maximum amount of information within its context window without wasting tokens.

-----

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
