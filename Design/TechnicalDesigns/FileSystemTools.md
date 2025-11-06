# FileSystemTools - Technical Design Document

## Overview

The FileSystemTools module provides secure, efficient, and user-friendly file system operations for HelixCode. This design is inspired by Cline's file tools and Qwen Code's file operations, with enhancements for security, performance, and error handling.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    FileSystemTools                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │  Reader  │  │  Writer  │  │  Editor  │  │ Searcher │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  │
│       │             │             │             │         │
│  ┌────┴─────────────┴─────────────┴─────────────┴─────┐  │
│  │              PathValidator                          │  │
│  └────┬─────────────┬─────────────┬─────────────┬─────┘  │
│       │             │             │             │         │
│  ┌────┴─────┐  ┌────┴─────┐  ┌────┴─────┐  ┌────┴─────┐  │
│  │Permission│  │   Cache  │  │   Lock   │  │  Logger  │  │
│  │ Checker  │  │  Manager │  │ Manager  │  │          │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
                   OS File System
```

## Core Interfaces

### FileReader Interface

```go
// FileReader provides methods for reading file contents
type FileReader interface {
    // Read reads the entire file content
    Read(ctx context.Context, path string) (*FileContent, error)

    // ReadLines reads specific lines from a file
    ReadLines(ctx context.Context, path string, start, end int) (*FileContent, error)

    // ReadWithLimit reads file with size limit
    ReadWithLimit(ctx context.Context, path string, maxBytes int64) (*FileContent, error)

    // GetInfo returns file metadata
    GetInfo(ctx context.Context, path string) (*FileInfo, error)

    // Exists checks if a file exists
    Exists(ctx context.Context, path string) (bool, error)
}

// FileContent represents the content and metadata of a file
type FileContent struct {
    Path         string
    Content      []byte
    Lines        []string
    TotalLines   int
    Size         int64
    ModTime      time.Time
    IsPartial    bool
    StartLine    int
    EndLine      int
    Encoding     string
    LineEndings  LineEndingType
}

// FileInfo contains file metadata
type FileInfo struct {
    Path         string
    Name         string
    Size         int64
    Mode         os.FileMode
    ModTime      time.Time
    IsDir        bool
    IsSymlink    bool
    SymlinkTarget string
    MimeType     string
    Encoding     string
    Checksum     string // SHA-256
}

// LineEndingType represents the type of line endings
type LineEndingType int

const (
    LineEndingUnknown LineEndingType = iota
    LineEndingLF      // Unix/Linux (\n)
    LineEndingCRLF    // Windows (\r\n)
    LineEndingCR      // Old Mac (\r)
)
```

### FileWriter Interface

```go
// FileWriter provides methods for writing file contents
type FileWriter interface {
    // Write writes content to a file, creating it if it doesn't exist
    Write(ctx context.Context, path string, content []byte) error

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
```

### FileEditor Interface

```go
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
    Type         EditType
    StartLine    int
    EndLine      int
    StartCol     int
    EndCol       int
    Content      string
    Pattern      string
    Replacement  string
    IsRegex      bool
}

// EditType represents the type of edit operation
type EditType int

const (
    EditInsert EditType = iota
    EditDelete
    EditReplace
    EditReplacePattern
)

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
```

### FileSearcher Interface

```go
// FileSearcher provides methods for searching files
type FileSearcher interface {
    // Search searches for files matching criteria
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)

    // SearchContent searches file contents for a pattern
    SearchContent(ctx context.Context, opts ContentSearchOptions) ([]ContentMatch, error)

    // Glob performs glob pattern matching
    Glob(ctx context.Context, pattern string) ([]string, error)

    // Walk walks a directory tree
    Walk(ctx context.Context, root string, fn WalkFunc) error
}

// SearchOptions configures file search
type SearchOptions struct {
    Root           string
    Pattern        string
    IncludePattern []string
    ExcludePattern []string
    MaxDepth       int
    FollowSymlinks bool
    IncludeDirs    bool
    IncludeHidden  bool
    MaxResults     int
    SortBy         SortType
}

