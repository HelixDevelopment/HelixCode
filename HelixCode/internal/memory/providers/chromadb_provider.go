package providers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromadb/chromadb"
	"github.com/chromadb/chromadb/api"
	"github.com/chromadb/chromadb/openai"
	"github.com/google/uuid"
)

// ChromaDBProvider implements VectorProvider interface for ChromaDB
type ChromaDBProvider struct {
	client       chromadb.Client
	collections  map[string]api.Collection
	config       *ChromaDBConfig
	logger       logging.Logger
	initialized  bool
	started      bool
	stats        *ProviderStats
}

// ChromaDBConfig represents ChromaDB configuration
type ChromaDBConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Path         string        `json:"path"`
	APIKey       string        `json:"api_key"`
	Tenant       string        `json:"tenant"`
	Database     string        `json:"database"`
	Timeout      time.Duration `json:"timeout"`
	MaxRetries   int           `json:"max_retries"`
	BatchSize    int           `json:"batch_size"`
	Compression  bool          `json:"compression"`
	Metric       string        `json:"metric"`
	Dimension    int           `json:"dimension"`
}

// ProviderStats represents provider statistics
type ProviderStats struct {
	TotalVectors     int64         `json:"total_vectors"`
	TotalCollections int64         `json:"total_collections"`
	TotalSize       int64         `json:"total_size"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastOperation   time.Time     `json:"last_operation"`
	ErrorCount      int64         `json:"error_count"`
	Uptime          time.Duration `json:"uptime"`
}

// NewChromaDBProvider creates a new ChromaDB provider
func NewChromaDBProvider(config interface{}) (VectorProvider, error) {
	chromadbConfig, err := parseChromaDBConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ChromaDB config: %w", err)
	}
	
	logger := logging.NewLogger("chromadb_provider")
	
	return &ChromaDBProvider{
		collections: make(map[string]api.Collection),
		config:      chromadbConfig,
		logger:      logger,
		stats: &ProviderStats{
			TotalVectors:     0,
			TotalCollections: 0,
			TotalSize:        0,
			AverageLatency:    0,
			LastOperation:     time.Now(),
			ErrorCount:       0,
			Uptime:           0,
		},
	}, nil
}

// Initialize initializes the ChromaDB provider
func (p *ChromaDBProvider) Initialize(ctx context.Context, config interface{}) error {
	p.logger.Info("Initializing ChromaDB provider...")
	
	// Parse configuration
	chromadbConfig, err := parseChromaDBConfig(config)
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}
	
	p.config = chromadbConfig
	
	// Create ChromaDB client
	clientOptions := []chromadb.Option{
		chromadb.WithHost(fmt.Sprintf("%s:%d", chromadbConfig.Host, chromadbConfig.Port)),
		chromadb.WithTimeout(chromadbConfig.Timeout),
	}
	
	// Add API key if provided
	if chromadbConfig.APIKey != "" {
		clientOptions = append(clientOptions, chromadb.WithAPIKey(chromadbConfig.APIKey))
	}
	
	// Create client
	p.client = chromadb.NewClient(clientOptions...)
	
	// Test connection
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to ChromaDB: %w", err)
	}
	
	p.initialized = true
	p.logger.Info("ChromaDB provider initialized successfully")
	
	return nil
}

// Start starts the ChromaDB provider
func (p *ChromaDBProvider) Start(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}
	
	p.logger.Info("Starting ChromaDB provider...")
	
	// Load existing collections
	if err := p.loadCollections(ctx); err != nil {
		return fmt.Errorf("failed to load collections: %w", err)
	}
	
	p.started = true
	p.stats.Uptime = time.Since(time.Now())
	
	p.logger.Info("ChromaDB provider started successfully")
	return nil
}

// Store stores vectors in ChromaDB
func (p *ChromaDBProvider) Store(ctx context.Context, vectors []*VectorData) error {
	if !p.started {
		return fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("store", duration, nil)
	}()
	
	p.logger.Debug("Storing vectors", "count", len(vectors))
	
	// Group vectors by collection
	vectorsByCollection := make(map[string][]*VectorData)
	for _, vector := range vectors {
		collection := vector.Collection
		if collection == "" {
			collection = "default"
		}
		vectorsByCollection[collection] = append(vectorsByCollection[collection], vector)
	}
	
	// Store vectors in each collection
	for collection, collectionVectors := range vectorsByCollection {
		if err := p.storeInCollection(ctx, collection, collectionVectors); err != nil {
			p.updateStats("store", time.Since(start), err)
			return fmt.Errorf("failed to store in collection %s: %w", collection, err)
		}
	}
	
	p.logger.Debug("Vectors stored successfully", "count", len(vectors))
	return nil
}

// Retrieve retrieves vectors from ChromaDB
func (p *ChromaDBProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("retrieve", duration, nil)
	}()
	
	p.logger.Debug("Retrieving vectors", "count", len(ids))
	
	var results []*VectorData
	
	// Since ChromaDB doesn't have a direct get by IDs across collections,
	// we'll need to search in each collection (this is a limitation)
	for name, collection := range p.collections {
		// Try to get vectors from this collection
		collectionResults, err := p.retrieveFromCollection(ctx, name, collection, ids)
		if err != nil {
			p.logger.Warn("Failed to retrieve from collection", "collection", name, "error", err)
			continue
		}
		
		results = append(results, collectionResults...)
	}
	
	p.logger.Debug("Vectors retrieved successfully", "count", len(results))
	return results, nil
}

// Search performs vector search in ChromaDB
func (p *ChromaDBProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("search", duration, nil)
	}()
	
	p.logger.Debug("Searching vectors", "collection", query.Collection, "top_k", query.TopK)
	
	// Get collection
	collection, exists := p.collections[query.Collection]
	if !exists {
		return nil, fmt.Errorf("collection %s not found", query.Collection)
	}
	
	// Convert query to ChromaDB format
	chromadbQuery := []api.Query{
		{
			QueryEmbeddings: [][]float32{p.convertFloat64ToFloat32(query.Vector)},
			NResults:        &query.TopK,
			Where:          p.convertFilters(query.Filters),
		},
	}
	
	// Execute search
	result, err := collection.Query(
		ctx,
		chromadbQuery,
	)
	if err != nil {
		p.updateStats("search", time.Since(start), err)
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	
	// Convert results
	var searchResults []*VectorSearchResultItem
	if len(result) > 0 && len(result[0].IDs) > 0 {
		for i, id := range result[0].IDs {
			searchItem := &VectorSearchResultItem{
				ID:       id,
				Vector:   p.convertFloat32ToFloat64(result[0].Embeddings[i]),
				Score:    result[0].Distances[i],
				Distance: result[0].Distances[i],
			}
			
			// Add metadata
			if i < len(result[0].Metadatas) && result[0].Metadatas[i] != nil {
				searchItem.Metadata = p.convertMetadata(result[0].Metadatas[i])
			}
			
			searchResults = append(searchResults, searchItem)
		}
	}
	
	// Create search result
	searchResult := &VectorSearchResult{
		Results:   searchResults,
		Total:     len(searchResults),
		Query:     query,
		Duration:  time.Since(start),
		Namespace: query.Namespace,
	}
	
	p.logger.Debug("Search completed successfully", "results", len(searchResults))
	return searchResult, nil
}

// FindSimilar finds similar vectors in ChromaDB
func (p *ChromaDBProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("find_similar", duration, nil)
	}()
	
	p.logger.Debug("Finding similar vectors", "k", k)
	
	var allResults []*VectorSimilarityResult
	
	// Search in all collections
	for name, collection := range p.collections {
		query := []api.Query{
			{
				QueryEmbeddings: [][]float32{p.convertFloat64ToFloat32(embedding)},
				NResults:        &k,
				Where:          p.convertFilters(filters),
			},
		}
		
		result, err := collection.Query(ctx, query)
		if err != nil {
			p.logger.Warn("Failed to find similar in collection", "collection", name, "error", err)
			continue
		}
		
		// Convert results
		if len(result) > 0 && len(result[0].IDs) > 0 {
			for i, id := range result[0].IDs {
				similarItem := &VectorSimilarityResult{
					ID:       id,
					Vector:   p.convertFloat32ToFloat64(result[0].Embeddings[i]),
					Score:    result[0].Distances[i],
					Distance: result[0].Distances[i],
				}
				
				// Add metadata
				if i < len(result[0].Metadatas) && result[0].Metadatas[i] != nil {
					similarItem.Metadata = p.convertMetadata(result[0].Metadatas[i])
				}
				
				allResults = append(allResults, similarItem)
			}
		}
	}
	
	p.logger.Debug("Similar vectors found", "count", len(allResults))
	return allResults, nil
}

// CreateCollection creates a new collection in ChromaDB
func (p *ChromaDBProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	if !p.started {
		return fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("create_collection", duration, nil)
	}()
	
	p.logger.Debug("Creating collection", "name", name, "dimension", config.Dimension)
	
	// Check if collection already exists
	if _, exists := p.collections[name]; exists {
		return fmt.Errorf("collection %s already exists", name)
	}
	
	// Create collection
	collection, err := p.client.CreateCollection(
		name,
		p.convertMetric(config.Metric),
		map[string]interface{}{
			"hnsw:space": p.convertMetric(config.Metric),
			"hnsw:M":      16,
			"hnsw:ef_construction": 64,
		},
	)
	if err != nil {
		p.updateStats("create_collection", time.Since(start), err)
		return fmt.Errorf("failed to create collection: %w", err)
	}
	
	// Store collection
	p.collections[name] = collection
	p.stats.TotalCollections++
	
	p.logger.Debug("Collection created successfully", "name", name)
	return nil
}

// GetStats returns provider statistics
func (p *ChromaDBProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}
	
	// Update uptime
	if p.stats.Uptime > 0 {
		p.stats.Uptime = time.Since(time.Now().Add(-p.stats.Uptime))
	}
	
	// Return copy of stats
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

// Optimize optimizes the ChromaDB provider
func (p *ChromaDBProvider) Optimize(ctx context.Context) error {
	if !p.started {
		return fmt.Errorf("provider not started")
	}
	
	p.logger.Info("Optimizing ChromaDB provider...")
	
	// ChromaDB doesn't have explicit optimization commands
	// Optimization happens automatically during operations
	
	p.logger.Info("ChromaDB provider optimization completed")
	return nil
}

// Health returns health status
func (p *ChromaDBProvider) Health(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("health", duration, nil)
	}()
	
	status := &HealthStatus{
		Status:      "healthy",
		LastCheck:   time.Now(),
		ResponseTime: time.Since(start),
		Metrics:     make(map[string]float64),
		Dependencies: make(map[string]string),
	}
	
	// Check if started
	if !p.started {
		status.Status = "unhealthy"
		status.Error = "provider not started"
		return status, nil
	}
	
	// Test connection
	if err := p.testConnection(ctx); err != nil {
		status.Status = "unhealthy"
		status.Error = err.Error()
		return status, nil
	}
	
	// Add metrics
	status.Metrics["total_collections"] = float64(p.stats.TotalCollections)
	status.Metrics["total_vectors"] = float64(p.stats.TotalVectors)
	status.Metrics["error_count"] = float64(p.stats.ErrorCount)
	
	return status, nil
}

// GetName returns provider name
func (p *ChromaDBProvider) GetName() string {
	return "ChromaDB"
}

// GetType returns provider type
func (p *ChromaDBProvider) GetType() ProviderType {
	return ProviderTypeChromaDB
}

// GetCapabilities returns provider capabilities
func (p *ChromaDBProvider) GetCapabilities() []string {
	return []string{
		"vector_storage",
		"vector_search",
		"metadata_filtering",
		"batch_operations",
		"collection_management",
		"similarity_search",
		"hybrid_search",
		"local_deployment",
	}
}

// GetConfiguration returns provider configuration
func (p *ChromaDBProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns false for ChromaDB (it's local)
func (p *ChromaDBProvider) IsCloud() bool {
	return false
}

// GetCostInfo returns cost information (always local, no cost)
func (p *ChromaDBProvider) GetCostInfo() *CostInfo {
	return &CostInfo{
		StorageCost:   0,
		ComputeCost:   0,
		TransferCost:  0,
		TotalCost:      0,
		Currency:      "USD",
		BillingPeriod:  "local",
		FreeTierUsed:   false,
		FreeTierLimit:  0,
	}
}

// Stop stops the ChromaDB provider
func (p *ChromaDBProvider) Stop(ctx context.Context) error {
	p.logger.Info("Stopping ChromaDB provider...")
	
	p.started = false
	p.collections = make(map[string]api.Collection)
	
	p.logger.Info("ChromaDB provider stopped successfully")
	return nil
}

// Private helper methods

func (p *ChromaDBProvider) testConnection(ctx context.Context) error {
	// Try to list collections to test connection
	_, err := p.client.ListCollections(ctx)
	return err
}

func (p *ChromaDBProvider) loadCollections(ctx context.Context) error {
	collections, err := p.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}
	
	for _, collection := range collections {
		p.collections[collection.Name()] = collection
	}
	
	p.logger.Info("Loaded collections", "count", len(p.collections))
	return nil
}

func (p *ChromaDBProvider) storeInCollection(ctx context.Context, collectionName string, vectors []*VectorData) error {
	// Get or create collection
	collection, exists := p.collections[collectionName]
	if !exists {
		// Create collection with default config
		collectionConfig := &CollectionConfig{
			Name:      collectionName,
			Dimension: len(vectors[0].Vector),
			Metric:    "cosine",
		}
		
		if err := p.CreateCollection(ctx, collectionName, collectionConfig); err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
		
		collection = p.collections[collectionName]
	}
	
	// Prepare data for batch insert
	ids := make([]string, len(vectors))
	embeddings := make([][]float32, len(vectors))
	metadatas := make([]map[string]interface{}, len(vectors))
	
	for i, vector := range vectors {
		if vector.ID == "" {
			vector.ID = uuid.New().String()
		}
		
		ids[i] = vector.ID
		embeddings[i] = p.convertFloat64ToFloat32(vector.Vector)
		metadatas[i] = p.convertToChromaDBMetadata(vector)
	}
	
	// Batch insert
	if err := collection.Add(ctx, ids, embeddings, metadatas, nil, nil); err != nil {
		return fmt.Errorf("failed to add vectors: %w", err)
	}
	
	// Update stats
	p.stats.TotalVectors += int64(len(vectors))
	
	return nil
}

func (p *ChromaDBProvider) retrieveFromCollection(ctx context.Context, collectionName string, collection api.Collection, ids []string) ([]*VectorData, error) {
	// Get vectors by ID
	result, err := collection.Get(ctx, ids, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get vectors: %w", err)
	}
	
	var vectors []*VectorData
	
	// Convert results
	for i, id := range result.IDs {
		vector := &VectorData{
			ID:         id,
			Vector:     p.convertFloat32ToFloat64(result.Embeddings[i]),
			Metadata:   p.convertMetadata(result.Metadatas[i]),
			Collection: collectionName,
		}
		
		// Add timestamp from metadata if available
		if result.Metadatas[i] != nil {
			if timestamp, exists := result.Metadatas[i]["timestamp"]; exists {
				if timestampStr, ok := timestamp.(string); ok {
					if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
						vector.Timestamp = t
					}
				}
			}
		}
		
		vectors = append(vectors, vector)
	}
	
	return vectors, nil
}

func (p *ChromaDBProvider) convertFloat64ToFloat32(vector []float64) []float32 {
	result := make([]float32, len(vector))
	for i, v := range vector {
		result[i] = float32(v)
	}
	return result
}

func (p *ChromaDBProvider) convertFloat32ToFloat64(vector []float32) []float64 {
	result := make([]float64, len(vector))
	for i, v := range vector {
		result[i] = float64(v)
	}
	return result
}

func (p *ChromaDBProvider) convertToChromaDBMetadata(vector *VectorData) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Copy metadata
	for k, v := range vector.Metadata {
		metadata[k] = v
	}
	
	// Add built-in fields
	metadata["collection"] = vector.Collection
	metadata["namespace"] = vector.Namespace
	metadata["timestamp"] = vector.Timestamp.Format(time.RFC3339)
	
	if vector.TTL != nil {
		metadata["ttl"] = vector.TTL.String()
	}
	
	return metadata
}

func (p *ChromaDBProvider) convertMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return make(map[string]interface{})
	}
	
	result := make(map[string]interface{})
	for k, v := range metadata {
		result[k] = v
	}
	
	return result
}

func (p *ChromaDBProvider) convertFilters(filters map[string]interface{}) api.WhereDocument {
	if len(filters) == 0 {
		return nil
	}
	
	// Convert filters to ChromaDB where clause
	// This is a simplified conversion - in practice, you'd want more sophisticated filtering
	var conditions []map[string]interface{}
	
	for key, value := range filters {
		condition := map[string]interface{}{
			key: value,
		}
		conditions = append(conditions, condition)
	}
	
	if len(conditions) == 1 {
		return api.WhereDocument{conditions[0]}
	}
	
	return api.WhereDocument{"$and": conditions}
}

func (p *ChromaDBProvider) convertMetric(metric string) string {
	switch strings.ToLower(metric) {
	case "cosine":
		return "cosine"
	case "l2":
		return "l2"
	case "ip":
		return "ip"
	default:
		return "cosine"
	}
}

func (p *ChromaDBProvider) updateStats(operation string, duration time.Duration, err error) {
	p.stats.LastOperation = time.Now()
	
	// Update average latency
	if p.stats.AverageLatency == 0 {
		p.stats.AverageLatency = duration
	} else {
		p.stats.AverageLatency = (p.stats.AverageLatency + duration) / 2
	}
	
	// Update error count
	if err != nil {
		p.stats.ErrorCount++
	}
}

func parseChromaDBConfig(config interface{}) (*ChromaDBConfig, error) {
	chromadbConfig := &ChromaDBConfig{
		Host:       "localhost",
		Port:       8000,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		BatchSize:  100,
		Compression: true,
		Metric:     "cosine",
		Dimension:  1536,
		Tenant:     "default_tenant",
		Database:   "default_database",
	}
	
	if config != nil {
		// Parse configuration from map or struct
		// This is a simplified parser - in practice, you'd want more robust parsing
		if configMap, ok := config.(map[string]interface{}); ok {
			if host, exists := configMap["host"]; exists {
				if hostStr, ok := host.(string); ok {
					chromadbConfig.Host = hostStr
				}
			}
			if port, exists := configMap["port"]; exists {
				if portInt, ok := port.(int); ok {
					chromadbConfig.Port = portInt
				}
				if portStr, ok := port.(string); ok {
					if portInt, err := strconv.Atoi(portStr); err == nil {
						chromadbConfig.Port = portInt
					}
				}
			}
			if apikey, exists := configMap["api_key"]; exists {
				if apikeyStr, ok := apikey.(string); ok {
					chromadbConfig.APIKey = apikeyStr
				}
			}
			if timeout, exists := configMap["timeout"]; exists {
				if timeoutStr, ok := timeout.(string); ok {
					if timeoutDur, err := time.ParseDuration(timeoutStr); err == nil {
						chromadbConfig.Timeout = timeoutDur
					}
				}
			}
		}
	}
	
	return chromadbConfig, nil
}

// Import required packages
import (
	"dev.helix.code/internal/logging"
)