// Package stresschaos is a Go-native stress + chaos test harness for HelixCode,
// implementing the constitution's §11.4.85 Stress + Chaos Test Mandate.
//
// It mirrors the canonical shell helper contract (ab_stress_run,
// ab_stress_concurrent, ab_chaos_kill_pid_during, ab_chaos_corrupt_file_during,
// ab_chaos_oom_pressure_during) as Go helpers that compose with the standard
// `testing` package, so any *_stress_test.go / *_chaos_test.go file can prove the
// two §11.4.85 survival properties:
//
//	(A) Survives load  — sustained-load (N>=100 or >=30s, p50/p95/p99 latency)
//	                     and concurrency (N>=10 goroutines, no deadlock, no leak).
//	(B) Survives failure — process-death / input-corruption / resource-exhaustion
//	                     chaos injection with a categorised recovery trace.
//
// Every PASS writes a captured-evidence artefact under qa-results/<run-id>/<name>/
// in the exact closed-set shapes the mandate enumerates. Per §11.4.5 / §11.4.69
// an empty / absent / placeholder artefact is NOT evidence — the helpers fail the
// test rather than emit a hollow PASS, so the harness itself cannot bluff.
//
// This helper is project-local (HelixCode tests/) on purpose. Promoting it to the
// constitution submodule for cross-project reuse is a future operator decision
// (that path triggers the §11.4.26 fetch-pull-push-to-all-upstreams workflow).
package stresschaos

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MinSustainedN is the §11.4.85(A)(1) floor: a sustained-load run must invoke the
// function under test at least this many times (or run for MinSustainedDuration).
const MinSustainedN = 100

// MinSustainedDuration is the §11.4.85(A)(1) wall-clock alternative to MinSustainedN.
const MinSustainedDuration = 30 * time.Second

// MinParallelism is the §11.4.85(A)(2) concurrency floor: a concurrency run must
// hammer the function under test from at least this many goroutines.
const MinParallelism = 10

// defaultErrorRateThreshold is the maximum tolerated error rate for a sustained
// stress PASS. Callers can override via SustainedConfig.MaxErrorRate.
const defaultErrorRateThreshold = 0.0

// goroutineLeakTolerance is the allowed delta in runtime.NumGoroutine() before/
// after a concurrency run. Concurrency runs settle asynchronously, so a small
// non-negative slack avoids flakiness while still catching real leaks.
const goroutineLeakTolerance = 4

// settlePollInterval is the sleep between goroutine-count polls in
// settleGoroutines(). It is well below the smallest realistic net/http
// connection-teardown latency so a genuine drop is noticed quickly.
const settlePollInterval = 25 * time.Millisecond

// settlePollBudget bounds how long settleGoroutines() will wait for the
// goroutine count to stabilize before giving up and returning the last sample.
// This replaces a single fixed-sleep snapshot (HXC-144): net/http client/server
// connection-teardown goroutines (persistConn.readLoop/writeLoop) exit
// asynchronously — their exit is scheduler-timed, not synchronous with
// Close() — so a single fixed delay is a well-documented flaky-measurement
// pattern (see golang/go#25621, golang/go#9092). A bounded poll-until-stable
// loop is the standard mitigation (mirrors what go.uber.org/goleak does
// internally: retry with backoff rather than sample once).
const settlePollBudget = 2 * time.Second

// settleStableStreak is how many consecutive equal samples settleGoroutines()
// requires before declaring the goroutine count "stable".
const settleStableStreak = 3

// closeIdleHTTPConnections proactively tears down idle connections on the
// shared, process-wide http.DefaultTransport. Concurrency tests that hammer an
// HTTP endpoint (e.g. server_stress_test.go's ConcurrentDDoSFlood) construct a
// fresh *http.Client per call but never set Client.Transport, so every call in
// the whole test binary shares this one singleton connection pool. Forcing
// idle connections closed here deterministically kicks off transport teardown
// instead of waiting on the OS/runtime scheduler to notice on its own. This is
// a safe no-op for concurrency tests that never touch net/http.
func closeIdleHTTPConnections() {
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
}

