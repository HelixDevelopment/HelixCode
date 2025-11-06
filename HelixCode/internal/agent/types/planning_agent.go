package types

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dev.helix.code/internal/agent"
	"dev.helix.code/internal/agent/task"
	"dev.helix.code/internal/llm"
)

// PlanningAgent is specialized in analyzing requirements and creating task plans
type PlanningAgent struct {
	*agent.BaseAgent
	llmProvider llm.Provider
}

// NewPlanningAgent creates a new planning agent
func NewPlanningAgent(config *agent.AgentConfig, provider llm.Provider) (*PlanningAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required for planning agent")
	}

	baseAgent := agent.NewBaseAgent(config)

	return &PlanningAgent{
		BaseAgent:   baseAgent,
		llmProvider: provider,
	}, nil
}

// Initialize initializes the planning agent
func (a *PlanningAgent) Initialize(ctx context.Context, config *agent.AgentConfig) error {
	a.SetStatus(agent.StatusIdle)
	return nil
}

// Execute performs planning for a given task
func (a *PlanningAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	a.SetStatus(agent.StatusBusy)
	defer a.SetStatus(agent.StatusIdle)

	startTime := time.Now()
	result := task.NewResult(t.ID, a.ID())

	// Increment task count
	a.IncrementTaskCount()

	// Extract requirements from task input
	requirements, ok := t.Input["requirements"].(string)
	if !ok {
		a.IncrementErrorCount()
		err := fmt.Errorf("requirements not found in task input")
		result.SetFailure(err)
		return result, err
	}

	// Generate plan using LLM
	plan, err := a.generatePlan(ctx, requirements)
	if err != nil {
		a.IncrementErrorCount()
		result.SetFailure(err)
		return result, err
	}

	// Parse plan into subtasks
	subtasks, err := a.parseSubtasks(plan)
	if err != nil {
		a.IncrementErrorCount()
		result.SetFailure(err)
		return result, err
	}

	// Set result
	output := map[string]interface{}{
		"plan":         plan,
		"subtasks":     subtasks,
		"total_tasks":  len(subtasks),
		"estimated_duration": a.estimateDuration(subtasks),
	}

	result.SetSuccess(output, 0.85) // 85% confidence for planning
	result.Duration = time.Since(startTime)

	return result, nil
}

// generatePlan uses LLM to generate a detailed plan
func (a *PlanningAgent) generatePlan(ctx context.Context, requirements string) (string, error) {
	prompt := fmt.Sprintf(`You are a software planning agent. Analyze the following requirements and create a detailed technical plan.

Requirements:
%s

Please provide:
1. A brief analysis of the requirements
2. Key technical decisions
3. A breakdown of subtasks (with task type, description, priority, estimated duration, and dependencies)
4. Potential risks and mitigations

Format your response as a structured plan with clear sections.`, requirements)

	// Get a model from the provider
	models := a.llmProvider.GetModels()
	if len(models) == 0 {
		return "", fmt.Errorf("no models available from provider")
	}

	request := &llm.LLMRequest{
		Model:       models[0].Name,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   2000,
		Temperature: 0.3, // Low temperature for consistent planning
	}

	response, err := a.llmProvider.Generate(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to generate plan: %w", err)
	}

	if response.Content == "" {
		return "", fmt.Errorf("no plan generated")
	}

	return response.Content, nil
}

