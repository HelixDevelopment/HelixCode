#!/usr/bin/env bash
# scripts/git_hooks/test_precommit_secret_scanner_unusable.sh
# HXC-282 / F1 — §11.4.115 RED-first polarity test + §11.4.135 standing
# regression guard for scripts/git_hooks/pre-commit STEP 4 (the staged-content
# credential scanner) failing OPEN when its own instrument is unusable.
#
# THE DEFECT (RED_MODE=1 reproduces it on the frozen pre-fix artifact):
#   Step 4's guard was a bare `if [ -x "$SECRET_SCANNER" ]; then … fi` with NO
#   `else`. An absent or non-executable scanner made the ENTIRE step evaporate
#   — no banner, no notice, exit 0 — and a staged, key-shaped credential was
#   COMMITTED. This is the §11.4.201 false-null shape at the single most
#   expensive seam in the hook: step 4 has NO sibling layer (step 3b may
#   degrade to a NOTICE because step 3a still runs; step 4 is the ONLY
#   credential check at the commit seam), and a committed credential is in
#   history the instant it lands — CONST-042 / §12.1 make it a release blocker
#   requiring rotation and a post-mortem.
#
# THE FIX (live in scripts/git_hooks/pre-commit): the guard is
#   `[ -f ] && [ -r ] && [ -x ]`, and a new `else` branch REFUSES (`block=1`)
#   with a per-state reason — `not found` / `not a regular file` /
#   `not readable (permissions)` / `not executable (chmod +x it)`. The hook
#   already `exit 0`s when nothing is staged, so this refusal can only fire
#   when there is real staged content to scan (CASE 8 below proves exactly
#   that). The audited escape is unchanged: `git commit --no-verify` plus the
#   commit-msg hook's mandatory `Bypass-rationale:` footer (§11.4.75).
#
# POLARITY SWITCH (§11.4.115): ONE test script, TWO hook artifacts.
#   RED_MODE=0 (default) — the standing GREEN regression guard: installs the
#     REAL, CURRENT scripts/git_hooks/pre-commit. Every unusable-scanner state
#     MUST be BLOCKED, with the precise per-state diagnostic.
#   RED_MODE=1 — reproduces the historical defect on the FROZEN pre-fix
#     artifact scripts/git_hooks/testdata/hxc282_f1_pre_fix_step4_pre_commit.sh.
#     The same states MUST fail open (or block only ACCIDENTALLY — see the
#     measured matrix below) — proving the defect was real and that this
#     test's oracle is capable of catching it.
#   Both artifact sha256 values are RE-DERIVED at runtime and the run ABORTS
#   loudly on any mismatch, so this test can never silently validate a
#   different artifact than the one it documents.
#
# ---------------------------------------------------------------------------
# FROZEN ARTIFACT — CONSTRUCTION RECIPE (deliberately documented)
# ---------------------------------------------------------------------------
# A frozen artifact whose construction is unspecified is un-auditable: a
# reader cannot tell whether it is a faithful pre-fix snapshot or something
# hand-edited into agreeing with the test. This is a deliberate lesson carried
# over from a separate finding, so the recipe is recorded here in full. It was
# produced MECHANICALLY from the POST-FIX hook by exactly these three steps:
#
#   1. Replace the line
#        if [ -f "$SECRET_SCANNER" ] && [ -r "$SECRET_SCANNER" ] && [ -x "$SECRET_SCANNER" ]; then
#      with
#        if [ -x "$SECRET_SCANNER" ]; then
#      (asserted: EXACTLY 1 substitution).
#   2. Delete the whole `else` branch — from the line `else` followed by
#        `  # The instrument is unusable. REFUSE`
#      through and including the `  block=1` + `fi` that close it — replacing
#      the removed block with a bare `fi`.
#   3. Assert: 0 remaining occurrences of `The instrument is unusable`; the
#      mutated sha256 differs from the pristine sha256; `bash -n` passes.
#
# WHY FROZEN AT ALL, AND WHY THE RECIPE IS THE ONLY AUDIT TRAIL. The ORIGINAL
# pre-fix bytes (sha256_16 `73346fa49074f6d2`) were an UNCOMMITTED WORKING-TREE
# state — they never reached git history and are therefore NOT recoverable by
# `git show`, `git log -S`, or any other history query (§11.4.6: this is a
# stated limitation, not an assumption). That is precisely why the artifact is
# frozen in testdata/ and why its construction is documented here rather than
# left to be re-derived. Re-deriving a "pre-fix" hook from history would also
# stop meaning "pre-fix" the moment the fix is committed.
#
# ---------------------------------------------------------------------------
# MEASURED MATRIX (2026-08-13, real `git commit` through both artifacts)
# ---------------------------------------------------------------------------
# The originally-reported matrix covered the two states the defect was found
# with. Measuring all four states through the frozen artifact CORRECTS it: two
# of the four do NOT fail open, because `[ -x ]` is TRUE for a directory
# (search permission) and TRUE for a 0300 file (owner execute) — so the frozen
# hook proceeds to EXECUTE the path, the exec fails, and the non-zero status is
# misreported through the SCAN-FAILURE banner as though the scanner had found
# something. It blocks, but ACCIDENTALLY and UNDIAGNOSED:
#
#   scanner state   LIVE (fixed)                      FROZEN (pre-fix)
#   ------------    ------------------------------    -----------------------------
#   present+exec    BLOCK  (real scan finding)        BLOCK  (real scan finding)
#   chmod -x        BLOCK  "not executable"           ALLOW — leak.md COMMITTED
#   deleted         BLOCK  "not found"                ALLOW — leak.md COMMITTED
#   directory       BLOCK  "not a regular file"       BLOCK — "Is a directory"
#   chmod 0300      BLOCK  "not readable (perms)"     BLOCK — "Permission denied"
#
# So the rows are split by WHAT is polarity-switched, and each is asserted on
# the strongest honest oracle available for it:
#   • chmod -x / deleted  → the VERDICT switches. RED must ALLOW and the leak
#     must actually LAND in the throwaway commit.
#   • directory / 0300    → the verdict is BLOCK in BOTH. Asserting the verdict
#     alone here would be a bluff: it would "pass" in RED for a reason that has
#     nothing to do with the fix. What genuinely switches is the DIAGNOSIS —
#     GREEN refuses deliberately and names the exact state; RED emits neither
#     the UNUSABLE banner nor any `reason:` line, and blocks only because an
#     exec error leaked into the scan-failure banner. That is what is asserted.
#
# ---------------------------------------------------------------------------
# PER-ROW LIVENESS CANARIES (mandatory — an inert step reads as a passing row)
# ---------------------------------------------------------------------------
# This defect existed partly because "step 4 produced no output" and "step 4
# found nothing" were indistinguishable. A row that asserts ALLOW proves
# nothing unless the hook demonstrably RAN in that state, so every row carries
# TWO step-4-INDEPENDENT canaries, both fired by a single probe commit made in
# the row's own repo and scanner state:
#   canary-B (step 2): a content-free `canary.pem` is staged and a commit
#     attempted. Step 2 matches on the NAME alone, so it MUST return non-zero
#     with a `forbidden class` banner. Then reset + removed.
#   canary-A (step 3b): `scripts/lib/mutation_baseline.sh` is deliberately NOT
#     copied into the throwaway repo, so step 3b MUST emit its
#     `step 3b (baseline-comparison residue check) SKIPPED` NOTICE.
# A row whose canary fails FAILS the test rather than passing silently.
# CASE 8 additionally carries a third, genuinely in-row canary — the hook's own
# `ATMO_PRECOMMIT_RAN` marker — because that row's whole point is that the hook
# produces NO output, and silence is exactly what an inert hook also produces.
#
# CASE 8 — CORRECTED (round-24, F-F). Earlier revisions of this block claimed
# the empty-index row "cannot carry an IN-ROW canary", on the reasoning that
# the hook exits at the staged-empty guard before any step runs and emits 0
# bytes. The premise (0 bytes) is true; the CONCLUSION drawn from it was FALSE,
# and it was falsified by reading the hook rather than reasoning about it:
#
#   scripts/git_hooks/pre-commit writes a purpose-built run-marker
#       : > "$GITDIR/ATMO_PRECOMMIT_RAN" 2>/dev/null || true
#   at the TOP of the hook body — measured at line 54, and critically ~380
#   lines AHEAD of the staged-empty guard (`exit 0` at line 438). It exists so
#   scripts/git_hooks/commit-msg can detect a --no-verify bypass, and the
#   frozen pre-fix artifact carries it at the same position, so it is available
#   in BOTH polarities.
#
# So EVERY row can carry a true in-row liveness canary, the empty-index row
# included: clear the marker immediately before the attempt, and assert it
# exists afterwards. That converts "the hook emitted 0 bytes" from an ambiguous
# silence — indistinguishable from a hook that never ran, which is the very
# confusion that let this defect survive — into "the hook ran and deliberately
# said nothing". The adjacent canary probe is RETAINED as well (it proves step
# 2 and step 3b are reachable in this repo and scanner state); the marker is an
# additional, genuinely IN-ROW witness rather than a replacement.
#
# The row is still run with the scanner DELETED — the state that otherwise
# refuses — so rc=0 can only mean the empty-index guard short-circuited ahead
# of step 4.
#
# Verified by paired §1.1 mutation: deleting ONLY the marker-write line from
# the hook (nothing else) makes this run FAIL with exactly one failure, the
# `run-marker ABSENT after the attempt` assertion.
#
# ANTI-BLUFF: every verdict comes from a REAL `git commit` against the real
# hook bytes (or the frozen artifact) in a throwaway `mktemp -d` repo — never a
# re-grep of hook source. The credential fixture is key-SHAPED but NEVER REAL:
# it is assembled at RUNTIME from separate fragments and its value is NEVER
# printed, echoed, or written to any log by this script.
#
# ...and "the real hook bytes" is ENFORCED, not assumed: `core.hooksPath` can
# silently redirect git to a different file, so every throwaway repo verifies
# that the hook git will actually execute is byte-identical to the pinned
# artifact (see the four-layer note at the top of the code). Measured with a
# QUIET decoy — a behaviourally identical copy of the hook — this is the ONLY
# check that catches the substitution: with the decoy in place, 49 of 57
# assertions still passed and only the 8 identity checks failed.
#
# COMPLETION CONTRACT: the run ends by asserting that every case executed and
# that the sweep is the expected SIZE. Assertion counts are how this file
# proves it measured what it claims; without that check a dropped case left no
# trace at all (measured: a deleted case still reported `53 passed, 0 failed`,
# exit 0). PASS / FAIL / SKIP are all counted — a SKIP is an outcome that is
# genuinely not assertable, never a quiet way to shrink the sweep.
#
# MIRROR-ONLY: this script never writes to the live hook, the live scanner,
# `.scan-secrets-allow`, or any real repo state; it never stages, commits, or
# pushes anything in the real repository, and never uses --no-verify. Both live
# artifacts' sha256 values are asserted unchanged at the end of the run.
#
# Usage:   scripts/git_hooks/test_precommit_secret_scanner_unusable.sh
#          RED_MODE=1 scripts/git_hooks/test_precommit_secret_scanner_unusable.sh
# Exit:    0 iff every case passed for the active RED_MODE; 1 otherwise.
# Dependencies: git, bash, mktemp, sha256sum (or shasum), seq.
# Cross-references: §11.4.115 / §11.4.135 / §11.4.201 / §11.4.236 / §11.4.84 /
#   CONST-042 / §12.1 / §1.1; scripts/git_hooks/pre-commit (step 4);
#   scripts/secret_scan.sh; scripts/git_hooks/testdata/
#   hxc282_f1_pre_fix_step4_pre_commit.sh; sibling polarity tests
#   scripts/git_hooks/test_mutation_baseline_deletion_detection.sh and
#   scripts/git_hooks/test_mutation_residue_evidence_exempt.sh (this file does
#   not duplicate their cases).

