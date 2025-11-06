package snapshots

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RestoreSnapshot restores the workspace to a specific snapshot
func (m *Manager) RestoreSnapshot(ctx context.Context, snapshotID string, opts *RestoreOptions) (*RestoreResult, error) {
	if opts == nil {
		opts = &RestoreOptions{
			CreateBackup: true,
		}
	}

	startTime := time.Now()
	result := &RestoreResult{
		Success:       false,
		FilesRestored: []string{},
		ConflictFiles: []string{},
		Errors:        []string{},
	}

	// Load snapshot
	snapshot, err := m.metadataStore.Load(ctx, snapshotID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to load snapshot: %v", err))
		return result, err
	}

	// Verify snapshot exists
	if !m.verifySnapshot(ctx, snapshot) {
		err := fmt.Errorf("snapshot not found in git stash: %s", snapshotID)
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Check for uncommitted changes (unless force is set)
	if !opts.Force && !opts.DryRun {
		hasChanges, err := m.hasUncommittedChanges(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to check for changes: %v", err))
			return result, err
		}
		if hasChanges {
			err := fmt.Errorf("workspace has uncommitted changes, use --force to override or commit changes first")
			result.Errors = append(result.Errors, err.Error())
			return result, err
		}
	}

	// Create backup if requested
	if opts.CreateBackup && !opts.DryRun {
		backupOpts := &CreateOptions{
			Description:      fmt.Sprintf("Backup before restoring to %s", snapshotID),
			Tags:             []string{"backup"},
			IncludeUntracked: true,
		}
		backup, err := m.CreateSnapshot(ctx, backupOpts)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create backup: %v", err))
			// Continue anyway - backup failure shouldn't prevent restore
		} else {
			result.BackupSnapshot = backup
		}
	}

	// Dry run - just show what would be restored
	if opts.DryRun {
		files, err := m.GetSnapshotFiles(ctx, snapshotID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to get snapshot files: %v", err))
			return result, err
		}
		result.FilesRestored = files
		result.Success = true
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Apply the stash
	args := []string{"stash", "apply"}
	if !opts.KeepIndex {
		args = append(args, "--index")
	}
	args = append(args, snapshot.StashRef)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = m.repoPath
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check for conflicts
	if err != nil || strings.Contains(outputStr, "CONFLICT") {
		// Parse conflicts
		conflicts := m.parseConflicts(outputStr)
		result.ConflictFiles = conflicts

		if len(conflicts) > 0 {
			err := fmt.Errorf("conflicts detected during restore")
			result.Errors = append(result.Errors, err.Error())

			// If we had conflicts, try to abort the merge
			abortCmd := exec.CommandContext(ctx, "git", "reset", "--merge")
			abortCmd.Dir = m.repoPath
			abortCmd.Run()

			result.Duration = time.Since(startTime)
			return result, err
		}

		// If error but no conflicts, return the error
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to apply stash: %v, output: %s", err, outputStr))
			result.Duration = time.Since(startTime)
			return result, err
		}
	}

	// Get list of restored files
	files, err := m.GetSnapshotFiles(ctx, snapshotID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to get restored files: %v", err))
	} else {
		result.FilesRestored = files
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	return result, nil
}

// ValidateRestore checks if restore is safe to perform
func (m *Manager) ValidateRestore(ctx context.Context, snapshotID string) error {
	// Load snapshot
	snapshot, err := m.metadataStore.Load(ctx, snapshotID)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Verify snapshot exists
	if !m.verifySnapshot(ctx, snapshot) {
		return fmt.Errorf("snapshot not found in git stash: %s", snapshotID)
	}

	// Check for uncommitted changes
	hasChanges, err := m.hasUncommittedChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("workspace has uncommitted changes")
	}

	return nil
}

// PreviewRestore shows what would be restored without actually restoring
func (m *Manager) PreviewRestore(ctx context.Context, snapshotID string) (*RestoreResult, error) {
	opts := &RestoreOptions{
		DryRun:       true,
		CreateBackup: false,
	}
	return m.RestoreSnapshot(ctx, snapshotID, opts)
}

// hasUncommittedChanges checks if there are uncommitted changes in the workspace
func (m *Manager) hasUncommittedChanges(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = m.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// parseConflicts parses conflict information from git output
func (m *Manager) parseConflicts(output string) []string {
	conflicts := []string{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "CONFLICT") {
			// Extract file path from conflict message
			// Format: "CONFLICT (content): Merge conflict in <file>"
			if idx := strings.Index(line, " in "); idx != -1 {
				file := strings.TrimSpace(line[idx+4:])
				conflicts = append(conflicts, file)
			}
		}
	}

	return conflicts
}

// RollbackRestore attempts to rollback a failed restore
func (m *Manager) RollbackRestore(ctx context.Context, backupID string) error {
	if backupID == "" {
		return fmt.Errorf("no backup ID provided")
	}

	// Simply restore to the backup
	opts := &RestoreOptions{
		CreateBackup: false,
		Force:        true,
	}

	result, err := m.RestoreSnapshot(ctx, backupID, opts)
	if err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("rollback failed: %v", result.Errors)
	}

	return nil
}

// CanRestore checks if a snapshot can be safely restored
func (m *Manager) CanRestore(ctx context.Context, snapshotID string) (bool, error) {
	err := m.ValidateRestore(ctx, snapshotID)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetRestoreConflicts predicts potential conflicts for a restore operation
func (m *Manager) GetRestoreConflicts(ctx context.Context, snapshotID string) ([]string, error) {
	// This is a best-effort prediction - actual conflicts may differ
	// We check if any files in the snapshot have been modified in the workspace

	// Get files in the snapshot
	snapshotFiles, err := m.GetSnapshotFiles(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot files: %w", err)
	}

	// Get current status
	status, err := m.getRepoStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo status: %w", err)
	}

	// Find files that are in both the snapshot and current changes
	conflicts := []string{}
	modifiedMap := make(map[string]bool)

	for _, file := range status.Modified {
		modifiedMap[file] = true
	}
	for _, file := range status.Added {
		modifiedMap[file] = true
	}
	for _, file := range status.Deleted {
		modifiedMap[file] = true
	}

	for _, file := range snapshotFiles {
		if modifiedMap[file] {
			conflicts = append(conflicts, file)
		}
	}

	return conflicts, nil
}
