package llm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReasoningConfigDefaults tests default reasoning configuration
func TestReasoningConfigDefaults(t *testing.T) {
	config := DefaultReasoningConfig()

	assert.False(t, config.Enabled)
	assert.True(t, config.ExtractThinking)
	assert.False(t, config.HideFromUser)
	assert.Equal(t, "thinking", config.ThinkingTags)
	assert.Equal(t, 0, config.ThinkingBudget) // unlimited
	assert.Equal(t, "medium", config.ReasoningEffort)
}

// TestReasoningModelDetection tests automatic reasoning model detection
func TestReasoningModelDetection(t *testing.T) {
	tests := []struct {
		modelName    string
		shouldDetect bool
		expectedType ReasoningModelType
	}{
		{"o1-preview", true, ReasoningModelOpenAI_O1},
		{"o1-mini", true, ReasoningModelOpenAI_O1},
		{"o3-turbo", true, ReasoningModelOpenAI_O3},
		{"claude-4-sonnet", true, ReasoningModelClaude_Sonnet},
		{"claude-3-7-sonnet-20250219", true, ReasoningModelClaude_Sonnet},
		{"claude-3-opus-20240229", true, ReasoningModelClaude_Opus},
		{"deepseek-r1", true, ReasoningModelDeepSeek_R1},
		{"qwq-32b-preview", true, ReasoningModelQwQ_32B},
		{"gpt-4o", false, ReasoningModelGeneric},
		{"claude-3-haiku", false, ReasoningModelGeneric},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			isReasoning, modelType := IsReasoningModel(tt.modelName)
			assert.Equal(t, tt.shouldDetect, isReasoning, "Detection mismatch for %s", tt.modelName)
			if tt.shouldDetect {
				assert.Equal(t, tt.expectedType, modelType, "Model type mismatch for %s", tt.modelName)
			}
		})
	}
}

// TestCacheConfigStrategies tests cache configuration strategies
func TestCacheConfigStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy CacheStrategy
		messages []Message
		tools    []Tool
		expected int // number of cached items
	}{
		{
			name:     "None strategy",
			strategy: CacheStrategyNone,
			messages: []Message{{Role: "system", Content: "You are helpful"}},
			tools:    nil,
			expected: 0,
		},
		{
			name:     "System strategy",
			strategy: CacheStrategySystem,
			messages: []Message{{Role: "system", Content: "You are helpful"}},
			tools:    nil,
			expected: 1, // system message cached
		},
		{
			name:     "Tools strategy",
			strategy: CacheStrategyTools,
			messages: []Message{{Role: "system", Content: "You are helpful"}},
			tools:    []Tool{{Type: "function", Function: FunctionDefinition{Name: "test"}}},
			expected: 1, // system message cached (tools present)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CacheConfig{
				Enabled:  true,
				Strategy: tt.strategy,
			}

			cacheableMessages := ApplyCacheControl(tt.messages, tt.tools, config)

			cachedCount := 0
			for _, msg := range cacheableMessages {
				if msg.CacheControl != nil {
					cachedCount++
				}
			}

			assert.Equal(t, tt.expected, cachedCount, "Cached message count mismatch")
		})
	}
}

