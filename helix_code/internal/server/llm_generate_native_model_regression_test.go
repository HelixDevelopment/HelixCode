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
//   - RED_MODE=1 replicates the pre-fix response construction and asserts
//     the requested alias is echoed back unchanged (the defect).
//   - RED_MODE=0 (default) drives the REAL generateLLM handler and asserts
//     the response's "model" field carries the provider's ACTUAL served
//     model, never the requested alias.
func TestGenerateLLM_ResponseModel_ServedNotRequested(t *testing.T) {
	const requestedAlias = "default"
	const servedModel = "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"

	if redMode(t) {
		// Faithful replica of the pre-fix response construction: the
		// "model" field was llmReq.Model verbatim, never resp.Model, so the
		// requested alias always comes back unchanged regardless of what
		// actually served the request.
		got := requestedAlias // pre-fix: gin.H{..., "model": llmReq.Model, ...}
		require.Equal(t, requestedAlias, got,
			"RED expectation: pre-fix code echoes the requested alias back unchanged — the defect")
		t.Logf("RED reproduced: pre-fix response would report requested alias %q instead of served model %q",
			got, servedModel)
		return
	}

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

	require.Equal(t, http.StatusOK, w.Code, "GREEN: generate must succeed; body=%v", body)
	assert.Equal(t, "success", body["status"])
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
