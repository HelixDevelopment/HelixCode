#!/usr/bin/env bash
# scripts/lib/mutation_baseline_test.sh
# HXC-282 — unit tests for scripts/lib/mutation_baseline.sh in isolation from
# the pre-commit hook (the hook-integration proof lives in
# scripts/git_hooks/test_mutation_baseline_deletion_detection.sh).
#
# This is BRAND-NEW code (no prior "broken" version existed to reproduce), so
# there is no §11.4.115 RED/GREEN polarity here — a plain PASS/FAIL suite,
# run against the real library via `source`, exercising real files on real
# disk under a throwaway git repo (mktemp -d). Never a re-grep of the
# library's own source.
#
# Usage:   scripts/lib/mutation_baseline_test.sh
# Exit:    0 iff every case passed; 1 otherwise.
# Side-effects: creates + removes a temp dir; touches no real repo state.
# Dependencies: git, bash, mktemp, sha256sum (or shasum).
# Cross-references: §11.4.84 / §11.4.6 / HXC-282; scripts/lib/
#   mutation_baseline.sh; scripts/git_hooks/pre-commit;
#   scripts/git_hooks/test_mutation_baseline_deletion_detection.sh.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PASS=0
FAIL=0
declare -a FAILED_CASES=()

ok()  { PASS=$((PASS+1)); printf 'PASS  %s\n' "$1"; }
bad() { FAIL=$((FAIL+1)); FAILED_CASES+=("$1"); printf 'FAIL  %s\n' "$1"; }

# assert_rc <case-name> <expected-rc> <actual-rc>
assert_rc() {
  local name="$1" expect="$2" actual="$3"
  if [ "$actual" = "$expect" ]; then
    ok "$name (rc=$actual as expected)"
  else
    bad "$name (expected rc=$expect, got rc=$actual)"
  fi
}

TMP=$(mktemp -d 2>/dev/null) || { echo "mktemp failed"; exit 1; }
cleanup() { rm -rf "$TMP" 2>/dev/null || true; }
trap cleanup EXIT

cd "$TMP" || exit 1
git init -q
git config user.email "test@example.com"
git config user.name "HXC-282 Lib Test"
git config commit.gpgsign false

mkdir -p scripts/lib sub/dir
cp "$HERE/mutation_baseline.sh" scripts/lib/mutation_baseline.sh
# shellcheck source=/dev/null
. scripts/lib/mutation_baseline.sh

echo "=== HXC-282 mutation_baseline.sh unit tests ==="
echo

# ---------------------------------------------------------------------------
# Case 1: record() on an existing file captures a real sha256 + preserved
# copy, and is_open() sees it.
# ---------------------------------------------------------------------------
printf 'alpha\nprotective_rule_here\nomega\n' > sub/dir/guard.sh
orig_hash=$(sha256sum sub/dir/guard.sh | awk '{print $1}')

mutation_baseline_record sub/dir/guard.sh >/tmp/mb_rec_out.$$ 2>&1
assert_rc "C1a-record-new-file" "0" "$?"

if mutation_baseline_is_open sub/dir/guard.sh; then
  ok "C1b-is-open-after-record"
else
  bad "C1b-is-open-after-record"
fi

got_hash=$(mutation_baseline_get_hash sub/dir/guard.sh)
if [ "$got_hash" = "$orig_hash" ]; then
  ok "C1c-get-hash-matches-real-sha256sum ($got_hash)"
else
  bad "C1c-get-hash-matches-real-sha256sum (want $orig_hash got $got_hash)"
fi

gitdir=$(git rev-parse --git-dir)
record_id=$(printf '%s' "sub/dir/guard.sh" | sha256sum | awk '{print $1}')
preserved="$gitdir/mutation_baselines/records/$record_id/baseline_copy"
if [ -f "$preserved" ] && cmp -s "$preserved" sub/dir/guard.sh; then
  ok "C1d-preserved-copy-byte-identical-to-original"
else
  bad "C1d-preserved-copy-byte-identical-to-original (missing or differs: $preserved)"
fi

# ---------------------------------------------------------------------------
# Case 2: record() while ALREADY open on the same path REFUSES (§11.4.84
# serialisation) rather than silently overwriting the earlier baseline.
# ---------------------------------------------------------------------------
mutation_baseline_record sub/dir/guard.sh >/dev/null 2>&1
assert_rc "C2-record-while-open-refuses" "1" "$?"
got_hash2=$(mutation_baseline_get_hash sub/dir/guard.sh)
if [ "$got_hash2" = "$orig_hash" ]; then
  ok "C2b-original-baseline-not-clobbered-by-refused-second-record"
else
  bad "C2b-original-baseline-not-clobbered-by-refused-second-record"
fi

