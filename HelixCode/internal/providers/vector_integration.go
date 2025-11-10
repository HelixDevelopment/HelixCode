package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory/providers"
)

// VectorConfig contains configuration for vector integration
type VectorConfig struct {
	DefaultProvider string                                     `json:"default_provider"`
	Providers       map[string]*providers.SingleProviderConfig `json:"providers"`
	LoadBalancing   providers.LoadBalanceType                  `json:"load_balancing"`
	FailoverEnabled bool                                       `json:"failover_enabled"`
}

// VectorIntegration provides vector database integration for all providers
type VectorIntegration struct {
	mu        sync.RWMutex
	registry  *providers.ProviderRegistry
	manager   *providers.ProviderManager
	logger    *logging.Logger
	config    *VectorConfig
	providers map[string]providers.VectorProvider
}

// NewVectorIntegration creates a new vector integration instance
func NewVectorIntegration(config *VectorConfig) *VectorIntegration {
	return &VectorIntegration{
		registry:  providers.GetRegistry(),
		logger:    logging.NewLogger(logging.INFO),
		config:    config,
		providers: make(map[string]providers.VectorProvider),
	}
}

// Initialize initializes vector integration
func (vi *VectorIntegration) Initialize(ctx context.Context) error {
	vi.mu.Lock()
	defer vi.mu.Unlock()

	vi.logger.Info("Initializing vector integration: default_provider=%s, providers_count=%d", vi.config.DefaultProvider, len(vi.config.Providers))

	// Create and initialize all providers
	providerConfigs := make(map[string]*providers.SingleProviderConfig)

	for name, providerConfig := range vi.config.Providers {
		if !providerConfig.Enabled {
			vi.logger.Info("Skipping disabled provider: name=%s", name)
			continue
		}

		providerConfigs[name] = &providers.SingleProviderConfig{
			Name:     name,
			Type:     providerConfig.Type,
			Config:   providerConfig.Config,
			Priority: 1,
			Enabled:  true,
		}
	}

	// Create provider manager
	managerConfig := &providers.ManagerConfig{
		Providers:             providerConfigs,
		DefaultProvider:       vi.config.DefaultProvider,
		LoadBalancing:         vi.config.LoadBalancing,
		FailoverEnabled:       vi.config.FailoverEnabled,
		RetryAttempts:         3,
		RetryBackoff:          1000,
		HealthCheckInterval:   60000,
		PerformanceMonitoring: true,
		CostTracking:          true,
		BackupEnabled:         false,
	}

	var err error
	vi.manager, err = providers.NewProviderManager(managerConfig)
	if err != nil {
		return fmt.Errorf("failed to create provider manager: %w", err)
	}

	// Initialize provider manager
	if err := vi.manager.Initialize(ctx, nil); err != nil {
		return fmt.Errorf("failed to initialize provider manager: %w", err)
	}

	// Start provider manager
	if err := vi.manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start provider manager: %w", err)
	}

	vi.logger.Info("Vector integration initialized successfully")
	return nil
}

// StoreVector stores a vector in default provider
func (vi *VectorIntegration) StoreVector(ctx context.Context, vector *VectorData) error {
	return vi.StoreVectorInProvider(ctx, vi.config.DefaultProvider, vector)
}

// StoreVectorInProvider stores a vector in a specific provider
func (vi *VectorIntegration) StoreVectorInProvider(ctx context.Context, providerName string, vector *VectorData) error {
	// Convert to provider format
	providerVector := vi.convertToProviderVector(vector)

	// Store using provider manager
	return vi.manager.Store(ctx, []*providers.VectorData{providerVector})
}

// RetrieveVector retrieves a vector by ID
func (vi *VectorIntegration) RetrieveVector(ctx context.Context, id string) (*VectorData, error) {
	// Retrieve using provider manager
	results, err := vi.manager.Retrieve(ctx, []string{id})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("vector not found: %s", id)
	}

	// Convert from provider format
	return vi.convertFromProviderVector(results[0]), nil
}

