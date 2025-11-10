package vision

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

// TestImageDetectionByMIME tests image detection using MIME types
func TestImageDetectionByMIME(t *testing.T) {
	config := DefaultDetectionConfig()
	detector := NewImageDetector(config)
	ctx := context.Background()

	tests := []struct {
		name       string
		input      *Input
		wantImages bool
		wantCount  int
	}{
		{
			name: "detect image by MIME type",
			input: &Input{
				Files: []*File{
					{
						Path:      "test.jpg",
						Name:      "test.jpg",
						Extension: "jpg",
						MIMEType:  "image/jpeg",
						Size:      1024,
					},
				},
			},
			wantImages: true,
			wantCount:  1,
		},
		{
			name: "multiple images",
			input: &Input{
				Files: []*File{
					{
						Path:     "image1.png",
						MIMEType: "image/png",
					},
					{
						Path:     "image2.jpg",
						MIMEType: "image/jpeg",
					},
				},
			},
			wantImages: true,
			wantCount:  2,
		},
		{
			name: "no images - text file",
			input: &Input{
				Files: []*File{
					{
						Path:     "document.txt",
						MIMEType: "text/plain",
					},
				},
			},
			wantImages: false,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := detector.Detect(ctx, tt.input)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			if result.HasImages != tt.wantImages {
				t.Errorf("HasImages = %v, want %v", result.HasImages, tt.wantImages)
			}

			if result.ImageCount != tt.wantCount {
				t.Errorf("ImageCount = %v, want %v", result.ImageCount, tt.wantCount)
			}
		})
	}
}

// TestImageDetectionByExtension tests image detection using file extensions
func TestImageDetectionByExtension(t *testing.T) {
	config := DefaultDetectionConfig()
	detector := NewImageDetector(config)

	tests := []struct {
		name      string
		extension string
		wantImage bool
	}{
		{"jpg extension", "jpg", true},
		{"jpeg extension", "jpeg", true},
		{"png extension", "png", true},
		{"gif extension", "gif", true},
		{"webp extension", "webp", true},
		{"bmp extension", "bmp", true},
		{"txt extension", "txt", false},
		{"pdf extension", "pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &File{
				Path:      "test." + tt.extension,
				Extension: tt.extension,
			}

			isImage, err := detector.DetectInFile(file)
			if err != nil {
				t.Fatalf("DetectInFile() error = %v", err)
			}

			if isImage != tt.wantImage {
				t.Errorf("DetectInFile() = %v, want %v", isImage, tt.wantImage)
			}
		})
	}
}

// TestImageDetectionBase64 tests base64 image detection
func TestImageDetectionBase64(t *testing.T) {
	config := DefaultDetectionConfig()
	detector := NewImageDetector(config)
	ctx := context.Background()

	tests := []struct {
		name      string
		text      string
		wantCount int
	}{
		{
			name:      "base64 PNG image",
			text:      "Here's an image: data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			wantCount: 1,
		},
		{
			name:      "base64 JPEG image",
			text:      "Check this: data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oACAEBAAA/AKp/2Q==",
			wantCount: 1,
		},
		{
			name:      "multiple base64 images",
			text:      "Image 1: data:image/png;base64,iVBORw0KGgoAAAA= and Image 2: data:image/jpeg;base64,/9j/4AAQSkZJRg==",
			wantCount: 2,
		},
		{
			name:      "no base64 images",
			text:      "Just plain text without images",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &Input{Text: tt.text}
			result, err := detector.Detect(ctx, input)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			if result.ImageCount != tt.wantCount {
				t.Errorf("ImageCount = %v, want %v", result.ImageCount, tt.wantCount)
			}
		})
	}
}

// TestImageDetectionURL tests URL-based image detection
func TestImageDetectionURL(t *testing.T) {
	config := DefaultDetectionConfig()
	detector := NewImageDetector(config)

	tests := []struct {
		name      string
		text      string
		wantCount int
	}{
		{
			name:      "single image URL",
			text:      "Check out this image: https://example.com/photo.jpg",
			wantCount: 1,
		},
		{
			name:      "multiple image URLs",
			text:      "https://example.com/img1.png and https://example.com/img2.jpeg",
			wantCount: 2,
		},
		{
			name:      "mixed extensions",
			text:      "https://site.com/a.gif https://site.com/b.webp",
			wantCount: 2,
		},
		{
			name:      "no image URLs",
			text:      "https://example.com/page.html",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, err := detector.DetectInText(tt.text)
			if err != nil {
				t.Fatalf("DetectInText() error = %v", err)
			}

			if len(images) != tt.wantCount {
				t.Errorf("got %d images, want %d", len(images), tt.wantCount)
			}
		})
	}
}

