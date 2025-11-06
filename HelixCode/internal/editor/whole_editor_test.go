package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWholeEditorApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "whole_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		initial     string
		newContent  string
		expectError bool
	}{
		{
			name:        "Simple replacement",
			initial:     "old content",
			newContent:  "new content",
			expectError: false,
		},
		{
			name:        "Empty to content",
			initial:     "",
			newContent:  "some content",
			expectError: false,
		},
		{
			name:        "Content to empty",
			initial:     "some content",
			newContent:  "",
			expectError: false,
		},
		{
			name:        "Large content",
			initial:     "small",
			newContent:  strings.Repeat("large content\n", 1000),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test_"+tt.name+".txt")

			// Create initial file
			if err := os.WriteFile(testFile, []byte(tt.initial), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			editor := NewWholeEditor()
			edit := Edit{
				FilePath: testFile,
				Format:   EditFormatWhole,
				Content:  tt.newContent,
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

				if string(result) != tt.newContent {
					t.Errorf("Content mismatch:\nGot:  %q\nWant: %q", string(result), tt.newContent)
				}
			}
		})
	}
}

func TestWholeEditorNewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "whole_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file in non-existent subdirectory
	testFile := filepath.Join(tmpDir, "subdir", "new.txt")
	content := "new file content"

	editor := NewWholeEditor()
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatWhole,
		Content:  content,
	}

	if err := editor.Apply(edit); err != nil {
		t.Fatalf("Failed to apply edit: %v", err)
	}

	// Verify file was created
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	if string(result) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(result), content)
	}
}

func TestWholeEditorValidateGoSyntax(t *testing.T) {
	editor := NewWholeEditor()

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "Valid Go file",
			content: `package main

import "fmt"

func main() {
	fmt.Println("Hello")
}`,
			expectError: false,
		},
		{
			name: "Missing package",
			content: `import "fmt"

func main() {
	fmt.Println("Hello")
}`,
			expectError: true,
		},
		{
			name: "Unbalanced braces - extra open",
			content: `package main

func main() {
	{
		fmt.Println("Hello")
	}
`,
			expectError: true,
		},
		{
			name: "Unbalanced braces - extra close",
			content: `package main

func main() {
	fmt.Println("Hello")
}
}`,
			expectError: true,
		},
		{
			name:        "Empty file",
			content:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := editor.validateGoSyntax(tt.content)
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

func TestWholeEditorValidateJSONSyntax(t *testing.T) {
	editor := NewWholeEditor()

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "Valid JSON object",
			content:     `{"key": "value", "number": 123}`,
			expectError: false,
		},
		{
			name:        "Valid JSON array",
			content:     `[1, 2, 3, "test"]`,
			expectError: false,
		},
		{
			name:        "Invalid - no brackets",
			content:     `key: value`,
			expectError: true,
		},
		{
			name:        "Unbalanced braces",
			content:     `{"key": "value"`,
			expectError: true,
		},
		{
			name:        "Unbalanced brackets",
			content:     `[1, 2, 3`,
			expectError: true,
		},
		{
			name:        "Unclosed string",
			content:     `{"key": "value}`,
			expectError: true,
		},
		{
			name:        "Empty content",
			content:     "",
			expectError: true,
		},
		{
			name:        "Nested structure",
			content:     `{"outer": {"inner": [1, 2, {"deep": true}]}}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := editor.validateJSONSyntax(tt.content)
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

func TestWholeEditorValidateYAMLSyntax(t *testing.T) {
	editor := NewWholeEditor()

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "Valid YAML",
			content: `key: value
nested:
  inner: true
  number: 123`,
			expectError: false,
		},
		{
			name: "Valid YAML with list",
			content: `items:
  - item1
  - item2
  - item3`,
			expectError: false,
		},
		{
			name: "YAML with comments",
			content: `# This is a comment
key: value  # inline comment
nested:
  inner: true`,
			expectError: false,
		},
		{
			name: "Invalid - tabs",
			content: "key: value\n\tindented: wrong",
			expectError: true,
		},
		{
			name:        "Invalid - empty key",
			content:     `: value`,
			expectError: true,
		},
		{
			name: "Empty lines are OK",
			content: `key1: value1

key2: value2`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := editor.validateYAMLSyntax(tt.content)
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

func TestWholeEditorValidateSyntax(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "whole_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	editor := NewWholeEditor()

	tests := []struct {
		name        string
		fileName    string
		content     string
		expectError bool
	}{
		{
			name:     "Go file with validation",
			fileName: "test.go",
			content: `package main
func main() {}`,
			expectError: false,
		},
		{
			name:        "JSON file with validation",
			fileName:    "test.json",
			content:     `{"valid": true}`,
			expectError: false,
		},
		{
			name:        "YAML file with validation",
			fileName:    "test.yaml",
			content:     "key: value",
			expectError: false,
		},
		{
			name:        "Unknown file type - no validation",
			fileName:    "test.txt",
			content:     "any content {{{ }}}",
			expectError: false,
		},
		{
			name:        "Go file with error",
			fileName:    "bad.go",
			content:     "func main() {",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.fileName)
			err := editor.validateSyntax(filePath, tt.content)
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

func TestWholeEditorGetFileStats(t *testing.T) {
	editor := NewWholeEditor()

	tests := []struct {
		name         string
		oldContent   string
		newContent   string
		expectedOld  int
		expectedNew  int
		expectedAdd  int
		expectedDel  int
	}{
		{
			name:         "Same line count",
			oldContent:   "line1\nline2\nline3",
			newContent:   "new1\nnew2\nnew3",
			expectedOld:  3,
			expectedNew:  3,
			expectedAdd:  0,
			expectedDel:  0,
		},
		{
			name:         "Lines added",
			oldContent:   "line1\nline2",
			newContent:   "line1\nline2\nline3\nline4",
			expectedOld:  2,
			expectedNew:  4,
			expectedAdd:  2,
			expectedDel:  -2,
		},
		{
			name:         "Lines removed",
			oldContent:   "line1\nline2\nline3\nline4",
			newContent:   "line1\nline2",
			expectedOld:  4,
			expectedNew:  2,
			expectedAdd:  -2,
			expectedDel:  2,
		},
		{
			name:         "Empty to content",
			oldContent:   "",
			newContent:   "line1\nline2",
			expectedOld:  1,
			expectedNew:  2,
			expectedAdd:  1,
			expectedDel:  -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := editor.GetFileStats(tt.oldContent, tt.newContent)
			if stats.OldLineCount != tt.expectedOld {
				t.Errorf("OldLineCount: got %d, want %d", stats.OldLineCount, tt.expectedOld)
			}
			if stats.NewLineCount != tt.expectedNew {
				t.Errorf("NewLineCount: got %d, want %d", stats.NewLineCount, tt.expectedNew)
			}
			if stats.LinesAdded != tt.expectedAdd {
				t.Errorf("LinesAdded: got %d, want %d", stats.LinesAdded, tt.expectedAdd)
			}
			if stats.LinesRemoved != tt.expectedDel {
				t.Errorf("LinesRemoved: got %d, want %d", stats.LinesRemoved, tt.expectedDel)
			}
		})
	}
}

func TestWholeEditorInvalidContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "whole_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")

	editor := NewWholeEditor()

	// Test with non-string content
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatWhole,
		Content:  12345, // Wrong type
	}

	err = editor.Apply(edit)
	if err == nil {
		t.Error("Expected error for non-string content")
	}
}
