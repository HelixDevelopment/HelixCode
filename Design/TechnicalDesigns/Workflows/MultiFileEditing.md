# Multi-File Atomic Editing - Technical Design

## Overview

Multi-file atomic editing enables making changes across multiple files as a single transactional operation with all-or-nothing semantics. This design ensures consistency, provides rollback capabilities, and includes conflict detection and resolution.

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    EditCoordinator                           │
│  - Orchestrates multi-file editing operations                │
│  - Manages transaction lifecycle                             │
└────────────┬───────────────────────────────┬────────────────┘
             │                               │
             ▼                               ▼
┌────────────────────────┐      ┌──────────────────────────┐
│   TransactionManager   │      │      PreviewEngine       │
│  - Begin/Commit/Abort  │      │  - Generate diffs        │
│  - Rollback logic      │      │  - Preview changes       │
│  - State tracking      │      │  - Conflict detection    │
└──────────┬─────────────┘      └────────────┬─────────────┘
           │                                  │
           ├──────────────┬───────────────────┤
           ▼              ▼                   ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────────┐
│BackupManager │  │ DiffManager  │  │ConflictResolver  │
│- Save states │  │- Unified dif │  │- Detection       │
│- Restore     │  │- Apply patch │  │- Resolution      │
└──────────────┘  └──────────────┘  └──────────────────┘
           │              │                   │
           └──────────────┴───────────────────┘
                          ▼
                  ┌──────────────┐
                  │  FileSystem  │
                  │   Adapter    │
                  └──────────────┘
```

### Core Components

#### 1. EditCoordinator

```go
package multiedit

import (
    "context"
    "time"
)

// EditCoordinator orchestrates multi-file editing operations
type EditCoordinator struct {
    txManager      *TransactionManager
    previewEngine  *PreviewEngine
    backupManager  *BackupManager
    conflictResolver *ConflictResolver
    gitIntegration *GitIntegration
}

// NewEditCoordinator creates a new coordinator
func NewEditCoordinator(opts ...Option) *EditCoordinator {
    ec := &EditCoordinator{
        txManager:      NewTransactionManager(),
        previewEngine:  NewPreviewEngine(),
        backupManager:  NewBackupManager(),
        conflictResolver: NewConflictResolver(),
        gitIntegration: NewGitIntegration(),
    }

    for _, opt := range opts {
        opt(ec)
    }

    return ec
}

// BeginEdit starts a new multi-file edit transaction
func (ec *EditCoordinator) BeginEdit(ctx context.Context, opts EditOptions) (*EditTransaction, error)

// Preview generates a preview of changes without applying them
func (ec *EditCoordinator) Preview(ctx context.Context, tx *EditTransaction) (*PreviewResult, error)

// Commit applies all changes atomically
func (ec *EditCoordinator) Commit(ctx context.Context, tx *EditTransaction) error

// Rollback reverts all changes in the transaction
func (ec *EditCoordinator) Rollback(ctx context.Context, tx *EditTransaction) error
```

#### 2. TransactionManager

```go
// TransactionManager handles transaction lifecycle
type TransactionManager struct {
    mu           sync.RWMutex
    transactions map[string]*EditTransaction
    maxDuration  time.Duration
}

// EditTransaction represents a multi-file edit operation
type EditTransaction struct {
    ID          string
    State       TransactionState
    Files       []*FileEdit
    CreatedAt   time.Time
    UpdatedAt   time.Time
    Options     EditOptions
    Metadata    map[string]interface{}

    mu          sync.RWMutex
    backupPaths map[string]string // file path -> backup path
}

// TransactionState represents the current state
type TransactionState int

const (
    StatePending TransactionState = iota
    StatePreview
    StateReady
    StateCommitting
    StateCommitted
    StateRollingBack
    StateRolledBack
    StateAborted
    StateFailed
)

// FileEdit represents a single file edit operation
type FileEdit struct {
    FilePath    string
    Operation   EditOperation
    OldContent  []byte
    NewContent  []byte
    Checksum    string // SHA256 of original content
    Applied     bool
    Error       error
}

// EditOperation type
type EditOperation int

