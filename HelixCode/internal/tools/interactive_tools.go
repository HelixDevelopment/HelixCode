package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AskUserTool implements interactive user questions
type AskUserTool struct {
	registry *ToolRegistry
}

func (t *AskUserTool) Name() string { return "ask_user" }

func (t *AskUserTool) Description() string {
	return "Ask the user a question and wait for their response"
}

func (t *AskUserTool) Category() ToolCategory {
	return CategoryInteractive
}

func (t *AskUserTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "Question to ask the user",
			},
			"options": map[string]interface{}{
				"type":        "array",
				"description": "Optional list of predefined options",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"default": map[string]interface{}{
				"type":        "string",
				"description": "Default answer if user doesn't respond",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds (default: no timeout)",
			},
		},
		Required:    []string{"question"},
		Description: "Ask the user a question and wait for their response",
	}
}

func (t *AskUserTool) Validate(params map[string]interface{}) error {
	if _, ok := params["question"]; !ok {
		return fmt.Errorf("question is required")
	}
	return nil
}

func (t *AskUserTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	question := params["question"].(string)

	response := &UserResponse{
		Question:  question,
		Timestamp: time.Now(),
	}

	if options, ok := params["options"].([]interface{}); ok {
		response.Options = make([]string, len(options))
		for i, opt := range options {
			response.Options[i] = opt.(string)
		}
	}

	if defaultVal, ok := params["default"].(string); ok {
		response.Default = defaultVal
	}

	// In a real implementation, this would interact with the user
	// For now, return the structure that should be filled by the user interface
	return response, nil
}

// UserResponse represents a user's response to a question
type UserResponse struct {
	Question  string    `json:"question"`
	Options   []string  `json:"options,omitempty"`
	Default   string    `json:"default,omitempty"`
	Answer    string    `json:"answer"`
	Timestamp time.Time `json:"timestamp"`
}

// TaskTrackerTool implements task tracking
type TaskTrackerTool struct {
	registry *ToolRegistry
	tasks    map[string]*Task
	mu       sync.RWMutex
}

func (t *TaskTrackerTool) Name() string { return "task_tracker" }

func (t *TaskTrackerTool) Description() string {
	return "Track and manage tasks during execution"
}

func (t *TaskTrackerTool) Category() ToolCategory {
	return CategoryInteractive
}

func (t *TaskTrackerTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Action to perform: create, update, list, get, complete",
			},
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "Task ID (for update, get, complete)",
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Task title (for create)",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Task description (for create/update)",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "Task status: pending, in_progress, completed, failed",
			},
			"progress": map[string]interface{}{
				"type":        "integer",
				"description": "Progress percentage (0-100)",
			},
		},
		Required:    []string{"action"},
		Description: "Track and manage tasks during execution",
	}
}

func (t *TaskTrackerTool) Validate(params map[string]interface{}) error {
	if _, ok := params["action"]; !ok {
		return fmt.Errorf("action is required")
	}

	action := params["action"].(string)
	validActions := map[string]bool{
		"create":   true,
		"update":   true,
		"list":     true,
		"get":      true,
		"complete": true,
	}

	if !validActions[action] {
		return fmt.Errorf("invalid action: %s", action)
	}

	return nil
}

func (t *TaskTrackerTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if t.tasks == nil {
		t.tasks = make(map[string]*Task)
	}

	action := params["action"].(string)

	switch action {
	case "create":
		return t.createTask(params)
	case "update":
		return t.updateTask(params)
	case "list":
		return t.listTasks()
	case "get":
		return t.getTask(params)
	case "complete":
		return t.completeTask(params)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (t *TaskTrackerTool) createTask(params map[string]interface{}) (*Task, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	task := &Task{
		ID:        fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if title, ok := params["title"].(string); ok {
		task.Title = title
	}

	if desc, ok := params["description"].(string); ok {
		task.Description = desc
	}

	t.tasks[task.ID] = task
	return task, nil
}

func (t *TaskTrackerTool) updateTask(params map[string]interface{}) (*Task, error) {
	taskID, ok := params["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required for update")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	task, exists := t.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	if status, ok := params["status"].(string); ok {
		task.Status = TaskStatus(status)
	}

	if progress, ok := params["progress"].(int); ok {
		task.Progress = progress
	}

	if desc, ok := params["description"].(string); ok {
		task.Description = desc
	}

	task.UpdatedAt = time.Now()
	return task, nil
}

func (t *TaskTrackerTool) listTasks() ([]*Task, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tasks := make([]*Task, 0, len(t.tasks))
	for _, task := range t.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (t *TaskTrackerTool) getTask(params map[string]interface{}) (*Task, error) {
	taskID, ok := params["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required for get")
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	task, exists := t.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

func (t *TaskTrackerTool) completeTask(params map[string]interface{}) (*Task, error) {
	taskID, ok := params["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required for complete")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	task, exists := t.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	task.Status = TaskStatusCompleted
	task.Progress = 100
	task.CompletedAt = time.Now()
	task.UpdatedAt = time.Now()

	return task, nil
}

// Task represents a tracked task
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Progress    int        `json:"progress"` // 0-100
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)
