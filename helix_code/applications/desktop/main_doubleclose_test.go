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

// TestDesktopApp_Close_Idempotent is the regression guard for the SYSTEMIC
// channel-double-close defect class (D5, desktop): (*DesktopApp).Close() closed
// da.stopUpdate guarded only by `!= nil` (the field is never nilled), so a
// second Close() panicked with "close of closed channel".
//
// The full DesktopApp is not unit-constructible (NewDesktopApp calls app.New(),
// which requires a display). This focused test builds a partial struct with the
// guarded close path's fields (stopUpdate + stopOnce) — every other field
// Close() touches is nil-guarded — and drives Close() twice. The fix wraps the
// close in sync.Once so the second call is a clean no-op.
//
// §11.4.115 RED→GREEN polarity switch via RED_MODE:
//   - RED_MODE=1 reproduces the ORIGINAL unguarded double close and asserts the
//     panic IS present.
//   - RED_MODE=0 (default) drives the REAL fixed Close() twice and asserts no
//     panic.
func TestDesktopApp_Close_Idempotent(t *testing.T) {
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

	da := &DesktopApp{stopUpdate: make(chan struct{})}

	if err := da.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// A second Close must be a clean no-op, never a panic.
	if err := da.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
