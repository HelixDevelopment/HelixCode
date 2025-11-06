// Package vision provides automatic vision model switching for HelixCode.
//
// The vision package enables seamless detection of image content in user input
// and automatic switching to vision-capable models when needed. This allows
// users to work with images without manually changing models.
//
// # Key Features
//
//   - Automatic image detection in multiple formats (base64, file paths, URLs)
//   - Smart model switching with user confirmation
//   - Multiple switch modes: once (one-time), session (current session), persist (always)
//   - Model capability registry with vision support tracking
//   - Fallback to text-only models when vision unavailable
//
// # Architecture
//
// The package consists of several core components:
//
//   - ImageDetector: Detects images in user input using various methods
//   - ModelRegistry: Maintains registry of available models and capabilities
//   - CapabilityChecker: Checks if models support vision features
//   - VisionSwitchManager: Coordinates detection and switching
//   - SwitchController: Manages switch state and history
//
// # Image Detection
//
// Images can be detected using multiple methods:
//
//   - MIME type checking (image/jpeg, image/png, etc.)
//   - File extension analysis (*.jpg, *.png, *.gif, *.webp)
//   - Base64 data URI detection (data:image/png;base64,...)
//   - URL pattern matching (http://example.com/image.jpg)
//   - Content inspection using magic numbers
//
// # Switch Modes
//
// Three switch modes control persistence:
//
// Once Mode:
//   - Switches for a single request only
//   - Automatically reverts to original model after
//   - No persistent state
//
// Session Mode:
//   - Switches for the current session
//   - All requests in session use vision model
//   - Reverts when session ends
//
// Persist Mode:
//   - Permanently switches to vision model
//   - Saves preference to configuration
//   - All future sessions use vision model
//
// # Vision-Capable Models
//
// The following models support vision by default:
//
//   - Anthropic: All Claude models (claude-3-5-sonnet, claude-3-opus, etc.)
//   - OpenAI: gpt-4o, gpt-4-turbo, gpt-4-vision-preview
//   - Google: gemini-2.0-flash, gemini-pro-vision
//   - Qwen: qwen-vl-plus, qwen-vl-max
//   - Copilot: claude-3.5-sonnet, gpt-4o, gemini-2.0-flash
//
// # Usage Example
//
// Basic usage with automatic detection and switching:
//
//	// Create registry with default vision models
//	registry := vision.NewModelRegistry()
//
//	// Configure auto-switch behavior
//	config := vision.DefaultConfig()
//	config.SwitchMode = vision.SwitchSession
//	config.RequireConfirm = true
//
//	// Create switch manager
//	manager, err := vision.NewVisionSwitchManager(config, registry)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Set current model
//	manager.SetCurrentModel("gpt-3.5-turbo")
//
//	// Process user input with image
//	input := &vision.Input{
//	    Files: []*vision.File{
//	        {
//	            Path:     "screenshot.png",
//	            MIMEType: "image/png",
//	        },
//	    },
//	}
//
//	result, err := manager.ProcessInput(context.Background(), input)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if result.SwitchPerformed {
//	    fmt.Printf("Switched from %s to %s\n",
//	        result.FromModel.Name, result.ToModel.Name)
//	}
//
// # Custom Model Registration
//
// Register custom models with vision capabilities:
//
//	registry := vision.NewModelRegistry()
//
//	customModel := &vision.Model{
//	    ID:       "custom-vision-model",
//	    Name:     "Custom Vision Model",
//	    Provider: "custom",
//	    Capabilities: &vision.Capabilities{
//	        SupportsVision:   true,
//	        MaxImageSize:     10 * 1024 * 1024,
//	        MaxImages:        5,
//	        SupportedFormats: []string{"jpg", "png"},
//	        ContextWindow:    32000,
//	    },
//	}
//
//	err := registry.Register(customModel)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Configuration
//
// Configure detection and switching behavior:
//
//	config := &vision.Config{
//	    EnableAutoDetect: true,
//	    DetectionMethods: []vision.DetectionMethod{
//	        vision.DetectByMIME,
//	        vision.DetectByExtension,
//	        vision.DetectByBase64,
//	    },
//	    SwitchMode:           vision.SwitchSession,
//	    RequireConfirm:       true,
//	    FallbackModel:        "claude-3-5-sonnet-20241022",
//	    PreferredVisionModel: "gpt-4o",
//	    ModelPriority:        []string{"gpt-4o", "claude-3-5-sonnet"},
//	    AutoRevert:           false,
//	    KeepForSession:       true,
//	}
//
// # Error Handling
//
// The package defines several error types:
//
//   - ErrNoImagesDetected: No images found in input
//   - ErrModelNotFound: Requested model not in registry
//   - ErrNoVisionSupport: Model does not support vision
//   - ErrNoVisionModels: No vision-capable models available
//   - ErrSwitchFailed: Model switch operation failed
//   - ErrSwitchDenied: User denied switch confirmation
//
// # Thread Safety
//
// All components are thread-safe and can be used concurrently:
//
//   - ModelRegistry uses sync.RWMutex for concurrent access
//   - VisionSwitchManager protects state with mutexes
//   - SwitchController safely tracks switch history
//
// # Performance
//
// Detection is optimized for performance:
//
//   - MIME and extension checks are fast (< 1ms)
//   - Base64 detection uses compiled regex
//   - Content inspection only reads first 16 bytes
//   - Results are returned immediately without buffering
//
// # Integration
//
// The package integrates with HelixCode's LLM system:
//
//   - Uses existing llm.Provider interface
//   - Compatible with all provider types
//   - Supports tool calling and streaming
//   - Respects rate limits and quotas
//
// For more information, see the technical design document at:
// Design/TechnicalDesigns/Advanced/VisionAutoSwitch.md
package vision
