#!/usr/bin/env bash
# capture_feat_cohere_v2migrate_67c9a9bc.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   67c9a9bc  fix(provider): Cohere v1->v2 endpoint + model default update
#             (§11.4.99/§11.4.150)
#
# WHAT THE FIX CHANGED (helix_code/internal/llm/providers/cohere/client.go)
#   BEFORE: CohereBaseURL = https://api.cohere.com/v1/chat
#           default model = command-r-plus
#           request body   = singular {model,message,chat_history,...}
#           response parse = top-level {text, finish_reason}
#   AFTER:  CohereBaseURL = https://api.cohere.com/v2/chat
#           default model = command-r-08-2024
#           request body   = {model, messages:[{role,content}], ...}
#           response parse = message.content[] where type=="text"
#
# WHAT THIS CAPTURE VERIFIES ON THE LIVE COHERE API (§11.4.99 deep
# verification against the live service — not just restating the commit
# message's own prose)
#   Exploratory live probing done while authoring this script (all four
#   combinations of {v1,v2} x {old model, new model}) established a slightly
#   SHARPER root cause than "the v1 endpoint was retired":
#     * v1/chat + command-r-08-2024 (new model, OLD wire shape) -> HTTP 200.
#       The v1 ENDPOINT ITSELF still answers for a currently-supported model.
#     * v1/chat + command-r-plus    (old model, OLD wire shape) -> HTTP 404,
#       body: "model 'command-r-plus' was removed on September 15, 2025."
#     * v2/chat + command-r-plus    (old model, NEW wire shape) -> HTTP 404,
#       the SAME "model ... was removed" error.
#     * v2/chat + command-r-08-2024 (new model, NEW wire shape) -> HTTP 200,
#       a genuine v2-shaped completion (message.content[].type=="text").
#   So the 404 the pre-fix code hit is fundamentally a RETIRED-MODEL error,
#   not a dead endpoint in the strictest sense — but that distinction is moot
#   for HelixCode's users: the OLD code's EXACT request (v1 shape + the old
#   command-r-plus default) is broken regardless of which half of it Cohere
#   blames, and the NEW code's EXACT request (v2 shape + the new
#   command-r-08-2024 default) works. PHASE 1/2 below reproduce those two
#   EXACT request shapes byte-for-byte against the live API (mirroring the
#   pre-fix cohereRequest / post-fix v2Request Go structs, field for field).
#   PHASE 3 isolates endpoint-vs-model as an honest, non-bluffed supporting
#   fact (§11.4.6) instead of silently repeating the commit's own summary.
#
# CREDENTIALS (§11.4.10 / CONST-042)
#   Reads COHERE_API_KEY from the environment. NEVER echoed, printed, or
#   logged by this script. qa_redact scrubs it from every transcript.
#
# ANTI-BLUFF (§11.4 / §11.4.1 / §11.4.3)
#   An unset/rejected key or a rate-limited/unreachable live API is an honest
#   SKIP with reason for the affected case(s) — never a fabricated PASS.
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED (migration regressed, or the
#             pre-fix path unexpectedly started working) |
#             2 INCOMPLETE (a case SKIPped with reason, §11.4.3)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_feat_cohere_v2migrate_67c9a9bc.sh"

COMMIT="67c9a9bcdae90caafe7980c79ebadb96aede5c23"
PARENT_COMMIT="${COMMIT}^"

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "feat_cohere_v2migrate" "$COMMIT" \
    "Cohere provider (helix_code/internal/llm/providers/cohere) genuinely uses the v2 endpoint; v1+retired-model path confirmed broken"

# Small local retry helper for transient (429/5xx/curl-error) responses.
# Wraps qa_http ONLY in this script — the shared library is unmodified.
# Genuine functional responses (2xx, or a definitive non-transient error) are
# returned immediately without retrying.
qa_http_retry() {
    local case_id="$1"; shift
    local attempt
    for attempt in 1 2 3; do
        qa_http "$case_id" "$@"
        case "${QA_LAST_STATUS:-}" in
            429|500|502|503|504|"")
                [ "$attempt" -lt 3 ] && sleep $((attempt * 3))
                continue
                ;;
            *) return 0 ;;
        esac
    done
}

