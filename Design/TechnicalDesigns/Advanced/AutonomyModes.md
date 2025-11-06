# Autonomy Modes - Technical Design

## Overview

The Autonomy Modes system provides a five-level spectrum of AI assistance, from fully manual control to completely autonomous operation. This design enables users to control how much the AI can do independently, balancing automation with user oversight.

**References:**
- Plandex autonomy modes (plan_config.go)
- Agentic framework patterns
- Human-in-the-loop design principles

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Autonomy System                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────────┐
        │         AutonomyController              │
        │  - Mode Management                      │
        │  - Permission Checking                  │
        │  - Auto-escalation                      │
        │  - Guardrails                           │
        └─────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐     ┌──────────────┐
│     Mode     │    │  Permission  │     │   Action     │
│   Manager    │    │   Manager    │     │  Executor    │
│              │    │              │     │              │
│ • Modes      │    │ • Check      │     │ • Context    │
│ • Switching  │    │ • Request    │     │ • Apply      │
│ • Config     │    │ • Grant      │     │ • Execute    │
│ • Persist    │    │ • Deny       │     │ • Debug      │
└──────────────┘    └──────────────┘     └──────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐     ┌──────────────┐
│ Escalation   │    │  Guardrails  │     │   Metrics    │
│   Engine     │    │   Checker    │     │   Tracker    │
└──────────────┘    └──────────────┘     └──────────────┘
```

---

## Autonomy Levels

### Five Modes

```go
package autonomy

// AutonomyMode represents the level of AI autonomy
type AutonomyMode string

const (
    // ModeNone - Manual control only
    // AI provides suggestions but takes no automatic actions
    ModeNone AutonomyMode = "none"

    // ModeBasic - Basic automation
    // AI can load context automatically but requires approval for actions
    ModeBasic AutonomyMode = "basic"

    // ModeBasicPlus - Enhanced basic automation
    // AI can load context and apply simple changes automatically
    ModeBasicPlus AutonomyMode = "basic_plus"

    // ModeSemiAuto - Semi-autonomous operation
    // AI can load context, apply changes, and execute safe commands
    ModeSemiAuto AutonomyMode = "semi_auto"

    // ModeFullAuto - Full autonomy
    // AI operates independently with automatic error recovery
    ModeFullAuto AutonomyMode = "full_auto"
)

// ModeCapabilities defines what each mode can do
type ModeCapabilities struct {
    Mode            AutonomyMode
    AutoContext     bool  // Automatically load relevant context
    AutoApply       bool  // Automatically apply code changes
    AutoExecute     bool  // Automatically execute commands
    AutoDebug       bool  // Automatically retry on errors
    MaxRetries      int   // Maximum automatic retry attempts
    RequireConfirm  bool  // Require user confirmation
    AllowRisky      bool  // Allow risky operations
    AutoEscalate    bool  // Can escalate to higher mode
}

// GetCapabilities returns the capabilities for a mode
func GetCapabilities(mode AutonomyMode) *ModeCapabilities {
    switch mode {
    case ModeNone:
        return &ModeCapabilities{
            Mode:            ModeNone,
            AutoContext:     false,
            AutoApply:       false,
            AutoExecute:     false,
            AutoDebug:       false,
            MaxRetries:      0,
            RequireConfirm:  true,
            AllowRisky:      false,
            AutoEscalate:    false,
        }

    case ModeBasic:
        return &ModeCapabilities{
            Mode:            ModeBasic,
            AutoContext:     true,   // Can load context
            AutoApply:       false,  // Must ask before changes
            AutoExecute:     false,  // Must ask before commands
            AutoDebug:       false,  // No auto-retry
            MaxRetries:      0,
            RequireConfirm:  true,
            AllowRisky:      false,
            AutoEscalate:    true,   // Can ask to escalate
        }

    case ModeBasicPlus:
        return &ModeCapabilities{
            Mode:            ModeBasicPlus,
            AutoContext:     true,
            AutoApply:       true,   // Can apply safe changes
            AutoExecute:     false,  // Still asks for commands
            AutoDebug:       false,
            MaxRetries:      0,
            RequireConfirm:  true,   // Confirm risky operations
            AllowRisky:      false,
            AutoEscalate:    true,
        }

    case ModeSemiAuto:
        return &ModeCapabilities{
            Mode:            ModeSemiAuto,
            AutoContext:     true,
            AutoApply:       true,
            AutoExecute:     true,   // Can run safe commands
            AutoDebug:       true,   // Can retry on errors
            MaxRetries:      3,
            RequireConfirm:  true,   // Only for risky ops
            AllowRisky:      false,
            AutoEscalate:    true,
        }

    case ModeFullAuto:
        return &ModeCapabilities{
            Mode:            ModeFullAuto,
            AutoContext:     true,
            AutoApply:       true,
            AutoExecute:     true,
            AutoDebug:       true,
            MaxRetries:      5,
            RequireConfirm:  false,  // No confirmation needed
            AllowRisky:      true,   // Can do risky operations
            AutoEscalate:    false,  // Already at max
        }

    default:
        return GetCapabilities(ModeBasic) // Safe default
    }
}
```

---

## Component Interfaces

### AutonomyController

```go
package autonomy

