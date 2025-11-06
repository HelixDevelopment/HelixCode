package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultCacheConfig tests the default cache configuration
func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	assert.True(t, config.Enabled, "Cache should be enabled by default")
	assert.Equal(t, CacheStrategyTools, config.Strategy, "Default strategy should be Tools")
	assert.Equal(t, 1024, config.MinTokensForCache, "Default min tokens should be 1024")
	assert.Equal(t, 300, config.CacheTTL, "Default TTL should be 300 seconds")
}

// TestCacheControl tests basic CacheControl structure
func TestCacheControl(t *testing.T) {
	tests := []struct {
		name     string
		cacheCtl *CacheControl
		wantType string
	}{
		{
			name:     "ephemeral cache control",
			cacheCtl: &CacheControl{Type: "ephemeral"},
			wantType: "ephemeral",
		},
		{
			name:     "nil cache control",
			cacheCtl: nil,
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cacheCtl != nil {
				assert.Equal(t, tt.wantType, tt.cacheCtl.Type)
			} else {
				assert.Nil(t, tt.cacheCtl)
			}
		})
	}
}

// TestConvertToCacheable tests converting regular messages to cacheable messages
func TestConvertToCacheable(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		want     int
	}{
		{
			name:     "empty messages",
			messages: []Message{},
			want:     0,
		},
		{
			name: "single message",
			messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			want: 1,
		},
		{
			name: "multiple messages",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToCacheable(tt.messages)

			assert.Len(t, result, tt.want)
			for i, msg := range result {
				assert.Equal(t, tt.messages[i].Role, msg.Role)
				assert.Equal(t, tt.messages[i].Content, msg.Content)
				assert.Nil(t, msg.CacheControl, "Cache control should be nil for convertToCacheable")
			}
		})
	}
}

// TestCacheStrategyNone tests that no caching is applied with CacheStrategyNone
func TestCacheStrategyNone(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}

	config := CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyNone,
	}

	result := ApplyCacheControl(messages, nil, config)

	assert.Len(t, result, 2)
	for _, msg := range result {
		assert.Nil(t, msg.CacheControl, "CacheStrategyNone should not apply cache control")
	}
}

// TestCacheStrategyNoneDisabled tests that no caching is applied when disabled
func TestCacheStrategyNoneDisabled(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}

	config := CacheConfig{
		Enabled:  false,
		Strategy: CacheStrategyTools,
	}

	result := ApplyCacheControl(messages, nil, config)

	assert.Len(t, result, 2)
	for _, msg := range result {
		assert.Nil(t, msg.CacheControl, "Disabled caching should not apply cache control")
	}
}

// TestCacheStrategySystem tests system-only caching strategy
func TestCacheStrategySystem(t *testing.T) {
	tests := []struct {
		name          string
		messages      []Message
		tools         []Tool
		expectCached  []int // indices of messages that should be cached
	}{
		{
			name: "system message cached",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi"},
			},
			tools:        nil,
			expectCached: []int{0},
		},
		{
			name: "no system message",
			messages: []Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi"},
			},
			tools:        nil,
			expectCached: []int{},
		},
		{
			name: "system message not first",
			messages: []Message{
				{Role: "user", Content: "Hello"},
				{Role: "system", Content: "You are a helpful assistant"},
			},
			tools:        nil,
			expectCached: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CacheConfig{
				Enabled:  true,
				Strategy: CacheStrategySystem,
			}

			result := ApplyCacheControl(tt.messages, tt.tools, config)

			assert.Len(t, result, len(tt.messages))

			for i, msg := range result {
				shouldBeCached := false
				for _, idx := range tt.expectCached {
					if i == idx {
						shouldBeCached = true
						break
					}
				}

				if shouldBeCached {
					require.NotNil(t, msg.CacheControl, "Message at index %d should have cache control", i)
					assert.Equal(t, "ephemeral", msg.CacheControl.Type)
				} else {
					assert.Nil(t, msg.CacheControl, "Message at index %d should not have cache control", i)
				}
			}
		})
	}
}

