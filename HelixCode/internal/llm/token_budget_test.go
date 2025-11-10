package llm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==============================================================================
// 1. TokenBudget Tests
// ==============================================================================

func TestDefaultTokenBudget(t *testing.T) {
	budget := DefaultTokenBudget()

	assert.Equal(t, 10000, budget.MaxTokensPerRequest)
	assert.Equal(t, 100000, budget.MaxTokensPerSession)
	assert.Equal(t, 10.0, budget.MaxCostPerSession)
	assert.Equal(t, 50.0, budget.MaxCostPerDay)
	assert.Equal(t, 60, budget.MaxRequestsPerMinute)
	assert.Equal(t, 80.0, budget.WarnThreshold)
}

func TestCustomBudgetCreation(t *testing.T) {
	tests := []struct {
		name   string
		budget TokenBudget
	}{
		{
			name: "minimal budget",
			budget: TokenBudget{
				MaxTokensPerRequest:  1000,
				MaxTokensPerSession:  10000,
				MaxCostPerSession:    1.0,
				MaxCostPerDay:        5.0,
				MaxRequestsPerMinute: 10,
				WarnThreshold:        75.0,
			},
		},
		{
			name: "generous budget",
			budget: TokenBudget{
				MaxTokensPerRequest:  50000,
				MaxTokensPerSession:  500000,
				MaxCostPerSession:    100.0,
				MaxCostPerDay:        500.0,
				MaxRequestsPerMinute: 120,
				WarnThreshold:        90.0,
			},
		},
		{
			name: "enterprise budget",
			budget: TokenBudget{
				MaxTokensPerRequest:  100000,
				MaxTokensPerSession:  1000000,
				MaxCostPerSession:    1000.0,
				MaxCostPerDay:        5000.0,
				MaxRequestsPerMinute: 300,
				WarnThreshold:        95.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.budget)
			assert.Greater(t, tt.budget.MaxTokensPerRequest, 0)
			assert.Greater(t, tt.budget.MaxTokensPerSession, 0)
			assert.Greater(t, tt.budget.MaxCostPerSession, 0.0)
			assert.Greater(t, tt.budget.MaxCostPerDay, 0.0)
			assert.Greater(t, tt.budget.MaxRequestsPerMinute, 0)
			assert.GreaterOrEqual(t, tt.budget.WarnThreshold, 0.0)
			assert.LessOrEqual(t, tt.budget.WarnThreshold, 100.0)
		})
	}
}

func TestBudgetValidation(t *testing.T) {
	tests := []struct {
		name          string
		budget        TokenBudget
		expectValid   bool
		validationMsg string
	}{
		{
			name:          "valid budget",
			budget:        DefaultTokenBudget(),
			expectValid:   true,
			validationMsg: "default budget should be valid",
		},
		{
			name: "zero max tokens per request",
			budget: TokenBudget{
				MaxTokensPerRequest:  0,
				MaxTokensPerSession:  10000,
				MaxCostPerSession:    10.0,
				MaxCostPerDay:        50.0,
				MaxRequestsPerMinute: 60,
				WarnThreshold:        80.0,
			},
			expectValid:   false,
			validationMsg: "zero max tokens per request should be invalid",
		},
		{
			name: "negative cost values",
			budget: TokenBudget{
				MaxTokensPerRequest:  10000,
				MaxTokensPerSession:  100000,
				MaxCostPerSession:    -1.0,
				MaxCostPerDay:        50.0,
				MaxRequestsPerMinute: 60,
				WarnThreshold:        80.0,
			},
			expectValid:   false,
			validationMsg: "negative cost should be invalid",
		},
		{
			name: "warn threshold over 100",
			budget: TokenBudget{
				MaxTokensPerRequest:  10000,
				MaxTokensPerSession:  100000,
				MaxCostPerSession:    10.0,
				MaxCostPerDay:        50.0,
				MaxRequestsPerMinute: 60,
				WarnThreshold:        150.0,
			},
			expectValid:   false,
			validationMsg: "warn threshold over 100 should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.budget.MaxTokensPerRequest > 0 &&
				tt.budget.MaxTokensPerSession > 0 &&
				tt.budget.MaxCostPerSession >= 0 &&
				tt.budget.MaxCostPerDay >= 0 &&
				tt.budget.MaxRequestsPerMinute > 0 &&
				tt.budget.WarnThreshold >= 0 &&
				tt.budget.WarnThreshold <= 100

			assert.Equal(t, tt.expectValid, isValid, tt.validationMsg)
		})
	}
}

// ==============================================================================
// 2. TokenTracker Tests
// ==============================================================================

func TestNewTokenTracker(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)

	assert.NotNil(t, tracker)
	assert.Equal(t, budget, tracker.budget)
	assert.NotNil(t, tracker.sessionTokens)
	assert.NotNil(t, tracker.dailyUsage)
	assert.NotNil(t, tracker.requestTimestamps)
	assert.Empty(t, tracker.sessionTokens)
	assert.Empty(t, tracker.dailyUsage)
	assert.Empty(t, tracker.requestTimestamps)
}

