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
// MockTool for testing tool interactions
type MockTool struct {
	name        string
	executeFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return "Mock tool for testing"
}

func (m *MockTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{}
}

func (m *MockTool) Category() tools.ToolCategory {
	return tools.CategoryFileSystem
}

func (m *MockTool) Validate(params map[string]interface{}) error {
	return nil
}

// MockToolRegistry for testing
type MockToolRegistry struct {
	tools map[string]tools.Tool
}

func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		tools: make(map[string]tools.Tool),
	}
}

func (m *MockToolRegistry) Register(tool tools.Tool) error {
	m.tools[tool.Name()] = tool
	return nil
}

func (m *MockToolRegistry) Get(name string) (tools.Tool, error) {
	tool, exists := m.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	return tool, nil
}

// TestDebuggingAgentReadFile tests the readFile helper function
func TestDebuggingAgentReadFile(t *testing.T) {
	t.Run("Successful file read", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{}
		
		// Create mock tool registry with FSRead tool
		mockRegistry := NewMockToolRegistry()
		fsReadTool := &MockTool{
			name: "FSRead",
			executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return "file content here", nil
			},
		}
		mockRegistry.Register(fsReadTool)

		debuggingAgent := &DebuggingAgent{
			BaseAgent:    agent.NewBaseAgent(config),
			llmProvider:  provider,
			toolRegistry: (*tools.ToolRegistry)(unsafe.Pointer(mockRegistry)),
		}

		ctx := context.Background()
		content, err := debuggingAgent.readFile(ctx, "/path/to/file.go")
		require.NoError(t, err)
		assert.Equal(t, "file content here", content)
	})

	t.Run("Tool not found error", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{}
		mockRegistry := NewMockToolRegistry()

		debuggingAgent := &DebuggingAgent{
			BaseAgent:    agent.NewBaseAgent(config),
			llmProvider:  provider,
			toolRegistry: (*tools.ToolRegistry)(unsafe.Pointer(mockRegistry)),
		}

		ctx := context.Background()
		_, err := debuggingAgent.readFile(ctx, "/path/to/file.go")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get FSRead tool")
	})
}

//TestDebuggingAgentRunDiagnostics tests the runDiagnostics helper function
func TestDebuggingAgentRunDiagnostics(t *testing.T) {
	t.Run("Successful diagnostics", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{}
		
		mockRegistry := NewMockToolRegistry()
		shellTool := &MockTool{
			name: "Shell",
			executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return "command output", nil
			},
		}
		mockRegistry.Register(shellTool)

		debuggingAgent := &DebuggingAgent{
			BaseAgent:    agent.NewBaseAgent(config),
			llmProvider:  provider,
			toolRegistry: (*tools.ToolRegistry)(unsafe.Pointer(mockRegistry)),
		}

		ctx := context.Background()
		results, err := debuggingAgent.runDiagnostics(ctx, "test.go", "compile error")
		require.NoError(t, err)
		assert.NotNil(t, results)
		assert.Contains(t, results, "go_vet")
		assert.Contains(t, results, "go_build")
		assert.Contains(t, results, "go_test")
	})

	t.Run("Shell tool not found", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{}
		mockRegistry := NewMockToolRegistry()

		debuggingAgent := &DebuggingAgent{
			BaseAgent:    agent.NewBaseAgent(config),
			llmProvider:  provider,
			toolRegistry: (*tools.ToolRegistry)(unsafe.Pointer(mockRegistry)),
		}

		ctx := context.Background()
		_, err := debuggingAgent.runDiagnostics(ctx, "test.go", "error")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get Shell tool")
	})
}

// TestDebuggingAgentApplyFix tests the applyFix helper function
func TestDebuggingAgentApplyFix(t *testing.T) {
	t.Run("Successful fix application", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{
			generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
				return &llm.LLMResponse{
					Content: `{"fixed_code": "fixed code content"}`,
				}, nil
			},
		}
		
		mockRegistry := NewMockToolRegistry()
		fsReadTool := &MockTool{
			name: "FSRead",
			executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return "original code", nil
			},
		}
		fsWriteTool := &MockTool{
			name: "FSWrite",
			executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return nil, nil
			},
		}
		mockRegistry.Register(fsReadTool)
		mockRegistry.Register(fsWriteTool)

		debuggingAgent := &DebuggingAgent{
			BaseAgent:    agent.NewBaseAgent(config),
			llmProvider:  provider,
			toolRegistry: (*tools.ToolRegistry)(unsafe.Pointer(mockRegistry)),
		}

		ctx := context.Background()
		result, err := debuggingAgent.applyFix(ctx, "test.go", "fix the bug")
		require.NoError(t, err)
		assert.Equal(t, "success", result["status"])
		assert.Equal(t, "test.go", result["file_path"])
	})

	t.Run("Empty file path", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{}
		mockRegistry := NewMockToolRegistry()

		debuggingAgent := &DebuggingAgent{
			BaseAgent:    agent.NewBaseAgent(config),
			llmProvider:  provider,
			toolRegistry: (*tools.ToolRegistry)(unsafe.Pointer(mockRegistry)),
		}

		ctx := context.Background()
		result, err := debuggingAgent.applyFix(ctx, "", "fix")
		require.NoError(t, err)
		assert.Equal(t, "skipped", result["status"])
	})
}

// TestDebuggingAgentGenerateFixedCode tests the generateFixedCode helper function
func TestDebuggingAgentGenerateFixedCode(t *testing.T) {
	t.Run("Successful code generation", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{
			generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
				return &llm.LLMResponse{
					Content: `{"fixed_code": "corrected code"}`,
				}, nil
			},
		}
		
		mockRegistry := NewMockToolRegistry()
		fsReadTool := &MockTool{
			name: "FSRead",
			executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return "original code", nil
			},
		}
		mockRegistry.Register(fsReadTool)

		debuggingAgent := &DebuggingAgent{
			BaseAgent:    agent.NewBaseAgent(config),
			llmProvider:  provider,
			toolRegistry: (*tools.ToolRegistry)(unsafe.Pointer(mockRegistry)),
		}

		ctx := context.Background()
		fixedCode, err := debuggingAgent.generateFixedCode(ctx, "test.go", "apply fix")
		require.NoError(t, err)
		assert.Equal(t, "corrected code", fixedCode)
	})

	t.Run("No models available", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "debugging-1",
			Type: agent.AgentTypeDebugging,
			Name: "Test Debugging Agent",
		}
		provider := &MockLLMProvider{
			models: []llm.ModelInfo{}, // No models
		}
		
		mockRegistry := NewMockToolRegistry()
		fsReadTool := &MockTool{
			name: "FSRead",
			executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return "original code", nil
			},
		}
		mockRegistry.Register(fsReadTool)

		debuggingAgent := &DebuggingAgent{
			BaseAgent:    agent.NewBaseAgent(config),
			llmProvider:  provider,
			toolRegistry: (*tools.ToolRegistry)(unsafe.Pointer(mockRegistry)),
		}

		ctx := context.Background()
		_, err := debuggingAgent.generateFixedCode(ctx, "test.go", "apply fix")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no models available")
	})
}
