package multiedit

import (
	"context"
	"fmt"
)

// Editor is the main interface for multi-file editing
type Editor interface {
	// BeginEdit starts a new edit transaction
	BeginEdit(ctx context.Context, opts EditOptions) (*EditTransaction, error)

	// AddEdit adds a file edit to the transaction
	AddEdit(ctx context.Context, tx *EditTransaction, edit *FileEdit) error

	// Preview generates a preview of changes
	Preview(ctx context.Context, tx *EditTransaction) (*PreviewResult, error)

	// Commit applies all changes atomically
	Commit(ctx context.Context, tx *EditTransaction) error

	// Rollback reverts all changes
	Rollback(ctx context.Context, tx *EditTransaction) error

	// GetTransaction retrieves a transaction by ID
	GetTransaction(ctx context.Context, txID string) (*EditTransaction, error)
}

// BackupProvider handles file backups
type BackupProvider interface {
	Backup(ctx context.Context, filePath string) (string, error)
	Restore(ctx context.Context, backupPath, targetPath string) error
	Cleanup(ctx context.Context) error
}

// DiffProvider generates and applies diffs
type DiffProvider interface {
	GenerateDiff(old, new []byte, path string) (*Diff, error)
	ApplyDiff(diff *Diff) ([]byte, error)
	ParseDiff(diffText string) (*Diff, error)
}

// ConflictDetector detects conflicts
type ConflictDetector interface {
	DetectConflicts(ctx context.Context, tx *EditTransaction) ([]*Conflict, error)
	Resolve(ctx context.Context, conflict *Conflict, strategy ConflictStrategy) error
}

// PreviewEngine generates previews of changes
type PreviewEngine struct {
	diffManager  *DiffManager
	formatter    *PreviewFormatter
	contextLines int
}

// NewPreviewEngine creates a new preview engine
func NewPreviewEngine(diffManager *DiffManager, contextLines int) *PreviewEngine {
	return &PreviewEngine{
		diffManager:  diffManager,
		formatter:    NewPreviewFormatter(FormatPlain),
		contextLines: contextLines,
	}
}

// Preview generates a preview of all changes
func (pe *PreviewEngine) Preview(ctx context.Context, tx *EditTransaction) (*PreviewResult, error) {
	result := &PreviewResult{
		TransactionID: tx.ID,
		Files:         make([]*FilePreview, 0, len(tx.Files)),
		Summary:       &PreviewSummary{},
	}

	for _, edit := range tx.Files {
		preview, err := pe.previewFile(ctx, edit)
		if err != nil {
			return nil, fmt.Errorf("failed to preview file %s: %w", edit.FilePath, err)
		}
		result.Files = append(result.Files, preview)

		// Update summary
		pe.updateSummary(result.Summary, preview)
	}

	return result, nil
}

// previewFile generates preview for a single file
func (pe *PreviewEngine) previewFile(ctx context.Context, edit *FileEdit) (*FilePreview, error) {
	preview := &FilePreview{
		FilePath:  edit.FilePath,
		Operation: edit.Operation,
		Status:    StatusOK,
	}

	// Generate diff
	if edit.Operation == OpUpdate {
		diff, err := pe.diffManager.GenerateDiff(edit.OldContent, edit.NewContent, edit.FilePath)
		if err != nil {
			preview.Status = StatusError
			return preview, fmt.Errorf("failed to generate diff: %w", err)
		}
		preview.Diff = diff
		preview.Stats = FileStats{
			LinesAdded:   diff.Stats.LinesAdded,
			LinesDeleted: diff.Stats.LinesDeleted,
			LinesChanged: diff.Stats.LinesAdded + diff.Stats.LinesDeleted,
			SizeChange:   int64(len(edit.NewContent)) - int64(len(edit.OldContent)),
		}
	} else if edit.Operation == OpCreate {
		preview.Stats = FileStats{
			LinesAdded: len(splitLines(string(edit.NewContent))),
			SizeChange: int64(len(edit.NewContent)),
		}
	} else if edit.Operation == OpDelete {
		preview.Stats = FileStats{
			LinesDeleted: len(splitLines(string(edit.OldContent))),
			SizeChange:   -int64(len(edit.OldContent)),
		}
	}

	return preview, nil
}

