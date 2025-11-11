package worker

import (
	"context"
	"fmt"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DatabaseManager handles worker lifecycle and operations with database persistence
type DatabaseManager struct {
	db database.DatabaseInterface
}

// NewDatabaseManager creates a new worker manager with database persistence
func NewDatabaseManager(db database.DatabaseInterface) *DatabaseManager {
	return &DatabaseManager{
		db: db,
	}
}

// GetWorker retrieves a worker by ID from database
func (m *DatabaseManager) GetWorker(ctx context.Context, id string) (*Worker, error) {
	workerID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid worker ID: %v", err)
	}

	query := `
		SELECT 
			id, hostname, display_name, ssh_config, capabilities, resources,
			status, health_status, last_heartbeat, cpu_usage_percent,
			memory_usage_percent, disk_usage_percent, current_tasks_count,
			max_concurrent_tasks, created_at, updated_at
		FROM workers
		WHERE id = $1
	`

	var (
		dbID               uuid.UUID
		hostname           string
		displayName        string
		sshConfig          map[string]interface{}
		capabilities       []string
		resources          map[string]interface{}
		status             string
		healthStatus       string
		lastHeartbeat      *time.Time
		cpuUsagePercent    *float64
		memoryUsagePercent *float64
		diskUsagePercent   *float64
		currentTasksCount  int
		maxConcurrentTasks int
		createdAt          time.Time
		updatedAt          time.Time
	)

	err = m.db.QueryRow(ctx, query, workerID).Scan(
		&dbID, &hostname, &displayName, &sshConfig, &capabilities, &resources,
		&status, &healthStatus, &lastHeartbeat, &cpuUsagePercent,
		&memoryUsagePercent, &diskUsagePercent, &currentTasksCount,
		&maxConcurrentTasks, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("worker not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get worker from database: %v", err)
	}

	// Convert nullable fields
	var lastHeartbeatTime time.Time
	if lastHeartbeat != nil {
		lastHeartbeatTime = *lastHeartbeat
	}

	var cpuUsage float64
	if cpuUsagePercent != nil {
		cpuUsage = *cpuUsagePercent
	}

	var memoryUsage float64
	if memoryUsagePercent != nil {
		memoryUsage = *memoryUsagePercent
	}

	var diskUsage float64
	if diskUsagePercent != nil {
		diskUsage = *diskUsagePercent
	}

	worker := &Worker{
		ID:                 dbID,
		Hostname:           hostname,
		DisplayName:        displayName,
		SSHConfig:          sshConfig,
		Capabilities:       capabilities,
		Resources:          parseResources(resources),
		Status:             WorkerStatus(status),
		HealthStatus:       WorkerHealth(healthStatus),
		LastHeartbeat:      lastHeartbeatTime,
		CPUUsagePercent:    cpuUsage,
		MemoryUsagePercent: memoryUsage,
		DiskUsagePercent:   diskUsage,
		CurrentTasksCount:  currentTasksCount,
		MaxConcurrentTasks: maxConcurrentTasks,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}

	return worker, nil
}

// ListWorkers returns all workers from database
func (m *DatabaseManager) ListWorkers(ctx context.Context) ([]*Worker, error) {
	query := `
		SELECT 
			id, hostname, display_name, ssh_config, capabilities, resources,
			status, health_status, last_heartbeat, cpu_usage_percent,
			memory_usage_percent, disk_usage_percent, current_tasks_count,
			max_concurrent_tasks, created_at, updated_at
		FROM workers
		ORDER BY created_at DESC
	`

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query workers: %v", err)
	}
	defer rows.Close()

	var workers []*Worker
	for rows.Next() {
		var (
			dbID               uuid.UUID
			hostname           string
			displayName        string
			sshConfig          map[string]interface{}
			capabilities       []string
			resources          map[string]interface{}
			status             string
			healthStatus       string
			lastHeartbeat      *time.Time
			cpuUsagePercent    *float64
			memoryUsagePercent *float64
			diskUsagePercent   *float64
			currentTasksCount  int
			maxConcurrentTasks int
			createdAt          time.Time
			updatedAt          time.Time
		)

		if err := rows.Scan(
			&dbID, &hostname, &displayName, &sshConfig, &capabilities, &resources,
			&status, &healthStatus, &lastHeartbeat, &cpuUsagePercent,
			&memoryUsagePercent, &diskUsagePercent, &currentTasksCount,
			&maxConcurrentTasks, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan worker row: %v", err)
		}

		// Convert nullable fields
		var lastHeartbeatTime time.Time
		if lastHeartbeat != nil {
			lastHeartbeatTime = *lastHeartbeat
		}

		var cpuUsage float64
		if cpuUsagePercent != nil {
			cpuUsage = *cpuUsagePercent
		}

		var memoryUsage float64
		if memoryUsagePercent != nil {
			memoryUsage = *memoryUsagePercent
		}

		var diskUsage float64
		if diskUsagePercent != nil {
			diskUsage = *diskUsagePercent
		}

		worker := &Worker{
			ID:                 dbID,
			Hostname:           hostname,
			DisplayName:        displayName,
			SSHConfig:          sshConfig,
			Capabilities:       capabilities,
			Resources:          parseResources(resources),
			Status:             WorkerStatus(status),
			HealthStatus:       WorkerHealth(healthStatus),
			LastHeartbeat:      lastHeartbeatTime,
			CPUUsagePercent:    cpuUsage,
			MemoryUsagePercent: memoryUsage,
			DiskUsagePercent:   diskUsage,
			CurrentTasksCount:  currentTasksCount,
			MaxConcurrentTasks: maxConcurrentTasks,
			CreatedAt:          createdAt,
			UpdatedAt:          updatedAt,
		}

		workers = append(workers, worker)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating worker rows: %v", err)
	}

	return workers, nil
}