import (
    "context"
)

// AutonomyController manages autonomy modes and permissions
type AutonomyController struct {
    modeManager   *ModeManager
    permManager   *PermissionManager
    executor      *ActionExecutor
    escalator     *EscalationEngine
    guardrails    *GuardrailsChecker
    config        *Config
    metrics       *Metrics
}

// Config contains autonomy system configuration
type Config struct {
    // Mode settings
    DefaultMode       AutonomyMode
    AllowModeSwitch   bool
    PersistMode       bool
    SessionScoped     bool  // Mode only for current session

    // Escalation settings
    AllowEscalation   bool
    AutoDeEscalate    bool  // De-escalate after task
    EscalationTimeout time.Duration

    // Safety settings
    EnableGuardrails  bool
    RiskThreshold     RiskLevel
    RequireReason     bool  // Require reason for risky ops

    // Confirmation settings
    ConfirmRisky      bool
    ConfirmBulk       bool  // Confirm bulk operations
    BulkThreshold     int   // Number of files for bulk

    // Auto-debug settings
    DebugEnabled      bool
    MaxRetries        int
    RetryDelay        time.Duration
    LearnFromErrors   bool
}

// NewAutonomyController creates a new autonomy controller
func NewAutonomyController(config *Config) (*AutonomyController, error)

// GetCurrentMode returns the active autonomy mode
func (a *AutonomyController) GetCurrentMode() AutonomyMode

// SetMode changes the autonomy mode
func (a *AutonomyController) SetMode(ctx context.Context, mode AutonomyMode) error

// RequestPermission checks if an action is permitted
func (a *AutonomyController) RequestPermission(ctx context.Context, action *Action) (*Permission, error)

// ExecuteAction executes an action with appropriate permissions
func (a *AutonomyController) ExecuteAction(ctx context.Context, action *Action) (*ActionResult, error)

// RequestEscalation requests temporary mode escalation
func (a *AutonomyController) RequestEscalation(ctx context.Context, reason string, duration time.Duration) error

// DeEscalate returns to previous mode
func (a *AutonomyController) DeEscalate(ctx context.Context) error
```

### Action Definition

```go
package autonomy

// Action represents an operation requiring permission
type Action struct {
    Type        ActionType
    Description string
    Risk        RiskLevel
    Context     *ActionContext
    Metadata    map[string]interface{}
}

// ActionType categorizes actions
type ActionType string

const (
    ActionLoadContext   ActionType = "load_context"
    ActionApplyChange   ActionType = "apply_change"
    ActionExecuteCmd    ActionType = "execute_command"
    ActionDebugRetry    ActionType = "debug_retry"
    ActionFileDelete    ActionType = "file_delete"
    ActionBulkEdit      ActionType = "bulk_edit"
    ActionNetworkCall   ActionType = "network_call"
    ActionSystemChange  ActionType = "system_change"
)

// RiskLevel categorizes action risk
type RiskLevel string

const (
    RiskNone     RiskLevel = "none"      // No risk
    RiskLow      RiskLevel = "low"       // Low risk, easily reversible
    RiskMedium   RiskLevel = "medium"    // Medium risk, may need effort to reverse
    RiskHigh     RiskLevel = "high"      // High risk, difficult to reverse
    RiskCritical RiskLevel = "critical"  // Critical risk, potentially destructive
)

// ActionContext provides context for permission decisions
type ActionContext struct {
    TaskID        string
    StepNumber    int
    FilesAffected []string
    CommandToRun  string
    ExpectedOutcome string
    Reversible    bool
}

// Permission represents the result of a permission check
type Permission struct {
    Granted       bool
    Reason        string
    RequiresConfirm bool
    ConfirmPrompt string
    Conditions    []Condition
    ExpiresAt     time.Time
}

// Condition is a requirement for permission
type Condition struct {
    Type        ConditionType
    Description string
    Met         bool
}

// ConditionType categorizes permission conditions
type ConditionType string

const (
    ConditionUserConfirm  ConditionType = "user_confirm"
    ConditionBackupExists ConditionType = "backup_exists"
    ConditionTestsPass    ConditionType = "tests_pass"
    ConditionReviewable   ConditionType = "reviewable"
)
```

### ModeManager

```go
package autonomy

import (
    "context"
    "time"
)

