package main

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"dev.helix.code/internal/llm"
)

// HXC-162 — skill auto-trigger must actually FIRE in the interactive CLI.
//
// # RUNTIME SIGNATURE (§11.4.108)
//
// The machine-checkable observable that proves this fix is BOTH active AND
// working — and which is NEVER a re-grep of the source — is:
//
//	a user utterance typed on real stdin into the real REPL (handleInteractive)
//	arrives at the LLM provider as the matched skill's RENDERED BODY, not as the
//	raw text the user typed.
//
// That signature cannot be satisfied by a SkillDispatcher merely existing. It
// requires the dispatcher to be reachable from the input path, to be consulted
// for every plain-language line, to match, to render, and for the rendered
// output to replace the outgoing prompt. Only a genuinely wired dispatcher
// produces it — which is exactly what HXC-162 says is missing (the dispatcher
// was constructed at cmd/cli/main.go:1060 and immediately discarded into `_`).
//
// The test drives the REAL production startup path (c.ensureSubsystems, which
// is what main() calls at main.go:1393 immediately before handleInteractive),
// so it exercises the same construction sequence a real session does. It uses
// the `conventional-commit` skill, which already shipped in the embedded
// built-in tier BEFORE this fix — so the RED reproduction is independent of any
// skill-manifest work done for HXC-163.
//
// # POLARITY SWITCH — RED_MODE (§11.4.115, repo convention)
//
//   - RED_MODE unset / "0" (DEFAULT — the standing GREEN regression guard):
//     the prompt reaching the provider MUST be the rendered skill body. Run
//     against the PRE-FIX tree this FAILS (the raw utterance is forwarded
//     verbatim), which is the captured proof the guard is not blind.
//
//   - RED_MODE=1 (defect reproduction / golden-bad harness): asserts the
//     PRE-FIX defect shape — the raw utterance reaches the provider untouched
//     and no skill fired. Once the fix lands this branch FAILS, which is
//     correct: the defect is gone.
//
// Run the RED reproduction:
//
//	RED_MODE=1 go test -count=1 -run TestSkillAutoTrigger ./cmd/cli/
//
// Run the standing GREEN guard:
//
//	go test -count=1 -run TestSkillAutoTrigger ./cmd/cli/
func skillWiringRedMode() bool { return os.Getenv("RED_MODE") == "1" }

// skillWiringProvider is a unit-test-only llm.Provider that records every
// request it is asked to stream, so the test can assert on the EXACT prompt
// text the REPL sent. It streams nothing back (the assertion is about the
// outbound prompt, not the completion).
//
// CONST-050(A): fakes are permitted ONLY in unit-test sources — this file is a
// *_test.go compiled without the integration build tag.
type skillWiringProvider struct {
	mu       sync.Mutex
	requests []*llm.LLMRequest
}

func (p *skillWiringProvider) GetType() llm.ProviderType  { return llm.ProviderType("skill-wiring") }
func (p *skillWiringProvider) GetName() string            { return "skill-wiring-recorder" }
func (p *skillWiringProvider) GetModels() []llm.ModelInfo { return []llm.ModelInfo{{Name: "recorder-1"}} }
func (p *skillWiringProvider) GetCapabilities() []llm.ModelCapability { return nil }
func (p *skillWiringProvider) IsAvailable(ctx context.Context) bool   { return true }
func (p *skillWiringProvider) GetHealth(ctx context.Context) (*llm.ProviderHealth, error) {
	return &llm.ProviderHealth{}, nil
}
func (p *skillWiringProvider) Close() error                        { return nil }
func (p *skillWiringProvider) GetContextWindow() int               { return 8192 }
func (p *skillWiringProvider) CountTokens(text string) (int, error) { return len(text) / 4, nil }

func (p *skillWiringProvider) Generate(ctx context.Context, req *llm.LLMRequest) (*llm.LLMResponse, error) {
	p.record(req)
	return &llm.LLMResponse{}, nil
}

func (p *skillWiringProvider) GenerateStream(ctx context.Context, req *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	p.record(req)
	close(ch) // majority provider contract: the provider closes the channel
	return nil
}

func (p *skillWiringProvider) record(req *llm.LLMRequest) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, req)
}

// lastUserPrompt returns the content of the final user message across all
// recorded requests — i.e. exactly what the REPL decided to send.
func (p *skillWiringProvider) lastUserPrompt() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := len(p.requests) - 1; i >= 0; i-- {
		msgs := p.requests[i].Messages
		for j := len(msgs) - 1; j >= 0; j-- {
			if msgs[j].Role == "user" {
				return msgs[j].Content, true
			}
		}
	}
	return "", false
}

