// §11.4.115 RED-baseline-on-the-broken-artifact regression guard for the
// Together client's discarded-NewRequestWithContext-error nil dereference.
//
// DEFECT (pre-fix source, internal/llm/providers/together/client.go):
//
//	client.go:72  httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL, ...)  // error DISCARDED
//	client.go:73  httpReq.Header.Set("Content-Type", "application/json")                  // nil deref
//
// This is the same defect class already fixed at
// internal/llm/llamacpp_provider.go GenerateStream: when
// http.NewRequestWithContext cannot build the request it returns
// (nil, err); the discarded error means the very next statement
// dereferences a nil *http.Request. Captured mechanism (FACT, probed on
// go1.26.4 before this test was written):
//
//	nil ctx                        -> req==nil err="net/http: nil Context"
//	unbracketed IPv6 literal +port -> req==nil err=`invalid port "::1:44047" after host`
//	(*http.Request)(nil).Header.Set -> panic: runtime error: invalid memory
//	                                   address or nil pointer dereference
//
// A panic here is not a failed request — production callers dispatch
// provider Generate calls from spawned goroutines, where an unrecovered
// panic takes down the whole process rather than failing one request.
//
// REACHABILITY (honest boundary, §11.4.6 — the two cases differ):
//
//   - nil context: REACHABLE TODAY through the exported API. Generate's ctx
//     is caller-supplied; any caller handing it a nil Context turns a
//     caller-side mistake into a process kill instead of a returned error.
//   - malformed base URL: NOT reachable today — baseURL is set only from the
//     TogetherBaseURL constant and the package exposes no setter (unlike the
//     replicate client's exported NewClientWithBaseURL). It is covered here
//     because the field is package-writable (client_test.go already
//     overrides it the same way) and because the guard must hold if a
//     configurable base URL is ever introduced.
//
// POLARITY SWITCH (§11.4.115): one source, two roles.
//
//	RED_MODE=1 (default) — reproduce-and-assert-defect-PRESENT.
//	                       PASSes on the broken artifact; FAILs once fixed.
//	RED_MODE=0           — the standing GREEN regression guard.
//	                       FAILs on the broken artifact; PASSes once fixed.

package together

import (
	"context"
	"os"
	"testing"

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
// production contract: Generate MUST surface an un-buildable HTTP request as a
// returned error and MUST NOT panic.
//
// No network call is required or made: http.NewRequestWithContext rejects both
// inputs before any dial.
func TestClient_Generate_UnbuildableRequest_ReturnsErrorNotPanic(t *testing.T) {
	tests := []struct {
		name string
		// ctx is deliberately nillable — the nil-Context case is the
		// production-reachable trigger through the exported Generate API.
		ctx context.Context
		// baseURL, when non-empty, overrides the production constant.
		baseURL string
		// wantErrSubstr pins WHICH error surfaces in GREEN mode, so the guard
		// cannot be satisfied by an unrelated (e.g. network) failure.
		wantErrSubstr string
	}{
		{
			name:          "nil context reaches NewRequestWithContext and yields a nil request",
			ctx:           nil,
			baseURL:       "",
			wantErrSubstr: "nil Context",
		},
		{
			name:          "malformed base URL (unbracketed IPv6 literal + port)",
			ctx:           context.Background(),
			baseURL:       "http://::1:44047/v1/chat/completions",
			wantErrSubstr: "invalid port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient("test-together-key")
			if tt.baseURL != "" {
				c.baseURL = tt.baseURL
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
					Model:    "meta-llama/Llama-3-70b-chat-hf",
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
