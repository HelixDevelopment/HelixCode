# Checkpoint Snapshots - Technical Design

## Overview

Checkpoint snapshots provide a version control system for workspace state, enabling users to save, compare, and restore project states at different points in time. Built on Git, this system provides atomic snapshots with rich metadata and powerful comparison capabilities.

**References:**
- Cline's checkpoint system
- Gemini CLI checkpointing
- Git worktree and branch management

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Checkpoint System                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────────┐
        │        CheckpointManager                │
        │  - Create Checkpoint                    │
        │  - List Checkpoints                     │
        │  - Compare Checkpoints                  │
        │  - Restore Checkpoint                   │
        └─────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐     ┌──────────────┐
│ Snapshot     │    │  Comparator  │     │   Restore    │
│ Creator      │    │              │     │   Manager    │
│              │    │ • Diff       │     │              │
│ • Git Branch │    │ • Stats      │     │ • Validation │
│ • Metadata   │    │ • Conflicts  │     │ • Rollback   │
│ • Cleanup    │    │ • Summary    │     │ • Merge      │
└──────────────┘    └──────────────┘     └──────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐     ┌──────────────┐
│ Git Storage  │◀───│  Metadata    │────▶│   History    │
│ (Branches)   │    │   Storage    │     │   Tracker    │
└──────────────┘    └──────────────┘     └──────────────┘
```

---

## Component Interfaces

### CheckpointManager

```go
package checkpoint

import (
    "context"
    "time"
)

// CheckpointManager orchestrates checkpoint operations
type CheckpointManager struct {
    repo        *Repository
    creator     *SnapshotCreator
    comparator  *Comparator
    restorer    *RestoreManager
    config      *Config
    history     *HistoryTracker
}

