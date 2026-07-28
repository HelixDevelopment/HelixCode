package llm

// bedrock_provider_send_leak_test.go — §11.4.115 RED-baseline + polarity-switch
// regression guard for the unguarded blocking send (goroutine leak) in
// bedrock_provider.go's processEventStream, fixed alongside
// anthropic/azure/gemini/vertexai/groq in commit 905a0b0a but left without its
// own guard (bedrock consumes an AWS-SDK-shaped event stream rather than a
// plain HTTP response body, which is why the httptest-server harness in
// provider_stream_send_leak_test.go does not apply here — see the type-based
// reproduction strategy below).
//
// DEFECT (pre-fix): processEventStream took NO ctx parameter for its
// per-chunk send, so `ch <- LLMResponse{...}` had NO ctx.Done() escape hatch.
// A caller that cancels ctx and stops draining ch (exactly what happens on
// real client cancellation) leaves the goroutine blocked forever on the
// unguarded send — a goroutine leak.
//
// FIX: guard the send with
// `select { case ch <- resp: case <-ctx.Done(): return ctx.Err() }` — the same
// pattern applied to every other provider in this package.
//
// REPRODUCTION STRATEGY: bedrock_provider.go's processEventStream takes its
// event source as `reader bedrockruntime.ResponseStreamReader` — an EXPORTED
// interface from the AWS SDK (`Events() <-chan types.ResponseStream`, `Close()
// error`, `Err() error`). The concrete stream type the real SDK returns
// (`*bedrockruntime.InvokeModelWithResponseStreamEventStream`) is
// unconstructible from outside the SDK package (its backing fields are
// unexported and populated only by the SDK's deserialize middleware over a
// live HTTP response), so driving processEventStream through the FULL
// GenerateStream → InvokeModelWithResponseStream → AWS wire chain is
// genuinely disproportionate for this defect class. processEventStream's
// parameter type, however, is the SDK's OWN exported interface — so a
// same-package fake implementing exactly that interface drives the REAL,
// unexported, production `(*BedrockProvider).processEventStream` method
// directly (not a replica of its logic — the actual method under test),
// substituting only the network-facing event source, exactly as the other
// guards substitute a local httptest.Server for the real HTTP endpoint.
//
// §11.4.115 polarity switch via RED_MODE (reusing the package-level redMode()
// helper declared in sp1_redmode_test.go):
//
//	RED_MODE=1        : assert a goroutine IS found parked in
//	                     processEventStream's closure, blocked in "chan send"
//	                     state, after ctx cancellation — proving the defect is
//	                     genuinely reproducible on this artifact.
//	RED_MODE=0 (default): the standing GREEN guard — assert NO such goroutine
//	                     remains, because the fix wraps the send in a
//	                     ctx-guarded select.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeBedrockEventReader implements bedrockruntime.ResponseStreamReader
// (Events() <-chan types.ResponseStream, Close() error, Err() error) — the
// EXACT interface bedrock_provider.go's processEventStream consumes — so it
// can be handed directly to the real production method without touching any
// AWS SDK internals.
type fakeBedrockEventReader struct {
	events chan types.ResponseStream
	stop   chan struct{}
	once   sync.Once
}

func newFakeBedrockEventReader() *fakeBedrockEventReader {
	return &fakeBedrockEventReader{
		events: make(chan types.ResponseStream),
		stop:   make(chan struct{}),
	}
}

func (r *fakeBedrockEventReader) Events() <-chan types.ResponseStream { return r.events }
func (r *fakeBedrockEventReader) Err() error                         { return nil }

// Close stops feedForever and is safe to call multiple times, matching the
// real SDK reader's documented "must allow multiple concurrent calls"
// contract for Close.
func (r *fakeBedrockEventReader) Close() error {
	r.once.Do(func() { close(r.stop) })
	return nil
}

// feedForever emits valid chunk events at a fast, steady rate until stop() is
// closed — mirroring streamForeverSSE's "the upstream never stops emitting on
// its own" behaviour in provider_stream_send_leak_test.go, so the only thing
// that can end the flow is the consumer side (ctx cancellation) under test.
// Deliberately does not early-exit on a blocked send: once processEventStream
// stops draining (post ctx-cancel on pre-fix code), this goroutine blocks on
// its own send too — cleaned up via Close() in the test's defer.
func (r *fakeBedrockEventReader) feedForever() {
	defer close(r.events)
	chunk := &types.ResponseStreamMemberChunk{
		Value: types.PayloadPart{Bytes: []byte(`{"delta":"x"}`)},
	}
	for {
		select {
		case <-r.stop:
			return
		default:
		}
		select {
		case r.events <- chunk:
		case <-r.stop:
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// TestBedrockProvider_ProcessEventStream_NoGoroutineLeak_OnCtxCancelWithoutDrain
// is the §11.4.115 RED-baseline + polarity-switch regression guard for the
// goroutine leak in bedrock_provider.go's processEventStream.
func TestBedrockProvider_ProcessEventStream_NoGoroutineLeak_OnCtxCancelWithoutDrain(t *testing.T) {
	bp := &BedrockProvider{}
	reader := newFakeBedrockEventReader()
	t.Cleanup(func() { reader.Close() })
	go reader.feedForever()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan LLMResponse, 2)
	requestID := uuid.New()
	go func() {
		_ = bp.processEventStream(ctx, reader, ch, requestID, modelFamilyClaude)
	}()

	// Prove the pipe is genuinely live before we stop draining — otherwise a
	// setup failure could masquerade as "never parks", a false GREEN.
	liveDeadline := time.After(2 * time.Second)
	received := 0
	for received < 2 {
		select {
		case _, ok := <-ch:
			if !ok {
				t.Fatalf("bedrock: channel closed before 2 chunks were observed (got %d) — the fake "+
					"reader or provider setup is broken, not exercising the send-leak path under test", received)
			}
			received++
		case <-liveDeadline:
			t.Fatal("bedrock: did not observe 2 chunks within 2s — the pipe never became live")
		}
	}

	// STOP draining. processEventStream will decode further events and
	// attempt to forward them; with nobody reading, ch's 2-slot buffer fills
	// and the NEXT send blocks — pre-fix, unconditionally; post-fix, on the
	// select's ch<- case (with ctx.Done() as the escape not yet ready).
	time.Sleep(150 * time.Millisecond)

	cancel()

	const settle = 1500 * time.Millisecond
	symbolPrefix := "dev.helix.code/internal/llm.(*BedrockProvider).processEventStream"

	if redMode() {
		parked := waitUntilParkedInSend(symbolPrefix, settle)
		require.True(t, parked, "bedrock: RED expectation failed: no goroutine found parked in %s's "+
			"closure (blocked in chan-send state) within %s of ctx cancellation on this artifact — "+
			"the leak should be reproducible here. If this fails, the defect is already fixed; flip "+
			"RED_MODE=0.", symbolPrefix, settle)
		return
	}

	parked := notParkedInSendAfterSettling(symbolPrefix, settle)
	require.False(t, parked, "bedrock: GREEN failed: a goroutine is still parked in %s's closure "+
		"(blocked in chan-send state) %s after ctx cancellation with the channel never drained — an "+
		"unguarded blocking send leaks a goroutine forever. Every send site must use "+
		"`select { case ch <- resp: case <-ctx.Done(): return ctx.Err() }`.", symbolPrefix, settle)
}
