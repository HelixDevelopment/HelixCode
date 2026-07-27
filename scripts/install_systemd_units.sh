#!/usr/bin/env bash
#
# install_systemd_units.sh — install the Helix platform's systemd *user* units.
#
# Installs every unit in scripts/systemd/ into ~/.config/systemd/user/, expanding
# the @HELIX_ROOT@ / @HELIXLLM_BIN@ placeholders for THIS checkout, then enables
# them so the whole platform comes up on system boot and survives restarts.
#
# User scope (not system scope) is deliberate: the services run rootless podman
# (§11.4.161) as the invoking user and read that user's ~/.config/cdi GPU specs.
# `loginctl enable-linger` is what makes user units start at BOOT rather than at
# first login — without it these would only start when the operator logs in.
#
# Idempotent: safe to re-run. Re-running picks up unit-file edits.
#
# Usage:
#   scripts/install_systemd_units.sh              # install + enable (no restart)
#   scripts/install_systemd_units.sh --start      # install + enable + start now
#   scripts/install_systemd_units.sh --uninstall  # stop + disable + remove
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UNIT_SRC="${REPO_ROOT}/scripts/systemd"
UNIT_DST="${XDG_CONFIG_HOME:-${HOME}/.config}/systemd/user"

# Ordered: infra first, server last. Enable order does not imply start order
# (that is the units' After=/Requires=), but a stable list keeps output readable.
UNITS=(
  helix.target
  helixcode-infra.service
  helixllm-coder.service
  helixllm-gateway.service
  helixagent.service
  helixcode-server.service
)

log()  { printf '  %s\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*"; }
die()  { printf '  \033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

# --- preflight ---------------------------------------------------------------
command -v systemctl >/dev/null 2>&1 || die "systemctl not found (systemd required)"
systemctl --user show-environment >/dev/null 2>&1 \
  || die "no systemd --user instance reachable (need a logind session or DBUS_SESSION_BUS_ADDRESS)"

# Resolve the helixllm binary rather than hardcoding a path (§11.4.111:
# resolve by name, not by a baked-in location).
HELIXLLM_BIN="$(command -v helixllm 2>/dev/null || true)"
[ -n "${HELIXLLM_BIN}" ] || HELIXLLM_BIN="${HOME}/.local/bin/helixllm"

# --- uninstall ---------------------------------------------------------------
if [ "${1:-}" = "--uninstall" ]; then
  echo "Uninstalling Helix systemd units..."
  for u in "${UNITS[@]}"; do
    systemctl --user disable --now "$u" >/dev/null 2>&1 || true
    rm -f "${UNIT_DST}/${u}"
    log "removed $u"
  done
  systemctl --user daemon-reload
  ok "uninstalled"
  exit 0
fi

echo "Installing Helix systemd user units"
log "repo   : ${REPO_ROOT}"
log "target : ${UNIT_DST}"
log "helixllm: ${HELIXLLM_BIN}"
echo

mkdir -p "${UNIT_DST}"

# --- install -----------------------------------------------------------------
for u in "${UNITS[@]}"; do
  src="${UNIT_SRC}/${u}"
  [ -f "$src" ] || die "missing unit source: $src"
  # Expand placeholders. '|' delimiter so paths containing '/' are safe.
  sed -e "s|@HELIX_ROOT@|${REPO_ROOT}|g" \
      -e "s|@HELIXLLM_BIN@|${HELIXLLM_BIN}|g" \
      "$src" > "${UNIT_DST}/${u}"
  # Fail loudly rather than installing a unit with an unexpanded placeholder,
  # which would produce a baffling runtime error instead of an install error.
  if grep -q '@[A-Z_]*@' "${UNIT_DST}/${u}"; then
    die "unexpanded placeholder left in ${u}: $(grep -o '@[A-Z_]*@' "${UNIT_DST}/${u}" | sort -u | tr '\n' ' ')"
  fi
  log "installed $u"
done

systemctl --user daemon-reload
ok "daemon-reload"

# --- linger: the difference between "starts at login" and "starts at boot" ----
if [ "$(loginctl show-user "${USER}" -p Linger --value 2>/dev/null || echo no)" = "yes" ]; then
  ok "linger already enabled (units start at boot)"
elif loginctl enable-linger "${USER}" 2>/dev/null; then
  ok "linger enabled (units now start at boot)"
else
  warn "could not enable linger — units will start at LOGIN, not at BOOT."
  warn "fix with: sudo loginctl enable-linger ${USER}"
fi

# --- enable ------------------------------------------------------------------
for u in "${UNITS[@]}"; do
  systemctl --user enable "$u" >/dev/null 2>&1 && log "enabled $u" || warn "could not enable $u"
done
ok "all units enabled"

# --- optional start ----------------------------------------------------------
if [ "${1:-}" = "--start" ]; then
  echo
  echo "Starting helix.target (this can take several minutes on a cold model cache)..."
  systemctl --user start helix.target || warn "helix.target start reported failure — see status below"
  echo
  systemctl --user --no-pager --plain list-units 'helix*' || true
fi

echo
ok "done. Inspect with:  systemctl --user status helixagent"
