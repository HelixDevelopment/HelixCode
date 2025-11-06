package agent

import (
	"context"
	"sync"
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

func TestBaseAgentCapabilityMatching(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent",
		Type: AgentTypePlanning,
		Name: "Test Agent",
		Capabilities: []Capability{
			CapabilityPlanning,
			CapabilityCodeAnalysis,
		},
	}

	agent := NewBaseAgent(config)

	// Test task requiring only one capability that matches
	t1 := task.NewTask(task.TaskTypePlanning, "Task 1", "Test", task.PriorityNormal)
	t1.RequiredCapabilities = []string{string(CapabilityPlanning)}
	if !agent.CanHandle(t1) {
		t.Error("Agent should handle task with single matching capability")
	}

	// Test task requiring multiple capabilities, all matching
	t2 := task.NewTask(task.TaskTypePlanning, "Task 2", "Test", task.PriorityNormal)
	t2.RequiredCapabilities = []string{
		string(CapabilityPlanning),
		string(CapabilityCodeAnalysis),
	}
	if !agent.CanHandle(t2) {
		t.Error("Agent should handle task with all capabilities matching")
	}

	// Test task with no required capabilities
	t3 := task.NewTask(task.TaskTypePlanning, "Task 3", "Test", task.PriorityNormal)
	t3.RequiredCapabilities = []string{}
	if !agent.CanHandle(t3) {
		t.Error("Agent should handle task with no required capabilities")
	}
}

func TestBaseAgentHealthWithOperations(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent",
		Type: AgentTypeCoding,
		Name: "Test Agent",
	}

	agent := NewBaseAgent(config)

	// Initial health should be healthy
	health := agent.Health()
	if !health.Healthy {
		t.Error("Agent should be healthy initially")
	}

	// Perform some operations
	agent.IncrementTaskCount()
	agent.IncrementTaskCount()
	agent.IncrementTaskCount()

	health = agent.Health()
	if health.TaskCount != 3 {
		t.Errorf("Expected 3 tasks, got %d", health.TaskCount)
	}
	if !health.Healthy {
		t.Error("Agent should still be healthy with low error rate")
	}

	// Add errors
	agent.IncrementErrorCount()
	health = agent.Health()
	errorRate := 1.0 / 3.0
	if health.ErrorRate != errorRate {
		t.Errorf("Expected error rate %f, got %f", errorRate, health.ErrorRate)
	}
}

func TestBaseAgentErrorRateCalculation(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent",
		Type: AgentTypeTesting,
		Name: "Test Agent",
	}

	agent := NewBaseAgent(config)

	// Test with zero tasks
	health := agent.Health()
	if health.ErrorRate != 0.0 {
		t.Errorf("Expected error rate 0.0 with no tasks, got %f", health.ErrorRate)
	}

	// Test with tasks but no errors
	for i := 0; i < 10; i++ {
		agent.IncrementTaskCount()
	}
	health = agent.Health()
	if health.ErrorRate != 0.0 {
		t.Errorf("Expected error rate 0.0 with no errors, got %f", health.ErrorRate)
	}

	// Test with 50% error rate
	for i := 0; i < 5; i++ {
		agent.IncrementErrorCount()
	}
	health = agent.Health()
	if health.ErrorRate != 0.5 {
		t.Errorf("Expected error rate 0.5, got %f", health.ErrorRate)
	}
}

func TestBaseAgentStatusSequence(t *testing.T) {
	config := &AgentConfig{
		ID:   "test-agent",
		Type: AgentTypeDebugging,
		Name: "Test Agent",
	}

	agent := NewBaseAgent(config)

	// Test status sequence: idle -> busy -> waiting -> busy -> idle
	statuses := []AgentStatus{StatusBusy, StatusWaiting, StatusBusy, StatusIdle}
	for _, status := range statuses {
		agent.SetStatus(status)
		if agent.Status() != status {
			t.Errorf("Expected status %s, got %s", status, agent.Status())
		}
	}

	// Test error status
	agent.SetStatus(StatusError)
	if agent.Status() != StatusError {
		t.Error("Expected error status")
	}

	health := agent.Health()
	if health.Healthy {
		t.Error("Agent with error status should be unhealthy")
	}
}

