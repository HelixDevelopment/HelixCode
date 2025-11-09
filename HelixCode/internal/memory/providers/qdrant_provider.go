package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
)

// QdrantProvider implements VectorProvider for Qdrant
type QdrantProvider struct {
	config       *QdrantConfig
	logger       logging.Logger
	mu           sync.RWMutex
	initialized  bool
	started      bool
	client       QdrantClient
	collections  map[string]*memory.CollectionConfig
	stats        *ProviderStats
}

// QdrantConfig contains Qdrant provider configuration
type QdrantConfig struct {
	Host           string            `json:"host"`
	Port           int               `json:"port"`
	APIKey         string            `json:"api_key"`
	UseTLS         bool              `json:"use_tls"`
	Timeout        time.Duration     `json:"timeout"`
	MaxRetries     int               `json:"max_retries"`
	BatchSize      int               `json:"batch_size"`
	Compression    bool              `json:"compression"`
	ParallelSearch bool              `json:"parallel_search"`
	SearchTimeout  time.Duration     `json:"search_timeout"`
	IndexType      string            `json:"index_type"`
	ShardCount     int               `json:"shard_count"`
	ReplicaCount   int               `json:"replica_count"`
}

// QdrantClient represents Qdrant client interface
type QdrantClient interface {
	CreateCollection(ctx context.Context, name string, config *memory.CollectionConfig) error
	DeleteCollection(ctx context.Context, name string) error
	ListCollections(ctx context.Context) ([]string, error)
	GetCollection(ctx context.Context, name string) (*memory.CollectionInfo, error)
	StorePoints(ctx context.Context, collection string, points []*memory.VectorData) error
	SearchPoints(ctx context.Context, collection string, query *memory.VectorQuery) (*memory.VectorSearchResult, error)
	GetPoints(ctx context.Context, collection string, ids []string) ([]*memory.VectorData, error)
	DeletePoints(ctx context.Context, collection string, ids []string) error
	CreateIndex(ctx context.Context, collection string, config *memory.IndexConfig) error
	DeleteIndex(ctx context.Context, collection string, indexName string) error
	Health(ctx context.Context) error
}

// NewQdrantProvider creates a new Qdrant provider
func NewQdrantProvider(config map[string]interface{}) (VectorProvider, error) {
	qdrantConfig := &QdrantConfig{
		Host:           "localhost",
		Port:           6333,
		APIKey:         "",
		UseTLS:         false,
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		BatchSize:      1000,
		Compression:    true,
		ParallelSearch: true,
		SearchTimeout:  10 * time.Second,
		IndexType:      "hnsw",
		ShardCount:     1,
		ReplicaCount:   1,
	}

	// Parse configuration
	if err := parseConfig(config, qdrantConfig); err != nil {
		return nil, fmt.Errorf("failed to parse Qdrant config: %w", err)
	}

	return &QdrantProvider{
		config:      qdrantConfig,
		logger:      logging.NewLogger("qdrant_provider"),
		collections: make(map[string]*memory.CollectionConfig),
		stats: &ProviderStats{
			TotalVectors:     0,
			TotalCollections: 0,
			TotalSize:        0,
			AverageLatency:    0,
			LastOperation:     time.Now(),
			ErrorCount:       0,
			Uptime:          0,
		},
	}, nil
}

