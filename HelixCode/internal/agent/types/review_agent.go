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

// ReviewAgent is specialized in code review and quality analysis
type ReviewAgent struct {
	*agent.BaseAgent
	llmProvider  llm.Provider
	toolRegistry *tools.ToolRegistry
}

// NewReviewAgent creates a new review agent
func NewReviewAgent(config *agent.AgentConfig, provider llm.Provider, toolRegistry *tools.ToolRegistry) (*ReviewAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required for review agent")
	}
	if toolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required for review agent")
	}

	baseAgent := agent.NewBaseAgent(config)

	return &ReviewAgent{
		BaseAgent:    baseAgent,
		llmProvider:  provider,
		toolRegistry: toolRegistry,
	}, nil
}

// Initialize initializes the review agent
func (a *ReviewAgent) Initialize(ctx context.Context, config *agent.AgentConfig) error {
	a.SetStatus(agent.StatusIdle)
	return nil
}

// Execute performs code review for a given task
func (a *ReviewAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	a.SetStatus(agent.StatusBusy)
	defer a.SetStatus(agent.StatusIdle)

	startTime := time.Now()
	result := task.NewResult(t.ID, a.ID())

	// Increment task count
	a.IncrementTaskCount()

	// Extract code to review from task input
	codeToReview, ok := t.Input["code"].(string)
	if !ok {
		// Try to get from file path
		filePath, pathOk := t.Input["file_path"].(string)
		if !pathOk {
			a.IncrementErrorCount()
			err := fmt.Errorf("code or file_path not found in task input")
			result.SetFailure(err)
			return result, err
		}

		// Read code from file
		var err error
		codeToReview, err = a.readFile(ctx, filePath)
		if err != nil {
			a.IncrementErrorCount()
			result.SetFailure(err)
			return result, err
		}
	}

	filePath, _ := t.Input["file_path"].(string)
	reviewType, _ := t.Input["review_type"].(string)
	if reviewType == "" {
		reviewType = "comprehensive" // Default review type
	}

	// Perform code review using LLM
	reviewResult, issues, suggestions, metrics, err := a.performReview(ctx, codeToReview, filePath, reviewType)
	if err != nil {
		a.IncrementErrorCount()
		result.SetFailure(err)
		return result, err
	}

	// Run static analysis if requested
	var staticAnalysisResults map[string]interface{}
	runStaticAnalysis, _ := t.Input["run_static_analysis"].(bool)
	if runStaticAnalysis && filePath != "" {
		staticAnalysisResults, err = a.runStaticAnalysis(ctx, filePath)
		if err != nil {
			// Non-fatal, continue without static analysis
			staticAnalysisResults = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		}
	}

	// Set result
	output := map[string]interface{}{
		"review_type":             reviewType,
		"review_result":           reviewResult,
		"issues":                  issues,
		"suggestions":             suggestions,
		"metrics":                 metrics,
		"static_analysis_results": staticAnalysisResults,
	}

	// Calculate confidence based on number of issues
	confidence := 0.85
	if len(issues) > 10 {
		confidence = 0.75 // Lower confidence if many issues found
	}

	result.SetSuccess(output, confidence)
	result.Duration = time.Since(startTime)

	// Set metrics
	result.Metrics = &task.TaskMetrics{
		LinesAdded:    0,
		LinesRemoved:  0,
		ExecutionTime: result.Duration,
	}

	return result, nil
}