// TestCacheStrategyTools tests system + tools caching strategy
func TestCacheStrategyTools(t *testing.T) {
	tests := []struct {
		name          string
		messages      []Message
		tools         []Tool
		expectCached  []int
	}{
		{
			name: "system cached with tools",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
			},
			tools: []Tool{
				{Type: "function", Function: FunctionDefinition{Name: "test"}},
			},
			expectCached: []int{0},
		},
		{
			name: "system not cached without tools",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
			},
			tools:        nil,
			expectCached: []int{},
		},
		{
			name: "system not cached with empty tools",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
			},
			tools:        []Tool{},
			expectCached: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CacheConfig{
				Enabled:  true,
				Strategy: CacheStrategyTools,
			}

			result := ApplyCacheControl(tt.messages, tt.tools, config)

			for i, msg := range result {
				shouldBeCached := false
				for _, idx := range tt.expectCached {
					if i == idx {
						shouldBeCached = true
						break
					}
				}

				if shouldBeCached {
					require.NotNil(t, msg.CacheControl, "Message at index %d should have cache control", i)
					assert.Equal(t, "ephemeral", msg.CacheControl.Type)
				} else {
					assert.Nil(t, msg.CacheControl, "Message at index %d should not have cache control", i)
				}
			}
		})
	}
}

// TestCacheStrategyContext tests system + context caching strategy
func TestCacheStrategyContext(t *testing.T) {
	tests := []struct {
		name          string
		messages      []Message
		expectCached  []int
	}{
		{
			name: "system and recent user messages cached",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Message 1"},
				{Role: "assistant", Content: "Response 1"},
				{Role: "user", Content: "Message 2"},
				{Role: "assistant", Content: "Response 2"},
				{Role: "user", Content: "Message 3"},
			},
			expectCached: []int{0, 3, 5}, // system + last 2 user messages (indices 3, 5)
		},
		{
			name: "only system cached with few messages",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
			},
			expectCached: []int{0, 1}, // system + user (within last 3)
		},
		{
			name: "no system message",
			messages: []Message{
				{Role: "user", Content: "Message 1"},
				{Role: "user", Content: "Message 2"},
				{Role: "user", Content: "Message 3"},
			},
			expectCached: []int{0, 1, 2}, // all user messages (within last 3)
		},
		{
			name: "assistant messages not cached",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi"},
				{Role: "user", Content: "How are you?"},
				{Role: "assistant", Content: "I'm good"},
			},
			expectCached: []int{0, 3}, // system + last user message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CacheConfig{
				Enabled:  true,
				Strategy: CacheStrategyContext,
			}

			result := ApplyCacheControl(tt.messages, nil, config)

			for i, msg := range result {
				shouldBeCached := false
				for _, idx := range tt.expectCached {
					if i == idx {
						shouldBeCached = true
						break
					}
				}

				if shouldBeCached {
					require.NotNil(t, msg.CacheControl, "Message at index %d should have cache control", i)
					assert.Equal(t, "ephemeral", msg.CacheControl.Type)
				} else {
					assert.Nil(t, msg.CacheControl, "Message at index %d should not have cache control", i)
				}
			}
		})
	}
}

// TestCacheStrategyAggressive tests aggressive caching strategy
func TestCacheStrategyAggressive(t *testing.T) {
	tests := []struct {
		name          string
		messages      []Message
		expectCached  []int
	}{
		{
			name: "system and all user messages cached",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Message 1"},
				{Role: "assistant", Content: "Response 1"},
				{Role: "user", Content: "Message 2"},
				{Role: "assistant", Content: "Response 2"},
			},
			expectCached: []int{0, 1, 3}, // system + all user messages
		},
		{
			name: "only user messages",
			messages: []Message{
				{Role: "user", Content: "Message 1"},
				{Role: "user", Content: "Message 2"},
			},
			expectCached: []int{0, 1},
		},
		{
			name: "no cacheable messages",
			messages: []Message{
				{Role: "assistant", Content: "Response 1"},
				{Role: "assistant", Content: "Response 2"},
			},
			expectCached: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CacheConfig{
				Enabled:  true,
				Strategy: CacheStrategyAggressive,
			}

			result := ApplyCacheControl(tt.messages, nil, config)

			for i, msg := range result {
				shouldBeCached := false
				for _, idx := range tt.expectCached {
					if i == idx {
						shouldBeCached = true
						break
					}
				}

				if shouldBeCached {
					require.NotNil(t, msg.CacheControl, "Message at index %d should have cache control", i)
					assert.Equal(t, "ephemeral", msg.CacheControl.Type)
				} else {
					assert.Nil(t, msg.CacheControl, "Message at index %d should not have cache control", i)
				}
			}
		})
	}
}

