#!/usr/bin/env bash
set -euo pipefail

BINARY="msc"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
NONINTERACTIVE="${NONINTERACTIVE:-}"
REMOVE_CONFIG="${REMOVE_CONFIG:-}"
REMOVE_CACHE="${REMOVE_CACHE:-}"
REMOVE_SKILLS="${REMOVE_SKILLS:-}"

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

abort() {
    fail "$1"
    exit 1
}

require_bash() {
    if [ -z "${BASH_VERSION:-}" ]; then
        abort "Bash is required to run this uninstaller."
    fi
}

is_noninteractive() {
    [ -n "$NONINTERACTIVE" ]
}

confirm() {
    local prompt="$1"
    local default_value="$2"
    local answer=""

    if is_noninteractive; then
        [ "$default_value" = "yes" ]
        return
    fi

    if [ ! -r /dev/tty ]; then
        abort "Interactive input requires /dev/tty. Re-run with NONINTERACTIVE=1 and explicit REMOVE_* options."
    fi

    ask "$prompt"
    IFS= read -r answer < /dev/tty
    answer="${answer:-$default_value}"
    case "$answer" in
        y|Y|yes|YES)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

remove_path() {
    local path="$1"
    local label="$2"

    if [ ! -e "$path" ]; then
        warn "$label not found: $path"
        return
    fi

    rm -rf "$path"
    ok "Removed $label: $path"
}

require_bash

info "Uninstalling msc"

BINARY_PATH="$INSTALL_DIR/$BINARY"
CLAUDE_SKILL="$HOME/.claude/skills/mintlify-search.md"
CURSOR_SKILL="$HOME/.cursor/rules/mintlify-search.md"
CODEX_SKILL="$HOME/.codex/mintlify-search.md"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/msc"
CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/msc"

if [ "$(uname -s)" = "Darwin" ]; then
    CONFIG_DIR="$HOME/Library/Application Support/msc"
    CACHE_DIR="$HOME/Library/Caches/msc"
fi

if confirm "Remove binary at $BINARY_PATH? [Y/n]:" "yes"; then
    remove_path "$BINARY_PATH" "binary"
else
    warn "Skipped binary removal"
fi

if [ -n "$REMOVE_SKILLS" ]; then
    remove_path "$CLAUDE_SKILL" "Claude Code skill"
    remove_path "$CURSOR_SKILL" "Cursor skill"
    remove_path "$CODEX_SKILL" "Codex skill"
elif confirm "Remove installed agent skills? [y/N]:" "no"; then
    remove_path "$CLAUDE_SKILL" "Claude Code skill"
    remove_path "$CURSOR_SKILL" "Cursor skill"
    remove_path "$CODEX_SKILL" "Codex skill"
else
    warn "Skipped skill removal"
fi

if [ -n "$REMOVE_CONFIG" ]; then
    remove_path "$CONFIG_DIR" "config directory"
elif confirm "Remove config directory $CONFIG_DIR? [y/N]:" "no"; then
    remove_path "$CONFIG_DIR" "config directory"
else
    warn "Skipped config removal"
fi

if [ -n "$REMOVE_CACHE" ]; then
    remove_path "$CACHE_DIR" "cache directory"
elif confirm "Remove cache directory $CACHE_DIR? [y/N]:" "no"; then
    remove_path "$CACHE_DIR" "cache directory"
else
    warn "Skipped cache removal"
fi

echo ""
info "Uninstall complete!"
