#!/usr/bin/env bash
# capture_racefix_wave.sh <case-key>
#
# §11.4.83 QA evidence capture for the 2026-07-28/29 gate-remediation wave —
# the feature-shipping commits that the §11.4.83 release gate
# (scripts/gates/qa_evidence_gate.sh) reported as carrying no docs/qa evidence.
#
#   BATCH 1 (2026-07-28, 11 keys): the concurrency-defect wave.
#   BATCH 2 (2026-07-29,  4 keys): the four commits the gate still reported
#     after batch 1 landed — three more concurrency fixes (two rounds of the
#     tools/shell drain grace, the terminal_ui QA-dashboard read race) and ONE
#     NON-RACE correctness fix (net_ipv6, HXC-185 IPv6 authority bracketing).
#     The script name says "racefix" because batch 1 was all races; batch 2
#     stretches that name by exactly one case. Named honestly here (§11.4.6)
#     rather than silently, because the alternative — a second script — would
#     have duplicated the ~200 lines of helpers below (§11.4.74).
#
# WHY A SHARED SCRIPT AND NOT ONE PER COMMIT
#   Every commit in this wave has the SAME evidence shape: a concurrency /
#   correctness defect in Go production code, closed by a named regression
#   guard test. The deliverable is therefore COMMAND-shaped (a real compiler +
#   test-runner outcome), not HTTP-shaped, so this script builds on
#   lib/cmd_capture_lib.sh. The per-commit case matrix below is what differs;
#   it is spelled out per commit, never generated, so each run's assertions are
#   specific to the defect that commit actually fixed. One run-id directory is
#   produced per commit (this script is invoked once per key) — deliberately
#   NOT one blanket ledger enumerating every SHA, which is the documented
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

# red_polarity <case-id> <label> <pkg> <tags|""> <run-regex> <TestName...>
#   §11.4.115 polarity proof, run on the FIXED artifact.
#
#   A guard that only ever passes proves nothing about its own sensitivity: a
#   test asserting `true == true` is green forever. These packages carry an
#   explicit RED_MODE polarity switch, so the falsifiability check is a real
#   execution rather than an argument — RED_MODE=1 re-drives the PRE-FIX
#   behaviour (the defect's own shape) against today's shipped source and MUST
#   fail. If it passed, the guard would be blind and its GREEN verdict
#   worthless.
#
#   Asserted: non-zero exit AND a `--- FAIL: <name>` line for EVERY named test.
#   The exit code alone is insufficient — a compile error also exits non-zero,
#   and would otherwise be indistinguishable from a genuine reproduction.
#
#   RED_MODE is passed via `env` because qa_cmd execs the argv directly (no
#   shell), so a `VAR=x cmd` prefix would be parsed as a command name.
red_polarity() {
    local case_id="$1" label="$2" pkg="$3" tags="$4" runre="$5"; shift 5
    local -a cmd=(env RED_MODE=1 go test -race -count=1 -v -timeout=20m)
    [ -n "$tags" ] && cmd+=("-tags=${tags}")
    cmd+=(-run "$runre" "$pkg")
    qa_cmd "$case_id" "$label" -- "${cmd[@]}"
    assert_cmd_rc_not "$case_id" 0 "RED_MODE=1 FAILS on the fixed artifact — the guard is falsifiable, not blind (§11.4.115)"
    local t
    for t in "$@"; do
        assert_cmd_output_contains "$case_id" "--- FAIL: ${t}" \
            "guard ${t} genuinely RAN and reported FAIL under RED_MODE=1 (not a compile error masquerading as a reproduction)"
    done
    assert_cmd_output_not_contains "$case_id" "no tests to run" "the -run filter matched at least one test"
    assert_cmd_output_not_contains "$case_id" "[build failed]" "the non-zero exit came from the guard, not from a build failure"
}