// TestCalculateCacheSavings tests cache savings calculations
func TestCalculateCacheSavings(t *testing.T) {
	tests := []struct {
		name              string
		stats             CacheStats
		inputCostPer1K    float64
		cacheCostPer1K    float64
		wantSavings       float64
		wantSavingsPercent float64
		wantTokensCached  int
		wantTokensRead    int
	}{
		{
			name: "basic cache savings",
			stats: CacheStats{
				CacheCreationInputTokens: 1000,
				CacheReadInputTokens:     2000,
				InputTokens:              2500,
				OutputTokens:             500,
			},
			inputCostPer1K:     0.01,
			cacheCostPer1K:     0.001,
			wantSavings:        0.018, // (3500 * 0.01 / 1000) - ((1000 * 0.01 + 2000 * 0.001 + 500 * 0.01) / 1000)
			wantSavingsPercent: 51.428571428571423,
			wantTokensCached:   1000,
			wantTokensRead:     2000,
		},
		{
			name: "no cache usage",
			stats: CacheStats{
				CacheCreationInputTokens: 0,
				CacheReadInputTokens:     0,
				InputTokens:              1000,
				OutputTokens:             500,
			},
			inputCostPer1K:     0.01,
			cacheCostPer1K:     0.001,
			wantSavings:        0.0,
			wantSavingsPercent: 0.0,
			wantTokensCached:   0,
			wantTokensRead:     0,
		},
		{
			name: "high cache read ratio",
			stats: CacheStats{
				CacheCreationInputTokens: 500,
				CacheReadInputTokens:     10000,
				InputTokens:              10500,
				OutputTokens:             1000,
			},
			inputCostPer1K:     0.01,
			cacheCostPer1K:     0.001,
			wantSavings:        0.09, // Significant savings from cache reads
			wantSavingsPercent: 81.81818181818181,
			wantTokensCached:   500,
			wantTokensRead:     10000,
		},
		{
			name: "zero costs",
			stats: CacheStats{
				CacheCreationInputTokens: 1000,
				CacheReadInputTokens:     2000,
				InputTokens:              3000,
				OutputTokens:             500,
			},
			inputCostPer1K:     0.0,
			cacheCostPer1K:     0.0,
			wantSavings:        0.0,
			wantSavingsPercent: 0.0,
			wantTokensCached:   1000,
			wantTokensRead:     2000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCacheSavings(tt.stats, tt.inputCostPer1K, tt.cacheCostPer1K)

			assert.InDelta(t, tt.wantSavings, result.Savings, 0.001, "Savings mismatch")
			assert.InDelta(t, tt.wantSavingsPercent, result.SavingsPercent, 0.01, "Savings percent mismatch")
			assert.Equal(t, tt.wantTokensCached, result.TokensCached)
			assert.Equal(t, tt.wantTokensRead, result.TokensRead)
			assert.GreaterOrEqual(t, result.CostWithoutCache, result.CostWithCache, "Cost without cache should be >= cost with cache")
		})
	}
}

