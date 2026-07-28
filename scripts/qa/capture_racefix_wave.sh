#!/usr/bin/env bash
# capture_racefix_wave.sh <case-key>
#
# §11.4.83 QA evidence capture for the 2026-07-28/29 concurrency-defect wave —
# the eleven feature-shipping commits that the §11.4.83 release gate
# (scripts/gates/qa_evidence_gate.sh) reported as carrying no docs/qa evidence.
#
# WHY A SHARED SCRIPT AND NOT ELEVEN
#   Every commit in this wave has the SAME evidence shape: a concurrency /
#   correctness defect in Go production code, closed by a named regression
#   guard test. The deliverable is therefore COMMAND-shaped (a real compiler +
#   test-runner outcome), not HTTP-shaped, so this script builds on
#   lib/cmd_capture_lib.sh. The per-commit case matrix below is what differs;
#   it is spelled out per commit, never generated, so each run's assertions are
#   specific to the defect that commit actually fixed. One run-id directory is
#   produced per commit (this script is invoked once per key) — deliberately
#   NOT one blanket ledger enumerating eleven SHAs, which is the documented
#   residual risk in scripts/verify_qa_evidence.sh's RULE 3 header.
#
# WHAT EVERY RUN PROVES (nothing here is asserted from a commit message)
#   1. GROUNDING (§11.4.108): the fix commit is an ancestor of HEAD, and the
#      production files under test have NO uncommitted local edits — so the
#      source exercised below is the source that shipped, not a dirty tree.
#   2. RUNTIME SIGNATURE (§11.4.108 SOURCE layer): the fix's own mechanism is
#      still present in the CURRENT tracked source — a real `git grep` for the
#      exact text the commit introduced. A later revert would fail this.
#   3. GUARD EXECUTION: the commit's named regression tests are executed for
#      real under `-race -count=2 -v`, and the transcript is asserted to
#      contain `--- PASS: <TestName>` for EVERY named test. That second
#      assertion is what stops the classic filter bluff — `go test -run` with a
#      regex that matches nothing exits 0 and looks green.
#   4. NO RACE: the transcript is asserted NOT to contain `DATA RACE`.
#   5. PACKAGE SUITE: the whole affected package under `-race`, asserted green.
#
# HONEST BOUNDARY (§11.4.6 / §11.4.3)
#   Two commits in this wave (0a4df699, ccce0b77) changed UI-thread dispatch
#   sites that NO test in the tree exercises — the commits say so themselves.
#   Those runs record an explicit SKIP-with-reason for the un-exercised half
#   and therefore finish INCOMPLETE (exit 2), never PASS. A SKIP is never
#   counted as a pass by this harness.
#
# USAGE
#   scripts/qa/capture_racefix_wave.sh --list
#   scripts/qa/capture_racefix_wave.sh <case-key>
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED | 2 INCOMPLETE (a SKIP)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
# shellcheck source=lib/cmd_capture_lib.sh
source "${SCRIPT_DIR}/lib/cmd_capture_lib.sh"
QA_SCRIPT_NAME="capture_racefix_wave.sh"

APPDIR="${QA_REPO_ROOT}/helix_code"   # the inner Go module (module dev.helix.code)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# sig <case-id> <label> <needle> <repo-relative-path...>
#   Real `git grep` over the CURRENT tracked source for the exact text the fix
#   introduced. Proves the mechanism is still shipped (§11.4.108), and cannot
#   be satisfied by an untracked scratch file.
sig() {
    local case_id="$1" label="$2" needle="$3"; shift 3
    qa_cmd "$case_id" "$label" -- git -C "$QA_REPO_ROOT" grep -n -F -e "$needle" -- "$@"
    assert_cmd_rc "$case_id" 0 "runtime signature present in the shipped tracked source: ${label}"
    # The needle itself is NOT usable as the second assertion: qa_cmd writes a
    # `### command: …` header containing it, so "output contains <needle>"
    # passes even when git grep found nothing (§11.4.201 — an assertion that
    # cannot fail is not an assertion). A `<file>.go:<line>:` hit line can only
    # be produced by a real match, so that is what is asserted.
    assert_cmd_output_contains "$case_id" ".go:" "a real file:line hit line was printed — the match is in a tracked file, not just in the command echo"
}

# absent <case-id> <label> <needle> <repo-relative-path...>
#   The inverse: proves a pattern the fix REMOVED is really gone.
#
#   NOTE on what is asserted, and what deliberately is NOT (§11.4.201):
#   the real condition is `git grep` exiting non-zero — it found nothing. An
#   "output does not contain <needle>" assertion CANNOT be used here and was
#   removed after it produced a guaranteed false FAIL on correct code: qa_cmd
#   writes a `### command: …` header into every transcript, so the needle is
#   always present in the file as part of the command line itself. The second
#   assertion below therefore looks for a `<file>.go:<line>:` hit line, which
#   only a real match can produce — the header carries no such shape.
absent() {
    local case_id="$1" label="$2" needle="$3"; shift 3
    qa_cmd "$case_id" "$label" -- git -C "$QA_REPO_ROOT" grep -n -F -e "$needle" -- "$@"
    assert_cmd_rc_not "$case_id" 0 "removed pattern is genuinely absent (git grep finds nothing): ${label}"
    assert_cmd_output_not_contains "$case_id" ".go:" "no file:line hit line was printed for the removed pattern"
}

# noncomment_grep <needle> <repo-relative-path...>
#   git grep restricted to lines that are not whole-line `//` comments.
#   Exists because a naive absence check for a removed CALL also matches the
#   comment that explains why the call was removed — which would make the
#   assertion fire on correct code (a §11.4.201 false-positive refusal, as
#   damaging as a false pass). Defined as a function so the transcript records
#   the real command name and this file shows exactly what it does.
noncomment_grep() {
    local needle="$1"; shift
    ( cd "$QA_REPO_ROOT" && git grep -n -F -e "$needle" -- "$@" | grep -v -E ':[0-9]+:[[:space:]]*//' )
}

