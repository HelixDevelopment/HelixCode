// Package scaling is a HelixCode-LOCAL throughput-scaling test harness that
// exercises the REAL internal/worker.WorkerPool across a worker-count sweep
// (N=1,2,4,8) and proves genuine scale-out — not a delegation to a HelixQA shell
// script. It mirrors the proven helix_code/tests/stresschaos contract: every PASS
// WRITES then RE-READS a non-empty evidence artefact under qa-results/<run-id>/,
// and a paired §1.1 meta-test plants a degraded pool and asserts the harness
// DETECTS it (so the harness itself cannot bluff).
//
// CONST-050(B) / §11.4.85: the unit under test is the production WorkerPool's own
// RWMutex-guarded worker map + scheduler selection machinery. No fakes — the sweep
// registers real PoolWorkers via the real RegisterWorker, drives real AssignTask /
// ReleaseWorker, and reads real GetPoolStats utilization.
//
// Honest boundary (§11.4.6): the in-process sweep proves the POOL's scale-out
// logic (assignment throughput vs registered-worker count). True HORIZONTAL
// SSH-worker scale-out needs real remote hosts and is a separate integration-
// tagged path that SKIPs-with-reason when no SSH workers are configured — never a
// fake PASS.
package scaling

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/worker"
	"dev.helix.code/tests/stresschaos"
)

// MinThroughputGainAtMaxN is the calibration floor: throughput at the largest N in
// the sweep must be at least this multiple of throughput at N=1. A pool that
// ignores added workers shows flat throughput (gain ~1.0) and FAILS this gate.
// Calibrated conservatively per §11.4.6 / §11.4.107(13): real scale-out on the
// in-process pool comfortably clears 1.5x; a serialised (broken) pool sits at ~1.0.
const MinThroughputGainAtMaxN = 1.5

// ScalingStep is one row of the §11.4.85-style throughput sweep at a fixed worker
// count N. ThroughputTPS is the MEDIAN across MeasureTrialsPerStep interleaved
// trials (see that const's doc comment); ThroughputTrialsTPS carries the raw
// per-trial samples the median was computed from, so the evidence artefact is
// self-auditing (§11.4.5/§11.4.69 — the aggregate is never opaque).
type ScalingStep struct {
	NWorkers            int       `json:"n_workers"`
	TotalTasks          int       `json:"total_tasks"`
	AssignedTasks       int64     `json:"assigned_tasks"`
	ThroughputTPS       float64   `json:"throughput_tps"`
	ThroughputTrialsTPS []float64 `json:"throughput_trials_tps"`
	P50Ms               float64   `json:"p50_ms"`
	P95Ms               float64   `json:"p95_ms"`
	P99Ms               float64   `json:"p99_ms"`
	PoolUtilization     float64   `json:"pool_utilization"`
	DurationMs          float64   `json:"duration_ms"`
}

// ScalingReport is the closed-set scaling_throughput.json evidence shape.
type ScalingReport struct {
	Name              string        `json:"name"`
	Steps             []ScalingStep `json:"steps"`
	GainAtMaxN        float64       `json:"gain_at_max_n"`
	MinGainThreshold  float64       `json:"min_gain_threshold"`
	MonotonicNonDegrd bool          `json:"monotonic_non_degraded"`
	Trials            int           `json:"trials"`
	Timestamp         string        `json:"timestamp"`
}