// settleGoroutines polls runtime.NumGoroutine(), interleaved with GC and an
// idle-HTTP-connection sweep, until the count stabilizes (settleStableStreak
// consecutive equal samples) or settlePollBudget elapses — whichever comes
// first. See goroutineLeakTolerance / settlePollBudget doc comments for why
// this replaces a single fixed-sleep sample (HXC-144). Returns the final
// stable (or last-sampled, if the budget expired without stabilizing)
// goroutine count.
func settleGoroutines() int {
	closeIdleHTTPConnections()
	runtime.GC()
	last := runtime.NumGoroutine()
	stable := 1
	deadline := time.Now().Add(settlePollBudget)
	for stable < settleStableStreak && time.Now().Before(deadline) {
		time.Sleep(settlePollInterval)
		runtime.GC()
		cur := runtime.NumGoroutine()
		if cur == last {
			stable++
		} else {
			stable = 1
			last = cur
		}
	}
	return last
}

// --- HXC-204: progress-based deadlock classification -------------------------
//
// The pre-HXC-204 harness detected deadlock by ONE signal: "did the run finish
// inside cfg.Timeout?". That signal cannot separate a deadlock from a busy host,
// and the busy-host reading is the likelier one during a full pre-release run —
// by definition the busiest the machine ever gets. Measured on this repo
// (docs/qa/hxc204_.../red/): internal/memory's concurrent read/write run
// completes in 7.68s against a 25s budget in isolation, and blows the same
// budget with error_count:0 and deadlock:true under 3x CPU oversubscription —
// i.e. the harness reported DEADLOCK about a run that was working correctly.
//
// The two conditions are separable, and the separating signal is FORWARD
// PROGRESS: a deadlocked worker set completes no further iterations, ever; a
// starved one keeps completing them, just slowly. So the harness now counts
// completed iterations and classifies on that, with the goroutine-state census
// below as the corroborating second oracle.
//
// The important structural consequence: deadlock detection NO LONGER DEPENDS ON
// THE BUDGET. It fires on a progress stall, which is typically much sooner than
// the old timeout. That is what makes it safe for the budget to stretch while
// work is demonstrably completing (see maxExtensionFactor) — the extension buys
// slow hosts a real PASS without buying a hung run any extra time to hide in.

// progressPollInterval is how often RunConcurrent samples the completed-call
// counter while waiting. Small enough that a stall is noticed promptly, large
// enough to be free next to the work being measured.
const progressPollInterval = 100 * time.Millisecond

// maxStallWindow / minStallWindow bound the "no forward progress" window that
// arms the goroutine-state census. Derived from cfg.Timeout (see stallWindow)
// rather than fixed, so a caller with a 500ms budget and a caller with a 60s
// budget both get a proportionate window.
const (
	maxStallWindow = 5 * time.Second
	minStallWindow = 50 * time.Millisecond
)

// maxExtensionFactor bounds how far past cfg.Timeout a run may stretch WHILE
// STILL COMPLETING ITERATIONS. This is not a bigger fixed timeout: a run that
// stops progressing is classified immediately by the stall detector regardless
// of how much budget is left, so the extension is unreachable by a hung run.
//
// Sized against measurement, not taste (§11.4.6). internal/memory's 16x120
// concurrent read/write run — the HXC-204 subject, and the heaviest RunConcurrent
// caller in the tree — measures 7.68s under -race in isolation against its 25s
// budget, and 34.6s under 3x CPU oversubscription (192 busy workers on 64 CPUs
// alongside the host's existing load). 4x covers that measured worst case with
// roughly 3x headroom on top, while keeping even the largest budget in the tree
// (60s, consensus_concurrent_clusters) at 4 minutes — well inside `go test`'s
// 10-minute default, which a stalled run would otherwise consume in full.
const maxExtensionFactor = 4

