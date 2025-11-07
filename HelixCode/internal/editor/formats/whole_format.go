package formats

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// WholeFormat replaces entire file content
type WholeFormat struct{}

// NewWholeFormat creates a new whole-file format handler
func NewWholeFormat() *WholeFormat {
	return &WholeFormat{}
}

// Type returns the format type
func (wf *WholeFormat) Type() FormatType {
	return FormatTypeWhole
}

// Name returns the human-readable name
func (wf *WholeFormat) Name() string {
	return "Whole File"
}

// Description returns the format description
func (wf *WholeFormat) Description() string {
	return "Replace entire file content with new content"
}

// CanHandle checks if this format can handle the given content
func (wf *WholeFormat) CanHandle(content string) bool {
	// Must have code blocks AND file marker to be whole-file format
	hasCodeBlock := strings.Contains(content, "```")
	hasFileMarker := strings.Contains(content, "File:") || strings.Contains(content, "FILE:")

	// Check for explicit whole-file markers
	hasWholeFileMarker := strings.Contains(content, "// Entire file content") ||
		strings.Contains(content, "# Entire file content")

	// Require both code block and file marker, OR explicit whole-file marker
	return (hasCodeBlock && hasFileMarker) || hasWholeFileMarker
}

// Parse parses the edit content and returns structured edits
func (wf *WholeFormat) Parse(ctx context.Context, content string) ([]*FileEdit, error) {
	edits := make([]*FileEdit, 0)

	// Pattern: File: <path>\n```\n<content>\n```
	// or: <path>\n```\n<content>\n```
	codeBlockPattern := regexp.MustCompile(`(?ms)^(?:File|Path):\s*(.+?)\s*\n\x60{3}(?:\w+)?\n(.*?)\n\x60{3}`)
	matches := codeBlockPattern.FindAllStringSubmatch(content, -1)

	if len(matches) > 0 {
		for _, match := range matches {
			filePath := strings.TrimSpace(match[1])
			newContent := match[2]

			edit := &FileEdit{
				FilePath:   filePath,
				Operation:  EditOperationUpdate,
				NewContent: newContent,
				Metadata:   map[string]interface{}{"format": "whole"},
			}

			if err := ValidateEdit(edit); err != nil {
				return nil, fmt.Errorf("invalid edit for %s: %w", filePath, err)
			}

			edits = append(edits, edit)
		}
		return edits, nil
	}

	// Alternative pattern: Just a code block with filename in first line
	// ```filename.go
	// <content>
	// ```
	altPattern := regexp.MustCompile(`(?s)\x60{3}(\S+)\n(.*?)\n\x60{3}`)
	altMatches := altPattern.FindAllStringSubmatch(content, -1)

	if len(altMatches) > 0 {
		for _, match := range altMatches {
			filePath := strings.TrimSpace(match[1])
			newContent := match[2]

			// Skip if filename doesn't look like a path (e.g., just language name)
			if !strings.Contains(filePath, ".") && !strings.Contains(filePath, "/") {
				continue
			}

			edit := &FileEdit{
				FilePath:   filePath,
				Operation:  EditOperationUpdate,
				NewContent: newContent,
				Metadata:   map[string]interface{}{"format": "whole"},
			}

			if err := ValidateEdit(edit); err != nil {
				return nil, fmt.Errorf("invalid edit for %s: %w", filePath, err)
			}

			edits = append(edits, edit)
		}
		return edits, nil
	}

	// If no matches found, return error
	if len(edits) == 0 {
		return nil, fmt.Errorf("no valid whole-file edits found in content")
	}

	return edits, nil
}

// Format formats structured edits into this format's representation
func (wf *WholeFormat) Format(edits []*FileEdit) (string, error) {
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits to format")
	}

	var sb strings.Builder

	for i, edit := range edits {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		// Determine file extension for syntax highlighting hint
		ext := ""
		if dotIdx := strings.LastIndex(edit.FilePath, "."); dotIdx != -1 {
			ext = edit.FilePath[dotIdx+1:]
		}

		sb.WriteString(fmt.Sprintf("File: %s\n", edit.FilePath))
		sb.WriteString("```")
		if ext != "" {
			sb.WriteString(ext)
		}
		sb.WriteString("\n")
		sb.WriteString(edit.NewContent)
		if !strings.HasSuffix(edit.NewContent, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```")
	}

	return sb.String(), nil
}

// PromptTemplate returns the prompt template for this format
func (wf *WholeFormat) PromptTemplate() string {
	return `When editing files, provide the complete new file content in this format:

File: <file_path>
` + "```" + `<language>
<complete file content>
` + "```" + `

Example:

File: src/main.go
` + "```" + `go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

Important:
- Include the ENTIRE file content, not just changes
- Use proper language identifier for syntax highlighting
- Ensure proper indentation and formatting
- Include all imports, functions, and content
`
}

// Validate validates that the format is correctly used
func (wf *WholeFormat) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Check for required markers
	if !strings.Contains(content, "```") {
		return fmt.Errorf("content must contain code blocks (```)")
	}

	// Try to parse and see if we get any edits
	edits, err := wf.Parse(context.Background(), content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(edits) == 0 {
		return fmt.Errorf("no valid edits found")
	}

	return nil
}
