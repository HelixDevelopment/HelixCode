package verifier

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dev.helix.code/tests/stresschaos"
)

// §11.4.85 STRESS suite for internal/verifier — exercises the REAL in-process
// concurrency-rich components (HealthMonitor circuit breaker, Cache TTL store,
// EventPublisher fan-out) under sustained load + concurrent contention + boundary
// conditions. No mocks of the unit under test. Run under -race to surface data
// races. Anything requiring a live LLMsVerifier server/network is honestly skipped.

// inMemoryRedis is a real, minimal, in-process RedisClient implementation used as
// the Cache's optional second tier. Per the task brief a real in-process impl in
// the test file is acceptable for the RedisClient seam (it is not a mock of the
// Cache under test — it is a genuine working key/value store with TTL semantics).
type inMemoryRedis struct {
	mu   sync.RWMutex
	data map[string]redisVal
}

type redisVal struct {
	value   string
	expires time.Time
}

func newInMemoryRedis() *inMemoryRedis {
	return &inMemoryRedis{data: make(map[string]redisVal)}
}

func (r *inMemoryRedis) Get(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.data[key]
	if !ok {
		return "", fmt.Errorf("redis: key %q not found", key)
	}
	if !v.expires.IsZero() && time.Now().After(v.expires) {
		return "", fmt.Errorf("redis: key %q expired", key)
	}
	return v.value, nil
}

func (r *inMemoryRedis) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	r.data[key] = redisVal{value: value, expires: exp}
	return nil
}

func sampleModels(n int, provider string) []*VerifiedModel {
	out := make([]*VerifiedModel, n)
	for i := 0; i < n; i++ {
		out[i] = &VerifiedModel{
			ID:           fmt.Sprintf("%s/model-%d", provider, i),
			Name:         fmt.Sprintf("Model %d", i),
			Provider:     provider,
			OverallScore: float64(i % 10),
			Source:       "verifier",
		}
	}
	return out
}

// --- HealthMonitor ---------------------------------------------------------

// TestStress_HealthMonitor_SustainedRecordSuccess hammers RecordSuccess + State
// N>=100 times and records p50/p95/p99.
func TestStress_HealthMonitor_SustainedRecordSuccess(t *testing.T) {
	h := NewHealthMonitor(5, 3, time.Second)
	rep := stresschaos.RunSustainedLoad(t, "verifier_health_record_success",
		stresschaos.SustainedConfig{N: 2000}, func(i int) error {
			h.RecordSuccess()
			_ = h.State()
			_ = h.AllowRequest()
			return nil
		})
	if rep.N < 100 {
		t.Fatalf("sustained N=%d below floor", rep.N)
	}
}

// TestStress_HealthMonitor_ConcurrentMixed hammers RecordSuccess / RecordFailure /
// State / AllowRequest from >=10 goroutines. Under -race this proves the RWMutex
// guards all shared state with no torn reads.
func TestStress_HealthMonitor_ConcurrentMixed(t *testing.T) {
	h := NewHealthMonitor(5, 3, 50*time.Millisecond)
	stresschaos.RunConcurrent(t, "verifier_health_concurrent_mixed",
		stresschaos.ConcurrencyConfig{Parallelism: 16, IterationsPerGoroutine: 300},
		func(g, it int) error {
			switch (g + it) % 4 {
			case 0:
				h.RecordSuccess()
			case 1:
				h.RecordFailure()
			case 2:
				_ = h.State()
			default:
				_ = h.AllowRequest()
			}
			// State must always be one of the three valid enum values (no torn read).
			s := h.State()
			if s != CircuitClosed && s != CircuitHalfOpen && s != CircuitOpen {
				return fmt.Errorf("torn/invalid circuit state observed: %d", s)
			}
			return nil
		})
}

