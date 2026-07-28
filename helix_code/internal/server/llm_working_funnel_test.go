package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/verifier"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// §11.4.115 polarity for the server-side working-model funnel wiring guard is
// carried by the package-shared redMode(t) helper
// (llm_generate_regression_test.go) — ONE polarity shape for the whole package:
//
//	RED_MODE=0 (default): the standing GREEN guard — the wired handler runs
//	            GetWorkingModels with the llm.PresentProviderNames()
//	            key-presence gate, so only the single working+key-present model
//	            survives end-to-end over HTTP.
//	RED_MODE=1: reproduce the not-wired defect — the pre-fix handler called
//	            GetVerifiedModels directly, so the SAME real handler leaked
//	            failed / pending / sub-threshold / no-key models to the client.
//
// Both polarities drive the REAL shipped handler over the SAME served response;
// only the assertion flips (§11.4.115). A RED branch that asserted against the
// raw adapter instead would pass on every artifact ever built — the handler is
// what the fix changed, so the handler is what the RED branch must observe.

// newFunnelCatalogServer serves the same mixed catalog the verifier-layer test
// uses: one working anthropic model, one sub-threshold, one failed, one
// pending, and one fully-verified openai model the caller has NO key for.
func newFunnelCatalogServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		models := []*verifier.VerifiedModel{
			{ID: "claude-good", Provider: "anthropic", Verified: true,
				VerificationStatus: "verified", OverallScore: 8.0},
			{ID: "claude-lowscore", Provider: "anthropic", Verified: true,
				VerificationStatus: "verified", OverallScore: 4.0},
			{ID: "claude-failed", Provider: "anthropic", Verified: false,
				VerificationStatus: "failed", OverallScore: 9.0},
			{ID: "claude-pending", Provider: "anthropic", Verified: false,
				VerificationStatus: "pending", OverallScore: 9.0},
			{ID: "gpt-good", Provider: "openai", Verified: true,
				VerificationStatus: "verified", OverallScore: 9.5},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models)
	}))
}

func newFunnelAdapter(t *testing.T, url string) *verifier.Adapter {
	t.Helper()
	client := verifier.NewClient(url, "", 0)
	health := verifier.NewHealthMonitor(5, 3, 60*time.Second)
	cache := verifier.NewCache(5*time.Minute, nil)
	cfg := &verifier.AdapterConfig{Enabled: true} // default MinAcceptableScore -> 6.0
	return verifier.NewAdapter(client, cache, health, cfg)
}

// clearProviderEnv unsets every recognized provider alias so the test starts
// from a hermetic key-presence baseline.
func clearProviderEnv(t *testing.T) {
	t.Helper()
	for _, aliases := range llm.ProviderEnvAliases() {
		for _, a := range aliases {
			if _, ok := os.LookupEnv(a); ok {
				t.Setenv(a, "")
			}
		}
	}
}

// TestServerListLLMModels_WorkingFunnelEndToEnd proves the server's
// /api/v1/llm/models handler runs the working-model funnel end-to-end:
// only key-present ∧ Verified ∧ status=="verified" ∧ score>=min models are
// returned over real HTTP. A no-key provider's model is hidden.
func TestServerListLLMModels_WorkingFunnelEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)
	catalog := newFunnelCatalogServer(t)
	defer catalog.Close()

	clearProviderEnv(t)
	// anthropic key PRESENT; openai key ABSENT.
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-realvalue-1234567890")

	adapter := newFunnelAdapter(t, catalog.URL)
	srv := &Server{
		verifierResult: &verifier.BootstrapResult{Adapter: adapter},
	}

	router := gin.New()
	router.GET("/api/v1/llm/models", srv.listLLMModels)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/llm/models", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	served := idSetFromModels(t, resp["models"])

	if redMode(t) {
		// RED: on the PRE-FIX artifact listLLMModels called GetVerifiedModels
		// directly, so the handler's OWN response leaked every unusable model.
		// This drives the REAL shipped handler (same request, same response as
		// GREEN) and asserts the leak is present — false on the fixed artifact.
		require.Equal(t, "success", resp["status"])
		require.Equal(t, "verifier", resp["source"])
		assert.True(t, served["claude-failed"],
			"RED: the pre-fix handler leaks the FAILED model to the client")
		assert.True(t, served["claude-lowscore"],
			"RED: the pre-fix handler leaks the sub-threshold model to the client")
		assert.True(t, served["claude-pending"],
			"RED: the pre-fix handler leaks the PENDING model to the client")
		assert.True(t, served["gpt-good"],
			"RED: the pre-fix handler leaks the no-key openai model to the client")
		assert.Len(t, served, 5, "RED: the pre-fix handler serves the whole unfiltered catalog")
		return
	}

	// GREEN: the wired handler emits only the working+key-present model.
	require.Equal(t, "success", resp["status"])
	require.Equal(t, "verifier", resp["source"], "must be sourced from the working-model funnel, not fallback")
	assert.True(t, served["claude-good"], "the single working+key-present model must be listed")
	assert.False(t, served["claude-lowscore"], "sub-threshold model must be hidden (D-4)")
	assert.False(t, served["claude-failed"], "failed model must be hidden (D-2/D-4)")
	assert.False(t, served["claude-pending"], "pending model must be hidden (D-2/D-4)")
	assert.False(t, served["gpt-good"], "no-key provider model must be hidden (key-gate)")
	assert.Len(t, served, 1)
}

// TestServerListLLMProviders_WorkingFunnelEndToEnd proves the
// /api/v1/llm/providers handler hides providers with no key end-to-end:
// only providers backed by ≥1 working+key-present model appear.
func TestServerListLLMProviders_WorkingFunnelEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)
	catalog := newFunnelCatalogServer(t)
	defer catalog.Close()

	clearProviderEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-realvalue-1234567890") // openai absent

	adapter := newFunnelAdapter(t, catalog.URL)
	srv := &Server{verifierResult: &verifier.BootstrapResult{Adapter: adapter}}

	router := gin.New()
	router.GET("/api/v1/llm/providers", srv.listLLMProviders)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/llm/providers", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	require.Equal(t, "success", resp["status"])
	require.Equal(t, "verifier", resp["source"])
	provs, ok := resp["providers"].([]interface{})
	require.True(t, ok, "providers field must be an array")
	names := map[string]bool{}
	for _, p := range provs {
		pm := p.(map[string]interface{})
		if id, ok := pm["id"].(string); ok {
			names[id] = true
		}
	}

	if redMode(t) {
		// RED: on the PRE-FIX artifact listLLMProviders called
		// GetVerifiedModels, so a provider the caller has NO key for was still
		// derived from the catalog and listed as available. Drives the REAL
		// shipped handler; false on the fixed artifact. Previously this case
		// merely t.Skip()ed, so the key-gate had NO reproduction at all.
		assert.True(t, names["openai"],
			"RED: the pre-fix handler lists openai despite NO openai key being present")
		return
	}

	assert.True(t, names["anthropic"], "anthropic key-present + working model -> provider listed")
	assert.False(t, names["openai"], "openai has NO key -> provider hidden (key-gate, anti-bluff)")
}

func idSetFromModels(t *testing.T, v interface{}) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	arr, ok := v.([]interface{})
	if !ok {
		return out
	}
	for _, m := range arr {
		mm, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := mm["id"].(string); ok {
			out[id] = true
		}
	}
	return out
}
