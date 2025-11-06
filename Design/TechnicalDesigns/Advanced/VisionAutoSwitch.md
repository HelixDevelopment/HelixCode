# Vision Auto-Switch - Technical Design

## Overview

Vision Auto-Switch automatically detects when image content is present in user input and switches to a vision-capable model if the current model doesn't support vision. This provides seamless multimodal interaction without requiring users to manually change models.

**References:**
- Qwen Code vision auto-switching implementation
- Anthropic Claude vision capabilities
- OpenAI GPT-4 Vision API

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│               Vision Auto-Switch System                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────────┐
        │        VisionSwitchManager              │
        │  - Detect Images                        │
        │  - Check Model Capabilities             │
        │  - Switch Models                        │
        │  - Track Switch State                   │
        └─────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐     ┌──────────────┐
│   Image      │    │    Model     │     │   Switch     │
│  Detector    │    │  Capability  │     │  Controller  │
│              │    │   Checker    │     │              │
│ • MIME       │    │              │     │ • Once       │
│ • Base64     │    │ • Registry   │     │ • Session    │
│ • Extension  │    │ • Metadata   │     │ • Persist    │
│ • Content    │    │ • Fallback   │     │ • Revert     │
└──────────────┘    └──────────────┘     └──────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐     ┌──────────────┐
│  Detection   │    │    Model     │     │   Switch     │
│   Result     │    │   Registry   │     │   History    │
└──────────────┘    └──────────────┘     └──────────────┘
```

---

## Component Interfaces

### VisionSwitchManager

```go
package vision

import (
    "context"
)

// VisionSwitchManager manages automatic vision model switching
type VisionSwitchManager struct {
    detector       *ImageDetector
    capChecker     *CapabilityChecker
    switchCtrl     *SwitchController
    config         *Config
    currentModel   *Model
    originalModel  *Model
    switchActive   bool
}

// Config contains vision switch configuration
type Config struct {
    // Detection settings
    EnableAutoDetect   bool          // Enable automatic detection
    DetectionMethods   []DetectionMethod // Detection methods to use
    ContentInspection  bool          // Inspect file content

    // Switch behavior
    SwitchMode         SwitchMode    // once, session, persist
    RequireConfirm     bool          // Ask before switching
    FallbackModel      string        // Default vision model
    AllowDowngrade     bool          // Allow switching to less capable model

    // Model preferences
    PreferredVisionModel string      // Preferred vision model
    ModelPriority       []string     // Model selection priority
    ProviderPreference  []string     // Provider preference order

    // Revert settings
    AutoRevert         bool          // Revert when no images
    RevertDelay        time.Duration // Delay before revert
    KeepForSession     bool          // Keep until session ends
}

// SwitchMode defines switch persistence
type SwitchMode string

const (
    SwitchOnce    SwitchMode = "once"    // Switch for single interaction
    SwitchSession SwitchMode = "session" // Switch for current session
    SwitchPersist SwitchMode = "persist" // Persist across sessions
)

// NewVisionSwitchManager creates a new vision switch manager
func NewVisionSwitchManager(config *Config, modelRegistry *ModelRegistry) (*VisionSwitchManager, error)

// ProcessInput checks input for images and switches if needed
func (v *VisionSwitchManager) ProcessInput(ctx context.Context, input *Input) (*SwitchResult, error)

// CheckAndSwitch checks if switch is needed and performs it
func (v *VisionSwitchManager) CheckAndSwitch(ctx context.Context, hasImages bool) (*SwitchResult, error)

// RevertSwitch reverts to the original model
func (v *VisionSwitchManager) RevertSwitch(ctx context.Context) error

// GetCurrentModel returns the currently active model
func (v *VisionSwitchManager) GetCurrentModel() *Model

// IsSwitchActive returns true if a switch is currently active
func (v *VisionSwitchManager) IsSwitchActive() bool

// GetSwitchHistory returns the switch history
func (v *VisionSwitchManager) GetSwitchHistory() []*SwitchEvent
```

### Input Processing

```go
package vision

// Input represents user input to be processed
type Input struct {
    Text        string
    Files       []*File
    Attachments []*Attachment
    Metadata    map[string]interface{}
}

// File represents a file in user input
type File struct {
    Path        string
    Name        string
    Extension   string
    Size        int64
    MIMEType    string
    Content     []byte
    IsImage     bool
}

// Attachment represents an attachment (URL, base64, etc.)
type Attachment struct {
    Type        AttachmentType
    Content     string
    MIMEType    string
    IsImage     bool
}