const (
    OpCreate EditOperation = iota
    OpUpdate
    OpDelete
    OpRename
)

// EditOptions configures edit behavior
type EditOptions struct {
    DryRun          bool
    ConflictPolicy  ConflictPolicy
    BackupEnabled   bool
    GitAware        bool
    PreCommitHooks  bool
    MaxFileSize     int64
    AllowedPaths    []string
    DeniedPaths     []string
}

// ConflictPolicy defines how to handle conflicts
type ConflictPolicy int

const (
    ConflictPolicyAbort ConflictPolicy = iota
    ConflictPolicySkip
    ConflictPolicyOverwrite
    ConflictPolicyAsk
)
```

### State Machine

```
┌─────────┐
│ Pending │
└────┬────┘
     │ Preview()
     ▼
┌─────────┐
│ Preview │◄─────────┐
└────┬────┘          │
     │ Validate()    │ Modify()
     ▼               │
┌─────────┐          │
│  Ready  ├──────────┘
└────┬────┘
     │ Commit()
     ▼
┌────────────┐
│ Committing │
└─────┬──────┘
      │
      ├──Success──┐
      │           ▼
      │      ┌───────────┐
      │      │ Committed │
      │      └───────────┘
      │
      └──Error────┐
                  ▼
            ┌──────────────┐
            │ RollingBack  │
            └──────┬───────┘
                   │
                   ▼
            ┌──────────────┐
            │ RolledBack   │
            └──────────────┘

Any state can transition to:
┌─────────┐
│ Aborted │  (on Cancel/Timeout)
└─────────┘
```

#### 3. BackupManager

```go
// BackupManager handles file backups and restoration
type BackupManager struct {
    backupDir string
    retention time.Duration
}

// Backup creates a backup of a file
func (bm *BackupManager) Backup(ctx context.Context, filePath string) (backupPath string, err error) {
    // Read original file
    content, err := os.ReadFile(filePath)
    if err != nil {
        return "", fmt.Errorf("read file: %w", err)
    }

    // Generate backup path with timestamp
    backupPath = bm.generateBackupPath(filePath)

    // Write backup with metadata
    metadata := BackupMetadata{
        OriginalPath: filePath,
        BackupTime:   time.Now(),
        Checksum:     sha256.Sum256(content),
        FileMode:     getFileMode(filePath),
    }

    if err := bm.writeBackupWithMetadata(backupPath, content, metadata); err != nil {
        return "", fmt.Errorf("write backup: %w", err)
    }

    return backupPath, nil
}

// Restore restores a file from backup
func (bm *BackupManager) Restore(ctx context.Context, backupPath, targetPath string) error

// Cleanup removes old backups
func (bm *BackupManager) Cleanup(ctx context.Context) error

// BackupMetadata stores backup information
type BackupMetadata struct {
    OriginalPath string
    BackupTime   time.Time
    Checksum     [32]byte
    FileMode     os.FileMode
    GitRef       string // Git commit if available
}
```

#### 4. DiffManager

```go
// DiffManager generates and applies diffs
type DiffManager struct {
    format DiffFormat
}

// DiffFormat specifies the diff format
type DiffFormat int

const (
    FormatUnified DiffFormat = iota
    FormatContext
    FormatWholeLine
    FormatSearchReplace
)

// GenerateDiff creates a unified diff
func (dm *DiffManager) GenerateDiff(oldContent, newContent []byte, filePath string) (*Diff, error) {
    // Use go-diff library for unified diff
    edits := myers.ComputeEdits(string(oldContent), string(newContent))
    unified := gotextdiff.ToUnified(filePath, filePath+".new", string(oldContent), edits)

    return &Diff{
        FilePath:   filePath,
        OldContent: oldContent,
        NewContent: newContent,
        Unified:    unified,
        Hunks:      dm.parseHunks(unified),
    }, nil
}

// ApplyDiff applies a diff to a file
func (dm *DiffManager) ApplyDiff(diff *Diff) error

// Diff represents file differences
type Diff struct {
    FilePath   string
    OldContent []byte
    NewContent []byte
    Unified    string
    Hunks      []*DiffHunk
}

