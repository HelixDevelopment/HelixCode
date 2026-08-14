#!/usr/bin/env bash
# scripts/git_hooks/test_mutation_residue_evidence_exempt.sh
# HXC-223 — §11.4.115 RED-first polarity test for the §11.4.84
# mutation-residue scan's CAPTURED-EVIDENCE exemption path.
#
# THE DEFECT (RED_MODE=1 reproduces it on the pre-fix hook):
#   The pre-commit hook's §11.4.84 residue scan blocks any staged file
#   carrying `MUTATED for paired` (et al). Its only exemption requires
#   BOTH (a) the opt-in marker `11.4.84-mutation-test-exempt` AND (b) a
#   PROVEN restore idiom (`trap <target> EXIT` whose target's own body
#   restores via `cp ... backup ...` / `git checkout --`).
#   A captured-evidence artifact (`docs/qa/**/*.log`) can satisfy (a) but
#   can NEVER satisfy (b): it is captured OUTPUT — no trap, no restore,
#   nothing to execute. So §1.1, which REQUIRES the mutation proof be
#   captured, and §11.4.84, which refuses to let it be committed, stand in
#   direct contradiction as implemented.
#
# POLARITY SWITCH (§11.4.115): ONE source, TWO roles.
#   RED_MODE=0 (DEFAULT) — the standing GREEN regression guard: the real
#     evidence log MUST be ALLOWED (defect ABSENT).
#   RED_MODE=1 — reproduce-and-assert-defect-PRESENT on a PRE-FIX hook: that
#     same evidence log MUST be BLOCKED.
#   Every OTHER case below asserts IDENTICALLY in both modes — they are the
#   no-regression quadrants, and a fix that widens the exemption into a
#   blanket bypass FAILs them in either polarity.
#
# ANTI-BLUFF: runs real `git add` / `git commit` against the real hook in a
# throwaway repo (mktemp -d) — never a re-grep of the hook source. The
# evidence-log fixture is the BYTE-FOR-BYTE content of the actually-blocked
# file, not a synthetic stand-in.
#
# ANTI-BLUFF, SECOND AXIS (round-24, see the two blocks below): a case PASSes
# only when the §11.4.84 RESIDUE DETECTOR ITSELF is demonstrably what produced
# the outcome — the residue banner naming that exact path (BLOCK cases) or the
# detector's own audit NOTICE for that exact path (ALLOW cases). A bare
# exit-code assertion is not enough: this hook has six other blocking gates, so
# `rc != 0` can be produced by something that never looked at residue at all.
#
# Usage:   scripts/git_hooks/test_mutation_residue_evidence_exempt.sh
#          RED_MODE=1 scripts/git_hooks/test_mutation_residue_evidence_exempt.sh
#          HOOK_UNDER_TEST=/path/to/mutant-pre-commit  <this script>
# Exit:    0 iff every case passed; 1 otherwise.
# Side-effects: creates + removes a temp dir; touches NO real repo state,
#   installs NO hooks into the real .git/hooks, never uses --no-verify.
# Dependencies: git, bash, mktemp.
# Cross-references: §11.4.84 / §11.4.115 / §11.4.135 / §1.1 / HXC-223;
#   scripts/git_hooks/pre-commit; scripts/git_hooks/test_hooks.sh.

set -uo pipefail

