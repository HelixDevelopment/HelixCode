package planmode

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/google/uuid"
)

// Task represents a user task to be planned
type Task struct {
	ID           string
	Description  string
	Context      *TaskContext
	Requirements []string
	Constraints  []string
	Priority     Priority
	Deadline     *time.Time
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

// String returns string representation of step type
func (st StepType) String() string {
	return [...]string{"FileOperation", "ShellCommand", "CodeGeneration", "CodeAnalysis", "Validation", "Testing"}[st]
}

// StepStatus represents the status of a step
type StepStatus int

const (
	StepPending StepStatus = iota
	StepInProgress
	StepCompleted
	StepFailed
	StepSkipped
)

// String returns string representation of step status
func (ss StepStatus) String() string {
	return [...]string{"Pending", "InProgress", "Completed", "Failed", "Skipped"}[ss]
}

// StepResult contains the result of executing a step
type StepResult struct {
	Success      bool
	Output       string
	Error        error
	Duration     time.Duration
	FilesChanged []string
	Metrics      map[string]interface{}
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

// String returns string representation of risk impact
func (ri RiskImpact) String() string {
	return [...]string{"Low", "Medium", "High", "Critical"}[ri]
}

// RiskLikelihood defines risk likelihood levels
type RiskLikelihood int

const (
	LikelihoodLow RiskLikelihood = iota
	LikelihoodMedium
	LikelihoodHigh
)

// String returns string representation of risk likelihood
func (rl RiskLikelihood) String() string {
	return [...]string{"Low", "Medium", "High"}[rl]
}

// Estimates contains time and resource estimates
type Estimates struct {
	Duration   time.Duration
	Complexity Complexity
	Confidence float64 // 0-1
}

// Complexity defines complexity levels
type Complexity int

const (
	ComplexityLow Complexity = iota
	ComplexityMedium
	ComplexityHigh
	ComplexityVeryHigh
)

// String returns string representation of complexity
func (c Complexity) String() string {
	return [...]string{"Low", "Medium", "High", "VeryHigh"}[c]
}

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

// String returns string representation of plan status
func (ps PlanStatus) String() string {
	return [...]string{"Draft", "Ready", "InProgress", "Completed", "Failed", "Cancelled"}[ps]
}

// Priority defines task priority
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// String returns string representation of priority
func (p Priority) String() string {
	return [...]string{"Low", "Medium", "High", "Critical"}[p]
}

// ValidationResult contains plan validation results
type ValidationResult struct {
	Valid   bool
	Message string
	Errors  []string
}

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

// LLMPlanner implements Planner using an LLM
type LLMPlanner struct {
	llmProvider   llm.Provider
	promptBuilder *PromptBuilder
	validator     *PlanValidator
}

// NewLLMPlanner creates a new LLM-based planner
func NewLLMPlanner(llmProvider llm.Provider) *LLMPlanner {
	return &LLMPlanner{
		llmProvider:   llmProvider,
		promptBuilder: NewPromptBuilder(),
		validator:     NewPlanValidator(),
	}
}

// GeneratePlan generates a plan for a task
func (p *LLMPlanner) GeneratePlan(ctx context.Context, task *Task) (*Plan, error) {
	prompt := p.promptBuilder.BuildPlanPrompt(task)

	request := &llm.LLMRequest{
		Model: "default",
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are an expert software architect and planner. Generate detailed, actionable implementation plans.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	response, err := p.llmProvider.Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	plan, err := p.parsePlan(response.Content, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	return plan, nil
}

// GenerateOptions generates multiple implementation options
func (p *LLMPlanner) GenerateOptions(ctx context.Context, task *Task) ([]*PlanOption, error) {
	prompt := p.promptBuilder.BuildOptionPrompt(task)

	request := &llm.LLMRequest{
		Model: "default",
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are an expert software architect. Generate multiple distinct implementation approaches, each with pros, cons, and risk assessments.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   8192,
		Temperature: 0.8,
	}

	response, err := p.llmProvider.Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate options: %w", err)
	}

	options, err := p.parseOptions(response.Content, task.ID)
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

	// Sort by score (descending)
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

// RefinePlan refines a plan based on feedback
func (p *LLMPlanner) RefinePlan(ctx context.Context, plan *Plan, feedback string) (*Plan, error) {
	prompt := p.promptBuilder.BuildRefinementPrompt(plan, feedback)

	request := &llm.LLMRequest{
		Model: "default",
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are an expert software architect. Refine the implementation plan based on user feedback.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	response, err := p.llmProvider.Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to refine plan: %w", err)
	}

	refinedPlan, err := p.parsePlan(response.Content, plan.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse refined plan: %w", err)
	}

	refinedPlan.Version = plan.Version + 1
	return refinedPlan, nil
}

// ValidatePlan validates a plan
func (p *LLMPlanner) ValidatePlan(ctx context.Context, plan *Plan) (*ValidationResult, error) {
	return p.validator.ValidatePlan(ctx, plan)
}

// parsePlan parses LLM response into a plan
func (p *LLMPlanner) parsePlan(response string, taskID string) (*Plan, error) {
	var parsed struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Steps       []struct {
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			Type         string   `json:"type"`
			Action       string   `json:"action"`
			Dependencies []string `json:"dependencies"`
			Estimated    int      `json:"estimated_minutes"`
		} `json:"steps"`
		Risks []struct {
			Description string `json:"description"`
			Impact      string `json:"impact"`
			Likelihood  string `json:"likelihood"`
			Mitigation  string `json:"mitigation"`
		} `json:"risks"`
		Estimates struct {
			Duration   int     `json:"duration_minutes"`
			Complexity string  `json:"complexity"`
			Confidence float64 `json:"confidence"`
		} `json:"estimates"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	plan := &Plan{
		ID:          uuid.New().String(),
		TaskID:      taskID,
		Title:       parsed.Title,
		Description: parsed.Description,
		Steps:       make([]*PlanStep, len(parsed.Steps)),
		Risks:       make([]Risk, len(parsed.Risks)),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
		Status:      PlanDraft,
	}

	// Parse steps
	for i, step := range parsed.Steps {
		plan.Steps[i] = &PlanStep{
			ID:           uuid.New().String(),
			Order:        i + 1,
			Title:        step.Title,
			Description:  step.Description,
			Type:         parseStepType(step.Type),
			Action:       step.Action,
			Dependencies: step.Dependencies,
			Estimated:    time.Duration(step.Estimated) * time.Minute,
			Status:       StepPending,
		}
	}

	// Parse risks
	for i, risk := range parsed.Risks {
		plan.Risks[i] = Risk{
			Description: risk.Description,
			Impact:      parseRiskImpact(risk.Impact),
			Likelihood:  parseRiskLikelihood(risk.Likelihood),
			Mitigation:  risk.Mitigation,
		}
	}

	// Parse estimates
	plan.Estimates = Estimates{
		Duration:   time.Duration(parsed.Estimates.Duration) * time.Minute,
		Complexity: parseComplexity(parsed.Estimates.Complexity),
		Confidence: parsed.Estimates.Confidence,
	}

	return plan, nil
}

// parseOptions parses LLM response into options
func (p *LLMPlanner) parseOptions(response string, taskID string) ([]*PlanOption, error) {
	var parsed struct {
		Options []struct {
			Title       string          `json:"title"`
			Description string          `json:"description"`
			Plan        json.RawMessage `json:"plan"`
			Pros        []string        `json:"pros"`
			Cons        []string        `json:"cons"`
		} `json:"options"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options: %w", err)
	}

	options := make([]*PlanOption, len(parsed.Options))
	for i, opt := range parsed.Options {
		plan, err := p.parsePlan(string(opt.Plan), taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse plan for option %d: %w", i, err)
		}

		options[i] = &PlanOption{
			ID:          uuid.New().String(),
			Title:       opt.Title,
			Description: opt.Description,
			Plan:        plan,
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

// Helper parsing functions

func parseStepType(s string) StepType {
	switch s {
	case "file_operation":
		return StepTypeFileOperation
	case "shell_command":
		return StepTypeShellCommand
	case "code_generation":
		return StepTypeCodeGeneration
	case "code_analysis":
		return StepTypeCodeAnalysis
	case "validation":
		return StepTypeValidation
	case "testing":
		return StepTypeTesting
	default:
		return StepTypeFileOperation
	}
}

func parseRiskImpact(s string) RiskImpact {
	switch s {
	case "low":
		return ImpactLow
	case "medium":
		return ImpactMedium
	case "high":
		return ImpactHigh
	case "critical":
		return ImpactCritical
	default:
		return ImpactLow
	}
}

func parseRiskLikelihood(s string) RiskLikelihood {
	switch s {
	case "low":
		return LikelihoodLow
	case "medium":
		return LikelihoodMedium
	case "high":
		return LikelihoodHigh
	default:
		return LikelihoodLow
	}
}

func parseComplexity(s string) Complexity {
	switch s {
	case "low":
		return ComplexityLow
	case "medium":
		return ComplexityMedium
	case "high":
		return ComplexityHigh
	case "very_high":
		return ComplexityVeryHigh
	default:
		return ComplexityMedium
	}
}

// PlanValidator validates plans
type PlanValidator struct{}

// NewPlanValidator creates a new plan validator
func NewPlanValidator() *PlanValidator {
	return &PlanValidator{}
}

// ValidatePlan validates a plan
func (v *PlanValidator) ValidatePlan(ctx context.Context, plan *Plan) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Validate basic fields
	if plan.Title == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "plan title is required")
	}

	if len(plan.Steps) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "plan must have at least one step")
	}

	// Validate steps
	for i, step := range plan.Steps {
		if step.Title == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("step %d: title is required", i+1))
		}

		// Check dependencies exist
		for _, depID := range step.Dependencies {
			found := false
			for _, s := range plan.Steps {
				if s.ID == depID {
					found = true
					break
				}
			}
			if !found {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("step %d: dependency %s not found", i+1, depID))
			}
		}
	}

	// Set message
	if result.Valid {
		result.Message = "Plan is valid"
	} else {
		result.Message = fmt.Sprintf("Plan validation failed with %d errors", len(result.Errors))
	}

	return result, nil
}