// ModeManager handles mode switching and persistence
type ModeManager struct {
    currentMode   AutonomyMode
    previousMode  AutonomyMode
    sessionMode   AutonomyMode
    persistentMode AutonomyMode
    config        *ModeConfig
    history       *ModeHistory
}

// ModeConfig configures mode management
type ModeConfig struct {
    PersistPath      string
    AllowDowngrade   bool
    RequireReason    bool
    AuditChanges     bool
}

// ModeHistory tracks mode changes
type ModeHistory struct {
    Changes []ModeChange
}

// ModeChange records a mode transition
type ModeChange struct {
    From      AutonomyMode
    To        AutonomyMode
    Timestamp time.Time
    Reason    string
    Duration  time.Duration
    UserID    string
}

// NewModeManager creates a new mode manager
func NewModeManager(config *ModeConfig) (*ModeManager, error)

// GetMode returns the current mode
func (m *ModeManager) GetMode() AutonomyMode

// SetMode changes the active mode
func (m *ModeManager) SetMode(ctx context.Context, mode AutonomyMode, reason string) error

// TemporaryMode sets a temporary mode with auto-revert
func (m *ModeManager) TemporaryMode(ctx context.Context, mode AutonomyMode, duration time.Duration) error

// RevertMode returns to the previous mode
func (m *ModeManager) RevertMode(ctx context.Context) error

// SaveMode persists the current mode
func (m *ModeManager) SaveMode(ctx context.Context) error

// LoadMode loads the persisted mode
func (m *ModeManager) LoadMode(ctx context.Context) (AutonomyMode, error)

// GetHistory returns mode change history
func (m *ModeManager) GetHistory() *ModeHistory
```

### PermissionManager

```go
package autonomy

import (
    "context"
)