# ROUND-9 FIX (HXC-282, 2026-08-13): default flipped 1 -> 0.
#
# WHY: at its former `:-1` default this test was a PERMANENT RED. Unlike the
# sibling deletion-detection test, it installs the LIVE hook in BOTH polarities
# (see the `cp "$HOOKS_SRC_DIR/pre-commit"` below) rather than a frozen pre-fix
# fixture, so once HXC-223 landed the fix, the RED assertion could never pass
# again: the live hook now correctly ALLOWS the evidence log, which is exactly
# what RED_MODE=1 asserts must NOT happen. Measured immediately before this
# change, on the live hook:
#     (default, RED_MODE=1)  exit 1 — PASS=8 FAIL=1
#                            Q1-evidence-log-with-marker: expected BLOCK, hook ALLOWED
#     (RED_MODE=0)           exit 0 — PASS=10 FAIL=0
# A test that is red by construction trains readers to ignore its red, which
# is the failure mode a standing guard exists to avoid.
#
# NO RED EVIDENCE IS LOST BY THIS FLIP (§11.4.115):
#   * the RED path stays explicitly reachable — `RED_MODE=1 <this script>`;
#   * it already could not reproduce against the live hook, so nothing that
#     used to work stops working;
#   * the original RED baseline is durably CAPTURED, not merely asserted:
#     docs/qa/hxc223_.../evidence.md lines 68 + 71 record all four quadrants,
#     including `1_red_baseline_prefix_hook.log` (RED_MODE=1 against the
#     PRE-FIX hook -> exit 0, 9/9 PASS, defect reproduced on the real
#     artifact) and `4_red_now_fails_postfix.log` (RED_MODE=1 against the
#     POST-FIX hook -> exit 1, polarity proven real).
#
# Also aligns with this repo's convention: 12 sibling gates/tests default to
# `:-0`, and the only other `:-1` is a separate item.
#
# HONEST BOUNDARY (§11.4.6): this makes the DEFAULT run meaningful again.
#
# ROUND-24 UPDATE — the sentence that used to stand here ("does NOT restore RED
# re-executability") is now WRONG and is retracted rather than left standing.
# The HOOK_UNDER_TEST override added below supplies exactly the missing
# mechanism, and RED was re-executed with it. Measured, all four quadrants,
# 2026-08-13, and REPRODUCED IDENTICALLY across two successive revisions of the
# working-tree hook while a concurrent stream was editing it
# (b25427107f4abf66… then f0e7b0150034e3b2…), so the numbers below are a
# property of this harness, not of one momentary hook snapshot:
#   RED_MODE=1 + pre-fix-SHAPED hook  -> exit 0, PASS=9  FAIL=0   defect reproduced
#   RED_MODE=1 + live (fixed) hook    -> exit 1, PASS=8  FAIL=1   polarity is real
#   RED_MODE=0 + live (fixed) hook    -> exit 0, PASS=10 FAIL=0   standing guard
#   RED_MODE=0 + neutered-detector    -> exit 1, PASS=0  FAIL=10  detector load-bearing
# (the pre-fix SHAPE was produced by disabling `is_captured_evidence_exempt` in
# a throwaway MIRROR of the hook — never in the live one.)
#
# What is still genuinely OUTSTANDING is narrower than the retracted claim: a
# COMMITTED frozen pre-fix hook snapshot (the pattern the deletion-detection
# test uses via scripts/git_hooks/testdata/hxc282_pre_fix_pre_commit.sh) so the
# RED polarity runs from tracked bytes as a STANDING gate rather than from an
# artifact an operator has to construct. That fixture is a SEPARATE tracked item
# (testdata/ is another stream's scope this round) and is NOT built here.
RED_MODE="${RED_MODE:-0}"
HOOKS_SRC_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$HOOKS_SRC_DIR/../.." && pwd)

# ROUND-24 FIX A (HXC-282, 2026-08-13) — §11.4.120 RECONCILIATION, NOT a gate
# weakening. Measured immediately before this change, on the live hook:
#     live hook   -> exit 1, PASS=8 FAIL=2  (Q1 and Q3, both "expected ALLOW,
#                    hook BLOCKED rc=1")
#     HEAD hook   -> exit 0, PASS=10 FAIL=0
# The delta is NOT harness state leakage and NOT a regression in the hook. It
# is a CORRECT new refusal that this harness never provisioned for:
#
#   * the hook's step 4 (§11.4.135/§11.4.138) resolves its scanner as
#         SECRET_SCANNER="$REPO_ROOT/scripts/secret_scan.sh"
#     with `REPO_ROOT=$(git rev-parse --show-toplevel …)` (pre-commit:48,1398),
#     i.e. INSIDE whatever repo the hook is running in — here, the throwaway
#     one this script creates;
#   * this harness provisioned ONLY the hook into that throwaway repo, so the
#     scanner was ABSENT there and had been for its whole life;
#   * step 4 used to fail OPEN on an absent scanner (silently skipping the
#     credential check). The F1 fix made it REFUSE instead — an unrun check and
#     a failed check are both refusals (§11.4.201/§11.4.236).
#
# So the hook now blocks EVERY commit in the throwaway repo, including the two
# the harness expects to be ALLOWED. The fix under §11.4.120 is to reconcile
# the STALE HARNESS (provision the scanner it never provisioned), never to
# fake-pass or weaken the correct new refusal.
#
# NOTE the second-order hazard this closes: while the scanner was absent, the
# BLOCK-expecting cases were passing for the WRONG REASON — the absent-scanner
# refusal, not residue detection. That is exactly the tautology FIX B removes.
SECRET_SCANNER_SRC="${SECRET_SCANNER_SRC:-$REPO_ROOT/scripts/secret_scan.sh}"

