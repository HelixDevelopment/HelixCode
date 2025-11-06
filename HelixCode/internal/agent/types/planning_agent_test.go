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
	models         []llm.ModelInfo
	generateFunc   func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error)
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
