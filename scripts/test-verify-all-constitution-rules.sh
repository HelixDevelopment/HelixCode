#!/usr/bin/env bash
# scripts/test-verify-all-constitution-rules.sh
#
# Paired-mutation meta-test for verify-all-constitution-rules.sh per
# CONST-055 (§11.4.32) anti-bluff requirement:
#
#   "the sweep's own meta-test (paired mutation per §1.1) plants a
#    known violation of each enforced gate and asserts the sweep
#    reports FAIL for the planted gate. A sweep that exits PASS
#    without running every implementable gate is a CONST-055
#    violation."
#
# What this script does:
#   For each of the implementable gates G2, G3, G5 (gates whose
#   violation can be safely planted + reverted within this script's
#   lifetime — G1 cascade + G4 nested-own-org + G6 case-conformance
#   are project-state checks whose mutation requires committed
#   changes, so they're not covered by THIS script; their meta-tests
#   live with the underlying check scripts):
#
#     1. Plant a violation in a known location.
#     2. Run `bash scripts/verify-all-constitution-rules.sh --gate=<id>`.
#     3. Assert the sweep exits 1 AND reports FAIL for the planted gate.
#     4. Revert the violation.
#     5. Run the sweep again; assert it exits 0.
#
# Anti-bluff: the planted-then-reverted file mutation is captured
# evidence per §11.4.2 + paired-mutation per §1.1. If any sub-test
# fails (planted violation NOT caught, or revert doesn't restore
# green), the meta-test exits non-zero — proving the sweep is itself
# a bluff gate that needs hardening.
#
# Exit codes:
#   0 — every mutation caught + every revert restored green
#   1 — at least one mutation was NOT caught (sweep is a bluff gate)
#   2 — script setup error

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PASS=0
FAIL=0

check() {
    local label="$1" expected_rc="$2" actual_rc="$3"
    if [[ "$actual_rc" == "$expected_rc" ]]; then
        echo "  ✓ $label (rc=$actual_rc as expected)"
        PASS=$((PASS + 1))
    else
        echo "  ✗ $label (rc=$actual_rc, expected $expected_rc) — SWEEP IS BLUFFING"
        FAIL=$((FAIL + 1))
    fi
}

# ---------------------------------------------------------------------------
# Baseline: sweep must be green BEFORE any mutation
# ---------------------------------------------------------------------------
echo "=== Baseline (no mutation) ==="
bash scripts/verify-all-constitution-rules.sh --gate=G2 --quiet > /tmp/baseline-g2.out 2>&1
baseline_g2_rc=$?
bash scripts/verify-all-constitution-rules.sh --gate=G3 --quiet > /tmp/baseline-g3.out 2>&1
baseline_g3_rc=$?
bash scripts/verify-all-constitution-rules.sh --gate=G5 --quiet > /tmp/baseline-g5.out 2>&1
baseline_g5_rc=$?
check "G2 baseline green" 0 "$baseline_g2_rc"
check "G3 baseline green" 0 "$baseline_g3_rc"
check "G5 baseline green" 0 "$baseline_g5_rc"

# ---------------------------------------------------------------------------
# G2 mutation — plant a "simulated" bluff in production code
# ---------------------------------------------------------------------------
echo
echo "=== G2 mutation (plant bluff marker in production code) ==="
G2_TARGET="helix_code/internal/llm/_mutation_test_bluff.go"
cat > "$G2_TARGET" <<'EOF'
package llm

// MUTATION: this file plants a bluff marker for the G2 paired-mutation
// meta-test. The "simulated" string in the comment below MUST trip the
// CONST-035 anti-bluff smoke gate.
//
// simulated response — this is a TODO implement scaffold
EOF
bash scripts/verify-all-constitution-rules.sh --gate=G2 --quiet > /tmp/mutated-g2.out 2>&1
mutated_g2_rc=$?
rm -f "$G2_TARGET"  # revert immediately
check "G2 mutation caught (sweep exits 1)" 1 "$mutated_g2_rc"