// TestTokenBudgetEnforcement tests token budget enforcement
func TestTokenBudgetEnforcement(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  1000,
		MaxTokensPerSession:  5000,
		MaxCostPerSession:    1.0,
		MaxRequestsPerMinute: 10,
		WarnThreshold:        80.0,
	}

	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// Test per-request limit
	err := tracker.CheckBudget(ctx, sessionID, 1500, 0.1)
	assert.Error(t, err, "Should reject request exceeding per-request limit")
	assert.Contains(t, err.Error(), "exceeds max tokens per request")

	// Test successful request within limits
	err = tracker.CheckBudget(ctx, sessionID, 800, 0.08)
	assert.NoError(t, err, "Should accept request within limits")

	// Simulate tracking the request
	mockRequest := &LLMRequest{
		ID:        uuid.New(),
		SessionID: sessionID,
		Messages:  []Message{{Role: "user", Content: "test"}},
	}
	mockResponse := &LLMResponse{
		Usage: Usage{
			PromptTokens:     400,
			CompletionTokens: 400,
			TotalTokens:      800,
		},
	}
	tracker.TrackRequest(sessionID, mockRequest, mockResponse, 0.08)

	// Check session usage
	usage, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 800, usage.TotalTokens)
	assert.InDelta(t, 0.08, usage.TotalCost, 0.01)

	// Test session limit after multiple requests
	for i := 0; i < 5; i++ {
		tracker.TrackRequest(sessionID, mockRequest, mockResponse, 0.08)
	}

	// Should now be close to session limit
	err = tracker.CheckBudget(ctx, sessionID, 800, 0.08)
	assert.Error(t, err, "Should reject request approaching session limit")
}

// TestTokenBudgetWarnings tests warning thresholds
func TestTokenBudgetWarnings(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerSession: 10000,
		MaxCostPerSession:   10.0,
		WarnThreshold:       80.0, // Warn at 80%
	}

	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// Add usage up to 85% of limit
	mockRequest := &LLMRequest{ID: uuid.New(), SessionID: sessionID}
	mockResponse := &LLMResponse{
		Usage: Usage{TotalTokens: 8500}, // 85% of 10000
	}
	tracker.TrackRequest(sessionID, mockRequest, mockResponse, 8.5)

	// Check budget should return warning
	err := tracker.CheckBudget(ctx, sessionID, 100, 0.1)
	assert.Error(t, err, "Should warn when approaching limit")
	assert.Contains(t, err.Error(), "warning")
	assert.Contains(t, err.Error(), "approaching")
}

// TestRateLimiting tests request rate limiting
func TestRateLimiting(t *testing.T) {
	budget := TokenBudget{
		MaxRequestsPerMinute: 5,
	}

	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// Make 5 requests quickly
	for i := 0; i < 5; i++ {
		err := tracker.CheckBudget(ctx, sessionID, 100, 0.01)
		assert.NoError(t, err, "Request %d should succeed", i+1)

		// Track the request
		tracker.TrackRequest(sessionID,
			&LLMRequest{ID: uuid.New()},
			&LLMResponse{Usage: Usage{TotalTokens: 100}},
			0.01)
	}

	// 6th request should fail
	err := tracker.CheckBudget(ctx, sessionID, 100, 0.01)
	assert.Error(t, err, "Should fail due to rate limit")
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

// TestBudgetStatus tests budget status reporting
func TestBudgetStatus(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerSession: 10000,
		MaxCostPerSession:   10.0,
	}

	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	// Add some usage
	mockRequest := &LLMRequest{ID: uuid.New()}
	mockResponse := &LLMResponse{
		Usage: Usage{
			PromptTokens:     2000,
			CompletionTokens: 1000,
			TotalTokens:      3000,
		},
	}
	tracker.TrackRequest(sessionID, mockRequest, mockResponse, 3.0)

	// Get status
	status := tracker.GetBudgetStatus(sessionID)
	require.NotNil(t, status)

	assert.Equal(t, 3000, status.TokensUsed)
	assert.Equal(t, 7000, status.TokensRemaining)
	assert.InDelta(t, 30.0, status.TokenUsagePercent, 0.1)

	assert.InDelta(t, 3.0, status.CostUsed, 0.01)
	assert.InDelta(t, 7.0, status.CostRemaining, 0.01)
	assert.InDelta(t, 30.0, status.CostUsagePercent, 0.1)

	assert.Equal(t, 1, status.RequestCount)
}

