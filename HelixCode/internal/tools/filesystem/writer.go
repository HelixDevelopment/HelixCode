package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileWriter provides methods for writing file contents
type FileWriter interface {
	// Write writes content to a file, creating it if it doesn't exist
	Write(ctx context.Context, path string, content []byte) error

	// WriteWithOptions writes content with specific options
	WriteWithOptions(ctx context.Context, path string, content []byte, opts WriteOptions) error

	// WriteLines writes lines to a file
	WriteLines(ctx context.Context, path string, lines []string) error

	// Append appends content to an existing file
	Append(ctx context.Context, path string, content []byte) error

	// Create creates a new file (fails if exists)
	Create(ctx context.Context, path string, content []byte) error

	// CreateDirectory creates a directory and all parent directories
	CreateDirectory(ctx context.Context, path string, mode os.FileMode) error

	// Delete deletes a file or directory
	Delete(ctx context.Context, path string, recursive bool) error

	// Move moves or renames a file
	Move(ctx context.Context, src, dst string) error

	// Copy copies a file
	Copy(ctx context.Context, src, dst string) error
}

// WriteOptions configures write operations
type WriteOptions struct {
	Mode         os.FileMode
	CreateParent bool
	Atomic       bool // Use atomic write (write to temp, then rename)
	Backup       bool // Create backup before overwrite
	PreserveMode bool // Preserve existing file mode
	LineEnding   LineEndingType
}

// DefaultWriteOptions returns default write options
func DefaultWriteOptions() WriteOptions {
	return WriteOptions{
		Mode:         0644,
		CreateParent: true,
		Atomic:       true,
		Backup:       false,
		PreserveMode: false,
		LineEnding:   LineEndingLF,
	}
}

// fileWriter implements FileWriter
type fileWriter struct {
	fs *FileSystemTools
}

// Write writes content to a file
func (w *fileWriter) Write(ctx context.Context, path string, content []byte) error {
	return w.WriteWithOptions(ctx, path, content, DefaultWriteOptions())
}

