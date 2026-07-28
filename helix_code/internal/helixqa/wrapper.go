// Package helixqa provides an embedded wrapper for the helix_qa testing framework.
package helixqa

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/config"

	hqaConfig "digital.vasic.helixqa/pkg/config"
	hqaEvidence "digital.vasic.helixqa/pkg/evidence"
	hqaOrchestrator "digital.vasic.helixqa/pkg/orchestrator"
	"digital.vasic.helixqa/pkg/reporter"
	hqaScreenshot "digital.vasic.helixqa/pkg/screenshot"
)

// SessionState tracks a single QA session within the HelixCode server.
type SessionState struct {
	ID               string                  `json:"id"`
	Status           string                  `json:"status"` // pending|running|completed|failed|cancelled
	Phase            string                  `json:"phase"`
	PhaseProgress    float64                 `json:"phase_progress"`
	Platforms        []string                `json:"platforms"`
	Banks            []string                `json:"banks"`
	StartTime        time.Time               `json:"start_time"`
	EndTime          *time.Time              `json:"end_time,omitempty"`
	Result           *hqaOrchestrator.Result `json:"result,omitempty"`
	AutonomousResult interface{}             `json:"autonomous_result,omitempty"`
	ReportPath       string                  `json:"report_path,omitempty"`
	CancelFunc       context.CancelFunc      `json:"-"`
	Mu               sync.RWMutex            `json:"-"`
}

// MarshalJSON serialises the session state under its own RLock so that the
// HTTP handlers (which call `json.Marshal(state)`) cannot race with the
// orchestrator goroutine spawned by StartSession that mutates Status /
// Phase / PhaseProgress / EndTime / Result / ReportPath as the session
// progresses. Without this, every QA-handler test that returned a
// non-terminal session tripped the race detector — encoding/json reads the
// raw fields via reflection and has no awareness of state.Mu.
//
// The shadow type below is the standard "alias-without-MarshalJSON"
// pattern: it has the same JSON tags but no method receiver, so calling
// json.Marshal on a sessionStateJSON value does not recurse into this
// MarshalJSON.
func (s *SessionState) MarshalJSON() ([]byte, error) {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	type sessionStateJSON SessionState
	return json.Marshal((*sessionStateJSON)(s))
}

// creationSnapshot returns a DETACHED copy of the session's field values as of
// the moment of the call. The copy carries its own zero mutex and a nil
// CancelFunc, so nothing the orchestrator goroutine subsequently writes to the
// live session is visible through it, and nothing written to it can reach the
// live session.
//
// This exists because StartSession used to hand its caller the LIVE session it
// had just spawned a goroutine to mutate (HXC-154): the goroutine's first act
// is `state.Mu.Lock(); state.Status = "running"`, so the value a caller read
// back depended purely on which goroutine the scheduler ran first. That is a
// race CONDITION, not a data race — MarshalJSON and the orchestrator both go
// through state.Mu, so `-race` reports nothing — but it made the HTTP 201
// Created body non-deterministic: measured on the pre-fix artifact, 2.6% of
// responses reported "running" instead of "pending", and a few reported
// "completed", i.e. a 201 Created announcing an already-finished resource.
//
// Slices are copied rather than aliased so a future mutation of the live
// session's Platforms/Banks cannot reach a snapshot already handed out.
func (s *SessionState) creationSnapshot() *SessionState {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	cp := &SessionState{
		ID:               s.ID,
		Status:           s.Status,
		Phase:            s.Phase,
		PhaseProgress:    s.PhaseProgress,
		Platforms:        append([]string(nil), s.Platforms...),
		Banks:            append([]string(nil), s.Banks...),
		StartTime:        s.StartTime,
		Result:           s.Result,
		AutonomousResult: s.AutonomousResult,
		ReportPath:       s.ReportPath,
	}
	if s.EndTime != nil {
		end := *s.EndTime
		cp.EndTime = &end
	}
	return cp
}