if [ -z "${COHERE_API_KEY:-}" ]; then
    qa_skip "cohere_v2_generate" \
        "Cohere v2 endpoint genuinely used and works" \
        "COHERE_API_KEY not set in the environment — cannot authenticate to the live Cohere API"
    qa_finish
fi

qa_redact "$COHERE_API_KEY"

echo
echo "### PHASE 0 — endpoint + default-model constants extracted from source (never literals in this script)"
echo

NEW_BASE_URL="$(git -C "$QA_REPO_ROOT" show "${COMMIT}:helix_code/internal/llm/providers/cohere/client.go" 2>/dev/null | grep -oP 'CohereBaseURL = "\K[^"]+')"
NEW_DEFAULT_MODEL="$(git -C "$QA_REPO_ROOT" show "${COMMIT}:helix_code/internal/llm/providers/cohere/client.go" 2>/dev/null | grep -oP 'DefaultModel\s*=\s*"\K[^"]+')"
OLD_BASE_URL="$(git -C "$QA_REPO_ROOT" show "${PARENT_COMMIT}:helix_code/internal/llm/providers/cohere/client.go" 2>/dev/null | grep -oP 'CohereBaseURL = "\K[^"]+')"
OLD_DEFAULT_MODEL="$(git -C "$QA_REPO_ROOT" show "${PARENT_COMMIT}:helix_code/internal/llm/providers/cohere/client.go" 2>/dev/null | grep -oP '\bbody\.Model = "\K[^"]+')"

echo "-- post-fix : base=${NEW_BASE_URL:-<extract-failed>} default-model=${NEW_DEFAULT_MODEL:-<extract-failed>}"
echo "-- pre-fix  : base=${OLD_BASE_URL:-<extract-failed>} default-model=${OLD_DEFAULT_MODEL:-<extract-failed>}"

if [ -z "$NEW_BASE_URL" ] || [ -z "$NEW_DEFAULT_MODEL" ] || [ -z "$OLD_BASE_URL" ] || [ -z "$OLD_DEFAULT_MODEL" ]; then
    qa_skip "cohere_v2_generate" \
        "Cohere v2 endpoint genuinely used and works" \
        "could not extract one or more of {new,old} x {base-url,default-model} from commit source — cannot construct the RED/GREEN request pair"
    qa_finish
fi

NONCE="qanonce-co-$(head -c 6 /dev/urandom | od -An -tx1 | tr -d ' \n')"

echo
echo "### PHASE 1 (RED) — the PRE-FIX code's exact request: old wire shape + old base URL + old default model"
echo

RED_BODY="$(printf '{"model":"%s","message":"Reply with exactly this token and nothing else: %s","max_tokens":32,"temperature":0.7}' "$OLD_DEFAULT_MODEL" "$NONCE")"
qa_http_retry "cohere_v1_red" POST "$OLD_BASE_URL" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${COHERE_API_KEY}" \
    -d "$RED_BODY" --max-time 30

case "${QA_LAST_STATUS:-}" in
    401|403)
        qa_skip "cohere_v1_red" \
            "pre-fix request (v1 + ${OLD_DEFAULT_MODEL}) is broken (reproduces the pre-fix defect)" \
            "COHERE_API_KEY rejected by the live API (HTTP ${QA_LAST_STATUS}) — cannot verify"
        ;;
    429|"")
        qa_skip "cohere_v1_red" \
            "pre-fix request (v1 + ${OLD_DEFAULT_MODEL}) is broken (reproduces the pre-fix defect)" \
            "live Cohere API rate-limited/unreachable this run (status='${QA_LAST_STATUS:-<none>}', after retries) — cannot verify"
        ;;
    *)
        # The proof: the OLD code's EXACT request must NOT succeed.
        assert_status_not cohere_v1_red 200
        assert_body_contains cohere_v1_red "$OLD_DEFAULT_MODEL"
        ;;
esac

echo
echo "### PHASE 2 (GREEN) — the POST-FIX code's exact request: v2 messages-array shape + v2 base URL + new default model"
echo

GREEN_BODY="$(printf '{"model":"%s","messages":[{"role":"user","content":"Reply with exactly this token and nothing else: %s"}],"max_tokens":32,"temperature":0.7}' "$NEW_DEFAULT_MODEL" "$NONCE")"
qa_http_retry "cohere_v2_generate" POST "$NEW_BASE_URL" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${COHERE_API_KEY}" \
    -d "$GREEN_BODY" --max-time 30

case "${QA_LAST_STATUS:-}" in
    200)
        assert_status cohere_v2_generate 200
        # Real, prompt-conditioned round trip: the freshly minted nonce comes
        # back. No cached/canned/replayed body can contain it.
        assert_body_contains cohere_v2_generate "$NONCE"
        # Genuine v2 nested response shape (message.content[].type=="text"),
        # exactly the field the FIXED parser reads — differentiates from the
        # old v1 flat {text,finish_reason} shape.
        assert_body_contains cohere_v2_generate '"message":{"role":"assistant"'
        assert_body_contains cohere_v2_generate '"type":"text"'
        assert_body_contains cohere_v2_generate '"finish_reason"'
        ;;
    401|403)
        qa_skip "cohere_v2_generate" \
            "post-fix request (v2 + ${NEW_DEFAULT_MODEL}) produces a real completion echoing a per-run nonce" \
            "COHERE_API_KEY rejected by the live API (HTTP ${QA_LAST_STATUS}) — cannot verify"
        ;;
    429|"")
        qa_skip "cohere_v2_generate" \
            "post-fix request (v2 + ${NEW_DEFAULT_MODEL}) produces a real completion echoing a per-run nonce" \
            "live Cohere API rate-limited/unreachable this run (status='${QA_LAST_STATUS:-<none>}', after retries) — cannot verify"
        ;;
    *)
        # Genuine functional failure (e.g. a definitive non-2xx that survived
        # retries) — record as a hard FAIL: the migrated path did not work.
        assert_status cohere_v2_generate 200
        ;;
