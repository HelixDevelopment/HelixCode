package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLineEditorApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "line_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		initial        string
		edits          []LineEdit
		expectedResult string
		expectError    bool
	}{
		{
			name:    "Single line replacement",
			initial: "line1\nline2\nline3",
			edits: []LineEdit{
				{StartLine: 2, EndLine: 2, NewContent: "modified"},
			},
			expectedResult: "line1\nmodified\nline3\n",
			expectError:    false,
		},
		{
			name:    "Multiple line replacement",
			initial: "line1\nline2\nline3\nline4",
			edits: []LineEdit{
				{StartLine: 2, EndLine: 3, NewContent: "new\ncontent"},
			},
			expectedResult: "line1\nnew\ncontent\nline4\n",
			expectError:    false,
		},
		{
			name:    "Insert at beginning",
			initial: "line1\nline2\nline3",
			edits: []LineEdit{
				{StartLine: 1, EndLine: 1, NewContent: "inserted\nline1"},
			},
			expectedResult: "inserted\nline1\nline2\nline3\n",
			expectError:    false,
		},
		{
			name:    "Append at end",
			initial: "line1\nline2",
			edits: []LineEdit{
				{StartLine: 3, EndLine: 3, NewContent: "line3"},
			},
			expectedResult: "line1\nline2\nline3\n",
			expectError:    false,
		},
		{
			name:    "Delete lines",
			initial: "line1\nline2\nline3\nline4",
			edits: []LineEdit{
				{StartLine: 2, EndLine: 3, NewContent: ""},
			},
			expectedResult: "line1\n\nline4\n",
			expectError:    false,
		},
		{
			name:    "Multiple non-overlapping edits",
			initial: "line1\nline2\nline3\nline4\nline5",
			edits: []LineEdit{
				{StartLine: 1, EndLine: 1, NewContent: "modified1"},
				{StartLine: 5, EndLine: 5, NewContent: "modified5"},
			},
			expectedResult: "modified1\nline2\nline3\nline4\nmodified5\n",
			expectError:    false,
		},
		{
			name:    "Invalid start line (too small)",
			initial: "line1\nline2",
			edits: []LineEdit{
				{StartLine: 0, EndLine: 1, NewContent: "bad"},
			},
			expectedResult: "",
			expectError:    true,
		},
		{
			name:    "Invalid end line (less than start)",
			initial: "line1\nline2\nline3",
			edits: []LineEdit{
				{StartLine: 3, EndLine: 1, NewContent: "bad"},
			},
			expectedResult: "",
			expectError:    true,
		},
		{
			name:    "Overlapping edits",
			initial: "line1\nline2\nline3\nline4",
			edits: []LineEdit{
				{StartLine: 1, EndLine: 2, NewContent: "edit1"},
				{StartLine: 2, EndLine: 3, NewContent: "edit2"},
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

			editor := NewLineEditor()
			edit := Edit{
				FilePath: testFile,
				Format:   EditFormatLines,
				Content:  tt.edits,
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
					t.Errorf("Result mismatch:\nGot:\n%q\nWant:\n%q", string(result), tt.expectedResult)
				}
			}
		})
	}
}

