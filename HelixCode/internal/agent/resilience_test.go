package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"dev.helix.code/internal/agent/task"
	"github.com/stretchr/testify/assert"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test-agent", 5, 2, 60*time.Second)

	assert.NotNil(t, cb)
	assert.Equal(t, "test-agent", cb.agentID)
	assert.Equal(t, CircuitBreakerClosed, cb.state)
	assert.Equal(t, 0, cb.failureCount)
	assert.Equal(t, 0, cb.successCount)
	assert.Equal(t, 5, cb.failureThreshold)
	assert.Equal(t, 2, cb.successThreshold)
	assert.Equal(t, 60*time.Second, cb.timeout)
}

func TestCircuitBreakerClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-agent", 3, 2, 60*time.Second)

	// Initially closed
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())

	// Record failures up to threshold
	for i := 0; i < 2; i++ {
		cb.recordFailure()
		assert.Equal(t, CircuitBreakerClosed, cb.GetState())
	}

	// One more failure should open the circuit
	cb.recordFailure()
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-agent", 2, 2, 10*time.Millisecond)

	// Trip the circuit
	cb.recordFailure()
	cb.recordFailure()
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(15 * time.Millisecond)

	// Next call should transition to half-open
	ctx := context.Background()
	err := cb.Call(ctx, func(ctx context.Context) error {
		return nil // Success
	})

	assert.NoError(t, err)
	// After successful call in half-open, should still be half-open
	// (need successThreshold successes to close)
	assert.Equal(t, CircuitBreakerHalfOpen, cb.GetState())
}

func TestCircuitBreakerHalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-agent", 2, 2, 10*time.Millisecond)

	// Trip the circuit
	cb.recordFailure()
	cb.recordFailure()
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(15 * time.Millisecond)

	// Transition to half-open and fail
	ctx := context.Background()
	err := cb.Call(ctx, func(ctx context.Context) error {
		return fmt.Errorf("test error")
	})

	assert.Error(t, err)
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())
}

func TestCircuitBreakerHalfOpenToClosed(t *testing.T) {
	cb := NewCircuitBreaker("test-agent", 2, 2, 10*time.Millisecond)

	// Trip the circuit
	cb.recordFailure()
	cb.recordFailure()
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(15 * time.Millisecond)

	// Transition to half-open with enough successes
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		err := cb.Call(ctx, func(ctx context.Context) error {
			return nil
		})
		assert.NoError(t, err)
	}

	// Should be closed now
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestCircuitBreakerCallWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-agent", 2, 2, 60*time.Second)

	// Trip the circuit
	cb.recordFailure()
	cb.recordFailure()
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Calls should be rejected
	ctx := context.Background()
	err := cb.Call(ctx, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker open")
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker("test-agent", 2, 2, 60*time.Second)

	// Trip the circuit
	cb.recordFailure()
	cb.recordFailure()
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Reset
	cb.Reset()

	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
	assert.Equal(t, 0, cb.failureCount)
	assert.Equal(t, 0, cb.successCount)
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	assert.NotNil(t, policy)
	assert.Equal(t, 3, policy.MaxRetries)
	assert.Equal(t, 1*time.Second, policy.InitialDelay)
	assert.Equal(t, 30*time.Second, policy.MaxDelay)
	assert.Equal(t, 2.0, policy.BackoffFactor)
}

func TestRetryPolicyShouldRetry(t *testing.T) {
	policy := DefaultRetryPolicy()

	// Nil error should not retry
	assert.False(t, policy.ShouldRetry(nil))

	// Context errors should not retry
	assert.False(t, policy.ShouldRetry(context.Canceled))
	assert.False(t, policy.ShouldRetry(context.DeadlineExceeded))

	// Other errors should retry
	assert.True(t, policy.ShouldRetry(errors.New("some error")))
	assert.True(t, policy.ShouldRetry(fmt.Errorf("test error")))
}

func TestRetryPolicyShouldRetryWithSpecificErrors(t *testing.T) {
	retryableError := errors.New("retryable")
	nonRetryableError := errors.New("non-retryable")

	policy := &RetryPolicy{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		RetryableErrors: []error{retryableError},
	}

	// Should only retry retryable error
	assert.True(t, policy.ShouldRetry(retryableError))
	assert.False(t, policy.ShouldRetry(nonRetryableError))
	assert.False(t, policy.ShouldRetry(errors.New("other error")))
}