# absent_code <case-id> <label> <needle> <repo-relative-path...>
#   Absence assertion that ignores comment lines (see noncomment_grep).
absent_code() {
    local case_id="$1" label="$2" needle="$3"; shift 3
    qa_cmd "$case_id" "$label" -- noncomment_grep "$needle" "$@"
    assert_cmd_rc_not "$case_id" 0 "no NON-COMMENT occurrence survives: ${label}"
    assert_cmd_output_not_contains "$case_id" ".go:" "no file:line hit line survived the comment filter"
}

# hostctx — records host load at capture time.
#   Not a pass/fail criterion: a recorded confounder (§11.4.50). Some committed
#   guards in the GUI packages are load-sensitive by construction — e.g.
#   TestAuroraDashboardTab_TickerWidgetMutationsAreUIThreadSafe drives a real
#   renderer for 3.5s and calls t.Fatalf when it observes fewer than 2 repaints
#   ("too little renderer/worker overlap for this guard to be meaningful").
#   On a contended host that is an INCONCLUSIVE run reported as a failure, so
#   any suite verdict for those packages is only readable next to the load that
#   produced it. Captured here so a reader never has to guess.
hostctx() {
    qa_cmd host_context "host load average + cpu count at capture time (§11.4.50 confounder record)" \
        -- uptime
    assert_cmd_rc host_context 0 "host context captured as a recorded fact (not a pass/fail criterion)"
}

# clean_tree <case-id> <repo-relative-path...>
#   Proves the production files under test are byte-identical to HEAD, so the
#   test run below exercises committed code, not another agent's dirty tree.
clean_tree() {
    local case_id="$1"; shift
    qa_cmd "$case_id" "production files under test are unmodified vs HEAD" \
        -- git -C "$QA_REPO_ROOT" diff --exit-code --stat HEAD -- "$@"
    assert_cmd_rc "$case_id" 0 "no uncommitted local edit to the files under test (§11.4.108: tested source == shipped source)"
}

# guard <case-id> <label> <pkg> <tags|""> <run-regex> <TestName...>
#   Executes the commit's own regression guards for real, then asserts each
#   named test actually reported PASS (defeats the empty-filter bluff).
guard() {
    local case_id="$1" label="$2" pkg="$3" tags="$4" runre="$5"; shift 5
    local -a cmd=(go test -race -count=2 -v -timeout=20m)
    [ -n "$tags" ] && cmd+=("-tags=${tags}")
    cmd+=(-run "$runre" "$pkg")
    qa_cmd "$case_id" "$label" -- "${cmd[@]}"
    assert_cmd_rc "$case_id" 0 "guard tests exit 0 under -race -count=2"
    local t
    for t in "$@"; do
        assert_cmd_output_contains "$case_id" "--- PASS: ${t}" \
            "guard ${t} genuinely RAN and passed (not an empty -run filter exiting 0)"
    done
    assert_cmd_output_not_contains "$case_id" "DATA RACE" "race detector reported no data race"
    assert_cmd_output_not_contains "$case_id" "no tests to run" "the -run filter matched at least one test"
}

# The renderer-overlap guards added by 9afc3da2 (HXC-158). Each drives a REAL
# Fyne renderer for 3.5 s and calls t.Fatalf when it observes fewer than two
# repaints, with the message "too little renderer/worker overlap for this guard
# to be meaningful". On a contended host the render loop is starved and that
# bail fires — which is an INCONCLUSIVE run reported as a FAIL, not a defect in
# the code under test (verified: the same guards pass on the same commit when
# host load is low; see transcripts/host_context.txt in each run).
LOAD_SENSITIVE_GUARDS='^Test(AuroraLLMTab_Background|AuroraDashboardTab_Ticker|HarmonyLLMTab_Background)WidgetMutationsAreUIThreadSafe$'

# _only_load_sensitive_failures <transcript>
#   TRUE only when every `--- FAIL: <name>` line in the transcript names one of
#   the load-sensitive guards AND the self-declared-inconclusive bail message is
#   present. Deliberately narrow: this must NEVER be able to excuse an
#   unrelated failing test, which would be the forbidden "scope it until it is
#   green" move. Any other failing test makes this false and the suite FAILs.
_only_load_sensitive_failures() {
    local t="$1" name
    grep -q 'too little renderer/worker overlap' "$t" || return 1
    grep -q '^--- FAIL: ' "$t" || return 1
    while IFS= read -r name; do
        printf '%s\n' "$name" | grep -qE "$LOAD_SENSITIVE_GUARDS" || return 1
    done < <(grep '^--- FAIL: ' "$t" | sed -E 's/^--- FAIL: ([^ ]+).*/\1/')
    return 0
}

# suite_or_scoped <case-id> <label> <pkg> <tags|"">
#   Runs the FULL package suite. On a quiet host that is the whole verdict.
#   If it fails and EVERY failing test is one of the self-declared-inconclusive
#   renderer-overlap guards, the run does NOT claim a pass and does NOT blame
#   the commit: it re-runs the package with exactly those guards excluded, and
#   records an explicit §11.4.3 SKIP naming what was therefore NOT certified —
#   so the run finishes INCOMPLETE (exit 2), never falsely green. Any other
#   failure is a real FAIL.
suite_or_scoped() {
    local case_id="$1" label="$2" pkg="$3" tags="$4"
    local -a cmd=(go test -race -count=1 -timeout=30m)
    [ -n "$tags" ] && cmd+=("-tags=${tags}")
    cmd+=("$pkg")
    qa_cmd "$case_id" "$label" -- "${cmd[@]}"
    if [ "$QA_LAST_RC" = "0" ]; then
        assert_cmd_rc "$case_id" 0 "full package suite exits 0 under -race"
        assert_cmd_output_not_contains "$case_id" "DATA RACE" "race detector reported no data race across the package"
        assert_cmd_output_not_contains "$case_id" "FAIL" "no test in the package failed"
        return
    fi
    if _only_load_sensitive_failures "$QA_LAST_TRANSCRIPT"; then
        local -a scoped=(go test -race -count=1 -timeout=30m -skip "$LOAD_SENSITIVE_GUARDS")
        [ -n "$tags" ] && scoped+=("-tags=${tags}")
        scoped+=("$pkg")
        qa_cmd "${case_id}_scoped" "package suite EXCLUDING the load-sensitive renderer-overlap guards" -- "${scoped[@]}"
        assert_cmd_rc "${case_id}_scoped" 0 "every remaining test in the package passes under -race"
        assert_cmd_output_not_contains "${case_id}_scoped" "DATA RACE" "race detector reported no data race in the scoped run"
        assert_cmd_output_not_contains "${case_id}_scoped" "FAIL" "no remaining test failed"
        qa_skip "${case_id}_renderer_guards" \
            "whole-package verdict INCLUDING the renderer-overlap guards (${LOAD_SENSITIVE_GUARDS})" \
            "host contention at capture time: those guards drive a real renderer and themselves declare the run inconclusive ('too little renderer/worker overlap for this guard to be meaningful') rather than reporting a defect. Recorded as SKIP per §11.4.3, never as a pass. Host load for this run is in transcripts/host_context.txt; the guards belong to HXC-158 and are unrelated to the commit under test."
        return
    fi
    # Genuine failure — record it as one.
    assert_cmd_rc "$case_id" 0 "full package suite exits 0 under -race"
    assert_cmd_output_not_contains "$case_id" "DATA RACE" "race detector reported no data race across the package"
    assert_cmd_output_not_contains "$case_id" "FAIL" "no test in the package failed"
}

