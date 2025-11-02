package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/rivo/tview"
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

	// Check system preference (simplified - in real implementation would check OS settings)
	if runtime.GOOS == "darwin" {
		// On macOS, could check defaults read -g AppleInterfaceStyle
		// For now, default to dark
		return &DarkTheme
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

// ApplyTheme applies the current theme to a tview application
func (tm *ThemeManager) ApplyTheme(app *tview.Application) {
	// For now, themes are applied through color codes in text
	// tview doesn't have direct theme support, so we use color tags in text
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

// FormatColor formats text with theme color
func (tm *ThemeManager) FormatColor(text, colorType string) string {
	color := tm.GetColor(colorType)
	return fmt.Sprintf("[%s]%s[white]", color, text)
}