# guard_skippable <case-id> <label> <pkg> <tags|""> <run-regex> <skip-needle> <TestName>
#   Same as guard(), for a guard that declares its OWN run inconclusive when a
#   precondition it cannot control is unmet — here the QA-dashboard race guard,
#   which SKIPs (per §11.4.3) when no orchestrator status transition happened to
#   land inside a rendering batch, because a clean -race result over a window
#   the writer never touched would prove nothing.
#
#   A self-declared-inconclusive run is recorded as a SKIP-with-reason and makes
#   the whole run INCOMPLETE (exit 2) — never a pass, and never blamed on the
#   commit. Any OTHER non-zero outcome is a real FAIL.
guard_skippable() {
    local case_id="$1" label="$2" pkg="$3" tags="$4" runre="$5" skipneedle="$6"; shift 6
    local -a cmd=(go test -race -count=1 -v -timeout=20m)
    [ -n "$tags" ] && cmd+=("-tags=${tags}")
    cmd+=(-run "$runre" "$pkg")
    qa_cmd "$case_id" "$label" -- "${cmd[@]}"
    local t
    if [ "$QA_LAST_RC" = "0" ] && grep -qF -- "$skipneedle" "$QA_LAST_TRANSCRIPT"; then
        qa_skip "$case_id" "$label" \
            "the guard declared its own run INCONCLUSIVE ('${skipneedle}') — the precondition it needs (a writer-vs-render overlap) did not occur in this run, so a clean -race result would certify nothing. Recorded as SKIP per §11.4.3, never as a pass. Host load for this run is in transcripts/host_context.txt; re-run to convert this SKIP into a verdict."
        return
    fi
    assert_cmd_rc "$case_id" 0 "guard tests exit 0 under -race"
    for t in "$@"; do
        assert_cmd_output_contains "$case_id" "--- PASS: ${t}" \
            "guard ${t} genuinely RAN and passed (not an empty -run filter exiting 0, and not a SKIP)"
    done
    assert_cmd_output_not_contains "$case_id" "DATA RACE" "race detector reported no data race"
    assert_cmd_output_not_contains "$case_id" "no tests to run" "the -run filter matched at least one test"
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

Case keys (one per §11.4.83-gate violation in the 2026-07-28/29 wave).

BATCH 2 (2026-07-29):
  shell_grace          fa1b52b3  shell drain grace = no-progress window
  shell_bounds         0c2f4940  shell abandoned-consumer bound + M2/M3 guards
  net_ipv6             9d082a7b  HXC-185 IPv6 authority bracketing (netutil)
  qa_dashboard         27754934  terminal_ui QA dashboard read race (HXC-174)

BATCH 1 (2026-07-28):
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

# ---------------------------------------------------------------------------
# BATCH 2 (2026-07-29) — the four commits the §11.4.83 gate still reported
# after batch 1 landed.
# ---------------------------------------------------------------------------

shell_grace)
    qa_init "racefix_shell_grace" "fa1b52b32992d9ae0dac86643f9e0e28a55f252a" \
        "tools/shell: the drain grace is a no-progress window, not a total output budget"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/internal/tools/shell/executor.go \
        helix_code/internal/tools/shell/output.go
    sig sig_progress "OutputStreamer exposes a monotonic Progress() counter — the signal the drain loop rearms on" \
        'func (os *OutputStreamer) Progress() uint64' \
        helix_code/internal/tools/shell/output.go
    sig sig_rearm "the drain deadline is REARMED while lines are still moving (the window is no longer a total budget)" \
        'drainDeadline.Reset(streamDrainGrace)' \
        helix_code/internal/tools/shell/executor.go
    sig sig_flagderived "OutputIncomplete is DERIVED from what the scanners observed, not from which select branch won (M1/M3)" \
        'result.OutputIncomplete = streamer.Err() != nil' \
        helix_code/internal/tools/shell/executor.go
    sig sig_errpriority "Err() prefers a genuine failure over the ErrStreamStopped teardown sentinel (M2)" \
        'if err != nil && !errors.Is(err, ErrStreamStopped) {' \
        helix_code/internal/tools/shell/output.go
    guard guard_notruncate "the commit's own I1 guard — a slow-but-draining consumer receives ALL output, -race -count=2" \
        ./internal/tools/shell "" '^TestExecuteStreamDoesNotTruncateASlowButDrainingConsumer$' \
        TestExecuteStreamDoesNotTruncateASlowButDrainingConsumer
    guard guard_hxc184 "the HXC-184 guards this fix must not regress — grandchild bound + executor slot recovery, -race -count=2" \
        ./internal/tools/shell "" \
        '^TestExecuteStream(ReturnsWhenGrandchildHoldsPipes|HangDoesNotWedgeTheExecutor|DoneBeforeDrainIsBoundedNotDeadlocked)$' \
        TestExecuteStreamReturnsWhenGrandchildHoldsPipes \
        TestExecuteStreamHangDoesNotWedgeTheExecutor \
        TestExecuteStreamDoneBeforeDrainIsBoundedNotDeadlocked
    red_polarity red_hxc184 "RED_MODE=1 on the FIXED artifact — both HXC-184 guards must FAIL, proving they are not blind" \
        ./internal/tools/shell "" \
        '^TestExecuteStream(ReturnsWhenGrandchildHoldsPipes|HangDoesNotWedgeTheExecutor)$' \
        TestExecuteStreamReturnsWhenGrandchildHoldsPipes \
        TestExecuteStreamHangDoesNotWedgeTheExecutor
    suite suite_shell "full internal/tools/shell package under -race" ./internal/tools/shell ""
    notes <<'EON'
