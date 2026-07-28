package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dev.helix.code/internal/llm"
	"github.com/gin-gonic/gin"
)

// Model-identity regression guards (CONST-036 / CONST-037, §11.4.135).
//
// THE DEFECT THESE GUARD (captured 2026-07-28, §11.4.115 RED baseline)
// llm.LLMResponse carried no Model field at all, so the wire facade could only
// echo the model string the CALLER asked for. A client that requested the
// alias "default" against the local HelixLLM coder was told
//
//	"model":"default"
//
// while /models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf actually served the
// request — a model identity that never served anything. The coder itself
// reports the real path both in /v1/models and in its own completion response;
// HelixCode dropped it on the floor.
//
// POLARITY SWITCH (§11.4.115)
// Run with RED_MODE=1 to assert the DEFECT IS PRESENT (the pre-fix behaviour:
// the facade echoes the request). Default RED_MODE=0 is the standing GREEN
// guard asserting the defect is ABSENT. One source, two roles — the
// bug-catcher IS the regression guard, so a future refactor that drops the
// servedModel preference turns these red instead of sailing through.
//
// Why these are unit tests on the conversion functions rather than only an
// end-to-end capture: the §11.4.83 QA capture proves the behaviour on the live
// helixllm route, but it needs a running coder and a built artifact. These run
// on every `go test` with no infrastructure, so the guard cannot rot silently
// when the coder is unavailable.
//
// The RED_MODE polarity helper is the package's existing redMode() (declared in
// llm_generate_regression_test.go) — reused rather than redeclared so both
// guard families flip polarity from the same switch.

// TestOpenAIFacade_ReportsServedModelNotRequestedAlias is the core guard: the
// provider served a concrete model, the caller asked for an alias, and the
// OpenAI-wire response must name the concrete model.
func TestOpenAIFacade_ReportsServedModelNotRequestedAlias(t *testing.T) {
	const (
		requestedAlias = "default"
		servedModel    = "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"
	)

	resp := &llm.LLMResponse{
		Content:      "hello",
		FinishReason: "stop",
		Model:        servedModel, // what the provider actually served
	}

	out := llmResponseToOpenAI(resp, requestedAlias)

	if redMode(t) {
		// Pre-fix behaviour: the facade echoed the request. If this assertion
		// fails while RED_MODE=1, the defect is already gone and the RED
		// baseline is stale — that is a finding, not a pass (§11.4.115).
		if out.Model != requestedAlias {
			t.Fatalf("RED_MODE: expected the pre-fix echo %q, got %q — the defect is not reproducible, so this RED baseline no longer characterises anything",
				requestedAlias, out.Model)
		}
		return
	}

	if out.Model != servedModel {
		t.Errorf("response must name the model that ACTUALLY served the request (CONST-036/037): got %q, want %q", out.Model, servedModel)
	}
	if out.Model == requestedAlias {
		t.Errorf("response echoed the requested alias %q — this is the exact defect this guard exists to catch", requestedAlias)
	}
}

// TestOpenAIFacade_FallsBackToRequestedModelWhenProviderSilent pins the other
// half of the contract: providers that report no model must NOT produce an
// empty model field. Without this, "prefer the served model" could regress
// into "emit empty", which is worse than the original defect.
func TestOpenAIFacade_FallsBackToRequestedModelWhenProviderSilent(t *testing.T) {
	if redMode(t) {
		t.Skip("fallback behaviour is unchanged by the fix; nothing to reproduce")
	}
	const requested = "llama3.2"

	// Model deliberately unset — the shape every non-OpenAI-compatible
	// provider still produces.
	out := llmResponseToOpenAI(&llm.LLMResponse{Content: "hi"}, requested)

	if out.Model != requested {
		t.Errorf("with a silent provider the facade must fall back to the requested model: got %q, want %q", out.Model, requested)
	}
	if out.Model == "" {
		t.Error("model field must never be emitted empty")
	}
}

// TestAnthropicFacade_ReportsServedModelNotRequestedAlias covers the second
// client-visible wire. The review that caught the original gap noted this
// defect class sat one function away from the fixed one; this guard closes
// the extend-to-all-cases hole (§11.4.146 STEP 3).
func TestAnthropicFacade_ReportsServedModelNotRequestedAlias(t *testing.T) {
	const (
		requestedAlias = "default"
		servedModel    = "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"
	)

	resp := &llm.LLMResponse{Content: "hello", FinishReason: "stop", Model: servedModel}
	out := llmResponseToAnthropic(resp, requestedAlias)

	if redMode(t) {
		if out.Model != requestedAlias {
			t.Fatalf("RED_MODE: expected the pre-fix echo %q, got %q", requestedAlias, out.Model)
		}
		return
	}

	if out.Model != servedModel {
		t.Errorf("anthropic response must name the served model: got %q, want %q", out.Model, servedModel)
	}
}

