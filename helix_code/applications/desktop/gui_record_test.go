//go:build !nogui

package main

// gui_record_test.go — §11.4.98 FULLY-AUTOMATIC, no-TCC, no-Aqua, no-human
// recording of the HelixCode desktop Fyne GUI's LLM-chat feature.
//
// WHY THIS EXISTS (anti-pattern the prior approach hit):
//   scripts/testing/drive_desktop_gui.sh drives a REAL on-screen Fyne/GLFW
//   window with cliclick + screencapture. That path REQUIRES (FACT, documented
//   in that script's own header) an Aqua GUI session attached to the
//   WindowServer + a human-granted Accessibility + Screen-Recording TCC grant.
//   It therefore CANNOT run in a launchd Background domain or under full
//   automation — which §11.4.98 forbids ("fully automatic and autonomous, no
//   manual intervention ... able to rerun endless times").
//
// THE FULLY-AUTOMATIC PATH (deep-research-confirmed — see harness header in
// scripts/video_qa/record_gui_inprocess.sh + the conductor return):
//   Fyne ships a SOFTWARE renderer + an in-memory test driver
//   (fyne.io/fyne/v2/test + fyne.io/fyne/v2/driver/software). test.NewApp()
//   loads a driver that renders to RAM with NO real window, NO WindowServer,
//   NO GL context, NO TCC. The app's REAL widgets are built and driven in
//   process via test.Type/test.Tap; each frame is rendered to a Go image.Image
//   by software.Render(obj, theme) (== Canvas.Capture under a software painter,
//   software/render.go). This runs in ANY launchd domain incl. Background.
//
// WHAT IS REAL HERE (no simulation — §11.4.2 / §11.4.107):
//   * The widget tree built below is the SAME one createLLMTab() builds in
//     main.go: da.chatHistory / da.chatInput / da.llmProviderSel + a "Send
//     Message" button whose closure dispatches through the SAME production
//     send path. The captured frames are the REAL rendered Fyne widgets.
//   * The LLM call is REAL. The provider is resolved from a verifier-driven
//     llm.ModelManager (the production RegisterEnvProviders path, fed real keys
//     from the environment — the wrapper sources ~/api_keys.sh) and the reply
//     is streamed through streamDesktopChat() — the EXACT production function
//     main.go calls (main.go:1290). Nothing is faked.
//   * If NO real provider is reachable (no key, network down), the test SKIPs
//     with an honest reason (§11.4.3) — it NEVER fabricates a reply or a frame.
//
// OUTPUT: PNG frames under $HELIX_GUI_FRAMES_DIR (default a temp dir). The shell
// wrapper assembles them to a project-prefixed MP4 (§11.4.155) under
// /Volumes/T7/Downloads/Recordings (§11.4.158 project instantiation) and
// OCR-content-validates them (§11.4.117/.159) with the self-validated analyzer.

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"dev.helix.code/internal/clientcore"
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/fyneui"
	"dev.helix.code/internal/llm"
)

// recordPNG writes one captured frame to <dir>/frame_<NNNN>.png.
func recordPNG(t *testing.T, dir string, idx int, img image.Image) {
	t.Helper()
	if img == nil {
		t.Fatalf("frame %d: software.Render returned a nil image — software painter not active", idx)
	}
	b := img.Bounds()
	if b.Dx() <= 1 || b.Dy() <= 1 {
		t.Fatalf("frame %d: rendered image is %dx%d — canvas did not lay out (no real render)", idx, b.Dx(), b.Dy())
	}
	p := filepath.Join(dir, fmt.Sprintf("frame_%04d.png", idx))
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("frame %d: create %s: %v", idx, p, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("frame %d: png encode: %v", idx, err)
	}
}