// RegisterWorker registers a new worker in the system
func (m *DatabaseManager) RegisterWorker(ctx context.Context, hostname, displayName string, sshConfig map[string]interface{}, capabilities []string, resources map[string]interface{}) (*Worker, error) {
	worker := &Worker{
		ID:                 uuid.New(),
		Hostname:           hostname,
		DisplayName:        displayName,
		SSHConfig:          sshConfig,
		Capabilities:       capabilities,
		Resources:          parseResources(resources),
		Status:             "active",
		HealthStatus:       "healthy",
		CurrentTasksCount:  0,
		MaxConcurrentTasks: 10,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	query := `
		INSERT INTO workers (
			id, hostname, display_name, ssh_config, capabilities, resources,
			status, health_status, current_tasks_count, max_concurrent_tasks,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := m.db.QueryRow(ctx, query,
		worker.ID, worker.Hostname, worker.DisplayName, worker.SSHConfig,
		worker.Capabilities, worker.Resources, worker.Status, worker.HealthStatus,
		worker.CurrentTasksCount, worker.MaxConcurrentTasks,
		worker.CreatedAt, worker.UpdatedAt,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to register worker in database: %v", err)
	}

	worker.CreatedAt = createdAt
	worker.UpdatedAt = updatedAt

	return worker, nil
}

// UpdateWorkerHeartbeat updates worker heartbeat and metrics
func (m *DatabaseManager) UpdateWorkerHeartbeat(ctx context.Context, id string, metrics map[string]interface{}) error {
	workerID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid worker ID: %v", err)
	}

	// Update worker record
	query := `
		UPDATE workers
		SET last_heartbeat = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err = m.db.Exec(ctx, query, workerID)
	if err != nil {
		return fmt.Errorf("failed to update worker heartbeat: %v", err)
	}

	// Store metrics
	metricsQuery := `
		INSERT INTO worker_metrics (
			worker_id, cpu_usage_percent, memory_usage_percent, disk_usage_percent,
			network_rx_bytes, network_tx_bytes, current_tasks_count, temperature_celsius
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	cpuUsage, _ := metrics["cpu_usage_percent"].(float64)
	memoryUsage, _ := metrics["memory_usage_percent"].(float64)
	diskUsage, _ := metrics["disk_usage_percent"].(float64)
	networkRx, _ := metrics["network_rx_bytes"].(int64)
	networkTx, _ := metrics["network_tx_bytes"].(int64)
	currentTasks, _ := metrics["current_tasks_count"].(int)
	temperature, _ := metrics["temperature_celsius"].(float64)

	_, err = m.db.Exec(ctx, metricsQuery,
		workerID, cpuUsage, memoryUsage, diskUsage,
		networkRx, networkTx, currentTasks, temperature,
	)

	if err != nil {
		return fmt.Errorf("failed to store worker metrics: %v", err)
	}

	return nil
}

// Helper functions for parsing

func getStringDB(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

func getIntDB(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func getInt64DB(m map[string]interface{}, key string, defaultValue int64) int64 {
	if val, ok := m[key].(float64); ok {
		return int64(val)
	}
	return defaultValue
}

// parseResources converts a map to Resources struct
func parseResources(resources map[string]interface{}) Resources {
	return Resources{
		CPUCount:    getIntDB(resources, "cpu_count", 0),
		TotalMemory: getInt64DB(resources, "total_memory", 0),
		TotalDisk:   getInt64DB(resources, "total_disk", 0),
		GPUCount:    getIntDB(resources, "gpu_count", 0),
		GPUModel:    getStringDB(resources, "gpu_model", ""),
		GPUMemory:   getInt64DB(resources, "gpu_memory", 0),
	}
}