func TestBaseAgentEmptyConfig(t *testing.T) {
	// Test with minimal config
	config := &AgentConfig{
		ID:   "minimal-agent",
		Type: AgentTypePlanning,
		Name: "Minimal Agent",
	}

	agent := NewBaseAgent(config)

	if agent.ID() != "minimal-agent" {
		t.Errorf("Expected ID minimal-agent, got %s", agent.ID())
	}

	if agent.Status() != StatusIdle {
		t.Errorf("Expected initial status idle, got %s", agent.Status())
	}

	// Should have no capabilities
	caps := agent.Capabilities()
	if len(caps) != 0 {
		t.Errorf("Expected 0 capabilities, got %d", len(caps))
	}

	// Can handle task with no requirements
	t1 := task.NewTask(task.TaskTypePlanning, "Task", "Test", task.PriorityNormal)
	if !agent.CanHandle(t1) {
		t.Error("Agent should handle task with no capability requirements")
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

func TestAgentRegistryListMultiple(t *testing.T) {
	registry := NewAgentRegistry()

	// Initially empty
	agents := registry.List()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents initially, got %d", len(agents))
	}

	// Register some agents
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

	registry.Register(agent1)
	registry.Register(agent2)

	agents = registry.List()
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestAgentRegistryUnregisterNonExistent(t *testing.T) {
	registry := NewAgentRegistry()

	// Unregistering non-existent agent should not panic
	registry.Unregister("non-existent-id")

	if registry.Count() != 0 {
		t.Errorf("Expected count 0, got %d", registry.Count())
	}
}

func TestAgentRegistryMultipleUnregister(t *testing.T) {
	registry := NewAgentRegistry()

	agent := &MockAgent{
		BaseAgent: NewBaseAgent(&AgentConfig{
			ID:   "agent-1",
			Type: AgentTypePlanning,
			Name: "Agent 1",
		}),
	}

	registry.Register(agent)

	// Unregister multiple times
	registry.Unregister("agent-1")
	registry.Unregister("agent-1")
	registry.Unregister("agent-1")

	if registry.Count() != 0 {
		t.Errorf("Expected count 0, got %d", registry.Count())
	}
}

func TestAgentRegistryGetByTypeEmpty(t *testing.T) {
	registry := NewAgentRegistry()

	// Query for agents of a type when registry is empty
	agents := registry.GetByType(AgentTypePlanning)
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}

	// Register agent of different type
	agent := &MockAgent{
		BaseAgent: NewBaseAgent(&AgentConfig{
			ID:   "agent-1",
			Type: AgentTypeCoding,
			Name: "Agent 1",
		}),
	}
	registry.Register(agent)

	// Query for different type should return empty
	agents = registry.GetByType(AgentTypePlanning)
	if len(agents) != 0 {
		t.Errorf("Expected 0 planning agents, got %d", len(agents))
	}
}

func TestAgentRegistryGetByCapabilityEmpty(t *testing.T) {
	registry := NewAgentRegistry()

	// Query for capability when registry is empty
	agents := registry.GetByCapability(CapabilityPlanning)
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}

	// Register agent without the capability
	agent := &MockAgent{
		BaseAgent: NewBaseAgent(&AgentConfig{
			ID:   "agent-1",
			Type: AgentTypeCoding,
			Name: "Agent 1",
			Capabilities: []Capability{
				CapabilityCodeGeneration,
			},
		}),
	}
	registry.Register(agent)

	// Query for different capability should return empty
	agents = registry.GetByCapability(CapabilityPlanning)
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents with planning capability, got %d", len(agents))
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

func TestBaseAgentConcurrentTaskCounting(t *testing.T) {
	config := &AgentConfig{
		ID:   "concurrent-agent",
		Type: AgentTypeCoding,
		Name: "Concurrent Agent",
	}

	agent := NewBaseAgent(config)

	// Concurrently increment task count
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			agent.IncrementTaskCount()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	health := agent.Health()
	if health.TaskCount != 100 {
		t.Errorf("Expected task count 100, got %d", health.TaskCount)
	}
}

func TestBaseAgentConcurrentErrorCounting(t *testing.T) {
	config := &AgentConfig{
		ID:   "concurrent-agent",
		Type: AgentTypeTesting,
		Name: "Concurrent Agent",
	}

	agent := NewBaseAgent(config)

	// Setup some tasks first
	for i := 0; i < 100; i++ {
		agent.IncrementTaskCount()
	}

	// Concurrently increment error count
	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func() {
			agent.IncrementErrorCount()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	health := agent.Health()
	if health.ErrorCount != 50 {
		t.Errorf("Expected error count 50, got %d", health.ErrorCount)
	}
	if health.ErrorRate != 0.5 {
		t.Errorf("Expected error rate 0.5, got %f", health.ErrorRate)
	}
}

func TestBaseAgentConcurrentStatusChanges(t *testing.T) {
	config := &AgentConfig{
		ID:   "concurrent-agent",
		Type: AgentTypeDebugging,
		Name: "Concurrent Agent",
	}

	agent := NewBaseAgent(config)

	// Concurrently change status
	done := make(chan bool, 50)
	statuses := []AgentStatus{StatusBusy, StatusWaiting, StatusIdle, StatusBusy}

	for i := 0; i < 50; i++ {
		go func(idx int) {
			status := statuses[idx%len(statuses)]
			agent.SetStatus(status)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Agent should have some valid status
	finalStatus := agent.Status()
	validStatuses := map[AgentStatus]bool{
		StatusIdle:    true,
		StatusBusy:    true,
		StatusWaiting: true,
	}

	if !validStatuses[finalStatus] {
		t.Errorf("Expected valid status, got %s", finalStatus)
	}
}

func TestBaseAgentConcurrentCapabilityChecks(t *testing.T) {
	config := &AgentConfig{
		ID:   "concurrent-agent",
		Type: AgentTypePlanning,
		Name: "Concurrent Agent",
		Capabilities: []Capability{
			CapabilityPlanning,
			CapabilityCodeAnalysis,
		},
	}

	agent := NewBaseAgent(config)

	// Create tasks
	t1 := task.NewTask(task.TaskTypePlanning, "Task 1", "Test", task.PriorityNormal)
	t1.RequiredCapabilities = []string{string(CapabilityPlanning)}

	t2 := task.NewTask(task.TaskTypePlanning, "Task 2", "Test", task.PriorityNormal)
	t2.RequiredCapabilities = []string{string(CapabilityCodeAnalysis)}

	// Concurrently check if agent can handle tasks
	done := make(chan bool, 50)
	for i := 0; i < 25; i++ {
		go func() {
			if !agent.CanHandle(t1) {
				t.Error("Agent should handle task with planning capability")
			}
			done <- true
		}()
	}

	for i := 0; i < 25; i++ {
		go func() {
			if !agent.CanHandle(t2) {
				t.Error("Agent should handle task with code analysis capability")
			}
			done <- true
		}()
	}

	// Wait for all checks
	for i := 0; i < 50; i++ {
		<-done
	}
}

func TestBaseAgentConcurrentHealthChecks(t *testing.T) {
	config := &AgentConfig{
		ID:   "concurrent-agent",
		Type: AgentTypeReview,
		Name: "Concurrent Agent",
	}

	agent := NewBaseAgent(config)

	// Concurrently check health while modifying counters
	var wg sync.WaitGroup

	// Start health checkers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			health := agent.Health()
			if health.AgentID != agent.ID() {
				t.Errorf("Expected agent ID %s, got %s", agent.ID(), health.AgentID)
			}
		}()
	}

	// Start counter incrementers
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			agent.IncrementTaskCount()
		}()
	}

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			agent.IncrementErrorCount()
		}()
	}

	// Wait for all operations
	wg.Wait()

	// Final health check
	health := agent.Health()
	if health.TaskCount != 25 {
		t.Errorf("Expected task count 25, got %d", health.TaskCount)
	}
	if health.ErrorCount != 25 {
		t.Errorf("Expected error count 25, got %d", health.ErrorCount)
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
