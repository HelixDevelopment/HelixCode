#!/usr/bin/env bash
# scripts/tests/qa_evidence_citation_coincidence_meta_test.sh
#
# HXC-186 — §11.4.115 RED-baseline-on-the-broken-artifact + standing §11.4.135
# regression guard for the COINCIDENCE hole in the §11.4.83 evidence gate's
# RULE 3 (SHA citation).
#
# THE DEFECT THIS REPRODUCES
#   scripts/verify_qa_evidence.sh RULE 3 clears a feature commit when a tracked
#   file under docs/qa/ contains that commit's 8-hex-char prefix ANYWHERE in
#   its content (`git grep -F <needle>`). But evidence files also record
#   PROVENANCE: scripts/qa/lib/sec_capture_lib.sh's qa_capture_grounding()
#   writes
#       ## repo HEAD
#       <git log --oneline -1>          <-- 8-hex abbreviation of HEAD
#   into docs/qa/<run-id>/grounding.txt. That stamp is the SHA of whatever the
#   repository's HEAD happened to be WHEN THE CAPTURE RAN — which in a
#   multi-agent checkout (§11.4.103 mandates >=3 concurrent streams committing
#   to main) is routinely a DIFFERENT commit from the one the evidence is FOR.
#   A commit stamped into an unrelated run's grounding.txt is therefore cleared
#   by RULE 3 while that evidence demonstrates something else entirely — a
#   §11.4 PASS-bluff inside an ENFORCING release gate.
#
#   Verified on the real repository 2026-07-31: `git log --oneline` abbreviates
#   to exactly 8 hex chars here, which is precisely RULE 3's needle width, so
#   every HEAD stamp is a live match candidate.
#
# THE FIX THIS GUARDS
#   RULE 3 becomes two-tier. For a commit AFTER the citation baseline
#   (QA_CITATION_BASELINE), the needle must appear on a line that ALSO carries
#   a DECLARED CITATION LABEL from a closed documented set — an explicit
#   assertion "this evidence is FOR that commit". A bare provenance stamp
#   carries no label and therefore no longer clears anything. Commits at or
#   before the baseline keep the legacy substring behaviour (see the fix
#   commit's rationale + docs/qa/README.md).
#
# POLARITY SWITCH (§11.4.115)
#   RED_MODE=1 (default) — assert the DEFECT IS PRESENT on the artifact under
#                          test. Passes on the PRE-fix scanner, fails on the
#                          fixed one. This is the reproduction proof.
#   RED_MODE=0           — assert the DEFECT IS ABSENT. This is the STANDING
#                          regression guard wired into G27. Fails on the
#                          pre-fix scanner, passes on the fixed one.
#
#   Both modes run the SAME fixtures and the SAME scanner invocations; only the
#   expected outcomes flip. One source, two roles.
#
# POSITIVE CONTROLS (guard against a fix that simply rejects everything)
#   Case 2 asserts a genuine DECLARED citation still clears its commit, and
#   case 3 asserts a commit that ships its OWN evidence (RULE 1) is untouched.
#   Both must hold in BOTH modes — they are invariants, not polarity cases.
#
# Usage:
#   scripts/tests/qa_evidence_citation_coincidence_meta_test.sh        # RED
#   RED_MODE=0 scripts/tests/qa_evidence_citation_coincidence_meta_test.sh
#
# Exit codes:
#   0  every assertion held for the selected polarity
#   1  at least one assertion failed
#   2  setup error (git unavailable, scanner missing, mktemp failed)
#
# Dependencies: git, bash, POSIX coreutils. No network, no build.
# Cross-references: scripts/verify_qa_evidence.sh, scripts/gates/qa_evidence_gate.sh,
#                   scripts/qa/lib/sec_capture_lib.sh (qa_capture_grounding),
#                   docs/qa/README.md, constitution Constitution.md
#                   §11.4.83 / §11.4.115 / §11.4.135 / §1.1.

set -u

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCANNER="$REPO_ROOT/scripts/verify_qa_evidence.sh"
RED_MODE="${RED_MODE:-1}"

case "$RED_MODE" in
	0|1) ;;
	*) echo "meta-test: RED_MODE must be 0 or 1 (got '$RED_MODE')" >&2; exit 2 ;;
esac

if ! command -v git >/dev/null 2>&1; then
	echo "meta-test: git not on PATH — cannot run" >&2
	exit 2
fi
if [ ! -f "$SCANNER" ]; then
	echo "meta-test: scanner not found at $SCANNER" >&2
	exit 2
fi

TMP="$(mktemp -d -t qa-citation-coincidence.XXXXXX)" || {
	echo "meta-test: mktemp failed" >&2; exit 2; }
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

