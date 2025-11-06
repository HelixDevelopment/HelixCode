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
)

func TestNewGroqProvider(t *testing.T) {
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
				Type:     ProviderTypeGroq,
				Endpoint: "https://api.groq.com",
				APIKey:   "gsk_test_key",
			},
			expectError: false,
		},
		{
			name: "valid config with env API key",
			config: ProviderConfigEntry{
				Type:     ProviderTypeGroq,
				Endpoint: "https://api.groq.com",
			},
			envKey:      "gsk_test_env_key",
			expectError: false,
		},
		{
			name: "missing API key",
			config: ProviderConfigEntry{
				Type:     ProviderTypeGroq,
				Endpoint: "https://api.groq.com",
			},
			expectError: true,
			errorMsg:    "API key not provided",
		},
		{
			name: "default endpoint",
			config: ProviderConfigEntry{
				Type:   ProviderTypeGroq,
				APIKey: "gsk_test_key",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if specified
			if tt.envKey != "" {
				os.Setenv("GROQ_API_KEY", tt.envKey)
				defer os.Unsetenv("GROQ_API_KEY")
			} else {
				os.Unsetenv("GROQ_API_KEY")
			}

			provider, err := NewGroqProvider(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, ProviderTypeGroq, provider.GetType())
				assert.Equal(t, "Groq", provider.GetName())
				assert.NotNil(t, provider.latencyMetrics)
			}
		})
	}
}

func TestGroqProvider_GetType(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGroq,
		APIKey: "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	assert.Equal(t, ProviderTypeGroq, provider.GetType())
}

func TestGroqProvider_GetName(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGroq,
		APIKey: "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "Groq", provider.GetName())
}

func TestGroqProvider_GetModels(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGroq,
		APIKey: "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	models := provider.GetModels()
	assert.NotEmpty(t, models)
	assert.GreaterOrEqual(t, len(models), 6, "Should have at least 6 models")

	// Check that we have expected models
	modelNames := make(map[string]bool)
	for _, model := range models {
		modelNames[model.Name] = true
		assert.Equal(t, ProviderTypeGroq, model.Provider)
		assert.Greater(t, model.ContextSize, 0)
		assert.NotEmpty(t, model.Description)
	}

	// Verify key models exist
	assert.True(t, modelNames["llama-3.3-70b-versatile"], "Should have Llama 3.3 70B")
	assert.True(t, modelNames["llama-3.1-70b-versatile"], "Should have Llama 3.1 70B")
	assert.True(t, modelNames["llama-3.1-8b-instant"], "Should have Llama 3.1 8B Instant")
	assert.True(t, modelNames["mixtral-8x7b-32768"], "Should have Mixtral 8x7B")
	assert.True(t, modelNames["gemma2-9b-it"], "Should have Gemma2 9B")
	assert.True(t, modelNames["gemma-7b-it"], "Should have Gemma 7B")
}

func TestGroqProvider_GetCapabilities(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGroq,
		APIKey: "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	caps := provider.GetCapabilities()
	assert.NotEmpty(t, caps)

	// Convert to map for easy checking
	capMap := make(map[ModelCapability]bool)
	for _, cap := range caps {
		capMap[cap] = true
	}

	// Verify expected capabilities
	assert.True(t, capMap[CapabilityTextGeneration])
	assert.True(t, capMap[CapabilityCodeGeneration])
	assert.True(t, capMap[CapabilityCodeAnalysis])
	assert.True(t, capMap[CapabilityPlanning])
	assert.True(t, capMap[CapabilityDebugging])
	assert.True(t, capMap[CapabilityRefactoring])
	assert.True(t, capMap[CapabilityTesting])
}

func TestGroqProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name      string
		config    ProviderConfigEntry
		available bool
	}{
		{
			name: "available with API key",
			config: ProviderConfigEntry{
				Type:   ProviderTypeGroq,
				APIKey: "gsk_test_key",
			},
			available: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGroqProvider(tt.config)
			require.NoError(t, err)

			available := provider.IsAvailable(context.Background())
			assert.Equal(t, tt.available, available)
		})
	}
}

