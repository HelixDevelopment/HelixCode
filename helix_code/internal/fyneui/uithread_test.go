package fyneui

import (
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
)

// newTestAppForUIThread installs fyne's headless test driver so the Do /
// DoAndWait dispatch path has a live app to route through. No window, no GL,
// no display required.
func newTestAppForUIThread(t *testing.T) {
	t.Helper()
	test.NewApp()
}

// Mutation-verified coverage map for TickOrStop's three checks.
//
// These tests were themselves validated by paired mutation: each check was
// deleted in an isolated copy of the package and the suite re-run. The result
// is stated honestly rather than aspirationally, because the FIRST version of
// this file claimed all three checks were independently pinned and BOTH
// mutations survived it — the precise "guard is under-specified, half of it can
// be deleted unnoticed" defect these tests exist to prevent.
//
//	check (1) non-blocking pre-check    -> PINNED by TestTickOrStop_PreCheck_
//	                                       DoesNotSwallowPendingTickWhenStopped
//	                                       (deleting it: 1027/2000 iterations
//	                                       swallow the tick -> test FAILS)
//	check (2) blocking select stop case -> PINNED by TestTickOrStop_Blocking
//	                                       Select_ReturnsFalseWhenStopClosesLater
//	                                       (deleting it: the call never returns
//	                                       -> test FAILS on its timeout)
//	check (3) residual re-check         -> NOT PINNED. Deliberately so: see the
//	                                       honesty note on TickOrStop. Its
//	                                       window is not deterministically
//	                                       reachable from outside, and it is a
//	                                       best-effort narrowing rather than a
//	                                       hard invariant. Deleting it does NOT
//	                                       fail this suite, and no test here
//	                                       pretends otherwise.
//
// Return-value equivalence note: because (1) and (3) are mutually redundant for
// the RETURN VALUE, a test that only asserts "returns false" can never
// distinguish them. The (1) test therefore asserts the side effect — the
// pending tick must survive — which is the one externally observable difference.

// tickOrStopIterations is high enough that a uniformly-random select would
// take the wrong branch at least once with overwhelming probability
// (P(all-2000-correct-by-chance) = 2^-2000).
const tickOrStopIterations = 2000

// TestTickOrStop_PreCheck_DoesNotSwallowPendingTickWhenStopped pins check (1).
//
// Both channels are ready BEFORE the call: stop is closed and a tick is already
// buffered. The return value alone cannot distinguish the pre-check (check (3)
// would produce false too), so this asserts the observable SIDE EFFECT instead:
// an already-stopped call must not consume the pending tick. Without the
// pre-check the blocking select sees two ready cases, picks uniformly at
// random, and swallows the tick about half the time.
func TestTickOrStop_PreCheck_DoesNotSwallowPendingTickWhenStopped(t *testing.T) {
	for i := 0; i < tickOrStopIterations; i++ {
		tick := make(chan time.Time, 1)
		tick <- time.Now() // tick already pending
		stop := make(chan struct{})
		close(stop) // stop already closed

		if TickOrStop(tick, stop) {
			t.Fatalf("iteration %d: TickOrStop returned true although stop was "+
				"closed before the call — a pending tick must never outrank a "+
				"closed stop", i)
		}
		if len(tick) != 1 {
			t.Fatalf("iteration %d: the pending tick was consumed by a call that "+
				"was already stopped on entry — the non-blocking priority "+
				"pre-check is missing, so the blocking select raced for it", i)
		}
	}
}

// TestTickOrStop_BlockingSelect_ReturnsFalseWhenStopClosesLater isolates
// check (2): no tick EVER arrives, so only the blocking select's stop case can
// return. If that case were removed this test would hang rather than fail, so
// it is bounded by an explicit timeout to report a failure instead.
func TestTickOrStop_BlockingSelect_ReturnsFalseWhenStopClosesLater(t *testing.T) {
	tick := make(chan time.Time) // never fires
	stop := make(chan struct{})

	done := make(chan bool, 1)
	go func() { done <- TickOrStop(tick, stop) }()

	// Let the call reach its blocking select, then release it via stop only.
	time.Sleep(20 * time.Millisecond)
	close(stop)

	select {
	case got := <-done:
		if got {
			t.Fatal("TickOrStop returned true although only stop was signalled")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("TickOrStop did not return after stop was closed — the blocking " +
			"select has no stop case (it can only be woken by a tick)")
	}
}

// TestTickOrStop_StopClosedConcurrentlyWithTick exercises the concurrent
// close-then-tick interleaving.
//
// NOTE (§11.4.6): this does NOT pin check (3). It was written believing it did;
// paired mutation proved otherwise — deleting the residual re-check leaves this
// test green, because the blocking select almost always wakes on stop long
// before the tick is published, so the tick branch that check (3) guards is
// essentially never taken here. It is retained as a genuine concurrent-usage
// regression test (a stopped loop must not report "do work"), not as a mutation
// gate. See the coverage map at the top of this file.
func TestTickOrStop_StopClosedConcurrentlyWithTick(t *testing.T) {
	for i := 0; i < tickOrStopIterations; i++ {
		tick := make(chan time.Time, 1)
		stop := make(chan struct{})

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			close(stop)        // stop is closed FIRST ...
			tick <- time.Now() // ... and only then does a tick become available
		}()

		got := TickOrStop(tick, stop)
		wg.Wait()

		if got {
			t.Fatalf("iteration %d: TickOrStop returned true, but stop was closed "+
				"before the tick was ever published", i)
		}
	}
}

// TestTickOrStop_TickProceedsWhileStopOpen is the positive case: with stop open
// a tick must actually authorise a round of work. Without it the guard could be
// "fixed" by always returning false, which would silently freeze every UI
// refresh loop.
func TestTickOrStop_TickProceedsWhileStopOpen(t *testing.T) {
	tick := make(chan time.Time, 1)
	tick <- time.Now()
	stop := make(chan struct{}) // open

	if !TickOrStop(tick, stop) {
		t.Fatal("TickOrStop returned false for a pending tick with stop still " +
			"open — refresh loops would never run")
	}
}

// TestTickOrStop_NilStopNeverStops documents the nil-channel contract: a nil
// stop blocks forever, so the loop simply keeps ticking rather than panicking.
// Call sites whose app has no stop channel wired rely on this.
func TestTickOrStop_NilStopNeverStops(t *testing.T) {
	tick := make(chan time.Time, 1)
	tick <- time.Now()

	if !TickOrStop(tick, nil) {
		t.Fatal("TickOrStop with a nil stop channel must still honour ticks")
	}
}

// TestSyncSerializesAgainstDo proves the claim the whole package rests on: a
// background goroutine dispatching through Do is mutually exclusive with a
// renderer calling Sync. Under `-race` an unsynchronized implementation trips
// the detector on the shared counter; without `-race` a lost update still
// fails the final count.
//
// This runs on the fyne TEST driver, which is precisely the driver that offers
// no serialization of its own — so it is the case that actually needs proving.
func TestSyncSerializesAgainstDo(t *testing.T) {
	newTestAppForUIThread(t)

	const rounds = 500
	shared := 0

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			DoAndWait(func() { shared++ })
		}
	}()

	observed := 0
	for i := 0; i < rounds; i++ {
		Sync(func() { observed = shared })
	}
	wg.Wait()

	Sync(func() { observed = shared })
	if observed != rounds {
		t.Fatalf("shared counter = %d, want %d — Do/Sync did not serialize", observed, rounds)
	}
}