// DiffHunk represents a single diff hunk
type DiffHunk struct {
    OldStart int
    OldLines int
    NewStart int
    NewLines int
    Lines    []DiffLine
}

// DiffLine represents a single line in a diff
type DiffLine struct {
    Type    LineType // Add, Delete, Context
    Content string
    LineNo  int
}

type LineType int

const (
    LineContext LineType = iota
    LineAdd
    LineDelete
)
```

#### 5. ConflictResolver

```go
// ConflictResolver detects and resolves conflicts
type ConflictResolver struct {
    gitIntegration *GitIntegration
}

// DetectConflicts checks for conflicts before applying edits
func (cr *ConflictResolver) DetectConflicts(ctx context.Context, tx *EditTransaction) ([]*Conflict, error) {
    var conflicts []*Conflict

    for _, edit := range tx.Files {
        // Check if file was modified since transaction started
        conflict, err := cr.detectFileConflict(ctx, edit)
        if err != nil {
            return nil, err
        }
        if conflict != nil {
            conflicts = append(conflicts, conflict)
        }
    }

    return conflicts, nil
}

// Resolve attempts to resolve conflicts
func (cr *ConflictResolver) Resolve(ctx context.Context, conflict *Conflict, strategy ConflictStrategy) error

// Conflict represents a detected conflict
type Conflict struct {
    Type        ConflictType
    FilePath    string
    Expected    string // Expected checksum
    Actual      string // Actual checksum
    Description string
    Resolution  *ConflictResolution
}

// ConflictType categorizes conflicts
type ConflictType int

const (
    ConflictModified ConflictType = iota
    ConflictDeleted
    ConflictMoved
    ConflictPermissions
)

// ConflictStrategy defines resolution approach
type ConflictStrategy int

const (
    StrategyAbort ConflictStrategy = iota
    StrategyTheirs
    StrategyOurs
    StrategyManual
)

// ConflictResolution stores resolution result
type ConflictResolution struct {
    Strategy    ConflictStrategy
    ResolvedBy  string
    Resolution  string
    Timestamp   time.Time
}
```

#### 6. PreviewEngine

```go
// PreviewEngine generates previews of changes
type PreviewEngine struct {
    diffManager *DiffManager
    formatter   *PreviewFormatter
}

