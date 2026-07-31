// This file is deliberately UNTAGGED so it compiles under both the GUI build
// (`//go:build !nogui`, main.go) and the headless build (`//go:build nogui`,
// main_nogui.go). The GUI build needs X11/GL system libraries that CI hosts do
// not carry, so anything that lives only in main.go cannot be compiled — let
// alone tested — on such a host. Keeping the address composition here is what
// makes it verifiable at all.

package main

import (
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/netutil"
)

// defaultAPIServerURL is the API base URL used when the configuration does not
// name a server address and port. It is a complete URL literal, NOT a host, and
// must never be fed to a host:port join.
const defaultAPIServerURL = "http://localhost:8080"

// apiServerURL composes the base URL the HarmonyOS client uses to reach the
// HelixCode API server, from the same two settings the server itself binds:
// cfg.Server.Address and cfg.Server.Port.
//
// # HXC-202 (sibling miss of HXC-185)
//
// cfg.Server.Address is a bare HOST — internal/server/server.go binds that very
// field through netutil.JoinHostPort — so it can legitimately be a bare IPv6
// literal. An IPv6 literal contains colons, so the previous
// `fmt.Sprintf("http://%s:%d", ...)` produced an authority that is invalid per
// RFC 3986 §3.2.2 and that net/url rejects outright:
//
//	"http://::1:8080"   -> url.Parse error, unreachable   (WRONG)
//	"http://[::1]:8080" -> valid                          (right)
//
// The failure mode is maximally misleading: the server binds correctly via the
// shared helper while this client builds an address it can never reach, so the
// operator sees a server that "is down" while it is running perfectly.
//
// netutil.JoinHostPort is the ONE shared join (internal/netutil); it also
// absorbs the double-bracket trap, so a host that already arrives bracketed is
// not turned into "[[::1]]". Hostnames and IPv4 literals pass through
// byte-for-byte unchanged, which keeps this a strict repair of the IPv6 case.
//
// The default is returned VERBATIM: it is a complete URL, not a host, and
// joining a port onto it would yield "[http://localhost:8080]:9000".
func apiServerURL(cfg *config.Config) string {
	if cfg == nil {
		return defaultAPIServerURL
	}
	if cfg.Server.Address == "" || cfg.Server.Port <= 0 {
		return defaultAPIServerURL
	}
	return "http://" + netutil.JoinHostPort(cfg.Server.Address, cfg.Server.Port)
}
