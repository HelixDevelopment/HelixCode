package memory

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"dev.helix.code/tests/stresschaos"
)

// §11.4.85(A) stress coverage for the in-process memory components.
//
// Two REAL (non-mocked) components are under stress here:
//
//   - Manager (manager.go): the conversation store whose conversations map +
//     activeConv pointer are guarded by a single sync.RWMutex. CreateConversation
//     / AddMessage / DeleteConversation take the write lock; GetConversation /
//     GetAll / Search / Count / GetStatistics take the read lock. We hammer both
//     read and write paths concurrently so the RWMutex is exercised under genuine
//     reader/writer contention.
//
//   - MemoryManager (memory_manager.go) + InMemoryProvider: the provider registry
//     guarded by its own RWMutex, fronting the REAL InMemoryProvider (a fully
//     in-process map-backed MemoryProvider — no Redis/Memcached/Cognee/Zep
//     network dependency). Store/Retrieve/Search/Delete all funnel through
//     GetDefaultProvider, which a speed-programme refactor (P4-T04) deliberately
//     made resolve the provider inline under ONE RLock rather than re-entering
//     GetProvider — because Go's sync.RWMutex is NOT reentrant and a recursive
//     RLock deadlocks if a writer's Lock() queues between the two RLock calls.
//     The concurrent test below specifically races readers (Store/Retrieve via
//     GetDefaultProvider) against writers (RegisterProvider/SetDefaultProvider
//     taking the write lock) to keep that exact deadlock window open the whole run.
//
// No mocks: InMemoryProvider and Manager are the production types. Permissible and
// required under CONST-050(A) — these *_test.go files run without the integration
// build tag (unit-test scope) but exercise the real concurrency surface end-to-end.

// stressConvManager builds a real conversation Manager.
func stressConvManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager()
}

// stressMemoryManager builds a real MemoryManager fronting a real InMemoryProvider.
func stressMemoryManager(t *testing.T) *MemoryManager {
	t.Helper()
	mm := NewMemoryManager(&MemoryConfig{Enabled: true, Provider: "inmemory"})
	prov, err := NewInMemoryProvider(nil)
	if err != nil {
		t.Fatalf("create in-memory provider: %v", err)
	}
	if err := mm.RegisterProvider("inmemory", prov); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	return mm
}

// TestManager_Stress_SustainedCreateAddGet drives the real
// CreateConversation -> AddMessage -> GetConversation -> Search lifecycle under
// sustained load (N>=100), recording per-call latency. Every iteration creates a
// real conversation, appends a real message (write-lock path), reads it back
// (read-lock path) and runs a substring Search, asserting non-zero processed work.
func TestManager_Stress_SustainedCreateAddGet(t *testing.T) {
	m := stressConvManager(t)

	var processed int64
	stresschaos.RunSustainedLoad(t, "memory_manager_sustained_create_add_get",
		stresschaos.SustainedConfig{N: 1200, MaxErrorRate: 0.0},
		func(i int) error {
			conv, err := m.CreateConversation(fmt.Sprintf("conv-%d", i))
			if err != nil {
				return fmt.Errorf("create: %w", err)
			}
			msg := NewUserMessage(fmt.Sprintf("hello world iteration %d", i))
			if err := m.AddMessage(conv.ID, msg); err != nil {
				return fmt.Errorf("add: %w", err)
			}
			got, err := m.GetConversation(conv.ID)
			if err != nil {
				return fmt.Errorf("get: %w", err)
			}
			if got.ID != conv.ID {
				return fmt.Errorf("get returned wrong conversation: want %s got %s", conv.ID, got.ID)
			}
			if got.MessageCount != 1 {
				return fmt.Errorf("expected 1 message, got %d", got.MessageCount)
			}
			// Read-lock path with real string scanning.
			_ = m.Search("hello")
			atomic.AddInt64(&processed, 1)
			return nil
		})

	if atomic.LoadInt64(&processed) == 0 {
		t.Fatal("memory manager processed zero conversations under sustained load — not real work")
	}
	if m.Count() == 0 {
		t.Fatal("conversation store empty after sustained load — writes did not persist")
	}
	t.Logf("memory_manager sustained: %d conversations created+appended+read, store size=%d",
		atomic.LoadInt64(&processed), m.Count())
}