// Config contains checkpoint system configuration
type Config struct {
    // Storage settings
    BranchPrefix       string        // Default: "checkpoint/"
    MetadataPath       string        // Default: ".helix/checkpoints"
    MaxCheckpoints     int           // Default: 50
    AutoCleanup        bool          // Default: true

    // Automatic checkpoint settings
    AutoCheckpoint     bool          // Enable automatic checkpoints
    CheckpointInterval time.Duration // Default: 0 (disabled)
    OnTaskStep         bool          // Checkpoint after each task step
    OnError            bool          // Checkpoint on errors

    // Snapshot settings
    IncludeUntracked   bool          // Include untracked files
    ExcludePatterns    []string      // Patterns to exclude
    CompressMetadata   bool          // Compress metadata

    // Restore settings
    CreateBackup       bool          // Backup before restore
    AllowDirtyRestore  bool          // Allow restore with uncommitted changes
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(repoPath string, config *Config) (*CheckpointManager, error)

// CreateCheckpoint creates a new checkpoint
func (c *CheckpointManager) CreateCheckpoint(ctx context.Context, opts *CreateOptions) (*Checkpoint, error)

// ListCheckpoints returns all checkpoints, optionally filtered
func (c *CheckpointManager) ListCheckpoints(ctx context.Context, filter *Filter) ([]*Checkpoint, error)

// GetCheckpoint retrieves a specific checkpoint by ID
func (c *CheckpointManager) GetCheckpoint(ctx context.Context, id string) (*Checkpoint, error)

// CompareCheckpoints compares two checkpoints
func (c *CheckpointManager) CompareCheckpoints(ctx context.Context, from, to string) (*Comparison, error)

// RestoreCheckpoint restores the workspace to a checkpoint
func (c *CheckpointManager) RestoreCheckpoint(ctx context.Context, id string, opts *RestoreOptions) error

// DeleteCheckpoint removes a checkpoint
func (c *CheckpointManager) DeleteCheckpoint(ctx context.Context, id string) error

// PruneCheckpoints removes old checkpoints based on policy
func (c *CheckpointManager) PruneCheckpoints(ctx context.Context, policy *PrunePolicy) (int, error)
```

### Checkpoint Data Structure

```go
package checkpoint

import (
    "time"
)

// Checkpoint represents a workspace snapshot
type Checkpoint struct {
    ID          string            // Unique identifier
    BranchName  string            // Git branch name
    CommitHash  string            // Git commit SHA
    CreatedAt   time.Time         // Creation timestamp
    Description string            // User description
    TaskID      string            // Associated task ID
    Status      CheckpointStatus  // Checkpoint status
    Metadata    *Metadata         // Rich metadata
    Tags        []string          // User-defined tags
    Size        int64             // Approximate size in bytes
    FileCount   int               // Number of files
}

// CheckpointStatus represents checkpoint state
type CheckpointStatus string

const (
    StatusActive    CheckpointStatus = "active"     // Normal checkpoint
    StatusArchived  CheckpointStatus = "archived"   // Archived checkpoint
    StatusCorrupted CheckpointStatus = "corrupted"  // Corrupted/invalid
)

// Metadata contains detailed checkpoint information
type Metadata struct {
    // Context information
    WorkingDirectory string            // Working directory at checkpoint
    Branch           string            // Active git branch
    CommitMessage    string            // Commit message

    // File statistics
    FilesAdded       []string          // Added files
    FilesModified    []string          // Modified files
    FilesDeleted     []string          // Deleted files
    UntrackedFiles   []string          // Untracked files (if included)

    // Project state
    Language         string            // Primary language
    Framework        string            // Framework/tool
    Dependencies     map[string]string // Key dependencies

    // Task context
    TaskDescription  string            // Task being worked on
    TaskStep         int               // Current task step
    LastCommand      string            // Last executed command
    LastOutput       string            // Last command output (truncated)

    // System information
    HelixVersion     string            // Helix version
    ModelUsed        string            // AI model in use
    TokensUsed       int64             // Tokens used in session

    // Custom fields
    Custom           map[string]interface{} // User-defined metadata
}

// CreateOptions specifies checkpoint creation options
type CreateOptions struct {
    Description     string            // User description
    TaskID          string            // Associated task
    Tags            []string          // Tags to apply
    IncludeUntracked bool             // Include untracked files
    Metadata        map[string]interface{} // Custom metadata
    AutoGenerate    bool              // Auto-generate description
}

// Filter for querying checkpoints
type Filter struct {
    TaskID      string            // Filter by task
    Tags        []string          // Filter by tags
    Status      CheckpointStatus  // Filter by status
    FromDate    time.Time         // Created after
    ToDate      time.Time         // Created before
    Limit       int               // Maximum results
    Offset      int               // Pagination offset
    SortBy      SortField         // Sort field
    SortOrder   SortOrder         // Sort direction
}

// SortField defines sorting options
type SortField string

const (
    SortByCreatedAt SortField = "created_at"
    SortBySize      SortField = "size"
    SortByTaskID    SortField = "task_id"
)

// SortOrder defines sort direction
type SortOrder string

const (
    SortAsc  SortOrder = "asc"
    SortDesc SortOrder = "desc"
)
```

### SnapshotCreator

```go
package checkpoint

import (
    "context"
)

// SnapshotCreator handles checkpoint creation
type SnapshotCreator struct {
    repo           *Repository
    metadataStore  *MetadataStore
    idGenerator    *IDGenerator
}

// NewSnapshotCreator creates a new snapshot creator
func NewSnapshotCreator(repo *Repository, store *MetadataStore) *SnapshotCreator

// Create creates a new checkpoint
func (s *SnapshotCreator) Create(ctx context.Context, opts *CreateOptions) (*Checkpoint, error)

// CreateAutomatic creates an automatic checkpoint
func (s *SnapshotCreator) CreateAutomatic(ctx context.Context, trigger AutoTrigger) (*Checkpoint, error)

// ValidateWorkspace checks if workspace is ready for checkpointing
func (s *SnapshotCreator) ValidateWorkspace(ctx context.Context) error

// GenerateDescription creates an automatic description
func (s *SnapshotCreator) GenerateDescription(ctx context.Context) (string, error)

// AutoTrigger represents automatic checkpoint triggers
type AutoTrigger string

const (
    TriggerTaskStep    AutoTrigger = "task_step"
    TriggerInterval    AutoTrigger = "interval"
    TriggerError       AutoTrigger = "error"
    TriggerManual      AutoTrigger = "manual"
)
```

### Comparator

```go
package checkpoint

import (
    "context"
)

// Comparator handles checkpoint comparison
type Comparator struct {
    repo *Repository
}

// Comparison represents a comparison between two checkpoints
type Comparison struct {
    From         *Checkpoint       // Source checkpoint
    To           *Checkpoint       // Target checkpoint
    Summary      *Summary          // High-level summary
    FileDiffs    []*FileDiff       // File-level differences
    Conflicts    []*Conflict       // Potential conflicts
    Statistics   *Statistics       // Diff statistics
}

// Summary provides high-level comparison information
type Summary struct {
    FilesAdded      int
    FilesModified   int
    FilesDeleted    int
    LinesAdded      int
    LinesDeleted    int
    TimeElapsed     time.Duration
    TasksCompleted  []string
}

// FileDiff represents changes to a single file
type FileDiff struct {
    Path        string
    Status      DiffStatus
    OldContent  string
    NewContent  string
    Diff        string  // Unified diff format
    LinesAdded  int
    LinesDeleted int
}

// DiffStatus represents file change status
type DiffStatus string

const (
    DiffAdded    DiffStatus = "added"
    DiffModified DiffStatus = "modified"
    DiffDeleted  DiffStatus = "deleted"
    DiffRenamed  DiffStatus = "renamed"
)

// Conflict represents a potential restore conflict
type Conflict struct {
    Path        string
    Type        ConflictType
    Description string
    Resolution  ResolutionStrategy
}

// ConflictType categorizes conflicts
type ConflictType string

const (
    ConflictModified ConflictType = "modified"
    ConflictDeleted  ConflictType = "deleted"
    ConflictAdded    ConflictType = "added"
)

// ResolutionStrategy defines how to handle conflicts
type ResolutionStrategy string

const (
    ResolutionKeepCurrent   ResolutionStrategy = "keep_current"
    ResolutionUseCheckpoint ResolutionStrategy = "use_checkpoint"
    ResolutionMerge         ResolutionStrategy = "merge"
    ResolutionAbort         ResolutionStrategy = "abort"
)

// Statistics contains detailed diff statistics
type Statistics struct {
    TotalFiles      int
    TotalLines      int
    LinesAdded      int
    LinesDeleted    int
    BinaryFiles     int
    LanguageStats   map[string]*LanguageStats
}

// LanguageStats tracks changes per language
type LanguageStats struct {
    Files        int
    LinesAdded   int
    LinesDeleted int
}

// NewComparator creates a new comparator
func NewComparator(repo *Repository) *Comparator

// Compare compares two checkpoints
func (c *Comparator) Compare(ctx context.Context, from, to string) (*Comparison, error)

// GenerateDiff generates a unified diff between checkpoints
func (c *Comparator) GenerateDiff(ctx context.Context, from, to string, opts *DiffOptions) (string, error)

// DetectConflicts identifies potential conflicts
func (c *Comparator) DetectConflicts(ctx context.Context, checkpoint, current string) ([]*Conflict, error)

// DiffOptions specifies diff generation options
type DiffOptions struct {
    Context       int      // Lines of context
    IgnoreWhitespace bool  // Ignore whitespace changes
    Unified       bool     // Unified diff format
    Stat          bool     // Include statistics
    Files         []string // Specific files to diff
}
```

### RestoreManager

```go
package checkpoint

import (
    "context"
)

// RestoreManager handles checkpoint restoration
type RestoreManager struct {
    repo       *Repository
    comparator *Comparator
    validator  *Validator
}

// RestoreOptions specifies restoration behavior
type RestoreOptions struct {
    CreateBackup       bool               // Create backup before restore
    DryRun             bool               // Preview without applying
    Force              bool               // Force restore (skip checks)
    ConflictResolution ResolutionStrategy // How to handle conflicts
    PreserveFiles      []string           // Files to preserve
    Interactive        bool               // Prompt for conflicts
}

// RestoreResult contains restoration results
type RestoreResult struct {
    Success          bool
    BackupCheckpoint *Checkpoint  // Backup created (if any)
    FilesRestored    []string
    FilesPreserved   []string
    Conflicts        []*Conflict
    Errors           []error
    Duration         time.Duration
}

// NewRestoreManager creates a new restore manager
func NewRestoreManager(repo *Repository, comparator *Comparator) *RestoreManager

// Restore restores workspace to a checkpoint
func (r *RestoreManager) Restore(ctx context.Context, checkpointID string, opts *RestoreOptions) (*RestoreResult, error)

// ValidateRestore checks if restore is safe
func (r *RestoreManager) ValidateRestore(ctx context.Context, checkpointID string) error

// PreviewRestore shows what would be restored
func (r *RestoreManager) PreviewRestore(ctx context.Context, checkpointID string) (*RestorePreview, error)

// RestorePreview shows restoration details
type RestorePreview struct {
    Checkpoint       *Checkpoint
    Comparison       *Comparison
    WillOverwrite    []string
    WillDelete       []string
    WillCreate       []string
    PotentialIssues  []string
}
```

---

## State Machines

### Checkpoint Creation State Machine

```
                    ┌─────────┐
                    │  INIT   │
                    └────┬────┘
                         │
                  Validate│
                         │
                         ▼
                    ┌─────────┐
                    │ VALIDATE│
                    └────┬────┘
                         │
            ┌────────────┼────────────┐
            │                         │
      Invalid│                        │Valid
            │                         │
            ▼                         ▼
       ┌─────────┐              ┌─────────┐
       │  ERROR  │              │  STAGE  │
       └─────────┘              └────┬────┘
                                     │
                                     │Stage Files
                                     │
                                     ▼
                                ┌─────────┐
                                │ COMMIT  │
                                └────┬────┘
                                     │
                                     │Create Branch
                                     │
                                     ▼
                                ┌─────────┐
                                │METADATA │
                                └────┬────┘
                                     │
                                     │Save Metadata
                                     │
                                     ▼
                                ┌─────────┐
                                │  DONE   │
                                └─────────┘
```

### Checkpoint Restore State Machine

```
                    ┌─────────┐
                    │  INIT   │
                    └────┬────┘
                         │
                  Load   │Checkpoint
                         │
                         ▼
                    ┌─────────┐
                    │  LOAD   │
                    └────┬────┘
                         │
                         ▼
                    ┌─────────┐
                    │ VALIDATE│
                    └────┬────┘
                         │
            ┌────────────┼────────────┐
            │                         │
         Error│                       │Valid
            │                         │
            ▼                         ▼
       ┌─────────┐              ┌─────────┐
       │  ERROR  │              │ BACKUP  │
       └─────────┘              └────┬────┘
                                     │
                                     │Create Backup
                                     │
                                     ▼
                                ┌─────────┐
                                │CONFLICTS│
                                └────┬────┘
                                     │
            ┌────────────────────────┼────────────┐
            │                        │            │
      Conflicts                      │            │No Conflicts
            │                        │Resolve     │
            ▼                        │            │
       ┌─────────┐                   │            │
       │ RESOLVE │───────────────────┘            │
       └─────────┘                                │
                                                  │
                                                  ▼
                                             ┌─────────┐
                                             │ RESTORE │
                                             └────┬────┘
                                                  │
                                                  │Apply Changes
                                                  │
                                                  ▼
                                             ┌─────────┐
                                             │ VERIFY  │
                                             └────┬────┘
                                                  │
                     ┌────────────────────────────┼─────────┐
                     │                                      │
               Success│                                     │Failed
                     │                                      │
                     ▼                                      ▼
                ┌─────────┐                            ┌─────────┐
                │  DONE   │                            │ROLLBACK │
                └─────────┘                            └────┬────┘
                                                            │
                                                            ▼
                                                       ┌─────────┐
                                                       │  ERROR  │
                                                       └─────────┘
```

---

## Git Integration

### Branch Strategy

```go
// Branch naming convention: checkpoint/{timestamp}-{short-id}
// Example: checkpoint/20250105-143022-a1b2c3

// BranchManager handles checkpoint branch operations
type BranchManager struct {
    repo *Repository
}

// CreateCheckpointBranch creates a new checkpoint branch
func (b *BranchManager) CreateCheckpointBranch(ctx context.Context, checkpoint *Checkpoint) error {
    // 1. Create branch from current HEAD
    // 2. Commit all changes
    // 3. Tag with checkpoint metadata
    // 4. Return to original branch
}

// ListCheckpointBranches lists all checkpoint branches
func (b *BranchManager) ListCheckpointBranches(ctx context.Context) ([]string, error)

// DeleteCheckpointBranch removes a checkpoint branch
func (b *BranchManager) DeleteCheckpointBranch(ctx context.Context, branchName string) error

// RestoreFromBranch restores workspace from checkpoint branch
func (b *BranchManager) RestoreFromBranch(ctx context.Context, branchName string) error
```

### Repository Operations

```go
package checkpoint

import (
    "github.com/go-git/go-git/v5"
)

// Repository wraps git operations
type Repository struct {
    repo     *git.Repository
    worktree *git.Worktree
    path     string
}

// NewRepository opens or initializes a repository
func NewRepository(path string) (*Repository, error)

// Status returns current repository status
func (r *Repository) Status() (*Status, error)

// Status represents repository state
type Status struct {
    Branch        string
    Clean         bool
    Modified      []string
    Added         []string
    Deleted       []string
    Untracked     []string
    Conflicts     []string
}

// Commit creates a commit with all changes
func (r *Repository) Commit(message string) (string, error)

// CreateBranch creates a new branch
func (r *Repository) CreateBranch(name string) error

// CheckoutBranch switches to a branch
func (r *Repository) CheckoutBranch(name string) error

// DeleteBranch deletes a branch
func (r *Repository) DeleteBranch(name string) error

// Diff generates diff between commits
func (r *Repository) Diff(from, to string) (string, error)

// ListBranches lists branches matching pattern
func (r *Repository) ListBranches(pattern string) ([]string, error)
```

---

## Metadata Storage

### Metadata Schema

```json
{
  "version": "1.0",
  "checkpoint": {
    "id": "cp_20250105_143022_a1b2c3",
    "branch_name": "checkpoint/20250105-143022-a1b2c3",
    "commit_hash": "abc123def456...",
    "created_at": "2025-01-05T14:30:22Z",
    "description": "Before refactoring auth module",
    "task_id": "task_001",
    "status": "active",
    "tags": ["before-refactor", "stable"],
    "size": 1048576,
    "file_count": 42
  },
  "metadata": {
    "working_directory": "/home/user/project",
    "branch": "main",
    "commit_message": "Checkpoint: Before refactoring auth module",
    "files_added": ["src/new_feature.go"],
    "files_modified": ["src/main.go", "README.md"],
    "files_deleted": [],
    "untracked_files": ["tmp/cache.db"],
    "language": "go",
    "framework": "gin",
    "dependencies": {
      "gin-gonic/gin": "v1.9.0",
      "spf13/cobra": "v1.8.0"
    },
    "task_description": "Refactor authentication module",
    "task_step": 3,
    "last_command": "go test ./...",
    "last_output": "PASS\nok\tproject/auth\t0.123s\n...",
    "helix_version": "0.1.0",
    "model_used": "claude-sonnet-4.5",
    "tokens_used": 15432,
    "custom": {
      "user_note": "Working state before major changes"
    }
  }
}
```

### MetadataStore

```go
package checkpoint

import (
    "context"
    "encoding/json"
)

// MetadataStore handles checkpoint metadata persistence
type MetadataStore struct {
    path       string
    compression bool
}

// NewMetadataStore creates a new metadata store
func NewMetadataStore(path string, compress bool) (*MetadataStore, error)

// Save persists checkpoint metadata
func (m *MetadataStore) Save(ctx context.Context, checkpoint *Checkpoint) error

// Load retrieves checkpoint metadata
func (m *MetadataStore) Load(ctx context.Context, id string) (*Checkpoint, error)

// List returns all checkpoints
func (m *MetadataStore) List(ctx context.Context, filter *Filter) ([]*Checkpoint, error)

// Delete removes checkpoint metadata
func (m *MetadataStore) Delete(ctx context.Context, id string) error

// Update modifies existing checkpoint metadata
func (m *MetadataStore) Update(ctx context.Context, id string, updates map[string]interface{}) error
```

---

## Automatic Checkpoints

### Auto-Checkpoint System

```go
package checkpoint

import (
    "context"
    "time"
)

// AutoCheckpointer manages automatic checkpoint creation
type AutoCheckpointer struct {
    manager   *CheckpointManager
    config    *AutoConfig
    scheduler *Scheduler
    hooks     *HookManager
}

// AutoConfig configures automatic checkpointing
type AutoConfig struct {
    Enabled         bool
    Interval        time.Duration
    OnTaskStep      bool
    OnError         bool
    OnSignificantChange bool
    MinFileChanges  int           // Minimum files changed
    MinTimeElapsed  time.Duration // Minimum time since last checkpoint
    MaxPerTask      int           // Maximum checkpoints per task
}

// NewAutoCheckpointer creates an auto-checkpointer
func NewAutoCheckpointer(manager *CheckpointManager, config *AutoConfig) *AutoCheckpointer

// Start begins automatic checkpointing
func (a *AutoCheckpointer) Start(ctx context.Context) error

// Stop halts automatic checkpointing
func (a *AutoCheckpointer) Stop()

// OnTaskStep is called after each task step
func (a *AutoCheckpointer) OnTaskStep(ctx context.Context, taskID string, step int) error

// OnError is called when an error occurs
func (a *AutoCheckpointer) OnError(ctx context.Context, err error) error

// OnSignificantChange is called when significant changes detected
func (a *AutoCheckpointer) OnSignificantChange(ctx context.Context) error
```

### Hook System

```go
package checkpoint

// HookManager manages checkpoint lifecycle hooks
type HookManager struct {
    beforeCreate []BeforeCreateHook
    afterCreate  []AfterCreateHook
    beforeRestore []BeforeRestoreHook
    afterRestore []AfterRestoreHook
}

// BeforeCreateHook is called before checkpoint creation
type BeforeCreateHook func(ctx context.Context, opts *CreateOptions) error

// AfterCreateHook is called after checkpoint creation
type AfterCreateHook func(ctx context.Context, checkpoint *Checkpoint) error

// BeforeRestoreHook is called before restoration
type BeforeRestoreHook func(ctx context.Context, checkpoint *Checkpoint) error

// AfterRestoreHook is called after restoration
type AfterRestoreHook func(ctx context.Context, result *RestoreResult) error

// RegisterBeforeCreate registers a before-create hook
func (h *HookManager) RegisterBeforeCreate(hook BeforeCreateHook)

// RegisterAfterCreate registers an after-create hook
func (h *HookManager) RegisterAfterCreate(hook AfterCreateHook)
```

---

## Configuration Schema

### YAML Configuration

```yaml
checkpoint:
  # Storage settings
  storage:
    branch_prefix: "checkpoint/"
    metadata_path: ".helix/checkpoints"
    max_checkpoints: 50
    auto_cleanup: true
    compress_metadata: true

  # Automatic checkpoint settings
  automatic:
    enabled: true
    interval: 0  # 0 disables interval-based
    on_task_step: true
    on_error: true
    on_significant_change: true
    min_file_changes: 3
    min_time_elapsed: 5m
    max_per_task: 10

  # Snapshot settings
  snapshot:
    include_untracked: false
    exclude_patterns:
      - "*.log"
      - "node_modules/"
      - ".venv/"
      - "__pycache__/"
      - "*.pyc"
      - ".DS_Store"

  # Restore settings
  restore:
    create_backup: true
    allow_dirty_restore: false
    default_resolution: "keep_current"

  # Pruning policy
  pruning:
    enabled: true
    keep_last_n: 20
    keep_daily: 7   # Keep one per day for 7 days
    keep_weekly: 4  # Keep one per week for 4 weeks
    keep_monthly: 6 # Keep one per month for 6 months
    remove_after: 90d  # Remove checkpoints older than 90 days
```

---

## Error Handling

### Error Types

```go
package checkpoint

import "errors"

var (
    // Creation errors
    ErrDirtyWorkspace     = errors.New("workspace has uncommitted changes")
    ErrNotGitRepo         = errors.New("not a git repository")
    ErrCreateFailed       = errors.New("failed to create checkpoint")
    ErrBranchExists       = errors.New("checkpoint branch already exists")

    // Retrieval errors
    ErrCheckpointNotFound = errors.New("checkpoint not found")
    ErrInvalidCheckpointID = errors.New("invalid checkpoint ID")
    ErrMetadataCorrupted  = errors.New("checkpoint metadata is corrupted")

    // Comparison errors
    ErrInvalidComparison  = errors.New("invalid checkpoint comparison")
    ErrDiffFailed         = errors.New("failed to generate diff")

    // Restore errors
    ErrRestoreFailed      = errors.New("failed to restore checkpoint")
    ErrConflictDetected   = errors.New("conflicts detected")
    ErrBackupFailed       = errors.New("failed to create backup")
    ErrValidationFailed   = errors.New("restore validation failed")

    // Cleanup errors
    ErrDeleteFailed       = errors.New("failed to delete checkpoint")
    ErrPruneFailed        = errors.New("failed to prune checkpoints")
)

// CheckpointError provides detailed error information
type CheckpointError struct {
    Op           string   // Operation that failed
    CheckpointID string   // Related checkpoint ID
    Err          error    // Underlying error
    Details      string   // Additional details
    Recoverable  bool     // Whether error is recoverable
}

func (e *CheckpointError) Error() string {
    return fmt.Sprintf("%s (checkpoint: %s): %v - %s",
        e.Op, e.CheckpointID, e.Err, e.Details)
}

func (e *CheckpointError) Unwrap() error {
    return e.Err
}
```

---

## Testing Strategy

### Unit Tests

```go
package checkpoint_test

import (
    "context"
    "testing"
    "time"

    "github.com/yourusername/helix/internal/checkpoint"
)

// TestCheckpointCreation tests checkpoint creation
func TestCheckpointCreation(t *testing.T) {
    ctx := context.Background()

    // Setup test repository
    repo := setupTestRepo(t)
    defer cleanupTestRepo(t, repo)

    config := &checkpoint.Config{
        BranchPrefix: "test-checkpoint/",
        MetadataPath: ".test/checkpoints",
    }

    manager, err := checkpoint.NewCheckpointManager(repo, config)
    if err != nil {
        t.Fatalf("NewCheckpointManager() error = %v", err)
    }

    // Create test files
    createTestFiles(t, repo, []string{"file1.txt", "file2.txt"})

    opts := &checkpoint.CreateOptions{
        Description: "Test checkpoint",
        Tags:        []string{"test"},
    }

    cp, err := manager.CreateCheckpoint(ctx, opts)
    if err != nil {
        t.Fatalf("CreateCheckpoint() error = %v", err)
    }

    if cp.ID == "" {
        t.Error("expected non-empty checkpoint ID")
    }

    if cp.FileCount != 2 {
        t.Errorf("expected file count 2, got %d", cp.FileCount)
    }
}

// TestCheckpointComparison tests checkpoint comparison
func TestCheckpointComparison(t *testing.T) {
    ctx := context.Background()
    manager := setupTestManager(t)

    // Create first checkpoint
    cp1, _ := manager.CreateCheckpoint(ctx, &checkpoint.CreateOptions{
        Description: "First checkpoint",
    })

    // Make changes
    modifyTestFiles(t, manager.Repository())

    // Create second checkpoint
    cp2, _ := manager.CreateCheckpoint(ctx, &checkpoint.CreateOptions{
        Description: "Second checkpoint",
    })

    // Compare checkpoints
    comparison, err := manager.CompareCheckpoints(ctx, cp1.ID, cp2.ID)
    if err != nil {
        t.Fatalf("CompareCheckpoints() error = %v", err)
    }

    if comparison.Summary.FilesModified == 0 {
        t.Error("expected modified files")
    }

    if len(comparison.FileDiffs) == 0 {
        t.Error("expected file diffs")
    }
}

// TestCheckpointRestore tests checkpoint restoration
func TestCheckpointRestore(t *testing.T) {
    ctx := context.Background()
    manager := setupTestManager(t)

    // Create checkpoint
    cp, _ := manager.CreateCheckpoint(ctx, &checkpoint.CreateOptions{
        Description: "Before changes",
    })

    // Make changes
    originalContent := readTestFile(t, "test.txt")
    writeTestFile(t, "test.txt", "modified content")

    // Restore checkpoint
    opts := &checkpoint.RestoreOptions{
        CreateBackup: true,
        DryRun:       false,
    }

    result, err := manager.RestoreCheckpoint(ctx, cp.ID, opts)
    if err != nil {
        t.Fatalf("RestoreCheckpoint() error = %v", err)
    }

    if !result.Success {
        t.Error("expected successful restore")
    }

    // Verify content restored
    restoredContent := readTestFile(t, "test.txt")
    if restoredContent != originalContent {
        t.Error("file content not restored correctly")
    }

    // Verify backup created
    if result.BackupCheckpoint == nil {
        t.Error("expected backup checkpoint")
    }
}

// TestAutomaticCheckpoints tests auto-checkpoint system
func TestAutomaticCheckpoints(t *testing.T) {
    ctx := context.Background()
    manager := setupTestManager(t)

    config := &checkpoint.AutoConfig{
        Enabled:     true,
        OnTaskStep:  true,
        OnError:     true,
        MinFileChanges: 2,
    }

    autoCP := checkpoint.NewAutoCheckpointer(manager, config)

    // Start auto-checkpointing
    if err := autoCP.Start(ctx); err != nil {
        t.Fatalf("Start() error = %v", err)
    }
    defer autoCP.Stop()

    // Simulate task step
    makeFileChanges(t, 3)
    if err := autoCP.OnTaskStep(ctx, "task_001", 1); err != nil {
        t.Fatalf("OnTaskStep() error = %v", err)
    }

    // Verify checkpoint created
    checkpoints, _ := manager.ListCheckpoints(ctx, &checkpoint.Filter{
        TaskID: "task_001",
    })

    if len(checkpoints) == 0 {
        t.Error("expected automatic checkpoint")
    }
}
```

### Integration Tests

```go
package checkpoint_test

// TestCheckpointLifecycle tests full checkpoint lifecycle
func TestCheckpointLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // 1. Create initial state
    repo := createRealRepo(t)
    defer cleanupRealRepo(t, repo)

    manager, _ := checkpoint.NewCheckpointManager(repo.Path(), nil)

    // 2. Create checkpoint
    cp1, err := manager.CreateCheckpoint(ctx, &checkpoint.CreateOptions{
        Description: "Initial state",
        Tags:        []string{"v1"},
    })
    if err != nil {
        t.Fatalf("CreateCheckpoint() error = %v", err)
    }

    // 3. Make significant changes
    makeRealChanges(t, repo)

    // 4. Create second checkpoint
    cp2, err := manager.CreateCheckpoint(ctx, &checkpoint.CreateOptions{
        Description: "After changes",
        Tags:        []string{"v2"},
    })
    if err != nil {
        t.Fatalf("CreateCheckpoint() error = %v", err)
    }

    // 5. Compare checkpoints
    comparison, err := manager.CompareCheckpoints(ctx, cp1.ID, cp2.ID)
    if err != nil {
        t.Fatalf("CompareCheckpoints() error = %v", err)
    }

    t.Logf("Comparison: %+v", comparison.Summary)

    // 6. Make more changes
    makeMoreChanges(t, repo)

    // 7. Restore to cp2
    result, err := manager.RestoreCheckpoint(ctx, cp2.ID, &checkpoint.RestoreOptions{
        CreateBackup: true,
    })
    if err != nil {
        t.Fatalf("RestoreCheckpoint() error = %v", err)
    }

    if !result.Success {
        t.Fatalf("Restore failed: %v", result.Errors)
    }

    // 8. Verify state
    verifyRepoState(t, repo, cp2)

    // 9. Restore to cp1
    _, err = manager.RestoreCheckpoint(ctx, cp1.ID, &checkpoint.RestoreOptions{
        CreateBackup: false,
    })
    if err != nil {
        t.Fatalf("RestoreCheckpoint() error = %v", err)
    }

    // 10. Verify final state
    verifyRepoState(t, repo, cp1)
}
```

---

## Performance Considerations

### Optimization Strategies

1. **Incremental Snapshots**
   - Use Git's object storage for deduplication
   - Only store changed files
   - Leverage Git's compression

2. **Metadata Caching**
   - Cache checkpoint list in memory
   - Invalidate on create/delete
   - Use file system watchers for updates

3. **Lazy Loading**
   - Load metadata on demand
   - Stream large diffs
   - Paginate checkpoint lists

4. **Parallel Operations**
   - Concurrent checkpoint creation
   - Parallel diff generation
   - Async metadata updates

### Performance Metrics

```go
// Metrics tracks checkpoint system performance
type Metrics struct {
    CheckpointsCreated   atomic.Int64
    CheckpointsRestored  atomic.Int64
    CheckpointsDeleted   atomic.Int64
    ComparisonsPerformed atomic.Int64

    AverageCreateTime    atomic.Int64 // milliseconds
    AverageRestoreTime   atomic.Int64 // milliseconds
    AverageCompareTime   atomic.Int64 // milliseconds

    TotalStorageUsed     atomic.Int64 // bytes
    MetadataSize         atomic.Int64 // bytes
}

