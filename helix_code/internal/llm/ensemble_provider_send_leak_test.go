package llm

// ensemble_provider_send_leak_test.go — §11.4.115 RED-baseline + polarity-switch
// regression guards for the TWO independent defects fixed in
// ensemble_provider.go's GenerateStream by commit 905a0b0a, left without their
// own guards.
//
// DEFECT 1 (close-on-error / guaranteed hang): pre-fix, `close(ch)` ran ONLY on
// the success path (after the `ch <- *resp` send); an e.Generate error
// returned WITHOUT ever closing ch. Every real caller in this codebase drains
// via `for resp := range ch`, so a never-closed channel left that drain loop
// blocked FOREVER on every ensemble Generate failure — a guaranteed hang, not
// a leak.
//
// DEFECT 2 (unguarded send / goroutine leak): pre-fix, the single
// `ch <- *resp` on the success path had NO ctx.Done() escape hatch. A caller
// that cancels ctx and never drains ch (an unbuffered or abandoned channel)
// leaves the GenerateStream goroutine blocked forever on the unguarded send.
//
// FIX: `defer close(ch)` at the top of GenerateStream (fixes DEFECT 1) plus
// `select { case ch <- *resp: case <-ctx.Done(): return ctx.Err() }` on the
// send (fixes DEFECT 2).
//
// REPRODUCTION STRATEGY: both guards drive the REAL, exported
// (*EnsembleProvider).GenerateStream — never a hand-rolled replica.
//
//   - DEFECT 1 is reproduced with ZERO member providers: Generate's very
//     first line (`len(members) == 0`) returns an error deterministically and
//     synchronously, with no goroutines/timing involved — the simplest,
//     least-flaky way to drive the e.Generate-error path.
//   - DEFECT 2 is reproduced with ONE real ensembleStubProvider member (the
//     package's existing unit-test provider double, ensemble_provider_test.go)
//     that succeeds immediately, and an UNBUFFERED, never-drained ch — proving
//     liveness via the stub's call counter (atomic) before asserting on
//     goroutine state, then using the same runtime.Stack(all=true) "chan send"
//     detection technique as provider_stream_send_leak_test.go /
//     bedrock_provider_send_leak_test.go.
//
// §11.4.115 polarity switch via RED_MODE (reusing the package-level redMode()
// helper declared in sp1_redmode_test.go).

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestEnsembleProvider_GenerateStream_ClosesChannelOnGenerateError is the
// §11.4.115 RED-baseline + polarity-switch regression guard for DEFECT 1
// (close-on-error / guaranteed hang) in ensemble_provider.go's GenerateStream.
//
// RED_MODE=1        : assert the channel drain does NOT complete within the
//
//	settle window — proving the hang is genuinely reproducible on this
//	artifact (pre-fix: ch is never closed on the error path).
//
// RED_MODE=0 (default): the standing GREEN guard — assert the channel drain
//
//	DOES complete (ch gets closed) because GenerateStream now
//	`defer close(ch)`s on every return path, including the error path.
func TestEnsembleProvider_GenerateStream_ClosesChannelOnGenerateError(t *testing.T) {
	// Zero members: e.Generate's first line deterministically and
	// synchronously returns "no member providers configured" — no goroutines,
	// no timing dependence, driving the real e.Generate-error path in
	// GenerateStream with no risk of a flaky false result.
	ens := NewEnsembleProvider(EnsembleProviderConfig{})

	req := &LLMRequest{
		ID:       uuid.New(),
		Model:    EnsembleModelName,
		Messages: []Message{{Role: "user", Content: "hi"}},
	}

	ch := make(chan LLMResponse, 4)
	genErrCh := make(chan error, 1)
	go func() {
		genErrCh <- ens.GenerateStream(context.Background(), req, ch)
	}()

	// GenerateStream itself must return promptly (the e.Generate error path is
	// synchronous) regardless of which side of the fix we're testing — only
	// whether ch subsequently gets CLOSED is in question.
	select {
	case err := <-genErrCh:
		require.Error(t, err, "ensemble: GenerateStream with zero members must return the "+
			"\"no member providers configured\" error")
	case <-time.After(2 * time.Second):
		t.Fatal("ensemble: GenerateStream did not return within 2s — the fake-provider setup " +
			"is broken, not exercising the close-on-error path under test")
	}

	drained := make(chan struct{})
	go func() {
		for range ch {
		}
		close(drained)
	}()

	const settle = 1500 * time.Millisecond

	if redMode() {
		select {
		case <-drained:
			t.Fatalf("ensemble: RED expectation failed: the channel drain completed (ch was closed) "+
				"within %s of the e.Generate error — the close-on-error defect should be reproducible "+
				"here (ch should never close). If this fails, the defect is already fixed; flip "+
				"RED_MODE=0.", settle)
		case <-time.After(settle):
			// Expected on the pre-fix artifact: the drain loop hangs forever
			// because ch is never closed on the error path.
		}
		return
	}

	select {
	case <-drained:
		// GREEN: ch was closed on the error path, so `for range ch` returned.
	case <-time.After(settle):
		t.Fatalf("ensemble: GREEN failed: the channel drain did not complete within %s of the "+
			"e.Generate error — GenerateStream did not close ch on the error path, so every real "+
			"caller's `for resp := range ch` hangs forever. GenerateStream must "+
			"`defer close(ch)` on every return path.", settle)
	}
}

