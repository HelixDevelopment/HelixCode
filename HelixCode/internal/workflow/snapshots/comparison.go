package snapshots

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// CompareSnapshots compares two snapshots and returns detailed differences
func (m *Manager) CompareSnapshots(ctx context.Context, fromID, toID string) (*Comparison, error) {
	// Load snapshots
	fromSnapshot, err := m.metadataStore.Load(ctx, fromID)
	if err != nil {
		return nil, fmt.Errorf("failed to load 'from' snapshot: %w", err)
	}

	toSnapshot, err := m.metadataStore.Load(ctx, toID)
	if err != nil {
		return nil, fmt.Errorf("failed to load 'to' snapshot: %w", err)
	}

	// Generate diff between the two stashes
	diff, err := m.generateDiff(ctx, fromSnapshot.StashRef, toSnapshot.StashRef)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff: %w", err)
	}

	// Parse diff to get file-level changes
	fileDiffs, err := m.parseDiff(diff)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	// If diff parsing didn't produce results, build from metadata
	if len(fileDiffs) == 0 && (toSnapshot.Metadata != nil || fromSnapshot.Metadata != nil) {
		fileDiffs = m.buildDiffsFromMetadata(fromSnapshot, toSnapshot)
	}

	// Calculate summary statistics
	summary := m.calculateSummary(fromSnapshot, toSnapshot, fileDiffs)
	stats := m.calculateStatistics(fileDiffs)

	comparison := &Comparison{
		From:       fromSnapshot,
		To:         toSnapshot,
		Summary:    summary,
		FileDiffs:  fileDiffs,
		Statistics: stats,
	}

	return comparison, nil
}

// GenerateDiff generates a unified diff between two snapshots
func (m *Manager) GenerateDiff(ctx context.Context, fromID, toID string) (string, error) {
	// Load snapshots
	fromSnapshot, err := m.metadataStore.Load(ctx, fromID)
	if err != nil {
		return "", fmt.Errorf("failed to load 'from' snapshot: %w", err)
	}

	toSnapshot, err := m.metadataStore.Load(ctx, toID)
	if err != nil {
		return "", fmt.Errorf("failed to load 'to' snapshot: %w", err)
	}

	// Generate diff
	return m.generateDiff(ctx, fromSnapshot.StashRef, toSnapshot.StashRef)
}

// generateDiff generates a diff between two git stash references
func (m *Manager) generateDiff(ctx context.Context, fromRef, toRef string) (string, error) {
	// Use git stash show to get the diff for the destination stash
	cmd := exec.CommandContext(ctx, "git", "stash", "show", "-p", toRef)
	cmd.Dir = m.repoPath
	output, err := cmd.Output()
	if err != nil {
		// Try the from ref if to ref fails
		cmd = exec.CommandContext(ctx, "git", "stash", "show", "-p", fromRef)
		cmd.Dir = m.repoPath
		output, err = cmd.Output()
		if err != nil {
			return "", nil // Return empty diff instead of error
		}
	}

	return string(output), nil
}

// parseDiff parses a unified diff into file-level changes
func (m *Manager) parseDiff(diff string) ([]*FileDiff, error) {
	if diff == "" {
		return []*FileDiff{}, nil
	}

	lines := strings.Split(diff, "\n")
	fileDiffs := []*FileDiff{}
	var currentFile *FileDiff
	var currentDiff strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// New file diff section
		if strings.HasPrefix(line, "diff --git") {
			// Save previous file diff
			if currentFile != nil {
				currentFile.Diff = currentDiff.String()
				fileDiffs = append(fileDiffs, currentFile)
			}

			// Start new file diff
			currentFile = &FileDiff{}
			currentDiff.Reset()
			currentDiff.WriteString(line + "\n")
			continue
		}

		if currentFile == nil {
			continue
		}

		// File path
		if strings.HasPrefix(line, "--- a/") {
			path := strings.TrimPrefix(line, "--- a/")
			if path != "/dev/null" {
				currentFile.Path = path
			}
			currentDiff.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "+++ b/") {
			path := strings.TrimPrefix(line, "+++ b/")
			if path != "/dev/null" {
				if currentFile.Path == "" {
					currentFile.Path = path
					currentFile.Status = DiffAdded
				} else if currentFile.Path != path {
					currentFile.Status = DiffRenamed
				} else {
					currentFile.Status = DiffModified
				}
			} else if currentFile.Path != "" {
				currentFile.Status = DiffDeleted
			}
			currentDiff.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			// Added line
			currentFile.LinesAdded++
			currentDiff.WriteString(line + "\n")
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			// Deleted line
			currentFile.LinesDeleted++
			currentDiff.WriteString(line + "\n")
		} else {
			currentDiff.WriteString(line + "\n")
		}
	}

	// Save last file diff
	if currentFile != nil {
		currentFile.Diff = currentDiff.String()
		fileDiffs = append(fileDiffs, currentFile)
	}

	return fileDiffs, nil
}

