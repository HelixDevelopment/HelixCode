package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.projects)
	assert.Nil(t, manager.activeProject)
}

func TestCreateProject(t *testing.T) {
	manager := NewManager()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a go.mod file to make it a Go project
	goModPath := filepath.Join(tempDir, "go.mod")
	err = os.WriteFile(goModPath, []byte("module test\n\ngo 1.21"), 0644)
	assert.NoError(t, err)

	project, err := manager.CreateProject(context.Background(), "test-project", "A test project", tempDir, "go")
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "test-project", project.Name)
	assert.Equal(t, "go", project.Type)
	assert.Equal(t, "go build", project.Metadata.BuildCommand)
	assert.Contains(t, manager.projects, project.ID)
}

func TestCreateProject_InvalidPath(t *testing.T) {
	manager := NewManager()

	_, err := manager.CreateProject(context.Background(), "test", "desc", "/nonexistent/path", "go")
	assert.Error(t, err)
}

func TestGetProject(t *testing.T) {
	manager := NewManager()

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	project, err := manager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	retrieved, err := manager.GetProject(context.Background(), project.ID)
	assert.NoError(t, err)
	assert.Equal(t, project, retrieved)
}

func TestGetProject_NotFound(t *testing.T) {
	manager := NewManager()

	_, err := manager.GetProject(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestListProjects(t *testing.T) {
	manager := NewManager()

	tempDir1, err := os.MkdirTemp("", "test_project1")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir1)

	tempDir2, err := os.MkdirTemp("", "test_project2")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir2)

	project1, err := manager.CreateProject(context.Background(), "test1", "desc1", tempDir1, "generic")
	assert.NoError(t, err)

	project2, err := manager.CreateProject(context.Background(), "test2", "desc2", tempDir2, "generic")
	assert.NoError(t, err)

	projects, err := manager.ListProjects(context.Background())
	assert.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Contains(t, projects, project1)
	assert.Contains(t, projects, project2)
}

func TestSetActiveProject(t *testing.T) {
	manager := NewManager()

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	project, err := manager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	err = manager.SetActiveProject(context.Background(), project.ID)
	assert.NoError(t, err)
	assert.True(t, project.Active)
	assert.Equal(t, project, manager.activeProject)
}

func TestGetActiveProject(t *testing.T) {
	manager := NewManager()

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	project, err := manager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	manager.SetActiveProject(context.Background(), project.ID)

	active, err := manager.GetActiveProject(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, project, active)
}

func TestGetActiveProject_NoActive(t *testing.T) {
	manager := NewManager()

	_, err := manager.GetActiveProject(context.Background())
	assert.Error(t, err)
}

func TestUpdateProjectMetadata(t *testing.T) {
	manager := NewManager()

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	project, err := manager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	newMetadata := Metadata{
		BuildCommand: "echo build",
		TestCommand:  "echo test",
	}

	err = manager.UpdateProjectMetadata(context.Background(), project.ID, newMetadata)
	assert.NoError(t, err)
	assert.Equal(t, newMetadata, project.Metadata)
}

func TestDeleteProject(t *testing.T) {
	manager := NewManager()

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	project, err := manager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	// Set as active
	manager.SetActiveProject(context.Background(), project.ID)

	err = manager.DeleteProject(context.Background(), project.ID)
	assert.NoError(t, err)
	assert.NotContains(t, manager.projects, project.ID)
	assert.Nil(t, manager.activeProject)
}
