package unit

// HXC-203 — standing regression guard for the LocalLLMManager provider-status
// data race (release blocker for helix-code-1.2.0-dev-0.0.1).
//
// # THE DEFECT (three compounding parts, all captured on the pre-fix artifact)
//
//  1. GetProviderStatus was named as a query but MUTATED shared state: it
//     overwrote each provider's Status and stamped LastCheck on every call.
//  2. LocalLLMManager carried NO mutex of any kind, so two concurrent callers
//     wrote over each other. The pre-fix race detector report:
//
//     WARNING: DATA RACE
//     Write at 0x00c000332fe8 by goroutine 80:
//     (*LocalLLMManager).GetProviderStatus()  local_llm_manager.go:613
//     Previous write at 0x00c000332fe8 by goroutine 75:
//     (*LocalLLMManager).GetProviderStatus()  local_llm_manager.go:613
//     (*LocalLLMManager).GetRunningProviders() local_llm_manager.go:621
//
//  3. It returned m.providers — the LIVE internal map — so every caller got
//     pointers into manager state that it could also mutate. The race the
//     detector caught was therefore only the instance that happened to be
//     exercised.
//
// # POLARITY SWITCH — RED_MODE (§11.4.115, repo convention)
//
// A data race has no in-process assertion API: the verdict is the race
// detector's own report plus the non-zero exit code. So, exactly as in
// applications/harmony_os/gui_thread_race_test.go, the polarity is carried by
// WHICH synchronization is removed, and the evidence is the -race output.
// Both directions are real and they falsify different halves:
//
//   - RED_MODE unset / "0" (DEFAULT — the standing GREEN regression guard,
//     §11.4.135): drives the REAL manager's GetProviderStatus and
//     GetRunningProviders concurrently. On the shipped artifact this is
//     race-free. It goes RED on a PRODUCTION regression — if anyone drops the
//     locking out of the status path, or returns the live providers map
//     again. That is the §11.4.115 RED-on-the-broken-artifact direction, and
//     the one that guards the fix. Captured going RED on the pre-fix artifact
//     in docs/qa/hxc203_localllm_race_*/red_baseline_prefix.log.
//
//   - RED_MODE=1 (harness self-validation — the §11.4.107(10) golden-bad
//     fixture): concurrently performs the SAME unsynchronized write the
//     pre-fix code performed at :613 — `provider.LastCheck = time.Now()` on a
//     shared *llm.LocalLLMProvider — with no lock. This provokes the same
//     defect class on ANY artifact, including the fixed one, and so proves
//     the harness can actually SEE an unsynchronized write to this data
//     class: that a GREEN result means "no race", not "blind test".
//     EXPECTED OUTCOME under -race: a DATA RACE report and a non-zero exit.
//     If RED_MODE=1 ever completes CLEANLY under -race, this guard has lost
//     its teeth and every GREEN result from it is worthless.
//
// Run the GREEN guard (default):
//
//	go test -race -count=3 -run TestLocalLLMManager_HXC203 ./tests/unit/
//
// Run the harness self-validation (expect a DATA RACE + non-zero exit):
//
//	RED_MODE=1 go test -race -count=1 -run TestLocalLLMManager_HXC203_HarnessSelfValidation ./tests/unit/

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hxc203RedMode() bool { return os.Getenv("RED_MODE") == "1" }

// TestLocalLLMManager_HXC203_ConcurrentStatusIsRaceFree is the standing GREEN
// guard. It hammers the two exported readers that raced, from many goroutines
// at once, on a real initialized manager.
func TestLocalLLMManager_HXC203_ConcurrentStatusIsRaceFree(t *testing.T) {
	manager := llm.NewLocalLLMManager(t.TempDir())
	manager.SetSkipProviderInstall(true)
	ctx := context.Background()

	require.NoError(t, manager.Initialize(ctx))

	const goroutines = 16
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Both arms of the captured race stack: the direct call, and
				// the call reached through GetRunningProviders.
				status := manager.GetProviderStatus(ctx)
				assert.NotNil(t, status)

				// Read the fields the pre-fix code wrote, so a reader is
				// genuinely touching Status/LastCheck concurrently with the
				// refresh a sibling goroutine is performing.
				for _, provider := range status {
					_ = provider.Status
					_ = provider.LastCheck
				}

				running := manager.GetRunningProviders(ctx)
				assert.NotNil(t, running)
			}
		}()
	}
	wg.Wait()
}

// TestLocalLLMManager_HXC203_StatusIsNotLiveInternalState guards defect (3)
// with a real in-process assertion, so this half of the fix is falsifiable
// even without the race detector: the returned map must be a private copy,
// not the manager's live state.
func TestLocalLLMManager_HXC203_StatusIsNotLiveInternalState(t *testing.T) {
	manager := llm.NewLocalLLMManager(t.TempDir())
	manager.SetSkipProviderInstall(true)
	ctx := context.Background()

	require.NoError(t, manager.Initialize(ctx))

	first := manager.GetProviderStatus(ctx)
	require.NotEmpty(t, first, "manager should expose its provider definitions")

	// Pick any provider and corrupt the caller's copy the way a careless
	// consumer would.
	var name string
	for candidate := range first {
		name = candidate
		break
	}
	require.NotEmpty(t, name)

	first[name].Status = "corrupted-by-caller"
	first[name].DefaultPort = -1
	delete(first, name)

	second := manager.GetProviderStatus(ctx)

	require.Contains(t, second, name,
		"deleting from the returned map must not deregister the provider — "+
			"GetProviderStatus is handing back the LIVE internal map (HXC-203 defect 3)")
	assert.NotEqual(t, "corrupted-by-caller", second[name].Status,
		"mutating a returned record must not reach manager state — "+
			"GetProviderStatus is handing back live pointers (HXC-203 defect 3)")
	assert.NotEqual(t, -1, second[name].DefaultPort,
		"mutating a returned record must not reach manager state (HXC-203 defect 3)")

	// Two calls must not alias each other either.
	third := manager.GetProviderStatus(ctx)
	assert.NotSame(t, second[name], third[name],
		"successive calls must return independent records, not the same pointer")
}

// TestLocalLLMManager_HXC203_HarnessSelfValidation is the §11.4.107(10)
// golden-bad fixture. Under RED_MODE=1 it reproduces the pre-fix write
// verbatim — concurrent unsynchronized `provider.LastCheck = time.Now()` on a
// shared record — which MUST be reported by the race detector. Its purpose is
// to prove the GREEN guard above is capable of failing.
func TestLocalLLMManager_HXC203_HarnessSelfValidation(t *testing.T) {
	if !hxc203RedMode() {
		t.Skip("SKIP-OK: #HXC-203 golden-bad fixture; runs only under RED_MODE=1 " +
			"(it deliberately provokes a data race to prove this harness can see one)")
	}

	// A single shared record, exactly the shape GetProviderStatus used to
	// mutate through the live map it returned.
	shared := &llm.LocalLLMProvider{Name: "hxc203-golden-bad", Status: "running"}

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				// local_llm_manager.go:613, pre-fix, with no lock.
				shared.LastCheck = time.Now()
				shared.Status = "running"
			}
		}()
	}
	wg.Wait()

	t.Log("RED_MODE=1: drove the pre-fix unsynchronized write concurrently; " +
		"under -race this MUST have produced a DATA RACE report and a non-zero exit. " +
		"A clean run here means the harness is blind.")
}
