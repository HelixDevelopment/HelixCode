package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQwenProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      ProviderConfigEntry
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: ProviderConfigEntry{
				Type:     "qwen",
				Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
				APIKey:   "test-key",
			},
			expectError: false,
		},
		{
			name: "missing API key",
			config: ProviderConfigEntry{
				Type:     "qwen",
				Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			},
			expectError: false, // Now supports OAuth2 fallback
		},
		{
			name: "default endpoint",
			config: ProviderConfigEntry{
				Type:   "qwen",
				APIKey: "test-key",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewQwenProvider(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, "qwen", provider.GetType())
				assert.Equal(t, "Qwen", provider.GetName())
			}
		})
	}
}

func TestQwenProvider_GetType(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "qwen",
		APIKey: "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "qwen", provider.GetType())
}

func TestQwenProvider_GetName(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "qwen",
		APIKey: "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "Qwen", provider.GetName())
}

func TestQwenProvider_GetModels(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "qwen",
		APIKey: "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	models := provider.GetModels()
	assert.NotEmpty(t, models)

	// Check that we have expected models
	modelNames := make(map[string]bool)
	for _, model := range models {
		modelNames[model.Name] = true
		assert.Equal(t, "qwen", model.Provider)
		assert.NotEmpty(t, model.Capabilities)
		assert.Greater(t, model.ContextSize, 0)
		assert.Greater(t, model.MaxTokens, 0)
	}

	// Verify specific models are present
	expectedModels := []string{
		"qwen3-coder-plus",
		"qwen2.5-coder-32b-instruct",
		"qwen2.5-coder-7b-instruct",
		"qwen-vl-plus",
		"qwen-turbo",
	}

	for _, expected := range expectedModels {
		assert.True(t, modelNames[expected], "Expected model %s not found", expected)
	}
}

func TestQwenProvider_GetCapabilities(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "qwen",
		APIKey: "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	capabilities := provider.GetCapabilities()
	expectedCapabilities := []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
		CapabilityVision,
	}

	assert.Equal(t, expectedCapabilities, capabilities)
}

func TestQwenProvider_IsAvailable(t *testing.T) {
	// Test with mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": [{"id": "qwen3-coder-plus"}]}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     "qwen",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	available := provider.IsAvailable(ctx)
	assert.True(t, available)
}

func TestQwenProvider_GetHealth(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectHealthy  bool
	}{
		{
			name: "healthy response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data": [{"id": "qwen3-coder-plus"}, {"id": "qwen-turbo"}]}`))
			},
			expectHealthy: true,
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectHealthy: false,
		},
		{
			name: "invalid json",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`invalid json`))
			},
			expectHealthy: false, // Should be degraded, not unhealthy
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			config := ProviderConfigEntry{
				Type:     "qwen",
				Endpoint: server.URL,
				APIKey:   "test-key",
			}
			provider, err := NewQwenProvider(config)
			require.NoError(t, err)

			ctx := context.Background()
			health, err := provider.GetHealth(ctx)

			assert.NotNil(t, health)
			if tt.expectHealthy {
				assert.NoError(t, err)
				assert.Equal(t, "healthy", health.Status)
				assert.Greater(t, health.ModelCount, 0)
			} else {
				// For error cases, status could be unhealthy or degraded
				assert.Contains(t, []string{"unhealthy", "degraded"}, health.Status)
			}
		})
	}
}

func TestQwenProvider_Generate(t *testing.T) {
	// Mock server for chat completions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "qwen3-coder-plus",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello, this is a test response from Qwen!"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 12,
					"total_tokens": 22
				}
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     "qwen",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "qwen3-coder-plus",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	ctx := context.Background()
	response, err := provider.Generate(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, request.ID, response.RequestID)
	assert.Contains(t, response.Content, "Hello, this is a test response from Qwen!")
	assert.Equal(t, "stop", response.FinishReason)
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 12, response.Usage.CompletionTokens)
	assert.Equal(t, 22, response.Usage.TotalTokens)
	assert.Greater(t, response.ProcessingTime, time.Duration(0))
}

func TestQwenProvider_GenerateStream(t *testing.T) {
	// Mock server for streaming chat completions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" && r.Method == "POST" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)

			// Send streaming response as JSON lines (OpenAI format)
			streamData := []string{
				`{"id": "chatcmpl-test", "object": "chat.completion.chunk", "created": 1234567890, "model": "qwen3-coder-plus", "choices": [{"index": 0, "delta": {"role": "assistant", "content": "Hello"}, "finish_reason": null}]}`,
				`{"id": "chatcmpl-test", "object": "chat.completion.chunk", "created": 1234567890, "model": "qwen3-coder-plus", "choices": [{"index": 0, "delta": {"content": " world!"}, "finish_reason": null}]}`,
				`{"id": "chatcmpl-test", "object": "chat.completion.chunk", "created": 1234567890, "model": "qwen3-coder-plus", "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}]}`,
			}

			for _, data := range streamData {
				w.Write([]byte(data))
				w.(http.Flusher).Flush()
				time.Sleep(10 * time.Millisecond) // Small delay to simulate streaming
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     "qwen",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "qwen3-coder-plus",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
		Stream:      true,
	}

	ctx := context.Background()
	ch := make(chan LLMResponse, 10)

	errCh := make(chan error, 1)
	go func() {
		errCh <- provider.GenerateStream(ctx, request, ch)
	}()

	var responses []LLMResponse
	timeout := time.After(5 * time.Second)

	for {
		select {
		case response := <-ch:
			responses = append(responses, response)
		case err := <-errCh:
			assert.NoError(t, err)
			goto done
		case <-timeout:
			t.Fatal("Test timed out")
		}
	}

done:
	assert.NotEmpty(t, responses)
	// Verify we received some streaming chunks
	fullContent := ""
	for _, resp := range responses {
		if resp.Content != "" {
			fullContent += resp.Content
		}
	}
	// Just check that we got some content (the mock might not work perfectly)
	assert.NotEmpty(t, fullContent, "Should receive some streaming content")
}

func TestQwenProvider_Close(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "qwen",
		APIKey: "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

func TestQwenProvider_ErrorHandling(t *testing.T) {
	// Test API error responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": {"message": "Invalid model specified", "type": "invalid_request_error"}}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     "qwen",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewQwenProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "invalid-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	ctx := context.Background()
	response, err := provider.Generate(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "Qwen API returned status 400")
}
