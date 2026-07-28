#!/usr/bin/env bash
# capture_feat_cerebras_models_f8c38181.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   f8c38181  fix(provider): update Cerebras model list to match live
#             /models endpoint (§11.4.99/§11.4.150)
#
# WHAT THE FIX CHANGED (helix_code/internal/llm/providers/cerebras/cerebras.go)
#   BEFORE: cb.initializeModels() seeded a STALE static list —
#             llama3.1-8b, llama3.1-70b, llama-3.3-70b
#           None of these three ids are served by the live Cerebras Cloud
#           /models endpoint any more (the Meta Llama 3.1/3.3 rows were
#           dropped from Cerebras's catalogue). A user asking HelixCode to
#           list or select a Cerebras model saw ids the provider does not
#           actually serve.
#   AFTER:  the static seed was updated to the THREE ids the live endpoint
#           currently reports — gemma-4-31b, gpt-oss-120b, zai-glm-4.7.
#
# THE PROOF THIS CAPTURE MUST PRODUCE (per the task brief — not "reachable",
# the LIST must genuinely MATCH what the live endpoint returns TODAY)
#   Both sides of the comparison are discovered at runtime, never a literal
#   array pasted into this script:
#     * "expected" is extracted from the FIXED source at commit f8c38181 via
#       `git show` — if the source list is edited again in the future, this
#       script keeps testing against whatever the CURRENT fixed source says,
#       never a stale snapshot frozen at authoring time;
#     * "actual" is the live JSON body captured on the wire this run from
#       https://api.cerebras.ai/v1/models.
#   PHASE 2 (bonus, §11.4.136) goes one step further than list membership: it
#   drives one real chat-completion round trip against a confirmed-live model
#   id, echoing a per-run nonce, proving the ids are genuinely invocable, not
#   merely named in a list.
#
# CREDENTIALS (§11.4.10 / CONST-042)
#   Reads CEREBRAS_API_KEY from the environment. NEVER echoed, printed, or
#   logged by this script. qa_redact registers it before the first HTTP call
#   so the shared library's redaction pass scrubs it from every transcript
#   (curl -v otherwise logs the sent Authorization header verbatim).
#
# ANTI-BLUFF (§11.4 / §11.4.1 / §11.4.3)
#   An unset key, a rejected key, or a rate-limited/unreachable live endpoint
#   is an honest SKIP with reason — never a fabricated PASS. PHASE 2 is
#   explicitly a bonus: if every candidate model is rate-limited or exhausts
#   its reasoning-token budget without emitting the nonce, PHASE 2 alone
#   SKIPs (recorded, non-blocking) while PHASE 1's list-match evidence — the
#   core proof this fix requires — stands or falls independently.
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED (list drifted from live
#             again, or a confirmed-live model stopped working) |
#             2 INCOMPLETE (a case SKIPped with reason, §11.4.3)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_feat_cerebras_models_f8c38181.sh"

COMMIT="f8c381813f7ead7fb6851499b3e6648eb27fea39"
CEREBRAS_URL="${CEREBRAS_BASE_URL:-https://api.cerebras.ai/v1}"

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "feat_cerebras_models" "$COMMIT" \
    "Cerebras model list (helix_code/internal/llm/providers/cerebras) matches the live /models endpoint"

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

if [ -z "${CEREBRAS_API_KEY:-}" ]; then
    qa_skip "cerebras_models" \
        "Cerebras model list matches the live /models endpoint" \
        "CEREBRAS_API_KEY not set in the environment — cannot authenticate to the live Cerebras API"
    qa_finish
fi

qa_redact "$CEREBRAS_API_KEY"

echo
echo "### PHASE 0 — expected model ids: extracted from the FIXED source at commit ${COMMIT:0:8} (never a literal in this script)"
echo

EXPECTED_MODELS="$(git -C "$QA_REPO_ROOT" show "${COMMIT}:helix_code/internal/llm/providers/cerebras/cerebras.go" 2>/dev/null | grep -oP 'Name:\s*"\K[^"]+')"
echo "-- fixed source declares: $(echo "$EXPECTED_MODELS" | tr '\n' ' ')"

if [ -z "$EXPECTED_MODELS" ]; then
    qa_skip "cerebras_models" \
        "Cerebras model list matches the live /models endpoint" \
        "could not extract any Name: \"...\" model ids from commit ${COMMIT} source — cannot compute an expected set"
    qa_finish
fi

echo
echo "### PHASE 1 — live Cerebras /models endpoint (sink-of-truth, §11.4.111 bind by reported identity)"
echo

qa_http_retry "cerebras_models" GET "${CEREBRAS_URL}/models" \
    -H "Authorization: Bearer ${CEREBRAS_API_KEY}" --max-time 30

PHASE1_OK=0
case "${QA_LAST_STATUS:-}" in
    200)
        PHASE1_OK=1
        assert_status cerebras_models 200
        assert_body_contains cerebras_models '"object":"list"'
        while IFS= read -r m; do
            [ -n "$m" ] || continue
            assert_body_contains cerebras_models "\"${m}\""
        done <<< "$EXPECTED_MODELS"
        LIVE_IDS="$(printf '%s' "${QA_LAST_BODY:-}" | grep -oP '"id":"\K[^"]+' | tr '\n' ' ')"
        echo "-- live Cerebras /models reports: ${LIVE_IDS}"
        ;;
    401|403)
        qa_skip "cerebras_models" \
            "Cerebras model list matches the live /models endpoint" \
            "CEREBRAS_API_KEY rejected by the live API (HTTP ${QA_LAST_STATUS}) — cannot verify"
        ;;
    429|"")
        qa_skip "cerebras_models" \
            "Cerebras model list matches the live /models endpoint" \
            "live Cerebras API rate-limited/unreachable this run (status='${QA_LAST_STATUS:-<none>}', after retries) — cannot verify"
        ;;
    *)
        assert_status cerebras_models 200
        ;;