set -uo pipefail

# ---------------------------------------------------------------------------
# core.hooksPath NEUTRALISATION, LAYER 1 of 4 — hostile inherited environment.
#
# This test pins its subject hook by sha256 and PRINTS that sha. But `git` does
# not have to run the hook at `.git/hooks/pre-commit`: `core.hooksPath`
# redirects hook lookup to an arbitrary directory, and it can arrive from a
# global/system config file OR from the environment. Left undefended, a run can
# truthfully report the pinned sha while every commit executes DIFFERENT BYTES.
#
# MEASURED (2026-08-13, git 2.50.1), against this suite unmodified:
#   * loud decoy   (`exit 0` hook) via GIT_CONFIG_GLOBAL -> PASS=12 FAIL=35,
#     and `leak.md LANDED` 10 times: the credential fixture reached commits.
#   * QUIET decoy  (a byte-appended COPY of the live hook, sha d2dceebdcc7bd53e)
#     via GIT_CONFIG_GLOBAL -> `live hook sha256_16 = b86464bb3bebb859` and
#     `47 passed, 0 failed`, exit 0. A fully green sweep that validated an
#     artifact it never executed. THAT is the bluff this defence closes; the
#     loud decoy is merely its noisy cousin.
#
# MEASURED PRECEDENCE (each probed directly, never assumed — §11.4.6):
#   command-line `-c`  >  GIT_CONFIG_COUNT/KEY_n/VALUE_n env  >  repo-local
#   .git/config  >  global (incl. GIT_CONFIG_GLOBAL)  >  system.
# So repo-local config alone is NOT sufficient: with GIT_CONFIG_COUNT set, the
# decoy ran and the commit was ALLOWED despite a repo-local pin. Unsetting
# GIT_CONFIG_COUNT alone neutralises KEY_n/VALUE_n (measured: git ignores them
# when the count is absent), and our own `-c` outranks the env vector outright.
#
# Hence four independent layers, no one of which is trusted alone:
#   L1 (here)  strip every config/layout-redirecting variable from the env.
#   L2 new_repo() pins repo-local core.hooksPath to the repo's own hooks dir.
#   L3 git_hardened() passes `-c core.hooksPath=...` — highest precedence — on
#      EVERY hook-running invocation, so even L1+L2 being bypassed loses.
#   L4 assert_hook_identity() verifies, per repo, that the file git WILL run
#      is byte-identical to the pinned artifact. L4 is the load-bearing one:
#      L1-L3 enumerate known vectors, L4 checks the CONTENT actually reached
#      and therefore catches vectors nobody enumerated.
# ---------------------------------------------------------------------------
unset GIT_CONFIG_COUNT GIT_CONFIG_PARAMETERS GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM \
      GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY \
      GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_COMMON_DIR GIT_NAMESPACE \
      GIT_CEILING_DIRECTORIES 2>/dev/null || true

