// HXC-218 — the `-action=start` half of the IPv6 class that HXC-202 closed for
// `-action=status` only.
//
// # Why one action worked and its sibling did not
//
// `-action=status` builds its target through sonarqubeStatusTarget, which sets
// HealthTarget.URL explicitly (the HXC-202 repair). health.CheckHTTP honours a
// non-empty URL verbatim, so that path was repaired.
//
// `-action=start` never goes near that function. It builds ServiceEndpoints
// with WithHost/WithPort and NO WithURL, hands them to boot.BootManager, and
// BootAll -> HealthCheckAll copies Host/Port into a HealthTarget whose URL is
// therefore EMPTY. CheckHTTP then composes the authority itself, and
// containers/pkg/health composed it without bracketing the host. The postgres
// endpoint is worse: it is HealthType "tcp", so it goes through
// HealthTarget.ResolvedAddress and is dialled directly.
//
// The repair landed in the containers submodule (internal/netaddr), so this
// file changes no production code here — it is the consuming-project guard that
// the start path is genuinely repaired end to end, and stays repaired if the
// submodule pointer ever moves backwards.
//
// # What was measured (go1.26.4)
//
//	net.ResolveTCPAddr("tcp", "::1:8080")  -> too many colons in address
//	url.Parse("http://::1:8080/x")         -> parses; Hostname="::1" Port="8080"
//
// url.Parse is LENIENT — it splits at the last colon — so the HTTP probe
// survived the unbracketed form while the TCP dial did not. The assertions
// below are pinned to that measured split: the TCP endpoint is the one that was
// hard-broken, and the HTTP endpoint's defect is the malformed authority it
// published. Asserting that the HTTP probe itself failed would be a bluff.
//
//	RED_MODE=1  assert the defect is PRESENT (PASSes only on the pre-fix artifact)
//	RED_MODE=0  assert the defect is ABSENT  (standing regression guard) [default]
package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"digital.vasic.containers/pkg/boot"
	"digital.vasic.containers/pkg/endpoint"
	"digital.vasic.containers/pkg/health"
)

// hxc218IPv6Listener returns a listener on the IPv6 loopback, or an honest SKIP
// when the host has no IPv6 loopback (§11.4.3).
func hxc218IPv6Listener(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("SKIP-OK: hardware_not_present — no IPv6 loopback on this host: %v", err)
	}
	return ln
}

func hxc218Port(t *testing.T, addr string) string {
	t.Helper()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SplitHostPort(%q) failed: %v", addr, err)
	}
	return port
}

