package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"dev.helix.code/internal/agent/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock agent for coordinator testing
type mockCoordAgent struct {
	*BaseAgent
	executeFunc func(context.Context, *task.Task) (*task.Result, error)
}

func newMockCoordAgent(id string, agentType AgentType, caps []Capability) *mockCoordAgent {
	config := &AgentConfig{
		ID:           id,
		Type:         agentType,
		Name:         "Mock Coordinator Agent",
		Capabilities: caps,
	}
	base := NewBaseAgent(config)
	return &mockCoordAgent{
		BaseAgent: base,
	}
}

func (m *mockCoordAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	m.IncrementTaskCount()
	if m.executeFunc != nil {
		return m.executeFunc(ctx, t)
	}
	return &task.Result{
		TaskID:    t.ID,
		AgentID:   m.id,
		Success:   true,
		Output:    map[string]interface{}{"status": "completed"},
		Timestamp: time.Now(),
	}, nil
}

func (m *mockCoordAgent) Initialize(ctx context.Context, config *AgentConfig) error {
	return nil
}

func (m *mockCoordAgent) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockCoordAgent) Collaborate(ctx context.Context, agents []Agent, t *task.Task) (*CollaborationResult, error) {
	return &CollaborationResult{
		Success: true,
		Results: make(map[string]*task.Result),
	}, nil
}

// TestCoordinatorSubmitTask tests task submission
func TestCoordinatorSubmitTask(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Create a task
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)

	// Submit the task
	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Verify task is tracked
	retrievedTask, err := coordinator.GetTaskStatus(testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, testTask.ID, retrievedTask.ID)
	assert.Equal(t, task.StatusPending, retrievedTask.Status)
}

// TestCoordinatorSubmitNilTask tests submitting nil task
func TestCoordinatorSubmitNilTask(t *testing.T) {
	coordinator := NewCoordinator(nil)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestCoordinatorExecuteTask tests task execution
func TestCoordinatorExecuteTask(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register a mock agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	// Create and submit a task
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Execute the task
	result, err := coordinator.ExecuteTask(ctx, testTask.ID)

	// Verify successful execution
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, testTask.ID, result.TaskID)
	assert.Equal(t, agent.ID(), result.AgentID)
}