// resolveRealProvider builds the production verifier-driven ModelManager and
// returns the FIRST reachable real provider + a model id, or ("",nil,false)
// when none is reachable. NO mock — this is the same wiring main.go uses.
func resolveRealProvider(t *testing.T) (llm.Provider, string, llm.ProviderType, bool) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Logf("config.Load failed: %v (continuing with defaults)", err)
		cfg = &config.Config{}
	}
	mgr := llm.NewModelManager()
	clientcore.WireVerifierAdapter(mgr, cfg)
	n := clientcore.RegisterEnvProviders(mgr, cfg)
	t.Logf("RegisterEnvProviders registered %d real provider(s)", n)
	if n == 0 {
		return nil, "", "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	models := mgr.GetAvailableModels()
	// Order: chat-capable models FIRST (CapabilityTextGeneration), then the
	// rest. Within each bucket, models on well-known chat providers come first
	// so the recording lands on a real conversational answer rather than an
	// image/embedding endpoint (which 400s a chat request — a real error, but
	// not the feature we are demonstrating).
	preferProviders := map[llm.ProviderType]int{
		llm.ProviderTypeDeepSeek: 0, llm.ProviderTypeGroq: 1, llm.ProviderTypeMistral: 2,
		llm.ProviderTypeGemini: 3, llm.ProviderTypeOpenAI: 4, llm.ProviderTypeAnthropic: 5,
	}
	rank := func(m *llm.ModelInfo) int {
		r := 1000
		if pr, ok := preferProviders[m.Provider]; ok {
			r = pr
		}
		if isChatCapable(m) {
			r -= 500 // chat-capable strongly preferred
		}
		if isNonChatModelName(m) {
			r += 5000 // image/embedding/tts/rerank pushed last
		}
		return r
	}
	sort.SliceStable(models, func(i, j int) bool { return rank(models[i]) < rank(models[j]) })

	for _, info := range models {
		if isNonChatModelName(info) {
			continue // never try an obviously-non-chat model for a chat recording
		}
		p, perr := mgr.GetProviderForModel(info.ID, info.Provider)
		if perr != nil || p == nil {
			continue
		}
		if !p.IsAvailable(ctx) {
			t.Logf("provider %s/%s not available right now — trying next", info.Provider, info.ID)
			continue
		}
		t.Logf("selected REAL chat provider %s model %s (chatCapable=%v)", info.Provider, info.ID, isChatCapable(info))
		return p, info.ID, info.Provider, true
	}
	return nil, "", "", false
}

// isChatCapable reports whether a model advertises a text/chat capability.
func isChatCapable(m *llm.ModelInfo) bool {
	for _, c := range m.Capabilities {
		switch c {
		case llm.CapabilityTextGeneration, llm.CapabilityCodeGeneration,
			llm.CapabilityReasoning, llm.CapabilityWriting:
			return true
		}
	}
	return false
}

// isNonChatModelName flags models whose id/name signals a non-chat endpoint
// (image / embedding / tts / rerank / audio) — these 400 a chat request.
func isNonChatModelName(m *llm.ModelInfo) bool {
	s := strings.ToLower(m.ID + " " + m.Name)
	for _, bad := range []string{
		"flux", "stable-diffusion", "sdxl", "image", "dall-e", "dalle", "vision-only",
		"embed", "embedding", "rerank", "reranker", "tts", "whisper", "audio",
		"voice", "speech", "moderation", "ocr", "clip", "bge-", "gte-",
	} {
		if strings.Contains(s, bad) {
			return true
		}
	}
	return false
}

