#!/usr/bin/env bash
# scripts/tests/verify_qa_evidence_meta_test.sh
#
# §1.1 paired-mutation meta-test for the §11.4.83 docs/qa/ evidence gate
# (scripts/verify_qa_evidence.sh --enforce). HXC-019.
#
# Anti-bluff guarantee: the gate is only trustworthy if it can be SHOWN to
# FAIL when evidence is missing and PASS when it is present. This meta-test
# builds a throwaway temp git repository, drives the scanner against
# synthesized history, and asserts:
#
#   (A) a post-baseline feature commit WITH NO docs/qa/<run-id>/ dir
#       → --enforce exits NON-zero (1).
#   (B) after adding the matching docs/qa/<run-id>/ transcript
#       → --enforce exits 0.
#   (C) a [no-qa-evidence]-tagged feature commit is EXEMPT
#       → --enforce exits 0 even with no evidence dir.
#
# Plus two robustness assertions:
#   (D) --enforce WITHOUT --since is misuse → exit 2 (baseline mandatory).
#   (E) commits BEFORE the baseline are exempt (scope is <since>..HEAD).
#
# The meta-test itself exits NON-zero if ANY assertion fails (so it is a
# real gate, not a bluff). All artefacts live under a mktemp dir cleaned in
# a trap on EXIT.
#
# Usage:
#   scripts/tests/verify_qa_evidence_meta_test.sh
#
# Exit codes:
#   0  all assertions passed
#   1  at least one assertion failed
#   2  setup error (git unavailable, scanner missing, etc.)
#
# Dependencies: git, bash, POSIX coreutils.
# Cross-references: scripts/verify_qa_evidence.sh, docs/qa/README.md,
#                   constitution Constitution.md §11.4.83 / §1.1.

set -u

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCANNER="$REPO_ROOT/scripts/verify_qa_evidence.sh"

if ! command -v git >/dev/null 2>&1; then
	echo "meta-test: git not on PATH — cannot run" >&2
	exit 2
fi
if [ ! -f "$SCANNER" ]; then
	echo "meta-test: scanner not found at $SCANNER" >&2
	exit 2
fi

TMP="$(mktemp -d -t qa-evidence-meta.XXXXXX)" || { echo "meta-test: mktemp failed" >&2; exit 2; }
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

mkdir -p docs/qa "helix_code/internal/auth"

# Commit 0: PRE-baseline feature commit (no docs/qa anything yet).
printf 'package auth\n// legacy pre-convention prod code\n' > helix_code/internal/auth/legacy.go
git add helix_code/internal/auth/legacy.go
git commit -q -m "legacy: pre-convention feature touching prod code"

# Commit 1: BASELINE — adds docs/qa/README.md (the convention introduction).
printf '# docs/qa\nconvention established\n' > docs/qa/README.md
git add docs/qa/README.md
git commit -q -m "feat(qa): establish docs/qa/ evidence tree (baseline)"
BASELINE="$(git rev-parse HEAD)"

run_enforce() {
	# Runs the scanner in enforcing mode against the temp repo; echoes exit code.
	bash "$SCANNER" --enforce --since "$BASELINE" >/dev/null 2>&1
	echo "$?"
}

echo "=== §11.4.83 docs/qa evidence gate — meta-test (HXC-019) ==="
echo "    temp repo: $WORK"
echo "    baseline : $BASELINE"
echo "-----------------------------------------------------------------------"

# --------- Assertion (E): pre-baseline commit is out of scope ---------
# At this point the only post-baseline commits are: the baseline itself
# (touches docs/qa, not prod code → not a feature commit). The pre-baseline
# legacy feature commit must NOT be evaluated. Expect PASS (exit 0).
rc="$(run_enforce)"
assert_eq "(E) pre-baseline legacy feature commit is exempt → exit 0" "0" "$rc"

# --------- Assertion (A): post-baseline feature, NO evidence → FAIL ---------
printf 'package auth\n// new feature seam\nfunc NewThing() {}\n' > helix_code/internal/auth/feature.go
git add helix_code/internal/auth/feature.go
git commit -q -m "feat(auth): add NewThing seam (HXC-META-A)"
rc="$(run_enforce)"
assert_eq "(A) post-baseline feature commit, no docs/qa dir → exit 1" "1" "$rc"

# --------- Assertion (B): add matching docs/qa dir → PASS ---------
# The run-id (HXC-META-A) must appear in BOTH the commit subject (it does,
# above) AND as a docs/qa/<run-id>/ directory name. Add the transcript.
mkdir -p docs/qa/HXC-META-A
printf '# transcript\nsent: x\nrecv: y\n' > docs/qa/HXC-META-A/transcript.md
git add docs/qa/HXC-META-A/transcript.md
git commit -q -m "docs(qa): add HXC-META-A end-user evidence transcript [no-qa-evidence]"
# NOTE: the docs commit itself touches no prod code → not a feature commit;
# the [no-qa-evidence] tag just keeps it clearly exempt. The earlier feature
# commit (HXC-META-A) is now satisfied because docs/qa/HXC-META-A/ exists.
rc="$(run_enforce)"
assert_eq "(B) matching docs/qa/HXC-META-A/ now present → exit 0" "0" "$rc"

