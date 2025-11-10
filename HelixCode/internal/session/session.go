package session

import (
	"fmt"
	"time"
)

// Session represents a development session
type Session struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Mode         Mode                   `json:"mode"`
	Status       Status                 `json:"status"`
	FocusChainID string                 `json:"focus_chain_id"`
	Context      map[string]interface{} `json:"context"`
	Metadata     map[string]string      `json:"metadata"`
	Tags         []string               `json:"tags"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	StartedAt    time.Time              `json:"started_at,omitempty"`
	CompletedAt  time.Time              `json:"completed_at,omitempty"`
	EndedAt      time.Time              `json:"ended_at,omitempty"`
	Duration     time.Duration          `json:"duration"`
}

// Mode represents the session mode
type Mode string

const (
	ModePlanning    Mode = "planning"
	ModeBuilding    Mode = "building"
	ModeTesting     Mode = "testing"
	ModeRefactoring Mode = "refactoring"
	ModeDebugging   Mode = "debugging"
	ModeDeployment  Mode = "deployment"
)

// IsValid checks if mode is valid
func (m Mode) IsValid() bool {
	switch m {
	case ModePlanning, ModeBuilding, ModeTesting, ModeRefactoring, ModeDebugging, ModeDeployment:
		return true
	}
	return false
}

// String returns string representation
func (m Mode) String() string {
	return string(m)
}

// Status represents the session status
type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// IsValid checks if status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusPaused, StatusCompleted, StatusFailed:
		return true
	}
	return false
}

// String returns string representation
func (s Status) String() string {
	return string(s)
}

// AddTag adds a tag to the session
func (s *Session) AddTag(tag string) {
	for _, t := range s.Tags {
		if t == tag {
			return // Already exists
		}
	}
	s.Tags = append(s.Tags, tag)
}

// HasTag checks if session has a specific tag
func (s *Session) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// RemoveTag removes a tag from the session
func (s *Session) RemoveTag(tag string) {
	for i, t := range s.Tags {
		if t == tag {
			s.Tags = append(s.Tags[:i], s.Tags[i+1:]...)
			return
		}
	}
}

// SetContext sets a context value
func (s *Session) SetContext(key string, value interface{}) {
	if s.Context == nil {
		s.Context = make(map[string]interface{})
	}
	s.Context[key] = value
}

// GetContext gets a context value
func (s *Session) GetContext(key string) (interface{}, bool) {
	if s.Context == nil {
		return nil, false
	}
	value, ok := s.Context[key]
	return value, ok
}

// SetMetadata sets a metadata value
func (s *Session) SetMetadata(key, value string) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	s.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (s *Session) GetMetadata(key string) (string, bool) {
	if s.Metadata == nil {
		return "", false
	}
	value, ok := s.Metadata[key]
	return value, ok
}

// Clone creates a copy of the session
func (s *Session) Clone() *Session {
	clone := &Session{
		ID:           s.ID,
		ProjectID:    s.ProjectID,
		Name:         s.Name,
		Description:  s.Description,
		Mode:         s.Mode,
		Status:       s.Status,
		FocusChainID: s.FocusChainID,
		Context:      make(map[string]interface{}),
		Metadata:     make(map[string]string),
		Tags:         make([]string, len(s.Tags)),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
		StartedAt:    s.StartedAt,
		CompletedAt:  s.CompletedAt,
		Duration:     s.Duration,
	}

	// Deep copy context
	for k, v := range s.Context {
		clone.Context[k] = v
	}

	// Deep copy metadata
	for k, v := range s.Metadata {
		clone.Metadata[k] = v
	}

	// Copy tags
	copy(clone.Tags, s.Tags)

	return clone
}

// String returns a string representation of the session
func (s *Session) String() string {
	return fmt.Sprintf("Session %s: %s (%s) - %s", s.ID, s.Name, s.Mode, s.Status)
}

// Validate validates the session
func (s *Session) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if s.ProjectID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	if s.Name == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	if !s.Mode.IsValid() {
		return fmt.Errorf("invalid mode: %s", s.Mode)
	}

	if !s.Status.IsValid() {
		return fmt.Errorf("invalid status: %s", s.Status)
	}

	return nil
}
