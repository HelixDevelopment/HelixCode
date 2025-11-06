package planmode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkflowProgress tracks workflow progress
type WorkflowProgress struct {
	Phase          string
	Status         string
	OptionsCount   int
	TotalSteps     int
	CurrentStep    int
	CompletedSteps int
}

// StateManager manages plan mode state
type StateManager struct {
	currentMode Mode
	plans       sync.Map // map[string]*Plan
	options     sync.Map // map[string][]*PlanOption
	selections  sync.Map // map[string]*Selection
	executions  sync.Map // map[string]*ExecutionResult
	mu          sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		currentMode: ModeNormal,
	}
}

// StorePlan stores a plan
func (sm *StateManager) StorePlan(plan *Plan) error {
	if plan.ID == "" {
		return fmt.Errorf("plan ID is required")
	}
	sm.plans.Store(plan.ID, plan)
	return nil
}

// GetPlan retrieves a plan
func (sm *StateManager) GetPlan(id string) (*Plan, error) {
	val, ok := sm.plans.Load(id)
	if !ok {
		return nil, fmt.Errorf("plan not found: %s", id)
	}
	return val.(*Plan), nil
}

// StoreOptions stores options for a plan
func (sm *StateManager) StoreOptions(planID string, options []*PlanOption) error {
	if len(options) == 0 {
		return fmt.Errorf("at least one option required")
	}
	sm.options.Store(planID, options)
	return nil
}

// GetOptions retrieves options for a plan
func (sm *StateManager) GetOptions(planID string) ([]*PlanOption, error) {
	val, ok := sm.options.Load(planID)
	if !ok {
		return nil, fmt.Errorf("options not found for plan: %s", planID)
	}
	return val.([]*PlanOption), nil
}

// StoreSelection stores a user selection
func (sm *StateManager) StoreSelection(planID string, selection *Selection) error {
	sm.selections.Store(planID, selection)
	return nil
}

// GetSelection retrieves a selection
func (sm *StateManager) GetSelection(planID string) (*Selection, error) {
	val, ok := sm.selections.Load(planID)
	if !ok {
		return nil, fmt.Errorf("selection not found for plan: %s", planID)
	}
	return val.(*Selection), nil
}

// StoreExecution stores an execution result
func (sm *StateManager) StoreExecution(execution *ExecutionResult) error {
	sm.executions.Store(execution.ID, execution)
	return nil
}

// GetExecution retrieves an execution result
func (sm *StateManager) GetExecution(id string) (*ExecutionResult, error) {
	val, ok := sm.executions.Load(id)
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	return val.(*ExecutionResult), nil
}

// ListPlans lists all plans
func (sm *StateManager) ListPlans() []*Plan {
	var plans []*Plan
	sm.plans.Range(func(key, value interface{}) bool {
		plans = append(plans, value.(*Plan))
		return true
	})
	return plans
}

// ClearPlan removes a plan and its related data
func (sm *StateManager) ClearPlan(planID string) {
	sm.plans.Delete(planID)
	sm.options.Delete(planID)
	sm.selections.Delete(planID)
}

// PlanModeWorkflow orchestrates the plan mode workflow
type PlanModeWorkflow struct {
	planner      Planner
	presenter    OptionPresenter
	executor     Executor
	stateManager *StateManager
	controller   ModeController
}

// NewPlanModeWorkflow creates a new plan mode workflow
func NewPlanModeWorkflow(
	planner Planner,
	presenter OptionPresenter,
	executor Executor,
	stateManager *StateManager,
	controller ModeController,
) *PlanModeWorkflow {
	return &PlanModeWorkflow{
		planner:      planner,
		presenter:    presenter,
		executor:     executor,
		stateManager: stateManager,
		controller:   controller,
	}
}

