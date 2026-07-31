// HXC-202 — sibling miss of HXC-185: security_scan composed the SonarQube
// endpoint with a plain `fmt.Sprintf("http://%s:%s", host, port)`. The host is
// operator-supplied via HELIX_SONARQUBE_HOST, so it can be a bare IPv6 literal,
// and the result was then used BOTH as the address reported to the operator and
// as the address probed.
//
// # Why the probe, not only the report
//
// health.CheckHTTP honours HealthTarget.URL when it is non-empty and otherwise
// builds `scheme://host:port/path` itself, unbracketed
// (containers/pkg/health/http.go). Setting URL to a correctly bracketed value is
// therefore what makes the probe actually reach an IPv6 SonarQube — and it also
// guarantees the reported endpoint is byte-identical to the probed one.
//
// # Polarity switch (§11.4.115)
//
//	RED_MODE=1  assert the defect is PRESENT (PASSes only pre-fix)
//	RED_MODE=0  assert the defect is ABSENT  (standing guard)  [default]
//
// The tests drive the REAL production functions; nothing re-implements the join.
package main

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

// withSonarEnv runs fn with the SonarQube host/port env overrides applied and
// restores the previous values afterwards.
func withSonarEnv(t *testing.T, host, port string, fn func()) {
	t.Helper()
	t.Setenv(envSonarqubeHost, host)
	t.Setenv(envSonarqubePort, port)
	fn()
}

func inRedMode() bool { return os.Getenv("RED_MODE") == "1" }

// TestHXC202_SonarqubeHealthURL_IPv6IsBracketed is the positive case.
func TestHXC202_SonarqubeHealthURL_IPv6IsBracketed(t *testing.T) {
	withSonarEnv(t, "::1", "9000", func() {
		got := sonarqubeHealthURL()

		if inRedMode() {
			const prefix = "http://::1:9000"
			if !strings.HasPrefix(got, prefix) {
				t.Fatalf("RED_MODE=1 expected the pre-fix unbracketed authority "+
					"%q, got %q — the defect is not present on this artifact, so "+
					"this is not a valid RED baseline", prefix, got)
			}
			if _, err := url.Parse(got); err == nil {
				t.Fatalf("RED_MODE=1 expected url.Parse to REJECT %q, but it "+
					"parsed — the defect is not reproduced", got)
			}
			t.Logf("RED reproduced on the pre-fix artifact: sonarqubeHealthURL "+
				"-> %q (url.Parse rejects it)", got)
			return
		}

		want := "http://[::1]:9000" + sonarqubeHealth
		if got != want {
			t.Fatalf("sonarqubeHealthURL() = %q, want %q — an IPv6 literal must "+
				"be bracketed or the probe can never reach SonarQube", got, want)
		}
		u, err := url.Parse(got)
		if err != nil {
			t.Fatalf("url.Parse(%q) failed: %v — the probed URL is unusable", got, err)
		}
		if u.Hostname() != "::1" {
			t.Fatalf("url.Parse(%q).Hostname() = %q, want %q", got, u.Hostname(), "::1")
		}
		if u.Port() != "9000" {
			t.Fatalf("url.Parse(%q).Port() = %q, want %q", got, u.Port(), "9000")
		}
		if u.Path != sonarqubeHealth {
			t.Fatalf("url.Parse(%q).Path = %q, want %q", got, u.Path, sonarqubeHealth)
		}
	})
}

// TestHXC202_SonarqubeStatusTarget_ProbesTheReportedURL is the sink-side
// assertion: the health target handed to the checker must carry the bracketed
// URL, because that is the field health.CheckHTTP honours. Without it the
// operator would be told one endpoint while a different, unreachable one was
// probed.
func TestHXC202_SonarqubeStatusTarget_ProbesTheReportedURL(t *testing.T) {
	withSonarEnv(t, "::1", "9000", func() {
		target := sonarqubeStatusTarget()

		if inRedMode() {
			if target.URL != "" {
				t.Fatalf("RED_MODE=1 expected the pre-fix target to carry NO "+
					"explicit URL (so CheckHTTP falls back to unbracketed "+
					"host:port), but URL = %q", target.URL)
			}
			t.Logf("RED reproduced: status target carries no URL, so CheckHTTP "+
				"composes the unbracketed form from Host=%q Port=%q",
				target.URL, target.Port)
			return
		}

		if target.URL == "" {
			t.Fatal("status target carries no URL — health.CheckHTTP would fall " +
				"back to its own unbracketed scheme://host:port composition, so " +
				"an IPv6 SonarQube would never be reached")
		}
		if target.URL != sonarqubeHealthURL() {
			t.Fatalf("status target URL = %q but the operator is told %q — the "+
				"reported endpoint must be the probed endpoint",
				target.URL, sonarqubeHealthURL())
		}
		u, err := url.Parse(target.URL)
		if err != nil {
			t.Fatalf("url.Parse(target.URL=%q) failed: %v", target.URL, err)
		}
		if u.Hostname() != "::1" {
			t.Fatalf("probed host = %q, want %q", u.Hostname(), "::1")
		}
	})
}

// TestHXC202_SonarqubeHealthURL_NegativeCases pins what MUST NOT change. Both
// artifacts agree on these rows; that agreement is the point.
func TestHXC202_SonarqubeHealthURL_NegativeCases(t *testing.T) {
	tests := []struct {
		name string
		host string
		port string
		want string
		why  string
	}{
		{
			name: "default_localhost_unchanged",
			host: defaultSonarqubeHost,
			port: defaultSonarqubePort,
			want: "http://localhost:9000" + sonarqubeHealth,
			why:  "the shipped default must not be perturbed",
		},
		{
			name: "ipv4_literal_unchanged",
			host: "127.0.0.1",
			port: "9000",
			want: "http://127.0.0.1:9000" + sonarqubeHealth,
			why:  "an IPv4 literal carries no colon and must pass through untouched",
		},
		{
			name: "hostname_unchanged",
			host: "sonar.internal",
			port: "9000",
			want: "http://sonar.internal:9000" + sonarqubeHealth,
			why:  "a hostname must never be bracketed",
		},
		{
			name: "already_bracketed_not_double_bracketed",
			host: "[::1]",
			port: "9000",
			want: "http://[::1]:9000" + sonarqubeHealth,
			why: "net.JoinHostPort brackets unconditionally on seeing a colon, so " +
				"an already-bracketed host would become [[::1]] — rejected just as " +
				"hard as the unbracketed form",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withSonarEnv(t, tt.host, tt.port, func() {
				// The bracketed rows are repaired cases: they differ between
				// artifacts, so assert them only in the GREEN polarity.
				if inRedMode() && strings.Contains(tt.want, "[") {
					t.Skipf("SKIP-OK: RED_MODE=1 — %q is a repaired case, "+
						"asserted in the GREEN polarity", tt.name)
				}

				got := sonarqubeHealthURL()
				if got != tt.want {
					t.Fatalf("sonarqubeHealthURL() with host=%q port=%q = %q, "+
						"want %q — %s", tt.host, tt.port, got, tt.want, tt.why)
				}
				if _, err := url.Parse(got); err != nil {
					t.Fatalf("url.Parse(%q) failed: %v", got, err)
				}
			})
		})
	}
}
