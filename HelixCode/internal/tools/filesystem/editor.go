package filesystem

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// FileEditor provides methods for editing files
type FileEditor interface {
	// Edit applies a series of edit operations to a file
	Edit(ctx context.Context, path string, ops []EditOperation) (*EditResult, error)

	// Replace replaces all occurrences of a pattern
	Replace(ctx context.Context, path string, pattern, replacement string, regex bool) (*EditResult, error)

	// InsertAt inserts content at a specific line
	InsertAt(ctx context.Context, path string, line int, content string) (*EditResult, error)

	// DeleteLines deletes specific lines
	DeleteLines(ctx context.Context, path string, start, end int) (*EditResult, error)

	// Diff generates a diff between current and proposed changes
	Diff(ctx context.Context, path string, ops []EditOperation) (string, error)
}

// EditOperation represents a single edit operation
type EditOperation struct {
	Type        EditType
	StartLine   int
	EndLine     int
	StartCol    int
	EndCol      int
	Content     string
	Pattern     string
	Replacement string
	IsRegex     bool
}

// EditType represents the type of edit operation
type EditType int

const (
	EditInsert EditType = iota
	EditDelete
	EditReplace
	EditReplacePattern
)

func (e EditType) String() string {
	switch e {
	case EditInsert:
		return "Insert"
	case EditDelete:
		return "Delete"
	case EditReplace:
		return "Replace"
	case EditReplacePattern:
		return "ReplacePattern"
	default:
		return "Unknown"
	}
}

// EditResult contains the result of an edit operation
type EditResult struct {
	Path            string
	OriginalContent []byte
	NewContent      []byte
	Operations      []EditOperation
	LinesChanged    int
	BytesChanged    int64
	Diff            string
	Success         bool
	Error           error
}

// fileEditor implements FileEditor
type fileEditor struct {
	fs *FileSystemTools
}