// Preview generates a preview of all changes
func (pe *PreviewEngine) Preview(ctx context.Context, tx *EditTransaction) (*PreviewResult, error) {
    result := &PreviewResult{
        TransactionID: tx.ID,
        Files:         make([]*FilePreview, 0, len(tx.Files)),
    }

    for _, edit := range tx.Files {
        preview, err := pe.previewFile(ctx, edit)
        if err != nil {
            return nil, fmt.Errorf("preview file %s: %w", edit.FilePath, err)
        }
        result.Files = append(result.Files, preview)
    }

    result.Summary = pe.generateSummary(result)
    return result, nil
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
    FilePath   string
    Operation  EditOperation
    Diff       *Diff
    Stats      FileStats
    Status     PreviewStatus
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

// PreviewSummary summarizes all changes
type PreviewSummary struct {
    TotalFiles      int
    FilesCreated    int
    FilesModified   int
    FilesDeleted    int
    TotalLinesAdded int
    TotalLinesDeleted int
    HasConflicts    bool
}

// PreviewFormatter formats preview output
type PreviewFormatter struct {
    format OutputFormat
}

type OutputFormat int

const (
    FormatPlain OutputFormat = iota
    FormatMarkdown
    FormatJSON
    FormatHTML
)
```

### Git Integration

```go
// GitIntegration provides git-aware operations
type GitIntegration struct {
    repo *git.Repository
}

// IsGitRepo checks if directory is a git repository
func (gi *GitIntegration) IsGitRepo(path string) bool

// GetFileStatus gets git status of a file
func (gi *GitIntegration) GetFileStatus(filePath string) (GitStatus, error)

// StageFiles stages modified files
func (gi *GitIntegration) StageFiles(files []string) error

// CheckUncommittedChanges verifies no uncommitted changes exist
func (gi *GitIntegration) CheckUncommittedChanges(files []string) ([]string, error) {
    var modified []string

    worktree, err := gi.repo.Worktree()
    if err != nil {
        return nil, err
    }

    status, err := worktree.Status()
    if err != nil {
        return nil, err
    }

    for _, file := range files {
        if fileStatus := status.File(file); fileStatus.Worktree != git.Unmodified {
            modified = append(modified, file)
        }
    }

    return modified, nil
}

// GitStatus represents git file status
type GitStatus int

const (
    StatusUntracked GitStatus = iota
    StatusModified
    StatusAdded
    StatusDeleted
    StatusRenamed
    StatusUnmodified
)
```

## Data Flow

### Commit Flow

```
User Request
     │
     ▼
┌─────────────────┐
│ BeginEdit()     │
│ - Create TX     │
│ - Validate opts │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Add FileEdits   │
│ - Validate path │
│ - Create backup │
│ - Calculate chk │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Preview()       │
│ - Generate diff │
│ - Check conflict│
│ - Show preview  │
└────────┬────────┘
         │
         ▼
    User Review
         │
         ▼
┌─────────────────┐
│ Commit()        │
└────────┬────────┘
         │
         ├──────────────────┐
         │                  │
         ▼                  ▼
┌─────────────────┐  ┌──────────────┐
│ Verify Files    │  │ Lock Files   │
│ - Check exists  │  │ - Prevent    │
│ - Verify chksum │  │   concurrent │
└────────┬────────┘  └──────┬───────┘
         │                  │
         └──────┬───────────┘
                ▼
        ┌──────────────┐
        │ Apply Edits  │
        │ - Write files│
        │ - Track prog │
        └──────┬───────┘
               │
               ├──Success──┐
               │           ▼
               │    ┌─────────────┐
               │    │ Cleanup     │
               │    │ - Remove bak│
               │    │ - Unlock    │
               │    └─────────────┘
               │
               └──Error───┐
                          ▼
                   ┌─────────────┐
                   │ Rollback    │
                   │ - Restore   │
                   │ - Log error │
                   └─────────────┘
```

### Rollback Flow

```
Rollback Triggered
     │
     ▼
┌─────────────────┐
│ Identify Applied│
│ - Check TX state│
│ - List applied  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Restore Backups │
│ - Reverse order │
│ - Verify chksum │
└────────┬────────┘
         │
         ├──Success──┐
         │           ▼
         │    ┌─────────────┐
         │    │ Verify State│
         │    │ - Check all │
         │    │ - Clean bak │
         │    └─────────────┘
         │
         └──Error───┐
                    ▼
             ┌─────────────┐
             │ Partial Fail│
             │ - Log state │
             │ - Manual fix│
             └─────────────┘
```

## Interface Definitions

```go
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
    ApplyDiff(diff *Diff) error
    ParseDiff(diffText string) (*Diff, error)
}

// ConflictDetector detects conflicts
type ConflictDetector interface {
    DetectConflicts(ctx context.Context, tx *EditTransaction) ([]*Conflict, error)
    Resolve(ctx context.Context, conflict *Conflict, strategy ConflictStrategy) error
}
```

## Error Handling

```go
// Error types
var (
    ErrTransactionNotFound   = errors.New("transaction not found")
    ErrInvalidState          = errors.New("invalid transaction state")
    ErrConflictDetected      = errors.New("conflict detected")
    ErrBackupFailed          = errors.New("backup failed")
    ErrRollbackFailed        = errors.New("rollback failed")
    ErrFileNotFound          = errors.New("file not found")
    ErrChecksumMismatch      = errors.New("checksum mismatch")
    ErrPermissionDenied      = errors.New("permission denied")
    ErrTransactionTimeout    = errors.New("transaction timeout")
    ErrInvalidPath           = errors.New("invalid file path")
)

// EditError wraps errors with context
type EditError struct {
    Op       string
    FilePath string
    Err      error
    Code     ErrorCode
}

func (e *EditError) Error() string {
    return fmt.Sprintf("%s: %s: %v", e.Op, e.FilePath, e.Err)
}

