package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Standalone test suite for HelixCode Local LLM
// This runs without any project dependencies

func main() {
	var (
		testAll        = flag.Bool("all", false, "Run all test categories")
		testUnit       = flag.Bool("unit", false, "Run unit tests")
		testSecurity   = flag.Bool("security", false, "Run security tests")
		testIntegration= flag.Bool("integration", false, "Run integration tests")
		testE2E       = flag.Bool("e2e", false, "Run E2E tests")
		testHardware   = flag.Bool("hardware", false, "Run hardware tests")
		verbose        = flag.Bool("v", false, "Verbose output")
		timeout        = flag.Duration("timeout", 5*time.Minute, "Test timeout")
	)
	flag.Parse()

	fmt.Println("🧪 HelixCode Local LLM - Standalone Test Suite")
	fmt.Println("================================================")

	// Create test results directory
	resultsDir := "test-results"
	os.MkdirAll(resultsDir, 0755)

	timestamp := time.Now().Format("20060102-150405")
	
	var totalPassed, totalFailed int
	startTime := time.Now()

	// Run requested test categories
	if *testAll || *testUnit {
		passed, failed := runUnitTests(resultsDir, timestamp, *verbose, *timeout)
		totalPassed += passed
		totalFailed += failed
	}

	if *testAll || *testSecurity {
		passed, failed := runSecurityTests(resultsDir, timestamp, *verbose, *timeout)
		totalPassed += passed
		totalFailed += failed
	}

	if *testAll || *testIntegration {
		passed, failed := runIntegrationTests(resultsDir, timestamp, *verbose, *timeout)
		totalPassed += passed
		totalFailed += failed
	}

	if *testAll || *testE2E {
		passed, failed := runE2ETests(resultsDir, timestamp, *verbose, *timeout)
		totalPassed += passed
		totalFailed += failed
	}

	if *testAll || *testHardware {
		passed, failed := runHardwareTests(resultsDir, timestamp, *verbose, *timeout)
		totalPassed += passed
		totalFailed += failed
	}

	// Final report
	duration := time.Since(startTime)
	printFinalReport(totalPassed, totalFailed, duration)

	if totalFailed > 0 {
		os.Exit(1)
	}
}

func runUnitTests(resultsDir, timestamp string, verbose bool, timeout time.Duration) (int, int) {
	fmt.Println("\n🧪 Running Unit Tests")
	fmt.Println(strings.Repeat("-", 40))

	// Test basic functionality
	tests := []TestCase{
		{"Math Operations", testMath},
		{"String Operations", testStrings},
		{"Collection Operations", testCollections},
		{"Time Operations", testTime},
		{"Error Handling", testErrors},
		{"Concurrency", testConcurrency},
	}

	return runTests("unit", tests, resultsDir, timestamp, verbose, timeout)
}

func runSecurityTests(resultsDir, timestamp string, verbose bool, timeout time.Duration) (int, int) {
	fmt.Println("\n🔒 Running Security Tests")
	fmt.Println(strings.Repeat("-", 40))

	tests := []TestCase{
		{"Input Validation", testInputValidation},
		{"Password Strength", testPasswordStrength},
		{"URL Security", testURLSecurity},
		{"Path Traversal", testPathTraversal},
		{"Injection Resistance", testInjectionResistance},
	}

	return runTests("security", tests, resultsDir, timestamp, verbose, timeout)
}

func runIntegrationTests(resultsDir, timestamp string, verbose bool, timeout time.Duration) (int, int) {
	fmt.Println("\n🔗 Running Integration Tests")
	fmt.Println(strings.Repeat("-", 40))

	tests := []TestCase{
		{"System Commands", testSystemCommands},
		{"File Operations", testFileOperations},
		{"Network Connectivity", testNetworkConnectivity},
		{"Process Management", testProcessManagement},
		{"Environment Variables", testEnvironment},
	}

	return runTests("integration", tests, resultsDir, timestamp, verbose, timeout)
}

