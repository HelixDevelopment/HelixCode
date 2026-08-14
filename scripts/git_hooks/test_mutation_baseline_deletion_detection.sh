#!/usr/bin/env bash
# scripts/git_hooks/test_mutation_baseline_deletion_detection.sh
# HXC-282 — §11.4.115 RED-first polarity test for the §11.4.84 pre-commit
# hook's baseline-comparison residue check (scripts/lib/mutation_baseline.sh
# + scripts/git_hooks/pre-commit step 3b).
#
# THE DEFECT (RED_MODE=1 reproduces it on the frozen pre-fix hook):
#   The §11.4.84 residue scan used to rely ENTIRELY on a grep for tell-tale
#   markers a mutation experiment is supposed to leave behind (`MUTATED for
#   paired`, `// always pass`, …). An experiment whose whole method is to
#   DELETE something (e.g. a protective masking rule) leaves no such marker,
#   so the grep-only sweep reported clean while a real, un-restored deletion
#   sat staged and one commit away from shipping. Forensic case (HXC-282): a
#   mutation that removed a credential-masking rule from a live source file
#   was left un-restored on disk, and the sweep — like every other check —
#   reported clean.
#
# THE FIX: scripts/lib/mutation_baseline.sh lets an experiment RECORD a
# known-good sha256 + preserved copy of a file BEFORE mutating it, and the
# pre-commit hook, for any staged file carrying an OPEN baseline record,
# compares the STAGED BLOB's sha256 against the recorded one — a comparison,
# not a phrase search, so it catches an unintended difference of ANY shape:
# added, changed, or removed.
#
# POLARITY SWITCH (§11.4.115): ONE test script, TWO hook artifacts.
#   RED_MODE=0 (default) — the standing GREEN regression guard: installs the
#     REAL, CURRENT scripts/git_hooks/pre-commit (this fix). The un-restored
#     deletion MUST be BLOCKED.
#   RED_MODE=1 — reproduces the historical defect on a FROZEN, byte-for-byte
#     snapshot of the pre-fix hook (scripts/git_hooks/testdata/
#     hxc282_pre_fix_pre_commit.sh, captured before this fix landed — never
#     re-derived from git history, which would stop meaning "pre-fix" the
#     moment this fix is committed). The SAME un-restored deletion MUST be
#     ALLOWED — proving the defect this fix closes was real, and that this
#     test's oracle is capable of catching it.
#   Most other cases assert IDENTICALLY in both modes: they exercise layer
#   3a (the pre-existing marker sweep, untouched by this fix) or paths where
#   no OPEN baseline record exists at all — the no-regression / no-over-
#   catch quadrants. A fix that turned baseline comparison into a blanket
#   "block any staged diff" would fail these in either polarity. Two
#   sub-checks inside case H1/I are ALSO polarity-switched (a second,
#   independent defect this fix closes): the frozen pre-fix hook still uses
#   the ORIGINAL, unwrapped `blob=$(git show ":$f")` capture, so it
#   legitimately DOES leak bash's "ignored null byte" warning on binary
#   content — reproducing a real coordinator-reported finding — while the
#   live hook must not.
#
# ANTI-BLUFF: runs real `git add` / `git commit` against the real hook (or
# its frozen pre-fix snapshot) in a throwaway repo (mktemp -d) — never a
# re-grep of hook or library source. Fixtures are real files mutated on real
# disk via the real library functions, not simulated outcomes.
#
# Usage:   scripts/git_hooks/test_mutation_baseline_deletion_detection.sh
#          RED_MODE=1 scripts/git_hooks/test_mutation_baseline_deletion_detection.sh
# Exit:    0 iff every case passed for the active RED_MODE; 1 otherwise.
# Side-effects: creates + removes a temp dir; touches NO real repo state,
#   installs NO hooks into the real .git/hooks, never uses --no-verify.
# Dependencies: git, bash, mktemp, sha256sum (or shasum).
# Cross-references: §11.4.84 / §11.4.115 / §1.1 / HXC-282;
#   scripts/git_hooks/pre-commit; scripts/lib/mutation_baseline.sh;
#   scripts/git_hooks/testdata/hxc282_pre_fix_pre_commit.sh;
#   scripts/git_hooks/test_mutation_residue_evidence_exempt.sh (sibling
#   §11.4.84 polarity test for the captured-evidence exemption path — this
#   file does not duplicate its cases).

set -uo pipefail

RED_MODE="${RED_MODE:-0}"
HOOKS_SRC_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$HOOKS_SRC_DIR/../.." && pwd)
FROZEN_PRE_FIX_HOOK="$HOOKS_SRC_DIR/testdata/hxc282_pre_fix_pre_commit.sh"
LIVE_HOOK="$HOOKS_SRC_DIR/pre-commit"
LIB_SRC="$REPO_ROOT/scripts/lib/mutation_baseline.sh"

if [ ! -f "$FROZEN_PRE_FIX_HOOK" ]; then
  echo "FATAL: frozen pre-fix fixture missing: $FROZEN_PRE_FIX_HOOK" >&2
  exit 1
fi
if [ ! -f "$LIB_SRC" ]; then
  echo "FATAL: library missing: $LIB_SRC" >&2
  exit 1
fi

PASS=0
FAIL=0
declare -a FAILED_CASES=()

ok()  { PASS=$((PASS+1)); printf 'PASS  %s\n' "$1"; }
bad() { FAIL=$((FAIL+1)); FAILED_CASES+=("$1"); printf 'FAIL  %s\n' "$1"; }

# assert_block <case-name> <expected: BLOCK|ALLOW> <actual-exit>
assert_block() {
  local name="$1" expect="$2" rc="$3"
  if [ "$expect" = "BLOCK" ]; then
    if [ "$rc" -ne 0 ]; then ok "$name (blocked as expected)"
    else bad "$name (expected BLOCK, hook ALLOWED)"; fi
  else
    if [ "$rc" -eq 0 ]; then ok "$name (allowed as expected)"
    else bad "$name (expected ALLOW, hook BLOCKED rc=$rc)"; fi
  fi
}

TMP=$(mktemp -d 2>/dev/null) || { echo "mktemp failed"; exit 1; }
cleanup() { rm -rf "$TMP" 2>/dev/null || true; }
trap cleanup EXIT

cd "$TMP" || exit 1
git init -q
git config user.email "test@example.com"
git config user.name "HXC-282 Test"
git config commit.gpgsign false

mkdir -p .git/hooks scripts/lib
cp "$LIB_SRC" scripts/lib/mutation_baseline.sh
if [ "$RED_MODE" = "1" ]; then
  cp "$FROZEN_PRE_FIX_HOOK" .git/hooks/pre-commit
  ACTIVE_HOOK_LABEL="frozen pre-fix snapshot"
else
  cp "$LIVE_HOOK" .git/hooks/pre-commit
  ACTIVE_HOOK_LABEL="live (fixed) hook"
fi
chmod +x .git/hooks/pre-commit

# shellcheck source=/dev/null
. scripts/lib/mutation_baseline.sh

echo "seed" > README.txt
git add README.txt scripts/lib/mutation_baseline.sh
git commit -q -m "seed" 2>/dev/null

HK_OUT="$TMP/hook_output.txt"
try_commit() {
  git commit -q -m "$1" >"$HK_OUT" 2>&1
  echo $?
}
unstage() { git reset -q HEAD -- "$@" 2>/dev/null || true; }

echo "=== HXC-282 mutation-baseline deletion detection — RED_MODE=$RED_MODE ($ACTIVE_HOOK_LABEL) ==="
echo

# ---------------------------------------------------------------------------
# CASE A (POLARITY-SWITCHED, the core fix): a protective rule is DELETED
# from a file with an OPEN baseline record and left un-restored. No marker
# of any kind is present -- this is exactly the shape a phrase grep cannot
# see, and exactly the HXC-282 forensic scenario.
#   RED_MODE=0 (live hook)         -> MUST be BLOCKED.
#   RED_MODE=1 (frozen pre-fix)    -> MUST be ALLOWED (proves the historical
#                                      defect was real and this oracle can
#                                      catch it).
# ---------------------------------------------------------------------------
mkdir -p case_a
cat > case_a/guard.sh <<'GUARD'
#!/usr/bin/env bash
log_event() { printf '%s\n' "$1"; }
mask_secret() { printf '%s' "${1//[A-Za-z0-9]/*}"; }  # protective rule: mask credential-shaped values before logging
emit() { log_event "$(mask_secret "$1")"; }
GUARD
git add case_a/guard.sh
git commit -q -m "case A: seed guard.sh with protective rule" 2>/dev/null

mutation_baseline_record case_a/guard.sh >/dev/null 2>&1

# Delete ONLY the protective-rule line. No insertion, no marker text.
sed -i '/protective rule:/d' case_a/guard.sh
if grep -qE 'MUTATED for paired|// always pass|# always pass|MUTATION-RESIDUE|_mutated_' case_a/guard.sh; then
  bad "CaseA-sanity-fixture-must-leave-no-marker (fixture is broken -- the case would pass for the wrong reason)"
else
  ok "CaseA-sanity-fixture-leaves-no-marker"
fi

git add case_a/guard.sh
rc=$(try_commit "case A: (accidentally) commit the un-restored deletion")
if [ "$RED_MODE" = "1" ]; then
  assert_block "CaseA-undone-deletion-on-FROZEN-pre-fix-hook (RED: reproduces the defect)" ALLOW "$rc"
else
  assert_block "CaseA-undone-deletion-on-LIVE-fixed-hook (GREEN: defect closed)" BLOCK "$rc"
  if grep -qF "BLOCKED by pre-commit hook (§11.4.84 mutation-baseline residue)" "$HK_OUT" 2>/dev/null \
    && grep -qF "case_a/guard.sh" "$HK_OUT" 2>/dev/null; then
    ok "CaseA-block-message-names-the-file-and-the-mechanism"
  else
    bad "CaseA-block-message-names-the-file-and-the-mechanism (hook output did not)"
  fi
fi
unstage case_a/guard.sh

# ---------------------------------------------------------------------------
# CASE B (regression guard, asserts IDENTICALLY in both modes): an
# UN-RESTORED INSERTION carrying the pre-existing marker, with NO baseline
# record at all, is still caught by the pre-existing marker sweep (layer
# 3a). This fix must not regress the case the grep already caught.
# ---------------------------------------------------------------------------
mkdir -p case_b
cat > case_b/live_gate.sh <<'LIVE'
#!/usr/bin/env bash
verify_token() { return 0; } # MUTATED for paired §1.1 mutation test
LIVE
git add case_b/live_gate.sh
assert_block "CaseB-marker-insertion-no-baseline-still-blocked" BLOCK "$(try_commit 'case B: marker residue, no baseline registered')"
unstage case_b/live_gate.sh

