package main

import (
	"fmt"
	"image/color"
	"os"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Theme represents a UI theme
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

// Available themes
var (
	DarkTheme = Theme{
		Name:       "Dark",
		IsDark:     true,
		Primary:    "#2E86AB",
		Secondary:  "#A23B72",
		Accent:     "#F18F01",
		Text:       "#FFFFFF",
		Background: "#1E1E1E",
		Border:     "#404040",
		Success:    "#4CAF50",
		Warning:    "#FF9800",
		Error:      "#F44336",
		Info:       "#2196F3",
	}

	LightTheme = Theme{
		Name:       "Light",
		IsDark:     false,
		Primary:    "#1976D2",
		Secondary:  "#7B1FA2",
		Accent:     "#FF6F00",
		Text:       "#212121",
		Background: "#FFFFFF",
		Border:     "#BDBDBD",
		Success:    "#4CAF50",
		Warning:    "#FF9800",
		Error:      "#F44336",
		Info:       "#2196F3",
	}

	HelixTheme = Theme{
		Name:       "Helix",
		IsDark:     true,
		Primary:    "#C2E95B",
		Secondary:  "#C0E853",
		Accent:     "#B8ECD7",
		Text:       "#2D3047",
		Background: "#1A1A1A",
		Border:     "#404040",
		Success:    "#4CAF50",
		Warning:    "#FF9800",
		Error:      "#F44336",
		Info:       "#2196F3",
	}

	AuroraTheme = Theme{
		Name:       "Aurora",
		IsDark:     true,
		Primary:    "#00D4FF",
		Secondary:  "#0099CC",
		Accent:     "#FF6B6B",
		Text:       "#FFFFFF",
		Background: "#0F1419",
		Border:     "#1E3A5F",
		Success:    "#00FF88",
		Warning:    "#FFB347",
		Error:      "#FF4757",
		Info:       "#3742FA",
	}
)

// ThemeManager manages UI themes
type ThemeManager struct {
	currentTheme *Theme
	themes       map[string]*Theme
}

// NewThemeManager creates a new theme manager
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		themes: make(map[string]*Theme),
	}

	// Register themes
	tm.themes["dark"] = &DarkTheme
	tm.themes["light"] = &LightTheme
	tm.themes["helix"] = &HelixTheme
	tm.themes["aurora"] = &AuroraTheme

	// Set default theme based on system preference
	tm.currentTheme = tm.detectSystemTheme()

	return tm
}

// detectSystemTheme detects the system's preferred theme
func (tm *ThemeManager) detectSystemTheme() *Theme {
	// Check environment variables
	if theme := os.Getenv("HELIX_THEME"); theme != "" {
		if t, exists := tm.themes[strings.ToLower(theme)]; exists {
			return t
		}
	}

	// For Aurora OS, default to Aurora theme
	if runtime.GOOS == "aurora" {
		return &AuroraTheme
	}

	// Default to helix theme
	return &HelixTheme
}

// GetCurrentTheme returns the current theme
func (tm *ThemeManager) GetCurrentTheme() *Theme {
	return tm.currentTheme
}

// SetTheme sets the current theme
func (tm *ThemeManager) SetTheme(themeName string) bool {
	if theme, exists := tm.themes[strings.ToLower(themeName)]; exists {
		tm.currentTheme = theme
		return true
	}
	return false
}

// GetAvailableThemes returns list of available theme names
func (tm *ThemeManager) GetAvailableThemes() []string {
	names := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		names = append(names, name)
	}
	return names
}

// GetColor returns a color code for the given type
func (tm *ThemeManager) GetColor(colorType string) string {
	theme := tm.currentTheme
	switch strings.ToLower(colorType) {
	case "primary":
		return theme.Primary
	case "secondary":
		return theme.Secondary
	case "accent":
		return theme.Accent
	case "text":
		return theme.Text
	case "background":
		return theme.Background
	case "border":
		return theme.Border
	case "success":
		return theme.Success
	case "warning":
		return theme.Warning
	case "error":
		return theme.Error
	case "info":
		return theme.Info
	default:
		return theme.Text
	}
}

