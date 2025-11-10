package types

import (
	"context"
	"testing"
	"time"

	"dev.helix.code/internal/agent"
	"dev.helix.code/internal/agent/task"
	"dev.helix.code/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMProvider is a simple mock for testing
type MockLLMProvider struct {
	models       []llm.ModelInfo
	generateFunc func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error)
}

func (m *MockLLMProvider) GetType() llm.ProviderType {
	return llm.ProviderType("mock")
}

func (m *MockLLMProvider) GetName() string {
	return "mock"
}

func (m *MockLLMProvider) GetModels() []llm.ModelInfo {
	if m.models == nil {
		return []llm.ModelInfo{{Name: "test-model", Provider: "test"}}
	}
	return m.models
}

func (m *MockLLMProvider) GetCapabilities() []llm.ModelCapability {
	return []llm.ModelCapability{}
}

func (m *MockLLMProvider) Generate(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, request)
	}
	return &llm.LLMResponse{Content: "test response"}, nil
}

func (m *MockLLMProvider) GenerateStream(ctx context.Context, request *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	return nil
}

func (m *MockLLMProvider) IsAvailable(ctx context.Context) bool {
	return true
}

func (m *MockLLMProvider) GetHealth(ctx context.Context) (*llm.ProviderHealth, error) {
	return &llm.ProviderHealth{Status: "healthy"}, nil
}

func (m *MockLLMProvider) Close() error {
	return nil
}

// TestNewPlanningAgent tests planning agent creation
func TestNewPlanningAgent(t *testing.T) {
	t.Run("Valid creation", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "test-planning-agent",
			Type: agent.AgentTypePlanning,
			Name: "Test Planning Agent",
		}
		provider := &MockLLMProvider{}

		planningAgent, err := NewPlanningAgent(config, provider)
		require.NoError(t, err)
		require.NotNil(t, planningAgent)
		assert.Equal(t, "test-planning-agent", planningAgent.ID())
		assert.Equal(t, agent.AgentTypePlanning, planningAgent.Type())
	})

	t.Run("Nil provider", func(t *testing.T) {
		config := &agent.AgentConfig{
			ID:   "test-planning-agent",
			Type: agent.AgentTypePlanning,
			Name: "Test Planning Agent",
		}

		agent, err := NewPlanningAgent(config, nil)
		assert.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "LLM provider is required")
	})
}

// TestPlanningAgentInitialize tests agent initialization
func TestPlanningAgentInitialize(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}
	provider := &MockLLMProvider{}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	ctx := context.Background()
	err = planningAgent.Initialize(ctx, config)
	require.NoError(t, err)

	// Check status is set to idle
	assert.Equal(t, agent.StatusIdle, planningAgent.Status())
}

// TestPlanningAgentShutdown tests agent shutdown
func TestPlanningAgentShutdown(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}
	provider := &MockLLMProvider{}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	ctx := context.Background()
	err = planningAgent.Shutdown(ctx)
	require.NoError(t, err)

	// Check status is set to shutdown
	assert.Equal(t, agent.StatusShutdown, planningAgent.Status())
}

