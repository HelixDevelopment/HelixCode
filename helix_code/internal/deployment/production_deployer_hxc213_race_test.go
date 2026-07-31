package deployment

// HXC-213 regression guards (§11.4.115 RED→GREEN, §11.4.135 standing suite).
//
// DEFECT. ProductionDeployer DECLARES a sync.RWMutex at production_deployer.go:28
// and holds it at exactly ONE site (addNotification), against 92 pd.status.*
// accesses across the type. That single site is the residue of the HXC-014 fix,
// which guarded only the two lines the race detector happened to point at
// (recorded verbatim in the comment at production_deployer.go:1179-1186) and
// left every other access untouched. The result is worse than carrying no lock:
// the file READS as synchronized, and the package's own concurrent tests
// (production_deployer_chaos_test.go:132-135) already follow the advertised
// protocol — take pd.mutex.RLock() to read pd.status — a discipline that is
// only sound if EVERY writer takes the lock. Writers at :242 :271 :358 :415
// :517 :570 :734 :1078 :1096 :1119 (among others) do not.
//
// Two independent halves, guarded by two independent tests:
//
//   (A) LOCK COVERAGE. A reader honouring the type's own RLock protocol races
//       with any concurrent unguarded writer. TestHXC213_StatusReadsUnderRLock_
//       DoNotRaceWithConcurrentWriters reproduces this on the pre-fix artifact
//       under -race. Its oracle is the race detector, so it is polarity-neutral
//       and meaningful in BOTH directions; the RED baseline is captured by
//       running it against the pre-fix source (see the evidence README).
//       It reads exactly the fields the HXC-014 partial fix left unguarded —
//       NOT Notifications, which that fix already covered and which the chaos
//       test already exercises.
//
//   (B) ALIASING. StartProductionDeployment returns pd.status — the LIVE
//       internal pointer — at :267 :276 :281 :291, and it is the type's ONLY
//       exported method, so it is the only way an external caller can observe
//       status at all. Every caller therefore receives a handle it can mutate,
//       and which a subsequent deployment on the same deployer keeps writing
//       (the single-flight atomic.Bool is reset by defer on return, so a second
//       deployment is legal). TestHXC213_ReturnedStatus_IsCallerOwnedDeepCopy
//       carries the §11.4.115 polarity switch and is INDEPENDENT of the race
//       detector: it proves ownership by mutation, not by timing.
//
// RED_MODE=1 asserts the defect is PRESENT (pre-fix artifact); RED_MODE=0
// (default) asserts it is ABSENT — the standing regression guard.

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// hxc213Config builds a deployment that reaches the status-writing paths fast:
// credentials present (so preparation passes), every optional gate disabled (no
// network I/O), auto-rollback on (so triggerRollback's writes are exercised too).
func hxc213Config(name string, servers ...string) *DeploymentConfig {
	return &DeploymentConfig{
		ProjectName:            name,
		BinaryPath:             "/tmp/helixcode-test-binary",
		Environment:            "test",
		DeploymentStrategy:     ProductionDeploy,
		TargetServers:          servers,
		Credentials:            map[string]string{"deploy_key": "test_key"},
		SecurityGateEnabled:    false,
		PerformanceGateEnabled: false,
		HealthCheckEnabled:     false,
		MonitoringEnabled:      false,
		AutoRollbackEnabled:    true,
	}
}