# guard_load_sensitive <case-id> <label> <pkg> <tags|""> <run-regex> <TestName...>
#   Same as guard(), but a self-declared-inconclusive renderer bail is recorded
#   as a SKIP-with-reason instead of a FAIL. Used where the ONLY available
#   runtime proof is one of those guards.
guard_load_sensitive() {
    local case_id="$1" label="$2" pkg="$3" tags="$4" runre="$5"; shift 5
    local -a cmd=(go test -race -count=1 -v -timeout=20m)
    [ -n "$tags" ] && cmd+=("-tags=${tags}")
    cmd+=(-run "$runre" "$pkg")
    qa_cmd "$case_id" "$label" -- "${cmd[@]}"
    local t
    if [ "$QA_LAST_RC" = "0" ]; then
        assert_cmd_rc "$case_id" 0 "guard tests exit 0 under -race"
        for t in "$@"; do
            assert_cmd_output_contains "$case_id" "--- PASS: ${t}" \
                "guard ${t} genuinely RAN and passed (not an empty -run filter exiting 0)"
        done
        assert_cmd_output_not_contains "$case_id" "DATA RACE" "race detector reported no data race"
        return
    fi
    if grep -q 'too little renderer/worker overlap' "$QA_LAST_TRANSCRIPT"; then
        qa_skip "$case_id" "$label" \
            "host contention at capture time: the guard drives a real Fyne renderer and declares its own run inconclusive ('too little renderer/worker overlap for this guard to be meaningful'). An inconclusive run is a SKIP per §11.4.3, never a pass and never a defect attributed to the commit. Host load is in transcripts/host_context.txt; re-run on a quiet host to convert this SKIP into a verdict."
        return
    fi
    assert_cmd_rc "$case_id" 0 "guard tests exit 0 under -race"
}

# suite <case-id> <label> <pkg> <tags|"">
suite() {
    local case_id="$1" label="$2" pkg="$3" tags="$4"
    local -a cmd=(go test -race -count=1 -timeout=30m)
    [ -n "$tags" ] && cmd+=("-tags=${tags}")
    cmd+=("$pkg")
    qa_cmd "$case_id" "$label" -- "${cmd[@]}"
    assert_cmd_rc "$case_id" 0 "full package suite exits 0 under -race"
    assert_cmd_output_not_contains "$case_id" "DATA RACE" "race detector reported no data race across the package"
    assert_cmd_output_not_contains "$case_id" "FAIL" "no test in the package failed"
}

# notes <text...> — an accurate, run-shaped companion to the shared
# EVIDENCE.md preamble (which is worded for the HTTP-shaped harness).
notes() {
    cat > "${QA_RUN_DIR}/NOTES.md"
    echo "-- notes written: ${QA_RUN_DIR}/NOTES.md"
}

# finish — qa_finish, plus a correction of the shared preamble.
#
#   lib/sec_capture_lib.sh's qa_finish hardcodes an HTTP-shaped description
#   ("a real HTTP capture harness", "transcripts/<case-id>.http"). That is
#   accurate for the security captures the library was written for and WRONG
#   for these command-shaped runs, whose transcripts are `.txt` command
#   captures. Rather than edit a shared library that other agents' scripts are
#   using concurrently, qa_finish is run in a SUBSHELL — so its `exit` ends the
#   subshell, not this script — and an accurate description is appended to the
#   EVIDENCE.md it just wrote. Its exit status is preserved verbatim.
finish() {
    ( qa_finish )
    local rc=$?
    cat >> "${QA_RUN_DIR}/EVIDENCE.md" <<'EOC'

## Correction to the preamble above (§11.4.6)

The "How this evidence was produced" section above is boilerplate emitted by the
shared harness `scripts/qa/lib/sec_capture_lib.sh`, which was written for
HTTP-shaped captures. It is inaccurate for THIS run in two specifics, corrected
here rather than left to mislead:

* This is **not** an HTTP capture. No HTTP request is made and nothing is
  observed "on the wire". Every case is a real COMMAND execution
  (`scripts/qa/lib/cmd_capture_lib.sh`), captured with its full combined
  stdout+stderr and its real exit code.
* Transcripts are `transcripts/<case-id>.**txt**`, not `.http`. Each one records
  the command sent (`### command:`), the working directory, and everything the
  command returned — the bidirectional record §11.4.83 asks for, in
  command/response rather than request/response form.

What is accurate as written: no assertion can emit a PASS without a captured,
non-empty transcript carrying an exit-code trailer, and a SKIP is never counted
as a pass. See `NOTES.md` in this directory for what this specific run does and
does not certify.
EOC
    echo "-- preamble correction appended to ${QA_RUN_DIR}/EVIDENCE.md"
    exit "$rc"
}

