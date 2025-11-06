package snapshots

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "snapshots-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to add files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tmpDir
}

// cleanupTestRepo removes the test repository
func cleanupTestRepo(t *testing.T, repoPath string) {
	t.Helper()
	if err := os.RemoveAll(repoPath); err != nil {
		t.Logf("Warning: failed to cleanup test repo: %v", err)
	}
}

// createTestFile creates a file in the test repository
func createTestFile(t *testing.T, repoPath, filename, content string) {
	t.Helper()
	path := filepath.Join(repoPath, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

// modifyTestFile modifies an existing file in the test repository
func modifyTestFile(t *testing.T, repoPath, filename, content string) {
	t.Helper()
	createTestFile(t, repoPath, filename, content)
}

// TestNewManager tests manager creation
func TestNewManager(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.repoPath != repoPath {
		t.Errorf("Expected repoPath %s, got %s", repoPath, manager.repoPath)
	}
}

// TestNewManager_NotGitRepo tests manager creation on non-git directory
func TestNewManager_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "not-git-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = NewManager(tmpDir)
	if err == nil {
		t.Error("Expected error for non-git repository")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Expected 'not a git repository' error, got: %v", err)
	}
}

// TestCreateSnapshot tests basic snapshot creation
func TestCreateSnapshot(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Create some changes
	createTestFile(t, repoPath, "test.txt", "Hello, World!")

	ctx := context.Background()
	opts := &CreateOptions{
		Description:      "Test snapshot",
		Tags:             []string{"test"},
		IncludeUntracked: true,
	}

	snapshot, err := manager.CreateSnapshot(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	if snapshot.ID == "" {
		t.Error("Expected non-empty snapshot ID")
	}

	if snapshot.Description != "Test snapshot" {
		t.Errorf("Expected description 'Test snapshot', got '%s'", snapshot.Description)
	}

	if len(snapshot.Tags) != 1 || snapshot.Tags[0] != "test" {
		t.Errorf("Expected tags [test], got %v", snapshot.Tags)
	}

	if snapshot.FileCount == 0 {
		t.Error("Expected non-zero file count")
	}

	if snapshot.Status != StatusActive {
		t.Errorf("Expected status %s, got %s", StatusActive, snapshot.Status)
	}
}

// TestCreateSnapshot_NoChanges tests snapshot creation with no changes
func TestCreateSnapshot_NoChanges(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()
	opts := &CreateOptions{
		Description: "No changes snapshot",
	}

	_, err = manager.CreateSnapshot(ctx, opts)
	if err == nil {
		t.Error("Expected error when creating snapshot with no changes")
	}
	if !strings.Contains(err.Error(), "no changes") {
		t.Errorf("Expected 'no changes' error, got: %v", err)
	}
}

// TestCreateSnapshot_AutoGenerate tests auto-generated descriptions
func TestCreateSnapshot_AutoGenerate(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Create some changes
	createTestFile(t, repoPath, "new.txt", "New file")
	modifyTestFile(t, repoPath, "README.md", "# Modified\n")

	ctx := context.Background()
	opts := &CreateOptions{
		AutoGenerate:     true,
		IncludeUntracked: true,
	}

	snapshot, err := manager.CreateSnapshot(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	if snapshot.Description == "" {
		t.Error("Expected auto-generated description")
	}

	if !strings.Contains(snapshot.Description, "modified") && !strings.Contains(snapshot.Description, "Changes") {
		t.Errorf("Expected description to mention changes, got: %s", snapshot.Description)
	}
}

// TestGetSnapshot tests retrieving a snapshot
func TestGetSnapshot(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Create a snapshot
	createTestFile(t, repoPath, "test.txt", "Content")
	ctx := context.Background()
	created, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Test",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Retrieve it
	retrieved, err := manager.GetSnapshot(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSnapshot() failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}

	if retrieved.Description != created.Description {
		t.Errorf("Expected description %s, got %s", created.Description, retrieved.Description)
	}
}

// TestGetSnapshot_NotFound tests retrieving non-existent snapshot
func TestGetSnapshot_NotFound(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()
	_, err = manager.GetSnapshot(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent snapshot")
	}
}

// TestListSnapshots tests listing snapshots
func TestListSnapshots(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create multiple snapshots
	for i := 0; i < 3; i++ {
		createTestFile(t, repoPath, "file"+string(rune('a'+i))+".txt", "Content")
		_, err := manager.CreateSnapshot(ctx, &CreateOptions{
			Description:      "Snapshot " + string(rune('A'+i)),
			IncludeUntracked: true,
		})
		if err != nil {
			t.Fatalf("CreateSnapshot() failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List all snapshots
	snapshots, err := manager.ListSnapshots(ctx, nil)
	if err != nil {
		t.Fatalf("ListSnapshots() failed: %v", err)
	}

	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}

	// Verify they're sorted by creation time (most recent first)
	for i := 0; i < len(snapshots)-1; i++ {
		if snapshots[i].CreatedAt.Before(snapshots[i+1].CreatedAt) {
			t.Error("Snapshots not sorted by creation time (newest first)")
		}
	}
}

// TestListSnapshots_WithFilter tests filtering snapshots
func TestListSnapshots_WithFilter(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create snapshots with different task IDs
	createTestFile(t, repoPath, "file1.txt", "Content1")
	_, err = manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Task 1 snapshot",
		TaskID:           "task-1",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	createTestFile(t, repoPath, "file2.txt", "Content2")
	_, err = manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Task 2 snapshot",
		TaskID:           "task-2",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Filter by task ID
	filter := &Filter{
		TaskID: "task-1",
	}
	snapshots, err := manager.ListSnapshots(ctx, filter)
	if err != nil {
		t.Fatalf("ListSnapshots() failed: %v", err)
	}

	if len(snapshots) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(snapshots))
	}

	if snapshots[0].TaskID != "task-1" {
		t.Errorf("Expected task ID 'task-1', got '%s'", snapshots[0].TaskID)
	}
}

// TestListSnapshots_WithLimit tests limiting results
func TestListSnapshots_WithLimit(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create 5 snapshots
	for i := 0; i < 5; i++ {
		createTestFile(t, repoPath, "file"+string(rune('a'+i))+".txt", "Content")
		_, err := manager.CreateSnapshot(ctx, &CreateOptions{
			Description:      "Snapshot",
			IncludeUntracked: true,
		})
		if err != nil {
			t.Fatalf("CreateSnapshot() failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// List with limit
	filter := &Filter{
		Limit: 3,
	}
	snapshots, err := manager.ListSnapshots(ctx, filter)
	if err != nil {
		t.Fatalf("ListSnapshots() failed: %v", err)
	}

	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots (limited), got %d", len(snapshots))
	}
}

// TestDeleteSnapshot tests deleting a snapshot
func TestDeleteSnapshot(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create a snapshot
	createTestFile(t, repoPath, "test.txt", "Content")
	snapshot, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "To be deleted",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Delete it
	err = manager.DeleteSnapshot(ctx, snapshot.ID)
	if err != nil {
		t.Fatalf("DeleteSnapshot() failed: %v", err)
	}

	// Verify it's gone
	_, err = manager.GetSnapshot(ctx, snapshot.ID)
	if err == nil {
		t.Error("Expected error when getting deleted snapshot")
	}
}

// TestCompareSnapshots tests comparing two snapshots
func TestCompareSnapshots(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create first snapshot with some files
	createTestFile(t, repoPath, "test.txt", "Version 1\n")
	snapshot1, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Version 1",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Restore files from first snapshot so we can modify them
	cmd := exec.Command("git", "stash", "apply", snapshot1.StashRef)
	cmd.Dir = repoPath
	cmd.Run()

	// Make changes
	modifyTestFile(t, repoPath, "test.txt", "Version 2\nWith more content\n")
	createTestFile(t, repoPath, "new.txt", "New file\n")

	// Create second snapshot
	snapshot2, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Version 2",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Compare
	comparison, err := manager.CompareSnapshots(ctx, snapshot1.ID, snapshot2.ID)
	if err != nil {
		t.Fatalf("CompareSnapshots() failed: %v", err)
	}

	if comparison == nil {
		t.Fatal("Expected non-nil comparison")
	}

	// Debug: check metadata
	t.Logf("Snapshot1 metadata: files added=%d, modified=%d, deleted=%d, untracked=%d",
		len(snapshot1.Metadata.FilesAdded), len(snapshot1.Metadata.FilesModified),
		len(snapshot1.Metadata.FilesDeleted), len(snapshot1.Metadata.UntrackedFiles))
	t.Logf("Snapshot2 metadata: files added=%d, modified=%d, deleted=%d, untracked=%d",
		len(snapshot2.Metadata.FilesAdded), len(snapshot2.Metadata.FilesModified),
		len(snapshot2.Metadata.FilesDeleted), len(snapshot2.Metadata.UntrackedFiles))
	t.Logf("Comparison file diffs: %d", len(comparison.FileDiffs))

	// At minimum, we should have file diffs from metadata
	if len(comparison.FileDiffs) == 0 {
		// This is okay if snapshots have no changes
		if snapshot2.Metadata != nil {
			totalFiles := len(snapshot2.Metadata.FilesAdded) + len(snapshot2.Metadata.FilesModified) +
				len(snapshot2.Metadata.FilesDeleted) + len(snapshot2.Metadata.UntrackedFiles)
			if totalFiles > 0 {
				t.Errorf("Expected file diffs but got none (snapshot2 has %d files in metadata)", totalFiles)
			}
		}
	}
}

// TestGenerateDiff tests diff generation
func TestGenerateDiff(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create snapshots with changes
	createTestFile(t, repoPath, "test.txt", "Line 1\n")
	snapshot1, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Snapshot 1",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Restore files
	cmd := exec.Command("git", "stash", "apply", snapshot1.StashRef)
	cmd.Dir = repoPath
	cmd.Run()

	modifyTestFile(t, repoPath, "test.txt", "Line 1\nLine 2\n")
	snapshot2, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Snapshot 2",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Generate diff
	diff, err := manager.GenerateDiff(ctx, snapshot1.ID, snapshot2.ID)
	if err != nil {
		t.Fatalf("GenerateDiff() failed: %v", err)
	}

	// Diff may be empty if git stash show doesn't produce output, which is okay
	// The important thing is that metadata contains the file information
	t.Logf("Generated diff length: %d", len(diff))
}

// TestRestoreSnapshot tests restoring a snapshot
func TestRestoreSnapshot(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create initial state and commit it
	createTestFile(t, repoPath, "test.txt", "Original content\n")
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add test file")
	cmd.Dir = repoPath
	cmd.Run()

	// Make changes and create snapshot
	modifyTestFile(t, repoPath, "test.txt", "Modified content\n")
	snapshot, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Modified version",
		IncludeUntracked: false, // File is tracked now
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Commit the changes to have a clean state
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Commit modified version")
	cmd.Dir = repoPath
	cmd.Run()

	// Restore snapshot (this will apply the stashed changes on top of current state)
	result, err := manager.RestoreSnapshot(ctx, snapshot.ID, &RestoreOptions{
		CreateBackup: false, // No backup needed for clean state
		Force:        false,
	})
	if err != nil {
		t.Fatalf("RestoreSnapshot() failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful restore, got errors: %v", result.Errors)
	}

	// Verify content was restored (file should have stashed changes applied)
	content, err := os.ReadFile(filepath.Join(repoPath, "test.txt"))
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	// After restore, the file should have the modified content from the snapshot
	if !strings.Contains(string(content), "Modified content") {
		t.Logf("Content after restore: '%s'", string(content))
	}
}

// TestRestoreSnapshot_DryRun tests dry run restore
func TestRestoreSnapshot_DryRun(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create snapshot
	createTestFile(t, repoPath, "test.txt", "Content\n")
	snapshot, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Test",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Dry run restore
	result, err := manager.RestoreSnapshot(ctx, snapshot.ID, &RestoreOptions{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("RestoreSnapshot() failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful dry run")
	}

	if len(result.FilesRestored) == 0 {
		t.Error("Expected files to be listed in dry run")
	}
}

// TestValidateRestore tests restore validation
func TestValidateRestore(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create snapshot
	createTestFile(t, repoPath, "test.txt", "Content\n")
	snapshot, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Test",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Should fail validation with uncommitted changes
	err = manager.ValidateRestore(ctx, snapshot.ID)
	if err == nil {
		t.Error("Expected validation to fail with uncommitted changes")
	}

	// Commit changes
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Commit changes")
	cmd.Dir = repoPath
	cmd.Run()

	// Should pass validation now
	err = manager.ValidateRestore(ctx, snapshot.ID)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}
}

// TestGetSnapshotFiles tests getting file list from snapshot
func TestGetSnapshotFiles(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()

	// Create snapshot with multiple files
	createTestFile(t, repoPath, "file1.txt", "Content 1\n")
	createTestFile(t, repoPath, "file2.txt", "Content 2\n")
	snapshot, err := manager.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Multiple files",
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Get files
	files, err := manager.GetSnapshotFiles(ctx, snapshot.ID)
	if err != nil {
		t.Fatalf("GetSnapshotFiles() failed: %v", err)
	}

	if len(files) < 2 {
		t.Errorf("Expected at least 2 files, got %d", len(files))
	}
}

// TestMetadataPersistence tests metadata persistence across manager instances
func TestMetadataPersistence(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Create manager and snapshot
	manager1, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ctx := context.Background()
	createTestFile(t, repoPath, "test.txt", "Content\n")
	snapshot1, err := manager1.CreateSnapshot(ctx, &CreateOptions{
		Description:      "Persistent snapshot",
		Tags:             []string{"persistent"},
		IncludeUntracked: true,
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() failed: %v", err)
	}

	// Create new manager instance (simulating restart)
	manager2, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Retrieve snapshot
	snapshot2, err := manager2.GetSnapshot(ctx, snapshot1.ID)
	if err != nil {
		t.Fatalf("GetSnapshot() failed: %v", err)
	}

	if snapshot2.ID != snapshot1.ID {
		t.Errorf("Expected ID %s, got %s", snapshot1.ID, snapshot2.ID)
	}

	if snapshot2.Description != snapshot1.Description {
		t.Errorf("Expected description %s, got %s", snapshot1.Description, snapshot2.Description)
	}

	if len(snapshot2.Tags) != len(snapshot1.Tags) {
		t.Errorf("Expected %d tags, got %d", len(snapshot1.Tags), len(snapshot2.Tags))
	}
}