RED_MODE="${RED_MODE:-0}"
HOOKS_SRC_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$HOOKS_SRC_DIR/../.." && pwd)
LIVE_HOOK="$HOOKS_SRC_DIR/pre-commit"
FROZEN_HOOK="$HOOKS_SRC_DIR/testdata/hxc282_f1_pre_fix_step4_pre_commit.sh"
LIVE_SCANNER="$REPO_ROOT/scripts/secret_scan.sh"

# Expected identities, re-derived at runtime and asserted below (never
# hardcode-and-trust: a test that validates whatever happens to be on disk
# while claiming to validate a specific artifact is itself a bluff).
# Round-24, re-pinned 2026-08-13. Superseded pins, in order, all observed
# WITHIN A SINGLE WORKING SESSION: live b86464bb3bebb859 -> f0e7b0150034e3b2
# -> be4f3bc1a01f584d, frozen 2d1499fee0fe1b52 -> 77f57695a2fea98d ->
# 7ff0d624adff6d78. A sibling stream owns scripts/git_hooks/pre-commit and was
# landing round-24 findings (F-A: `is_mutation_test_exempt` now anchors its
# `trap … EXIT` candidates to COMMAND POSITION, so a quoted example inside a
# string can no longer grant an exemption; F-C: a correction to the step-4
# header prose), re-deriving the frozen artifact from the new live hook by the
# recipe above each time. Both artifacts moved TOGETHER and coherently.
#
# RE-PINNED ONLY AFTER RE-RUNNING THIS FILE'S OWN CHECKLIST — each time, on the
# actual new bytes, never on the assumption that "the change was probably
# elsewhere" (§11.4.6). Measured on the currently-pinned bytes:
#   * step-4 guard `[ -f ] && [ -r ] && [ -x ]`                 — 1 occurrence
#   * else-branch marker `The instrument is unusable. REFUSE`   — 1 occurrence
#   * per-state `secret_scanner_why=` assignments               — 5 occurrences
#   * `ATMO_PRECOMMIT_RAN` marker write (CASE 8's canary)       — 1, at line 54
#   * staged-empty guard `exit 0`                               — line 438
#   * live vs frozen diff — 51 lines in exactly 2 hunks, BOTH inside the step-4
#     region, so the polarity still cannot be confounded by unrelated drift.
#     (Deliberately described by REGION, not by @@ line numbers: those shift
#     with every unrelated edit above them and would rot into a false claim.)
# The round-24 edits landed in `is_mutation_test_exempt` and in comments — a
# DIFFERENT function and prose — so step 4, this guard's whole subject, is
# byte-unaffected.
#
# HONEST BOUNDARY (§11.4.6): a pin is a moment-in-time fact about a file this
# stream does NOT own, and it moved twice in one session. If it FATALs, that is
# the assertion WORKING, not a flake — redo the checklist above and re-pin; do
# not relax it. For a measurement that must be reproducible while the subject
# is in flight, copy the pinned artifacts into a layout-faithful mirror
# (<root>/scripts/git_hooks/{pre-commit,testdata/} + <root>/scripts/
# secret_scan.sh) and run this script from there: HOOKS_SRC_DIR and REPO_ROOT
# are both derived from $0, so a mirror resolves correctly with no edits.
EXPECT_LIVE_HOOK_SHA16="be4f3bc1a01f584d"
EXPECT_FROZEN_HOOK_SHA16="7ff0d624adff6d78"

GIT_ID=(-c user.email=hxc282-f1-test@example.invalid -c user.name="HXC-282 F1 Test" -c commit.gpgsign=false)

PASS=0
FAIL=0
SKIP=0
declare -a FAILED_CASES=()
declare -a CASES_RAN=()

ok()  { PASS=$((PASS+1)); printf 'PASS  %s\n' "$1"; }
bad() { FAIL=$((FAIL+1)); FAILED_CASES+=("$1"); printf 'FAIL  %s\n' "$1"; }
# skip() is for an outcome that is genuinely NOT assertable in this run — never
# a way to make an inconvenient assertion disappear. It deliberately does NOT
# increment PASS, and it IS counted by the completion assertion, so a skip can
# never quietly shrink the sweep the way a stray `ok` quietly inflated it.
skip() { SKIP=$((SKIP+1)); printf 'SKIP  %s\n' "$1"; }

# case_ran <slug> — registered at the END of a case body, so it records only
# cases that actually reached completion. A case aborted at setup never
# registers, and the completion assertion below names it.
case_ran() { CASES_RAN+=("$1"); }

# ---------------------------------------------------------------------------
# COMPLETION CONTRACT — what a finished run MUST have produced.
#
# WHY THIS EXISTS. Before it, a case could vanish and the run still reported a
# clean 47/0 sweep: `repo=$(new_repo …)` runs in a command-substitution
# SUBSHELL, so a setup failure's FAIL increment was discarded in the parent and
# its message was captured into `$repo` instead of printed. Nothing anywhere
# compared "how many assertions did we make" against "how many were we supposed
# to make", so a SHRUNKEN sweep and a COMPLETE sweep were indistinguishable —
# the §11.4.201 false-null shape applied to the harness itself rather than to
# the hook under test.
#
# These two constants are an INDEPENDENT statement of the expected work. They
# are deliberately hardcoded and derived by hand below, NOT computed from the
# same loops that emit the assertions — a self-computed expectation would move
# in lockstep with a dropped case and prove nothing.
#
#   per case                       GREEN  RED
#   ---------------------------    -----  ---
#   hook identity (L4)                 1    1
#   canary-B + canary-A                2    2
#   Cases 1-4 only: fixture shape      1    1
#   verdict (assert_block)             1    1
#   leak-landed                        1    1
#   GREEN: UNUSABLE banner + reason    2    -
#   RED:   no-diagnostic               -    1
#   RED:   accidental exec error       -    1  (dir + mode0300 rows ONLY)
#
#   Cases 1-4  = 4 x 8 = 32   |  4 x 7 = 28, +2 exec-error rows      = 30
#   working    = 1+2+1+1+1    =  6   (fixture shape is not asserted here)
#   clean      = 1+2+1        =  4
#   gitlink    = 1+2+1+1      =  5
#   empty      = 1+2+1+1+1+1+1=  8   (index-empty, marker-absent, verdict,
#                                     silent-exit, marker-present)
#   live-file integrity        =  2   (hook unchanged; scanner unchanged-or-SKIP)
#   ---------------------------------------------------------------------
#   TOTAL (excluding the two completion assertions themselves)  57   55
#
# The two completion assertions are EXCLUDED from the constants and land after
# the comparison, so the final summary reads 59 (GREEN) / 57 (RED). Counting
# them inside their own expectation would make the check partly self-fulfilling.
# ---------------------------------------------------------------------------
EXPECTED_ASSERTIONS_GREEN=57
EXPECTED_ASSERTIONS_RED=55
EXPECTED_CASES=(noexec deleted dir mode0300 working clean gitlink empty)

