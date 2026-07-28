package server

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// llm_default_model_regression_test.go — standing regression guard (§11.4.135)
// for a REAL reproduced server defect, authored RED-on-the-broken-artifact with
// a single RED_MODE polarity switch (§11.4.115) and a §11.4.146 STEP-3 extend
// pass across the provider set.
//
// THE DEFECT (RED, captured on the pre-fix artifact via a REAL DeepSeek call):
// a Generate / Stream request that OMITS the model (Model == "") for a named
// cloud provider left llm.LLMRequest.Model == "" all the way to the wire. The
// handler's resolveLLMProvider only set entry.Models when model != "", so the
// provider received an empty model. DeepSeek (which does not synthesise its own
// default) rejected it upstream:
//
//	DeepSeek API returned status 400: {"error":{"message":"The supported API
//	model names are deepseek-v4-pro or deepseek-v4-flash, but you passed .", ...}}
//
// which the generateLLM handler then surfaced as HTTP 502. (deepseek-chat /
// deepseek-coder / deepseek-reasoner — the offline SEED list — are ALL
// deprecated; only deepseek-v4-pro / deepseek-v4-flash are served today, and
// the LIVE /models catalog returns exactly those two.)
//
// THE FIX (CONST-036/037): when the request omits the model, the handler
// resolves it to a verified-available model from provider.GetModels() (the
// LLMsVerifier-sourced / live-/models catalog) BEFORE calling Generate, so an
// empty model is never sent upstream. No hardcoded model literal.
//
// CONST-050(A): the fakes below live ONLY in this *_test.go unit file.
// Production code never references them.

// modelRecordingProvider is a deterministic, network-free llm.Provider that
// records the Model field of the request it is asked to Generate/Stream, and
// advertises a fixed catalog. It is the oracle for "what model did the handler
// pass to the provider?" — the exact observable the defect is about.
type modelRecordingProvider struct {
	name     string
	ptype    llm.ProviderType
	catalog  []llm.ModelInfo
	gotModel string // the Model the handler passed to Generate/Stream
	// respModel, when non-empty, is echoed back as LLMResponse.Model — the
	// backend's ACTUAL served-model identity, which may legitimately differ
	// from the requested alias (gotModel). Zero value ("") preserves prior
	// behaviour for every existing test in this file that does not set it.
	respModel string
}

func (p *modelRecordingProvider) GetType() llm.ProviderType              { return p.ptype }
func (p *modelRecordingProvider) GetName() string                        { return p.name }
func (p *modelRecordingProvider) GetModels() []llm.ModelInfo             { return p.catalog }
func (p *modelRecordingProvider) GetCapabilities() []llm.ModelCapability { return nil }
func (p *modelRecordingProvider) IsAvailable(ctx context.Context) bool   { return true }
func (p *modelRecordingProvider) GetContextWindow() int                  { return 128000 }
func (p *modelRecordingProvider) CountTokens(text string) (int, error)   { return len(text) / 4, nil }
func (p *modelRecordingProvider) Close() error                           { return nil }

func (p *modelRecordingProvider) GetHealth(ctx context.Context) (*llm.ProviderHealth, error) {
	return &llm.ProviderHealth{Status: "healthy", LastCheck: time.Now()}, nil
}

func (p *modelRecordingProvider) Generate(ctx context.Context, req *llm.LLMRequest) (*llm.LLMResponse, error) {
	p.gotModel = req.Model
	return &llm.LLMResponse{ID: uuid.New(), Content: "4", Model: p.respModel}, nil
}

func (p *modelRecordingProvider) GenerateStream(ctx context.Context, req *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	p.gotModel = req.Model
	defer close(ch)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- llm.LLMResponse{ID: uuid.New(), Content: "4", CreatedAt: time.Now()}:
	}
	return nil
}

// deepseekLiveCatalog is the verified-available DeepSeek catalog observed LIVE
// (GET /models) on the broken artifact during RED capture — the offline seed
// (deepseek-chat/coder/reasoner) is deprecated; these two are what is served.
func deepseekLiveCatalog() []llm.ModelInfo {
	return []llm.ModelInfo{
		{Name: "deepseek-v4-flash", Provider: llm.ProviderTypeDeepSeek, ContextSize: 128000, MaxTokens: 8192},
		{Name: "deepseek-v4-pro", Provider: llm.ProviderTypeDeepSeek, ContextSize: 128000, MaxTokens: 8192},
	}
}