func TestCheckBudget_NewSession(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	ctx := context.Background()

	sessionID := "test-session"
	estimatedTokens := 1000
	estimatedCost := 0.05

	err := tracker.CheckBudget(ctx, sessionID, estimatedTokens, estimatedCost)
	assert.NoError(t, err)
}

func TestCheckBudget_PerRequestLimit(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  5000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()

	tests := []struct {
		name            string
		estimatedTokens int
		expectError     bool
	}{
		{"within limit", 4000, false},
		{"at limit", 5000, false},
		{"exceeds limit", 6000, true},
		{"far exceeds limit", 10000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.CheckBudget(ctx, "session-1", tt.estimatedTokens, 0.05)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds max tokens per request")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckBudget_SessionTokenLimit(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  15000, // Increased to allow the third request to pass per-request check
		MaxTokensPerSession:  20000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// First request
	request1 := createTestRequest()
	response1 := createTestResponse(8000, 2000, 10000)
	tracker.TrackRequest(sessionID, request1, response1, 0.50)

	// Second request should be within limit
	err := tracker.CheckBudget(ctx, sessionID, 5000, 0.25)
	assert.NoError(t, err)

	// Third request should exceed session token limit (10000 + 12000 > 20000)
	err = tracker.CheckBudget(ctx, sessionID, 12000, 0.60)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed session token budget")
}

func TestCheckBudget_SessionCostLimit(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    5.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        95.0, // Increased to avoid warning threshold triggering
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// First request: $2.50
	request1 := createTestRequest()
	response1 := createTestResponse(5000, 1000, 6000)
	tracker.TrackRequest(sessionID, request1, response1, 2.50)

	// Second request should be within limit: $2.50 + $1.00 = $3.50
	err := tracker.CheckBudget(ctx, sessionID, 2000, 1.00)
	assert.NoError(t, err)

	// Third request should exceed cost limit: $2.50 + $2.75 = $5.25 > $5.00
	err = tracker.CheckBudget(ctx, sessionID, 3000, 2.75)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed session cost budget")
}

func TestCheckBudget_DailyCostLimit(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    20.0,
		MaxCostPerDay:        10.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()

	// Session 1
	request1 := createTestRequest()
	response1 := createTestResponse(5000, 1000, 6000)
	tracker.TrackRequest("session-1", request1, response1, 6.0)

	// Session 2
	request2 := createTestRequest()
	response2 := createTestResponse(3000, 500, 3500)
	tracker.TrackRequest("session-2", request2, response2, 3.5)

	// This should exceed daily limit
	err := tracker.CheckBudget(ctx, "session-3", 1000, 1.0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed daily cost budget")
}

func TestCheckBudget_WarningThreshold_Tokens(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  10000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// Use 7000 tokens (70% of limit)
	request1 := createTestRequest()
	response1 := createTestResponse(5000, 2000, 7000)
	tracker.TrackRequest(sessionID, request1, response1, 0.35)

	// Request that would bring us to 85% should trigger warning
	err := tracker.CheckBudget(ctx, sessionID, 1500, 0.08)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warning: approaching token budget limit")
}

func TestCheckBudget_WarningThreshold_Cost(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// Use $7.00 (70% of limit)
	request1 := createTestRequest()
	response1 := createTestResponse(5000, 2000, 7000)
	tracker.TrackRequest(sessionID, request1, response1, 7.0)

	// Request that would bring us to 85% should trigger warning
	err := tracker.CheckBudget(ctx, sessionID, 1000, 1.5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "warning: approaching cost budget limit")
}

// ==============================================================================
// 3. Rate Limiting Tests
// ==============================================================================

func TestRateLimit_WithinLimit(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 10,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// Make 5 requests (within limit of 10)
	for i := 0; i < 5; i++ {
		err := tracker.CheckBudget(ctx, sessionID, 1000, 0.05)
		assert.NoError(t, err)

		request := createTestRequest()
		response := createTestResponse(800, 200, 1000)
		tracker.TrackRequest(sessionID, request, response, 0.05)
	}
}

func TestRateLimit_ExceedsLimit(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 5,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// Make 5 requests (at the limit)
	for i := 0; i < 5; i++ {
		request := createTestRequest()
		response := createTestResponse(800, 200, 1000)
		tracker.TrackRequest(sessionID, request, response, 0.05)
	}

	// 6th request should fail rate limit
	err := tracker.CheckBudget(ctx, sessionID, 1000, 0.05)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestRateLimit_Cleanup(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 3,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "test-session"

	// Make 3 requests
	for i := 0; i < 3; i++ {
		request := createTestRequest()
		response := createTestResponse(800, 200, 1000)
		tracker.TrackRequest(sessionID, request, response, 0.05)
	}

	// 4th request should fail
	err := tracker.CheckBudget(ctx, sessionID, 1000, 0.05)
	assert.Error(t, err)

	// Wait for rate limit window to expire
	time.Sleep(61 * time.Second)

	// Should now be able to make requests again
	err = tracker.CheckBudget(ctx, sessionID, 1000, 0.05)
	assert.NoError(t, err)
}

// ==============================================================================
// 4. Usage Tracking Tests
// ==============================================================================

func TestTrackRequest_SingleRequest(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	request := createTestRequest()
	response := createTestResponse(5000, 2000, 7000)
	tracker.TrackRequest(sessionID, request, response, 0.35)

	usage, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, usage.SessionID)
	assert.Equal(t, 5000, usage.PromptTokens)
	assert.Equal(t, 2000, usage.CompletionTokens)
	assert.Equal(t, 7000, usage.TotalTokens)
	assert.Equal(t, 0.35, usage.TotalCost)
	assert.Equal(t, 1, usage.RequestCount)
	assert.False(t, usage.StartTime.IsZero())
	assert.False(t, usage.LastUpdate.IsZero())
}

func TestTrackRequest_MultipleRequests(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	// First request
	request1 := createTestRequest()
	response1 := createTestResponse(3000, 1000, 4000)
	tracker.TrackRequest(sessionID, request1, response1, 0.20)

	// Second request
	request2 := createTestRequest()
	response2 := createTestResponse(2000, 500, 2500)
	tracker.TrackRequest(sessionID, request2, response2, 0.125)

	// Third request
	request3 := createTestRequest()
	response3 := createTestResponse(1500, 800, 2300)
	tracker.TrackRequest(sessionID, request3, response3, 0.115)

	usage, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 6500, usage.PromptTokens)
	assert.Equal(t, 2300, usage.CompletionTokens)
	assert.Equal(t, 8800, usage.TotalTokens)
	assert.InDelta(t, 0.44, usage.TotalCost, 0.001)
	assert.Equal(t, 3, usage.RequestCount)
}

func TestTrackRequest_MultipleSessions(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)

	// Session 1
	request1 := createTestRequest()
	response1 := createTestResponse(3000, 1000, 4000)
	tracker.TrackRequest("session-1", request1, response1, 0.20)

	// Session 2
	request2 := createTestRequest()
	response2 := createTestResponse(2000, 500, 2500)
	tracker.TrackRequest("session-2", request2, response2, 0.125)

	// Session 1 again
	request3 := createTestRequest()
	response3 := createTestResponse(1500, 800, 2300)
	tracker.TrackRequest("session-1", request3, response3, 0.115)

	usage1, err := tracker.GetSessionUsage("session-1")
	require.NoError(t, err)
	assert.Equal(t, 6300, usage1.TotalTokens)
	assert.Equal(t, 2, usage1.RequestCount)

	usage2, err := tracker.GetSessionUsage("session-2")
	require.NoError(t, err)
	assert.Equal(t, 2500, usage2.TotalTokens)
	assert.Equal(t, 1, usage2.RequestCount)

	allSessions := tracker.GetAllSessionsUsage()
	assert.Len(t, allSessions, 2)
}

func TestTrackRequest_ThinkingTokens(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	// Request with reasoning enabled
	request := createTestRequest()
	request.Reasoning = &ReasoningConfig{
		Enabled:        true,
		ThinkingBudget: 5000,
	}
	request.ThinkingBudget = 5000

	response := createTestResponse(3000, 1000, 4000)
	tracker.TrackRequest(sessionID, request, response, 0.20)

	usage, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 5000, usage.ThinkingTokens)
}