// Engine is the singleton QA engine embedded in the HelixCode server.
type Engine struct {
	sessions    map[string]*SessionState
	sessionMu   sync.RWMutex
	cfg         *config.Config
	qaCfg       *hqaConfig.Config
	evidenceDir string
	enabled     bool
	// activeWG tracks every session goroutine spawned by StartSession so that
	// Shutdown can wait for them to drain. Without this, tests using a
	// t.TempDir-backed OutputDir race with the orchestrator goroutine: the
	// test returns, Go's testing framework removes the temp dir, and the
	// still-running orchestrator writes a new file into a half-deleted dir,
	// producing "unlinkat ... directory not empty" cleanup errors.
	activeWG sync.WaitGroup
}

// NewEngine builds the embedded QA engine from HelixCode configuration.
func NewEngine(cfg *config.Config) (*Engine, error) {
	if !cfg.QA.Enabled {
		return &Engine{enabled: false}, nil
	}
	qaCfg, err := buildQAConfig(cfg)
	if err != nil {
		// CONST-046: user-facing error literal resolved via tr().
		// NewEngine has no caller-supplied context; Background is
		// the canonical fallback per rounds 146..158.
		msg := tr(context.Background(), "internal_helixqa_config_build_failed", map[string]any{"Err": err.Error()})
		return nil, errors.New(msg)
	}
	return &Engine{
		sessions:    make(map[string]*SessionState),
		cfg:         cfg,
		qaCfg:       qaCfg,
		evidenceDir: cfg.QA.OutputDir,
		enabled:     true,
	}, nil
}

// Enabled returns true if the QA engine is configured and active.
func (e *Engine) Enabled() bool {
	return e.enabled
}

// StartSession begins a new QA session.
//
// The returned *SessionState is a DETACHED snapshot of the session AS CREATED
// (Status "pending", no EndTime, no CancelFunc) — NOT a live handle. It never
// changes, so callers that render it (notably the HTTP 201 Created body) get a
// deterministic answer instead of whatever the orchestrator goroutine happened
// to have written by the time they looked (HXC-154).
//
// To observe a session's progress, or to cancel it, go through the engine:
// GetSession(id) returns the live *SessionState (read its mutable fields under
// state.Mu, or marshal it — MarshalJSON locks for you), and CancelSession(id)
// cancels. Writing to the value returned here affects nothing.
func (e *Engine) StartSession(ctx context.Context, id string, platforms, banks []string, autonomous bool) (*SessionState, error) {
	if !e.enabled {
		// CONST-046: tr() resolves the literal via the package-level
		// Translator (NoopTranslator echoes the message ID).
		return nil, errors.New(tr(ctx, "internal_helixqa_qa_disabled", nil))
	}
	if id == "" {
		return nil, errors.New(tr(ctx, "internal_helixqa_session_id_required", nil))
	}

	// Validate banks exist
	for _, bank := range banks {
		if _, err := os.Stat(bank); err != nil {
			return nil, errors.New(tr(ctx, "internal_helixqa_bank_not_found", map[string]any{"Bank": bank}))
		}
	}

	sessionCtx, cancel := context.WithCancel(ctx)
	state := &SessionState{
		ID:         id,
		Status:     "pending",
		Platforms:  platforms,
		Banks:      banks,
		StartTime:  time.Now(),
		CancelFunc: cancel,
	}

	e.sessionMu.Lock()
	e.sessions[id] = state
	e.sessionMu.Unlock()

	// Capture the creation snapshot BEFORE the orchestrator goroutine exists,
	// so what we hand back is a stable record of the session AS CREATED rather
	// than a live view the goroutine races us to change (HXC-154). Callers that
	// need to observe the session's progress use GetSession(id) — see the
	// doc comment on StartSession.
	created := state.creationSnapshot()

	e.activeWG.Add(1)
	go func() {
		defer e.activeWG.Done()
		defer cancel()
		state.Mu.Lock()
		state.Status = "running"
		state.Phase = "orchestration"
		state.PhaseProgress = 0.0
		state.Mu.Unlock()

		// Per-session config. Earlier revisions did `cfg := e.qaCfg` —
		// which aliased the Engine's shared *hqaConfig.Config pointer —
		// and then mutated `cfg.Banks` / `cfg.Platforms`. With more than
		// one concurrent session, both goroutines clobbered the same
		// shared struct, and the race detector caught it (writes at
		// wrapper.go:119,130 from two goroutines to identical addresses).
		// Worse than the test failure, the production effect was that
		// concurrent sessions read each other's banks/platforms.
		// Resolution: shallow-copy the engine's config so each session
		// owns its own Banks/Platforms fields. Slices are reference
		// types but assigning new slices to the copy does not touch
		// the engine's shared instance.
		sessionCfg := *e.qaCfg
		sessionCfg.Banks = banks
		parsedPlatforms, err := hqaConfig.ParsePlatforms(strings.Join(platforms, ","))
		if err != nil {
			state.Mu.Lock()
			state.Status = "failed"
			state.Phase = "error"
			now := time.Now()
			state.EndTime = &now
			state.Mu.Unlock()
			return
		}
		sessionCfg.Platforms = parsedPlatforms

		orc := hqaOrchestrator.New(&sessionCfg)
		res, err := orc.Run(sessionCtx)

		state.Mu.Lock()
		defer state.Mu.Unlock()
		if err != nil && sessionCtx.Err() != context.Canceled {
			state.Status = "failed"
		} else if sessionCtx.Err() == context.Canceled {
			state.Status = "cancelled"
		} else {
			state.Status = "completed"
			state.Result = res
			if res != nil {
				state.ReportPath = res.ReportPath
			}
		}
		now := time.Now()
		state.EndTime = &now
		state.PhaseProgress = 1.0
	}()

	return created, nil
}

