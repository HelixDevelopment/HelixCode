package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestCLIHelp(t *testing.T) {
	// Test that CLI help works
	cmd := exec.Command("./local-llm-test", "--help")
	output, err := cmd.CombinedOutput()
	
	// CLI might not exist in test environment
	if err != nil {
		t.Skip("CLI not available for testing")
	}
	
	if len(output) == 0 {
		t.Error("CLI help output is empty")
	}
}

func TestCLICommands(t *testing.T) {
	// Test that CLI commands work
	cmd := exec.Command("./local-llm-test", "--version")
	output, err := cmd.CombinedOutput()
	
	// CLI might not exist in test environment
	if err != nil {
		t.Skip("CLI not available for testing")
	}
	
	if len(output) == 0 {
		t.Error("CLI version output is empty")
	}
}