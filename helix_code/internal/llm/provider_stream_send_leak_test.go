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

// HARNESS LOCATION (moved 2026-07-29, HXC-183): the probe/handler/driver
// helpers below are now THIN DELEGATIONS to the shared, generic
// dev.helix.code/internal/llm/streamleak package. They were hoisted out of this
// file because providers/* sub-packages (which import internal/llm for the
// shared Provider types) cannot be driven from a `package llm` test file — Go
// rejects it with "import cycle not allowed in test" (verified empirically).
// That wall is precisely why the 905a0b0a/97d5ad2b round left
// providers/helixagent unguarded. The wrappers are kept at their original
// names and signatures so every existing call site in this file, in
// bedrock_provider_send_leak_test.go and in ensemble_provider_send_leak_test.go
// is untouched; the harness BEHAVIOUR is now single-sourced and shared with
// providers/helixagent/helixagent_send_leak_test.go.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dev.helix.code/internal/llm/streamleak"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// waitUntilParkedInSend polls the shared stack probe until it reports a
// goroutine parked in "chan send" state matching symbolPrefix, or the deadline
// elapses. Delegates to streamleak; also used by
// bedrock_provider_send_leak_test.go and ensemble_provider_send_leak_test.go.
func waitUntilParkedInSend(symbolPrefix string, timeout time.Duration) bool {
	return streamleak.WaitUntilParkedInSend(symbolPrefix, timeout)
}

// notParkedInSendAfterSettling sleeps the full settle window and then takes a
// single decisive snapshot. Delegates to streamleak; also used by
// bedrock_provider_send_leak_test.go and ensemble_provider_send_leak_test.go.
func notParkedInSendAfterSettling(symbolPrefix string, settle time.Duration) bool {
	return streamleak.ParkedInSendAfterSettling(symbolPrefix, settle)
}

// streamForeverSSE writes valid SSE `data: <json>\n\n` frames (or, when
// rawJSON is true, raw back-to-back JSON values with no SSE framing — the
// shape anthropic/gemini's json.Decoder-based parse loops require) at a fast
// , steady rate until the request's own context is done. Used as the
// httptest handler for every case below so the upstream "LLM" never stops
// emitting on its own — the only thing that stops the flow is the
// client-side ctx cancellation under test.
func streamForeverSSE(t *testing.T, encode func() []byte, rawSSE bool) http.HandlerFunc {
	t.Helper()
	return streamleak.StreamForeverSSE(encode, rawSSE)
}

// driveProviderSendLeak is the shared harness: constructs the given real
// Provider, drives its real GenerateStream against a small (2-slot) result
// channel, drains a handful of chunks to prove liveness, stops draining,
// cancels ctx, and asserts (per RED_MODE) whether a goroutine remains parked
// in "chan send" state matching symbolPrefix.
func driveProviderSendLeak(t *testing.T, name, symbolPrefix string, provider Provider, req *LLMRequest) {
	t.Helper()
	streamleak.DriveSendLeak(t, name, symbolPrefix, func(ctx context.Context, ch chan<- LLMResponse) error {
		return provider.GenerateStream(ctx, req, ch)
	})
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