// ContentSearchOptions configures content search
type ContentSearchOptions struct {
    Root           string
    Pattern        string
    IsRegex        bool
    CaseSensitive  bool
    IncludeFiles   []string
    ExcludeFiles   []string
    MaxMatches     int
    ContextLines   int
    MaxFileSize    int64
}

// SearchResult represents a file search result
type SearchResult struct {
    Path      string
    Name      string
    Size      int64
    ModTime   time.Time
    IsDir     bool
    Depth     int
    Match     string // What matched
}

// ContentMatch represents a content search match
type ContentMatch struct {
    Path         string
    LineNumber   int
    ColumnNumber int
    Line         string
    Match        string
    Context      []string // Surrounding lines
}

// WalkFunc is called for each file during directory walking
type WalkFunc func(path string, info FileInfo, err error) error

// SortType defines how to sort search results
type SortType int

const (
    SortByName SortType = iota
    SortBySize
    SortByModTime
    SortByDepth
)
```

## Data Structures

### Path Validation

```go
// PathValidator validates and normalizes file paths
type PathValidator struct {
    workspaceRoot string
    allowedPaths  []string
    blockedPaths  []string
    followSymlinks bool
}

// ValidationResult contains path validation results
type ValidationResult struct {
    IsValid       bool
    NormalizedPath string
    IsAbsolute    bool
    IsSymlink     bool
    ResolvedPath  string
    Error         error
}

// Validate validates a path
func (v *PathValidator) Validate(path string) (*ValidationResult, error) {
    result := &ValidationResult{}

    // Normalize path
    normalized, err := filepath.Abs(path)
    if err != nil {
        return nil, fmt.Errorf("failed to normalize path: %w", err)
    }
    result.NormalizedPath = normalized
    result.IsAbsolute = filepath.IsAbs(path)

    // Check for path traversal
    if strings.Contains(normalized, "..") {
        return nil, &SecurityError{
            Type:    "path_traversal",
            Message: "path traversal detected",
            Path:    path,
        }
    }

    // Check if within workspace
    if v.workspaceRoot != "" {
        rel, err := filepath.Rel(v.workspaceRoot, normalized)
        if err != nil || strings.HasPrefix(rel, "..") {
            return nil, &SecurityError{
                Type:    "outside_workspace",
                Message: "path is outside workspace",
                Path:    path,
            }
        }
    }

    // Check blocked paths
    for _, blocked := range v.blockedPaths {
        if strings.HasPrefix(normalized, blocked) {
            return nil, &SecurityError{
                Type:    "blocked_path",
                Message: "path is blocked",
                Path:    path,
            }
        }
    }

    // Check if symlink
    info, err := os.Lstat(normalized)
    if err == nil && info.Mode()&os.ModeSymlink != 0 {
        result.IsSymlink = true
        if v.followSymlinks {
            resolved, err := filepath.EvalSymlinks(normalized)
            if err != nil {
                return nil, fmt.Errorf("failed to resolve symlink: %w", err)
            }
            result.ResolvedPath = resolved
            // Validate resolved path recursively
            _, err = v.Validate(resolved)
            if err != nil {
                return nil, err
            }
        } else {
            return nil, &SecurityError{
                Type:    "symlink_not_allowed",
                Message: "symlinks are not allowed",
                Path:    path,
            }
        }
    }

    result.IsValid = true
    return result, nil
}
```

### Cache Manager

```go
// CacheManager manages file content caching
type CacheManager struct {
    cache     *lru.Cache
    ttl       time.Duration
    maxSize   int64
    stats     *CacheStats
}

// CacheEntry represents a cached file
type CacheEntry struct {
    Path      string
    Content   []byte
    ModTime   time.Time
    Size      int64
    ExpiresAt time.Time
    Checksum  string
}

