// §11.4.115 RED-baseline-on-the-broken-artifact guard for HXC-185 at
// internal/cognee/client.go (NewClient base-URL composition).
//
// DEFECT (captured on the pre-fix artifact, go1.26.4):
//
//	client.go  baseURL := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
//
// cfg.Host is operator-supplied and can be a bare IPv6 literal. An IPv6
// literal is only valid in a URL authority when bracketed (RFC 3986 §3.2.2),
// and under this module's go1.26 language version net/url rejects the
// unbracketed form outright:
//
//	parse "http://::1:8000/health": invalid port "::1:8000" after host
//
// so EVERY request built from the base URL failed before dialling.
//
// This drives the REAL exported constructor (cognee.NewClient) and the REAL
// TestConnection request path against a REAL IPv6 HTTP server, and asserts the
// server actually served the request (positive sink-side evidence, §11.4.69).
package cognee

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"

	"dev.helix.code/internal/config"
)

func hxc185RedMode() bool { return os.Getenv("RED_MODE") == "1" }

// hxc185StartIPv6HealthServer starts a real HTTP server on IPv6 loopback that
// answers 200 on /health, returning the UNBRACKETED host, the port, and a hit
// counter.
func hxc185StartIPv6HealthServer(t *testing.T) (host string, port int, hits func() int) {
	t.Helper()
	ln, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("SKIP-OK HXC-185: IPv6 loopback unavailable on this host: %v", err)
	}

	var mu sync.Mutex
	n := 0
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	_ = srv.Listener.Close()
	srv.Listener = ln
	srv.Start()
	t.Cleanup(srv.Close)

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, func() int {
		mu.Lock()
		defer mu.Unlock()
		return n
	}
}

func TestNewClient_IPv6Host_BaseURLReachesServer(t *testing.T) {
	host, port, hits := hxc185StartIPv6HealthServer(t)

	cases := []struct {
		name      string
		host      string
		redBroken bool
	}{
		{"bare_ipv6", host, true},
		{"already_bracketed_ipv6", "[" + host + "]", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(&config.CogneeConfig{Host: tc.host, Port: port})
			base := c.GetBaseURL()

			if hxc185RedMode() {
				if !tc.redBroken {
					t.Skipf("SKIP-OK HXC-185: pre-fix composition already correct for %q", tc.host)
				}
				if _, err := url.Parse(base + "/health"); err == nil {
					t.Fatalf("RED_MODE=1: expected the composed base URL %q to be REJECTED by "+
						"net/url, but it parsed — defect did not reproduce", base)
				} else {
					t.Logf("defect reproduced (captured evidence): url.Parse(%q) -> %v", base+"/health", err)
				}
				return
			}

			// The base URL must be a well-formed URL whose authority round-trips.
			u, err := url.Parse(base)
			if err != nil {
				t.Fatalf("NewClient produced an unparseable base URL %q: %v", base, err)
			}
			wantAuthority := net.JoinHostPort(host, strconv.Itoa(port))
			if u.Host != wantAuthority {
				t.Fatalf("base URL authority = %q, want %q (IPv6 literals MUST be bracketed)",
					u.Host, wantAuthority)
			}

			// Positive sink-side evidence: the REAL request path must reach the
			// REAL server.
			before := hits()
			if ok := c.TestConnection(context.Background()); !ok {
				t.Fatalf("TestConnection returned false against a live IPv6 server (base=%q)", base)
			}
			if delta := hits() - before; delta < 1 {
				t.Fatalf("TestConnection reported success but the server never served a request "+
					"(delta=%d, base=%q) — the PASS has no sink-side evidence behind it", delta, base)
			}
			t.Logf("positive sink-side evidence: base=%q served the /health probe", base)
		})
	}
}
