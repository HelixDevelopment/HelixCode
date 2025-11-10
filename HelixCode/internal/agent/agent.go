package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/agent/task"
	"github.com/google/uuid"
)

// Agent represents an intelligent agent that can perform tasks
type Agent interface {
	// Identity
	ID() string
	Type() AgentType
	Name() string

	// Capabilities
	Capabilities() []Capability
	CanHandle(task *task.Task) bool

	// Execution
	Execute(ctx context.Context, task *task.Task) (*task.Result, error)

	// Collaboration
	Collaborate(ctx context.Context, agents []Agent, task *task.Task) (*CollaborationResult, error)

	// Lifecycle
	Initialize(ctx context.Context, config *AgentConfig) error
	Shutdown(ctx context.Context) error

	// Status
	Status() AgentStatus
	Health() *HealthCheck
}

// AgentType defines the type/role of an agent
type AgentType string

const (
	AgentTypePlanning      AgentType = "planning"
	AgentTypeCoding        AgentType = "coding"
	AgentTypeTesting       AgentType = "testing"
	AgentTypeDebugging     AgentType = "debugging"
	AgentTypeReview        AgentType = "review"
	AgentTypeRefactoring   AgentType = "refactoring"
	AgentTypeDocumentation AgentType = "documentation"
	AgentTypeCoordinator   AgentType = "coordinator"
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	StatusIdle     AgentStatus = "idle"
	StatusBusy     AgentStatus = "busy"
	StatusWaiting  AgentStatus = "waiting"
	StatusError    AgentStatus = "error"
	StatusShutdown AgentStatus = "shutdown"
)

// Capability represents a specific capability an agent has
type Capability string

const (
	CapabilityPlanning            Capability = "planning"
	CapabilityCodeGeneration      Capability = "code_generation"
	CapabilityCodeAnalysis        Capability = "code_analysis"
	CapabilityTestGeneration      Capability = "test_generation"
	CapabilityTestExecution       Capability = "test_execution"
	CapabilityDebugging           Capability = "debugging"
	CapabilityRefactoring         Capability = "refactoring"
	CapabilityDocumentation       Capability = "documentation"
	CapabilityCodeReview          Capability = "code_review"
	CapabilitySecurityAudit       Capability = "security_audit"
	CapabilityPerformanceAnalysis Capability = "performance_analysis"
)

