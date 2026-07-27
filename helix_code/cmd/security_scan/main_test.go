// cmd/security_scan main_test.go — real, in-package (white-box) tests for the
// HelixCode security-scan bootstrap tool (HXC-123 / F-D1-03).
//
// Anti-bluff discipline (§11.4.6 / §11.4.115): these tests call the actual
// exported/unexported functions declared in main.go — resolveProjectDir,
// handleSonarQube, handleSnyk — against a REAL filesystem (t.TempDir) and,
// for the status-check path, a REAL (loopback) TCP/HTTP round trip. Nothing
// here is reimplemented or mocked; the containers-package types constructed
// along the way (endpoint.ServiceEndpoint, boot.BootManager, health.Checker)
// are the real digital.vasic.containers types the production code uses.
//
// Scope note: the "start" action (mgr.BootAll) is intentionally NOT exercised
// here because it shells out to a real container runtime / compose file that
// does not exist in this test's temp directory and would either fail
// non-deterministically or attempt real container orchestration outside the
// scope of this test suite (see evidence file for the full rationale).
package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ctrruntime "digital.vasic.containers/pkg/runtime"
)

// closedLoopbackPort binds an ephemeral loopback port, records it, and closes
// the listener — yielding a port the kernel just confirmed was bindable and
// which now has nothing listening on it.
//
// Why this exists (§11.4.50 determinism): the unhealthy-path test previously
// relied on the default SonarQube port (9000) being unoccupied. That made the
// verdict depend on ambient host state — it passed on a machine with no
// SonarQube and FAILED the moment the platform was actually running, which is
// exactly backwards for a test asserting the unhealthy branch. Pointing the
// health check at a port we have just proven closed makes the branch
// deterministic on any host, running platform or not, while keeping the real
// TCP round trip this suite is built around.
func closedLoopbackPort(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not bind an ephemeral loopback port: %v", err)
	}
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		ln.Close()
		t.Fatalf("could not parse ephemeral listener address %q: %v", ln.Addr(), err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("could not close ephemeral listener on port %s: %v", port, err)
	}
	return port
}

// ---------------------------------------------------------------------------
// resolveProjectDir
// ---------------------------------------------------------------------------

func TestResolveProjectDir_Success(t *testing.T) {
	dir := t.TempDir()
	goModPath := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module example.test\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("failed to write fixture go.mod: %v", err)
	}

	t.Chdir(dir)

	got, err := resolveProjectDir()
	if err != nil {
		t.Fatalf("resolveProjectDir() returned unexpected error: %v", err)
	}

	// Resolve both sides through EvalSymlinks so /tmp vs /private/tmp (or any
	// other symlinked temp-root) style differences do not produce a false
	// failure on any host.
	wantResolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("failed to resolve expected dir: %v", err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("failed to resolve returned dir %q: %v", got, err)
	}
	if gotResolved != wantResolved {
		t.Fatalf("resolveProjectDir() = %q (resolved %q), want %q", got, gotResolved, wantResolved)
	}
}

func TestResolveProjectDir_MissingGoMod(t *testing.T) {
	dir := t.TempDir() // deliberately no go.mod written here

	t.Chdir(dir)

	got, err := resolveProjectDir()
	if err == nil {
		t.Fatalf("resolveProjectDir() = %q, nil; want an error because go.mod is absent", got)
	}
	if !strings.Contains(err.Error(), "go.mod not found") {
		t.Fatalf("resolveProjectDir() error = %q; want it to mention %q", err.Error(), "go.mod not found")
	}
}

// ---------------------------------------------------------------------------
// handleSonarQube / handleSnyk — "stop" action (fixed not-implemented error)
// ---------------------------------------------------------------------------

func TestHandleSonarQube_StopAction_ReturnsNotImplementedError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rt ctrruntime.ContainerRuntime // nil: never invoked on the "stop" path (verified below)
	err := handleSonarQube(ctx, t.TempDir(), rt, "stop")
	if err == nil {
		t.Fatal("handleSonarQube(..., \"stop\") returned nil error; want the not-yet-implemented error")
	}
	if !strings.Contains(err.Error(), "stop action not yet implemented") {
		t.Fatalf("handleSonarQube(..., \"stop\") error = %q; want it to mention %q", err.Error(), "stop action not yet implemented")
	}
}

func TestHandleSnyk_StopAction_ReturnsNotImplementedError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rt ctrruntime.ContainerRuntime
	err := handleSnyk(ctx, t.TempDir(), rt, "stop")
	if err == nil {
		t.Fatal("handleSnyk(..., \"stop\") returned nil error; want the not-yet-implemented error")
	}
	if !strings.Contains(err.Error(), "stop action not yet implemented") {
		t.Fatalf("handleSnyk(..., \"stop\") error = %q; want it to mention %q", err.Error(), "stop action not yet implemented")
	}
}

// ---------------------------------------------------------------------------
// handleSonarQube / handleSnyk — unknown action (default branch)
// ---------------------------------------------------------------------------