sha16() {   # $1=path -> first 16 hex chars of sha256, or empty
  local h=""
  if command -v sha256sum >/dev/null 2>&1; then
    h=$(sha256sum "$1" 2>/dev/null)
  elif command -v shasum >/dev/null 2>&1; then
    h=$(shasum -a 256 "$1" 2>/dev/null)
  fi
  printf '%s' "${h:0:16}"
}

# assert_block <case-name> <expected: BLOCK|ALLOW> <actual-exit>
assert_block() {
  local name="$1" expect="$2" rc="$3"
  if [ "$expect" = "BLOCK" ]; then
    if [ "$rc" -ne 0 ]; then ok "$name (blocked as expected)"
    else bad "$name (expected BLOCK, hook ALLOWED rc=$rc)"; fi
  else
    if [ "$rc" -eq 0 ]; then ok "$name (allowed as expected)"
    else bad "$name (expected ALLOW, hook BLOCKED rc=$rc)"; fi
  fi
}

# --- preconditions -----------------------------------------------------------
for f in "$LIVE_HOOK" "$FROZEN_HOOK" "$LIVE_SCANNER"; do
  if [ ! -f "$f" ]; then
    echo "FATAL: required artifact missing: $f" >&2
    exit 1
  fi
done

START_LIVE_HOOK_SHA=$(sha16 "$LIVE_HOOK")
START_FROZEN_SHA=$(sha16 "$FROZEN_HOOK")
START_SCANNER_SHA=$(sha16 "$LIVE_SCANNER")

if [ "$START_LIVE_HOOK_SHA" != "$EXPECT_LIVE_HOOK_SHA16" ]; then
  echo "FATAL: live hook sha256_16 mismatch." >&2
  echo "  expected: $EXPECT_LIVE_HOOK_SHA16   actual: $START_LIVE_HOOK_SHA" >&2
  echo "  $LIVE_HOOK changed since this guard was written. Re-verify the step-4" >&2
  echo "  guard + else branch, re-measure the matrix in this file's header, and" >&2
  echo "  update EXPECT_LIVE_HOOK_SHA16 — do NOT relax this assertion." >&2
  exit 1
fi
if [ "$START_FROZEN_SHA" != "$EXPECT_FROZEN_HOOK_SHA16" ]; then
  echo "FATAL: frozen pre-fix artifact sha256_16 mismatch." >&2
  echo "  expected: $EXPECT_FROZEN_HOOK_SHA16   actual: $START_FROZEN_SHA" >&2
  echo "  $FROZEN_HOOK is not the artifact this test documents. A frozen RED" >&2
  echo "  baseline that silently changed proves nothing; rebuild it per the" >&2
  echo "  construction recipe in this file's header." >&2
  exit 1
fi

TMP=$(mktemp -d 2>/dev/null) || { echo "FATAL: mktemp failed" >&2; exit 1; }
cleanup() {
  # Directory-shaped fixtures are created inside $TMP; chmod them back so rm
  # can always descend, then remove the single tree this script created.
  chmod -R u+rwX "$TMP" 2>/dev/null || true
  rm -rf "$TMP" 2>/dev/null || true
}
trap cleanup EXIT

if [ "$RED_MODE" = "1" ]; then
  ACTIVE_HOOK="$FROZEN_HOOK"
  ACTIVE_LABEL="frozen pre-fix artifact"
  ACTIVE_HOOK_SHA="$START_FROZEN_SHA"
  EXPECTED_ASSERTIONS="$EXPECTED_ASSERTIONS_RED"
else
  ACTIVE_HOOK="$LIVE_HOOK"
  ACTIVE_LABEL="live (fixed) hook"
  ACTIVE_HOOK_SHA="$START_LIVE_HOOK_SHA"
  EXPECTED_ASSERTIONS="$EXPECTED_ASSERTIONS_GREEN"
fi

echo "=== HXC-282/F1 pre-commit secret-scanner-unusable guard — RED_MODE=$RED_MODE ($ACTIVE_LABEL) ==="
echo "    live hook   sha256_16 = $START_LIVE_HOOK_SHA"
echo "    frozen hook sha256_16 = $START_FROZEN_SHA"
echo "    live scanner sha256_16 = $START_SCANNER_SHA"
echo

# make_leak_fixture <path>
#   Writes a file containing a key-SHAPED but NEVER-REAL Google credential,
#   assembled at runtime from separate fragments so the literal never exists
#   in this script's source, in any log, or on any terminal. The value is
#   never printed by this script.
make_leak_fixture() {
  local out="$1" kp kb
  kp='AIza'
  kb=$(printf 'A%.0s' $(seq 1 35))
  {
    printf 'incident notes\n'
    printf 'observed token: %s%s\n' "$kp" "$kb"
    printf 'end of notes\n'
  } > "$out"
}

# git_hardened <repo> <git args...>
#   core.hooksPath NEUTRALISATION, LAYER 3 — the SINGLE place any hook-running
#   git command is issued. `-c` is the highest-precedence config source
#   (measured: it beats GIT_CONFIG_COUNT/KEY_n, which in turn beats repo-local
#   config), so this pins hook lookup to the repo's own hooks dir even against
#   a hostile inherited environment that survived layer 1.
#
#   It is a FUNCTION, not a copied flag, precisely so the flags cannot drift:
#   assert_hook_identity() resolves the hook path through this same wrapper, so
#   the path it verifies is BY CONSTRUCTION the path do_commit() will execute.
#   Two hand-maintained copies of the flags could silently diverge, and then
#   the verification would be checking a path nothing runs.
git_hardened() {
  local repo="$1"; shift
  git -C "$repo" "${GIT_ID[@]}" -c core.hooksPath="$repo/.git/hooks" "$@"
}