// feedStdin replaces os.Stdin with a pipe carrying the supplied lines and
// restores the original when the test ends. Closing the writer yields EOF,
// which is the REPL's clean-exit path.
func feedStdin(t *testing.T, lines string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})
	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.WriteString(lines)
	}()
}

// TestSkillAutoTrigger_WiredIntoInteractiveREPL is the HXC-162 runtime signature.
func TestSkillAutoTrigger_WiredIntoInteractiveREPL(t *testing.T) {
	// The `conventional-commit` built-in skill's trigger is
	//   (?i)^commit message for (?P<summary>.+)$
	// so this utterance MUST route to it and capture the summary.
	const utterance = "commit message for adding retry logic to the worker pool"
	const capturedSummary = "adding retry logic to the worker pool"

	// Isolate both on-disk skill tiers so the built-in tier is the only source
	// and a developer's real ~/.config skills cannot shadow the fixture.
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })

	restore := redirectStdout(t)
	defer restore()

	ctx := context.Background()
	c := NewCLI()
	if c == nil {
		t.Fatal("NewCLI returned nil")
	}
	// Drive the REAL production startup path — this is what main() calls at
	// main.go:1393, immediately before handleInteractive. It is where the
	// skill registry, loader and dispatcher are constructed.
	if err := c.ensureSubsystems(ctx); err != nil {
		t.Fatalf("ensureSubsystems: %v", err)
	}
	defer c.runCleanups()

	// Swap in the recording provider AFTER startup so no real network provider
	// is contacted; handleInteractive reads c.llmProvider at call time.
	rec := &skillWiringProvider{}
	c.llmProvider = rec

	feedStdin(t, utterance+"\n")

	if err := c.handleInteractive(ctx); err != nil {
		t.Fatalf("handleInteractive: %v", err)
	}

	sent, ok := rec.lastUserPrompt()
	if !ok {
		t.Fatalf("the REPL sent no user prompt to the provider at all; " +
			"the utterance never reached the send path (test harness problem, not a verdict)")
	}

	if skillWiringRedMode() {
		// RED: reproduce the defect — the raw utterance is forwarded verbatim
		// because the dispatcher was built and thrown away.
		if sent != utterance {
			t.Fatalf("RED_MODE=1 expected the RAW utterance to reach the provider unchanged "+
				"(the HXC-162 defect: dispatcher constructed then discarded), but got a transformed "+
				"prompt of %d bytes:\n%s\n\nIf the fix has landed this failure is EXPECTED — "+
				"run without RED_MODE for the standing guard.", len(sent), sent)
		}
		return
	}

	// GREEN: the skill fired — the provider received the RENDERED skill body.
	if sent == utterance {
		t.Fatalf("skill auto-trigger did NOT fire: the provider received the raw utterance %q unchanged.\n"+
			"A matching skill (conventional-commit) is registered, so the dispatcher was never consulted "+
			"for this input — the HXC-162 defect.", utterance)
	}
	// Prove it is specifically the conventional-commit skill's rendered body,
	// and that the named capture was substituted into it. Asserting on the
	// rendered CONTENT (not merely "something changed") is what makes this
	// signature unfakeable by any transformation other than a real render.
	if !strings.Contains(sent, "Conventional Commits") {
		t.Fatalf("the prompt sent to the provider is not the conventional-commit skill body:\n%s", sent)
	}
	if !strings.Contains(sent, capturedSummary) {
		t.Fatalf("the skill rendered without its named capture %q substituted; got:\n%s", capturedSummary, sent)
	}
	// The capture token itself must be gone — proof a real render ran rather
	// than the raw manifest body being pasted through.
	if strings.Contains(sent, "{{ARG.summary}}") {
		t.Fatalf("the skill body reached the provider with its capture token unsubstituted:\n%s", sent)
	}
	// NOTE (separate pre-existing defect, deliberately NOT asserted here):
	// the shipped conventional-commit manifest writes its declared variable as
	// `{{spec_url}}`, but the substitution engine resolves declared variables
	// only under the `{{ARG.<name>}}` form (see MarkdownCommand.buildResolver —
	// unknown tokens are left verbatim by design). That bare token therefore
	// leaks into the outgoing prompt. It is a manifest-authoring bug that
	// predates HXC-162 and is independent of skill wiring, so it is reported
	// rather than silently changed under this fix. Asserting on it here would
	// couple the wiring guard to an unrelated defect.
}
