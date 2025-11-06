package filesystem

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// FileSystemTools provides comprehensive file system operations
type FileSystemTools struct {
	config            *Config
	reader            FileReader
	writer            FileWriter
	editor            FileEditor
	searcher          FileSearcher
	pathValidator     *PathValidator
	permissionChecker *PermissionChecker
	cacheManager      *CacheManager
	lockManager       *LockManager
	sensitiveDetector *SensitiveFileDetector
}

// Config contains file system configuration
type Config struct {
	WorkspaceRoot     string
	CacheEnabled      bool
	CacheTTL          time.Duration
	MaxCacheSize      int64
	MaxFileSize       int64
	MaxBatchSize      int
	FollowSymlinks    bool
	AllowedPaths      []string
	BlockedPaths      []string
	SensitivePatterns []string
	Concurrency       int
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
		},
		SensitivePatterns: defaultSensitivePatterns,
		Concurrency:       10,
	}
}

// NewFileSystemTools creates a new file system tools instance
func NewFileSystemTools(config *Config) (*FileSystemTools, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Set workspace root to current directory if not specified
	if config.WorkspaceRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		config.WorkspaceRoot = wd
	}

	// Normalize workspace root
	absRoot, err := filepath.Abs(config.WorkspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize workspace root: %w", err)
	}
	config.WorkspaceRoot = absRoot

	fs := &FileSystemTools{
		config:            config,
		lockManager:       NewLockManager(),
		sensitiveDetector: NewSensitiveFileDetector(config.SensitivePatterns),
	}

	// Initialize path validator
	fs.pathValidator = &PathValidator{
		workspaceRoot:  config.WorkspaceRoot,
		allowedPaths:   config.AllowedPaths,
		blockedPaths:   config.BlockedPaths,
		followSymlinks: config.FollowSymlinks,
	}

	// Initialize permission checker
	fs.permissionChecker = &PermissionChecker{
		allowedOperations: make(map[string][]Operation),
	}

	// Initialize cache manager
	if config.CacheEnabled {
		cache, err := lru.New[string, *CacheEntry](1000)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
		fs.cacheManager = &CacheManager{
			cache:   cache,
			ttl:     config.CacheTTL,
			maxSize: config.MaxCacheSize,
			stats:   &CacheStats{},
		}
	}

	// Initialize components
	fs.reader = &fileReader{fs: fs}
	fs.writer = &fileWriter{fs: fs}
	fs.editor = &fileEditor{fs: fs}
	fs.searcher = &fileSearcher{fs: fs}

	return fs, nil
}

// Reader returns the file reader
func (fs *FileSystemTools) Reader() FileReader {
	return fs.reader
}

// Writer returns the file writer
func (fs *FileSystemTools) Writer() FileWriter {
	return fs.writer
}

// Editor returns the file editor
func (fs *FileSystemTools) Editor() FileEditor {
	return fs.editor
}

// Searcher returns the file searcher
func (fs *FileSystemTools) Searcher() FileSearcher {
	return fs.searcher
}

// PathValidator validates and normalizes file paths
type PathValidator struct {
	workspaceRoot  string
	allowedPaths   []string
	blockedPaths   []string
	followSymlinks bool
}

