//go:build e2e

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2EAllLocalProviders tests all local providers end-to-end with mock servers
func TestE2EAllLocalProviders(t *testing.T) {
	providers := map[string]func(t *testing.T) Provider{
		"VLLM":       testE2EVLLMProvider,
		"LocalAI":     testE2ELocalAIProvider,
		"FastChat":    testE2EFastChatProvider,
		"TextGen":     testE2ETextGenProvider,
		"LM Studio":   testE2ELMStudioProvider,
		"Jan AI":      testE2EJanProvider,
		"KoboldAI":    testE2EKoboldAIProvider,
		"GPT4All":     testE2EGPT4AllProvider,
		"TabbyAPI":    testE2ETabbyAPIProvider,
		"MLX":         testE2EMLXProvider,
		"MistralRS":   testE2EMistralRSProvider,
	}

	for name, testFunc := range providers {
		t.Run(name, func(t *testing.T) {
			provider := testFunc(t)
			defer provider.Close()
			testE2EProviderFunctionality(t, provider, name)
		})
	}
}

// testE2EProviderFunctionality tests core provider functionality
func testE2EProviderFunctionality(t *testing.T, provider Provider, providerName string) {
	ctx := context.Background()

	// Test provider availability
	require.True(t, provider.IsAvailable(ctx), "%s provider should be available", providerName)

	// Test model listing
	models := provider.GetModels()
	assert.NotEmpty(t, models, "%s should return at least one model", providerName)

	// Test capabilities
	capabilities := provider.GetCapabilities()
	assert.NotEmpty(t, capabilities, "%s should have capabilities", providerName)

	// Test health check
	health, err := provider.GetHealth(ctx)
	require.NoError(t, err, "%s health check should not fail", providerName)
	assert.Equal(t, "healthy", health.Status, "%s should be healthy", providerName)

	// Test generation
	testE2EGeneration(t, provider, providerName)

	// Test streaming
	testE2EStreaming(t, provider, providerName)
}

// testE2EGeneration tests non-streaming generation
func testE2EGeneration(t *testing.T, provider Provider, providerName string) {
	ctx := context.Background()
	models := provider.GetModels()
	if len(models) == 0 {
		t.Skip("No models available for generation test")
	}

	request := &LLMRequest{
		ID:           uuid.New(),
		ProviderType: provider.GetType(),
		Model:        models[0].Name,
		Messages: []Message{
			{Role: "user", Content: "Hello! Please respond with just 'Hello World'."},
		},
		MaxTokens:   50,
		Temperature: 0.1,
		Stream:      false,
		CreatedAt:   time.Now(),
	}

	response, err := provider.Generate(ctx, request)
	require.NoError(t, err, "%s generation should not fail", providerName)
	require.NotNil(t, response, "%s should return a response", providerName)

	assert.NotEmpty(t, response.Content, "%s response should have content", providerName)
	assert.True(t, response.ProcessingTime > 0, "%s should track processing time", providerName)
	assert.True(t, response.Usage.TotalTokens > 0, "%s should report token usage", providerName)
}

// testE2EStreaming tests streaming generation
func testE2EStreaming(t *testing.T, provider Provider, providerName string) {
	ctx := context.Background()
	models := provider.GetModels()
	if len(models) == 0 {
		t.Skip("No models available for streaming test")
	}

	request := &LLMRequest{
		ID:           uuid.New(),
		ProviderType: provider.GetType(),
		Model:        models[0].Name,
		Messages: []Message{
			{Role: "user", Content: "Count from 1 to 5, one number per response."},
		},
		MaxTokens:   100,
		Temperature: 0.1,
		Stream:      true,
		CreatedAt:   time.Now(),
	}

	ch := make(chan LLMResponse, 10)
	err := provider.GenerateStream(ctx, request, ch)
	require.NoError(t, err, "%s streaming should not fail", providerName)

	// Collect streaming responses
	responseCount := 0
	totalContent := ""
	for response := range ch {
		responseCount++
		totalContent += response.Content
		assert.NotEmpty(t, response.ID, "%s streaming response should have ID", providerName)
	}

	assert.Greater(t, responseCount, 0, "%s should send at least one streaming response", providerName)
	assert.NotEmpty(t, totalContent, "%s streaming should accumulate content", providerName)
}

