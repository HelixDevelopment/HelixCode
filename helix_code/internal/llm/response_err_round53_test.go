// Round-53 §11.4 anti-bluff tests for LLMResponse.Err propagation —
// extends round-46 (commit d39251f) + round-50 (commit 993fd1e) to
// four more providers:
//   - xAI (Grok)               finish_reason: length / content_filter (OpenAI-compatible)
//   - OpenRouter (proxy)       finish_reason: length / content_filter (OpenAI-compatible normalised)
//   - Llama.cpp (local)        OpenAI-compat /v1/completions + legacy /completion stop flags
//   - OpenAICompatibleProvider finish_reason: length / content_filter (covers VLLM, LMStudio,
//                                                                     Jan, LocalAI, FastChat,
//                                                                     TextGen, KoboldAI, GPT4All,
//                                                                     TabbyAPI, MLX, MistralRS —
//                                                                     11 backends in one wire)
//
// Cerebras was the original 4th target but no `cerebras_provider.go` exists
// in `helix_code/internal/llm/` — substituted with `OpenAICompatibleProvider`
// (documented in commit body; deferred for a future round once a Cerebras
// provider file is introduced).
//
// Round-46 wired openai / anthropic / ollama (the top three providers);
// round-50 closed 4 more (gemini / deepseek / groq / mistral); round-53
// closes 4 more for 11/17 deferred providers wired (~65%), leaving 6 of
// the original 13 deferred for round-54+: Bedrock, Azure, Replicate,
// Vertex AI, Qwen, Copilot. (Note: OpenAICompatibleProvider replaces 11
// of the "Local" / generic-OpenAI-compatible items in one wire.)
//
// CONST-035 / CONST-050(A)+(B) / Article XI §11.9: every PASS in this
// file is backed by an httptest fixture that exercises the real provider
// HTTP code path (real JSON encode/decode, real http.Client, real
// LLMResponse construction). No mocks of internal helpers — only the
// remote API is faked at the HTTP transport boundary, which is the
// canonical pattern for provider-layer unit tests.
package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// xAI (Grok) — OpenAI-compatible
// =========================================================================

// TestRound53_XAI_Generate_FinishReasonLength_PopulatesTruncated asserts
// that xAI's "length" finish_reason maps to ErrResponseTruncated and that
// partial Content survives.
func TestRound53_XAI_Generate_FinishReasonLength_PopulatesTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id": "chatcmpl-round53-xai-length", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "grok-3-beta",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "xAI Grok partial output here",
					},
					"finish_reason": "length",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 8, "completion_tokens": 5, "total_tokens": 13,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewXAIProvider(ProviderConfigEntry{
		Type:     ProviderTypeXAI,
		APIKey:   "test-key",
		Endpoint: server.URL,
		Enabled:  true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID:        uuid.New(),
		Model:     "grok-3-beta",
		Messages:  []Message{{Role: "user", Content: "test"}},
		MaxTokens: 5,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content, "Content MUST hold partial output even when truncated")
	require.NotNil(t, resp.Err, "Err MUST be populated for finish_reason=length")
	assert.True(t, errors.Is(resp.Err, ErrResponseTruncated),
		"Err MUST be ErrResponseTruncated; got %v", resp.Err)
	assert.Equal(t, "length", resp.FinishReason,
		"FinishReason MUST preserve literal API value alongside Err")
}

// TestRound53_XAI_Generate_ContentFilter_PopulatesBlocked asserts that
// xAI's "content_filter" finish_reason maps to ErrResponseContentBlocked.
func TestRound53_XAI_Generate_ContentFilter_PopulatesBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id": "chatcmpl-round53-xai-cf", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "grok-3-beta",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": ""},
					"finish_reason": "content_filter",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 3, "completion_tokens": 0, "total_tokens": 3,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewXAIProvider(ProviderConfigEntry{
		Type: ProviderTypeXAI, APIKey: "test-key", Endpoint: server.URL, Enabled: true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID: uuid.New(), Model: "grok-3-beta",
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Err)
	assert.True(t, errors.Is(resp.Err, ErrResponseContentBlocked),
		"Err MUST be ErrResponseContentBlocked; got %v", resp.Err)
}

