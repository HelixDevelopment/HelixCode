// Package fyneui provides the UI-goroutine dispatch primitives every Fyne
// front-end in this repository MUST use when it mutates a widget from a
// background goroutine.
//
// # Why this exists (the defect it closes)
//
// Fyne widgets are NOT goroutine-safe. Since Fyne 2.6 the contract is stated
// verbatim in fyne.io/fyne/v2/thread.go:
//
//	DoAndWait is used to execute a specified function in the main Fyne runtime
//	context. This is required when a background process wishes to adjust
//	graphical elements of a running app.
//
// Mutating a widget directly from a worker goroutine races the renderer, which
// reads the same fields from the main goroutine. Under `-race` this surfaces as
// a write in (*widget.Entry).SetText / (*BaseWidget).Refresh against a read in
// (*textRenderer).Layout / (*BaseWidget).Theme.
//
// # Why fyne.Do alone is not sufficient here
//
// The two drivers this repository builds against behave DIFFERENTLY:
//
//   - The real (glfw) driver queues the closure onto the main runtime
//     goroutine's FIFO func-queue (internal/driver/glfw/loop.go:
//     funcQueue = async.NewUnboundedChan[funcData]) and the main loop drains it
//     in order. Mutation and rendering therefore both happen on the main
//     goroutine: correct, and ordered.
//
//   - The TEST driver (fyne.io/fyne/v2/test) does NOT queue at all. Its
//     DoFromGoroutine delegates straight to internal/async.EnsureNotMain, which
//     — when called off the main goroutine — simply invokes the closure INLINE
//     on the calling goroutine. Its documented model is "tests all run on a
//     single thread", so it offers no serialization to code that legitimately
//     spawns workers.
//
// uiMu closes exactly that gap, so the SAME production code is race-free under
// both drivers. Under glfw the lock is uncontended (everything already runs on
// one goroutine) and therefore effectively free; under the test driver it is
// what actually provides mutual exclusion. A renderer (in production the main
// loop; in a headless recording test the goroutine calling
// software.RenderCanvas) participates by calling Sync.
//
// # Usage rules
//
//   - Call Do / DoAndWait ONLY from a goroutine you spawned. Calling them from
//     the main/UI goroutine is a bug: under glfw fyne.Do would defer the work
//     instead of applying it immediately, and under the test driver
//     fyne.DoAndWait logs a thread error and re-dispatches onto a NEW goroutine
//     — manufacturing the very race this package exists to remove.
//   - Perform the whole read-modify-write inside ONE closure. Splitting it
//     (reading widget state outside, writing inside) leaves the read racing.
//   - Do NOT nest these calls: uiMu is not reentrant.
//
// # Known limitation: DoAndWait across driver shutdown
//
// HONESTY (§11.4.6): under glfw, DoAndWait enqueues fn onto the driver's
// func-queue and blocks on a done signal that only the main loop sends. That
// loop's shutdown path drains the pending entries — releasing their waiters
// WITHOUT running them — then sets a `drained` flag and closes the queue; once
// that flag is visible, later calls run fn INLINE on the caller rather than
// queueing it (internal/driver/glfw/loop.go: runOnMainWithWait). So the plain
// "called after shutdown" case is already handled by the driver. The residual
// exposure is the interleaving where an enqueue lands after the drain has
// emptied the queue but before the enqueuing goroutine observes `drained`:
// nothing will ever apply that entry or signal its done channel, so the caller
// parks forever. Do never waits, so it cannot park; its entry is simply
// discarded. Impact in THIS repository today is nil — the front-ends stop their
// worker goroutines via stopUpdate before teardown, and process exit reaps
// anything still parked. It would matter for a long-lived multi-window app that
// tears down one window while the process keeps running, where such a worker
// would leak for the remaining process lifetime. No guard is implemented here
// because a correct one needs a shutdown signal this package does not own.
package fyneui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

