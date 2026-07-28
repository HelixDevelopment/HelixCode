#!/usr/bin/env bash
# scripts/gates/summary_sync_gate.sh — CM-ISSUES-SUMMARY-SYNC + CM-FIXED-SUMMARY-SYNC
#
# §11.4.12 + §11.4.53 + §11.4.91 + §11.4.86 freshness gate. The summary docs
# (docs/Issues_Summary.md, docs/Fixed_Summary.md) MUST be a mechanical
# projection of their authoritative source. This gate asserts they are.
#
# ---------------------------------------------------------------------------
# §11.4.120 GATE RECONCILIATION (2026-07-27) — the mechanism this gate asserts
# was SUPERSEDED; the gate is rewritten to assert the NEW one.
# ---------------------------------------------------------------------------
# WAS: this gate ran the LEGACY shell generators
#   scripts/generate_issues_summary.sh --check
#   scripts/generate_fixed_summary.sh  --check
# which derive the summaries from the TEXT trackers (docs/Issues.md,
# docs/Fixed.md) and emit headers like "# HelixCode — Fixed Items Summary".
#
# IS NOW: per §11.4.93 (SQLite single-source-of-truth for workable items) and
# §11.4.95 (the DB is TRACKED in git, it IS authoritative source data — never a
# build artefact), docs/workable_items.db is the single source of truth and the
# summary docs are DERIVED FROM IT by the constitution submodule's Go binary:
#
#   constitution/scripts/workable-items  →  `workable-items export`
#     (cmd/workable-items/export.go: renderIssuesSummary / renderFixedSummary)
#
# The committed docs carry that generator's headers verbatim:
#   "Open workable items (current_location = Issues), regenerated from the
#    SQLite single-source-of-truth (§11.4.12)."
#   "Closed workable items (current_location = Fixed), regenerated from the
#    SQLite single-source-of-truth (§11.4.53)."
#
# WHY THE OLD GATE HAD TO BE REWRITTEN, NOT SATISFIED (§11.4.120):
# Running the legacy generators to make the old gate green would OVERWRITE the
# SQLite-derived summaries with older text-derived output (measured 2026-07-27:
# legacy output tallies 188 closed items; the SQLite-derived committed doc
# tallies 344) — i.e. it would turn the gate green by CORRUPTING the docs and
# silently reverting a constitutional upgrade. §11.4.120 forbids all four
# suppressions (fake-pass / weaken-to-tautology / delete / revert-the-correct-
# change) and REQUIRES exactly this: rewrite the gate to assert the NEW
# mechanism, backed by captured evidence of the new correct behaviour.
#
# COMPANION GATES: workable_items_sync_gate.sh (§11.4.93/95) proves the DB
# validates and round-trips byte-identically against docs/Issues.md +
# docs/Fixed.md — but it does NOT cover the SUMMARIES. THIS gate closes that
# gap: it proves the summaries are the DB's current projection.
# summary_clarity_gate.sh (§11.4.91) checks the rows are MEANINGFUL; THIS gate
# checks they are FRESH.
#
# INVARIANT: a fresh `workable-items export` from the committed DB reproduces
# docs/Issues_Summary.md and docs/Fixed_Summary.md BYTE-IDENTICALLY.
#
# SAFETY: the tracked docs are NEVER written by this gate. The export runs into
# a temp dir, against a temp COPY of the DB (SQLite mutates a DB file's header
# even on a read-open in WAL mode, which would dirty the working tree — the same
# precaution workable_items_sync_gate.sh takes).
#
# Exit 0 = both summaries in sync (or an honest SKIP); non-zero = at least one
# summary has drifted from the DB single-source-of-truth.
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT" || exit 1

DB="docs/workable_items.db"
ISSUES_SUMMARY="docs/Issues_Summary.md"
FIXED_SUMMARY="docs/Fixed_Summary.md"
WI_SRC="constitution/scripts/workable-items"

fail() { echo "CM-SUMMARY-SYNC: FAIL — $1" >&2; exit 1; }
skip() { echo "CM-SUMMARY-SYNC: SKIP-OK — $1"; exit 0; }

[ -f "$DB" ]              || fail "tracked $DB missing (§11.4.95 requires it tracked — it is the summary source of truth)"
[ -f "$ISSUES_SUMMARY" ]  || fail "$ISSUES_SUMMARY missing"
[ -f "$FIXED_SUMMARY" ]   || fail "$FIXED_SUMMARY missing"
[ -d "$WI_SRC" ]          || skip "workable-items binary source not present at $WI_SRC (constitution submodule not checked out)"
command -v go >/dev/null 2>&1 || skip "go toolchain not available"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

BIN="$TMP/wi"
if ! ( cd "$WI_SRC" && go build -o "$BIN" ./cmd/workable-items ) 2>"$TMP/build.err"; then
    # CGO/sqlite unavailable is an environment limitation, not a sync defect.
    if grep -qiE 'cgo|sqlite|gcc|cc1|exec: "gcc"' "$TMP/build.err"; then
        skip "workable-items binary cannot build in this env (CGO/sqlite): $(tail -1 "$TMP/build.err")"
    fi
    fail "workable-items binary build failed: $(tail -3 "$TMP/build.err" | tr '\n' ' ')"
fi

# Export a fresh projection from a COPY of the committed DB into a temp dir.
# --no-formats keeps the gate independent of pandoc/weasyprint: the .md siblings
# are the contract this gate asserts (HTML/PDF freshness is §11.4.65's gate).
mkdir -p "$TMP/out"
cp "$DB" "$TMP/out/committed.db"
if ! "$BIN" export --db "$TMP/out/committed.db" --out-dir "$TMP/out" --no-formats >"$TMP/export.out" 2>&1; then
    fail "workable-items export failed: $(tail -3 "$TMP/export.out" | tr '\n' ' ')"
fi

REMEDY="regenerate from the DB: go run -C $WI_SRC ./cmd/workable-items export --db $DB --out-dir docs"

rc=0
for pair in "$ISSUES_SUMMARY:Issues_Summary.md:CM-ISSUES-SUMMARY-SYNC" \
            "$FIXED_SUMMARY:Fixed_Summary.md:CM-FIXED-SUMMARY-SYNC"; do
    committed="${pair%%:*}"
    rest="${pair#*:}"
    base="${rest%%:*}"
    gate="${rest#*:}"
    fresh="$TMP/out/$base"

    if [ ! -f "$fresh" ]; then
        echo "$gate: FAIL — export produced no $base (generator contract broken)" >&2
        rc=1
        continue
    fi
    if diff -u "$committed" "$fresh" >"$TMP/$base.diff" 2>&1; then
        echo "$gate: PASS — $committed matches the DB projection"
    else
        echo "$gate: FAIL — $committed has DRIFTED from the SQLite single-source-of-truth ($DB); $REMEDY" >&2
        head -40 "$TMP/$base.diff" >&2
        rc=1
    fi
done

if [ "$rc" -eq 0 ]; then
    echo "CM-SUMMARY-SYNC: PASS — Issues_Summary.md + Fixed_Summary.md are the current projection of $DB (§11.4.93/§11.4.95)"
fi
exit "$rc"