esac

echo
echo "### PHASE 3 (isolation, honest supporting evidence §11.4.6) — endpoint-vs-model, cross-matrix"
echo

qa_http_retry "cohere_v1_isolation_newmodel" POST "$OLD_BASE_URL" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${COHERE_API_KEY}" \
    -d "$(printf '{"model":"%s","message":"ping"}' "$NEW_DEFAULT_MODEL")" --max-time 30
case "${QA_LAST_STATUS:-}" in
    429|401|403|"")
        qa_skip "cohere_v1_isolation_newmodel" \
            "v1 endpoint alone (new model) still answers — isolates endpoint-vs-model as the pre-fix root cause" \
            "live Cohere API rate-limited/unauthorized/unreachable this run (status='${QA_LAST_STATUS:-<none>}') — cannot verify"
        ;;
    *)
        assert_status cohere_v1_isolation_newmodel 200
        ;;
esac

qa_http_retry "cohere_v2_isolation_oldmodel" POST "$NEW_BASE_URL" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${COHERE_API_KEY}" \
    -d "$(printf '{"model":"%s","messages":[{"role":"user","content":"ping"}],"max_tokens":8}' "$OLD_DEFAULT_MODEL")" --max-time 30
case "${QA_LAST_STATUS:-}" in
    429|401|403|"")
        qa_skip "cohere_v2_isolation_oldmodel" \
            "v2 endpoint alone (old model) still rejects — isolates endpoint-vs-model as the pre-fix root cause" \
            "live Cohere API rate-limited/unauthorized/unreachable this run (status='${QA_LAST_STATUS:-<none>}') — cannot verify"
        ;;
    *)
        assert_status_not cohere_v2_isolation_oldmodel 200
        assert_body_contains cohere_v2_isolation_oldmodel "$OLD_DEFAULT_MODEL"
        ;;
esac

qa_finish
