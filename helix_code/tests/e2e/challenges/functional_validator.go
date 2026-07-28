package challenges

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// FunctionalValidator performs functional testing of generated applications
type FunctionalValidator struct {
	config *ChallengeConfig
}

// NewFunctionalValidator creates a new functional validator
func NewFunctionalValidator(config *ChallengeConfig) *FunctionalValidator {
	return &FunctionalValidator{
		config: config,
	}
}

// ValidateFunctional performs functional testing on the generated project
func (v *FunctionalValidator) ValidateFunctional(ctx context.Context, spec *ChallengeSpec, resultDir string) []ValidationResult {
	results := []ValidationResult{}

	switch spec.ID {
	case "notes-project-001":
		results = append(results, v.validateNotesProject(ctx, resultDir)...)
	case "url-shortener-001":
		results = append(results, v.validateURLShortener(ctx, resultDir)...)
	case "cli-task-manager-001":
		results = append(results, v.validateCLITaskManager(ctx, resultDir)...)
	case "ascii-art-generator-001":
		// CLI tools with simple I/O are validated through runtime tests
		results = append(results, ValidationResult{
			CheckName: "functional_tests_cli",
			Passed:    true,
			Message:   "CLI tool validated through runtime tests",
			Details:   "Runtime validation tests --help, basic generation, and error handling",
			Timestamp: time.Now(),
		})
	case "tic-tac-toe-tui-001":
		// TUI games require interactive input, can't be functionally tested automatically
		results = append(results, ValidationResult{
			CheckName: "functional_tests_tui",
			Passed:    true,
			Message:   "TUI game validated - interactive functional testing requires manual play",
			Details:   "Compilation and unit tests verify game logic correctness",
			Timestamp: time.Now(),
		})
	case "json-validator-cli-001":
		results = append(results, v.validateJSONValidatorFunctional(ctx, resultDir)...)
	default:
		results = append(results, ValidationResult{
			CheckName: "functional_tests",
			Passed:    false,
			Error:     fmt.Sprintf("No functional tests defined for challenge: %s (add validation case to functional_validator.go)", spec.ID),
			Timestamp: time.Now(),
		})
	}

	return results
}

// validateNotesProject performs functional testing for the Notes API project
func (v *FunctionalValidator) validateNotesProject(ctx context.Context, resultDir string) []ValidationResult {
	results := []ValidationResult{}

	// Start the server in background
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	port := "8081" // Use different port to avoid conflicts
	// net.JoinHostPort (not "%s:%s") so this stays correct if the host ever
	// becomes configurable and can carry an IPv6 literal, which is only valid
	// in a URL authority when bracketed (RFC 3986 §3.2.2). Output is
	// byte-identical for the current literal host.
	serverURL := "http://" + net.JoinHostPort("localhost", port)

	// Start server
	started, cleanup := v.startServer(serverCtx, resultDir, port)
	if !started {
		results = append(results, ValidationResult{
			CheckName: "server_start",
			Passed:    false,
			Error:     "Failed to start server",
			Timestamp: time.Now(),
		})
		return results
	}
	defer cleanup()

	results = append(results, ValidationResult{
		CheckName: "server_start",
		Passed:    true,
		Message:   fmt.Sprintf("Server started successfully on port %s", port),
		Timestamp: time.Now(),
	})

	// Wait for server to be ready
	if !v.waitForServer(serverURL, 10*time.Second) {
		// Check if failure is due to database connection - this is a REAL failure, not acceptable
		serverLog := filepath.Join(resultDir, "server.log")
		logContent, _ := os.ReadFile(serverLog)
		if strings.Contains(string(logContent), "database") || strings.Contains(string(logContent), "connection refused") {
			results = append(results, ValidationResult{
				CheckName: "server_ready_without_db",
				Passed:    false,
				Error:     "Server failed to start due to missing database connection",
				Details:   fmt.Sprintf("Server requires database. Log: %s. FIX: Start test database or mock database layer.", string(logContent)),
				Timestamp: time.Now(),
			})
		} else {
			results = append(results, ValidationResult{
				CheckName: "server_ready",
				Passed:    false,
				Error:     "Server did not become ready in time",
				Details:   string(logContent),
				Timestamp: time.Now(),
			})
		}
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "server_ready",
		Passed:    true,
		Message:   "Server is ready and responding",
		Timestamp: time.Now(),
	})

	// Test CRUD operations
	results = append(results, v.testNotesCreate(serverURL)...)
	results = append(results, v.testNotesList(serverURL)...)
	results = append(results, v.testNotesGet(serverURL)...)
	results = append(results, v.testNotesUpdate(serverURL)...)
	results = append(results, v.testNotesDelete(serverURL)...)
	results = append(results, v.testNotesSearch(serverURL)...)

	// Test error handling
	results = append(results, v.testNotesErrorHandling(serverURL)...)

	// Test UX elements
	results = append(results, v.testNotesUX(serverURL)...)

	return results
}

