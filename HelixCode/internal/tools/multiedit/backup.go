package multiedit

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupManager handles file backups and restoration
type BackupManager struct {
	backupDir string
	retention time.Duration
}

// BackupMetadata stores backup information
type BackupMetadata struct {
	OriginalPath string      `json:"original_path"`
	BackupTime   time.Time   `json:"backup_time"`
	Checksum     string      `json:"checksum"`
	FileMode     os.FileMode `json:"file_mode"`
	GitRef       string      `json:"git_ref,omitempty"` // Git commit if available
	Size         int64       `json:"size"`
	Compressed   bool        `json:"compressed"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupDir string, retention time.Duration) *BackupManager {
	return &BackupManager{
		backupDir: backupDir,
		retention: retention,
	}
}

// Backup creates a backup of a file
func (bm *BackupManager) Backup(ctx context.Context, filePath string) (string, error) {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// Read original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Generate backup path with timestamp
	backupPath := bm.generateBackupPath(filePath)

	// Ensure backup directory exists
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create metadata
	metadata := BackupMetadata{
		OriginalPath: filePath,
		BackupTime:   time.Now(),
		Checksum:     calculateChecksum(content),
		FileMode:     info.Mode(),
		Size:         info.Size(),
		Compressed:   false,
	}

	// Write backup with metadata
	if err := bm.writeBackupWithMetadata(backupPath, content, metadata); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// BackupWithCompression creates a compressed backup
func (bm *BackupManager) BackupWithCompression(ctx context.Context, filePath string) (string, error) {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// Read original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Generate backup path with .gz extension
	backupPath := bm.generateBackupPath(filePath) + ".gz"

	// Ensure backup directory exists
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create metadata
	metadata := BackupMetadata{
		OriginalPath: filePath,
		BackupTime:   time.Now(),
		Checksum:     calculateChecksum(content),
		FileMode:     info.Mode(),
		Size:         info.Size(),
		Compressed:   true,
	}

	// Compress and write backup
	if err := bm.writeCompressedBackup(backupPath, content, metadata); err != nil {
		return "", fmt.Errorf("failed to write compressed backup: %w", err)
	}

	return backupPath, nil
}

// Restore restores a file from backup
func (bm *BackupManager) Restore(ctx context.Context, backupPath, targetPath string) error {
	// Read backup metadata
	metadata, err := bm.readBackupMetadata(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup metadata: %w", err)
	}

	// Read backup content
	var content []byte
	if metadata.Compressed {
		content, err = bm.readCompressedBackup(backupPath)
	} else {
		content, err = bm.readBackup(backupPath)
	}
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Verify checksum
	if metadata.Checksum != "" {
		currentChecksum := calculateChecksum(content)
		if currentChecksum != metadata.Checksum {
			return fmt.Errorf("%w: backup file corrupted", ErrChecksumMismatch)
		}
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write file atomically
	tmpFile, err := os.CreateTemp(targetDir, ".restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(tmpPath, metadata.FileMode); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// Cleanup removes old backups
func (bm *BackupManager) Cleanup(ctx context.Context) error {
	if bm.retention == 0 {
		return nil // No retention policy
	}

	cutoff := time.Now().Add(-bm.retention)

	return filepath.Walk(bm.backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if backup file
		if filepath.Ext(path) == ".backup" || filepath.Ext(path) == ".gz" {
			// Read metadata to get backup time
			metadata, err := bm.readBackupMetadata(path)
			if err != nil {
				// If we can't read metadata, check file modification time
				if info.ModTime().Before(cutoff) {
					return os.Remove(path)
				}
				return nil
			}

			if metadata.BackupTime.Before(cutoff) {
				return os.Remove(path)
			}
		}

		return nil
	})
}

// List returns all backups for a file
func (bm *BackupManager) List(filePath string) ([]BackupMetadata, error) {
	var backups []BackupMetadata

	// Get backup file pattern
	pattern := bm.getBackupPattern(filePath)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob backups: %w", err)
	}

	for _, match := range matches {
		metadata, err := bm.readBackupMetadata(match)
		if err != nil {
			continue // Skip corrupted backups
		}
		backups = append(backups, metadata)
	}

	return backups, nil
}

// generateBackupPath generates a unique backup path
func (bm *BackupManager) generateBackupPath(filePath string) string {
	// Clean path and get base name
	cleanPath := filepath.Clean(filePath)
	baseName := filepath.Base(cleanPath)

	// Create subdirectory based on file path to avoid collisions
	pathHash := calculateChecksum([]byte(cleanPath))[:8]

	// Generate timestamp-based backup name
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.%s.%s.backup", baseName, timestamp, pathHash)

	return filepath.Join(bm.backupDir, backupName)
}

// getBackupPattern returns the glob pattern for finding backups of a file
func (bm *BackupManager) getBackupPattern(filePath string) string {
	cleanPath := filepath.Clean(filePath)
	baseName := filepath.Base(cleanPath)
	pathHash := calculateChecksum([]byte(cleanPath))[:8]

	pattern := filepath.Join(bm.backupDir, fmt.Sprintf("%s.*.%s.backup*", baseName, pathHash))
	return pattern
}

// writeBackupWithMetadata writes backup file with metadata
func (bm *BackupManager) writeBackupWithMetadata(backupPath string, content []byte, metadata BackupMetadata) error {
	// Marshal metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create backup file
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	// Write metadata size (4 bytes)
	metadataSize := uint32(len(metadataJSON))
	if err := writeUint32(file, metadataSize); err != nil {
		return fmt.Errorf("failed to write metadata size: %w", err)
	}

	// Write metadata
	if _, err := file.Write(metadataJSON); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Write content
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// writeCompressedBackup writes compressed backup
func (bm *BackupManager) writeCompressedBackup(backupPath string, content []byte, metadata BackupMetadata) error {
	// Marshal metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create backup file
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	// Write metadata size (4 bytes)
	metadataSize := uint32(len(metadataJSON))
	if err := writeUint32(file, metadataSize); err != nil {
		return fmt.Errorf("failed to write metadata size: %w", err)
	}

	// Write metadata
	if _, err := file.Write(metadataJSON); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Create gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Write compressed content
	if _, err := gzipWriter.Write(content); err != nil {
		return fmt.Errorf("failed to write compressed content: %w", err)
	}

	// Close gzip writer
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// readBackupMetadata reads backup metadata
func (bm *BackupManager) readBackupMetadata(backupPath string) (BackupMetadata, error) {
	var metadata BackupMetadata

	file, err := os.Open(backupPath)
	if err != nil {
		return metadata, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Read metadata size
	metadataSize, err := readUint32(file)
	if err != nil {
		return metadata, fmt.Errorf("failed to read metadata size: %w", err)
	}

	// Read metadata
	metadataJSON := make([]byte, metadataSize)
	if _, err := io.ReadFull(file, metadataJSON); err != nil {
		return metadata, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Unmarshal metadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return metadata, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return metadata, nil
}

// readBackup reads backup content
func (bm *BackupManager) readBackup(backupPath string) ([]byte, error) {
	file, err := os.Open(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Read metadata size
	metadataSize, err := readUint32(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata size: %w", err)
	}

	// Skip metadata
	if _, err := file.Seek(int64(metadataSize), io.SeekCurrent); err != nil {
		return nil, fmt.Errorf("failed to skip metadata: %w", err)
	}

	// Read content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return content, nil
}

// readCompressedBackup reads compressed backup content
func (bm *BackupManager) readCompressedBackup(backupPath string) ([]byte, error) {
	file, err := os.Open(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Read metadata size
	metadataSize, err := readUint32(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata size: %w", err)
	}

	// Skip metadata
	if _, err := file.Seek(int64(metadataSize), io.SeekCurrent); err != nil {
		return nil, fmt.Errorf("failed to skip metadata: %w", err)
	}

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Read decompressed content
	content, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read compressed content: %w", err)
	}

	return content, nil
}

// Helper functions for binary I/O

func writeUint32(w io.Writer, value uint32) error {
	bytes := []byte{
		byte(value >> 24),
		byte(value >> 16),
		byte(value >> 8),
		byte(value),
	}
	_, err := w.Write(bytes)
	return err
}

func readUint32(r io.Reader) (uint32, error) {
	bytes := make([]byte, 4)
	if _, err := io.ReadFull(r, bytes); err != nil {
		return 0, err
	}
	return uint32(bytes[0])<<24 | uint32(bytes[1])<<16 | uint32(bytes[2])<<8 | uint32(bytes[3]), nil
}
