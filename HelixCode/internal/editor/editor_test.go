package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCodeEditor(t *testing.T) {
	tests := []struct {
		name        string
		format      EditFormat
		expectError bool
	}{
		{"Valid diff format", EditFormatDiff, false},
		{"Valid whole format", EditFormatWhole, false},
		{"Valid search/replace format", EditFormatSearchReplace, false},
		{"Valid lines format", EditFormatLines, false},
		{"Invalid format", EditFormat("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor, err := NewCodeEditor(tt.format)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if editor == nil {
					t.Error("Expected editor but got nil")
				}
				if editor.GetFormat() != tt.format {
					t.Errorf("Expected format %s, got %s", tt.format, editor.GetFormat())
				}
			}
		})
	}
}

func TestCodeEditorSetFormat(t *testing.T) {
	editor, err := NewCodeEditor(EditFormatDiff)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}

	tests := []struct {
		name        string
		format      EditFormat
		expectError bool
	}{
		{"Change to whole", EditFormatWhole, false},
		{"Change to search/replace", EditFormatSearchReplace, false},
		{"Change to lines", EditFormatLines, false},
		{"Invalid format", EditFormat("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := editor.SetFormat(tt.format)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if editor.GetFormat() != tt.format {
					t.Errorf("Expected format %s, got %s", tt.format, editor.GetFormat())
				}
			}
		})
	}
}

func TestCodeEditorValidateEdit(t *testing.T) {
	editor, err := NewCodeEditor(EditFormatDiff)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}

	tests := []struct {
		name        string
		edit        Edit
		expectError bool
	}{
		{
			name: "Valid diff edit",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormatDiff,
				Content:  "diff content",
			},
			expectError: false,
		},
		{
			name: "Missing file path",
			edit: Edit{
				FilePath: "",
				Format:   EditFormatDiff,
				Content:  "diff content",
			},
			expectError: true,
		},
		{
			name: "Missing content",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormatDiff,
				Content:  nil,
			},
			expectError: true,
		},
		{
			name: "Invalid format",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormat("invalid"),
				Content:  "content",
			},
			expectError: true,
		},
		{
			name: "Wrong content type for diff",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormatDiff,
				Content:  123,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := editor.ValidateEdit(tt.edit)
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

func TestCodeEditorBackup(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "editor_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "original content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor, err := NewCodeEditor(EditFormatWhole)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}

	// Apply edit with backup
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatWhole,
		Content:  "new content",
		Backup:   true,
	}

	if err := editor.ApplyEdit(edit); err != nil {
		t.Fatalf("Failed to apply edit: %v", err)
	}

	// Check backup was created
	backupFile := testFile + ".bak"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}

	// Verify backup content
	backupContent, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}
	if string(backupContent) != originalContent {
		t.Errorf("Backup content mismatch: got %q, want %q", string(backupContent), originalContent)
	}

	// Verify new content
	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}
	if string(newContent) != "new content" {
		t.Errorf("Modified content mismatch: got %q, want %q", string(newContent), "new content")
	}
}

func TestCodeEditorConcurrentEdits(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "editor_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	numFiles := 10
	for i := 0; i < numFiles; i++ {
		testFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".txt")
		content := "file " + string(rune('0'+i))
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	editor, err := NewCodeEditor(EditFormatWhole)
	if err != nil {
		t.Fatalf("Failed to create editor: %v", err)
	}

	// Apply edits concurrently (but mutex should serialize them)
	done := make(chan bool, numFiles)
	for i := 0; i < numFiles; i++ {
		go func(index int) {
			testFile := filepath.Join(tmpDir, "test"+string(rune('0'+index))+".txt")
			edit := Edit{
				FilePath: testFile,
				Format:   EditFormatWhole,
				Content:  "modified " + string(rune('0'+index)),
			}
			if err := editor.ApplyEdit(edit); err != nil {
				t.Errorf("Failed to apply edit %d: %v", index, err)
			}
			done <- true
		}(i)
	}

	// Wait for all edits to complete
	for i := 0; i < numFiles; i++ {
		<-done
	}

	// Verify all files were modified correctly
	for i := 0; i < numFiles; i++ {
		testFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".txt")
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file %d: %v", i, err)
			continue
		}
		expected := "modified " + string(rune('0'+i))
		if string(content) != expected {
			t.Errorf("File %d content mismatch: got %q, want %q", i, string(content), expected)
		}
	}
}

func TestDefaultValidator(t *testing.T) {
	validator := NewDefaultValidator()

	tests := []struct {
		name        string
		edit        Edit
		expectError bool
	}{
		{
			name: "Valid edit",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormatDiff,
				Content:  "content",
			},
			expectError: false,
		},
		{
			name: "Empty file path",
			edit: Edit{
				FilePath: "",
				Format:   EditFormatDiff,
				Content:  "content",
			},
			expectError: true,
		},
		{
			name: "Invalid format",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormat("invalid"),
				Content:  "content",
			},
			expectError: true,
		},
		{
			name: "Nil content",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormatDiff,
				Content:  nil,
			},
			expectError: true,
		},
		{
			name: "Wrong type for diff",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormatDiff,
				Content:  123,
			},
			expectError: true,
		},
		{
			name: "Wrong type for search/replace",
			edit: Edit{
				FilePath: "/tmp/test.txt",
				Format:   EditFormatSearchReplace,
				Content:  "string",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.edit)
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

func TestCodeEditorApplyEditIntegration(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "editor_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		format         EditFormat
		initialContent string
		editContent    interface{}
		expectedResult string
		expectError    bool
	}{
		{
			name:           "Whole file replacement",
			format:         EditFormatWhole,
			initialContent: "old content",
			editContent:    "new content",
			expectedResult: "new content",
			expectError:    false,
		},
		{
			name:           "Search replace",
			format:         EditFormatSearchReplace,
			initialContent: "hello world",
			editContent: []SearchReplace{
				{Search: "world", Replace: "universe", Count: -1, Regex: false},
			},
			expectedResult: "hello universe",
			expectError:    false,
		},
		{
			name:           "Line edit",
			format:         EditFormatLines,
			initialContent: "line1\nline2\nline3",
			editContent: []LineEdit{
				{StartLine: 2, EndLine: 2, NewContent: "modified"},
			},
			expectedResult: "line1\nmodified\nline3\n",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".txt")

			// Create initial file
			if err := os.WriteFile(testFile, []byte(tt.initialContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			editor, err := NewCodeEditor(tt.format)
			if err != nil {
				t.Fatalf("Failed to create editor: %v", err)
			}

			edit := Edit{
				FilePath: testFile,
				Format:   tt.format,
				Content:  tt.editContent,
			}

			err = editor.ApplyEdit(edit)
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