// TestRound53_XAI_Generate_CleanStop_LeavesErrNil asserts clean stop
// leaves Err nil (round-46 backward-compat invariant preserved).
func TestRound53_XAI_Generate_CleanStop_LeavesErrNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id": "ok", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "grok-3-beta",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "Done"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 2, "completion_tokens": 1, "total_tokens": 3,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewXAIProvider(ProviderConfigEntry{
		Type: ProviderTypeXAI, APIKey: "test-key", Endpoint: server.URL, Enabled: true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID: uuid.New(), Model: "grok-3-beta",
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	assert.Nil(t, resp.Err, "Err MUST be nil for clean finish_reason=stop")
	assert.Equal(t, "Done", resp.Content)
}

// TestRound53_XAI_Stream_FinishReasonLength_PropagatesToTerminalFrame
// asserts that streaming-path truncation emits a terminal Err-bearing
// frame on the channel.
func TestRound53_XAI_Stream_FinishReasonLength_PropagatesToTerminalFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Per-token chunks then terminal chunk with finish_reason=length
		chunks := []map[string]interface{}{
			{
				"id": "s1", "object": "chat.completion.chunk", "model": "grok-3-beta",
				"choices": []map[string]interface{}{
					{"index": 0, "delta": map[string]interface{}{"role": "assistant", "content": "Hello"}, "finish_reason": ""},
				},
			},
			{
				"id": "s2", "object": "chat.completion.chunk", "model": "grok-3-beta",
				"choices": []map[string]interface{}{
					{"index": 0, "delta": map[string]interface{}{"content": " wo"}, "finish_reason": ""},
				},
			},
			{
				"id": "s3", "object": "chat.completion.chunk", "model": "grok-3-beta",
				"choices": []map[string]interface{}{
					{"index": 0, "delta": map[string]interface{}{}, "finish_reason": "length"},
				},
			},
		}
		enc := json.NewEncoder(w)
		for _, c := range chunks {
			_ = enc.Encode(c)
		}
	}))
	defer server.Close()

	provider, err := NewXAIProvider(ProviderConfigEntry{
		Type: ProviderTypeXAI, APIKey: "test-key", Endpoint: server.URL, Enabled: true,
	})
	require.NoError(t, err)

	ch := make(chan LLMResponse, 16)
	go func() {
		_ = provider.GenerateStream(context.Background(), &LLMRequest{
			ID: uuid.New(), Model: "grok-3-beta", Stream: true,
			Messages: []Message{{Role: "user", Content: "Hello"}},
		}, ch)
	}()

	var sawErrFrame bool
	for resp := range ch {
		if resp.Err != nil {
			sawErrFrame = true
			assert.True(t, errors.Is(resp.Err, ErrResponseTruncated),
				"terminal stream Err MUST be ErrResponseTruncated; got %v", resp.Err)
			assert.Equal(t, "length", resp.FinishReason,
				"terminal stream FinishReason MUST be preserved")
		}
	}
	assert.True(t, sawErrFrame, "stream MUST emit a terminal Err-bearing frame on finish_reason=length")
}

// =========================================================================
// OpenRouter (OpenAI-compatible proxy)
// =========================================================================

// TestRound53_OpenRouter_Generate_FinishReasonLength_PopulatesTruncated
// asserts that OpenRouter's "length" finish_reason maps to
// ErrResponseTruncated regardless of which upstream provider the proxy
// routed to.
func TestRound53_OpenRouter_Generate_FinishReasonLength_PopulatesTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OpenRouter's /models endpoint hits during init via fetchCatalog
		// (10s timeout). Mock both /models and /chat/completions.
		if strings.HasSuffix(r.URL.Path, "/models") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "openai/gpt-oss-20b:free", "name": "GPT-OSS 20B Free",
						"context_length": 131072,
						"top_provider":   map[string]interface{}{"max_completion_tokens": 4096}},
				},
			})
			return
		}
		response := map[string]interface{}{
			"id": "chatcmpl-round53-or-length", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "openai/gpt-oss-20b:free",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "OpenRouter partial output",
					},
					"finish_reason": "length",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 6, "completion_tokens": 3, "total_tokens": 9,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewOpenRouterProvider(ProviderConfigEntry{
		Type:     ProviderTypeOpenRouter,
		APIKey:   "test-key",
		Endpoint: server.URL,
		Enabled:  true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID: uuid.New(), Model: "openai/gpt-oss-20b:free",
		Messages:  []Message{{Role: "user", Content: "test"}},
		MaxTokens: 3,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content, "Content MUST hold partial output")
	require.NotNil(t, resp.Err)
	assert.True(t, errors.Is(resp.Err, ErrResponseTruncated),
		"Err MUST be ErrResponseTruncated; got %v", resp.Err)
	assert.Equal(t, "length", resp.FinishReason)
}