// ExecuteWorkflow executes the full plan mode workflow
func (w *PlanModeWorkflow) ExecuteWorkflow(ctx context.Context, task *Task) (*ExecutionResult, error) {
	// Phase 1: Planning
	if err := w.controller.TransitionTo(ModePlan); err != nil {
		return nil, fmt.Errorf("failed to enter plan mode: %w", err)
	}

	// Generate options
	options, err := w.planner.GenerateOptions(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate options: %w", err)
	}

	// Store options
	planID := uuid.New().String()
	if err := w.stateManager.StoreOptions(planID, options); err != nil {
		return nil, fmt.Errorf("failed to store options: %w", err)
	}

	// Present options to user
	selection, err := w.presenter.Present(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to present options: %w", err)
	}

	// Store selection
	if err := w.stateManager.StoreSelection(planID, selection); err != nil {
		return nil, fmt.Errorf("failed to store selection: %w", err)
	}

	// Get selected option
	var selectedOption *PlanOption
	for _, opt := range options {
		if opt.ID == selection.OptionID {
			selectedOption = opt
			break
		}
	}
	if selectedOption == nil {
		return nil, fmt.Errorf("selected option not found: %s", selection.OptionID)
	}

	// Phase 2: Execution
	if err := w.controller.TransitionTo(ModeAct); err != nil {
		return nil, fmt.Errorf("failed to enter act mode: %w", err)
	}

	// Execute selected plan
	result, err := w.executor.Execute(ctx, selectedOption.Plan)
	if err != nil {
		return nil, fmt.Errorf("failed to execute plan: %w", err)
	}

	// Store execution result
	if err := w.stateManager.StoreExecution(result); err != nil {
		return nil, fmt.Errorf("failed to store execution: %w", err)
	}

	// Return to normal mode
	if err := w.controller.TransitionTo(ModeNormal); err != nil {
		return nil, fmt.Errorf("failed to return to normal mode: %w", err)
	}

	return result, nil
}

// ExecuteWithProgress executes the workflow with progress tracking
func (w *PlanModeWorkflow) ExecuteWithProgress(
	ctx context.Context,
	task *Task,
	progressFn func(*WorkflowProgress),
) (*ExecutionResult, error) {
	progress := &WorkflowProgress{
		Phase:  "Planning",
		Status: "Generating options",
	}
	progressFn(progress)

	// Phase 1: Planning
	if err := w.controller.TransitionTo(ModePlan); err != nil {
		return nil, err
	}

	progress.Status = "Analyzing task"
	progressFn(progress)

	options, err := w.planner.GenerateOptions(ctx, task)
	if err != nil {
		return nil, err
	}

	planID := uuid.New().String()
	w.stateManager.StoreOptions(planID, options)

	progress.Status = "Presenting options"
	progress.OptionsCount = len(options)
	progressFn(progress)

	selection, err := w.presenter.Present(ctx, options)
	if err != nil {
		return nil, err
	}

	w.stateManager.StoreSelection(planID, selection)

	var selectedOption *PlanOption
	for _, opt := range options {
		if opt.ID == selection.OptionID {
			selectedOption = opt
			break
		}
	}

	// Phase 2: Execution
	progress.Phase = "Execution"
	progress.Status = "Preparing execution"
	progressFn(progress)

	if err := w.controller.TransitionTo(ModeAct); err != nil {
		return nil, err
	}

	progress.Status = "Executing plan"
	progress.TotalSteps = len(selectedOption.Plan.Steps)
	progressFn(progress)

	result, err := w.executeWithTracking(ctx, selectedOption.Plan, func(execProgress *ExecutionProgress) {
		progress.CurrentStep = execProgress.CurrentStep
		progress.CompletedSteps = execProgress.CompletedSteps
		progress.Status = execProgress.Status
		progressFn(progress)
	})
	if err != nil {
		return nil, err
	}

	w.stateManager.StoreExecution(result)

	progress.Phase = "Completed"
	progress.Status = "Plan executed successfully"
	progressFn(progress)

	w.controller.TransitionTo(ModeNormal)

	return result, nil
}

