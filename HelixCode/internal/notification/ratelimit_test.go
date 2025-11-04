package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(10, 1*time.Second)

	assert.NotNil(t, limiter)
	assert.Equal(t, 10, limiter.maxRequests)
	assert.Equal(t, 1*time.Second, limiter.window)
	assert.Equal(t, 10, limiter.tokens)
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(3, 1*time.Second)

	// Should allow first 3 requests
	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())

	// Should block 4th request
	assert.False(t, limiter.Allow())
}

func TestRateLimiter_Refill(t *testing.T) {
	limiter := NewRateLimiter(2, 100*time.Millisecond)

	// Consume all tokens
	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())
	assert.False(t, limiter.Allow())

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should allow again after refill
	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())
}

func TestRateLimiter_GetAvailableTokens(t *testing.T) {
	limiter := NewRateLimiter(5, 1*time.Second)

	assert.Equal(t, 5, limiter.GetAvailableTokens())

	limiter.Allow()
	assert.Equal(t, 4, limiter.GetAvailableTokens())

	limiter.Allow()
	limiter.Allow()
	assert.Equal(t, 2, limiter.GetAvailableTokens())
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter := NewRateLimiter(3, 1*time.Second)

	// Consume tokens
	limiter.Allow()
	limiter.Allow()
	assert.Equal(t, 1, limiter.GetAvailableTokens())

	// Reset
	limiter.Reset()
	assert.Equal(t, 3, limiter.GetAvailableTokens())
}

func TestRateLimiter_Wait(t *testing.T) {
	limiter := NewRateLimiter(2, 200*time.Millisecond)

	// Consume all tokens
	limiter.Allow()
	limiter.Allow()

	// Wait should block and succeed after refill
	start := time.Now()
	err := limiter.Wait(context.Background())
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
}

func TestRateLimiter_WaitContextCanceled(t *testing.T) {
	limiter := NewRateLimiter(1, 10*time.Second)

	// Consume token
	limiter.Allow()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait should fail due to context cancellation
	err := limiter.Wait(ctx)
	assert.Error(t, err)
}

func TestRateLimiter_Concurrent(t *testing.T) {
	limiter := NewRateLimiter(100, 1*time.Second)

	var wg sync.WaitGroup
	allowed := 0
	blocked := 0
	var mu sync.Mutex

	// Try 200 concurrent requests (should allow 100, block 100)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow() {
				mu.Lock()
				allowed++
				mu.Unlock()
			} else {
				mu.Lock()
				blocked++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, 100, allowed)
	assert.Equal(t, 100, blocked)
}

func TestNewRateLimitedChannel(t *testing.T) {
	mockCh := &retryMockChannel{}
	limiter := NewRateLimiter(10, 1*time.Second)
	rateLimited := NewRateLimitedChannel(mockCh, limiter)

	assert.NotNil(t, rateLimited)
	assert.NotNil(t, rateLimited.channel)
	assert.NotNil(t, rateLimited.rateLimiter)
	assert.NotNil(t, rateLimited.stats)
}

func TestRateLimitedChannel_Send(t *testing.T) {
	sendCount := 0
	mockCh := &retryMockChannel{
		sendFunc: func(ctx context.Context, notif *Notification) error {
			sendCount++
			return nil
		},
	}

	limiter := NewRateLimiter(3, 1*time.Second)
	rateLimited := NewRateLimitedChannel(mockCh, limiter)

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	// First 3 should succeed
	for i := 0; i < 3; i++ {
		err := rateLimited.Send(context.Background(), notif)
		assert.NoError(t, err)
	}

	assert.Equal(t, 3, sendCount)

	stats := rateLimited.GetStats()
	assert.Equal(t, int64(3), stats.TotalRequests)
	assert.Equal(t, int64(3), stats.AllowedRequests)
	assert.Equal(t, int64(0), stats.BlockedRequests)
}

func TestRateLimitedChannel_SendBlocked(t *testing.T) {
	mockCh := &retryMockChannel{
		sendFunc: func(ctx context.Context, notif *Notification) error {
			return nil
		},
	}

	limiter := NewRateLimiter(1, 10*time.Second)
	rateLimited := NewRateLimitedChannel(mockCh, limiter)

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	// First should succeed
	err := rateLimited.Send(context.Background(), notif)
	assert.NoError(t, err)

	// Second should block and fail with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = rateLimited.Send(ctx, notif)
	assert.Error(t, err)

	stats := rateLimited.GetStats()
	assert.Equal(t, int64(2), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.AllowedRequests)
	assert.Equal(t, int64(1), stats.BlockedRequests)
}

func TestRateLimitedChannel_GetName(t *testing.T) {
	mockCh := &retryMockChannel{
		getNameFunc: func() string {
			return "test-channel"
		},
	}

	limiter := NewRateLimiter(10, 1*time.Second)
	rateLimited := NewRateLimitedChannel(mockCh, limiter)

	assert.Equal(t, "test-channel", rateLimited.GetName())
}

func TestRateLimitedChannel_IsEnabled(t *testing.T) {
	mockCh := &retryMockChannel{
		isEnabledFunc: func() bool {
			return true
		},
	}

	limiter := NewRateLimiter(10, 1*time.Second)
	rateLimited := NewRateLimitedChannel(mockCh, limiter)

	assert.True(t, rateLimited.IsEnabled())
}

func TestRateLimitedChannel_GetConfig(t *testing.T) {
	mockCh := &retryMockChannel{
		getConfigFunc: func() map[string]interface{} {
			return map[string]interface{}{
				"webhook": "https://example.com",
			}
		},
	}

	limiter := NewRateLimiter(10, 1*time.Second)
	rateLimited := NewRateLimitedChannel(mockCh, limiter)

	config := rateLimited.GetConfig()
	assert.Equal(t, "https://example.com", config["webhook"])
	assert.Equal(t, 10, config["rate_limit_max_requests"])
	assert.NotEmpty(t, config["rate_limit_window"])
}

func TestRateLimitedChannel_ResetStats(t *testing.T) {
	mockCh := &retryMockChannel{
		sendFunc: func(ctx context.Context, notif *Notification) error {
			return nil
		},
	}

	limiter := NewRateLimiter(10, 1*time.Second)
	rateLimited := NewRateLimitedChannel(mockCh, limiter)

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	rateLimited.Send(context.Background(), notif)

	stats := rateLimited.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)

	rateLimited.ResetStats()

	stats = rateLimited.GetStats()
	assert.Equal(t, int64(0), stats.TotalRequests)
}

func TestGetDefaultRateLimiter(t *testing.T) {
	tests := []struct {
		channelType    string
		expectedMax    int
		expectedWindow time.Duration
	}{
		{"slack", 1, 1 * time.Second},
		{"discord", 5, 5 * time.Second},
		{"telegram", 30, 1 * time.Second},
		{"email", 10, 1 * time.Minute},
		{"webhook", 100, 1 * time.Minute},
		{"unknown", 10, 1 * time.Second}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.channelType, func(t *testing.T) {
			limiter := GetDefaultRateLimiter(tt.channelType)
			assert.Equal(t, tt.expectedMax, limiter.maxRequests)
			assert.Equal(t, tt.expectedWindow, limiter.window)
		})
	}
}

func TestChannelRateLimits(t *testing.T) {
	assert.NotNil(t, ChannelRateLimits["slack"])
	assert.NotNil(t, ChannelRateLimits["discord"])
	assert.NotNil(t, ChannelRateLimits["telegram"])
	assert.NotNil(t, ChannelRateLimits["email"])
	assert.NotNil(t, ChannelRateLimits["webhook"])
}
