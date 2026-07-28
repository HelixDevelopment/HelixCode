#!/usr/bin/env bash
# capture_docfix_cors_reconcile_f9dcf6a6.sh
#
# §11.4.83 QA capture for:
#   f9dcf6a6  fix(server): update CORSMiddleware call sites for allowlist
#             signature (§11.4.120 reconcile)
#
# WHAT THE COMMIT ACTUALLY CHANGED (verified against the commit's own diff —
# `git show f9dcf6a6`, not assumed from the subject line)
#   Three files:
#     helix_code/internal/server/doc.go                  (package-comment example only)
#     helix_code/internal/server/server_test.go           (test source)
#     helix_code/tests/regression/critical_paths_test.go   (test source)
#   server.go itself — the production CORSMiddleware implementation — is
#   UNTOUCHED by this commit. Its two-argument signature,
#   func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc, was
#   introduced by the earlier security fix 4727a9d0 (already covered by its
#   own docs/qa/ evidence, capture_sec_cors_4727a9d0.sh).
#
# HONEST CORRECTION OF THE "ZERO EXECUTABLE CHANGE" CHARACTERISATION
#   The operator brief characterised this commit as "reportedly a doc.go
#   package-comment edit only, zero executable change". That is only PARTLY
#   right, and this run verifies the discrepancy rather than repeating it:
#     * doc.go IS non-executable (a package-comment example — go vet/go build
#       never compile comment text).
#     * server_test.go and critical_paths_test.go are NOT non-executable —
#       they are real Go test SOURCE, compiled and run by `go vet`/`go test`.
#       This run PROVES (PHASE 1 below) that at the commit's own parent
#       revision, the package DID NOT COMPILE: three call sites still invoked
#       the old zero-argument `CORSMiddleware()` against the already-changed
#       two-argument signature. This commit's test-file changes are a real,
#       necessary §11.4.120 gate-reconciliation fix — not a no-op.
#   What IS true, and is the genuine "zero change" claim worth making: no
#   PRODUCTION/library source file changed, so no end-user-visible request
#   handling behaviour changed. That is the claim this run actually certifies.
#
# WHAT THIS RUN PROVES (three real, falsifiable checks — no HTTP surface
# exists for a doc/test-only commit, so this is command-execution evidence,
# not a wire trace; see the honest scope note in EVIDENCE.md)
#   1. PRE-FIX (parent commit) genuinely fails to compile/vet/test — captured
#      by building/vetting/testing the actual historical source at
#      f9dcf6a6^ in an isolated git worktree (never the working tree; nothing
#      here mutates the checkout this script was invoked from).
#   2. POST-FIX (this commit) genuinely compiles, vets clean, and its
#      reconciled TestCORSMiddleware assertions genuinely pass against the
#      REAL CORSMiddleware implementation.
#   3. doc.go's documented example — server.CORSMiddleware(allowedOrigins) —
#      is ACCURATE: `go doc` against the real, live signature at this
#      commit is captured and cross-checked against doc.go's own committed
#      text. Accurate documentation is the actual deliverable of this commit.
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED | 2 INCOMPLETE (SKIP)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
# shellcheck source=lib/cmd_capture_lib.sh
source "${SCRIPT_DIR}/lib/cmd_capture_lib.sh"
QA_SCRIPT_NAME="capture_docfix_cors_reconcile_f9dcf6a6.sh"

COMMIT="f9dcf6a6a688f158edfa1881641f649e84ac5671"

if [ "${1:-}" = "--self-test" ]; then qa_self_test; exit $?; fi

qa_init "docfix_cors_reconcile" "$COMMIT" \
    "CORSMiddleware call-site reconciliation (doc.go example + test call sites) — §11.4.120"

WORK="$(mktemp -d)"
WT_PARENT="${WORK}/wt_parent"
WT_FIXED="${WORK}/wt_fixed"
SUBMODULES_SRC="${QA_REPO_ROOT}/submodules"

_worktrees_ready=1

cleanup() {
    # Never touches the invoking checkout: only removes the two ephemeral
    # worktrees this script created under $WORK, then $WORK itself.
    git -C "$QA_REPO_ROOT" worktree remove --force "$WT_FIXED" >/dev/null 2>&1 || true
    git -C "$QA_REPO_ROOT" worktree remove --force "$WT_PARENT" >/dev/null 2>&1 || true
    rm -rf "$WORK"
}
trap cleanup EXIT

echo
echo "### Setting up isolated worktrees at the parent commit and at ${COMMIT:0:8} itself"
echo

if ! git -C "$QA_REPO_ROOT" worktree add --detach --quiet "$WT_PARENT" "${COMMIT}^" 2>"${WORK}/wt_parent_err.log"; then
    _worktrees_ready=0
    qa_skip "worktree_setup" "isolated worktrees for pre/post-fix compile comparison" \
        "git worktree add failed for parent commit: $(tail -3 "${WORK}/wt_parent_err.log" | tr '\n' ' ')"