fail_count=0
assert_eq() {
	# $1 = description, $2 = expected, $3 = actual
	if [ "$2" = "$3" ]; then
		echo "  PASS: $1 (got $3)"
	else
		echo "  FAIL: $1 (expected $2, got $3)" >&2
		fail_count=$((fail_count + 1))
	fi
}

# --------- Build a throwaway git repo ---------
WORK="$TMP/repo"
mkdir -p "$WORK"
cd "$WORK" || { echo "meta-test: cd failed" >&2; exit 2; }

git init -q
git config user.email "meta-test@helix.local"
git config user.name "meta-test"
git config commit.gpgsign false
git config core.hooksPath /dev/null

mkdir -p docs/qa helix_code/internal/seam

# Commit 1: BASELINE — establishes the docs/qa convention.
printf '# docs/qa\nconvention established\n' > docs/qa/README.md
git add docs/qa/README.md
git commit -q -m "feat(qa): establish docs/qa/ evidence tree (baseline)"
BASELINE="$(git rev-parse HEAD)"

# Every fixture commit below is a DESCENDANT of BASELINE, so setting the
# citation baseline to BASELINE puts them all in the STRICT forward tier.
export QA_CITATION_BASELINE="$BASELINE"

run_enforce() {
	# Enforcing scan of the temp repo. The exit code is read directly from the
	# scanner invocation — never after a pipe (§11.4.6).
	bash "$SCANNER" --enforce --since "$BASELINE" >"$TMP/scan.out" 2>&1
	echo "$?"
}

echo "==========================================================================="
echo " HXC-186 — RULE 3 citation-vs-coincidence meta-test"
echo "   RED_MODE=$RED_MODE  ($([ "$RED_MODE" = 1 ] && echo 'asserting the DEFECT IS PRESENT (reproduction)' || echo 'asserting the DEFECT IS ABSENT (regression guard)'))"
echo "   temp repo : $WORK"
echo "   baseline  : $BASELINE"
echo "==========================================================================="

# ===========================================================================
# CASE 1 — THE COINCIDENCE.
#
# A feature commit that:
#   * ships NO evidence of its own          (defeats RULE 1)
#   * has a subject naming no run-id        (defeats RULE 2)
#   * is NOT cited by any declared field    (must defeat a correct RULE 3)
# but whose 8-char SHA is auto-stamped as the "## repo HEAD" provenance line
# of an unrelated run's grounding.txt — exactly what qa_capture_grounding()
# writes when a capture runs while HEAD sits on that commit.
# ===========================================================================
printf 'package seam\n\nfunc Orphan() {}\n' > helix_code/internal/seam/orphan.go
git add helix_code/internal/seam/orphan.go
git commit -q -m "feat(seam): add Orphan with no evidence of its own"
ORPHAN_FULL="$(git rev-parse HEAD)"
ORPHAN_STAMP="${ORPHAN_FULL:0:8}"

# The unrelated run. Its EVIDENCE.md DECLARES a different commit; only the
# machine-written provenance stamp mentions ORPHAN.
mkdir -p docs/qa/unrelated_run_20260731T000000Z
{
	echo "# Unrelated capture — QA evidence (§11.4.83)"
	echo
	echo "**Run ID:** \`unrelated_run_20260731T000000Z\`"
	echo "**Source commit:** \`deadbeefcafe0123456789abcdef0123456789ab\`"
	echo
	echo "This transcript documents something else entirely."
} > docs/qa/unrelated_run_20260731T000000Z/EVIDENCE.md
# Byte-for-byte the shape qa_capture_grounding() emits (heading on its own
# line, `git log --oneline -1` output on the next).
{
	echo "# Grounding facts captured 2026-07-31T00:00:00Z"
	echo "# §11.4.108: the running artifact must be the one built from the fixed source."
	echo
	echo "## repo HEAD"
	echo "${ORPHAN_STAMP} feat(seam): add Orphan with no evidence of its own"
	echo
	echo "## fix commit under test"
	echo "deadbeef chore: the commit this evidence is actually about"
} > docs/qa/unrelated_run_20260731T000000Z/grounding.txt
git add docs/qa/unrelated_run_20260731T000000Z
git commit -q -m "docs(qa): unrelated capture [no-qa-evidence]"

rc="$(run_enforce)"
if [ "$RED_MODE" = "1" ]; then
	assert_eq "(1) RED: a bare HEAD-provenance stamp CLEARS an unrelated feature commit → exit 0" "0" "$rc"
else
	assert_eq "(1) GREEN: a bare HEAD-provenance stamp does NOT clear an unrelated feature commit → exit 1" "1" "$rc"
fi

# Diagnostic: show how the scanner classified the orphan commit.
if grep -q "$ORPHAN_STAMP" "$TMP/scan.out" 2>/dev/null; then
	echo "      scanner said: $(grep -m1 -A1 "$ORPHAN_STAMP" "$TMP/scan.out" | tail -1 | sed -E 's/^[[:space:]]+//' | cut -c1-96)"
