package workflow

import (
	"context"
	"os"
	"testing"

	"dev.helix.code/internal/project"
	"github.com/stretchr/testify/assert"
)

func TestNewExecutor(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	assert.NotNil(t, executor)
	assert.Equal(t, projectManager, executor.projectManager)
}

func TestExecutePlanningWorkflow(t *testing.T) {
	projectManager := project.NewManager()

	// Create a test project
	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj, err := projectManager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	executor := NewExecutor(projectManager)

	workflow, err := executor.ExecutePlanningWorkflow(context.Background(), proj.ID)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "planning", workflow.Mode)
	assert.Equal(t, WorkflowStatusPending, workflow.Status)
	assert.Len(t, workflow.Steps, 2)
}

func TestExecuteBuildingWorkflow(t *testing.T) {
	projectManager := project.NewManager()

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj, err := projectManager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	executor := NewExecutor(projectManager)

	workflow, err := executor.ExecuteBuildingWorkflow(context.Background(), proj.ID)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "building", workflow.Mode)
	assert.Len(t, workflow.Steps, 2)
}

func TestExecuteTestingWorkflow(t *testing.T) {
	projectManager := project.NewManager()

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj, err := projectManager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	executor := NewExecutor(projectManager)

	workflow, err := executor.ExecuteTestingWorkflow(context.Background(), proj.ID)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "testing", workflow.Mode)
	assert.Len(t, workflow.Steps, 2)
}

func TestExecuteRefactoringWorkflow(t *testing.T) {
	projectManager := project.NewManager()

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj, err := projectManager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	executor := NewExecutor(projectManager)

	workflow, err := executor.ExecuteRefactoringWorkflow(context.Background(), proj.ID)
	assert.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "refactoring", workflow.Mode)
	assert.Len(t, workflow.Steps, 2)
}

func TestExecuteWorkflow_InvalidProject(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	_, err := executor.ExecutePlanningWorkflow(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestExecuteStep_Analysis(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj, err := projectManager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	step := &Step{
		ID:          "test",
		Name:        "Test Step",
		Description: "Test analysis",
		Type:        StepTypeAnalysis,
		Action:      StepActionAnalyzeCode,
	}

	result, err := executor.executeStep(context.Background(), step, proj)
	assert.NoError(t, err)
	assert.Contains(t, result, "Analysis completed")
}

func TestExecuteStep_Generation(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj, err := projectManager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	step := &Step{
		ID:          "test",
		Name:        "Test Step",
		Description: "Test generation",
		Type:        StepTypeGeneration,
		Action:      StepActionGenerateCode,
	}

	result, err := executor.executeStep(context.Background(), step, proj)
	assert.NoError(t, err)
	assert.Contains(t, result, "Code generation completed")
}

func TestExecuteStep_UnknownAction(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj, err := projectManager.CreateProject(context.Background(), "test", "desc", tempDir, "generic")
	assert.NoError(t, err)

	step := &Step{
		ID:          "test",
		Name:        "Test Step",
		Description: "Test",
		Type:        StepTypeExecution,
		Action:      "unknown_action",
	}

	_, err = executor.executeStep(context.Background(), step, proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown step action")
}

func TestAreDependenciesCompleted(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	workflow := &Workflow{
		Steps: []Step{
			{ID: "step1", Status: StepStatusCompleted},
			{ID: "step2", Status: StepStatusPending},
		},
	}

	step := &Step{
		Dependencies: []string{"step1"},
	}

	assert.True(t, executor.areDependenciesCompleted(workflow, step))

	step.Dependencies = []string{"step1", "step2"}
	assert.False(t, executor.areDependenciesCompleted(workflow, step))
}
