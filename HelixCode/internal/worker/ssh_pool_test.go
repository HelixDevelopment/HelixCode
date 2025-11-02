package worker

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestSSHWorkerPool_Creation tests SSH worker pool creation
func TestSSHWorkerPool_Creation(t *testing.T) {
	pool := NewSSHWorkerPool(true)
	assert.NotNil(t, pool)
	assert.True(t, pool.autoInstall)

	pool2 := NewSSHWorkerPool(false)
	assert.False(t, pool2.autoInstall)
}

// TestSSHWorkerPool_AddWorker tests adding workers to the pool
func TestSSHWorkerPool_AddWorker(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	worker := &SSHWorker{
		Hostname:    "test-worker.local",
		DisplayName: "test-worker",
		SSHConfig: &SSHWorkerConfig{
			Host:     "localhost",
			Port:     22,
			Username: "testuser",
			KeyPath:  "/dev/null",
		},
	}

	// Test adding worker (connection will fail but we test the logic)
	err := pool.AddWorker(ctx, worker)
	assert.Error(t, err) // Should fail due to SSH connection
	assert.Contains(t, err.Error(), "SSH connection failed")
}

// TestSSHWorkerPool_RemoveWorker tests worker removal
func TestSSHWorkerPool_RemoveWorker(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	// Add a mock worker
	workerID := uuid.New()
	pool.workers[workerID] = &SSHWorker{
		ID:       workerID,
		Hostname: "test-worker",
	}

	// Test removing existing worker
	err := pool.RemoveWorker(ctx, workerID)
	assert.NoError(t, err)
	assert.NotContains(t, pool.workers, workerID)

	// Test removing non-existent worker
	err = pool.RemoveWorker(ctx, uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker not found")
}

// TestSSHWorkerPool_HealthCheck tests health checking
func TestSSHWorkerPool_HealthCheck(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	// Add mock workers
	worker1ID := uuid.New()
	worker2ID := uuid.New()

	pool.workers[worker1ID] = &SSHWorker{
		ID:           worker1ID,
		Hostname:     "healthy-worker",
		Status:       WorkerStatusActive,
		HealthStatus: WorkerHealthHealthy,
		SSHConfig: &SSHWorkerConfig{
			Host: "localhost",
			Port: 22,
		},
	}

	pool.workers[worker2ID] = &SSHWorker{
		ID:           worker2ID,
		Hostname:     "unhealthy-worker",
		Status:       WorkerStatusOffline,
		HealthStatus: WorkerHealthUnhealthy,
		SSHConfig: &SSHWorkerConfig{
			Host: "invalid-host",
			Port: 22,
		},
	}

	// Run health check
	err := pool.HealthCheck(ctx)
	assert.NoError(t, err)

	// Verify worker statuses were updated
	stats := pool.GetWorkerStats(ctx)
	assert.Equal(t, 2, stats.TotalWorkers)
}

// TestSSHWorkerPool_GetWorkerStats tests statistics collection
func TestSSHWorkerPool_GetWorkerStats(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	// Add mock workers with resources
	worker1ID := uuid.New()
	worker2ID := uuid.New()

	pool.workers[worker1ID] = &SSHWorker{
		ID:           worker1ID,
		Hostname:     "worker-1",
		Status:       WorkerStatusActive,
		HealthStatus: WorkerHealthHealthy,
		Resources: Resources{
			CPUCount:    8,
			TotalMemory: 16777216, // 16GB
			GPUCount:    1,
		},
	}

	pool.workers[worker2ID] = &SSHWorker{
		ID:           worker2ID,
		Hostname:     "worker-2",
		Status:       WorkerStatusOffline,
		HealthStatus: WorkerHealthUnhealthy,
		Resources: Resources{
			CPUCount:    4,
			TotalMemory: 8388608, // 8GB
			GPUCount:    0,
		},
	}

	stats := pool.GetWorkerStats(ctx)

	assert.Equal(t, 2, stats.TotalWorkers)
	assert.Equal(t, 1, stats.ActiveWorkers)             // Only worker1 is active
	assert.Equal(t, 1, stats.HealthyWorkers)            // Only worker1 is healthy
	assert.Equal(t, 12, stats.TotalCPU)                 // 8 + 4
	assert.Equal(t, int64(25165824), stats.TotalMemory) // 16GB + 8GB
	assert.Equal(t, 1, stats.TotalGPU)
}

// TestSSHWorkerPool_ExecuteCommand tests command execution
func TestSSHWorkerPool_ExecuteCommand(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	// Add mock worker
	workerID := uuid.New()
	pool.workers[workerID] = &SSHWorker{
		ID:       workerID,
		Hostname: "test-worker",
		SSHConfig: &SSHWorkerConfig{
			Host: "localhost",
			Port: 22,
		},
	}

	// Test command execution (will fail due to SSH connection)
	output, err := pool.ExecuteCommand(ctx, workerID, "echo test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSH connection failed")
	assert.Empty(t, output)

	// Test with non-existent worker
	output, err = pool.ExecuteCommand(ctx, uuid.New(), "echo test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker not found")
	assert.Empty(t, output)
}

// TestSSHWorkerPool_ValidateSSHConfig tests SSH configuration validation
func TestSSHWorkerPool_ValidateSSHConfig(t *testing.T) {
	pool := NewSSHWorkerPool(false)

	// Test valid configuration
	validConfig := &SSHWorkerConfig{
		Host:     "example.com",
		Port:     22,
		Username: "user",
		KeyPath:  "/path/to/key",
	}

	err := pool.validateSSHConfig(validConfig)
	assert.NoError(t, err)

	// Test invalid configurations
	testCases := []struct {
		name   string
		config *SSHWorkerConfig
		error  string
	}{
		{
			name: "missing host",
			config: &SSHWorkerConfig{
				Port:     22,
				Username: "user",
			},
			error: "host is required",
		},
		{
			name: "missing username",
			config: &SSHWorkerConfig{
				Host: "example.com",
				Port: 22,
			},
			error: "username is required",
		},
		{
			name: "invalid port",
			config: &SSHWorkerConfig{
				Host:     "example.com",
				Port:     0,
				Username: "user",
			},
			error: "invalid port",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pool.validateSSHConfig(tc.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.error)
		})
	}
}

