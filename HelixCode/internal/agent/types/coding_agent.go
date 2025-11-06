package types

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dev.helix.code/internal/agent"
	"dev.helix.code/internal/agent/task"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/tools"
)

// CodingAgent is specialized in code generation and modification
type CodingAgent struct {
	*agent.BaseAgent
	llmProvider  llm.Provider
	toolRegistry *tools.ToolRegistry
}

// NewCodingAgent creates a new coding agent
func NewCodingAgent(config *agent.AgentConfig, provider llm.Provider, toolRegistry *tools.ToolRegistry) (*CodingAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required for coding agent")
	}
	if toolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required for coding agent")
	}

	baseAgent := agent.NewBaseAgent(config)

	return &CodingAgent{
		BaseAgent:    baseAgent,
		llmProvider:  provider,
		toolRegistry: toolRegistry,
	}, nil
}

// Initialize initializes the coding agent
func (a *CodingAgent) Initialize(ctx context.Context, config *agent.AgentConfig) error {
	a.SetStatus(agent.StatusIdle)
	return nil
}

// Execute performs code generation or modification for a given task
func (a *CodingAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	a.SetStatus(agent.StatusBusy)
	defer a.SetStatus(agent.StatusIdle)

	startTime := time.Now()
	result := task.NewResult(t.ID, a.ID())

	// Increment task count
	a.IncrementTaskCount()

	// Extract code requirements from task input
	requirements, ok := t.Input["requirements"].(string)
	if !ok {
		a.IncrementErrorCount()
		err := fmt.Errorf("requirements not found in task input")
		result.SetFailure(err)
		return result, err
	}

	// Get file path if editing existing code
	filePath, _ := t.Input["file_path"].(string)
	existingCode, _ := t.Input["existing_code"].(string)

	// Determine operation type
	var operationType string
	if existingCode != "" {
		operationType = "edit"
	} else {
		operationType = "create"
	}

	// Generate or modify code using LLM
	generatedCode, explanation, err := a.generateCode(ctx, requirements, existingCode, operationType)
	if err != nil {
		a.IncrementErrorCount()
		result.SetFailure(err)
		return result, err
	}

	// Apply code changes using tools
	artifacts, err := a.applyCodeChanges(ctx, filePath, generatedCode, operationType)
	if err != nil {
		a.IncrementErrorCount()
		result.SetFailure(err)
		return result, err
	}

	// Set result
	output := map[string]interface{}{
		"operation":   operationType,
		"file_path":   filePath,
		"code":        generatedCode,
		"explanation": explanation,
		"artifacts":   artifacts,
	}

	result.SetSuccess(output, 0.8) // 80% confidence for code generation
	result.Duration = time.Since(startTime)
	result.Artifacts = artifacts

	// Set metrics
	result.Metrics = &task.TaskMetrics{
		FilesModified:  len(artifacts),
		LinesAdded:     countLines(generatedCode),
		ExecutionTime:  result.Duration,
	}

	return result, nil
}

