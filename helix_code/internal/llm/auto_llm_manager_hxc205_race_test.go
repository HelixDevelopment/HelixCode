package llm

// HXC-205 — standing regression guard for the AutoLLMManager provider-state
// data race (release blocker for helix-code-1.2.0-dev-0.0.1).
//
// # THE DEFECT
//
// AutoLLMManager DECLARES a sync.RWMutex (auto_llm_manager.go:84) and uses it
// in seven methods — SetMetricsRecorder, Initialize, Start, GetStatus,
// GetRunningEndpoints, Stop, and a partial RLock in updatePerformanceMetrics.
// It then writes provider health, process handles and last-checked timestamps
// from NINE other methods that take no lock at all:
//
//	autoInstallAllProviders     Status, LastHealthCheck        :474 :475
//	autoConfigureProvider       Config                         :566
//	autoStartProvider           Process, Status, LastHealthCheck :643 :644 :645 :655
//	autoHealthCheck             Health.*, RetryCount, LastHealthCheck :762-:782
//	autoRecoverProvider         Process                        :816
//	autoUpdateProvider          Process                        :965
//	updatePerformanceMetrics    Metrics.*                      :880 :897-:903
//	runBackgroundTask           task.IsRunning, task.LastRun   :728 :735 :739 :746
//	HealthMonitor.updateProviderHealth  Health.*, RetryCount   health_monitor.go:107-:121
//
// That is worse than carrying no lock at all: a reader sees synchronization
// and reasonably assumes it applies throughout.
//
// Two writers reach the SAME *HealthStatus concurrently in production. When
// AutoMonitor is enabled, Initialize starts the "health" background task
// (-> autoHealthCheck) and Start launches the HealthMonitor goroutine
// (-> performHealthChecks). Both stamp Health.LastCheck / ResponseTime /
// IsHealthy / Status / Error on the same record, with nothing ordering them,
// while GetStatus concurrently value-copies that record for the CLI and the
// load balancer. A verdict can therefore be applied field-by-field and
// interleaved with another verdict, so a provider is reported healthy when it
// was found stopped, or stopped when it was found running.
//
// A third, compounding part: GetStatus returned `providerCopy := *v`, a
// SHALLOW copy. Health, Metrics and Config are pointer/map fields, so every
// caller received live references into manager state that it could mutate.
//
// # POLARITY SWITCH — RED_MODE (§11.4.115, repo convention)
//
// A data race has no in-process assertion API: the verdict is the race
// detector's own report plus the non-zero exit code. So, exactly as in
// tests/unit/local_llm_manager_hxc203_race_test.go, the polarity is carried by
// WHICH synchronization is removed, and the evidence is the -race output.
//
//   - RED_MODE unset / "0" (DEFAULT — the standing GREEN regression guard,
//     §11.4.135): drives the REAL production concurrency — autoHealthCheck and
//     HealthMonitor.performHealthChecks writing the same records while
//     GetStatus reads them. On the shipped artifact this is race-free. It goes
//     RED if anyone drops the locking back out of the provider-state path.
//     Captured going RED on the pre-fix artifact in
//     docs/qa/hxc205_autollm_race_*/red_baseline_prefix.log.
//
//   - RED_MODE=1 (harness self-validation — the §11.4.107(10) golden-bad
//     fixture): concurrently performs the SAME unsynchronized write the
//     pre-fix code performed at :762 — `provider.Health.LastCheck = time.Now()`
//     on a shared *AutoProvider — with no lock. This provokes the defect class
//     on ANY artifact, including the fixed one, and so proves the harness can
//     actually SEE an unsynchronized write to this data class: that a GREEN
//     result means "no race", not "blind test".
//     EXPECTED OUTCOME under -race: a DATA RACE report and a non-zero exit.
//     If RED_MODE=1 ever completes CLEANLY under -race, this guard has lost
//     its teeth and every GREEN result from it is worthless.
//
// # WHY THIS GUARD IS IN-PACKAGE
//
// Unlike the HXC-203 twin, AutoLLMManager exposes no way to register a
// provider without Initialize, and Initialize spawns autoInstallAllProviders,
// which shells out to `git clone` for every provider definition. Seeding
// m.providers directly — the convention already used by
// health_monitor_test.go:288 and auto_llm_manager_test.go — keeps this guard
// hermetic, network-free and fast, and lets it drive the two REAL concurrent
// writers rather than a proxy for them. No production API was widened to make
// the type testable.
//
// Run the GREEN guard (default):
//
//	go test -race -count=3 -timeout 180s -run TestAutoLLMManager_HXC205 ./internal/llm/
//
// Run the harness self-validation (expect a DATA RACE + non-zero exit):
//
//	RED_MODE=1 go test -race -count=1 -timeout 120s \
//	  -run TestAutoLLMManager_HXC205_HarnessSelfValidation ./internal/llm/

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hxc205RedMode() bool { return os.Getenv("RED_MODE") == "1" }

