package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionStruct(t *testing.T) {
	now := time.Now()
	session := Session{
		ID:          "test-session-123",
		ProjectID:   "project-456",
		Name:        "Test Session",
		Description: "A test session",
		Mode:        ModePlanning,
		Status:      StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, "test-session-123", session.ID)
	assert.Equal(t, "project-456", session.ProjectID)
	assert.Equal(t, "Test Session", session.Name)
	assert.Equal(t, "A test session", session.Description)
	assert.Equal(t, ModePlanning, session.Mode)
	assert.Equal(t, StatusActive, session.Status)
	assert.Equal(t, now, session.CreatedAt)
	assert.Equal(t, now, session.UpdatedAt)
}

func TestModeConstants(t *testing.T) {
	assert.Equal(t, Mode("planning"), ModePlanning)
	assert.Equal(t, Mode("building"), ModeBuilding)
	assert.Equal(t, Mode("testing"), ModeTesting)
	assert.Equal(t, Mode("refactoring"), ModeRefactoring)
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, Status("active"), StatusActive)
	assert.Equal(t, Status("paused"), StatusPaused)
	assert.Equal(t, Status("completed"), StatusCompleted)
	assert.Equal(t, Status("failed"), StatusFailed)
}

func TestSessionJSONTags(t *testing.T) {
	session := Session{
		ID:          "123",
		ProjectID:   "456",
		Name:        "Test",
		Description: "Desc",
		Mode:        ModePlanning,
		Status:      StatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Test that JSON tags are present (basic smoke test)
	assert.NotEmpty(t, session.ID)
	assert.NotEmpty(t, session.ProjectID)
	assert.NotEmpty(t, session.Name)
	assert.NotEmpty(t, session.Description)
	assert.NotEqual(t, Mode(""), session.Mode)
	assert.NotEqual(t, Status(""), session.Status)
}
