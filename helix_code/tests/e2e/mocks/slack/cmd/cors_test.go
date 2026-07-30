package main

// HXC-194 — standing regression guard for the hand-rolled CORS middleware.
//
// THE DEFECT (reproduced by this file at RED_MODE=1): the mock Slack service
// built its Gin server by hand instead of using the framework's own CORS
// toolkit, and its middleware unconditionally emitted
// "Access-Control-Allow-Origin: *" while never inspecting the request Origin.
// Any web page on any site, opened by anyone able to route to this service,
// could therefore READ every message and webhook body the system under test
// had posted (GET /api/messages, GET /api/webhooks) and DESTROY that captured
// state (DELETE on both) — a genuine read-and-drive hole, not a theoretical
// header nit.
//
// NOT the published gin-contrib/cors advisory. That advisory concerns the
// framework's own CORS package, which this module has never depended on (see
// go.mod: gin-gonic/gin only). Upgrading or patching that package would have
// silenced the scanner warning while leaving this hole wide open. The two are
// separate problems that merely look alike.
//
// §11.4.115 POLARITY SWITCH. The committed default is RED_MODE=0 — the GREEN
// standing guard asserting the defect is ABSENT, so an ordinary `go test`
// stays green. RED_MODE=1 flips the same source into the reproduction that
// asserts the defect is PRESENT: it PASSES only against the pre-fix artifact
// (wildcard middleware) and MUST FAIL against fixed code.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

// Test-fixture origins. These are assertions about behaviour, not service
// configuration — the service itself hardcodes no origin (CONST-045): its
// allowlist comes from MOCK_SLACK_ALLOWED_ORIGINS.
const (
	permittedOrigin = "https://permitted.example"
	hostileOrigin   = "https://evil.example"
)

func redMode() bool { return os.Getenv("RED_MODE") == "1" }

// newCORSTestRouter builds a router wired with the REAL middleware under test
// and stand-in handlers on the service's real route shapes.
func newCORSTestRouter(t *testing.T, allowed []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(corsMiddleware(allowed))

	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
	r.GET("/health", ok)
	r.POST("/api/chat.postMessage", ok)
	r.GET("/api/messages", ok)
	r.DELETE("/api/messages", ok)
	r.GET("/api/webhooks", ok)
	r.DELETE("/api/webhooks", ok)
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
		if method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", "POST")
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestCORS_HostileOriginIsRejected covers BOTH the simple-request path and the
// preflight (OPTIONS) path, on the read route and the destructive route.
// Checking one and not the other is the exact sibling-miss that lets a hole
// survive a "fix".
func TestCORS_HostileOriginIsRejected(t *testing.T) {
	r := newCORSTestRouter(t, []string{permittedOrigin})

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"SimpleRequest_ReadMessages", http.MethodGet, "/api/messages"},
		{"SimpleRequest_ReadWebhooks", http.MethodGet, "/api/webhooks"},
		{"SimpleRequest_DestructiveClear", http.MethodDelete, "/api/messages"},
		{"SimpleRequest_PostMessage", http.MethodPost, "/api/chat.postMessage"},
		{"Preflight_OPTIONS", http.MethodOptions, "/api/messages"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(t, r, tc.method, tc.path, hostileOrigin)
			got := w.Header().Get("Access-Control-Allow-Origin")

			if redMode() {
				if got != "*" {
					t.Fatalf("RED_MODE=1 expected to REPRODUCE the defect on the pre-fix artifact: "+
						"wanted Access-Control-Allow-Origin=%q for hostile origin %q, got %q. "+
						"Against fixed code this failure is CORRECT.", "*", hostileOrigin, got)
				}
				t.Logf("RED reproduced: %s %s with Origin:%s -> Access-Control-Allow-Origin=%q",
					tc.method, tc.path, hostileOrigin, got)
				return
			}

			if got != "" {
				t.Errorf("hostile origin %q must receive NO Access-Control-Allow-Origin header "+
					"(default-deny); got %q", hostileOrigin, got)
			}
			if got == "*" {
				t.Errorf("wildcard Access-Control-Allow-Origin must never be emitted")
			}
		})
	}
}

