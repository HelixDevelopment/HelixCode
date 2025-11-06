package snapshots

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// MetadataFileName is the name of the metadata file
	MetadataFileName = ".helix.snapshots.json"
)

// MetadataStore handles snapshot metadata persistence
type MetadataStore struct {
	repoPath string
	mu       sync.RWMutex
	cache    map[string]*Snapshot
}

// metadataFile represents the structure of the metadata file
type metadataFile struct {
	Version   string               `json:"version"`
	Snapshots map[string]*Snapshot `json:"snapshots"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// NewMetadataStore creates a new metadata store
func NewMetadataStore(repoPath string) (*MetadataStore, error) {
	store := &MetadataStore{
		repoPath: repoPath,
		cache:    make(map[string]*Snapshot),
	}

	// Load existing metadata
	if err := store.load(); err != nil {
		// If file doesn't exist, that's okay
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load metadata: %w", err)
		}
	}

	return store, nil
}

// Save persists snapshot metadata
func (m *MetadataStore) Save(ctx context.Context, snapshot *Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache[snapshot.ID] = snapshot
	return m.persist()
}

// Load retrieves snapshot metadata
func (m *MetadataStore) Load(ctx context.Context, id string) (*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot, ok := m.cache[id]
	if !ok {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}

	return snapshot, nil
}

// List returns all snapshots, optionally filtered
func (m *MetadataStore) List(ctx context.Context, filter *Filter) ([]*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]*Snapshot, 0, len(m.cache))
	for _, snapshot := range m.cache {
		if m.matchesFilter(snapshot, filter) {
			snapshots = append(snapshots, snapshot)
		}
	}

	// Sort by creation time (most recent first)
	for i := 0; i < len(snapshots); i++ {
		for j := i + 1; j < len(snapshots); j++ {
			if snapshots[i].CreatedAt.Before(snapshots[j].CreatedAt) {
				snapshots[i], snapshots[j] = snapshots[j], snapshots[i]
			}
		}
	}

	// Apply limit if specified
	if filter != nil && filter.Limit > 0 && len(snapshots) > filter.Limit {
		snapshots = snapshots[:filter.Limit]
	}

	return snapshots, nil
}

// Delete removes snapshot metadata
func (m *MetadataStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.cache, id)
	return m.persist()
}

// Update modifies existing snapshot metadata
func (m *MetadataStore) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, ok := m.cache[id]
	if !ok {
		return fmt.Errorf("snapshot not found: %s", id)
	}

	// Apply updates
	if desc, ok := updates["description"].(string); ok {
		snapshot.Description = desc
	}
	if tags, ok := updates["tags"].([]string); ok {
		snapshot.Tags = tags
	}
	if status, ok := updates["status"].(SnapshotStatus); ok {
		snapshot.Status = status
	}

	return m.persist()
}

// matchesFilter checks if a snapshot matches the given filter
func (m *MetadataStore) matchesFilter(snapshot *Snapshot, filter *Filter) bool {
	if filter == nil {
		return true
	}

	// Filter by task ID
	if filter.TaskID != "" && snapshot.TaskID != filter.TaskID {
		return false
	}

	// Filter by status
	if filter.Status != "" && snapshot.Status != filter.Status {
		return false
	}

	// Filter by tags
	if len(filter.Tags) > 0 {
		hasTag := false
		for _, filterTag := range filter.Tags {
			for _, snapshotTag := range snapshot.Tags {
				if filterTag == snapshotTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	// Filter by date range
	if !filter.FromDate.IsZero() && snapshot.CreatedAt.Before(filter.FromDate) {
		return false
	}
	if !filter.ToDate.IsZero() && snapshot.CreatedAt.After(filter.ToDate) {
		return false
	}

	return true
}

// load reads metadata from disk
func (m *MetadataStore) load() error {
	path := m.metadataPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var file metadataFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	m.cache = file.Snapshots
	if m.cache == nil {
		m.cache = make(map[string]*Snapshot)
	}

	return nil
}

// persist writes metadata to disk
func (m *MetadataStore) persist() error {
	file := metadataFile{
		Version:   "1.0",
		Snapshots: m.cache,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	path := m.metadataPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Write atomically
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename metadata file: %w", err)
	}

	return nil
}

// metadataPath returns the path to the metadata file
func (m *MetadataStore) metadataPath() string {
	return filepath.Join(m.repoPath, MetadataFileName)
}
