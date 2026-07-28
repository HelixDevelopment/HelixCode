package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/auth"
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/project"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// argEqualsUUID reports whether a mock QueryRow argument equals the given UUID.
// The project manager passes owner_id as a uuid.UUID value.
func argEqualsUUID(arg interface{}, want uuid.UUID) bool {
	if u, ok := arg.(uuid.UUID); ok {
		return u == want
	}
	return false
}

// handlers_project_idor_test.go — §11.4.115 RED-baseline-on-the-broken-artifact +
// polarity-switch coverage for the Projects IDOR (insecure-direct-object-reference)
// vulnerability, plus the standing GREEN owner-scoping guard.
//
// THE BUG (RED, captured on the pre-fix artifact): getProject / updateProject /
// deleteProject / getProjectSessions fetch/mutate/delete ANY project by :id with
// NO owner check, while listProjects correctly scopes to the authenticated
// user.ID. So an authenticated user B can read/rename/delete user A's project
// just by knowing (or guessing) its id.
//
// THE GUARD (GREEN, post-fix): cross-user access MUST 404 (existence not leaked),
// and same-owner access MUST succeed.
//
// Anti-bluff posture: the System-Under-Test (gin router + authMiddleware +
// real *project.DatabaseManager + the four handlers) is REAL and unmocked. Only
// the database boundary is mocked — this is a unit test (no integration tag),
// where mocks are permitted (CONST-050(A)). Per-SQL mock expectations make the
// auth user-lookup and the project owner-lookup return distinct, realistic rows,
// so the owner-scoping decision is exercised end-to-end through the real router.
//
// Polarity switch RED_PROJECT_IDOR: default ("" or "0") is the standing GREEN
// guard (cross-user access 404s = fixed). Set to "1" to assert the DEFECT is
// present on the pre-fix artifact (cross-user access SUCCEEDS = bug).
//
// The two roles share ONE test body; the env switch flips the assertion polarity
// (§11.4.115 one-source-two-roles).

const (
	idorUserAEmail = "owner-a@example.test"
	idorUserBEmail = "attacker-b@example.test"
)

// idorEnv reports the polarity: true ⇒ reproduce-and-assert-defect (RED),
// false ⇒ standing GREEN guard.
func idorRedMode(t *testing.T) bool {
	t.Helper()
	v := strings.TrimSpace(getenvDefault("RED_PROJECT_IDOR", "0"))
	return v == "1"
}

// authQuerySQL is the exact SQL GetUserByID issues (auth_db.go).
const authQuerySQL = `
		SELECT id, username, email, display_name, is_active, is_verified, mfa_enabled, last_login, created_at, updated_at
		FROM users
		WHERE id = $1`

// projectGetForUserSQL is the exact SQL DatabaseManager.GetProjectForUser
// issues — the owner-scoped getter the IDOR-fixed handlers call. It scopes by
// owner_id IN THE QUERY (WHERE owner_id = $2), so a cross-user lookup returns
// pgx.ErrNoRows (→ ErrProjectNotFound → 404) with no existence leak.
const projectGetForUserSQL = `
		SELECT id, name, description, owner_id, workspace_path, config, status, created_at, updated_at
		FROM projects
		WHERE id = $1 AND owner_id = $2 AND status = 'active'
	`