// TestStress_HealthMonitor_StateMachineConsistency drives the breaker through a
// deterministic open->half-open->closed cycle and asserts every transition.
func TestStress_HealthMonitor_StateMachineConsistency(t *testing.T) {
	h := NewHealthMonitor(3, 2, 10*time.Millisecond)
	if h.State() != CircuitClosed {
		t.Fatalf("expected initial CircuitClosed, got %d", h.State())
	}
	for i := 0; i < 3; i++ {
		h.RecordFailure()
	}
	if h.State() != CircuitOpen {
		t.Fatalf("expected CircuitOpen after threshold failures, got %d", h.State())
	}
	if h.AllowRequest() {
		t.Fatal("open circuit must block requests before half-open timeout")
	}
	time.Sleep(20 * time.Millisecond)
	if !h.AllowRequest() {
		t.Fatal("open circuit must allow a probe after half-open timeout")
	}
	// A success transitions Open -> HalfOpen, then enough successes -> Closed.
	h.RecordSuccess()
	if h.State() != CircuitHalfOpen {
		t.Fatalf("expected CircuitHalfOpen after first success, got %d", h.State())
	}
	h.RecordSuccess()
	if h.State() != CircuitClosed {
		t.Fatalf("expected CircuitClosed after recovery threshold, got %d", h.State())
	}
}

// --- Cache -----------------------------------------------------------------

// TestStress_Cache_SustainedSetGet (memory-only) sustains Set+Get N times.
func TestStress_Cache_SustainedSetGet(t *testing.T) {
	c := NewCache(time.Minute, nil)
	models := sampleModels(20, "openai")
	rep := stresschaos.RunSustainedLoad(t, "verifier_cache_set_get_mem",
		stresschaos.SustainedConfig{N: 2000}, func(i int) error {
			c.SetModels("openai", models)
			if got, ok := c.GetModels("openai"); !ok || len(got) != len(models) {
				return fmt.Errorf("cache miss/short read at i=%d ok=%v len=%d", i, ok, len(got))
			}
			return nil
		})
	if rep.ErrorRate != 0 {
		t.Fatalf("cache sustained error rate %.4f", rep.ErrorRate)
	}
}

// TestStress_Cache_ConcurrentTiered hammers Set/Get/Invalidate/SetScores with the
// real in-process Redis second tier from >=10 goroutines. Proves the Cache RWMutex
// and the Redis seam are concurrency-safe (under -race).
func TestStress_Cache_ConcurrentTiered(t *testing.T) {
	c := NewCache(time.Minute, newInMemoryRedis())
	providers := []string{"openai", "anthropic", "gemini", "ollama"}
	stresschaos.RunConcurrent(t, "verifier_cache_concurrent_tiered",
		stresschaos.ConcurrencyConfig{Parallelism: 20, IterationsPerGoroutine: 200},
		func(g, it int) error {
			p := providers[(g+it)%len(providers)]
			switch (g + it) % 5 {
			case 0:
				c.SetModels(p, sampleModels(5, p))
			case 1:
				_, _ = c.GetModels(p)
			case 2:
				_, _ = c.GetModelsStale(p)
			case 3:
				c.SetScores(map[string]float64{p + "/m": float64(it)})
				_, _ = c.GetModelScore(p + "/m")
			default:
				c.Invalidate(p)
			}
			return nil
		})
}

// TestStress_Cache_ConcurrentEviction drives the eviction path under contention by
// inserting far more than maxSize distinct keys from many goroutines. The cache
// must never grow unbounded and must not race during eviction.
func TestStress_Cache_ConcurrentEviction(t *testing.T) {
	c := NewCache(time.Minute, nil)
	c.maxSize = 64 // shrink so eviction fires constantly
	stresschaos.RunConcurrent(t, "verifier_cache_concurrent_eviction",
		stresschaos.ConcurrencyConfig{Parallelism: 12, IterationsPerGoroutine: 500},
		func(g, it int) error {
			key := fmt.Sprintf("p-%d-%d", g, it)
			c.SetModels(key, sampleModels(1, key))
			return nil
		})
	c.mu.RLock()
	size := len(c.entries)
	c.mu.RUnlock()
	if size > c.maxSize+1 {
		t.Fatalf("cache exceeded maxSize after eviction: size=%d max=%d", size, c.maxSize)
	}
}

// --- Cache boundary conditions ---------------------------------------------

