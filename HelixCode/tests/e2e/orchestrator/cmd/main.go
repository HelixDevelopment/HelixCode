package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"dev.helix.code/tests/e2e/orchestrator/pkg"
	"dev.helix.code/tests/e2e/orchestrator/pkg/executor"
	"dev.helix.code/tests/e2e/orchestrator/pkg/reporter"
	"dev.helix.code/tests/e2e/orchestrator/pkg/scheduler"
)

var (
	// Global flags
	parallel       bool
	maxConcurrency int
	timeout        time.Duration
	retryCount     int
	failFast       bool
	verbose        bool
	outputFormat   string
	outputFile     string
	testIDs        []string
	testTags       []string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "e2e-orchestrator",
		Short: "E2E Test Orchestrator for HelixCode",
		Long:  `A powerful test orchestrator for running end-to-end tests with parallel execution, scheduling, and reporting.`,
	}

	// Run command
	runCmd := &cobra.Command{
		Use:   "run [test-suite]",
		Short: "Run test suite",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runTests,
	}

	runCmd.Flags().BoolVarP(&parallel, "parallel", "p", true, "Run tests in parallel")
	runCmd.Flags().IntVarP(&maxConcurrency, "concurrency", "c", 10, "Maximum concurrent tests")
	runCmd.Flags().DurationVarP(&timeout, "timeout", "t", 30*time.Minute, "Global timeout")
	runCmd.Flags().IntVarP(&retryCount, "retry", "r", 0, "Retry count for failed tests")
	runCmd.Flags().BoolVarP(&failFast, "fail-fast", "f", false, "Stop on first failure")
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	runCmd.Flags().StringVarP(&outputFormat, "format", "o", "json", "Output format (json, junit)")
	runCmd.Flags().StringVar(&outputFile, "output", "", "Output file path")
	runCmd.Flags().StringSliceVar(&testIDs, "tests", []string{}, "Specific test IDs to run")
	runCmd.Flags().StringSliceVar(&testTags, "tags", []string{}, "Filter tests by tags")

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available tests",
		RunE:  listTests,
	}

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("E2E Test Orchestrator v1.0.0")
		},
	}

	rootCmd.AddCommand(runCmd, listCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTests(cmd *cobra.Command, args []string) error {
	// Create test suite
	suite := createSampleTestSuite()

	// Apply filters
	sched := scheduler.NewScheduler()
	tests := suite.Tests

	if len(testIDs) > 0 {
		tests = sched.FilterByIDs(tests, testIDs)
	}

	if len(testTags) > 0 {
		tests = sched.FilterByTags(tests, testTags)
	}

	// Schedule tests by priority
	tests = sched.Schedule(tests)
	suite.Tests = tests

	if len(tests) == 0 {
		return fmt.Errorf("no tests to run")
	}

	// Create execution config
	config := &pkg.ExecutionConfig{
		Parallel:       parallel,
		MaxConcurrency: maxConcurrency,
		Timeout:        timeout,
		RetryCount:     retryCount,
		FailFast:       failFast,
		Verbose:        verbose,
	}

	// Execute tests
	exec := executor.NewExecutor(config)
	ctx := context.Background()

	fmt.Printf("Running %d tests...\n", len(tests))
	report := exec.ExecuteSuite(ctx, suite)

	// Generate report
	var rep reporter.Reporter
	switch strings.ToLower(outputFormat) {
	case "junit":
		rep = reporter.NewReporter(reporter.FormatJUnit)
	default:
		rep = reporter.NewReporter(reporter.FormatJSON)
	}

	data, err := rep.Generate(report)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Output report
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
		fmt.Printf("\nReport written to: %s\n", outputFile)
	} else {
		fmt.Println(string(data))
	}

	// Print summary
	printSummary(report)

	// Exit with non-zero if tests failed
	if report.Failed > 0 || report.TimedOut > 0 {
		os.Exit(1)
	}

	return nil
}

