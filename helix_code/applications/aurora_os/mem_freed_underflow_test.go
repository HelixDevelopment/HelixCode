// Package main (aurora_os) — §11.4.115 RED-baseline-on-the-broken-artifact
// regression guard for the aurora_os "Optimize performance" diagnostic's
// uint64-underflow defect.
//
// FILES fixed by this guard:
//   - applications/aurora_os/main.go (GUI optimizePerformance)
//   - applications/aurora_os/main_nogui.go (CLI runOptimization)
//
// Both previously computed:
//
//	freed := float64(before.Alloc-after.Alloc) / 1024 / 1024
//
// runtime.MemStats.Alloc is uint64. That subtraction ran in UNSIGNED
// arithmetic BEFORE the float64() conversion. runtime.GC() does NOT
// guarantee Alloc decreases -- if the app allocates between the two
// ReadMemStats calls, after.Alloc can exceed before.Alloc and the
// subtraction wraps to ~1.8e19, producing a nonsensical multi-exabyte
// "freed" value shown directly to the end user (GUI dialog / CLI stdout).
//
// The fix (memFreedMB in mem_stats.go, shared -- no build tag -- by both
// build variants) converts each operand to float64 BEFORE subtracting,
// producing a correctly SIGNED result: positive when memory shrank
// (before > after), negative when memory grew (before < after -- the
// underflow-triggering case here).
//
// This test file has NO //go:build constraint (matching mem_stats.go) so
// the guard runs identically on both the GUI and headless build variants.
//
// Mirrors the established repo-wide pattern for this exact defect class:
// submodules/helix_agent/tests/unit/stress_legacy/heap_growth_underflow_red_test.go
// (heapGrowthMB / runtime.MemStats.HeapInuse). Like that guard, RED_MODE=1
// calls the SAME shared helper the GREEN path calls (never an inline
// re-derivation of the naive formula) so the guard is meaningful against
// whichever revision of the helper is actually checked out:
//
//   - RED_MODE=1 (run against a PRE-FIX memFreedMB): asserts the growth
//     case produces the absurd multi-exabyte underflow artifact (> 1e12
//     MB), reproducing the defect. Against the CURRENT, already-fixed
//     helper this branch honestly FAILS ("the bug is already fixed, flip
//     RED_MODE=0") -- proving the guard is not a tautology.
//   - RED_MODE unset / "0" (the DEFAULT, standing GREEN guard): asserts a
//     shrink yields a small correctly-signed POSITIVE number and a growth
//     yields a small correctly-signed NEGATIVE number -- BOTH directions,
//     never the underflow artifact.
package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestMemFreedMB_NoUint64Underflow is the primary RED/GREEN polarity-switch
// guard, driven by the exact forensic scenario: the app allocated between
// runtime.GC() and the second runtime.ReadMemStats call, so after.Alloc
// ended up larger than before.Alloc (memory grew rather than shrank).
func TestMemFreedMB_NoUint64Underflow(t *testing.T) {
	redMode := os.Getenv("RED_MODE") == "1"

	const before uint64 = 50 * 1024 * 1024 // 50 MiB
	const after uint64 = 62 * 1024 * 1024  // 62 MiB (grew by 12 MiB)

	got := memFreedMB(before, after)

	if redMode {
		if got < 1e12 {
			t.Fatalf("RED_MODE=1: expected the pre-fix uint64-subtraction-before-float64-conversion "+
				"bug to produce an absurd underflow artifact (> 1e12 MB) for a growing heap "+
				"(before=%d after=%d), got %.4f MB -- the bug is already fixed, flip RED_MODE=0",
				before, after, got)
		}
		t.Logf("RED_MODE=1: reproduced the uint64 underflow -- memFreedMB(%d, %d) = %.4f MB "+
			"(absurd multi-exabyte artifact from a growing heap, matching the wild "+
			"\"heap growth: 17592186044415.94 MB\"-class report)", before, after, got)
		return
	}

	// GREEN guard: a growing heap MUST yield a small, correctly-signed
	// NEGATIVE number -- never the absurd underflow artifact.
	if got >= 0 || got < -1e6 {
		t.Fatalf("memFreedMB(%d, %d) = %.2f MB, want a small NEGATIVE value (memory grew), not an absurd/non-negative result", before, after, got)
	}
	const wantMB = -12.0
	if diff := got - wantMB; diff > 0.01 || diff < -0.01 {
		t.Fatalf("memFreedMB(%d, %d) = %.2f MB, want %.2f MB", before, after, got, wantMB)
	}
	t.Logf("growth case OK: memFreedMB(%d, %d) = %.2f MB (correctly negative, no underflow)", before, after, got)
}