# Re-run to confirm revert restored green
bash scripts/verify-all-constitution-rules.sh --gate=G2 --quiet > /tmp/reverted-g2.out 2>&1
check "G2 reverted (sweep exits 0)" 0 "$?"

# ---------------------------------------------------------------------------
# G3 mutation — plant a production file that imports internal/mocks
# ---------------------------------------------------------------------------
echo
echo "=== G3 mutation (plant mock-import in production code) ==="
G3_TARGET="helix_code/cmd/_mutation_test_mock_import.go"
cat > "$G3_TARGET" <<'EOF'
package main

// MUTATION: this file plants a mock-from-production import for the G3
// paired-mutation meta-test. The internal/mocks import below MUST trip
// the CONST-050(A) gate.
import _ "dev.helix.code/internal/mocks"
EOF
bash scripts/verify-all-constitution-rules.sh --gate=G3 --quiet > /tmp/mutated-g3.out 2>&1
mutated_g3_rc=$?
rm -f "$G3_TARGET"
check "G3 mutation caught (sweep exits 1)" 1 "$mutated_g3_rc"

bash scripts/verify-all-constitution-rules.sh --gate=G3 --quiet > /tmp/reverted-g3.out 2>&1
check "G3 reverted (sweep exits 0)" 0 "$?"

# ---------------------------------------------------------------------------
# G5 mutation — plant a tracked sensitive file
# ---------------------------------------------------------------------------
# The .gitignore at repo root already prevents `*.pem` from being added
# normally (defense in depth). To simulate an operator who force-adds
# past .gitignore (the actual failure mode G5 needs to catch), we use
# `git add -f` here. Revert with `git rm --cached`.
#
# The heredoc body carries an EXAMPLE marker on the BEGIN line (2026-08-13).
# Reason: this SCRIPT is tracked, so scripts/secret_scan.sh scans this line as
# ordinary file content and reported it as a "Private key header (generic)"
# hit — the scanner cannot tell a planted fixture from a real header. The
# marker is the remedy the scanner's own FAIL text recommends first ("add a
# redaction marker such as REDACTED / EXAMPLE / ... to the line"), and it
# costs ZERO detection coverage here because G5 keys off the `.pem` FILENAME
# and never reads the body (see verify-all-constitution-rules.sh: the gate is
# `git ls-files | grep -E '…\.pem$…'`). Do NOT instead allowlist this whole
# script in .scan-secrets-allow: that would permanently blind BOTH scanners to
# every future line of a live executable, where this marker blinds nothing.
# The marker must stay ON the BEGIN line — the scanner's content allowlist is
# per-line, so moving it to a neighbouring line restores the finding.
echo
echo "=== G5 mutation (force-add tracked sensitive file past .gitignore) ==="
G5_TARGET="_mutation_test.pem"
cat > "$G5_TARGET" <<'EOF'
-----BEGIN PRIVATE KEY----- (EXAMPLE — planted G5 fixture, not a real key)
this is a planted-fixture key for the G5 paired-mutation meta-test
-----END PRIVATE KEY-----
EOF
git add -f "$G5_TARGET" 2>/dev/null
bash scripts/verify-all-constitution-rules.sh --gate=G5 --quiet > /tmp/mutated-g5.out 2>&1
mutated_g5_rc=$?
git rm --cached "$G5_TARGET" >/dev/null 2>&1
rm -f "$G5_TARGET"
check "G5 mutation caught (sweep exits 1)" 1 "$mutated_g5_rc"

bash scripts/verify-all-constitution-rules.sh --gate=G5 --quiet > /tmp/reverted-g5.out 2>&1
check "G5 reverted (sweep exits 0)" 0 "$?"

# ---------------------------------------------------------------------------
# Final summary
# ---------------------------------------------------------------------------
echo
echo "=== test-verify-all-constitution-rules.sh summary ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
if [[ "$FAIL" -gt 0 ]]; then
    echo
    echo "FAIL: $FAIL paired-mutation(s) NOT caught — verify-all-constitution-rules.sh is a BLUFF gate"
    exit 1
fi
echo
echo "PASS: every implementable mutation caught + reverted cleanly — sweep honoured per §1.1"
exit 0