// TestEstimateDuration tests the duration estimation method
func TestEstimateDuration(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}
	provider := &MockLLMProvider{}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	tests := []struct {
		name     string
		subtasks []*task.Task
		expected time.Duration
	}{
		{
			name:     "Empty subtasks",
			subtasks: []*task.Task{},
			expected: 0,
		},
		{
			name: "Single task",
			subtasks: []*task.Task{
				{EstimatedDuration: 10 * time.Minute},
			},
			expected: 12 * time.Minute, // 10 * 1.2
		},
		{
			name: "Multiple tasks",
			subtasks: []*task.Task{
				{EstimatedDuration: 10 * time.Minute},
				{EstimatedDuration: 20 * time.Minute},
				{EstimatedDuration: 30 * time.Minute},
			},
			expected: 72 * time.Minute, // 60 * 1.2
		},
		{
			name: "Tasks with zero duration",
			subtasks: []*task.Task{
				{EstimatedDuration: 0},
				{EstimatedDuration: 10 * time.Minute},
			},
			expected: 12 * time.Minute, // 10 * 1.2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := planningAgent.estimateDuration(tt.subtasks)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCreateTaskFromData tests task creation from parsed data
func TestCreateTaskFromData(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}
	provider := &MockLLMProvider{}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	t.Run("Complete task data", func(t *testing.T) {
		data := map[string]interface{}{
			"title":                      "Implement feature X",
			"description":                "Add new feature",
			"type":                       "code_generation",
			"priority":                   float64(3), // High priority
			"estimated_duration_minutes": float64(30),
			"required_capabilities":      []interface{}{"code_generation", "testing"},
			"depends_on":                 []interface{}{"task-1", "task-2"},
		}

		createdTask := planningAgent.createTaskFromData(data)
		require.NotNil(t, createdTask)
		assert.Equal(t, "Implement feature X", createdTask.Title)
		assert.Equal(t, "Add new feature", createdTask.Description)
		assert.Equal(t, task.TaskType("code_generation"), createdTask.Type)
		assert.Equal(t, task.Priority(3), createdTask.Priority)
		assert.Equal(t, 30*time.Minute, createdTask.EstimatedDuration)
		assert.Equal(t, []string{"code_generation", "testing"}, createdTask.RequiredCapabilities)
		assert.Equal(t, []string{"task-1", "task-2"}, createdTask.DependsOn)
		assert.Equal(t, planningAgent.ID(), createdTask.CreatedBy)
	})

	t.Run("Minimal task data", func(t *testing.T) {
		data := map[string]interface{}{
			"title":       "Simple task",
			"description": "Do something",
		}

		createdTask := planningAgent.createTaskFromData(data)
		require.NotNil(t, createdTask)
		assert.Equal(t, "Simple task", createdTask.Title)
		assert.Equal(t, "Do something", createdTask.Description)
		assert.Equal(t, planningAgent.ID(), createdTask.CreatedBy)
	})

	t.Run("Priority clamping - too low", func(t *testing.T) {
		data := map[string]interface{}{
			"title":       "Task",
			"description": "Desc",
			"priority":    float64(0), // Below minimum
		}

		createdTask := planningAgent.createTaskFromData(data)
		assert.Equal(t, task.PriorityNormal, createdTask.Priority)
	})

	t.Run("Priority clamping - too high", func(t *testing.T) {
		data := map[string]interface{}{
			"title":       "Task",
			"description": "Desc",
			"priority":    float64(10), // Above maximum
		}

		createdTask := planningAgent.createTaskFromData(data)
		assert.Equal(t, task.PriorityCritical, createdTask.Priority)
	})

	t.Run("Empty capabilities array", func(t *testing.T) {
		data := map[string]interface{}{
			"title":                 "Task",
			"description":           "Desc",
			"required_capabilities": []interface{}{},
		}

		createdTask := planningAgent.createTaskFromData(data)
		assert.NotNil(t, createdTask.RequiredCapabilities)
		assert.Empty(t, createdTask.RequiredCapabilities)
	})

	t.Run("Empty dependencies array", func(t *testing.T) {
		data := map[string]interface{}{
			"title":       "Task",
			"description": "Desc",
			"depends_on":  []interface{}{},
		}

		createdTask := planningAgent.createTaskFromData(data)
		assert.NotNil(t, createdTask.DependsOn)
		assert.Empty(t, createdTask.DependsOn)
	})
}

// TestEstimateDurationEdgeCases tests edge cases for duration estimation
func TestEstimateDurationEdgeCases(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}
	provider := &MockLLMProvider{}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	t.Run("Nil subtasks slice", func(t *testing.T) {
		result := planningAgent.estimateDuration(nil)
		assert.Equal(t, time.Duration(0), result)
	})

	t.Run("Very large duration", func(t *testing.T) {
		subtasks := []*task.Task{
			{EstimatedDuration: 24 * time.Hour},
		}
		result := planningAgent.estimateDuration(subtasks)
		expected := time.Duration(float64(24*time.Hour) * 1.2)
		assert.Equal(t, expected, result)
	})

	t.Run("Many small tasks", func(t *testing.T) {
		subtasks := make([]*task.Task, 100)
		for i := range subtasks {
			subtasks[i] = &task.Task{EstimatedDuration: 1 * time.Minute}
		}
		result := planningAgent.estimateDuration(subtasks)
		expected := time.Duration(float64(100*time.Minute) * 1.2)
		assert.Equal(t, expected, result)
	})
}