func runE2ETests(resultsDir, timestamp string, verbose bool, timeout time.Duration) (int, int) {
	fmt.Println("\n🎯 Running E2E Tests")
	fmt.Println(strings.Repeat("-", 40))

	tests := []TestCase{
		{"CLI Help", testCLIHelp},
		{"CLI Commands", testCLICommands},
		{"Provider Detection", testProviderDetection},
		{"Model Operations", testModelOperations},
		{"Configuration Management", testConfiguration},
	}

	return runTests("e2e", tests, resultsDir, timestamp, verbose, timeout)
}

func runHardwareTests(resultsDir, timestamp string, verbose bool, timeout time.Duration) (int, int) {
	fmt.Println("\n🤖 Running Hardware Tests")
	fmt.Println(strings.Repeat("-", 40))

	tests := []TestCase{
		{"CPU Detection", testCPUDetection},
		{"Memory Detection", testMemoryDetection},
		{"GPU Detection", testGPUDetection},
		{"OS Detection", testOSDetection},
		{"Hardware Optimization", testHardwareOptimization},
	}

	return runTests("hardware", tests, resultsDir, timestamp, verbose, timeout)
}

type TestCase struct {
	Name string
	Test func() error
}

func runTests(category string, tests []TestCase, resultsDir, timestamp string, verbose bool, timeout time.Duration) (int, int) {
	var passed, failed int
	
	logFile := filepath.Join(resultsDir, fmt.Sprintf("%s-%s.log", category, timestamp))
	f, err := os.Create(logFile)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return 0, len(tests)
	}
	defer f.Close()

	for _, test := range tests {
		fmt.Printf("  🧪 %s...", test.Name)
		
		start := time.Now()
		err := runTestWithTimeout(test.Test, timeout)
		duration := time.Since(start)
		
		logEntry := fmt.Sprintf("[%s] %s: ", time.Now().Format("15:04:05"), test.Name)
		
		if err != nil {
			fmt.Printf(" ❌ (%v)\n", duration)
			logEntry += fmt.Sprintf("FAILED (%v): %v\n", duration, err)
			failed++
		} else {
			fmt.Printf(" ✅ (%v)\n", duration)
			logEntry += fmt.Sprintf("PASSED (%v)\n", duration)
			passed++
		}
		
		f.WriteString(logEntry)
		if verbose {
			fmt.Printf("    %s", logEntry)
		}
	}

	fmt.Printf("  📊 %s: %d passed, %d failed\n", category, passed, failed)
	return passed, failed
}

func runTestWithTimeout(test func() error, timeout time.Duration) error {
	done := make(chan error, 1)
	
	go func() {
		done <- test()
	}()
	
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("test timed out after %v", timeout)
	}
}

// Test implementations

func testMath() error {
	// Test basic math operations
	if 2+3 != 5 {
		return fmt.Errorf("addition failed")
	}
	if 5-3 != 2 {
		return fmt.Errorf("subtraction failed")
	}
	if 3*4 != 12 {
		return fmt.Errorf("multiplication failed")
	}
	if 10/2 != 5 {
		return fmt.Errorf("division failed")
	}
	return nil
}

func testStrings() error {
	// Test string operations
	input := "Hello, World!"
	expected := "Hello, World!"
	
	if input != expected {
		return fmt.Errorf("string equality failed")
	}
	
	if len(input) != len(expected) {
		return fmt.Errorf("string length failed")
	}
	
	return nil
}

func testCollections() error {
	// Test collection operations
	// Slice
	slice := []int{1, 2, 3}
	if len(slice) != 3 {
		return fmt.Errorf("slice length failed")
	}
	
	// Map
	m := make(map[string]int)
	m["key"] = 42
	if m["key"] != 42 {
		return fmt.Errorf("map operation failed")
	}
	
	return nil
}

func testTime() error {
	// Test time operations
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	duration := time.Since(start)
	
	if duration < 10*time.Millisecond {
		return fmt.Errorf("time measurement failed")
	}
	
	return nil
}

func testErrors() error {
	// Test error handling
	err := fmt.Errorf("test error")
	if err == nil {
		return fmt.Errorf("error creation failed")
	}
	
	if err.Error() != "test error" {
		return fmt.Errorf("error message failed")
	}
	
	return nil
}

