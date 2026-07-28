package llm

// tool_provider_streamwithtools_deadlock_test.go — §11.4.115 RED-baseline +
// polarity-switch regression guard for the guaranteed deadlock in
// StreamWithTools (tool_provider.go).
//
// DEFECT (pre-fix, tool_provider.go's StreamWithTools, ~line 240 and ~line
// 335): the Provider interface's documented contract (doc.go:83-93) requires
// GenerateStream be invoked inside its own goroutine so the caller can drain
// the response channel CONCURRENTLY while it streams. The pre-fix code called
// `p.baseProvider.GenerateStream(ctx, streamReq, streamCh)` INLINE
// (synchronously) and only began draining streamCh in a `for resp := range
// streamCh` loop AFTER GenerateStream returned. streamCh is buffered at 100;
// any response exceeding 100 chunks made the provider's internal
// `ch <- LLMResponse{...}` block FOREVER on a channel nobody was draining yet
// — a DETERMINISTIC hang for any tool-augmented generation whose response
// exceeds 100 chunks (routine for anything beyond a short reply), not a
// probabilistic race. The SAME defect existed a second time in the same
// function, in the "stream the final response after executing tool calls"
// branch (pre-fix ~line 335).
//
// FIX: launch GenerateStream in its own goroutine (both call sites) so the
// drain loop runs concurrently, exactly per doc.go's documented contract.
//
// §11.4.115 polarity switch via RED_MODE (reusing the package-level redMode()
// helper declared in sp1_redmode_test.go):
//
//	RED_MODE=1        : assert the deadlock IS reproducible — StreamWithTools
//	                     does NOT finish draining a >100-chunk response within
//	                     the bounded timeout.
//	RED_MODE=0 (default): the standing GREEN regression guard — assert
//	                     StreamWithTools DOES finish draining every chunk
//	                     within the bounded timeout.
//
// CRITICAL (anti-bluff): this test drives the REAL, production
// ToolCallingProvider.StreamWithTools — not a hand-rolled replica of the old
// synchronous-call-then-drain pattern. A replica-based RED assertion proves
// nothing about the actual shipped code path and is a bluff gate. The fake
// baseProvider below exists ONLY to emit >100 chunks through the REAL
// StreamWithTools call; its own GenerateStream is a correct (ctx-guarded)
// implementation, isolating StreamWithTools's own defect from the six
// provider-file defects fixed alongside it (see the *_send_leak_test.go /
// equivalent guards for those).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// swtManyChunks exceeds streamCh's 100-slot buffer so the pre-fix
// synchronous-call-then-drain ordering is guaranteed to block forever on the
// 101st send, rather than merely being likely to under a race detector.
const swtManyChunks = 250

// swtManyChunkProvider is a minimal, CORRECT (ctx-guarded) Provider whose
// GenerateStream emits swtManyChunks small chunks. It exists only to drive
// StreamWithTools with a response long enough to expose StreamWithTools's OWN
// synchronous-call-then-drain defect; it does not itself exhibit the
// unguarded-send defect fixed in anthropic/azure/gemini/bedrock/groq/vertexai
// (that family is covered by the reproduction/regression tests alongside
// those files).
type swtManyChunkProvider struct{}

func (p *swtManyChunkProvider) GetType() ProviderType              { return ProviderTypeOllama }
func (p *swtManyChunkProvider) GetName() string                    { return "swt-many-chunk-fake" }
func (p *swtManyChunkProvider) GetModels() []ModelInfo             { return nil }
func (p *swtManyChunkProvider) GetCapabilities() []ModelCapability { return nil }

func (p *swtManyChunkProvider) Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	return &LLMResponse{Content: "unused"}, nil
}

// GenerateStream is intentionally ctx-guarded (correct) so that any hang
// observed in the test below is attributable ONLY to StreamWithTools's own
// call-then-drain ordering, not to this fake's send discipline.
func (p *swtManyChunkProvider) GenerateStream(ctx context.Context, req *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)
	for i := 0; i < swtManyChunks; i++ {
		select {
		case ch <- LLMResponse{Content: "x"}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (p *swtManyChunkProvider) IsAvailable(ctx context.Context) bool { return true }
func (p *swtManyChunkProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	return &ProviderHealth{Status: "healthy"}, nil
}
func (p *swtManyChunkProvider) Close() error                         { return nil }
func (p *swtManyChunkProvider) GetContextWindow() int                { return 8192 }
func (p *swtManyChunkProvider) CountTokens(text string) (int, error) { return len(text), nil }

// TestStreamWithTools_ManyChunks_DoesNotDeadlock drives the REAL production
// ToolCallingProvider.StreamWithTools with a backing provider emitting more
// chunks than streamCh's 100-slot buffer, and asserts (per RED_MODE) whether
// the call completes within a bounded timeout.
func TestStreamWithTools_ManyChunks_DoesNotDeadlock(t *testing.T) {
	tcp := NewToolCallingProvider(&swtManyChunkProvider{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := tcp.StreamWithTools(ctx, ToolGenerationRequest{
		ID:     uuid.New(),
		Prompt: "drive many chunks through StreamWithTools",
	})
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Drain the RETURNED channel concurrently — exactly what a real caller
	// does, and required so the only possible hang under test is
	// StreamWithTools's own inner streamCh deadlock (an undrained outer `ch`
	// would itself block on ordinary backpressure, which is not the defect
	// under test).
	done := make(chan int, 1)
	go func() {
		n := 0
		for range ch {
			n++
		}
		done <- n
	}()

	const timeout = 3 * time.Second
	select {
	case n := <-done:
		// StreamWithTools finished draining within the timeout: the
		// deadlock is ABSENT on this artifact.
		if redMode() {
			t.Fatalf("RED expectation failed: StreamWithTools drained %d chunks and completed "+
				"within %s on this artifact — the deadlock did not reproduce. If this fails, the "+
				"defect is already fixed; flip RED_MODE=0.", n, timeout)
		}
		require.GreaterOrEqual(t, n, 1, "GREEN: expected at least one forwarded chunk, got %d", n)
	case <-time.After(timeout):
		// StreamWithTools did NOT finish draining within the timeout: it is
		// hung — the deadlock is PRESENT on this artifact.
		if !redMode() {
			t.Fatalf("GREEN failed: StreamWithTools did not finish draining %d chunks within %s — it "+
				"deadlocked. StreamWithTools MUST invoke baseProvider.GenerateStream in its own "+
				"goroutine and drain streamCh concurrently, per doc.go's documented Provider contract "+
				"(doc.go:83-93).", swtManyChunks, timeout)
		}
		// RED_MODE=1 and StreamWithTools hung: this is the expected
		// reproduction of the historical defect.
	}
}