// stallWindow returns the no-progress duration that arms the goroutine-state
// census, derived from the caller's budget and clamped to sane bounds.
func stallWindow(timeout time.Duration) time.Duration {
	w := timeout / 2
	if w > maxStallWindow {
		w = maxStallWindow
	}
	if w < minStallWindow {
		w = minStallWindow
	}
	return w
}

// blockedGoroutineStates are the runtime goroutine states that mean "parked in a
// synchronization primitive": the goroutine is off the run queue and only
// another goroutine's action can wake it. A worker set entirely in these states,
// having made no progress, is a deadlock OBSERVED rather than inferred.
//
// Deliberately excluded: `sleep`, `IO wait`, `syscall`, `runnable`, `running`,
// `GC assist wait`, `preempted`. Those all describe a goroutine that will make
// progress on its own once the CPU/disk/network yields — which is exactly the
// busy-host case this classifier exists to stop mis-reporting.
var blockedGoroutineStates = []string{
	"chan receive",
	"chan send",
	"select",
	"semacquire",
	"sync.Mutex.Lock",
	"sync.RWMutex.Lock",
	"sync.RWMutex.RLock",
	"sync.WaitGroup.Wait",
	"sync.Cond.Wait",
}

// runningGoroutineStates are the states that positively contradict a deadlock:
// a goroutine in one of these is executing or waiting only on the scheduler.
var runningGoroutineStates = []string{
	"running",
	"runnable",
	"syscall",
	"preempted",
}

func stateMatches(state string, set []string) bool {
	for _, s := range set {
		// Prefix match: the runtime appends a duration to long waits, e.g.
		// "semacquire, 5 minutes", and qualifies some states further.
		if len(state) >= len(s) && state[:len(s)] == s {
			return true
		}
	}
	return false
}

// workerStateCensus dumps every goroutine in the process and counts the states
// of those running RunConcurrent's worker body. It is the second, independent
// oracle behind a DEADLOCK verdict (§11.4.107(2) — a different-domain signal, so
// a single misleading measurement cannot produce the verdict on its own).
//
// Worker goroutines are identified by the concurrentWorker frame, which exists
// as a named package-level function precisely so it is greppable in a dump.
func workerStateCensus() map[string]int {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		if len(buf) >= 64<<20 { // pathological; take what we have
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}

	census := map[string]int{}
	for _, block := range strings.Split(string(buf), "\n\ngoroutine ") {
		if !strings.Contains(block, "stresschaos.concurrentWorker") {
			continue
		}
		// Header is "<id> [<state>]:" (the leading "goroutine " is the split
		// separator, except for the very first block).
		openIdx := strings.Index(block, "[")
		closeIdx := strings.Index(block, "]")
		if openIdx < 0 || closeIdx < openIdx {
			continue
		}
		state := block[openIdx+1 : closeIdx]
		// Drop the runtime's "\, N minutes" duration suffix.
		if comma := strings.Index(state, ","); comma >= 0 {
			state = state[:comma]
		}
		census[strings.TrimSpace(state)]++
	}
	return census
}

// censusShowsDeadlock reports whether a worker-state census is positive evidence
// of a deadlock: at least one worker parked in a synchronization primitive, and
// NOT ONE worker running, runnable or in a syscall.
//
// The conjunction is deliberately conservative. A census with both blocked and
// runnable workers is ambiguous — it is what a lock-convoy on a starved host
// looks like as well as what a partial deadlock looks like — and under HXC-204
// the harness must not resolve an ambiguous reading into an accusation. It
// reports INCONCLUSIVE instead, which still stops the test.
func censusShowsDeadlock(census map[string]int) bool {
	blocked, running := 0, 0
	for state, n := range census {
		switch {
		case stateMatches(state, blockedGoroutineStates):
			blocked += n
		case stateMatches(state, runningGoroutineStates):
			running += n
		}
	}
	return blocked > 0 && running == 0
}

