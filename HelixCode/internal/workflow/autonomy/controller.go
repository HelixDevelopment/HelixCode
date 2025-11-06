package autonomy

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AutonomyController manages autonomy modes and permissions
type AutonomyController struct {
	mu          sync.RWMutex
	modeManager *ModeManager
	permManager *PermissionManager
	executor    *ActionExecutor
	escalator   *EscalationEngine
	guardrails  *GuardrailsChecker
	config      *Config
	metrics     *Metrics
}

// NewAutonomyController creates a new autonomy controller
func NewAutonomyController(config *Config) (*AutonomyController, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create mode manager
	modeConfig := &ModeConfig{
		PersistPath:    config.PersistPath,
		AllowDowngrade: true,
		RequireReason:  config.RequireReason,
		AuditChanges:   true,
	}
	modeManager, err := NewModeManager(modeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create mode manager: %w", err)
	}

	// Set default mode
	if config.DefaultMode.IsValid() {
		_ = modeManager.SetMode(context.Background(), config.DefaultMode, "initialization")
	}

	// Create guardrails checker
	var guardrails *GuardrailsChecker
	if config.EnableGuardrails {
		guardrails = NewGuardrailsChecker()
	}

	// Create permission manager
	currentMode := modeManager.GetMode()
	permManager := NewPermissionManager(currentMode, guardrails)

	// Create action executor
	executor := NewActionExecutor(permManager)
	if config.DebugEnabled {
		executor.SetRetryConfig(config.MaxRetries, config.RetryDelay)
	}

	// Create escalation engine
	escalationConfig := &EscalationConfig{
		AllowEscalation: config.AllowEscalation,
		MaxDuration:     config.EscalationTimeout,
		RequireReason:   config.RequireReason,
		AutoRevert:      config.AutoDeEscalate,
		NotifyOnRevert:  true,
	}
	escalator := NewEscalationEngine(modeManager, escalationConfig)

	// Create metrics
	metrics := NewMetrics()

	return &AutonomyController{
		modeManager: modeManager,
		permManager: permManager,
		executor:    executor,
		escalator:   escalator,
		guardrails:  guardrails,
		config:      config,
		metrics:     metrics,
	}, nil
}

// GetCurrentMode returns the active autonomy mode
func (a *AutonomyController) GetCurrentMode() AutonomyMode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.modeManager.GetMode()
}

// GetCapabilities returns the capabilities for the current mode
func (a *AutonomyController) GetCapabilities() *ModeCapabilities {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.modeManager.GetCapabilities()
}

// SetMode changes the autonomy mode
func (a *AutonomyController) SetMode(ctx context.Context, mode AutonomyMode) error {
	return a.SetModeWithReason(ctx, mode, "manual mode change")
}

// SetModeWithReason changes the autonomy mode with a reason
func (a *AutonomyController) SetModeWithReason(ctx context.Context, mode AutonomyMode, reason string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.config.AllowModeSwitch {
		return ErrModeSwitchDenied
	}

	oldMode := a.modeManager.GetMode()

	if err := a.modeManager.SetMode(ctx, mode, reason); err != nil {
		return err
	}

	// Update permission manager capabilities
	newCaps := GetCapabilities(mode)
	a.permManager.UpdateCapabilities(newCaps)

	// Record metric
	a.metrics.RecordModeChange()

	fmt.Printf("Mode changed: %s -> %s (%s)\n", oldMode, mode, reason)

	return nil
}

// RequestPermission checks if an action is permitted
func (a *AutonomyController) RequestPermission(ctx context.Context, action *Action) (*Permission, error) {
	start := time.Now()

	perm, err := a.permManager.Check(ctx, action)

	// Record metrics
	duration := time.Since(start)
	a.metrics.RecordPermissionCheck(duration, perm != nil && perm.Granted)

	return perm, err
}

// ExecuteAction executes an action with appropriate permissions
func (a *AutonomyController) ExecuteAction(ctx context.Context, action *Action) (*ActionResult, error) {
	a.mu.RLock()
	caps := a.modeManager.GetCapabilities()
	a.mu.RUnlock()

	// Check if auto-debug/retry is enabled
	if caps.AutoDebug && caps.MaxRetries > 0 {
		return a.executor.ExecuteWithRetry(ctx, action, caps.MaxRetries)
	}

	return a.executor.Execute(ctx, action)
}

