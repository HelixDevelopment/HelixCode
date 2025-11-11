package project

import (
	"context"
	"fmt"
	"testing"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ========================================
// DatabaseManager Constructor Tests
// ========================================

func TestNewDatabaseManager(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	assert.NotNil(t, dm)
	assert.Equal(t, mockDB, dm.db)
}

// ========================================
// CreateProject Tests
// ========================================

func TestDatabaseManager_CreateProjectSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()
	now := time.Now()

	mockRow := database.NewMockRowWithValues(now, now)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.CreateProject(ctx, "Test Project", "Test description", "/test/path", "go", ownerID.String())

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, "Test description", project.Description)
	assert.Equal(t, "/test/path", project.Path)
	assert.Equal(t, "go", project.Type)
	assert.False(t, project.Active)
	assert.Equal(t, "go build", project.Metadata.BuildCommand)
	assert.Equal(t, "go test ./...", project.Metadata.TestCommand)
	assert.Equal(t, "gofmt -l .", project.Metadata.LintCommand)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_CreateProjectInvalidOwnerID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	project, err := dm.CreateProject(ctx, "Test Project", "Test description", "/test/path", "go", "invalid-uuid")

	assert.Error(t, err)
	assert.Nil(t, project)
	assert.Contains(t, err.Error(), "invalid owner ID")
}

func TestDatabaseManager_CreateProjectDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()

	mockRow := database.NewMockRowWithError(fmt.Errorf("database connection failed"))
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.CreateProject(ctx, "Test Project", "Test description", "/test/path", "go", ownerID.String())

	assert.Error(t, err)
	assert.Nil(t, project)
	assert.Contains(t, err.Error(), "failed to create project in database")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_CreateProjectNodeType(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()
	now := time.Now()

	mockRow := database.NewMockRowWithValues(now, now)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.CreateProject(ctx, "Node Project", "Node app", "/node/path", "node", ownerID.String())

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "node", project.Type)
	assert.Equal(t, "npm run build", project.Metadata.BuildCommand)
	assert.Equal(t, "npm test", project.Metadata.TestCommand)
	assert.Equal(t, "npm run lint", project.Metadata.LintCommand)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_CreateProjectPythonType(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()
	now := time.Now()

	mockRow := database.NewMockRowWithValues(now, now)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.CreateProject(ctx, "Python Project", "Python app", "/python/path", "python", ownerID.String())

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "python", project.Type)
	assert.Equal(t, "python setup.py build", project.Metadata.BuildCommand)
	assert.Equal(t, "python -m pytest", project.Metadata.TestCommand)
	assert.Equal(t, "flake8 .", project.Metadata.LintCommand)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_CreateProjectRustType(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()
	now := time.Now()

	mockRow := database.NewMockRowWithValues(now, now)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.CreateProject(ctx, "Rust Project", "Rust app", "/rust/path", "rust", ownerID.String())

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "rust", project.Type)
	assert.Equal(t, "cargo build", project.Metadata.BuildCommand)
	assert.Equal(t, "cargo test", project.Metadata.TestCommand)
	assert.Equal(t, "cargo clippy", project.Metadata.LintCommand)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_CreateProjectUnknownType(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()
	now := time.Now()

	mockRow := database.NewMockRowWithValues(now, now)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.CreateProject(ctx, "Unknown Project", "Unknown type", "/unknown/path", "unknown", ownerID.String())

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "unknown", project.Type)
	assert.Equal(t, "echo 'No build command configured'", project.Metadata.BuildCommand)
	assert.Equal(t, "echo 'No test command configured'", project.Metadata.TestCommand)
	assert.Equal(t, "echo 'No lint command configured'", project.Metadata.LintCommand)
	mockDB.AssertExpectations(t)
}

// ========================================
// GetProject Tests
// ========================================

func TestDatabaseManager_GetProjectSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()
	ownerID := uuid.New()
	now := time.Now()

	config := map[string]interface{}{
		"type": "go",
		"metadata": map[string]interface{}{
			"build_command": "go build",
			"test_command":  "go test ./...",
			"lint_command":  "gofmt -l .",
		},
	}

	mockRow := database.NewMockRowWithValues(
		projectID, "Test Project", "Test description", ownerID, "/test/path",
		config, "active", now, now,
	)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.GetProject(ctx, projectID.String())

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, projectID.String(), project.ID)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, "Test description", project.Description)
	assert.Equal(t, "/test/path", project.Path)
	assert.Equal(t, "go", project.Type)
	assert.Equal(t, "go build", project.Metadata.BuildCommand)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_GetProjectInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	project, err := dm.GetProject(ctx, "invalid-uuid")

	assert.Error(t, err)
	assert.Nil(t, project)
	assert.Contains(t, err.Error(), "invalid project ID")
}