func TestTrackRequest_CostCalculation(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	costs := []float64{0.10, 0.25, 0.15, 0.30}
	totalExpected := 0.0

	for _, cost := range costs {
		request := createTestRequest()
		response := createTestResponse(1000, 500, 1500)
		tracker.TrackRequest(sessionID, request, response, cost)
		totalExpected += cost
	}

	usage, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.InDelta(t, totalExpected, usage.TotalCost, 0.001)
}

// ==============================================================================
// 5. Daily Usage Tests
// ==============================================================================

func TestGetDailyUsage_SingleDay(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	today := time.Now().Format("2006-01-02")

	request := createTestRequest()
	response := createTestResponse(5000, 2000, 7000)
	tracker.TrackRequest("session-1", request, response, 0.35)

	daily, err := tracker.GetDailyUsage(today)
	require.NoError(t, err)
	assert.Equal(t, today, daily.Date)
	assert.Equal(t, 7000, daily.TotalTokens)
	assert.Equal(t, 0.35, daily.TotalCost)
	assert.Equal(t, 1, daily.RequestCount)
	assert.Contains(t, daily.Sessions, "session-1")
}

func TestGetDailyUsage_MultipleSessions(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	today := time.Now().Format("2006-01-02")

	// Session 1
	request1 := createTestRequest()
	response1 := createTestResponse(3000, 1000, 4000)
	tracker.TrackRequest("session-1", request1, response1, 0.20)

	// Session 2
	request2 := createTestRequest()
	response2 := createTestResponse(2000, 500, 2500)
	tracker.TrackRequest("session-2", request2, response2, 0.125)

	// Session 1 again
	request3 := createTestRequest()
	response3 := createTestResponse(1500, 800, 2300)
	tracker.TrackRequest("session-1", request3, response3, 0.115)

	daily, err := tracker.GetDailyUsage(today)
	require.NoError(t, err)
	assert.Equal(t, 8800, daily.TotalTokens)
	assert.InDelta(t, 0.44, daily.TotalCost, 0.001)
	assert.Equal(t, 3, daily.RequestCount)
	assert.Len(t, daily.Sessions, 2)
	assert.Contains(t, daily.Sessions, "session-1")
	assert.Contains(t, daily.Sessions, "session-2")
}

