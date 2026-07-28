#!/usr/bin/env bash
# capture_testfix_cognee_determinism_b058c7c2.sh
#
# §11.4.83 QA capture for:
#   b058c7c2  fix(tests/cognee): make the package deterministic under load
#             (§11.4.50/§11.4.6) — no product change
#
# WHAT THE COMMIT CHANGED (verified against the commit's own diff)
#   internal/cognee/client_stresschaos_test.go              (test source)
#   internal/cognee/performance_optimizer.go                (PRODUCTION source)
#   internal/cognee/performance_optimizer_gpu_amd_test.go    (test source)
#   internal/cognee/performance_optimizer_gpu_apple_test.go  (test source)
#   internal/cognee/performance_optimizer_gpu_intel_test.go  (test source)
#
# HONEST SCOPE (§11.4.6) — the "no product change" claim, checked, not assumed
#   The subject says "no product change", but one PRODUCTION file
#   (performance_optimizer.go) genuinely changed: three GPU-probe shell-out
#   timeouts (nvidiaSmiQueryTimeout, appleIoregQueryTimeout,
#   intelGPUTopQueryTimeout) were converted from `const` to `var`, mirroring
#   the pre-existing rocmSmiQueryTimeout var. This run does NOT repeat the
#   subject's claim uncritically — it VERIFIES the specific, narrower claim
#   that actually holds: (a) the default VALUES are unchanged (still 2s/2s/3s
#   — a text diff, checked below), and (b) no PRODUCTION (non-test) code path
#   ever reassigns these vars — only test files do, to raise the timeout for
#   load-robustness. A var whose default is unchanged and which production
#   code never mutates behaves identically to the former const for every
#   shipped code path. That is the real, checked meaning of "no product
#   change" here — a mutability change enabling a test hook, not a behaviour
#   change. This run's deliverable is different: DETERMINISM. The end-user
#   surface for a test-only fix is, honestly, none — see EVIDENCE.md.
#
# WHAT THIS RUN PROVES (real, repeated command execution — not a single
# lucky run, and not an assertion made from the commit message's own
# "Verified:" claim)
#   1. No non-test file reassigns any of the four GPU-probe timeout vars —
#      the const->var change is genuinely inert for the shipped binary.
#   2. The full internal/cognee package passes repeatedly and consecutively
#      (>=5 full-package runs in one `-count` invocation, PLUS several fully
#      independent process invocations back-to-back) — the load-flakiness
#      the commit describes does not reproduce post-fix.
#   3. The specific test the commit names as previously flaky even in
#      isolation (TestClientHTTP_Stress_ConcurrentRequests) passes repeatedly
#      when run alone, repeatedly.
#   4. The GPU probe-chain tests the commit names as previously flaky under
#      parallel load pass repeatedly.
#   5. `go test -race` is clean — the keep-alive/timeout rework did not
#      introduce a data race.
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED | 2 INCOMPLETE (SKIP)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
# shellcheck source=lib/cmd_capture_lib.sh
source "${SCRIPT_DIR}/lib/cmd_capture_lib.sh"
QA_SCRIPT_NAME="capture_testfix_cognee_determinism_b058c7c2.sh"

COMMIT="b058c7c28c48dd09aa30cfaa5b5bf60c8565c35e"
REPEATS="${QA_COGNEE_REPEATS:-5}"   # "-count=5 or more" per the operator brief

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "testfix_cognee_determinism" "$COMMIT" \
    "internal/cognee package determinism under repeated/parallel load — §11.4.50/§11.4.6"

INNER="${QA_REPO_ROOT}/helix_code"

echo
echo "### PHASE 1 — verify the narrower, actually-true 'no product change' claim"
echo

( cd "$INNER" && qa_cmd "prod_var_never_reassigned" \
    "grep for reassignment of the four GPU-probe timeout vars OUTSIDE test files (declaration lines and test files excluded)" \
    -- bash -c "grep -RIn -E '^[[:space:]]*(nvidiaSmiQueryTimeout|rocmSmiQueryTimeout|appleIoregQueryTimeout|intelGPUTopQueryTimeout)[[:space:]]*=' internal/cognee --include='*.go' | grep -v _test.go | grep -v '\\bvar\\b'" )
