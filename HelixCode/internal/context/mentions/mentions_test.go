package mentions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMentionParser_Parse(t *testing.T) {
	parser := NewMentionParser()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single file mention",
			input:    "Check @file[main.go] for issues",
			expected: []string{"@file[main.go]"},
		},
		{
			name:     "multiple mentions",
			input:    "@file[main.go] and @folder[src] need review",
			expected: []string{"@file[main.go]", "@folder[src]"},
		},
		{
			name:     "mention with options",
			input:    "@folder[src](recursive=true,content=false)",
			expected: []string{"@folder[src](recursive=true,content=false)"},
		},
		{
			name:     "no mentions",
			input:    "This is plain text",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.input)
			assert.Equal(t, len(tt.expected), len(result))
			for i, expected := range tt.expected {
				if i < len(result) {
					assert.Equal(t, expected, result[i])
				}
			}
		})
	}
}

func TestFileMentionHandler(t *testing.T) {
	// Create temp directory with test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	handler := NewFileMentionHandler(tmpDir)

	t.Run("resolve existing file", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, "test.txt", nil)

		require.NoError(t, err)
		assert.Equal(t, MentionTypeFile, result.Type)
		assert.Equal(t, "test.txt", result.Target)
		assert.Equal(t, testContent, result.Content)
		assert.Greater(t, result.TokenCount, 0)
	})

	t.Run("file not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := handler.Resolve(ctx, "nonexistent.txt", nil)

		assert.Error(t, err)
	})

	t.Run("empty target error", func(t *testing.T) {
		ctx := context.Background()
		_, err := handler.Resolve(ctx, "", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("with absolute path", func(t *testing.T) {
		ctx := context.Background()
		absPath := filepath.Join(tmpDir, "test.txt")
		result, err := handler.Resolve(ctx, absPath, nil)

		require.NoError(t, err)
		assert.Equal(t, MentionTypeFile, result.Type)
		assert.Equal(t, testContent, result.Content)
	})
}

func TestFolderMentionHandler(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("content2"), 0644))

	handler := NewFolderMentionHandler(tmpDir, 8000)

	t.Run("non-recursive listing", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, ".", map[string]string{"recursive": "false"})

		require.NoError(t, err)
		assert.Equal(t, MentionTypeFolder, result.Type)
		assert.Contains(t, result.Content, "file1.txt")
		assert.NotContains(t, result.Content, "file2.txt") // In subdir
	})

	t.Run("recursive listing", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, ".", map[string]string{"recursive": "true"})

		require.NoError(t, err)
		assert.Contains(t, result.Content, "file1.txt")
		assert.Contains(t, result.Content, "file2.txt")
	})

	t.Run("with content inclusion", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, ".", map[string]string{"content": "true", "recursive": "false"})

		require.NoError(t, err)
		assert.Contains(t, result.Content, "content1")
	})

	t.Run("empty target error", func(t *testing.T) {
		ctx := context.Background()
		_, err := handler.Resolve(ctx, "", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("non-existent folder", func(t *testing.T) {
		ctx := context.Background()
		_, err := handler.Resolve(ctx, "nonexistent", nil)

		assert.Error(t, err)
	})

	t.Run("subdir listing", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, "subdir", map[string]string{"recursive": "false"})

		require.NoError(t, err)
		assert.Contains(t, result.Content, "file2.txt")
	})
}

func TestFuzzySearch(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{
		"file_mention.go",
		"folder_mention.go",
		"main.go",
		"test/mention_test.go",
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
	}

	fs := NewFuzzySearch(tmpDir)

	t.Run("exact match", func(t *testing.T) {
		matches := fs.Search("main.go", 5)
		require.Greater(t, len(matches), 0)
		assert.Contains(t, matches[0].Path, "main.go")
		assert.Greater(t, matches[0].Score, 0)
	})

	t.Run("partial match", func(t *testing.T) {
		matches := fs.Search("mention", 5)
		require.Greater(t, len(matches), 0)
		// Should match files containing "mention"
		for _, match := range matches {
			assert.Contains(t, match.Path, "mention")
		}
	})

	t.Run("no matches", func(t *testing.T) {
		matches := fs.Search("nonexistent_xyz", 5)
		assert.Equal(t, 0, len(matches))
	})
}

