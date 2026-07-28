package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/helixqa"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/require"
)

// HXC-174 — runtime race guard for the QA dashboard's session reads.
//
// # THE DEFECT
//
// showQA used to render straight off the LIVE *SessionState pointers returned
// by Engine.ListSessions(), reading Status / Phase / PhaseProgress / EndTime /
// StartTime / Platforms / Banks with no lock held, while the orchestrator
// goroutine spawned by Engine.StartSession writes exactly those fields under
// state.Mu.Lock(). A lock held on ONE side establishes no happens-before edge
// with an unsynchronized access on the other, so this is a genuine data race,
// not merely a race condition.
//
// The sharpest instance was the duration cell:
//
//	if s.EndTime != nil { s.EndTime.Sub(s.StartTime) }
//
// a check-then-use against a pointer published by the writer, so the reader
// could observe a non-nil pointer to a time.Time whose bytes were not yet
// fully published.
//
// Every other consumer of these sessions already went through the lock:
// internal/server/qa_handlers.go serialises them with json.Marshal, which
// SessionState.MarshalJSON guards with s.Mu.RLock. The TUI was the sole
// unguarded reader in the tree.
//
// # POLARITY SWITCH — RED_MODE (§11.4.115, repo convention)
//
//   - RED_MODE unset / "0" (DEFAULT — the standing GREEN regression guard):
//     drives the REAL production showQA() against live sessions while the
//     orchestrator goroutines advance them. Under -race this MUST complete
//     cleanly. On the PRE-FIX artifact this same default run reports a DATA
//     RACE and exits non-zero — that is the §11.4.115 RED baseline, captured
//     on the broken artifact before the fix landed.
//
//   - RED_MODE=1 (harness self-validation — the §11.4.107(10) golden-bad
//     fixture): drives the LEGACY unguarded read pattern showQA used to use,
//     reading the live pointers' fields with no lock. Under -race this MUST
//     report a DATA RACE and exit non-zero. If RED_MODE=1 ever completes
//     cleanly, this harness is blind and its green result means nothing.
//
// Run GREEN guard: go test -tags=ci -race -count=1 -run TestShowQA_SessionReads ./applications/terminal_ui/...
// Run golden-bad: RED_MODE=1 go test -tags=ci -race -count=1 -run TestShowQA_SessionReads ./applications/terminal_ui/...
func TestShowQA_SessionReadsAreRaceFree(t *testing.T) {
	engine, bankFile := newQARaceTestEngine(t)
	tui := newQARaceTestTUI(engine)

	// The drive is ADAPTIVE (the §11.4.3 lesson from HXC-173): a busy host can
	// finish every orchestrator goroutine before the first render lands, which
	// is an INCONCLUSIVE run, not a failing one. Rounds are added until the
	// overlap oracle below is satisfied or the deadline passes; an unmet floor
	// SKIPs with a reason, it never FAILs.
	const (
		overlapFloor  = 1
		hardDeadline  = 30 * time.Second
		rendersPerLap = 40
	)

	deadline := time.Now().Add(hardDeadline)
	start := time.Now()
	overlaps, renders, round := 0, 0, 0

	for overlaps < overlapFloor && time.Now().Before(deadline) {
		round++

		// Fresh sessions each round: the orchestrator goroutine's write burst
		// is what the renders must overlap, and a session that already reached
		// a terminal status writes nothing more.
		for i := 0; i < 4; i++ {
			_, err := engine.StartSession(
				context.Background(),
				fmt.Sprintf("hxc174-r%d-s%d", round, i),
				[]string{"web"},
				[]string{bankFile},
				false,
			)
			require.NoError(t, err, "StartSession must succeed for the drive to exercise anything")
		}

		// OVERLAP ORACLE (§11.4.2 — a clean -race result is worthless if the
		// writer never ran during the read window). The status vector is
		// sampled through the sessions' own RWMutex, which brackets a BATCH of
		// renders rather than each individual one. Bracketing per-render would
		// make the test goroutine acquire state.Mu between every read, and each
		// acquisition creates a happens-before edge with the writer that could
		// MASK the very race under test for writes already retired. Sampling
		// once per batch keeps that edge rare while still proving that writes
		// landed inside the batch.
		before := lockedStatusVector(engine)
		for i := 0; i < rendersPerLap; i++ {
			driveQADashboardRead(tui, engine)
			renders++
		}
		after := lockedStatusVector(engine)

		if statusVectorsDiffer(before, after) {
			overlaps++
		}
	}

	elapsed := time.Since(start)

	if overlaps < overlapFloor {
		t.Skipf("inconclusive: no QA session status transition was observed across %d rendering batches "+
			"(%d renders over %d rounds in %s, deadline %s) — the orchestrator goroutines retired before any "+
			"render overlapped them, so a clean -race result here would prove nothing about the reads under test",
			round, renders, round, elapsed.Round(time.Millisecond), hardDeadline)
	}

	t.Logf("drove %d QA dashboard reads across %d rounds in %s; %d batches bracketed a live orchestrator "+
		"status transition, so the reads under test genuinely overlapped the writer",
		renders, round, elapsed.Round(time.Millisecond), overlaps)

	if redMode() {
		t.Logf("RED_MODE=1 (golden-bad harness): the dashboard fields were read straight off the LIVE " +
			"session pointers with no lock, exactly as showQA did pre-HXC-174. Under -race this MUST report " +
			"a DATA RACE and exit non-zero. A clean completion here means this harness is blind.")
	}
}

