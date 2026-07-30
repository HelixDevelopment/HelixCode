#!/usr/bin/env bash
# mcp_server_cors_allowlist_gate.sh — CM-MCP-SERVER-CORS-ALLOWLIST
#
# WHAT THIS GUARDS (HXC-212)
#   The mcp-server SSE transport once answered every browser with
#   `Access-Control-Allow-Origin: *` on BOTH the preflight and the response
#   path, never reading `Origin`. That hole was DORMANT while the service only
#   spoke stdio and never opened a socket.
#
#   Commit 30c81925 (HXC-193/HXC-195) correctly repaired a real defect — the
#   service read no environment configuration, so its port knob was inert and
#   its health check could never pass. Part of that fix sets
#   `ENV MCP_TRANSPORT=sse` in the Dockerfile. So the container now SELECTS the
#   SSE path, which listens on a published port. A correct fix to one defect
#   ACTIVATED a second one that had only ever been safe by accident.
#
#   e27581da closed it with an origin allowlist. This gate keeps it closed.
#
# WHY THIS CHECKS TWO LAYERS (§11.4.108)
#   The tracked `dist/index.js` carried the wildcard independently of
#   `src/index.ts`. A source-only check would have passed while a fresh clone
#   running `node dist/index.js` without building still served the hole. So the
#   SOURCE layer and the shipped ARTIFACT layer are asserted separately.
#
# WHY IT RUNS THE SUBMODULE'S OWN SUITE RATHER THAN REIMPLEMENTING IT
#   The guard in the submodule spawns the real built artifact as a child process
#   and makes real HTTP requests over a real socket. Reimplementing that here
#   would duplicate it and drift. §11.4.28 keeps the engine in the submodule;
#   this wrapper only binds it into the consuming project's standing sweep.
#
# ASSERTIONS
#   A. GREEN  — the submodule's `npm test` (build + RED_MODE=0) passes.
#   B. RED    — `npm run test:cors:red` (RED_MODE=1) FAILS on the fixed artifact.
#               This is the load-bearing half: a guard that cannot be made to
#               fail proves nothing (§1.1). If B ever PASSES, either the fix
#               regressed or the guard became a tautology — both are failures.
#   C. SOURCE — no wildcard Allow-Origin literal in src/.
#   D. ARTIFACT — no wildcard Allow-Origin literal in the tracked dist/.
#
# EXIT CODES  (§11.4.3 / the G21 lesson: an environment SKIP is NOT a defect)
#   0  all assertions hold
#   1  substantive violation — the allowlist regressed, or the guard went blind
#   2  could not RUN (submodule absent, node absent, npm absent). Certifies
#      NOTHING. Never reported as a detected regression.
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
SVC="$ROOT/submodules/helix_agent/plugins/mcp-server"

skip() { echo "CM-MCP-SERVER-CORS-ALLOWLIST: SKIP — $1" >&2; exit 2; }
fail() { echo "CM-MCP-SERVER-CORS-ALLOWLIST: FAIL — $1" >&2; exit 1; }

# --- environment preconditions: SKIP, never FAIL -----------------------------
[ -d "$SVC" ]                || skip "submodule path absent: $SVC (submodule not checked out?)"
[ -f "$SVC/package.json" ]   || skip "no package.json at $SVC"
command -v node >/dev/null   || skip "node not on PATH"
command -v npm  >/dev/null   || skip "npm not on PATH"
grep -q '"test:cors"' "$SVC/package.json" \
    || skip "package.json has no test:cors script — guard not wired in the submodule"

violations=0
assertions=0

# --- C. SOURCE layer ---------------------------------------------------------
assertions=$((assertions + 1))
src_hits=$(grep -rn "['\"]Access-Control-Allow-Origin['\"][[:space:]]*:[[:space:]]*['\"]\*['\"]" \
    "$SVC/src" 2>/dev/null | wc -l)
if [ "$src_hits" -ne 0 ]; then
    echo "  VIOLATION (source): wildcard Allow-Origin literal present in src/" >&2
    grep -rn "['\"]Access-Control-Allow-Origin['\"][[:space:]]*:[[:space:]]*['\"]\*['\"]" "$SVC/src" >&2
    violations=$((violations + 1))
else
    echo "  ok  source:   no wildcard Allow-Origin literal in src/"
fi

# --- D. ARTIFACT layer (§11.4.108 — the tracked build output ships too) ------
assertions=$((assertions + 1))
if [ -d "$SVC/dist" ]; then
    dist_hits=$(grep -rn "Access-Control-Allow-Origin[\"']*[[:space:]]*:[[:space:]]*[\"']\*" \
        "$SVC/dist" 2>/dev/null | wc -l)
    if [ "$dist_hits" -ne 0 ]; then
        echo "  VIOLATION (artifact): wildcard Allow-Origin literal in the TRACKED dist/ —" >&2
        echo "      a fresh clone running 'node dist/index.js' without building serves the hole" >&2
        grep -rn "Access-Control-Allow-Origin[\"']*[[:space:]]*:[[:space:]]*[\"']\*" "$SVC/dist" >&2
        violations=$((violations + 1))
    else
        echo "  ok  artifact: no wildcard Allow-Origin literal in tracked dist/"
    fi
else
    echo "  ok  artifact: no tracked dist/ present (nothing to ship stale)"
fi

# --- A. GREEN ----------------------------------------------------------------
assertions=$((assertions + 1))
green_out="$(cd "$SVC" && npm test 2>&1)"; green_rc=$?
if [ "$green_rc" -ne 0 ]; then
    echo "  VIOLATION (green): the submodule CORS guard FAILED — the allowlist regressed" >&2
    printf '%s\n' "$green_out" | tail -20 >&2
    violations=$((violations + 1))
else
    echo "  ok  green:    hostile origin denied (preflight + simple), permitted origin echoed"
    echo "               with Vary: Origin, no-Origin client unaffected"
fi

# --- B. RED (falsifiability) -------------------------------------------------
assertions=$((assertions + 1))
red_out="$(cd "$SVC" && npm run test:cors:red 2>&1)"; red_rc=$?
if [ "$red_rc" -eq 0 ]; then
    echo "  VIOLATION (red): RED_MODE=1 PASSED on the fixed artifact." >&2
    echo "      RED_MODE=1 asserts the DEFECT IS PRESENT. Passing means either the" >&2
    echo "      wildcard is back, or the guard has decayed into a tautology that" >&2
    echo "      agrees with whatever the code does. Both are failures (§1.1)." >&2
    echo "      Do NOT weaken this assertion to make the sweep green (§11.4.120)." >&2
    violations=$((violations + 1))
else
    echo "  ok  red:      RED_MODE=1 correctly fails — defect absent, guard falsifiable"
fi

echo
if [ "$violations" -ne 0 ]; then
    fail "$violations of $assertions assertion(s) violated — reopen HXC-212 (§11.4.34) rather than relaxing this gate"
fi
echo "CM-MCP-SERVER-CORS-ALLOWLIST: PASS — $assertions assertion(s) evaluated, 0 violation(s)"
echo "RESULT: green=1/1 red_falsifiable=1/1 source_clean=1/1 artifact_clean=1/1"
exit 0