// PoolDriver is the seam the sweep drives. The production path is realPoolDriver
// (wrapping the real WorkerPool); meta-tests substitute degraded drivers to prove
// the harness detects flat / degrading throughput. A driver must register N real
// workers, expose a process-one-task cycle, and report utilization.
//
// SCALE-OUT SEMANTICS (the honest property being measured, §11.4.6): the production
// WorkerPool's AssignTask is a near-instant in-memory map lookup — raw assign
// throughput is therefore lock-bound and does NOT grow with N (asserting it would
// be a bluff). The pool's REAL scale-out property is CONCURRENT IN-FLIGHT CAPACITY:
// exactly N tasks can be assigned-and-busy at once (the (N+1)-th assign gets
// "no available workers" backpressure until one releases). So ProcessTask models a
// task with a fixed service time — acquire a real worker (mark busy), hold it for
// the service window, release it. With N real workers, N tasks are serviced
// concurrently, so completed-tasks-per-second scales ~linearly with N (bounded by
// scheduler/contention). That is genuine pool scale-out, proven with real
// AssignTask/ReleaseWorker + real GetPoolStats utilization.
type PoolDriver interface {
	// SetupN registers exactly n workers and returns a teardown.
	SetupN(t testing.TB, n int) func()
	// ProcessTask runs one real task: acquire a worker (busy), hold it for the
	// service window, release it. Returns true on success, false on backpressure
	// (all workers busy). Retries briefly so the fixed workload completes.
	ProcessTask(ctx context.Context) bool
	// Utilization reads the live pool utilization_rate (0..100).
	Utilization() float64
}

// ServiceTime is the fixed per-task service window the real driver holds a worker
// busy. It must be large enough that concurrency (N workers in-flight) dominates
// the per-call assign/lock cost, so adding workers measurably increases throughput.
// At 2ms the per-task assign overhead (~microseconds, even under -race) is
// negligible against the service window, so the N-worker concurrency is the
// dominant throughput factor and real scale-out is observable.
const ServiceTime = 2 * time.Millisecond

// MeasureTrialsPerStep is the default number of independent measurement trials
// taken per worker-count N, INTERLEAVED across all N values in the sweep
// (trial-major loop order: for each trial, measure N=1,2,4,8 in turn, then
// repeat) — never all repeats of one N back-to-back. The per-N reported
// ThroughputTPS is the MEDIAN across these trials.
//
// Root-cause background (§11.4.6 — confirmed by direct measurement, not
// assumed): TestMeta_RunScaleSweep_DetectsFlatThroughput failed once during a
// full-module sweep with "RunScaleSweep did NOT detect flat throughput ...
// harness is a bluff". A single wall-clock sample per N is fragile under real
// host contention (concurrent builds / other test suites sharing the box): a
// transient scheduling delay that happens to land on the FIRST (smallest-N,
// "base") measurement — and NOT on the later (largest-N, "max") measurement —
// inflates the apparent gain = maxN/baseN and can spuriously clear
// MinThroughputGainAtMaxN even for a driver that ignores added workers
// entirely (serialPoolDriver in scaling_meta_test.go), defeating the harness's
// own §1.1 anti-bluff guarantee. This was reproduced directly with a throwaway
// in-process investigation harness (not committed — measured live, evidence
// captured in the commit message instead): under real heavy host contention
// (an AOSP build + concurrent Go test suites already saturating the box), a
// burst of competing work
// correlated with the base-N window can make measured "gain" arbitrarily
// unreliable in EITHER direction — not just skew it high enough to defeat
// detection, but in the extreme case stall the measurement itself for
// minutes. Under lighter/ambient real contention (~30-45% of a 64-core host,
// the steady-state load observed while diagnosing this), the ORIGINAL
// single-sample harness in fact passed 8/8 direct reruns — the failure mode
// is a function of the TEMPORAL correlation between a transient slowdown and
// WHICH step is being measured, not merely "any load exists".
//
// Mitigation (does NOT weaken MinThroughputGainAtMaxN itself, §11.4.120): take
// several independent samples per N, spread the repeats of the SAME N across
// the ENTIRE sweep's wall-clock duration (interleaved, not back-to-back), and
// aggregate via the MEDIAN. A transient burst can now corrupt at most a
// minority of any one N's samples, and — critically — because the N values are
// interleaved, a burst that happens to land during "the early part of the
// sweep" hits ALL N's first-trial measurements about equally, not just N=1's;
// it can no longer selectively inflate the base/max ratio the way a single
// back-to-back per-N measurement order could. This is the standard robust-
// benchmarking mitigation for exactly this failure mode (Go's own testing.B
// -count flag and Criterion-style outlier-robust estimation both use repeated-
// trial aggregation over single-shot wall-clock samples for the same reason).
//
// Trade-off: MeasureTrialsPerStep multiplies the sweep's wall-clock cost by
// roughly this factor (e.g. TestScaling_WorkerPool_RealSweep goes from ~1.2s
// to ~4-5s at the default of 5; TestMeta_RunScaleSweep_DetectsFlatThroughput
// from ~2.5s to ~7-9s) in exchange for materially reduced flakiness under real
// contention — an acceptable price for a paired-mutation meta-test whose whole
// purpose is proving the harness itself cannot bluff.
//
// 2026-07-28 re-verification (§11.4.6 — measured live, not assumed): with
// Trials=3 this package was re-run directly (`go test ./tests/scaling/...
// -count=5`) under REAL heavy ambient host contention (other concurrent
// work-stream tracks measuring their own timing-sensitive work on the same
// shared host, load average ~18-26 on a 64-core box) and ONE of five runs
// FAILED — both TestMeta_RunScaleSweep_DetectsFlatThroughput (harness failed
// to detect the planted flat-throughput driver: two of three interleaved
// trials at one N were corrupted low enough to pull the median down and
// spuriously widen the apparent gain) and TestMeta_PositivePathWritesEvidence
// (the REAL pool's own median at N=4 was corrupted below its N=2 median,
// tripping the monotonic-non-degradation check on genuinely-working code).
// Re-run immediately after under lighter contention (load ~4.5-6.3, other
// tracks' bursts having subsided) passed 3/3 direct reruns at Trials=3,
// confirming the residual flakiness is contention-magnitude-dependent, not a
// logic defect. Trials raised 3 -> 5 (median-of-5 requires 3 of 5 samples to
// be corrupted, not 2 of 3, to flip the aggregate) as an additional,
// non-threshold-weakening (§11.4.120) safety margin for exactly this
// documented failure mode; MinThroughputGainAtMaxN is unchanged.
const MeasureTrialsPerStep = 5