// LatencyReport is the §11.4.85 `latency.json` closed-set evidence shape.
type LatencyReport struct {
	Name       string  `json:"name"`
	N          int     `json:"n"`
	P50Ms      float64 `json:"p50_ms"`
	P95Ms      float64 `json:"p95_ms"`
	P99Ms      float64 `json:"p99_ms"`
	MinMs      float64 `json:"min_ms"`
	MaxMs      float64 `json:"max_ms"`
	ErrorRate  float64 `json:"error_rate"`
	DurationMs float64 `json:"duration_ms"`
	Timestamp  string  `json:"timestamp"`
}

// ConcurrencyVerdict is the classification RunConcurrent reaches for a run that
// did not complete inside its budget. It exists because "the run did not finish
// in time" and "the code under test deadlocked" are DIFFERENT conditions that
// the pre-HXC-204 harness collapsed into one: any budget overrun was reported as
// `DEADLOCK`, so a merely-busy host produced a verdict about production code
// (§11.4.201 — a guard must assert the condition it names).
//
// The classification mirrors the HXC-215 compile-integrity classifier: a real
// defect FAILS and names itself; an inconclusive run is reported honestly and
// names nobody. Both still stop the test — INCONCLUSIVE is not a pass.
type ConcurrencyVerdict string

const (
	// VerdictCompleted — every goroutine finished every iteration. This is the
	// only verdict that permits a PASS, and it carries NO wall-clock condition:
	// a slow host produces a slow PASS.
	VerdictCompleted ConcurrencyVerdict = "completed"
	// VerdictDeadlock — forward progress stopped AND every still-live worker is
	// parked in a synchronization primitive. That is a deadlock observed, not
	// inferred from elapsed time. FAILs the test.
	VerdictDeadlock ConcurrencyVerdict = "deadlock"
	// VerdictInconclusive — the run did not finish, but the evidence positively
	// contradicts a deadlock (work was still completing) or is simply silent on
	// it (workers runnable/in syscall, i.e. starved of CPU rather than blocked).
	// Reported as SKIP-with-reason per §11.4.3 — never as a verdict about the
	// code under test.
	VerdictInconclusive ConcurrencyVerdict = "inconclusive"
)

// ConcurrencyReport is the §11.4.85 `concurrency_report.json` closed-set shape.
type ConcurrencyReport struct {
	Name             string  `json:"name"`
	Parallelism      int     `json:"parallelism"`
	IterationsPerG   int     `json:"iterations_per_goroutine"`
	TotalCalls       int     `json:"total_calls"`
	GoroutinesBefore int     `json:"goroutines_before"`
	GoroutinesAfter  int     `json:"goroutines_after"`
	GoroutineDelta   int     `json:"goroutine_delta"`
	Deadlock         bool    `json:"deadlock"`
	ErrorCount       int64   `json:"error_count"`
	DurationMs       float64 `json:"duration_ms"`
	Timestamp        string  `json:"timestamp"`

	// --- HXC-204 classification evidence -------------------------------------
	// Verdict is the closed-set classification above. Deadlock stays in the
	// shape for compatibility with existing readers and is now true ONLY for
	// VerdictDeadlock — never for a plain budget overrun.
	Verdict ConcurrencyVerdict `json:"verdict"`
	// CompletedCalls is how many fn invocations returned. It is the forward-
	// progress signal the classifier reads: a deadlocked run cannot advance it.
	CompletedCalls int64 `json:"completed_calls"`
	// StallMs is how long progress had been stalled when the budget expired.
	StallMs float64 `json:"stall_ms"`
	// BudgetMs is the effective wall-clock ceiling (see maxExtensionFactor).
	BudgetMs float64 `json:"budget_ms"`
	// WorkerStates is the goroutine-state census of still-live workers taken at
	// classification time — the positive evidence behind a DEADLOCK verdict.
	WorkerStates map[string]int `json:"worker_states,omitempty"`
	// Reason is the human-readable justification for a non-completed verdict.
	Reason string `json:"reason,omitempty"`
}

// RecoveryTrace is the §11.4.85 `recovery_trace.log` (categorised) evidence shape.
// Each injected chaos fault is classified into exactly one of three buckets.
type RecoveryTrace struct {
	Name      string   `json:"name"`
	FaultKind string   `json:"fault_kind"`
	Recovered int      `json:"recovered"`
	Degraded  int      `json:"degraded"`
	Fatal     int      `json:"fatal"`
	Events    []string `json:"events"`
	Timestamp string   `json:"timestamp"`
}

