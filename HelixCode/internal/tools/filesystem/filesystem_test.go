package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// TestNewFileSystemTools tests creating a new file system tools instance
func TestNewFileSystemTools(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "default config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "custom config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "with workspace root",
			config: &Config{
				WorkspaceRoot: "/tmp/test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := NewFileSystemTools(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileSystemTools() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && fs == nil {
				t.Error("NewFileSystemTools() returned nil")
			}
		})
	}
}

// TestFileReader tests file reading operations
func TestFileReader(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file system tools
	fs, err := NewFileSystemTools(&Config{
		WorkspaceRoot: tmpDir,
		CacheEnabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	t.Run("read existing file", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "Hello, World!\nLine 2\nLine 3\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		// Read file
		fileContent, err := fs.Reader().Read(ctx, testFile)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if string(fileContent.Content) != content {
			t.Errorf("Read() content = %v, want %v", string(fileContent.Content), content)
		}

		if fileContent.TotalLines != 3 {
			t.Errorf("Read() lines = %v, want 3", fileContent.TotalLines)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := fs.Reader().Read(ctx, filepath.Join(tmpDir, "missing.txt"))
		if err == nil {
			t.Error("Read() expected error for non-existent file")
		}
	})

	t.Run("read lines", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "lines.txt")
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		// Read lines 2-4
		fileContent, err := fs.Reader().ReadLines(ctx, testFile, 2, 4)
		if err != nil {
			t.Fatalf("ReadLines() error = %v", err)
		}

		if len(fileContent.Lines) != 3 {
			t.Errorf("ReadLines() lines = %v, want 3", len(fileContent.Lines))
		}

		if fileContent.Lines[0] != "Line 2" {
			t.Errorf("ReadLines() first line = %v, want 'Line 2'", fileContent.Lines[0])
		}
	})

	t.Run("file exists", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "exists.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		exists, err := fs.Reader().Exists(ctx, testFile)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}

		if !exists {
			t.Error("Exists() = false, want true")
		}

		exists, err = fs.Reader().Exists(ctx, filepath.Join(tmpDir, "not-exists.txt"))
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}

		if exists {
			t.Error("Exists() = true, want false")
		}
	})

	t.Run("get file info", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "info.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Reader().GetInfo(ctx, testFile)
		if err != nil {
			t.Fatalf("GetInfo() error = %v", err)
		}

		if info.Name != "info.txt" {
			t.Errorf("GetInfo() name = %v, want 'info.txt'", info.Name)
		}

		if info.Size != 12 {
			t.Errorf("GetInfo() size = %v, want 12", info.Size)
		}
	})
}

