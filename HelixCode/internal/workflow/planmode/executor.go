package planmode

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ExecutionResult contains the result of plan execution
type ExecutionResult struct {
	ID           string
	PlanID       string
	Success      bool
	Steps        []*StepResult
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	FilesChanged []string
	Errors       []error
	Metrics      *ExecutionMetrics
}

// ExecutionProgress tracks execution progress
type ExecutionProgress struct {
	ExecutionID        string
	CurrentStep        int
	TotalSteps         int
	CompletedSteps     int
	FailedSteps        int
	SkippedSteps       int
	ElapsedTime        time.Duration
	EstimatedRemaining time.Duration
	Status             string
}

// ExecutionMetrics contains execution metrics
type ExecutionMetrics struct {
	StepsCompleted   int
	StepsFailed      int
	FilesModified    int
	FilesCreated     int
	FilesDeleted     int
	LinesChanged     int
	CommandsExecuted int
	Errors           int
	Warnings         int
}

// Executor executes plans
type Executor interface {
	// Execute executes a plan
	Execute(ctx context.Context, plan *Plan) (*ExecutionResult, error)

	// ExecuteStep executes a single step
	ExecuteStep(ctx context.Context, step *PlanStep) (*StepResult, error)

	// Pause pauses execution
	Pause(executionID string) error

	// Resume resumes execution
	Resume(executionID string) error

	// Cancel cancels execution
	Cancel(executionID string) error

	// GetProgress returns execution progress
	GetProgress(executionID string) (*ExecutionProgress, error)
}

// DefaultExecutor implements Executor
type DefaultExecutor struct {
	executions      sync.Map // map[string]*activeExecution
	progressTracker *ProgressTracker
	workspaceRoot   string
}

// activeExecution tracks an active execution
type activeExecution struct {
	result   *ExecutionResult
	progress *ExecutionProgress
	ctx      context.Context
	cancel   context.CancelFunc
	paused   bool
	mu       sync.RWMutex
}

// NewDefaultExecutor creates a new default executor
func NewDefaultExecutor(workspaceRoot string) *DefaultExecutor {
	return &DefaultExecutor{
		progressTracker: NewProgressTracker(),
		workspaceRoot:   workspaceRoot,
	}
}

// Execute executes a plan
func (e *DefaultExecutor) Execute(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	executionID := uuid.New().String()
	execCtx, cancel := context.WithCancel(ctx)

	result := &ExecutionResult{
		ID:        executionID,
		PlanID:    plan.ID,
		StartTime: time.Now(),
		Steps:     make([]*StepResult, 0),
		Errors:    make([]error, 0),
		Metrics:   &ExecutionMetrics{},
	}

	progress := &ExecutionProgress{
		ExecutionID: executionID,
		TotalSteps:  len(plan.Steps),
		Status:      "Starting execution",
	}

	active := &activeExecution{
		result:   result,
		progress: progress,
		ctx:      execCtx,
		cancel:   cancel,
		paused:   false,
	}

	e.executions.Store(executionID, active)
	defer e.executions.Delete(executionID)

	// Execute steps
	for i, step := range plan.Steps {
		// Check for cancellation
		select {
		case <-execCtx.Done():
			result.Success = false
			result.Errors = append(result.Errors, fmt.Errorf("execution cancelled"))
			e.finalizeExecution(result)
			return result, fmt.Errorf("execution cancelled")
		default:
		}

		// Check if paused
		active.mu.RLock()
		for active.paused {
			active.mu.RUnlock()
			time.Sleep(100 * time.Millisecond)
			active.mu.RLock()
		}
		active.mu.RUnlock()

		// Check dependencies
		if !e.areDependenciesSatisfied(plan, step, result) {
			step.Status = StepSkipped
			progress.SkippedSteps++
			progress.Status = fmt.Sprintf("Skipped: %s (dependencies not met)", step.Title)
			continue
		}

		// Update progress
		progress.CurrentStep = i + 1
		progress.Status = fmt.Sprintf("Executing: %s", step.Title)
		step.Status = StepInProgress

		// Execute step
		stepResult, err := e.ExecuteStep(execCtx, step)
		if err != nil {
			result.Errors = append(result.Errors, err)
			result.Metrics.Errors++
			step.Status = StepFailed
			progress.FailedSteps++
		} else if stepResult.Success {
			step.Status = StepCompleted
			progress.CompletedSteps++
			result.Metrics.StepsCompleted++
		} else {
			step.Status = StepFailed
			progress.FailedSteps++
			result.Metrics.StepsFailed++
		}

		step.Result = stepResult
		result.Steps = append(result.Steps, stepResult)

		// Track files changed
		if stepResult.FilesChanged != nil {
			result.FilesChanged = append(result.FilesChanged, stepResult.FilesChanged...)
			result.Metrics.FilesModified += len(stepResult.FilesChanged)
		}

		// Update elapsed time
		progress.ElapsedTime = time.Since(result.StartTime)

		// Estimate remaining time
		if progress.CompletedSteps > 0 {
			avgTimePerStep := progress.ElapsedTime / time.Duration(progress.CompletedSteps)
			remainingSteps := progress.TotalSteps - progress.CurrentStep
			progress.EstimatedRemaining = avgTimePerStep * time.Duration(remainingSteps)
		}
	}

	e.finalizeExecution(result)
	return result, nil
}

