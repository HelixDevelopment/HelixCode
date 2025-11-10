package main

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Theme represents a color theme configuration
type Theme struct {
	Name       string
	IsDark     bool
	Primary    string
	Secondary  string
	Accent     string
	Text       string
	Background string
	Border     string
	Success    string
	Warning    string
	Error      string
	Info       string
}

// Predefined themes
var (
	// DarkTheme is the default dark theme
	DarkTheme = Theme{
		Name:       "Dark",
		IsDark:     true,
		Primary:    "#3E4E5E",
		Secondary:  "#2A3A4A",
		Accent:     "#5E6E7E",
		Text:       "#FFFFFF",
		Background: "#1E2E3E",
		Border:     "#4E5E6E",
		Success:    "#4CAF50",
		Warning:    "#FFC107",
		Error:      "#F44336",
		Info:       "#2196F3",
	}

	// LightTheme is the default light theme
	LightTheme = Theme{
		Name:       "Light",
		IsDark:     false,
		Primary:    "#2196F3",
		Secondary:  "#64B5F6",
		Accent:     "#1976D2",
		Text:       "#212121",
		Background: "#FAFAFA",
		Border:     "#BDBDBD",
		Success:    "#4CAF50",
		Warning:    "#FFC107",
		Error:      "#F44336",
		Info:       "#2196F3",
	}

	// HelixTheme is the HelixCode branded theme
	HelixTheme = Theme{
		Name:       "Helix",
		IsDark:     true,
		Primary:    "#6B46C1",
		Secondary:  "#8B5CF6",
		Accent:     "#A78BFA",
		Text:       "#F3F4F6",
		Background: "#1F2937",
		Border:     "#4B5563",
		Success:    "#10B981",
		Warning:    "#F59E0B",
		Error:      "#EF4444",
		Info:       "#3B82F6",
	}

	// HarmonyTheme is the Harmony OS-specific theme with warm colors
	HarmonyTheme = Theme{
		Name:       "Harmony",
		IsDark:     true,
		Primary:    "#FF6B35", // Warm orange
		Secondary:  "#F7931E", // Golden orange
		Accent:     "#FDB462", // Light amber
		Text:       "#FFFFFF",
		Background: "#1A1512", // Dark warm brown
		Border:     "#3D2A1F", // Medium warm brown
		Success:    "#52C41A", // Green
		Warning:    "#FAAD14", // Amber
		Error:      "#FF4D4F", // Red
		Info:       "#1890FF", // Blue
	}
)

// CustomTheme implements the fyne.Theme interface with customizable colors
type CustomTheme struct {
	currentTheme *Theme
}

// NewCustomTheme creates a new custom theme
func NewCustomTheme(t *Theme) *CustomTheme {
	return &CustomTheme{
		currentTheme: t,
	}
}

// Color returns the color for the specified theme color name
func (ct *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	// Primary colors
	case theme.ColorNamePrimary:
		return parseHexColor(ct.currentTheme.Primary)
	case theme.ColorNameBackground:
		return parseHexColor(ct.currentTheme.Background)
	case theme.ColorNameForeground:
		return parseHexColor(ct.currentTheme.Text)

	// Button colors
	case theme.ColorNameButton:
		return parseHexColor(ct.currentTheme.Primary)
	case theme.ColorNameDisabled:
		return color.RGBA{128, 128, 128, 255}
	case theme.ColorNameFocus:
		return parseHexColor(ct.currentTheme.Accent)
	case theme.ColorNameHover:
		return ct.lighten(parseHexColor(ct.currentTheme.Primary), 0.1)
	case theme.ColorNamePressed:
		return ct.darken(parseHexColor(ct.currentTheme.Primary), 0.1)

	// Text colors
	case theme.ColorNameDisabledButton:
		return color.RGBA{100, 100, 100, 255}
	case theme.ColorNamePlaceHolder:
		return color.RGBA{150, 150, 150, 255}

	// Input colors
	case theme.ColorNameInputBackground:
		return ct.darken(parseHexColor(ct.currentTheme.Background), 0.05)
	case theme.ColorNameInputBorder:
		return parseHexColor(ct.currentTheme.Border)

	// Selection colors
	case theme.ColorNameSelection:
		return parseHexColor(ct.currentTheme.Accent)

	// Scrollbar colors
	case theme.ColorNameScrollBar:
		return parseHexColor(ct.currentTheme.Secondary)

	// Shadow colors
	case theme.ColorNameShadow:
		return color.RGBA{0, 0, 0, 100}

	// Menu colors
	case theme.ColorNameMenuBackground:
		return parseHexColor(ct.currentTheme.Background)

	// Overlay colors
	case theme.ColorNameOverlayBackground:
		return color.RGBA{0, 0, 0, 180}

	// Status colors
	case theme.ColorNameSuccess:
		return parseHexColor(ct.currentTheme.Success)
	case theme.ColorNameWarning:
		return parseHexColor(ct.currentTheme.Warning)
	case theme.ColorNameError:
		return parseHexColor(ct.currentTheme.Error)

	// Separator colors
	case theme.ColorNameSeparator:
		return parseHexColor(ct.currentTheme.Border)

	// Header colors
	case theme.ColorNameHeaderBackground:
		return ct.darken(parseHexColor(ct.currentTheme.Background), 0.1)

	// Hyperlink colors
	case theme.ColorNameHyperlink:
		return parseHexColor(ct.currentTheme.Info)

	default:
		// Fallback to default theme
		if ct.currentTheme.IsDark {
			return theme.DefaultTheme().Color(name, theme.VariantDark)
		}
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
}