func TestGitMentionHandler(t *testing.T) {
	// Check if in a git repository by trying to run git status
	ctx := context.Background()
	testCmd := exec.CommandContext(ctx, "git", "status")
	if err := testCmd.Run(); err != nil {
		t.Skip("Not in a git repository or git not available")
	}

	handler := NewGitMentionHandler(".")

	t.Run("git changes", func(t *testing.T) {
		result, err := handler.Resolve(ctx, "", nil)

		require.NoError(t, err)
		assert.Equal(t, MentionTypeGitChanges, result.Type)
		// Content might be empty if no changes
		assert.NotNil(t, result.Content)
		assert.GreaterOrEqual(t, result.TokenCount, 0)
		assert.NotNil(t, result.Metadata)
	})

	t.Run("git changes with explicit target", func(t *testing.T) {
		result, err := handler.Resolve(ctx, "git-changes", nil)

		require.NoError(t, err)
		assert.Equal(t, MentionTypeGitChanges, result.Type)
		assert.NotNil(t, result.Content)
	})

	t.Run("git commit - HEAD", func(t *testing.T) {
		// Try to resolve HEAD commit
		result, err := handler.Resolve(ctx, "HEAD", nil)

		if err != nil {
			t.Skip("Could not resolve HEAD commit (repo might be empty)")
		}

		assert.Equal(t, MentionTypeCommit, result.Type)
		assert.Equal(t, "HEAD", result.Target)
		assert.NotEmpty(t, result.Content)
		assert.Greater(t, result.TokenCount, 0)
	})

	t.Run("git invalid commit", func(t *testing.T) {
		// Try to resolve invalid commit
		_, err := handler.Resolve(ctx, "invalid-commit-hash-xyz", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get commit")
	})
}

func TestTerminalMentionHandler(t *testing.T) {
	handler := NewTerminalMentionHandler()

	// Add some terminal output
	handler.AddOutput("$ go test")
	handler.AddOutput("PASS")
	handler.AddOutput("ok    package    0.123s")

	t.Run("resolve terminal output", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, "", map[string]string{"lines": "10"})

		require.NoError(t, err)
		assert.Equal(t, MentionTypeTerminal, result.Type)
		assert.Contains(t, result.Content, "go test")
		assert.Contains(t, result.Content, "PASS")
	})
}

func TestProblemsMentionHandler(t *testing.T) {
	handler := NewProblemsMentionHandler()

	// Add some problems
	handler.AddProblem(Problem{
		Type:    "error",
		File:    "main.go",
		Line:    10,
		Column:  5,
		Message: "undefined variable",
		Source:  "compiler",
	})
	handler.AddProblem(Problem{
		Type:    "warning",
		File:    "util.go",
		Line:    20,
		Column:  15,
		Message: "unused variable",
		Source:  "linter",
	})

	t.Run("all problems", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, "", map[string]string{"type": "all"})

		require.NoError(t, err)
		assert.Equal(t, MentionTypeProblems, result.Type)
		assert.Contains(t, result.Content, "undefined variable")
		assert.Contains(t, result.Content, "unused variable")
		assert.Equal(t, 1, result.Metadata["error_count"])
		assert.Equal(t, 1, result.Metadata["warning_count"])
	})

	t.Run("errors only", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, "", map[string]string{"type": "errors"})

		require.NoError(t, err)
		assert.Contains(t, result.Content, "undefined variable")
		assert.NotContains(t, result.Content, "unused variable")
	})
}

