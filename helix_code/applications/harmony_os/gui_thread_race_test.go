//go:build !nogui

// Exercises GUI-only symbols declared in the !nogui-tagged sources
// (theme.go / main.go). Under -tags=nogui those symbols do not exist.
//
// BUILD NOTE (§11.4.6, §11.4.201): this package transitively imports
// go-gl/glfw. A PLAIN `go build` / `go test` therefore needs X11/GL
// development headers and fails with `fatal error: X11/Xlib.h: No such file
// or directory` on a host that lacks them. That is a HOST limitation, NOT a
// defect in this package — reporting it as a test failure would be a
// §11.4.201 FAIL-bluff. Fyne's `ci` build tag selects the non-GL driver, so
// this file is built and run with -tags=ci:
//
//	go test -tags=ci -race -count=3 -run TestHarmonyLLMTab_ ./applications/harmony_os/...
package main

// gui_thread_race_test.go — HXC-158.
//
// # WHY THIS FILE EXISTS
//
// Commit 0a4df699 routed every background widget mutation in this file's
// main.go through internal/fyneui (Do / DoAndWait / TickOrStop), mirroring
// the fix e879702c landed in applications/desktop. That commit's own
// HONESTY note recorded the gap this file closes, verbatim:
//
//	"HONESTY (§11.4.6): that run does NOT exercise these paths. No test in
//	 either package constructs the tabs, so the workers never start and the
//	 detector never observes them ... These specific sites are review-
//	 justified against the fyne contract and the desktop precedent, NOT
//	 runtime-proven."
//
// A fix no test can falsify is not a proven fix: with NO test constructing
// the tab, deleting the fyneui wrapping breaks ZERO tests (§11.4.118 —
// absence of a failing test is not absence of the defect). The desktop
// sibling IS runtime-proven precisely because its suite drives widgets from
// worker goroutines under -race. This file gives harmony_os the same teeth.
//
// # WHAT IT ACTUALLY DOES (no simulation — §11.4.2)
//
// It calls the REAL production constructor (*HarmonyApp).createLLMTab(),
// which builds the REAL widget tree AND spawns the REAL provider-health
// goroutine, then renders that tree with Fyne's software painter from a
// background goroutine while the production worker goroutines mutate it.
// Two production mutation sites are covered, both reachable with NO external
// dependency (llmManager == nil is a real, nil-checked production branch —
// not a stub):
//
//   - the provider-health poller  → healthLabel.SetText  (main.go:1633, the
//     eager pre-loop checkHealth() call, which runs on the worker goroutine)
//   - the chat send worker        → chatHistory.SetText  (main.go:1583, the
//     site that carried the FALSE "Update UI on main thread" comment)
//
// # WHY EVERY UI TOUCH IN THIS TEST GOES THROUGH fyneui.Sync
//
// This is load-bearing, and it is the fyneui package's own documented
// contract — not a device to make the test pass. Sync is defined as "the
// entry point for a renderer or for main-goroutine code that must be
// mutually exclusive with concurrent Do / DoAndWait closures."
//
// The two drivers differ, and the difference is FACT read from the vendored
// source (fyne v2.7.0), not assumption:
//
//   - glfw (shipped): DoFromGoroutine → runOnMainWithWait ENQUEUES the
//     closure on funcQueue (internal/driver/glfw/loop.go:52) and the main
//     loop drains it (loop.go:148). A worker's widget write therefore
//     executes ON the main goroutine, and cannot overlap main-goroutine code
//     that has not yet returned to the event loop.
//   - test driver: DoFromGoroutine → async.EnsureNotMain (test/driver.go:55)
//     which, off the main goroutine, invokes the closure INLINE on the
//     CALLING goroutine. Nothing is queued and nothing is serialized; uiMu is
//     the ONLY mutual exclusion.
//
// Consequence for a test that calls a production CONSTRUCTOR: createLLMTab
// spawns the health worker at main.go:1626 and then, still inside the same
// call, runs rightPanel.SetOffset(0.7) at main.go:1665, which lays the tree
// out and READS healthLabel.Text. Under the test driver the worker's write
// really does execute concurrently with that read; under glfw it provably
// cannot. Running the constructor under fyneui.Sync closes that
// DRIVER-SPECIFIC window — the window that does not exist in the shipped
// app — so what remains under test is exactly the production defect class:
// a worker's widget write against a renderer's read.
//
// This does NOT blunt the guard. In RED the worker's write no longer takes
// uiMu at all, so it races the renderer regardless of what the constructor
// holds. Measured, not asserted — see the HXC-158 commit body.
//
// # POLARITY SWITCH — RED_MODE (§11.4.115, repo convention)
//
// A data race has no in-process assertion API: the verdict is the race
// detector's own report plus the non-zero exit code, so the polarity is
// carried by WHICH synchronization is removed, and the evidence is the
// -race output. Both directions are real, and they falsify different halves:
//
//   - RED_MODE unset / "0" (DEFAULT — the standing GREEN regression guard,
//     §11.4.135): the renderer takes the UI lock via fyneui.Sync. On the
//     shipped artifact this is race-free. It goes RED on a PRODUCTION
//     regression — i.e. if anyone reverts a fyneui.Do/DoAndWait site in
//     main.go back to a bare widget mutation. That is the §11.4.115
//     RED-on-the-broken-artifact direction, and the one that guards the fix.
//
//   - RED_MODE=1 (harness self-validation — the §11.4.107(10) golden-bad
//     fixture): the renderer deliberately does NOT take the UI lock. This
//     provokes the SAME defect class on ANY artifact, including the fixed
//     one, and so proves the harness can actually SEE an unsynchronized
//     widget read/write — that a GREEN run means "no race", not "blind
//     test". EXPECTED OUTCOME under -race: a DATA RACE report and a non-zero
//     exit. If RED_MODE=1 ever completes CLEANLY under -race, this harness
//     has lost its teeth and every GREEN result from it is worthless.
//
// Run the GREEN guard (default):
//
//	go test -tags=ci -race -count=3 -run TestHarmonyLLMTab_ ./applications/harmony_os/...
//
// Run the harness self-validation (expect a DATA RACE + non-zero exit):
//
//	RED_MODE=1 go test -tags=ci -race -count=1 -run TestHarmonyLLMTab_ ./applications/harmony_os/...

