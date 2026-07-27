//go:build !nogui

// Exercises GUI-only symbols declared in the !nogui-tagged sources
// (theme.go / main.go). Without this constraint the file is compiled under
// -tags=nogui, where those symbols do not exist, so `go vet` and
// `go test -tags=nogui` fail to build the test binary. (`go build` is
// unaffected — it never compiles _test.go files.)
package main

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// hxc_theme_brand_test.go — anti-bluff guard (§11.4 / §11.4.135) for the
// HelixCode brand dark theme derived from assets/Logo.png (the lime-green→teal
// nautilus spiral). It asserts the CustomTheme.Color mapping returns the EXACT
// FACT brand palette for the brand-relevant Fyne ColorNames, in the DARK
// variant, regardless of the host OS appearance setting.
//
// RED-then-GREEN: written against the brand palette; before the brand mapping
// landed in theme.go these assertions FAIL (the legacy mapping returned the old
// blue/grey approximation), after it they PASS. A regression that reverts the
// palette flips these red again.

// nrgba parses a brand FACT hex (RRGGBB) into the color.NRGBA the brand mapping
// emits, so the test compares like-for-like with parseHexColor's output.
func nrgba(t *testing.T, hex string) color.NRGBA {
	t.Helper()
	c := parseHexColor(hex)
	got, ok := c.(color.NRGBA)
	if !ok {
		t.Fatalf("parseHexColor(%q) = %T, want color.NRGBA", hex, c)
	}
	return got
}

// brandTheme builds a CustomTheme pinned to the brand HelixTheme so the test is
// independent of HELIX_THEME env state.
func brandTheme() *CustomTheme {
	ht := HelixTheme
	return &CustomTheme{currentTheme: &ht}
}

func TestBrandTheme_PaletteMapping(t *testing.T) {
	ct := brandTheme()

	if !ct.currentTheme.IsBrand {
		t.Fatalf("HelixTheme.IsBrand = false, want true (brand mapping would not engage)")
	}

	cases := []struct {
		name fyne.ThemeColorName
		hex  string
		what string
	}{
		{theme.ColorNamePrimary, "#A8DD22", "PRIMARY (lime)"},
		{theme.ColorNameFocus, "#A8DD22", "PRIMARY focus ring"},
		{theme.ColorNameBackground, "#0E1310", "BG_BASE"},
		{theme.ColorNameForeground, "#ECF3E8", "FG_TEXT"},
		{theme.ColorNameButton, "#202A22", "BG_RAISED"},
		{theme.ColorNameInputBackground, "#18201A", "BG_SURFACE"},
		{theme.ColorNameHyperlink, "#8FC9B8", "SECONDARY (teal)"},
		{theme.ColorNamePlaceHolder, "#9DB0A0", "FG_MUTED placeholder"},
		{theme.ColorNameDisabled, "#9DB0A0", "FG_MUTED disabled"},
		{theme.ColorNameSelection, "#7FA81B", "PRIMARY_DIM selection"},
		{theme.ColorNamePressed, "#7FA81B", "PRIMARY_DIM pressed"},
		{theme.ColorNameSeparator, "#2A352C", "BORDER separator"},
		{theme.ColorNameShadow, "#2A352C", "BORDER shadow"},
		{theme.ColorNameError, "#E06A5A", "ERROR"},
		{theme.ColorNameSuccess, "#A8DD22", "SUCCESS (lime)"},
		{theme.ColorNameWarning, "#E0C040", "WARNING"},
	}

	for _, c := range cases {
		got := ct.Color(c.name, theme.VariantDark)
		want := nrgba(t, c.hex)
		gotN, ok := got.(color.NRGBA)
		if !ok {
			t.Errorf("%s: Color(%v) = %T, want color.NRGBA", c.what, c.name, got)
			continue
		}
		if gotN != want {
			t.Errorf("%s: Color(%v) = %+v, want %+v (%s)", c.what, c.name, gotN, want, c.hex)
		}
	}
}

// TestBrandTheme_DarkRegardlessOfVariant proves the brand mapping ignores the
// Fyne variant — the brand identity is the same whether the host requests
// light or dark, so BG_BASE is returned for both.
func TestBrandTheme_DarkRegardlessOfVariant(t *testing.T) {
	ct := brandTheme()
	want := nrgba(t, "#0E1310")
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		got, ok := ct.Color(theme.ColorNameBackground, v).(color.NRGBA)
		if !ok || got != want {
			t.Errorf("Color(Background, variant=%d) = %+v, want %+v (brand is always dark)", v, got, want)
		}
	}
}

// TestBrandTheme_OnPrimaryForegroundIsDark proves text on the bright lime
// primary/success/warning accents is the near-black base for legibility.
func TestBrandTheme_OnPrimaryForegroundIsDark(t *testing.T) {
	ct := brandTheme()
	want := nrgba(t, "#0E1310")
	for _, n := range []fyne.ThemeColorName{
		theme.ColorNameForegroundOnPrimary,
		theme.ColorNameForegroundOnSuccess,
		theme.ColorNameForegroundOnWarning,
	} {
		got, ok := ct.Color(n, theme.VariantDark).(color.NRGBA)
		if !ok || got != want {
			t.Errorf("Color(%v) = %+v, want near-black base %+v", n, got, want)
		}
	}
}

// TestBrandTheme_DefaultIsBrand proves the theme manager defaults to the brand
// theme (no HELIX_THEME override) so the GUI launches in brand identity.
func TestBrandTheme_DefaultIsBrand(t *testing.T) {
	t.Setenv("HELIX_THEME", "") // ensure no operator override
	tm := NewThemeManager()
	cur := tm.GetCurrentTheme()
	if !cur.IsBrand {
		t.Fatalf("default theme %q IsBrand = false, want the brand theme as default", cur.Name)
	}
}
