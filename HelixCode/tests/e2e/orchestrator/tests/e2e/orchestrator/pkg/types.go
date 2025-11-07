package pkg

import (
	"context"
	"time"
)

// TestStatus represents the status of a test
type TestStatus string

const (
	StatusPending  TestStatus = "pending"
	StatusRunning  TestStatus = "running"
	StatusPassed   TestStatus = "passed"
	StatusFailed   TestStatus = "failed"
	StatusSkipped  TestStatus = "skipped"
	StatusTimedOut TestStatus = "timeout"
)

// Priority represents test priority level
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// TestCase represents a single test case
type TestCase struct {
	ID          string
	Name        string
	Description string
	Priority    Priority
	Timeout     time.Duration
	Tags        []string
	
	// Lifecycle hooks
	Setup    func(ctx context.Context) error
	Execute  func(ctx context.Context) error
	Teardown func(ctx context.Context) error
	
	// Metadata for custom data
	Metadata map[string]interface{}
}

// TestResult represents the result of a test execution
type TestResult struct {
	TestID     string
	TestName   string
	Status     TestStatus
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Error      error
	ErrorMsg   string
	Output     string
	Assertions []AssertionResult
	Metadata   map[string]interface{}
}

// TestSuite represents a collection of tests
type TestSuite struct {
	Name        string
	Description string
	Tests       []*TestCase
	Setup       func(ctx context.Context) error
	Teardown    func(ctx context.Context) error
}

// ExecutionConfig contains configuration for test execution
type ExecutionConfig struct {
	Parallel       bool
	MaxConcurrency int
	Timeout        time.Duration
	RetryCount     int
	RetryDelay     time.Duration
	FailFast       bool
	Verbose        bool
}

// TestReport represents a comprehensive test execution report
type TestReport struct {
	SuiteName   string
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	TotalTests  int
	Passed      int
	Failed      int
	Skipped     int
	TimedOut    int
	SuccessRate float64
	Results     []*TestResult
}

// AssertionResult represents the result of an assertion
type AssertionResult struct {
	Description string
	Passed      bool
	Expected    string
	Actual      string
	Message     string
}
