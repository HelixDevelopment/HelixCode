#!/usr/bin/env bash
# capture_feat_wirefacade_51c058b1.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   51c058b1  feat(server): dual OpenAI + Anthropic wire facade on HelixCode's
#             own server
#
# WHAT THE FEATURE SHIPS (helix_code/internal/server/wire_facade.go)
#   Two new routes in front of HelixCode's EXISTING llm routing:
#     POST /v1/chat/completions  — OpenAI Chat Completions wire shape
#     POST /v1/messages          — Anthropic Messages wire shape
#   Both translate their wire body into the same internal llm.LLMRequest,
#   resolve a provider through the existing llmProviderResolver seam, call the
#   existing resolveDefaultModel + provider.Generate, and translate the real
#   llm.LLMResponse BACK into their own wire shape. Zero duplicated routing.
#
# THE LOAD-BEARING PROPERTY: SHAPE DIVERGENCE
#   The whole point of a DUAL facade is that the two responses are NOT the same
#   bytes. A facade that returned one shape from both routes would still answer
#   200 and still look "working", so status alone proves nothing. This capture
#   asserts the two responses carry MUTUALLY EXCLUSIVE, shape-defining fields,
#   observed on the wire in the same run against the same backend:
#
#     OpenAI  /v1/chat/completions   Anthropic  /v1/messages
#     ----------------------------   ---------------------------
#     "object":"chat.completion"     "type":"message"
#     "choices":[...]                "content":[{"type":"text"...}]
#     "finish_reason"                "stop_reason"
#     "prompt_tokens" /              "input_tokens" /
#     "completion_tokens"            "output_tokens"
#
#   Each field is asserted PRESENT in its own shape AND ABSENT from the other.
#   That cross-product is what makes this a proof of two distinct translations
#   rather than one shape emitted twice.
#
# WHY PHASE 2 NEEDS AN EPHEMERAL INSTANCE (§11.4.6 honest scope)
#   The already-running server (localhost:8081) is fail-closed:
#   HELIX_WIRE_FACADE_API_KEYS is UNSET, so every facade request is 401 before
#   reaching a handler (that auth gate is a DIFFERENT commit — 2ff55c31 — and
#   is evidenced by scripts/qa/capture_sec_wirefacade_2ff55c31.sh; it is NOT
#   re-litigated here). The translation behaviour is therefore only reachable
#   on a purpose-configured instance, which PHASE 2 boots on a free loopback
#   port. If it cannot boot, PHASE 2 is SKIP-with-reason (§11.4.3) and the run
#   exits 2 (INCOMPLETE) — never a PASS.
#
#   PHASE 2 points that instance at the in-repo HelixLLM coder (:18434) so the
#   completion is REAL — a genuine model round-trip, not a stub. If the coder
#   is unreachable the phase SKIPs rather than asserting against an error body.
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED (facade broken/regressed) |
#             2 INCOMPLETE (a case SKIPped with reason, §11.4.3)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_feat_wirefacade_51c058b1.sh"

COMMIT="51c058b1ed57bb78c84eb223feb7d7e68ca91f9e"
QA_BASE_URL="${QA_BASE_URL:-http://localhost:8081}"
CODER_URL="${HELIX_LLM_LOCAL_OPENAI_ENDPOINT:-http://127.0.0.1:18434}"

OPENAI_BODY='{"model":"default","messages":[{"role":"user","content":"Reply with the single word: OK"}],"max_tokens":24}'
ANTHROPIC_BODY='{"model":"default","max_tokens":24,"messages":[{"role":"user","content":"Reply with the single word: OK"}]}'

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "feat_wirefacade" "$COMMIT" \
    "Dual OpenAI + Anthropic wire facade: both routes ship and translate into MUTUALLY EXCLUSIVE wire shapes"

trap qa_stop_ephemeral_server EXIT

echo
echo "### PHASE 1 — both facade routes are REGISTERED on the live artifact (${QA_BASE_URL})"
echo "### (auth semantics belong to 2ff55c31 and are evidenced separately;"
echo "###  here 401-not-404 is used purely as proof the route EXISTS)"
echo

qa_http "live_openai_route_registered" POST "${QA_BASE_URL}/v1/chat/completions" \
    -H "Content-Type: application/json" -d "$OPENAI_BODY" --max-time 25
assert_status            live_openai_route_registered 401
assert_body_not_contains live_openai_route_registered "page not found"

qa_http "live_anthropic_route_registered" POST "${QA_BASE_URL}/v1/messages" \
    -H "Content-Type: application/json" -d "$ANTHROPIC_BODY" --max-time 25
