package formats

import "fmt"

// RegisterAllFormats registers all built-in edit formats
func RegisterAllFormats() (*FormatRegistry, error) {
	registry := NewFormatRegistry()

	// Register all 8 formats
	formats := []EditFormat{
		NewWholeFormat(),
		NewDiffFormat(),
		NewUDiffFormat(),
		NewSearchReplaceFormat(),
		NewEditorFormat(),
		NewArchitectFormat(),
		NewAskFormat(),
		NewLineNumberFormat(),
	}

	for _, format := range formats {
		if err := registry.Register(format); err != nil {
			return nil, fmt.Errorf("failed to register format '%s': %w", format.Type(), err)
		}
	}

	return registry, nil
}

// GetDefaultFormat returns the default format (whole-file)
func GetDefaultFormat() EditFormat {
	return NewWholeFormat()
}

// GetFormatByName returns a format by its type name
func GetFormatByName(name string) (EditFormat, error) {
	switch FormatType(name) {
	case FormatTypeWhole:
		return NewWholeFormat(), nil
	case FormatTypeDiff:
		return NewDiffFormat(), nil
	case FormatTypeUDiff:
		return NewUDiffFormat(), nil
	case FormatTypeSearchReplace:
		return NewSearchReplaceFormat(), nil
	case FormatTypeEditor:
		return NewEditorFormat(), nil
	case FormatTypeArchitect:
		return NewArchitectFormat(), nil
	case FormatTypeAsk:
		return NewAskFormat(), nil
	case FormatTypeLineNumber:
		return NewLineNumberFormat(), nil
	default:
		return nil, fmt.Errorf("unknown format type: %s", name)
	}
}

// GetAllFormatTypes returns all available format types
func GetAllFormatTypes() []FormatType {
	return []FormatType{
		FormatTypeWhole,
		FormatTypeDiff,
		FormatTypeUDiff,
		FormatTypeSearchReplace,
		FormatTypeEditor,
		FormatTypeArchitect,
		FormatTypeAsk,
		FormatTypeLineNumber,
	}
}

// GetFormatDescriptions returns descriptions of all formats
func GetFormatDescriptions() map[FormatType]string {
	return map[FormatType]string{
		FormatTypeWhole:         "Replace entire file content - simplest format, good for small files",
		FormatTypeDiff:          "Standard unified diff - precise changes with context lines",
		FormatTypeUDiff:         "Git-style unified diff - includes git metadata and headers",
		FormatTypeSearchReplace: "Regex search and replace - find exact text and replace",
		FormatTypeEditor:        "Line-based editing - insert, delete, replace specific lines",
		FormatTypeArchitect:     "High-level structural changes - file operations and refactoring",
		FormatTypeAsk:           "Question/confirmation mode - clarify before making changes",
		FormatTypeLineNumber:    "Direct line number editing - numbered line prefixes",
	}
}

// GetFormatRecommendations returns format recommendations based on use case
func GetFormatRecommendations() map[string]FormatType {
	return map[string]FormatType{
		"small_file":         FormatTypeWhole,
		"precise_changes":    FormatTypeDiff,
		"git_workflow":       FormatTypeUDiff,
		"simple_replacement": FormatTypeSearchReplace,
		"specific_lines":     FormatTypeEditor,
		"refactoring":        FormatTypeArchitect,
		"need_clarification": FormatTypeAsk,
		"line_by_line":       FormatTypeLineNumber,
	}
}
