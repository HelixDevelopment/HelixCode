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
#   RED_MODE=1 (default) — reproduce-and-assert-defect-PRESENT on the
#     CURRENT (pre-fix) hook: the real blocked evidence log MUST be BLOCKED.
#   RED_MODE=0 — the standing GREEN regression guard: that same evidence
#     log MUST be ALLOWED (defect ABSENT).
#   Every OTHER case below asserts IDENTICALLY in both modes — they are the
#   no-regression quadrants, and a fix that widens the exemption into a
#   blanket bypass FAILs them in either polarity.
#
# ANTI-BLUFF: runs real `git add` / `git commit` against the real hook in a
# throwaway repo (mktemp -d) — never a re-grep of the hook source. The
# evidence-log fixture is the BYTE-FOR-BYTE content of the actually-blocked
# file, not a synthetic stand-in.
#
# Usage:   scripts/git_hooks/test_mutation_residue_evidence_exempt.sh
#          RED_MODE=0 scripts/git_hooks/test_mutation_residue_evidence_exempt.sh
# Exit:    0 iff every case passed; 1 otherwise.
# Side-effects: creates + removes a temp dir; touches NO real repo state,
#   installs NO hooks into the real .git/hooks, never uses --no-verify.
# Dependencies: git, bash, mktemp.
# Cross-references: §11.4.84 / §11.4.115 / §11.4.135 / §1.1 / HXC-223;
#   scripts/git_hooks/pre-commit; scripts/git_hooks/test_hooks.sh.

set -uo pipefail

RED_MODE="${RED_MODE:-1}"
HOOKS_SRC_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$HOOKS_SRC_DIR/../.." && pwd)

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
git config user.name "HXC-223 Test"
git config commit.gpgsign false

mkdir -p .git/hooks
cp "$HOOKS_SRC_DIR/pre-commit" .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

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
rc=$(try_commit 'docs(qa): capture §1.1 mutation proof')
if [ "$RED_MODE" = "1" ]; then
  assert_block "Q1-evidence-log-with-marker [$EV_SRC] (RED: defect present)" BLOCK "$rc"
else
  assert_block "Q1-evidence-log-with-marker [$EV_SRC] (GREEN: defect absent)" ALLOW "$rc"
  # GREEN additionally requires the audit NOTICE (never a silent exemption).
  if grep -qF "NOTICE (§11.4.84 audit): captured-evidence exemption granted for staged file: $EV_DIR/3_mutation_proof.log" "$HK_OUT" 2>/dev/null; then
    ok "Q1b-evidence-exemption-emits-audit-notice"
  else
    bad "Q1b-evidence-exemption-emits-audit-notice (no NOTICE line in hook output)"
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
assert_block "Q2a-live-source-scripts-with-marker" BLOCK "$(try_commit 'scripts residue')"
unstage scripts/live_gate.sh

cat > helix_code/internal/auth/jwt.go <<'LIVEGO'
package auth

func Verify(t string) bool { return true } // always pass
LIVEGO
git add -- helix_code/internal/auth/jwt.go
assert_block "Q2b-live-source-helix_code-with-marker" BLOCK "$(try_commit 'go residue')"
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
assert_block "Q3-mutation-test-script-with-opt-in-and-restore-idiom" ALLOW "$(try_commit 'exempt mutation test fixture')"

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
assert_block "Q4-opt-in-without-restore-idiom-outside-evidence-dir" BLOCK "$(try_commit 'fake exempt')"
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
assert_block "N1-executable-sh-under-docs-qa-with-marker" BLOCK "$(try_commit 'qa harness residue')"
unstage docs/qa/some_run/harness/run_proof.sh

# N2: a .log carrying a marker OUTSIDE the evidence tree -> BLOCKED.
mkdir -p scripts/logs
printf 'x # MUTATED for paired §1.1 mutation test\n' > scripts/logs/out.log
git add -- scripts/logs/out.log
assert_block "N2-log-with-marker-outside-evidence-tree" BLOCK "$(try_commit 'stray log residue')"
unstage scripts/logs/out.log

# N3: a .sh under docs/qa staged NON-executable (mode 100644) carrying a
# marker -> BLOCKED. Mode alone must not be the whole test: an executable
# EXTENSION is still code regardless of its staged bit.
mkdir -p docs/qa/some_run2
printf '#!/usr/bin/env bash\necho hi # MUTATED for paired §1.1 mutation test\n' > docs/qa/some_run2/helper.sh
chmod 644 docs/qa/some_run2/helper.sh
git add -- docs/qa/some_run2/helper.sh
assert_block "N3-nonexec-mode-but-sh-extension-under-docs-qa" BLOCK "$(try_commit 'qa sh 644 residue')"
unstage docs/qa/some_run2/helper.sh

# N4: an evidence-class .log under docs/qa staged with the EXECUTABLE bit set
# -> BLOCKED. Extension alone must not be the whole test either: a staged
# executable is code the moment it lands, whatever it is named.
mkdir -p docs/qa/some_run3
printf 'output # MUTATED for paired §1.1 mutation test\n' > docs/qa/some_run3/exec_evidence.log
chmod 755 docs/qa/some_run3/exec_evidence.log
git add -- docs/qa/some_run3/exec_evidence.log
assert_block "N4-executable-mode-log-under-docs-qa" BLOCK "$(try_commit 'qa exec log residue')"
unstage docs/qa/some_run3/exec_evidence.log

echo
echo "=== HXC-223 summary (RED_MODE=$RED_MODE): PASS=$PASS FAIL=$FAIL ==="
if [ "$FAIL" -ne 0 ]; then
  printf 'FAILED CASES:\n'
  for c in "${FAILED_CASES[@]}"; do printf '  - %s\n' "$c"; done
  exit 1
fi
exit 0