// TestSessionCleanup tests old session cleanup
func TestSessionCleanup(t *testing.T) {
	tracker := NewTokenTracker(DefaultTokenBudget())

	// Create some sessions
	for i := 0; i < 5; i++ {
		sessionID := uuid.New().String()
		tracker.TrackRequest(sessionID,
			&LLMRequest{ID: uuid.New()},
			&LLMResponse{Usage: Usage{TotalTokens: 100}},
			0.01)
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Cleanup sessions older than 50ms (should remove all)
	cleaned := tracker.CleanupOldSessions(50 * time.Millisecond)
	assert.Equal(t, 5, cleaned, "Should clean all old sessions")

	// Try to get status for a cleaned session
	_, err := tracker.GetSessionUsage("non-existent")
	assert.Error(t, err, "Should not find cleaned session")
}

// TestCacheSavingsCalculation tests cache cost savings calculation
func TestCacheSavingsCalculation(t *testing.T) {
	stats := CacheStats{
		CacheCreationInputTokens: 5000,  // Created 5k tokens in cache
		CacheReadInputTokens:     10000, // Read 10k tokens from cache
		InputTokens:              15000, // Total input
		OutputTokens:             2000,
	}

	inputCostPer1K := 0.01  // $0.01 per 1K tokens
	cacheCostPer1K := 0.001 // $0.001 per 1K cached tokens (10x cheaper)

	savings := CalculateCacheSavings(stats, inputCostPer1K, cacheCostPer1K)

	// Cost with cache:
	// - 5k creation at $0.01/1k = $0.05
	// - 10k reads at $0.001/1k = $0.01
	// - 0k regular at $0.01/1k = $0.00
	// Total: $0.06

	// Cost without cache:
	// - 15k at $0.01/1k = $0.15
	// Total: $0.15

	// Savings: $0.15 - $0.06 = $0.09 (60% reduction)

	assert.InDelta(t, 0.06, savings.CostWithCache, 0.01)
	assert.InDelta(t, 0.15, savings.CostWithoutCache, 0.01)
	assert.InDelta(t, 0.09, savings.Savings, 0.01)
	assert.InDelta(t, 60.0, savings.SavingsPercent, 1.0)
}

// TestProviderManagerWithBudget tests ProviderManager with token budgets
func TestProviderManagerWithBudget(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerSession: 5000,
		MaxCostPerSession:   5.0,
	}

	config := ProviderConfig{
		DefaultProvider: ProviderTypeAnthropic,
	}

	pm := NewProviderManagerWithBudget(config, budget)

	require.NotNil(t, pm)
	require.NotNil(t, pm.tokenTracker)
	require.NotNil(t, pm.cacheMetrics)

	// Verify budget
	tracker := pm.GetTokenTracker()
	require.NotNil(t, tracker)

	// Create a test session
	sessionID := uuid.New().String()

	// Get initial status (should be empty)
	status := pm.GetBudgetStatus(sessionID)
	assert.NotNil(t, status)
	assert.Equal(t, 0, status.TokensUsed)

	// Reset session
	pm.ResetSession(sessionID)

	// Get usage after reset (should be nil or error)
	_, err := pm.GetSessionUsage(sessionID)
	assert.Error(t, err, "Session should not exist after reset")
}

// TestReasoningCostCalculation tests reasoning cost calculation
func TestReasoningCostCalculation(t *testing.T) {
	trace := &ReasoningTrace{
		ThinkingTokens: 10000,
		OutputTokens:   2000,
		TotalTokens:    12000,
	}

	tests := []struct {
		modelType             ReasoningModelType
		expectedThinkingCost  float64
		expectedOutputCost    float64
		expectedTotalCost     float64
	}{
		{
			ReasoningModelOpenAI_O1,
			0.15,  // 10k tokens * $15/1M = $0.15
			0.12,  // 2k tokens * $60/1M = $0.12
			0.27,  // Total
		},
		{
			ReasoningModelClaude_Sonnet,
			0.03,  // 10k tokens * $3/1M = $0.03
			0.03,  // 2k tokens * $15/1M = $0.03
			0.06,  // Total
		},
		{
			ReasoningModelDeepSeek_R1,
			0.0219, // 10k tokens * $2.19/1M
			0.01638, // 2k tokens * $8.19/1M
			0.03828,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.modelType), func(t *testing.T) {
			thinkingCost, outputCost, totalCost := CalculateReasoningCost(
				trace,
				&ReasoningConfig{},
				tt.modelType,
			)

			assert.InDelta(t, tt.expectedThinkingCost, thinkingCost, 0.01)
			assert.InDelta(t, tt.expectedOutputCost, outputCost, 0.01)
			assert.InDelta(t, tt.expectedTotalCost, totalCost, 0.01)
		})
	}
}

