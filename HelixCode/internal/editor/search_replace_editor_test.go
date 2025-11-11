package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchReplaceEditorApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "search_replace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		initial        string
		operations     []SearchReplace
		expectedResult string
		expectError    bool
	}{
		{
			name:    "Simple replace all",
			initial: "hello world hello universe",
			operations: []SearchReplace{
				{Search: "hello", Replace: "goodbye", Count: -1, Regex: false},
			},
			expectedResult: "goodbye world goodbye universe",
			expectError:    false,
		},
		{
			name:    "Replace first occurrence",
			initial: "hello world hello universe",
			operations: []SearchReplace{
				{Search: "hello", Replace: "goodbye", Count: 1, Regex: false},
			},
			expectedResult: "goodbye world hello universe",
			expectError:    false,
		},
		{
			name:    "Replace two occurrences",
			initial: "test test test test",
			operations: []SearchReplace{
				{Search: "test", Replace: "pass", Count: 2, Regex: false},
			},
			expectedResult: "pass pass test test",
			expectError:    false,
		},
		{
			name:    "Multiple operations",
			initial: "hello world, hello universe",
			operations: []SearchReplace{
				{Search: "hello", Replace: "hi", Count: -1, Regex: false},
				{Search: "world", Replace: "earth", Count: -1, Regex: false},
			},
			expectedResult: "hi earth, hi universe",
			expectError:    false,
		},
		{
			name:    "Regex replacement",
			initial: "test123 test456 test789",
			operations: []SearchReplace{
				{Search: `test\d+`, Replace: "pass", Count: -1, Regex: true},
			},
			expectedResult: "pass pass pass",
			expectError:    false,
		},
		{
			name:    "Regex with capture groups",
			initial: "hello123 world456",
			operations: []SearchReplace{
				{Search: `(\w+?)(\d+)`, Replace: "${1}-${2}", Count: -1, Regex: true},
			},
			expectedResult: "hello-123 world-456",
			expectError:    false,
		},
		{
			name:    "Search not found with ReplaceAll",
			initial: "hello world",
			operations: []SearchReplace{
				{Search: "notfound", Replace: "something", Count: -1, Regex: false},
			},
			expectedResult: "hello world",
			expectError:    false,
		},
		{
			name:    "Search not found with Count",
			initial: "hello world",
			operations: []SearchReplace{
				{Search: "notfound", Replace: "something", Count: 1, Regex: false},
			},
			expectedResult: "",
			expectError:    true,
		},
		{
			name:    "Invalid regex",
			initial: "test",
			operations: []SearchReplace{
				{Search: "[invalid", Replace: "something", Count: -1, Regex: true},
			},
			expectedResult: "",
			expectError:    true,
		},
		{
			name:    "Empty search",
			initial: "test",
			operations: []SearchReplace{
				{Search: "", Replace: "x", Count: -1, Regex: false},
			},
			expectedResult: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test_"+tt.name+".txt")

			// Create test file
			if err := os.WriteFile(testFile, []byte(tt.initial), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			editor := NewSearchReplaceEditor()
			edit := Edit{
				FilePath: testFile,
				Format:   EditFormatSearchReplace,
				Content:  tt.operations,
			}

			err := editor.Apply(edit)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify result
				result, err := os.ReadFile(testFile)
				if err != nil {
					t.Fatalf("Failed to read result: %v", err)
				}

				if string(result) != tt.expectedResult {
					t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", string(result), tt.expectedResult)
				}
			}
		})
	}
}

func TestSearchReplaceEditorApplyToLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "search_replace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	initial := "line1 test\nline2 test\nline3 other"

	if err := os.WriteFile(testFile, []byte(initial), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewSearchReplaceEditor()
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatSearchReplace,
		Content: []SearchReplace{
			{Search: "test", Replace: "pass", Count: -1, Regex: false},
		},
	}

	if err := editor.ApplyToLines(edit); err != nil {
		t.Fatalf("Failed to apply: %v", err)
	}

	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	expected := "line1 pass\nline2 pass\nline3 other\n"
	if string(result) != expected {
		t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", string(result), expected)
	}
}

