/*
Package filesystem provides secure, efficient, and user-friendly file system operations for HelixCode.

This package is inspired by Cline's file tools and Qwen Code's file operations, with enhancements
for security, performance, and error handling.

# Overview

The filesystem package offers a comprehensive set of tools for working with files and directories:

  - FileReader: Read files with support for partial reads, line ranges, and size limits
  - FileWriter: Write files with atomic operations, backups, and line ending conversion
  - FileEditor: Edit files in-place with operations like insert, delete, and replace
  - FileSearcher: Search files by name patterns or content with glob and regex support

# Basic Usage

Create a new FileSystemTools instance:

	config := filesystem.DefaultConfig()
	config.WorkspaceRoot = "/path/to/workspace"
	fs, err := filesystem.NewFileSystemTools(config)
	if err != nil {
		log.Fatal(err)
	}

Read a file:

	ctx := context.Background()
	content, err := fs.Reader().Read(ctx, "/path/to/file.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(content.Content))

Write a file:

	content := []byte("Hello, World!")
	err := fs.Writer().Write(ctx, "/path/to/file.txt", content)
	if err != nil {
		log.Fatal(err)
	}

Edit a file:

	result, err := fs.Editor().Replace(ctx, "/path/to/file.txt", "old", "new", false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Changed %d lines\n", result.LinesChanged)

Search files:

	opts := filesystem.SearchOptions{
		Root:    "/path/to/search",
		Pattern: "*.go",
	}
	results, err := fs.Searcher().Search(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

# Security Features

The package implements several security measures:

Path Validation:
  - All paths are validated against the workspace root
  - Path traversal attempts (../) are blocked
  - Symlinks can be optionally followed or blocked
  - Sensitive paths (.git, .env, etc.) can be blocked

Permission Checking:
  - OS-level permission checks before operations
  - Custom permission rules can be configured
  - Operations are validated against allowed operations

Sensitive File Detection:
  - Automatic detection of potentially sensitive files
  - Blocks writing to files like .env, *.key, *.pem, etc.
  - Customizable sensitive file patterns

# Performance Optimization

The package includes several performance optimizations:

Caching:
  - LRU cache for frequently accessed files
  - Automatic cache invalidation on modifications
  - Configurable TTL and size limits

Atomic Operations:
  - Write-to-temp-then-rename for atomic writes
  - Prevents partial writes in case of failures
  - Ensures data integrity

File Locking:
  - Prevents concurrent modifications to the same file
  - Context-aware lock acquisition with timeouts
  - Automatic lock release on operation completion

# Error Handling

The package provides structured error types:

	_, err := fs.Reader().Read(ctx, "/path/to/file.txt")
	if err != nil {
		var fsErr *filesystem.FileSystemError
		if errors.As(err, &fsErr) {
			switch fsErr.Type {
			case filesystem.ErrorFileNotFound:
				fmt.Println("File not found")
			case filesystem.ErrorPermissionDenied:
				fmt.Println("Permission denied")
			default:
				fmt.Printf("Error: %v\n", err)
			}
		}
	}

# Advanced Features

Read Specific Lines:

	content, err := fs.Reader().ReadLines(ctx, path, 10, 20) // Lines 10-20

Write with Options:

	opts := filesystem.DefaultWriteOptions()
	opts.Backup = true
	opts.Atomic = true
	opts.LineEnding = filesystem.LineEndingCRLF
	err := fs.Writer().WriteWithOptions(ctx, path, content, opts)

Content Search with Context:

	opts := filesystem.ContentSearchOptions{
		Root:         "/path/to/search",
		Pattern:      "TODO",
		ContextLines: 2, // Include 2 lines before and after
	}
	matches, err := fs.Searcher().SearchContent(ctx, opts)

Edit Operations:

	ops := []filesystem.EditOperation{
		{
			Type:      filesystem.EditInsert,
			StartLine: 5,
			Content:   "// New comment",
		},
		{
			Type:        filesystem.EditReplacePattern,
			Pattern:     "old_function",
			Replacement: "new_function",
			IsRegex:     false,
		},
	}
	result, err := fs.Editor().Edit(ctx, path, ops)

# Configuration

The Config struct provides extensive configuration options:

	config := &filesystem.Config{
		WorkspaceRoot:     "/workspace",
		CacheEnabled:      true,
		CacheTTL:          5 * time.Minute,
		MaxCacheSize:      100 * 1024 * 1024, // 100 MB
		MaxFileSize:       50 * 1024 * 1024,  // 50 MB
		MaxBatchSize:      100,
		FollowSymlinks:    false,
		BlockedPaths:      []string{".git", "node_modules"},
		SensitivePatterns: []string{"*.key", "*.env"},
		Concurrency:       10,
	}

# Thread Safety

All operations are thread-safe:
  - File locking prevents concurrent modifications
  - Cache operations use appropriate synchronization
  - Context cancellation is properly handled

# Best Practices

1. Always use context for cancellation support:
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()

2. Check for specific error types:
   var secErr *filesystem.SecurityError
   if errors.As(err, &secErr) {
       // Handle security error
   }

3. Use atomic writes for critical files:
   opts := filesystem.DefaultWriteOptions()
   opts.Atomic = true
   opts.Backup = true

4. Configure workspace root to prevent unauthorized access:
   config.WorkspaceRoot = "/safe/workspace"

5. Use appropriate permissions:
   opts.Mode = 0644 // rw-r--r--

# Examples

See the test files for more examples of usage patterns and edge cases.
*/
package filesystem
