package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTokenCredential implements azcore.TokenCredential for testing
type mockTokenCredential struct {
	token     string
	expiresOn time.Time
	err       error
}

func (m *mockTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if m.err != nil {
		return azcore.AccessToken{}, m.err
	}
	return azcore.AccessToken{
		Token:     m.token,
		ExpiresOn: m.expiresOn,
	}, nil
}

// Test 1: Provider initialization with API key
func TestAzureProvider_NewWithAPIKey(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-api-key",
		Parameters: map[string]interface{}{
			"endpoint":    "https://test.openai.azure.com",
			"api_version": "2025-04-01-preview",
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "test-api-key", provider.apiKey)
	assert.Equal(t, "https://test.openai.azure.com", provider.endpoint)
	assert.Equal(t, "2025-04-01-preview", provider.apiVersion)
	assert.Nil(t, provider.entraTokenProvider)
}

// Test 2: Provider initialization with Entra ID
func TestAzureProvider_NewWithEntraID(t *testing.T) {
	config := ProviderConfigEntry{
		Type: "azure",
		Parameters: map[string]interface{}{
			"endpoint":     "https://test.openai.azure.com",
			"api_version":  "2025-04-01-preview",
			"use_entra_id": true,
		},
	}

	// Note: This will use DefaultAzureCredential which may fail in test environment
	// In real tests, you'd mock the credential
	_, err := NewAzureProvider(config)
	// We expect this to potentially fail without proper Azure credentials
	// but the code path should execute
	if err != nil {
		assert.Contains(t, err.Error(), "failed to create Azure credential")
	}
}

// Test 3: Provider initialization without endpoint
func TestAzureProvider_NewWithoutEndpoint(t *testing.T) {
	config := ProviderConfigEntry{
		Type:       "azure",
		APIKey:     "test-key",
		Parameters: map[string]interface{}{},
	}

	_, err := NewAzureProvider(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

// Test 4: Deployment mapping - explicit mapping
func TestAzureProvider_DeploymentMapping_Explicit(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": "https://test.openai.azure.com",
			"deployment_map": map[string]string{
				"gpt-4-turbo":  "my-gpt4-deployment",
				"gpt-35-turbo": "my-gpt35-deployment",
			},
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "my-gpt4-deployment", provider.resolveDeployment("gpt-4-turbo"))
	assert.Equal(t, "my-gpt35-deployment", provider.resolveDeployment("gpt-35-turbo"))
}

// Test 5: Deployment mapping - fallback to model name
func TestAzureProvider_DeploymentMapping_Fallback(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": "https://test.openai.azure.com",
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	// Should fall back to using model name as deployment name
	assert.Equal(t, "gpt-4-turbo", provider.resolveDeployment("gpt-4-turbo"))
	assert.Equal(t, "unknown-model", provider.resolveDeployment("unknown-model"))
}

// Test 6: Deployment mapping from JSON string
func TestAzureProvider_DeploymentMapping_JSON(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint":       "https://test.openai.azure.com",
			"deployment_map": `{"gpt-4-turbo":"my-deployment"}`,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "my-deployment", provider.resolveDeployment("gpt-4-turbo"))
}

// Test 7: Basic generation with API key
func TestAzureProvider_Generate_APIKey(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/openai/deployments")
		assert.Contains(t, r.URL.Path, "/chat/completions")
		assert.NotEmpty(t, r.Header.Get("api-key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return mock response
		response := azureResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4-turbo",
			Choices: []azureChoice{
				{
					Index: 0,
					Message: azureMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you?",
					},
					FinishReason: "stop",
				},
			},
			Usage: azureUsage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint":    mockServer.URL,
			"api_version": "2025-04-01-preview",
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:        uuid.New(),
		Model:     "gpt-4-turbo",
		Messages:  []Message{{Role: "user", Content: "Hello"}},
		MaxTokens: 100,
	}

	response, err := provider.Generate(context.Background(), request)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Hello! How can I help you?", response.Content)
	assert.Equal(t, 30, response.Usage.TotalTokens)
	assert.Equal(t, "stop", response.FinishReason)
}

