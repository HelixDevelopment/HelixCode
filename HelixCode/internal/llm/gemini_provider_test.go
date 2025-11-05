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

func TestNewGeminiProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      ProviderConfigEntry
		envKey      string
		googleKey   string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with API key",
			config: ProviderConfigEntry{
				Type:     ProviderTypeGemini,
				Endpoint: "https://generativelanguage.googleapis.com/v1beta",
				APIKey:   "test-key",
			},
			expectError: false,
		},
		{
			name: "valid config with GEMINI_API_KEY env",
			config: ProviderConfigEntry{
				Type:     ProviderTypeGemini,
				Endpoint: "https://generativelanguage.googleapis.com/v1beta",
			},
			envKey:      "gemini-test-key",
			expectError: false,
		},
		{
			name: "valid config with GOOGLE_API_KEY env",
			config: ProviderConfigEntry{
				Type:     ProviderTypeGemini,
				Endpoint: "https://generativelanguage.googleapis.com/v1beta",
			},
			googleKey:   "google-test-key",
			expectError: false,
		},
		{
			name: "missing API key",
			config: ProviderConfigEntry{
				Type:     ProviderTypeGemini,
				Endpoint: "https://generativelanguage.googleapis.com/v1beta",
			},
			expectError: true,
			errorMsg:    "API key not provided",
		},
		{
			name: "default endpoint",
			config: ProviderConfigEntry{
				Type:   ProviderTypeGemini,
				APIKey: "test-key",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all API key env vars first
			os.Unsetenv("GEMINI_API_KEY")
			os.Unsetenv("GOOGLE_API_KEY")

			// Set environment variables if specified
			if tt.envKey != "" {
				os.Setenv("GEMINI_API_KEY", tt.envKey)
				defer os.Unsetenv("GEMINI_API_KEY")
			}
			if tt.googleKey != "" {
				os.Setenv("GOOGLE_API_KEY", tt.googleKey)
				defer os.Unsetenv("GOOGLE_API_KEY")
			}

			provider, err := NewGeminiProvider(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, ProviderTypeGemini, provider.GetType())
				assert.Equal(t, "Gemini", provider.GetName())
			}
		})
	}
}

func TestGeminiProvider_GetType(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGemini,
		APIKey: "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	assert.Equal(t, ProviderTypeGemini, provider.GetType())
}

func TestGeminiProvider_GetName(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGemini,
		APIKey: "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "Gemini", provider.GetName())
}

func TestGeminiProvider_GetModels(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGemini,
		APIKey: "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	models := provider.GetModels()
	assert.NotEmpty(t, models)

	// Check that we have expected models
	modelNames := make(map[string]bool)
	for _, model := range models {
		modelNames[model.Name] = true
		assert.Equal(t, ProviderTypeGemini, model.Provider)
		assert.Greater(t, model.ContextSize, 0)
		assert.NotEmpty(t, model.Description)
	}

	// Verify key models exist
	assert.True(t, modelNames["gemini-2.5-pro"], "Should have Gemini 2.5 Pro")
	assert.True(t, modelNames["gemini-2.5-flash"], "Should have Gemini 2.5 Flash")
	assert.True(t, modelNames["gemini-2.0-flash"], "Should have Gemini 2.0 Flash")
	assert.True(t, modelNames["gemini-1.5-pro"], "Should have Gemini 1.5 Pro")
	assert.True(t, modelNames["gemini-1.5-flash"], "Should have Gemini 1.5 Flash")

	// Check massive context models
	for _, model := range models {
		if model.Name == "gemini-2.5-pro" || model.Name == "gemini-1.5-pro" {
			assert.Equal(t, 2097152, model.ContextSize, "Pro models should have 2M context")
		}
	}
}

func TestGeminiProvider_GetCapabilities(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGemini,
		APIKey: "test-key",
	}
	provider, err := NewGeminiProvider(config)
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

func TestGeminiProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name      string
		config    ProviderConfigEntry
		available bool
	}{
		{
			name: "available with API key",
			config: ProviderConfigEntry{
				Type:   ProviderTypeGemini,
				APIKey: "test-key",
			},
			available: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiProvider(tt.config)
			require.NoError(t, err)

			available := provider.IsAvailable(context.Background())
			assert.Equal(t, tt.available, available)
		})
	}
}

func TestGeminiProvider_Generate(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Contains(t, r.URL.Path, "generateContent")
		assert.Contains(t, r.URL.Query().Get("key"), "test-key")

		// Verify request body
		var req geminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.NotEmpty(t, req.Contents)
		assert.NotNil(t, req.GenerationConfig)

		// Return mock response
		response := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role: "model",
						Parts: []geminiPart{
							map[string]interface{}{
								"text": "Hello! This is a test response from Gemini.",
							},
						},
					},
					FinishReason: "STOP",
					Index:        0,
				},
			},
			UsageMetadata: &geminiUsageMetadata{
				PromptTokenCount:     15,
				CandidatesTokenCount: 25,
				TotalTokenCount:      40,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with mock endpoint
	config := ProviderConfigEntry{
		Type:     ProviderTypeGemini,
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	// Test generation
	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Content, "Hello! This is a test response from Gemini.")
	assert.Equal(t, 15, response.Usage.PromptTokens)
	assert.Equal(t, 25, response.Usage.CompletionTokens)
	assert.Equal(t, 40, response.Usage.TotalTokens)
	assert.Equal(t, "STOP", response.FinishReason)
}

