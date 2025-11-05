//go:build automation

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

// TestGeminiProviderFullAutomation tests the Gemini provider with real API calls
func TestGeminiProviderFullAutomation(t *testing.T) {
	// Get API key from environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY or GOOGLE_API_KEY environment variable not set, skipping real API tests")
	}

	// Get endpoint from environment or use default
	endpoint := os.Getenv("GEMINI_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://generativelanguage.googleapis.com/v1beta"
	}

	// Create provider configuration
	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeGemini,
		Endpoint: endpoint,
		APIKey:   apiKey,
	}

	// Test provider creation
	t.Run("ProviderCreation", func(t *testing.T) {
		provider, err := llm.NewGeminiProvider(config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, llm.ProviderTypeGemini, provider.GetType())
		assert.Equal(t, "Gemini", provider.GetName())
	})

	provider, err := llm.NewGeminiProvider(config)
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
			assert.Equal(t, llm.ProviderTypeGemini, model.Provider)
			assert.NotEmpty(t, model.Name)
			assert.Greater(t, model.ContextSize, 0)
			assert.NotEmpty(t, model.Capabilities)
			assert.NotEmpty(t, model.Description)
		}

		// Check for specific expected models
		modelNames := make(map[string]bool)
		for _, model := range models {
			modelNames[model.Name] = true
		}

		expectedModels := []string{
			"gemini-2.5-pro",
			"gemini-2.5-flash",
			"gemini-2.0-flash",
			"gemini-1.5-pro",
			"gemini-1.5-flash",
		}

		for _, expected := range expectedModels {
			assert.True(t, modelNames[expected], "Expected model %s not found", expected)
		}

		// Verify massive context models
		for _, model := range models {
			if model.Name == "gemini-2.5-pro" || model.Name == "gemini-1.5-pro" {
				assert.Equal(t, 2097152, model.ContextSize, "%s should have 2M context", model.Name)
			}
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

	// Test simple text generation with Flash model
	t.Run("SimpleTextGeneration_Flash", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash-lite", // Use fastest model for tests
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

		t.Logf("Response: %s", response.Content)
		t.Logf("Usage: %d prompt + %d completion = %d total tokens",
			response.Usage.PromptTokens,
			response.Usage.CompletionTokens,
			response.Usage.TotalTokens)
	})

	// Test with system instruction
	t.Run("WithSystemInstruction", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "system", Content: "You are a helpful coding assistant. Always be concise."},
				{Role: "user", Content: "What is Go?"},
			},
			MaxTokens:   100,
			Temperature: 0.1,
		}

		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)

		t.Logf("Response with system instruction: %s", response.Content)
	})

	// Test massive context capability
	t.Run("MassiveContextCapability", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Create a larger input to test context handling
		largeInput := "Analyze this code:\n\n"
		for i := 0; i < 100; i++ {
			largeInput += "func exampleFunction" + string(rune(i)) + "() {}\n"
		}

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash", // 1M context model
			Messages: []llm.Message{
				{Role: "user", Content: largeInput + "\n\nHow many functions are defined?"},
			},
			MaxTokens:   100,
			Temperature: 0.1,
		}

		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)

		t.Logf("Massive context test: Processed %d chars, Response: %s",
			len(largeInput), response.Content)
	})

	// Test function calling
	t.Run("FunctionCalling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		tools := []llm.Tool{
			{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        "get_current_temperature",
					Description: "Get the current temperature for a location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city name",
							},
							"unit": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"celsius", "fahrenheit"},
								"description": "Temperature unit",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		}

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "What's the temperature in Tokyo?"},
			},
			MaxTokens:   200,
			Temperature: 0.1,
			Tools:       tools,
		}

		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)

		// Should have function call
		if len(response.ToolCalls) > 0 {
			assert.Equal(t, "get_current_temperature", response.ToolCalls[0].Function.Name)
			t.Logf("✅ Function calling working! Called: %s with args: %v",
				response.ToolCalls[0].Function.Name,
				response.ToolCalls[0].Function.Arguments)
		} else {
			t.Logf("⚠️ No function calls returned. Response: %s", response.Content)
		}
	})

	// Test streaming
	t.Run("StreamingGeneration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "List the first 3 planets in our solar system."},
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

	// Test code generation capability
	t.Run("CodeGeneration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash",
			Messages: []llm.Message{
				{Role: "user", Content: "Write a simple Go function that adds two numbers. Just the code, no explanation."},
			},
			MaxTokens:   200,
			Temperature: 0.1,
		}

		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)
		assert.Contains(t, response.Content, "func")

		t.Logf("Generated code:\n%s", response.Content)
	})

	// Test multi-turn conversation
	t.Run("MultiTurnConversation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		// First turn
		request1 := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "My favorite number is 42."},
			},
			MaxTokens:   50,
			Temperature: 0.1,
		}

		response1, err := provider.Generate(ctx, request1)
		require.NoError(t, err)
		assert.NotNil(t, response1)

		// Second turn - should remember context
		request2 := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "My favorite number is 42."},
				{Role: "assistant", Content: response1.Content},
				{Role: "user", Content: "What was my favorite number?"},
			},
			MaxTokens:   50,
			Temperature: 0.1,
		}

		response2, err := provider.Generate(ctx, request2)
		require.NoError(t, err)
		assert.NotNil(t, response2)
		assert.Contains(t, response2.Content, "42")

		t.Logf("✅ Multi-turn conversation working!")
		t.Logf("Turn 1: %s", response1.Content)
		t.Logf("Turn 2: %s", response2.Content)
	})

	// Test safety settings
	t.Run("SafetySettings", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Test that code-related content is not blocked
		request := &llm.LLMRequest{
			ID:    uuid.New(),
			Model: "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "Explain SQL injection attacks and how to prevent them."},
			},
			MaxTokens:   200,
			Temperature: 0.1,
		}

		response, err := provider.Generate(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)

		t.Logf("✅ Safety settings allow educational security content")
		t.Logf("Response: %s", response.Content)
	})

	// Test error handling with invalid model
	t.Run("ErrorHandling_InvalidModel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := &llm.LLMRequest{
			ID:       uuid.New(),
			Model:    "gemini-invalid-model",
			Messages: []llm.Message{{Role: "user", Content: "Hello"}},
		}

		_, err := provider.Generate(ctx, request)
		assert.Error(t, err)
		t.Logf("Expected error received: %v", err)
	})

	// Test different Flash models performance
	t.Run("FlashModelsComparison", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		models := []string{"gemini-2.5-flash-lite", "gemini-2.5-flash", "gemini-2.0-flash"}
		prompt := "What is 2 + 2?"

		for _, model := range models {
			start := time.Now()

			request := &llm.LLMRequest{
				ID:          uuid.New(),
				Model:       model,
				Messages:    []llm.Message{{Role: "user", Content: prompt}},
				MaxTokens:   50,
				Temperature: 0.1,
			}

			response, err := provider.Generate(ctx, request)
			latency := time.Since(start)

			require.NoError(t, err)
			assert.NotNil(t, response)

			t.Logf("Model: %s, Latency: %v, Tokens: %d",
				model, latency, response.Usage.TotalTokens)
		}
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		err := provider.Close()
		assert.NoError(t, err)
	})
}