import (
	"image"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"dev.helix.code/applications/harmony_os/i18n"
	"dev.helix.code/internal/fyneui"
)

// walkHarmonyTree visits o and every descendant reachable through the
// container/widget types this package's tabs are actually built from
// (*fyne.Container, *container.Split, *container.Scroll, *widget.Card). It is
// deliberately narrow: an unhandled container type means a widget is NOT
// found, and the caller fails loudly rather than silently testing nothing.
func walkHarmonyTree(o fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	if o == nil {
		return
	}
	visit(o)
	switch c := o.(type) {
	case *fyne.Container:
		for _, child := range c.Objects {
			walkHarmonyTree(child, visit)
		}
	case *container.Split:
		walkHarmonyTree(c.Leading, visit)
		walkHarmonyTree(c.Trailing, visit)
	case *container.Scroll:
		walkHarmonyTree(c.Content, visit)
	case *widget.Card:
		walkHarmonyTree(c.Content, visit)
	}
}

// findHarmonyButton returns the first *widget.Button in the tree whose label
// equals want, or nil.
func findHarmonyButton(root fyne.CanvasObject, want string) *widget.Button {
	var found *widget.Button
	walkHarmonyTree(root, func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && found == nil && b.Text == want {
			found = b
		}
	})
	return found
}

