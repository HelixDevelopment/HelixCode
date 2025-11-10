// internal/ascii/logo_generator.go
package ascii

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
)

// LogoASCIIGenerator converts the Logo.png to colored ASCII art
type LogoASCIIGenerator struct {
	logoPath string
	width    int
	height   int
}

func NewLogoASCIIGenerator(logoPath string, width, height int) *LogoASCIIGenerator {
	return &LogoASCIIGenerator{
		logoPath: logoPath,
		width:    width,
		height:   height,
	}
}

// Color schemes for ASCII art
var (
	// Greenish color palette matching the logo
	GreenPalette = []string{
		"\033[38;5;22m",  // Dark green
		"\033[38;5;28m",  // Medium green
		"\033[38;5;34m",  // Green
		"\033[38;5;40m",  // Bright green
		"\033[38;5;46m",  // Very bright green
		"\033[38;5;118m", // Light green
	}
	
	// ASCII characters from darkest to lightest
	ASCIIChars = []string{" ", ".", ":", "-", "=", "+", "*", "#", "%", "@"}
)

// GenerateFromLogo converts the PNG logo to colored ASCII art
func (g *LogoASCIIGenerator) GenerateFromLogo() (string, error) {
	// Open and decode the logo file
	file, err := os.Open(g.logoPath)
	if err != nil {
		return g.GenerateFallbackLogo(), fmt.Errorf("failed to open logo: %w", err)
	}
	defer file.Close()
	
	img, err := png.Decode(file)
	if err != nil {
		return g.GenerateFallbackLogo(), fmt.Errorf("failed to decode PNG: %w", err)
	}
	
	// Resize image to desired ASCII dimensions
	resized := g.resizeImage(img, g.width, g.height)
	
	// Convert to ASCII with colors
	asciiArt := g.convertToASCII(resized)
	
	return asciiArt, nil
}

// resizeImage resizes the image for ASCII conversion
func (g *LogoASCIIGenerator) resizeImage(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	
	// Calculate aspect ratio preserving dimensions
	aspectRatio := float64(imgWidth) / float64(imgHeight)
	newWidth := width
	newHeight := int(float64(width) / aspectRatio)
	
	if newHeight > height {
		newHeight = height
		newWidth = int(float64(height) * aspectRatio)
	}
	
	// Create new image with calculated dimensions
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	
	// Simple nearest-neighbor resize
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := x * imgWidth / newWidth
			srcY := y * imgHeight / newHeight
			resized.Set(x, y, img.At(srcX, srcY))
		}
	}
	
	return resized
}

// convertToASCII converts the image to colored ASCII art
func (g *LogoASCIIGenerator) convertToASCII(img image.Image) string {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	var ascii strings.Builder
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get pixel color
			c := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			
			// Convert to brightness (0-255)
			brightness := float64(c.Y)
			
			// Map brightness to ASCII character
			charIndex := int((brightness / 255.0) * float64(len(ASCIIChars)-1))
			if charIndex < 0 {
				charIndex = 0
			}
			if charIndex >= len(ASCIIChars) {
				charIndex = len(ASCIIChars) - 1
			}
			
			// Map brightness to green color
			colorIndex := int((brightness / 255.0) * float64(len(GreenPalette)-1))
			if colorIndex < 0 {
				colorIndex = 0
			}
			if colorIndex >= len(GreenPalette) {
				colorIndex = len(GreenPalette) - 1
			}
			
			// Write colored character
			ascii.WriteString(GreenPalette[colorIndex])
			ascii.WriteString(ASCIIChars[charIndex])
		}
		ascii.WriteString("\033[0m\n") // Reset color and newline
	}
	
	return ascii.String()
}

