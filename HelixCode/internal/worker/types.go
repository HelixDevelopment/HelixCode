package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WorkerConfig represents the configuration for distributed worker management
type WorkerConfig struct {
	Enabled             bool                         `json:"enabled"`
	Pool                map[string]WorkerConfigEntry `json:"pool"`
	AutoInstall         bool                         `json:"auto_install"`
	HealthCheckInterval int                          `json:"health_check_interval"`
	MaxConcurrentTasks  int                          `json:"max_concurrent_tasks"`
	TaskTimeout         int                          `json:"task_timeout"`
}

// WorkerConfigEntry represents a single worker configuration entry
type WorkerConfigEntry struct {
	Host         string   `json:"host"`
	Port         int      `json:"port"`
	Username     string   `json:"username"`
	KeyPath      string   `json:"key_path"`
	Capabilities []string `json:"capabilities"`
	DisplayName  string   `json:"display_name"`
}

// DistributedTask represents a task for distributed execution
type DistributedTask struct {
	ID           uuid.UUID              `json:"id"`
	Type         string                 `json:"type"`
	Payload      map[string]interface{} `json:"payload"`
	Data         map[string]interface{} `json:"data"`
	WorkerID     uuid.UUID              `json:"worker_id"`
	Status       TaskStatus             `json:"status"`
	Priority     int                    `json:"priority"`
	Criticality  Criticality            `json:"criticality"`
	MaxRetries   int                    `json:"max_retries"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at"`
	ErrorMessage string                 `json:"error_message"`
	Result       map[string]interface{} `json:"result"`
}

// TaskStatus represents the status of a distributed task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Criticality represents the criticality level of a task
type Criticality string

const (
	CriticalityLow      Criticality = "low"
	CriticalityNormal   Criticality = "normal"
	CriticalityHigh     Criticality = "high"
	CriticalityCritical Criticality = "critical"
)

// DistributedWorkerManager manages distributed workers
type DistributedWorkerManager struct {
	config  WorkerConfig
	workers map[uuid.UUID]*Worker
	tasks   map[uuid.UUID]*DistributedTask
	sshPool *SSHWorkerPool
}

// NewDistributedWorkerManager creates a new distributed worker manager
func NewDistributedWorkerManager(config WorkerConfig) *DistributedWorkerManager {
	return &DistributedWorkerManager{
		config:  config,
		workers: make(map[uuid.UUID]*Worker),
		tasks:   make(map[uuid.UUID]*DistributedTask),
		sshPool: NewSSHWorkerPool(config.AutoInstall),
	}
}

// Initialize initializes the distributed worker manager
func (dwm *DistributedWorkerManager) Initialize(ctx context.Context) error {
	// Initialize SSH connections to configured workers
	for name, entry := range dwm.config.Pool {
		worker := &SSHWorker{
			Hostname:    entry.Host,
			DisplayName: entry.DisplayName,
			SSHConfig: &SSHWorkerConfig{
				Host:     entry.Host,
				Port:     entry.Port,
				Username: entry.Username,
				KeyPath:  entry.KeyPath,
			},
			Capabilities: entry.Capabilities,
		}

		if err := dwm.sshPool.AddWorker(ctx, worker); err != nil {
			return fmt.Errorf("failed to add worker %s: %v", name, err)
		}
	}

	return nil
}

// GetAvailableWorkers returns all available workers
func (dwm *DistributedWorkerManager) GetAvailableWorkers() []*Worker {
	workers := make([]*Worker, 0, len(dwm.workers))
	for _, worker := range dwm.workers {
		if worker.Status == WorkerStatusActive && worker.HealthStatus == WorkerHealthHealthy {
			workers = append(workers, worker)
		}
	}
	return workers
}

// GetWorkerStats returns statistics about workers
func (dwm *DistributedWorkerManager) GetWorkerStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["total_workers"] = len(dwm.workers)

	activeCount := 0
	healthyCount := 0
	totalTasks := 0

	for _, worker := range dwm.workers {
		if worker.Status == WorkerStatusActive {
			activeCount++
		}
		if worker.HealthStatus == WorkerHealthHealthy {
			healthyCount++
		}
		totalTasks += worker.CurrentTasksCount
	}

	stats["active_workers"] = activeCount
	stats["healthy_workers"] = healthyCount
	stats["total_tasks"] = totalTasks

	return stats
}

// SubmitTask submits a task for distributed execution
func (dwm *DistributedWorkerManager) SubmitTask(task *DistributedTask) error {
	task.ID = uuid.New()
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()

	dwm.tasks[task.ID] = task

	// Find suitable worker
	availableWorkers := dwm.GetAvailableWorkers()
	if len(availableWorkers) == 0 {
		return fmt.Errorf("no available workers")
	}

	// Simple round-robin assignment for now
	worker := availableWorkers[0]
	task.WorkerID = worker.ID

	// Execute task (in real implementation, this would be async)
	return dwm.executeTask(task)
}

// executeTask executes a task on the assigned worker
func (dwm *DistributedWorkerManager) executeTask(task *DistributedTask) error {
	now := time.Now()
	task.StartedAt = &now
	task.Status = TaskStatusRunning

	// Simulate task execution
	// In real implementation, this would execute via SSH
	time.Sleep(100 * time.Millisecond)

	completedAt := time.Now()
	task.CompletedAt = &completedAt
	task.Status = TaskStatusCompleted
	task.Result = map[string]interface{}{
		"output":   "Task completed successfully",
		"duration": completedAt.Sub(now).String(),
	}

	return nil
}
