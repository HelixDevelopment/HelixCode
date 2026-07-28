package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/mcp"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// ws_auth_test.go — §11.4.115 RED-baseline + standing GREEN guard for the
// unauthenticated /ws WebSocket endpoint security finding.
//
// THE BUG (RED, captured on the pre-fix artifact): GET /ws
// (internal/server/server.go's s.router.GET("/ws", s.handleWebSocket))
// was registered with ZERO middleware, and
// internal/mcp/server.go's websocket.Upgrader.CheckOrigin unconditionally
// returned true — any client, from any origin, with no credential, could
// complete the MCP WebSocket handshake. Full analysis + STEP-0 fact-check
// (no browser/JS client in this codebase opens /ws — grepped
// web/frontend/**, applications/**, cmd/**) in
// docs/research/07.2026/05_mcp_acp_protocols/WS_ENDPOINT_AUTH_DESIGN.md.
//
// THE GUARD (GREEN, post-fix): the route is gated by
// server.go's s.wsAuthMiddleware() (mirrors wireFacadeAuthMiddleware,
// reuses cfg.Auth.WireFacadeAPIKeys, fails CLOSED on empty config), and
// internal/mcp/server.go's CheckOrigin now validates the handshake's Origin
// header against an explicit allowlist (mcp.newOriginChecker /
// MCPServer.SetAllowedOrigins) instead of unconditionally accepting every
// origin.
//
// Polarity switch RED_WS_AUTH (mirrors wire_facade_auth_test.go's
// RED_WIRE_FACADE_AUTH): default ("0") = standing GREEN guard; "1" =
// reproduce-and-assert-the-defect (exercised once, by hand, against the
// pre-fix source to capture the RED evidence cited in this fix's commit
// message; run against the current, fixed HEAD it necessarily fails, which
// is expected and documents the fix took effect — see §11.4.115).
//
// Anti-bluff: real gin router (srv.setupRoutes()) + the real
// wsAuthMiddleware + the real mcp.MCPServer's Upgrader, exercised
// end-to-end via httptest.NewServer + an actual gorilla/websocket dial —
// not a unit call of the middleware function in isolation and not a mocked
// WS client (§11.4.27(A) unit-test-only mock exception does not apply
// here; CONST-050(A)).

func wsAuthRedMode(t *testing.T) bool {
	t.Helper()
	return redModeFor(t, "RED_WS_AUTH")
}

const wsAuthTestAPIKey = "test-only-ws-auth-key-not-a-real-secret"

// wsAuthFixture builds a real Server with the real route table
// (srv.setupRoutes()), a real mcp.MCPServer (so s.handleWebSocket's
// s.mcp.HandleWebSocket call is exercised against real upgrader logic, not
// a nil-pointer), and an operator-configured WireFacadeAPIKeys list so the
// "valid key passes" half of the guard is meaningful. extraOrigins is
// forwarded to mcp.MCPServer.SetAllowedOrigins exactly as server.New does.
func wsAuthFixture(t *testing.T, extraOrigins []string) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:         "test-secret-key-for-testing-only",
			TokenExpiry:       3600,
			BcryptCost:        4,
			WireFacadeAPIKeys: wsAuthTestAPIKey,
			WSAllowedOrigins:  strings.Join(extraOrigins, ","),
		},
		Logging: config.LoggingConfig{Level: "error"},
	}

	mcpServer := mcp.NewMCPServer()
	mcpServer.SetAllowedOrigins(extraOrigins)

	srv := &Server{
		config: cfg,
		router: gin.New(),
		mcp:    mcpServer,
	}
	srv.setupRoutes()
	return srv
}

// dialWS attempts a WebSocket handshake against ts (an httptest.Server
// wrapping srv.router) at path "/ws", with the given headers. Returns the
// dial error (nil on a successful 101 Switching Protocols upgrade) and, if
// available, the underlying HTTP response (so callers can inspect the
// status code on a rejected handshake).
func dialWS(t *testing.T, ts *httptest.Server, header http.Header) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	dialer := &websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	return dialer.Dial(wsURL, header)
}

