package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenBudget defines token and cost limits for LLM usage
type TokenBudget struct {
	// MaxTokensPerRequest limits tokens for a single request
	MaxTokensPerRequest int `json:"max_tokens_per_request"`

	// MaxTokensPerSession limits total tokens for a session
	MaxTokensPerSession int `json:"max_tokens_per_session"`

	// MaxCostPerSession limits total cost for a session (in USD)
	MaxCostPerSession float64 `json:"max_cost_per_session"`

	// MaxCostPerDay limits total cost per day (in USD)
	MaxCostPerDay float64 `json:"max_cost_per_day"`

	// MaxRequestsPerMinute limits request rate
	MaxRequestsPerMinute int `json:"max_requests_per_minute"`

	// WarnThreshold percentage (0-100) at which to warn user
	WarnThreshold float64 `json:"warn_threshold"`
}

// DefaultTokenBudget returns sensible default budget limits
func DefaultTokenBudget() TokenBudget {
	return TokenBudget{
		MaxTokensPerRequest:  10000,
		MaxTokensPerSession:  100000,
		MaxCostPerSession:    10.0, // $10 per session
		MaxCostPerDay:        50.0, // $50 per day
		MaxRequestsPerMinute: 60,   // 1 per second
		WarnThreshold:        80.0, // Warn at 80%
	}
}

// TokenTracker tracks token usage and enforces budgets
type TokenTracker struct {
	mu                sync.RWMutex
	sessionTokens     map[string]*SessionUsage
	dailyUsage        map[string]*DailyUsage
	requestTimestamps map[string][]time.Time
	budget            TokenBudget
}

