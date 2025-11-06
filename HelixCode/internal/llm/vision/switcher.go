package vision

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
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
	mu             sync.RWMutex
}

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

// SwitchReason explains why switch occurred
type SwitchReason string

const (
	ReasonImageDetected   SwitchReason = "image_detected"
	ReasonNoVisionSupport SwitchReason = "no_vision_support"
	ReasonUserRequest     SwitchReason = "user_request"
	ReasonAutoRevert      SwitchReason = "auto_revert"
	ReasonSessionEnd      SwitchReason = "session_end"
)

// SwitchEvent records a model switch
type SwitchEvent struct {
	ID            string
	Timestamp     time.Time
	FromModel     *Model
	ToModel       *Model
	Reason        SwitchReason
	Mode          SwitchMode
	Reverted      bool
	RevertedAt    *time.Time
	UserConfirmed bool
}

// SwitchController manages model switching
type SwitchController struct {
	config  *SwitchConfig
	history *SwitchHistory
	mu      sync.RWMutex
}

// SwitchHistory tracks model switches
type SwitchHistory struct {
	events []SwitchEvent
	mu     sync.RWMutex
}

// NewVisionSwitchManager creates a new vision switch manager
func NewVisionSwitchManager(config *Config, modelRegistry *ModelRegistry) (*VisionSwitchManager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	detectionConfig := &DetectionConfig{
		Methods:          config.DetectionMethods,
		SupportedFormats: []string{"jpg", "jpeg", "png", "gif", "webp", "bmp"},
		MaxFileSize:      10 * 1024 * 1024, // 10MB
		InspectContent:   config.ContentInspection,
	}

	switchConfig := &SwitchConfig{
		Mode:           config.SwitchMode,
		RequireConfirm: config.RequireConfirm,
		AutoRevert:     config.AutoRevert,
		RevertDelay:    config.RevertDelay,
	}

	return &VisionSwitchManager{
		detector:   NewImageDetector(detectionConfig),
		capChecker: NewCapabilityChecker(modelRegistry),
		switchCtrl: NewSwitchController(switchConfig),
		config:     config,
	}, nil
}

// ProcessInput checks input for images and switches if needed
func (v *VisionSwitchManager) ProcessInput(ctx context.Context, input *Input) (*SwitchResult, error) {
	startTime := time.Now()

	// Detect images in input
	detectionResult, err := v.detector.Detect(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("image detection failed: %w", err)
	}

	// Check if switch is needed
	result, err := v.CheckAndSwitch(ctx, detectionResult.HasImages)
	if err != nil {
		return nil, err
	}

	result.ImagesDetected = detectionResult.ImageCount
	result.Duration = time.Since(startTime)

	return result, nil
}

// CheckAndSwitch checks if switch is needed and performs it
func (v *VisionSwitchManager) CheckAndSwitch(ctx context.Context, hasImages bool) (*SwitchResult, error) {
	v.mu.RLock()
	currentModel := v.currentModel
	v.mu.RUnlock()

	result := &SwitchResult{
		SwitchPerformed: false,
		FromModel:       currentModel,
	}

	// No images detected, no switch needed
	if !hasImages {
		return result, nil
	}

	// Check if current model supports vision
	if currentModel != nil && currentModel.Capabilities.SupportsVision {
		result.Reason = "current model already supports vision"
		return result, nil
	}

	// Find a vision-capable model
	preferences := &ModelPreferences{
		PreferredModels: v.config.ModelPriority,
	}

	visionModel, err := v.capChecker.FindBestVisionModel(ctx, preferences)
	if err != nil {
		return nil, fmt.Errorf("no vision-capable model found: %w", err)
	}

	// If confirmation required, set flag
	result.RequiredConfirm = v.config.RequireConfirm
	if v.config.RequireConfirm {
		// In a real implementation, this would prompt the user
		// For now, we'll assume confirmation
		result.UserConfirmed = true
	} else {
		result.UserConfirmed = true
	}

	// Perform switch
	if result.UserConfirmed {
		event, err := v.switchCtrl.Switch(ctx, currentModel, visionModel, ReasonImageDetected)
		if err != nil {
			return nil, fmt.Errorf("switch failed: %w", err)
		}

		v.mu.Lock()
		if v.originalModel == nil {
			v.originalModel = currentModel
		}
		v.currentModel = visionModel
		v.switchActive = true
		v.mu.Unlock()

		result.SwitchPerformed = true
		result.ToModel = visionModel
		result.Reason = fmt.Sprintf("switched from %s to %s for vision support",
			currentModel.Name, visionModel.Name)

		// Handle switch mode
		if v.config.SwitchMode == SwitchOnce {
			// Schedule revert after this request
			go v.scheduleOnceRevert(ctx, event.ID)
		}
	}

	return result, nil
}

