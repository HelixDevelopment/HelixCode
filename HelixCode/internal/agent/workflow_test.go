package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"dev.helix.code/internal/agent/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkflow(t *testing.T) {
	workflow := NewWorkflow("Test Workflow", "Test workflow description")

	assert.NotNil(t, workflow)
	assert.NotEmpty(t, workflow.ID)
	assert.Equal(t, "Test Workflow", workflow.Name)
	assert.Equal(t, "Test workflow description", workflow.Description)
	assert.Equal(t, WorkflowStatusPending, workflow.Status)
	assert.NotNil(t, workflow.Steps)
	assert.NotNil(t, workflow.Results)
	assert.Nil(t, workflow.StartedAt)
	assert.Nil(t, workflow.CompletedAt)
	assert.False(t, workflow.CreatedAt.IsZero())
}

func TestWorkflowAddStep(t *testing.T) {
	workflow := NewWorkflow("Test", "Test")

	step1 := &WorkflowStep{
		ID:        "step1",
		Name:      "First Step",
		AgentType: AgentTypePlanning,
		Input:     map[string]interface{}{"key": "value"},
	}

	step2 := &WorkflowStep{
		ID:        "step2",
		Name:      "Second Step",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"step1"},
	}

	workflow.AddStep(step1)
	workflow.AddStep(step2)

	assert.Len(t, workflow.Steps, 2)
	assert.Equal(t, "step1", workflow.Steps[0].ID)
	assert.Equal(t, "step2", workflow.Steps[1].ID)
}

func TestWorkflowStateTransitions(t *testing.T) {
	workflow := NewWorkflow("Test", "Test")

	// Test Start
	workflow.Start()
	assert.Equal(t, WorkflowStatusRunning, workflow.Status)
	assert.NotNil(t, workflow.StartedAt)
	assert.Nil(t, workflow.CompletedAt)

	// Test Complete
	workflow.Complete()
	assert.Equal(t, WorkflowStatusCompleted, workflow.Status)
	assert.NotNil(t, workflow.CompletedAt)

	// Test Fail
	workflow2 := NewWorkflow("Test2", "Test2")
	workflow2.Start()
	workflow2.Fail()
	assert.Equal(t, WorkflowStatusFailed, workflow2.Status)
	assert.NotNil(t, workflow2.CompletedAt)

	// Test Cancel
	workflow3 := NewWorkflow("Test3", "Test3")
	workflow3.Start()
	workflow3.Cancel()
	assert.Equal(t, WorkflowStatusCancelled, workflow3.Status)
	assert.NotNil(t, workflow3.CompletedAt)
}

func TestWorkflowSetGetStepResult(t *testing.T) {
	workflow := NewWorkflow("Test", "Test")

	result := &task.Result{
		TaskID:    "step1",
		AgentID:   "agent1",
		Success:   true,
		Timestamp: time.Now(),
	}

	workflow.SetStepResult("step1", result)

	retrieved, ok := workflow.GetStepResult("step1")
	assert.True(t, ok)
	assert.Equal(t, result, retrieved)

	_, ok = workflow.GetStepResult("nonexistent")
	assert.False(t, ok)
}