# ---------------------------------------------------------------------------
# Case 3: an UN-RESTORED DELETION (no marker left behind at all) is caught
# by verify_restored() as a MISMATCH -- the whole point of HXC-282. The
# record stays OPEN (residue, not silently forgiven).
# ---------------------------------------------------------------------------
sed -i '/protective_rule_here/d' sub/dir/guard.sh
# HXC-282 round 6 / Blocker 2: built from parts rather than one literal
# contiguous string -- this is genuine test CODE (not documentation, see
# mutation_baseline.sh's own note for the doc-comment equivalent of this
# same discipline), and reproducing the residue marker byte-for-byte in
# THIS file's own source would make this test script trip the very
# pre-commit residue sweep it exists to validate the library for.
_c3_marker_frag1="_mutat"
_c3_marker_frag2="ed_"
if grep -q "MUTATED\\|always pass\\|${_c3_marker_frag1}${_c3_marker_frag2}" sub/dir/guard.sh; then
  bad "C3-sanity-fixture-leaves-no-marker (fixture is wrong)"
else
  ok "C3-sanity-fixture-leaves-no-marker"
fi
mutation_baseline_verify_restored sub/dir/guard.sh >/tmp/mb_verify_out.$$ 2>&1
assert_rc "C3a-verify-restored-on-undone-deletion-mismatches" "1" "$?"
if mutation_baseline_is_open sub/dir/guard.sh; then
  ok "C3b-record-stays-open-after-mismatch"
else
  bad "C3b-record-stays-open-after-mismatch"
fi

# ---------------------------------------------------------------------------
# Case 4: properly restoring to the EXACT original content, then
# verify_restored(), MATCHES and CLOSES the record.
# ---------------------------------------------------------------------------
printf 'alpha\nprotective_rule_here\nomega\n' > sub/dir/guard.sh
mutation_baseline_verify_restored sub/dir/guard.sh >/tmp/mb_verify_out2.$$ 2>&1
assert_rc "C4a-verify-restored-on-genuine-restore-matches" "0" "$?"
if mutation_baseline_is_open sub/dir/guard.sh; then
  bad "C4b-record-closed-after-match"
else
  ok "C4b-record-closed-after-match"
fi

# ---------------------------------------------------------------------------
# Case 5: the verification -- BOTH outcomes -- is CAPTURED in the append-
# only events log, not merely asserted in-memory (§11.4.84 acceptance
# criterion: "the verification is captured").
# ---------------------------------------------------------------------------
events="$gitdir/mutation_baselines/events.jsonl"
if [ -f "$events" ] \
  && grep -q '"event":"open".*"path":"sub/dir/guard.sh"' "$events" \
  && grep -q '"event":"verified_restored".*"result":"mismatch"' "$events" \
  && grep -q '"event":"verified_restored".*"result":"match"' "$events"; then
  ok "C5-events-log-captures-open-mismatch-and-match"
else
  bad "C5-events-log-captures-open-mismatch-and-match (missing expected lines in $events)"
fi

# ---------------------------------------------------------------------------
# Case 6: verify_restored() with NO open record is a usage error (rc=2), not
# a silent no-op and not treated as a match.
# ---------------------------------------------------------------------------
mutation_baseline_verify_restored sub/dir/never_recorded.sh >/dev/null 2>&1
assert_rc "C6-verify-without-record-is-usage-error" "2" "$?"

# ---------------------------------------------------------------------------
# Case 7: abandon() without a reason REFUSES (never a silent close);
# abandon() WITH a reason closes an open record even on a genuine
# content mismatch, and is captured distinctly from a verified match.
# ---------------------------------------------------------------------------
mutation_baseline_record sub/dir/guard.sh >/dev/null 2>&1
echo "extra intentional line" >> sub/dir/guard.sh   # deliberate, non-restoring edit

mutation_baseline_abandon sub/dir/guard.sh >/dev/null 2>&1
assert_rc "C7a-abandon-without-reason-refuses" "2" "$?"
if mutation_baseline_is_open sub/dir/guard.sh; then
  ok "C7b-record-still-open-after-refused-abandon"
else
  bad "C7b-record-still-open-after-refused-abandon"
fi

mutation_baseline_abandon sub/dir/guard.sh "intentional follow-up edit, not residue" >/dev/null 2>&1
assert_rc "C7c-abandon-with-reason-succeeds" "0" "$?"
if mutation_baseline_is_open sub/dir/guard.sh; then
  bad "C7d-record-closed-after-abandon"
else
  ok "C7d-record-closed-after-abandon"
fi
if grep -q '"event":"abandoned".*"reason":"intentional follow-up edit, not residue"' "$events"; then
  ok "C7e-abandon-reason-captured-in-events-log"
else
  bad "C7e-abandon-reason-captured-in-events-log"
fi

