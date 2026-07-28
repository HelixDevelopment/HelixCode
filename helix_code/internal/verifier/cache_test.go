package verifier

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_GetModels_Miss(t *testing.T) {
	c := NewCache(5*time.Minute, nil)
	_, ok := c.GetModels("all")
	assert.False(t, ok)
}

func TestCache_GetModels_Hit(t *testing.T) {
	c := NewCache(5*time.Minute, nil)
	models := []*VerifiedModel{{ID: "gpt-4o", Provider: "openai"}}
	c.SetModels("all", models)

	got, ok := c.GetModels("all")
	require.True(t, ok)
	require.Len(t, got, 1)
	assert.Equal(t, "gpt-4o", got[0].ID)
}

// testClock is a deterministic, race-safe clock for cache expiry tests.
//
// It replaces wall-clock sleeps in the TTL / stale-window boundary tests.
// time.Sleep(d) guarantees only that AT LEAST d elapses — never at most — so a
// sleep-driven boundary assertion silently overshoots under host load and its
// verdict becomes a function of machine load rather than of the code under
// test (§11.4.50: a test that passes or fails depending on host load is itself
// a defect). Advancing an injected clock makes each boundary EXACT to the
// nanosecond, which is strictly stricter than any sleep-and-hope window.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock() *testClock {
	return &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

// Now is the func(*Cache).now hook. Mutex-guarded so it stays race-detector
// clean if a Cache under test is read concurrently.
func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by exactly d.
func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestCache_GetModels_Expired(t *testing.T) {
	clk := newTestClock()
	c := NewCache(50*time.Millisecond, nil)
	c.now = clk.Now

	models := []*VerifiedModel{{ID: "gpt-4o", Provider: "openai"}}
	c.SetModels("all", models)

	_, ok := c.GetModels("all")
	require.True(t, ok)

	// Exactly at the TTL the entry is still fresh: cache.go:80 misses only when
	// age is STRICTLY greater than ttl. Pinning this exact-boundary semantic is
	// impossible with a sleep.
	clk.Advance(50 * time.Millisecond)
	_, ok = c.GetModels("all")
	assert.True(t, ok, "at exactly TTL the entry is still fresh (age > ttl is the miss condition)")

	// One nanosecond past the TTL it must miss.
	clk.Advance(time.Nanosecond)
	_, ok = c.GetModels("all")
	assert.False(t, ok)
}

func TestCache_GetModelsStale_SlightlyExpired(t *testing.T) {
	clk := newTestClock()
	c := NewCache(50*time.Millisecond, nil)
	c.now = clk.Now

	models := []*VerifiedModel{{ID: "gpt-4o", Provider: "openai"}}
	c.SetModels("all", models)

	// Past the 50ms TTL but well inside the 2x-TTL (100ms) stale window.
	clk.Advance(60 * time.Millisecond)

	// Normal Get should miss
	_, ok := c.GetModels("all")
	assert.False(t, ok)

	// Stale get should hit (up to 2x TTL)
	got, ok := c.GetModelsStale("all")
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", got[0].ID)

	// At EXACTLY 2x TTL the entry is still stale-readable: cache.go:95 misses
	// only when age is strictly greater than 2*ttl.
	clk.Advance(40 * time.Millisecond)
	_, ok = c.GetModelsStale("all")
	assert.True(t, ok, "at exactly 2x TTL the entry is still stale-readable")

	// One nanosecond past 2x TTL, stale must also miss.
	clk.Advance(time.Nanosecond)
	_, ok = c.GetModelsStale("all")
	assert.False(t, ok)
}

func TestCache_Invalidate(t *testing.T) {
	c := NewCache(5*time.Minute, nil)
	c.SetModels("all", []*VerifiedModel{{ID: "gpt-4o"}})
	_, ok := c.GetModels("all")
	require.True(t, ok)

	c.Invalidate("all")
	_, ok = c.GetModels("all")
	assert.False(t, ok)
}

func TestCache_InvalidateAll(t *testing.T) {
	c := NewCache(5*time.Minute, nil)
	c.SetModels("all", []*VerifiedModel{{ID: "gpt-4o"}})
	c.SetModels("openai", []*VerifiedModel{{ID: "gpt-4"}})

	c.InvalidateAll()
	_, ok := c.GetModels("all")
	assert.False(t, ok)
	_, ok = c.GetModels("openai")
	assert.False(t, ok)
}

func TestCache_Eviction(t *testing.T) {
	c := NewCache(5*time.Minute, nil)
	c.maxSize = 3
	c.SetModels("a", []*VerifiedModel{{ID: "1"}})
	c.SetModels("b", []*VerifiedModel{{ID: "2"}})
	c.SetModels("c", []*VerifiedModel{{ID: "3"}})
	c.SetModels("d", []*VerifiedModel{{ID: "4"}})

	// One of the first three should be evicted
	count := 0
	for _, key := range []string{"a", "b", "c", "d"} {
		if _, ok := c.GetModels(key); ok {
			count++
		}
	}
	assert.Equal(t, 3, count)
}

func TestCache_GetModelScore(t *testing.T) {
	c := NewCache(5*time.Minute, nil)
	c.SetScores(map[string]float64{"gpt-4o": 9.1})

	score, ok := c.GetModelScore("gpt-4o")
	require.True(t, ok)
	assert.Equal(t, 9.1, score)

	_, ok = c.GetModelScore("missing")
	assert.False(t, ok)
}