func TestSearchReplaceEditorValidateOperation(t *testing.T) {
	editor := NewSearchReplaceEditor()

	tests := []struct {
		name        string
		operation   SearchReplace
		expectError bool
	}{
		{
			name:        "Valid literal",
			operation:   SearchReplace{Search: "test", Replace: "pass", Count: -1, Regex: false},
			expectError: false,
		},
		{
			name:        "Valid regex",
			operation:   SearchReplace{Search: `\d+`, Replace: "number", Count: -1, Regex: true},
			expectError: false,
		},
		{
			name:        "Empty search",
			operation:   SearchReplace{Search: "", Replace: "something", Count: -1, Regex: false},
			expectError: true,
		},
		{
			name:        "Invalid regex",
			operation:   SearchReplace{Search: "[invalid(", Replace: "x", Count: -1, Regex: true},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := editor.ValidateOperation(tt.operation)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSearchReplaceEditorCountMatches(t *testing.T) {
	editor := NewSearchReplaceEditor()

	tests := []struct {
		name          string
		content       string
		search        string
		regex         bool
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Literal matches",
			content:       "test test test",
			search:        "test",
			regex:         false,
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "No matches",
			content:       "hello world",
			search:        "notfound",
			regex:         false,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Regex matches",
			content:       "test123 test456 test789",
			search:        `test\d+`,
			regex:         true,
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "Invalid regex",
			content:       "test",
			search:        "[invalid",
			regex:         true,
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := editor.CountMatches(tt.content, tt.search, tt.regex)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if count != tt.expectedCount {
					t.Errorf("Count mismatch: got %d, want %d", count, tt.expectedCount)
				}
			}
		})
	}
}

func TestSearchReplaceEditorGetStats(t *testing.T) {
	editor := NewSearchReplaceEditor()

	tests := []struct {
		name               string
		content            string
		operations         []SearchReplace
		expectedMatches    int
		expectedReplaced   int
		expectedOperations int
	}{
		{
			name:    "Single operation, all replacements",
			content: "test test test",
			operations: []SearchReplace{
				{Search: "test", Replace: "pass", Count: -1, Regex: false},
			},
			expectedMatches:    3,
			expectedReplaced:   3,
			expectedOperations: 1,
		},
		{
			name:    "Single operation, limited replacements",
			content: "test test test",
			operations: []SearchReplace{
				{Search: "test", Replace: "pass", Count: 2, Regex: false},
			},
			expectedMatches:    3,
			expectedReplaced:   2,
			expectedOperations: 1,
		},
		{
			name:    "Multiple operations",
			content: "hello world, hello universe",
			operations: []SearchReplace{
				{Search: "hello", Replace: "hi", Count: -1, Regex: false},
				{Search: "world", Replace: "earth", Count: -1, Regex: false},
			},
			expectedMatches:    3,
			expectedReplaced:   3,
			expectedOperations: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := editor.GetStats(tt.content, tt.operations)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if stats.OperationCount != tt.expectedOperations {
				t.Errorf("OperationCount: got %d, want %d", stats.OperationCount, tt.expectedOperations)
			}
			if stats.TotalMatches != tt.expectedMatches {
				t.Errorf("TotalMatches: got %d, want %d", stats.TotalMatches, tt.expectedMatches)
			}
			if stats.TotalReplaced != tt.expectedReplaced {
				t.Errorf("TotalReplaced: got %d, want %d", stats.TotalReplaced, tt.expectedReplaced)
			}
		})
	}
}

func TestSearchReplaceEditorLargeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "search_replace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "large.txt")

	// Create large file with many occurrences
	var lines []string
	for i := 0; i < 10000; i++ {
		lines = append(lines, "line "+strings.Repeat("test ", 10))
	}
	content := strings.Join(lines, "\n")

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewSearchReplaceEditor()
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatSearchReplace,
		Content: []SearchReplace{
			{Search: "test", Replace: "pass", Count: -1, Regex: false},
		},
	}

	if err := editor.Apply(edit); err != nil {
		t.Fatalf("Failed to apply: %v", err)
	}

	// Verify replacements
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	if strings.Contains(string(result), "test") {
		t.Error("Found unreplaced 'test' strings")
	}
	if !strings.Contains(string(result), "pass") {
		t.Error("Expected 'pass' not found")
	}
}

func TestSearchReplaceEditorRegexEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "search_replace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		initial        string
		operation      SearchReplace
		expectedResult string
	}{
		{
			name:    "Word boundaries",
			initial: "test testing tested tester",
			operation: SearchReplace{
				Search:  `\btest\b`,
				Replace: "exam",
				Count:   -1,
				Regex:   true,
			},
			expectedResult: "exam testing tested tester",
		},
		{
			name:    "Start of string",
			initial: "test123\n  test456\ntest789",
			operation: SearchReplace{
				Search:  `^test\d+`,
				Replace: "pass",
				Count:   -1,
				Regex:   true,
			},
			expectedResult: "pass\n  test456\ntest789",
		},
		{
			name:    "Email pattern",
			initial: "contact: user@example.com and admin@test.org",
			operation: SearchReplace{
				Search:  `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
				Replace: "[email]",
				Count:   -1,
				Regex:   true,
			},
			expectedResult: "contact: [email] and [email]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test_"+tt.name+".txt")

			if err := os.WriteFile(testFile, []byte(tt.initial), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			editor := NewSearchReplaceEditor()
			edit := Edit{
				FilePath: testFile,
				Format:   EditFormatSearchReplace,
				Content:  []SearchReplace{tt.operation},
			}

			if err := editor.Apply(edit); err != nil {
				t.Fatalf("Failed to apply: %v", err)
			}

			result, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read result: %v", err)
			}

			if string(result) != tt.expectedResult {
				t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", string(result), tt.expectedResult)
			}
		})
	}
}

func TestSearchReplaceEditorNoOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "search_replace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewSearchReplaceEditor()
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatSearchReplace,
		Content:  []SearchReplace{},
	}

	err = editor.Apply(edit)
	if err == nil {
		t.Error("Expected error for empty operations")
	}
}

func TestSearchReplaceEditor_ApplyRegexOperation(t *testing.T) {
	editor := NewSearchReplaceEditor()

	t.Run("replace all matches with regex", func(t *testing.T) {
		content := "foo bar foo baz foo"
		op := SearchReplace{
			Search:  "foo",
			Replace: "qux",
			Count:   -1,
			Regex:   true,
		}

		result, err := editor.applyRegexOperation(content, op)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "qux bar qux baz qux"
		if result != expected {
			t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", result, expected)
		}
	})

	t.Run("replace limited number of matches", func(t *testing.T) {
		content := "foo bar foo baz foo"
		op := SearchReplace{
			Search:  "foo",
			Replace: "qux",
			Count:   2,
			Regex:   true,
		}

		result, err := editor.applyRegexOperation(content, op)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "qux bar qux baz foo"
		if result != expected {
			t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", result, expected)
		}
	})

	t.Run("replace single match", func(t *testing.T) {
		content := "foo bar foo baz"
		op := SearchReplace{
			Search:  "foo",
			Replace: "qux",
			Count:   1,
			Regex:   true,
		}

		result, err := editor.applyRegexOperation(content, op)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "qux bar foo baz"
		if result != expected {
			t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", result, expected)
		}
	})

	t.Run("invalid regex pattern", func(t *testing.T) {
		content := "test content"
		op := SearchReplace{
			Search:  "[invalid",
			Replace: "replacement",
			Count:   -1,
			Regex:   true,
		}

		_, err := editor.applyRegexOperation(content, op)

		if err == nil {
			t.Error("Expected error for invalid regex")
		}
		if !strings.Contains(err.Error(), "invalid regex pattern") {
			t.Errorf("Expected 'invalid regex pattern' error, got: %v", err)
		}
	})

	t.Run("pattern not found error", func(t *testing.T) {
		content := "hello world"
		op := SearchReplace{
			Search:  "xyz",
			Replace: "abc",
			Count:   1,
			Regex:   true,
		}

		_, err := editor.applyRegexOperation(content, op)

		if err == nil {
			t.Error("Expected error for pattern not found")
		}
		if !strings.Contains(err.Error(), "regex pattern not found") {
			t.Errorf("Expected 'regex pattern not found' error, got: %v", err)
		}
	})

	t.Run("complex regex with groups", func(t *testing.T) {
		content := "user@example.com and admin@test.org"
		op := SearchReplace{
			Search:  `(\w+)@(\w+\.\w+)`,
			Replace: "$1 at $2",
			Count:   -1,
			Regex:   true,
		}

		result, err := editor.applyRegexOperation(content, op)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		expected := "user at example.com and admin at test.org"
		if result != expected {
			t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", result, expected)
		}
	})
}
