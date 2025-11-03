package workflow

import (
	"time"
)

// Workflow represents a development workflow
type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Mode        string         `json:"mode"`
	Steps       []Step         `json:"steps"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Status      WorkflowStatus `json:"status"`
}

// Step represents a workflow step
type Step struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Type         StepType   `json:"type"`
	Action       StepAction `json:"action"`
	Dependencies []string   `json:"dependencies"`
	Status       StepStatus `json:"status"`
	Error        string     `json:"error,omitempty"`
}

// StepType represents the type of workflow step
type StepType string

const (
	StepTypeAnalysis   StepType = "analysis"
	StepTypeGeneration StepType = "generation"
	StepTypeExecution  StepType = "execution"
	StepTypeValidation StepType = "validation"
)

// StepAction represents the action to perform in a step
type StepAction string

const (
	StepActionAnalyzeCode    StepAction = "analyze_code"
	StepActionGenerateCode   StepAction = "generate_code"
	StepActionExecuteCommand StepAction = "execute_command"
	StepActionRunTests       StepAction = "run_tests"
	StepActionLintCode       StepAction = "lint_code"
	StepActionBuildProject   StepAction = "build_project"
)

// StepStatus represents the status of a workflow step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// WorkflowStatus represents the overall workflow status
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
)
