package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"dev.helix.code/internal/logo"
)

func main() {
	// Get the project root directory
	projectRoot, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current directory:", err)
	}

	// Define paths
	sourceLogo := filepath.Join(projectRoot, "..", "..", "assets", "images", "logo.png")
	outputDir := filepath.Join(projectRoot, "..", "..", "assets")

	// Check if source logo exists
	if _, err := os.Stat(sourceLogo); os.IsNotExist(err) {
		log.Printf("Source logo not found at %s, using default colors", sourceLogo)
		// Create output directories anyway
		createOutputDirs(outputDir)
		generateDefaultAssets(outputDir)
		return
	}

	// Create output directories
	createOutputDirs(outputDir)

	// Initialize logo processor
	processor := logo.NewLogoProcessor(sourceLogo, outputDir)

	// Extract colors from logo
	fmt.Println("🔍 Extracting colors from logo...")
	if err := processor.ExtractColors(); err != nil {
		log.Printf("Warning: Could not extract colors from logo: %v", err)
		log.Println("Using default color scheme")
	}

	// Generate ASCII art
	fmt.Println("🎨 Generating ASCII art...")
	asciiArt, err := processor.GenerateASCIIArt()
	if err != nil {
		log.Printf("Warning: Could not generate ASCII art: %v", err)
	} else {
		// Save ASCII art
		asciiPath := filepath.Join(outputDir, "images", "logo-ascii.txt")
		if err := os.WriteFile(asciiPath, []byte(asciiArt), 0644); err != nil {
			log.Printf("Warning: Could not save ASCII art: %v", err)
		} else {
			fmt.Println("✅ ASCII art saved to:", asciiPath)
		}

		// Print ASCII art to console
		fmt.Println("\n" + asciiArt)
	}

	// Generate icons
	fmt.Println("📱 Generating platform icons...")
	if err := processor.GenerateIcons(); err != nil {
		log.Printf("Warning: Could not generate icons: %v", err)
	} else {
		fmt.Println("✅ Icons generated in:", filepath.Join(outputDir, "icons"))
	}

	// Save color scheme
	fmt.Println("🎨 Saving color scheme...")
	if err := processor.SaveColorScheme(); err != nil {
		log.Printf("Warning: Could not save color scheme: %v", err)
	} else {
		fmt.Println("✅ Color scheme saved to:", filepath.Join(outputDir, "colors", "color-scheme.json"))
	}

	// Generate theme files
	fmt.Println("🎨 Generating theme files...")
	if err := processor.GenerateThemeFiles(); err != nil {
		log.Printf("Warning: Could not generate theme files: %v", err)
	} else {
		fmt.Println("✅ Theme files generated")
	}

	fmt.Println("\n✨ Logo asset generation complete!")
	fmt.Printf("\n🎨 Color Scheme:\n")
	fmt.Printf("   Primary:   %s\n", processor.Colors.Primary)
	fmt.Printf("   Secondary: %s\n", processor.Colors.Secondary)
	fmt.Printf("   Accent:    %s\n", processor.Colors.Accent)
	fmt.Printf("   Text:      %s\n", processor.Colors.Text)
	fmt.Printf("   Background: %s\n", processor.Colors.Background)
}

func createOutputDirs(outputDir string) {
	dirs := []string{
		filepath.Join(outputDir, "icons"),
		filepath.Join(outputDir, "images"),
		filepath.Join(outputDir, "colors"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Warning: Could not create directory %s: %v", dir, err)
		}
	}
}

func generateDefaultAssets(outputDir string) {
	// Create default color scheme
	defaultColors := `{
  "primary": "#2E86AB",
  "secondary": "#A23B72", 
  "accent": "#F18F01",
  "text": "#2D3047",
  "background": "#F5F5F5"
}`

	colorPath := filepath.Join(outputDir, "colors", "color-scheme.json")
	if err := os.WriteFile(colorPath, []byte(defaultColors), 0644); err != nil {
		log.Printf("Warning: Could not save default color scheme: %v", err)
	}

	// Create default ASCII art
	defaultASCII := `
    ╔══════════════════════════════════╗
    ║        HELIXCODE CLI             ║
    ║    Distributed AI Development    ║
    ║                                  ║
    ║        🌀  Helix  🌀            ║
    ║                                  ║
    ║   Intelligent Task Division      ║
    ║   Work Preservation System       ║
    ║   Cross-Platform Architecture    ║
    ╚══════════════════════════════════╝
`

	asciiPath := filepath.Join(outputDir, "images", "logo-ascii.txt")
	if err := os.WriteFile(asciiPath, []byte(defaultASCII), 0644); err != nil {
		log.Printf("Warning: Could not save default ASCII art: %v", err)
	}

	// Create default theme files
	defaultCSS := `/* HelixCode Default Theme */
:root {
  --primary-color: #2E86AB;
  --secondary-color: #A23B72;
  --accent-color: #F18F01;
  --text-color: #2D3047;
  --background-color: #F5F5F5;
  --border-color: #2E86AB80;
  --hover-color: #2E86AB1a;
}`

	cssPath := filepath.Join(outputDir, "colors", "helix-theme.css")
	if err := os.WriteFile(cssPath, []byte(defaultCSS), 0644); err != nil {
		log.Printf("Warning: Could not save default CSS theme: %v", err)
	}

	// Create default Go theme
	defaultGoTheme := `package theme

// Color constants for HelixCode
const (
	PrimaryColor = "#2E86AB"
	SecondaryColor = "#A23B72"
	AccentColor = "#F18F01"
	TextColor = "#2D3047"
	BackgroundColor = "#F5F5F5"
)`

	goPath := filepath.Join(outputDir, "..", "..", "internal", "theme", "theme.go")
	if err := os.WriteFile(goPath, []byte(defaultGoTheme), 0644); err != nil {
		log.Printf("Warning: Could not save default Go theme: %v", err)
	}

	fmt.Println("✅ Default assets generated (logo not found)")
}