func TestDatabaseManager_GetProjectNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()

	mockRow := database.NewMockRowWithError(fmt.Errorf("no rows in result set"))
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.GetProject(ctx, projectID.String())

	assert.Error(t, err)
	assert.Nil(t, project)
	assert.Contains(t, err.Error(), "failed to get project from database")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_GetProjectDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()

	mockRow := database.NewMockRowWithError(fmt.Errorf("database connection failed"))
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	project, err := dm.GetProject(ctx, projectID.String())

	assert.Error(t, err)
	assert.Nil(t, project)
	assert.Contains(t, err.Error(), "failed to get project from database")
	mockDB.AssertExpectations(t)
}

// ========================================
// ListProjects Tests
// ========================================

func TestDatabaseManager_ListProjectsSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()
	projectID1 := uuid.New()
	projectID2 := uuid.New()
	now := time.Now()

	config1 := map[string]interface{}{
		"type": "go",
		"metadata": map[string]interface{}{
			"build_command": "go build",
		},
	}

	config2 := map[string]interface{}{
		"type": "node",
		"metadata": map[string]interface{}{
			"build_command": "npm run build",
		},
	}

	mockRows := database.NewMockRows([][]interface{}{
		{projectID1, "Project 1", "Description 1", ownerID, "/path1", config1, "active", now, now},
		{projectID2, "Project 2", "Description 2", ownerID, "/path2", config2, "active", now, now},
	})

	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	projects, err := dm.ListProjects(ctx, ownerID.String())

	assert.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "Project 1", projects[0].Name)
	assert.Equal(t, "Project 2", projects[1].Name)
	assert.Equal(t, "go", projects[0].Type)
	assert.Equal(t, "node", projects[1].Type)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_ListProjectsEmpty(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()

	mockRows := database.NewMockRows([][]interface{}{})
	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	projects, err := dm.ListProjects(ctx, ownerID.String())

	assert.NoError(t, err)
	assert.Empty(t, projects)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_ListProjectsInvalidOwnerID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	projects, err := dm.ListProjects(ctx, "invalid-uuid")

	assert.Error(t, err)
	assert.Nil(t, projects)
	assert.Contains(t, err.Error(), "invalid owner ID")
}

func TestDatabaseManager_ListProjectsQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()

	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(nil, fmt.Errorf("query failed"))

	projects, err := dm.ListProjects(ctx, ownerID.String())

	assert.Error(t, err)
	assert.Nil(t, projects)
	assert.Contains(t, err.Error(), "failed to query projects")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_ListProjectsIterationError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	ownerID := uuid.New()

	mockRows := database.NewMockRowsWithError(fmt.Errorf("iteration failed"))
	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	projects, err := dm.ListProjects(ctx, ownerID.String())

	assert.Error(t, err)
	assert.Nil(t, projects)
	assert.Contains(t, err.Error(), "error iterating project rows")
	mockDB.AssertExpectations(t)
}

// ========================================
// UpdateProjectMetadata Tests
// ========================================

func TestDatabaseManager_UpdateProjectMetadataSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()

	currentConfig := map[string]interface{}{
		"type": "go",
		"metadata": map[string]interface{}{
			"build_command": "go build",
		},
	}

	newMetadata := Metadata{
		BuildCommand: "go build -v",
		TestCommand:  "go test -v ./...",
		LintCommand:  "golangci-lint run",
		Dependencies: []string{"dep1", "dep2"},
		Environment:  map[string]string{"GO_ENV": "production"},
	}

	// Mock getting current config
	mockRow := database.NewMockRowWithValues(currentConfig)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow).Once()

	// Mock update execution
	mockDB.MockExecSuccess(1)

	err := dm.UpdateProjectMetadata(ctx, projectID.String(), newMetadata)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_UpdateProjectMetadataInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	metadata := Metadata{BuildCommand: "go build"}

	err := dm.UpdateProjectMetadata(ctx, "invalid-uuid", metadata)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid project ID")
}

func TestDatabaseManager_UpdateProjectMetadataGetConfigError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()
	metadata := Metadata{BuildCommand: "go build"}

	mockRow := database.NewMockRowWithError(fmt.Errorf("config not found"))
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	err := dm.UpdateProjectMetadata(ctx, projectID.String(), metadata)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get project config")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_UpdateProjectMetadataUpdateError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()

	currentConfig := map[string]interface{}{
		"type": "go",
	}

	metadata := Metadata{BuildCommand: "go build"}

	// Mock getting current config
	mockRow := database.NewMockRowWithValues(currentConfig)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow).Once()

	// Mock update execution failure
	mockDB.MockExecError(fmt.Errorf("update failed"))

	err := dm.UpdateProjectMetadata(ctx, projectID.String(), metadata)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update project metadata")
	mockDB.AssertExpectations(t)
}

