// §11.4.115 RED-baseline-on-the-broken-artifact guard for HXC-185 at
// tests/testinfra/testinfra.go (service URL + Redis address composition).
//
// DEFECT (captured on the pre-fix artifact, go1.26.4):
//
//	testinfra.go  fmt.Sprintf("http://%s:%s", c.CogneeHost, c.CogneePort)   // and siblings
//	testinfra.go  Addr: fmt.Sprintf("%s:%s", i.config.RedisHost, i.config.RedisPort)
//
// Every *Host field is populated from an environment variable (see getEnv), so
// each is operator-supplied and can be a bare IPv6 literal. An IPv6 literal is
// only valid in a URL authority when bracketed (RFC 3986 §3.2.2), and under
// this module's go1.26 language version net/url rejects the unbracketed form:
//
//	parse "http://::1:8000/" -> invalid port "::1:8000" after host
//
// while the resolver rejects the bare dial address outright:
//
//	net.Dial("tcp", "::1:6379") -> address ::1:6379: too many colons in address
//
// The URL assertions drive the REAL exported Config accessors and then issue a
// REAL request against a REAL IPv6 HTTP server, asserting the server actually
// served it (positive sink-side evidence, §11.4.69).
package testinfra

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
)

func hxc185RedMode() bool { return os.Getenv("RED_MODE") == "1" }

func hxc185StartIPv6HTTP(t *testing.T) (host, port string, hits func() int) {
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
	return addr.IP.String(), strconv.Itoa(addr.Port), func() int {
		mu.Lock()
		defer mu.Unlock()
		return n
	}
}

// TestConfigURLBuilders_IPv6 drives every REAL exported URL accessor on
// Config and proves each produces a URL that parses, carries a bracketed
// authority, and actually reaches a live IPv6 server.
func TestConfigURLBuilders_IPv6(t *testing.T) {
	host, port, hits := hxc185StartIPv6HTTP(t)

	for _, shape := range []struct {
		name      string
		host      string
		redBroken bool
	}{
		{"bare_ipv6", host, true},
		{"already_bracketed_ipv6", "[" + host + "]", false},
	} {
		t.Run(shape.name, func(t *testing.T) {
			c := &Config{
				CogneeHost: shape.host, CogneePort: port,
				ChromaDBHost: shape.host, ChromaDBPort: port,
				QdrantHost: shape.host, QdrantPort: port,
				OllamaHost: shape.host, OllamaPort: port,
			}

			builders := map[string]func() string{
				"CogneeURL":   c.CogneeURL,
				"ChromaDBURL": c.ChromaDBURL,
				"QdrantURL":   c.QdrantURL,
				"OllamaURL":   c.OllamaURL,
			}

			wantAuthority := net.JoinHostPort(host, port)

			for name, build := range builders {
				got := build()

				if hxc185RedMode() {
					if !shape.redBroken {
						t.Skipf("SKIP-OK HXC-185: pre-fix composition already correct for %q", shape.host)
					}
					if _, err := url.Parse(got); err == nil {
						t.Fatalf("RED_MODE=1: expected %s() = %q to be REJECTED by net/url, but it "+
							"parsed — defect did not reproduce", name, got)
					}
					continue
				}

				u, err := url.Parse(got)
				if err != nil {
					t.Fatalf("%s() produced an unparseable URL %q: %v", name, got, err)
				}
				if u.Host != wantAuthority {
					t.Fatalf("%s() authority = %q, want %q (IPv6 literals MUST be bracketed)",
						name, u.Host, wantAuthority)
				}

				before := hits()
				resp, gerr := http.Get(got)
				if gerr != nil {
					t.Fatalf("%s() = %q did not reach the live IPv6 server: %v", name, got, gerr)
				}
				_ = resp.Body.Close()
				if delta := hits() - before; delta < 1 {
					t.Fatalf("%s() = %q reported success but the server served no request (delta=%d)",
						name, got, delta)
				}
			}

			if hxc185RedMode() {
				t.Logf("defect reproduced (captured evidence): every URL builder emitted an "+
					"unbracketed IPv6 authority for host %q", shape.host)
				return
			}
			t.Logf("positive sink-side evidence: all 4 URL builders reached the live IPv6 server at %q",
				wantAuthority)
		})
	}
}

// TestConfigDialURLs_IPv6 covers the two credentialed URL builders and the
// Redis dial address, whose authorities must also bracket IPv6 literals.
func TestConfigDialURLs_IPv6(t *testing.T) {
	host, port, _ := hxc185StartIPv6HTTP(t)
	wantAuthority := net.JoinHostPort(host, port)

	c := &Config{
		PostgresHost: host, PostgresPort: port, PostgresUser: "u", PostgresPassword: "p", PostgresDB: "d",
		RedisHost: host, RedisPort: port, RedisPassword: "p",
	}

	for name, got := range map[string]string{
		"PostgresURL": c.PostgresURL(),
		"RedisURL":    c.RedisURL(),
	} {
		u, err := url.Parse(got)
		if err != nil {
			t.Fatalf("%s() produced an unparseable URL %q: %v", name, got, err)
		}
		if u.Host != wantAuthority {
			t.Fatalf("%s() authority = %q, want %q (IPv6 literals MUST be bracketed)",
				name, u.Host, wantAuthority)
		}
		if _, rerr := net.ResolveTCPAddr("tcp", u.Host); rerr != nil {
			t.Fatalf("%s() authority %q is not a resolvable address: %v", name, u.Host, rerr)
		}
		t.Logf("%s() -> %q (authority %q resolvable)", name, redact(got), u.Host)
	}
}

// redact hides the credential portion of a URL so a failing test never prints
// a password (CONST-042 / §11.4.10).
func redact(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<unparseable>"
	}
	if u.User != nil {
		u.User = url.User("REDACTED")
	}
	return u.String()
}