// AttachmentType categorizes attachments
type AttachmentType string

const (
    AttachmentFile   AttachmentType = "file"
    AttachmentURL    AttachmentType = "url"
    AttachmentBase64 AttachmentType = "base64"
)

// SwitchResult contains the result of switch processing
type SwitchResult struct {
    SwitchPerformed bool
    FromModel       *Model
    ToModel         *Model
    Reason          string
    ImagesDetected  int
    RequiredConfirm bool
    UserConfirmed   bool
    Duration        time.Duration
}
```

### ImageDetector

```go
package vision

import (
    "context"
    "io"
)

// ImageDetector detects images in user input
type ImageDetector struct {
    methods          []DetectionMethod
    contentInspector *ContentInspector
    config           *DetectionConfig
}

// DetectionMethod defines how to detect images
type DetectionMethod string

const (
    DetectByMIME      DetectionMethod = "mime"      // MIME type checking
    DetectByExtension DetectionMethod = "extension" // File extension
    DetectByBase64    DetectionMethod = "base64"    // Base64 pattern
    DetectByContent   DetectionMethod = "content"   // Content inspection
    DetectByURL       DetectionMethod = "url"       // Image URL patterns
)

// DetectionConfig configures image detection
type DetectionConfig struct {
    Methods           []DetectionMethod
    SupportedFormats  []string // jpg, png, gif, webp, etc.
    MaxFileSize       int64    // Maximum file size to inspect
    InspectContent    bool     // Deep content inspection
    URLPatterns       []string // URL patterns to recognize
}

// DetectionResult contains detection results
type DetectionResult struct {
    HasImages       bool
    ImageCount      int
    Images          []*DetectedImage
    DetectionMethod DetectionMethod
    Confidence      float64
}

// DetectedImage represents a detected image
type DetectedImage struct {
    Source      ImageSource
    Location    string  // File path, URL, or identifier
    Format      string  // jpg, png, etc.
    Size        int64
    Dimensions  *Dimensions
    MIMEType    string
    Valid       bool
}

// ImageSource indicates where the image came from
type ImageSource string

const (
    SourceFile       ImageSource = "file"
    SourceURL        ImageSource = "url"
    SourceBase64     ImageSource = "base64"
    SourceClipboard  ImageSource = "clipboard"
)

// Dimensions represents image dimensions
type Dimensions struct {
    Width  int
    Height int
}

// NewImageDetector creates a new image detector
func NewImageDetector(config *DetectionConfig) *ImageDetector

// Detect checks input for images
func (d *ImageDetector) Detect(ctx context.Context, input *Input) (*DetectionResult, error)

// DetectInText looks for image references in text
func (d *ImageDetector) DetectInText(text string) ([]*DetectedImage, error)

// DetectInFile checks if a file is an image
func (d *ImageDetector) DetectInFile(file *File) (bool, error)

// DetectBase64 detects base64-encoded images
func (d *ImageDetector) DetectBase64(content string) ([]*DetectedImage, error)

// ValidateImage validates that detected content is a valid image
func (d *ImageDetector) ValidateImage(reader io.Reader) (bool, string, error)
```

### ContentInspector

```go
package vision

import (
    "io"
)

// ContentInspector performs deep content inspection
type ContentInspector struct {
    signatures map[string][]byte // Magic number signatures
}

// NewContentInspector creates a content inspector
func NewContentInspector() *ContentInspector

// InspectContent checks file content for image signatures
func (c *ContentInspector) InspectContent(reader io.Reader) (*InspectionResult, error)

// InspectionResult contains content inspection results
type InspectionResult struct {
    IsImage     bool
    Format      string
    MIMEType    string
    Confidence  float64
    Dimensions  *Dimensions
}