func TestGroqProvider_Generate(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/openai/v1/chat/completions", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var req GroqRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, "llama-3.3-70b-versatile", req.Model)
		assert.Equal(t, 1000, req.MaxTokens)

		// Return mock response
		response := GroqResponse{
			ID:      "chatcmpl-groq-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "llama-3.3-70b-versatile",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hello! This is a test response from Groq.",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with mock endpoint
	config := ProviderConfigEntry{
		Type:     ProviderTypeGroq,
		Endpoint: server.URL,
		APIKey:   "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	// Test generation
	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "llama-3.3-70b-versatile",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Hello! This is a test response from Groq.", response.Content)
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 20, response.Usage.CompletionTokens)
	assert.Equal(t, 30, response.Usage.TotalTokens)
	assert.Equal(t, "stop", response.FinishReason)
}

func TestGroqProvider_GenerateStream(t *testing.T) {
	// Create mock server that returns SSE stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/openai/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		// Verify streaming is enabled
		var req GroqRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.True(t, req.Stream)

		w.Header().Set("Content-Type", "text/event-stream")

		// Send streaming chunks
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.3-70b-versatile","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":""}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.3-70b-versatile","choices":[{"index":0,"delta":{"content":" from"},"finish_reason":""}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.3-70b-versatile","choices":[{"index":0,"delta":{"content":" Groq"},"finish_reason":""}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.3-70b-versatile","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":15,"total_tokens":25}}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte("data: " + chunk + "\n\n"))
			w.(http.Flusher).Flush()
		}

		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     ProviderTypeGroq,
		Endpoint: server.URL,
		APIKey:   "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "llama-3.3-70b-versatile",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 1000,
		Stream:    true,
	}

	responseCh := make(chan LLMResponse)
	var responses []LLMResponse

	go func() {
		err := provider.GenerateStream(context.Background(), request, responseCh)
		assert.NoError(t, err)
	}()

	for response := range responseCh {
		responses = append(responses, response)
	}

	assert.NotEmpty(t, responses)

	// Check that we received incremental updates
	var contentBuilder strings.Builder
	for _, resp := range responses {
		contentBuilder.WriteString(resp.Content)
	}

	fullContent := contentBuilder.String()
	assert.Contains(t, fullContent, "Hello")
	assert.Contains(t, fullContent, "from")
	assert.Contains(t, fullContent, "Groq")

	// Last response should have finish reason and metadata
	lastResponse := responses[len(responses)-1]
	assert.Equal(t, "stop", lastResponse.FinishReason)

	if metadata, ok := lastResponse.ProviderMetadata.(map[string]interface{}); ok {
		assert.NotNil(t, metadata["first_token_latency_ms"])
		assert.NotNil(t, metadata["total_latency_ms"])
		assert.NotNil(t, metadata["tokens_per_second"])
	}
}

func TestGroqProvider_LatencyTracking(t *testing.T) {
	tracker := NewLatencyTracker(100)

	// Record some metrics
	tracker.RecordRequest(50*time.Millisecond, 500*time.Millisecond, 400.0)
	tracker.RecordRequest(60*time.Millisecond, 600*time.Millisecond, 450.0)
	tracker.RecordRequest(70*time.Millisecond, 700*time.Millisecond, 500.0)

	metrics := tracker.GetMetrics()

	assert.Equal(t, 3, metrics.SampleCount)
	assert.Greater(t, metrics.AvgFirstTokenLatency, time.Duration(0))
	assert.Greater(t, metrics.AvgTotalLatency, time.Duration(0))
	assert.Greater(t, metrics.AvgTokensPerSecond, 0.0)
	assert.Greater(t, metrics.P50FirstTokenLatency, time.Duration(0))
	assert.Greater(t, metrics.P95FirstTokenLatency, time.Duration(0))
	assert.Greater(t, metrics.P99FirstTokenLatency, time.Duration(0))
}

func TestGroqProvider_LatencyMetricsRetrieval(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGroq,
		APIKey: "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	// Initially should have no metrics
	metrics := provider.GetLatencyMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, 0, metrics.SampleCount)

	// Manually record some metrics
	provider.latencyMetrics.RecordRequest(50*time.Millisecond, 500*time.Millisecond, 400.0)

	// Should now have metrics
	metrics = provider.GetLatencyMetrics()
	assert.Equal(t, 1, metrics.SampleCount)
	assert.Greater(t, metrics.AvgTokensPerSecond, 0.0)
}

