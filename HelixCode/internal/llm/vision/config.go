package vision

import (
	"fmt"
	"time"
)

// SwitchMode defines switch persistence behavior
type SwitchMode string

const (
	// SwitchOnce switches for a single request only
	SwitchOnce SwitchMode = "once"

	// SwitchSession switches for the current session
	SwitchSession SwitchMode = "session"

	// SwitchPersist permanently switches and saves to config
	SwitchPersist SwitchMode = "persist"
)

// String returns string representation of switch mode
func (s SwitchMode) String() string {
	return string(s)
}

// IsValid checks if switch mode is valid
func (s SwitchMode) IsValid() bool {
	switch s {
	case SwitchOnce, SwitchSession, SwitchPersist:
		return true
	default:
		return false
	}
}

// Config contains vision auto-switch configuration
type Config struct {
	// Detection settings
	EnableAutoDetect   bool             `yaml:"enable_auto_detect" json:"enable_auto_detect"`
	DetectionMethods   []DetectionMethod `yaml:"detection_methods" json:"detection_methods"`
	ContentInspection  bool             `yaml:"content_inspection" json:"content_inspection"`

	// Switch behavior
	SwitchMode         SwitchMode       `yaml:"switch_mode" json:"switch_mode"`
	RequireConfirm     bool             `yaml:"require_confirm" json:"require_confirm"`
	FallbackModel      string           `yaml:"fallback_model" json:"fallback_model"`
	AllowDowngrade     bool             `yaml:"allow_downgrade" json:"allow_downgrade"`

	// Model preferences
	PreferredVisionModel string         `yaml:"preferred_vision_model" json:"preferred_vision_model"`
	ModelPriority       []string        `yaml:"model_priority" json:"model_priority"`
	ProviderPreference  []string        `yaml:"provider_preference" json:"provider_preference"`

	// Revert settings
	AutoRevert         bool             `yaml:"auto_revert" json:"auto_revert"`
	RevertDelay        time.Duration    `yaml:"revert_delay" json:"revert_delay"`
	KeepForSession     bool             `yaml:"keep_for_session" json:"keep_for_session"`
}

// DetectionConfig configures image detection
type DetectionConfig struct {
	Methods           []DetectionMethod `yaml:"methods" json:"methods"`
	SupportedFormats  []string         `yaml:"supported_formats" json:"supported_formats"`
	MaxFileSize       int64            `yaml:"max_file_size" json:"max_file_size"`
	InspectContent    bool             `yaml:"inspect_content" json:"inspect_content"`
	URLPatterns       []string         `yaml:"url_patterns" json:"url_patterns"`
}

// SwitchConfig configures switching behavior
type SwitchConfig struct {
	Mode              SwitchMode    `yaml:"mode" json:"mode"`
	RequireConfirm    bool          `yaml:"require_confirm" json:"require_confirm"`
	AutoRevert        bool          `yaml:"auto_revert" json:"auto_revert"`
	RevertDelay       time.Duration `yaml:"revert_delay" json:"revert_delay"`
	MaxSwitchesPerSession int       `yaml:"max_switches_per_session" json:"max_switches_per_session"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		EnableAutoDetect: true,
		DetectionMethods: []DetectionMethod{
			DetectByMIME,
			DetectByExtension,
			DetectByBase64,
		},
		ContentInspection:    false,
		SwitchMode:           SwitchSession,
		RequireConfirm:       true,
		FallbackModel:        "claude-3-5-sonnet-20241022",
		AllowDowngrade:       false,
		PreferredVisionModel: "claude-3-5-sonnet-20241022",
		ModelPriority: []string{
			"claude-3-5-sonnet-20241022",
			"gpt-4o",
			"gemini-2.0-flash",
		},
		ProviderPreference: []string{
			"anthropic",
			"openai",
			"google",
		},
		AutoRevert:      false,
		RevertDelay:     5 * time.Minute,
		KeepForSession:  true,
	}
}

// DefaultDetectionConfig returns default detection configuration
func DefaultDetectionConfig() *DetectionConfig {
	return &DetectionConfig{
		Methods: []DetectionMethod{
			DetectByMIME,
			DetectByExtension,
			DetectByBase64,
		},
		SupportedFormats: []string{
			"jpg", "jpeg", "png", "gif", "webp", "bmp",
		},
		MaxFileSize:    10 * 1024 * 1024, // 10MB
		InspectContent: false,
		URLPatterns: []string{
			"*.jpg", "*.jpeg", "*.png", "*.gif", "*.webp",
			"data:image/*",
		},
	}
}

// DefaultSwitchConfig returns default switch configuration
func DefaultSwitchConfig() *SwitchConfig {
	return &SwitchConfig{
		Mode:                  SwitchSession,
		RequireConfirm:        true,
		AutoRevert:            false,
		RevertDelay:           5 * time.Minute,
		MaxSwitchesPerSession: 10,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if !c.SwitchMode.IsValid() {
		return fmt.Errorf("invalid switch mode: %s", c.SwitchMode)
	}

	if len(c.DetectionMethods) == 0 {
		return fmt.Errorf("at least one detection method must be enabled")
	}

	if c.FallbackModel == "" && c.PreferredVisionModel == "" {
		return fmt.Errorf("either fallback_model or preferred_vision_model must be set")
	}

	if c.RevertDelay < 0 {
		return fmt.Errorf("revert_delay cannot be negative")
	}

	return nil
}

// Validate validates detection configuration
func (dc *DetectionConfig) Validate() error {
	if len(dc.Methods) == 0 {
		return fmt.Errorf("at least one detection method must be enabled")
	}

	if dc.MaxFileSize <= 0 {
		return fmt.Errorf("max_file_size must be positive")
	}

	if len(dc.SupportedFormats) == 0 {
		return fmt.Errorf("at least one supported format must be specified")
	}

	return nil
}

// Validate validates switch configuration
func (sc *SwitchConfig) Validate() error {
	if !sc.Mode.IsValid() {
		return fmt.Errorf("invalid switch mode: %s", sc.Mode)
	}

	if sc.RevertDelay < 0 {
		return fmt.Errorf("revert_delay cannot be negative")
	}

	if sc.MaxSwitchesPerSession < 0 {
		return fmt.Errorf("max_switches_per_session cannot be negative")
	}

	return nil
}
