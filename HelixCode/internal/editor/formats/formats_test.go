package formats

import (
	"context"
	"strings"
	"testing"
)

// TestFormatRegistry tests the format registry
func TestFormatRegistry(t *testing.T) {
	t.Run("register format", func(t *testing.T) {
		registry := NewFormatRegistry()
		format := NewWholeFormat()

		err := registry.Register(format)
		if err != nil {
			t.Fatalf("failed to register format: %v", err)
		}

		retrieved, err := registry.Get(FormatTypeWhole)
		if err != nil {
			t.Fatalf("failed to get format: %v", err)
		}

		if retrieved.Type() != FormatTypeWhole {
			t.Errorf("expected type %s, got %s", FormatTypeWhole, retrieved.Type())
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		registry := NewFormatRegistry()
		format := NewWholeFormat()

		_ = registry.Register(format)
		err := registry.Register(format)

		if err == nil {
			t.Error("expected error for duplicate registration")
		}
	})

	t.Run("get non-existent format", func(t *testing.T) {
		registry := NewFormatRegistry()

		_, err := registry.Get("nonexistent")
		if err == nil {
			t.Error("expected error for non-existent format")
		}
	})

	t.Run("list formats", func(t *testing.T) {
		registry, err := RegisterAllFormats()
		if err != nil {
			t.Fatalf("failed to register all formats: %v", err)
		}

		formats := registry.ListFormats()
		if len(formats) != 8 {
			t.Errorf("expected 8 formats, got %d", len(formats))
		}
	})
}

// TestWholeFormat tests the whole-file format
func TestWholeFormat(t *testing.T) {
	format := NewWholeFormat()

	t.Run("can handle", func(t *testing.T) {
		content := "File: test.go\n```go\npackage main\n```"
		if !format.CanHandle(content) {
			t.Error("should handle content with code blocks")
		}
	})

	t.Run("parse simple", func(t *testing.T) {
		content := "File: test.go\n```go\npackage main\n\nfunc main() {}\n```"
		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.FilePath != "test.go" {
			t.Errorf("expected FilePath 'test.go', got '%s'", edit.FilePath)
		}

		expectedContent := "package main\n\nfunc main() {}"
		if edit.NewContent != expectedContent {
			t.Errorf("expected content:\n%s\ngot:\n%s", expectedContent, edit.NewContent)
		}
	})

	t.Run("parse alternative format", func(t *testing.T) {
		content := "```test.go\npackage main\n```"
		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}
	})

	t.Run("format", func(t *testing.T) {
		edit := &FileEdit{
			FilePath:   "test.go",
			Operation:  EditOperationUpdate,
			NewContent: "package main",
		}

		formatted, err := format.Format([]*FileEdit{edit})
		if err != nil {
			t.Fatalf("format failed: %v", err)
		}

		if !strings.Contains(formatted, "File: test.go") {
			t.Error("formatted output should contain file path")
		}
		if !strings.Contains(formatted, "package main") {
			t.Error("formatted output should contain content")
		}
	})

	t.Run("validate", func(t *testing.T) {
		validContent := "File: test.go\n```\ncontent\n```"
		if err := format.Validate(validContent); err != nil {
			t.Errorf("validation should pass: %v", err)
		}

		invalidContent := "no code blocks here"
		if err := format.Validate(invalidContent); err == nil {
			t.Error("validation should fail for content without code blocks")
		}
	})
}

// TestDiffFormat tests the diff format
func TestDiffFormat(t *testing.T) {
	format := NewDiffFormat()

	t.Run("can handle", func(t *testing.T) {
		content := "--- a/test.go\n+++ b/test.go\n@@ -1,3 +1,4 @@"
		if !format.CanHandle(content) {
			t.Error("should handle diff content")
		}
	})

	t.Run("parse diff", func(t *testing.T) {
		content := `--- a/test.go
+++ b/test.go
@@ -1,3 +1,4 @@
 package main

 import "fmt"
+import "os"
`
		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.FilePath != "test.go" {
			t.Errorf("expected FilePath 'test.go', got '%s'", edit.FilePath)
		}
	})

	t.Run("validate", func(t *testing.T) {
		validContent := "--- a/test.go\n+++ b/test.go\n@@ -1,1 +1,1 @@\n-old\n+new"
		if err := format.Validate(validContent); err != nil {
			t.Errorf("validation should pass: %v", err)
		}

		invalidContent := "no diff markers"
		if err := format.Validate(invalidContent); err == nil {
			t.Error("validation should fail for content without diff markers")
		}
	})
}

