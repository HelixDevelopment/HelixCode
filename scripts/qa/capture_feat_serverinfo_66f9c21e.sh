#!/usr/bin/env bash
# capture_feat_serverinfo_66f9c21e.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   66f9c21e  fix(server): advertise streaming_enabled + plugins_enabled in
#             /server/info; wire plugin registry
#
# WHAT THE FIX CHANGED (helix_code/internal/server/{handlers.go,server.go})
#   BEFORE: GET /api/v1/server/info advertised only 3 feature flags
#           (auth_enabled, mcp_enabled, notifications_enabled). The
#           POST /api/v1/llm/stream route was registered and real, but the
#           corresponding capability was simply NOT advertised; and the
#           *plugins.Registry was never wired into the HTTP Server at all.
#   AFTER:  the features object also carries
#             streaming_enabled : true (the streamLLM route is registered
#                                 unconditionally and real)
#             plugins_enabled   : s.pluginRegistry != nil — the registry is
#                                 genuinely constructed at boot and plugins
#                                 loaded from the "plugins" dir via
#                                 plugins.Loader.
#           ensemble / lsp / skills are deliberately NOT advertised, because
#           internal/{ensemble,lsp,skills} do not exist — advertising them
#           would be a §11.4.122 bluff (advertise ONLY real capabilities).
#
# THE ASSERTION SET
#   POSITIVE (the RED->GREEN delta): the two keys are present AND true. Pre-fix
#     the keys were ABSENT entirely, so "present and true" is exactly the
#     falsifiable property the fix introduced.
#   CAPABILITY-IS-REAL (the load-bearing anti-bluff pair): a flag that merely
#     says "true" proves nothing on its own. So the advertised streaming
#     capability is corroborated on the wire by ROUTE-REGISTRATION
#     DISCRIMINATION — POST /api/v1/llm/stream returns 401 (route exists, is
#     registered, and is auth-gated) while a deliberately nonexistent sibling
#     route under the same prefix returns 404 "page not found". The 404 control
#     case is what makes the 401 meaningful: it proves this harness can tell a
#     registered route from an unregistered one, so the 401 is evidence of
#     registration rather than an artefact of a catch-all.
#   NEGATIVE (§11.4.122 honesty): the response must NOT advertise
#     ensemble_enabled / lsp_enabled / skills_enabled. Advertising a capability
#     the server does not implement is the exact bluff this commit refused to
#     ship; asserting their ABSENCE keeps that refusal permanently guarded.
#
# HONEST BOUNDARY (§11.4.6) — stated, not papered over
#   `plugins_enabled` is computed as `s.pluginRegistry != nil`. Over HTTP alone
#   a genuinely-wired registry and a hardcoded `true` literal are
#   INDISTINGUISHABLE. This capture therefore proves (a) the key is advertised
#   at all — the falsifiable pre/post delta — and (b) that the sibling
#   streaming capability it was shipped with is backed by a really-registered
#   route. It does NOT, and does not claim to, prove the registry object is
#   non-nil by observation. Proving that requires the boot log
#   ("loaded 1 plugin") or a plugin-invoking endpoint; neither is reachable
#   read-only from this surface. No assertion here asserts more than it sees.
#
# SCOPE: runs entirely against the ALREADY-RUNNING server. Boots nothing,
#   binds no port, mutates no state — every request is a read-only GET/POST
#   that the server rejects or answers without side effects (§11.4.119).
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED (feature regressed) |
#             2 INCOMPLETE (a case SKIPped with reason, §11.4.3)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_feat_serverinfo_66f9c21e.sh"

COMMIT="66f9c21e7ee92d431544a022ab09e21f49c3fb0e"
QA_BASE_URL="${QA_BASE_URL:-http://localhost:8081}"

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "feat_serverinfo" "$COMMIT" \
    "/server/info advertises streaming_enabled + plugins_enabled (and does NOT advertise absent subsystems)"

echo
echo "### PHASE 1 — the advertised capability flags (${QA_BASE_URL})"
echo

qa_http "info_flags" GET "${QA_BASE_URL}/api/v1/server/info" \
    -H "Accept: application/json" --max-time 15
assert_status        info_flags 200

# POSITIVE: the two keys the fix added. Pre-fix BOTH were absent.
assert_body_contains info_flags '"streaming_enabled":true'
assert_body_contains info_flags '"plugins_enabled":true'

# Sanity: the pre-existing flags are still advertised — the fix ADDED keys and
# did not disturb the ones already shipped.
assert_body_contains info_flags '"auth_enabled"'
assert_body_contains info_flags '"mcp_enabled"'
assert_body_contains info_flags '"notifications_enabled"'

echo
echo "### PHASE 2 — §11.4.122 honesty: absent subsystems must NOT be advertised"
echo

# internal/{ensemble,lsp,skills} do not exist in the tree. Advertising them
# would be a capability bluff. Their ABSENCE from the payload is the assertion.
assert_body_not_contains info_flags '"ensemble_enabled"'
assert_body_not_contains info_flags '"lsp_enabled"'
assert_body_not_contains info_flags '"skills_enabled"'

echo
echo "### PHASE 3 — the advertised streaming capability is backed by a REAL route"
echo

# P3: the streaming route the flag advertises. 401 == registered + auth-gated.
qa_http "stream_route_registered" POST "${QA_BASE_URL}/api/v1/llm/stream" \
    -H "Content-Type: application/json" \
    -d '{"prompt":"QA probe: route-registration check only"}' \
    --max-time 20
assert_status            stream_route_registered 401
# A registered-but-gated route must NOT answer with the router's 404 text.
assert_body_not_contains stream_route_registered "page not found"

# P4: the CONTROL that makes P3 meaningful — an unregistered sibling route
#     under the same prefix must 404. Without this, a catch-all returning 401
#     everywhere would make P3 vacuous.
qa_http "unregistered_control" POST "${QA_BASE_URL}/api/v1/llm/qa-probe-no-such-route" \
    -H "Content-Type: application/json" -d '{}' --max-time 15
assert_status        unregistered_control 404
assert_body_contains unregistered_control "page not found"

qa_finish
