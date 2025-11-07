package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/tests/e2e/orchestrator/pkg"
)

// Executor handles test execution with support for parallel execution and retries
type Executor struct {
	config *pkg.ExecutionConfig
	mu     sync.RWMutex
}

// NewExecutor creates a new test executor
func NewExecutor(config *pkg.ExecutionConfig) *Executor {
	if config == nil {
		config = &pkg.ExecutionConfig{
			Parallel:       true,
			MaxConcurrency: 10,
			Timeout:        30 * time.Minute,
			RetryCount:     0,
			RetryDelay:     1 * time.Second,
			FailFast:       false,
			Verbose:        false,
		}
	}
	
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	
	return &Executor{
		config: config,
	}
}

// ExecuteSuite executes a test suite and returns a comprehensive report
func (e *Executor) ExecuteSuite(ctx context.Context, suite *pkg.TestSuite) *pkg.TestReport {
	report := &pkg.TestReport{
		SuiteName:  suite.Name,
		StartTime:  time.Now(),
		TotalTests: len(suite.Tests),
		Results:    make([]*pkg.TestResult, 0, len(suite.Tests)),
	}

	// Run suite setup if defined
	if suite.Setup != nil {
		if err := suite.Setup(ctx); err != nil {
			// Suite setup failed, mark all tests as skipped
			for _, test := range suite.Tests {
				report.Results = append(report.Results, &pkg.TestResult{
					TestID:    test.ID,
					TestName:  test.Name,
					Status:    pkg.StatusSkipped,
					ErrorMsg:  fmt.Sprintf("Suite setup failed: %v", err),
					StartTime: time.Now(),
					EndTime:   time.Now(),
				})
				report.Skipped++
			}
			report.EndTime = time.Now()
			report.Duration = report.EndTime.Sub(report.StartTime)
			e.calculateSuccessRate(report)
			return report
		}
	}

	// Execute tests
	var results []*pkg.TestResult
	if e.config.Parallel {
		results = e.executeParallel(ctx, suite.Tests)
	} else {
		results = e.executeSequential(ctx, suite.Tests)
	}

	report.Results = results

	// Run suite teardown if defined
	if suite.Teardown != nil {
		if err := suite.Teardown(ctx); err != nil {
			if e.config.Verbose {
				fmt.Printf("Suite teardown failed: %v\n", err)
			}
		}
	}

	// Calculate statistics
	for _, result := range results {
		switch result.Status {
		case pkg.StatusPassed:
			report.Passed++
		case pkg.StatusFailed:
			report.Failed++
		case pkg.StatusSkipped:
			report.Skipped++
		case pkg.StatusTimedOut:
			report.TimedOut++
		}
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)
	e.calculateSuccessRate(report)

	return report
}

// executeSequential executes tests one by one
func (e *Executor) executeSequential(ctx context.Context, tests []*pkg.TestCase) []*pkg.TestResult {
	results := make([]*pkg.TestResult, 0, len(tests))

	for _, test := range tests {
		result := e.Execute(ctx, test)
		results = append(results, result)

		// Check fail-fast mode
		if e.config.FailFast && (result.Status == pkg.StatusFailed || result.Status == pkg.StatusTimedOut) {
			// Skip remaining tests
			for i := len(results); i < len(tests); i++ {
				results = append(results, &pkg.TestResult{
					TestID:    tests[i].ID,
					TestName:  tests[i].Name,
					Status:    pkg.StatusSkipped,
					ErrorMsg:  "Skipped due to fail-fast mode",
					StartTime: time.Now(),
					EndTime:   time.Now(),
				})
			}
			break
		}
	}

	return results
}

// executeParallel executes tests in parallel with concurrency control
func (e *Executor) executeParallel(ctx context.Context, tests []*pkg.TestCase) []*pkg.TestResult {
	results := make([]*pkg.TestResult, len(tests))
	sem := make(chan struct{}, e.config.MaxConcurrency)
	var wg sync.WaitGroup
	failedCh := make(chan bool, 1)
	stopCh := make(chan struct{})

	for i, test := range tests {
		wg.Add(1)
		go func(idx int, tc *pkg.TestCase) {
			defer wg.Done()

			// Check if we should stop due to fail-fast
			select {
			case <-stopCh:
				results[idx] = &pkg.TestResult{
					TestID:    tc.ID,
					TestName:  tc.Name,
					Status:    pkg.StatusSkipped,
					ErrorMsg:  "Skipped due to fail-fast mode",
					StartTime: time.Now(),
					EndTime:   time.Now(),
				}
				return
			default:
			}

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Execute test
			result := e.Execute(ctx, tc)
			results[idx] = result

			// Check fail-fast mode
			if e.config.FailFast && (result.Status == pkg.StatusFailed || result.Status == pkg.StatusTimedOut) {
				select {
				case failedCh <- true:
					close(stopCh)
				default:
				}
			}
		}(i, test)
	}

	wg.Wait()
	close(sem)
	return results
}

// Execute runs a single test with retry logic
func (e *Executor) Execute(ctx context.Context, test *pkg.TestCase) *pkg.TestResult {
	result := &pkg.TestResult{
		TestID:     test.ID,
		TestName:   test.Name,
		Status:     pkg.StatusPending,
		StartTime:  time.Now(),
		Assertions: make([]pkg.AssertionResult, 0),
		Metadata:   test.Metadata,
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
			if e.config.Verbose {
				fmt.Printf("Retrying test %s (attempt %d/%d)\n", test.Name, attempt+1, maxAttempts)
			}
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
				result.Error = testCtx.Err()
				result.ErrorMsg = fmt.Sprintf("Test timed out after %v", timeout)
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result
			}
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

// executeOnce executes a single test attempt
func (e *Executor) executeOnce(ctx context.Context, test *pkg.TestCase) error {
	// Run setup
	if test.Setup != nil {
		if err := test.Setup(ctx); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
	}

	// Run main test execution
	var execErr error
	if test.Execute != nil {
		execErr = test.Execute(ctx)
	}

	// Always run teardown even if execute failed
	if test.Teardown != nil {
		if err := test.Teardown(ctx); err != nil {
			if execErr != nil {
				return fmt.Errorf("execute failed: %v, teardown also failed: %w", execErr, err)
			}
			return fmt.Errorf("teardown failed: %w", err)
		}
	}

	return execErr
}

// calculateSuccessRate calculates the success rate for the report
func (e *Executor) calculateSuccessRate(report *pkg.TestReport) {
	if report.TotalTests == 0 {
		report.SuccessRate = 0
		return
	}
	report.SuccessRate = float64(report.Passed) / float64(report.TotalTests) * 100
}
