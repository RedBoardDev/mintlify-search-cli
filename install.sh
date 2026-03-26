set -euo pipefail

REPO="redboard/mintlify-search-cli"
BINARY="msc"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_SOURCE="$SCRIPT_DIR/skill.md"

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

render_skill_file() {
    local output="$1"

    if [ ! -f "$SKILL_SOURCE" ]; then
        return 1
    fi

    if [ -n "${MCP_URL:-}" ]; then
        sed "s|__MSC_MCP_URL__|$MCP_URL|g" "$SKILL_SOURCE" > "$output"
    else
        cp "$SKILL_SOURCE" "$output"
    fi
}

info "Installing msc"

if ! command -v go &> /dev/null; then
    fail "Go is not installed. Install Go 1.24+ first."
    exit 1
fi

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

info "Configuration"

CURRENT_MCP_URL="${MSC_MCP_URL:-}"
if [ -z "$CURRENT_MCP_URL" ]; then
    ask "Mintlify MCP URL (https://<docs>/mcp or /authed/mcp):"
    read -r MCP_URL
    if [ -z "$MCP_URL" ]; then
        warn "Skipped — set later with: msc config set-mcp-url <url>"
    else
        "$INSTALL_DIR/$BINARY" config set-mcp-url "$MCP_URL" > /dev/null
        ok "MCP URL saved to config file"
    fi
else
    MCP_URL="$CURRENT_MCP_URL"
    ok "MCP URL detected from MSC_MCP_URL env var"
    "$INSTALL_DIR/$BINARY" config set-mcp-url "$MCP_URL" > /dev/null
    ok "MCP URL saved to config file"
fi

echo ""

info "Agent integrations"
echo -e "  ${DIM}Install msc as a skill in your AI coding agents.${RESET}"
echo ""

setup_claude_code() {
    local settings_dir="$HOME/.claude"
    local skills_dir="$settings_dir/skills"
    local rendered_skill

    mkdir -p "$settings_dir" "$skills_dir"

    if [ -f "$SKILL_SOURCE" ]; then
        rendered_skill="$(mktemp)"
        render_skill_file "$rendered_skill"
        cp "$rendered_skill" "$skills_dir/mintlify-search.md"
        rm -f "$rendered_skill"
        ok "Skill file installed to $skills_dir/mintlify-search.md"
    fi
}

setup_cursor() {
    local cursor_dir="$HOME/.cursor"
    local rules_dir="$cursor_dir/rules"
    local rendered_skill

    mkdir -p "$rules_dir"

    if [ -f "$SKILL_SOURCE" ]; then
        rendered_skill="$(mktemp)"
        render_skill_file "$rendered_skill"
        cp "$rendered_skill" "$rules_dir/mintlify-search.md"
        rm -f "$rendered_skill"
        ok "Skill file installed to $rules_dir/mintlify-search.md"
    fi
}

setup_codex() {
    local codex_dir="$HOME/.codex"
    local rendered_skill

    mkdir -p "$codex_dir"

    if [ -f "$SKILL_SOURCE" ]; then
        rendered_skill="$(mktemp)"
        render_skill_file "$rendered_skill"
        cp "$rendered_skill" "$codex_dir/mintlify-search.md"
        rm -f "$rendered_skill"
        ok "Skill file installed to $codex_dir/mintlify-search.md"
    fi
}

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

info "Verification"
echo ""
if [ -n "${MCP_URL:-}" ]; then
    "$INSTALL_DIR/$BINARY" doctor --mcp-url "$MCP_URL"
else
    warn "Skipped verification — MCP URL not configured"
fi
echo ""

info "Setup complete!"
echo ""
echo -e "  ${DIM}Quick start:${RESET}"
echo "    msc search \"getting started\""
echo "    msc search \"authentication\" --json"
echo "    msc search \"authentication\" --raw"
echo "    msc open \"quickstart\""
echo ""
