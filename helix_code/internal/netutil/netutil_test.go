// §11.4.115 polarity-switched guard for the shared address-composition helper
// introduced by HXC-185.
//
// This file guards the HELPER ITSELF. It is not the primary anti-bluff
// evidence for HXC-185 — that lives in the per-package guards
// (`hxc185_ipv6_addr_test.go` under internal/redis, internal/discovery,
// internal/memory, internal/notification, internal/worker, internal/server,
// internal/cognee and tests/testinfra), each of which drives the REAL
// production constructor against a REAL IPv6 loopback listener and asserts the
// listener actually accepted an inbound connection.
//
// The assertions below are pinned to behaviour captured on go1.26.4:
//
//	net.Dial("tcp", "::1:6379")   -> address ::1:6379: too many colons in address
//	net.Dial("tcp", "[[::1]]:25") -> address [[::1]]:25: missing port in address
//	net.Dial("tcp", "[::1]:6379") -> connect: connection refused  (ACCEPTED)
//
// POLARITY SWITCH (§11.4.115):
//
//	RED_MODE=1 — assert the DEFECT is present: the naive
//	             fmt.Sprintf("%s:%d") composition, and a bare
//	             net.JoinHostPort on an already-bracketed host, both yield
//	             addresses the resolver rejects. Passes on any artifact,
//	             because it characterises the stdlib behaviour that motivates
//	             the helper.
//	RED_MODE=0 — (DEFAULT) standing GREEN guard: netutil.JoinHostPort yields
//	             an address the resolver accepts for every host shape.
package netutil

import (
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func hxc185RedMode() bool { return os.Getenv("RED_MODE") == "1" }

// resolverAccepts reports whether the Go resolver accepts addr as a
// syntactically valid TCP address. It deliberately uses the REAL resolver
// (net.ResolveTCPAddr) rather than a string check, so the assertion is about
// what the network stack does, not about what a regex thinks.
func resolverAccepts(addr string) error {
	_, err := net.ResolveTCPAddr("tcp", addr)
	return err
}

func TestJoinHostPort_ProducesResolvableAddress(t *testing.T) {
	cases := []struct {
		name string
		host string
		port int
		// wantAuthority is the exact authority the helper must produce.
		wantAuthority string
	}{
		{"ipv4", "127.0.0.1", 6379, "127.0.0.1:6379"},
		{"hostname", "localhost", 8080, "localhost:8080"},
		{"ipv6_bare_loopback", "::1", 6379, "[::1]:6379"},
		{"ipv6_bracketed_loopback", "[::1]", 6379, "[::1]:6379"},
		{"ipv6_full", "2001:db8::1", 8080, "[2001:db8::1]:8080"},
		{"ipv6_full_bracketed", "[2001:db8::1]", 8080, "[2001:db8::1]:8080"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := JoinHostPort(tc.host, tc.port)

			if hxc185RedMode() {
				// Reproduce the defect the helper replaces: the naive
				// composition every fixed call site used to perform.
				naive := tc.host + ":" + strconv.Itoa(tc.port)
				if !strings.Contains(tc.host, ":") {
					t.Skipf("SKIP-OK HXC-185: %q has no colon, the naive composition is "+
						"already correct for it — nothing to reproduce", tc.host)
				}
				if err := resolverAccepts(naive); err == nil {
					t.Fatalf("RED_MODE=1: expected the naive composition %q to be REJECTED "+
						"by the resolver, but it was accepted — the defect did not reproduce "+
						"on this toolchain (%s)", naive, runtime.Version())
				} else {
					t.Logf("defect reproduced (captured evidence): ResolveTCPAddr(%q) -> %v", naive, err)
				}
				return
			}

			// RED_MODE=0 — standing GREEN guard.
			if got != tc.wantAuthority {
				t.Fatalf("JoinHostPort(%q, %d) = %q, want %q", tc.host, tc.port, got, tc.wantAuthority)
			}
			if err := resolverAccepts(got); err != nil {
				t.Fatalf("JoinHostPort(%q, %d) produced %q which the resolver REJECTS: %v",
					tc.host, tc.port, got, err)
			}
			// Round-trip: the authority must split back to the unbracketed host.
			h, p, err := net.SplitHostPort(got)
			if err != nil {
				t.Fatalf("SplitHostPort(%q) failed: %v", got, err)
			}
			if h != UnbracketHost(tc.host) {
				t.Fatalf("round-trip host = %q, want %q", h, UnbracketHost(tc.host))
			}
			if p != strconv.Itoa(tc.port) {
				t.Fatalf("round-trip port = %q, want %q", p, strconv.Itoa(tc.port))
			}
		})
	}
}

// TestJoinHostPort_NoDoubleBracket pins the trap that makes a bare
// net.JoinHostPort unsafe here: it brackets unconditionally on seeing a colon,
// so an already-bracketed host becomes "[[::1]]" — which the resolver rejects
// just as hard as the unbracketed form.
func TestJoinHostPort_NoDoubleBracket(t *testing.T) {
	const bracketed = "[::1]"

	rawJoin := net.JoinHostPort(bracketed, "25")
	if rawJoin != "[[::1]]:25" {
		t.Fatalf("precondition changed: net.JoinHostPort(%q, \"25\") = %q, expected the "+
			"double-bracket form %q — re-derive this guard against the current stdlib",
			bracketed, rawJoin, "[[::1]]:25")
	}
	if err := resolverAccepts(rawJoin); err == nil {
		t.Fatalf("precondition changed: the resolver now ACCEPTS %q; the double-bracket "+
			"trap this helper defends against no longer exists", rawJoin)
	} else {
		t.Logf("double-bracket trap confirmed: ResolveTCPAddr(%q) -> %v", rawJoin, err)
	}

	if hxc185RedMode() {
		t.Logf("RED_MODE=1: defect characterised above — a bare net.JoinHostPort on an "+
			"already-bracketed host yields the unresolvable %q", rawJoin)
		return
	}

	got := JoinHostPort(bracketed, 25)
	if got != "[::1]:25" {
		t.Fatalf("JoinHostPort(%q, 25) = %q, want %q (exactly one bracket layer)",
			bracketed, got, "[::1]:25")
	}
	if err := resolverAccepts(got); err != nil {
		t.Fatalf("JoinHostPort(%q, 25) produced %q which the resolver REJECTS: %v",
			bracketed, got, err)
	}

	// Idempotence: feeding the helper's own unbracketed output back in must not
	// change the result.
	if again := JoinHostPort(UnbracketHost(bracketed), 25); again != got {
		t.Fatalf("JoinHostPort is not idempotent across bracket shapes: %q vs %q", again, got)
	}
}

func TestUnbracketHost(t *testing.T) {
	cases := map[string]string{
		"[::1]":         "::1",
		"::1":           "::1",
		"[2001:db8::1]": "2001:db8::1",
		"127.0.0.1":     "127.0.0.1",
		"localhost":     "localhost",
		"":              "",
		"[":             "[",
		"[]":            "",
	}
	for in, want := range cases {
		if got := UnbracketHost(in); got != want {
			t.Errorf("UnbracketHost(%q) = %q, want %q", in, got, want)
		}
	}
}
