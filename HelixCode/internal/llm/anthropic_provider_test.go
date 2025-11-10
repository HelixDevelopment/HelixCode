package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnthropicProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      ProviderConfigEntry
		envKey      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with API key",
			config: ProviderConfigEntry{
				Type:     "anthropic",
				Endpoint: "https://api.anthropic.com/v1/messages",
				APIKey:   "test-key",
			},
			expectError: false,
		},
		{
			name: "valid config with env API key",
			config: ProviderConfigEntry{
				Type:     "anthropic",
				Endpoint: "https://api.anthropic.com/v1/messages",
			},
			envKey:      "sk-ant-test-key",
			expectError: false,
		},
		{
			name: "missing API key",
			config: ProviderConfigEntry{
				Type:     "anthropic",
				Endpoint: "https://api.anthropic.com/v1/messages",
			},
			expectError: true,
			errorMsg:    "API key not provided",
		},
		{
			name: "default endpoint",
			config: ProviderConfigEntry{
				Type:   "anthropic",
				APIKey: "test-key",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if specified
			if tt.envKey != "" {
				os.Setenv("ANTHROPIC_API_KEY", tt.envKey)
				defer os.Unsetenv("ANTHROPIC_API_KEY")
			} else {
				os.Unsetenv("ANTHROPIC_API_KEY")
			}

			provider, err := NewAnthropicProvider(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, "anthropic", provider.GetType())
				assert.Equal(t, "Anthropic", provider.GetName())
			}
		})
	}
}

func TestAnthropicProvider_GetType(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "anthropic",
		APIKey: "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "anthropic", provider.GetType())
}

func TestAnthropicProvider_GetName(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "anthropic",
		APIKey: "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "Anthropic", provider.GetName())
}

func TestAnthropicProvider_GetModels(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "anthropic",
		APIKey: "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	models := provider.GetModels()
	assert.NotEmpty(t, models)

	// Check that we have expected models
	modelNames := make(map[string]bool)
	for _, model := range models {
		modelNames[model.Name] = true
		assert.Equal(t, "anthropic", model.Provider)
		assert.Greater(t, model.ContextSize, 0)
		assert.NotEmpty(t, model.Description)
	}

	// Verify key models exist
	assert.True(t, modelNames["claude-4-sonnet"], "Should have Claude 4 Sonnet")
	assert.True(t, modelNames["claude-3-5-sonnet-latest"], "Should have Claude 3.5 Sonnet Latest")
	assert.True(t, modelNames["claude-3-5-haiku-latest"], "Should have Claude 3.5 Haiku Latest")
	assert.True(t, modelNames["claude-3-opus-latest"], "Should have Claude 3 Opus Latest")
}

func TestAnthropicProvider_GetCapabilities(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "anthropic",
		APIKey: "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	caps := provider.GetCapabilities()
	assert.NotEmpty(t, caps)

	// Convert to map for easy checking
	capMap := make(map[ModelCapability]bool)
	for _, cap := range caps {
		capMap[cap] = true
	}

	// Verify all expected capabilities
	assert.True(t, capMap[CapabilityTextGeneration])
	assert.True(t, capMap[CapabilityCodeGeneration])
	assert.True(t, capMap[CapabilityCodeAnalysis])
	assert.True(t, capMap[CapabilityPlanning])
	assert.True(t, capMap[CapabilityVision])
}

func TestAnthropicProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name      string
		config    ProviderConfigEntry
		available bool
	}{
		{
			name: "available with API key",
			config: ProviderConfigEntry{
				Type:   "anthropic",
				APIKey: "test-key",
			},
			available: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAnthropicProvider(tt.config)
			require.NoError(t, err)

			available := provider.IsAvailable(context.Background())
			assert.Equal(t, tt.available, available)
		})
	}
}

func TestAnthropicProvider_Generate(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		// Verify request body
		var req anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, "claude-3-5-sonnet-latest", req.Model)
		assert.Equal(t, 1000, req.MaxTokens)

		// Return mock response
		response := anthropicResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{
					Type: "text",
					Text: "Hello! This is a test response.",
				},
			},
			Model:      "claude-3-5-sonnet-latest",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with mock endpoint
	config := ProviderConfigEntry{
		Type:     "anthropic",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	// Test generation
	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "claude-3-5-sonnet-latest",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Hello! This is a test response.", response.Content)
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 20, response.Usage.CompletionTokens)
	assert.Equal(t, 30, response.Usage.TotalTokens)
	assert.Equal(t, "end_turn", response.FinishReason)
}