# ROUND-24 FIX B (HXC-282, 2026-08-13) — the hook artifact under test is
# overridable so the §1.1 PAIRED MUTATION for this harness is runnable WITHOUT
# copying this script somewhere else and WITHOUT ever mutating the live hook:
#     cp scripts/git_hooks/pre-commit "$T/pre-commit"   # mirror
#     # neuter step 3's residue-marker regex in the MIRROR only
#     HOOK_UNDER_TEST="$T/pre-commit" <this script>     # MUST now go red
# Defaults to the live hook, so an ordinary run is unchanged. (Same pattern the
# sibling deletion-detection test uses for its frozen pre-fix fixture.)
HOOK_UNDER_TEST="${HOOK_UNDER_TEST:-$HOOKS_SRC_DIR/pre-commit}"

# The REAL file the hook currently refuses (§11.4.115: reproduce on the
# actual artifact, never a synthetic failure the fix is written to agree
# with). Falls back to an inline equivalent only if it has already been
# committed and moved.
REAL_EVIDENCE="$REPO_ROOT/docs/qa/hxc220_module_identity_gate_20260805T120714Z/3_mutation_proof.log"

PASS=0
FAIL=0
declare -a FAILED_CASES=()

ok()  { PASS=$((PASS+1)); printf 'PASS  %s\n' "$1"; }
bad() { FAIL=$((FAIL+1)); FAILED_CASES+=("$1"); printf 'FAIL  %s\n' "$1"; }

# --- ATTRIBUTED ASSERTIONS (round-24 FIX B) --------------------------------
# The asserter these replace tested ONLY the commit's exit code. That made 7 of
# its 8 passing cases TAUTOLOGICAL — measured: with step 3's residue-marker
# regex neutered in a mirror hook, 7 of 8 kept passing, because this hook has
# six OTHER blocking gates and any of them satisfies a bare `rc != 0`. A check
# that still passes when the feature under test is deleted certifies nothing
# (§11.4 / §1.1).
#
# Both asserters below therefore demand evidence ONLY the residue detector can
# emit, naming the exact staged path:
#   BLOCK -> the §11.4.84 residue banner AND that path among the listed hits
#            (pre-commit step 3 formats each hit as a line of exactly "  <path>")
#   ALLOW -> that path's §11.4.84 audit NOTICE, which is emitted ONLY from
#            inside the marker-matched branch — so an allow additionally proves
#            the detector INSPECTED the file and deliberately exempted it,
#            rather than never having looked (a dead detector also "allows").
RESIDUE_BANNER='BLOCKED by pre-commit hook (§11.4.84 mutation residue)'

# assert_blocked_by_residue <case-name> <actual-exit> <staged-path>
assert_blocked_by_residue() {
  local name="$1" rc="$2" path="$3"
  if [ "$rc" -eq 0 ]; then
    bad "$name (expected BLOCK, hook ALLOWED)"; return
  fi
  if ! grep -qF -- "$RESIDUE_BANNER" "$HK_OUT" 2>/dev/null; then
    bad "$name (refused, but NOT by the §11.4.84 residue scan — no residue banner in hook output; another gate refused it, so this case proves nothing about residue detection)"; return
  fi
  if ! grep -qxF -- "  $path" "$HK_OUT" 2>/dev/null; then
    bad "$name (residue banner present, but '$path' is NOT among the listed hits — the block came from a different staged file)"; return
  fi
  ok "$name (blocked BY THE RESIDUE SCAN, which named this exact path)"
}

# assert_allowed_with_notice <case-name> <actual-exit> <expected-NOTICE-substring>
assert_allowed_with_notice() {
  local name="$1" rc="$2" notice="$3"
  if [ "$rc" -ne 0 ]; then
    bad "$name (expected ALLOW, hook BLOCKED rc=$rc)"; return
  fi
  if ! grep -qF -- "$notice" "$HK_OUT" 2>/dev/null; then
    bad "$name (allowed, but the §11.4.84 audit NOTICE proving the detector inspected and EXEMPTED it is absent — indistinguishable from a detector that never ran)"; return
  fi
  ok "$name (allowed, and the detector's own audit NOTICE proves it inspected + exempted this path)"
}

TMP=$(mktemp -d 2>/dev/null) || { echo "mktemp failed"; exit 1; }
cleanup() { rm -rf "$TMP" 2>/dev/null || true; }
trap cleanup EXIT

