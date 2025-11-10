package hooks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Executor executes hooks for events
type Executor struct {
	maxConcurrent int                // Maximum concurrent async executions
	semaphore     chan struct{}      // Semaphore for controlling concurrency
	wg            sync.WaitGroup     // Wait group for async executions
	results       []*ExecutionResult // Results of recent executions
	resultsMu     sync.RWMutex       // Mutex for results
	maxResults    int                // Maximum results to keep
	onComplete    []ResultCallback   // Callbacks on completion
	onError       []ResultCallback   // Callbacks on error
}

// ResultCallback is called when a hook execution completes
type ResultCallback func(*ExecutionResult)

// NewExecutor creates a new hook executor
func NewExecutor() *Executor {
	return &Executor{
		maxConcurrent: 10,
		semaphore:     make(chan struct{}, 10),
		results:       make([]*ExecutionResult, 0),
		maxResults:    100,
		onComplete:    make([]ResultCallback, 0),
		onError:       make([]ResultCallback, 0),
	}
}

// NewExecutorWithLimit creates a new executor with concurrency limit
func NewExecutorWithLimit(maxConcurrent int) *Executor {
	e := NewExecutor()
	e.maxConcurrent = maxConcurrent
	e.semaphore = make(chan struct{}, maxConcurrent)
	return e
}

// Execute executes a single hook
func (e *Executor) Execute(ctx context.Context, hook *Hook, event *Event) *ExecutionResult {
	result := NewExecutionResult(hook)

	// Check if hook should execute
	if !hook.ShouldExecute(event) {
		result.Skip()
		e.addResult(result)
		return result
	}

	// Execute asynchronously or synchronously
	if hook.Async {
		e.executeAsync(ctx, hook, event, result)
	} else {
		e.executeSync(ctx, hook, event, result)
	}

	return result
}

// ExecuteAll executes all hooks for an event
func (e *Executor) ExecuteAll(ctx context.Context, hooks []*Hook, event *Event) []*ExecutionResult {
	// Sort hooks by priority (highest first)
	sortedHooks := make([]*Hook, len(hooks))
	copy(sortedHooks, hooks)
	e.sortByPriority(sortedHooks)

	results := make([]*ExecutionResult, 0, len(sortedHooks))

	// Execute each hook
	for _, hook := range sortedHooks {
		result := e.Execute(ctx, hook, event)
		results = append(results, result)

		// Check if context was canceled
		if ctx.Err() != nil {
			// Cancel remaining hooks
			for i := len(results); i < len(sortedHooks); i++ {
				cancelResult := NewExecutionResult(sortedHooks[i])
				cancelResult.Cancel()
				results = append(results, cancelResult)
			}
			break
		}
	}

	return results
}

// ExecuteSync executes hooks synchronously and waits for all to complete
func (e *Executor) ExecuteSync(ctx context.Context, hooks []*Hook, event *Event) []*ExecutionResult {
	// Sort by priority
	sortedHooks := make([]*Hook, len(hooks))
	copy(sortedHooks, hooks)
	e.sortByPriority(sortedHooks)

	results := make([]*ExecutionResult, 0, len(sortedHooks))

	for _, hook := range sortedHooks {
		result := NewExecutionResult(hook)

		if !hook.ShouldExecute(event) {
			result.Skip()
			e.addResult(result)
			results = append(results, result)
			continue
		}

		e.executeSync(ctx, hook, event, result)
		results = append(results, result)

		// Check context
		if ctx.Err() != nil {
			break
		}
	}

	return results
}

// ExecuteAndWait executes hooks and waits for all (including async) to complete
func (e *Executor) ExecuteAndWait(ctx context.Context, hooks []*Hook, event *Event) []*ExecutionResult {
	results := e.ExecuteAll(ctx, hooks, event)
	e.Wait()
	return results
}

// Wait waits for all async executions to complete
func (e *Executor) Wait() {
	e.wg.Wait()
}

// executeSync executes a hook synchronously
func (e *Executor) executeSync(ctx context.Context, hook *Hook, event *Event, result *ExecutionResult) {
	result.Status = StatusRunning

	// Execute handler
	err := hook.Execute(ctx, event)

	result.Complete(err)
	e.addResult(result)
	e.triggerCallbacks(result)
}

// executeAsync executes a hook asynchronously
func (e *Executor) executeAsync(ctx context.Context, hook *Hook, event *Event, result *ExecutionResult) {
	e.wg.Add(1)

	go func() {
		defer e.wg.Done()

		// Acquire semaphore slot
		e.semaphore <- struct{}{}
		defer func() { <-e.semaphore }()

		result.Status = StatusRunning

		// Execute handler
		err := hook.Execute(ctx, event)

		result.Complete(err)
		e.addResult(result)
		e.triggerCallbacks(result)
	}()
}

