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

// ========================================
// Checkpoint Manager Tests
// ========================================

func TestNewCheckpointManager(t *testing.T) {
	cm := NewCheckpointManager(MockDatabase())
	if cm == nil {
		t.Fatal("Expected checkpoint manager, got nil")
	}
}

func TestCheckpointManager_CreateCheckpoint(t *testing.T) {
	// Skip these tests as they require a real database
	// The functions will panic with nil database
	t.Skip("Checkpoint tests require real database - skipping for coverage")
}

func TestCheckpointManager_GetCheckpoints(t *testing.T) {
	// Skip these tests as they require a real database
	t.Skip("Checkpoint tests require real database - skipping for coverage")
}

func TestCheckpointManager_GetLatestCheckpoint(t *testing.T) {
	// Skip these tests as they require a real database
	t.Skip("Checkpoint tests require real database - skipping for coverage")
}

func TestCheckpointManager_DeleteCheckpoint(t *testing.T) {
	// Skip these tests as they require a real database
	t.Skip("Checkpoint tests require real database - skipping for coverage")
}

func TestCheckpointManager_DeleteAllCheckpoints(t *testing.T) {
	// Skip these tests as they require a real database
	t.Skip("Checkpoint tests require real database - skipping for coverage")
}

// ========================================
// Dependency Manager Tests
// ========================================

func TestNewDependencyManager(t *testing.T) {
	dm := NewDependencyManager(MockDatabase())
	if dm == nil {
		t.Fatal("Expected dependency manager, got nil")
	}
}

func TestDependencyManager_ValidateDependencies(t *testing.T) {
	t.Run("with empty dependencies", func(t *testing.T) {
		dm := NewDependencyManager(MockDatabase())
		err := dm.ValidateDependencies([]uuid.UUID{})
		if err != nil {
			t.Errorf("Expected no error with empty dependencies, got %v", err)
		}
	})

	t.Run("with nil dependencies", func(t *testing.T) {
		dm := NewDependencyManager(MockDatabase())
		err := dm.ValidateDependencies(nil)
		if err != nil {
			t.Errorf("Expected no error with nil dependencies, got %v", err)
		}
	})

	t.Run("with dependencies", func(t *testing.T) {
		t.Skip("Requires real database connection")
	})
}

