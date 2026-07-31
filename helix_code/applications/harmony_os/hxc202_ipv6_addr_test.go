// HXC-202 — sibling miss of HXC-185: the HarmonyOS client composed its API
// base URL from cfg.Server.Address + cfg.Server.Port with a plain
// `fmt.Sprintf("http://%s:%d", ...)`, while internal/server/server.go binds the
// SAME two settings through netutil.JoinHostPort. With an IPv6 address
// configured the server therefore starts correctly while this client builds an
// authority it can never reach — a failure that presents as "the server is
// down" while the server is running perfectly.
//
// # Polarity switch (§11.4.115)
//
// One source, two roles, selected by the RED_MODE environment variable:
//
//	RED_MODE=1  reproduce-and-assert-the-defect-is-PRESENT (the RED baseline;
//	            PASSes only on the pre-fix artifact)
//	RED_MODE=0  assert-the-defect-is-ABSENT (the standing GREEN regression
//	            guard; PASSes only on the fixed artifact)  [default]
//
// Both roles run the REAL production function apiServerURL — no test re-implements
// the join, so neither role is a replica.
//
// # Negative cases are load-bearing
//
// A guard that only checked IPv6 would not catch the opposite defect: applying a
// host:port join to a value that is already a full URL, or double-bracketing a
// host that already carries brackets. Both are asserted here, and the IPv4 /
// hostname cases are asserted BYTE-FOR-BYTE identical to the pre-fix output so a
// "fix" that perturbs a working address fails this guard.
package main

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"dev.helix.code/internal/config"
)

// redMode reports whether the guard runs in reproduce-the-defect mode.
func redMode(t *testing.T) bool {
	t.Helper()
	return os.Getenv("RED_MODE") == "1"
}

// cfgWithServer builds a config carrying only the two settings under test.
func cfgWithServer(addr string, port int) *config.Config {
	cfg := &config.Config{}
	cfg.Server.Address = addr
	cfg.Server.Port = port
	return cfg
}

// TestHXC202_HarmonyAPIServerURL_IPv6IsBracketed is the positive case: a bare
// IPv6 literal MUST produce a parseable authority whose host round-trips.
func TestHXC202_HarmonyAPIServerURL_IPv6IsBracketed(t *testing.T) {
	got := apiServerURL(cfgWithServer("::1", 8080))

	if redMode(t) {
		// RED: the pre-fix artifact emits the unbracketed, unusable form.
		if got != "http://::1:8080" {
			t.Fatalf("RED_MODE=1 expected the pre-fix unbracketed authority "+
				"%q, got %q — the defect is not present on this artifact, so "+
				"this is not a valid RED baseline", "http://::1:8080", got)
		}
		if _, err := url.Parse(got); err == nil {
			t.Fatalf("RED_MODE=1 expected url.Parse to REJECT the pre-fix "+
				"authority %q, but it parsed — the defect is not reproduced", got)
		}
		t.Logf("RED reproduced on the pre-fix artifact: apiServerURL -> %q "+
			"(url.Parse rejects it)", got)
		return
	}

	// GREEN: the fixed artifact emits a bracketed, parseable authority.
	const want = "http://[::1]:8080"
	if got != want {
		t.Fatalf("apiServerURL(::1, 8080) = %q, want %q — an IPv6 literal must "+
			"be bracketed or the client can never reach the server", got, want)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse(%q) failed: %v — the composed base URL is unusable", got, err)
	}
	if u.Hostname() != "::1" {
		t.Fatalf("url.Parse(%q).Hostname() = %q, want %q", got, u.Hostname(), "::1")
	}
	if u.Port() != "8080" {
		t.Fatalf("url.Parse(%q).Port() = %q, want %q", got, u.Port(), "8080")
	}
}

// TestHXC202_HarmonyAPIServerURL_NegativeCases pins everything that MUST NOT
// change. These run identically in both polarities: the pre-fix and fixed
// artifacts agree here, and that agreement is the point — the repair is scoped
// to the IPv6 case alone.
func TestHXC202_HarmonyAPIServerURL_NegativeCases(t *testing.T) {
	tests := []struct {
		name string
		addr string
		port int
		want string
		why  string
	}{
		{
			name: "ipv4_literal_unchanged",
			addr: "127.0.0.1",
			port: 8080,
			want: "http://127.0.0.1:8080",
			why:  "an IPv4 literal carries no colon and must pass through untouched",
		},
		{
			name: "bind_all_ipv4_unchanged",
			addr: "0.0.0.0",
			port: 8080,
			want: "http://0.0.0.0:8080",
			why:  "the shipped dev/prod default must not be perturbed",
		},
		{
			name: "hostname_unchanged",
			addr: "helix.internal",
			port: 9090,
			want: "http://helix.internal:9090",
			why:  "a hostname must never be bracketed",
		},
		{
			name: "already_bracketed_not_double_bracketed",
			addr: "[::1]",
			port: 8080,
			want: "http://[::1]:8080",
			why: "net.JoinHostPort brackets unconditionally on seeing a colon, so " +
				"an already-bracketed host would become [[::1]] — rejected just as " +
				"hard as the unbracketed form",
		},
		{
			name: "full_ipv6_literal_bracketed_once",
			addr: "2001:db8::1",
			port: 443,
			want: "http://[2001:db8::1]:443",
			why:  "a routable IPv6 literal is bracketed exactly once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiServerURL(cfgWithServer(tt.addr, tt.port))

			// The already-bracketed and full-IPv6 rows are repaired cases: they
			// differ between artifacts, so only assert them in GREEN.
			isRepaired := strings.Contains(tt.want, "[")
			if redMode(t) && isRepaired {
				t.Skipf("SKIP-OK: RED_MODE=1 — %q is a repaired case, asserted "+
					"in the GREEN polarity", tt.name)
			}

			if got != tt.want {
				t.Fatalf("apiServerURL(%q, %d) = %q, want %q — %s",
					tt.addr, tt.port, got, tt.want, tt.why)
			}
			if _, err := url.Parse(got); err != nil {
				t.Fatalf("url.Parse(%q) failed: %v", got, err)
			}
		})
	}
}

// TestHXC202_HarmonyAPIServerURL_DefaultIsAFullURLNotAHost guards the classification
// mistake that is the mirror image of this bug: defaultAPIServerURL is a complete
// URL, not a host. Feeding a full URL to a host:port join yields
// "[http://localhost:8080]:9000" and BREAKS working code. The fallback must be
// returned verbatim.
func TestHXC202_HarmonyAPIServerURL_DefaultIsAFullURLNotAHost(t *testing.T) {
	for _, tt := range []struct {
		name string
		cfg  *config.Config
	}{
		{name: "nil_config", cfg: nil},
		{name: "empty_address", cfg: cfgWithServer("", 8080)},
		{name: "zero_port", cfg: cfgWithServer("::1", 0)},
		{name: "negative_port", cfg: cfgWithServer("::1", -1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := apiServerURL(tt.cfg)
			if got != defaultAPIServerURL {
				t.Fatalf("apiServerURL = %q, want the verbatim default %q",
					got, defaultAPIServerURL)
			}
			if strings.Contains(got, "[http") {
				t.Fatalf("apiServerURL = %q — the default was fed through a "+
					"host:port join, which corrupts a value that is already a "+
					"full URL", got)
			}
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("url.Parse(%q) failed: %v", got, err)
			}
			if u.Scheme != "http" || u.Hostname() != "localhost" || u.Port() != "8080" {
				t.Fatalf("default URL %q did not round-trip: scheme=%q host=%q port=%q",
					got, u.Scheme, u.Hostname(), u.Port())
			}
		})
	}
}
