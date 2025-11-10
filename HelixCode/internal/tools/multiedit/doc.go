// Package multiedit provides atomic multi-file editing capabilities with transactional semantics.
//
// # Overview
//
// The multiedit package enables making changes across multiple files as a single transactional
// operation with all-or-nothing semantics. It ensures consistency, provides rollback capabilities,
// and includes conflict detection and resolution.
//
// # Key Features
//
//   - Atomic Operations: All-or-nothing edits across multiple files
//   - Transaction Management: Full transaction lifecycle with state tracking
//   - Backup and Restore: Automatic backup before edits with rollback support
//   - Diff Generation: Unified diff generation and application
//   - Conflict Detection: Checksum-based conflict detection
//   - Preview Mode: Preview changes before applying them
//   - Multiple Operations: Support for create, update, delete operations
//
// # Architecture
//
// The package is organized into several core components:
//
//   - MultiFileEditor: Main coordinator for multi-file editing operations
//   - TransactionManager: Manages transaction lifecycle and state
//   - BackupManager: Handles file backups and restoration
//   - DiffManager: Generates and applies unified diffs
//   - ConflictResolver: Detects and resolves conflicts
//   - PreviewEngine: Generates previews of changes
//
// # Usage Example
//
// Basic multi-file edit:
//
//	// Create editor
//	editor, err := multiedit.NewMultiFileEditor()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Begin transaction
//	tx, err := editor.BeginEdit(ctx, multiedit.EditOptions{
//	    BackupEnabled: true,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Add file edits
//	err = editor.AddEdit(ctx, tx, &multiedit.FileEdit{
//	    FilePath:   "main.go",
//	    Operation:  multiedit.OpUpdate,
//	    OldContent: []byte("old content"),
//	    NewContent: []byte("new content"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Preview changes
//	preview, err := editor.Preview(ctx, tx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Will modify %d files\n", preview.Summary.TotalFiles)
//
//	// Commit changes
//	err = editor.Commit(ctx, tx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Transaction States
//
// Transactions progress through the following states:
//
//   - Pending: Transaction created, edits can be added
//   - Preview: Preview generated, changes can be reviewed
//   - Ready: Validation passed, ready to commit
//   - Committing: Changes being applied
//   - Committed: Successfully completed
//   - RollingBack: Reverting changes
//   - RolledBack: Successfully reverted
//   - Aborted: Cancelled by user or timeout
//   - Failed: Error occurred during commit
//
// # State Machine
//
// Valid state transitions:
//
//	Pending -> Preview -> Ready -> Committing -> Committed
//	                               |
//	                               +-> RollingBack -> RolledBack
//	Any state -> Aborted (on timeout or cancel)
//
// # Conflict Detection
//
// The package detects conflicts by:
//
//   - Checksum verification: Files are checksummed before editing
//   - Modification detection: Changes since transaction started are detected
//   - Git integration: Optional git-aware conflict detection
//
// # Backup Strategy
//
// Backups are created before applying changes:
//
//   - Timestamped backups: Each backup has a unique timestamp
//   - Metadata storage: File mode, checksum, and git ref stored
//   - Compression support: Optional gzip compression for large files
//   - Automatic cleanup: Old backups cleaned up based on retention policy
//
// # Error Handling
//
// The package uses custom error types for different scenarios:
//
//   - ErrTransactionNotFound: Transaction ID not found
//   - ErrInvalidState: Invalid state transition
//   - ErrConflictDetected: File conflict detected
//   - ErrBackupFailed: Backup operation failed
//   - ErrRollbackFailed: Rollback operation failed
//   - ErrChecksumMismatch: File checksum mismatch
//
// # Performance Considerations
//
// The package includes several optimizations:
//
//   - Parallel writes: Multiple files written concurrently
//   - Streaming diffs: Large files processed in streams
//   - Memory pooling: Reusable buffers for I/O operations
//   - Checksum caching: File checksums cached for performance
//
// # Configuration
//
// The editor can be configured with various options:
//
//	config := multiedit.DefaultConfig()
//	config.MaxFiles = 1000              // Maximum files per transaction
//	config.MaxFileSize = 10 * 1024 * 1024  // 10MB file size limit
//	config.BackupRetention = 7 * 24 * time.Hour  // 7 days retention
//	config.ParallelWrites = 4           // 4 concurrent writes
//
//	editor, err := multiedit.NewMultiFileEditor(
//	    multiedit.WithConfig(config),
//	)
//
// # Integration with Filesystem
//
// The package integrates with the filesystem package for:
//
//   - Path validation and security checks
//   - Permission checking
//   - Atomic write operations
//   - File locking
//
// # Safety Features
//
//   - Path validation: Prevents directory traversal
//   - Permission checks: Validates write permissions
//   - Atomic operations: All-or-nothing semantics
//   - Checksum verification: Ensures file integrity
//   - Backup before write: Always backup before modifying
//   - Rollback on error: Automatic rollback on failures
//
// # Limitations
//
//   - Rename operation not yet implemented
//   - Git integration is placeholder (future enhancement)
//   - Simplified diff algorithm (not full Myers algorithm)
//   - No three-way merge support yet
//
// # Future Enhancements
//
//   - Full Myers diff algorithm implementation
//   - Three-way merge support
//   - Git integration for better conflict detection
//   - Distributed transaction support
//   - Incremental commits
//   - Change history with undo/redo
//
// # Thread Safety
//
// The package is thread-safe:
//
//   - TransactionManager uses sync.RWMutex for concurrent access
//   - BackupManager operations are atomic
//   - File locks prevent concurrent modifications
//
// # References
//
// The design is inspired by:
//
//   - Cline multi-file editing: Transaction-based with preview
//   - Aider edit formats: Search/replace and unified diffs
//   - Database transactions: ACID properties
//   - Git: Version control and conflict resolution
//
// # Related Packages
//
//   - dev.helix.code/internal/tools/filesystem: File system operations
//   - github.com/google/uuid: UUID generation
//   - compress/gzip: Backup compression
package multiedit