func (m *Metrics) RecordCreate(duration time.Duration, size int64) {
    m.CheckpointsCreated.Add(1)
    current := m.AverageCreateTime.Load()
    newAvg := (current*9 + duration.Milliseconds()) / 10
    m.AverageCreateTime.Store(newAvg)
    m.TotalStorageUsed.Add(size)
}
```

---

## User Experience Flow

### CLI Interface

```bash
# Create checkpoint
$ helix checkpoint create --description "Before refactoring"
Creating checkpoint... Done!
Checkpoint ID: cp_20250105_143022_a1b2c3
Description: Before refactoring
Files: 42 files, 1.2 MB

# List checkpoints
$ helix checkpoint list
ID                          Created              Description           Files
cp_20250105_143022_a1b2c3  2025-01-05 14:30:22  Before refactoring    42
cp_20250105_120000_xyz789  2025-01-05 12:00:00  Initial setup         25
cp_20250104_180000_abc123  2025-01-04 18:00:00  Feature complete      38

# Compare checkpoints
$ helix checkpoint compare cp_20250105_120000_xyz789 cp_20250105_143022_a1b2c3
Comparing checkpoints...

Summary:
  Files added: 5
  Files modified: 12
  Files deleted: 0
  Lines added: 247
  Lines deleted: 83
  Time elapsed: 2h 30m