// ExecuteStep executes a single step
func (e *DefaultExecutor) ExecuteStep(ctx context.Context, step *PlanStep) (*StepResult, error) {
	startTime := time.Now()
	result := &StepResult{
		Success: false,
		Metrics: make(map[string]interface{}),
	}

	switch step.Type {
	case StepTypeFileOperation:
		err := e.executeFileOperation(ctx, step, result)
		result.Success = err == nil
		result.Error = err

	case StepTypeShellCommand:
		err := e.executeShellCommand(ctx, step, result)
		result.Success = err == nil
		result.Error = err

	case StepTypeCodeGeneration:
		err := e.executeCodeGeneration(ctx, step, result)
		result.Success = err == nil
		result.Error = err

	case StepTypeCodeAnalysis:
		err := e.executeCodeAnalysis(ctx, step, result)
		result.Success = err == nil
		result.Error = err

	case StepTypeValidation:
		err := e.executeValidation(ctx, step, result)
		result.Success = err == nil
		result.Error = err

	case StepTypeTesting:
		err := e.executeTesting(ctx, step, result)
		result.Success = err == nil
		result.Error = err

	default:
		result.Error = fmt.Errorf("unknown step type: %s", step.Type)
	}

	result.Duration = time.Since(startTime)
	return result, result.Error
}