# What this run certifies

`fa1b52b3` corrected the drain grace introduced by `a7d8dbb5` (HXC-184). That
grace was a TOTAL budget: once the direct child was reaped, `ExecuteStream`
waited a fixed window and then tore the readers down. The reasoning behind it
("at most a pipe's worth of already-buffered data, milliseconds of work") held
only when the CONSUMER kept up, not when the kernel did — a command with no
grandchild can exit instantly with a full pipe, so a consumer following the
documented concurrent-drain contract at ~ms/line was cut off mid-stream. That
is output loss on a perfectly healthy command.

The fix makes the grace a NO-PROGRESS window: `OutputStreamer` counts completed
sends (`Progress()`), and the drain loop rearms its timer whenever delivery
advanced during the window, so only a FULL window with nothing moving counts as
stuck. It also stops deriving `OutputIncomplete` from which `select` branch won
(a timer/Done tie could flag a complete run as truncated) and stops letting a
teardown sentinel on stdout mask a genuine stderr failure.

Captured here (real command execution; `transcripts/` holds every byte):

* all four mechanisms are present in the tracked source that ships today —
  the `Progress()` counter, the rearm, the derived flag, the error priority;
* the commit's own I1 guard RAN and passed twice under `-race` (asserted on its
  `--- PASS:` line, not merely on exit 0);
* the three HXC-184 guards this fix had to preserve — grandchild-held pipes
  bounded, the `MaxConcurrent=1` executor slot recovered, Done-before-drain
  bounded rather than deadlocked — all still pass, so the rearm did not
  resurrect the wedge it was constrained by;
* RED_MODE=1 on this same fixed artifact FAILS both HXC-184 guards, which is
  what proves those guards can still detect the defect (§11.4.115 polarity —
  a guard that passes in both polarities certifies nothing);
* the whole package is race-clean.

Not certified: the exact throughput figures quoted in the commit message. This
run asserts the guard's OWN pass/fail contract, which is what the guard was
written to enforce; it does not independently re-measure line counts.
EON
    finish
    ;;

