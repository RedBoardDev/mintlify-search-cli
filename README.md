# mintlify-search-cli

`msc` is a CLI for searching Mintlify documentation from the terminal or from AI agents.

## Install

### Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/RedBoardDev/mintlify-search-cli/refs/heads/main/install.sh | bash
```

The installer builds `msc`, installs it to `/usr/local/bin`, and can optionally configure Claude Code, Cursor, or Codex.

### From source

```bash
git clone https://github.com/redboard/mintlify-search-cli.git
cd mintlify-search-cli
make install
```

### Requirements

- Go 1.24+

## Configuration

Resolution order: `flags > env vars > config file`

### Environment variables

```bash
export MSC_API_KEY="mint_dsc_xxxxx"
export MSC_DOMAIN="docs.example.com"
```

### Config file

```bash
msc config set-key <mintlify-api-key>
msc config set-domain <docs-domain>
msc config show
```

Config path: `~/.config/msc/config.json`

### One-shot usage

```bash
msc search "auth" --api-key mint_dsc_xxx --domain docs.example.com
```

## Commands

```bash
msc search "authentication"
msc search "webhooks" --limit 3
msc search "authentication" --json
msc open "getting started"
msc doctor
```

## JSON Output

`msc search --json` returns minified, flat JSON designed for agent consumption.

See `skill.md` for the agent integration file.

## License

MIT. See [LICENSE](LICENSE).