// RevertSwitch reverts to the original model
func (v *VisionSwitchManager) RevertSwitch(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.switchActive || v.originalModel == nil {
		return fmt.Errorf("no active switch to revert")
	}

	// Perform revert
	err := v.switchCtrl.Revert(ctx, "")
	if err != nil {
		return fmt.Errorf("revert failed: %w", err)
	}

	v.currentModel = v.originalModel
	v.switchActive = false

	return nil
}

// GetCurrentModel returns the currently active model
func (v *VisionSwitchManager) GetCurrentModel() *Model {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.currentModel
}

// SetCurrentModel sets the current model
func (v *VisionSwitchManager) SetCurrentModel(modelID string) error {
	// In a real implementation, this would look up the model from registry
	// For now, we create a simple model
	model := &Model{
		ID:   modelID,
		Name: modelID,
		Capabilities: &Capabilities{
			SupportsVision: false,
		},
	}

	v.mu.Lock()
	v.currentModel = model
	v.mu.Unlock()

	return nil
}

// IsSwitchActive returns true if a switch is currently active
func (v *VisionSwitchManager) IsSwitchActive() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.switchActive
}

// GetSwitchHistory returns the switch history
func (v *VisionSwitchManager) GetSwitchHistory() []*SwitchEvent {
	return v.switchCtrl.GetHistory()
}

// scheduleOnceRevert schedules a revert for once mode
func (v *VisionSwitchManager) scheduleOnceRevert(ctx context.Context, eventID string) {
	// In once mode, we revert immediately after the request completes
	// This would typically be called after the LLM response is received
	time.Sleep(100 * time.Millisecond) // Small delay to ensure request completes
	v.RevertSwitch(ctx)
}

// NewSwitchController creates a switch controller
func NewSwitchController(config *SwitchConfig) *SwitchController {
	return &SwitchController{
		config: config,
		history: &SwitchHistory{
			events: []SwitchEvent{},
		},
	}
}

// Switch performs a model switch
func (s *SwitchController) Switch(ctx context.Context, from, to *Model, reason SwitchReason) (*SwitchEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := SwitchEvent{
		ID:            uuid.New().String(),
		Timestamp:     time.Now(),
		FromModel:     from,
		ToModel:       to,
		Reason:        reason,
		Mode:          s.config.Mode,
		Reverted:      false,
		UserConfirmed: true,
	}

	s.history.mu.Lock()
	s.history.events = append(s.history.events, event)
	s.history.mu.Unlock()

	return &event, nil
}

// Revert reverts a model switch
func (s *SwitchController) Revert(ctx context.Context, eventID string) error {
	s.history.mu.Lock()
	defer s.history.mu.Unlock()

	// Find the event and mark as reverted
	for i := range s.history.events {
		if eventID == "" || s.history.events[i].ID == eventID {
			now := time.Now()
			s.history.events[i].Reverted = true
			s.history.events[i].RevertedAt = &now
			return nil
		}
	}

	return fmt.Errorf("event not found")
}

// ShouldRevert determines if auto-revert should occur
func (s *SwitchController) ShouldRevert(ctx context.Context) (bool, error) {
	if !s.config.AutoRevert {
		return false, nil
	}

	s.history.mu.RLock()
	defer s.history.mu.RUnlock()

	// Check if there's an active switch that should be reverted
	for _, event := range s.history.events {
		if !event.Reverted && time.Since(event.Timestamp) > s.config.RevertDelay {
			return true, nil
		}
	}

	return false, nil
}

// GetHistory returns switch history
func (s *SwitchController) GetHistory() []*SwitchEvent {
	s.history.mu.RLock()
	defer s.history.mu.RUnlock()

	// Return a copy to prevent external modification
	events := make([]*SwitchEvent, len(s.history.events))
	for i, event := range s.history.events {
		eventCopy := event
		events[i] = &eventCopy
	}

	return events
}

// GetActiveSwitch returns the current active switch (if any)
func (s *SwitchController) GetActiveSwitch() *SwitchEvent {
	s.history.mu.RLock()
	defer s.history.mu.RUnlock()

	// Find the most recent non-reverted switch
	for i := len(s.history.events) - 1; i >= 0; i-- {
		if !s.history.events[i].Reverted {
			event := s.history.events[i]
			return &event
		}
	}

	return nil
}
