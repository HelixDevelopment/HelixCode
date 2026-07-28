package discovery

// TestDiscoverTimeoutSelect_ResultAlwaysWinsOverSpuriousTimeout is the
// §11.4.115 RED->GREEN polarity-switch regression guard for the FOURTH
// confirmed instance of the "unprioritized select" defect class audited this
// session, after internal/persistence/store.go autoSaveTick, helix_agent's
// internal/llm/lazy_provider.go createProviderWithContext, and the
// applications/{desktop,harmony_os,aurora_os} background-update-loop
// teardown races (see main_racefix_test.go in each of those packages).
//
// THE DEFECT (pre-fix DiscoverWithTimeout, inlined before the
// discoverTimeoutSelect extraction below):
//
//	select {
//	case result := <-resultChan:
//	    return result, nil
//	case err := <-errorChan:
//	    return nil, err
//	case <-time.After(timeout):
//	    return nil, fmt.Errorf(...timeout...)
//	}
//
// Go's `select` chooses UNIFORMLY AT RANDOM among ALL cases ready at the
// instant it is evaluated. If the background discovery goroutine's answer
// becomes ready at (scheduler-)nearly the same instant the timeout fires --
// a real occurrence under host scheduler load, where this goroutine is
// delayed long enough for both events to have already happened by the time
// it is next scheduled -- the random pick can still choose the timeout
// branch and discard a REAL discovered result (or a real discovery error) in
// favour of a spurious timeout error.
//
// DiscoverWithTimeout's select decision is extracted into the small helper
// discoverTimeoutSelect(resultChan, errorChan, timeoutC, ...) (identical
// behaviour, same select statement, zero change to DiscoverWithTimeout's
// observable behaviour) purely so this test can force the race
// DETERMINISTICALLY per §11.4.115 -- rather than relying on a lucky
// scheduler stall -- by pre-buffering a real result into a buffered
// result-shaped channel AND pre-firing a closed timeout-shaped channel
// BEFORE discoverTimeoutSelect (and its internal select) is ever invoked, so
// BOTH cases are PROVABLY ready at the exact instant the select is
// evaluated, on every single trial, with zero timing dependency.
//
//   - RED_MODE=1 (meaningful ONLY against the historical PRE-FIX shape of
//     DiscoverWithTimeout -- i.e. with the priority pre-check + inner
//     re-check temporarily removed by hand, reverting discoverTimeoutSelect
//     to a single naked select): asserts at least one of many forced-race
//     trials has the timeout branch win (a real, already-ready result is
//     discarded) despite a real answer being ready at the same instant --
//     reproducing the defect. Measured against the genuinely-reverted
//     naked-select shape during this fix's development: ~50% of forced
//     trials (matching the theoretical uniform 2-of-N-ready-cases odds) --
//     see the fix commit message / PR description for the captured
//     terminal output. Against the SHIPPED (fixed) discoverTimeoutSelect,
//     this assertion is EXPECTED to fail (0 timeout-wins) -- that is the
//     correct, expected outcome once the fix is in place; RED_MODE is a
//     manual verification aid for reproducing the historical defect on
//     demand, not an automatically-switched code path (mirrors
//     internal/persistence/store_autosave_residual_window_test.go's
//     documented convention for this exact situation).
//   - RED_MODE unset / "0" (the DEFAULT, standing GREEN guard): asserts
//     EVERY trial has the real result win, because the fix's non-blocking
//     priority pre-check on resultChan/errorChan makes a ready real answer
//     win unconditionally before the blocking select that races the timeout
//     is ever reached.
//
// Run GREEN guard (default, on fixed code): go test -race -run TestDiscoverTimeoutSelect ./internal/discovery/...
// Run RED reproduction (pre-check + inner re-check manually removed first): RED_MODE=1 go test -race -run TestDiscoverTimeoutSelect ./internal/discovery/...
import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// redMode reports whether the polarity switch is in reproduce-the-defect
// mode. Default (unset / "0") is the GREEN standing regression guard so a
// bare `go test` on the fixed artifact stays GREEN (§11.4.135).
func redMode() bool {
	return os.Getenv("RED_MODE") == "1"
}

