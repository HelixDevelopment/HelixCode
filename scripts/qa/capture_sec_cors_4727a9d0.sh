#!/usr/bin/env bash
# capture_sec_cors_4727a9d0.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   4727a9d0  fix(security): CORS no wildcard-origin-with-credentials
#             — allowlist + Vary:Origin
#
# WHAT THE FIX CHANGED (helix_code/internal/server/server.go CORSMiddleware)
#   BEFORE: every response, unconditionally:
#             Access-Control-Allow-Origin: *
#             Access-Control-Allow-Credentials: true
#           That pairing is forbidden by the Fetch/CORS spec, and an
#           implementation reflecting the wildcard as the literal request
#           Origin lets ANY origin make credentialed cross-origin requests.
#   AFTER:  CORSMiddleware(allowedOrigins []string). Only an Origin present in
#           cfg.Auth.CORSAllowedOrigins (HELIX_CORS_ALLOWED_ORIGINS) is echoed
#           back — as THAT SPECIFIC origin, never "*" — together with
#           "Vary: Origin", and only then is Allow-Credentials set. Every other
#           request gets NO CORS-allow headers at all (default-deny).
#
# THE ASSERTION PAIR
#   NEGATIVE (the security proof — would have passed before the fix):
#     A request carrying `Origin: https://evil.example` + credentials must NOT
#     receive `Access-Control-Allow-Origin: *` (or any ACAO), and must NOT
#     receive `Access-Control-Allow-Credentials`.
#   POSITIVE (the fix did not simply break CORS):
#     A request carrying an ALLOWLISTED Origin receives that exact origin back,
#     plus `Vary: Origin`, plus `Access-Control-Allow-Credentials: true`.
#   INVARIANT (checked across every captured response):
#     never ACAO:* together with Allow-Credentials.
#
# HONEST SCOPE NOTE (§11.4.6)
#   The already-running server (localhost:8081) has HELIX_CORS_ALLOWED_ORIGINS
#   UNSET — it is in pure default-deny mode. The NEGATIVE case is therefore
#   fully provable against it, but the POSITIVE (allowlisted-origin) case is
#   NOT reachable there: with an empty allowlist NO origin is ever echoed. The
#   positive case is exercised on a purpose-configured ephemeral instance this
#   script boots on a free loopback port. If that instance cannot boot, the
#   positive case is recorded as SKIP-with-reason (§11.4.3) and the run exits 2
#   (INCOMPLETE) — never as a PASS.
#
# EXIT CODES: 0 all PASS | 1 a security assertion FAILED | 2 INCOMPLETE (SKIP)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_sec_cors_4727a9d0.sh"

COMMIT="4727a9d0d76be565e80a04afc587a906c318ecb3"
QA_BASE_URL="${QA_BASE_URL:-http://localhost:8081}"

ALLOWED_ORIGIN="https://app.allowed.example"
EVIL_ORIGIN="https://evil.example"

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "sec_cors" "$COMMIT" "CORS: no wildcard-origin-with-credentials (allowlist + Vary:Origin)"
trap qa_stop_ephemeral_server EXIT

echo
echo "### PHASE 1 — NEGATIVE cases against the LIVE server (${QA_BASE_URL})"
echo "### (default-deny: HELIX_CORS_ALLOWED_ORIGINS unset)"
echo

# N1: the classic cross-origin credentialed request from an attacker origin.
qa_http "live_evil_origin_get" GET "${QA_BASE_URL}/health" \
    -H "Origin: ${EVIL_ORIGIN}" \
    -H "Cookie: session=qa-probe" \
    --max-time 15
assert_status            live_evil_origin_get 200
assert_header_absent     live_evil_origin_get "Access-Control-Allow-Origin"
assert_header_absent     live_evil_origin_get "Access-Control-Allow-Credentials"

# N2: CORS preflight from the attacker origin — the browser's actual gate.
qa_http "live_evil_origin_preflight" OPTIONS "${QA_BASE_URL}/api/v1/tasks" \
    -H "Origin: ${EVIL_ORIGIN}" \
    -H "Access-Control-Request-Method: POST" \
    -H "Access-Control-Request-Headers: authorization,content-type" \
    --max-time 15
assert_header_absent     live_evil_origin_preflight "Access-Control-Allow-Origin"
assert_header_absent     live_evil_origin_preflight "Access-Control-Allow-Credentials"

# N3: an origin that only LOOKS like an allowlisted one (substring/prefix probe).
qa_http "live_lookalike_origin" GET "${QA_BASE_URL}/health" \
    -H "Origin: ${ALLOWED_ORIGIN}.evil.example" \
    --max-time 15
assert_header_absent     live_lookalike_origin "Access-Control-Allow-Origin"
assert_header_absent     live_lookalike_origin "Access-Control-Allow-Credentials"

# N4: no Origin header at all (same-origin / non-browser) — must be unaffected.
qa_http "live_no_origin" GET "${QA_BASE_URL}/health" --max-time 15
assert_status            live_no_origin 200
assert_header_absent     live_no_origin "Access-Control-Allow-Origin"

echo
echo "### PHASE 2 — POSITIVE case on a purpose-configured ephemeral instance"
echo

if qa_boot_ephemeral_server \
        HELIX_CORS_ALLOWED_ORIGINS="${ALLOWED_ORIGIN},https://admin.allowed.example" \
        HELIX_AUTH_JWT_SECRET="${HELIX_AUTH_JWT_SECRET:-}" \
        HELIX_DATABASE_PASSWORD="${HELIX_DATABASE_PASSWORD:-}"
then
    EPH="http://127.0.0.1:${QA_EPH_PORT}"

    # P1: allowlisted origin — echoed as THAT origin (never "*"), with Vary.
    qa_http "eph_allowed_origin" GET "${EPH}/health" \
        -H "Origin: ${ALLOWED_ORIGIN}" \
        -H "Cookie: session=qa-probe" \
        --max-time 15
    assert_status        eph_allowed_origin 200
    assert_header_equals eph_allowed_origin "Access-Control-Allow-Origin" "${ALLOWED_ORIGIN}"
    assert_header_present eph_allowed_origin "Vary" "Origin"
    assert_header_equals eph_allowed_origin "Access-Control-Allow-Credentials" "true"

    # P2: second allowlisted origin — proves it is a list, not a single value.
    qa_http "eph_allowed_origin_2" GET "${EPH}/health" \
        -H "Origin: https://admin.allowed.example" \
        --max-time 15
    assert_header_equals eph_allowed_origin_2 "Access-Control-Allow-Origin" "https://admin.allowed.example"

    # N5: on the SAME configured instance, a non-allowlisted origin still denied.
    #     This is the sharpest negative: the allowlist is live and non-empty, yet
    #     the attacker origin gets nothing.
    qa_http "eph_evil_origin" GET "${EPH}/health" \
        -H "Origin: ${EVIL_ORIGIN}" \
        -H "Cookie: session=qa-probe" \
        --max-time 15
    assert_header_absent eph_evil_origin "Access-Control-Allow-Origin"
    assert_header_absent eph_evil_origin "Access-Control-Allow-Credentials"

    qa_stop_ephemeral_server
else
    qa_skip "eph_allowed_origin" \
        "POSITIVE: allowlisted Origin echoed specifically + Vary:Origin + Allow-Credentials" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
    qa_skip "eph_evil_origin" \
        "NEGATIVE-with-allowlist-configured: non-allowlisted Origin still denied" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
fi

echo
echo "### PHASE 3 — CORS spec invariant across every captured response"
echo
assert_no_wildcard_with_credentials

qa_finish
