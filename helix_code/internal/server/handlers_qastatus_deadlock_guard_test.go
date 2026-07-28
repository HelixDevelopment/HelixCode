package server

// Standing regression guard for the getQASessionStatus recursive-RLock
// self-deadlock (§11.4.135 permanent guard, §11.4.115 RED-on-broken-artifact
// polarity, §11.4.118 discovery-pressure — the existing QA-status tests
// asserted success but never exercised the writer-contention window where the
// deadlock fires).
//
// Defect (reproduced before the fix): getQASessionStatus held
// state.Mu.RLock() across json.Marshal(state) / c.JSON(state). But
// *helixqa.SessionState implements MarshalJSON (internal/helixqa/wrapper.go),
// which ITSELF acquires state.Mu.RLock(). Holding the lock in the handler too
// is a RECURSIVE read-lock. Go's sync.RWMutex is documented as NOT safe for
// recursive read-locking: "if a goroutine holds a RWMutex for reading and
// another goroutine might call Lock, no goroutine should expect to be able to
// acquire a read lock until the initial read lock is released." The QA
// orchestrator goroutine spawned by StartSession calls state.Mu.Lock() on
// every phase transition (wrapper.go:138/160/173). If that write-Lock lands
// between the handler's OUTER RLock and MarshalJSON's INNER RLock, the inner
// RLock blocks behind the pending writer while the writer blocks behind the
// outer RLock held by the same handler goroutine — a permanent self-deadlock
// that hangs the request forever (DoS).
//
// Fix: the handler hands the bare *SessionState pointer to json.Marshal /
// c.JSON and does NOT pre-lock — MarshalJSON is the single correct lock point
// and acquires state.Mu exactly once.
//
// Polarity switch (§11.4.115): set HELIX_QASTATUS_RED_MODE=1 to run the RED
// reproduction. BOTH polarities drive the REAL shipped getQASessionStatus
// handler on a REAL live session while a writer goroutine hammers
// state.Mu.Lock(), and BOTH observe it from the SAME child probe process; only
// the assertion flips. RED asserts the probe DIED reporting a wedge (true on
// the pre-fix artifact, false on the fixed one); DEFAULT (no env) runs the
// GREEN guard and asserts the probe survived every attempt.
//
// An earlier revision of this file reproduced the defect by hand-rolling the
// lock sequence in a test-LOCAL helper. That helper only re-proved a documented
// property of sync.RWMutex: it never touched the shipped handler, so it
// behaved identically on every build ever made — pre-fix, fixed, or
// arbitrarily broken — and the RED branch could not fail (§11.4.1). Converted,
// not deleted: deleting a bluff gate removes coverage, converting it creates
// real coverage.
//
// WHY A CHILD PROCESS IS REQUIRED. A first conversion drove the real handler
// IN-PROCESS and returned on wedge. It could never report PASS on a pre-fix
// artifact: the wedged handler goroutine holds state.Mu.RLock() forever, and
// setupQATestServer registers t.Cleanup(qaEngine.Shutdown) — Shutdown takes
// s.Mu.Lock() (internal/helixqa/wrapper.go:210), so cleanup blocks behind the
// wedged reader and the binary dies on the go-test timeout instead. Measured on
// a pre-fix artifact (0dfd0fbc reverted): the defect DID reproduce
// ("wedged on attempt 1") and the run STILL exited 1 with
// "panic: test timed out after 1m30s", the stack showing
// helixqa.(*Engine).Shutdown <- testing.(*common).runCleanup. A guard whose
// top-left quadrant is unreachable is not falsifiable, so the observation point
// moved out of process: the probe calls os.Exit on wedge, which bypasses
// t.Cleanup entirely and turns "the handler hung" into a bounded, readable exit
// code. Observing a deadlock that poisons its own teardown means observing a
// process — the same reason the sibling double-close guard in
// llm_generate_regression_test.go uses a child probe.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"dev.helix.code/internal/helixqa"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// startLiveQASession boots a real QA engine, starts a real session, and returns
// the server plus the live *SessionState the orchestrator goroutine mutates
// under state.Mu.Lock(). BOTH polarities share this fixture so they exercise
// the identical shipped code path.
func startLiveQASession(t *testing.T) (*Server, *helixqa.SessionState) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	server, w, c, bankFile := setupQATestServer(t)

	startReq := StartSessionRequest{Platforms: []string{"web"}, Banks: []string{bankFile}}
	body, _ := json.Marshal(startReq)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/qa/session", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	server.startQASession(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: startQASession returned %d (body=%s)", w.Code, w.Body.String())
	}
	var created helixqa.SessionState
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("setup: bad start response: %v", err)
	}

	state, ok := server.qaEngine.GetSession(created.ID)
	if !ok {
		t.Fatalf("setup: session %s not found in engine", created.ID)
	}
	return server, state
}

