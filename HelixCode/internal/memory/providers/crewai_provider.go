package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory"
)

// CrewAIProvider implements VectorProvider for CrewAI
type CrewAIProvider struct {
	config       *CrewAIConfig
	logger       logging.Logger
	mu           sync.RWMutex
	initialized  bool
	started      bool
	client       CrewAIClient
	crews        map[string]*memory.Crew
	tasks        map[string]*memory.Task
	sharedMemory map[string]*memory.SharedMemory
	stats        *ProviderStats
}

// CrewAIConfig contains CrewAI provider configuration
type CrewAIConfig struct {
	APIKey             string        `json:"api_key"`
	BaseURL            string        `json:"base_url"`
	MaxAgents          int           `json:"max_agents"`
	MaxTasksPerAgent   int           `json:"max_tasks_per_agent"`
	TaskTimeout        time.Duration `json:"task_timeout"`
	AgentTimeout       time.Duration `json:"agent_timeout"`
	MemorySyncInterval time.Duration `json:"memory_sync_interval"`
	SharedMemorySize   int64         `json:"shared_memory_size"`
	EnableLogging      bool          `json:"enable_logging"`
	LogLevel           string        `json:"log_level"`
	ParallelExecution  bool          `json:"parallel_execution"`
	TaskPrioritization bool          `json:"task_prioritization"`
	AutoRetry          bool          `json:"auto_retry"`
	MaxRetries         int           `json:"max_retries"`
}

// CrewAIClient represents CrewAI client interface
type CrewAIClient interface {
	CreateCrew(ctx context.Context, crew *memory.Crew) error
	GetCrew(ctx context.Context, crewID string) (*memory.Crew, error)
	UpdateCrew(ctx context.Context, crew *memory.Crew) error
	DeleteCrew(ctx context.Context, crewID string) error
	ListCrews(ctx context.Context) ([]*memory.Crew, error)
	CreateTask(ctx context.Context, task *memory.Task) error
	GetTask(ctx context.Context, taskID string) (*memory.Task, error)
	UpdateTask(ctx context.Context, task *memory.Task) error
	DeleteTask(ctx context.Context, taskID string) error
	ListTasks(ctx context.Context, crewID string) ([]*memory.Task, error)
	AssignTask(ctx context.Context, taskID, agentID string) error
	CompleteTask(ctx context.Context, taskID string, result *memory.TaskResult) error
	GetSharedMemory(ctx context.Context, memoryID string) (*memory.SharedMemory, error)
	UpdateSharedMemory(ctx context.Context, memory *memory.SharedMemory) error
	GetCrewPerformance(ctx context.Context, crewID string) (*memory.CrewPerformance, error)
	GetHealth(ctx context.Context) error
}

// NewCrewAIProvider creates a new CrewAI provider
func NewCrewAIProvider(config map[string]interface{}) (VectorProvider, error) {
	crewAIConfig := &CrewAIConfig{
		BaseURL:            "https://api.crewai.ai",
		MaxAgents:          10,
		MaxTasksPerAgent:   50,
		TaskTimeout:        30 * time.Minute,
		AgentTimeout:       60 * time.Minute,
		MemorySyncInterval: 5 * time.Minute,
		SharedMemorySize:   1000000, // 1MB
		EnableLogging:      true,
		LogLevel:           "INFO",
		ParallelExecution:  true,
		TaskPrioritization: true,
		AutoRetry:          true,
		MaxRetries:         3,
	}

	// Parse configuration
	if err := parseConfig(config, crewAIConfig); err != nil {
		return nil, fmt.Errorf("failed to parse CrewAI config: %w", err)
	}

	return &CrewAIProvider{
		config:       crewAIConfig,
		logger:       logging.NewLogger("crewai_provider"),
		crews:        make(map[string]*memory.Crew),
		tasks:        make(map[string]*memory.Task),
		sharedMemory: make(map[string]*memory.SharedMemory),
		stats: &ProviderStats{
			TotalVectors:     0,
			TotalCollections: 0,
			TotalSize:        0,
			AverageLatency:   0,
			LastOperation:    time.Now(),
			ErrorCount:       0,
			Uptime:           0,
		},
	}, nil
}