// generateCode uses LLM to generate or modify code
func (a *CodingAgent) generateCode(ctx context.Context, requirements, existingCode, operationType string) (string, string, error) {
	var prompt string

	if operationType == "edit" {
		prompt = fmt.Sprintf(`You are a code generation agent. Modify the following code according to the requirements.

Requirements:
%s

Existing Code:
%s

Please provide:
1. The modified code
2. A brief explanation of changes made

Format your response as JSON:
{
  "code": "the complete modified code",
  "explanation": "explanation of changes"
}

Only return the JSON, no other text.`, requirements, existingCode)
	} else {
		prompt = fmt.Sprintf(`You are a code generation agent. Generate code according to the following requirements.

Requirements:
%s

Please provide:
1. The complete code implementation
2. A brief explanation of the implementation

Format your response as JSON:
{
  "code": "the complete code",
  "explanation": "explanation of implementation"
}

Only return the JSON, no other text.`, requirements)
	}

	// Get a model from the provider
	models := a.llmProvider.GetModels()
	if len(models) == 0 {
		return "", "", fmt.Errorf("no models available from provider")
	}

	request := &llm.LLMRequest{
		Model:       models[0].Name,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4000,
		Temperature: 0.2, // Low temperature for consistent code generation
	}

	response, err := a.llmProvider.Generate(ctx, request)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate code: %w", err)
	}

	if response.Content == "" {
		return "", "", fmt.Errorf("no code generated")
	}

	// Parse JSON response
	var codeResponse struct {
		Code        string `json:"code"`
		Explanation string `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(response.Content), &codeResponse); err != nil {
		return "", "", fmt.Errorf("failed to parse code response: %w", err)
	}

	return codeResponse.Code, codeResponse.Explanation, nil
}

// applyCodeChanges applies the generated code using tool registry
func (a *CodingAgent) applyCodeChanges(ctx context.Context, filePath, code, operationType string) ([]task.Artifact, error) {
	var artifacts []task.Artifact

	if filePath == "" {
		// No file path specified, return code as artifact only
		artifact := task.Artifact{
			ID:        fmt.Sprintf("code-%d", time.Now().Unix()),
			Type:      "code",
			Path:      "generated",
			Content:   code,
			Size:      int64(len(code)),
			CreatedAt: time.Now(),
		}
		artifacts = append(artifacts, artifact)
		return artifacts, nil
	}

	// Get the appropriate tool
	var toolName string
	if operationType == "edit" {
		toolName = "FSEdit"
	} else {
		toolName = "FSWrite"
	}

	tool, err := a.toolRegistry.Get(toolName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool %s: %w", toolName, err)
	}

	// Prepare tool parameters
	params := map[string]interface{}{
		"path":    filePath,
		"content": code,
	}

	// Execute tool
	_, err = tool.Execute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool %s: %w", toolName, err)
	}

	// Create artifact from tool result
	artifact := task.Artifact{
		ID:        fmt.Sprintf("file-%d", time.Now().Unix()),
		Type:      "code",
		Path:      filePath,
		Content:   code,
		Size:      int64(len(code)),
		CreatedAt: time.Now(),
	}

	artifacts = append(artifacts, artifact)

	return artifacts, nil
}

// Collaborate allows this agent to work with other agents
func (a *CodingAgent) Collaborate(ctx context.Context, agents []agent.Agent, t *task.Task) (*agent.CollaborationResult, error) {
	result := &agent.CollaborationResult{
		Success:      true,
		Results:      make(map[string]*task.Result),
		Participants: []string{a.ID()},
		Messages:     []*agent.CollaborationMessage{},
	}

	startTime := time.Now()

	// Execute our own coding task
	myResult, err := a.Execute(ctx, t)
	if err != nil {
		result.Success = false
		return result, err
	}
	result.Results[a.ID()] = myResult

	// Check if there are review agents to validate the code
	for _, other := range agents {
		if other.Type() == agent.AgentTypeReview {
			// Create a review task
			reviewTask := task.NewTask(
				task.TaskTypeReview,
				"Code Review",
				"Review generated code",
				task.PriorityNormal,
			)
			reviewTask.Input = map[string]interface{}{
				"code":     myResult.Output["code"],
				"file_path": myResult.Output["file_path"],
			}

			reviewResult, err := other.Execute(ctx, reviewTask)
			if err != nil {
				continue // Skip failed reviews
			}

			result.Results[other.ID()] = reviewResult
			result.Participants = append(result.Participants, other.ID())

			// Add collaboration message
			msg := &agent.CollaborationMessage{
				ID:        fmt.Sprintf("msg-%d", time.Now().Unix()),
				From:      a.ID(),
				To:        other.ID(),
				Type:      agent.MessageTypeRequest,
				Content:   "Please review the generated code",
				Timestamp: time.Now(),
			}
			result.Messages = append(result.Messages, msg)
		}
	}

	// Use our result as consensus
	result.Consensus = myResult
	result.Duration = time.Since(startTime)

	return result, nil
}

// Shutdown cleanly shuts down the agent
func (a *CodingAgent) Shutdown(ctx context.Context) error {
	a.SetStatus(agent.StatusShutdown)
	return nil
}

// Helper functions

func countLines(code string) int {
	if code == "" {
		return 0
	}
	lines := 1
	for _, c := range code {
		if c == '\n' {
			lines++
		}
	}
	return lines
}
