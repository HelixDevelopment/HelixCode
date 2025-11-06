package snapshots

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Manager manages workspace snapshots using git stash
type Manager struct {
	repoPath      string
	metadataStore *MetadataStore
}

// NewManager creates a new snapshot manager
func NewManager(repoPath string) (*Manager, error) {
	// Verify it's a git repository
	if err := verifyGitRepo(repoPath); err != nil {
		return nil, err
	}

	store, err := NewMetadataStore(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata store: %w", err)
	}

	return &Manager{
		repoPath:      repoPath,
		metadataStore: store,
	}, nil
}

// CreateSnapshot creates a new workspace snapshot
func (m *Manager) CreateSnapshot(ctx context.Context, opts *CreateOptions) (*Snapshot, error) {
	if opts == nil {
		opts = &CreateOptions{}
	}

	// Generate snapshot ID
	id := fmt.Sprintf("helix-snapshot-%s", uuid.New().String()[:8])

	// Get current repository status
	status, err := m.getRepoStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo status: %w", err)
	}

	// Build stash message
	description := opts.Description
	if description == "" && opts.AutoGenerate {
		description = m.generateDescription(status)
	}
	if description == "" {
		description = "Snapshot"
	}

	stashMessage := fmt.Sprintf("%s: %s", id, description)

	// Create stash with appropriate flags
	args := []string{"stash", "save"}
	if opts.IncludeUntracked {
		args = append(args, "--include-untracked")
	}
	args = append(args, stashMessage)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = m.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create stash: %w, output: %s", err, string(output))
	}

	// Check if anything was actually stashed
	outputStr := string(output)
	if strings.Contains(outputStr, "No local changes to save") {
		return nil, fmt.Errorf("no changes to snapshot")
	}

	// Get stash reference
	stashRef, stashIndex, err := m.findStashByMessage(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find created stash: %w", err)
	}

	// Get metadata
	metadata, err := m.collectMetadata(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metadata: %w", err)
	}

	// Add custom metadata
	if opts.Metadata != nil {
		if metadata.Custom == nil {
			metadata.Custom = make(map[string]string)
		}
		for k, v := range opts.Metadata {
			metadata.Custom[k] = v
		}
	}

	// Get file count and size
	fileCount := len(status.Modified) + len(status.Added) + len(status.Deleted)
	if opts.IncludeUntracked {
		fileCount += len(status.Untracked)
	}

	// Create snapshot
	snapshot := &Snapshot{
		ID:          id,
		StashRef:    stashRef,
		StashIndex:  stashIndex,
		CreatedAt:   time.Now(),
		Description: description,
		TaskID:      opts.TaskID,
		Status:      StatusActive,
		Metadata:    metadata,
		Tags:        opts.Tags,
		FileCount:   fileCount,
		Size:        0, // We'll calculate this if needed
	}

	// Save metadata
	if err := m.metadataStore.Save(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	return snapshot, nil
}

// GetSnapshot retrieves a snapshot by ID
func (m *Manager) GetSnapshot(ctx context.Context, id string) (*Snapshot, error) {
	return m.metadataStore.Load(ctx, id)
}

// ListSnapshots returns all snapshots, optionally filtered
func (m *Manager) ListSnapshots(ctx context.Context, filter *Filter) ([]*Snapshot, error) {
	// Get snapshots from metadata
	snapshots, err := m.metadataStore.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Verify each snapshot still exists in git stash
	validSnapshots := make([]*Snapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if m.verifySnapshot(ctx, snapshot) {
			validSnapshots = append(validSnapshots, snapshot)
		} else {
			// Mark as corrupted
			snapshot.Status = StatusCorrupted
			m.metadataStore.Save(ctx, snapshot)
		}
	}

	return validSnapshots, nil
}

// DeleteSnapshot removes a snapshot
func (m *Manager) DeleteSnapshot(ctx context.Context, id string) error {
	// Load snapshot
	snapshot, err := m.metadataStore.Load(ctx, id)
	if err != nil {
		return err
	}

	// Drop the stash
	cmd := exec.CommandContext(ctx, "git", "stash", "drop", snapshot.StashRef)
	cmd.Dir = m.repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to drop stash: %w, output: %s", err, string(output))
	}

	// Delete metadata
	return m.metadataStore.Delete(ctx, id)
}

// repoStatus represents the current repository status
type repoStatus struct {
	Branch     string
	CommitHash string
	Modified   []string
	Added      []string
	Deleted    []string
	Untracked  []string
}

// getRepoStatus retrieves current repository status
func (m *Manager) getRepoStatus(ctx context.Context) (*repoStatus, error) {
	status := &repoStatus{}

	// Get current branch
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = m.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	status.Branch = strings.TrimSpace(string(output))

	// Get current commit hash
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = m.repoPath
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}
	status.CommitHash = strings.TrimSpace(string(output))

	// Get status
	cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = m.repoPath
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	// Parse status output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		statusCode := line[:2]
		file := strings.TrimSpace(line[3:])

		switch {
		case strings.HasPrefix(statusCode, "M"):
			status.Modified = append(status.Modified, file)
		case strings.HasPrefix(statusCode, "A"):
			status.Added = append(status.Added, file)
		case strings.HasPrefix(statusCode, "D"):
			status.Deleted = append(status.Deleted, file)
		case strings.HasPrefix(statusCode, "??"):
			status.Untracked = append(status.Untracked, file)
		}
	}

	return status, nil
}

// collectMetadata collects metadata for a snapshot
func (m *Manager) collectMetadata(ctx context.Context, status *repoStatus) (*Metadata, error) {
	metadata := &Metadata{
		WorkingDirectory: m.repoPath,
		Branch:           status.Branch,
		CommitHash:       status.CommitHash,
		FilesAdded:       status.Added,
		FilesModified:    status.Modified,
		FilesDeleted:     status.Deleted,
		UntrackedFiles:   status.Untracked,
		Custom:           make(map[string]string),
	}

	return metadata, nil
}

// generateDescription auto-generates a description based on changes
func (m *Manager) generateDescription(status *repoStatus) string {
	parts := []string{}

	if len(status.Added) > 0 {
		parts = append(parts, fmt.Sprintf("%d added", len(status.Added)))
	}
	if len(status.Modified) > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", len(status.Modified)))
	}
	if len(status.Deleted) > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", len(status.Deleted)))
	}
	if len(status.Untracked) > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", len(status.Untracked)))
	}

	if len(parts) == 0 {
		return "Snapshot"
	}

	return "Changes: " + strings.Join(parts, ", ")
}

// findStashByMessage finds a stash by its message prefix
func (m *Manager) findStashByMessage(ctx context.Context, messagePrefix string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "git", "stash", "list")
	cmd.Dir = m.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", -1, fmt.Errorf("failed to list stashes: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, messagePrefix) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 {
				return parts[0], i, nil
			}
		}
	}

	return "", -1, fmt.Errorf("stash not found with prefix: %s", messagePrefix)
}

// verifySnapshot verifies that a snapshot still exists in git stash
func (m *Manager) verifySnapshot(ctx context.Context, snapshot *Snapshot) bool {
	cmd := exec.CommandContext(ctx, "git", "stash", "list")
	cmd.Dir = m.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), snapshot.ID)
}

// verifyGitRepo verifies that the path is a git repository
func verifyGitRepo(path string) error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository: %s", path)
	}
	return nil
}