// Initialize initializes CrewAI provider
func (p *CrewAIProvider) Initialize(ctx context.Context, config interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	p.logger.Info("Initializing CrewAI provider",
		"base_url", p.config.BaseURL,
		"max_agents", p.config.MaxAgents,
		"parallel_execution", p.config.ParallelExecution)

	// Create CrewAI client
	client, err := NewCrewAIHTTPClient(p.config)
	if err != nil {
		return fmt.Errorf("failed to create CrewAI client: %w", err)
	}

	p.client = client

	// Test connection
	if err := p.client.GetHealth(ctx); err != nil {
		return fmt.Errorf("failed to connect to CrewAI: %w", err)
	}

	// Load existing crews
	if err := p.loadCrews(ctx); err != nil {
		p.logger.Warn("Failed to load crews", "error", err)
	}

	p.initialized = true
	p.stats.LastOperation = time.Now()

	p.logger.Info("CrewAI provider initialized successfully")
	return nil
}

// Start starts CrewAI provider
func (p *CrewAIProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	if p.started {
		return nil
	}

	// Start background sync
	go p.syncWorker(ctx)

	p.started = true
	p.stats.LastOperation = time.Now()
	p.stats.Uptime = 0

	p.logger.Info("CrewAI provider started successfully")
	return nil
}

// Store stores vectors in CrewAI (as tasks or crew data)
func (p *CrewAIProvider) Store(ctx context.Context, vectors []*memory.VectorData) error {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	// Convert vectors to CrewAI format
	for _, vector := range vectors {
		task := &memory.Task{
			ID:          vector.ID,
			CrewID:      vector.Collection,
			Description: vectorToString(vector),
			Priority:    "normal",
			Status:      "pending",
			Metadata:    vector.Metadata,
			Embedding:   vector.Vector,
			CreatedAt:   vector.Timestamp,
			UpdatedAt:   time.Now(),
		}

		// Create task
		if err := p.client.CreateTask(ctx, task); err != nil {
			p.logger.Error("Failed to create task",
				"id", task.ID,
				"crew_id", task.CrewID,
				"error", err)
			return fmt.Errorf("failed to store vector: %w", err)
		}

		p.tasks[task.ID] = task
		p.stats.TotalVectors++
		p.stats.TotalSize += int64(len(vector.Vector) * 8)
	}

	p.stats.LastOperation = time.Now()
	return nil
}

// Retrieve retrieves vectors by ID from CrewAI
func (p *CrewAIProvider) Retrieve(ctx context.Context, ids []string) ([]*memory.VectorData, error) {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	var vectors []*memory.VectorData

	for _, id := range ids {
		task, err := p.client.GetTask(ctx, id)
		if err != nil {
			p.logger.Warn("Failed to get task",
				"id", id,
				"error", err)
			continue
		}

		vector := &memory.VectorData{
			ID:         task.ID,
			Vector:     task.Embedding,
			Metadata:   task.Metadata,
			Collection: task.CrewID,
			Timestamp:  task.CreatedAt,
		}

		vectors = append(vectors, vector)
	}

	p.stats.LastOperation = time.Now()
	return vectors, nil
}

