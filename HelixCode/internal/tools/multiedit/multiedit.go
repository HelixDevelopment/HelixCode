package multiedit

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"dev.helix.code/internal/tools/filesystem"
)

// MultiFileEditor coordinates multi-file editing operations
type MultiFileEditor struct {
	config           *Config
	txManager        *TransactionManager
	backupManager    *BackupManager
	diffManager      *DiffManager
	conflictResolver *ConflictResolver
	previewEngine    *PreviewEngine
	fileSystem       *filesystem.FileSystemTools
	logger           *slog.Logger
	mu               sync.RWMutex
}

// Config contains configuration for multi-file editing
type Config struct {
	// Transaction settings
	MaxDuration time.Duration
	MaxFiles    int
	MaxFileSize int64
	Timeout     time.Duration

	// Backup settings
	BackupEnabled     bool
	BackupDir         string
	BackupRetention   time.Duration
	BackupCompression bool

	// Conflict resolution
	ConflictPolicy   ConflictPolicy
	AutoResolve      bool
	DetectGitChanges bool

	// Preview settings
	PreviewFormat   DiffFormat
	ContextLines    int
	SyntaxHighlight bool
	ShowLineNumbers bool

	// Git integration
	GitEnabled       bool
	GitAutoStage     bool
	CheckUncommitted bool
	RespectGitignore bool

	// Safety settings
	RequirePreview bool
	WorkspaceRoot  string
	AllowedPaths   []string
	DeniedPaths    []string
	MaxRetries     int

	// Performance
	ParallelWrites int
	BufferSize     int
	UseMemoryCache bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxDuration:       1 * time.Hour,
		MaxFiles:          1000,
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		Timeout:           30 * time.Minute,
		BackupEnabled:     true,
		BackupDir:         ".helix/backups",
		BackupRetention:   7 * 24 * time.Hour,
		BackupCompression: true,
		ConflictPolicy:    ConflictPolicyAbort,
		AutoResolve:       false,
		DetectGitChanges:  true,
		PreviewFormat:     FormatUnified,
		ContextLines:      3,
		SyntaxHighlight:   true,
		ShowLineNumbers:   true,
		GitEnabled:        true,
		GitAutoStage:      false,
		CheckUncommitted:  true,
		RespectGitignore:  true,
		RequirePreview:    true,
		AllowedPaths: []string{
			"**/*.go",
			"**/*.md",
			"**/config/**",
		},
		DeniedPaths: []string{
			"**/.git/**",
			"**/node_modules/**",
			"**/vendor/**",
		},
		MaxRetries:     3,
		ParallelWrites: 4,
		BufferSize:     4096,
		UseMemoryCache: true,
	}
}

// Option is a functional option for configuring MultiFileEditor
type Option func(*MultiFileEditor)

// WithConfig sets the configuration
func WithConfig(config *Config) Option {
	return func(mfe *MultiFileEditor) {
		mfe.config = config
	}
}

// WithLogger sets the logger
func WithLogger(logger *slog.Logger) Option {
	return func(mfe *MultiFileEditor) {
		mfe.logger = logger
	}
}

// WithFileSystem sets the file system tools
func WithFileSystem(fs *filesystem.FileSystemTools) Option {
	return func(mfe *MultiFileEditor) {
		mfe.fileSystem = fs
	}
}

// WithBackupDir sets the backup directory
func WithBackupDir(dir string) Option {
	return func(mfe *MultiFileEditor) {
		mfe.config.BackupDir = dir
	}
}

// NewMultiFileEditor creates a new multi-file editor
func NewMultiFileEditor(opts ...Option) (*MultiFileEditor, error) {
	mfe := &MultiFileEditor{
		config: DefaultConfig(),
		logger: slog.Default(),
	}

	// Apply options
	for _, opt := range opts {
		opt(mfe)
	}

	// Initialize file system if not provided
	if mfe.fileSystem == nil {
		fsConfig := filesystem.DefaultConfig()
		fsConfig.WorkspaceRoot = mfe.config.WorkspaceRoot
		fsConfig.MaxFileSize = mfe.config.MaxFileSize
		fsConfig.AllowedPaths = mfe.config.AllowedPaths
		fsConfig.BlockedPaths = mfe.config.DeniedPaths

		fs, err := filesystem.NewFileSystemTools(fsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create filesystem tools: %w", err)
		}
		mfe.fileSystem = fs
	}

	// Initialize components
	mfe.backupManager = NewBackupManager(mfe.config.BackupDir, mfe.config.BackupRetention)
	mfe.diffManager = NewDiffManager(mfe.config.PreviewFormat)
	mfe.txManager = NewTransactionManager(mfe.config.MaxDuration)
	mfe.conflictResolver = NewConflictResolver(mfe.config.DetectGitChanges)
	mfe.previewEngine = NewPreviewEngine(mfe.diffManager, mfe.config.ContextLines)

	return mfe, nil
}

