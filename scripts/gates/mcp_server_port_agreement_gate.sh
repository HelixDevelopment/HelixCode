#!/usr/bin/env bash
# =============================================================================
# CM-MCP-SERVER-PORT-AGREEMENT
#
# Guards HXC-193 (High) + HXC-195 (Medium) — ONE root cause, two tickets.
#
# THE DEFECT (reproduced at RED_MODE=1, see below)
# ------------------------------------------------
# `submodules/helix_agent/plugins/mcp-server` is a small service we wrote and
# ship in a container. Its container definition set `MCP_PORT=9100`, published
# `9100`, and declared a health check against `http://localhost:9100/health`.
# The service read NOTHING from its environment (`grep -rn process.env src/`
# returned zero matches), so:
#
#   * MCP_PORT was INERT — an operator moving the service by editing it saw no
#     change and no error, with no way to tell why                    (HXC-195)
#   * the transport defaulted to `stdio`, and `runSSE()` — the ONLY code path
#     that calls `listen()` — was never reached, so NOTHING listened on ANY
#     port, let alone the published one                               (HXC-193)
#   * the health check therefore queried an address where nothing could ever
#     answer and reported the container unhealthy no matter how well the
#     service was working; anything that replaces unhealthy containers would
#     do so endlessly                                                 (HXC-193)
#
# A THIRD cause, not in either ticket, was found while fixing and is also
# guarded here: the compose health check invoked `curl`, which is NOT present
# in `node:20-alpine` (verified against the real image). Even once something
# WAS listening, that probe could not have succeeded.
#
# WHAT THIS GUARD ASSERTS
# -----------------------
# The three ports the two tickets say must line up actually line up, and the
# probe is runnable:
#
#   (1) BIND      — the port the service ACTUALLY binds, observed at RUNTIME by
#                   starting the built artifact under the container's own
#                   environment and reading the listening socket. Not parsed
#                   from source: a source-only check is exactly the class of
#                   assertion that let this defect ship (§11.4.108 — SOURCE
#                   green says nothing about RUNTIME).
#   (2) PUBLISHED — the container-side port in compose `ports:` and `EXPOSE`.
#   (3) PROBED    — the port the health checks address, in BOTH compose and
#                   the Dockerfile.
#   plus: a transport that actually opens a listener, and a health-check
#   command that exists inside the image.
#
# Ports are not pinned to a literal, so a deliberate move of the service to a
# different port cannot false-FAIL this gate (§11.4.201) — it fails only when
# the three DISAGREE, which is the defect.
#
# POLARITY (§11.4.115) — one source, two roles
#   RED_MODE=1  Reproduce the defect on a synthesized PRE-FIX artifact (copies
#               only, the real tree is never touched) and assert this guard
#               FAILs on it. Proves the guard is not a blind test.
#   RED_MODE=0  DEFAULT. The standing GREEN regression guard.
#
# EXIT CODES
#   0  gate satisfied for the active RED_MODE
#   1  gate VIOLATED (substantive — the ports disagree / the probe is unusable)
#   2  gate could NOT RUN (§11.4.3 environment SKIP: no node, no built artifact,
#      no submodule checkout). Certifies nothing — reported as a FAIL by the
#      runner, but MUST NOT be worded as a detected regression.
# =============================================================================
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
GATE="CM-MCP-SERVER-PORT-AGREEMENT"
RED_MODE="${RED_MODE:-0}"

SVC_DIR="${SVC_DIR_OVERRIDE:-$ROOT/submodules/helix_agent/plugins/mcp-server}"
COMPOSE="${COMPOSE_OVERRIDE:-$ROOT/submodules/helix_agent/docker-compose.protocols.yml}"
DOCKERFILE="${DOCKERFILE_OVERRIDE:-$SVC_DIR/Dockerfile}"
ENTRY="${ENTRY_OVERRIDE:-$SVC_DIR/dist/index.js}"

echo "$GATE  RED_MODE=$RED_MODE"

