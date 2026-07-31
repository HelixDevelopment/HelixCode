#!/usr/bin/env bash
# HXC-215 — §1.1 wolf-catching proof that works in the CURRENT environment.
#
# WHY THIS EXISTS. The gate's built-in --self-test pins (KNOWN_BAD 3fd55a4d /
# KNOWN_GOOD 34e264e1) have bit-rotted: they are 130+ commits old and their
# helix_code/go.mod (sha1 9c9b5912) no longer resolves against today's submodule
# trees (HEAD's is 4960895d), which check_commit symlinks in to hold submodule
# state constant. `go test` there dies with "go: updates to go.mod needed"
# before compiling anything, so neither pin exercises the gate.
#
# This builds a throwaway repo from CURRENT HEAD via `git archive` (no
# `git worktree` against the live repo, so no .git lock contention with other
# agents) containing:
#   c1  base            — HEAD's content, initial commit
#   c2  harmless edit   — touches a .go file, MUST be reported COMPILES
#   c3  real breakage   — references an undefined symbol, MUST be reported
#                         DOES-NOT-COMPILE and MUST be the ONLY commit named
# Then runs the gate with --last 2 so both halves of the §1.1 pairing are
# exercised in one invocation.
set -uo pipefail
SRC="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)"
GATE_SRC="$SRC/scripts/gates/commit_compile_integrity_gate.sh"
TMP="$(mktemp -d /tmp/hxc215-wolf.XXXXXX)"
trap 'rm -rf "$TMP"' EXIT
R="$TMP/repo"; mkdir -p "$R"

echo "source repo:     $SRC"
echo "gate under test: $GATE_SRC"
echo "gate sha256:     $(sha256sum "$GATE_SRC" | awk '{print $1}')"
echo "synthetic repo:  $R"
echo

git -C "$SRC" archive HEAD | tar -x -C "$R"
# The inner module `replace`s to ../submodules/*; point at the live trees exactly
# as check_commit does.
rm -rf "$R/submodules"; ln -s "$SRC/submodules" "$R/submodules"
# The archive carries HEAD's committed gate; the artifact under test is the
# CURRENT working-tree gate, so overwrite it.
cp "$GATE_SRC" "$R/scripts/gates/commit_compile_integrity_gate.sh"

cd "$R"
git init -q . && git config user.email t@t && git config user.name t
git add -A >/dev/null 2>&1; git commit -qm "c1: base from HEAD" >/dev/null

printf '\n// HXC-215 synthetic: harmless comment, must still COMPILE.\n' >> helix_code/internal/version/doc.go
git add helix_code/internal/version/doc.go; git commit -qm "c2: harmless edit to a .go file" >/dev/null
C2="$(git rev-parse --short=8 HEAD)"

cat > helix_code/internal/version/hxc215_synthetic_wolf.go <<'GO'
package version

// HXC-215 synthetic breakage: references a symbol that does not exist, so the
// compiler MUST emit a file.go:line:col diagnostic.
func hxc215SyntheticWolf() int { return hxc215ThisSymbolDoesNotExist }
GO
git add helix_code/internal/version/hxc215_synthetic_wolf.go
git commit -qm "c3: real compile error (undefined symbol)" >/dev/null
C3="$(git rev-parse --short=8 HEAD)"

echo "c2 (must COMPILE):          $C2"
echo "c3 (must NOT compile):      $C3"
echo
OUT="$TMP/out"
bash scripts/gates/commit_compile_integrity_gate.sh --last 2 >"$OUT" 2>&1
RC=$?          # captured DIRECTLY, never after a pipe (§11.4.6)
echo "==================== gate output (--last 2) ===================="
cat "$OUT"
echo "==============================================================="
echo "gate exit: $RC"
echo

fail=0
[ "$RC" -eq 1 ] || { echo "ASSERT FAIL: expected exit 1 (a real non-compiling commit), got $RC"; fail=1; }
grep -q "  $C3  DOES-NOT-COMPILE" "$OUT" || { echo "ASSERT FAIL: c3 $C3 not reported DOES-NOT-COMPILE"; fail=1; }
grep -q "  $C2  COMPILES"         "$OUT" || { echo "ASSERT FAIL: c2 $C2 not reported COMPILES"; fail=1; }
grep -q "^NON-COMPILING COMMITS: $C3$" "$OUT" || { echo "ASSERT FAIL: blamed set is not exactly {$C3}: $(grep '^NON-COMPILING COMMITS:' "$OUT")"; fail=1; }
grep -qE 'hxc215_synthetic_wolf\.go:[0-9]+:[0-9]+: ' "$OUT" || { echo "ASSERT FAIL: no file.go:line:col diagnostic naming the injected file"; fail=1; }
echo "--- diagnostic the gate surfaced ---"
grep -E '\.go:[0-9]+:[0-9]+: ' "$OUT" | head -3 | sed 's/^/  /'
echo
if [ $fail -eq 0 ]; then
  echo "SYNTHETIC WOLF TEST PASS: the gate catches a genuinely broken commit, names"
  echo "ONLY that commit, backs it with a real compiler diagnostic, and exits 1."
  exit 0
fi
echo "SYNTHETIC WOLF TEST FAIL"; exit 1
