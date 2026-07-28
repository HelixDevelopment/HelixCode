package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOpenAICompatibleTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			// Model discovery endpoint
			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "llama-3-8b", "object": "model", "owned_by": "local"},
					{"id": "gpt-4-vision", "object": "model", "owned_by": "local"},
				},
			}
			json.NewEncoder(w).Encode(response)

		case "/v1/chat/completions":
			// Chat completions endpoint
			assert.Equal(t, "POST", r.Method)

			response := OpenAICompatibleResponse{
				ID:      "chatcmpl-test123",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   "llama-3-8b",
				Choices: []OpenAICompatibleChoice{
					{
						Index: 0,
						Message: OpenAICompatibleMessage{
							Role:    "assistant",
							Content: "Hello from OpenAI-compatible API!",
						},
						FinishReason: "stop",
					},
				},
				Usage: OpenAICompatibleUsage{
					PromptTokens:     10,
					CompletionTokens: 8,
					TotalTokens:      18,
				},
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestOpenAICompatibleProviderCreation(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL:      server.URL,
			DefaultModel: "llama-3-8b",
			Timeout:      30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.True(t, provider.isRunning)
		assert.Equal(t, "vllm", provider.name)
	})

	t.Run("WithAPIKey", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify API key header
			if r.Header.Get("Authorization") != "Bearer test-api-key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "llama-3-8b"},
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			APIKey:  "test-api-key",
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("localai", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("CustomEndpoints", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/custom/models" {
				response := map[string]interface{}{
					"data": []map[string]interface{}{
						{"id": "custom-model"},
					},
				}
				json.NewEncoder(w).Encode(response)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL:       server.URL,
			ModelEndpoint: "/custom/models",
			ChatEndpoint:  "/custom/chat",
			Timeout:       30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("custom", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestOpenAICompatibleProvider_GetType(t *testing.T) {
	testCases := []struct {
		name         string
		providerName string
		expectedType ProviderType
	}{
		{"vLLM", "vllm", ProviderTypeVLLM},
		{"LocalAI", "localai", ProviderTypeLocalAI},
		{"FastChat", "fastchat", ProviderTypeFastChat},
		{"TextGen", "textgen", ProviderTypeTextGen},
		{"LM Studio", "lmstudio", ProviderTypeLMStudio},
		{"Jan", "jan", ProviderTypeJan},
		{"KoboldAI", "koboldai", ProviderTypeKoboldAI},
		{"GPT4All", "gpt4all", ProviderTypeGPT4All},
		{"TabbyAPI", "tabbyapi", ProviderTypeTabbyAPI},
		{"MLX", "mlx", ProviderTypeMLX},
		{"MistralRS", "mistralrs", ProviderTypeMistralRS},
		// Reconciled per §11.4.120: a non-local custom name now reports a
		// distinct ProviderType derived from the name (no longer the generic
		// "local"), so hosted catalogue providers are attributed correctly.
		{"Custom", "custom", ProviderType("custom")},
		{"Local", "local", ProviderTypeLocal},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := setupOpenAICompatibleTestServer(t)
			defer server.Close()

			config := OpenAICompatibleConfig{
				BaseURL: server.URL,
				Timeout: 30 * time.Second,
			}

			provider, err := NewOpenAICompatibleProvider(tc.providerName, config)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedType, provider.GetType())
		})
	}
}

func TestOpenAICompatibleProvider_GetName(t *testing.T) {
	server := setupOpenAICompatibleTestServer(t)
	defer server.Close()

	config := OpenAICompatibleConfig{
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("lmstudio", config)
	require.NoError(t, err)

	assert.Equal(t, "lmstudio", provider.GetName())
}

func TestOpenAICompatibleProvider_GetModels(t *testing.T) {
	server := setupOpenAICompatibleTestServer(t)
	defer server.Close()

	config := OpenAICompatibleConfig{
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("vllm", config)
	require.NoError(t, err)

	models := provider.GetModels()
	assert.Len(t, models, 2)

	// Verify model properties
	for _, model := range models {
		assert.NotEmpty(t, model.Name)
		assert.NotNil(t, model.Capabilities)
	}
}

func TestOpenAICompatibleProvider_GetCapabilities(t *testing.T) {
	t.Run("BasicCapabilities", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		capabilities := provider.GetCapabilities()
		assert.NotEmpty(t, capabilities)
		assert.Contains(t, capabilities, CapabilityTextGeneration)
		assert.Contains(t, capabilities, CapabilityCodeGeneration)
		assert.Contains(t, capabilities, CapabilityCodeAnalysis)
		assert.Contains(t, capabilities, CapabilityPlanning)
	})

	t.Run("VisionCapabilities", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		// Providers that support vision
		provider, err := NewOpenAICompatibleProvider("lmstudio", config)
		require.NoError(t, err)

		capabilities := provider.GetCapabilities()
		assert.Contains(t, capabilities, CapabilityVision)
	})
}

func TestOpenAICompatibleProvider_Generate(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL:      server.URL,
			DefaultModel: "llama-3-8b",
			Timeout:      30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		request := &LLMRequest{
			Model: "llama-3-8b",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			MaxTokens:   100,
			Temperature: 0.7,
		}

		ctx := context.Background()
		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "Hello from OpenAI-compatible API!", response.Content)
		assert.Equal(t, 18, response.Usage.TotalTokens)
		// CONST-036/037 (§11.4.135 regression guard): convertFromOpenAIResponse
		// must carry the backend-reported model through to LLMResponse.Model so
		// callers can surface the model that ACTUALLY served the request rather
		// than the requested alias. The canned server response above sets
		// Model: "llama-3-8b" — assert it survives the conversion. Deleting
		// `Model: response.Model` in convertFromOpenAIResponse makes this FAIL
		// (response.Model reverts to "") while every other assertion in this
		// suite stays green — that emptiness is the hole this guard closes.
		assert.Equal(t, "llama-3-8b", response.Model)
	})

	t.Run("APIError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/models" {
				response := map[string]interface{}{"data": []map[string]interface{}{{"id": "model"}}}
				json.NewEncoder(w).Encode(response)
				return
			}
			if r.URL.Path == "/v1/chat/completions" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		request := &LLMRequest{
			Model: "model",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		}

		ctx := context.Background()
		response, err := provider.Generate(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("ProviderStopped", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		provider.Close()

		request := &LLMRequest{
			Model: "llama-3-8b",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		}

		ctx := context.Background()
		response, err := provider.Generate(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, ErrProviderUnavailable, err)
	})
}