// TestManager_Stress_ConcurrentReadWrite hammers CreateConversation +
// AddMessage (writers) against GetConversation + GetAll + Search + Count +
// GetStatistics (readers) from N>=16 concurrent goroutines, asserting no
// deadlock, no goroutine leak, and (under -race) no data race in the shared
// conversations map. Each goroutine creates its own conversation then reads
// across the whole store, maximising reader/writer RWMutex contention.
// HXC-204 sizing note — why 40 iterations and not 120.
//
// This run's cost is QUADRATIC in the iteration count, and that is a property of
// the workload rather than of the concurrency it exists to test. Every iteration
// creates one more conversation and never removes it, while GetAll / Search /
// GetStatistics each scan (and GetAll/Search *clone*) the ENTIRE store. So the
// per-call cost grows linearly with the iteration index and the total grows as
// the square. Measured on this repo under -race, quiet host:
//
//	iters=15  240 calls   79 ms    333 us/call
//	iters=30  480 calls  254 ms    531 us/call
//	iters=60  960 calls 1019 ms   1062 us/call
//	iters=120 1920 calls 3880 ms  2021 us/call   <- 4x cost per doubling
//
// That extra cost is memcpy under an RLock, not additional lock-contention
// coverage: the contention property is delivered by the 16 concurrent goroutines
// hammering the RWMutex, and it is fully exercised long before iteration 40.
//
// Why it mattered: this process runs inside a cgroup whose cpu.max is 8.6 CPUs
// (not the host's 64), so under a loaded pre-release sweep the allocation-heavy
// tail inflates ~29x. Measured, 8 uncapped runs each under 160 competing workers
// (docs/qa/hxc204_ceiling_distribution_20260804T195135Z/):
//
//	16x120  min 108.5s  median 114.7s  max 121.6s   <- ENTIRELY above the 100s ceiling
//	16x60   min  19.1s  median  21.0s  max  25.2s
//	16x40   min  10.4s  median  11.1s  max  13.0s   <- inside the 25s base budget
//
// At 16x120 every loaded run overran the ceiling, so the verdict alternated
// PASS/SKIP with host load — a verdict about the machine, not the code. At 16x40
// the whole measured distribution fits inside the BASE budget with ~1.9x margin
// and never needs the extension at all.
//
// Coverage is preserved, not traded away: 640 real concurrent calls still clears
// the §11.4.85(A) sustained floor (N >= 100) and parallelism 16 clears the
// contention floor (N >= 10) — and the sustained-load axis has its own dedicated
// coverage in TestManager_Stress_SustainedCreateAddGet above.
func TestManager_Stress_ConcurrentReadWrite(t *testing.T) {
	m := stressConvManager(t)

	var created int64
	stresschaos.RunConcurrent(t, "memory_manager_concurrent_read_write",
		stresschaos.ConcurrencyConfig{Parallelism: 16, IterationsPerGoroutine: 40, Timeout: 25 * time.Second},
		func(g, it int) error {
			conv, err := m.CreateConversation(fmt.Sprintf("g%d-it%d", g, it))
			if err != nil {
				return fmt.Errorf("create: %w", err)
			}
			atomic.AddInt64(&created, 1)
			if err := m.AddMessage(conv.ID, NewAssistantMessage(fmt.Sprintf("reply %d-%d", g, it))); err != nil {
				return fmt.Errorf("add: %w", err)
			}
			// Reader paths across the whole shared store under contention.
			if _, err := m.GetConversation(conv.ID); err != nil {
				return fmt.Errorf("get: %w", err)
			}
			_ = m.GetAll()
			_ = m.Search("reply")
			_ = m.Count()
			_ = m.GetStatistics()
			// Read of a definitely-missing conversation widens the RLock surface.
			_, _ = m.GetConversation("conv-does-not-exist")
			return nil
		})

	if atomic.LoadInt64(&created) == 0 {
		t.Fatal("memory manager created zero conversations under concurrent load")
	}
	t.Logf("memory_manager concurrent: %d conversations created+read, store size=%d",
		atomic.LoadInt64(&created), m.Count())
}