func TestWorkflowIsStepReady(t *testing.T) {
	workflow := NewWorkflow("Test", "Test")

	step1 := &WorkflowStep{
		ID:        "step1",
		Name:      "Step 1",
		AgentType: AgentTypePlanning,
	}

	step2 := &WorkflowStep{
		ID:        "step2",
		Name:      "Step 2",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"step1"},
	}

	step3 := &WorkflowStep{
		ID:        "step3",
		Name:      "Step 3",
		AgentType: AgentTypeTesting,
		DependsOn: []string{"step1", "step2"},
	}

	workflow.AddStep(step1)
	workflow.AddStep(step2)
	workflow.AddStep(step3)

	// Step 1 should be ready (no dependencies)
	assert.True(t, workflow.IsStepReady(step1))

	// Step 2 should not be ready (step1 not completed)
	assert.False(t, workflow.IsStepReady(step2))

	// Step 3 should not be ready (dependencies not completed)
	assert.False(t, workflow.IsStepReady(step3))

	// Complete step 1
	workflow.SetStepResult("step1", &task.Result{
		TaskID:  "step1",
		AgentID: "agent1",
		Success: true,
	})

	// Now step 2 should be ready
	assert.True(t, workflow.IsStepReady(step2))

	// Step 3 still not ready (step2 not completed)
	assert.False(t, workflow.IsStepReady(step3))

	// Complete step 2
	workflow.SetStepResult("step2", &task.Result{
		TaskID:  "step2",
		AgentID: "agent1",
		Success: true,
	})

	// Now step 3 should be ready
	assert.True(t, workflow.IsStepReady(step3))
}

func TestWorkflowIsStepReadyWithOptionalDependency(t *testing.T) {
	workflow := NewWorkflow("Test", "Test")

	optionalStep := &WorkflowStep{
		ID:        "optional",
		Name:      "Optional Step",
		AgentType: AgentTypePlanning,
		Optional:  true,
	}

	dependentStep := &WorkflowStep{
		ID:        "dependent",
		Name:      "Dependent Step",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"optional"},
	}

	workflow.AddStep(optionalStep)
	workflow.AddStep(dependentStep)

	// Mark optional step as failed
	workflow.SetStepResult("optional", &task.Result{
		TaskID:  "optional",
		AgentID: "agent1",
		Success: false,
	})

	// Dependent step should still be ready if optional dependency failed
	assert.True(t, workflow.IsStepReady(dependentStep))
}

func TestWorkflowGetReadySteps(t *testing.T) {
	workflow := NewWorkflow("Test", "Test")

	step1 := &WorkflowStep{ID: "step1", Name: "Step 1", AgentType: AgentTypePlanning}
	step2 := &WorkflowStep{ID: "step2", Name: "Step 2", AgentType: AgentTypeCoding, DependsOn: []string{"step1"}}
	step3 := &WorkflowStep{ID: "step3", Name: "Step 3", AgentType: AgentTypeTesting, DependsOn: []string{"step1"}}
	step4 := &WorkflowStep{ID: "step4", Name: "Step 4", AgentType: AgentTypeReview, DependsOn: []string{"step2", "step3"}}

	workflow.AddStep(step1)
	workflow.AddStep(step2)
	workflow.AddStep(step3)
	workflow.AddStep(step4)

	// Initially, only step1 should be ready
	ready := workflow.GetReadySteps()
	assert.Len(t, ready, 1)
	assert.Equal(t, "step1", ready[0].ID)

	// Complete step1
	workflow.SetStepResult("step1", &task.Result{TaskID: "step1", Success: true})

	// Now step2 and step3 should be ready (parallel)
	ready = workflow.GetReadySteps()
	assert.Len(t, ready, 2)
	ids := []string{ready[0].ID, ready[1].ID}
	assert.Contains(t, ids, "step2")
	assert.Contains(t, ids, "step3")

	// Complete step2 but not step3
	workflow.SetStepResult("step2", &task.Result{TaskID: "step2", Success: true})

	// Only step3 should be ready
	ready = workflow.GetReadySteps()
	assert.Len(t, ready, 1)
	assert.Equal(t, "step3", ready[0].ID)

	// Complete step3
	workflow.SetStepResult("step3", &task.Result{TaskID: "step3", Success: true})

	// Now step4 should be ready
	ready = workflow.GetReadySteps()
	assert.Len(t, ready, 1)
	assert.Equal(t, "step4", ready[0].ID)

	// Complete step4
	workflow.SetStepResult("step4", &task.Result{TaskID: "step4", Success: true})

	// No steps should be ready
	ready = workflow.GetReadySteps()
	assert.Len(t, ready, 0)
}