# --------- Assertion (C): [no-qa-evidence]-tagged feature is exempt ---------
printf 'package auth\n// refactor, no user-facing behaviour\nfunc internalHelper() {}\n' > helix_code/internal/auth/refactor.go
git add helix_code/internal/auth/refactor.go
git commit -q -m "refactor(auth): rename internal helper [no-qa-evidence]"
rc="$(run_enforce)"
assert_eq "(C) [no-qa-evidence]-tagged feature commit is exempt → exit 0" "0" "$rc"

# Negative control for (C): an UNTAGGED feature commit with no evidence must
# still FAIL — proving the exemption is the opt-out token, not a blanket pass.
printf 'package auth\n// genuinely new user-facing seam\nfunc PublicSeam() {}\n' > helix_code/internal/auth/public.go
git add helix_code/internal/auth/public.go
git commit -q -m "feat(auth): add PublicSeam (untagged, no evidence)"
rc="$(run_enforce)"
assert_eq "(C-neg) untagged feature commit, no evidence → exit 1" "1" "$rc"

# --------- Assertion (F): RULE 3 — evidence lands in the PAIRED close commit
# and cites the feature commit's SHA. This is the project's normal two-commit
# workflow (feature commit ships code; `close(...)` commit lands
# docs/qa/<run-id>/EVIDENCE.md). RULE 1 cannot see it (wrong tree) and RULE 2
# cannot see it (the run-id is a timestamped slug absent from the subject), so
# this exercises RULE 3 in isolation. The (C-neg) feature commit above is
# still outstanding — satisfying it here is what flips the gate green.
feat_sha="$(git log -1 --format='%H' --grep='add PublicSeam')"
feat_sha8="$(printf '%s' "$feat_sha" | cut -c1-8)"
meta_runid="publicseam_meta_20260728T000000Z"
mkdir -p "docs/qa/$meta_runid"
printf '# PublicSeam — QA evidence\n\n**Fix commit:** %s\n\nsent: x\nrecv: y\n' \
	"$feat_sha8" > "docs/qa/$meta_runid/EVIDENCE.md"
git add "docs/qa/$meta_runid/EVIDENCE.md"
git commit -q -m "close(meta): land PublicSeam evidence [no-qa-evidence]"
rc="$(run_enforce)"
assert_eq "(F) RULE 3 — paired close commit cites feature SHA → exit 0" "0" "$rc"

# Mutation 1 for (F): the SAME evidence file stays ON DISK but becomes
# UNTRACKED. RULE 3 matches with `git grep`, which searches tracked files
# only, so an untracked drop-in MUST NOT clear the gate.
git rm -q --cached "docs/qa/$meta_runid/EVIDENCE.md"
git commit -q -m "mutate(meta): untrack the evidence file [no-qa-evidence]"
rc="$(run_enforce)"
assert_eq "(F-mut1) RULE 3 — untracked evidence must NOT pass → exit 1" "1" "$rc"

# Mutation 2 for (F): tracked again, but citing a DIFFERENT SHA. Proves
# RULE 3 binds to THIS commit — it is not a blanket "some evidence exists"
# pass, which would be the tautology this rule must never become.
printf '# unrelated evidence\n\n**Fix commit:** deadbeefcafe1234\n' \
	> "docs/qa/$meta_runid/EVIDENCE.md"
git add -f "docs/qa/$meta_runid/EVIDENCE.md"
git commit -q -m "mutate(meta): cite a different SHA [no-qa-evidence]"
rc="$(run_enforce)"
assert_eq "(F-mut2) RULE 3 — evidence citing another SHA must NOT pass → exit 1" "1" "$rc"

# Restore the genuine citation, then assert the gate recovers — the paired
# mutation is only meaningful if the un-mutated state is provably green.
printf '# PublicSeam — QA evidence\n\n**Fix commit:** %s\n\nsent: x\nrecv: y\n' \
	"$feat_sha8" > "docs/qa/$meta_runid/EVIDENCE.md"
git add -f "docs/qa/$meta_runid/EVIDENCE.md"
git commit -q -m "restore(meta): genuine evidence citation [no-qa-evidence]"
rc="$(run_enforce)"
assert_eq "(F-restore) RULE 3 — genuine citation restored → exit 0" "0" "$rc"

# --------- Assertion (D): --enforce without --since is misuse ---------
bash "$SCANNER" --enforce >/dev/null 2>&1
rc="$?"
assert_eq "(D) --enforce without --since → exit 2 (misuse)" "2" "$rc"

echo "-----------------------------------------------------------------------"
if [ "$fail_count" -eq 0 ]; then
	echo "=== meta-test RESULT: PASS (all assertions held) ==="
	exit 0
fi
echo "=== meta-test RESULT: FAIL ($fail_count assertion(s) failed) ===" >&2
exit 1
