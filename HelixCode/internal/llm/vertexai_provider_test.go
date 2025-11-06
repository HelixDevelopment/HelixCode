package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Mock credentials for testing
const mockServiceAccountJSON = `{
  "type": "service_account",
  "project_id": "test-project",
  "private_key_id": "test-key-id",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAy8Dbv8prpJ/0kKhlGeJYozo2t60EG8L0561g13R29LvMR5hy\nvGZlGJpmn65+A4xHXInJYiPuKzrKUnApeLZ+vw1HocOAZtWK0z3r26uA8kQYOKX9\nQt/DbCdvsF9wF8gRK0ptx9M2B2wz9B0QVQG1L3KVi5q4WZFz1rL7rXYm4BQKE0Wy\nQfJAFhZvW5RXCJfX3X8cXl9x8H5jWLr4GHPz8xo4XVEYqTEbq9qqR9UXRJbDiLxg\nHLRXXLqLmP0n4AqDpxQCwrW/VcXaJPmJdxGFhiPlEGDXvGqSYXMB+UpP+cCvuDJ5\n2YPdYPPP9p4RdvJ7x6CGl9R8PGApWO2cKsYEjQIDAQABAoIBADDCwk9X3f6vdK8g\nwFmLDLlTxR8HKd8J0hQuU8LQCHl3c3j/R0JRG9xE0n1QMHEVjQP8gNqfxL0UvGqA\nDsmPb7lIQPKi6t8gGTtZj8L3R3rOiZvF9KqPLF0N0y0hqHZJkHjqhChBLqNMHnFU\n5tMlGRi/nBaHqYJI5m5r5KJfxKKFYk3BqoKQvGYBYWC5yPFq6BvBZXPvPBMx5MvI\n2nNbYMY/TxG0cVlLN6b5j3XLvPVMBY6EYgP8LwVvLFYLFqQJxPTmP3WLB8qGx6OT\nyHFdqYQPKL4VQj0X8JvXqBnmOXYQJ8Y6e5hLJQiRJ7LQYh0RQx0mYNQYoXvPvGG8\nJkHYkwECgYEA5iX2xB7JN3zqGH1uGI3Rlv6hB3bvXGsGrNqF8G0GQQlGP7F8G7EG\nhKFPvEK3r9C7U9IKvVXlF1TLqEJqJ2R8RXxqFxlzKGYJpF7L9dVqcZLQrNpqXGTZ\nGPqB7j6rZBqF3FrGmVPqHQKVzCHRPLMF8tFXP4vL5L7mVrKQ1yMQqSECgYEA4lDM\nwKqvP6xZ8U0YpNGH4FYR5lZrKqRPqXqFvWYwGlj5cCKKqRPnPqY3FmGvKDqZmJ9Q\nmFqGl8lVXqXJH3FmKLFqGqH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3Fq\nGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH0CgYEAl8mQqJGH0R8tRqJGH3Fq\nGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1Yp\nYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH3m\nQqJGH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGECgYBqGmH1YpYqZH3mQqJG\nH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tR\nqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3F\nqGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH3mQKBgFqGmH1YpYqZH3mQqJG\nH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tR\nqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3FqGmH1YpYqZH3mQqJGH0R8tRqJGH3F\n-----END RSA PRIVATE KEY-----",
  "client_email": "test@test-project.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com"
}`

func TestNewVertexAIProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      ProviderConfigEntry
		envVars     map[string]string
		credFile    bool
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with credentials file",
			config: ProviderConfigEntry{
				Type: ProviderTypeVertexAI,
				Parameters: map[string]interface{}{
					"project_id": "test-project",
					"location":   "us-central1",
				},
			},
			credFile:    true,
			expectError: false,
		},
		{
			name: "valid config with env vars",
			config: ProviderConfigEntry{
				Type:       ProviderTypeVertexAI,
				Parameters: map[string]interface{}{},
			},
			envVars: map[string]string{
				"VERTEXAI_PROJECT":  "test-project",
				"VERTEXAI_LOCATION": "us-central1",
			},
			credFile:    true,
			expectError: false,
		},
		{
			name: "missing project ID and credentials",
			config: ProviderConfigEntry{
				Type: ProviderTypeVertexAI,
				Parameters: map[string]interface{}{
					"location": "us-central1",
				},
			},
			credFile:    false,
			expectError: true,
			errorMsg:    "failed to find credentials",
		},
		{
			name: "default location",
			config: ProviderConfigEntry{
				Type: ProviderTypeVertexAI,
				Parameters: map[string]interface{}{
					"project_id": "test-project",
				},
			},
			credFile:    true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
			os.Unsetenv("VERTEXAI_PROJECT")
			os.Unsetenv("VERTEXAI_LOCATION")
			os.Unsetenv("GCP_PROJECT")

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Create temporary credentials file if needed
			var tmpFile *os.File
			if tt.credFile {
				var err error
				tmpFile, err = os.CreateTemp("", "credentials-*.json")
				require.NoError(t, err)
				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.Write([]byte(mockServiceAccountJSON))
				require.NoError(t, err)
				tmpFile.Close()

				// Set credentials path in config or env
				if tt.config.Parameters["credentials_path"] == nil {
					os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tmpFile.Name())
					defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
				} else {
					tt.config.Parameters["credentials_path"] = tmpFile.Name()
				}
			}

			provider, err := NewVertexAIProvider(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, ProviderTypeVertexAI, provider.GetType())
				assert.Equal(t, "Vertex AI", provider.GetName())
				assert.NotEmpty(t, provider.projectID)
			}
		})
	}
}