// driveQADashboardRead performs one QA-dashboard read pass.
//
// In the DEFAULT (GREEN) polarity it calls the REAL production showQA, so the
// guard covers the shipped code path rather than a re-creation of it. Under
// RED_MODE=1 it instead replays the legacy unguarded pattern showQA used to
// use — the golden-bad fixture proving the race detector really does fire on
// this shape.
func driveQADashboardRead(tui *TerminalUI, engine *helixqa.Engine) {
	if redMode() {
		readSessionsUnguarded(engine)
		return
	}
	tui.showQA()
}

// readSessionsUnguarded is the golden-bad fixture: the exact read shape
// showQA carried before HXC-174 — live pointers off ListSessions, every field
// read with no lock held, including the check-then-use on EndTime.
//
// The reads are accumulated into package-visible sinks so the compiler cannot
// elide them; a race the optimiser deleted would not be detected.
func readSessionsUnguarded(engine *helixqa.Engine) {
	for _, s := range engine.ListSessions() {
		qaRaceStringSink = s.Status
		qaRaceStringSink = s.Phase
		qaRaceFloatSink = s.PhaseProgress
		if !s.StartTime.IsZero() {
			if s.EndTime != nil {
				qaRaceDurationSink = s.EndTime.Sub(s.StartTime)
			} else {
				qaRaceDurationSink = time.Since(s.StartTime)
			}
		}
		qaRaceIntSink = len(s.Platforms) + len(s.Banks)
	}
}

var (
	qaRaceStringSink   string
	qaRaceFloatSink    float64
	qaRaceDurationSink time.Duration
	qaRaceIntSink      int
)

// lockedStatusVector reads every session's status through the session's own
// RWMutex — the lock discipline the rest of the tree observes — so this
// sampling is never itself the race under test.
func lockedStatusVector(engine *helixqa.Engine) map[string]string {
	out := make(map[string]string)
	for _, s := range engine.ListSessions() {
		s.Mu.RLock()
		out[s.ID] = s.Status
		s.Mu.RUnlock()
	}
	return out
}

func statusVectorsDiffer(before, after map[string]string) bool {
	for id, status := range after {
		if prev, ok := before[id]; !ok || prev != status {
			return true
		}
	}
	return false
}

// redMode reports whether the golden-bad polarity is selected. Mirrors the
// convention already used by applications/aurora_os and applications/desktop.
func redMode() bool { return os.Getenv("RED_MODE") == "1" }

// newQARaceTestEngine builds a REAL enabled QA engine over a temp dir, with a
// real bank file, and drains its orchestrator goroutines on cleanup so they
// cannot outlive the test and write into a half-removed TempDir.
func newQARaceTestEngine(t *testing.T) (*helixqa.Engine, string) {
	t.Helper()

	tmpDir := t.TempDir()
	bankFile := filepath.Join(tmpDir, "hxc174-bank.yaml")
	require.NoError(t, os.WriteFile(bankFile, []byte("test: true\n"), 0o644))

	cfg := qaRaceTestConfig(tmpDir)
	engine, err := helixqa.NewEngine(cfg)
	require.NoError(t, err)
	require.True(t, engine.Enabled(), "the QA engine must be enabled or the dashboard renders its disabled hint and reads nothing")
	t.Cleanup(engine.Shutdown)

	return engine, bankFile
}

// newQARaceTestTUI wires the minimum TerminalUI showQA actually touches: the
// engine it lists sessions from, the page container it installs itself into,
// the status bar its button closures write to, and a config for the coverage
// target in the stats panel.
func newQARaceTestTUI(engine *helixqa.Engine) *TerminalUI {
	return &TerminalUI{
		app:       tview.NewApplication(),
		config:    qaRaceTestConfig(""),
		qaEngine:  engine,
		content:   tview.NewPages(),
		pages:     tview.NewPages(),
		statusBar: tview.NewTextView(),
	}
}

func qaRaceTestConfig(outputDir string) *config.Config {
	return &config.Config{
		QA: config.QAConfig{
			Enabled:        true,
			OutputDir:      outputDir,
			Platforms:      []string{"web"},
			BanksDir:       outputDir,
			CoverageTarget: 0.95,
		},
		Logging: config.LoggingConfig{Level: "info"},
	}
}
