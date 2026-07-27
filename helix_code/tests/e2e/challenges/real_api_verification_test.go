package challenges

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVerifyRealLLMAPIsUsed is a comprehensive test that ensures the framework
// NEVER uses mock/stub mechanisms and ALWAYS calls real LLM APIs.
//
// This test was added to prevent a critical bug where all 126 tests used
// identical mock data instead of calling real LLM providers.
//
// Test Strategy:
// 1. Execute a challenge with a real provider configuration
// 2. Verify execution logs show "Using REAL LLM API" (not "Using mock generator")
// 3. Verify LLM API call attempts are logged
// 4. Ensure no mock generator methods are called
// 5. Verify that failures are due to API connectivity (not mock data)
func TestVerifyRealLLMAPIsUsed(t *testing.T) {
	ctx := context.Background()

	// Test configuration
	testCases := []struct {
		name         string
		provider     LLMProviderType
		model        string
		expectedLogs []string // Logs that MUST appear
		bannedLogs   []string // Logs that MUST NOT appear
	}{
		{
			name:     "Ollama_Real_API",
			provider: ProviderOllama,
			model:    "llama2",
			expectedLogs: []string{
				"Using REAL LLM API for code generation",
				"LLM client created successfully",
				"Calling real LLM API",
			},
			bannedLogs: []string{
				"Using mock generator",
				"Generating mock ASCII Art",
				"Generating mock Tic-Tac-Toe",
				"Generating mock Notes",
			},
		},
		{
			name:     "Gemini_Real_API",
			provider: ProviderGemini,
			model:    "gemini-pro",
			expectedLogs: []string{
				"Using REAL LLM API for code generation",
				"LLM client created successfully",
				"Calling real LLM API",
			},
			bannedLogs: []string{
				"Using mock generator",
				"Generating mock ASCII Art",
				"Generating mock Tic-Tac-Toe",
				"Generating mock Notes",
			},
		},
		{
			name:     "DeepSeek_Real_API",
			provider: ProviderDeepSeek,
			model:    "deepseek-coder",
			expectedLogs: []string{
				"Using REAL LLM API for code generation",
				"LLM client created successfully",
				"Calling real LLM API",
			},
			bannedLogs: []string{
				"Using mock generator",
				"Generating mock ASCII Art",
				"Generating mock Tic-Tac-Toe",
				"Generating mock Notes",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load challenge spec
			spec, err := LoadChallengeSpec("definitions/ascii-art-generator.json")
			if err != nil {
				t.Fatalf("Failed to load challenge spec: %v", err)
			}

			// Create executor with minimal config
			config := &ChallengeConfig{
				ResultsBaseDir: "_test-results",
				LogsBaseDir:    "_test-results/logs",
			}
			executor := NewChallengeExecutor(config)

			// Execute challenge (will fail due to no LLM API, but that's expected)
			execution, _ := executor.Execute(ctx, spec, InterfaceCLI, DistributionSingle, tc.provider, tc.model)

			// Read execution log
			logPath := filepath.Join("_test-results/logs", execution.ID, "execution.log")
			logContent, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("Failed to read execution log: %v", err)
			}

			logText := string(logContent)

			// CRITICAL: Verify expected logs are present
			for _, expectedLog := range tc.expectedLogs {
				if !strings.Contains(logText, expectedLog) {
					t.Errorf("❌ CRITICAL: Expected log entry not found: %q", expectedLog)
					t.Errorf("This indicates the framework may not be using real LLM APIs!")
					t.Errorf("Log content:\n%s", logText)
				} else {
					t.Logf("✅ Found expected log: %q", expectedLog)
				}
			}

			// CRITICAL: Verify banned logs are NOT present
			for _, bannedLog := range tc.bannedLogs {
				if strings.Contains(logText, bannedLog) {
					t.Errorf("❌ CRITICAL: Banned log entry found: %q", bannedLog)
					t.Errorf("This indicates MOCK DATA is being used instead of real LLM APIs!")
					t.Errorf("This is the exact bug this test was designed to prevent!")
					t.Errorf("Log content:\n%s", logText)
					t.FailNow()
				} else {
					t.Logf("✅ Confirmed banned log NOT present: %q", bannedLog)
				}
			}

			// Additional verification: Check for LLM API call evidence
			hasAPICall := strings.Contains(logText, "LLM API call") ||
				strings.Contains(logText, "Calling real LLM API") ||
				strings.Contains(logText, "Post \"http") // HTTP requests to LLM APIs

			if !hasAPICall {
				t.Errorf("❌ CRITICAL: No evidence of LLM API calls found in logs")
				t.Errorf("This suggests mock data may be used")
			} else {
				t.Logf("✅ Confirmed LLM API call attempts present")
			}

			// Note: Result directory may not exist if API failed early - this is expected behavior
			t.Logf("ℹ️  Test completed (API failures are expected when LLM services not running)")

			t.Logf("✅ Real API verification passed for %s/%s", tc.provider, tc.model)
		})
	}
}

// TestNoMockGeneratorInProduction ensures the MockGenerator is never instantiated
// in production code paths.
func TestNoMockGeneratorInProduction(t *testing.T) {
	// This test verifies that the executor.go file doesn't instantiate MockGenerator
	// in the production code path

	executorPath := "executor.go"
	content, err := os.ReadFile(executorPath)
	if err != nil {
		t.Fatalf("Failed to read executor.go: %v", err)
	}

	executorCode := string(content)

	// Check for banned patterns that indicate mock usage
	bannedPatterns := []string{
		"NewMockGenerator()",
		"mockGen :=",
		"Using mock generator",
		"GenerateASCIIArtGenerator",
		"GenerateTicTacToeGame",
		"GenerateNotesProject",
	}

	for _, pattern := range bannedPatterns {
		if strings.Contains(executorCode, pattern) {
			t.Errorf("❌ CRITICAL: Found banned mock pattern in executor.go: %q", pattern)
			t.Errorf("The executor must NEVER use mock generators in production!")
			t.Errorf("All code generation must use REAL LLM APIs!")
		}
	}

	// Check for required patterns that indicate real LLM usage
	requiredPatterns := []string{
		"Using REAL LLM API",
		"NewLLMClient",
		"client.Complete",
	}

	for _, pattern := range requiredPatterns {
		if !strings.Contains(executorCode, pattern) {
			t.Errorf("❌ CRITICAL: Required real LLM pattern missing in executor.go: %q", pattern)
		} else {
			t.Logf("✅ Found required pattern: %q", pattern)
		}
	}
}

// TestVerifyCodeDiversityAcrossProviders ensures that different LLM providers
// generate DIFFERENT code, not identical mock data.
//
// This test addresses the critical bug where all 21 ASCII art tests had
// identical MD5 hash: cdde8a3be5b39ca5e58a644e01052132
func TestVerifyCodeDiversityAcrossProviders(t *testing.T) {
	t.Skip("Skipping until LLM APIs are configured - requires real API keys and running services")  // SKIP-OK: #requires-upstream-key

	// This test would:
	// 1. Execute same challenge with 2+ different providers
	// 2. Compare MD5 hashes of generated files
	// 3. FAIL if hashes are identical (indicates mock data usage)
	// 4. PASS if hashes differ (indicates real LLM generation)

	// Implementation note: This test requires actual LLM API access,
	// so it's skipped by default but serves as documentation of the
	// testing strategy.
}