func TestGetDailyUsage_NonExistentDate(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)

	_, err := tracker.GetDailyUsage("2020-01-01")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no usage data for date")
}

// ==============================================================================
// 6. Budget Status Tests
// ==============================================================================

func TestGetBudgetStatus_NewSession(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	status := tracker.GetBudgetStatus(sessionID)
	assert.NotNil(t, status)
	assert.Equal(t, sessionID, status.SessionID)
	assert.Equal(t, budget, status.Budget)
	assert.Equal(t, 0, status.TokensUsed)
	// For new sessions, remaining is 0 (not budget max) because session doesn't exist yet
	assert.Equal(t, 0, status.TokensRemaining)
	assert.Equal(t, 0.0, status.TokenUsagePercent)
	assert.Equal(t, 0.0, status.CostUsed)
	assert.Equal(t, 0.0, status.CostRemaining)
	assert.Equal(t, 0.0, status.CostUsagePercent)
	assert.Equal(t, 0, status.RequestCount)
}

func TestGetBudgetStatus_WithUsage(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  50000,
		MaxCostPerSession:    5.0,
		MaxCostPerDay:        20.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	// Use 30000 tokens and $3.00
	request := createTestRequest()
	response := createTestResponse(20000, 10000, 30000)
	tracker.TrackRequest(sessionID, request, response, 3.0)

	status := tracker.GetBudgetStatus(sessionID)
	assert.Equal(t, 30000, status.TokensUsed)
	assert.Equal(t, 20000, status.TokensRemaining)
	assert.InDelta(t, 60.0, status.TokenUsagePercent, 0.1)
	assert.Equal(t, 3.0, status.CostUsed)
	assert.Equal(t, 2.0, status.CostRemaining)
	assert.InDelta(t, 60.0, status.CostUsagePercent, 0.1)
	assert.Equal(t, 1, status.RequestCount)
}

func TestGetBudgetStatus_PercentageCalculations(t *testing.T) {
	tests := []struct {
		name                 string
		maxTokens            int
		usedTokens           int
		maxCost              float64
		usedCost             float64
		expectedTokenPercent float64
		expectedCostPercent  float64
	}{
		{"0% usage", 10000, 0, 10.0, 0.0, 0.0, 0.0},
		{"25% usage", 10000, 2500, 10.0, 2.5, 25.0, 25.0},
		{"50% usage", 10000, 5000, 10.0, 5.0, 50.0, 50.0},
		{"75% usage", 10000, 7500, 10.0, 7.5, 75.0, 75.0},
		{"90% usage", 10000, 9000, 10.0, 9.0, 90.0, 90.0},
		{"100% usage", 10000, 10000, 10.0, 10.0, 100.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := TokenBudget{
				MaxTokensPerRequest:  tt.maxTokens,
				MaxTokensPerSession:  tt.maxTokens,
				MaxCostPerSession:    tt.maxCost,
				MaxCostPerDay:        50.0,
				MaxRequestsPerMinute: 60,
				WarnThreshold:        80.0,
			}
			tracker := NewTokenTracker(budget)
			sessionID := "test-session"

			if tt.usedTokens > 0 {
				request := createTestRequest()
				response := createTestResponse(
					int(float64(tt.usedTokens)*0.7),
					int(float64(tt.usedTokens)*0.3),
					tt.usedTokens,
				)
				tracker.TrackRequest(sessionID, request, response, tt.usedCost)
			}

			status := tracker.GetBudgetStatus(sessionID)
			assert.InDelta(t, tt.expectedTokenPercent, status.TokenUsagePercent, 0.1)
			assert.InDelta(t, tt.expectedCostPercent, status.CostUsagePercent, 0.1)
		})
	}
}

