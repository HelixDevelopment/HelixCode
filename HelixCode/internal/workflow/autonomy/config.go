package autonomy

import "time"

// Config contains autonomy system configuration
type Config struct {
	// Mode settings
	DefaultMode     AutonomyMode
	AllowModeSwitch bool
	PersistMode     bool
	SessionScoped   bool // Mode only for current session

	// Escalation settings
	AllowEscalation   bool
	AutoDeEscalate    bool // De-escalate after task
	EscalationTimeout time.Duration

	// Safety settings
	EnableGuardrails bool
	RiskThreshold    RiskLevel
	RequireReason    bool // Require reason for risky ops

	// Confirmation settings
	ConfirmRisky  bool
	ConfirmBulk   bool // Confirm bulk operations
	BulkThreshold int  // Number of files for bulk

	// Auto-debug settings
	DebugEnabled    bool
	MaxRetries      int
	RetryDelay      time.Duration
	LearnFromErrors bool

	// Storage
	PersistPath string // Path to persist mode state
}

// NewDefaultConfig returns a Config with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		// Mode settings
		DefaultMode:     ModeSemiAuto,
		AllowModeSwitch: true,
		PersistMode:     true,
		SessionScoped:   false,

		// Escalation settings
		AllowEscalation:   true,
		AutoDeEscalate:    true,
		EscalationTimeout: 1 * time.Hour,

		// Safety settings
		EnableGuardrails: true,
		RiskThreshold:    RiskMedium,
		RequireReason:    true,

		// Confirmation settings
		ConfirmRisky:  true,
		ConfirmBulk:   true,
		BulkThreshold: 5,

		// Auto-debug settings
		DebugEnabled:    true,
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
		LearnFromErrors: true,

		// Storage
		PersistPath: ".helixcode/autonomy.json",
	}
}

// ModeConfig configures mode management
type ModeConfig struct {
	PersistPath    string
	AllowDowngrade bool
	RequireReason  bool
	AuditChanges   bool
}

// NewDefaultModeConfig returns a ModeConfig with sensible defaults
func NewDefaultModeConfig() *ModeConfig {
	return &ModeConfig{
		PersistPath:    ".helixcode/mode.json",
		AllowDowngrade: true,
		RequireReason:  true,
		AuditChanges:   true,
	}
}

// EscalationConfig configures escalation behavior
type EscalationConfig struct {
	AllowEscalation bool
	MaxDuration     time.Duration
	RequireReason   bool
	AutoRevert      bool
	NotifyOnRevert  bool
}

// NewDefaultEscalationConfig returns an EscalationConfig with sensible defaults
func NewDefaultEscalationConfig() *EscalationConfig {
	return &EscalationConfig{
		AllowEscalation: true,
		MaxDuration:     1 * time.Hour,
		RequireReason:   true,
		AutoRevert:      true,
		NotifyOnRevert:  true,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if !c.DefaultMode.IsValid() {
		return ErrInvalidMode
	}

	if c.BulkThreshold < 1 {
		c.BulkThreshold = 5
	}

	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}

	if c.EscalationTimeout < 0 {
		c.EscalationTimeout = 1 * time.Hour
	}

	if c.RetryDelay < 0 {
		c.RetryDelay = 2 * time.Second
	}

	return nil
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}