func TestGeminiProvider_GenerateWithSystemInstruction(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req geminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify system instruction is present
		assert.NotNil(t, req.SystemInstruction)
		assert.NotEmpty(t, req.SystemInstruction.Parts)

		response := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role: "model",
						Parts: []geminiPart{
							map[string]interface{}{"text": "Response with system instruction"},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &geminiUsageMetadata{
				PromptTokenCount:     20,
				CandidatesTokenCount: 10,
				TotalTokenCount:      30,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     ProviderTypeGemini,
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 1000,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Content, "Response with system instruction")
}

func TestGeminiProvider_GenerateWithTools(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req geminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify tools are present
		assert.NotEmpty(t, req.Tools)
		assert.NotNil(t, req.ToolConfig)
		assert.Equal(t, "AUTO", req.ToolConfig.FunctionCallingConfig.Mode)

		// Return function call response
		response := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role: "model",
						Parts: []geminiPart{
							map[string]interface{}{
								"functionCall": map[string]interface{}{
									"name": "get_weather",
									"args": map[string]interface{}{
										"location": "San Francisco",
									},
								},
							},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &geminiUsageMetadata{
				PromptTokenCount:     50,
				CandidatesTokenCount: 20,
				TotalTokenCount:      70,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     ProviderTypeGemini,
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{Role: "user", Content: "What's the weather in San Francisco?"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
		Tools: []Tool{
			{
				Type: "function",
				Function: FunctionDefinition{
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
	assert.Equal(t, "get_weather", response.ToolCalls[0].Function.Name)
	location, ok := response.ToolCalls[0].Function.Arguments["location"].(string)
	assert.True(t, ok)
	assert.Equal(t, "San Francisco", location)
}

func TestGeminiProvider_SafetySettings(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req geminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify safety settings are present
		assert.NotEmpty(t, req.SafetySettings)
		assert.Len(t, req.SafetySettings, 4)

		// Verify all are set to BLOCK_ONLY_HIGH
		for _, setting := range req.SafetySettings {
			assert.Equal(t, "BLOCK_ONLY_HIGH", setting.Threshold)
		}

		response := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role:  "model",
						Parts: []geminiPart{map[string]interface{}{"text": "Safe response"}},
					},
					FinishReason: "STOP",
				},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     ProviderTypeGemini,
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "gemini-2.5-flash",
		Messages: []Message{{Role: "user", Content: "Test"}},
	}

	_, err = provider.Generate(context.Background(), request)
	require.NoError(t, err)
}

func TestGeminiProvider_ErrorHandling(t *testing.T) {
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
				"error": {
					"code": 400,
					"message": "Invalid request",
					"status": "INVALID_ARGUMENT"
				}
			}`,
			expectedError: "Invalid request",
		},
		{
			name:       "API error 401",
			statusCode: 401,
			responseBody: `{
				"error": {
					"code": 401,
					"message": "Invalid API key",
					"status": "UNAUTHENTICATED"
				}
			}`,
			expectedError: "Invalid API key",
		},
		{
			name:       "API error 429",
			statusCode: 429,
			responseBody: `{
				"error": {
					"code": 429,
					"message": "Resource exhausted",
					"status": "RESOURCE_EXHAUSTED"
				}
			}`,
			expectedError: "Resource exhausted",
		},
		{
			name:       "API error 500",
			statusCode: 500,
			responseBody: `{
				"error": {
					"code": 500,
					"message": "Internal server error",
					"status": "INTERNAL"
				}
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
				Type:     ProviderTypeGemini,
				Endpoint: server.URL,
				APIKey:   "test-key",
			}
			provider, err := NewGeminiProvider(config)
			require.NoError(t, err)

			request := &LLMRequest{
				ID:       uuid.New(),
				Model:    "gemini-2.5-flash",
				Messages: []Message{{Role: "user", Content: "Hello"}},
			}

			_, err = provider.Generate(context.Background(), request)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestGeminiProvider_GetHealth(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Role:  "model",
						Parts: []geminiPart{map[string]interface{}{"text": "OK"}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &geminiUsageMetadata{
				PromptTokenCount:     5,
				CandidatesTokenCount: 5,
				TotalTokenCount:      10,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     ProviderTypeGemini,
		Endpoint: server.URL,
		APIKey:   "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	health, err := provider.GetHealth(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
	assert.Greater(t, health.Latency, time.Duration(0))
	assert.Greater(t, health.ModelCount, 0)
}

func TestGeminiProvider_Close(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGemini,
		APIKey: "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

func TestGeminiProvider_MessageConversion(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGemini,
		APIKey: "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	messages := []Message{
		{Role: "system", Content: "System message"},
		{Role: "user", Content: "User message 1"},
		{Role: "assistant", Content: "Assistant message 1"},
		{Role: "user", Content: "User message 2"},
	}

	systemMsg, contents := provider.convertMessages(messages)

	// Verify system message extracted
	assert.Equal(t, "System message", systemMsg)

	// Verify contents converted correctly
	assert.Len(t, contents, 3) // System message excluded from contents

	assert.Equal(t, "user", contents[0].Role)
	assert.Equal(t, "model", contents[1].Role) // assistant -> model
	assert.Equal(t, "user", contents[2].Role)
}

func TestGeminiProvider_MassiveContext(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGemini,
		APIKey: "test-key",
	}
	provider, err := NewGeminiProvider(config)
	require.NoError(t, err)

	models := provider.GetModels()

	// Find 2M context models
	massiveContextModels := []string{}
	for _, model := range models {
		if model.ContextSize == 2097152 {
			massiveContextModels = append(massiveContextModels, model.Name)
		}
	}

	// Verify we have 2M context models
	assert.NotEmpty(t, massiveContextModels)
	assert.Contains(t, massiveContextModels, "gemini-2.5-pro")
	assert.Contains(t, massiveContextModels, "gemini-1.5-pro")
}
