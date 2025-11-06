package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// CacheManager manages caching of web content
type CacheManager struct {
	memCache  *lru.Cache[string, *CacheEntry]
	diskCache *DiskCache
	ttl       time.Duration
	maxSize   int64
	stats     *CacheStats
	mu        sync.RWMutex
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key       string
	Value     []byte
	Timestamp time.Time
	ExpiresAt time.Time
	Size      int64
}

// CacheStats tracks cache performance
type CacheStats struct {
	Hits        atomic.Int64
	Misses      atomic.Int64
	Evictions   atomic.Int64
	BytesCached atomic.Int64
}

// DiskCache manages disk-based caching
type DiskCache struct {
	dir string
	mu  sync.RWMutex
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cacheDir string, ttl time.Duration, maxSize int64) *CacheManager {
	// Create LRU cache with 1000 entries
	memCache, err := lru.New[string, *CacheEntry](1000)
	if err != nil {
		// Fallback to no memory cache
		memCache = nil
	}

	// Set default cache directory
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "helixcode-web-cache")
	}

	cm := &CacheManager{
		memCache:  memCache,
		diskCache: NewDiskCache(cacheDir),
		ttl:       ttl,
		maxSize:   maxSize,
		stats:     &CacheStats{},
	}

	return cm
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(key string) ([]byte, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Check memory cache first
	if cm.memCache != nil {
		if entry, ok := cm.memCache.Get(key); ok {
			// Check if expired
			if time.Now().Before(entry.ExpiresAt) {
				cm.stats.Hits.Add(1)
				return entry.Value, true
			}
			// Expired - remove
			cm.memCache.Remove(key)
		}
	}

	// Check disk cache
	if entry, ok := cm.diskCache.Get(key); ok {
		// Check if expired
		if time.Now().Before(entry.ExpiresAt) {
			// Restore to memory cache
			if cm.memCache != nil {
				cm.memCache.Add(key, entry)
			}
			cm.stats.Hits.Add(1)
			return entry.Value, true
		}
		// Expired - remove
		cm.diskCache.Remove(key)
	}

	cm.stats.Misses.Add(1)
	return nil, false
}

// Set stores a value in cache
func (cm *CacheManager) Set(key string, value []byte) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entry := &CacheEntry{
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(cm.ttl),
		Size:      int64(len(value)),
	}

	// Store in memory cache
	if cm.memCache != nil {
		cm.memCache.Add(key, entry)
	}

	// Store in disk cache
	if cm.diskCache != nil {
		cm.diskCache.Set(key, entry)
	}

	cm.stats.BytesCached.Add(int64(len(value)))
}

// Remove removes a value from cache
func (cm *CacheManager) Remove(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.memCache != nil {
		cm.memCache.Remove(key)
	}

	if cm.diskCache != nil {
		cm.diskCache.Remove(key)
	}
}

// Clear clears all cache
func (cm *CacheManager) Clear() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.memCache != nil {
		cm.memCache.Purge()
	}

	if cm.diskCache != nil {
		return cm.diskCache.Clear()
	}

	return nil
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() *CacheStats {
	return cm.stats
}

// Cleanup removes expired entries
func (cm *CacheManager) Cleanup() error {
	if cm.diskCache != nil {
		return cm.diskCache.Cleanup(cm.ttl)
	}
	return nil
}

// Close closes the cache manager
func (cm *CacheManager) Close() error {
	return cm.Cleanup()
}

// NewDiskCache creates a new disk cache
func NewDiskCache(dir string) *DiskCache {
	return &DiskCache{
		dir: dir,
	}
}

// Get retrieves from disk cache
func (dc *DiskCache) Get(key string) (*CacheEntry, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	path := dc.keyPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	return &entry, true
}

// Set stores in disk cache
func (dc *DiskCache) Set(key string, entry *CacheEntry) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	path := dc.keyPath(key)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	return nil
}

// Remove removes from disk cache
func (dc *DiskCache) Remove(key string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	path := dc.keyPath(key)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Clear clears all disk cache
func (dc *DiskCache) Clear() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if err := os.RemoveAll(dc.dir); err != nil {
		return fmt.Errorf("remove cache dir: %w", err)
	}

	return nil
}

// Cleanup removes expired entries
func (dc *DiskCache) Cleanup(ttl time.Duration) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	return filepath.Walk(dc.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		// Check if file is expired
		if time.Since(info.ModTime()) > ttl {
			if err := os.Remove(path); err != nil {
				return err
			}
		}

		return nil
	})
}

// keyPath converts key to file path
func (dc *DiskCache) keyPath(key string) string {
	hash := sha256.Sum256([]byte(key))
	hashStr := fmt.Sprintf("%x", hash)
	// Create subdirectories to avoid too many files in one directory
	return filepath.Join(dc.dir, hashStr[:2], hashStr[2:4], hashStr)
}