# ---------------------------------------------------------------------------
# CASE C (asserts IDENTICALLY in both modes): an ordinary, unrelated edit to
# a file that has NEVER been registered with mutation_baseline_record. The
# discriminator is "does this file have an open record", never "did this
# file change" -- so this MUST be allowed regardless of how large the diff.
# ---------------------------------------------------------------------------
mkdir -p case_c
printf 'line one\nline two\nline three\n' > case_c/ordinary.txt
git add case_c/ordinary.txt
git commit -q -m "case C: seed ordinary.txt" 2>/dev/null
printf 'line one\nline two REPLACED\nline three\nline four (new)\n' > case_c/ordinary.txt
git add case_c/ordinary.txt
assert_block "CaseC-ordinary-unrelated-edit-never-registered" ALLOW "$(try_commit 'case C: ordinary edit')"

# ---------------------------------------------------------------------------
# CASE D (asserts IDENTICALLY in both modes): a mutation that IS correctly
# restored to byte-identical content before commit is allowed, whether or
# not the experiment remembered to formally close the record with
# verify_restored. Content equality is the ground truth, not the open/closed
# administrative flag.
# ---------------------------------------------------------------------------
# NOTE ON FIXTURE SHAPE: the baseline is deliberately recorded on PENDING,
# not-yet-committed content (simulating ongoing work already in flight when
# the experiment begins -- the realistic shape of the forensic case, and
# the general contract of mutation_baseline_record: it captures whatever is
# on disk right now, not necessarily HEAD). If the baseline instead equalled
# HEAD's already-committed content, "restoring to baseline" would produce a
# staged diff of EXACTLY ZERO against HEAD, and `git commit` would refuse
# with "nothing to commit" -- a real git behaviour, unrelated to the hook,
# that would make this case pass or fail for the wrong reason.
mkdir -p case_d
printf 'alpha\nbeta\ngamma\n' > case_d/restored_no_close.sh
git add case_d/restored_no_close.sh
git commit -q -m "case D1: seed" 2>/dev/null
printf 'alpha\nbeta-pending-work\ngamma\n' > case_d/restored_no_close.sh   # pending, uncommitted work
mutation_baseline_record case_d/restored_no_close.sh >/dev/null 2>&1
printf 'alpha\nMUTATED_TEMPORARILY\ngamma\n' > case_d/restored_no_close.sh   # mutate
printf 'alpha\nbeta-pending-work\ngamma\n' > case_d/restored_no_close.sh    # restore exactly to the pending baseline, but never call verify_restored
git add case_d/restored_no_close.sh
assert_block "CaseD1-restored-exact-match-without-formal-close" ALLOW "$(try_commit 'case D1: restored but record left open')"
# F5 (round-4 review finding, low, POLARITY-SWITCHED): this is step 3b's
# "genuinely restored" MATCH-NOTICE branch (staged_hash == baseline_hash,
# record still open) -- CaseD1 above already exercises the CODE PATH that
# emits it (unlike CaseD2 below, which formally closes the record BEFORE
# staging, so the hook's `mutation_baseline_is_open` check is false there
# and the whole branch is never reached), but nothing previously asserted
# the NOTICE text itself was produced. CaseD2 cannot be reused for this:
# only a still-OPEN record reaches the "does it match" comparison at all.
#   RED_MODE=0 (live hook, step 3b exists)        -> NOTICE MUST appear.
#   RED_MODE=1 (frozen pre-fix, no step 3b at all) -> NOTICE cannot appear
#     (there is nothing wrong with this snapshot for lacking a NOTICE from
#     a layer it predates -- not asserting the negative here, since CaseD1's
#     own ALLOW assertion above already covers that snapshot's behaviour).
if [ "$RED_MODE" = "1" ]; then
  ok "CaseD1-genuinely-restored-match-NOTICE-n/a-on-FROZEN-pre-fix-hook (step 3b does not exist on that snapshot)"
elif grep -qF "staged file matches its OPEN mutation baseline (genuinely restored): case_d/restored_no_close.sh" "$HK_OUT" 2>/dev/null; then
  ok "CaseD1-genuinely-restored-match-NOTICE-is-emitted"
else
  bad "CaseD1-genuinely-restored-match-NOTICE-is-emitted (expected the step-3b MATCH-branch NOTICE naming the file in hook output)"
fi

printf 'one\ntwo\nthree\n' > case_d/restored_and_closed.sh
git add case_d/restored_and_closed.sh
git commit -q -m "case D2: seed" 2>/dev/null
printf 'one\ntwo-pending-work\nthree\n' > case_d/restored_and_closed.sh   # pending, uncommitted work
mutation_baseline_record case_d/restored_and_closed.sh >/dev/null 2>&1
printf 'one\nMUTATED_TEMPORARILY\nthree\n' > case_d/restored_and_closed.sh
printf 'one\ntwo-pending-work\nthree\n' > case_d/restored_and_closed.sh
mutation_baseline_verify_restored case_d/restored_and_closed.sh >/dev/null 2>&1
git add case_d/restored_and_closed.sh
assert_block "CaseD2-restored-exact-match-with-formal-close" ALLOW "$(try_commit 'case D2: restored and closed')"

# ---------------------------------------------------------------------------
# CASE E (the "same-session legitimate edit" behaviour named in HXC-282):
# a stream mutates a file, restores it correctly, formally closes the
# record via verify_restored, and THEN makes a genuine, unrelated further
# edit to the SAME file in the SAME session. That further edit must not be
# blocked -- the closed record no longer applies.
# ---------------------------------------------------------------------------
mkdir -p case_e
printf 'v1 content\n' > case_e/shared.sh
git add case_e/shared.sh
git commit -q -m "case E: seed" 2>/dev/null
mutation_baseline_record case_e/shared.sh >/dev/null 2>&1
printf 'v1 content MUTATED\n' > case_e/shared.sh
printf 'v1 content\n' > case_e/shared.sh
mutation_baseline_verify_restored case_e/shared.sh >/dev/null 2>&1   # closes the record
printf 'v2 content -- genuine follow-up work, unrelated to the mutation\n' > case_e/shared.sh
git add case_e/shared.sh
assert_block "CaseE-legit-further-edit-after-mutation-properly-closed" ALLOW "$(try_commit 'case E: legitimate follow-up edit')"

# ---------------------------------------------------------------------------
# CASE F (the "agent's own scratch copies beside the real file" behaviour
# named in HXC-282): a scratch/backup sibling file living next to the real
# file must not be treated as residue merely for existing beside it -- the
# mechanism is keyed on the EXACT registered path, not on filename
# similarity or directory proximity.
# ---------------------------------------------------------------------------
mkdir -p case_f
printf 'real content\n' > case_f/real.sh
git add case_f/real.sh
git commit -q -m "case F: seed" 2>/dev/null
mutation_baseline_record case_f/real.sh >/dev/null 2>&1
printf 'real content MUTATED\n' > case_f/real.sh
printf 'real content\n' > case_f/real.sh
mutation_baseline_verify_restored case_f/real.sh >/dev/null 2>&1   # closes the record
# The agent's own scratch copy, never registered, sitting right beside it.
printf 'real content -- scratch working copy, not the tracked file\n' > case_f/real.sh.scratch_copy
git add case_f/real.sh case_f/real.sh.scratch_copy
assert_block "CaseF-scratch-copy-beside-real-file-not-flagged" ALLOW "$(try_commit 'case F: real file plus scratch sibling')"

# ---------------------------------------------------------------------------
# CASE G (mixed commit, per-file correctness): in a SINGLE commit, one
# staged file has an open baseline that genuinely mismatches (residue), a
# second has an open baseline that genuinely matches (fine), and a third
# has no baseline at all (ordinary). Only the first should block.
# ---------------------------------------------------------------------------
mkdir -p case_g
printf 'mismatch-me original\n' > case_g/mismatch.sh
printf 'match-me original\n' > case_g/match.sh
printf 'ordinary original\n' > case_g/ordinary.sh
git add case_g/mismatch.sh case_g/match.sh case_g/ordinary.sh
git commit -q -m "case G: seed" 2>/dev/null

mutation_baseline_record case_g/mismatch.sh >/dev/null 2>&1
mutation_baseline_record case_g/match.sh >/dev/null 2>&1

printf 'mismatch-me MUTATED and left broken\n' > case_g/mismatch.sh          # left un-restored
printf 'match-me MUTATED\n' > case_g/match.sh
printf 'match-me original\n' > case_g/match.sh                               # restored exactly
printf 'ordinary EDITED, never registered\n' > case_g/ordinary.sh

git add case_g/mismatch.sh case_g/match.sh case_g/ordinary.sh
rc=$(try_commit "case G: mixed commit")
if [ "$RED_MODE" = "1" ]; then
  assert_block "CaseG-mixed-commit-on-FROZEN-pre-fix-hook (RED: no marker anywhere -> allowed)" ALLOW "$rc"
else
  assert_block "CaseG-mixed-commit-blocks-only-for-the-genuine-mismatch" BLOCK "$rc"
  if grep -qF "case_g/mismatch.sh" "$HK_OUT" 2>/dev/null \
    && ! grep -qF "case_g/match.sh" "$HK_OUT" 2>/dev/null \
    && ! grep -qF "case_g/ordinary.sh" "$HK_OUT" 2>/dev/null; then
    ok "CaseG-block-message-names-only-the-mismatched-file"
  else
    bad "CaseG-block-message-names-only-the-mismatched-file (over- or under-reported)"
  fi
fi
unstage case_g/mismatch.sh case_g/match.sh case_g/ordinary.sh

