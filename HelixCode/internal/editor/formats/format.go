package formats

import (
	"context"
	"fmt"
)

// FormatType represents the type of edit format
type FormatType string

const (
	FormatTypeWhole         FormatType = "whole"          // Replace entire file
	FormatTypeDiff          FormatType = "diff"           // Standard unified diff
	FormatTypeUDiff         FormatType = "udiff"          // Git-style unified diff
	FormatTypeSearchReplace FormatType = "search-replace" // Regex search/replace
	FormatTypeEditor        FormatType = "editor"         // Line-based editing
	FormatTypeArchitect     FormatType = "architect"      // High-level structural changes
	FormatTypeAsk           FormatType = "ask"            // Question/confirmation mode
	FormatTypeLineNumber    FormatType = "line-number"    // Direct line number editing
)

// EditFormat represents an interface for different edit formats
type EditFormat interface {
	// Type returns the format type
	Type() FormatType

	// Name returns the human-readable name
	Name() string

	// Description returns the format description
	Description() string

	// CanHandle checks if this format can handle the given content
	CanHandle(content string) bool

	// Parse parses the edit content and returns structured edits
	Parse(ctx context.Context, content string) ([]*FileEdit, error)

	// Format formats structured edits into this format's representation
	Format(edits []*FileEdit) (string, error)

	// PromptTemplate returns the prompt template for this format
	PromptTemplate() string

	// Validate validates that the format is correctly used
	Validate(content string) error
}

// FileEdit represents a single file edit operation
type FileEdit struct {
	FilePath      string                 // Path to the file being edited
	Operation     EditOperation          // Type of operation (create, update, delete)
	OldContent    string                 // Original content (for validation)
	NewContent    string                 // New content
	LineNumber    int                    // Starting line number (for line-based edits)
	LineCount     int                    // Number of lines affected
	SearchPattern string                 // Search pattern (for search/replace)
	ReplaceWith   string                 // Replacement text (for search/replace)
	Metadata      map[string]interface{} // Additional metadata
}

// EditOperation represents the type of edit operation
type EditOperation string

const (
	EditOperationCreate EditOperation = "create" // Create new file
	EditOperationUpdate EditOperation = "update" // Update existing file
	EditOperationDelete EditOperation = "delete" // Delete file
	EditOperationRename EditOperation = "rename" // Rename file
)

// ParseResult represents the result of parsing edit content
type ParseResult struct {
	Edits    []*FileEdit            // Parsed edits
	Warnings []string               // Non-fatal warnings
	Metadata map[string]interface{} // Additional metadata
}

// FormatRegistry manages available edit formats
type FormatRegistry struct {
	formats map[FormatType]EditFormat
}

// NewFormatRegistry creates a new format registry
func NewFormatRegistry() *FormatRegistry {
	return &FormatRegistry{
		formats: make(map[FormatType]EditFormat),
	}
}

// Register registers a new format
func (fr *FormatRegistry) Register(format EditFormat) error {
	if format == nil {
		return fmt.Errorf("format cannot be nil")
	}

	formatType := format.Type()
	if formatType == "" {
		return fmt.Errorf("format type cannot be empty")
	}

	if _, exists := fr.formats[formatType]; exists {
		return fmt.Errorf("format '%s' already registered", formatType)
	}

	fr.formats[formatType] = format
	return nil
}

// Get retrieves a format by type
func (fr *FormatRegistry) Get(formatType FormatType) (EditFormat, error) {
	format, exists := fr.formats[formatType]
	if !exists {
		return nil, fmt.Errorf("format '%s' not found", formatType)
	}
	return format, nil
}

// DetectFormat attempts to auto-detect the format from content
func (fr *FormatRegistry) DetectFormat(content string) (EditFormat, error) {
	for _, format := range fr.formats {
		if format.CanHandle(content) {
			return format, nil
		}
	}
	return nil, fmt.Errorf("no suitable format detected for content")
}

// ListFormats returns all registered formats
func (fr *FormatRegistry) ListFormats() []EditFormat {
	formats := make([]EditFormat, 0, len(fr.formats))
	for _, format := range fr.formats {
		formats = append(formats, format)
	}
	return formats
}

// ParseWithFormat parses content using a specific format
func (fr *FormatRegistry) ParseWithFormat(ctx context.Context, formatType FormatType, content string) ([]*FileEdit, error) {
	format, err := fr.Get(formatType)
	if err != nil {
		return nil, err
	}

	return format.Parse(ctx, content)
}

// ParseWithAutoDetect parses content with auto-detected format
func (fr *FormatRegistry) ParseWithAutoDetect(ctx context.Context, content string) ([]*FileEdit, EditFormat, error) {
	format, err := fr.DetectFormat(content)
	if err != nil {
		return nil, nil, err
	}

	edits, err := format.Parse(ctx, content)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse with detected format '%s': %w", format.Type(), err)
	}

	return edits, format, nil
}

// ValidateEdit validates a file edit
func ValidateEdit(edit *FileEdit) error {
	if edit == nil {
		return fmt.Errorf("edit cannot be nil")
	}

	if edit.FilePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if edit.Operation == "" {
		return fmt.Errorf("operation cannot be empty")
	}

	// Validate operation-specific fields
	switch edit.Operation {
	case EditOperationCreate:
		// NewContent can be empty for empty files, so no validation needed
	case EditOperationUpdate:
		// Update can have various forms (whole file, partial, etc.)
		// So we don't enforce strict requirements here
	case EditOperationDelete:
		// Delete doesn't require content
	case EditOperationRename:
		if edit.Metadata == nil || edit.Metadata["new_path"] == nil {
			return fmt.Errorf("new_path required in metadata for rename operation")
		}
	default:
		return fmt.Errorf("unknown operation: %s", edit.Operation)
	}

	return nil
}
