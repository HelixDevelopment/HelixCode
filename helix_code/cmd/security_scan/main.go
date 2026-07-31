// cmd/security_scan — HelixCode security scanner bootstrap via containers BootManager.
//
// This binary wires the SonarQube and Snyk container lifecycle through the
// digital.vasic.containers BootManager (pkg/boot, pkg/endpoint, pkg/health, pkg/runtime).
// It replaces bare docker-compose calls in scripts/security-scan.sh (P0-T08.7/4).
//
// Usage:
//
//	go run ./cmd/security_scan -scanner=sonarqube [-action=start|stop|status]
//	go run ./cmd/security_scan -scanner=snyk [-action=start|stop|status]
//
// Credentials are read from the environment (loaded by the calling script from .env).
// No credentials are baked into this binary.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"digital.vasic.containers/pkg/boot"
	"digital.vasic.containers/pkg/endpoint"
	"digital.vasic.containers/pkg/health"
	"digital.vasic.containers/pkg/runtime"

	"dev.helix.code/internal/netutil"
)

const (
	defaultSonarqubeHost = "localhost"
	defaultSonarqubePort = "9000"
	sonarqubeHealth      = "/api/system/status"
	defaultTimeout       = 5 * time.Minute
	defaultRetries       = 30
	retryInterval        = 10 * time.Second

	// Defaults for SonarQube's backing PostgreSQL sidecar (same compose file).
	defaultSonarqubeDBHost = "localhost"
	defaultSonarqubeDBPort = "5432"

	// Env overrides for the SonarQube endpoint. Keeping the address
	// configurable rather than hardcoded means (a) a non-default deployment
	// needs no code change, and (b) the health-check tests can point at a
	// deterministically-closed port instead of depending on whether a real
	// SonarQube happens to be listening on the host running the suite.
	//
	// scripts/security-scan.sh reads HELIX_SONARQUBE_HOST / _PORT with the
	// SAME defaults, so the shell half of the scan path and this binary always
	// resolve the same address (no split brain).
	envSonarqubeHost = "HELIX_SONARQUBE_HOST"
	envSonarqubePort = "HELIX_SONARQUBE_PORT"

	// Env overrides for the SonarQube PostgreSQL sidecar, for symmetry with
	// the two above: a deployment that relocates SonarQube almost always
	// relocates its database with it, and a hardcoded localhost:5432 would
	// silently TCP-probe the wrong (or a completely unrelated) Postgres.
	// These are deliberately SonarQube-scoped so they cannot collide with the
	// application's own DB_HOST / DB_PORT.
	envSonarqubeDBHost = "HELIX_SONARQUBE_DB_HOST"
	envSonarqubeDBPort = "HELIX_SONARQUBE_DB_PORT"
)

// envOrDefault returns the value of env var name, or def when it is unset/empty.
func envOrDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// sonarqubeHost returns the configured SonarQube host, or the default.
func sonarqubeHost() string { return envOrDefault(envSonarqubeHost, defaultSonarqubeHost) }

// sonarqubePort returns the configured SonarQube port, or the default.
func sonarqubePort() string { return envOrDefault(envSonarqubePort, defaultSonarqubePort) }

// sonarqubeDBHost returns the configured SonarQube PostgreSQL host, or the default.
func sonarqubeDBHost() string { return envOrDefault(envSonarqubeDBHost, defaultSonarqubeDBHost) }

// sonarqubeDBPort returns the configured SonarQube PostgreSQL port, or the default.
func sonarqubeDBPort() string { return envOrDefault(envSonarqubeDBPort, defaultSonarqubeDBPort) }