// TestCalculateCacheSavingsEdgeCases tests edge cases for cache savings
func TestCalculateCacheSavingsEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		stats          CacheStats
		inputCostPer1K float64
		cacheCostPer1K float64
	}{
		{
			name: "negative tokens (should still calculate)",
			stats: CacheStats{
				CacheCreationInputTokens: -100,
				CacheReadInputTokens:     0,
				InputTokens:              1000,
				OutputTokens:             500,
			},
			inputCostPer1K: 0.01,
			cacheCostPer1K: 0.001,
		},
		{
			name: "very large token counts",
			stats: CacheStats{
				CacheCreationInputTokens: 1000000,
				CacheReadInputTokens:     5000000,
				InputTokens:              6000000,
				OutputTokens:             100000,
			},
			inputCostPer1K: 0.01,
			cacheCostPer1K: 0.001,
		},
		{
			name: "cache read tokens exceed input tokens",
			stats: CacheStats{
				CacheCreationInputTokens: 1000,
				CacheReadInputTokens:     3000,
				InputTokens:              2000,
				OutputTokens:             500,
			},
			inputCostPer1K: 0.01,
			cacheCostPer1K: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := CalculateCacheSavings(tt.stats, tt.inputCostPer1K, tt.cacheCostPer1K)
			assert.NotNil(t, result)
		})
	}
}

// TestCacheMetricsUpdateMetrics tests updating cache metrics
func TestCacheMetricsUpdateMetrics(t *testing.T) {
	tests := []struct {
		name                   string
		initialMetrics         CacheMetrics
		updates                []struct {
			stats   CacheStats
			savings CacheSavings
		}
		wantTotalRequests     int
		wantRequestsWithCache int
		wantCacheHitRate      float64
	}{
		{
			name:           "single update with cache",
			initialMetrics: CacheMetrics{},
			updates: []struct {
				stats   CacheStats
				savings CacheSavings
			}{
				{
					stats: CacheStats{
						CacheCreationInputTokens: 1000,
						CacheReadInputTokens:     2000,
						InputTokens:              3000,
						OutputTokens:             500,
					},
					savings: CacheSavings{
						Savings:        0.02,
						SavingsPercent: 50.0,
						TokensCached:   1000,
						TokensRead:     2000,
					},
				},
			},
			wantTotalRequests:     1,
			wantRequestsWithCache: 1,
			wantCacheHitRate:      1.0,
		},
		{
			name:           "multiple updates",
			initialMetrics: CacheMetrics{},
			updates: []struct {
				stats   CacheStats
				savings CacheSavings
			}{
				{
					stats: CacheStats{
						CacheCreationInputTokens: 1000,
						CacheReadInputTokens:     0,
						InputTokens:              1000,
					},
					savings: CacheSavings{
						SavingsPercent: 0.0,
					},
				},
				{
					stats: CacheStats{
						CacheCreationInputTokens: 0,
						CacheReadInputTokens:     2000,
						InputTokens:              2000,
					},
					savings: CacheSavings{
						SavingsPercent: 80.0,
					},
				},
				{
					stats: CacheStats{
						CacheCreationInputTokens: 0,
						CacheReadInputTokens:     0,
						InputTokens:              1000,
					},
					savings: CacheSavings{
						SavingsPercent: 0.0,
					},
				},
			},
			wantTotalRequests:     3,
			wantRequestsWithCache: 2,
			wantCacheHitRate:      0.6666666666666666,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := tt.initialMetrics

			for _, update := range tt.updates {
				metrics.UpdateMetrics(update.stats, update.savings)
			}

			assert.Equal(t, tt.wantTotalRequests, metrics.TotalRequests)
			assert.Equal(t, tt.wantRequestsWithCache, metrics.RequestsWithCache)
			assert.InDelta(t, tt.wantCacheHitRate, metrics.CacheHitRate, 0.01)
		})
	}
}

