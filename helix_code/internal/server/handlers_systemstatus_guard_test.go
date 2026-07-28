package server

// Standing regression guard for the getSystemStatus nil-database panic
// (§11.4.135 permanent guard, §11.4.115 RED-on-broken-artifact polarity).
//
// Defect (reproduced before the fix): getSystemStatus called
// s.db.HealthCheck() WITHOUT a `s.db != nil` guard. The server is
// constructible without a database (server.New(cfg, nil, rds) — auth and
// every manager are guarded with `if db != nil`). With s.db == nil the
// method call dereferences the nil *Database receiver (to read db.Pool) and
// SIGSEGVs — a nil-RECEIVER panic, distinct from the nil-*pool* path that
// (*Database).HealthCheck itself guards. Sibling endpoints healthCheck and
// getServerInfo already guard `s.db != nil`; getSystemStatus did not.
//
// Fix: report dbStatus "disabled" when s.db == nil, "healthy"/"unhealthy"
// otherwise — matching the getServerInfo `"enabled": s.db != nil` contract.
//
// Polarity switch (§11.4.115): set HELIX_RED_MODE=1 to run the RED
// reproduction. BOTH polarities drive the REAL shipped getSystemStatus handler
// on the SAME nil-db Server; only the assertion flips. RED asserts the
// nil-receiver panic is genuinely present (true on the pre-fix artifact, false
// on the fixed one); DEFAULT (no env) runs the GREEN guard and asserts a clean
// 200 with "database":"disabled" and NO panic.
//
// An earlier revision of this file reproduced the defect against a test-LOCAL
// copy of the unguarded call. A local copy is not part of the artifact, so it
// panicked identically on every build ever made — the RED branch could not
// fail and therefore guarded nothing (§11.4.1). Converted, not deleted:
// deleting a bluff gate removes coverage, converting it creates real coverage.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"dev.helix.code/internal/database"
	"github.com/gin-gonic/gin"
)

func TestGuard_GetSystemStatus_NilDB(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{} // db is nil — server built without a database backend.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)

	if os.Getenv("HELIX_RED_MODE") == "1" {
		// RED reproduction: drive the REAL shipped handler. On the pre-fix
		// artifact its unguarded s.db.HealthCheck() dereferences the nil
		// *Database receiver and panics. If it does NOT panic, the defect is
		// not reproduced and the guard would be blind (§11.4.115 honest
		// boundary) — which is exactly what must happen on a FIXED artifact.
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("RED_MODE: expected the real getSystemStatus to panic with " +
					"a nil-receiver dereference on a nil-db Server, got none — " +
					"the defect did not reproduce")
			}
		}()
		s.getSystemStatus(c)
		t.Fatal("RED_MODE: unreachable — the pre-fix handler should have panicked")
		return
	}

	// GREEN guard (DEFAULT): the real fixed handler must NOT panic and must
	// report the database as disabled with a clean 200.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("getSystemStatus panicked on a nil-db Server: %v", r)
		}
	}()

	s.getSystemStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from getSystemStatus with nil db, got %d (body=%s)",
			w.Code, w.Body.String())
	}

	var body struct {
		Status string `json:"status"`
		System struct {
			Database string `json:"database"`
			API      string `json:"api"`
		} `json:"system"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (body=%s)", err, w.Body.String())
	}
	if body.Status != "success" {
		t.Fatalf("expected status \"success\", got %q", body.Status)
	}
	if body.System.Database != "disabled" {
		t.Fatalf("expected database \"disabled\" for a nil-db server, got %q",
			body.System.Database)
	}
}

// TestGuard_GetSystemStatus_WithDB_StillReports proves the fix did not break
// the database-present path: with a NON-NIL db wired, the handler must take
// the `s.db != nil` branch and surface the REAL HealthCheck verdict, never
// collapse to "disabled" (regression guard for the §11.4.120 reconciliation —
// the new nil-guard must not swallow the real health check).
//
// HXC-153: an earlier revision of this test carried this name while building
// a `&Server{}` with NO database at all and asserting only `w.Code == 200`.
// That re-ran the nil-db case its sibling above already covers, so the
// database-PRESENT path this name advertises had no coverage from it — a
// name-versus-behaviour defect of the same class as HXC-152. The fix is a
// real seam, not a rename: `&database.Database{Pool: nil}` is a genuinely
// non-nil *Database whose HealthCheck returns an error (database.go:307
// guards a nil Pool), so it drives the `s.db != nil` branch end-to-end with
// zero infrastructure. Asserting "unhealthy" — a value the nil-db branch can
// NEVER produce — is what makes this test earn its name: it is falsifiable by
// any change that lets the nil-guard swallow the health check.
//
// Honest boundary (§11.4.6): this covers db-present + HealthCheck-FAILS
// ("unhealthy"). The db-present + HealthCheck-SUCCEEDS ("healthy") arm needs
// a live pool and is exercised by the integration suite against real
// PostgreSQL; it is deliberately NOT claimed here.
func TestGuard_GetSystemStatus_WithDB_StillReports(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Non-nil *Database with no pool: HealthCheck() returns an error, so the
	// db-present branch must report the real verdict "unhealthy".
	s := &Server{db: &database.Database{Pool: nil}}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("getSystemStatus panicked with a db wired: %v", r)
		}
	}()

	s.getSystemStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}

	var body struct {
		Status string `json:"status"`
		System struct {
			Database string `json:"database"`
		} `json:"system"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (body=%s)", err, w.Body.String())
	}
	if body.Status != "success" {
		t.Fatalf("expected status \"success\", got %q", body.Status)
	}
	// The load-bearing assertion: a wired db MUST yield a real health verdict.
	// "disabled" here would mean the nil-guard swallowed the health check.
	if body.System.Database == "disabled" {
		t.Fatalf("db-present path reported \"disabled\" — the nil-guard swallowed " +
			"the real HealthCheck; \"disabled\" must be reachable ONLY when s.db == nil")
	}
	if body.System.Database != "unhealthy" {
		t.Fatalf("expected database \"unhealthy\" for a wired db whose HealthCheck "+
			"fails (nil pool), got %q", body.System.Database)
	}
}
