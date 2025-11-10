package mentions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileMentionHandler handles @file mentions
type FileMentionHandler struct {
	workspaceRoot string
	fuzzySearch   *FuzzySearch
}

// NewFileMentionHandler creates a new file mention handler
func NewFileMentionHandler(workspaceRoot string) *FileMentionHandler {
	return &FileMentionHandler{
		workspaceRoot: workspaceRoot,
		fuzzySearch:   NewFuzzySearch(workspaceRoot),
	}
}

// Type returns the mention type
func (h *FileMentionHandler) Type() MentionType {
	return MentionTypeFile
}

// CanHandle checks if this handler can handle the mention
func (h *FileMentionHandler) CanHandle(mention string) bool {
	return strings.HasPrefix(mention, "@file[") || strings.HasPrefix(mention, "@file(")
}

// Resolve resolves the file mention and returns its content
func (h *FileMentionHandler) Resolve(ctx context.Context, target string, options map[string]string) (*MentionContext, error) {
	if target == "" {
		return nil, fmt.Errorf("file target cannot be empty")
	}

	// Check if fuzzy search is needed
	filePath := target
	if !filepath.IsAbs(filePath) {
		// Try exact match first
		absPath := filepath.Join(h.workspaceRoot, filePath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			// Try fuzzy search
			matches := h.fuzzySearch.Search(target, 1)
			if len(matches) == 0 {
				return nil, fmt.Errorf("file not found: %s", target)
			}
			filePath = matches[0].Path
		} else {
			filePath = absPath
		}
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate token count (rough estimate: 1 token ≈ 4 characters)
	tokenCount := len(content) / 4

	// Get relative path for display
	relPath, _ := filepath.Rel(h.workspaceRoot, filePath)
	if relPath == "" {
		relPath = filepath.Base(filePath)
	}

	mentionCtx := &MentionContext{
		Type:       MentionTypeFile,
		Target:     relPath,
		Content:    string(content),
		TokenCount: tokenCount,
		Metadata: map[string]interface{}{
			"full_path": filePath,
			"size":      fileInfo.Size(),
			"modified":  fileInfo.ModTime(),
			"extension": filepath.Ext(filePath),
		},
		ResolvedAt: time.Now(),
	}

	return mentionCtx, nil
}

// SearchFiles performs a fuzzy search for files
func (h *FileMentionHandler) SearchFiles(query string, limit int) []FuzzyMatch {
	return h.fuzzySearch.Search(query, limit)
}
