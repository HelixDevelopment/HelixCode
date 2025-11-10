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

// DebuggingAgent is specialized in analyzing and fixing errors
type DebuggingAgent struct {
	*agent.BaseAgent
	llmProvider  llm.Provider
	toolRegistry *tools.ToolRegistry
}

// NewDebuggingAgent creates a new debugging agent
func NewDebuggingAgent(config *agent.AgentConfig, provider llm.Provider, toolRegistry *tools.ToolRegistry) (*DebuggingAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required for debugging agent")
	}
	if toolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required for debugging agent")
	}

	baseAgent := agent.NewBaseAgent(config)

	return &DebuggingAgent{
		BaseAgent:    baseAgent,
		llmProvider:  provider,
		toolRegistry: toolRegistry,
	}, nil
}

// Initialize initializes the debugging agent
func (a *DebuggingAgent) Initialize(ctx context.Context, config *agent.AgentConfig) error {
	a.SetStatus(agent.StatusIdle)
	return nil
}

// Execute performs error analysis and debugging for a given task
func (a *DebuggingAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	a.SetStatus(agent.StatusBusy)
	defer a.SetStatus(agent.StatusIdle)

	startTime := time.Now()
	result := task.NewResult(t.ID, a.ID())

	// Increment task count
	a.IncrementTaskCount()

	// Extract error information from task input
	errorMessage, ok := t.Input["error"].(string)
	if !ok {
		a.IncrementErrorCount()
		err := fmt.Errorf("error message not found in task input")
		result.SetFailure(err)
		return result, err
	}

	filePath, _ := t.Input["file_path"].(string)
	stackTrace, _ := t.Input["stack_trace"].(string)
	codeContext, _ := t.Input["code_context"].(string)

	// If code context not provided, try to read from file
	if codeContext == "" && filePath != "" {
		var err error
		codeContext, err = a.readFile(ctx, filePath)
		if err != nil {
			// Non-fatal, continue without context
			codeContext = "Unable to read file context"
		}
	}

	// Analyze error using LLM
	analysis, rootCause, suggestedFixes, err := a.analyzeError(ctx, errorMessage, stackTrace, codeContext, filePath)
	if err != nil {
		a.IncrementErrorCount()
		result.SetFailure(err)
		return result, err
	}

	// Execute diagnostic commands if needed
	var diagnosticResults map[string]interface{}
	runDiagnostics, _ := t.Input["run_diagnostics"].(bool)
	if runDiagnostics {
		diagnosticResults, err = a.runDiagnostics(ctx, filePath, errorMessage)
		if err != nil {
			// Non-fatal, continue without diagnostic results
			diagnosticResults = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		}
	}

	// Apply fixes if auto_fix is enabled
	var fixResults map[string]interface{}
	autoFix, _ := t.Input["auto_fix"].(bool)
	if autoFix && len(suggestedFixes) > 0 {
		fixResults, err = a.applyFix(ctx, filePath, suggestedFixes[0])
		if err != nil {
			// Non-fatal, just record the error
			fixResults = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		}
	}

	// Set result
	output := map[string]interface{}{
		"error_message":      errorMessage,
		"analysis":           analysis,
		"root_cause":         rootCause,
		"suggested_fixes":    suggestedFixes,
		"diagnostic_results": diagnosticResults,
		"fix_results":        fixResults,
	}

	result.SetSuccess(output, 0.80) // 80% confidence for debugging
	result.Duration = time.Since(startTime)

	// Set metrics
	result.Metrics = &task.TaskMetrics{
		FilesModified: 0, // Will be 1 if auto_fix applied
		LinesAdded:    0,
		ExecutionTime: result.Duration,
	}

	if fixResults != nil && fixResults["status"] == "success" {
		result.Metrics.FilesModified = 1
	}

	return result, nil
}

