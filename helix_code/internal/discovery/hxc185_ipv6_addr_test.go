// §11.4.115 RED-baseline-on-the-broken-artifact guard for HXC-185
// (unbracketed / double-bracketed IPv6 host:port composition) in
// internal/discovery.
//
// SITES GUARDED (all reached through their REAL production entry points):
//
//	registry.go      ServiceInfo.Address()          — the canonical address accessor
//	registry.go      (*ServiceRegistry).checkTCPHealth   — net.Dialer.DialContext
//	registry.go      (*ServiceRegistry).checkHTTPHealth  — composed health URL
//	health_monitor.go (*HealthMonitor).checkTCP         — net.DialTimeout
//	health_monitor.go (*HealthMonitor).checkHTTP        — composed health URL
//
// ServiceInfo.Host is populated by whoever registers the service (config,
// discovery, an operator), so it can legitimately be either a bare IPv6
// literal ("::1") or an already-bracketed one ("[::1]") — net.SplitHostPort
// strips brackets while a URL authority carries them, so both shapes occur.
//
// TWO DISTINCT PRE-FIX DEFECTS, captured on go1.26.4:
//
//  1. Naive join `fmt.Sprintf("%s:%d", host, port)` on a BARE IPv6 literal:
//     net.Dial("tcp", "::1:6379") -> address ::1:6379: too many colons in address
//
//  2. Hand-rolled bracketing `fmt.Sprintf("[%s]:%d", host, port)` (the pre-fix
//     health_monitor.checkTCP) on an ALREADY-BRACKETED host:
//     net.Dial("tcp", "[[::1]]:25") -> address [[::1]]:25: missing port in address
//
// The two defects are complementary: site (1) breaks on the bare shape and
// works on the bracketed one, site (2) breaks on the bracketed shape and works
// on the bare one. The tables below therefore carry a per-case `redBroken`
// flag rather than assuming a single polarity.
//
// The assertions rest on POSITIVE SINK-SIDE EVIDENCE (§11.4.69): a real
// listener / real httptest server bound to IPv6 loopback, and the production
// check function's own boolean/error verdict. Nothing here re-implements the
// join.
//
// POLARITY SWITCH (§11.4.115): RED_MODE=1 asserts the defect is PRESENT
// (passes only on a PRE-FIX artifact); RED_MODE=0 (DEFAULT) is the standing
// GREEN regression guard.
package discovery

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func hxc185RedMode() bool { return os.Getenv("RED_MODE") == "1" }

type hxc185Sink struct {
	mu       sync.Mutex
	accepted int
	host     string // UNBRACKETED, e.g. "::1"
	port     int
}

func (s *hxc185Sink) Accepted() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accepted
}

func hxc185StartIPv6Sink(t *testing.T) *hxc185Sink {
	t.Helper()
	ln, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("SKIP-OK HXC-185: IPv6 loopback unavailable on this host: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	addr := ln.Addr().(*net.TCPAddr)
	s := &hxc185Sink{host: addr.IP.String(), port: addr.Port}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.accepted++
			s.mu.Unlock()
			_ = conn.Close()
		}
	}()
	return s
}

// hxc185StartIPv6HTTP starts a real HTTP server bound to IPv6 loopback that
// serves 200 on any path, and reports the UNBRACKETED host and the port.
func hxc185StartIPv6HTTP(t *testing.T) (host string, port int, hits func() int) {
	t.Helper()
	ln, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("SKIP-OK HXC-185: IPv6 loopback unavailable on this host: %v", err)
	}

	var mu sync.Mutex
	count := 0
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
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
		return count
	}
}