// newHarmonyRenderLoop starts a background goroutine calling render until the
// returned stop func is called, which is the faithful shape — in the shipped
// app the renderer is the main loop, a different goroutine from the workers.
// The returned stop func reports how many repaints completed.
//
// The counter is written only by the render goroutine and read only after the
// join inside stop, so it needs no synchronization of its own. That is
// deliberate: adding a shared atomic to this hot path could create
// happens-before edges that MASK the very race under test.
//
// This is the loop that previously sat inline in the test below, lifted out
// unchanged so driveForOverlap can run it repeatedly. Same goroutine shape,
// same unsynchronized counter, same read-after-join.
func newHarmonyRenderLoop(render func() image.Image) (stop func() int) {
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	count := 0
	go func() {
		defer close(doneCh)
		for {
			select {
			case <-stopCh:
				return
			default:
			}
			render()
			count++
		}
	}()
	return func() int {
		close(stopCh)
		<-doneCh
		return count
	}
}

// harmonyOverlapFloor is the minimum number of renderer repaints that must
// have overlapped the driven production workers before a clean-under-race
// result is allowed to mean anything. It is an ANTI-VACUITY floor: "no races
// detected" from a renderer that never ran concurrently with a worker is not
// evidence about the code, so a run below the floor proves nothing in EITHER
// direction.
const harmonyOverlapFloor = 2

// harmonyOverlapDeadlineDefault bounds the adaptive drive in driveForOverlap.
//
// Sized against measurement rather than taste: on an unstarved host a single
// round clears the floor several times over, while a round pinned to one
// heavily contended CPU still lands about one repaint. Ten rounds of headroom
// is therefore ample for any host that can render at all, while keeping the
// worst case well inside the 10-minute default `go test` timeout even at
// -count=3.
const harmonyOverlapDeadlineDefault = 30 * time.Second

// overlapDeadline returns the wall-clock budget the adaptive drive has to
// reach harmonyOverlapFloor before the run is reported INCONCLUSIVE.
//
// GUI_RACE_OVERLAP_DEADLINE_MS shortens it. That knob exists so BOTH verdict
// branches are demonstrable on a healthy machine (§11.4.107(10)): reaching the
// SKIP path otherwise requires starving a host so hard that nothing else can
// run on it, and a branch that cannot be exercised is a branch that is not
// proven. It can only make the harness give up SOONER — it can never turn a
// FAIL into a PASS, nor a real data race into a clean run — so it is a
// self-validation knob, not an escape hatch. A value that does not parse, or
// is <= 0, falls back to the default rather than silently disabling the drive
// (§11.4.6: a malformed knob must not quietly change the verdict).
func overlapDeadline() time.Duration {
	raw := os.Getenv("GUI_RACE_OVERLAP_DEADLINE_MS")
	if raw == "" {
		return harmonyOverlapDeadlineDefault
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return harmonyOverlapDeadlineDefault
	}
	return time.Duration(ms) * time.Millisecond
}

// driveForOverlap runs bounded ROUNDS of {start a fresh render loop → run the
// round body → stop the loop and JOIN it → read that loop's counter} until the
// accumulated repaint count reaches harmonyOverlapFloor or deadline elapses.
// It returns the accumulated repaints and the number of rounds run.
//
// # WHY ADAPTIVE AT ALL
//
// Whether a fixed window clears the floor is a property of the HOST, not of
// the code under test: a contended machine can starve the render goroutine
// down to a single repaint. Failing on that turns host load into a verdict
// about production code — a §11.4.201 FAIL-bluff, since the guard fires
// without the condition it names being true. The honest response to "not
// enough overlap yet" is to KEEP DRIVING, and only if the floor is still unmet
// at a hard deadline to report the run INCONCLUSIVE (§11.4.3). A slow host
// must produce a SLOW PASS, never a false failure.
//
// # WHY ROUNDS, AND WHY THIS DOES NOT MASK THE RACE UNDER TEST
//
// newHarmonyRenderLoop's counter is deliberately unsynchronized — written only
// by the render goroutine, read only after the join inside stop — because
// adding a shared atomic to that hot path could create happens-before edges
// that MASK the very race under test (see its doc comment). An adaptive loop
// must therefore never poll that counter while a race window is live, which
// rules out the obvious design of extending one window until enough repaints
// are seen.
//
// This design never polls it. Each round is the SAME race window the
// single-shot version already used — one render loop, its counter untouched
// until its goroutine has fully joined — and rounds merely repeat it. NOT ONE
// synchronizing operation is added INSIDE a round, so the interleavings the
// detector can observe within a round are exactly those it could observe
// before this change.
//
// The join at a round BOUNDARY does order round N's render goroutine ahead of
// round N+1's workers. That is harmless, and it is precisely why accumulation
// is done this way: it orders across DISTINCT race windows, never within one.
// Round N's renderer still races round N's workers with no edge between them.
// What crosses a boundary is only the COUNT — evidence ABOUT the overlap,
// never synchronization THROUGH it.
func driveForOverlap(render func() image.Image, deadline time.Duration, round func(i int)) (renders, rounds int) {
	start := time.Now()
	for {
		stopRender := newHarmonyRenderLoop(render)
		round(rounds)
		// Read ONLY here: stopRender joins the render goroutine first, so this
		// is the same read-after-join the single-shot version performed.
		renders += stopRender()
		rounds++
		if renders >= harmonyOverlapFloor || time.Since(start) >= deadline {
			return renders, rounds
		}
	}
}