// WriteWithOptions writes content with specific options
func (w *fileWriter) WriteWithOptions(ctx context.Context, path string, content []byte, opts WriteOptions) error {
	// Validate path
	validationResult, err := w.fs.pathValidator.Validate(path)
	if err != nil {
		return err
	}
	normalizedPath := validationResult.NormalizedPath

	// Check if sensitive file
	if w.fs.sensitiveDetector.IsSensitive(normalizedPath) {
		return &SecurityError{
			Type:    "sensitive_file",
			Message: "refusing to write potentially sensitive file",
			Path:    normalizedPath,
		}
	}

	// Acquire lock
	lock, err := w.fs.lockManager.Acquire(ctx, normalizedPath, "writer")
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer w.fs.lockManager.Release(lock)

	// Check if file exists
	existingInfo, existsErr := os.Stat(normalizedPath)
	fileExists := existsErr == nil

	// Preserve mode if requested
	if opts.PreserveMode && fileExists {
		opts.Mode = existingInfo.Mode()
	}

	// Create parent directory if needed
	if opts.CreateParent {
		parentDir := filepath.Dir(normalizedPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	// Check permissions
	if err := w.fs.permissionChecker.CheckPermission(normalizedPath, OpWrite); err != nil {
		return err
	}

	// Create backup if requested and file exists
	if opts.Backup && fileExists {
		backupPath := normalizedPath + ".bak." + time.Now().Format("20060102-150405")
		if err := w.copyFile(normalizedPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Convert line endings if needed
	content = w.convertLineEndings(content, opts.LineEnding)

	// Write the file
	if opts.Atomic {
		if err := AtomicWrite(normalizedPath, content, opts.Mode); err != nil {
			return fmt.Errorf("failed to write file atomically: %w", err)
		}
	} else {
		if err := os.WriteFile(normalizedPath, content, opts.Mode); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	// Invalidate cache
	if w.fs.cacheManager != nil {
		w.fs.cacheManager.Invalidate(normalizedPath)
	}

	return nil
}

// WriteLines writes lines to a file
func (w *fileWriter) WriteLines(ctx context.Context, path string, lines []string) error {
	content := []byte(strings.Join(lines, "\n"))
	if len(lines) > 0 {
		content = append(content, '\n')
	}
	return w.Write(ctx, path, content)
}

// Append appends content to an existing file
func (w *fileWriter) Append(ctx context.Context, path string, content []byte) error {
	// Validate path
	validationResult, err := w.fs.pathValidator.Validate(path)
	if err != nil {
		return err
	}
	normalizedPath := validationResult.NormalizedPath

	// Check permissions
	if err := w.fs.permissionChecker.CheckPermission(normalizedPath, OpWrite); err != nil {
		return err
	}

	// Acquire lock
	lock, err := w.fs.lockManager.Acquire(ctx, normalizedPath, "writer")
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer w.fs.lockManager.Release(lock)

	// Open file for appending
	file, err := os.OpenFile(normalizedPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for appending: %w", err)
	}
	defer file.Close()

	// Write content
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("failed to append to file: %w", err)
	}

	// Invalidate cache
	if w.fs.cacheManager != nil {
		w.fs.cacheManager.Invalidate(normalizedPath)
	}

	return nil
}

// Create creates a new file (fails if exists)
func (w *fileWriter) Create(ctx context.Context, path string, content []byte) error {
	// Validate path
	validationResult, err := w.fs.pathValidator.Validate(path)
	if err != nil {
		return err
	}
	normalizedPath := validationResult.NormalizedPath

	// Check if file exists
	if _, err := os.Stat(normalizedPath); err == nil {
		return &FileSystemError{
			Type:    ErrorFileExists,
			Path:    normalizedPath,
			Message: "file already exists",
		}
	}

	// Create the file
	return w.Write(ctx, normalizedPath, content)
}

// CreateDirectory creates a directory and all parent directories
func (w *fileWriter) CreateDirectory(ctx context.Context, path string, mode os.FileMode) error {
	// Validate path
	validationResult, err := w.fs.pathValidator.Validate(path)
	if err != nil {
		return err
	}
	normalizedPath := validationResult.NormalizedPath

	// Create directory
	if err := os.MkdirAll(normalizedPath, mode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// Delete deletes a file or directory
func (w *fileWriter) Delete(ctx context.Context, path string, recursive bool) error {
	// Validate path
	validationResult, err := w.fs.pathValidator.Validate(path)
	if err != nil {
		return err
	}
	normalizedPath := validationResult.NormalizedPath

	// Check permissions
	if err := w.fs.permissionChecker.CheckPermission(normalizedPath, OpDelete); err != nil {
		return err
	}

	// Get file info
	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewFileNotFoundError(normalizedPath)
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Acquire lock
	lock, err := w.fs.lockManager.Acquire(ctx, normalizedPath, "writer")
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer w.fs.lockManager.Release(lock)

	// Delete
	if info.IsDir() {
		if recursive {
			if err := os.RemoveAll(normalizedPath); err != nil {
				return fmt.Errorf("failed to delete directory: %w", err)
			}
		} else {
			if err := os.Remove(normalizedPath); err != nil {
				return fmt.Errorf("failed to delete directory: %w", err)
			}
		}
	} else {
		if err := os.Remove(normalizedPath); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	// Invalidate cache
	if w.fs.cacheManager != nil {
		w.fs.cacheManager.Invalidate(normalizedPath)
	}

	return nil
}

// Move moves or renames a file
func (w *fileWriter) Move(ctx context.Context, src, dst string) error {
	// Validate paths
	srcValidation, err := w.fs.pathValidator.Validate(src)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	srcPath := srcValidation.NormalizedPath

	dstValidation, err := w.fs.pathValidator.Validate(dst)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}
	dstPath := dstValidation.NormalizedPath

	// Check permissions
	if err := w.fs.permissionChecker.CheckPermission(srcPath, OpWrite); err != nil {
		return err
	}
	if err := w.fs.permissionChecker.CheckPermission(dstPath, OpWrite); err != nil {
		return err
	}

	// Acquire locks
	srcLock, err := w.fs.lockManager.Acquire(ctx, srcPath, "writer")
	if err != nil {
		return fmt.Errorf("failed to acquire source lock: %w", err)
	}
	defer w.fs.lockManager.Release(srcLock)

	dstLock, err := w.fs.lockManager.Acquire(ctx, dstPath, "writer")
	if err != nil {
		return fmt.Errorf("failed to acquire destination lock: %w", err)
	}
	defer w.fs.lockManager.Release(dstLock)

	// Check if source exists
	if _, err := os.Stat(srcPath); err != nil {
		if os.IsNotExist(err) {
			return NewFileNotFoundError(srcPath)
		}
		return fmt.Errorf("failed to stat source: %w", err)
	}

	// Create parent directory for destination
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	// Invalidate cache
	if w.fs.cacheManager != nil {
		w.fs.cacheManager.Invalidate(srcPath)
		w.fs.cacheManager.Invalidate(dstPath)
	}

	return nil
}

// Copy copies a file
func (w *fileWriter) Copy(ctx context.Context, src, dst string) error {
	// Validate paths
	srcValidation, err := w.fs.pathValidator.Validate(src)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	srcPath := srcValidation.NormalizedPath

	dstValidation, err := w.fs.pathValidator.Validate(dst)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}
	dstPath := dstValidation.NormalizedPath

	// Check permissions
	if err := w.fs.permissionChecker.CheckPermission(srcPath, OpRead); err != nil {
		return err
	}
	if err := w.fs.permissionChecker.CheckPermission(dstPath, OpWrite); err != nil {
		return err
	}

	// Copy file
	if err := w.copyFile(srcPath, dstPath); err != nil {
		return err
	}

	// Invalidate cache
	if w.fs.cacheManager != nil {
		w.fs.cacheManager.Invalidate(dstPath)
	}

	return nil
}

// copyFile copies a file from src to dst
func (w *fileWriter) copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Check if source is a directory
	if srcInfo.IsDir() {
		return &FileSystemError{
			Type:    ErrorIsDirectory,
			Path:    src,
			Message: "source is a directory",
		}
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Sync to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// convertLineEndings converts content to specified line ending type
func (w *fileWriter) convertLineEndings(content []byte, lineEnding LineEndingType) []byte {
	if lineEnding == LineEndingUnknown || len(content) == 0 {
		return content
	}

	// Normalize to LF first
	content = []byte(strings.ReplaceAll(string(content), "\r\n", "\n"))
	content = []byte(strings.ReplaceAll(string(content), "\r", "\n"))

	// Convert to target line ending
	switch lineEnding {
	case LineEndingCRLF:
		content = []byte(strings.ReplaceAll(string(content), "\n", "\r\n"))
	case LineEndingCR:
		content = []byte(strings.ReplaceAll(string(content), "\n", "\r"))
	case LineEndingLF:
		// Already normalized to LF
	}

	return content
}
