// §11.4.115 RED-baseline-on-the-broken-artifact guard for HXC-185
// (unbracketed IPv6 host:port composition) at internal/redis/redis.go.
//
// DEFECT (captured on the pre-fix artifact, go1.26.4):
//
//	redis.go:37  Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
//
// cfg.Host comes from configuration, so it can legitimately be a bare IPv6
// literal. The naive join then yields "::1:6379", which the Go resolver
// rejects outright — the client never dials at all:
//
//	net.Dial("tcp", "::1:6379")   -> dial tcp: address ::1:6379: too many colons in address
//	net.Dial("tcp", "[::1]:6379") -> connect: connection refused   (address ACCEPTED)
//
// This test does NOT re-implement the join. It drives the REAL exported
// constructor (redis.NewClient) and asserts on POSITIVE SINK-SIDE EVIDENCE
// (§11.4.69): a real TCP listener bound to IPv6 loopback records whether an
// inbound connection was actually accepted. A correctly composed address
// produces an accepted connection; a malformed one never reaches the wire.
//
// POLARITY SWITCH (§11.4.115):
//
//	RED_MODE=1 — assert the defect is PRESENT (no connection reaches the
//	             listener, and the error is the resolver's address rejection).
//	             Passes only on a PRE-FIX artifact.
//	RED_MODE=0 — (DEFAULT) standing GREEN regression guard: the listener MUST
//	             observe a real inbound connection.
package redis

import (
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"dev.helix.code/internal/config"
)

func hxc185RedMode() bool { return os.Getenv("RED_MODE") == "1" }

// hxc185Sink is a real TCP listener on IPv6 loopback that counts accepted
// connections. The count is the positive sink-side evidence (§11.4.69) that
// the address under test was composed well enough to actually reach the wire.
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
		// §11.4.3 honest SKIP: without IPv6 loopback the invariant genuinely
		// cannot be exercised on this host. Never a faked PASS.
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

// hxc185IsAddressRejection reports whether err is the resolver refusing the
// address itself (as opposed to a connection-level failure, which proves the
// address WAS accepted and dialed).
func hxc185IsAddressRejection(err error) bool {
	if err == nil {
		return false
	}
	m := err.Error()
	return strings.Contains(m, "too many colons in address") ||
		strings.Contains(m, "missing port in address")
}

func TestNewClient_IPv6Host_ReachesRealListener(t *testing.T) {
	sink := hxc185StartIPv6Sink(t)

	cases := []struct {
		name string
		host string
		// redBroken records whether the PRE-FIX `fmt.Sprintf("%s:%d")`
		// composition is broken for this host shape. A bare IPv6 literal is;
		// an already-bracketed one happened to compose correctly even pre-fix.
		redBroken bool
	}{
		{"bare_ipv6_literal", sink.host, true},
		{"already_bracketed_ipv6", "[" + sink.host + "]", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := sink.Accepted()

			_, err := NewClient(&config.RedisConfig{
				Enabled: true,
				Host:    tc.host,
				Port:    sink.port,
			})
			// NewClient always errors here: the sink closes the connection
			// immediately, so PING cannot complete. The verdict rests on
			// whether the connection reached the listener at all, not on
			// NewClient's own success.
			waitForSink(sink, before)
			delta := sink.Accepted() - before

			if hxc185RedMode() {
				if !tc.redBroken {
					t.Skipf("SKIP-OK HXC-185: the pre-fix composition was already correct "+
						"for host shape %q — nothing to reproduce here", tc.host)
				}
				if delta != 0 {
					t.Fatalf("RED_MODE=1: expected the malformed address to never reach the "+
						"wire, but the listener accepted %d connection(s) — the defect did not "+
						"reproduce; re-run with RED_MODE=0 for the standing guard", delta)
				}
				if !hxc185IsAddressRejection(err) {
					t.Fatalf("RED_MODE=1: expected a resolver address rejection, got: %v", err)
				}
				t.Logf("defect reproduced (captured evidence): host=%q port=%d -> %v",
					tc.host, sink.port, err)
				return
			}

			// RED_MODE=0 — standing GREEN regression guard.
			if hxc185IsAddressRejection(err) {
				t.Fatalf("NewClient composed an address the resolver REJECTS for host %q: %v",
					tc.host, err)
			}
			if delta < 1 {
				t.Fatalf("no connection reached the IPv6 listener for host %q port %d "+
					"(accepted delta=%d); NewClient err=%v. The composed address never "+
					"made it to the wire.", tc.host, sink.port, delta, err)
			}
			t.Logf("positive sink-side evidence: host=%q port=%d -> listener accepted %d connection(s)",
				tc.host, sink.port, delta)
		})
	}
}

// waitForSink gives the accept loop a bounded window to record a connection
// that the dialer has already established, so the assertion is not racing the
// listener goroutine.
func waitForSink(s *hxc185Sink, before int) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.Accepted() > before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
