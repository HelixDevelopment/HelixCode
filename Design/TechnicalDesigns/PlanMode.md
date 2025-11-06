# PlanMode - Technical Design Document

## Overview

PlanMode implements a two-phase workflow where HelixCode first generates and presents multiple implementation options to the user, then executes the selected approach. This design is inspired by Cline's Plan Mode, enhanced with better option presentation, state management, and execution tracking.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         PlanMode                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   ModeController                         │  │
│  │  (Manages mode transitions: Normal ↔ Plan ↔ Act)       │  │
│  └──────────────┬───────────────────────────┬───────────────┘  │
│                 │                           │                   │
│         ┌───────▼────────┐         ┌────────▼────────┐         │
│         │                │         │                 │         │
│    ┌────┴────┐      ┌────┴────┐   │    Executor     │         │
│    │ Planner │      │ Option  │   │                 │         │
│    │         │──────│Presenter│   │                 │         │
│    └────┬────┘      └────┬────┘   └────────┬────────┘         │
│         │                │                  │                   │
│    ┌────▼────────────────▼──────────────────▼────────┐         │
│    │              StateManager                        │         │
│    │  (Plan, Options, Selection, Execution State)    │         │
│    └──────────────────────────────────────────────────┘         │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐          │
│  │            ProgressTracker                       │          │
│  │  (Tracks execution progress and status)         │          │
│  └──────────────────────────────────────────────────┘          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
   FileSystem       ShellExecution      LLM Provider
```

## Core Interfaces

### ModeController Interface

```go
// ModeController manages operational modes
type ModeController interface {
    // GetMode returns the current mode
    GetMode() Mode

    // SetMode sets the current mode
    SetMode(mode Mode) error

    // CanTransition checks if a mode transition is allowed
    CanTransition(from, to Mode) bool

    // TransitionTo transitions to a new mode
    TransitionTo(mode Mode) error

    // RegisterCallback registers a callback for mode changes
    RegisterCallback(fn ModeChangeCallback)
}

// Mode represents the operational mode
type Mode int

const (
    ModeNormal Mode = iota // Normal operation
    ModePlan               // Planning phase
    ModeAct                // Execution phase
    ModePaused             // Paused execution
)

func (m Mode) String() string {
    return [...]string{"Normal", "Plan", "Act", "Paused"}[m]
}

// ModeChangeCallback is called when mode changes
type ModeChangeCallback func(from, to Mode, state *ModeState)

// ModeState contains state information for a mode
type ModeState struct {
    Mode        Mode
    PlanID      string
    OptionID    string
    ExecutionID string
    Metadata    map[string]interface{}
}
```

### Planner Interface

```go
// Planner generates implementation plans
type Planner interface {
    // GeneratePlan generates a plan for a task
    GeneratePlan(ctx context.Context, task *Task) (*Plan, error)

    // GenerateOptions generates multiple implementation options
    GenerateOptions(ctx context.Context, task *Task) ([]*PlanOption, error)

    // RefinePlan refines a plan based on feedback
    RefinePlan(ctx context.Context, plan *Plan, feedback string) (*Plan, error)

    // ValidatePlan validates a plan
    ValidatePlan(ctx context.Context, plan *Plan) (*ValidationResult, error)
}

// Task represents a user task to be planned
type Task struct {
    ID          string
    Description string
    Context     *TaskContext
    Requirements []string
    Constraints []string
    Priority    Priority
    Deadline    *time.Time
}

// TaskContext provides context for planning
type TaskContext struct {
    WorkspaceRoot string
    CurrentFiles  []string
    RecentChanges []string
    Dependencies  []string
    Environment   map[string]string
}

// Plan represents an implementation plan
type Plan struct {
    ID          string
    TaskID      string
    Title       string
    Description string
    Steps       []*PlanStep
    Resources   []string
    Risks       []Risk
    Estimates   Estimates
    CreatedAt   time.Time
    UpdatedAt   time.Time
    Version     int
    Status      PlanStatus
}

