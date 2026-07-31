#!/usr/bin/env bash
# verify_qa_evidence.sh — §11.4.83 docs/qa/ end-user evidence gate.
#
# Purpose:
#   Scans feature-shipping commits and reports when a feature commit has
#   no matching docs/qa/<run-id>/ directory carrying its end-user evidence
#   transcript (per constitution submodule Constitution.md §11.4.83 —
#   docs/qa/ end-user evidence mandate).
#
# Two modes:
#   ADVISORY  (default)  — ALWAYS exits 0. Prints a warn-mode notice for
#                          ad-hoc visibility. NOT wired into any git hook.
#   ENFORCING (--enforce) — exits 1 if any in-scope feature-shipping commit
#                          lacks its docs/qa/<run-id>/ directory. This is the
#                          §11.4.83 operative-rule-(5) release gate. The
#                          operator AUTHORISED promotion to a blocking
#                          release gate on 2026-05-28 (HXC-019). Wired into
#                          scripts/release-gate-test.sh via
#                          scripts/gates/qa_evidence_gate.sh — release-gate
#                          ONLY, never pre-commit / pre-push.
#
# Baseline scoping (--since):
#   §11.4.83 / the docs/qa convention was introduced by the commit that
#   ADDED docs/qa/README.md:
#       ed84f90e  2026-05-28T16:09:55+03:00
#       feat(qa): establish docs/qa/ end-user evidence tree + advisory gate
#                 (§11.4.83) (HXC-019)
#   Find it yourself with:
#       git log --diff-filter=A --format='%H %cI %s' -- docs/qa/README.md
#   Commits BEFORE the baseline predate the convention and are EXEMPT — the
#   tree did not exist, so they cannot be expected to carry a docs/qa entry.
#   --enforce mode therefore REQUIRES a --since baseline; without it the
#   whole repository history (thousands of legacy feature commits) would be
#   blocking and HEAD would become un-releasable. --since accepts any git
#   revision OR an approxidate (e.g. a SHA, a tag, or "2026-05-28").
#   Implementation: only commits in the range "<since>..HEAD" are evaluated
#   (i.e. descendants of the baseline), so the baseline commit itself is
#   exempt and merge-ancestry — not author-date sorting — defines scope.
#
# Per-commit opt-out:
#   A commit whose message (subject OR body) contains the literal token
#       [no-qa-evidence]
#   is EXEMPT (e.g. a pure refactor, governance-only change, or doc-only
#   change that trips the heuristic). Documented in docs/qa/README.md.
#
# Heuristic for "feature-shipping commit":
#   A commit that touches non-test production code under any of:
#     helix_code/internal/**   helix_code/cmd/**   helix_code/applications/**
#   excluding *_test.go files. Deliberately loose — false positives are
#   handled by the [no-qa-evidence] opt-out in enforcing mode.
#
# Match rule (does a commit carry its evidence?):
#   RULE 1 — EXACT (primary, §11.4.120 reconciliation 2026-07-27):
#     The commit ITSELF added/changed a path under docs/qa/, and that
#     evidence entry still exists on disk. A commit that commits its own
#     evidence path IS the proof — unfakeable, derived from the commit's
#     own tree via `git diff-tree --no-commit-id --name-only -r <sha>`.
#     BOTH §11.4.83 evidence shapes are recognised:
#       * run-id directory : docs/qa/<run-id>/<file>
#       * flat transcript  : docs/qa/<name>.md
#     docs/qa/README.md is the CONVENTION document, not evidence — excluded.
#
#   RULE 3 — CITATION (proof, §11.4.120 reconciliation 2026-07-28;
#            TIGHTENED to a DECLARED citation 2026-07-31, HXC-186):
#     A TRACKED docs/qa/ evidence file cites this commit's SHA (full or
#     abbreviated, >=8 hex chars). Matched with `git grep`, which searches
#     TRACKED files only — an untracked file dropped into the working tree
#     can NOT clear the gate (verified by negative control).
#     docs/qa/README.md is excluded as the convention doc.
#
#     RULE 3 IS TWO-TIER (HXC-186). What counts as a "citation" depends on
#     whether the commit is newer than the CITATION BASELINE:
#
#       STRICT tier  — commits strictly AFTER $QA_CITATION_BASELINE.
#         The SHA must appear ON THE SAME LINE as a DECLARED CITATION LABEL
#         drawn from this closed set (case-insensitive, markdown decoration
#         such as `**`/`|`/`#`/`>` tolerated before the label):
#             Evidence-For-Commit:      (the canonical machine-readable field)
#             Source commit:            (emitted by qa_finish, see below)
#             Fix commit:
#             Commit under test:
#         i.e. an EXPLICIT assertion "this evidence is FOR that commit".
#         A SHA appearing anywhere else in the file no longer clears it.
#
#       LEGACY tier  — commits at or before the baseline: the historical
#         substring behaviour, reported as "legacy substring" so it is never
#         mistaken for a declaration.
#
#   WHY THE STRICT TIER EXISTS (HXC-186 — the coincidence hole):
#     Evidence files record PROVENANCE as well as citations.
#     scripts/qa/lib/sec_capture_lib.sh's qa_capture_grounding() writes
#         ## repo HEAD
#         <git log --oneline -1>
#     into docs/qa/<run-id>/grounding.txt — the SHA of whatever the repo's
#     HEAD happened to be WHEN THE CAPTURE RAN. In this repository
#     `git log --oneline` abbreviates to exactly 8 hex chars, which is
#     precisely the RULE 3 needle width (measured 2026-07-31), so every such
#     stamp was a live match. In a multi-agent checkout (§11.4.103 mandates
#     >=3 concurrent streams committing to main) that HEAD is routinely a
#     DIFFERENT commit from the one the evidence documents — so an unrelated
#     feature commit could be cleared by evidence demonstrating something
#     else entirely: a §11.4 PASS-bluff inside an ENFORCING release gate.
#     The stamp is NOT the only such class — the corpus also carries SHAs in
#     scratch-checkout paths (`chdir /tmp/.../<sha>/helix_code`), request
#     nonces and build logs. The strict tier therefore keys on the PRESENCE
#     OF A DECLARATION, not on a denylist of recognised stamp formats: a
#     denylist could only ever exclude the coincidence classes someone
#     already thought of, which is the same shape of hole as the defect.
#     The provenance stamp is deliberately KEPT — it is genuinely useful
#     §11.4.108 grounding; it simply no longer counts as a citation.
#
#   Why the legacy tier exists (the 400+ existing evidence directories):
#     At the time of the fix, 42 of the 57 in-range feature commits were
#     cleared by RULE 3. All 42 were inspected individually: every one
#     resolves to a DELIBERATE citation of the commit under test — 37 via a
#     labelled field, 5 via a markdown/HTML table cell or prose ("| Commits |
#     `b741d7da` …", "commit `a52a523a`; greppable via …"). NONE was a
#     coincidence. Requiring the strict form retroactively would therefore
#     have turned 5 honest, human-written citations into release-blocking
#     violations without improving truthfulness. They are quarantined by the
#     baseline instead — the same instrument this gate already uses for
#     historical scope — and the strict rule binds every commit from the
#     baseline forward. This is a QUARANTINE, not a claim of cleanliness:
#     the legacy tier remains coincidence-capable by construction, which is
#     exactly why nothing new may enter it.
#
#   Citation baseline (--citation-baseline / $QA_CITATION_BASELINE):
#     Default: the repository HEAD at the time HXC-186 landed. If the value
#     cannot be resolved in this checkout the gate FAILS CLOSED — every
#     commit is held to the STRICT tier — because an unknown baseline must
#     never silently grant legacy leniency (§11.4.6).
#
#   Why RULE 3 exists (the second false-positive class this reconciles):
#     The project's normal workflow is a PAIR of commits: the feature
#     commit ships the code, and a following `close(HXC-NNN): … (→ Fixed.md)`
#     commit lands docs/qa/<run-id>/EVIDENCE.md and moves the tracker item.
#     RULE 1 only inspects the commit's OWN tree, so it cannot see evidence
#     that landed in the paired close commit; RULE 2 cannot see it either,
#     because the subject says "HXC-119" while the run-id directory is
#     "hxc119_20260712T193000Z" (case + timestamp suffix differ). Five such
#     commits were false-positives before this reconciliation — each with a
#     tracked EVIDENCE.md that names its fix commit SHA verbatim:
#       fbfffd7d -> docs/qa/hxc119_20260712T193000Z/EVIDENCE.md
#       0e3bb747 -> docs/qa/hxc145_147_20260712T190000Z/EVIDENCE.md
#       6efadd15 -> docs/qa/hxc148_20260712T173000Z/EVIDENCE.md
#       225cdf77 -> docs/qa/hxc118_20260712T151500Z/EVIDENCE.md
#       aa6b20b4 -> docs/qa/hxc117_20260712T140900Z/EVIDENCE.md
#     A SHA citation is PROOF, not a guess: the evidence document names the
#     exact commit it documents, so the binding is machine-checkable and
#     cannot be satisfied by coincidence the way RULE 2 can.
#
#   Honest boundary for RULE 3 (§11.4.6 — stated, not papered over):
#     RULE 3 asserts an evidence file EXISTS and is BOUND to this commit.
#     It does NOT assert the evidence is deep enough to satisfy §11.4.83's
#     bidirectional-transcript bar — that remains a human/QA judgement.
#     Residual risk: a single ledger-style document enumerating many commit
#     SHAs would clear every commit it lists. This is NOT mechanically
#     capped here (any cap would be an invented threshold, §11.4.6); instead
#     the gate PRINTS THE CITING FILE PATH on every RULE 3 pass, so a
#     reviewer can tell a per-feature transcript from a blanket ledger at a
#     glance. No such ledger exists in docs/qa today (verified 2026-07-28:
#     each of the five matches resolved to exactly one ticket-scoped
#     EVIDENCE.md, and the 13 remaining violations matched nothing).
#
#   RULE 2 — LEGACY subject-substring heuristic (fallback, retained):
#     The commit subject contains the basename of an existing
#     docs/qa/<run-id>/ directory (e.g. subject mentions "HXC-019" and
#     docs/qa/HXC-019/ exists).
#
#   Why RULE 1 exists (the false-positive this reconciles):
#     RULE 2 alone required the COMMIT SUBJECT to contain the evidence
#     directory's basename. Run-ids are timestamped slugs — e.g.
#     hxc135_20260712T130921Z — that no commit subject would ever contain,
#     so a commit that DID ship its evidence was still reported VIOL.
#     RULE 2 also only ever enumerated DIRECTORIES (`for d in "$QA_DIR"/*/`),
#     so evidence committed as a flat docs/qa/*.md transcript was invisible.
#     Six such commits were false-positives before this reconciliation
#     (4 timestamped-run-id, 2 flat-file). RULE 2 is kept ONLY so that no
#     previously-passing commit regresses; it is a GUESS (it can match by
#     pure coincidence) and is labelled as such in the report output, so a
#     reader can tell proof from guess at a glance.
#
#   RULES 1 and 3 are strictly ADDITIVE — they can only clear commits whose
#   evidence genuinely exists in the repository (committed by the commit
#   itself, or committed by its paired close commit and citing its SHA). A
#   feature commit that adds NO docs/qa/ path, is cited by NO tracked
#   evidence file, and whose subject names no existing run-id is still a
#   VIOLATION. Evaluation order is PROOF FIRST, GUESS LAST:
#   RULE 1 (own tree) -> RULE 3 (SHA citation) -> RULE 2 (subject guess).
#
# Usage:
#   scripts/verify_qa_evidence.sh [N]
#       Advisory scan of the last N commits (default 20). Always exit 0.
#   scripts/verify_qa_evidence.sh --enforce --since <ref-or-date> [N]
#       Enforcing scan over <ref-or-date>..HEAD. Exit 1 on any violation,
#       exit 0 when clean. N caps the number of in-range commits scanned
#       (default: all commits in range).
#   scripts/verify_qa_evidence.sh --enforce            (no --since)
#       ERROR — refuses to run (exit 2). --since is mandatory in enforcing
#       mode to avoid blocking on pre-convention legacy history.
#   scripts/verify_qa_evidence.sh --help
#
# Inputs:
#   git history of the current repository; the docs/qa/ tree on disk.
#
# Outputs:
#   A human-readable report on stdout; diagnostics on stderr in --enforce.
#   Exit: advisory → always 0; enforcing → 0 clean / 1 violation / 2 misuse.
#
# Side-effects:
#   None. Read-only over git history + the working tree.
#
# Dependencies:
#   git, POSIX coreutils, bash (honest shebang — uses bash arrays).
#
# Cross-references:
#   docs/qa/README.md (the convention), docs/scripts/verify_qa_evidence.md
#   (companion guide), scripts/gates/qa_evidence_gate.sh (release-gate
#   wiring), scripts/release-gate-test.sh, constitution Constitution.md
#   §11.4.83.

