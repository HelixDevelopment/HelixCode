package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/agent/task"
)

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	ID           string
	Name         string
	AgentType    AgentType
	RequiredCaps []Capability
	Input        map[string]interface{}
	DependsOn    []string // IDs of steps that must complete first
	Optional     bool     // If true, workflow continues even if this step fails
}

// Workflow represents a multi-step process executed by multiple agents
type Workflow struct {
	ID          string
	Name        string
	Description string
	Steps       []*WorkflowStep
	Status      WorkflowStatus
	Results     map[string]*task.Result // step ID -> result
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	mu          sync.RWMutex
}

// WorkflowStatus represents the current status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// NewWorkflow creates a new workflow
func NewWorkflow(name, description string) *Workflow {
	return &Workflow{
		ID:          GenerateWorkflowID(),
		Name:        name,
		Description: description,
		Steps:       make([]*WorkflowStep, 0),
		Status:      WorkflowStatusPending,
		Results:     make(map[string]*task.Result),
		CreatedAt:   time.Now(),
	}
}

// AddStep adds a step to the workflow
func (w *Workflow) AddStep(step *WorkflowStep) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Steps = append(w.Steps, step)
}

// Start marks the workflow as started
func (w *Workflow) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	w.StartedAt = &now
	w.Status = WorkflowStatusRunning
}

// Complete marks the workflow as completed
func (w *Workflow) Complete() {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	w.CompletedAt = &now
	w.Status = WorkflowStatusCompleted
}

// Fail marks the workflow as failed
func (w *Workflow) Fail() {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	w.CompletedAt = &now
	w.Status = WorkflowStatusFailed
}

// Cancel marks the workflow as cancelled
func (w *Workflow) Cancel() {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	w.CompletedAt = &now
	w.Status = WorkflowStatusCancelled
}

// SetStepResult records the result of a workflow step
func (w *Workflow) SetStepResult(stepID string, result *task.Result) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Results[stepID] = result
}

// GetStepResult retrieves the result of a workflow step
func (w *Workflow) GetStepResult(stepID string) (*task.Result, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result, ok := w.Results[stepID]
	return result, ok
}

// IsStepReady checks if a step's dependencies are satisfied
func (w *Workflow) IsStepReady(step *WorkflowStep) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Check if all dependencies are completed successfully
	for _, depID := range step.DependsOn {
		result, ok := w.Results[depID]
		if !ok {
			return false // Dependency not yet executed
		}
		if !result.Success {
			// If dependency failed, check if it was optional
			for _, s := range w.Steps {
				if s.ID == depID && s.Optional {
					continue // Optional step can fail
				}
			}
			return false // Required dependency failed
		}
	}
	return true
}

// GetReadySteps returns all steps that are ready to execute
func (w *Workflow) GetReadySteps() []*WorkflowStep {
	w.mu.RLock()
	defer w.mu.RUnlock()

	ready := make([]*WorkflowStep, 0)
	for _, step := range w.Steps {
		// Skip if already executed
		if _, executed := w.Results[step.ID]; executed {
			continue
		}
		// Check if dependencies are satisfied
		if w.IsStepReady(step) {
			ready = append(ready, step)
		}
	}
	return ready
}

// WorkflowExecutor executes workflows using the coordinator
type WorkflowExecutor struct {
	coordinator *Coordinator
	workflows   map[string]*Workflow
	mu          sync.RWMutex
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(coordinator *Coordinator) *WorkflowExecutor {
	return &WorkflowExecutor{
		coordinator: coordinator,
		workflows:   make(map[string]*Workflow),
	}
}

// Execute executes a workflow
func (we *WorkflowExecutor) Execute(ctx context.Context, workflow *Workflow) error {
	// Register workflow
	we.mu.Lock()
	we.workflows[workflow.ID] = workflow
	we.mu.Unlock()

	// Start workflow
	workflow.Start()

	// Execute steps in dependency order
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			workflow.Cancel()
			return ctx.Err()
		default:
		}

		// Get ready steps
		readySteps := workflow.GetReadySteps()
		if len(readySteps) == 0 {
			// No more steps ready - check if workflow is complete
			workflow.mu.RLock()
			totalSteps := len(workflow.Steps)
			completedSteps := len(workflow.Results)
			workflow.mu.RUnlock()

			if completedSteps == totalSteps {
				// All steps completed
				workflow.Complete()
				return nil
			}

			// Check if we're stuck (no ready steps but not complete)
			// This happens when all remaining steps depend on failed required steps
			allStepsDone := true
			for _, step := range workflow.Steps {
				if _, executed := workflow.Results[step.ID]; !executed {
					allStepsDone = false
					break
				}
			}

			if !allStepsDone {
				// We have unexecuted steps but none are ready - workflow is stuck
				workflow.Fail()
				return fmt.Errorf("workflow stuck: remaining steps have unsatisfied dependencies")
			}

			// All steps are done
			workflow.Complete()
			return nil
		}

		// Execute ready steps in parallel
		var wg sync.WaitGroup
		errChan := make(chan error, len(readySteps))

		for _, step := range readySteps {
			wg.Add(1)
			go func(s *WorkflowStep) {
				defer wg.Done()
				if err := we.executeStep(ctx, workflow, s); err != nil {
					errChan <- err
				}
			}(step)
		}

		wg.Wait()
		close(errChan)

		// Check for errors from non-optional steps
		for err := range errChan {
			if err != nil {
				workflow.Fail()
				return err
			}
		}

		// Continue to next iteration to process newly ready steps
	}
}

