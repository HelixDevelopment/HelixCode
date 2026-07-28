#!/usr/bin/env bash
# fixed_h2_pipe_row_parity_gate.sh — §11.4.135 standing regression guard for the
# §11.4.90/.91/.53 docs-tooling drift that hid HXC-044 from Fixed_Summary.
#
# GATE ID: CM-FIXED-SUMMARY-ITEM-VISIBILITY
# (The FILE name still says "h2_pipe_row_parity" — the mechanism this guard used
# BEFORE the 2026-07-29 reconciliation below. The path is retained deliberately:
# docs/Issues.md, docs/Issues_Summary.md and docs/scripts/ reference it, and
# tracker-doc regeneration is operator-deferred, so a rename would strand those
# references. Renaming the file is tracked as follow-up, not done silently.)
#
# ===========================================================================
# §11.4.120 RECONCILIATION RECORD (2026-07-29, HXC-161) — READ BEFORE EDITING
# ===========================================================================
# THE ORIGINAL DRIFT (forensic, FACT, 2026-06-16): docs/Fixed.md is MIXED — a
# pipe table (`| Closure | Title | Type | Status | Round | Commit(s) | Evidence |`)
# AND H2 detail sections (`## HXC/ATM-NNN — …`). The THEN-canonical summary
# generator, scripts/generate_fixed_summary.sh, read ONLY the pipe table. So an
# H2 closure section with NO matching pipe-table row was invisible to the
# summary (HXC-044 was Obsolete in the DB + had an H2 section but no pipe row →
# absent from Fixed_Summary twice over).
#
# THIS GUARD ORIGINALLY asserted "every H2 closure heading has a pipe-table row"
# as a PROXY for the thing that actually mattered: "no closed item is missing
# from Fixed_Summary.md". Under the pipe-table-reading generator the proxy was
# sound — having a pipe row was exactly what made an item visible.
#
# WHAT CHANGED (the consumer, not the data). Commit 8494380a (2026-07-29):
#   - scripts/generate_fixed_summary.sh is SUPERSEDED and now REFUSES to run
#     (exits 2 in every mode, writes nothing) after a hand invocation at
#     0a4df699 rewrote the closed-item tally 344 -> 188, dropping 156 items.
#   - docs/Fixed_Summary.md is now DERIVED FROM docs/workable_items.db by the
#     constitution submodule's Go exporter (renderFixedSummary, export.go),
#     which selects every item with current_location=='Fixed' and is
#     REPRESENTATION-AGNOSTIC — a 'section' item and a 'table' item land in the
#     summary identically.
#
# THEREFORE the proxy no longer tracks the harm. Measured at HEAD (bbd236c9):
# 56 H2 closure sections have no pipe row, and ALL 56 are present in
# Fixed_Summary.md — the invisibility the proxy existed to prevent does not
# occur. The pipe-row requirement had become a standing 57-failure red that
# asserted an obsolete era's mechanism (§11.4.120: a gate that FAILs because a
# correct change removed the old mechanism must be RECONCILED to the NEW
# mechanism — never fake-passed, never reverted-against).
#
# WHY THIS IS A STRENGTHENING, NOT A LOOSENING (the §11.4.120 discriminator).
# Run against 0a4df699 — the REAL data-loss commit, where Fixed_Summary.md was
# corrupted down to 20 lines:
#   - the OLD invariant produced 0 summary-visibility failures. It never
#     noticed. (It exited 1, but only on pipe-row-parity noise — red for an
#     unrelated reason, which is precisely how a 57-failure standing red masks
#     a real one.)
#   - THIS invariant flags 148 of 149 H2 sections as absent from the summary.
# The reconciled gate catches the real incident the retired one missed.
#
# ===========================================================================
# WHAT THIS GUARD ASSERTS NOW — summary VISIBILITY, the purpose the retired
# pipe-row proxy stood in for. Fixed.md and Fixed_Summary.md are rendered by
# DIFFERENT code paths (doc_segments walk vs. renderFixedSummary item select),
# so agreement between them is a real, falsifiable cross-check — not a
# tautology, as 0a4df699 proved by breaking it.
#   (A) every docs/Fixed.md H2 closure heading (## HXC-NNN / ## ATM-NNN)
#       appears in docs/Fixed_Summary.md;
#   (B) every docs/Fixed.md pipe-table row whose Title cell carries a ticket id
#       appears in docs/Fixed_Summary.md;
#   (C) every `Obsolete (→ Fixed.md)` pipe-table item appears in
#       docs/Fixed_Summary.md — retained as an explicitly-named check because
#       it is the original HXC-044 forensic anchor (a strict subset of (B); it
#       stays named so an Obsolete-specific regression is reported as such).
#
# HONEST BOUNDARY (§11.4.6 / §11.4.118 — stated, never silently skipped):
#   - Ticket-id matching is bounded (non-alphanumeric on both sides) so ATM-97
#     does not spuriously satisfy itself via ATM-970.
#   - Legacy pipe rows whose Title cell carries NO ticket id (e.g. the
#     2026-05-19 i18n-round rows, "Tracker HTML + PDF exports per §11.4.19")
#     cannot be keyed by id and are OUT of scope for (B). Their count is
#     REPORTED on every run so the exclusion is visible, not hidden.
#   - This is a DOCUMENT-PARITY guard (Fixed.md ↔ Fixed_Summary.md) and reads
#     no database: it proves the two rendered documents agree, NOT that either
#     matches the SQLite SSoT. That md⟷db direction is CM-WORKABLE-ITEMS-MD-DB-
#     IN-SYNC + CM-SUMMARY-SYNC's job, and both remain required.
#
# Purpose / Usage / Inputs / Outputs / Side-effects / Dependencies / Cross-refs:
#   Purpose:      §11.4.135 regression guard (drift that hid HXC-044).
#   Usage:        scripts/gates/fixed_h2_pipe_row_parity_gate.sh
#                 RED_MODE=1 scripts/gates/fixed_h2_pipe_row_parity_gate.sh
#                            # §11.4.115 polarity: reproduce the defect on a
#                            # synthesized broken copy (an item deleted from
#                            # Fixed_Summary.md) and assert the guard FAILs.
#                            # Default RED_MODE=0 = standing GREEN guard.
#   Inputs:       docs/Fixed.md, docs/Fixed_Summary.md
#   Outputs:      PASS/FAIL line on stdout; exit 0 PASS, exit 1 FAIL.
#   Side-effects: none (RED_MODE uses temp copies under $TMPDIR; cleaned on EXIT).
#   Dependencies: awk, grep, mktemp.
#   Cross-refs:   §11.4.53 (Fixed_Summary parity), §11.4.90 (Obsolete status),
#                 §11.4.91 (summary clarity), §11.4.115 (RED polarity),
#                 §11.4.120 (reconcile, never fake-pass), §11.4.135 (standing
#                 regression guard), §11.4.124 (retained-not-removed).
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
FIXED="$ROOT/docs/Fixed.md"
SUMMARY="$ROOT/docs/Fixed_Summary.md"
GATE="CM-FIXED-SUMMARY-ITEM-VISIBILITY"
RED_MODE="${RED_MODE:-0}"