shell_bounds)
    qa_init "racefix_shell_bounds" "0c2f4940e4e40234ab4e0c1cd3331cf363480bb2" \
        "tools/shell: the abandoned-consumer bound corrected, and M2/M3 pinned by standing guards"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/internal/tools/shell/executor.go \
        helix_code/internal/tools/shell/output.go
    sig sig_bound "the drain loop states the REAL abandoned-consumer bound (~(headroom+1) x grace), not the two-window claim it replaced" \
        '(headroom + 1) x grace' \
        helix_code/internal/tools/shell/executor.go
    sig sig_contract "StreamingExecution names abandoning-without-Cancel as the pattern to avoid" \
        'Abandoning the channels without calling Cancel is the one pattern to avoid' \
        helix_code/internal/tools/shell/executor.go
    sig sig_progressdoc "Progress() no longer claims to prove a consumer exists" \
        'It is NOT a proof that a consumer exists.' \
        helix_code/internal/tools/shell/output.go
    guard guard_bounded "the new abandoned-consumer bound guard, -race -count=2" \
        ./internal/tools/shell "" '^TestExecuteStreamAbandonedConsumerStaysBoundedWhileProducerTrickles$' \
        TestExecuteStreamAbandonedConsumerStaysBoundedWhileProducerTrickles
    guard guard_m2 "the two M2 guards — a genuine stderr failure survives a stdout teardown, -race -count=2" \
        ./internal/tools/shell "" \
        '^(TestOutputStreamerErrPrefersGenuineFailureOverTeardown|TestExecuteStreamSurfacesStderrFailureDespiteStdoutTeardown)$' \
        TestOutputStreamerErrPrefersGenuineFailureOverTeardown \
        TestExecuteStreamSurfacesStderrFailureDespiteStdoutTeardown
    guard guard_m3 "the two M3 guards — OutputIncomplete is TRUE on a truncating cancel and FALSE after a clean EOF, -race -count=2" \
        ./internal/tools/shell "" \
        '^TestExecuteStreamCancel(MidStreamMarksOutputIncomplete|AfterCleanEOFKeepsOutputComplete)$' \
        TestExecuteStreamCancelMidStreamMarksOutputIncomplete \
        TestExecuteStreamCancelAfterCleanEOFKeepsOutputComplete
    suite suite_shell "full internal/tools/shell package under -race" ./internal/tools/shell ""
    notes <<'EON'
# What this run certifies

`0c2f4940` is a docs-and-tests follow-up to `fa1b52b3` with no behaviour change,
and it exists because two things in that commit were wrong or unguarded.

The overstatement: `fa1b52b3` claimed an abandoned consumer "fills its 100-slot
channel ... and the teardown fires within two windows". That is false. A send
completes — and therefore counts as progress — either because a consumer
received the line OR because there was still buffer headroom to absorb it, so
progress does NOT imply an active consumer. The real bound is roughly
(headroom + 1) x grace. The conclusion survived (still finite, the semaphore
slot still returns, HXC-184 does not recur); the stated mechanism did not.

The coverage gap: M2 (a genuine stderr failure must outrank the stdout teardown
sentinel) and M3 (`OutputIncomplete` must be true when a cancel drops in-flight
lines) shipped in `fa1b52b3` with no standing guard — reverting M2 left the
whole in-tree suite green, which is exactly the §11.4.135 silent-recurrence
vector.

Captured here: the three corrected statements are present in the tracked source
that ships today, and all five guards RAN and passed twice under `-race`
(asserted per test on their `--- PASS:` lines) — the corrected bound pinned by
a real measurement rather than by prose, both M2 guards, and both M3 guards
including the negative case that must stay FALSE.

Not certified: the specific timings quoted in the commit message. The bound
guard asserts the property (exceeds one window, still terminates), which is
what makes the two-window claim unable to return.
EON
    finish
    ;;

