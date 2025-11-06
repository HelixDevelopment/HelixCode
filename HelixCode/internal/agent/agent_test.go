package agent

import (
	"context"
	"testing"
	"time"

	"dev.helix.code/internal/agent/task"
)

func TestNewBaseAgent(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent-1",
		Type: AgentTypePlanning,
		Name: "Test Planning Agent",
		Capabilities: []Capability{
			CapabilityPlanning,
			CapabilityCodeAnalysis,
		},
	}

	agent := NewBaseAgent(config)

	if agent.ID() != config.ID {
		t.Errorf("Expected ID %s, got %s", config.ID, agent.ID())
	}

	if agent.Type() != config.Type {
		t.Errorf("Expected Type %s, got %s", config.Type, agent.Type())
	}

	if agent.Name() != config.Name {
		t.Errorf("Expected Name %s, got %s", config.Name, agent.Name())
	}

	if agent.Status() != StatusIdle {
		t.Errorf("Expected initial status %s, got %s", StatusIdle, agent.Status())
	}

	caps := agent.Capabilities()
	if len(caps) != len(config.Capabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(config.Capabilities), len(caps))
	}
}

func TestBaseAgentStatusManagement(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent-2",
		Type: AgentTypeCoding,
		Name: "Test Coding Agent",
	}

	agent := NewBaseAgent(config)

	// Test status transitions
	agent.SetStatus(StatusBusy)
	if agent.Status() != StatusBusy {
		t.Errorf("Expected status %s, got %s", StatusBusy, agent.Status())
	}

	agent.SetStatus(StatusWaiting)
	if agent.Status() != StatusWaiting {
		t.Errorf("Expected status %s, got %s", StatusWaiting, agent.Status())
	}

	agent.SetStatus(StatusIdle)
	if agent.Status() != StatusIdle {
		t.Errorf("Expected status %s, got %s", StatusIdle, agent.Status())
	}
}

func TestBaseAgentTaskCounters(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent-3",
		Type: AgentTypeTesting,
		Name: "Test Testing Agent",
	}

	agent := NewBaseAgent(config)

	// Initially should be 0
	health := agent.Health()
	if health.TaskCount != 0 {
		t.Errorf("Expected initial task count 0, got %d", health.TaskCount)
	}
	if health.ErrorCount != 0 {
		t.Errorf("Expected initial error count 0, got %d", health.ErrorCount)
	}

	// Increment task count
	agent.IncrementTaskCount()
	agent.IncrementTaskCount()
	health = agent.Health()
	if health.TaskCount != 2 {
		t.Errorf("Expected task count 2, got %d", health.TaskCount)
	}

	// Increment error count
	agent.IncrementErrorCount()
	health = agent.Health()
	if health.ErrorCount != 1 {
		t.Errorf("Expected error count 1, got %d", health.ErrorCount)
	}

	// Check error rate calculation
	expectedRate := 1.0 / 2.0
	if health.ErrorRate != expectedRate {
		t.Errorf("Expected error rate %f, got %f", expectedRate, health.ErrorRate)
	}
}

func TestBaseAgentCanHandle(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent-4",
		Type: AgentTypePlanning,
		Name: "Test Agent",
		Capabilities: []Capability{
			CapabilityPlanning,
			CapabilityCodeAnalysis,
		},
	}

	agent := NewBaseAgent(config)

	// Test with nil task
	if agent.CanHandle(nil) {
		t.Error("Agent should not handle nil task")
	}

	// Test with task requiring matching capabilities
	t1 := task.NewTask(task.TaskTypePlanning, "Test Task", "Description", task.PriorityNormal)
	t1.RequiredCapabilities = []string{string(CapabilityPlanning)}

	if !agent.CanHandle(t1) {
		t.Error("Agent should handle task with matching capability")
	}

	// Test with task requiring non-existent capability
	t2 := task.NewTask(task.TaskTypeCodeGeneration, "Test Task 2", "Description", task.PriorityNormal)
	t2.RequiredCapabilities = []string{string(CapabilityCodeGeneration)}

	if agent.CanHandle(t2) {
		t.Error("Agent should not handle task without required capability")
	}

	// Test with task requiring multiple capabilities (one missing)
	t3 := task.NewTask(task.TaskTypeAnalysis, "Test Task 3", "Description", task.PriorityNormal)
	t3.RequiredCapabilities = []string{
		string(CapabilityPlanning),
		string(CapabilityCodeGeneration), // This one is missing
	}

	if agent.CanHandle(t3) {
		t.Error("Agent should not handle task with missing required capability")
	}
}