// TestCacheMetricsCacheHitRate tests cache hit rate calculation
func TestCacheMetricsCacheHitRate(t *testing.T) {
	tests := []struct {
		name             string
		totalRequests    int
		cacheRequests    int
		wantCacheHitRate float64
	}{
		{
			name:             "100% hit rate",
			totalRequests:    10,
			cacheRequests:    10,
			wantCacheHitRate: 1.0,
		},
		{
			name:             "50% hit rate",
			totalRequests:    10,
			cacheRequests:    5,
			wantCacheHitRate: 0.5,
		},
		{
			name:             "0% hit rate",
			totalRequests:    10,
			cacheRequests:    0,
			wantCacheHitRate: 0.0,
		},
		{
			name:             "no requests",
			totalRequests:    0,
			cacheRequests:    0,
			wantCacheHitRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := CacheMetrics{}

			for i := 0; i < tt.totalRequests; i++ {
				stats := CacheStats{}
				if i < tt.cacheRequests {
					stats.CacheCreationInputTokens = 100
				}
				savings := CacheSavings{}
				metrics.UpdateMetrics(stats, savings)
			}

			assert.InDelta(t, tt.wantCacheHitRate, metrics.CacheHitRate, 0.01)
		})
	}
}

// TestCacheMetricsAverageSavings tests average savings calculation
func TestCacheMetricsAverageSavings(t *testing.T) {
	metrics := CacheMetrics{}

	// First update: 50% savings
	metrics.UpdateMetrics(
		CacheStats{CacheCreationInputTokens: 1000},
		CacheSavings{SavingsPercent: 50.0},
	)
	assert.InDelta(t, 50.0, metrics.AverageSavingsPercent, 0.01)

	// Second update: 80% savings (average should be 65%)
	metrics.UpdateMetrics(
		CacheStats{CacheCreationInputTokens: 1000},
		CacheSavings{SavingsPercent: 80.0},
	)
	assert.InDelta(t, 65.0, metrics.AverageSavingsPercent, 0.01)

	// Third update: 20% savings (average should be 50%)
	metrics.UpdateMetrics(
		CacheStats{CacheCreationInputTokens: 1000},
		CacheSavings{SavingsPercent: 20.0},
	)
	assert.InDelta(t, 50.0, metrics.AverageSavingsPercent, 0.01)
}

// TestCacheMetricsTotals tests total accumulation
func TestCacheMetricsTotals(t *testing.T) {
	metrics := CacheMetrics{}

	updates := []struct {
		stats   CacheStats
		savings CacheSavings
	}{
		{
			stats: CacheStats{
				CacheCreationInputTokens: 1000,
				CacheReadInputTokens:     2000,
			},
			savings: CacheSavings{Savings: 0.01},
		},
		{
			stats: CacheStats{
				CacheCreationInputTokens: 500,
				CacheReadInputTokens:     1500,
			},
			savings: CacheSavings{Savings: 0.005},
		},
		{
			stats: CacheStats{
				CacheCreationInputTokens: 2000,
				CacheReadInputTokens:     5000,
			},
			savings: CacheSavings{Savings: 0.02},
		},
	}

	expectedTokensCached := 0
	expectedTokensRead := 0
	expectedSavings := 0.0

	for _, update := range updates {
		metrics.UpdateMetrics(update.stats, update.savings)
		expectedTokensCached += update.stats.CacheCreationInputTokens
		expectedTokensRead += update.stats.CacheReadInputTokens
		expectedSavings += update.savings.Savings
	}

	assert.Equal(t, expectedTokensCached, metrics.TotalTokensCached)
	assert.Equal(t, expectedTokensRead, metrics.TotalTokensRead)
	assert.InDelta(t, expectedSavings, metrics.TotalSavings, 0.001)
}