// TestContentInspection tests magic number-based content inspection
func TestContentInspection(t *testing.T) {
	inspector := NewContentInspector()

	tests := []struct {
		name       string
		content    []byte
		wantImage  bool
		wantFormat string
	}{
		{
			name:       "PNG file",
			content:    []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			wantImage:  true,
			wantFormat: "png",
		},
		{
			name:       "JPEG file",
			content:    []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
			wantImage:  true,
			wantFormat: "jpg",
		},
		{
			name:       "GIF file",
			content:    []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},
			wantImage:  true,
			wantFormat: "gif",
		},
		{
			name:       "BMP file",
			content:    []byte{0x42, 0x4D, 0x00, 0x00, 0x00, 0x00},
			wantImage:  true,
			wantFormat: "bmp",
		},
		{
			name:       "text file",
			content:    []byte("This is just text"),
			wantImage:  false,
			wantFormat: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.content)
			result, err := inspector.InspectContent(reader)
			if err != nil {
				t.Fatalf("InspectContent() error = %v", err)
			}

			if result.IsImage != tt.wantImage {
				t.Errorf("IsImage = %v, want %v", result.IsImage, tt.wantImage)
			}

			if result.Format != tt.wantFormat {
				t.Errorf("Format = %v, want %v", result.Format, tt.wantFormat)
			}
		})
	}
}

// TestModelCapabilities tests model capability checking
func TestModelCapabilities(t *testing.T) {
	registry := NewModelRegistry()
	checker := NewCapabilityChecker(registry)
	ctx := context.Background()

	tests := []struct {
		name       string
		modelID    string
		wantVision bool
		wantError  bool
	}{
		{
			name:       "Claude 3.5 Sonnet supports vision",
			modelID:    "claude-3-5-sonnet-20241022",
			wantVision: true,
			wantError:  false,
		},
		{
			name:       "GPT-4o supports vision",
			modelID:    "gpt-4o",
			wantVision: true,
			wantError:  false,
		},
		{
			name:       "Gemini 2.0 Flash supports vision",
			modelID:    "gemini-2.0-flash",
			wantVision: true,
			wantError:  false,
		},
		{
			name:       "unknown model",
			modelID:    "unknown-model",
			wantVision: false,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supports, err := checker.SupportsVision(ctx, tt.modelID)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantError && supports != tt.wantVision {
				t.Errorf("SupportsVision() = %v, want %v", supports, tt.wantVision)
			}
		})
	}
}

// TestAutoSwitch tests automatic model switching
func TestAutoSwitch(t *testing.T) {
	registry := NewModelRegistry()
	config := DefaultConfig()
	config.RequireConfirm = false // Auto-approve for test

	manager, err := NewVisionSwitchManager(config, registry)
	if err != nil {
		t.Fatalf("NewVisionSwitchManager() error = %v", err)
	}

	ctx := context.Background()

	// Set initial model to text-only
	textModel := &Model{
		ID:   "text-model",
		Name: "Text Only Model",
		Capabilities: &Capabilities{
			SupportsVision: false,
		},
	}
	manager.mu.Lock()
	manager.currentModel = textModel
	manager.mu.Unlock()

	// Test 1: Text input - no switch
	t.Run("no switch for text input", func(t *testing.T) {
		input := &Input{
			Text: "Hello, how are you?",
		}

		result, err := manager.ProcessInput(ctx, input)
		if err != nil {
			t.Fatalf("ProcessInput() error = %v", err)
		}

		if result.SwitchPerformed {
			t.Error("unexpected switch for text input")
		}
	})

	// Test 2: Image input - should switch
	t.Run("switch for image input", func(t *testing.T) {
		input := &Input{
			Files: []*File{
				{
					Path:     "screenshot.png",
					MIMEType: "image/png",
				},
			},
		}

		result, err := manager.ProcessInput(ctx, input)
		if err != nil {
			t.Fatalf("ProcessInput() error = %v", err)
		}

		if !result.SwitchPerformed {
			t.Error("expected switch for image input")
		}

		if result.ToModel == nil || !result.ToModel.Capabilities.SupportsVision {
			t.Error("switched to non-vision model")
		}

		if !manager.IsSwitchActive() {
			t.Error("expected switch to be active")
		}
	})
}

// TestSwitchModes tests different switch modes
func TestSwitchModes(t *testing.T) {
	tests := []struct {
		name       string
		mode       SwitchMode
		wantActive bool
	}{
		{
			name:       "once mode",
			mode:       SwitchOnce,
			wantActive: false, // Should revert automatically
		},
		{
			name:       "session mode",
			mode:       SwitchSession,
			wantActive: true, // Should remain active
		},
		{
			name:       "persist mode",
			mode:       SwitchPersist,
			wantActive: true, // Should remain active
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewModelRegistry()
			config := DefaultConfig()
			config.SwitchMode = tt.mode
			config.RequireConfirm = false

			manager, err := NewVisionSwitchManager(config, registry)
			if err != nil {
				t.Fatalf("NewVisionSwitchManager() error = %v", err)
			}

			ctx := context.Background()

			// Set text-only model
			manager.SetCurrentModel("text-model")

			// Process image input
			input := &Input{
				Files: []*File{
					{Path: "test.png", MIMEType: "image/png"},
				},
			}

			result, err := manager.ProcessInput(ctx, input)
			if err != nil {
				t.Fatalf("ProcessInput() error = %v", err)
			}

			if !result.SwitchPerformed {
				t.Error("expected switch to occur")
			}

			// For once mode, wait for revert
			if tt.mode == SwitchOnce {
				time.Sleep(200 * time.Millisecond)
			}

			// Check if switch is still active
			active := manager.IsSwitchActive()
			if active != tt.wantActive {
				t.Errorf("IsSwitchActive() = %v, want %v", active, tt.wantActive)
			}
		})
	}
}

