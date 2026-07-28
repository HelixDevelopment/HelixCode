// §11.4.115 RED-baseline-on-the-broken-artifact guard for HXC-185 at
// internal/notification/engine.go (EmailChannel SMTP address).
//
// DEFECT (captured on the pre-fix artifact, go1.26.4):
//
//	engine.go  addr := fmt.Sprintf("%s:%d", c.smtpServer, c.port)
//
// c.smtpServer is operator-supplied and can be a bare IPv6 literal. The naive
// join yields "::1:25", which smtp.SendMail's net.Dial rejects:
//
//	net.Dial("tcp", "::1:25")   -> address ::1:25: too many colons in address
//	net.Dial("tcp", "[::1]:25") -> connect: connection refused   (ACCEPTED)
//
// so no mail could ever be sent to an IPv6 SMTP relay.
//
// This drives the REAL exported constructor (NewEmailChannel) and the REAL
// Send path against a REAL IPv6 listener, asserting the listener actually
// accepted an inbound connection (positive sink-side evidence, §11.4.69).
// Send still fails afterwards — the sink is not a real SMTP server — but the
// verdict rests on whether the dial reached the wire, which is precisely the
// invariant HXC-185 restores.
package notification

import (
	"context"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestEmailChannel_Send_IPv6_ReachesListener(t *testing.T) {
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
			ch := NewEmailChannel(tc.host, sink.port, "user", "pass", "from@example.test")

			n := &Notification{
				ID:       uuid.New(),
				Title:    "hxc185",
				Message:  "hxc185",
				Metadata: map[string]interface{}{"recipients": []string{"to@example.test"}},
			}

			before := sink.Accepted()
			sendErr := ch.Send(context.Background(), n)

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
					t.Fatalf("RED_MODE=1: expected the malformed SMTP address never to reach the "+
						"wire, but the listener accepted %d connection(s) — defect did not reproduce", delta)
				}
				t.Logf("defect reproduced (captured evidence): smtpServer=%q -> %v", tc.host, sendErr)
				return
			}

			if delta < 1 {
				t.Fatalf("no SMTP connection reached the IPv6 listener for host %q port %d "+
					"(delta=%d, Send err=%v) — the composed address never made it to the wire",
					tc.host, sink.port, delta, sendErr)
			}
			t.Logf("positive sink-side evidence: smtpServer=%q port=%d -> listener accepted %d connection(s)",
				tc.host, sink.port, delta)
		})
	}
}