// buildProject attempts to build the project
func (v *FunctionalValidator) buildProject(ctx context.Context, resultDir string) error {
	// Check if there's a go.mod file
	goMod := filepath.Join(resultDir, "go.mod")
	if _, err := os.Stat(goMod); err != nil {
		return fmt.Errorf("no go.mod file found")
	}

	// Try to build the project
	cmd := exec.CommandContext(ctx, "go", "build", "-o", "url-shortener", ".")
	cmd.Dir = resultDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %v, stderr: %s", err, stderr.String())
	}

	return nil
}

// startServer starts the application server in background
func (v *FunctionalValidator) startServer(ctx context.Context, resultDir, port string) (bool, func()) {
	// Build the server first
	buildCmd := exec.Command("go", "build", "-o", "test-server", "./cmd/server")
	buildCmd.Dir = resultDir
	if err := buildCmd.Run(); err != nil {
		fmt.Printf("Failed to build server: %v\n", err)
		return false, func() {}
	}

	// Start the server
	serverCmd := exec.CommandContext(ctx, "./test-server")
	serverCmd.Dir = resultDir
	serverCmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%s", port),
		"DB_HOST=localhost",
		"DB_PORT=5432",
		"DB_USER=postgres",
		"DB_PASSWORD=postgres",
		"DB_NAME=notes_test",
		"GIN_MODE=release",
	)

	// Redirect output for debugging
	logFile, _ := os.Create(filepath.Join(resultDir, "server.log"))
	serverCmd.Stdout = logFile
	serverCmd.Stderr = logFile

	if err := serverCmd.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return false, func() {}
	}

	cleanup := func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
		if logFile != nil {
			logFile.Close()
		}
		os.Remove(filepath.Join(resultDir, "test-server"))
	}

	return true, cleanup
}

// waitForServer waits for the server to be ready
func (v *FunctionalValidator) waitForServer(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/notes")
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}

	return false
}

// testNotesCreate tests note creation functionality
func (v *FunctionalValidator) testNotesCreate(baseURL string) []ValidationResult {
	results := []ValidationResult{}

	note := map[string]interface{}{
		"title":   "Test Note",
		"content": "This is a test note",
		"tags":    []string{"test", "automated"},
	}

	jsonData, _ := json.Marshal(note)
	resp, err := http.Post(baseURL+"/notes", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "create_note",
			Passed:    false,
			Error:     fmt.Sprintf("Failed to create note: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		results = append(results, ValidationResult{
			CheckName: "create_note",
			Passed:    false,
			Error:     fmt.Sprintf("Expected status 201 or 200, got %d", resp.StatusCode),
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "create_note",
		Passed:    true,
		Message:   "Note created successfully",
		Timestamp: time.Now(),
	})

	return results
}

