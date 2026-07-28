package verifier

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Cache provides a two-tier cache (in-memory LRU + Redis fallback) for verifier data.
// If Redis is unavailable, falls back to memory-only.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	redis   RedisClient // optional
	maxSize int

	// now is the cache's clock. Production always uses time.Now (set by
	// NewCache); it is a field rather than a direct time.Now() call so that
	// TTL/stale-window boundary tests can advance time EXPLICITLY instead of
	// sleeping past an approximate wall-clock window. time.Sleep guarantees a
	// lower bound only, never an upper one, so a sleep-driven boundary test
	// silently overshoots under host load and its verdict becomes a function of
	// machine load rather than of the code under test (§11.4.50). With an
	// injected clock the boundary is exact to the nanosecond — strictly
	// stricter than any sleep could be. Never mutated after construction
	// outside tests.
	now func() time.Time
}

// RedisClient is the minimal interface needed from the Redis wrapper.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

// cacheEntry stores cached data with metadata.
type cacheEntry struct {
	Models    []*VerifiedModel
	Scores    map[string]float64
	FetchedAt time.Time
	Source    string // "verifier", "fallback"
}

// NewCache creates a new verifier cache.
// If redisClient is nil, operates in memory-only mode.
func NewCache(ttl time.Duration, redisClient RedisClient) *Cache {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &Cache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
		redis:   redisClient,
		maxSize: 1024,
		now:     time.Now,
	}
}

// clock returns the cache's current time. It tolerates a zero-value Cache
// (now == nil) by falling back to time.Now, so behaviour is identical whether
// the Cache came from NewCache or from a composite literal.
func (c *Cache) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// GetModels returns cached models for a provider (or "all") if fresh.
func (c *Cache) GetModels(provider string) ([]*VerifiedModel, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[provider]
	if !ok || entry == nil {
		return nil, false
	}
	if c.clock().Sub(entry.FetchedAt) > c.ttl {
		return nil, false
	}
	return entry.Models, true
}

// GetModelsStale returns cached models even if slightly stale (up to 2x TTL).
func (c *Cache) GetModelsStale(provider string) ([]*VerifiedModel, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[provider]
	if !ok || entry == nil {
		return nil, false
	}
	if c.clock().Sub(entry.FetchedAt) > 2*c.ttl {
		return nil, false
	}
	return entry.Models, true
}

// SetModels stores models in the cache.
func (c *Cache) SetModels(provider string, models []*VerifiedModel) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[provider] = &cacheEntry{
		Models:    models,
		FetchedAt: c.clock(),
		Source:    "verifier",
	}

	// Also store in Redis if available
	if c.redis != nil {
		data, _ := json.Marshal(entryToSerializable(c.entries[provider]))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.redis.Set(ctx, cacheKeyModels(provider), string(data), c.ttl)
	}

	c.evictIfNeeded()
}

// GetModelScore returns a cached model score if fresh.
func (c *Cache) GetModelScore(modelID string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, entry := range c.entries {
		if entry.Scores != nil {
			if score, ok := entry.Scores[modelID]; ok {
				if c.clock().Sub(entry.FetchedAt) <= c.ttl {
					return score, true
				}
			}
		}
	}
	return 0, false
}

// SetScores stores provider scores in the cache.
func (c *Cache) SetScores(scores map[string]float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &cacheEntry{
		Scores:    scores,
		FetchedAt: c.clock(),
		Source:    "verifier",
	}
	c.entries["__scores__"] = entry

	if c.redis != nil {
		data, _ := json.Marshal(scores)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.redis.Set(ctx, cacheKeyScores(), string(data), c.ttl)
	}
}

// Invalidate removes a provider's cached data.
func (c *Cache) Invalidate(provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, provider)
}

// InvalidateAll clears all cached data.
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

func (c *Cache) evictIfNeeded() {
	if len(c.entries) <= c.maxSize {
		return
	}
	// Simple eviction: remove oldest entry
	var oldestKey string
	var oldestTime time.Time
	for k, v := range c.entries {
		if oldestTime.IsZero() || v.FetchedAt.Before(oldestTime) {
			oldestTime = v.FetchedAt
			oldestKey = k
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func cacheKeyModels(provider string) string {
	return "helix:verifier:models:" + provider
}

func cacheKeyScores() string {
	return "helix:verifier:scores"
}

type serializableEntry struct {
	Models    []*VerifiedModel `json:"models"`
	Scores    map[string]float64 `json:"scores"`
	FetchedAt time.Time        `json:"fetched_at"`
	Source    string           `json:"source"`
}

func entryToSerializable(e *cacheEntry) *serializableEntry {
	if e == nil {
		return nil
	}
	return &serializableEntry{
		Models:    e.Models,
		Scores:    e.Scores,
		FetchedAt: e.FetchedAt,
		Source:    e.Source,
	}
}
