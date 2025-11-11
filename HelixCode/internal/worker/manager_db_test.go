package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// ========================================
// DatabaseManager Constructor Tests
// ========================================

func TestNewDatabaseManager(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	assert.NotNil(t, dm)
	assert.Equal(t, mockDB, dm.db)
}

// ========================================
// GetWorker Tests
// ========================================

func TestDatabaseManager_GetWorkerSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID := uuid.New()
	now := time.Now()
	lastHeartbeat := now
	cpuUsage := 45.5
	memoryUsage := 60.2
	diskUsage := 30.1

	sshConfig := map[string]interface{}{
		"host": "worker1.example.com",
		"port": float64(22),
	}

	resources := map[string]interface{}{
		"cpu_count":    float64(8),
		"total_memory": float64(16000000000),
		"total_disk":   float64(500000000000),
		"gpu_count":    float64(1),
		"gpu_model":    "NVIDIA RTX 3080",
		"gpu_memory":   float64(10000000000),
	}

	mockRow := database.NewMockRowWithValues(
		workerID, "worker1", "Worker One", sshConfig,
		[]string{"python", "go", "docker"}, resources,
		"active", "healthy", &lastHeartbeat, &cpuUsage, &memoryUsage, &diskUsage,
		2, 10, now, now,
	)

	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	worker, err := dm.GetWorker(ctx, workerID.String())

	assert.NoError(t, err)
	assert.NotNil(t, worker)
	assert.Equal(t, workerID, worker.ID)
	assert.Equal(t, "worker1", worker.Hostname)
	assert.Equal(t, "Worker One", worker.DisplayName)
	assert.Equal(t, WorkerStatus("active"), worker.Status)
	assert.Equal(t, WorkerHealth("healthy"), worker.HealthStatus)
	assert.Equal(t, 45.5, worker.CPUUsagePercent)
	assert.Equal(t, 60.2, worker.MemoryUsagePercent)
	assert.Equal(t, 30.1, worker.DiskUsagePercent)
	assert.Equal(t, 2, worker.CurrentTasksCount)
	assert.Equal(t, 10, worker.MaxConcurrentTasks)
	assert.Equal(t, 8, worker.Resources.CPUCount)
	assert.Equal(t, int64(16000000000), worker.Resources.TotalMemory)
	assert.Equal(t, 1, worker.Resources.GPUCount)
	assert.Equal(t, "NVIDIA RTX 3080", worker.Resources.GPUModel)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_GetWorkerNullableFields(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID := uuid.New()
	now := time.Now()

	sshConfig := map[string]interface{}{}
	resources := map[string]interface{}{}

	// Test with nil nullable fields
	mockRow := database.NewMockRowWithValues(
		workerID, "worker2", "Worker Two", sshConfig,
		[]string{}, resources,
		"idle", "healthy", nil, nil, nil, nil,
		0, 5, now, now,
	)

	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	worker, err := dm.GetWorker(ctx, workerID.String())

	assert.NoError(t, err)
	assert.NotNil(t, worker)
	assert.True(t, worker.LastHeartbeat.IsZero())
	assert.Equal(t, 0.0, worker.CPUUsagePercent)
	assert.Equal(t, 0.0, worker.MemoryUsagePercent)
	assert.Equal(t, 0.0, worker.DiskUsagePercent)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_GetWorkerInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	worker, err := dm.GetWorker(ctx, "invalid-uuid")

	assert.Error(t, err)
	assert.Nil(t, worker)
	assert.Contains(t, err.Error(), "invalid worker ID")
}

func TestDatabaseManager_GetWorkerNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID := uuid.New()

	mockRow := database.NewMockRowWithError(fmt.Errorf("no rows in result set"))
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	worker, err := dm.GetWorker(ctx, workerID.String())

	assert.Error(t, err)
	assert.Nil(t, worker)
	assert.Contains(t, err.Error(), "failed to get worker from database")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_GetWorkerDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID := uuid.New()

	mockRow := database.NewMockRowWithError(fmt.Errorf("database connection failed"))
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	worker, err := dm.GetWorker(ctx, workerID.String())

	assert.Error(t, err)
	assert.Nil(t, worker)
	assert.Contains(t, err.Error(), "failed to get worker from database")
	mockDB.AssertExpectations(t)
}

// ========================================
// ListWorkers Tests
// ========================================

