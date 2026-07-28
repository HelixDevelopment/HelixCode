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

// TestAuroraApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick is the
// §11.4.115 RED->GREEN polarity-switch regression guard for one of four
// confirmed instances of the "unprioritized select" defect class audited
// this session (siblings: internal/persistence/store.go autoSaveTick,
// internal/discovery/client.go discoverTimeoutSelect, and the identical fix
// in applications/desktop + applications/harmony_os).
//
// THE DEFECT (pre-fix startDataUpdates, inlined before the updateLoopTick
// extraction):
//
//	select {
//	case <-auroraApp.updateTicker.C:
//	    auroraApp.refreshData()
//	    auroraApp.refreshSystemInfo()
//	case <-auroraApp.stopUpdate:
//	    auroraApp.updateTicker.Stop()
//	    return
//	}
//
// This is the site with the sharpest real-world consequence in this batch:
// auroraApp.Close() closes auroraApp.db immediately after closing
// stopUpdate, with NO join -- if the ticker.C branch wins this race, the
// resulting refreshData() call runs CONCURRENTLY with that db teardown
// (fixed separately by the updateDone join in Close(), proven by
// TestAuroraApp_Close_JoinsUpdateLoopBeforeReturning below).
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
// Run GREEN guard (default, on fixed code, on a host WITH X11 dev headers): go test -race -run TestAuroraApp_UpdateLoopTick ./applications/aurora_os/...
// Run RED reproduction (pre-check + inner re-check manually removed first): RED_MODE=1 go test -race -run TestAuroraApp_UpdateLoopTick ./applications/aurora_os/...
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

func TestAuroraApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick(t *testing.T) {
	// systemMonitor must be non-nil: under the documented RED_MODE manual
	// reproduction protocol (priority pre-check removed by hand, see the
	// doc comment above), the tickerC branch calls refreshData() (safe --
	// nil-guarded) followed unconditionally by refreshSystemInfo(), which
	// does auroraApp.systemMonitor.mu.Lock() -- a nil *AuroraSystemMonitor
	// there panics the whole test binary instead of reproducing the race.
	// Under the DEFAULT RED_MODE=0 path this field is never touched (the
	// priority pre-check returns false before either refresh call runs),
	// so this addition changes nothing about the standing GREEN guard --
	// it only makes the documented RED_MODE=1 protocol actually runnable.
	// See main_doubleclose_test.go for the established
	// minimal-construction pattern this mirrors.
	auroraApp := &AuroraApp{
		stopUpdate:    make(chan struct{}),
		systemMonitor: &AuroraSystemMonitor{},
	}

	const trials = 5000
	tickWon := 0

	for i := 0; i < trials; i++ {
		tickerC := make(chan time.Time, 1)
		tickerC <- time.Now()
		stop := make(chan struct{})
		close(stop)

		if auroraApp.updateLoopTick(tickerC, stop) {
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

// TestAuroraApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect
// mirrors internal/persistence/store_autosave_residual_window_test.go: it
// forces close(stop) to land deterministically inside the residual window
// between updateLoopTick's pre-check and its blocking select, via the
// updateLoopTickRaceHook test seam.
func TestAuroraApp_UpdateLoopTick_ResidualWindowStopRaceDuringBlockingSelect(t *testing.T) {
	// systemMonitor must be non-nil for the same reason as in
	// TestAuroraApp_UpdateLoopTick_StopAlwaysWinsOverPendingTick above:
	// the documented RED_MODE=1 manual protocol (inner re-check removed by
	// hand) reaches refreshSystemInfo(), which unconditionally locks
	// auroraApp.systemMonitor.mu. Under the DEFAULT RED_MODE=0 path this
	// field is never touched, so the standing GREEN guard is unaffected.
	auroraApp := &AuroraApp{
		stopUpdate:    make(chan struct{}),
		systemMonitor: &AuroraSystemMonitor{},
	}

	const trials = 5000
	strayApplies := 0

	for i := 0; i < trials; i++ {
		tickerC := make(chan time.Time, 1)
		tickerC <- time.Now()
		stop := make(chan struct{})

		auroraApp.updateLoopTickRaceHook = func() {
			close(stop)
		}

		strayApplied := auroraApp.updateLoopTick(tickerC, stop)
		auroraApp.updateLoopTickRaceHook = nil

		if strayApplied {
			strayApplies++
		}
	}

	if redMode() {
		if strayApplies == 0 {
			t.Fatalf("RED_MODE=1: expected at least one of %d forced-race trials to have the tickerC branch chosen despite stop closing concurrently with the decision; got 0 -- only meaningful with the inner re-check manually removed first", trials)
		}
		t.Logf("RED_MODE=1: observed %d/%d stray refreshData/refreshSystemInfo calls with the residual-window race forced", strayApplies, trials)
	} else {
		if strayApplies != 0 {
			t.Fatalf("RED_MODE=0 (GREEN guard): the inner re-check inside updateLoopTick's tickerC case MUST catch a stop-close landing in the residual window; got %d/%d trials where the refresh pair ran anyway", strayApplies, trials)
		}
	}
}

// TestAuroraApp_Close_JoinsUpdateLoopBeforeReturning is the §11.4.115 /
// §11.4.108 JOIN-half regression guard -- and the USE-AFTER-CLOSE proof
// requested for this site specifically, since it is the one where the task
// analysis calls out the concrete hazard ("Close() closes auroraApp.db
// immediately after closing stopUpdate ... if refreshData() wins the race
// it runs CONCURRENTLY with DB/tooling teardown"). It proves Close() does
// not return until the background update-loop goroutine has itself
// confirmed it returned (closed updateDone), which is the mechanism that
// makes it IMPOSSIBLE for the loop to still be running (and therefore
// impossible for it to touch a torn-down auroraApp.db) once Close()
// proceeds past the join to call auroraApp.db.Close().
//
// Drives the REAL auroraApp.startDataUpdates() and auroraApp.Close() (not a
// replica): a minimally-constructed AuroraApp (matching
// main_doubleclose_test.go's established pattern -- the full app is not
// unit-constructible because NewAuroraApp calls app.New(), which requires a
// display) with the non-nil securityManager Close() unconditionally audits,
// so every manager field refreshData() touches is nil, making it a fast
// nil-guarded no-op.
func TestAuroraApp_Close_JoinsUpdateLoopBeforeReturning(t *testing.T) {
	auroraApp := &AuroraApp{
		stopUpdate:      make(chan struct{}),
		securityManager: NewAuroraSecurityManager(),
	}
	auroraApp.startDataUpdates()

	if err := auroraApp.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// JOIN / USE-AFTER-CLOSE PROOF: a non-blocking read on updateDone must
	// observe it already closed the instant Close() returns -- proof the
	// loop goroutine (the one that would otherwise call refreshData() /
	// refreshSystemInfo() concurrently with auroraApp.db.Close() below) has
	// fully exited BEFORE the db teardown ran.
	select {
	case <-auroraApp.updateDone:
	default:
		t.Fatal("updateDone not closed immediately after Close() returned -- Close() did not join the background update loop before returning, so auroraApp.db.Close() could race a still-running refreshData()/refreshSystemInfo() call (use-after-close)")
	}

	// Idempotency: Close() must remain safe to call twice.
	if err := auroraApp.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