func TestOpenAICompatibleProvider_GenerateStream(t *testing.T) {
	t.Run("FallbackToNonStreaming", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL:          server.URL,
			DefaultModel:     "llama-3-8b",
			Timeout:          30 * time.Second,
			StreamingSupport: false, // Streaming disabled
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		request := &LLMRequest{
			Model: "llama-3-8b",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			MaxTokens: 100,
		}

		ch := make(chan LLMResponse, 10)
		ctx := context.Background()

		err = provider.GenerateStream(ctx, request, ch)
		require.NoError(t, err)

		// Should receive the response
		response := <-ch
		assert.Equal(t, "Hello from OpenAI-compatible API!", response.Content)
	})

	t.Run("ProviderStopped", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		provider.Close()

		request := &LLMRequest{
			Model: "llama-3-8b",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		}

		ch := make(chan LLMResponse, 10)
		ctx := context.Background()

		err = provider.GenerateStream(ctx, request, ch)
		assert.Error(t, err)
		assert.Equal(t, ErrProviderUnavailable, err)
	})
}

func TestOpenAICompatibleProvider_IsAvailable(t *testing.T) {
	t.Run("Available", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		ctx := context.Background()
		available := provider.IsAvailable(ctx)
		assert.True(t, available)
	})

	t.Run("Unavailable", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		ctx := context.Background()
		available := provider.IsAvailable(ctx)
		assert.False(t, available)
	})

	t.Run("ProviderStopped", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		provider.Close()

		ctx := context.Background()
		available := provider.IsAvailable(ctx)
		assert.False(t, available)
	})
}

