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
set -uo pipefail

URL="${HXC233_URL:-https://localhost:8443/v1/chat/completions}"
MODEL="${HXC233_MODEL:-qwen}"
TIMEOUT="${HXC233_TIMEOUT:-90}"
RED_MODE="${RED_MODE:-0}"
fail() { echo "GUARD FAILED (HXC-233): $*" >&2; exit 1; }

BODY="$(curl -sk -m "$TIMEOUT" -X POST "$URL" \
  -H 'Content-Type: application/json' \
  -d "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"What is 2+2? Reply with just the number.\"}],\"max_tokens\":32}" \
  2>/dev/null)"
CURL_RC=$?   # captured directly off curl — never after a pipe (§11.4.6)

if [ "$CURL_RC" -ne 0 ] || [ -z "$BODY" ]; then
    [ "$RED_MODE" = "1" ] && { echo "RED confirmed: completion endpoint unreachable (curl rc=$CURL_RC)"; exit 0; }
    fail "endpoint unreachable (curl rc=$CURL_RC). The gateway may be down; \
this guard asserts a LIVE completion, so an unreachable endpoint is a FAIL, \
never a silent skip."
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