// TickOrStop blocks until either the next tick fires (returns true, meaning
// "do one round of work") or stop is closed (returns false, meaning "return
// from the loop").
//
// It exists because a UI refresh ticker written as the obvious
// `for range ticker.C { … }` can NEVER be stopped: the goroutine outlives the
// window it paints into, keeps mutating widgets after teardown, and leaks for
// the lifetime of the process. That is a real defect in its own right, and it
// also produces post-teardown races against whatever set the widgets up next.
//
// STOP WINS over a pending tick, via three checks — the same priority-select
// discipline already applied to this repository's data-update loops
// (§11.4.115). What each check actually buys, stated precisely, because two of
// them overlap and it would be easy to assume more coverage than exists:
//
//  1. A non-blocking pre-check. Guarantees that when stop is ALREADY closed on
//     entry the function returns false WITHOUT consuming a pending tick. Go's
//     select picks UNIFORMLY AT RANDOM among ready cases, so without this the
//     blocking select would swallow a buffered tick roughly half the time.
//     (Measured on this implementation: 1027 of 2000 ticks swallowed with the
//     pre-check removed, 0 of 2000 with it present.)
//  2. The blocking select. Guarantees the call returns when stop closes even
//     though no tick ever arrives — otherwise a stopped loop would hang until
//     the next tick.
//  3. A non-blocking re-check inside the tick branch. Narrows the window where
//     stop is closed BETWEEN (1) and (2) while a tick is already buffered, in
//     which case (2) may legitimately have picked the tick.
//
// HONESTY (§11.4.6) — on the limits of the above. Checks (1) and (3) are
// REDUNDANT with each other for the return VALUE: with either one present, an
// entry-time-closed stop yields false. They differ only in the side effect
// described in (1) and in the interleaving covered by (3). Check (3)'s window
// (stop closing in the few instructions between (1) and (2)) is NOT
// deterministically reachable from outside this function, and it is a
// best-effort NARROWING rather than a hard invariant: if stop closes strictly
// after a tick was legitimately taken, returning true is a defensible outcome.
// Consequently (3) is deliberately NOT claimed to be covered by a test that
// fails when it is deleted — the package tests pin (1) and (2) only. It is kept
// because it is cheap, correct, and mirrors the established loop pattern in
// this repository; not because it is proven necessary.
//
// A nil stop channel blocks forever, so passing one yields a never-stopping
// ticker — the previous behaviour — rather than a nil-channel panic.
func TickOrStop(tick <-chan time.Time, stop <-chan struct{}) bool {
	// (1) Priority pre-check: stop already closed ⇒ never do more work.
	select {
	case <-stop:
		return false
	default:
	}

	// (2) Blocking wait for whichever comes first.
	select {
	case <-stop:
		return false
	case <-tick:
	}

	// (3) Residual re-check: stop may have closed between (1) and (2) while a
	// tick was already buffered, in which case (2) could have picked the tick.
	select {
	case <-stop:
		return false
	default:
	}
	return true
}

// uiMu serializes UI mutation against UI rendering. See the package doc for
// why the fyne.Do family alone does not provide this under the test driver.
var uiMu sync.Mutex

// Do runs fn on the Fyne UI goroutine and returns WITHOUT waiting for it.
//
// Ordering: under the glfw driver the closures queued by a single goroutine are
// applied in the order they were submitted, because the driver's func-queue is
// an unbounded FIFO channel drained sequentially by the main loop. Successive
// appends from one worker therefore land in order.
//
// Prefer Do for high-frequency, fire-and-forget updates — notably per-token
// streaming appends — where blocking the producer on the render loop would
// throttle the stream.
//
// MUST be called from a non-main goroutine (see package doc).
func Do(fn func()) {
	fyne.Do(func() {
		uiMu.Lock()
		defer uiMu.Unlock()
		fn()
	})
}

// DoAndWait runs fn on the Fyne UI goroutine and blocks until it has completed.
//
// Prefer DoAndWait when the caller must observe the UI mutation as having
// happened before it proceeds — for example a terminal/final update whose
// completion the caller (or a test) synchronizes on.
//
// MUST be called from a non-main goroutine (see package doc).
func DoAndWait(fn func()) {
	fyne.DoAndWait(func() {
		uiMu.Lock()
		defer uiMu.Unlock()
		fn()
	})
}

// Sync runs fn on the CALLER's goroutine while holding the UI lock.
//
// It is the entry point for a renderer or for main-goroutine code that must be
// mutually exclusive with concurrent Do / DoAndWait closures. Unlike Do it
// neither hops goroutines nor defers the work, so it is safe to call from the
// main/UI goroutine — which is precisely where a renderer runs.
func Sync(fn func()) {
	uiMu.Lock()
	defer uiMu.Unlock()
	fn()
}