usage() {
    cat <<'EOF'
Usage: scripts/qa/capture_racefix_wave.sh <case-key>

Case keys (one per §11.4.83-gate violation in the 2026-07-28/29 wave):
  persist_selectprio   dd3c0c3b  persistence autoSaveLoop select priority
  fable3               ed0cdcb8  3 defects from the independent Fable review
  aurora_underflow     640db264  aurora_os uint64 memory-freed underflow
  llm_deadlock         905a0b0a  llm StreamWithTools deadlock + send leaks
  select4              3c8197cf  4 unprioritized-select sites + teardown joins
  harmony_panic        99ff7d8e  harmony/aurora test constructions that panic
  shell_pipes          fa9f0247  shell stream pipes torn down by Cmd.Wait
  verifier_clock       048ca700  verifier cache TTL tests on an injected clock
  harmony_monitor      ccce0b77  harmony system monitor stop-on-channel
  fyneui_route         0a4df699  harmony/aurora widget updates via fyneui
  fyneui_docov         e1cde063  fyneui Do coverage + stopOnce bypass
EOF
}

case "${1:-}" in
    --list|-l|--help|-h|"") usage; exit 0 ;;
    --self-check)
        # §11.4.107(10): an analyzer that cannot FAIL is a bluff gate. This runs
        # the three source-assertion helpers against a golden-GOOD input (which
        # must produce zero FAILs) and then against a golden-BAD input (which
        # must produce FAILs). Both fixtures are REAL repository content, so
        # nothing has to be mutated in the shared tree to prove falsifiability.
        # Scratch run dir under /tmp — never lands in docs/qa.
        tmp="$(mktemp -d)"; QA_RUN_DIR="$tmp"; QA_RUN_ID="selfcheck"
        mkdir -p "$tmp/transcripts"; QA_VERDICTS="$tmp/verdicts.tsv"; : > "$QA_VERDICTS"

        echo "== golden-GOOD: every helper must report PASS"
        QA_PASS=0; QA_FAIL=0
        sig good_sig "a signature that IS in the tree" \
            'func (c *Cache) clock() time.Time' helix_code/internal/verifier/cache.go
        absent good_absent "a bare-close bypass that is NOT in the tree" \
            't.Cleanup(func() { close(da.stopUpdate) })' helix_code/applications/desktop/
        absent_code good_abscode "an executable time.Sleep( that is NOT in the tree" \
            'time.Sleep(' helix_code/internal/verifier/cache_test.go
        good_pass=$QA_PASS; good_fail=$QA_FAIL

        echo "== golden-BAD: every helper must report FAIL (the anti-bluff proof)"
        QA_PASS=0; QA_FAIL=0
        sig bad_sig "a signature that is NOT in the tree — sig must FAIL" \
            'func (c *Cache) thisSignatureDoesNotExist() time.Time' \
            helix_code/internal/verifier/cache.go
        absent bad_absent "a pattern that IS in the tree — absent must FAIL" \
            't.Cleanup(func() { da.stopOnce.Do(func() { close(da.stopUpdate) }) })' \
            helix_code/applications/desktop/
        absent_code bad_abscode "a non-comment line that IS in the tree — absent_code must FAIL" \
            'func (c *Cache) clock() time.Time' helix_code/internal/verifier/cache.go
        bad_pass=$QA_PASS; bad_fail=$QA_FAIL

        echo
        echo "golden-GOOD: PASS=${good_pass} FAIL=${good_fail}   (expect FAIL=0)"
        echo "golden-BAD : PASS=${bad_pass} FAIL=${bad_fail}   (expect FAIL>=3 — one per helper)"
        rm -rf "$tmp"
        if [ "$good_fail" -eq 0 ] && [ "$bad_fail" -ge 3 ]; then
            echo "SELF-CHECK PASS: the source assertions genuinely discriminate good from bad."
            exit 0
        fi
        echo "SELF-CHECK FAIL: the assertion engine does not discriminate — do not trust its verdicts." >&2
        exit 1
        ;;
esac
KEY="$1"

# ---------------------------------------------------------------------------
# Per-commit case matrices
# ---------------------------------------------------------------------------
case "$KEY" in

persist_selectprio)
    qa_init "racefix_persist_selectprio" "dd3c0c3b8a8e098496a44b77cca013cb22c40666" \
        "persistence autoSaveLoop: stop always wins over a pending tick (§11.4.50)"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/internal/persistence/store.go
    sig sig_tickfn "autoSaveTick was extracted as a testable, stop-prioritised step" \
        'func (s *Store) autoSaveTick(tickerC <-chan time.Time, stop <-chan struct{}) bool' \
        helix_code/internal/persistence/store.go
    guard guard_selectprio "the commit's own regression guard, -race -count=2" \
        ./internal/persistence "" '^TestAutoSaveTick_StopAlwaysWinsOverPendingTick$' \
        TestAutoSaveTick_StopAlwaysWinsOverPendingTick
    suite suite_persistence "full internal/persistence package under -race" ./internal/persistence ""
    notes <<'EON'
# What this run certifies

`dd3c0c3b` replaced an unprioritised two-case `select` in the persistence
auto-save loop with an explicit stop-first probe, extracted into
`(*Store).autoSaveTick` so the ordering is directly testable. Go's `select`
picks uniformly at random among ready cases, so with both a pending tick and a
closed stop channel the loop could perform another save AFTER being told to
stop.

Captured here (real command execution, transcripts/ holds every byte):
* the extracted function is present in the tracked source that ships today;
* the commit's own guard `TestAutoSaveTick_StopAlwaysWinsOverPendingTick`
  actually RAN (asserted on `--- PASS:` lines, not merely on exit 0) and passed
  twice under `-race`;
* the whole package is race-clean.

Not certified: nothing about the wider auto-save schedule under real workload —
this is a select-ordering guarantee, not a durability claim.
EON
    finish
    ;;

