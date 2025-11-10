//go:build test_runner
// +build test_runner

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"dev.helix.code/tests/automation"
	"dev.helix.code/tests/e2e"
	"dev.helix.code/tests/integration"
	"dev.helix.code/tests/unit"
	"github.com/fatih/color"
)

// Test runner configuration
type TestConfig struct {
	RunUnit        bool
	RunIntegration bool
	RunE2E         bool
	RunAutomation  bool
	RunSecurity    bool
	RunAll         bool
	SkipExpensive  bool
	SkipHardware   bool
	Verbose        bool
	TestTimeout    time.Duration
	Parallel       int
	OutputDir      string
}

// Test results
type TestResults struct {
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Suites   []SuiteResult
}

type SuiteResult struct {
	Name    string
	Passed  int
	Failed  int
	Skipped int
	Error   error
}

func main() {
	config := parseFlags()

	if config.Verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	fmt.Println("🧪 HelixCode Local LLM - Comprehensive Test Suite")
	fmt.Println(strings.Repeat("=", 60))

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Run test suites
	results := &TestResults{}
	startTime := time.Now()

	if config.RunAll || config.RunSecurity {
		results.Suites = append(results.Suites, runSecurityTests(config))
	}

	if config.RunAll || config.RunUnit {
		results.Suites = append(results.Suites, runUnitTests(config))
	}

	if config.RunAll || config.RunIntegration {
		results.Suites = append(results.Suites, runIntegrationTests(config))
	}

	if config.RunAll || config.RunE2E {
		results.Suites = append(results.Suites, runE2ETests(config))
	}

	if config.RunAll || config.RunAutomation {
		results.Suites = append(results.Suites, runAutomationTests(config))
	}

	// Calculate final results
	results.Duration = time.Since(startTime)
	for _, suite := range results.Suites {
		results.Total += suite.Passed + suite.Failed + suite.Skipped
		results.Passed += suite.Passed
		results.Failed += suite.Failed
		results.Skipped += suite.Skipped
	}

	// Print final report
	printFinalReport(results)

	// Exit with appropriate code
	if results.Failed > 0 {
		os.Exit(1)
	}
}

func parseFlags() *TestConfig {
	config := &TestConfig{
		TestTimeout: 10 * time.Minute,
		Parallel:    runtime.NumCPU(),
		OutputDir:   "test-results",
	}

	flag.BoolVar(&config.RunUnit, "unit", false, "Run unit tests")
	flag.BoolVar(&config.RunIntegration, "integration", false, "Run integration tests")
	flag.BoolVar(&config.RunE2E, "e2e", false, "Run end-to-end tests")
	flag.BoolVar(&config.RunAutomation, "automation", false, "Run hardware automation tests")
	flag.BoolVar(&config.RunSecurity, "security", false, "Run security tests")
	flag.BoolVar(&config.RunAll, "all", false, "Run all test suites")
	flag.BoolVar(&config.SkipExpensive, "skip-expensive", false, "Skip expensive tests")
	flag.BoolVar(&config.SkipHardware, "skip-hardware", false, "Skip hardware-dependent tests")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.DurationVar(&config.TestTimeout, "timeout", 10*time.Minute, "Test timeout")
	flag.IntVar(&config.Parallel, "parallel", runtime.NumCPU(), "Parallel test execution")
	flag.StringVar(&config.OutputDir, "output", "test-results", "Output directory for results")

	flag.Parse()

	return config
}

func runSecurityTests(config *TestConfig) SuiteResult {
	color.Cyan("\n🔒 Running Security Tests")
	fmt.Println(strings.Repeat("-", 40))

	suite := SuiteResult{Name: "Security"}

	// Set environment for security tests
	env := os.Environ()
	if config.SkipExpensive {
		env = append(env, "SKIP_EXPENSIVE_TESTS=true")
	}

	cmd := exec.Command("go", "test", "-v", "-race", "./security/")
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		suite.Error = fmt.Errorf("Security tests failed: %v", err)
		suite.Failed++
		fmt.Printf("❌ Security tests failed: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
	} else {
		suite.Passed++
		fmt.Printf("✅ Security tests passed\n")
	}

	// Save results
	saveTestResults(config, "security", string(output))

	return suite
}

