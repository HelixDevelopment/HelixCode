#!/usr/bin/env bash
# obsolete_details_meta_test.sh — paired-mutation meta-test (§1.1) for the
# §11.4.90 obsolete-details gate (scripts/gates/obsolete_details_gate.sh).
#
# Plants a known violation in a temp fixture docs/ tree, asserts the gate
# FAILs; then repairs it, asserts the gate PASSes. Also asserts a fully
# compliant Obsolete item PASSes. Exits non-zero if any assertion fails.
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
GATE="$ROOT/scripts/gates/obsolete_details_gate.sh"

TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/docs"
# Fixed.md is required by the gate file list; keep it minimal + compliant.
cat > "$TMP/docs/Fixed.md" <<'EOF'
# Fixed
EOF

fail=0
assert_fail() {
    if "$GATE" "$TMP/docs" >/dev/null 2>&1; then
        echo "ASSERT-FAIL: gate PASSED but should have FAILED ($1)"
        fail=1
    else
        echo "ASSERT-OK:   gate FAILED as expected ($1)"
    fi
}
assert_pass() {
    if "$GATE" "$TMP/docs" >/dev/null 2>&1; then
        echo "ASSERT-OK:   gate PASSED as expected ($1)"
    else
        echo "ASSERT-FAIL: gate FAILED but should have PASSED ($1)"
        fail=1
    fi
}

# --- Mutation 1: Obsolete item with NO Obsolete-Details line → FAIL ---
cat > "$TMP/docs/Issues.md" <<'EOF'
# Issues

## XXX-001 — planted obsolete item missing details

**Status:** Obsolete (→ Fixed.md)
**Type:** Bug
**Discovered:** 2026-05-28

Some narrative but no obsolete-details line at all.
EOF
assert_fail "no Obsolete-Details line"

# --- Mutation 2: Obsolete-Details present but Reason out of closed vocab → FAIL ---
cat > "$TMP/docs/Issues.md" <<'EOF'
# Issues

## XXX-002 — planted obsolete item bad reason

**Status:** Obsolete (→ Fixed.md)
**Type:** Bug
**Obsolete-Details:** Since: 2026-05-28; Reason: just-because; Superseding-item: XXX-003; Triple-check: checked thrice in qa-results/x.log
EOF
assert_fail "Reason not in closed vocab"

# --- Repair: fully compliant Obsolete item → PASS ---
cat > "$TMP/docs/Issues.md" <<'EOF'
# Issues

## XXX-003 — planted obsolete item fully compliant

**Status:** Obsolete (→ Fixed.md)
**Type:** Bug
**Obsolete-Details:** Since: 2026-05-28; Reason: superseded-by-later-mandate; Superseding-item: XXX-004; Triple-check: re-verified in qa-results/obsolete/xxx003.log
EOF
assert_pass "compliant Obsolete item"

# --- Reconciliation (§11.4.120): Reason: not-reproducible is in the
#     §11.4.90 closed vocabulary and MUST be accepted (not a tautology —
#     Mutation 2's `just-because` is still rejected above). ---
cat > "$TMP/docs/Issues.md" <<'EOF'
# Issues

## XXX-006 — planted obsolete item with not-reproducible reason

**Status:** Obsolete (→ Fixed.md)
**Type:** Bug
**Obsolete-Details:** Since: 2026-06-22; Reason: not-reproducible; Superseding-item: none; Triple-check: 10/10 canonical-tree non-reproduction in qa-results/obsolete/xxx006.log
EOF
assert_pass "compliant Obsolete item with Reason: not-reproducible"

# --- Reconciliation (§11.4.120, 2026-07-27): the canonical §11.4.90 sub-fact
#     label is "Triple-check evidence:" — emitted verbatim by the §11.4.93/95
#     SSoT renderer (workable-items obsolete.go:187) when projecting
#     docs/workable_items.db → docs/Fixed.md. The gate MUST accept it. ---
cat > "$TMP/docs/Issues.md" <<'EOF'
# Issues

## XXX-007 — planted obsolete item using the canonical renderer label

**Status:** Obsolete (→ Fixed.md)
**Type:** Bug
**Obsolete-Details:** Since: 2026-07-12; Reason: duplicate-of; Superseding-item: b058c7c2; Triple-check evidence: scratch/discovery/fixes/XXX007_evidence.md
EOF
assert_pass "compliant Obsolete item with canonical 'Triple-check evidence:' label"

# --- Mutation 3 (pairs the reconciliation above — proves the widened label is
#     NOT a tautology): canonical label present but the evidence value is
#     EMPTY → MUST still FAIL. If this assertion ever passes, the gate has
#     been weakened into a label-presence check and no longer asserts that
#     positive captured evidence exists (§11.4.90 / §11.4.6). ---
cat > "$TMP/docs/Issues.md" <<'EOF'
# Issues

## XXX-008 — planted obsolete item with EMPTY triple-check evidence

**Status:** Obsolete (→ Fixed.md)
**Type:** Bug
**Obsolete-Details:** Since: 2026-07-12; Reason: duplicate-of; Superseding-item: b058c7c2; Triple-check evidence:
EOF
assert_fail "canonical label present but evidence value EMPTY"

# --- Control: no Obsolete items at all → PASS ---
cat > "$TMP/docs/Issues.md" <<'EOF'
# Issues

## XXX-005 — an ordinary closed bug

**Status:** Fixed (→ Fixed.md)
**Type:** Bug
EOF
assert_pass "no Obsolete items present"

if [[ "$fail" -ne 0 ]]; then
    echo "META-TEST FAIL: §11.4.90 obsolete-details gate did not behave as specified"
    exit 1
fi
echo "META-TEST PASS: §11.4.90 obsolete-details gate paired-mutation verified"
exit 0