fable3)
    qa_init "racefix_fable3" "ed0cdcb861631d84b22b258b6c873507f729b933" \
        "3 defects from the independent Fable review: autosave residual window, served-model reporting"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/internal/persistence/store.go helix_code/internal/server/llm_generate.go
    sig sig_racehook "the residual-window race hook seam exists in the shipped store" \
        'autoSaveTickRaceHook' helix_code/internal/persistence/store.go
    sig sig_servedmodel "the server reports the model that ACTUALLY served the request" \
        'servedModel := llmReq.Model' helix_code/internal/server/llm_generate.go
    guard guard_residual "persistence residual-window guard, -race -count=2" \
        ./internal/persistence "" '^TestAutoSaveTick_ResidualWindowStopRaceDuringBlockingSelect$' \
        TestAutoSaveTick_ResidualWindowStopRaceDuringBlockingSelect
    guard guard_servedmodel "server served-model regression guards, -race -count=2" \
        ./internal/server "" '^TestGenerateLLM_ResponseModel_(ServedNotRequested|FallsBackWhenProviderOmitsIt)$' \
        TestGenerateLLM_ResponseModel_ServedNotRequested \
        TestGenerateLLM_ResponseModel_FallsBackWhenProviderOmitsIt
    suite suite_persistence "full internal/persistence package under -race" ./internal/persistence ""
    notes <<'EON'
# What this run certifies

`ed0cdcb8` landed three fixes found by an independent review. Two are covered
here by their own named guards: the auto-save residual window (a stop that
arrives while the loop is already parked in the blocking select) and the
server's response `model` field, which reported the REQUESTED model rather than
the one that actually served the request.

Scope note (§11.4.6): the full `internal/server` suite is NOT run here. That
package was under concurrent edit by other agents during this capture, so a
whole-package verdict would report their in-flight state, not this commit's.
The server half is therefore certified by its two NAMED guards, executed under
`-race -count=2` with `--- PASS:` asserted per test; the persistence half is
certified by both its named guard and the full package suite.
EON
    finish
    ;;

aurora_underflow)
    qa_init "racefix_aurora_underflow" "640db2649616bc13428fbdcf4e4c0d83715fa2d5" \
        "aurora_os: uint64 underflow in the Optimize-performance memory-freed diagnostic"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/applications/aurora_os/mem_stats.go \
        helix_code/applications/aurora_os/main.go helix_code/applications/aurora_os/main_nogui.go
    sig sig_memfreed "the subtraction now happens in float64, so a growth cannot wrap" \
        'func memFreedMB(before, after uint64) float64' \
        helix_code/applications/aurora_os/mem_stats.go
    guard guard_underflow "the commit's four boundary guards, -race -count=2" \
        ./applications/aurora_os ci \
        '^TestMemFreedMB_(NoUint64Underflow|ShrinkYieldsPositive|NoChange|FormattingHandlesNegativeDelta)$' \
        TestMemFreedMB_NoUint64Underflow TestMemFreedMB_ShrinkYieldsPositive \
        TestMemFreedMB_NoChange TestMemFreedMB_FormattingHandlesNegativeDelta
    suite_or_scoped suite_aurora "full applications/aurora_os package under -race" ./applications/aurora_os ci
    notes <<'EON'
# What this run certifies

`640db264` fixed an unsigned wrap: `before.Alloc - after.Alloc` on two `uint64`
values underflows to a number near 2^64 whenever the heap GREW across the
measured window, so the "memory freed" diagnostic could report roughly 1.8e13 MB
instead of a small negative delta. The subtraction now converts to `float64`
first, in a single named helper shared by the GUI and no-GUI entry points.

Captured here: the helper's signature in the shipped source, and all four of the
commit's boundary guards executed for real (grow / shrink / no-change /
formatting) with `--- PASS:` asserted per test, plus the package under `-race`.
EON
    finish
    ;;

llm_deadlock)
    qa_init "racefix_llm_deadlock" "905a0b0afbd97bdb622bc8a737a43cba3d6370af" \
        "llm: guaranteed StreamWithTools deadlock + the provider send-leak family"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/internal/llm/anthropic_provider.go \
        helix_code/internal/llm/tool_provider.go helix_code/internal/llm/azure_provider.go \
        helix_code/internal/llm/bedrock_provider.go helix_code/internal/llm/ensemble_provider.go \
        helix_code/internal/llm/gemini_provider.go helix_code/internal/llm/groq_provider.go \
        helix_code/internal/llm/vertexai_provider.go
    sig sig_anthropic "the Anthropic stream parser is a ctx-aware function, not an inline send loop" \
        'func (ap *AnthropicProvider) parseStreamingResponse(ctx context.Context, body io.Reader, ch chan<- LLMResponse, requestID uuid.UUID) error' \
        helix_code/internal/llm/anthropic_provider.go
    sig sig_azure "the Azure SSE stream parser carries the same ctx-aware shape" \
        'func (ap *AzureProvider) parseSSEStream(ctx context.Context, reader io.Reader, ch chan<- LLMResponse, requestID uuid.UUID) error' \
        helix_code/internal/llm/azure_provider.go
    guard guard_deadlock "the commit's deadlock + goroutine-leak guards, -race -count=2" \
        ./internal/llm "" \
        '^(TestStreamWithTools_ManyChunks_DoesNotDeadlock|TestProviderGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain)$' \
        TestStreamWithTools_ManyChunks_DoesNotDeadlock \
        TestProviderGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain
    suite suite_llm "full internal/llm package under -race" ./internal/llm ""
    notes <<'EON'
# What this run certifies

`905a0b0a` closed a deadlock that was guaranteed rather than probabilistic —
`StreamWithTools` collected a provider stream into a channel nothing was draining
past its buffer — and the sibling class across eight providers, where a stream
goroutine blocked forever on `ch <- …` after its consumer went away, leaking a
goroutine per abandoned request. Every send is now paired with `case <-ctx.Done()`.

Captured here: the ctx-aware parser signatures in the shipped source of two
providers, both named guards executed under `-race -count=2` with `--- PASS:`
asserted per test, and the whole `internal/llm` package race-clean.

Not certified: no live provider is contacted by this run. The guards drive the
stream machinery with in-process fakes, which is what makes them deterministic;
the wire behaviour of a real provider is a separate evidence class.
EON
    finish
    ;;

