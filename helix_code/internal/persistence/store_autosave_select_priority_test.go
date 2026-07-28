package persistence

// TestAutoSaveTick_StopAlwaysWinsOverPendingTick is the §11.4.115
// RED->GREEN polarity-switch regression guard for the THIRD confirmed
// instance of the "unprioritized select" defect class in this codebase.
// Siblings already fixed this same session: helix_agent's
// internal/llm/lazy_provider.go createProviderWithContext, and
// internal/database/debate_log_repository.go StartCleanupWorker's cleanup
// loop.
//
// THE DEFECT (pre-fix autoSaveLoop, inlined before the autoSaveTick
// extraction below):
//
//	select {
//	case <-ticker.C:
//	    if err := s.SaveAll(); err != nil { s.triggerError(err) }
//	case <-stop:
//	    return
//	}
//
// Go's `select` chooses UNIFORMLY AT RANDOM among ALL cases ready at the
// instant it is evaluated. When DisableAutoSave() closes `stop` at nearly
// the same instant the interval ticker has ALSO delivered a pending tick to
// ticker.C -- a real occurrence under host scheduler load, where the loop's
// goroutine is delayed long enough for BOTH events to have already happened
// by the time it is next scheduled -- the random pick can still choose the
// ticker.C branch and run one more SaveAll() EVEN THOUGH the caller already
// believes auto-save is disabled. That stray save is exactly the symptom
// captured by TestAutoSaveLifecycle_ReEnableTicks's "disabled auto-save must
// not tick" assertion (store_autosave_lifecycle_test.go), which was observed
// failing under host load with a ~5.19ms drift on a 60ms disabled-window
// check against a 20ms ticker.
//
// autoSaveLoop's per-iteration select logic is extracted into the small
// helper autoSaveTick(tickerC, stop) (identical behaviour, same select
// statement, zero change to autoSaveLoop's observable behaviour) purely so
// this test can force the race DETERMINISTICALLY per §11.4.115 -- rather
// than relying on a lucky scheduler stall -- by pre-buffering a tick into a
// buffered ticker-shaped channel AND pre-closing the stop channel BEFORE
// autoSaveTick (and its internal select) is ever invoked, so BOTH cases are
// PROVABLY ready at the exact instant the select is evaluated, on every
// single trial, with zero timing dependency.
//
//   - RED_MODE=1 (run against the PRE-FIX code): asserts at least one of
//     many forced-race trials has the ticker.C branch win (SaveAll runs,
//     autoSaveTick returns true) despite stop already being closed --
//     reproducing the defect.
//   - RED_MODE unset / "0" (the DEFAULT, standing GREEN guard): asserts
//     EVERY trial has stop win (autoSaveTick returns false, SaveAll is
//     NEVER called), because the fix's non-blocking priority pre-check on
//     `stop` makes disable win unconditionally before the blocking select
//     that races ticker.C is ever reached.
//
// Run GREEN guard (default, on fixed code): go test -race -run TestAutoSaveTick ./internal/persistence/...
// Run RED reproduction (on broken code):    RED_MODE=1 go test -race -run TestAutoSaveTick ./internal/persistence/...
import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAutoSaveTick_StopAlwaysWinsOverPendingTick(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	const trials = 5000
	tickWon := 0

	for i := 0; i < trials; i++ {
		// Deterministically construct "both ready": a buffered
		// ticker-shaped channel already holding a value, and an
		// already-closed stop channel -- BEFORE autoSaveTick (and its
		// internal select) ever runs. No sleep, no scheduler luck: every
		// trial is identical and reproducible on demand.
		tickerC := make(chan time.Time, 1)
		tickerC <- time.Now()
		stop := make(chan struct{})
		close(stop)

		if store.autoSaveTick(tickerC, stop) {
			tickWon++
		}
	}

	if redMode() {
		require.Greaterf(t, tickWon, 0,
			"RED_MODE=1: expected at least one of %d forced-race trials (ticker.C pre-buffered + stop pre-closed) to have the ticker.C branch win despite stop already being closed (unprioritized select race in autoSaveTick/autoSaveLoop); got 0 -- defect did not reproduce under this forcing, this is a FINDING not evidence of a fix",
			trials)
		t.Logf("RED_MODE=1: reproduced the unprioritized-select defect in %d/%d forced-race trials", tickWon, trials)
	} else {
		require.Equalf(t, 0, tickWon,
			"RED_MODE=0 (GREEN guard): stop MUST always win once closed, regardless of a pending tick; got %d/%d trials where the ticker.C branch won instead",
			tickWon, trials)
	}
}