// PermissionManager handles permission checks
type PermissionManager struct {
    capabilities *ModeCapabilities
    guardrails   *GuardrailsChecker
    confirmQueue *ConfirmQueue
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(mode AutonomyMode, guardrails *GuardrailsChecker) *PermissionManager

// Check determines if an action is permitted
func (p *PermissionManager) Check(ctx context.Context, action *Action) (*Permission, error)

// RequestConfirmation asks user to confirm an action
func (p *PermissionManager) RequestConfirmation(ctx context.Context, action *Action) (bool, error)

// GrantPermission explicitly grants permission
func (p *PermissionManager) GrantPermission(ctx context.Context, action *Action, duration time.Duration) error

// RevokePermission revokes a granted permission
func (p *PermissionManager) RevokePermission(ctx context.Context, actionType ActionType) error

// UpdateCapabilities updates capabilities for mode change
func (p *PermissionManager) UpdateCapabilities(capabilities *ModeCapabilities)
```

### ActionExecutor

```go
package autonomy

import (
    "context"
)

// ActionExecutor executes actions with proper permission checks
type ActionExecutor struct {
    permManager *PermissionManager
    retryEngine *RetryEngine
    metrics     *Metrics
}

// ActionResult contains execution results
type ActionResult struct {
    Success      bool
    Action       *Action
    Output       string
    Error        error
    Duration     time.Duration
    Retries      int
    Confirmed    bool
}

// NewActionExecutor creates a new action executor
func NewActionExecutor(permManager *PermissionManager) *ActionExecutor

// Execute runs an action with permission checking
func (a *ActionExecutor) Execute(ctx context.Context, action *Action) (*ActionResult, error)

// ExecuteWithRetry executes with automatic retry on failure
func (a *ActionExecutor) ExecuteWithRetry(ctx context.Context, action *Action, maxRetries int) (*ActionResult, error)

// CanExecuteAutomatically checks if action can run without confirmation
func (a *ActionExecutor) CanExecuteAutomatically(action *Action) bool

// LoadContext automatically loads relevant context
func (a *ActionExecutor) LoadContext(ctx context.Context, task string) error

// ApplyChange automatically applies code changes
func (a *ActionExecutor) ApplyChange(ctx context.Context, change *CodeChange) error

// ExecuteCommand runs a command with safety checks
func (a *ActionExecutor) ExecuteCommand(ctx context.Context, cmd string) (*ActionResult, error)

// CodeChange represents a code modification
type CodeChange struct {
    FilePath    string
    OldContent  string
    NewContent  string
    Description string
    Reversible  bool
}
```

### GuardrailsChecker

```go
package autonomy

import (
    "context"
)

// GuardrailsChecker enforces safety constraints
type GuardrailsChecker struct {
    rules       []GuardrailRule
    violations  *ViolationTracker
}

// GuardrailRule defines a safety constraint
type GuardrailRule struct {
    Name        string
    Description string
    Check       func(context.Context, *Action) (bool, string)
    Severity    RiskLevel
    Enabled     bool
}

// ViolationTracker records guardrail violations
type ViolationTracker struct {
    violations []Violation
}

// Violation represents a guardrail breach
type Violation struct {
    Rule      string
    Action    *Action
    Timestamp time.Time
    Severity  RiskLevel
    Allowed   bool
    Reason    string
}

// NewGuardrailsChecker creates a checker with default rules
func NewGuardrailsChecker() *GuardrailsChecker

// Check verifies action against all rules
func (g *GuardrailsChecker) Check(ctx context.Context, action *Action) (bool, []string, error)

// AddRule adds a custom guardrail rule
func (g *GuardrailsChecker) AddRule(rule GuardrailRule)

// DisableRule disables a specific rule
func (g *GuardrailsChecker) DisableRule(name string)

// GetViolations returns recent violations
func (g *GuardrailsChecker) GetViolations() []Violation

// Default guardrail rules
var DefaultGuardrails = []GuardrailRule{
    {
        Name:        "no_system_file_delete",
        Description: "Prevent deletion of system files",
        Severity:    RiskCritical,
        Check: func(ctx context.Context, action *Action) (bool, string) {
            if action.Type != ActionFileDelete {
                return true, ""
            }
            // Check if file is system file
            // Return false if system file
            return true, ""
        },
    },
    {
        Name:        "no_bulk_unreviewed",
        Description: "Prevent bulk changes without review",
        Severity:    RiskHigh,
        Check: func(ctx context.Context, action *Action) (bool, string) {
            if action.Type != ActionBulkEdit {
                return true, ""
            }
            if len(action.Context.FilesAffected) > 10 {
                return false, "bulk operation affects too many files"
            }
            return true, ""
        },
    },
    // ... more default rules
}
```

### EscalationEngine

```go
package autonomy

import (
    "context"
    "time"
)

// EscalationEngine handles temporary mode escalation
type EscalationEngine struct {
    modeManager   *ModeManager
    escalations   map[string]*Escalation
    config        *EscalationConfig
}

// EscalationConfig configures escalation behavior
type EscalationConfig struct {
    AllowEscalation bool
    MaxDuration     time.Duration
    RequireReason   bool
    AutoRevert      bool
    NotifyOnRevert  bool
}

// Escalation represents a temporary mode increase
type Escalation struct {
    ID          string
    From        AutonomyMode
    To          AutonomyMode
    Reason      string
    StartTime   time.Time
    Duration    time.Duration
    ExpiresAt   time.Time
    Active      bool
}

// NewEscalationEngine creates an escalation engine
func NewEscalationEngine(modeManager *ModeManager, config *EscalationConfig) *EscalationEngine

// Request requests a temporary escalation
func (e *EscalationEngine) Request(ctx context.Context, targetMode AutonomyMode, reason string, duration time.Duration) (*Escalation, error)

// Approve approves an escalation request
func (e *EscalationEngine) Approve(ctx context.Context, escalationID string) error

// Deny denies an escalation request
func (e *EscalationEngine) Deny(ctx context.Context, escalationID string, reason string) error

// Revert manually reverts an escalation
func (e *EscalationEngine) Revert(ctx context.Context, escalationID string) error

// CheckExpired checks and reverts expired escalations
func (e *EscalationEngine) CheckExpired(ctx context.Context) error

// GetActive returns active escalations
func (e *EscalationEngine) GetActive() []*Escalation
```

---

## State Machines

### Mode Transition State Machine

```
                    ┌─────────┐
                    │  NONE   │
                    └────┬────┘
                         │
                  Upgrade│
                         │
                         ▼
                    ┌─────────┐
              ┌────▶│  BASIC  │◀────┐
              │     └────┬────┘     │
              │          │          │
        Down- │   Upgrade│          │ Down-
        grade │          │          │ grade
              │          ▼          │
              │     ┌──────────┐    │
              │     │BASIC PLUS│    │
              │     └────┬─────┘    │
              │          │          │
              │   Upgrade│          │
              │          │          │
              │          ▼          │
              │     ┌─────────┐     │
              │  ┌─▶│SEMI AUTO│─┐   │
              │  │  └────┬────┘ │   │
              │  │       │      │   │
              │  │Upgrade│      │   │
              │  │       │      │   │
     Revert   │  │       ▼      │   │ Revert
              │  │  ┌─────────┐ │   │
              └──┼──│FULL AUTO│◀┘   │
                 │  └─────────┘     │
                 │                  │
                 └──────────────────┘
                    (Escalation
                     Auto-Revert)
```

### Permission Check State Machine

```
                    ┌─────────┐
                    │ REQUEST │
                    └────┬────┘
                         │
                  Check  │Capabilities
                         │
                         ▼
                    ┌─────────┐
                    │  CHECK  │
                    └────┬────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   Denied│               │Allowed         │May Allow
        │                │                │
        ▼                ▼                ▼
   ┌─────────┐      ┌─────────┐     ┌─────────┐
   │  DENY   │      │  GRANT  │     │GUARDRAIL│
   └─────────┘      └─────────┘     └────┬────┘
                                          │
                            ┌─────────────┼─────────┐
                            │                       │
                       Passed│                      │Failed
                            │                       │
                            ▼                       ▼
                       ┌─────────┐             ┌─────────┐
                       │ CONFIRM │             │  DENY   │
                       └────┬────┘             └─────────┘
                            │
                ┌───────────┼───────────┐
                │                       │
          Approved│                     │Rejected
                │                       │
                ▼                       ▼
           ┌─────────┐             ┌─────────┐
           │  GRANT  │             │  DENY   │
           └─────────┘             └─────────┘
```

### Auto-Debug State Machine

```
                    ┌─────────┐
                    │ EXECUTE │
                    └────┬────┘
                         │
                Execute  │Action
                         │
                         ▼
                    ┌─────────┐
                    │ RUNNING │
                    └────┬────┘
                         │
            ┌────────────┼────────────┐
            │                         │
       Error│                         │Success
            │                         │
            ▼                         ▼
       ┌─────────┐                ┌─────────┐
       │  ERROR  │                │ SUCCESS │
       └────┬────┘                └─────────┘
            │
    Auto-   │Debug?
    Debug   │
    Enabled?│
            ▼
       ┌─────────┐
       │ ANALYZE │
       └────┬────┘
            │
            │Determine Fix
            │
            ▼
       ┌─────────┐
       │  RETRY  │
       └────┬────┘
            │
    ┌───────┼───────┐
    │               │
Max │               │More
Retries             │Attempts
Reached│            │
    │               │
    ▼               │
┌─────────┐         │
│  FAIL   │         │
└─────────┘         │
                    │
                    └────────▶ EXECUTE
                              (Retry)
```

---

## Configuration Schema

### YAML Configuration

```yaml
autonomy:
  # Mode settings
  mode:
    default: basic             # Default mode: none, basic, basic_plus, semi_auto, full_auto
    allow_switch: true         # Allow mode switching
    persist: true              # Persist mode across sessions
    session_scoped: false      # Mode only for current session

  # Escalation settings
  escalation:
    allow: true
    auto_de_escalate: true     # Revert after task completion
    timeout: 1h                # Maximum escalation duration
    require_reason: true

  # Safety settings
  safety:
    enable_guardrails: true
    risk_threshold: medium     # none, low, medium, high, critical
    require_reason_risky: true

  # Confirmation settings
  confirmation:
    confirm_risky: true
    confirm_bulk: true
    bulk_threshold: 5          # Files count for bulk operations

  # Auto-debug settings
  auto_debug:
    enabled: true
    max_retries: 3
    retry_delay: 2s
    learn_from_errors: true

  # Mode-specific overrides
  modes:
    none:
      auto_context: false
      auto_apply: false
      auto_execute: false
      auto_debug: false

    basic:
      auto_context: true
      auto_apply: false
      auto_execute: false
      auto_debug: false

    basic_plus:
      auto_context: true
      auto_apply: true
      auto_execute: false
      auto_debug: false
      safe_operations_only: true

    semi_auto:
      auto_context: true
      auto_apply: true
      auto_execute: true
      auto_debug: true
      max_retries: 3
      confirm_risky: true

    full_auto:
      auto_context: true
      auto_apply: true
      auto_execute: true
      auto_debug: true
      max_retries: 5
      confirm_risky: false
      allow_risky: true

  # Guardrail rules
  guardrails:
    - name: no_system_file_delete
      enabled: true
      severity: critical

    - name: no_bulk_unreviewed
      enabled: true
      severity: high
      threshold: 10

    - name: no_destructive_commands
      enabled: true
      severity: critical
      commands:
        - "rm -rf"
        - "dd"
        - "mkfs"

    - name: require_tests_before_deploy
      enabled: false
      severity: high
```

---

## Error Handling

### Error Types

```go
package autonomy

import "errors"

var (
    // Mode errors
    ErrInvalidMode        = errors.New("invalid autonomy mode")
    ErrModeSwitchDenied   = errors.New("mode switch not allowed")
    ErrModeNotPersisted   = errors.New("failed to persist mode")

    // Permission errors
    ErrPermissionDenied   = errors.New("permission denied")
    ErrConfirmationFailed = errors.New("user confirmation failed")
    ErrGuardrailViolation = errors.New("guardrail violation")

    // Execution errors
    ErrActionFailed       = errors.New("action execution failed")
    ErrRetryExhausted     = errors.New("retry attempts exhausted")
    ErrUnsafeOperation    = errors.New("operation deemed unsafe")

    // Escalation errors
    ErrEscalationDenied   = errors.New("escalation request denied")
    ErrEscalationExpired  = errors.New("escalation has expired")
    ErrAlreadyEscalated   = errors.New("already at requested level")
)

// AutonomyError provides detailed error information
type AutonomyError struct {
    Op       string       // Operation that failed
    Mode     AutonomyMode // Current mode
    Action   *Action      // Related action
    Err      error        // Underlying error
    Reason   string       // Human-readable reason
    Fixable  bool         // Whether error can be fixed
}

func (e *AutonomyError) Error() string {
    return fmt.Sprintf("%s (mode: %s): %v - %s",
        e.Op, e.Mode, e.Err, e.Reason)
}

func (e *AutonomyError) Unwrap() error {
    return e.Err
}
```

---

## Testing Strategy

### Unit Tests

```go
package autonomy_test

import (
    "context"
    "testing"

    "github.com/yourusername/helix/internal/autonomy"
)

// TestModeCapabilities tests mode capability definitions
func TestModeCapabilities(t *testing.T) {
    tests := []struct {
        mode       autonomy.AutonomyMode
        wantContext bool
        wantApply  bool
        wantExecute bool
        wantDebug  bool
    }{
        {autonomy.ModeNone, false, false, false, false},
        {autonomy.ModeBasic, true, false, false, false},
        {autonomy.ModeBasicPlus, true, true, false, false},
        {autonomy.ModeSemiAuto, true, true, true, true},
        {autonomy.ModeFullAuto, true, true, true, true},
    }

    for _, tt := range tests {
        t.Run(string(tt.mode), func(t *testing.T) {
            caps := autonomy.GetCapabilities(tt.mode)

            if caps.AutoContext != tt.wantContext {
                t.Errorf("AutoContext = %v, want %v", caps.AutoContext, tt.wantContext)
            }
            if caps.AutoApply != tt.wantApply {
                t.Errorf("AutoApply = %v, want %v", caps.AutoApply, tt.wantApply)
            }
            if caps.AutoExecute != tt.wantExecute {
                t.Errorf("AutoExecute = %v, want %v", caps.AutoExecute, tt.wantExecute)
            }
            if caps.AutoDebug != tt.wantDebug {
                t.Errorf("AutoDebug = %v, want %v", caps.AutoDebug, tt.wantDebug)
            }
        })
    }
}

// TestPermissionChecking tests permission logic
func TestPermissionChecking(t *testing.T) {
    guardrails := autonomy.NewGuardrailsChecker()
    permManager := autonomy.NewPermissionManager(autonomy.ModeBasic, guardrails)

    ctx := context.Background()

    tests := []struct {
        name        string
        action      *autonomy.Action
        wantGranted bool
        wantConfirm bool
    }{
        {
            name: "load context in basic mode",
            action: &autonomy.Action{
                Type: autonomy.ActionLoadContext,
                Risk: autonomy.RiskNone,
            },
            wantGranted: true,
            wantConfirm: false,
        },
        {
            name: "apply change in basic mode",
            action: &autonomy.Action{
                Type: autonomy.ActionApplyChange,
                Risk: autonomy.RiskLow,
            },
            wantGranted: true,
            wantConfirm: true, // Basic mode requires confirmation
        },
        {
            name: "execute command in basic mode",
            action: &autonomy.Action{
                Type: autonomy.ActionExecuteCmd,
                Risk: autonomy.RiskMedium,
            },
            wantGranted: false, // Basic mode can't auto-execute
            wantConfirm: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            perm, err := permManager.Check(ctx, tt.action)
            if err != nil {
                t.Fatalf("Check() error = %v", err)
            }

            if perm.Granted != tt.wantGranted {
                t.Errorf("Granted = %v, want %v", perm.Granted, tt.wantGranted)
            }

            if perm.RequiresConfirm != tt.wantConfirm {
                t.Errorf("RequiresConfirm = %v, want %v",
                    perm.RequiresConfirm, tt.wantConfirm)
            }
        })
    }
}

