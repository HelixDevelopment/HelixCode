#!/usr/bin/env bash
# HXC-244 — the gateway's health endpoint must actually CHECK something.
#
# THE DEFECT
# ----------
# GET /internal/health returns 200 {"status":"healthy","components":[]} — and it
# returns exactly that no matter what is broken. cmd/helixllm/main.go:148
# constructs health.NewChecker() and hands it to the server with no Register()
# or RegisterOptional() call in between. Measured 2026-08-08: of the 41
# .Register( call sites in the tree, ZERO are on the health checker (they belong
# to the tool registry, the provider router, the MCP registry, and metrics).
#
# The handler itself is honest — server.go:183 returns 503 when the report is
# not healthy. The aggregate simply has nothing to aggregate, so "healthy" is
# the only verdict it can produce. Redis, the verifier, and every provider could
# be dead and this endpoint would still answer 200 healthy.
#
# That is the §11.4.1 absence-of-error PASS, sitting in the one endpoint an
# orchestrator, a load balancer, or a human on-call would trust. It is worse
# than having no health endpoint at all: no endpoint is an honest gap, while
# this one actively asserts a state it never verified.
#
# WHY IT ASSERTS THE COMPONENT LIST, NOT THE STATUS FIELD
# -------------------------------------------------------
# Asserting status=="healthy" is the bluff — it passes today, on the broken
# build, for the wrong reason. The falsifiable property is that the report names
# the dependencies it checked. A report that checks Redis and finds it up is
# evidence; a report that checks nothing and says "healthy" is a claim.
#
# So: the component list MUST be non-empty, and each entry MUST carry a name.
# What the components ARE is deliberately not constrained here — that belongs to
# the fix's design, and over-specifying would make this guard brittle against a
# correct implementation that names them differently.
#
# POLARITY (§11.4.115): RED_MODE=1 asserts the pre-fix broken shape — an empty
# component list — and MUST fail once the fix lands. RED_MODE=0 is the standing
# guard and is now the DEFAULT.
#
# THE DEFAULT FLIPPED 2026-08-09, AND WHY
# ---------------------------------------
# This guard shipped with RED_MODE=1 as its default because it was authored
# against the broken build, and its own header instructed: "flip this guard to
# RED_MODE=0 and register it in the standing suite" once the fix landed. The
# fix has landed — helix_llm commit 8260cf8 ("make /internal/health actually
# check the gateway's dependencies") registers the checks at
# cmd/helixllm/main.go:422, and the live endpoint measured 2026-08-09 names
# four real components (llm_providers, kv_cache_redis, vector_store_qdrant,
# llms_verifier). Leaving the default at 1 would mean a bare invocation FAILs
# on a CORRECT tree, which is itself the §11.4.201 false-positive refusal. The
# RED reproduction is not lost, only un-defaulted: RED_MODE=1 still replays the
# defect against a pre-fix deployment (§11.4.115).
#
# EXIT CONTRACT — ABSENT IS NOT DEFECTIVE  (§11.4.201 / §11.4.3)
# --------------------------------------------------------------
#   0  GREEN — a LIVE report was read and it names the components it checked
#   1  FAIL  — a LIVE report was read and it names nothing (or nothing named)
#   2  SKIP  — nothing was listening; the guard certified nothing and says so
#
# SKIP is keyed to curl exit 6 (unresolvable host) and 7 (connection refused)
# ONLY — the two outcomes that mean there is no service. Every other outcome
# implies a listener answered badly and stays a FAIL: 28 (timeout — a closed
# loopback port refuses instantly, so a hang means a listener wedged), 35/60
# (TLS failure), 56/52 (connection established then reset or empty — an
# independent review MEASURED 56, not the 52 an earlier revision of this
# comment claimed), and any HTTP response including a 500. The defect this
# guard exists to catch — a gateway that answers 200 healthy while checking
# nothing — necessarily connects, so it can never reach SKIP (§11.4.69
# CM-NO-FAIL-OPEN-SKIP).
#
# This holds only because the request bypasses any proxy (see --noproxy below);
# without that, exit 7 could describe the proxy rather than the target.
set -uo pipefail

URL="${HXC244_URL:-https://localhost:8443/internal/health}"
TIMEOUT="${HXC244_TIMEOUT:-15}"
RED_MODE="${RED_MODE:-0}"
fail() { echo "GUARD FAILED (HXC-244): $*" >&2; exit 1; }
skip_env() { echo "SKIP(env): HXC-244 — $*" >&2; exit 2; }

