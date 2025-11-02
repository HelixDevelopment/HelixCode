package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
)

func TestNewDesktopApp(t *testing.T) {
	app := NewDesktopApp()
	assert.NotNil(t, app)
	assert.NotNil(t, app.fyneApp)
}

func TestNewThemeManager(t *testing.T) {
	tm := NewThemeManager()
	assert.NotNil(t, tm)
	assert.NotNil(t, tm.currentTheme)
	assert.NotEmpty(t, tm.themes)
}

func TestThemeManager_GetAvailableThemes(t *testing.T) {
	tm := NewThemeManager()
	themes := tm.GetAvailableThemes()
	assert.Contains(t, themes, "dark")
	assert.Contains(t, themes, "light")
	assert.Contains(t, themes, "helix")
}

func TestThemeManager_SetTheme(t *testing.T) {
	tm := NewThemeManager()

	// Test valid theme
	assert.True(t, tm.SetTheme("light"))
	assert.Equal(t, "Light", tm.GetCurrentTheme().Name)

	// Test invalid theme
	assert.False(t, tm.SetTheme("invalid"))
}

func TestThemeManager_GetColor(t *testing.T) {
	tm := NewThemeManager()

	// Test valid color types
	assert.NotEmpty(t, tm.GetColor("primary"))
	assert.NotEmpty(t, tm.GetColor("text"))
	assert.NotEmpty(t, tm.GetColor("background"))

	// Test invalid color type (should return text color)
	assert.NotEmpty(t, tm.GetColor("invalid"))
}

func TestNewCustomTheme(t *testing.T) {
	theme := NewCustomTheme()
	assert.NotNil(t, theme)
	assert.NotNil(t, theme.currentTheme)
}

func TestCustomTheme_Color(t *testing.T) {
	ct := NewCustomTheme()

	// Test that colors are returned (exact color values depend on theme)
	color := ct.Color(theme.ColorNamePrimary, theme.VariantDark)
	assert.NotNil(t, color)
}

func TestCustomTheme_Size(t *testing.T) {
	ct := NewCustomTheme()

	// Test text size
	size := ct.Size(theme.SizeNameText)
	assert.Greater(t, size, float32(0))
}

func TestCustomTheme_Font(t *testing.T) {
	ct := NewCustomTheme()

	// Test font retrieval
	font := ct.Font(fyne.TextStyle{})
	assert.NotNil(t, font)
}

func TestCustomTheme_Icon(t *testing.T) {
	ct := NewCustomTheme()

	// Test icon retrieval
	icon := ct.Icon(theme.IconNameHome)
	assert.NotNil(t, icon)
}

func TestParseHexColor(t *testing.T) {
	// Test valid hex color
	color := parseHexColor("#FF0000")
	r, g, b, a := color.RGBA()
	assert.Equal(t, uint32(65535), r) // Full red
	assert.Equal(t, uint32(0), g)     // No green
	assert.Equal(t, uint32(0), b)     // No blue
	assert.Equal(t, uint32(65535), a) // Full alpha

	// Test hex color without #
	color2 := parseHexColor("00FF00")
	_, g2, _, _ := color2.RGBA()
	assert.Equal(t, uint32(65535), g2) // Full green
}

func TestParseHexPair(t *testing.T) {
	// Test valid hex pairs
	val1, err1 := parseHexPair("FF")
	assert.NoError(t, err1)
	assert.Equal(t, uint8(255), val1)

	val2, err2 := parseHexPair("00")
	assert.NoError(t, err2)
	assert.Equal(t, uint8(0), val2)

	val3, err3 := parseHexPair("A5")
	assert.NoError(t, err3)
	assert.Equal(t, uint8(165), val3)

	// Test invalid hex pair
	_, err4 := parseHexPair("GG")
	assert.Error(t, err4)
}
