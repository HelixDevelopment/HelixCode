#!/usr/bin/env bash
# hxc237_workable_items_binary_freshness_gate.sh — CM-WORKABLE-ITEMS-BINARY-FRESH
#
# ===========================================================================
# WHY THIS EXISTS
# ===========================================================================
# HXC-237. The command this project uses to validate its own workable-items
# records — `constitution/scripts/workable-items/bin/workable-items validate` —
# is a TRACKED PREBUILT BINARY. On 2026-08-06 commit 09ac917 added a new
# invariant to the SOURCE (`unresolvableClosureEvidence()` in
# constitution/scripts/workable-items/cmd/workable-items/sync.go): a closed
# item whose `item_history.evidence_path` does not resolve to a real
# filesystem entry is a violation (HXC-217, §11.4.5/§11.4.69/§11.4.123).
#
# The prebuilt binary was NOT rebuilt. It was last built 2026-07-27 — ten days
# before that invariant existed. So the check was present in the source (and
# looked correct to anyone reading it) while being entirely ABSENT from the
# program people actually run. Measured 2026-08-08 on one deliberately
# corrupted copy of the records, everything else held identical:
#
#   shipped bin/workable-items validate  -> exit 0, "OK — 437 items"   (BLIND)
#   go run -C … ./cmd/workable-items     -> exit 1, 1 violation found  (SEEING)
#
# That is the §11.4.108 SOURCE-vs-ARTIFACT gap: source-green said nothing about
# the artifact operators actually invoke. Every closure signed off with the
# stale binary since 2026-08-06 carried less assurance than it appeared to.
#
# ===========================================================================
# WHAT THIS GATE ASSERTS — BEHAVIOUR, NOT MTIME
# ===========================================================================
# A modification-time comparison (binary older than newest source) is only a
# PROXY, and §11.4.201 forbids gating on a proxy when the REAL condition is
# checkable: a `touch` on the binary would satisfy an mtime gate while leaving
# the artifact just as blind. This gate therefore asserts the REAL condition —
# that the SHIPPED ARTIFACT genuinely CARRIES the invariant — by RUNNING it
# against a golden-bad fixture and requiring it to report the violation.
#
# The mtime comparison is retained only as a secondary EARLY WARNING (advisory
# note, never the pass/fail basis), because it can flag drift before a future
# invariant is even fixtured.
#
# Fixtures are SYNTHETIC and self-contained (a fresh DB from the tool's own
# schema.sql, seeded with one closed item), so this gate NEVER depends on the
# live docs/workable_items.db being clean — an unrelated real-DB violation
# must not turn this guard into a §11.4.201(1) false-positive refusal. The
# assertion is on the SPECIFIC evidence-resolvability message, not on the
# validator's overall exit code, for the same reason: the synthetic fixture
# legitimately trips unrelated body_md/doc_segments invariants.
#
# ===========================================================================
# POLARITY SWITCH (Constitution §11.4.115)
# ===========================================================================
#   RED_MODE=1 — reproduce the DEFECT on the REAL PRE-FIX ARTIFACT. Extracts
#       the actual stale binary blob from the constitution submodule's git
#       object store (blob PREFIX_BLOB below, the tree state before the
#       HXC-237 rebuild) and asserts it does NOT emit the evidence-
#       resolvability violation on the golden-bad fixture — i.e. the shipped
#       artifact really was blind. PASS here is the reproduction proof. This
#       is a genuine §11.4.115 RED-on-the-broken-artifact, not a synthetic
#       stand-in: it runs the very bytes that shipped.
#
#   RED_MODE=0 — DEFAULT. The standing GREEN regression guard. Asserts the
#       CURRENTLY TRACKED binary DOES emit the violation on the golden-bad
#       fixture (it carries the invariant) AND does NOT emit it on the
#       golden-good fixture (it is not merely always-failing — the
#       §11.4.201(1) false-positive guard).
#
# Exit codes: 0 = satisfied · 1 = violated · 2 = environment SKIP (§11.4.3,
# certifies nothing).
#
# Usage:
#   bash scripts/gates/hxc237_workable_items_binary_freshness_gate.sh
#   RED_MODE=1 bash scripts/gates/hxc237_workable_items_binary_freshness_gate.sh
# ===========================================================================
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WI_DIR="$ROOT/constitution/scripts/workable-items"
BIN="$WI_DIR/bin/workable-items"
SCHEMA="$WI_DIR/schema.sql"
SRC_DIR="$WI_DIR/cmd/workable-items"

# The pre-fix binary blob: the tracked artifact as it stood BEFORE the HXC-237
# rebuild (last written by constitution commit 159399c, 2026-07-21). Pinned by
# blob hash so the RED reproduction is durable and content-addressed — it
# cannot drift with branch movement.
PREFIX_BLOB="a36be0b02f3f21969d5ffa923e32726ac94093bb"

# The exact violation substring the HXC-217 invariant emits.
NEEDLE="closure evidence_path does not resolve"

RED_MODE="${RED_MODE:-0}"
WORK=""

cleanup() { [[ -n "$WORK" && -d "$WORK" ]] && rm -rf "$WORK"; }
trap cleanup EXIT

skip() { echo "SKIP (§11.4.3 environment): $*"; exit 2; }
fail() { echo "FAIL: $*"; exit 1; }

# --- environment preconditions ------------------------------------------
command -v sqlite3 >/dev/null 2>&1 || skip "sqlite3 not on PATH — cannot build the fixture"
[[ -f "$SCHEMA" ]] || skip "schema.sql not found at $SCHEMA"
[[ -x "$BIN" ]]    || skip "tracked binary not present/executable at $BIN"

WORK="$(mktemp -d)" || skip "cannot create a temp working directory"