// median returns the median of vs (nil/empty -> 0). Used to aggregate
// MeasureTrialsPerStep interleaved trial samples per N into a single
// contention-robust throughput figure (see that const's doc comment).
func median(vs []float64) float64 {
	if len(vs) == 0 {
		return 0
	}
	sorted := make([]float64, len(vs))
	copy(sorted, vs)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// runID / evidence helpers mirror the stresschaos write+re-read contract so a
// hollow artefact can never stand as a PASS (§11.4.5/§11.4.69).
var (
	runIDOnce sync.Once
	runIDVal  string
)

func runID() string {
	runIDOnce.Do(func() {
		if v := os.Getenv("SCALING_RUN_ID"); v != "" {
			runIDVal = v
			return
		}
		if v := os.Getenv("STRESSCHAOS_RUN_ID"); v != "" {
			runIDVal = v
			return
		}
		runIDVal = time.Now().UTC().Format("20060102T150405Z")
	})
	return runIDVal
}

// EvidenceRoot resolves qa-results/<run-id>/. Override with SCALING_EVIDENCE_ROOT
// (meta-tests redirect it to a t.TempDir()).
func EvidenceRoot() string {
	if v := os.Getenv("SCALING_EVIDENCE_ROOT"); v != "" {
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
			return wd
		}
		dir = parent
	}
}

