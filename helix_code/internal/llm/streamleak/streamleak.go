// Package streamleak hosts the SHARED §11.4.115 RED-baseline + polarity-switch
// harness for the provider "unguarded blocking send" defect class — a provider
// goroutine that forwards stream chunks with a bare `ch <- resp` (no
// `select { case ch <- resp: case <-ctx.Done(): return ctx.Err() }` escape
// hatch) parks FOREVER once the caller cancels ctx and stops draining, leaking
// both the goroutine and the open upstream HTTP response body it holds via a
// `defer resp.Body.Close()` that never runs.
//
// WHY THIS IS A SEPARATE PACKAGE (and not, as it began, a set of unexported
// helpers inside internal/llm's own _test.go files):
//
// The defect class spans providers living in BOTH dev.helix.code/internal/llm
// AND its dev.helix.code/internal/llm/providers/* sub-packages. Every
// sub-package imports internal/llm for the shared Provider/LLMResponse types,
// so an in-package (`package llm`) test file CANNOT import a sub-package back
// to drive it — Go rejects that outright:
//
//	package dev.helix.code/internal/llm
//	    imports dev.helix.code/internal/llm/providers/helixagent from x_test.go
//	    imports dev.helix.code/internal/llm from embeddings.go: import cycle not allowed in test
//
// (verified empirically, 2026-07-29 — not assumed). That language-level wall is
// exactly why the 905a0b0a/97d5ad2b round of send-leak fixes stopped at the
// internal/llm package boundary and left providers/helixagent unguarded: there
// was no seam through which the sub-package could reach the guard.
//
// Hoisting the harness here — a package that imports NOTHING from
// dev.helix.code and is generic over the response element type — gives every
// consumer the seam: internal/llm's own tests and every providers/* sub-package
// test drive the IDENTICAL harness. A provider is therefore either registered
// in the shared guard or visibly absent from it; it can never be "covered" by a
// divergent bespoke copy that quietly asserts something weaker.
//
// CONST-050(A) / test-support scope: this package is imported ONLY by _test.go
// files. It contains no production behaviour, is referenced by no production
// code path, and deliberately depends on nothing but the standard library plus
// testify.
package streamleak

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// RedMode reports whether the RED_MODE polarity switch (§11.4.115) is active.
//
//	RED_MODE=1         : reproduce-and-assert-the-defect-is-present on the
//	                     current (pre-fix) artifact — the proof the guard is
//	                     genuinely able to fail.
//	RED_MODE=0/unset   : the standing GREEN regression guard asserting the
//	                     defect is ABSENT.
//
// Mirrors internal/llm's package-level redMode() so both sides of the seam read
// the same environment contract.
func RedMode() bool {
	return os.Getenv("RED_MODE") == "1"
}

// ParkedInSend dumps every live goroutine's stack via runtime.Stack(buf, all=true)
// and reports whether any goroutine is currently parked (blocked) with a stack
// frame matching symbolPrefix AND in a channel-send wait state.
//
// A deterministic, unforgeable liveness probe: it does NOT rely on counting
// runtime.NumGoroutine() (polluted by unrelated goroutines in the test binary),
// it names the specific symbol and the specific "chan send" wait state.
func ParkedInSend(symbolPrefix string) bool {
	buf := make([]byte, 1<<16)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	dump := string(buf)
	for _, block := range strings.Split(dump, "\n\n") {
		if strings.Contains(block, symbolPrefix) && strings.Contains(block, "chan send") {
			return true
		}
	}
	return false
}