// TestHXC213_StatusReadsUnderRLock_DoNotRaceWithConcurrentWriters — half (A).
//
// Readers honour the type's advertised protocol (pd.mutex.RLock) and read the
// status fields the HXC-014 partial fix left unguarded. Concurrently, the
// failure/rollback path writes those same fields — exactly the concurrent
// surface production_deployer_chaos_test.go already drives, which today only
// reads Notifications and therefore never touches the unguarded fields.
//
// Pre-fix: DATA RACE (writer unguarded on the same words the reader reads).
// Post-fix: clean — every writer takes the write lock.
//
// The race detector is the oracle here, so there is no RED_MODE branch: the
// test is a faithful reproduction on the broken artifact AND the standing guard
// on the fixed one.
func TestHXC213_StatusReadsUnderRLock_DoNotRaceWithConcurrentWriters(t *testing.T) {
	pd, err := NewProductionDeployer(hxc213Config("hxc213-lock-coverage", "s1"))
	if err != nil {
		t.Fatalf("NewProductionDeployer: %v", err)
	}

	const readers = 6
	const writers = 3
	const iters = 300

	var wg sync.WaitGroup

	// Writers: the real failure path. failDeployment writes EndTime, Duration,
	// Status, CurrentPhase and appends FailedPhases; with AutoRollbackEnabled it
	// also runs triggerRollback, which writes RollbackTriggered, RollbackReason,
	// CurrentPhase and the two Metrics rollback fields.
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for it := 0; it < iters; it++ {
				pd.failDeployment(PhaseDeployment, fmt.Errorf("hxc213 writer %d iter %d", id, it))
			}
		}(w)
	}

	// Readers: the advertised protocol — RLock, read pd.status, RUnlock.
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := 0; it < iters; it++ {
				pd.mutex.RLock()
				_ = pd.status.CurrentPhase
				_ = pd.status.Status
				_ = pd.status.RollbackTriggered
				_ = pd.status.RollbackReason
				_ = len(pd.status.FailedPhases)
				_ = len(pd.status.ServersDeployed)
				_ = pd.status.EndTime
				_ = pd.status.Duration
				_ = pd.status.Metrics.RollbackServers
				pd.mutex.RUnlock()
			}
		}()
	}

	wg.Wait()

	// Consistency (detector-independent): every failDeployment call appended
	// exactly one FailedPhases entry. A lost update here proves a torn
	// read-modify-write on the slice even if the detector missed the window.
	pd.mutex.RLock()
	gotFailed := len(pd.status.FailedPhases)
	pd.mutex.RUnlock()
	if want := writers * iters; gotFailed != want {
		t.Fatalf("FailedPhases lost updates under contention: want %d entries, got %d "+
			"(unsynchronized append on pd.status.FailedPhases)", want, gotFailed)
	}
}