# ---------------------------------------------------------------------------
# CASE H (coordinator finding, 2026-08-12): BINARY content, real NUL bytes.
# The concern raised was that a hash comparison routed through a bash
# variable (`$(...)`) would silently drop NUL bytes and let a MODIFIED
# binary file compare equal to a truncated version of itself. Layer 3b
# never captures staged content into a bash variable at all -- it pipes
# `git show` directly into `sha256sum` -- so this proves, with REAL NUL
# bytes in a REAL commit through the REAL hook, that a modified binary is
# refused and an unchanged one is accepted.
#   RED_MODE=0 (live hook)      -> the modified-binary case MUST be BLOCKED
#                                   (this is what layer 3b is FOR).
#   RED_MODE=1 (frozen pre-fix) -> layer 3b does not exist yet on that
#                                   snapshot, so mutation_baseline_is_open
#                                   is never consulted and this case is
#                                   necessarily ALLOWED there too -- not a
#                                   claim that the pre-fix hook was "fine"
#                                   for binaries, simply an honest
#                                   reflection of what that snapshot does.
# ---------------------------------------------------------------------------
mkdir -p case_h
# A real binary fixture: printable bytes interleaved with real NUL bytes,
# not merely "contains one stray 0x00" -- this is closer in shape to the
# real tracked artifact (docs/workable_items.db, §11.4.95) that prompted
# the finding.
printf 'HDR\x00\x00\x00PAGE1\x00\x00DATA-BLOCK-ONE\x00\x00\x00TAIL\x00\x00\x00' > case_h/binary.dat
git add case_h/binary.dat
git commit -q -m "case H: seed binary.dat" 2>/dev/null

# As in case D: the baseline is recorded on PENDING, not-yet-committed
# binary content (ongoing work already in flight), never on content that
# is already byte-identical to HEAD -- otherwise "restore exactly" in H2
# would produce a staged diff of zero against HEAD, and `git commit` would
# refuse with "nothing to commit", which is a real git behaviour unrelated
# to the hook and would make H2 pass or fail for the wrong reason.
printf 'HDR\x00\x00\x00PAGE1\x00\x00DATA-BLOCK-PENDING\x00\x00\x00TAIL\x00\x00\x00' > case_h/binary.dat
mutation_baseline_record case_h/binary.dat >/dev/null 2>&1
# Modify further, still full of NUL bytes, and leave it un-restored.
printf 'HDR\x00\x00\x00PAGE1\x00\x00DATA-BLOCK-CORRUPTED\x00\x00\x00TAIL\x00\x00\x00' > case_h/binary.dat
git add case_h/binary.dat
rc=$(try_commit "case H1: modified binary, un-restored")
if [ "$RED_MODE" = "1" ]; then
  assert_block "CaseH1-modified-binary-on-FROZEN-pre-fix-hook (RED: layer 3b absent)" ALLOW "$rc"
else
  assert_block "CaseH1-modified-binary-blocked-on-LIVE-fixed-hook (GREEN)" BLOCK "$rc"
fi
# POLARITY-SWITCHED sub-check: the frozen pre-fix snapshot still has the
# ORIGINAL, unwrapped `blob=$(git show ":$f")` line (this fix's warning
# suppression did not exist yet), so it legitimately DOES leak the warning
# on binary content -- that is itself evidence the coordinator's finding
# was real prior to this fix, not a flaw in the assertion.
if [ "$RED_MODE" = "1" ]; then
  if grep -qi "ignored null byte" "$HK_OUT" 2>/dev/null; then
    ok "CaseH1-null-byte-warning-DOES-leak-on-FROZEN-pre-fix-hook (RED: reproduces the coordinator finding)"
  else
    bad "CaseH1-null-byte-warning-DOES-leak-on-FROZEN-pre-fix-hook (RED: expected the pre-fix leak, saw none)"
  fi
else
  if grep -qi "ignored null byte" "$HK_OUT" 2>/dev/null; then
    bad "CaseH1-no-null-byte-warning-leaked-to-hook-output (GREEN)"
  else
    ok "CaseH1-no-null-byte-warning-leaked-to-hook-output (GREEN)"
  fi
fi
unstage case_h/binary.dat

# H2: restore the binary content EXACTLY to its recorded (pending) baseline
# -- byte-for-byte, including every NUL -- and confirm it is accepted: the
# positive control proving H1 was a genuine catch, not an over-broad "any
# staged binary is blocked" bug.
printf 'HDR\x00\x00\x00PAGE1\x00\x00DATA-BLOCK-PENDING\x00\x00\x00TAIL\x00\x00\x00' > case_h/binary.dat
if cmp -s <(git show HEAD:case_h/binary.dat) case_h/binary.dat; then
  bad "CaseH2-sanity-restored-content-differs-from-HEAD-seed (fixture regressed to the nothing-to-commit trap)"
else
  ok "CaseH2-sanity-restored-content-differs-from-HEAD-seed"
fi
recorded_hash=$(mutation_baseline_get_hash case_h/binary.dat 2>/dev/null)
current_hash=$(sha256sum case_h/binary.dat 2>/dev/null | awk '{print $1}')
if [ -n "$recorded_hash" ] && [ "$recorded_hash" = "$current_hash" ]; then
  ok "CaseH2-sanity-restored-content-matches-recorded-baseline-hash"
else
  bad "CaseH2-sanity-restored-content-matches-recorded-baseline-hash (recorded=$recorded_hash current=$current_hash)"
fi
git add case_h/binary.dat
assert_block "CaseH2-unchanged-binary-restored-exactly-is-accepted" ALLOW "$(try_commit 'case H2: binary restored to its recorded (pending) baseline')"

# ---------------------------------------------------------------------------
# CASE I (regression guard for the coordinator finding): committing a
# LARGE, heavily-NUL-populated binary file with NO baseline registered at
# all (the common case -- most binary commits are never mutation
# experiments) must produce ZERO "ignored null byte" warning on stdout or
# stderr, and must still succeed. This is the literal scenario the
# coordinator observed live (docs/workable_items.db, a real ~1.6MB SQLite
# file with ~275k NUL bytes) reproduced at smaller scale so the test stays
# fast and hermetic.
# ---------------------------------------------------------------------------
mkdir -p case_i
# 2000 repetitions of a 4-byte pattern incl. a NUL -> a real multi-NUL
# binary blob (8000 bytes, 2000 NUL bytes), no marker text anywhere in it.
i=0
: > case_i/big_binary.dat
while [ "$i" -lt 2000 ]; do
  printf 'AB\x00C' >> case_i/big_binary.dat
  i=$((i + 1))
done
git add case_i/big_binary.dat
rc=$(try_commit "case I: ordinary large binary, no baseline registered")
assert_block "CaseI-ordinary-large-binary-no-baseline-commits-cleanly" ALLOW "$rc"
# POLARITY-SWITCHED, same reasoning as CaseH1 above: the frozen pre-fix
# hook legitimately still leaks the warning here (this is layer 3a's
# pre-existing marker-sweep code path, unchanged on that snapshot).
if [ "$RED_MODE" = "1" ]; then
  if grep -qi "ignored null byte" "$HK_OUT" 2>/dev/null; then
    ok "CaseI-null-byte-warning-DOES-leak-on-FROZEN-pre-fix-hook (RED: reproduces the coordinator finding)"
  else
    bad "CaseI-null-byte-warning-DOES-leak-on-FROZEN-pre-fix-hook (RED: expected the pre-fix leak, saw none)"
  fi
else
  if grep -qi "ignored null byte" "$HK_OUT" 2>/dev/null; then
    bad "CaseI-no-null-byte-warning-on-ordinary-binary-commit (GREEN)"
  else
    ok "CaseI-no-null-byte-warning-on-ordinary-binary-commit (GREEN)"
  fi
fi

# ---------------------------------------------------------------------------
# CASE J (F1, POLARITY-SWITCHED, blocking finding from review): a staged
# WHOLE-FILE deletion of a file with an open, real-content baseline, left
# un-restored. `git diff --cached --name-only --diff-filter=ACMR` never
# contains a deleted path, so the ORIGINAL step 3b (iterating only that
# list) could not see this at all -- the limiting case of the exact residue
# class this item exists to close: an experiment deletes a guard FILE
# instead of a guard LINE, dies before restoring it, and a later broad
# `git add` stages the deletion.
#   RED_MODE=0 (live hook, has the --diff-filter=D pass) -> MUST be BLOCKED.
#   RED_MODE=1 (frozen pre-fix, no D-filter pass at all)  -> MUST be ALLOWED
#     (proves the historical gap was real: the file simply never appeared
#     in ANY list step 3b examined).
# ---------------------------------------------------------------------------
mkdir -p case_j
printf 'protective content\n' > case_j/guarded.sh
git add case_j/guarded.sh
git commit -q -m "case J: seed guarded.sh" 2>/dev/null
mutation_baseline_record case_j/guarded.sh >/dev/null 2>&1
git rm -q case_j/guarded.sh
rc=$(try_commit "case J: un-restored whole-file deletion")
if [ "$RED_MODE" = "1" ]; then
  assert_block "CaseJ-whole-file-deletion-on-FROZEN-pre-fix-hook (RED: D-filter absent, invisible to step 3b)" ALLOW "$rc"
else
  assert_block "CaseJ-whole-file-deletion-blocked-on-LIVE-fixed-hook (GREEN: F1 closed)" BLOCK "$rc"
  if grep -qF "case_j/guarded.sh" "$HK_OUT" 2>/dev/null && grep -qF "staged as DELETED" "$HK_OUT" 2>/dev/null; then
    ok "CaseJ-block-message-identifies-it-as-a-staged-deletion"
  else
    bad "CaseJ-block-message-identifies-it-as-a-staged-deletion"
  fi
fi
git reset -q HEAD -- case_j/guarded.sh 2>/dev/null || true
git checkout -q -- case_j/guarded.sh 2>/dev/null || true

# ---------------------------------------------------------------------------
# CASE K (F1 companion, explicitly requested in review: "confirm the ABSENT
# case still legitimately passes, so a baseline recorded on an absent file
# isn't turned into a permanent refusal"). A file is deleted from the
# working tree BEFORE the baseline is recorded (so the library's ABSENT
# sentinel is what gets captured), and that deletion is then STAGED. This
# must be ALLOWED in BOTH modes: on the live hook because an ABSENT
# baseline legitimately matches "still absent"; on the frozen pre-fix hook
# because it never examines deletions at all (never a false BLOCK either).
# ---------------------------------------------------------------------------
mkdir -p case_k
printf 'will be deleted before any baseline is even recorded\n' > case_k/tracked.sh
git add case_k/tracked.sh
git commit -q -m "case K: seed tracked.sh" 2>/dev/null
rm -f case_k/tracked.sh
mutation_baseline_record case_k/tracked.sh >/dev/null 2>&1
git add -A -- case_k/tracked.sh
rc=$(try_commit "case K: staged deletion matches ABSENT baseline")
assert_block "CaseK-staged-deletion-matching-ABSENT-baseline-is-accepted" ALLOW "$rc"

