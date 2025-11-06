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
	registry     *AgentRegistry
	tasks        map[string]*task.Task
	taskQueue    []*task.Task
	results      map[string]*task.Result
	mu           sync.RWMutex
	config       *CoordinatorConfig
}

// CoordinatorConfig holds coordinator configuration
type CoordinatorConfig struct {
	MaxConcurrentTasks int
	TaskTimeout        time.Duration
	EnableCollaboration bool
	ConflictResolution ResolutionMethod
}

// NewCoordinator creates a new agent coordinator
func NewCoordinator(config *CoordinatorConfig) *Coordinator {
	if config == nil {
		config = &CoordinatorConfig{
			MaxConcurrentTasks: 10,
			TaskTimeout:        30 * time.Minute,
			EnableCollaboration: true,
			ConflictResolution: ResolutionMethodVoting,
		}
	}

	return &Coordinator{
		registry:  NewAgentRegistry(),
		tasks:     make(map[string]*task.Task),
		taskQueue: make([]*task.Task, 0),
		results:   make(map[string]*task.Result),
		config:    config,
	}
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

	// Execute task
	t.Start(agent.ID())
	result, err := agent.Execute(ctx, t)

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