func hxc185WaitSink(s *hxc185Sink, before int) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.Accepted() > before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestServiceInfo_Address_BracketsIPv6 drives the REAL ServiceInfo.Address()
// accessor and proves its output is an address the resolver accepts and that
// round-trips through net.SplitHostPort.
func TestServiceInfo_Address_BracketsIPv6(t *testing.T) {
	cases := []struct {
		name      string
		host      string
		port      int
		want      string
		redBroken bool
	}{
		{"ipv4", "127.0.0.1", 8080, "127.0.0.1:8080", false},
		{"hostname", "localhost", 8080, "localhost:8080", false},
		{"ipv6_bare", "::1", 8080, "[::1]:8080", true},
		{"ipv6_bracketed", "[::1]", 8080, "[::1]:8080", false},
		{"ipv6_full_bare", "2001:db8::1", 8080, "[2001:db8::1]:8080", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			si := &ServiceInfo{Name: "svc", Host: tc.host, Port: tc.port}
			got := si.Address()

			if hxc185RedMode() {
				if !tc.redBroken {
					t.Skipf("SKIP-OK HXC-185: pre-fix composition already correct for %q", tc.host)
				}
				if _, err := net.ResolveTCPAddr("tcp", got); err == nil {
					t.Fatalf("RED_MODE=1: expected Address() = %q to be REJECTED by the "+
						"resolver, but it was accepted — defect did not reproduce", got)
				} else {
					t.Logf("defect reproduced (captured evidence): Address()=%q -> %v", got, err)
				}
				return
			}

			if got != tc.want {
				t.Fatalf("ServiceInfo{Host:%q,Port:%d}.Address() = %q, want %q",
					tc.host, tc.port, got, tc.want)
			}
			if _, err := net.ResolveTCPAddr("tcp", got); err != nil {
				t.Fatalf("Address() produced %q which the resolver REJECTS: %v", got, err)
			}
			h, p, err := net.SplitHostPort(got)
			if err != nil {
				t.Fatalf("Address() output %q does not split: %v", got, err)
			}
			if p != strconv.Itoa(tc.port) {
				t.Fatalf("port round-trip = %q, want %d", p, tc.port)
			}
			if h == "" {
				t.Fatalf("host round-trip empty for %q", got)
			}
		})
	}
}

// TestHealthMonitor_checkTCP_IPv6 drives the REAL (*HealthMonitor).checkTCP
// against a REAL IPv6 listener. The already-bracketed case is the one the
// pre-fix hand-rolled `fmt.Sprintf("[%s]:%d", ...)` breaks on.
func TestHealthMonitor_checkTCP_IPv6(t *testing.T) {
	sink := hxc185StartIPv6Sink(t)
	hm := NewHealthMonitor(DefaultHealthMonitorConfig(), NewServiceRegistry(RegistryConfig{}))

	cases := []struct {
		name      string
		host      string
		redBroken bool
	}{
		// Pre-fix checkTCP bracketed any host containing ":", so the BARE
		// literal already worked; the ALREADY-BRACKETED one became "[[::1]]".
		{"bare_ipv6", sink.host, false},
		{"already_bracketed_ipv6", "[" + sink.host + "]", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := sink.Accepted()
			err := hm.checkTCP(&ServiceInfo{Name: "svc", Host: tc.host, Port: sink.port})
			hxc185WaitSink(sink, before)
			delta := sink.Accepted() - before

			if hxc185RedMode() {
				if !tc.redBroken {
					t.Skipf("SKIP-OK HXC-185: pre-fix checkTCP already correct for %q", tc.host)
				}
				if delta != 0 || err == nil {
					t.Fatalf("RED_MODE=1: expected checkTCP to fail before reaching the wire "+
						"for host %q (accepted delta=%d, err=%v) — defect did not reproduce",
						tc.host, delta, err)
				}
				t.Logf("defect reproduced (captured evidence): checkTCP(host=%q) -> %v", tc.host, err)
				return
			}

			if err != nil {
				t.Fatalf("checkTCP failed against a live IPv6 listener for host %q: %v", tc.host, err)
			}
			if delta < 1 {
				t.Fatalf("checkTCP reported success for host %q but the listener never accepted "+
					"a connection (delta=%d) — the PASS has no sink-side evidence behind it",
					tc.host, delta)
			}
			t.Logf("positive sink-side evidence: checkTCP(host=%q port=%d) accepted %d connection(s)",
				tc.host, sink.port, delta)
		})
	}
}