// BeginEdit starts a new multi-file edit transaction
func (mfe *MultiFileEditor) BeginEdit(ctx context.Context, opts EditOptions) (*EditTransaction, error) {
	mfe.mu.Lock()
	defer mfe.mu.Unlock()

	// Create transaction
	tx, err := mfe.txManager.Begin(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	mfe.logger.Info("transaction started",
		"tx_id", tx.ID,
		"options", opts,
	)

	return tx, nil
}

// AddEdit adds a file edit to the transaction
func (mfe *MultiFileEditor) AddEdit(ctx context.Context, tx *EditTransaction, edit *FileEdit) error {
	if err := mfe.validateEdit(edit); err != nil {
		return fmt.Errorf("invalid edit: %w", err)
	}

	// Validate path (skip for create operations)
	if edit.Operation != OpCreate {
		_, err := mfe.fileSystem.Reader().GetInfo(ctx, edit.FilePath)
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}
	}

	// Calculate checksum of old content
	if edit.Operation == OpUpdate || edit.Operation == OpDelete {
		if edit.Checksum == "" && len(edit.OldContent) > 0 {
			edit.Checksum = calculateChecksum(edit.OldContent)
		}
	}

	// Add to transaction
	if err := mfe.txManager.AddEdit(tx, edit); err != nil {
		return fmt.Errorf("failed to add edit: %w", err)
	}

	mfe.logger.Debug("edit added",
		"tx_id", tx.ID,
		"file", edit.FilePath,
		"operation", edit.Operation,
	)

	return nil
}

// Preview generates a preview of changes without applying them
func (mfe *MultiFileEditor) Preview(ctx context.Context, tx *EditTransaction) (*PreviewResult, error) {
	mfe.mu.RLock()
	defer mfe.mu.RUnlock()

	// Validate transaction state
	if tx.State != StatePending && tx.State != StatePreview && tx.State != StateReady {
		return nil, ErrInvalidState
	}

	// Update state
	if err := mfe.txManager.UpdateState(tx, StatePreview); err != nil {
		return nil, err
	}

	// Generate preview
	preview, err := mfe.previewEngine.Preview(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate preview: %w", err)
	}

	// Detect conflicts
	conflicts, err := mfe.conflictResolver.DetectConflicts(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect conflicts: %w", err)
	}
	preview.Conflicts = conflicts
	preview.Summary.HasConflicts = len(conflicts) > 0

	// Update state
	if len(conflicts) == 0 {
		if err := mfe.txManager.UpdateState(tx, StateReady); err != nil {
			return nil, err
		}
	}

	mfe.logger.Info("preview generated",
		"tx_id", tx.ID,
		"files", len(preview.Files),
		"conflicts", len(conflicts),
	)

	return preview, nil
}

// Commit applies all changes atomically
func (mfe *MultiFileEditor) Commit(ctx context.Context, tx *EditTransaction) error {
	mfe.mu.Lock()
	defer mfe.mu.Unlock()

	// Validate state
	if tx.State != StateReady {
		return ErrInvalidState
	}

	// Update state
	if err := mfe.txManager.UpdateState(tx, StateCommitting); err != nil {
		return err
	}

	mfe.logger.Info("committing transaction",
		"tx_id", tx.ID,
		"files", len(tx.Files),
	)

	// Create backups if enabled
	if mfe.config.BackupEnabled {
		if err := mfe.createBackups(ctx, tx); err != nil {
			mfe.rollbackWithError(ctx, tx, fmt.Errorf("backup failed: %w", err))
			return err
		}
	}

	// Apply edits
	if err := mfe.applyEdits(ctx, tx); err != nil {
		mfe.logger.Error("failed to apply edits, rolling back",
			"tx_id", tx.ID,
			"error", err,
		)
		return mfe.Rollback(ctx, tx)
	}

	// Update state
	if err := mfe.txManager.UpdateState(tx, StateCommitted); err != nil {
		return err
	}

	mfe.logger.Info("transaction committed",
		"tx_id", tx.ID,
		"duration", time.Since(tx.CreatedAt),
	)

	return nil
}

// Rollback reverts all changes in the transaction
func (mfe *MultiFileEditor) Rollback(ctx context.Context, tx *EditTransaction) error {
	mfe.mu.Lock()
	defer mfe.mu.Unlock()

	return mfe.rollbackWithError(ctx, tx, nil)
}