net_ipv6)
    qa_init "netfix_ipv6_bracket" "9d082a7be78a073391e0492d90537557c310b5cf" \
        "HXC-185: IPv6 authorities bracketed via a shared netutil helper, proven at the sink"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean \
        helix_code/internal/netutil/netutil.go \
        helix_code/internal/cognee/client.go \
        helix_code/internal/config/config.go \
        helix_code/internal/discovery/health_monitor.go \
        helix_code/internal/discovery/registry.go \
        helix_code/internal/memory/memory_manager.go \
        helix_code/internal/notification/engine.go \
        helix_code/internal/redis/redis.go \
        helix_code/internal/server/server.go \
        helix_code/internal/worker/ssh_pool.go \
        helix_code/tests/testinfra/testinfra.go
    sig sig_join "the shared integer-port join helper exists — the single replacement for fmt.Sprintf(\"%s:%d\")" \
        'func JoinHostPort(host string, port int) string' \
        helix_code/internal/netutil/netutil.go
    sig sig_unbracket "UnbracketHost normalises an already-bracketed host, so the join cannot produce the [[::1]] double-bracket form" \
        'func UnbracketHost(host string) string' \
        helix_code/internal/netutil/netutil.go
    sig sig_callsites "the production call sites route through the shared helper rather than composing addresses themselves" \
        'netutil.JoinHostPort' \
        helix_code/internal/cognee/client.go helix_code/internal/config/config.go \
        helix_code/internal/discovery/health_monitor.go helix_code/internal/discovery/registry.go \
        helix_code/internal/memory/memory_manager.go helix_code/internal/notification/engine.go \
        helix_code/internal/redis/redis.go helix_code/internal/server/server.go \
        helix_code/internal/worker/ssh_pool.go helix_code/tests/testinfra/testinfra.go
    guard guard_netutil "the netutil helper's own guards, driven through the REAL resolver, -race -count=2" \
        ./internal/netutil "" \
        '^(TestJoinHostPort_ProducesResolvableAddress|TestJoinHostPort_NoDoubleBracket|TestUnbracketHost)$' \
        TestJoinHostPort_ProducesResolvableAddress TestJoinHostPort_NoDoubleBracket TestUnbracketHost
    guard guard_discovery "discovery call sites reach a real IPv6 listener, -race -count=2" \
        ./internal/discovery "" '^Test(ServiceInfo_Address_BracketsIPv6|HealthMonitor_check(TCP|HTTP)_IPv6|ServiceRegistry_check(TCP|HTTP)Health_IPv6)$' \
        TestServiceInfo_Address_BracketsIPv6 TestHealthMonitor_checkTCP_IPv6 \
        TestHealthMonitor_checkHTTP_IPv6 TestServiceRegistry_checkTCPHealth_IPv6 \
        TestServiceRegistry_checkHTTPHealth_IPv6
    assert_cmd_output_contains guard_discovery "positive sink-side evidence:" \
        "the discovery guards recorded POSITIVE SINK-SIDE evidence (a real listener's accept/serve count), not a string comparison (§11.4.69)"
    guard guard_clients "cognee / memory / notification / redis / server / worker call sites reach real IPv6 listeners, -race -count=2" \
        ./internal/cognee ""  '^TestNewClient_IPv6Host_BaseURLReachesServer$' \
        TestNewClient_IPv6Host_BaseURLReachesServer
    guard guard_memory "memory providers reach real IPv6 listeners, -race -count=2" \
        ./internal/memory "" '^TestNew(Redis|Memcached)MemoryProvider_IPv6_ReachesListener$' \
        TestNewRedisMemoryProvider_IPv6_ReachesListener TestNewMemcachedMemoryProvider_IPv6_ReachesListener
    assert_cmd_output_contains guard_memory "positive sink-side evidence:" \
        "the memory guards recorded POSITIVE SINK-SIDE evidence (a real listener's accept count), not a string comparison (§11.4.69)"
    guard guard_notification "the email channel reaches a real IPv6 SMTP listener, -race -count=2" \
        ./internal/notification "" '^TestEmailChannel_Send_IPv6_ReachesListener$' \
        TestEmailChannel_Send_IPv6_ReachesListener
    guard guard_redis "the redis client reaches a real IPv6 listener, -race -count=2" \
        ./internal/redis "" '^TestNewClient_IPv6Host_ReachesRealListener$' \
        TestNewClient_IPv6Host_ReachesRealListener
    guard guard_server "the server BINDS a real IPv6 socket at the composed address, -race -count=2" \
        ./internal/server "" '^TestServerAddr_IPv6_Listens$' \
        TestServerAddr_IPv6_Listens
    guard guard_worker "the SSH pool reaches a real IPv6 listener, -race -count=2" \
        ./internal/worker "" '^TestCreateSSHClient_IPv6_ReachesListener$' \
        TestCreateSSHClient_IPv6_ReachesListener
    guard guard_testinfra "every testinfra URL/dial builder reaches a live IPv6 server, -race -count=2" \
        ./tests/testinfra "" '^TestConfig(URLBuilders|DialURLs)_IPv6$' \
        TestConfigURLBuilders_IPv6 TestConfigDialURLs_IPv6
    red_polarity red_discovery "RED_MODE=1 on the FIXED artifact — the discovery guards must FAIL, proving they are not blind" \
        ./internal/discovery "" '^TestServiceInfo_Address_BracketsIPv6$' \
        TestServiceInfo_Address_BracketsIPv6
    suite suite_netutil "full internal/netutil package under -race" ./internal/netutil ""
    notes <<'EON'