// CacheStats tracks cache performance
type CacheStats struct {
    Hits       atomic.Int64
    Misses     atomic.Int64
    Evictions  atomic.Int64
    BytesCached atomic.Int64
}

// Get retrieves a file from cache
func (cm *CacheManager) Get(path string) (*CacheEntry, bool) {
    entry, ok := cm.cache.Get(path)
    if !ok {
        cm.stats.Misses.Add(1)
        return nil, false
    }

    cached := entry.(*CacheEntry)

    // Check if expired
    if time.Now().After(cached.ExpiresAt) {
        cm.cache.Remove(path)
        cm.stats.Misses.Add(1)
        return nil, false
    }

    // Verify file hasn't changed
    info, err := os.Stat(path)
    if err != nil || !info.ModTime().Equal(cached.ModTime) {
        cm.cache.Remove(path)
        cm.stats.Misses.Add(1)
        return nil, false
    }

    cm.stats.Hits.Add(1)
    return cached, true
}

// Set adds a file to cache
func (cm *CacheManager) Set(path string, content []byte, modTime time.Time) {
    entry := &CacheEntry{
        Path:      path,
        Content:   content,
        ModTime:   modTime,
        Size:      int64(len(content)),
        ExpiresAt: time.Now().Add(cm.ttl),
        Checksum:  fmt.Sprintf("%x", sha256.Sum256(content)),
    }

    cm.cache.Add(path, entry)
    cm.stats.BytesCached.Add(int64(len(content)))
}
```

### Lock Manager

```go
// LockManager manages file locks to prevent concurrent modifications
type LockManager struct {
    locks sync.Map
}

// FileLock represents a lock on a file
type FileLock struct {
    Path      string
    Owner     string
    AcquiredAt time.Time
    mu        sync.RWMutex
}

// Acquire acquires a lock on a file
func (lm *LockManager) Acquire(path, owner string, timeout time.Duration) (*FileLock, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    for {
        lock := &FileLock{
            Path:      path,
            Owner:     owner,
            AcquiredAt: time.Now(),
        }

        if _, loaded := lm.locks.LoadOrStore(path, lock); !loaded {
            return lock, nil
        }

        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("failed to acquire lock: timeout")
        case <-time.After(10 * time.Millisecond):
            // Retry
        }
    }
}

// Release releases a file lock
func (lm *LockManager) Release(lock *FileLock) {
    lm.locks.Delete(lock.Path)
}
```

## Error Handling

```go
// FileSystemError represents a file system error
type FileSystemError struct {
    Type    ErrorType
    Path    string
    Message string
    Cause   error
}

func (e *FileSystemError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *FileSystemError) Unwrap() error {
    return e.Cause
}

// ErrorType represents the type of error
type ErrorType string

const (
    ErrorFileNotFound     ErrorType = "file_not_found"
    ErrorPermissionDenied ErrorType = "permission_denied"
    ErrorInvalidPath      ErrorType = "invalid_path"
    ErrorFileExists       ErrorType = "file_exists"
    ErrorIsDirectory      ErrorType = "is_directory"
    ErrorNotDirectory     ErrorType = "not_directory"
    ErrorFileTooLarge     ErrorType = "file_too_large"
    ErrorInvalidEncoding  ErrorType = "invalid_encoding"
    ErrorDiskFull         ErrorType = "disk_full"
    ErrorTimeout          ErrorType = "timeout"
)

// SecurityError represents a security-related error
type SecurityError struct {
    Type    string
    Message string
    Path    string
}

func (e *SecurityError) Error() string {
    return fmt.Sprintf("security error [%s]: %s (path: %s)", e.Type, e.Message, e.Path)
}

// Common error constructors
func NewFileNotFoundError(path string) error {
    return &FileSystemError{
        Type:    ErrorFileNotFound,
        Path:    path,
        Message: "file not found",
    }
}