// SearchVectors searches for similar vectors
func (vi *VectorIntegration) SearchVectors(ctx context.Context, query *VectorSearchQuery) ([]*VectorSearchResult, error) {
	// Convert to provider format
	providerQuery := vi.convertToProviderQuery(query)

	// Search using provider manager
	result, err := vi.manager.Search(ctx, providerQuery)
	if err != nil {
		return nil, err
	}

	// Convert results
	var searchResults []*VectorSearchResult
	for _, item := range result.Results {
		searchResults = append(searchResults, &VectorSearchResult{
			ID:       item.ID,
			Vector:   item.Vector,
			Metadata: item.Metadata,
			Score:    item.Score,
			Distance: 1 - item.Score,
		})
	}

	return searchResults, nil
}

// FindSimilarVectors finds vectors similar to given vector
func (vi *VectorIntegration) FindSimilarVectors(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSearchResult, error) {
	// Find similar using provider manager
	results, err := vi.manager.FindSimilar(ctx, embedding, k, filters)
	if err != nil {
		return nil, err
	}

	// Convert results
	var searchResults []*VectorSearchResult
	for _, item := range results {
		searchResults = append(searchResults, &VectorSearchResult{
			ID:       item.ID,
			Vector:   item.Vector,
			Metadata: item.Metadata,
			Score:    item.Score,
			Distance: 1 - item.Score,
		})
	}

	return searchResults, nil
}

// CreateVectorIndex creates a vector index
func (vi *VectorIntegration) CreateVectorIndex(ctx context.Context, indexName string, config *VectorIndexConfig) error {
	// Convert to provider format
	providerConfig := &providers.CollectionConfig{
		Name:        indexName,
		Dimension:   config.Dimension,
		Metric:      config.Metric,
		Description: config.Description,
	}

	return vi.manager.CreateCollection(ctx, indexName, providerConfig)
}

// DeleteVectorIndex deletes a vector index
func (vi *VectorIntegration) DeleteVectorIndex(ctx context.Context, indexName string) error {
	return vi.manager.DeleteCollection(ctx, indexName)
}

// ListVectorIndexes lists all vector indexes
func (vi *VectorIntegration) ListVectorIndexes(ctx context.Context) ([]*VectorIndexInfo, error) {
	// List collections using provider manager
	collections, err := vi.manager.ListCollections(ctx)
	if err != nil {
		return nil, err
	}

	// Convert results
	var indexes []*VectorIndexInfo
	for _, collectionInfo := range collections {
		indexes = append(indexes, &VectorIndexInfo{
			Name:        collectionInfo.Name,
			Description: "",
			Dimension:   collectionInfo.Dimension,
			Metric:      collectionInfo.Metric,
			VectorCount: collectionInfo.VectorCount,
			Size:        collectionInfo.Size,
			CreatedAt:   collectionInfo.CreatedAt,
			UpdatedAt:   collectionInfo.UpdatedAt,
		})
	}

	return indexes, nil
}

// GetVectorStats returns statistics about vector storage
func (vi *VectorIntegration) GetVectorStats(ctx context.Context) (*VectorStats, error) {
	// Get stats from provider manager
	stats, err := vi.manager.GetStats()
	if err != nil {
		return nil, err
	}

	cost := 0.0
	if stats.CostInfo != nil {
		cost = stats.CostInfo.TotalCost
	}

	return &VectorStats{
		TotalVectors:   stats.TotalVectors,
		TotalIndexes:   stats.TotalCollections,
		TotalSize:      stats.TotalSize,
		AverageLatency: stats.AverageLatency,
		LastOperation:  stats.LastOperation,
		ErrorCount:     stats.ErrorCount,
		Uptime:         stats.Uptime,
		Cost:           cost,
	}, nil
}

// HealthCheck performs a health check on all vector providers
func (vi *VectorIntegration) HealthCheck(ctx context.Context) (*VectorHealthStatus, error) {
	// Get health status from provider manager
	healthStatuses, err := vi.manager.Health(ctx)
	if err != nil {
		return nil, err
	}

	// Aggregate health status
	status := "healthy"
	unhealthyCount := 0
	for providerName, health := range healthStatuses {
		if health.Status != "healthy" {
			status = "degraded"
			unhealthyCount++
			vi.logger.Warn("Provider unhealthy: provider=%s, status=%s", providerName, health.Status)
		}
	}

	if unhealthyCount == len(healthStatuses) {
		status = "unhealthy"
	}

	return &VectorHealthStatus{
		Status:             status,
		TotalProviders:     len(healthStatuses),
		HealthyProviders:   len(healthStatuses) - unhealthyCount,
		UnhealthyProviders: unhealthyCount,
		ProviderStatuses:   healthStatuses,
		LastCheck:          time.Now(),
	}, nil
}