func runUnitTests(config *TestConfig) SuiteResult {
	color.Cyan("\n🧪 Running Unit Tests")
	fmt.Println(strings.Repeat("-", 40))

	suite := SuiteResult{Name: "Unit"}

	args := []string{
		"test", "-v", "-race", "-count=1",
		fmt.Sprintf("-timeout=%v", config.TestTimeout),
		"-parallel", fmt.Sprintf("%d", config.Parallel),
		"./tests/unit/",
	}

	cmd := exec.Command("go", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		suite.Error = fmt.Errorf("Unit tests failed: %v", err)
		suite.Failed++
		fmt.Printf("❌ Unit tests failed: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
	} else {
		suite.Passed++
		fmt.Printf("✅ Unit tests passed\n")
	}

	// Parse output for detailed results
	suite.Passed, suite.Failed, suite.Skipped = parseTestOutput(string(output))

	// Save results
	saveTestResults(config, "unit", string(output))

	return suite
}

func runIntegrationTests(config *TestConfig) SuiteResult {
	color.Cyan("\n🔗 Running Integration Tests")
	fmt.Println(strings.Repeat("-", 40))

	suite := SuiteResult{Name: "Integration"}

	// Set environment for integration tests
	env := os.Environ()
	if config.SkipExpensive {
		env = append(env, "SKIP_EXPENSIVE_TESTS=true")
	}
	if config.SkipHardware {
		env = append(env, "SKIP_HARDWARE_TESTS=true")
	}
	if testing.Short() {
		env = append(env, "TESTING_SHORT=true")
	}

	args := []string{
		"test", "-v",
		fmt.Sprintf("-timeout=%v", config.TestTimeout*2), // Longer timeout for integration
		"./tests/integration/",
	}

	cmd := exec.Command("go", args...)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		suite.Error = fmt.Errorf("Integration tests failed: %v", err)
		suite.Failed++
		fmt.Printf("❌ Integration tests failed: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
	} else {
		suite.Passed++
		fmt.Printf("✅ Integration tests passed\n")
	}

	// Parse output for detailed results
	suite.Passed, suite.Failed, suite.Skipped = parseTestOutput(string(output))

	// Save results
	saveTestResults(config, "integration", string(output))

	return suite
}

func runE2ETests(config *TestConfig) SuiteResult {
	color.Cyan("\n🎯 Running End-to-End Tests")
	fmt.Println(strings.Repeat("-", 40))

	suite := SuiteResult{Name: "E2E"}

	// Set environment for E2E tests
	env := os.Environ()
	if config.SkipExpensive {
		env = append(env, "SKIP_EXPENSIVE_TESTS=true")
	}
	if config.SkipHardware {
		env = append(env, "SKIP_REAL_EXECUTION=true")
		env = append(env, "SKIP_HARDWARE_TESTS=true")
	}

	args := []string{
		"test", "-v",
		fmt.Sprintf("-timeout=%v", config.TestTimeout*3), // Longer timeout for E2E
		"./tests/e2e/",
	}

	cmd := exec.Command("go", args...)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		suite.Error = fmt.Errorf("E2E tests failed: %v", err)
		suite.Failed++
		fmt.Printf("❌ E2E tests failed: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
	} else {
		suite.Passed++
		fmt.Printf("✅ E2E tests passed\n")
	}

	// Parse output for detailed results
	suite.Passed, suite.Failed, suite.Skipped = parseTestOutput(string(output))

	// Save results
	saveTestResults(config, "e2e", string(output))

	return suite
}

func runAutomationTests(config *TestConfig) SuiteResult {
	color.Cyan("\n🤖 Running Hardware Automation Tests")
	fmt.Println(strings.Repeat("-", 40))

	suite := SuiteResult{Name: "Automation"}

	// Set environment for automation tests
	env := os.Environ()
	if config.SkipHardware {
		env = append(env, "SKIP_HARDWARE_TESTS=true")
		env = append(env, "SKIP_REAL_EXECUTION=true")
		env = append(env, "SKIP_BENCHMARKS=true")
	}

	args := []string{
		"test", "-v",
		fmt.Sprintf("-timeout=%v", config.TestTimeout*4), // Longest timeout for automation
		"./tests/automation/",
	}

	cmd := exec.Command("go", args...)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		suite.Error = fmt.Errorf("Automation tests failed: %v", err)
		suite.Failed++
		fmt.Printf("❌ Automation tests failed: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
	} else {
		suite.Passed++
		fmt.Printf("✅ Automation tests passed\n")
	}

	// Parse output for detailed results
	suite.Passed, suite.Failed, suite.Skipped = parseTestOutput(string(output))

	// Save results
	saveTestResults(config, "automation", string(output))

	return suite
}

func parseTestOutput(output string) (passed, failed, skipped int) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "PASS") {
			passed++
		} else if strings.Contains(line, "FAIL") {
			failed++
		} else if strings.Contains(line, "SKIP") {
			skipped++
		}
	}
	return
}

