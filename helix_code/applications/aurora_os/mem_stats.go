// Package main (aurora_os) — shared memory-diagnostics helper.
//
// This file carries NO //go:build constraint: it is compiled into BOTH the
// GUI (!nogui, main.go) and headless (nogui, main_nogui.go) variants of the
// aurora_os application, so the "Optimize performance" diagnostic computes
// its memory-freed delta identically on both builds via a single, tested
// source of truth.
package main

// memFreedMB computes the signed change in allocated heap memory, in
// megabytes, between a "before" and "after" runtime.MemStats.Alloc reading
// taken around a runtime.GC() cycle for the "Optimize performance"
// diagnostic (applications/aurora_os/main.go's optimizePerformance and
// applications/aurora_os/main_nogui.go's runOptimization).
//
// runtime.MemStats.Alloc is a uint64. The naive formula
// `float64(before-after) / 1024 / 1024` performs the subtraction in
// UNSIGNED (uint64) arithmetic BEFORE ever converting to float64.
// runtime.GC() does NOT guarantee Alloc decreases: if the application
// allocates memory between the first runtime.ReadMemStats call and the
// second one (taken after GC), after can exceed before and the subtraction
// wraps around to a value near 2^64 -- silently producing an absurd
// multi-exabyte "freed" value that is shown directly to the end user in
// the optimization report dialog (GUI) or printed to stdout (CLI). See
// TestMemFreedMB_NoUint64Underflow (mem_freed_underflow_test.go) for the
// §11.4.115 RED->GREEN regression guard and forensic detail.
//
// Converting each operand to float64 BEFORE subtracting avoids the wrap
// entirely -- heap sizes here are on the order of megabytes, far below
// float64's 53-bit exact-integer mantissa, so the subtraction is exact and
// correctly SIGNED regardless of which operand is larger. A negative
// result is legitimate and expected here: it means memory GREW between the
// two readings rather than shrank, and every caller/formatter of this
// value MUST be able to display that signed delta sensibly rather than
// assume "freed" is always non-negative.
func memFreedMB(before, after uint64) float64 {
	return (float64(before) - float64(after)) / 1024 / 1024
}
