package formats

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// UDiffFormat handles Git-style unified diff format
type UDiffFormat struct {
	diffFormat *DiffFormat // Reuse diff format parser
}

// NewUDiffFormat creates a new git-style diff format handler
func NewUDiffFormat() *UDiffFormat {
	return &UDiffFormat{
		diffFormat: NewDiffFormat(),
	}
}

// Type returns the format type
func (udf *UDiffFormat) Type() FormatType {
	return FormatTypeUDiff
}

// Name returns the human-readable name
func (udf *UDiffFormat) Name() string {
	return "Git Unified Diff"
}

// Description returns the format description
func (udf *UDiffFormat) Description() string {
	return "Git-style unified diff with extended headers (index, mode, etc.)"
}

// CanHandle checks if this format can handle the given content
func (udf *UDiffFormat) CanHandle(content string) bool {
	// Look for git-specific markers
	gitMarkers := []string{
		"diff --git",
		"index ",
		"new file mode",
		"deleted file mode",
		"similarity index",
	}

	for _, marker := range gitMarkers {
		if strings.Contains(content, marker) {
			return true
		}
	}

	return false
}

// Parse parses the edit content and returns structured edits
func (udf *UDiffFormat) Parse(ctx context.Context, content string) ([]*FileEdit, error) {
	edits := make([]*FileEdit, 0)

	// Split into individual git diffs
	diffSections := udf.splitGitDiffs(content)

	for _, section := range diffSections {
		edit, err := udf.parseGitDiff(section)
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
		return nil, fmt.Errorf("no valid git diff edits found in content")
	}

	return edits, nil
}

// splitGitDiffs splits content into individual git diff sections
func (udf *UDiffFormat) splitGitDiffs(content string) []string {
	sections := make([]string, 0)

	// Split by "diff --git" headers
	lines := strings.Split(content, "\n")
	currentSection := strings.Builder{}

	for _, line := range lines {
		// Start of new git diff section
		if strings.HasPrefix(line, "diff --git ") && currentSection.Len() > 0 {
			sections = append(sections, currentSection.String())
			currentSection.Reset()
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

// parseGitDiff parses a single git diff section
func (udf *UDiffFormat) parseGitDiff(content string) (*FileEdit, error) {
	lines := strings.Split(content, "\n")

	var filePath string
	var oldFilePath string
	var operation EditOperation = EditOperationUpdate
	metadata := make(map[string]interface{})
	metadata["format"] = "udiff"

	// Pattern: diff --git a/path b/path
	gitDiffPattern := regexp.MustCompile(`^diff --git a/(.+?) b/(.+?)$`)
	// Pattern: index <hash>..<hash> <mode>
	indexPattern := regexp.MustCompile(`^index\s+([a-f0-9]+)\.\.([a-f0-9]+)(?:\s+(\d+))?$`)

	for _, line := range lines {
		// Extract file paths from git diff header
		if match := gitDiffPattern.FindStringSubmatch(line); match != nil {
			oldFilePath = match[1]
			filePath = match[2]
		}

		// Check for new file
		if strings.HasPrefix(line, "new file mode ") {
			operation = EditOperationCreate
			modeStr := strings.TrimPrefix(line, "new file mode ")
			metadata["mode"] = modeStr
		}

		// Check for deleted file
		if strings.HasPrefix(line, "deleted file mode ") {
			operation = EditOperationDelete
		}

		// Check for renamed file
		if strings.HasPrefix(line, "rename from ") {
			oldFilePath = strings.TrimPrefix(line, "rename from ")
			operation = EditOperationRename
		}
		if strings.HasPrefix(line, "rename to ") {
			filePath = strings.TrimPrefix(line, "rename to ")
		}

		// Extract index information
		if match := indexPattern.FindStringSubmatch(line); match != nil {
			metadata["old_hash"] = match[1]
			metadata["new_hash"] = match[2]
			if match[3] != "" {
				metadata["mode"] = match[3]
			}
		}

		// Extract similarity index for renames
		if strings.HasPrefix(line, "similarity index ") {
			similarityStr := strings.TrimPrefix(line, "similarity index ")
			similarityStr = strings.TrimSuffix(similarityStr, "%")
			metadata["similarity"] = similarityStr
		}
	}

	if filePath == "" {
		return nil, fmt.Errorf("no file path found in git diff")
	}

	// Parse hunks using diff format parser
	hunks := udf.diffFormat.parseHunks(content)
	if len(hunks) > 0 {
		metadata["hunks"] = hunks
	}

	edit := &FileEdit{
		FilePath:  filePath,
		Operation: operation,
		Metadata:  metadata,
	}

	// For rename operations, store old path and new path
	if operation == EditOperationRename {
		edit.Metadata["old_path"] = oldFilePath
		edit.Metadata["new_path"] = filePath
	}

	// For create operations, extract content from hunks or set empty
	if operation == EditOperationCreate {
		if len(hunks) > 0 {
			// Extract content from hunks
			var contentBuilder strings.Builder
			for _, hunk := range hunks {
				for _, line := range hunk.Lines {
					if strings.HasPrefix(line, "+") {
						contentBuilder.WriteString(strings.TrimPrefix(line, "+"))
						contentBuilder.WriteString("\n")
					}
				}
			}
			edit.NewContent = contentBuilder.String()
		} else {
			edit.NewContent = "" // Empty new file
		}
	}

	return edit, nil
}

// Format formats structured edits into this format's representation
func (udf *UDiffFormat) Format(edits []*FileEdit) (string, error) {
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits to format")
	}

	var sb strings.Builder

	for i, edit := range edits {
		if i > 0 {
			sb.WriteString("\n")
		}

		// Git diff header
		oldPath := edit.FilePath
		if op, ok := edit.Metadata["old_path"].(string); ok {
			oldPath = op
		}
		sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", oldPath, edit.FilePath))

		// Operation-specific headers
		switch edit.Operation {
		case EditOperationCreate:
			mode := "100644"
			if m, ok := edit.Metadata["mode"].(string); ok {
				mode = m
			}
			sb.WriteString(fmt.Sprintf("new file mode %s\n", mode))
			sb.WriteString(fmt.Sprintf("index 0000000..%s\n", udf.generateHash(edit.NewContent)))

		case EditOperationDelete:
			mode := "100644"
			if m, ok := edit.Metadata["mode"].(string); ok {
				mode = m
			}
			sb.WriteString(fmt.Sprintf("deleted file mode %s\n", mode))
			sb.WriteString(fmt.Sprintf("index %s..0000000\n", udf.generateHash(edit.OldContent)))

		case EditOperationRename:
			if similarity, ok := edit.Metadata["similarity"].(string); ok {
				sb.WriteString(fmt.Sprintf("similarity index %s%%\n", similarity))
			}
			sb.WriteString(fmt.Sprintf("rename from %s\n", oldPath))
			sb.WriteString(fmt.Sprintf("rename to %s\n", edit.FilePath))

		case EditOperationUpdate:
			oldHash := udf.generateHash(edit.OldContent)
			newHash := udf.generateHash(edit.NewContent)
			mode := "100644"
			if m, ok := edit.Metadata["mode"].(string); ok {
				mode = m
			}
			sb.WriteString(fmt.Sprintf("index %s..%s %s\n", oldHash, newHash, mode))
		}

		// File markers
		sb.WriteString(fmt.Sprintf("--- a/%s\n", oldPath))
		sb.WriteString(fmt.Sprintf("+++ b/%s\n", edit.FilePath))

		// Hunks
		if hunks, ok := edit.Metadata["hunks"].([]*Hunk); ok {
			for _, hunk := range hunks {
				sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
					hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount))
				for _, line := range hunk.Lines {
					sb.WriteString(line)
					sb.WriteString("\n")
				}
			}
		}
	}

	return sb.String(), nil
}