func TestDependencyManager_CheckDependenciesCompleted(t *testing.T) {
	t.Run("with empty dependencies", func(t *testing.T) {
		dm := NewDependencyManager(MockDatabase())
		completed, err := dm.CheckDependenciesCompleted([]uuid.UUID{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !completed {
			t.Error("Expected true for empty dependencies")
		}
	})

	t.Run("with dependencies", func(t *testing.T) {
		t.Skip("Requires real database connection")
	})
}

func TestDependencyManager_GetBlockingDependencies(t *testing.T) {
	t.Run("with empty dependencies", func(t *testing.T) {
		dm := NewDependencyManager(MockDatabase())
		blocking, err := dm.GetBlockingDependencies([]uuid.UUID{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(blocking) != 0 {
			t.Errorf("Expected 0 blocking dependencies, got %d", len(blocking))
		}
	})

	t.Run("with dependencies", func(t *testing.T) {
		t.Skip("Requires real database connection")
	})
}

func TestDependencyManager_DetectCircularDependencies(t *testing.T) {
	t.Run("with empty dependencies", func(t *testing.T) {
		dm := NewDependencyManager(MockDatabase())
		taskID := uuid.New()

		circular, err := dm.DetectCircularDependencies(taskID, []uuid.UUID{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if circular {
			t.Error("Expected false for empty dependencies")
		}
	})

	t.Run("with dependencies", func(t *testing.T) {
		t.Skip("Requires real database connection")
	})
}

func TestDependencyManager_GetDependencyChain(t *testing.T) {
	t.Skip("Requires real database connection")
}

func TestDependencyManager_GetDependentTasks(t *testing.T) {
	t.Skip("Requires real database connection")
}

// ========================================
// Cache Tests
// ========================================

func TestTaskManager_CacheOperations(t *testing.T) {
	t.Run("cacheTask with nil Redis", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)
		task := &Task{
			ID:   uuid.New(),
			Type: TaskTypePlanning,
		}

		err := tm.cacheTask(nil, task)
		if err != nil {
			t.Errorf("Expected no error with nil Redis, got %v", err)
		}
	})

	t.Run("getCachedTask with nil Redis", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)
		taskID := uuid.New()

		cachedTask, err := tm.getCachedTask(nil, taskID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if cachedTask != nil {
			t.Error("Expected nil task with no Redis")
		}
	})

	t.Run("invalidateTaskCache with nil Redis", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)
		taskID := uuid.New()

		err := tm.invalidateTaskCache(nil, taskID)
		if err != nil {
			t.Errorf("Expected no error with nil Redis, got %v", err)
		}
	})

	t.Run("cacheTaskStats with nil Redis", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)
		stats := map[string]interface{}{
			"total": 10,
			"pending": 5,
		}

		err := tm.cacheTaskStats(nil, stats)
		if err != nil {
			t.Errorf("Expected no error with nil Redis, got %v", err)
		}
	})

	t.Run("getCachedTaskStats with nil Redis", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)

		stats, err := tm.getCachedTaskStats(nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if stats != nil {
			t.Error("Expected nil stats with no Redis")
		}
	})

	t.Run("cacheWorkerTasks with nil Redis", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)
		workerID := uuid.New()
		taskIDs := []uuid.UUID{uuid.New(), uuid.New()}

		err := tm.cacheWorkerTasks(nil, workerID, taskIDs)
		if err != nil {
			t.Errorf("Expected no error with nil Redis, got %v", err)
		}
	})

	t.Run("getCachedWorkerTasks with nil Redis", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)
		workerID := uuid.New()

		taskIDs, err := tm.getCachedWorkerTasks(nil, workerID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if taskIDs != nil {
			t.Error("Expected nil task IDs with no Redis")
		}
	})
}

func TestTaskManager_GetTaskWithCache(t *testing.T) {
	t.Run("task exists in memory", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)

		// Create a task
		task, err := tm.CreateTask(
			TaskTypePlanning,
			map[string]interface{}{"test": "data"},
			PriorityNormal,
			CriticalityNormal,
			[]uuid.UUID{},
		)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		// Get task with cache
		cachedTask, err := tm.GetTaskWithCache(nil, task.ID)
		if err != nil {
			t.Fatalf("Expected task to be found, got error: %v", err)
		}
		if cachedTask.ID != task.ID {
			t.Errorf("Expected task ID %s, got %s", task.ID, cachedTask.ID)
		}
	})

	t.Run("task does not exist", func(t *testing.T) {
		tm := NewTaskManager(MockDatabase(), nil)
		nonExistentID := uuid.New()

		_, err := tm.GetTaskWithCache(nil, nonExistentID)
		if err == nil {
			t.Error("Expected error for non-existent task")
		}
	})
}

// ========================================
// Additional Task Manager Tests
// ========================================

func TestTaskManager_AssignTask(t *testing.T) {
	tm := NewTaskManager(MockDatabase(), MockRedis())

	// Create a task
	task, err := tm.CreateTask(
		TaskTypeBuilding,
		map[string]interface{}{"description": "Build test"},
		PriorityHigh,
		CriticalityNormal,
		[]uuid.UUID{},
	)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	workerID := uuid.New()
	err = tm.AssignTask(task.ID, workerID)

	// Without a real database or worker, this should error
	if err == nil {
		t.Error("Expected error when assigning task without worker")
	}
}

func TestTaskManager_SplitTask(t *testing.T) {
	t.Skip("SplitTask requires SplitStrategy implementation - skipping for coverage")
}

// ========================================
// Task Queue Additional Tests
// ========================================