func NewPermissionDeniedError(path string, cause error) error {
    return &FileSystemError{
        Type:    ErrorPermissionDenied,
        Path:    path,
        Message: "permission denied",
        Cause:   cause,
    }
}

func NewFileTooLargeError(path string, size, limit int64) error {
    return &FileSystemError{
        Type:    ErrorFileTooLarge,
        Path:    path,
        Message: fmt.Sprintf("file too large: %d bytes (limit: %d bytes)", size, limit),
    }
}
```

## Security Considerations

### Permission Checks

```go
// PermissionChecker validates file permissions
type PermissionChecker struct {
    allowedOperations map[string][]Operation
}

// Operation represents a file operation
type Operation int

const (
    OpRead Operation = iota
    OpWrite
    OpExecute
    OpDelete
)

// CheckPermission checks if an operation is allowed on a path
func (pc *PermissionChecker) CheckPermission(path string, op Operation) error {
    // Check OS permissions
    info, err := os.Stat(path)
    if err != nil {
        return err
    }

    mode := info.Mode()
    switch op {
    case OpRead:
        if mode&0400 == 0 {
            return NewPermissionDeniedError(path, nil)
        }
    case OpWrite:
        if mode&0200 == 0 {
            return NewPermissionDeniedError(path, nil)
        }
    case OpExecute:
        if mode&0100 == 0 {
            return NewPermissionDeniedError(path, nil)
        }
    }

    // Check custom permissions
    allowed, ok := pc.allowedOperations[path]
    if ok {
        found := false
        for _, allowedOp := range allowed {
            if allowedOp == op {
                found = true
                break
            }
        }
        if !found {
            return &SecurityError{
                Type:    "operation_not_allowed",
                Message: fmt.Sprintf("operation %v not allowed", op),
                Path:    path,
            }
        }
    }

    return nil
}
```

### Path Validation Rules

1. **No Path Traversal**: Reject paths containing `..`
2. **Workspace Boundary**: Ensure all paths are within the workspace root
3. **Symlink Handling**: Configurable - follow or reject symlinks
4. **Blocked Paths**: Support for blocking sensitive directories (.git, node_modules, etc.)
5. **Absolute Path Resolution**: Always resolve to absolute paths internally

### Sensitive File Detection

```go
// SensitiveFileDetector detects potentially sensitive files
type SensitiveFileDetector struct {
    patterns []string
}

var defaultSensitivePatterns = []string{
    "*.key",
    "*.pem",
    "*.p12",
    "*.pfx",
    "*.env",
    "*.env.*",
    "*secrets*",
    "*credentials*",
    ".aws/credentials",
    ".ssh/id_*",
    "*.keystore",
}

// IsSensitive checks if a file is potentially sensitive
func (d *SensitiveFileDetector) IsSensitive(path string) bool {
    basename := filepath.Base(path)
    for _, pattern := range d.patterns {
        matched, _ := filepath.Match(pattern, basename)
        if matched {
            return true
        }
    }
    return false
}
```

## Performance Optimization

### Batch Operations

```go
// BatchReader reads multiple files efficiently
type BatchReader struct {
    reader    FileReader
    semaphore chan struct{}
}

// ReadBatch reads multiple files concurrently
func (br *BatchReader) ReadBatch(ctx context.Context, paths []string) ([]*FileContent, error) {
    results := make([]*FileContent, len(paths))
    errs := make([]error, len(paths))

    var wg sync.WaitGroup
    for i, path := range paths {
        wg.Add(1)
        go func(idx int, p string) {
            defer wg.Done()

            // Acquire semaphore
            br.semaphore <- struct{}{}
            defer func() { <-br.semaphore }()

            content, err := br.reader.Read(ctx, p)
            results[idx] = content
            errs[idx] = err
        }(i, path)
    }

    wg.Wait()

    // Check for errors
    var firstErr error
    for _, err := range errs {
        if err != nil && firstErr == nil {
            firstErr = err
        }
    }

    return results, firstErr
}
```

### Streaming for Large Files

```go
// StreamReader provides streaming access to large files
type StreamReader struct {
    file       *os.File
    bufReader  *bufio.Reader
    chunkSize  int
}