func TestVertexAIProvider_GetType(t *testing.T) {
	provider := createMockVertexAIProvider(t)
	assert.Equal(t, ProviderTypeVertexAI, provider.GetType())
}

func TestVertexAIProvider_GetName(t *testing.T) {
	provider := createMockVertexAIProvider(t)
	assert.Equal(t, "Vertex AI", provider.GetName())
}

func TestVertexAIProvider_GetModels(t *testing.T) {
	provider := createMockVertexAIProvider(t)
	models := provider.GetModels()

	assert.NotEmpty(t, models)

	// Check that we have expected models
	modelNames := make(map[string]bool)
	for _, model := range models {
		modelNames[model.Name] = true
		assert.Equal(t, ProviderTypeVertexAI, model.Provider)
		assert.Greater(t, model.ContextSize, 0)
		assert.NotEmpty(t, model.Description)
	}

	// Verify Gemini models
	assert.True(t, modelNames["gemini-2.5-pro"], "Should have Gemini 2.5 Pro")
	assert.True(t, modelNames["gemini-2.5-flash"], "Should have Gemini 2.5 Flash")
	assert.True(t, modelNames["gemini-1.5-pro"], "Should have Gemini 1.5 Pro")
	assert.True(t, modelNames["gemini-1.5-flash"], "Should have Gemini 1.5 Flash")

	// Verify Claude models via Model Garden
	assert.True(t, modelNames["claude-sonnet-4@20250514"], "Should have Claude Sonnet 4")
	assert.True(t, modelNames["claude-3-7-sonnet@20250219"], "Should have Claude 3.7 Sonnet")

	// Check massive context models
	for _, model := range models {
		if model.Name == "gemini-2.5-pro" || model.Name == "gemini-1.5-pro" {
			assert.Equal(t, 2097152, model.ContextSize, "Pro models should have 2M context")
		}
	}
}