func TestHandleSonarQube_UnknownAction_ReturnsError(t *testing.T) {
	cases := []string{"", "bogus", "START", "Status", "  start", "stop "}
	for _, action := range cases {
		action := action
		t.Run("action="+action, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var rt ctrruntime.ContainerRuntime
			err := handleSonarQube(ctx, t.TempDir(), rt, action)
			if err == nil {
				t.Fatalf("handleSonarQube(..., %q) returned nil error; want unknown-action error", action)
			}
			want := "unknown action \"" + action + "\""
			if err.Error() != want {
				t.Fatalf("handleSonarQube(..., %q) error = %q, want %q", action, err.Error(), want)
			}
		})
	}
}

func TestHandleSnyk_UnknownAction_ReturnsError(t *testing.T) {
	cases := []string{"", "bogus", "START", "Status", "  start", "stop "}
	for _, action := range cases {
		action := action
		t.Run("action="+action, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var rt ctrruntime.ContainerRuntime
			err := handleSnyk(ctx, t.TempDir(), rt, action)
			if err == nil {
				t.Fatalf("handleSnyk(..., %q) returned nil error; want unknown-action error", action)
			}
			want := "unknown action \"" + action + "\""
			if err.Error() != want {
				t.Fatalf("handleSnyk(..., %q) error = %q, want %q", action, err.Error(), want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// handleSnyk — "status" action (real stdout write path, no I/O beyond that)
// ---------------------------------------------------------------------------

func TestHandleSnyk_StatusAction_PrintsMessageAndReturnsNil(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stdout := captureStdout(t, func() {
		var rt ctrruntime.ContainerRuntime
		if err := handleSnyk(ctx, t.TempDir(), rt, "status"); err != nil {
			t.Fatalf("handleSnyk(..., \"status\") returned unexpected error: %v", err)
		}
	})

	if !strings.Contains(stdout, "Snyk") {
		t.Fatalf("handleSnyk status output = %q; want it to mention %q", stdout, "Snyk")
	}
}

// ---------------------------------------------------------------------------
// handleSonarQube — "status" action, unhealthy path: real HTTP round trip to
// a loopback address nothing listens on, real os.Exit(1). Exercised via a
// subprocess re-exec of this same test binary (the standard Go
// TestHelperProcess pattern used by package os/exec's own test suite) so the
// exit call does not terminate the primary `go test` process.
// ---------------------------------------------------------------------------

const helperProcessEnvVar = "SECURITY_SCAN_TEST_HELPER_PROC"

func TestHelperProcess_SonarQubeStatusUnhealthy(t *testing.T) {
	if os.Getenv(helperProcessEnvVar) != "sonarqube_status_unhealthy" {
		// Not invoked as a helper process during a normal `go test` run —
		// intentionally a no-op return, NOT t.Skip(), so this never shows up
		// as a skipped test (per §11.4 no-silent-skip discipline: this guard
		// is the standard Go re-exec pattern, not an unimplemented feature).
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dir, err := os.MkdirTemp("", "security-scan-helper-*")
	if err != nil {
		os.Exit(90)
	}
	defer os.RemoveAll(dir)

	var rt ctrruntime.ContainerRuntime // nil: the "status" branch never touches rt or mgr
	err = handleSonarQube(ctx, dir, rt, "status")
	if err != nil {
		// handleSonarQube's "status" branch never returns a non-nil error in
		// the current implementation (it os.Exit(1)s on unhealthy instead) —
		// reaching this branch means the contract changed under us.
		os.Exit(77)
	}
	os.Exit(0)
}

func TestHandleSonarQube_StatusAction_Unhealthy_ExitsWithCode1(t *testing.T) {
	deadPort := closedLoopbackPort(t)

	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess_SonarQubeStatusUnhealthy$", "-test.v=true")
	cmd.Env = append(os.Environ(),
		helperProcessEnvVar+"=sonarqube_status_unhealthy",
		// Point the health check at a port we just proved is closed, so the
		// unhealthy branch is exercised regardless of whether a real
		// SonarQube is running on this host.
		envSonarqubeHost+"=127.0.0.1",
		envSonarqubePort+"="+deadPort,
	)

	out, runErr := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("expected helper subprocess to exit non-zero (unhealthy SonarQube on closed port 127.0.0.1:%s); "+
			"got err=%v output=%s", deadPort, runErr, out)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 (handleSonarQube status/unhealthy path), got %d; output=%s",
			exitErr.ExitCode(), out)
	}
	if !strings.Contains(string(out), "SonarQube: unhealthy") {
		t.Fatalf("expected helper output to contain %q; got: %s", "SonarQube: unhealthy", out)
	}
}

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

// captureStdout redirects os.Stdout for the duration of fn and returns
// whatever fn wrote to it. Used to assert on real fmt.Print* output from the
// functions under test without touching any production code.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe for stdout capture: %v", err)
	}
	os.Stdout = w

	done := make(chan struct{})
	var captured strings.Builder
	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				captured.Write(buf[:n])
			}
			if readErr != nil {
				break
			}
		}
		close(done)
	}()

	fn()

	os.Stdout = orig
	_ = w.Close()
	<-done
	_ = r.Close()

	return captured.String()
}
