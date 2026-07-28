package llm

// provider_stream_send_leak_test.go — §11.4.115 RED-baseline + polarity-switch
// regression guard for the unguarded blocking sends (goroutine + HTTP-body
// leak) in anthropic_provider.go / azure_provider.go / gemini_provider.go /
// vertexai_provider.go / groq_provider.go (§11.4.135: every one of the five
// SSE-parse-helper defects fixed in commit 905a0b0a now carries its own guard
// here, sharing the identical driveProviderSendLeak/streamForeverSSE harness
// since all five are the SAME defect class over an httptest SSE server).
// bedrock_provider.go's processEventStream (AWS-SDK event-stream shaped, not
// plain-HTTP SSE) and ensemble_provider.go's GenerateStream (single-shot send,
// not a per-chunk parse loop, plus its own separate missing-close defect) do
// NOT fit this harness and are guarded independently in
// bedrock_provider_send_leak_test.go and ensemble_provider_send_leak_test.go.
//
// DEFECT (pre-fix): parseStreamingResponse (anthropic, gemini) and
// parseSSEStream (azure) took NO ctx parameter, so every `ch <- LLMResponse{...}`
// send inside those helpers had NO ctx.Done() escape hatch. The outer channel
// is caller-buffered (100 in every real caller in this codebase), which
// raises the bar but does not remove the defect: when the HTTP client
// disconnects mid-stream (the consumer bails on ctx.Done() and stops
// draining — exactly what happens on every real client cancellation) while
// the upstream LLM keeps emitting past the buffered capacity, the provider
// goroutine blocks FOREVER on the unguarded send, leaking the goroutine AND
// the open upstream HTTP response body it holds via its `defer
// httpResp.Body.Close()` that never runs.
//
// FIX: thread ctx into these helpers and guard every send with
// `select { case ch <- v: case <-ctx.Done(): return ctx.Err() }` — the same
// pattern already used by ollama_provider.go / openai_compatible_provider.go
// in this package (see ollama_provider.go's makeStreamingRequest).
//
// REPRODUCTION STRATEGY (per the reference
// submodules/helix_agent/internal/llm/providers/ollama/ollama_completestream_leak_test.go):
// point each real provider at a local httptest.Server whose handler streams
// valid events indefinitely (until the request's own context is done), drain
// a few chunks to prove the pipe is genuinely live, STOP draining so the
// caller-side channel fills and the provider goroutine blocks on its next
// send, THEN cancel ctx, and dump every live goroutine's stack via
// runtime.Stack(all=true) looking for the specific parse-helper's symbol
// blocked in "chan send" state.
//
// §11.4.115 polarity switch via RED_MODE (reusing the package-level redMode()
// helper declared in sp1_redmode_test.go):
//
//	RED_MODE=1        : assert a goroutine IS found parked in the parse
//	                     helper's closure, blocked in "chan send" state,
//	                     after ctx cancellation — proving the defect is
//	                     genuinely reproducible on this artifact.
//	RED_MODE=0 (default): the standing GREEN guard — assert NO such goroutine
//	                     remains, because the fix wraps every send in a
//	                     ctx-guarded select.
//
// CRITICAL (anti-bluff): every case below drives the REAL, exported
// GenerateStream of a REAL provider instance constructed via its real
// NewXProvider constructor, talking to a REAL local HTTP server — never a
// hand-rolled replica of the parse loop.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// goroutineParkedInSend dumps every live goroutine's stack via
// runtime.Stack(buf, all=true) and reports whether any goroutine is currently
// parked (blocked) with a stack frame matching symbolPrefix AND in a
// channel-send wait state. Mirrors the reference technique in
// ollama_completestream_leak_test.go: a deterministic, unforgeable liveness
// probe that does not rely on counting runtime.NumGoroutine() (polluted by
// unrelated goroutines in the test binary).
func goroutineParkedInSend(symbolPrefix string) bool {
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

// waitUntilParkedInSend polls goroutineParkedInSend until it reports true or
// the deadline elapses. Early-exit-on-true is sound here: once an unguarded
// send blocks a goroutine forever, the parked state never reverts on its own,
// so observing true once is conclusive (mirrors waitUntilLeaked in the
// reference ollama test).
func waitUntilParkedInSend(symbolPrefix string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if goroutineParkedInSend(symbolPrefix) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// notParkedInSendAfterSettling sleeps the full settle window (deliberately no
// early exit — a momentary "not parked" reading taken before the goroutine
// even attempts its send would falsely pass on genuinely broken code) and
// then takes a single decisive snapshot.
func notParkedInSendAfterSettling(symbolPrefix string, settle time.Duration) bool {
	time.Sleep(settle)
	return goroutineParkedInSend(symbolPrefix)
}

// streamForeverSSE writes valid SSE `data: <json>\n\n` frames (or, when
// rawJSON is true, raw back-to-back JSON values with no SSE framing — the
// shape anthropic/gemini's json.Decoder-based parse loops require) at a fast
// , steady rate until the request's own context is done. Used as the
// httptest handler for every case below so the upstream "LLM" never stops
// emitting on its own — the only thing that stops the flow is the
// client-side ctx cancellation under test.
func streamForeverSSE(t *testing.T, encode func() []byte, rawSSE bool) http.HandlerFunc {
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

// driveProviderSendLeak is the shared harness: constructs the given real
// Provider, drives its real GenerateStream against a small (2-slot) result
// channel, drains a handful of chunks to prove liveness, stops draining,
// cancels ctx, and asserts (per RED_MODE) whether a goroutine remains parked
// in "chan send" state matching symbolPrefix.
func driveProviderSendLeak(t *testing.T, name, symbolPrefix string, provider Provider, req *LLMRequest) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan LLMResponse, 2)
	go func() {
		_ = provider.GenerateStream(ctx, req, ch)
	}()

	// Prove the pipe is genuinely live before we stop draining — otherwise a
	// request-setup failure could masquerade as "never parks", a false
	// GREEN.
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
	if redMode() {
		parked := waitUntilParkedInSend(symbolPrefix, settle)
		require.True(t, parked, "%s: RED expectation failed: no goroutine found parked in %s's "+
			"closure (blocked in chan-send state) within %s of ctx cancellation on this artifact — "+
			"the leak should be reproducible here. If this fails, the defect is already fixed; flip "+
			"RED_MODE=0.", name, symbolPrefix, settle)
		return
	}

	parked := notParkedInSendAfterSettling(symbolPrefix, settle)
	require.False(t, parked, "%s: GREEN failed: a goroutine is still parked in %s's closure "+
		"(blocked in chan-send state) %s after ctx cancellation with the channel never drained — an "+
		"unguarded blocking send leaks a goroutine (and the open HTTP response body it holds) "+
		"forever. Every send site must use "+
		"`select { case ch <- resp: case <-ctx.Done(): return ctx.Err() }`.", name, symbolPrefix, settle)
}

func TestProviderGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain(t *testing.T) {
	t.Run("anthropic", func(t *testing.T) {
		server := httptest.NewServer(streamForeverSSE(t, func() []byte {
			b, _ := json.Marshal(anthropicStreamEvent{
				Type:  "content_block_delta",
				Delta: &anthropicDelta{Type: "text_delta", Text: "x"},
			})
			return b
		}, false)) // anthropic's parseStreamingResponse decodes raw JSON via json.Decoder, no SSE framing.
		defer server.Close()

		provider, err := NewAnthropicProvider(ProviderConfigEntry{
			Type:     ProviderTypeAnthropic,
			Endpoint: server.URL,
			APIKey:   "test-key",
		})
		require.NoError(t, err)

		req := &LLMRequest{
			ID:        uuid.New(),
			Model:     "claude-3-5-sonnet-latest",
			Messages:  []Message{{Role: "user", Content: "hi"}},
			MaxTokens: 100,
			Stream:    true,
		}

		driveProviderSendLeak(t, "anthropic",
			"dev.helix.code/internal/llm.(*AnthropicProvider).parseStreamingResponse", provider, req)
	})

	t.Run("azure", func(t *testing.T) {
		server := httptest.NewServer(streamForeverSSE(t, func() []byte {
			b, _ := json.Marshal(map[string]interface{}{
				"choices": []map[string]interface{}{
					{"delta": map[string]interface{}{"content": "x"}},
				},
			})
			return b
		}, true)) // azure's parseSSEStream scans `data: {...}` lines via bufio.Scanner.
		defer server.Close()

		provider, err := NewAzureProvider(ProviderConfigEntry{
			Type: ProviderTypeAzure,
			Parameters: map[string]interface{}{
				"endpoint": server.URL,
			},
			APIKey: "test-key",
		})
		require.NoError(t, err)

		req := &LLMRequest{
			ID:        uuid.New(),
			Model:     "test-deployment",
			Messages:  []Message{{Role: "user", Content: "hi"}},
			MaxTokens: 100,
			Stream:    true,
		}

		driveProviderSendLeak(t, "azure",
			"dev.helix.code/internal/llm.(*AzureProvider).parseSSEStream", provider, req)
	})

	t.Run("gemini", func(t *testing.T) {
		server := httptest.NewServer(streamForeverSSE(t, func() []byte {
			b, _ := json.Marshal(geminiResponse{
				Candidates: []geminiCandidate{
					{Content: geminiContent{Parts: []geminiPart{map[string]interface{}{"text": "x"}}}},
				},
			})
			return b
		}, false)) // gemini's parseStreamingResponse decodes raw JSON via json.Decoder, no SSE framing.
		defer server.Close()

		provider, err := NewGeminiProvider(ProviderConfigEntry{
			Type:     ProviderTypeGemini,
			Endpoint: server.URL,
			APIKey:   "test-key",
		})
		require.NoError(t, err)

		req := &LLMRequest{
			ID:        uuid.New(),
			Model:     "gemini-2.5-flash",
			Messages:  []Message{{Role: "user", Content: "hi"}},
			MaxTokens: 100,
			Stream:    true,
		}

		driveProviderSendLeak(t, "gemini",
			"dev.helix.code/internal/llm.(*GeminiProvider).parseStreamingResponse", provider, req)
	})

	t.Run("groq", func(t *testing.T) {
		server := httptest.NewServer(streamForeverSSE(t, func() []byte {
			b, _ := json.Marshal(map[string]interface{}{
				"choices": []map[string]interface{}{
					{"delta": map[string]interface{}{"content": "x"}},
				},
			})
			return b
		}, true)) // groq's parseSSEStreamWithMetrics scans `data: {...}` lines via bufio.Scanner, identical to azure.
		defer server.Close()

		provider, err := NewGroqProvider(ProviderConfigEntry{
			Type:     ProviderTypeGroq,
			Endpoint: server.URL,
			APIKey:   "test-key",
		})
		require.NoError(t, err)

		req := &LLMRequest{
			ID:        uuid.New(),
			Model:     "llama-3.1-8b-instant",
			Messages:  []Message{{Role: "user", Content: "hi"}},
			MaxTokens: 100,
			Stream:    true,
		}

		driveProviderSendLeak(t, "groq",
			"dev.helix.code/internal/llm.(*GroqProvider).parseSSEStreamWithMetrics", provider, req)
	})

	t.Run("vertexai", func(t *testing.T) {
		server := httptest.NewServer(streamForeverSSE(t, func() []byte {
			b, _ := json.Marshal(map[string]interface{}{
				"candidates": []map[string]interface{}{
					{"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "x"},
						},
					}},
				},
			})
			return b
		}, true)) // vertexai's parseSSEStream scans `data: {...}` lines via bufio.Scanner, identical to azure/groq.
		defer server.Close()

		// createMockVertexAIProviderWithEndpoint (vertexai_provider_test.go, same
		// package) constructs a REAL *VertexAIProvider with a pre-cached OAuth2
		// token — this is the established in-package pattern for driving
		// VertexAIProvider's real methods without a live GCP credential/token
		// exchange (see TestVertexAIProvider_StreamingGemini using the identical
		// helper). It is NOT a replica of parseSSEStream: GenerateStream →
		// generateGeminiStream → parseSSEStream all run for real against our
		// httptest server; only the OAuth2 token source (irrelevant to the
		// send-leak defect under test) is pre-seeded.
		provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

		req := &LLMRequest{
			ID:        uuid.New(),
			Model:     "gemini-2.5-flash",
			Messages:  []Message{{Role: "user", Content: "hi"}},
			MaxTokens: 100,
			Stream:    true,
		}

		driveProviderSendLeak(t, "vertexai",
			"dev.helix.code/internal/llm.(*VertexAIProvider).parseSSEStream", provider, req)
	})
}