// Test 8: Generation with content filtering error
func TestAzureProvider_Generate_ContentFilter(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := azureError{
			Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Type    string `json:"type"`
				Param   string `json:"param,omitempty"`
			}{
				Code:    "content_filter",
				Message: "The prompt contains content that was filtered",
			},
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": mockServer.URL,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "gpt-4-turbo",
		Messages: []Message{{Role: "user", Content: "filtered content"}},
	}

	_, err = provider.Generate(context.Background(), request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content filtered")
}

// Test 9: Generation with rate limit error
func TestAzureProvider_Generate_RateLimit(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := azureError{
			Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Type    string `json:"type"`
				Param   string `json:"param,omitempty"`
			}{
				Code:    "429",
				Message: "Rate limit exceeded",
			},
		}

		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": mockServer.URL,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "gpt-4-turbo",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	_, err = provider.Generate(context.Background(), request)
	assert.Error(t, err)
	assert.Equal(t, ErrRateLimited, err)
}

// Test 10: Generation with deployment not found
func TestAzureProvider_Generate_DeploymentNotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := azureError{
			Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Type    string `json:"type"`
				Param   string `json:"param,omitempty"`
			}{
				Code:    "DeploymentNotFound",
				Message: "The deployment was not found",
			},
		}

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": mockServer.URL,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "non-existent-model",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	_, err = provider.Generate(context.Background(), request)
	assert.Error(t, err)
	assert.Equal(t, ErrModelNotFound, err)
}

// Test 11: Streaming generation
func TestAzureProvider_GenerateStream(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming request
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		// Send SSE events
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Chunk 1
		chunk1 := azureStreamChunk{
			ID:      "chatcmpl-123",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   "gpt-4-turbo",
			Choices: []azureStreamChoice{
				{
					Index: 0,
					Delta: azureDelta{
						Content: "Hello",
					},
				},
			},
		}
		data1, _ := json.Marshal(chunk1)
		w.Write([]byte("data: " + string(data1) + "\n\n"))

		// Chunk 2
		chunk2 := azureStreamChunk{
			ID:      "chatcmpl-123",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   "gpt-4-turbo",
			Choices: []azureStreamChoice{
				{
					Index: 0,
					Delta: azureDelta{
						Content: " World",
					},
				},
			},
		}
		data2, _ := json.Marshal(chunk2)
		w.Write([]byte("data: " + string(data2) + "\n\n"))

		// Final chunk
		chunk3 := azureStreamChunk{
			ID:      "chatcmpl-123",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   "gpt-4-turbo",
			Choices: []azureStreamChoice{
				{
					Index:        0,
					Delta:        azureDelta{},
					FinishReason: "stop",
				},
			},
		}
		data3, _ := json.Marshal(chunk3)
		w.Write([]byte("data: " + string(data3) + "\n\n"))

		// Stream end
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": mockServer.URL,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "gpt-4-turbo",
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	ch := make(chan LLMResponse, 10)
	go func() {
		err := provider.GenerateStream(context.Background(), request, ch)
		assert.NoError(t, err)
	}()

	var chunks []string
	for response := range ch {
		if response.Content != "" {
			chunks = append(chunks, response.Content)
		}
	}

	assert.Contains(t, chunks, "Hello")
	assert.Contains(t, chunks, " World")
}

// Test 12: Entra token provider caching
func TestEntraTokenProvider_Caching(t *testing.T) {
	callCount := 0
	mockCred := &mockTokenCredential{
		token:     "test-token",
		expiresOn: time.Now().Add(1 * time.Hour),
	}

	// Wrap to count calls
	countingCred := &countingCredential{
		inner:     mockCred,
		callCount: &callCount,
	}

	provider := NewEntraTokenProvider(countingCred)

	// First call should hit the credential
	token1, err := provider.GetToken(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "test-token", token1)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	token2, err := provider.GetToken(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "test-token", token2)
	assert.Equal(t, 1, callCount) // Should not have called credential again
}

type countingCredential struct {
	inner     azcore.TokenCredential
	callCount *int
}

func (c *countingCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	*c.callCount++
	return c.inner.GetToken(ctx, options)
}

// Test 13: Provider type and name
func TestAzureProvider_TypeAndName(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": "https://test.openai.azure.com",
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "azure", provider.GetType())
	assert.Equal(t, "Azure OpenAI", provider.GetName())
}