func TestVertexAIProvider_GetCapabilities(t *testing.T) {
	provider := createMockVertexAIProvider(t)
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

func TestVertexAIProvider_IsAvailable(t *testing.T) {
	provider := createMockVertexAIProvider(t)
	available := provider.IsAvailable(context.Background())
	assert.True(t, available)
}

func TestVertexAIProvider_GenerateGemini(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Contains(t, r.URL.Path, "/publishers/google/models")
		assert.Contains(t, r.URL.Path, "generateContent")
		assert.NotEmpty(t, r.Header.Get("Authorization"))

		// Verify request body
		var req vertexRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.NotEmpty(t, req.Contents)
		assert.NotNil(t, req.GenerationConfig)

		// Return mock response
		response := vertexResponse{
			Candidates: []vertexCandidate{
				{
					Content: vertexContent{
						Role: "model",
						Parts: []vertexPart{
							map[string]interface{}{
								"text": "Hello! This is a test response from Vertex AI Gemini.",
							},
						},
					},
					FinishReason: "STOP",
					Index:        0,
				},
			},
			UsageMetadata: &vertexUsageMetadata{
				PromptTokenCount:     15,
				CandidatesTokenCount: 25,
				TotalTokenCount:      40,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

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
	assert.Contains(t, response.Content, "Hello! This is a test response from Vertex AI Gemini.")
	assert.Equal(t, 15, response.Usage.PromptTokens)
	assert.Equal(t, 25, response.Usage.CompletionTokens)
	assert.Equal(t, 40, response.Usage.TotalTokens)
	assert.Equal(t, "STOP", response.FinishReason)
}

func TestVertexAIProvider_GenerateClaude(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Claude endpoint
		assert.Contains(t, r.URL.Path, "/publishers/anthropic/models")
		assert.Contains(t, r.URL.Path, "rawPredict")
		assert.NotEmpty(t, r.Header.Get("Authorization"))

		// Verify request body
		var req anthropicVertexRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, "vertex-2023-10-16", req.AnthropicVersion)
		assert.NotEmpty(t, req.Messages)

		// Return Claude response
		response := anthropicVertexResponse{
			ID:   "msg-123",
			Type: "message",
			Role: "assistant",
			Content: []anthropicVertexContent{
				{Type: "text", Text: "Hello from Claude via Vertex AI Model Garden!"},
			},
			StopReason: "end_turn",
			Usage: anthropicVertexUsage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

	// Test generation with Claude model
	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "claude-3-7-sonnet@20250219",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Content, "Hello from Claude via Vertex AI Model Garden!")
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 20, response.Usage.CompletionTokens)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

func TestVertexAIProvider_GenerateWithSystemInstruction(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req vertexRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify system instruction is present
		assert.NotNil(t, req.SystemInstruction)
		assert.NotEmpty(t, req.SystemInstruction.Parts)

		response := vertexResponse{
			Candidates: []vertexCandidate{
				{
					Content: vertexContent{
						Role: "model",
						Parts: []vertexPart{
							map[string]interface{}{"text": "Response with system instruction"},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &vertexUsageMetadata{
				PromptTokenCount:     20,
				CandidatesTokenCount: 10,
				TotalTokenCount:      30,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

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

func TestVertexAIProvider_GenerateWithTools(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req vertexRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify tools are present
		assert.NotEmpty(t, req.Tools)
		assert.NotNil(t, req.ToolConfig)
		assert.Equal(t, "AUTO", req.ToolConfig.FunctionCallingConfig.Mode)

		// Return function call response
		response := vertexResponse{
			Candidates: []vertexCandidate{
				{
					Content: vertexContent{
						Role: "model",
						Parts: []vertexPart{
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
			UsageMetadata: &vertexUsageMetadata{
				PromptTokenCount:     50,
				CandidatesTokenCount: 20,
				TotalTokenCount:      70,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

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

func TestVertexAIProvider_SafetySettings(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req vertexRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify safety settings are present
		assert.NotEmpty(t, req.SafetySettings)
		assert.Len(t, req.SafetySettings, 4)

		// Verify all are set to BLOCK_ONLY_HIGH
		for _, setting := range req.SafetySettings {
			assert.Equal(t, "BLOCK_ONLY_HIGH", setting.Threshold)
		}

		response := vertexResponse{
			Candidates: []vertexCandidate{
				{
					Content: vertexContent{
						Role:  "model",
						Parts: []vertexPart{map[string]interface{}{"text": "Safe response"}},
					},
					FinishReason: "STOP",
				},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "gemini-2.5-flash",
		Messages: []Message{{Role: "user", Content: "Test"}},
	}

	_, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
}

func TestVertexAIProvider_ErrorHandling(t *testing.T) {
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
			expectedError: "invalid request",
		},
		{
			name:       "API error 401",
			statusCode: 401,
			responseBody: `{
				"error": {
					"code": 401,
					"message": "Invalid credentials",
					"status": "UNAUTHENTICATED"
				}
			}`,
			expectedError: "authentication failed",
		},
		{
			name:       "API error 403",
			statusCode: 403,
			responseBody: `{
				"error": {
					"code": 403,
					"message": "Permission denied",
					"status": "PERMISSION_DENIED"
				}
			}`,
			expectedError: "permission denied",
		},
		{
			name:       "API error 404",
			statusCode: 404,
			responseBody: `{
				"error": {
					"code": 404,
					"message": "Model not found",
					"status": "NOT_FOUND"
				}
			}`,
			expectedError: "model not found",
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
			expectedError: "rate limited",
		},
		{
			name:       "API error 503",
			statusCode: 503,
			responseBody: `{
				"error": {
					"code": 503,
					"message": "Service unavailable",
					"status": "UNAVAILABLE"
				}
			}`,
			expectedError: "service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

			request := &LLMRequest{
				ID:       uuid.New(),
				Model:    "gemini-2.5-flash",
				Messages: []Message{{Role: "user", Content: "Hello"}},
			}

			_, err := provider.Generate(context.Background(), request)
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), tt.expectedError)
		})
	}
}

func TestVertexAIProvider_StreamingGemini(t *testing.T) {
	// Create mock server for streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "streamGenerateContent")
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "text/event-stream")

		// Send streaming responses
		responses := []string{
			`data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}`,
			`data: {"candidates":[{"content":{"parts":[{"text":" world"}]}}]}`,
			`data: {"candidates":[{"content":{"parts":[{"text":"!"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":10,"totalTokenCount":15}}`,
		}

		for _, resp := range responses {
			w.Write([]byte(resp + "\n\n"))
		}
	}))
	defer server.Close()

	provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "gemini-2.5-flash",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	responseCh := make(chan LLMResponse, 10)
	go func() {
		err := provider.GenerateStream(context.Background(), request, responseCh)
		assert.NoError(t, err)
	}()

	// Collect responses
	var responses []LLMResponse
	for resp := range responseCh {
		responses = append(responses, resp)
	}

	// Verify we got responses
	assert.NotEmpty(t, responses)

	// Check final response
	finalResp := responses[len(responses)-1]
	assert.Equal(t, "STOP", finalResp.FinishReason)
	assert.Equal(t, 5, finalResp.Usage.PromptTokens)
	assert.Equal(t, 10, finalResp.Usage.CompletionTokens)
}

func TestVertexAIProvider_StreamingClaudeNotSupported(t *testing.T) {
	provider := createMockVertexAIProvider(t)

	request := &LLMRequest{
		ID:       uuid.New(),
		Model:    "claude-3-7-sonnet@20250219",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	responseCh := make(chan LLMResponse)
	err := provider.GenerateStream(context.Background(), request, responseCh)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "streaming not supported for Claude")
}

func TestVertexAIProvider_GetHealth(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := vertexResponse{
			Candidates: []vertexCandidate{
				{
					Content: vertexContent{
						Role:  "model",
						Parts: []vertexPart{map[string]interface{}{"text": "OK"}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &vertexUsageMetadata{
				PromptTokenCount:     5,
				CandidatesTokenCount: 5,
				TotalTokenCount:      10,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := createMockVertexAIProviderWithEndpoint(t, server.URL)

	health, err := provider.GetHealth(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
	assert.Greater(t, health.Latency, time.Duration(0))
	assert.Greater(t, health.ModelCount, 0)
}

func TestVertexAIProvider_Close(t *testing.T) {
	provider := createMockVertexAIProvider(t)
	err := provider.Close()
	assert.NoError(t, err)
}

func TestVertexAIProvider_MessageConversion(t *testing.T) {
	provider := createMockVertexAIProvider(t)

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

func TestVertexAIProvider_IsClaudeModel(t *testing.T) {
	provider := createMockVertexAIProvider(t)

	tests := []struct {
		model    string
		isClaude bool
	}{
		{"gemini-2.5-flash", false},
		{"gemini-1.5-pro", false},
		{"claude-3-7-sonnet@20250219", true},
		{"claude-sonnet-4@20250514", true},
		{"text-bison@002", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := provider.isClaudeModel(tt.model)
			assert.Equal(t, tt.isClaude, result)
		})
	}
}

func TestTokenProvider_GetToken(t *testing.T) {
	// Create mock token
	mockToken := &oauth2.Token{
		AccessToken: "test-access-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	tp := &TokenProvider{
		tokenCache: mockToken,
	}

	// Test token retrieval
	token, err := tp.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", token)

	// Test token caching (should return same token)
	token2, err := tp.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, token, token2)
}

func TestTokenProvider_ExpiredToken(t *testing.T) {
	// Create expired token
	expiredToken := &oauth2.Token{
		AccessToken: "expired-token",
		Expiry:      time.Now().Add(-1 * time.Hour), // Expired
	}

	tp := &TokenProvider{
		tokenCache: expiredToken,
	}

	// Test that expired token is detected
	assert.False(t, tp.tokenCache.Valid())
}

// Helper functions for testing

func createMockVertexAIProvider(t *testing.T) *VertexAIProvider {
	return &VertexAIProvider{
		config: ProviderConfigEntry{
			Type: ProviderTypeVertexAI,
		},
		credentials: &google.Credentials{
			ProjectID: "test-project",
		},
		projectID:  "test-project",
		location:   "us-central1",
		endpoint:   "https://us-central1-aiplatform.googleapis.com",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		tokenProvider: &TokenProvider{
			tokenCache: &oauth2.Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(1 * time.Hour),
			},
		},
		models: getVertexAIModels(),
	}
}

func createMockVertexAIProviderWithEndpoint(t *testing.T, endpoint string) *VertexAIProvider {
	provider := createMockVertexAIProvider(t)
	provider.endpoint = endpoint
	return provider
}
