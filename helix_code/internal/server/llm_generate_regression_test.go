package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// llm_generate_regression_test.go — standing regression guards (§11.4.135) for
// two REAL reproduced server defects, each authored RED-on-the-broken-artifact
// with a single RED_MODE polarity switch (§11.4.115):
//
//   - RED_MODE=1 reproduces the historical defect by driving the SHIPPED code
//     path and asserting the defect IS present. Run against a pre-fix artifact
//     it PASSES (it captures the crash / the Ollama mask); run against the fixed
//     artifact it FAILS. That falsifiability is what makes it a real baseline —
//     a RED branch that reconstructs the old logic locally would behave the same
//     on every artifact ever built and could never fail (§11.4.1 / §11.4.115).
//   - RED_MODE=0 (default) is the standing GREEN guard asserting the defect is
//     ABSENT in the shipped handler.
//
// Defect #5 (CRITICAL): /api/v1/llm/stream double channel-close. The provider's
// GenerateStream owns `defer close(ch)`; the OLD streamLLM ALSO did
// `defer close(chunkChan)` on the same channel → `panic: close of closed
// channel` in the spawned producer goroutine → uncatchable by gin.Recovery →
// the whole process dies from one client request.
//
// Defect #4 (MEDIUM): an explicitly-named UNKNOWN provider silently fell back to
// the local Ollama default, so a user's provider typo surfaced as a misleading
// Ollama 404 instead of a clear "unknown provider" 400.
//
// CONST-050(A): the fake provider below lives ONLY in this *_test.go unit file.
// Production code never references it.

// redMode moved to red_polarity_convention_test.go (HXC-155): the package's
// polarity switches now share one helper shape and one enumeration point.
// Per §11.4.115 the SAME source still serves both polarities.

// closingFakeProvider is a deterministic, network-free llm.Provider whose
// GenerateStream obeys the channel-ownership contract: it sends a chunk then
// closes ch via `defer close(ch)` — exactly like deepseek/openai and the other
// SENDER-closes providers. Driving the real streamLLM with this provider is
// what would double-close (and crash) under the OLD consumer-also-closes code.
type closingFakeProvider struct {
	chunks []string
}

func (f *closingFakeProvider) GetType() llm.ProviderType              { return llm.ProviderTypeOllama }
func (f *closingFakeProvider) GetName() string                        { return "fake-closing" }
func (f *closingFakeProvider) GetModels() []llm.ModelInfo             { return nil }
func (f *closingFakeProvider) GetCapabilities() []llm.ModelCapability { return nil }
func (f *closingFakeProvider) IsAvailable(ctx context.Context) bool   { return true }
func (f *closingFakeProvider) GetContextWindow() int                  { return 4096 }
func (f *closingFakeProvider) CountTokens(text string) (int, error)   { return len(text) / 4, nil }
func (f *closingFakeProvider) Close() error                           { return nil }

func (f *closingFakeProvider) GetHealth(ctx context.Context) (*llm.ProviderHealth, error) {
	return &llm.ProviderHealth{Status: "healthy", LastCheck: time.Now()}, nil
}

func (f *closingFakeProvider) Generate(ctx context.Context, req *llm.LLMRequest) (*llm.LLMResponse, error) {
	return &llm.LLMResponse{ID: uuid.New(), Content: strings.Join(f.chunks, "")}, nil
}

// GenerateStream is the SENDER and SOLE closer of ch (the contract).
func (f *closingFakeProvider) GenerateStream(ctx context.Context, req *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	defer close(ch)
	for _, c := range f.chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- llm.LLMResponse{ID: uuid.New(), Content: c, CreatedAt: time.Now()}:
		}
	}
	return nil
}

// withFakeResolver temporarily points the handler's provider resolver at a fake
// that returns p, restoring the real resolver on cleanup.
func withFakeResolver(t *testing.T, p llm.Provider) {
	t.Helper()
	prev := llmProviderResolver
	llmProviderResolver = func(providerName, model string) (llm.Provider, error) { return p, nil }
	t.Cleanup(func() { llmProviderResolver = prev })
}

// streamCrashProbeEnv turns this test binary into the one-shot child probe that
// the RED branch of TestStreamLLM_NoDoubleCloseCrash_RegressionGuard re-executes.
const streamCrashProbeEnv = "HELIX_STREAM_DOUBLECLOSE_PROBE"

// streamProbeJoinWindow is how long the child probe waits after ServeHTTP for
// the handler's producer goroutine to run its deferred close. ServeHTTP can
// return before that goroutine finishes, and without the wait a pre-fix artifact
// could let the probe exit 0 before the panic lands — a false GREEN. Generous
// relative to an in-memory channel pump (microseconds), and paid only in RED_MODE.
const streamProbeJoinWindow = 2 * time.Second

