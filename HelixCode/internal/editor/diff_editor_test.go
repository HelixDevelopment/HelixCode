package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffEditorApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "diff_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		originalLines  string
		diffContent    string
		expectedResult string
		expectError    bool
	}{
		{
			name:          "Simple addition",
			originalLines: "line1\nline2\nline3",
			diffContent: `--- test.txt
+++ test.txt
@@ -1,3 +1,4 @@
 line1
+new line
 line2
 line3`,
			expectedResult: "line1\nnew line\nline2\nline3\n",
			expectError:    false,
		},
		{
			name:          "Simple deletion",
			originalLines: "line1\nline2\nline3",
			diffContent: `--- test.txt
+++ test.txt
@@ -1,3 +1,2 @@
 line1
-line2
 line3`,
			expectedResult: "line1\nline3\n",
			expectError:    false,
		},
		{
			name:          "Simple modification",
			originalLines: "line1\nline2\nline3",
			diffContent: `--- test.txt
+++ test.txt
@@ -1,3 +1,3 @@
 line1
-line2
+modified line
 line3`,
			expectedResult: "line1\nmodified line\nline3\n",
			expectError:    false,
		},
		{
			name:          "Multiple hunks",
			originalLines: "line1\nline2\nline3\nline4\nline5",
			diffContent: `--- test.txt
+++ test.txt
@@ -1,2 +1,2 @@
-line1
+modified1
 line2
@@ -4,2 +4,2 @@
 line4
-line5
+modified5`,
			expectedResult: "modified1\nline2\nline3\nline4\nmodified5\n",
			expectError:    false,
		},
		{
			name:          "Context mismatch",
			originalLines: "line1\nline2\nline3",
			diffContent: `--- test.txt
+++ test.txt
@@ -1,3 +1,3 @@
 wrong
-line2
+modified
 line3`,
			expectedResult: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test_"+tt.name+".txt")

			// Create test file
			if err := os.WriteFile(testFile, []byte(tt.originalLines), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			editor := NewDiffEditor()
			edit := Edit{
				FilePath: testFile,
				Format:   EditFormatDiff,
				Content:  tt.diffContent,
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

func TestDiffEditorParseDiff(t *testing.T) {
	editor := NewDiffEditor()

	tests := []struct {
		name        string
		diffContent string
		expectError bool
		numHunks    int
	}{
		{
			name: "Single hunk",
			diffContent: `--- test.txt
+++ test.txt
@@ -1,3 +1,4 @@
 line1
+new
 line2
 line3`,
			expectError: false,
			numHunks:    1,
		},
		{
			name: "Multiple hunks",
			diffContent: `--- test.txt
+++ test.txt
@@ -1,2 +1,2 @@
-old1
+new1
 line2
@@ -5,1 +5,2 @@
 line5
+added`,
			expectError: false,
			numHunks:    2,
		},
		{
			name: "Empty diff",
			diffContent: `--- test.txt
+++ test.txt`,
			expectError: false,
			numHunks:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunks, err := editor.parseDiff(tt.diffContent)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(hunks) != tt.numHunks {
					t.Errorf("Expected %d hunks, got %d", tt.numHunks, len(hunks))
				}
			}
		})
	}
}

func TestDiffEditorParseHunkHeader(t *testing.T) {
	editor := NewDiffEditor()

	tests := []struct {
		name        string
		header      string
		expectError bool
		oldStart    int
		oldCount    int
		newStart    int
		newCount    int
	}{
		{
			name:        "Standard header",
			header:      "@@ -1,3 +1,4 @@",
			expectError: false,
			oldStart:    1,
			oldCount:    3,
			newStart:    1,
			newCount:    4,
		},
		{
			name:        "Single line old",
			header:      "@@ -5 +5,2 @@",
			expectError: false,
			oldStart:    5,
			oldCount:    1,
			newStart:    5,
			newCount:    2,
		},
		{
			name:        "Single line both",
			header:      "@@ -10 +10 @@",
			expectError: false,
			oldStart:    10,
			oldCount:    1,
			newStart:    10,
			newCount:    1,
		},
		{
			name:        "Invalid header",
			header:      "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunk, err := editor.parseHunkHeader(tt.header)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if hunk.OldStart != tt.oldStart {
					t.Errorf("OldStart mismatch: got %d, want %d", hunk.OldStart, tt.oldStart)
				}
				if hunk.OldCount != tt.oldCount {
					t.Errorf("OldCount mismatch: got %d, want %d", hunk.OldCount, tt.oldCount)
				}
				if hunk.NewStart != tt.newStart {
					t.Errorf("NewStart mismatch: got %d, want %d", hunk.NewStart, tt.newStart)
				}
				if hunk.NewCount != tt.newCount {
					t.Errorf("NewCount mismatch: got %d, want %d", hunk.NewCount, tt.newCount)
				}
			}
		})
	}
}