# --- RED_MODE: §11.4.115 reproduce-the-defect-on-a-broken-artifact -----------
# Synthesize the PRE-FIX artifact on COPIES and assert this same checker FAILs.
# The mutation is the exact pre-fix shape, not an arbitrary break:
#   * dist/index.js  — `envConfig()` -> `{}`, i.e. the service reads NOTHING
#                      from its environment (the shared root cause)
#   * Dockerfile     — MCP_TRANSPORT removed, wget/curl probe restored
#   * compose        — MCP_TRANSPORT removed, `curl` probe restored
if [[ "$RED_MODE" == "1" ]]; then
  if [[ ! -f "$ENTRY" || ! -f "$COMPOSE" || ! -f "$DOCKERFILE" ]]; then
    echo "$GATE: SKIP — cannot synthesize the pre-fix artifact; sources absent" >&2
    exit 2
  fi
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT
  mkdir -p "$TMP/svc/dist"
  # Copy the whole service dir so relative requires (./transport, ./tools) resolve.
  cp -r "$SVC_DIR/." "$TMP/svc/" 2>/dev/null || true
  # The root cause: the service reads no configuration from its environment.
  sed 's/const config = envConfig();/const config = {};/' \
      "$ENTRY" > "$TMP/svc/dist/index.js"
  # The cascade: container knobs written to match the inert setting.
  grep -v 'MCP_TRANSPORT' "$DOCKERFILE" > "$TMP/Dockerfile"
  grep -v 'MCP_TRANSPORT' "$COMPOSE" \
    | sed 's#- "require(.http.).get.*#- CMD\n        - curl#' > "$TMP/compose.yml"

  # CASE A — the full pre-fix artifact: inert env + container knobs written to
  # match it + a curl probe that is not in the image.
  if RED_MODE=0 SVC_DIR_OVERRIDE="$TMP/svc" ENTRY_OVERRIDE="$TMP/svc/dist/index.js" \
       COMPOSE_OVERRIDE="$TMP/compose.yml" DOCKERFILE_OVERRIDE="$TMP/Dockerfile" \
       "$0" >/dev/null 2>&1; then
    echo "$GATE: RED FAIL (case A) — the guard PASSed on the full pre-fix artifact (blind test)" >&2
    exit 1
  fi

  # CASE B — the load-bearing one. Container definitions are the CORRECT,
  # post-fix ones; ONLY the root cause is reverted (the service ignores its
  # environment). Every static check therefore PASSes and the gate must be
  # driven to FAIL by the RUNTIME bind probe alone. Without this case a future
  # edit could quietly reduce the gate to a source/config-only check — the
  # exact §11.4.108 SOURCE-green-says-nothing-about-RUNTIME failure that let
  # this defect ship — and case A would still pass, hiding the regression.
  if RED_MODE=0 SVC_DIR_OVERRIDE="$TMP/svc" ENTRY_OVERRIDE="$TMP/svc/dist/index.js" \
       "$0" >/dev/null 2>&1; then
    echo "$GATE: RED FAIL (case B) — the guard PASSed with correct container definitions but a service that ignores its environment; the RUNTIME bind probe is not actually falsifying anything" >&2
    exit 1
  fi

  echo "$GATE: RED OK — guard FAILs on both pre-fix shapes: (A) full pre-fix artifact, (B) correct container definitions but a service that ignores its environment, caught by the RUNTIME bind probe alone — the exact HXC-193/HXC-195 root cause"
  exit 0
fi

# --- Environment preconditions (§11.4.3 SKIP, never a fake pass) -------------
command -v node >/dev/null 2>&1 || { echo "$GATE: SKIP — node not on PATH; the runtime bind probe cannot execute" >&2; exit 2; }
[[ -f "$ENTRY" ]]      || { echo "$GATE: SKIP — built artifact absent ($ENTRY); run 'npm run build' in the service dir" >&2; exit 2; }
[[ -f "$COMPOSE" ]]    || { echo "$GATE: SKIP — compose file absent ($COMPOSE)" >&2; exit 2; }
[[ -f "$DOCKERFILE" ]] || { echo "$GATE: SKIP — Dockerfile absent ($DOCKERFILE)" >&2; exit 2; }

FAILURES=0
fail() { echo "$GATE: FAIL — $*" >&2; FAILURES=$((FAILURES + 1)); }

# Extract ONLY the helixagent-mcp service block: from its own 2-space key to
# the next 2-space key. The start pattern is anchored (`^  helixagent-mcp:`)
# because the bare string also appears mid-value in a later service's
# `HELIXAGENT_MCP_URL=http://helixagent-mcp:9100`, which would otherwise
# restart the sed range and drag a sibling service's health check into scope.
svc_block() { sed -n '/^  helixagent-mcp:/,/^  [a-z]/p' "$1"; }

# Strip `#` comment lines before scanning a block for a command name, so this
# gate's own prose about `curl` cannot be mistaken for a curl invocation.
strip_comments() { grep -vE '^[[:space:]]*#'; }

