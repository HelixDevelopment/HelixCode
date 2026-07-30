package main

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// corsMiddleware enforces an origin allowlist for cross-origin browser access.
//
// HXC-194. This middleware previously emitted a blanket
// "Access-Control-Allow-Origin: *" on every response and never looked at the
// request Origin at all, so any web page on any site — loaded by anyone able
// to route to this service — could drive every route and read the responses.
//
// This is NOT the published gin-contrib/cors advisory, and patching or
// upgrading that package would not have touched this code: the server here is
// hand-rolled and this module has never depended on that package (go.mod
// requires gin-gonic/gin only). Closing the advisory would have removed the
// warning while leaving this hole open. Same shape, different defect.
//
// Behaviour now, mirroring the already-hardened CORSMiddleware in
// internal/server/server.go rather than inventing a second dialect:
//
//   - Origin present AND on the allowlist -> echo back that SPECIFIC origin,
//     never "*", plus "Vary: Origin" so a shared cache cannot hand one
//     origin's response to another.
//   - Origin present and NOT on the allowlist -> no Allow-Origin header at
//     all. The browser blocks the caller from reading the response.
//   - No Origin header -> untouched. Non-browser clients (the Go/e2e HTTP
//     clients that actually drive this mock) send no Origin, ignore CORS
//     headers entirely, and are unaffected by the default-deny allowlist.
//
// The advertised methods/headers and the 204 preflight short-circuit are
// preserved exactly as before (§11.4.122): this restricts WHO may read a
// response, it removes no capability.
//
// allowedOrigins comes from MOCK_LLM_ALLOWED_ORIGINS via config.Load and
// defaults to empty, i.e. default-deny. No origin is hardcoded (CONST-045).
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		// A literal "*" is rejected on purpose: it would reintroduce the
		// exact defect through configuration rather than through code.
		if o = strings.TrimSpace(o); o != "" && o != "*" {
			allowed[o] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		if origin := c.Request.Header.Get("Origin"); origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Add("Vary", "Origin")
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