// TestIntegrationFullCachingWorkflow tests a complete caching workflow
func TestIntegrationFullCachingWorkflow(t *testing.T) {
	// Setup
	config := CacheConfig{
		Enabled:           true,
		Strategy:          CacheStrategyTools,
		MinTokensForCache: 1024,
		CacheTTL:          300,
	}

	messages := []Message{
		{Role: "system", Content: "You are a helpful AI assistant."},
		{Role: "user", Content: "What is Go?"},
		{Role: "assistant", Content: "Go is a programming language."},
		{Role: "user", Content: "Tell me more about it."},
	}

	tools := []Tool{
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "search",
				Description: "Search for information",
			},
		},
	}

	// Apply cache control
	cacheableMessages := ApplyCacheControl(messages, tools, config)

	// Verify system message is cached with tools
	require.NotNil(t, cacheableMessages[0].CacheControl)
	assert.Equal(t, "ephemeral", cacheableMessages[0].CacheControl.Type)

	// Simulate cache statistics
	stats := CacheStats{
		CacheCreationInputTokens: 2000,
		CacheReadInputTokens:     8000,
		InputTokens:              10000,
		OutputTokens:             2000,
	}

	// Calculate savings
	savings := CalculateCacheSavings(stats, 0.01, 0.001)
	assert.Greater(t, savings.Savings, 0.0)
	assert.Greater(t, savings.SavingsPercent, 0.0)

	// Update metrics
	metrics := CacheMetrics{}
	metrics.UpdateMetrics(stats, savings)

	assert.Equal(t, 1, metrics.TotalRequests)
	assert.Equal(t, 1, metrics.RequestsWithCache)
	assert.Equal(t, 1.0, metrics.CacheHitRate)
	assert.Equal(t, 2000, metrics.TotalTokensCached)
	assert.Equal(t, 8000, metrics.TotalTokensRead)
}

// TestIntegrationRealMessageSequence tests with a realistic conversation
func TestIntegrationRealMessageSequence(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are an expert Go developer helping with code reviews."},
		{Role: "user", Content: "Review this function: func Add(a, b int) int { return a + b }"},
		{Role: "assistant", Content: "The function looks good. It's simple and clear."},
		{Role: "user", Content: "What about error handling?"},
		{Role: "assistant", Content: "For addition, error handling isn't typically needed."},
		{Role: "user", Content: "Can you show me a more complex example?"},
	}

	tests := []struct {
		name             string
		strategy         CacheStrategy
		tools            []Tool
		expectCachedIdxs []int
	}{
		{
			name:             "system strategy",
			strategy:         CacheStrategySystem,
			tools:            nil,
			expectCachedIdxs: []int{0},
		},
		{
			name:     "tools strategy with tools",
			strategy: CacheStrategyTools,
			tools: []Tool{
				{Type: "function", Function: FunctionDefinition{Name: "lint"}},
			},
			expectCachedIdxs: []int{0},
		},
		{
			name:             "context strategy",
			strategy:         CacheStrategyContext,
			tools:            nil,
			expectCachedIdxs: []int{0, 3, 5}, // system + last 2 user messages
		},
		{
			name:             "aggressive strategy",
			strategy:         CacheStrategyAggressive,
			tools:            nil,
			expectCachedIdxs: []int{0, 1, 3, 5}, // system + all user messages
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CacheConfig{
				Enabled:  true,
				Strategy: tt.strategy,
			}

			result := ApplyCacheControl(messages, tt.tools, config)

			for i, msg := range result {
				shouldBeCached := false
				for _, idx := range tt.expectCachedIdxs {
					if i == idx {
						shouldBeCached = true
						break
					}
				}

				if shouldBeCached {
					require.NotNil(t, msg.CacheControl, "Message at index %d should be cached for strategy %s", i, tt.strategy)
				} else {
					assert.Nil(t, msg.CacheControl, "Message at index %d should not be cached for strategy %s", i, tt.strategy)
				}
			}
		})
	}
}

// TestIntegrationToolCachingScenarios tests various tool caching scenarios
func TestIntegrationToolCachingScenarios(t *testing.T) {
	baseMessages := []Message{
		{Role: "system", Content: "You are a helpful assistant with tools."},
		{Role: "user", Content: "Help me with something."},
	}

	tests := []struct {
		name         string
		tools        []Tool
		strategy     CacheStrategy
		expectCached bool
	}{
		{
			name:         "no tools with tools strategy",
			tools:        nil,
			strategy:     CacheStrategyTools,
			expectCached: false,
		},
		{
			name:         "empty tools with tools strategy",
			tools:        []Tool{},
			strategy:     CacheStrategyTools,
			expectCached: false,
		},
		{
			name: "single tool with tools strategy",
			tools: []Tool{
				{Type: "function", Function: FunctionDefinition{Name: "search"}},
			},
			strategy:     CacheStrategyTools,
			expectCached: true,
		},
		{
			name: "multiple tools with tools strategy",
			tools: []Tool{
				{Type: "function", Function: FunctionDefinition{Name: "search"}},
				{Type: "function", Function: FunctionDefinition{Name: "calculate"}},
				{Type: "function", Function: FunctionDefinition{Name: "translate"}},
			},
			strategy:     CacheStrategyTools,
			expectCached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CacheConfig{
				Enabled:  true,
				Strategy: tt.strategy,
			}

			result := ApplyCacheControl(baseMessages, tt.tools, config)

			if tt.expectCached {
				require.NotNil(t, result[0].CacheControl, "System message should be cached")
				assert.Equal(t, "ephemeral", result[0].CacheControl.Type)
			} else {
				assert.Nil(t, result[0].CacheControl, "System message should not be cached")
			}
		})
	}
}