// GetSession retrieves a session by ID.
func (e *Engine) GetSession(id string) (*SessionState, bool) {
	e.sessionMu.RLock()
	defer e.sessionMu.RUnlock()
	s, ok := e.sessions[id]
	return s, ok
}

// Shutdown cancels every active session and blocks until all background
// goroutines spawned by StartSession have returned. Intended for graceful
// server shutdown and for test cleanup (t.Cleanup) so that the temp-dir
// teardown cannot race with the orchestrator goroutine still writing
// evidence files into OutputDir.
func (e *Engine) Shutdown() {
	e.sessionMu.Lock()
	for _, s := range e.sessions {
		s.Mu.Lock()
		if s.CancelFunc != nil {
			s.CancelFunc()
		}
		s.Mu.Unlock()
	}
	e.sessionMu.Unlock()
	e.activeWG.Wait()
}

// CancelSession signals cancellation for a running session.
func (e *Engine) CancelSession(id string) error {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	s, ok := e.sessions[id]
	if !ok {
		// CONST-046: CancelSession has no caller-supplied context;
		// Background is the canonical fallback per rounds 146..158.
		return errors.New(tr(context.Background(), "internal_helixqa_session_not_found", map[string]any{"ID": id}))
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	// A session that has already reached a terminal state
	// (completed/failed/cancelled) MUST NOT be relabelled "cancelled".
	// CancelFunc is set once at StartSession and never cleared, so
	// gating on `CancelFunc != nil` alone would clobber the truthful
	// terminal status of an already-finished run (e.g. a stale "stop"
	// click on a completed session) — a §11.4 PASS-bluff: a session
	// that genuinely completed with a real Result/ReportPath would be
	// silently reported as "cancelled". Only non-terminal sessions are
	// transitioned. The cancel func is still invoked to release the
	// context resources regardless (idempotent no-op once done).
	if s.CancelFunc != nil {
		s.CancelFunc()
	}
	// Only transition a NON-terminal session — never clobber the truthful
	// record of a session that already finished (isTerminalStatus is the single
	// source of truth so a future terminal status can't silently fall through).
	if !isTerminalStatus(s.Status) {
		s.Status = "cancelled"
		now := time.Now()
		s.EndTime = &now
	}
	return nil
}

// isTerminalStatus reports whether a QA session status marks the run as
// finished. It is the single source of truth for terminal-state checks; update
// this set whenever StartSession's goroutine introduces a new terminal status.
func isTerminalStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}