func (e *EditError) Unwrap() error {
    return e.Err
}

// ErrorCode categorizes errors
type ErrorCode int

const (
    CodeUnknown ErrorCode = iota
    CodeValidation
    CodeConflict
    CodeFileSystem
    CodeTransaction
    CodeBackup
    CodeRollback
)

// Recovery strategies
func (ec *EditCoordinator) handleError(ctx context.Context, tx *EditTransaction, err error) error {
    switch {
    case errors.Is(err, ErrConflictDetected):
        // Try automatic resolution
        return ec.resolveConflictsAuto(ctx, tx)

    case errors.Is(err, ErrChecksumMismatch):
        // Abort transaction
        return ec.Rollback(ctx, tx)

    case errors.Is(err, ErrFileNotFound):
        // Skip file and continue
        return ec.skipFile(ctx, tx, err)

    default:
        // Rollback on unknown errors
        return ec.Rollback(ctx, tx)
    }
}
```

## Configuration Schema

```yaml
# multi_file_editing.yaml

editing:
  # Transaction settings
  transaction:
    max_duration: 1h
    max_files: 1000
    max_file_size: 10MB
    timeout: 30m

  # Backup settings
  backup:
    enabled: true
    dir: .helix/backups
    retention: 7d
    compression: true

  # Conflict resolution
  conflicts:
    policy: ask  # ask, abort, skip, overwrite
    auto_resolve: false
    detect_git_changes: true

  # Preview settings
  preview:
    format: unified  # unified, context, side-by-side
    context_lines: 3
    syntax_highlight: true
    show_line_numbers: true

  # Git integration
  git:
    enabled: true
    auto_stage: false
    check_uncommitted: true
    respect_gitignore: true

  # Safety settings
  safety:
    require_preview: true
    allowed_paths:
      - "**/*.go"
      - "**/*.md"
      - "**/config/**"
    denied_paths:
      - "**/.git/**"
      - "**/node_modules/**"
      - "**/vendor/**"
    max_retries: 3

  # Performance
  performance:
    parallel_writes: 4
    buffer_size: 4096
    use_memory_cache: true
```

```go
// Config represents configuration
type Config struct {
    Transaction TransactionConfig `yaml:"transaction"`
    Backup      BackupConfig      `yaml:"backup"`
    Conflicts   ConflictConfig    `yaml:"conflicts"`
    Preview     PreviewConfig     `yaml:"preview"`
    Git         GitConfig         `yaml:"git"`
    Safety      SafetyConfig      `yaml:"safety"`
    Performance PerformanceConfig `yaml:"performance"`
}

// Load configuration
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    return &cfg, cfg.Validate()
}
```

## Testing Strategy

### Unit Tests

```go
func TestTransactionManager_Lifecycle(t *testing.T) {
    tm := NewTransactionManager()

    // Begin transaction
    tx, err := tm.Begin(context.Background(), EditOptions{})
    require.NoError(t, err)
    assert.Equal(t, StatePending, tx.State)

    // Add file edits
    edit := &FileEdit{
        FilePath:   "/tmp/test.txt",
        Operation:  OpUpdate,
        OldContent: []byte("old"),
        NewContent: []byte("new"),
    }
    err = tm.AddEdit(tx, edit)
    require.NoError(t, err)

    // Commit
    err = tm.Commit(context.Background(), tx)
    require.NoError(t, err)
    assert.Equal(t, StateCommitted, tx.State)
}

func TestBackupManager_BackupRestore(t *testing.T) {
    bm := NewBackupManager(t.TempDir(), 24*time.Hour)

    // Create test file
    testFile := filepath.Join(t.TempDir(), "test.txt")
    content := []byte("test content")
    err := os.WriteFile(testFile, content, 0644)
    require.NoError(t, err)

    // Backup
    backupPath, err := bm.Backup(context.Background(), testFile)
    require.NoError(t, err)
    assert.FileExists(t, backupPath)

    // Modify original
    err = os.WriteFile(testFile, []byte("modified"), 0644)
    require.NoError(t, err)

    // Restore
    err = bm.Restore(context.Background(), backupPath, testFile)
    require.NoError(t, err)

    // Verify
    restored, err := os.ReadFile(testFile)
    require.NoError(t, err)
    assert.Equal(t, content, restored)
}

