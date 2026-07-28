package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dev.helix.code/internal/auth"
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/project"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// llm_auth_guard_test.go — §11.4.115 RED-baseline + standing GREEN guard for the
// unauthenticated-LLM-cost-endpoints vulnerability.
//
// THE BUG (RED, captured on the pre-fix artifact): POST /api/v1/llm/generate,
// /api/v1/llm/stream and /api/v1/specify are registered with NO auth middleware,
// so an UNAUTHENTICATED caller (no Authorization header) reaches the handler and
// triggers real, paid LLM-provider calls.
//
// THE GUARD (GREEN, post-fix): a no-token request to any of the three endpoints
// MUST be rejected 401 by authMiddleware BEFORE the handler runs; a valid-token
// request MUST pass the middleware and reach the handler (NOT 401).
//
// Polarity switch RED_LLM_AUTH: default ("0") = standing GREEN guard;
// "1" = reproduce-and-assert-the-defect on the pre-fix artifact.
//
// Anti-bluff: real gin router + real authMiddleware + real auth service. The DB
// boundary is mocked (unit test, mocks permitted per CONST-050(A)); the auth
// decision is exercised end-to-end through the real router.

func llmAuthRedMode(t *testing.T) bool {
	t.Helper()
	return redModeFor(t, "RED_LLM_AUTH")
}

// llmAuthFixture builds a real Server whose auth user-lookup returns an ACTIVE
// user for `requesterID`, so a valid token passes VerifyJWTWithDB.
func llmAuthFixture(t *testing.T, requesterID uuid.UUID) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:   "test-secret-key-for-testing-only",
			TokenExpiry: 3600,
			BcryptCost:  4,
		},
		Logging: config.LoggingConfig{Level: "error"},
	}

	mockDB := database.NewMockDatabase()
	now := time.Now()
	userRow := database.NewMockRowWithValues(
		requesterID, "requester", "requester@test", nil,
		true, true, false, nil, now, now,
	)
	mockDB.On("QueryRow", mock.Anything, authQuerySQL, mock.Anything).Return(userRow)

	authConfig := auth.AuthConfig{JWTSecret: cfg.Auth.JWTSecret, TokenExpiry: time.Hour, BcryptCost: 4}
	authService := auth.NewAuthService(authConfig, auth.NewAuthDB(mockDB))

	srv := &Server{
		config:         cfg,
		auth:           authService,
		projectManager: project.NewDatabaseManager(mockDB),
		router:         gin.New(),
	}
	srv.setupRoutes()
	return srv
}

// llmCostEndpoints are the three paid surfaces that MUST require auth.
var llmCostEndpoints = []struct {
	name string
	path string
	body string
}{
	{"generate", "/api/v1/llm/generate", `{"prompt":"hi","model":"llama3.2"}`},
	{"stream", "/api/v1/llm/stream", `{"prompt":"hi","model":"llama3.2"}`},
	{"specify", "/api/v1/specify", `{"prompt":"build a todo app"}`},
}

func TestLLMCostEndpoints_NoTokenRejected(t *testing.T) {
	red := llmAuthRedMode(t)
	srv := llmAuthFixture(t, uuid.New())

	for _, ep := range llmCostEndpoints {
		t.Run(ep.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Content-Type", "application/json")
			// NO Authorization header — the whole point.
			srv.router.ServeHTTP(w, req)

			if red {
				// PRE-FIX artifact: the bug — no auth gate, so the request is
				// NOT rejected 401 (it falls through to the handler and would
				// hit a real provider).
				require.NotEqualf(t, http.StatusUnauthorized, w.Code,
					"RED expects the unauth bug PRESENT on %s: no-token request must NOT be 401 (got %d, body=%s)",
					ep.path, w.Code, w.Body.String())
				t.Logf("RED captured: %s served WITHOUT a token, status=%d body=%s", ep.path, w.Code, w.Body.String())
			} else {
				// POST-FIX guard: middleware rejects the no-token request 401
				// BEFORE the handler runs.
				require.Equalf(t, http.StatusUnauthorized, w.Code,
					"GREEN: %s must reject a no-token request 401 (got %d, body=%s)",
					ep.path, w.Code, w.Body.String())
				var body map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				require.Equal(t, "error", body["status"])
			}
		})
	}
}

// TestLLMCostEndpoints_ValidTokenPassesMiddleware proves the GREEN fix doesn't
// over-reach: a request carrying a VALID token passes authMiddleware and reaches
// the handler (the handler then fails for its own reasons — no reachable
// provider — but crucially NOT with a 401). This is the positive half of the
// guard. GREEN-only (in RED mode there's no middleware so it's trivially true).
func TestLLMCostEndpoints_ValidTokenPassesMiddleware(t *testing.T) {
	if llmAuthRedMode(t) {
		t.Skip("SKIP-OK: positive-half guard only meaningful post-fix (GREEN mode)")
	}
	requester := uuid.New()
	srv := llmAuthFixture(t, requester)
	tok, err := srv.auth.GenerateJWT(&auth.User{ID: requester, Username: "u", Email: "u@test"})
	require.NoError(t, err)

	for _, ep := range llmCostEndpoints {
		t.Run(ep.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tok)
			srv.router.ServeHTTP(w, req)

			require.NotEqualf(t, http.StatusUnauthorized, w.Code,
				"GREEN: valid-token request to %s must pass the middleware (got 401, body=%s)",
				ep.path, w.Body.String())
		})
	}
}