# ---------------------------------------------------------------------------
# Case 8: the ABSENT sentinel -- record() on a not-yet-existing file, then
# verify_restored() requires the file to be absent AGAIN to match (creating
# it and forgetting to remove it again is exactly as much residue as
# forgetting to revert an edit).
# ---------------------------------------------------------------------------
rm -f sub/dir/new_file.sh
mutation_baseline_record sub/dir/new_file.sh >/dev/null 2>&1
assert_rc "C8a-record-on-absent-file" "0" "$?"
if [ "$(mutation_baseline_get_hash sub/dir/new_file.sh)" = "ABSENT" ]; then
  ok "C8b-absent-sentinel-recorded"
else
  bad "C8b-absent-sentinel-recorded (got: $(mutation_baseline_get_hash sub/dir/new_file.sh))"
fi

echo "created by an experiment, left behind" > sub/dir/new_file.sh
mutation_baseline_verify_restored sub/dir/new_file.sh >/dev/null 2>&1
assert_rc "C8c-leftover-created-file-mismatches-absent-baseline" "1" "$?"

rm -f sub/dir/new_file.sh
mutation_baseline_verify_restored sub/dir/new_file.sh >/dev/null 2>&1
assert_rc "C8d-removing-the-created-file-again-matches-absent-baseline" "0" "$?"

# ---------------------------------------------------------------------------
# Case 9: get_hash() / is_open() on a path with NO record at all (the
# overwhelming common case for an ordinary commit) never errors loudly and
# simply reports "not open" -- proving the discriminator costs nothing for
# files nobody ever registered.
# ---------------------------------------------------------------------------
echo "ordinary file" > sub/dir/ordinary.txt
if mutation_baseline_is_open sub/dir/ordinary.txt; then
  bad "C9a-never-registered-file-reports-not-open"
else
  ok "C9a-never-registered-file-reports-not-open"
fi
if mutation_baseline_get_hash sub/dir/ordinary.txt >/dev/null 2>&1; then
  bad "C9b-get-hash-on-unregistered-file-fails"
else
  ok "C9b-get-hash-on-unregistered-file-fails"
fi

# ---------------------------------------------------------------------------
# Case 10: queries do not write. In a FRESH repo with NO .git/
# mutation_baselines/ directory at all yet, calling is_open / get_hash /
# get_meta_field on files that were never registered must NOT create that
# storage directory as a side effect -- a query is not a write (coordinator
# finding, 2026-08-12: a real commit through the live hook left an empty
# .git/mutation_baselines/{records/,events.jsonl} behind purely from
# is_open checks on files with no baseline at all).
# ---------------------------------------------------------------------------
QTMP=$(mktemp -d 2>/dev/null)
(
  cd "$QTMP" || exit 1
  git init -q
  git config user.email "q@example.com"; git config user.name "q"; git config commit.gpgsign false
  # shellcheck source=/dev/null
  . "$HERE/mutation_baseline.sh"
  echo "content" > never_registered.txt
  mutation_baseline_is_open never_registered.txt >/dev/null 2>&1
  mutation_baseline_get_hash never_registered.txt >/dev/null 2>&1
  mutation_baseline_get_meta_field never_registered.txt status >/dev/null 2>&1
)
if [ -e "$QTMP/.git/mutation_baselines" ]; then
  bad "C10-read-only-queries-do-not-create-storage-directory (found: $QTMP/.git/mutation_baselines)"
else
  ok "C10-read-only-queries-do-not-create-storage-directory"
fi
# A WRITE (record) on the same fresh repo, by contrast, MUST create it.
(
  cd "$QTMP" || exit 1
  # shellcheck source=/dev/null
  . "$HERE/mutation_baseline.sh"
  mutation_baseline_record never_registered.txt >/dev/null 2>&1
)
if [ -d "$QTMP/.git/mutation_baselines/records" ]; then
  ok "C10b-record-does-create-storage-directory-when-actually-writing"
else
  bad "C10b-record-does-create-storage-directory-when-actually-writing"
fi
rm -rf "$QTMP" 2>/dev/null || true

