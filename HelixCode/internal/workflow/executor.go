package workflow

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"dev.helix.code/internal/project"
)

// Executor handles workflow execution
type Executor struct {
	projectManager *project.Manager
}

// NewExecutor creates a new workflow executor
func NewExecutor(projectManager *project.Manager) *Executor {
	return &Executor{
		projectManager: projectManager,
	}
}

// ExecutePlanningWorkflow executes a planning workflow
func (e *Executor) ExecutePlanningWorkflow(ctx context.Context, projectID string) (*Workflow, error) {
	proj, err := e.projectManager.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	workflow := &Workflow{
		ID:          fmt.Sprintf("planning_%s_%d", projectID, time.Now().UnixNano()),
		Name:        "Project Architecture Planning",
		Description: "Generate system architecture and design for project",
		Mode:        "planning",
		Steps:       e.createPlanningSteps(proj),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      WorkflowStatusPending,
	}

	// Execute workflow
	go e.executeWorkflow(ctx, workflow, proj)

	return workflow, nil
}

// ExecuteBuildingWorkflow executes a building workflow
func (e *Executor) ExecuteBuildingWorkflow(ctx context.Context, projectID string) (*Workflow, error) {
	proj, err := e.projectManager.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	workflow := &Workflow{
		ID:          fmt.Sprintf("building_%s_%d", projectID, time.Now().UnixNano()),
		Name:        "Project Build",
		Description: "Build and compile project",
		Mode:        "building",
		Steps:       e.createBuildingSteps(proj),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      WorkflowStatusPending,
	}

	// Execute workflow
	go e.executeWorkflow(ctx, workflow, proj)

	return workflow, nil
}

// ExecuteTestingWorkflow executes a testing workflow
func (e *Executor) ExecuteTestingWorkflow(ctx context.Context, projectID string) (*Workflow, error) {
	proj, err := e.projectManager.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	workflow := &Workflow{
		ID:          fmt.Sprintf("testing_%s_%d", projectID, time.Now().UnixNano()),
		Name:        "Project Testing",
		Description: "Run comprehensive test suite",
		Mode:        "testing",
		Steps:       e.createTestingSteps(proj),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      WorkflowStatusPending,
	}

	// Execute workflow
	go e.executeWorkflow(ctx, workflow, proj)

	return workflow, nil
}

// ExecuteRefactoringWorkflow executes a refactoring workflow
func (e *Executor) ExecuteRefactoringWorkflow(ctx context.Context, projectID string) (*Workflow, error) {
	proj, err := e.projectManager.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	workflow := &Workflow{
		ID:          fmt.Sprintf("refactoring_%s_%d", projectID, time.Now().UnixNano()),
		Name:        "Code Refactoring",
		Description: "Refactor and improve code quality",
		Mode:        "refactoring",
		Steps:       e.createRefactoringSteps(proj),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      WorkflowStatusPending,
	}

	// Execute workflow
	go e.executeWorkflow(ctx, workflow, proj)

	return workflow, nil
}

// executeWorkflow executes a workflow
func (e *Executor) executeWorkflow(ctx context.Context, workflow *Workflow, proj *project.Project) {
	workflow.Status = WorkflowStatusRunning
	workflow.UpdatedAt = time.Now()

	for i := range workflow.Steps {
		step := &workflow.Steps[i]

		// Check if all dependencies are completed
		if !e.areDependenciesCompleted(workflow, step) {
			step.Status = StepStatusSkipped
			continue
		}

		step.Status = StepStatusRunning
		workflow.UpdatedAt = time.Now()

		// Execute step
		result, err := e.executeStep(ctx, step, proj)
		if err != nil {
			step.Status = StepStatusFailed
			step.Error = err.Error()
			workflow.Status = WorkflowStatusFailed
			workflow.UpdatedAt = time.Now()
			return
		}

		step.Status = StepStatusCompleted
		workflow.UpdatedAt = time.Now()

		// Add result to step (simplified for now)
		if result != "" {
			step.Status = StepStatusCompleted
		}
	}

	workflow.Status = WorkflowStatusCompleted
	workflow.UpdatedAt = time.Now()
}

// executeStep executes a single workflow step
func (e *Executor) executeStep(ctx context.Context, step *Step, proj *project.Project) (string, error) {
	switch step.Action {
	case StepActionAnalyzeCode:
		return e.executeAnalysisStep(ctx, step, proj)
	case StepActionGenerateCode:
		return e.executeGenerationStep(ctx, step, proj)
	case StepActionExecuteCommand:
		return e.executeCommandStep(ctx, step, proj)
	case StepActionRunTests:
		return e.executeTestStep(ctx, step, proj)
	case StepActionLintCode:
		return e.executeLintStep(ctx, step, proj)
	case StepActionBuildProject:
		return e.executeBuildStep(ctx, step, proj)
	default:
		return "", fmt.Errorf("unknown step action: %s", step.Action)
	}
}

// executeAnalysisStep executes an analysis step
func (e *Executor) executeAnalysisStep(ctx context.Context, step *Step, proj *project.Project) (string, error) {
	// For now, return a placeholder result
	return fmt.Sprintf("Analysis completed for: %s", step.Description), nil
}

// executeGenerationStep executes a code generation step
func (e *Executor) executeGenerationStep(ctx context.Context, step *Step, proj *project.Project) (string, error) {
	// For now, return a placeholder result
	return fmt.Sprintf("Code generation completed for: %s", step.Description), nil
}