// TestRound53_OpenRouter_Generate_ContentFilter_PopulatesBlocked asserts
// content_filter mapping.
func TestRound53_OpenRouter_Generate_ContentFilter_PopulatesBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "openai/gpt-oss-20b:free", "name": "GPT-OSS 20B Free",
						"context_length": 131072,
						"top_provider":   map[string]interface{}{"max_completion_tokens": 4096}},
				},
			})
			return
		}
		response := map[string]interface{}{
			"id": "chatcmpl-round53-or-cf", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "openai/gpt-oss-20b:free",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": ""},
					"finish_reason": "content_filter",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 4, "completion_tokens": 0, "total_tokens": 4,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewOpenRouterProvider(ProviderConfigEntry{
		Type: ProviderTypeOpenRouter, APIKey: "test-key", Endpoint: server.URL, Enabled: true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID: uuid.New(), Model: "openai/gpt-oss-20b:free",
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Err)
	assert.True(t, errors.Is(resp.Err, ErrResponseContentBlocked),
		"Err MUST be ErrResponseContentBlocked; got %v", resp.Err)
}

// TestRound53_OpenRouter_Generate_CleanStop_LeavesErrNil asserts clean
// stop leaves Err nil.
func TestRound53_OpenRouter_Generate_CleanStop_LeavesErrNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "openai/gpt-oss-20b:free", "name": "GPT-OSS 20B Free",
						"context_length": 131072,
						"top_provider":   map[string]interface{}{"max_completion_tokens": 4096}},
				},
			})
			return
		}
		response := map[string]interface{}{
			"id": "ok", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "openai/gpt-oss-20b:free",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "OK"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 2, "completion_tokens": 1, "total_tokens": 3,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewOpenRouterProvider(ProviderConfigEntry{
		Type: ProviderTypeOpenRouter, APIKey: "test-key", Endpoint: server.URL, Enabled: true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID: uuid.New(), Model: "openai/gpt-oss-20b:free",
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	assert.Nil(t, resp.Err, "Err MUST be nil for clean finish_reason=stop")
}

// =========================================================================
// Llama.cpp — legacy stop-flag path + OpenAI-compat path
// =========================================================================

// TestRound53_LlamaCpp_Generate_LegacyStoppedLimit_PopulatesTruncated
// asserts that a top-level `stopped_limit:true` from llama.cpp's legacy
// /v1/completions response shape maps to ErrResponseTruncated.
func TestRound53_LlamaCpp_Generate_LegacyStoppedLimit_PopulatesTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"content":       "llama.cpp partial",
			"stopped_eos":   false,
			"stopped_limit": true,
			"stopped_word":  false,
			"usage": map[string]interface{}{
				"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	host, port := mustSplitHostPort(t, server.URL)
	provider, err := NewLlamaCPPProvider(LlamaConfig{
		Model:         "llama-3-8b",
		ContextSize:   4096,
		ServerHost:    host,
		ServerPort:    port,
		ServerTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID:        uuid.New(),
		Model:     "llama-3-8b",
		Messages:  []Message{{Role: "user", Content: "test"}},
		MaxTokens: 3,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content, "Content MUST hold partial output")
	require.NotNil(t, resp.Err, "Err MUST be populated for stopped_limit=true")
	assert.True(t, errors.Is(resp.Err, ErrResponseTruncated),
		"Err MUST be ErrResponseTruncated; got %v", resp.Err)
	assert.Equal(t, "length", resp.FinishReason,
		"FinishReason MUST be synthesised as 'length' for stopped_limit=true")
}

// TestRound53_LlamaCpp_Generate_OpenAICompatLength_PopulatesTruncated
// asserts that the OpenAI-compatible choices[].finish_reason="length"
// shape also maps to ErrResponseTruncated.
func TestRound53_LlamaCpp_Generate_OpenAICompatLength_PopulatesTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "compat-shape partial",
					},
					"finish_reason": "length",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 6, "completion_tokens": 2, "total_tokens": 8,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	host, port := mustSplitHostPort(t, server.URL)
	provider, err := NewLlamaCPPProvider(LlamaConfig{
		Model:         "llama-3-8b",
		ContextSize:   4096,
		ServerHost:    host,
		ServerPort:    port,
		ServerTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID:        uuid.New(),
		Model:     "llama-3-8b",
		Messages:  []Message{{Role: "user", Content: "test"}},
		MaxTokens: 2,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Err)
	assert.True(t, errors.Is(resp.Err, ErrResponseTruncated),
		"OpenAI-compat finish_reason=length MUST map to ErrResponseTruncated; got %v", resp.Err)
	assert.Equal(t, "length", resp.FinishReason)
	assert.Equal(t, "compat-shape partial", resp.Content)
}

// TestRound53_LlamaCpp_Generate_CleanEOS_LeavesErrNil asserts that
// stopped_eos=true (clean end-of-sequence) leaves Err nil.
func TestRound53_LlamaCpp_Generate_CleanEOS_LeavesErrNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"content":       "complete answer",
			"stopped_eos":   true,
			"stopped_limit": false,
			"stopped_word":  false,
			"usage": map[string]interface{}{
				"prompt_tokens": 4, "completion_tokens": 2, "total_tokens": 6,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	host, port := mustSplitHostPort(t, server.URL)
	provider, err := NewLlamaCPPProvider(LlamaConfig{
		Model:         "llama-3-8b",
		ContextSize:   4096,
		ServerHost:    host,
		ServerPort:    port,
		ServerTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID:       uuid.New(),
		Model:    "llama-3-8b",
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	assert.Nil(t, resp.Err, "Err MUST be nil for stopped_eos=true")
	assert.Equal(t, "stop", resp.FinishReason,
		"FinishReason MUST be synthesised as 'stop' for stopped_eos=true")
	assert.Equal(t, "complete answer", resp.Content)
}

// TestRound53_LlamaCpp_Stream_StoppedLimit_PropagatesToTerminalFrame
// asserts that the streaming /completion SSE path emits a terminal
// Err-bearing frame when the final chunk carries stopped_limit=true.
func TestRound53_LlamaCpp_Stream_StoppedLimit_PropagatesToTerminalFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, _ := w.(http.Flusher)

		// Per-token chunks
		_, _ = fmt.Fprintf(w, "data: {\"content\":\"Hel\"}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprintf(w, "data: {\"content\":\"lo\"}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		// Terminal chunk with stopped_limit:true
		_, _ = fmt.Fprintf(w, "data: {\"content\":\"\",\"stop\":true,\"stopped_eos\":false,\"stopped_limit\":true,\"stopped_word\":false}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	host, port := mustSplitHostPort(t, server.URL)
	provider, err := NewLlamaCPPProvider(LlamaConfig{
		Model:         "llama-3-8b",
		ContextSize:   4096,
		ServerHost:    host,
		ServerPort:    port,
		ServerTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ch := make(chan LLMResponse, 16)
	go func() {
		_ = provider.GenerateStream(context.Background(), &LLMRequest{
			ID: uuid.New(), Model: "llama-3-8b", Stream: true,
			Messages: []Message{{Role: "user", Content: "Hello"}},
		}, ch)
	}()

	var sawErrFrame bool
	deadline := time.After(3 * time.Second)
	for {
		select {
		case resp, ok := <-ch:
			if !ok {
				goto done
			}
			if resp.Err != nil {
				sawErrFrame = true
				assert.True(t, errors.Is(resp.Err, ErrResponseTruncated),
					"terminal stream Err MUST be ErrResponseTruncated; got %v", resp.Err)
				assert.Equal(t, "length", resp.FinishReason)
			}
		case <-deadline:
			goto done
		}
	}
done:
	assert.True(t, sawErrFrame, "stream MUST emit a terminal Err-bearing frame on stopped_limit=true")
}

// =========================================================================
// OpenAI-compatible provider (VLLM / LMStudio / Jan / LocalAI / etc.)
// =========================================================================

// TestRound53_OpenAICompatible_Generate_FinishReasonLength_PopulatesTruncated
// covers the 11 backends fronted by OpenAICompatibleProvider in a single
// fixture (they all advertise OpenAI-compatible finish_reason semantics).
func TestRound53_OpenAICompatible_Generate_FinishReasonLength_PopulatesTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/v1/models") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "llama-3-8b", "object": "model", "owned_by": "vllm"},
				},
			})
			return
		}
		response := map[string]interface{}{
			"id": "chatcmpl-round53-vllm-length", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "llama-3-8b",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "VLLM partial output",
					},
					"finish_reason": "length",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 7, "completion_tokens": 3, "total_tokens": 10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewOpenAICompatibleProvider("vllm", OpenAICompatibleConfig{
		BaseURL:          server.URL,
		Timeout:          5 * time.Second,
		StreamingSupport: true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID: uuid.New(), Model: "llama-3-8b",
		Messages:  []Message{{Role: "user", Content: "test"}},
		MaxTokens: 3,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content, "Content MUST hold partial output")
	require.NotNil(t, resp.Err)
	assert.True(t, errors.Is(resp.Err, ErrResponseTruncated),
		"Err MUST be ErrResponseTruncated; got %v", resp.Err)
	assert.Equal(t, "length", resp.FinishReason)
}

// TestRound53_OpenAICompatible_Generate_ContentFilter_PopulatesBlocked
// asserts content_filter mapping.
func TestRound53_OpenAICompatible_Generate_ContentFilter_PopulatesBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/v1/models") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "llama-3-8b", "object": "model", "owned_by": "lmstudio"},
				},
			})
			return
		}
		response := map[string]interface{}{
			"id": "chatcmpl-round53-lms-cf", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "llama-3-8b",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": ""},
					"finish_reason": "content_filter",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 3, "completion_tokens": 0, "total_tokens": 3,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewOpenAICompatibleProvider("lmstudio", OpenAICompatibleConfig{
		BaseURL:          server.URL,
		Timeout:          5 * time.Second,
		StreamingSupport: true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID: uuid.New(), Model: "llama-3-8b",
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Err)
	assert.True(t, errors.Is(resp.Err, ErrResponseContentBlocked),
		"Err MUST be ErrResponseContentBlocked; got %v", resp.Err)
}

// TestRound53_OpenAICompatible_Generate_CleanStop_LeavesErrNil asserts
// clean stop leaves Err nil.
func TestRound53_OpenAICompatible_Generate_CleanStop_LeavesErrNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/v1/models") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "llama-3-8b", "object": "model", "owned_by": "vllm"},
				},
			})
			return
		}
		response := map[string]interface{}{
			"id": "ok", "object": "chat.completion",
			"created": time.Now().Unix(), "model": "llama-3-8b",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "OK"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens": 2, "completion_tokens": 1, "total_tokens": 3,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewOpenAICompatibleProvider("vllm", OpenAICompatibleConfig{
		BaseURL:          server.URL,
		Timeout:          5 * time.Second,
		StreamingSupport: true,
	})
	require.NoError(t, err)

	resp, err := provider.Generate(context.Background(), &LLMRequest{
		ID: uuid.New(), Model: "llama-3-8b",
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	assert.Nil(t, resp.Err, "Err MUST be nil for clean finish_reason=stop")
}

// =========================================================================
// Paired-mutation mapper-pinning tests (closed-set regression)
// =========================================================================

// TestRound53_LlamaCpp_StopFlagsMapper_AllCases pins the round-53-new
// mapLlamaCppStopFlagsToErr helper. Per CONST-050(B) paired-mutation:
// if a future llama.cpp release adds a new stop flag, this test (or a
// sibling) MUST be extended in the same commit.
func TestRound53_LlamaCpp_StopFlagsMapper_AllCases(t *testing.T) {
	// All false → mid-stream / unknown termination → "", nil
	fr, err := mapLlamaCppStopFlagsToErr(false, false, false)
	assert.Equal(t, "", fr)
	assert.Nil(t, err)

	// stopped_eos=true → clean stop
	fr, err = mapLlamaCppStopFlagsToErr(true, false, false)
	assert.Equal(t, "stop", fr)
	assert.Nil(t, err)

	// stopped_word=true → clean stop (custom stop sequence hit)
	fr, err = mapLlamaCppStopFlagsToErr(false, false, true)
	assert.Equal(t, "stop", fr)
	assert.Nil(t, err)

	// stopped_limit=true → truncation
	fr, err = mapLlamaCppStopFlagsToErr(false, true, false)
	assert.Equal(t, "length", fr)
	require.NotNil(t, err)
	assert.True(t, errors.Is(err, ErrResponseTruncated),
		"stopped_limit=true MUST map to ErrResponseTruncated; got %v", err)

	// stopped_limit takes precedence over stopped_eos (defensive: degraded
	// signal wins so callers don't miss truncation)
	fr, err = mapLlamaCppStopFlagsToErr(true, true, false)
	assert.Equal(t, "length", fr)
	assert.True(t, errors.Is(err, ErrResponseTruncated))
}

// TestRound53_XAIReusesOpenAIMapper pins the architectural decision that
// xAI (Grok) is OpenAI-compatible and reuses mapOpenAIFinishReasonToErr
// verbatim. If xAI diverges, this test MUST be replaced with a
// provider-specific mapper test in the same commit.
func TestRound53_XAIReusesOpenAIMapper(t *testing.T) {
	assert.True(t, errors.Is(mapOpenAIFinishReasonToErr("length"), ErrResponseTruncated))
	assert.True(t, errors.Is(mapOpenAIFinishReasonToErr("content_filter"), ErrResponseContentBlocked))
	assert.Nil(t, mapOpenAIFinishReasonToErr("stop"))
	assert.Nil(t, mapOpenAIFinishReasonToErr("tool_calls"))
}

// TestRound53_OpenRouterReusesOpenAIMapper pins the architectural
// decision that OpenRouter normalises every backend's finish_reason to
// OpenAI-compatible values and reuses mapOpenAIFinishReasonToErr. If
// OpenRouter passes through a backend's custom finish_reason in the
// future, this test MUST be replaced with a provider-specific mapper
// test in the same commit.
func TestRound53_OpenRouterReusesOpenAIMapper(t *testing.T) {
	assert.True(t, errors.Is(mapOpenAIFinishReasonToErr("length"), ErrResponseTruncated))
	assert.True(t, errors.Is(mapOpenAIFinishReasonToErr("content_filter"), ErrResponseContentBlocked))
	assert.Nil(t, mapOpenAIFinishReasonToErr("stop"))
}

// TestRound53_OpenAICompatibleReusesOpenAIMapper pins the architectural
// decision that all 11 OpenAI-compatible local backends (VLLM, LMStudio,
// Jan, LocalAI, FastChat, TextGen, KoboldAI, GPT4All, TabbyAPI, MLX,
// MistralRS) reuse mapOpenAIFinishReasonToErr. If any backend diverges
// in the future, this test MUST be replaced with a backend-specific
// path in the same commit.
func TestRound53_OpenAICompatibleReusesOpenAIMapper(t *testing.T) {
	assert.True(t, errors.Is(mapOpenAIFinishReasonToErr("length"), ErrResponseTruncated))
	assert.True(t, errors.Is(mapOpenAIFinishReasonToErr("content_filter"), ErrResponseContentBlocked))
	assert.Nil(t, mapOpenAIFinishReasonToErr("stop"))
	assert.Nil(t, mapOpenAIFinishReasonToErr("tool_calls"))
}

// TestRound53_ProvidersWired is a quick smoke that all 4 round-53
// providers actually wire LLMResponse.Err (catches silent no-op
// regressions where a refactor strips the Err assignment). Each helper
// MUST return a non-nil sentinel for at least one input.
func TestRound53_ProvidersWired(t *testing.T) {
	// Round-46 OpenAI mapper reused by xAI / OpenRouter / OpenAICompatible
	require.NotNil(t, mapOpenAIFinishReasonToErr("length"),
		"OpenAI mapper (reused by xAI/OpenRouter/OpenAICompatible) MUST recognise 'length'")
	require.NotNil(t, mapOpenAIFinishReasonToErr("content_filter"),
		"OpenAI mapper MUST recognise 'content_filter'")

	// Round-53-new Llama.cpp legacy stop-flag mapper
	_, sentinel := mapLlamaCppStopFlagsToErr(false, true, false)
	require.NotNil(t, sentinel,
		"mapLlamaCppStopFlagsToErr MUST return non-nil for stopped_limit=true")
}

// =========================================================================
// Test helpers
// =========================================================================

// mustSplitHostPort splits an httptest.Server URL into a host (with
// scheme) and a port. NewLlamaCPPProvider's Generate method composes
// the URL as `serverHost:serverPort/...`, so we feed it ServerHost
// "http://127.0.0.1" and ServerPort 12345 separately.
//
// IPv6 correctness (regression guard: TestMustSplitHostPort_BracketsIPv6).
// httptest.NewServer binds 127.0.0.1:0 and, when that bind fails (e.g. the
// host's IPv4 ephemeral range is exhausted), falls back to tcp6 "[::1]:0"
// (see net/http/httptest.newLocalListener). server.URL is then
// "http://[::1]:44047". net.SplitHostPort STRIPS the brackets from an IPv6
// literal ("[::1]" -> "::1"), so re-joining with plain concatenation yields
// "http://::1", and the provider's `fmt.Sprintf("%s:%d", host, port)`
// composition produces "http://::1:44047" — which url.Parse rejects with
// `invalid port "::1:44047" after host`, failing the test for an
// environmental reason unrelated to the behaviour under test.
// net.JoinHostPort re-applies the brackets exactly when the host is an IPv6
// literal; the empty port argument leaves a trailing ":" that we trim, so
// the caller gets a host the provider can safely append ":<port>" to.
func mustSplitHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	host, portStr, err := net.SplitHostPort(u.Host)
	require.NoError(t, err)
	port := 0
	_, scanErr := fmt.Sscanf(portStr, "%d", &port)
	require.NoError(t, scanErr)
	hostForURL := strings.TrimSuffix(net.JoinHostPort(host, ""), ":")
	return u.Scheme + "://" + hostForURL, port
}