// updateSummary updates the preview summary
func (pe *PreviewEngine) updateSummary(summary *PreviewSummary, preview *FilePreview) {
	summary.TotalFiles++

	switch preview.Operation {
	case OpCreate:
		summary.FilesCreated++
	case OpUpdate:
		summary.FilesModified++
	case OpDelete:
		summary.FilesDeleted++
	}

	summary.TotalLinesAdded += preview.Stats.LinesAdded
	summary.TotalLinesDeleted += preview.Stats.LinesDeleted
}

// PreviewResult contains preview information
type PreviewResult struct {
	TransactionID string
	Files         []*FilePreview
	Summary       *PreviewSummary
	Conflicts     []*Conflict
}

// FilePreview contains preview for a single file
type FilePreview struct {
	FilePath  string
	Operation EditOperation
	Diff      *Diff
	Stats     FileStats
	Status    PreviewStatus
}

// FileStats contains file statistics
type FileStats struct {
	LinesAdded   int
	LinesDeleted int
	LinesChanged int
	SizeChange   int64
}

// PreviewStatus indicates preview status
type PreviewStatus int

const (
	StatusOK PreviewStatus = iota
	StatusConflict
	StatusError
)

// String returns the string representation of the status
func (ps PreviewStatus) String() string {
	switch ps {
	case StatusOK:
		return "ok"
	case StatusConflict:
		return "conflict"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// PreviewSummary summarizes all changes
type PreviewSummary struct {
	TotalFiles        int
	FilesCreated      int
	FilesModified     int
	FilesDeleted      int
	TotalLinesAdded   int
	TotalLinesDeleted int
	HasConflicts      bool
}

// PreviewFormatter formats preview output
type PreviewFormatter struct {
	format OutputFormat
}

// OutputFormat specifies the output format
type OutputFormat int

const (
	FormatPlain OutputFormat = iota
	FormatMarkdown
	FormatJSON
	FormatHTML
)

// String returns the string representation of the format
func (of OutputFormat) String() string {
	switch of {
	case FormatPlain:
		return "plain"
	case FormatMarkdown:
		return "markdown"
	case FormatJSON:
		return "json"
	case FormatHTML:
		return "html"
	default:
		return "unknown"
	}
}

// NewPreviewFormatter creates a new preview formatter
func NewPreviewFormatter(format OutputFormat) *PreviewFormatter {
	return &PreviewFormatter{
		format: format,
	}
}

// Format formats a preview result
func (pf *PreviewFormatter) Format(result *PreviewResult) (string, error) {
	switch pf.format {
	case FormatPlain:
		return pf.formatPlain(result)
	case FormatMarkdown:
		return pf.formatMarkdown(result)
	case FormatJSON:
		return pf.formatJSON(result)
	case FormatHTML:
		return pf.formatHTML(result)
	default:
		return "", fmt.Errorf("unsupported format: %v", pf.format)
	}
}

// formatPlain formats as plain text
func (pf *PreviewFormatter) formatPlain(result *PreviewResult) (string, error) {
	var output string

	output += fmt.Sprintf("Transaction: %s\n", result.TransactionID)
	output += fmt.Sprintf("Summary: %d files (%d created, %d modified, %d deleted)\n",
		result.Summary.TotalFiles,
		result.Summary.FilesCreated,
		result.Summary.FilesModified,
		result.Summary.FilesDeleted)
	output += fmt.Sprintf("Changes: +%d -%d lines\n",
		result.Summary.TotalLinesAdded,
		result.Summary.TotalLinesDeleted)

	if result.Summary.HasConflicts {
		output += fmt.Sprintf("\nConflicts: %d\n", len(result.Conflicts))
		for _, conflict := range result.Conflicts {
			output += fmt.Sprintf("  - %s: %s\n", conflict.FilePath, conflict.Description)
		}
	}

	output += "\nFiles:\n"
	for _, file := range result.Files {
		output += fmt.Sprintf("\n%s (%s)\n", file.FilePath, file.Operation)
		if file.Diff != nil {
			output += file.Diff.Unified
		}
	}

	return output, nil
}

// formatMarkdown formats as markdown
func (pf *PreviewFormatter) formatMarkdown(result *PreviewResult) (string, error) {
	var output string

	output += fmt.Sprintf("# Transaction Preview: %s\n\n", result.TransactionID)
	output += "## Summary\n\n"
	output += fmt.Sprintf("- **Total Files**: %d\n", result.Summary.TotalFiles)
	output += fmt.Sprintf("- **Created**: %d\n", result.Summary.FilesCreated)
	output += fmt.Sprintf("- **Modified**: %d\n", result.Summary.FilesModified)
	output += fmt.Sprintf("- **Deleted**: %d\n", result.Summary.FilesDeleted)
	output += fmt.Sprintf("- **Lines Added**: +%d\n", result.Summary.TotalLinesAdded)
	output += fmt.Sprintf("- **Lines Deleted**: -%d\n\n", result.Summary.TotalLinesDeleted)

	if result.Summary.HasConflicts {
		output += "## Conflicts\n\n"
		for _, conflict := range result.Conflicts {
			output += fmt.Sprintf("- **%s**: %s\n", conflict.FilePath, conflict.Description)
		}
		output += "\n"
	}

	output += "## Files\n\n"
	for _, file := range result.Files {
		output += fmt.Sprintf("### %s (%s)\n\n", file.FilePath, file.Operation)
		if file.Diff != nil {
			output += "```diff\n"
			output += file.Diff.Unified
			output += "```\n\n"
		}
	}

	return output, nil
}

// formatJSON formats as JSON
func (pf *PreviewFormatter) formatJSON(result *PreviewResult) (string, error) {
	// This would use encoding/json in a real implementation
	return fmt.Sprintf(`{"transaction_id":"%s","summary":{"total_files":%d}}`,
		result.TransactionID, result.Summary.TotalFiles), nil
}

// formatHTML formats as HTML
func (pf *PreviewFormatter) formatHTML(result *PreviewResult) (string, error) {
	var output string

	output += "<html><body>\n"
	output += fmt.Sprintf("<h1>Transaction Preview: %s</h1>\n", result.TransactionID)
	output += "<h2>Summary</h2>\n"
	output += "<ul>\n"
	output += fmt.Sprintf("<li>Total Files: %d</li>\n", result.Summary.TotalFiles)
	output += fmt.Sprintf("<li>Created: %d</li>\n", result.Summary.FilesCreated)
	output += fmt.Sprintf("<li>Modified: %d</li>\n", result.Summary.FilesModified)
	output += fmt.Sprintf("<li>Deleted: %d</li>\n", result.Summary.FilesDeleted)
	output += "</ul>\n"

	if result.Summary.HasConflicts {
		output += "<h2>Conflicts</h2>\n"
		output += "<ul>\n"
		for _, conflict := range result.Conflicts {
			output += fmt.Sprintf("<li>%s: %s</li>\n", conflict.FilePath, conflict.Description)
		}
		output += "</ul>\n"
	}

	output += "<h2>Files</h2>\n"
	for _, file := range result.Files {
		output += fmt.Sprintf("<h3>%s (%s)</h3>\n", file.FilePath, file.Operation)
		if file.Diff != nil {
			output += "<pre><code>\n"
			output += file.Diff.Unified
			output += "</code></pre>\n"
		}
	}

	output += "</body></html>\n"

	return output, nil
}