func TestGenerateWorkflowID(t *testing.T) {
	id1 := GenerateWorkflowID()
	time.Sleep(1 * time.Millisecond)
	id2 := GenerateWorkflowID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "workflow-")
	assert.Contains(t, id2, "workflow-")
}

// Mock agent for testing workflow executor
type mockWorkflowAgent struct {
	id           string
	agentType    AgentType
	capabilities []Capability
	executeFunc  func(context.Context, *task.Task) (*task.Result, error)
}

func (m *mockWorkflowAgent) ID() string                         { return m.id }
func (m *mockWorkflowAgent) Type() AgentType                    { return m.agentType }
func (m *mockWorkflowAgent) Name() string                       { return "Mock Agent" }
func (m *mockWorkflowAgent) Status() AgentStatus                { return StatusIdle }
func (m *mockWorkflowAgent) Capabilities() []Capability         { return m.capabilities }
func (m *mockWorkflowAgent) CanHandle(t *task.Task) bool        { return true }
func (m *mockWorkflowAgent) Health() *HealthCheck               { return &HealthCheck{} }
func (m *mockWorkflowAgent) Shutdown(ctx context.Context) error { return nil }
func (m *mockWorkflowAgent) Initialize(ctx context.Context, config *AgentConfig) error {
	return nil
}

func (m *mockWorkflowAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, t)
	}
	result := task.NewResult(t.ID, m.id)
	result.SetSuccess(map[string]interface{}{"output": "test"}, 1.0)
	return result, nil
}

func (m *mockWorkflowAgent) Collaborate(ctx context.Context, agents []Agent, t *task.Task) (*CollaborationResult, error) {
	return nil, nil
}

func TestWorkflowExecutorSimpleWorkflow(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register mock agents
	planningAgent := &mockWorkflowAgent{
		id:        "planning-agent",
		agentType: AgentTypePlanning,
	}
	codingAgent := &mockWorkflowAgent{
		id:        "coding-agent",
		agentType: AgentTypeCoding,
	}

	err := coordinator.RegisterAgent(planningAgent)
	require.NoError(t, err)
	err = coordinator.RegisterAgent(codingAgent)
	require.NoError(t, err)

	// Create workflow
	workflow := NewWorkflow("Simple Workflow", "Test simple workflow")

	step1 := &WorkflowStep{
		ID:        "plan",
		Name:      "Planning",
		AgentType: AgentTypePlanning,
		Input:     map[string]interface{}{"requirement": "test"},
	}

	step2 := &WorkflowStep{
		ID:        "code",
		Name:      "Coding",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"plan"},
	}

	workflow.AddStep(step1)
	workflow.AddStep(step2)

	// Execute workflow
	ctx := context.Background()
	err = coordinator.ExecuteWorkflow(ctx, workflow)
	require.NoError(t, err)

	// Verify workflow completed
	assert.Equal(t, WorkflowStatusCompleted, workflow.Status)
	assert.NotNil(t, workflow.StartedAt)
	assert.NotNil(t, workflow.CompletedAt)

	// Verify both steps completed
	result1, ok := workflow.GetStepResult("plan")
	assert.True(t, ok)
	assert.True(t, result1.Success)

	result2, ok := workflow.GetStepResult("code")
	assert.True(t, ok)
	assert.True(t, result2.Success)
}