// TestAnthropicFacade_FallsBackToRequestedModelWhenProviderSilent mirrors the
// OpenAI fallback guard for the Anthropic wire.
func TestAnthropicFacade_FallsBackToRequestedModelWhenProviderSilent(t *testing.T) {
	if redMode(t) {
		t.Skip("fallback behaviour is unchanged by the fix")
	}
	const requested = "claude-3-5-sonnet"

	out := llmResponseToAnthropic(&llm.LLMResponse{Content: "hi"}, requested)
	if out.Model != requested {
		t.Errorf("fallback failed: got %q, want %q", out.Model, requested)
	}
}

// TestLLMResponse_ModelSurvivesJSONRoundTrip guards the wiring that is easiest
// to break silently. LLMResponse has custom MarshalJSON/UnmarshalJSON via an
// llmResponseJSON envelope, so a new field that is added to the struct but not
// to BOTH marshal directions is dropped on the floor with no compile error and
// no test failure — the response would arrive model-less through any path that
// serialises it (cache, queue, cross-process hop).
func TestLLMResponse_ModelSurvivesJSONRoundTrip(t *testing.T) {
	if redMode(t) {
		t.Skip("round-trip wiring did not exist pre-fix; there is no prior behaviour to reproduce")
	}
	const served = "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"

	blob, err := json.Marshal(llm.LLMResponse{Content: "x", Model: served})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(blob), served) {
		t.Fatalf("marshalled payload lost the model field entirely: %s", blob)
	}

	var back llm.LLMResponse
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Model != served {
		t.Errorf("model did not survive the round trip: got %q, want %q — check that BOTH MarshalJSON and UnmarshalJSON carry the field", back.Model, served)
	}
}

// TestOpenAIFacade_StreamingChunksReportServedModel is the guard whose absence
// a re-review proved by deleting the streaming fix and watching the entire
// suite stay green (§11.4.135: the guard must land with the fix, or a future
// refactor silently regresses `stream:true` back to echoing the alias).
//
// It drives the real handler end-to-end — chatCompletions -> streamOpenAIChat
// Completion -> SSE frames — with a provider whose chunks carry a concrete
// model, and asserts the emitted chat.completion.chunk frames name THAT model
// rather than the alias in the request.
func TestOpenAIFacade_StreamingChunksReportServedModel(t *testing.T) {
	const (
		requestedAlias = "default"
		servedModel    = "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"
	)

	fake := &wireFacadeFakeProvider{content: "hi", servedModel: servedModel}
	withFakeResolver(t, fake)

	srv := &Server{}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1/chat/completions", srv.chatCompletions)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		bytes.NewBufferString(`{"model":"`+requestedAlias+`","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	body := w.Body.String()

	// Collect the model field of every emitted chunk frame.
	var models []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			continue
		}
		var frame struct {
			Object string `json:"object"`
			Model  string `json:"model"`
		}
		if err := json.Unmarshal([]byte(payload), &frame); err != nil {
			t.Fatalf("emitted a non-JSON SSE frame %q: %v", payload, err)
		}
		models = append(models, frame.Model)
	}

	if len(models) == 0 {
		t.Fatalf("no chat.completion.chunk frames were emitted; body:\n%s", body)
	}

	if redMode(t) {
		for _, m := range models {
			if m != requestedAlias {
				t.Fatalf("RED_MODE: expected the pre-fix echo %q in every frame, got %q — the streaming defect is not reproducible", requestedAlias, m)
			}
		}
		return
	}

	for i, m := range models {
		if m != servedModel {
			t.Errorf("stream frame %d names %q; it must name the model that ACTUALLY served the request (%q). A stream:true request reporting the requested alias is the CONST-036/037 defect this guard exists to catch.",
				i, m, servedModel)
		}
	}
}

// TestOpenAIFacade_StreamingFallsBackToRequestedModel pins the fallback half of
// the streaming contract: a provider that reports no model must not cause empty
// model fields on the wire.
func TestOpenAIFacade_StreamingFallsBackToRequestedModel(t *testing.T) {
	if redMode(t) {
		t.Skip("streaming fallback behaviour is unchanged by the fix")
	}
	const requested = "llama3.2"

	// servedModel deliberately unset — the shape every provider that does not
	// report a model still produces.
	fake := &wireFacadeFakeProvider{content: "hi"}
	withFakeResolver(t, fake)

	srv := &Server{}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1/chat/completions", srv.chatCompletions)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		bytes.NewBufferString(`{"model":"`+requested+`","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	for _, line := range strings.Split(w.Body.String(), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") || strings.TrimPrefix(line, "data: ") == "[DONE]" {
			continue
		}
		var frame struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &frame); err != nil {
			continue
		}
		if frame.Model != requested {
			t.Errorf("with a silent provider every frame must fall back to the requested model: got %q, want %q", frame.Model, requested)
		}
		if frame.Model == "" {
			t.Error("stream frame emitted an empty model field")
		}
	}
}
