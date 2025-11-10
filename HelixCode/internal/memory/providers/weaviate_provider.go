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

// WeaviateProvider implements VectorProvider for Weaviate
type WeaviateProvider struct {
	config      *WeaviateConfig
	logger      *logging.Logger
	httpClient  *http.Client
	mu          sync.RWMutex
	initialized bool
	started     bool
}

// WeaviateConfig holds configuration for Weaviate
type WeaviateConfig struct {
	URL       string `json:"url"`
	APIKey    string `json:"api_key"`
	Class     string `json:"class"`
	BatchSize int    `json:"batch_size"`
}

// NewWeaviateProvider creates a new Weaviate provider
func NewWeaviateProvider(config map[string]interface{}) (VectorProvider, error) {
	cfg := &WeaviateConfig{
		URL:       getStringConfig(config, "url", "http://localhost:8080"),
		APIKey:    getStringConfig(config, "api_key", ""),
		Class:     getStringConfig(config, "class", "Vector"),
		BatchSize: getIntConfig(config, "batch_size", 100),
	}

	logger := logging.NewLoggerWithName("weaviate_provider")

	return &WeaviateProvider{
		config: cfg,
		logger: logger,
	}, nil
}

// testConnection tests the connection to Weaviate
func (p *WeaviateProvider) testConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/meta", p.config.URL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Weaviate meta failed with status: %d", resp.StatusCode)
	}

	return nil
}

// Initialize initializes the Weaviate provider
func (p *WeaviateProvider) Initialize(ctx context.Context, config interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Initializing Weaviate provider url=%s class=%s", p.config.URL, p.config.Class)

	// Initialize HTTP client
	p.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Test connection to Weaviate
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to Weaviate: %w", err)
	}

	p.initialized = true
	p.logger.Info("Weaviate provider initialized successfully")
	return nil
}

// Start starts the Weaviate provider
func (p *WeaviateProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	p.logger.Info("Starting Weaviate provider")

	// TODO: Implement startup logic

	p.started = true
	p.logger.Info("Weaviate provider started successfully")
	return nil
}

// Stop stops the Weaviate provider
func (p *WeaviateProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Stopping Weaviate provider")

	// TODO: Implement shutdown logic

	p.started = false
	p.logger.Info("Weaviate provider stopped successfully")
	return nil
}

// GetName returns the provider name
func (p *WeaviateProvider) GetName() string {
	return "weaviate"
}

// GetType returns the provider type
func (p *WeaviateProvider) GetType() string {
	return string(ProviderTypeWeaviate)
}

// GetCapabilities returns provider capabilities
func (p *WeaviateProvider) GetCapabilities() []string {
	return []string{"vector_storage", "similarity_search", "metadata_filtering"}
}

// GetConfiguration returns the current configuration
func (p *WeaviateProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether this is a cloud provider
func (p *WeaviateProvider) IsCloud() bool {
	return false // Weaviate can be self-hosted or cloud
}

// GetCostInfo returns cost information
func (p *WeaviateProvider) GetCostInfo() *CostInfo {
	return &CostInfo{
		Currency:      "USD",
		ComputeCost:   0.0,
		TransferCost:  0.0,
		StorageCost:   0.0,
		TotalCost:     0.0,
		BillingPeriod: "monthly",
	}
}

// Store stores vectors in Weaviate
func (p *WeaviateProvider) Store(ctx context.Context, vectors []*VectorData) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	if len(vectors) == 0 {
		return nil
	}

	p.logger.Info("Storing %d vectors in Weaviate", len(vectors))

	// Ensure class exists
	if err := p.ensureClass(ctx, vectors[0]); err != nil {
		return err
	}

	// Store vectors in batches
	batchSize := p.config.BatchSize
	for i := 0; i < len(vectors); i += batchSize {
		end := i + batchSize
		if end > len(vectors) {
			end = len(vectors)
		}

		batch := vectors[i:end]
		if err := p.storeBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to store batch %d-%d: %w", i, end, err)
		}
	}

	return nil
}