func TestWorkflowExecutorParallelSteps(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register mock agents
	planningAgent := &mockWorkflowAgent{
		id:        "planning-agent",
		agentType: AgentTypePlanning,
	}
	codingAgent1 := &mockWorkflowAgent{
		id:        "coding-agent-1",
		agentType: AgentTypeCoding,
	}
	codingAgent2 := &mockWorkflowAgent{
		id:        "coding-agent-2",
		agentType: AgentTypeCoding,
	}

	coordinator.RegisterAgent(planningAgent)
	coordinator.RegisterAgent(codingAgent1)
	coordinator.RegisterAgent(codingAgent2)

	// Create workflow with parallel steps
	workflow := NewWorkflow("Parallel Workflow", "Test parallel execution")

	step1 := &WorkflowStep{
		ID:        "plan",
		Name:      "Planning",
		AgentType: AgentTypePlanning,
	}

	step2 := &WorkflowStep{
		ID:        "code-frontend",
		Name:      "Code Frontend",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"plan"},
	}

	step3 := &WorkflowStep{
		ID:        "code-backend",
		Name:      "Code Backend",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"plan"},
	}

	workflow.AddStep(step1)
	workflow.AddStep(step2)
	workflow.AddStep(step3)

	// Execute workflow
	ctx := context.Background()
	err := coordinator.ExecuteWorkflow(ctx, workflow)
	require.NoError(t, err)

	// Verify all steps completed
	assert.Equal(t, WorkflowStatusCompleted, workflow.Status)
	assert.Len(t, workflow.Results, 3)
}

func TestWorkflowExecutorOptionalStep(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register mock agents
	planningAgent := &mockWorkflowAgent{
		id:        "planning-agent",
		agentType: AgentTypePlanning,
	}

	// Agent that always fails
	failingAgent := &mockWorkflowAgent{
		id:        "failing-agent",
		agentType: AgentTypeDebugging,
		executeFunc: func(ctx context.Context, t *task.Task) (*task.Result, error) {
			return nil, fmt.Errorf("intentional failure")
		},
	}

	codingAgent := &mockWorkflowAgent{
		id:        "coding-agent",
		agentType: AgentTypeCoding,
	}

	coordinator.RegisterAgent(planningAgent)
	coordinator.RegisterAgent(failingAgent)
	coordinator.RegisterAgent(codingAgent)

	// Create workflow with optional failing step
	workflow := NewWorkflow("Optional Step Workflow", "Test optional step")

	step1 := &WorkflowStep{
		ID:        "plan",
		Name:      "Planning",
		AgentType: AgentTypePlanning,
	}

	step2 := &WorkflowStep{
		ID:        "debug",
		Name:      "Debug (Optional)",
		AgentType: AgentTypeDebugging,
		DependsOn: []string{"plan"},
		Optional:  true,
	}

	step3 := &WorkflowStep{
		ID:        "code",
		Name:      "Coding",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"debug"},
	}

	workflow.AddStep(step1)
	workflow.AddStep(step2)
	workflow.AddStep(step3)

	// Execute workflow
	ctx := context.Background()
	err := coordinator.ExecuteWorkflow(ctx, workflow)
	require.NoError(t, err)

	// Verify workflow completed despite optional step failure
	assert.Equal(t, WorkflowStatusCompleted, workflow.Status)

	// Verify step2 failed but step3 succeeded
	result2, ok := workflow.GetStepResult("debug")
	assert.True(t, ok)
	assert.False(t, result2.Success)

	result3, ok := workflow.GetStepResult("code")
	assert.True(t, ok)
	assert.True(t, result3.Success)
}

func TestWorkflowExecutorMissingAgent(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Don't register any agents
	workflow := NewWorkflow("Missing Agent Workflow", "Test missing agent")

	step := &WorkflowStep{
		ID:        "step1",
		Name:      "Step 1",
		AgentType: AgentTypePlanning,
	}

	workflow.AddStep(step)

	// Execute workflow
	ctx := context.Background()
	err := coordinator.ExecuteWorkflow(ctx, workflow)

	// Should fail because no agent available
	assert.Error(t, err)
	assert.Equal(t, WorkflowStatusFailed, workflow.Status)
}

