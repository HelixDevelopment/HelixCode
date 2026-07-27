#!/usr/bin/env bash
# obsolete_details_gate.sh — Constitution §11.4.90 gate.
#
# Walks docs/Issues.md + docs/Fixed.md. For every "## " heading whose
# **Status:** is "Obsolete (→ Fixed.md)", asserts that an
# "**Obsolete-Details:**" line exists within 8 non-blank lines of the heading,
# and that it carries all four mandated sub-facts:
#   - Since:           an ISO date (YYYY-MM-DD)
#   - Reason:          a value from the closed vocabulary
#   - Superseding-item: a non-empty reference
#   - Triple-check evidence: non-empty evidence
#
# Triple-check label reconciliation (§11.4.120, 2026-07-27):
#   The canonical sub-fact name in constitution/Constitution.md §11.4.90 is
#   "Triple-check evidence:", and the §11.4.93/95 SSoT renderer
#   (constitution/scripts/workable-items/cmd/workable-items/obsolete.go:187)
#   emits exactly `...; Triple-check evidence: <path>` when projecting
#   docs/workable_items.db → docs/Fixed.md. This gate previously asserted only
#   the narrower literal `Triple-check:`, so it FALSE-POSITIVE-refused every
#   item written by the canonical renderer (a §11.4.201 false-positive
#   refusal). Hand-editing the docs to the narrow label was NOT an option: it
#   would contradict the canonical constitution wording AND break G11
#   (CM-WORKABLE-ITEMS-MD-DB-IN-SYNC), whose md→db→md round-trip regenerates
#   the renderer's label byte-identically. The gate is therefore reconciled to
#   the mechanism the fix introduced: BOTH `Triple-check:` (legacy
#   hand-authored) and `Triple-check evidence:` (canonical renderer) are
#   accepted, and BOTH still require a NON-EMPTY value — this is a label
#   widening, never a tautology. Paired §1.1 mutation in
#   scripts/tests/obsolete_details_meta_test.sh asserts an EMPTY
#   `Triple-check evidence:` still FAILs.
#
# Closed Reason vocabulary (§11.4.90):
#   superseded-by-design-change | superseded-by-later-mandate |
#   feature-removed | duplicate-of | unsupported-topology | not-reproducible
# (`not-reproducible` = a reported defect that does NOT reproduce on the
#  canonical tree/baseline — an environment/isolated-worktree artifact, per
#  the §11.4.90 closed vocabulary; the gate's list MUST mirror that source.)
#
# Exit: 0 if every Obsolete item is compliant (or none present),
#       1 on any violation, 2 on usage / environment error.
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
DOCS="${1:-$ROOT/docs}"

FILES=("$DOCS/Issues.md" "$DOCS/Fixed.md")

OBSOLETE_STATUS='Obsolete (→ Fixed.md)'
# Pipe-separated closed vocabulary for awk regex matching.
REASON_VOCAB='superseded-by-design-change|superseded-by-later-mandate|feature-removed|duplicate-of|unsupported-topology|not-reproducible'

violations=0
checked=0

for f in "${FILES[@]}"; do
    if [[ ! -f "$f" ]]; then
        echo "SKIP $f — not present"
        continue
    fi

    # awk walks the file heading-by-heading. For each "## " heading it records
    # the heading line, then scans the following lines: capturing **Status:**,
    # and (within 8 non-blank lines) any **Obsolete-Details:** line. At the next
    # heading / EOF it evaluates the previous block.
    out="$(awk -v status="$OBSOLETE_STATUS" -v vocab="$REASON_VOCAB" -v file="$f" '
        function eval_block(   missing) {
            if (cur_heading == "") return
            if (cur_status != status) return
            # This is an Obsolete item — must have a compliant details line.
            if (details_line == "") {
                printf("VIOLATION\t%s\t%s\tno **Obsolete-Details:** line within 8 non-blank lines\n", file, cur_heading)
                viol++
                return
            }
            missing = ""
            if (details_line !~ /Since:[ ]*[0-9]{4}-[0-9]{2}-[0-9]{2}/)        missing = missing " Since(ISO-date)"
            if (details_line !~ ("Reason:[ ]*(" vocab ")"))                    missing = missing " Reason(closed-vocab)"
            if (details_line !~ /Superseding-item:[ ]*[^ ]/)                   missing = missing " Superseding-item"
            # §11.4.120 reconciliation: accept the canonical renderer label
            # "Triple-check evidence:" AND the legacy "Triple-check:"; both
            # still require a non-empty value (see header note).
            if (details_line !~ /Triple-check([ ]+evidence)?:[ ]*[^ ]/)         missing = missing " Triple-check(evidence)"
            if (missing != "") {
                printf("VIOLATION\t%s\t%s\tmissing/invalid sub-fact(s):%s\n", file, cur_heading, missing)
                viol++
            } else {
                printf("OK\t%s\t%s\n", file, cur_heading)
                ok++
            }
        }
        /^## / {
            eval_block()
            cur_heading = $0
            cur_status = ""
            details_line = ""
            nonblank = 0
            in_block = 1
            next
        }
        in_block == 1 {
            # Stop scanning sub-facts after 8 non-blank lines past the heading.
            if ($0 ~ /[^ \t]/) nonblank++
            if (cur_status == "" && $0 ~ /^\*\*Status:\*\*/) {
                cur_status = $0
                sub(/^\*\*Status:\*\*[ ]*/, "", cur_status)
            }
            if (details_line == "" && nonblank <= 8 && $0 ~ /^\*\*Obsolete-Details:\*\*/) {
                details_line = $0
            }
        }
        END {
            eval_block()
            printf("SUMMARY\t%d\t%d\n", ok+0, viol+0)
        }
    ' "$f")"

    while IFS=$'\t' read -r kind a b c; do
        case "$kind" in
            VIOLATION)
                echo "FAIL §11.4.90  [$a] $b — $c" >&2
                violations=$((violations + 1))
                ;;
            OK)
                checked=$((checked + 1))
                ;;
            SUMMARY)
                : # per-file counts folded into globals below
                ;;
        esac
    done <<< "$out"
done

echo "§11.4.90 obsolete-details gate: ${checked} compliant Obsolete item(s), ${violations} violation(s)"
if [[ "$violations" -gt 0 ]]; then
    exit 1
fi
exit 0
