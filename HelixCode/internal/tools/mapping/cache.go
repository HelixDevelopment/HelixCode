package mapping

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// CacheVersion is the current cache format version
	CacheVersion = "v1"

	// DefaultCacheDir is the default cache directory name
	DefaultCacheDir = ".helix.cache"

	// CacheDirPermissions are the permissions for cache directory
	CacheDirPermissions = 0755

	// CacheFilePermissions are the permissions for cache files
	CacheFilePermissions = 0644
)

// CacheManager manages codebase map cache
type CacheManager interface {
	// Load loads a cached map
	Load(root string) (*CodebaseMap, error)

	// Save saves a map to cache
	Save(cmap *CodebaseMap) error

	// LoadFile loads a cached file map
	LoadFile(path string) (*FileMap, error)

	// SaveFile saves a file map to cache
	SaveFile(fileMap *FileMap) error

	// Invalidate invalidates cache for specific files
	Invalidate(files []string) error

	// Clear clears all cache
	Clear() error

	// GetCacheDir returns the cache directory
	GetCacheDir() string

	// GetCacheStats returns cache statistics
	GetCacheStats() (*CacheStats, error)
}

// CacheStats contains cache statistics
type CacheStats struct {
	TotalFiles  int       `json:"total_files"`
	TotalSize   int64     `json:"total_size"`
	HitRate     float64   `json:"hit_rate"`
	LastUpdated time.Time `json:"last_updated"`
	Version     string    `json:"version"`
}

// DiskCacheManager implements CacheManager with disk-based caching
type DiskCacheManager struct {
	cacheDir string
	mu       sync.RWMutex
	hits     int64
	misses   int64
	maxSize  int64 // Maximum cache size in bytes
}

// NewDiskCacheManager creates a new disk cache manager
func NewDiskCacheManager(workspaceRoot string) *DiskCacheManager {
	cacheDir := filepath.Join(workspaceRoot, DefaultCacheDir)
	_ = os.MkdirAll(cacheDir, CacheDirPermissions)

	return &DiskCacheManager{
		cacheDir: cacheDir,
		maxSize:  1024 * 1024 * 1024, // 1 GB default
	}
}

// NewDiskCacheManagerWithDir creates a cache manager with a custom cache directory
func NewDiskCacheManagerWithDir(cacheDir string) *DiskCacheManager {
	_ = os.MkdirAll(cacheDir, CacheDirPermissions)

	return &DiskCacheManager{
		cacheDir: cacheDir,
		maxSize:  1024 * 1024 * 1024,
	}
}

// Load loads a cached codebase map
func (cm *DiskCacheManager) Load(root string) (*CodebaseMap, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cachePath := cm.getCodebaseMapCachePath(root)

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		cm.misses++
		return nil, fmt.Errorf("cache not found")
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		cm.misses++
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	// Unmarshal
	var cmap CodebaseMap
	if err := json.Unmarshal(data, &cmap); err != nil {
		cm.misses++
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Verify version
	if cmap.Version != CacheVersion {
		cm.misses++
		return nil, fmt.Errorf("cache version mismatch: expected %s, got %s", CacheVersion, cmap.Version)
	}

	cm.hits++
	return &cmap, nil
}

// Save saves a codebase map to cache
func (cm *DiskCacheManager) Save(cmap *CodebaseMap) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Set version
	cmap.Version = CacheVersion
	cmap.UpdatedAt = time.Now()

	// Marshal to JSON
	data, err := json.MarshalIndent(cmap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Check cache size limit
	if err := cm.evictIfNeeded(int64(len(data))); err != nil {
		return fmt.Errorf("failed to evict cache: %w", err)
	}

	cachePath := cm.getCodebaseMapCachePath(cmap.Root)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), CacheDirPermissions); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write to temp file first (atomic write)
	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, CacheFilePermissions); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	// Rename to actual path (atomic)
	if err := os.Rename(tmpPath, cachePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename cache: %w", err)
	}

	return nil
}