// ========================================
// DeleteProject Tests
// ========================================

func TestDatabaseManager_DeleteProjectSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()

	mockDB.MockExecSuccess(1)

	err := dm.DeleteProject(ctx, projectID.String())

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_DeleteProjectInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	err := dm.DeleteProject(ctx, "invalid-uuid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid project ID")
}

func TestDatabaseManager_DeleteProjectNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()

	mockDB.MockExecSuccess(0) // No rows affected

	err := dm.DeleteProject(ctx, projectID.String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_DeleteProjectDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	projectID := uuid.New()

	mockDB.MockExecError(fmt.Errorf("database error"))

	err := dm.DeleteProject(ctx, projectID.String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete project")
	mockDB.AssertExpectations(t)
}

// ========================================
// Helper Method Tests
// ========================================

func TestDatabaseManager_DetectProjectTypeGo(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	metadata := Metadata{Environment: make(map[string]string)}
	dm.detectProjectType("/test/path", "go", &metadata)

	assert.Equal(t, "go build", metadata.BuildCommand)
	assert.Equal(t, "go test ./...", metadata.TestCommand)
	assert.Equal(t, "gofmt -l .", metadata.LintCommand)
}

func TestDatabaseManager_DetectProjectTypeNode(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	metadata := Metadata{Environment: make(map[string]string)}
	dm.detectProjectType("/test/path", "node", &metadata)

	assert.Equal(t, "npm run build", metadata.BuildCommand)
	assert.Equal(t, "npm test", metadata.TestCommand)
	assert.Equal(t, "npm run lint", metadata.LintCommand)
}

func TestDatabaseManager_DetectProjectTypePython(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	metadata := Metadata{Environment: make(map[string]string)}
	dm.detectProjectType("/test/path", "python", &metadata)

	assert.Equal(t, "python setup.py build", metadata.BuildCommand)
	assert.Equal(t, "python -m pytest", metadata.TestCommand)
	assert.Equal(t, "flake8 .", metadata.LintCommand)
}

func TestDatabaseManager_DetectProjectTypeRust(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	metadata := Metadata{Environment: make(map[string]string)}
	dm.detectProjectType("/test/path", "rust", &metadata)

	assert.Equal(t, "cargo build", metadata.BuildCommand)
	assert.Equal(t, "cargo test", metadata.TestCommand)
	assert.Equal(t, "cargo clippy", metadata.LintCommand)
}

func TestDatabaseManager_DetectProjectTypeDefault(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	metadata := Metadata{Environment: make(map[string]string)}
	dm.detectProjectType("/test/path", "unknown", &metadata)

	assert.Equal(t, "echo 'No build command configured'", metadata.BuildCommand)
	assert.Equal(t, "echo 'No test command configured'", metadata.TestCommand)
	assert.Equal(t, "echo 'No lint command configured'", metadata.LintCommand)
}

func TestDatabaseManager_ConvertToMetadataComplete(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	data := map[string]interface{}{
		"build_command":    "go build",
		"test_command":     "go test",
		"lint_command":     "golangci-lint",
		"dependencies":     []string{"dep1", "dep2"},
		"environment":      map[string]string{"GO_ENV": "test"},
		"framework":        "gin",
		"language_version": "1.21",
	}

	metadata := dm.convertToMetadata(data)

	assert.Equal(t, "go build", metadata.BuildCommand)
	assert.Equal(t, "go test", metadata.TestCommand)
	assert.Equal(t, "golangci-lint", metadata.LintCommand)
	assert.Equal(t, []string{"dep1", "dep2"}, metadata.Dependencies)
	assert.Equal(t, map[string]string{"GO_ENV": "test"}, metadata.Environment)
	assert.Equal(t, "gin", metadata.Framework)
	assert.Equal(t, "1.21", metadata.LanguageVersion)
}

func TestDatabaseManager_ConvertToMetadataEmpty(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	data := map[string]interface{}{}

	metadata := dm.convertToMetadata(data)

	assert.Empty(t, metadata.BuildCommand)
	assert.Empty(t, metadata.TestCommand)
	assert.Empty(t, metadata.LintCommand)
	assert.Empty(t, metadata.Dependencies)
	assert.NotNil(t, metadata.Environment)
	assert.Empty(t, metadata.Framework)
	assert.Empty(t, metadata.LanguageVersion)
}

func TestDatabaseManager_ConvertToMetadataPartial(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	data := map[string]interface{}{
		"build_command": "make build",
		"framework":     "echo",
	}

	metadata := dm.convertToMetadata(data)

	assert.Equal(t, "make build", metadata.BuildCommand)
	assert.Empty(t, metadata.TestCommand)
	assert.Empty(t, metadata.LintCommand)
	assert.Equal(t, "echo", metadata.Framework)
}