# ---------------------------------------------------------------------------
# CASE M (F2, REQUIRED finding from review): a genuine paired §1.1 mutation
# test proving the oracle itself is falsifiable. The reviewer's own mutation
# swapped step 3b's staged-blob oracle for a worktree read
# (`git show ":$f" | sha256sum` -> `sha256sum "$f"`); it survived all 18
# GREEN cases above because every fixture writes-then-immediately-`git
# add`s, so index and worktree never diverge anywhere else in this file.
# This section deliberately makes them diverge, in BOTH directions, and
# builds the reviewer's exact mutant to prove the divergence kills it.
#
# This section always tests the LIVE (fixed) hook and a mutant derived from
# it -- independent of the outer RED_MODE -- because the property under
# test ("does the oracle read the INDEX, never the worktree") is a fact
# about the current, fixed hook; the frozen pre-fix snapshot predates step
# 3b entirely and has no staged-blob oracle to mutate.
# ---------------------------------------------------------------------------
MUTANT_HOOK="$TMP/mutant_pre_commit.sh"
sed "s#staged_hash=\$(git show \":\$f\" 2>/dev/null | sha256sum 2>/dev/null | awk '{print \$1}')#staged_hash=\$(sha256sum \"\$f\" 2>/dev/null | awk '{print \$1}')#" \
  "$LIVE_HOOK" > "$MUTANT_HOOK"
if diff -q "$LIVE_HOOK" "$MUTANT_HOOK" >/dev/null 2>&1; then
  bad "CaseM-sanity-mutant-hook-actually-differs-from-live-hook (sed produced no change -- mutation did not apply)"
else
  ok "CaseM-sanity-mutant-hook-actually-differs-from-live-hook"
fi
chmod +x "$MUTANT_HOOK"

# oracle_probe <hook-file> <expect-L1> <expect-L2> <label>
#   Runs the staged<>worktree divergence pair against a FRESH throwaway repo
#   using <hook-file>, in a NESTED temp dir under $TMP (auto-cleaned by the
#   outer trap). Prints PASS/FAIL via ok()/bad() using the case names below.
oracle_probe() {
  local hookfile="$1" expect_l1="$2" expect_l2="$3" label="$4" probe_tmp
  probe_tmp=$(mktemp -d "$TMP/oracle_probe.XXXXXX" 2>/dev/null) || { bad "CaseM-$label-mktemp-failed"; return; }
  (
    cd "$probe_tmp" || exit 1
    git init -q
    git config user.email "test@example.com"; git config user.name "oracle probe"; git config commit.gpgsign false
    mkdir -p .git/hooks scripts/lib
    cp "$hookfile" .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit
    cp "$REPO_ROOT/scripts/lib/mutation_baseline.sh" scripts/lib/mutation_baseline.sh
    # shellcheck source=/dev/null
    . scripts/lib/mutation_baseline.sh
    echo seed >README.txt
    git add README.txt scripts/lib/mutation_baseline.sh
    git commit -q -m seed

    # L1: staged MUTATED / worktree RESTORED. Correct oracle (reads the
    # INDEX): BLOCK, because the mutated content is what is about to ship.
    printf 'original content\n' >oracle_check.sh
    git add oracle_check.sh
    git commit -q -m "L1 seed"
    mutation_baseline_record oracle_check.sh >/dev/null 2>&1
    printf 'MUTATED content, staged this way\n' >oracle_check.sh
    git add oracle_check.sh
    printf 'original content\n' >oracle_check.sh   # worktree restored, NOT re-staged
    git commit -q -m "L1 attempt" >l1_out.txt 2>&1
    printf '%s' "$?" >l1_rc.txt
    # L1 is EXPECTED to be correctly blocked on the real hook, which leaves
    # oracle_check.sh still staged with mismatched content -- unstage it so
    # it cannot bleed into L2's commit attempt below (the exact "leftover
    # staged state from a blocked commit" trap hit earlier in this file for
    # cases J/K).
    git reset -q HEAD -- oracle_check.sh 2>/dev/null || true

    # L2: staged CLEAN (matches a PENDING, not-yet-committed baseline) /
    # worktree MUTATED after staging, never staged. Correct oracle: ALLOW,
    # because what is about to ship (the index) matches the baseline: the
    # worktree's further unstaged edit is ordinary uncommitted work.
    printf 'v1 seed content\n' >oracle_check2.sh
    git add oracle_check2.sh
    git commit -q -m "L2 seed"
    printf 'v2 pending work, not yet committed\n' >oracle_check2.sh
    mutation_baseline_record oracle_check2.sh >/dev/null 2>&1
    printf 'v3 temp mutation\n' >oracle_check2.sh
    printf 'v2 pending work, not yet committed\n' >oracle_check2.sh
    git add oracle_check2.sh
    printf 'v2 pending work, not yet committed, BUT FURTHER MUTATED\n' >oracle_check2.sh
    git commit -q -m "L2 attempt" >l2_out.txt 2>&1
    printf '%s' "$?" >l2_rc.txt
  )
  local l1_rc l2_rc
  l1_rc=$(cat "$probe_tmp/l1_rc.txt" 2>/dev/null)
  l2_rc=$(cat "$probe_tmp/l2_rc.txt" 2>/dev/null)
  assert_block "CaseM-L1-staged-mutated-worktree-restored [$label]" "$expect_l1" "${l1_rc:-1}"
  assert_block "CaseM-L2-staged-clean-worktree-mutated [$label]" "$expect_l2" "${l2_rc:-1}"
  rm -rf "$probe_tmp" 2>/dev/null || true
}

# Against the MUTANT (worktree-based oracle): both verdicts must INVERT --
# L1 wrongly ALLOWED (worktree was restored, matches baseline), L2 wrongly
# BLOCKED (worktree was further mutated, mismatches baseline). This is the
# "mutation detected" half of the pair: if it did NOT invert, these two
# cases would not actually be capable of catching the reviewer's mutation.
oracle_probe "$MUTANT_HOOK" ALLOW BLOCK "mutant: mutation detected"

# Against the REAL, unmutated live hook: both verdicts must be correct --
# "invariant intact".
oracle_probe "$LIVE_HOOK" BLOCK ALLOW "real hook: invariant intact"

# ---------------------------------------------------------------------------
# CASE N (W2, round-3 review finding, POLARITY-SWITCHED): a baseline
# registered on a file INSIDE scripts/git_hooks/ -- including a helper
# living right beside the hook's own source -- with un-restored residue must
# be caught exactly like any other tracked file. Before this fix, step 3b
# inherited step 3a's `scripts/git_hooks/*` carve-out verbatim (needed
# THERE to keep the marker sweep from false-positiving on hook source that
# legitimately NAMES marker strings as documentation/fixtures). Step 3b's
# own discriminator is "does this file carry an OPEN, explicitly-registered
# baseline record" -- an opt-in condition with no such false-positive risk
# -- so the inherited carve-out silently exempted exactly this class of
# residue, reproducing the HXC-282 shape one directory deeper: a real
# mutation, explicitly registered and left un-restored, going undetected
# purely because of where the file happened to live.
#   RED_MODE=0 (live hook, carve-out removed)      -> MUST be BLOCKED.
#   RED_MODE=1 (frozen pre-fix, no step 3b at all)  -> MUST be ALLOWED (step
#     3b does not exist on that snapshot regardless of any carve-out, so
#     this is not a claim the carve-out removal "fixed" the frozen
#     snapshot -- it never had step 3b to begin with).
# ---------------------------------------------------------------------------
mkdir -p scripts/git_hooks
cat > scripts/git_hooks/case_n_test_helper.sh <<'HELPER'
#!/usr/bin/env bash
log_event() { printf '%s\n' "$1"; }
mask_secret() { printf '%s' "${1//[A-Za-z0-9]/*}"; }  # protective rule: mask credential-shaped values before logging
HELPER
git add scripts/git_hooks/case_n_test_helper.sh
git commit -q -m "case N: seed a helper under scripts/git_hooks/" 2>/dev/null

mutation_baseline_record scripts/git_hooks/case_n_test_helper.sh >/dev/null 2>&1
sed -i '/protective rule:/d' scripts/git_hooks/case_n_test_helper.sh
if grep -qE 'MUTATED for paired|// always pass|# always pass|MUTATION-RESIDUE|_mutated_' scripts/git_hooks/case_n_test_helper.sh; then
  bad "CaseN-sanity-fixture-must-leave-no-marker (fixture is broken -- would pass step 3a alone, not testing 3b)"
else
  ok "CaseN-sanity-fixture-leaves-no-marker"
fi

git add scripts/git_hooks/case_n_test_helper.sh
rc=$(try_commit "case N: un-restored deletion inside scripts/git_hooks/")
if [ "$RED_MODE" = "1" ]; then
  assert_block "CaseN-scripts-git-hooks-residue-on-FROZEN-pre-fix-hook (RED: step 3b absent entirely on that snapshot)" ALLOW "$rc"
else
  assert_block "CaseN-scripts-git-hooks-residue-blocked-on-LIVE-fixed-hook (GREEN: W2 carve-out removed)" BLOCK "$rc"
  if grep -qF "scripts/git_hooks/case_n_test_helper.sh" "$HK_OUT" 2>/dev/null; then
    ok "CaseN-block-message-names-the-file-inside-scripts-git-hooks"
  else
    bad "CaseN-block-message-names-the-file-inside-scripts-git-hooks"
  fi
fi
unstage scripts/git_hooks/case_n_test_helper.sh
git checkout -q -- scripts/git_hooks/case_n_test_helper.sh 2>/dev/null || true

# ---------------------------------------------------------------------------
# CASE O (W1, round-3 review finding): a MISSING scripts/lib/
# mutation_baseline.sh must not be a silent no-op. Before this fix, step
# 3b's absence and "no staged file has an open baseline record" (the
# overwhelming common case for an ordinary commit) were indistinguishable --
# a genuine tooling gap read exactly like nothing to report, the same
# §11.4.201 false-null shape this whole layer exists to catch elsewhere.
# This always tests the LIVE hook specifically (the frozen pre-fix snapshot
# predates step 3b and its MUTATION_BASELINE_LIB reference entirely, so
# there is no library-shaped thing to remove there), in its own isolated
# throwaway repo so deleting the library does not disturb any other case in
# this file -- same pattern as the CaseM oracle_probe helper above.
# ---------------------------------------------------------------------------
w1_probe_tmp=$(mktemp -d "$TMP/w1_probe.XXXXXX" 2>/dev/null)
if [ -n "$w1_probe_tmp" ]; then
  (
    cd "$w1_probe_tmp" || exit 1
    git init -q
    git config user.email "w1@example.com"; git config user.name "w1"; git config commit.gpgsign false
    mkdir -p .git/hooks
    cp "$LIVE_HOOK" .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit
    # Deliberately NO scripts/lib/mutation_baseline.sh anywhere in this repo.
    echo "ordinary content" > ordinary.txt
    git add ordinary.txt
    git commit -q -m "ordinary commit with the library absent" >w1_out.txt 2>&1
    echo "$?" >w1_rc.txt
  )
  w1_rc=$(cat "$w1_probe_tmp/w1_rc.txt" 2>/dev/null)
  assert_block "CaseO-ordinary-commit-not-blocked-when-library-absent (W1 is a WARN, never a BLOCK on its own)" ALLOW "${w1_rc:-1}"
  if grep -qF "scripts/lib/mutation_baseline.sh not found" "$w1_probe_tmp/w1_out.txt" 2>/dev/null; then
    ok "CaseO-notice-emitted-when-library-absent (W1: no longer a silent no-op)"
  else
    bad "CaseO-notice-emitted-when-library-absent (expected an explicit NOTICE naming the missing library on stderr/stdout)"
  fi
  rm -rf "$w1_probe_tmp" 2>/dev/null || true
