// §11.4.115 RED-baseline-on-the-broken-artifact guard for HXC-185 at
// internal/worker/ssh_pool.go (createSSHClient dial address).
//
// DEFECT (captured on the pre-fix artifact, go1.26.4):
//
//	ssh_pool.go  ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), sshConfig)
//
// config.Host is operator-supplied and can be a bare IPv6 literal. The naive
// join yields "::1:22", which ssh.Dial's underlying net.Dial rejects:
//
//	net.Dial("tcp", "::1:22")   -> address ::1:22: too many colons in address
//	net.Dial("tcp", "[::1]:22") -> connect: connection refused   (ACCEPTED)
//
// so an IPv6 worker host could never be reached.
//
// This drives the REAL (*SSHWorkerPool).createSSHClient against a REAL IPv6
// listener and asserts the listener actually accepted an inbound connection
// (positive sink-side evidence, §11.4.69). The SSH handshake still fails
// afterwards — the sink is not an SSH server — but the verdict rests on
// whether the dial reached the wire, which is the invariant under test.
package worker

import (
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

func TestCreateSSHClient_IPv6_ReachesListener(t *testing.T) {
	sink := hxc185StartIPv6Sink(t)
	pool := NewSSHWorkerPool(false)
	t.Cleanup(pool.Close)

	for _, tc := range []struct {
		name      string
		host      string
		redBroken bool
	}{
		{"bare_ipv6", sink.host, true},
		{"already_bracketed_ipv6", "[" + sink.host + "]", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := sink.Accepted()

			_, dialErr := pool.createSSHClient(&SSHWorkerConfig{
				Host:     tc.host,
				Port:     sink.port,
				Username: "hxc185",
				Password: "hxc185",
			})

			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) && sink.Accepted() == before {
				time.Sleep(10 * time.Millisecond)
			}
			delta := sink.Accepted() - before

			if hxc185RedMode() {
				if !tc.redBroken {
					t.Skipf("SKIP-OK HXC-185: pre-fix composition already correct for %q", tc.host)
				}
				if delta != 0 {
					t.Fatalf("RED_MODE=1: expected the malformed address never to reach the wire, "+
						"but the listener accepted %d connection(s) — defect did not reproduce", delta)
				}
				t.Logf("defect reproduced (captured evidence): host=%q -> %v", tc.host, dialErr)
				return
			}

			if delta < 1 {
				t.Fatalf("no SSH connection reached the IPv6 listener for host %q port %d "+
					"(delta=%d, dial err=%v) — the composed address never made it to the wire",
					tc.host, sink.port, delta, dialErr)
			}
			t.Logf("positive sink-side evidence: ssh host=%q port=%d -> listener accepted %d connection(s)",
				tc.host, sink.port, delta)
		})
	}
}
