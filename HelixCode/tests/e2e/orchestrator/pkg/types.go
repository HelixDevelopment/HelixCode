package pkg

import (
	"context"
	"time"
)

// TestStatus represents the status of a test execution
type TestStatus string

const (
	StatusPending  TestStatus = "pending"
	StatusRunning  TestStatus = "running"
	StatusPassed   TestStatus = "passed"
	StatusFailed   TestStatus = "failed"
	StatusSkipped  TestStatus = "skipped"
	StatusTimedOut TestStatus = "timeout"
)

// Priority levels for test execution
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// TestCase represents a single test case
type TestCase struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Priority    Priority               `json:"priority"`
	Timeout     time.Duration          `json:"timeout"`
	Tags        []string               `json:"tags"`
	Setup       func(ctx context.Context) error
	Execute     func(ctx context.Context) error
	Teardown    func(ctx context.Context) error
	Metadata    map[string]interface{} `json:"metadata"`
}

// TestResult represents the result of a test execution
type TestResult struct {
	TestID     string                 `json:"test_id"`
	TestName   string                 `json:"test_name"`
	Status     TestStatus             `json:"status"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Error      error                  `json:"error,omitempty"`
	ErrorMsg   string                 `json:"error_msg,omitempty"`
	Output     string                 `json:"output,omitempty"`
	Assertions []AssertionResult      `json:"assertions,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AssertionResult represents a single assertion result
type AssertionResult struct {
	Description string `json:"description"`
	Passed      bool   `json:"passed"`
	Expected    string `json:"expected,omitempty"`
	Actual      string `json:"actual,omitempty"`
	Message     string `json:"message,omitempty"`
}

// TestSuite represents a collection of test cases
type TestSuite struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Tests       []*TestCase `json:"tests"`
	Setup       func(ctx context.Context) error
	Teardown    func(ctx context.Context) error
}

// ExecutionConfig configures test execution
type ExecutionConfig struct {
	Parallel       bool          `json:"parallel"`
	MaxConcurrency int           `json:"max_concurrency"`
	Timeout        time.Duration `json:"timeout"`
	RetryCount     int           `json:"retry_count"`
	RetryDelay     time.Duration `json:"retry_delay"`
	FailFast       bool          `json:"fail_fast"`
	Verbose        bool          `json:"verbose"`
}

// DefaultExecutionConfig returns default execution configuration
func DefaultExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		Parallel:       true,
		MaxConcurrency: 10,
		Timeout:        30 * time.Minute,
		RetryCount:     0,
		RetryDelay:     5 * time.Second,
		FailFast:       false,
		Verbose:        false,
	}
}

// TestReport represents a complete test execution report
type TestReport struct {
	SuiteName   string                 `json:"suite_name"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	TotalTests  int                    `json:"total_tests"`
	Passed      int                    `json:"passed"`
	Failed      int                    `json:"failed"`
	Skipped     int                    `json:"skipped"`
	TimedOut    int                    `json:"timed_out"`
	SuccessRate float64                `json:"success_rate"`
	Results     []*TestResult          `json:"results"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CalculateStats calculates statistics for the test report
func (tr *TestReport) CalculateStats() {
	tr.TotalTests = len(tr.Results)
	tr.Passed = 0
	tr.Failed = 0
	tr.Skipped = 0
	tr.TimedOut = 0

	for _, result := range tr.Results {
		switch result.Status {
		case StatusPassed:
			tr.Passed++
		case StatusFailed:
			tr.Failed++
		case StatusSkipped:
			tr.Skipped++
		case StatusTimedOut:
			tr.TimedOut++
		}
	}

	if tr.TotalTests > 0 {
		tr.SuccessRate = float64(tr.Passed) / float64(tr.TotalTests) * 100
	}
}
