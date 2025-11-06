package browser

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ChromeType defines the type of Chrome installation
type ChromeType int

const (
	ChromeTypeChrome ChromeType = iota
	ChromeTypeChromium
	ChromeTypeEdge
	ChromeTypeBrave
)

// String returns the string representation of ChromeType
func (ct ChromeType) String() string {
	switch ct {
	case ChromeTypeChrome:
		return "Chrome"
	case ChromeTypeChromium:
		return "Chromium"
	case ChromeTypeEdge:
		return "Edge"
	case ChromeTypeBrave:
		return "Brave"
	default:
		return "Unknown"
	}
}

// ChromeInfo contains Chrome installation info
type ChromeInfo struct {
	Path    string
	Version string
	Type    ChromeType
}

// ChromeDiscovery discovers Chrome installations
type ChromeDiscovery interface {
	// FindChrome finds Chrome/Chromium executable
	FindChrome() (string, error)

	// FindChromeVersion returns the Chrome version
	FindChromeVersion(path string) (string, error)

	// GetDefaultPaths returns default Chrome paths for the platform
	GetDefaultPaths() []string

	// FindAll finds all Chrome installations
	FindAll() ([]ChromeInfo, error)
}

// DefaultChromeDiscovery implements ChromeDiscovery
type DefaultChromeDiscovery struct{}

// NewDefaultChromeDiscovery creates a new Chrome discovery
func NewDefaultChromeDiscovery() *DefaultChromeDiscovery {
	return &DefaultChromeDiscovery{}
}

// FindChrome finds Chrome/Chromium executable
func (d *DefaultChromeDiscovery) FindChrome() (string, error) {
	paths := d.GetDefaultPaths()

	// Check default paths first
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try PATH environment variable
	chromeBinaries := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"chrome",
	}

	for _, binary := range chromeBinaries {
		path, err := exec.LookPath(binary)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Chrome not found on this system")
}

// GetDefaultPaths returns default Chrome paths for the platform
func (d *DefaultChromeDiscovery) GetDefaultPaths() []string {
	switch runtime.GOOS {
	case "darwin": // macOS
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			os.ExpandEnv("$HOME/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
			os.ExpandEnv("$HOME/Applications/Chromium.app/Contents/MacOS/Chromium"),
		}
	case "linux":
		return []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
			"/usr/bin/brave-browser",
			"/usr/bin/microsoft-edge",
			"/opt/google/chrome/google-chrome",
			"/opt/chromium/chromium",
			os.ExpandEnv("$HOME/.local/bin/chrome"),
			os.ExpandEnv("$HOME/.local/bin/chromium"),
		}
	case "windows":
		return []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Chromium\Application\chrome.exe`,
			`C:\Program Files (x86)\Chromium\Application\chrome.exe`,
			`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
			`C:\Program Files (x86)\BraveSoftware\Brave-Browser\Application\brave.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			os.ExpandEnv(`%LOCALAPPDATA%\Google\Chrome\Application\chrome.exe`),
			os.ExpandEnv(`%PROGRAMFILES%\Google\Chrome\Application\chrome.exe`),
			os.ExpandEnv(`%PROGRAMFILES(X86)%\Google\Chrome\Application\chrome.exe`),
		}
	default:
		return []string{}
	}
}

// FindChromeVersion returns the Chrome version
func (d *DefaultChromeDiscovery) FindChromeVersion(path string) (string, error) {
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative version flag
		cmd = exec.Command(path, "-version")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get Chrome version: %w", err)
		}
	}

	// Parse version from output
	// Example outputs:
	// "Google Chrome 120.0.6099.109"
	// "Chromium 120.0.6099.71"
	// "Brave Browser 1.60.118"
	version := strings.TrimSpace(string(output))
	parts := strings.Fields(version)
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected version format: %s", version)
	}

	// Return the last part which should be the version number
	return parts[len(parts)-1], nil
}

// FindAll finds all Chrome installations on the system
func (d *DefaultChromeDiscovery) FindAll() ([]ChromeInfo, error) {
	var chromes []ChromeInfo
	paths := d.GetDefaultPaths()

	// Check all default paths
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			version, _ := d.FindChromeVersion(path)
			chromeType := d.detectChromeType(path)

			chromes = append(chromes, ChromeInfo{
				Path:    path,
				Version: version,
				Type:    chromeType,
			})
		}
	}

	// Also check PATH
	chromeBinaries := map[string]ChromeType{
		"google-chrome":        ChromeTypeChrome,
		"google-chrome-stable": ChromeTypeChrome,
		"chromium":             ChromeTypeChromium,
		"chromium-browser":     ChromeTypeChromium,
		"chrome":               ChromeTypeChrome,
		"brave-browser":        ChromeTypeBrave,
		"microsoft-edge":       ChromeTypeEdge,
		"msedge":               ChromeTypeEdge,
	}

	for binary, chromeType := range chromeBinaries {
		path, err := exec.LookPath(binary)
		if err == nil {
			// Check if we already have this path
			exists := false
			for _, chrome := range chromes {
				if chrome.Path == path {
					exists = true
					break
				}
			}

			if !exists {
				version, _ := d.FindChromeVersion(path)
				chromes = append(chromes, ChromeInfo{
					Path:    path,
					Version: version,
					Type:    chromeType,
				})
			}
		}
	}

	if len(chromes) == 0 {
		return nil, fmt.Errorf("no Chrome installations found")
	}

	return chromes, nil
}

// detectChromeType detects the type of Chrome from the path
func (d *DefaultChromeDiscovery) detectChromeType(path string) ChromeType {
	lowerPath := strings.ToLower(path)

	if strings.Contains(lowerPath, "brave") {
		return ChromeTypeBrave
	}
	if strings.Contains(lowerPath, "edge") || strings.Contains(lowerPath, "msedge") {
		return ChromeTypeEdge
	}
	if strings.Contains(lowerPath, "chromium") {
		return ChromeTypeChromium
	}
	// Default to Chrome (includes "chrome" in path)
	return ChromeTypeChrome
}

// GetPreferredChrome returns the preferred Chrome installation
// Priority: Chrome > Edge > Brave > Chromium
func GetPreferredChrome() (string, error) {
	discovery := NewDefaultChromeDiscovery()
	chromes, err := discovery.FindAll()
	if err != nil {
		return discovery.FindChrome()
	}

	// Priority order
	priorities := []ChromeType{
		ChromeTypeChrome,
		ChromeTypeEdge,
		ChromeTypeBrave,
		ChromeTypeChromium,
	}

	for _, priority := range priorities {
		for _, chrome := range chromes {
			if chrome.Type == priority {
				return chrome.Path, nil
			}
		}
	}

	// Fallback to first found
	if len(chromes) > 0 {
		return chromes[0].Path, nil
	}

	return "", fmt.Errorf("no Chrome installation found")
}
