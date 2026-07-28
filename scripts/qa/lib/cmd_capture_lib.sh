#!/usr/bin/env bash
# cmd_capture_lib.sh — shared command-execution capture/assert helpers for
# §11.4.83 QA runs whose deliverable is a REAL COMMAND OUTCOME (a compiler
# error, a test run, a `go doc` signature, a host-fact probe) rather than an
# HTTP wire trace.
#
# WHY THIS FILE EXISTS
#   scripts/qa/lib/sec_capture_lib.sh already provides the run-id / EVIDENCE.md
#   / verdicts.tsv conventions (qa_init, _qa_record, qa_finish, qa_skip,
#   qa_iso) and an HTTP-shaped capture primitive (qa_http + assert_status /
#   assert_header_* / assert_body_*). Several real §11.4.83 evidence targets
#   are not HTTP-shaped at all — a doc-comment reconciliation fix, a build-host
#   prerequisites doc, a test-determinism fix — and forcing them through
#   qa_http would either fabricate a nonexistent HTTP surface (a bluff) or
#   force the run-id/EVIDENCE.md plumbing to be reinvented per script. This
#   file adds the missing COMMAND-shaped capture primitive (qa_cmd +
#   assert_cmd_rc[_not] / assert_cmd_output_contains[_not] /
#   assert_file_exists_nonempty) on top of the SAME _qa_record verdict ledger,
#   so every script in this tree — HTTP-shaped or command-shaped — produces an
#   identical docs/qa/<run-id>/{verdicts.tsv,EVIDENCE.md,transcripts/} shape.
#
# USAGE (sourced AFTER sec_capture_lib.sh, which must be sourced first — this
# file uses qa_iso/_qa_record/QA_RUN_DIR from it and does not redefine them):
#   source ".../lib/sec_capture_lib.sh"
#   source ".../lib/cmd_capture_lib.sh"
#   qa_init "<slug>" "<full-commit-sha>" "<title>"
#   qa_cmd <case-id> "<label>" -- <command> [args...]
#   assert_cmd_rc <case-id> <expected-rc> ["<label>"]
#   qa_finish
#
# ANTI-BLUFF CONTRACT (§11.4 / §11.4.1 / §11.4.5 / §11.4.6) — same posture as
# sec_capture_lib.sh:
#   * Every qa_cmd call actually executes the given command; nothing here
#     ever asserts from memory, from a prior session, or from an invented
#     transcript. The command's real stdout+stderr+exit-code, captured at
#     the moment this script runs, are what every assert_cmd_* helper reads.
#   * assert_cmd_* FAIL (never SKIP, never silently pass) when the case's
#     transcript is missing, empty, or lacks the "--- exit code:" trailer
#     this library always appends — you cannot assert on a command that
#     was never actually run.
#   * Exit-code semantics are identical to sec_capture_lib.sh's qa_finish:
#     0 = all PASS, 1 = a FAIL happened, 2 = INCOMPLETE (a SKIP happened).

set -uo pipefail

# ---------------------------------------------------------------------------
# qa_cmd <case-id> <label> -- <command...>
#
# Executes <command...> for real (no dry-run, no simulation), capturing
# combined stdout+stderr and the real exit code into
# docs/qa/<run-id>/transcripts/<case-id>.txt.
#
# Sets: QA_LAST_CASE, QA_LAST_RC, QA_LAST_TRANSCRIPT, QA_LAST_OUTPUT
# ---------------------------------------------------------------------------
qa_cmd() {
    local case_id="$1"; shift
    local label="$1"; shift
    if [ "${1:-}" = "--" ]; then shift; fi

    local t="${QA_RUN_DIR}/transcripts/${case_id}.txt"
    {
        echo "### case: ${case_id}"
        echo "### label: ${label}"
        echo "### time: $(qa_iso)"
        echo "### command: $*"
        echo "### cwd: $(pwd)"
        echo
        echo "--- COMBINED STDOUT+STDERR (real command execution, captured verbatim) ---"
    } > "$t"

    local out_f; out_f="$(mktemp)"

    # Same errexit-suspension discipline as sec_capture_lib.sh's qa_http: the
    # command's non-zero rc is DATA (asserted on below), not a script failure.
    # errexit is restored to whatever it was before — never switched ON.
    local _qa_cmd_errexit_was_on=0
    case $- in *e*) _qa_cmd_errexit_was_on=1 ;; esac
    set +e
    "$@" > "$out_f" 2>&1
    local rc=$?
    if [ "$_qa_cmd_errexit_was_on" = 1 ]; then set -e; fi

    cat "$out_f" >> "$t"
    {
        echo
        echo "--- exit code: ${rc} ---"
    } >> "$t"

    QA_LAST_CASE="$case_id"
    QA_LAST_RC="$rc"
    QA_LAST_TRANSCRIPT="$t"
    QA_LAST_OUTPUT="$(cat "$out_f")"
    rm -f "$out_f"

    echo "-- ${case_id}: rc=${rc}  (${label})"
}