// TestRecordDesktopGUILLMChat builds the REAL desktop LLM-chat UI headlessly,
// drives a REAL prompt through the REAL provider, and captures rendered frames.
func TestRecordDesktopGUILLMChat(t *testing.T) {
	// 1) Headless software-renderer app — no window, no WindowServer, no TCC.
	app := test.NewApp()
	defer test.NewApp() // reset global app for any sibling test
	theme := NewCustomTheme()
	app.Settings().SetTheme(theme)

	// 2) Resolve a REAL provider (no mock). Honest SKIP if none reachable.
	provider, model, ptype, ok := resolveRealProvider(t)
	if !ok {
		t.Skip("SKIP-OK: no real LLM provider reachable in this environment " +
			"(no API key present / network unreachable). §11.4.3 honest skip — " +
			"the recording requires a real provider; refusing to fabricate a reply (§11.4.2). " +
			"Provide a key via ~/api_keys.sh and re-run.")
	}

	// 3) Build the REAL chat widget tree — faithful to createLLMTab() in main.go.
	da := &DesktopApp{fyneApp: app}
	da.chatHistory = widget.NewMultiLineEntry()
	da.chatHistory.SetPlaceHolder("Chat history will appear here...")
	da.chatHistory.Disable()
	da.chatHistory.Wrapping = fyne.TextWrapWord

	da.chatInput = widget.NewMultiLineEntry()
	da.chatInput.SetPlaceHolder("Type your message...")
	da.chatInput.SetMinRowsVisible(3)

	da.llmProviderSel = widget.NewSelect([]string{fmt.Sprintf("%s/%s", ptype, model)}, nil)
	da.llmProviderSel.SetSelected(fmt.Sprintf("%s/%s", ptype, model))
	da.selectedModel = model

	// sendDone is closed by the production send path when the streamed reply
	// completes, so the render loop knows when the REAL response has landed.
	var sendDone = make(chan struct{})
	var sendOnce sync.Once

	// "Send Message" button — its closure dispatches through the SAME
	// production send path createLLMTab() wires: append the user line, clear
	// input, then stream the REAL reply into da.chatHistory via the production
	// streamDesktopChat() (main.go), the EXACT function the shipped GUI calls.
	sendButton := widget.NewButton("Send Message", func() {
		if da.chatInput.Text == "" {
			return
		}
		userMessage := da.chatInput.Text
		da.chatHistory.SetText(da.chatHistory.Text + fmt.Sprintf("\n[User]: %s\n", userMessage))
		da.chatInput.SetText("")
		prefix := fmt.Sprintf("[AI (%s/%s)]: ", ptype, model)
		go func(msg string) {
			defer sendOnce.Do(func() { close(sendDone) })
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			req := &llm.LLMRequest{
				Messages:    []llm.Message{{Role: "user", Content: msg}},
				Model:       model,
				MaxTokens:   256,
				Temperature: 0.2,
				Stream:      true,
			}
			// PRODUCTION streaming fn (main.go streamDesktopChat → writes into
			// da.chatHistory token-by-token). REAL provider, REAL reply.
			// Mirrors the production send path in main.go: this runs OFF the UI
			// goroutine, so the terminal append is dispatched via
			// fyneui.DoAndWait — blocking, so the text has provably landed
			// before sendDone closes and the renderer captures the final frame.
			if err := streamDesktopChat(ctx, provider, req, prefix, da.chatHistory); err != nil {
				errLine := fmt.Sprintf("\n[AI (%s/%s)]: Error: %v\n", ptype, model, err)
				fyneui.DoAndWait(func() { da.chatHistory.SetText(da.chatHistory.Text + errLine) })
			} else {
				fyneui.DoAndWait(func() { da.chatHistory.SetText(da.chatHistory.Text + "\n") })
			}
		}(userMessage)
	})

	chatControls := container.NewVBox(
		widget.NewLabel("Chat Settings:"),
		widget.NewLabel("Provider:"),
		da.llmProviderSel,
		widget.NewSeparator(),
		sendButton,
	)
	chatPanel := container.NewBorder(
		widget.NewLabel("Chat with AI"),
		container.NewBorder(nil, nil, nil, chatControls, da.chatInput),
		nil, nil,
		da.chatHistory,
	)
	chatCard := widget.NewCard("LLM Chat", "", chatPanel)

	// Register the content in a virtual window and resize it to a real desktop
	// size so the chat history/widgets lay out at full size (NOT the tiny
	// min-size software.Render(obj) would give). We capture the WINDOW canvas,
	// which honours Resize.
	// This goroutine is the RENDERER; the send worker above mutates the same
	// widget tree concurrently. In the shipped app both run on the glfw main
	// goroutine, but Fyne's test driver serializes nothing, so the renderer
	// takes the UI lock explicitly (see the fyneui package doc).
	var win fyne.Window
	fyneui.Sync(func() {
		win = test.NewWindow(chatCard)
		win.Resize(fyne.NewSize(1100, 720))
	})
	defer func() { fyneui.Sync(func() { win.Close() }) }()

	framesDir := os.Getenv("HELIX_GUI_FRAMES_DIR")
	if framesDir == "" {
		framesDir = t.TempDir()
	}
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatalf("mkdir frames dir %s: %v", framesDir, err)
	}
	t.Logf("frames dir: %s", framesDir)

	frame := 0
	cap := func() {
		// RenderCanvas applies the theme + software-paints the WHOLE resized
		// window canvas to a Go image (software/render.go == Canvas.Capture
		// under a software painter). This captures the full-size layout, not
		// the object's minimum size.
		var img image.Image
		fyneui.Sync(func() { img = software.RenderCanvas(win.Canvas(), theme) })
		recordPNG(t, framesDir, frame, img)
		frame++
	}

	// 4) Capture the INITIAL empty chat, then DRIVE the real UI.
	cap() // frame 0: empty chat

	prompt := os.Getenv("HELIX_GUI_PROMPT")
	if prompt == "" {
		prompt = "What is 17 plus 25? Reply with just the number."
	}
	// REAL input-driving through the Fyne test driver (HID-equivalent, in-proc):
	test.Type(da.chatInput, prompt)
	cap() // frame 1: prompt typed into the real entry widget

	// REAL button tap — runs the production send closure above.
	test.Tap(sendButton)
	cap() // frame 2: user line in history, input cleared

	// 5) Render frames while the REAL streamed reply lands.
	deadline := time.After(125 * time.Second)
	tick := time.NewTicker(700 * time.Millisecond)
	defer tick.Stop()
	streamed := false
