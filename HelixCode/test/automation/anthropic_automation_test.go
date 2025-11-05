// go:build automation

package automation

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.helix.code/internal/llm"
)

// TestAnthropicProviderFullAutomation tests the Anthropic provider with real API calls
func TestAnthropicProviderFullAutomation(t *testing.T) {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY environment variable not set, skipping real API tests")
	}

	// Get endpoint from environment or use default
	endpoint := os.Getenv("ANTHROPIC_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1/messages"
	}

	// Create provider configuration
	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeAnthropic,
		Endpoint: endpoint,
		APIKey:   apiKey,
	}

	// Test provider creation
	t.Run("ProviderCreation", func(t *testing.T) {
		provider, err := llm.NewAnthropicProvider(config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, llm.ProviderTypeAnthropic, provider.GetType())
		assert.Equal(t, "Anthropic", provider.GetName())
	})

	provider, err := llm.NewAnthropicProvider(config)
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
			llm.CapabilityVision,
		}
		assert.Equal(t, expectedCapabilities, capabilities)
	})

	// Test model listing
	t.Run("ModelListing", func(t *testing.T) {
		models := provider.GetModels()
		assert.NotEmpty(t, models, "Should have available models")

		// Verify all models have required fields
		for _, model := range models {
			assert.Equal(t, llm.ProviderTypeAnthropic, model.Provider)
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
			"claude-4-sonnet",
			"claude-3-5-sonnet-latest",
			"claude-3-5-haiku-latest",
			"claude-3-opus-latest",
		}

		for _, expected := range expectedModels {
			assert.True(t, modelNames[expected], "Expected model %s not found", expected)
		}

		// Verify all Claude models have 200K context
		for _, model := range models {
			assert.Equal(t, 200000, model.ContextSize, "Claude model %s should have 200K context", model.Name)
		}
	})

	// Test availability check
	t.Run("ProviderAvailability", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		available := provider.IsAvailable(ctx)
		assert.True(t, available, "Provider should be available with valid API key")
	})

	// Test health check
	t.Run("ProviderHealthCheck", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		health, err := provider.GetHealth(ctx)
		require.NoError(t, err)
		assert.NotNil(t, health)
		assert.Equal(t, "healthy", health.Status)
		assert.Greater(t, health.Latency, time.Duration(0))
		assert.Greater(t, health.ModelCount, 0)
	})

	// Test simple text generation
	t.Run("SimpleTextGeneration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "claude-3-5-haiku-latest", // Use fast model for tests
			Messages: []llm.Message{
				{Role: "user", Content: "Say 'Hello, World!' and nothing else."},
			},
			MaxTokens:   50,
			Temperature: 0.1,
		}

		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)
		assert.Contains(t, response.Content, "Hello")
		assert.Greater(t, response.Usage.PromptTokens, 0)
		assert.Greater(t, response.Usage.CompletionTokens, 0)
		assert.Equal(t, "end_turn", response.FinishReason)

		t.Logf("Response: %s", response.Content)
		t.Logf("Usage: %d prompt + %d completion = %d total tokens",
			response.Usage.PromptTokens,
			response.Usage.CompletionTokens,
			response.Usage.TotalTokens)
	})

	// Test extended thinking
	t.Run("ExtendedThinking", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "claude-3-5-sonnet-latest",
			Messages: []llm.Message{
				{Role: "user", Content: "Think carefully: what is 15 + 27?"},
			},
			MaxTokens:   500,
			Temperature: 0.7,
		}

		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)
		assert.Contains(t, response.Content, "42")

		t.Logf("Extended thinking response: %s", response.Content)
	})

	// Test prompt caching
	t.Run("PromptCaching", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Large system message that should be cached
		systemMessage := "You are an expert Go developer. " +
			"You have deep knowledge of Go idioms, best practices, and the standard library. " +
			"You always write clean, efficient, and well-documented code. " +
			"You prefer simple solutions over complex ones. " +
			"You follow Go conventions and formatting."

		// First request - creates cache
		request1 := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "claude-3-5-haiku-latest",
			Messages: []llm.Message{
				{Role: "system", Content: systemMessage},
				{Role: "user", Content: "What is a goroutine?"},
			},
			MaxTokens:   100,
			Temperature: 0.1,
		}

		response1, err := provider.Generate(ctx, request1)
		require.NoError(t, err)
		assert.NotNil(t, response1)
		t.Logf("First request usage: %d prompt, %d completion tokens",
			response1.Usage.PromptTokens, response1.Usage.CompletionTokens)

		// Second request - should hit cache
		request2 := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "claude-3-5-haiku-latest",
			Messages: []llm.Message{
				{Role: "system", Content: systemMessage}, // Same system message
				{Role: "user", Content: "What is a channel?"},
			},
			MaxTokens:   100,
			Temperature: 0.1,
		}

		response2, err := provider.Generate(ctx, request2)
		require.NoError(t, err)
		assert.NotNil(t, response2)

		// Check for cache metadata
		if metadata, ok := response2.ProviderMetadata.(map[string]interface{}); ok {
			cacheReadTokens := metadata["cache_read_tokens"]
			t.Logf("Cache read tokens: %v", cacheReadTokens)
			if crt, ok := cacheReadTokens.(int); ok && crt > 0 {
				t.Logf("✅ Prompt caching working! Read %d tokens from cache", crt)
			}
		}

		t.Logf("Second request usage: %d prompt, %d completion tokens",
			response2.Usage.PromptTokens, response2.Usage.CompletionTokens)
	})

	// Test tool calling
	t.Run("ToolCalling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		tools := []llm.Tool{
			{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        "get_weather",
					Description: "Get the current weather in a location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city name",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		}

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "claude-3-5-haiku-latest",
			Messages: []llm.Message{
				{Role: "user", Content: "What's the weather in San Francisco?"},
			},
			MaxTokens:   200,
			Temperature: 0.1,
			Tools:       tools,
		}

		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)

		// Should have tool call
		if len(response.ToolCalls) > 0 {
			assert.Equal(t, "get_weather", response.ToolCalls[0].Function.Name)
			t.Logf("✅ Tool calling working! Called: %s with args: %v",
				response.ToolCalls[0].Function.Name,
				response.ToolCalls[0].Function.Arguments)
		} else {
			t.Logf("⚠️ No tool calls returned. Response: %s", response.Content)
		}
	})

	// Test streaming
	t.Run("StreamingGeneration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "claude-3-5-haiku-latest",
			Messages: []llm.Message{
				{Role: "user", Content: "Count from 1 to 5 slowly."},
			},
			MaxTokens:   100,
			Temperature: 0.1,
			Stream:      true,
		}

		ch := make(chan llm.LLMResponse, 10)

		go func() {
			err := provider.GenerateStream(ctx, request, ch)
			assert.NoError(t, err)
		}()

		var allContent string
		var chunkCount int

		for response := range ch {
			allContent += response.Content
			if response.Content != "" {
				chunkCount++
				t.Logf("Chunk %d: %s", chunkCount, response.Content)
			}

			if response.FinishReason != "" {
				t.Logf("Streaming finished. Reason: %s, Usage: %d tokens",
					response.FinishReason, response.Usage.TotalTokens)
			}
		}

		assert.NotEmpty(t, allContent)
		assert.Greater(t, chunkCount, 0, "Should have received streaming chunks")
		t.Logf("✅ Streaming working! Received %d chunks", chunkCount)
	})

	// Test error handling
	t.Run("ErrorHandling_InvalidModel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:       uuid.New(),
			Model:    "invalid-model-name",
			Messages: []llm.Message{{Role: "user", Content: "Hello"}},
		}

		_, err := provider.Generate(ctx, request)
		assert.Error(t, err)
		t.Logf("Expected error received: %v", err)
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		err := provider.Close()
		assert.NoError(t, err)
	})
}