cd "$TMP" || exit 1
git init -q
git config user.email "test@example.com"
git config user.name "HXC-223 Test"
git config commit.gpgsign false

mkdir -p .git/hooks
cp "$HOOK_UNDER_TEST" .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

# --- ROUND-24 FIX A: provision the hook's OTHER instrument ------------------
# The hook resolves its credential scanner INSIDE this throwaway repo
# ("$REPO_ROOT/scripts/secret_scan.sh", pre-commit:1398, REPO_ROOT from
# `git rev-parse --show-toplevel`), and now correctly REFUSES when it is
# unusable. So the throwaway repo needs a real, executable copy at exactly that
# path, or every case below blocks for a reason that has nothing to do with
# §11.4.84. A COPY is used deliberately: never a symlink into the live tree,
# and the live scanner is never modified or chmod-ed.
# The copy is left UNTRACKED here — the hook only stat/execs it, and staging it
# would change what each case's `--staged` scan sees.
mkdir -p scripts
if [ ! -f "$SECRET_SCANNER_SRC" ] || [ ! -r "$SECRET_SCANNER_SRC" ]; then
  echo "HARNESS ABORT (§11.4.201 invalid instrument): cannot provision the hook's" >&2
  echo "credential scanner — source missing or unreadable: $SECRET_SCANNER_SRC" >&2
  echo "Without it the hook refuses EVERY commit here, so this run would report" >&2
  echo "failures about §11.4.84 that are really about a missing scanner." >&2
  exit 2
fi
cp "$SECRET_SCANNER_SRC" scripts/secret_scan.sh
chmod +x scripts/secret_scan.sh
# Re-assert the hook's OWN usability triple (-f -r -x) on the provisioned copy,
# rather than assuming the cp+chmod took (§11.4.6).
if [ ! -f scripts/secret_scan.sh ] || [ ! -r scripts/secret_scan.sh ] || [ ! -x scripts/secret_scan.sh ]; then
  echo "HARNESS ABORT (§11.4.201 invalid instrument): provisioned scanner copy is" >&2
  echo "not a readable+executable regular file at $TMP/scripts/secret_scan.sh" >&2
  exit 2
fi
# Stat-able is not the same as WORKING. Run it once on this repo's (empty)
# index and require a clean rc=0 — measured behaviour on a fresh repo, with or
# without a HEAD commit, is `rc=0, "OK: no unallowlisted key-shaped secret
# pattern found (mode=staged)"`. If the scanner is mid-edit or broken, the hook
# would refuse every commit below and this harness would report a cascade of
# §11.4.84 failures that are really about a broken scanner. Abort honestly
# instead (exit 2 — distinct from 1 = genuine case failure).
scanner_probe_out=$(./scripts/secret_scan.sh --staged 2>&1)
scanner_probe_rc=$?
if [ "$scanner_probe_rc" -ne 0 ]; then
  echo "HARNESS ABORT (§11.4.201 invalid instrument): the provisioned credential" >&2
  echo "scanner exits $scanner_probe_rc on an EMPTY index, so it would block every" >&2
  echo "commit below for reasons unrelated to §11.4.84. Source: $SECRET_SCANNER_SRC" >&2
  printf '%s\n' "$scanner_probe_out" >&2
  exit 2
fi

echo "seed" > README.txt
git add README.txt
git commit -q -m "seed" 2>/dev/null

HK_OUT="$TMP/hook_output.txt"
try_commit() {
  git commit -q -m "$1" > "$HK_OUT" 2>&1
  echo $?
}
# Abandon a staged attempt without ever using --no-verify.
unstage() { git reset -q HEAD -- "$@" 2>/dev/null || true; rm -f "$@" 2>/dev/null || true; }

echo "=== HXC-223 §11.4.84 captured-evidence exemption — RED_MODE=$RED_MODE ==="
# Provenance, printed so a run against a MUTANT mirror can never be mistaken
# for a run against the live hook when someone reads the log later (§11.4.6).
echo "hook under test : $HOOK_UNDER_TEST"
echo "                  sha256 $(sha256sum "$HOOK_UNDER_TEST" 2>/dev/null | awk '{print $1}')"
echo "scanner source  : $SECRET_SCANNER_SRC"
echo