func TestDatabaseManager_ListWorkersSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID1 := uuid.New()
	workerID2 := uuid.New()
	now := time.Now()
	heartbeat1 := now
	cpu1 := 50.0
	mem1 := 70.0
	disk1 := 40.0

	sshConfig1 := map[string]interface{}{"host": "worker1"}
	sshConfig2 := map[string]interface{}{"host": "worker2"}
	resources1 := map[string]interface{}{"cpu_count": float64(4)}
	resources2 := map[string]interface{}{"cpu_count": float64(8)}

	mockRows := database.NewMockRows([][]interface{}{
		{workerID1, "worker1", "Worker 1", sshConfig1, []string{"python"}, resources1,
			"active", "healthy", &heartbeat1, &cpu1, &mem1, &disk1, 1, 5, now, now},
		{workerID2, "worker2", "Worker 2", sshConfig2, []string{"go"}, resources2,
			"idle", "healthy", nil, nil, nil, nil, 0, 10, now, now},
	})

	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	workers, err := dm.ListWorkers(ctx)

	assert.NoError(t, err)
	assert.Len(t, workers, 2)
	assert.Equal(t, "worker1", workers[0].Hostname)
	assert.Equal(t, "worker2", workers[1].Hostname)
	assert.Equal(t, 50.0, workers[0].CPUUsagePercent)
	assert.Equal(t, 0.0, workers[1].CPUUsagePercent) // nil converted to 0
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_ListWorkersEmpty(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	mockRows := database.NewMockRows([][]interface{}{})
	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	workers, err := dm.ListWorkers(ctx)

	assert.NoError(t, err)
	assert.Empty(t, workers)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_ListWorkersQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(nil, fmt.Errorf("query failed"))

	workers, err := dm.ListWorkers(ctx)

	assert.Error(t, err)
	assert.Nil(t, workers)
	assert.Contains(t, err.Error(), "failed to query workers")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_ListWorkersIterationError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	mockRows := database.NewMockRowsWithError(fmt.Errorf("iteration failed"))
	mockDB.On("Query", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	workers, err := dm.ListWorkers(ctx)

	assert.Error(t, err)
	assert.Nil(t, workers)
	assert.Contains(t, err.Error(), "error iterating worker rows")
	mockDB.AssertExpectations(t)
}

// ========================================
// RegisterWorker Tests
// ========================================

func TestDatabaseManager_RegisterWorkerSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	now := time.Now()

	sshConfig := map[string]interface{}{
		"host": "worker3.example.com",
		"port": 22,
		"user": "deploy",
	}

	capabilities := []string{"python", "docker", "kubernetes"}

	resources := map[string]interface{}{
		"cpu_count":    8,
		"total_memory": int64(32000000000),
		"total_disk":   int64(1000000000000),
		"gpu_count":    2,
		"gpu_model":    "NVIDIA A100",
		"gpu_memory":   int64(40000000000),
	}

	mockRow := database.NewMockRowWithValues(now, now)
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	worker, err := dm.RegisterWorker(ctx, "worker3", "Worker Three", sshConfig, capabilities, resources)

	assert.NoError(t, err)
	assert.NotNil(t, worker)
	assert.Equal(t, "worker3", worker.Hostname)
	assert.Equal(t, "Worker Three", worker.DisplayName)
	assert.Equal(t, WorkerStatus("active"), worker.Status)
	assert.Equal(t, WorkerHealth("healthy"), worker.HealthStatus)
	assert.Equal(t, 0, worker.CurrentTasksCount)
	assert.Equal(t, 10, worker.MaxConcurrentTasks)
	assert.NotEqual(t, uuid.Nil, worker.ID)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_RegisterWorkerDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()

	sshConfig := map[string]interface{}{"host": "worker4"}
	capabilities := []string{"python"}
	resources := map[string]interface{}{}

	mockRow := database.NewMockRowWithError(fmt.Errorf("duplicate key violation"))
	mockDB.On("QueryRow", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	worker, err := dm.RegisterWorker(ctx, "worker4", "Worker Four", sshConfig, capabilities, resources)

	assert.Error(t, err)
	assert.Nil(t, worker)
	assert.Contains(t, err.Error(), "failed to register worker in database")
	mockDB.AssertExpectations(t)
}

// ========================================
// UpdateWorkerHeartbeat Tests
// ========================================

func TestDatabaseManager_UpdateWorkerHeartbeatSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID := uuid.New()

	metrics := map[string]interface{}{
		"cpu_usage_percent":    55.5,
		"memory_usage_percent": 70.2,
		"disk_usage_percent":   45.8,
		"network_rx_bytes":     int64(1000000),
		"network_tx_bytes":     int64(500000),
		"current_tasks_count":  3,
		"temperature_celsius":  68.5,
	}

	// Mock heartbeat update
	mockDB.MockExecSuccess(1)

	// Mock metrics insert
	mockDB.MockExecSuccess(1)

	err := dm.UpdateWorkerHeartbeat(ctx, workerID.String(), metrics)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_UpdateWorkerHeartbeatInvalidID(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	metrics := map[string]interface{}{}

	err := dm.UpdateWorkerHeartbeat(ctx, "invalid-uuid", metrics)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid worker ID")
}

func TestDatabaseManager_UpdateWorkerHeartbeatUpdateError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID := uuid.New()
	metrics := map[string]interface{}{}

	mockDB.MockExecError(fmt.Errorf("database connection lost"))

	err := dm.UpdateWorkerHeartbeat(ctx, workerID.String(), metrics)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update worker heartbeat")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_UpdateWorkerHeartbeatMetricsError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID := uuid.New()
	metrics := map[string]interface{}{}

	// First Exec succeeds (heartbeat update)
	tag1 := pgconn.NewCommandTag("UPDATE 1")
	mockDB.On("Exec", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(tag1, nil).Once()

	// Second Exec fails (metrics insert)
	tag2 := pgconn.NewCommandTag("INSERT 0 0")
	mockDB.On("Exec", ctx, mockDB.AnyString(), mockDB.AnyArgs()).Return(tag2, fmt.Errorf("metrics table constraint violation")).Once()

	err := dm.UpdateWorkerHeartbeat(ctx, workerID.String(), metrics)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store worker metrics")
	mockDB.AssertExpectations(t)
}

func TestDatabaseManager_UpdateWorkerHeartbeatEmptyMetrics(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDatabaseManager(mockDB)

	ctx := context.Background()
	workerID := uuid.New()

	// Empty metrics map - all values should be zero/default
	metrics := map[string]interface{}{}

	mockDB.MockExecSuccess(1) // heartbeat update
	mockDB.MockExecSuccess(1) // metrics insert

	err := dm.UpdateWorkerHeartbeat(ctx, workerID.String(), metrics)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

// ========================================
// Helper Function Tests
// ========================================

func TestGetStringDB(t *testing.T) {
	tests := []struct {
		name         string
		data         map[string]interface{}
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "existing string",
			data:         map[string]interface{}{"model": "NVIDIA RTX 3080"},
			key:          "model",
			defaultValue: "unknown",
			expected:     "NVIDIA RTX 3080",
		},
		{
			name:         "missing key",
			data:         map[string]interface{}{},
			key:          "model",
			defaultValue: "unknown",
			expected:     "unknown",
		},
		{
			name:         "wrong type",
			data:         map[string]interface{}{"model": 123},
			key:          "model",
			defaultValue: "unknown",
			expected:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringDB(tt.data, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntDB(t *testing.T) {
	tests := []struct {
		name         string
		data         map[string]interface{}
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "existing int as float64",
			data:         map[string]interface{}{"count": float64(8)},
			key:          "count",
			defaultValue: 0,
			expected:     8,
		},
		{
			name:         "missing key",
			data:         map[string]interface{}{},
			key:          "count",
			defaultValue: 0,
			expected:     0,
		},
		{
			name:         "wrong type",
			data:         map[string]interface{}{"count": "eight"},
			key:          "count",
			defaultValue: 0,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntDB(tt.data, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInt64DB(t *testing.T) {
	tests := []struct {
		name         string
		data         map[string]interface{}
		key          string
		defaultValue int64
		expected     int64
	}{
		{
			name:         "existing int64 as float64",
			data:         map[string]interface{}{"memory": float64(16000000000)},
			key:          "memory",
			defaultValue: 0,
			expected:     16000000000,
		},
		{
			name:         "missing key",
			data:         map[string]interface{}{},
			key:          "memory",
			defaultValue: 0,
			expected:     0,
		},
		{
			name:         "wrong type",
			data:         map[string]interface{}{"memory": "16GB"},
			key:          "memory",
			defaultValue: 0,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInt64DB(tt.data, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseResources(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected Resources
	}{
		{
			name: "complete resources",
			input: map[string]interface{}{
				"cpu_count":    float64(16),
				"total_memory": float64(64000000000),
				"total_disk":   float64(2000000000000),
				"gpu_count":    float64(4),
				"gpu_model":    "NVIDIA A100",
				"gpu_memory":   float64(80000000000),
			},
			expected: Resources{
				CPUCount:    16,
				TotalMemory: 64000000000,
				TotalDisk:   2000000000000,
				GPUCount:    4,
				GPUModel:    "NVIDIA A100",
				GPUMemory:   80000000000,
			},
		},
		{
			name:  "empty resources",
			input: map[string]interface{}{},
			expected: Resources{
				CPUCount:    0,
				TotalMemory: 0,
				TotalDisk:   0,
				GPUCount:    0,
				GPUModel:    "",
				GPUMemory:   0,
			},
		},
		{
			name: "partial resources",
			input: map[string]interface{}{
				"cpu_count":    float64(8),
				"total_memory": float64(32000000000),
			},
			expected: Resources{
				CPUCount:    8,
				TotalMemory: 32000000000,
				TotalDisk:   0,
				GPUCount:    0,
				GPUModel:    "",
				GPUMemory:   0,
			},
		},
		{
			name: "wrong types",
			input: map[string]interface{}{
				"cpu_count":  "8",
				"gpu_model":  123,
				"gpu_memory": "10GB",
			},
			expected: Resources{
				CPUCount:    0,
				TotalMemory: 0,
				TotalDisk:   0,
				GPUCount:    0,
				GPUModel:    "",
				GPUMemory:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseResources(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