set -u

# --------- Argument parsing ---------
MODE_ENFORCE=0
SINCE_REF=""
SCAN_N=""
WANT_HELP=0

usage() {
	cat <<'EOF'
Usage:
  scripts/verify_qa_evidence.sh [N]
      Advisory scan of the last N commits (default 20). ALWAYS exits 0.

  scripts/verify_qa_evidence.sh --enforce --since <ref-or-date> [N]
      Enforcing release gate over <ref-or-date>..HEAD.
      Exit 0 when every in-range feature-shipping commit carries its
      docs/qa/<run-id>/ directory; exit 1 on any violation.
      --since is MANDATORY in enforcing mode (baseline scoping — avoids
      blocking on pre-convention legacy history). N optionally caps the
      number of in-range commits scanned.

  scripts/verify_qa_evidence.sh --help

Per-commit opt-out: a commit whose message contains the token
[no-qa-evidence] is exempt (pure refactor / governance / docs).

Exit codes:
  0  clean (advisory always; enforcing when no violations)
  1  enforcing: at least one in-range feature commit lacks docs/qa evidence
  2  misuse (e.g. --enforce without --since, unknown flag, bad --since ref)
EOF
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--enforce)        MODE_ENFORCE=1 ;;
		--since)          shift; SINCE_REF="${1:-}" ;;
		--since=*)        SINCE_REF="${1#--since=}" ;;
		-h|--help|help)   WANT_HELP=1 ;;
		--*)              echo "verify_qa_evidence.sh: unknown flag: $1" >&2; usage >&2; exit 2 ;;
		*)
			if [ -z "$SCAN_N" ]; then
				SCAN_N="$1"
			else
				echo "verify_qa_evidence.sh: unexpected argument: $1" >&2
				usage >&2
				exit 2
			fi
			;;
	esac
	shift