// sonarqubeBaseURL returns the scheme+authority of the configured SonarQube.
//
// HXC-202 (sibling miss of HXC-185): the host arrives from HELIX_SONARQUBE_HOST
// and is a bare HOST, so it can be a bare IPv6 literal. An IPv6 literal contains
// colons, so `fmt.Sprintf("http://%s:%s", host, port)` produced an authority
// that is invalid per RFC 3986 §3.2.2 and that net/url rejects. netutil is the
// ONE shared join and is idempotent for an already-bracketed host; hostnames and
// IPv4 literals pass through byte-for-byte unchanged.
func sonarqubeBaseURL() string {
	return "http://" + netutil.JoinHostPortStr(sonarqubeHost(), sonarqubePort())
}

// sonarqubeHealthURL returns the full URL of the SonarQube system-status
// endpoint. It is BOTH the address that is actually probed and the address
// reported to the operator, so the two can never disagree.
func sonarqubeHealthURL() string {
	return sonarqubeBaseURL() + sonarqubeHealth
}

// sonarqubeStatusTarget builds the health target probed by `-action=status`.
//
// HXC-202: URL is set explicitly. health.CheckHTTP honours a non-empty URL and
// otherwise composes `scheme://host:port/path` itself WITHOUT bracketing
// (containers pkg/health/http.go), so an IPv6 SonarQube would never be reached.
// Setting URL both fixes the probe and guarantees the endpoint reported to the
// operator is byte-identical to the one actually probed. Host/Port/Path are kept
// so the target stays self-describing for any other consumer.
func sonarqubeStatusTarget() health.HealthTarget {
	return health.HealthTarget{
		Name:    "sonarqube",
		Host:    sonarqubeHost(),
		Port:    sonarqubePort(),
		Path:    sonarqubeHealth,
		URL:     sonarqubeHealthURL(),
		Type:    health.HealthHTTP,
		Timeout: 10 * time.Second,
	}
}