// TestFileWriter tests file writing operations
func TestFileWriter(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file system tools
	fs, err := NewFileSystemTools(&Config{
		WorkspaceRoot: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	t.Run("write new file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "write.txt")
		content := []byte("Hello, World!")

		if err := fs.Writer().Write(ctx, testFile, content); err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Verify content
		readContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		if string(readContent) != string(content) {
			t.Errorf("Write() content = %v, want %v", string(readContent), string(content))
		}
	})

	t.Run("write with backup", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "backup.txt")
		originalContent := []byte("original")
		newContent := []byte("modified")

		// Write original
		if err := os.WriteFile(testFile, originalContent, 0644); err != nil {
			t.Fatal(err)
		}

		// Write with backup
		opts := DefaultWriteOptions()
		opts.Backup = true
		if err := fs.Writer().WriteWithOptions(ctx, testFile, newContent, opts); err != nil {
			t.Fatalf("WriteWithOptions() error = %v", err)
		}

		// Verify backup exists
		backupPattern := testFile + ".bak.*"
		matches, err := filepath.Glob(backupPattern)
		if err != nil {
			t.Fatal(err)
		}

		if len(matches) == 0 {
			t.Error("WriteWithOptions() backup not created")
		}
	})

	t.Run("append to file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "append.txt")
		initialContent := []byte("Line 1\n")
		appendContent := []byte("Line 2\n")

		// Write initial content
		if err := os.WriteFile(testFile, initialContent, 0644); err != nil {
			t.Fatal(err)
		}

		// Append
		if err := fs.Writer().Append(ctx, testFile, appendContent); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		// Verify
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		expected := string(initialContent) + string(appendContent)
		if string(content) != expected {
			t.Errorf("Append() content = %v, want %v", string(content), expected)
		}
	})

	t.Run("create file (fail if exists)", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "create.txt")

		// Create file
		if err := fs.Writer().Create(ctx, testFile, []byte("content")); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Try to create again
		err := fs.Writer().Create(ctx, testFile, []byte("new content"))
		if err == nil {
			t.Error("Create() expected error for existing file")
		}
	})

	t.Run("create directory", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "subdir", "nested")

		if err := fs.Writer().CreateDirectory(ctx, testDir, 0755); err != nil {
			t.Fatalf("CreateDirectory() error = %v", err)
		}

		// Verify directory exists
		info, err := os.Stat(testDir)
		if err != nil {
			t.Fatal(err)
		}

		if !info.IsDir() {
			t.Error("CreateDirectory() path is not a directory")
		}
	})

	t.Run("delete file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "delete.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := fs.Writer().Delete(ctx, testFile, false); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deleted
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("Delete() file still exists")
		}
	})

	t.Run("move file", func(t *testing.T) {
		srcFile := filepath.Join(tmpDir, "move-src.txt")
		dstFile := filepath.Join(tmpDir, "move-dst.txt")
		content := []byte("move test")

		if err := os.WriteFile(srcFile, content, 0644); err != nil {
			t.Fatal(err)
		}

		if err := fs.Writer().Move(ctx, srcFile, dstFile); err != nil {
			t.Fatalf("Move() error = %v", err)
		}

		// Verify source deleted
		if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
			t.Error("Move() source file still exists")
		}

		// Verify destination exists
		dstContent, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatal(err)
		}

		if string(dstContent) != string(content) {
			t.Error("Move() destination content incorrect")
		}
	})

	t.Run("copy file", func(t *testing.T) {
		srcFile := filepath.Join(tmpDir, "copy-src.txt")
		dstFile := filepath.Join(tmpDir, "copy-dst.txt")
		content := []byte("copy test")

		if err := os.WriteFile(srcFile, content, 0644); err != nil {
			t.Fatal(err)
		}

		if err := fs.Writer().Copy(ctx, srcFile, dstFile); err != nil {
			t.Fatalf("Copy() error = %v", err)
		}

		// Verify both exist
		if _, err := os.Stat(srcFile); err != nil {
			t.Error("Copy() source file deleted")
		}

		dstContent, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatal(err)
		}

		if string(dstContent) != string(content) {
			t.Error("Copy() destination content incorrect")
		}
	})
}

// TestFileEditor tests file editing operations
func TestFileEditor(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file system tools
	fs, err := NewFileSystemTools(&Config{
		WorkspaceRoot: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	t.Run("insert at line", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "insert.txt")
		content := "Line 1\nLine 2\nLine 3\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := fs.Editor().InsertAt(ctx, testFile, 2, "Inserted Line")
		if err != nil {
			t.Fatalf("InsertAt() error = %v", err)
		}

		if !result.Success {
			t.Error("InsertAt() failed")
		}

		// Verify content
		newContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(newContent), "Inserted Line") {
			t.Error("InsertAt() line not inserted")
		}
	})

	t.Run("delete lines", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "delete-lines.txt")
		content := "Line 1\nLine 2\nLine 3\nLine 4\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := fs.Editor().DeleteLines(ctx, testFile, 2, 3)
		if err != nil {
			t.Fatalf("DeleteLines() error = %v", err)
		}

		if !result.Success {
			t.Error("DeleteLines() failed")
		}

		// Verify content
		newContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(string(newContent), "\n")
		if len(lines) < 2 || lines[1] == "Line 2" || lines[1] == "Line 3" {
			t.Error("DeleteLines() lines not deleted")
		}
	})

	t.Run("replace pattern", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "replace.txt")
		content := "foo bar foo baz"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := fs.Editor().Replace(ctx, testFile, "foo", "qux", false)
		if err != nil {
			t.Fatalf("Replace() error = %v", err)
		}

		if !result.Success {
			t.Error("Replace() failed")
		}

		// Verify content
		newContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		expected := "qux bar qux baz"
		if string(newContent) != expected {
			t.Errorf("Replace() content = %v, want %v", string(newContent), expected)
		}
	})

	t.Run("replace with regex", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "regex.txt")
		content := "test123 and test456"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := fs.Editor().Replace(ctx, testFile, `test\d+`, "replaced", true)
		if err != nil {
			t.Fatalf("Replace() error = %v", err)
		}

		if !result.Success {
			t.Error("Replace() failed")
		}

		// Verify content
		newContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(newContent), "replaced") {
			t.Error("Replace() regex not applied")
		}
	})
}