# ---------------------------------------------------------------------------
# QUADRANT 1 (POLARITY-SWITCHED): captured-evidence log carrying a residue
# marker. RED_MODE=1 -> the defect: BLOCKED. RED_MODE=0 -> fixed: ALLOWED.
# Fixture is the byte-for-byte real blocked artifact.
# ---------------------------------------------------------------------------
EV_DIR="docs/qa/hxc220_module_identity_gate_20260805T120714Z"
mkdir -p "$EV_DIR"
if [ -r "$REAL_EVIDENCE" ]; then
  cp "$REAL_EVIDENCE" "$EV_DIR/3_mutation_proof.log"
  EV_SRC="real-blocked-artifact"
else
  # Equivalent captured-output shape (already-committed fallback).
  cat > "$EV_DIR/3_mutation_proof.log" <<'EVLOG'
$ git diff scripts/gates/hxc199_module_identity_exact_match_gate.sh
+  # MUTATED for paired §1.1 mutation test (HXC-220) — reverts to the exact
+  # pre-fix comparison so the gate must FAIL.
EVLOG
  EV_SRC="inline-fallback"
fi
chmod 644 "$EV_DIR/3_mutation_proof.log"
git add -- "$EV_DIR/3_mutation_proof.log"
EV_PATH="$EV_DIR/3_mutation_proof.log"
rc=$(try_commit 'docs(qa): capture §1.1 mutation proof')
if [ "$RED_MODE" = "1" ]; then
  # RED is now ATTRIBUTED too: reproducing the defect means the RESIDUE SCAN
  # itself listed the evidence log as residue — not merely that some gate
  # refused the commit. Under the old exit-code-only RED, an unrelated refusal
  # (e.g. the absent-scanner block above) would have read as a successful
  # defect reproduction, which is a false RED.
  assert_blocked_by_residue "Q1-evidence-log-with-marker [$EV_SRC] (RED: defect present)" "$rc" "$EV_PATH"
else
  assert_allowed_with_notice "Q1-evidence-log-with-marker [$EV_SRC] (GREEN: defect absent)" "$rc" \
    "NOTICE (§11.4.84 audit): captured-evidence exemption granted for staged file: $EV_PATH"
  # Q1b is NOT a duplicate of Q1's NOTICE check: Q1 asserts that a grant was
  # announced, Q1b asserts the grant is EXPLAINED — the audit trail states the
  # structural reason it is inert, so a reviewer reading hook output can judge
  # the exemption instead of taking it on faith.
  if grep -qF -- "reason: non-executable captured-output artifact under docs/qa/" "$HK_OUT" 2>/dev/null; then
    ok "Q1b-evidence-exemption-audit-notice-states-its-reason"
  else
    bad "Q1b-evidence-exemption-audit-notice-states-its-reason (grant not explained in hook output)"
  fi
fi
unstage "$EV_DIR/3_mutation_proof.log"

# ---------------------------------------------------------------------------
# QUADRANT 2: LIVE SOURCE carrying a residue marker -> BLOCKED in BOTH modes.
# This is the §11.4.84 forensic case itself (an `// always pass` bypass swept
# into an unrelated commit). If a fix regresses this, it is a blanket bypass.
# ---------------------------------------------------------------------------
mkdir -p scripts helix_code/internal/auth
cat > scripts/live_gate.sh <<'LIVE'
#!/usr/bin/env bash
verify_token() { return 0; } # MUTATED for paired §1.1 mutation test
LIVE
git add -- scripts/live_gate.sh
assert_blocked_by_residue "Q2a-live-source-scripts-with-marker" "$(try_commit 'scripts residue')" scripts/live_gate.sh
unstage scripts/live_gate.sh

cat > helix_code/internal/auth/jwt.go <<'LIVEGO'
package auth

func Verify(t string) bool { return true } // always pass
LIVEGO
git add -- helix_code/internal/auth/jwt.go
assert_blocked_by_residue "Q2b-live-source-helix_code-with-marker" "$(try_commit 'go residue')" helix_code/internal/auth/jwt.go
unstage helix_code/internal/auth/jwt.go

# ---------------------------------------------------------------------------
# QUADRANT 3: mutation-TEST script satisfying (a) opt-in + (b) proven restore
# idiom -> still ALLOWED in BOTH modes (pre-existing exemption preserved).
# ---------------------------------------------------------------------------
cat > mutation_test_fixture.sh <<'FIXTURE'
#!/usr/bin/env bash
# §11.4.84-mutation-test-exempt: this file's markers are trap-restored test logic
restore() { cp "$TARGET.backup" "$TARGET"; }
trap restore EXIT
cp "$TARGET" "$TARGET.backup"
# Replace the pattern (MUTATED for paired §1.1 mutation test — restored above).
sed -i 's/return false/return true/' "$TARGET"
FIXTURE
git add -- mutation_test_fixture.sh
assert_allowed_with_notice "Q3-mutation-test-script-with-opt-in-and-restore-idiom" "$(try_commit 'exempt mutation test fixture')" \
  "NOTICE (§11.4.84 audit): mutation-test exemption granted for staged file: mutation_test_fixture.sh"

