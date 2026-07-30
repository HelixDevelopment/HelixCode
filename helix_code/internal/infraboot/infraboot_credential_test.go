package infraboot

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMain supplies a TEST-ONLY Postgres credential for the whole package.
//
// Reconciliation note (§11.4.120): before HXC-168 the package compiled a
// literal default password into the binary, so every EnsureInfra test reached
// the boot path with no credential configured. Removing that literal (CONST-042)
// correctly made those tests fail — the FAIL was the signal the fix landed, not
// a regression. The gate is therefore reconciled to assert the NEW mechanism
// (credential comes from the environment) rather than weakened: the value below
// is a throwaway fixture that exists only inside `go test`, and the fail-closed
// behaviour it bypasses is asserted explicitly by
// TestResolvePgPassword_FailsClosedWhenUnset below.
func TestMain(m *testing.M) {
	if err := os.Setenv(envDatabasePassword, "test-only-not-a-real-credential"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// TestResolvePgPassword_FailsClosedWhenUnset is the HXC-168 regression guard
// (§11.4.135). It pins the security property that replaced the published
// literal: with no credential configured, infraboot MUST refuse to boot rather
// than silently fall back to a compiled-in default.
//
// Falsifiability: reintroducing any literal fallback in resolvePgPassword makes
// this test FAIL, because the call would return a password instead of an error.
func TestResolvePgPassword_FailsClosedWhenUnset(t *testing.T) {
	t.Setenv(envAutobootPgPassword, "")
	t.Setenv(envDatabasePassword, "")

	pw, err := resolvePgPassword()
	if err == nil {
		t.Fatalf("resolvePgPassword must fail closed when no credential is configured, got password of length %d", len(pw))
	}
	if pw != "" {
		t.Fatalf("no password may be returned on the error path, got a non-empty value of length %d", len(pw))
	}
	// The message must be actionable: it has to name the variable to set and
	// point at the private configuration source (.env), never leak a value.
	for _, want := range []string{envDatabasePassword, envAutobootPgPassword, ".env"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message must mention %q so the operator can act on it; got: %v", want, err)
		}
	}
}

// TestEnsureInfra_FailsClosedWhenNoCredential proves the fail-closed property
// holds at the exported entry point, and — critically — that NO container is
// booted when the credential is missing. Booting infra with an unset password
// is exactly the outcome HXC-168 forbids.
func TestEnsureInfra_FailsClosedWhenNoCredential(t *testing.T) {
	t.Setenv(envAutobootPgPassword, "")
	t.Setenv(envDatabasePassword, "")

	cfg := freshCfg()
	b := &fakeBooter{runtimeAvail: true, runtimeName: "podman"}

	_, err := EnsureInfra(context.Background(), cfg, &Options{Booter: b})
	if err == nil {
		t.Fatal("EnsureInfra must return an error when no credential is configured")
	}
	if !errors.Is(err, errNoPgPassword) {
		t.Fatalf("expected the credential error, got: %v", err)
	}
	if b.upCalls() != 0 {
		t.Fatalf("compose up must NOT run without a credential, got %d calls", b.upCalls())
	}
	if cfg.Database.Host != "postgres" {
		t.Fatalf("config must be left untouched on the error path, got host=%s", cfg.Database.Host)
	}
}

// TestResolvePgPassword_ExportsForComposeChild proves the load-bearing wiring:
// the autoboot compose file reads ${HELIX_AUTOBOOT_PG_PASSWORD}, so the value
// resolved in-process MUST be exported under that name before `compose up`, or
// the container and the in-process config would disagree and every query would
// fail authentication.
func TestResolvePgPassword_ExportsForComposeChild(t *testing.T) {
	t.Setenv(envAutobootPgPassword, "")
	t.Setenv(envDatabasePassword, "from-dot-env-fixture")

	pw, err := resolvePgPassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw != "from-dot-env-fixture" {
		t.Fatalf("expected the .env value to be used, got a different value of length %d", len(pw))
	}
	if got := os.Getenv(envAutobootPgPassword); got != "from-dot-env-fixture" {
		t.Fatalf("%s must be exported for the compose child process, got a value of length %d", envAutobootPgPassword, len(got))
	}
}

// TestResolvePgPassword_AutobootOverrideWins pins the documented precedence:
// the auto-boot-specific variable overrides the project-wide one, matching the
// HELIX_AUTOBOOT_PG_PORT / HELIX_AUTOBOOT_REDIS_PORT override convention.
func TestResolvePgPassword_AutobootOverrideWins(t *testing.T) {
	t.Setenv(envAutobootPgPassword, "autoboot-specific-fixture")
	t.Setenv(envDatabasePassword, "project-wide-fixture")

	pw, err := resolvePgPassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw != "autoboot-specific-fixture" {
		t.Fatalf("the auto-boot-specific override must win over the project-wide variable")
	}
}

// TestRewriteConfig_UsesResolvedCredential proves the resolved secret actually
// reaches the in-process config — the step that makes the booted container
// usable. Before HXC-168 this assigned a compiled-in literal.
func TestRewriteConfig_UsesResolvedCredential(t *testing.T) {
	cfg := freshCfg()
	rewriteConfig(cfg, 55432, 56379, "resolved-fixture-value")

	if cfg.Database.Password != "resolved-fixture-value" {
		t.Fatalf("rewriteConfig must apply the resolved credential, not a built-in default")
	}
	if cfg.Database.Host != bootHost || cfg.Database.Port != 55432 {
		t.Fatalf("endpoint rewrite regressed: host=%s port=%d", cfg.Database.Host, cfg.Database.Port)
	}
}

// TestNoCredentialLiteralInPackageSource is a source-level guard mirroring the
// repository gate (scripts/gates/no_hardcoded_db_credential_gate.sh) at the
// package boundary, so a reintroduced literal fails `go test` even if the shell
// gate is not run. Kept deliberately narrow: it scans only this package's
// non-test sources.
func TestNoCredentialLiteralInPackageSource(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("cannot read package dir: %v", err)
	}
	// Assembled at runtime so this guard does not itself contain the literal.
	banned := "helix" + "pass"

	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("cannot read %s: %v", name, rerr)
		}
		scanned++
		if strings.Contains(string(body), banned) {
			t.Errorf("%s contains a hardcoded database credential literal (CONST-042 / HXC-168); source it from the environment instead", name)
		}
	}
	if scanned == 0 {
		t.Fatal("guard scanned no files — it would pass vacuously; check the package layout")
	}
}

// compile-time reminder that the health budget default is still a duration.
var _ = time.Second