// rollbackWithError performs the actual rollback
func (mfe *MultiFileEditor) rollbackWithError(ctx context.Context, tx *EditTransaction, cause error) error {
	if tx.State == StateRolledBack || tx.State == StateAborted {
		return nil
	}

	// Update state
	if err := mfe.txManager.UpdateState(tx, StateRollingBack); err != nil {
		return err
	}

	mfe.logger.Warn("rolling back transaction",
		"tx_id", tx.ID,
		"cause", cause,
	)

	// Restore backups in reverse order
	var errors []error
	for i := len(tx.Files) - 1; i >= 0; i-- {
		edit := tx.Files[i]
		if !edit.Applied {
			continue
		}

		backupPath, ok := tx.backupPaths[edit.FilePath]
		if !ok {
			continue
		}

		if err := mfe.backupManager.Restore(ctx, backupPath, edit.FilePath); err != nil {
			errors = append(errors, fmt.Errorf("failed to restore %s: %w", edit.FilePath, err))
			mfe.logger.Error("failed to restore file",
				"file", edit.FilePath,
				"error", err,
			)
		}
	}

	// Update state
	if err := mfe.txManager.UpdateState(tx, StateRolledBack); err != nil {
		return err
	}

	if len(errors) > 0 {
		return fmt.Errorf("rollback completed with errors: %v", errors)
	}

	mfe.logger.Info("transaction rolled back",
		"tx_id", tx.ID,
	)

	return cause
}

// GetTransaction retrieves a transaction by ID
func (mfe *MultiFileEditor) GetTransaction(ctx context.Context, txID string) (*EditTransaction, error) {
	return mfe.txManager.Get(txID)
}

// createBackups creates backups for all files
func (mfe *MultiFileEditor) createBackups(ctx context.Context, tx *EditTransaction) error {
	for _, edit := range tx.Files {
		if edit.Operation == OpCreate {
			continue // No need to backup new files
		}

		backupPath, err := mfe.backupManager.Backup(ctx, edit.FilePath)
		if err != nil {
			return fmt.Errorf("failed to backup %s: %w", edit.FilePath, err)
		}

		tx.mu.Lock()
		tx.backupPaths[edit.FilePath] = backupPath
		tx.mu.Unlock()

		mfe.logger.Debug("file backed up",
			"file", edit.FilePath,
			"backup", backupPath,
		)
	}

	return nil
}

// applyEdits applies all edits in the transaction
func (mfe *MultiFileEditor) applyEdits(ctx context.Context, tx *EditTransaction) error {
	// Use semaphore to limit parallel writes
	sem := make(chan struct{}, mfe.config.ParallelWrites)
	var wg sync.WaitGroup
	errChan := make(chan error, len(tx.Files))

	for _, edit := range tx.Files {
		wg.Add(1)
		go func(e *FileEdit) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := mfe.applyEdit(ctx, tx, e); err != nil {
				errChan <- err
			}
		}(edit)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// applyEdit applies a single edit
func (mfe *MultiFileEditor) applyEdit(ctx context.Context, tx *EditTransaction, edit *FileEdit) error {
	// Verify checksum before applying
	if edit.Operation == OpUpdate || edit.Operation == OpDelete {
		if err := mfe.verifyChecksum(ctx, edit); err != nil {
			edit.Error = err
			return err
		}
	}

	// Apply operation
	switch edit.Operation {
	case OpCreate, OpUpdate:
		if err := mfe.fileSystem.Writer().Write(ctx, edit.FilePath, edit.NewContent); err != nil {
			edit.Error = err
			return fmt.Errorf("failed to write %s: %w", edit.FilePath, err)
		}
	case OpDelete:
		if err := mfe.fileSystem.Writer().Delete(ctx, edit.FilePath, false); err != nil {
			edit.Error = err
			return fmt.Errorf("failed to delete %s: %w", edit.FilePath, err)
		}
	case OpRename:
		// Rename not implemented yet
		return fmt.Errorf("rename operation not yet implemented")
	default:
		return fmt.Errorf("unknown operation: %v", edit.Operation)
	}

	edit.Applied = true
	return nil
}

// verifyChecksum verifies the file hasn't changed since transaction started
func (mfe *MultiFileEditor) verifyChecksum(ctx context.Context, edit *FileEdit) error {
	if edit.Checksum == "" {
		return nil // Skip verification if no checksum
	}

	content, err := mfe.fileSystem.Reader().Read(ctx, edit.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	currentChecksum := calculateChecksum(content.Content)
	if currentChecksum != edit.Checksum {
		return &ConflictError{
			Type:     ConflictModified,
			FilePath: edit.FilePath,
			Expected: edit.Checksum,
			Actual:   currentChecksum,
			Message:  "file was modified since transaction started",
		}
	}

	return nil
}

// validateEdit validates an edit
func (mfe *MultiFileEditor) validateEdit(edit *FileEdit) error {
	if edit.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	switch edit.Operation {
	case OpCreate, OpUpdate:
		if len(edit.NewContent) == 0 {
			return fmt.Errorf("new content is required for create/update")
		}
		if int64(len(edit.NewContent)) > mfe.config.MaxFileSize {
			return fmt.Errorf("file size exceeds limit: %d > %d", len(edit.NewContent), mfe.config.MaxFileSize)
		}
	case OpDelete:
		// No additional validation needed
	case OpRename:
		// Not implemented yet
		return fmt.Errorf("rename operation not yet implemented")
	default:
		return fmt.Errorf("invalid operation: %v", edit.Operation)
	}

	return nil
}
