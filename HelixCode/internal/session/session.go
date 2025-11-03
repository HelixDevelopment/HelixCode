package session

import (
	"time"
)

// Session represents a development session
type Session struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Mode        Mode      `json:"mode"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Mode represents the session mode
type Mode string

const (
	ModePlanning    Mode = "planning"
	ModeBuilding    Mode = "building"
	ModeTesting     Mode = "testing"
	ModeRefactoring Mode = "refactoring"
)

// Status represents the session status
type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)