// streamProbeTimeout bounds the child probe so a hang cannot wedge the suite.
const streamProbeTimeout = 90 * time.Second

// TestStreamLLM_DoubleCloseProbe is the CHILD half of the RED branch below. It
// is not a standalone guard: it is skipped unless re-executed by that parent
// with streamCrashProbeEnv set.
//
// It drives the REAL streamLLM handler with a SENDER-closes provider IN THIS
// PROCESS. On a pre-fix artifact the handler's own producer goroutine also
// closes chunkChan, panicking ("close of closed channel") inside a goroutine
// that gin.Recovery cannot reach — so THIS PROCESS DIES. That process death is
// exactly the observable the parent asserts on, and it is why the defect was
// CRITICAL: a single client request killed the whole server. On the fixed
// artifact the request completes and this process exits 0.
//
// A child process is REQUIRED: an unrecovered panic in a spawned goroutine
// cannot be caught in-process by recover() or require.NotPanics — it takes the
// whole test binary down with it. Observing the crash therefore means observing
// a process, which is what makes this RED branch falsifiable instead of a
// self-fulfilling local replica (§11.4.1 / §11.4.115).
func TestStreamLLM_DoubleCloseProbe(t *testing.T) {
	if os.Getenv(streamCrashProbeEnv) != "1" {
		t.Skip("SKIP-OK: child-process probe, driven only by the RED branch of " +
			"TestStreamLLM_NoDoubleCloseCrash_RegressionGuard; it is not a standalone guard")
	}

	fake := &closingFakeProvider{chunks: []string{"Hello", " world"}}
	withFakeResolver(t, fake)
	gin.SetMode(gin.TestMode)
	srv := &Server{}
	router := gin.New()
	router.Use(gin.Recovery()) // mirrors production — and provably cannot save us here
	router.POST("/api/v1/llm/stream", srv.streamLLM)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/v1/llm/stream",
		strings.NewReader(`{"prompt":"hi"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	time.Sleep(streamProbeJoinWindow) // join the producer goroutine's deferred close
	t.Logf("probe survived: handler streamed %d bytes with no double-close", w.Body.Len())
}

// TestStreamLLM_NoDoubleCloseCrash_RegressionGuard — Defect #5 guard.
//
// BOTH polarities drive the REAL streamLLM handler over the same SENDER-closes
// provider; only the observation point and the assertion flip (§11.4.115
// one-source-two-roles). RED observes the handler from a child process (the only
// way to see a goroutine panic, which is process death); GREEN observes it
// in-process, where a crash would take this binary down and fail the test just
// as loudly.
//
// RED_MODE=1: re-execute this binary as a probe child and assert it DIES with
// "close of closed channel". True on a pre-fix artifact; false on the fixed one,
// where the child exits 0.
//
// RED_MODE=0 (default, GREEN guard): assert the request completes with a real
// SSE body (data: ... + [DONE]) and no crash — the consumer no longer
// double-closes.
func TestStreamLLM_NoDoubleCloseCrash_RegressionGuard(t *testing.T) {
	fake := &closingFakeProvider{chunks: []string{"Hello", " world"}}

	if redMode(t) {
		ctx, cancel := context.WithTimeout(context.Background(), streamProbeTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0],
			"-test.run=^TestStreamLLM_DoubleCloseProbe$", "-test.v=true")
		cmd.Env = append(os.Environ(), streamCrashProbeEnv+"=1")
		out, runErr := cmd.CombinedOutput()

		require.NoError(t, ctx.Err(),
			"the child probe must terminate on its own, not hit the %s timeout.\nchild output:\n%s",
			streamProbeTimeout, out)
		require.Error(t, runErr,
			"RED expectation: driving the REAL streamLLM handler on a PRE-FIX artifact MUST kill the "+
				"probe process (unrecovered double-close panic in the producer goroutine). A clean exit "+
				"means the consumer no longer double-closes, so the defect is absent and this RED "+
				"baseline no longer characterises anything.\nchild output:\n%s", out)
		require.Contains(t, string(out), "close of closed channel",
			"RED: the child must die from the double-close crash this guard exists to prevent, not "+
				"from an unrelated failure.\nchild output:\n%s", out)
		t.Logf("RED reproduced: the real streamLLM handler killed the probe process (%v)", runErr)
		return
	}

	// GREEN guard: the REAL handler must NOT crash and MUST stream honestly.
	withFakeResolver(t, fake)
	gin.SetMode(gin.TestMode)
	srv := &Server{}
	router := gin.New()
	router.Use(gin.Recovery()) // mirrors production; proves we don't even rely on it
	router.POST("/api/v1/llm/stream", srv.streamLLM)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/v1/llm/stream",
		strings.NewReader(`{"prompt":"hi"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// A double-close panics in the handler's producer goroutine, which no
	// recover() can reach — it kills this process outright, so this test fails
	// hard rather than reporting a caught panic. NotPanics still covers the
	// in-handler synchronous paths.
	require.NotPanics(t, func() { router.ServeHTTP(w, req) },
		"streamLLM must not panic — a double channel-close would crash the server process")

	body := w.Body.String()
	assert.Contains(t, body, "data: Hello", "real provider chunk must be streamed as SSE")
	assert.Contains(t, body, "data: [DONE]", "stream must terminate with the [DONE] frame")
	assert.NotContains(t, strings.ToLower(body), "close of closed channel")
}

// TestResolveLLMProvider_UnknownProviderNoSilentOllamaFallback_RegressionGuard
// — Defect #4 guard.
//
// Both polarities drive the SAME real shipped resolveLLMProvider over the SAME
// unknown provider name; only the assertion flips (§11.4.115
// one-source-two-roles). The RED branch deliberately does NOT reconstruct the
// pre-fix fall-through locally — a local replica (constructing an Ollama
// provider by hand and asserting its name is "ollama") is true on EVERY artifact
// ever built, so it could never fail and would prove nothing (§11.4.1).
//
// RED_MODE=1: assert the OLD behaviour — the real resolver swallows the unknown
// name and hands back the Ollama default with NO error, masking the typo. True
// on a pre-fix artifact; false on the fixed one (errUnknownProvider, nil
// provider).
//
// RED_MODE=0 (default, GREEN guard): assert the FIXED behaviour — an unknown
// named provider yields errUnknownProvider (no provider), and the handler
// answers 400 naming the bad provider, never a silent Ollama fallback.
func TestResolveLLMProvider_UnknownProviderNoSilentOllamaFallback_RegressionGuard(t *testing.T) {
	t.Setenv("HELIX_LLM_PROVIDER", "") // ensure only the request-named provider matters

	// SHARED drive path — the REAL shipped resolver, both polarities.
	prov, err := resolveLLMProvider("definitely-not-a-real-provider", "")
	if prov != nil {
		defer func() { _ = prov.Close() }()
	}

	if redMode(t) {
		// RED: the defect's exact observable on the pre-fix artifact — llm.Select
		// fails for the unknown name, the resolver falls through to the local
		// Ollama default and returns it with NO error, so the user's typo
		// surfaced later as a misleading Ollama 404 instead of a clear 400.
		require.NoError(t, err,
			"RED expectation: the PRE-FIX resolver silently swallowed the unknown provider name. "+
				"An error here (%v) means the unknown-provider rejection is present, so the defect "+
				"is absent and this RED baseline no longer characterises anything", err)
		require.NotNil(t, prov, "RED expectation: the PRE-FIX resolver returned a provider anyway")
		assert.Equal(t, "ollama", prov.GetName(),
			"RED expectation: the pre-fix path silently returned the Ollama default for an unknown provider")
		return
	}

	// GREEN guard 1: resolveLLMProvider rejects the unknown provider, no fallback.
	require.Error(t, err, "an explicitly-named unknown provider must NOT silently fall back to Ollama")
	assert.Nil(t, prov, "no provider must be constructed for an unknown provider name")
	assert.ErrorIs(t, err, errUnknownProvider, "the error must be the unknown-provider sentinel")
	assert.Contains(t, err.Error(), "definitely-not-a-real-provider",
		"the error must echo the bad provider name the user supplied")

	// GREEN guard 2: the handler maps that to a real 400 (client error), not 503.
	srv := &Server{}
	w, body := postJSON(t, "/api/v1/llm/generate", srv.generateLLM,
		`{"prompt":"hi","provider":"definitely-not-a-real-provider"}`)
	require.Equal(t, http.StatusBadRequest, w.Code,
		"an unknown provider is a client error (400), never a 503 or a masked Ollama 404")
	require.NotNil(t, body)
	assert.Equal(t, "error", body["status"])
	errMsg, _ := body["error"].(string)
	assert.Contains(t, errMsg, "unknown provider", "the 400 body must name the unknown-provider cause")
	assert.NotContains(t, strings.ToLower(errMsg), "ollama",
		"the unknown-provider error must NOT be masked as an Ollama failure")
}

// TestResolveLLMProvider_NoProviderNamedStillFallsBackToOllama proves the fix is
// surgical: when NO provider is named, the default Ollama fallback is preserved
// (out-of-the-box zero-config behaviour). Only an EXPLICIT unknown name is
// rejected.
func TestResolveLLMProvider_NoProviderNamedStillFallsBackToOllama(t *testing.T) {
	t.Setenv("HELIX_LLM_PROVIDER", "")
	prov, err := resolveLLMProvider("", "")
	require.NoError(t, err, "no provider named must still resolve to the local Ollama default")
	require.NotNil(t, prov)
	defer func() { _ = prov.Close() }()
	assert.Equal(t, "ollama", prov.GetName(),
		"zero-config default must remain the local Ollama provider")
}