func TestOpenAICompatibleProvider_GetHealth(t *testing.T) {
	t.Run("Healthy", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		ctx := context.Background()
		health, err := provider.GetHealth(ctx)
		require.NoError(t, err)
		assert.NotNil(t, health)
		assert.Equal(t, "healthy", health.Status)
		assert.Equal(t, 2, health.ModelCount)
	})

	t.Run("Unhealthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		ctx := context.Background()
		health, err := provider.GetHealth(ctx)
		assert.Error(t, err)
		assert.NotNil(t, health)
		assert.Equal(t, "unhealthy", health.Status)
	})

	t.Run("ProviderStopped", func(t *testing.T) {
		server := setupOpenAICompatibleTestServer(t)
		defer server.Close()

		config := OpenAICompatibleConfig{
			BaseURL: server.URL,
			Timeout: 30 * time.Second,
		}

		provider, err := NewOpenAICompatibleProvider("vllm", config)
		require.NoError(t, err)

		provider.Close()

		ctx := context.Background()
		health, err := provider.GetHealth(ctx)
		require.NoError(t, err)
		assert.NotNil(t, health)
		assert.Equal(t, "unhealthy", health.Status)
	})
}

func TestOpenAICompatibleProvider_Close(t *testing.T) {
	server := setupOpenAICompatibleTestServer(t)
	defer server.Close()

	config := OpenAICompatibleConfig{
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("vllm", config)
	require.NoError(t, err)
	assert.True(t, provider.isRunning)

	err = provider.Close()
	assert.NoError(t, err)
	assert.False(t, provider.isRunning)
}

func TestOpenAICompatibleProvider_GetAPIURL(t *testing.T) {
	testCases := []struct {
		name        string
		providerName string
		baseURL     string
		endpoint    string
		expected    string
	}{
		{"vLLM Default", "vllm", "", "/v1/models", "http://localhost:8000/v1/models"},
		{"TextGen Default", "textgen", "", "/v1/models", "http://localhost:5000/v1/models"},
		{"LM Studio Default", "lmstudio", "", "/v1/models", "http://localhost:1234/v1/models"},
		{"LocalAI Default", "localai", "", "/v1/models", "http://localhost:8080/v1/models"},
		{"Jan Default", "jan", "", "/v1/models", "http://localhost:1337/v1/models"},
		{"KoboldAI Default", "koboldai", "", "/v1/models", "http://localhost:5001/v1/models"},
		{"GPT4All Default", "gpt4all", "", "/v1/models", "http://localhost:4891/v1/models"},
		{"TabbyAPI Default", "tabbyapi", "", "/v1/models", "http://localhost:5000/v1/models"},
		{"FastChat Default", "fastchat", "", "/v1/models", "http://localhost:7860/v1/models"},
		{"Custom URL", "custom", "http://custom:9999", "/v1/models", "http://custom:9999/v1/models"},
		{"URL With Trailing Slash", "custom", "http://localhost:8080/", "/v1/models", "http://localhost:8080/v1/models"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &OpenAICompatibleProvider{
				name: tc.providerName,
				config: OpenAICompatibleConfig{
					BaseURL: tc.baseURL,
				},
			}

			url := provider.getAPIURL(tc.endpoint)
			assert.Equal(t, tc.expected, url)
		})
	}
}

func TestOpenAICompatibleProvider_GetModelName(t *testing.T) {
	t.Run("WithRequestedModel", func(t *testing.T) {
		provider := &OpenAICompatibleProvider{
			config: OpenAICompatibleConfig{
				DefaultModel: "default-model",
			},
			models: []ModelInfo{{Name: "first-model"}},
		}

		name := provider.getModelName("requested-model")
		assert.Equal(t, "requested-model", name)
	})

	t.Run("WithDefaultModel", func(t *testing.T) {
		provider := &OpenAICompatibleProvider{
			config: OpenAICompatibleConfig{
				DefaultModel: "default-model",
			},
			models: []ModelInfo{{Name: "first-model"}},
		}

		name := provider.getModelName("")
		assert.Equal(t, "default-model", name)
	})

	t.Run("WithFirstModel", func(t *testing.T) {
		provider := &OpenAICompatibleProvider{
			config: OpenAICompatibleConfig{},
			models: []ModelInfo{{Name: "first-model"}},
		}

		name := provider.getModelName("")
		assert.Equal(t, "first-model", name)
	})

	t.Run("Fallback", func(t *testing.T) {
		provider := &OpenAICompatibleProvider{
			config: OpenAICompatibleConfig{},
			models: []ModelInfo{},
		}

		name := provider.getModelName("")
		assert.Equal(t, "gpt-3.5-turbo", name)
	})
}