# assert_hook_identity <repo> <case-label>
#   core.hooksPath NEUTRALISATION, LAYER 4 — the load-bearing check, and the
#   only one that does not depend on having enumerated the threat correctly.
#   Layers 1-3 block the vectors that are KNOWN; this asks the question that
#   actually matters: are the bytes git is about to execute the bytes this test
#   claims to be validating?
#
#   It resolves the effective hook path with `rev-parse --git-path`, which
#   honours core.hooksPath (verified directly: under a hostile override it
#   returns the decoy path, not `.git/hooks`), then compares the CONTENT hash
#   at that path against the ACTIVE artifact. A redirect to a decoy fails on
#   the path; a substituted file at the right path fails on the hash. The
#   quiet-decoy bluff — a behaviourally identical COPY of the hook, which
#   passes every functional case while the summary prints the pinned sha — is
#   caught here and nowhere else, because it is invisible to every behavioural
#   assertion in this file.
assert_hook_identity() {
  local repo="$1" label="$2" hookpath actual
  hookpath=$(git_hardened "$repo" rev-parse --git-path hooks/pre-commit 2>/dev/null)
  if [ -z "$hookpath" ] || [ ! -f "$hookpath" ]; then
    bad "$label :: hook-identity — git resolves no runnable pre-commit hook (path='${hookpath:-<empty>}'); this row's verdict is NOT attributable to the pinned artifact"
    return 1
  fi
  # Resolve relative to the repo: `--git-path` may return a repo-relative path.
  case "$hookpath" in
    /*) : ;;
    *)  hookpath="$repo/$hookpath" ;;
  esac
  actual=$(sha16 "$hookpath")
  if [ "$actual" = "$ACTIVE_HOOK_SHA" ]; then
    ok "$label :: hook-identity — git will execute the PINNED $ACTIVE_LABEL ($actual)"
  else
    bad "$label :: hook-identity — git resolves a DIFFERENT hook than the pinned $ACTIVE_LABEL (expected $ACTIVE_HOOK_SHA, resolved $actual at $hookpath); core.hooksPath redirect or substituted file — this run would report the pinned sha while executing other bytes"
  fi
}

# new_repo <slug> -> echoes the path of a fresh throwaway repo with the ACTIVE
#   hook and a live-scanner COPY installed. scripts/lib/mutation_baseline.sh is
#   deliberately NOT copied (that absence is canary-A).
#
#   The repo is named from the CALLER-SUPPLIED slug, never from a counter.
#   This is deliberate: `repo=$(new_repo)` runs in a command-substitution
#   SUBSHELL, so any counter incremented inside would be discarded in the
#   parent and every case would silently reuse ONE directory — inheriting the
#   previous case's scanner state and committed fixtures. (Measured: that bug
#   made cases 5-8 fail because the `dir` case left a DIRECTORY at the scanner
#   path for every later case.) Slugs must be unique; that is asserted below
#   rather than assumed.
#
#   THE SAME SUBSHELL PROPERTY APPLIES TO FAILURE REPORTING, and this function
#   used to get it wrong: it called `bad` on the collision path. That `bad` ran
#   in the subshell, so (i) its FAIL increment was discarded in the parent and
#   (ii) its `FAIL …` line was CAPTURED INTO `$repo` instead of being printed —
#   a setup failure that registered nowhere and left the caller holding a
#   non-path string. It therefore reports failures the only way a
#   command-substituted function honestly can: a diagnostic on STDERR (which is
#   not captured) plus a NON-ZERO RETURN, leaving stdout empty. Registering the
#   FAIL is the CALLER's job, in the parent shell, where the counter lives.
new_repo() {
  local slug="$1"
  local repo="$TMP/repo_$slug"
  if [ -e "$repo" ]; then
    printf 'new_repo: slug collision on "%s" — a reused repo would inherit the previous case'"'"'s state and invalidate its verdict\n' "$slug" >&2
    return 1
  fi
  mkdir -p "$repo/.git" "$repo/scripts" || { printf 'new_repo: mkdir failed for %s\n' "$repo" >&2; return 1; }
  git init -q -b main "$repo" >/dev/null 2>&1 || { printf 'new_repo: git init failed for %s\n' "$repo" >&2; return 1; }
  mkdir -p "$repo/.git/hooks" || { printf 'new_repo: hooks mkdir failed\n' >&2; return 1; }
  cp "$ACTIVE_HOOK" "$repo/.git/hooks/pre-commit" || { printf 'new_repo: hook copy failed\n' >&2; return 1; }
  chmod +x "$repo/.git/hooks/pre-commit" || { printf 'new_repo: hook chmod failed\n' >&2; return 1; }
  cp "$LIVE_SCANNER" "$repo/scripts/secret_scan.sh" || { printf 'new_repo: scanner copy failed\n' >&2; return 1; }
  chmod +x "$repo/scripts/secret_scan.sh" || { printf 'new_repo: scanner chmod failed\n' >&2; return 1; }
  # core.hooksPath NEUTRALISATION, LAYER 2 — pin hook lookup to this repo's own
  # hooks dir in repo-local config, which outranks any global/system setting
  # (measured). L3 covers the env vector that outranks even this.
  git -C "$repo" config core.hooksPath "$repo/.git/hooks" >/dev/null 2>&1 \
    || { printf 'new_repo: could not pin core.hooksPath\n' >&2; return 1; }
  printf 'seed\n' > "$repo/README.md" || { printf 'new_repo: seed write failed\n' >&2; return 1; }
  git -C "$repo" add README.md >/dev/null 2>&1 || { printf 'new_repo: seed add failed\n' >&2; return 1; }
  # Seed commit goes through the SAME hardened invocation as every measured
  # commit, so a redirected hook cannot run even during setup.
  git_hardened "$repo" commit -q -m "seed" >/dev/null 2>&1 \
    || { printf 'new_repo: seed commit failed\n' >&2; return 1; }
  printf '%s' "$repo"
}

# apply_scanner_state <repo> <state>
apply_scanner_state() {
  local repo="$1" state="$2"
  case "$state" in
    present)  : ;;
    noexec)   chmod -x "$repo/scripts/secret_scan.sh" ;;
    deleted)  rm -f "$repo/scripts/secret_scan.sh" ;;
    dir)      rm -f "$repo/scripts/secret_scan.sh"; mkdir -p "$repo/scripts/secret_scan.sh" ;;
    mode0300) chmod 0300 "$repo/scripts/secret_scan.sh" ;;
    *)        bad "internal-unknown-scanner-state:$state"; return 1 ;;
  esac
}

# do_commit <repo> <outfile> <message> [extra git-commit args...]
#   Runs a REAL commit and reports the exit code DIRECTLY (never through a
#   pipeline, where $? would describe the last pipeline stage instead).
do_commit() {
  local repo="$1" out="$2" msg="$3"; shift 3
  git_hardened "$repo" commit -q "$@" -m "$msg" >"$out" 2>&1
  echo $?
}

# run_canaries <repo> <case-label>
#   One probe commit in the row's own repo + scanner state fires BOTH
#   step-4-independent canaries. Leaves the index as it found it.
run_canaries() {
  local repo="$1" label="$2" out rc slug
  slug="${repo##*/repo_}"
  out="$TMP/canary_${slug}.txt"
  : > "$repo/canary.pem"
  git -C "$repo" add canary.pem >/dev/null 2>&1
  rc=$(do_commit "$repo" "$out" "canary probe (must be refused by step 2)")

  if [ "$rc" -ne 0 ] && grep -qF 'forbidden class' "$out" 2>/dev/null; then
    ok "$label :: canary-B step-2 forbidden-class fired (hook is LIVE in this state)"
  else
    bad "$label :: canary-B step-2 forbidden-class fired (rc=$rc, banner missing -- the hook may not have run at all, so this row's verdict is NOT attributable to step 4)"
  fi

  if grep -qF 'step 3b (baseline-comparison residue check) SKIPPED' "$out" 2>/dev/null; then
    ok "$label :: canary-A step-3b NOTICE emitted (hook reached step 3b)"
  else
    bad "$label :: canary-A step-3b NOTICE emitted (absent -- hook did not reach step 3b)"
  fi

  git -C "$repo" reset -q HEAD -- canary.pem >/dev/null 2>&1 || true
  rm -f "$repo/canary.pem"
}

