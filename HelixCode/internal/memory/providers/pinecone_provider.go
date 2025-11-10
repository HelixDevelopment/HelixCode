package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"dev.helix.code/internal/logging"
)

// PineconeProvider implements VectorProvider for Pinecone
type PineconeProvider struct {
	config      *PineconeConfig
	logger      *logging.Logger
	httpClient  *http.Client
	mu          sync.RWMutex
	initialized bool
	started     bool
}

// PineconeConfig holds configuration for Pinecone
type PineconeConfig struct {
	APIKey  string `json:"api_key"`
	Index   string `json:"index"`
	Project string `json:"project"`
	Host    string `json:"host"`
}

// NewPineconeProvider creates a new Pinecone provider
func NewPineconeProvider(config map[string]interface{}) (VectorProvider, error) {
	cfg := &PineconeConfig{
		APIKey:  getStringConfig(config, "api_key", ""),
		Index:   getStringConfig(config, "index", ""),
		Project: getStringConfig(config, "project", ""),
		Host:    getStringConfig(config, "host", ""),
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Pinecone API key is required")
	}

	if cfg.Index == "" {
		return nil, fmt.Errorf("Pinecone index name is required")
	}

	logger := logging.NewLoggerWithName("pinecone_provider")

	return &PineconeProvider{
		config: cfg,
		logger: logger,
	}, nil
}

// testConnection tests the connection to Pinecone
func (p *PineconeProvider) testConnection(ctx context.Context) error {
	// For Pinecone, we can test by trying to describe the index
	url := fmt.Sprintf("https://api.pinecone.io/indexes/%s", p.config.Index)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Api-Key", p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Pinecone connection test failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get host if not provided
	if p.config.Host == "" {
		var indexResp struct {
			Host string `json:"host"`
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(body, &indexResp); err != nil {
			return err
		}
		p.config.Host = indexResp.Host
	}

	return nil
}

// Initialize initializes the Pinecone provider
func (p *PineconeProvider) Initialize(ctx context.Context, config interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Initializing Pinecone provider index=%s", p.config.Index)

	// Initialize HTTP client
	p.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Test connection to Pinecone
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to Pinecone: %w", err)
	}

	p.initialized = true
	p.logger.Info("Pinecone provider initialized successfully")
	return nil
}

// Start starts the Pinecone provider
func (p *PineconeProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	p.logger.Info("Starting Pinecone provider")

	// Test connection before starting
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to Pinecone during start: %w", err)
	}

	p.started = true
	p.logger.Info("Pinecone provider started successfully")
	return nil
}

// Stop stops the Pinecone provider
func (p *PineconeProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Stopping Pinecone provider")

	// Close HTTP client connections
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}

	p.started = false
	p.logger.Info("Pinecone provider stopped successfully")
	return nil
}

// GetName returns the provider name
func (p *PineconeProvider) GetName() string {
	return "pinecone"
}

// GetType returns the provider type
func (p *PineconeProvider) GetType() string {
	return string(ProviderTypePinecone)
}

// GetCapabilities returns provider capabilities
func (p *PineconeProvider) GetCapabilities() []string {
	return []string{"vector_storage", "similarity_search", "metadata_filtering", "namespaces"}
}

// GetConfiguration returns the current configuration
func (p *PineconeProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether this is a cloud provider
func (p *PineconeProvider) IsCloud() bool {
	return true
}

// GetCostInfo returns cost information
func (p *PineconeProvider) GetCostInfo() *CostInfo {
	return &CostInfo{
		Currency:      "USD",
		ComputeCost:   0.0,
		TransferCost:  0.0,
		StorageCost:   0.0,
		TotalCost:     0.0,
		BillingPeriod: "monthly",
	}
}

// Store stores vectors in Pinecone
func (p *PineconeProvider) Store(ctx context.Context, vectors []*VectorData) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	if len(vectors) == 0 {
		return nil
	}

	p.logger.Info("Storing %d vectors in Pinecone", len(vectors))

	// Group vectors by namespace
	namespaces := make(map[string][]*VectorData)
	for _, v := range vectors {
		namespace := v.Collection
		if namespace == "" {
			namespace = "default"
		}
		namespaces[namespace] = append(namespaces[namespace], v)
	}

	// Store in each namespace
	for namespace, vecs := range namespaces {
		if err := p.storeInNamespace(ctx, namespace, vecs); err != nil {
			return fmt.Errorf("failed to store in namespace %s: %w", namespace, err)
		}
	}

	return nil
}