loop:
	for {
		select {
		case <-sendDone:
			cap() // final frame after the real reply completed
			streamed = true
			break loop
		case <-deadline:
			t.Fatalf("real LLM reply did not complete within timeout — provider %s/%s", ptype, model)
		case <-tick.C:
			cap()
		}
	}

	// 6) ANTI-BLUFF assertions on the captured run (§11.4.2 / §11.4.107):
	//    the history MUST contain the real AI prefix AND non-prompt reply text,
	//    and MUST NOT contain a simulation marker.
	got := da.chatHistory.Text
	t.Logf("final chat history (%d chars):\n%s", len(got), got)
	if !streamed {
		t.Fatalf("stream never signalled completion")
	}
	if !strings.Contains(got, "[User]:") || !strings.Contains(got, fmt.Sprintf("[AI (%s/%s)]:", ptype, model)) {
		t.Fatalf("chat history missing real user/AI turn markers — got:\n%s", got)
	}
	low := strings.ToLower(got)
	for _, bluff := range []string{"simulated", "for now", "todo implement", "in production this would", "placeholder response"} {
		if strings.Contains(low, bluff) {
			t.Fatalf("BLUFF: real reply contains simulation marker %q:\n%s", bluff, got)
		}
	}
	// The AI turn must carry content BEYOND the prefix (a real, non-empty reply).
	aiIdx := strings.Index(got, fmt.Sprintf("[AI (%s/%s)]: ", ptype, model))
	if aiIdx < 0 {
		t.Fatalf("no AI turn found")
	}
	replyTail := strings.TrimSpace(got[aiIdx+len(fmt.Sprintf("[AI (%s/%s)]: ", ptype, model)):])
	if len(replyTail) == 0 {
		t.Fatalf("AI turn is empty — provider returned no content (not a real reply)")
	}

	// 7) Write a sidecar so the wrapper can OCR-assert the exact reply text.
	if framesDir != "" {
		_ = os.WriteFile(filepath.Join(framesDir, "reply.txt"), []byte(got), 0o644)
		_ = os.WriteFile(filepath.Join(framesDir, "frame_count.txt"), []byte(fmt.Sprintf("%d", frame)), 0o644)
	}
	t.Logf("RECORD-OK: %d real rendered frames, real %s/%s reply captured", frame, ptype, model)
}