done

if [ "$WANT_HELP" -eq 1 ]; then
	usage
	exit 0
fi

# Resolve repo root so the script works from any cwd.
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$REPO_ROOT" ]; then
	if [ "$MODE_ENFORCE" -eq 1 ]; then
		echo "verify_qa_evidence.sh: not inside a git repository (--enforce requires one)." >&2
		exit 2
	fi
	echo "verify_qa_evidence.sh: not inside a git repository — nothing to scan (advisory)."
	exit 0
fi
cd "$REPO_ROOT" || exit 2

QA_DIR="docs/qa"

# --------- Citation baseline (HXC-186) ---------
# Commits strictly AFTER this ref must carry a DECLARED citation (see the
# RULE 3 header); commits at or before it keep the legacy substring match.
# Default = repository HEAD when HXC-186 landed (2026-07-31).
CITATION_BASELINE_DEFAULT="249ae5dcef3d4107d1161515a9b1d9206bfa5ca5"
CITATION_BASELINE="${QA_CITATION_BASELINE:-$CITATION_BASELINE_DEFAULT}"
CITATION_BASELINE_RESOLVED=""
if git rev-parse --verify --quiet "${CITATION_BASELINE}^{commit}" >/dev/null 2>&1; then
	CITATION_BASELINE_RESOLVED="$(git rev-parse "${CITATION_BASELINE}^{commit}" 2>/dev/null || true)"
