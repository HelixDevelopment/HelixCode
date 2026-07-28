// §11.4.115 RED-baseline-on-the-broken-artifact guard for HXC-185 at
// internal/memory/memory_manager.go (Redis + Memcached provider addresses).
//
// DEFECT (captured on the pre-fix artifact, go1.26.4):
//
//	memory_manager.go  Addr:    fmt.Sprintf("%s:%d", host, port)   // redis
//	memory_manager.go  servers = []string{fmt.Sprintf("%s:%d", host, port)}  // memcached
//
// `host` comes from the provider config map, so it can be a bare IPv6
// literal. Both backends hand the string straight to the Go resolver, which
// rejects it:
//
//	net.Dial("tcp", "::1:6379")   -> address ::1:6379: too many colons in address
//	net.Dial("tcp", "[::1]:6379") -> connect: connection refused   (ACCEPTED)
//
// Both tests drive the REAL exported constructors and REAL data-path calls
// against a REAL IPv6 listener, asserting the listener actually accepted an
// inbound connection (positive sink-side evidence, §11.4.69). Nothing here
// re-implements the join.
package memory

import (
	"context"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func hxc185RedMode() bool { return os.Getenv("RED_MODE") == "1" }

type hxc185Sink struct {
	mu       sync.Mutex
	accepted int
	host     string // UNBRACKETED
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

func hxc185WaitSink(s *hxc185Sink, before int) {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s.Accepted() > before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestNewRedisMemoryProvider_IPv6_ReachesListener(t *testing.T) {
	sink := hxc185StartIPv6Sink(t)

	for _, tc := range []struct {
		name      string
		host      string
		redBroken bool
	}{
		{"bare_ipv6", sink.host, true},
		{"already_bracketed_ipv6", "[" + sink.host + "]", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewRedisMemoryProvider(map[string]interface{}{
				"host": tc.host,
				"port": sink.port,
			})
			if err != nil {
				t.Fatalf("NewRedisMemoryProvider returned an error: %v", err)
			}

			before := sink.Accepted()
			// Health drives a REAL PING through the real go-redis client.
			healthErr := p.Health(context.Background())
			hxc185WaitSink(sink, before)
			delta := sink.Accepted() - before

			if hxc185RedMode() {
				if !tc.redBroken {
					t.Skipf("SKIP-OK HXC-185: pre-fix composition already correct for %q", tc.host)
				}
				if delta != 0 {
					t.Fatalf("RED_MODE=1: expected the malformed address never to reach the wire, "+
						"but the listener accepted %d connection(s) — defect did not reproduce", delta)
				}
				t.Logf("defect reproduced (captured evidence): host=%q -> %v", tc.host, healthErr)
				return
			}

			if delta < 1 {
				t.Fatalf("no connection reached the IPv6 listener for host %q port %d "+
					"(delta=%d, Health err=%v) — the composed address never made it to the wire",
					tc.host, sink.port, delta, healthErr)
			}
			t.Logf("positive sink-side evidence: redis provider host=%q port=%d -> listener accepted %d connection(s)",
				tc.host, sink.port, delta)
		})
	}
}

func TestNewMemcachedMemoryProvider_IPv6_ReachesListener(t *testing.T) {
	sink := hxc185StartIPv6Sink(t)

	for _, tc := range []struct {
		name      string
		host      string
		redBroken bool
	}{
		{"bare_ipv6", sink.host, true},
		{"already_bracketed_ipv6", "[" + sink.host + "]", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewMemcachedMemoryProvider(map[string]interface{}{
				"host": tc.host,
				"port": sink.port,
			})
			if err != nil {
				t.Fatalf("NewMemcachedMemoryProvider returned an error: %v", err)
			}

			before := sink.Accepted()
			// Store drives a REAL SET through the real gomemcache client.
			storeErr := p.Store(context.Background(), "hxc185", map[string]string{"k": "v"})
			hxc185WaitSink(sink, before)
			delta := sink.Accepted() - before

			if hxc185RedMode() {
				if !tc.redBroken {
					t.Skipf("SKIP-OK HXC-185: pre-fix composition already correct for %q", tc.host)
				}
				if delta != 0 {
					t.Fatalf("RED_MODE=1: expected the malformed address never to reach the wire, "+
						"but the listener accepted %d connection(s) — defect did not reproduce", delta)
				}
				t.Logf("defect reproduced (captured evidence): host=%q -> %v", tc.host, storeErr)
				return
			}

			if delta < 1 {
				t.Fatalf("no connection reached the IPv6 listener for host %q port %d "+
					"(delta=%d, Store err=%v) — the composed server address never made it to the wire",
					tc.host, sink.port, delta, storeErr)
			}
			t.Logf("positive sink-side evidence: memcached provider host=%q port=%d -> listener accepted %d connection(s)",
				tc.host, sink.port, delta)
		})
	}
}