// TestHealthMonitor_checkHTTP_IPv6 drives the REAL (*HealthMonitor).checkHTTP
// against a REAL IPv6 HTTP server and asserts the server actually served the
// request (positive sink-side evidence).
func TestHealthMonitor_checkHTTP_IPv6(t *testing.T) {
	host, port, hits := hxc185StartIPv6HTTP(t)
	hm := NewHealthMonitor(DefaultHealthMonitorConfig(), NewServiceRegistry(RegistryConfig{}))

	for _, h := range []string{host, "[" + host + "]"} {
		t.Run(h, func(t *testing.T) {
			before := hits()
			err := hm.checkHTTP(&ServiceInfo{Name: "svc", Host: h, Port: port})
			delta := hits() - before

			if err != nil {
				t.Fatalf("checkHTTP failed against a live IPv6 HTTP server for host %q: %v", h, err)
			}
			if delta < 1 {
				t.Fatalf("checkHTTP reported success for host %q but the server never served a "+
					"request (delta=%d) — the PASS has no sink-side evidence behind it", h, delta)
			}
			t.Logf("positive sink-side evidence: checkHTTP(host=%q port=%d) served %d request(s)",
				h, port, delta)
		})
	}
}

// TestServiceRegistry_checkTCPHealth_IPv6 drives the REAL
// (*ServiceRegistry).checkTCPHealth against a REAL IPv6 listener.
func TestServiceRegistry_checkTCPHealth_IPv6(t *testing.T) {
	sink := hxc185StartIPv6Sink(t)
	reg := NewServiceRegistry(RegistryConfig{})

	cases := []struct {
		name      string
		host      string
		redBroken bool
	}{
		{"bare_ipv6", sink.host, true},
		{"already_bracketed_ipv6", "[" + sink.host + "]", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := sink.Accepted()
			ok := reg.checkTCPHealth(context.Background(),
				&ServiceInfo{Name: "svc", Host: tc.host, Port: sink.port})
			hxc185WaitSink(sink, before)
			delta := sink.Accepted() - before

			if hxc185RedMode() {
				if !tc.redBroken {
					t.Skipf("SKIP-OK HXC-185: pre-fix composition already correct for %q", tc.host)
				}
				if ok || delta != 0 {
					t.Fatalf("RED_MODE=1: expected checkTCPHealth to fail before the wire for "+
						"host %q (ok=%v, delta=%d) — defect did not reproduce", tc.host, ok, delta)
				}
				t.Logf("defect reproduced (captured evidence): checkTCPHealth(host=%q) = false, "+
					"listener saw nothing", tc.host)
				return
			}

			if !ok {
				t.Fatalf("checkTCPHealth returned false against a live IPv6 listener for host %q", tc.host)
			}
			if delta < 1 {
				t.Fatalf("checkTCPHealth returned true for host %q but the listener never accepted "+
					"a connection (delta=%d)", tc.host, delta)
			}
			t.Logf("positive sink-side evidence: checkTCPHealth(host=%q port=%d) accepted %d connection(s)",
				tc.host, sink.port, delta)
		})
	}
}

// TestServiceRegistry_checkHTTPHealth_IPv6 drives the REAL
// (*ServiceRegistry).checkHTTPHealth against a REAL IPv6 HTTP server.
func TestServiceRegistry_checkHTTPHealth_IPv6(t *testing.T) {
	host, port, hits := hxc185StartIPv6HTTP(t)
	reg := NewServiceRegistry(RegistryConfig{})

	for _, h := range []string{host, "[" + host + "]"} {
		t.Run(h, func(t *testing.T) {
			before := hits()
			ok := reg.checkHTTPHealth(context.Background(),
				&ServiceInfo{Name: "svc", Host: h, Port: port, Protocol: "http"})
			delta := hits() - before

			if !ok {
				t.Fatalf("checkHTTPHealth returned false against a live IPv6 HTTP server for host %q", h)
			}
			if delta < 1 {
				t.Fatalf("checkHTTPHealth returned true for host %q but the server never served a "+
					"request (delta=%d)", h, delta)
			}
			t.Logf("positive sink-side evidence: checkHTTPHealth(host=%q port=%d) served %d request(s)",
				h, port, delta)
		})
	}
}
