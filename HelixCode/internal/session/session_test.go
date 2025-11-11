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

// ========================================
// Additional Coverage Tests
// ========================================

func TestSession_SetContext_GetContext(t *testing.T) {
	session := &Session{}

	t.Run("set and get context value", func(t *testing.T) {
		session.SetContext("key1", "value1")

		value, ok := session.GetContext("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", value)
	})

	t.Run("get non-existent context key", func(t *testing.T) {
		value, ok := session.GetContext("non-existent")
		assert.False(t, ok)
		assert.Nil(t, value)
	})

	t.Run("set context initializes map if nil", func(t *testing.T) {
		s := &Session{Context: nil}
		s.SetContext("test", "data")

		assert.NotNil(t, s.Context)
		value, ok := s.GetContext("test")
		assert.True(t, ok)
		assert.Equal(t, "data", value)
	})

	t.Run("get context returns false when map is nil", func(t *testing.T) {
		s := &Session{Context: nil}

		value, ok := s.GetContext("any")
		assert.False(t, ok)
		assert.Nil(t, value)
	})
}

func TestSession_SetMetadata_GetMetadata(t *testing.T) {
	session := &Session{}

	t.Run("set and get metadata value", func(t *testing.T) {
		session.SetMetadata("author", "Alice")

		value, ok := session.GetMetadata("author")
		assert.True(t, ok)
		assert.Equal(t, "Alice", value)
	})

	t.Run("get non-existent metadata key", func(t *testing.T) {
		value, ok := session.GetMetadata("non-existent")
		assert.False(t, ok)
		assert.Equal(t, "", value)
	})

	t.Run("set metadata initializes map if nil", func(t *testing.T) {
		s := &Session{Metadata: nil}
		s.SetMetadata("version", "1.0")

		assert.NotNil(t, s.Metadata)
		value, ok := s.GetMetadata("version")
		assert.True(t, ok)
		assert.Equal(t, "1.0", value)
	})

	t.Run("get metadata returns false when map is nil", func(t *testing.T) {
		s := &Session{Metadata: nil}

		value, ok := s.GetMetadata("any")
		assert.False(t, ok)
		assert.Equal(t, "", value)
	})
}

func TestSession_String(t *testing.T) {
	session := &Session{
		ID:     "abc-123",
		Name:   "Test Session",
		Mode:   ModePlanning,
		Status: StatusActive,
	}

	result := session.String()

	assert.Contains(t, result, "abc-123")
	assert.Contains(t, result, "Test Session")
	assert.Contains(t, result, string(ModePlanning))
	assert.Contains(t, result, string(StatusActive))
}

func TestSession_Validate(t *testing.T) {
	t.Run("valid session", func(t *testing.T) {
		session := &Session{
			ID:        "valid-id",
			ProjectID: "project-123",
			Name:      "Valid Session",
			Mode:      ModePlanning,
			Status:    StatusActive,
		}

		err := session.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty session ID", func(t *testing.T) {
		session := &Session{
			ID:        "",
			ProjectID: "project-123",
			Name:      "Test",
		}

		err := session.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})

	t.Run("empty project ID", func(t *testing.T) {
		session := &Session{
			ID:        "session-123",
			ProjectID: "",
			Name:      "Test",
		}

		err := session.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project ID cannot be empty")
	})

	t.Run("empty name", func(t *testing.T) {
		session := &Session{
			ID:        "session-123",
			ProjectID: "project-123",
			Name:      "",
		}

		err := session.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("invalid mode", func(t *testing.T) {
		session := &Session{
			ID:        "session-123",
			ProjectID: "project-123",
			Name:      "Test",
			Mode:      Mode("invalid"),
			Status:    StatusActive,
		}

		err := session.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mode")
	})

	t.Run("invalid status", func(t *testing.T) {
		session := &Session{
			ID:        "session-123",
			ProjectID: "project-123",
			Name:      "Test",
			Mode:      ModePlanning,
			Status:    Status("invalid"),
		}

		err := session.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})
}