// hammerWriteLock reproduces the orchestrator's write pattern — the pattern the
// pre-fix handler raced against — by taking state.Mu.Lock() in a tight loop
// until stop is closed.
func hammerWriteLock(state *helixqa.SessionState, stop <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-stop:
			return
		default:
			state.Mu.Lock()
			state.Phase = "orchestration"
			state.PhaseProgress += 0.0
			state.Mu.Unlock()
		}
	}
}

// callQAStatus drives the REAL shipped getQASessionStatus for one Accept mode
// and reports whether it returned within deadline (plus the recorder, valid
// only when it did return).
func callQAStatus(server *Server, id, accept string, deadline time.Duration) (*httptest.ResponseRecorder, bool) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/qa/session/"+id+"/status", nil)
	if accept != "" {
		c.Request.Header.Set("Accept", accept)
	}
	c.Params = gin.Params{{Key: "id", Value: id}}

	done := make(chan struct{})
	go func() {
		server.getQASessionStatus(c)
		close(done)
	}()

	select {
	case <-done:
		return w, true
	case <-time.After(deadline):
		return w, false
	}
}

// qaStatusProbeEnv turns this test binary into the one-shot child probe that
// TestGuard_GetQASessionStatus_RecursiveRLockDeadlock re-executes.
const qaStatusProbeEnv = "HELIX_QASTATUS_DEADLOCK_PROBE"

// qaStatusProbeWedgeExit is the probe's distinct "I observed the handler wedge"
// exit code. It is deliberately NOT 1 and NOT 2, so a wedge is distinguishable
// from an ordinary test failure (1) or a go-test timeout panic (2) — otherwise
// an unrelated breakage would masquerade as a reproduced deadlock (§11.4.6).
const qaStatusProbeWedgeExit = 97

// Markers the parent matches on, so a wedge/survival is asserted on the probe's
// own words rather than on an exit code alone.
const (
	qaStatusWedgeMarker    = "QASTATUS_PROBE_WEDGED"
	qaStatusSurvivedMarker = "QASTATUS_PROBE_SURVIVED"
)

// qaStatusWedgeDeadline is how long one getQASessionStatus call may take before
// the probe calls it wedged. The handler marshals an in-memory struct, and a
// transient wait behind one writer resolves in microseconds, so this is ~6
// orders of magnitude of headroom — chosen so a loaded shared host cannot
// manufacture a false wedge (a false wedge would make the standing GREEN guard
// flaky, §11.4.50).
const qaStatusWedgeDeadline = 8 * time.Second

// qaStatusProbeAttempts is how many times the probe re-drives the handler before
// concluding no wedge occurs. The deadlock needs a writer to land between the
// handler's outer RLock and MarshalJSON's inner RLock, so it is racy per call;
// on a pre-fix artifact it reproduced on attempt 1, and this many attempts
// against a tight writer loop makes a miss vanishingly unlikely.
const qaStatusProbeAttempts = 40

// qaStatusProbeTimeout bounds the child probe so a hang cannot wedge the suite.
const qaStatusProbeTimeout = 120 * time.Second