func TestConflictResolver_DetectModification(t *testing.T) {
    cr := NewConflictResolver(nil)

    // Create file with known content
    testFile := filepath.Join(t.TempDir(), "test.txt")
    original := []byte("original content")
    err := os.WriteFile(testFile, original, 0644)
    require.NoError(t, err)

    // Create edit with checksum
    edit := &FileEdit{
        FilePath:   testFile,
        OldContent: original,
        NewContent: []byte("new content"),
        Checksum:   fmt.Sprintf("%x", sha256.Sum256(original)),
    }

    // Modify file externally
    err = os.WriteFile(testFile, []byte("modified externally"), 0644)
    require.NoError(t, err)

    // Detect conflict
    tx := &EditTransaction{Files: []*FileEdit{edit}}
    conflicts, err := cr.DetectConflicts(context.Background(), tx)
    require.NoError(t, err)
    assert.Len(t, conflicts, 1)
    assert.Equal(t, ConflictModified, conflicts[0].Type)
}

func TestDiffManager_GenerateApply(t *testing.T) {
    dm := NewDiffManager()

    old := []byte("line 1\nline 2\nline 3\n")
    new := []byte("line 1\nmodified line 2\nline 3\n")

    // Generate diff
    diff, err := dm.GenerateDiff(old, new, "test.txt")
    require.NoError(t, err)
    assert.NotEmpty(t, diff.Unified)
    assert.Len(t, diff.Hunks, 1)

    // Apply diff
    result, err := dm.ApplyDiff(diff)
    require.NoError(t, err)
    assert.Equal(t, new, result)
}
```

### Integration Tests

```go
func TestEditCoordinator_MultiFileEdit(t *testing.T) {
    // Setup
    tmpDir := t.TempDir()
    ec := NewEditCoordinator(WithBackupDir(tmpDir))

    // Create test files
    file1 := filepath.Join(tmpDir, "file1.txt")
    file2 := filepath.Join(tmpDir, "file2.txt")
    os.WriteFile(file1, []byte("content 1"), 0644)
    os.WriteFile(file2, []byte("content 2"), 0644)

    // Begin transaction
    tx, err := ec.BeginEdit(context.Background(), EditOptions{
        BackupEnabled: true,
    })
    require.NoError(t, err)

    // Add edits
    tx.Files = []*FileEdit{
        {
            FilePath:   file1,
            Operation:  OpUpdate,
            OldContent: []byte("content 1"),
            NewContent: []byte("updated 1"),
        },
        {
            FilePath:   file2,
            Operation:  OpUpdate,
            OldContent: []byte("content 2"),
            NewContent: []byte("updated 2"),
        },
    }

    // Preview
    preview, err := ec.Preview(context.Background(), tx)
    require.NoError(t, err)
    assert.Len(t, preview.Files, 2)

    // Commit
    err = ec.Commit(context.Background(), tx)
    require.NoError(t, err)

    // Verify
    content1, _ := os.ReadFile(file1)
    content2, _ := os.ReadFile(file2)
    assert.Equal(t, []byte("updated 1"), content1)
    assert.Equal(t, []byte("updated 2"), content2)
}

