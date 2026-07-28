package helixqa

// wrapper_created_snapshot_test.go — standing regression guard (§11.4.135) for
// HXC-154: StartSession handed its caller the LIVE *SessionState that the
// orchestrator goroutine it had just spawned was concurrently advancing
// pending -> running -> completed.
//
// Root cause (§11.4.102, confirmed from source):
//
//	StartSession built `state` with Status "pending", registered it in
//	e.sessions, spawned a goroutine whose FIRST act is
//	`state.Mu.Lock(); state.Status = "running"`, and then returned that same
//	pointer. Every caller therefore held a handle whose value changed
//	underneath it at an instant governed purely by the Go scheduler.
//
// NOTE ON WHAT THIS IS *NOT* (§11.4.6 — the reported mechanism, refuted):
// this is NOT a data race. (*SessionState).MarshalJSON takes state.Mu.RLock()
// and the orchestrator takes state.Mu.Lock(), so every read and write is
// properly synchronised and `go test -race` reports nothing. Measured on the
// pre-fix artifact: 128 parallel `-race` runs of the server-side handler test
// produced 3 failures, ALL of them plain assertion failures
// ("expected: pending / actual: running") and ZERO "WARNING: DATA RACE"
// reports. The defect is an unordered *race condition* (a non-deterministic
// happens-before), not an unsynchronised memory access.
//
// Fix (§11.4.1, at source): StartSession captures a DETACHED creation
// snapshot BEFORE spawning the goroutine and returns that. The live handle
// stays reachable via GetSession(id), which is the documented way to observe
// a session's progress.
//
// §11.4.115 polarity switch — ONE source, two roles:
//
//	RED_MODE=1  drives the REAL production StartSession and asserts the
//	            DEFECT: after Shutdown drains the orchestrator goroutine, the
//	            value returned to the caller has been mutated to a terminal
//	            status, and it is the very same object as the live session.
//	            PASSES on the pre-fix artifact, FAILS on the fixed one.
//	RED_MODE=0  (DEFAULT) the standing GREEN guard: the returned value still
//	            reads "pending" after the goroutine has fully drained, and is
//	            a distinct object from the live session.
//
// This guard is DETERMINISTIC in both polarities: Shutdown() blocks on the
// engine's WaitGroup, and every exit path of the orchestrator goroutine
// commits a terminal status under state.Mu, so once Shutdown returns the live
// session is terminal with probability 1 — no sleeps, no scheduling luck.
//
// Mocks are NOT used: this drives the real Engine and the real SessionState.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeTempBank materialises a real on-disk bank file. StartSession stats every
// bank path it is given, so this must be a genuine file, not a fabricated path.
func writeTempBank(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test-bank.yaml")
	require.NoError(t, os.WriteFile(path, []byte("test: true\n"), 0o644))
	return path
}

func TestStartSession_ReturnsDetachedCreationSnapshot(t *testing.T) {
	engine := newEnabledEngine(t)

	bankFile := writeTempBank(t)
	const sessionID = "hxc154-detached-snapshot"

	returned, err := engine.StartSession(context.Background(), sessionID,
		[]string{"web"}, []string{bankFile}, false)
	require.NoError(t, err)
	require.NotNil(t, returned)

	live, ok := engine.GetSession(sessionID)
	require.True(t, ok, "the session must be registered under its ID")

	// Drain the orchestrator goroutine. Every exit path of that goroutine
	// commits a terminal status under state.Mu before returning, and
	// Shutdown blocks on the engine WaitGroup, so after this call the LIVE
	// session is terminal deterministically.
	engine.Shutdown()

	live.Mu.RLock()
	liveStatus := live.Status
	live.Mu.RUnlock()

	returned.Mu.RLock()
	returnedStatus := returned.Status
	returned.Mu.RUnlock()

	// ANTI-BLUFF (both polarities): prove the goroutine genuinely ran and
	// genuinely advanced the session. Without this the GREEN branch below
	// could pass trivially on an artifact where the orchestrator never
	// started at all — a guard that asserts "pending" against a session
	// nothing ever touched proves nothing.
	require.Contains(t, []string{"completed", "failed", "cancelled"}, liveStatus,
		"the live session must have reached a terminal status once Shutdown drained "+
			"its goroutine — otherwise this guard is not observing a session that ran")

	if redMode() {
		// RED: the caller's handle IS the live session, so the orchestrator
		// mutated the value out from under it. This is the defect.
		require.Same(t, live, returned,
			"RED_MODE: pre-fix StartSession returns the LIVE session pointer")
		require.Equal(t, liveStatus, returnedStatus,
			"RED_MODE: pre-fix the caller's handle tracks the live status")
		require.NotEqual(t, "pending", returnedStatus,
			"RED_MODE: pre-fix the value handed to the caller no longer reads "+
				"'pending' — the orchestrator changed it underneath them")
		return
	}

	// GREEN: the returned value is a detached record of the session AS
	// CREATED and is immune to the orchestrator's subsequent writes.
	require.NotSame(t, live, returned,
		"StartSession must return a DETACHED snapshot, not the live session pointer")
	require.Equal(t, "pending", returnedStatus,
		"the creation snapshot must still read 'pending' after the orchestrator "+
			"goroutine has fully drained — it is a record of the session as created")

	// The snapshot must still be a faithful record, not an empty husk.
	require.Equal(t, sessionID, returned.ID)
	require.Equal(t, []string{"web"}, returned.Platforms)
	require.Equal(t, []string{bankFile}, returned.Banks)
	require.False(t, returned.StartTime.IsZero(), "the snapshot must carry the real StartTime")
	require.Nil(t, returned.EndTime, "a session as created has no EndTime")

	// The snapshot must not carry the live cancellation handle: cancelling a
	// session goes through Engine.CancelSession(id), never through a copy.
	require.Nil(t, returned.CancelFunc,
		"the detached snapshot must not expose the live CancelFunc")

	// Mutating the snapshot must not corrupt the live session.
	returned.Status = "tampered"
	live.Mu.RLock()
	stillLive := live.Status
	live.Mu.RUnlock()
	require.Equal(t, liveStatus, stillLive,
		"writing to the returned snapshot must not reach the live session")
}
