// §11.4.115 RED-baseline-on-the-broken-artifact regression guard for the two
// discarded-NewRequestWithContext-error nil dereferences in the Replicate
// client.
//
// DEFECT (pre-fix source, internal/llm/providers/replicate/client.go):
//
//	client.go:87   httpReq, _ := http.NewRequestWithContext(ctx, "POST", predURL, ...)   // error DISCARDED
//	client.go:88   httpReq.Header.Set("Content-Type", "application/json")                 // nil deref
//
//	client.go:129  req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/predictions/"+id, nil)
//	client.go:130  req.Header.Set("Authorization", "Bearer "+c.apiKey)                    // nil deref
//
// Same defect class already fixed at internal/llm/llamacpp_provider.go
// GenerateStream: http.NewRequestWithContext returns (nil, err) when the
// request cannot be built; the discarded error means the very next statement
// dereferences a nil *http.Request. Captured mechanism (FACT, probed on
// go1.26.4 before this test was written):
//
//	"…/models/bad\nmodel/predictions" -> req==nil err="net/url: invalid control character in URL"
//	"…/predictions/bad\nid"           -> req==nil err="net/url: invalid control character in URL"
//	(*http.Request)(nil).Header.Set   -> panic: runtime error: invalid memory
//	                                     address or nil pointer dereference
//
// A panic here is not a failed request — production callers dispatch provider
// Generate calls from spawned goroutines, where an unrecovered panic takes
// down the whole process rather than failing one request.
//
// REACHABILITY (§11.4.6 — both sites are genuinely reachable, from opposite
// directions):
//
//   - client.go:87  — the URL embeds req.Model, which is CALLER/USER supplied.
//     Any model string carrying a URL-invalid byte (control character) makes
//     the composed prediction URL unparseable. The base URL is also
//     production-settable via the EXPORTED NewClientWithBaseURL.
//   - client.go:129 — the URL embeds pred.ID, decoded from the UPSTREAM
//     Replicate JSON response. A remote server (or a proxy, or a corrupted
//     response) returning an id with a URL-invalid byte kills the process.
//
// Note on ctx: unlike the Together client, a nil Context is NOT used as the
// trigger for waitForCompletion, because that function selects on ctx.Done()
// BEFORE building the request — a nil ctx would panic at the select, at a
// different line, and the test would be mis-attributed rather than pinning
// line 129.
//
// POLARITY SWITCH (§11.4.115): one source, two roles.
//
//	RED_MODE=1 (default) — reproduce-and-assert-defect-PRESENT.
//	                       PASSes on the broken artifact; FAILs once fixed.
//	RED_MODE=0           — the standing GREEN regression guard.
//	                       FAILs on the broken artifact; PASSes once fixed.

package replicate

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"dev.helix.code/internal/llm"
)

// redModeEnabled reports whether the polarity switch is in reproduce-the-defect
// mode. Reproduce mode is OPT-IN (RED_MODE=1); the DEFAULT is the standing
// GREEN guard, so the ordinary `go test ./...` invocation stays green on a
// fixed tree (§11.4.135 — the standing guard suite is a release-gate blocker
// and must not be red). Matches the repo precedent
// scripts/gates/fixed_h2_pipe_row_parity_gate.sh:62 (`RED_MODE="${RED_MODE:-0}"`).
// §11.4.115's "default 1" governs while the defect is live; once the fix has
// landed, its own text makes the RED_MODE=0 role the standing guard.
func redModeEnabled() bool {
	return os.Getenv("RED_MODE") == "1"
}