// WaitUntilParkedInSend polls ParkedInSend until it reports true or the deadline
// elapses. Early-exit-on-true is sound: once an unguarded send blocks a
// goroutine forever, the parked state never reverts on its own, so observing
// true once is conclusive.
func WaitUntilParkedInSend(symbolPrefix string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if ParkedInSend(symbolPrefix) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// ParkedInSendAfterSettling sleeps the FULL settle window (deliberately no early
// exit — a momentary "not parked" reading taken before the goroutine even
// attempts its send would falsely pass on genuinely broken code) and then takes
// a single decisive snapshot.
func ParkedInSendAfterSettling(symbolPrefix string, settle time.Duration) bool {
	time.Sleep(settle)
	return ParkedInSend(symbolPrefix)
}

// StreamForeverSSE writes valid SSE `data: <json>\n\n` frames (or, when rawSSE
// is false, raw back-to-back JSON values with no SSE framing — the shape
// json.Decoder-based parse loops require) at a fast, steady rate until the
// request's own context is done.
//
// The upstream "LLM" therefore NEVER stops emitting on its own: the only thing
// that can stop the flow is the client-side ctx cancellation under test.
func StreamForeverSSE(encode func() []byte, rawSSE bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			payload := encode()
			if rawSSE {
				if _, err := w.Write([]byte("data: ")); err != nil {
					return
				}
				if _, err := w.Write(payload); err != nil {
					return
				}
				if _, err := w.Write([]byte("\n\n")); err != nil {
					return
				}
			} else {
				if _, err := w.Write(payload); err != nil {
					return
				}
			}
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
}

// DriveSendLeak is the shared harness. It drives the caller-supplied stream
// function — which MUST invoke the REAL provider's REAL exported streaming
// method, never a hand-rolled replica of its parse loop — against a small
// (2-slot) result channel, drains a handful of elements to prove the pipe is
// genuinely live, STOPS draining so the buffer fills and the provider goroutine
// blocks on its next send, cancels ctx, and then asserts (per RED_MODE) whether
// a goroutine remains parked in "chan send" state matching symbolPrefix.
//
// It is generic over the channel element type solely so that this package need
// not import dev.helix.code/internal/llm (see the package doc comment: that
// import is what makes the sub-package guards impossible).
func DriveSendLeak[T any](t *testing.T, name, symbolPrefix string, stream func(ctx context.Context, ch chan<- T) error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan T, 2)
	go func() {
		_ = stream(ctx, ch)
	}()

	// Prove the pipe is genuinely live before we stop draining — otherwise a
	// request-setup failure could masquerade as "never parks", a false GREEN.
	liveDeadline := time.After(2 * time.Second)
	received := 0
	for received < 2 {
		select {
		case _, ok := <-ch:
			if !ok {
				t.Fatalf("%s: channel closed before %d chunks were observed (got %d) — the fake "+
					"server or provider setup is broken, not exercising the send-leak path under test",
					name, 2, received)
			}
			received++
		case <-liveDeadline:
			t.Fatalf("%s: did not observe %d chunks within 2s — the pipe never became live", name, 2)
		}
	}

	// STOP draining. The provider goroutine will decode further events and
	// attempt to forward them; with nobody reading, ch's 2-slot buffer fills
	// and the NEXT send blocks — pre-fix, unconditionally; post-fix, on the
	// select's ch<- case (with ctx.Done() as the escape not yet ready).
	time.Sleep(150 * time.Millisecond)

	cancel()

	const settle = 1500 * time.Millisecond
	if RedMode() {
		parked := WaitUntilParkedInSend(symbolPrefix, settle)
		require.True(t, parked, "%s: RED expectation failed: no goroutine found parked in %s's "+
			"closure (blocked in chan-send state) within %s of ctx cancellation on this artifact — "+
			"the leak should be reproducible here. If this fails, the defect is already fixed; flip "+
			"RED_MODE=0.", name, symbolPrefix, settle)
		return
	}

	parked := ParkedInSendAfterSettling(symbolPrefix, settle)
	require.False(t, parked, "%s: GREEN failed: a goroutine is still parked in %s's closure "+
		"(blocked in chan-send state) %s after ctx cancellation with the channel never drained — an "+
		"unguarded blocking send leaks a goroutine (and the open HTTP response body it holds) "+
		"forever. Every send site must use "+
		"`select { case ch <- resp: case <-ctx.Done(): return ctx.Err() }`.", name, symbolPrefix, settle)
}