// TestWebSocketUnauthenticatedHandshake is the RED/GREEN core assertion
// (§11.4.115 §7.1/§7.2 of the design doc): a handshake with no
// Authorization / x-api-key header and no ticket.
func TestWebSocketUnauthenticatedHandshake(t *testing.T) {
	red := wsAuthRedMode(t)
	srv := wsAuthFixture(t, nil)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	conn, resp, err := dialWS(t, ts, nil)
	if conn != nil {
		defer conn.Close()
	}

	if red {
		// PRE-FIX artifact: the bug — no auth gate of any kind, so an
		// unauthenticated handshake SUCCEEDS (101 Switching Protocols, no
		// error). This is the captured, reproducing proof of the defect,
		// run against the actual broken artifact, not a synthetic
		// scenario.
		require.NoErrorf(t, err, "RED expects the unauth bug PRESENT: no-credential /ws dial must SUCCEED (dial failed instead: %v)", err)
		t.Logf("RED captured: /ws accepted an unauthenticated, no-Origin dial with no rejection of any kind")
	} else {
		// POST-FIX guard: the dial MUST fail — either the HTTP upgrade
		// itself returns 401 (wsAuthMiddleware rejects before Upgrade()),
		// or (not expected for this specific case, since no Origin header
		// is sent) the connection opens and is immediately closed. The
		// design doc's §7.2 accepts either "rejected" shape without
		// over-coupling to the exact mechanic; for Option B (this fix) the
		// pre-upgrade 401 is the actual behavior.
		require.Errorf(t, err, "GREEN: unauthenticated /ws dial must be REJECTED (got a successful upgrade instead)")
		require.NotNilf(t, resp, "GREEN: rejection must carry an HTTP response to inspect (got nil response alongside err=%v)", err)
		if resp != nil {
			require.Equalf(t, http.StatusUnauthorized, resp.StatusCode,
				"GREEN: unauthenticated /ws dial must be rejected 401 (got %d)", resp.StatusCode)
		}
	}
}

// TestWebSocketValidBearerAccepted proves the GREEN fix doesn't over-reach:
// a handshake carrying a Bearer token matching the configured
// WireFacadeAPIKeys list, from an allowed origin, completes the upgrade
// (101 Switching Protocols). GREEN mode only.
func TestWebSocketValidBearerAccepted(t *testing.T) {
	if wsAuthRedMode(t) {
		t.Skip("SKIP-OK: positive-half guard only meaningful post-fix (GREEN mode)")
	}
	srv := wsAuthFixture(t, nil)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+wsAuthTestAPIKey)
	conn, resp, err := dialWS(t, ts, header)
	if conn != nil {
		defer conn.Close()
	}

	require.NoErrorf(t, err, "GREEN: valid-Bearer /ws dial (no Origin header, same as a non-browser MCP SDK client) must be ACCEPTED (dial failed: %v)", err)
	require.NotNilf(t, resp, "expected an HTTP response for a successful upgrade")
	if resp != nil {
		require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	}
}

// TestWebSocketValidXAPIKeyAccepted mirrors the Bearer case for the
// x-api-key header (the Anthropic-native wire convention this codebase
// also accepts, per wireFacadeAuthMiddleware precedent).
func TestWebSocketValidXAPIKeyAccepted(t *testing.T) {
	if wsAuthRedMode(t) {
		t.Skip("SKIP-OK: positive-half guard only meaningful post-fix (GREEN mode)")
	}
	srv := wsAuthFixture(t, nil)
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	header := http.Header{}
	header.Set("x-api-key", wsAuthTestAPIKey)
	conn, resp, err := dialWS(t, ts, header)
	if conn != nil {
		defer conn.Close()
	}

	require.NoErrorf(t, err, "GREEN: valid-x-api-key /ws dial must be ACCEPTED (dial failed: %v)", err)
	require.NotNilf(t, resp, "expected an HTTP response for a successful upgrade")
	if resp != nil {
		require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	}
}

