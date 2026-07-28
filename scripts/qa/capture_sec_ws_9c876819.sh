#!/usr/bin/env bash
# capture_sec_ws_9c876819.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   9c876819  fix(security): authenticate /ws MCP WebSocket + fix CSWSH
#             CheckOrigin gap
#
# WHAT THE FIX CHANGED — two distinct sub-fixes:
#   (a) AUTH GATE (helix_code/internal/server/server.go)
#       BEFORE: s.router.GET("/ws", s.handleWebSocket) — registered with ZERO
#               middleware. Any client, any origin, no credential, completed the
#               MCP WebSocket handshake and spawned an MCP session goroutine.
#       AFTER:  s.router.GET("/ws", s.wsAuthMiddleware(), s.handleWebSocket).
#               Bearer / x-api-key checked against cfg.Auth.WireFacadeAPIKeys,
#               fail-closed on empty config, and it runs BEFORE Upgrade() — so an
#               unauthenticated caller gets a plain HTTP 401 and never upgrades,
#               never spawns a goroutine (closes the DoS finding too).
#   (b) CSWSH / CheckOrigin (helix_code/internal/mcp/server.go)
#       BEFORE: websocket.Upgrader.CheckOrigin returned `true` unconditionally
#               ("In production, you should validate the origin").
#       AFTER:  newOriginChecker(extraAllowed) — same-origin / localhost /
#               absent-Origin allowed; every other Origin denied unless
#               explicitly allowlisted (HELIX_WS_ALLOWED_ORIGINS). Never a
#               wildcard.
#
# THE ASSERTION PAIRS
#   (a) NEGATIVE: an unauthenticated /ws upgrade must be refused with 401 and
#       must NOT reach "101 Switching Protocols".  (Pre-fix: 101.)
#       POSITIVE: with a valid key and no Origin (a legitimate non-browser MCP
#       client), the upgrade must SUCCEED with 101 — proving the gate admits
#       real clients rather than breaking the endpoint.
#   (b) NEGATIVE: with a VALID key but a foreign `Origin: https://evil.example`,
#       the upgrade must be REFUSED (gorilla rejects with 403). This is the CSWSH
#       proof.  (Pre-fix: CheckOrigin returned true -> 101.)
#       POSITIVE: with a valid key and a same-origin Origin header, 101.
#
# HONEST SCOPE NOTE (§11.4.6) — LOAD-BEARING
#   Sub-fix (b) sits BEHIND sub-fix (a). The already-running server
#   (localhost:8081) has HELIX_WIRE_FACADE_API_KEYS UNSET, so wsAuthMiddleware
#   fail-closes and returns 401 to EVERY /ws request — newOriginChecker is never
#   reached there. The CheckOrigin allowlist therefore CANNOT be exercised
#   against the live instance; it is only observable on an instance with a key
#   configured. This script boots such an instance on a free loopback port. If it
#   cannot boot, sub-fix (b) is recorded as SKIP-with-reason (§11.4.3) and the run
#   exits 2 (INCOMPLETE) — the CSWSH fix is then NOT certified by this run.
#
# EXIT CODES: 0 all PASS | 1 a security assertion FAILED | 2 INCOMPLETE (SKIP)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_sec_ws_9c876819.sh"

COMMIT="9c876819817ad927fdf8ae55d4084ef9c1200a64"
QA_BASE_URL="${QA_BASE_URL:-http://localhost:8081}"
EVIL_ORIGIN="https://evil.example"

# A WebSocket handshake needs these four headers. Sec-WebSocket-Key is the
# RFC 6455 example value; it is not a credential.
WS_HDRS=(
    -H "Connection: Upgrade"
    -H "Upgrade: websocket"
    -H "Sec-WebSocket-Version: 13"
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ=="
)

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "sec_ws_cswsh" "$COMMIT" "/ws MCP WebSocket: auth gate + CSWSH CheckOrigin allowlist"
trap qa_stop_ephemeral_server EXIT

# A successful upgrade leaves the connection open, so curl will hit its timeout
# (rc 28) AFTER printing "< HTTP/1.1 101". assert_status reads the status from the
# captured received-half of the trace, so rc 28 is not itself a verdict.

echo
echo "### PHASE 1 — AUTH GATE negatives against the LIVE server (${QA_BASE_URL})"
echo "### (fail-closed: HELIX_WIRE_FACADE_API_KEYS unset)"
echo

# N1: the pre-fix attack — plain unauthenticated upgrade.
qa_http "live_ws_unauth_no_origin" GET "${QA_BASE_URL}/ws" "${WS_HDRS[@]}" --max-time 10
assert_status            live_ws_unauth_no_origin 401
assert_header_absent     live_ws_unauth_no_origin "Sec-WebSocket-Accept"
assert_body_not_contains live_ws_unauth_no_origin "101 Switching Protocols"

# N2: unauthenticated upgrade from a hostile browser origin (the CSWSH shape).
qa_http "live_ws_unauth_evil_origin" GET "${QA_BASE_URL}/ws" "${WS_HDRS[@]}" \
    -H "Origin: ${EVIL_ORIGIN}" --max-time 10