// hxc205SeedManager returns a manager holding `count` providers in the
// "running" state whose health endpoint is a live local server. Providers are
// installed directly into the map because there is no exported path to
// register one without triggering git clones (see the file header).
func hxc205SeedManager(t *testing.T, count int) (*AutoLLMManager, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	manager := NewAutoLLMManager(t.TempDir())
	manager.config.Health.AutoRecovery = false // never fork a real recovery from a unit test
	manager.config.Health.MaxRetries = 3

	for i := 0; i < count; i++ {
		name := "hxc205-provider-" + string(rune('a'+i))
		manager.providers[name] = &AutoProvider{
			LocalLLMProvider: LocalLLMProvider{
				Name:        name,
				HealthURL:   server.URL,
				DefaultPort: 30000 + i,
			},
			Status:  "running",
			Config:  map[string]interface{}{"seeded": true},
			Health:  &HealthStatus{Status: "unknown"},
			Metrics: &PerformanceMetrics{},
		}
	}

	return manager, server
}

// TestAutoLLMManager_HXC205_ConcurrentProviderStateIsRaceFree is the standing
// GREEN guard. It runs the two production health writers concurrently with the
// exported reader, which is precisely the topology Initialize+Start creates.
func TestAutoLLMManager_HXC205_ConcurrentProviderStateIsRaceFree(t *testing.T) {
	if hxc205RedMode() {
		t.Skip("SKIP-OK: #HXC-205 RED_MODE=1 selects the golden-bad fixture only")
	}

	manager, _ := hxc205SeedManager(t, 3)
	monitor := NewHealthMonitor(manager)

	const writers = 4
	const readers = 4
	const iterations = 6

	var wg sync.WaitGroup

	// Writer arm A: the background-task health path (autoHealthCheck).
	wg.Add(writers)
	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				assert.NoError(t, manager.autoHealthCheck())
			}
		}()
	}

	// Writer arm B: the HealthMonitor path (performHealthChecks). Pre-fix this
	// wrote Health.* through the shallow copy GetStatus handed back, landing on
	// the very same words arm A was writing.
	wg.Add(writers)
	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				monitor.performHealthChecks()
			}
		}()
	}

	// Reader arm: the exported surface the CLI and load balancer use.
	wg.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations*3; j++ {
				status := manager.GetStatus()
				for _, provider := range status {
					_ = provider.Status
					_ = provider.LastHealthCheck
					_ = provider.RetryCount
					if provider.Health != nil {
						_ = provider.Health.IsHealthy
						_ = provider.Health.LastCheck
						_ = provider.Health.ResponseTime
					}
					if provider.Metrics != nil {
						_ = provider.Metrics.ErrorRate
					}
				}
				_ = manager.GetRunningEndpoints()
			}
		}()
	}

	wg.Wait()
}

// TestAutoLLMManager_HXC205_GetStatusIsNotLiveInternalState guards the
// shallow-copy half with a real in-process assertion, so it is falsifiable
// WITHOUT the race detector: a caller mutating the returned records must not
// reach manager state.
func TestAutoLLMManager_HXC205_GetStatusIsNotLiveInternalState(t *testing.T) {
	if hxc205RedMode() {
		t.Skip("SKIP-OK: #HXC-205 RED_MODE=1 selects the golden-bad fixture only")
	}

	manager, _ := hxc205SeedManager(t, 2)

	first := manager.GetStatus()
	require.NotEmpty(t, first, "manager should expose its seeded providers")

	var name string
	for candidate := range first {
		name = candidate
		break
	}
	require.NotEmpty(t, name)

	// Corrupt the caller's copy exactly the way a careless consumer would.
	first[name].Status = "corrupted-by-caller"
	first[name].Health.Status = "corrupted-by-caller"
	first[name].Health.IsHealthy = true
	first[name].Metrics.ErrorRate = 99.0
	first[name].Config["poisoned"] = true
	first[name].RetryCount = 4242
	delete(first, name)

	// Read manager state back through the lock, not through the map, so this
	// assertion is about manager state rather than about another copy.
	manager.mutex.RLock()
	live, stillRegistered := manager.providers[name]
	require.True(t, stillRegistered,
		"deleting from the returned map must not deregister the provider (HXC-205)")
	liveStatus := live.Status
	liveHealthStatus := live.Health.Status
	liveHealthy := live.Health.IsHealthy
	liveErrorRate := live.Metrics.ErrorRate
	_, poisoned := live.Config["poisoned"]
	liveRetry := live.RetryCount
	manager.mutex.RUnlock()

	assert.NotEqual(t, "corrupted-by-caller", liveStatus,
		"mutating a returned record must not reach manager state (HXC-205)")
	assert.NotEqual(t, "corrupted-by-caller", liveHealthStatus,
		"GetStatus returned a SHALLOW copy — Health is a shared pointer (HXC-205)")
	assert.False(t, liveHealthy,
		"a caller flipping Health.IsHealthy must not change what the manager reports (HXC-205)")
	assert.NotEqual(t, 99.0, liveErrorRate,
		"GetStatus returned a SHALLOW copy — Metrics is a shared pointer (HXC-205)")
	assert.False(t, poisoned,
		"GetStatus returned a SHALLOW copy — Config is a shared map (HXC-205)")
	assert.NotEqual(t, 4242, liveRetry,
		"mutating a returned record must not reach manager state (HXC-205)")

	// Successive calls must not alias one another either.
	second := manager.GetStatus()
	third := manager.GetStatus()
	require.Contains(t, second, name)
	assert.NotSame(t, second[name], third[name],
		"successive calls must return independent records, not the same pointer")
	assert.NotSame(t, second[name].Health, third[name].Health,
		"successive calls must return independent Health records")
	assert.NotSame(t, second[name].Metrics, third[name].Metrics,
		"successive calls must return independent Metrics records")
}