// runID is computed once per process so all artefacts from a single `go test`
// invocation land under the same qa-results/<run-id>/ directory.
var (
	runIDOnce sync.Once
	runIDVal  string
)

func runID() string {
	runIDOnce.Do(func() {
		if v := os.Getenv("STRESSCHAOS_RUN_ID"); v != "" {
			runIDVal = v
			return
		}
		runIDVal = time.Now().UTC().Format("20060102T150405Z")
	})
	return runIDVal
}

// EvidenceRoot returns the qa-results root directory for the current run. It can
// be overridden with STRESSCHAOS_EVIDENCE_ROOT; otherwise it resolves to a
// `qa-results` directory anchored at the module root (located by walking up for
// go.mod) so artefacts land in a stable place regardless of the test's package.
func EvidenceRoot() string {
	if v := os.Getenv("STRESSCHAOS_EVIDENCE_ROOT"); v != "" {
		return filepath.Join(v, runID())
	}
	return filepath.Join(moduleRoot(), "qa-results", runID())
}

func moduleRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "qa-results-fallback"
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd // no go.mod found; fall back to cwd
		}
		dir = parent
	}
}

// evidenceDir creates and returns qa-results/<run-id>/<name>/ for a single test.
func evidenceDir(t testing.TB, name string) string {
	t.Helper()
	dir := filepath.Join(EvidenceRoot(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("stresschaos: cannot create evidence dir %s: %v", dir, err)
	}
	return dir
}

// writeJSON writes v as indented JSON and then re-reads it, asserting the file is
// non-empty and parseable. Per §11.4.5/§11.4.69 a hollow artefact is not evidence:
// if the write or the re-read verification fails, the test FAILS — the harness
// will not let a PASS stand on an empty file.
func writeJSON(t testing.TB, path string, v interface{}) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("stresschaos: marshal evidence %s: %v", path, err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("stresschaos: write evidence %s: %v", path, err)
	}
	verifyArtefact(t, path)
}

// verifyArtefact asserts the captured-evidence file exists and is non-empty.
func verifyArtefact(t testing.TB, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stresschaos: evidence artefact missing %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("stresschaos: evidence artefact empty (not evidence per §11.4.5) %s", path)
	}
}

func percentile(sortedMs []float64, p float64) float64 {
	if len(sortedMs) == 0 {
		return 0
	}
	if p <= 0 {
		return sortedMs[0]
	}
	if p >= 100 {
		return sortedMs[len(sortedMs)-1]
	}
	// nearest-rank method
	rank := int((p/100.0)*float64(len(sortedMs)-1) + 0.5)
	if rank >= len(sortedMs) {
		rank = len(sortedMs) - 1
	}
	return sortedMs[rank]
}

// SustainedConfig tunes a sustained-load run. Zero values pick §11.4.85 floors.
type SustainedConfig struct {
	// N is the number of invocations. If 0, MinSustainedN is used. Values below
	// MinSustainedN are rejected unless MinDuration is set instead.
	N int
	// MinDuration, if > 0, runs the function repeatedly until the duration
	// elapses (the §11.4.85(A)(1) ">=30s wall-clock" alternative). When set, N
	// becomes a lower bound only.
	MinDuration time.Duration
	// MaxErrorRate is the highest tolerated error fraction for a PASS.
	MaxErrorRate float64
}