func TestGroqProvider_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:       "bad request",
			statusCode: 400,
			responseBody: `{
				"error": {
					"message": "Invalid model specified",
					"type": "invalid_request_error",
					"code": "invalid_model"
				}
			}`,
			expectedError: "invalid request",
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			responseBody: `{
				"error": {
					"message": "Invalid API key",
					"type": "authentication_error",
					"code": "invalid_api_key"
				}
			}`,
			expectedError: "unauthorized",
		},
		{
			name:       "rate limited",
			statusCode: 429,
			responseBody: `{
				"error": {
					"message": "Rate limit exceeded",
					"type": "rate_limit_error",
					"code": "rate_limit_exceeded"
				}
			}`,
			expectedError: "rate limited",
		},
		{
			name:       "context too long",
			statusCode: 400,
			responseBody: `{
				"error": {
					"message": "context_length_exceeded: Maximum context length exceeded",
					"type": "invalid_request_error",
					"code": "context_length_exceeded"
				}
			}`,
			expectedError: "context too long",
		},
		{
			name:       "service unavailable",
			statusCode: 503,
			responseBody: `{
				"error": {
					"message": "Service temporarily unavailable",
					"type": "service_error",
					"code": "service_unavailable"
				}
			}`,
			expectedError: "service unavailable",
		},
		{
			name:       "groq overloaded",
			statusCode: 529,
			responseBody: `{
				"error": {
					"message": "Service overloaded",
					"type": "overloaded_error",
					"code": "overloaded"
				}
			}`,
			expectedError: "groq overloaded",
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
				Type:     ProviderTypeGroq,
				Endpoint: server.URL,
				APIKey:   "gsk_test_key",
			}
			provider, err := NewGroqProvider(config)
			require.NoError(t, err)

			request := &LLMRequest{
				ID:    uuid.New(),
				Model: "llama-3.3-70b-versatile",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens: 1000,
			}

			_, err = provider.Generate(context.Background(), request)
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), tt.expectedError)
		})
	}
}

func TestGroqProvider_GetHealth(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GroqResponse{
			ID:      "chatcmpl-health",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "llama-3.1-8b-instant",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hi",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     5,
				CompletionTokens: 5,
				TotalTokens:      10,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     ProviderTypeGroq,
		Endpoint: server.URL,
		APIKey:   "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	health, err := provider.GetHealth(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
	assert.Greater(t, health.Latency, time.Duration(0))
	assert.Greater(t, health.ModelCount, 0)
}

func TestGroqProvider_HealthCheckFailure(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":{"message":"Service unavailable","type":"service_error"}}`))
	}))
	defer server.Close()

	config := ProviderConfigEntry{
		Type:     ProviderTypeGroq,
		Endpoint: server.URL,
		APIKey:   "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	health, err := provider.GetHealth(context.Background())
	assert.Error(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Equal(t, 1, health.ErrorCount)
}

func TestGroqProvider_Close(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGroq,
		APIKey: "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

func TestGroqProvider_ModelContextSizes(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGroq,
		APIKey: "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	models := provider.GetModels()

	// Verify Llama 3.x models have large context
	for _, model := range models {
		if strings.HasPrefix(model.Name, "llama-3") {
			assert.Equal(t, 131072, model.ContextSize, "Llama 3.x should have 128K context")
		}
		if model.Name == "mixtral-8x7b-32768" {
			assert.Equal(t, 32768, model.ContextSize, "Mixtral should have 32K context")
		}
		if strings.HasPrefix(model.Name, "gemma") {
			assert.Equal(t, 8192, model.ContextSize, "Gemma should have 8K context")
		}
	}
}

func TestGroqProvider_PercentileCalculations(t *testing.T) {
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}

	p50 := percentile(durations, 0.5)
	p95 := percentile(durations, 0.95)
	p99 := percentile(durations, 0.99)

	assert.Greater(t, p50, time.Duration(0))
	assert.Greater(t, p95, p50)
	assert.Greater(t, p99, p50)
	assert.LessOrEqual(t, p99, 100*time.Millisecond)
}

func TestGroqProvider_HTTP2Support(t *testing.T) {
	config := ProviderConfigEntry{
		Type:   ProviderTypeGroq,
		APIKey: "gsk_test_key",
	}
	provider, err := NewGroqProvider(config)
	require.NoError(t, err)

	// Verify HTTP/2 is enabled
	transport := provider.httpClient.Transport.(*http.Transport)
	assert.True(t, transport.ForceAttemptHTTP2, "HTTP/2 should be enabled for optimal performance")
	assert.Equal(t, 100, transport.MaxIdleConns, "Should have connection pooling")
	assert.Equal(t, 100, transport.MaxIdleConnsPerHost, "Should have per-host pooling")
}
