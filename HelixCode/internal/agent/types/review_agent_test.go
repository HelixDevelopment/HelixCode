package types

import (
	"context"
	"testing"

	"dev.helix.code/internal/agent"
	"dev.helix.code/internal/agent/task"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewReviewAgent tests review agent creation
func TestNewReviewAgent(t *testing.T) {
	t.Run("Valid creation", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "review-1",
			Type: agent.AgentTypeReview,
			Name: "Test Review Agent",
		}
		provider := &MockLLMProvider{}
		registry, err := tools.NewToolRegistry(nil)
		require.NoError(t, err)

		reviewAgent, err := NewReviewAgent(config, provider, registry)
		require.NoError(t, err)
		require.NotNil(t, reviewAgent)
		assert.Equal(t, "review-1", reviewAgent.ID())
		assert.Equal(t, agent.AgentTypeReview, reviewAgent.Type())
	})

	t.Run("Nil provider", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "review-1",
			Type: agent.AgentTypeReview,
			Name: "Test Review Agent",
		}
		registry, err := tools.NewToolRegistry(nil)
		require.NoError(t, err)

		agent, err := NewReviewAgent(config, nil, registry)
		assert.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "LLM provider is required")
	})

	t.Run("Nil tool registry", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "review-1",
			Type: agent.AgentTypeReview,
			Name: "Test Review Agent",
		}
		provider := &MockLLMProvider{}

		agent, err := NewReviewAgent(config, provider, nil)
		assert.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "tool registry is required")
	})
}

// TestReviewAgentInitialize tests agent initialization
func TestReviewAgentInitialize(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}
	provider := &MockLLMProvider{}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	err = reviewAgent.Initialize(ctx, config)
	require.NoError(t, err)

	assert.Equal(t, agent.StatusIdle, reviewAgent.Status())
}

// TestReviewAgentShutdown tests agent shutdown
func TestReviewAgentShutdown(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}
	provider := &MockLLMProvider{}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	err = reviewAgent.Shutdown(ctx)
	require.NoError(t, err)

	assert.Equal(t, agent.StatusShutdown, reviewAgent.Status())
}

// TestReviewAgentExecuteWithCode tests basic code review with inline code
func TestReviewAgentExecuteWithCode(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			return &llm.LLMResponse{
				Content: `{"review_result": "Code looks good", "issues": [{"severity": "low", "line": 10, "message": "Consider adding error handling"}], "suggestions": ["Add unit tests"], "metrics": {"complexity": 5, "maintainability": 8}}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Review Code",
		"Review function implementation",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"code": "func add(a, b int) int { return a + b }",
	}

	result, err := reviewAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "review_result")
	assert.Contains(t, result.Output, "issues")
	assert.Contains(t, result.Output, "suggestions")
	assert.Contains(t, result.Output, "metrics")
}

// TestReviewAgentExecuteSecurityReview tests security-focused review
func TestReviewAgentExecuteSecurityReview(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			// Verify security review type is in prompt
			assert.Contains(t, request.Messages[0].Content, "security")
			return &llm.LLMResponse{
				Content: `{"review_result": "Security concerns found", "issues": [{"severity": "high", "line": 5, "message": "SQL injection risk"}], "suggestions": ["Use parameterized queries"], "metrics": {"security_score": 6}}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Security Review",
		"Review for security vulnerabilities",
		task.PriorityHigh,
	)
	testTask.Input = map[string]interface{}{
		"code":        "db.Exec(\"SELECT * FROM users WHERE id = \" + userInput)",
		"review_type": "security",
	}

	result, err := reviewAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "security", result.Output["review_type"])
}