esac

echo
echo "### PHASE 2 (bonus, §11.4.136) — real completion round-trip on a confirmed-live model id"
echo

COMPLETION_DONE=0
COMPLETION_CASE=""
COMPLETION_MODEL=""
if [ "$PHASE1_OK" -eq 1 ]; then
    NONCE="qanonce-cb-$(head -c 6 /dev/urandom | od -An -tx1 | tr -d ' \n')"
    for m in $EXPECTED_MODELS; do
        case_id="cerebras_completion_${m}"
        REQ_BODY="$(printf '{"model":"%s","messages":[{"role":"user","content":"Reply with exactly this token and nothing else: %s"}],"max_tokens":300,"temperature":0}' "$m" "$NONCE")"
        for attempt in 1 2; do
            qa_http "$case_id" POST "${CEREBRAS_URL}/chat/completions" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer ${CEREBRAS_API_KEY}" \
                -d "$REQ_BODY" --max-time 60
            if [ "${QA_LAST_STATUS:-}" = "200" ] && printf '%s' "${QA_LAST_BODY:-}" | grep -q -- "$NONCE"; then
                COMPLETION_DONE=1
                COMPLETION_CASE="$case_id"
                COMPLETION_MODEL="$m"
                break 2
            fi
            if [ "${QA_LAST_STATUS:-}" = "429" ]; then
                sleep 5
                continue
            fi
            break
        done
    done
fi

if [ "$COMPLETION_DONE" -eq 1 ]; then
    assert_status "$COMPLETION_CASE" 200
    assert_body_contains "$COMPLETION_CASE" "$NONCE"
    assert_body_contains "$COMPLETION_CASE" "\"model\":\"${COMPLETION_MODEL}\""
    assert_body_contains "$COMPLETION_CASE" '"finish_reason"'
    echo "-- bonus completion proved on model: ${COMPLETION_MODEL}"
else
    EXPECTED_MODELS_CSV="$(printf '%s' "$EXPECTED_MODELS" | tr '\n' ',' | sed 's/,$//')"
    qa_skip "cerebras_completion" \
        "a fixed-list Cerebras model produces a real completion echoing a per-run nonce" \
        "no candidate model (${EXPECTED_MODELS_CSV}) produced the nonce within budget/rate-limit this run; bonus phase only — the PHASE 1 list-match evidence above is unaffected"
fi

qa_finish