# --- RED_MODE: §11.4.115 reproduce-the-defect-on-a-broken-artifact ----------
# Synthesize the defect on COPIES — delete items from Fixed_Summary.md — run the
# same checker against them, and assert it FAILs. One source, two roles: the
# bug-catcher IS the regression guard. Proves the guard is not a blind test.
# Two victims are removed, covering BOTH representations, so a future edit that
# silently narrows the guard to one of them is caught here:
#   - HXC-044: the original forensic anchor (Obsolete, has a pipe row) -> (B)+(C)
#   - HXC-157: a section-only closure (H2, NO pipe row)                -> (A)
# HXC-157 is the load-bearing one: under the RETIRED pipe-row invariant its
# disappearance from the summary was completely invisible.
if [[ "$RED_MODE" == "1" ]]; then
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT
  cp "$FIXED" "$TMP/Fixed.md"
  grep -vE 'HXC-044|HXC-157' "$SUMMARY" > "$TMP/Fixed_Summary.md" || true
  if RED_MODE=0 FIXED_OVERRIDE="$TMP/Fixed.md" SUMMARY_OVERRIDE="$TMP/Fixed_Summary.md" \
       "$0" >/dev/null 2>&1; then
    echo "$GATE: RED FAIL — guard PASSed on the known-broken artifact (blind test)" >&2
    exit 1
  fi
  echo "$GATE: RED OK — guard correctly FAILs on the broken artifact (HXC-044 Obsolete pipe-row item AND HXC-157 section-only closure both deleted from Fixed_Summary.md — the exact §11.4.90/.53 data-loss shape that 0a4df699 shipped)"
  exit 0
fi

# Allow RED_MODE's recursive invocation to point at the synthesized copies.
FIXED="${FIXED_OVERRIDE:-$FIXED}"
SUMMARY="${SUMMARY_OVERRIDE:-$SUMMARY}"

[[ -f "$FIXED" ]]   || { echo "$GATE: FAIL — $FIXED missing" >&2; exit 1; }
[[ -f "$SUMMARY" ]] || { echo "$GATE: FAIL — $SUMMARY missing" >&2; exit 1; }