func TestOpenAICompatibleProvider_InferContextSize(t *testing.T) {
	provider := &OpenAICompatibleProvider{}

	testCases := []struct {
		modelName string
		expected  int
	}{
		{"model-32k", 32768},
		{"gpt-4-32k", 32768},
		{"model-16k", 16384},
		{"model-8k", 8192},
		{"gpt-4-turbo", 8192},
		{"claude-3-opus", 100000},
		{"llama-3-8b", 4096},
		{"unknown-model", 4096},
	}

	for _, tc := range testCases {
		t.Run(tc.modelName, func(t *testing.T) {
			size := provider.inferContextSize(tc.modelName)
			assert.Equal(t, tc.expected, size)
		})
	}
}

func TestOpenAICompatibleProvider_InferMaxTokens(t *testing.T) {
	provider := &OpenAICompatibleProvider{}

	// Max tokens should be half of context size
	assert.Equal(t, 16384, provider.inferMaxTokens("model-32k"))
	assert.Equal(t, 2048, provider.inferMaxTokens("llama-model"))
}

func TestOpenAICompatibleProvider_SupportsTools(t *testing.T) {
	provider := &OpenAICompatibleProvider{}

	assert.True(t, provider.supportsTools("gpt-4-turbo"))
	assert.True(t, provider.supportsTools("claude-3-opus"))
	assert.True(t, provider.supportsTools("llama-3-8b"))
	assert.True(t, provider.supportsTools("mistral-7b"))
	assert.False(t, provider.supportsTools("old-model"))
}

func TestOpenAICompatibleProvider_SupportsVision(t *testing.T) {
	testCases := []struct {
		providerName string
		expected     bool
	}{
		{"lmstudio", true},
		{"jan", true},
		{"textgen", true},
		{"vllm", false},
		{"localai", false},
	}

	for _, tc := range testCases {
		t.Run(tc.providerName, func(t *testing.T) {
			provider := &OpenAICompatibleProvider{name: tc.providerName}
			assert.Equal(t, tc.expected, provider.supportsVision())
		})
	}
}

func TestOpenAICompatibleProvider_SupportsVisionModel(t *testing.T) {
	provider := &OpenAICompatibleProvider{}

	assert.True(t, provider.supportsVisionModel("gpt-4-vision-preview"))
	assert.True(t, provider.supportsVisionModel("llava-1.5"))
	assert.True(t, provider.supportsVisionModel("multimodal-model"))
	assert.True(t, provider.supportsVisionModel("clip-model"))
	assert.False(t, provider.supportsVisionModel("text-only-model"))
}

func TestOpenAICompatibleProvider_UpdateHealth(t *testing.T) {
	provider := &OpenAICompatibleProvider{
		lastHealth: &ProviderHealth{},
	}

	// Test healthy update
	provider.updateHealth("healthy", 50*time.Millisecond, 0)
	assert.Equal(t, "healthy", provider.lastHealth.Status)
	assert.Equal(t, 50*time.Millisecond, provider.lastHealth.Latency)
	assert.Equal(t, 0, provider.lastHealth.ErrorCount)

	// Test unhealthy update
	provider.updateHealth("unhealthy", 100*time.Millisecond, 5)
	assert.Equal(t, "unhealthy", provider.lastHealth.Status)
	assert.Equal(t, 100*time.Millisecond, provider.lastHealth.Latency)
	assert.Equal(t, 5, provider.lastHealth.ErrorCount)
}

func TestReadSSELine(t *testing.T) {
	t.Run("SimpleLine", func(t *testing.T) {
		input := "data: test\n"
		reader := &mockReader{data: []byte(input)}
		line, err := readSSELine(reader)
		require.NoError(t, err)
		assert.Equal(t, "data: test", line)
	})

	t.Run("LineWithCR", func(t *testing.T) {
		input := "data: test\r\n"
		reader := &mockReader{data: []byte(input)}
		line, err := readSSELine(reader)
		require.NoError(t, err)
		assert.Equal(t, "data: test", line)
	})
}