# ---------------------------------------------------------------------------
# QUADRANT 4: claims (a) opt-in but has NO restore idiom (b), and is NOT under
# an evidence dir -> still BLOCKED in BOTH modes.
# ---------------------------------------------------------------------------
cat > fake_exempt.sh <<'FAKE'
#!/usr/bin/env bash
# §11.4.84-mutation-test-exempt: claims the opt-in but never restores anything
sed -i 's/return false/return true/' "$TARGET"  # MUTATED for paired §1.1 mutation test
FAKE
git add -- fake_exempt.sh
assert_blocked_by_residue "Q4-opt-in-without-restore-idiom-outside-evidence-dir" "$(try_commit 'fake exempt')" fake_exempt.sh
unstage fake_exempt.sh

# ---------------------------------------------------------------------------
# NARROWNESS CASES — prove the evidence path is not "anything under docs/qa".
# docs/qa/** genuinely DOES contain tracked executables (harness run_proof.sh,
# mode 100755), so LOCATION ALONE would be an unsound discriminator.
# ---------------------------------------------------------------------------
# N1: executable harness .sh under docs/qa carrying a marker -> BLOCKED.
mkdir -p docs/qa/some_run/harness
cat > docs/qa/some_run/harness/run_proof.sh <<'HARNESS'
#!/usr/bin/env bash
echo "running"  # MUTATED for paired §1.1 mutation test
HARNESS
chmod 755 docs/qa/some_run/harness/run_proof.sh
git add -- docs/qa/some_run/harness/run_proof.sh
assert_blocked_by_residue "N1-executable-sh-under-docs-qa-with-marker" "$(try_commit 'qa harness residue')" docs/qa/some_run/harness/run_proof.sh
unstage docs/qa/some_run/harness/run_proof.sh

# N2: a .log carrying a marker OUTSIDE the evidence tree -> BLOCKED.
mkdir -p scripts/logs
printf 'x # MUTATED for paired §1.1 mutation test\n' > scripts/logs/out.log
git add -- scripts/logs/out.log
assert_blocked_by_residue "N2-log-with-marker-outside-evidence-tree" "$(try_commit 'stray log residue')" scripts/logs/out.log
unstage scripts/logs/out.log

# N3: a .sh under docs/qa staged NON-executable (mode 100644) carrying a
# marker -> BLOCKED. Mode alone must not be the whole test: an executable
# EXTENSION is still code regardless of its staged bit.
mkdir -p docs/qa/some_run2
printf '#!/usr/bin/env bash\necho hi # MUTATED for paired §1.1 mutation test\n' > docs/qa/some_run2/helper.sh
chmod 644 docs/qa/some_run2/helper.sh
git add -- docs/qa/some_run2/helper.sh
assert_blocked_by_residue "N3-nonexec-mode-but-sh-extension-under-docs-qa" "$(try_commit 'qa sh 644 residue')" docs/qa/some_run2/helper.sh
unstage docs/qa/some_run2/helper.sh

# N4: an evidence-class .log under docs/qa staged with the EXECUTABLE bit set
# -> BLOCKED. Extension alone must not be the whole test either: a staged
# executable is code the moment it lands, whatever it is named.
mkdir -p docs/qa/some_run3
printf 'output # MUTATED for paired §1.1 mutation test\n' > docs/qa/some_run3/exec_evidence.log
chmod 755 docs/qa/some_run3/exec_evidence.log
git add -- docs/qa/some_run3/exec_evidence.log
assert_blocked_by_residue "N4-executable-mode-log-under-docs-qa" "$(try_commit 'qa exec log residue')" docs/qa/some_run3/exec_evidence.log
unstage docs/qa/some_run3/exec_evidence.log

echo
echo "=== HXC-223 summary (RED_MODE=$RED_MODE): PASS=$PASS FAIL=$FAIL ==="
if [ "$FAIL" -ne 0 ]; then
  printf 'FAILED CASES:\n'
  for c in "${FAILED_CASES[@]}"; do printf '  - %s\n' "$c"; done
  exit 1
fi
exit 0