func TestStress_Cache_Boundary(t *testing.T) {
	// empty: nil redis, zero ttl normalises, missing key miss.
	c := NewCache(0, nil)
	if _, ok := c.GetModels("never-set"); ok {
		t.Fatal("boundary empty: expected miss for unset provider")
	}
	// empty slice stored is still a hit (len 0).
	c.SetModels("empty", []*VerifiedModel{})
	if got, ok := c.GetModels("empty"); !ok || len(got) != 0 {
		t.Fatalf("boundary empty-slice: ok=%v len=%d", ok, len(got))
	}
	// max-ish: a large model set round-trips.
	big := sampleModels(5000, "big")
	c.SetModels("big", big)
	if got, ok := c.GetModels("big"); !ok || len(got) != 5000 {
		t.Fatalf("boundary max: ok=%v len=%d", ok, len(got))
	}
	// off-by-one TTL expiry boundary — driven by an injected clock (cache.go
	// `Cache.now`) rather than time.Sleep. A sleep only guarantees a LOWER
	// bound on elapsed time, so the previous 15ms-TTL / 20ms-sleep form left a
	// 10ms slack that host load routinely overshot, making the verdict a
	// function of machine load instead of of the code under test (§11.4.50).
	// Advancing the clock makes every boundary below exact to the nanosecond.
	clk := newTestClock()
	short := NewCache(15*time.Millisecond, nil)
	short.now = clk.Now
	short.SetModels("p", sampleModels(1, "p"))
	if _, ok := short.GetModels("p"); !ok {
		t.Fatal("boundary ttl: fresh entry should hit")
	}
	// Exactly at TTL the entry is still fresh: cache.go:80 misses only when
	// age is STRICTLY greater than ttl.
	clk.Advance(15 * time.Millisecond)
	if _, ok := short.GetModels("p"); !ok {
		t.Fatal("boundary ttl: entry at exactly TTL should still hit (miss condition is age > ttl)")
	}
	// One nanosecond past TTL it must miss.
	clk.Advance(time.Nanosecond)
	if _, ok := short.GetModels("p"); ok {
		t.Fatal("boundary ttl: expired entry should miss")
	}
	// stale window (up to 2x TTL) still returns.
	short.SetModels("q", sampleModels(1, "q"))
	clk.Advance(20 * time.Millisecond)
	if _, ok := short.GetModelsStale("q"); !ok {
		t.Fatal("boundary stale: entry within 2x TTL should be stale-readable")
	}
	// Exactly at 2x TTL (30ms) still stale-readable; cache.go:95 misses only
	// when age is strictly greater than 2*ttl.
	clk.Advance(10 * time.Millisecond)
	if _, ok := short.GetModelsStale("q"); !ok {
		t.Fatal("boundary stale: entry at exactly 2x TTL should still be stale-readable")
	}
	// One nanosecond past 2x TTL it must miss.
	clk.Advance(time.Nanosecond)
	if _, ok := short.GetModelsStale("q"); ok {
		t.Fatal("boundary stale: entry past 2x TTL should miss")
	}
}

// --- EventPublisher --------------------------------------------------------

// TestStress_EventPublisher_ConcurrentPublishSubscribe hammers Subscribe + Publish
// from >=10 goroutines. Subscribers run in their own goroutines; we wait for all
// to drain so the goroutine-leak detector in RunConcurrent stays meaningful.
func TestStress_EventPublisher_ConcurrentPublishSubscribe(t *testing.T) {
	ep := NewEventPublisher()
	var delivered int64
	var wg sync.WaitGroup
	// a handful of stable subscribers
	for i := 0; i < 4; i++ {
		ep.Subscribe(func(ChangeEvent) {
			defer wg.Done()
			atomic.AddInt64(&delivered, 1)
		})
	}
	stresschaos.RunConcurrent(t, "verifier_events_concurrent_pubsub",
		stresschaos.ConcurrencyConfig{Parallelism: 10, IterationsPerGoroutine: 100, Timeout: 60 * time.Second},
		func(g, it int) error {
			wg.Add(4) // 4 subscribers each get one async delivery
			return ep.Publish(ChangeEvent{Type: "model.discovered", Timestamp: time.Now()})
		})
	wg.Wait() // drain all async deliveries before RunConcurrent snapshots goroutines
	if delivered == 0 {
		t.Fatal("expected at least one event delivered to subscribers")
	}
}
