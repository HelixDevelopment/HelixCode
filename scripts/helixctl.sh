#!/usr/bin/env bash
#
# helixctl.sh — Operator control wrapper for the Helix platform systemd user units.
#
# Ties every lifecycle action to systemctl --user, so the whole platform is
# controlled through a single, documented command.
#
# Usage:
#   scripts/helixctl.sh install          # build + install systemd units + secrets
#   scripts/helixctl.sh start            # systemctl --user start helix.target
#   scripts/helixctl.sh stop             # systemctl --user stop helix.target
#   scripts/helixctl.sh restart          # systemctl --user restart helix.target
#   scripts/helixctl.sh status           # list all helix units + states
#   scripts/helixctl.sh status <unit>    # detailed status for one unit
#   scripts/helixctl.sh enable           # enable all units + linger
#   scripts/helixctl.sh disable          # disable all units
#   scripts/helixctl.sh validate         # run the platform validation script
#   scripts/helixctl.sh logs <unit>      # follow journal for a unit
#   scripts/helixctl.sh help             # this message
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UNIT_NAMES=(
  helix.target
  helixcode-infra.service
  llmsverifier.service
  helixllm-coder.service
  helixllm-gateway.service
  helixagent.service
  helixcode-server.service
)

log()  { printf '%s\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*"; }
die()  { printf '  \033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Helix platform control (systemd --user)

  install          Run setup.sh (build binaries, generate secrets, install units)
  start            Start the whole platform: systemctl --user start helix.target
  stop             Stop the whole platform: systemctl --user stop helix.target
  restart          Restart the whole platform
  status           List all helix units
  status <unit>    Show detailed status for <unit>
  enable           Enable all units and ensure linger (boot-persistent)
  disable          Disable all units
  validate         Run scripts/validate-helix-platform.sh
  logs <unit>      Follow journal for <unit>
  help             Show this help

Examples:
  scripts/helixctl.sh status
  scripts/helixctl.sh logs helixcode-server
  scripts/helixctl.sh validate
EOF
}

cmd_install() {
  cd "${REPO_ROOT}"
  log "Running setup.sh (build + systemd install)..."
  ./setup.sh
  log "Starting platform..."
  systemctl --user start helix.target
}

cmd_start() {
  log "Starting helix.target..."
  systemctl --user start helix.target
  ok "start requested"
}

cmd_stop() {
  log "Stopping helix.target..."
  systemctl --user stop helix.target
  ok "stop requested"
}

cmd_restart() {
  log "Restarting helix.target..."
  systemctl --user restart helix.target
  ok "restart requested"
}

cmd_status() {
  local unit="${1:-}"
  if [ -n "$unit" ]; then
    systemctl --user --no-pager status "$unit"
  else
    log "Helix platform units:"
    systemctl --user --no-pager --plain list-units 'helix*' 'llmsverifier*' || true
    echo
    log "Unit files:"
    systemctl --user --no-pager --plain list-unit-files 'helix*' 'llmsverifier*' || true
  fi
}

cmd_enable() {
  log "Enabling units..."
  for u in "${UNIT_NAMES[@]}"; do
    systemctl --user enable "$u" >/dev/null 2>&1 && ok "enabled $u" || warn "could not enable $u"
  done
  if [ "$(loginctl show-user "${USER}" -p Linger --value 2>/dev/null || echo no)" = "yes" ]; then
    ok "linger already enabled"
  elif loginctl enable-linger "${USER}" 2>/dev/null; then
    ok "linger enabled (units start at boot)"
  else
    warn "could not enable linger — run: sudo loginctl enable-linger ${USER}"
  fi
}

cmd_disable() {
  log "Disabling units..."
  for u in "${UNIT_NAMES[@]}"; do
    systemctl --user disable "$u" >/dev/null 2>&1 && ok "disabled $u" || warn "could not disable $u"
  done
}

cmd_validate() {
  exec "${REPO_ROOT}/scripts/validate-helix-platform.sh"
}

cmd_logs() {
  local unit="${1:-}"
  [ -n "$unit" ] || die "logs requires a unit name (e.g. helixcode-server)"
  exec journalctl --user -u "$unit" -f
}

main() {
  command -v systemctl >/dev/null 2>&1 || die "systemctl not found"
  systemctl --user show-environment >/dev/null 2>&1 || die "systemd --user not reachable"

  local cmd="${1:-help}"
  shift || true
  case "$cmd" in
    install)   cmd_install "$@" ;;
    start)     cmd_start "$@" ;;
    stop)      cmd_stop "$@" ;;
    restart)   cmd_restart "$@" ;;
    status)    cmd_status "$@" ;;
    enable)    cmd_enable "$@" ;;
    disable)   cmd_disable "$@" ;;
    validate)  cmd_validate "$@" ;;
    logs)      cmd_logs "$@" ;;
    help|-h|--help) usage ;;
    *) die "unknown command: $cmd (try 'help')" ;;
  esac
}

main "$@"