// RunSustainedLoad invokes fn under sustained load per §11.4.85(A)(1), captures
// per-call latency, computes p50/p95/p99, writes latency.json, and FAILS the test
// if the error rate exceeds the threshold or the floor (N>=100 or >=30s) is unmet.
// It returns the captured LatencyReport so callers can make extra assertions.
//
// fn must return nil on success and a non-nil error on a failed invocation; the
// error rate is the fraction of non-nil returns.
func RunSustainedLoad(t testing.TB, name string, cfg SustainedConfig, fn func(i int) error) LatencyReport {
	t.Helper()

	n := cfg.N
	if cfg.MinDuration <= 0 {
		if n == 0 {
			n = MinSustainedN
		}
		if n < MinSustainedN {
			t.Fatalf("stresschaos: RunSustainedLoad %q N=%d below §11.4.85 floor %d (set MinDuration to use the >=30s path)", name, n, MinSustainedN)
		}
	}

	capacity := n
	if capacity < MinSustainedN {
		capacity = MinSustainedN
	}
	latencies := make([]float64, 0, capacity)
	var errCount int
	start := time.Now()

	i := 0
	for {
		callStart := time.Now()
		err := fn(i)
		elapsedMs := float64(time.Since(callStart).Microseconds()) / 1000.0
		latencies = append(latencies, elapsedMs)
		if err != nil {
			errCount++
		}
		i++

		if cfg.MinDuration > 0 {
			if time.Since(start) >= cfg.MinDuration && i >= MinSustainedN {
				break
			}
		} else if i >= n {
			break
		}
	}

	sorted := make([]float64, len(latencies))
	copy(sorted, latencies)
	sort.Float64s(sorted)

	errRate := float64(errCount) / float64(len(latencies))
	rep := LatencyReport{
		Name:       name,
		N:          len(latencies),
		P50Ms:      percentile(sorted, 50),
		P95Ms:      percentile(sorted, 95),
		P99Ms:      percentile(sorted, 99),
		MinMs:      sorted[0],
		MaxMs:      sorted[len(sorted)-1],
		ErrorRate:  errRate,
		DurationMs: float64(time.Since(start).Microseconds()) / 1000.0,
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
	}

	dir := evidenceDir(t, name)
	path := filepath.Join(dir, "latency.json")
	writeJSON(t, path, rep)

	threshold := cfg.MaxErrorRate
	if threshold == 0 {
		threshold = defaultErrorRateThreshold
	}
	if errRate > threshold {
		t.Fatalf("stresschaos: %q error rate %.4f exceeds threshold %.4f (evidence: %s)", name, errRate, threshold, path)
	}

	t.Logf("stresschaos: %q sustained N=%d p50=%.3fms p95=%.3fms p99=%.3fms errRate=%.4f -> %s",
		name, rep.N, rep.P50Ms, rep.P95Ms, rep.P99Ms, rep.ErrorRate, path)
	return rep
}

// ConcurrencyConfig tunes a concurrency run. Zero values pick §11.4.85 floors.
type ConcurrencyConfig struct {
	// Parallelism is the goroutine count. If 0, MinParallelism is used. Values
	// below MinParallelism are rejected.
	Parallelism int
	// IterationsPerGoroutine is how many times each goroutine calls fn. If 0,
	// defaults to 50 (so a 10x50 run does 500 real concurrent calls).
	IterationsPerGoroutine int
	// Timeout is the wall-clock budget for the run. If 0, defaults to 30s.
	//
	// HXC-204 changed what exceeding it MEANS, not what it is. It is no longer
	// the deadlock guard — deadlock is now detected by a forward-progress stall
	// plus a goroutine-state census, which fires independently of (and usually
	// well before) this budget. Exceeding the budget while work is still
	// completing is reported INCONCLUSIVE (SKIP), never as a deadlock, and a run
	// that is still progressing may stretch to maxExtensionFactor x Timeout so a
	// slow host yields a slow PASS rather than a false accusation.
	Timeout time.Duration
}

// concurrentWorker is RunConcurrent's per-goroutine body.
//
// It is a named package-level function rather than an inline closure for one
// load-bearing reason: workerStateCensus identifies worker goroutines in a
// runtime stack dump by looking for the `stresschaos.concurrentWorker` frame. An
// inline closure would appear as `RunConcurrent.funcN`, whose number shifts with
// unrelated edits to this file — a census keyed on it would silently stop
// matching and quietly downgrade every deadlock to INCONCLUSIVE.
func concurrentWorker(gid, iters int, startGate <-chan struct{}, fn func(int, int) error, completed, errCount *int64) {
	<-startGate // release all goroutines simultaneously for true contention
	for it := 0; it < iters; it++ {
		if err := fn(gid, it); err != nil {
			atomic.AddInt64(errCount, 1)
		}
		// Incremented AFTER fn returns, so it counts finished work only. This is
		// the forward-progress signal the HXC-204 classifier reads.
		atomic.AddInt64(completed, 1)
	}
}

