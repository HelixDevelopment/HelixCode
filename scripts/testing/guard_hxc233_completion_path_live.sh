#!/usr/bin/env bash
# HXC-233 standing guard (§11.4.135) — the gateway's completion path must return
# a REAL model answer, proven end-to-end through the public endpoint.
#
# WHY THIS GUARD EXISTS
# ---------------------
# For an indeterminate period the gateway returned HTTP 500 on EVERY completion
# ("all providers exhausted"). With no cloud keys configured the local provider
# is the only completion path, so the product's primary capability was dead —
# and no test noticed, because nothing exercised the path end-to-end.
#
# The cause was routing, not capability: HELIX_LLM_LOCAL_RPC_PORT defaults to
# 50052 (internal/shared/config/config.go:49), nothing on this host listens
# there, and the real model served :18434 the whole time. The fix is two
# Environment= lines in the unit — exactly the kind of config that a re-render,
# a stale template, or an operator .env can silently undo.
#
# WHY IT ASSERTS THE ANSWER, NOT THE STATUS CODE
# ----------------------------------------------
# HTTP 200 is not proof. The HXC-243 bank audit found this same service class
# answering 200 with a JSON-RPC ERROR ENVELOPE, and 72 assertion-free steps
# scored PASS against a service returning errors. A status-only assertion here
# would pass on an empty choices array, an error body, or a stub. This guard
# requires a non-empty assistant message AND a model identifier — the shape only
# a real generation produces.
#
# ARITHMETIC AS THE ORACLE
# ------------------------
# The probe asks a question with exactly one correct answer ("2+2") and requires
# "4" in the reply. That is deliberately weak as a quality measure and strong as
# a liveness measure: a stub, a canned string, or a truncated stream will not
# contain it, while any genuinely-generating model will. This guard proves the
# path is ALIVE — not that the model is good (§11.4.6).
#
# POLARITY (§11.4.115): RED_MODE=1 asserts the pre-fix shape (completions fail).
# It MUST fail on a fixed deployment.
#
# EXIT CONTRACT — ABSENT IS NOT DEFECTIVE  (§11.4.201 / §11.4.3)
# --------------------------------------------------------------
#   0  GREEN — a LIVE endpoint answered and the answer is a real generation
#   1  FAIL  — a LIVE endpoint answered and the answer is not one
#   2  SKIP  — nothing was listening; the guard certified nothing and says so
#
# The earlier revision exited 1 on an unreachable endpoint, reasoning that "an
# unreachable endpoint is a FAIL, never a silent skip". The instinct is right —
# a silent skip WOULD be a bluff — but the conclusion overshoots: a gateway that
# is not deployed has not regressed, and reporting a dead completion path on a
# host where no gateway was ever started is the §11.4.201 false-positive
# refusal. The honest report is SKIP-with-reason, which is loud, not silent:
# it prints its reason and the sweep records SKIP, never PASS.
#
# WHY THE SKIP CANNOT FAIL OPEN  (§11.4.69 CM-NO-FAIL-OPEN-SKIP)
# ---------------------------------------------------------------
# SKIP is keyed to two curl exit codes ONLY:
#     6  — could not resolve host      → nothing to connect to
#     7  — failed to connect           → nothing listening on the port
# Every other outcome means something IS there and answered badly, so it stays
# a FAIL: 28 (timeout — on a loopback target a closed port refuses instantly,
# so a timeout means a listener accepted and then wedged), 35/60 (TLS handshake
# or certificate failure — a server is present and misconfigured), 56/52
# (connection established then reset or empty — an independent review MEASURED
# 56, not the 52 an earlier revision of this comment claimed), and any HTTP
# response at all, including a 500. A dead-but-listening gateway — the precise
# HXC-233 defect — connects, so it can never reach the SKIP branch.
#
# This holds only because the request bypasses any proxy (see --noproxy below);
# without that, exit 7 could describe the proxy rather than the target.
set -uo pipefail

URL="${HXC233_URL:-https://localhost:8443/v1/chat/completions}"
MODEL="${HXC233_MODEL:-qwen}"
TIMEOUT="${HXC233_TIMEOUT:-90}"
RED_MODE="${RED_MODE:-0}"
fail() { echo "GUARD FAILED (HXC-233): $*" >&2; exit 1; }
skip_env() { echo "SKIP(env): HXC-233 — $*" >&2; exit 2; }