// generateHash generates a simple hash (first 7 chars) for demo purposes
func (udf *UDiffFormat) generateHash(content string) string {
	// Simple hash generation (not cryptographically secure)
	// In real implementation, use proper git object hashing
	hash := 0
	for _, c := range content {
		hash = (hash*31 + int(c)) & 0xFFFFFFF
	}
	return fmt.Sprintf("%07x", hash)
}

// PromptTemplate returns the prompt template for this format
func (udf *UDiffFormat) PromptTemplate() string {
	return `When editing files, provide changes in git-style unified diff format:

diff --git a/<file_path> b/<file_path>
index <old_hash>..<new_hash> <mode>
--- a/<file_path>
+++ b/<file_path>
@@ -old_start,old_count +new_start,new_count @@
 context line
-removed line
+added line
 context line

For new files:
diff --git a/<file_path> b/<file_path>
new file mode 100644
index 0000000..<hash>
--- /dev/null
+++ b/<file_path>
@@ -0,0 +1,<lines> @@
+new content

For deleted files:
diff --git a/<file_path> b/<file_path>
deleted file mode 100644
index <hash>..0000000
--- a/<file_path>
+++ /dev/null

For renamed files:
diff --git a/<old_path> b/<new_path>
similarity index <percent>%
rename from <old_path>
rename to <new_path>

Important:
- Include git headers (diff --git, index, mode)
- Use proper hunk markers with line numbers
- Lines with '-' are removed, '+' are added, ' ' are context
`
}

// Validate validates that the format is correctly used
func (udf *UDiffFormat) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Check for git-specific markers
	if !strings.Contains(content, "diff --git") {
		return fmt.Errorf("git diff must contain 'diff --git' header")
	}

	// Try to parse and see if we get any edits
	edits, err := udf.Parse(context.Background(), content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(edits) == 0 {
		return fmt.Errorf("no valid edits found")
	}

	return nil
}