// ensureClass ensures the Weaviate class exists
func (p *WeaviateProvider) ensureClass(ctx context.Context, sampleVector *VectorData) error {
	// Check if class exists
	url := fmt.Sprintf("%s/v1/schema/classes/%s", p.config.URL, p.config.Class)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Class exists
		return nil
	}

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to check class existence: %s", string(body))
	}

	// Create class
	classDef := map[string]interface{}{
		"class": p.config.Class,
		"properties": []map[string]interface{}{
			{
				"name":        "id",
				"dataType":    []string{"string"},
				"description": "Unique identifier",
			},
			{
				"name":        "timestamp",
				"dataType":    []string{"date"},
				"description": "Creation timestamp",
			},
		},
		"vectorizer": "none", // We'll provide vectors manually
		"vectorIndexConfig": map[string]interface{}{
			"distance": "cosine",
		},
	}

	jsonData, err := json.Marshal(classDef)
	if err != nil {
		return err
	}

	url = fmt.Sprintf("%s/v1/schema/classes", p.config.URL)
	req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err = p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create class: %s", string(body))
	}

	return nil
}

// storeBatch stores a batch of vectors
func (p *WeaviateProvider) storeBatch(ctx context.Context, vectors []*VectorData) error {
	objects := make([]map[string]interface{}, len(vectors))

	for i, v := range vectors {
		properties := make(map[string]interface{})
		for k, val := range v.Metadata {
			properties[k] = val
		}
		properties["id"] = v.ID
		properties["timestamp"] = v.Timestamp.Format(time.RFC3339)

		objects[i] = map[string]interface{}{
			"class":      p.config.Class,
			"properties": properties,
			"vector":     v.Vector,
		}
	}

	data := map[string]interface{}{
		"objects": objects,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v1/objects", p.config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Weaviate batch store failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Retrieve retrieves vectors by IDs
func (p *WeaviateProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Retrieving %d vectors from Weaviate", len(ids))

	// TODO: Implement actual vector retrieval

	return []*VectorData{}, nil
}

// Update updates a vector
func (p *WeaviateProvider) Update(ctx context.Context, id string, vector *VectorData) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	p.logger.Info("Updating vector %s in Weaviate", id)

	// TODO: Implement actual vector update

	return nil
}

// Delete deletes vectors by IDs
func (p *WeaviateProvider) Delete(ctx context.Context, ids []string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	p.logger.Info("Deleting %d vectors from Weaviate", len(ids))

	// TODO: Implement actual vector deletion

	return nil
}

// Search performs vector similarity search
func (p *WeaviateProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Searching vectors in Weaviate with top_k=%d", query.TopK)

	// Build GraphQL query
	graphqlQuery := fmt.Sprintf(`
	{
	  Get {
		%s(
		  nearVector: {vector: %s}
		  limit: %d
		) {
		  id
		  _additional {
			distance
		  }
		}
	  }
	}`, p.config.Class, vectorToGraphQLString(query.Vector), query.TopK)

	data := map[string]interface{}{
		"query": graphqlQuery,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/graphql", p.config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Weaviate GraphQL query failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse response
	var gqlResp struct {
		Data struct {
			Get map[string][]struct {
				ID         string `json:"id"`
				Additional struct {
					Distance float64 `json:"distance"`
				} `json:"_additional"`
			} `json:"Get"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, err
	}

	results := []*VectorSearchResultItem{}
	if objects, ok := gqlResp.Data.Get[p.config.Class]; ok {
		for _, obj := range objects {
			item := &VectorSearchResultItem{
				ID:       obj.ID,
				Distance: obj.Additional.Distance,
				Score:    1.0 - obj.Additional.Distance, // Convert distance to similarity
				Metadata: make(map[string]interface{}),
			}
			results = append(results, item)
		}
	}

	return &VectorSearchResult{
		Results: results,
		Total:   len(results),
		Query:   query,
	}, nil
}

// vectorToGraphQLString converts a vector to GraphQL string format
func vectorToGraphQLString(vec []float64) string {
	if len(vec) == 0 {
		return "[]"
	}

	result := "["
	for i, v := range vec {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%.6f", v)
	}
	result += "]"
	return result
}

// FindSimilar finds similar vectors
func (p *WeaviateProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Finding %d similar vectors in Weaviate", k)

	// TODO: Implement actual similarity search

	return []*VectorSimilarityResult{}, nil
}

// BatchFindSimilar performs batch similarity search
func (p *WeaviateProvider) BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Batch finding similar vectors for %d queries in Weaviate", len(queries))

	// TODO: Implement actual batch similarity search

	return [][]*VectorSimilarityResult{}, nil
}

// CreateCollection creates a new collection
func (p *WeaviateProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Creating collection %s in Weaviate", name)

	// TODO: Implement actual collection creation

	return nil
}

// DeleteCollection deletes a collection
func (p *WeaviateProvider) DeleteCollection(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting collection %s from Weaviate", name)

	// TODO: Implement actual collection deletion

	return nil
}

// ListCollections lists all collections
func (p *WeaviateProvider) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Listing collections in Weaviate")

	// TODO: Implement actual collection listing

	return []*CollectionInfo{}, nil
}

// GetCollection gets collection information
func (p *WeaviateProvider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting collection %s info from Weaviate", name)

	// TODO: Implement actual collection info retrieval

	return &CollectionInfo{Name: name}, nil
}

// CreateIndex creates an index
func (p *WeaviateProvider) CreateIndex(ctx context.Context, collection string, config *IndexConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Creating index %s in collection %s in Weaviate", config.Name, collection)

	// TODO: Implement actual index creation

	return nil
}

// DeleteIndex deletes an index
func (p *WeaviateProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting index %s from collection %s in Weaviate", name, collection)

	// TODO: Implement actual index deletion

	return nil
}

// ListIndexes lists indexes in a collection
func (p *WeaviateProvider) ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Listing indexes in collection %s in Weaviate", collection)

	// TODO: Implement actual index listing

	return []*IndexInfo{}, nil
}

// AddMetadata adds metadata to a vector
func (p *WeaviateProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Adding metadata to vector %s in Weaviate", id)

	// TODO: Implement actual metadata addition

	return nil
}

// UpdateMetadata updates metadata
func (p *WeaviateProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Updating metadata for vector %s in Weaviate", id)

	// TODO: Implement actual metadata update

	return nil
}

// GetMetadata gets metadata for vectors
func (p *WeaviateProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting metadata for %d vectors from Weaviate", len(ids))

	// TODO: Implement actual metadata retrieval

	return map[string]map[string]interface{}{}, nil
}

// DeleteMetadata deletes metadata
func (p *WeaviateProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting metadata for %d vectors in Weaviate", len(ids))

	// TODO: Implement actual metadata deletion

	return nil
}

// GetStats returns provider statistics
func (p *WeaviateProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting stats from Weaviate provider")

	// TODO: Implement actual stats retrieval

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

// Optimize optimizes the provider
func (p *WeaviateProvider) Optimize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Optimizing Weaviate provider")

	// TODO: Implement actual optimization

	return nil
}

// Backup creates a backup
func (p *WeaviateProvider) Backup(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Creating backup at %s for Weaviate provider", path)

	// TODO: Implement actual backup

	return nil
}

// Restore restores from backup
func (p *WeaviateProvider) Restore(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Restoring from backup at %s for Weaviate provider", path)

	// TODO: Implement actual restore

	return nil
}

// Health checks provider health
func (p *WeaviateProvider) Health(ctx context.Context) (*HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Checking health of Weaviate provider")

	// TODO: Implement actual health check

	return &HealthStatus{
		Status:       "healthy",
		ResponseTime: time.Millisecond * 100,
		Timestamp:    time.Now(),
	}, nil
}

// Helper functions
func getStringConfig(config map[string]interface{}, key, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getIntConfig(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		if num, ok := val.(int); ok {
			return num
		}
	}
	return defaultValue
}
