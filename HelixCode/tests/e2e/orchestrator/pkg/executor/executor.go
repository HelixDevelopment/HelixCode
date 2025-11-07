package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/tests/e2e/orchestrator/pkg"
)

// Executor executes test cases
type Executor struct {
	config *pkg.ExecutionConfig
	mu     sync.RWMutex
}

// NewExecutor creates a new test executor
func NewExecutor(config *pkg.ExecutionConfig) *Executor {
	if config == nil {
		config = pkg.DefaultExecutionConfig()
	}
	return &Executor{
		config: config,
	}
}

// Execute runs a single test case
func (e *Executor) Execute(ctx context.Context, test *pkg.TestCase) *pkg.TestResult {
	result := &pkg.TestResult{
		TestID:    test.ID,
		TestName:  test.Name,
		Status:    pkg.StatusRunning,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Apply timeout
	timeout := test.Timeout
	if timeout == 0 {
		timeout = e.config.Timeout
	}

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute with retry logic
	var lastErr error
	maxAttempts := e.config.RetryCount + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(e.config.RetryDelay)
		}

		err := e.executeOnce(testCtx, test)
		if err == nil {
			result.Status = pkg.StatusPassed
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}

		lastErr = err

		// Check if context was cancelled or timed out
		select {
		case <-testCtx.Done():
			if testCtx.Err() == context.DeadlineExceeded {
				result.Status = pkg.StatusTimedOut
			} else {
				result.Status = pkg.StatusFailed
			}
			result.Error = testCtx.Err()
			result.ErrorMsg = fmt.Sprintf("Test interrupted: %v", testCtx.Err())
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		default:
		}
	}

	// All attempts failed
	result.Status = pkg.StatusFailed
	result.Error = lastErr
	if lastErr != nil {
		result.ErrorMsg = lastErr.Error()
	}
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// executeOnce executes a test case once (setup -> execute -> teardown)
func (e *Executor) executeOnce(ctx context.Context, test *pkg.TestCase) error {
	// Setup
	if test.Setup != nil {
		if err := test.Setup(ctx); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
	}

	// Always run teardown
	if test.Teardown != nil {
		defer func() {
			if teardownErr := test.Teardown(context.Background()); teardownErr != nil {
				// Log teardown error but don't fail the test
				fmt.Printf("Warning: Teardown failed for test %s: %v\n", test.ID, teardownErr)
			}
		}()
	}

	// Execute
	if test.Execute == nil {
		return fmt.Errorf("test has no execute function")
	}

	return test.Execute(ctx)
}

// ExecuteSuite runs all tests in a suite
func (e *Executor) ExecuteSuite(ctx context.Context, suite *pkg.TestSuite) *pkg.TestReport {
	report := &pkg.TestReport{
		SuiteName: suite.Name,
		StartTime: time.Now(),
		Results:   make([]*pkg.TestResult, 0, len(suite.Tests)),
		Metadata:  make(map[string]interface{}),
	}

	// Suite setup
	if suite.Setup != nil {
		if err := suite.Setup(ctx); err != nil {
			// Mark all tests as failed if suite setup fails
			for _, test := range suite.Tests {
				result := &pkg.TestResult{
					TestID:    test.ID,
					TestName:  test.Name,
					Status:    pkg.StatusFailed,
					StartTime: time.Now(),
					EndTime:   time.Now(),
					Error:     err,
					ErrorMsg:  fmt.Sprintf("Suite setup failed: %v", err),
				}
				report.Results = append(report.Results, result)
			}
			report.EndTime = time.Now()
			report.Duration = report.EndTime.Sub(report.StartTime)
			report.CalculateStats()
			return report
		}
	}

	// Suite teardown
	if suite.Teardown != nil {
		defer func() {
			if err := suite.Teardown(context.Background()); err != nil {
				fmt.Printf("Warning: Suite teardown failed: %v\n", err)
			}
		}()
	}

	// Execute tests
	if e.config.Parallel {
		report.Results = e.executeParallel(ctx, suite.Tests)
	} else {
		report.Results = e.executeSequential(ctx, suite.Tests)
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)
	report.CalculateStats()

	return report
}

// executeSequential runs tests one by one
func (e *Executor) executeSequential(ctx context.Context, tests []*pkg.TestCase) []*pkg.TestResult {
	results := make([]*pkg.TestResult, 0, len(tests))

	for _, test := range tests {
		result := e.Execute(ctx, test)
		results = append(results, result)

		// Fail fast if enabled
		if e.config.FailFast && result.Status == pkg.StatusFailed {
			// Mark remaining tests as skipped
			for i := len(results); i < len(tests); i++ {
				results = append(results, &pkg.TestResult{
					TestID:   tests[i].ID,
					TestName: tests[i].Name,
					Status:   pkg.StatusSkipped,
					ErrorMsg: "Skipped due to fail-fast",
				})
			}
			break
		}
	}

	return results
}

// executeParallel runs tests concurrently
func (e *Executor) executeParallel(ctx context.Context, tests []*pkg.TestCase) []*pkg.TestResult {
	results := make([]*pkg.TestResult, len(tests))
	sem := make(chan struct{}, e.config.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	failedOnce := false

	for i, test := range tests {
		wg.Add(1)

		go func(idx int, t *pkg.TestCase) {
			defer wg.Done()

			// Check fail-fast
			if e.config.FailFast {
				mu.Lock()
				if failedOnce {
					results[idx] = &pkg.TestResult{
						TestID:   t.ID,
						TestName: t.Name,
						Status:   pkg.StatusSkipped,
						ErrorMsg: "Skipped due to fail-fast",
					}
					mu.Unlock()
					return
				}
				mu.Unlock()
			}

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Execute test
			result := e.Execute(ctx, t)

			mu.Lock()
			results[idx] = result
			if result.Status == pkg.StatusFailed {
				failedOnce = true
			}
			mu.Unlock()
		}(i, test)
	}

	wg.Wait()
	return results
}
