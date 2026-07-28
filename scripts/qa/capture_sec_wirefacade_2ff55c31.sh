#!/usr/bin/env bash
# capture_sec_wirefacade_2ff55c31.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   2ff55c31  fix(security): auth-gate the wire facade
#             (/v1/chat/completions, /v1/messages) — unauth->401
#
# WHAT THE FIX CHANGED (helix_code/internal/server/server.go)
#   BEFORE: s.router.POST("/v1/chat/completions", s.chatCompletions)
#           s.router.POST("/v1/messages",         s.anthropicMessages)
#           No authentication at all — while every shipped config profile binds
#           server.address to 0.0.0.0. An unauthenticated, real-provider,
#           token-consuming surface reachable on any interface. The commit's own
#           RED evidence: an unauthenticated request fell straight through to a
#           real provider.Generate call (observed dialing localhost:11434).
#   AFTER:  both routes carry s.wireFacadeAuthMiddleware() — Bearer / x-api-key
#           checked against cfg.Auth.WireFacadeAPIKeys
#           (HELIX_WIRE_FACADE_API_KEYS). Deliberately FAIL-CLOSED: an empty key
#           list rejects EVERY request, even one carrying a bearer token (unlike
#           the DZ-05 ancestor's "empty keys => open access" convenience mode),
#           because these routes drive real paid-provider calls.
#
# THE ASSERTION PAIR
#   NEGATIVE (the security proof — this is exactly what passed before the fix):
#     POST /v1/chat/completions and POST /v1/messages with NO credential must
#     return 401, and the response body must NOT contain a provider completion
#     (no "choices" / no "content" success shape) — proving the request was
#     stopped at the middleware and never reached a paid provider.
#   FAIL-CLOSED NEGATIVE:
#     With no keys configured, even a request carrying a bearer token is 401.
#   POSITIVE (the fix did not simply break the facade):
#     On an instance with a key configured, a request presenting that key via
#     `Authorization: Bearer` or `x-api-key` must NOT be rejected 401 — the
#     middleware admits it and hands off to the handler.
#
# WHY THE POSITIVE ASSERTION IS "NOT 401" AND NOT "200" (§11.4.6)
#   The security property under test is the AUTH DECISION, not provider
#   availability. Asserting 200 would make this security capture depend on a
#   live LLM provider and a valid model name; a provider timeout would then be
#   reported as a security failure, which would be false. "Not 401, and not the
#   authentication_error body shape" is the precise, honest boundary of what
#   this fix guarantees. The downstream status actually observed is recorded
#   verbatim in the transcript either way.
#
# HONEST SCOPE NOTE (§11.4.6)
#   The already-running server (localhost:8081) has HELIX_WIRE_FACADE_API_KEYS
#   UNSET, so it is fail-closed: every negative case is provable there, but no
#   key exists that could exercise the allow-path. The positive case runs on a
#   purpose-configured ephemeral instance on a free loopback port. If it cannot
#   boot, the positive case is SKIP-with-reason (§11.4.3) and the run exits 2
#   (INCOMPLETE) — never a PASS.
#
# EXIT CODES: 0 all PASS | 1 a security assertion FAILED | 2 INCOMPLETE (SKIP)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_sec_wirefacade_2ff55c31.sh"

COMMIT="2ff55c314182e8f51fd2ff30fcb2104a53e91198"
QA_BASE_URL="${QA_BASE_URL:-http://localhost:8081}"

OPENAI_BODY='{"model":"llama3.2","messages":[{"role":"user","content":"QA probe: reply with the single word OK"}]}'
ANTHROPIC_BODY='{"model":"llama3.2","max_tokens":16,"messages":[{"role":"user","content":"QA probe: reply with the single word OK"}]}'

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "sec_wirefacade" "$COMMIT" "Wire facade auth gate (/v1/chat/completions, /v1/messages): unauth -> 401"
trap qa_stop_ephemeral_server EXIT

echo
echo "### PHASE 1 — NEGATIVE cases against the LIVE server (${QA_BASE_URL})"
echo "### (fail-closed: HELIX_WIRE_FACADE_API_KEYS unset)"
echo

# N1: the pre-fix attack, OpenAI wire — unauthenticated completion request.
qa_http "live_openai_unauth" POST "${QA_BASE_URL}/v1/chat/completions" \
    -H "Content-Type: application/json" -d "$OPENAI_BODY" --max-time 25