// sortByPriority sorts hooks by priority (highest first)
func (e *Executor) sortByPriority(hooks []*Hook) {
	// Simple bubble sort by priority
	for i := 0; i < len(hooks)-1; i++ {
		for j := i + 1; j < len(hooks); j++ {
			if hooks[j].Priority > hooks[i].Priority {
				hooks[i], hooks[j] = hooks[j], hooks[i]
			}
		}
	}
}

// addResult adds a result to the results list
func (e *Executor) addResult(result *ExecutionResult) {
	e.resultsMu.Lock()
	defer e.resultsMu.Unlock()

	e.results = append(e.results, result)

	// Trim if exceeds max
	if e.maxResults > 0 && len(e.results) > e.maxResults {
		e.results = e.results[len(e.results)-e.maxResults:]
	}
}

// GetResults returns recent execution results
func (e *Executor) GetResults(n int) []*ExecutionResult {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	if n <= 0 || n > len(e.results) {
		n = len(e.results)
	}

	results := make([]*ExecutionResult, n)
	copy(results, e.results[len(e.results)-n:])
	return results
}

// GetAllResults returns all execution results
func (e *Executor) GetAllResults() []*ExecutionResult {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	results := make([]*ExecutionResult, len(e.results))
	copy(results, e.results)
	return results
}

// GetResultsByStatus returns results with specific status
func (e *Executor) GetResultsByStatus(status HookStatus) []*ExecutionResult {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	results := make([]*ExecutionResult, 0)
	for _, result := range e.results {
		if result.Status == status {
			results = append(results, result)
		}
	}
	return results
}

// ClearResults clears all stored results
func (e *Executor) ClearResults() {
	e.resultsMu.Lock()
	defer e.resultsMu.Unlock()

	e.results = make([]*ExecutionResult, 0)
}

// GetStatistics returns execution statistics
func (e *Executor) GetStatistics() *ExecutorStatistics {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	stats := &ExecutorStatistics{
		TotalExecutions: len(e.results),
		ByStatus:        make(map[HookStatus]int),
	}

	var totalDuration time.Duration
	for _, result := range e.results {
		stats.ByStatus[result.Status]++
		totalDuration += result.Duration
	}

	if stats.TotalExecutions > 0 {
		stats.AverageDuration = totalDuration / time.Duration(stats.TotalExecutions)
	}

	stats.SuccessRate = 0
	if stats.TotalExecutions > 0 {
		completed := stats.ByStatus[StatusCompleted]
		stats.SuccessRate = float64(completed) / float64(stats.TotalExecutions)
	}

	return stats
}

// OnComplete registers a callback for completed executions
func (e *Executor) OnComplete(callback ResultCallback) {
	e.onComplete = append(e.onComplete, callback)
}

// OnError registers a callback for failed executions
func (e *Executor) OnError(callback ResultCallback) {
	e.onError = append(e.onError, callback)
}

// triggerCallbacks triggers registered callbacks
func (e *Executor) triggerCallbacks(result *ExecutionResult) {
	// Trigger completion callbacks
	for _, callback := range e.onComplete {
		callback(result)
	}

	// Trigger error callbacks if failed
	if result.Status == StatusFailed {
		for _, callback := range e.onError {
			callback(result)
		}
	}
}

// SetMaxConcurrent sets the maximum concurrent async executions
func (e *Executor) SetMaxConcurrent(max int) {
	e.maxConcurrent = max
	e.semaphore = make(chan struct{}, max)
}

// SetMaxResults sets the maximum number of results to keep
func (e *Executor) SetMaxResults(max int) {
	e.resultsMu.Lock()
	defer e.resultsMu.Unlock()

	e.maxResults = max

	// Trim if needed
	if max > 0 && len(e.results) > max {
		e.results = e.results[len(e.results)-max:]
	}
}

// ExecutorStatistics contains execution statistics
type ExecutorStatistics struct {
	TotalExecutions int                // Total number of executions
	ByStatus        map[HookStatus]int // Count by status
	AverageDuration time.Duration      // Average execution duration
	SuccessRate     float64            // Success rate (0.0 - 1.0)
}

// String returns a string representation of the statistics
func (s *ExecutorStatistics) String() string {
	return fmt.Sprintf("Total: %d, Success: %.1f%%, Avg: %.2fms",
		s.TotalExecutions,
		s.SuccessRate*100,
		float64(s.AverageDuration.Microseconds())/1000)
}

// ExecuteWithTimeout executes a hook with a specific timeout
func (e *Executor) ExecuteWithTimeout(hook *Hook, event *Event, timeout time.Duration) *ExecutionResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return e.Execute(ctx, hook, event)
}

// ExecuteWithDeadline executes a hook with a specific deadline
func (e *Executor) ExecuteWithDeadline(hook *Hook, event *Event, deadline time.Time) *ExecutionResult {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	return e.Execute(ctx, hook, event)
}