assert_cmd_rc_not prod_var_never_reassigned 0 \
    "zero matches: no production (non-test) code path anywhere in internal/cognee ever reassigns these vars — grep found nothing to reassign, so it correctly exits non-zero"
assert_cmd_output_not_contains prod_var_never_reassigned "performance_optimizer.go" \
    "the one production file that changed does not itself contain a reassignment — only the var declarations (checked next), which is not a reassignment"

( cd "$INNER" && qa_cmd "prod_defaults_unchanged" \
    "the current default values of the three converted timeouts, read from source" \
    -- bash -c "grep -n 'var nvidiaSmiQueryTimeout\\|var appleIoregQueryTimeout\\|var intelGPUTopQueryTimeout' internal/cognee/performance_optimizer.go" )
assert_cmd_rc prod_defaults_unchanged 0 \
    "the three var declarations are present and readable"
assert_cmd_output_contains prod_defaults_unchanged "var nvidiaSmiQueryTimeout = 2 * time.Second" \
    "nvidia default is unchanged at 2s (matches the const value before this commit)"
assert_cmd_output_contains prod_defaults_unchanged "var appleIoregQueryTimeout = 2 * time.Second" \
    "apple ioreg default is unchanged at 2s"
assert_cmd_output_contains prod_defaults_unchanged "var intelGPUTopQueryTimeout = 3 * time.Second" \
    "intel default is unchanged at 3s"

echo
echo "### PHASE 2 — full-package determinism: ${REPEATS}x in one -count invocation"
echo

( cd "$INNER" && qa_cmd "full_package_count5" \
    "go test ./internal/cognee/... -count=${REPEATS} — the full package run ${REPEATS} times in one invocation" \
    -- go test ./internal/cognee/... -count="${REPEATS}" )
assert_cmd_rc full_package_count5 0 \
    "the full package passes cleanly across ${REPEATS} repeated runs in a single invocation"
assert_cmd_output_not_contains full_package_count5 "FAIL" \
    "no run in the ${REPEATS}x repetition reports a failure"

echo
echo "### PHASE 3 — full-package determinism: 3 fully INDEPENDENT process invocations"
echo "###           (catches process-level/scheduling nondeterminism that -count reuse can mask)"
echo

for i in 1 2 3; do
    ( cd "$INNER" && qa_cmd "full_package_independent_run_${i}" \
        "go test ./internal/cognee/... -count=1 — independent process invocation #${i} of 3" \
        -- go test ./internal/cognee/... -count=1 )
    assert_cmd_rc "full_package_independent_run_${i}" 0 \
        "independent process invocation #${i} passes"
done

echo
echo "### PHASE 4 — the specific test the commit names as previously flaky, alone, repeated"
echo

( cd "$INNER" && qa_cmd "stress_concurrent_alone_repeated" \
    "go test ./internal/cognee/... -run TestClientHTTP_Stress_ConcurrentRequests -count=${REPEATS} -v" \
    -- go test ./internal/cognee/... -run TestClientHTTP_Stress_ConcurrentRequests -count="${REPEATS}" -v )
assert_cmd_rc stress_concurrent_alone_repeated 0 \
    "TestClientHTTP_Stress_ConcurrentRequests, run alone and repeated ${REPEATS}x, passes every time — the commit's own '12/15 fail even alone' pre-fix failure mode does not reproduce"
assert_cmd_output_not_contains stress_concurrent_alone_repeated "FAIL" \
    "zero failures across the repeated stress-test-alone runs"

echo
echo "### PHASE 5 — the GPU probe-chain tests the commit names as previously flaky under parallel load"
echo

( cd "$INNER" && qa_cmd "gpu_probe_chain_repeated" \
    "go test ./internal/cognee/... -run TestGetGPUUsage_ProbeChain -count=${REPEATS} -v" \
    -- go test ./internal/cognee/... -run TestGetGPUUsage_ProbeChain -count="${REPEATS}" -v )
