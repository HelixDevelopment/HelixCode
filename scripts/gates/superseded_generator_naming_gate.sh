#!/usr/bin/env bash
# scripts/gates/superseded_generator_naming_gate.sh — CM-SUPERSEDED-GENERATOR-NAMING
#
# §11.4.91 + §11.4.93 + §11.4.95 + §11.4.120 + §11.4.157 standing regression
# guard for HXC-160.
#
# ---------------------------------------------------------------------------
# WHAT THIS ASSERTS
# ---------------------------------------------------------------------------
# docs/Issues_Summary.md and docs/Fixed_Summary.md are DERIVED FROM the tracked
# SQLite single source of truth docs/workable_items.db (§11.4.93 / §11.4.95) by
# the constitution submodule's Go exporter:
#
#   go run -C constitution/scripts/workable-items ./cmd/workable-items \
#       export --db "$PWD/docs/workable_items.db" --out-dir "$PWD/docs"
#
#   (cmd/workable-items/export.go: renderIssuesSummary / renderFixedSummary)
#
#   HXC-201: --db/--out-dir MUST be ABSOLUTE. `go run -C <dir>` changes the
#   BUILT BINARY's own working directory to <dir> (not just the build step),
#   so a relative --db/--out-dir resolves against constitution/scripts/
#   workable-items/ instead of the caller's real docs/ tree — a run that
#   prints success-looking "wrote docs/Issues.md" lines while writing nothing
#   useful to the real project and silently materialising a fresh, zero-row
#   docs/workable_items.db inside the tool's own directory. As a second,
#   independent layer of defense the exporter itself (export.go / db.go)
#   anchors a relative --db/--out-dir against the invoking shell's directory
#   via $PWD when it is set — but the documented invocation MUST still pass
#   absolute paths; never rely on that fallback alone.
#
# The legacy shell generators scripts/generate_issues_summary.sh and
# scripts/generate_fixed_summary.sh derive the summaries from the MARKDOWN
# trackers instead — the exact INVERSE of the §11.4.93 db-to-md direction — and
# their awk predicate matches only pipe rows whose column 1 is a date, so every
# item carried as an H2 section is INVISIBLE to them.
#
# MEASURED DATA LOSS (2026-07-29, commit 0a4df699): a hand invocation rewrote
# the closed-item tally 344 -> 188 (156 items of tracked state dropped),
# corrupted every per-type count (Bug 166->79, Feature 95->79, Task 83->30,
# Obsolete 3->1), replaced the per-item table with bare aggregates, and the
# false counts were exported to .html/.pdf/.docx before CM-FIXED-SUMMARY-SYNC
# caught them.
#
# The scripts now refuse to run (exit 2). This gate closes the REMAINING half of
# HXC-160: the governance manuals must never again PRESENT them as the current
# mechanism. Comment-only de-wiring is vigilance, not enforcement (§11.4.75) —
# a manual that silently reintroduces the stale naming must FAIL a gate.
#
# ---------------------------------------------------------------------------
# THE RULE
# ---------------------------------------------------------------------------
# Every LINE in a governed carrier that names `generate_issues_summary.sh` or
# `generate_fixed_summary.sh` MUST also carry a supersession marker on that same
# line. Naming the scripts is NOT forbidden — deleting them from the record
# would destroy the audit trail (§11.4.124) — but naming them WITHOUT marking
# them superseded is.
#
# Accepted supersession markers (case-insensitive): supersed(e|ed|es|ing),
# refuse(s|d), legacy, historical, removed, unwired, deprecated, "WAS:".
#
# §11.4.157 five-carrier lockstep is asserted separately: every root carrier
# present MUST agree on the corrected-occurrence count, so a fix that lands in
# CLAUDE.md but not GEMINI.md FAILs.
#
# Paired §1.1 mutation: strip the supersession marker from any governed line
# (e.g. `sed -i 's/SUPERSEDES/uses/' CLAUDE.md`) and this gate MUST FAIL.
#
# Exit codes: 0 = pass, 1 = violation.
# ---------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

GATE="CM-SUPERSEDED-GENERATOR-NAMING"
STALE_RE='generate_issues_summary\.sh|generate_fixed_summary\.sh'
MARKER_RE='supersed|refuse|legacy|historical|removed|unwired|deprecated|WAS:'

# Governed carriers: the parent repo's own governance manuals. Owned-submodule
# copies are NOT in scope here — those are cascaded restatements of canonical
# constitution text and are governed by the CONST-047 cascade (see HXC-160
# report: 129 submodule files enumerated, deferred pending a constitution-canon
# push per CONST-049).
CARRIERS=(
    CLAUDE.md AGENTS.md QWEN.md GEMINI.md CONSTITUTION.md CRUSH.md
    helix_code/CLAUDE.md helix_code/AGENTS.md helix_code/CONSTITUTION.md
    github_pages_website/CLAUDE.md
    github_pages_website/AGENTS.md
    github_pages_website/CONSTITUTION.md
)

violations=0
checked=0

for f in "${CARRIERS[@]}"; do
    [[ -f "$f" ]] || continue
    while IFS=: read -r lineno text; do
        [[ -n "$lineno" ]] || continue
        if grep -qiE "$MARKER_RE" <<<"$text"; then
            checked=$((checked + 1))
        else
            echo "FAIL $GATE  $f:$lineno names a superseded shell summary generator with no supersession marker" >&2
            echo "     line: $(cut -c1-120 <<<"$text")..." >&2
            violations=$((violations + 1))
        fi
    done < <(grep -nE "$STALE_RE" "$f" 2>/dev/null)
done

# --- §11.4.157 five-carrier lockstep -------------------------------------
# Every root carrier that exists must agree on how many corrected occurrences
# it carries. A partial fix (landed in CLAUDE.md, missed in GEMINI.md) is itself
# a violation — GEMINI.md has silently drifted 14 mandates behind before.
declare -A COUNTS=()
lockstep_ref=""
for f in CLAUDE.md AGENTS.md QWEN.md GEMINI.md CONSTITUTION.md; do
    [[ -f "$f" ]] || continue
    # grep -c already prints 0 on no-match (and exits 1); do NOT append another.
    n="$(grep -cE "$STALE_RE" "$f" 2>/dev/null)" || true
    n="${n:-0}"
    COUNTS["$f"]="$n"
    [[ -z "$lockstep_ref" ]] && lockstep_ref="$n"
done
for f in "${!COUNTS[@]}"; do
    if [[ "${COUNTS[$f]}" != "$lockstep_ref" ]]; then
        echo "FAIL $GATE  §11.4.157 lockstep break: $f has ${COUNTS[$f]} occurrence(s), expected $lockstep_ref (matching its sibling carriers)" >&2
        violations=$((violations + 1))
    fi
done

echo "$GATE: ${checked} correctly-marked reference(s) across ${#CARRIERS[@]} governed carrier(s), ${violations} violation(s)"
[[ "$violations" -gt 0 ]] && exit 1
exit 0
