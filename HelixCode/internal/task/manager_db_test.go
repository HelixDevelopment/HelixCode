package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

// ========================================
// DatabaseManager Tests with MockDatabase
// ========================================

func TestDatabaseManager_CreateTaskSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	now := time.Now()

	// Mock QueryRow to return created_at and updated_at
	mockRow := database.NewMockRowWithValues(now, now)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	task, err := dm.CreateTask(ctx, "Test Task", "Description", "planning", "high",
		map[string]interface{}{"key": "value"}, []string{})

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.NotEqual(t, uuid.Nil, task.ID)
	assert.Equal(t, TaskType("planning"), task.Type)
	assert.Equal(t, PriorityHigh, task.Priority)
	assert.Equal(t, TaskStatusPending, task.Status)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_CreateTaskPriorityMapping(t *testing.T) {
	tests := []struct {
		name         string
		priority     string
		expectedPrio TaskPriority
	}{
		{"high priority", "high", PriorityHigh},
		{"critical priority", "critical", PriorityCritical},
		{"low priority", "low", PriorityLow},
		{"default priority", "unknown", PriorityNormal},
		{"normal priority", "normal", PriorityNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := database.NewMockDatabase()
			dm := NewDatabaseManager(mockDB)

			ctx := context.Background()
			now := time.Now()

			mockRow := database.NewMockRowWithValues(now, now)
			mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

			task, err := dm.CreateTask(ctx, "Test", "Desc", "planning", tt.priority,
				map[string]interface{}{}, []string{})

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPrio, task.Priority)
			mockDB.AssertExpectations(t)
		})
	}
}

func TestDatabaseManager_CreateTaskDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	dbError := errors.New("database connection failed")

	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	task, err := dm.CreateTask(ctx, "Test Task", "Description", "planning", "high",
		map[string]interface{}{"key": "value"}, []string{})

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "failed to create task in database")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_GetTaskSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()
	now := time.Now()

	// Create mock row with all required fields
	mockRow := database.NewMockRowWithValues(
		taskID,                        // id
		"planning",                    // task_type
		map[string]interface{}{},      // task_data
		"pending",                     // status
		5,                             // priority
		"normal",                      // criticality
		nil,                           // assigned_worker_id
		nil,                           // original_worker_id
		[]uuid.UUID{},                 // dependencies
		0,                             // retry_count
		3,                             // max_retries
		nil,                           // error_message
		map[string]interface{}{},      // result_data
		map[string]interface{}{},      // checkpoint_data
		nil,                           // estimated_duration
		nil,                           // started_at
		nil,                           // completed_at
		now,                           // created_at
		now,                           // updated_at
	)

	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	task, err := dm.GetTask(ctx, taskID.String())

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, taskID, task.ID)
	assert.Equal(t, TaskType("planning"), task.Type)
	assert.Equal(t, TaskStatus("pending"), task.Status)
	assert.Equal(t, PriorityNormal, task.Priority)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_GetTaskInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	task, err := dm.GetTask(ctx, "invalid-uuid")

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestDatabaseManager_GetTaskNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()

	mockRow := database.NewMockRowWithError(pgx.ErrNoRows)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	task, err := dm.GetTask(ctx, taskID.String())

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "task not found")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_GetTaskPriorityConversion(t *testing.T) {
	tests := []struct {
		name         string
		dbPriority   int
		expectedPrio TaskPriority
	}{
		{"low priority", 1, PriorityLow},
		{"normal priority", 5, PriorityNormal},
		{"high priority", 10, PriorityHigh},
		{"critical priority", 20, PriorityCritical},
		{"unknown priority", 999, PriorityNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := database.NewMockDatabase()
			dm := NewDatabaseManager(mockDB)

			ctx := context.Background()
			taskID := uuid.New()
			now := time.Now()

			mockRow := database.NewMockRowWithValues(
				taskID, "planning", map[string]interface{}{}, "pending",
				tt.dbPriority, "normal", nil, nil, []uuid.UUID{},
				0, 3, nil, map[string]interface{}{}, map[string]interface{}{},
				nil, nil, nil, now, now,
			)

			mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

			task, err := dm.GetTask(ctx, taskID.String())

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPrio, task.Priority)
			mockDB.AssertExpectations(t)
		})
	}
}