// CustomTheme implements fyne.Theme for HelixCode
type CustomTheme struct {
	currentTheme *Theme
}

// NewCustomTheme creates a new custom theme
func NewCustomTheme() *CustomTheme {
	tm := NewThemeManager()
	return &CustomTheme{
		currentTheme: tm.GetCurrentTheme(),
	}
}

// SetTheme sets the current theme
func (ct *CustomTheme) SetTheme(themeName string) {
	tm := NewThemeManager()
	if tm.SetTheme(themeName) {
		ct.currentTheme = tm.GetCurrentTheme()
	}
}

// PrimaryColor returns the primary color
func (ct *CustomTheme) PrimaryColor() color.Color {
	return parseHexColor(ct.currentTheme.Primary)
}

// HyperlinkColor returns the hyperlink color
func (ct *CustomTheme) HyperlinkColor() color.Color {
	return parseHexColor(ct.currentTheme.Accent)
}

// TextColor returns the text color
func (ct *CustomTheme) TextColor() color.Color {
	return parseHexColor(ct.currentTheme.Text)
}

// BackgroundColor returns the background color
func (ct *CustomTheme) BackgroundColor() color.Color {
	return parseHexColor(ct.currentTheme.Background)
}

// ButtonColor returns the button color
func (ct *CustomTheme) ButtonColor() color.Color {
	return parseHexColor(ct.currentTheme.Secondary)
}

// DisabledButtonColor returns the disabled button color
func (ct *CustomTheme) DisabledButtonColor() color.Color {
	return parseHexColor(ct.currentTheme.Border)
}

// DisabledTextColor returns the disabled text color
func (ct *CustomTheme) DisabledTextColor() color.Color {
	return parseHexColor(ct.currentTheme.Border)
}

// IconColor returns the icon color
func (ct *CustomTheme) IconColor() color.Color {
	return parseHexColor(ct.currentTheme.Text)
}

// DisabledIconColor returns the disabled icon color
func (ct *CustomTheme) DisabledIconColor() color.Color {
	return parseHexColor(ct.currentTheme.Border)
}

// PlaceHolderColor returns the placeholder color
func (ct *CustomTheme) PlaceHolderColor() color.Color {
	return parseHexColor(ct.currentTheme.Border)
}

// PrimaryTextColor returns the primary text color
func (ct *CustomTheme) PrimaryTextColor() color.Color {
	return parseHexColor(ct.currentTheme.Text)
}

// FocusColor returns the focus color
func (ct *CustomTheme) FocusColor() color.Color {
	return parseHexColor(ct.currentTheme.Accent)
}

// ScrollBarColor returns the scrollbar color
func (ct *CustomTheme) ScrollBarColor() color.Color {
	return parseHexColor(ct.currentTheme.Border)
}

// ShadowColor returns the shadow color
func (ct *CustomTheme) ShadowColor() color.Color {
	return color.RGBA{0, 0, 0, 100}
}

// TextSize returns the text size
func (ct *CustomTheme) TextSize() float32 {
	return 14
}

// TextFont returns the text font
func (ct *CustomTheme) TextFont() fyne.Resource {
	return theme.DefaultTextFont()
}

// TextBoldFont returns the bold text font
func (ct *CustomTheme) TextBoldFont() fyne.Resource {
	return theme.DefaultTextBoldFont()
}

// TextItalicFont returns the italic text font
func (ct *CustomTheme) TextItalicFont() fyne.Resource {
	return theme.DefaultTextItalicFont()
}

// TextBoldItalicFont returns the bold italic text font
func (ct *CustomTheme) TextBoldItalicFont() fyne.Resource {
	return theme.DefaultTextBoldItalicFont()
}

// TextMonospaceFont returns the monospace text font
func (ct *CustomTheme) TextMonospaceFont() fyne.Resource {
	return theme.DefaultTextMonospaceFont()
}

// HeadingTextSize returns the heading text size
func (ct *CustomTheme) HeadingTextSize() float32 {
	return 24
}