// TestFileSearcher tests file searching operations
func TestFileSearcher(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := map[string]string{
		"file1.txt":        "content 1",
		"file2.txt":        "content 2",
		"file3.go":         "package main",
		"subdir/file4.txt": "content 4",
		".hidden":          "hidden content",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create file system tools
	fs, err := NewFileSystemTools(&Config{
		WorkspaceRoot: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	t.Run("search files by pattern", func(t *testing.T) {
		opts := SearchOptions{
			Root:    tmpDir,
			Pattern: "*.txt",
		}

		results, err := fs.Searcher().Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		if len(results) < 2 {
			t.Errorf("Search() found %d files, want at least 2", len(results))
		}
	})

	t.Run("search with max depth", func(t *testing.T) {
		opts := SearchOptions{
			Root:     tmpDir,
			Pattern:  "*.txt",
			MaxDepth: 1,
		}

		results, err := fs.Searcher().Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		// Should not include subdir/file4.txt
		for _, result := range results {
			if strings.Contains(result.Path, "subdir") {
				t.Error("Search() found file beyond max depth")
			}
		}
	})

	t.Run("search content", func(t *testing.T) {
		opts := ContentSearchOptions{
			Root:    tmpDir,
			Pattern: "content",
		}

		matches, err := fs.Searcher().SearchContent(ctx, opts)
		if err != nil {
			t.Fatalf("SearchContent() error = %v", err)
		}

		if len(matches) < 2 {
			t.Errorf("SearchContent() found %d matches, want at least 2", len(matches))
		}
	})

	t.Run("glob pattern", func(t *testing.T) {
		pattern := filepath.Join(tmpDir, "*.txt")
		matches, err := fs.Searcher().Glob(ctx, pattern)
		if err != nil {
			t.Fatalf("Glob() error = %v", err)
		}

		if len(matches) < 2 {
			t.Errorf("Glob() found %d matches, want at least 2", len(matches))
		}
	})
}

// TestPathValidator tests path validation
func TestPathValidator(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	validator := &PathValidator{
		workspaceRoot: tmpDir,
		blockedPaths:  []string{filepath.Join(tmpDir, ".git")},
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType string
	}{
		{
			name:    "valid path",
			path:    filepath.Join(tmpDir, "file.txt"),
			wantErr: false,
		},
		{
			name:    "outside workspace",
			path:    "/etc/passwd",
			wantErr: true,
			errType: "outside_workspace",
		},
		{
			name:    "blocked path",
			path:    filepath.Join(tmpDir, ".git", "config"),
			wantErr: true,
			errType: "blocked_path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != "" {
				if secErr, ok := err.(*SecurityError); ok {
					if secErr.Type != tt.errType {
						t.Errorf("Validate() error type = %v, want %v", secErr.Type, tt.errType)
					}
				} else {
					t.Error("Validate() error is not SecurityError")
				}
			}

			if !tt.wantErr && result != nil && !result.IsValid {
				t.Error("Validate() result not valid")
			}
		})
	}
}

// TestCacheManager tests cache functionality
func TestCacheManager(t *testing.T) {
	cache, err := newCacheManager(100*1024*1024, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	path := "/test/file.txt"
	content := []byte("test content")
	modTime := time.Now()

	t.Run("set and get", func(t *testing.T) {
		cache.Set(path, content, modTime)

		entry, ok := cache.Get(path)
		if !ok {
			t.Error("Get() returned false, want true")
		}

		if entry == nil {
			t.Fatal("Get() returned nil entry")
		}

		if string(entry.Content) != string(content) {
			t.Errorf("Get() content = %v, want %v", string(entry.Content), string(content))
		}
	})

	t.Run("invalidate", func(t *testing.T) {
		cache.Set(path, content, modTime)
		cache.Invalidate(path)

		_, ok := cache.Get(path)
		if ok {
			t.Error("Get() returned true after invalidation, want false")
		}
	})
}

// TestSensitiveFileDetector tests sensitive file detection
func TestSensitiveFileDetector(t *testing.T) {
	detector := NewSensitiveFileDetector(nil)

	tests := []struct {
		name      string
		path      string
		sensitive bool
	}{
		{
			name:      "env file",
			path:      "/path/to/.env",
			sensitive: true,
		},
		{
			name:      "key file",
			path:      "/path/to/private.key",
			sensitive: true,
		},
		{
			name:      "normal file",
			path:      "/path/to/file.txt",
			sensitive: false,
		},
		{
			name:      "go file",
			path:      "/path/to/main.go",
			sensitive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.IsSensitive(tt.path)
			if result != tt.sensitive {
				t.Errorf("IsSensitive() = %v, want %v", result, tt.sensitive)
			}
		})
	}
}

// Helper function to create cache manager
func newCacheManager(maxSize int64, ttl time.Duration) (*CacheManager, error) {
	cache, err := lru.New[string, *CacheEntry](1000)
	if err != nil {
		return nil, err
	}
	return &CacheManager{
		cache:   cache,
		ttl:     ttl,
		maxSize: maxSize,
		stats:   &CacheStats{},
	}, nil
}
