package formats

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// ArchitectFormat handles high-level structural changes
type ArchitectFormat struct{}

// NewArchitectFormat creates a new architect format handler
func NewArchitectFormat() *ArchitectFormat {
	return &ArchitectFormat{}
}

// Type returns the format type
func (af *ArchitectFormat) Type() FormatType {
	return FormatTypeArchitect
}

// Name returns the human-readable name
func (af *ArchitectFormat) Name() string {
	return "Architect Mode"
}

// Description returns the format description
func (af *ArchitectFormat) Description() string {
	return "High-level structural changes with architectural directives"
}

// CanHandle checks if this format can handle the given content
func (af *ArchitectFormat) CanHandle(content string) bool {
	// Look for architect-mode markers
	markers := []string{
		"CREATE FILE",
		"MODIFY FILE",
		"DELETE FILE",
		"RENAME FILE",
		"MOVE FILE",
		"RESTRUCTURE",
		"REFACTOR",
		"create file",
		"modify file",
		"delete file",
	}

	markerCount := 0
	for _, marker := range markers {
		if strings.Contains(strings.ToLower(content), strings.ToLower(marker)) {
			markerCount++
		}
	}

	// Require at least 1 marker to consider it architect mode
	return markerCount >= 1
}

// Parse parses the edit content and returns structured edits
func (af *ArchitectFormat) Parse(ctx context.Context, content string) ([]*FileEdit, error) {
	edits := make([]*FileEdit, 0)

	// Parse different operation types
	edits = append(edits, af.parseCreateFile(content)...)
	edits = append(edits, af.parseModifyFile(content)...)
	edits = append(edits, af.parseDeleteFile(content)...)
	edits = append(edits, af.parseRenameFile(content)...)

	if len(edits) == 0 {
		return nil, fmt.Errorf("no valid architect operations found")
	}

	// Validate all edits
	for _, edit := range edits {
		if err := ValidateEdit(edit); err != nil {
			return nil, fmt.Errorf("invalid edit for %s: %w", edit.FilePath, err)
		}
	}

	return edits, nil
}

// parseCreateFile parses CREATE FILE operations
func (af *ArchitectFormat) parseCreateFile(content string) []*FileEdit {
	edits := make([]*FileEdit, 0)

	// Pattern: CREATE FILE <path>[\n<content or description>]
	// Content is optional - may or may not have newline and content after filepath
	createPattern := regexp.MustCompile(`(?mis)CREATE FILE[:\s]+([^\n]+)(?:\n(.*?))?(?:(?:\n(?:CREATE|MODIFY|DELETE|RENAME))|\z)`)
	matches := createPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		filePath := strings.TrimSpace(match[1])
		fileContent := strings.TrimSpace(match[2])

		// Check if content is in code block
		codeBlockPattern := regexp.MustCompile(`(?s)\x60{3}(?:\w+)?\n(.*?)\n\x60{3}`)
		if codeMatch := codeBlockPattern.FindStringSubmatch(fileContent); codeMatch != nil {
			fileContent = codeMatch[1]
		}

		edits = append(edits, &FileEdit{
			FilePath:   filePath,
			Operation:  EditOperationCreate,
			NewContent: fileContent,
			Metadata: map[string]interface{}{
				"format": "architect",
				"action": "create",
			},
		})
	}

	return edits
}

// parseModifyFile parses MODIFY FILE operations
func (af *ArchitectFormat) parseModifyFile(content string) []*FileEdit {
	edits := make([]*FileEdit, 0)

	// Pattern: MODIFY FILE <path>\n<changes description>
	modifyPattern := regexp.MustCompile(`(?mis)MODIFY FILE[:\s]+(.+?)\s*\nChanges:\s*\n(.*?)(?:\n(?:CREATE|MODIFY|DELETE|RENAME|$))`)
	matches := modifyPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		filePath := strings.TrimSpace(match[1])
		changes := strings.TrimSpace(match[2])

		edits = append(edits, &FileEdit{
			FilePath:  filePath,
			Operation: EditOperationUpdate,
			Metadata: map[string]interface{}{
				"format":      "architect",
				"action":      "modify",
				"description": changes,
			},
		})
	}

	return edits
}