assert_cmd_rc gpu_probe_chain_repeated 0 \
    "all TestGetGPUUsage_ProbeChain_* variants pass across ${REPEATS} repeated runs"
assert_cmd_output_not_contains gpu_probe_chain_repeated "FAIL" \
    "zero failures across the repeated GPU probe-chain runs"

echo
echo "### PHASE 6 — go test -race: the keep-alive/timeout rework introduced no data race"
echo

( cd "$INNER" && qa_cmd "full_package_race" \
    "go test ./internal/cognee/... -race -count=1" \
    -- go test ./internal/cognee/... -race -count=1 )
assert_cmd_rc full_package_race 0 \
    "-race is clean across the full package"
assert_cmd_output_not_contains full_package_race "DATA RACE" \
    "no data race detected"

(
    qa_finish
)
rc=$?

# qa_finish()'s boilerplate (in sec_capture_lib.sh, not modified here) is
# written for its primary audience, HTTP wire captures. This commit is a
# test-determinism fix with (almost) no HTTP surface, so append a candid,
# plainly worded scope section directly to the EVIDENCE.md file qa_finish
# just wrote (§11.4.6 — state the boundary, do not let a generic template
# imply more than was actually exercised).
cat >> "${QA_REPO_ROOT}/docs/qa/${QA_RUN_ID}/EVIDENCE.md" <<SCOPE_EOF

## Honest scope statement (§11.4.6) — read this before the table above

**What this commit's user-visible surface actually is: effectively none.**
\`b058c7c2\` touches five files under \`internal/cognee\`: four are Go test
source, and one — \`performance_optimizer.go\` — is production source, but
the change there is a mutability change only (three GPU-probe shell-out
timeout constants converted from \`const\` to \`var\`, mirroring the
pre-existing \`rocmSmiQueryTimeout\` var), not a behaviour change: the
default values are unchanged and no non-test code path anywhere in the
package ever reassigns them (verified live in PHASE 1 below — this run does
not just repeat the commit message's "no product change" claim, it checks
it). There is no request/response behaviour, no CLI output, and no
end-user journey this run exercises, and none is claimed above.

**What this commit's actual deliverable is, and what this run proves**:
DETERMINISM — that the \`internal/cognee\` package, previously load-flaky
(a different test failing each run, all passing in isolation), now passes
consistently under repeated and parallel execution. This run captures real,
repeated \`go test\` invocations in \`transcripts/*.txt\` (not \`*.http\` —
there is no wire trace for a test-only change):

1. The full package passes cleanly across ${REPEATS} repeated runs in a
   single \`-count=${REPEATS}\` invocation, AND across 3 fully independent
   process invocations run back-to-back (independent processes catch
   scheduling-level nondeterminism that in-process repetition can mask).
2. \`TestClientHTTP_Stress_ConcurrentRequests\` — the test the commit
   specifically names as having failed even in isolation pre-fix
   ("12/15 fail even alone") — passes every time across ${REPEATS} repeated
   solo runs.
3. The \`TestGetGPUUsage_ProbeChain_*\` family — the tests the commit
   describes as flaking under parallel \`go test ./...\` load due to a fake
   subprocess being signal-killed before the (pre-fix, fixed-duration) probe
   timeout elapsed — pass every time across ${REPEATS} repeated runs.
4. \`go test -race\` is clean across the full package: the keep-alive /
   client-timeout rework in \`client_stresschaos_test.go\` introduced no
   new data race.

**What this run does NOT prove:** that any end user or production request
path is affected, because none is — this is a test-infrastructure fix. It
also does not prove ABSOLUTE non-flakiness (no finite number of repeated
runs can); it proves the specific, previously-documented failure modes this
commit targeted do not reproduce across ${REPEATS}+ repeated and 3
independent runs in this session, which is the concrete, falsifiable claim
the commit message actually makes ("Verified: ... -count=3 PASS twice").
SCOPE_EOF

exit "$rc"