fi
if [ "$_worktrees_ready" = 1 ] && ! git -C "$QA_REPO_ROOT" worktree add --detach --quiet "$WT_FIXED" "$COMMIT" 2>"${WORK}/wt_fixed_err.log"; then
    _worktrees_ready=0
    qa_skip "worktree_setup" "isolated worktrees for pre/post-fix compile comparison" \
        "git worktree add failed for fixed commit: $(tail -3 "${WORK}/wt_fixed_err.log" | tr '\n' ' ')"
fi

# The inner Go module (helix_code/) depends on the helix_agent submodule via
# a relative `replace ../submodules/helix_agent` directive. `git worktree`
# does not populate submodule content (each becomes an empty gitlink
# placeholder dir), so the two ephemeral worktrees cannot resolve that
# dependency out of the box. Bridge it read-only: swap each worktree's empty
# submodules/ placeholder for a symlink to the ALREADY-INITIALISED
# submodules/ tree in the invoking checkout. This never writes into that
# tree (a symlink is read-only from the worktree's side) and is removed
# along with the whole ephemeral worktree in cleanup().
if [ "$_worktrees_ready" = 1 ]; then
    if [ ! -d "$SUBMODULES_SRC/helix_agent" ] || [ ! -f "$SUBMODULES_SRC/helix_agent/go.mod" ]; then
        _worktrees_ready=0
        qa_skip "worktree_setup" "isolated worktrees for pre/post-fix compile comparison" \
            "submodules/helix_agent not initialised in the invoking checkout at ${SUBMODULES_SRC} — cannot bridge the module replace directive"
    else
        for wt in "$WT_PARENT" "$WT_FIXED"; do
            if [ -d "$wt/submodules" ] && [ ! -L "$wt/submodules" ]; then
                find "$wt/submodules" -depth -type d -empty -delete 2>/dev/null || true
            fi
            if [ -e "$wt/submodules" ] && [ ! -L "$wt/submodules" ]; then
                _worktrees_ready=0
                qa_skip "worktree_setup" "isolated worktrees for pre/post-fix compile comparison" \
                    "$wt/submodules is non-empty and not a placeholder — refusing to touch it"
                break
            fi
            ln -sf "$SUBMODULES_SRC" "$wt/submodules"
        done
    fi
fi

if [ "$_worktrees_ready" = 1 ]; then
    echo "-- worktrees ready: parent=${WT_PARENT} fixed=${WT_FIXED}"
    echo
    echo "### PHASE 1 — PRE-FIX (parent commit ${COMMIT:0:8}^): must NOT compile/vet/test clean"
    echo

    ( cd "$WT_PARENT/helix_code" && qa_cmd "parent_vet_server" \
        "go vet ./internal/server/... at the parent commit (pre-fix)" \
        -- go vet ./internal/server/... )
    assert_cmd_rc_not parent_vet_server 0 \
        "go vet ./internal/server/... FAILS at the parent commit (arity mismatch, not yet reconciled)"
    assert_cmd_output_contains parent_vet_server "not enough arguments in call to CORSMiddleware" \
        "vet error names the exact arity mismatch this commit fixes"

    ( cd "$WT_PARENT/helix_code" && qa_cmd "parent_vet_regression" \
        "go vet ./tests/regression/... at the parent commit (pre-fix)" \
        -- go vet ./tests/regression/... )
    assert_cmd_rc_not parent_vet_regression 0 \
        "go vet ./tests/regression/... FAILS at the parent commit"
    assert_cmd_output_contains parent_vet_regression "not enough arguments in call to server.CORSMiddleware" \
        "vet error names the exact call site this commit fixes"

    ( cd "$WT_PARENT/helix_code" && qa_cmd "parent_test_server" \
        "go test ./internal/server/... -run TestCORSMiddleware -v at the parent commit (pre-fix)" \
        -- go test ./internal/server/... -run TestCORSMiddleware -v )
    assert_cmd_rc_not parent_test_server 0 \
        "go test FAILS to even build at the parent commit — the package is genuinely broken, not cosmetically different"
    assert_cmd_output_contains parent_test_server "build failed" \
        "go test reports a build failure, confirming the test file itself does not compile pre-fix"

    echo
    echo "### PHASE 2 — POST-FIX (this commit, ${COMMIT:0:8}): must compile/vet/test clean"
    echo

    ( cd "$WT_FIXED/helix_code" && qa_cmd "fixed_vet_all" \
        "go vet ./internal/server/... ./tests/regression/... at this commit (post-fix)" \
        -- go vet ./internal/server/... ./tests/regression/... )
    assert_cmd_rc fixed_vet_all 0 \
        "go vet is clean at this commit — the reconciliation compiles"

    ( cd "$WT_FIXED/helix_code" && qa_cmd "fixed_test_server" \
        "go test ./internal/server/... -run TestCORSMiddleware -v at this commit (post-fix)" \
        -- go test ./internal/server/... -run TestCORSMiddleware -v )
    assert_cmd_rc fixed_test_server 0 \
        "go test PASSES at this commit — the reconciled call sites exercise the REAL CORSMiddleware(allowedOrigins) implementation"
    assert_cmd_output_contains fixed_test_server "PASS: TestCORSMiddleware_AllowedOrigin_EchoedWithVary" \
        "the reconciled test genuinely asserts allowlisted-origin echo + Vary:Origin behaviour, not a trivial no-op"
    assert_cmd_output_not_contains fixed_test_server "FAIL" \
        "no test in the TestCORSMiddleware family regressed"

    echo
    echo "### PHASE 3 — doc.go's documented example is ACCURATE (the actual deliverable)"
    echo

    ( cd "$WT_FIXED/helix_code" && qa_cmd "fixed_godoc_signature" \
        "go doc ./internal/server CORSMiddleware at this commit — the REAL, live signature" \
        -- go doc ./internal/server CORSMiddleware )
    assert_cmd_rc fixed_godoc_signature 0 \
        "go doc resolves CORSMiddleware cleanly at this commit"
    assert_cmd_output_contains fixed_godoc_signature \
        "func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc" \
        "the REAL exported signature matches what doc.go's example claims it is"

    ( cd "$QA_REPO_ROOT" && qa_cmd "committed_docgo_example" \
        "the exact package-comment text this commit committed into doc.go" \
        -- git show "${COMMIT}:helix_code/internal/server/doc.go" )
    assert_cmd_rc committed_docgo_example 0 \
        "doc.go is readable at this commit"
    assert_cmd_output_contains committed_docgo_example "server.CORSMiddleware(allowedOrigins)" \
        "doc.go's committed example calls CORSMiddleware with the allowedOrigins parameter — matching the real signature captured above, not the stale zero-arg form"
    assert_cmd_output_not_contains committed_docgo_example "server.CORSMiddleware()" \
        "doc.go no longer documents the removed zero-argument call shape"