// Font returns the font resource for the specified text style
func (ct *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Size returns the size for the specified theme size name
func (ct *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNameScrollBar:
		return 12
	case theme.SizeNameScrollBarSmall:
		return 8
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameInputBorder:
		return 1
	default:
		return theme.DefaultTheme().Size(name)
	}
}

// Icon returns the icon resource for the specified theme icon name
func (ct *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// parseHexColor converts a hex color string to color.Color
func parseHexColor(hex string) color.Color {
	if len(hex) == 0 {
		return color.Black
	}

	// Remove # prefix if present
	if hex[0] == '#' {
		hex = hex[1:]
	}

	// Ensure we have a valid length
	if len(hex) != 6 {
		return color.Black
	}

	// Parse RGB components
	r, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return color.Black
	}

	g, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return color.Black
	}

	b, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return color.Black
	}

	return color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: 255,
	}
}

// lighten makes a color lighter by the specified factor (0.0-1.0)
func (ct *CustomTheme) lighten(c color.Color, factor float32) color.Color {
	r, g, b, a := c.RGBA()

	// Convert to 0-255 range
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)
	a8 := uint8(a >> 8)

	// Lighten
	r8 = uint8(float32(r8) + (255-float32(r8))*factor)
	g8 = uint8(float32(g8) + (255-float32(g8))*factor)
	b8 = uint8(float32(b8) + (255-float32(b8))*factor)

	return color.RGBA{R: r8, G: g8, B: b8, A: a8}
}

// darken makes a color darker by the specified factor (0.0-1.0)
func (ct *CustomTheme) darken(c color.Color, factor float32) color.Color {
	r, g, b, a := c.RGBA()

	// Convert to 0-255 range
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)
	a8 := uint8(a >> 8)

	// Darken
	r8 = uint8(float32(r8) * (1 - factor))
	g8 = uint8(float32(g8) * (1 - factor))
	b8 = uint8(float32(b8) * (1 - factor))

	return color.RGBA{R: r8, G: g8, B: b8, A: a8}
}

// ThemeManager manages theme switching
type ThemeManager struct {
	themes       map[string]*Theme
	currentTheme string
	customTheme  *CustomTheme
}

// NewThemeManager creates a new theme manager
func NewThemeManager() *ThemeManager {
	themes := map[string]*Theme{
		"Dark":    &DarkTheme,
		"Light":   &LightTheme,
		"Helix":   &HelixTheme,
		"Harmony": &HarmonyTheme,
	}

	return &ThemeManager{
		themes:       themes,
		currentTheme: "Harmony", // Default to Harmony theme
		customTheme:  NewCustomTheme(&HarmonyTheme),
	}
}

// SetTheme sets the current theme by name
func (tm *ThemeManager) SetTheme(name string) {
	if t, ok := tm.themes[name]; ok {
		tm.currentTheme = name
		tm.customTheme = NewCustomTheme(t)
	}
}

// GetCurrentTheme returns the current theme
func (tm *ThemeManager) GetCurrentTheme() *Theme {
	return tm.themes[tm.currentTheme]
}

// GetCustomTheme returns the custom theme implementation
func (tm *ThemeManager) GetCustomTheme() fyne.Theme {
	return tm.customTheme
}

// GetAvailableThemes returns a list of available theme names
func (tm *ThemeManager) GetAvailableThemes() []string {
	themes := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		themes = append(themes, name)
	}
	return themes
}

// AddTheme adds a custom theme to the manager
func (tm *ThemeManager) AddTheme(name string, t *Theme) {
	tm.themes[name] = t
}

// RemoveTheme removes a theme from the manager
func (tm *ThemeManager) RemoveTheme(name string) {
	// Prevent removing built-in themes
	if name == "Dark" || name == "Light" || name == "Helix" || name == "Harmony" {
		return
	}
	delete(tm.themes, name)
}