// ReadChunks reads file in chunks
func (sr *StreamReader) ReadChunks(ctx context.Context) (<-chan []byte, <-chan error) {
    chunks := make(chan []byte)
    errs := make(chan error, 1)

    go func() {
        defer close(chunks)
        defer close(errs)

        buffer := make([]byte, sr.chunkSize)
        for {
            select {
            case <-ctx.Done():
                errs <- ctx.Err()
                return
            default:
                n, err := sr.bufReader.Read(buffer)
                if n > 0 {
                    chunk := make([]byte, n)
                    copy(chunk, buffer[:n])
                    chunks <- chunk
                }
                if err == io.EOF {
                    return
                }
                if err != nil {
                    errs <- err
                    return
                }
            }
        }
    }()

    return chunks, errs
}
```

### Caching Strategy

- **LRU Cache**: Use LRU eviction for frequently accessed files
- **TTL**: Configurable time-to-live for cache entries
- **Size Limit**: Maximum cache size in bytes
- **Invalidation**: Automatic invalidation on file modification
- **Warm Cache**: Pre-load frequently accessed files

## Testing Strategy

### Unit Tests

```go
// TestFileReader tests the file reader
func TestFileReader(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        want    string
        wantErr error
    }{
        {
            name: "read existing file",
            path: "testdata/sample.txt",
            want: "Hello, World!\n",
            wantErr: nil,
        },
        {
            name: "read non-existent file",
            path: "testdata/missing.txt",
            want: "",
            wantErr: ErrorFileNotFound,
        },
        {
            name: "read file with invalid encoding",
            path: "testdata/binary.bin",
            want: "",
            wantErr: ErrorInvalidEncoding,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            reader := NewFileReader()
            content, err := reader.Read(context.Background(), tt.path)

            if tt.wantErr != nil {
                assert.Error(t, err)
                assert.ErrorIs(t, err, tt.wantErr)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, string(content.Content))
            }
        })
    }
}

// TestFileEditor tests the file editor
func TestFileEditor(t *testing.T) {
    editor := NewFileEditor()

    t.Run("insert at line", func(t *testing.T) {
        // Create temp file
        tmpfile, err := os.CreateTemp("", "test-*.txt")
        require.NoError(t, err)
        defer os.Remove(tmpfile.Name())

        content := "line 1\nline 2\nline 3\n"
        _, err = tmpfile.WriteString(content)
        require.NoError(t, err)
        tmpfile.Close()

        // Insert at line 2
        result, err := editor.InsertAt(context.Background(), tmpfile.Name(), 2, "inserted line\n")
        require.NoError(t, err)
        assert.Equal(t, 1, result.LinesChanged)

        // Verify content
        newContent, err := os.ReadFile(tmpfile.Name())
        require.NoError(t, err)
        expected := "line 1\ninserted line\nline 2\nline 3\n"
        assert.Equal(t, expected, string(newContent))
    })

    t.Run("replace pattern", func(t *testing.T) {
        tmpfile, err := os.CreateTemp("", "test-*.txt")
        require.NoError(t, err)
        defer os.Remove(tmpfile.Name())

        content := "foo bar foo baz"
        _, err = tmpfile.WriteString(content)
        require.NoError(t, err)
        tmpfile.Close()

        result, err := editor.Replace(context.Background(), tmpfile.Name(), "foo", "qux", false)
        require.NoError(t, err)
        assert.Equal(t, 1, result.LinesChanged)

        newContent, err := os.ReadFile(tmpfile.Name())
        require.NoError(t, err)
        assert.Equal(t, "qux bar qux baz", string(newContent))
    })
}