// storeInNamespace stores vectors in a specific namespace
func (p *PineconeProvider) storeInNamespace(ctx context.Context, namespace string, vectors []*VectorData) error {
	// Prepare data for Pinecone
	pineconeVectors := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		vector := map[string]interface{}{
			"id":     v.ID,
			"values": v.Vector,
		}

		if v.Metadata != nil {
			vector["metadata"] = v.Metadata
		}

		pineconeVectors[i] = vector
	}

	data := map[string]interface{}{
		"vectors":   pineconeVectors,
		"namespace": namespace,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/vectors/upsert", p.config.Host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Api-Key", p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Pinecone upsert failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Retrieve retrieves vectors by IDs
func (p *PineconeProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Retrieving %d vectors from Pinecone", len(ids))

	// Pinecone doesn't have a direct retrieve by IDs API
	// We need to use query with ID, but that's inefficient for multiple IDs
	// For now, return not supported
	return nil, fmt.Errorf("Pinecone does not support batch retrieval by IDs")
}

// Update updates a vector
func (p *PineconeProvider) Update(ctx context.Context, id string, vector *VectorData) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	p.logger.Info("Updating vector %s in Pinecone", id)

	// Update is same as upsert in Pinecone
	return p.Store(ctx, []*VectorData{vector})
}

// Delete deletes vectors by IDs
func (p *PineconeProvider) Delete(ctx context.Context, ids []string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	p.logger.Info("Deleting %d vectors from Pinecone", len(ids))

	if len(ids) == 0 {
		return nil
	}

	// Pinecone delete requires namespace, assume default
	namespace := "default"

	data := map[string]interface{}{
		"ids":       ids,
		"namespace": namespace,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/vectors/delete", p.config.Host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Api-Key", p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Pinecone delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Search performs vector similarity search
func (p *PineconeProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Searching vectors in Pinecone with top_k=%d", query.TopK)

	namespace := query.Collection
	if namespace == "" {
		namespace = "default"
	}

	data := map[string]interface{}{
		"vector":          query.Vector,
		"topK":            query.TopK,
		"namespace":       namespace,
		"includeValues":   query.IncludeVector,
		"includeMetadata": true,
	}

	if query.Filters != nil {
		data["filter"] = query.Filters
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s/query", p.config.Host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Api-Key", p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Pinecone query failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pineconeResp struct {
		Matches []struct {
			ID       string                 `json:"id"`
			Score    float64                `json:"score"`
			Values   []float64              `json:"values,omitempty"`
			Metadata map[string]interface{} `json:"metadata,omitempty"`
		} `json:"matches"`
	}

	if err := json.Unmarshal(body, &pineconeResp); err != nil {
		return nil, err
	}

	results := make([]*VectorSearchResultItem, len(pineconeResp.Matches))
	for i, match := range pineconeResp.Matches {
		results[i] = &VectorSearchResultItem{
			ID:       match.ID,
			Score:    match.Score,
			Distance: 1.0 - match.Score, // Convert similarity to distance
			Metadata: match.Metadata,
		}

		if query.IncludeVector && match.Values != nil {
			results[i].Vector = match.Values
		}
	}

	return &VectorSearchResult{
		Results: results,
		Total:   len(results),
		Query:   query,
	}, nil
}

// FindSimilar finds similar vectors
func (p *PineconeProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Finding %d similar vectors in Pinecone", k)

	query := &VectorQuery{
		Vector:        embedding,
		TopK:          k,
		Filters:       filters,
		IncludeVector: false,
	}

	result, err := p.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	similarities := make([]*VectorSimilarityResult, len(result.Results))
	for i, item := range result.Results {
		similarities[i] = &VectorSimilarityResult{
			ID:    item.ID,
			Score: item.Score,
		}
	}

	return similarities, nil
}

// BatchFindSimilar performs batch similarity search
func (p *PineconeProvider) BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Batch finding similar vectors for %d queries in Pinecone", len(queries))

	results := make([][]*VectorSimilarityResult, len(queries))
	for i, query := range queries {
		similar, err := p.FindSimilar(ctx, query, k, nil)
		if err != nil {
			return nil, err
		}
		results[i] = similar
	}

	return results, nil
}

// CreateCollection creates a new collection (namespace in Pinecone)
func (p *PineconeProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Creating collection %s in Pinecone", name)

	// In Pinecone, collections are namespaces, and they're created automatically on upsert
	// No explicit creation needed
	return nil
}

// DeleteCollection deletes a collection
func (p *PineconeProvider) DeleteCollection(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting collection %s from Pinecone", name)

	// Pinecone doesn't support deleting namespaces via API
	return fmt.Errorf("Pinecone does not support deleting namespaces")
}

// ListCollections lists all collections
func (p *PineconeProvider) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Listing collections in Pinecone")

	// Pinecone doesn't provide API to list namespaces
	// Return empty list
	return []*CollectionInfo{}, nil
}

// GetCollection gets collection information
func (p *PineconeProvider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting collection %s info from Pinecone", name)

	// Pinecone doesn't provide detailed namespace info
	return &CollectionInfo{Name: name}, nil
}

// CreateIndex creates an index
func (p *PineconeProvider) CreateIndex(ctx context.Context, collection string, config *IndexConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Creating index %s in collection %s in Pinecone", config.Name, collection)

	// Pinecone indexes are managed at the account level, not per collection
	return fmt.Errorf("Pinecone does not support manual index creation")
}

// DeleteIndex deletes an index
func (p *PineconeProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting index %s from collection %s in Pinecone", name, collection)

	// Pinecone indexes are managed at the account level
	return fmt.Errorf("Pinecone does not support manual index deletion")
}

// ListIndexes lists indexes in a collection
func (p *PineconeProvider) ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Listing indexes in collection %s in Pinecone", collection)

	// Pinecone uses automatic indexing
	return []*IndexInfo{
		{
			Name:  "default",
			Type:  "hnsw",
			State: "active",
		},
	}, nil
}