# --noproxy '*' is load-bearing, not decoration (independent review, F1). curl
# honours https_proxy/HTTPS_PROXY, so with a proxy variable set in the caller's
# environment — a classic CI/shell leak — a failure to reach the PROXY also
# returns exit 7. That made "rc=7 means nothing is listening at the target" a
# false statement: the reviewer pointed this guard at the LIVE, healthy gateway
# with https_proxy=http://127.0.0.1:59999 and got SKIP, seconds after the same
# guard had PASSed. Bypassing the proxy restores the invariant that rc=7 is a
# fact about the TARGET, which is the whole basis of the SKIP branch below.
# "temperature":0 pins the sampler so the arithmetic oracle is deterministic
# across runs (§11.4.50).
BODY="$(curl -sk --noproxy '*' -m "$TIMEOUT" -X POST "$URL" \
  -H 'Content-Type: application/json' \
  -d "{\"model\":\"$MODEL\",\"temperature\":0,\"messages\":[{\"role\":\"user\",\"content\":\"What is 2+2? Reply with just the number.\"}],\"max_tokens\":32}" \
  2>/dev/null)"
CURL_RC=$?   # captured directly off curl — never after a pipe (§11.4.6)

# Absence is checked FIRST and in BOTH polarities: a defect cannot be
# reproduced (RED) nor proven absent (GREEN) against a port nobody is serving.
# The previous revision let RED_MODE=1 report "RED confirmed" on an unreachable
# endpoint — a nothing-there result masquerading as a reproduced defect.
if [ "$CURL_RC" -eq 6 ] || [ "$CURL_RC" -eq 7 ]; then
    skip_env "nothing is listening at $URL (curl rc=$CURL_RC — $([ "$CURL_RC" -eq 6 ] \
&& echo 'host does not resolve' || echo 'connection refused')). The gateway is \
not deployed here; an absent service has not regressed. Start the stack to \
enforce this guard."
fi

if [ "$CURL_RC" -ne 0 ] || [ -z "$BODY" ]; then
    fail "the endpoint is REACHABLE but did not return a usable body \
(curl rc=$CURL_RC, $(printf '%s' "$BODY" | wc -c) bytes). Something accepted \
the connection and failed to answer — that is a live defect, not an absent \
service, so it is a FAIL and not a SKIP."
fi

# Parse in one pass. python3 exits non-zero on any shape violation and prints
# the reason, so a malformed/error body cannot be mistaken for a pass.
VERDICT="$(printf '%s' "$BODY" | python3 -c '
import sys, json
try:
    d = json.load(sys.stdin)
except Exception as e:
    print(f"NOT_JSON: {e}"); sys.exit(2)
if isinstance(d, dict) and d.get("error"):
    msg = d["error"].get("message") if isinstance(d["error"], dict) else d["error"]
    print(f"ERROR_ENVELOPE: {str(msg)[:160]}"); sys.exit(3)
ch = d.get("choices") or []
if not ch:
    print("NO_CHOICES: 200 response carried an empty choices array"); sys.exit(4)
content = (ch[0].get("message") or {}).get("content") or ""
if not content.strip():
    print("EMPTY_CONTENT: choices[0].message.content is blank"); sys.exit(5)
model = d.get("model") or ""
if not model:
    print("NO_MODEL: response names no model — a stub, not a generation"); sys.exit(6)
if "4" not in content:
    print(f"WRONG_ANSWER: asked 2+2, got {content.strip()[:80]!r}"); sys.exit(7)
print(f"OK|{model}|{content.strip()[:60]}")
' 2>&1)"
PARSE_RC=$?

# The analyzer must not be able to bluff by CRASHING (independent review, F4;
# §11.4.107(10)). Its findings are exits 2-7, deliberately starting at 2,
# because a Python traceback exits 1 — and treating that 1 as "defect found"
# makes a SyntaxError or a broken interpreter read as RED-confirmed against
# every input, including a healthy one. The reviewer demonstrated exactly that
# here with a python3 shim exiting 9: this guard printed "RED confirmed" and
# exited 0 against the LIVE, healthy gateway, while its HXC-244 sibling — which
# already carried this check — correctly reported a crashed analyzer. A broken
# instrument reports nothing about the subject, so it is a hard FAIL in BOTH
# polarities, never a verdict.
if [ "$PARSE_RC" -ne 0 ] && { [ "$PARSE_RC" -lt 2 ] || [ "$PARSE_RC" -gt 7 ]; }; then
    fail "ANALYZER CRASHED (exit $PARSE_RC) — this is a defect in the guard, \
not a verdict about the completion endpoint. Output: $VERDICT"
fi

if [ "$RED_MODE" = "1" ]; then
    [ "$PARSE_RC" -ne 0 ] || fail \
      "RED baseline did NOT reproduce: the endpoint returned a real answer \
($VERDICT). This deployment already carries the fix — run RED against a \
pre-fix deployment."
    echo "RED confirmed: $VERDICT"
    exit 0
fi

[ "$PARSE_RC" -eq 0 ] || fail "$VERDICT"

MODEL_ID="$(printf '%s' "$VERDICT" | cut -d'|' -f2)"
ANSWER="$(printf '%s' "$VERDICT" | cut -d'|' -f3)"
echo "GREEN (HXC-233): live completion answered \"$ANSWER\" from model \
\"$MODEL_ID\" — real generation through $URL, not a status code."
