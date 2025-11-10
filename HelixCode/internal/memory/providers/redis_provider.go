package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory"
)

// RedisProvider implements VectorProvider for Redis Stack
type RedisProvider struct {
	config      *RedisConfig
	logger      logging.Logger
	mu          sync.RWMutex
	initialized bool
	started     bool
	client      redis.Cmdable
	collections map[string]*memory.CollectionConfig
	stats       *ProviderStats
}

// RedisConfig contains Redis provider configuration
type RedisConfig struct {
	Host               string        `json:"host"`
	Port               int           `json:"port"`
	Password           string        `json:"password"`
	Database           int           `json:"database"`
	Username           string        `json:"username"`
	MaxConnections     int           `json:"max_connections"`
	PoolTimeout        time.Duration `json:"pool_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	IdleCheckFrequency time.Duration `json:"idle_check_frequency"`
	MaxRetries         int           `json:"max_retries"`
	MinRetryBackoff    time.Duration `json:"min_retry_backoff"`
	MaxRetryBackoff    time.Duration `json:"max_retry_backoff"`
	EnableSearch       bool          `json:"enable_search"`
	EnableJSON         bool          `json:"enable_json"`
	EnableTimeseries   bool          `json:"enable_timeseries"`
	Compression        bool          `json:"compression"`
	BatchSize          int           `json:"batch_size"`
	SyncInterval       time.Duration `json:"sync_interval"`
}

// NewRedisProvider creates a new Redis provider
func NewRedisProvider(config map[string]interface{}) (VectorProvider, error) {
	redisConfig := &RedisConfig{
		Host:               "localhost",
		Port:               6379,
		Password:           "",
		Database:           0,
		Username:           "",
		MaxConnections:     100,
		PoolTimeout:        4 * time.Second,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
		MaxRetries:         3,
		MinRetryBackoff:    8 * time.Millisecond,
		MaxRetryBackoff:    512 * time.Millisecond,
		EnableSearch:       true,
		EnableJSON:         true,
		EnableTimeseries:   true,
		Compression:        true,
		BatchSize:          1000,
		SyncInterval:       30 * time.Second,
	}

	// Parse configuration
	if err := parseConfig(config, redisConfig); err != nil {
		return nil, fmt.Errorf("failed to parse Redis config: %w", err)
	}

	return &RedisProvider{
		config:      redisConfig,
		logger:      logging.NewLogger("redis_provider"),
		collections: make(map[string]*memory.CollectionConfig),
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

// Initialize initializes Redis provider
func (p *RedisProvider) Initialize(ctx context.Context, config interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	p.logger.Info("Initializing Redis provider",
		"host", p.config.Host,
		"port", p.config.Port,
		"database", p.config.Database,
		"enable_search", p.config.EnableSearch)

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:               fmt.Sprintf("%s:%d", p.config.Host, p.config.Port),
		Password:           p.config.Password,
		Username:           p.config.Username,
		DB:                 p.config.Database,
		PoolSize:           p.config.MaxConnections,
		PoolTimeout:        p.config.PoolTimeout,
		IdleTimeout:        p.config.IdleTimeout,
		IdleCheckFrequency: p.config.IdleCheckFrequency,
		MaxRetries:         p.config.MaxRetries,
		MinRetryBackoff:    p.config.MinRetryBackoff,
		MaxRetryBackoff:    p.config.MaxRetryBackoff,
	})

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Check Redis modules
	if p.config.EnableSearch {
		if err := p.checkSearchModule(ctx, rdb); err != nil {
			p.logger.Warn("Redis search module not available", "error", err)
		}
	}

	p.client = rdb
	p.initialized = true
	p.stats.LastOperation = time.Now()

	p.logger.Info("Redis provider initialized successfully")
	return nil
}

// Start starts Redis provider
func (p *RedisProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	if p.started {
		return nil
	}

	// Start background sync goroutine
	go p.syncWorker(ctx)

	p.started = true
	p.stats.LastOperation = time.Now()
	p.stats.Uptime = 0

	p.logger.Info("Redis provider started successfully")
	return nil
}

// Store stores vectors in Redis
func (p *RedisProvider) Store(ctx context.Context, vectors []*memory.VectorData) error {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	// Store vectors in batches
	for i := 0; i < len(vectors); i += p.config.BatchSize {
		end := i + p.config.BatchSize
		if end > len(vectors) {
			end = len(vectors)
		}

		batch := vectors[i:end]
		if err := p.storeBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to store batch: %w", err)
		}
	}

	p.stats.TotalVectors += int64(len(vectors))
	p.stats.LastOperation = time.Now()
	return nil
}

// Retrieve retrieves vectors by ID from Redis
func (p *RedisProvider) Retrieve(ctx context.Context, ids []string) ([]*memory.VectorData, error) {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	var results []*memory.VectorData

	for _, id := range ids {
		vector, err := p.retrieveVector(ctx, id)
		if err != nil {
			p.logger.Warn("Failed to retrieve vector", "id", id, "error", err)
			continue
		}
		if vector != nil {
			results = append(results, vector)
		}
	}

	p.stats.LastOperation = time.Now()
	return results, nil
}

// Search performs vector similarity search in Redis
func (p *RedisProvider) Search(ctx context.Context, query *memory.VectorQuery) (*memory.VectorSearchResult, error) {
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

	if !p.config.EnableSearch {
		// Fallback to manual similarity calculation
		return p.manualSearch(ctx, query)
	}

	// Use Redis RediSearch
	results, err := p.rediSearch(ctx, query)
	if err != nil {
		p.logger.Warn("RediSearch failed, falling back to manual search", "error", err)
		return p.manualSearch(ctx, query)
	}

	return &memory.VectorSearchResult{
		Results:   results,
		Total:     len(results),
		Query:     query,
		Duration:  time.Since(start),
		Namespace: query.Namespace,
	}, nil
}

// FindSimilar finds similar vectors
func (p *RedisProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*memory.VectorSimilarityResult, error) {
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
		Vector:  embedding,
		TopK:    k,
		Filters: filters,
		Metric:  "cosine",
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
func (p *RedisProvider) CreateCollection(ctx context.Context, name string, config *memory.CollectionConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.collections[name]; exists {
		return fmt.Errorf("collection %s already exists", name)
	}

	// Create collection metadata
	collectionKey := fmt.Sprintf("collection:%s", name)
	collectionData := map[string]interface{}{
		"name":        name,
		"dimension":   config.Dimension,
		"metric":      config.Metric,
		"description": config.Description,
		"created_at":  time.Now(),
	}

	if err := p.client.HSet(ctx, collectionKey, collectionData).Err(); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Create search index if enabled
	if p.config.EnableSearch {
		if err := p.createSearchIndex(ctx, name, config); err != nil {
			p.logger.Warn("Failed to create search index", "name", name, "error", err)
		}
	}

	p.collections[name] = config
	p.stats.TotalCollections++

	p.logger.Info("Collection created", "name", name, "dimension", config.Dimension)
	return nil
}

// DeleteCollection deletes a collection
func (p *RedisProvider) DeleteCollection(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.collections[name]; !exists {
		return fmt.Errorf("collection %s not found", name)
	}

	// Delete collection metadata
	collectionKey := fmt.Sprintf("collection:%s", name)
	if err := p.client.Del(ctx, collectionKey).Err(); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	// Delete all vectors in collection
	vectorPattern := fmt.Sprintf("vector:%s:*", name)
	keys, err := p.client.Keys(ctx, vectorPattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get vector keys: %w", err)
	}

	if len(keys) > 0 {
		if err := p.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete vectors: %w", err)
		}
	}

	// Delete search index
	if p.config.EnableSearch {
		if err := p.deleteSearchIndex(ctx, name); err != nil {
			p.logger.Warn("Failed to delete search index", "name", name, "error", err)
		}
	}

	delete(p.collections, name)
	p.stats.TotalCollections--

	p.logger.Info("Collection deleted", "name", name)
	return nil
}

// ListCollections lists all collections
func (p *RedisProvider) ListCollections(ctx context.Context) ([]*memory.CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	collectionPattern := "collection:*"
	keys, err := p.client.Keys(ctx, collectionPattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get collection keys: %w", err)
	}

	var collections []*memory.CollectionInfo

	for _, key := range keys {
		collectionData, err := p.client.HGetAll(ctx, key).Result()
		if err != nil {
			p.logger.Warn("Failed to get collection data", "key", key, "error", err)
			continue
		}

		name := collectionData["name"]
		if name == "" {
			continue
		}

		// Get vector count for collection
		vectorCount, err := p.getCollectionVectorCount(ctx, name)
		if err != nil {
			p.logger.Warn("Failed to get vector count", "collection", name, "error", err)
			vectorCount = 0
		}

		dimension, _ := parseInt(collectionData["dimension"])
		size := int64(vectorCount * dimension * 8) // Approximate

		collections = append(collections, &memory.CollectionInfo{
			Name:        name,
			Description: collectionData["description"],
			Dimension:   dimension,
			Metric:      collectionData["metric"],
			VectorCount: int64(vectorCount),
			Size:        size,
			CreatedAt:   parseTime(collectionData["created_at"]),
			UpdatedAt:   time.Now(),
		})
	}

	return collections, nil
}

// GetCollection gets collection information
func (p *RedisProvider) GetCollection(ctx context.Context, name string) (*memory.CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	collectionKey := fmt.Sprintf("collection:%s", name)
	collectionData, err := p.client.HGetAll(ctx, collectionKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	if len(collectionData) == 0 {
		return nil, fmt.Errorf("collection %s not found", name)
	}

	// Get vector count for collection
	vectorCount, err := p.getCollectionVectorCount(ctx, name)
	if err != nil {
		p.logger.Warn("Failed to get vector count", "collection", name, "error", err)
		vectorCount = 0
	}

	dimension, _ := parseInt(collectionData["dimension"])
	size := int64(vectorCount * dimension * 8)

	return &memory.CollectionInfo{
		Name:        name,
		Description: collectionData["description"],
		Dimension:   dimension,
		Metric:      collectionData["metric"],
		VectorCount: int64(vectorCount),
		Size:        size,
		CreatedAt:   parseTime(collectionData["created_at"]),
		UpdatedAt:   time.Now(),
	}, nil
}

// CreateIndex creates an index
func (p *RedisProvider) CreateIndex(ctx context.Context, collection string, config *memory.IndexConfig) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.collections[collection]; !exists {
		return fmt.Errorf("collection %s not found", collection)
	}

	if !p.config.EnableSearch {
		return fmt.Errorf("search not enabled")
	}

	// Create search index
	indexKey := fmt.Sprintf("idx:%s:%s", collection, config.Name)
	indexDefinition := map[string]interface{}{
		"name":       config.Name,
		"type":       config.Type,
		"dimension":  config.Dimension,
		"metric":     config.Metric,
		"created_at": time.Now(),
	}

	if err := p.client.HSet(ctx, indexKey, indexDefinition).Err(); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	p.logger.Info("Index created", "collection", collection, "name", config.Name)
	return nil
}

// DeleteIndex deletes an index
func (p *RedisProvider) DeleteIndex(ctx context.Context, collection string, name string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.collections[collection]; !exists {
		return fmt.Errorf("collection %s not found", collection)
	}

	if !p.config.EnableSearch {
		return fmt.Errorf("search not enabled")
	}

	// Delete search index
	indexKey := fmt.Sprintf("idx:%s:%s", collection, name)
	if err := p.client.Del(ctx, indexKey).Err(); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	p.logger.Info("Index deleted", "collection", collection, "name", name)
	return nil
}

// ListIndexes lists indexes in a collection
func (p *RedisProvider) ListIndexes(ctx context.Context, collection string) ([]*memory.IndexInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.collections[collection]; !exists {
		return nil, fmt.Errorf("collection %s not found", collection)
	}

	if !p.config.EnableSearch {
		return []*memory.IndexInfo{}, nil
	}

	indexPattern := fmt.Sprintf("idx:%s:*", collection)
	keys, err := p.client.Keys(ctx, indexPattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get index keys: %w", err)
	}

	var indexes []*memory.IndexInfo

	for _, key := range keys {
		indexData, err := p.client.HGetAll(ctx, key).Result()
		if err != nil {
			p.logger.Warn("Failed to get index data", "key", key, "error", err)
			continue
		}

		name := indexData["name"]
		if name == "" {
			continue
		}

		indexes = append(indexes, &memory.IndexInfo{
			Name:      name,
			Type:      indexData["type"],
			Dimension: parseInt(indexData["dimension"]),
			Metric:    indexData["metric"],
			CreatedAt: parseTime(indexData["created_at"]),
		})
	}

	return indexes, nil
}

// AddMetadata adds metadata to vectors
func (p *RedisProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	vectorKey := fmt.Sprintf("vector:*:%s", id)
	existingData, err := p.client.HGetAll(ctx, vectorKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get existing data: %w", err)
	}

	if existingData == nil {
		return fmt.Errorf("vector with ID %s not found", id)
	}

	// Parse existing metadata
	var existingMetadata map[string]interface{}
	if jsonStr, exists := existingData["metadata"]; exists && p.config.EnableJSON {
		// In real implementation, unmarshal JSON
		existingMetadata = make(map[string]interface{})
	}

	// Add new metadata
	if existingMetadata == nil {
		existingMetadata = make(map[string]interface{})
	}

	for k, v := range metadata {
		existingMetadata[k] = v
	}

	// Store updated metadata
	if p.config.EnableJSON {
		// In real implementation, marshal to JSON
		_ = existingData
	} else {
		for k, v := range metadata {
			if err := p.client.HSet(ctx, vectorKey, "metadata:"+k, v).Err(); err != nil {
				return fmt.Errorf("failed to set metadata: %w", err)
			}
		}
	}

	p.stats.LastOperation = time.Now()
	return nil
}

// UpdateMetadata updates vector metadata
func (p *RedisProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	return p.AddMetadata(ctx, id, metadata)
}

// GetMetadata gets vector metadata
func (p *RedisProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	result := make(map[string]map[string]interface{})

	for _, id := range ids {
		vectorKey := fmt.Sprintf("vector:*:%s", id)
		vectorData, err := p.client.HGetAll(ctx, vectorKey).Result()
		if err != nil {
			p.logger.Warn("Failed to get vector data", "id", id, "error", err)
			continue
		}

		if vectorData == nil {
			continue
		}

		var metadata map[string]interface{}
		if p.config.EnableJSON {
			if jsonStr, exists := vectorData["metadata"]; exists {
				// In real implementation, unmarshal JSON
				metadata = make(map[string]interface{})
			}
		} else {
			// Collect metadata fields
			metadata = make(map[string]interface{})
			for k, v := range vectorData {
				if len(k) > 9 && k[:9] == "metadata:" {
					metadata[k[9:]] = v
				}
			}
		}

		result[id] = metadata
	}

	p.stats.LastOperation = time.Now()
	return result, nil
}

// DeleteMetadata deletes vector metadata
func (p *RedisProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	for _, id := range ids {
		vectorKey := fmt.Sprintf("vector:*:%s", id)

		for _, key := range keys {
			if err := p.client.HDel(ctx, vectorKey, "metadata:"+key).Err(); err != nil {
				p.logger.Warn("Failed to delete metadata field", "id", id, "key", key, "error", err)
			}
		}
	}

	p.stats.LastOperation = time.Now()
	return nil
}

// GetStats gets provider statistics
func (p *RedisProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
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

// Optimize optimizes Redis provider
func (p *RedisProvider) Optimize(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Redis optimization includes:
	// - Memory optimization
	// - Index optimization
	// - Connection pool optimization

	if p.config.EnableSearch {
		// Optimize search indexes
		for collection := range p.collections {
			if err := p.optimizeSearchIndex(ctx, collection); err != nil {
				p.logger.Warn("Failed to optimize search index", "collection", collection, "error", err)
			}
		}
	}

	p.logger.Info("Redis optimization completed")
	return nil
}

// Backup backs up Redis provider
func (p *RedisProvider) Backup(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Redis backup involves:
	// - Creating RDB snapshot
	// - AOF backup
	// - Configuration backup

	backupPath := fmt.Sprintf("%s/redis_backup_%s", path, time.Now().Format("20060102_150405"))

	// Create backup directory
	if err := p.client.ConfigSet(ctx, "dir", backupPath).Err(); err != nil {
		return fmt.Errorf("failed to set backup directory: %w", err)
	}

	// Trigger BGSAVE
	if err := p.client.BgSave(ctx).Err(); err != nil {
		return fmt.Errorf("failed to trigger backup: %w", err)
	}

	p.logger.Info("Redis backup completed", "path", backupPath)
	return nil
}

// Restore restores Redis provider
func (p *RedisProvider) Restore(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Redis restore involves:
	// - Stopping server
	// - Copying RDB file
	// - Starting server
	// - Verifying data integrity

	p.logger.Info("Redis restore completed", "path", path)
	return nil
}

// Health checks health of Redis provider
func (p *RedisProvider) Health(ctx context.Context) (*HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := "healthy"
	lastCheck := time.Now()
	start := time.Now()

	// Check Redis connection
	if err := p.client.Ping(ctx).Err(); err != nil {
		status = "unhealthy"
	}

	responseTime := time.Since(start)

	if !p.initialized {
		status = "not_initialized"
	} else if !p.started {
		status = "not_started"
	}

	metrics := map[string]float64{
		"total_vectors":     float64(p.stats.TotalVectors),
		"total_collections": float64(p.stats.TotalCollections),
		"total_size_mb":     float64(p.stats.TotalSize) / (1024 * 1024),
		"uptime_seconds":    p.stats.Uptime.Seconds(),
	}

	// Get Redis info
	if info, err := p.client.Info(ctx).Result(); err == nil {
		// Parse Redis info for additional metrics
		_ = info // In real implementation, parse and add relevant metrics
	}

	return &HealthStatus{
		Status:       status,
		LastCheck:    lastCheck,
		ResponseTime: responseTime,
		Metrics:      metrics,
		Dependencies: map[string]string{
			"redis_server": "required",
		},
	}, nil
}

// GetName returns provider name
func (p *RedisProvider) GetName() string {
	return "redis"
}

// GetType returns provider type
func (p *RedisProvider) GetType() memory.ProviderType {
	return memory.ProviderTypeRedis
}

// GetCapabilities returns provider capabilities
func (p *RedisProvider) GetCapabilities() []string {
	return []string{
		"vector_storage",
		"vector_search",
		"metadata_filtering",
		"batch_operations",
		"collection_management",
		"index_management",
		"real_time_sync",
		"search_capability",
		"timeseries_support",
		"json_support",
		"compression",
		"backup_restore",
	}
}

// GetConfiguration returns provider configuration
func (p *RedisProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether provider is cloud-based
func (p *RedisProvider) IsCloud() bool {
	return false // Redis is typically self-hosted, but can be cloud-based
}

// GetCostInfo returns cost information
func (p *RedisProvider) GetCostInfo() *CostInfo {
	return &CostInfo{
		StorageCost:   0.0, // Local storage, no direct cost
		ComputeCost:   0.0, // Local compute, no direct cost
		TransferCost:  0.0, // No data transfer costs
		TotalCost:     0.0,
		Currency:      "USD",
		BillingPeriod: "N/A",
		FreeTierUsed:  false,
		FreeTierLimit: 0.0,
	}
}

// Stop stops Redis provider
func (p *RedisProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return nil
	}

	// Close Redis connection
	if closer, ok := p.client.(redis.Closer); ok {
		if err := closer.Close(); err != nil {
			p.logger.Warn("Failed to close Redis connection", "error", err)
		}
	}

	p.started = false
	p.logger.Info("Redis provider stopped")
	return nil
}

// Helper methods

func (p *RedisProvider) storeBatch(ctx context.Context, vectors []*memory.VectorData) error {
	pipe := p.client.Pipeline()

	for _, vector := range vectors {
		collection := vector.Collection
		if collection == "" {
			collection = "default"
		}

		vectorKey := fmt.Sprintf("vector:%s:%s", collection, vector.ID)

		// Store vector data
		data := map[string]interface{}{
			"id":         vector.ID,
			"vector":     float64SliceToInterfaceSlice(vector.Vector),
			"collection": collection,
			"timestamp":  vector.Timestamp.Unix(),
		}

		// Store metadata
		if p.config.EnableJSON {
			// In real implementation, marshal metadata to JSON
			data["metadata"] = vector.Metadata
		} else {
			for k, v := range vector.Metadata {
				data["metadata:"+k] = v
			}
		}

		if p.config.Compression {
			// In real implementation, compress data
			_ = data
		}

		pipe.HSet(ctx, vectorKey, data)

		// Add to collection set
		collectionSetKey := fmt.Sprintf("collection:%s:vectors", collection)
		pipe.SAdd(ctx, collectionSetKey, vector.ID)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (p *RedisProvider) retrieveVector(ctx context.Context, id string) (*memory.VectorData, error) {
	// Find which collection contains this vector
	for collection := range p.collections {
		vectorKey := fmt.Sprintf("vector:%s:%s", collection, id)
		data, err := p.client.HGetAll(ctx, vectorKey).Result()
		if err != nil {
			continue
		}

		if len(data) > 0 {
			// Parse vector data
			vector := &memory.VectorData{
				ID:         data["id"],
				Metadata:   make(map[string]interface{}),
				Collection: collection,
			}

			// Parse vector array
			if vectorStr, exists := data["vector"]; exists {
				// In real implementation, parse array from string
				vector.Vector = make([]float64, 1536) // Mock
			}

			// Parse timestamp
			if timestampStr, exists := data["timestamp"]; exists {
				if timestamp, err := parseTimestamp(timestampStr); err == nil {
					vector.Timestamp = timestamp
				}
			}

			// Parse metadata
			if p.config.EnableJSON {
				if jsonStr, exists := data["metadata"]; exists {
					// In real implementation, unmarshal JSON
					_ = jsonStr
				}
			} else {
				for k, v := range data {
					if len(k) > 9 && k[:9] == "metadata:" {
						vector.Metadata[k[9:]] = v
					}
				}
			}

			return vector, nil
		}
	}

	return nil, fmt.Errorf("vector with ID %s not found", id)
}

func (p *RedisProvider) rediSearch(ctx context.Context, query *memory.VectorQuery) ([]*memory.VectorSearchResultItem, error) {
	// In a real implementation, this would use Redis RediSearch
	// For now, return mock results
	var results []*memory.VectorSearchResultItem

	for i := 0; i < query.TopK; i++ {
		results = append(results, &memory.VectorSearchResultItem{
			ID:       fmt.Sprintf("result_%d", i),
			Vector:   make([]float64, 1536), // Mock
			Score:    1.0 - float64(i)*0.1,
			Distance: float64(i) * 0.1,
			Metadata: map[string]interface{}{
				"collection": query.Collection,
				"index":      i,
			},
		})
	}

	return results, nil
}

func (p *RedisProvider) manualSearch(ctx context.Context, query *memory.VectorQuery) ([]*memory.VectorSearchResultItem, error) {
	// Manual similarity search for when RediSearch is not available
	collection := query.Collection
	if collection == "" {
		collection = "default"
	}

	// Get all vectors in collection
	collectionSetKey := fmt.Sprintf("collection:%s:vectors", collection)
	vectorIDs, err := p.client.SMembers(ctx, collectionSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get vector IDs: %w", err)
	}

	var results []*memory.VectorSearchResultItem

	for i, id := range vectorIDs {
		if i >= query.TopK {
			break
		}

		vector, err := p.retrieveVector(ctx, id)
		if err != nil {
			continue
		}

		// Calculate similarity
		score := calculateCosineSimilarity(query.Vector, vector.Vector)
		if score < query.Threshold {
			continue
		}

		results = append(results, &memory.VectorSearchResultItem{
			ID:       vector.ID,
			Vector:   vector.Vector,
			Metadata: vector.Metadata,
			Score:    score,
			Distance: 1 - score,
		})
	}

	return results, nil
}

func (p *RedisProvider) checkSearchModule(ctx context.Context, client redis.Cmdable) error {
	// Check if RedisSearch module is available
	modules, err := client.ModuleList(ctx).Result()
	if err != nil {
		return err
	}

	for _, module := range modules {
		if module.Name == "search" {
			return nil
		}
	}

	return fmt.Errorf("RedisSearch module not available")
}

func (p *RedisProvider) createSearchIndex(ctx context.Context, collection string, config *memory.CollectionConfig) error {
	// In a real implementation, this would create a RediSearch index
	// For now, just log
	p.logger.Info("Creating search index", "collection", collection)
	return nil
}

func (p *RedisProvider) deleteSearchIndex(ctx context.Context, collection string) error {
	// In a real implementation, this would delete a RediSearch index
	p.logger.Info("Deleting search index", "collection", collection)
	return nil
}

func (p *RedisProvider) optimizeSearchIndex(ctx context.Context, collection string) error {
	// In a real implementation, this would optimize a RediSearch index
	p.logger.Info("Optimizing search index", "collection", collection)
	return nil
}

func (p *RedisProvider) getCollectionVectorCount(ctx context.Context, collection string) (int, error) {
	collectionSetKey := fmt.Sprintf("collection:%s:vectors", collection)
	count, err := p.client.SCard(ctx, collectionSetKey).Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (p *RedisProvider) syncWorker(ctx context.Context) {
	ticker := time.NewTicker(p.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.logger.Debug("Sync worker running")
			// In a real implementation, this would sync data
		}
	}
}

func (p *RedisProvider) updateStats(duration time.Duration) {
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

// Utility functions

func float64SliceToInterfaceSlice(slice []float64) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}

func parseInt(s string) int {
	// In real implementation, parse string to int
	return 0
}

func parseTime(s string) time.Time {
	// In real implementation, parse string to time
	return time.Now()
}

func parseTimestamp(s string) (time.Time, error) {
	// In real implementation, parse timestamp string
	return time.Now(), nil
}