// OptimizeIndexes optimizes all vector indexes
func (vi *VectorIntegration) OptimizeIndexes(ctx context.Context) error {
	return vi.manager.Optimize(ctx)
}

// BackupVectors backs up all vector data
func (vi *VectorIntegration) BackupVectors(ctx context.Context, backupPath string) error {
	return vi.manager.Backup(ctx, backupPath)
}

// RestoreVectors restores vector data from backup
func (vi *VectorIntegration) RestoreVectors(ctx context.Context, backupPath string) error {
	return vi.manager.Restore(ctx, backupPath)
}

// convertToProviderVector converts to provider vector format
func (vi *VectorIntegration) convertToProviderVector(vector *VectorData) *providers.VectorData {
	return &providers.VectorData{
		ID:         vector.ID,
		Vector:     vector.Embedding,
		Metadata:   vector.Metadata,
		Collection: vector.IndexName,
		Timestamp:  vector.CreatedAt,
	}
}

// convertFromProviderVector converts from provider vector format
func (vi *VectorIntegration) convertFromProviderVector(vector *providers.VectorData) *VectorData {
	return &VectorData{
		ID:        vector.ID,
		Embedding: vector.Vector,
		Metadata:  vector.Metadata,
		IndexName: vector.Collection,
		CreatedAt: vector.Timestamp,
	}
}

// convertToProviderQuery converts to provider query format
func (vi *VectorIntegration) convertToProviderQuery(query *VectorSearchQuery) *providers.VectorQuery {
	return &providers.VectorQuery{
		Vector:     query.Embedding,
		Collection: query.IndexName,
		TopK:       query.K,
		Threshold:  query.Threshold,
		Filters:    query.Filters,
	}
}

// Stop stops vector integration
func (vi *VectorIntegration) Stop(ctx context.Context) error {
	vi.mu.Lock()
	defer vi.mu.Unlock()

	vi.logger.Info("Stopping vector integration")

	if vi.manager != nil {
		if err := vi.manager.Stop(ctx); err != nil {
			vi.logger.Warn("Failed to stop provider manager: %v", err)
		}
	}

	vi.logger.Info("Vector integration stopped")
	return nil
}

// VectorData represents a vector with metadata
type VectorData struct {
	ID        string                 `json:"id"`
	Embedding []float64              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
	IndexName string                 `json:"index_name"`
	CreatedAt time.Time              `json:"created_at"`
}

// VectorSearchQuery represents a vector search query
type VectorSearchQuery struct {
	Embedding []float64              `json:"embedding"`
	IndexName string                 `json:"index_name"`
	K         int                    `json:"k"`
	Threshold float64                `json:"threshold"`
	Filters   map[string]interface{} `json:"filters"`
	Metric    string                 `json:"metric"`
}

// VectorSearchResult represents a vector search result
type VectorSearchResult struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float64                `json:"score"`
	Distance float64                `json:"distance"`
}

// VectorIndexConfig contains configuration for a vector index
type VectorIndexConfig struct {
	Dimension   int    `json:"dimension"`
	Metric      string `json:"metric"`
	Description string `json:"description"`
}

// VectorIndexInfo contains information about a vector index
type VectorIndexInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Dimension   int       `json:"dimension"`
	Metric      string    `json:"metric"`
	VectorCount int64     `json:"vector_count"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// VectorStats contains statistics about vector storage
type VectorStats struct {
	TotalVectors   int64         `json:"total_vectors"`
	TotalIndexes   int64         `json:"total_indexes"`
	TotalSize      int64         `json:"total_size"`
	AverageLatency time.Duration `json:"average_latency"`
	LastOperation  time.Time     `json:"last_operation"`
	ErrorCount     int64         `json:"error_count"`
	Uptime         time.Duration `json:"uptime"`
	Cost           float64       `json:"cost"`
}

// VectorHealthStatus contains health status of vector providers
type VectorHealthStatus struct {
	Status             string                             `json:"status"`
	TotalProviders     int                                `json:"total_providers"`
	HealthyProviders   int                                `json:"healthy_providers"`
	UnhealthyProviders int                                `json:"unhealthy_providers"`
	ProviderStatuses   map[string]*providers.HealthStatus `json:"provider_statuses"`
	LastCheck          time.Time                          `json:"last_check"`
}