func TestWorkflowExecutorContextCancellation(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register slow agent
	slowAgent := &mockWorkflowAgent{
		id:        "slow-agent",
		agentType: AgentTypePlanning,
		executeFunc: func(ctx context.Context, t *task.Task) (*task.Result, error) {
			time.Sleep(100 * time.Millisecond)
			return task.NewResult(t.ID, "slow-agent"), nil
		},
	}

	coordinator.RegisterAgent(slowAgent)

	workflow := NewWorkflow("Cancellable Workflow", "Test cancellation")
	workflow.AddStep(&WorkflowStep{
		ID:        "slow-step",
		Name:      "Slow Step",
		AgentType: AgentTypePlanning,
	})

	// Create context with quick timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Execute workflow
	err := coordinator.ExecuteWorkflow(ctx, workflow)

	// Should be cancelled
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, WorkflowStatusCancelled, workflow.Status)
}

func TestWorkflowExecutorInputChaining(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Agent that outputs data
	producerAgent := &mockWorkflowAgent{
		id:        "producer",
		agentType: AgentTypePlanning,
		executeFunc: func(ctx context.Context, t *task.Task) (*task.Result, error) {
			result := task.NewResult(t.ID, "producer")
			output := map[string]interface{}{
				"plan":   "test plan",
				"status": "ready",
			}
			result.SetSuccess(output, 1.0)
			return result, nil
		},
	}

	// Agent that consumes data
	var receivedInput map[string]interface{}
	consumerAgent := &mockWorkflowAgent{
		id:        "consumer",
		agentType: AgentTypeCoding,
		executeFunc: func(ctx context.Context, t *task.Task) (*task.Result, error) {
			receivedInput = t.Input
			result := task.NewResult(t.ID, "consumer")
			result.SetSuccess(map[string]interface{}{}, 1.0)
			return result, nil
		},
	}

	coordinator.RegisterAgent(producerAgent)
	coordinator.RegisterAgent(consumerAgent)

	// Create workflow
	workflow := NewWorkflow("Input Chaining Workflow", "Test input chaining")

	step1 := &WorkflowStep{
		ID:        "produce",
		Name:      "Producer",
		AgentType: AgentTypePlanning,
		Input:     map[string]interface{}{"initial": "data"},
	}

	step2 := &WorkflowStep{
		ID:        "consume",
		Name:      "Consumer",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"produce"},
		Input:     map[string]interface{}{"extra": "field"},
	}

	workflow.AddStep(step1)
	workflow.AddStep(step2)

	// Execute workflow
	ctx := context.Background()
	err := coordinator.ExecuteWorkflow(ctx, workflow)
	require.NoError(t, err)

	// Verify consumer received producer's output
	assert.NotNil(t, receivedInput)
	assert.Equal(t, "test plan", receivedInput["plan"])
	assert.Equal(t, "ready", receivedInput["status"])
	assert.Equal(t, "field", receivedInput["extra"])
}

func TestWorkflowExecutorGetWorkflow(t *testing.T) {
	coordinator := NewCoordinator(nil)
	executor := coordinator.workflowExecutor

	workflow := NewWorkflow("Test", "Test")

	// Register workflow
	executor.mu.Lock()
	executor.workflows[workflow.ID] = workflow
	executor.mu.Unlock()

	// Retrieve workflow
	retrieved, err := executor.GetWorkflow(workflow.ID)
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, retrieved.ID)

	// Try non-existent workflow
	_, err = executor.GetWorkflow("nonexistent")
	assert.Error(t, err)
}

func TestWorkflowExecutorListWorkflows(t *testing.T) {
	coordinator := NewCoordinator(nil)
	executor := coordinator.workflowExecutor

	workflow1 := NewWorkflow("Workflow 1", "Test 1")
	workflow2 := NewWorkflow("Workflow 2", "Test 2")

	// Register workflows
	executor.mu.Lock()
	executor.workflows[workflow1.ID] = workflow1
	executor.workflows[workflow2.ID] = workflow2
	executor.mu.Unlock()

	// List workflows
	workflows := executor.ListWorkflows()
	require.Len(t, workflows, 2, "Expected 2 workflows")

	ids := []string{workflows[0].ID, workflows[1].ID}
	assert.Contains(t, ids, workflow1.ID)
	assert.Contains(t, ids, workflow2.ID)
}