// TestModeTransitions tests mode switching
func TestModeTransitions(t *testing.T) {
    config := &autonomy.ModeConfig{
        AllowDowngrade: true,
    }

    manager, err := autonomy.NewModeManager(config)
    if err != nil {
        t.Fatalf("NewModeManager() error = %v", err)
    }

    ctx := context.Background()

    // Test upgrade
    if err := manager.SetMode(ctx, autonomy.ModeBasicPlus, "test upgrade"); err != nil {
        t.Fatalf("SetMode() error = %v", err)
    }

    if manager.GetMode() != autonomy.ModeBasicPlus {
        t.Errorf("GetMode() = %v, want %v", manager.GetMode(), autonomy.ModeBasicPlus)
    }

    // Test temporary escalation
    if err := manager.TemporaryMode(ctx, autonomy.ModeSemiAuto, 5*time.Minute); err != nil {
        t.Fatalf("TemporaryMode() error = %v", err)
    }

    if manager.GetMode() != autonomy.ModeSemiAuto {
        t.Errorf("GetMode() = %v, want %v", manager.GetMode(), autonomy.ModeSemiAuto)
    }

    // Test revert
    if err := manager.RevertMode(ctx); err != nil {
        t.Fatalf("RevertMode() error = %v", err)
    }

    if manager.GetMode() != autonomy.ModeBasicPlus {
        t.Errorf("GetMode() = %v, want %v after revert",
            manager.GetMode(), autonomy.ModeBasicPlus)
    }
}