// TestUDiffFormat tests the git unified diff format
func TestUDiffFormat(t *testing.T) {
	format := NewUDiffFormat()

	t.Run("can handle", func(t *testing.T) {
		content := "diff --git a/test.go b/test.go\nindex 123..456\n--- a/test.go\n+++ b/test.go"
		if !format.CanHandle(content) {
			t.Error("should handle git diff content")
		}
	})

	t.Run("parse git diff", func(t *testing.T) {
		content := `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
`
		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.FilePath != "test.go" {
			t.Errorf("expected FilePath 'test.go', got '%s'", edit.FilePath)
		}
	})

	t.Run("parse new file", func(t *testing.T) {
		content := `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/new.go
`
		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.Operation != EditOperationCreate {
			t.Errorf("expected operation CREATE, got %s", edit.Operation)
		}
	})

	t.Run("parse rename", func(t *testing.T) {
		content := `diff --git a/old.go b/new.go
similarity index 100%
rename from old.go
rename to new.go
`
		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.Operation != EditOperationRename {
			t.Errorf("expected operation RENAME, got %s", edit.Operation)
		}
	})
}

// TestSearchReplaceFormat tests the search/replace format
func TestSearchReplaceFormat(t *testing.T) {
	format := NewSearchReplaceFormat()

	t.Run("can handle", func(t *testing.T) {
		content := "File: test.go\n<<<<<<< SEARCH\nold\n=======\nnew\n>>>>>>> REPLACE"
		if !format.CanHandle(content) {
			t.Error("should handle search/replace content")
		}
	})

	t.Run("parse block style", func(t *testing.T) {
		content := `File: test.go
<<<<<<< SEARCH
func oldFunction() {
    return "old"
}
=======
func newFunction() {
    return "new"
}
>>>>>>> REPLACE`

		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.FilePath != "test.go" {
			t.Errorf("expected FilePath 'test.go', got '%s'", edit.FilePath)
		}

		if !strings.Contains(edit.SearchPattern, "oldFunction") {
			t.Error("search pattern should contain 'oldFunction'")
		}

		if !strings.Contains(edit.ReplaceWith, "newFunction") {
			t.Error("replace text should contain 'newFunction'")
		}
	})

	t.Run("validate", func(t *testing.T) {
		validContent := "File: test.go\nSEARCH:\nold\nREPLACE:\nnew"
		if err := format.Validate(validContent); err != nil {
			t.Errorf("validation should pass: %v", err)
		}

		invalidContent := "no search or replace markers"
		if err := format.Validate(invalidContent); err == nil {
			t.Error("validation should fail for content without markers")
		}
	})
}

// TestEditorFormat tests the editor format
func TestEditorFormat(t *testing.T) {
	format := NewEditorFormat()

	t.Run("can handle", func(t *testing.T) {
		content := "File: test.go\nINSERT AT LINE 5:\nimport \"fmt\""
		if !format.CanHandle(content) {
			t.Error("should handle editor content")
		}
	})

	t.Run("parse operations", func(t *testing.T) {
		content := `File: test.go
INSERT AT LINE 5:
import "fmt"

DELETE LINE 10

REPLACE LINE 15:
func newFunction() {
`

		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		operations := edit.Metadata["operations"].([]*LineOperation)

		if len(operations) != 3 {
			t.Errorf("expected 3 operations, got %d", len(operations))
		}

		// Check operation types
		if operations[0].Type != "insert" {
			t.Errorf("expected first operation to be insert, got %s", operations[0].Type)
		}
		if operations[1].Type != "delete" {
			t.Errorf("expected second operation to be delete, got %s", operations[1].Type)
		}
		if operations[2].Type != "replace" {
			t.Errorf("expected third operation to be replace, got %s", operations[2].Type)
		}
	})

	t.Run("validate", func(t *testing.T) {
		validContent := "File: test.go\nINSERT AT LINE 1:\ntest"
		if err := format.Validate(validContent); err != nil {
			t.Errorf("validation should pass: %v", err)
		}

		invalidContent := "no file marker"
		if err := format.Validate(invalidContent); err == nil {
			t.Error("validation should fail for content without File marker")
		}
	})
}