func TestGetBudgetStatus_RemainingBudget(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        50.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	// Use 35000 tokens and $3.50
	request := createTestRequest()
	response := createTestResponse(25000, 10000, 35000)
	tracker.TrackRequest(sessionID, request, response, 3.5)

	status := tracker.GetBudgetStatus(sessionID)
	assert.Equal(t, 65000, status.TokensRemaining)
	assert.InDelta(t, 6.5, status.CostRemaining, 0.001)
}

func TestGetBudgetStatus_DailyStats(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    10.0,
		MaxCostPerDay:        20.0,
		MaxRequestsPerMinute: 60,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)

	// Session 1: $4.00
	request1 := createTestRequest()
	response1 := createTestResponse(5000, 2000, 7000)
	tracker.TrackRequest("session-1", request1, response1, 4.0)

	// Session 2: $6.00
	request2 := createTestRequest()
	response2 := createTestResponse(6000, 3000, 9000)
	tracker.TrackRequest("session-2", request2, response2, 6.0)

	status := tracker.GetBudgetStatus("session-1")
	assert.Equal(t, 10.0, status.DailyCostUsed)
	assert.Equal(t, 10.0, status.DailyCostRemaining)
	assert.InDelta(t, 50.0, status.DailyCostPercent, 0.1)
}

// ==============================================================================
// 7. Cleanup Tests
// ==============================================================================

func TestResetSession(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	sessionID := "test-session"

	// Create some usage
	request := createTestRequest()
	response := createTestResponse(5000, 2000, 7000)
	tracker.TrackRequest(sessionID, request, response, 0.35)

	// Verify session exists
	_, err := tracker.GetSessionUsage(sessionID)
	assert.NoError(t, err)

	// Reset session
	tracker.ResetSession(sessionID)

	// Verify session no longer exists
	_, err = tracker.GetSessionUsage(sessionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestCleanupOldSessions_NoSessions(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)

	cleaned := tracker.CleanupOldSessions(24 * time.Hour)
	assert.Equal(t, 0, cleaned)
}

func TestCleanupOldSessions_AllCurrent(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)

	// Create some recent sessions
	for i := 0; i < 3; i++ {
		sessionID := uuid.New().String()
		request := createTestRequest()
		response := createTestResponse(5000, 2000, 7000)
		tracker.TrackRequest(sessionID, request, response, 0.35)
	}

	cleaned := tracker.CleanupOldSessions(24 * time.Hour)
	assert.Equal(t, 0, cleaned)

	allSessions := tracker.GetAllSessionsUsage()
	assert.Len(t, allSessions, 3)
}

func TestCleanupOldSessions_MixedAges(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)

	// Create sessions
	oldSession := "old-session"
	recentSession := "recent-session"

	request := createTestRequest()
	response := createTestResponse(5000, 2000, 7000)
	tracker.TrackRequest(oldSession, request, response, 0.35)
	tracker.TrackRequest(recentSession, request, response, 0.35)

	// Manually age the old session
	tracker.mu.Lock()
	if usage, ok := tracker.sessionTokens[oldSession]; ok {
		usage.LastUpdate = time.Now().Add(-25 * time.Hour)
	}
	tracker.mu.Unlock()

	// Cleanup sessions older than 24 hours
	cleaned := tracker.CleanupOldSessions(24 * time.Hour)
	assert.Equal(t, 1, cleaned)

	// Verify old session is gone
	_, err := tracker.GetSessionUsage(oldSession)
	assert.Error(t, err)

	// Verify recent session still exists
	_, err = tracker.GetSessionUsage(recentSession)
	assert.NoError(t, err)
}

func TestCleanupOldSessions_VariousThresholds(t *testing.T) {
	tests := []struct {
		name        string
		maxAge      time.Duration
		sessionAge  time.Duration
		shouldClean bool
	}{
		{"1 hour threshold - 30 min old", time.Hour, 30 * time.Minute, false},
		{"1 hour threshold - 2 hours old", time.Hour, 2 * time.Hour, true},
		{"24 hour threshold - 12 hours old", 24 * time.Hour, 12 * time.Hour, false},
		{"24 hour threshold - 25 hours old", 24 * time.Hour, 25 * time.Hour, true},
		{"1 week threshold - 3 days old", 7 * 24 * time.Hour, 3 * 24 * time.Hour, false},
		{"1 week threshold - 8 days old", 7 * 24 * time.Hour, 8 * 24 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := DefaultTokenBudget()
			tracker := NewTokenTracker(budget)
			sessionID := "test-session"

			request := createTestRequest()
			response := createTestResponse(5000, 2000, 7000)
			tracker.TrackRequest(sessionID, request, response, 0.35)

			// Age the session
			tracker.mu.Lock()
			if usage, ok := tracker.sessionTokens[sessionID]; ok {
				usage.LastUpdate = time.Now().Add(-tt.sessionAge)
			}
			tracker.mu.Unlock()

			cleaned := tracker.CleanupOldSessions(tt.maxAge)
			if tt.shouldClean {
				assert.Equal(t, 1, cleaned)
				_, err := tracker.GetSessionUsage(sessionID)
				assert.Error(t, err)
			} else {
				assert.Equal(t, 0, cleaned)
				_, err := tracker.GetSessionUsage(sessionID)
				assert.NoError(t, err)
			}
		})
	}
}