select4)
    qa_init "racefix_select4" "3c8197cfba3b50b08383d7471abebabad81d86f0" \
        "desktop/harmony/aurora/discovery: teardown joins + 4 unprioritized-select sites"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/applications/aurora_os/main.go \
        helix_code/applications/desktop/main.go helix_code/applications/harmony_os/main.go \
        helix_code/internal/discovery/client.go
    sig sig_aurora_tick "aurora's update loop step is extracted and stop-prioritised" \
        'func (auroraApp *AuroraApp) updateLoopTick(tickerC <-chan time.Time, stop <-chan struct{}) bool' \
        helix_code/applications/aurora_os/main.go
    sig sig_updatedone "the update goroutine publishes a done channel the teardown can join" \
        'updateDone chan struct{}' helix_code/applications/aurora_os/main.go
    guard guard_aurora "aurora select-priority + join guards" ./applications/aurora_os ci \
        '^TestAuroraApp_(UpdateLoopTick_StopAlwaysWinsOverPendingTick|UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect|Close_JoinsUpdateLoopBeforeReturning)$' \
        TestAuroraApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick \
        TestAuroraApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect \
        TestAuroraApp_Close_JoinsUpdateLoopBeforeReturning
    guard guard_harmony "harmony select-priority + join guards" ./applications/harmony_os ci \
        '^TestHarmonyApp_(UpdateLoopTick_StopAlwaysWinsOverPendingTick|UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect|Cleanup_JoinsUpdateLoopBeforeReturning)$' \
        TestHarmonyApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick \
        TestHarmonyApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect \
        TestHarmonyApp_Cleanup_JoinsUpdateLoopBeforeReturning
    guard guard_desktop "desktop select-priority + join guards" ./applications/desktop ci \
        '^TestDesktopApp_(UpdateLoopTick_StopAlwaysWinsOverPendingTick|UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect|Close_JoinsUpdateLoopBeforeReturning)$' \
        TestDesktopApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick \
        TestDesktopApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect \
        TestDesktopApp_Close_JoinsUpdateLoopBeforeReturning
    guard guard_discovery "discovery answer-beats-spurious-timeout guards" ./internal/discovery "" \
        '^TestDiscoverTimeoutSelect_(ResultAlwaysWinsOverSpuriousTimeout|ErrorAlwaysWinsOverSpuriousTimeout)$' \
        TestDiscoverTimeoutSelect_ResultAlwaysWinsOverSpuriousTimeout \
        TestDiscoverTimeoutSelect_ErrorAlwaysWinsOverSpuriousTimeout
    suite suite_discovery "full internal/discovery package under -race" ./internal/discovery ""
    suite_or_scoped suite_aurora "full applications/aurora_os package under -race" ./applications/aurora_os ci
    suite_or_scoped suite_harmony "full applications/harmony_os package under -race" ./applications/harmony_os ci
    suite suite_desktop "full applications/desktop package under -race" ./applications/desktop ci
    notes <<'EON'
# What this run certifies

`3c8197cf` fixed the same defect shape in four places: a two-case `select` where
a real answer and a spurious signal were both ready, so Go's uniform random
choice could discard the answer (three UI update loops discarding "stop", one
discovery call discarding a delivered result in favour of its own timeout). It
also gave the three UI loops a done channel so teardown JOINS the goroutine
instead of merely asking it to stop.

Captured here: the extracted stop-prioritised step and the done channel in the
shipped source, eleven named guards executed under `-race -count=2` with
`--- PASS:` asserted per test, and all four affected packages race-clean.
EON
    finish
    ;;

harmony_panic)
    qa_init "racefix_harmony_panic" "99ff7d8e15d06eec3589c0144e02706d85729dcd" \
        "harmony_os/aurora_os: test constructions that panic on first execution"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/applications/harmony_os/main.go
    sig sig_nilguard "the API refresh path degrades to local data instead of dereferencing a nil client" \
        'log.Printf("API tasks unavailable, using local: %v", err)' \
        helix_code/applications/harmony_os/main.go
    guard guard_aurora "the aurora guard this commit repaired, -race -count=2" \
        ./applications/aurora_os ci \
        '^TestAuroraApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick$' \
        TestAuroraApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick
    suite_or_scoped suite_harmony "full applications/harmony_os package under -race" ./applications/harmony_os ci
    suite_or_scoped suite_aurora "full applications/aurora_os package under -race" ./applications/aurora_os ci
    notes <<'EON'
# What this run certifies

`99ff7d8e` repaired guards that panicked on their FIRST execution — a test that
cannot run is not a guard. The production half added a nil-client guard on
harmony's API refresh path so it degrades to local data rather than panicking.

Captured here: the nil-guard's log line in the shipped source, the repaired
aurora guard executed under `-race -count=2` with `--- PASS:` asserted, and both
application packages green under `-race` — which is the direct falsification of
"panics on first execution".
EON
    finish
    ;;

shell_pipes)
    qa_init "racefix_shell_pipes" "fa9f0247fef65e190e189991ddddf7e4c4e62f55" \
        "tools/shell: Cmd.Wait tore down the stream pipes before the scanners read them"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/internal/tools/shell/executor.go \
        helix_code/internal/tools/shell/output.go
    sig sig_pipes "parent-owned os.Pipe files replace StdoutPipe/StderrPipe, which Cmd.Wait closes" \
        'func newStreamPipes(execCmd *exec.Cmd) (*streamPipes, error)' \
        helix_code/internal/tools/shell/executor.go
    sig sig_scannererr "the scanner's read error is recorded instead of being read as clean EOF" \
        'streamer.Err()' helix_code/internal/tools/shell/executor.go
    guard guard_pipes "the commit's three regression guards, -race -count=2" \
        ./internal/tools/shell "" \
        '^(TestStreamPipesSurviveCmdWait|TestOutputStreamerSurfacesReadError|TestExecuteStreamDeliversAllOutputBeforeDone)$' \
        TestStreamPipesSurviveCmdWait TestOutputStreamerSurfacesReadError \
        TestExecuteStreamDeliversAllOutputBeforeDone
    suite suite_shell "full internal/tools/shell package under -race" ./internal/tools/shell ""
    notes <<'EON'
# What this run certifies

`fa9f0247` closed a silent-empty-output defect: `ExecuteStream` could return exit
code 0 with ZERO captured output. `Cmd.Wait` closes the parent's read ends of
`StdoutPipe`/`StderrPipe`, and it was being called concurrently with the
scanners, so for a fast-exiting command Wait routinely won the race. `bufio.Scanner`
then reported that hard read error exactly as it reports clean EOF, and the
discarded `scanner.Err()` turned an I/O failure into a perfectly green empty
result — the exact shape of a §11.4 PASS-bluff at the tooling layer.

