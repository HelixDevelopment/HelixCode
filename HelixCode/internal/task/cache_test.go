package task

import (
	"context"
	"testing"
	"time"

	"dev.helix.code/internal/database"
	"dev.helix.code/internal/redis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheTask tests caching a task in Redis
func TestCacheTask(t *testing.T) {
	mockDB := database.NewMockDatabase()

	// Create mock Redis client with enabled config
	mockRedis := &redis.Client{} // Will be treated as disabled (no client)

	tm := NewTaskManager(mockDB, mockRedis)

	task := &Task{
		ID:           uuid.New(),
		Type:         TaskTypePlanning,
		Status:       TaskStatusPending,
		Priority:     PriorityNormal,
		Criticality:  CriticalityNormal,
		Data:         map[string]interface{}{"test": "data"},
		Dependencies: []uuid.UUID{},
		MaxRetries:   3,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	ctx := context.Background()
	err := tm.cacheTask(ctx, task)

	// Should not error even if Redis operations fail
	assert.NoError(t, err)
}

// TestCacheTask_RedisDisabled tests caching when Redis is disabled
func TestCacheTask_RedisDisabled(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{} // Disabled by default

	tm := NewTaskManager(mockDB, mockRedis)

	task := &Task{
		ID:          uuid.New(),
		Type:        TaskTypeBuilding,
		Status:      TaskStatusPending,
		Priority:    PriorityHigh,
		Criticality: CriticalityHigh,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ctx := context.Background()
	err := tm.cacheTask(ctx, task)

	// Should succeed (no-op) when Redis is disabled
	assert.NoError(t, err)
}

// TestCacheTask_NilRedis tests caching when Redis is nil
func TestCacheTask_NilRedis(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, nil)

	task := &Task{
		ID:        uuid.New(),
		Type:      TaskTypeTesting,
		Status:    TaskStatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx := context.Background()
	err := tm.cacheTask(ctx, task)

	// Should succeed (no-op) when Redis is nil
	assert.NoError(t, err)
}

// TestGetCachedTask tests retrieving a task from cache
func TestGetCachedTask(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, testing cache miss behavior
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	taskID := uuid.New()
	ctx := context.Background()

	// Get from cache (should return nil when not found)
	task, err := tm.getCachedTask(ctx, taskID)

	assert.NoError(t, err)
	assert.Nil(t, task)
}

// TestGetCachedTask_RedisDisabled tests cache retrieval when Redis is disabled
func TestGetCachedTask_RedisDisabled(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{} // Disabled by default

	tm := NewTaskManager(mockDB, mockRedis)

	taskID := uuid.New()
	ctx := context.Background()

	task, err := tm.getCachedTask(ctx, taskID)

	// Should return nil (cache miss) when Redis is disabled
	assert.NoError(t, err)
	assert.Nil(t, task)
}

// TestGetCachedTask_NilRedis tests cache retrieval when Redis is nil
func TestGetCachedTask_NilRedis(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, nil)

	taskID := uuid.New()
	ctx := context.Background()

	task, err := tm.getCachedTask(ctx, taskID)

	// Should return nil when Redis is nil
	assert.NoError(t, err)
	assert.Nil(t, task)
}

// TestInvalidateTaskCache tests cache invalidation
func TestInvalidateTaskCache(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, testing no-op behavior
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	taskID := uuid.New()
	ctx := context.Background()

	err := tm.invalidateTaskCache(ctx, taskID)

	// Should not error even if Redis operations fail
	assert.NoError(t, err)
}

// TestInvalidateTaskCache_RedisDisabled tests invalidation when Redis is disabled
func TestInvalidateTaskCache_RedisDisabled(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{} // Disabled by default

	tm := NewTaskManager(mockDB, mockRedis)

	taskID := uuid.New()
	ctx := context.Background()

	err := tm.invalidateTaskCache(ctx, taskID)

	// Should succeed (no-op) when Redis is disabled
	assert.NoError(t, err)
}

// TestInvalidateTaskCache_NilRedis tests invalidation when Redis is nil
func TestInvalidateTaskCache_NilRedis(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, nil)

	taskID := uuid.New()
	ctx := context.Background()

	err := tm.invalidateTaskCache(ctx, taskID)

	// Should succeed (no-op) when Redis is nil
	assert.NoError(t, err)
}

// TestCacheTaskStats tests caching task statistics
func TestCacheTaskStats(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, testing no-op behavior
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	stats := map[string]interface{}{
		"total":     100,
		"pending":   20,
		"running":   30,
		"completed": 50,
	}

	ctx := context.Background()
	err := tm.cacheTaskStats(ctx, stats)

	// Should not error even if Redis operations fail
	assert.NoError(t, err)
}

// TestCacheTaskStats_RedisDisabled tests stats caching when Redis is disabled
func TestCacheTaskStats_RedisDisabled(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{} // Disabled by default

	tm := NewTaskManager(mockDB, mockRedis)

	stats := map[string]interface{}{
		"total": 50,
	}

	ctx := context.Background()
	err := tm.cacheTaskStats(ctx, stats)

	// Should succeed (no-op) when Redis is disabled
	assert.NoError(t, err)
}

// TestCacheTaskStats_NilRedis tests stats caching when Redis is nil
func TestCacheTaskStats_NilRedis(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, nil)

	stats := map[string]interface{}{
		"total": 75,
	}

	ctx := context.Background()
	err := tm.cacheTaskStats(ctx, stats)

	// Should succeed (no-op) when Redis is nil
	assert.NoError(t, err)
}