func TestMentionParser_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644))

	parser := NewMentionParser()
	parser.RegisterHandler(NewFileMentionHandler(tmpDir))
	parser.RegisterHandler(NewFolderMentionHandler(tmpDir, 8000))

	t.Run("parse and resolve file mention", func(t *testing.T) {
		ctx := context.Background()
		input := "Please review @file[test.go] for issues"

		result, err := parser.ParseAndResolve(ctx, input)
		require.NoError(t, err)

		assert.Equal(t, input, result.OriginalText)
		assert.NotEqual(t, input, result.ProcessedText)
		assert.Contains(t, result.ProcessedText, "package main")
		assert.Equal(t, 1, len(result.Contexts))
		assert.Equal(t, MentionTypeFile, result.Contexts[0].Type)
	})

	t.Run("parse with no mentions", func(t *testing.T) {
		ctx := context.Background()
		input := "This text has no mentions"

		result, err := parser.ParseAndResolve(ctx, input)
		require.NoError(t, err)

		assert.Equal(t, input, result.OriginalText)
		assert.Equal(t, input, result.ProcessedText)
		assert.Equal(t, 0, len(result.Contexts))
	})

	t.Run("parse with unknown handler", func(t *testing.T) {
		ctx := context.Background()
		input := "Check @unknown[something]"

		result, err := parser.ParseAndResolve(ctx, input)
		require.NoError(t, err)

		// Should still return result, just skip unknown mentions
		assert.Equal(t, input, result.OriginalText)
	})

	t.Run("parse with failed resolution", func(t *testing.T) {
		ctx := context.Background()
		input := "Check @file[nonexistent.go]"

		_, err := parser.ParseAndResolve(ctx, input)
		// Should error on failed resolution
		assert.Error(t, err)
	})
}

// TestHandlerTypeMethods tests Type() methods for all handlers
func TestHandlerTypeMethods(t *testing.T) {
	t.Run("FileMentionHandler Type", func(t *testing.T) {
		handler := NewFileMentionHandler(t.TempDir())
		assert.Equal(t, MentionTypeFile, handler.Type())
	})

	t.Run("FolderMentionHandler Type", func(t *testing.T) {
		handler := NewFolderMentionHandler(t.TempDir(), 8000)
		assert.Equal(t, MentionTypeFolder, handler.Type())
	})

	t.Run("GitMentionHandler Type", func(t *testing.T) {
		handler := NewGitMentionHandler(".")
		assert.Equal(t, MentionTypeGitChanges, handler.Type())
	})

	t.Run("TerminalMentionHandler Type", func(t *testing.T) {
		handler := NewTerminalMentionHandler()
		assert.Equal(t, MentionTypeTerminal, handler.Type())
	})

	t.Run("ProblemsMentionHandler Type", func(t *testing.T) {
		handler := NewProblemsMentionHandler()
		assert.Equal(t, MentionTypeProblems, handler.Type())
	})

	t.Run("URLMentionHandler Type", func(t *testing.T) {
		handler := NewURLMentionHandler()
		assert.Equal(t, MentionTypeURL, handler.Type())
	})
}

// TestHandlerCanHandle tests CanHandle() methods
func TestHandlerCanHandle(t *testing.T) {
	t.Run("FileMentionHandler CanHandle", func(t *testing.T) {
		handler := NewFileMentionHandler(t.TempDir())
		assert.True(t, handler.CanHandle("@file[test.go]"))
		assert.True(t, handler.CanHandle("@file(test.go)"))
		assert.False(t, handler.CanHandle("@folder[src]"))
	})

	t.Run("FolderMentionHandler CanHandle", func(t *testing.T) {
		handler := NewFolderMentionHandler(t.TempDir(), 8000)
		assert.True(t, handler.CanHandle("@folder[src]"))
		assert.True(t, handler.CanHandle("@folder(src)"))
		assert.False(t, handler.CanHandle("@file[test.go]"))
	})

	t.Run("GitMentionHandler CanHandle", func(t *testing.T) {
		handler := NewGitMentionHandler(".")
		assert.True(t, handler.CanHandle("@git-changes"))
		assert.True(t, handler.CanHandle("@[something]"))
		assert.False(t, handler.CanHandle("@file[test.go]"))
	})

	t.Run("TerminalMentionHandler CanHandle", func(t *testing.T) {
		handler := NewTerminalMentionHandler()
		assert.True(t, handler.CanHandle("@terminal"))
		assert.True(t, handler.CanHandle("@terminal[]"))
		assert.False(t, handler.CanHandle("@file[test.go]"))
	})

	t.Run("ProblemsMentionHandler CanHandle", func(t *testing.T) {
		handler := NewProblemsMentionHandler()
		assert.True(t, handler.CanHandle("@problems"))
		assert.True(t, handler.CanHandle("@problems[]"))
		assert.False(t, handler.CanHandle("@file[test.go]"))
	})

	t.Run("URLMentionHandler CanHandle", func(t *testing.T) {
		handler := NewURLMentionHandler()
		assert.True(t, handler.CanHandle("@url[https://example.com]"))
		assert.True(t, handler.CanHandle("@url(https://example.com)"))
		assert.False(t, handler.CanHandle("@file[test.go]"))
	})
}