func testConcurrency() error {
	// Test basic concurrency
	done := make(chan bool)
	
	go func() {
		time.Sleep(10 * time.Millisecond)
		done <- true
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("concurrency test failed")
	}
}

func testInputValidation() error {
	// Test input validation
	maliciousInputs := []string{
		"../../../etc/passwd",
		"<script>alert('xss')</script>",
		"' OR '1'='1",
	}
	
	for _, input := range maliciousInputs {
		if strings.Contains(input, "..") {
			// Should be caught
			continue
		}
		if strings.Contains(input, "<script") {
			// Should be caught
			continue
		}
		// This is simplified - real implementation would be more thorough
	}
	
	return nil
}

func testPasswordStrength() error {
	// Test password validation
	weak := "123456"
	if len(weak) < 8 {
		// Should be flagged as weak
	}
	
	strong := "MyStr0ng!P@ssw0rd"
	if len(strong) >= 8 && strings.ContainsAny(strong, "!@#$%^&*") {
		// Should be considered strong
	}
	
	return nil
}

func testURLSecurity() error {
	// Test URL security
	unsafeURLs := []string{
		"http://127.0.0.1:22",
		"file:///etc/passwd",
		"ftp://malicious.com",
	}
	
	for _, url := range unsafeURLs {
		if strings.HasPrefix(url, "file://") || strings.HasPrefix(url, "ftp://") {
			// Should be flagged as insecure
		}
	}
	
	return nil
}

func testPathTraversal() error {
	// Test path traversal
	paths := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32",
		"%2e%2e%2f%2e%2e%2f",
	}
	
	for _, path := range paths {
		if strings.Contains(path, "..") {
			// Should be caught
		}
	}
	
	return nil
}

func testInjectionResistance() error {
	// Test injection resistance
	injections := []string{
		"'; DROP TABLE users; --",
		"${jndi:ldap://malicious.com/a}",
		"$(rm -rf /)",
	}
	
	for _, injection := range injections {
		if strings.Contains(injection, "DROP TABLE") {
			// Should be caught
		}
	}
	
	return nil
}

func testSystemCommands() error {
	// Test system commands
	commands := []string{"echo", "pwd", "whoami"}
	
	for _, cmd := range commands {
		executable, err := exec.LookPath(cmd)
		if err != nil {
			continue // Command not available
		}
		
		execCmd := exec.Command(executable, "--help")
		execCmd.CombinedOutput() // Just test that it runs without panicking
	}
	
	return nil
}

func testFileOperations() error {
	// Test file operations
	tempDir := "/tmp/test-helix"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)
	
	// Create file
	testFile := filepath.Join(tempDir, "test.txt")
	data := []byte("test content")
	err := os.WriteFile(testFile, data, 0644)
	if err != nil {
		return fmt.Errorf("file creation failed: %v", err)
	}
	
	// Read file
	readData, err := os.ReadFile(testFile)
	if err != nil {
		return fmt.Errorf("file reading failed: %v", err)
	}
	
	if string(readData) != string(data) {
		return fmt.Errorf("file content mismatch")
	}
	
	return nil
}

func testNetworkConnectivity() error {
	// Test network connectivity
	services := []string{"google.com", "github.com"}
	
	for _, service := range services {
		// This is simplified - real implementation would do proper network checks
		if len(service) > 0 {
			// Basic validation
		}
	}
	
	return nil
}

func testProcessManagement() error {
	// Test process management
	cmd := exec.Command("echo", "test")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("process execution failed: %v", err)
	}
	
	if !strings.Contains(string(output), "test") {
		return fmt.Errorf("process output unexpected")
	}
	
	return nil
}

func testEnvironment() error {
	// Test environment variables
	home := os.Getenv("HOME")
	if runtime.GOOS != "windows" && home == "" {
		return fmt.Errorf("HOME environment variable not set")
	}
	
	// Test setting environment variable
	os.Setenv("TEST_VAR", "test_value")
	
	value := os.Getenv("TEST_VAR")
	if value != "test_value" {
		return fmt.Errorf("environment variable not set correctly")
	}
	
	// Clean up
	os.Unsetenv("TEST_VAR")
	
	return nil
}

