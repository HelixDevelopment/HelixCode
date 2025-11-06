package planmode

import (
	"fmt"
	"sync"
)

// Mode represents the operational mode
type Mode int

const (
	ModeNormal Mode = iota // Normal operation
	ModePlan               // Planning phase
	ModeAct                // Execution phase
	ModePaused             // Paused execution
)

// String returns string representation of mode
func (m Mode) String() string {
	return [...]string{"Normal", "Plan", "Act", "Paused"}[m]
}

// ModeChangeCallback is called when mode changes
type ModeChangeCallback func(from, to Mode, state *ModeState)

// ModeState contains state information for a mode
type ModeState struct {
	Mode        Mode
	PlanID      string
	OptionID    string
	ExecutionID string
	Metadata    map[string]interface{}
}

// ModeController manages operational modes
type ModeController interface {
	// GetMode returns the current mode
	GetMode() Mode

	// SetMode sets the current mode
	SetMode(mode Mode) error

	// CanTransition checks if a mode transition is allowed
	CanTransition(from, to Mode) bool

	// TransitionTo transitions to a new mode
	TransitionTo(mode Mode) error

	// RegisterCallback registers a callback for mode changes
	RegisterCallback(fn ModeChangeCallback)

	// GetState returns current mode state
	GetState() *ModeState

	// UpdateState updates mode state
	UpdateState(state *ModeState) error
}

// DefaultModeController implements ModeController
type DefaultModeController struct {
	currentMode Mode
	state       *ModeState
	callbacks   []ModeChangeCallback
	mu          sync.RWMutex
}

// NewModeController creates a new mode controller
func NewModeController() ModeController {
	return &DefaultModeController{
		currentMode: ModeNormal,
		state: &ModeState{
			Mode:     ModeNormal,
			Metadata: make(map[string]interface{}),
		},
		callbacks: make([]ModeChangeCallback, 0),
	}
}

// GetMode returns the current mode
func (mc *DefaultModeController) GetMode() Mode {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.currentMode
}

// SetMode sets the current mode
func (mc *DefaultModeController) SetMode(mode Mode) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.canTransitionUnlocked(mc.currentMode, mode) {
		return fmt.Errorf("invalid mode transition from %s to %s", mc.currentMode, mode)
	}

	oldMode := mc.currentMode
	mc.currentMode = mode
	mc.state.Mode = mode

	// Notify callbacks
	for _, cb := range mc.callbacks {
		cb(oldMode, mode, mc.state)
	}

	return nil
}

// CanTransition checks if a mode transition is allowed
func (mc *DefaultModeController) CanTransition(from, to Mode) bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.canTransitionUnlocked(from, to)
}

// canTransitionUnlocked checks transition without locking (internal use)
func (mc *DefaultModeController) canTransitionUnlocked(from, to Mode) bool {
	// Define valid transitions
	validTransitions := map[Mode][]Mode{
		ModeNormal: {ModePlan},
		ModePlan:   {ModeAct, ModeNormal},
		ModeAct:    {ModePaused, ModeNormal},
		ModePaused: {ModeAct, ModeNormal},
	}

	allowedModes, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedModes {
		if allowed == to {
			return true
		}
	}

	return false
}

// TransitionTo transitions to a new mode
func (mc *DefaultModeController) TransitionTo(mode Mode) error {
	return mc.SetMode(mode)
}

// RegisterCallback registers a callback for mode changes
func (mc *DefaultModeController) RegisterCallback(fn ModeChangeCallback) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.callbacks = append(mc.callbacks, fn)
}

// GetState returns current mode state
func (mc *DefaultModeController) GetState() *ModeState {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy to prevent external modifications
	stateCopy := &ModeState{
		Mode:        mc.state.Mode,
		PlanID:      mc.state.PlanID,
		OptionID:    mc.state.OptionID,
		ExecutionID: mc.state.ExecutionID,
		Metadata:    make(map[string]interface{}),
	}

	for k, v := range mc.state.Metadata {
		stateCopy.Metadata[k] = v
	}

	return stateCopy
}

// UpdateState updates mode state
func (mc *DefaultModeController) UpdateState(state *ModeState) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	mc.state = &ModeState{
		Mode:        state.Mode,
		PlanID:      state.PlanID,
		OptionID:    state.OptionID,
		ExecutionID: state.ExecutionID,
		Metadata:    make(map[string]interface{}),
	}

	for k, v := range state.Metadata {
		mc.state.Metadata[k] = v
	}

	return nil
}