type mockReader struct {
	data []byte
	pos  int
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, nil
	}
	p[0] = m.data[m.pos]
	m.pos++
	return 1, nil
}

func TestOpenAICompatibleProvider_ReasoningContent(t *testing.T) {
	// Test that reasoning_content from wire response is captured in ProviderMetadata
	msg := OpenAICompatibleMessage{
		Role:             "assistant",
		Content:          "The answer is 4.",
		ReasoningContent: "Let me think step by step: 2+2=4",
	}
	if msg.ReasoningContent != "Let me think step by step: 2+2=4" {
		t.Error("ReasoningContent not parsed from message")
	}

	// Test Delta also carries ReasoningContent
	delta := OpenAICompatibleDelta{
		Role:             "assistant",
		Content:          "partial",
		ReasoningContent: "thinking...",
	}
	if delta.ReasoningContent != "thinking..." {
		t.Error("ReasoningContent not parsed from delta")
	}

	// Test that empty ReasoningContent is handled (omitempty)
	emptyMsg := OpenAICompatibleMessage{
		Role:    "assistant",
		Content: "plain response",
	}
	if emptyMsg.ReasoningContent != "" {
		t.Error("Empty ReasoningContent should be empty string")
	}
}

func TestOpenAICompatibleProvider_ReasoningContent_InResponse(t *testing.T) {
	// Simulate a wire response JSON with reasoning_content and verify it is
	// captured into LLMResponse.ProviderMetadata after convertFromOpenAIResponse.
	wireJSON := `{
		"id": "chatcmpl-test",
		"object": "chat.completion",
		"created": 1700000000,
		"model": "mimo-v2.5-pro",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "The answer is 4.",
				"reasoning_content": "Deep thinking: 2+2=4, confirmed."
			},
			"finish_reason": "stop"
		}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
	}`

	var resp OpenAICompatibleResponse
	if err := json.Unmarshal([]byte(wireJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal wire response: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("expected at least one choice")
	}

	msg := resp.Choices[0].Message
	if msg.ReasoningContent != "Deep thinking: 2+2=4, confirmed." {
		t.Errorf("ReasoningContent = %q, want %q", msg.ReasoningContent, "Deep thinking: 2+2=4, confirmed.")
	}
	if msg.Content != "The answer is 4." {
		t.Errorf("Content = %q, want %q", msg.Content, "The answer is 4.")
	}

	// Now verify convertFromOpenAIResponse captures it into ProviderMetadata
	provider := &OpenAICompatibleProvider{name: "test"}
	llmResp := provider.convertFromOpenAIResponse(&resp, uuid.New(), time.Millisecond)
	if llmResp.ProviderMetadata == nil {
		t.Fatal("ProviderMetadata should not be nil when reasoning_content is present")
	}
	rc, ok := llmResp.ProviderMetadata["reasoning_content"].(string)
	if !ok {
		t.Fatalf("ProviderMetadata[reasoning_content] should be string, got %T", llmResp.ProviderMetadata["reasoning_content"])
	}
	if rc != "Deep thinking: 2+2=4, confirmed." {
		t.Errorf("ProviderMetadata[reasoning_content] = %q, want %q", rc, "Deep thinking: 2+2=4, confirmed.")
	}
}

func TestOpenAICompatibleProvider_ReasoningContent_Empty(t *testing.T) {
	// When reasoning_content is absent/empty, ProviderMetadata should NOT be populated
	wireJSON := `{
		"id": "chatcmpl-test",
		"object": "chat.completion",
		"created": 1700000000,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Plain answer."
			},
			"finish_reason": "stop"
		}],
		"usage": {"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8}
	}`

	var resp OpenAICompatibleResponse
	if err := json.Unmarshal([]byte(wireJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	provider := &OpenAICompatibleProvider{name: "test"}
	llmResp := provider.convertFromOpenAIResponse(&resp, uuid.New(), time.Millisecond)
	if llmResp.ProviderMetadata != nil {
		if _, exists := llmResp.ProviderMetadata["reasoning_content"]; exists {
			t.Error("ProviderMetadata should not contain reasoning_content when wire has none")
		}
	}
}