# --- fixtures -------------------------------------------------------------
# golden-good: the closed item's evidence_path RESOLVES.
# golden-bad : identical except the evidence_path points nowhere.
mkdir -p "$WORK/evid"
printf 'captured runtime evidence\n' > "$WORK/evid/proof.log"

sqlite3 "$WORK/good.db" < "$SCHEMA" >/dev/null 2>&1
[[ -s "$WORK/good.db" ]] || skip "could not materialise the fixture DB from schema.sql"

sqlite3 "$WORK/good.db" "
INSERT INTO items (atm_id,type,status,severity,title,description,current_location)
VALUES ('FIX-001','Bug','Fixed (→ Fixed.md)','High',
        'fixture closed item for the HXC-237 binary-freshness gate',
        'A synthetic closed item used only by the HXC-237 binary-freshness gate fixture.',
        'Fixed');
INSERT INTO item_history (atm_id,event_type,by,on_date,reason,evidence_path)
VALUES ('FIX-001','Fixed','AI','2026-08-08','test-failed','$WORK/evid/proof.log');" >/dev/null 2>&1
[[ "$?" -eq 0 ]] || skip "could not seed the fixture DB"

cp "$WORK/good.db" "$WORK/bad.db"
sqlite3 "$WORK/bad.db" \
  "UPDATE item_history SET evidence_path='$WORK/NO_SUCH_DIR/missing.log' WHERE atm_id='FIX-001';" \
  >/dev/null 2>&1
[[ "$?" -eq 0 ]] || skip "could not materialise the golden-bad fixture"

# emits_violation <binary> <db>  -> 0 when the evidence violation is reported
emits_violation() {
    local exe="$1" db="$2" out
    out="$("$exe" validate --db "$db" 2>&1)"
    # NOTE: the validator's overall exit code is deliberately IGNORED — the
    # synthetic fixture legitimately trips unrelated invariants. We assert on
    # the SPECIFIC evidence-resolvability message.
    case "$out" in *"$NEEDLE"*) return 0 ;; *) return 1 ;; esac
}

# ==========================================================================
# RED_MODE=1 — reproduce the defect on the REAL pre-fix artifact
# ==========================================================================
if [[ "$RED_MODE" == "1" ]]; then
    echo "=== RED_MODE=1 — reproducing HXC-237 on the real pre-fix artifact ==="
    git -C "$ROOT/constitution" cat-file -e "$PREFIX_BLOB" 2>/dev/null
    [[ "$?" -eq 0 ]] || skip "pre-fix blob $PREFIX_BLOB unreachable in the constitution object store"

    git -C "$ROOT/constitution" cat-file blob "$PREFIX_BLOB" > "$WORK/prefix-bin" 2>/dev/null
    [[ "$?" -eq 0 && -s "$WORK/prefix-bin" ]] || skip "could not extract the pre-fix blob"
    chmod +x "$WORK/prefix-bin"

    if emits_violation "$WORK/prefix-bin" "$WORK/bad.db"; then
        fail "the PRE-FIX binary DID report the evidence violation — the RED reproduction did not reproduce. Either PREFIX_BLOB no longer names the stale artifact, or the defect's mechanism is misunderstood (§11.4.115: a RED that passes on the known-broken artifact is a blind test)."
    fi
    echo "RED PASS — the pre-fix shipped binary is BLIND to an unresolvable"
    echo "           closure evidence_path (blob $PREFIX_BLOB). This is the"
    echo "           HXC-237 defect, reproduced on the artifact that shipped."
    exit 0
fi

# ==========================================================================
# RED_MODE=0 — the standing GREEN guard
# ==========================================================================
echo "=== RED_MODE=0 — standing guard: the SHIPPED binary carries the invariant ==="

# S1 — the tracked binary MUST see the planted violation.
if ! emits_violation "$BIN" "$WORK/bad.db"; then
    fail "the TRACKED binary ($BIN) did NOT report an unresolvable closure evidence_path on the golden-bad fixture. The shipped artifact is stale relative to its source — rebuild it:
    cd constitution/scripts/workable-items && go build -o bin/workable-items ./cmd/workable-items
(§11.4.108 SOURCE-vs-ARTIFACT: a check present in the source is worth nothing if the binary operators run does not contain it.)"
fi
echo "  S1 yes — tracked binary reports the violation on the golden-bad fixture"

# S2 — false-positive guard: it must NOT report it when the evidence resolves.
if emits_violation "$BIN" "$WORK/good.db"; then
    fail "the TRACKED binary reported an unresolvable closure evidence_path on the golden-GOOD fixture, whose evidence file demonstrably exists ($WORK/evid/proof.log). The guard is not discriminating — it would refuse valid closures (§11.4.201(1) false-positive refusal is a FAIL-bluff)."
fi
echo "  S2 yes — tracked binary stays silent when the evidence file exists"

# S3 — secondary, ADVISORY ONLY: mtime drift early warning.
newest_src=""
if [[ -d "$SRC_DIR" ]]; then
    newest_src="$(find "$SRC_DIR" -name '*.go' -newer "$BIN" -print -quit 2>/dev/null)"
fi
if [[ -n "$newest_src" ]]; then
    echo "  S3 NOTE (advisory, not a failure basis) — at least one source file is"
    echo "         newer than the tracked binary: $newest_src"
    echo "         S1/S2 still pass, so the invariant under guard IS present; but a"
    echo "         newer source may carry OTHER changes the artifact lacks. Consider"
    echo "         rebuilding and extending this gate's fixtures for them."
else
    echo "  S3 yes — no source file is newer than the tracked binary"
fi

echo "GREEN PASS — the shipped workable-items binary genuinely carries the"
echo "             closure-evidence-resolvability invariant (behaviour-asserted,"
echo "             not mtime-inferred)."
exit 0