// TestPlanningAgentExecuteSuccess tests successful plan generation and execution
func TestPlanningAgentExecuteSuccess(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			// First call: generate plan
			if request.MaxTokens == 2000 {
				return &llm.LLMResponse{
					Content: `Analysis: This is a test plan
					
Key Decisions:
- Use Go for implementation
- Apply TDD methodology

Subtasks:
1. Setup project structure
2. Implement core logic
3. Write tests`,
				}, nil
			}
			// Second call: parse subtasks
			return &llm.LLMResponse{
				Content: `[
					{
						"title": "Setup project structure",
						"description": "Create directories and files",
						"type": "planning",
						"priority": 3,
						"estimated_duration_minutes": 30,
						"depends_on": [],
						"required_capabilities": ["planning"]
					},
					{
						"title": "Implement core logic",
						"description": "Write main functionality",
						"type": "code_generation",
						"priority": 3,
						"estimated_duration_minutes": 120,
						"depends_on": ["Setup project structure"],
						"required_capabilities": ["code_generation"]
					}
				]`,
			}, nil
		},
	}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypePlanning,
		"Create Implementation Plan",
		"Plan for new feature implementation",
		task.PriorityHigh,
	)
	testTask.Input = map[string]interface{}{
		"requirements": "Build a new user authentication system with JWT tokens",
	}

	result, err := planningAgent.Execute(ctx, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "plan")
	assert.Contains(t, result.Output, "subtasks")
	assert.Contains(t, result.Output, "total_tasks")
	assert.Contains(t, result.Output, "estimated_duration")

	subtasks := result.Output["subtasks"].([]*task.Task)
	assert.Len(t, subtasks, 2)
	assert.Equal(t, "Setup project structure", subtasks[0].Title)
	assert.Equal(t, "Implement core logic", subtasks[1].Title)
}

// TestPlanningAgentExecuteMissingRequirements tests error when requirements are missing
func TestPlanningAgentExecuteMissingRequirements(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}
	provider := &MockLLMProvider{}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypePlanning,
		"Test Task",
		"Test",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"other_field": "value",
	}

	result, err := planningAgent.Execute(ctx, testTask)
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "requirements not found")

	health := planningAgent.Health()
	assert.Equal(t, 1, health.ErrorCount)
}

// TestPlanningAgentExecuteGeneratePlanError tests error during plan generation
func TestPlanningAgentExecuteGeneratePlanError(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}

	provider := &MockLLMProvider{
		models: []llm.ModelInfo{}, // No models available
	}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypePlanning,
		"Test Task",
		"Test",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"requirements": "Build something",
	}

	result, err := planningAgent.Execute(ctx, testTask)
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "no models available")

	health := planningAgent.Health()
	assert.Equal(t, 1, health.ErrorCount)
}

// TestPlanningAgentExecuteParseSubtasksError tests error during subtask parsing
func TestPlanningAgentExecuteParseSubtasksError(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}

	callCount := 0
	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			callCount++
			// First call: generate plan (succeeds)
			if callCount == 1 {
				return &llm.LLMResponse{
					Content: "Test plan content",
				}, nil
			}
			// Second call: parse subtasks (returns invalid JSON)
			return &llm.LLMResponse{
				Content: "Not valid JSON",
			}, nil
		},
	}

	planningAgent, err := NewPlanningAgent(config, provider)
	require.NoError(t, err)

	ctx := context.Background()
	testTask := task.NewTask(
		task.TaskTypePlanning,
		"Test Task",
		"Test",
		task.PriorityNormal,
	)
	testTask.Input = map[string]interface{}{
		"requirements": "Build something",
	}

	result, err := planningAgent.Execute(ctx, testTask)
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "failed to parse subtasks JSON")

	health := planningAgent.Health()
	assert.Equal(t, 1, health.ErrorCount)
}

// TestPlanningAgentCollaborate tests collaboration with other agents
func TestPlanningAgentCollaborate(t *testing.T) {
	config := &agent.AgentConfig{
		ID:   "test-planning-agent",
		Type: agent.AgentTypePlanning,
		Name: "Test Planning Agent",
	}

	provider := &MockLLMProvider{
		generateFunc: func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
			if request.MaxTokens == 2000 {
				return &llm.LLMResponse{
					Content: "Test plan with collaboration",
				}, nil
			}
			return &llm.LLMResponse{
				Content: `[{"title": "Task 1", "description": "Desc", "type": "planning", "priority": 2, "estimated_duration_minutes": 60}]`,
			}, nil
		},
	}

	planningAgent, err := NewPlanningAgent(config, provider)
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
		task.TaskTypePlanning,
		"Collaborative Planning",
		"Plan with input from coding agent",
		task.PriorityHigh,
	)
	testTask.Input = map[string]interface{}{
		"requirements": "Design system architecture",
	}

	result, err := planningAgent.Collaborate(ctx, []agent.Agent{codingAgent}, testTask)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Participants, planningAgent.ID())
	// Note: coding agent won't be in participants since Collaborate only includes other planning agents
	assert.NotNil(t, result.Consensus)
}