// TestGuardrails tests safety guardrails
func TestGuardrails(t *testing.T) {
    checker := autonomy.NewGuardrailsChecker()
    ctx := context.Background()

    tests := []struct {
        name       string
        action     *autonomy.Action
        wantPass   bool
        wantReason string
    }{
        {
            name: "safe file edit",
            action: &autonomy.Action{
                Type: autonomy.ActionApplyChange,
                Context: &autonomy.ActionContext{
                    FilesAffected: []string{"src/main.go"},
                },
            },
            wantPass: true,
        },
        {
            name: "bulk unreviewed edit",
            action: &autonomy.Action{
                Type: autonomy.ActionBulkEdit,
                Context: &autonomy.ActionContext{
                    FilesAffected: make([]string, 15), // Over threshold
                },
            },
            wantPass: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            passed, reasons, err := checker.Check(ctx, tt.action)
            if err != nil {
                t.Fatalf("Check() error = %v", err)
            }

            if passed != tt.wantPass {
                t.Errorf("Check() = %v, want %v", passed, tt.wantPass)
                if len(reasons) > 0 {
                    t.Logf("Reasons: %v", reasons)
                }
            }
        })
    }
}
```

### Integration Tests

```go
package autonomy_test

// TestAutonomyWorkflow tests complete autonomy workflow
func TestAutonomyWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    config := &autonomy.Config{
        DefaultMode:      autonomy.ModeBasic,
        AllowEscalation:  true,
        EnableGuardrails: true,
    }

    controller, err := autonomy.NewAutonomyController(config)
    if err != nil {
        t.Fatalf("NewAutonomyController() error = %v", err)
    }

    // Test auto-context loading (allowed in basic mode)
    action := &autonomy.Action{
        Type:        autonomy.ActionLoadContext,
        Description: "Load project context",
        Risk:        autonomy.RiskNone,
    }

    result, err := controller.ExecuteAction(ctx, action)
    if err != nil {
        t.Fatalf("ExecuteAction() error = %v", err)
    }

    if !result.Success {
        t.Error("expected successful context loading")
    }

    // Test auto-apply (requires confirmation in basic mode)
    changeAction := &autonomy.Action{
        Type:        autonomy.ActionApplyChange,
        Description: "Apply code change",
        Risk:        autonomy.RiskLow,
    }

    perm, _ := controller.RequestPermission(ctx, changeAction)
    if !perm.RequiresConfirm {
        t.Error("expected confirmation requirement")
    }

    // Request escalation to semi-auto
    err = controller.RequestEscalation(ctx, "need to execute tests", 10*time.Minute)
    if err != nil {
        t.Fatalf("RequestEscalation() error = %v", err)
    }

    if controller.GetCurrentMode() != autonomy.ModeSemiAuto {
        t.Errorf("expected mode %v, got %v",
            autonomy.ModeSemiAuto, controller.GetCurrentMode())
    }

    // Now test auto-execution (allowed in semi-auto)
    cmdAction := &autonomy.Action{
        Type:        autonomy.ActionExecuteCmd,
        Description: "Run tests",
        Risk:        autonomy.RiskLow,
        Context: &autonomy.ActionContext{
            CommandToRun: "go test ./...",
        },
    }

    result, err = controller.ExecuteAction(ctx, cmdAction)
    if err != nil {
        t.Fatalf("ExecuteAction() error = %v", err)
    }

    if !result.Success {
        t.Error("expected successful command execution")
    }

    // Test auto-debug on failure
    failingAction := &autonomy.Action{
        Type: autonomy.ActionExecuteCmd,
        Context: &autonomy.ActionContext{
            CommandToRun: "exit 1", // Will fail
        },
    }

    // Should retry automatically in semi-auto mode
    result, _ = controller.ExecuteAction(ctx, failingAction)
    if result.Retries == 0 {
        t.Error("expected retry attempts")
    }
}
```

---

## Performance Considerations

### Optimization Guidelines

1. **Permission Caching**
   - Cache permission decisions for identical actions
   - Invalidate on mode change
   - Time-bound cache entries

2. **Guardrail Evaluation**
   - Lazy evaluation of rules
   - Short-circuit on first failure
   - Parallel rule checking for independent rules

3. **Mode Persistence**
   - Async persistence
   - Batch mode history writes
   - In-memory caching

### Performance Metrics

```go
// Metrics tracks autonomy system performance
type Metrics struct {
    PermissionChecks    atomic.Int64
    PermissionsGranted  atomic.Int64
    PermissionsDenied   atomic.Int64
    ActionsExecuted     atomic.Int64
    ActionsFailed       atomic.Int64
    AutoRetries         atomic.Int64
    ModeChanges         atomic.Int64

    AverageCheckTime    atomic.Int64 // microseconds
    AverageExecuteTime  atomic.Int64 // milliseconds
}

