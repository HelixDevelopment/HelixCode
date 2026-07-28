#!/usr/bin/env bash
# generate_fixed_summary.sh — mechanically regenerate docs/Fixed_Summary.md
# from the docs/Fixed.md closure table per Constitution §11.4.19 + §11.4.57
# (Type-aware closure vocabulary) + §11.4.91.
#
# Fixed.md is a pipe table:  | Closure | Title | Type | Status | Round | Commit(s) | Evidence |
# This generator counts closures by Type and emits the aggregate-counts block.
#
# Idempotent.
#
# Usage:
#   scripts/generate_fixed_summary.sh                 # write docs/Fixed_Summary.md (+status line)
#   scripts/generate_fixed_summary.sh --check         # verify docs/Fixed_Summary.md is in sync
#   scripts/generate_fixed_summary.sh --stdout        # emit ONLY the summary content to stdout
#                                                     #   (no status line, no file write)
#   scripts/generate_fixed_summary.sh --stdout <in...> <outfile> [extra...]
#                                                     #   docs_chain derive-node contract: write
#                                                     #   canonical content to <outfile> (the last
#                                                     #   positional), no status line, no side-effect
#                                                     #   write to docs/Fixed_Summary.md (§11.4.106).
#
# The docs_chain engine (internal/runner.execTransform) invokes an exec
# transform as `script <input_files...> <output_file> <extra_args...>` and reads
# the produced <output_file> back as the node content (it does NOT capture
# stdout). The default (no-flag) behavior — writing docs/Fixed_Summary.md and
# printing a "wrote ..." status line to stdout — is UNCHANGED for the existing
# script/gate callers; the docs_chain contract is honored only under --stdout.
set -euo pipefail

# ---------------------------------------------------------------------------
# SUPERSEDED — THIS SCRIPT REFUSES TO RUN (§11.4.106 / §11.4.93 / §11.4.95 /
# §11.4.75). The body below is retained UNCHANGED for forensics; this is a
# refusal guard, NOT a removal (§11.4.124). Retirement is tracked as HXC-160.
# ---------------------------------------------------------------------------
# docs/Fixed_Summary.md is DERIVED FROM docs/workable_items.db (the tracked
# SQLite single source of truth), NOT from docs/Fixed.md. The canonical
# generator is the constitution submodule's Go exporter (renderFixedSummary in
# cmd/workable-items/export.go).
#
# THIS script derives the summary from the MARKDOWN — the inverse of the
# §11.4.93 db-to-md direction — and its awk predicate matches only pipe rows
# whose column 1 is a date, so every item carried as an H2 section is INVISIBLE
# to it.
#
# MEASURED DATA LOSS (commit 0a4df699, 2026-07-29): a hand invocation of this
# script overwrote the DB-derived summary, rewriting the closed-item tally
# 344 -> 188 (156 items of tracked state dropped), corrupting every per-type
# count (Bug 166->79, Feature 95->79, Task 83->30, Obsolete 3->1) and replacing
# the full per-item table with aggregate counts only. The false counts were
# committed and exported to .html/.pdf/.docx before CM-FIXED-SUMMARY-SYNC
# caught them.
#
# .docs_chain/contexts/fixed.yaml de-wired this script as a docs_chain
# transform on 2026-07-28 and recorded "must not be re-attached" in a comment.
# A hand invocation re-broke the document the NEXT DAY. Comment-only de-wiring
# is vigilance, not enforcement (§11.4.75) — hence this in-script refusal.
cat >&2 <<'REFUSED'
generate_fixed_summary.sh: REFUSING TO RUN — superseded and data-destroying.

  docs/Fixed_Summary.md is derived from docs/workable_items.db, NOT docs/Fixed.md.
  This script sees only Fixed.md's pipe rows and silently drops every
  H2-section item (measured 2026-07-29: 344 closed items -> 188, all
  per-type counts corrupted, per-item table replaced by bare aggregates).

  Use the canonical DB exporter instead:

    (cd constitution/scripts/workable-items && go run ./cmd/workable-items) \
        export --db docs/workable_items.db --out-dir docs

  Then verify:  bash scripts/gates/summary_sync_gate.sh

  Retirement of this script is tracked as HXC-160.
REFUSED
exit 2

# --- Argument mode parsing -------------------------------------------------
# Recognize --check / --stdout anywhere in the args; collect the remaining
# positional args (docs_chain passes <input_files...> <output_file> [extra]).
MODE="file"          # file | check | stdout
POSITIONAL=()
for arg in "$@"; do
  case "$arg" in
    --check)  MODE="check"  ;;
    --stdout) MODE="stdout" ;;
    *)        POSITIONAL+=("$arg") ;;
  esac
done

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$ROOT/docs/Fixed.md"
OUT="$ROOT/docs/Fixed_Summary.md"
[[ -f "$SRC" ]] || { echo "FATAL: $SRC missing" >&2; exit 2; }