func TestBaseAgentHealth(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent-5",
		Type: AgentTypeDebugging,
		Name: "Test Debugging Agent",
	}

	agent := NewBaseAgent(config)

	// Initial health
	health := agent.Health()
	if health.AgentID != agent.ID() {
		t.Errorf("Expected AgentID %s, got %s", agent.ID(), health.AgentID)
	}
	if !health.Healthy {
		t.Error("New agent should be healthy")
	}
	if health.Uptime <= 0 {
		t.Error("Uptime should be positive")
	}

	// Sleep briefly to test uptime
	time.Sleep(100 * time.Millisecond)
	health = agent.Health()
	if health.Uptime < 100*time.Millisecond {
		t.Error("Uptime should increase over time")
	}

	// Test unhealthy due to error status
	agent.SetStatus(StatusError)
	health = agent.Health()
	if health.Healthy {
		t.Error("Agent with error status should be unhealthy")
	}

	// Test unhealthy due to high error rate
	agent.SetStatus(StatusIdle)
	for i := 0; i < 5; i++ {
		agent.IncrementTaskCount()
	}
	for i := 0; i < 2; i++ {
		agent.IncrementErrorCount()
	}
	health = agent.Health()
	if health.Healthy {
		t.Error("Agent with high error rate (>20%) should be unhealthy")
	}
}

func TestAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()

	// Test empty registry
	if registry.Count() != 0 {
		t.Errorf("Expected empty registry, got count %d", registry.Count())
	}

	// Test registering agents
	agent1 := &MockAgent{
		BaseAgent: NewBaseAgent(&AgentConfig{
			ID:   "agent-1",
			Type: AgentTypePlanning,
			Name: "Agent 1",
		}),
	}
	agent2 := &MockAgent{
		BaseAgent: NewBaseAgent(&AgentConfig{
			ID:   "agent-2",
			Type: AgentTypeCoding,
			Name: "Agent 2",
		}),
	}

	err := registry.Register(agent1)
	if err != nil {
		t.Errorf("Failed to register agent1: %v", err)
	}

	err = registry.Register(agent2)
	if err != nil {
		t.Errorf("Failed to register agent2: %v", err)
	}

	if registry.Count() != 2 {
		t.Errorf("Expected count 2, got %d", registry.Count())
	}

	// Test registering nil agent
	err = registry.Register(nil)
	if err != ErrNilAgent {
		t.Error("Expected ErrNilAgent when registering nil agent")
	}

	// Test retrieving agent by ID
	retrieved, err := registry.Get("agent-1")
	if err != nil {
		t.Errorf("Failed to get agent: %v", err)
	}
	if retrieved.ID() != "agent-1" {
		t.Errorf("Expected agent ID agent-1, got %s", retrieved.ID())
	}

	// Test retrieving non-existent agent
	_, err = registry.Get("non-existent")
	if err != ErrAgentNotFound {
		t.Error("Expected ErrAgentNotFound for non-existent agent")
	}

	// Test getting agents by type
	planningAgents := registry.GetByType(AgentTypePlanning)
	if len(planningAgents) != 1 {
		t.Errorf("Expected 1 planning agent, got %d", len(planningAgents))
	}
	if planningAgents[0].ID() != "agent-1" {
		t.Error("Wrong agent returned for planning type")
	}

	// Test unregistering agent
	registry.Unregister("agent-1")
	if registry.Count() != 1 {
		t.Errorf("Expected count 1 after unregister, got %d", registry.Count())
	}

	_, err = registry.Get("agent-1")
	if err != ErrAgentNotFound {
		t.Error("Agent should not be found after unregister")
	}
}