// PlanStep represents a single step in a plan
type PlanStep struct {
    ID           string
    Order        int
    Title        string
    Description  string
    Type         StepType
    Action       string
    Dependencies []string
    Estimated    time.Duration
    Status       StepStatus
    Result       *StepResult
}

// StepType defines the type of plan step
type StepType int

const (
    StepTypeFileOperation StepType = iota
    StepTypeShellCommand
    StepTypeCodeGeneration
    StepTypeCodeAnalysis
    StepTypeValidation
    StepTypeTesting
)

// StepStatus represents the status of a step
type StepStatus int

const (
    StepPending StepStatus = iota
    StepInProgress
    StepCompleted
    StepFailed
    StepSkipped
)

// StepResult contains the result of executing a step
type StepResult struct {
    Success     bool
    Output      string
    Error       error
    Duration    time.Duration
    FilesChanged []string
    Metrics     map[string]interface{}
}

// Risk represents a potential risk in the plan
type Risk struct {
    Description string
    Impact      RiskImpact
    Likelihood  RiskLikelihood
    Mitigation  string
}

// RiskImpact defines risk impact levels
type RiskImpact int

const (
    ImpactLow RiskImpact = iota
    ImpactMedium
    ImpactHigh
    ImpactCritical
)

// RiskLikelihood defines risk likelihood levels
type RiskLikelihood int

const (
    LikelihoodLow RiskLikelihood = iota
    LikelihoodMedium
    LikelihoodHigh
)

// Estimates contains time and resource estimates
type Estimates struct {
    Duration    time.Duration
    Complexity  Complexity
    Confidence  float64 // 0-1
}

// Complexity defines complexity levels
type Complexity int

const (
    ComplexityLow Complexity = iota
    ComplexityMedium
    ComplexityHigh
    ComplexityVeryHigh
)

// PlanStatus represents the status of a plan
type PlanStatus int

const (
    PlanDraft PlanStatus = iota
    PlanReady
    PlanInProgress
    PlanCompleted
    PlanFailed
    PlanCancelled
)

// Priority defines task priority
type Priority int

const (
    PriorityLow Priority = iota
    PriorityMedium
    PriorityHigh
    PriorityCritical
)
```

### OptionPresenter Interface

```go
// OptionPresenter presents plan options to the user
type OptionPresenter interface {
    // Present presents options to the user
    Present(ctx context.Context, options []*PlanOption) (*Selection, error)

    // CompareOptions compares multiple options
    CompareOptions(options []*PlanOption) (*Comparison, error)

    // RankOptions ranks options by various criteria
    RankOptions(options []*PlanOption, criteria []RankCriterion) ([]*RankedOption, error)
}

// PlanOption represents an implementation option
type PlanOption struct {
    ID          string
    Title       string
    Description string
    Plan        *Plan
    Pros        []string
    Cons        []string
    Rank        int
    Score       float64
    Recommended bool
}

// Selection represents a user's option selection
type Selection struct {
    OptionID  string
    Timestamp time.Time
    Feedback  string
    Custom    bool // If user provided custom modifications
}

// Comparison contains a comparison of options
type Comparison struct {
    Options     []*PlanOption
    Criteria    []string
    Matrix      [][]ComparisonCell
    Summary     string
}

// ComparisonCell represents a single comparison cell
type ComparisonCell struct {
    OptionID string
    Criterion string
    Value    string
    Score    float64
}

// RankCriterion defines criteria for ranking options
type RankCriterion struct {
    Name   string
    Weight float64
    Type   CriterionType
}

// CriterionType defines the type of ranking criterion
type CriterionType int

const (
    CriterionSpeed CriterionType = iota
    CriterionSafety
    CriterionSimplicity
    CriterionMaintainability
    CriterionPerformance
    CriterionCost
)