# --noproxy '*' is load-bearing (independent review, F1): curl honours
# https_proxy/HTTPS_PROXY, so a proxy variable leaked into the caller's
# environment makes exit 7 mean "could not reach the PROXY" rather than
# "nothing is listening at the target". The reviewer demonstrated a SKIP against
# the LIVE, healthy gateway that way. Bypassing the proxy is what makes rc=7 a
# fact about the TARGET, which the SKIP branch below depends on entirely.
BODY="$(curl -sk --noproxy '*' -m "$TIMEOUT" "$URL" 2>/dev/null)"
CURL_RC=$?   # straight off curl, never after a pipe (§11.4.6)

# Absence first, in BOTH polarities: an absent endpoint neither reproduces the
# defect nor proves it gone.
if [ "$CURL_RC" -eq 6 ] || [ "$CURL_RC" -eq 7 ]; then
    skip_env "nothing is listening at $URL (curl rc=$CURL_RC — $([ "$CURL_RC" -eq 6 ] \
&& echo 'host does not resolve' || echo 'connection refused')). The gateway is \
not deployed here; an absent health endpoint has not regressed. Start the \
stack to enforce this guard."
fi

if [ "$CURL_RC" -ne 0 ] || [ -z "$BODY" ]; then
    fail "the health endpoint is REACHABLE but returned no usable body \
(curl rc=$CURL_RC, $(printf '%s' "$BODY" | wc -c) bytes). Something accepted \
the connection and failed to answer — a live defect, not an absent service."
fi

VERDICT="$(printf '%s' "$BODY" | python3 -c '
import sys, json
try:
    d = json.load(sys.stdin)
except Exception as e:
    print(f"NOT_JSON: {e}"); sys.exit(2)
comps = d.get("components")
if comps is None:
    print("NO_COMPONENTS_KEY: report omits the components field entirely")
    sys.exit(3)
if not isinstance(comps, list):
    print(f"COMPONENTS_NOT_LIST: got {type(comps).__name__}"); sys.exit(4)
if len(comps) == 0:
    print("EMPTY: status=" + repr(d.get("status")) + " but zero components checked")
    sys.exit(5)
unnamed = [c for c in comps if not (isinstance(c, dict) and str(c.get("name", "")).strip())]
if unnamed:
    print(f"UNNAMED: {len(unnamed)} of {len(comps)} components carry no name")
    sys.exit(6)
names = ",".join(str(c.get("name")) for c in comps)
print(f"OK|{len(comps)}|{names[:120]}")
' 2>&1)"
PARSE_RC=$?

# The analyzer must not be able to bluff by CRASHING. Its findings are exits
# 2-6, deliberately starting at 2: a Python traceback exits 1, and an earlier
# revision of this guard treated that 1 as "defect found" — so a SyntaxError in
# the analyzer read as RED-confirmed against every input, including a healthy
# fixture. Caught by the golden-good fixture in step 3 of this guard's own
# validation, which is exactly why §11.4.107(10) requires one. Any exit outside
# {0} ∪ {2..6} means the instrument broke, and a broken instrument reports
# nothing about the subject — hard FAIL in BOTH polarities.
if [ "$PARSE_RC" -ne 0 ] && { [ "$PARSE_RC" -lt 2 ] || [ "$PARSE_RC" -gt 6 ]; }; then
    fail "ANALYZER CRASHED (exit $PARSE_RC) — this is a defect in the guard, \
not a verdict about the endpoint. Output: $VERDICT"
fi

if [ "$RED_MODE" = "1" ]; then
    [ "$PARSE_RC" -ne 0 ] || fail \
      "RED baseline did NOT reproduce: the health report names real components \
($VERDICT). The fix has landed — flip this guard to RED_MODE=0 and register it \
in the standing suite."
    echo "RED confirmed (HXC-244): $VERDICT"
    echo "  The endpoint reports a verdict it never earned. Body: ${BODY:0:160}"
    exit 0
fi

[ "$PARSE_RC" -eq 0 ] || fail "$VERDICT — body: ${BODY:0:200}"

COUNT="$(printf '%s' "$VERDICT" | cut -d'|' -f2)"
NAMES="$(printf '%s' "$VERDICT" | cut -d'|' -f3)"
echo "GREEN (HXC-244): health report names $COUNT checked component(s): $NAMES"
