package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Simple test runner that doesn't depend on full project
func main() {
	var (
		testType   = flag.String("type", "unit", "Type of test to run (unit, security, integration)")
		timeout    = flag.Duration("timeout", 5*time.Minute, "Test timeout")
		skipExp    = flag.Bool("skip-expensive", false, "Skip expensive tests")
		skipHard   = flag.Bool("skip-hardware", false, "Skip hardware tests")
	)
	flag.Parse()

	fmt.Println("🧪 HelixCode Local LLM - Simple Test Runner")
	fmt.Println("========================================")

	env := os.Environ()
	if *skipExp {
		env = append(env, "SKIP_EXPENSIVE_TESTS=true")
		fmt.Println("⏭️  Skipping expensive tests")
	}
	if *skipHard {
		env = append(env, "SKIP_HARDWARE_TESTS=true")
		fmt.Println("⏭️  Skipping hardware tests")
	}

	var cmd *exec.Cmd
	switch *testType {
	case "unit":
		fmt.Println("🧪 Running unit tests...")
		cmd = exec.Command("go", "test", "-v", "-race", 
			fmt.Sprintf("-timeout=%v", *timeout),
			"./tests/unit/")
		
	case "security":
		fmt.Println("🔒 Running security tests...")
		cmd = exec.Command("go", "test", "-v",
			fmt.Sprintf("-timeout=%v", *timeout),
			"./security/")
		
	case "integration":
		fmt.Println("🔗 Running integration tests...")
		cmd = exec.Command("go", "test", "-v",
			fmt.Sprintf("-timeout=%v", *timeout*2),
			"./tests/integration/")
		
	case "e2e":
		fmt.Println("🎯 Running E2E tests...")
		cmd = exec.Command("go", "test", "-v",
			fmt.Sprintf("-timeout=%v", *timeout*3),
			"./tests/e2e/")
		
	case "automation":
		fmt.Println("🤖 Running automation tests...")
		cmd = exec.Command("go", "test", "-v",
			fmt.Sprintf("-timeout=%v", *timeout*4),
			"./tests/automation/")
		
	default:
		fmt.Printf("❌ Unknown test type: %s\n", *testType)
		os.Exit(1)
	}

	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("\n❌ Tests failed after %v: %v\n", duration, err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ All tests passed in %v!\n", duration)
}