package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/agent/task"
)

// Coordinator manages and orchestrates multiple agents
type Coordinator struct {
	registry         *AgentRegistry
	tasks            map[string]*task.Task
	taskQueue        []*task.Task
	results          map[string]*task.Result
	mu               sync.RWMutex
	config           *CoordinatorConfig
	workflowExecutor *WorkflowExecutor
	circuitBreakers  *CircuitBreakerManager
	retryPolicy      *RetryPolicy
}

// CoordinatorConfig holds coordinator configuration
type CoordinatorConfig struct {
	MaxConcurrentTasks int
	TaskTimeout        time.Duration
	EnableCollaboration bool
	ConflictResolution ResolutionMethod
	EnableResilience   bool          // Enable circuit breakers and retries
	FailureThreshold   int           // Circuit breaker failure threshold
	SuccessThreshold   int           // Circuit breaker success threshold
	CircuitBreakerTimeout time.Duration // Circuit breaker timeout
}

// NewCoordinator creates a new agent coordinator
func NewCoordinator(config *CoordinatorConfig) *Coordinator {
	if config == nil {
		config = &CoordinatorConfig{
			MaxConcurrentTasks: 10,
			TaskTimeout:        30 * time.Minute,
			EnableCollaboration: true,
			ConflictResolution: ResolutionMethodVoting,
			EnableResilience:   true,
			FailureThreshold:   5,
			SuccessThreshold:   2,
			CircuitBreakerTimeout: 60 * time.Second,
		}
	}

	coordinator := &Coordinator{
		registry:  NewAgentRegistry(),
		tasks:     make(map[string]*task.Task),
		taskQueue: make([]*task.Task, 0),
		results:   make(map[string]*task.Result),
		config:    config,
	}

	// Initialize workflow executor
	coordinator.workflowExecutor = NewWorkflowExecutor(coordinator)

	// Initialize circuit breaker manager if resilience is enabled
	if config.EnableResilience {
		coordinator.circuitBreakers = NewCircuitBreakerManager(
			config.FailureThreshold,
			config.SuccessThreshold,
			config.CircuitBreakerTimeout,
		)
		coordinator.retryPolicy = DefaultRetryPolicy()
	}

	return coordinator
}

// RegisterAgent registers an agent with the coordinator
func (c *Coordinator) RegisterAgent(agent Agent) error {
	return c.registry.Register(agent)
}

// SubmitTask submits a new task for execution
func (c *Coordinator) SubmitTask(ctx context.Context, t *task.Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if t == nil {
		return fmt.Errorf("task cannot be nil")
	}

	c.tasks[t.ID] = t
	c.taskQueue = append(c.taskQueue, t)

	return nil
}

// ExecuteTask assigns and executes a task
func (c *Coordinator) ExecuteTask(ctx context.Context, taskID string) (*task.Result, error) {
	c.mu.RLock()
	t, exists := c.tasks[taskID]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// Find suitable agent
	agent, err := c.findSuitableAgent(t)
	if err != nil {
		return nil, fmt.Errorf("no suitable agent found: %w", err)
	}

	// Execute task with or without resilience
	var result *task.Result
	t.Start(agent.ID())

	if c.config.EnableResilience && c.circuitBreakers != nil {
		// Execute with circuit breaker and retry
		circuitBreaker := c.circuitBreakers.GetOrCreate(agent.ID())
		resilientExecutor := NewResilientExecutor(agent, circuitBreaker, c.retryPolicy)
		result, err = resilientExecutor.Execute(ctx, t)
	} else {
		// Execute directly
		result, err = agent.Execute(ctx, t)
	}

	if err != nil {
		t.Fail(err.Error())
		return nil, err
	}

	t.Complete(result.Output)

	c.mu.Lock()
	c.results[taskID] = result
	c.mu.Unlock()

	return result, nil
}

