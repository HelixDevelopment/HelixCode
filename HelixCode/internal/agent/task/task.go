package task

import (
	"time"

	"github.com/google/uuid"
)

// Task represents a unit of work to be performed by an agent
type Task struct {
	ID          string     `json:"id"`
	Type        TaskType   `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    Priority   `json:"priority"`
	Status      TaskStatus `json:"status"`

	// Requirements
	RequiredCapabilities []string      `json:"required_capabilities"`
	EstimatedDuration    time.Duration `json:"estimated_duration"`
	Deadline             *time.Time    `json:"deadline,omitempty"`

	// Dependencies
	DependsOn []string `json:"depends_on"` // Task IDs
	BlockedBy []string `json:"blocked_by"` // Task IDs

	// Input/Output
	Input  map[string]interface{} `json:"input"`
	Output map[string]interface{} `json:"output"`

	// Execution
	AssignedTo  string        `json:"assigned_to"` // Agent ID
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration"`

	// Metadata
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	CreatedBy string                 `json:"created_by"` // Agent ID or "user"
	Tags      []string               `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// TaskType defines the category of a task
type TaskType string

const (
	TaskTypePlanning       TaskType = "planning"
	TaskTypeAnalysis       TaskType = "analysis"
	TaskTypeCodeGeneration TaskType = "code_generation"
	TaskTypeCodeEdit       TaskType = "code_edit"
	TaskTypeRefactoring    TaskType = "refactoring"
	TaskTypeTesting        TaskType = "testing"
	TaskTypeDebugging      TaskType = "debugging"
	TaskTypeReview         TaskType = "review"
	TaskTypeDocumentation  TaskType = "documentation"
	TaskTypeResearch       TaskType = "research"
)

// Priority defines task priority levels
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityNormal   Priority = 2
	PriorityHigh     Priority = 3
	PriorityCritical Priority = 4
)

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusReady      TaskStatus = "ready"
	StatusAssigned   TaskStatus = "assigned"
	StatusInProgress TaskStatus = "in_progress"
	StatusBlocked    TaskStatus = "blocked"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
	StatusCancelled  TaskStatus = "cancelled"
)

// NewTask creates a new task
func NewTask(taskType TaskType, title, description string, priority Priority) *Task {
	now := time.Now()
	return &Task{
		ID:          uuid.New().String(),
		Type:        taskType,
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      StatusPending,
		Input:       make(map[string]interface{}),
		Output:      make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        []string{},
		Metadata:    make(map[string]interface{}),
	}
}

// IsReady checks if a task is ready to be executed
func (t *Task) IsReady(completedTasks map[string]bool) bool {
	// Check if already completed or in progress
	if t.Status == StatusCompleted || t.Status == StatusInProgress {
		return false
	}

	// Check if blocked
	if len(t.BlockedBy) > 0 {
		return false
	}

	// Check dependencies
	for _, depID := range t.DependsOn {
		if !completedTasks[depID] {
			return false
		}
	}

	return true
}

// CanStart checks if task can be started now
func (t *Task) CanStart() bool {
	return t.Status == StatusReady || t.Status == StatusPending
}

// Start marks the task as started
func (t *Task) Start(agentID string) {
	now := time.Now()
	t.Status = StatusInProgress
	t.AssignedTo = agentID
	t.StartedAt = &now
	t.UpdatedAt = now
}

// Complete marks the task as completed
func (t *Task) Complete(output map[string]interface{}) {
	now := time.Now()
	t.Status = StatusCompleted
	t.CompletedAt = &now
	t.Output = output
	if t.StartedAt != nil {
		t.Duration = now.Sub(*t.StartedAt)
	}
	t.UpdatedAt = now
}

// Fail marks the task as failed
func (t *Task) Fail(reason string) {
	now := time.Now()
	t.Status = StatusFailed
	if t.Metadata == nil {
		t.Metadata = make(map[string]interface{})
	}
	t.Metadata["failure_reason"] = reason
	t.Metadata["failed_at"] = now
	t.UpdatedAt = now
}

// Block marks the task as blocked
func (t *Task) Block(reason string, blockedBy []string) {
	now := time.Now()
	t.Status = StatusBlocked
	t.BlockedBy = blockedBy
	if t.Metadata == nil {
		t.Metadata = make(map[string]interface{})
	}
	t.Metadata["block_reason"] = reason
	t.UpdatedAt = now
}

// Unblock removes blocks from the task
func (t *Task) Unblock() {
	if t.Status == StatusBlocked {
		t.Status = StatusReady
		t.BlockedBy = []string{}
		t.UpdatedAt = time.Now()
	}
}

// Cancel marks the task as cancelled
func (t *Task) Cancel(reason string) {
	now := time.Now()
	t.Status = StatusCancelled
	if t.Metadata == nil {
		t.Metadata = make(map[string]interface{})
	}
	t.Metadata["cancellation_reason"] = reason
	t.UpdatedAt = now
}

// IsCompleted checks if the task is completed
func (t *Task) IsCompleted() bool {
	return t.Status == StatusCompleted
}

// IsFailed checks if the task failed
func (t *Task) IsFailed() bool {
	return t.Status == StatusFailed
}

// IsActive checks if the task is currently being worked on
func (t *Task) IsActive() bool {
	return t.Status == StatusInProgress
}

// Result represents the outcome of a task execution
type Result struct {
	TaskID     string                 `json:"task_id"`
	AgentID    string                 `json:"agent_id"`
	Success    bool                   `json:"success"`
	Output     map[string]interface{} `json:"output"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Confidence float64                `json:"confidence"` // 0.0 to 1.0
	Artifacts  []Artifact             `json:"artifacts"`
	Metrics    *TaskMetrics           `json:"metrics"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Artifact represents a file or resource created by a task
type Artifact struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "code", "test", "doc", "config"
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	Size      int64     `json:"size"`
	Checksum  string    `json:"checksum"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskMetrics contains metrics about task execution
type TaskMetrics struct {
	TokensUsed     int           `json:"tokens_used"`
	LLMCalls       int           `json:"llm_calls"`
	ToolCalls      int           `json:"tool_calls"`
	FilesModified  int           `json:"files_modified"`
	LinesAdded     int           `json:"lines_added"`
	LinesRemoved   int           `json:"lines_removed"`
	TestsGenerated int           `json:"tests_generated"`
	ExecutionTime  time.Duration `json:"execution_time"`
}

// NewResult creates a new result
func NewResult(taskID, agentID string) *Result {
	return &Result{
		TaskID:     taskID,
		AgentID:    agentID,
		Success:    false,
		Output:     make(map[string]interface{}),
		Artifacts:  []Artifact{},
		Confidence: 0.0,
		Timestamp:  time.Now(),
	}
}

// SetSuccess marks the result as successful
func (r *Result) SetSuccess(output map[string]interface{}, confidence float64) {
	r.Success = true
	r.Output = output
	r.Confidence = confidence
}

// SetFailure marks the result as failed
func (r *Result) SetFailure(err error) {
	r.Success = false
	if err != nil {
		r.Error = err.Error()
	}
	r.Confidence = 0.0
}

// AddArtifact adds an artifact to the result
func (r *Result) AddArtifact(artifact Artifact) {
	r.Artifacts = append(r.Artifacts, artifact)
}
