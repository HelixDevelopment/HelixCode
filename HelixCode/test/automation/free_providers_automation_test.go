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

// TestAllFreeProvidersAutomation tests all free AI providers with real API calls
func TestAllFreeProvidersAutomation(t *testing.T) {
	providers := []struct {
		name       string
		configFunc func() (llm.ProviderConfigEntry, bool)
		model      string
	}{
		{
			name: "XAI",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				apiKey := os.Getenv("XAI_API_KEY")
				if apiKey == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeXAI,
					APIKey: apiKey,
				}, true
			},
			model: "grok-3-mini-fast-beta",
		},
		{
			name: "OpenRouter",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				apiKey := os.Getenv("OPENROUTER_API_KEY")
				if apiKey == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeOpenRouter,
					APIKey: apiKey,
				}, true
			},
			model: "deepseek-r1-free",
		},
		{
			name: "GitHub Copilot",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				token := os.Getenv("GITHUB_TOKEN")
				if token == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeCopilot,
					APIKey: token,
				}, true
			},
			model: "gpt-4o-mini",
		},
		{
			name: "Qwen",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				apiKey := os.Getenv("QWEN_API_KEY")
				if apiKey == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeQwen,
					APIKey: apiKey,
				}, true
			},
			model: "qwen-turbo",
		},
	}

	testedProviders := 0
	successfulProviders := 0

	for _, provider := range providers {
		t.Run(fmt.Sprintf("Provider_%s", provider.name), func(t *testing.T) {
			config, available := provider.configFunc()
			if !available {
				t.Skipf("%s API key not available, skipping tests", provider.name)
				return
			}

			testedProviders++

			// Test provider creation
			t.Run("Creation", func(t *testing.T) {
				p, err := llm.NewProviderFactory().CreateProvider(config)
				require.NoError(t, err)
				assert.NotNil(t, p)
				assert.Equal(t, config.Type, p.GetType())
				assert.Contains(t, p.GetName(), provider.name)
			})

			p, err := llm.NewProviderFactory().CreateProvider(config)
			require.NoError(t, err)

			// Test provider capabilities
			t.Run("Capabilities", func(t *testing.T) {
				capabilities := p.GetCapabilities()
				assert.NotEmpty(t, capabilities, "Should have capabilities")
				assert.Contains(t, capabilities, llm.CapabilityTextGeneration)
			})

			// Test model listing
			t.Run("Models", func(t *testing.T) {
				models := p.GetModels()
				assert.NotEmpty(t, models, "Should have available models")

				// Check if our test model exists
				modelNames := make(map[string]bool)
				for _, model := range models {
					modelNames[model.Name] = true
				}
				assert.True(t, modelNames[provider.model], "Test model %s should be available", provider.model)
			})

			// Test health check
			t.Run("Health", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				health, err := p.GetHealth(ctx)
				assert.NoError(t, err)
				assert.NotNil(t, health)
				assert.NotEmpty(t, health.Status)
			})

			// Test basic generation
			t.Run("BasicGeneration", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				request := &llm.LLMRequest{
					ID:           uuid.New(),
					ProviderType: config.Type,
					Model:        provider.model,
					Messages: []llm.Message{
						{Role: "user", Content: fmt.Sprintf("Hello from %s! Respond with exactly 'Hello from %s!'", provider.name, provider.name)},
					},
					MaxTokens:   50,
					Temperature: 0.1,
					CreatedAt:   time.Now(),
				}

				response, err := p.Generate(ctx, request)
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.Content)
				assert.Greater(t, response.Usage.TotalTokens, 0)
			})

			// Test streaming
			t.Run("Streaming", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				request := &llm.LLMRequest{
					ID:           uuid.New(),
					ProviderType: config.Type,
					Model:        provider.model,
					Messages: []llm.Message{
						{Role: "user", Content: "Count from 1 to 3 slowly."},
					},
					MaxTokens:   100,
					Temperature: 0.1,
					Stream:      true,
					CreatedAt:   time.Now(),
				}

				ch := make(chan llm.LLMResponse, 50)
				errCh := make(chan error, 1)

				go func() {
					errCh <- p.GenerateStream(ctx, request, ch)
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
			})

			// Test code generation
			t.Run("CodeGeneration", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cancel()

				request := &llm.LLMRequest{
					ID:           uuid.New(),
					ProviderType: config.Type,
					Model:        provider.model,
					Messages: []llm.Message{
						{Role: "user", Content: "Write a simple function in any language that adds two numbers. Include a comment."},
					},
					MaxTokens:   200,
					Temperature: 0.3,
					CreatedAt:   time.Now(),
				}

				response, err := p.Generate(ctx, request)
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.Content)
				assert.Greater(t, response.Usage.TotalTokens, 0)
			})

			// Test provider cleanup
			t.Run("Cleanup", func(t *testing.T) {
				err := p.Close()
				assert.NoError(t, err)
			})

			successfulProviders++
			t.Logf("✅ %s provider tests completed successfully", provider.name)
		})
	}

	t.Logf("🎯 Free Providers Automation Test Results:")
	t.Logf("   Providers tested: %d", testedProviders)
	t.Logf("   Successful: %d", successfulProviders)
	t.Logf("   Failed: %d", testedProviders-successfulProviders)

	if testedProviders == 0 {
		t.Skip("No free provider API keys available for testing")
	}

	assert.Greater(t, successfulProviders, 0, "At least one free provider should work")
}

