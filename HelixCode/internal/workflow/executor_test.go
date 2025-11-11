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

// ========================================
// executeLintStep Tests
// ========================================

func TestExecuteLintStep_Go(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_go_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a simple Go file for linting
	goFile := tempDir + "/main.go"
	err = os.WriteFile(goFile, []byte("package main\n\nfunc main() {\n}\n"), 0644)
	assert.NoError(t, err)

	proj := &project.Project{
		Type: "go",
		Path: tempDir,
	}

	step := &Step{
		ID:     "lint",
		Name:   "Lint Code",
		Action: StepActionLintCode,
	}

	output, err := executor.executeLintStep(context.Background(), step, proj)
	// gofmt should succeed on properly formatted code
	assert.NoError(t, err)
	assert.NotNil(t, output)
}

func TestExecuteLintStep_Node(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_node_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj := &project.Project{
		Type: "node",
		Path: tempDir,
	}

	step := &Step{
		ID:     "lint",
		Name:   "Lint Code",
		Action: StepActionLintCode,
	}

	// This will fail because npm run lint requires package.json
	_, err = executor.executeLintStep(context.Background(), step, proj)
	// We expect an error since there's no package.json
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lint execution failed")
}

func TestExecuteLintStep_Python(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_python_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj := &project.Project{
		Type: "python",
		Path: tempDir,
	}

	step := &Step{
		ID:     "lint",
		Name:   "Lint Code",
		Action: StepActionLintCode,
	}

	// This will likely fail if flake8 is not installed or no Python files exist
	_, err = executor.executeLintStep(context.Background(), step, proj)
	// Error is expected in test environment
	if err != nil {
		assert.Contains(t, err.Error(), "lint execution failed")
	}
}

func TestExecuteLintStep_Rust(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_rust_project")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj := &project.Project{
		Type: "rust",
		Path: tempDir,
	}

	step := &Step{
		ID:     "lint",
		Name:   "Lint Code",
		Action: StepActionLintCode,
	}

	// This will fail because cargo clippy requires Cargo.toml
	_, err = executor.executeLintStep(context.Background(), step, proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lint execution failed")
}

func TestExecuteLintStep_UnsupportedType(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	proj := &project.Project{
		Type: "unsupported",
		Path: "/tmp",
	}

	step := &Step{
		ID:     "lint",
		Name:   "Lint Code",
		Action: StepActionLintCode,
	}

	_, err := executor.executeLintStep(context.Background(), step, proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported project type for linting")
}

// ========================================
// executeBuildStep Tests
// ========================================

func TestExecuteBuildStep_Go(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_go_build")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a simple buildable Go file
	goFile := tempDir + "/main.go"
	err = os.WriteFile(goFile, []byte("package main\n\nfunc main() {\n}\n"), 0644)
	assert.NoError(t, err)

	// Create go.mod file for Go modules
	goMod := tempDir + "/go.mod"
	err = os.WriteFile(goMod, []byte("module testbuild\n\ngo 1.21\n"), 0644)
	assert.NoError(t, err)

	proj := &project.Project{
		Type: "go",
		Path: tempDir,
	}

	step := &Step{
		ID:     "build",
		Name:   "Build Project",
		Action: StepActionBuildProject,
	}

	output, err := executor.executeBuildStep(context.Background(), step, proj)
	// go build should succeed on valid code with go.mod
	assert.NoError(t, err)
	assert.NotNil(t, output)
}

func TestExecuteBuildStep_Node(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_node_build")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj := &project.Project{
		Type: "node",
		Path: tempDir,
	}

	step := &Step{
		ID:     "build",
		Name:   "Build Project",
		Action: StepActionBuildProject,
	}

	// This will fail because npm run build requires package.json
	_, err = executor.executeBuildStep(context.Background(), step, proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "build execution failed")
}

func TestExecuteBuildStep_Python(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_python_build")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj := &project.Project{
		Type: "python",
		Path: tempDir,
	}

	step := &Step{
		ID:     "build",
		Name:   "Build Project",
		Action: StepActionBuildProject,
	}

	// This will fail because setup.py is required
	_, err = executor.executeBuildStep(context.Background(), step, proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "build execution failed")
}

func TestExecuteBuildStep_Rust(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	tempDir, err := os.MkdirTemp("", "test_rust_build")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	proj := &project.Project{
		Type: "rust",
		Path: tempDir,
	}

	step := &Step{
		ID:     "build",
		Name:   "Build Project",
		Action: StepActionBuildProject,
	}

	// This will fail because cargo build requires Cargo.toml
	_, err = executor.executeBuildStep(context.Background(), step, proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "build execution failed")
}

func TestExecuteBuildStep_UnsupportedType(t *testing.T) {
	projectManager := project.NewManager()
	executor := NewExecutor(projectManager)

	proj := &project.Project{
		Type: "unsupported",
		Path: "/tmp",
	}

	step := &Step{
		ID:     "build",
		Name:   "Build Project",
		Action: StepActionBuildProject,
	}

	_, err := executor.executeBuildStep(context.Background(), step, proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported project type for building")
}