else
  bad "CaseO-mktemp-failed"
fi

# ---------------------------------------------------------------------------
# CASE P (POLARITY-SWITCHED, round-9 FIX 1): step 3a's marker sweep must scan
# the STAGED BLOB, not the worktree. The gate on that loop used to be
# `[ -f "$f" ] || continue` -- a WORKTREE-existence test that flatly
# contradicted the comment directly above it ("Scan the STAGED blob (index),
# not the worktree") and handed the whole step a bypass:
#     printf ... > mutated.sh   # content carries `MUTATED for paired`
#     git add mutated.sh        # index now holds the residue blob
#     rm mutated.sh             # worktree file gone -> `[ -f ]` false -> skip
#     git commit                # commits the INDEX -> residue ships
# `git diff --cached ... ACMR` is index-vs-HEAD, so the path DID reach the
# loop and was then silently skipped. This is precisely §11.4.84's forensic
# shape: an experiment dies, ends by deleting its scratch file, and a later
# broad `git add` sweeps the residue in. Fixed to `[ -n "$f" ]`; the real
# index gate is the `git show ":$f"` capture below it.
#
# Round 9 proved this through a real `git commit` harness, but that harness
# lived outside the repo, so nothing standing prevented the bypass returning.
# This case registers it permanently (§11.4.135).
#
#   P-ATTACK  RED_MODE=1 (frozen pre-fix) -> MUST be ALLOWED (the defect).
#             RED_MODE=0 (live hook)      -> MUST be BLOCKED (defect closed).
#
# The three CLEAN sub-cases are LOAD-BEARING, not decoration: without them
# this case would be satisfied by a hook that simply blocks everything, which
# would "close" the bypass while making the repo uncommittable. P-CONTROL is
# equally load-bearing in the OTHER direction: it must BLOCK on the frozen
# hook too, so P-ATTACK's ALLOW there is attributable to the worktree gate
# specifically rather than to a sweep that was blind for some unrelated
# reason.
#
# PAYLOAD SIZE IS DELIBERATE AND MEASURED (2026-08-13). The frozen pre-fix
# fixture predates the HXC-309 pipe_grep fix (it still uses the raw
# `printf '%s' "$blob" | grep -qE` pipeline; `grep -c pipe_grep` -> 0 on the
# fixture vs 11 on the live hook), so on THAT artifact a large blob lands in
# the HXC-309 SIGPIPE fail-open race. Measured on this fixture, 10 reps each:
#     ~2 KiB payload : CONTROL=1 BLOCKED 10/10, ATTACK=0 ALLOWED 10/10
#                      -> the `[ -f ]` bypass is the ONLY variable. Usable.
#     56,828 B       : CONTROL=0 ALLOWED 10/10  (and ATTACK=0 10/10)
#                      -> the race swallows the control; ATTACK's ALLOW would
#                         then be indistinguishable from HXC-309 blindness and
#                         would prove nothing about this bypass. Unusable.
# Round 9's own 56,828-byte matrix was measured against the round-7 hook,
# which HAD pipe_grep -- it does not transfer to this frozen fixture. Hence a
# small payload here, with the size asserted below so a future edit that
# balloons it fails loudly instead of sliding silently into the race regime.
#
# CASEP_MAX_BYTES IS A CONSERVATIVE PROXY, NOT THE REAL PREDICATE. It bounds
# TOTAL FILE SIZE, whereas the hook's own analysis of HXC-309 (pre-commit,
# "POSITION, NOT SIZE, is what moves it") establishes that the driver is
# BYTES REMAINING AFTER THE MATCHING LINE. The two coincide here because the
# marker sits NEAR THE TOP of every fixture, close to the worst case, where
# bytes-after-match is maximal and total size is its tightest upper bound.
#
# ROUND-14 CORRECTION (HXC-329, F4): this sentence used to read "these
# fixtures keep the marker on line 2", and CASE Q below adopts this bound
# "for the same measured reason as CASE P" on the strength of it. That is
# true of `casep_payload` and FALSE of `caseq_payload`, which emits two
# header lines before the marker. Measured on both fixtures as they stand:
#     casep_payload  total=2550 B  marker_line=2  bytes_after_match=2480
#     caseq_payload  total=1155 B  marker_line=3  bytes_after_match=1060
# The direction is SAFE -- a marker one line further down leaves FEWER bytes
# after the match, so the bound is even more over-conservative for CASE Q
# than for CASE P, and no verdict is affected. Only the sentence licensing
# the reuse was wrong, and it is corrected rather than left to be trusted by
# a future reader who moves a fixture and checks it against a false premise.
# Measured with the marker moved to the TAIL instead, the same fixture blocks
# 10/10 at 16,450 / 32,830 / 56,880 B -- no race at any size. So a future edit
# that moved the marker down the payload would make this bound merely
# OVER-conservative (it would reject a payload that is in fact safe); it can
# never make it UNSAFE while the marker stays near the top. Keep the marker on
# line 2 and this bound honest; move it, and re-derive the bound rather than
# assuming this figure still means what it says.
# ---------------------------------------------------------------------------
CASEP_MAX_BYTES=8192

casep_payload() {   # $1=outfile  $2=with_marker(1|0)
  local out="$1" with="$2" i=0
  {
    printf '#!/usr/bin/env bash\n'
    if [ "$with" = "1" ]; then
      printf 'verify_token() { return 0; } # MUTATED for paired\n'
    fi
    while [ "$i" -lt 40 ]; do
      printf 'filler_%03d() { printf "x"; }  # multi-line padding, no marker\n' "$i"
      i=$((i+1))
    done
  } > "$out"
}

mkdir -p case_p

# --- fixture sanity: the marked payload carries the marker, the clean one
# --- does not, and both stay clear of the HXC-309 regime on the frozen hook.
casep_payload case_p/control.sh 1
casep_payload case_p/clean_probe.sh 0
casep_bytes=$(wc -c < case_p/control.sh 2>/dev/null | tr -d ' ')
if grep -qF 'MUTATED for paired' case_p/control.sh 2>/dev/null; then
  ok "CaseP-sanity-marked-payload-carries-the-marker"
else
  bad "CaseP-sanity-marked-payload-carries-the-marker (fixture broken -- CONTROL/ATTACK would pass for the wrong reason)"
fi
if grep -qE 'MUTATED for paired|// always pass|# always pass|MUTATION-RESIDUE|_mutated_' case_p/clean_probe.sh 2>/dev/null; then
  bad "CaseP-sanity-clean-payload-leaves-no-marker (fixture broken -- the CLEAN cases would not test over-blocking)"
else
  ok "CaseP-sanity-clean-payload-leaves-no-marker"
fi
if [ -n "$casep_bytes" ] && [ "$casep_bytes" -lt "$CASEP_MAX_BYTES" ]; then
  ok "CaseP-sanity-payload-below-HXC-309-race-regime ($casep_bytes B < $CASEP_MAX_BYTES B)"
else
  bad "CaseP-sanity-payload-below-HXC-309-race-regime ($casep_bytes B >= $CASEP_MAX_BYTES B -- on the frozen fixture the control would fail open and RED would prove nothing)"
fi
rm -f case_p/clean_probe.sh

# --- CLEAN-C precondition: a committed, marker-free file to genuinely delete.
casep_payload case_p/predel.sh 0
git add case_p/predel.sh
git commit -q -m "case P: seed a marker-free file for the genuine-deletion sub-case" >/dev/null 2>&1

# --- P-CONTROL: marker staged AND present in the worktree. The sweep's
# --- baseline behaviour -- MUST block in BOTH polarities.
git add case_p/control.sh
assert_block "CaseP-CONTROL-marker-staged-and-present-blocked (both polarities: proves the sweep is live)" BLOCK "$(try_commit 'case P control: residue staged, file present')"
unstage case_p/control.sh
rm -f case_p/control.sh

# --- P-ATTACK: the bypass. Same content staged, then removed from the
# --- worktree. Index is unchanged; only worktree existence differs.
casep_payload case_p/attack.sh 1
git add case_p/attack.sh
rm -f case_p/attack.sh
rc=$(try_commit 'case P attack: residue staged, worktree file rm-ed')
if [ "$RED_MODE" = "1" ]; then
  assert_block "CaseP-ATTACK-staged-residue-with-rm-ed-worktree-file-on-FROZEN-pre-fix-hook (RED: reproduces the [ -f ] bypass)" ALLOW "$rc"
else
  assert_block "CaseP-ATTACK-staged-residue-with-rm-ed-worktree-file-on-LIVE-fixed-hook (GREEN: bypass closed)" BLOCK "$rc"
  if grep -qF "case_p/attack.sh" "$HK_OUT" 2>/dev/null; then
    ok "CaseP-ATTACK-block-message-names-the-rm-ed-file"
  else
    bad "CaseP-ATTACK-block-message-names-the-rm-ed-file (hook output did not)"
  fi
fi
unstage case_p/attack.sh

# --- P-CLEAN-A: no marker, staged then rm-ed. The exact worktree state of
# --- the ATTACK case, minus the residue. MUST be allowed in both polarities:
# --- this is what proves the fix did not become "block any staged-then-
# --- deleted path".
casep_payload case_p/clean_a.sh 0
git add case_p/clean_a.sh
rm -f case_p/clean_a.sh
assert_block "CaseP-CLEAN-A-no-marker-staged-then-rm-ed-allowed (both polarities: no over-block on absent worktree file)" ALLOW "$(try_commit 'case P clean A: no residue, worktree file rm-ed')"

