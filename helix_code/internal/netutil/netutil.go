// Package netutil provides the single, shared host:port composition helper
// used across HelixCode.
//
// # Why this package exists (HXC-185)
//
// An IPv6 literal contains colons. Joining it to a port with
// `fmt.Sprintf("%s:%d", host, port)` produces an address that is BOTH
// unparseable as a URL authority (RFC 3986 §3.2.2 requires brackets) and
// rejected outright by the Go resolver. Captured on go1.26.4:
//
//	net.Dial("tcp", "::1:6379")   -> dial tcp: address ::1:6379: too many colons in address
//	net.Listen("tcp", "::1:0")    -> listen tcp: address ::1:0: too many colons in address
//	net.Dial("tcp", "[::1]:6379") -> connect: connection refused   (address ACCEPTED)
//
// Any host that reaches these call sites from configuration, service
// discovery, or an environment variable can legitimately be a bare IPv6
// literal, so every such join MUST bracket.
//
// # The double-bracket trap
//
// net.JoinHostPort brackets unconditionally whenever the host contains a
// colon — including a host that is ALREADY bracketed:
//
//	net.JoinHostPort("[::1]", "80") == "[[::1]]:80"     // WRONG
//	net.JoinHostPort("::1",   "80") == "[::1]:80"       // right
//
// and the double-bracketed form is rejected just as hard as the unbracketed
// one:
//
//	net.Dial("tcp", "[[::1]]:25")  -> address [[::1]]:25: missing port in address
//	url.Parse("http://[[::1]]:39247/") -> invalid IP-literal
//
// Hosts arrive in BOTH shapes in this codebase: net.SplitHostPort strips the
// brackets ("[::1]" -> "::1") while a URL authority carries them, so callers
// legitimately hold either. JoinHostPort below normalises first, so it is
// safe — and idempotent — for both.
package netutil

import (
	"net"
	"strconv"
	"strings"
)

// UnbracketHost removes one layer of surrounding square brackets from an IPv6
// host literal, so an already-bracketed host can be handed to net.JoinHostPort
// without producing the "[[::1]]" double-bracket form.
//
// It is deliberately conservative: it strips only when the value both starts
// with '[' and ends with ']'. Hostnames, IPv4 literals, and bare IPv6 literals
// pass through untouched, which makes it idempotent.
func UnbracketHost(host string) string {
	if len(host) >= 2 && strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host[1 : len(host)-1]
	}
	return host
}

// JoinHostPortStr joins a host and a string port into an authority that is
// valid for net.Dial, net.Listen, and a URL authority component, bracketing
// IPv6 literals exactly once regardless of whether the caller's host was
// already bracketed.
func JoinHostPortStr(host, port string) string {
	return net.JoinHostPort(UnbracketHost(host), port)
}

// JoinHostPort is the integer-port form of JoinHostPortStr. It is the helper
// that replaces `fmt.Sprintf("%s:%d", host, port)` at every address-composing
// call site.
func JoinHostPort(host string, port int) string {
	return JoinHostPortStr(host, strconv.Itoa(port))
}
