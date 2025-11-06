package snapshots

import (
	"time"
)

// Snapshot represents a workspace snapshot
type Snapshot struct {
	ID          string           `json:"id"`
	StashRef    string           `json:"stash_ref"`    // Git stash reference
	StashIndex  int              `json:"stash_index"`  // Index in stash list
	CreatedAt   time.Time        `json:"created_at"`
	Description string           `json:"description"`
	TaskID      string           `json:"task_id,omitempty"`
	Status      SnapshotStatus   `json:"status"`
	Metadata    *Metadata        `json:"metadata"`
	Tags        []string         `json:"tags,omitempty"`
	FileCount   int              `json:"file_count"`
	Size        int64            `json:"size"` // Approximate size in bytes
}

// SnapshotStatus represents snapshot state
type SnapshotStatus string

const (
	StatusActive    SnapshotStatus = "active"
	StatusArchived  SnapshotStatus = "archived"
	StatusCorrupted SnapshotStatus = "corrupted"
)

// Metadata contains detailed snapshot information
type Metadata struct {
	// Context information
	WorkingDirectory string `json:"working_directory"`
	Branch           string `json:"branch"`
	CommitHash       string `json:"commit_hash"`

	// File statistics
	FilesAdded     []string `json:"files_added,omitempty"`
	FilesModified  []string `json:"files_modified,omitempty"`
	FilesDeleted   []string `json:"files_deleted,omitempty"`
	UntrackedFiles []string `json:"untracked_files,omitempty"`

	// Task context
	TaskDescription string `json:"task_description,omitempty"`
	TaskStep        int    `json:"task_step,omitempty"`

	// System information
	HelixVersion string            `json:"helix_version,omitempty"`
	Custom       map[string]string `json:"custom,omitempty"`
}

// CreateOptions specifies snapshot creation options
type CreateOptions struct {
	Description      string            `json:"description"`
	TaskID           string            `json:"task_id,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	IncludeUntracked bool              `json:"include_untracked"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	AutoGenerate     bool              `json:"auto_generate"` // Auto-generate description
}

// Comparison represents a comparison between two snapshots
type Comparison struct {
	From       *Snapshot    `json:"from"`
	To         *Snapshot    `json:"to"`
	Summary    *Summary     `json:"summary"`
	FileDiffs  []*FileDiff  `json:"file_diffs"`
	Statistics *Statistics  `json:"statistics"`
}

// Summary provides high-level comparison information
type Summary struct {
	FilesAdded    int           `json:"files_added"`
	FilesModified int           `json:"files_modified"`
	FilesDeleted  int           `json:"files_deleted"`
	LinesAdded    int           `json:"lines_added"`
	LinesDeleted  int           `json:"lines_deleted"`
	TimeElapsed   time.Duration `json:"time_elapsed"`
}

// FileDiff represents changes to a single file
type FileDiff struct {
	Path         string     `json:"path"`
	Status       DiffStatus `json:"status"`
	Diff         string     `json:"diff"` // Unified diff format
	LinesAdded   int        `json:"lines_added"`
	LinesDeleted int        `json:"lines_deleted"`
}

// DiffStatus represents file change status
type DiffStatus string

const (
	DiffAdded    DiffStatus = "added"
	DiffModified DiffStatus = "modified"
	DiffDeleted  DiffStatus = "deleted"
	DiffRenamed  DiffStatus = "renamed"
)

// Statistics contains detailed diff statistics
type Statistics struct {
	TotalFiles   int `json:"total_files"`
	TotalLines   int `json:"total_lines"`
	LinesAdded   int `json:"lines_added"`
	LinesDeleted int `json:"lines_deleted"`
	BinaryFiles  int `json:"binary_files"`
}

// RestoreOptions specifies restoration behavior
type RestoreOptions struct {
	CreateBackup bool   `json:"create_backup"` // Create backup before restore
	DryRun       bool   `json:"dry_run"`       // Preview without applying
	Force        bool   `json:"force"`         // Force restore (skip checks)
	KeepIndex    bool   `json:"keep_index"`    // Keep current index (staged changes)
}

// RestoreResult contains restoration results
type RestoreResult struct {
	Success          bool       `json:"success"`
	BackupSnapshot   *Snapshot  `json:"backup_snapshot,omitempty"`
	FilesRestored    []string   `json:"files_restored"`
	ConflictFiles    []string   `json:"conflict_files,omitempty"`
	Errors           []string   `json:"errors,omitempty"`
	Duration         time.Duration `json:"duration"`
}

// Filter for querying snapshots
type Filter struct {
	TaskID    string         `json:"task_id,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	Status    SnapshotStatus `json:"status,omitempty"`
	FromDate  time.Time      `json:"from_date,omitempty"`
	ToDate    time.Time      `json:"to_date,omitempty"`
	Limit     int            `json:"limit,omitempty"`
}