// AddMetadata adds metadata to a vector
func (p *PineconeProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Adding metadata to vector %s in Pinecone", id)

	// To add metadata, we need to get current vector, merge metadata, and upsert
	// But Pinecone doesn't support getting a single vector
	return fmt.Errorf("Pinecone does not support adding metadata to existing vectors")
}

// UpdateMetadata updates metadata
func (p *PineconeProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Updating metadata for vector %s in Pinecone", id)

	// Similar to AddMetadata
	return fmt.Errorf("Pinecone does not support updating metadata for existing vectors")
}

// GetMetadata gets metadata for vectors
func (p *PineconeProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting metadata for %d vectors from Pinecone", len(ids))

	// Pinecone doesn't support batch metadata retrieval
	return nil, fmt.Errorf("Pinecone does not support batch metadata retrieval")
}

// DeleteMetadata deletes metadata
func (p *PineconeProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting metadata for %d vectors in Pinecone", len(ids))

	// Not supported
	return fmt.Errorf("Pinecone does not support deleting metadata")
}

// GetStats returns provider statistics
func (p *PineconeProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting stats from Pinecone provider")

	// Get index stats
	url := fmt.Sprintf("https://api.pinecone.io/indexes/%s", p.config.Index)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Api-Key", p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.logger.Warn("Failed to get index stats: %s", string(body))
		return &ProviderStats{
			Name:             p.GetName(),
			Type:             p.GetType(),
			Status:           "operational",
			TotalVectors:     0,
			TotalCollections: 0,
			TotalSize:        0,
			LastHealthCheck:  time.Now(),
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var indexStats struct {
		TotalVectorCount int64 `json:"totalVectorCount"`
		Dimension        int   `json:"dimension"`
	}

	if err := json.Unmarshal(body, &indexStats); err != nil {
		return nil, err
	}

	return &ProviderStats{
		Name:             p.GetName(),
		Type:             p.GetType(),
		Status:           "operational",
		TotalVectors:     indexStats.TotalVectorCount,
		TotalCollections: 1, // Index level
		TotalSize:        0, // Not available
		LastHealthCheck:  time.Now(),
	}, nil
}

// Optimize optimizes the provider
func (p *PineconeProvider) Optimize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Optimizing Pinecone provider")

	// Pinecone handles optimization automatically
	return nil
}

// Backup creates a backup
func (p *PineconeProvider) Backup(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Creating backup at %s for Pinecone provider", path)

	// Pinecone doesn't provide backup functionality
	return fmt.Errorf("Pinecone does not support backup operations")
}

// Restore restores from backup
func (p *PineconeProvider) Restore(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Restoring from backup at %s for Pinecone provider", path)

	// Pinecone doesn't provide restore functionality
	return fmt.Errorf("Pinecone does not support restore operations")
}

// Health checks provider health
func (p *PineconeProvider) Health(ctx context.Context) (*HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Checking health of Pinecone provider")

	start := time.Now()
	err := p.testConnection(ctx)
	responseTime := time.Since(start)

	status := "healthy"
	if err != nil {
		status = "unhealthy"
		p.logger.Warn("Health check failed: %v", err)
	}

	return &HealthStatus{
		Status:       status,
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
	}, nil
}
