// Package snapshots provides git-based workspace snapshot functionality for HelixCode.
//
// This package implements a checkpoint system that allows users to save, compare,
// and restore workspace states at different points in time. It uses git stash
// internally for efficient storage and leverages Git's diff capabilities for
// comparisons.
//
// # Overview
//
// The snapshots package provides:
//   - Workspace snapshot creation using git stash
//   - Comparison between any two snapshots with detailed diffs
//   - Restore to specific snapshots with safety checks
//   - Snapshot metadata tracking (timestamp, task, description, etc.)
//   - List and filter snapshots
//   - Snapshot verification and integrity checking
//
// # Basic Usage
//
// Creating a snapshot:
//
//	manager, err := snapshots.NewManager("/path/to/repo")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	snapshot, err := manager.CreateSnapshot(ctx, &snapshots.CreateOptions{
//	    Description: "Before refactoring",
//	    TaskID: "task-123",
//	    Tags: []string{"stable"},
//	    IncludeUntracked: true,
//	})
//
// Listing snapshots:
//
//	snapshots, err := manager.ListSnapshots(ctx, &snapshots.Filter{
//	    TaskID: "task-123",
//	    Limit: 10,
//	})
//
// Comparing snapshots:
//
//	comparison, err := manager.CompareSnapshots(ctx, snapshot1.ID, snapshot2.ID)
//	fmt.Printf("Files modified: %d\n", comparison.Summary.FilesModified)
//	fmt.Printf("Lines added: %d\n", comparison.Summary.LinesAdded)
//
// Restoring a snapshot:
//
//	result, err := manager.RestoreSnapshot(ctx, snapshot.ID, &snapshots.RestoreOptions{
//	    CreateBackup: true,
//	    DryRun: false,
//	})
//	if !result.Success {
//	    log.Printf("Restore failed: %v", result.Errors)
//	}
//
// # Architecture
//
// The package consists of several components:
//
//   - Manager: Main orchestrator for snapshot operations
//   - MetadataStore: Persists snapshot metadata to .helix.snapshots.json
//   - Snapshot: Represents a workspace state with metadata
//   - Comparison: Represents differences between snapshots
//
// # Git Integration
//
// Snapshots are stored using git stash with tagged names:
//   - Stash message format: "helix-snapshot-{uuid}: {description}"
//   - Metadata stored separately in .helix.snapshots.json (gitignored)
//   - Uses git diff for comparisons
//   - Uses git stash apply for restore operations
//
// # Safety Features
//
//   - Verify clean working tree before restore (unless --force)
//   - Conflict detection and reporting
//   - Automatic backup creation before restore
//   - Snapshot verification before operations
//   - Atomic metadata updates
//
// # Metadata
//
// Each snapshot includes rich metadata:
//   - ID: Unique identifier
//   - Created timestamp
//   - Description and tags
//   - Task ID (if applicable)
//   - File statistics (added, modified, deleted)
//   - Git context (branch, commit hash)
//   - Custom metadata fields
//
// # Error Handling
//
// All operations return errors that can be checked:
//
//	snapshot, err := manager.CreateSnapshot(ctx, opts)
//	if err != nil {
//	    if strings.Contains(err.Error(), "no changes") {
//	        // Handle no changes case
//	    } else if strings.Contains(err.Error(), "not a git repository") {
//	        // Handle non-git repo case
//	    } else {
//	        // Handle other errors
//	    }
//	}
//
// # Thread Safety
//
// The MetadataStore uses mutex locks to ensure thread-safe access to metadata.
// Multiple goroutines can safely create, list, and restore snapshots concurrently.
//
// # Performance
//
// The package leverages Git's efficient storage:
//   - Snapshots share objects with the repository (deduplication)
//   - Only changed files are stored
//   - Git's compression is used automatically
//   - Metadata file is kept separate for fast querying
//
// # Limitations
//
//   - Requires a git repository
//   - Snapshot IDs must be unique across the repository
//   - Large binary files may slow down operations
//   - Restore conflicts require manual resolution
package snapshots