// LoadFile loads a cached file map
func (cm *DiskCacheManager) LoadFile(path string) (*FileMap, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cachePath := cm.getFileMapCachePath(path)

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		cm.misses++
		return nil, fmt.Errorf("cache not found")
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		cm.misses++
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	// Unmarshal
	var fileMap FileMap
	if err := json.Unmarshal(data, &fileMap); err != nil {
		cm.misses++
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Verify file hasn't changed
	if !cm.isFileMapValid(&fileMap, path) {
		cm.misses++
		return nil, fmt.Errorf("cached file map is stale")
	}

	cm.hits++
	return &fileMap, nil
}

// SaveFile saves a file map to cache
func (cm *DiskCacheManager) SaveFile(fileMap *FileMap) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Marshal to JSON
	data, err := json.MarshalIndent(fileMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cachePath := cm.getFileMapCachePath(fileMap.Path)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), CacheDirPermissions); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write to temp file first
	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, CacheFilePermissions); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	// Rename to actual path
	if err := os.Rename(tmpPath, cachePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename cache: %w", err)
	}

	return nil
}

// Invalidate invalidates cache for specific files
func (cm *DiskCacheManager) Invalidate(files []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, file := range files {
		cachePath := cm.getFileMapCachePath(file)
		if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to invalidate cache for %s: %w", file, err)
		}
	}

	return nil
}

// Clear clears all cache
func (cm *DiskCacheManager) Clear() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := os.RemoveAll(cm.cacheDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Recreate cache directory
	if err := os.MkdirAll(cm.cacheDir, CacheDirPermissions); err != nil {
		return fmt.Errorf("failed to recreate cache directory: %w", err)
	}

	cm.hits = 0
	cm.misses = 0

	return nil
}

// GetCacheDir returns the cache directory
func (cm *DiskCacheManager) GetCacheDir() string {
	return cm.cacheDir
}

// GetCacheStats returns cache statistics
func (cm *DiskCacheManager) GetCacheStats() (*CacheStats, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var totalFiles int
	var totalSize int64
	var lastUpdated time.Time

	err := filepath.Walk(cm.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			totalFiles++
			totalSize += info.Size()
			if info.ModTime().After(lastUpdated) {
				lastUpdated = info.ModTime()
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate cache stats: %w", err)
	}

	// Calculate hit rate
	total := cm.hits + cm.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(cm.hits) / float64(total)
	}

	return &CacheStats{
		TotalFiles:  totalFiles,
		TotalSize:   totalSize,
		HitRate:     hitRate,
		LastUpdated: lastUpdated,
		Version:     CacheVersion,
	}, nil
}

// getCodebaseMapCachePath returns the cache file path for a codebase map
func (cm *DiskCacheManager) getCodebaseMapCachePath(root string) string {
	hash := sha256.Sum256([]byte(root))
	filename := fmt.Sprintf("codebase_%x.json", hash[:16])
	return filepath.Join(cm.cacheDir, "maps", filename)
}

// getFileMapCachePath returns the cache file path for a file map
func (cm *DiskCacheManager) getFileMapCachePath(path string) string {
	hash := sha256.Sum256([]byte(path))
	filename := fmt.Sprintf("file_%x.json", hash[:16])
	return filepath.Join(cm.cacheDir, "files", filename)
}

// isFileMapValid checks if a cached file map is still valid
func (cm *DiskCacheManager) isFileMapValid(fileMap *FileMap, path string) bool {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check file size
	if info.Size() != fileMap.Size {
		return false
	}

	// Check modification time (allow 1 second tolerance)
	if info.ModTime().After(fileMap.ParsedAt.Add(time.Second)) {
		return false
	}

	return true
}

// evictIfNeeded evicts old cache entries if needed (LRU)
func (cm *DiskCacheManager) evictIfNeeded(newSize int64) error {
	// Calculate current cache size
	currentSize := int64(0)
	type cacheEntry struct {
		path    string
		modTime time.Time
		size    int64
	}
	var entries []cacheEntry

	err := filepath.Walk(cm.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			currentSize += info.Size()
			entries = append(entries, cacheEntry{
				path:    path,
				modTime: info.ModTime(),
				size:    info.Size(),
			})
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Check if we need to evict
	if currentSize+newSize <= cm.maxSize {
		return nil
	}

	// Sort by modification time (oldest first)
	// Simple bubble sort for small datasets
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].modTime.After(entries[j].modTime) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Evict oldest entries until we have enough space
	targetSize := cm.maxSize - newSize
	for _, entry := range entries {
		if currentSize <= targetSize {
			break
		}

		if err := os.Remove(entry.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to evict cache entry: %w", err)
		}

		currentSize -= entry.size
	}

	return nil
}

// SetMaxSize sets the maximum cache size in bytes
func (cm *DiskCacheManager) SetMaxSize(size int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.maxSize = size
}