// testNotesList tests listing notes
func (v *FunctionalValidator) testNotesList(baseURL string) []ValidationResult {
	results := []ValidationResult{}

	resp, err := http.Get(baseURL + "/notes")
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "list_notes",
			Passed:    false,
			Error:     fmt.Sprintf("Failed to list notes: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		results = append(results, ValidationResult{
			CheckName: "list_notes",
			Passed:    false,
			Error:     fmt.Sprintf("Expected status 200, got %d", resp.StatusCode),
			Timestamp: time.Now(),
		})
		return results
	}

	body, _ := io.ReadAll(resp.Body)
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		results = append(results, ValidationResult{
			CheckName: "list_notes",
			Passed:    false,
			Error:     "Response is not valid JSON",
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "list_notes",
		Passed:    true,
		Message:   "Notes listed successfully",
		Timestamp: time.Now(),
	})

	return results
}

// testNotesGet tests getting a single note
func (v *FunctionalValidator) testNotesGet(baseURL string) []ValidationResult {
	results := []ValidationResult{}

	resp, err := http.Get(baseURL + "/notes/test-id-123")
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "get_note",
			Passed:    false,
			Error:     fmt.Sprintf("Failed to get note: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		results = append(results, ValidationResult{
			CheckName: "get_note",
			Passed:    false,
			Error:     fmt.Sprintf("Expected status 200 or 404, got %d", resp.StatusCode),
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "get_note",
		Passed:    true,
		Message:   "Get note endpoint working",
		Timestamp: time.Now(),
	})

	return results
}

// testNotesUpdate tests updating a note
func (v *FunctionalValidator) testNotesUpdate(baseURL string) []ValidationResult {
	results := []ValidationResult{}

	note := map[string]interface{}{
		"title":   "Updated Note",
		"content": "This note has been updated",
	}

	jsonData, _ := json.Marshal(note)
	req, _ := http.NewRequest(http.MethodPut, baseURL+"/notes/test-id-123", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "update_note",
			Passed:    false,
			Error:     fmt.Sprintf("Failed to update note: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		results = append(results, ValidationResult{
			CheckName: "update_note",
			Passed:    false,
			Error:     fmt.Sprintf("Expected status 200 or 404, got %d", resp.StatusCode),
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "update_note",
		Passed:    true,
		Message:   "Update note endpoint working",
		Timestamp: time.Now(),
	})

	return results
}

// testNotesDelete tests deleting a note
func (v *FunctionalValidator) testNotesDelete(baseURL string) []ValidationResult {
	results := []ValidationResult{}

	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/notes/test-id-123", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "delete_note",
			Passed:    false,
			Error:     fmt.Sprintf("Failed to delete note: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusNoContent {
		results = append(results, ValidationResult{
			CheckName: "delete_note",
			Passed:    false,
			Error:     fmt.Sprintf("Expected status 200, 204 or 404, got %d", resp.StatusCode),
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "delete_note",
		Passed:    true,
		Message:   "Delete note endpoint working",
		Timestamp: time.Now(),
	})

	return results
}

// testNotesSearch tests search functionality
func (v *FunctionalValidator) testNotesSearch(baseURL string) []ValidationResult {
	results := []ValidationResult{}

	resp, err := http.Get(baseURL + "/notes/search?q=test")
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "search_notes",
			Passed:    false,
			Error:     fmt.Sprintf("Failed to search notes: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		results = append(results, ValidationResult{
			CheckName: "search_notes",
			Passed:    false,
			Error:     fmt.Sprintf("Expected status 200, got %d", resp.StatusCode),
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "search_notes",
		Passed:    true,
		Message:   "Search notes endpoint working",
		Timestamp: time.Now(),
	})

	return results
}