func TestDiscoverTimeoutSelect_ResultAlwaysWinsOverSpuriousTimeout(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	client := NewDiscoveryClient(DefaultDiscoveryClientConfig(registry, allocator))

	const trials = 5000
	timeoutWon := 0

	for i := 0; i < trials; i++ {
		// Deterministically construct "both ready": a buffered
		// result-shaped channel already holding a real *DiscoveryResult,
		// and an already-fired timeout-shaped channel -- BEFORE
		// discoverTimeoutSelect (and its internal select) ever runs. No
		// sleep, no scheduler luck: every trial is identical and
		// reproducible on demand.
		resultChan := make(chan *DiscoveryResult, 1)
		realResult := &DiscoveryResult{
			ServiceInfo: &ServiceInfo{Name: "race-fixture", Host: "localhost", Port: 1, Protocol: "tcp", Healthy: true},
			Strategy:    StrategyDefaultPort,
		}
		resultChan <- realResult
		errorChan := make(chan error, 1)
		timeoutC := make(chan time.Time)
		close(timeoutC) // already "fired"

		got, err := client.discoverTimeoutSelect(resultChan, errorChan, timeoutC, "race-fixture", time.Millisecond)
		if err != nil || got != realResult {
			timeoutWon++
		}
	}

	if redMode() {
		require.Greaterf(t, timeoutWon, 0,
			"RED_MODE=1: expected at least one of %d forced-race trials (real result pre-buffered + timeout pre-fired) to have the timeout branch win despite a real result already being ready; got 0 -- this assertion is only meaningful with discoverTimeoutSelect's priority pre-check + inner re-check manually removed first; with the fix present, 0 is the correct and expected result",
			trials)
		t.Logf("RED_MODE=1: observed %d/%d forced-race trials where the timeout discarded a ready real result (~%.1f%%)", timeoutWon, trials, 100*float64(timeoutWon)/float64(trials))
	} else {
		require.Equalf(t, 0, timeoutWon,
			"RED_MODE=0 (GREEN guard): a real, already-ready discovery result MUST always win over a spurious concurrent timeout; got %d/%d trials where the timeout branch discarded it instead",
			timeoutWon, trials)
	}
}

// TestDiscoverTimeoutSelect_ErrorAlwaysWinsOverSpuriousTimeout mirrors the
// result case above for the errorChan branch (a real discovery ERROR --
// e.g. ErrServiceUnavailable -- must also win over a concurrent spurious
// timeout, not just a successful result).
func TestDiscoverTimeoutSelect_ErrorAlwaysWinsOverSpuriousTimeout(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	client := NewDiscoveryClient(DefaultDiscoveryClientConfig(registry, allocator))

	const trials = 5000
	timeoutWon := 0

	for i := 0; i < trials; i++ {
		resultChan := make(chan *DiscoveryResult, 1)
		errorChan := make(chan error, 1)
		errorChan <- ErrServiceUnavailable
		timeoutC := make(chan time.Time)
		close(timeoutC)

		got, err := client.discoverTimeoutSelect(resultChan, errorChan, timeoutC, "race-fixture", time.Millisecond)
		if got != nil || err != ErrServiceUnavailable {
			timeoutWon++
		}
	}

	if redMode() {
		require.Greaterf(t, timeoutWon, 0,
			"RED_MODE=1: expected at least one forced-race trial to discard a ready real discovery error in favour of the timeout; got 0 -- only meaningful with the fix manually reverted first")
		t.Logf("RED_MODE=1: observed %d/%d forced-race trials where the timeout discarded a ready real error", timeoutWon, trials)
	} else {
		require.Equalf(t, 0, timeoutWon,
			"RED_MODE=0 (GREEN guard): a real, already-ready discovery ERROR MUST always win over a spurious concurrent timeout; got %d/%d trials where the timeout branch discarded it instead",
			timeoutWon, trials)
	}
}