func saveTestResults(config *TestConfig, suiteName, output string) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.log", suiteName, timestamp)
	filepath := filepath.Join(config.OutputDir, filename)

	err := os.WriteFile(filepath, []byte(output), 0644)
	if err != nil {
		log.Printf("Failed to save test results: %v", err)
	}

	// Also save as latest
	latestPath := filepath.Join(config.OutputDir, fmt.Sprintf("%s-latest.log", suiteName))
	err = os.WriteFile(latestPath, []byte(output), 0644)
	if err != nil {
		log.Printf("Failed to save latest test results: %v", err)
	}
}

func printFinalReport(results *TestResults) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	color.Cyan("📊 FINAL TEST REPORT")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("Duration: %v\n", results.Duration)
	fmt.Printf("Total Tests: %d\n", results.Total)
	fmt.Printf("Passed: %s%d%s\n", color.GreenString(""), results.Passed, color.ResetString())
	fmt.Printf("Failed: %s%d%s\n", color.RedString(""), results.Failed, color.ResetString())
	fmt.Printf("Skipped: %s%d%s\n", color.YellowString(""), results.Skipped, color.ResetString())

	fmt.Printf("\nSuccess Rate: %.1f%%\n", float64(results.Passed)/float64(results.Total)*100)

	// Print suite results
	fmt.Println("\nSuite Results:")
	for _, suite := range results.Suites {
		status := "✅"
		if suite.Failed > 0 {
			status = "❌"
		} else if suite.Skipped > 0 {
			status = "⚠️"
		}

		fmt.Printf("%s %s: %d passed, %d failed, %d skipped\n",
			status, suite.Name, suite.Passed, suite.Failed, suite.Skipped)

		if suite.Error != nil {
			fmt.Printf("   Error: %v\n", suite.Error)
		}
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	if results.Failed > 0 {
		color.Red("❌ TESTS FAILED - See logs for details")
	} else {
		color.Green("✅ ALL TESTS PASSED")
	}
	fmt.Println(strings.Repeat("=", 60))

	// Print recommendations
	printRecommendations(results)
}

func printRecommendations(results *TestResults) {
	fmt.Println("\n💡 Recommendations:")

	if results.Failed > 0 {
		fmt.Println("• Check test logs in output directory for failure details")
		fmt.Println("• Ensure all dependencies are installed")
		fmt.Println("• Verify environment configuration")
	}

	if results.Skipped > 0 {
		fmt.Printf("• %d tests were skipped - run without --skip-expensive to include them\n", results.Skipped)
	}

	if results.Duration > 5*time.Minute {
		fmt.Println("• Tests took significant time - consider parallel execution")
	}

	fmt.Printf("• Test results saved to: %s\n", config.OutputDir)
}

// Pre-flight checks

func preflightChecks() error {
	// Check Go version
	cmd := exec.Command("go", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Go not installed or not in PATH")
	}

	fmt.Printf("✅ %s", string(output))

	// Check required dependencies
	requiredCmds := []string{"git"}
	for _, cmdName := range requiredCmds {
		cmd := exec.Command(cmdName, "--version")
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("Required dependency not found: %s", cmdName)
		}
		fmt.Printf("✅ %s available\n", cmdName)
	}

	// Check test dependencies
	testDeps := []string{"github.com/stretchr/testify"}
	for _, dep := range testDeps {
		cmd := exec.Command("go", "list", dep)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("Test dependency not found: %s", dep)
		}
		fmt.Printf("✅ %s available\n", dep)
	}

	return nil
}

// Init function for pre-flight checks
func init() {
	if len(os.Args) > 1 && os.Args[1] == "--preflight" {
		fmt.Println("🔍 Running pre-flight checks...")

		if err := preflightChecks(); err != nil {
			fmt.Printf("❌ Pre-flight checks failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ All pre-flight checks passed")
		os.Exit(0)
	}
}
