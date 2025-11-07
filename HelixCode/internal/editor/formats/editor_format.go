package formats

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// EditorFormat handles line-based editing with line numbers
type EditorFormat struct{}

// NewEditorFormat creates a new editor format handler
func NewEditorFormat() *EditorFormat {
	return &EditorFormat{}
}

// Type returns the format type
func (ef *EditorFormat) Type() FormatType {
	return FormatTypeEditor
}

// Name returns the human-readable name
func (ef *EditorFormat) Name() string {
	return "Editor (Line-Based)"
}

// Description returns the format description
func (ef *EditorFormat) Description() string {
	return "Line-based editing with line numbers and operations (insert, delete, replace)"
}

// CanHandle checks if this format can handle the given content
func (ef *EditorFormat) CanHandle(content string) bool {
	// Look for line-based operation markers
	markers := []string{
		"INSERT AT LINE",
		"DELETE LINE",
		"REPLACE LINE",
		"insert at line",
		"delete line",
		"replace line",
	}

	for _, marker := range markers {
		if strings.Contains(strings.ToLower(content), strings.ToLower(marker)) {
			return true
		}
	}

	// Also check for line number patterns like "L123:"
	linePattern := regexp.MustCompile(`(?i)L\d+:`)
	return linePattern.MatchString(content)
}

// Parse parses the edit content and returns structured edits
func (ef *EditorFormat) Parse(ctx context.Context, content string) ([]*FileEdit, error) {
	edits := make([]*FileEdit, 0)

	// Pattern: File: <path>\n<operations>
	// Use \z for end of string (not $ which matches end of line in multiline mode)
	filePattern := regexp.MustCompile(`(?ms)File:\s*([^\n]+)\n(.*?)(?:\nFile:|\z)`)
	fileMatches := filePattern.FindAllStringSubmatch(content, -1)

	if len(fileMatches) == 0 {
		return nil, fmt.Errorf("no file sections found")
	}

	for _, fileMatch := range fileMatches {
		filePath := strings.TrimSpace(fileMatch[1])
		operations := fileMatch[2]

		// Parse individual operations
		ops, err := ef.parseOperations(operations)
		if err != nil {
			return nil, fmt.Errorf("failed to parse operations for %s: %w", filePath, err)
		}

		if len(ops) == 0 {
			continue
		}

		edit := &FileEdit{
			FilePath:  filePath,
			Operation: EditOperationUpdate,
			Metadata: map[string]interface{}{
				"format":     "editor",
				"operations": ops,
			},
		}

		if err := ValidateEdit(edit); err != nil {
			return nil, fmt.Errorf("invalid edit for %s: %w", filePath, err)
		}

		edits = append(edits, edit)
	}

	if len(edits) == 0 {
		return nil, fmt.Errorf("no valid editor edits found")
	}

	return edits, nil
}

// LineOperation represents a line-based operation
type LineOperation struct {
	Type       string // insert, delete, replace
	LineNumber int
	LineCount  int    // For delete operations
	Content    string // For insert/replace operations
}

