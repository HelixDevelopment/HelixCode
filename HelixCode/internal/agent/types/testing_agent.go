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

// TestingAgent is specialized in generating and executing tests
type TestingAgent struct {
	*agent.BaseAgent
	llmProvider  llm.Provider
	toolRegistry *tools.ToolRegistry
}

// NewTestingAgent creates a new testing agent
func NewTestingAgent(config *agent.AgentConfig, provider llm.Provider, toolRegistry *tools.ToolRegistry) (*TestingAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required for testing agent")
	}
	if toolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required for testing agent")
	}

	baseAgent := agent.NewBaseAgent(config)

	return &TestingAgent{
		BaseAgent:    baseAgent,
		llmProvider:  provider,
		toolRegistry: toolRegistry,
	}, nil
}

// Initialize initializes the testing agent
func (a *TestingAgent) Initialize(ctx context.Context, config *agent.AgentConfig) error {
	a.SetStatus(agent.StatusIdle)
	return nil
}

// Execute performs test generation and execution for a given task
func (a *TestingAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	a.SetStatus(agent.StatusBusy)
	defer a.SetStatus(agent.StatusIdle)

	startTime := time.Now()
	result := task.NewResult(t.ID, a.ID())

	// Increment task count
	a.IncrementTaskCount()

	// Extract code to test from task input
	codeToTest, ok := t.Input["code"].(string)
	if !ok {
		a.IncrementErrorCount()
		err := fmt.Errorf("code not found in task input")
		result.SetFailure(err)
		return result, err
	}

	filePath, _ := t.Input["file_path"].(string)
	testFramework, _ := t.Input["test_framework"].(string)
	if testFramework == "" {
		testFramework = "testing" // Default Go testing framework
	}

	// Generate tests using LLM
	testCode, testCases, err := a.generateTests(ctx, codeToTest, filePath, testFramework)
	if err != nil {
		a.IncrementErrorCount()
		result.SetFailure(err)
		return result, err
	}

	// Save test file
	testFilePath := getTestFilePath(filePath)
	artifacts, err := a.saveTestFile(ctx, testFilePath, testCode)
	if err != nil {
		a.IncrementErrorCount()
		result.SetFailure(err)
		return result, err
	}

	// Execute tests if requested
	var testResults map[string]interface{}
	executeTests, _ := t.Input["execute_tests"].(bool)
	if executeTests {
		testResults, err = a.executeTests(ctx, testFilePath)
		if err != nil {
			// Don't fail the task if tests fail, just record it
			testResults = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		}
	}

	// Set result
	output := map[string]interface{}{
		"test_file":    testFilePath,
		"test_code":    testCode,
		"test_cases":   testCases,
		"artifacts":    artifacts,
		"test_results": testResults,
	}

	result.SetSuccess(output, 0.85) // 85% confidence for test generation
	result.Duration = time.Since(startTime)
	result.Artifacts = artifacts

	// Set metrics
	result.Metrics = &task.TaskMetrics{
		FilesModified:  len(artifacts),
		TestsGenerated: len(testCases),
		LinesAdded:     countLines(testCode),
		ExecutionTime:  result.Duration,
	}

	return result, nil
}

// generateTests uses LLM to generate test cases
func (a *TestingAgent) generateTests(ctx context.Context, code, filePath, framework string) (string, []string, error) {
	prompt := fmt.Sprintf(`You are a test generation agent. Generate comprehensive tests for the following code.

Code to test:
%s

File path: %s
Test framework: %s

Please provide:
1. Complete test code with multiple test cases
2. A list of test case names

Format your response as JSON:
{
  "test_code": "complete test code with all test cases",
  "test_cases": ["TestCase1", "TestCase2", ...]
}

Generate tests that cover:
- Happy path scenarios
- Edge cases
- Error handling
- Boundary conditions

Only return the JSON, no other text.`, code, filePath, framework)

	// Get a model from the provider
	models := a.llmProvider.GetModels()
	if len(models) == 0 {
		return "", nil, fmt.Errorf("no models available from provider")
	}

	request := &llm.LLMRequest{
		Model:       models[0].Name,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4000,
		Temperature: 0.3, // Low temperature for consistent test generation
	}

	response, err := a.llmProvider.Generate(ctx, request)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate tests: %w", err)
	}

	if response.Content == "" {
		return "", nil, fmt.Errorf("no tests generated")
	}

	// Parse JSON response
	var testResponse struct {
		TestCode  string   `json:"test_code"`
		TestCases []string `json:"test_cases"`
	}

	if err := json.Unmarshal([]byte(response.Content), &testResponse); err != nil {
		return "", nil, fmt.Errorf("failed to parse test response: %w", err)
	}

	return testResponse.TestCode, testResponse.TestCases, nil
}