// parseSubtasks parses the LLM-generated plan into structured subtasks
func (a *PlanningAgent) parseSubtasks(plan string) ([]*task.Task, error) {
	// Use LLM to extract structured subtasks
	prompt := fmt.Sprintf(`Extract subtasks from the following plan and format them as JSON array.

Plan:
%s

Return a JSON array with this structure:
[
  {
    "title": "Task title",
    "description": "Task description",
    "type": "planning|analysis|code_generation|code_edit|refactoring|testing|debugging|review|documentation|research",
    "priority": 1-4 (1=low, 2=normal, 3=high, 4=critical),
    "estimated_duration_minutes": number,
    "depends_on": ["task_title_1", "task_title_2"],
    "required_capabilities": ["capability1", "capability2"]
  }
]

Only return the JSON array, no other text.`, plan)

	// Get a model from the provider
	models := a.llmProvider.GetModels()
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available from provider")
	}

	request := &llm.LLMRequest{
		Model:       models[0].Name,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   1500,
		Temperature: 0.2, // Very low temperature for structured output
	}

	response, err := a.llmProvider.Generate(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subtasks: %w", err)
	}

	if response.Content == "" {
		return nil, fmt.Errorf("no subtasks generated")
	}

	// Parse JSON response
	var subtaskData []map[string]interface{}
	content := response.Content
	if err := json.Unmarshal([]byte(content), &subtaskData); err != nil {
		return nil, fmt.Errorf("failed to parse subtasks JSON: %w", err)
	}

	// Convert to Task objects
	subtasks := make([]*task.Task, 0, len(subtaskData))
	for _, data := range subtaskData {
		t := a.createTaskFromData(data)
		subtasks = append(subtasks, t)
	}

	return subtasks, nil
}

// createTaskFromData creates a Task from parsed data
func (a *PlanningAgent) createTaskFromData(data map[string]interface{}) *task.Task {
	title, _ := data["title"].(string)
	description, _ := data["description"].(string)
	taskType, _ := data["type"].(string)
	priorityFloat, _ := data["priority"].(float64)
	durationMinutes, _ := data["estimated_duration_minutes"].(float64)

	priority := task.Priority(priorityFloat)
	if priority < task.PriorityLow {
		priority = task.PriorityNormal
	}
	if priority > task.PriorityCritical {
		priority = task.PriorityCritical
	}

	t := task.NewTask(task.TaskType(taskType), title, description, priority)
	t.EstimatedDuration = time.Duration(durationMinutes) * time.Minute
	t.CreatedBy = a.ID()

	// Set required capabilities
	if caps, ok := data["required_capabilities"].([]interface{}); ok {
		t.RequiredCapabilities = make([]string, len(caps))
		for i, cap := range caps {
			if capStr, ok := cap.(string); ok {
				t.RequiredCapabilities[i] = capStr
			}
		}
	}

	// Set dependencies (will be resolved later by coordinator)
	if deps, ok := data["depends_on"].([]interface{}); ok {
		t.DependsOn = make([]string, len(deps))
		for i, dep := range deps {
			if depStr, ok := dep.(string); ok {
				t.DependsOn[i] = depStr
			}
		}
	}

	return t
}

// estimateDuration estimates total duration for all subtasks
func (a *PlanningAgent) estimateDuration(subtasks []*task.Task) time.Duration {
	var total time.Duration
	for _, t := range subtasks {
		total += t.EstimatedDuration
	}
	// Add 20% buffer for coordination overhead
	return time.Duration(float64(total) * 1.2)
}

// Collaborate allows this agent to work with other agents
func (a *PlanningAgent) Collaborate(ctx context.Context, agents []agent.Agent, t *task.Task) (*agent.CollaborationResult, error) {
	// Planning agents typically lead collaboration
	// They can consult with other planning agents for consensus

	result := &agent.CollaborationResult{
		Success:      true,
		Results:      make(map[string]*task.Result),
		Participants: []string{a.ID()},
		Messages:     []*agent.CollaborationMessage{},
	}

	startTime := time.Now()

	// Execute our own plan
	myResult, err := a.Execute(ctx, t)
	if err != nil {
		result.Success = false
		return result, err
	}
	result.Results[a.ID()] = myResult

	// If there are other planning agents, get their input too
	for _, other := range agents {
		if other.Type() == agent.AgentTypePlanning && other.ID() != a.ID() {
			otherResult, err := other.Execute(ctx, t)
			if err != nil {
				continue // Skip failed agents
			}
			result.Results[other.ID()] = otherResult
			result.Participants = append(result.Participants, other.ID())
		}
	}

	// Use our result as consensus (could implement voting mechanism here)
	result.Consensus = myResult
	result.Duration = time.Since(startTime)

	return result, nil
}

// Shutdown cleanly shuts down the agent
func (a *PlanningAgent) Shutdown(ctx context.Context) error {
	a.SetStatus(agent.StatusShutdown)
	return nil
}