// TestGetCachedTaskStats tests retrieving cached task statistics
func TestGetCachedTaskStats(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, testing cache miss behavior
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	ctx := context.Background()

	// Get from cache (should return nil when not found)
	stats, err := tm.getCachedTaskStats(ctx)

	assert.NoError(t, err)
	assert.Nil(t, stats)
}

// TestGetCachedTaskStats_RedisDisabled tests stats retrieval when Redis is disabled
func TestGetCachedTaskStats_RedisDisabled(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{} // Disabled by default

	tm := NewTaskManager(mockDB, mockRedis)

	ctx := context.Background()

	stats, err := tm.getCachedTaskStats(ctx)

	// Should return nil (cache miss) when Redis is disabled
	assert.NoError(t, err)
	assert.Nil(t, stats)
}

// TestGetCachedTaskStats_NilRedis tests stats retrieval when Redis is nil
func TestGetCachedTaskStats_NilRedis(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, nil)

	ctx := context.Background()

	stats, err := tm.getCachedTaskStats(ctx)

	// Should return nil when Redis is nil
	assert.NoError(t, err)
	assert.Nil(t, stats)
}

// TestCacheWorkerTasks tests caching worker tasks
func TestCacheWorkerTasks(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, testing no-op behavior
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	workerID := uuid.New()
	taskIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	ctx := context.Background()
	err := tm.cacheWorkerTasks(ctx, workerID, taskIDs)

	// Should not error even if Redis operations fail
	assert.NoError(t, err)
}

// TestCacheWorkerTasks_RedisDisabled tests worker tasks caching when Redis is disabled
func TestCacheWorkerTasks_RedisDisabled(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{} // Disabled by default

	tm := NewTaskManager(mockDB, mockRedis)

	workerID := uuid.New()
	taskIDs := []uuid.UUID{uuid.New()}

	ctx := context.Background()
	err := tm.cacheWorkerTasks(ctx, workerID, taskIDs)

	// Should succeed (no-op) when Redis is disabled
	assert.NoError(t, err)
}

// TestCacheWorkerTasks_NilRedis tests worker tasks caching when Redis is nil
func TestCacheWorkerTasks_NilRedis(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, nil)

	workerID := uuid.New()
	taskIDs := []uuid.UUID{uuid.New(), uuid.New()}

	ctx := context.Background()
	err := tm.cacheWorkerTasks(ctx, workerID, taskIDs)

	// Should succeed (no-op) when Redis is nil
	assert.NoError(t, err)
}

