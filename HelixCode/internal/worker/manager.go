package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkerStatus represents the status of a worker
type WorkerStatus string

const (
	WorkerStatusActive      WorkerStatus = "active"
	WorkerStatusInactive    WorkerStatus = "inactive"
	WorkerStatusMaintenance WorkerStatus = "maintenance"
	WorkerStatusFailed      WorkerStatus = "failed"
	WorkerStatusOffline     WorkerStatus = "offline"
)

// WorkerHealth represents the health status of a worker
type WorkerHealth string

const (
	WorkerHealthHealthy   WorkerHealth = "healthy"
	WorkerHealthDegraded  WorkerHealth = "degraded"
	WorkerHealthUnhealthy WorkerHealth = "unhealthy"
	WorkerHealthUnknown   WorkerHealth = "unknown"
)

// Worker represents a distributed worker node
type Worker struct {
	ID                 uuid.UUID              `json:"id"`
	Hostname           string                 `json:"hostname"`
	DisplayName        string                 `json:"display_name"`
	SSHConfig          map[string]interface{} `json:"ssh_config"`
	Capabilities       []string               `json:"capabilities"`
	Resources          Resources              `json:"resources"`
	Status             WorkerStatus           `json:"status"`
	HealthStatus       WorkerHealth           `json:"health_status"`
	LastHeartbeat      time.Time              `json:"last_heartbeat"`
	CPUUsagePercent    float64                `json:"cpu_usage_percent"`
	MemoryUsagePercent float64                `json:"memory_usage_percent"`
	DiskUsagePercent   float64                `json:"disk_usage_percent"`
	CurrentTasksCount  int                    `json:"current_tasks_count"`
	MaxConcurrentTasks int                    `json:"max_concurrent_tasks"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// Resources represents worker hardware resources
type Resources struct {
	CPUCount    int    `json:"cpu_count"`
	TotalMemory int64  `json:"total_memory"` // in bytes
	TotalDisk   int64  `json:"total_disk"`   // in bytes
	GPUCount    int    `json:"gpu_count"`
	GPUModel    string `json:"gpu_model"`
	GPUMemory   int64  `json:"gpu_memory"` // in bytes
}

// WorkerMetrics represents metrics collected from a worker
type WorkerMetrics struct {
	ID                 uuid.UUID `json:"id"`
	WorkerID           uuid.UUID `json:"worker_id"`
	CPUUsagePercent    float64   `json:"cpu_usage_percent"`
	MemoryUsagePercent float64   `json:"memory_usage_percent"`
	DiskUsagePercent   float64   `json:"disk_usage_percent"`
	NetworkRxBytes     int64     `json:"network_rx_bytes"`
	NetworkTxBytes     int64     `json:"network_tx_bytes"`
	CurrentTasksCount  int       `json:"current_tasks_count"`
	TemperatureCelsius float64   `json:"temperature_celsius"`
	RecordedAt         time.Time `json:"recorded_at"`
}

// WorkerRepository defines the interface for worker data storage
type WorkerRepository interface {
	CreateWorker(ctx context.Context, worker *Worker) error
	GetWorker(ctx context.Context, id uuid.UUID) (*Worker, error)
	GetWorkerByHostname(ctx context.Context, hostname string) (*Worker, error)
	ListWorkers(ctx context.Context, status WorkerStatus) ([]*Worker, error)
	UpdateWorker(ctx context.Context, worker *Worker) error
	DeleteWorker(ctx context.Context, id uuid.UUID) error
	RecordMetrics(ctx context.Context, metrics *WorkerMetrics) error
	GetWorkerMetrics(ctx context.Context, workerID uuid.UUID, since time.Time) ([]*WorkerMetrics, error)
}

// WorkerManager manages distributed workers
type WorkerManager struct {
	repo      WorkerRepository
	workers   map[uuid.UUID]*Worker
	mutex     sync.RWMutex
	healthTTL time.Duration
}

// NewWorkerManager creates a new worker manager
func NewWorkerManager(repo WorkerRepository, healthTTL time.Duration) *WorkerManager {
	return &WorkerManager{
		repo:      repo,
		workers:   make(map[uuid.UUID]*Worker),
		healthTTL: healthTTL,
	}
}

// RegisterWorker registers a new worker
func (wm *WorkerManager) RegisterWorker(ctx context.Context, worker *Worker) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	// Check if worker already exists
	existing, err := wm.repo.GetWorkerByHostname(ctx, worker.Hostname)
	if err == nil && existing != nil {
		// Update existing worker
		worker.ID = existing.ID
		worker.CreatedAt = existing.CreatedAt
	} else {
		// Create new worker
		worker.ID = uuid.New()
		worker.CreatedAt = time.Now()
	}

	worker.UpdatedAt = time.Now()
	worker.Status = WorkerStatusActive
	worker.HealthStatus = WorkerHealthHealthy
	worker.LastHeartbeat = time.Now()

	if err := wm.repo.CreateWorker(ctx, worker); err != nil {
		return fmt.Errorf("failed to register worker: %v", err)
	}

	// Cache worker
	wm.workers[worker.ID] = worker

	log.Printf("✅ Worker registered: %s (%s)", worker.Hostname, worker.ID)
	return nil
}

// UpdateWorkerHeartbeat updates a worker's heartbeat
func (wm *WorkerManager) UpdateWorkerHeartbeat(ctx context.Context, workerID uuid.UUID, metrics *WorkerMetrics) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	worker, err := wm.repo.GetWorker(ctx, workerID)
	if err != nil {
		return fmt.Errorf("worker not found: %v", err)
	}

	worker.LastHeartbeat = time.Now()
	worker.UpdatedAt = time.Now()

	// Update metrics if provided
	if metrics != nil {
		worker.CPUUsagePercent = metrics.CPUUsagePercent
		worker.MemoryUsagePercent = metrics.MemoryUsagePercent
		worker.DiskUsagePercent = metrics.DiskUsagePercent
		worker.CurrentTasksCount = metrics.CurrentTasksCount

		// Record metrics
		metrics.ID = uuid.New()
		metrics.WorkerID = workerID
		metrics.RecordedAt = time.Now()
		if err := wm.repo.RecordMetrics(ctx, metrics); err != nil {
			log.Printf("Warning: Failed to record metrics: %v", err)
		}

		// Update health status based on metrics
		worker.HealthStatus = wm.calculateHealthStatus(worker, metrics)
	}

	if err := wm.repo.UpdateWorker(ctx, worker); err != nil {
		return fmt.Errorf("failed to update worker heartbeat: %v", err)
	}

	// Update cache
	wm.workers[workerID] = worker

	return nil
}

// GetAvailableWorkers returns available workers for task assignment
func (wm *WorkerManager) GetAvailableWorkers(ctx context.Context, capabilities []string) ([]*Worker, error) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	// Get all active workers
	workers, err := wm.repo.ListWorkers(ctx, WorkerStatusActive)
	if err != nil {
		return nil, err
	}

	var available []*Worker
	for _, worker := range workers {
		// Check if worker is healthy and has capacity
		if worker.HealthStatus == WorkerHealthHealthy &&
			worker.CurrentTasksCount < worker.MaxConcurrentTasks &&
			wm.hasCapabilities(worker, capabilities) &&
			wm.isWorkerHealthy(worker) {
			available = append(available, worker)
		}
	}

	return available, nil
}

// AssignTask assigns a task to a worker
func (wm *WorkerManager) AssignTask(ctx context.Context, workerID uuid.UUID) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	worker, err := wm.repo.GetWorker(ctx, workerID)
	if err != nil {
		return fmt.Errorf("worker not found: %v", err)
	}

	if worker.CurrentTasksCount >= worker.MaxConcurrentTasks {
		return errors.New("worker at maximum capacity")
	}

	worker.CurrentTasksCount++
	worker.UpdatedAt = time.Now()

	if err := wm.repo.UpdateWorker(ctx, worker); err != nil {
		return fmt.Errorf("failed to assign task: %v", err)
	}

	// Update cache
	wm.workers[workerID] = worker

	return nil
}

// CompleteTask marks a task as completed on a worker
func (wm *WorkerManager) CompleteTask(ctx context.Context, workerID uuid.UUID) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	worker, err := wm.repo.GetWorker(ctx, workerID)
	if err != nil {
		return fmt.Errorf("worker not found: %v", err)
	}

	if worker.CurrentTasksCount > 0 {
		worker.CurrentTasksCount--
		worker.UpdatedAt = time.Now()

		if err := wm.repo.UpdateWorker(ctx, worker); err != nil {
			return fmt.Errorf("failed to complete task: %v", err)
		}

		// Update cache
		wm.workers[workerID] = worker
	}

	return nil
}

// HealthCheck performs health checks on all workers
func (wm *WorkerManager) HealthCheck(ctx context.Context) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	workers, err := wm.repo.ListWorkers(ctx, WorkerStatusActive)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, worker := range workers {
		// Check if worker has timed out
		if now.Sub(worker.LastHeartbeat) > wm.healthTTL {
			worker.HealthStatus = WorkerHealthUnhealthy
			worker.Status = WorkerStatusOffline
			worker.UpdatedAt = now

			if err := wm.repo.UpdateWorker(ctx, worker); err != nil {
				log.Printf("Warning: Failed to update unhealthy worker %s: %v", worker.Hostname, err)
			} else {
				log.Printf("⚠️  Worker marked as unhealthy: %s (last heartbeat: %v)",
					worker.Hostname, worker.LastHeartbeat)
			}

			// Update cache
			wm.workers[worker.ID] = worker
		}
	}

	return nil
}

// GetWorkerStats returns statistics about workers
func (wm *WorkerManager) GetWorkerStats(ctx context.Context) (*WorkerStats, error) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	workers, err := wm.repo.ListWorkers(ctx, "") // All workers
	if err != nil {
		return nil, err
	}

	stats := &WorkerStats{
		TotalWorkers:       len(workers),
		ActiveWorkers:      0,
		HealthyWorkers:     0,
		TotalTasks:         0,
		AvailableTasks:     0,
		AverageCPUUsage:    0,
		AverageMemoryUsage: 0,
	}

	totalCPU := 0.0
	totalMemory := 0.0

	for _, worker := range workers {
		if worker.Status == WorkerStatusActive {
			stats.ActiveWorkers++
		}
		if worker.HealthStatus == WorkerHealthHealthy {
			stats.HealthyWorkers++
		}

		stats.TotalTasks += worker.CurrentTasksCount
		stats.AvailableTasks += worker.MaxConcurrentTasks - worker.CurrentTasksCount

		totalCPU += worker.CPUUsagePercent
		totalMemory += worker.MemoryUsagePercent
	}

	if len(workers) > 0 {
		stats.AverageCPUUsage = totalCPU / float64(len(workers))
		stats.AverageMemoryUsage = totalMemory / float64(len(workers))
	}

	return stats, nil
}

// WorkerStats represents statistics about workers
type WorkerStats struct {
	TotalWorkers       int     `json:"total_workers"`
	ActiveWorkers      int     `json:"active_workers"`
	HealthyWorkers     int     `json:"healthy_workers"`
	TotalTasks         int     `json:"total_tasks"`
	AvailableTasks     int     `json:"available_tasks"`
	AverageCPUUsage    float64 `json:"average_cpu_usage"`
	AverageMemoryUsage float64 `json:"average_memory_usage"`
}

// Helper methods

func (wm *WorkerManager) hasCapabilities(worker *Worker, required []string) bool {
	if len(required) == 0 {
		return true
	}

	workerCaps := make(map[string]bool)
	for _, cap := range worker.Capabilities {
		workerCaps[cap] = true
	}

	for _, req := range required {
		if !workerCaps[req] {
			return false
		}
	}

	return true
}

func (wm *WorkerManager) isWorkerHealthy(worker *Worker) bool {
	return worker.HealthStatus == WorkerHealthHealthy &&
		time.Since(worker.LastHeartbeat) <= wm.healthTTL
}

func (wm *WorkerManager) calculateHealthStatus(worker *Worker, metrics *WorkerMetrics) WorkerHealth {
	// Simple health calculation based on resource usage
	if metrics.CPUUsagePercent > 90 || metrics.MemoryUsagePercent > 90 || metrics.DiskUsagePercent > 90 {
		return WorkerHealthUnhealthy
	}

	if metrics.CPUUsagePercent > 70 || metrics.MemoryUsagePercent > 70 || metrics.DiskUsagePercent > 70 {
		return WorkerHealthDegraded
	}

	return WorkerHealthHealthy
}

// SSHConfig represents SSH configuration for worker connections
type SSHConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	PrivateKey string `json:"private_key"`
	Password   string `json:"password"`
}

// ParseSSHConfig parses SSH configuration from JSON
func ParseSSHConfig(configJSON string) (*SSHConfig, error) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	sshConfig := &SSHConfig{
		Host:     getString(config, "host", ""),
		Port:     getInt(config, "port", 22),
		Username: getString(config, "username", ""),
	}

	// Handle private key and password
	if privateKey, ok := config["private_key"].(string); ok {
		sshConfig.PrivateKey = privateKey
	}
	if password, ok := config["password"].(string); ok {
		sshConfig.Password = password
	}

	return sshConfig, nil
}

func getString(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}
