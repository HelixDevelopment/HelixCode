package persistence

// TestAutoSaveTick_ResidualWindowStopRaceDuringBlockingSelect is the
// §11.4.115 review-follow-up regression guard for the RESIDUAL race window
// left open by autoSaveTick's wide-window fix (see
// store_autosave_select_priority_test.go for that original defect and its
// TestAutoSaveTick_StopAlwaysWinsOverPendingTick guard).
//
// THE RESIDUAL DEFECT (autoSaveTick as it stood immediately after the
// wide-window fix, i.e. WITHOUT the inner re-check inside the tickerC case
// that this commit adds):
//
//	select {
//	case <-stop:          // pre-check: catches stop closed BEFORE this call
//	    return false
//	default:
//	}
//
//	select {
//	case <-tickerC:
//	    if err := s.SaveAll(); err != nil { s.triggerError(err) }
//	    return true
//	case <-stop:
//	    return false
//	}
//
// The non-blocking pre-check only proves stop was open at the instant it
// ran. If close(stop) lands in the gap BETWEEN the pre-check returning and
// the blocking select below being evaluated -- and a tick is already
// buffered on tickerC, a real occurrence when the auto-save interval fires
// at nearly the same moment DisableAutoSave() runs -- both cases are ready
// when the blocking select is evaluated, and Go's uniform-random pick can
// still choose the tickerC branch: one stray SaveAll() after
// DisableAutoSave() has already returned to its caller. The pre-check does
// NOT close this gap; it only shrinks the window from "any time before this
// call" to "this one blocking select's evaluation instant".
//
// FORCING THE RESIDUAL WINDOW DETERMINISTICALLY: unlike the wide-window
// defect (forceable by simply pre-closing stop before the call), this window
// sits INSIDE autoSaveTick, between two statements a caller cannot observe
// or interpose on from outside. The autoSaveTickRaceHook test seam (store.go)
// exists exactly for this: it fires synchronously immediately after the
// pre-check observes stop open, but before the blocking select is reached --
// precisely the residual window -- letting this test close(stop) there on
// every single trial instead of depending on a nanosecond-wide scheduler
// stall to land the same race by luck.
//
// Once the hook closes stop, tickerC (pre-buffered) and stop (just closed)
// are BOTH ready by the time the blocking select is evaluated, so Go picks
// between them uniformly at random (language spec: "a single one that can
// proceed is chosen via a uniform pseudo-random selection"). Over many
// trials that is indistinguishable, statistically, from the real race.
//
//   - RED_MODE=1: this assertion is meaningful ONLY when the inner re-check
//     added inside the tickerC case (the three lines immediately following
//     `case <-tickerC:` in store.go's autoSaveTick) have been temporarily
//     removed by hand first, reverting to the residual-window-vulnerable
//     shape quoted above. Against THAT code, this trial count reliably
//     produces stray SaveAll() calls (autoSaveTick returning true) despite
//     stop having already closed. Against the shipped (fixed) code the
//     RED_MODE=1 assertion will itself fail with 0 stray applies -- which is
//     the expected, correct outcome once the fix is back in place; RED_MODE
//     is a manual verification aid for reproducing the defect on demand, not
//     an automatically-switched code path.
//   - RED_MODE unset / "0" (the DEFAULT, standing GREEN guard): asserts
//     EVERY trial has the fixed code return false with SaveAll never called,
//     because the inner re-check inside the tickerC case observes the
//     just-closed stop and bails before calling SaveAll, regardless of which
//     branch Go's random pick chose at the outer select.
//
// Run GREEN guard (default, on fixed code): go test -race -run TestAutoSaveTick_ResidualWindow ./internal/persistence/...
// Run RED reproduction (inner re-check manually removed first): RED_MODE=1 go test -race -run TestAutoSaveTick_ResidualWindow ./internal/persistence/...
import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAutoSaveTick_ResidualWindowStopRaceDuringBlockingSelect(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	const trials = 5000
	strayApplies := 0

	for i := 0; i < trials; i++ {
		tickerC := make(chan time.Time, 1)
		tickerC <- time.Now()
		stop := make(chan struct{})

		// Land close(stop) deterministically inside the residual window: the
		// hook fires right after the pre-check has already observed stop
		// open, and right before the blocking select below is evaluated --
		// by which time both tickerC (pre-buffered above) and stop (closed
		// by the hook) are ready, reproducing on every trial the race a
		// coincidental scheduler stall would otherwise only occasionally
		// produce.
		store.autoSaveTickRaceHook = func() {
			close(stop)
		}

		strayApplied := store.autoSaveTick(tickerC, stop)
		store.autoSaveTickRaceHook = nil

		if strayApplied {
			strayApplies++
		}
	}

	if redMode() {
		require.Greaterf(t, strayApplies, 0,
			"RED_MODE=1: expected at least one of %d forced-race trials (stop closed inside the residual window between the pre-check and the blocking select, tick pre-buffered) to have the tickerC branch chosen and SaveAll run despite stop closing concurrently with the decision; got 0 -- this assertion is only meaningful with the inner re-check (store.go autoSaveTick, inside the tickerC case) manually removed first; with the re-check present, 0 is the correct and expected result",
			trials)
		t.Logf("RED_MODE=1: observed %d/%d stray SaveAll calls with the residual-window race forced", strayApplies, trials)
	} else {
		require.Equalf(t, 0, strayApplies,
			"RED_MODE=0 (GREEN guard): the inner re-check inside autoSaveTick's tickerC case MUST catch a stop-close landing in the residual window (between the pre-check and the blocking select's evaluation); got %d/%d trials where SaveAll ran anyway despite stop already being closed",
			strayApplies, trials)
	}
}