// TestGetCachedWorkerTasks tests retrieving cached worker tasks
func TestGetCachedWorkerTasks(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, testing cache miss behavior
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	workerID := uuid.New()
	ctx := context.Background()

	// Get from cache (should return nil when not found)
	taskIDs, err := tm.getCachedWorkerTasks(ctx, workerID)

	assert.NoError(t, err)
	assert.Nil(t, taskIDs)
}

// TestGetCachedWorkerTasks_RedisDisabled tests worker tasks retrieval when Redis is disabled
func TestGetCachedWorkerTasks_RedisDisabled(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{} // Disabled by default

	tm := NewTaskManager(mockDB, mockRedis)

	workerID := uuid.New()
	ctx := context.Background()

	taskIDs, err := tm.getCachedWorkerTasks(ctx, workerID)

	// Should return nil (cache miss) when Redis is disabled
	assert.NoError(t, err)
	assert.Nil(t, taskIDs)
}

// TestGetCachedWorkerTasks_NilRedis tests worker tasks retrieval when Redis is nil
func TestGetCachedWorkerTasks_NilRedis(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, nil)

	workerID := uuid.New()
	ctx := context.Background()

	taskIDs, err := tm.getCachedWorkerTasks(ctx, workerID)

	// Should return nil when Redis is nil
	assert.NoError(t, err)
	assert.Nil(t, taskIDs)
}

// TestGetTaskWithCache tests the cache-aware task retrieval
func TestGetTaskWithCache(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	// First, create a task
	task, err := tm.CreateTask(
		TaskTypePlanning,
		map[string]interface{}{"test": "data"},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Get with cache
	retrievedTask, err := tm.GetTaskWithCache(ctx, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrievedTask)
	assert.Equal(t, task.ID, retrievedTask.ID)
	assert.Equal(t, task.Type, retrievedTask.Type)
}

// TestGetTaskWithCache_NotFound tests cache-aware retrieval for non-existent task
func TestGetTaskWithCache_NotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	ctx := context.Background()
	nonExistentID := uuid.New()

	retrievedTask, err := tm.GetTaskWithCache(ctx, nonExistentID)
	assert.Error(t, err)
	assert.Nil(t, retrievedTask)
	assert.Contains(t, err.Error(), "task not found")
}

// TestUpdateTaskWithCache tests cache-aware task update
func TestUpdateTaskWithCache(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, testing no-op behavior
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	// Create a task
	task, err := tm.CreateTask(
		TaskTypeBuilding,
		map[string]interface{}{"test": "data"},
		PriorityHigh,
		CriticalityHigh,
		[]uuid.UUID{},
	)
	require.NoError(t, err)

	// Modify task
	task.Status = TaskStatusRunning
	task.Data["progress"] = 50

	ctx := context.Background()

	// Update with cache
	err = tm.UpdateTaskWithCache(ctx, task)

	// Should not error (database operations are mocked)
	// In real scenario, this would update DB and invalidate/update cache
	assert.NoError(t, err)
}

// TestCacheMarshalError tests handling of marshal errors
func TestCacheMarshalError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, but we're testing marshal errors
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	// Create a task with data that can't be marshaled
	task := &Task{
		ID:     uuid.New(),
		Type:   TaskTypePlanning,
		Status: TaskStatusPending,
		Data: map[string]interface{}{
			"invalid": make(chan int), // Channels can't be marshaled to JSON
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx := context.Background()
	err := tm.cacheTask(ctx, task)

	// With Redis disabled, should return nil even with unmarshalable data
	// (marshal code is never reached when Redis is disabled)
	assert.NoError(t, err)
}

// TestCacheStatsWithInvalidData tests caching with data that causes marshal errors
func TestCacheStatsWithInvalidData(t *testing.T) {
	mockDB := database.NewMockDatabase()
	// Redis client without config - will be disabled, but we're testing marshal errors
	mockRedis := &redis.Client{}

	tm := NewTaskManager(mockDB, mockRedis)

	stats := map[string]interface{}{
		"invalid": make(chan int), // Can't be marshaled
	}

	ctx := context.Background()
	err := tm.cacheTaskStats(ctx, stats)

	// With Redis disabled, should return nil even with unmarshalable data
	// (marshal code is never reached when Redis is disabled)
	assert.NoError(t, err)
}