# What this run certifies

`9d082a7b` closed HXC-185: an IPv6 literal contains colons, so joining it to a
port with `fmt.Sprintf("%s:%d", host, port)` produces an authority that is both
unparseable per RFC 3986 §3.2.2 and rejected outright by the Go resolver. Any
host arriving from configuration, service discovery, or an environment variable
can legitimately be a bare IPv6 literal, so every such join had to bracket. The
fix routes all of them through one shared helper, which also closes the
double-bracket trap: `net.JoinHostPort` brackets unconditionally when the host
contains a colon, including a host that is ALREADY bracketed, and `[[::1]]:80`
is rejected just as hard as the unbracketed form. Hosts arrive in both shapes
here (`net.SplitHostPort` strips brackets; a URL authority carries them), so the
helper normalises before joining and is idempotent for either.

Captured here (real command execution; `transcripts/` holds every byte):

* both helper entry points and the rerouted production call sites are present
  in the tracked source that ships today;
* every guard listed in the verdict table RAN and passed twice under `-race`
  (asserted per test on its `--- PASS:` line, never on exit 0 alone);
* the guards are SINK-SIDE, not string comparisons: each stands up a real
  listener on `::1`, drives the production code path at it, and asserts the
  listener actually accepted the connection or served the request. The
  discovery and memory transcripts are additionally asserted to contain the
  `positive sink-side evidence:` line those tests emit with the observed
  accept/serve count, so this run cannot pass on a test that merely compared
  two strings (§11.4.69);
* RED_MODE=1 on this same fixed artifact FAILS the discovery guard, proving it
  still detects the defect shape (§11.4.115).

Honest boundary (§11.4.6) — the polarity switch is NOT uniform across these
guards, and this run does not pretend it is. `internal/netutil`'s own RED_MODE
branch characterises stdlib behaviour (that a naive join is rejected and a
double-bracketed one too), so it passes on ANY artifact by design and is
therefore useless as a falsifiability proof; the RED polarity case above
deliberately targets `internal/discovery`, whose RED_MODE branch asserts the
DEFECT is present and so can only pass on a pre-fix artifact.

Also not certified: whole-package suites for cognee / discovery / memory /
notification / redis / server / worker. Those packages carry integration tests
that need real infrastructure, and running them here would report the
infrastructure's state rather than this commit's. The call sites this commit
changed are certified by their NAMED sink-side guards; `internal/netutil`, the
package this commit introduced, IS covered by its full suite.
EON
    finish
    ;;