FAILURES=0

# in_summary <ticket-id> — bounded match so ATM-97 is NOT satisfied by ATM-970.
in_summary() {
  grep -qE "(^|[^A-Za-z0-9_-])$1([^A-Za-z0-9_-]|\$)" "$SUMMARY"
}

# (A) Every H2 closure section is visible in Fixed_Summary.md.
A_SEEN=0
while IFS= read -r id; do
  [[ -z "$id" ]] && continue
  A_SEEN=$((A_SEEN + 1))
  if ! in_summary "$id"; then
    echo "$GATE: FAIL — Fixed.md H2 closure section '## $id' is MISSING from Fixed_Summary.md (§11.4.53/.135)" >&2
    FAILURES=$((FAILURES + 1))
  fi
done < <(grep -oE '^## (HXC|ATM)-[0-9A-Za-z]+' "$FIXED" | sed 's/^## //' | sort -u)

# (B) Every ticket-id-titled pipe-table row is visible in Fixed_Summary.md.
B_SEEN=0
while IFS= read -r id; do
  [[ -z "$id" ]] && continue
  B_SEEN=$((B_SEEN + 1))
  if ! in_summary "$id"; then
    echo "$GATE: FAIL — Fixed.md pipe-table row '$id' is MISSING from Fixed_Summary.md (§11.4.53/.135)" >&2
    FAILURES=$((FAILURES + 1))
  fi
done < <(awk -F'|' '
    /^\|/ {
      c1=$2; gsub(/^[ \t]+|[ \t]+$/, "", c1)
      if (c1 !~ /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/) next
      title=$3; gsub(/^[ \t]+|[ \t]+$/, "", title)
      if (match(title, /^(HXC|ATM)-[0-9A-Za-z]+/)) print substr(title, RSTART, RLENGTH)
    }
  ' "$FIXED" | sort -u)

# (C) Every Obsolete (→ Fixed.md) pipe-table item is visible in Fixed_Summary.md
#     — the original HXC-044 forensic anchor, kept explicitly named.
C_SEEN=0
while IFS= read -r id; do
  [[ -z "$id" ]] && continue
  C_SEEN=$((C_SEEN + 1))
  if ! in_summary "$id"; then
    echo "$GATE: FAIL — Obsolete item '$id' missing from Fixed_Summary.md (§11.4.90/.53)" >&2
    FAILURES=$((FAILURES + 1))
  fi
done < <(awk -F'|' '
    /^\|/ {
      c1=$2; gsub(/^[ \t]+|[ \t]+$/, "", c1)
      if (c1 !~ /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/) next
      s=$5; gsub(/^[ \t]+|[ \t]+$/, "", s)
      if (s !~ /^Obsolete \(/) next
      title=$3; gsub(/^[ \t]+|[ \t]+$/, "", title)
      if (match(title, /^(HXC|ATM)-[0-9A-Za-z]+/)) print substr(title, RSTART, RLENGTH)
    }
  ' "$FIXED" | sort -u)

# Out-of-scope census (§11.4.118 — the exclusion is reported, never hidden).
UNKEYED="$(awk -F'|' '
    /^\|/ {
      c1=$2; gsub(/^[ \t]+|[ \t]+$/, "", c1)
      if (c1 !~ /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/) next
      title=$3; gsub(/^[ \t]+|[ \t]+$/, "", title)
      if (title !~ /^(HXC|ATM)-[0-9A-Za-z]+/) n++
    }
    END { print n+0 }
  ' "$FIXED")"

# Anti-tautology tripwire: if NOTHING was checked, the extractors have silently
# stopped matching (a format change) and a PASS would be vacuous. Refuse it.
if [[ "$A_SEEN" -eq 0 && "$B_SEEN" -eq 0 ]]; then
  echo "$GATE: FAIL — extracted 0 H2 closure sections AND 0 ticket-id pipe rows from $FIXED; a PASS here would be vacuous (format drift?)" >&2
  exit 1
fi

echo "$GATE: checked $A_SEEN H2 closure section(s), $B_SEEN ticket-id pipe row(s), $C_SEEN Obsolete item(s) against Fixed_Summary.md; $UNKEYED legacy pipe row(s) carry no ticket id and are out of scope (§11.4.118)"

if [[ "$FAILURES" -gt 0 ]]; then
  echo "$GATE: FAIL — $FAILURES closed item(s) missing from Fixed_Summary.md" >&2
  exit 1
fi
echo "$GATE: PASS — every Fixed.md closure (H2 section AND ticket-id pipe row, incl. every Obsolete item) is present in Fixed_Summary.md"
exit 0