// idorFixture builds a real Server (real auth + real project.DatabaseManager)
// over a MockDatabase whose QueryRow is keyed by exact SQL:
//   - the auth user-lookup returns an ACTIVE user matching the requester's id;
//   - the project owner-lookup returns a project owned by ownerID.
func idorFixture(t *testing.T, requesterID, ownerID uuid.UUID, projectID string) *Server {
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

	// Auth user lookup → active user with the requester's id.
	now := time.Now()
	userRow := database.NewMockRowWithValues(
		requesterID,      // id (uuid.UUID)
		"requester",      // username
		"requester@test", // email
		nil,              // display_name (sql.NullString → skipped → zero)
		true,             // is_active
		true,             // is_verified
		false,            // mfa_enabled
		nil,              // last_login (sql.NullTime → skipped)
		now,              // created_at
		now,              // updated_at
	)
	mockDB.On("QueryRow", mock.Anything, authQuerySQL, mock.Anything).Return(userRow)

	// Project row owned by ownerID (9 columns matching GetProject /
	// GetProjectForUser SELECT order).
	pid := uuid.MustParse(projectID)
	cfgMap := map[string]interface{}{"type": "go"}
	projRow := database.NewMockRowWithValues(
		pid,              // id (uuid.UUID)
		"victim-project", // name
		"a project",      // description
		ownerID,          // owner_id (uuid.UUID)
		"/tmp/victim",    // workspace_path
		cfgMap,           // config (map)
		"active",         // status
		now,              // created_at
		now,              // updated_at
	)

	// PRE-FIX (RED): the old handlers called the BARE GetProject (no owner
	// scope), whose SQL is projectGetForUserSQL minus the "owner_id = $2"
	// clause. Match anything that SELECTs from projects but does NOT scope by
	// owner_id, and always return the victim row → that is the IDOR.
	mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.Contains(sql, "FROM projects") &&
			strings.Contains(sql, "WHERE id = $1 AND status = 'active'")
	}), mock.Anything).Return(projRow)

	// POST-FIX (GREEN): the owner-scoped GetProjectForUser SQL carries
	// "owner_id = $2". The query returns the row ONLY when the requester is the
	// owner; a cross-user lookup yields pgx.ErrNoRows (→ 404, no existence
	// leak). We branch on the actual owner arg ($2) passed by the handler.
	noRow := database.NewMockRowWithError(pgx.ErrNoRows)
	mockDB.On("QueryRow", mock.Anything,
		mock.MatchedBy(func(sql string) bool {
			return strings.Contains(sql, "owner_id = $2")
		}),
		mock.MatchedBy(func(args []interface{}) bool {
			// args == []interface{}{projectID(uuid), ownerArg(uuid)}; the row
			// is visible only when the requester owns it.
			return len(args) == 2 && argEqualsUUID(args[1], ownerID)
		}),
	).Return(projRow)
	mockDB.On("QueryRow", mock.Anything,
		mock.MatchedBy(func(sql string) bool {
			return strings.Contains(sql, "owner_id = $2")
		}),
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 2 && !argEqualsUUID(args[1], ownerID)
		}),
	).Return(noRow)

	// UPDATE … RETURNING path (manager_db.go UpdateProject): 9-column row.
	// Mocked so the PRE-FIX update handler (which currently goes straight to
	// UpdateProject with NO owner gate) completes and we can observe it was
	// NOT gated. POST-FIX the owner-scoped getter short-circuits to 404 before
	// this is ever reached.
	updRow := database.NewMockRowWithValues(
		pid,              // id
		"victim-project", // name (RETURNING)
		"a project",      // description
		"/tmp/victim",    // workspace_path
		ownerID,          // owner_id
		now,              // created_at
		now,              // updated_at
		"active",         // status
		cfgMap,           // config
	)
	mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return strings.Contains(sql, "UPDATE projects")
	}), mock.Anything).Return(updRow)

	// DELETE path uses Exec (status='deleted'); allow it so PRE-FIX delete
	// completes ungated.
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	authConfig := auth.AuthConfig{
		JWTSecret:   cfg.Auth.JWTSecret,
		TokenExpiry: time.Hour,
		BcryptCost:  4,
	}
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

// tokenFor mints a valid JWT for the given user id via the real auth service.
func tokenFor(t *testing.T, srv *Server, id uuid.UUID, email string) string {
	t.Helper()
	tok, err := srv.auth.GenerateJWT(&auth.User{ID: id, Username: "u", Email: email})
	require.NoError(t, err)
	return tok
}