// analyzeError uses LLM to analyze the error and identify root cause
func (a *DebuggingAgent) analyzeError(ctx context.Context, errorMsg, stackTrace, codeContext, filePath string) (string, string, []string, error) {
	prompt := fmt.Sprintf(`You are a debugging agent. Analyze the following error and identify the root cause.

Error Message:
%s

Stack Trace:
%s

Code Context:
%s

File Path: %s

Please provide:
1. Detailed analysis of the error
2. Root cause identification
3. List of suggested fixes (in order of likelihood to solve the issue)

Format your response as JSON:
{
  "analysis": "detailed analysis of what went wrong",
  "root_cause": "specific root cause identified",
  "suggested_fixes": [
    "Fix 1: description and implementation",
    "Fix 2: alternative approach",
    ...
  ]
}

Only return the JSON, no other text.`, errorMsg, stackTrace, codeContext, filePath)

	// Get a model from the provider
	models := a.llmProvider.GetModels()
	if len(models) == 0 {
		return "", "", nil, fmt.Errorf("no models available from provider")
	}

	request := &llm.LLMRequest{
		Model:       models[0].Name,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   3000,
		Temperature: 0.2, // Low temperature for precise debugging
	}

	response, err := a.llmProvider.Generate(ctx, request)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to analyze error: %w", err)
	}

	if response.Content == "" {
		return "", "", nil, fmt.Errorf("no analysis generated")
	}

	// Parse JSON response
	var debugResponse struct {
		Analysis       string   `json:"analysis"`
		RootCause      string   `json:"root_cause"`
		SuggestedFixes []string `json:"suggested_fixes"`
	}

	if err := json.Unmarshal([]byte(response.Content), &debugResponse); err != nil {
		return "", "", nil, fmt.Errorf("failed to parse debug response: %w", err)
	}

	return debugResponse.Analysis, debugResponse.RootCause, debugResponse.SuggestedFixes, nil
}

// readFile reads a file using FSRead tool
func (a *DebuggingAgent) readFile(ctx context.Context, filePath string) (string, error) {
	tool, err := a.toolRegistry.Get("FSRead")
	if err != nil {
		return "", fmt.Errorf("failed to get FSRead tool: %w", err)
	}

	params := map[string]interface{}{
		"path": filePath,
	}

	output, err := tool.Execute(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Convert output to string
	content, ok := output.(string)
	if !ok {
		return "", fmt.Errorf("unexpected output type from FSRead")
	}

	return content, nil
}

// runDiagnostics executes diagnostic commands to gather more information
func (a *DebuggingAgent) runDiagnostics(ctx context.Context, filePath, errorMsg string) (map[string]interface{}, error) {
	tool, err := a.toolRegistry.Get("Shell")
	if err != nil {
		return nil, fmt.Errorf("failed to get Shell tool: %w", err)
	}

	// Determine diagnostic commands based on file type and error
	commands := a.determineDiagnosticCommands(filePath, errorMsg)

	results := make(map[string]interface{})
	for name, cmd := range commands {
		params := map[string]interface{}{
			"command": cmd,
			"timeout": 10000, // 10 second timeout
		}

		output, err := tool.Execute(ctx, params)
		if err != nil {
			results[name] = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		} else {
			results[name] = map[string]interface{}{
				"status": "success",
				"output": output,
			}
		}
	}

	return results, nil
}

// determineDiagnosticCommands determines which diagnostic commands to run
func (a *DebuggingAgent) determineDiagnosticCommands(filePath, errorMsg string) map[string]string {
	commands := make(map[string]string)

	// For Go files
	if len(filePath) > 3 && filePath[len(filePath)-3:] == ".go" {
		dir := getTestDirectory(filePath)
		commands["go_vet"] = fmt.Sprintf("go vet %s", dir)
		commands["go_build"] = fmt.Sprintf("go build %s", dir)
		commands["go_test"] = fmt.Sprintf("go test -v %s", dir)
	}

	// Add more language-specific diagnostics as needed

	return commands
}

// applyFix applies a suggested fix to the code
func (a *DebuggingAgent) applyFix(ctx context.Context, filePath string, fix string) (map[string]interface{}, error) {
	if filePath == "" {
		return map[string]interface{}{
			"status":  "skipped",
			"message": "no file path specified",
		}, nil
	}

	// Use LLM to generate the fixed code
	fixedCode, err := a.generateFixedCode(ctx, filePath, fix)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fixed code: %w", err)
	}

	// Apply the fix using FSWrite tool
	tool, err := a.toolRegistry.Get("FSWrite")
	if err != nil {
		return nil, fmt.Errorf("failed to get FSWrite tool: %w", err)
	}

	params := map[string]interface{}{
		"path":    filePath,
		"content": fixedCode,
	}

	_, err = tool.Execute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to write fixed code: %w", err)
	}

	return map[string]interface{}{
		"status":      "success",
		"file_path":   filePath,
		"fix_applied": fix,
	}, nil
}