// TestCoordinatorExecuteTaskNotFound tests execution of non-existent task
func TestCoordinatorExecuteTaskNotFound(t *testing.T) {
	coordinator := NewCoordinator(nil)

	ctx := context.Background()
	result, err := coordinator.ExecuteTask(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// TestCoordinatorExecuteTaskNoAgent tests execution when no suitable agent exists
func TestCoordinatorExecuteTaskNoAgent(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Create task with specific requirements
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)
	testTask.RequiredCapabilities = []string{string(CapabilityCodeGeneration)}

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Execute without registering any agents
	result, err := coordinator.ExecuteTask(ctx, testTask.ID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no suitable agent")
}

// TestCoordinatorExecuteTaskAgentError tests handling of agent execution errors
func TestCoordinatorExecuteTaskAgentError(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent that returns error
	agent := newMockCoordAgent("error-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent.executeFunc = func(ctx context.Context, t *task.Task) (*task.Result, error) {
		return nil, errors.New("execution failed")
	}
	coordinator.RegisterAgent(agent)

	// Create and submit task
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Execute the task
	result, err := coordinator.ExecuteTask(ctx, testTask.ID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestCoordinatorGetTaskStatus tests task status retrieval
func TestCoordinatorGetTaskStatus(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Create and submit task
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Get task status
	retrievedTask, err := coordinator.GetTaskStatus(testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, testTask.ID, retrievedTask.ID)
	assert.Equal(t, task.StatusPending, retrievedTask.Status)
}

// TestCoordinatorGetTaskStatusNotFound tests status retrieval for non-existent task
func TestCoordinatorGetTaskStatusNotFound(t *testing.T) {
	coordinator := NewCoordinator(nil)

	retrievedTask, err := coordinator.GetTaskStatus("non-existent")
	assert.Error(t, err)
	assert.Nil(t, retrievedTask)
	assert.Contains(t, err.Error(), "not found")
}

// TestCoordinatorGetResult tests result retrieval
func TestCoordinatorGetResult(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	// Create, submit and execute task
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	result, err := coordinator.ExecuteTask(ctx, testTask.ID)
	require.NoError(t, err)

	// Get the result
	retrievedResult, err := coordinator.GetResult(testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, result.TaskID, retrievedResult.TaskID)
	assert.Equal(t, result.AgentID, retrievedResult.AgentID)
	assert.Equal(t, result.Success, retrievedResult.Success)
}

// TestCoordinatorGetResultNotFound tests result retrieval for non-existent task
func TestCoordinatorGetResultNotFound(t *testing.T) {
	coordinator := NewCoordinator(nil)

	result, err := coordinator.GetResult("non-existent")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// TestCoordinatorListAgents tests agent listing
func TestCoordinatorListAgents(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register multiple agents
	agent1 := newMockCoordAgent("agent-1", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent2 := newMockCoordAgent("agent-2", AgentTypeTesting, []Capability{CapabilityTestGeneration})
	agent3 := newMockCoordAgent("agent-3", AgentTypeDebugging, []Capability{CapabilityDebugging})

	coordinator.RegisterAgent(agent1)
	coordinator.RegisterAgent(agent2)
	coordinator.RegisterAgent(agent3)

	// List all agents
	agents := coordinator.ListAgents()
	require.Len(t, agents, 3)

	// Verify agent IDs
	ids := make([]string, len(agents))
	for i, a := range agents {
		ids[i] = a.ID()
	}
	assert.Contains(t, ids, "agent-1")
	assert.Contains(t, ids, "agent-2")
	assert.Contains(t, ids, "agent-3")
}

// TestCoordinatorGetAgentStats tests agent statistics retrieval
func TestCoordinatorGetAgentStats(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agents
	agent1 := newMockCoordAgent("agent-1", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent2 := newMockCoordAgent("agent-2", AgentTypeTesting, []Capability{CapabilityTestGeneration})

	coordinator.RegisterAgent(agent1)
	coordinator.RegisterAgent(agent2)

	// Get stats
	stats := coordinator.GetAgentStats()

	// Verify stats structure
	require.NotNil(t, stats)
	require.Contains(t, stats, "agent-1")
	require.Contains(t, stats, "agent-2")

	// Verify agent 1 stats
	agent1Stats := stats["agent-1"]
	assert.Equal(t, "agent-1", agent1Stats.AgentID)
	assert.Equal(t, AgentTypeCoding, agent1Stats.Type)
	assert.Equal(t, StatusIdle, agent1Stats.Status)
}

// TestCoordinatorShutdown tests coordinator shutdown
func TestCoordinatorShutdown(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	// Shutdown
	ctx := context.Background()
	err := coordinator.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestCoordinatorConcurrentTaskSubmission tests concurrent task submission
func TestCoordinatorConcurrentTaskSubmission(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Submit tasks concurrently
	ctx := context.Background()
	numTasks := 20
	errors := make(chan error, numTasks)

	for i := 0; i < numTasks; i++ {
		go func(taskNum int) {
			testTask := task.NewTask(
				task.TaskType("test"),
				"Concurrent Task",
				"A concurrent test task",
				task.PriorityNormal,
			)
			errors <- coordinator.SubmitTask(ctx, testTask)
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numTasks; i++ {
		err := <-errors
		if err == nil {
			successCount++
		}
	}

	// Verify all tasks submitted successfully
	assert.Equal(t, numTasks, successCount)
}

// TestCoordinatorConcurrentTaskExecution tests concurrent task execution
func TestCoordinatorConcurrentTaskExecution(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register multiple agents
	for i := 0; i < 3; i++ {
		agent := newMockCoordAgent(GenerateAgentID(AgentTypeCoding), AgentTypeCoding, []Capability{CapabilityCodeGeneration})
		coordinator.RegisterAgent(agent)
	}

	// Submit tasks
	ctx := context.Background()
	numTasks := 10
	taskIDs := make([]string, numTasks)

	for i := 0; i < numTasks; i++ {
		testTask := task.NewTask(
			task.TaskType("test"),
			"Concurrent Task",
			"A concurrent test task",
			task.PriorityNormal,
		)
		err := coordinator.SubmitTask(ctx, testTask)
		require.NoError(t, err)
		taskIDs[i] = testTask.ID
	}

	// Execute tasks concurrently
	results := make(chan *task.Result, numTasks)
	errors := make(chan error, numTasks)

	for _, taskID := range taskIDs {
		go func(id string) {
			result, err := coordinator.ExecuteTask(ctx, id)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(taskID)
	}

	// Collect results
	successCount := 0
	errorCount := 0
	timeout := time.After(10 * time.Second)

	for i := 0; i < numTasks; i++ {
		select {
		case <-results:
			successCount++
		case <-errors:
			errorCount++
		case <-timeout:
			t.Fatal("Test timed out")
		}
	}

	// Verify all tasks completed
	assert.Equal(t, numTasks, successCount+errorCount)
	assert.True(t, successCount > 0, "Expected at least some tasks to succeed")
}

// TestCoordinatorContextCancellation tests handling of context cancellation
func TestCoordinatorContextCancellation(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent with slow execution
	agent := newMockCoordAgent("slow-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent.executeFunc = func(ctx context.Context, t *task.Task) (*task.Result, error) {
		select {
		case <-time.After(5 * time.Second):
			return &task.Result{
				TaskID:    t.ID,
				AgentID:   "slow-agent",
				Success:   true,
				Timestamp: time.Now(),
			}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	coordinator.RegisterAgent(agent)

	// Create and submit task
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Execute with cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := coordinator.ExecuteTask(cancelCtx, testTask.ID)

	// Verify context cancellation was handled
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestAgentRegistryList tests the List method
func TestAgentRegistryList(t *testing.T) {
	registry := NewAgentRegistry()

	// Register multiple agents
	agent1 := newMockCoordAgent("agent-1", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent2 := newMockCoordAgent("agent-2", AgentTypeTesting, []Capability{CapabilityTestGeneration})
	agent3 := newMockCoordAgent("agent-3", AgentTypeDebugging, []Capability{CapabilityDebugging})

	registry.Register(agent1)
	registry.Register(agent2)
	registry.Register(agent3)

	// List all agents
	agents := registry.List()

	// Verify all agents are returned
	require.Len(t, agents, 3)

	// Verify agent IDs
	ids := make(map[string]bool)
	for _, a := range agents {
		ids[a.ID()] = true
	}
	assert.True(t, ids["agent-1"])
	assert.True(t, ids["agent-2"])
	assert.True(t, ids["agent-3"])
}

// TestFindSuitableAgentByCapability tests agent selection by capability
func TestFindSuitableAgentByCapability(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agents with different capabilities
	agent1 := newMockCoordAgent("coding-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent2 := newMockCoordAgent("testing-agent", AgentTypeTesting, []Capability{CapabilityTestGeneration})

	coordinator.RegisterAgent(agent1)
	coordinator.RegisterAgent(agent2)

	// Create task requiring code generation
	testTask := task.NewTask(
		task.TaskType("coding"),
		"Coding Task",
		"A coding task",
		task.PriorityNormal,
	)
	testTask.RequiredCapabilities = []string{string(CapabilityCodeGeneration)}

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Execute - should select coding-agent
	result, err := coordinator.ExecuteTask(ctx, testTask.ID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "coding-agent", result.AgentID)
}

// TestFindSuitableAgentBusyAgent tests that busy agents are not selected
func TestFindSuitableAgentBusyAgent(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register busy agent
	busyAgent := newMockCoordAgent("busy-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	busyAgent.SetStatus(StatusBusy)

	// Register idle agent
	idleAgent := newMockCoordAgent("idle-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})

	coordinator.RegisterAgent(busyAgent)
	coordinator.RegisterAgent(idleAgent)

	// Create task
	testTask := task.NewTask(
		task.TaskType("coding"),
		"Coding Task",
		"A coding task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Execute - should select idle-agent, not busy-agent
	result, err := coordinator.ExecuteTask(ctx, testTask.ID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "idle-agent", result.AgentID)
}

// TestCoordinatorMultipleAgentsSameType tests task execution with multiple agents of same type
func TestCoordinatorMultipleAgentsSameType(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register three coding agents
	agent1 := newMockCoordAgent("coding-1", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent2 := newMockCoordAgent("coding-2", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent3 := newMockCoordAgent("coding-3", AgentTypeCoding, []Capability{CapabilityCodeGeneration})

	coordinator.RegisterAgent(agent1)
	coordinator.RegisterAgent(agent2)
	coordinator.RegisterAgent(agent3)

	// Create task
	testTask := task.NewTask(
		task.TaskType("coding"),
		"Coding Task",
		"A coding task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Execute - should select one of the agents
	result, err := coordinator.ExecuteTask(ctx, testTask.ID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Should be one of our registered agents
	validAgents := []string{"coding-1", "coding-2", "coding-3"}
	assert.Contains(t, validAgents, result.AgentID)
}

// TestCoordinatorTaskPriority tests task priority handling
func TestCoordinatorTaskPriority(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	ctx := context.Background()

	// Submit tasks with different priorities
	lowPriorityTask := task.NewTask(
		task.TaskType("test"),
		"Low Priority",
		"Low priority task",
		task.PriorityLow,
	)

	highPriorityTask := task.NewTask(
		task.TaskType("test"),
		"High Priority",
		"High priority task",
		task.PriorityHigh,
	)

	criticalTask := task.NewTask(
		task.TaskType("test"),
		"Critical",
		"Critical task",
		task.PriorityCritical,
	)

	err := coordinator.SubmitTask(ctx, lowPriorityTask)
	require.NoError(t, err)

	err = coordinator.SubmitTask(ctx, highPriorityTask)
	require.NoError(t, err)

	err = coordinator.SubmitTask(ctx, criticalTask)
	require.NoError(t, err)

	// All tasks should be submitted successfully
	retrievedLow, err := coordinator.GetTaskStatus(lowPriorityTask.ID)
	require.NoError(t, err)
	assert.Equal(t, task.PriorityLow, retrievedLow.Priority)

	retrievedHigh, err := coordinator.GetTaskStatus(highPriorityTask.ID)
	require.NoError(t, err)
	assert.Equal(t, task.PriorityHigh, retrievedHigh.Priority)

	retrievedCritical, err := coordinator.GetTaskStatus(criticalTask.ID)
	require.NoError(t, err)
	assert.Equal(t, task.PriorityCritical, retrievedCritical.Priority)
}

// TestCoordinatorCircuitBreaker tests circuit breaker state management
func TestCoordinatorCircuitBreaker(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	// Get initial circuit breaker state (may be nil if not yet tracked)
	state := coordinator.GetCircuitBreakerState(agent.ID())
	// State is returned even if agent hasn't been used yet
	_ = state

	// Get all circuit breaker stats (may be empty initially)
	stats := coordinator.GetCircuitBreakerStats()
	assert.NotNil(t, stats)
}

// TestCoordinatorAgentTaskCount tests tracking agent task counts
func TestCoordinatorAgentTaskCount(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	// Submit and execute a task
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Execute task
	_, err = coordinator.ExecuteTask(ctx, testTask.ID)
	require.NoError(t, err)

	// Verify task count increased
	stats := coordinator.GetAgentStats()
	agentStats := stats["test-agent"]
	assert.Equal(t, 1, agentStats.TaskCount)
}

// TestCoordinatorListAgentsByCapability tests finding agents by capability
func TestCoordinatorListAgentsByCapability(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agents with different capabilities
	agent1 := newMockCoordAgent("agent-1", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	agent2 := newMockCoordAgent("agent-2", AgentTypeTesting, []Capability{CapabilityTestGeneration})

	coordinator.RegisterAgent(agent1)
	coordinator.RegisterAgent(agent2)

	// List all agents
	allAgents := coordinator.ListAgents()
	require.Len(t, allAgents, 2)

	// Verify agent properties
	foundAgent1 := false
	for _, a := range allAgents {
		if a.ID() == "agent-1" {
			foundAgent1 = true
			assert.Equal(t, AgentTypeCoding, a.Type())
		}
	}
	assert.True(t, foundAgent1, "agent-1 not found in list")
}

// TestCoordinatorTaskStatusTransitions tests task status through lifecycle
func TestCoordinatorTaskStatusTransitions(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	// Create and submit task
	testTask := task.NewTask(
		task.TaskType("test"),
		"Test Task",
		"A test task",
		task.PriorityNormal,
	)

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Initial status should be Pending
	retrieved, err := coordinator.GetTaskStatus(testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, task.StatusPending, retrieved.Status)

	// Execute task
	result, err := coordinator.ExecuteTask(ctx, testTask.ID)
	require.NoError(t, err)

	// After execution, task should be completed
	retrieved, err = coordinator.GetTaskStatus(testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, task.StatusCompleted, retrieved.Status)

	// Result should be available
	assert.True(t, result.Success)
}

// TestCoordinatorEmptyAgentRegistry tests behavior with no registered agents
func TestCoordinatorEmptyAgentRegistry(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// List should be empty
	agents := coordinator.ListAgents()
	assert.Len(t, agents, 0)

	// Stats should be empty
	stats := coordinator.GetAgentStats()
	assert.Len(t, stats, 0)

	// Workflows should be empty
	workflows := coordinator.ListWorkflows()
	assert.Len(t, workflows, 0)
}

// TestCoordinatorMultipleCapabilities tests agent with multiple capabilities
func TestCoordinatorMultipleCapabilities(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent with multiple capabilities
	multiCapAgent := newMockCoordAgent(
		"multi-cap-agent",
		AgentTypeCoding,
		[]Capability{
			CapabilityCodeGeneration,
			CapabilityRefactoring,
			CapabilityTestGeneration,
		},
	)
	coordinator.RegisterAgent(multiCapAgent)

	// Create task requiring code generation
	testTask := task.NewTask(
		task.TaskType("coding"),
		"Coding Task",
		"A coding task",
		task.PriorityNormal,
	)
	testTask.RequiredCapabilities = []string{string(CapabilityCodeGeneration)}

	ctx := context.Background()
	err := coordinator.SubmitTask(ctx, testTask)
	require.NoError(t, err)

	// Agent should be selected
	result, err := coordinator.ExecuteTask(ctx, testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, "multi-cap-agent", result.AgentID)
}

// TestCoordinatorAgentStats tests agent statistics
func TestCoordinatorAgentStats(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	// Get agent stats
	stats := coordinator.GetAgentStats()
	require.Contains(t, stats, "test-agent")

	agentStats := stats["test-agent"]
	assert.Equal(t, "test-agent", agentStats.AgentID)
	assert.Equal(t, AgentTypeCoding, agentStats.Type)
	assert.Equal(t, StatusIdle, agentStats.Status)
	assert.Equal(t, 0, agentStats.TaskCount)
	assert.Equal(t, 0, agentStats.ErrorCount)
}

// TestCoordinatorWorkflowList tests workflow listing
func TestCoordinatorWorkflowList(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent
	agent := newMockCoordAgent("test-agent", AgentTypeCoding, []Capability{CapabilityCodeGeneration})
	coordinator.RegisterAgent(agent)

	// Initially no workflows
	workflows := coordinator.ListWorkflows()
	assert.Len(t, workflows, 0)

	// Note: ExecuteWorkflow would add workflows to the list
	// but that requires more complex setup with actual agent execution
}
