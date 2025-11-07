package formats

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// LineNumberFormat handles direct line number editing with line prefixes
type LineNumberFormat struct{}

// NewLineNumberFormat creates a new line number format handler
func NewLineNumberFormat() *LineNumberFormat {
	return &LineNumberFormat{}
}

// Type returns the format type
func (lnf *LineNumberFormat) Type() FormatType {
	return FormatTypeLineNumber
}

// Name returns the human-readable name
func (lnf *LineNumberFormat) Name() string {
	return "Line Number"
}

// Description returns the format description
func (lnf *LineNumberFormat) Description() string {
	return "Direct line number editing with numbered line prefixes"
}

// CanHandle checks if this format can handle the given content
func (lnf *LineNumberFormat) CanHandle(content string) bool {
	// Look for numbered line patterns
	// Pattern: <num>|<content> or <num> <content>
	lineNumPattern := regexp.MustCompile(`(?m)^\s*\d+\s*[|:]\s*.+$`)
	matches := lineNumPattern.FindAllString(content, -1)

	// Need at least 3 consecutive numbered lines to be confident
	return len(matches) >= 3
}

// Parse parses the edit content and returns structured edits
func (lnf *LineNumberFormat) Parse(ctx context.Context, content string) ([]*FileEdit, error) {
	edits := make([]*FileEdit, 0)

	// Pattern: File: <path>\n<numbered lines>
	// Use \z for end of string (not $ which matches end of line in multiline mode)
	filePattern := regexp.MustCompile(`(?ms)File:\s*([^\n]+)\n(.*?)(?:\nFile:|\z)`)
	fileMatches := filePattern.FindAllStringSubmatch(content, -1)

	if len(fileMatches) == 0 {
		return nil, fmt.Errorf("no file sections found")
	}

	for _, fileMatch := range fileMatches {
		filePath := strings.TrimSpace(fileMatch[1])
		numberedContent := fileMatch[2]

		// Parse numbered lines
		lines, err := lnf.parseNumberedLines(numberedContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse numbered lines for %s: %w", filePath, err)
		}

		if len(lines) == 0 {
			continue
		}

		// Reconstruct content from numbered lines
		newContent := lnf.reconstructContent(lines)

		edit := &FileEdit{
			FilePath:   filePath,
			Operation:  EditOperationUpdate,
			NewContent: newContent,
			Metadata: map[string]interface{}{
				"format":        "line-number",
				"numbered_lines": lines,
			},
		}

		if err := ValidateEdit(edit); err != nil {
			return nil, fmt.Errorf("invalid edit for %s: %w", filePath, err)
		}

		edits = append(edits, edit)
	}

	if len(edits) == 0 {
		return nil, fmt.Errorf("no valid line number edits found")
	}

	return edits, nil
}

// NumberedLine represents a line with its number
type NumberedLine struct {
	LineNumber int
	Content    string
}

// parseNumberedLines parses content with line numbers
func (lnf *LineNumberFormat) parseNumberedLines(content string) ([]*NumberedLine, error) {
	lines := make([]*NumberedLine, 0)

	// Patterns to match:
	// 1| content
	// 1: content
	// 1 content
	linePattern := regexp.MustCompile(`(?m)^\s*(\d+)\s*[|:]\s*(.*)$`)

	contentLines := strings.Split(content, "\n")
	for _, line := range contentLines {
		if match := linePattern.FindStringSubmatch(line); match != nil {
			lineNum, _ := strconv.Atoi(match[1])
			lineContent := match[2]

			lines = append(lines, &NumberedLine{
				LineNumber: lineNum,
				Content:    lineContent,
			})
		}
	}

	return lines, nil
}

// reconstructContent reconstructs file content from numbered lines
func (lnf *LineNumberFormat) reconstructContent(lines []*NumberedLine) string {
	if len(lines) == 0 {
		return ""
	}

	var sb strings.Builder

	for i, line := range lines {
		sb.WriteString(line.Content)
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// Format formats structured edits into this format's representation
func (lnf *LineNumberFormat) Format(edits []*FileEdit) (string, error) {
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits to format")
	}

	var sb strings.Builder

	for i, edit := range edits {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		sb.WriteString(fmt.Sprintf("File: %s\n", edit.FilePath))

		// Get numbered lines from metadata, or generate from content
		var lines []*NumberedLine
		if nl, ok := edit.Metadata["numbered_lines"].([]*NumberedLine); ok {
			lines = nl
		} else {
			// Generate from new content
			lines = lnf.generateNumberedLines(edit.NewContent)
		}

		// Format with line numbers
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("%d| %s\n", line.LineNumber, line.Content))
		}
	}

	return sb.String(), nil
}

// generateNumberedLines generates numbered lines from content
func (lnf *LineNumberFormat) generateNumberedLines(content string) []*NumberedLine {
	lines := make([]*NumberedLine, 0)
	contentLines := strings.Split(content, "\n")

	for i, line := range contentLines {
		lines = append(lines, &NumberedLine{
			LineNumber: i + 1,
			Content:    line,
		})
	}

	return lines
}

// PromptTemplate returns the prompt template for this format
func (lnf *LineNumberFormat) PromptTemplate() string {
	return `When editing files, provide content with line numbers:

File: <file_path>
1| <first line content>
2| <second line content>
3| <third line content>
...

Alternative separator styles:
<num>: <content>
<num> <content>

Example:

File: src/main.go
1| package main
2|
3| import (
4|     "fmt"
5|     "os"
6| )
7|
8| func main() {
9|     fmt.Println("Hello, World!")
10|     os.Exit(0)
11| }

File: config.yaml
1: timeout: 60
2: max_retries: 5
3: debug: true

Important:
- Include ALL lines you want in the file
- Line numbers start at 1
- Use consistent spacing and indentation
- Include empty lines where needed (just line number + separator)
- This replaces the entire file content
- Gaps in line numbers are treated as content, not deleted lines
`
}

// Validate validates that the format is correctly used
func (lnf *LineNumberFormat) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Check for file marker
	if !strings.Contains(content, "File:") {
		return fmt.Errorf("content must contain 'File:' marker")
	}

	// Try to parse and see if we get any edits
	edits, err := lnf.Parse(context.Background(), content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(edits) == 0 {
		return fmt.Errorf("no valid edits found")
	}

	// Validate numbered lines
	for _, edit := range edits {
		if lines, ok := edit.Metadata["numbered_lines"].([]*NumberedLine); ok {
			if len(lines) == 0 {
				return fmt.Errorf("no numbered lines found for %s", edit.FilePath)
			}

			// Check for valid line numbers
			for _, line := range lines {
				if line.LineNumber < 1 {
					return fmt.Errorf("invalid line number: %d (must be >= 1)", line.LineNumber)
				}
			}
		}
	}

	return nil
}