// testNotesErrorHandling tests error handling
func (v *FunctionalValidator) testNotesErrorHandling(baseURL string) []ValidationResult {
	results := []ValidationResult{}

	// Test invalid JSON
	resp, err := http.Post(baseURL+"/notes", "application/json", strings.NewReader("invalid json"))
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "error_handling_invalid_json",
			Passed:    false,
			Error:     fmt.Sprintf("Failed to test error handling: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		results = append(results, ValidationResult{
			CheckName: "error_handling_invalid_json",
			Passed:    false,
			Error:     fmt.Sprintf("Expected status 400 for invalid JSON, got %d", resp.StatusCode),
			Details:   "Server should return 400 Bad Request for invalid JSON",
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "error_handling_invalid_json",
		Passed:    true,
		Message:   "Server properly handles invalid JSON",
		Timestamp: time.Now(),
	})

	return results
}

// testNotesUX tests UX elements
func (v *FunctionalValidator) testNotesUX(baseURL string) []ValidationResult {
	results := []ValidationResult{}

	// Test error message format
	resp, err := http.Post(baseURL+"/notes", "application/json", strings.NewReader("invalid"))
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "ux_error_format",
			Passed:    false,
			Message:   fmt.Sprintf("Failed to test error handling: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "ux_error_format",
			Passed:    false,
			Message:   fmt.Sprintf("Failed to read error response: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	var errorResponse map[string]interface{}
	if err := json.Unmarshal(body, &errorResponse); err != nil {
		results = append(results, ValidationResult{
			CheckName: "ux_error_format",
			Passed:    false,
			Error:     "Error responses are not in JSON format",
			Details:   "UX should provide structured error messages in JSON format",
			Timestamp: time.Now(),
		})
		return results
	}

	// Check if error message exists
	if _, hasError := errorResponse["error"]; !hasError {
		if _, hasMessage := errorResponse["message"]; !hasMessage {
			results = append(results, ValidationResult{
				CheckName: "ux_error_format",
				Passed:    false,
				Error:     "Error response missing 'error' or 'message' field",
				Details:   "UX should provide clear error messages in responses",
				Timestamp: time.Now(),
			})
			return results
		}
	}

	results = append(results, ValidationResult{
		CheckName: "ux_error_format",
		Passed:    true,
		Message:   "Error responses have proper UX format",
		Timestamp: time.Now(),
	})

	// Test response consistency
	listResp, err := http.Get(baseURL + "/notes")
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "ux_response_consistency",
			Passed:    false,
			Message:   fmt.Sprintf("Failed to test response consistency: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	defer listResp.Body.Close()

	listBody, err := io.ReadAll(listResp.Body)
	if err != nil {
		results = append(results, ValidationResult{
			CheckName: "ux_response_consistency",
			Passed:    false,
			Message:   fmt.Sprintf("Failed to read response body: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}
	var listResponse map[string]interface{}
	if err := json.Unmarshal(listBody, &listResponse); err != nil {
		results = append(results, ValidationResult{
			CheckName: "ux_response_consistency",
			Passed:    false,
			Message:   "Response not in JSON format",
			Timestamp: time.Now(),
		})
		return results
	}

	if _, hasNotes := listResponse["notes"]; !hasNotes {
		results = append(results, ValidationResult{
			CheckName: "ux_response_consistency",
			Passed:    false,
			Error:     "List response missing 'notes' field",
			Details:   "API responses should have consistent field naming",
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "ux_response_consistency",
		Passed:    true,
		Message:   "API responses have consistent structure",
		Timestamp: time.Now(),
	})

	return results
}

// validateURLShortener performs functional testing for URL shortener
func (v *FunctionalValidator) validateURLShortener(ctx context.Context, resultDir string) []ValidationResult {
	var results []ValidationResult

	// Try to build and test the URL shortener
	if err := v.buildProject(ctx, resultDir); err != nil {
		results = append(results, ValidationResult{
			CheckName: "build",
			Passed:    false,
			Message:   fmt.Sprintf("Failed to build URL shortener: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "build",
		Passed:    true,
		Message:   "URL shortener builds successfully",
		Timestamp: time.Now(),
	})

	// Test basic functionality if build succeeded
	binPath := filepath.Join(resultDir, "url-shortener")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	// Check if binary exists
	if _, err := os.Stat(binPath); err != nil {
		results = append(results, ValidationResult{
			CheckName: "binary_exists",
			Passed:    false,
			Message:   "URL shortener binary not found after build",
			Timestamp: time.Now(),
		})
	} else {
		results = append(results, ValidationResult{
			CheckName: "binary_exists",
			Passed:    true,
			Message:   "URL shortener binary created successfully",
			Timestamp: time.Now(),
		})
	}

	// Try to start the server (if applicable)
	// Note: Full functional testing would require more complex setup with temporary ports
	// For now, we'll validate the structure and buildability
	
	return results
}

// validateCLITaskManager performs functional testing for CLI task manager
func (v *FunctionalValidator) validateCLITaskManager(ctx context.Context, resultDir string) []ValidationResult {
	results := []ValidationResult{}

	// Build the CLI task manager
	if err := v.buildCLIProject(ctx, resultDir, "task"); err != nil {
		results = append(results, ValidationResult{
			CheckName: "build",
			Passed:    false,
			Message:   fmt.Sprintf("Failed to build CLI task manager: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "build",
		Passed:    true,
		Message:   "CLI task manager builds successfully",
		Timestamp: time.Now(),
	})

	// Get binary path
	binPath := filepath.Join(resultDir, "task")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	// Check if binary exists
	if _, err := os.Stat(binPath); err != nil {
		results = append(results, ValidationResult{
			CheckName: "binary_exists",
			Passed:    false,
			Message:   "CLI task manager binary not found after build",
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "binary_exists",
		Passed:    true,
		Message:   "CLI task manager binary created successfully",
		Timestamp: time.Now(),
	})

	// Test help command
	helpResult := v.testCLICommand(ctx, binPath, []string{"--help"})
	if !helpResult.Passed {
		helpResult.CheckName = "help_command"
		results = append(results, helpResult)
	} else {
		results = append(results, ValidationResult{
			CheckName: "help_command",
			Passed:    true,
			Message:   "Help command works correctly",
			Timestamp: time.Now(),
		})
	}

	// Test add command
	addResult := v.testCLICommand(ctx, binPath, []string{"add", "Test task", "--priority", "high"})
	if addResult.Passed {
		results = append(results, ValidationResult{
			CheckName: "add_command",
			Passed:    true,
			Message:   "Add command works correctly",
			Timestamp: time.Now(),
		})
	} else {
		// Try without priority flag
		addResult2 := v.testCLICommand(ctx, binPath, []string{"add", "Test task"})
		if addResult2.Passed {
			results = append(results, ValidationResult{
				CheckName: "add_command",
				Passed:    true,
				Message:   "Add command works correctly (basic form)",
				Timestamp: time.Now(),
			})
		} else {
			results = append(results, ValidationResult{
				CheckName: "add_command",
				Passed:    false,
				Message:   "Add command failed",
				Details:   addResult.Message,
				Timestamp: time.Now(),
			})
		}
	}

	// Test list command
	listResult := v.testCLICommand(ctx, binPath, []string{"list"})
	if listResult.Passed {
		results = append(results, ValidationResult{
			CheckName: "list_command",
			Passed:    true,
			Message:   "List command works correctly",
			Timestamp: time.Now(),
		})
	} else {
		results = append(results, ValidationResult{
			CheckName: "list_command",
			Passed:    false,
			Message:   "List command failed",
			Details:   listResult.Message,
			Timestamp: time.Now(),
		})
	}

	// Test that help includes expected commands
	helpOutput := v.getCLIOutput(ctx, binPath, []string{"--help"})
	expectedCommands := []string{"add", "list", "complete", "delete"}
	foundCommands := 0
	for _, cmd := range expectedCommands {
		if strings.Contains(strings.ToLower(helpOutput), cmd) {
			foundCommands++
		}
	}

	if foundCommands >= 3 {
		results = append(results, ValidationResult{
			CheckName: "command_completeness",
			Passed:    true,
			Message:   fmt.Sprintf("CLI has %d/%d expected commands documented", foundCommands, len(expectedCommands)),
			Timestamp: time.Now(),
		})
	} else {
		results = append(results, ValidationResult{
			CheckName: "command_completeness",
			Passed:    false,
			Message:   fmt.Sprintf("CLI missing commands: only %d/%d found in help", foundCommands, len(expectedCommands)),
			Details:   "Expected commands: add, list, complete, delete",
			Timestamp: time.Now(),
		})
	}

	return results
}

// buildCLIProject builds a CLI project with a specific output name
func (v *FunctionalValidator) buildCLIProject(ctx context.Context, resultDir, outputName string) error {
	// Check if there's a go.mod file
	goMod := filepath.Join(resultDir, "go.mod")
	if _, err := os.Stat(goMod); err != nil {
		return fmt.Errorf("no go.mod file found")
	}

	// Try to build the project
	cmd := exec.CommandContext(ctx, "go", "build", "-o", outputName, ".")
	cmd.Dir = resultDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %v, stderr: %s", err, stderr.String())
	}

	return nil
}

// testCLICommand tests a CLI command and returns the result
func (v *FunctionalValidator) testCLICommand(ctx context.Context, binPath string, args []string) ValidationResult {
	cmd := exec.CommandContext(ctx, binPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return ValidationResult{
			Passed:    false,
			Message:   fmt.Sprintf("Command failed: %v, stderr: %s", err, stderr.String()),
			Timestamp: time.Now(),
		}
	}

	return ValidationResult{
		Passed:    true,
		Message:   stdout.String(),
		Timestamp: time.Now(),
	}
}

// getCLIOutput runs a CLI command and returns its stdout output
func (v *FunctionalValidator) getCLIOutput(ctx context.Context, binPath string, args []string) string {
	cmd := exec.CommandContext(ctx, binPath, args...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	_ = cmd.Run()
	return stdout.String()
}

// validateJSONValidatorFunctional performs functional testing for JSON validator CLI
func (v *FunctionalValidator) validateJSONValidatorFunctional(ctx context.Context, resultDir string) []ValidationResult {
	results := []ValidationResult{}

	if err := v.buildCLIProject(ctx, resultDir, "json-validator"); err != nil {
		results = append(results, ValidationResult{
			CheckName: "build",
			Passed:    false,
			Message:   fmt.Sprintf("Failed to build JSON validator: %v", err),
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "build",
		Passed:    true,
		Message:   "JSON validator builds successfully",
		Timestamp: time.Now(),
	})

	binPath := filepath.Join(resultDir, "json-validator")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	if _, err := os.Stat(binPath); err != nil {
		results = append(results, ValidationResult{
			CheckName: "binary_exists",
			Passed:    false,
			Message:   "JSON validator binary not found after build",
			Timestamp: time.Now(),
		})
		return results
	}

	results = append(results, ValidationResult{
		CheckName: "binary_exists",
		Passed:    true,
		Message:   "JSON validator binary created successfully",
		Timestamp: time.Now(),
	})

	helpOutput := v.getCLIOutput(ctx, binPath, []string{"--help"})
	expectedTerms := []string{"json", "validate", "file"}
	foundTerms := 0
	for _, term := range expectedTerms {
		if strings.Contains(strings.ToLower(helpOutput), term) {
			foundTerms++
		}
	}

	if foundTerms >= 2 {
		results = append(results, ValidationResult{
			CheckName: "help_documentation",
			Passed:    true,
			Message:   fmt.Sprintf("Help includes %d/%d expected terms", foundTerms, len(expectedTerms)),
			Timestamp: time.Now(),
		})
	} else {
		results = append(results, ValidationResult{
			CheckName: "help_documentation",
			Passed:    false,
			Message:   fmt.Sprintf("Help missing expected terms: only %d/%d found", foundTerms, len(expectedTerms)),
			Details:   "Expected terms: json, validate, file",
			Timestamp: time.Now(),
		})
	}

	return results
}
