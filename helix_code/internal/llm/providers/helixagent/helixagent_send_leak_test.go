package helixagent

// helixagent_send_leak_test.go — §11.4.115 RED-baseline + polarity-switch
// regression guard registering THIS provider into the SHARED send-leak harness
// (dev.helix.code/internal/llm/streamleak), the same harness that guards
// anthropic / azure / gemini / groq / vertexai in
// internal/llm/provider_stream_send_leak_test.go. Deliberately NOT a bespoke
// copy: a divergent local re-implementation is exactly how a provider ends up
// "covered" by a guard that quietly asserts something weaker.
//
// DEFECT (pre-fix, HXC-183): Provider.GenerateStream forwarded every decoded
// SSE chunk with a BARE, unguarded send —
//
//	ch <- llm.LLMResponse{...}   (the per-delta chunk send)
//	ch <- final                  (the finish_reason send)
//
// — with no `case <-ctx.Done()` escape hatch. Commit 905a0b0a fixed exactly
// this defect class and 97d5ad2b guarded it across six sibling providers, but
// the fan-out stopped at the internal/llm package boundary and never reached
// this sub-package. When the caller cancels ctx and stops draining (precisely
// what happens on every real client disconnect / timeout) while the upstream
// HelixAgent server keeps emitting past the channel's buffered capacity, this
// goroutine blocks FOREVER on its next send — leaking the goroutine AND the
// open upstream HTTP response body it holds via the `defer resp.Body.Close()`
// that consequently never runs. Every cancelled stream against this provider
// permanently consumed a little more of the process until restart.
//
// FIX: guard both sends with
// `select { case ch <- v: case <-ctx.Done(): return ctx.Err() }` — the exact
// pattern already used by the six siblings (see groq_provider.go's
// parseSSEStreamWithMetrics) and by ollama_provider.go /
// openai_compatible_provider.go before them.
//
// §11.4.115 polarity switch via RED_MODE (streamleak.RedMode, the same
// environment contract as internal/llm's package-level redMode()):
//
//	RED_MODE=1        : assert a goroutine IS parked in GenerateStream, blocked
//	                     in "chan send" state, after ctx cancellation — proving
//	                     the defect is genuinely reproducible on this artifact.
//	RED_MODE=0 (default): the standing GREEN guard — assert NO such goroutine
//	                     remains, because both sends are now ctx-guarded.
//
// CRITICAL (anti-bluff): this drives the REAL, exported GenerateStream of a
// REAL *Provider built by the REAL New() constructor, talking to a REAL local
// HTTP server over a REAL socket. Nothing about the SSE scan loop, the chunk
// decode, or the send sites is replicated in this file.

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/llm/streamleak"
	"github.com/google/uuid"
)

func TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain(t *testing.T) {
	// The fake HelixAgent server emits a valid OpenAI-shape streaming chunk
	// (`data: {...}\n\n`) forever — GenerateStream scans lines with
	// bufio.Scanner and requires the `data:` prefix, so rawSSE=true is the
	// correct framing here (identical to the groq/azure/vertexai cases in the
	// shared harness). It never sends `[DONE]` and never sends a
	// finish_reason, so the loop can only be stopped by the client-side ctx
	// cancellation under test.
	server := httptest.NewServer(streamleak.StreamForeverSSE(func() []byte {
		b, _ := json.Marshal(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "x"}},
			},
		})
		return b
	}, true))
	defer server.Close()

	provider := New(server.URL)

	req := &llm.LLMRequest{
		ID:        uuid.New(),
		Model:     DefaultModel,
		Messages:  []llm.Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
		Stream:    true,
	}

	streamleak.DriveSendLeak(t, "helixagent",
		"dev.helix.code/internal/llm/providers/helixagent.(*Provider).GenerateStream",
		func(ctx context.Context, ch chan<- llm.LLMResponse) error {
			return provider.GenerateStream(ctx, req, ch)
		})
}