func TestDiffEditorApplyHunks(t *testing.T) {
	editor := NewDiffEditor()

	tests := []struct {
		name          string
		original      []string
		hunks         []DiffHunk
		expected      []string
		expectError   bool
	}{
		{
			name:     "Simple addition",
			original: []string{"line1", "line2", "line3"},
			hunks: []DiffHunk{
				{
					OldStart: 2,
					OldCount: 1,
					NewStart: 2,
					NewCount: 2,
					Lines: []DiffLine{
						{Type: ' ', Content: "line2"},
						{Type: '+', Content: "inserted"},
					},
				},
			},
			expected:    []string{"line1", "line2", "inserted", "line3"},
			expectError: false,
		},
		{
			name:     "Simple deletion",
			original: []string{"line1", "line2", "line3"},
			hunks: []DiffHunk{
				{
					OldStart: 2,
					OldCount: 1,
					NewStart: 2,
					NewCount: 0,
					Lines: []DiffLine{
						{Type: '-', Content: "line2"},
					},
				},
			},
			expected:    []string{"line1", "line3"},
			expectError: false,
		},
		{
			name:     "Context mismatch",
			original: []string{"line1", "line2", "line3"},
			hunks: []DiffHunk{
				{
					OldStart: 2,
					OldCount: 1,
					NewStart: 2,
					NewCount: 1,
					Lines: []DiffLine{
						{Type: ' ', Content: "wrong"},
					},
				},
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := editor.applyHunks(tt.original, tt.hunks)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) != len(tt.expected) {
					t.Errorf("Length mismatch: got %d, want %d", len(result), len(tt.expected))
				}
				for i := range tt.expected {
					if i < len(result) && result[i] != tt.expected[i] {
						t.Errorf("Line %d mismatch: got %q, want %q", i, result[i], tt.expected[i])
					}
				}
			}
		})
	}
}

func TestDiffEditorLargeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "diff_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "large.txt")

	// Create a large file
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "line "+strings.Repeat("x", 100))
	}
	content := strings.Join(lines, "\n")

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create diff that modifies middle of file
	diff := `--- large.txt
+++ large.txt
@@ -500,3 +500,3 @@
 ` + lines[499] + `
-` + lines[500] + `
+modified line
 ` + lines[501]

	editor := NewDiffEditor()
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatDiff,
		Content:  diff,
	}

	if err := editor.Apply(edit); err != nil {
		t.Fatalf("Failed to apply diff: %v", err)
	}

	// Verify modification
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultLines := strings.Split(string(result), "\n")
	if !strings.Contains(resultLines[500], "modified line") {
		t.Error("Expected modified line not found")
	}
}

func TestDiffEditorNewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "diff_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "new.txt")

	// Apply diff to non-existent file (should treat as empty)
	diff := `--- new.txt
+++ new.txt
@@ -0,0 +1,3 @@
+line1
+line2
+line3`

	editor := NewDiffEditor()
	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatDiff,
		Content:  diff,
	}

	if err := editor.Apply(edit); err != nil {
		t.Fatalf("Failed to apply diff: %v", err)
	}

	// Verify file was created with correct content
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	expected := "line1\nline2\nline3\n"
	if string(result) != expected {
		t.Errorf("Result mismatch:\nGot:  %q\nWant: %q", string(result), expected)
	}
}