func (m *Metrics) RecordPermissionCheck(duration time.Duration, granted bool) {
    m.PermissionChecks.Add(1)
    if granted {
        m.PermissionsGranted.Add(1)
    } else {
        m.PermissionsDenied.Add(1)
    }

    current := m.AverageCheckTime.Load()
    newAvg := (current*9 + duration.Microseconds()) / 10
    m.AverageCheckTime.Store(newAvg)
}
```

---

## User Experience Flow

### CLI Interface

```bash
# Check current mode
$ helix mode
Current autonomy mode: basic

Capabilities:
  ✓ Auto-load context
  ✗ Auto-apply changes (requires confirmation)
  ✗ Auto-execute commands (requires confirmation)
  ✗ Auto-debug/retry

# Set mode
$ helix mode set semi_auto
Autonomy mode changed: basic → semi_auto

New capabilities:
  ✓ Auto-load context
  ✓ Auto-apply changes
  ✓ Auto-execute safe commands
  ✓ Auto-debug (up to 3 retries)

# Request temporary escalation
$ helix mode escalate full_auto --duration 30m --reason "debugging critical issue"
Escalation requested: semi_auto → full_auto
Duration: 30 minutes
Reason: debugging critical issue

⚠️  Full auto mode has no safety confirmations. Proceed? (y/N): y

