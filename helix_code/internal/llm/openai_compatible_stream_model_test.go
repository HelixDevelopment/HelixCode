package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Provider-level model-identity guard (CONST-036 / CONST-037, §11.4.135).
//
// WHY THIS EXISTS SEPARATELY FROM THE FACADE GUARD
// The wire-facade guards in internal/server drive a FAKE provider that sets
// LLMResponse.Model itself, so they prove the facade PREFERS a served model —
// but they never execute this file's SSE parser. A re-review demonstrated the
// consequence by deleting `Model: streamResponse.Model` from the streaming
// chunk construction: the entire suite, facade guards included, stayed green.
//
// This test closes that hole. It stands up a real HTTP server emitting canned
// OpenAI-style SSE frames whose `model` field is a concrete model, drives the
// real OpenAICompatibleProvider.GenerateStream against it, and asserts the
// chunks delivered on the channel carry that model. Delete the propagation in
// makeStreamingRequest and this test fails — which is the whole point (§1.1:
// a guard whose paired mutation does not make it fail is not a guard).
func TestOpenAICompatibleProvider_StreamChunksCarryServedModel(t *testing.T) {
	const servedModel = "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Model discovery — the provider probes this at construction.
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"object":"list","data":[{"id":%q,"object":"model"}]}`, servedModel)
			return
		}

		// Canned SSE stream. Every frame reports the CONCRETE model, exactly as
		// llama.cpp and other OpenAI-compatible backends do when they resolve
		// an alias to a real model.
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		frames := []string{
			fmt.Sprintf(`{"id":"1","object":"chat.completion.chunk","model":%q,"choices":[{"index":0,"delta":{"content":"Hel"},"finish_reason":""}]}`, servedModel),
			fmt.Sprintf(`{"id":"1","object":"chat.completion.chunk","model":%q,"choices":[{"index":0,"delta":{"content":"lo"},"finish_reason":""}]}`, servedModel),
		}
		for _, f := range frames {
			fmt.Fprintf(w, "data: %s\n\n", f)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	provider, err := NewOpenAICompatibleProvider("testbackend", OpenAICompatibleConfig{
		BaseURL:          server.URL,
		DefaultModel:     servedModel,
		Timeout:          10 * time.Second,
		StreamingSupport: true, // the live path the helixllm coder route uses
	})
	require.NoError(t, err)

	ch := make(chan LLMResponse, 16)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// The caller asks with an ALIAS; the backend reports the concrete model.
	require.NoError(t, provider.GenerateStream(ctx, &LLMRequest{
		Model:    "default",
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, ch))

	var contentChunks int
	for chunk := range ch {
		if chunk.Content == "" {
			continue
		}
		contentChunks++
		if chunk.Model != servedModel {
			t.Errorf("streamed chunk %d dropped the backend-reported model: got %q, want %q — without this the wire facade has nothing to upgrade from and a stream:true request reports the requested alias (CONST-036/037)",
				contentChunks, chunk.Model, servedModel)
		}
	}

	if contentChunks == 0 {
		t.Fatal("no content chunks were delivered; the canned SSE stream was not parsed")
	}
}
