//go:build !nogui

// Exercises GUI-only symbols declared in the !nogui-tagged sources
// (theme.go / main.go). Without this constraint the file is compiled under
// -tags=nogui, where those symbols do not exist. NOTE (§11.4.6 honesty):
// this whole package transitively imports go-gl/glfw, which requires X11
// development headers to build; on a host without them (`fatal error:
// X11/Xlib.h: No such file or directory`) this file cannot be compiled or
// run at all, a pre-existing, documented host limitation unrelated to this
// fix. See the fix commit / PR description for what WAS built and run
// (internal/discovery's identical fix, fully verified with `go build` +
// `go test -race`).
package main

// TestDesktopApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick is the
// §11.4.115 RED->GREEN polarity-switch regression guard for one of four
// confirmed instances of the "unprioritized select" defect class audited
// this session (siblings: internal/persistence/store.go autoSaveTick,
// internal/discovery/client.go discoverTimeoutSelect, and the identical fix
// in applications/harmony_os + applications/aurora_os).
//
// THE DEFECT (pre-fix startDataUpdates, inlined before the updateLoopTick
// extraction):
//
//	select {
//	case <-da.updateTicker.C:
//	    da.refreshData()
//	case <-da.stopUpdate:
//	    da.updateTicker.Stop()
//	    return
//	}
//
// Go's `select` chooses UNIFORMLY AT RANDOM among ALL cases ready at the
// instant it is evaluated. When Close() closes da.stopUpdate at nearly the
// same instant the 5s interval ticker has ALSO delivered a pending tick to
// updateTicker.C -- a real occurrence under host scheduler load -- the
// random pick can still choose the ticker.C branch and run one more
// refreshData() EVEN THOUGH the caller already believes the app is closed.
//
// startDataUpdates's per-iteration select logic is extracted into the small
// helper updateLoopTick(tickerC, stop) (identical behaviour, same select
// statement) purely so this test can force the race DETERMINISTICALLY per
// §11.4.115 -- rather than relying on a lucky scheduler stall -- by
// pre-buffering a tick into a buffered ticker-shaped channel AND pre-closing
// the stop channel BEFORE updateLoopTick (and its internal select) is ever
// invoked, so BOTH cases are PROVABLY ready at the exact instant the select
// is evaluated, on every single trial, with zero timing dependency.
//
//   - RED_MODE=1 (meaningful ONLY against the historical PRE-FIX shape of
//     startDataUpdates -- i.e. with updateLoopTick's priority pre-check +
//     inner re-check temporarily removed by hand, reverting to a single
//     naked select): asserts at least one of many forced-race trials has
//     the ticker.C branch win despite stop already being closed --
//     reproducing the defect. Measured against the sibling
//     internal/discovery/client.go fix (identical select shape, fully
//     buildable/testable on a host without X11) with the equivalent naked
//     select temporarily restored: ~50% of forced trials (2500/5000, see
//     that package's client_racefix_test.go + the fix's commit message /
//     PR description for the captured terminal output) -- this package's
//     select is structurally identical so the same ~50% is expected here.
//     Against the SHIPPED (fixed) updateLoopTick, this assertion is
//     EXPECTED to fail (0 ticker-wins) -- that is the correct, expected
//     outcome once the fix is in place; RED_MODE is a manual verification
//     aid for reproducing the historical defect on demand, not an
//     automatically-switched code path (mirrors
//     internal/persistence/store_autosave_select_priority_test.go's
//     documented convention for this exact situation).
//   - RED_MODE unset / "0" (the DEFAULT, standing GREEN guard): asserts
//     EVERY trial has stop win (updateLoopTick returns false, refreshData is
//     NEVER called), because the fix's non-blocking priority pre-check on
//     stop makes Close win unconditionally before the blocking select that
//     races ticker.C is ever reached.
//
// Run GREEN guard (default, on fixed code, on a host WITH X11 dev headers): go test -race -run TestDesktopApp_UpdateLoopTick ./applications/desktop/...
// Run RED reproduction (pre-check + inner re-check manually removed first): RED_MODE=1 go test -race -run TestDesktopApp_UpdateLoopTick ./applications/desktop/...
import (
	"os"
	"testing"
	"time"
)

// redMode reports whether the polarity switch is in reproduce-the-defect
// mode. Default (unset / "0") is the GREEN standing regression guard so a
// bare `go test` on the fixed artifact stays GREEN (§11.4.135).
func redMode() bool {
	return os.Getenv("RED_MODE") == "1"
}

func TestDesktopApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick(t *testing.T) {
	da := &DesktopApp{stopUpdate: make(chan struct{})}

	const trials = 5000
	tickWon := 0

	for i := 0; i < trials; i++ {
		// Deterministically construct "both ready": a buffered
		// ticker-shaped channel already holding a value, and an
		// already-closed stop channel -- BEFORE updateLoopTick (and its
		// internal select) ever runs. No sleep, no scheduler luck: every
		// trial is identical and reproducible on demand.
		tickerC := make(chan time.Time, 1)
		tickerC <- time.Now()
		stop := make(chan struct{})
		close(stop)

		if da.updateLoopTick(tickerC, stop) {
			tickWon++
		}
	}

	if redMode() {
		if tickWon == 0 {
			t.Fatalf("RED_MODE=1: expected at least one of %d forced-race trials (ticker.C pre-buffered + stop pre-closed) to have the ticker.C branch win despite stop already being closed (unprioritized select race in updateLoopTick/startDataUpdates); got 0 -- defect did not reproduce under this forcing, this is a FINDING not evidence of a fix", trials)
		}
		t.Logf("RED_MODE=1: reproduced the unprioritized-select defect in %d/%d forced-race trials (~%.1f%%)", tickWon, trials, 100*float64(tickWon)/float64(trials))
	} else {
		if tickWon != 0 {
			t.Fatalf("RED_MODE=0 (GREEN guard): stop MUST always win once closed, regardless of a pending tick; got %d/%d trials where the ticker.C branch won instead", tickWon, trials)
		}
	}
}

// TestDesktopApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect is
// the §11.4.115 review-follow-up regression guard for the RESIDUAL race
// window left open by updateLoopTick's wide-window fix above -- mirrors
// internal/persistence/store_autosave_residual_window_test.go exactly.
//
// Forces close(stop) to land deterministically inside the residual window
// (after the pre-check has observed stop open, before the blocking select is
// evaluated) via the updateLoopTickRaceHook test seam (main.go), so tickerC
// (pre-buffered) and stop (closed by the hook) are BOTH ready by the time
// the blocking select runs, reproducing on every trial what a coincidental
// scheduler stall would otherwise only occasionally produce.
//
//   - RED_MODE=1: meaningful ONLY with the inner re-check inside
//     updateLoopTick's tickerC case manually removed first. Against the
//     shipped (fixed) code this assertion is expected to fail (0 stray
//     applies), which is correct.
//   - RED_MODE unset / "0" (DEFAULT, GREEN guard): asserts EVERY trial has
//     the fixed code return false with refreshData never called.
func TestDesktopApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect(t *testing.T) {
	da := &DesktopApp{stopUpdate: make(chan struct{})}

	const trials = 5000
	strayApplies := 0

	for i := 0; i < trials; i++ {
		tickerC := make(chan time.Time, 1)
		tickerC <- time.Now()
		stop := make(chan struct{})

		da.updateLoopTickRaceHook = func() {
			close(stop)
		}

		strayApplied := da.updateLoopTick(tickerC, stop)
		da.updateLoopTickRaceHook = nil

		if strayApplied {
			strayApplies++
		}
	}

	if redMode() {
		if strayApplies == 0 {
			t.Fatalf("RED_MODE=1: expected at least one of %d forced-race trials (stop closed inside the residual window, tick pre-buffered) to have the tickerC branch chosen despite stop closing concurrently with the decision; got 0 -- only meaningful with the inner re-check manually removed first", trials)
		}
		t.Logf("RED_MODE=1: observed %d/%d stray refreshData calls with the residual-window race forced", strayApplies, trials)
	} else {
		if strayApplies != 0 {
			t.Fatalf("RED_MODE=0 (GREEN guard): the inner re-check inside updateLoopTick's tickerC case MUST catch a stop-close landing in the residual window; got %d/%d trials where refreshData ran anyway", strayApplies, trials)
		}
	}
}

// TestDesktopApp_Close_JoinsUpdateLoopBeforeReturning is the §11.4.115 /
// §11.4.108 JOIN-half regression guard: it proves Close() does not return
// until the background update-loop goroutine has itself confirmed it
// returned (closed updateDone), which is the mechanism that prevents Close's
// da.db / da.agenticTools teardown from ever running concurrently with a
// still-in-flight refreshData() call -- the use-after-close hazard described
// on Close's doc comment ("if refreshData() wins the [select] race it runs
// CONCURRENTLY with DB/tooling teardown").
//
// This drives the REAL da.startDataUpdates() and da.Close() (not a replica):
// a minimally-constructed DesktopApp (matching main_doubleclose_test.go's
// established pattern -- the full app is not unit-constructible because
// NewDesktopApp calls app.New(), which requires a display) so every manager
// field Close()/refreshData() touch is nil, making refreshData() a fast
// nil-guarded no-op -- exercising the REAL lifecycle wiring without
// requiring a live DB/agentic-tools instance.
//
// PRE-FIX equivalent (Close() closing stopUpdate and immediately returning,
// with no updateDone/join at all): the non-blocking check below would be
// flaky -- sometimes finding the loop goroutine not yet scheduled to observe
// the close, i.e. Close() returning while the loop could still be about to
// run one more iteration -- because nothing forces Close() to wait. The fix
// makes it deterministic: Close() unconditionally blocks on <-updateDone (or
// times out per closeJoinTimeout) before returning, so it is *impossible*
// for updateDone to still be open the instant Close() returns.
func TestDesktopApp_Close_JoinsUpdateLoopBeforeReturning(t *testing.T) {
	da := &DesktopApp{stopUpdate: make(chan struct{})}
	da.startDataUpdates()

	if err := da.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// JOIN PROOF: a non-blocking read on updateDone must observe it already
	// closed the instant Close() returns -- proof the loop goroutine has
	// fully exited (and therefore cannot be mid-refreshData(), and cannot
	// race the da.db / da.agenticTools teardown that follows in Close())
	// BEFORE the teardown steps ran.
	select {
	case <-da.updateDone:
	default:
		t.Fatal("updateDone not closed immediately after Close() returned -- Close() did not join the background update loop before returning, so its da.db/da.agenticTools teardown could race a still-running refreshData()")
	}

	// Idempotency: Close() must remain safe to call twice even with the
	// loop already gone (stopOnce + a closed updateDone must not panic or
	// hang on the second call).
	if err := da.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
