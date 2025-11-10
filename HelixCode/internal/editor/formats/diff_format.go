package formats

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DiffFormat handles standard unified diff format
type DiffFormat struct{}

// NewDiffFormat creates a new diff format handler
func NewDiffFormat() *DiffFormat {
	return &DiffFormat{}
}

// Type returns the format type
func (df *DiffFormat) Type() FormatType {
	return FormatTypeDiff
}

// Name returns the human-readable name
func (df *DiffFormat) Name() string {
	return "Unified Diff"
}

// Description returns the format description
func (df *DiffFormat) Description() string {
	return "Standard unified diff format with @@ hunks"
}

// CanHandle checks if this format can handle the given content
func (df *DiffFormat) CanHandle(content string) bool {
	// Look for diff markers
	markers := []string{
		"---",  // Original file marker
		"+++",  // Modified file marker
		"@@",   // Hunk marker
		"diff", // Diff header
	}

	markerCount := 0
	for _, marker := range markers {
		if strings.Contains(content, marker) {
			markerCount++
		}
	}

	// Require at least 2 markers to consider it a diff
	return markerCount >= 2
}

// Parse parses the edit content and returns structured edits
func (df *DiffFormat) Parse(ctx context.Context, content string) ([]*FileEdit, error) {
	edits := make([]*FileEdit, 0)

	// Split into individual diffs (separated by "diff" or "---" headers)
	diffSections := df.splitDiffs(content)

	for _, section := range diffSections {
		edit, err := df.parseSingleDiff(section)
		if err != nil {
			// Skip invalid sections but continue processing
			continue
		}

		if edit != nil {
			if err := ValidateEdit(edit); err != nil {
				return nil, fmt.Errorf("invalid edit for %s: %w", edit.FilePath, err)
			}
			edits = append(edits, edit)
		}
	}

	if len(edits) == 0 {
		return nil, fmt.Errorf("no valid diff edits found in content")
	}

	return edits, nil
}

// splitDiffs splits content into individual diff sections
func (df *DiffFormat) splitDiffs(content string) []string {
	sections := make([]string, 0)

	// Split by diff headers or --- markers
	lines := strings.Split(content, "\n")
	currentSection := strings.Builder{}

	for _, line := range lines {
		// Start of new diff section
		if strings.HasPrefix(line, "diff ") ||
			(strings.HasPrefix(line, "--- ") && currentSection.Len() > 0) {
			if currentSection.Len() > 0 {
				sections = append(sections, currentSection.String())
				currentSection.Reset()
			}
		}

		currentSection.WriteString(line)
		currentSection.WriteString("\n")
	}

	// Add last section
	if currentSection.Len() > 0 {
		sections = append(sections, currentSection.String())
	}

	return sections
}

// parseSingleDiff parses a single diff section
func (df *DiffFormat) parseSingleDiff(content string) (*FileEdit, error) {
	lines := strings.Split(content, "\n")

	var filePath string

	// Extract file path from --- or +++ lines
	filePathPattern := regexp.MustCompile(`^(?:---|\+\+\+)\s+(?:a/|b/)?(.+?)(?:\s+|$)`)

	for _, line := range lines {
		if match := filePathPattern.FindStringSubmatch(line); match != nil {
			// Use +++ line for the file path (modified file)
			if strings.HasPrefix(line, "+++") {
				filePath = strings.TrimSpace(match[1])
			}
		}
	}

	if filePath == "" {
		return nil, fmt.Errorf("no file path found in diff")
	}

	// Parse hunks and reconstruct content
	hunks := df.parseHunks(content)
	if len(hunks) == 0 {
		return nil, fmt.Errorf("no hunks found in diff")
	}

	edit := &FileEdit{
		FilePath:  filePath,
		Operation: EditOperationUpdate,
		Metadata: map[string]interface{}{
			"format": "diff",
			"hunks":  hunks,
		},
	}

	return edit, nil
}

// Hunk represents a diff hunk
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []string
}

// parseHunks parses diff hunks from content
func (df *DiffFormat) parseHunks(content string) []*Hunk {
	hunks := make([]*Hunk, 0)

	// Pattern: @@ -old_start,old_count +new_start,new_count @@
	hunkPattern := regexp.MustCompile(`@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@`)

	lines := strings.Split(content, "\n")
	var currentHunk *Hunk

	for _, line := range lines {
		if match := hunkPattern.FindStringSubmatch(line); match != nil {
			// Save previous hunk
			if currentHunk != nil {
				hunks = append(hunks, currentHunk)
			}

			// Parse hunk header
			oldStart, _ := strconv.Atoi(match[1])
			oldCount := 1
			if match[2] != "" {
				oldCount, _ = strconv.Atoi(match[2])
			}
			newStart, _ := strconv.Atoi(match[3])
			newCount := 1
			if match[4] != "" {
				newCount, _ = strconv.Atoi(match[4])
			}

			currentHunk = &Hunk{
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
				Lines:    make([]string, 0),
			}
		} else if currentHunk != nil {
			// Add line to current hunk
			currentHunk.Lines = append(currentHunk.Lines, line)
		}
	}

	// Save last hunk
	if currentHunk != nil {
		hunks = append(hunks, currentHunk)
	}

	return hunks
}

// Format formats structured edits into this format's representation
func (df *DiffFormat) Format(edits []*FileEdit) (string, error) {
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits to format")
	}

	var sb strings.Builder

	for i, edit := range edits {
		if i > 0 {
			sb.WriteString("\n")
		}

		sb.WriteString(fmt.Sprintf("--- a/%s\n", edit.FilePath))
		sb.WriteString(fmt.Sprintf("+++ b/%s\n", edit.FilePath))

		// If hunks are provided in metadata, use them
		if hunks, ok := edit.Metadata["hunks"].([]*Hunk); ok {
			for _, hunk := range hunks {
				sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
					hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount))
				for _, line := range hunk.Lines {
					sb.WriteString(line)
					sb.WriteString("\n")
				}
			}
		} else {
			// Simple format: show entire content as added
			newLines := strings.Split(edit.NewContent, "\n")
			sb.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(newLines)))
			for _, line := range newLines {
				sb.WriteString("+")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}

	return sb.String(), nil
}

// PromptTemplate returns the prompt template for this format
func (df *DiffFormat) PromptTemplate() string {
	return `When editing files, provide changes in unified diff format:

--- a/<file_path>
+++ b/<file_path>
@@ -old_start,old_count +new_start,new_count @@
 context line
-removed line
+added line
 context line

Example:

--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,6 @@
 package main

 import "fmt"
+import "os"

 func main() {
@@ -10,3 +11,4 @@
     fmt.Println("Hello, World!")
+    os.Exit(0)
 }

Important:
- Lines starting with '-' are removed
- Lines starting with '+' are added
- Lines starting with ' ' (space) are context
- Include @@ hunk headers with line numbers
`
}

// Validate validates that the format is correctly used
func (df *DiffFormat) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Check for required markers
	if !strings.Contains(content, "@@") {
		return fmt.Errorf("diff must contain hunk markers (@@)")
	}

	// Try to parse and see if we get any edits
	edits, err := df.Parse(context.Background(), content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(edits) == 0 {
		return fmt.Errorf("no valid edits found")
	}

	return nil
}