func TestAnthropicProvider_GenerateWithTools(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify tools are present and cached
		assert.NotEmpty(t, req.Tools)
		assert.Equal(t, "get_weather", req.Tools[0].Name)
		// Last tool should have cache control
		assert.NotNil(t, req.Tools[len(req.Tools)-1].CacheControl)
		assert.Equal(t, "ephemeral", req.Tools[len(req.Tools)-1].CacheControl.Type)

		// Return tool use response
		response := anthropicResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{
					Type: "tool_use",
					ID:   "toolu_123",
					Name: "get_weather",
					Input: map[string]interface{}{
						"location": "San Francisco",
					},
				},
			},
			Model:      "claude-3-5-sonnet-latest",
			StopReason: "tool_use",
			Usage: anthropicUsage{
				InputTokens:         50,
				OutputTokens:        30,
				CacheCreationTokens: 100,
				CacheReadTokens:     0,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     "anthropic",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "claude-3-5-sonnet-latest",
		Messages: []Message{
			{Role: "user", Content: "What's the weather in San Francisco?"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
		Tools: []Tool{
			{
				Type: "function",
				Function: ToolFunction{
					Name:        "get_weather",
					Description: "Get the current weather",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "City name",
							},
						},
					},
				},
			},
		},
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.ToolCalls)
	assert.Equal(t, "toolu_123", response.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", response.ToolCalls[0].Function.Name)

	// Verify caching metadata
	metadata := response.ProviderMetadata
	assert.Equal(t, 100, metadata["cache_creation_tokens"])
	assert.Equal(t, 0, metadata["cache_read_tokens"])
}

func TestAnthropicProvider_ExtendedThinking(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify extended thinking is enabled
		assert.NotNil(t, req.Thinking)
		assert.Equal(t, "enabled", req.Thinking.Type)
		assert.Greater(t, req.Thinking.Budget, 0)

		response := anthropicResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{
					Type: "text",
					Text: "After careful consideration, the answer is...",
				},
			},
			Model:      "claude-3-5-sonnet-latest",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  50,
				OutputTokens: 100,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     "anthropic",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	// Request with thinking keywords
	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "claude-3-5-sonnet-latest",
		Messages: []Message{
			{Role: "user", Content: "Think carefully and explain why 2+2=4"},
		},
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Content, "After careful consideration")
}

func TestAnthropicProvider_PromptCaching(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify system message has cache control
		if systemBlocks, ok := req.System.([]anthropicSystemBlock); ok {
			assert.NotEmpty(t, systemBlocks)
			assert.NotNil(t, systemBlocks[0].CacheControl)
			assert.Equal(t, "ephemeral", systemBlocks[0].CacheControl.Type)
		}

		// Verify last message has cache control
		if len(req.Messages) > 1 {
			lastMsg := req.Messages[len(req.Messages)-1]
			if content, ok := lastMsg.Content.([]anthropicContentBlock); ok {
				assert.NotNil(t, content[len(content)-1].CacheControl)
			}
		}

		response := anthropicResponse{
			ID:         "msg_test",
			Type:       "message",
			Role:       "assistant",
			Content:    []anthropicContentBlock{{Type: "text", Text: "Response"}},
			Model:      "claude-3-5-sonnet-latest",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:         100,
				OutputTokens:        50,
				CacheCreationTokens: 200,
				CacheReadTokens:     300,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     "anthropic",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "claude-3-5-sonnet-latest",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "First message"},
			{Role: "assistant", Content: "Response"},
			{Role: "user", Content: "Second message"},
		},
		MaxTokens: 1000,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)

	// Verify cache metadata
	metadata := response.ProviderMetadata
	assert.Equal(t, 200, metadata["cache_creation_tokens"])
	assert.Equal(t, 300, metadata["cache_read_tokens"])
}

func TestAnthropicProvider_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:       "API error 400",
			statusCode: 400,
			responseBody: `{
				"type": "error",
				"message": "Invalid request: model not found"
			}`,
			expectedError: "Invalid request",
		},
		{
			name:       "API error 401",
			statusCode: 401,
			responseBody: `{
				"type": "authentication_error",
				"message": "Invalid API key"
			}`,
			expectedError: "Invalid API key",
		},
		{
			name:       "API error 429",
			statusCode: 429,
			responseBody: `{
				"type": "rate_limit_error",
				"message": "Rate limit exceeded"
			}`,
			expectedError: "Rate limit exceeded",
		},
		{
			name:       "API error 500",
			statusCode: 500,
			responseBody: `{
				"type": "server_error",
				"message": "Internal server error"
			}`,
			expectedError: "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := ProviderConfigEntry{
				Type:     "anthropic",
				Endpoint: server.URL,
				APIKey:   "test-key",
			}
			provider, err := NewAnthropicProvider(config)
			require.NoError(t, err)

			request := &LLMRequest{
				ID:    uuid.New(),
				Model: "claude-3-5-sonnet-latest",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens: 1000,
			}

			_, err = provider.Generate(context.Background(), request)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestAnthropicProvider_GetHealth(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := anthropicResponse{
			ID:   "msg_health",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContentBlock{
				{Type: "text", Text: "OK"},
			},
			Model:      "claude-3-5-haiku-latest",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  5,
				OutputTokens: 5,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     "anthropic",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	health, err := provider.GetHealth(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
	assert.Greater(t, health.Latency, time.Duration(0))
	assert.Greater(t, health.ModelCount, 0)
}

func TestAnthropicProvider_Close(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "anthropic",
		APIKey: "test-key",
	}
	provider, err := NewAnthropicProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}