// TestClient_Generate_UnbuildableRequest_ReturnsErrorNotPanic pins the
// production contract for client.go:87 — Generate MUST surface an un-buildable
// HTTP request as a returned error and MUST NOT panic.
//
// No network call is required or made: http.NewRequestWithContext rejects each
// input at parse time, before any dial.
func TestClient_Generate_UnbuildableRequest_ReturnsErrorNotPanic(t *testing.T) {
	tests := []struct {
		name string
		// ctx is deliberately nillable — Generate touches ctx for the first
		// time at the NewRequestWithContext call being guarded.
		ctx context.Context
		// baseURL, when non-empty, is applied via the EXPORTED
		// NewClientWithBaseURL constructor.
		baseURL string
		// model is written straight into the composed prediction URL.
		model string
		// wantErrSubstr pins WHICH error surfaces in GREEN mode, so the guard
		// cannot be satisfied by an unrelated (e.g. network) failure.
		wantErrSubstr string
	}{
		{
			name:          "caller-supplied model carrying a URL-invalid control character",
			ctx:           context.Background(),
			baseURL:       "",
			model:         "meta/meta-llama-3-70b\ninstruct",
			wantErrSubstr: "invalid control character in URL",
		},
		{
			name:          "malformed base URL via the exported NewClientWithBaseURL",
			ctx:           context.Background(),
			baseURL:       "http://::1:44047/v1",
			model:         "meta/meta-llama-3-70b-instruct",
			wantErrSubstr: "invalid port",
		},
		{
			name:          "nil context reaches NewRequestWithContext and yields a nil request",
			ctx:           nil,
			baseURL:       "",
			model:         "meta/meta-llama-3-70b-instruct",
			wantErrSubstr: "nil Context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c *Client
			if tt.baseURL != "" {
				c = NewClientWithBaseURL("test-replicate-key", tt.baseURL)
			} else {
				c = NewClient("test-replicate-key")
			}

			var (
				panicked bool
				panicVal any
				resp     *llm.LLMResponse
				retErr   error
			)

			// Call Generate in THIS goroutine so recover() can observe the
			// panic. In production the caller dispatches from a spawned
			// goroutine, where the same panic is unrecoverable and kills the
			// process — that is the severity being guarded.
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
						panicVal = r
					}
				}()
				resp, retErr = c.Generate(tt.ctx, &llm.LLMRequest{
					Model:    tt.model,
					Messages: []llm.Message{{Role: "user", Content: "Hello"}},
				})
			}()

			if redModeEnabled() {
				require.True(t, panicked,
					"RED_MODE=1: the defect MUST reproduce on this artifact — "+
						"Generate was expected to panic when the request cannot be built. "+
						"If this fails, the defect is already fixed; re-run with RED_MODE=0.")
				t.Logf("defect reproduced (captured evidence): %v", panicVal)
				return
			}

			// RED_MODE=0 — standing regression guard.
			require.False(t, panicked,
				"Generate MUST NOT panic when the request cannot be built; got panic: %v", panicVal)
			require.Error(t, retErr,
				"Generate MUST return an error when the request cannot be constructed")
			require.Nil(t, resp,
				"Generate MUST NOT return a response alongside a construction error")
			require.Contains(t, retErr.Error(), tt.wantErrSubstr,
				"the returned error MUST carry the underlying request-construction cause")
		})
	}
}

// TestClient_waitForCompletion_UnbuildableRequest_ReturnsErrorNotPanic pins the
// production contract for client.go:129 — the polling loop MUST surface an
// un-buildable HTTP request as a returned error and MUST NOT panic.
//
// The prediction id is decoded from the UPSTREAM response, so this is the
// remote-controlled half of the defect class. The poll interval is driven to
// 1ms through the existing SetPollInterval test hook so the guard does not
// wait out the production 2s tick. No network call is made: the URL is
// rejected at parse time, before any dial.
func TestClient_waitForCompletion_UnbuildableRequest_ReturnsErrorNotPanic(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		predictionID  string
		wantErrSubstr string
	}{
		{
			name:          "upstream-supplied prediction id carrying a URL-invalid control character",
			baseURL:       "",
			predictionID:  "pred\n123",
			wantErrSubstr: "invalid control character in URL",
		},
		{
			name:          "malformed base URL via the exported NewClientWithBaseURL",
			baseURL:       "http://::1:44047/v1",
			predictionID:  "pred123",
			wantErrSubstr: "invalid port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c *Client
			if tt.baseURL != "" {
				c = NewClientWithBaseURL("test-replicate-key", tt.baseURL)
			} else {
				c = NewClient("test-replicate-key")
			}
			c.SetPollInterval(time.Millisecond)

			var (
				panicked bool
				panicVal any
				output   string
				retErr   error
			)

			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
						panicVal = r
					}
				}()
				output, _, _, retErr = c.waitForCompletion(context.Background(), tt.predictionID)
			}()

			if redModeEnabled() {
				require.True(t, panicked,
					"RED_MODE=1: the defect MUST reproduce on this artifact — "+
						"waitForCompletion was expected to panic when the poll request "+
						"cannot be built. If this fails, the defect is already fixed; "+
						"re-run with RED_MODE=0.")
				t.Logf("defect reproduced (captured evidence): %v", panicVal)
				return
			}

			// RED_MODE=0 — standing regression guard.
			require.False(t, panicked,
				"waitForCompletion MUST NOT panic when the poll request cannot be built; got panic: %v", panicVal)
			require.Error(t, retErr,
				"waitForCompletion MUST return an error when the request cannot be constructed")
			require.Empty(t, output,
				"waitForCompletion MUST NOT return output alongside a construction error")
			require.Contains(t, retErr.Error(), tt.wantErrSubstr,
				"the returned error MUST carry the underlying request-construction cause")
		})
	}
}