// TestDefaultModelResolution_PolaritySwitch is the §11.4.115 RED/GREEN guard at
// the handler layer. BOTH polarities drive the SAME real generateLLM handler
// over the SAME fixture — a request that OMITS the model, answered by a provider
// that records the model it was actually asked for. Only the final assertion
// flips (§11.4.115 one-source-two-roles).
//
//   - RED_MODE=1: assert the provider received an EMPTY model. That is TRUE on a
//     pre-fix artifact (no catalog-backed resolution existed, so the omitted
//     model reached the wire empty — the captured DeepSeek 400) and FALSE on the
//     fixed artifact, where the handler resolves a verified-available model.
//   - RED_MODE=0 (default, GREEN): assert the provider received a
//     verified-available catalog model — the defect is ABSENT.
//
// The RED branch deliberately does NOT reconstruct the pre-fix resolution
// locally: a local replica would assert against a copy of the old code, which
// behaves identically on every artifact ever built, so it could never fail and
// would prove nothing (§11.4.1 / §11.4.115).
func TestDefaultModelResolution_PolaritySwitch(t *testing.T) {
	rec := &modelRecordingProvider{
		name:    "DeepSeek",
		ptype:   llm.ProviderTypeDeepSeek,
		catalog: deepseekLiveCatalog(),
	}
	withFakeResolver(t, rec)

	// SHARED drive path — the REAL handler, both polarities.
	srv := &Server{}
	w, body := postJSON(t, "/api/v1/llm/generate", srv.generateLLM,
		`{"prompt":"What is 2+2? Reply with only the number."}`)

	require.Equal(t, http.StatusOK, w.Code,
		"the omitted-model generate must reach the provider (not fail early); body=%v", body)
	assert.Equal(t, "success", body["status"])

	if redMode(t) {
		// RED: the defect's exact observable on the pre-fix artifact — the
		// handler forwarded the omitted model to the provider UNCHANGED, i.e.
		// empty, which is what DeepSeek rejected upstream with the captured 400.
		require.Equal(t, "", rec.gotModel,
			"RED expectation: the PRE-FIX handler passes the omitted model through to the provider "+
				"unchanged (empty) — the defect. A non-empty %q means the catalog-backed default "+
				"resolution is present, so the defect is absent and this RED baseline no longer "+
				"characterises anything", rec.gotModel)
		t.Logf("RED reproduced: handler sent empty model %q to the provider (would 400 upstream)", rec.gotModel)
		return
	}

	// GREEN: the model the handler passed to the provider MUST be a
	// verified-available catalog model, never empty.
	require.NotEmpty(t, rec.gotModel,
		"GREEN: handler must resolve the omitted model to a verified-available catalog model, got empty (the defect)")
	catalogNames := modelNames(rec.catalog)
	assert.Contains(t, catalogNames, rec.gotModel,
		"GREEN: resolved model %q must come from the provider's catalog %v (CONST-036/037, no hardcoded literal)",
		rec.gotModel, catalogNames)
	// And it must NOT be one of the deprecated seed names that caused the 400.
	for _, dead := range []string{"deepseek-chat", "deepseek-coder", "deepseek-reasoner", ""} {
		assert.NotEqual(t, dead, rec.gotModel,
			"GREEN: resolved model must not be the deprecated/empty model %q that produced the upstream 400", dead)
	}
	t.Logf("GREEN: omitted-model generate resolved to verified-available model %q", rec.gotModel)
}

// TestDefaultModelResolution_Stream_PolaritySwitch proves the streaming handler
// carries the identical fix (the streamLLM path passed req.Model verbatim too).
// Same one-source-two-roles shape as the non-streaming guard above: BOTH
// polarities drive the REAL streamLLM handler and read the SAME observable —
// the model the provider was actually asked for — with only the assertion
// flipping.
func TestDefaultModelResolution_Stream_PolaritySwitch(t *testing.T) {
	rec := &modelRecordingProvider{
		name:    "DeepSeek",
		ptype:   llm.ProviderTypeDeepSeek,
		catalog: deepseekLiveCatalog(),
	}
	withFakeResolver(t, rec)

	// SHARED drive path — the REAL streaming handler, both polarities.
	srv := &Server{}
	w, _ := postJSON(t, "/api/v1/llm/stream", srv.streamLLM, `{"prompt":"hi"}`)
	require.Equal(t, http.StatusOK, w.Code, "streamLLM with an omitted model must not error out at resolution")

	if redMode(t) {
		require.Equal(t, "", rec.gotModel,
			"RED expectation: the PRE-FIX streaming handler also forwards the omitted model to the "+
				"provider empty — the defect. Got %q, which means the fix is present", rec.gotModel)
		return
	}

	require.NotEmpty(t, rec.gotModel, "GREEN: streamLLM must resolve the omitted model to a catalog model")
	assert.Contains(t, modelNames(rec.catalog), rec.gotModel)
}

