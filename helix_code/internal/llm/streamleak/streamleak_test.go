package streamleak

// streamleak_test.go — §11.4.107(10) SELF-VALIDATION of the harness itself.
//
// The seven provider guards that call this package all rest on ONE assumption:
// that ParkedInSend can actually tell an unguarded blocking send apart from a
// ctx-guarded one. If that probe were broken — if it silently reported "not
// parked" for everything — every guard built on it would report a cheerful
// GREEN on genuinely leaking code. An unvalidated analyzer is itself a bluff
// gate, so this file pins the probe against a golden-BAD / golden-GOOD fixture
// pair:
//
//	golden-BAD  : a bare `ch <- v` on a full channel  => MUST be detected
//	golden-GOOD : the ctx-guarded select              => MUST NOT be detected
//
// SCOPE (important, so this is not mistaken for a provider guard): these two
// fixtures deliberately ARE local, minimal send loops — they are the
// analyzer's test INPUT, not a stand-in for any provider. Nothing here
// substitutes for the real provider guards, which drive real constructors and
// real exported streaming methods over real sockets (see
// internal/llm/provider_stream_send_leak_test.go and
// internal/llm/providers/helixagent/helixagent_send_leak_test.go). This file
// tests the ruler; those files do the measuring.

import (
	"context"
	"testing"
	"time"
)

// goldenBadUnguardedSend reproduces the DEFECT shape: a bare send with no
// ctx.Done() escape hatch. Given a channel smaller than n it parks forever on
// the first send past capacity — exactly the state the probe must catch.
func goldenBadUnguardedSend(ch chan<- int, n int) {
	for i := 0; i < n; i++ {
		ch <- i
	}
}

// goldenGoodGuardedSend reproduces the FIXED shape: every send carries a
// ctx.Done() escape, so a cancelled ctx unwinds the goroutine instead of
// parking it.
func goldenGoodGuardedSend(ctx context.Context, ch chan<- int, n int) error {
	for i := 0; i < n; i++ {
		select {
		case ch <- i:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestParkedInSend_SelfValidation(t *testing.T) {
	const badSym = "dev.helix.code/internal/llm/streamleak.goldenBadUnguardedSend"
	const goodSym = "dev.helix.code/internal/llm/streamleak.goldenGoodGuardedSend"

	t.Run("golden_bad_unguarded_send_IS_detected", func(t *testing.T) {
		ch := make(chan int, 1)
		go goldenBadUnguardedSend(ch, 3)

		if !WaitUntilParkedInSend(badSym, 2*time.Second) {
			t.Fatalf("SELF-VALIDATION FAILED: the probe did NOT detect a goroutine parked on a "+
				"bare unguarded send (%s). The probe is blind, so every provider guard that calls "+
				"it is reporting GREEN on faith rather than on evidence.", badSym)
		}

		// Release the parked goroutine so it does not outlive the test.
		for i := 0; i < 3; i++ {
			<-ch
		}
	})

	t.Run("golden_good_guarded_send_is_NOT_detected", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan int, 1)
		done := make(chan error, 1)
		go func() { done <- goldenGoodGuardedSend(ctx, ch, 3) }()

		// Let it fill the buffer and reach the blocking send, then cancel —
		// the guarded select must take the ctx.Done() branch.
		time.Sleep(100 * time.Millisecond)
		cancel()

		if ParkedInSendAfterSettling(goodSym, 500*time.Millisecond) {
			t.Fatalf("SELF-VALIDATION FAILED: the probe reported a goroutine parked in %s, but "+
				"that send IS ctx-guarded and ctx was cancelled — the probe yields false positives, "+
				"which would make every provider guard fail on correct code.", goodSym)
		}

		select {
		case err := <-done:
			if err != context.Canceled {
				t.Fatalf("golden-good fixture returned %v, want context.Canceled — the fixture "+
					"itself is not exercising the ctx.Done() branch, so this case proves nothing", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("golden-good fixture never returned after ctx cancellation")
		}
	})
}

func TestRedMode_ReadsEnvContract(t *testing.T) {
	t.Setenv("RED_MODE", "1")
	if !RedMode() {
		t.Fatal("RedMode() must report true when RED_MODE=1 — the §11.4.115 polarity switch is " +
			"inert, so a RED run would silently execute the GREEN assertion")
	}
	t.Setenv("RED_MODE", "0")
	if RedMode() {
		t.Fatal("RedMode() must report false when RED_MODE=0")
	}
	t.Setenv("RED_MODE", "")
	if RedMode() {
		t.Fatal("RedMode() must default to false (standing GREEN guard) when RED_MODE is unset")
	}
}