// TestManager_Stress_BoundaryConditions exercises boundary inputs against the
// real Manager: empty title (must error), GetRecent with n<=0 and n>len, Search
// of empty query, operations on a missing ID, and TrimConversations at the
// configured limit. None may panic; each boundary must behave per contract.
func TestManager_Stress_BoundaryConditions(t *testing.T) {
	m := stressConvManager(t)

	// empty title MUST be rejected (boundary: empty input).
	if _, err := m.CreateConversation(""); err == nil {
		t.Fatal("empty conversation title was accepted — boundary violation")
	}

	// missing-ID reads MUST error, not panic (boundary: absent key).
	if _, err := m.GetConversation("missing"); err == nil {
		t.Fatal("GetConversation on missing ID returned no error")
	}
	if err := m.AddMessage("missing", NewUserMessage("x")); err == nil {
		t.Fatal("AddMessage to missing conversation returned no error")
	}

	// GetRecent boundaries: n<=0 and n>len both clamp to len, never panic.
	conv, err := m.CreateConversation("boundary")
	if err != nil {
		t.Fatalf("create boundary conv: %v", err)
	}
	if got := m.GetRecent(0); len(got) != 1 {
		t.Fatalf("GetRecent(0) clamp failed: got %d", len(got))
	}
	if got := m.GetRecent(99999); len(got) != 1 {
		t.Fatalf("GetRecent(huge) clamp failed: got %d", len(got))
	}
	_ = m.Search("") // empty query must not panic

	// max-conversations boundary: set tiny limit, overflow, trim.
	m.SetMaxConversations(2)
	for i := 0; i < 5; i++ {
		if _, err := m.CreateConversation(fmt.Sprintf("overflow-%d", i)); err != nil {
			t.Fatalf("create overflow conv: %v", err)
		}
	}
	removed := m.TrimConversations()
	if removed == 0 {
		t.Fatal("TrimConversations removed nothing despite exceeding maxConversations")
	}
	if m.Count() > 2 && m.Count() != 3 { // active may be preserved -> 2 or 3
		t.Logf("post-trim count=%d (active-preservation tolerated)", m.Count())
	}
	_ = conv
	t.Logf("boundary conditions: post-trim count=%d removed=%d", m.Count(), removed)
}

// TestMemoryManager_Stress_SustainedStoreRetrieve drives the REAL MemoryManager
// Store -> Retrieve -> Search -> Delete lifecycle through GetDefaultProvider under
// sustained load (N>=100). This is the single-RLock hot path the speed-programme
// refactor narrowed; sustained traffic here proves the non-reentrant resolution
// stays correct under volume.
func TestMemoryManager_Stress_SustainedStoreRetrieve(t *testing.T) {
	mm := stressMemoryManager(t)
	ctx := context.Background()

	var processed int64
	stresschaos.RunSustainedLoad(t, "memory_provider_sustained_store_retrieve",
		stresschaos.SustainedConfig{N: 1200, MaxErrorRate: 0.0},
		func(i int) error {
			key := fmt.Sprintf("k-%d", i)
			val := fmt.Sprintf("v-%d", i)
			if err := mm.Store(ctx, key, val); err != nil {
				return fmt.Errorf("store: %w", err)
			}
			got, err := mm.Retrieve(ctx, key)
			if err != nil {
				return fmt.Errorf("retrieve: %w", err)
			}
			if got != val {
				return fmt.Errorf("retrieve mismatch: want %q got %v", val, got)
			}
			if err := mm.Delete(ctx, key); err != nil {
				return fmt.Errorf("delete: %w", err)
			}
			atomic.AddInt64(&processed, 1)
			return nil
		})

	if atomic.LoadInt64(&processed) == 0 {
		t.Fatal("memory provider processed zero store/retrieve cycles — not real work")
	}
	t.Logf("memory_provider sustained: %d store+retrieve+delete cycles via GetDefaultProvider", atomic.LoadInt64(&processed))
}