// RequestEscalation requests temporary mode escalation
func (a *AutonomyController) RequestEscalation(ctx context.Context, reason string, duration time.Duration) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	currentMode := a.modeManager.GetMode()

	// Determine target mode (one level up)
	var targetMode AutonomyMode
	switch currentMode {
	case ModeNone:
		targetMode = ModeBasic
	case ModeBasic:
		targetMode = ModeBasicPlus
	case ModeBasicPlus:
		targetMode = ModeSemiAuto
	case ModeSemiAuto:
		targetMode = ModeFullAuto
	case ModeFullAuto:
		return ErrAlreadyEscalated
	default:
		return ErrInvalidMode
	}

	escalation, err := a.escalator.Request(ctx, targetMode, reason, duration)
	if err != nil {
		return err
	}

	// Update permission manager capabilities
	newCaps := GetCapabilities(targetMode)
	a.permManager.UpdateCapabilities(newCaps)

	// Record metric
	a.metrics.RecordEscalation()

	fmt.Printf("Escalation approved: %s\n", escalation.String())

	return nil
}

// RequestEscalationTo requests escalation to a specific mode
func (a *AutonomyController) RequestEscalationTo(ctx context.Context, targetMode AutonomyMode, reason string, duration time.Duration) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	escalation, err := a.escalator.Request(ctx, targetMode, reason, duration)
	if err != nil {
		return err
	}

	// Update permission manager capabilities
	newCaps := GetCapabilities(targetMode)
	a.permManager.UpdateCapabilities(newCaps)

	// Record metric
	a.metrics.RecordEscalation()

	fmt.Printf("Escalation approved: %s\n", escalation.String())

	return nil
}

// DeEscalate returns to previous mode
func (a *AutonomyController) DeEscalate(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.modeManager.RevertMode(ctx); err != nil {
		return err
	}

	// Update permission manager capabilities
	currentMode := a.modeManager.GetMode()
	newCaps := GetCapabilities(currentMode)
	a.permManager.UpdateCapabilities(newCaps)

	fmt.Printf("De-escalated to mode: %s\n", currentMode)

	return nil
}

// LoadContext loads relevant context for a task
func (a *AutonomyController) LoadContext(ctx context.Context, task string) error {
	caps := a.GetCapabilities()

	if !caps.AutoContext {
		return fmt.Errorf("auto-context not enabled in mode: %s", a.GetCurrentMode())
	}

	return a.executor.LoadContext(ctx, task)
}

// ApplyChange applies a code change
func (a *AutonomyController) ApplyChange(ctx context.Context, change *CodeChange) error {
	caps := a.GetCapabilities()

	if !caps.AutoApply && caps.RequireConfirm {
		// Would need user confirmation
		// For now, we allow it but mark it as requiring confirmation
	}

	return a.executor.ApplyChange(ctx, change)
}

// ExecuteCommand executes a command
func (a *AutonomyController) ExecuteCommand(ctx context.Context, cmd string) (*ActionResult, error) {
	caps := a.GetCapabilities()

	if !caps.AutoExecute {
		return nil, fmt.Errorf("auto-execute not enabled in mode: %s", a.GetCurrentMode())
	}

	return a.executor.ExecuteCommand(ctx, cmd)
}

// GetMetrics returns system metrics
func (a *AutonomyController) GetMetrics() *Metrics {
	return a.metrics
}

// GetModeHistory returns mode change history
func (a *AutonomyController) GetModeHistory() *ModeHistory {
	return a.modeManager.GetHistory()
}

// GetActiveEscalations returns active escalations
func (a *AutonomyController) GetActiveEscalations() []*Escalation {
	return a.escalator.GetActive()
}

// GetGuardrailViolations returns recent guardrail violations
func (a *AutonomyController) GetGuardrailViolations() []Violation {
	if a.guardrails == nil {
		return nil
	}
	return a.guardrails.GetViolations()
}

// AddGuardrailRule adds a custom guardrail rule
func (a *AutonomyController) AddGuardrailRule(rule GuardrailRule) {
	if a.guardrails != nil {
		a.guardrails.AddRule(rule)
	}
}

// DisableGuardrailRule disables a specific guardrail rule
func (a *AutonomyController) DisableGuardrailRule(name string) {
	if a.guardrails != nil {
		a.guardrails.DisableRule(name)
	}
}

// EnableGuardrailRule enables a specific guardrail rule
func (a *AutonomyController) EnableGuardrailRule(name string) {
	if a.guardrails != nil {
		a.guardrails.EnableRule(name)
	}
}

// GetConfig returns the controller configuration
func (a *AutonomyController) GetConfig() *Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Clone()
}

// UpdateConfig updates the controller configuration
func (a *AutonomyController) UpdateConfig(config *Config) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.config = config.Clone()

	// Update retry config
	if config.DebugEnabled {
		a.executor.SetRetryConfig(config.MaxRetries, config.RetryDelay)
	}

	return nil
}

// Shutdown gracefully shuts down the controller
func (a *AutonomyController) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Save current mode
	if err := a.modeManager.SaveMode(ctx); err != nil {
		return fmt.Errorf("failed to save mode: %w", err)
	}

	// Check and revert any active escalations
	if err := a.escalator.CheckExpired(ctx); err != nil {
		return fmt.Errorf("failed to check escalations: %w", err)
	}

	return nil
}