// parseOperations parses line operations from text
func (ef *EditorFormat) parseOperations(text string) ([]*LineOperation, error) {
	ops := make([]*LineOperation, 0)

	// Pattern: INSERT AT LINE <num>:\n<content>
	insertPattern := regexp.MustCompile(`(?mis)INSERT AT LINE (\d+):\s*\n(.*?)(?:\n(?:INSERT|DELETE|REPLACE)|\z)`)
	insertMatches := insertPattern.FindAllStringSubmatch(text, -1)
	for _, match := range insertMatches {
		lineNum, _ := strconv.Atoi(match[1])
		content := strings.TrimSpace(match[2])
		ops = append(ops, &LineOperation{
			Type:       "insert",
			LineNumber: lineNum,
			Content:    content,
		})
	}

	// Pattern: DELETE LINE <num>[-<end>]
	deletePattern := regexp.MustCompile(`(?mi)DELETE LINE (\d+)(?:-(\d+))?`)
	deleteMatches := deletePattern.FindAllStringSubmatch(text, -1)
	for _, match := range deleteMatches {
		lineNum, _ := strconv.Atoi(match[1])
		lineCount := 1
		if match[2] != "" {
			endLine, _ := strconv.Atoi(match[2])
			lineCount = endLine - lineNum + 1
		}
		ops = append(ops, &LineOperation{
			Type:       "delete",
			LineNumber: lineNum,
			LineCount:  lineCount,
		})
	}

	// Pattern: REPLACE LINE <num>:\n<content>
	replacePattern := regexp.MustCompile(`(?mis)REPLACE LINE (\d+):\s*\n(.*?)(?:\n(?:INSERT|DELETE|REPLACE)|\z)`)
	replaceMatches := replacePattern.FindAllStringSubmatch(text, -1)
	for _, match := range replaceMatches {
		lineNum, _ := strconv.Atoi(match[1])
		content := strings.TrimSpace(match[2])
		ops = append(ops, &LineOperation{
			Type:       "replace",
			LineNumber: lineNum,
			Content:    content,
		})
	}

	// Alternative pattern: L<num>: <content>
	linePattern := regexp.MustCompile(`(?m)^L(\d+):\s*(.*)$`)
	lineMatches := linePattern.FindAllStringSubmatch(text, -1)
	for _, match := range lineMatches {
		lineNum, _ := strconv.Atoi(match[1])
		content := match[2]

		// If content is empty, treat as delete
		if strings.TrimSpace(content) == "" {
			ops = append(ops, &LineOperation{
				Type:       "delete",
				LineNumber: lineNum,
				LineCount:  1,
			})
		} else {
			ops = append(ops, &LineOperation{
				Type:       "replace",
				LineNumber: lineNum,
				Content:    content,
			})
		}
	}

	return ops, nil
}

// Format formats structured edits into this format's representation
func (ef *EditorFormat) Format(edits []*FileEdit) (string, error) {
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits to format")
	}

	var sb strings.Builder

	for i, edit := range edits {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		sb.WriteString(fmt.Sprintf("File: %s\n", edit.FilePath))

		// Get operations from metadata
		if ops, ok := edit.Metadata["operations"].([]*LineOperation); ok {
			for _, op := range ops {
				switch op.Type {
				case "insert":
					sb.WriteString(fmt.Sprintf("INSERT AT LINE %d:\n%s\n", op.LineNumber, op.Content))
				case "delete":
					if op.LineCount > 1 {
						sb.WriteString(fmt.Sprintf("DELETE LINE %d-%d\n", op.LineNumber, op.LineNumber+op.LineCount-1))
					} else {
						sb.WriteString(fmt.Sprintf("DELETE LINE %d\n", op.LineNumber))
					}
				case "replace":
					sb.WriteString(fmt.Sprintf("REPLACE LINE %d:\n%s\n", op.LineNumber, op.Content))
				}
			}
		}
	}

	return sb.String(), nil
}

// PromptTemplate returns the prompt template for this format
func (ef *EditorFormat) PromptTemplate() string {
	return `When editing files, use line-based operations:

File: <file_path>
INSERT AT LINE <num>:
<content to insert>

DELETE LINE <num>
DELETE LINE <start>-<end>

REPLACE LINE <num>:
<new content for line>

Alternative compact format:
L<num>: <content>

Examples:

File: src/main.go
INSERT AT LINE 5:
import "fmt"

DELETE LINE 10

REPLACE LINE 15:
func newFunction() {

File: config.yaml
L10: timeout: 60
L20:
L30: max_retries: 5

Important:
- Line numbers start at 1
- INSERT adds new line(s) at the specified position
- DELETE removes the specified line(s)
- REPLACE changes the content of the specified line
- Operations are applied in order
- Use L<num>: format for quick edits (empty content = delete)
`
}

// Validate validates that the format is correctly used
func (ef *EditorFormat) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Check for file marker
	if !strings.Contains(content, "File:") {
		return fmt.Errorf("content must contain 'File:' marker")
	}

	// Try to parse and see if we get any edits
	edits, err := ef.Parse(context.Background(), content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(edits) == 0 {
		return fmt.Errorf("no valid edits found")
	}

	// Validate operations
	for _, edit := range edits {
		if ops, ok := edit.Metadata["operations"].([]*LineOperation); ok {
			for _, op := range ops {
				if op.LineNumber < 1 {
					return fmt.Errorf("invalid line number: %d (must be >= 1)", op.LineNumber)
				}
				if op.Type == "delete" && op.LineCount < 1 {
					return fmt.Errorf("invalid line count: %d (must be >= 1)", op.LineCount)
				}
			}
		}
	}

	return nil
}