// performReview uses LLM to perform comprehensive code review
func (a *ReviewAgent) performReview(ctx context.Context, code, filePath, reviewType string) (string, []map[string]interface{}, []string, map[string]interface{}, error) {
	var prompt string

	switch reviewType {
	case "security":
		prompt = fmt.Sprintf(`You are a security-focused code review agent. Review the following code for security vulnerabilities.

Code to review:
%s

File path: %s

Please analyze for:
- SQL injection vulnerabilities
- XSS vulnerabilities
- Authentication/authorization issues
- Data exposure risks
- Input validation problems
- Cryptography misuse
- Dependency vulnerabilities

Format your response as JSON:
{
  "review_summary": "overall security assessment",
  "issues": [
    {
      "severity": "critical|high|medium|low",
      "type": "security",
      "description": "detailed description",
      "line_number": 0,
      "recommendation": "how to fix"
    }
  ],
  "suggestions": ["suggestion 1", "suggestion 2", ...],
  "metrics": {
    "security_score": 0-100,
    "critical_issues": 0,
    "high_issues": 0,
    "medium_issues": 0,
    "low_issues": 0
  }
}

Only return the JSON, no other text.`, code, filePath)

	case "performance":
		prompt = fmt.Sprintf(`You are a performance-focused code review agent. Review the following code for performance issues.

Code to review:
%s

File path: %s

Please analyze for:
- Algorithm complexity issues
- Memory leaks
- Inefficient loops
- Unnecessary allocations
- Database query optimization
- Caching opportunities
- Concurrency issues

Format your response as JSON:
{
  "review_summary": "overall performance assessment",
  "issues": [
    {
      "severity": "critical|high|medium|low",
      "type": "performance",
      "description": "detailed description",
      "line_number": 0,
      "recommendation": "how to fix"
    }
  ],
  "suggestions": ["suggestion 1", "suggestion 2", ...],
  "metrics": {
    "performance_score": 0-100,
    "complexity_issues": 0,
    "memory_issues": 0
  }
}

Only return the JSON, no other text.`, code, filePath)

	default: // comprehensive
		prompt = fmt.Sprintf(`You are a comprehensive code review agent. Review the following code for quality, maintainability, and best practices.

Code to review:
%s

File path: %s

Please analyze for:
- Code quality and readability
- Best practices adherence
- Error handling
- Documentation and comments
- Code organization
- Naming conventions
- Testing coverage
- Security concerns
- Performance issues

Format your response as JSON:
{
  "review_summary": "overall code assessment",
  "issues": [
    {
      "severity": "critical|high|medium|low",
      "type": "quality|security|performance|style|documentation",
      "description": "detailed description",
      "line_number": 0,
      "recommendation": "how to fix"
    }
  ],
  "suggestions": ["suggestion 1", "suggestion 2", ...],
  "metrics": {
    "overall_score": 0-100,
    "quality_score": 0-100,
    "maintainability_score": 0-100,
    "readability_score": 0-100,
    "total_issues": 0,
    "critical_issues": 0,
    "high_issues": 0,
    "medium_issues": 0,
    "low_issues": 0
  }
}

Only return the JSON, no other text.`, code, filePath)
	}

	// Get a model from the provider
	models := a.llmProvider.GetModels()
	if len(models) == 0 {
		return "", nil, nil, nil, fmt.Errorf("no models available from provider")
	}

	request := &llm.LLMRequest{
		Model:       models[0].Name,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4000,
		Temperature: 0.3, // Low temperature for consistent reviews
	}

	response, err := a.llmProvider.Generate(ctx, request)
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("failed to perform review: %w", err)
	}

	if response.Content == "" {
		return "", nil, nil, nil, fmt.Errorf("no review generated")
	}

	// Parse JSON response
	var reviewResponse struct {
		ReviewSummary string                   `json:"review_summary"`
		Issues        []map[string]interface{} `json:"issues"`
		Suggestions   []string                 `json:"suggestions"`
		Metrics       map[string]interface{}   `json:"metrics"`
	}

	if err := json.Unmarshal([]byte(response.Content), &reviewResponse); err != nil {
		return "", nil, nil, nil, fmt.Errorf("failed to parse review response: %w", err)
	}

	return reviewResponse.ReviewSummary, reviewResponse.Issues, reviewResponse.Suggestions, reviewResponse.Metrics, nil
}