func TestProjectIDOR_CrossUserGetProject(t *testing.T) {
	red := idorRedMode(t)

	ownerA := uuid.New()
	attackerB := uuid.New()
	projectID := uuid.New().String()

	// Requester is attacker B; the project is owned by A.
	srv := idorFixture(t, attackerB, ownerA, projectID)
	tok := tokenFor(t, srv, attackerB, idorUserBEmail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	srv.router.ServeHTTP(w, req)

	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)

	if red {
		// PRE-FIX artifact: the bug — B can read A's project (200 + project).
		require.Equalf(t, http.StatusOK, w.Code,
			"RED expects the IDOR bug PRESENT: B reads A's project (got %d, body=%s)", w.Code, w.Body.String())
		require.Contains(t, body, "project")
		t.Logf("RED captured: cross-user getProject SUCCEEDED (IDOR present), status=%d body=%s", w.Code, w.Body.String())
	} else {
		// POST-FIX guard: cross-user read MUST 404, existence NOT leaked.
		require.Equalf(t, http.StatusNotFound, w.Code,
			"GREEN: cross-user getProject must 404 (got %d, body=%s)", w.Code, w.Body.String())
	}
}

func TestProjectIDOR_CrossUserUpdateProject(t *testing.T) {
	red := idorRedMode(t)

	ownerA := uuid.New()
	attackerB := uuid.New()
	projectID := uuid.New().String()

	srv := idorFixture(t, attackerB, ownerA, projectID)
	tok := tokenFor(t, srv, attackerB, idorUserBEmail)

	payload, _ := json.Marshal(map[string]string{"name": "pwned", "description": "owned"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/projects/"+projectID, bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w, req)

	if red {
		// PRE-FIX: B can mutate A's project; UpdateProject runs (it returns the
		// UPDATE RETURNING row — mocked below would 500 only if no owner-gate).
		// The salient defect signal is that the request is NOT 404 (no owner gate).
		require.NotEqualf(t, http.StatusNotFound, w.Code,
			"RED expects NO owner-gate on update: B's request reaches the mutation (got 404 means gate already present)")
		t.Logf("RED captured: cross-user updateProject not gated, status=%d body=%s", w.Code, w.Body.String())
	} else {
		require.Equalf(t, http.StatusNotFound, w.Code,
			"GREEN: cross-user updateProject must 404 (got %d, body=%s)", w.Code, w.Body.String())
	}
}

func TestProjectIDOR_CrossUserDeleteProject(t *testing.T) {
	red := idorRedMode(t)

	ownerA := uuid.New()
	attackerB := uuid.New()
	projectID := uuid.New().String()

	srv := idorFixture(t, attackerB, ownerA, projectID)
	tok := tokenFor(t, srv, attackerB, idorUserBEmail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	srv.router.ServeHTTP(w, req)

	if red {
		require.NotEqualf(t, http.StatusNotFound, w.Code,
			"RED expects NO owner-gate on delete: B's request reaches the deletion (got 404 means gate already present)")
		t.Logf("RED captured: cross-user deleteProject not gated, status=%d body=%s", w.Code, w.Body.String())
	} else {
		require.Equalf(t, http.StatusNotFound, w.Code,
			"GREEN: cross-user deleteProject must 404 (got %d, body=%s)", w.Code, w.Body.String())
	}
}

// TestProjectIDOR_SameOwnerStillWorks is the positive half of the GREEN guard:
// the legitimate owner MUST still be able to read their own project after the
// fix. Runs in GREEN mode only (in RED mode the fix isn't present so this is
// trivially satisfied; it is the regression guard that the fix didn't break the
// happy path).
func TestProjectIDOR_SameOwnerStillWorks(t *testing.T) {
	ownerA := uuid.New()
	projectID := uuid.New().String()

	// Requester == owner.
	srv := idorFixture(t, ownerA, ownerA, projectID)
	tok := tokenFor(t, srv, ownerA, idorUserAEmail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	srv.router.ServeHTTP(w, req)

	require.Equalf(t, http.StatusOK, w.Code,
		"owner reading their own project must 200 (got %d, body=%s)", w.Code, w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Contains(t, body, "project")
}

// getenvDefault is a tiny local helper to keep the polarity switch readable.
func getenvDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