// TestHarmonyLLMTab_BackgroundWidgetMutationsAreUIThreadSafe is the HXC-158
// runtime proof for the harmony_os half of the 0a4df699 thread-affinity fix.
//
// It CONSTRUCTS the real LLM tab (which starts the real provider-health
// goroutine), drives the real chat-send worker repeatedly, and renders the
// live widget tree from a background goroutine throughout. Under -race the
// shipped artifact is clean; a reverted fyneui site races. See this file's
// header for the full polarity contract.
func TestHarmonyLLMTab_BackgroundWidgetMutationsAreUIThreadSafe(t *testing.T) {
	fyneApp := test.NewApp()
	defer test.NewApp() // reset the global app for sibling tests

	// Minimal construction, mirroring main_doubleclose_test.go /
	// main_racefix_test.go: the full HarmonyApp is not unit-constructible
	// (NewHarmonyApp calls app.New()).
	//
	// translator is set EXPLICITLY rather than left nil on purpose:
	// (*HarmonyApp).tr lazily ASSIGNS app.translator when it is nil, which is
	// itself a write several goroutines could race. Pre-setting it keeps this
	// test measuring the widget-thread-affinity defect it is named for and
	// nothing else (§11.4.201 — a guard must assert the REAL condition).
	app := &HarmonyApp{
		fyneApp:    fyneApp,
		stopUpdate: make(chan struct{}),
		translator: i18n.NoopTranslator{},
	}
	// llmManager is deliberately nil: that is a real, nil-checked production
	// branch in BOTH covered sites (health poller → "no manager" text; chat
	// worker → "llm not initialized" text). Both still perform the widget
	// mutation this test exists to police, with no network and no provider.

	// Build the REAL tab under the UI lock (see the header: the constructor
	// lays the tree out AFTER spawning the health worker, and only the test
	// driver lets those overlap). This spawns the production provider-health
	// goroutine, whose eager pre-loop checkHealth() mutates healthLabel from
	// that worker.
	var content fyne.CanvasObject
	fyneui.Sync(func() { content = app.createLLMTab() })
	if content == nil {
		t.Fatal("createLLMTab returned nil — cannot exercise the tab")
	}
	// Stop the production goroutine before returning so it does not outlive
	// the test and keep mutating a torn-down tree across -count=N runs.
	defer func() {
		close(app.stopUpdate)
		time.Sleep(50 * time.Millisecond) // let the poller observe the close
	}()

	if app.chatHistory == nil || app.chatInput == nil {
		t.Fatal("createLLMTab did not wire chatHistory/chatInput — the production constructor changed shape")
	}

	sendLabel := app.tr("harmony_os_gui_button_send_message", nil)
	var sendButton *widget.Button
	fyneui.Sync(func() { sendButton = findHarmonyButton(content, sendLabel) })
	if sendButton == nil {
		t.Fatalf("send button %q not found in the constructed tab — the tree shape changed and this test would otherwise silently exercise nothing", sendLabel)
	}

	// Register the tree in a virtual window so the whole tree genuinely lays
	// out and the renderer really reads every label.
	//
	// SIZING (learned the hard way, recorded so it is not re-learned): the
	// canvas MUST NOT be smaller than the content's own MinSize. A test window
	// is not clamped the way a real one is, so an undersized canvas hands the
	// split layout negative child extents and the software rasterizer panics
	// with "slice bounds out of range" inside x/image's vector.Rasterizer.
	// That is a harness sizing error, not a defect in the code under test —
	// floor the window at MinSize and keep it modest above that, because every
	// repaint walks and rasterizes the whole tree and an oversized canvas
	// leaves too few interleavings for the guard to be meaningful.
	var win fyne.Window
	fyneui.Sync(func() {
		size := content.MinSize()
		if size.Width < 900 {
			size.Width = 900
		}
		if size.Height < 700 {
			size.Height = 700
		}
		win = test.NewWindow(content)
		win.Resize(size)
	})
	defer func() { fyneui.Sync(func() { win.Close() }) }()

	// render paints the whole window canvas. In the DEFAULT (GREEN) polarity
	// it holds the UI lock, which is what makes the shipped artifact
	// race-free under the test driver. Under RED_MODE=1 it deliberately does
	// not — the golden-bad fixture (see header).
	render := func() image.Image {
		var img image.Image
		if redMode() {
			img = software.RenderCanvas(win.Canvas(), test.Theme())
		} else {
			fyneui.Sync(func() { img = software.RenderCanvas(win.Canvas(), test.Theme()) })
		}
		return img
	}

	if img := render(); img == nil {
		t.Fatal("software.RenderCanvas returned nil — the software painter is not active, so nothing is reading the widgets")
	} else if b := img.Bounds(); b.Dx() <= 1 || b.Dy() <= 1 {
		t.Fatalf("rendered canvas is %dx%d — the tree did not lay out, so the renderer is not really reading the widgets", b.Dx(), b.Dy())
	}

	// DRIVE (adaptive — see driveForOverlap): fire the production send path
	// repeatedly while a dedicated renderer goroutine repaints, so many worker
	// goroutines mutate chatHistory while it is being read. Each tap runs the
	// REAL production button closure, which spawns the REAL worker goroutine.
	//
	// The tap is UI-goroutine work (it reads chatInput.Text / llmProviderSel
	// .Selected and writes chatHistory), so it takes the UI lock exactly as
	// the fyneui contract requires of main-goroutine code that can overlap a
	// worker's Do.
	// 12 is a deliberate balance: in the GREEN polarity every tap and every
	// repaint contend for the same UI lock, so each tap costs roughly one full
	// repaint of wall-clock. More sends buy no additional coverage — one
	// worker write meeting one concurrent read is already sufficient for the
	// detector — while making a -count=3 run needlessly long.
	//
	// On an unstarved host round 0 alone clears the floor and the loop exits
	// at once, so this IS the original single-shot drive. Extra rounds RE-DRIVE
	// the sends rather than merely repainting for longer: once the workers have
	// finished there is nothing left for a repaint to overlap WITH, and overlap
	// is the only thing the floor measures.
	//
	// Extra rounds send fewer than round 0 on purpose. One worker write meeting
	// one concurrent read already suffices for the detector, while every extra
	// turn lengthens chatHistory and so makes each repaint slower — piling on a
	// further 12 per round would fight the very starvation being compensated
	// for.
	const sends = 12
	const extraSends = 2
	deadline := overlapDeadline()
	driven := 0
	start := time.Now()
	renders, rounds := driveForOverlap(render, deadline, func(i int) {
		n := sends
		if i > 0 {
			n = extraSends
		}
		for k := 0; k < n; k++ {
			fyneui.Sync(func() {
				// The production closure returns early on empty input, so seed it.
				app.chatInput.SetText("hxc158 probe")
				test.Tap(sendButton)
			})
		}
		driven += n

		// Let every in-flight worker land its terminal DoAndWait while the
		// renderer is still repainting, so late writes still meet a live
		// reader. A full software repaint of this tree under -race costs on the
		// order of half a second, so this window spans several of them.
		time.Sleep(2500 * time.Millisecond)
	})
	elapsed := time.Since(start)

	// PROOF THE PATH REALLY RAN (§11.4.2 — a green race result is worthless
	// if the covered code never executed). Read widget state through the same
	// lock discipline the renderer uses, so this read is not itself the race
	// under test.
	//
	// These checks run BEFORE the overlap verdict on purpose. They detect REAL
	// defects — the production send path not running at all — which stay FAIL
	// on every host, however loaded. Only the overlap floor is an
	// inconclusiveness axis, so only it may downgrade to SKIP.
	var history string
	if redMode() {
		history = app.chatHistory.Text
	} else {
		fyneui.Sync(func() { history = app.chatHistory.Text })
	}
	gotTurns := strings.Count(history, "hxc158 probe")
	if gotTurns == 0 {
		t.Fatalf("chat history never received the driven user message — the production send path did not run, so this test proved nothing about it; history=%q", history)
	}
	// Compared against the ACCUMULATED drive, not the round-0 constant: the
	// adaptive loop may have run extra rounds. The equality stays exact however
	// many rounds ran, because the user turn is appended SYNCHRONOUSLY by the
	// tap on the UI goroutine before the worker is spawned — it does not depend
	// on worker timing, so host load cannot make it flaky.
	if gotTurns != driven {
		t.Fatalf("chat history holds %d driven turns, want %d — some sends did not reach the widget, so the covered path is only partly exercised", gotTurns, driven)
	}
	// The worker half must have landed too: the nil-llmManager branch appends
	// its own response line, so its absence means only the UI-goroutine half
	// of the turn ran and no worker ever mutated the widget.
	wantWorkerMark := app.tr("harmony_os_gui_chat_llm_not_initialized_fmt", map[string]any{
		"Provider": app.llmProviderSel.Selected, "Model": "llama2",
	})
	if wantWorkerMark != "" && !strings.Contains(history, wantWorkerMark) {
		t.Fatalf("chat history holds no worker-produced response line (%q) — the background goroutine never mutated the widget, so the thread-affinity site under test was never exercised; history=%q",
			wantWorkerMark, history)
	}

	t.Logf("drove %d production chat-send workers across %d adaptive round(s) against %d concurrent software renders of the real LLM tab in %s; chat history holds %d turns (%d chars)",
		driven, rounds, renders, elapsed.Round(time.Millisecond), gotTurns, len(history))
	if redMode() {
		t.Logf("RED_MODE=1 (golden-bad harness): the renderer and the post-drive reads ran WITHOUT the UI lock. " +
			"Under -race this MUST report a DATA RACE and exit non-zero. A clean completion here means the harness is blind.")
	}

	// VERDICT ON THE RACE WINDOW ITSELF (§11.4.3 / §11.4.201). Everything above
	// held, so the production path demonstrably ran and was verified. What may
	// still be missing is enough renderer/worker OVERLAP for a clean -race
	// result to carry meaning. That is a property of the HOST, not of the code
	// under test, so it is reported INCONCLUSIVE — never as a failure of this
	// package. The floor itself is NOT relaxed: it still has to be met, the
	// drive just keeps going until it is or the deadline runs out.
	if renders < harmonyOverlapFloor {
		t.Skipf("INCONCLUSIVE — not a failure: the renderer completed only %d repaints (floor %d) across %d adaptive round(s) in %s, exhausting the %s overlap deadline. The host was too contended for this guard to observe meaningful renderer/worker overlap, so a clean -race result here would prove nothing. The production send path itself ran and was verified above; only the race window is unproven. Re-run on a less contended host.",
			renders, harmonyOverlapFloor, rounds, elapsed.Round(time.Millisecond), deadline)
	}
}
