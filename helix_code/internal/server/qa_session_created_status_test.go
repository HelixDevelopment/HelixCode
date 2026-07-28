package server

// qa_session_created_status_test.go — standing regression guard (§11.4.135)
// for HXC-154 at the HTTP layer, and the §11.4.146 STEP-3 "extend to the
// end-user-visible symptom" companion to the deterministic engine-level guard
// in internal/helixqa/wrapper_created_snapshot_test.go.
//
// THE SYMPTOM. POST /api/v1/qa/session answered 201 Created with a body whose
// "status" was whichever value the orchestrator goroutine happened to have
// written by the time gin marshalled the response — usually "pending",
// sometimes "running", and (measured, rarely) even "completed". A 201 Created
// carrying "status":"completed" is an incoherent pair: the response announces
// a resource it simultaneously reports as already finished. Two byte-identical
// requests produced different bodies for reasons no caller can influence,
// observe, or retry around.
//
// ROOT CAUSE (§11.4.102). Engine.StartSession returned the LIVE *SessionState
// that the goroutine it had just spawned was advancing pending -> running;
// startQASession passed that live pointer straight to c.JSON. Fixed at source
// (§11.4.1) by having StartSession return a detached creation snapshot; the
// handler is unchanged because it was never the wrong layer — it faithfully
// renders whatever the engine hands it.
//
// NOT A DATA RACE (§11.4.6). (*SessionState).MarshalJSON RLocks state.Mu and
// the orchestrator write-Locks it, so `-race` reports nothing; 128 parallel
// pre-fix `-race` runs produced 3 failures, all plain assertion failures and
// zero DATA RACE reports.
//
// WHY THIS GUARD IS STATISTICAL AND THE ENGINE-LEVEL ONE IS NOT. Once gin has
// written the response bytes, the body is frozen — a test cannot afterwards
// force the losing interleaving to have happened. There is therefore no
// deterministic HTTP-level oracle for this defect, so this guard drives the
// endpoint enough times to make the defective interleaving essentially certain
// to appear. The DETERMINISTIC proof lives one layer down, where Shutdown()
// gives an exact happens-after edge; the two guards are complementary and both
// are required.
//
// TRIAL COUNT IS MEASURED, NOT GUESSED (§11.4.6, calibrated on this repo's own
// fixtures per §11.4.107(13)). Pre-fix per-request probability of a non-pending
// body, measured over 9000 requests on this package:
//
//	with -race   : 238/9000 = 2.64%
//	without -race:  78/9000 = 0.87%   <- the conservative figure
//
// At the conservative 0.87%, the chance that 2000 consecutive pre-fix requests
// ALL report "pending" — i.e. the chance this guard fails to notice a
// regression — is (1-0.0087)^2000 ~= 3e-8. Measured cost of the 2000 trials:
// ~1.3 s under -race, ~0.07 s without. Sessions are created against one server
// and drained by the shared t.Cleanup Shutdown.
//
// §11.4.115 polarity switch — ONE source, two roles:
//
//	RED_MODE=1  assert the DEFECT is present: at least one 201 body reports a
//	            status other than "pending". PASSES pre-fix, FAILS post-fix.
//	RED_MODE=0  (DEFAULT) the standing GREEN guard: every 201 body reports
//	            exactly "pending".
//
// SCOPE (§11.4.146 STEP 3, deliberate). Only the CREATION response is pinned.
// GET /qa/session/:id/status and GET /qa/sessions render live state on
// purpose — reporting a session's current status is exactly their job, and
// pinning them would be wrong. The defect is specific to a 201 Created body,
// which must describe the resource as created.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"dev.helix.code/internal/helixqa"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// hxc154Trials is calibrated from the measured pre-fix defect rate above.
const hxc154Trials = 2000