// TestMemFreedMB_ShrinkYieldsPositive proves the both-directions guarantee
// for the common case: a genuine shrink (GC reclaims memory) yields a
// sensible POSITIVE "freed" value, not merely "not an underflow". This is
// the GREEN-only companion to the growth case exercised above (a shrink
// never underflows under the naive formula, so there is no RED_MODE branch
// here — see TestMemFreedMB_NoUint64Underflow for the polarity switch).
func TestMemFreedMB_ShrinkYieldsPositive(t *testing.T) {
	const before uint64 = 80 * 1024 * 1024 // 80 MiB
	const after uint64 = 55 * 1024 * 1024  // 55 MiB (shrank by 25 MiB)

	got := memFreedMB(before, after)
	const wantMB = 25.0
	if diff := got - wantMB; diff > 0.01 || diff < -0.01 {
		t.Fatalf("memFreedMB(%d, %d) = %.2f MB, want %.2f MB", before, after, got, wantMB)
	}
}

// TestMemFreedMB_NoChange proves the boundary case: identical readings
// yield exactly zero, never a near-zero-but-wrapped artifact.
func TestMemFreedMB_NoChange(t *testing.T) {
	const v uint64 = 42 * 1024 * 1024
	got := memFreedMB(v, v)
	if got != 0 {
		t.Fatalf("memFreedMB(%d, %d) = %.2f MB, want exactly 0", v, v, got)
	}
}

// TestMemFreedMB_FormattingHandlesNegativeDelta checks the user-facing
// rendering path: both aurora_os_optimization_report_fmt (GUI) and
// aurora_os_cli_memory_freed (CLI) format the freed value with the plain
// "%.2f MB" verb. This proves that verb renders a negative delta (memory
// grew) sensibly -- a signed decimal like "-12.00 MB" -- rather than
// scientific notation, a wrapped multi-exabyte figure, or other garbage.
func TestMemFreedMB_FormattingHandlesNegativeDelta(t *testing.T) {
	// var, not const: the real call sites read before.Alloc/after.Alloc from
	// a runtime.MemStats struct (runtime values), so the naive-formula
	// comparison below must also subtract at RUNTIME (wrapping, like the
	// real uint64 arithmetic) rather than as a compile-time constant
	// expression (which Go would instead reject with an overflow error).
	var before uint64 = 50 * 1024 * 1024
	var after uint64 = 62 * 1024 * 1024

	freed := memFreedMB(before, after)
	rendered := fmt.Sprintf("Memory freed: %.2f MB", freed)
	// Isolate just the formatted NUMBER (not the surrounding English prose,
	// which legitimately contains the letter 'e' in "Memory"/"freed") before
	// checking for scientific notation.
	numOnly := fmt.Sprintf("%.2f", freed)

	if !strings.Contains(rendered, "-12.00") {
		t.Fatalf("rendered = %q, want a sensible signed value containing -12.00", rendered)
	}
	if strings.ContainsAny(numOnly, "eE") {
		t.Fatalf("formatted number = %q (in rendered = %q), contains scientific notation -- looks like garbage to end users", numOnly, rendered)
	}

	// The historical bug's rendering, for contrast/documentation: the fixed
	// rendering MUST NOT match what the naive uint64-underflow formula would
	// have produced for this same (before, after) pair.
	naiveGarbage := fmt.Sprintf("Memory freed: %.2f MB", float64(before-after)/1024/1024)
	if rendered == naiveGarbage {
		t.Fatalf("fixed rendering (%q) matches the naive uint64-underflow rendering (%q) -- fix not effective", rendered, naiveGarbage)
	}
}