// TestSwitchHistory tests switch event tracking
func TestSwitchHistory(t *testing.T) {
	config := DefaultSwitchConfig()
	controller := NewSwitchController(config)
	ctx := context.Background()

	fromModel := &Model{ID: "model-a", Name: "Model A"}
	toModel := &Model{ID: "model-b", Name: "Model B"}

	// Perform switch
	event, err := controller.Switch(ctx, fromModel, toModel, ReasonImageDetected)
	if err != nil {
		t.Fatalf("Switch() error = %v", err)
	}

	if event.ID == "" {
		t.Error("event ID should not be empty")
	}

	// Check history
	history := controller.GetHistory()
	if len(history) != 1 {
		t.Errorf("history length = %d, want 1", len(history))
	}

	// Check active switch
	activeSwitch := controller.GetActiveSwitch()
	if activeSwitch == nil {
		t.Error("expected active switch")
	}

	if activeSwitch.FromModel.ID != fromModel.ID {
		t.Errorf("FromModel.ID = %s, want %s", activeSwitch.FromModel.ID, fromModel.ID)
	}

	// Revert switch
	err = controller.Revert(ctx, event.ID)
	if err != nil {
		t.Fatalf("Revert() error = %v", err)
	}

	// Check that switch is reverted
	activeSwitch = controller.GetActiveSwitch()
	if activeSwitch != nil {
		t.Error("expected no active switch after revert")
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name:      "valid default config",
			config:    DefaultConfig(),
			wantError: false,
		},
		{
			name: "invalid switch mode",
			config: &Config{
				SwitchMode:       "invalid",
				DetectionMethods: []DetectionMethod{DetectByMIME},
				FallbackModel:    "claude-3-5-sonnet-20241022",
			},
			wantError: true,
		},
		{
			name: "no detection methods",
			config: &Config{
				SwitchMode:       SwitchSession,
				DetectionMethods: []DetectionMethod{},
				FallbackModel:    "claude-3-5-sonnet-20241022",
			},
			wantError: true,
		},
		{
			name: "no fallback model",
			config: &Config{
				SwitchMode:           SwitchSession,
				DetectionMethods:     []DetectionMethod{DetectByMIME},
				FallbackModel:        "",
				PreferredVisionModel: "",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestFindBestVisionModel tests finding the best vision model
func TestFindBestVisionModel(t *testing.T) {
	registry := NewModelRegistry()
	checker := NewCapabilityChecker(registry)
	ctx := context.Background()

	tests := []struct {
		name        string
		preferences *ModelPreferences
		wantModel   string
	}{
		{
			name: "prefer Claude",
			preferences: &ModelPreferences{
				PreferredModels: []string{"claude-3-5-sonnet-20241022"},
			},
			wantModel: "claude-3-5-sonnet-20241022",
		},
		{
			name: "prefer GPT-4o",
			preferences: &ModelPreferences{
				PreferredModels: []string{"gpt-4o"},
			},
			wantModel: "gpt-4o",
		},
		{
			name:        "no preferences - use default",
			preferences: &ModelPreferences{},
			wantModel:   "", // Will accept any vision model
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := checker.FindBestVisionModel(ctx, tt.preferences)
			if err != nil {
				t.Fatalf("FindBestVisionModel() error = %v", err)
			}

			if model == nil {
				t.Fatal("expected a model, got nil")
			}

			if !model.Capabilities.SupportsVision {
				t.Error("returned model does not support vision")
			}

			if tt.wantModel != "" && model.ID != tt.wantModel {
				t.Errorf("model.ID = %s, want %s", model.ID, tt.wantModel)
			}
		})
	}
}

// TestSwitchModeValidation tests switch mode validation
func TestSwitchModeValidation(t *testing.T) {
	tests := []struct {
		name  string
		mode  SwitchMode
		valid bool
	}{
		{"once mode", SwitchOnce, true},
		{"session mode", SwitchSession, true},
		{"persist mode", SwitchPersist, true},
		{"invalid mode", SwitchMode("invalid"), false},
		{"empty mode", SwitchMode(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.mode.IsValid()
			if valid != tt.valid {
				t.Errorf("IsValid() = %v, want %v", valid, tt.valid)
			}
		})
	}
}

// BenchmarkImageDetection benchmarks image detection performance
func BenchmarkImageDetection(b *testing.B) {
	config := DefaultDetectionConfig()
	detector := NewImageDetector(config)
	ctx := context.Background()

	input := &Input{
		Text: strings.Repeat("Some text ", 100) + " https://example.com/image.jpg",
		Files: []*File{
			{Path: "test.png", MIMEType: "image/png"},
			{Path: "test.jpg", Extension: "jpg"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.Detect(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