# ---------------------------------------------------------------------------
# Case 11 (F4, review finding): record()'s check-then-write was NOT atomic
# against a second concurrent record() on the SAME path -- both could
# observe "not open" before either wrote, and the later `mv` would silently
# clobber the earlier baseline with no error to either caller. Measured by
# the reviewer, 2026-08-12: 20 concurrent rounds on one path -> 20/20 rc=0,
# 40 "open" events logged for 20 record() invocations (this file registers
# TWO baselines per successful round in the earlier cases, so the
# reviewer's "40 events / 20 paths" scales to this test's "20 events / 1
# path" shape). Fixed via _mb_locked (a per-record-id flock, repo
# precedent scripts/lib/port_sweep_lock.sh / §11.4.180).
#
# This is a genuine paired §1.1 mutation test, not merely a fixture: it
# first builds an UNLOCKED mutant of the real library (replaces _mb_locked
# with a pass-through) and proves that mutant DOES race (mutation
# detected), then proves the real, shipped library does NOT (invariant
# intact) -- run in that order so the test is proven capable of catching
# the exact defect before it certifies the fix.
# ---------------------------------------------------------------------------
RTMP=$(mktemp -d 2>/dev/null)
UNLOCKED_MUTANT="$RTMP/mutant_mutation_baseline.sh"
awk '
  BEGIN { in_fn = 0 }
  /^_mb_locked\(\) \{$/ {
    print "_mb_locked() {"
    print "  local record_id=\"${1:-}\""
    print "  shift"
    print "  \"$@\""
    print "}"
    in_fn = 1
    next
  }
  in_fn && /^}$/ { in_fn = 0; next }
  in_fn { next }
  { print }
' "$HERE/mutation_baseline.sh" >"$UNLOCKED_MUTANT"
if diff -q "$HERE/mutation_baseline.sh" "$UNLOCKED_MUTANT" >/dev/null 2>&1; then
  bad "C11-sanity-unlocked-mutant-actually-differs-from-real-library (awk produced no change)"
else
  ok "C11-sanity-unlocked-mutant-actually-differs-from-real-library"
fi

concurrent_record_race() {
  # $1 = library file to source. Runs 20 concurrent mutation_baseline_record
  # calls on the SAME path in a FRESH throwaway repo; echoes
  # "<successes> <open-events>" to stdout.
  local libfile="$1" ctmp n
  ctmp=$(mktemp -d 2>/dev/null) || { echo "0 0"; return; }
  (
    cd "$ctmp" || exit 1
    git init -q
    git config user.email "race@example.com"; git config user.name "race"; git config commit.gpgsign false
    echo "target content" >racy_file.sh
    for n in $(seq 1 20); do
      ( # The outer test script already sourced the REAL library earlier
        # (for Cases 1-10), which sets _MUTATION_BASELINE_SOURCED=1 in this
        # subshell's INHERITED environment. mutation_baseline.sh's own
        # idempotent-source guard would then treat re-sourcing (even the
        # UNLOCKED MUTANT, a different file) as a no-op and silently keep
        # the inherited REAL (locked) functions -- masking the mutation
        # entirely and making this case unable to fail no matter how
        # unlocked the mutant is. Unset it so sourcing genuinely happens.
        unset _MUTATION_BASELINE_SOURCED
        # shellcheck source=/dev/null
        . "$libfile"
        mutation_baseline_record racy_file.sh >/dev/null 2>&1
        echo "$?" >"race_rc_${n}.txt"
      ) &
    done
    wait
  )
  local successes=0 rc
  for n in $(seq 1 20); do
    rc=$(cat "$ctmp/race_rc_${n}.txt" 2>/dev/null)
    [ "$rc" = "0" ] && successes=$((successes + 1))
  done
  local open_events
  open_events=$(grep -c '"event":"open"' "$ctmp/.git/mutation_baselines/events.jsonl" 2>/dev/null || echo 0)
  rm -rf "$ctmp" 2>/dev/null || true
  echo "$successes $open_events"
}

read -r mutant_successes mutant_opens <<<"$(concurrent_record_race "$UNLOCKED_MUTANT")"
if [ "${mutant_successes:-0}" -gt 1 ] && [ "${mutant_opens:-0}" -gt 1 ]; then
  ok "C11a-unlocked-mutant-races (mutation detected: successes=$mutant_successes open-events=$mutant_opens, expected exactly 1 of each)"
else
  bad "C11a-unlocked-mutant-races (expected the race to reproduce: successes=$mutant_successes open-events=$mutant_opens)"
fi

read -r real_successes real_opens <<<"$(concurrent_record_race "$HERE/mutation_baseline.sh")"
if [ "${real_successes:-0}" = "1" ] && [ "${real_opens:-0}" = "1" ]; then
  ok "C11b-real-library-serialises-concurrent-record (invariant intact: successes=$real_successes open-events=$real_opens)"
else
  bad "C11b-real-library-serialises-concurrent-record (expected exactly 1 success and 1 open-event: got successes=$real_successes open-events=$real_opens)"
fi
rm -rf "$RTMP" 2>/dev/null || true

echo
echo "=== HXC-282 mutation_baseline.sh unit tests: PASS=$PASS FAIL=$FAIL ==="
if [ "$FAIL" -ne 0 ]; then
  printf 'FAILED CASES:\n'
  for c in "${FAILED_CASES[@]}"; do printf '  - %s\n' "$c"; done
  exit 1
fi
exit 0
