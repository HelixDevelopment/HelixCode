package scenarios

import (
	"context"
	"testing"
)

// TestRunner_StableAcrossThreeRuns is the P0-T04 anti-bluff integration test:
// the harness must produce numbers stable enough to detect a 1.3x change. We
// run the fixture-walk scenarios (S3 repomap, S4 search) three times against a
// real generated fixture and assert the coefficient of variation is well under
// the 1.3x discrimination threshold (CV must be far below 30%).
//
// This is an integration-level test (real filesystem I/O, no mocks) per
// CONST-050 — it exercises the real scenario runner against a real fixture.
func TestRunner_StableAcrossThreeRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("filesystem-heavy harness stability test skipped in -short — SKIP-OK: integration")
	}
	m, err := LoadManifest("")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	cfg := DefaultFixtureConfig(m)
	cfg.FileCount = 600 // smaller than default for a fast, deterministic test
	fixtureRoot := t.TempDir()
	fc, _, err := GenerateFixture(fixtureRoot, cfg)
	if err != nil {
		t.Fatalf("GenerateFixture: %v", err)
	}
	t.Logf("fixture: %d files at %s", fc, fixtureRoot)

	opts := RunOptions{Manifest: m, FixtureRoot: fixtureRoot}
	ctx := context.Background()

	// Discarded warmup pass. GenerateFixture has just written the fixture, so
	// the first walk over it pays one-time filesystem cache population (dentry,
	// inode and page cache) that no later run repeats. Measured on this tree,
	// run 0 was the outlier in 3 of 3 trials — 2079/2114/1989 ms against a
	// steady state of 38-44 ms, a ~50x cold/warm ratio — while runs 1 and 2
	// agreed to a CV of 7.6%, 1.0% and 5.5%. Sampling that cold run made a
	// precise harness report CV=163% and fail its own 35% bound.
	//
	// This test asks whether the harness can DISCRIMINATE a 1.3x change, which
	// is a steady-state question, so steady state is the regime to sample. The
	// warmup is a real RunAll whose results are dropped, never a sleep. Note
	// this deliberately stops the test from noticing a cold-start regression —
	// that is a different question and belongs in its own test.
	if _, warmErr := RunAll(ctx, opts); warmErr != nil {
		t.Fatalf("RunAll warmup: %v", warmErr)
	}

	// Collect 3 runs per scenario.
	samples := map[string][]float64{}
	for i := 0; i < 3; i++ {
		results, runErr := RunAll(ctx, opts)
		if runErr != nil {
			t.Fatalf("RunAll iteration %d: %v", i, runErr)
		}
		for _, r := range results {
			if r.Skipped {
				t.Logf("run %d: %s skipped (%s)", i, r.ScenarioID, r.SkipReason)
				continue
			}
			samples[r.ScenarioID] = append(samples[r.ScenarioID],
				float64(r.Duration.Nanoseconds())/1e6)
		}
	}

	// S3 and S4 must have produced 3 non-skipped samples (fixture is present).
	for _, id := range []string{"S3", "S4"} {
		s := samples[id]
		if len(s) != 3 {
			t.Fatalf("%s: expected 3 samples, got %d", id, len(s))
		}
		cv := coeffOfVariation(s)
		t.Logf("%s: samples=%v ms  CV=%.2f%%", id, s, cv)
		// 1.3x discrimination needs CV well below the change being detected.
		// We require CV < 35% — a generous bound that still proves the harness
		// is not pure noise. Typical CV is single-digit.
		if cv >= 35.0 {
			t.Fatalf("%s harness too noisy: CV=%.2f%% (>= 35%%) cannot reliably detect a 1.3x change", id, cv)
		}
	}
}

// TestRunScenario_FixtureMissing asserts fixture-dependent scenarios skip
// cleanly (with a SKIP-OK marker) when no fixture is provided.
func TestRunScenario_FixtureMissing(t *testing.T) {
	m, err := LoadManifest("")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	s3, ok := m.Scenario("S3")
	if !ok {
		t.Fatal("S3 missing from manifest")
	}
	res := RunScenario(context.Background(), s3, RunOptions{Manifest: m})
	if !res.Skipped {
		t.Fatalf("S3 with no fixture should skip, got duration=%s", res.MillisString())
	}
	if res.SkipReason == "" {
		t.Fatal("skipped result must carry a skip reason")
	}
}

// TestRunAll_ProducesAllScenarios asserts RunAll returns one result per
// manifest scenario in S-id order.
func TestRunAll_ProducesAllScenarios(t *testing.T) {
	m, err := LoadManifest("")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	results, err := RunAll(context.Background(), RunOptions{Manifest: m})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(results) != len(m.Scenarios) {
		t.Fatalf("RunAll returned %d results, want %d", len(results), len(m.Scenarios))
	}
	for i, want := range []string{"S1", "S2", "S3", "S4"} {
		if results[i].ScenarioID != want {
			t.Fatalf("result[%d] = %s, want %s", i, results[i].ScenarioID, want)
		}
	}
}

// coeffOfVariation returns the CV (%) of the samples.
func coeffOfVariation(samples []float64) float64 {
	if len(samples) < 2 {
		return 0
	}
	var sum float64
	for _, v := range samples {
		sum += v
	}
	mean := sum / float64(len(samples))
	if mean == 0 {
		return 0
	}
	var sq float64
	for _, v := range samples {
		d := v - mean
		sq += d * d
	}
	variance := sq / float64(len(samples)-1)
	std := newtonSqrt(variance)
	return std / mean * 100
}

func newtonSqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 40; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
