package autonomy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EscalationEngine handles temporary mode escalation
type EscalationEngine struct {
	mu          sync.RWMutex
	modeManager *ModeManager
	escalations map[string]*Escalation
	config      *EscalationConfig
}

// Escalation represents a temporary mode increase
type Escalation struct {
	ID        string
	From      AutonomyMode
	To        AutonomyMode
	Reason    string
	StartTime time.Time
	Duration  time.Duration
	ExpiresAt time.Time
	Active    bool
	UserID    string
}

// NewEscalationEngine creates an escalation engine
func NewEscalationEngine(modeManager *ModeManager, config *EscalationConfig) *EscalationEngine {
	if config == nil {
		config = NewDefaultEscalationConfig()
	}

	return &EscalationEngine{
		modeManager: modeManager,
		escalations: make(map[string]*Escalation),
		config:      config,
	}
}

// Request requests a temporary escalation
func (e *EscalationEngine) Request(ctx context.Context, targetMode AutonomyMode, reason string, duration time.Duration) (*Escalation, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.config.AllowEscalation {
		return nil, ErrEscalationDenied
	}

	if !targetMode.IsValid() {
		return nil, ErrInvalidMode
	}

	currentMode := e.modeManager.GetMode()

	// Check if target mode is higher than current
	if targetMode.Level() <= currentMode.Level() {
		return nil, fmt.Errorf("%w: target mode %s is not higher than current %s",
			ErrAlreadyEscalated, targetMode, currentMode)
	}

	// Check if reason is required
	if e.config.RequireReason && reason == "" {
		return nil, fmt.Errorf("reason required for escalation")
	}

	// Check duration limits
	if duration > e.config.MaxDuration {
		duration = e.config.MaxDuration
	}
	if duration <= 0 {
		duration = 1 * time.Hour
	}

	// Create escalation
	escalation := &Escalation{
		ID:        uuid.New().String(),
		From:      currentMode,
		To:        targetMode,
		Reason:    reason,
		StartTime: time.Now(),
		Duration:  duration,
		ExpiresAt: time.Now().Add(duration),
		Active:    false,
	}

	// Store pending escalation
	e.escalations[escalation.ID] = escalation

	// Auto-approve for now (in production, this would require approval)
	// Perform mode escalation
	if err := e.modeManager.TemporaryMode(ctx, escalation.To, escalation.Duration); err != nil {
		delete(e.escalations, escalation.ID)
		return nil, fmt.Errorf("failed to escalate mode: %w", err)
	}

	escalation.Active = true
	escalation.StartTime = time.Now()
	escalation.ExpiresAt = time.Now().Add(escalation.Duration)

	// Schedule auto-revert if configured
	if e.config.AutoRevert {
		go e.scheduleRevert(ctx, escalation.ID, escalation.Duration)
	}

	return escalation, nil
}

// Approve approves an escalation request
func (e *EscalationEngine) Approve(ctx context.Context, escalationID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	escalation, exists := e.escalations[escalationID]
	if !exists {
		return fmt.Errorf("escalation not found: %s", escalationID)
	}

	if escalation.Active {
		return fmt.Errorf("escalation already active")
	}

	// Perform mode escalation
	if err := e.modeManager.TemporaryMode(ctx, escalation.To, escalation.Duration); err != nil {
		return fmt.Errorf("failed to escalate mode: %w", err)
	}

	escalation.Active = true
	escalation.StartTime = time.Now()
	escalation.ExpiresAt = time.Now().Add(escalation.Duration)

	// Schedule auto-revert if configured
	if e.config.AutoRevert {
		go e.scheduleRevert(ctx, escalationID, escalation.Duration)
	}

	return nil
}

// Deny denies an escalation request
func (e *EscalationEngine) Deny(ctx context.Context, escalationID string, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	escalation, exists := e.escalations[escalationID]
	if !exists {
		return fmt.Errorf("escalation not found: %s", escalationID)
	}

	if escalation.Active {
		return fmt.Errorf("cannot deny active escalation")
	}

	// Remove the escalation
	delete(e.escalations, escalationID)

	return nil
}

// Revert manually reverts an escalation
func (e *EscalationEngine) Revert(ctx context.Context, escalationID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	escalation, exists := e.escalations[escalationID]
	if !exists {
		return fmt.Errorf("escalation not found: %s", escalationID)
	}

	if !escalation.Active {
		return fmt.Errorf("escalation not active")
	}

	// Revert mode
	if err := e.modeManager.RevertMode(ctx); err != nil {
		return fmt.Errorf("failed to revert mode: %w", err)
	}

	escalation.Active = false

	// Clean up
	delete(e.escalations, escalationID)

	return nil
}

// CheckExpired checks and reverts expired escalations
func (e *EscalationEngine) CheckExpired(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	var expiredIDs []string

	for id, escalation := range e.escalations {
		if escalation.Active && now.After(escalation.ExpiresAt) {
			expiredIDs = append(expiredIDs, id)
		}
	}

	// Revert expired escalations
	for _, id := range expiredIDs {
		escalation := e.escalations[id]
		if err := e.modeManager.RevertMode(ctx); err != nil {
			return fmt.Errorf("failed to revert expired escalation %s: %w", id, err)
		}

		escalation.Active = false
		delete(e.escalations, id)

		if e.config.NotifyOnRevert {
			// In production, this would send a notification
			fmt.Printf("Escalation %s expired and reverted\n", id)
		}
	}

	return nil
}

// GetActive returns active escalations
func (e *EscalationEngine) GetActive() []*Escalation {
	e.mu.RLock()
	defer e.mu.RUnlock()

	active := make([]*Escalation, 0)
	for _, escalation := range e.escalations {
		if escalation.Active {
			active = append(active, escalation)
		}
	}

	return active
}

// GetEscalation returns an escalation by ID
func (e *EscalationEngine) GetEscalation(escalationID string) (*Escalation, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	escalation, exists := e.escalations[escalationID]
	if !exists {
		return nil, fmt.Errorf("escalation not found: %s", escalationID)
	}

	return escalation, nil
}

// GetAll returns all escalations
func (e *EscalationEngine) GetAll() []*Escalation {
	e.mu.RLock()
	defer e.mu.RUnlock()

	all := make([]*Escalation, 0, len(e.escalations))
	for _, escalation := range e.escalations {
		all = append(all, escalation)
	}

	return all
}

// scheduleRevert schedules automatic revert after duration
func (e *EscalationEngine) scheduleRevert(ctx context.Context, escalationID string, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		if err := e.Revert(ctx, escalationID); err != nil {
			// Log error but don't fail
			fmt.Printf("auto-revert failed for escalation %s: %v\n", escalationID, err)
		}
	case <-ctx.Done():
		return
	}
}

// TimeRemaining returns the time remaining for an active escalation
func (esc *Escalation) TimeRemaining() time.Duration {
	if !esc.Active {
		return 0
	}

	remaining := time.Until(esc.ExpiresAt)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// IsExpired returns true if the escalation has expired
func (esc *Escalation) IsExpired() bool {
	return esc.Active && time.Now().After(esc.ExpiresAt)
}

// String returns a human-readable string for the escalation
func (esc *Escalation) String() string {
	status := "pending"
	if esc.Active {
		status = "active"
	}

	return fmt.Sprintf("Escalation[%s]: %s -> %s (%s) - %s",
		status, esc.From, esc.To, esc.Duration, esc.Reason)
}