// executeWithTracking executes a plan with progress tracking
func (w *PlanModeWorkflow) executeWithTracking(
	ctx context.Context,
	plan *Plan,
	progressFn func(*ExecutionProgress),
) (*ExecutionResult, error) {
	result := &ExecutionResult{
		ID:        uuid.New().String(),
		PlanID:    plan.ID,
		StartTime: time.Now(),
		Metrics:   &ExecutionMetrics{},
	}

	progress := &ExecutionProgress{
		ExecutionID: result.ID,
		TotalSteps:  len(plan.Steps),
		Status:      "Starting execution",
	}

	for i, step := range plan.Steps {
		progress.CurrentStep = i + 1
		progress.Status = fmt.Sprintf("Executing: %s", step.Title)
		progressFn(progress)

		stepResult, err := w.executor.ExecuteStep(ctx, step)
		if err != nil {
			result.Errors = append(result.Errors, err)
			result.Metrics.Errors++
			progress.FailedSteps++
		} else if stepResult.Success {
			progress.CompletedSteps++
			result.Metrics.StepsCompleted++
		}

		result.Steps = append(result.Steps, stepResult)
		step.Result = stepResult
		step.Status = StepCompleted
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0

	return result, nil
}

// ExecuteYOLOMode executes in YOLO mode (auto-select best option and execute)
func (w *PlanModeWorkflow) ExecuteYOLOMode(ctx context.Context, task *Task) (*ExecutionResult, error) {
	// Phase 1: Planning
	if err := w.controller.TransitionTo(ModePlan); err != nil {
		return nil, fmt.Errorf("failed to enter plan mode: %w", err)
	}

	// Generate options
	options, err := w.planner.GenerateOptions(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate options: %w", err)
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no options generated")
	}

	// Auto-select best option (first one, as they're ranked)
	selectedOption := options[0]

	planID := uuid.New().String()
	w.stateManager.StoreOptions(planID, options)
	w.stateManager.StoreSelection(planID, &Selection{
		OptionID:  selectedOption.ID,
		Timestamp: time.Now(),
		Custom:    false,
	})

	// Phase 2: Execution
	if err := w.controller.TransitionTo(ModeAct); err != nil {
		return nil, fmt.Errorf("failed to enter act mode: %w", err)
	}

	// Execute selected plan
	result, err := w.executor.Execute(ctx, selectedOption.Plan)
	if err != nil {
		return nil, fmt.Errorf("failed to execute plan: %w", err)
	}

	// Store execution result
	w.stateManager.StoreExecution(result)

	// Return to normal mode
	w.controller.TransitionTo(ModeNormal)

	return result, nil
}

// PauseExecution pauses a running execution
func (w *PlanModeWorkflow) PauseExecution(executionID string) error {
	if err := w.executor.Pause(executionID); err != nil {
		return err
	}

	return w.controller.TransitionTo(ModePaused)
}

// ResumeExecution resumes a paused execution
func (w *PlanModeWorkflow) ResumeExecution(executionID string) error {
	if err := w.executor.Resume(executionID); err != nil {
		return err
	}

	return w.controller.TransitionTo(ModeAct)
}

// CancelExecution cancels a running execution
func (w *PlanModeWorkflow) CancelExecution(executionID string) error {
	if err := w.executor.Cancel(executionID); err != nil {
		return err
	}

	return w.controller.TransitionTo(ModeNormal)
}

// GetExecutionProgress returns progress for an execution
func (w *PlanModeWorkflow) GetExecutionProgress(executionID string) (*ExecutionProgress, error) {
	return w.executor.GetProgress(executionID)
}

// Config contains plan mode configuration
type Config struct {
	DefaultOptionCount  int
	MaxOptionCount      int
	AutoSelectBest      bool
	ShowComparison      bool
	EnableProgressBar   bool
	ConfidenceThreshold float64
	MaxPlanComplexity   Complexity
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultOptionCount:  3,
		MaxOptionCount:      5,
		AutoSelectBest:      false,
		ShowComparison:      true,
		EnableProgressBar:   true,
		ConfidenceThreshold: 0.7,
		MaxPlanComplexity:   ComplexityHigh,
	}
}

// PlanMode is the main entry point for plan mode functionality
type PlanMode struct {
	workflow *PlanModeWorkflow
	config   *Config
}

// NewPlanMode creates a new plan mode instance
func NewPlanMode(workflow *PlanModeWorkflow, config *Config) *PlanMode {
	if config == nil {
		config = DefaultConfig()
	}

	return &PlanMode{
		workflow: workflow,
		config:   config,
	}
}

// Run executes plan mode with the given task
func (pm *PlanMode) Run(ctx context.Context, task *Task) (*ExecutionResult, error) {
	if pm.config.AutoSelectBest {
		return pm.workflow.ExecuteYOLOMode(ctx, task)
	}

	return pm.workflow.ExecuteWorkflow(ctx, task)
}

// RunWithProgress executes plan mode with progress tracking
func (pm *PlanMode) RunWithProgress(
	ctx context.Context,
	task *Task,
	progressFn func(*WorkflowProgress),
) (*ExecutionResult, error) {
	return pm.workflow.ExecuteWithProgress(ctx, task, progressFn)
}

// Pause pauses execution
func (pm *PlanMode) Pause(executionID string) error {
	return pm.workflow.PauseExecution(executionID)
}

// Resume resumes execution
func (pm *PlanMode) Resume(executionID string) error {
	return pm.workflow.ResumeExecution(executionID)
}

// Cancel cancels execution
func (pm *PlanMode) Cancel(executionID string) error {
	return pm.workflow.CancelExecution(executionID)
}

// GetProgress returns execution progress
func (pm *PlanMode) GetProgress(executionID string) (*ExecutionProgress, error) {
	return pm.workflow.GetExecutionProgress(executionID)
}
