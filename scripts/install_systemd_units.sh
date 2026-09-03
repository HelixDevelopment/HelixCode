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

# INSTALLED: every unit file is written to ~/.config/systemd/user/, so the whole
# platform stays addressable (`systemctl --user start llmsverifier`) and nothing
# is removed from the operator's reach.
#
# Ordered: infra first, server last. Install order does not imply start order
# (that is the units' After=/Requires=), but a stable list keeps output readable.
UNITS=(
  helix.target
  helixcode-infra.service
  llmsverifier.service
  helixllm-coder.service
  helixllm-gateway.service
  helixagent.service
  helixcode-server.service
)

# ENABLED (start at boot): a deliberate SUBSET of the above.
#
# Installing a unit makes it available; ENABLING it makes it start at boot. The
# two are separated because three installed units are not part of the deployed
# topology and enabling them would start real workloads unasked:
#
#   helixcode-infra.service  — boots ~10 containers including a Postgres+Redis
#                              pair that DUPLICATES the `helixcode-autoboot-*`
#                              pair helixcode-server already owns, plus Ollama,
#                              Weaviate and Selenium (§12.6 memory).
#   helixllm-coder.service   — loads Qwen3-Coder-30B onto the GPU.
#   llmsverifier.service     — not currently running; :8100 is unbound.
#
# None is removed, disabled or deleted — each stays one `systemctl --user enable
# <unit>` away (§11.4.122: no silent removal of an existing component). Override
# for a session with:  HELIX_ENABLE_UNITS="unit-a unit-b" scripts/install_systemd_units.sh
if [ -n "${HELIX_ENABLE_UNITS:-}" ]; then
  # shellcheck disable=SC2206  # deliberate word-split of an operator-supplied list
  ENABLE_UNITS=(${HELIX_ENABLE_UNITS})
else
  ENABLE_UNITS=(
    helix.target
    helixllm-gateway.service
    helixagent.service
    helixcode-server.service
  )
fi

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
#
# THIS CHECKOUT'S OWN BUILD IS PREFERRED over whatever happens to be on PATH.
# Rationale: this installer wires units for THIS repo — silently binding the
# service to some other checkout's binary found earlier on PATH is the class of
# mistake §11.4.111 exists to prevent.
#
# The previous order (PATH, then ~/.local/bin/helixllm) was also outright broken
# here: measured 2026-09-03, `helixllm` was NOT on PATH and
# ~/.local/bin/helixllm did NOT exist, so it resolved to a nonexistent path and
# installed a unit whose ExecStart could never run — failing at first start with
# a confusing 203/EXEC instead of at install time.
HELIXLLM_BIN=""
for candidate in \
    "${REPO_ROOT}/submodules/helix_llm/bin/helixllm" \
    "$(command -v helixllm 2>/dev/null || true)" \
    "${HOME}/.local/bin/helixllm"
do
  if [ -n "${candidate}" ] && [ -x "${candidate}" ]; then
    HELIXLLM_BIN="${candidate}"
    break
  fi
done
if [ -z "${HELIXLLM_BIN}" ]; then
  die "no executable helixllm found (looked in submodules/helix_llm/bin, PATH, ~/.local/bin) — build it first: (cd submodules/helix_llm && make build)"
fi

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
for u in "${ENABLE_UNITS[@]}"; do
  systemctl --user enable "$u" >/dev/null 2>&1 && log "enabled $u (starts at boot)" || warn "could not enable $u"
done
for u in "${UNITS[@]}"; do
  case " ${ENABLE_UNITS[*]} " in
    *" $u "*) ;;
    *) log "installed but NOT enabled: $u  (enable with: systemctl --user enable $u)" ;;
  esac
done
ok "enable set applied"

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