# Guard: every assertion requires a real, non-empty transcript with the
# exit-code trailer this library always writes.
_qa_cmd_have_evidence() {
    local case_id="$1"
    local t="${QA_RUN_DIR}/transcripts/${case_id}.txt"
    [ -s "$t" ] || return 1
    grep -q '^--- exit code:' "$t" || return 1
    return 0
}

_qa_cmd_get_rc() {
    grep '^--- exit code:' "${QA_RUN_DIR}/transcripts/${1}.txt" \
        | tail -1 \
        | sed -E 's/^--- exit code: ([0-9-]+) ---/\1/'
}

# assert_cmd_rc <case-id> <expected-rc> [assertion-label]
assert_cmd_rc() {
    local case_id="$1" want="$2" label="${3:-exit code == ${2}}"
    if ! _qa_cmd_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "$label" "$want" "NO-EVIDENCE (command was not captured)"
        return 1
    fi
    local got; got="$(_qa_cmd_get_rc "$case_id")"
    if [ "$got" = "$want" ]; then
        _qa_record PASS "$case_id" "$label" "$want" "$got"
    else
        _qa_record FAIL "$case_id" "$label" "$want" "${got:-<none>}"
    fi
}

# assert_cmd_rc_not <case-id> <forbidden-rc> [assertion-label]
assert_cmd_rc_not() {
    local case_id="$1" bad="$2" label="${3:-exit code != ${2}}"
    if ! _qa_cmd_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "$label" "not ${bad}" "NO-EVIDENCE"
        return 1
    fi
    local got; got="$(_qa_cmd_get_rc "$case_id")"
    if [ -n "$got" ] && [ "$got" != "$bad" ]; then
        _qa_record PASS "$case_id" "$label" "not ${bad}" "$got"
    else
        _qa_record FAIL "$case_id" "$label" "not ${bad}" "${got:-<none>}"
    fi
}

# assert_cmd_output_contains <case-id> <literal-needle> [assertion-label]
assert_cmd_output_contains() {
    local case_id="$1" needle="$2" label="${3:-output contains '${2}'}"
    if ! _qa_cmd_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "$label" "$needle" "NO-EVIDENCE"
        return 1
    fi
    if grep -qF -- "$needle" "${QA_RUN_DIR}/transcripts/${case_id}.txt"; then
        _qa_record PASS "$case_id" "$label" "$needle" "present"
    else
        _qa_record FAIL "$case_id" "$label" "$needle" "<absent>"
    fi
}

# assert_cmd_output_not_contains <case-id> <literal-needle> [assertion-label]
assert_cmd_output_not_contains() {
    local case_id="$1" needle="$2" label="${3:-output excludes '${2}'}"
    if ! _qa_cmd_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "$label" "absent" "NO-EVIDENCE"
        return 1
    fi
    if grep -qF -- "$needle" "${QA_RUN_DIR}/transcripts/${case_id}.txt"; then
        _qa_record FAIL "$case_id" "$label" "absent" "PRESENT"
    else
        _qa_record PASS "$case_id" "$label" "absent" "absent"
    fi
}

# assert_file_exists_nonempty <case-id> <path> [assertion-label]
# Not command-shaped, but shares the same verdict ledger — used to prove a
# build genuinely produced a real, non-empty artifact (not just rc==0).
assert_file_exists_nonempty() {
    local case_id="$1" path="$2" label="${3:-artifact '${2}' exists and is non-empty}"
    if [ -s "$path" ]; then
        _qa_record PASS "$case_id" "$label" "non-empty file" "$(wc -c < "$path") bytes"
    else
        _qa_record FAIL "$case_id" "$label" "non-empty file" "missing or empty"
    fi
}
