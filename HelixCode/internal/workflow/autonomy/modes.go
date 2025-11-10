package autonomy

import "fmt"

// AutonomyMode represents the level of AI autonomy
type AutonomyMode string

const (
	// ModeNone - Complete manual control (Level 1)
	// AI provides suggestions but takes no automatic actions
	ModeNone AutonomyMode = "none"

	// ModeBasic - Basic automation (Level 2)
	// AI can load context automatically but requires approval for actions
	ModeBasic AutonomyMode = "basic"

	// ModeBasicPlus - Enhanced basic automation (Level 3)
	// AI can load context and apply simple changes automatically
	ModeBasicPlus AutonomyMode = "basic_plus"

	// ModeSemiAuto - Semi-autonomous operation (Level 4)
	// AI can load context, apply changes, and execute safe commands
	// Default mode - best for most development workflows
	ModeSemiAuto AutonomyMode = "semi_auto"

	// ModeFullAuto - Full autonomy (Level 5)
	// AI operates independently with automatic error recovery
	ModeFullAuto AutonomyMode = "full_auto"
)

// ModeCapabilities defines what each mode can do
type ModeCapabilities struct {
	Mode           AutonomyMode
	AutoContext    bool // Automatically load relevant context
	AutoApply      bool // Automatically apply code changes
	AutoExecute    bool // Automatically execute commands
	AutoDebug      bool // Automatically retry on errors
	MaxRetries     int  // Maximum automatic retry attempts
	RequireConfirm bool // Require user confirmation
	AllowRisky     bool // Allow risky operations
	AutoEscalate   bool // Can escalate to higher mode
	IterationLimit int  // Maximum automatic iterations
}

// GetCapabilities returns the capabilities for a mode
func GetCapabilities(mode AutonomyMode) *ModeCapabilities {
	switch mode {
	case ModeNone:
		return &ModeCapabilities{
			Mode:           ModeNone,
			AutoContext:    false,
			AutoApply:      false,
			AutoExecute:    false,
			AutoDebug:      false,
			MaxRetries:     0,
			RequireConfirm: true,
			AllowRisky:     false,
			AutoEscalate:   false,
			IterationLimit: 0,
		}

	case ModeBasic:
		return &ModeCapabilities{
			Mode:           ModeBasic,
			AutoContext:    false, // Manual context gathering
			AutoApply:      false, // Must ask before changes
			AutoExecute:    false, // Must ask before commands
			AutoDebug:      false, // No auto-retry
			MaxRetries:     0,
			RequireConfirm: true,
			AllowRisky:     false,
			AutoEscalate:   true, // Can ask to escalate
			IterationLimit: 1,
		}

	case ModeBasicPlus:
		return &ModeCapabilities{
			Mode:           ModeBasicPlus,
			AutoContext:    false, // Context suggestions provided
			AutoApply:      false, // Apply with confirmation
			AutoExecute:    false, // Still asks for commands
			AutoDebug:      false,
			MaxRetries:     0,
			RequireConfirm: true, // Confirm risky operations
			AllowRisky:     false,
			AutoEscalate:   true,
			IterationLimit: 5,
		}

	case ModeSemiAuto:
		return &ModeCapabilities{
			Mode:           ModeSemiAuto,
			AutoContext:    true,  // Automatic context gathering
			AutoApply:      false, // Manual approval for changes (one-click)
			AutoExecute:    false, // Manual approval for execution
			AutoDebug:      false, // No auto-retry
			MaxRetries:     0,
			RequireConfirm: true, // Confirm before apply
			AllowRisky:     false,
			AutoEscalate:   true,
			IterationLimit: 10,
		}

	case ModeFullAuto:
		return &ModeCapabilities{
			Mode:           ModeFullAuto,
			AutoContext:    true,
			AutoApply:      true,
			AutoExecute:    true,
			AutoDebug:      true,
			MaxRetries:     5,
			RequireConfirm: false, // No confirmation needed
			AllowRisky:     true,  // Can do risky operations
			AutoEscalate:   false, // Already at max
			IterationLimit: -1,    // Unlimited
		}

	default:
		return GetCapabilities(ModeBasic) // Safe default
	}
}

// IsValid checks if a mode is valid
func (m AutonomyMode) IsValid() bool {
	switch m {
	case ModeNone, ModeBasic, ModeBasicPlus, ModeSemiAuto, ModeFullAuto:
		return true
	default:
		return false
	}
}

// Level returns the numeric level of the mode (1-5)
func (m AutonomyMode) Level() int {
	switch m {
	case ModeNone:
		return 1
	case ModeBasic:
		return 2
	case ModeBasicPlus:
		return 3
	case ModeSemiAuto:
		return 4
	case ModeFullAuto:
		return 5
	default:
		return 0
	}
}

// String returns a human-readable string for the mode
func (m AutonomyMode) String() string {
	switch m {
	case ModeNone:
		return "None (Manual Control)"
	case ModeBasic:
		return "Basic (Manual Steps)"
	case ModeBasicPlus:
		return "Basic Plus (Smart Semi-Automation)"
	case ModeSemiAuto:
		return "Semi Auto (Automated with Approval)"
	case ModeFullAuto:
		return "Full Auto (Fully Autonomous)"
	default:
		return "Unknown"
	}
}

// CanTransitionTo checks if transition to target mode is allowed
func (m AutonomyMode) CanTransitionTo(target AutonomyMode) error {
	if !m.IsValid() {
		return fmt.Errorf("invalid source mode: %s", m)
	}
	if !target.IsValid() {
		return fmt.Errorf("invalid target mode: %s", target)
	}
	// All transitions are allowed
	return nil
}

// Compare compares two modes by level
// Returns: -1 if m < other, 0 if m == other, 1 if m > other
func (m AutonomyMode) Compare(other AutonomyMode) int {
	mLevel := m.Level()
	otherLevel := other.Level()

	if mLevel < otherLevel {
		return -1
	} else if mLevel > otherLevel {
		return 1
	}
	return 0
}

// GetDefaultMode returns the default autonomy mode
func GetDefaultMode() AutonomyMode {
	return ModeSemiAuto
}

// ParseMode parses a string into an AutonomyMode
func ParseMode(s string) (AutonomyMode, error) {
	mode := AutonomyMode(s)
	if !mode.IsValid() {
		return "", fmt.Errorf("invalid autonomy mode: %s", s)
	}
	return mode, nil
}

// AllModes returns all valid autonomy modes in order
func AllModes() []AutonomyMode {
	return []AutonomyMode{
		ModeNone,
		ModeBasic,
		ModeBasicPlus,
		ModeSemiAuto,
		ModeFullAuto,
	}
}
