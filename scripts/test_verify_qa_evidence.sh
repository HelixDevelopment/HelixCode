#!/usr/bin/env bash
# scripts/test_verify_qa_evidence.sh
#
# Paired-mutation meta-test (§1.1) for the §11.4.83 docs/qa/ end-user
# evidence gate — scripts/verify_qa_evidence.sh, wired as gate G7 of
# scripts/verify-all-constitution-rules.sh.
#
# Why this script exists (§11.4.120 reconciliation, 2026-07-27):
#   G7's match rule was reconciled from a commit-SUBJECT substring guess to
#   an EXACT signal: the docs/qa/ paths the commit ITSELF added, read from
#   `git diff-tree`. A reconciled gate MUST be proven not to be a tautology
#   (§11.4.120: "after reconciliation the gate + mutation still form a valid
#   §1.1 pair"). This meta-test is that proof:
#
#     * MUTATION (evidence absent) -> the gate MUST exit 1 (FAIL).
#     * CONTROL  (evidence present as its own committed path) -> exit 0 (PASS).
#
#   If the reconciled gate had been weakened into "always pass", cases 1, 4
#   and 5 below would go green and this meta-test exits 1.
#
# Why throwaway fixture repositories:
#   The gate's input is git history, so its mutation is a COMMIT. Planting a
#   commit in the real repository is forbidden (no commit / no history
#   rewrite), so each case builds a self-contained repo under `mktemp -d`
#   (same fixture pattern as scripts/test-scan-secrets.sh) and runs the real
#   scripts/verify_qa_evidence.sh against it. The real repository is never
#   written to, committed to, or rewritten.
#
# Cases:
#   1. MUTATION  feature commit, NO evidence of any kind            -> FAIL (1)
#   2. CONTROL   feature commit + docs/qa/<timestamped-run-id>/     -> PASS (0)
#                (run-id is a timestamped slug ABSENT from the subject, so
#                 ONLY the exact rule can match it — this is precisely the
#                 false-positive that was reconciled)
#   3. CONTROL   feature commit + flat docs/qa/<name>.md transcript -> PASS (0)
#                (the second reconciled shape: flat file, not a directory)
#   4. MUTATION  case 2's commit, evidence DELETED from the tree    -> FAIL (1)
#                (proves the still-exists check is real: a commit cannot
#                 bank evidence and then remove it)
#   5. MUTATION  feature commit whose subject merely NAMES docs/qa/ -> FAIL (1)
#                (proves talking about evidence is not committing it)
#   6. CONTROL   non-feature commit (docs only)                     -> PASS (0)
#   7. CONTROL   feature commit with [no-qa-evidence] opt-out       -> PASS (0)
#
# Exit codes:
#   0 — every mutation caught + every control green (gate honoured per §1.1)
#   1 — at least one mutation NOT caught, or a control wrongly failed
#   2 — script setup error
#
# Usage:
#   bash scripts/test_verify_qa_evidence.sh
#
# Side-effects:
#   Creates and removes temporary directories under $TMPDIR. Never touches
#   the invoking repository's index, worktree, or history.

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GATE="$ROOT/scripts/verify_qa_evidence.sh"