# --- (2) PUBLISHED + declared env, read from the container definitions -------
# Container-side of the `ports:` mapping ("<host>:<container>") for the service.
PUBLISHED="$(svc_block "$COMPOSE" \
  | grep -oE '"[^"]*:[0-9]+"' | tail -1 | grep -oE '[0-9]+"$' | tr -d '"')"
EXPOSED="$(grep -oE '^EXPOSE[[:space:]]+[0-9]+' "$DOCKERFILE" | grep -oE '[0-9]+$')"
COMPOSE_MCP_PORT="$(svc_block "$COMPOSE" \
  | grep -oE 'MCP_PORT=[0-9]+' | head -1 | cut -d= -f2)"
DOCKER_MCP_PORT="$(grep -oE '^ENV MCP_PORT=[0-9]+' "$DOCKERFILE" | cut -d= -f2)"

[[ -n "$PUBLISHED" ]] || fail "could not read the published container-side port from $COMPOSE"
[[ -n "$COMPOSE_MCP_PORT" ]] || fail "compose declares no MCP_PORT for the service"

# --- Transport: without a listening transport, no port can ever agree --------
COMPOSE_TRANSPORT="$(svc_block "$COMPOSE" \
  | grep -oE 'MCP_TRANSPORT=[a-z]+' | head -1 | cut -d= -f2)"
DOCKER_TRANSPORT="$(grep -oE '^ENV MCP_TRANSPORT=[a-z]+' "$DOCKERFILE" | cut -d= -f2)"
if [[ "$COMPOSE_TRANSPORT" != "sse" ]]; then
  fail "compose does not select the SSE transport (MCP_TRANSPORT=sse). Under stdio the service binds NO port and serves no /health, so the published port and the health check below address something that cannot exist — the HXC-193 shape."
fi
if [[ "$DOCKER_TRANSPORT" != "sse" ]]; then
  fail "Dockerfile does not default MCP_TRANSPORT=sse; a plain 'docker run' of this image binds nothing while EXPOSE/HEALTHCHECK claim port $EXPOSED"
fi

# --- (3) PROBED: the health checks must address the same port ---------------
# Both probes read MCP_PORT at runtime; a literal port hardcoded in the probe
# would be a second source of truth that can silently drift from the bind port.
COMPOSE_HC="$(svc_block "$COMPOSE" | sed -n '/healthcheck:/,/retries:/p' | strip_comments)"
DOCKER_HC="$(sed -n '/^HEALTHCHECK/,/^$/p' "$DOCKERFILE" | strip_comments)"

grep -q 'MCP_PORT' <<<"$COMPOSE_HC" \
  || fail "the compose health check does not derive its port from MCP_PORT — it is a second source of truth that can drift from the port the service binds"
grep -q 'MCP_PORT' <<<"$DOCKER_HC" \
  || fail "the Dockerfile HEALTHCHECK does not derive its port from MCP_PORT"

# The probe binary must exist inside the image. `curl` does NOT ship in
# node:20-alpine (verified against the image), so a curl probe can never pass.
if grep -qE '(^|[^a-z])curl([^a-z]|$)' <<<"$COMPOSE_HC"; then
  fail "the compose health check invokes 'curl', which is not installed in node:20-alpine — the probe can never succeed regardless of what is listening"
fi
if grep -qE '(^|[^a-z])curl([^a-z]|$)' <<<"$DOCKER_HC"; then
  fail "the Dockerfile HEALTHCHECK invokes 'curl', which is not installed in node:20-alpine"
fi

# --- Declared ports must agree with each other ------------------------------
for pair in "EXPOSE:$EXPOSED" "compose-MCP_PORT:$COMPOSE_MCP_PORT" "Dockerfile-MCP_PORT:$DOCKER_MCP_PORT"; do
  name="${pair%%:*}"; value="${pair#*:}"
  if [[ -n "$value" && -n "$PUBLISHED" && "$value" != "$PUBLISHED" ]]; then
    fail "$name is $value but the container publishes $PUBLISHED — a person changing one would not move the other"
  fi
done

# --- (1) BIND: RUNTIME evidence. What does the service ACTUALLY listen on? ---
# Start the built artifact under the container's own environment and observe
# the socket. This is the assertion a source-only gate cannot make.
OBSERVED=""
if [[ -n "$COMPOSE_MCP_PORT" && "$COMPOSE_TRANSPORT" == "sse" ]]; then
  PROBE_LOG="$(mktemp)"
  env NODE_ENV=production MCP_TRANSPORT="$COMPOSE_TRANSPORT" MCP_PORT="$COMPOSE_MCP_PORT" \
      node "$ENTRY" </dev/null >/dev/null 2>"$PROBE_LOG" &
  SVC_PID=$!
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    sleep 0.5
    kill -0 "$SVC_PID" 2>/dev/null || break
    OBSERVED="$(ss -ltnp 2>/dev/null | grep "pid=$SVC_PID," | grep -oE ':[0-9]+ ' | head -1 | tr -d ': ')"
    [[ -n "$OBSERVED" ]] && break
  done
  kill "$SVC_PID" 2>/dev/null
  wait "$SVC_PID" 2>/dev/null
  BANNER="$(head -1 "$PROBE_LOG")"
  rm -f "$PROBE_LOG"

  if [[ -z "$OBSERVED" ]]; then
    fail "RUNTIME: the service bound NO port under the container's own environment (MCP_TRANSPORT=$COMPOSE_TRANSPORT MCP_PORT=$COMPOSE_MCP_PORT). The health check probes ${COMPOSE_MCP_PORT}/health, so it can never answer. Service said: ${BANNER:-<nothing>}"
  elif [[ "$OBSERVED" != "$COMPOSE_MCP_PORT" ]]; then
    fail "RUNTIME: the service bound port $OBSERVED but the container publishes/probes $COMPOSE_MCP_PORT — MCP_PORT is not steering the listener"
  fi
else
  fail "cannot run the RUNTIME bind probe: transport/port not declared as expected in compose"
fi

# --- Verdict ----------------------------------------------------------------
if [[ "$FAILURES" -gt 0 ]]; then
  echo "$GATE: FAIL — $FAILURES check(s) failed" >&2
  exit 1
fi

echo "$GATE: PASS — bind=$OBSERVED published=$PUBLISHED probed=via-MCP_PORT transport=$COMPOSE_TRANSPORT; all three agree, observed at runtime under the container's own environment"
exit 0
