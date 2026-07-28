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

	// RENDERER: a dedicated goroutine repainting continuously, which is the
	// faithful shape — in the shipped app the renderer is the main loop, a
	// different goroutine from the workers. renderCount is written only here
	// and read only after the join below, so it needs no synchronization of
	// its own (adding a shared atomic in this hot path could create
	// happens-before edges that MASK the very race under test).
	stopRender := make(chan struct{})
	renderDone := make(chan struct{})
	renderCount := 0
	go func() {
		defer close(renderDone)
		for {
			select {
			case <-stopRender:
				return
			default:
			}
			render()
			renderCount++
		}
	}()

	// DRIVE: fire the production send path repeatedly while the renderer
	// repaints, so many worker goroutines mutate chatHistory while it is
	// being read. Each tap runs the REAL production button closure, which
	// spawns the REAL worker goroutine.
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
	const sends = 12
	start := time.Now()
	for i := 0; i < sends; i++ {
		fyneui.Sync(func() {
			// The production closure returns early on empty input, so seed it.
			app.chatInput.SetText("hxc158 probe")
			test.Tap(sendButton)
		})
	}

	// Let every in-flight worker land its terminal DoAndWait while the
	// renderer is still repainting, so late writes still meet a live reader.
	// A full software repaint of this tree under -race costs on the order of
	// half a second, so this window is sized to span several of them.
	time.Sleep(2500 * time.Millisecond)
	close(stopRender)
	<-renderDone
	elapsed := time.Since(start)

	if renderCount < 2 {
		t.Fatalf("renderer completed only %d repaints during the drive — too little renderer/worker overlap for this guard to be meaningful", renderCount)
	}

	// PROOF THE PATH REALLY RAN (§11.4.2 — a green race result is worthless
	// if the covered code never executed). Read widget state through the same
	// lock discipline the renderer uses, so this read is not itself the race
	// under test.
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
	if gotTurns != sends {
		t.Fatalf("chat history holds %d driven turns, want %d — some sends did not reach the widget, so the covered path is only partly exercised", gotTurns, sends)
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

	t.Logf("drove %d production chat-send workers against %d concurrent software renders of the real LLM tab in %s; chat history holds %d turns (%d chars)",
		sends, renderCount, elapsed.Round(time.Millisecond), gotTurns, len(history))
	if redMode() {
		t.Logf("RED_MODE=1 (golden-bad harness): the renderer and the post-drive reads ran WITHOUT the UI lock. " +
			"Under -race this MUST report a DATA RACE and exit non-zero. A clean completion here means the harness is blind.")
	}
}