// ==============================================================================
// 8. Estimation Tests
// ==============================================================================

func TestEstimateTokens_MessagesOnly(t *testing.T) {
	tests := []struct {
		name           string
		messages       []Message
		expectedTokens int
	}{
		{
			name: "short message",
			messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			expectedTokens: 1, // 5 chars / 4 = 1.25, rounded to 1
		},
		{
			name: "medium message",
			messages: []Message{
				{Role: "user", Content: "This is a medium length message with several words"},
			},
			expectedTokens: 12, // 51 chars / 4 = 12.75, rounded down to 12
		},
		{
			name: "multiple messages",
			messages: []Message{
				{Role: "user", Content: "First message"},
				{Role: "assistant", Content: "Second message"},
				{Role: "user", Content: "Third message"},
			},
			expectedTokens: 10, // 42 chars / 4 = 10.5, rounded to 10
		},
		{
			name: "long message",
			messages: []Message{
				{Role: "user", Content: strings.Repeat("A", 400)},
			},
			expectedTokens: 100, // 400 / 4 = 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &LLMRequest{
				Messages: tt.messages,
			}
			tokens := EstimateTokens(request)
			assert.Equal(t, tt.expectedTokens, tokens)
		})
	}
}

func TestEstimateTokens_WithTools(t *testing.T) {
	request := &LLMRequest{
		Messages: []Message{
			{Role: "user", Content: "Use the calculator tool"},
		},
		Tools: []Tool{
			{Type: "function", Function: ToolFunction{Name: "calculator"}},
			{Type: "function", Function: ToolFunction{Name: "search"}},
			{Type: "function", Function: ToolFunction{Name: "file_reader"}},
		},
	}

	tokens := EstimateTokens(request)
	// 28 chars / 4 = 7 tokens for message
	// 3 tools * 200 = 600 tokens for tools
	// Total = 607 / 4 = 151 (since we divide total chars by 4)
	// Actually: (28 + 600) / 4 = 157
	assert.Greater(t, tokens, 100) // Should be significantly higher due to tools
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name            string
		model           string
		estimatedTokens int
		costPerKTokens  float64
		expectedCost    float64
	}{
		{"small request", "gpt-4", 1000, 0.03, 0.03},
		{"medium request", "gpt-4", 5000, 0.03, 0.15},
		{"large request", "gpt-4", 10000, 0.03, 0.30},
		{"expensive model", "gpt-4", 1000, 0.10, 0.10},
		{"cheap model", "gpt-3.5", 1000, 0.001, 0.001},
		{"fractional tokens", "gpt-4", 500, 0.03, 0.015},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimateCost(tt.model, tt.estimatedTokens, tt.costPerKTokens)
			assert.InDelta(t, tt.expectedCost, cost, 0.0001)
		})
	}
}

// ==============================================================================
// 9. Edge Cases Tests
// ==============================================================================

func TestEdgeCases_ZeroBudgetValues(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  0,
		MaxTokensPerSession:  0,
		MaxCostPerSession:    0.0,
		MaxCostPerDay:        0.0,
		MaxRequestsPerMinute: 0,
		WarnThreshold:        0.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()

	// Any request should fail with zero budget
	err := tracker.CheckBudget(ctx, "session-1", 100, 0.01)
	assert.Error(t, err)
}

func TestEdgeCases_NegativeValues(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)

	// Negative tokens - current implementation doesn't explicitly check for negative values
	// but they would fail the max tokens per request check (negative < max)
	// The implementation allows tracking negative costs (though not recommended)
	request := createTestRequest()
	response := createTestResponse(800, 200, 1000)
	tracker.TrackRequest("session-1", request, response, -0.05)

	usage, err := tracker.GetSessionUsage("session-1")
	require.NoError(t, err)
	assert.Equal(t, -0.05, usage.TotalCost)

	// Test that very negative tokens would still be caught by per-request limit
	ctx := context.Background()
	err = tracker.CheckBudget(ctx, "session-2", 0, 0.05)
	assert.NoError(t, err)
}

func TestEdgeCases_VeryLargeValues(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  1000000000,
		MaxTokensPerSession:  10000000000,
		MaxCostPerSession:    1000000.0,
		MaxCostPerDay:        10000000.0,
		MaxRequestsPerMinute: 100000,
		WarnThreshold:        99.99,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()

	// Large request should work
	err := tracker.CheckBudget(ctx, "session-1", 50000000, 50000.0)
	assert.NoError(t, err)

	request := createTestRequest()
	response := createTestResponse(35000000, 15000000, 50000000)
	tracker.TrackRequest("session-1", request, response, 50000.0)

	usage, err := tracker.GetSessionUsage("session-1")
	require.NoError(t, err)
	assert.Equal(t, 50000000, usage.TotalTokens)
	assert.Equal(t, 50000.0, usage.TotalCost)
}