// ListSessions returns all session states as LIVE pointers.
//
// The returned sessions are the very objects the orchestrator goroutine
// spawned by StartSession keeps mutating under state.Mu. A caller that reads
// their fields MUST therefore either hold s.Mu itself or go through a
// lock-aware path such as MarshalJSON — reading s.Status / s.Phase /
// s.PhaseProgress / s.EndTime directly off these pointers is a data race, and
// a lock held only by the writer establishes no happens-before edge that would
// make it safe.
//
// Callers that just want to RENDER a session (a dashboard, a report, a log
// line) should prefer ListSessionSnapshots, which hands back detached copies
// and cannot be misused this way. This function remains for callers that need
// session IDENTITY — e.g. to pair a listing with a later CancelSession call.
func (e *Engine) ListSessions() []*SessionState {
	e.sessionMu.RLock()
	defer e.sessionMu.RUnlock()
	out := make([]*SessionState, 0, len(e.sessions))
	for _, s := range e.sessions {
		out = append(out, s)
	}
	return out
}

// ListSessionSnapshots returns a DETACHED copy of every session state as of
// the moment of the call, each taken under that session's own read lock.
//
// This is the read-only counterpart to ListSessions and exists because the
// terminal UI's QA dashboard used to render straight off the live pointers,
// reading Status / Phase / PhaseProgress / EndTime / StartTime / Platforms /
// Banks with no lock while the orchestrator goroutine wrote exactly those
// fields under state.Mu (HXC-174). Its sharpest instance was the duration
// cell's `if s.EndTime != nil { s.EndTime.Sub(s.StartTime) }` — a
// check-then-use against a pointer the writer publishes, so the reader could
// observe a non-nil pointer to a not-yet-published value. The race detector
// reported it on the pre-fix artifact at seven distinct read sites.
//
// Each element is produced by creationSnapshot, which is the established
// point-in-time detach in this package (added for HXC-154): it copies under
// s.Mu.RLock, carries its own zero mutex and a nil CancelFunc, and copies the
// Platforms/Banks slices rather than aliasing them, so a later mutation of the
// live session cannot reach a snapshot already handed out.
//
// Snapshots are consistent PER SESSION, not across the set — sessions are
// snapshotted one after another, so two sessions in one result may reflect
// slightly different instants. That is the correct granularity for rendering
// a list, and no caller needs a globally atomic view.
//
// Ordering follows ListSessions (Go map iteration order, i.e. unspecified).
func (e *Engine) ListSessionSnapshots() []*SessionState {
	live := e.ListSessions()
	out := make([]*SessionState, 0, len(live))
	for _, s := range live {
		out = append(out, s.creationSnapshot())
	}
	return out
}

// EvidenceCollector returns the evidence collector for on-demand screenshots.
func (e *Engine) EvidenceCollector(platform hqaConfig.Platform) *hqaEvidence.Collector {
	return hqaEvidence.New(
		hqaEvidence.WithOutputDir(e.evidenceDir),
		hqaEvidence.WithPlatform(platform),
	)
}

// GenerateReport creates a report for a completed session in the requested format.
func (e *Engine) GenerateReport(state *SessionState, format string) ([]byte, string, error) {
	if state == nil || state.Result == nil || state.Result.Report == nil {
		// CONST-046: GenerateReport has no caller-supplied context;
		// Background is the canonical fallback per rounds 146..158.
		return nil, "", errors.New(tr(context.Background(), "internal_helixqa_no_report_available", nil))
	}
	rpt := reporter.New(
		reporter.WithOutputDir(e.evidenceDir),
		reporter.WithReportFormat(hqaConfig.ReportMarkdown),
	)

	switch format {
	case "html":
		rpt = reporter.New(
			reporter.WithOutputDir(e.evidenceDir),
			reporter.WithReportFormat(hqaConfig.ReportHTML),
		)
	case "json":
		rpt = reporter.New(
			reporter.WithOutputDir(e.evidenceDir),
			reporter.WithReportFormat(hqaConfig.ReportJSON),
		)
	}

	path, err := rpt.WriteReport(state.Result.Report, e.evidenceDir)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	return data, path, err
}

