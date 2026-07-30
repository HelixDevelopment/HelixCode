#!/usr/bin/env bash
# scripts/gates/export_output_location_gate.sh — CM-EXPORT-OUTPUT-LOCATION
#
# §11.4.108 (four-layer fix-verification) + §11.4.135 (standing regression
# guard) guard for HXC-201.
#
# ---------------------------------------------------------------------------
# THE BUG THIS GUARDS
# ---------------------------------------------------------------------------
# The command every reference site documents for regenerating the tracker
# documents is:
#
#   go run -C constitution/scripts/workable-items ./cmd/workable-items \
#       export --db docs/workable_items.db --out-dir docs
#
# `go run -C <dir>` changes the BUILT BINARY's own working directory to <dir>
# — not merely a build-time chdir, the child process's real cwd — so a
# relative --db/--out-dir resolves against constitution/scripts/
# workable-items/ instead of the caller's real project root. The exporter
# printed success-looking "wrote docs/Issues.md" lines while writing nothing
# useful to the real project's docs/ tree, leaving it hours stale, and (since
# no DB existed at the relocated path) silently materialised a fresh,
# zero-row docs/workable_items.db inside the tool's own directory — the exact
# stub previously found and initially misattributed to a different mistake.
#
# ---------------------------------------------------------------------------
# WHAT THIS ASSERTS
# ---------------------------------------------------------------------------
# The gate reproduces the DOCUMENTED invocation VERBATIM (relative --db /
# --out-dir, unchanged) inside a disposable scratch project root — NEVER
# touching the real project's docs/ tree (a fresh copy of the tracked DB
# seeds the scratch root; the real docs/workable_items.db is only read) —
# and asserts:
#
#   (a) the documented invocation exits 0 and the four regenerated docs
#       (Issues.md / Fixed.md / Issues_Summary.md / Fixed_Summary.md) land
#       at the scratch project's real docs/ path, non-empty;
#   (b) nothing is written inside the relocated tool directory
#       (constitution/scripts/workable-items/docs/) — the exact location the
#       historical bug silently wrote to instead.
#
# A regression in EITHER the export.go/db.go $PWD-anchoring fix (HXC-201
# tool-side hardening) OR the documented invocation's absolute-path form
# would surface here: this gate exercises the invocation LITERALLY, not the
# Go package internals (see also the constitution submodule's own
# TestExportCmd_RelativeFlags_AnchorAtInvocationPWD_NotProcessCwd for the
# in-process equivalent proof).
#
# Exit codes: 0 = pass (bug does not reproduce), 1 = violation (bug
# reproduces or expected output missing), 2 = §11.4.3 environment SKIP (no Go
# toolchain / workable-items source / tracked DB present — certifies
# nothing, MUST NOT be worded as a detected regression by the caller).
# ---------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
WI_SRC="$ROOT/constitution/scripts/workable-items"
REAL_DB="$ROOT/docs/workable_items.db"

GATE="CM-EXPORT-OUTPUT-LOCATION"

if ! command -v go >/dev/null 2>&1; then
    echo "$GATE: SKIP-OK — no Go toolchain on PATH"
    exit 2
fi
if [[ ! -d "$WI_SRC" ]]; then
    echo "$GATE: SKIP-OK — $WI_SRC not present (constitution submodule not checked out)"
    exit 2
fi
if [[ ! -f "$REAL_DB" ]]; then
    echo "$GATE: SKIP-OK — $REAL_DB not present"
    exit 2
fi

TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

SCRATCH="$TMP/project"
mkdir -p "$SCRATCH/constitution/scripts" "$SCRATCH/docs"
cp -r "$WI_SRC" "$SCRATCH/constitution/scripts/workable-items"
# Clean slate: strip any pre-existing (gitignored) stub under the copied
# tool's own docs/ so a later find there is unambiguous evidence of a FRESH
# write by THIS run, never a leftover from a prior local reproduction.
rm -rf "$SCRATCH/constitution/scripts/workable-items/docs"
cp "$REAL_DB" "$SCRATCH/docs/workable_items.db"

pushd "$SCRATCH" >/dev/null || exit 2

# The EXACT documented invocation — relative --db / --out-dir, unchanged.
if ! out=$(go run -C constitution/scripts/workable-items ./cmd/workable-items export \
    --db docs/workable_items.db --out-dir docs --no-formats 2>&1); then
    popd >/dev/null || true
    echo "$GATE: FAIL — the documented export invocation exited non-zero" >&2
    echo "$out" >&2
    exit 1
fi

fail=0

# (a) The four docs must exist at the scratch project's real docs/ path,
#     non-empty — the destination the docs promise.
for f in Issues.md Fixed.md Issues_Summary.md Fixed_Summary.md; do
    p="$SCRATCH/docs/$f"
    if [[ ! -s "$p" ]]; then
        echo "$GATE: FAIL — $f missing or empty at docs/$f (HXC-201 regression: output did not land at the documented destination)" >&2
        fail=1
    fi
done

# (b) Nothing must have been written inside the relocated tool directory —
#     the historical bug's actual write location.
if [[ -e "$SCRATCH/constitution/scripts/workable-items/docs" ]]; then
    echo "$GATE: FAIL — export wrote into constitution/scripts/workable-items/docs/ (the relocated -C cwd) instead of the caller's docs/ tree — HXC-201 regressed" >&2
    ls -la "$SCRATCH/constitution/scripts/workable-items/docs" >&2
    fail=1
fi

popd >/dev/null || true

if [[ "$fail" -ne 0 ]]; then
    exit 1
fi

echo "$GATE: PASS — documented export invocation wrote docs/{Issues,Fixed,Issues_Summary,Fixed_Summary}.md to the caller's real docs/ tree; nothing landed inside the relocated tool directory"
exit 0
