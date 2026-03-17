#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────
# msc — Mintlify Search CLI installer & setup wizard
# ─────────────────────────────────────────────────────────

REPO="redboard/mintlify-search-cli"
BINARY="msc"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_SOURCE="$SCRIPT_DIR/skill.md"

# Colors (disabled if not a terminal).
if [ -t 1 ]; then
    BOLD='\033[1m' DIM='\033[2m' GREEN='\033[32m' YELLOW='\033[33m'
    CYAN='\033[36m' RED='\033[31m' RESET='\033[0m'
else
    BOLD='' DIM='' GREEN='' YELLOW='' CYAN='' RED='' RESET=''
fi

info()  { echo -e "${CYAN}==> ${RESET}${BOLD}$1${RESET}"; }
ok()    { echo -e "  ${GREEN}✓${RESET} $1"; }
warn()  { echo -e "  ${YELLOW}!${RESET} $1"; }
fail()  { echo -e "  ${RED}✗${RESET} $1"; }
ask()   { echo -en "  ${BOLD}$1${RESET} "; }

# ─────────────────────────────────────────────────────────
# Step 1: Build & install binary
# ─────────────────────────────────────────────────────────
info "Installing msc"

if ! command -v go &> /dev/null; then
    fail "Go is not installed. Install Go 1.24+ first."
    exit 1
fi

# Build from local source if available, otherwise clone.
if [ -f "$SCRIPT_DIR/go.mod" ]; then
    BUILD_DIR="$SCRIPT_DIR"
else
    BUILD_DIR=$(mktemp -d)
    trap 'rm -rf "$BUILD_DIR"' EXIT
    echo "  Cloning repository..."
    git clone --depth 1 "https://github.com/$REPO.git" "$BUILD_DIR"
fi

cd "$BUILD_DIR"
VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
go build -ldflags "-s -w -X github.com/$REPO/internal/cli.Version=$VERSION" -o "$BINARY" ./cmd/msc

install -m 755 "$BINARY" "$INSTALL_DIR/$BINARY" 2>/dev/null || {
    warn "Cannot write to $INSTALL_DIR, trying with sudo..."
    sudo install -m 755 "$BINARY" "$INSTALL_DIR/$BINARY"
}

ok "Installed $BINARY $VERSION to $INSTALL_DIR/$BINARY"
echo ""

# ─────────────────────────────────────────────────────────
# Step 2: Configuration wizard
# ─────────────────────────────────────────────────────────
info "Configuration"

# API Key
CURRENT_KEY="${MSC_API_KEY:-}"
if [ -z "$CURRENT_KEY" ]; then
    ask "Mintlify API key (mint_dsc_...):"
    read -r API_KEY
    if [ -z "$API_KEY" ]; then
        warn "Skipped — set later with: msc config set-key <key>"
    else
        "$INSTALL_DIR/$BINARY" config set-key "$API_KEY" > /dev/null
        ok "API key saved to config file"
    fi
else
    ok "API key detected from MSC_API_KEY env var"
    API_KEY="$CURRENT_KEY"
fi

# Domain
CURRENT_DOMAIN="${MSC_DOMAIN:-}"
if [ -z "$CURRENT_DOMAIN" ]; then
    ask "Documentation domain (e.g. docs.example.com):"
    read -r DOMAIN
    if [ -z "$DOMAIN" ]; then
        warn "Skipped — set later with: msc config set-domain <domain>"
    else
        "$INSTALL_DIR/$BINARY" config set-domain "$DOMAIN" > /dev/null
        ok "Domain saved to config file"
    fi
else
    ok "Domain detected from MSC_DOMAIN env var"
    DOMAIN="$CURRENT_DOMAIN"
fi

echo ""

# ─────────────────────────────────────────────────────────
# Step 3: Agent integrations
# ─────────────────────────────────────────────────────────
info "Agent integrations"
echo -e "  ${DIM}Install msc as a skill in your AI coding agents.${RESET}"
echo ""

API_KEY="${API_KEY:-}"
DOMAIN="${DOMAIN:-}"