// TestFreeProvidersLoadTest performs load testing on all available free providers
func TestFreeProvidersLoadTest(t *testing.T) {
	providers := []struct {
		name       string
		configFunc func() (llm.ProviderConfigEntry, bool)
		model      string
	}{
		{
			name: "XAI",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				apiKey := os.Getenv("XAI_API_KEY")
				if apiKey == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeXAI,
					APIKey: apiKey,
				}, true
			},
			model: "grok-3-mini-fast-beta",
		},
		{
			name: "OpenRouter",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				apiKey := os.Getenv("OPENROUTER_API_KEY")
				if apiKey == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeOpenRouter,
					APIKey: apiKey,
				}, true
			},
			model: "deepseek-r1-free",
		},
	}

	for _, provider := range providers {
		t.Run(fmt.Sprintf("LoadTest_%s", provider.name), func(t *testing.T) {
			config, available := provider.configFunc()
			if !available {
				t.Skipf("%s API key not available, skipping load test", provider.name)
				return
			}

			p, err := llm.NewProviderFactory().CreateProvider(config)
			require.NoError(t, err)
			defer p.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer cancel()

			numRequests := 5 // Reduced for free providers
			results := make(chan time.Duration, numRequests)
			errors := make(chan error, numRequests)

			startTime := time.Now()

			// Launch concurrent requests
			for i := 0; i < numRequests; i++ {
				go func(requestNum int) {
					reqStart := time.Now()

					request := &llm.LLMRequest{
						ID:           uuid.New(),
						ProviderType: config.Type,
						Model:        provider.model,
						Messages: []llm.Message{
							{Role: "user", Content: fmt.Sprintf("Say 'test %d'", requestNum)},
						},
						MaxTokens:   20,
						Temperature: 0.1,
						CreatedAt:   time.Now(),
					}

					_, err := p.Generate(ctx, request)
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

			if len(durations) > 0 {
				var totalDuration time.Duration
				for _, d := range durations {
					totalDuration += d
				}
				avgDuration := totalDuration / time.Duration(len(durations))

				t.Logf("%s Load test results:", provider.name)
				t.Logf("  Requests: %d successful, %d failed", len(durations), errorCount)
				t.Logf("  Total time: %v", totalTime)
				t.Logf("  Average response time: %v", avgDuration)
				t.Logf("  Requests per second: %.2f", float64(len(durations))/totalTime.Seconds())

				assert.Greater(t, len(durations), 0, "Should have at least one successful request")
			}
		})
	}
}

// TestFreeProvidersFeatureComparison compares features across free providers
func TestFreeProvidersFeatureComparison(t *testing.T) {
	providers := []struct {
		name       string
		configFunc func() (llm.ProviderConfigEntry, bool)
	}{
		{
			name: "XAI",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				apiKey := os.Getenv("XAI_API_KEY")
				if apiKey == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeXAI,
					APIKey: apiKey,
				}, true
			},
		},
		{
			name: "OpenRouter",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				apiKey := os.Getenv("OPENROUTER_API_KEY")
				if apiKey == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeOpenRouter,
					APIKey: apiKey,
				}, true
			},
		},
		{
			name: "GitHub Copilot",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				token := os.Getenv("GITHUB_TOKEN")
				if token == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeCopilot,
					APIKey: token,
				}, true
			},
		},
		{
			name: "Qwen",
			configFunc: func() (llm.ProviderConfigEntry, bool) {
				apiKey := os.Getenv("QWEN_API_KEY")
				if apiKey == "" {
					return llm.ProviderConfigEntry{}, false
				}
				return llm.ProviderConfigEntry{
					Type:   llm.ProviderTypeQwen,
					APIKey: apiKey,
				}, true
			},
		},
	}

	t.Logf("🔍 Free AI Providers Feature Comparison")
	t.Logf("=====================================")

	for _, provider := range providers {
		t.Run(fmt.Sprintf("Features_%s", provider.name), func(t *testing.T) {
			config, available := provider.configFunc()
			if !available {
				t.Skipf("%s not available", provider.name)
				return
			}

			p, err := llm.NewProviderFactory().CreateProvider(config)
			require.NoError(t, err)
			defer p.Close()

			models := p.GetModels()
			capabilities := p.GetCapabilities()

			t.Logf("\n📊 %s Provider:", provider.name)
			t.Logf("   Models available: %d", len(models))
			t.Logf("   Capabilities: %v", capabilities)

			// Test a simple prompt to verify functionality
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if len(models) > 0 {
				request := &llm.LLMRequest{
					ID:           uuid.New(),
					ProviderType: config.Type,
					Model:        models[0].Name,
					Messages: []llm.Message{
						{Role: "user", Content: "Say 'OK'"},
					},
					MaxTokens:   10,
					Temperature: 0.1,
					CreatedAt:   time.Now(),
				}

				response, err := p.Generate(ctx, request)
				if err != nil {
					t.Logf("   Status: ❌ Error - %v", err)
				} else {
					t.Logf("   Status: ✅ Working - Response time: %v", response.ProcessingTime)
					t.Logf("   Tokens used: %d", response.Usage.TotalTokens)
				}
			}
		})
	}
}
