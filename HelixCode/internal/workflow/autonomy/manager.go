package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ModeManager handles mode switching and persistence
type ModeManager struct {
	mu             sync.RWMutex
	currentMode    AutonomyMode
	previousMode   AutonomyMode
	sessionMode    AutonomyMode
	persistentMode AutonomyMode
	config         *ModeConfig
	history        *ModeHistory
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
	Temporary bool
}

// persistedMode represents the mode state that is saved to disk
type persistedMode struct {
	Mode      AutonomyMode `json:"mode"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// NewModeManager creates a new mode manager
func NewModeManager(config *ModeConfig) (*ModeManager, error) {
	if config == nil {
		config = NewDefaultModeConfig()
	}

	m := &ModeManager{
		currentMode: GetDefaultMode(),
		config:      config,
		history: &ModeHistory{
			Changes: make([]ModeChange, 0),
		},
	}

	// Load persisted mode if available
	if config.PersistPath != "" {
		if mode, err := m.loadModeFromDisk(); err == nil {
			m.currentMode = mode
			m.persistentMode = mode
		}
	}

	return m, nil
}

// GetMode returns the current mode
func (m *ModeManager) GetMode() AutonomyMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentMode
}

// SetMode changes the active mode
func (m *ModeManager) SetMode(ctx context.Context, mode AutonomyMode, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !mode.IsValid() {
		return ErrInvalidMode
	}

	// Check if transition is allowed
	if err := m.currentMode.CanTransitionTo(mode); err != nil {
		return fmt.Errorf("%w: %v", ErrModeSwitchDenied, err)
	}

	// Check if downgrade is allowed
	if !m.config.AllowDowngrade && mode.Level() < m.currentMode.Level() {
		return ErrModeSwitchDenied
	}

	// Record the change
	change := ModeChange{
		From:      m.currentMode,
		To:        mode,
		Timestamp: time.Now(),
		Reason:    reason,
		Temporary: false,
	}

	m.previousMode = m.currentMode
	m.currentMode = mode

	if m.config.AuditChanges {
		m.history.Changes = append(m.history.Changes, change)
	}

	// Persist if configured
	if m.config.PersistPath != "" {
		if err := m.saveModeToDisk(); err != nil {
			// Log error but don't fail the mode change
			fmt.Fprintf(os.Stderr, "warning: failed to persist mode: %v\n", err)
		}
	}

	return nil
}

// TemporaryMode sets a temporary mode with auto-revert
func (m *ModeManager) TemporaryMode(ctx context.Context, mode AutonomyMode, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !mode.IsValid() {
		return ErrInvalidMode
	}

	// Record the change as temporary
	change := ModeChange{
		From:      m.currentMode,
		To:        mode,
		Timestamp: time.Now(),
		Duration:  duration,
		Temporary: true,
	}

	m.previousMode = m.currentMode
	m.sessionMode = m.currentMode
	m.currentMode = mode

	if m.config.AuditChanges {
		m.history.Changes = append(m.history.Changes, change)
	}

	// Schedule auto-revert
	go func() {
		timer := time.NewTimer(duration)
		defer timer.Stop()

		select {
		case <-timer.C:
			_ = m.RevertMode(ctx)
		case <-ctx.Done():
			return
		}
	}()

	return nil
}

// RevertMode returns to the previous mode
func (m *ModeManager) RevertMode(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.previousMode == "" {
		return fmt.Errorf("no previous mode to revert to")
	}

	// Record the revert
	change := ModeChange{
		From:      m.currentMode,
		To:        m.previousMode,
		Timestamp: time.Now(),
		Reason:    "revert",
		Temporary: false,
	}

	m.currentMode = m.previousMode
	m.previousMode = ""

	if m.config.AuditChanges {
		m.history.Changes = append(m.history.Changes, change)
	}

	return nil
}

// SaveMode persists the current mode
func (m *ModeManager) SaveMode(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config.PersistPath == "" {
		return nil
	}

	return m.saveModeToDisk()
}

// LoadMode loads the persisted mode
func (m *ModeManager) LoadMode(ctx context.Context) (AutonomyMode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config.PersistPath == "" {
		return m.currentMode, nil
	}

	mode, err := m.loadModeFromDisk()
	if err != nil {
		return "", err
	}

	m.currentMode = mode
	m.persistentMode = mode

	return mode, nil
}

// GetHistory returns mode change history
func (m *ModeManager) GetHistory() *ModeHistory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	historyCopy := &ModeHistory{
		Changes: make([]ModeChange, len(m.history.Changes)),
	}
	copy(historyCopy.Changes, m.history.Changes)

	return historyCopy
}

// GetCapabilities returns the capabilities for the current mode
func (m *ModeManager) GetCapabilities() *ModeCapabilities {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return GetCapabilities(m.currentMode)
}

// saveModeToDisk saves the current mode to disk
func (m *ModeManager) saveModeToDisk() error {
	persisted := persistedMode{
		Mode:      m.currentMode,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mode: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.config.PersistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first
	tmpPath := m.config.PersistPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write mode file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, m.config.PersistPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename mode file: %w", err)
	}

	return nil
}

// loadModeFromDisk loads the mode from disk
func (m *ModeManager) loadModeFromDisk() (AutonomyMode, error) {
	data, err := os.ReadFile(m.config.PersistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return GetDefaultMode(), nil
		}
		return "", fmt.Errorf("failed to read mode file: %w", err)
	}

	var persisted persistedMode
	if err := json.Unmarshal(data, &persisted); err != nil {
		return "", fmt.Errorf("failed to unmarshal mode: %w", err)
	}

	if !persisted.Mode.IsValid() {
		return "", ErrInvalidMode
	}

	return persisted.Mode, nil
}

// ClearHistory clears the mode change history
func (m *ModeManager) ClearHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history.Changes = make([]ModeChange, 0)
}