// findSuitableAgent finds an agent that can handle the task
func (c *Coordinator) findSuitableAgent(t *task.Task) (Agent, error) {
	agents := c.registry.List()

	for _, agent := range agents {
		if agent.CanHandle(t) && agent.Status() == StatusIdle {
			return agent, nil
		}
	}

	return nil, fmt.Errorf("no available agent found")
}

// GetTaskStatus returns the status of a task
func (c *Coordinator) GetTaskStatus(taskID string) (*task.Task, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	t, exists := c.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return t, nil
}

// GetResult returns the result of a completed task
func (c *Coordinator) GetResult(taskID string) (*task.Result, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, exists := c.results[taskID]
	if !exists {
		return nil, fmt.Errorf("result not found: %s", taskID)
	}

	return result, nil
}

// ListAgents returns all registered agents
func (c *Coordinator) ListAgents() []Agent {
	return c.registry.List()
}

// GetAgentStats returns statistics about agent performance
func (c *Coordinator) GetAgentStats() map[string]*AgentStats {
	stats := make(map[string]*AgentStats)

	agents := c.registry.List()
	for _, agent := range agents {
		health := agent.Health()
		stats[agent.ID()] = &AgentStats{
			AgentID:    agent.ID(),
			Type:       agent.Type(),
			Status:     agent.Status(),
			TaskCount:  health.TaskCount,
			ErrorCount: health.ErrorCount,
			ErrorRate:  health.ErrorRate,
			Uptime:     health.Uptime,
		}
	}

	return stats
}

// AgentStats contains agent statistics
type AgentStats struct {
	AgentID    string        `json:"agent_id"`
	Type       AgentType     `json:"type"`
	Status     AgentStatus   `json:"status"`
	TaskCount  int           `json:"task_count"`
	ErrorCount int           `json:"error_count"`
	ErrorRate  float64       `json:"error_rate"`
	Uptime     time.Duration `json:"uptime"`
}

// Shutdown gracefully shuts down all agents
func (c *Coordinator) Shutdown(ctx context.Context) error {
	agents := c.registry.List()

	for _, agent := range agents {
		if err := agent.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown agent %s: %w", agent.ID(), err)
		}
	}

	return nil
}

// ExecuteWorkflow executes a multi-step workflow
func (c *Coordinator) ExecuteWorkflow(ctx context.Context, workflow *Workflow) error {
	if c.workflowExecutor == nil {
		return fmt.Errorf("workflow executor not initialized")
	}
	return c.workflowExecutor.Execute(ctx, workflow)
}

// GetWorkflow retrieves a workflow by ID
func (c *Coordinator) GetWorkflow(id string) (*Workflow, error) {
	if c.workflowExecutor == nil {
		return nil, fmt.Errorf("workflow executor not initialized")
	}
	return c.workflowExecutor.GetWorkflow(id)
}

// ListWorkflows returns all workflows
func (c *Coordinator) ListWorkflows() []*Workflow {
	if c.workflowExecutor == nil {
		return []*Workflow{}
	}
	return c.workflowExecutor.ListWorkflows()
}

// GetCircuitBreakerState returns the state of a circuit breaker for an agent
func (c *Coordinator) GetCircuitBreakerState(agentID string) CircuitBreakerState {
	if c.circuitBreakers == nil {
		return CircuitBreakerClosed
	}
	return c.circuitBreakers.GetState(agentID)
}

// ResetCircuitBreaker resets a circuit breaker for an agent
func (c *Coordinator) ResetCircuitBreaker(agentID string) {
	if c.circuitBreakers != nil {
		c.circuitBreakers.Reset(agentID)
	}
}

// GetCircuitBreakerStats returns statistics for all circuit breakers
func (c *Coordinator) GetCircuitBreakerStats() map[string]CircuitBreakerState {
	if c.circuitBreakers == nil {
		return make(map[string]CircuitBreakerState)
	}
	return c.circuitBreakers.GetStats()
}