func TestRetryPolicyGetDelay(t *testing.T) {
	policy := DefaultRetryPolicy()

	// First retry
	delay0 := policy.GetDelay(0)
	assert.Equal(t, 1*time.Second, delay0)

	// Second retry (exponential backoff)
	delay1 := policy.GetDelay(1)
	assert.Equal(t, 2*time.Second, delay1)

	// Third retry
	delay2 := policy.GetDelay(2)
	assert.Equal(t, 4*time.Second, delay2)

	// Fourth retry (should cap at MaxDelay)
	delay10 := policy.GetDelay(10)
	assert.Equal(t, 30*time.Second, delay10)
}

func TestRetrySuccess(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	attempts := 0
	err := Retry(context.Background(), policy, func(ctx context.Context) error {
		attempts++
		return nil // Success on first try
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetryFailureThenSuccess(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	attempts := 0
	err := Retry(context.Background(), policy, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil // Success on third try
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetryMaxRetriesExceeded(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	attempts := 0
	err := Retry(context.Background(), policy, func(ctx context.Context) error {
		attempts++
		return errors.New("persistent error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
	assert.Equal(t, 4, attempts) // Initial + 3 retries
}

func TestRetryContextCancellation(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:    10,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	attempts := 0
	err := Retry(ctx, policy, func(ctx context.Context) error {
		attempts++
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	// Should have attempted at least once
	assert.GreaterOrEqual(t, attempts, 1)
}

func TestRetryNonRetryableError(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	attempts := 0
	err := Retry(context.Background(), policy, func(ctx context.Context) error {
		attempts++
		return context.Canceled // Non-retryable
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 1, attempts) // Should not retry
}

// Mock agent for testing ResilientExecutor
type mockResilientAgent struct {
	id           string
	executeFunc  func(context.Context, *task.Task) (*task.Result, error)
	executeCalls int
}

func (m *mockResilientAgent) ID() string                         { return m.id }
func (m *mockResilientAgent) Type() AgentType                    { return AgentTypePlanning }
func (m *mockResilientAgent) Name() string                       { return "Mock Resilient Agent" }
func (m *mockResilientAgent) Status() AgentStatus                { return StatusIdle }
func (m *mockResilientAgent) Capabilities() []Capability         { return []Capability{} }
func (m *mockResilientAgent) CanHandle(t *task.Task) bool        { return true }
func (m *mockResilientAgent) Health() *HealthCheck               { return &HealthCheck{} }
func (m *mockResilientAgent) Shutdown(ctx context.Context) error { return nil }
func (m *mockResilientAgent) Initialize(ctx context.Context, config *AgentConfig) error {
	return nil
}

func (m *mockResilientAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	m.executeCalls++
	if m.executeFunc != nil {
		return m.executeFunc(ctx, t)
	}
	result := task.NewResult(t.ID, m.id)
	result.SetSuccess(map[string]interface{}{}, 1.0)
	return result, nil
}

func (m *mockResilientAgent) Collaborate(ctx context.Context, agents []Agent, t *task.Task) (*CollaborationResult, error) {
	return nil, nil
}

func TestResilientExecutorSuccess(t *testing.T) {
	agent := &mockResilientAgent{
		id: "test-agent",
	}

	cb := NewCircuitBreaker("test-agent", 5, 2, 60*time.Second)
	policy := &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	executor := NewResilientExecutor(agent, cb, policy)

	t1 := task.NewTask(task.TaskTypePlanning, "Test", "Test task", task.PriorityNormal)

	result, err := executor.Execute(context.Background(), t1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 1, agent.executeCalls)
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestResilientExecutorRetry(t *testing.T) {
	failCount := 0
	agent := &mockResilientAgent{
		id: "test-agent",
		executeFunc: func(ctx context.Context, t *task.Task) (*task.Result, error) {
			failCount++
			if failCount < 3 {
				return nil, errors.New("temporary failure")
			}
			result := task.NewResult(t.ID, "test-agent")
			result.SetSuccess(map[string]interface{}{}, 1.0)
			return result, nil
		},
	}

	cb := NewCircuitBreaker("test-agent", 10, 2, 60*time.Second)
	policy := &RetryPolicy{
		MaxRetries:    5,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	executor := NewResilientExecutor(agent, cb, policy)

	t1 := task.NewTask(task.TaskTypePlanning, "Test", "Test task", task.PriorityNormal)

	result, err := executor.Execute(context.Background(), t1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 3, failCount) // Failed twice, succeeded on third
}

func TestResilientExecutorCircuitBreakerTrip(t *testing.T) {
	agent := &mockResilientAgent{
		id: "test-agent",
		executeFunc: func(ctx context.Context, t *task.Task) (*task.Result, error) {
			return nil, errors.New("persistent failure")
		},
	}

	cb := NewCircuitBreaker("test-agent", 2, 2, 60*time.Second)
	policy := &RetryPolicy{
		MaxRetries:    1,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	executor := NewResilientExecutor(agent, cb, policy)

	t1 := task.NewTask(task.TaskTypePlanning, "Test", "Test task", task.PriorityNormal)

	// Execute twice to trip circuit breaker
	for i := 0; i < 2; i++ {
		result, err := executor.Execute(context.Background(), t1)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Success)
	}

	// Circuit should be open now
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Next call should be rejected by circuit breaker
	result, err := executor.Execute(context.Background(), t1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker open")
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestNewCircuitBreakerManager(t *testing.T) {
	manager := NewCircuitBreakerManager(5, 2, 60*time.Second)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.breakers)
	assert.Equal(t, 5, manager.failureThreshold)
	assert.Equal(t, 2, manager.successThreshold)
	assert.Equal(t, 60*time.Second, manager.timeout)
}

func TestCircuitBreakerManagerGetOrCreate(t *testing.T) {
	manager := NewCircuitBreakerManager(5, 2, 60*time.Second)

	// Create first breaker
	cb1 := manager.GetOrCreate("agent1")
	assert.NotNil(t, cb1)
	assert.Equal(t, "agent1", cb1.agentID)

	// Get same breaker
	cb2 := manager.GetOrCreate("agent1")
	assert.Equal(t, cb1, cb2)

	// Create different breaker
	cb3 := manager.GetOrCreate("agent2")
	assert.NotNil(t, cb3)
	assert.NotEqual(t, cb1, cb3)
	assert.Equal(t, "agent2", cb3.agentID)
}

func TestCircuitBreakerManagerReset(t *testing.T) {
	manager := NewCircuitBreakerManager(2, 2, 60*time.Second)

	cb := manager.GetOrCreate("agent1")

	// Trip the circuit
	cb.recordFailure()
	cb.recordFailure()
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Reset through manager
	manager.Reset("agent1")

	// Should be closed now
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestCircuitBreakerManagerGetState(t *testing.T) {
	manager := NewCircuitBreakerManager(2, 2, 60*time.Second)

	// Non-existent breaker should return closed
	state := manager.GetState("nonexistent")
	assert.Equal(t, CircuitBreakerClosed, state)

	// Create and trip a breaker
	cb := manager.GetOrCreate("agent1")
	cb.recordFailure()
	cb.recordFailure()

	// Should return open state
	state = manager.GetState("agent1")
	assert.Equal(t, CircuitBreakerOpen, state)
}

func TestCircuitBreakerManagerGetStats(t *testing.T) {
	manager := NewCircuitBreakerManager(2, 2, 60*time.Second)

	// Create multiple breakers with different states
	cb1 := manager.GetOrCreate("agent1")
	cb1.recordFailure()
	cb1.recordFailure()

	_ = manager.GetOrCreate("agent2")
	// Keep closed

	stats := manager.GetStats()

	assert.Len(t, stats, 2)
	assert.Equal(t, CircuitBreakerOpen, stats["agent1"])
	assert.Equal(t, CircuitBreakerClosed, stats["agent2"])
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cb := NewCircuitBreaker("test-agent", 10, 2, 60*time.Second)

	// Run multiple concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			ctx := context.Background()
			cb.Call(ctx, func(ctx context.Context) error {
				time.Sleep(1 * time.Millisecond)
				return nil
			})
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should remain closed
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestRetryBackoffTiming(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:    2,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	attempts := 0
	start := time.Now()

	err := Retry(context.Background(), policy, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("error")
		}
		return nil
	})

	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	// Should have delays: 10ms + 20ms = 30ms minimum
	assert.GreaterOrEqual(t, elapsed, 30*time.Millisecond)
}

func TestCircuitBreakerEdgeCases(t *testing.T) {
	// Test with threshold of 1
	cb := NewCircuitBreaker("test-agent", 1, 1, 10*time.Millisecond)

	cb.recordFailure()
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(15 * time.Millisecond)

	// One success should close it
	ctx := context.Background()
	err := cb.Call(ctx, func(ctx context.Context) error {
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestResilientExecutorWithNilResult(t *testing.T) {
	agent := &mockResilientAgent{
		id: "test-agent",
		executeFunc: func(ctx context.Context, t *task.Task) (*task.Result, error) {
			return nil, errors.New("execution failed")
		},
	}

	cb := NewCircuitBreaker("test-agent", 5, 2, 60*time.Second)
	policy := &RetryPolicy{
		MaxRetries:    2,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	executor := NewResilientExecutor(agent, cb, policy)

	t1 := task.NewTask(task.TaskTypePlanning, "Test", "Test task", task.PriorityNormal)

	result, err := executor.Execute(context.Background(), t1)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	// Should have retried
	assert.Equal(t, 3, agent.executeCalls) // Initial + 2 retries
}
