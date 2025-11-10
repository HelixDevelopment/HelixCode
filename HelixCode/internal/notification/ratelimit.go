package notification

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter controls the rate of notifications
type RateLimiter struct {
	maxRequests int           // Maximum requests per window
	window      time.Duration // Time window
	tokens      int           // Available tokens
	lastRefill  time.Time     // Last token refill time
	mutex       sync.Mutex    // Thread safety
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		tokens:      maxRequests,
		lastRefill:  time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (r *RateLimiter) Allow() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.refill()

	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

// Wait blocks until a token is available
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.Allow() {
			return nil
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Retry
		}
	}
}

// refill adds tokens based on elapsed time (must be called with lock held)
func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)

	if elapsed >= r.window {
		r.tokens = r.maxRequests
		r.lastRefill = now
	}
}

// GetAvailableTokens returns the current number of available tokens
func (r *RateLimiter) GetAvailableTokens() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.refill()
	return r.tokens
}

// Reset resets the rate limiter
func (r *RateLimiter) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.tokens = r.maxRequests
	r.lastRefill = time.Now()
}

// RateLimitedChannel wraps a channel with rate limiting
type RateLimitedChannel struct {
	channel     NotificationChannel
	rateLimiter *RateLimiter
	stats       *RateLimitStats
}

// RateLimitStats tracks rate limiting statistics
type RateLimitStats struct {
	TotalRequests   int64
	AllowedRequests int64
	BlockedRequests int64
	mutex           sync.Mutex
}

// NewRateLimitedChannel creates a new rate-limited channel
func NewRateLimitedChannel(channel NotificationChannel, limiter *RateLimiter) *RateLimitedChannel {
	return &RateLimitedChannel{
		channel:     channel,
		rateLimiter: limiter,
		stats:       &RateLimitStats{},
	}
}

// Send sends a notification with rate limiting
func (r *RateLimitedChannel) Send(ctx context.Context, notification *Notification) error {
	r.stats.mutex.Lock()
	r.stats.TotalRequests++
	r.stats.mutex.Unlock()

	// Wait for rate limit
	if err := r.rateLimiter.Wait(ctx); err != nil {
		r.stats.mutex.Lock()
		r.stats.BlockedRequests++
		r.stats.mutex.Unlock()
		return fmt.Errorf("rate limit wait failed: %w", err)
	}

	r.stats.mutex.Lock()
	r.stats.AllowedRequests++
	r.stats.mutex.Unlock()

	return r.channel.Send(ctx, notification)
}

// GetName returns the channel name
func (r *RateLimitedChannel) GetName() string {
	return r.channel.GetName()
}

// IsEnabled returns whether the channel is enabled
func (r *RateLimitedChannel) IsEnabled() bool {
	return r.channel.IsEnabled()
}

// GetConfig returns the channel configuration
func (r *RateLimitedChannel) GetConfig() map[string]interface{} {
	config := r.channel.GetConfig()
	config["rate_limit_max_requests"] = r.rateLimiter.maxRequests
	config["rate_limit_window"] = r.rateLimiter.window.String()
	return config
}

// GetStats returns rate limiting statistics
func (r *RateLimitedChannel) GetStats() RateLimitStats {
	r.stats.mutex.Lock()
	defer r.stats.mutex.Unlock()
	return *r.stats
}

// ResetStats resets the statistics
func (r *RateLimitedChannel) ResetStats() {
	r.stats.mutex.Lock()
	defer r.stats.mutex.Unlock()
	r.stats = &RateLimitStats{}
}

// ChannelRateLimits defines common rate limits for different channels
var ChannelRateLimits = map[string]*RateLimiter{
	"slack":    NewRateLimiter(1, 1*time.Second),   // 1 per second
	"discord":  NewRateLimiter(5, 5*time.Second),   // 5 per 5 seconds
	"telegram": NewRateLimiter(30, 1*time.Second),  // 30 per second
	"email":    NewRateLimiter(10, 1*time.Minute),  // 10 per minute
	"webhook":  NewRateLimiter(100, 1*time.Minute), // 100 per minute
}

// GetDefaultRateLimiter returns a default rate limiter for a channel type
func GetDefaultRateLimiter(channelType string) *RateLimiter {
	if limiter, ok := ChannelRateLimits[channelType]; ok {
		// Return a copy to avoid shared state
		return NewRateLimiter(limiter.maxRequests, limiter.window)
	}
	// Default: 10 requests per second
	return NewRateLimiter(10, 1*time.Second)
}