// TestCacheableMessageStructure tests the CacheableMessage structure
func TestCacheableMessageStructure(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello world",
		Name:    "TestUser",
	}

	cacheable := CacheableMessage{
		Message:      msg,
		CacheControl: &CacheControl{Type: "ephemeral"},
	}

	assert.Equal(t, msg.Role, cacheable.Role)
	assert.Equal(t, msg.Content, cacheable.Content)
	assert.Equal(t, msg.Name, cacheable.Name)
	require.NotNil(t, cacheable.CacheControl)
	assert.Equal(t, "ephemeral", cacheable.CacheControl.Type)
}

// TestCacheStatsStructure tests the CacheStats structure
func TestCacheStatsStructure(t *testing.T) {
	stats := CacheStats{
		CacheCreationInputTokens: 1000,
		CacheReadInputTokens:     2000,
		InputTokens:              3000,
		OutputTokens:             500,
	}

	assert.Equal(t, 1000, stats.CacheCreationInputTokens)
	assert.Equal(t, 2000, stats.CacheReadInputTokens)
	assert.Equal(t, 3000, stats.InputTokens)
	assert.Equal(t, 500, stats.OutputTokens)
}

// TestCacheSavingsStructure tests the CacheSavings structure
func TestCacheSavingsStructure(t *testing.T) {
	savings := CacheSavings{
		CostWithCache:    0.01,
		CostWithoutCache: 0.03,
		Savings:          0.02,
		SavingsPercent:   66.67,
		TokensCached:     1000,
		TokensRead:       2000,
	}

	assert.Equal(t, 0.01, savings.CostWithCache)
	assert.Equal(t, 0.03, savings.CostWithoutCache)
	assert.Equal(t, 0.02, savings.Savings)
	assert.Equal(t, 66.67, savings.SavingsPercent)
	assert.Equal(t, 1000, savings.TokensCached)
	assert.Equal(t, 2000, savings.TokensRead)
}

// BenchmarkApplyCacheControl benchmarks cache control application
func BenchmarkApplyCacheControl(b *testing.B) {
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I'm doing well."},
	}

	tools := []Tool{
		{Type: "function", Function: FunctionDefinition{Name: "test"}},
	}

	config := DefaultCacheConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ApplyCacheControl(messages, tools, config)
	}
}

// BenchmarkCalculateCacheSavings benchmarks savings calculation
func BenchmarkCalculateCacheSavings(b *testing.B) {
	stats := CacheStats{
		CacheCreationInputTokens: 1000,
		CacheReadInputTokens:     5000,
		InputTokens:              6000,
		OutputTokens:             1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateCacheSavings(stats, 0.01, 0.001)
	}
}

// BenchmarkCacheMetricsUpdate benchmarks metrics update
func BenchmarkCacheMetricsUpdate(b *testing.B) {
	metrics := CacheMetrics{}
	stats := CacheStats{
		CacheCreationInputTokens: 1000,
		CacheReadInputTokens:     2000,
		InputTokens:              3000,
		OutputTokens:             500,
	}
	savings := CacheSavings{
		Savings:        0.01,
		SavingsPercent: 50.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.UpdateMetrics(stats, savings)
	}
}