# --- P-CLEAN-B: no marker, staged, present. The ordinary commit. MUST be
# --- allowed in both polarities -- the hook must stay usable.
casep_payload case_p/clean_b.sh 0
git add case_p/clean_b.sh
assert_block "CaseP-CLEAN-B-no-marker-staged-and-present-allowed (both polarities: ordinary commit unaffected)" ALLOW "$(try_commit 'case P clean B: ordinary marker-free commit')"

# --- P-CLEAN-C: a genuine tracked deletion (`git rm`) of a marker-free file
# --- carrying no baseline record. MUST be allowed in both polarities: a
# --- legitimate delete is not residue.
git rm -q case_p/predel.sh >/dev/null 2>&1
assert_block "CaseP-CLEAN-C-genuine-git-rm-deletion-allowed (both polarities: a real delete is not residue)" ALLOW "$(try_commit 'case P clean C: genuine tracked deletion')"

# ---------------------------------------------------------------------------
# CASE Q (POLARITY-SWITCHED, HXC-324): the captured-evidence exemption's
# STAGED-MODE probe must be an EXACT lookup, not a pathspec match.
#
#     mode=$(git ls-files --stage -- "$f" | awk 'NR==1{print $1}')
#
# The `--` argument to `ls-files` is a PATHSPEC. A staged path containing a
# glob metacharacter (`?`, `*`, `[`) therefore matched SEVERAL index entries;
# `ls-files` sorts its output, so `awk NR==1` returned the mode of whichever
# SIBLING SORTED FIRST -- a different file than the one being judged. The
# decoy need only be TRACKED, not co-staged, so one pre-existing sibling in
# the same directory is enough.
#
# This is the exact hazard step 3a's own rationale already warns about ~46
# lines below the offending line, when it explains why `git show ":$f"` was
# chosen over "an `ls-files --stage` mode probe": `:PATH` is an exact object
# lookup, a pathspec is not. The warning was written; this line was never
# brought into line with it. Fixed with `:(literal)`, which disables
# pathspec globbing.
#
# BOTH POLARITIES OF THE SAME ROOT CAUSE ARE REGISTERED HERE:
#
#   Q-ATTACK-FAIL-OPEN  an EXECUTABLE (100755) residue-bearing evidence file
#     is granted the exemption on a 100644 decoy's mode, and the residue
#     ships. This defeats the exemption's own stated premise -- that the file
#     is inert and never executes, so a quoted marker cannot leak a bypass
#     into a running artifact.
#       RED_MODE=1 (frozen pre-fix) -> ALLOWED (the defect)
#       RED_MODE=0 (live hook)      -> BLOCKED (defect closed)
#
#   Q-ATTACK-OVER-BLOCK  a LEGITIMATE 100644 §11.4.83 transcript that merely
#     quotes a marker is denied the exemption on a 100755 decoy's mode and is
#     wrongly BLOCKED -- a usability failure on exactly the captured evidence
#     §11.4.83 requires be committed.
#       RED_MODE=1 (frozen pre-fix) -> BLOCKED (the defect)
#       RED_MODE=0 (live hook)      -> ALLOWED (defect closed)
#
# THE CONTROLS ARE LOAD-BEARING, not decoration. Each attack has a control
# that is byte-identical in every respect EXCEPT that no decoy is tracked --
# same metacharacter filename, same mode, same payload. Without them, a hook
# that simply blocked (or simply exempted) everything in docs/qa would
# "satisfy" the attacks for entirely the wrong reason. The two BASELINE
# sub-cases below anchor the other end: they prove, on ORDINARY filenames,
# that the exemption is reachable at all and that the mode gate is live -- so
# a future change that broke metacharacter handling outright could not hide
# behind a dead exemption path.
#
# Payload size is held below CASEP_MAX_BYTES for the same measured reason as
# CASE P: the frozen pre-fix fixture predates the HXC-309 pipe_grep fix, so a
# large blob there lands in the SIGPIPE fail-open race and every verdict
# becomes unattributable. Fixtures carry real newlines (a single-line blob
# cannot exercise the race regime the size bound exists to avoid).
# ---------------------------------------------------------------------------
mkdir -p docs/qa

# A §11.4.83-shaped captured-evidence transcript: no `#!` first line (that
# would disqualify it as script-shaped at check (iv), testing the wrong gate).
caseq_payload() {   # $1=outfile  $2=with_marker(1|0)
  local out="$1" with="$2" i=0
  {
    printf 'HXC-324 captured evidence transcript\n'
    printf 'run-id: caseq\n'
    if [ "$with" = "1" ]; then
      printf 'observed in hook output: MUTATED for paired\n'
    fi
    while [ "$i" -lt 20 ]; do
      printf 'evidence line %02d: ordinary narrative text, no marker\n' "$i"
      i=$((i+1))
    done
  } > "$out"
}

# stage/unstage/remove by EXACT path -- these helpers take pathspecs too, so
# a metacharacter name would over-match here as well and silently stage or
# reset the decoy, destroying the very asymmetry under test.
caseq_add()     { git add -- ":(literal)$1"; }
caseq_unstage() { git reset -q HEAD -- ":(literal)$1" 2>/dev/null || true; }

caseq_payload docs/qa/q_marked_probe.log 1
caseq_payload docs/qa/q_clean_probe.log 0
caseq_bytes=$(wc -c < docs/qa/q_marked_probe.log 2>/dev/null | tr -d ' ')
if grep -qF 'MUTATED for paired' docs/qa/q_marked_probe.log 2>/dev/null; then
  ok "CaseQ-sanity-marked-transcript-carries-the-marker"
else
  bad "CaseQ-sanity-marked-transcript-carries-the-marker (fixture broken -- every CaseQ verdict would be unattributable)"
fi
if [ -n "$caseq_bytes" ] && [ "$caseq_bytes" -lt "$CASEP_MAX_BYTES" ]; then
  ok "CaseQ-sanity-payload-below-HXC-309-race-regime ($caseq_bytes B < $CASEP_MAX_BYTES B)"
else
  bad "CaseQ-sanity-payload-below-HXC-309-race-regime ($caseq_bytes B >= $CASEP_MAX_BYTES B -- on the frozen fixture the verdicts would be race artefacts)"
fi
rm -f docs/qa/q_marked_probe.log docs/qa/q_clean_probe.log

# --- Q-BASELINE-EXEMPT: ordinary filename, 100644, carries a marker. The
# --- exemption's happy path. MUST be allowed in BOTH polarities -- if this
# --- ever fails, the exemption is dead and the attacks below prove nothing.
caseq_payload docs/qa/q5_plain.log 1
caseq_add docs/qa/q5_plain.log
assert_block "CaseQ-BASELINE-plain-name-100644-evidence-exempted (both polarities: the exemption path is live)" ALLOW "$(try_commit 'case Q baseline: ordinary evidence transcript, exempt')"
# ROUND-14 (HXC-329, F5): unstage + remove, like every other CASE Q case.
# This case and Q-CONTROL-2 were the only two that left their fixture staged;
# an ALLOW case is exactly where it matters, because a committed-or-still-
# staged leftover silently joins the index of every LATER case in this file
# and can flip an unrelated verdict. Proved by mutation H2 (drop `log` from
# the exemption extension list): the run went PASS=51 FAIL=5, and two of the
# five failures were CaseR-gitlink-only-commit-allowed and
# CaseR-gitlink-to-present-object-allowed -- COLLATERAL, caused by the
# leftover staged fixture rather than by the mutated code. That is the same
# cross-case coupling round 12 removed for R4-vs-HEAD.
caseq_unstage docs/qa/q5_plain.log
rm -f docs/qa/q5_plain.log

# --- Q-BASELINE-MODE: ordinary filename, 100755, carries a marker. The mode
# --- gate itself. MUST be blocked in BOTH polarities.
caseq_payload docs/qa/q6_plain.log 1
chmod +x docs/qa/q6_plain.log
caseq_add docs/qa/q6_plain.log
assert_block "CaseQ-BASELINE-plain-name-100755-evidence-blocked (both polarities: the staged-mode gate is live)" BLOCK "$(try_commit 'case Q baseline: executable evidence, not exempt')"
caseq_unstage docs/qa/q6_plain.log
rm -f docs/qa/q6_plain.log

# --- Q-CONTROL-1: metacharacter filename, 100755, marker, NO decoy tracked.
# --- Identical to Q-ATTACK-1 in every respect but the decoy. MUST block in
# --- BOTH polarities: this is what makes the attack's ALLOW attributable to
# --- the pathspec over-match rather than to metacharacter names generally.
caseq_payload 'docs/qa/q3_?.log' 1
chmod +x 'docs/qa/q3_?.log'
caseq_add 'docs/qa/q3_?.log'
assert_block "CaseQ-CONTROL-1-metachar-name-100755-no-decoy-blocked (both polarities: isolates the decoy as the only variable)" BLOCK "$(try_commit 'case Q control 1: executable metachar-named evidence, no decoy')"
caseq_unstage 'docs/qa/q3_?.log'
rm -f 'docs/qa/q3_?.log'

# --- Q-ATTACK-1 (FAIL-OPEN): tracked 100644 decoy `q1_0.log` sorts BEFORE
# --- the staged 100755 `q1_?.log` ('0'=0x30 < '?'=0x3F), so `awk NR==1`
# --- reads 100644 and the exemption is granted to an EXECUTABLE file.
caseq_payload docs/qa/q1_0.log 0
caseq_add docs/qa/q1_0.log
git commit -q -m "case Q: seed the 100644 decoy that sorts first" >/dev/null 2>&1
caseq_payload 'docs/qa/q1_?.log' 1
chmod +x 'docs/qa/q1_?.log'
caseq_add 'docs/qa/q1_?.log'
rc=$(try_commit 'case Q attack 1: executable residue exempted on the decoy mode')
if [ "$RED_MODE" = "1" ]; then
  assert_block "CaseQ-ATTACK-1-FAIL-OPEN-executable-residue-exempted-on-FROZEN-pre-fix-hook (RED: reproduces the pathspec over-match)" ALLOW "$rc"
else
  assert_block "CaseQ-ATTACK-1-FAIL-OPEN-executable-residue-exempted-on-LIVE-fixed-hook (GREEN: exact :(literal) lookup, fail-open closed)" BLOCK "$rc"
  if grep -qF 'q1_?.log' "$HK_OUT" 2>/dev/null; then
    ok "CaseQ-ATTACK-1-block-message-names-the-metachar-named-file"
  else
    bad "CaseQ-ATTACK-1-block-message-names-the-metachar-named-file (hook output did not)"
  fi