// CaptureScreenshot captures a standalone screenshot for the given platform.
func (e *Engine) CaptureScreenshot(ctx context.Context, platform string, opts hqaScreenshot.CaptureOptions) (*hqaScreenshot.Result, error) {
	if !e.enabled {
		// CONST-046: tr() resolves the literal via the package-level
		// Translator (NoopTranslator echoes the message ID).
		return nil, errors.New(tr(ctx, "internal_helixqa_qa_disabled", nil))
	}
	mgr := hqaScreenshot.NewManager(nil)
	// Register all available engines
	mgr.RegisterEngine(hqaConfig.PlatformWeb, hqaScreenshot.NewWebEngine(""))
	mgr.RegisterEngine(hqaConfig.PlatformLinux, hqaScreenshot.NewLinuxEngine())
	mgr.RegisterEngine(hqaConfig.PlatformIOS, hqaScreenshot.NewIOSEngine(""))
	mgr.RegisterEngine(hqaConfig.PlatformAndroid, hqaScreenshot.NewAndroidEngine(""))
	mgr.RegisterEngine(hqaConfig.PlatformDesktop, hqaScreenshot.NewLinuxEngine())
	return mgr.Capture(ctx, hqaConfig.Platform(platform), opts)
}

// ListScreenshotEngines returns the names of supported screenshot engines.
func (e *Engine) ListScreenshotEngines(ctx context.Context) []string {
	if !e.enabled {
		return nil
	}
	mgr := hqaScreenshot.NewManager(nil)
	mgr.RegisterEngine(hqaConfig.PlatformWeb, hqaScreenshot.NewWebEngine(""))
	mgr.RegisterEngine(hqaConfig.PlatformLinux, hqaScreenshot.NewLinuxEngine())
	mgr.RegisterEngine(hqaConfig.PlatformIOS, hqaScreenshot.NewIOSEngine(""))
	mgr.RegisterEngine(hqaConfig.PlatformAndroid, hqaScreenshot.NewAndroidEngine(""))
	mgr.RegisterEngine(hqaConfig.PlatformDesktop, hqaScreenshot.NewLinuxEngine())
	var names []string
	for _, plat := range mgr.SupportedPlatforms(ctx) {
		names = append(names, string(plat))
	}
	return names
}

func buildQAConfig(cfg *config.Config) (*hqaConfig.Config, error) {
	qc := cfg.QA
	platforms, err := hqaConfig.ParsePlatforms(strings.Join(qc.Platforms, ","))
	if err != nil {
		// CONST-046: buildQAConfig has no caller-supplied context;
		// Background is the canonical fallback per rounds 146..158.
		msg := tr(context.Background(), "internal_helixqa_parse_platforms_failed", map[string]any{"Err": err.Error()})
		return nil, errors.New(msg)
	}
	banks := hqaConfig.ParseBanks(qc.BanksDir)
	if len(banks) == 0 && qc.BanksDir != "" {
		banks = []string{qc.BanksDir}
	}

	reportFormat := hqaConfig.ReportMarkdown
	if len(qc.ReportFormats) > 0 {
		switch qc.ReportFormats[0] {
		case "html":
			reportFormat = hqaConfig.ReportHTML
		case "json":
			reportFormat = hqaConfig.ReportJSON
		}
	}

	return &hqaConfig.Config{
		Banks:         banks,
		Platforms:     platforms,
		Device:        qc.DeviceID,
		OutputDir:     qc.OutputDir,
		ReportFormat:  reportFormat,
		ValidateSteps: true,
		Record:        qc.RecordVideo,
		Verbose:       cfg.Logging.Level == "debug",
		Timeout:       2 * time.Hour,
		StepTimeout:   5 * time.Minute,
		Autonomous: hqaConfig.AutonomousConfig{
			Enabled:              qc.Autonomous,
			CoverageTarget:       qc.CoverageTarget,
			CuriosityEnabled:     qc.CuriosityEnabled,
			CuriosityTimeout:     30 * time.Minute,
			VisionProvider:       qc.VisionProvider,
			LLMProvider:          qc.LLMProvider,
			LLMAPIKey:            qc.LLMAPIKey,
			RecordingScreenshots: qc.RecordScreenshots,
			RecordingVideo:       qc.RecordVideo,
		},
	}, nil
}