// PromptBuilder builds prompts for the LLM
type PromptBuilder struct{}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildPlanPrompt builds a prompt for generating a plan
func (pb *PromptBuilder) BuildPlanPrompt(task *Task) string {
	prompt := fmt.Sprintf(`Generate an implementation plan for the following task:

Task: %s

`, task.Description)

	if task.Context != nil {
		prompt += fmt.Sprintf("Context:\n- Workspace: %s\n", task.Context.WorkspaceRoot)
		if len(task.Context.CurrentFiles) > 0 {
			prompt += fmt.Sprintf("- Current Files: %v\n", task.Context.CurrentFiles)
		}
	}

	if len(task.Requirements) > 0 {
		prompt += "\nRequirements:\n"
		for _, req := range task.Requirements {
			prompt += fmt.Sprintf("- %s\n", req)
		}
	}

	if len(task.Constraints) > 0 {
		prompt += "\nConstraints:\n"
		for _, con := range task.Constraints {
			prompt += fmt.Sprintf("- %s\n", con)
		}
	}

	prompt += `
Provide a detailed implementation plan in JSON format with:
1. Title and description
2. Step-by-step plan with dependencies
3. Risk assessment
4. Time and complexity estimates

Format your response as JSON.`

	return prompt
}

// BuildOptionPrompt builds a prompt for generating options
func (pb *PromptBuilder) BuildOptionPrompt(task *Task) string {
	prompt := pb.BuildPlanPrompt(task)
	prompt += `

Generate 3-4 different implementation options, each with:
- A distinct approach
- Pros and cons
- Risk assessment
- Estimates

Format as JSON with an "options" array.`

	return prompt
}

// BuildRefinementPrompt builds a prompt for refining a plan
func (pb *PromptBuilder) BuildRefinementPrompt(plan *Plan, feedback string) string {
	prompt := fmt.Sprintf(`Refine the following implementation plan based on user feedback:

Current Plan: %s
Description: %s

User Feedback: %s

Provide an improved plan in JSON format, addressing the feedback while maintaining the same overall structure.`,
		plan.Title, plan.Description, feedback)

	return prompt
}