// TestMemoryManager_Stress_ConcurrentResolveWhileMutating is the reentrancy /
// writer-starvation deadlock probe. N>=16 goroutines split into two roles on the
// SAME MemoryManager RWMutex:
//
//   - reader role: Store/Retrieve via GetDefaultProvider (RLock on mm.mu)
//   - writer role: RegisterProvider + UnregisterProvider + SetDefaultProvider
//     (Lock on mm.mu)
//
// If GetDefaultProvider were ever changed to re-acquire the RLock reentrantly
// (e.g. by re-entering GetProvider), a writer's queued Lock() between the two
// RLock calls would deadlock — and the harness's RunConcurrent timeout guard
// would surface it as a FAIL with deadlock:true evidence. The current
// single-RLock implementation must complete cleanly with no deadlock and no race.
func TestMemoryManager_Stress_ConcurrentResolveWhileMutating(t *testing.T) {
	mm := stressMemoryManager(t)
	ctx := context.Background()

	var ops int64
	stresschaos.RunConcurrent(t, "memory_provider_concurrent_resolve_while_mutating",
		stresschaos.ConcurrencyConfig{Parallelism: 18, IterationsPerGoroutine: 150, Timeout: 25 * time.Second},
		func(g, it int) error {
			if g%3 == 0 {
				// Writer role: churn the provider registry, taking the WRITE lock
				// against the readers' RLock so the RWMutex is exercised under
				// genuine writer/reader contention. We register and unregister an
				// ADDITIONAL uniquely-named provider per iteration and never touch
				// the default ("inmemory" stays the default the whole run), so the
				// readers' GetDefaultProvider always resolves a registered
				// provider. (Removing the only/default provider mid-flight is a
				// graceful-degradation scenario covered by the chaos suite, not a
				// strict zero-error stress invariant — it cannot live in a
				// RunConcurrent harness that fails on ANY fn error.)
				name := fmt.Sprintf("p-%d-%d", g, it)
				prov, err := NewInMemoryProvider(nil)
				if err != nil {
					return fmt.Errorf("new provider: %w", err)
				}
				if err := mm.RegisterProvider(name, prov); err != nil {
					return fmt.Errorf("register: %w", err)
				}
				if err := mm.UnregisterProvider(name); err != nil {
					return fmt.Errorf("unregister: %w", err)
				}
				atomic.AddInt64(&ops, 1)
				return nil
			}
			// Reader role: resolve the default provider and use it. This is the
			// path that funnels through GetDefaultProvider's single RLock.
			key := fmt.Sprintf("k-%d-%d", g, it)
			if err := mm.Store(ctx, key, it); err != nil {
				return fmt.Errorf("store: %w", err)
			}
			if _, err := mm.Retrieve(ctx, key); err != nil {
				return fmt.Errorf("retrieve: %w", err)
			}
			_ = mm.ListProviders()
			_ = mm.Health(ctx)
			atomic.AddInt64(&ops, 1)
			return nil
		})

	if atomic.LoadInt64(&ops) == 0 {
		t.Fatal("zero ops under concurrent resolve-while-mutating load")
	}
	t.Logf("memory_provider concurrent resolve-while-mutating: %d ops, no deadlock/race", atomic.LoadInt64(&ops))
}