func TestEditCoordinator_RollbackOnError(t *testing.T) {
    tmpDir := t.TempDir()
    ec := NewEditCoordinator(WithBackupDir(tmpDir))

    // Create test file
    file1 := filepath.Join(tmpDir, "file1.txt")
    file2 := filepath.Join(tmpDir, "file2.txt")
    os.WriteFile(file1, []byte("content 1"), 0644)
    // file2 doesn't exist - will cause error

    // Begin transaction
    tx, err := ec.BeginEdit(context.Background(), EditOptions{
        BackupEnabled: true,
    })
    require.NoError(t, err)

    tx.Files = []*FileEdit{
        {
            FilePath:   file1,
            Operation:  OpUpdate,
            OldContent: []byte("content 1"),
            NewContent: []byte("updated 1"),
        },
        {
            FilePath:   file2,
            Operation:  OpUpdate,
            OldContent: []byte("content 2"),
            NewContent: []byte("updated 2"),
        },
    }

    // Commit should fail and rollback
    err = ec.Commit(context.Background(), tx)
    assert.Error(t, err)

    // Verify file1 was rolled back
    content1, _ := os.ReadFile(file1)
    assert.Equal(t, []byte("content 1"), content1)
}
```

### Rollback Scenario Tests

```go
func TestRollbackScenarios(t *testing.T) {
    tests := []struct {
        name     string
        scenario func(*testing.T, *EditCoordinator, string)
    }{
        {
            name: "rollback single file",
            scenario: testRollbackSingleFile,
        },
        {
            name: "rollback multiple files",
            scenario: testRollbackMultipleFiles,
        },
        {
            name: "rollback with partial success",
            scenario: testRollbackPartialSuccess,
        },
        {
            name: "rollback after timeout",
            scenario: testRollbackTimeout,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tmpDir := t.TempDir()
            ec := NewEditCoordinator(WithBackupDir(tmpDir))
            tt.scenario(t, ec, tmpDir)
        })
    }
}