[ -x "$GATE" ] || [ -f "$GATE" ] || { echo "setup error: $GATE not found" >&2; exit 2; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

PASS=0
FAIL=0

check() {
    local label="$1" expected_rc="$2" actual_rc="$3"
    if [ "$actual_rc" = "$expected_rc" ]; then
        echo "  ✓ $label (rc=$actual_rc as expected)"
        PASS=$((PASS + 1))
    else
        echo "  ✗ $label (rc=$actual_rc, expected $expected_rc) — GATE IS BLUFFING"
        FAIL=$((FAIL + 1))
    fi
}

# git in a hermetic fixture: no user config, no hooks, no signing.
fixture_git() {
    git -c user.name='Meta Test' \
        -c user.email='meta-test@example.invalid' \
        -c commit.gpgsign=false \
        -c core.hooksPath=/dev/null \
        "$@"
}

# new_fixture <name> -> echoes the fixture repo path with a baseline commit
# already in place (the commit that introduces docs/qa/README.md, mirroring
# the real baseline). The baseline SHA is written to .baseline inside it.
new_fixture() {
    local name="$1" repo="$WORK/$1"
    mkdir -p "$repo/docs/qa"
    (
        cd "$repo" || exit 2
        fixture_git init --quiet -b main
        printf '# docs/qa\n\nEvidence tree (fixture).\n' > docs/qa/README.md
        fixture_git add docs/qa/README.md
        fixture_git commit --quiet --no-verify -m 'feat(qa): establish docs/qa/ evidence tree (baseline)'
        fixture_git rev-parse HEAD > .baseline
    ) || return 2
    echo "$repo"
}

# add_feature_commit <repo> <subject>  — non-test production code under a
# watched root, so the gate's feature-shipping heuristic trips.
add_feature_commit() {
    local repo="$1" subject="$2"
    (
        cd "$repo" || exit 2
        mkdir -p helix_code/internal/demo_feature
        printf 'package demo_feature\n\nfunc Serve() string { return "real" }\n' \
            > helix_code/internal/demo_feature/service.go
        fixture_git add helix_code/internal/demo_feature/service.go
        [ -d docs/qa ] && fixture_git add -A docs/qa
        fixture_git commit --quiet --no-verify -m "$subject"
    )
}

# run_gate <repo> -> exit code of the enforcing gate over baseline..HEAD
run_gate() {
    local repo="$1" out="$2"
    (
        cd "$repo" || exit 2
        bash "$GATE" --enforce --since "$(cat .baseline)" > "$out" 2>&1
    )
    return $?
}

echo "=== §11.4.83 / G7 paired-mutation meta-test (§1.1) ==="
echo "    fixtures under: $WORK"
echo

# ---------------------------------------------------------------------------
# Case 1 — MUTATION: feature commit ships with NO evidence at all.
#          The gate MUST FAIL. This is the non-weakening proof: if the
#          reconciliation had turned G7 into a rubber stamp, this goes green.
# ---------------------------------------------------------------------------
echo "=== Case 1 MUTATION: feature commit, no evidence of any kind ==="
R1="$(new_fixture unevidenced)" || exit 2
add_feature_commit "$R1" 'feat(demo): ship a user-visible feature with no evidence' || exit 2
run_gate "$R1" "$WORK/case1.out"; rc1=$?
check "unevidenced feature commit is caught" 1 "$rc1"

# ---------------------------------------------------------------------------
# Case 2 — CONTROL: the reconciled case. Evidence lives in a TIMESTAMPED
#          run-id directory whose slug appears NOWHERE in the subject, so the
#          legacy subject-substring rule provably cannot match it. Only the
#          exact `git diff-tree` rule can — a PASS here proves the fix works.
# ---------------------------------------------------------------------------
echo
echo "=== Case 2 CONTROL: feature commit adds docs/qa/<timestamped-run-id>/ ==="
R2="$(new_fixture evidenced_dir)" || exit 2
RUNID='demo_feature_20260727T120000Z'
mkdir -p "$R2/docs/qa/$RUNID"
printf '# Evidence\n\nFull bidirectional transcript (fixture).\n' \
    > "$R2/docs/qa/$RUNID/EVIDENCE.md"
add_feature_commit "$R2" 'feat(demo): ship a user-visible feature (evidence attached)' || exit 2
# Guard the premise: the subject must NOT contain the run-id, or this case
# would prove nothing about the exact rule.
if (cd "$R2" && git show --no-patch --format=%s HEAD | grep -q "$RUNID"); then
    echo "  ✗ fixture premise broken: subject contains the run-id" >&2
    FAIL=$((FAIL + 1))
else
    echo "  · premise ok: subject does NOT contain '$RUNID' (legacy rule cannot match)"
fi
run_gate "$R2" "$WORK/case2.out"; rc2=$?
check "commit that adds its own run-id directory passes" 0 "$rc2"
if grep -q 'added by this commit (exact)' "$WORK/case2.out"; then
    echo "  ✓ matched via the EXACT rule (not the legacy guess)"
    PASS=$((PASS + 1))
else
    echo "  ✗ did not match via the EXACT rule — reconciliation not in effect"
    FAIL=$((FAIL + 1))
fi

# ---------------------------------------------------------------------------
# Case 3 — CONTROL: the second reconciled shape — evidence committed as a
#          FLAT docs/qa/<name>.md transcript, which the directory-only
#          enumeration could never see.
# ---------------------------------------------------------------------------
echo
echo "=== Case 3 CONTROL: feature commit adds a flat docs/qa/<name>.md ==="
R3="$(new_fixture evidenced_flat)" || exit 2
printf '# Evidence\n\nFlat transcript (fixture).\n' \
    > "$R3/docs/qa/demo_feature_run_20260727T120500Z.md"
add_feature_commit "$R3" 'feat(demo): ship a user-visible feature (flat transcript)' || exit 2
run_gate "$R3" "$WORK/case3.out"; rc3=$?
check "commit that adds a flat docs/qa/*.md transcript passes" 0 "$rc3"

# ---------------------------------------------------------------------------
# Case 4 — MUTATION: take case 2's PASSING repo and DELETE the evidence from
#          the worktree. The commit still references it, but the evidence is
#          gone, so it is no longer auditable. The gate MUST FAIL.
# ---------------------------------------------------------------------------
echo
echo "=== Case 4 MUTATION: evidence committed, then deleted from the tree ==="
rm -rf "$R2/docs/qa/$RUNID"
run_gate "$R2" "$WORK/case4.out"; rc4=$?
check "deleted-after-commit evidence is caught" 1 "$rc4"

# ---------------------------------------------------------------------------
# Case 5 — MUTATION: the subject TALKS about a docs/qa path but the commit
#          adds none. Naming evidence is not committing it. MUST FAIL.
# ---------------------------------------------------------------------------
echo
echo "=== Case 5 MUTATION: subject merely mentions a docs/qa path ==="
R5="$(new_fixture subject_claims_only)" || exit 2
add_feature_commit "$R5" \
    'feat(demo): ship a feature; see docs/qa/demo_feature_20260727T130000Z/' || exit 2
run_gate "$R5" "$WORK/case5.out"; rc5=$?
check "claiming evidence in the subject does not satisfy the gate" 1 "$rc5"

# ---------------------------------------------------------------------------
# Case 6 — CONTROL: a commit that touches no watched production root is not a
#          feature-shipping commit and must not be dragged in.
# ---------------------------------------------------------------------------
echo
echo "=== Case 6 CONTROL: non-feature (docs-only) commit ==="
R6="$(new_fixture docs_only)" || exit 2
(
    cd "$R6" || exit 2
    printf 'notes\n' > NOTES.md
    fixture_git add NOTES.md
    fixture_git commit --quiet --no-verify -m 'docs: add notes'
) || exit 2
run_gate "$R6" "$WORK/case6.out"; rc6=$?
check "docs-only commit is out of scope" 0 "$rc6"

# ---------------------------------------------------------------------------
# Case 7 — CONTROL: the documented [no-qa-evidence] opt-out still works.
# ---------------------------------------------------------------------------
echo
echo "=== Case 7 CONTROL: [no-qa-evidence] opt-out honoured ==="
R7="$(new_fixture optout)" || exit 2
add_feature_commit "$R7" 'refactor(demo): rename internals only [no-qa-evidence]' || exit 2
run_gate "$R7" "$WORK/case7.out"; rc7=$?
check "[no-qa-evidence] opt-out commit passes" 0 "$rc7"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo
echo "=== test_verify_qa_evidence.sh summary ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
if [ "$FAIL" -gt 0 ]; then
    echo
    echo "FAIL: the §11.4.83 G7 gate did not form a valid §1.1 pair —"
    echo "      a mutation went uncaught or a control wrongly failed."
    exit 1
fi
echo
echo "PASS: G7 fails on genuinely-unevidenced commits and passes only on"
echo "      commits that actually committed their own docs/qa/ evidence."
exit 0
