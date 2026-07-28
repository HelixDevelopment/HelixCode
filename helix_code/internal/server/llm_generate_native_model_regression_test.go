package server

import (
	"net/http"
	"testing"

	"dev.helix.code/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// llm_generate_native_model_regression_test.go — standing regression guard
// (§11.4.135) closing a THIRD instance of the CONST-036/037 served-model
// fabrication defect, found by an independent code review of commit
// e330149f. That commit fixed the client-visible "model" field on the OpenAI
// and Anthropic wire facades (wire_facade.go llmResponseToOpenAI /
// llmResponseToAnthropic) so each reports the model the provider ACTUALLY
// served rather than the caller's requested alias — but left this project's
// OWN primary API, POST /api/v1/llm/generate, doing exactly the same thing
// wrong: generateLLM's success response hardcoded "model": llmReq.Model (the
// REQUESTED string, e.g. "default") even when the provider reported a
// different, concrete resp.Model (e.g. a resolved llama.cpp GGUF path). A
// client that asked for "default" and got served by a completely different
// model was told "default" back — the identical CONST-036/037 defect on a
// third client-visible JSON surface.
//
// Both polarities drive the SAME real handler over the SAME fixture — only
// the final assertion flips (§11.4.115 one-source-two-roles):
//
//   - RED_MODE=1 asserts the response's "model" is the REQUESTED alias. That
//     holds on a pre-fix artifact (where generateLLM emitted
//     `"model": llmReq.Model`), so the test PASSES there and reproduces the
//     defect; on the fixed artifact it FAILS, and that failure is the proof
//     the fix is genuinely present in the binary under test.
//   - RED_MODE=0 (default) asserts the response's "model" carries the
//     provider's ACTUAL served model, never the requested alias.
//
// The RED branch deliberately does NOT reconstruct the pre-fix expression
// locally: a local replica would assert against a copy of the old code and
// pass on every artifact ever built, proving nothing (§11.4.1 bluff-gate).
func TestGenerateLLM_ResponseModel_ServedNotRequested(t *testing.T) {
	const requestedAlias = "default"
	const servedModel = "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"

	rec := &modelRecordingProvider{
		name:      "llamacpp",
		ptype:     llm.ProviderTypeLlamaCpp,
		catalog:   []llm.ModelInfo{{Name: servedModel, Provider: llm.ProviderTypeLlamaCpp}},
		respModel: servedModel, // the backend's ACTUAL served model
	}
	withFakeResolver(t, rec)

	srv := &Server{}
	w, body := postJSON(t, "/api/v1/llm/generate", srv.generateLLM,
		`{"prompt":"hi","model":"`+requestedAlias+`"}`)

	require.Equal(t, http.StatusOK, w.Code, "generate must succeed; body=%v", body)
	assert.Equal(t, "success", body["status"])
	require.Equal(t, requestedAlias, rec.gotModel,
		"fixture sanity: the handler must have asked the provider for the requested alias")

	if redMode(t) {
		assert.Equal(t, requestedAlias, body["model"],
			"RED expectation: the PRE-FIX handler echoes the requested alias %q back unchanged — the defect. "+
				"On the FIXED artifact this assertion FAILS (the response now carries the served model %q), "+
				"and that failure is the proof the fix reached the binary.",
			requestedAlias, servedModel)
		return
	}

	assert.Equal(t, servedModel, body["model"],
		"GREEN: response 'model' must report the model the provider ACTUALLY served (%q), never the requested alias (%q) — CONST-036/037",
		servedModel, requestedAlias)
	assert.NotEqual(t, requestedAlias, body["model"],
		"GREEN: response 'model' must not merely echo the requested alias back")
}

// TestGenerateLLM_ResponseModel_FallsBackWhenProviderOmitsIt proves the
// honest fallback half of the fix: when the provider's response omits Model
// entirely (an honest "backend reported nothing" case, not a bug), the
// response must still carry a non-empty, meaningful model — the
// resolved/requested value — rather than emitting an empty string, so the
// field is never empty regardless of provider behaviour.
func TestGenerateLLM_ResponseModel_FallsBackWhenProviderOmitsIt(t *testing.T) {
	rec := &modelRecordingProvider{
		name:    "llamacpp",
		ptype:   llm.ProviderTypeLlamaCpp,
		catalog: []llm.ModelInfo{{Name: "llama3.2", Provider: llm.ProviderTypeLlamaCpp}},
		// respModel intentionally left "" — provider reports no served model.
	}
	withFakeResolver(t, rec)

	srv := &Server{}
	w, body := postJSON(t, "/api/v1/llm/generate", srv.generateLLM,
		`{"prompt":"hi","model":"llama3.2"}`)

	require.Equal(t, http.StatusOK, w.Code, "body=%v", body)
	assert.Equal(t, "llama3.2", body["model"],
		"GREEN: when the provider reports no served model, the response must fall back to the resolved requested model rather than emitting empty")
}