Top changes:
  src/auth/handler.go    +89 -23
  src/db/queries.go      +45 -12
  tests/auth_test.go     +113 -48

# Show detailed diff
$ helix checkpoint diff cp_20250105_120000_xyz789 cp_20250105_143022_a1b2c3
[Shows unified diff output]

# Restore checkpoint
$ helix checkpoint restore cp_20250105_120000_xyz789
Warning: This will restore your workspace to the checkpoint state.
Current changes will be backed up automatically.

Proceed? (y/N): y

Creating backup... Done! (backup ID: cp_20250105_143500_backup)
Restoring checkpoint... Done!
Files restored: 42
Time elapsed: 2.3s

# Delete checkpoint
$ helix checkpoint delete cp_20250104_180000_abc123
Delete checkpoint 'Feature complete'? (y/N): y
Deleted checkpoint cp_20250104_180000_abc123

# Automatic checkpointing
$ helix checkpoint auto enable --on-task-step
Automatic checkpointing enabled
  • On task step: enabled
  • On error: disabled
  • Interval: disabled
```

### TUI Interface

```
┌──────────────────────────────────────────────────────────────┐
│ Checkpoints                                                  │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│ ● cp_20250105_143022 - Before refactoring    2h ago        │
│   42 files, 1.2 MB                           [Compare] [⟳] │
│                                                              │
│ ○ cp_20250105_120000 - Initial setup         4h ago        │
│   25 files, 856 KB                           [Compare] [⟳] │
│                                                              │
│ ○ cp_20250104_180000 - Feature complete      22h ago       │
│   38 files, 1.1 MB                           [Compare] [⟳] │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ [C]reate  [D]elete  [A]uto Config  [Q]uit                  │
└──────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 1: Core Infrastructure (Week 1-2)
- [ ] Repository wrapper and Git operations
- [ ] Basic checkpoint creation and storage
- [ ] Metadata schema and storage
- [ ] Unit tests for core components