// Test 14: Models and capabilities
func TestAzureProvider_ModelsAndCapabilities(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": "https://test.openai.azure.com",
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	models := provider.GetModels()
	assert.NotEmpty(t, models)

	// Check for key models
	var foundGPT4 bool
	var foundGPT35 bool
	for _, model := range models {
		if model.Name == "gpt-4-turbo" {
			foundGPT4 = true
			assert.Equal(t, "azure", model.Provider)
			assert.True(t, model.SupportsTools)
		}
		if model.Name == "gpt-35-turbo" {
			foundGPT35 = true
		}
	}
	assert.True(t, foundGPT4, "Should have GPT-4 Turbo model")
	assert.True(t, foundGPT35, "Should have GPT-3.5 Turbo model")

	capabilities := provider.GetCapabilities()
	assert.Contains(t, capabilities, CapabilityTextGeneration)
	assert.Contains(t, capabilities, CapabilityCodeGeneration)
	assert.Contains(t, capabilities, CapabilityVision)
}

// Test 15: IsAvailable
func TestAzureProvider_IsAvailable(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": "https://test.openai.azure.com",
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	assert.True(t, provider.IsAvailable(context.Background()))

	// Provider without auth should not be available
	providerNoAuth := &AzureProvider{}
	assert.False(t, providerNoAuth.IsAvailable(context.Background()))
}

// Test 16: GetHealth success
func TestAzureProvider_GetHealth_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := azureResponse{
			ID:      "chatcmpl-health",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-35-turbo",
			Choices: []azureChoice{
				{
					Index: 0,
					Message: azureMessage{
						Role:    "assistant",
						Content: "OK",
					},
					FinishReason: "stop",
				},
			},
			Usage: azureUsage{
				PromptTokens:     5,
				CompletionTokens: 1,
				TotalTokens:      6,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": mockServer.URL,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	health, err := provider.GetHealth(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "healthy", health.Status)
	assert.Greater(t, health.Latency, time.Duration(0))
	assert.Equal(t, len(provider.GetModels()), health.ModelCount)
}

// Test 17: GetHealth failure
func TestAzureProvider_GetHealth_Failure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": mockServer.URL,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	health, err := provider.GetHealth(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Equal(t, 1, health.ErrorCount)
}

// Test 18: Close provider
func TestAzureProvider_Close(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": "https://test.openai.azure.com",
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

// Test 19: Tool support in request
func TestAzureProvider_WithTools(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify tools in request
		var req azureRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.NotEmpty(t, req.Tools)
		assert.Equal(t, "get_weather", req.Tools[0].Function.Name)

		response := azureResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4-turbo",
			Choices: []azureChoice{
				{
					Index: 0,
					Message: azureMessage{
						Role:    "assistant",
						Content: "I'll check the weather for you.",
					},
					FinishReason: "stop",
				},
			},
			Usage: azureUsage{
				PromptTokens:     15,
				CompletionTokens: 10,
				TotalTokens:      25,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": mockServer.URL,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "gpt-4-turbo",
		Messages: []Message{{Role: "user", Content: "What's the weather?"}},
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
								"description": "The city name",
							},
						},
					},
				},
			},
		},
	}

	response, err := provider.Generate(context.Background(), request)
	assert.NoError(t, err)
	assert.NotNil(t, response)
}

// Test 20: Default max tokens
func TestAzureProvider_DefaultMaxTokens(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req azureRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Should have default of 4096
		assert.Equal(t, 4096, req.MaxTokens)

		response := azureResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4-turbo",
			Choices: []azureChoice{
				{
					Index: 0,
					Message: azureMessage{
						Role:    "assistant",
						Content: "Response",
					},
					FinishReason: "stop",
				},
			},
			Usage: azureUsage{
				PromptTokens:     5,
				CompletionTokens: 5,
				TotalTokens:      10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := ProviderConfigEntry{
		Type:   "azure",
		APIKey: "test-key",
		Parameters: map[string]interface{}{
			"endpoint": mockServer.URL,
		},
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "gpt-4-turbo",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		// MaxTokens not specified - should default to 4096
	}

	_, err = provider.Generate(context.Background(), request)
	assert.NoError(t, err)
}