// TestPathValidator tests path validation
func TestPathValidator(t *testing.T) {
    validator := &PathValidator{
        workspaceRoot: "/workspace",
        blockedPaths:  []string{"/workspace/.git", "/workspace/node_modules"},
    }

    tests := []struct {
        name    string
        path    string
        wantErr bool
        errType string
    }{
        {
            name:    "valid path",
            path:    "/workspace/src/main.go",
            wantErr: false,
        },
        {
            name:    "path traversal",
            path:    "/workspace/../etc/passwd",
            wantErr: true,
            errType: "path_traversal",
        },
        {
            name:    "outside workspace",
            path:    "/etc/passwd",
            wantErr: true,
            errType: "outside_workspace",
        },
        {
            name:    "blocked path",
            path:    "/workspace/.git/config",
            wantErr: true,
            errType: "blocked_path",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := validator.Validate(tt.path)
            if tt.wantErr {
                require.Error(t, err)
                var secErr *SecurityError
                require.ErrorAs(t, err, &secErr)
                assert.Equal(t, tt.errType, secErr.Type)
            } else {
                require.NoError(t, err)
                assert.True(t, result.IsValid)
            }
        })
    }
}
```

### Integration Tests

```go
// TestFileSystemIntegration tests the full file system workflow
func TestFileSystemIntegration(t *testing.T) {
    // Setup
    tmpDir, err := os.MkdirTemp("", "fs-integration-test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)

    fs := NewFileSystem(&Config{
        WorkspaceRoot:  tmpDir,
        CacheEnabled:   true,
        CacheTTL:       5 * time.Minute,
        MaxFileSize:    10 * 1024 * 1024,
        FollowSymlinks: false,
    })

    ctx := context.Background()

    t.Run("create, read, edit, delete workflow", func(t *testing.T) {
        path := filepath.Join(tmpDir, "test.txt")

        // Create file
        err := fs.Writer.Write(ctx, path, []byte("Hello, World!"))
        require.NoError(t, err)

        // Read file
        content, err := fs.Reader.Read(ctx, path)
        require.NoError(t, err)
        assert.Equal(t, "Hello, World!", string(content.Content))

        // Edit file
        result, err := fs.Editor.Replace(ctx, path, "World", "HelixCode", false)
        require.NoError(t, err)
        assert.True(t, result.Success)

        // Read again
        content, err = fs.Reader.Read(ctx, path)
        require.NoError(t, err)
        assert.Equal(t, "Hello, HelixCode!", string(content.Content))

        // Delete file
        err = fs.Writer.Delete(ctx, path, false)
        require.NoError(t, err)

        // Verify deleted
        exists, err := fs.Reader.Exists(ctx, path)
        require.NoError(t, err)
        assert.False(t, exists)
    })

    t.Run("search and batch operations", func(t *testing.T) {
        // Create multiple files
        files := []string{"file1.txt", "file2.txt", "file3.go"}
        for _, file := range files {
            path := filepath.Join(tmpDir, file)
            err := fs.Writer.Write(ctx, path, []byte(fmt.Sprintf("Content of %s", file)))
            require.NoError(t, err)
        }

        // Search for .txt files
        results, err := fs.Searcher.Search(ctx, SearchOptions{
            Root:    tmpDir,
            Pattern: "*.txt",
        })
        require.NoError(t, err)
        assert.Equal(t, 2, len(results))

        // Batch read
        paths := []string{
            filepath.Join(tmpDir, "file1.txt"),
            filepath.Join(tmpDir, "file2.txt"),
        }
        batchReader := &BatchReader{
            reader:    fs.Reader,
            semaphore: make(chan struct{}, 5),
        }
        contents, err := batchReader.ReadBatch(ctx, paths)
        require.NoError(t, err)
        assert.Equal(t, 2, len(contents))
    })
}
```

### Benchmark Tests

```go
// BenchmarkFileRead benchmarks file reading performance
func BenchmarkFileRead(b *testing.B) {
    tmpfile, err := os.CreateTemp("", "bench-*.txt")
    if err != nil {
        b.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    content := bytes.Repeat([]byte("Hello, World!\n"), 1000)
    _, err = tmpfile.Write(content)
    if err != nil {
        b.Fatal(err)
    }
    tmpfile.Close()

    reader := NewFileReader()
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := reader.Read(ctx, tmpfile.Name())
        if err != nil {
            b.Fatal(err)
        }
    }
}

// BenchmarkCachedRead benchmarks cached file reading
func BenchmarkCachedRead(b *testing.B) {
    tmpfile, err := os.CreateTemp("", "bench-*.txt")
    if err != nil {
        b.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    content := bytes.Repeat([]byte("Hello, World!\n"), 1000)
    _, err = tmpfile.Write(content)
    if err != nil {
        b.Fatal(err)
    }
    tmpfile.Close()

    fs := NewFileSystem(&Config{
        CacheEnabled: true,
        CacheTTL:     5 * time.Minute,
    })
    ctx := context.Background()

    // Warm cache
    _, _ = fs.Reader.Read(ctx, tmpfile.Name())

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := fs.Reader.Read(ctx, tmpfile.Name())
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Implementation Notes

### Configuration

```go
// Config contains file system configuration
type Config struct {
    WorkspaceRoot    string
    CacheEnabled     bool
    CacheTTL         time.Duration
    MaxCacheSize     int64
    MaxFileSize      int64
    MaxBatchSize     int
    FollowSymlinks   bool
    AllowedPaths     []string
    BlockedPaths     []string
    SensitivePatterns []string
    Concurrency      int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
    return &Config{
        CacheEnabled:   true,
        CacheTTL:       5 * time.Minute,
        MaxCacheSize:   100 * 1024 * 1024, // 100 MB
        MaxFileSize:    50 * 1024 * 1024,  // 50 MB
        MaxBatchSize:   100,
        FollowSymlinks: false,
        BlockedPaths: []string{
            ".git",
            "node_modules",
            ".env",
            ".env.*",
        },
        SensitivePatterns: defaultSensitivePatterns,
        Concurrency:       10,
    }
}
```

### Atomic Operations

For critical write operations, use atomic writes:

1. Write to temporary file
2. Sync to disk
3. Rename to target (atomic on POSIX)
4. Clean up on failure

```go
// AtomicWrite writes content atomically
func AtomicWrite(path string, content []byte) error {
    dir := filepath.Dir(path)
    tmpfile, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil {
        return err
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.Write(content); err != nil {
        tmpfile.Close()
        return err
    }

    if err := tmpfile.Sync(); err != nil {
        tmpfile.Close()
        return err
    }

    if err := tmpfile.Close(); err != nil {
        return err
    }

    return os.Rename(tmpfile.Name(), path)
}
```

### Error Recovery

- **Backup on Write**: Optionally create backups before overwriting
- **Transaction Log**: Log all operations for potential rollback
- **Checksum Verification**: Verify file integrity after write

## References

### Cline's File Tools

- **Location**: `src/core/webview/WebviewProvider.ts`
- **Features**:
  - Read/write files with line number support
  - Edit operations with diff generation
  - Search with glob patterns
  - File tree navigation

### Qwen Code's File Operations

- **Location**: Various file tools
- **Features**:
  - Efficient file reading with encoding detection
  - Batch operations
  - Smart caching

### Additional References

- Go standard library `os`, `io`, `path/filepath`
- `github.com/hashicorp/golang-lru` for LRU cache
- `github.com/fsnotify/fsnotify` for file watching (future enhancement)

## Future Enhancements

1. **File Watching**: Real-time file change notifications
2. **Compression**: Transparent compression for large files
3. **Encryption**: Optional encryption for sensitive files
4. **Version Control Integration**: Track file versions
5. **Remote File Systems**: Support for remote file systems (S3, SFTP)
6. **Diff Viewer**: Enhanced diff visualization
7. **Binary File Support**: Improved handling of binary files
8. **Large File Streaming**: Better support for files > 1GB