// TestArchitectFormat tests the architect format
func TestArchitectFormat(t *testing.T) {
	format := NewArchitectFormat()

	t.Run("can handle", func(t *testing.T) {
		content := "CREATE FILE: test.go\n```\npackage main\n```"
		if !format.CanHandle(content) {
			t.Error("should handle architect content")
		}
	})

	t.Run("parse create", func(t *testing.T) {
		content := `CREATE FILE: test.go
` + "```go\npackage main\n```"

		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.Operation != EditOperationCreate {
			t.Errorf("expected operation CREATE, got %s", edit.Operation)
		}
		if edit.FilePath != "test.go" {
			t.Errorf("expected FilePath 'test.go', got '%s'", edit.FilePath)
		}
	})

	t.Run("parse delete", func(t *testing.T) {
		content := "DELETE FILE: old.go"

		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.Operation != EditOperationDelete {
			t.Errorf("expected operation DELETE, got %s", edit.Operation)
		}
	})

	t.Run("parse rename", func(t *testing.T) {
		content := "RENAME FILE: old.go TO new.go"

		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		if edit.Operation != EditOperationRename {
			t.Errorf("expected operation RENAME, got %s", edit.Operation)
		}
	})

	t.Run("validate", func(t *testing.T) {
		validContent := "CREATE FILE: test.go"
		if err := format.Validate(validContent); err != nil {
			t.Errorf("validation should pass: %v", err)
		}

		invalidContent := "no architect operations"
		if err := format.Validate(invalidContent); err == nil {
			t.Error("validation should fail for content without operations")
		}
	})
}

// TestAskFormat tests the ask format
func TestAskFormat(t *testing.T) {
	format := NewAskFormat()

	t.Run("can handle", func(t *testing.T) {
		content := "QUESTION: Should I proceed with this change?"
		if !format.CanHandle(content) {
			t.Error("should handle ask content")
		}
	})

	t.Run("parse question", func(t *testing.T) {
		content := `QUESTION: Should I use mutex or channel?
File: worker/pool.go
Context: Managing worker queue`

		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		questions := edit.Metadata["questions"].([]*Question)

		if len(questions) != 1 {
			t.Errorf("expected 1 question, got %d", len(questions))
		}

		question := questions[0]
		if !strings.Contains(question.QuestionText, "mutex or channel") {
			t.Error("question should contain expected text")
		}
	})

	t.Run("parse proposal", func(t *testing.T) {
		content := `PROPOSED CHANGE:
File: auth/middleware.go
Description: Add JWT validation
Rationale: Security improvement`

		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		edit := edits[0]
		proposals := edit.Metadata["proposals"].([]*Proposal)

		if len(proposals) != 1 {
			t.Errorf("expected 1 proposal, got %d", len(proposals))
		}

		proposal := proposals[0]
		if proposal.FilePath != "auth/middleware.go" {
			t.Errorf("expected FilePath 'auth/middleware.go', got '%s'", proposal.FilePath)
		}
	})

	t.Run("validate", func(t *testing.T) {
		validContent := "QUESTION: Should I proceed?"
		if err := format.Validate(validContent); err != nil {
			t.Errorf("validation should pass: %v", err)
		}

		invalidContent := "no questions"
		if err := format.Validate(invalidContent); err == nil {
			t.Error("validation should fail for content without questions")
		}
	})
}

// TestLineNumberFormat tests the line number format
func TestLineNumberFormat(t *testing.T) {
	format := NewLineNumberFormat()

	t.Run("can handle", func(t *testing.T) {
		content := "File: test.go\n1| package main\n2| func main() {}\n3| }"
		if !format.CanHandle(content) {
			t.Error("should handle line number content")
		}
	})

	t.Run("parse numbered lines", func(t *testing.T) {
		content := `File: test.go
1| package main
2|
3| func main() {
4|     println("hello")
5| }`

		edits, err := format.Parse(context.Background(), content)

		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}

		edit := edits[0]
		lines := edit.Metadata["numbered_lines"].([]*NumberedLine)

		if len(lines) != 5 {
			t.Errorf("expected 5 lines, got %d", len(lines))
		}

		// Check first line
		if lines[0].LineNumber != 1 {
			t.Errorf("expected line number 1, got %d", lines[0].LineNumber)
		}
		if lines[0].Content != "package main" {
			t.Errorf("expected content 'package main', got '%s'", lines[0].Content)
		}
	})

	t.Run("format", func(t *testing.T) {
		edit := &FileEdit{
			FilePath:   "test.go",
			Operation:  EditOperationUpdate,
			NewContent: "package main\nfunc main() {}",
		}

		formatted, err := format.Format([]*FileEdit{edit})
		if err != nil {
			t.Fatalf("format failed: %v", err)
		}

		if !strings.Contains(formatted, "1| package main") {
			t.Error("formatted output should contain numbered lines")
		}
	})

	t.Run("validate", func(t *testing.T) {
		validContent := "File: test.go\n1| package main\n2| func main() {}"
		if err := format.Validate(validContent); err != nil {
			t.Errorf("validation should pass: %v", err)
		}

		invalidContent := "no numbered lines"
		if err := format.Validate(invalidContent); err == nil {
			t.Error("validation should fail for content without numbered lines")
		}
	})
}