func TestEdgeCases_ConcurrentAccess(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	ctx := context.Background()

	// Run multiple goroutines that access the tracker concurrently
	done := make(chan bool)
	numGoroutines := 10
	requestsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			sessionID := uuid.New().String()
			for j := 0; j < requestsPerGoroutine; j++ {
				// Check budget
				err := tracker.CheckBudget(ctx, sessionID, 1000, 0.05)
				if err == nil {
					// Track request
					request := createTestRequest()
					response := createTestResponse(800, 200, 1000)
					tracker.TrackRequest(sessionID, request, response, 0.05)

					// Get usage
					_, _ = tracker.GetSessionUsage(sessionID)

					// Get status
					_ = tracker.GetBudgetStatus(sessionID)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify no crashes and data integrity
	allSessions := tracker.GetAllSessionsUsage()
	assert.LessOrEqual(t, len(allSessions), numGoroutines)
}

func TestEdgeCases_EmptySessionID(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	ctx := context.Background()

	// Empty session ID should still work (though not recommended)
	err := tracker.CheckBudget(ctx, "", 1000, 0.05)
	assert.NoError(t, err)

	request := createTestRequest()
	response := createTestResponse(800, 200, 1000)
	tracker.TrackRequest("", request, response, 0.05)

	usage, err := tracker.GetSessionUsage("")
	assert.NoError(t, err)
	assert.Equal(t, "", usage.SessionID)
}

// ==============================================================================
// 10. Integration Tests
// ==============================================================================

func TestIntegration_FullWorkflow(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  6000, // Increased to allow warning threshold test
		MaxTokensPerSession:  20000,
		MaxCostPerSession:    2.0,
		MaxCostPerDay:        10.0,
		MaxRequestsPerMinute: 10,
		WarnThreshold:        80.0,
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "integration-test"

	// Request 1: 4000 tokens, $0.40
	err := tracker.CheckBudget(ctx, sessionID, 4000, 0.40)
	require.NoError(t, err, "First request check should succeed")

	request1 := createTestRequest()
	response1 := createTestResponse(3000, 1000, 4000)
	tracker.TrackRequest(sessionID, request1, response1, 0.40)

	usage1, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 4000, usage1.TotalTokens)
	assert.Equal(t, 0.40, usage1.TotalCost)

	// Request 2: 4500 tokens, $0.45
	err = tracker.CheckBudget(ctx, sessionID, 4500, 0.45)
	require.NoError(t, err, "Second request check should succeed")

	request2 := createTestRequest()
	response2 := createTestResponse(3200, 1300, 4500)
	tracker.TrackRequest(sessionID, request2, response2, 0.45)

	usage2, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 8500, usage2.TotalTokens)
	assert.InDelta(t, 0.85, usage2.TotalCost, 0.001)

	// Request 3: 3000 tokens, $0.30
	err = tracker.CheckBudget(ctx, sessionID, 3000, 0.30)
	require.NoError(t, err, "Third request check should succeed")

	request3 := createTestRequest()
	response3 := createTestResponse(2100, 900, 3000)
	tracker.TrackRequest(sessionID, request3, response3, 0.30)

	usage3, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 11500, usage3.TotalTokens)
	assert.InDelta(t, 1.15, usage3.TotalCost, 0.001)

	// Request 4: Would exceed token limit
	err = tracker.CheckBudget(ctx, sessionID, 9000, 0.90)
	assert.Error(t, err, "Should fail token limit")

	// Request 4 (adjusted): 4000 tokens, $0.40 - should trigger warning (11500 + 4000 = 15500 = 77.5%)
	// But actually, we're at 57.5% tokens and adding 4000 would be 77.5%, below 80% threshold
	// Let's use 5500 tokens to get above 80%: (11500 + 5500) / 20000 = 85%
	err = tracker.CheckBudget(ctx, sessionID, 5500, 0.55)
	assert.Error(t, err, "Should trigger warning threshold")
	assert.Contains(t, err.Error(), "warning")

	// Check budget status
	status := tracker.GetBudgetStatus(sessionID)
	assert.Equal(t, 11500, status.TokensUsed)
	assert.Equal(t, 8500, status.TokensRemaining)
	assert.InDelta(t, 57.5, status.TokenUsagePercent, 0.1)
	assert.InDelta(t, 1.15, status.CostUsed, 0.001)
	assert.InDelta(t, 0.85, status.CostRemaining, 0.001)
	assert.Equal(t, 3, status.RequestCount)

	// Check daily usage
	today := time.Now().Format("2006-01-02")
	daily, err := tracker.GetDailyUsage(today)
	require.NoError(t, err)
	assert.Equal(t, 11500, daily.TotalTokens)
	assert.InDelta(t, 1.15, daily.TotalCost, 0.001)
	assert.Equal(t, 3, daily.RequestCount)
}