// Image format magic numbers
var ImageSignatures = map[string][]byte{
    "png":  {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
    "jpg":  {0xFF, 0xD8, 0xFF},
    "gif":  {0x47, 0x49, 0x46, 0x38},
    "webp": {0x52, 0x49, 0x46, 0x46}, // RIFF
    "bmp":  {0x42, 0x4D},
    "tiff": {0x49, 0x49, 0x2A, 0x00}, // Little-endian
}
```

### CapabilityChecker

```go
package vision

import (
    "context"
)

// CapabilityChecker checks model capabilities
type CapabilityChecker struct {
    registry *ModelRegistry
    cache    *CapabilityCache
}

// Model represents a language model
type Model struct {
    ID           string
    Name         string
    Provider     string
    Capabilities *Capabilities
    Metadata     *ModelMetadata
}

// Capabilities defines model capabilities
type Capabilities struct {
    SupportsVision      bool
    SupportsAudio       bool
    SupportsVideo       bool
    MaxImageSize        int64
    MaxImages           int
    SupportedFormats    []string
    ContextWindow       int
    OutputTokens        int
    FunctionCalling     bool
    StreamingSupport    bool
}

// ModelMetadata contains additional model information
type ModelMetadata struct {
    Version         string
    Released        time.Time
    Deprecated      bool
    ReplacementID   string
    Pricing         *Pricing
    RateLimits      *RateLimits
}

// Pricing contains model pricing information
type Pricing struct {
    InputCost    float64 // per 1M tokens
    OutputCost   float64 // per 1M tokens
    ImageCost    float64 // per image (if applicable)
}

// RateLimits defines rate limiting constraints
type RateLimits struct {
    RequestsPerMinute int
    TokensPerMinute   int
    ImagesPerMinute   int
}

// NewCapabilityChecker creates a capability checker
func NewCapabilityChecker(registry *ModelRegistry) *CapabilityChecker

// SupportsVision checks if a model supports vision
func (c *CapabilityChecker) SupportsVision(ctx context.Context, modelID string) (bool, error)

// GetVisionCapabilities returns vision-specific capabilities
func (c *CapabilityChecker) GetVisionCapabilities(modelID string) (*VisionCapabilities, error)

// VisionCapabilities contains vision-specific capabilities
type VisionCapabilities struct {
    MaxImageSize     int64
    MaxImages        int
    SupportedFormats []string
    MaxResolution    *Dimensions
    DetailLevels     []string // low, high, auto
}

// FindBestVisionModel finds the best vision-capable model
func (c *CapabilityChecker) FindBestVisionModel(ctx context.Context, preferences *ModelPreferences) (*Model, error)

// ModelPreferences defines model selection preferences
type ModelPreferences struct {
    Provider         string
    MinContextWindow int
    MaxCost          float64
    RequireStreaming bool
    PreferredModels  []string
}
```

### ModelRegistry

```go
package vision

import (
    "context"
    "sync"
)

// ModelRegistry maintains a registry of available models
type ModelRegistry struct {
    models map[string]*Model
    mu     sync.RWMutex
    cache  *RegistryCache
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *ModelRegistry

// Register registers a model
func (r *ModelRegistry) Register(model *Model) error

// Get retrieves a model by ID
func (r *ModelRegistry) Get(modelID string) (*Model, error)

// List returns all registered models
func (r *ModelRegistry) List(filter *ModelFilter) ([]*Model, error)

// ModelFilter filters model queries
type ModelFilter struct {
    Provider       string
    SupportsVision bool
    SupportsAudio  bool
    MinContext     int
    MaxCost        float64
}

// FindVisionModels returns all vision-capable models
func (r *ModelRegistry) FindVisionModels() ([]*Model, error)

// GetDefaultVisionModel returns the default vision model
func (r *ModelRegistry) GetDefaultVisionModel() (*Model, error)

// Update updates model information (for dynamic capabilities)
func (r *ModelRegistry) Update(modelID string, updates *ModelUpdate) error

// ModelUpdate contains fields to update
type ModelUpdate struct {
    Capabilities *Capabilities
    Metadata     *ModelMetadata
    Deprecated   *bool
}

// Default vision-capable models
var DefaultVisionModels = []*Model{
    {
        ID:       "claude-3-5-sonnet-20241022",
        Name:     "Claude 3.5 Sonnet",
        Provider: "anthropic",
        Capabilities: &Capabilities{
            SupportsVision:   true,
            MaxImageSize:     10 * 1024 * 1024, // 10MB
            MaxImages:        20,
            SupportedFormats: []string{"jpg", "png", "gif", "webp"},
            ContextWindow:    200000,
            OutputTokens:     8192,
        },
    },
    {
        ID:       "gpt-4-vision-preview",
        Name:     "GPT-4 Vision",
        Provider: "openai",
        Capabilities: &Capabilities{
            SupportsVision:   true,
            MaxImageSize:     20 * 1024 * 1024, // 20MB
            MaxImages:        10,
            SupportedFormats: []string{"jpg", "png", "gif", "webp"},
            ContextWindow:    128000,
            OutputTokens:     4096,
        },
    },
    {
        ID:       "gemini-pro-vision",
        Name:     "Gemini Pro Vision",
        Provider: "google",
        Capabilities: &Capabilities{
            SupportsVision:   true,
            MaxImageSize:     4 * 1024 * 1024, // 4MB
            MaxImages:        16,
            SupportedFormats: []string{"jpg", "png", "webp"},
            ContextWindow:    32760,
            OutputTokens:     2048,
        },
    },
}
```

### SwitchController

```go
package vision

import (
    "context"
    "time"
)

// SwitchController manages model switching
type SwitchController struct {
    config        *SwitchConfig
    history       *SwitchHistory
    confirmQueue  *ConfirmQueue
}

// SwitchConfig configures switching behavior
type SwitchConfig struct {
    Mode              SwitchMode
    RequireConfirm    bool
    AutoRevert        bool
    RevertDelay       time.Duration
    MaxSwitchesPerSession int
}

// SwitchHistory tracks model switches
type SwitchHistory struct {
    events []SwitchEvent
    mu     sync.RWMutex
}

// SwitchEvent records a model switch
type SwitchEvent struct {
    ID          string
    Timestamp   time.Time
    FromModel   *Model
    ToModel     *Model
    Reason      SwitchReason
    Mode        SwitchMode
    Reverted    bool
    RevertedAt  *time.Time
    UserConfirmed bool
}

// SwitchReason explains why switch occurred
type SwitchReason string

const (
    ReasonImageDetected  SwitchReason = "image_detected"
    ReasonNoVisionSupport SwitchReason = "no_vision_support"
    ReasonUserRequest    SwitchReason = "user_request"
    ReasonAutoRevert     SwitchReason = "auto_revert"
    ReasonSessionEnd     SwitchReason = "session_end"
)

// NewSwitchController creates a switch controller
func NewSwitchController(config *SwitchConfig) *SwitchController

// Switch performs a model switch
func (s *SwitchController) Switch(ctx context.Context, from, to *Model, reason SwitchReason) (*SwitchEvent, error)

// Revert reverts a model switch
func (s *SwitchController) Revert(ctx context.Context, eventID string) error

// ShouldRevert determines if auto-revert should occur
func (s *SwitchController) ShouldRevert(ctx context.Context) (bool, error)

// GetHistory returns switch history
func (s *SwitchController) GetHistory() []SwitchEvent

// GetActiveSwitch returns the current active switch (if any)
func (s *SwitchController) GetActiveSwitch() *SwitchEvent
```

---

## State Machines

### Auto-Switch State Machine

```
                    ┌─────────┐
                    │  IDLE   │
                    └────┬────┘
                         │
                 Process │Input
                         │
                         ▼
                    ┌─────────┐
                    │ DETECT  │
                    └────┬────┘
                         │
        ┌────────────────┼────────────────┐
        │                                 │
   No   │Images                          │Images
   Detected                              │Detected
        │                                 │
        ▼                                 ▼
   ┌─────────┐                      ┌─────────┐
   │CONTINUE │                      │  CHECK  │
   └─────────┘                      │ VISION  │
                                    └────┬────┘
                                         │
                    ┌────────────────────┼──────────┐
                    │                               │
              Supports│                             │No
              Vision  │                             │Support
                    │                               │
                    ▼                               ▼
               ┌─────────┐                    ┌─────────┐
               │CONTINUE │                    │  FIND   │
               └─────────┘                    │ VISION  │
                                              │  MODEL  │
                                              └────┬────┘
                                                   │
                                            ┌──────┼──────┐
                                            │             │
                                       Found│             │Not
                                            │             │Found
                                            ▼             ▼
                                       ┌─────────┐   ┌─────────┐
                                       │ CONFIRM │   │  ERROR  │
                                       └────┬────┘   └─────────┘
                                            │
                        ┌───────────────────┼───────────┐
                        │                               │
                  Confirmed│                            │Denied
                        │                               │
                        ▼                               ▼
                   ┌─────────┐                     ┌─────────┐
                   │ SWITCH  │                     │CONTINUE │
                   └────┬────┘                     │(NO SWITCH)
                        │                          └─────────┘
                        ▼
                   ┌─────────┐
                   │ ACTIVE  │
                   └────┬────┘
                        │
        ┌───────────────┼───────────────┐
        │                               │
 Auto   │Revert                        │Session
 Trigger│                               │End
        │                               │
        ▼                               ▼
   ┌─────────┐                     ┌─────────┐
   │ REVERT  │                     │ REVERT  │
   └────┬────┘                     └────┬────┘
        │                               │
        └───────────────┬───────────────┘
                        │
                        ▼
                   ┌─────────┐
                   │  IDLE   │
                   └─────────┘
```

### Detection Flow State Machine

```
                    ┌─────────┐
                    │  START  │
                    └────┬────┘
                         │
                         ▼
                    ┌─────────┐
                    │  MIME   │
                    │  CHECK  │
                    └────┬────┘
                         │
            ┌────────────┼────────────┐
            │                         │
       Found│Image                    │Not
            │MIME                     │Image
            │                         │
            ▼                         ▼
       ┌─────────┐              ┌─────────┐
       │DETECTED │              │EXTENSION│
       └─────────┘              │  CHECK  │
                                └────┬────┘
                                     │
                    ┌────────────────┼────────┐
                    │                         │
               Image│Extension                │Not
                    │                         │Image
                    ▼                         ▼
               ┌─────────┐              ┌─────────┐
               │DETECTED │              │ BASE64  │
               └─────────┘              │  CHECK  │
                                        └────┬────┘
                                             │
                            ┌────────────────┼────────┐
                            │                         │
                       Base64│Image                   │Not
                            │                         │Base64
                            ▼                         ▼
                       ┌─────────┐              ┌─────────┐
                       │DETECTED │              │ CONTENT │
                       └─────────┘              │INSPECT  │
                                                └────┬────┘
                                                     │
                                    ┌────────────────┼────────┐
                                    │                         │
                               Valid│Image                    │Not
                                    │                         │Image
                                    ▼                         ▼
                               ┌─────────┐              ┌─────────┐
                               │DETECTED │              │NOT FOUND│
                               └─────────┘              └─────────┘
```

---

## Configuration Schema

### YAML Configuration

```yaml
vision:
  # Auto-switch settings
  auto_switch:
    enabled: true
    mode: session              # once, session, persist
    require_confirm: true
    fallback_model: "claude-3-5-sonnet-20241022"

  # Detection settings
  detection:
    methods:
      - mime
      - extension
      - base64
      - content
    supported_formats:
      - jpg
      - jpeg
      - png
      - gif
      - webp
      - bmp
    max_file_size: 10485760    # 10MB
    inspect_content: true
    url_patterns:
      - "*.jpg"
      - "*.png"
      - "data:image/*"

  # Model preferences
  models:
    preferred: "claude-3-5-sonnet-20241022"
    priority:
      - "claude-3-5-sonnet-20241022"
      - "gpt-4-vision-preview"
      - "gemini-pro-vision"
    provider_preference:
      - anthropic
      - openai
      - google
    allow_downgrade: false

  # Revert settings
  revert:
    auto_revert: true
    revert_delay: 30s          # Delay before auto-revert
    keep_for_session: false    # Keep until session ends
    revert_on_error: true      # Revert if vision model errors

  # Confirmation settings
  confirmation:
    show_model_info: true      # Show model details in prompt
    show_cost_estimate: true   # Show cost comparison
    timeout: 30s               # Confirmation timeout
    default_action: deny       # deny or allow
```

### Go Configuration Struct

```go
type Config struct {
    Vision VisionSettings `yaml:"vision"`
}

type VisionSettings struct {
    AutoSwitch   AutoSwitchSettings   `yaml:"auto_switch"`
    Detection    DetectionSettings    `yaml:"detection"`
    Models       ModelSettings        `yaml:"models"`
    Revert       RevertSettings       `yaml:"revert"`
    Confirmation ConfirmationSettings `yaml:"confirmation"`
}

type AutoSwitchSettings struct {
    Enabled        bool       `yaml:"enabled"`
    Mode           SwitchMode `yaml:"mode"`
    RequireConfirm bool       `yaml:"require_confirm"`
    FallbackModel  string     `yaml:"fallback_model"`
}

type DetectionSettings struct {
    Methods          []DetectionMethod `yaml:"methods"`
    SupportedFormats []string          `yaml:"supported_formats"`
    MaxFileSize      int64             `yaml:"max_file_size"`
    InspectContent   bool              `yaml:"inspect_content"`
    URLPatterns      []string          `yaml:"url_patterns"`
}

type ModelSettings struct {
    Preferred          string   `yaml:"preferred"`
    Priority           []string `yaml:"priority"`
    ProviderPreference []string `yaml:"provider_preference"`
    AllowDowngrade     bool     `yaml:"allow_downgrade"`
}

type RevertSettings struct {
    AutoRevert      bool          `yaml:"auto_revert"`
    RevertDelay     time.Duration `yaml:"revert_delay"`
    KeepForSession  bool          `yaml:"keep_for_session"`
    RevertOnError   bool          `yaml:"revert_on_error"`
}

type ConfirmationSettings struct {
    ShowModelInfo    bool          `yaml:"show_model_info"`
    ShowCostEstimate bool          `yaml:"show_cost_estimate"`
    Timeout          time.Duration `yaml:"timeout"`
    DefaultAction    string        `yaml:"default_action"`
}
```

---

## Error Handling

### Error Types

```go
package vision

import "errors"

var (
    // Detection errors
    ErrNoImagesDetected     = errors.New("no images detected")
    ErrInvalidImageFormat   = errors.New("invalid image format")
    ErrImageTooLarge        = errors.New("image exceeds size limit")
    ErrImageCorrupted       = errors.New("image data is corrupted")

    // Model errors
    ErrModelNotFound        = errors.New("model not found")
    ErrNoVisionSupport      = errors.New("model does not support vision")
    ErrNoVisionModels       = errors.New("no vision-capable models available")
    ErrModelDeprecated      = errors.New("model is deprecated")

    // Switch errors
    ErrSwitchFailed         = errors.New("model switch failed")
    ErrSwitchDenied         = errors.New("user denied model switch")
    ErrSwitchTimeout        = errors.New("switch confirmation timeout")
    ErrAlreadySwitched      = errors.New("already using vision model")

    // Capability errors
    ErrCapabilityCheck      = errors.New("capability check failed")
    ErrRegistryUnavailable  = errors.New("model registry unavailable")
)

// VisionError provides detailed error information
type VisionError struct {
    Op        string // Operation that failed
    ModelID   string // Related model ID
    ImagePath string // Related image path
    Err       error  // Underlying error
    Details   string // Additional details
}

func (e *VisionError) Error() string {
    if e.ImagePath != "" {
        return fmt.Sprintf("%s (model: %s, image: %s): %v - %s",
            e.Op, e.ModelID, e.ImagePath, e.Err, e.Details)
    }
    return fmt.Sprintf("%s (model: %s): %v - %s",
        e.Op, e.ModelID, e.Err, e.Details)
}

func (e *VisionError) Unwrap() error {
    return e.Err
}
```

---

## Testing Strategy

### Unit Tests

```go
package vision_test

import (
    "context"
    "testing"

    "github.com/yourusername/helix/internal/vision"
)

// TestImageDetection tests image detection methods
func TestImageDetection(t *testing.T) {
    config := &vision.DetectionConfig{
        Methods:          []vision.DetectionMethod{
            vision.DetectByMIME,
            vision.DetectByExtension,
        },
        SupportedFormats: []string{"jpg", "png", "gif"},
        InspectContent:   false,
    }

    detector := vision.NewImageDetector(config)
    ctx := context.Background()

    tests := []struct {
        name       string
        input      *vision.Input
        wantImages bool
        wantCount  int
    }{
        {
            name: "detect image file",
            input: &vision.Input{
                Files: []*vision.File{
                    {
                        Path:      "test.jpg",
                        Extension: "jpg",
                        MIMEType:  "image/jpeg",
                    },
                },
            },
            wantImages: true,
            wantCount:  1,
        },
        {
            name: "detect base64 image",
            input: &vision.Input{
                Text: "Here's an image: data:image/png;base64,iVBORw0KGgo...",
            },
            wantImages: true,
            wantCount:  1,
        },
        {
            name: "no images",
            input: &vision.Input{
                Text: "Just text, no images",
                Files: []*vision.File{
                    {
                        Path:      "file.txt",
                        Extension: "txt",
                        MIMEType:  "text/plain",
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

// TestModelCapabilities tests capability checking
func TestModelCapabilities(t *testing.T) {
    registry := vision.NewModelRegistry()

    // Register test models
    registry.Register(&vision.Model{
        ID:   "vision-model",
        Name: "Vision Model",
        Capabilities: &vision.Capabilities{
            SupportsVision: true,
            MaxImages:      10,
        },
    })

    registry.Register(&vision.Model{
        ID:   "text-only-model",
        Name: "Text Only Model",
        Capabilities: &vision.Capabilities{
            SupportsVision: false,
        },
    })

    checker := vision.NewCapabilityChecker(registry)
    ctx := context.Background()

    tests := []struct {
        name          string
        modelID       string
        wantSupports  bool
    }{
        {
            name:         "vision model supports vision",
            modelID:      "vision-model",
            wantSupports: true,
        },
        {
            name:         "text model doesn't support vision",
            modelID:      "text-only-model",
            wantSupports: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            supports, err := checker.SupportsVision(ctx, tt.modelID)
            if err != nil {
                t.Fatalf("SupportsVision() error = %v", err)
            }

            if supports != tt.wantSupports {
                t.Errorf("SupportsVision() = %v, want %v",
                    supports, tt.wantSupports)
            }
        })
    }
}

// TestAutoSwitch tests automatic switching logic
func TestAutoSwitch(t *testing.T) {
    registry := vision.NewModelRegistry()
    registry.Register(&vision.Model{
        ID:   "text-model",
        Capabilities: &vision.Capabilities{
            SupportsVision: false,
        },
    })
    registry.Register(&vision.Model{
        ID:   "vision-model",
        Capabilities: &vision.Capabilities{
            SupportsVision: true,
        },
    })

    config := &vision.Config{
        EnableAutoDetect: true,
        SwitchMode:       vision.SwitchSession,
        RequireConfirm:   false, // Auto-approve for test
        FallbackModel:    "vision-model",
    }

    manager, err := vision.NewVisionSwitchManager(config, registry)
    if err != nil {
        t.Fatalf("NewVisionSwitchManager() error = %v", err)
    }

    ctx := context.Background()

    // Set initial model to text-only
    manager.SetCurrentModel("text-model")

    // Process input with image
    input := &vision.Input{
        Files: []*vision.File{
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
        t.Error("expected switch to be performed")
    }

    if result.ToModel.ID != "vision-model" {
        t.Errorf("switched to %s, want vision-model", result.ToModel.ID)
    }

    if !manager.IsSwitchActive() {
        t.Error("expected switch to be active")
    }
}

// TestContentInspection tests deep content inspection
func TestContentInspection(t *testing.T) {
    inspector := vision.NewContentInspector()

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
```

### Integration Tests

```go
package vision_test

// TestVisionWorkflow tests end-to-end vision switching
func TestVisionWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // Setup
    registry := setupModelRegistry(t)
    config := &vision.Config{
        EnableAutoDetect: true,
        SwitchMode:       vision.SwitchSession,
        RequireConfirm:   false,
        AutoRevert:       true,
        RevertDelay:      100 * time.Millisecond,
    }

    manager, _ := vision.NewVisionSwitchManager(config, registry)

    // Start with text-only model
    textModel, _ := registry.Get("text-only-model")
    manager.SetCurrentModel(textModel)

    // 1. Send text input - no switch
    textInput := &vision.Input{
        Text: "Hello, how are you?",
    }

    result, err := manager.ProcessInput(ctx, textInput)
    if err != nil {
        t.Fatalf("ProcessInput(text) error = %v", err)
    }

    if result.SwitchPerformed {
        t.Error("unexpected switch for text input")
    }

    // 2. Send image input - should switch
    imageInput := &vision.Input{
        Files: []*vision.File{
            {
                Path:     "screenshot.png",
                MIMEType: "image/png",
            },
        },
    }

    result, err = manager.ProcessInput(ctx, imageInput)
    if err != nil {
        t.Fatalf("ProcessInput(image) error = %v", err)
    }

    if !result.SwitchPerformed {
        t.Error("expected switch for image input")
    }

    currentModel := manager.GetCurrentModel()
    if !currentModel.Capabilities.SupportsVision {
        t.Error("current model should support vision")
    }

    // 3. Wait for auto-revert
    time.Sleep(200 * time.Millisecond)

    // Send text input - should trigger revert
    result, err = manager.ProcessInput(ctx, textInput)
    if err != nil {
        t.Fatalf("ProcessInput(text after delay) error = %v", err)
    }

    currentModel = manager.GetCurrentModel()
    if currentModel.ID != textModel.ID {
        t.Errorf("expected revert to %s, got %s",
            textModel.ID, currentModel.ID)
    }

    // 4. Verify switch history
    history := manager.GetSwitchHistory()
    if len(history) != 2 { // switch + revert
        t.Errorf("expected 2 history events, got %d", len(history))
    }
}
```

---

## Performance Considerations

### Optimization Strategies

1. **Detection Caching**
   - Cache detection results for files
   - Use file hash for cache keys
   - TTL-based cache invalidation

2. **Capability Caching**
   - Cache model capabilities
   - Periodic refresh from registry
   - In-memory capability map

3. **Lazy Loading**
   - Load model metadata on demand
   - Defer content inspection
   - Stream large image processing

4. **Parallel Processing**
   - Concurrent file detection
   - Parallel capability checks
   - Async switch operations

### Performance Metrics

```go
// Metrics tracks vision system performance
type Metrics struct {
    DetectionAttempts   atomic.Int64
    ImagesDetected      atomic.Int64
    SwitchesPerformed   atomic.Int64
    SwitchesDenied      atomic.Int64
    AutoReverts         atomic.Int64

    AverageDetectTime   atomic.Int64 // microseconds
    AverageSwitchTime   atomic.Int64 // milliseconds
}

func (m *Metrics) RecordDetection(duration time.Duration, found bool) {
    m.DetectionAttempts.Add(1)
    if found {
        m.ImagesDetected.Add(1)
    }

    current := m.AverageDetectTime.Load()
    newAvg := (current*9 + duration.Microseconds()) / 10
    m.AverageDetectTime.Store(newAvg)
}
```

---

## User Experience Flow

### CLI Interface

```bash
# Check current model capabilities
$ helix model info
Model: gpt-3.5-turbo
Provider: OpenAI
Vision Support: ❌ No

# Send image with auto-switch
$ helix chat "What's in this image?" --file screenshot.png
🔄 Detected image input, but current model doesn't support vision.
   Switch to claude-3-5-sonnet-20241022? (Y/n): y

Switching model...
New model: Claude 3.5 Sonnet
Vision Support: ✅ Yes
Max Images: 20

[Claude's response about the image]

# Check switch status
$ helix model status
Current Model: claude-3-5-sonnet-20241022
Active Switch: Yes (session mode)
Original Model: gpt-3.5-turbo
Switch Reason: Image detected
Duration: 5m 23s

Revert to original model? (y/N): n

# Configure auto-switch
$ helix config set vision.auto_switch.mode persist
Vision auto-switch mode set to: persist

$ helix config set vision.auto_switch.require_confirm false
Auto-switch will no longer require confirmation
```

### TUI Interface

```
┌──────────────────────────────────────────────────────────────┐
│ Model Switch Required                                        │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│ 🖼️  Image detected: screenshot.png (1.2 MB)                 │
│                                                              │
│ Current model (gpt-3.5-turbo) does not support vision.      │
│                                                              │
│ Recommended model: Claude 3.5 Sonnet                         │
│                                                              │
│ ┌──────────────────────────────────────────────────────┐   │
│ │ Comparison:                                          │   │
│ │                                                      │   │
│ │              Current         Recommended            │   │
│ │ Model:       gpt-3.5-turbo  claude-3.5-sonnet       │   │
│ │ Vision:      ❌              ✅                      │   │
│ │ Max Images:  0              20                       │   │
│ │ Context:     16K            200K                     │   │
│ │ Cost/1M:     $0.50          $3.00                    │   │
│ └──────────────────────────────────────────────────────┘   │
│                                                              │
│ Switch mode: ○ Once  ● Session  ○ Persist                  │
│                                                              │
│ [ Switch ] [ Cancel ] [ Configure ]                         │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 1: Core Detection (Week 1)
- [ ] Image detector implementation
- [ ] Content inspector
- [ ] Detection methods (MIME, extension, base64)
- [ ] Unit tests

### Phase 2: Model Registry (Week 2)
- [ ] Model registry
- [ ] Capability checker
- [ ] Default vision models
- [ ] Capability tests

### Phase 3: Switch Logic (Week 3)
- [ ] Switch controller
- [ ] Switch modes (once, session, persist)
- [ ] Auto-revert logic
- [ ] Switch tests

### Phase 4: Integration (Week 4)
- [ ] Vision switch manager
- [ ] Confirmation system
- [ ] Error handling
- [ ] Integration tests

### Phase 5: Polish (Week 5)
- [ ] CLI commands
- [ ] TUI interface
- [ ] Configuration
- [ ] Documentation

---

## Security Considerations

1. **Image Validation**
   - Validate image formats
   - Check file sizes
   - Sanitize file paths
   - Prevent malicious images

2. **Model Access**
   - Validate model IDs
   - Check API permissions
   - Rate limiting
   - Cost controls

3. **Data Privacy**
   - Image data handling
   - Temporary file cleanup
   - No persistent image storage
   - Respect user privacy settings

---

## References

- **Qwen Code**: Vision auto-switching implementation
- **Anthropic Claude**: Vision API capabilities
- **OpenAI GPT-4V**: Vision model integration
- **Google Gemini**: Multi-modal capabilities
- **Image processing**: Magic number detection, MIME types