assert_status            live_ws_unauth_evil_origin 401
assert_header_absent     live_ws_unauth_evil_origin "Sec-WebSocket-Accept"

# N3: fail-closed proof — a bearer token is present, but no keys are configured,
#     so it must STILL be rejected (not "empty config => open access").
qa_http "live_ws_bogus_bearer" GET "${QA_BASE_URL}/ws" "${WS_HDRS[@]}" \
    -H "Authorization: Bearer qa-probe-not-a-real-key" --max-time 10
assert_status            live_ws_bogus_bearer 401
assert_header_absent     live_ws_bogus_bearer "Sec-WebSocket-Accept"

# N4: same via the Anthropic-style header.
qa_http "live_ws_bogus_apikey" GET "${QA_BASE_URL}/ws" "${WS_HDRS[@]}" \
    -H "x-api-key: qa-probe-not-a-real-key" --max-time 10
assert_status            live_ws_bogus_apikey 401
assert_header_absent     live_ws_bogus_apikey "Sec-WebSocket-Accept"

echo
echo "### PHASE 2 — CSWSH CheckOrigin allowlist, on an instance WITH a key"
echo "### (sub-fix (b) is unreachable on the live server: auth 401s first)"
echo

# Ephemeral, per-run, thrown away at exit. Not a real secret, but redacted from
# the committed transcripts anyway (§11.4.10 hygiene).
EPH_KEY="qa-ephemeral-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
qa_redact "$EPH_KEY"

if qa_boot_ephemeral_server \
        HELIX_WIRE_FACADE_API_KEYS="$EPH_KEY" \
        HELIX_AUTH_JWT_SECRET="${HELIX_AUTH_JWT_SECRET:-}" \
        HELIX_DATABASE_PASSWORD="${HELIX_DATABASE_PASSWORD:-}"
then
    EPH="http://127.0.0.1:${QA_EPH_PORT}"

    # P1 — POSITIVE for sub-fix (a): a legitimate non-browser MCP client with a
    #      valid key and no Origin header must actually upgrade (101).
    qa_http "eph_ws_valid_key_no_origin" GET "${EPH}/ws" "${WS_HDRS[@]}" \
        -H "Authorization: Bearer ${EPH_KEY}" --max-time 8
    assert_status         eph_ws_valid_key_no_origin 101
    assert_header_present eph_ws_valid_key_no_origin "Sec-WebSocket-Accept"

    # P2 — POSITIVE: same-origin browser client is allowed by newOriginChecker.
    qa_http "eph_ws_valid_key_same_origin" GET "${EPH}/ws" "${WS_HDRS[@]}" \
        -H "Authorization: Bearer ${EPH_KEY}" \
        -H "Origin: http://127.0.0.1:${QA_EPH_PORT}" --max-time 8
    assert_status         eph_ws_valid_key_same_origin 101

    # N5 — THE CSWSH NEGATIVE (sub-fix (b)). Authenticated, so the auth gate is
    #      satisfied and newOriginChecker is genuinely reached; a foreign Origin
    #      must be refused. Pre-fix CheckOrigin returned true -> this was 101.
    qa_http "eph_ws_valid_key_evil_origin" GET "${EPH}/ws" "${WS_HDRS[@]}" \
        -H "Authorization: Bearer ${EPH_KEY}" \
        -H "Origin: ${EVIL_ORIGIN}" --max-time 8
    assert_status_not     eph_ws_valid_key_evil_origin 101
    assert_header_absent  eph_ws_valid_key_evil_origin "Sec-WebSocket-Accept"

    # N6 — lookalike origin (allowlist must not do substring matching).
    qa_http "eph_ws_valid_key_lookalike_origin" GET "${EPH}/ws" "${WS_HDRS[@]}" \
        -H "Authorization: Bearer ${EPH_KEY}" \
        -H "Origin: http://127.0.0.1.evil.example" --max-time 8
    assert_status_not     eph_ws_valid_key_lookalike_origin 101
    assert_header_absent  eph_ws_valid_key_lookalike_origin "Sec-WebSocket-Accept"

    # N7 — wrong key on the configured instance still refused.
    qa_http "eph_ws_wrong_key" GET "${EPH}/ws" "${WS_HDRS[@]}" \
        -H "Authorization: Bearer wrong-key-qa-probe" --max-time 8
    assert_status         eph_ws_wrong_key 401
    assert_header_absent  eph_ws_wrong_key "Sec-WebSocket-Accept"

    qa_stop_ephemeral_server
else
    qa_skip "eph_ws_valid_key_no_origin" \
        "POSITIVE (a): valid key + no Origin upgrades to 101" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
    qa_skip "eph_ws_valid_key_evil_origin" \
        "NEGATIVE (b) CSWSH: valid key + foreign Origin must NOT upgrade" \
        "${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable} — the CheckOrigin allowlist fix is NOT certified by this run"
fi

qa_finish