// Edit applies a series of edit operations to a file
func (e *fileEditor) Edit(ctx context.Context, path string, ops []EditOperation) (*EditResult, error) {
	// Validate path
	validationResult, err := e.fs.pathValidator.Validate(path)
	if err != nil {
		return nil, err
	}
	normalizedPath := validationResult.NormalizedPath

	// Check permissions
	if err := e.fs.permissionChecker.CheckPermission(normalizedPath, OpWrite); err != nil {
		return nil, err
	}

	// Acquire lock
	lock, err := e.fs.lockManager.Acquire(ctx, normalizedPath, "editor")
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer e.fs.lockManager.Release(lock)

	// Read original content
	originalContent, err := os.ReadFile(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Apply operations
	newContent := originalContent
	linesChanged := 0

	for _, op := range ops {
		newContent, err = e.applyOperation(newContent, op)
		if err != nil {
			return &EditResult{
				Path:            normalizedPath,
				OriginalContent: originalContent,
				NewContent:      originalContent,
				Operations:      ops,
				Success:         false,
				Error:           err,
			}, err
		}
		linesChanged++
	}

	// Write back to file
	opts := DefaultWriteOptions()
	opts.Backup = true
	if err := e.fs.writer.WriteWithOptions(ctx, normalizedPath, newContent, opts); err != nil {
		return &EditResult{
			Path:            normalizedPath,
			OriginalContent: originalContent,
			NewContent:      newContent,
			Operations:      ops,
			Success:         false,
			Error:           err,
		}, err
	}

	// Generate diff
	diff := e.generateDiff(originalContent, newContent)

	return &EditResult{
		Path:            normalizedPath,
		OriginalContent: originalContent,
		NewContent:      newContent,
		Operations:      ops,
		LinesChanged:    linesChanged,
		BytesChanged:    int64(len(newContent) - len(originalContent)),
		Diff:            diff,
		Success:         true,
	}, nil
}

// Replace replaces all occurrences of a pattern
func (e *fileEditor) Replace(ctx context.Context, path string, pattern, replacement string, regex bool) (*EditResult, error) {
	op := EditOperation{
		Type:        EditReplacePattern,
		Pattern:     pattern,
		Replacement: replacement,
		IsRegex:     regex,
	}
	return e.Edit(ctx, path, []EditOperation{op})
}

// InsertAt inserts content at a specific line
func (e *fileEditor) InsertAt(ctx context.Context, path string, line int, content string) (*EditResult, error) {
	op := EditOperation{
		Type:      EditInsert,
		StartLine: line,
		Content:   content,
	}
	return e.Edit(ctx, path, []EditOperation{op})
}

// DeleteLines deletes specific lines
func (e *fileEditor) DeleteLines(ctx context.Context, path string, start, end int) (*EditResult, error) {
	op := EditOperation{
		Type:      EditDelete,
		StartLine: start,
		EndLine:   end,
	}
	return e.Edit(ctx, path, []EditOperation{op})
}

// Diff generates a diff between current and proposed changes
func (e *fileEditor) Diff(ctx context.Context, path string, ops []EditOperation) (string, error) {
	// Validate path
	validationResult, err := e.fs.pathValidator.Validate(path)
	if err != nil {
		return "", err
	}
	normalizedPath := validationResult.NormalizedPath

	// Read original content
	originalContent, err := os.ReadFile(normalizedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Apply operations to get new content
	newContent := originalContent
	for _, op := range ops {
		newContent, err = e.applyOperation(newContent, op)
		if err != nil {
			return "", err
		}
	}

	// Generate diff
	return e.generateDiff(originalContent, newContent), nil
}

// applyOperation applies a single edit operation to content
func (e *fileEditor) applyOperation(content []byte, op EditOperation) ([]byte, error) {
	switch op.Type {
	case EditInsert:
		return e.applyInsert(content, op)
	case EditDelete:
		return e.applyDelete(content, op)
	case EditReplace:
		return e.applyReplace(content, op)
	case EditReplacePattern:
		return e.applyReplacePattern(content, op)
	default:
		return nil, fmt.Errorf("unknown edit operation type: %v", op.Type)
	}
}

// applyInsert inserts content at a specific line
func (e *fileEditor) applyInsert(content []byte, op EditOperation) ([]byte, error) {
	lines := splitLines(content)

	// Validate line number
	if op.StartLine < 1 || op.StartLine > len(lines)+1 {
		return nil, fmt.Errorf("invalid line number: %d (file has %d lines)", op.StartLine, len(lines))
	}

	// Insert content
	insertIndex := op.StartLine - 1
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIndex]...)
	newLines = append(newLines, op.Content)
	newLines = append(newLines, lines[insertIndex:]...)

	return []byte(strings.Join(newLines, "\n") + "\n"), nil
}

// applyDelete deletes specific lines
func (e *fileEditor) applyDelete(content []byte, op EditOperation) ([]byte, error) {
	lines := splitLines(content)

	// Validate line numbers
	if op.StartLine < 1 || op.StartLine > len(lines) {
		return nil, fmt.Errorf("invalid start line: %d (file has %d lines)", op.StartLine, len(lines))
	}
	if op.EndLine < op.StartLine || op.EndLine > len(lines) {
		return nil, fmt.Errorf("invalid end line: %d (must be >= %d and <= %d)", op.EndLine, op.StartLine, len(lines))
	}

	// Delete lines
	newLines := make([]string, 0, len(lines)-(op.EndLine-op.StartLine+1))
	newLines = append(newLines, lines[:op.StartLine-1]...)
	newLines = append(newLines, lines[op.EndLine:]...)

	if len(newLines) == 0 {
		return []byte{}, nil
	}
	return []byte(strings.Join(newLines, "\n") + "\n"), nil
}

// applyReplace replaces specific lines with new content
func (e *fileEditor) applyReplace(content []byte, op EditOperation) ([]byte, error) {
	lines := splitLines(content)

	// Validate line numbers
	if op.StartLine < 1 || op.StartLine > len(lines) {
		return nil, fmt.Errorf("invalid start line: %d (file has %d lines)", op.StartLine, len(lines))
	}
	if op.EndLine < op.StartLine || op.EndLine > len(lines) {
		return nil, fmt.Errorf("invalid end line: %d (must be >= %d and <= %d)", op.EndLine, op.StartLine, len(lines))
	}

	// Replace lines
	newLines := make([]string, 0, len(lines))
	newLines = append(newLines, lines[:op.StartLine-1]...)
	newLines = append(newLines, op.Content)
	newLines = append(newLines, lines[op.EndLine:]...)

	return []byte(strings.Join(newLines, "\n") + "\n"), nil
}