// TestFileMentionSearchFiles tests SearchFiles functionality
func TestFileMentionSearchFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "util.go"), []byte("test"), 0644))

	handler := NewFileMentionHandler(tmpDir)

	t.Run("search with results", func(t *testing.T) {
		matches := handler.SearchFiles("main", 5)
		assert.Greater(t, len(matches), 0)
		assert.Contains(t, matches[0].Path, "main.go")
	})

	t.Run("search with no results", func(t *testing.T) {
		matches := handler.SearchFiles("nonexistent", 5)
		assert.Equal(t, 0, len(matches))
	})
}

// TestFuzzySearchRefreshCache tests RefreshCache functionality
func TestFuzzySearchRefreshCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte("test"), 0644))

	fs := NewFuzzySearch(tmpDir)

	// Verify initial search works
	matches := fs.Search("file1", 5)
	assert.Greater(t, len(matches), 0)

	// Add new file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("test"), 0644))

	// Refresh cache
	fs.RefreshCache()

	// Verify new file is found
	matches = fs.Search("file2", 5)
	assert.Greater(t, len(matches), 0, "New file should be found after cache refresh")
}

// TestProblemsMentionHandlerMethods tests ClearProblems and SetProblems
func TestProblemsMentionHandlerMethods(t *testing.T) {
	handler := NewProblemsMentionHandler()

	t.Run("SetProblems", func(t *testing.T) {
		problems := []Problem{
			{Type: "error", File: "main.go", Line: 10, Message: "test error"},
			{Type: "warning", File: "util.go", Line: 20, Message: "test warning"},
		}

		handler.SetProblems(problems)

		ctx := context.Background()
		result, err := handler.Resolve(ctx, "", nil)
		require.NoError(t, err)
		assert.Contains(t, result.Content, "test error")
		assert.Contains(t, result.Content, "test warning")
	})

	t.Run("ClearProblems", func(t *testing.T) {
		handler.AddProblem(Problem{Type: "error", File: "test.go", Line: 1, Message: "error"})

		handler.ClearProblems()

		ctx := context.Background()
		result, err := handler.Resolve(ctx, "", nil)
		require.NoError(t, err)
		assert.Equal(t, 0, result.Metadata["error_count"])
		assert.Equal(t, 0, result.Metadata["warning_count"])
	})
}

// TestTerminalMentionHandlerAddOutput tests AddOutput with truncation
func TestTerminalMentionHandlerAddOutput(t *testing.T) {
	handler := NewTerminalMentionHandler()

	t.Run("add multiple outputs", func(t *testing.T) {
		handler.AddOutput("line 1")
		handler.AddOutput("line 2")
		handler.AddOutput("line 3")

		ctx := context.Background()
		result, err := handler.Resolve(ctx, "", nil)
		require.NoError(t, err)
		assert.Contains(t, result.Content, "line 1")
		assert.Contains(t, result.Content, "line 2")
		assert.Contains(t, result.Content, "line 3")
	})

	t.Run("output truncation", func(t *testing.T) {
		handler := NewTerminalMentionHandler()

		// Add more than max lines (default is 1000)
		for i := 0; i < 1100; i++ {
			handler.AddOutput("test line")
		}

		ctx := context.Background()
		result, err := handler.Resolve(ctx, "", nil)
		require.NoError(t, err)
		// Should have truncated to 1000 lines
		assert.NotNil(t, result.Content)
	})
}

