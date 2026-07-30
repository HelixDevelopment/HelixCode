#!/usr/bin/env bash
#
# CM-MOCK-SERVICES-CORS-ALLOWLIST — HXC-194 standing regression guard.
#
# THE DEFECT THIS GUARDS. Both e2e mock services (mock LLM provider, mock
# Slack) hand-rolled their Gin CORS middleware instead of using the framework's
# own CORS toolkit. Each emitted a blanket "Access-Control-Allow-Origin: *" on
# every response and never inspected the request Origin, so any web page on any
# site — opened by anyone able to route to the service — could drive every route
# and read the responses. On the Slack mock that meant reading back every
# captured message and webhook body and clearing that state via DELETE.
#
# NOT the published gin-contrib/cors advisory. Neither module has ever depended
# on that package (go.mod requires gin-gonic/gin only). Patching or upgrading it
# would silence the scanner while leaving these holes open — which is precisely
# why this gate asserts OUR behaviour rather than a dependency version.
#
# WHAT THIS GATE PROVES (runtime evidence, §11.4.5 — not a grep verdict):
#   1. GREEN  — the guard suite passes at RED_MODE=0 in BOTH modules: a hostile
#               origin gets no Allow-Origin on simple AND preflight requests, a
#               permitted origin is still accepted, a no-Origin (non-browser)
#               client is unaffected, and "*" is never emitted.
#   2. RED    — the SAME source at RED_MODE=1 FAILS. This is the §1.1 paired
#               mutation built into the test's polarity switch: it passes only
#               against the pre-fix wildcard middleware. A guard that cannot
#               fail proves nothing, so an unexpectedly-passing RED run is
#               reported as a FAILURE of this gate.
#   3. SOURCE — no wildcard Allow-Origin literal has crept back into either
#               service's source.
#
# EXIT CODES, mirroring G21/G22 deliberately:
#   0 — pass
#   1 — substantive failure (a real regression)
#   2 — §11.4.3 environment SKIP (no Go toolchain / module tree absent). Still
#       reported as a failure by the sweep, because a guard that could not run
#       has certified nothing — but it MUST NOT be worded as a detected hole.
#
# HONEST BOUNDARY (§11.4.108). This proves the SOURCE layer and a `go test`
# form of the RUNTIME layer. It does NOT prove behaviour of a deployed
# container: `go test` rebuilds from source, so a stale image would not be
# caught here.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MOCKS="$ROOT/helix_code/tests/e2e/mocks"

MODULES=(
    "llm-provider"
    "slack"
)

fail=0

echo "== CM-MOCK-SERVICES-CORS-ALLOWLIST (HXC-194) =="

if ! command -v go >/dev/null 2>&1; then
    echo "SKIP(env): no Go toolchain on PATH — gate could not run, certified nothing"
    exit 2
fi

for m in "${MODULES[@]}"; do
    dir="$MOCKS/$m"
    if [[ ! -f "$dir/go.mod" ]]; then
        echo "SKIP(env): module tree absent: $dir"
        exit 2
    fi
done

for m in "${MODULES[@]}"; do
    dir="$MOCKS/$m"
    echo "-- module: $m"

    # (1) GREEN: the standing guard must pass.
    green_out="$(cd "$dir" && RED_MODE=0 go test ./cmd/... -run TestCORS -count=1 2>&1)"
    green_rc=$?
    if [[ "$green_rc" -ne 0 ]]; then
        echo "   FAIL green: CORS allowlist guard is RED at RED_MODE=0 — the origin"
        echo "               check has regressed in $m"
        echo "$green_out" | tail -20
        fail=1
        continue
    fi
    echo "   ok   green: hostile origin denied (simple + preflight), permitted origin accepted,"
    echo "              no-Origin client unaffected, no wildcard emitted"

    # (2) RED: the same source at RED_MODE=1 must FAIL against fixed code.
    #     Exit 0 here would mean the wildcard is back OR the guard went blind.
    red_out="$(cd "$dir" && RED_MODE=1 go test ./cmd/... -run TestCORS_HostileOriginIsRejected -count=1 2>&1)"
    red_rc=$?
    if [[ "$red_rc" -eq 0 ]]; then
        echo "   FAIL red: RED_MODE=1 PASSED against $m — it asserts the wildcard defect is"
        echo "             PRESENT, so either the hole is back or the guard is no longer"
        echo "             falsifiable. Both are release blockers."
        echo "$red_out" | tail -20
        fail=1
        continue
    fi
    echo "   ok   red:   RED_MODE=1 correctly fails — defect absent, guard falsifiable"

    # (3) SOURCE: no wildcard literal in the service source.
    if grep -RIn --include='*.go' 'Access-Control-Allow-Origin"[[:space:]]*,[[:space:]]*"\*"' \
        "$dir/cmd" "$dir/handlers" 2>/dev/null | grep -v '_test\.go:' > /tmp/hxc194-wildcard-"$m".out; then
        echo "   FAIL source: a wildcard Access-Control-Allow-Origin literal is back in $m:"
        cat /tmp/hxc194-wildcard-"$m".out
        fail=1
        continue
    fi
    echo "   ok   source: no wildcard Allow-Origin literal in service code"
done

if [[ "$fail" -ne 0 ]]; then
    echo "RESULT: FAIL — mock-service CORS origin allowlist regressed"
    exit 1
fi

echo "RESULT: PASS — green=2/2 red_falsifiable=2/2 source_clean=2/2"
exit 0
