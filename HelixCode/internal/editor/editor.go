package editor

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// EditFormat represents the type of edit format to use
type EditFormat string

const (
	// EditFormatDiff uses Unix unified diff format
	EditFormatDiff EditFormat = "diff"
	// EditFormatWhole replaces entire file content
	EditFormatWhole EditFormat = "whole"
	// EditFormatSearchReplace performs search and replace operations
	EditFormatSearchReplace EditFormat = "search_replace"
	// EditFormatLines edits specific line ranges
	EditFormatLines EditFormat = "lines"
)

// EditValidator validates edits before applying them
type EditValidator interface {
	Validate(edit Edit) error
}

// EditApplier applies edits to files
type EditApplier interface {
	Apply(edit Edit) error
}

// Edit represents a code edit operation
type Edit struct {
	FilePath string
	Format   EditFormat
	Content  interface{} // format-specific content
	Backup   bool        // whether to create backup
}

// EditResult contains the result of an edit operation
type EditResult struct {
	FilePath    string
	Success     bool
	Error       error
	LinesChanged int
	BackupPath  string
}

// CodeEditor is the main editor that coordinates different edit formats
type CodeEditor struct {
	format    EditFormat
	validator EditValidator
	applier   EditApplier
	mu        sync.RWMutex
	editors   map[EditFormat]EditApplier
}

// NewCodeEditor creates a new code editor with the specified default format
func NewCodeEditor(format EditFormat) (*CodeEditor, error) {
	ce := &CodeEditor{
		format:  format,
		editors: make(map[EditFormat]EditApplier),
	}

	// Register all format editors
	ce.editors[EditFormatDiff] = NewDiffEditor()
	ce.editors[EditFormatWhole] = NewWholeEditor()
	ce.editors[EditFormatSearchReplace] = NewSearchReplaceEditor()
	ce.editors[EditFormatLines] = NewLineEditor()

	// Set the default applier
	applier, ok := ce.editors[format]
	if !ok {
		return nil, fmt.Errorf("unsupported edit format: %s", format)
	}
	ce.applier = applier

	// Use default validator
	ce.validator = NewDefaultValidator()

	return ce, nil
}

// ApplyEdit applies an edit operation to a file
func (ce *CodeEditor) ApplyEdit(edit Edit) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Validate the edit first
	if err := ce.validator.Validate(edit); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Get the appropriate applier for this format
	applier, ok := ce.editors[edit.Format]
	if !ok {
		return fmt.Errorf("unsupported edit format: %s", edit.Format)
	}

	// Create backup if requested
	if edit.Backup {
		if err := ce.createBackup(edit.FilePath); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
	}

	// Apply the edit
	if err := applier.Apply(edit); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	return nil
}

// ValidateEdit validates an edit without applying it
func (ce *CodeEditor) ValidateEdit(edit Edit) error {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	return ce.validator.Validate(edit)
}

// SetFormat changes the default edit format
func (ce *CodeEditor) SetFormat(format EditFormat) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	applier, ok := ce.editors[format]
	if !ok {
		return fmt.Errorf("unsupported edit format: %s", format)
	}

	ce.format = format
	ce.applier = applier

	return nil
}

// GetFormat returns the current default edit format
func (ce *CodeEditor) GetFormat() EditFormat {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.format
}

// SetValidator sets a custom validator
func (ce *CodeEditor) SetValidator(validator EditValidator) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.validator = validator
}

// createBackup creates a backup of the file before editing
func (ce *CodeEditor) createBackup(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No backup needed for new files
	}

	// Create backup file path
	backupPath := filePath + ".bak"

	// Open source file
	src, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Create backup file
	dst, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}

	return nil
}

// DefaultValidator is the default edit validator
type DefaultValidator struct{}

// NewDefaultValidator creates a new default validator
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

// Validate performs basic validation on an edit
func (dv *DefaultValidator) Validate(edit Edit) error {
	// Check file path is provided
	if edit.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	// Check format is valid
	switch edit.Format {
	case EditFormatDiff, EditFormatWhole, EditFormatSearchReplace, EditFormatLines:
		// Valid format
	default:
		return fmt.Errorf("invalid edit format: %s", edit.Format)
	}

	// Check content is provided
	if edit.Content == nil {
		return fmt.Errorf("edit content is required")
	}

	// Format-specific validation
	switch edit.Format {
	case EditFormatDiff:
		if _, ok := edit.Content.(string); !ok {
			return fmt.Errorf("diff format requires string content")
		}
	case EditFormatWhole:
		if _, ok := edit.Content.(string); !ok {
			return fmt.Errorf("whole format requires string content")
		}
	case EditFormatSearchReplace:
		if _, ok := edit.Content.([]SearchReplace); !ok {
			return fmt.Errorf("search_replace format requires []SearchReplace content")
		}
	case EditFormatLines:
		if _, ok := edit.Content.([]LineEdit); !ok {
			return fmt.Errorf("lines format requires []LineEdit content")
		}
	}

	return nil
}
