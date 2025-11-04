package notification

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retry attempts
	InitialBackoff time.Duration // Initial backoff duration
	MaxBackoff     time.Duration // Maximum backoff duration
	BackoffFactor  float64       // Multiplier for exponential backoff
	Enabled        bool          // Whether retries are enabled
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
		Enabled:        true,
	}
}

// RetryableChannel wraps a channel with retry logic
type RetryableChannel struct {
	channel NotificationChannel
	config  RetryConfig
	stats   *RetryStats
}

// RetryStats tracks retry statistics
type RetryStats struct {
	TotalAttempts   int64
	SuccessfulSends int64
	FailedSends     int64
	Retries         int64
}

// NewRetryableChannel creates a new retryable channel
func NewRetryableChannel(channel NotificationChannel, config RetryConfig) *RetryableChannel {
	return &RetryableChannel{
		channel: channel,
		config:  config,
		stats:   &RetryStats{},
	}
}

// Send sends a notification with retry logic
func (r *RetryableChannel) Send(ctx context.Context, notification *Notification) error {
	if !r.config.Enabled {
		return r.channel.Send(ctx, notification)
	}

	var lastErr error
	backoff := r.config.InitialBackoff

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		r.stats.TotalAttempts++

		// Try to send
		err := r.channel.Send(ctx, notification)
		if err == nil {
			r.stats.SuccessfulSends++
			if attempt > 0 {
				log.Printf("Notification sent successfully after %d retries (channel: %s)",
					attempt, r.channel.GetName())
			}
			return nil
		}

		lastErr = err

		// Check if we should retry
		if attempt == r.config.MaxRetries {
			r.stats.FailedSends++
			log.Printf("Notification failed after %d attempts (channel: %s): %v",
				attempt+1, r.channel.GetName(), err)
			break
		}

		r.stats.Retries++

		// Log retry attempt
		log.Printf("Notification send failed (attempt %d/%d, channel: %s), retrying in %v: %v",
			attempt+1, r.config.MaxRetries+1, r.channel.GetName(), backoff, err)

		// Wait with backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled during retry: %w", ctx.Err())
		case <-time.After(backoff):
			// Continue to next attempt
		}

		// Calculate next backoff with exponential increase
		backoff = time.Duration(float64(backoff) * r.config.BackoffFactor)
		if backoff > r.config.MaxBackoff {
			backoff = r.config.MaxBackoff
		}
	}

	return fmt.Errorf("notification failed after %d attempts: %w", r.config.MaxRetries+1, lastErr)
}

// SendWithCustomRetry sends a notification with custom retry configuration
func (r *RetryableChannel) SendWithCustomRetry(ctx context.Context, notification *Notification, maxRetries int) error {
	originalMaxRetries := r.config.MaxRetries
	r.config.MaxRetries = maxRetries
	defer func() {
		r.config.MaxRetries = originalMaxRetries
	}()

	return r.Send(ctx, notification)
}

// GetName returns the channel name
func (r *RetryableChannel) GetName() string {
	return r.channel.GetName()
}

// IsEnabled returns whether the channel is enabled
func (r *RetryableChannel) IsEnabled() bool {
	return r.channel.IsEnabled()
}

// GetConfig returns the channel configuration
func (r *RetryableChannel) GetConfig() map[string]interface{} {
	config := r.channel.GetConfig()
	config["retry_enabled"] = r.config.Enabled
	config["retry_max_retries"] = r.config.MaxRetries
	config["retry_initial_backoff"] = r.config.InitialBackoff.String()
	config["retry_max_backoff"] = r.config.MaxBackoff.String()
	return config
}

// GetStats returns retry statistics
func (r *RetryableChannel) GetStats() RetryStats {
	return *r.stats
}

// ResetStats resets retry statistics
func (r *RetryableChannel) ResetStats() {
	r.stats = &RetryStats{}
}

// CalculateBackoff calculates backoff duration for a given attempt
func (r *RetryConfig) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return r.InitialBackoff
	}

	backoff := float64(r.InitialBackoff) * math.Pow(r.BackoffFactor, float64(attempt))
	if backoff > float64(r.MaxBackoff) {
		return r.MaxBackoff
	}

	return time.Duration(backoff)
}

// IsRetryableError determines if an error should trigger a retry
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common retryable error patterns
	errStr := err.Error()

	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"503",
		"502",
		"504",
		"rate limit",
		"too many requests",
	}

	for _, pattern := range retryablePatterns {
		if stringContains(errStr, pattern) {
			return true
		}
	}

	return false
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
