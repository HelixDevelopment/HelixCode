package repomap

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"crypto/sha256"
	"encoding/hex"
)

func init() {
	// Register types for gob encoding
	gob.Register([]Symbol{})
	gob.Register(Symbol{})
}

// RepoCache provides disk-based caching for parsed symbols
type RepoCache struct {
	cacheDir string
	ttl      time.Duration
	mu       sync.RWMutex
	entries  map[string]*cacheEntry
}

// cacheEntry represents a cached item with metadata
type cacheEntry struct {
	Key       string
	Value     interface{}
	CachedAt  time.Time
	ExpiresAt time.Time
}

// NewRepoCache creates a new cache instance
func NewRepoCache(cacheDir string, ttl time.Duration) (*RepoCache, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &RepoCache{
		cacheDir: cacheDir,
		ttl:      ttl,
		entries:  make(map[string]*cacheEntry),
	}

	// Load existing cache entries
	if err := cache.loadFromDisk(); err != nil {
		// Non-fatal error, just log and continue
		fmt.Printf("Warning: failed to load cache: %v\n", err)
	}

	return cache, nil
}

// Get retrieves a value from the cache
func (rc *RepoCache) Get(key string) (interface{}, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.entries[key]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in the cache
func (rc *RepoCache) Set(key string, value interface{}) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	entry := &cacheEntry{
		Key:       key,
		Value:     value,
		CachedAt:  now,
		ExpiresAt: now.Add(rc.ttl),
	}

	rc.entries[key] = entry

	// Persist to disk asynchronously
	go func() {
		if err := rc.saveToDisk(key, entry); err != nil {
			fmt.Printf("Warning: failed to save cache entry: %v\n", err)
		}
	}()
}

// Invalidate removes a specific entry from the cache
func (rc *RepoCache) Invalidate(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	delete(rc.entries, key)

	// Remove from disk
	go func() {
		filename := rc.getCacheFilename(key)
		os.Remove(filename)
	}()
}

// InvalidateAll clears the entire cache
func (rc *RepoCache) InvalidateAll() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.entries = make(map[string]*cacheEntry)

	// Remove all cache files
	return os.RemoveAll(rc.cacheDir)
}

// Size returns the number of cached entries
func (rc *RepoCache) Size() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return len(rc.entries)
}

// Cleanup removes expired entries
func (rc *RepoCache) Cleanup() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range rc.entries {
		if now.After(entry.ExpiresAt) {
			delete(rc.entries, key)
			removed++

			// Remove from disk
			go func(k string) {
				filename := rc.getCacheFilename(k)
				os.Remove(filename)
			}(key)
		}
	}

	return removed
}

// GetStats returns cache statistics
func (rc *RepoCache) GetStats() CacheStats {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	stats := CacheStats{
		TotalEntries:   len(rc.entries),
		ExpiredEntries: 0,
		TotalSize:      0,
	}

	now := time.Now()
	for _, entry := range rc.entries {
		if now.After(entry.ExpiresAt) {
			stats.ExpiredEntries++
		}

		// Estimate size
		stats.TotalSize += estimateSize(entry.Value)
	}

	return stats
}

// CacheStats provides statistics about the cache
type CacheStats struct {
	TotalEntries   int
	ExpiredEntries int
	TotalSize      int64 // Bytes
}

// saveToDisk persists a cache entry to disk
func (rc *RepoCache) saveToDisk(key string, entry *cacheEntry) error {
	filename := rc.getCacheFilename(key)

	// Create directory if needed
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Encode entry
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode cache entry: %w", err)
	}

	// Write to disk atomically
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	if err := os.Rename(tempFile, filename); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

// loadFromDisk loads all cache entries from disk
func (rc *RepoCache) loadFromDisk() error {
	// Walk through cache directory
	return filepath.Walk(rc.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		if info.IsDir() {
			return nil
		}

		// Skip temp files
		if filepath.Ext(path) == ".tmp" {
			return nil
		}

		// Load entry
		entry, err := rc.loadEntryFromFile(path)
		if err != nil {
			// Remove corrupted cache file
			os.Remove(path)
			return nil
		}

		// Check if expired
		if time.Now().After(entry.ExpiresAt) {
			os.Remove(path)
			return nil
		}

		rc.entries[entry.Key] = entry
		return nil
	})
}

// loadEntryFromFile loads a single cache entry from a file
func (rc *RepoCache) loadEntryFromFile(filename string) (*cacheEntry, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var entry cacheEntry
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode cache entry: %w", err)
	}

	return &entry, nil
}

// getCacheFilename generates a filename for a cache key
func (rc *RepoCache) getCacheFilename(key string) string {
	// Hash the key to create a safe filename
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	// Use first 2 chars for subdirectory (better distribution)
	subdir := hashStr[:2]

	// Use shorter filename - just first 16 chars of hash
	filename := hashStr[:16] + ".cache"

	return filepath.Join(rc.cacheDir, subdir, filename)
}

// estimateSize estimates the memory size of a value
func estimateSize(value interface{}) int64 {
	// Encode to get approximate size
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return 0
	}
	return int64(buf.Len())
}

// StartCleanupRoutine starts a background goroutine to periodically clean up expired entries
func (rc *RepoCache) StartCleanupRoutine(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				removed := rc.Cleanup()
				if removed > 0 {
					fmt.Printf("Cache cleanup: removed %d expired entries\n", removed)
				}
			case <-stop:
				return
			}
		}
	}()

	return stop
}

// GetOrCompute retrieves a value from cache or computes it if not found
func (rc *RepoCache) GetOrCompute(key string, compute func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, found := rc.Get(key); found {
		return value, nil
	}

	// Not in cache, compute it
	value, err := compute()
	if err != nil {
		return nil, err
	}

	// Store in cache
	rc.Set(key, value)

	return value, nil
}

// Has checks if a key exists in the cache (without returning the value)
func (rc *RepoCache) Has(key string) bool {
	_, found := rc.Get(key)
	return found
}

// Keys returns all cache keys
func (rc *RepoCache) Keys() []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	keys := make([]string, 0, len(rc.entries))
	for key := range rc.entries {
		keys = append(keys, key)
	}

	return keys
}

// Export exports cache entries to a writer (for backup)
func (rc *RepoCache) Export(writer *os.File) error {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	encoder := gob.NewEncoder(writer)
	return encoder.Encode(rc.entries)
}

// Import imports cache entries from a reader (for restore)
func (rc *RepoCache) Import(reader *os.File) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	decoder := gob.NewDecoder(reader)
	entries := make(map[string]*cacheEntry)

	if err := decoder.Decode(&entries); err != nil {
		return fmt.Errorf("failed to decode cache: %w", err)
	}

	// Filter out expired entries
	now := time.Now()
	for key, entry := range entries {
		if now.Before(entry.ExpiresAt) {
			rc.entries[key] = entry
		}
	}

	return nil
}

// SetTTL updates the TTL for the cache
func (rc *RepoCache) SetTTL(ttl time.Duration) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.ttl = ttl
}

// GetTTL returns the current TTL
func (rc *RepoCache) GetTTL() time.Duration {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.ttl
}