# Count data rows. Columns (pipe-delimited): 1=Closure date, 2=Title, 3=Type,
# 4=Status, 5=Round, 6=Commit(s), 7=Evidence. Skip the header row
# ("| Closure |") and the separator ("|---|"). A data row's col-1 is a date.
#
# Two orthogonal classifications are emitted (§11.4.16 Type + §11.4.90 Obsolete
# Status — Obsolete is a terminal Status orthogonal to Type, so an Obsolete row
# is BOTH counted in its Type bucket AND broken out in the Obsolete aggregate):
#   - Type buckets (3rd field): Bug / Feature / Task  (existing behavior, kept).
#   - Obsolete count (4th field Status == "Obsolete (→ Fixed.md)").
read -r BUG FEAT TASK OBSOLETE TOTAL <<<"$(
  awk -F'|' '
    /^\|/ {
      c1=$2; gsub(/^[ \t]+|[ \t]+$/, "", c1)
      if (c1 !~ /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/) next      # only YYYY-MM-DD data rows
      t=$4; gsub(/^[ \t]+|[ \t]+$/, "", t)
      if (t=="Bug") bug++; else if (t=="Feature") feat++; else if (t=="Task") task++
      s=$5; gsub(/^[ \t]+|[ \t]+$/, "", s)
      if (s ~ /^Obsolete \(/) obsolete++
      total++
    }
    END { printf "%d %d %d %d %d\n", bug, feat, task, obsolete, total }
  ' "$SRC"
)"

# Per-item Obsolete list (§11.4.90 visibility): ID + title + Obsolete reason if
# present in the Evidence/Status cells. Each line "  - <ID> — <title>".
OBSOLETE_LIST="$(
  awk -F'|' '
    /^\|/ {
      c1=$2; gsub(/^[ \t]+|[ \t]+$/, "", c1)
      if (c1 !~ /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/) next
      s=$5; gsub(/^[ \t]+|[ \t]+$/, "", s)
      if (s !~ /^Obsolete \(/) next
      title=$3; gsub(/^[ \t]+|[ \t]+$/, "", title)
      printf "- %s\n", title
    }
  ' "$SRC"
)"

TODAY="$(date +%Y-%m-%d)"
NEW="$(cat <<EOF
# HelixCode — Fixed Items Summary

> Generated **mechanically** from \`docs/Fixed.md\` by \`scripts/generate_fixed_summary.sh\` per Constitution §11.4.19 + §11.4.57 (Type-aware closure vocabulary). Counts only — do not hand-edit, re-run the generator.

## Aggregate counts

| Type | Count | Closure vocabulary (CONST-057) |
|---|---|---|
| Bug | $BUG | \`Fixed (→ Fixed.md)\` |
| Feature | $FEAT | \`Implemented (→ Fixed.md)\` |
| Task | $TASK | \`Completed (→ Fixed.md)\` |
| Obsolete | $OBSOLETE | \`Obsolete (→ Fixed.md)\` (§11.4.90 — terminal Status, orthogonal to Type) |

**Total closed items**: $TOTAL (counted directly from the \`docs/Fixed.md\` closure table).

## Obsolete items (§11.4.90)

${OBSOLETE_LIST:-_(none)_}

*Last regenerated: $TODAY by \`scripts/generate_fixed_summary.sh\`. HTML/PDF exports via \`scripts/regenerate-tracker-exports.sh\`.*
EOF
)"

case "$MODE" in
  check)
    if ! diff -u <(printf '%s\n' "$NEW") "$OUT" >/tmp/fixed_summary.diff 2>/dev/null; then
      echo "CM-FIXED-SUMMARY-SYNC: FAIL — docs/Fixed_Summary.md is stale; run scripts/generate_fixed_summary.sh" >&2
      cat /tmp/fixed_summary.diff >&2
      exit 1
    fi
    echo "CM-FIXED-SUMMARY-SYNC: PASS — summary in sync with Fixed.md"
    exit 0
    ;;
  stdout)
    # docs_chain derive-node contract: the LAST positional arg (if any) is the
    # engine-supplied output file the node content is read back from; the
    # preceding positionals are staged input files we don't need (SRC is read
    # directly above). Write canonical content there — NO status line, NO write
    # to docs/Fixed_Summary.md (so verify never mutates the live artefact and is
    # not polluted by a "wrote ..." status line). With no positionals, emit to
    # real stdout for ad-hoc inspection.
    if [[ ${#POSITIONAL[@]} -gt 0 ]]; then
      OUTFILE="${POSITIONAL[${#POSITIONAL[@]}-1]}"
      printf '%s\n' "$NEW" > "$OUTFILE"
    else
      printf '%s\n' "$NEW"
    fi
    exit 0
    ;;
  *)
    printf '%s\n' "$NEW" > "$OUT"
    echo "wrote $OUT (Bug=$BUG Feature=$FEAT Task=$TASK total=$TOTAL)"
    ;;
esac