// TestAutoLLMManager_HXC205_HealthVerdictStillReachesManager pins the half of
// the contract the deep copy could silently destroy. HealthMonitor exists to
// WRITE health verdicts back into manager state; pre-fix it did so by
// accident, through the shallow copy's aliased Health pointer. If the fix
// deep-copies GetStatus without giving the monitor a real write path, health
// monitoring becomes a no-op that still looks green. This asserts the verdict
// lands, and that RetryCount — which pre-fix was written to a throwaway copy
// and lost every cycle — now persists.
func TestAutoLLMManager_HXC205_HealthVerdictStillReachesManager(t *testing.T) {
	if hxc205RedMode() {
		t.Skip("SKIP-OK: #HXC-205 RED_MODE=1 selects the golden-bad fixture only")
	}

	// A server that is up: the verdict must be recorded as healthy.
	manager, _ := hxc205SeedManager(t, 1)
	monitor := NewHealthMonitor(manager)

	var name string
	manager.mutex.RLock()
	for candidate := range manager.providers {
		name = candidate
	}
	manager.mutex.RUnlock()
	require.NotEmpty(t, name)

	monitor.performHealthChecks()

	status := manager.GetStatus()
	require.Contains(t, status, name)
	assert.True(t, status[name].Health.IsHealthy,
		"HealthMonitor.performHealthChecks must write its verdict into manager state")
	assert.Equal(t, "healthy", status[name].Health.Status,
		"HealthMonitor.performHealthChecks must write its verdict into manager state")
	assert.False(t, status[name].Health.LastCheck.IsZero(),
		"the health verdict must carry a real timestamp")

	// Now point the provider at a dead endpoint and confirm the failure verdict
	// AND the retry counter both persist across calls.
	manager.mutex.Lock()
	manager.providers[name].HealthURL = "http://127.0.0.1:1/dead"
	manager.mutex.Unlock()

	monitor.performHealthChecks()
	monitor.performHealthChecks()

	status = manager.GetStatus()
	assert.False(t, status[name].Health.IsHealthy,
		"a failing probe must be recorded in manager state")
	assert.Equal(t, "unhealthy", status[name].Health.Status)
	assert.GreaterOrEqual(t, status[name].RetryCount, 2,
		"RetryCount must accumulate in manager state across cycles — pre-fix it was "+
			"written to the throwaway copy GetStatus returned and lost every cycle (HXC-205)")
}

// TestAutoLLMManager_HXC205_HarnessSelfValidation is the §11.4.107(10)
// golden-bad fixture. Under RED_MODE=1 it reproduces the pre-fix write
// verbatim — concurrent unsynchronized `provider.Health.LastCheck = time.Now()`
// on a shared *AutoProvider — which MUST be reported by the race detector. Its
// purpose is to prove the GREEN guards above are capable of failing.
func TestAutoLLMManager_HXC205_HarnessSelfValidation(t *testing.T) {
	if !hxc205RedMode() {
		t.Skip("SKIP-OK: #HXC-205 golden-bad fixture; runs only under RED_MODE=1 " +
			"(it deliberately provokes a data race to prove this harness can see one)")
	}

	// A single shared record, exactly the shape autoHealthCheck and
	// HealthMonitor.updateProviderHealth used to mutate with no lock held.
	shared := &AutoProvider{
		LocalLLMProvider: LocalLLMProvider{Name: "hxc205-golden-bad"},
		Status:           "running",
		Health:           &HealthStatus{},
		Metrics:          &PerformanceMetrics{},
	}

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				// auto_llm_manager.go:762-:764 and health_monitor.go:107-:109,
				// pre-fix, with no lock.
				shared.Health.LastCheck = time.Now()
				shared.Health.ResponseTime = j
				shared.Health.IsHealthy = true
				shared.LastHealthCheck = time.Now()
				shared.RetryCount++
			}
		}()
	}
	wg.Wait()

	t.Log("RED_MODE=1: drove the pre-fix unsynchronized write concurrently; " +
		"under -race this MUST have produced a DATA RACE report and a non-zero exit. " +
		"A clean run here means the harness is blind.")
}
