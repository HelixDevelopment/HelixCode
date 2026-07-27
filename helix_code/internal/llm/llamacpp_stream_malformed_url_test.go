// §11.4.115 RED-baseline-on-the-broken-artifact regression guard for the
// LlamaCPPProvider.GenerateStream nil-dereference crash.
//
// DEFECT (captured on the pre-fix artifact):
//
//	panic: runtime error: invalid memory address or nil pointer dereference
//	[signal SIGSEGV: segmentation violation code=0x1 addr=0x38 ...]
//	dev.helix.code/internal/llm.(*LlamaCPPProvider).GenerateStream
//	    internal/llm/llamacpp_provider.go:346
//
// ROOT CAUSE (established as FACT, not inference):
//
//	llamacpp_provider.go:345  req, _ := http.NewRequestWithContext(...)   // error DISCARDED
//	llamacpp_provider.go:346  req.Header.Set("Content-Type", ...)         // nil deref
//
// When the composed URL is malformed, http.NewRequestWithContext returns
// (nil, err). The error is discarded, so the very next statement
// dereferences a nil *http.Request. The fault address 0x38 (56) is exactly
// offsetof(http.Request.Header) — the crash address and the source line
// agree to the byte.
//
// The sibling Generate method (llamacpp_provider.go:178) does this
// correctly with `req, err := ...` + an error check. GenerateStream is the
// copy-drift outlier. A malformed URL must yield a returned error, never a
// process-killing panic — a panic in a caller-spawned goroutine takes down
// the whole process, not just the request.
//
// POLARITY SWITCH (§11.4.115): one source, two roles.
//
//	RED_MODE unset or 0 (DEFAULT) — the standing GREEN regression guard.
//	                       Asserts the defect is ABSENT. This is the
//	                       committed default because the fix has LANDED:
//	                       the default invocation (`go test ./...`,
//	                       `make test`) must be GREEN on a fixed tree
//	                       (§11.4.135 — the standing guard suite is a
//	                       release-gate blocker and must not be red).
//	RED_MODE=1           — historical reproduction mode. Asserts the
//	                       defect is PRESENT, so it PASSes only on a
//	                       PRE-FIX artifact and FAILs on a fixed one.
//
// Default matches the repo's existing §11.4.115 precedent,
// scripts/gates/fixed_h2_pipe_row_parity_gate.sh:62 (`RED_MODE="${RED_MODE:-0}"`).
// §11.4.115's "default 1" applies while the defect is still live; once the
// fix lands, its own text makes the RED_MODE=0 role the standing guard.
//
// Captured polarity proof (both artifacts, real runs):
//
//	pre-fix  RED_MODE=1 -> PASS (panic reproduced)   RED_MODE=0 -> FAIL
//	post-fix RED_MODE=1 -> FAIL (defect gone)        RED_MODE=0 -> PASS

package llm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// redModeEnabled reports whether the polarity switch is in reproduce-the-defect
// mode. Reproduce mode is OPT-IN (RED_MODE=1); the default is the standing
// GREEN guard, so the ordinary `go test ./...` invocation is green on a fixed
// tree per §11.4.135.
func redModeEnabled() bool {
	return os.Getenv("RED_MODE") == "1"
}

// TestLlamaCPP_GenerateStream_MalformedURL_ReturnsErrorNotPanic pins the
// production contract: GenerateStream MUST surface a malformed-URL condition
// as a returned error and MUST NOT panic.
//
// ServerHost is an UNBRACKETED IPv6 literal — the exact shape that composes
// into the malformed "http://::1:44047/completion" seen in the captured
// crash. No network call is required or made: http.NewRequestWithContext
// rejects the URL at parse time, before any dial.
func TestLlamaCPP_GenerateStream_MalformedURL_ReturnsErrorNotPanic(t *testing.T) {
	provider, err := NewLlamaCPPProvider(LlamaConfig{
		Model:         "llama-3-8b",
		ContextSize:   4096,
		ServerHost:    "http://::1", // unbracketed IPv6 => malformed once ":port" is appended
		ServerPort:    44047,
		ServerTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ch := make(chan LLMResponse, 4)

	var (
		panicked bool
		panicVal any
		retErr   error
	)

	// Call GenerateStream in THIS goroutine so recover() can observe the panic.
	// In production the caller spawns a goroutine, where the same panic is
	// unrecoverable and kills the process — that is the severity being guarded.
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicVal = r
			}
		}()
		retErr = provider.GenerateStream(context.Background(), &LLMRequest{
			ID:       uuid.New(),
			Model:    "llama-3-8b",
			Stream:   true,
			Messages: []Message{{Role: "user", Content: "Hello"}},
		}, ch)
	}()

	if redModeEnabled() {
		require.True(t, panicked,
			"RED_MODE=1: the defect MUST reproduce on this artifact — "+
				"GenerateStream was expected to panic on a malformed URL. "+
				"If this fails, the defect is already fixed; re-run with RED_MODE=0.")
		t.Logf("defect reproduced (captured evidence): %v", panicVal)
		return
	}

	// RED_MODE=0 — standing regression guard.
	require.False(t, panicked,
		"GenerateStream MUST NOT panic on a malformed URL; got panic: %v", panicVal)
	require.Error(t, retErr,
		"GenerateStream MUST return an error when the request cannot be constructed")
	// Assert WHICH error surfaced. Without this the guard could be satisfied by
	// an unrelated failure (e.g. a dial error under a laxer URL parser), which
	// would let it pass for the wrong reason.
	require.ErrorContains(t, retErr, "stream request creation failed",
		"the error MUST come from request construction, not from a later dial")
	require.ErrorContains(t, retErr, "invalid port",
		"the wrapped cause MUST be the URL-parse failure (%%w chain preserved)")
}
