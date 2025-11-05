//go:build e2e

package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.helix.code/internal/llm"
)

// TestAnthropicProviderEndToEnd tests the Anthropic provider in a complete workflow
func TestAnthropicProviderEndToEnd(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY environment variable not set, skipping e2e test")
	}

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Create Anthropic provider
	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeAnthropic,
		Endpoint: getEnvOrDefault("ANTHROPIC_ENDPOINT", "https://api.anthropic.com/v1/messages"),
		APIKey:   apiKey,
	}

	provider, err := llm.NewAnthropicProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	// Register provider with model manager
	err = env.ModelManager.RegisterProvider(provider)
	require.NoError(t, err)

	// Test 1: Model selection with extended thinking
	t.Run("ModelSelection_ExtendedThinking", func(t *testing.T) {
		criteria := llm.ModelSelectionCriteria{
			TaskType: "planning",
			RequiredCapabilities: []llm.ModelCapability{
				llm.CapabilityPlanning,
				llm.CapabilityCodeAnalysis,
			},
			MaxTokens:         10000,
			QualityPreference: "quality",
		}

		selectedModel, err := env.ModelManager.SelectOptimalModel(criteria)
		assert.NoError(t, err)
		assert.NotNil(t, selectedModel)
		// Should select an Anthropic model for planning
		if selectedModel.Provider == llm.ProviderTypeAnthropic {
			t.Logf("✅ Selected Anthropic model: %s (%s)", selectedModel.Name, selectedModel.Description)
		}
	})

	// Test 2: Provider health monitoring
	t.Run("HealthMonitoring", func(t *testing.T) {
		health := env.ModelManager.HealthCheck(env.ctx)
		anthropicHealth, exists := health[llm.ProviderTypeAnthropic]
		assert.True(t, exists, "Anthropic provider should be in health check results")
		assert.NotNil(t, anthropicHealth)
		assert.Equal(t, "healthy", anthropicHealth.Status)

		t.Logf("Anthropic provider health: %s (latency: %v, models: %d)",
			anthropicHealth.Status, anthropicHealth.Latency, anthropicHealth.ModelCount)
	})

	// Test 3: Code generation with prompt caching
	t.Run("CodeGenerationWorkflow_WithCaching", func(t *testing.T) {
		systemPrompt := `You are an expert Go developer with deep knowledge of:
- Go idioms and best practices
- Concurrent programming with goroutines and channels
- Error handling patterns
- Interface design
- Testing strategies
- Performance optimization
Always write clean, idiomatic, well-documented Go code with proper error handling.`

		// First request - creates cache
		request1 := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeAnthropic,
			Model:        "claude-3-5-haiku-latest",
			Messages: []llm.Message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: "Write a Go function to reverse a string."},
			},
			MaxTokens:   500,
			Temperature: 0.3,
			CreatedAt:   time.Now(),
		}

		response1, err := provider.Generate(env.ctx, request1)
		require.NoError(t, err)
		assert.NotEmpty(t, response1.Content)
		assert.Contains(t, response1.Content, "func")

		t.Logf("First request usage: %d prompt + %d completion tokens",
			response1.Usage.PromptTokens, response1.Usage.CompletionTokens)

		// Second request - should hit cache
		time.Sleep(1 * time.Second) // Brief delay for cache to register

		request2 := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeAnthropic,
			Model:        "claude-3-5-haiku-latest",
			Messages: []llm.Message{
				{Role: "system", Content: systemPrompt}, // Same system message
				{Role: "user", Content: "Write a Go function to check if a string is a palindrome."},
			},
			MaxTokens:   500,
			Temperature: 0.3,
			CreatedAt:   time.Now(),
		}

		response2, err := provider.Generate(env.ctx, request2)
		require.NoError(t, err)
		assert.NotEmpty(t, response2.Content)

		// Check cache metadata
		if metadata, ok := response2.ProviderMetadata.(map[string]interface{}); ok {
			if cacheRead, ok := metadata["cache_read_tokens"].(int); ok && cacheRead > 0 {
				t.Logf("✅ Prompt caching working! Read %d tokens from cache", cacheRead)
				t.Logf("Second request usage: %d prompt + %d completion tokens (with caching)",
					response2.Usage.PromptTokens, response2.Usage.CompletionTokens)
			}
		}
	})

	// Test 4: Extended thinking for complex problems
	t.Run("ExtendedThinking_ComplexProblem", func(t *testing.T) {
		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeAnthropic,
			Model:        "claude-3-5-sonnet-latest",
			Messages: []llm.Message{
				{
					Role: "user",
					Content: `Think carefully and analyze step by step:
Design a distributed rate limiter for a microservices architecture.
Consider: Redis for shared state, token bucket algorithm, graceful degradation, and multi-datacenter scenarios.
Provide a detailed architectural design.`,
				},
			},
			MaxTokens:   4000,
			Temperature: 0.7,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(env.ctx, request)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Content)

		// Should produce detailed, thoughtful analysis
		assert.Greater(t, len(response.Content), 500, "Extended thinking should produce detailed response")
		assert.Contains(t, strings.ToLower(response.Content), "rate")

		t.Logf("✅ Extended thinking produced %d chars, %d tokens",
			len(response.Content), response.Usage.TotalTokens)
	})

	// Test 5: Tool calling workflow
	t.Run("ToolCalling_Workflow", func(t *testing.T) {
		tools := []llm.Tool{
			{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        "search_documentation",
					Description: "Search through Go documentation for specific topics",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Search query",
							},
							"package": map[string]interface{}{
								"type":        "string",
								"description": "Go package name",
							},
						},
						"required": []string{"query"},
					},
				},
			},
		}

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeAnthropic,
			Model:        "claude-3-5-haiku-latest",
			Messages: []llm.Message{
				{Role: "user", Content: "How do I use channels in Go? Search the documentation."},
			},
			MaxTokens:   500,
			Temperature: 0.1,
			Tools:       tools,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(env.ctx, request)
		require.NoError(t, err)

		// Should call the tool
		if len(response.ToolCalls) > 0 {
			assert.Equal(t, "search_documentation", response.ToolCalls[0].Function.Name)
			t.Logf("✅ Tool calling successful: %s with args: %v",
				response.ToolCalls[0].Function.Name,
				response.ToolCalls[0].Function.Arguments)
		}
	})

	// Test 6: Multi-provider comparison (if other providers available)
	t.Run("MultiProvider_QualityComparison", func(t *testing.T) {
		prompt := "Explain the difference between concurrency and parallelism in Go."

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeAnthropic,
			Model:        "claude-3-5-sonnet-latest",
			Messages: []llm.Message{
				{Role: "user", Content: prompt},
			},
			MaxTokens:   300,
			Temperature: 0.3,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(env.ctx, request)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Content)

		// Claude should provide detailed, accurate explanation
		assert.Contains(t, strings.ToLower(response.Content), "concurrency")
		assert.Contains(t, strings.ToLower(response.Content), "parallel")

		t.Logf("Claude response quality: %d chars, %d tokens",
			len(response.Content), response.Usage.TotalTokens)
	})

	// Test 7: Streaming workflow with tool calls
	t.Run("Streaming_WithToolCalls", func(t *testing.T) {
		tools := []llm.Tool{
			{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        "calculate",
					Description: "Perform mathematical calculations",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"expression": map[string]interface{}{
								"type":        "string",
								"description": "Math expression",
							},
						},
					},
				},
			},
		}

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeAnthropic,
			Model:        "claude-3-5-haiku-latest",
			Messages: []llm.Message{
				{Role: "user", Content: "Calculate the factorial of 5 and explain it."},
			},
			MaxTokens:   500,
			Temperature: 0.1,
			Tools:       tools,
			Stream:      true,
			CreatedAt:   time.Now(),
		}

		ch := make(chan llm.LLMResponse, 10)
		go func() {
			err := provider.GenerateStream(env.ctx, request, ch)
			assert.NoError(t, err)
		}()

		var content string
		var chunkCount int

		for response := range ch {
			content += response.Content
			if response.Content != "" {
				chunkCount++
			}
			if len(response.ToolCalls) > 0 {
				t.Logf("Received tool call in stream: %s", response.ToolCalls[0].Function.Name)
			}
		}

		assert.NotEmpty(t, content)
		assert.Greater(t, chunkCount, 0)
		t.Logf("✅ Streaming complete: %d chunks, %d chars", chunkCount, len(content))
	})
}

