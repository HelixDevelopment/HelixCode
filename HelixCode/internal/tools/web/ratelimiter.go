package web

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for web requests
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	limits   map[string]RateLimit
}

// RateLimit defines rate limit configuration
type RateLimit struct {
	RequestsPerSecond float64
	Burst             int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		limits: map[string]RateLimit{
			"google":     {RequestsPerSecond: 10, Burst: 20},
			"bing":       {RequestsPerSecond: 5, Burst: 10},
			"duckduckgo": {RequestsPerSecond: 2, Burst: 5},
			"default":    {RequestsPerSecond: 1, Burst: 3},
		},
	}
}

// Wait waits for rate limit
func (rl *RateLimiter) Wait(ctx context.Context, key string) error {
	limiter := rl.getLimiter(key)
	return limiter.Wait(ctx)
}

// Allow checks if a request is allowed without waiting
func (rl *RateLimiter) Allow(key string) bool {
	limiter := rl.getLimiter(key)
	return limiter.Allow()
}

// Reserve reserves a token for future use
func (rl *RateLimiter) Reserve(key string) *rate.Reservation {
	limiter := rl.getLimiter(key)
	return limiter.Reserve()
}

// getLimiter gets or creates limiter for key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, ok := rl.limiters[key]
	rl.mu.RUnlock()

	if ok {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, ok := rl.limiters[key]; ok {
		return limiter
	}

	// Create new limiter
	limit := rl.limits[key]
	if limit.RequestsPerSecond == 0 {
		limit = rl.limits["default"]
	}

	limiter = rate.NewLimiter(rate.Limit(limit.RequestsPerSecond), limit.Burst)
	rl.limiters[key] = limiter

	return limiter
}

// SetLimit sets rate limit for a key
func (rl *RateLimiter) SetLimit(key string, limit RateLimit) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.limits[key] = limit
	// Remove existing limiter so it will be recreated with new limit
	delete(rl.limiters, key)
}

// GetLimit gets rate limit for a key
func (rl *RateLimiter) GetLimit(key string) RateLimit {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limit, ok := rl.limits[key]
	if !ok {
		return rl.limits["default"]
	}
	return limit
}

// RemoveLimit removes rate limit for a key
func (rl *RateLimiter) RemoveLimit(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.limits, key)
	delete(rl.limiters, key)
}

// Clear clears all limiters
func (rl *RateLimiter) Clear() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.limiters = make(map[string]*rate.Limiter)
}