// TestURLMentionHandler tests URL mention handler
func TestURLMentionHandler(t *testing.T) {
	t.Run("ClearCache", func(t *testing.T) {
		handler := NewURLMentionHandler()
		// Just verify it doesn't panic
		handler.ClearCache()
	})

	t.Run("Resolve with mock server", func(t *testing.T) {
		// Create mock HTTP server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Test content"))
		}))
		defer ts.Close()

		handler := NewURLMentionHandler()
		ctx := context.Background()

		result, err := handler.Resolve(ctx, ts.URL, nil)
		require.NoError(t, err)
		assert.Equal(t, MentionTypeURL, result.Type)
		assert.Contains(t, result.Content, "Test content")
		assert.Greater(t, result.TokenCount, 0)
	})

	t.Run("Resolve with HTML content", func(t *testing.T) {
		// Create mock HTTP server with HTML
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<html><head><title>Test</title></head><body><h1>Test Page</h1><p>Content</p></body></html>"))
		}))
		defer ts.Close()

		handler := NewURLMentionHandler()
		ctx := context.Background()

		result, err := handler.Resolve(ctx, ts.URL, nil)
		require.NoError(t, err)
		assert.Equal(t, MentionTypeURL, result.Type)
		assert.NotEmpty(t, result.Content)
	})

	t.Run("Resolve with JSON content", func(t *testing.T) {
		// Create mock HTTP server with JSON
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"key":"value"}`))
		}))
		defer ts.Close()

		handler := NewURLMentionHandler()
		ctx := context.Background()

		result, err := handler.Resolve(ctx, ts.URL, nil)
		require.NoError(t, err)
		assert.Equal(t, MentionTypeURL, result.Type)
		assert.Contains(t, result.Content, "key")
	})

	t.Run("Resolve adds https prefix", func(t *testing.T) {
		// This will fail to resolve but tests the prefix logic
		handler := NewURLMentionHandler()
		ctx := context.Background()

		_, err := handler.Resolve(ctx, "invalid-url-that-does-not-exist.example.invalid", nil)
		// Should error but the URL should have been prefixed
		assert.Error(t, err)
	})

	t.Run("Resolve empty URL error", func(t *testing.T) {
		handler := NewURLMentionHandler()
		ctx := context.Background()

		_, err := handler.Resolve(ctx, "", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("Resolve HTTP error status", func(t *testing.T) {
		// Create mock HTTP server that returns 404
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		handler := NewURLMentionHandler()
		ctx := context.Background()

		_, err := handler.Resolve(ctx, ts.URL, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP error")
	})

	t.Run("Resolve cache works", func(t *testing.T) {
		callCount := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Test content"))
		}))
		defer ts.Close()

		handler := NewURLMentionHandler()
		ctx := context.Background()

		// First call
		result1, err1 := handler.Resolve(ctx, ts.URL, nil)
		require.NoError(t, err1)

		// Second call should use cache
		result2, err2 := handler.Resolve(ctx, ts.URL, nil)
		require.NoError(t, err2)

		assert.Equal(t, result1.Content, result2.Content)
		assert.Equal(t, 1, callCount, "Should only make one HTTP request due to caching")
	})
}

// TestParserExtractMentionInfo tests ExtractMentionInfo
func TestParserExtractMentionInfo(t *testing.T) {
	parser := NewMentionParser()

	t.Run("file mention", func(t *testing.T) {
		mentionType, target, options := parser.ExtractMentionInfo("@file[main.go]")
		assert.Equal(t, "file", mentionType)
		assert.Equal(t, "main.go", target)
		assert.NotNil(t, options)
	})

	t.Run("mention with options", func(t *testing.T) {
		mentionType, target, options := parser.ExtractMentionInfo("@folder[src](recursive=true,content=false)")
		assert.Equal(t, "folder", mentionType)
		assert.Equal(t, "src", target)
		assert.Equal(t, "true", options["recursive"])
		assert.Equal(t, "false", options["content"])
	})

	t.Run("mention without target", func(t *testing.T) {
		mentionType, target, options := parser.ExtractMentionInfo("@git")
		assert.Equal(t, "git", mentionType)
		assert.Equal(t, "", target)
		assert.NotNil(t, options)
	})

	t.Run("URL mention", func(t *testing.T) {
		mentionType, target, options := parser.ExtractMentionInfo("@url[https://example.com]")
		assert.Equal(t, "url", mentionType)
		assert.Equal(t, "https://example.com", target)
		assert.NotNil(t, options)
	})
}

// ========================================
// Additional Coverage Tests
// ========================================

func TestNewFolderMentionHandler_DefaultMaxTokens(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("with zero maxTokens uses default", func(t *testing.T) {
		handler := NewFolderMentionHandler(tmpDir, 0)
		assert.NotNil(t, handler)
		assert.Equal(t, 8000, handler.maxTokens, "Should use default of 8000 when maxTokens is 0")
	})

	t.Run("with explicit maxTokens", func(t *testing.T) {
		handler := NewFolderMentionHandler(tmpDir, 5000)
		assert.NotNil(t, handler)
		assert.Equal(t, 5000, handler.maxTokens)
	})
}

func TestFolderMentionHandler_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure with edge cases
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".hidden"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "node_modules"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "vendor"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".git"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "dist"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "build"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "bin"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "valid"), 0755))

	// Add hidden file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hiddenfile"), []byte("hidden"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hidden", "file.txt"), []byte("in hidden dir"), 0644))

	// Add files in excluded directories
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "node_modules", "dep.js"), []byte("dependency"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "vendor", "lib.go"), []byte("vendor lib"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("git config"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dist", "bundle.js"), []byte("built file"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "build", "output"), []byte("build output"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bin", "binary"), []byte("binary"), 0755))

	// Add valid file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "valid", "file.txt"), []byte("valid content"), 0644))

	handler := NewFolderMentionHandler(tmpDir, 8000)
	ctx := context.Background()

	t.Run("skips hidden files and directories", func(t *testing.T) {
		result, err := handler.Resolve(ctx, ".", map[string]string{"recursive": "true"})

		require.NoError(t, err)
		assert.NotContains(t, result.Content, ".hiddenfile", "Should skip hidden files")
		assert.NotContains(t, result.Content, ".hidden", "Should skip hidden directories")
	})

	t.Run("skips node_modules directory", func(t *testing.T) {
		result, err := handler.Resolve(ctx, ".", map[string]string{"recursive": "true"})

		require.NoError(t, err)
		assert.NotContains(t, result.Content, "node_modules", "Should skip node_modules")
		assert.NotContains(t, result.Content, "dep.js", "Should skip files in node_modules")
	})

	t.Run("skips vendor directory", func(t *testing.T) {
		result, err := handler.Resolve(ctx, ".", map[string]string{"recursive": "true"})

		require.NoError(t, err)
		assert.NotContains(t, result.Content, "vendor", "Should skip vendor")
		assert.NotContains(t, result.Content, "lib.go", "Should skip files in vendor")
	})

	t.Run("skips .git directory", func(t *testing.T) {
		result, err := handler.Resolve(ctx, ".", map[string]string{"recursive": "true"})

		require.NoError(t, err)
		assert.NotContains(t, result.Content, ".git", "Should skip .git")
	})

	t.Run("skips dist, build, bin directories", func(t *testing.T) {
		result, err := handler.Resolve(ctx, ".", map[string]string{"recursive": "true"})

		require.NoError(t, err)
		assert.NotContains(t, result.Content, "dist", "Should skip dist")
		assert.NotContains(t, result.Content, "build", "Should skip build")
		assert.NotContains(t, result.Content, "bin", "Should skip bin")
		assert.NotContains(t, result.Content, "bundle.js", "Should skip files in dist")
		assert.NotContains(t, result.Content, "output", "Should skip files in build")
		assert.NotContains(t, result.Content, "binary", "Should skip files in bin")
	})

	t.Run("includes valid directories", func(t *testing.T) {
		result, err := handler.Resolve(ctx, ".", map[string]string{"recursive": "true"})

		require.NoError(t, err)
		assert.Contains(t, result.Content, "valid", "Should include valid directory")
		assert.Contains(t, result.Content, "file.txt", "Should include files in valid directory")
	})

	t.Run("target is file not directory", func(t *testing.T) {
		// Create a file to test error case
		testFile := filepath.Join(tmpDir, "notadir.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		_, err := handler.Resolve(ctx, "notadir.txt", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})

	t.Run("token limit with content inclusion", func(t *testing.T) {
		// Create handler with very small token limit
		smallHandler := NewFolderMentionHandler(tmpDir, 10)

		// Create a file with enough content to exceed limit
		largeContent := strings.Repeat("x", 100)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(largeContent), 0644))

		result, err := smallHandler.Resolve(ctx, ".", map[string]string{"content": "true", "recursive": "false"})

		require.NoError(t, err)
		// Should contain message about token limit
		assert.Contains(t, result.Content, "token limit reached", "Should indicate token limit was reached")
	})
}

func TestFuzzySearch_BuildCache_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// Create structure with hidden files and excluded directories
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".hidden"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "node_modules"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "vendor"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".git"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "dist"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "build"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "bin"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "valid"), 0755))

	// Add hidden files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hiddenfile"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hidden", "file.go"), []byte("test"), 0644))

	// Add files in excluded directories
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "node_modules", "package.json"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "vendor", "lib.go"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dist", "app.js"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "build", "output"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bin", "binary"), []byte("test"), 0755))

	// Add valid files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "valid", "test.go"), []byte("test"), 0644))

	fs := NewFuzzySearch(tmpDir)

	t.Run("excludes hidden files from cache", func(t *testing.T) {
		matches := fs.Search("hiddenfile", 10)
		assert.Equal(t, 0, len(matches), "Hidden files should not be in cache")
	})

	t.Run("excludes node_modules from cache", func(t *testing.T) {
		matches := fs.Search("package.json", 10)
		assert.Equal(t, 0, len(matches), "node_modules files should not be in cache")
	})

	t.Run("excludes vendor from cache", func(t *testing.T) {
		matches := fs.Search("lib.go", 10)
		// Note: main.go might match "lib.go" fuzzy search, but vendor/lib.go should not be in cache
		for _, match := range matches {
			assert.NotContains(t, match.Path, "vendor", "vendor files should not be in cache")
		}
	})

	t.Run("excludes .git from cache", func(t *testing.T) {
		matches := fs.Search("config", 10)
		for _, match := range matches {
			assert.NotContains(t, match.Path, ".git", ".git files should not be in cache")
		}
	})

	t.Run("excludes dist, build, bin from cache", func(t *testing.T) {
		matches := fs.Search("app.js", 10)
		for _, match := range matches {
			assert.NotContains(t, match.Path, "dist", "dist files should not be in cache")
		}

		matches = fs.Search("output", 10)
		for _, match := range matches {
			assert.NotContains(t, match.Path, "build", "build files should not be in cache")
		}

		matches = fs.Search("binary", 10)
		for _, match := range matches {
			assert.NotContains(t, match.Path, "bin", "bin files should not be in cache")
		}
	})

	t.Run("includes valid files in cache", func(t *testing.T) {
		matches := fs.Search("main.go", 10)
		require.Greater(t, len(matches), 0, "Valid files should be in cache")
		assert.Contains(t, matches[0].Path, "main.go")
	})

	t.Run("RefreshCache rebuilds cache", func(t *testing.T) {
		initialCount := len(fs.fileCache)

		// Add a new file
		newFile := filepath.Join(tmpDir, "newfile.go")
		require.NoError(t, os.WriteFile(newFile, []byte("test"), 0644))

		// Refresh cache
		fs.RefreshCache()

		// New file should be findable
		matches := fs.Search("newfile.go", 10)
		require.Greater(t, len(matches), 0, "New file should be in refreshed cache")
		assert.Contains(t, matches[0].Path, "newfile.go")

		// Cache should have more files now
		assert.Greater(t, len(fs.fileCache), initialCount, "Cache should have grown after refresh")
	})
}

func TestFileMentionHandler_ErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	t.Run("fuzzy search finds file", func(t *testing.T) {
		// Create a file in subdirectory first
		testDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.Mkdir(testDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "myfile.txt"), []byte("content"), 0644))

		// Create handler AFTER file exists so it's in the cache
		handler := NewFileMentionHandler(tmpDir)

		// Search with the exact filename - fuzzy search will find it
		result, err := handler.Resolve(ctx, "myfile.txt", nil)

		require.NoError(t, err)
		assert.Equal(t, "content", result.Content)
	})

	t.Run("absolute path resolution", func(t *testing.T) {
		absFile := filepath.Join(tmpDir, "absolute.txt")
		require.NoError(t, os.WriteFile(absFile, []byte("abs content"), 0644))

		handler := NewFileMentionHandler(tmpDir)
		result, err := handler.Resolve(ctx, absFile, nil)

		require.NoError(t, err)
		assert.Equal(t, "abs content", result.Content)
	})
}

func TestURLMentionHandler_ExtractHTMLContent(t *testing.T) {
	handler := NewURLMentionHandler()

	t.Run("extracts text from HTML", func(t *testing.T) {
		html := `<html>
			<head><title>Test Page</title></head>
			<body>
				<h1>Hello World</h1>
				<p>This is a test paragraph.</p>
				<script>alert('test');</script>
				<style>body { color: red; }</style>
			</body>
		</html>`

		content, metadata := handler.extractHTMLContent(html)

		assert.Contains(t, content, "Hello World")
		assert.Contains(t, content, "This is a test paragraph")
		assert.NotContains(t, content, "alert", "Should not include script content")
		assert.NotContains(t, content, "color: red", "Should not include style content")
		assert.Contains(t, metadata["title"], "Test Page")
	})

	t.Run("handles HTML without title", func(t *testing.T) {
		html := `<html><body><p>No title here</p></body></html>`

		content, metadata := handler.extractHTMLContent(html)

		assert.Contains(t, content, "No title here")
		// Title may be nil or empty string when not present
		if title, ok := metadata["title"]; ok {
			assert.Equal(t, "", title)
		}
	})

	t.Run("handles plain text", func(t *testing.T) {
		plainText := "Just plain text, no HTML tags"

		content, metadata := handler.extractHTMLContent(plainText)

		assert.Contains(t, content, "Just plain text")
		assert.NotNil(t, metadata)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		html := ""

		content, metadata := handler.extractHTMLContent(html)

		assert.Equal(t, "", content)
		assert.NotNil(t, metadata)
	})
}

func TestParseAndResolve_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	parser := NewMentionParser()
	parser.RegisterHandler(NewFileMentionHandler(tmpDir))
	parser.RegisterHandler(NewFolderMentionHandler(tmpDir, 8000))

	ctx := context.Background()

	t.Run("handles mention resolution errors", func(t *testing.T) {
		// Try to resolve a file that doesn't exist
		input := "@file[nonexistent_file_xyz.txt] should fail"

		result, err := parser.ParseAndResolve(ctx, input)

		// Should return error for failed resolution
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("resolves valid mentions", func(t *testing.T) {
		// Create a valid file
		testFile := filepath.Join(tmpDir, "validfile.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

		input := "@file[validfile.txt] should work"

		result, err := parser.ParseAndResolve(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, len(result.Contexts))
		assert.Contains(t, result.Contexts[0].Content, "test content")
	})
}