Mode escalated. Will revert automatically in 30 minutes.

# Show mode history
$ helix mode history
Mode Changes:
  2025-01-05 14:30  basic → semi_auto        (manual)
  2025-01-05 15:00  semi_auto → full_auto    (escalation: debugging)
  2025-01-05 15:30  full_auto → semi_auto    (auto-revert)
```

### TUI Interface

```
┌──────────────────────────────────────────────────────────────┐
│ Autonomy Mode: SEMI AUTO                        [Change]    │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│ Current Capabilities:                                        │
│   ✓ Auto-load context                                       │
│   ✓ Auto-apply changes                                      │
│   ✓ Auto-execute commands                                   │
│   ✓ Auto-debug (max 3 retries)                             │
│                                                              │
│ Pending Actions:                                             │
│   • Execute command: npm test                               │
│     Risk: Low | Auto-approved                               │
│                                                              │
│   • Apply changes to 3 files                                │
│     Risk: Medium | Requires confirmation                    │
│     [Approve] [Deny] [Details]                              │
│                                                              │
│ Recent Activity:                                             │
│   15:30  Loaded project context  (auto)                     │
│   15:32  Applied auth.go changes (confirmed)                │
│   15:35  Executed tests (auto)                              │
│   15:35  Retry #1 after test failure (auto)                 │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ [E]scalate  [H]istory  [G]uardrails  [Q]uit                │
└──────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 1: Core Infrastructure (Week 1)
- [ ] Mode definitions and capabilities
- [ ] ModeManager implementation
- [ ] Basic permission system
- [ ] Unit tests

### Phase 2: Permission System (Week 2)
- [ ] PermissionManager
- [ ] GuardrailsChecker
- [ ] Default guardrail rules
- [ ] Integration tests

### Phase 3: Action Execution (Week 3)
- [ ] ActionExecutor
- [ ] Auto-debug/retry logic
- [ ] Error handling
- [ ] Execution tests

### Phase 4: Escalation System (Week 4)
- [ ] EscalationEngine
- [ ] Temporary mode elevation
- [ ] Auto-revert logic
- [ ] Escalation tests

### Phase 5: Integration (Week 5)
- [ ] CLI commands
- [ ] TUI interface
- [ ] Configuration management
- [ ] End-to-end testing

---

## Security Considerations

1. **Privilege Management**
   - Validate mode transitions
   - Audit escalations
   - Time-limited elevated permissions
   - User authentication for sensitive modes

2. **Action Validation**
   - Sanitize command inputs
   - Validate file paths
   - Check operation safety
   - Rate limiting for actions

3. **Audit Trail**
   - Log all mode changes
   - Record permission decisions
   - Track action executions
   - Exportable audit logs

---

## References

- **Plandex autonomy modes**: Go implementation of tiered autonomy
- **Human-in-the-loop ML**: Patterns for user oversight
- **Agentic frameworks**: AutoGPT, BabyAGI autonomy patterns
- **Safety research**: AI safety and alignment principles