// parseDeleteFile parses DELETE FILE operations
func (af *ArchitectFormat) parseDeleteFile(content string) []*FileEdit {
	edits := make([]*FileEdit, 0)

	// Pattern: DELETE FILE <path>
	deletePattern := regexp.MustCompile(`(?mi)DELETE FILE[:\s]+(.+?)(?:\s*\n|$)`)
	matches := deletePattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		filePath := strings.TrimSpace(match[1])

		edits = append(edits, &FileEdit{
			FilePath:  filePath,
			Operation: EditOperationDelete,
			Metadata: map[string]interface{}{
				"format": "architect",
				"action": "delete",
			},
		})
	}

	return edits
}

// parseRenameFile parses RENAME FILE operations
func (af *ArchitectFormat) parseRenameFile(content string) []*FileEdit {
	edits := make([]*FileEdit, 0)

	// Pattern: RENAME FILE <old_path> TO <new_path>
	renamePattern := regexp.MustCompile(`(?mi)RENAME FILE[:\s]+(.+?)\s+TO\s+(.+?)(?:\s*\n|$)`)
	matches := renamePattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		oldPath := strings.TrimSpace(match[1])
		newPath := strings.TrimSpace(match[2])

		edits = append(edits, &FileEdit{
			FilePath:  newPath,
			Operation: EditOperationRename,
			Metadata: map[string]interface{}{
				"format":   "architect",
				"action":   "rename",
				"old_path": oldPath,
				"new_path": newPath,
			},
		})
	}

	return edits
}

// Format formats structured edits into this format's representation
func (af *ArchitectFormat) Format(edits []*FileEdit) (string, error) {
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits to format")
	}

	var sb strings.Builder

	for i, edit := range edits {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		switch edit.Operation {
		case EditOperationCreate:
			sb.WriteString(fmt.Sprintf("CREATE FILE: %s\n", edit.FilePath))
			if edit.NewContent != "" {
				ext := ""
				if dotIdx := strings.LastIndex(edit.FilePath, "."); dotIdx != -1 {
					ext = edit.FilePath[dotIdx+1:]
				}
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

		case EditOperationUpdate:
			sb.WriteString(fmt.Sprintf("MODIFY FILE: %s\n", edit.FilePath))
			if desc, ok := edit.Metadata["description"].(string); ok {
				sb.WriteString("Changes:\n")
				sb.WriteString(desc)
			} else if edit.NewContent != "" {
				sb.WriteString("Changes:\n")
				sb.WriteString(edit.NewContent)
			}

		case EditOperationDelete:
			sb.WriteString(fmt.Sprintf("DELETE FILE: %s\n", edit.FilePath))

		case EditOperationRename:
			oldPath := edit.FilePath
			if op, ok := edit.Metadata["old_path"].(string); ok {
				oldPath = op
			}
			sb.WriteString(fmt.Sprintf("RENAME FILE: %s TO %s\n", oldPath, edit.FilePath))
		}
	}

	return sb.String(), nil
}

// PromptTemplate returns the prompt template for this format
func (af *ArchitectFormat) PromptTemplate() string {
	return `When making structural changes, use architect-mode directives:

CREATE FILE: <path>
` + "```" + `<language>
<file content>
` + "```" + `

MODIFY FILE: <path>
Changes:
- <description of changes>
- <what to add/remove/update>

DELETE FILE: <path>

RENAME FILE: <old_path> TO <new_path>

MOVE FILE: <old_path> TO <new_directory>

Examples:

CREATE FILE: src/models/user.go
` + "```" + `go
package models

type User struct {
    ID       int
    Username string
    Email    string
}
` + "```" + `

MODIFY FILE: src/main.go
Changes:
- Add user authentication middleware
- Update route handlers to use new User model
- Add error handling for database operations

DELETE FILE: src/legacy/old_module.go

RENAME FILE: config.yml TO config.yaml

MOVE FILE: utils/helper.go TO src/utils/helper.go

Important:
- Use high-level descriptions for modifications
- Be specific about structural changes
- Group related changes together
- Include reasons for major refactoring
- Use code blocks for new file content
`
}

// Validate validates that the format is correctly used
func (af *ArchitectFormat) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Try to parse and see if we get any edits
	edits, err := af.Parse(context.Background(), content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(edits) == 0 {
		return fmt.Errorf("no valid architect operations found")
	}

	return nil
}