func TestWorkflowExecutorCapabilityMatching(t *testing.T) {
	coordinator := NewCoordinator(nil)

	// Register agent with specific capabilities
	agent := &mockWorkflowAgent{
		id:           "specialized-agent",
		agentType:    AgentTypeCoding,
		capabilities: []Capability{CapabilityCodeGeneration, CapabilityRefactoring},
	}

	coordinator.RegisterAgent(agent)

	// Create workflow with capability requirement
	workflow := NewWorkflow("Capability Workflow", "Test capability matching")

	step := &WorkflowStep{
		ID:           "specialized-step",
		Name:         "Specialized Step",
		AgentType:    AgentTypeCoding,
		RequiredCaps: []Capability{CapabilityCodeGeneration},
	}

	workflow.AddStep(step)

	// Execute workflow
	ctx := context.Background()
	err := coordinator.ExecuteWorkflow(ctx, workflow)
	require.NoError(t, err)

	// Verify step completed
	result, ok := workflow.GetStepResult("specialized-step")
	assert.True(t, ok)
	assert.True(t, result.Success)
	assert.Equal(t, "specialized-agent", result.AgentID)
}

// TestWorkflowComplexDependencies tests workflows with complex dependency chains
func TestWorkflowComplexDependencies(t *testing.T) {
	workflow := NewWorkflow("Complex Dependencies", "Test complex dependency graph")

	// Create a diamond dependency pattern:
	// step1 -> step2 -> step4
	//       -> step3 -> step4
	step1 := &WorkflowStep{ID: "step1", Name: "Step 1", AgentType: AgentTypePlanning}
	step2 := &WorkflowStep{ID: "step2", Name: "Step 2", AgentType: AgentTypeCoding, DependsOn: []string{"step1"}}
	step3 := &WorkflowStep{ID: "step3", Name: "Step 3", AgentType: AgentTypeTesting, DependsOn: []string{"step1"}}
	step4 := &WorkflowStep{ID: "step4", Name: "Step 4", AgentType: AgentTypeReview, DependsOn: []string{"step2", "step3"}}

	workflow.AddStep(step1)
	workflow.AddStep(step2)
	workflow.AddStep(step3)
	workflow.AddStep(step4)

	// Only step1 should be ready initially
	ready := workflow.GetReadySteps()
	assert.Len(t, ready, 1)
	assert.Equal(t, "step1", ready[0].ID)

	// Complete step1
	workflow.SetStepResult("step1", &task.Result{TaskID: "step1", Success: true})

	// Step2 and step3 should be ready (parallel)
	ready = workflow.GetReadySteps()
	assert.Len(t, ready, 2)

	// Complete step2 and step3
	workflow.SetStepResult("step2", &task.Result{TaskID: "step2", Success: true})
	workflow.SetStepResult("step3", &task.Result{TaskID: "step3", Success: true})

	// Step4 should now be ready
	ready = workflow.GetReadySteps()
	assert.Len(t, ready, 1)
	assert.Equal(t, "step4", ready[0].ID)
}

// TestWorkflowRequiredStepFailure tests that workflow fails when required step fails
func TestWorkflowRequiredStepFailure(t *testing.T) {
	workflow := NewWorkflow("Required Failure", "Test required step failure")

	step1 := &WorkflowStep{ID: "step1", Name: "Step 1", AgentType: AgentTypePlanning}
	step2 := &WorkflowStep{ID: "step2", Name: "Step 2", AgentType: AgentTypeCoding, DependsOn: []string{"step1"}}

	workflow.AddStep(step1)
	workflow.AddStep(step2)

	// Mark step1 as failed
	workflow.SetStepResult("step1", &task.Result{TaskID: "step1", Success: false})

	// Step2 should not be ready
	assert.False(t, workflow.IsStepReady(step2))

	// No ready steps
	ready := workflow.GetReadySteps()
	assert.Len(t, ready, 0)
}