// TestMustSplitHostPort_BracketsIPv6 is the permanent regression guard for
// the IPv6-bracket defect that made the round-53 Llama.cpp tests fail in the
// 2026-07-27 full-retest sweep with:
//
//	failed to create request: parse "http://::1:44047/v1/completions": invalid port "::1:44047" after host
//
// It is deterministic and needs no network: it drives the exact chain the
// provider drives — mustSplitHostPort -> fmt.Sprintf("%s:%d", host, port) ->
// http.NewRequestWithContext — for both an IPv4 and an IPv6 httptest-shaped
// URL, and asserts the composed URL is parseable and round-trips to the
// original authority. Reverting mustSplitHostPort to plain concatenation
// makes the IPv6 case FAIL here.
func TestMustSplitHostPort_BracketsIPv6(t *testing.T) {
	cases := []struct {
		name          string
		serverURL     string
		wantHost      string
		wantAuthority string
	}{
		{"ipv4", "http://127.0.0.1:44047", "http://127.0.0.1", "127.0.0.1:44047"},
		{"ipv6_loopback", "http://[::1]:44047", "http://[::1]", "[::1]:44047"},
		{"ipv6_full", "http://[fe80::1]:8080", "http://[fe80::1]", "[fe80::1]:8080"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host, port := mustSplitHostPort(t, tc.serverURL)
			assert.Equal(t, tc.wantHost, host,
				"IPv6 literals MUST keep their brackets (net.JoinHostPort)")

			// Exact composition performed by LlamaCPPProvider.Generate and
			// GenerateStream.
			baseURL := fmt.Sprintf("%s:%d", host, port)
			req, err := http.NewRequestWithContext(
				context.Background(), "POST", baseURL+"/v1/completions", nil)
			require.NoError(t, err,
				"composed provider URL MUST be parseable; unbracketed IPv6 produces "+
					`invalid port "…" after host`)
			require.NotNil(t, req,
				"a nil *http.Request here is what panicked GenerateStream in the sweep")
			assert.Equal(t, tc.wantAuthority, req.URL.Host,
				"composed URL MUST round-trip to the original server authority")
		})
	}
}

// silence the unused-import linter when no bufio reference survives a
// future edit pass (bufio is used in llamacpp_provider.go for SSE
// scanning; this no-op keeps test compilation independent of edit
// churn in the round-53 wiring).
var _ = bufio.ScanLines
