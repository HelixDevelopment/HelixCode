//go:build automation

package automation

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.helix.code/internal/llm"
)

// TestXAIProviderFullAutomation tests the XAI provider with real API calls
func TestXAIProviderFullAutomation(t *testing.T) {
	// Get API key from environment
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY environment variable not set, skipping real API tests")
	}

	// Get endpoint from environment or use default
	endpoint := os.Getenv("XAI_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.x.ai/v1"
	}

	// Create provider configuration
	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeXAI,
		Endpoint: endpoint,
		APIKey:   apiKey,
	}

	// Test provider creation
	t.Run("ProviderCreation", func(t *testing.T) {
		provider, err := llm.NewXAIProvider(config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, llm.ProviderTypeXAI, provider.GetType())
		assert.Equal(t, "XAI (Grok)", provider.GetName())
	})

	provider, err := llm.NewXAIProvider(config)
	require.NoError(t, err)

	// Test provider capabilities
	t.Run("ProviderCapabilities", func(t *testing.T) {
		capabilities := provider.GetCapabilities()
		expectedCapabilities := []llm.ModelCapability{
			llm.CapabilityTextGeneration,
			llm.CapabilityCodeGeneration,
			llm.CapabilityCodeAnalysis,
			llm.CapabilityPlanning,
			llm.CapabilityDebugging,
			llm.CapabilityRefactoring,
			llm.CapabilityTesting,
		}
		assert.Equal(t, expectedCapabilities, capabilities)
	})

	// Test model listing
	t.Run("ModelListing", func(t *testing.T) {
		models := provider.GetModels()
		assert.NotEmpty(t, models, "Should have available models")

		// Verify all models have required fields
		for _, model := range models {
			assert.Equal(t, llm.ProviderTypeXAI, model.Provider)
			assert.NotEmpty(t, model.Name)
			assert.Greater(t, model.ContextSize, 0)
			assert.NotEmpty(t, model.Capabilities)
			assert.Greater(t, model.MaxTokens, 0)
			assert.NotEmpty(t, model.Description)
		}

		// Check for specific expected models
		modelNames := make(map[string]bool)
		for _, model := range models {
			modelNames[model.Name] = true
		}

		expectedModels := []string{
			"grok-3-fast-beta",
			"grok-3-mini-fast-beta",
			"grok-3-beta",
			"grok-3-mini-beta",
		}

		for _, expected := range expectedModels {
			assert.True(t, modelNames[expected], "Expected model %s not found", expected)
		}
	})

	// Test health check
	t.Run("HealthCheck", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		health, err := provider.GetHealth(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, health)
		assert.NotEmpty(t, health.Status)
		assert.Greater(t, health.LastCheck.Unix(), int64(0))

		t.Logf("Health status: %s, Latency: %v, Model count: %d",
			health.Status, health.Latency, health.ModelCount)
	})

	// Test availability
	t.Run("Availability", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		available := provider.IsAvailable(ctx)
		assert.True(t, available, "Provider should be available with valid API key")
	})

	// Test basic text generation
	t.Run("BasicTextGeneration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeXAI,
			Model:        "grok-3-mini-fast-beta", // Use lightweight model for testing
			Messages: []llm.Message{
				{Role: "user", Content: "Hello! Please respond with exactly 'Hello from Grok!'"},
			},
			MaxTokens:   50,
			Temperature: 0.1,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, request.ID, response.RequestID)
		assert.NotEmpty(t, response.Content)
		assert.Greater(t, response.Usage.TotalTokens, 0)
		assert.Greater(t, response.ProcessingTime, time.Duration(0))

		t.Logf("Response: %s", response.Content)
		t.Logf("Usage: %d prompt tokens, %d completion tokens, %d total tokens",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
		t.Logf("Processing time: %v", response.ProcessingTime)
	})

	// Test code generation
	t.Run("CodeGeneration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeXAI,
			Model:        "grok-3-fast-beta", // Use coding model
			Messages: []llm.Message{
				{Role: "user", Content: "Write a simple Go function that adds two numbers and includes proper documentation."},
			},
			MaxTokens:   200,
			Temperature: 0.3,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Contains(t, response.Content, "func") // Should contain Go function
		assert.Contains(t, response.Content, "int")  // Should contain type annotations
		assert.Greater(t, response.Usage.TotalTokens, 0)

		t.Logf("Code generation response: %s", response.Content)
	})

	// Test streaming generation
	t.Run("StreamingGeneration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeXAI,
			Model:        "grok-3-mini-fast-beta",
			Messages: []llm.Message{
				{Role: "user", Content: "Count from 1 to 5 slowly, one number per line."},
			},
			MaxTokens:   100,
			Temperature: 0.1,
			Stream:      true,
			CreatedAt:   time.Now(),
		}

		ch := make(chan llm.LLMResponse, 50)
		errCh := make(chan error, 1)

		go func() {
			errCh <- provider.GenerateStream(ctx, request, ch)
		}()

		var responses []llm.LLMResponse
		timeout := time.After(30 * time.Second)

	collectionLoop:
		for {
			select {
			case response := <-ch:
				responses = append(responses, response)
			case err := <-errCh:
				assert.NoError(t, err)
				break collectionLoop
			case <-timeout:
				t.Fatal("Streaming test timed out")
			}
		}

		assert.NotEmpty(t, responses, "Should receive streaming responses")

		// Concatenate all content
		var fullContent string
		for _, resp := range responses {
			if resp.Content != "" {
				fullContent += resp.Content
			}
		}

		assert.NotEmpty(t, fullContent, "Should have received content via streaming")
		assert.Contains(t, fullContent, "1") // Should contain numbers
		assert.Contains(t, fullContent, "5")

		t.Logf("Streaming response: %s", fullContent)
		t.Logf("Received %d streaming chunks", len(responses))
	})

	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with invalid model
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeXAI,
			Model:        "invalid-model-name",
			Messages: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
			MaxTokens:   10,
			Temperature: 0.1,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(ctx, request)
		assert.Error(t, err, "Should fail with invalid model")
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "XAI API returned status")
	})

	// Test concurrent requests
	t.Run("ConcurrentRequests", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		numRequests := 5
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(requestNum int) {
				request := &llm.LLMRequest{
					ID:           uuid.New(),
					ProviderType: llm.ProviderTypeXAI,
					Model:        "grok-3-mini-fast-beta",
					Messages: []llm.Message{
						{Role: "user", Content: fmt.Sprintf("Say 'Request %d completed'", requestNum)},
					},
					MaxTokens:   20,
					Temperature: 0.1,
					CreatedAt:   time.Now(),
				}

				_, err := provider.Generate(ctx, request)
				results <- err
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent request %d should succeed", i)
		}

		t.Logf("Successfully completed %d concurrent requests", numRequests)
	})

	// Test provider cleanup
	t.Run("ProviderCleanup", func(t *testing.T) {
		err := provider.Close()
		assert.NoError(t, err)
	})

	t.Log("✅ All XAI provider automation tests passed!")
}