### Phase 2: Comparison System (Week 2-3)
- [ ] Diff generation
- [ ] Comparison statistics
- [ ] Conflict detection
- [ ] Integration tests

### Phase 3: Restore System (Week 3-4)
- [ ] Restore validation
- [ ] Backup creation
- [ ] Conflict resolution
- [ ] Rollback mechanism

### Phase 4: Automatic Checkpoints (Week 4-5)
- [ ] Hook system
- [ ] Auto-checkpoint triggers
- [ ] Pruning policies
- [ ] Configuration management

### Phase 5: CLI/TUI Integration (Week 5-6)
- [ ] CLI commands
- [ ] TUI interface
- [ ] User documentation
- [ ] End-to-end testing

---

## Security Considerations

1. **Sensitive Data Protection**
   - Respect .gitignore patterns
   - Exclude credential files
   - Sanitize metadata
   - Support encryption

2. **Access Control**
   - Validate checkpoint operations
   - Prevent unauthorized restores
   - Audit checkpoint access
   - Secure metadata storage

3. **Data Integrity**
   - Verify checkpoint consistency
   - Checksum validation
   - Detect corruption
   - Atomic operations

---

## References

- **Cline's checkpoint system**: Branch-based snapshots with metadata
- **Gemini CLI checkpointing**: Task-aware checkpoint management
- **Git worktree**: Alternative isolation approach
- **go-git**: Pure Go git implementation
