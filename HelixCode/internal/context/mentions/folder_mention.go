package mentions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FolderMentionHandler handles @folder mentions
type FolderMentionHandler struct {
	workspaceRoot string
	maxTokens     int
}

// NewFolderMentionHandler creates a new folder mention handler
func NewFolderMentionHandler(workspaceRoot string, maxTokens int) *FolderMentionHandler {
	if maxTokens == 0 {
		maxTokens = 8000 // Default max tokens
	}
	return &FolderMentionHandler{
		workspaceRoot: workspaceRoot,
		maxTokens:     maxTokens,
	}
}

// Type returns the mention type
func (h *FolderMentionHandler) Type() MentionType {
	return MentionTypeFolder
}

// CanHandle checks if this handler can handle the mention
func (h *FolderMentionHandler) CanHandle(mention string) bool {
	return strings.HasPrefix(mention, "@folder[") || strings.HasPrefix(mention, "@folder(")
}

// Resolve resolves the folder mention and returns directory listing
func (h *FolderMentionHandler) Resolve(ctx context.Context, target string, options map[string]string) (*MentionContext, error) {
	if target == "" {
		return nil, fmt.Errorf("folder target cannot be empty")
	}

	// Resolve folder path
	folderPath := target
	if !filepath.IsAbs(folderPath) {
		folderPath = filepath.Join(h.workspaceRoot, target)
	}

	// Check if folder exists
	folderInfo, err := os.Stat(folderPath)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %s", target)
	}
	if !folderInfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", target)
	}

	// Parse options
	recursive := options["recursive"] == "true"
	includeContent := options["content"] == "true"

	// Build folder content
	var content strings.Builder
	var fileCount int
	var totalSize int64
	tokenCount := 0

	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip if not recursive and not in root folder (but don't skip root itself)
		if !recursive && path != folderPath && filepath.Dir(path) != folderPath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files/folders (but not the root folder itself)
		if path != folderPath && strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common ignore patterns
		if info.IsDir() {
			name := filepath.Base(path)
			if name == "node_modules" || name == "vendor" || name == ".git" ||
				name == "dist" || name == "build" || name == "bin" {
				return filepath.SkipDir
			}
		}

		// Get relative path
		relPath, _ := filepath.Rel(folderPath, path)
		if relPath == "." {
			return nil // Continue walking but don't add root folder itself
		}

		// Add to content
		if info.IsDir() {
			content.WriteString(fmt.Sprintf("📁 %s/\n", relPath))
		} else {
			fileCount++
			totalSize += info.Size()

			// Add file info
			content.WriteString(fmt.Sprintf("📄 %s (%d bytes)\n", relPath, info.Size()))

			// Include file content if requested and within token limit
			if includeContent && tokenCount < h.maxTokens {
				fileContent, err := os.ReadFile(path)
				if err == nil {
					// Check if adding this file would exceed token limit
					fileTokens := len(fileContent) / 4
					if tokenCount+fileTokens <= h.maxTokens {
						content.WriteString(fmt.Sprintf("   Content:\n   ```\n   %s\n   ```\n", string(fileContent)))
						tokenCount += fileTokens
					} else {
						content.WriteString("   (Content omitted - token limit reached)\n")
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk folder: %w", err)
	}

	// Get relative path for display
	relPath, _ := filepath.Rel(h.workspaceRoot, folderPath)
	if relPath == "" {
		relPath = filepath.Base(folderPath)
	}

	// If no content tokens were counted, estimate from listing
	if tokenCount == 0 {
		tokenCount = len(content.String()) / 4
	}

	mentionCtx := &MentionContext{
		Type:       MentionTypeFolder,
		Target:     relPath,
		Content:    content.String(),
		TokenCount: tokenCount,
		Metadata: map[string]interface{}{
			"full_path":       folderPath,
			"file_count":      fileCount,
			"total_size":      totalSize,
			"recursive":       recursive,
			"include_content": includeContent,
		},
		ResolvedAt: time.Now(),
	}

	return mentionCtx, nil
}