func TestAgentRegistryByCapability(t *testing.T) {
	registry := NewAgentRegistry()

	agent1 := &MockAgent{
		BaseAgent: NewBaseAgent(&AgentConfig{
			ID:   "agent-1",
			Type: AgentTypePlanning,
			Name: "Agent 1",
			Capabilities: []Capability{
				CapabilityPlanning,
				CapabilityCodeAnalysis,
			},
		}),
	}

	agent2 := &MockAgent{
		BaseAgent: NewBaseAgent(&AgentConfig{
			ID:   "agent-2",
			Type: AgentTypeCoding,
			Name: "Agent 2",
			Capabilities: []Capability{
				CapabilityCodeGeneration,
				CapabilityCodeAnalysis,
			},
		}),
	}

	registry.Register(agent1)
	registry.Register(agent2)

	// Test finding agents by capability
	analysisAgents := registry.GetByCapability(CapabilityCodeAnalysis)
	if len(analysisAgents) != 2 {
		t.Errorf("Expected 2 agents with code analysis capability, got %d", len(analysisAgents))
	}

	planningAgents := registry.GetByCapability(CapabilityPlanning)
	if len(planningAgents) != 1 {
		t.Errorf("Expected 1 agent with planning capability, got %d", len(planningAgents))
	}

	generationAgents := registry.GetByCapability(CapabilityCodeGeneration)
	if len(generationAgents) != 1 {
		t.Errorf("Expected 1 agent with code generation capability, got %d", len(generationAgents))
	}
}

func TestGenerateAgentID(t *testing.T) {
	id1 := GenerateAgentID(AgentTypePlanning)
	id2 := GenerateAgentID(AgentTypePlanning)

	// Check format
	if len(id1) == 0 {
		t.Error("Generated ID should not be empty")
	}

	// Check uniqueness
	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	// Check prefix
	if id1[:8] != "planning" {
		t.Error("ID should start with agent type")
	}
}

// MockAgent implements the Agent interface for testing
type MockAgent struct {
	*BaseAgent
	executeFunc func(ctx context.Context, task *task.Task) (*task.Result, error)
}

func (m *MockAgent) Initialize(ctx context.Context, config *AgentConfig) error {
	return nil
}

func (m *MockAgent) Execute(ctx context.Context, t *task.Task) (*task.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, t)
	}
	result := task.NewResult(t.ID, m.ID())
	result.SetSuccess(map[string]interface{}{"status": "completed"}, 1.0)
	return result, nil
}

func (m *MockAgent) Collaborate(ctx context.Context, agents []Agent, t *task.Task) (*CollaborationResult, error) {
	return &CollaborationResult{
		Success: true,
		Results: map[string]*task.Result{},
	}, nil
}

func (m *MockAgent) Shutdown(ctx context.Context) error {
	m.SetStatus(StatusShutdown)
	return nil
}

func TestMockAgent(t *testing.T) {
	config := &AgentConfig{
		ID:   "mock-agent-1",
		Type: AgentTypeCoding,
		Name: "Mock Agent",
	}

	mockAgent := &MockAgent{
		BaseAgent: NewBaseAgent(config),
	}

	// Test basic interface implementation
	if mockAgent.ID() != config.ID {
		t.Errorf("Expected ID %s, got %s", config.ID, mockAgent.ID())
	}

	// Test execute
	testTask := task.NewTask(task.TaskTypeCodeGeneration, "Test", "Test task", task.PriorityNormal)
	result, err := mockAgent.Execute(context.Background(), testTask)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if !result.Success {
		t.Error("Expected successful result")
	}

	// Test shutdown
	err = mockAgent.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	if mockAgent.Status() != StatusShutdown {
		t.Error("Expected shutdown status")
	}
}
