package logo

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
)

// ColorScheme represents the extracted color palette from the logo
type ColorScheme struct {
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Accent     string `json:"accent"`
	Text       string `json:"text"`
	Background string `json:"background"`
}

// LogoProcessor handles logo processing and asset generation
type LogoProcessor struct {
	SourcePath string
	OutputDir  string
	Colors     ColorScheme
}

// NewLogoProcessor creates a new logo processor
func NewLogoProcessor(sourcePath, outputDir string) *LogoProcessor {
	return &LogoProcessor{
		SourcePath: sourcePath,
		OutputDir:  outputDir,
		Colors: ColorScheme{
			Primary:    "#2E86AB", // Deep blue
			Secondary:  "#A23B72", // Purple accent
			Accent:     "#F18F01", // Orange accent
			Text:       "#2D3047", // Dark text
			Background: "#F5F5F5", // Light background
		},
	}
}

// ExtractColors extracts the dominant colors from the logo
func (lp *LogoProcessor) ExtractColors() error {
	file, err := os.Open(lp.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to open logo file: %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode logo image: %v", err)
	}

	// Analyze image for dominant colors
	bounds := img.Bounds()
	colorCounts := make(map[color.Color]int)

	// Sample pixels to find dominant colors
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 10 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 10 {
			c := img.At(x, y)
			colorCounts[c]++
		}
	}

	// Find the most common colors
	var dominantColors []color.Color
	for c, count := range colorCounts {
		if count > 50 { // Threshold for dominant colors
			dominantColors = append(dominantColors, c)
		}
	}

	// Update color scheme based on extracted colors
	if len(dominantColors) > 0 {
		lp.updateColorScheme(dominantColors)
	}

	return nil
}

// updateColorScheme updates the color scheme based on extracted colors
func (lp *LogoProcessor) updateColorScheme(colors []color.Color) {
	// Convert colors to hex and update scheme
	// This is a simplified implementation - in practice you'd want more sophisticated color analysis
	if len(colors) >= 2 {
		lp.Colors.Primary = colorToHex(colors[0])
		lp.Colors.Secondary = colorToHex(colors[1])
	}
	if len(colors) >= 3 {
		lp.Colors.Accent = colorToHex(colors[2])
	}
}

// GenerateASCIIArt creates ASCII art from the logo
func (lp *LogoProcessor) GenerateASCIIArt() (string, error) {
	file, err := os.Open(lp.SourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to open logo file: %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode logo image: %v", err)
	}

	// Resize to a reasonable size for ASCII art
	resized := resize.Resize(40, 0, img, resize.Lanczos3)
	bounds := resized.Bounds()

	var ascii strings.Builder
	ascii.WriteString("\n")

	// Convert image to ASCII
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := resized.At(x, y)
			gray := color.GrayModel.Convert(c).(color.Gray)
			asciiChar := grayToASCII(gray.Y)
			ascii.WriteString(asciiChar)
		}
		ascii.WriteString("\n")
	}

	return ascii.String(), nil
}

// GenerateIcons generates various icon sizes for different platforms
func (lp *LogoProcessor) GenerateIcons() error {
	sizes := []struct {
		width  int
		height int
		name   string
	}{
		{16, 16, "icon-16x16.png"},
		{32, 32, "icon-32x32.png"},
		{48, 48, "icon-48x48.png"},
		{64, 64, "icon-64x64.png"},
		{128, 128, "icon-128x128.png"},
		{256, 256, "icon-256x256.png"},
		{512, 512, "icon-512x512.png"},
	}

	file, err := os.Open(lp.SourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	for _, size := range sizes {
		resized := resize.Resize(uint(size.width), uint(size.height), img, resize.Lanczos3)

		outputPath := filepath.Join(lp.OutputDir, "icons", size.name)
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		err = png.Encode(outputFile, resized)
		if err != nil {
			return err
		}
	}

	return nil
}

// SaveColorScheme saves the extracted color scheme to a file
func (lp *LogoProcessor) SaveColorScheme() error {
	colorFile := filepath.Join(lp.OutputDir, "colors", "color-scheme.json")
	file, err := os.Create(colorFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	colorJSON := fmt.Sprintf(`{
  "primary": "%s",
  "secondary": "%s", 
  "accent": "%s",
  "text": "%s",
  "background": "%s"
}`, lp.Colors.Primary, lp.Colors.Secondary, lp.Colors.Accent, lp.Colors.Text, lp.Colors.Background)

	_, err = writer.WriteString(colorJSON)
	if err != nil {
		return err
	}

	return writer.Flush()
}

// GenerateThemeFiles generates theme files for different platforms
func (lp *LogoProcessor) GenerateThemeFiles() error {
	// Generate CSS theme
	cssTheme := fmt.Sprintf(`/* HelixCode Theme - Generated from Logo */
:root {
  --primary-color: %s;
  --secondary-color: %s;
  --accent-color: %s;
  --text-color: %s;
  --background-color: %s;
  --border-color: %s80;
  --hover-color: %s1a;
}

.helix-primary { color: %s; }
.helix-secondary { color: %s; }
.helix-accent { color: %s; }
.helix-bg { background-color: %s; }`,
		lp.Colors.Primary, lp.Colors.Secondary, lp.Colors.Accent, lp.Colors.Text,
		lp.Colors.Background, lp.Colors.Primary, lp.Colors.Primary,
		lp.Colors.Primary, lp.Colors.Secondary, lp.Colors.Accent, lp.Colors.Background)

	cssPath := filepath.Join(lp.OutputDir, "colors", "helix-theme.css")
	err := os.WriteFile(cssPath, []byte(cssTheme), 0644)
	if err != nil {
		return err
	}

	// Generate Go theme constants
	goTheme := fmt.Sprintf(`package theme

// Color constants extracted from HelixCode logo
const (
	PrimaryColor = "%s"
	SecondaryColor = "%s"
	AccentColor = "%s"
	TextColor = "%s"
	BackgroundColor = "%s"
)`, lp.Colors.Primary, lp.Colors.Secondary, lp.Colors.Accent, lp.Colors.Text, lp.Colors.Background)

	goPath := filepath.Join(lp.OutputDir, "..", "..", "internal", "theme", "theme.go")
	// Create directory if it doesn't exist
	goDir := filepath.Dir(goPath)
	err = os.MkdirAll(goDir, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(goPath, []byte(goTheme), 0644)
	if err != nil {
		return err
	}

	return nil
}

// Helper functions
func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

func grayToASCII(gray uint8) string {
	asciiChars := " .:-=+*#%@"
	index := int(gray) * (len(asciiChars) - 1) / 255
	return string(asciiChars[index])
}