# leak_landed <repo> -> YES|NO  (did leak.md actually reach the commit?)
leak_landed() {
  local n
  n=$(git -C "$1" ls-tree --name-only HEAD -- leak.md 2>/dev/null)
  if [ "$n" = "leak.md" ]; then echo YES; else echo NO; fi
}

# ---------------------------------------------------------------------------
# CASES 1-4 — POLARITY-DEPENDENT: the four ways the step-4 instrument can be
# unusable, each with the leak fixture staged. See the measured matrix in the
# header for why rows 1-2 switch on the VERDICT while rows 3-4 switch on the
# DIAGNOSIS (their verdict is BLOCK in both polarities, and asserting only the
# verdict there would pass in RED for a reason unrelated to this fix).
#
# Table columns:
#   state | live verdict | frozen verdict | frozen leak lands | live `reason:` text | frozen accidental-exec evidence
# ---------------------------------------------------------------------------
CASES_1_4=(
  "noexec|BLOCK|ALLOW|YES|not executable (chmod +x it)|"
  "deleted|BLOCK|ALLOW|YES|not found|"
  "dir|BLOCK|BLOCK|NO|not a regular file|Is a directory"
  "mode0300|BLOCK|BLOCK|NO|not readable (permissions)|Permission denied"
)

for row in "${CASES_1_4[@]}"; do
  IFS='|' read -r st live_v frozen_v frozen_leak reason exec_err <<<"$row"

  # A setup failure must REGISTER (new_repo cannot: it runs in a subshell) and
  # then abort THIS case, so no assertion below ever measures a garbage repo.
  repo=$(new_repo "$st") || { bad "Case-$st :: setup FAILED in new_repo (diagnostic on stderr) — case NOT executed"; continue; }
  assert_hook_identity "$repo" "Case-$st"
  # apply_scanner_state registers its own FAIL directly (it is NOT called in a
  # command substitution), so this guard only aborts the case.
  apply_scanner_state "$repo" "$st" || continue
  run_canaries "$repo" "Case-$st"

  make_leak_fixture "$repo/leak.md"
  # Fixture sanity WITHOUT printing the value: assert it is key-shaped, so a
  # broken fixture can never make a row pass for the wrong reason.
  if [ "$(grep -cE 'AIza[0-9A-Za-z_-]{35}' "$repo/leak.md" 2>/dev/null)" = "1" ]; then
    ok "Case-$st :: fixture is key-shaped (value never printed)"
  else
    bad "Case-$st :: fixture is key-shaped (fixture broken -- the row would pass for the wrong reason)"
  fi

  git -C "$repo" add leak.md >/dev/null 2>&1
  out="$TMP/case_${st}.txt"
  rc=$(do_commit "$repo" "$out" "stage a key-shaped credential with the scanner $st")
  landed=$(leak_landed "$repo")

  if [ "$RED_MODE" = "1" ]; then
    assert_block "Case-$st on FROZEN pre-fix hook (RED)" "$frozen_v" "$rc"

    if [ "$landed" = "$frozen_leak" ]; then
      if [ "$frozen_leak" = "YES" ]; then
        ok "Case-$st :: RED reproduces the leak -- leak.md ACTUALLY LANDED in the commit"
      else
        ok "Case-$st :: RED did not land the leak (blocked accidentally, as measured)"
      fi
    else
      bad "Case-$st :: expected leak-landed=$frozen_leak, measured $landed"
    fi

    # The frozen artifact must NOT diagnose: no UNUSABLE banner, no reason line.
    if grep -qF 'The staged-content credential scanner is UNUSABLE' "$out" 2>/dev/null \
       || grep -qF "reason:  $reason" "$out" 2>/dev/null; then
      bad "Case-$st :: RED must NOT emit the fix's diagnostic (found it on the pre-fix artifact -- wrong artifact installed?)"
    else
      ok "Case-$st :: RED emits NO refusal diagnostic (the defect: the step evaporated / blocked undiagnosed)"
    fi

    # For the two rows that block accidentally, prove it was the exec failure.
    if [ -n "$exec_err" ]; then
      if grep -qF "$exec_err" "$out" 2>/dev/null; then
        ok "Case-$st :: RED blocked only ACCIDENTALLY -- exec error '$exec_err' misreported through the scan-failure banner"
      else
        bad "Case-$st :: RED blocked, but expected the accidental exec error '$exec_err' in the banner and it was absent"
      fi
    fi
  else
    assert_block "Case-$st on LIVE fixed hook (GREEN)" "$live_v" "$rc"

    if [ "$landed" = "NO" ]; then
      ok "Case-$st :: GREEN kept the credential out of history (leak.md did NOT land)"
    else
      bad "Case-$st :: GREEN kept the credential out of history (leak.md LANDED -- fail-open regression)"
    fi

    if grep -qF 'The staged-content credential scanner is UNUSABLE' "$out" 2>/dev/null; then
      ok "Case-$st :: GREEN emits the UNUSABLE refusal banner"
    else
      bad "Case-$st :: GREEN emits the UNUSABLE refusal banner (absent)"
    fi

    if grep -qF "reason:  $reason" "$out" 2>/dev/null; then
      ok "Case-$st :: GREEN names the precise state -- 'reason:  $reason'"
    else
      bad "Case-$st :: GREEN names the precise state -- expected 'reason:  $reason', not found"
    fi
  fi

  case_ran "$st"
done