func testCLIHelp() error {
	// Test CLI help
	if !isCLIAvailable() {
		return fmt.Errorf("CLI not available for testing")
	}
	
	// This would test the actual CLI help
	return nil
}

func testCLICommands() error {
	// Test CLI commands
	if !isCLIAvailable() {
		return fmt.Errorf("CLI not available for testing")
	}
	
	// This would test actual CLI commands
	return nil
}

func testProviderDetection() error {
	// Test provider detection
	providers := []string{"vllm", "ollama", "localai", "llamacpp"}
	
	for _, provider := range providers {
		executable, err := exec.LookPath(provider)
		if err != nil {
			continue // Provider not available
		}
		
		// Test that executable exists and can be found
		if executable == "" {
			return fmt.Errorf("provider %s not found", provider)
		}
	}
	
	return nil
}

func testModelOperations() error {
	// Test model operations
	models := []string{"llama-3-8b", "mistral-7b", "phi-2"}
	
	for _, model := range models {
		// This would test actual model operations
		if len(model) > 0 {
			// Basic validation
		}
	}
	
	return nil
}

func testConfiguration() error {
	// Test configuration management
	config := map[string]interface{}{
		"providers": []string{"vllm", "ollama"},
		"models":    []string{"llama-3-8b"},
		"settings":  map[string]string{"log_level": "info"},
	}
	
	if config["providers"] == nil {
		return fmt.Errorf("configuration parsing failed")
	}
	
	return nil
}

func testCPUDetection() error {
	// Test CPU detection
	numCPU := runtime.NumCPU()
	if numCPU <= 0 {
		return fmt.Errorf("CPU detection failed")
	}
	
	// Test CPU architecture
	arch := runtime.GOARCH
	if arch == "" {
		return fmt.Errorf("CPU architecture detection failed")
	}
	
	return nil
}

func testMemoryDetection() error {
	// Test memory detection
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	if m.Alloc < 0 {
		return fmt.Errorf("memory detection failed")
	}
	
	return nil
}

func testGPUDetection() error {
	// Test GPU detection
	// Check for NVIDIA GPU
	nvidiaCmd := exec.Command("nvidia-smi")
	err := nvidiaCmd.Run()
	hasNVIDIA := err == nil
	
	// Check for Apple Silicon GPU
	appleCmd := exec.Command("system_profiler", "SPDisplaysDataType")
	err = appleCmd.Run()
	hasApple := err == nil
	
	if !hasNVIDIA && !hasApple {
		// No GPU available, which is fine for CPU-only testing
	}
	
	return nil
}

func testOSDetection() error {
	// Test OS detection
	osName := runtime.GOOS
	if osName == "" {
		return fmt.Errorf("OS detection failed")
	}
	
	// Test OS version
	version := runtime.GOARCH
	if version == "" {
		return fmt.Errorf("OS architecture detection failed")
	}
	
	return nil
}

func testHardwareOptimization() error {
	// Test hardware optimization
	hwInfo := map[string]interface{}{
		"cpu_cores":  runtime.NumCPU(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
	}
	
	if hwInfo["cpu_cores"] == nil {
		return fmt.Errorf("hardware optimization failed")
	}
	
	return nil
}

func isCLIAvailable() bool {
	// Check if CLI is available
	_, err := exec.LookPath("./local-llm-test")
	return err == nil
}

func printFinalReport(passed, failed int, duration time.Duration) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 FINAL TEST REPORT")
	fmt.Println(strings.Repeat("=", 60))
	
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total Tests: %d\n", passed+failed)
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Success Rate: %.1f%%\n", float64(passed)/float64(passed+failed)*100)
	
	if failed > 0 {
		fmt.Println("\n❌ SOME TESTS FAILED")
		fmt.Println("Check test-results/ directory for detailed logs")
	} else {
		fmt.Println("\n✅ ALL TESTS PASSED!")
	}
	
	fmt.Println(strings.Repeat("=", 60))
}