fi

# Declared-citation labels — the CLOSED set (§11.4.6: closed, documented, not
# open-ended pattern-guessing). The SHA must sit on the SAME LINE as one of
# these labels. Markdown/HTML decoration before the label is tolerated; the
# trailing colon is REQUIRED, which is what separates a declaration
# ("Source commit: <sha>") from a provenance heading ("## repo HEAD" with the
# SHA on the NEXT line). A bare "Commit:" is deliberately NOT accepted — it is
# generic enough to appear incidentally in a shell transcript.
QA_CITATION_LABEL_RE='^[[:space:]]*[-*#>|[:space:]]*(\*\*)?(evidence-for-commit|source[[:space:]]+commit|fix[[:space:]]+commit|commit[[:space:]]+under[[:space:]]+test)(\*\*)?[[:space:]]*:'

# qa_citation_tier_is_strict <sha>
#   Exit 0 when <sha> is strictly AFTER the citation baseline (STRICT tier),
#   1 when it is the baseline itself or an ancestor of it (LEGACY tier).
#   FAILS CLOSED: an unresolvable baseline yields STRICT for everything.
qa_citation_tier_is_strict() {
	[ -n "$CITATION_BASELINE_RESOLVED" ] || return 0
	c="$(git rev-parse "$1" 2>/dev/null || true)"
	[ -n "$c" ] || return 0
	[ "$c" = "$CITATION_BASELINE_RESOLVED" ] && return 1
	if git merge-base --is-ancestor "$CITATION_BASELINE_RESOLVED" "$c" 2>/dev/null; then
		return 0
	fi
	return 1
}