# ---------------------------------------------------------------------------
# CASE 5 — IDENTICAL IN BOTH POLARITIES. Scanner present + executable, leak
# staged: step 4 works normally. This path is untouched by the fix, so it must
# BLOCK in both modes. Load-bearing in the no-regression direction: it proves
# the RED artifact's step 4 is otherwise INTACT, so cases 1-4's RED behaviour
# is attributable to the missing `else` specifically and not to a step 4 that
# was broken outright.
# ---------------------------------------------------------------------------
if repo=$(new_repo working); then
  assert_hook_identity "$repo" "Case-working-scanner"
  apply_scanner_state "$repo" present
  run_canaries "$repo" "Case-working-scanner"
  make_leak_fixture "$repo/leak.md"
  git -C "$repo" add leak.md >/dev/null 2>&1
  out="$TMP/case_working.txt"
  rc=$(do_commit "$repo" "$out" "leak staged, scanner fully usable")
  assert_block "Case-working-scanner blocks the leak (BOTH polarities: step 4 is otherwise intact)" BLOCK "$rc"
  if [ "$(leak_landed "$repo")" = "NO" ]; then
    ok "Case-working-scanner :: leak.md did NOT land (both polarities)"
  else
    bad "Case-working-scanner :: leak.md LANDED despite a usable scanner"
  fi
  if grep -qF 'key-shaped secret pattern' "$out" 2>/dev/null; then
    ok "Case-working-scanner :: refusal came from a REAL scan finding (both polarities)"
  else
    bad "Case-working-scanner :: expected a real scan-finding banner, not found"
  fi
  case_ran working
else
  bad "Case-working-scanner :: setup FAILED in new_repo (diagnostic on stderr) — case NOT executed"
fi

# ---------------------------------------------------------------------------
# CASE 6 — IDENTICAL IN BOTH POLARITIES: an ordinary, clean file with a usable
# scanner must be ALLOWED. Load-bearing against over-blocking: a "fix" that
# turned step 4 into a blanket refusal would satisfy cases 1-4 while making the
# repository uncommittable, and this case is what refuses to let that pass.
# ---------------------------------------------------------------------------
if repo=$(new_repo clean); then
  assert_hook_identity "$repo" "Case-clean-file"
  apply_scanner_state "$repo" present
  run_canaries "$repo" "Case-clean-file"
  printf 'ordinary prose. nothing credential shaped lives here.\n' > "$repo/notes.md"
  git -C "$repo" add notes.md >/dev/null 2>&1
  out="$TMP/case_clean.txt"
  rc=$(do_commit "$repo" "$out" "ordinary clean file")
  assert_block "Case-clean-file allowed (BOTH polarities: no over-blocking)" ALLOW "$rc"
  case_ran clean
else
  bad "Case-clean-file :: setup FAILED in new_repo (diagnostic on stderr) — case NOT executed"
fi

# ---------------------------------------------------------------------------
# CASE 7 — IDENTICAL IN BOTH POLARITIES: a submodule bump (a gitlink, mode
# 160000) with a usable scanner must be ALLOWED. A gitlink has no blob content
# to scan, so it is exactly the shape that a naive "refuse anything the scanner
# cannot read" implementation would wrongly reject — which would block every
# submodule-pointer commit in a repo built on submodules.
# ---------------------------------------------------------------------------
if repo=$(new_repo gitlink); then
  assert_hook_identity "$repo" "Case-gitlink"
  apply_scanner_state "$repo" present
  run_canaries "$repo" "Case-gitlink"
  sha_a=$(git -C "$repo" rev-parse HEAD)
  git -C "$repo" update-index --add --cacheinfo 160000,"$sha_a",submodules/dep >/dev/null 2>&1
  out="$TMP/case_gitlink_add.txt"
  rc=$(do_commit "$repo" "$out" "add submodule gitlink")
  assert_block "Case-gitlink initial add allowed (BOTH polarities)" ALLOW "$rc"
  sha_b=$(git -C "$repo" rev-parse HEAD)
  git -C "$repo" update-index --cacheinfo 160000,"$sha_b",submodules/dep >/dev/null 2>&1
  out="$TMP/case_gitlink_bump.txt"
  rc=$(do_commit "$repo" "$out" "bump submodule gitlink")
  assert_block "Case-gitlink bump allowed (BOTH polarities: pointer-only commit, no blob to scan)" ALLOW "$rc"
  case_ran gitlink
else
  bad "Case-gitlink :: setup FAILED in new_repo (diagnostic on stderr) — case NOT executed"
fi

# ---------------------------------------------------------------------------
# CASE 8 — IDENTICAL IN BOTH POLARITIES: nothing staged at all must be ALLOWED,
# proving the new refusal can NEVER block an empty commit (the hook `exit 0`s
# at the staged-empty guard, long before step 4).
#
# Deliberately run with the scanner DELETED — the state that otherwise refuses
# — so rc=0 can only mean the empty-index guard short-circuited ahead of step 4.
# `--allow-empty` is used on purpose: with a plain `git commit` on an empty
# index, GIT ITSELF exits 1 ("Changes not staged for commit") regardless of any
# hook, so the row would fail for a reason having nothing to do with this fix.
#
# LIVENESS (round-24, F-F — see the corrected CASE 8 block in the header):
# this row's hook output is 0 bytes, which on its own is indistinguishable from
# a hook that never ran. It therefore carries a genuine IN-ROW witness: the
# hook's own `ATMO_PRECOMMIT_RAN` marker, written at the top of the hook body
# (line 54) — ~380 lines AHEAD of the staged-empty `exit 0` (line 438), and
# present in the frozen artifact at the same position, so it works in BOTH
# polarities. Cleared immediately before the attempt so a marker left by the
# adjacent probe cannot fake it; asserted present afterwards. The adjacent
# step-2 / step-3b probe is retained on top of that, proving those steps are
# reachable in this exact repo and scanner state.
# ---------------------------------------------------------------------------
if repo=$(new_repo empty); then
  assert_hook_identity "$repo" "Case-empty-index"
  apply_scanner_state "$repo" deleted
  run_canaries "$repo" "Case-empty-index"
  if git -C "$repo" diff --cached --quiet 2>/dev/null; then
    ok "Case-empty-index :: index is genuinely empty before the attempt"
  else
    bad "Case-empty-index :: index is genuinely empty before the attempt (leftover staged state would invalidate this row)"
  fi

  # IN-ROW LIVENESS CANARY (see the CASE 8 note in the header). Clear the
  # hook's own run-marker first, so a marker left by the adjacent canary probe
  # can never be mistaken for evidence that THIS attempt ran the hook.
  RAN_MARKER="$repo/.git/ATMO_PRECOMMIT_RAN"
  rm -f "$RAN_MARKER"
  if [ ! -e "$RAN_MARKER" ]; then
    ok "Case-empty-index :: run-marker cleared before the attempt (a stale marker cannot fake the canary below)"
  else
    bad "Case-empty-index :: run-marker could NOT be cleared ($RAN_MARKER) -- the liveness canary below would read a stale marker and prove nothing"
  fi

  out="$TMP/case_empty.txt"
  rc=$(do_commit "$repo" "$out" "empty commit, scanner deleted" --allow-empty)
  assert_block "Case-empty-index allowed with the scanner DELETED (BOTH polarities: the refusal cannot block an empty commit)" ALLOW "$rc"
  if [ ! -s "$out" ]; then
    ok "Case-empty-index :: hook emitted NOTHING (exited at the staged-empty guard, ahead of step 4)"
  else
    bad "Case-empty-index :: hook emitted output on an empty index (expected an early, silent exit 0)"
  fi
  # The marker is written at the TOP of the hook body, ahead of the
  # staged-empty guard, so its presence proves the hook RAN for this exact
  # attempt -- turning "emitted 0 bytes" from an ambiguous silence into
  # "ran, and deliberately said nothing".
  if [ -e "$RAN_MARKER" ]; then
    ok "Case-empty-index :: run-marker present after the attempt -- the hook BODY executed and chose to exit silently (0 bytes is a deliberate early exit, NOT an inert hook)"
  else
    bad "Case-empty-index :: run-marker ABSENT after the attempt -- the hook never ran, so the silent exit 0 proves nothing about the staged-empty guard"
  fi
  case_ran empty