// applyReplacePattern replaces all occurrences of a pattern
func (e *fileEditor) applyReplacePattern(content []byte, op EditOperation) ([]byte, error) {
	if op.IsRegex {
		// Use regex replacement
		re, err := regexp.Compile(op.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return re.ReplaceAll(content, []byte(op.Replacement)), nil
	}

	// Use simple string replacement
	return []byte(strings.ReplaceAll(string(content), op.Pattern, op.Replacement)), nil
}

// generateDiff generates a simple unified diff
func (e *fileEditor) generateDiff(original, new []byte) string {
	if bytes.Equal(original, new) {
		return ""
	}

	originalLines := splitLines(original)
	newLines := splitLines(new)

	var diff strings.Builder
	diff.WriteString("--- original\n")
	diff.WriteString("+++ modified\n")

	// Simple line-by-line diff
	maxLen := len(originalLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	lineNum := 1
	for i := 0; i < maxLen; i++ {
		var origLine, newLine string
		if i < len(originalLines) {
			origLine = originalLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if origLine != newLine {
			if origLine != "" {
				diff.WriteString(fmt.Sprintf("@@ -%d @@\n", lineNum))
				diff.WriteString(fmt.Sprintf("-%s\n", origLine))
			}
			if newLine != "" {
				diff.WriteString(fmt.Sprintf("@@ +%d @@\n", lineNum))
				diff.WriteString(fmt.Sprintf("+%s\n", newLine))
			}
		}
		lineNum++
	}

	return diff.String()
}

// BackupManager manages file backups
type BackupManager struct {
	backupDir string
	maxBackups int
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupDir string, maxBackups int) *BackupManager {
	return &BackupManager{
		backupDir:  backupDir,
		maxBackups: maxBackups,
	}
}

// CreateBackup creates a backup of a file
func (bm *BackupManager) CreateBackup(path string) (string, error) {
	// Read original file
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file for backup: %w", err)
	}

	// Create backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.%s.bak", filepath.Base(path), timestamp)
	backupPath := filepath.Join(bm.backupDir, backupName)

	// Create backup directory if needed
	if err := os.MkdirAll(bm.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Write backup
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	// Clean old backups
	if err := bm.cleanOldBackups(filepath.Base(path)); err != nil {
		// Log error but don't fail
		fmt.Printf("warning: failed to clean old backups: %v\n", err)
	}

	return backupPath, nil
}

// RestoreBackup restores a file from backup
func (bm *BackupManager) RestoreBackup(backupPath, targetPath string) error {
	// Read backup
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Write to target
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// ListBackups lists all backups for a file
func (bm *BackupManager) ListBackups(filename string) ([]string, error) {
	pattern := filepath.Join(bm.backupDir, filename+"*.bak")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	return matches, nil
}

// cleanOldBackups removes old backups exceeding maxBackups
func (bm *BackupManager) cleanOldBackups(filename string) error {
	backups, err := bm.ListBackups(filename)
	if err != nil {
		return err
	}

	if len(backups) <= bm.maxBackups {
		return nil
	}

	// Sort by modification time (oldest first)
	type backupInfo struct {
		path    string
		modTime time.Time
	}

	var backupInfos []backupInfo
	for _, backup := range backups {
		info, err := os.Stat(backup)
		if err != nil {
			continue
		}
		backupInfos = append(backupInfos, backupInfo{
			path:    backup,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time
	for i := 0; i < len(backupInfos); i++ {
		for j := i + 1; j < len(backupInfos); j++ {
			if backupInfos[i].modTime.After(backupInfos[j].modTime) {
				backupInfos[i], backupInfos[j] = backupInfos[j], backupInfos[i]
			}
		}
	}

	// Delete oldest backups
	toDelete := len(backupInfos) - bm.maxBackups
	for i := 0; i < toDelete; i++ {
		if err := os.Remove(backupInfos[i].path); err != nil {
			return fmt.Errorf("failed to delete old backup: %w", err)
		}
	}

	return nil
}
