package task

import (
	"context"
	"fmt"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DatabaseManager handles task lifecycle and operations with database persistence
type DatabaseManager struct {
	db database.DatabaseInterface
}

// NewDatabaseManager creates a new task manager with database persistence
func NewDatabaseManager(db database.DatabaseInterface) *DatabaseManager {
	return &DatabaseManager{
		db: db,
	}
}

// CreateTask creates a new task with database persistence
func (m *DatabaseManager) CreateTask(ctx context.Context, name, description, taskType, priority string, parameters map[string]interface{}, dependencies []string) (*Task, error) {
	// Convert priority string to TaskPriority
	var taskPriority TaskPriority
	switch priority {
	case "high":
		taskPriority = PriorityHigh
	case "critical":
		taskPriority = PriorityCritical
	case "low":
		taskPriority = PriorityLow
	default:
		taskPriority = PriorityNormal
	}

	// Convert dependencies to UUIDs
	var dependencyUUIDs []uuid.UUID
	for _, dep := range dependencies {
		if depUUID, err := uuid.Parse(dep); err == nil {
			dependencyUUIDs = append(dependencyUUIDs, depUUID)
		}
	}

	task := &Task{
		ID:           uuid.New(),
		Type:         TaskType(taskType),
		Data:         parameters,
		Status:       TaskStatusPending,
		Priority:     taskPriority,
		Criticality:  CriticalityNormal,
		Dependencies: dependencyUUIDs,
		MaxRetries:   3,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Insert into database
	query := `
		INSERT INTO distributed_tasks (
			id, task_type, task_data, status, priority, criticality, 
			dependencies, max_retries, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := m.db.QueryRow(ctx, query,
		task.ID, task.Type, task.Data, task.Status, task.Priority, task.Criticality,
		task.Dependencies, task.MaxRetries, task.CreatedAt, task.UpdatedAt,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create task in database: %v", err)
	}

	task.CreatedAt = createdAt
	task.UpdatedAt = updatedAt

	return task, nil
}

// GetTask retrieves a task by ID from database
func (m *DatabaseManager) GetTask(ctx context.Context, id string) (*Task, error) {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID: %v", err)
	}

	query := `
		SELECT 
			id, task_type, task_data, status, priority, criticality,
			assigned_worker_id, original_worker_id, dependencies,
			retry_count, max_retries, error_message, result_data,
			checkpoint_data, estimated_duration, started_at, completed_at,
			created_at, updated_at
		FROM distributed_tasks
		WHERE id = $1
	`

	var (
		dbID              uuid.UUID
		taskType          string
		taskData          map[string]interface{}
		status            string
		priority          int
		criticality       string
		assignedWorkerID  *uuid.UUID
		originalWorkerID  *uuid.UUID
		dependencies      []uuid.UUID
		retryCount        int
		maxRetries        int
		errorMessage      *string
		resultData        map[string]interface{}
		checkpointData    map[string]interface{}
		estimatedDuration *string
		startedAt         *time.Time
		completedAt       *time.Time
		createdAt         time.Time
		updatedAt         time.Time
	)

	err = m.db.QueryRow(ctx, query, taskID).Scan(
		&dbID, &taskType, &taskData, &status, &priority, &criticality,
		&assignedWorkerID, &originalWorkerID, &dependencies,
		&retryCount, &maxRetries, &errorMessage, &resultData,
		&checkpointData, &estimatedDuration, &startedAt, &completedAt,
		&createdAt, &updatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get task from database: %v", err)
	}

	// Convert priority int to TaskPriority
	var taskPriority TaskPriority
	switch priority {
	case 1:
		taskPriority = PriorityLow
	case 5:
		taskPriority = PriorityNormal
	case 10:
		taskPriority = PriorityHigh
	case 20:
		taskPriority = PriorityCritical
	default:
		taskPriority = PriorityNormal
	}

	task := &Task{
		ID:             dbID,
		Type:           TaskType(taskType),
		Data:           taskData,
		Status:         TaskStatus(status),
		Priority:       taskPriority,
		Criticality:    TaskCriticality(criticality),
		Dependencies:   dependencies,
		RetryCount:     retryCount,
		MaxRetries:     maxRetries,
		ErrorMessage:   getStringFromPtr(errorMessage),
		ResultData:     resultData,
		CheckpointData: checkpointData,
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	return task, nil
}

// ListTasks returns all tasks from database
func (m *DatabaseManager) ListTasks(ctx context.Context) ([]*Task, error) {
	query := `
		SELECT 
			id, task_type, task_data, status, priority, criticality,
			assigned_worker_id, original_worker_id, dependencies,
			retry_count, max_retries, error_message, result_data,
			checkpoint_data, estimated_duration, started_at, completed_at,
			created_at, updated_at
		FROM distributed_tasks
		ORDER BY created_at DESC
	`

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %v", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var (
			dbID              uuid.UUID
			taskType          string
			taskData          map[string]interface{}
			status            string
			priority          int
			criticality       string
			assignedWorkerID  *uuid.UUID
			originalWorkerID  *uuid.UUID
			dependencies      []uuid.UUID
			retryCount        int
			maxRetries        int
			errorMessage      *string
			resultData        map[string]interface{}
			checkpointData    map[string]interface{}
			estimatedDuration *string
			startedAt         *time.Time
			completedAt       *time.Time
			createdAt         time.Time
			updatedAt         time.Time
		)

		if err := rows.Scan(
			&dbID, &taskType, &taskData, &status, &priority, &criticality,
			&assignedWorkerID, &originalWorkerID, &dependencies,
			&retryCount, &maxRetries, &errorMessage, &resultData,
			&checkpointData, &estimatedDuration, &startedAt, &completedAt,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task row: %v", err)
		}

		// Convert priority int to TaskPriority
		var taskPriority TaskPriority
		switch priority {
		case 1:
			taskPriority = PriorityLow
		case 5:
			taskPriority = PriorityNormal
		case 10:
			taskPriority = PriorityHigh
		case 20:
			taskPriority = PriorityCritical
		default:
			taskPriority = PriorityNormal
		}

		// Convert nullable fields
		var errorMsg string
		if errorMessage != nil {
			errorMsg = *errorMessage
		}

		task := &Task{
			ID:             dbID,
			Type:           TaskType(taskType),
			Data:           taskData,
			Status:         TaskStatus(status),
			Priority:       taskPriority,
			Criticality:    TaskCriticality(criticality),
			Dependencies:   dependencies,
			RetryCount:     retryCount,
			MaxRetries:     maxRetries,
			ErrorMessage:   errorMsg,
			ResultData:     resultData,
			CheckpointData: checkpointData,
			StartedAt:      startedAt,
			CompletedAt:    completedAt,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task rows: %v", err)
	}

	return tasks, nil
}

// StartTask marks a task as running
func (m *DatabaseManager) StartTask(ctx context.Context, id string) error {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid task ID: %v", err)
	}

	query := `
		UPDATE distributed_tasks 
		SET status = 'running', started_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`

	result, err := m.db.Exec(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to start task: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found or not in pending state: %s", id)
	}

	return nil
}

// CompleteTask marks a task as completed
func (m *DatabaseManager) CompleteTask(ctx context.Context, id string, result map[string]interface{}) error {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid task ID: %v", err)
	}

	query := `
		UPDATE distributed_tasks 
		SET status = 'completed', result_data = $1, completed_at = NOW(), updated_at = NOW()
		WHERE id = $2 AND status = 'running'
	`

	execResult, err := m.db.Exec(ctx, query, result, taskID)
	if err != nil {
		return fmt.Errorf("failed to complete task: %v", err)
	}

	if execResult.RowsAffected() == 0 {
		return fmt.Errorf("task not found or not in running state: %s", id)
	}

	return nil
}

// FailTask marks a task as failed
func (m *DatabaseManager) FailTask(ctx context.Context, id, errorMessage string) error {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid task ID: %v", err)
	}

	query := `
		UPDATE distributed_tasks 
		SET status = 'failed', error_message = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := m.db.Exec(ctx, query, errorMessage, taskID)
	if err != nil {
		return fmt.Errorf("failed to mark task as failed: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

// DeleteTask deletes a task from database
func (m *DatabaseManager) DeleteTask(ctx context.Context, id string) error {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid task ID: %v", err)
	}

	query := `DELETE FROM distributed_tasks WHERE id = $1`

	result, err := m.db.Exec(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

// Helper function to convert pointer to string
func getStringFromPtr(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