func TestLineEditorValidateLineEdit(t *testing.T) {
	editor := NewLineEditor()

	tests := []struct {
		name        string
		edit        LineEdit
		totalLines  int
		expectError bool
	}{
		{
			name:        "Valid edit",
			edit:        LineEdit{StartLine: 1, EndLine: 2, NewContent: "new"},
			totalLines:  5,
			expectError: false,
		},
		{
			name:        "Start line too small",
			edit:        LineEdit{StartLine: 0, EndLine: 1, NewContent: "new"},
			totalLines:  5,
			expectError: true,
		},
		{
			name:        "End before start",
			edit:        LineEdit{StartLine: 3, EndLine: 1, NewContent: "new"},
			totalLines:  5,
			expectError: true,
		},
		{
			name:        "Start beyond file",
			edit:        LineEdit{StartLine: 10, EndLine: 10, NewContent: "new"},
			totalLines:  5,
			expectError: true,
		},
		{
			name:        "Append at end+1",
			edit:        LineEdit{StartLine: 6, EndLine: 6, NewContent: "new"},
			totalLines:  5,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := editor.validateLineEdit(tt.edit, tt.totalLines)
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

func TestLineEditorCheckOverlaps(t *testing.T) {
	editor := NewLineEditor()

	tests := []struct {
		name        string
		edits       []LineEdit
		expectError bool
	}{
		{
			name: "No overlap",
			edits: []LineEdit{
				{StartLine: 1, EndLine: 2, NewContent: "a"},
				{StartLine: 5, EndLine: 6, NewContent: "b"},
			},
			expectError: false,
		},
		{
			name: "Adjacent not overlapping",
			edits: []LineEdit{
				{StartLine: 1, EndLine: 3, NewContent: "a"},
				{StartLine: 4, EndLine: 6, NewContent: "b"},
			},
			expectError: false,
		},
		{
			name: "Overlapping ranges",
			edits: []LineEdit{
				{StartLine: 1, EndLine: 5, NewContent: "a"},
				{StartLine: 3, EndLine: 7, NewContent: "b"},
			},
			expectError: true,
		},
		{
			name: "One inside another",
			edits: []LineEdit{
				{StartLine: 1, EndLine: 10, NewContent: "a"},
				{StartLine: 3, EndLine: 5, NewContent: "b"},
			},
			expectError: true,
		},
		{
			name: "Same range",
			edits: []LineEdit{
				{StartLine: 5, EndLine: 7, NewContent: "a"},
				{StartLine: 5, EndLine: 7, NewContent: "b"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := editor.checkOverlaps(tt.edits)
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

func TestLineEditorInsertLines(t *testing.T) {
	editor := NewLineEditor()

	lines := []string{"line1", "line2", "line3"}
	newLines := []string{"inserted1", "inserted2"}

	result := editor.InsertLines(lines, 1, newLines)

	expected := []string{"line1", "inserted1", "inserted2", "line2", "line3"}

	if len(result) != len(expected) {
		t.Errorf("Length mismatch: got %d, want %d", len(result), len(expected))
	}

	for i := range expected {
		if i < len(result) && result[i] != expected[i] {
			t.Errorf("Line %d mismatch: got %q, want %q", i, result[i], expected[i])
		}
	}
}

func TestLineEditorDeleteLines(t *testing.T) {
	editor := NewLineEditor()

	lines := []string{"line1", "line2", "line3", "line4", "line5"}

	result := editor.DeleteLines(lines, 1, 3)

	expected := []string{"line1", "line5"}

	if len(result) != len(expected) {
		t.Errorf("Length mismatch: got %d, want %d", len(result), len(expected))
	}

	for i := range expected {
		if i < len(result) && result[i] != expected[i] {
			t.Errorf("Line %d mismatch: got %q, want %q", i, result[i], expected[i])
		}
	}
}

func TestLineEditorReplaceLines(t *testing.T) {
	editor := NewLineEditor()

	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	newLines := []string{"new1", "new2"}

	result := editor.ReplaceLines(lines, 1, 3, newLines)

	expected := []string{"line1", "new1", "new2", "line5"}

	if len(result) != len(expected) {
		t.Errorf("Length mismatch: got %d, want %d", len(result), len(expected))
	}

	for i := range expected {
		if i < len(result) && result[i] != expected[i] {
			t.Errorf("Line %d mismatch: got %q, want %q", i, result[i], expected[i])
		}
	}
}

func TestLineEditorGetLineRange(t *testing.T) {
	editor := NewLineEditor()

	lines := []string{"line1", "line2", "line3", "line4", "line5"}

	tests := []struct {
		name     string
		start    int
		end      int
		expected []string
	}{
		{
			name:     "Middle range",
			start:    1,
			end:      3,
			expected: []string{"line2", "line3", "line4"},
		},
		{
			name:     "Single line",
			start:    2,
			end:      2,
			expected: []string{"line3"},
		},
		{
			name:     "Full range",
			start:    0,
			end:      4,
			expected: lines,
		},
		{
			name:     "Invalid range (start > end)",
			start:    3,
			end:      1,
			expected: []string{},
		},
		{
			name:     "Out of bounds",
			start:    5,
			end:      10,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := editor.GetLineRange(lines, tt.start, tt.end)

			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch: got %d, want %d", len(result), len(tt.expected))
			}

			for i := range tt.expected {
				if i < len(result) && result[i] != tt.expected[i] {
					t.Errorf("Line %d mismatch: got %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestLineEditorGetStats(t *testing.T) {
	editor := NewLineEditor()

	lines := []string{"line1", "line2", "line3", "line4", "line5"}

	tests := []struct {
		name             string
		edits            []LineEdit
		expectedInserted int
		expectedDeleted  int
		expectedModified int
	}{
		{
			name: "Replace with more lines",
			edits: []LineEdit{
				{StartLine: 2, EndLine: 2, NewContent: "new1\nnew2\nnew3"},
			},
			expectedInserted: 2,
			expectedDeleted:  0,
			expectedModified: 0,
		},
		{
			name: "Replace with fewer lines",
			edits: []LineEdit{
				{StartLine: 2, EndLine: 4, NewContent: "new1"},
			},
			expectedInserted: 0,
			expectedDeleted:  2,
			expectedModified: 0,
		},
		{
			name: "Replace with same count",
			edits: []LineEdit{
				{StartLine: 2, EndLine: 3, NewContent: "new1\nnew2"},
			},
			expectedInserted: 0,
			expectedDeleted:  0,
			expectedModified: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := editor.GetStats(lines, tt.edits)

			if stats.LinesInserted != tt.expectedInserted {
				t.Errorf("Inserted mismatch: got %d, want %d", stats.LinesInserted, tt.expectedInserted)
			}
			if stats.LinesDeleted != tt.expectedDeleted {
				t.Errorf("Deleted mismatch: got %d, want %d", stats.LinesDeleted, tt.expectedDeleted)
			}
			if stats.LinesModified != tt.expectedModified {
				t.Errorf("Modified mismatch: got %d, want %d", stats.LinesModified, tt.expectedModified)
			}
		})
	}
}

func TestLineEditorApplySingleLineEdit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "line_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	initial := "line1\nline2\nline3"

	if err := os.WriteFile(testFile, []byte(initial), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewLineEditor()
	lineEdit := LineEdit{
		StartLine:  2,
		EndLine:    2,
		NewContent: "modified",
	}

	if err := editor.ApplySingleLineEdit(testFile, lineEdit); err != nil {
		t.Fatalf("Failed to apply edit: %v", err)
	}

	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	expected := "line1\nmodified\nline3\n"
	if string(result) != expected {
		t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", string(result), expected)
	}
}

func TestLineEditorLargeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "line_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "large.txt")

	// Create large file
	var lines []string
	for i := 0; i < 10000; i++ {
		lines = append(lines, "line "+strings.Repeat("x", 100))
	}
	content := strings.Join(lines, "\n")

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewLineEditor()
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatLines,
		Content: []LineEdit{
			{StartLine: 5000, EndLine: 5000, NewContent: "modified line"},
		},
	}

	if err := editor.Apply(edit); err != nil {
		t.Fatalf("Failed to apply edit: %v", err)
	}

	// Verify modification
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultLines := strings.Split(string(result), "\n")
	if !strings.Contains(resultLines[4999], "modified line") {
		t.Error("Expected modified line not found at correct position")
	}
}

func TestLineEditorEmptyContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "line_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewLineEditor()
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatLines,
		Content:  []LineEdit{},
	}

	err = editor.Apply(edit)
	if err == nil {
		t.Error("Expected error for empty edits")
	}
}