// executeStep executes a single workflow step
func (we *WorkflowExecutor) executeStep(ctx context.Context, workflow *Workflow, step *WorkflowStep) error {
	// Find suitable agent
	var agent Agent
	var err error

	if len(step.RequiredCaps) > 0 {
		// Find by capability
		agents := we.coordinator.registry.GetByCapability(step.RequiredCaps[0])
		if len(agents) == 0 {
			if step.Optional {
				// Create a failed result for optional step
				result := &task.Result{
					TaskID:    step.ID,
					AgentID:   "none",
					Success:   false,
					Error:     fmt.Sprintf("no agent found with capability %s", step.RequiredCaps[0]),
					Timestamp: time.Now(),
				}
				workflow.SetStepResult(step.ID, result)
				return nil
			}
			return fmt.Errorf("no agent found with capability %s for step %s", step.RequiredCaps[0], step.ID)
		}
		agent = agents[0]
	} else {
		// Find by type
		agents := we.coordinator.registry.GetByType(step.AgentType)
		if len(agents) == 0 {
			if step.Optional {
				result := &task.Result{
					TaskID:    step.ID,
					AgentID:   "none",
					Success:   false,
					Error:     fmt.Sprintf("no agent found of type %s", step.AgentType),
					Timestamp: time.Now(),
				}
				workflow.SetStepResult(step.ID, result)
				return nil
			}
			return fmt.Errorf("no agent found of type %s for step %s", step.AgentType, step.ID)
		}
		agent = agents[0]
	}

	// Prepare input by merging step input with outputs from dependencies
	input := make(map[string]interface{})
	for k, v := range step.Input {
		input[k] = v
	}

	// Add outputs from dependency steps
	for _, depID := range step.DependsOn {
		if result, ok := workflow.GetStepResult(depID); ok && result.Success {
			// Merge dependency output into input
			for k, v := range result.Output {
				// Use dependency output if not already specified in step input
				if _, exists := input[k]; !exists {
					input[k] = v
				}
			}
		}
	}

	// Create task for this step
	t := task.NewTask(
		task.TaskType(step.Name), // Use step name as task type
		step.Name,
		fmt.Sprintf("Workflow step: %s", step.Name),
		task.PriorityNormal,
	)
	t.Input = input
	t.RequiredCapabilities = make([]string, len(step.RequiredCaps))
	for i, cap := range step.RequiredCaps {
		t.RequiredCapabilities[i] = string(cap)
	}

	// Execute task
	result, err := agent.Execute(ctx, t)
	if err != nil {
		if step.Optional {
			// Record failure but continue
			result = &task.Result{
				TaskID:    step.ID,
				AgentID:   agent.ID(),
				Success:   false,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
			workflow.SetStepResult(step.ID, result)
			return nil
		}
		return fmt.Errorf("step %s failed: %w", step.ID, err)
	}

	// Record result
	workflow.SetStepResult(step.ID, result)
	return nil
}

// GetWorkflow retrieves a workflow by ID
func (we *WorkflowExecutor) GetWorkflow(id string) (*Workflow, error) {
	we.mu.RLock()
	defer we.mu.RUnlock()

	workflow, ok := we.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	return workflow, nil
}

// ListWorkflows returns all workflows
func (we *WorkflowExecutor) ListWorkflows() []*Workflow {
	we.mu.RLock()
	defer we.mu.RUnlock()

	workflows := make([]*Workflow, 0, len(we.workflows))
	for _, w := range we.workflows {
		workflows = append(workflows, w)
	}
	return workflows
}

// GenerateWorkflowID generates a unique workflow ID
func GenerateWorkflowID() string {
	return fmt.Sprintf("workflow-%d", time.Now().UnixNano())
}
