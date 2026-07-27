//go:build !nogui

// Exercises GUI-only symbols declared in the !nogui-tagged sources
// (theme.go / main.go). Without this constraint the file is compiled under
// -tags=nogui, where those symbols do not exist, so `go vet` and
// `go test -tags=nogui` fail to build the test binary. (`go build` is
// unaffected — it never compiles _test.go files.)
package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// hxc_theme_crash_test.go — regression guard (§11.4.135) for the desktop GUI
// launch crash: NewDesktopApp set the Fyne theme to a ZERO-VALUE &CustomTheme{}
// whose currentTheme was nil, so Fyne's canvas color lookups during window
// creation nil-deref'd (SIGSEGV on launch — the GUI never rendered). Fixed by
// using NewCustomTheme() in main.go + a nil-guard in CustomTheme.Color().
//
// This test reproduces the exact crash construction (zero-value &CustomTheme{})
// and asserts Color() does NOT panic and returns a real color for every name
// Fyne queries during canvas creation. Pre-fix this PANICs; post-fix it passes.
func TestCustomTheme_ZeroValue_ColorNoPanic(t *testing.T) {
	ct := &CustomTheme{} // nil currentTheme — the precise launch-crash construction
	names := []fyne.ThemeColorName{
		theme.ColorNamePrimary, theme.ColorNameBackground, theme.ColorNameForeground,
		theme.ColorNameButton, theme.ColorNameInputBackground, theme.ColorNameShadow,
		theme.ColorNameError, theme.ColorNameFocus, theme.ColorNameSeparator,
	}
	for _, n := range names {
		c := ct.Color(n, theme.VariantDark)
		if c == nil {
			t.Fatalf("CustomTheme{}.Color(%v) returned nil — must yield a real color", n)
		}
	}
}