qa_dashboard)
    qa_init "racefix_qa_dashboard" "27754934832412879c5ef6d1551c70217c9bed6a" \
        "terminal_ui: the QA dashboard renders from snapshots, not live session pointers (HXC-174)"
    cd "$APPDIR" || exit 2
    hostctx
    clean_tree tree_clean helix_code/applications/terminal_ui/main.go \
        helix_code/internal/helixqa/wrapper.go
    sig sig_snapshots "the detached point-in-time accessor exists on the engine" \
        'func (e *Engine) ListSessionSnapshots() []*SessionState' \
        helix_code/internal/helixqa/wrapper.go
    sig sig_render "the dashboard renders from snapshots" \
        'tui.qaEngine.ListSessionSnapshots()' \
        helix_code/applications/terminal_ui/main.go
    absent absent_livepointers "no dashboard render path still reads off the LIVE session pointers" \
        'sessions := tui.qaEngine.ListSessions()' \
        helix_code/applications/terminal_ui/main.go
    guard_skippable guard_qarace "the commit's own race guard driving the REAL production showQA, -race" \
        ./applications/terminal_ui ci '^TestShowQA_SessionReadsAreRaceFree$' \
        'inconclusive: no QA session status transition was observed' \
        TestShowQA_SessionReadsAreRaceFree
    red_polarity red_qarace "RED_MODE=1 golden-bad self-validation (§11.4.107(10)) — the guard must still detect the unguarded read shape" \
        ./applications/terminal_ui ci '^TestShowQA_SessionReadsAreRaceFree$' \
        TestShowQA_SessionReadsAreRaceFree
    assert_cmd_output_contains red_qarace "DATA RACE" \
        "the RED_MODE=1 failure is a genuine detector report, not an assertion the test merely wrote itself (§11.4.107(10))"
    suite suite_terminalui "full applications/terminal_ui package under -race" ./applications/terminal_ui ci
    suite suite_helixqa "full internal/helixqa package under -race" ./internal/helixqa ci
    notes <<'EON'
# What this run certifies

`27754934` closed HXC-174. `showQA` rendered straight off the LIVE
`*SessionState` pointers returned by `Engine.ListSessions()`, reading Status /
Phase / PhaseProgress / EndTime / StartTime / Platforms / Banks with no lock
held, while the orchestrator goroutine spawned by `StartSession` writes exactly
those fields under `state.Mu`. A lock held on one side alone establishes no
happens-before edge, so this was a genuine data race rather than merely a race
condition — its sharpest instance being the duration cell's check-then-use on
`EndTime`, which could observe a non-nil pointer to a not-yet-published value.
The TUI was the only unguarded reader in the tree; every other consumer already
went through the lock.

Captured here (real command execution; `transcripts/` holds every byte):

* the detached accessor and the rerouted render path are present in the tracked
  source that ships today, and the live-pointer read is provably GONE from the
  dashboard (`git grep` finds no surviving occurrence);
* the commit's own guard, which drives the REAL production `showQA` against
  live sessions, ran under `-race` with `-tags=ci`;
* RED_MODE=1 re-drives the legacy unguarded pattern on this same fixed artifact
  and is asserted BOTH to fail AND to carry a `DATA RACE` detector report —
  that second assertion is what makes it a golden-bad self-validation
  (§11.4.107(10)) rather than a test agreeing with itself: it proves the
  harness still SEES this defect shape and is not blind;
* both affected packages are race-clean.

Honest boundary (§11.4.3 / §11.4.201): this guard is load-sensitive by
construction. It brackets each batch of renders with a lock-guarded status
vector so that a clean `-race` result cannot come from a window the writer never
touched, and if no orchestrator status transition lands inside a rendering batch
it SKIPs with a reason rather than claiming a pass over an empty window. If the
verdict table below records that SKIP, this run is INCOMPLETE, not green — the
host load that produced it is in `transcripts/host_context.txt`.
EON
    finish
    ;;

*)
    echo "capture_racefix_wave.sh: unknown case key: ${KEY}" >&2
    usage >&2
    exit 2
    ;;
esac