// TestWorkflowMultipleOptionalSteps tests multiple optional steps
func TestWorkflowMultipleOptionalSteps(t *testing.T) {
	workflow := NewWorkflow("Multiple Optional", "Test multiple optional steps")

	step1 := &WorkflowStep{ID: "step1", Name: "Step 1", AgentType: AgentTypePlanning, Optional: true}
	step2 := &WorkflowStep{ID: "step2", Name: "Step 2", AgentType: AgentTypeCoding, Optional: true}
	step3 := &WorkflowStep{ID: "step3", Name: "Step 3", AgentType: AgentTypeTesting, DependsOn: []string{"step1", "step2"}}

	workflow.AddStep(step1)
	workflow.AddStep(step2)
	workflow.AddStep(step3)

	// Fail both optional steps
	workflow.SetStepResult("step1", &task.Result{TaskID: "step1", Success: false})
	workflow.SetStepResult("step2", &task.Result{TaskID: "step2", Success: false})

	// Step3 should still be ready despite failed optional dependencies
	assert.True(t, workflow.IsStepReady(step3))
}

// TestWorkflowLargeScale tests workflow with many steps
func TestWorkflowLargeScale(t *testing.T) {
	workflow := NewWorkflow("Large Scale", "Test large workflow")

	// Add 50 independent steps
	for i := 0; i < 50; i++ {
		step := &WorkflowStep{
			ID:        fmt.Sprintf("step-%d", i),
			Name:      fmt.Sprintf("Step %d", i),
			AgentType: AgentTypePlanning,
		}
		workflow.AddStep(step)
	}

	// All 50 steps should be ready (no dependencies)
	ready := workflow.GetReadySteps()
	assert.Len(t, ready, 50)

	// Complete half of them
	for i := 0; i < 25; i++ {
		workflow.SetStepResult(fmt.Sprintf("step-%d", i), &task.Result{
			TaskID:  fmt.Sprintf("step-%d", i),
			Success: true,
		})
	}

	// Remaining 25 should be ready
	ready = workflow.GetReadySteps()
	assert.Len(t, ready, 25)
}

// TestWorkflowLinearChain tests a long linear dependency chain
func TestWorkflowLinearChain(t *testing.T) {
	workflow := NewWorkflow("Linear Chain", "Test linear dependencies")

	// Create chain: step1 -> step2 -> step3 -> ... -> step10
	for i := 0; i < 10; i++ {
		step := &WorkflowStep{
			ID:        fmt.Sprintf("step-%d", i),
			Name:      fmt.Sprintf("Step %d", i),
			AgentType: AgentTypePlanning,
		}
		if i > 0 {
			step.DependsOn = []string{fmt.Sprintf("step-%d", i-1)}
		}
		workflow.AddStep(step)
	}

	// Only first step should be ready
	ready := workflow.GetReadySteps()
	assert.Len(t, ready, 1)
	assert.Equal(t, "step-0", ready[0].ID)

	// Complete each step in order
	for i := 0; i < 10; i++ {
		stepID := fmt.Sprintf("step-%d", i)

		// Mark current step complete
		workflow.SetStepResult(stepID, &task.Result{TaskID: stepID, Success: true})

		// Check next step is ready (if not at end)
		if i < 9 {
			ready = workflow.GetReadySteps()
			assert.Len(t, ready, 1)
			assert.Equal(t, fmt.Sprintf("step-%d", i+1), ready[0].ID)
		}
	}

	// All done - no ready steps
	ready = workflow.GetReadySteps()
	assert.Len(t, ready, 0)
}