// saveTestFile saves the test code to a file
func (a *TestingAgent) saveTestFile(ctx context.Context, testFilePath, testCode string) ([]task.Artifact, error) {
	var artifacts []task.Artifact

	if testFilePath == "" {
		// No file path specified, return code as artifact only
		artifact := task.Artifact{
			ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
			Type:      "test",
			Path:      "generated_test",
			Content:   testCode,
			Size:      int64(len(testCode)),
			CreatedAt: time.Now(),
		}
		artifacts = append(artifacts, artifact)
		return artifacts, nil
	}

	// Use FSWrite tool to save the test file
	tool, err := a.toolRegistry.Get("FSWrite")
	if err != nil {
		return nil, fmt.Errorf("failed to get FSWrite tool: %w", err)
	}

	// Prepare tool parameters
	params := map[string]interface{}{
		"path":    testFilePath,
		"content": testCode,
	}

	// Execute tool
	_, err = tool.Execute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to write test file: %w", err)
	}

	// Create artifact
	artifact := task.Artifact{
		ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
		Type:      "test",
		Path:      testFilePath,
		Content:   testCode,
		Size:      int64(len(testCode)),
		CreatedAt: time.Now(),
	}

	artifacts = append(artifacts, artifact)

	return artifacts, nil
}

// executeTests runs the generated tests
func (a *TestingAgent) executeTests(ctx context.Context, testFilePath string) (map[string]interface{}, error) {
	// Use Shell tool to execute tests
	tool, err := a.toolRegistry.Get("Shell")
	if err != nil {
		return nil, fmt.Errorf("failed to get Shell tool: %w", err)
	}

	// Prepare test command
	// For Go tests: go test -v ./path/to/test
	testDir := getTestDirectory(testFilePath)
	params := map[string]interface{}{
		"command": fmt.Sprintf("go test -v %s", testDir),
		"timeout": 30000, // 30 second timeout
	}

	// Execute tests
	output, err := tool.Execute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tests: %w", err)
	}

	// Parse test output
	results := map[string]interface{}{
		"status":     "completed",
		"raw_output": output,
	}

	return results, nil
}

// Collaborate allows this agent to work with other agents
func (a *TestingAgent) Collaborate(ctx context.Context, agents []agent.Agent, t *task.Task) (*agent.CollaborationResult, error) {
	result := &agent.CollaborationResult{
		Success:      true,
		Results:      make(map[string]*task.Result),
		Participants: []string{a.ID()},
		Messages:     []*agent.CollaborationMessage{},
	}

	startTime := time.Now()

	// Execute our own testing task
	myResult, err := a.Execute(ctx, t)
	if err != nil {
		result.Success = false
		return result, err
	}
	result.Results[a.ID()] = myResult

	// Check if there are coding agents that generated the code
	for _, other := range agents {
		if other.Type() == agent.AgentTypeCoding {
			// We could request additional context about the code
			msg := &agent.CollaborationMessage{
				ID:        fmt.Sprintf("msg-%d", time.Now().Unix()),
				From:      a.ID(),
				To:        other.ID(),
				Type:      agent.MessageTypeRequest,
				Content:   "Tests generated for your code",
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
func (a *TestingAgent) Shutdown(ctx context.Context) error {
	a.SetStatus(agent.StatusShutdown)
	return nil
}

// Helper functions

func getTestFilePath(filePath string) string {
	if filePath == "" {
		return "generated_test.go"
	}
	// For Go: convert "file.go" to "file_test.go"
	if len(filePath) > 3 && filePath[len(filePath)-3:] == ".go" {
		return filePath[:len(filePath)-3] + "_test.go"
	}
	return filePath + "_test"
}

func getTestDirectory(testFilePath string) string {
	// Extract directory from test file path
	for i := len(testFilePath) - 1; i >= 0; i-- {
		if testFilePath[i] == '/' {
			return testFilePath[:i]
		}
	}
	return "."
}

// Helper functions are in utils.go