// executeCommandStep executes a command execution step
func (e *Executor) executeCommandStep(ctx context.Context, step *Step, proj *project.Project) (string, error) {
	// Execute command in project directory
	cmd := exec.CommandContext(ctx, "bash", "-c", step.Description)
	cmd.Dir = proj.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %v\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// executeTestStep executes a test execution step
func (e *Executor) executeTestStep(ctx context.Context, step *Step, proj *project.Project) (string, error) {
	// Execute test command based on project type
	var cmd *exec.Cmd

	switch proj.Type {
	case "go":
		cmd = exec.CommandContext(ctx, "go", "test", "./...")
	case "node":
		cmd = exec.CommandContext(ctx, "npm", "test")
	case "python":
		cmd = exec.CommandContext(ctx, "python", "-m", "pytest")
	case "rust":
		cmd = exec.CommandContext(ctx, "cargo", "test")
	default:
		return "", fmt.Errorf("unsupported project type for testing: %s", proj.Type)
	}

	cmd.Dir = proj.Path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("test execution failed: %v\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// executeLintStep executes a linting step
func (e *Executor) executeLintStep(ctx context.Context, step *Step, proj *project.Project) (string, error) {
	// Execute lint command based on project type
	var cmd *exec.Cmd

	switch proj.Type {
	case "go":
		cmd = exec.CommandContext(ctx, "gofmt", "-l", ".")
	case "node":
		cmd = exec.CommandContext(ctx, "npm", "run", "lint")
	case "python":
		cmd = exec.CommandContext(ctx, "flake8", ".")
	case "rust":
		cmd = exec.CommandContext(ctx, "cargo", "clippy")
	default:
		return "", fmt.Errorf("unsupported project type for linting: %s", proj.Type)
	}

	cmd.Dir = proj.Path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("lint execution failed: %v\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// executeBuildStep executes a build step
func (e *Executor) executeBuildStep(ctx context.Context, step *Step, proj *project.Project) (string, error) {
	// Execute build command based on project type
	var cmd *exec.Cmd

	switch proj.Type {
	case "go":
		cmd = exec.CommandContext(ctx, "go", "build")
	case "node":
		cmd = exec.CommandContext(ctx, "npm", "run", "build")
	case "python":
		cmd = exec.CommandContext(ctx, "python", "setup.py", "build")
	case "rust":
		cmd = exec.CommandContext(ctx, "cargo", "build")
	default:
		return "", fmt.Errorf("unsupported project type for building: %s", proj.Type)
	}

	cmd.Dir = proj.Path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build execution failed: %v\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// areDependenciesCompleted checks if all step dependencies are completed
func (e *Executor) areDependenciesCompleted(workflow *Workflow, step *Step) bool {
	for _, depID := range step.Dependencies {
		depCompleted := false
		for _, s := range workflow.Steps {
			if s.ID == depID && s.Status == StepStatusCompleted {
				depCompleted = true
				break
			}
		}
		if !depCompleted {
			return false
		}
	}
	return true
}

// createPlanningSteps creates steps for planning workflow
func (e *Executor) createPlanningSteps(proj *project.Project) []Step {
	return []Step{
		{
			ID:          "analyze_requirements",
			Name:        "Analyze Requirements",
			Description: "Analyze project requirements and constraints",
			Type:        StepTypeAnalysis,
			Action:      StepActionAnalyzeCode,
			Status:      StepStatusPending,
		},
		{
			ID:           "generate_architecture",
			Name:         "Generate Architecture",
			Description:  "Generate system architecture and design",
			Type:         StepTypeGeneration,
			Action:       StepActionGenerateCode,
			Dependencies: []string{"analyze_requirements"},
			Status:       StepStatusPending,
		},
	}
}

// createBuildingSteps creates steps for building workflow
func (e *Executor) createBuildingSteps(proj *project.Project) []Step {
	return []Step{
		{
			ID:          "setup_environment",
			Name:        "Setup Environment",
			Description: "Setup build environment and dependencies",
			Type:        StepTypeExecution,
			Action:      StepActionExecuteCommand,
			Status:      StepStatusPending,
		},
		{
			ID:           "compile_code",
			Name:         "Compile Code",
			Description:  proj.Metadata.BuildCommand,
			Type:         StepTypeExecution,
			Action:       StepActionBuildProject,
			Dependencies: []string{"setup_environment"},
			Status:       StepStatusPending,
		},
	}
}

// createTestingSteps creates steps for testing workflow
func (e *Executor) createTestingSteps(proj *project.Project) []Step {
	return []Step{
		{
			ID:          "unit_tests",
			Name:        "Unit Tests",
			Description: "Run unit tests",
			Type:        StepTypeExecution,
			Action:      StepActionRunTests,
			Status:      StepStatusPending,
		},
		{
			ID:           "integration_tests",
			Name:         "Integration Tests",
			Description:  "Run integration tests",
			Type:         StepTypeExecution,
			Action:       StepActionRunTests,
			Dependencies: []string{"unit_tests"},
			Status:       StepStatusPending,
		},
	}
}

// createRefactoringSteps creates steps for refactoring workflow
func (e *Executor) createRefactoringSteps(proj *project.Project) []Step {
	return []Step{
		{
			ID:          "analyze_codebase",
			Name:        "Analyze Codebase",
			Description: "Analyze codebase for refactoring opportunities",
			Type:        StepTypeAnalysis,
			Action:      StepActionAnalyzeCode,
			Status:      StepStatusPending,
		},
		{
			ID:           "refactor_code",
			Name:         "Refactor Code",
			Description:  "Perform code refactoring",
			Type:         StepTypeGeneration,
			Action:       StepActionGenerateCode,
			Dependencies: []string{"analyze_codebase"},
			Status:       StepStatusPending,
		},
	}
}
