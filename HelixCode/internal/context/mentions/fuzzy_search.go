package mentions

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FuzzyMatch represents a fuzzy search match
type FuzzyMatch struct {
	Path  string
	Score int
}

// FuzzySearch provides fuzzy file searching
type FuzzySearch struct {
	workspaceRoot string
	fileCache     []string
}

// NewFuzzySearch creates a new fuzzy search instance
func NewFuzzySearch(workspaceRoot string) *FuzzySearch {
	fs := &FuzzySearch{
		workspaceRoot: workspaceRoot,
		fileCache:     make([]string, 0),
	}
	fs.buildCache()
	return fs
}

// buildCache builds the file cache
func (fs *FuzzySearch) buildCache() {
	filepath.Walk(fs.workspaceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden files and directories
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip node_modules, vendor, etc.
		if info.IsDir() {
			name := filepath.Base(path)
			if name == "node_modules" || name == "vendor" || name == ".git" ||
				name == "dist" || name == "build" || name == "bin" {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() {
			fs.fileCache = append(fs.fileCache, path)
		}

		return nil
	})
}

// Search performs fuzzy search on files
func (fs *FuzzySearch) Search(query string, limit int) []FuzzyMatch {
	query = strings.ToLower(query)
	matches := make([]FuzzyMatch, 0)

	for _, path := range fs.fileCache {
		score := fs.calculateScore(path, query)
		if score > 0 {
			matches = append(matches, FuzzyMatch{
				Path:  path,
				Score: score,
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Return top N matches
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return matches
}

// calculateScore calculates the fuzzy match score
func (fs *FuzzySearch) calculateScore(path, query string) int {
	path = strings.ToLower(path)
	filename := strings.ToLower(filepath.Base(path))

	// Exact match in filename gets highest score
	if filename == query {
		return 1000
	}

	score := 0
	hasMatch := false

	// Exact match in path
	if strings.Contains(path, query) {
		score += 500
		hasMatch = true
	}

	// Filename contains query
	if strings.Contains(filename, query) {
		score += 300
		hasMatch = true
	}

	// Check for sequential character matches
	queryChars := []rune(query)
	pathChars := []rune(filename)

	queryIdx := 0
	sequentialMatches := 0
	for i := 0; i < len(pathChars) && queryIdx < len(queryChars); i++ {
		if pathChars[i] == queryChars[queryIdx] {
			sequentialMatches++
			queryIdx++
		}
	}

	// All query characters found sequentially
	if queryIdx == len(queryChars) {
		score += 100 + (sequentialMatches * 10)
		hasMatch = true
	}

	// Check for word boundary matches (e.g., "fm" matches "file_mention.go")
	words := strings.FieldsFunc(filename, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == '/'
	})

	for _, word := range words {
		if strings.HasPrefix(word, query) {
			score += 200
			hasMatch = true
			break
		}
	}

	// Only return score if we actually have a meaningful match
	if !hasMatch {
		return 0
	}

	return score
}

// RefreshCache rebuilds the file cache
func (fs *FuzzySearch) RefreshCache() {
	fs.fileCache = make([]string, 0)
	fs.buildCache()
}