# --------- Enforcing-mode preconditions ---------
if [ "$MODE_ENFORCE" -eq 1 ]; then
	if [ -z "$SINCE_REF" ]; then
		echo "verify_qa_evidence.sh: --enforce requires --since <ref-or-date>." >&2
		echo "  Baseline scoping is mandatory: enforcing over the whole history would" >&2
		echo "  block on thousands of pre-convention legacy commits. Use the commit" >&2
		echo "  that added docs/qa/README.md as the baseline (see script header)." >&2
		exit 2
	fi
	# Validate that --since resolves to something git understands. A SHA / tag
	# resolves via rev-parse; an approxidate (e.g. "2026-05-28") does not, so
	# fall back to a since-date probe with rev-list.
	if ! git rev-parse --verify --quiet "${SINCE_REF}^{commit}" >/dev/null 2>&1; then
		if ! git rev-list -1 --since="$SINCE_REF" HEAD >/dev/null 2>&1; then
			echo "verify_qa_evidence.sh: --since value '$SINCE_REF' is not a valid git ref or date." >&2
			exit 2
		fi
	fi
fi

# --------- Build the set of existing run-id directories ---------
# Feeds the LEGACY subject-substring fallback (match RULE 2) ONLY. The
# EXACT rule (RULE 1) needs no inventory — it reads the commit's own tree.
# Deliberately NOT extended with flat docs/qa/*.md basenames: adding those
# here would create a NEW guess-path (a subject merely mentioning
# "RETRO_LEDGER" would pass). Flat-file evidence is handled exactly, by
# RULE 1, from the commit's own diff.
existing_runids=""
if [ -d "$QA_DIR" ]; then
	for d in "$QA_DIR"/*/; do
		[ -d "$d" ] || continue
		rid="$(basename "$d")"
		existing_runids="${existing_runids} ${rid}"
	done
fi

# qa_evidence_added_by_commit <sha>
#   RULE 1 (EXACT, unfakeable). Prints the docs/qa/ evidence entry that
#   this commit ITSELF added/changed, or nothing.
#   - Reads the commit's own tree (`git diff-tree`), never its subject text.
#   - Handles both evidence shapes: a run-id directory (docs/qa/<run-id>/…)
#     collapses to <run-id>; a flat transcript (docs/qa/<name>.md) collapses
#     to <name>.md.
#   - Requires the entry to STILL EXIST on disk, so evidence that was
#     committed and later deleted does not keep passing the gate. Collapsing
#     to the top-level entry (not the full path) tolerates renames INSIDE a
#     run-id directory while still catching wholesale deletion.
#   - docs/qa/README.md is the convention doc, not evidence — excluded.
qa_evidence_added_by_commit() {
	git diff-tree --no-commit-id --name-only -r "$1" 2>/dev/null \
		| grep '^docs/qa/' \
		| grep -v '^docs/qa/README\.md$' \
		| while IFS= read -r qa_path; do
			entry="${qa_path#docs/qa/}"   # <run-id>/<file…>  or  <name>.md
			entry="${entry%%/*}"          # -> <run-id>        or  <name>.md
			[ -n "$entry" ] || continue
			if [ -e "$QA_DIR/$entry" ]; then
				echo "$entry"
				break
			fi
		done \
		| head -1
}

# qa_evidence_citing_commit <sha>
#   RULE 3 (CITATION, proof). Prints the docs/qa/ evidence file whose TRACKED
#   content cites this commit's SHA — path relative to docs/qa/, e.g.
#   "hxc119_20260712T193000Z/EVIDENCE.md" — or nothing.
#   - Uses `git grep`, which searches TRACKED files ONLY: an untracked file
#     dropped into docs/qa/ can NOT clear the gate.
#   - Tries the full 40-char SHA first, then the abbreviated form; never a
#     needle shorter than 8 hex chars (collision floor).
#   - docs/qa/README.md is the convention doc, not evidence — excluded.
#   - Prints the citing FILE (not just the run-id) so a reviewer can see at a
#     glance whether the citation came from a per-feature transcript or from
#     a blanket ledger (the documented residual risk in the header).
# qa_evidence_declared_citation <sha>
#   RULE 3 / STRICT tier (HXC-186). Prints the docs/qa/ evidence file that
#   DECLARES this commit — i.e. carries the SHA on the same line as a label
#   from $QA_CITATION_LABEL_RE — or nothing.
#   - `git grep` again, so TRACKED files only: an untracked file dropped into
#     docs/qa/ still cannot clear the gate.
#   - A machine-written provenance stamp (grounding.txt's "## repo HEAD"
#     line, a scratch-checkout path, a nonce, a build log) carries no label
#     and therefore no longer matches. That is the whole point of the tier.
qa_evidence_declared_citation() {
	sha_full="$(git rev-parse "$1" 2>/dev/null || true)"
	[ "${#sha_full}" -ge 8 ] || return 0
	needle="${sha_full:0:8}"
	# Cheap prefilter: only files mentioning the SHA at all can declare it.
	candidates="$(git grep -l -F "$needle" -- "$QA_DIR" ":!$QA_DIR/README.md" 2>/dev/null || true)"
	[ -n "$candidates" ] || return 0
	printf '%s\n' "$candidates" \
		| while IFS= read -r f; do
			[ -n "$f" ] || continue
			# Collect the file's DECLARATION lines, then require the needle to
			# be on one of them. Captures stdout, never an exit code after a
			# pipe (§11.4.6).
			decl_lines="$(git grep -h -i -E "$QA_CITATION_LABEL_RE" -- "$f" 2>/dev/null || true)"
			case "$decl_lines" in
				*"$needle"*) echo "${f#"$QA_DIR"/}"; break ;;
			esac
		done \
		| head -1
}

qa_evidence_citing_commit() {
	sha_full="$(git rev-parse "$1" 2>/dev/null || true)"
	# The needle is the FIRST 8 CHARS of the full SHA — deliberately NOT
	# `git rev-parse --short`, whose width is repo-dependent (git auto-scales
	# it with object count; core.abbrev can override it). Because every
	# abbreviation of a SHA is a PREFIX of it, this single 8-char needle
	# substring-matches a citation of ANY length >= 8, including the full
	# 40-char form. Using --short instead silently stopped matching in a repo
	# that abbreviates to 7 (caught by the paired fixture mutation, §1.1).
	# 8 hex chars is the collision floor: shorter citations are NOT honoured.
	[ "${#sha_full}" -ge 8 ] || return 0
	needle="${sha_full:0:8}"
	hit="$(git grep -l -F "$needle" -- "$QA_DIR" ":!$QA_DIR/README.md" 2>/dev/null | head -1)"
	if [ -n "$hit" ]; then
		echo "${hit#"$QA_DIR"/}"
	fi
	return 0
}

# --------- Determine the commit window ---------
# Advisory  : last N commits (default 20) reachable from HEAD.
# Enforcing : commits in <since>..HEAD (descendants of the baseline), so the
#             baseline commit and everything before it is exempt by
#             merge-ancestry — NOT by author-date sorting. N optionally caps.
if [ "$MODE_ENFORCE" -eq 1 ]; then
	# Resolve the range. If --since is a ref, use range syntax; if it is a
	# date, use --since=. Prefer ref-range (ancestry-accurate) when possible.
	if git rev-parse --verify --quiet "${SINCE_REF}^{commit}" >/dev/null 2>&1; then
		if [ -n "$SCAN_N" ]; then
			commits="$(git rev-list --max-count="$SCAN_N" "${SINCE_REF}..HEAD" 2>/dev/null || true)"
		else
			commits="$(git rev-list "${SINCE_REF}..HEAD" 2>/dev/null || true)"
		fi
	else
		if [ -n "$SCAN_N" ]; then
			commits="$(git rev-list --max-count="$SCAN_N" --since="$SINCE_REF" HEAD 2>/dev/null || true)"
		else
			commits="$(git rev-list --since="$SINCE_REF" HEAD 2>/dev/null || true)"
		fi
	fi
else
	commits="$(git rev-list --max-count="${SCAN_N:-20}" HEAD 2>/dev/null || true)"
fi

# --------- Report header ---------
echo "==========================================================================="
if [ "$MODE_ENFORCE" -eq 1 ]; then
	echo " §11.4.83 docs/qa/ end-user evidence — ENFORCING release gate"
	echo "==========================================================================="
	echo " Scope     : ${SINCE_REF}..HEAD  (commits before the baseline are exempt)"
	echo " Opt-out   : commits whose message contains [no-qa-evidence] are exempt"
	echo " Exit      : 0 when clean, 1 on any violation (release-gate blocking)."
else
	echo " §11.4.83 docs/qa/ end-user evidence — ADVISORY scan (warn-mode, exit 0)"
	echo "==========================================================================="
	echo " Scanning the last ${SCAN_N:-20} commit(s) for feature-shipping commits"
	echo " lacking a matching docs/qa/<run-id>/ directory."
	echo " NOTE: advisory only — never blocks. Run with --enforce --since <ref>"
	echo "       for the §11.4.83 operative-rule-(5) blocking release gate."
fi
echo "---------------------------------------------------------------------------"

violation_count=0
feature_commit_count=0
exempt_count=0

for sha in $commits; do
	# Feature-shipping heuristic: non-test prod code under the watched roots.
	# Use diff-tree plumbing (robust across git versions; `git show
	# --no-patch --name-only` errors in modern git as -s conflicts with
	# --name-only).
	touched="$(git diff-tree --no-commit-id --name-only -r "$sha" 2>/dev/null \
		| grep -E '^helix_code/(internal|cmd|applications)/' \
		| grep -v '_test\.go$' || true)"
	[ -n "$touched" ] || continue

	short="$(git rev-parse --short "$sha" 2>/dev/null || echo "$sha")"
	subject="$(git show --no-patch --format='%s' "$sha" 2>/dev/null || true)"

	# Per-commit opt-out: [no-qa-evidence] anywhere in subject OR body.
	full_msg="$(git show --no-patch --format='%B' "$sha" 2>/dev/null || true)"
	case "$full_msg" in
		*'[no-qa-evidence]'*)
			exempt_count=$((exempt_count + 1))
			echo "  exempt ${short}  ${subject}"
			echo "         -> [no-qa-evidence] opt-out token present"
			continue
			;;
	esac

	feature_commit_count=$((feature_commit_count + 1))

	# RULE 1 (EXACT): did the commit add its own docs/qa/ evidence path?
	matched="$(qa_evidence_added_by_commit "$sha")"
	match_kind="added by this commit (exact)"

	# RULE 3 (CITATION): is there a TRACKED docs/qa evidence file that cites
	# this commit's SHA? Catches the normal two-commit workflow where the
	# feature commit ships code and its paired close commit lands the
	# evidence. Proof, not a guess — the evidence names the commit.
	if [ -z "$matched" ]; then
		if qa_citation_tier_is_strict "$sha"; then
			# STRICT tier (HXC-186): only an explicit DECLARATION counts. A
			# provenance stamp that merely happens to carry this SHA does not.
			matched="$(qa_evidence_declared_citation "$sha")"
			[ -n "$matched" ] && match_kind="declares this commit in a citation field (exact)"
		else
			# LEGACY tier: pre-baseline substring match, labelled as such so it
			# is never read as a declaration.
			matched="$(qa_evidence_citing_commit "$sha")"
			[ -n "$matched" ] && match_kind="cites this commit's SHA (legacy substring, pre-baseline — not coincidence-proof)"
		fi
	fi

	# RULE 2 (LEGACY heuristic): does the subject name an existing run-id
	# directory? Only consulted when RULES 1 and 3 found nothing. This is a
	# GUESS, not proof — reported as such so it is never mistaken for evidence.
	if [ -z "$matched" ]; then
		match_kind="named in commit subject (legacy heuristic — not proof)"
		for rid in $existing_runids; do
			case "$subject" in
				*"$rid"*) matched="$rid"; break ;;
			esac
		done
	fi

	if [ -n "$matched" ]; then
		echo "  ok     ${short}  ${subject}"
		echo "         -> docs/qa/${matched} present (${match_kind})"
	else
		violation_count=$((violation_count + 1))
		if [ "$MODE_ENFORCE" -eq 1 ]; then
			echo "  VIOL   ${short}  ${subject}" >&2
			echo "         -> no docs/qa evidence: commit adds none, none cites its SHA, subject names no run-id" >&2
		else
			echo "  WARN   ${short}  ${subject}"
			echo "         -> no docs/qa evidence: commit adds none, none cites its SHA, subject names no run-id"
		fi
	fi
done

echo "---------------------------------------------------------------------------"
echo " Feature-shipping commits evaluated : ${feature_commit_count}"
echo " Opt-out exempt commits             : ${exempt_count}"
if [ "$MODE_ENFORCE" -eq 1 ]; then
	echo " Violations                         : ${violation_count}"
else
	echo " Advisory warnings                  : ${violation_count}"
fi

if [ "$MODE_ENFORCE" -eq 1 ]; then
	if [ "$violation_count" -gt 0 ]; then
		echo >&2
		echo " RELEASE-GATE FAIL: the commit(s) above ship a feature but carry no" >&2
		echo " matching docs/qa/<run-id>/ evidence directory. Per §11.4.83 each" >&2
		echo " shipped feature MUST carry a recorded end-to-end transcript +" >&2
		echo " materials under docs/qa/<run-id>/ (see docs/qa/README.md). Either" >&2
		echo " add the evidence directory, or — for a non-feature change that" >&2
		echo " tripped the heuristic — annotate the commit with [no-qa-evidence]." >&2
		echo "===========================================================================" >&2
		echo " RESULT: FAIL (enforcing — ${violation_count} violation(s))." >&2
		exit 1
	fi
	echo "==========================================================================="
	echo " RESULT: PASS (enforcing — no violations in ${SINCE_REF}..HEAD)."
	exit 0
fi

if [ "$violation_count" -gt 0 ]; then
	echo
	echo " ADVISORY: the commits above appear to ship a feature but have no"
	echo " matching docs/qa/<run-id>/ evidence directory. Per §11.4.83 each"
	echo " shipped feature SHOULD carry a recorded end-to-end transcript +"
	echo " materials. Add one under docs/qa/<run-id>/ (see docs/qa/README.md)."
	echo " This is a notice only — no action is enforced. Use --enforce --since"
	echo " <ref> for the blocking release gate."
fi
echo "==========================================================================="
echo " RESULT: advisory scan complete (exit 0 — warn-mode)."
exit 0