// RankedOption is an option with ranking information
type RankedOption struct {
    Option *PlanOption
    Rank   int
    Score  float64
    Scores map[string]float64 // Scores per criterion
}
```

### Executor Interface

```go
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
    ExecutionID     string
    CurrentStep     int
    TotalSteps      int
    CompletedSteps  int
    FailedSteps     int
    SkippedSteps    int
    ElapsedTime     time.Duration
    EstimatedRemaining time.Duration
    Status          string
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
```

## State Management

### StateManager Implementation

```go
// StateManager manages plan mode state
type StateManager struct {
    currentMode    Mode
    plans          sync.Map // map[string]*Plan
    options        sync.Map // map[string][]*PlanOption
    selections     sync.Map // map[string]*Selection
    executions     sync.Map // map[string]*ExecutionResult
    mu             sync.RWMutex
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
```

## Workflow Implementation

### Two-Phase Workflow

```go
// PlanModeWorkflow orchestrates the plan mode workflow
type PlanModeWorkflow struct {
    planner       Planner
    presenter     OptionPresenter
    executor      Executor
    stateManager  *StateManager
    controller    ModeController
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
        Phase: "Planning",
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

// WorkflowProgress tracks workflow progress
type WorkflowProgress struct {
    Phase          string
    Status         string
    OptionsCount   int
    TotalSteps     int
    CurrentStep    int
    CompletedSteps int
}
```

## Planner Implementation

### LLM-Based Planner

```go
// LLMPlanner implements Planner using an LLM
type LLMPlanner struct {
    llm          LLMProvider
    promptBuilder *PromptBuilder
    validator    *PlanValidator
}

// GenerateOptions generates multiple implementation options
func (p *LLMPlanner) GenerateOptions(ctx context.Context, task *Task) ([]*PlanOption, error) {
    // Build prompt for option generation
    prompt := p.promptBuilder.BuildOptionPrompt(task)

    // Call LLM to generate options
    response, err := p.llm.Complete(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("failed to generate options: %w", err)
    }

    // Parse response into options
    options, err := p.parseOptions(response)
    if err != nil {
        return nil, fmt.Errorf("failed to parse options: %w", err)
    }

    // Validate and rank options
    for i, option := range options {
        validation, err := p.validator.ValidatePlan(ctx, option.Plan)
        if err != nil {
            return nil, fmt.Errorf("failed to validate option %d: %w", i, err)
        }

        if !validation.Valid {
            return nil, fmt.Errorf("option %d is invalid: %s", i, validation.Message)
        }

        option.Score = p.scoreOption(option)
    }

    // Sort by score
    sort.Slice(options, func(i, j int) bool {
        return options[i].Score > options[j].Score
    })

    // Set rank and recommended
    for i, option := range options {
        option.Rank = i + 1
        option.Recommended = i == 0
    }

    return options, nil
}

// parseOptions parses LLM response into options
func (p *LLMPlanner) parseOptions(response string) ([]*PlanOption, error) {
    // This would parse the structured LLM response
    // For example, JSON format:
    // {
    //   "options": [
    //     {
    //       "title": "Option 1",
    //       "description": "...",
    //       "plan": { ... },
    //       "pros": [...],
    //       "cons": [...]
    //     }
    //   ]
    // }

    var parsed struct {
        Options []struct {
            Title       string   `json:"title"`
            Description string   `json:"description"`
            Plan        *Plan    `json:"plan"`
            Pros        []string `json:"pros"`
            Cons        []string `json:"cons"`
        } `json:"options"`
    }

    if err := json.Unmarshal([]byte(response), &parsed); err != nil {
        return nil, err
    }

    options := make([]*PlanOption, len(parsed.Options))
    for i, opt := range parsed.Options {
        options[i] = &PlanOption{
            ID:          uuid.New().String(),
            Title:       opt.Title,
            Description: opt.Description,
            Plan:        opt.Plan,
            Pros:        opt.Pros,
            Cons:        opt.Cons,
        }
    }

    return options, nil
}

// scoreOption scores an option
func (p *LLMPlanner) scoreOption(option *PlanOption) float64 {
    score := 0.0

    // Score based on complexity (simpler is better)
    complexityScore := map[Complexity]float64{
        ComplexityLow:      1.0,
        ComplexityMedium:   0.75,
        ComplexityHigh:     0.5,
        ComplexityVeryHigh: 0.25,
    }
    score += complexityScore[option.Plan.Estimates.Complexity] * 30

    // Score based on confidence
    score += option.Plan.Estimates.Confidence * 30

    // Score based on pros vs cons
    prosScore := float64(len(option.Pros)) * 5
    consScore := float64(len(option.Cons)) * 3
    score += (prosScore - consScore)

    // Score based on risks
    riskScore := 0.0
    for _, risk := range option.Plan.Risks {
        impactWeight := map[RiskImpact]float64{
            ImpactLow:      0.25,
            ImpactMedium:   0.5,
            ImpactHigh:     0.75,
            ImpactCritical: 1.0,
        }
        likelihoodWeight := map[RiskLikelihood]float64{
            LikelihoodLow:    0.25,
            LikelihoodMedium: 0.5,
            LikelihoodHigh:   1.0,
        }
        riskScore += impactWeight[risk.Impact] * likelihoodWeight[risk.Likelihood] * 5
    }
    score -= riskScore

    // Normalize to 0-100
    if score < 0 {
        score = 0
    } else if score > 100 {
        score = 100
    }

    return score
}
```

### PromptBuilder

```go
// PromptBuilder builds prompts for the LLM
type PromptBuilder struct {
    templates map[string]string
}

// BuildOptionPrompt builds a prompt for generating options
func (pb *PromptBuilder) BuildOptionPrompt(task *Task) string {
    template := `You are tasked with generating multiple implementation options for the following task:

Task: {{.Description}}

Context:
- Workspace: {{.Context.WorkspaceRoot}}
- Current Files: {{join .Context.CurrentFiles ", "}}
{{if .Requirements}}
Requirements:
{{range .Requirements}}
- {{.}}
{{end}}
{{end}}
{{if .Constraints}}
Constraints:
{{range .Constraints}}
- {{.}}
{{end}}
{{end}}

Generate 3-4 different implementation options, each with:
1. A clear title and description
2. A detailed step-by-step plan
3. Pros and cons
4. Risk assessment
5. Time estimates
6. Complexity rating

Format your response as JSON following this structure:
{
  "options": [
    {
      "title": "Option title",
      "description": "Detailed description",
      "plan": {
        "steps": [
          {
            "title": "Step title",
            "description": "Step description",
            "type": "file_operation|shell_command|code_generation",
            "action": "Specific action to take"
          }
        ],
        "risks": [
          {
            "description": "Risk description",
            "impact": "low|medium|high|critical",
            "likelihood": "low|medium|high",
            "mitigation": "How to mitigate"
          }
        ],
        "estimates": {
          "duration": "estimated duration in minutes",
          "complexity": "low|medium|high|very_high",
          "confidence": 0.0-1.0
        }
      },
      "pros": ["Pro 1", "Pro 2"],
      "cons": ["Con 1", "Con 2"]
    }
  ]
}
`

    // Execute template with task data
    // This is simplified - real implementation would use text/template
    return template
}
```

## UI/Presentation Layer

### Option Presenter Implementation

```go
// CLIOptionPresenter presents options via CLI
type CLIOptionPresenter struct {
    output io.Writer
    input  io.Reader
}

// Present presents options to the user
func (p *CLIOptionPresenter) Present(ctx context.Context, options []*PlanOption) (*Selection, error) {
    fmt.Fprintln(p.output, "\n=== Implementation Options ===\n")

    for i, opt := range options {
        fmt.Fprintf(p.output, "Option %d: %s", i+1, opt.Title)
        if opt.Recommended {
            fmt.Fprint(p.output, " [RECOMMENDED]")
        }
        fmt.Fprintln(p.output)
        fmt.Fprintf(p.output, "Score: %.1f/100\n", opt.Score)
        fmt.Fprintf(p.output, "Description: %s\n", opt.Description)
        fmt.Fprintln(p.output)

        fmt.Fprintln(p.output, "Pros:")
        for _, pro := range opt.Pros {
            fmt.Fprintf(p.output, "  + %s\n", pro)
        }
        fmt.Fprintln(p.output)

        fmt.Fprintln(p.output, "Cons:")
        for _, con := range opt.Cons {
            fmt.Fprintf(p.output, "  - %s\n", con)
        }
        fmt.Fprintln(p.output)

        fmt.Fprintf(p.output, "Estimated Duration: %s\n", opt.Plan.Estimates.Duration)
        fmt.Fprintf(p.output, "Complexity: %s\n", opt.Plan.Estimates.Complexity)
        fmt.Fprintf(p.output, "Confidence: %.0f%%\n", opt.Plan.Estimates.Confidence*100)
        fmt.Fprintln(p.output, "\n---\n")
    }

    // Prompt for selection
    fmt.Fprint(p.output, "Select an option (1-", len(options), "): ")

    var choice int
    fmt.Fscanln(p.input, &choice)

    if choice < 1 || choice > len(options) {
        return nil, fmt.Errorf("invalid choice: %d", choice)
    }

    selected := options[choice-1]

    return &Selection{
        OptionID:  selected.ID,
        Timestamp: time.Now(),
    }, nil
}

// CompareOptions compares multiple options
func (p *CLIOptionPresenter) CompareOptions(options []*PlanOption) (*Comparison, error) {
    criteria := []string{
        "Complexity",
        "Duration",
        "Confidence",
        "Risk Level",
        "Score",
    }

    matrix := make([][]ComparisonCell, len(options))
    for i, opt := range options {
        matrix[i] = make([]ComparisonCell, len(criteria))

        matrix[i][0] = ComparisonCell{
            OptionID:  opt.ID,
            Criterion: "Complexity",
            Value:     opt.Plan.Estimates.Complexity.String(),
        }

        matrix[i][1] = ComparisonCell{
            OptionID:  opt.ID,
            Criterion: "Duration",
            Value:     opt.Plan.Estimates.Duration.String(),
        }

        matrix[i][2] = ComparisonCell{
            OptionID:  opt.ID,
            Criterion: "Confidence",
            Value:     fmt.Sprintf("%.0f%%", opt.Plan.Estimates.Confidence*100),
            Score:     opt.Plan.Estimates.Confidence,
        }

        riskLevel := p.calculateRiskLevel(opt.Plan.Risks)
        matrix[i][3] = ComparisonCell{
            OptionID:  opt.ID,
            Criterion: "Risk Level",
            Value:     riskLevel,
        }

        matrix[i][4] = ComparisonCell{
            OptionID:  opt.ID,
            Criterion: "Score",
            Value:     fmt.Sprintf("%.1f", opt.Score),
            Score:     opt.Score,
        }
    }

    return &Comparison{
        Options:  options,
        Criteria: criteria,
        Matrix:   matrix,
    }, nil
}

// calculateRiskLevel calculates overall risk level
func (p *CLIOptionPresenter) calculateRiskLevel(risks []Risk) string {
    if len(risks) == 0 {
        return "Low"
    }

    maxImpact := ImpactLow
    for _, risk := range risks {
        if risk.Impact > maxImpact && risk.Likelihood >= LikelihoodMedium {
            maxImpact = risk.Impact
        }
    }

    switch maxImpact {
    case ImpactCritical:
        return "Critical"
    case ImpactHigh:
        return "High"
    case ImpactMedium:
        return "Medium"
    default:
        return "Low"
    }
}
```

## Testing Strategy

### Unit Tests

```go
// TestPlanGeneration tests plan generation
func TestPlanGeneration(t *testing.T) {
    planner := NewLLMPlanner(mockLLM, promptBuilder, validator)

    task := &Task{
        ID:          "task-1",
        Description: "Add user authentication to the application",
        Context: &TaskContext{
            WorkspaceRoot: "/workspace",
            CurrentFiles:  []string{"main.go", "user.go"},
        },
        Requirements: []string{
            "Use JWT tokens",
            "Support email/password login",
        },
    }

    options, err := planner.GenerateOptions(context.Background(), task)
    require.NoError(t, err)
    assert.GreaterOrEqual(t, len(options), 2)
    assert.LessOrEqual(t, len(options), 4)

    for _, opt := range options {
        assert.NotEmpty(t, opt.Title)
        assert.NotEmpty(t, opt.Description)
        assert.NotNil(t, opt.Plan)
        assert.NotEmpty(t, opt.Plan.Steps)
        assert.Greater(t, opt.Score, 0.0)
    }

    // First option should be recommended
    assert.True(t, options[0].Recommended)
}

// TestStateManagement tests state management
func TestStateManagement(t *testing.T) {
    sm := NewStateManager()

    plan := &Plan{
        ID:          "plan-1",
        Title:       "Test Plan",
        Description: "Test description",
    }

    // Store plan
    err := sm.StorePlan(plan)
    require.NoError(t, err)

    // Retrieve plan
    retrieved, err := sm.GetPlan("plan-1")
    require.NoError(t, err)
    assert.Equal(t, plan.ID, retrieved.ID)
    assert.Equal(t, plan.Title, retrieved.Title)

    // Store options
    options := []*PlanOption{
        {ID: "opt-1", Title: "Option 1"},
        {ID: "opt-2", Title: "Option 2"},
    }
    err = sm.StoreOptions("plan-1", options)
    require.NoError(t, err)

    // Retrieve options
    retrievedOpts, err := sm.GetOptions("plan-1")
    require.NoError(t, err)
    assert.Equal(t, 2, len(retrievedOpts))
}

// TestModeTransitions tests mode transitions
func TestModeTransitions(t *testing.T) {
    controller := NewModeController()

    // Initial mode should be Normal
    assert.Equal(t, ModeNormal, controller.GetMode())

    // Transition to Plan
    err := controller.TransitionTo(ModePlan)
    require.NoError(t, err)
    assert.Equal(t, ModePlan, controller.GetMode())

    // Transition to Act
    err = controller.TransitionTo(ModeAct)
    require.NoError(t, err)
    assert.Equal(t, ModeAct, controller.GetMode())

    // Can pause from Act
    err = controller.TransitionTo(ModePaused)
    require.NoError(t, err)
    assert.Equal(t, ModePaused, controller.GetMode())

    // Can resume to Act
    err = controller.TransitionTo(ModeAct)
    require.NoError(t, err)
    assert.Equal(t, ModeAct, controller.GetMode())

    // Return to Normal
    err = controller.TransitionTo(ModeNormal)
    require.NoError(t, err)
    assert.Equal(t, ModeNormal, controller.GetMode())
}
```

### Integration Tests

```go
// TestPlanModeWorkflow tests the full workflow
func TestPlanModeWorkflow(t *testing.T) {
    planner := NewLLMPlanner(mockLLM, promptBuilder, validator)
    presenter := &MockPresenter{
        selectedOption: 0, // Select first option
    }
    executor := NewDefaultExecutor()
    stateManager := NewStateManager()
    controller := NewModeController()

    workflow := NewPlanModeWorkflow(
        planner,
        presenter,
        executor,
        stateManager,
        controller,
    )

    task := &Task{
        ID:          "task-1",
        Description: "Add logging to the application",
        Context: &TaskContext{
            WorkspaceRoot: "/workspace",
        },
    }

    result, err := workflow.ExecuteWorkflow(context.Background(), task)
    require.NoError(t, err)
    assert.NotNil(t, result)
    assert.True(t, result.Success)
    assert.Greater(t, len(result.Steps), 0)

    // Verify mode returned to Normal
    assert.Equal(t, ModeNormal, controller.GetMode())
}
```

## Configuration

```go
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
```

## References

### Cline's Plan Mode

- **Location**: `src/core/plan/PlanMode.ts`
- **Features**:
  - Two-phase workflow (plan → act)
  - Option generation and presentation
  - User selection interface
  - Progress tracking during execution

## Future Enhancements

1. **Interactive Plan Editing**: Allow users to modify plans before execution
2. **Plan Templates**: Pre-defined plan templates for common tasks
3. **Plan Versioning**: Track plan versions and iterations
4. **Rollback Support**: Ability to rollback failed executions
5. **Cost Estimation**: Estimate costs (time, resources) for each option
6. **Parallel Execution**: Execute independent steps in parallel
7. **Conditional Steps**: Steps that execute based on conditions
8. **Plan Visualization**: Visual representation of plans (flowcharts, diagrams)
9. **Collaboration**: Multi-user plan review and approval
10. **Learning**: Learn from past executions to improve future plans