// TestHXC213_ReturnedStatus_IsCallerOwnedDeepCopy — half (B), the §11.4.115
// polarity test. Independent of the race detector: it proves ownership by
// mutating the returned value and checking whether the mutation reached the
// deployer's internal state.
//
//   - RED_MODE=1 (pre-fix artifact): the returned *DeploymentStatus IS the live
//     internal pointer, so the identity check holds and caller mutations reach
//     manager state. This test asserts that, proving the defect is real.
//   - RED_MODE=0 (fixed artifact, default): the caller receives a deep copy it
//     owns outright — distinct pointer, distinct Metrics pointer, distinct
//     slice backing arrays — and NO caller mutation may reach the deployer.
func TestHXC213_ReturnedStatus_IsCallerOwnedDeepCopy(t *testing.T) {
	pd, err := NewProductionDeployer(hxc213Config("hxc213-aliasing", "s1"))
	if err != nil {
		t.Fatalf("NewProductionDeployer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, startErr := pd.StartProductionDeployment(ctx)
	if startErr != nil {
		t.Fatalf("StartProductionDeployment returned single-flight error unexpectedly: %v", startErr)
	}
	if status == nil {
		t.Fatalf("StartProductionDeployment returned nil status")
	}
	// The deployment must have produced a notification ledger; the aliasing
	// probes below need at least one element to test slice backing-array sharing.
	if len(status.Notifications) == 0 {
		t.Fatalf("expected a non-empty notification ledger to probe slice aliasing; got 0")
	}
	if status.Metrics == nil {
		t.Fatalf("expected non-nil Metrics on the returned status")
	}

	if redModeDeployment() {
		if status != pd.status {
			t.Fatalf("RED_MODE: expected the returned status to BE the live internal pointer " +
				"on the pre-fix artifact, but it was already a copy")
		}
		t.Logf("RED_MODE: reproduced — StartProductionDeployment returned the live internal " +
			"*DeploymentStatus; callers alias deployer state")
		return
	}

	// --- GREEN: the caller owns the returned value outright. ---

	if status == pd.status {
		t.Fatalf("StartProductionDeployment returned the LIVE internal *DeploymentStatus; " +
			"callers must receive a deep copy they own (the returned pointer is the only way " +
			"an external caller can observe status, so aliasing it publishes the deployer's " +
			"internal state)")
	}
	if status.Metrics == pd.status.Metrics {
		t.Fatalf("returned status shares the *DeploymentMetrics pointer with the deployer; " +
			"a shallow copy is not enough — Metrics must be deep-copied")
	}

	// Mutate every reachable part of the caller's copy, then prove none of it
	// reached the deployer.
	pd.mutex.RLock()
	wantPhase := pd.status.CurrentPhase
	wantNotifications := pd.status.Metrics.Notifications
	wantLedgerLen := len(pd.status.Notifications)
	wantFirstMessage := pd.status.Notifications[0].Message
	wantFailedLen := len(pd.status.FailedPhases)
	pd.mutex.RUnlock()

	status.CurrentPhase = "MUTATED-BY-CALLER"
	status.Metrics.Notifications = 999999
	status.Notifications[0].Message = "MUTATED-BY-CALLER"
	status.Notifications = append(status.Notifications, NotificationEvent{Type: "ghost"})
	status.FailedPhases = append(status.FailedPhases, "ghost-phase")

	pd.mutex.RLock()
	gotPhase := pd.status.CurrentPhase
	gotNotifications := pd.status.Metrics.Notifications
	gotLedgerLen := len(pd.status.Notifications)
	gotFirstMessage := pd.status.Notifications[0].Message
	gotFailedLen := len(pd.status.FailedPhases)
	pd.mutex.RUnlock()

	if gotPhase != wantPhase {
		t.Errorf("caller mutation of status.CurrentPhase reached the deployer: want %q, got %q",
			wantPhase, gotPhase)
	}
	if gotNotifications != wantNotifications {
		t.Errorf("caller mutation of status.Metrics.Notifications reached the deployer: want %d, got %d "+
			"(Metrics pointer is aliased)", wantNotifications, gotNotifications)
	}
	if gotFirstMessage != wantFirstMessage {
		t.Errorf("caller mutation of status.Notifications[0] reached the deployer: want %q, got %q "+
			"(slice backing array is shared)", wantFirstMessage, gotFirstMessage)
	}
	if gotLedgerLen != wantLedgerLen {
		t.Errorf("caller append to status.Notifications changed the deployer ledger length: want %d, got %d",
			wantLedgerLen, gotLedgerLen)
	}
	if gotFailedLen != wantFailedLen {
		t.Errorf("caller append to status.FailedPhases changed the deployer state: want %d, got %d",
			wantFailedLen, gotFailedLen)
	}
}

// TestHXC213_GoldenBad_UnsynchronizedWriteIsDetected is the §11.4.107(10)
// analyzer self-validation fixture. It performs the PRE-FIX pattern — an
// unsynchronized write to a shared DeploymentStatus concurrent with a read of
// the same field — which the race detector MUST report. Its purpose is to make
// a green run of the guards above mean "no race" rather than "blind test": if
// this fixture does NOT trip the detector, the harness cannot see races at all
// and every other green result in this file is worthless.
//
// It runs ONLY under RED_MODE=1 so the standing suite stays green; the RED_MODE=1
// invocation is expected to FAIL with a DATA RACE, and that failure is the
// evidence (captured in the evidence README).
func TestHXC213_GoldenBad_UnsynchronizedWriteIsDetected(t *testing.T) {
	if !redModeDeployment() {
		t.Skip("SKIP-OK: golden-bad detector self-validation fixture; runs under RED_MODE=1 only, " +
			"where it MUST be reported as a DATA RACE (§11.4.107(10))")
	}

	shared := &DeploymentStatus{Metrics: &DeploymentMetrics{}}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			// The pre-fix write shape: no lock held (cf. production_deployer.go:1099).
			shared.CurrentPhase = string(PhaseFailed)
			shared.Metrics.Notifications++
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			_ = shared.CurrentPhase
			_ = shared.Metrics.Notifications
		}
	}()
	wg.Wait()

	t.Logf("RED_MODE golden-bad fixture completed; the race detector MUST have reported a DATA RACE " +
		"— if this run is green, the harness is blind and every other guard here is void")
}