// Padding returns the padding
func (ct *CustomTheme) Padding() float32 {
	return 4
}

// IconInlineSize returns the inline icon size
func (ct *CustomTheme) IconInlineSize() float32 {
	return 20
}

// ScrollBarSize returns the scrollbar size
func (ct *CustomTheme) ScrollBarSize() float32 {
	return 16
}

// ScrollBarSmallSize returns the small scrollbar size
func (ct *CustomTheme) ScrollBarSmallSize() float32 {
	return 12
}

// parseHexColor parses a hex color string to color.Color
func parseHexColor(hex string) color.Color {
	// Remove # if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	// Parse hex color
	if len(hex) == 6 {
		var r, g, b uint8
		if n, err := parseHexPair(hex[0:2]); err == nil {
			r = n
		}
		if n, err := parseHexPair(hex[2:4]); err == nil {
			g = n
		}
		if n, err := parseHexPair(hex[4:6]); err == nil {
			b = n
		}
		return color.RGBA{r, g, b, 255}
	}

	// Default to black if parsing fails
	return color.Black
}

// Color returns the color for the given theme color name
func (ct *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return parseHexColor(ct.currentTheme.Primary)
	case theme.ColorNameBackground:
		return parseHexColor(ct.currentTheme.Background)
	case theme.ColorNameButton:
		return parseHexColor(ct.currentTheme.Secondary)
	case theme.ColorNameDisabledButton:
		return parseHexColor(ct.currentTheme.Border)
	case theme.ColorNameError:
		return parseHexColor(ct.currentTheme.Error)
	case theme.ColorNameFocus:
		return parseHexColor(ct.currentTheme.Accent)
	case theme.ColorNameForeground:
		return parseHexColor(ct.currentTheme.Text)
	case theme.ColorNameDisabled:
		return parseHexColor(ct.currentTheme.Border)
	case theme.ColorNamePlaceHolder:
		return parseHexColor(ct.currentTheme.Border)
	case theme.ColorNamePressed:
		return parseHexColor(ct.currentTheme.Accent)
	case theme.ColorNameScrollBar:
		return parseHexColor(ct.currentTheme.Border)
	case theme.ColorNameShadow:
		return color.RGBA{0, 0, 0, 100}
	case theme.ColorNameInputBackground:
		if ct.currentTheme.IsDark {
			return parseHexColor("#2A2A2A")
		}
		return parseHexColor("#F5F5F5")
	case theme.ColorNameMenuBackground:
		return parseHexColor(ct.currentTheme.Background)
	case theme.ColorNameOverlayBackground:
		return color.RGBA{0, 0, 0, 150}
	case theme.ColorNameSeparator:
		return parseHexColor(ct.currentTheme.Border)
	default:
		return parseHexColor(ct.currentTheme.Text)
	}
}

// Font returns the font for the given theme font style
func (ct *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Monospace {
		return theme.DefaultTextMonospaceFont()
	}
	if style.Bold {
		if style.Italic {
			return theme.DefaultTextBoldItalicFont()
		}
		return theme.DefaultTextBoldFont()
	}
	if style.Italic {
		return theme.DefaultTextItalicFont()
	}
	return theme.DefaultTextFont()
}

// Size returns the size for the given theme size name
func (ct *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 24
	case theme.SizeNameSubHeadingText:
		return 18
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameScrollBar:
		return 16
	case theme.SizeNameScrollBarSmall:
		return 12
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInlineIcon:
		return 20
	default:
		return 14
	}
}

// Icon returns the icon for the given theme icon name
func (ct *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// parseHexPair parses a hex pair to uint8
func parseHexPair(s string) (uint8, error) {
	var result uint8
	for _, c := range s {
		result <<= 4
		switch {
		case '0' <= c && c <= '9':
			result |= uint8(c - '0')
		case 'a' <= c && c <= 'f':
			result |= uint8(c - 'a' + 10)
		case 'A' <= c && c <= 'F':
			result |= uint8(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}