fi
caseq_unstage 'docs/qa/q1_?.log'
rm -f 'docs/qa/q1_?.log'

# --- Q-CONTROL-2: metacharacter filename, 100644, marker, NO decoy tracked.
# --- Identical to Q-ATTACK-2 but for the decoy. MUST be allowed in BOTH
# --- polarities, making the attack's BLOCK attributable to the over-match.
caseq_payload 'docs/qa/q4_?.log' 1
caseq_add 'docs/qa/q4_?.log'
assert_block "CaseQ-CONTROL-2-metachar-name-100644-no-decoy-exempted (both polarities: isolates the decoy as the only variable)" ALLOW "$(try_commit 'case Q control 2: metachar-named evidence, no decoy')"
# ROUND-14 (HXC-329, F5): see the note at Q-BASELINE-EXEMPT. `docs/qa/q4_?.log`
# staying staged past this point is the specific leftover that made the two
# CASE R gitlink assertions fail as collateral under mutation H2.
caseq_unstage 'docs/qa/q4_?.log'
rm -f 'docs/qa/q4_?.log'

# --- Q-ATTACK-2 (OVER-BLOCK): tracked 100755 decoy `q2_0.log` sorts BEFORE
# --- the staged 100644 `q2_?.log`, so `awk NR==1` reads 100755 and a
# --- legitimate §11.4.83 transcript is refused the exemption and blocked.
caseq_payload docs/qa/q2_0.log 0
chmod +x docs/qa/q2_0.log
caseq_add docs/qa/q2_0.log
git commit -q -m "case Q: seed the 100755 decoy that sorts first" >/dev/null 2>&1
caseq_payload 'docs/qa/q2_?.log' 1
caseq_add 'docs/qa/q2_?.log'
rc=$(try_commit 'case Q attack 2: legitimate evidence refused on the decoy mode')
if [ "$RED_MODE" = "1" ]; then
  assert_block "CaseQ-ATTACK-2-OVER-BLOCK-legitimate-evidence-refused-on-FROZEN-pre-fix-hook (RED: reproduces the opposite polarity)" BLOCK "$rc"
else
  assert_block "CaseQ-ATTACK-2-OVER-BLOCK-legitimate-evidence-refused-on-LIVE-fixed-hook (GREEN: exact :(literal) lookup, over-block closed)" ALLOW "$rc"
fi
caseq_unstage 'docs/qa/q2_?.log'
rm -f 'docs/qa/q2_?.log'

# ---------------------------------------------------------------------------
# CASE R (BOTH POLARITIES, HXC-282 round-12): step 3a's sweep must handle
# STAGED GITLINKS (submodule pointer entries, mode 160000).
#
# The round-9 rewrite replaced the loop's `[ -f "$f" ]` gate with the
# `git show ":$f"` capture, and its rationale block records that a gitlink
# yields rc=128 and is therefore skipped. That rc=128 is what keeps every
# submodule-pointer bump committable -- and NOTHING asserted it. Submodule
# bumps are routine in this repo (three submodules are modified in the very
# tree this case ships in), so the sweep's correctness on gitlinks is a
# common path, not an exotic one. This case pins it.
#
# WHY rc=128, stated precisely (measured 2026-08-13, git 2.50.1): it is NOT
# a property of the 160000 mode. `git show ":<path>"` resolves the index
# entry's object; for a gitlink that object is a COMMIT belonging to the
# SUBMODULE, which a superproject does not carry -- so the lookup fails with
# rc=128 and the entry is skipped. Verified on the real repo: `git show
# ":submodules/helix_agent"` -> rc=128.
#
# The converse was also measured, and the surface is LARGER than "the
# gitlink is skipped" suggests. If that commit object DOES happen to be
# present locally, `git show ":<gitlink>"` succeeds (rc=0) and emits the
# commit header, the commit MESSAGE, and THE COMMIT'S FULL PATCH -- all of
# which the sweep then greps like any other blob. Measured 2026-08-13, both
# variants BLOCKED the superproject's pointer bump, with the gitlink path
# named in the output:
#     marker in the pointed-at commit's MESSAGE  -> BLOCKED
#     marker only in the pointed-at commit's DIFF -> BLOCKED
# That is arguably the right call (a mutation marker reachable through a
# submodule bump is worth surfacing), it is NOT reachable in this repo (the
# superproject does not carry submodule objects, so the lookup is the rc=128
# path), and this round changes none of it -- so it is recorded here as a
# measured observation and deliberately left UNPINNED rather than frozen
# into a contract by an assertion.
#
# R4 therefore uses a PURPOSE-BUILT, provably marker-free commit rather than
# `HEAD`. An earlier draft pointed R4 at `git rev-parse HEAD`, which coupled
# it to whatever the preceding case happened to leave there: under the R3
# mutation below, R3's commit succeeded and became HEAD, its patch contained
# R3's residue fixture, and R4 then blocked -- failing for a reason that had
# nothing to do with what R4 asserts. The sanity check below pins the
# independence instead of assuming it.
#
# R3 is the load-bearing one: it proves the skip is a `continue` for that
# ONE entry and not an early exit that would leave the rest of a mixed
# commit unscanned -- the shape in which a gitlink bump could silently
# escort real residue past the sweep.
#
# R3'S RESIDUE PATH IS DELIBERATELY `zz_case_r_residue.sh`, AND THE NAME IS
# THE WHOLE POINT. The sweep iterates `git diff --cached --name-only` output,
# which git emits in index order -- i.e. sorted by path. R3 can only detect
# an early exit if the GITLINK is reached FIRST and the residue AFTER it. The
# first draft of this case used `case_p/caser_residue.sh`, which sorts BEFORE
# `submodules/caser_absent` ('c' < 's'): the residue was scanned first, block
# was already set, and the paired §1.1 mutation (`|| continue` -> `|| break`
# on the gitlink skip) left the case PASSING -- a case that looked load-
# bearing and asserted nothing. Caught by running that mutation; recorded
# here so the trap is not re-entered. `zz_` sorts after `submodules/`, so the
# gitlink is now genuinely upstream of the residue in the iteration, and the
# mutation kills this assertion (measured: `break` -> R3 FAILs, "expected
# BLOCK, hook ALLOWED", and nothing else fails).
# ---------------------------------------------------------------------------
# A gitlink whose target commit is absent from this repo's object store --
# i.e. exactly what a real submodule pointer looks like from the superproject.
caser_absent_sha=0123456789abcdef0123456789abcdef01234567
git update-index --add --cacheinfo "160000,$caser_absent_sha,submodules/caser_absent" 2>/dev/null

git show ":submodules/caser_absent" >/dev/null 2>&1
caser_rc=$?
if [ "$caser_rc" -eq 128 ]; then
  ok "CaseR-gitlink-git-show-returns-128 (the documented mechanism the sweep's skip rests on)"
else
  bad "CaseR-gitlink-git-show-returns-128 (got rc=$caser_rc -- the skip documented in pre-commit step 3a no longer holds)"
fi

# --- R2: a pure submodule-pointer bump. MUST be allowed in BOTH polarities.
assert_block "CaseR-gitlink-only-commit-allowed (both polarities: submodule bumps stay committable)" ALLOW "$(try_commit 'case R: submodule pointer bump only')"

# --- R3: the same gitlink staged ALONGSIDE a marker-bearing ordinary file.
# --- MUST block in BOTH polarities: skipping the gitlink must not abort the
# --- sweep and leave the rest of the commit unscanned.
git update-index --add --cacheinfo "160000,1111111111111111111111111111111111111111,submodules/caser_absent" 2>/dev/null
casep_payload zz_case_r_residue.sh 1
git add zz_case_r_residue.sh
# The precondition R3 actually needs is RELATIVE order -- the gitlink must be
# reached BEFORE the residue -- not an absolute position, which other staged
# paths would legitimately shift.
caser_gl_pos=$(git diff --cached --name-only --diff-filter=ACMR | grep -nxF 'submodules/caser_absent' | cut -d: -f1)
caser_rs_pos=$(git diff --cached --name-only --diff-filter=ACMR | grep -nxF 'zz_case_r_residue.sh' | cut -d: -f1)
if [ -n "$caser_gl_pos" ] && [ -n "$caser_rs_pos" ] && [ "$caser_gl_pos" -lt "$caser_rs_pos" ]; then
  ok "CaseR-sanity-gitlink-precedes-residue-in-sweep-order (gitlink #$caser_gl_pos before residue #$caser_rs_pos)"
else
  bad "CaseR-sanity-gitlink-precedes-residue-in-sweep-order (gitlink #${caser_gl_pos:-absent} vs residue #${caser_rs_pos:-absent} -- R3 below cannot detect an early exit unless the gitlink is scanned FIRST)"
fi
assert_block "CaseR-gitlink-does-not-derail-the-sweep (both polarities: residue staged AFTER a gitlink is still caught)" BLOCK "$(try_commit 'case R: gitlink bump plus real residue')"
unstage zz_case_r_residue.sh
rm -f zz_case_r_residue.sh

# --- R4: a gitlink whose target commit IS present locally (the rc=0 branch).
# --- Purpose-built and marker-free in BOTH message and patch, so the ALLOW
# --- is attributable to the branch not false-blocking rather than to
# --- whatever the previous case left at HEAD.
casep_payload caser_clean_target.txt 0
git add caser_clean_target.txt
git commit -q -m "case R: marker-free target commit for the present-object gitlink" >/dev/null 2>&1
caser_present_sha=$(git rev-parse HEAD 2>/dev/null)
git update-index --add --cacheinfo "160000,$caser_present_sha,submodules/caser_present" 2>/dev/null
if git show ":submodules/caser_present" 2>/dev/null | grep -qE 'MUTATED for paired|// always pass|# always pass|MUTATION-RESIDUE|_mutated_'; then
  bad "CaseR-sanity-present-object-target-is-marker-free (the target commit's message+patch carries a marker -- R4's verdict would not be about the rc=0 branch at all)"
else
  ok "CaseR-sanity-present-object-target-is-marker-free (message and patch both clean)"
fi
assert_block "CaseR-gitlink-to-present-object-allowed (both polarities: the rc=0 branch does not false-block)" ALLOW "$(try_commit 'case R: gitlink to a locally present commit')"