// TestGuard_QAStatusDeadlockProbe is the CHILD half of both polarities below. It
// is not a standalone guard: it is skipped unless re-executed with
// qaStatusProbeEnv set.
//
// It drives the REAL shipped getQASessionStatus against a REAL live session
// while a writer goroutine hammers state.Mu.Lock(). On a pre-fix artifact the
// handler's outer RLock and MarshalJSON's inner RLock straddle that pending
// writer and the request wedges PERMANENTLY; the probe reports it and calls
// os.Exit, which is load-bearing — returning normally would run
// t.Cleanup(qaEngine.Shutdown), which blocks forever behind the wedged reader
// (see the WHY A CHILD PROCESS IS REQUIRED note above). On the fixed artifact
// every attempt returns and the probe exits 0.
func TestGuard_QAStatusDeadlockProbe(t *testing.T) {
	if os.Getenv(qaStatusProbeEnv) != "1" {
		t.Skip("SKIP-OK: child-process probe, driven only by " +
			"TestGuard_GetQASessionStatus_RecursiveRLockDeadlock; it is not a standalone guard")
	}

	server, state := startLiveQASession(t)

	stop := make(chan struct{})
	var hammer sync.WaitGroup
	hammer.Add(1)
	go hammerWriteLock(state, stop, &hammer)

	for i := 0; i < qaStatusProbeAttempts; i++ {
		for _, accept := range []string{"", "text/event-stream"} {
			w, returned := callQAStatus(server, state.ID, accept, qaStatusWedgeDeadline)
			if !returned {
				// Written straight to stdout, not t.Logf: os.Exit skips the
				// testing framework's buffered-output flush, and this line is
				// the parent's evidence.
				fmt.Printf("%s accept=%q attempt=%d\n", qaStatusWedgeMarker, accept, i+1)
				os.Exit(qaStatusProbeWedgeExit)
			}
			// Status is asserted for the JSON mode only. The SSE branch never
			// sets a status explicitly — it writes the event body straight to
			// c.Writer — so the recorder simply reports its default 200.
			// Asserting that would assert httptest's default, not the handler
			// (§11.4.1); the SSE path's real observable is that it RETURNED at
			// all, which the wedge check above already covers.
			if w.Code != http.StatusOK && accept == "" {
				close(stop)
				hammer.Wait()
				t.Fatalf("getQASessionStatus(accept=%q) returned %d, want 200 (body=%s)",
					accept, w.Code, w.Body.String())
			}
		}
	}

	close(stop)
	hammer.Wait()
	fmt.Printf("%s attempts=%d\n", qaStatusSurvivedMarker, qaStatusProbeAttempts)
}

// TestGuard_GetQASessionStatus_RecursiveRLockDeadlock — the standing guard.
//
// BOTH polarities run the SAME child probe over the SAME fixture; only the
// assertion flips (§11.4.115 one-source-two-roles). Observing from a process in
// BOTH directions is deliberate: it is what lets a REGRESSION fail this guard
// cleanly and quickly. An in-process GREEN branch would instead hang in
// t.Cleanup and only surface as a whole-suite timeout.
//
// RED_MODE (HELIX_QASTATUS_RED_MODE=1): assert the probe DIED reporting a wedge.
// True on a pre-fix artifact; false on the fixed one, where it exits 0.
//
// DEFAULT (no env): assert the probe SURVIVED every attempt — the fixed handler
// never wedges under writer contention.
func TestGuard_GetQASessionStatus_RecursiveRLockDeadlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), qaStatusProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestGuard_QAStatusDeadlockProbe$", "-test.v=true",
		// Bound the child independently of the parent so a child-side hang
		// surfaces as its own panic rather than as a killed context.
		"-test.timeout="+(qaStatusProbeTimeout-20*time.Second).String())
	cmd.Env = append(os.Environ(), qaStatusProbeEnv+"=1")
	out, runErr := cmd.CombinedOutput()

	require.NoError(t, ctx.Err(),
		"the child probe must terminate on its own, not hit the %s timeout.\nchild output:\n%s",
		qaStatusProbeTimeout, out)

	exitCode := cmd.ProcessState.ExitCode()

	if os.Getenv("HELIX_QASTATUS_RED_MODE") == "1" {
		require.Error(t, runErr,
			"RED expectation: driving the REAL getQASessionStatus on a PRE-FIX artifact under "+
				"writer contention MUST wedge the probe. A clean exit means the handler no longer "+
				"self-deadlocks, so the defect is absent and this RED baseline no longer "+
				"characterises anything.\nchild output:\n%s", out)
		require.Equal(t, qaStatusProbeWedgeExit, exitCode,
			"RED: the probe must die from the recursive-RLock wedge (exit %d), not from an "+
				"unrelated failure or timeout.\nchild output:\n%s", qaStatusProbeWedgeExit, out)
		require.Contains(t, string(out), qaStatusWedgeMarker,
			"RED: the probe must report the wedge it observed.\nchild output:\n%s", out)
		t.Logf("RED reproduced: the real getQASessionStatus wedged the probe process (exit %d)", exitCode)
		return
	}

	// GREEN guard (DEFAULT).
	require.NoError(t, runErr,
		"DEADLOCK: the probe driving the REAL getQASessionStatus did not survive under concurrent "+
			"writer contention. Exit %d means the recursive-RLock self-deadlock has regressed.\n"+
			"child output:\n%s", exitCode, out)
	require.Contains(t, string(out), qaStatusSurvivedMarker,
		"GREEN: the probe must report that every attempt returned; a missing marker means it "+
			"exited 0 without completing the loop.\nchild output:\n%s", out)
	require.False(t, strings.Contains(string(out), qaStatusWedgeMarker),
		"GREEN: the probe reported a wedge.\nchild output:\n%s", out)
}