// TestAutoDetect tests format auto-detection
func TestAutoDetect(t *testing.T) {
	registry, err := RegisterAllFormats()
	if err != nil {
		t.Fatalf("failed to register formats: %v", err)
	}

	tests := []struct {
		name           string
		content        string
		expectedFormat FormatType
	}{
		{
			name:           "whole file",
			content:        "File: test.go\n```go\npackage main\n```",
			expectedFormat: FormatTypeWhole,
		},
		{
			name:           "diff",
			content:        "--- a/test.go\n+++ b/test.go\n@@ -1,1 +1,1 @@\n-old\n+new",
			expectedFormat: FormatTypeDiff,
		},
		{
			name:           "git diff",
			content:        "diff --git a/test.go b/test.go\nindex 123..456",
			expectedFormat: FormatTypeUDiff,
		},
		{
			name:           "search/replace",
			content:        "File: test.go\nSEARCH:\nold\nREPLACE:\nnew",
			expectedFormat: FormatTypeSearchReplace,
		},
		{
			name:           "editor",
			content:        "File: test.go\nINSERT AT LINE 1:\ntest",
			expectedFormat: FormatTypeEditor,
		},
		{
			name:           "architect",
			content:        "CREATE FILE: test.go",
			expectedFormat: FormatTypeArchitect,
		},
		{
			name:           "ask",
			content:        "QUESTION: Should I proceed?",
			expectedFormat: FormatTypeAsk,
		},
		{
			name:           "line number",
			content:        "File: test.go\n1| package main\n2| func main() {}\n3| }",
			expectedFormat: FormatTypeLineNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := registry.DetectFormat(tt.content)
			if err != nil {
				t.Fatalf("detect format failed: %v", err)
			}

			if format.Type() != tt.expectedFormat {
				t.Errorf("expected format %s, got %s", tt.expectedFormat, format.Type())
			}
		})
	}
}

// TestValidateEdit tests edit validation
func TestValidateEdit(t *testing.T) {
	t.Run("valid edit", func(t *testing.T) {
		edit := &FileEdit{
			FilePath:   "test.go",
			Operation:  EditOperationUpdate,
			NewContent: "package main",
		}

		if err := ValidateEdit(edit); err != nil {
			t.Errorf("validation should pass: %v", err)
		}
	})

	t.Run("nil edit", func(t *testing.T) {
		if err := ValidateEdit(nil); err == nil {
			t.Error("validation should fail for nil edit")
		}
	})

	t.Run("empty filepath", func(t *testing.T) {
		edit := &FileEdit{
			FilePath:  "",
			Operation: EditOperationUpdate,
		}

		if err := ValidateEdit(edit); err == nil {
			t.Error("validation should fail for empty filepath")
		}
	})

	t.Run("create without content", func(t *testing.T) {
		edit := &FileEdit{
			FilePath:   "test.go",
			Operation:  EditOperationCreate,
			NewContent: "",
		}

		// Empty content is now allowed for empty files
		if err := ValidateEdit(edit); err != nil {
			t.Errorf("validation should pass for create with empty content: %v", err)
		}
	})

	t.Run("rename without metadata", func(t *testing.T) {
		edit := &FileEdit{
			FilePath:  "new.go",
			Operation: EditOperationRename,
			Metadata:  nil,
		}

		if err := ValidateEdit(edit); err == nil {
			t.Error("validation should fail for rename without new_path in metadata")
		}
	})
}

// TestRegisterAllFormats tests registration of all formats
func TestRegisterAllFormats(t *testing.T) {
	registry, err := RegisterAllFormats()
	if err != nil {
		t.Fatalf("failed to register all formats: %v", err)
	}

	expectedFormats := []FormatType{
		FormatTypeWhole,
		FormatTypeDiff,
		FormatTypeUDiff,
		FormatTypeSearchReplace,
		FormatTypeEditor,
		FormatTypeArchitect,
		FormatTypeAsk,
		FormatTypeLineNumber,
	}

	for _, formatType := range expectedFormats {
		format, err := registry.Get(formatType)
		if err != nil {
			t.Errorf("failed to get format %s: %v", formatType, err)
		}

		if format.Type() != formatType {
			t.Errorf("expected type %s, got %s", formatType, format.Type())
		}
	}
}

// TestGetFormatByName tests getting format by name
func TestGetFormatByName(t *testing.T) {
	tests := []struct {
		name        string
		formatName  string
		expectError bool
	}{
		{"whole", "whole", false},
		{"diff", "diff", false},
		{"udiff", "udiff", false},
		{"search-replace", "search-replace", false},
		{"editor", "editor", false},
		{"architect", "architect", false},
		{"ask", "ask", false},
		{"line-number", "line-number", false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := GetFormatByName(tt.formatName)

			if tt.expectError {
				if err == nil {
					t.Error("expected error for invalid format name")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if format == nil {
					t.Error("expected non-nil format")
				}
			}
		})
	}
}