// TestWorkflowEmptyWorkflow tests empty workflow edge case
func TestWorkflowEmptyWorkflow(t *testing.T) {
	workflow := NewWorkflow("Empty", "Empty workflow")

	// No steps
	assert.Len(t, workflow.Steps, 0)

	// No ready steps
	ready := workflow.GetReadySteps()
	assert.Len(t, ready, 0)

	// Can still transition states
	workflow.Start()
	assert.Equal(t, WorkflowStatusRunning, workflow.Status)

	workflow.Complete()
	assert.Equal(t, WorkflowStatusCompleted, workflow.Status)
}

// TestWorkflowDuplicateStepID tests handling of duplicate step IDs
func TestWorkflowDuplicateStepID(t *testing.T) {
	workflow := NewWorkflow("Duplicate IDs", "Test duplicate step IDs")

	step1 := &WorkflowStep{ID: "duplicate", Name: "Step 1", AgentType: AgentTypePlanning}
	step2 := &WorkflowStep{ID: "duplicate", Name: "Step 2", AgentType: AgentTypeCoding}

	workflow.AddStep(step1)
	workflow.AddStep(step2)

	// Both steps added (no deduplication at add time)
	assert.Len(t, workflow.Steps, 2)

	// Complete one
	workflow.SetStepResult("duplicate", &task.Result{TaskID: "duplicate", Success: true})

	// Both steps will show as completed (same ID)
	ready := workflow.GetReadySteps()
	assert.Len(t, ready, 0) // Both filtered out as "executed"
}

// TestWorkflowNilDependencies tests steps with nil dependency slices
func TestWorkflowNilDependencies(t *testing.T) {
	workflow := NewWorkflow("Nil Dependencies", "Test nil dependency handling")

	step := &WorkflowStep{
		ID:        "step1",
		Name:      "Step 1",
		AgentType: AgentTypePlanning,
		DependsOn: nil, // Explicitly nil
	}

	workflow.AddStep(step)

	// Should be ready (no dependencies)
	assert.True(t, workflow.IsStepReady(step))

	ready := workflow.GetReadySteps()
	assert.Len(t, ready, 1)
}

// TestWorkflowStepInputMerging tests input merging from dependencies
func TestWorkflowStepInputMerging(t *testing.T) {
	workflow := NewWorkflow("Input Merging", "Test input merging")

	step1 := &WorkflowStep{
		ID:        "step1",
		Name:      "Step 1",
		AgentType: AgentTypePlanning,
		Input:     map[string]interface{}{"key1": "value1"},
	}

	step2 := &WorkflowStep{
		ID:        "step2",
		Name:      "Step 2",
		AgentType: AgentTypeCoding,
		DependsOn: []string{"step1"},
		Input:     map[string]interface{}{"key2": "value2"},
	}

	workflow.AddStep(step1)
	workflow.AddStep(step2)

	// Step1 result with output
	result1 := &task.Result{
		TaskID:  "step1",
		Success: true,
		Output:  map[string]interface{}{"output_key": "output_value"},
	}
	workflow.SetStepResult("step1", result1)

	// Verify step2 is ready
	assert.True(t, workflow.IsStepReady(step2))
}

// TestWorkflowConcurrentStateModification tests thread-safety
func TestWorkflowConcurrentStateModification(t *testing.T) {
	workflow := NewWorkflow("Concurrent", "Test concurrent modifications")

	// Add steps concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			step := &WorkflowStep{
				ID:        fmt.Sprintf("step-%d", idx),
				Name:      fmt.Sprintf("Step %d", idx),
				AgentType: AgentTypePlanning,
			}
			workflow.AddStep(step)
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have all 10 steps
	assert.Len(t, workflow.Steps, 10)

	// Set results concurrently
	for i := 0; i < 10; i++ {
		go func(idx int) {
			workflow.SetStepResult(fmt.Sprintf("step-%d", idx), &task.Result{
				TaskID:  fmt.Sprintf("step-%d", idx),
				Success: true,
			})
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	// All results should be recorded
	assert.Len(t, workflow.Results, 10)
}