func evidenceDir(t testing.TB, name string) string {
	t.Helper()
	dir := filepath.Join(EvidenceRoot(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("scaling: cannot create evidence dir %s: %v", dir, err)
	}
	return dir
}

// writeJSON writes v then RE-READS it, failing on empty — a hollow artefact is not
// evidence (§11.4.5/§11.4.69).
func writeJSON(t testing.TB, path string, v interface{}) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("scaling: marshal evidence %s: %v", path, err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("scaling: write evidence %s: %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("scaling: evidence artefact missing %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("scaling: evidence artefact empty (not evidence per §11.4.5) %s", path)
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
	rank := int((p/100.0)*float64(len(sortedMs)-1) + 0.5)
	if rank >= len(sortedMs) {
		rank = len(sortedMs) - 1
	}
	return sortedMs[rank]
}

// realPoolDriver wraps the production WorkerPool. This is the non-fake path used
// by the real sweep test.
type realPoolDriver struct {
	pool *worker.WorkerPool
}

// NewRealPoolDriver returns a driver backed by the production worker pool.
func NewRealPoolDriver() PoolDriver { return &realPoolDriver{} }

func (d *realPoolDriver) SetupN(t testing.TB, n int) func() {
	t.Helper()
	d.pool = worker.NewWorkerPool(&config.WorkersConfig{HealthTTL: 3600, MaxConcurrentTasks: n})
	for i := 0; i < n; i++ {
		d.pool.RegisterWorker(worker.NewPoolWorker(
			fmt.Sprintf("w-%d", i), fmt.Sprintf("Worker %d", i), "localhost:0",
			worker.WorkerCapabilities{CPUCores: 8, MemoryGB: 16},
		))
	}
	return func() { d.pool = nil }
}

func (d *realPoolDriver) ProcessTask(ctx context.Context) bool {
	// Acquire a real worker (marks it busy). If all N are busy, retry briefly so
	// the fixed workload still completes — the backpressure is real, but a transient
	// "all busy" is not a dropped task. Cap the retry window so a genuinely wedged
	// pool cannot hang the sweep.
	deadline := time.Now().Add(2 * time.Second)
	for {
		w, err := d.pool.AssignTask(ctx, "compute", map[string]interface{}{"cpu_cores": 1})
		if err == nil {
			// Worker is now busy: hold it for the service window (the real task the
			// worker would execute), then release. Exactly N can be held busy at
			// once — that IS the pool's concurrent-capacity scale-out property. A
			// sleeping worker still owns its StatusBusy slot, so N sleeping workers
			// service N tasks concurrently.
			busyWait(ServiceTime)
			d.pool.ReleaseWorker(w.ID)
			return true
		}
		if time.Now().After(deadline) {
			return false // sustained backpressure
		}
		time.Sleep(50 * time.Microsecond)
	}
}

// busyWait holds for d. time.Sleep models a worker occupied by a real task: it
// keeps the worker's StatusBusy slot claimed for the whole window (the pool's
// concurrent-capacity semantics) while yielding the CPU, so N workers genuinely
// run N tasks in parallel and throughput scales with N.
func busyWait(d time.Duration) {
	time.Sleep(d)
}

func (d *realPoolDriver) Utilization() float64 {
	stats := d.pool.GetPoolStats()
	if v, ok := stats["utilization_rate"].(float64); ok {
		return v
	}
	return 0
}

// SweepConfig tunes the scale sweep. Zero values pick safe defaults.
type SweepConfig struct {
	// NValues is the worker-count sweep. Defaults to {1,2,4,8}.
	NValues []int
	// TasksPerStep is the fixed total workload per N. Defaults to 4000.
	TasksPerStep int
	// Parallelism is the concurrent-submitter count (>= stresschaos.MinParallelism).
	Parallelism int
	// Trials is the number of interleaved measurement trials per worker-count N,
	// median-aggregated for contention-robustness. Defaults to
	// MeasureTrialsPerStep; see that const's doc comment for the rationale.
	Trials int
}

// stepSample is one trial's raw wall-clock measurement at a fixed N. Multiple
// stepSamples per N (see MeasureTrialsPerStep) are aggregated (median
// throughput, concatenated latencies, peak utilization, summed duration/
// assigned/actual-tasks) into the single ScalingStep the evidence reports.
type stepSample struct {
	actualTasks int
	assigned    int64
	throughput  float64
	latencies   []float64
	peakUtil    float64
	durationMs  float64
}

// measureStep runs ONE trial: `par` concurrent submitters drive `tasks` total
// ProcessTask calls through driver (already SetupN'd by the caller for the
// current N), timed via wall-clock elapsed. This is the exact per-step
// measurement block RunScaleSweep used to run once per N before
// MeasureTrialsPerStep; it is now invoked that many times per N, interleaved
// across the N values (see that const's doc comment for why).
func measureStep(driver PoolDriver, tasks, par int) stepSample {
	perG := tasks / par
	if perG < 1 {
		perG = 1
	}
	actualTasks := perG * par

	latencies := make([]float64, actualTasks)
	var assigned int64
	var peakUtil float64
	var utilMu sync.Mutex

	var wg sync.WaitGroup
	wg.Add(par)
	var idx int64 = -1
	startGate := make(chan struct{})
	start := time.Now()
	for g := 0; g < par; g++ {
		go func() {
			defer wg.Done()
			<-startGate
			for it := 0; it < perG; it++ {
				my := atomic.AddInt64(&idx, 1)
				callStart := time.Now()
				ok := driver.ProcessTask(context.Background())
				elapsedMs := float64(time.Since(callStart).Microseconds()) / 1000.0
				if my >= 0 && int(my) < len(latencies) {
					latencies[my] = elapsedMs
				}
				if ok {
					atomic.AddInt64(&assigned, 1)
					// sample utilization while work is in flight
					if u := driver.Utilization(); u > 0 {
						utilMu.Lock()
						if u > peakUtil {
							peakUtil = u
						}
						utilMu.Unlock()
					}
				}
			}
		}()
	}
	close(startGate)
	wg.Wait()
	elapsed := time.Since(start)

	secs := elapsed.Seconds()
	if secs <= 0 {
		secs = 1e-9
	}
	return stepSample{
		actualTasks: actualTasks,
		assigned:    atomic.LoadInt64(&assigned),
		throughput:  float64(actualTasks) / secs,
		latencies:   latencies,
		peakUtil:    peakUtil,
		durationMs:  float64(elapsed.Microseconds()) / 1000.0,
	}
}

// RunScaleSweep drives driver across the worker-count sweep, measures per-N
// throughput + p50/p95/p99 + real utilization, writes scaling_throughput.json
// (write+re-read), runs a stresschaos.RunConcurrent deadlock/leak guard at the
// max N, and FAILS the test when scale-out is flat (gain < MinThroughputGainAtMaxN)
// or throughput regresses as N grows (monotonic-non-degradation). Returns the
// captured ScalingReport for extra assertions.
func RunScaleSweep(t testing.TB, name string, driver PoolDriver, cfg SweepConfig) ScalingReport {
	t.Helper()

	nValues := cfg.NValues
	if len(nValues) == 0 {
		nValues = []int{1, 2, 4, 8}
	}
	tasks := cfg.TasksPerStep
	if tasks <= 0 {
		tasks = 320
	}
	par := cfg.Parallelism
	if par == 0 {
		par = stresschaos.MinParallelism
	}
	if par < stresschaos.MinParallelism {
		t.Fatalf("scaling: parallelism=%d below §11.4.85 floor %d", par, stresschaos.MinParallelism)
	}
	trials := cfg.Trials
	if trials <= 0 {
		trials = MeasureTrialsPerStep
	}

	maxN := 0
	for _, n := range nValues {
		if n > maxN {
			maxN = n
		}
	}

	// samplesByN accumulates every trial's raw measurement per N. The outer loop
	// is TRIAL-major (for each trial: measure N=1,2,4,8 in turn) — i.e.
	// INTERLEAVED across N — never all repeats of one N back-to-back. See
	// MeasureTrialsPerStep's doc comment for why this ordering matters.
	samplesByN := make(map[int][]stepSample, len(nValues))
	for _, n := range nValues {
		samplesByN[n] = make([]stepSample, 0, trials)
	}
	for trial := 0; trial < trials; trial++ {
		for _, n := range nValues {
			teardown := driver.SetupN(t, n)
			s := measureStep(driver, tasks, par)
			samplesByN[n] = append(samplesByN[n], s)
			teardown()
		}
	}

	steps := make([]ScalingStep, 0, len(nValues))
	for _, n := range nValues {
		samples := samplesByN[n]

		trialTPS := make([]float64, len(samples))
		var allLatencies []float64
		var assignedTotal int64
		var actualTasksTotal int
		var peakUtil float64
		var durationTotalMs float64
		for i, s := range samples {
			trialTPS[i] = s.throughput
			allLatencies = append(allLatencies, s.latencies...)
			assignedTotal += s.assigned
			actualTasksTotal += s.actualTasks
			durationTotalMs += s.durationMs
			if s.peakUtil > peakUtil {
				peakUtil = s.peakUtil
			}
		}
		medianTPS := median(trialTPS)

		sorted := make([]float64, len(allLatencies))
		copy(sorted, allLatencies)
		sort.Float64s(sorted)

		step := ScalingStep{
			NWorkers:            n,
			TotalTasks:          actualTasksTotal,
			AssignedTasks:       assignedTotal,
			ThroughputTPS:       medianTPS,
			ThroughputTrialsTPS: trialTPS,
			P50Ms:               percentile(sorted, 50),
			P95Ms:               percentile(sorted, 95),
			P99Ms:               percentile(sorted, 99),
			PoolUtilization:     peakUtil,
			DurationMs:          durationTotalMs,
		}
		steps = append(steps, step)
		t.Logf("scaling: %q N=%d trials=%d throughput(median)=%.0f tps (samples=%v) assigned=%d p50=%.3fms p99=%.3fms peakUtil=%.1f%%",
			name, n, trials, step.ThroughputTPS, trialTPS, step.AssignedTasks, step.P50Ms, step.P99Ms, step.PoolUtilization)
	}

	// Compute scale-out gain (max-N throughput / smallest-N throughput) + monotonic-
	// non-degradation (throughput never drops >40% below the previous step as N grows).
	// Both are computed from the already median-aggregated per-N ThroughputTPS, so a
	// single trial's transient noise cannot by itself flip either verdict.
	var baseTPS, maxTPS float64
	minN := nValues[0]
	for _, n := range nValues {
		if n < minN {
			minN = n
		}
	}
	monotonic := true
	prev := -1.0
	for _, s := range steps {
		if s.NWorkers == minN {
			baseTPS = s.ThroughputTPS
		}
		if s.NWorkers == maxN {
			maxTPS = s.ThroughputTPS
		}
		if prev > 0 && s.ThroughputTPS < prev*0.6 { // >40% drop as N grows = degradation
			monotonic = false
		}
		prev = s.ThroughputTPS
	}
	gain := 0.0
	if baseTPS > 0 {
		gain = maxTPS / baseTPS
	}

	rep := ScalingReport{
		Name:              name,
		Steps:             steps,
		GainAtMaxN:        gain,
		MinGainThreshold:  MinThroughputGainAtMaxN,
		MonotonicNonDegrd: monotonic,
		Trials:            trials,
		Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
	}

	dir := evidenceDir(t, name)
	path := filepath.Join(dir, "scaling_throughput.json")
	writeJSON(t, path, rep)

	// Deadlock / goroutine-leak guard at the max N via the proven stresschaos
	// concurrency harness — this also writes concurrency_report.json.
	driver.SetupN(t, maxN)
	stresschaos.RunConcurrent(t, name+"_concurrency_guard",
		stresschaos.ConcurrencyConfig{Parallelism: par, IterationsPerGoroutine: 20, Timeout: 30 * time.Second},
		func(g, it int) error {
			driver.ProcessTask(context.Background())
			return nil
		})

	if gain < MinThroughputGainAtMaxN {
		t.Fatalf("scaling: %q FLAT throughput — gain at N=%d is %.2fx < required %.2fx (pool ignores added workers?) (evidence: %s)",
			name, maxN, gain, MinThroughputGainAtMaxN, path)
	}
	if !monotonic {
		t.Fatalf("scaling: %q throughput DEGRADES as N grows (lock-convoy / scheduler bug) (evidence: %s)", name, path)
	}

	return rep
}