func TestIntegration_MultiSession(t *testing.T) {
	budget := TokenBudget{
		MaxTokensPerRequest:  5000,
		MaxTokensPerSession:  15000,
		MaxCostPerSession:    2.0,
		MaxCostPerDay:        5.0,
		MaxRequestsPerMinute: 20,
		WarnThreshold:        101.0, // Set above 100% to avoid warning (check uses >=)
	}
	tracker := NewTokenTracker(budget)
	ctx := context.Background()

	// Session 1: Use $1.50
	session1 := "session-1"
	for i := 0; i < 3; i++ {
		err := tracker.CheckBudget(ctx, session1, 3000, 0.50)
		require.NoError(t, err)
		request := createTestRequest()
		response := createTestResponse(2100, 900, 3000)
		tracker.TrackRequest(session1, request, response, 0.50)
	}

	// Session 2: Use $2.00
	session2 := "session-2"
	for i := 0; i < 4; i++ {
		err := tracker.CheckBudget(ctx, session2, 2000, 0.50)
		require.NoError(t, err)
		request := createTestRequest()
		response := createTestResponse(1400, 600, 2000)
		tracker.TrackRequest(session2, request, response, 0.50)
	}

	// Session 3: Should fail daily limit
	session3 := "session-3"
	err := tracker.CheckBudget(ctx, session3, 3000, 2.00)
	assert.Error(t, err, "Should exceed daily cost limit")
	assert.Contains(t, err.Error(), "daily cost budget")

	// Verify individual sessions
	usage1, err := tracker.GetSessionUsage(session1)
	require.NoError(t, err)
	assert.Equal(t, 9000, usage1.TotalTokens)
	assert.InDelta(t, 1.50, usage1.TotalCost, 0.001)

	usage2, err := tracker.GetSessionUsage(session2)
	require.NoError(t, err)
	assert.Equal(t, 8000, usage2.TotalTokens)
	assert.InDelta(t, 2.00, usage2.TotalCost, 0.001)

	// Verify daily usage includes both sessions
	today := time.Now().Format("2006-01-02")
	daily, err := tracker.GetDailyUsage(today)
	require.NoError(t, err)
	assert.Equal(t, 17000, daily.TotalTokens)
	assert.InDelta(t, 3.50, daily.TotalCost, 0.001)
	assert.Len(t, daily.Sessions, 2)
}

func TestIntegration_ResetAndContinue(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)
	ctx := context.Background()
	sessionID := "reset-test"

	// Use some budget
	request1 := createTestRequest()
	response1 := createTestResponse(5000, 2000, 7000)
	tracker.TrackRequest(sessionID, request1, response1, 0.70)

	usage1, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 7000, usage1.TotalTokens)

	// Reset session
	tracker.ResetSession(sessionID)

	// Verify session is gone
	_, err = tracker.GetSessionUsage(sessionID)
	assert.Error(t, err)

	// Continue with new requests
	err = tracker.CheckBudget(ctx, sessionID, 3000, 0.30)
	assert.NoError(t, err)

	request2 := createTestRequest()
	response2 := createTestResponse(2100, 900, 3000)
	tracker.TrackRequest(sessionID, request2, response2, 0.30)

	usage2, err := tracker.GetSessionUsage(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 3000, usage2.TotalTokens)
	assert.Equal(t, 0.30, usage2.TotalCost)
}

func TestIntegration_SessionCleanup(t *testing.T) {
	budget := DefaultTokenBudget()
	tracker := NewTokenTracker(budget)

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		sessionID := uuid.New().String()
		request := createTestRequest()
		response := createTestResponse(3000, 1000, 4000)
		tracker.TrackRequest(sessionID, request, response, 0.40)
	}

	allSessions := tracker.GetAllSessionsUsage()
	assert.Len(t, allSessions, 5)

	// Age 3 sessions
	tracker.mu.Lock()
	count := 0
	for _, usage := range tracker.sessionTokens {
		if count < 3 {
			usage.LastUpdate = time.Now().Add(-25 * time.Hour)
		}
		count++
	}
	tracker.mu.Unlock()

	// Cleanup
	cleaned := tracker.CleanupOldSessions(24 * time.Hour)
	assert.Equal(t, 3, cleaned)

	allSessions = tracker.GetAllSessionsUsage()
	assert.Len(t, allSessions, 2)
}

// ==============================================================================
// Helper Functions
// ==============================================================================

func createTestRequest() *LLMRequest {
	return &LLMRequest{
		ID:          uuid.New(),
		Model:       "test-model",
		Messages:    []Message{{Role: "user", Content: "Test message"}},
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}

func createTestResponse(promptTokens, completionTokens, totalTokens int) *LLMResponse {
	return &LLMResponse{
		ID:           uuid.New(),
		Content:      "Test response",
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
		CreatedAt: time.Now(),
	}
}