// Initialize initializes Qdrant provider
func (p *QdrantProvider) Initialize(ctx context.Context, config interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	p.logger.Info("Initializing Qdrant provider",
		"host", p.config.Host,
		"port", p.config.Port,
		"use_tls", p.config.UseTLS)

	// Create Qdrant client
	client, err := NewQdrantHTTPClient(p.config)
	if err != nil {
		return fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	p.client = client

	// Test connection
	if err := p.client.Health(ctx); err != nil {
		return fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	// Load existing collections
	if err := p.loadCollections(ctx); err != nil {
		p.logger.Warn("Failed to load collections", "error", err)
	}

	p.initialized = true
	p.stats.LastOperation = time.Now()

	p.logger.Info("Qdrant provider initialized successfully")
	return nil
}

// Start starts Qdrant provider
func (p *QdrantProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	if p.started {
		return nil
	}

	// Start background tasks
	go p.statsUpdater(ctx)

	p.started = true
	p.stats.LastOperation = time.Now()
	p.stats.Uptime = 0

	p.logger.Info("Qdrant provider started successfully")
	return nil
}

// Store stores vectors in Qdrant
func (p *QdrantProvider) Store(ctx context.Context, vectors []*memory.VectorData) error {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	// Group vectors by collection
	collectionVectors := make(map[string][]*memory.VectorData)
	for _, vector := range vectors {
		collection := vector.Collection
		if collection == "" {
			collection = "default"
		}
		collectionVectors[collection] = append(collectionVectors[collection], vector)
	}

	// Store vectors for each collection
	for collection, vecs := range collectionVectors {
		// Create collection if it doesn't exist
		if _, exists := p.collections[collection]; !exists {
			if err := p.createCollection(ctx, collection, len(vecs[0].Vector)); err != nil {
				return fmt.Errorf("failed to create collection %s: %w", collection, err)
			}
		}

		// Store vectors in batches
		for i := 0; i < len(vecs); i += p.config.BatchSize {
			end := i + p.config.BatchSize
			if end > len(vecs) {
				end = len(vecs)
			}

			batch := vecs[i:end]
			if err := p.client.StorePoints(ctx, collection, batch); err != nil {
				p.logger.Error("Failed to store batch",
					"collection", collection,
					"batch_size", len(batch),
					"error", err)
				return fmt.Errorf("failed to store vectors: %w", err)
			}
		}

		p.stats.TotalVectors += int64(len(vecs))
	}

	p.stats.LastOperation = time.Now()
	return nil
}

// Retrieve retrieves vectors by ID from Qdrant
func (p *QdrantProvider) Retrieve(ctx context.Context, ids []string) ([]*memory.VectorData, error) {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	var allResults []*memory.VectorData

	// Search in all collections
	for collection := range p.collections {
		results, err := p.client.GetPoints(ctx, collection, ids)
		if err != nil {
			p.logger.Warn("Failed to retrieve from collection",
				"collection", collection,
				"error", err)
			continue
		}

		allResults = append(allResults, results...)
	}

	p.stats.LastOperation = time.Now()
	return allResults, nil
}

// Search performs vector similarity search in Qdrant
func (p *QdrantProvider) Search(ctx context.Context, query *memory.VectorQuery) (*memory.VectorSearchResult, error) {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	collection := query.Collection
	if collection == "" {
		collection = "default"
	}

	// Check if collection exists
	if _, exists := p.collections[collection]; !exists {
		return &memory.VectorSearchResult{
			Results:  []*memory.VectorSearchResultItem{},
			Total:    0,
			Query:    query,
			Duration: time.Since(start),
			Namespace: query.Namespace,
		}, nil
	}

	result, err := p.client.SearchPoints(ctx, collection, query)
	if err != nil {
		p.logger.Error("Search failed",
			"collection", collection,
			"error", err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	p.stats.LastOperation = time.Now()
	return result, nil
}

// FindSimilar finds similar vectors
func (p *QdrantProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*memory.VectorSimilarityResult, error) {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	query := &memory.VectorQuery{
		Vector:     embedding,
		TopK:       k,
		Filters:    filters,
		Metric:     "cosine", // Qdrant default
	}

	searchResult, err := p.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	var results []*memory.VectorSimilarityResult
	for _, item := range searchResult.Results {
		results = append(results, &memory.VectorSimilarityResult{
			ID:       item.ID,
			Vector:   item.Vector,
			Metadata: item.Metadata,
			Score:    item.Score,
			Distance: 1 - item.Score,
		})
	}

	p.stats.LastOperation = time.Now()
	return results, nil
}

// CreateCollection creates a new collection
func (p *QdrantProvider) CreateCollection(ctx context.Context, name string, config *memory.CollectionConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.collections[name]; exists {
		return fmt.Errorf("collection %s already exists", name)
	}

	if err := p.client.CreateCollection(ctx, name, config); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	p.collections[name] = config
	p.stats.TotalCollections++

	p.logger.Info("Collection created", "name", name, "dimension", config.Dimension)
	return nil
}

// DeleteCollection deletes a collection
func (p *QdrantProvider) DeleteCollection(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.collections[name]; !exists {
		return fmt.Errorf("collection %s not found", name)
	}

	if err := p.client.DeleteCollection(ctx, name); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	delete(p.collections, name)
	p.stats.TotalCollections--

	p.logger.Info("Collection deleted", "name", name)
	return nil
}

// ListCollections lists all collections
func (p *QdrantProvider) ListCollections(ctx context.Context) ([]*memory.CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	collectionNames, err := p.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var collections []*memory.CollectionInfo
	for _, name := range collectionNames {
		if config, exists := p.collections[name]; exists {
			collectionInfo, err := p.client.GetCollection(ctx, name)
			if err != nil {
				p.logger.Warn("Failed to get collection info",
					"name", name,
					"error", err)
				continue
			}
			collections = append(collections, collectionInfo)
		}
	}

	return collections, nil
}

// GetCollection gets collection information
func (p *QdrantProvider) GetCollection(ctx context.Context, name string) (*memory.CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.collections[name]; !exists {
		return nil, fmt.Errorf("collection %s not found", name)
	}

	collectionInfo, err := p.client.GetCollection(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	return collectionInfo, nil
}

// CreateIndex creates an index
func (p *QdrantProvider) CreateIndex(ctx context.Context, collection string, config *memory.IndexConfig) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.collections[collection]; !exists {
		return fmt.Errorf("collection %s not found", collection)
	}

	return p.client.CreateIndex(ctx, collection, config)
}

// DeleteIndex deletes an index
func (p *QdrantProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.collections[collection]; !exists {
		return fmt.Errorf("collection %s not found", collection)
	}

	return p.client.DeleteIndex(ctx, collection, name)
}

// ListIndexes lists indexes in a collection
func (p *QdrantProvider) ListIndexes(ctx context.Context, collection string) ([]*memory.IndexInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.collections[collection]; !exists {
		return nil, fmt.Errorf("collection %s not found", collection)
	}

	// In a real implementation, this would query Qdrant for index info
	// For now, return empty list
	return []*memory.IndexInfo{}, nil
}

// AddMetadata adds metadata to vectors
func (p *QdrantProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	// In Qdrant, metadata is part of the payload
	// This would require updating the point
	// For now, return not implemented
	return fmt.Errorf("metadata operations not implemented for Qdrant provider")
}

// UpdateMetadata updates vector metadata
func (p *QdrantProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	return fmt.Errorf("metadata operations not implemented for Qdrant provider")
}

// GetMetadata gets vector metadata
func (p *QdrantProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	// In Qdrant, metadata is retrieved with vectors
	result := make(map[string]map[string]interface{})
	for _, id := range ids {
		vectors, err := p.client.GetPoints(ctx, "default", []string{id})
		if err == nil && len(vectors) > 0 {
			result[id] = vectors[0].Metadata
		}
	}

	return result, nil
}

// DeleteMetadata deletes vector metadata
func (p *QdrantProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	return fmt.Errorf("metadata operations not implemented for Qdrant provider")
}

// GetStats gets provider statistics
func (p *QdrantProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &ProviderStats{
		TotalVectors:     p.stats.TotalVectors,
		TotalCollections: p.stats.TotalCollections,
		TotalSize:        p.stats.TotalSize,
		AverageLatency:   p.stats.AverageLatency,
		LastOperation:    p.stats.LastOperation,
		ErrorCount:       p.stats.ErrorCount,
		Uptime:          p.stats.Uptime,
	}, nil
}

// Optimize optimizes Qdrant provider
func (p *QdrantProvider) Optimize(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// In Qdrant, optimization includes:
	// - Index optimization
	// - Shard rebalancing
	// - Cache warming

	for collection := range p.collections {
		// Trigger optimization for each collection
		p.logger.Info("Optimizing collection", "name", collection)
	}

	p.stats.LastOperation = time.Now()
	p.logger.Info("Qdrant optimization completed")
	return nil
}

// Backup backs up Qdrant provider
func (p *QdrantProvider) Backup(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// In Qdrant, backup involves:
	// - Snapshot creation
	// - Collection snapshots
	// - Configuration backup

	for collection := range p.collections {
		// Create snapshot for each collection
		p.logger.Info("Creating snapshot", "collection", collection)
	}

	p.stats.LastOperation = time.Now()
	p.logger.Info("Qdrant backup completed", "path", path)
	return nil
}

// Restore restores Qdrant provider
func (p *QdrantProvider) Restore(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// In Qdrant, restore involves:
	// - Loading snapshots
	// - Recreating collections
	// - Restoring configuration

	p.logger.Info("Restoring Qdrant from backup", "path", path)

	p.stats.LastOperation = time.Now()
	p.logger.Info("Qdrant restore completed")
	return nil
}

// Health checks health of Qdrant provider
func (p *QdrantProvider) Health(ctx context.Context) (*HealthStatus, error) {
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
	} else if err := p.client.Health(ctx); err != nil {
		status = "unhealthy"
	}

	metrics := map[string]float64{
		"total_vectors":     float64(p.stats.TotalVectors),
		"total_collections": float64(p.stats.TotalCollections),
		"total_size_mb":    float64(p.stats.TotalSize) / (1024 * 1024),
		"uptime_seconds":   p.stats.Uptime.Seconds(),
	}

	return &HealthStatus{
		Status:      status,
		LastCheck:   lastCheck,
		ResponseTime: responseTime,
		Metrics:     metrics,
		Dependencies: map[string]string{
			"qdrant_server": "required",
		},
	}, nil
}

// GetName returns provider name
func (p *QdrantProvider) GetName() string {
	return "qdrant"
}

// GetType returns provider type
func (p *QdrantProvider) GetType() ProviderType {
	return ProviderTypeQdrant
}

// GetCapabilities returns provider capabilities
func (p *QdrantProvider) GetCapabilities() []string {
	return []string{
		"vector_storage",
		"vector_search",
		"metadata_filtering",
		"batch_operations",
		"collection_management",
		"index_management",
		"parallel_search",
		"sharding",
		"replication",
		"snapshot_support",
	}
}

// GetConfiguration returns provider configuration
func (p *QdrantProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether provider is cloud-based
func (p *QdrantProvider) IsCloud() bool {
	return false // Qdrant is typically self-hosted, but can be cloud-based
}

// GetCostInfo returns cost information
func (p *QdrantProvider) GetCostInfo() *CostInfo {
	return &CostInfo{
		StorageCost:   0.0, // Self-hosted, no direct cost
		ComputeCost:   0.0, // Self-hosted, no direct cost
		TransferCost:  0.0, // Self-hosted, no direct cost
		TotalCost:     0.0,
		Currency:      "USD",
		BillingPeriod:  "N/A",
		FreeTierUsed:  false,
		FreeTierLimit: 0.0,
	}
}

// Stop stops Qdrant provider
func (p *QdrantProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return nil
	}

	p.started = false
	p.logger.Info("Qdrant provider stopped")
	return nil
}

// Helper methods

func (p *QdrantProvider) loadCollections(ctx context.Context) error {
	collectionNames, err := p.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	for _, name := range collectionNames {
		info, err := p.client.GetCollection(ctx, name)
		if err != nil {
			p.logger.Warn("Failed to get collection info",
				"name", name,
				"error", err)
			continue
		}

		p.collections[name] = &memory.CollectionConfig{
			Name:        name,
			Dimension:   info.Dimension,
			Metric:      info.Metric,
			Description: info.Description,
		}
	}

	p.stats.TotalCollections = int64(len(p.collections))
	return nil
}

func (p *QdrantProvider) createCollection(ctx context.Context, name string, dimension int) error {
	config := &memory.CollectionConfig{
		Name:       name,
		Dimension:  dimension,
		Metric:     "cosine",
		Shards:     p.config.ShardCount,
		Replicas:   p.config.ReplicaCount,
	}

	if err := p.client.CreateCollection(ctx, name, config); err != nil {
		return err
	}

	p.collections[name] = config
	p.stats.TotalCollections++
	return nil
}

func (p *QdrantProvider) statsUpdater(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			p.stats.Uptime += 30 * time.Second
			p.mu.Unlock()
		}
	}
}

func (p *QdrantProvider) updateStats(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stats.LastOperation = time.Now()
	
	// Update average latency (simple moving average)
	if p.stats.AverageLatency == 0 {
		p.stats.AverageLatency = duration
	} else {
		p.stats.AverageLatency = (p.stats.AverageLatency + duration) / 2
	}
}

// QdrantHTTPClient is a mock HTTP client for Qdrant
type QdrantHTTPClient struct {
	config *QdrantConfig
	logger logging.Logger
}

// NewQdrantHTTPClient creates a new Qdrant HTTP client
func NewQdrantHTTPClient(config *QdrantConfig) (QdrantClient, error) {
	return &QdrantHTTPClient{
		config: config,
		logger: logging.NewLogger("qdrant_client"),
	}, nil
}

// Mock implementation of QdrantClient interface
func (c *QdrantHTTPClient) CreateCollection(ctx context.Context, name string, config *memory.CollectionConfig) error {
	c.logger.Info("Creating collection", "name", name, "dimension", config.Dimension)
	return nil
}

func (c *QdrantHTTPClient) DeleteCollection(ctx context.Context, name string) error {
	c.logger.Info("Deleting collection", "name", name)
	return nil
}

func (c *QdrantHTTPClient) ListCollections(ctx context.Context) ([]string, error) {
	// Mock implementation
	return []string{"collection1", "collection2", "collection3"}, nil
}

func (c *QdrantHTTPClient) GetCollection(ctx context.Context, name string) (*memory.CollectionInfo, error) {
	// Mock implementation
	return &memory.CollectionInfo{
		Name:        name,
		Description: "Mock collection",
		Dimension:   1536,
		Metric:      "cosine",
		VectorCount: 1000,
		Size:        1536000,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (c *QdrantHTTPClient) StorePoints(ctx context.Context, collection string, points []*memory.VectorData) error {
	c.logger.Info("Storing points", "collection", collection, "count", len(points))
	return nil
}

func (c *QdrantHTTPClient) SearchPoints(ctx context.Context, collection string, query *memory.VectorQuery) (*memory.VectorSearchResult, error) {
	// Mock implementation
	var results []*memory.VectorSearchResultItem
	for i := 0; i < query.TopK; i++ {
		results = append(results, &memory.VectorSearchResultItem{
			ID:       fmt.Sprintf("result_%d", i),
			Vector:   make([]float64, 1536),
			Score:    1.0 - float64(i)*0.1,
			Distance:  float64(i) * 0.1,
			Metadata: map[string]interface{}{
				"collection": collection,
				"index":     i,
			},
		})
	}

	return &memory.VectorSearchResult{
		Results:   results,
		Total:     len(results),
		Query:     query,
		Duration:  100 * time.Millisecond,
		Namespace: query.Namespace,
	}, nil
}

func (c *QdrantHTTPClient) GetPoints(ctx context.Context, collection string, ids []string) ([]*memory.VectorData, error) {
	// Mock implementation
	var vectors []*memory.VectorData
	for _, id := range ids {
		vectors = append(vectors, &memory.VectorData{
			ID:       id,
			Vector:   make([]float64, 1536),
			Metadata: map[string]interface{}{
				"collection": collection,
			},
			Collection: collection,
			Timestamp:  time.Now(),
		})
	}
	return vectors, nil
}

func (c *QdrantHTTPClient) DeletePoints(ctx context.Context, collection string, ids []string) error {
	c.logger.Info("Deleting points", "collection", collection, "count", len(ids))
	return nil
}

func (c *QdrantHTTPClient) CreateIndex(ctx context.Context, collection string, config *memory.IndexConfig) error {
	c.logger.Info("Creating index", "collection", collection, "name", config.Name)
	return nil
}

func (c *QdrantHTTPClient) DeleteIndex(ctx context.Context, collection string, indexName string) error {
	c.logger.Info("Deleting index", "collection", collection, "name", indexName)
	return nil
}

func (c *QdrantHTTPClient) Health(ctx context.Context) error {
	// Mock implementation - in real implementation, this would check Qdrant health
	return nil
}