// readFile reads a file using FSRead tool
func (a *ReviewAgent) readFile(ctx context.Context, filePath string) (string, error) {
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

// runStaticAnalysis runs static analysis tools on the code
func (a *ReviewAgent) runStaticAnalysis(ctx context.Context, filePath string) (map[string]interface{}, error) {
	tool, err := a.toolRegistry.Get("Shell")
	if err != nil {
		return nil, fmt.Errorf("failed to get Shell tool: %w", err)
	}

	// Determine static analysis commands based on file type
	commands := a.determineStaticAnalysisCommands(filePath)

	results := make(map[string]interface{})
	for name, cmd := range commands {
		params := map[string]interface{}{
			"command": cmd,
			"timeout": 30000, // 30 second timeout
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

// determineStaticAnalysisCommands determines which static analysis commands to run
func (a *ReviewAgent) determineStaticAnalysisCommands(filePath string) map[string]string {
	commands := make(map[string]string)

	// For Go files
	if len(filePath) > 3 && filePath[len(filePath)-3:] == ".go" {
		dir := getTestDirectory(filePath)
		commands["go_vet"] = fmt.Sprintf("go vet %s", dir)
		commands["staticcheck"] = fmt.Sprintf("staticcheck %s", dir)
		commands["golint"] = fmt.Sprintf("golint %s", filePath)
	}

	// Add more language-specific static analysis tools as needed

	return commands
}

// Collaborate allows this agent to work with other agents
func (a *ReviewAgent) Collaborate(ctx context.Context, agents []agent.Agent, t *task.Task) (*agent.CollaborationResult, error) {
	result := &agent.CollaborationResult{
		Success:      true,
		Results:      make(map[string]*task.Result),
		Participants: []string{a.ID()},
		Messages:     []*agent.CollaborationMessage{},
	}

	startTime := time.Now()

	// Execute our own review task
	myResult, err := a.Execute(ctx, t)
	if err != nil {
		result.Success = false
		return result, err
	}
	result.Results[a.ID()] = myResult

	// Check if there are refactoring agents to address issues
	issues, ok := myResult.Output["issues"].([]map[string]interface{})
	if ok && len(issues) > 0 {
		for _, other := range agents {
			if other.Type() == agent.AgentTypeRefactoring {
				// Create a refactoring task for critical/high issues
				criticalIssues := make([]map[string]interface{}, 0)
				for _, issue := range issues {
					severity, _ := issue["severity"].(string)
					if severity == "critical" || severity == "high" {
						criticalIssues = append(criticalIssues, issue)
					}
				}

				if len(criticalIssues) > 0 {
					refactorTask := task.NewTask(
						task.TaskTypeRefactoring,
						"Address Review Issues",
						"Refactor code to address critical review findings",
						task.PriorityHigh,
					)
					refactorTask.Input = map[string]interface{}{
						"file_path": myResult.Output["file_path"],
						"issues":    criticalIssues,
					}

					refactorResult, err := other.Execute(ctx, refactorTask)
					if err != nil {
						continue // Skip failed refactoring
					}

					result.Results[other.ID()] = refactorResult
					result.Participants = append(result.Participants, other.ID())

					// Add collaboration message
					msg := &agent.CollaborationMessage{
						ID:        fmt.Sprintf("msg-%d", time.Now().Unix()),
						From:      a.ID(),
						To:        other.ID(),
						Type:      agent.MessageTypeRequest,
						Content:   fmt.Sprintf("Please address %d critical/high issues", len(criticalIssues)),
						Timestamp: time.Now(),
					}
					result.Messages = append(result.Messages, msg)
				}
			}
		}
	}

	// Use our result as consensus
	result.Consensus = myResult
	result.Duration = time.Since(startTime)

	return result, nil
}

// Shutdown cleanly shuts down the agent
func (a *ReviewAgent) Shutdown(ctx context.Context) error {
	a.SetStatus(agent.StatusShutdown)
	return nil
}

// Helper functions are in utils.go
