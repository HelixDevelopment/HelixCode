package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/agent/task"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"   // Normal operation
	CircuitBreakerOpen     CircuitBreakerState = "open"     // Blocking requests
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open" // Testing if service recovered
)

// CircuitBreaker implements the circuit breaker pattern for agent resilience
type CircuitBreaker struct {
	agentID         string
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	mu              sync.RWMutex

	// Configuration
	failureThreshold int           // Number of failures before opening
	successThreshold int           // Number of successes in half-open before closing
	timeout          time.Duration // Time to wait before moving to half-open
}

// NewCircuitBreaker creates a new circuit breaker for an agent
func NewCircuitBreaker(agentID string, failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		agentID:          agentID,
		state:            CircuitBreakerClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// Call executes a function through the circuit breaker
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	switch state {
	case CircuitBreakerOpen:
		// Check if enough time has passed to try half-open
		cb.mu.RLock()
		timeSinceFailure := time.Since(cb.lastFailureTime)
		cb.mu.RUnlock()

		if timeSinceFailure >= cb.timeout {
			// Try half-open
			cb.mu.Lock()
			cb.state = CircuitBreakerHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
		} else {
			return fmt.Errorf("circuit breaker open for agent %s", cb.agentID)
		}
	}

	// Execute function
	err := fn(ctx)

	// Record result
	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()
	cb.successCount = 0 // Reset success count

	if cb.state == CircuitBreakerHalfOpen {
		// Failure in half-open -> back to open
		cb.state = CircuitBreakerOpen
	} else if cb.failureCount >= cb.failureThreshold {
		// Too many failures -> open
		cb.state = CircuitBreakerOpen
	}
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0 // Reset failure count
	cb.successCount++

	if cb.state == CircuitBreakerHalfOpen && cb.successCount >= cb.successThreshold {
		// Enough successes in half-open -> close
		cb.state = CircuitBreakerClosed
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitBreakerClosed
	cb.failureCount = 0
	cb.successCount = 0
}

// RetryPolicy defines how to retry failed operations
type RetryPolicy struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []error
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// ShouldRetry determines if an error is retryable
func (rp *RetryPolicy) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// If specific retryable errors are defined, check against them
	if len(rp.RetryableErrors) > 0 {
		for _, retryableErr := range rp.RetryableErrors {
			if errors.Is(err, retryableErr) {
				return true
			}
		}
		return false
	}

	// By default, retry on most errors except context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return true
}

// GetDelay calculates the delay for a retry attempt using exponential backoff
func (rp *RetryPolicy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rp.InitialDelay
	}

	// Calculate exponential backoff
	delay := float64(rp.InitialDelay)
	for i := 0; i < attempt; i++ {
		delay *= rp.BackoffFactor
		if delay > float64(rp.MaxDelay) {
			return rp.MaxDelay
		}
	}

	return time.Duration(delay)
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, policy *RetryPolicy, fn func(context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// Execute function
		err := fn(ctx)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if !policy.ShouldRetry(err) {
			return err // Non-retryable error
		}

		// Check if we've exhausted retries
		if attempt >= policy.MaxRetries {
			break
		}

		// Calculate delay
		delay := policy.GetDelay(attempt)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// ResilientExecutor wraps agent execution with resilience patterns
type ResilientExecutor struct {
	agent          Agent
	circuitBreaker *CircuitBreaker
	retryPolicy    *RetryPolicy
}

// NewResilientExecutor creates a new resilient executor
func NewResilientExecutor(agent Agent, circuitBreaker *CircuitBreaker, retryPolicy *RetryPolicy) *ResilientExecutor {
	return &ResilientExecutor{
		agent:          agent,
		circuitBreaker: circuitBreaker,
		retryPolicy:    retryPolicy,
	}
}

// Execute executes a task with resilience patterns (circuit breaker + retry)
func (re *ResilientExecutor) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	var result *task.Result
	var execErr error

	// Execute through circuit breaker
	cbErr := re.circuitBreaker.Call(ctx, func(ctx context.Context) error {
		// Execute with retry logic
		retryErr := Retry(ctx, re.retryPolicy, func(ctx context.Context) error {
			var err error
			result, err = re.agent.Execute(ctx, t)
			return err
		})
		execErr = retryErr
		return retryErr
	})

	if cbErr != nil {
		// Circuit breaker rejected the call
		failedResult := task.NewResult(t.ID, re.agent.ID())
		failedResult.SetFailure(cbErr)
		return failedResult, cbErr
	}

	return result, execErr
}

// CircuitBreakerManager manages circuit breakers for all agents
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex

	// Default configuration
	failureThreshold int
	successThreshold int
	timeout          time.Duration
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers:         make(map[string]*CircuitBreaker),
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// GetOrCreate gets or creates a circuit breaker for an agent
func (cbm *CircuitBreakerManager) GetOrCreate(agentID string) *CircuitBreaker {
	cbm.mu.RLock()
	cb, exists := cbm.breakers[agentID]
	cbm.mu.RUnlock()

	if exists {
		return cb
	}

	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Double-check after acquiring write lock
	cb, exists = cbm.breakers[agentID]
	if exists {
		return cb
	}

	// Create new circuit breaker
	cb = NewCircuitBreaker(agentID, cbm.failureThreshold, cbm.successThreshold, cbm.timeout)
	cbm.breakers[agentID] = cb
	return cb
}

// Reset resets a circuit breaker for an agent
func (cbm *CircuitBreakerManager) Reset(agentID string) {
	cbm.mu.RLock()
	cb, exists := cbm.breakers[agentID]
	cbm.mu.RUnlock()

	if exists {
		cb.Reset()
	}
}

// GetState returns the state of a circuit breaker
func (cbm *CircuitBreakerManager) GetState(agentID string) CircuitBreakerState {
	cbm.mu.RLock()
	cb, exists := cbm.breakers[agentID]
	cbm.mu.RUnlock()

	if !exists {
		return CircuitBreakerClosed // No breaker = closed state
	}

	return cb.GetState()
}

// GetStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetStats() map[string]CircuitBreakerState {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	stats := make(map[string]CircuitBreakerState)
	for agentID, cb := range cbm.breakers {
		stats[agentID] = cb.GetState()
	}
	return stats
}
