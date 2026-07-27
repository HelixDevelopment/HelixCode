#!/usr/bin/env bash
# setup.sh — one-time dependency check, config bootstrap, and systemd unit
# installer for HelixCode and its infra stack.
#
# What it does:
#   1. Check required tooling (podman, go, git)
#   2. Clone/sync submodules
#   3. Build HelixCode binary (if not already installed)
#   4. Bootstrap config from template if missing
#   5. Install systemd user units for HelixCode server + infra
#   6. Install systemd user unit for HelixLLM coder (if GPU present)
#
# Idempotent: safe to re-run. Each step skips if already satisfied.
#
# Usage:
#   bash scripts/setup.sh              # full setup
#   bash scripts/setup.sh --check-only # dry-run, report missing deps
#   bash scripts/setup.sh --no-systemd # skip systemd unit install

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SYSTEMD_USER_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { printf '%b[INFO]%b  %s\n' "$GREEN" "$NC" "$*"; }
warn()  { printf '%b[WARN]%b  %s\n' "$YELLOW" "$NC" "$*" >&2; }
error() { printf '%b[ERROR]%b %s\n' "$RED" "$NC" "$*" >&2; }
header(){ printf '\n%b== %s ==%b\n' "$GREEN" "$*" "$NC"; }

CHECK_ONLY=0
SKIP_SYSTEMD=0
for arg in "$@"; do
  case "$arg" in
    --check-only) CHECK_ONLY=1 ;;
    --no-systemd) SKIP_SYSTEMD=1 ;;
  esac
done

# ─── 1. Tooling check ───────────────────────────────────────────────

header "1/5 Checking required tooling"

missing=0
for tool in podman go git curl nvidia-smi; do
  if command -v "$tool" >/dev/null 2>&1; then
    info "  $tool: found ($(command -v "$tool"))"
  else
    [ "$tool" = "nvidia-smi" ] && continue  # optional — no GPU is OK
    if [ "$tool" = "podman" ] && command -v docker >/dev/null 2>&1; then
      warn "  podman not found, but docker is available (suboptimal; §11.4.161 prefers rootless podman)"
    else
      error "  $tool: MISSING"
      missing=$((missing + 1))
    fi
  fi
done

if [ "$missing" -gt 0 ]; then
  error "$missing required tools are missing. Install them first."
  [ "$CHECK_ONLY" -eq 0 ] && exit 1
fi

if [ "$CHECK_ONLY" -eq 1 ]; then
  info "Check complete (--check-only)."
  exit "$missing"
fi

# ─── 2. Submodules ──────────────────────────────────────────────────

header "2/5 Syncing submodules"
git submodule update --init --recursive
info "submodules up to date"

# ─── 3. Build HelixCode binary ───────────────────────────────────────

header "3/5 Building HelixCode"

HELIXCODE_BIN="$REPO_ROOT/helix_code/bin/helixcode"
if [ -x "$HELIXCODE_BIN" ]; then
  info "HelixCode binary already exists at $HELIXCODE_BIN"
else
  info "Building HelixCode ..."
  ( cd "$REPO_ROOT/helix_code" && go build -ldflags="-s -w" -o bin/helixcode ./cmd/server )
  info "HelixCode built: $HELIXCODE_BIN"
fi

# ─── 4. Bootstrap config ─────────────────────────────────────────────

header "4/5 Bootstrapping config"

CONFIG_DIR="$HOME/.config/helixcode"
mkdir -p "$CONFIG_DIR"

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
  if [ -f "$REPO_ROOT/helix_code/config/replica-8081.yaml" ]; then
    cp "$REPO_ROOT/helix_code/config/replica-8081.yaml" "$CONFIG_DIR/config.yaml"
    info "Config copied: $CONFIG_DIR/config.yaml (from replica-8081)"
  elif [ -x "$HELIXCODE_BIN" ]; then
    "$HELIXCODE_BIN" init-config > "$CONFIG_DIR/config.yaml" 2>/dev/null || true
    info "Config initialised via 'helixcode init-config'"
  else
    warn "No config template found — create $CONFIG_DIR/config.yaml manually"
  fi
else
  info "Config already exists: $CONFIG_DIR/config.yaml"
fi

# ─── 5. Systemd units ────────────────────────────────────────────────

if [ "$SKIP_SYSTEMD" -eq 1 ]; then
  info "Skipping systemd unit install (--no-systemd)"
  exit 0
fi

header "5/5 Installing systemd user units"

mkdir -p "$SYSTEMD_USER_DIR"

install_unit() {
  local src="$1"
  local name="$2"
  local dst="$SYSTEMD_USER_DIR/$name"
  if [ -f "$src" ]; then
    cp "$src" "$dst"
    info "  $name installed"
  else
    warn "  source missing: $src — unit $name NOT installed"
  fi
}

enable_service() {
  local name="$1"
  systemctl --user enable "$name" 2>/dev/null || \
    warn "  could not enable $name (is systemd --user running?)"
}

install_unit "$REPO_ROOT/scripts/systemd/helixcode-server.service" \
  helixcode-server.service
install_unit "$REPO_ROOT/scripts/systemd/helixllm-coder.service" \
  helixllm-coder.service

systemctl --user daemon-reload 2>/dev/null || true

info "Enabling services ..."
enable_service helixcode-server.service
enable_service helixllm-coder.service

info ""
info "HelixCode setup complete."
info "Start services now:  systemctl --user start helixllm-coder helixcode-server"
info "Check status:        systemctl --user status helixcode-server"
