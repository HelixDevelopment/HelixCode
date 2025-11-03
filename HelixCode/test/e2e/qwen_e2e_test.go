//go:build e2e

package e2e

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

// TestQwenProviderEndToEnd tests the Qwen provider in a complete workflow
func TestQwenProviderEndToEnd(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("QWEN_API_KEY")
	if apiKey == "" {
		t.Skip("QWEN_API_KEY environment variable not set, skipping e2e test")
	}

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Create Qwen provider
	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeQwen,
		Endpoint: getEnvOrDefault("QWEN_ENDPOINT", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
		APIKey:   apiKey,
	}

	provider, err := llm.NewQwenProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	// Register provider with model manager
	err = env.ModelManager.RegisterProvider(provider)
	require.NoError(t, err)

	// Test 1: Basic model selection
	t.Run("ModelSelection", func(t *testing.T) {
		criteria := llm.ModelSelectionCriteria{
			TaskType: "code_generation",
			RequiredCapabilities: []llm.ModelCapability{
				llm.CapabilityCodeGeneration,
			},
			MaxTokens:         1000,
			QualityPreference: "balanced",
		}

		selectedModel, err := env.ModelManager.SelectOptimalModel(criteria)
		assert.NoError(t, err)
		assert.NotNil(t, selectedModel)
		assert.Equal(t, llm.ProviderTypeQwen, selectedModel.Provider)

		t.Logf("Selected model: %s (%s)", selectedModel.Name, selectedModel.Description)
	})

	// Test 2: Provider health monitoring
	t.Run("HealthMonitoring", func(t *testing.T) {
		health := env.ModelManager.HealthCheck(env.ctx)
		qwenHealth, exists := health[llm.ProviderTypeQwen]
		assert.True(t, exists, "Qwen provider should be in health check results")
		assert.NotNil(t, qwenHealth)
		assert.Equal(t, "healthy", qwenHealth.Status)

		t.Logf("Qwen provider health: %s (latency: %v, models: %d)",
			qwenHealth.Status, qwenHealth.Latency, qwenHealth.ModelCount)
	})

	// Test 3: End-to-end code generation workflow
	t.Run("CodeGenerationWorkflow", func(t *testing.T) {
		// Select a coding model
		criteria := llm.ModelSelectionCriteria{
			TaskType: "code_generation",
			RequiredCapabilities: []llm.ModelCapability{
				llm.CapabilityCodeGeneration,
				llm.CapabilityCodeAnalysis,
			},
			MaxTokens:         2000,
			QualityPreference: "quality",
		}

		model, err := env.ModelManager.SelectOptimalModel(criteria)
		require.NoError(t, err)

		// Generate code using the provider
		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeQwen,
			Model:        model.Name,
			Messages: []llm.Message{
				{
					Role:    "system",
					Content: "You are an expert Go developer. Write clean, well-documented code.",
				},
				{
					Role: "user",
					Content: `Write a Go function that implements a simple LRU cache with the following interface:

type LRUCache interface {
    Get(key string) (interface{}, bool)
    Put(key string, value interface{})
}

The cache should have a maximum capacity and evict the least recently used items when full.`,
				},
			},
			MaxTokens:   1500,
			Temperature: 0.3,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(env.ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)

		// Verify the response contains Go code
		assert.Contains(t, response.Content, "type LRUCache interface")
		assert.Contains(t, response.Content, "func")
		assert.Contains(t, response.Content, "Get")
		assert.Contains(t, response.Content, "Put")

		t.Logf("Generated LRU cache code (%d tokens):", response.Usage.TotalTokens)
		t.Log(response.Content)
	})

	// Test 4: Streaming workflow
	t.Run("StreamingWorkflow", func(t *testing.T) {
		model, err := env.ModelManager.SelectOptimalModel(llm.ModelSelectionCriteria{
			TaskType:  "text_generation",
			MaxTokens: 500,
		})
		require.NoError(t, err)

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeQwen,
			Model:        model.Name,
			Messages: []llm.Message{
				{Role: "user", Content: "Write a short story about a robot learning to paint. Make it exactly 3 paragraphs."},
			},
			MaxTokens:   800,
			Temperature: 0.7,
			Stream:      true,
			CreatedAt:   time.Now(),
		}

		ch := make(chan llm.LLMResponse, 100)
		errCh := make(chan error, 1)

		go func() {
			errCh <- provider.GenerateStream(env.ctx, request, ch)
		}()

		var fullContent string
		streamTimeout := time.After(60 * time.Second)
		chunkCount := 0

	collectionLoop:
		for {
			select {
			case response := <-ch:
				if response.Content != "" {
					fullContent += response.Content
					chunkCount++
				}
			case err := <-errCh:
				assert.NoError(t, err)
				break collectionLoop
			case <-streamTimeout:
				t.Fatal("Streaming workflow timed out")
			}
		}

		assert.NotEmpty(t, fullContent, "Should receive streaming content")
		assert.Greater(t, chunkCount, 0, "Should receive multiple chunks")
		assert.Contains(t, fullContent, "robot")
		assert.Contains(t, fullContent, "paint")

		t.Logf("Streaming story (%d chunks): %s", chunkCount, fullContent)
	})

	// Test 5: Multi-model comparison
	t.Run("MultiModelComparison", func(t *testing.T) {
		models := env.ModelManager.GetAvailableModels()
		qwenModels := filterModelsByProvider(models, llm.ProviderTypeQwen)

		if len(qwenModels) < 2 {
			t.Skip("Need at least 2 Qwen models for comparison test")
		}

		testPrompt := "Explain quantum computing in simple terms."
		results := make(map[string]string)

		for _, model := range qwenModels[:2] { // Test first 2 models
			request := &llm.LLMRequest{
				ID:           uuid.New(),
				ProviderType: llm.ProviderTypeQwen,
				Model:        model.Name,
				Messages: []llm.Message{
					{Role: "user", Content: testPrompt},
				},
				MaxTokens:   300,
				Temperature: 0.5,
				CreatedAt:   time.Now(),
			}

			response, err := provider.Generate(env.ctx, request)
			if err != nil {
				t.Logf("Model %s failed: %v", model.Name, err)
				continue
			}

			results[model.Name] = response.Content
			t.Logf("Model %s response (%d tokens): %s",
				model.Name, response.Usage.TotalTokens, response.Content[:min(100, len(response.Content))])
		}

		assert.Greater(t, len(results), 0, "At least one model should succeed")
	})

	// Test 6: Error handling and recovery
	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with invalid model
		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeQwen,
			Model:        "nonexistent-model-12345",
			Messages: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
			MaxTokens: 10,
			CreatedAt: time.Now(),
		}

		response, err := provider.Generate(env.ctx, request)
		assert.Error(t, err, "Should fail with invalid model")
		assert.Nil(t, response)

		// Test recovery with valid model
		request.Model = "qwen-turbo"
		response, err = provider.Generate(env.ctx, request)
		assert.NoError(t, err, "Should succeed after error")
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)
	})

	t.Log("✅ Qwen provider end-to-end tests completed successfully!")
}

// TestQwenProviderDistributedWorkflow tests Qwen in a distributed worker scenario
func TestQwenProviderDistributedWorkflow(t *testing.T) {
	// This test would require setting up distributed workers
	// For now, skip if not in full distributed environment
	if os.Getenv("DISTRIBUTED_TEST") != "true" {
		t.Skip("Skipping distributed workflow test - set DISTRIBUTED_TEST=true to run")
	}

	apiKey := os.Getenv("QWEN_API_KEY")
	if apiKey == "" {
		t.Skip("QWEN_API_KEY not set")
	}

	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Setup would include:
	// 1. Register Qwen provider
	// 2. Create distributed workers
	// 3. Submit tasks that use Qwen models
	// 4. Verify task completion across workers

	t.Log("Distributed workflow test not yet implemented")
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func filterModelsByProvider(models []*llm.ModelInfo, providerType llm.ProviderType) []*llm.ModelInfo {
	var filtered []*llm.ModelInfo
	for _, model := range models {
		if model.Provider == providerType {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