// ValidationResult contains path validation results
type ValidationResult struct {
	IsValid        bool
	NormalizedPath string
	IsAbsolute     bool
	IsSymlink      bool
	ResolvedPath   string
	Error          error
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

	// Clean path to resolve any .. elements
	cleaned := filepath.Clean(normalized)

	// Check if within workspace
	if v.workspaceRoot != "" {
		rel, err := filepath.Rel(v.workspaceRoot, cleaned)
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
		blockedAbs := blocked
		if !filepath.IsAbs(blocked) && v.workspaceRoot != "" {
			blockedAbs = filepath.Join(v.workspaceRoot, blocked)
		}
		if strings.HasPrefix(cleaned, blockedAbs) || cleaned == blockedAbs {
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

// CacheManager manages file content caching
type CacheManager struct {
	cache   *lru.Cache[string, *CacheEntry]
	ttl     time.Duration
	maxSize int64
	stats   *CacheStats
	mu      sync.RWMutex
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
	Hits        atomic.Int64
	Misses      atomic.Int64
	Evictions   atomic.Int64
	BytesCached atomic.Int64
}

// Get retrieves a file from cache
func (cm *CacheManager) Get(path string) (*CacheEntry, bool) {
	if cm == nil {
		return nil, false
	}

	cm.mu.RLock()
	entry, ok := cm.cache.Get(path)
	cm.mu.RUnlock()

	if !ok {
		cm.stats.Misses.Add(1)
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		cm.mu.Lock()
		cm.cache.Remove(path)
		cm.mu.Unlock()
		cm.stats.Misses.Add(1)
		return nil, false
	}

	// Verify file hasn't changed
	info, err := os.Stat(path)
	if err != nil || !info.ModTime().Equal(entry.ModTime) {
		cm.mu.Lock()
		cm.cache.Remove(path)
		cm.mu.Unlock()
		cm.stats.Misses.Add(1)
		return nil, false
	}

	cm.stats.Hits.Add(1)
	return entry, true
}

// Set adds a file to cache
func (cm *CacheManager) Set(path string, content []byte, modTime time.Time) {
	if cm == nil {
		return
	}

	entry := &CacheEntry{
		Path:      path,
		Content:   content,
		ModTime:   modTime,
		Size:      int64(len(content)),
		ExpiresAt: time.Now().Add(cm.ttl),
		Checksum:  fmt.Sprintf("%x", sha256.Sum256(content)),
	}

	cm.mu.Lock()
	cm.cache.Add(path, entry)
	cm.mu.Unlock()
	cm.stats.BytesCached.Add(int64(len(content)))
}

// Invalidate removes a file from cache
func (cm *CacheManager) Invalidate(path string) {
	if cm == nil {
		return
	}
	cm.mu.Lock()
	cm.cache.Remove(path)
	cm.mu.Unlock()
}

// LockManager manages file locks to prevent concurrent modifications
type LockManager struct {
	locks sync.Map
}

// FileLock represents a lock on a file
type FileLock struct {
	Path       string
	Owner      string
	AcquiredAt time.Time
	mu         sync.RWMutex
}

// NewLockManager creates a new lock manager
func NewLockManager() *LockManager {
	return &LockManager{}
}

// Acquire acquires a lock on a file
func (lm *LockManager) Acquire(ctx context.Context, path, owner string) (*FileLock, error) {
	for {
		lock := &FileLock{
			Path:       path,
			Owner:      owner,
			AcquiredAt: time.Now(),
		}

		if _, loaded := lm.locks.LoadOrStore(path, lock); !loaded {
			return lock, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to acquire lock: %w", ctx.Err())
		case <-time.After(10 * time.Millisecond):
			// Retry
		}
	}
}

// Release releases a file lock
func (lm *LockManager) Release(lock *FileLock) {
	lm.locks.Delete(lock.Path)
}

// PermissionChecker validates file permissions
type PermissionChecker struct {
	allowedOperations map[string][]Operation
	mu                sync.RWMutex
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
		if os.IsNotExist(err) {
			// File doesn't exist - allow create operations
			if op == OpWrite {
				// Check parent directory permissions
				parentDir := filepath.Dir(path)
				parentInfo, err := os.Stat(parentDir)
				if err != nil {
					return NewPermissionDeniedError(path, err)
				}
				if parentInfo.Mode()&0200 == 0 {
					return NewPermissionDeniedError(path, nil)
				}
				return nil
			}
		}
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
	pc.mu.RLock()
	allowed, ok := pc.allowedOperations[path]
	pc.mu.RUnlock()

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

// NewSensitiveFileDetector creates a new sensitive file detector
func NewSensitiveFileDetector(patterns []string) *SensitiveFileDetector {
	if len(patterns) == 0 {
		patterns = defaultSensitivePatterns
	}
	return &SensitiveFileDetector{
		patterns: patterns,
	}
}

// IsSensitive checks if a file is potentially sensitive
func (d *SensitiveFileDetector) IsSensitive(path string) bool {
	basename := filepath.Base(path)
	lowerPath := strings.ToLower(path)
	lowerBase := strings.ToLower(basename)

	for _, pattern := range d.patterns {
		matched, _ := filepath.Match(strings.ToLower(pattern), lowerBase)
		if matched {
			return true
		}
		// Also check if pattern is in path
		if strings.Contains(lowerPath, strings.ToLower(strings.Trim(pattern, "*"))) {
			return true
		}
	}
	return false
}

// AtomicWrite writes content atomically to a file
func AtomicWrite(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmpfile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpfile.Name()
	defer os.Remove(tmpPath)

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

	// Set permissions
	if err := os.Chmod(tmpPath, mode); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}