// TestReviewAgentExecutePerformanceReview tests performance-focused review
func TestReviewAgentExecutePerformanceReview(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			// Verify performance review type is in prompt
			assert.Contains(t, request.Messages[0].Content, "performance")
			return &llm.LLMResponse{
				Content: `{"review_result": "Performance can be improved", "issues": [{"severity": "medium", "line": 3, "message": "Inefficient loop"}], "suggestions": ["Use map for O(1) lookup"], "metrics": {"time_complexity": "O(n^2)"}}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Performance Review",
		"Review for performance issues",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"code":        "for i := 0; i < n; i++ { for j := 0; j < n; j++ { } }",
		"review_type": "performance",
	}

	result, err := reviewAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "performance", result.Output["review_type"])
}

// TestReviewAgentExecuteMissingCodeAndFile tests error when both code and file_path are missing
func TestReviewAgentExecuteMissingCodeAndFile(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}
	provider := &MockLLMProvider{}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Test Task",
		"Test",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"other_field": "value",
	}

	result, err := reviewAgent.Execute(ctx, testTask)
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "code or file_path not found")

	health := reviewAgent.Health()
	assert.Equal(t, 1, health.ErrorCount)
}

// TestReviewAgentExecuteLLMError tests LLM generation error
func TestReviewAgentExecuteLLMError(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}

	provider := &MockLLMProvider{
		models: []llm.ModelInfo{},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Test Task",
		"Test",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"code": "func test() {}",
	}

	result, err := reviewAgent.Execute(ctx, testTask)
	assert.Error(t, err)
	assert.False(t, result.Success)

	health := reviewAgent.Health()
	assert.Equal(t, 1, health.ErrorCount)
}

// TestReviewAgentExecuteWithManyIssues tests confidence adjustment for many issues
func TestReviewAgentExecuteWithManyIssues(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			// Return many issues to test confidence adjustment
			issues := make([]map[string]interface{}, 15)
			for i := 0; i < 15; i++ {
				issues[i] = map[string]interface{}{
					"severity": "medium",
					"line":     i + 1,
					"message":  "Issue " + string(rune('A'+i)),
				}
			}
			return &llm.LLMResponse{
				Content: `{"review_result": "Many issues found", "issues": [{"severity": "high", "line": 1, "message": "Issue 1"}, {"severity": "high", "line": 2, "message": "Issue 2"}, {"severity": "high", "line": 3, "message": "Issue 3"}, {"severity": "high", "line": 4, "message": "Issue 4"}, {"severity": "high", "line": 5, "message": "Issue 5"}, {"severity": "high", "line": 6, "message": "Issue 6"}, {"severity": "high", "line": 7, "message": "Issue 7"}, {"severity": "high", "line": 8, "message": "Issue 8"}, {"severity": "high", "line": 9, "message": "Issue 9"}, {"severity": "high", "line": 10, "message": "Issue 10"}, {"severity": "high", "line": 11, "message": "Issue 11"}], "suggestions": ["Fix all issues"], "metrics": {"quality": 3}}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Review Problematic Code",
		"Review code with many issues",
		task.PriorityHigh,
	)
	testTask.Input = map[string]interface{}{
		"code": "// problematic code here",
	}

	result, err := reviewAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	// Confidence should be lower (0.75) due to many issues
	assert.LessOrEqual(t, result.Confidence, 0.8)
}

// TestReviewAgentCollaborate tests collaboration with coding agents
func TestReviewAgentCollaborate(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			return &llm.LLMResponse{
				Content: `{"review_result": "Approved", "issues": [], "suggestions": ["Great work!"], "metrics": {"quality": 10}}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	// Create a mock coding agent
	codingConfig := &agent.AgentConfig{
		ID:   "coding-1",
		Type: agent.AgentTypeCoding,
		Name: "Test Coding Agent",
	}
	codingAgent := &MockCollabAgent{
		BaseAgent: agent.NewBaseAgent(codingConfig),
	}

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Review Task",
		"Review generated code",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"code": "func example() { }",
	}

	result, err := reviewAgent.Collaborate(ctx, []agent.Agent{codingAgent}, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Participants, reviewAgent.ID())
	assert.NotNil(t, result.Consensus)
}

// TestReviewAgentMetrics tests metrics recording
func TestReviewAgentMetrics(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			return &llm.LLMResponse{
				Content: `{"review_result": "Review complete", "issues": [{"severity": "low", "line": 5, "message": "Minor issue"}], "suggestions": ["Improve naming"], "metrics": {"maintainability": 7}}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Review Task",
		"Review code quality",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"code": "func process() { }",
	}

	result, err := reviewAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotNil(t, result.Metrics)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// TestReviewAgentDefaultReviewType tests default review type
func TestReviewAgentDefaultReviewType(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "review-1",
		Type: agent.AgentTypeReview,
		Name: "Test Review Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			// Verify comprehensive review is used by default
			assert.Contains(t, request.Messages[0].Content, "comprehensive")
			return &llm.LLMResponse{
				Content: `{"review_result": "Complete", "issues": [], "suggestions": [], "metrics": {}}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	reviewAgent, err := NewReviewAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeReview,
		"Review Task",
		"Default review",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"code": "func main() { }",
		// No review_type specified, should default to "comprehensive"
	}

	result, err := reviewAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "comprehensive", result.Output["review_type"])
}