// TestWebSocketNoKeyConfiguredStillRejected proves the fail-closed half of
// the design: even if a caller supplies SOME bearer token, an empty
// cfg.Auth.WireFacadeAPIKeys (the zero-value default in every shipped
// config) means every /ws request is rejected — there is no accidental
// "empty config means open access" fallback.
func TestWebSocketNoKeyConfiguredStillRejected(t *testing.T) {
	if wsAuthRedMode(t) {
		t.Skip("SKIP-OK: fail-closed-by-default guard only meaningful post-fix (GREEN mode)")
	}
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Auth:    config.AuthConfig{JWTSecret: "test-secret-key-for-testing-only", TokenExpiry: 3600, BcryptCost: 4},
		Logging: config.LoggingConfig{Level: "error"},
	}
	mcpServer := mcp.NewMCPServer()
	srv := &Server{config: cfg, router: gin.New(), mcp: mcpServer}
	srv.setupRoutes()
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	header := http.Header{}
	header.Set("Authorization", "Bearer some-caller-supplied-token")
	conn, resp, err := dialWS(t, ts, header)
	if conn != nil {
		defer conn.Close()
	}

	require.Errorf(t, err, "GREEN: /ws must reject EVERY request when no key is configured, even with a bearer token")
	require.NotNilf(t, resp, "GREEN: rejection must carry an HTTP response to inspect")
	if resp != nil {
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}
}

// TestWebSocketCrossOriginRejected proves the CheckOrigin allowlist fix:
// a handshake carrying a VALID Bearer token but a cross-origin Origin
// header that is NOT in the allowlist (and not localhost/same-origin) is
// rejected at the WebSocket upgrade layer — the Origin check is enforced
// independently of Bearer/x-api-key validity, closing the CSWSH finding.
func TestWebSocketCrossOriginRejected(t *testing.T) {
	if wsAuthRedMode(t) {
		t.Skip("SKIP-OK: Origin-allowlist guard only meaningful post-fix (GREEN mode); the pre-fix CheckOrigin unconditionally returned true so this scenario has no RED-mode analogue distinct from TestWebSocketUnauthenticatedHandshake's RED case")
	}
	srv := wsAuthFixture(t, nil) // no extra allowed origins configured
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+wsAuthTestAPIKey)
	header.Set("Origin", "https://evil.example.com")
	conn, resp, err := dialWS(t, ts, header)
	if conn != nil {
		defer conn.Close()
	}

	require.Errorf(t, err, "GREEN: a valid-Bearer /ws dial from a disallowed cross-origin Origin header must be REJECTED")
	require.NotNilf(t, resp, "GREEN: rejection must carry an HTTP response to inspect")
	if resp != nil {
		require.Equalf(t, http.StatusForbidden, resp.StatusCode,
			"GREEN: cross-origin /ws dial must be rejected at the WebSocket upgrade layer (gorilla/websocket's CheckOrigin failure path returns 403) (got %d)", resp.StatusCode)
	}
}

// TestWebSocketAllowlistedCrossOriginAccepted proves the allowlist fix
// doesn't over-reach: a cross-origin request whose Origin IS present in the
// operator-configured cfg.Auth.WSAllowedOrigins list, carrying a valid
// Bearer token, is accepted.
func TestWebSocketAllowlistedCrossOriginAccepted(t *testing.T) {
	if wsAuthRedMode(t) {
		t.Skip("SKIP-OK: Origin-allowlist guard only meaningful post-fix (GREEN mode)")
	}
	const allowedOrigin = "https://app.example.com"
	srv := wsAuthFixture(t, []string{allowedOrigin})
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+wsAuthTestAPIKey)
	header.Set("Origin", allowedOrigin)
	conn, resp, err := dialWS(t, ts, header)
	if conn != nil {
		defer conn.Close()
	}

	require.NoErrorf(t, err, "GREEN: a valid-Bearer /ws dial from an explicitly-allowlisted Origin must be ACCEPTED (dial failed: %v)", err)
	require.NotNilf(t, resp, "expected an HTTP response for a successful upgrade")
	if resp != nil {
		require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	}
}