// Pause pauses execution
func (e *DefaultExecutor) Pause(executionID string) error {
	val, ok := e.executions.Load(executionID)
	if !ok {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	active := val.(*activeExecution)
	active.mu.Lock()
	defer active.mu.Unlock()

	if active.paused {
		return fmt.Errorf("execution already paused")
	}

	active.paused = true
	active.progress.Status = "Paused"
	return nil
}

// Resume resumes execution
func (e *DefaultExecutor) Resume(executionID string) error {
	val, ok := e.executions.Load(executionID)
	if !ok {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	active := val.(*activeExecution)
	active.mu.Lock()
	defer active.mu.Unlock()

	if !active.paused {
		return fmt.Errorf("execution not paused")
	}

	active.paused = false
	active.progress.Status = "Resumed"
	return nil
}

// Cancel cancels execution
func (e *DefaultExecutor) Cancel(executionID string) error {
	val, ok := e.executions.Load(executionID)
	if !ok {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	active := val.(*activeExecution)
	active.cancel()
	active.progress.Status = "Cancelled"
	return nil
}

// GetProgress returns execution progress
func (e *DefaultExecutor) GetProgress(executionID string) (*ExecutionProgress, error) {
	val, ok := e.executions.Load(executionID)
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	active := val.(*activeExecution)
	active.mu.RLock()
	defer active.mu.RUnlock()

	// Return a copy
	progress := &ExecutionProgress{
		ExecutionID:        active.progress.ExecutionID,
		CurrentStep:        active.progress.CurrentStep,
		TotalSteps:         active.progress.TotalSteps,
		CompletedSteps:     active.progress.CompletedSteps,
		FailedSteps:        active.progress.FailedSteps,
		SkippedSteps:       active.progress.SkippedSteps,
		ElapsedTime:        active.progress.ElapsedTime,
		EstimatedRemaining: active.progress.EstimatedRemaining,
		Status:             active.progress.Status,
	}

	return progress, nil
}

// areDependenciesSatisfied checks if step dependencies are satisfied
func (e *DefaultExecutor) areDependenciesSatisfied(plan *Plan, step *PlanStep, result *ExecutionResult) bool {
	if len(step.Dependencies) == 0 {
		return true
	}

	for _, depID := range step.Dependencies {
		found := false
		for _, s := range plan.Steps {
			if s.ID == depID {
				if s.Status != StepCompleted {
					return false
				}
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// finalizeExecution finalizes an execution result
func (e *DefaultExecutor) finalizeExecution(result *ExecutionResult) {
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0 && result.Metrics.StepsFailed == 0
}

// Step execution implementations

func (e *DefaultExecutor) executeFileOperation(ctx context.Context, step *PlanStep, result *StepResult) error {
	// Placeholder implementation
	result.Output = fmt.Sprintf("File operation: %s", step.Description)
	result.FilesChanged = []string{step.Action}
	return nil
}

func (e *DefaultExecutor) executeShellCommand(ctx context.Context, step *PlanStep, result *StepResult) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", step.Action)
	cmd.Dir = e.workspaceRoot

	output, err := cmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func (e *DefaultExecutor) executeCodeGeneration(ctx context.Context, step *PlanStep, result *StepResult) error {
	// Placeholder - would integrate with LLM for code generation
	result.Output = fmt.Sprintf("Code generation: %s", step.Description)
	return nil
}

func (e *DefaultExecutor) executeCodeAnalysis(ctx context.Context, step *PlanStep, result *StepResult) error {
	// Placeholder - would integrate with code analysis tools
	result.Output = fmt.Sprintf("Code analysis: %s", step.Description)
	return nil
}

func (e *DefaultExecutor) executeValidation(ctx context.Context, step *PlanStep, result *StepResult) error {
	// Placeholder - would run validation checks
	result.Output = fmt.Sprintf("Validation: %s", step.Description)
	return nil
}

func (e *DefaultExecutor) executeTesting(ctx context.Context, step *PlanStep, result *StepResult) error {
	// Placeholder - would run tests
	result.Output = fmt.Sprintf("Testing: %s", step.Description)
	return nil
}

// ProgressTracker tracks execution progress
type ProgressTracker struct {
	mu        sync.RWMutex
	callbacks map[string][]ProgressCallback
}

// ProgressCallback is called when progress updates
type ProgressCallback func(*ExecutionProgress)

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		callbacks: make(map[string][]ProgressCallback),
	}
}

// RegisterCallback registers a progress callback
func (pt *ProgressTracker) RegisterCallback(executionID string, callback ProgressCallback) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if _, exists := pt.callbacks[executionID]; !exists {
		pt.callbacks[executionID] = make([]ProgressCallback, 0)
	}

	pt.callbacks[executionID] = append(pt.callbacks[executionID], callback)
}

// NotifyProgress notifies all callbacks for an execution
func (pt *ProgressTracker) NotifyProgress(progress *ExecutionProgress) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if callbacks, exists := pt.callbacks[progress.ExecutionID]; exists {
		for _, callback := range callbacks {
			callback(progress)
		}
	}
}

// ClearCallbacks clears callbacks for an execution
func (pt *ProgressTracker) ClearCallbacks(executionID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	delete(pt.callbacks, executionID)
}