func TestTaskQueue_RemoveTask(t *testing.T) {
	tq := NewTaskQueue()

	task := &Task{
		ID:       uuid.New(),
		Priority: PriorityHigh,
	}

	tq.AddTask(task)
	removed := tq.RemoveTask(task.ID.String())

	if !removed {
		t.Error("Expected task to be removed")
	}

	// Queue should be empty
	nextTask := tq.GetNextTask()
	if nextTask != nil {
		t.Error("Expected nil after removing task")
	}

	// Try to remove non-existent task
	removed = tq.RemoveTask(uuid.New().String())
	if removed {
		t.Error("Expected false when removing non-existent task")
	}
}

func TestTaskQueue_Clear(t *testing.T) {
	tq := NewTaskQueue()

	// Add multiple tasks
	for i := 0; i < 5; i++ {
		tq.AddTask(&Task{
			ID:       uuid.New(),
			Priority: PriorityNormal,
		})
	}

	tq.Clear()

	stats := tq.GetQueueStats()
	if stats.Total != 0 {
		t.Errorf("Expected 0 total tasks after clear, got %d", stats.Total)
	}

	// Queue should return nil when empty
	nextTask := tq.GetNextTask()
	if nextTask != nil {
		t.Error("Expected nil from empty queue")
	}
}

// ========================================
// Task Type Tests
// ========================================

func TestTaskTypes(t *testing.T) {
	types := []TaskType{
		TaskTypePlanning,
		TaskTypeBuilding,
		TaskTypeTesting,
		TaskTypeRefactoring,
		TaskTypeDebugging,
		TaskTypeDesign,
		TaskTypeDiagram,
		TaskTypeDeployment,
		TaskTypePorting,
	}

	for _, taskType := range types {
		if string(taskType) == "" {
			t.Errorf("Task type should not be empty: %v", taskType)
		}
	}
}

func TestTaskStatuses(t *testing.T) {
	statuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusAssigned,
		TaskStatusRunning,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusPaused,
		TaskStatusWaitingForWorker,
		TaskStatusWaitingForDeps,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("Task status should not be empty: %v", status)
		}
	}
}

func TestTaskPriorities(t *testing.T) {
	if PriorityLow >= PriorityNormal {
		t.Error("PriorityLow should be less than PriorityNormal")
	}
	if PriorityNormal >= PriorityHigh {
		t.Error("PriorityNormal should be less than PriorityHigh")
	}
	if PriorityHigh >= PriorityCritical {
		t.Error("PriorityHigh should be less than PriorityCritical")
	}
}

func TestTaskCriticalities(t *testing.T) {
	criticalities := []TaskCriticality{
		CriticalityLow,
		CriticalityNormal,
		CriticalityHigh,
		CriticalityCritical,
	}

	for _, crit := range criticalities {
		if string(crit) == "" {
			t.Errorf("Task criticality should not be empty: %v", crit)
		}
	}
}

func TestComplexityLevels(t *testing.T) {
	levels := []ComplexityLevel{
		ComplexityLow,
		ComplexityMedium,
		ComplexityHigh,
	}

	for _, level := range levels {
		if string(level) == "" {
			t.Errorf("Complexity level should not be empty: %v", level)
		}
	}
}

// ========================================
// DatabaseManager Tests
// ========================================

func TestNewDatabaseManager(t *testing.T) {
	dm := NewDatabaseManager(MockDatabase())
	if dm == nil {
		t.Fatal("Expected database manager, got nil")
	}
}

func TestDatabaseManager_CreateTask(t *testing.T) {
	t.Skip("Requires real database connection")
}

func TestDatabaseManager_GetTask(t *testing.T) {
	t.Skip("Requires real database connection")
}

func TestDatabaseManager_ListTasks(t *testing.T) {
	t.Skip("Requires real database connection")
}

func TestDatabaseManager_StartTask(t *testing.T) {
	t.Skip("Requires real database connection")
}

func TestDatabaseManager_CompleteTask(t *testing.T) {
	t.Skip("Requires real database connection")
}

