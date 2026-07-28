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

// TestHarmonyApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick is the
// §11.4.115 RED->GREEN polarity-switch regression guard for one of four
// confirmed instances of the "unprioritized select" defect class audited
// this session (siblings: internal/persistence/store.go autoSaveTick,
// internal/discovery/client.go discoverTimeoutSelect, and the identical fix
// in applications/desktop + applications/aurora_os).
//
// THE DEFECT (pre-fix startDataUpdates, inlined before the updateLoopTick
// extraction):
//
//	select {
//	case <-app.updateTicker.C:
//	    app.refreshData()
//	case <-app.stopUpdate:
//	    app.updateTicker.Stop()
//	    return
//	}
//
// Go's `select` chooses UNIFORMLY AT RANDOM among ALL cases ready at the
// instant it is evaluated. See applications/desktop/main_racefix_test.go
// (identical fix, identical shape) for the full defect narrative.
//
//   - RED_MODE=1 (meaningful ONLY against the historical PRE-FIX shape --
//     priority pre-check + inner re-check manually removed first): asserts
//     at least one forced-race trial has the ticker.C branch win despite
//     stop already being closed. Measured against the sibling
//     internal/discovery/client.go fix (identical select shape, fully
//     buildable/testable on a host without X11) with the equivalent naked
//     select temporarily restored: ~50% of forced trials (2500/5000, see
//     that package's client_racefix_test.go). Against the shipped (fixed)
//     updateLoopTick, this assertion is EXPECTED to fail (0 wins) -- the
//     correct, expected outcome once the fix is in place.
//   - RED_MODE unset / "0" (DEFAULT, GREEN guard): asserts EVERY trial has
//     stop win.
//
// Run GREEN guard (default, on fixed code, on a host WITH X11 dev headers): go test -race -run TestHarmonyApp_UpdateLoopTick ./applications/harmony_os/...
// Run RED reproduction (pre-check + inner re-check manually removed first): RED_MODE=1 go test -race -run TestHarmonyApp_UpdateLoopTick ./applications/harmony_os/...
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

func TestHarmonyApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick(t *testing.T) {
	app := &HarmonyApp{stopUpdate: make(chan struct{})}

	const trials = 5000
	tickWon := 0

	for i := 0; i < trials; i++ {
		tickerC := make(chan time.Time, 1)
		tickerC <- time.Now()
		stop := make(chan struct{})
		close(stop)

		if app.updateLoopTick(tickerC, stop) {
			tickWon++
		}
	}

	if redMode() {
		if tickWon == 0 {
			t.Fatalf("RED_MODE=1: expected at least one of %d forced-race trials to have the ticker.C branch win despite stop already being closed; got 0 -- defect did not reproduce under this forcing", trials)
		}
		t.Logf("RED_MODE=1: reproduced the unprioritized-select defect in %d/%d forced-race trials (~%.1f%%)", tickWon, trials, 100*float64(tickWon)/float64(trials))
	} else {
		if tickWon != 0 {
			t.Fatalf("RED_MODE=0 (GREEN guard): stop MUST always win once closed, regardless of a pending tick; got %d/%d trials where the ticker.C branch won instead", tickWon, trials)
		}
	}
}

// TestHarmonyApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect
// mirrors internal/persistence/store_autosave_residual_window_test.go: it
// forces close(stop) to land deterministically inside the residual window
// between updateLoopTick's pre-check and its blocking select, via the
// updateLoopTickRaceHook test seam.
func TestHarmonyApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect(t *testing.T) {
	app := &HarmonyApp{stopUpdate: make(chan struct{})}

	const trials = 5000
	strayApplies := 0

	for i := 0; i < trials; i++ {
		tickerC := make(chan time.Time, 1)
		tickerC <- time.Now()
		stop := make(chan struct{})

		app.updateLoopTickRaceHook = func() {
			close(stop)
		}

		strayApplied := app.updateLoopTick(tickerC, stop)
		app.updateLoopTickRaceHook = nil

		if strayApplied {
			strayApplies++
		}
	}

	if redMode() {
		if strayApplies == 0 {
			t.Fatalf("RED_MODE=1: expected at least one of %d forced-race trials to have the tickerC branch chosen despite stop closing concurrently with the decision; got 0 -- only meaningful with the inner re-check manually removed first", trials)
		}
		t.Logf("RED_MODE=1: observed %d/%d stray refreshData calls with the residual-window race forced", strayApplies, trials)
	} else {
		if strayApplies != 0 {
			t.Fatalf("RED_MODE=0 (GREEN guard): the inner re-check inside updateLoopTick's tickerC case MUST catch a stop-close landing in the residual window; got %d/%d trials where refreshData ran anyway", strayApplies, trials)
		}
	}
}

// TestHarmonyApp_Cleanup_JoinsUpdateLoopBeforeReturning is the §11.4.115 /
// §11.4.108 JOIN-half regression guard: proves Cleanup() does not return
// until the background update-loop goroutine has itself confirmed it
// returned (closed updateDone) -- the mechanism preventing Cleanup's app.db
// teardown from racing a still-in-flight refreshData() call. See
// applications/desktop/main_racefix_test.go's identical test for the full
// rationale (mirrored here for the Cleanup()/HarmonyApp naming).
func TestHarmonyApp_Cleanup_JoinsUpdateLoopBeforeReturning(t *testing.T) {
	// systemMonitor must be non-nil: Cleanup() unconditionally writes
	// app.systemMonitor.monitoring (see main_doubleclose_test.go for the
	// established minimal-construction pattern this mirrors).
	app := &HarmonyApp{
		stopUpdate:    make(chan struct{}),
		systemMonitor: &HarmonySystemMonitor{},
	}
	app.startDataUpdates()

	app.Cleanup()

	select {
	case <-app.updateDone:
	default:
		t.Fatal("updateDone not closed immediately after Cleanup() returned -- Cleanup() did not join the background update loop before returning, so its app.db teardown could race a still-running refreshData()")
	}

	// Idempotency: Cleanup() must remain safe to call twice.
	app.Cleanup()
}