// SessionUsage tracks usage for a single session
type SessionUsage struct {
	SessionID        string    `json:"session_id"`
	TotalTokens      int       `json:"total_tokens"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	ThinkingTokens   int       `json:"thinking_tokens"`
	TotalCost        float64   `json:"total_cost"`
	RequestCount     int       `json:"request_count"`
	StartTime        time.Time `json:"start_time"`
	LastUpdate       time.Time `json:"last_update"`
}

// DailyUsage tracks usage for a single day
type DailyUsage struct {
	Date         string   `json:"date"`
	TotalTokens  int      `json:"total_tokens"`
	TotalCost    float64  `json:"total_cost"`
	RequestCount int      `json:"request_count"`
	Sessions     []string `json:"sessions"`
}

// NewTokenTracker creates a new token tracker with budget
func NewTokenTracker(budget TokenBudget) *TokenTracker {
	return &TokenTracker{
		sessionTokens:     make(map[string]*SessionUsage),
		dailyUsage:        make(map[string]*DailyUsage),
		requestTimestamps: make(map[string][]time.Time),
		budget:            budget,
	}
}

// CheckBudget checks if a request would exceed any budget limits
func (tt *TokenTracker) CheckBudget(ctx context.Context, sessionID string, estimatedTokens int, estimatedCost float64) error {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	// Check rate limit
	if err := tt.checkRateLimit(sessionID); err != nil {
		return err
	}

	// Check per-request limit
	if estimatedTokens > tt.budget.MaxTokensPerRequest {
		return fmt.Errorf("request exceeds max tokens per request: %d > %d",
			estimatedTokens, tt.budget.MaxTokensPerRequest)
	}

	// Check session limits
	if session, ok := tt.sessionTokens[sessionID]; ok {
		if session.TotalTokens+estimatedTokens > tt.budget.MaxTokensPerSession {
			return fmt.Errorf("request would exceed session token budget: %d + %d > %d",
				session.TotalTokens, estimatedTokens, tt.budget.MaxTokensPerSession)
		}

		if session.TotalCost+estimatedCost > tt.budget.MaxCostPerSession {
			return fmt.Errorf("request would exceed session cost budget: $%.2f + $%.2f > $%.2f",
				session.TotalCost, estimatedCost, tt.budget.MaxCostPerSession)
		}
	}

	// Check daily limits
	today := time.Now().Format("2006-01-02")
	if daily, ok := tt.dailyUsage[today]; ok {
		if daily.TotalCost+estimatedCost > tt.budget.MaxCostPerDay {
			return fmt.Errorf("request would exceed daily cost budget: $%.2f + $%.2f > $%.2f",
				daily.TotalCost, estimatedCost, tt.budget.MaxCostPerDay)
		}
	}

	// Check for warnings (80% threshold)
	if session, ok := tt.sessionTokens[sessionID]; ok {
		tokenUsagePercent := float64(session.TotalTokens+estimatedTokens) / float64(tt.budget.MaxTokensPerSession) * 100
		if tokenUsagePercent >= tt.budget.WarnThreshold {
			return fmt.Errorf("warning: approaching token budget limit (%.1f%%)", tokenUsagePercent)
		}

		costUsagePercent := (session.TotalCost + estimatedCost) / tt.budget.MaxCostPerSession * 100
		if costUsagePercent >= tt.budget.WarnThreshold {
			return fmt.Errorf("warning: approaching cost budget limit (%.1f%%)", costUsagePercent)
		}
	}

	return nil
}

// checkRateLimit checks if rate limit would be exceeded
func (tt *TokenTracker) checkRateLimit(sessionID string) error {
	now := time.Now()
	oneMinuteAgo := now.Add(-time.Minute)

	// Clean old timestamps
	timestamps := tt.requestTimestamps[sessionID]
	var recent []time.Time
	for _, ts := range timestamps {
		if ts.After(oneMinuteAgo) {
			recent = append(recent, ts)
		}
	}

	if len(recent) >= tt.budget.MaxRequestsPerMinute {
		return fmt.Errorf("rate limit exceeded: %d requests in the last minute (max: %d)",
			len(recent), tt.budget.MaxRequestsPerMinute)
	}

	return nil
}

// TrackRequest records token usage for a request
func (tt *TokenTracker) TrackRequest(sessionID string, request *LLMRequest, response *LLMResponse, cost float64) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	now := time.Now()

	// Initialize session if needed
	if _, ok := tt.sessionTokens[sessionID]; !ok {
		tt.sessionTokens[sessionID] = &SessionUsage{
			SessionID: sessionID,
			StartTime: now,
		}
	}

	// Update session usage
	session := tt.sessionTokens[sessionID]
	session.PromptTokens += response.Usage.PromptTokens
	session.CompletionTokens += response.Usage.CompletionTokens
	session.TotalTokens += response.Usage.TotalTokens
	session.TotalCost += cost
	session.RequestCount++
	session.LastUpdate = now

	// Track thinking tokens if available
	if request.Reasoning != nil && request.Reasoning.Enabled {
		// Estimate thinking tokens (can be refined based on actual usage)
		session.ThinkingTokens += request.ThinkingBudget
	}

	// Update daily usage
	today := now.Format("2006-01-02")
	if _, ok := tt.dailyUsage[today]; !ok {
		tt.dailyUsage[today] = &DailyUsage{
			Date:     today,
			Sessions: []string{},
		}
	}

	daily := tt.dailyUsage[today]
	daily.TotalTokens += response.Usage.TotalTokens
	daily.TotalCost += cost
	daily.RequestCount++

	// Add session to daily sessions if not already present
	sessionFound := false
	for _, sid := range daily.Sessions {
		if sid == sessionID {
			sessionFound = true
			break
		}
	}
	if !sessionFound {
		daily.Sessions = append(daily.Sessions, sessionID)
	}

	// Track request timestamp for rate limiting
	tt.requestTimestamps[sessionID] = append(tt.requestTimestamps[sessionID], now)
}

// GetSessionUsage returns usage statistics for a session
func (tt *TokenTracker) GetSessionUsage(sessionID string) (*SessionUsage, error) {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	usage, ok := tt.sessionTokens[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Return a copy to prevent external modification
	usageCopy := *usage
	return &usageCopy, nil
}

// GetDailyUsage returns usage statistics for a date
func (tt *TokenTracker) GetDailyUsage(date string) (*DailyUsage, error) {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	usage, ok := tt.dailyUsage[date]
	if !ok {
		return nil, fmt.Errorf("no usage data for date: %s", date)
	}

	// Return a copy
	usageCopy := *usage
	usageCopy.Sessions = make([]string, len(usage.Sessions))
	copy(usageCopy.Sessions, usage.Sessions)

	return &usageCopy, nil
}

// GetAllSessionsUsage returns usage for all sessions
func (tt *TokenTracker) GetAllSessionsUsage() map[string]*SessionUsage {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	result := make(map[string]*SessionUsage)
	for sid, usage := range tt.sessionTokens {
		usageCopy := *usage
		result[sid] = &usageCopy
	}

	return result
}

// GetBudgetStatus returns current budget utilization
func (tt *TokenTracker) GetBudgetStatus(sessionID string) *BudgetStatus {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	status := &BudgetStatus{
		SessionID: sessionID,
		Budget:    tt.budget,
	}

	if session, ok := tt.sessionTokens[sessionID]; ok {
		status.TokensUsed = session.TotalTokens
		status.TokensRemaining = tt.budget.MaxTokensPerSession - session.TotalTokens
		status.TokenUsagePercent = float64(session.TotalTokens) / float64(tt.budget.MaxTokensPerSession) * 100

		status.CostUsed = session.TotalCost
		status.CostRemaining = tt.budget.MaxCostPerSession - session.TotalCost
		status.CostUsagePercent = (session.TotalCost / tt.budget.MaxCostPerSession) * 100

		status.RequestCount = session.RequestCount
	}

	// Add daily stats
	today := time.Now().Format("2006-01-02")
	if daily, ok := tt.dailyUsage[today]; ok {
		status.DailyCostUsed = daily.TotalCost
		status.DailyCostRemaining = tt.budget.MaxCostPerDay - daily.TotalCost
		status.DailyCostPercent = (daily.TotalCost / tt.budget.MaxCostPerDay) * 100
	}

	return status
}

// BudgetStatus represents current budget utilization
type BudgetStatus struct {
	SessionID          string      `json:"session_id"`
	Budget             TokenBudget `json:"budget"`
	TokensUsed         int         `json:"tokens_used"`
	TokensRemaining    int         `json:"tokens_remaining"`
	TokenUsagePercent  float64     `json:"token_usage_percent"`
	CostUsed           float64     `json:"cost_used"`
	CostRemaining      float64     `json:"cost_remaining"`
	CostUsagePercent   float64     `json:"cost_usage_percent"`
	DailyCostUsed      float64     `json:"daily_cost_used"`
	DailyCostRemaining float64     `json:"daily_cost_remaining"`
	DailyCostPercent   float64     `json:"daily_cost_percent"`
	RequestCount       int         `json:"request_count"`
}

// ResetSession clears usage data for a session
func (tt *TokenTracker) ResetSession(sessionID string) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	delete(tt.sessionTokens, sessionID)
	delete(tt.requestTimestamps, sessionID)
}

// CleanupOldSessions removes sessions older than the specified duration
func (tt *TokenTracker) CleanupOldSessions(maxAge time.Duration) int {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for sessionID, usage := range tt.sessionTokens {
		if now.Sub(usage.LastUpdate) > maxAge {
			delete(tt.sessionTokens, sessionID)
			delete(tt.requestTimestamps, sessionID)
			cleaned++
		}
	}

	return cleaned
}

// EstimateTokens estimates token count for a request (simplified)
func EstimateTokens(request *LLMRequest) int {
	// Simple estimation: ~4 characters per token
	totalChars := 0

	for _, msg := range request.Messages {
		totalChars += len(msg.Content)
	}

	// Add overhead for tools
	totalChars += len(request.Tools) * 200 // Rough estimate per tool

	return totalChars / 4
}

// EstimateCost estimates cost for a request based on model pricing
func EstimateCost(model string, estimatedTokens int, costPerKTokens float64) float64 {
	return float64(estimatedTokens) / 1000.0 * costPerKTokens
}