assert_status            live_openai_unauth 401
assert_body_contains     live_openai_unauth "authentication_error"
# The load-bearing part: the request must not have reached a paid provider.
assert_body_not_contains live_openai_unauth '"choices"'
assert_body_not_contains live_openai_unauth '"usage"'

# N2: the pre-fix attack, Anthropic wire.
qa_http "live_anthropic_unauth" POST "${QA_BASE_URL}/v1/messages" \
    -H "Content-Type: application/json" -d "$ANTHROPIC_BODY" --max-time 25
assert_status            live_anthropic_unauth 401
assert_body_contains     live_anthropic_unauth "authentication_error"
assert_body_not_contains live_anthropic_unauth '"content"'
assert_body_not_contains live_anthropic_unauth '"stop_reason"'

# N3/N4: fail-closed proof — a credential IS presented, but no keys are
# configured, so it must still be rejected (not "empty keys => open access").
qa_http "live_openai_bogus_bearer" POST "${QA_BASE_URL}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer sk-qa-probe-not-a-real-key" \
    -d "$OPENAI_BODY" --max-time 25
assert_status            live_openai_bogus_bearer 401
assert_body_not_contains live_openai_bogus_bearer '"choices"'

qa_http "live_anthropic_bogus_apikey" POST "${QA_BASE_URL}/v1/messages" \
    -H "Content-Type: application/json" \
    -H "x-api-key: qa-probe-not-a-real-key" \
    -d "$ANTHROPIC_BODY" --max-time 25
assert_status            live_anthropic_bogus_apikey 401
assert_body_not_contains live_anthropic_bogus_apikey '"content"'

# N5: streaming variant — a streamed unauth request must also be refused, not
#     leak a partial SSE completion.
qa_http "live_openai_unauth_stream" POST "${QA_BASE_URL}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d "${OPENAI_BODY%\}},\"stream\":true}" --max-time 25
assert_status            live_openai_unauth_stream 401
assert_body_not_contains live_openai_unauth_stream "data:"

echo
echo "### PHASE 2 — POSITIVE case on a purpose-configured ephemeral instance"
echo

EPH_KEY="qa-ephemeral-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
qa_redact "$EPH_KEY"

if qa_boot_ephemeral_server \
        HELIX_WIRE_FACADE_API_KEYS="$EPH_KEY" \
        HELIX_AUTH_JWT_SECRET="${HELIX_AUTH_JWT_SECRET:-}" \
        HELIX_DATABASE_PASSWORD="${HELIX_DATABASE_PASSWORD:-}"
then
    EPH="http://127.0.0.1:${QA_EPH_PORT}"

    # P1: valid key via the OpenAI wire convention — middleware must admit it.
    qa_http "eph_openai_valid_bearer" POST "${EPH}/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${EPH_KEY}" \
        -d "$OPENAI_BODY" --max-time 60
    assert_status_not        eph_openai_valid_bearer 401
    assert_body_not_contains eph_openai_valid_bearer "authentication_error"

    # P2: valid key via the Anthropic wire convention.
    qa_http "eph_anthropic_valid_apikey" POST "${EPH}/v1/messages" \
        -H "Content-Type: application/json" \
        -H "x-api-key: ${EPH_KEY}" \
        -d "$ANTHROPIC_BODY" --max-time 60
    assert_status_not        eph_anthropic_valid_apikey 401
    assert_body_not_contains eph_anthropic_valid_apikey "authentication_error"

    # N6: on the SAME configured instance, a wrong key is still refused — the
    #     sharpest negative (the key list is live and non-empty, yet this fails).
    qa_http "eph_openai_wrong_key" POST "${EPH}/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer wrong-key-qa-probe" \
        -d "$OPENAI_BODY" --max-time 25
    assert_status            eph_openai_wrong_key 401
    assert_body_not_contains eph_openai_wrong_key '"choices"'

    # N7: and no credential at all is still refused.
    qa_http "eph_openai_no_key" POST "${EPH}/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -d "$OPENAI_BODY" --max-time 25
    assert_status            eph_openai_no_key 401
    assert_body_not_contains eph_openai_no_key '"choices"'

    qa_stop_ephemeral_server
else
    qa_skip "eph_openai_valid_bearer" \
        "POSITIVE: valid key via Authorization: Bearer is admitted (not 401)" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
    qa_skip "eph_anthropic_valid_apikey" \
        "POSITIVE: valid key via x-api-key is admitted (not 401)" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
    qa_skip "eph_openai_wrong_key" \
        "NEGATIVE-with-keys-configured: a wrong key is still refused 401" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
fi

qa_finish