func main() {
	scanner := flag.String("scanner", "", "Scanner to boot: sonarqube|snyk")
	action := flag.String("action", "start", "Action: start|status (stop is not yet implemented)")
	flag.Parse()

	if *scanner == "" {
		fmt.Fprintln(os.Stderr, "Usage: security-scan -scanner=sonarqube|snyk [-action=start|stop|status]")
		os.Exit(1)
	}

	// Resolve project directory (two levels up from binary location or working dir)
	projectDir, err := resolveProjectDir()
	if err != nil {
		log.Fatalf("security-scan: failed to resolve project dir: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Auto-detect container runtime (Docker or Podman)
	rt, err := runtime.AutoDetect(ctx)
	if err != nil {
		log.Fatalf("security-scan: no container runtime available: %v", err)
	}
	log.Printf("security-scan: detected runtime: %s", rt.Name())

	switch *scanner {
	case "sonarqube", "sonar":
		if err := handleSonarQube(ctx, projectDir, rt, *action); err != nil {
			log.Fatalf("security-scan: sonarqube %s failed: %v", *action, err)
		}
	case "snyk":
		if err := handleSnyk(ctx, projectDir, rt, *action); err != nil {
			log.Fatalf("security-scan: snyk %s failed: %v", *action, err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown scanner %q. Use: sonarqube|snyk\n", *scanner)
		os.Exit(1)
	}
}

func handleSonarQube(ctx context.Context, projectDir string, rt runtime.ContainerRuntime, action string) error {
	composeFile := filepath.Join(projectDir, "docker", "security", "sonarqube", "docker-compose.yml")

	sonarEp := endpoint.NewEndpoint().
		WithHost(sonarqubeHost()).
		WithPort(sonarqubePort()).
		WithHealthPath(sonarqubeHealth).
		WithHealthType("http").
		WithRequired(true).
		WithEnabled(true).
		WithComposeFile(composeFile).
		WithServiceName("sonarqube").
		WithTimeout(30 * time.Second).
		WithRetryCount(defaultRetries).
		Build()

	postgresEp := endpoint.NewEndpoint().
		WithHost(sonarqubeDBHost()).
		WithPort(sonarqubeDBPort()).
		WithHealthType("tcp").
		WithEnabled(true).
		WithRequired(false).
		WithComposeFile(composeFile).
		WithServiceName("postgres").
		WithTimeout(15 * time.Second).
		WithRetryCount(10).
		Build()

	endpoints := map[string]endpoint.ServiceEndpoint{
		"sonarqube": sonarEp,
		"postgres":  postgresEp,
	}

	checker := health.NewDefaultChecker()
	mgr := boot.NewBootManager(
		endpoints,
		boot.WithRuntime(rt),
		boot.WithHealthChecker(checker),
		boot.WithProjectDir(projectDir),
	)

	switch action {
	case "start":
		log.Println("security-scan: booting SonarQube via containers BootManager...")
		summary, err := mgr.BootAll(ctx)
		if err != nil {
			return fmt.Errorf("BootAll failed: %w", err)
		}
		log.Printf("security-scan: SonarQube boot complete — started=%d skipped=%d failed=%d",
			summary.Started, summary.Skipped, summary.Failed)
		if summary.Failed > 0 {
			return fmt.Errorf("one or more required services failed to start")
		}
		log.Printf("security-scan: SonarQube ready at %s", sonarqubeBaseURL())
	case "status":
		target := sonarqubeStatusTarget()
		result := checker.Check(ctx, target)
		// Always name the endpoint that was actually probed. The address is
		// env-overridable, so a verdict without it cannot be audited — a
		// "healthy" line could refer to a different SonarQube entirely.
		endpointDesc := sonarqubeHealthURL()
		if result.Healthy {
			fmt.Printf("SonarQube: healthy at %s (checked in %v)\n",
				endpointDesc, result.Duration.Round(time.Millisecond))
		} else {
			fmt.Printf("SonarQube: unhealthy at %s — %s\n", endpointDesc, result.Error)
			os.Exit(1)
		}
	case "stop":
		return fmt.Errorf("stop action not yet implemented; use 'make scan-stop' or 'docker-compose -f <file> down' (TODO: wire ComposeOrchestrator.Down())")
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	return nil
}

func handleSnyk(ctx context.Context, projectDir string, rt runtime.ContainerRuntime, action string) error {
	composeFile := filepath.Join(projectDir, "docker", "security", "snyk", "docker-compose.yml")

	snykEp := endpoint.NewEndpoint().
		WithEnabled(true).
		WithRequired(false).
		WithComposeFile(composeFile).
		WithServiceName("snyk-full").
		WithProfile("full").
		WithTimeout(30 * time.Second).
		WithRetryCount(5).
		Build()

	endpoints := map[string]endpoint.ServiceEndpoint{
		"snyk": snykEp,
	}

	checker := health.NewDefaultChecker()
	mgr := boot.NewBootManager(
		endpoints,
		boot.WithRuntime(rt),
		boot.WithHealthChecker(checker),
		boot.WithProjectDir(projectDir),
	)

	switch action {
	case "start":
		log.Println("security-scan: starting Snyk container via containers BootManager...")
		summary, err := mgr.BootAll(ctx)
		if err != nil {
			return fmt.Errorf("BootAll failed: %w", err)
		}
		log.Printf("security-scan: Snyk boot complete — started=%d skipped=%d failed=%d",
			summary.Started, summary.Skipped, summary.Failed)
	case "status":
		fmt.Println("Snyk: container-based (no persistent health endpoint — check docker ps)")
	case "stop":
		return fmt.Errorf("stop action not yet implemented; use 'make scan-stop' or 'docker-compose -f <file> down' (TODO: wire ComposeOrchestrator.Down())")
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	return nil
}

// resolveProjectDir returns the HelixCode project directory.
// When running via `go run ./cmd/security_scan` the working directory is the
// module root; this function validates and returns it.
func resolveProjectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Verify it looks like the HelixCode project root (sanity check)
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return "", fmt.Errorf("go.mod not found in %s — run from HelixCode module root", dir)
	}
	return dir, nil
}