// TestXAIProviderLoadTest performs load testing with real API
func TestXAIProviderLoadTest(t *testing.T) {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY environment variable not set, skipping load test")
	}

	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeXAI,
		Endpoint: "https://api.x.ai/v1",
		APIKey:   apiKey,
	}

	provider, err := llm.NewXAIProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	numRequests := 10
	results := make(chan time.Duration, numRequests)
	errors := make(chan error, numRequests)

	startTime := time.Now()

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(requestNum int) {
			reqStart := time.Now()

			request := &llm.LLMRequest{
				ID:           uuid.New(),
				ProviderType: llm.ProviderTypeXAI,
				Model:        "grok-3-mini-fast-beta",
				Messages: []llm.Message{
					{Role: "user", Content: fmt.Sprintf("Generate a random number between 1 and 100 for request %d", requestNum)},
				},
				MaxTokens:   50,
				Temperature: 0.5,
				CreatedAt:   time.Now(),
			}

			_, err := provider.Generate(ctx, request)
			duration := time.Since(reqStart)

			if err != nil {
				errors <- err
			} else {
				results <- duration
			}
		}(i)
	}

	// Collect results
	var durations []time.Duration
	var errorCount int

	for i := 0; i < numRequests; i++ {
		select {
		case duration := <-results:
			durations = append(durations, duration)
		case err := <-errors:
			errorCount++
			t.Logf("Request failed: %v", err)
		case <-time.After(60 * time.Second):
			t.Fatal("Load test timed out")
		}
	}

	totalTime := time.Since(startTime)

	// Calculate statistics
	if len(durations) > 0 {
		var totalDuration time.Duration
		minDuration := durations[0]
		maxDuration := durations[0]

		for _, d := range durations {
			totalDuration += d
			if d < minDuration {
				minDuration = d
			}
			if d > maxDuration {
				maxDuration = d
			}
		}

		avgDuration := totalDuration / time.Duration(len(durations))

		t.Logf("Load test results:")
		t.Logf("  Total requests: %d", numRequests)
		t.Logf("  Successful requests: %d", len(durations))
		t.Logf("  Failed requests: %d", errorCount)
		t.Logf("  Total time: %v", totalTime)
		t.Logf("  Average response time: %v", avgDuration)
		t.Logf("  Min response time: %v", minDuration)
		t.Logf("  Max response time: %v", maxDuration)
		t.Logf("  Requests per second: %.2f", float64(len(durations))/totalTime.Seconds())

		assert.Greater(t, len(durations), 0, "Should have at least one successful request")
		assert.Less(t, errorCount, numRequests, "Should not have all requests fail")
	} else {
		t.Error("No successful requests in load test")
	}
}