// TestHXC218_StartPath_IPv6HealthCheckAll drives the REAL start-path plumbing:
// endpoints built exactly the way handleSonarQube builds them, a REAL
// BootManager, a REAL health checker, against a REAL HTTP server and a REAL TCP
// listener on the IPv6 loopback.
//
// This is the assertion the ticket turns on — it is the composed path, not a
// unit call, so it fails if any layer between the endpoint builder and the
// socket reintroduces an unbracketed authority.
func TestHXC218_StartPath_IPv6HealthCheckAll(t *testing.T) {
	httpLn := hxc218IPv6Listener(t)
	srv := &httptest.Server{
		Listener: httpLn,
		Config: &http.Server{Handler: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != sonarqubeHealth {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
			},
		)},
	}
	srv.Start()
	defer srv.Close()
	httpPort := hxc218Port(t, httpLn.Addr().String())

	tcpLn := hxc218IPv6Listener(t)
	defer func() { _ = tcpLn.Close() }()
	go func() {
		for {
			conn, err := tcpLn.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	tcpPort := hxc218Port(t, tcpLn.Addr().String())

	// Built exactly as handleSonarQube builds them: Host + Port, NO URL.
	sonarEp := endpoint.NewEndpoint().
		WithHost("::1").
		WithPort(httpPort).
		WithHealthPath(sonarqubeHealth).
		WithHealthType("http").
		WithRequired(true).
		WithEnabled(true).
		WithTimeout(5 * time.Second).
		Build()

	postgresEp := endpoint.NewEndpoint().
		WithHost("::1").
		WithPort(tcpPort).
		WithHealthType("tcp").
		WithEnabled(true).
		WithRequired(false).
		WithTimeout(5 * time.Second).
		Build()

	mgr := boot.NewBootManager(
		map[string]endpoint.ServiceEndpoint{
			"sonarqube": sonarEp,
			"postgres":  postgresEp,
		},
		boot.WithHealthChecker(health.NewDefaultChecker()),
	)

	errs := mgr.HealthCheckAll(context.Background())

	if inRedMode() {
		// The TCP endpoint is the one that is hard-broken pre-fix: the dial
		// string is unbracketed and the resolver refuses it outright.
		if errs["postgres"] == nil {
			t.Fatalf("RED_MODE=1 expected the pre-fix artifact to FAIL the TCP " +
				"health check of a LISTENING IPv6 socket, but it passed — the " +
				"defect is not present, so this is not a valid RED baseline")
		}
		t.Logf("RED reproduced on the pre-fix artifact: the start path cannot "+
			"health-check a LISTENING IPv6 socket — %v", errs["postgres"])
		return
	}

	if err := errs["sonarqube"]; err != nil {
		t.Fatalf("start path could not health-check a REACHABLE IPv6 HTTP "+
			"service at [::1]:%s: %v", httpPort, err)
	}
	if err := errs["postgres"]; err != nil {
		t.Fatalf("start path could not health-check a LISTENING IPv6 TCP "+
			"service at [::1]:%s: %v", tcpPort, err)
	}
}

// TestHXC218_StartPath_IPv4AndHostnameUnaffected is the negative case for the
// start path. The repair is host-shape-sensitive, so the shapes that already
// worked must be proven to still work — an over-eager bracket would break every
// existing IPv4 and hostname deployment, which is the far more common case.
func TestHXC218_StartPath_IPv4AndHostnameUnaffected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != sonarqubeHealth {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		},
	))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) failed: %v", srv.URL, err)
	}

	for _, host := range []string{u.Hostname(), "localhost"} {
		t.Run(host, func(t *testing.T) {
			ep := endpoint.NewEndpoint().
				WithHost(host).
				WithPort(u.Port()).
				WithHealthPath(sonarqubeHealth).
				WithHealthType("http").
				WithRequired(true).
				WithEnabled(true).
				WithTimeout(5 * time.Second).
				Build()

			mgr := boot.NewBootManager(
				map[string]endpoint.ServiceEndpoint{"sonarqube": ep},
				boot.WithHealthChecker(health.NewDefaultChecker()),
			)

			if err := mgr.HealthCheckAll(context.Background())["sonarqube"]; err != nil {
				t.Fatalf("start path regressed for host %q: %v — the IPv6 repair "+
					"must not perturb IPv4 or hostname deployments", host, err)
			}
		})
	}
}

// TestHXC218_StatusPath_StillWorks pins that the HXC-202 repair is intact.
// Both actions must work; a change that fixed start while regressing status
// would trade one broken action for another, which is exactly the failure this
// ticket family keeps producing.
func TestHXC218_StatusPath_StillWorks(t *testing.T) {
	ln := hxc218IPv6Listener(t)
	srv := &httptest.Server{
		Listener: ln,
		Config: &http.Server{Handler: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != sonarqubeHealth {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
			},
		)},
	}
	srv.Start()
	defer srv.Close()

	withSonarEnv(t, "::1", hxc218Port(t, ln.Addr().String()), func() {
		target := sonarqubeStatusTarget()
		if target.URL == "" {
			t.Fatal("sonarqubeStatusTarget no longer sets URL — the HXC-202 " +
				"repair has been removed")
		}
		res := health.NewDefaultChecker().Check(context.Background(), target)
		if !res.Healthy {
			t.Fatalf("status path could not reach a REACHABLE IPv6 SonarQube "+
				"at %s: %s", target.URL, res.Error)
		}
	})
}