# ── Claude Code ──────────────────────────────────────────
setup_claude_code() {
    local settings_dir="$HOME/.claude"
    local settings_local="$settings_dir/settings.local.json"
    local skills_dir="$settings_dir/skills"

    mkdir -p "$settings_dir" "$skills_dir"

    # Install skill file.
    if [ -f "$SKILL_SOURCE" ]; then
        cp "$SKILL_SOURCE" "$skills_dir/mintlify-search.md"
        ok "Skill file installed to $skills_dir/mintlify-search.md"
    fi

    # Inject env vars into settings.local.json.
    if [ -n "$API_KEY" ] && [ -n "$DOMAIN" ]; then
        if [ -f "$settings_local" ]; then
            # Merge env into existing file using a temporary approach.
            local tmp
            tmp=$(mktemp)
            # Use python/node to merge JSON if available, otherwise create fresh.
            if command -v python3 &> /dev/null; then
                python3 -c "
import json, sys
with open('$settings_local') as f:
    data = json.load(f)
env = data.setdefault('env', {})
env['MSC_API_KEY'] = '$API_KEY'
env['MSC_DOMAIN'] = '$DOMAIN'
with open('$tmp', 'w') as f:
    json.dump(data, f, indent=2)
" 2>/dev/null && mv "$tmp" "$settings_local" || {
                    rm -f "$tmp"
                    warn "Could not merge into $settings_local — add env vars manually"
                    return
                }
            else
                rm -f "$tmp"
                warn "python3 not found — add MSC_API_KEY and MSC_DOMAIN to $settings_local manually"
                return
            fi
        else
            cat > "$settings_local" << JSONEOF
{
  "env": {
    "MSC_API_KEY": "$API_KEY",
    "MSC_DOMAIN": "$DOMAIN"
  }
}
JSONEOF
        fi
        ok "Env vars added to $settings_local"
    fi
}

# ── Cursor ───────────────────────────────────────────────
setup_cursor() {
    local cursor_dir="$HOME/.cursor"
    local rules_dir="$cursor_dir/rules"

    mkdir -p "$rules_dir"

    if [ -f "$SKILL_SOURCE" ]; then
        cp "$SKILL_SOURCE" "$rules_dir/mintlify-search.md"
        ok "Skill file installed to $rules_dir/mintlify-search.md"
    fi

    if [ -n "$API_KEY" ] && [ -n "$DOMAIN" ]; then
        echo ""
        echo -e "  ${DIM}Add to your shell profile or .envrc:${RESET}"
        echo -e "    ${BOLD}export MSC_API_KEY=\"$API_KEY\"${RESET}"
        echo -e "    ${BOLD}export MSC_DOMAIN=\"$DOMAIN\"${RESET}"
    fi
}

# ── Codex ────────────────────────────────────────────────
setup_codex() {
    local codex_dir="$HOME/.codex"

    mkdir -p "$codex_dir"

    if [ -f "$SKILL_SOURCE" ]; then
        cp "$SKILL_SOURCE" "$codex_dir/mintlify-search.md"
        ok "Skill file installed to $codex_dir/mintlify-search.md"
    fi

    if [ -n "$API_KEY" ] && [ -n "$DOMAIN" ]; then
        echo ""
        echo -e "  ${DIM}Add to your shell profile or .envrc:${RESET}"
        echo -e "    ${BOLD}export MSC_API_KEY=\"$API_KEY\"${RESET}"
        echo -e "    ${BOLD}export MSC_DOMAIN=\"$DOMAIN\"${RESET}"
    fi
}

# ── Selection menu ───────────────────────────────────────
echo "  Which agents do you want to configure?"
echo ""
echo "    1) Claude Code"
echo "    2) Cursor"
echo "    3) Codex"
echo "    4) All"
echo "    5) None (skip)"
echo ""
ask "Choose [1-5]:"
read -r AGENT_CHOICE

echo ""

case "$AGENT_CHOICE" in
    1)
        setup_claude_code
        ;;
    2)
        setup_cursor
        ;;
    3)
        setup_codex
        ;;
    4)
        echo -e "  ${DIM}── Claude Code ──${RESET}"
        setup_claude_code
        echo ""
        echo -e "  ${DIM}── Cursor ──${RESET}"
        setup_cursor
        echo ""
        echo -e "  ${DIM}── Codex ──${RESET}"
        setup_codex
        ;;
    5|"")
        ok "Skipped agent setup"
        ;;
    *)
        warn "Unknown choice, skipping"
        ;;
esac

echo ""

# ─────────────────────────────────────────────────────────
# Step 4: Verification
# ─────────────────────────────────────────────────────────
info "Verification"
echo ""
"$INSTALL_DIR/$BINARY" doctor
echo ""

info "Setup complete!"
echo ""
echo -e "  ${DIM}Quick start:${RESET}"
echo "    msc search \"getting started\""
echo "    msc search \"authentication\" --json --limit 3"
echo "    msc open \"quickstart\""
echo ""