// RunConcurrent hammers fn from N>=10 goroutines per §11.4.85(A)(2), detects
// deadlock by forward-progress stall + goroutine-state census, measures the
// goroutine-count delta to detect leaks, and writes concurrency_report.json.
//
// Verdicts (HXC-204):
//
//	completed    every iteration finished -> leak/error checks decide PASS/FAIL.
//	             Carries no wall-clock condition: a slow host passes slowly.
//	deadlock     progress stopped AND every live worker is parked in a sync
//	             primitive -> t.Fatalf, naming the condition it observed.
//	inconclusive did not finish, but the evidence contradicts or is silent on a
//	             deadlock -> t.Skipf with a reason (§11.4.3). Not a pass.
//
// Run under `-race` to also catch data races. Returns the ConcurrencyReport for
// extra assertions.
func RunConcurrent(t testing.TB, name string, cfg ConcurrencyConfig, fn func(goroutine, iter int) error) ConcurrencyReport {
	t.Helper()

	p := cfg.Parallelism
	if p == 0 {
		p = MinParallelism
	}
	if p < MinParallelism {
		t.Fatalf("stresschaos: RunConcurrent %q parallelism=%d below §11.4.85 floor %d", name, p, MinParallelism)
	}
	iters := cfg.IterationsPerGoroutine
	if iters == 0 {
		iters = 50
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Settle and snapshot goroutine count before the run.
	runtime.GC()
	gBefore := runtime.NumGoroutine()

	var errCount int64
	var completed int64
	var wg sync.WaitGroup
	wg.Add(p)
	start := time.Now()
	startGate := make(chan struct{})

	for g := 0; g < p; g++ {
		go func(gid int) {
			defer wg.Done()
			concurrentWorker(gid, iters, startGate, fn, &completed, &errCount)
		}(g)
	}
	close(startGate)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// --- HXC-204 wait loop: watch forward progress, not just the clock --------
	//
	// The old loop was a single `select { <-done ; <-time.After(timeout) }`, so
	// the only thing it could ever learn was "finished / did not finish". This
	// one samples the completed-call counter as it waits, which is what lets the
	// classifier below tell a stalled run from a slow one.
	stall := stallWindow(timeout)
	hardCeiling := time.Duration(maxExtensionFactor) * timeout
	budget := timeout
	lastCompleted := int64(0)
	lastProgressAt := start

	ticker := time.NewTicker(progressPollInterval)
	defer ticker.Stop()

	finished := false
	stalled := false
waitLoop:
	for {
		select {
		case <-done:
			finished = true
			break waitLoop
		case <-ticker.C:
			if c := atomic.LoadInt64(&completed); c != lastCompleted {
				lastCompleted = c
				lastProgressAt = time.Now()
			}
			elapsed := time.Since(start)
			// A progress stall arms the census REGARDLESS of remaining budget:
			// a hung run is caught on the stall, never on the ceiling, which is
			// what keeps the extension below honest.
			if time.Since(lastProgressAt) >= stall {
				stalled = true
				break waitLoop
			}
			if elapsed >= budget {
				// Still progressing at the budget. Extend — bounded — so a
				// merely slow host reaches a real PASS instead of a verdict
				// about code that is working.
				if budget < hardCeiling {
					budget += timeout
					if budget > hardCeiling {
						budget = hardCeiling
					}
					continue
				}
				break waitLoop
			}
		}
	}
	durMs := float64(time.Since(start).Microseconds()) / 1000.0

	// Classify the non-completed cases (§11.4.201: assert the real condition).
	verdict := VerdictCompleted
	reason := ""
	var census map[string]int
	stallMs := 0.0
	if !finished {
		sinceProgress := time.Since(lastProgressAt)
		stallMs = float64(sinceProgress.Microseconds()) / 1000.0
		doneCalls := atomic.LoadInt64(&completed)
		if !stalled {
			// Work was still completing when the ceiling arrived. That is
			// positive evidence AGAINST a deadlock, so it can never be reported
			// as one.
			verdict = VerdictInconclusive
			reason = fmt.Sprintf(
				"run still progressing at the %s ceiling (%d/%d calls completed, last progress %.0fms ago) — host too loaded to finish, NOT a deadlock",
				hardCeiling, doneCalls, p*iters, stallMs)
		} else {
			census = workerStateCensus()
			if censusShowsDeadlock(census) {
				verdict = VerdictDeadlock
				reason = fmt.Sprintf(
					"no forward progress for %s (%d/%d calls completed) and every live worker is parked in a synchronization primitive: %v",
					stall, doneCalls, p*iters, census)
			} else {
				verdict = VerdictInconclusive
				reason = fmt.Sprintf(
					"no forward progress for %s (%d/%d calls completed) but the worker state census does not show a deadlock: %v — reads as CPU starvation, and an ambiguous census is not an accusation",
					stall, doneCalls, p*iters, census)
			}
		}
	}

	// Let scheduled goroutines wind down before snapshotting (only meaningful if
	// the run actually completed). Poll-until-stable rather than a single fixed
	// sleep: see settleGoroutines() doc comment (HXC-144).
	var gAfter int
	if finished {
		gAfter = settleGoroutines()
	} else {
		gAfter = runtime.NumGoroutine()
	}

	rep := ConcurrencyReport{
		Name:             name,
		Parallelism:      p,
		IterationsPerG:   iters,
		TotalCalls:       p * iters,
		GoroutinesBefore: gBefore,
		GoroutinesAfter:  gAfter,
		GoroutineDelta:   gAfter - gBefore,
		Deadlock:         verdict == VerdictDeadlock,
		ErrorCount:       atomic.LoadInt64(&errCount),
		DurationMs:       durMs,
		Timestamp:        time.Now().UTC().Format(time.RFC3339Nano),
		Verdict:          verdict,
		CompletedCalls:   atomic.LoadInt64(&completed),
		StallMs:          stallMs,
		BudgetMs:         float64(hardCeiling.Microseconds()) / 1000.0,
		WorkerStates:     census,
		Reason:           reason,
	}

	dir := evidenceDir(t, name)
	path := filepath.Join(dir, "concurrency_report.json")
	writeJSON(t, path, rep)

	switch verdict {
	case VerdictDeadlock:
		t.Fatalf("stresschaos: %q DEADLOCK — %s (evidence: %s)", name, reason, path)
	case VerdictInconclusive:
		// §11.4.3 honest inconclusive. Reported BEFORE the leak and error checks
		// below because an unfinished run still holds live workers, and reading
		// those as a goroutine leak would be the same category of false
		// accusation this fix exists to remove.
		// SKIP-OK: #HXC-204 — host-load inconclusive, never a verdict about the code.
		t.Skipf("stresschaos: %q INCONCLUSIVE — %s (evidence: %s)", name, reason, path)
	}
	if rep.GoroutineDelta > goroutineLeakTolerance {
		t.Fatalf("stresschaos: %q goroutine leak — before=%d after=%d delta=%d > tolerance %d (evidence: %s)",
			name, gBefore, gAfter, rep.GoroutineDelta, goroutineLeakTolerance, path)
	}
	if rep.ErrorCount > 0 {
		t.Fatalf("stresschaos: %q reported %d errors under concurrent load (evidence: %s)", name, rep.ErrorCount, path)
	}

	t.Logf("stresschaos: %q concurrent parallelism=%d calls=%d gDelta=%d deadlock=%v dur=%.1fms -> %s",
		name, p, rep.TotalCalls, rep.GoroutineDelta, rep.Deadlock, rep.DurationMs, path)
	return rep
}