// AgentConfig holds configuration for an agent
type AgentConfig struct {
	ID           string                 `json:"id"`
	Type         AgentType              `json:"type"`
	Name         string                 `json:"name"`
	Model        string                 `json:"model"`    // LLM model to use
	Provider     string                 `json:"provider"` // LLM provider
	Temperature  float64                `json:"temperature"`
	MaxTokens    int                    `json:"max_tokens"`
	Capabilities []Capability           `json:"capabilities"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	mu           sync.Mutex
	id           string
	agentType    AgentType
	name         string
	capabilities []Capability
	status       AgentStatus
	config       *AgentConfig
	startTime    time.Time
	taskCount    int
	errorCount   int
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(config *AgentConfig) *BaseAgent {
	return &BaseAgent{
		id:           config.ID,
		agentType:    config.Type,
		name:         config.Name,
		capabilities: config.Capabilities,
		status:       StatusIdle,
		config:       config,
		startTime:    time.Now(),
		taskCount:    0,
		errorCount:   0,
	}
}

// ID returns the agent's unique identifier
func (a *BaseAgent) ID() string {
	return a.id
}

// Type returns the agent's type
func (a *BaseAgent) Type() AgentType {
	return a.agentType
}

// Name returns the agent's name
func (a *BaseAgent) Name() string {
	return a.name
}

// Capabilities returns the agent's capabilities
func (a *BaseAgent) Capabilities() []Capability {
	return a.capabilities
}

// Status returns the agent's current status
func (a *BaseAgent) Status() AgentStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

// SetStatus updates the agent's status
func (a *BaseAgent) SetStatus(status AgentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = status
}

// CanHandle checks if the agent can handle a task based on capabilities
func (a *BaseAgent) CanHandle(task *task.Task) bool {
	if task == nil {
		return false
	}

	// Check if agent has required capabilities
	for _, reqCap := range task.RequiredCapabilities {
		found := false
		for _, agentCap := range a.capabilities {
			if agentCap == Capability(reqCap) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// IncrementTaskCount increments the task counter
func (a *BaseAgent) IncrementTaskCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.taskCount++
}

// IncrementErrorCount increments the error counter
func (a *BaseAgent) IncrementErrorCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.errorCount++
}

// Health returns health check information
func (a *BaseAgent) Health() *HealthCheck {
	a.mu.Lock()
	defer a.mu.Unlock()

	uptime := time.Since(a.startTime)
	errorRate := float64(0)
	if a.taskCount > 0 {
		errorRate = float64(a.errorCount) / float64(a.taskCount)
	}

	healthy := a.status != StatusError && a.status != StatusShutdown && errorRate < 0.2

	return &HealthCheck{
		AgentID:    a.id,
		Healthy:    healthy,
		Status:     a.status,
		Uptime:     uptime,
		TaskCount:  a.taskCount,
		ErrorCount: a.errorCount,
		ErrorRate:  errorRate,
		Timestamp:  time.Now(),
	}
}

// HealthCheck represents agent health information
type HealthCheck struct {
	AgentID    string        `json:"agent_id"`
	Healthy    bool          `json:"healthy"`
	Status     AgentStatus   `json:"status"`
	Uptime     time.Duration `json:"uptime"`
	TaskCount  int           `json:"task_count"`
	ErrorCount int           `json:"error_count"`
	ErrorRate  float64       `json:"error_rate"`
	Timestamp  time.Time     `json:"timestamp"`
}

// CollaborationResult represents the result of agent collaboration
type CollaborationResult struct {
	Success      bool                    `json:"success"`
	Results      map[string]*task.Result `json:"results"`   // Results from each agent
	Consensus    *task.Result            `json:"consensus"` // Agreed-upon result
	Conflicts    []*Conflict             `json:"conflicts"` // Any conflicts that occurred
	Duration     time.Duration           `json:"duration"`
	Participants []string                `json:"participants"` // Agent IDs
	Messages     []*CollaborationMessage `json:"messages"`     // Communication log
}

// CollaborationMessage represents a message between agents
type CollaborationMessage struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"` // Agent ID
	To        string                 `json:"to"`   // Agent ID (or "all" for broadcast)
	Type      MessageType            `json:"type"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// MessageType defines types of inter-agent messages
type MessageType string

const (
	MessageTypeRequest      MessageType = "request"
	MessageTypeResponse     MessageType = "response"
	MessageTypeProposal     MessageType = "proposal"
	MessageTypeAgreement    MessageType = "agreement"
	MessageTypeDisagreement MessageType = "disagreement"
	MessageTypeQuestion     MessageType = "question"
	MessageTypeAnswer       MessageType = "answer"
	MessageTypeBroadcast    MessageType = "broadcast"
)

// Conflict represents a disagreement between agents
type Conflict struct {
	ID         string      `json:"id"`
	Agents     []string    `json:"agents"`      // Agent IDs involved
	Issue      string      `json:"issue"`       // What they disagree on
	Proposals  []*Proposal `json:"proposals"`   // Each agent's proposal
	Resolution *Resolution `json:"resolution"`  // How it was resolved
	ResolvedBy string      `json:"resolved_by"` // Agent ID or "vote" or "coordinator"
	Timestamp  time.Time   `json:"timestamp"`
}

// Proposal represents an agent's proposed solution
type Proposal struct {
	AgentID    string                 `json:"agent_id"`
	Solution   string                 `json:"solution"`
	Reasoning  string                 `json:"reasoning"`
	Confidence float64                `json:"confidence"`
	Supporting map[string]interface{} `json:"supporting"` // Supporting evidence
}

// Resolution represents how a conflict was resolved
type Resolution struct {
	Method      ResolutionMethod `json:"method"`
	Winner      string           `json:"winner"` // Agent ID whose proposal was chosen
	Explanation string           `json:"explanation"`
	Final       *Proposal        `json:"final"` // Final agreed solution
}

// ResolutionMethod defines how conflicts are resolved
type ResolutionMethod string

const (
	ResolutionMethodVoting         ResolutionMethod = "voting"
	ResolutionMethodCoordinator    ResolutionMethod = "coordinator"
	ResolutionMethodConsensus      ResolutionMethod = "consensus"
	ResolutionMethodHighConfidence ResolutionMethod = "high_confidence"
)

// AgentRegistry maintains a registry of available agents
type AgentRegistry struct {
	agents map[string]Agent
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]Agent),
	}
}

// Register registers an agent
func (r *AgentRegistry) Register(agent Agent) error {
	if agent == nil {
		return ErrNilAgent
	}
	r.agents[agent.ID()] = agent
	return nil
}

// Unregister removes an agent from the registry
func (r *AgentRegistry) Unregister(agentID string) {
	delete(r.agents, agentID)
}

// Get retrieves an agent by ID
func (r *AgentRegistry) Get(agentID string) (Agent, error) {
	agent, exists := r.agents[agentID]
	if !exists {
		return nil, ErrAgentNotFound
	}
	return agent, nil
}

// GetByType retrieves all agents of a specific type
func (r *AgentRegistry) GetByType(agentType AgentType) []Agent {
	var result []Agent
	for _, agent := range r.agents {
		if agent.Type() == agentType {
			result = append(result, agent)
		}
	}
	return result
}

// GetByCapability retrieves all agents with a specific capability
func (r *AgentRegistry) GetByCapability(capability Capability) []Agent {
	var result []Agent
	for _, agent := range r.agents {
		for _, cap := range agent.Capabilities() {
			if cap == capability {
				result = append(result, agent)
				break
			}
		}
	}
	return result
}

// List returns all registered agents
func (r *AgentRegistry) List() []Agent {
	result := make([]Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		result = append(result, agent)
	}
	return result
}

// Count returns the number of registered agents
func (r *AgentRegistry) Count() int {
	return len(r.agents)
}

// GenerateAgentID generates a unique agent ID
func GenerateAgentID(agentType AgentType) string {
	return string(agentType) + "-" + uuid.New().String()[:8]
}

// Errors
var (
	ErrNilAgent      = fmt.Errorf("agent cannot be nil")
	ErrAgentNotFound = fmt.Errorf("agent not found")
	ErrInvalidTask   = fmt.Errorf("invalid task")
	ErrTaskFailed    = fmt.Errorf("task execution failed")
)