else
  bad "Case-empty-index :: setup FAILED in new_repo (diagnostic on stderr) — case NOT executed"
fi

# ---------------------------------------------------------------------------
# LIVE-FILE INTEGRITY (§11.4.84 quiescence): this test is mirror-only. The live
# hook MUST be byte-identical to what was read at start. The live scanner is
# reported honestly rather than enforced: a sibling stream owns
# scripts/secret_scan.sh and may legitimately be editing it concurrently. This
# script never writes to either file.
# ---------------------------------------------------------------------------
echo
END_LIVE_HOOK_SHA=$(sha16 "$LIVE_HOOK")
END_SCANNER_SHA=$(sha16 "$LIVE_SCANNER")
echo "--- live-file integrity ---"
echo "    live hook    start=$START_LIVE_HOOK_SHA end=$END_LIVE_HOOK_SHA"
echo "    live scanner start=$START_SCANNER_SHA end=$END_SCANNER_SHA"

if [ "$END_LIVE_HOOK_SHA" = "$START_LIVE_HOOK_SHA" ]; then
  ok "LiveHook-unchanged-across-this-run ($END_LIVE_HOOK_SHA)"
else
  bad "LiveHook-unchanged-across-this-run (start=$START_LIVE_HOOK_SHA end=$END_LIVE_HOOK_SHA)"
fi

# F-D: this conditional used to call `ok` on BOTH branches — an "assertion"
# that incremented PASS no matter what it observed and therefore could not
# fail. Only the equal branch states a checkable property; the other branch has
# nothing it can honestly assert (this script cannot prove a negative about a
# file a sibling stream owns), so it is now a SKIP-with-reason: it prints, it
# is counted by the completion contract, and it does NOT inflate PASS.
if [ "$END_SCANNER_SHA" = "$START_SCANNER_SHA" ]; then
  ok "LiveScanner-unchanged-across-this-run ($END_SCANNER_SHA)"
else
  # NOT a failure: this script only ever READS the scanner (see new_repo's
  # `cp`), and a sibling stream legitimately owns scripts/secret_scan.sh.
  # It is NOT a pass either — a mid-run change means different cases copied
  # different scanner bytes, so this run's cross-case comparability (§11.4.50)
  # is not established and there is no honest verdict to give.
  echo "NOTE  LiveScanner sha changed during this run (start=$START_SCANNER_SHA end=$END_SCANNER_SHA) -- attributed to the sibling stream that owns scripts/secret_scan.sh, NOT to this test, which only ever reads it."
  skip "LiveScanner-unchanged-across-this-run (concurrent sibling-stream edit; cases may have copied differing scanner bytes -- re-run when the tree is quiescent)"
fi

# ---------------------------------------------------------------------------
# COMPLETION CONTRACT (see EXPECTED_ASSERTIONS_* above for the derivation).
#
# Everything above this point asserts things about the HOOK. These two assert
# things about THIS RUN: that every case executed, and that the sweep is the
# size it is supposed to be. Without them, a case that aborted at setup left no
# trace at all — the remaining cases still passed, PASS was simply smaller, and
# nothing compared it to anything, so a shrunken sweep and a complete sweep
# printed the same reassuring `FAIL=0`.
#
# They are computed BEFORE they emit, against a constant that excludes them
# (§11.4.6: an expectation that counted its own checks would be partly
# self-fulfilling).
# ---------------------------------------------------------------------------
echo
echo "--- completion contract ---"
_observed=$((PASS + FAIL + SKIP))
_missing=""
for _want in "${EXPECTED_CASES[@]}"; do
  _seen=0
  for _got in ${CASES_RAN[@]+"${CASES_RAN[@]}"}; do
    [ "$_got" = "$_want" ] && { _seen=1; break; }
  done
  [ "$_seen" -eq 1 ] || _missing="$_missing $_want"
done
echo "    cases expected=${#EXPECTED_CASES[@]} ran=${#CASES_RAN[@]}  assertions expected=$EXPECTED_ASSERTIONS observed=$_observed (PASS=$PASS FAIL=$FAIL SKIP=$SKIP)"

if [ -z "$_missing" ] && [ "${#CASES_RAN[@]}" -eq "${#EXPECTED_CASES[@]}" ]; then
  ok "Completion-every-case-ran (all ${#EXPECTED_CASES[@]} cases reached the end of their body)"
else
  bad "Completion-every-case-ran (MISSING:${_missing:- none-named}; ran=${#CASES_RAN[@]} of ${#EXPECTED_CASES[@]}) -- a case was dropped, so this run measured LESS than it claims"
fi

if [ "$_observed" -eq "$EXPECTED_ASSERTIONS" ]; then
  ok "Completion-assertion-count ($_observed == $EXPECTED_ASSERTIONS expected for RED_MODE=$RED_MODE)"
else
  bad "Completion-assertion-count (observed $_observed, expected $EXPECTED_ASSERTIONS for RED_MODE=$RED_MODE) -- the sweep is the WRONG SIZE; assertions were skipped, dropped, or added without updating EXPECTED_ASSERTIONS_*"
fi

echo
echo "=== HXC-282/F1 summary (RED_MODE=$RED_MODE, $ACTIVE_LABEL): PASS=$PASS FAIL=$FAIL SKIP=$SKIP ==="
if [ "$FAIL" -ne 0 ]; then
  printf 'FAILED CASES:\n'
  for c in "${FAILED_CASES[@]}"; do printf '  - %s\n' "$c"; done
  printf 'Results: %d passed, %d failed\n' "$PASS" "$FAIL"
  exit 1
fi
printf 'Results: %d passed, %d failed\n' "$PASS" "$FAIL"
exit 0