Captured here: the parent-owned pipe constructor and the surfaced scanner error
in the shipped source, and all three named guards executed under `-race -count=2`
with `--- PASS:` asserted per test.
EON
    finish
    ;;

verifier_clock)
    qa_init "racefix_verifier_clock" "048ca7006da09c733cd03151121198624ed7975c" \
        "verifier cache: sleep-driven TTL boundary tests replaced by an injected clock (§11.4.50)"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/internal/verifier/cache.go
    sig sig_clock "the cache reads time through an injectable clock with a time.Now fallback" \
        'func (c *Cache) clock() time.Time' helix_code/internal/verifier/cache.go
    sig sig_clockuse "the TTL comparison itself goes through that clock" \
        'if c.clock().Sub(entry.FetchedAt) > c.ttl {' helix_code/internal/verifier/cache.go
    absent_code absent_sleep "no executable time.Sleep survives in the cache tests" \
        'time.Sleep(' helix_code/internal/verifier/cache_test.go
    guard guard_boundary "the TTL/stale boundary tests, now exact, -race -count=2" \
        ./internal/verifier "" \
        '^(TestCache_GetModels_Hit|TestCache_GetModelsStale_SlightlyExpired|TestStress_Cache_Boundary)$' \
        TestCache_GetModels_Hit TestCache_GetModelsStale_SlightlyExpired TestStress_Cache_Boundary
    suite suite_verifier "full internal/verifier package under -race" ./internal/verifier ""
    notes <<'EON'
# What this run certifies

`048ca700` removed a determinism hazard: the cache TTL and stale-window tests
asserted expiry by sleeping past an approximate wall-clock window. `time.Sleep`
guarantees only a LOWER bound on elapsed time, so under host load the verdict
became a function of machine load rather than of the code under test (§11.4.50).

This is a STRENGTHENING, not a weakening, and this run proves it as a fact
rather than repeating the claim: `git grep` finds NO surviving `time.Sleep` in
`cache_test.go`, the injectable clock is in the shipped source, and the boundary
tests — which can now assert at exactly `ttl` and at `ttl + 1ns`, a boundary no
sleep can express — run green under `-race -count=2`.

Scope: this commit changed a test seam. The production behaviour is unchanged by
construction — `clock()` falls back to `time.Now` when the field is unset, so a
zero-value Cache behaves exactly as before.
EON
    finish
    ;;

harmony_monitor)
    qa_init "racefix_harmony_monitor" "ccce0b77a25e5cd3d84e8bc5b40b7c467d54320b" \
        "harmony_os: system monitor stops on a channel, not an unsynchronised bool"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/applications/harmony_os/main.go
    sig sig_accessor "the monitoring flag is reached only through mutex-guarded accessors" \
        'func (m *HarmonySystemMonitor) setMonitoring(v bool) {' \
        helix_code/applications/harmony_os/main.go
    sig sig_monitordone "the monitor publishes a done channel the teardown joins" \
        'monitorDone chan struct{}' helix_code/applications/harmony_os/main.go
    sig sig_tickorstop "the loop stops on the app's stop channel via the shared primitive" \
        'fyneui.TickOrStop(' helix_code/applications/harmony_os/main.go
    guard guard_cleanup "TestCleanup — the test whose RED baseline this commit fixed, -race -count=2" \
        ./applications/harmony_os ci '^TestCleanup$' TestCleanup
    suite_or_scoped suite_harmony "full applications/harmony_os package under -race" ./applications/harmony_os ci
    qa_skip metric_race_runtime \
        "runtime exercise of the SECOND half of this commit — the five system-metric fields (cpu/mem/gpu/temperature/power) now guarded by a mutex between updateSystemMetrics and createHarmonySystemTab" \
        "hardware_not_present-class gap in the test tree, not in the fix: NO test in applications/harmony_os constructs the system tab, so the reader side never runs and the race detector never observes that pair. The commit says so itself. Certified here by inspection + the accessor signatures only; a runtime proof needs a test that builds the tab, which does not exist yet."
    notes <<'EON'
# What this run certifies — and what it explicitly does NOT

`ccce0b77` had two halves.

FIRST HALF (certified at runtime here): `monitorSystem` looped on
`for app.systemMonitor.monitoring`, a plain cross-goroutine bool read, while
`Cleanup()` wrote that bool from another goroutine — a data race whose store may
never be observed, so the monitor could outlive the teardown that told it to
stop. The commit's own RED baseline was `TestCleanup` failing with
`WARNING: DATA RACE`. That test is executed here under `-race -count=2` and is
asserted to have genuinely run (`--- PASS: TestCleanup`), and the whole package
is race-clean.

SECOND HALF (NOT certified — recorded as a SKIP, §11.4.3): the five metric fields
shared between the monitor goroutine and the system tab were also unsynchronised
and are now mutex-guarded. No test in this package constructs the system tab, so
nothing exercises the reader side; the detector cannot see a pair it never runs.
This run therefore certifies the accessors EXIST in the shipped source and
nothing more about that half. This run is INCOMPLETE (exit 2) for exactly that
reason — a SKIP is never counted as a pass.
EON
    finish
    ;;