func TestDatabaseManager_ListTasksSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	now := time.Now()

	// Create mock rows for two tasks
	taskID1 := uuid.New()
	taskID2 := uuid.New()

	mockRows := database.NewMockRows([][]interface{}{
		{
			taskID1, "planning", map[string]interface{}{}, "pending",
			5, "normal", nil, nil, []uuid.UUID{},
			0, 3, nil, map[string]interface{}{}, map[string]interface{}{},
			nil, nil, nil, now, now,
		},
		{
			taskID2, "building", map[string]interface{}{}, "running",
			10, "high", nil, nil, []uuid.UUID{},
			0, 3, nil, map[string]interface{}{}, map[string]interface{}{},
			nil, nil, nil, now, now,
		},
	})

	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	tasks, err := dm.ListTasks(ctx)

	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, taskID1, tasks[0].ID)
	assert.Equal(t, taskID2, tasks[1].ID)
	assert.Equal(t, TaskType("planning"), tasks[0].Type)
	assert.Equal(t, TaskType("building"), tasks[1].Type)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_ListTasksEmpty(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	mockRows := database.NewMockRows([][]interface{}{})
	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	tasks, err := dm.ListTasks(ctx)

	assert.NoError(t, err)
	assert.Len(t, tasks, 0)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_ListTasksQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	dbError := errors.New("query failed")

	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(nil, dbError)

	tasks, err := dm.ListTasks(ctx)

	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "failed to query tasks")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_StartTaskSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()

	// Mock successful update (1 row affected)
	mockDB.MockExecSuccess(1)

	err := dm.StartTask(ctx, taskID.String())

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_StartTaskInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	err := dm.StartTask(ctx, "invalid-uuid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestDatabaseManager_StartTaskNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()

	// Mock no rows affected
	mockDB.MockExecSuccess(0)

	err := dm.StartTask(ctx, taskID.String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found or not in pending state")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_StartTaskDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()
	dbError := errors.New("database error")

	mockDB.MockExecError(dbError)

	err := dm.StartTask(ctx, taskID.String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start task")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_CompleteTaskSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()
	result := map[string]interface{}{
		"output": "success",
	}

	mockDB.MockExecSuccess(1)

	err := dm.CompleteTask(ctx, taskID.String(), result)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_CompleteTaskInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	result := map[string]interface{}{"output": "success"}

	err := dm.CompleteTask(ctx, "invalid-uuid", result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestDatabaseManager_CompleteTaskNotRunning(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()
	result := map[string]interface{}{"output": "success"}

	// Mock no rows affected (task not in running state)
	mockDB.MockExecSuccess(0)

	err := dm.CompleteTask(ctx, taskID.String(), result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found or not in running state")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_FailTaskSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()
	errorMessage := "Task failed due to error"

	mockDB.MockExecSuccess(1)

	err := dm.FailTask(ctx, taskID.String(), errorMessage)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_FailTaskInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	err := dm.FailTask(ctx, "invalid-uuid", "error message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestDatabaseManager_FailTaskNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()

	mockDB.MockExecSuccess(0)

	err := dm.FailTask(ctx, taskID.String(), "error message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_DeleteTaskSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()

	mockDB.MockExecSuccess(1)

	err := dm.DeleteTask(ctx, taskID.String())

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_DeleteTaskInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	err := dm.DeleteTask(ctx, "invalid-uuid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestDatabaseManager_DeleteTaskNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	taskID := uuid.New()

	mockDB.MockExecSuccess(0)

	err := dm.DeleteTask(ctx, taskID.String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
	mockDB.AssertExpectations(t)
}

// ========================================
// Helper Function Tests
// ========================================
//
// Note: TestGetStringFromPtr is in manager_test.go