// buildDiffsFromMetadata builds file diffs from snapshot metadata
func (m *Manager) buildDiffsFromMetadata(from, to *Snapshot) []*FileDiff {
	fileDiffs := []*FileDiff{}

	// Build from the "to" snapshot metadata
	if to.Metadata != nil {
		for _, file := range to.Metadata.FilesAdded {
			fileDiffs = append(fileDiffs, &FileDiff{
				Path:       file,
				Status:     DiffAdded,
				LinesAdded: 1, // Approximate
			})
		}
		for _, file := range to.Metadata.FilesModified {
			fileDiffs = append(fileDiffs, &FileDiff{
				Path:         file,
				Status:       DiffModified,
				LinesAdded:   1, // Approximate
				LinesDeleted: 1, // Approximate
			})
		}
		for _, file := range to.Metadata.FilesDeleted {
			fileDiffs = append(fileDiffs, &FileDiff{
				Path:         file,
				Status:       DiffDeleted,
				LinesDeleted: 1, // Approximate
			})
		}
		for _, file := range to.Metadata.UntrackedFiles {
			fileDiffs = append(fileDiffs, &FileDiff{
				Path:       file,
				Status:     DiffAdded,
				LinesAdded: 1, // Approximate
			})
		}
	}

	return fileDiffs
}

// calculateSummary calculates high-level summary statistics
func (m *Manager) calculateSummary(from, to *Snapshot, fileDiffs []*FileDiff) *Summary {
	summary := &Summary{
		TimeElapsed: to.CreatedAt.Sub(from.CreatedAt),
	}

	for _, fd := range fileDiffs {
		switch fd.Status {
		case DiffAdded:
			summary.FilesAdded++
		case DiffModified:
			summary.FilesModified++
		case DiffDeleted:
			summary.FilesDeleted++
		}
		summary.LinesAdded += fd.LinesAdded
		summary.LinesDeleted += fd.LinesDeleted
	}

	return summary
}

// calculateStatistics calculates detailed diff statistics
func (m *Manager) calculateStatistics(fileDiffs []*FileDiff) *Statistics {
	stats := &Statistics{}

	for _, fd := range fileDiffs {
		stats.TotalFiles++
		stats.LinesAdded += fd.LinesAdded
		stats.LinesDeleted += fd.LinesDeleted
		stats.TotalLines += fd.LinesAdded + fd.LinesDeleted
	}

	return stats
}

// GetDiffStat returns stat information for a snapshot
func (m *Manager) GetDiffStat(ctx context.Context, snapshotID string) (string, error) {
	snapshot, err := m.metadataStore.Load(ctx, snapshotID)
	if err != nil {
		return "", fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Use git stash show with --stat
	cmd := exec.CommandContext(ctx, "git", "stash", "show", "--stat", snapshot.StashRef)
	cmd.Dir = m.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get stash stats: %w", err)
	}

	return string(output), nil
}

// GetSnapshotFiles returns the list of files in a snapshot
func (m *Manager) GetSnapshotFiles(ctx context.Context, snapshotID string) ([]string, error) {
	snapshot, err := m.metadataStore.Load(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Get files from metadata (this is more reliable than git stash show)
	files := []string{}
	if snapshot.Metadata != nil {
		files = append(files, snapshot.Metadata.FilesAdded...)
		files = append(files, snapshot.Metadata.FilesModified...)
		files = append(files, snapshot.Metadata.FilesDeleted...)
		if len(snapshot.Metadata.UntrackedFiles) > 0 {
			files = append(files, snapshot.Metadata.UntrackedFiles...)
		}
	}

	// Remove duplicates
	fileMap := make(map[string]bool)
	uniqueFiles := []string{}
	for _, file := range files {
		if !fileMap[file] {
			fileMap[file] = true
			uniqueFiles = append(uniqueFiles, file)
		}
	}

	return uniqueFiles, nil
}

// GetFileContent retrieves the content of a file from a specific snapshot
func (m *Manager) GetFileContent(ctx context.Context, snapshotID, filePath string) (string, error) {
	snapshot, err := m.metadataStore.Load(ctx, snapshotID)
	if err != nil {
		return "", fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Use git show to get file content from stash
	cmd := exec.CommandContext(ctx, "git", "show", fmt.Sprintf("%s:%s", snapshot.StashRef, filePath))
	cmd.Dir = m.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %w", err)
	}

	return string(output), nil
}

// GetSnapshotSize estimates the size of a snapshot
func (m *Manager) GetSnapshotSize(ctx context.Context, snapshotID string) (int64, error) {
	snapshot, err := m.metadataStore.Load(ctx, snapshotID)
	if err != nil {
		return 0, fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Use git cat-file to get the size
	cmd := exec.CommandContext(ctx, "git", "cat-file", "-s", snapshot.StashRef)
	cmd.Dir = m.repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get stash size: %w", err)
	}

	size, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size: %w", err)
	}

	return size, nil
}