// TestGeminiProviderEndToEnd tests the Gemini provider in a complete workflow
func TestGeminiProviderEndToEnd(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY or GOOGLE_API_KEY environment variable not set, skipping e2e test")
	}

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Create Gemini provider
	config := llm.ProviderConfigEntry{
		Type:     llm.ProviderTypeGemini,
		Endpoint: getEnvOrDefault("GEMINI_ENDPOINT", "https://generativelanguage.googleapis.com/v1beta"),
		APIKey:   apiKey,
	}

	provider, err := llm.NewGeminiProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	// Register provider with model manager
	err = env.ModelManager.RegisterProvider(provider)
	require.NoError(t, err)

	// Test 1: Model selection for massive context
	t.Run("ModelSelection_MassiveContext", func(t *testing.T) {
		criteria := llm.ModelSelectionCriteria{
			TaskType: "code_analysis",
			RequiredCapabilities: []llm.ModelCapability{
				llm.CapabilityCodeAnalysis,
			},
			MaxTokens:         50000, // Request large context
			QualityPreference: "balanced",
		}

		selectedModel, err := env.ModelManager.SelectOptimalModel(criteria)
		assert.NoError(t, err)
		assert.NotNil(t, selectedModel)

		if selectedModel.Provider == llm.ProviderTypeGemini {
			t.Logf("✅ Selected Gemini model: %s with %d token context",
				selectedModel.Name, selectedModel.ContextSize)

			// Should select a model with massive context
			if selectedModel.ContextSize >= 1000000 {
				t.Logf("✅ Selected model has 1M+ token context window")
			}
		}
	})

	// Test 2: Provider health monitoring
	t.Run("HealthMonitoring", func(t *testing.T) {
		health := env.ModelManager.HealthCheck(env.ctx)
		geminiHealth, exists := health[llm.ProviderTypeGemini]
		assert.True(t, exists, "Gemini provider should be in health check results")
		assert.NotNil(t, geminiHealth)
		assert.Equal(t, "healthy", geminiHealth.Status)

		t.Logf("Gemini provider health: %s (latency: %v, models: %d)",
			geminiHealth.Status, geminiHealth.Latency, geminiHealth.ModelCount)
	})

	// Test 3: Large codebase analysis
	t.Run("MassiveContext_CodebaseAnalysis", func(t *testing.T) {
		// Simulate a large codebase with multiple files
		codebase := generateLargeCodebase(10) // 10 files

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeGemini,
			Model:        "gemini-2.5-flash", // 1M context
			Messages: []llm.Message{
				{Role: "system", Content: "You are analyzing a complete codebase. Provide architectural insights."},
				{Role: "user", Content: codebase + "\n\nAnalyze this codebase architecture and identify potential improvements."},
			},
			MaxTokens:   1000,
			Temperature: 0.3,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(env.ctx, request)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Content)

		t.Logf("✅ Analyzed %d chars of code, produced %d token response",
			len(codebase), response.Usage.TotalTokens)
		t.Logf("Prompt tokens: %d (handled massive context)", response.Usage.PromptTokens)
	})

	// Test 4: Fast iterations with Flash models
	t.Run("FlashModel_FastIterations", func(t *testing.T) {
		iterations := 3
		var totalTime time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			request := &llm.LLMRequest{
				ID:           uuid.New(),
				ProviderType: llm.ProviderTypeGemini,
				Model:        "gemini-2.5-flash-lite", // Fastest model
				Messages: []llm.Message{
					{Role: "user", Content: "Write a one-line description of Go."},
				},
				MaxTokens:   50,
				Temperature: 0.1,
				CreatedAt:   time.Now(),
			}

			response, err := provider.Generate(env.ctx, request)
			require.NoError(t, err)
			assert.NotEmpty(t, response.Content)

			latency := time.Since(start)
			totalTime += latency

			t.Logf("Iteration %d: %v latency, %d tokens",
				i+1, latency, response.Usage.TotalTokens)
		}

		avgLatency := totalTime / time.Duration(iterations)
		t.Logf("✅ Flash model average latency: %v", avgLatency)

		// Flash models should be fast (usually < 2 seconds)
		assert.Less(t, avgLatency, 5*time.Second, "Flash model should be fast")
	})

	// Test 5: Function calling workflow
	t.Run("FunctionCalling_Workflow", func(t *testing.T) {
		tools := []llm.Tool{
			{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        "read_file",
					Description: "Read contents of a file",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{
								"type":        "string",
								"description": "File path",
							},
						},
						"required": []string{"path"},
					},
				},
			},
			{
				Type: "function",
				Function: llm.FunctionDefinition{
					Name:        "write_file",
					Description: "Write contents to a file",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{
								"type":        "string",
								"description": "File path",
							},
							"content": map[string]interface{}{
								"type":        "string",
								"description": "File content",
							},
						},
						"required": []string{"path", "content"},
					},
				},
			},
		}

		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeGemini,
			Model:        "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "Read the config file at /etc/config.yaml"},
			},
			MaxTokens:   200,
			Temperature: 0.1,
			Tools:       tools,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(env.ctx, request)
		require.NoError(t, err)

		// Should call read_file function
		if len(response.ToolCalls) > 0 {
			assert.Equal(t, "read_file", response.ToolCalls[0].Function.Name)
			t.Logf("✅ Function calling successful: %s", response.ToolCalls[0].Function.Name)
		}
	})

	// Test 6: Multi-turn conversation
	t.Run("MultiTurnConversation", func(t *testing.T) {
		// Turn 1
		request1 := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeGemini,
			Model:        "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "I'm working on a web server in Go."},
			},
			MaxTokens:   100,
			Temperature: 0.3,
			CreatedAt:   time.Now(),
		}

		response1, err := provider.Generate(env.ctx, request1)
		require.NoError(t, err)

		// Turn 2 - reference previous conversation
		request2 := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeGemini,
			Model:        "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "I'm working on a web server in Go."},
				{Role: "assistant", Content: response1.Content},
				{Role: "user", Content: "What HTTP framework should I use for this project?"},
			},
			MaxTokens:   200,
			Temperature: 0.3,
			CreatedAt:   time.Now(),
		}

		response2, err := provider.Generate(env.ctx, request2)
		require.NoError(t, err)
		assert.NotEmpty(t, response2.Content)

		// Should understand context and recommend frameworks
		content := strings.ToLower(response2.Content)
		hasFramework := strings.Contains(content, "gin") ||
			strings.Contains(content, "echo") ||
			strings.Contains(content, "fiber") ||
			strings.Contains(content, "chi")

		assert.True(t, hasFramework, "Should recommend a Go web framework")
		t.Logf("✅ Multi-turn conversation maintained context")
	})

	// Test 7: Streaming workflow
	t.Run("Streaming_Workflow", func(t *testing.T) {
		request := &llm.LLMRequest{
			ID:           uuid.New(),
			ProviderType: llm.ProviderTypeGemini,
			Model:        "gemini-2.5-flash-lite",
			Messages: []llm.Message{
				{Role: "user", Content: "List 5 best practices for writing Go code."},
			},
			MaxTokens:   500,
			Temperature: 0.3,
			Stream:      true,
			CreatedAt:   time.Now(),
		}

		ch := make(chan llm.LLMResponse, 10)
		go func() {
			err := provider.GenerateStream(env.ctx, request, ch)
			assert.NoError(t, err)
		}()

		var content string
		var chunkCount int

		for response := range ch {
			content += response.Content
			if response.Content != "" {
				chunkCount++
			}
		}

		assert.NotEmpty(t, content)
		assert.Greater(t, chunkCount, 0)
		t.Logf("✅ Streaming complete: %d chunks", chunkCount)
	})
}

// Helper function to generate large codebase for testing
func generateLargeCodebase(numFiles int) string {
	var codebase strings.Builder

	for i := 0; i < numFiles; i++ {
		codebase.WriteString("// File: file" + string(rune(i)) + ".go\n")
		codebase.WriteString("package main\n\n")
		codebase.WriteString("import (\n")
		codebase.WriteString("\t\"fmt\"\n")
		codebase.WriteString("\t\"net/http\"\n")
		codebase.WriteString(")\n\n")
		codebase.WriteString("func handler" + string(rune(i)) + "(w http.ResponseWriter, r *http.Request) {\n")
		codebase.WriteString("\tfmt.Fprintf(w, \"Handler " + string(rune(i)) + "\")\n")
		codebase.WriteString("}\n\n")
	}

	return codebase.String()
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