// generateFixedCode uses LLM to generate fixed code
func (a *DebuggingAgent) generateFixedCode(ctx context.Context, filePath, fix string) (string, error) {
	// Read current code
	currentCode, err := a.readFile(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read current code: %w", err)
	}

	prompt := fmt.Sprintf(`You are a debugging agent applying a fix to code.

Current Code:
%s

Fix to Apply:
%s

Please provide the complete fixed code with the suggested fix applied.

Format your response as JSON:
{
  "fixed_code": "the complete fixed code"
}

Only return the JSON, no other text.`, currentCode, fix)

	models := a.llmProvider.GetModels()
	if len(models) == 0 {
		return "", fmt.Errorf("no models available from provider")
	}

	request := &llm.LLMRequest{
		Model:       models[0].Name,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4000,
		Temperature: 0.1, // Very low temperature for precise fixes
	}

	response, err := a.llmProvider.Generate(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to generate fixed code: %w", err)
	}

	if response.Content == "" {
		return "", fmt.Errorf("no fixed code generated")
	}

	var fixResponse struct {
		FixedCode string `json:"fixed_code"`
	}

	if err := json.Unmarshal([]byte(response.Content), &fixResponse); err != nil {
		return "", fmt.Errorf("failed to parse fix response: %w", err)
	}

	return fixResponse.FixedCode, nil
}

// Collaborate allows this agent to work with other agents
func (a *DebuggingAgent) Collaborate(ctx context.Context, agents []agent.Agent, t *task.Task) (*agent.CollaborationResult, error) {
	result := &agent.CollaborationResult{
		Success:      true,
		Results:      make(map[string]*task.Result),
		Participants: []string{a.ID()},
		Messages:     []*agent.CollaborationMessage{},
	}

	startTime := time.Now()

	// Execute our own debugging task
	myResult, err := a.Execute(ctx, t)
	if err != nil {
		result.Success = false
		return result, err
	}
	result.Results[a.ID()] = myResult

	// Check if there are testing agents to verify the fix
	for _, other := range agents {
		if other.Type() == agent.AgentTypeTesting {
			// Create a testing task to verify the fix worked
			testTask := task.NewTask(
				task.TaskTypeTesting,
				"Verify Fix",
				"Test that the fix resolved the issue",
				task.PriorityHigh,
			)
			testTask.Input = map[string]interface{}{
				"file_path":     myResult.Output["file_path"],
				"execute_tests": true,
			}

			testResult, err := other.Execute(ctx, testTask)
			if err != nil {
				continue // Skip failed tests
			}

			result.Results[other.ID()] = testResult
			result.Participants = append(result.Participants, other.ID())

			// Add collaboration message
			msg := &agent.CollaborationMessage{
				ID:        fmt.Sprintf("msg-%d", time.Now().Unix()),
				From:      a.ID(),
				To:        other.ID(),
				Type:      agent.MessageTypeRequest,
				Content:   "Please verify the fix with tests",
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
func (a *DebuggingAgent) Shutdown(ctx context.Context) error {
	a.SetStatus(agent.StatusShutdown)
	return nil
}

// Helper functions are in utils.go
