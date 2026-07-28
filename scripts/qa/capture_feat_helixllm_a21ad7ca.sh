#!/usr/bin/env bash
# capture_feat_helixllm_a21ad7ca.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   a21ad7ca  feat(server): route HelixCode LLM to local HelixLLM coder
#             (:18434) — §11.4.115
#
# WHAT THE FEATURE SHIPS (helix_code/internal/server/llm_generate.go)
#   resolveLLMProvider gained a real branch: when the request (or
#   HELIX_LLM_PROVIDER) names "helixllm" or "local" (case-insensitive), it
#   returns a real *llm.OpenAICompatibleProvider pointed at
#   HELIX_LLM_LOCAL_OPENAI_ENDPOINT (default http://localhost:18434) — the
#   in-repo llama.cpp OpenAI-compatible coder sidecar. Before this commit that
#   name resolved to NOTHING: routing reached only Ollama (:11434) or the four
#   cloud backends, and resolveLLMProvider("helixllm") returned
#   errUnknownProvider — `unknown provider: "helixllm"`.
#
# THE RED SIGNATURE THIS GUARDS (§11.4.115)
#   Pre-fix, a helixllm-routed generation FAILED with `unknown provider`.
#   Post-fix it must produce a real completion. So the sharpest, most
#   falsifiable assertion is not "200" but the conjunction:
#     (a) the response does NOT carry the `unknown provider` error signature,
#     (b) the response echoes a per-run NONCE minted milliseconds earlier —
#         proving a real, prompt-conditioned generation rather than a canned
#         or cached body, and
#     (c) the model the response names is the model the CODER ITSELF advertises
#         — read live from the coder in PHASE 0 and matched, not hardcoded.
#
#   (c) is the routing proof and the sink-side binding (§11.4.13): the facade
#   could only echo the coder's own model id if the request genuinely reached
#   :18434. Had it fallen through to Ollama, the model id would be Ollama's.
#   The expected value is DISCOVERED at runtime from the coder's /v1/models —
#   never a literal in this file — so the assertion cannot drift out of sync
#   with whichever model the coder is actually serving (§11.4.111: bind by the
#   identity the sink reports, not by a value we assumed).
#
# WHY AN EPHEMERAL INSTANCE (§11.4.6 honest scope)
#   The already-running server on :8081 has neither HELIX_LLM_PROVIDER=helixllm
#   nor a wire-facade key configured, so this route is not exercisable there.
#   PHASE 1 boots a purpose-configured instance on a free loopback port. If the
#   coder is down or the instance cannot boot, PHASE 1 SKIPs with reason
#   (§11.4.3) and the run exits 2 (INCOMPLETE) — never a PASS. This capture
#   supplies the HTTP-level round-trip the commit message itself recorded as
#   "deferred" (it captured provider-level evidence instead).
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED (routing broken/regressed) |
#             2 INCOMPLETE (a case SKIPped with reason, §11.4.3)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_feat_helixllm_a21ad7ca.sh"

COMMIT="a21ad7ca4bdb8c17dd68d6065bc4ff8174a0b15d"
CODER_URL="${HELIX_LLM_LOCAL_OPENAI_ENDPOINT:-http://127.0.0.1:18434}"

# Per-run nonce: minted now, so no cached or canned body can contain it.
NONCE="qanonce$(head -c 6 /dev/urandom | od -An -tx1 | tr -d ' \n')"

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "feat_helixllm_route" "$COMMIT" \
    "HelixCode LLM routes to the local HelixLLM coder (:18434): real completion, coder-advertised model, no 'unknown provider'"

trap qa_stop_ephemeral_server EXIT

echo
echo "### PHASE 0 — sink identity: what does the coder itself advertise? (${CODER_URL})"
echo

qa_http "coder_models" GET "${CODER_URL}/v1/models" --max-time 20

CODER_UP=0
CODER_MODEL=""
if [ "${QA_LAST_STATUS:-}" = "200" ]; then
    CODER_UP=1
    assert_status        coder_models 200
    assert_body_contains coder_models '"object":"list"'
    # Discover the served model id from the coder's OWN response. Never a
    # literal in this file (§11.4.111 — bind by reported identity).
    CODER_MODEL="$(printf '%s' "${QA_LAST_BODY:-}" \
        | tr ',' '\n' | grep -m1 '"id"[[:space:]]*:' \
        | sed -E 's/.*"id"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')"
    echo "-- coder advertises model id: ${CODER_MODEL:-<could not parse>}"
else
    echo "-- coder not reachable (status=${QA_LAST_STATUS:-<none>})"
fi

echo
echo "### PHASE 1 — helixllm-routed generation through an ephemeral instance"
echo

EPH_KEY="qa-ephemeral-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
qa_redact "$EPH_KEY"

# Ask for the nonce back verbatim — a real generation conditioned on THIS run.
GEN_BODY="$(printf '{"model":"default","max_tokens":32,"messages":[{"role":"user","content":"Reply with exactly this token and nothing else: %s"}]}' "$NONCE")"

if [ "$CODER_UP" -eq 1 ] && [ -n "$CODER_MODEL" ] && qa_boot_ephemeral_server \
        HELIX_WIRE_FACADE_API_KEYS="$EPH_KEY" \
        HELIX_LLM_PROVIDER="helixllm" \
        HELIX_LLM_LOCAL_OPENAI_ENDPOINT="$CODER_URL" \
        HELIX_AUTH_JWT_SECRET="${HELIX_AUTH_JWT_SECRET:-}" \
        HELIX_DATABASE_PASSWORD="${HELIX_DATABASE_PASSWORD:-}"
then
    EPH="http://127.0.0.1:${QA_EPH_PORT}"

    qa_http "eph_helixllm_generate" POST "${EPH}/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${EPH_KEY}" \
        -d "$GEN_BODY" --max-time 180
    assert_status eph_helixllm_generate 200

    # (a) THE PRE-FIX RED SIGNATURE must be gone. This is the exact string the
    #     old errUnknownProvider path produced for this provider name.
    assert_body_not_contains eph_helixllm_generate "unknown provider"

    # (b) REAL, prompt-conditioned generation: the freshly minted nonce comes
    #     back. No cached, canned, or replayed body can contain it.
    assert_body_contains eph_helixllm_generate "$NONCE"

    # (c) ROUTING / SINK BINDING: the model named in the response is the model
    #     the coder advertised in PHASE 0 — so the request really reached
    #     :18434 and did not fall through to Ollama or a cloud backend.
    assert_body_contains eph_helixllm_generate "$CODER_MODEL"

    # Real usage accounting — a translated llm.LLMResponse, not an empty stub.
    assert_body_contains eph_helixllm_generate '"completion_tokens"'
    assert_body_contains eph_helixllm_generate '"finish_reason"'

    qa_stop_ephemeral_server
else
    if [ "$CODER_UP" -eq 0 ]; then
        SKIP_WHY="HelixLLM coder unreachable at ${CODER_URL} — the route's destination is down, so the round-trip cannot be honestly asserted"
    elif [ -z "$CODER_MODEL" ]; then
        SKIP_WHY="could not parse a model id from the coder's /v1/models response — the sink-binding assertion has no discovered expected value"
    else
        SKIP_WHY="${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
    fi
    qa_skip "eph_helixllm_generate" \
        "helixllm route produces a real completion echoing a per-run nonce, naming the coder-advertised model, with no 'unknown provider'" \
        "$SKIP_WHY"
fi

qa_finish