func TestStartQASession_CreatedResponseStatusIsDeterministic(t *testing.T) {
	server, _, _, bankFile := setupQATestServer(t)

	body, err := json.Marshal(StartSessionRequest{
		Platforms: []string{"web"},
		Banks:     []string{bankFile},
	})
	require.NoError(t, err)

	observed := map[string]int{}
	for i := 0; i < hxc154Trials; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, err = http.NewRequest("POST", "/api/v1/qa/session", bytes.NewReader(body))
		require.NoError(t, err)
		c.Request.Header.Set("Content-Type", "application/json")

		server.startQASession(c)
		require.Equal(t, http.StatusCreated, w.Code,
			"trial %d: session creation must succeed for the guard to mean anything", i)

		var state helixqa.SessionState
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &state),
			"trial %d: the 201 body must be a decodable SessionState", i)
		require.NotEmpty(t, state.ID, "trial %d: every created session must carry an ID", i)

		observed[state.Status]++
	}

	// Captured evidence (§11.4.5): the full observed distribution, logged in
	// both polarities so a PASS and a FAIL are equally auditable.
	t.Logf("HXC-154: %d POST /api/v1/qa/session responses, status distribution: %s",
		hxc154Trials, renderDistribution(observed))

	if redMode(t) {
		require.Greater(t, hxc154Trials-observed["pending"], 0,
			"RED_MODE: expected at least one 201 body to report a status other than "+
				"'pending' — the pre-fix endpoint renders the live session, whose "+
				"orchestrator goroutine races the response marshal. Observed: %s",
			renderDistribution(observed))
		return
	}

	require.Equal(t, map[string]int{"pending": hxc154Trials}, observed,
		"every 201 Created body must report exactly 'pending': the creation response "+
			"describes the session AS CREATED and must not leak the orchestrator "+
			"goroutine's concurrent progress. Observed: %s", renderDistribution(observed))
}

// TestStartQASession_CreatedResponseStatusIsDeterministic_Concurrent is the
// §11.4.146 STEP-3 extend case: the same invariant under the topology a real
// server actually sees — many clients creating sessions at once, so the engine's
// session map and a crowd of orchestrator goroutines are all contending.
//
// The serial guard above would not notice a regression that reintroduced the
// defect only under contention (for example a "cheaper" variant that took the
// creation snapshot AFTER spawning the goroutine — free of charge when nothing
// else is running, defective the moment the scheduler has other work). This
// case pins that down.
func TestStartQASession_CreatedResponseStatusIsDeterministic_Concurrent(t *testing.T) {
	const (
		clients          = 64
		requestsPerClint = 8
	)

	server, _, _, bankFile := setupQATestServer(t)

	body, err := json.Marshal(StartSessionRequest{
		Platforms: []string{"web"},
		Banks:     []string{bankFile},
	})
	require.NoError(t, err)

	statuses := make(chan string, clients*requestsPerClint)
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerClint; j++ {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				req, reqErr := http.NewRequest("POST", "/api/v1/qa/session", bytes.NewReader(body))
				if reqErr != nil {
					statuses <- "<request-build-failed>"
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				c.Request = req

				server.startQASession(c)
				if w.Code != http.StatusCreated {
					statuses <- fmt.Sprintf("<http-%d>", w.Code)
					continue
				}
				var state helixqa.SessionState
				if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
					statuses <- "<undecodable-body>"
					continue
				}
				statuses <- state.Status
			}
		}()
	}
	wg.Wait()
	close(statuses)

	observed := map[string]int{}
	for s := range statuses {
		observed[s]++
	}
	total := clients * requestsPerClint

	t.Logf("HXC-154 concurrent: %d clients x %d requests = %d responses, status distribution: %s",
		clients, requestsPerClint, total, renderDistribution(observed))

	if redMode(t) {
		require.Greater(t, total-observed["pending"], 0,
			"RED_MODE: expected at least one concurrently-created session to report a "+
				"status other than 'pending'. Observed: %s", renderDistribution(observed))
		return
	}

	require.Equal(t, map[string]int{"pending": total}, observed,
		"every concurrently-created session's 201 body must report exactly 'pending' — "+
			"contention must not leak orchestrator progress into a creation response. "+
			"Observed: %s", renderDistribution(observed))
}

// renderDistribution formats an observed status tally deterministically so the
// evidence line is stable across runs (map iteration order is randomised).
func renderDistribution(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := ""
	for i, k := range keys {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%s=%d", k, counts[k])
	}
	if out == "" {
		return "<none>"
	}
	return out
}