func listTests(cmd *cobra.Command, args []string) error {
	suite := createSampleTestSuite()

	fmt.Printf("Test Suite: %s\n", suite.Name)
	fmt.Printf("Total Tests: %d\n\n", len(suite.Tests))

	for _, test := range suite.Tests {
		fmt.Printf("ID: %s\n", test.ID)
		fmt.Printf("  Name: %s\n", test.Name)
		fmt.Printf("  Description: %s\n", test.Description)
		fmt.Printf("  Priority: %d\n", test.Priority)
		fmt.Printf("  Tags: %v\n", test.Tags)
		fmt.Println()
	}

	return nil
}

func printSummary(report *pkg.TestReport) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Suite:        %s\n", report.SuiteName)
	fmt.Printf("Duration:     %v\n", report.Duration)
	fmt.Printf("Total Tests:  %d\n", report.TotalTests)
	fmt.Printf("Passed:       %d\n", report.Passed)
	fmt.Printf("Failed:       %d\n", report.Failed)
	fmt.Printf("Skipped:      %d\n", report.Skipped)
	fmt.Printf("Timed Out:    %d\n", report.TimedOut)
	fmt.Printf("Success Rate: %.2f%%\n", report.SuccessRate)
	fmt.Println(strings.Repeat("=", 60))
}

// createSampleTestSuite creates a sample test suite for demonstration
func createSampleTestSuite() *pkg.TestSuite {
	return &pkg.TestSuite{
		Name:        "Sample E2E Test Suite",
		Description: "Demonstration test suite for the orchestrator",
		Tests: []*pkg.TestCase{
			{
				ID:          "TC-001",
				Name:        "Basic Health Check",
				Description: "Verify system health check endpoint",
				Priority:    pkg.PriorityCritical,
				Timeout:     5 * time.Second,
				Tags:        []string{"smoke", "health"},
				Execute: func(ctx context.Context) error {
					// Simulate health check
					time.Sleep(100 * time.Millisecond)
					return nil
				},
			},
			{
				ID:          "TC-002",
				Name:        "Service Discovery",
				Description: "Test service discovery mechanism",
				Priority:    pkg.PriorityHigh,
				Timeout:     10 * time.Second,
				Tags:        []string{"integration", "discovery"},
				Execute: func(ctx context.Context) error {
					// Simulate discovery test
					time.Sleep(200 * time.Millisecond)
					return nil
				},
			},
			{
				ID:          "TC-003",
				Name:        "Database Connection",
				Description: "Verify database connectivity",
				Priority:    pkg.PriorityHigh,
				Timeout:     10 * time.Second,
				Tags:        []string{"integration", "database"},
				Execute: func(ctx context.Context) error {
					// Simulate database test
					time.Sleep(150 * time.Millisecond)
					return nil
				},
			},
			{
				ID:          "TC-004",
				Name:        "LLM Provider Test",
				Description: "Test LLM provider connectivity",
				Priority:    pkg.PriorityNormal,
				Timeout:     15 * time.Second,
				Tags:        []string{"integration", "llm"},
				Execute: func(ctx context.Context) error {
					// Simulate LLM test
					time.Sleep(300 * time.Millisecond)
					return nil
				},
			},
			{
				ID:          "TC-005",
				Name:        "Worker Pool Test",
				Description: "Test distributed worker pool",
				Priority:    pkg.PriorityNormal,
				Timeout:     20 * time.Second,
				Tags:        []string{"integration", "workers"},
				Execute: func(ctx context.Context) error {
					// Simulate worker test
					time.Sleep(250 * time.Millisecond)
					return nil
				},
			},
		},
	}
}

func init() {
	// Ensure output directory exists if output file is specified
	cobra.OnInitialize(func() {
		if outputFile != "" {
			dir := filepath.Dir(outputFile)
			if dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to create output directory: %v\n", err)
				}
			}
		}
	})
}
