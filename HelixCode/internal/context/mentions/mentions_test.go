package mentions

import (
	"context"
	"os"
	"path/filepath"
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
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	handler := NewGitMentionHandler(".")

	t.Run("git changes", func(t *testing.T) {
		ctx := context.Background()
		result, err := handler.Resolve(ctx, "", nil)

		require.NoError(t, err)
		assert.Equal(t, MentionTypeGitChanges, result.Type)
		// Content might be empty if no changes
		assert.NotNil(t, result.Content)
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
}
