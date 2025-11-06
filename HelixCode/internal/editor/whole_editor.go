package editor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WholeEditor replaces entire file content
type WholeEditor struct{}

// NewWholeEditor creates a new whole file editor
func NewWholeEditor() *WholeEditor {
	return &WholeEditor{}
}

// Apply replaces the entire file content
func (we *WholeEditor) Apply(edit Edit) error {
	newContent, ok := edit.Content.(string)
	if !ok {
		return fmt.Errorf("whole editor content must be a string")
	}

	// Ensure directory exists
	dir := filepath.Dir(edit.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Optional: Validate syntax if possible
	if err := we.validateSyntax(edit.FilePath, newContent); err != nil {
		return fmt.Errorf("syntax validation failed: %w", err)
	}

	// Write the new content
	file, err := os.Create(edit.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	if _, err := writer.WriteString(newContent); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush content: %w", err)
	}

	return nil
}

// validateSyntax performs basic syntax validation based on file extension
func (we *WholeEditor) validateSyntax(filePath string, content string) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".go":
		return we.validateGoSyntax(content)
	case ".json":
		return we.validateJSONSyntax(content)
	case ".yaml", ".yml":
		return we.validateYAMLSyntax(content)
	// Add more language validators as needed
	default:
		// No validation for unknown file types
		return nil
	}
}

// validateGoSyntax performs basic Go syntax validation
func (we *WholeEditor) validateGoSyntax(content string) error {
	// Basic checks for Go files
	lines := strings.Split(content, "\n")

	hasPackage := false
	bracketBalance := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for package declaration
		if strings.HasPrefix(trimmed, "package ") {
			hasPackage = true
		}

		// Check bracket balance
		bracketBalance += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
	}

	if !hasPackage && len(content) > 0 {
		return fmt.Errorf("missing package declaration")
	}

	if bracketBalance != 0 {
		return fmt.Errorf("unbalanced braces: difference of %d", bracketBalance)
	}

	return nil
}

// validateJSONSyntax performs basic JSON syntax validation
func (we *WholeEditor) validateJSONSyntax(content string) error {
	trimmed := strings.TrimSpace(content)

	if len(trimmed) == 0 {
		return fmt.Errorf("empty JSON content")
	}

	// JSON must start with { or [
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return fmt.Errorf("JSON must start with { or [")
	}

	// JSON must end with } or ]
	if trimmed[len(trimmed)-1] != '}' && trimmed[len(trimmed)-1] != ']' {
		return fmt.Errorf("JSON must end with } or ]")
	}

	// Check basic bracket/brace balance
	braceBalance := 0
	bracketBalance := 0
	inString := false
	escape := false

	for _, ch := range trimmed {
		if escape {
			escape = false
			continue
		}

		if ch == '\\' {
			escape = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if !inString {
			switch ch {
			case '{':
				braceBalance++
			case '}':
				braceBalance--
			case '[':
				bracketBalance++
			case ']':
				bracketBalance--
			}
		}
	}

	if braceBalance != 0 {
		return fmt.Errorf("unbalanced braces in JSON")
	}

	if bracketBalance != 0 {
		return fmt.Errorf("unbalanced brackets in JSON")
	}

	if inString {
		return fmt.Errorf("unclosed string in JSON")
	}

	return nil
}

// validateYAMLSyntax performs basic YAML syntax validation
func (we *WholeEditor) validateYAMLSyntax(content string) error {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for tabs (YAML doesn't allow tabs for indentation)
		if strings.Contains(line, "\t") {
			return fmt.Errorf("YAML does not allow tabs for indentation (line %d)", i+1)
		}

		// Check for basic key-value syntax
		if strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				if key == "" {
					return fmt.Errorf("empty key in YAML (line %d)", i+1)
				}
			}
		}
	}

	return nil
}

// GetFileStats returns information about the file change
func (we *WholeEditor) GetFileStats(oldContent, newContent string) FileStats {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	return FileStats{
		OldLineCount: len(oldLines),
		NewLineCount: len(newLines),
		LinesAdded:   len(newLines) - len(oldLines),
		LinesRemoved: len(oldLines) - len(newLines),
	}
}

// FileStats contains statistics about file changes
type FileStats struct {
	OldLineCount int
	NewLineCount int
	LinesAdded   int
	LinesRemoved int
}