// TestReasoningTrace tests reasoning trace extraction
func TestReasoningTrace(t *testing.T) {
	content := `
<thinking>
Let me think about this step by step.
First, I'll analyze the problem.
Then, I'll formulate a solution.
</thinking>

Here is my final answer: The solution is X.
`

	config := &ReasoningConfig{
		Enabled:         true,
		ExtractThinking: true,
		ThinkingTags:    "thinking",
	}

	trace := ExtractReasoningTrace(content, config)

	require.NotNil(t, trace)
	assert.Len(t, trace.ThinkingContent, 1, "Should extract one thinking block")
	assert.Contains(t, trace.ThinkingContent[0], "step by step")
	assert.NotContains(t, trace.OutputContent, "thinking", "Output should not contain thinking tags")
	assert.Contains(t, trace.OutputContent, "final answer", "Output should contain answer")
	assert.Greater(t, trace.ThinkingTokens, 0, "Should count thinking tokens")
	assert.Greater(t, trace.OutputTokens, 0, "Should count output tokens")
}

// TestFormatReasoningPrompt tests reasoning prompt formatting
func TestFormatReasoningPrompt(t *testing.T) {
	tests := []struct {
		modelType      ReasoningModelType
		effort         string
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			ReasoningModelOpenAI_O1,
			"medium",
			[]string{"thorough"}, // o1 gets standard prompt
			[]string{"<think>"}, // o1 doesn't need special tags
		},
		{
			ReasoningModelDeepSeek_R1,
			"high",
			[]string{"<think>", "comprehensive"},
			[]string{},
		},
		{
			ReasoningModelClaude_Sonnet,
			"low",
			[]string{"step-by-step", "brief"},
			[]string{"<think>"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.modelType), func(t *testing.T) {
			config := &ReasoningConfig{
				Enabled:         true,
				ModelType:       tt.modelType,
				ReasoningEffort: tt.effort,
			}

			prompt := FormatReasoningPrompt("Test prompt", config)

			for _, substr := range tt.shouldContain {
				assert.Contains(t, prompt, substr,
					"Prompt should contain '%s' for %s", substr, tt.modelType)
			}

			for _, substr := range tt.shouldNotContain {
				assert.NotContains(t, prompt, substr,
					"Prompt should not contain '%s' for %s", substr, tt.modelType)
			}
		})
	}
}

// TestCacheMetricsTracking tests cache metrics accumulation
func TestCacheMetricsTracking(t *testing.T) {
	metrics := &CacheMetrics{}

	// Simulate several requests with caching
	for i := 0; i < 10; i++ {
		stats := CacheStats{
			CacheCreationInputTokens: 1000,
			CacheReadInputTokens:     2000,
			InputTokens:              3000,
			OutputTokens:             500,
		}

		savings := CalculateCacheSavings(stats, 0.01, 0.001)
		metrics.UpdateMetrics(stats, savings)
	}

	assert.Equal(t, 10, metrics.TotalRequests)
	assert.Equal(t, 10, metrics.RequestsWithCache)
	assert.InDelta(t, 1.0, metrics.CacheHitRate, 0.01) // 100% hit rate
	assert.Equal(t, 10000, metrics.TotalTokensCached)  // 1000 * 10
	assert.Equal(t, 20000, metrics.TotalTokensRead)    // 2000 * 10
	assert.Greater(t, metrics.TotalSavings, 0.0)
	assert.Greater(t, metrics.AverageSavingsPercent, 0.0)
}
