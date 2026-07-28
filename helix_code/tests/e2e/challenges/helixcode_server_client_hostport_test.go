package challenges

import (
	"net"
	"net/url"
	"strconv"
	"testing"
)

// helixcode_server_client_hostport_test.go — standing regression guard
// (§11.4.135) for the unbracketed IPv6 host:port join defect in
// helixCodeServerBaseURL.
//
// RED baseline (§11.4.115, captured as FACT before the fix): the pre-fix
// implementation built its base URL with
//
//	fmt.Sprintf("http://%s:%d", e.config.HelixCodeHost, e.config.HelixCodePort)
//
// which concatenates an IPv6 literal without the RFC 3986 brackets the URL
// authority component requires. Go's net/http/httptest.newLocalListener falls
// back to "[::1]:0" whenever it cannot bind IPv4 loopback; the test helper
// newHelixCodeTestConfig (executor_rest_server_test.go) derives
// ChallengeConfig.HelixCodeHost from that URL via net.SplitHostPort, which
// strips the brackets and yields the bare "::1". The unbracketed join then
// produced "http://::1:39929/health", which url.Parse rejects with:
//
//	parse "http://::1:39929/health": invalid port "::1:39929" after host
//
// That failure is captured in qa-results/full_retest/helixcode_unit_20260727T115828Z.log
// (TestExecuteREST_DrivesRealHelixCodeServer); the same defect recurs with
// other ephemeral ports in qa-results/full_retest/helix_code_inner_20260727T133220Z.log.
//
// GREEN (post-fix, this guard): helixCodeServerBaseURL uses net.JoinHostPort,
// which brackets IPv6 literals correctly, so every host form below yields a
// URL that url.Parse accepts and whose authority round-trips exactly.
//
// This guard is fully deterministic and performs ZERO network and ZERO
// filesystem I/O: helixCodeServerBaseURL reads nothing but e.config, so the
// executor is constructed directly rather than via NewChallengeExecutor
// (which would read api-keys.yaml from the host and make the result depend on
// operator-local state).

func TestHelixCodeServerBaseURL_BracketsIPv6(t *testing.T) {
	// The port the captured failure used, so the historical bad string
	// ("http://::1:39929") is reproduced exactly under the paired mutation.
	const port = 39929
	portStr := strconv.Itoa(port)

	tests := []struct {
		name string
		// configHost is what lands in ChallengeConfig.HelixCodeHost.
		configHost string
		// wantHostname is url.URL.Hostname(), which is always the
		// UNbracketed form regardless of how the authority was encoded.
		wantHostname string
	}{
		{"ipv4_loopback", "127.0.0.1", "127.0.0.1"},
		{"dns_hostname", "localhost", "localhost"},
		// The exact defect input: httptest's [::1]:0 fallback, run through
		// net.SplitHostPort by newHelixCodeTestConfig, arrives unbracketed.
		{"ipv6_loopback_unbracketed", "::1", "::1"},
		{"ipv6_linklocal_unbracketed", "fe80::1", "fe80::1"},
		// A hand-written JSON config plausibly carries the URL-style
		// bracketed form, which must NOT be double-bracketed.
		{"ipv6_loopback_bracketed", "[::1]", "::1"},
		{"ipv6_linklocal_bracketed", "[fe80::1]", "fe80::1"},
	}

	// Every real caller appends a path to the base URL, so this guard parses
	// exactly what the production callers parse:
	// checkHelixCodeServerReachable -> /health,
	// callHelixCodeGenerate -> /api/v1/llm/generate.
	paths := []string{"/health", "/api/v1/llm/generate"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultChallengeConfig()
			config.HelixCodeHost = tt.configHost
			config.HelixCodePort = port

			e := &ChallengeExecutor{config: config}

			base := e.helixCodeServerBaseURL()

			for _, path := range paths {
				full := base + path

				u, err := url.Parse(full)
				if err != nil {
					t.Fatalf("url.Parse(%q) failed for HelixCodeHost=%q: %v\n"+
						"the base URL must bracket IPv6 literals (use net.JoinHostPort)",
						full, tt.configHost, err)
				}

				if u.Scheme != "http" {
					t.Errorf("HelixCodeHost=%q: url %q parsed scheme = %q, want %q",
						tt.configHost, full, u.Scheme, "http")
				}

				// Hostname() strips brackets, so this asserts the address
				// survived the join intact — neither mangled nor
				// double-bracketed.
				if got := u.Hostname(); got != tt.wantHostname {
					t.Errorf("HelixCodeHost=%q: url %q parsed hostname = %q, want %q",
						tt.configHost, full, got, tt.wantHostname)
				}

				// The historical failure mode was precisely that the port was
				// unparseable ("invalid port \"::1:39929\" after host").
				if got := u.Port(); got != portStr {
					t.Errorf("HelixCodeHost=%q: url %q parsed port = %q, want %q",
						tt.configHost, full, got, portStr)
				}

				// Authority round-trip: the parsed host must be byte-identical
				// to the canonical encoding of (hostname, port).
				if want := net.JoinHostPort(tt.wantHostname, portStr); u.Host != want {
					t.Errorf("HelixCodeHost=%q: url %q parsed host = %q, want %q",
						tt.configHost, full, u.Host, want)
				}

				if u.Path != path {
					t.Errorf("HelixCodeHost=%q: url %q parsed path = %q, want %q",
						tt.configHost, full, u.Path, path)
				}
			}
		})
	}
}