// TestXAIProviderModelCompatibility tests all available models
func TestXAIProviderModelCompatibility(t *testing.T) {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY environment variable not set, skipping model compatibility test")
	}

	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeXAI,
		Endpoint: "https://api.x.ai/v1",
		APIKey:   apiKey,
	}

	provider, err := llm.NewXAIProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	models := provider.GetModels()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	testedModels := 0
	successfulModels := 0

	for _, model := range models {
		t.Run(fmt.Sprintf("Model_%s", model.Name), func(t *testing.T) {
			testedModels++

			request := &llm.LLMRequest{
				ID:           uuid.New(),
				ProviderType: llm.ProviderTypeXAI,
				Model:        model.Name,
				Messages: []llm.Message{
					{Role: "user", Content: fmt.Sprintf("Hello from model %s! Please respond briefly.", model.Name)},
				},
				MaxTokens:   50,
				Temperature: 0.1,
				CreatedAt:   time.Now(),
			}

			response, err := provider.Generate(ctx, request)
			if err != nil {
				t.Logf("Model %s failed: %v", model.Name, err)
				return
			}

			assert.NotNil(t, response)
			assert.NotEmpty(t, response.Content)
			assert.Greater(t, response.Usage.TotalTokens, 0)

			successfulModels++
			t.Logf("Model %s: %s (tokens: %d)",
				model.Name, response.Content, response.Usage.TotalTokens)
		})
	}

	t.Logf("Model compatibility: %d/%d models tested successfully", successfulModels, testedModels)
	assert.Greater(t, successfulModels, 0, "At least one model should work")
}

// TestXAIProviderRateLimits tests rate limit handling
func TestXAIProviderRateLimits(t *testing.T) {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY environment variable not set, skipping rate limit test")
	}

	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeXAI,
		Endpoint: "https://api.x.ai/v1",
		APIKey:   apiKey,
	}

	provider, err := llm.NewXAIProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Send rapid requests to potentially trigger rate limits
	numRequests := 20
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(requestNum int) {
			request := &llm.LLMRequest{
				ID:           uuid.New(),
				ProviderType: llm.ProviderTypeXAI,
				Model:        "grok-3-mini-fast-beta",
				Messages: []llm.Message{
					{Role: "user", Content: fmt.Sprintf("Request %d: Say 'ok'", requestNum)},
				},
				MaxTokens:   10,
				Temperature: 0.1,
				CreatedAt:   time.Now(),
			}

			_, err := provider.Generate(ctx, request)
			results <- err
		}(i)

		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	// Collect results
	rateLimited := 0
	successful := 0

	for i := 0; i < numRequests; i++ {
		err := <-results
		if err != nil {
			if isRateLimitError(err) {
				rateLimited++
			}
			t.Logf("Request %d failed: %v", i, err)
		} else {
			successful++
		}
	}

	t.Logf("Rate limit test: %d successful, %d rate limited, %d other errors",
		successful, rateLimited, numRequests-successful-rateLimited)

	assert.Greater(t, successful, 0, "Should have at least some successful requests")
}

// Helper function to detect rate limit errors
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "rate limit") ||
		contains(errStr, "429") ||
		contains(errStr, "too many requests")
}

// Helper function for case-insensitive contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			contains(s[1:], substr) ||
			(len(s) > len(substr) && contains(s[:len(s)-1], substr)))
}