// TestCORS_PermittedOriginIsAccepted is the positive half. A middleware that
// blocks everything is not a fix, and a guard that only tests rejection would
// not notice.
func TestCORS_PermittedOriginIsAccepted(t *testing.T) {
	if redMode() {
		t.Skip("RED_MODE=1 reproduces the defect only; the positive case is asserted by the GREEN guard")
	}

	r := newCORSTestRouter(t, []string{permittedOrigin})

	for _, tc := range []struct {
		name       string
		method     string
		wantStatus int
	}{
		{"SimpleRequest", http.MethodGet, http.StatusOK},
		{"Preflight", http.MethodOptions, http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(t, r, tc.method, "/api/messages", permittedOrigin)

			if got := w.Header().Get("Access-Control-Allow-Origin"); got != permittedOrigin {
				t.Errorf("permitted origin must be echoed back verbatim: want %q, got %q",
					permittedOrigin, got)
			}
			if got := w.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary: Origin must be set so shared caches cannot serve one origin's "+
					"response to another; got %q", got)
			}
			if w.Code != tc.wantStatus {
				t.Errorf("status: want %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

// TestCORS_NoOriginHeaderIsUnaffected pins the compatibility contract: the
// non-browser Go clients that actually drive this mock send no Origin header
// and must keep working with an empty allowlist (the shipped default).
func TestCORS_NoOriginHeaderIsUnaffected(t *testing.T) {
	if redMode() {
		t.Skip("RED_MODE=1 reproduces the defect only")
	}

	r := newCORSTestRouter(t, nil) // default-deny allowlist

	w := doRequest(t, r, http.MethodPost, "/api/chat.postMessage", "")
	if w.Code != http.StatusOK {
		t.Errorf("a non-browser client sending no Origin must be served normally under the "+
			"default-deny allowlist: want 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("no Origin request must get no Access-Control-Allow-Origin; got %q", got)
	}
}

// TestCORS_NeverEmitsWildcard is the blanket invariant: no allowlist
// configuration, however odd, may produce "*".
func TestCORS_NeverEmitsWildcard(t *testing.T) {
	if redMode() {
		t.Skip("RED_MODE=1 reproduces the defect only")
	}

	for _, allowed := range [][]string{
		nil,
		{},
		{""},
		{"*"},
		{permittedOrigin},
		{permittedOrigin, hostileOrigin},
	} {
		r := newCORSTestRouter(t, allowed)
		for _, origin := range []string{hostileOrigin, permittedOrigin, "*", ""} {
			for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodOptions} {
				w := doRequest(t, r, method, "/api/messages", origin)
				if got := w.Header().Get("Access-Control-Allow-Origin"); got == "*" {
					t.Errorf("wildcard emitted: allowlist=%v origin=%q method=%s", allowed, origin, method)
				}
			}
		}
	}
}

// TestCORS_PreservedCapabilities guards §11.4.122: the hardening restricts the
// origin check only — it must not strip the methods/headers advertisement or
// change the preflight status the service already returned.
func TestCORS_PreservedCapabilities(t *testing.T) {
	if redMode() {
		t.Skip("RED_MODE=1 reproduces the defect only")
	}

	r := newCORSTestRouter(t, []string{permittedOrigin})
	w := doRequest(t, r, http.MethodOptions, "/api/messages", permittedOrigin)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight must still short-circuit with 204; got %d", w.Code)
	}
	for _, want := range []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"} {
		if !contains(w.Header().Get("Access-Control-Allow-Methods"), want) {
			t.Errorf("Access-Control-Allow-Methods lost %q: got %q",
				want, w.Header().Get("Access-Control-Allow-Methods"))
		}
	}
	for _, want := range []string{"Content-Type", "Authorization"} {
		if !contains(w.Header().Get("Access-Control-Allow-Headers"), want) {
			t.Errorf("Access-Control-Allow-Headers lost %q: got %q",
				want, w.Header().Get("Access-Control-Allow-Headers"))
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