// TestEnsembleProvider_GenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain is
// the §11.4.115 RED-baseline + polarity-switch regression guard for DEFECT 2
// (unguarded send / goroutine leak) in ensemble_provider.go's GenerateStream.
func TestEnsembleProvider_GenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain(t *testing.T) {
	member := &ensembleStubProvider{
		ptype:   ProviderTypeGroq,
		name:    "sendleak-member",
		content: "member response",
		finish:  "stop",
		tokens:  1,
	}
	ens := NewEnsembleProvider(EnsembleProviderConfig{
		Members: []Provider{member},
		Timeout: 5 * time.Second,
	})

	req := &LLMRequest{
		ID:       uuid.New(),
		Model:    EnsembleModelName,
		Messages: []Message{{Role: "user", Content: "hi"}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// UNBUFFERED and deliberately never drained: ensemble's GenerateStream
	// sends exactly ONE response (the voted result), so — unlike the
	// per-chunk streaming providers — there is no "drain a few, then stop"
	// step; an unbuffered, undrained channel is the purest reproduction of
	// "ch is caller-supplied and may be ... already abandoned" (the fix's own
	// commit-message wording).
	ch := make(chan LLMResponse)

	go func() {
		_ = ens.GenerateStream(ctx, req, ch)
	}()

	// Prove the pipe is genuinely live before asserting on goroutine state —
	// otherwise a setup failure (e.g. the member never being called) could
	// masquerade as "never parks", a false GREEN. member.Generate returns
	// immediately (no delay configured), so observing the call proves
	// GenerateStream has reached (or is about to reach) the blocking send.
	liveDeadline := time.After(2 * time.Second)
	for {
		if atomic.LoadInt32(&member.calls) >= 1 {
			break
		}
		select {
		case <-liveDeadline:
			t.Fatal("ensemble: member.Generate was never called within 2s — the fake-provider setup " +
				"is broken, not exercising the send-leak path under test")
		default:
			time.Sleep(2 * time.Millisecond)
		}
	}

	// Give GenerateStream a moment to actually reach (and block on) the
	// unguarded send after the member call it just proved happened.
	time.Sleep(150 * time.Millisecond)

	cancel()

	const settle = 1500 * time.Millisecond
	symbolPrefix := "dev.helix.code/internal/llm.(*EnsembleProvider).GenerateStream"

	if redMode() {
		parked := waitUntilParkedInSend(symbolPrefix, settle)
		require.True(t, parked, "ensemble: RED expectation failed: no goroutine found parked in %s's "+
			"closure (blocked in chan-send state) within %s of ctx cancellation on this artifact — "+
			"the leak should be reproducible here. If this fails, the defect is already fixed; flip "+
			"RED_MODE=0.", symbolPrefix, settle)
		return
	}

	parked := notParkedInSendAfterSettling(symbolPrefix, settle)
	require.False(t, parked, "ensemble: GREEN failed: a goroutine is still parked in %s's closure "+
		"(blocked in chan-send state) %s after ctx cancellation with the channel never drained — an "+
		"unguarded blocking send leaks a goroutine forever. The send must use "+
		"`select { case ch <- *resp: case <-ctx.Done(): return ctx.Err() }`.", symbolPrefix, settle)
}
