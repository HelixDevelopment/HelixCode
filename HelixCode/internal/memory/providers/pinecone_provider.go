package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pinecone-io/go-pinecone/pinecone"
)

// PineconeProvider implements VectorProvider interface for Pinecone
type PineconeProvider struct {
	client       *pinecone.Client
	index        *pinecone.Index
	collections  map[string]*PineconeCollection
	config       *PineconeConfig
	logger       logging.Logger
	initialized  bool
	started      bool
	stats        *ProviderStats
}

// PineconeConfig represents Pinecone configuration
type PineconeConfig struct {
	APIKey         string        `json:"api_key"`
	Environment    string        `json:"environment"`
	ProjectID      string        `json:"project_id"`
	IndexName      string        `json:"index_name"`
	Dimension      int           `json:"dimension"`
	Metric         string        `json:"metric"`
	PodType        string        `json:"pod_type"`
	Pods           int           `json:"pods"`
	Replicas       int           `json:"replicas"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"max_retries"`
	BatchSize      int           `json:"batch_size"`
	Compression    bool          `json:"compression"`
	Namespace      string        `json:"namespace"`
}

// PineconeCollection represents a Pinecone collection
type PineconeCollection struct {
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace"`
	Dimension  int                    `json:"dimension"`
	Metric     string                 `json:"metric"`
	Size       int64                  `json:"size"`
	VectorCount int64                  `json:"vector_count"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// NewPineconeProvider creates a new Pinecone provider
func NewPineconeProvider(config interface{}) (VectorProvider, error) {
	pineconeConfig, err := parsePineconeConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Pinecone config: %w", err)
	}
	
	logger := logging.NewLogger("pinecone_provider")
	
	return &PineconeProvider{
		collections: make(map[string]*PineconeCollection),
		config:      pineconeConfig,
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

// Initialize initializes Pinecone provider
func (p *PineconeProvider) Initialize(ctx context.Context, config interface{}) error {
	p.logger.Info("Initializing Pinecone provider...")
	
	// Parse configuration
	pineconeConfig, err := parsePineconeConfig(config)
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}
	
	p.config = pineconeConfig
	
	// Validate configuration
	if err := p.validateConfig(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Create Pinecone client
	pineconeConfig, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: p.config.APIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create Pinecone client: %w", err)
	}
	
	p.client = pineconeConfig
	
	// Test connection
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to Pinecone: %w", err)
	}
	
	p.initialized = true
	p.logger.Info("Pinecone provider initialized successfully")
	
	return nil
}

// Start starts Pinecone provider
func (p *PineconeProvider) Start(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}
	
	p.logger.Info("Starting Pinecone provider...")
	
	// Connect to or create index
	if err := p.connectToIndex(ctx); err != nil {
		return fmt.Errorf("failed to connect to index: %w", err)
	}
	
	// Load existing collections
	if err := p.loadCollections(ctx); err != nil {
		return fmt.Errorf("failed to load collections: %w", err)
	}
	
	p.started = true
	p.stats.Uptime = time.Since(time.Now())
	
	p.logger.Info("Pinecone provider started successfully")
	return nil
}

// Store stores vectors in Pinecone
func (p *PineconeProvider) Store(ctx context.Context, vectors []*VectorData) error {
	if !p.started {
		return fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("store", duration, nil)
	}()
	
	p.logger.Debug("Storing vectors", "count", len(vectors))
	
	// Group vectors by namespace
	vectorsByNamespace := make(map[string][]*VectorData)
	for _, vector := range vectors {
		namespace := vector.Namespace
		if namespace == "" {
			namespace = p.config.Namespace
		}
		vectorsByNamespace[namespace] = append(vectorsByNamespace[namespace], vector)
	}
	
	// Store vectors in each namespace
	for namespace, namespaceVectors := range vectorsByNamespace {
		if err := p.storeInNamespace(ctx, namespace, namespaceVectors); err != nil {
			p.updateStats("store", time.Since(start), err)
			return fmt.Errorf("failed to store in namespace %s: %w", namespace, err)
		}
	}
	
	p.logger.Debug("Vectors stored successfully", "count", len(vectors))
	return nil
}

// Retrieve retrieves vectors from Pinecone
func (p *PineconeProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
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
	
	// Pinecone doesn't have a direct fetch by IDs, so we need to search for each ID
	// This is less efficient but necessary for compatibility
	for _, id := range ids {
		vector, err := p.fetchVectorByID(ctx, id)
		if err != nil {
			p.logger.Warn("Failed to fetch vector by ID", "id", id, "error", err)
			continue
		}
		
		if vector != nil {
			results = append(results, vector)
		}
	}
	
	p.logger.Debug("Vectors retrieved successfully", "count", len(results))
	return results, nil
}

// Search performs vector search in Pinecone
func (p *PineconeProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("search", duration, nil)
	}()
	
	p.logger.Debug("Searching vectors", "collection", query.Collection, "top_k", query.TopK)
	
	// Get namespace
	namespace := query.Namespace
	if namespace == "" {
		namespace = p.config.Namespace
	}
	
	// Prepare search request
	searchRequest := &pinecone.QueryRequest{
		Vector:        query.Vector,
		TopK:          query.TopK,
		IncludeValues:  query.IncludeVector,
		IncludeMetadata: true,
		Namespace:     namespace,
		Filter:        p.convertFilters(query.Filters),
	}
	
	// Apply threshold if specified
	if query.Threshold > 0 {
		// Pinecone doesn't have direct threshold support in query,
		// so we'll filter results after retrieval
	}
	
	// Execute search
	response, err := p.index.Query(ctx, searchRequest)
	if err != nil {
		p.updateStats("search", time.Since(start), err)
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	
	// Convert results
	var searchResults []*VectorSearchResultItem
	for _, match := range response.Matches {
		// Apply threshold if specified
		if query.Threshold > 0 && match.Score < query.Threshold {
			continue
		}
		
		searchItem := &VectorSearchResultItem{
			ID:       match.ID,
			Vector:   match.Values,
			Metadata: match.Metadata,
			Score:    match.Score,
			Distance: 1 - match.Score, // Convert similarity to distance
		}
		
		searchResults = append(searchResults, searchItem)
	}
	
	// Create search result
	searchResult := &VectorSearchResult{
		Results:   searchResults,
		Total:     len(searchResults),
		Query:     query,
		Duration:  time.Since(start),
		Namespace: namespace,
	}
	
	p.logger.Debug("Search completed successfully", "results", len(searchResults))
	return searchResult, nil
}

// FindSimilar finds similar vectors in Pinecone
func (p *PineconeProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("find_similar", duration, nil)
	}()
	
	p.logger.Debug("Finding similar vectors", "k", k)
	
	// Prepare search request
	searchRequest := &pinecone.QueryRequest{
		Vector:        embedding,
		TopK:          k,
		IncludeValues:  true,
		IncludeMetadata: true,
		Namespace:     p.config.Namespace,
		Filter:        p.convertFilters(filters),
	}
	
	// Execute search
	response, err := p.index.Query(ctx, searchRequest)
	if err != nil {
		p.updateStats("find_similar", time.Since(start), err)
		return nil, fmt.Errorf("failed to find similar vectors: %w", err)
	}
	
	// Convert results
	var results []*VectorSimilarityResult
	for _, match := range.Matches {
		similarItem := &VectorSimilarityResult{
			ID:       match.ID,
			Vector:   match.Values,
			Metadata: match.Metadata,
			Score:    match.Score,
			Distance: 1 - match.Score,
		}
		
		results = append(results, similarItem)
	}
	
	p.logger.Debug("Similar vectors found", "count", len(results))
	return results, nil
}

// CreateCollection creates a namespace in Pinecone
func (p *PineconeProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	if !p.started {
		return fmt.Errorf("provider not started")
	}
	
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		p.updateStats("create_collection", duration, nil)
	}()
	
	p.logger.Debug("Creating namespace", "name", name)
	
	// Pinecone uses namespaces instead of collections
	namespace := name
	
	// Check if namespace already exists
	if _, exists := p.collections[namespace]; exists {
		return fmt.Errorf("namespace %s already exists", namespace)
	}
	
	// Create namespace
	// In Pinecone, namespaces are created implicitly by upserting data
	p.collections[namespace] = &PineconeCollection{
		Name:       name,
		Namespace:  namespace,
		Dimension:  config.Dimension,
		Metric:     config.Metric,
		Size:       0,
		VectorCount: 0,
		Metadata:   config.Metadata,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	p.stats.TotalCollections++
	
	p.logger.Debug("Namespace created successfully", "name", name)
	return nil
}

// GetStats returns provider statistics
func (p *PineconeProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}
	
	// Update uptime
	if p.stats.Uptime > 0 {
		p.stats.Uptime = time.Since(time.Now().Add(-p.stats.Uptime))
	}
	
	// Get index statistics to update vector count
	indexDescription, err := p.index.DescribeIndexStats(ctx)
	if err == nil {
		if indexDescription.TotalVectorCount != nil {
			p.stats.TotalVectors = int64(*indexDescription.TotalVectorCount)
		}
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

// Optimize optimizes Pinecone provider
func (p *PineconeProvider) Optimize(ctx context.Context) error {
	if !p.started {
		return fmt.Errorf("provider not started")
	}
	
	p.logger.Info("Optimizing Pinecone provider...")
	
	// Pinecone handles optimization automatically
	// We can update index configuration if needed
	
	p.logger.Info("Pinecone provider optimization completed")
	return nil
}

// Health returns health status
func (p *PineconeProvider) Health(ctx context.Context) (*HealthStatus, error) {
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
	
	// Check index status
	indexDescription, err := p.index.Describe(ctx)
	if err != nil {
		status.Status = "unhealthy"
		status.Error = fmt.Sprintf("failed to describe index: %v", err)
		return status, nil
	}
	
	if indexDescription.Status.Ready != true {
		status.Status = "degraded"
		status.Error = fmt.Sprintf("index not ready: %v", indexDescription.Status)
		return status, nil
	}
	
	// Add metrics
	status.Metrics["total_vectors"] = float64(p.stats.TotalVectors)
	status.Metrics["total_namespaces"] = float64(len(p.collections))
	status.Metrics["error_count"] = float64(p.stats.ErrorCount)
	status.Dependencies["pinecone_api"] = "connected"
	
	return status, nil
}

// GetName returns provider name
func (p *PineconeProvider) GetName() string {
	return "Pinecone"
}

// GetType returns provider type
func (p *PineconeProvider) GetType() ProviderType {
	return ProviderTypePinecone
}

// GetCapabilities returns provider capabilities
func (p *PineconeProvider) GetCapabilities() []string {
	return []string{
		"vector_storage",
		"vector_search",
		"metadata_filtering",
		"batch_operations",
		"namespace_management",
		"similarity_search",
		"cloud_deployment",
		"managed_service",
		"auto_scaling",
		"high_availability",
		"global_distribution",
	}
}

// GetConfiguration returns provider configuration
func (p *PineconeProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns true for Pinecone (it's a cloud service)
func (p *PineconeProvider) IsCloud() bool {
	return true
}

// GetCostInfo returns cost information for Pinecone
func (p *PineconeProvider) GetCostInfo() *CostInfo {
	// Estimate costs based on current configuration
	storageCost := p.estimateStorageCost()
	computeCost := p.estimateComputeCost()
	
	return &CostInfo{
		StorageCost:   storageCost,
		ComputeCost:   computeCost,
		TransferCost:  p.estimateTransferCost(),
		TotalCost:     storageCost + computeCost + p.estimateTransferCost(),
		Currency:      "USD",
		BillingPeriod:  "monthly",
		FreeTierUsed:   p.isFreeTierUsed(),
		FreeTierLimit:  p.getFreeTierLimit(),
	}
}

// Stop stops Pinecone provider
func (p *PineconeProvider) Stop(ctx context.Context) error {
	p.logger.Info("Stopping Pinecone provider...")
	
	p.started = false
	p.collections = make(map[string]*PineconeCollection)
	p.index = nil
	
	p.logger.Info("Pinecone provider stopped successfully")
	return nil
}

// Private helper methods

func (p *PineconeProvider) validateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	
	if p.config.Environment == "" {
		return fmt.Errorf("environment is required")
	}
	
	if p.config.Dimension <= 0 {
		p.config.Dimension = 1536 // Default for OpenAI embeddings
	}
	
	if p.config.Metric == "" {
		p.config.Metric = "cosine" // Default metric
	}
	
	return nil
}

func (p *PineconeProvider) testConnection(ctx context.Context) error {
	// Try to list indexes to test connection
	_, err := p.client.ListIndexes(ctx)
	return err
}

func (p *PineconeProvider) connectToIndex(ctx context.Context) error {
	// Try to get existing index
	index, err := p.client.DescribeIndex(ctx, p.config.IndexName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// Create new index
			createRequest := &pinecone.CreateIndexRequest{
				Name:      p.config.IndexName,
				Dimension: p.config.Dimension,
				Metric:    pinecone.Metric(p.config.Metric),
				PodType:   p.config.PodType,
				Pods:      &p.config.Pods,
				Replicas:  &p.config.Replicas,
			}
			
			index, err = p.client.CreateIndex(ctx, createRequest)
			if err != nil {
				return fmt.Errorf("failed to create index: %w", err)
			}
			
			// Wait for index to be ready
			err = p.waitForIndexReady(ctx, index.Name())
			if err != nil {
				return fmt.Errorf("failed waiting for index to be ready: %w", err)
			}
		} else {
			return fmt.Errorf("failed to describe index: %w", err)
		}
	}
	
	// Connect to index
	p.index, err = p.client.Index(p.config.IndexName)
	if err != nil {
		return fmt.Errorf("failed to connect to index: %w", err)
	}
	
	p.logger.Info("Connected to index", "name", p.config.IndexName)
	return nil
}

func (p *PineconeProvider) waitForIndexReady(ctx context.Context, indexName string) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(30 * time.Minute)
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for index to be ready")
		case <-ticker.C:
			description, err := p.client.DescribeIndex(ctx, indexName)
			if err != nil {
				return fmt.Errorf("failed to describe index: %w", err)
			}
			
			if description.Status.Ready {
				return nil
			}
		}
	}
}

func (p *PineconeProvider) loadCollections(ctx context.Context) error {
	// Pinecone uses namespaces, which are not pre-created
	// We'll track namespaces we use for metadata
	p.collections = make(map[string]*PineconeCollection)
	
	// Create default namespace if not specified
	if p.config.Namespace != "" {
		p.collections[p.config.Namespace] = &PineconeCollection{
			Name:       "default",
			Namespace:  p.config.Namespace,
			Dimension:  p.config.Dimension,
			Metric:     p.config.Metric,
			Size:       0,
			VectorCount: 0,
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}
	
	p.logger.Info("Loaded namespaces", "count", len(p.collections))
	return nil
}

func (p *PineconeProvider) storeInNamespace(ctx context.Context, namespace string, vectors []*VectorData) error {
	// Prepare upsert request
	var upsertRequests []pinecone.UpsertRequest
	
	for _, vector := range vectors {
		if vector.ID == "" {
			vector.ID = uuid.New().String()
		}
		
		// Convert metadata
		metadata := p.convertToPineconeMetadata(vector)
		
		upsertRequest := pinecone.UpsertRequest{
			ID:       vector.ID,
			Values:   vector.Vector,
			Metadata: metadata,
			Namespace: namespace,
		}
		
		upsertRequests = append(upsertRequests, upsertRequest)
	}
	
	// Batch upsert
	batchSize := p.config.BatchSize
	if batchSize == 0 {
		batchSize = 100
	}
	
	for i := 0; i < len(upsertRequests); i += batchSize {
		end := i + batchSize
		if end > len(upsertRequests) {
			end = len(upsertRequests)
		}
		
		batch := upsertRequests[i:end]
		
		_, err := p.index.Upsert(ctx, &pinecone.UpsertRequest{
			Vectors:  batch,
			Namespace: namespace,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert batch: %w", err)
		}
	}
	
	// Update collection stats
	if collection, exists := p.collections[namespace]; exists {
		collection.VectorCount += int64(len(vectors))
		collection.UpdatedAt = time.Now()
	}
	
	// Update stats
	p.stats.TotalVectors += int64(len(vectors))
	
	return nil
}

func (p *PineconeProvider) fetchVectorByID(ctx context.Context, id string) (*VectorData, error) {
	// Pinecone doesn't support direct fetch by ID
	// We need to use a metadata filter with the ID
	searchRequest := &pinecone.QueryRequest{
		ID:             &id,
		TopK:           1,
		IncludeValues:  true,
		IncludeMetadata: true,
		Namespace:     p.config.Namespace,
	}
	
	response, err := p.index.Query(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vector by ID: %w", err)
	}
	
	if len(response.Matches) == 0 {
		return nil, nil
	}
	
	match := response.Matches[0]
	
	return &VectorData{
		ID:        match.ID,
		Vector:    match.Values,
		Metadata:  p.convertFromPineconeMetadata(match.Metadata),
		Namespace: p.config.Namespace,
		Timestamp: time.Now(),
	}, nil
}

func (p *PineconeProvider) convertToPineconeMetadata(vector *VectorData) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Copy existing metadata
	for k, v := range vector.Metadata {
		// Pinecone metadata has limitations on value types
		if p.isValidMetadataValue(v) {
			metadata[k] = v
		}
	}
	
	// Add built-in fields
	metadata["collection"] = vector.Collection
	metadata["timestamp"] = vector.Timestamp.Unix()
	
	if vector.TTL != nil {
		metadata["ttl"] = vector.TTL.Seconds()
	}
	
	return metadata
}

func (p *PineconeProvider) convertFromPineconeMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return make(map[string]interface{})
	}
	
	result := make(map[string]interface{})
	for k, v := range metadata {
		result[k] = v
	}
	
	return result
}

func (p *PineconeProvider) convertFilters(filters map[string]interface{}) *pinecone.Filter {
	if len(filters) == 0 {
		return nil
	}
	
	// Convert filters to Pinecone filter format
	// This is a simplified conversion
	filterConditions := make(map[string]interface{})
	
	for key, value := range filters {
		if p.isValidMetadataValue(value) {
			filterConditions[key] = map[string]interface{}{
				"$eq": value,
			}
		}
	}
	
	if len(filterConditions) > 0 {
		if len(filterConditions) == 1 {
			for k, v := range filterConditions {
				return &pinecone.Filter{
					Key:   k,
					Value: v,
				}
			}
		} else {
			// Multiple conditions - use AND
			return &pinecone.Filter{
				Key: "$and",
				Value: filterConditions,
			}
		}
	}
	
	return nil
}

func (p *PineconeProvider) isValidMetadataValue(value interface{}) bool {
	// Pinecone metadata supports only certain value types
	switch v := value.(type) {
	case string, float32, float64, int, int32, int64, bool:
		return true
	case []string, []float32, []float64, []int, []int32, []int64, []bool:
		return true
	case map[string]interface{}:
		// Check nested values
		for _, nestedValue := range v {
			if !p.isValidMetadataValue(nestedValue) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (p *PineconeProvider) estimateStorageCost() float64 {
	// Pinecone pricing: $0.70 per GB-month
	storageGB := float64(p.stats.TotalSize) / (1024 * 1024 * 1024)
	return storageGB * 0.70
}

func (p *PineconeProvider) estimateComputeCost() float64 {
	// Pinecone pricing: varies by pod type and number of pods
	// This is a rough estimate based on common configurations
	baseCost := 0.0
	
	switch p.config.PodType {
	case "p1.x1":
		baseCost = 0.116 * float64(p.config.Pods) * 730 // hourly cost * hours in month
	case "p1.x2":
		baseCost = 0.232 * float64(p.config.Pods) * 730
	case "p1.x4":
		baseCost = 0.464 * float64(p.config.Pods) * 730
	case "p1.x8":
		baseCost = 0.928 * float64(p.config.Pods) * 730
	case "p2.x1":
		baseCost = 0.126 * float64(p.config.Pods) * 730
	case "p2.x2":
		baseCost = 0.252 * float64(p.config.Pods) * 730
	case "p2.x4":
		baseCost = 0.504 * float64(p.config.Pods) * 730
	case "p2.x8":
		baseCost = 1.008 * float64(p.config.Pods) * 730
	default:
		baseCost = 0.116 * float64(p.config.Pods) * 730 // Default to p1.x1
	}
	
	return baseCost
}

func (p *PineconeProvider) estimateTransferCost() float64 {
	// Pinecone includes free data transfer within the same cloud provider region
	// Cross-region transfers incur costs
	// This is a rough estimate
	return 0.0 // Assume same region for simplicity
}

func (p *PineconeProvider) isFreeTierUsed() bool {
	// Pinecone has a free tier with limited operations
	// This would require tracking actual usage against free limits
	return p.stats.TotalVectors > 100000 // Rough estimate
}

func (p *PineconeProvider) getFreeTierLimit() float64 {
	return 100000 // Rough estimate for free tier
}

func (p *PineconeProvider) updateStats(operation string, duration time.Duration, err error) {
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

func parsePineconeConfig(config interface{}) (*PineconeConfig, error) {
	pineconeConfig := &PineconeConfig{
		Environment:  "us-west1-gcp",
		Dimension:    1536,
		Metric:       "cosine",
		PodType:      "p1.x1",
		Pods:         1,
		Replicas:     1,
		Timeout:      30 * time.Second,
		MaxRetries:   3,
		BatchSize:    100,
		Compression:  true,
	}
	
	if config != nil {
		// Parse configuration from map or struct
		if configMap, ok := config.(map[string]interface{}); ok {
			if apikey, exists := configMap["api_key"]; exists {
				if apikeyStr, ok := apikey.(string); ok {
					pineconeConfig.APIKey = apikeyStr
				}
			}
			if environment, exists := configMap["environment"]; exists {
				if environmentStr, ok := environment.(string); ok {
					pineconeConfig.Environment = environmentStr
				}
			}
			if indexName, exists := configMap["index_name"]; exists {
				if indexNameStr, ok := indexName.(string); ok {
					pineconeConfig.IndexName = indexNameStr
				}
			}
			if dimension, exists := configMap["dimension"]; exists {
				if dimensionInt, ok := dimension.(int); ok {
					pineconeConfig.Dimension = dimensionInt
				}
				if dimensionStr, ok := dimension.(string); ok {
					if dimensionInt, err := strconv.Atoi(dimensionStr); err == nil {
						pineconeConfig.Dimension = dimensionInt
					}
				}
			}
			if metric, exists := configMap["metric"]; exists {
				if metricStr, ok := metric.(string); ok {
					pineconeConfig.Metric = metricStr
				}
			}
			if podType, exists := configMap["pod_type"]; exists {
				if podTypeStr, ok := podType.(string); ok {
					pineconeConfig.PodType = podTypeStr
				}
			}
			if pods, exists := configMap["pods"]; exists {
				if podsInt, ok := pods.(int); ok {
					pineconeConfig.Pods = podsInt
				}
			}
			if replicas, exists := configMap["replicas"]; exists {
				if replicasInt, ok := replicas.(int); ok {
					pineconeConfig.Replicas = replicasInt
				}
			}
			if namespace, exists := configMap["namespace"]; exists {
				if namespaceStr, ok := namespace.(string); ok {
					pineconeConfig.Namespace = namespaceStr
				}
			}
		}
	}
	
	return pineconeConfig, nil
}

// Import required packages
import (
	"dev.helix.code/internal/logging"
)