# ---------------------------------------------------------------------------
# CASE S (POLARITY-SWITCHED, HXC-329/F1): PATH QUOTING. `git diff --cached
# --name-only` C-QUOTES any path containing a non-ASCII byte, a double quote,
# a backslash, or a control character (`core.quotePath` defaults to TRUE), so
# the sweep received `"r\303\251sidu.sh"` -- quotes and octal escapes and all
# -- instead of the path. `git show ":$f"` then failed rc=128 and the
# `|| continue` skipped the file ENTIRELY, in silence. Fail-OPEN: residue
# ships. No decoy, no tracked sibling, no size threshold; one accented
# character in a filename is the whole exploit.
#
# Measured on the pre-fix hooks, payload and mode byte-identical, FILENAME
# THE ONLY VARIABLE:
#     FROZEN pre-fix   ascii_resid.sh -> BLOCK    résidu.sh -> ALLOW
#     SHIPPED round-13 ascii_resid.sh -> BLOCK    résidu.sh -> ALLOW
#     PATCHED round-14 ascii_resid.sh -> BLOCK    résidu.sh -> BLOCK
# and the ALLOW really did commit the residue:
#     git show 'HEAD:résidu.sh' -> `# residue: MUTATED for paired`
#
# Every S-ATTACK case therefore has an ASCII CONTROL asserting IDENTICALLY in
# BOTH polarities, so a failure can only be attributed to the filename.
# ---------------------------------------------------------------------------
mkdir -p case_s

# Same shape as casep_payload (marker near the top, real newlines, well under
# CASEP_MAX_BYTES so no verdict here can land in the HXC-309 race regime).
cases_payload() {   # $1=outfile  $2=with_marker(1|0)
  local out="$1" with="$2" i=0
  {
    printf '#!/usr/bin/env bash\n'
    if [ "$with" = "1" ]; then
      printf 'verify_token() { return 0; } # MUTATED for paired\n'
    fi
    while [ "$i" -lt 20 ]; do
      printf 'filler_%03d() { printf "x"; }  # multi-line padding, no marker\n' "$i"
      i=$((i+1))
    done
  } > "$out"
}
cases_add()     { git add -- ":(literal)$1"; }
cases_unstage() { git reset -q HEAD -- ":(literal)$1" 2>/dev/null || true; }

# --- S-CONTROL: plain ASCII name, marker. MUST block in BOTH polarities --
# --- if this ever fails, the sweep is dead and the attacks prove nothing.
cases_payload case_s/ascii_resid.sh 1
cases_add case_s/ascii_resid.sh
assert_block "CaseS-CONTROL-ascii-name-marker-blocked (both polarities: proves the sweep is live, isolates the filename)" BLOCK "$(try_commit 'case S control: residue under an ASCII name')"
cases_unstage case_s/ascii_resid.sh
rm -f case_s/ascii_resid.sh

# --- S-ATTACK-*: identical payload, quoted-class names. RED reproduces the
# --- fail-open; GREEN proves `-z` + array collection closed it.
cases_attack() {   # $1=label  $2=filename
  local label="$1" fn="$2" rc
  cases_payload "$fn" 1
  cases_add "$fn"
  rc=$(try_commit "case S attack: residue under a quoted-class name ($label)")
  if [ "$RED_MODE" = "1" ]; then
    assert_block "CaseS-ATTACK-${label}-residue-skipped-on-FROZEN-pre-fix-hook (RED: reproduces the path-quoting fail-open)" ALLOW "$rc"
  else
    assert_block "CaseS-ATTACK-${label}-residue-caught-on-LIVE-fixed-hook (GREEN: -z + array collection, fail-open closed)" BLOCK "$rc"
  fi
  cases_unstage "$fn"
  rm -f -- "$fn"
}
cases_attack nonascii  'case_s/résidu.sh'
cases_attack tab       "$(printf 'case_s/tab\tx.sh')"
cases_attack dquote    'case_s/dqu"ote.sh'
cases_attack backslash 'case_s/back\slash.sh'
# A NEWLINE in a filename was not merely quoted but UNREPRESENTABLE in the old
# newline-joined list: it split into two bogus entries. Only the GREEN hook can
# be asked about it coherently, so this asserts on the live hook alone.
if [ "$RED_MODE" != "1" ]; then
  cases_attack newline "$(printf 'case_s/nl\nx.sh')"
fi

# --- S-CLEAN: quoted-class names, NO marker. MUST be allowed in BOTH
# --- polarities: the fix must not over-block ordinary non-ASCII filenames.
cases_clean() {   # $1=label  $2=filename
  local label="$1" fn="$2" rc
  cases_payload "$fn" 0
  cases_add "$fn"
  rc=$(try_commit "case S clean: ordinary marker-free file ($label)")
  assert_block "CaseS-CLEAN-${label}-no-marker-allowed (both polarities: no over-block on unusual filenames)" ALLOW "$rc"
  cases_unstage "$fn"
  rm -f -- "$fn"
}
cases_clean nonascii 'case_s/ordinaire_é.sh'
cases_clean tab      "$(printf 'case_s/ok\ty.sh')"

# --- S-3B: BLAST RADIUS beyond step 3a. `$staged` also keys step 3b's
# --- `mutation_baseline_is_open "$f"` lookup, so a quoted name reinstated
# --- the HXC-282 defect class ITSELF: a DELETION-shaped mutation carries no
# --- marker phrase at all, so step 3a can never see it and only the baseline
# --- comparison can. Measured, same mutation, filename the only variable:
# ---     SHIPPED round-13  guard.sh -> BLOCK    gardé.sh -> ALLOW (residue committed)
# ---     PATCHED round-14  guard.sh -> BLOCK    gardé.sh -> BLOCK
# --- Asserted on the LIVE hook only, and deliberately so: the frozen pre-fix
# --- fixture predates HXC-282 and contains NO step 3b whatsoever (measured:
# --- zero `mutation_baseline` references in it), so its ALLOW would be
# --- over-determined -- true for a superset of reasons -- and would not
# --- attribute anything to path quoting. An honest gap, not a skipped case.
if [ "$RED_MODE" != "1" ]; then
  for s_name in 's_guard.sh' 's_gardé.sh'; do
    printf 'guard() { return 1; }\nreal_logic\n' > "case_s/$s_name"
    cases_add "case_s/$s_name"
    try_commit "case S 3b: register $s_name" >/dev/null
    ( . scripts/lib/mutation_baseline.sh; mutation_baseline_record "case_s/$s_name" >/dev/null 2>&1 )
    # Deletion-shaped mutation: the guard is disabled, NO marker phrase added.
    printf 'guard() { return 0; }\nreal_logic\n' > "case_s/$s_name"
    cases_add "case_s/$s_name"
    assert_block "CaseS-3B-${s_name}-unrestored-baseline-caught (step 3b keyed on the real path, not the quoted one)" BLOCK "$(try_commit "case S 3b: un-restored mutation in $s_name")"
    cases_unstage "case_s/$s_name"
    rm -f -- "case_s/$s_name"
  done
fi

# ---------------------------------------------------------------------------
# CASE T (HXC-329/F2): PATHSPEC MAGIC vs THE ENVIRONMENT. Round 12 fixed the
# HXC-324 decoy hole by making the exemption's mode probe an exact
# `:(literal)$f` pathspec. `GIT_LITERAL_PATHSPECS=1` disables magic-prefix
# parsing outright, so `:(literal)` stops being a prefix and becomes part of
# the path being matched: the lookup returns nothing, `$mode` is empty, and
# the exemption is DENIED. Fail-CLOSED -- a legitimate §11.4.83 transcript
# refused -- i.e. round 12's own fix broken by an environment variable.
# Measured end-to-end on the round-13 shipped hook, environment the ONLY
# variable: no env -> ALLOWED (exempt); GIT_LITERAL_PATHSPECS=1 -> BLOCKED.
#
# Asserted in BOTH polarities, but note what each proves: the frozen pre-fix
# fixture probes with a RAW pathspec (`-- "$f"`, measured -- no magic prefix
# at all), so it was never exposed and passes trivially. F2 was introduced BY
# round 12 and exists only in the round-13 shipped hook, which is not in
# testdata -- so this is a standing GREEN guard, not a polarity-switched RED
# reproduction, and its falsifiability rests on the paired §1.1 mutation
# (strip the `env -u` and this case FAILS on the live hook).
#
# `GIT_GLOB_PATHSPECS=1` is asserted too: no pathspec form is env-immune on
# its own (`git --literal-pathspecs ls-files` survives the first variable but
# returns zero rows under this one), which is why the fix removes the
# variables from the child's environment rather than choosing another spelling.
# ---------------------------------------------------------------------------
caset_try_commit() {   # $1=message  $2...=VAR=VAL env assignments
  local msg="$1"; shift
  env "$@" git commit -q -m "$msg" >"$HK_OUT" 2>&1
  echo $?
}
caseq_payload docs/qa/t1_evidence.log 1
caseq_add docs/qa/t1_evidence.log
assert_block "CaseT-evidence-exempt-under-GIT_LITERAL_PATHSPECS (both polarities: magic-prefix parsing disabled must not deny the exemption)" ALLOW "$(caset_try_commit 'case T: evidence transcript, GIT_LITERAL_PATHSPECS=1' GIT_LITERAL_PATHSPECS=1)"
caseq_unstage docs/qa/t1_evidence.log
rm -f docs/qa/t1_evidence.log

caseq_payload docs/qa/t2_evidence.log 1
caseq_add docs/qa/t2_evidence.log
assert_block "CaseT-evidence-exempt-under-GIT_GLOB_PATHSPECS (both polarities: no pathspec spelling is env-immune, so the env is cleared)" ALLOW "$(caset_try_commit 'case T: evidence transcript, GIT_GLOB_PATHSPECS=1' GIT_GLOB_PATHSPECS=1)"
caseq_unstage docs/qa/t2_evidence.log
rm -f docs/qa/t2_evidence.log

# Control: the SAME transcript with NO env override must be exempt too, so a
# CaseT failure can only be attributed to the environment variable.
caseq_payload docs/qa/t3_evidence.log 1
caseq_add docs/qa/t3_evidence.log
assert_block "CaseT-CONTROL-evidence-exempt-with-no-env-override (both polarities: isolates the env var as the only variable)" ALLOW "$(try_commit 'case T control: evidence transcript, no env override')"
caseq_unstage docs/qa/t3_evidence.log
rm -f docs/qa/t3_evidence.log

echo
echo "=== HXC-282 summary (RED_MODE=$RED_MODE, $ACTIVE_HOOK_LABEL): PASS=$PASS FAIL=$FAIL ==="
if [ "$FAIL" -ne 0 ]; then
  printf 'FAILED CASES:\n'
  for c in "${FAILED_CASES[@]}"; do printf '  - %s\n' "$c"; done
  exit 1
fi
exit 0
