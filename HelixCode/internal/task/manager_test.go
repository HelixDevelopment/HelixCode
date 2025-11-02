package task

import (
	"testing"
	"time"

	"dev.helix.code/internal/database"
	"dev.helix.code/internal/redis"
	"github.com/google/uuid"
)

// MockDatabase creates a mock database for testing
func MockDatabase() *database.Database {
	// In a real test, you would use a test database
	// For now, return nil since we're testing the logic
	return nil
}

// MockRedis creates a mock Redis client for testing
func MockRedis() *redis.Client {
	// Create a disabled Redis client for testing
	return &redis.Client{}
}

func TestTaskManager_CreateTask(t *testing.T) {
	tm := NewTaskManager(MockDatabase(), MockRedis())

	task, err := tm.CreateTask(
		TaskTypePlanning,
		map[string]interface{}{
			"description": "Test planning task",
		},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)

	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if task.ID == uuid.Nil {
		t.Error("Task ID should not be nil")
	}

	if task.Type != TaskTypePlanning {
		t.Errorf("Expected task type %s, got %s", TaskTypePlanning, task.Type)
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Expected task status %s, got %s", TaskStatusPending, task.Status)
	}

	if task.Priority != PriorityNormal {
		t.Errorf("Expected task priority %d, got %d", PriorityNormal, task.Priority)
	}
}

func TestTaskManager_CompleteTask(t *testing.T) {
	tm := NewTaskManager(MockDatabase(), MockRedis())

	task, err := tm.CreateTask(
		TaskTypeBuilding,
		map[string]interface{}{
			"description": "Test building task",
		},
		PriorityHigh,
		CriticalityNormal,
		[]uuid.UUID{},
	)

	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	result := map[string]interface{}{
		"output":   "Build completed successfully",
		"duration": "2m30s",
	}

	err = tm.CompleteTask(task.ID, result)
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	// In a real test, we would retrieve the task and verify its status
	// For now, we'll just verify no error occurred
}

func TestTaskManager_FailTask(t *testing.T) {
	tm := NewTaskManager(MockDatabase(), MockRedis())

	task, err := tm.CreateTask(
		TaskTypeTesting,
		map[string]interface{}{
			"description": "Test testing task",
		},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)

	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	err = tm.FailTask(task.ID, "Test failure")
	if err != nil {
		t.Fatalf("Failed to mark task as failed: %v", err)
	}

	// In a real test, we would verify the task status and retry count
}

func TestTaskQueue_AddAndGet(t *testing.T) {
	tq := NewTaskQueue()

	// Create test tasks
	highPriorityTask := &Task{
		ID:          uuid.New(),
		Type:        TaskTypePlanning,
		Priority:    PriorityHigh,
		Criticality: CriticalityHigh,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	normalPriorityTask := &Task{
		ID:          uuid.New(),
		Type:        TaskTypeBuilding,
		Priority:    PriorityNormal,
		Criticality: CriticalityNormal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	lowPriorityTask := &Task{
		ID:          uuid.New(),
		Type:        TaskTypeTesting,
		Priority:    PriorityLow,
		Criticality: CriticalityLow,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Add tasks to queue
	tq.AddTask(highPriorityTask)
	tq.AddTask(normalPriorityTask)
	tq.AddTask(lowPriorityTask)

	// Get next task - should be high priority
	nextTask := tq.GetNextTask()
	if nextTask == nil {
		t.Fatal("Expected next task, got nil")
	}

	if nextTask.ID != highPriorityTask.ID {
		t.Errorf("Expected high priority task, got task with priority %d", nextTask.Priority)
	}

	// Get next task - should be normal priority
	nextTask = tq.GetNextTask()
	if nextTask == nil {
		t.Fatal("Expected next task, got nil")
	}

	if nextTask.ID != normalPriorityTask.ID {
		t.Errorf("Expected normal priority task, got task with priority %d", nextTask.Priority)
	}

	// Get next task - should be low priority
	nextTask = tq.GetNextTask()
	if nextTask == nil {
		t.Fatal("Expected next task, got nil")
	}

	if nextTask.ID != lowPriorityTask.ID {
		t.Errorf("Expected low priority task, got task with priority %d", nextTask.Priority)
	}

	// Queue should be empty now
	nextTask = tq.GetNextTask()
	if nextTask != nil {
		t.Error("Expected nil when queue is empty")
	}
}

func TestTaskQueue_Stats(t *testing.T) {
	tq := NewTaskQueue()

	// Add some tasks
	tq.AddTask(&Task{
		ID:       uuid.New(),
		Priority: PriorityHigh,
	})
	tq.AddTask(&Task{
		ID:       uuid.New(),
		Priority: PriorityNormal,
	})
	tq.AddTask(&Task{
		ID:       uuid.New(),
		Priority: PriorityLow,
	})

	stats := tq.GetQueueStats()

	if stats.HighPriority != 1 {
		t.Errorf("Expected 1 high priority task, got %d", stats.HighPriority)
	}

	if stats.NormalPriority != 1 {
		t.Errorf("Expected 1 normal priority task, got %d", stats.NormalPriority)
	}

	if stats.LowPriority != 1 {
		t.Errorf("Expected 1 low priority task, got %d", stats.LowPriority)
	}

	if stats.Total != 3 {
		t.Errorf("Expected 3 total tasks, got %d", stats.Total)
	}
}

func TestTaskManager_GetTaskProgress(t *testing.T) {
	tm := NewTaskManager(MockDatabase(), MockRedis())

	task, err := tm.CreateTask(
		TaskTypeRefactoring,
		map[string]interface{}{
			"description": "Test refactoring task",
		},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)

	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	progress, err := tm.GetTaskProgress(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task progress: %v", err)
	}

	if progress.TaskID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, progress.TaskID)
	}

	if progress.Status != TaskStatusPending {
		t.Errorf("Expected status %s, got %s", TaskStatusPending, progress.Status)
	}

	if progress.Progress != 0.0 {
		t.Errorf("Expected progress 0.0, got %f", progress.Progress)
	}
}