// Mock server setup functions for each provider type

func testE2EVLLMProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "llama-2-7b-chat-hf",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "vllm",
					},
				},
			})
		case "/v1/chat/completions":
			if r.Header.Get("Accept") == "text/event-stream" {
				// Streaming response
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				fmt.Fprintf(w, "data: %s\n", `{"id":"chat-1","object":"chat.completion.chunk","created":1234567890,"model":"llama-2-7b-chat-hf","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"}}]}`)
				fmt.Fprintf(w, "data: %s\n", `{"id":"chat-1","object":"chat.completion.chunk","created":1234567890,"model":"llama-2-7b-chat-hf","choices":[{"index":0,"delta":{"content":" World"}}]}`)
				fmt.Fprintf(w, "data: %s\n", `[DONE]`)
			} else {
				// Non-streaming response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id": "chat-1",
					"object": "chat.completion",
					"created": time.Now().Unix(),
					"model": "llama-2-7b-chat-hf",
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]interface{}{
								"role": "assistant",
								"content": "Hello World",
							},
							"finish_reason": "stop",
						},
					},
					"usage": map[string]interface{}{
						"prompt_tokens":     10,
						"completion_tokens": 5,
						"total_tokens":     15,
					},
				})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "llama-2-7b-chat-hf",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("vllm", config)
	require.NoError(t, err)
	return provider
}

func testE2ELocalAIProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "gpt-3.5-turbo",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "localai",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "gpt-3.5-turbo",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from LocalAI",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "gpt-3.5-turbo",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("localai", config)
	require.NoError(t, err)
	return provider
}

func testE2EFastChatProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "vicuna-13b-v1.5",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "fastchat",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "vicuna-13b-v1.5",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from FastChat",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "vicuna-13b-v1.5",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("fastchat", config)
	require.NoError(t, err)
	return provider
}

func testE2ETextGenProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "llama-2-7b-chat-hf",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "textgen",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "llama-2-7b-chat-hf",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from TextGen",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "llama-2-7b-chat-hf",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("textgen", config)
	require.NoError(t, err)
	return provider
}

func testE2ELMStudioProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "local-model",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "lmstudio",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "local-model",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from LM Studio",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "local-model",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("lmstudio", config)
	require.NoError(t, err)
	return provider
}

func testE2EJanProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "jan-model",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "jan",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "jan-model",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from Jan AI",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "jan-model",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("jan", config)
	require.NoError(t, err)
	return provider
}

func testE2EKoboldAIProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/model":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": "kobold-model",
			})
		case "/api/v1/generate":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []map[string]interface{}{
					{
						"text": "Hello from KoboldAI",
						"generated": true,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := KoboldAIConfig{
		BaseURL:         server.URL,
		DefaultModel:    "kobold-model",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewKoboldAIProvider(config)
	require.NoError(t, err)
	return provider
}

func testE2EGPT4AllProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "gpt4all-model",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "gpt4all",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "gpt4all-model",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from GPT4All",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "gpt4all-model",
		Timeout:         30 * time.Second,
		StreamingSupport: false, // GPT4All might not support streaming
	}

	provider, err := NewOpenAICompatibleProvider("gpt4all", config)
	require.NoError(t, err)
	return provider
}

func testE2ETabbyAPIProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "tabby-model",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "tabbyapi",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "tabby-model",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from TabbyAPI",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "tabby-model",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("tabbyapi", config)
	require.NoError(t, err)
	return provider
}

func testE2EMLXProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "mlx-model",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "mlx",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "mlx-model",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from MLX",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "mlx-model",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("mlx", config)
	require.NoError(t, err)
	return provider
}

func testE2EMistralRSProvider(t *testing.T) Provider {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":      "mistral-model",
						"object":  "model",
						"created": time.Now().Unix(),
						"owned_by": "mistralrs",
					},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"object": "chat.completion",
				"created": time.Now().Unix(),
				"model": "mistral-model",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"content": "Hello from MistralRS",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":     15,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	config := OpenAICompatibleConfig{
		BaseURL:         server.URL,
		DefaultModel:    "mistral-model",
		Timeout:         30 * time.Second,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("mistralrs", config)
	require.NoError(t, err)
	return provider
}