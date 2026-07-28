#!/usr/bin/env bash
# capture_feat_azureguard_eb233785.sh
#
# §11.4.83 retrospective end-user QA capture for the PRODUCTION sub-part of:
#   eb233785  fix(infra-defects): Azure nil-ptr panic + stale integration test
#             + e2e -all flag
#
# SCOPE OF THIS CAPTURE (§11.4.6 — stated up front, deliberately narrow)
#   eb233785 is a MIXED commit with three sub-parts. Only ONE of them ships
#   production behaviour, and only that one is evidenced here:
#     (1) EVIDENCED — internal/llm/azure_provider.go: NewAzureProvider now
#         TrimSpaces the resolved endpoint, so a whitespace-only endpoint is
#         treated as MISSING (clean error) instead of being accepted as a valid
#         resource URL and dereferenced later.
#     (2) NOT EVIDENCED — test/integration/integration_test.go reconciled to
#         the current llm.InitializeModelManager API. Test-only; changes no
#         user-visible behaviour, so there is nothing for an end-user
#         transcript to show.
#     (3) NOT EVIDENCED — a `-all` flag on the e2e challenge runner
#         (tests/e2e/challenges/cmd/runner/main.go). Test tooling, exercised
#         only by `go run`, which is out of scope for an HTTP capture.
#   This script therefore certifies sub-part (1) ONLY. It makes no claim about
#   (2) or (3), and the run's PASS must not be read as covering them.
#
# WHY THIS NEEDS NO AZURE CREDENTIALS
#   The guard fires during PROVIDER CONSTRUCTION, strictly before any network
#   call to Azure. So the whole path is exercisable against a purpose-configured
#   local instance with no Azure subscription, no key, and no endpoint.
#
# THE §11.4.115 POLARITY — WHY "WHITESPACE" IS THE LOAD-BEARING CASE
#   Pre-fix:   endpoint resolved from the env var, then `if endpoint == ""`.
#              A whitespace-only value ("   ") is NOT equal to "", so the guard
#              did NOT fire: the provider was constructed carrying a garbage
#              endpoint, which is the latent nil/garbage-deref this commit
#              closed.
#   Post-fix:  endpoint = strings.TrimSpace(endpoint) BEFORE the check, so
#              "   " collapses to "" and construction fails with the specific
#              message `azure endpoint is required`.
#   Therefore observing that exact message for a WHITESPACE-ONLY endpoint is a
#   property that could NOT have been observed before the fix. That is the
#   discriminating assertion.
#
#   An UNSET endpoint is deliberately NOT used as the proof: the `== ""` branch
#   predates this commit, so an unset-endpoint refusal would look identical
#   before and after and would prove nothing. Choosing the case that actually
#   discriminates is the whole point (§11.4.115).
#
# THE SERVER-REACHABILITY CHAIN (verified in source, not assumed)
#   resolveLLMProvider  (internal/server/llm_generate.go)
#     -> llm.Select / parseCloudProviderType  — accepts "azure"
#        (internal/llm/provider_factory.go:151)
#     -> llm.NewCloudProvider(ProviderTypeAzure, entry)
#        (internal/llm/provider_factory.go:76)
#     -> NewAzureProvider(cfg)                — the guard
#        (internal/llm/azure_provider.go:192)
#   The facade returns the construction error verbatim as
#   {"error":{"message": err.Error(), "type":"server_error"}} with status
#   503 (providerResolveStatus: non-errUnknownProvider -> ServiceUnavailable).
#   Note the server passes ProviderConfigEntry{Type,Enabled,Models} with NO
#   Parameters map, so the endpoint is resolved from AZURE_OPENAI_ENDPOINT —
#   which is exactly the input this capture controls.
#
# THE NO-CRASH ASSERTION
#   The defect class was a nil-ptr panic. So after the rejected request, the
#   instance is probed again on /health: a server that took a nil-deref on the
#   bad-endpoint path would be dead. A 200 there is positive evidence the
#   failure was a clean, contained refusal rather than a crash.
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED (guard regressed) |
#             2 INCOMPLETE (a case SKIPped with reason, §11.4.3)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_feat_azureguard_eb233785.sh"

COMMIT="eb23378513d94409447e324b1ad989ece20a1f08"

OPENAI_BODY='{"model":"default","messages":[{"role":"user","content":"QA probe: provider-construction guard check"}],"max_tokens":16}'

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "feat_azureguard" "$COMMIT" \
    "Azure provider: a whitespace-only endpoint is refused cleanly at construction (no garbage endpoint, no crash)"

trap qa_stop_ephemeral_server EXIT

echo
echo "### PHASE 1 — whitespace-only AZURE_OPENAI_ENDPOINT on an ephemeral instance"
echo "### (the discriminating case: pre-fix this was ACCEPTED, post-fix it is refused)"
echo

EPH_KEY="qa-ephemeral-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
qa_redact "$EPH_KEY"

if qa_boot_ephemeral_server \
        HELIX_WIRE_FACADE_API_KEYS="$EPH_KEY" \
        HELIX_LLM_PROVIDER="azure" \
        AZURE_OPENAI_ENDPOINT="   " \
        HELIX_AUTH_JWT_SECRET="${HELIX_AUTH_JWT_SECRET:-}" \
        HELIX_DATABASE_PASSWORD="${HELIX_DATABASE_PASSWORD:-}"
then
    EPH="http://127.0.0.1:${QA_EPH_PORT}"

    qa_http "eph_azure_whitespace_endpoint" POST "${EPH}/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${EPH_KEY}" \
        -d "$OPENAI_BODY" --max-time 60

    # Past the auth gate — this is a provider-resolution outcome, not a 401.
    assert_status_not eph_azure_whitespace_endpoint 401
    # providerResolveStatus: a construction failure maps to 503.
    assert_status     eph_azure_whitespace_endpoint 503

    # THE LOAD-BEARING §11.4.115 ASSERTION: the guard's exact message, for an
    # input that pre-fix did NOT trigger it.
    assert_body_contains eph_azure_whitespace_endpoint "azure endpoint is required"

    # The whitespace endpoint must never have been accepted as a real URL:
    # no completion may come back, and no Azure call may have been attempted.
    assert_body_not_contains eph_azure_whitespace_endpoint '"choices"'
    assert_body_not_contains eph_azure_whitespace_endpoint '"completion_tokens"'

    # NO-CRASH: the defect class was a nil-ptr panic. A dead server cannot
    # answer this; a 200 proves the refusal was clean and contained.
    qa_http "eph_alive_after_bad_endpoint" GET "${EPH}/health" --max-time 20
    assert_status eph_alive_after_bad_endpoint 200

    qa_stop_ephemeral_server
else
    qa_skip "eph_azure_whitespace_endpoint" \
        "whitespace-only AZURE_OPENAI_ENDPOINT is refused with 'azure endpoint is required' (pre-fix it was accepted)" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
    qa_skip "eph_alive_after_bad_endpoint" \
        "server survives the bad-endpoint path (no nil-ptr crash)" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
fi

qa_finish