func TestDatabaseManager_FailTask(t *testing.T) {
	t.Skip("Requires real database connection")
}

func TestDatabaseManager_DeleteTask(t *testing.T) {
	t.Skip("Requires real database connection")
}

// ========================================
// Helper Function Tests
// ========================================

func TestTaskManager_CreateCheckpoint(t *testing.T) {
	// Skip due to database requirement
	t.Skip("CreateCheckpoint requires real database connection")
}

// Test queue sorting behavior
func TestTaskQueue_PrioritySorting(t *testing.T) {
	tq := NewTaskQueue()

	// Add tasks with different priorities and criticalities
	criticalTask := &Task{
		ID:          uuid.New(),
		Priority:    PriorityCritical,
		Criticality: CriticalityCritical,
	}
	highTask := &Task{
		ID:          uuid.New(),
		Priority:    PriorityHigh,
		Criticality: CriticalityHigh,
	}
	normalTask := &Task{
		ID:          uuid.New(),
		Priority:    PriorityNormal,
		Criticality: CriticalityNormal,
	}

	// Add in random order
	tq.AddTask(normalTask)
	tq.AddTask(highTask)
	tq.AddTask(criticalTask)

	// Should get critical first
	next := tq.GetNextTask()
	if next.ID != criticalTask.ID {
		t.Errorf("Expected critical task first, got priority %d", next.Priority)
	}

	// Then high
	next = tq.GetNextTask()
	if next.ID != highTask.ID {
		t.Errorf("Expected high task second, got priority %d", next.Priority)
	}

	// Then normal
	next = tq.GetNextTask()
	if next.ID != normalTask.ID {
		t.Errorf("Expected normal task third, got priority %d", next.Priority)
	}
}

// Test queue with multiple high priority tasks
func TestTaskQueue_MultipleSamePriority(t *testing.T) {
	tq := NewTaskQueue()

	// Add multiple tasks with same priority but different criticality
	task1 := &Task{
		ID:          uuid.New(),
		Priority:    PriorityHigh,
		Criticality: CriticalityHigh,
	}
	task2 := &Task{
		ID:          uuid.New(),
		Priority:    PriorityHigh,
		Criticality: CriticalityCritical,
	}
	task3 := &Task{
		ID:          uuid.New(),
		Priority:    PriorityHigh,
		Criticality: CriticalityNormal,
	}

	tq.AddTask(task1)
	tq.AddTask(task2)
	tq.AddTask(task3)

	// Should get critical criticality first
	next := tq.GetNextTask()
	if next.Criticality != CriticalityCritical {
		t.Errorf("Expected critical criticality first, got %s", next.Criticality)
	}

	// Then high criticality
	next = tq.GetNextTask()
	if next.Criticality != CriticalityHigh {
		t.Errorf("Expected high criticality second, got %s", next.Criticality)
	}

	// Then normal criticality
	next = tq.GetNextTask()
	if next.Criticality != CriticalityNormal {
		t.Errorf("Expected normal criticality third, got %s", next.Criticality)
	}
}

// ========================================
// Test helper functions that don't need DB
// ========================================

func TestGetStringFromPtr(t *testing.T) {
	// Test with nil pointer
	nilStr := getStringFromPtr(nil)
	if nilStr != "" {
		t.Errorf("Expected empty string for nil pointer, got %s", nilStr)
	}

	// Test with valid pointer
	testStr := "test string"
	result := getStringFromPtr(&testStr)
	if result != testStr {
		t.Errorf("Expected %s, got %s", testStr, result)
	}
}

// Test contains helper function
func TestContains(t *testing.T) {
	slice := []string{"one", "two", "three"}

	if !contains(slice, "two") {
		t.Error("Expected 'two' to be in slice")
	}

	if contains(slice, "four") {
		t.Error("Expected 'four' to not be in slice")
	}

	// Test with empty slice
	if contains([]string{}, "test") {
		t.Error("Expected false for empty slice")
	}
}