// GenerateFallbackLogo creates a fallback ASCII logo if PNG conversion fails
func (g *LogoASCIIGenerator) GenerateFallbackLogo() string {
	// This is a stylized HelixCode logo in ASCII with green colors
	logo := `
` + GreenPalette[4] + `
    ╔══════════════════════════════════════════════════════════════════════════════╗
    ║` + GreenPalette[2] + `                                                                              ` + GreenPalette[4] + `║
    ║` + GreenPalette[2] + `          ██╗  ██╗███████╗██╗     ██╗██╗  ██╗██████╗  ██████╗ ██████╗ ███████╗ ` + GreenPalette[4] + `║
    ║` + GreenPalette[3] + `          ██║  ██║██╔════╝██║     ██║╚██╗██╔╝██╔══██╗██╔════╝██╔═══██╗██╔════╝ ` + GreenPalette[4] + `║
    ║` + GreenPalette[3] + `          ███████║█████╗  ██║     ██║ ╚███╔╝ ██████╔╝██║     ██║   ██║█████╗   ` + GreenPalette[4] + `║
    ║` + GreenPalette[4] + `          ██╔══██║██╔══╝  ██║     ██║ ██╔██╗ ██╔══██╗██║     ██║   ██║██╔══╝   ` + GreenPalette[4] + `║
    ║` + GreenPalette[4] + `          ██║  ██║███████╗███████╗██║██╔╝ ██╗██████╔╝╚██████╗╚██████╔╝███████╗ ` + GreenPalette[4] + `║
    ║` + GreenPalette[5] + `          ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝╚═╝  ╚═╝╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝ ` + GreenPalette[4] + `║
    ║` + GreenPalette[2] + `                                                                              ` + GreenPalette[4] + `║
    ║` + GreenPalette[3] + `                        Distributed AI Development Platform                    ` + GreenPalette[4] + `║
    ║` + GreenPalette[5] + `                                dev.helix.code v1.0.0                          ` + GreenPalette[4] + `║
    ║` + GreenPalette[2] + `                                                                              ` + GreenPalette[4] + `║
    ╚══════════════════════════════════════════════════════════════════════════════╝
` + "\033[0m"

	return logo
}

// GenerateSimpleLogo creates a simpler version for smaller terminals
func (g *LogoASCIIGenerator) GenerateSimpleLogo() string {
	logo := GreenPalette[4] + `
╔══════════════════════════════════════════════════════════════════════════════╗
║` + GreenPalette[3] + `                        HELIXCODE v1.0.0                         ` + GreenPalette[4] + `║
║` + GreenPalette[5] + `                 Distributed AI Development Platform              ` + GreenPalette[4] + `║
╚══════════════════════════════════════════════════════════════════════════════╝
` + "\033[0m"

	return logo
}

// GenerateMinimalLogo creates a minimal logo for very small terminals
func (g *LogoASCIIGenerator) GenerateMinimalLogo() string {
	logo := GreenPalette[4] + `
┌──────────────────────────────────────────────────────────────────────────────┐
│` + GreenPalette[3] + `                          HelixCode                           ` + GreenPalette[4] + `│
│` + GreenPalette[5] + `                      AI Development Platform                  ` + GreenPalette[4] + `│
└──────────────────────────────────────────────────────────────────────────────┘
` + "\033[0m"

	return logo
}

// GenerateLogoWithStatus creates a logo with system status information
func (g *LogoASCIIGenerator) GenerateLogoWithStatus(workers, models, sessions int) string {
	logo := g.GenerateSimpleLogo()
	
	status := fmt.Sprintf(`
`+GreenPalette[2]+`Status:`+GreenPalette[5]+` Workers: %d | Models: %d | Sessions: %d
`+GreenPalette[2]+`Ready for distributed AI development. Type 'help' for available commands.`+"\033[0m",
		workers, models, sessions)
	
	return logo + status
}

// GenerateAnimatedLogo creates an animated version with pulsing effect
func (g *LogoASCIIGenerator) GenerateAnimatedLogo(step int) string {
	// Cycle through different green shades for animation
	animationColors := []string{
		GreenPalette[2],
		GreenPalette[3], 
		GreenPalette[4],
		GreenPalette[5],
		GreenPalette[4],
		GreenPalette[3],
	}
	
	colorIndex := step % len(animationColors)
	
	logo := animationColors[colorIndex] + `
    ╦ ╦┌─┐┬  ┌─┐┌─┐┌┬┐┌─┐┌─┐
    ║║║├┤ │  │  │ ││││├┤ └─┐
    ╚╩╝└─┘┴─┘└─┘└─┘┴ ┴└─┘└─┘
` + "\033[0m" + GreenPalette[5] + `
    Distributed AI Development Platform
` + "\033[0m"

	return logo
}

// GenerateLogoForTerminalSize automatically selects appropriate logo size
func (g *LogoASCIIGenerator) GenerateLogoForTerminalSize(terminalWidth, terminalHeight int) string {
	if terminalWidth < 40 {
		return g.GenerateMinimalLogo()
	} else if terminalWidth < 80 {
		return g.GenerateSimpleLogo()
	} else {
		// Try to generate from actual PNG, fallback to ASCII
		asciiLogo, err := g.GenerateFromLogo()
		if err != nil {
			return g.GenerateFallbackLogo()
		}
		return asciiLogo
	}
}

// IsLogoFileExists checks if the logo file exists
func (g *LogoASCIIGenerator) IsLogoFileExists() bool {
	_, err := os.Stat(g.logoPath)
	return err == nil
}

// GetLogoDimensions gets the dimensions of the logo file
func (g *LogoASCIIGenerator) GetLogoDimensions() (int, int, error) {
	file, err := os.Open(g.logoPath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	
	img, err := png.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	
	return img.Width, img.Height, nil
}