func testRollbackSingleFile(t *testing.T, ec *EditCoordinator, tmpDir string) {
    file := filepath.Join(tmpDir, "test.txt")
    original := []byte("original")
    os.WriteFile(file, original, 0644)

    tx, _ := ec.BeginEdit(context.Background(), EditOptions{BackupEnabled: true})
    tx.Files = []*FileEdit{{
        FilePath:   file,
        Operation:  OpUpdate,
        NewContent: []byte("modified"),
    }}

    // Simulate error during commit
    ec.Rollback(context.Background(), tx)

    content, _ := os.ReadFile(file)
    assert.Equal(t, original, content)
}
```

## Performance Considerations

### Optimization Strategies

1. **Parallel File Operations**
```go
func (ec *EditCoordinator) commitParallel(ctx context.Context, tx *EditTransaction) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(tx.Files))
    semaphore := make(chan struct{}, ec.maxParallel)

    for _, edit := range tx.Files {
        wg.Add(1)
        go func(e *FileEdit) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            if err := ec.applyEdit(ctx, e); err != nil {
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
```

2. **Memory Management**
```go
// Use memory pooling for large files
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func (dm *DiffManager) GenerateDiff(old, new []byte, path string) (*Diff, error) {
    // Use streaming for large files
    if len(old) > 10*1024*1024 { // 10MB
        return dm.generateDiffStreaming(old, new, path)
    }
    return dm.generateDiffInMemory(old, new, path)
}
```

3. **Caching**
```go
// Cache file checksums
type ChecksumCache struct {
    mu    sync.RWMutex
    cache map[string]cachedChecksum
}

type cachedChecksum struct {
    checksum string
    modTime  time.Time
}

func (cc *ChecksumCache) Get(path string, modTime time.Time) (string, bool) {
    cc.mu.RLock()
    defer cc.mu.RUnlock()

    if cached, ok := cc.cache[path]; ok && cached.modTime.Equal(modTime) {
        return cached.checksum, true
    }
    return "", false
}
```

### Performance Metrics

```go
// Metrics tracks performance
type Metrics struct {
    TransactionsTotal     prometheus.Counter
    TransactionDuration   prometheus.Histogram
    FilesProcessed        prometheus.Counter
    BackupDuration        prometheus.Histogram
    RollbacksTotal        prometheus.Counter
    ConflictsDetected     prometheus.Counter
}

func (ec *EditCoordinator) recordMetrics(tx *EditTransaction, duration time.Duration) {
    ec.metrics.TransactionsTotal.Inc()
    ec.metrics.TransactionDuration.Observe(duration.Seconds())
    ec.metrics.FilesProcessed.Add(float64(len(tx.Files)))
}
```

## Security Considerations

### Path Validation

```go
// ValidatePath ensures path is safe
func ValidatePath(path string, allowedPaths, deniedPaths []string) error {
    // Normalize path
    normalized, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("normalize path: %w", err)
    }

    // Check for directory traversal
    if strings.Contains(normalized, "..") {
        return ErrInvalidPath
    }

    // Check against denied paths
    for _, denied := range deniedPaths {
        if matched, _ := filepath.Match(denied, normalized); matched {
            return ErrPermissionDenied
        }
    }

    // Check against allowed paths
    if len(allowedPaths) > 0 {
        allowed := false
        for _, pattern := range allowedPaths {
            if matched, _ := filepath.Match(pattern, normalized); matched {
                allowed = true
                break
            }
        }
        if !allowed {
            return ErrPermissionDenied
        }
    }

    return nil
}
```

### File Permissions

```go
// PreservePermissions maintains file permissions
func (bm *BackupManager) preservePermissions(src, dst string) error {
    info, err := os.Stat(src)
    if err != nil {
        return err
    }

    return os.Chmod(dst, info.Mode())
}
```

### Audit Logging

```go
// AuditLogger logs all operations
type AuditLogger struct {
    logger *slog.Logger
}

func (al *AuditLogger) LogTransaction(tx *EditTransaction, result string) {
    al.logger.Info("transaction completed",
        "tx_id", tx.ID,
        "files", len(tx.Files),
        "result", result,
        "duration", time.Since(tx.CreatedAt),
    )
}

func (al *AuditLogger) LogRollback(tx *EditTransaction, reason string) {
    al.logger.Warn("transaction rolled back",
        "tx_id", tx.ID,
        "reason", reason,
        "files_affected", len(tx.Files),
    )
}
```

## References

### Cline Multi-File Editing

- **Implementation**: `src/core/edit-manager.ts`
- **Features**:
  - Atomic multi-file edits with preview
  - Diff-based editing with search/replace
  - Transaction-based with rollback
  - Preview before commit

### Aider Edit Formats

- **Repository**: `aider/repo.py`, `aider/coders/editblock_coder.py`
- **Formats**:
  - Whole file editing
  - Search/replace blocks
  - Unified diff format
  - Edit blocks with markers

### Key Insights

1. **Atomic Operations**: All-or-nothing semantics prevent partial failures
2. **Preview Required**: Always show users what will change
3. **Conflict Detection**: Check file state before applying changes
4. **Git Awareness**: Integrate with git for better conflict detection
5. **Rollback Support**: Essential for recovery from errors

## Usage Examples

```go
// Example 1: Basic multi-file edit
func ExampleBasicEdit() {
    ec := NewEditCoordinator()

    // Begin transaction
    tx, _ := ec.BeginEdit(context.Background(), EditOptions{
        BackupEnabled: true,
    })

    // Add edits
    tx.Files = []*FileEdit{
        {
            FilePath:   "main.go",
            Operation:  OpUpdate,
            NewContent: []byte("package main\n..."),
        },
        {
            FilePath:   "config.yaml",
            Operation:  OpUpdate,
            NewContent: []byte("version: 2\n..."),
        },
    }

    // Preview changes
    preview, _ := ec.Preview(context.Background(), tx)
    fmt.Printf("Will modify %d files\n", preview.Summary.TotalFiles)

    // Commit
    ec.Commit(context.Background(), tx)
}

// Example 2: With conflict resolution
func ExampleConflictResolution() {
    ec := NewEditCoordinator()

    tx, _ := ec.BeginEdit(context.Background(), EditOptions{
        ConflictPolicy: ConflictPolicyAsk,
    })

    // Add edits...

    // Preview detects conflicts
    preview, _ := ec.Preview(context.Background(), tx)
    if preview.Summary.HasConflicts {
        for _, conflict := range preview.Conflicts {
            fmt.Printf("Conflict in %s: %s\n", conflict.FilePath, conflict.Description)
            // Resolve manually or with strategy
        }
    }
}
```

## Future Enhancements

1. **Merge Conflict Resolution**: Three-way merge support
2. **Distributed Transactions**: Multi-machine coordination
3. **Incremental Commits**: Commit files as they're ready
4. **Smart Retries**: Automatic retry with exponential backoff
5. **Change History**: Track all modifications with undo/redo