assert_status            live_anthropic_route_registered 401
assert_body_not_contains live_anthropic_route_registered "page not found"

# CONTROL: an unregistered sibling under the same /v1 prefix must 404. Without
# this, a blanket 401 everywhere would make the two assertions above vacuous.
qa_http "live_unregistered_control" POST "${QA_BASE_URL}/v1/qa-probe-no-such-route" \
    -H "Content-Type: application/json" -d '{}' --max-time 15
assert_status        live_unregistered_control 404
assert_body_contains live_unregistered_control "page not found"

echo
echo "### PHASE 2 — real round-trip through both wire shapes (ephemeral instance)"
echo

# Preflight: the facade must have a real backend or the translation cannot be
# exercised honestly. Probe the coder read-only; SKIP (never fake-pass) if down.
CODER_UP=0
if curl -sS -m 6 -o /dev/null "${CODER_URL}/v1/models" 2>/dev/null; then
    CODER_UP=1
    echo "-- backend coder reachable at ${CODER_URL}"
else
    echo "-- backend coder NOT reachable at ${CODER_URL}"
fi

EPH_KEY="qa-ephemeral-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
qa_redact "$EPH_KEY"

if [ "$CODER_UP" -eq 1 ] && qa_boot_ephemeral_server \
        HELIX_WIRE_FACADE_API_KEYS="$EPH_KEY" \
        HELIX_LLM_PROVIDER="helixllm" \
        HELIX_LLM_LOCAL_OPENAI_ENDPOINT="$CODER_URL" \
        HELIX_AUTH_JWT_SECRET="${HELIX_AUTH_JWT_SECRET:-}" \
        HELIX_DATABASE_PASSWORD="${HELIX_DATABASE_PASSWORD:-}"
then
    EPH="http://127.0.0.1:${QA_EPH_PORT}"

    # ---- OpenAI wire shape -------------------------------------------------
    qa_http "eph_openai_shape" POST "${EPH}/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${EPH_KEY}" \
        -d "$OPENAI_BODY" --max-time 120
    assert_status        eph_openai_shape 200
    # Fields that define the OpenAI shape.
    assert_body_contains eph_openai_shape '"object":"chat.completion"'
    assert_body_contains eph_openai_shape '"choices"'
    assert_body_contains eph_openai_shape '"finish_reason"'
    assert_body_contains eph_openai_shape '"prompt_tokens"'
    assert_body_contains eph_openai_shape '"completion_tokens"'
    # Fields that belong ONLY to the Anthropic shape must not leak in.
    assert_body_not_contains eph_openai_shape '"stop_reason"'
    assert_body_not_contains eph_openai_shape '"input_tokens"'
    assert_body_not_contains eph_openai_shape '"output_tokens"'
    assert_body_not_contains eph_openai_shape '"type":"message"'

    # ---- Anthropic wire shape ---------------------------------------------
    qa_http "eph_anthropic_shape" POST "${EPH}/v1/messages" \
        -H "Content-Type: application/json" \
        -H "x-api-key: ${EPH_KEY}" \
        -d "$ANTHROPIC_BODY" --max-time 120
    assert_status        eph_anthropic_shape 200
    # Fields that define the Anthropic shape.
    assert_body_contains eph_anthropic_shape '"type":"message"'
    assert_body_contains eph_anthropic_shape '"content"'
    assert_body_contains eph_anthropic_shape '"type":"text"'
    assert_body_contains eph_anthropic_shape '"stop_reason"'
    assert_body_contains eph_anthropic_shape '"input_tokens"'
    assert_body_contains eph_anthropic_shape '"output_tokens"'
    # Fields that belong ONLY to the OpenAI shape must not leak in.
    assert_body_not_contains eph_anthropic_shape '"choices"'
    assert_body_not_contains eph_anthropic_shape '"finish_reason"'
    assert_body_not_contains eph_anthropic_shape '"prompt_tokens"'
    assert_body_not_contains eph_anthropic_shape '"object":"chat.completion"'

    qa_stop_ephemeral_server
else
    if [ "$CODER_UP" -eq 0 ]; then
        SKIP_WHY="backend coder unreachable at ${CODER_URL} — a real completion cannot be produced, so the translation cannot be honestly asserted"
    else
        SKIP_WHY="${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
    fi
    qa_skip "eph_openai_shape" \
        "OpenAI wire shape: object/choices/finish_reason/prompt_tokens present, Anthropic-only fields absent" \
        "$SKIP_WHY"
    qa_skip "eph_anthropic_shape" \
        "Anthropic wire shape: type:message/content/stop_reason/input_tokens present, OpenAI-only fields absent" \
        "$SKIP_WHY"
fi

qa_finish