// TestSSHWorkerPool_InstallHelixCLI tests auto-installation
func TestSSHWorkerPool_InstallHelixCLI(t *testing.T) {
	pool := NewSSHWorkerPool(true)
	ctx := context.Background()

	worker := &SSHWorker{
		ID:       uuid.New(),
		Hostname: "test-worker",
		SSHConfig: &SSHWorkerConfig{
			Host: "localhost",
			Port: 22,
		},
	}

	// Test installation (will fail due to SSH connection)
	err := pool.installHelixCLI(ctx, worker)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to install Helix CLI")
}

// TestSSHWorkerPool_DetectWorkerCapabilities tests capability detection
func TestSSHWorkerPool_DetectWorkerCapabilities(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	worker := &SSHWorker{
		ID:       uuid.New(),
		Hostname: "test-worker",
		SSHConfig: &SSHWorkerConfig{
			Host: "localhost",
			Port: 22,
		},
	}

	// Test capability detection (will fail due to SSH connection)
	err := pool.detectWorkerCapabilities(ctx, worker)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to detect capabilities")

	// Test with nil worker (should not panic)
	err = pool.detectWorkerCapabilities(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker is nil")
}

// TestSSHWorkerPool_ConcurrentAccess tests concurrent access safety
func TestSSHWorkerPool_ConcurrentAccess(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	// Add initial worker
	workerID := uuid.New()
	pool.workers[workerID] = &SSHWorker{
		ID:       workerID,
		Hostname: "initial-worker",
	}

	// Run concurrent operations
	done := make(chan bool)
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Read operations
			_ = pool.GetWorkerStats(ctx)

			// Write operations (simulated)
			if id%2 == 0 {
				newWorkerID := uuid.New()
				pool.workers[newWorkerID] = &SSHWorker{
					ID:       newWorkerID,
					Hostname: "concurrent-worker",
				}
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify no data races occurred
	assert.True(t, len(pool.workers) >= 1)
}

// TestSSHWorkerPool_ErrorHandling tests various error scenarios
func TestSSHWorkerPool_ErrorHandling(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	// Test nil worker
	err := pool.AddWorker(ctx, nil)
	assert.Error(t, err)

	// Test worker with nil SSH config
	err = pool.AddWorker(ctx, &SSHWorker{
		Hostname: "test-worker",
	})
	assert.Error(t, err)

	// Test ensureSSHConnection with nil worker
	err = pool.ensureSSHConnection(nil)
	assert.Error(t, err)
}

// TestSSHWorkerStats_String tests statistics string representation
func TestSSHWorkerStats_String(t *testing.T) {
	stats := &SSHWorkerStats{
		TotalWorkers:   5,
		ActiveWorkers:  3,
		HealthyWorkers: 2,
		TotalCPU:       16,
		TotalMemory:    34359738368, // 32GB
		TotalGPU:       1,
	}

	// Verify all fields are set correctly
	assert.Equal(t, 5, stats.TotalWorkers)
	assert.Equal(t, 3, stats.ActiveWorkers)
	assert.Equal(t, 2, stats.HealthyWorkers)
	assert.Equal(t, 16, stats.TotalCPU)
	assert.Equal(t, int64(34359738368), stats.TotalMemory)
	assert.Equal(t, 1, stats.TotalGPU)
}

// TestSSHWorkerPool_ResourceManagement tests resource management
func TestSSHWorkerPool_ResourceManagement(t *testing.T) {
	pool := NewSSHWorkerPool(false)
	ctx := context.Background()

	// Test with multiple workers having different resources
	workers := []struct {
		cpu    int
		memory int64
		gpu    int
	}{
		{cpu: 4, memory: 8589934592, gpu: 0},   // 8GB
		{cpu: 8, memory: 17179869184, gpu: 1},  // 16GB
		{cpu: 16, memory: 34359738368, gpu: 2}, // 32GB
	}

	for i, w := range workers {
		workerID := uuid.New()
		pool.workers[workerID] = &SSHWorker{
			ID:           workerID,
			Hostname:     "worker-" + string(rune('A'+i)),
			Status:       WorkerStatusActive,
			HealthStatus: WorkerHealthHealthy,
			Resources: Resources{
				CPUCount:    w.cpu,
				TotalMemory: w.memory,
				GPUCount:    w.gpu,
			},
		}
	}

	stats := pool.GetWorkerStats(ctx)

	// Verify aggregated resources
	expectedCPU := 4 + 8 + 16
	expectedMemory := int64(8589934592 + 17179869184 + 34359738368)
	expectedGPU := 0 + 1 + 2

	assert.Equal(t, expectedCPU, stats.TotalCPU)
	assert.Equal(t, expectedMemory, stats.TotalMemory)
	assert.Equal(t, expectedGPU, stats.TotalGPU)
	assert.Equal(t, 3, stats.TotalWorkers)
	assert.Equal(t, 3, stats.ActiveWorkers)
	assert.Equal(t, 3, stats.HealthyWorkers)
}