fi

# ===========================================================================
# CASE 2 — POSITIVE CONTROL (invariant in BOTH modes).
#
# The SAME commit, now cited by a DECLARED field in a tracked evidence file,
# must be cleared. This is what stops the fix from degenerating into "reject
# every citation": a real declaration still works.
# ===========================================================================
mkdir -p docs/qa/orphan_fix_20260731T010000Z
{
	echo "# Orphan seam — QA evidence (§11.4.83)"
	echo
	echo "**Run ID:** \`orphan_fix_20260731T010000Z\`"
	echo "**Source commit:** \`${ORPHAN_FULL}\`"
	echo
	echo "sent: GET /seam"
	echo "recv: 200 OK"
} > docs/qa/orphan_fix_20260731T010000Z/EVIDENCE.md
git add docs/qa/orphan_fix_20260731T010000Z
git commit -q -m "docs(qa): declared evidence for the Orphan seam [no-qa-evidence]"

rc="$(run_enforce)"
assert_eq "(2) INVARIANT: a DECLARED '**Source commit:**' citation clears its commit → exit 0" "0" "$rc"

# ===========================================================================
# CASE 3 — POSITIVE CONTROL (invariant in BOTH modes).
#
# RULE 1 (the commit ships its own evidence) is untouched by this change.
# ===========================================================================
printf 'package seam\n\nfunc SelfEvidenced() {}\n' > helix_code/internal/seam/self.go
mkdir -p docs/qa/self_evidenced_20260731T020000Z
printf '# self-evidenced\nsent: x\nrecv: y\n' > docs/qa/self_evidenced_20260731T020000Z/EVIDENCE.md
git add helix_code/internal/seam/self.go docs/qa/self_evidenced_20260731T020000Z
git commit -q -m "feat(seam): SelfEvidenced ships its own transcript"

rc="$(run_enforce)"
assert_eq "(3) INVARIANT: RULE 1 (commit ships its own docs/qa path) still clears → exit 0" "0" "$rc"

# ===========================================================================
# CASE 4 — the coincidence class is not limited to the HEAD stamp.
#
# Any machine-written SHA qualifies today: a scratch-checkout path, a nonce,
# a build log. This case uses a temp-directory path of exactly the shape found
# in the real corpus (the compile-integrity gate's per-commit worktree), to
# prove the guard keys on the ABSENCE OF A DECLARATION rather than on one
# recognised stamp format. A denylist of known stamps would pass case 1 and
# fail here.
# ===========================================================================
printf 'package seam\n\nfunc Second() {}\n' > helix_code/internal/seam/second.go
git add helix_code/internal/seam/second.go
git commit -q -m "feat(seam): add Second with no evidence of its own"
SECOND_STAMP="$(git rev-parse HEAD)"; SECOND_STAMP="${SECOND_STAMP:0:8}"

mkdir -p docs/qa/other_run_20260731T030000Z
{
	echo "# Another capture — QA evidence (§11.4.83)"
	echo
	echo "**Source commit:** \`deadbeefcafe0123456789abcdef0123456789ab\`"
} > docs/qa/other_run_20260731T030000Z/EVIDENCE.md
{
	echo "### case: compile_probe"
	echo "chdir /tmp/.private/runner/cci-gate.XXXX/${SECOND_STAMP}/helix_code"
	echo "--- exit code: 0"
} > docs/qa/other_run_20260731T030000Z/transcript.txt
git add docs/qa/other_run_20260731T030000Z
git commit -q -m "docs(qa): another capture [no-qa-evidence]"

rc="$(run_enforce)"
if [ "$RED_MODE" = "1" ]; then
	assert_eq "(4) RED: an incidental scratch-path SHA CLEARS an unrelated feature commit → exit 0" "0" "$rc"
else
	assert_eq "(4) GREEN: an incidental scratch-path SHA does NOT clear it → exit 1" "1" "$rc"
fi

# --------- Verdict ---------
echo "---------------------------------------------------------------------------"
if [ "$fail_count" -eq 0 ]; then
	if [ "$RED_MODE" = "1" ]; then
		echo " RESULT: PASS — the coincidence defect is PRESENT on this artifact"
		echo "         (RED reproduction confirmed; run with RED_MODE=0 after the fix)."
	else
		echo " RESULT: PASS — the coincidence defect is ABSENT on this artifact"
		echo "         (regression guard green; declared citations and RULE 1 unaffected)."
	fi
	echo "==========================================================================="
	exit 0
fi
echo " RESULT: FAIL — ${fail_count} assertion(s) failed (RED_MODE=$RED_MODE)." >&2
echo "===========================================================================" >&2
exit 1