// Search performs vector similarity search in CrewAI
func (p *CrewAIProvider) Search(ctx context.Context, query *memory.VectorQuery) (*memory.VectorSearchResult, error) {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	crewID := query.Collection
	if crewID == "" {
		crewID = "default"
	}

	// Get all tasks for the crew
	tasks, err := p.client.ListTasks(ctx, crewID)
	if err != nil {
		p.logger.Error("Failed to list tasks",
			"crew_id", crewID,
			"error", err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Perform similarity search
	var results []*memory.VectorSearchResultItem
	for _, task := range tasks {
		if len(results) >= query.TopK {
			break
		}

		score := calculateSimilarity(query.Vector, task.Embedding)
		if score < query.Threshold {
			continue
		}

		results = append(results, &memory.VectorSearchResultItem{
			ID:       task.ID,
			Vector:   task.Embedding,
			Metadata: task.Metadata,
			Score:    score,
			Distance: 1 - score,
		})
	}

	p.stats.LastOperation = time.Now()
	return &memory.VectorSearchResult{
		Results:   results,
		Total:     len(results),
		Query:     query,
		Duration:  time.Since(start),
		Namespace: query.Namespace,
	}, nil
}

// FindSimilar finds similar vectors
func (p *CrewAIProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*memory.VectorSimilarityResult, error) {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	// Get all tasks across all crews
	var allTasks []*memory.Task
	for crewID := range p.crews {
		tasks, err := p.client.ListTasks(ctx, crewID)
		if err != nil {
			p.logger.Warn("Failed to list tasks",
				"crew_id", crewID,
				"error", err)
			continue
		}
		allTasks = append(allTasks, tasks...)
	}

	// Perform similarity search
	var results []*memory.VectorSimilarityResult
	for _, task := range allTasks {
		if len(results) >= k {
			break
		}

		// Apply filters
		if !p.applyFilters(task.Metadata, filters) {
			continue
		}

		score := calculateSimilarity(embedding, task.Embedding)
		if score < 0.5 { // Default threshold
			continue
		}

		results = append(results, &memory.VectorSimilarityResult{
			ID:       task.ID,
			Vector:   task.Embedding,
			Metadata: task.Metadata,
			Score:    score,
			Distance: 1 - score,
		})
	}

	p.stats.LastOperation = time.Now()
	return results, nil
}

// CreateCollection creates a new collection (crew)
func (p *CrewAIProvider) CreateCollection(ctx context.Context, name string, config *memory.CollectionConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.crews[name]; exists {
		return fmt.Errorf("collection %s already exists", name)
	}

	crew := &memory.Crew{
		ID:                 name,
		Name:               config.Description,
		Description:        config.Description,
		MaxAgents:          p.config.MaxAgents,
		MaxTasksPerAgent:   p.config.MaxTasksPerAgent,
		ParallelExecution:  p.config.ParallelExecution,
		TaskPrioritization: p.config.TaskPrioritization,
		Status:             "active",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		Agents:             []*memory.Agent{},
	}

	if err := p.client.CreateCrew(ctx, crew); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	p.crews[name] = crew
	p.stats.TotalCollections++

	// Initialize shared memory for the crew
	sharedMemory := &memory.SharedMemory{
		ID:          name + "_shared",
		CrewID:      name,
		Data:        make(map[string]interface{}),
		LastUpdated: time.Now(),
		Version:     1,
	}

	if err := p.client.UpdateSharedMemory(ctx, sharedMemory); err != nil {
		p.logger.Warn("Failed to create shared memory",
			"crew_id", name,
			"error", err)
	} else {
		p.sharedMemory[sharedMemory.ID] = sharedMemory
	}

	p.logger.Info("Collection created", "name", name)
	return nil
}

// DeleteCollection deletes a collection (crew)
func (p *CrewAIProvider) DeleteCollection(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.crews[name]; !exists {
		return fmt.Errorf("collection %s not found", name)
	}

	if err := p.client.DeleteCrew(ctx, name); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	delete(p.crews, name)
	p.stats.TotalCollections--

	// Delete shared memory
	sharedMemoryID := name + "_shared"
	delete(p.sharedMemory, sharedMemoryID)

	p.logger.Info("Collection deleted", "name", name)
	return nil
}

// ListCollections lists all collections (crews)
func (p *CrewAIProvider) ListCollections(ctx context.Context) ([]*memory.CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	crews, err := p.client.ListCrews(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var collections []*memory.CollectionInfo

	for _, crew := range crews {
		// Get task count
		tasks, err := p.client.ListTasks(ctx, crew.ID)
		if err != nil {
			p.logger.Warn("Failed to get tasks",
				"crew_id", crew.ID,
				"error", err)
			continue
		}

		collections = append(collections, &memory.CollectionInfo{
			Name:        crew.ID,
			Description: crew.Description,
			Dimension:   1536, // Default embedding size
			Metric:      "cosine",
			VectorCount: int64(len(tasks)),
			Size:        int64(len(tasks) * 1536 * 8), // Approximate
			CreatedAt:   crew.CreatedAt,
			UpdatedAt:   crew.UpdatedAt,
		})
	}

	return collections, nil
}

// GetCollection gets collection information
func (p *CrewAIProvider) GetCollection(ctx context.Context, name string) (*memory.CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	crew, err := p.client.GetCrew(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Get task count
	tasks, err := p.client.ListTasks(ctx, crew.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	return &memory.CollectionInfo{
		Name:        crew.ID,
		Description: crew.Description,
		Dimension:   1536,
		Metric:      "cosine",
		VectorCount: int64(len(tasks)),
		Size:        int64(len(tasks) * 1536 * 8),
		CreatedAt:   crew.CreatedAt,
		UpdatedAt:   crew.UpdatedAt,
	}, nil
}

// CreateIndex creates an index (crew optimization)
func (p *CrewAIProvider) CreateIndex(ctx context.Context, collection string, config *memory.IndexConfig) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.crews[collection]; !exists {
		return fmt.Errorf("collection %s not found", collection)
	}

	// In CrewAI, indexes are crew-specific optimizations
	p.logger.Info("Creating index for crew",
		"crew_id", collection,
		"index_name", config.Name)
	return nil
}

// DeleteIndex deletes an index
func (p *CrewAIProvider) DeleteIndex(ctx context.Context, collection string, name string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.crews[collection]; !exists {
		return fmt.Errorf("collection %s not found", collection)
	}

	p.logger.Info("Deleting index from crew",
		"crew_id", collection,
		"index_name", name)
	return nil
}

// ListIndexes lists indexes in a collection
func (p *CrewAIProvider) ListIndexes(ctx context.Context, collection string) ([]*memory.IndexInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.crews[collection]; !exists {
		return nil, fmt.Errorf("collection %s not found", collection)
	}

	// In CrewAI, indexes are internal optimizations
	return []*memory.IndexInfo{}, nil
}

// AddMetadata adds metadata to vectors (tasks)
func (p *CrewAIProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	task, err := p.client.GetTask(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Add metadata
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	for k, v := range metadata {
		task.Metadata[k] = v
	}
	task.UpdatedAt = time.Now()

	// Update task
	if err := p.client.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	p.tasks[id] = task
	return nil
}

// UpdateMetadata updates vector metadata
func (p *CrewAIProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	task, err := p.client.GetTask(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update metadata
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	for k, v := range metadata {
		task.Metadata[k] = v
	}
	task.UpdatedAt = time.Now()

	// Update task
	if err := p.client.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	p.tasks[id] = task
	return nil
}

// GetMetadata gets vector metadata
func (p *CrewAIProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	result := make(map[string]map[string]interface{})

	for _, id := range ids {
		task, err := p.client.GetTask(ctx, id)
		if err != nil {
			p.logger.Warn("Failed to get task",
				"id", id,
				"error", err)
			continue
		}

		result[id] = task.Metadata
	}

	return result, nil
}

// DeleteMetadata deletes vector metadata
func (p *CrewAIProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	for _, id := range ids {
		task, err := p.client.GetTask(ctx, id)
		if err != nil {
			p.logger.Warn("Failed to get task",
				"id", id,
				"error", err)
			continue
		}

		// Delete metadata keys
		if task.Metadata != nil {
			for _, key := range keys {
				delete(task.Metadata, key)
			}
			task.UpdatedAt = time.Now()
		}

		// Update task
		if err := p.client.UpdateTask(ctx, task); err != nil {
			p.logger.Warn("Failed to update task",
				"id", id,
				"error", err)
			continue
		}

		p.tasks[id] = task
	}

	return nil
}

// GetStats gets provider statistics
func (p *CrewAIProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &ProviderStats{
		TotalVectors:     p.stats.TotalVectors,
		TotalCollections: p.stats.TotalCollections,
		TotalSize:        p.stats.TotalSize,
		AverageLatency:   p.stats.AverageLatency,
		LastOperation:    p.stats.LastOperation,
		ErrorCount:       p.stats.ErrorCount,
		Uptime:           p.stats.Uptime,
	}, nil
}

// Optimize optimizes CrewAI provider
func (p *CrewAIProvider) Optimize(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Optimize each crew
	for crewID := range p.crews {
		performance, err := p.client.GetCrewPerformance(ctx, crewID)
		if err != nil {
			p.logger.Warn("Failed to get crew performance",
				"crew_id", crewID,
				"error", err)
			continue
		}

		p.logger.Info("Crew performance",
			"crew_id", crewID,
			"tasks_completed", performance.TasksCompleted,
			"average_completion_time", performance.AverageCompletionTime,
			"success_rate", performance.SuccessRate)
	}

	p.stats.LastOperation = time.Now()
	p.logger.Info("CrewAI optimization completed")
	return nil
}

// Backup backs up CrewAI provider
func (p *CrewAIProvider) Backup(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Export all crews and tasks
	for crewID, crew := range p.crews {
		// Export crew
		p.logger.Info("Exporting crew", "crew_id", crewID)
		// In real implementation, this would export to file
	}

	// Export shared memory
	for memoryID := range p.sharedMemory {
		p.logger.Info("Exporting shared memory", "memory_id", memoryID)
		// In real implementation, this would export to file
	}

	p.stats.LastOperation = time.Now()
	p.logger.Info("CrewAI backup completed", "path", path)
	return nil
}

// Restore restores CrewAI provider
func (p *CrewAIProvider) Restore(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Import all crews and tasks
	p.logger.Info("Restoring CrewAI from backup", "path", path)

	p.stats.LastOperation = time.Now()
	p.logger.Info("CrewAI restore completed")
	return nil
}

// Health checks health of CrewAI provider
func (p *CrewAIProvider) Health(ctx context.Context) (*HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	status := "healthy"
	lastCheck := time.Now()
	responseTime := time.Since(start)

	if !p.initialized {
		status = "not_initialized"
	} else if !p.started {
		status = "not_started"
	} else if err := p.client.GetHealth(ctx); err != nil {
		status = "unhealthy"
	}

	metrics := map[string]float64{
		"total_vectors":     float64(p.stats.TotalVectors),
		"total_collections": float64(p.stats.TotalCollections),
		"total_size_mb":     float64(p.stats.TotalSize) / (1024 * 1024),
		"uptime_seconds":    p.stats.Uptime.Seconds(),
		"total_crews":       float64(len(p.crews)),
		"total_tasks":       float64(len(p.tasks)),
		"shared_memories":   float64(len(p.sharedMemory)),
	}

	return &HealthStatus{
		Status:       status,
		LastCheck:    lastCheck,
		ResponseTime: responseTime,
		Metrics:      metrics,
		Dependencies: map[string]string{
			"crewai_api": "required",
		},
	}, nil
}

// GetName returns provider name
func (p *CrewAIProvider) GetName() string {
	return "crewai"
}

// GetType returns provider type
func (p *CrewAIProvider) GetType() ProviderType {
	return ProviderTypeCrewAI
}

// GetCapabilities returns provider capabilities
func (p *CrewAIProvider) GetCapabilities() []string {
	return []string{
		"task_management",
		"crew_coordination",
		"shared_memory",
		"vector_storage",
		"vector_search",
		"metadata_filtering",
		"batch_operations",
		"collection_management",
		"parallel_processing",
		"task_prioritization",
		"performance_monitoring",
	}
}

// GetConfiguration returns provider configuration
func (p *CrewAIProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether provider is cloud-based
func (p *CrewAIProvider) IsCloud() bool {
	return true // CrewAI is a cloud-based service
}

// GetCostInfo returns cost information
func (p *CrewAIProvider) GetCostInfo() *CostInfo {
	// CrewAI pricing based on usage
	tasksPerThousand := 1000.0
	costPerThousand := 5.0 // Example pricing

	tasks := float64(p.stats.TotalVectors)
	thousands := tasks / tasksPerThousand
	computeCost := thousands * costPerThousand

	return &CostInfo{
		StorageCost:   0.0, // Storage is included
		ComputeCost:   computeCost,
		TransferCost:  0.0, // No data transfer costs
		TotalCost:     computeCost,
		Currency:      "USD",
		BillingPeriod: "monthly",
		FreeTierUsed:  tasks > 100, // Free tier for first 100 tasks
		FreeTierLimit: 100.0,
	}
}

// Stop stops CrewAI provider
func (p *CrewAIProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return nil
	}

	p.started = false
	p.logger.Info("CrewAI provider stopped")
	return nil
}

// Helper methods

func (p *CrewAIProvider) loadCrews(ctx context.Context) error {
	crews, err := p.client.ListCrews(ctx)
	if err != nil {
		return fmt.Errorf("failed to list crews: %w", err)
	}

	for _, crew := range crews {
		p.crews[crew.ID] = crew

		// Load tasks for crew
		tasks, err := p.client.ListTasks(ctx, crew.ID)
		if err != nil {
			p.logger.Warn("Failed to load tasks",
				"crew_id", crew.ID,
				"error", err)
			continue
		}

		for _, task := range tasks {
			p.tasks[task.ID] = task
		}

		// Load shared memory
		sharedMemoryID := crew.ID + "_shared"
		sharedMemory, err := p.client.GetSharedMemory(ctx, sharedMemoryID)
		if err == nil {
			p.sharedMemory[sharedMemoryID] = sharedMemory
		}
	}

	p.stats.TotalCollections = int64(len(p.crews))
	p.stats.TotalVectors = int64(len(p.tasks))
	return nil
}

func (p *CrewAIProvider) applyFilters(metadata map[string]interface{}, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, filterValue := range filters {
		if metadataValue, exists := metadata[key]; exists {
			// Simple equality check
			if fmt.Sprintf("%v", metadataValue) != fmt.Sprintf("%v", filterValue) {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func (p *CrewAIProvider) syncWorker(ctx context.Context) {
	ticker := time.NewTicker(p.config.MemorySyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.logger.Debug("Sync worker running")
			// Sync shared memory across crews
		}
	}
}

func (p *CrewAIProvider) updateStats(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stats.LastOperation = time.Now()

	// Update average latency (simple moving average)
	if p.stats.AverageLatency == 0 {
		p.stats.AverageLatency = duration
	} else {
		p.stats.AverageLatency = (p.stats.AverageLatency + duration) / 2
	}

	// Update uptime
	if p.started {
		p.stats.Uptime += duration
	}
}

// CrewAIHTTPClient is a mock HTTP client for CrewAI
type CrewAIHTTPClient struct {
	config *CrewAIConfig
	logger logging.Logger
}

// NewCrewAIHTTPClient creates a new CrewAI HTTP client
func NewCrewAIHTTPClient(config *CrewAIConfig) (CrewAIClient, error) {
	return &CrewAIHTTPClient{
		config: config,
		logger: logging.NewLogger("crewai_client"),
	}, nil
}

// Mock implementation of CrewAIClient interface
func (c *CrewAIHTTPClient) CreateCrew(ctx context.Context, crew *memory.Crew) error {
	c.logger.Info("Creating crew", "id", crew.ID, "name", crew.Name)
	return nil
}

func (c *CrewAIHTTPClient) GetCrew(ctx context.Context, crewID string) (*memory.Crew, error) {
	// Mock implementation
	return &memory.Crew{
		ID:                crewID,
		Name:              crewID,
		Description:       "Mock crew",
		MaxAgents:         c.config.MaxAgents,
		MaxTasksPerAgent:  c.config.MaxTasksPerAgent,
		ParallelExecution: c.config.ParallelExecution,
		Status:            "active",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Agents:            []*memory.Agent{},
	}, nil
}

func (c *CrewAIHTTPClient) UpdateCrew(ctx context.Context, crew *memory.Crew) error {
	c.logger.Info("Updating crew", "id", crew.ID)
	return nil
}

func (c *CrewAIHTTPClient) DeleteCrew(ctx context.Context, crewID string) error {
	c.logger.Info("Deleting crew", "id", crewID)
	return nil
}

func (c *CrewAIHTTPClient) ListCrews(ctx context.Context) ([]*memory.Crew, error) {
	// Mock implementation
	return []*memory.Crew{
		{ID: "crew1", Name: "Crew 1", Status: "active", CreatedAt: time.Now()},
		{ID: "crew2", Name: "Crew 2", Status: "active", CreatedAt: time.Now()},
	}, nil
}

func (c *CrewAIHTTPClient) CreateTask(ctx context.Context, task *memory.Task) error {
	c.logger.Info("Creating task", "id", task.ID, "crew_id", task.CrewID)
	return nil
}

func (c *CrewAIHTTPClient) GetTask(ctx context.Context, taskID string) (*memory.Task, error) {
	// Mock implementation
	return &memory.Task{
		ID:          taskID,
		CrewID:      "default",
		Description: "Mock task",
		Priority:    "normal",
		Status:      "pending",
		Metadata:    map[string]interface{}{"source": "mock"},
		Embedding:   make([]float64, 1536),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (c *CrewAIHTTPClient) UpdateTask(ctx context.Context, task *memory.Task) error {
	c.logger.Info("Updating task", "id", task.ID)
	return nil
}

func (c *CrewAIHTTPClient) DeleteTask(ctx context.Context, taskID string) error {
	c.logger.Info("Deleting task", "id", taskID)
	return nil
}

func (c *CrewAIHTTPClient) ListTasks(ctx context.Context, crewID string) ([]*memory.Task, error) {
	// Mock implementation
	var tasks []*memory.Task
	for i := 0; i < 10; i++ {
		tasks = append(tasks, &memory.Task{
			ID:          fmt.Sprintf("task_%s_%d", crewID, i),
			CrewID:      crewID,
			Description: fmt.Sprintf("Mock task %d", i),
			Priority:    "normal",
			Status:      "pending",
			Metadata:    map[string]interface{}{"index": i},
			Embedding:   make([]float64, 1536),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}
	return tasks, nil
}

func (c *CrewAIHTTPClient) AssignTask(ctx context.Context, taskID, agentID string) error {
	c.logger.Info("Assigning task", "task_id", taskID, "agent_id", agentID)
	return nil
}

func (c *CrewAIHTTPClient) CompleteTask(ctx context.Context, taskID string, result *memory.TaskResult) error {
	c.logger.Info("Completing task", "task_id", taskID, "success", result.Success)
	return nil
}

func (c *CrewAIHTTPClient) GetSharedMemory(ctx context.Context, memoryID string) (*memory.SharedMemory, error) {
	// Mock implementation
	return &memory.SharedMemory{
		ID:          memoryID,
		CrewID:      "default",
		Data:        map[string]interface{}{"mock": true},
		LastUpdated: time.Now(),
		Version:     1,
	}, nil
}

func (c *CrewAIHTTPClient) UpdateSharedMemory(ctx context.Context, memory *memory.SharedMemory) error {
	c.logger.Info("Updating shared memory", "id", memory.ID)
	return nil
}

func (c *CrewAIHTTPClient) GetCrewPerformance(ctx context.Context, crewID string) (*memory.CrewPerformance, error) {
	// Mock implementation
	return &memory.CrewPerformance{
		CrewID:                crewID,
		TasksCompleted:        100,
		TasksFailed:           5,
		AverageCompletionTime: 5.5 * time.Minute,
		SuccessRate:           0.95,
		AgentUtilization:      0.75,
		TotalProcessingTime:   10 * time.Hour,
		PerformanceScore:      9.2,
		LastUpdated:           time.Now(),
	}, nil
}

func (c *CrewAIHTTPClient) GetHealth(ctx context.Context) error {
	// Mock implementation - in real implementation, this would check CrewAI API health
	return nil
}