fyneui_route)
    qa_init "racefix_fyneui_route" "0a4df699b32df9e0eb74c54aee08c52dd90c5765" \
        "harmony_os/aurora_os: background widget updates routed through internal/fyneui"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/applications/harmony_os/main.go \
        helix_code/applications/aurora_os/main.go
    sig sig_aurora_dispatch "aurora dispatches label updates onto the UI thread" \
        'fyneui.Do(func() { systemStatsLabel.SetText(systemText) })' \
        helix_code/applications/aurora_os/main.go
    sig sig_aurora_tickorstop "aurora's ticker loops honour the app's stop channel" \
        'fyneui.TickOrStop(ticker.C, auroraApp.stopUpdate)' \
        helix_code/applications/aurora_os/main.go
    sig sig_harmony_import "harmony imports the shared UI-thread package" \
        '"dev.helix.code/internal/fyneui"' \
        helix_code/applications/harmony_os/main.go
    suite_or_scoped suite_harmony "full applications/harmony_os package under -race" ./applications/harmony_os ci
    suite_or_scoped suite_aurora "full applications/aurora_os package under -race" ./applications/aurora_os ci
    # RUNTIME PROOF OF THE DISPATCH SITES.
    #
    # The commit itself recorded an honest gap: "No test in either package
    # constructs the tabs, so the workers never start and the detector never
    # observes them." That statement was true when the commit landed and is NOW
    # OBSOLETE — 9afc3da2 (HXC-158, "make the fyneui thread-affinity fix
    # falsifiable") added guards that really do build the tabs
    # (gui_thread_race_test.go calls auroraApp.createAuroraDashboardTab() and
    # drives the production TickOrStop loop) and watch the renderer repaint
    # while the worker mutates the widgets. Those guards are therefore the
    # runtime proof this commit's own message said did not exist, and they are
    # run here rather than skipped. They are load-sensitive by construction, so
    # an inconclusive run is recorded as SKIP-with-reason, never as a pass.
    guard_load_sensitive dispatch_runtime_aurora \
        "aurora dashboard + LLM tab dispatch sites, driven with a real renderer" \
        ./applications/aurora_os ci \
        '^Test(AuroraLLMTab_Background|AuroraDashboardTab_Ticker)WidgetMutationsAreUIThreadSafe$' \
        TestAuroraLLMTab_BackgroundWidgetMutationsAreUIThreadSafe \
        TestAuroraDashboardTab_TickerWidgetMutationsAreUIThreadSafe
    guard_load_sensitive dispatch_runtime_harmony \
        "harmony LLM tab dispatch sites, driven with a real renderer" \
        ./applications/harmony_os ci \
        '^TestHarmonyLLMTab_BackgroundWidgetMutationsAreUIThreadSafe$' \
        TestHarmonyLLMTab_BackgroundWidgetMutationsAreUIThreadSafe
    notes <<'EON'
# What this run certifies — and what it explicitly does NOT

`0a4df699` routed every background widget mutation in the harmony_os and
aurora_os Fyne apps through `internal/fyneui`, and gave three `for range
ticker.C` loops — which could never terminate and leaked for the process
lifetime — the app's existing stop channel.

CERTIFIED HERE: the dispatch sites and the stop-honouring loops are present in
the tracked source that ships today; both packages are green and race-clean
under `-race`; and — the part that actually matters — the dispatch sites are
RUNTIME-exercised with the race detector watching.

That last point is a correction to the commit's own honest caveat. `0a4df699`
said "No test in either package constructs the tabs, so the workers never start
and the detector never observes them", and noted aurora measured zero races
BEFORE the change for exactly that reason. That was true then. It is no longer
true: `9afc3da2` (HXC-158) added guards that build the real tabs, drive the
production `TickOrStop` loop, and watch a real renderer repaint while the worker
mutates the widgets. This run executes those guards rather than repeating the
commit's stale caveat — the gap the commit flagged has since been closed by
other work, and evidence that ignored that would understate what is provable
today.

Load sensitivity (§11.4.50): those guards need genuine renderer/worker overlap
and call `t.Fatalf` when they observe fewer than two repaints in 3.5 s, which a
contended host can cause. That is an INCONCLUSIVE run, not a defect, so it is
recorded as a SKIP with the host load attached (`transcripts/host_context.txt`)
and never as a pass. Check the verdict table above to see which applied to THIS
run.
EON
    finish
    ;;

fyneui_docov)
    qa_init "racefix_fyneui_docov" "e1cde063586961880c899bea49faaee5bfaa0dfd" \
        "fyneui: false test comment corrected, Do lock path covered, stopOnce bypass closed"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/internal/fyneui/uithread.go
    sig sig_limitation "the package doc records the DoAndWait/shutdown limitation instead of implying none" \
        '# Known limitation: DoAndWait across driver shutdown' \
        helix_code/internal/fyneui/uithread.go
    sig sig_docoverage "the Do lock path now has its own direct guard" \
        'func TestDoSerializesAgainstSync' helix_code/internal/fyneui/uithread_test.go
    sig sig_stoponce "the desktop test teardown now routes through the same sync.Once Close() uses" \
        't.Cleanup(func() { da.stopOnce.Do(func() { close(da.stopUpdate) }) })' \
        helix_code/applications/desktop/gui_record_features_test.go
    absent absent_bypass "no test closes stopUpdate directly, bypassing the sync.Once in Close()" \
        't.Cleanup(func() { close(da.stopUpdate) })' \
        helix_code/applications/desktop/
    guard guard_uithread "both UI-thread serialisation guards, -race -count=2" \
        ./internal/fyneui ci '^(TestDoSerializesAgainstSync|TestSyncSerializesAgainstDo)$' \
        TestDoSerializesAgainstSync TestSyncSerializesAgainstDo
    suite suite_fyneui "full internal/fyneui package under -race" ./internal/fyneui ci
    suite suite_desktop "full applications/desktop package under -race" ./applications/desktop ci
    notes <<'EON'
# What this run certifies

`e1cde063` closed three Minor findings from the independent review of `e879702c`:
a test comment that stated the opposite of the truth, a coverage gap where `Do`'s
lock path was pinned by no assertion, and a test that closed `stopUpdate`
directly, sidestepping the `sync.Once` that `Close()` uses (one edit away from a
"close of closed channel" panic).

Captured here: the new `TestDoSerializesAgainstSync` guard exists and RUNS (both
serialisation guards asserted on their `--- PASS:` lines under `-race -count=2`),
the `sync.Once` bypass is provably gone from the desktop test (`git grep` finds
no surviving occurrence), and both affected packages are race-clean.

Honest note carried forward from the commit: `-race`, and only `-race`, is what
pins these locks. The count assertion alone still passes with the locking
stripped, because every increment runs on a single worker goroutine under the
Fyne test driver. That is stated in the test itself rather than papered over.
EON
    finish
    ;;

*)
    echo "capture_racefix_wave.sh: unknown case key: ${KEY}" >&2
    usage >&2
    exit 2
    ;;
esac
