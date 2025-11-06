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

// TestNewDebuggingAgent tests debugging agent creation
func TestNewDebuggingAgent(t *testing.T) {
	t.Run("Valid creation", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{}
		registry, err := tools.NewToolRegistry(nil)
		require.NoError(t, err)

		debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
		require.NoError(t, err)
		require.NotNil(t, debuggingAgent)
		assert.Equal(t, "debugging-1", debuggingAgent.ID())
		assert.Equal(t, agent.AgentTypeDebugging, debuggingAgent.Type())
	})

	t.Run("Nil provider", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		registry, err := tools.NewToolRegistry(nil)
		require.NoError(t, err)

		agent, err := NewDebuggingAgent(config, nil, registry)
		assert.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "LLM provider is required")
	})

	t.Run("Nil tool registry", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{}

		agent, err := NewDebuggingAgent(config, provider, nil)
		assert.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "tool registry is required")
	})
}

// TestDebuggingAgentInitialize tests agent initialization
func TestDebuggingAgentInitialize(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "debugging-1",
		Type: agent.AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}
	provider := &MockLLMProvider{}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	err = debuggingAgent.Initialize(ctx, config)
	require.NoError(t, err)

	assert.Equal(t, agent.StatusIdle, debuggingAgent.Status())
}

// TestDebuggingAgentShutdown tests agent shutdown
func TestDebuggingAgentShutdown(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "debugging-1",
		Type: agent.AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}
	provider := &MockLLMProvider{}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	err = debuggingAgent.Shutdown(ctx)
	require.NoError(t, err)

	assert.Equal(t, agent.StatusShutdown, debuggingAgent.Status())
}

// TestDebuggingAgentExecuteBasic tests basic error analysis
func TestDebuggingAgentExecuteBasic(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "debugging-1",
		Type: agent.AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			return &llm.LLMResponse{
				Content: `{"analysis": "Null pointer error", "root_cause": "Variable not initialized", "suggested_fixes": ["Initialize variable before use"]}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeDebugging,
		"Debug Error",
		"Analyze null pointer error",
		task.PriorityHigh,
	)
	testTask.Input = map[string]interface{}{
		"error":        "NullPointerException at line 42",
		"stack_trace":  "at main.go:42\nat app.go:15",
		"code_context": "var x *int\nfmt.Println(*x)",
	}

	result, err := debuggingAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "analysis")
	assert.Contains(t, result.Output, "root_cause")
	assert.Contains(t, result.Output, "suggested_fixes")
}

// TestDebuggingAgentExecuteMissingError tests error when error message is missing
func TestDebuggingAgentExecuteMissingError(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "debugging-1",
		Type: agent.AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}
	provider := &MockLLMProvider{}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeDebugging,
		"Test Task",
		"Test",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"other_field": "value",
	}

	result, err := debuggingAgent.Execute(ctx, testTask)
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "error message not found")

	health := debuggingAgent.Health()
	assert.Equal(t, 1, health.ErrorCount)
}

// TestDebuggingAgentExecuteLLMError tests LLM generation error
func TestDebuggingAgentExecuteLLMError(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "debugging-1",
		Type: agent.AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}

	provider := &MockLLMProvider{
		models: []llm.ModelInfo{},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeDebugging,
		"Test Task",
		"Test",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"error": "Some error occurred",
	}

	result, err := debuggingAgent.Execute(ctx, testTask)
	assert.Error(t, err)
	assert.False(t, result.Success)

	health := debuggingAgent.Health()
	assert.Equal(t, 1, health.ErrorCount)
}

// TestDebuggingAgentCollaborate tests collaboration with testing agents
func TestDebuggingAgentCollaborate(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "debugging-1",
		Type: agent.AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			return &llm.LLMResponse{
				Content: `{"analysis": "Bug fixed", "root_cause": "Logic error", "suggested_fixes": ["Fix applied"]}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
	require.NoError(t, err)

	// Create a mock testing agent
	testingConfig := &agent.AgentConfig{
		ID:   "testing-1",
		Type: agent.AgentTypeTesting,
		Name: "Test Testing Agent",
	}
	testingAgent := &MockCollabAgent{
		BaseAgent: agent.NewBaseAgent(testingConfig),
	}

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeDebugging,
		"Debug Task",
		"Fix bug",
		task.PriorityHigh,
	)
	testTask.Input = map[string]interface{}{
		"error": "Bug in code",
	}

	result, err := debuggingAgent.Collaborate(ctx, []agent.Agent{testingAgent}, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Participants, debuggingAgent.ID())
	assert.NotNil(t, result.Consensus)
}

// TestDebuggingAgentDetermineDiagnosticCommands tests diagnostic command generation
func TestDebuggingAgentDetermineDiagnosticCommands(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "debugging-1",
		Type: agent.AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}
	provider := &MockLLMProvider{}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
	require.NoError(t, err)

	t.Run("Go file", func(t *testing.T) {
		commands := debuggingAgent.determineDiagnosticCommands("internal/api/handler.go", "error")
		assert.Contains(t, commands, "go_vet")
		assert.Contains(t, commands, "go_build")
		assert.Contains(t, commands, "go_test")
	})

	t.Run("Non-Go file", func(t *testing.T) {
		commands := debuggingAgent.determineDiagnosticCommands("script.js", "error")
		assert.Empty(t, commands)
	})
}

// TestDebuggingAgentMetrics tests metrics recording
func TestDebuggingAgentMetrics(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "debugging-1",
		Type: agent.AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			return &llm.LLMResponse{
				Content: `{"analysis": "Analysis complete", "root_cause": "Bug identified", "suggested_fixes": ["Fix 1", "Fix 2"]}`,
			}, nil
		},
	}
	registry, err := tools.NewToolRegistry(nil)
	require.NoError(t, err)

	debuggingAgent, err := NewDebuggingAgent(config, provider, registry)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypeDebugging,
		"Debug Task",
		"Analyze error",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"error": "Runtime error",
	}

	result, err := debuggingAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotNil(t, result.Metrics)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}
