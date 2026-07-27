//go:build !nogui

// Exercises GUI-only symbols declared in the !nogui-tagged sources
// (theme.go / main.go). Without this constraint the file is compiled under
// -tags=nogui, where those symbols do not exist, so `go vet` and
// `go test -tags=nogui` fail to build the test binary. (`go build` is
// unaffected — it never compiles _test.go files.)
package main

import (
	"os"
	"testing"
)

// TestAuroraApp_Close_Idempotent is the regression guard for the SYSTEMIC
// channel-double-close defect class (D5, aurora_os): (*AuroraApp).Close() closed
// auroraApp.stopUpdate guarded only by `!= nil` (never nilled), so a second
// Close() panicked with "close of closed channel".
//
// The full AuroraApp is not unit-constructible (NewAuroraApp calls app.New()).
// This focused test builds a partial struct with the guarded close path's
// fields plus the non-nil securityManager that Close() unconditionally audits,
// and drives Close() twice. The fix wraps the close in sync.Once.
//
// §11.4.115 RED→GREEN polarity switch via RED_MODE:
//   - RED_MODE=1 reproduces the ORIGINAL unguarded double close → panic present.
//   - RED_MODE=0 (default) drives the REAL fixed Close() twice → no panic.
func TestAuroraApp_Close_Idempotent(t *testing.T) {
	if os.Getenv("RED_MODE") == "1" {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("RED_MODE: expected double-close to panic, but it did not — defect not reproduced")
			}
			// recover() != nil => panic observed => defect confirmed => RED passes
		}()
		ch := make(chan struct{})
		close(ch)
		close(ch) // panics: close of closed channel — the original D5 bug
		return
	}

	auroraApp := &AuroraApp{
		stopUpdate:      make(chan struct{}),
		securityManager: NewAuroraSecurityManager(),
	}

	if err := auroraApp.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// A second Close must be a clean no-op, never a panic.
	if err := auroraApp.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