// TestResolveDefaultModel_ExtendAcrossProviders is the §11.4.146 STEP-3 fan-out:
// the default-resolution helper is exercised across the cloud-provider set, plus
// the boundary/edge cases (empty catalog, blank-Name-but-ID, explicit model,
// whitespace-only model). It proves the fix is SOUND for the provider set — no
// provider with a reachable catalog defaults to an empty/unavailable model, and
// an explicit model is always honoured unchanged.
func TestResolveDefaultModel_ExtendAcrossProviders(t *testing.T) {
	twoModel := func(pt llm.ProviderType, a, b string) []llm.ModelInfo {
		return []llm.ModelInfo{
			{Name: a, Provider: pt}, {Name: b, Provider: pt},
		}
	}

	tests := []struct {
		name      string
		catalog   []llm.ModelInfo
		requested string
		want      string
	}{
		// Provider-set fan-out: an omitted model resolves to the leading
		// verified-available catalog model for each provider.
		{"deepseek_omitted", deepseekLiveCatalog(), "", "deepseek-v4-flash"},
		{"openai_omitted", twoModel(llm.ProviderTypeOpenAI, "gpt-5", "gpt-5-mini"), "", "gpt-5"},
		{"mistral_omitted", twoModel(llm.ProviderTypeMistral, "mistral-large", "mistral-small"), "", "mistral-large"},
		{"groq_omitted", twoModel(llm.ProviderTypeGroq, "llama-3.3-70b", "llama-3.1-8b"), "", "llama-3.3-70b"},

		// Explicit model is honoured unchanged for every provider (no override).
		{"deepseek_explicit", deepseekLiveCatalog(), "deepseek-v4-pro", "deepseek-v4-pro"},
		{"openai_explicit", twoModel(llm.ProviderTypeOpenAI, "gpt-5", "gpt-5-mini"), "gpt-4o", "gpt-4o"},

		// Edge: whitespace-only requested model is treated as omitted.
		{"whitespace_treated_as_omitted", deepseekLiveCatalog(), "   ", "deepseek-v4-flash"},

		// Edge: a catalog whose first entry has a blank Name but a populated ID
		// falls back to the ID (some providers populate only ID).
		{"blank_name_falls_back_to_id", []llm.ModelInfo{
			{Name: "", ID: "id-only-model", Provider: llm.ProviderTypeOpenAI},
		}, "", "id-only-model"},

		// Edge: a catalog whose leading entry is fully blank is skipped in favour
		// of the next usable entry (defensive against junk rows).
		{"skips_blank_leading_entry", []llm.ModelInfo{
			{Name: "  ", ID: "  "},
			{Name: "deepseek-v4-pro", Provider: llm.ProviderTypeDeepSeek},
		}, "", "deepseek-v4-pro"},

		// HONEST BOUNDARY (§11.4.6): empty catalog (offline/unreachable) leaves
		// the model empty — the server does NOT invent a model; the provider's
		// own default/honest-error path takes over. This is the documented,
		// intended behaviour, NOT a regression.
		{"empty_catalog_left_empty", nil, "", ""},
		{"empty_catalog_explicit_kept", nil, "claude-x", "claude-x"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &modelRecordingProvider{catalog: tc.catalog}
			got := resolveDefaultModel(p, tc.requested)
			require.Equal(t, tc.want, got)
			// Anti-bluff invariant: when the catalog is non-empty AND the request
			// omitted the model, the result MUST be a real catalog entry, never
			// empty (the exact failure that produced the upstream 400).
			if strings.TrimSpace(tc.requested) == "" && len(tc.catalog) > 0 {
				require.NotEmpty(t, got,
					"non-empty catalog + omitted model must never resolve to empty (the defect)")
			}
		})
	}
}

// modelNames returns the catalog model Names for assertion messages.
func modelNames(catalog []llm.ModelInfo) []string {
	out := make([]string, 0, len(catalog))
	for _, m := range catalog {
		out = append(out, m.Name)
	}
	return out
}