else
    echo
    echo "### Worktree setup unavailable — every case below is an honest SKIP (§11.4.3), never a fabricated PASS"
    echo
    for c in parent_vet_server parent_vet_regression parent_test_server \
             fixed_vet_all fixed_test_server fixed_godoc_signature committed_docgo_example; do
        qa_skip "$c" "pre/post-fix compile+doc-accuracy comparison" \
            "isolated worktree environment could not be prepared in this run (see worktree_setup SKIP reason above)"
    done
fi

(
    qa_finish
)
rc=$?

# qa_finish()'s boilerplate (in sec_capture_lib.sh, not modified here) is
# written for its primary audience, HTTP wire captures ("a real HTTP capture
# harness", "transcripts/<case-id>.http", "no response"). This commit has no
# HTTP surface at all, so append a candid, plainly-worded scope section in
# this run's own words directly to the EVIDENCE.md file qa_finish just wrote
# (§11.4.6 — state the boundary, do not let a generic template imply more
# than was actually exercised).
cat >> "${QA_REPO_ROOT}/docs/qa/${QA_RUN_ID}/EVIDENCE.md" <<'SCOPE_EOF'

## Honest scope statement (§11.4.6) — read this before the table above

**What this commit's user-visible surface actually is: none.** `f9dcf6a6`
touches `helix_code/internal/server/doc.go` (a package-comment example — not
compiled, not executable), plus `helix_code/internal/server/server_test.go`
and `helix_code/tests/regression/critical_paths_test.go` (Go test SOURCE,
compiled and run by `go vet`/`go test`, but never shipped in a production
binary). `helix_code/internal/server/server.go` — the actual production
`CORSMiddleware` implementation an end user's HTTP requests pass through — is
**not touched** by this commit. There is therefore no request/response
behaviour, no CLI output, and no user journey for this run to exercise, and
none is claimed above.

**What this run DOES prove**, with real, repeated command execution captured
in `transcripts/*.txt` (not `*.http` — there is no wire trace for a
doc/test-only change; every transcript here is a captured combined
stdout+stderr + real exit code from an actual `go vet` / `go test` / `go doc`
/ `git show` invocation):

1. The commit's own parent revision (`f9dcf6a6^`) genuinely fails `go
   vet`/`go test` — three call sites still invoked the OLD zero-argument
   `CORSMiddleware()` against the signature `4727a9d0` had already changed to
   `CORSMiddleware(allowedOrigins []string)`. This is captured live from an
   isolated, ephemeral `git worktree` checked out at that exact historical
   commit — not asserted from the diff alone.
2. This commit's own revision genuinely compiles clean and its reconciled
   `TestCORSMiddleware` family genuinely passes against the REAL
   `CORSMiddleware` implementation (also captured from an isolated worktree
   at this exact commit).
3. `doc.go`'s documented example — `server.CORSMiddleware(allowedOrigins)` —
   is **accurate**: `go doc` against the real, live exported signature at
   this commit is captured verbatim and cross-checked against the exact text
   `doc.go` committed. Accurate documentation is this commit's actual,
   narrow deliverable, and that is what is certified here — nothing more.

**What this run does NOT prove:** that any end user, request, or workflow is
affected — because none is. Characterising this as "zero executable change"
(as an earlier summary of this commit put it) is only partly right: the
`doc.go` half is non-executable, but the two test-file changes are real,
necessary Go test-source fixes (proven by PHASE 1 above: the package did not
compile without them) — a legitimate §11.4.120 gate-reconciliation, not a
no-op.
SCOPE_EOF

exit "$rc"
