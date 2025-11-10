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

// QdrantProvider implements VectorProvider for Qdrant
type QdrantProvider struct {
	config      *QdrantConfig
	logger      *logging.Logger
	httpClient  *http.Client
	mu          sync.RWMutex
	initialized bool
	started     bool
}

// QdrantConfig holds configuration for Qdrant
type QdrantConfig struct {
	URL     string `json:"url"`
	APIKey  string `json:"api_key"`
	Timeout int    `json:"timeout"`
}

// NewQdrantProvider creates a new Qdrant provider
func NewQdrantProvider(config map[string]interface{}) (VectorProvider, error) {
	cfg := &QdrantConfig{
		URL:     getStringConfig(config, "url", "http://localhost:6333"),
		APIKey:  getStringConfig(config, "api_key", ""),
		Timeout: getIntConfig(config, "timeout", 30),
	}

	logger := logging.NewLoggerWithName("qdrant_provider")

	return &QdrantProvider{
		config: cfg,
		logger: logger,
	}, nil
}

// testConnection tests the connection to Qdrant
func (p *QdrantProvider) testConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", p.config.URL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qdrant health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Initialize initializes the Qdrant provider
func (p *QdrantProvider) Initialize(ctx context.Context, config interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Initializing Qdrant provider url=%s", p.config.URL)

	// Initialize HTTP client
	p.httpClient = &http.Client{
		Timeout: time.Duration(p.config.Timeout) * time.Second,
	}

	// Test connection to Qdrant
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	p.initialized = true
	p.logger.Info("Qdrant provider initialized successfully")
	return nil
}

// Start starts the Qdrant provider
func (p *QdrantProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	p.logger.Info("Starting Qdrant provider")

	// Test connection before starting
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to Qdrant during start: %w", err)
	}

	p.started = true
	p.logger.Info("Qdrant provider started successfully")
	return nil
}

// Stop stops the Qdrant provider
func (p *QdrantProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Stopping Qdrant provider")

	// Close HTTP client connections
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}

	p.started = false
	p.logger.Info("Qdrant provider stopped successfully")
	return nil
}

// GetName returns the provider name
func (p *QdrantProvider) GetName() string {
	return "qdrant"
}

// GetType returns the provider type
func (p *QdrantProvider) GetType() string {
	return string(ProviderTypeQdrant)
}

// GetCapabilities returns provider capabilities
func (p *QdrantProvider) GetCapabilities() []string {
	return []string{"vector_storage", "similarity_search", "metadata_filtering", "collections"}
}

// GetConfiguration returns the current configuration
func (p *QdrantProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether this is a cloud provider
func (p *QdrantProvider) IsCloud() bool {
	return false // Qdrant can be self-hosted or cloud
}

// GetCostInfo returns cost information
func (p *QdrantProvider) GetCostInfo() *CostInfo {
	return &CostInfo{
		Currency:      "USD",
		ComputeCost:   0.0,
		TransferCost:  0.0,
		StorageCost:   0.0,
		TotalCost:     0.0,
		BillingPeriod: "monthly",
	}
}

// Store stores vectors in Qdrant
func (p *QdrantProvider) Store(ctx context.Context, vectors []*VectorData) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	if len(vectors) == 0 {
		return nil
	}

	p.logger.Info("Storing %d vectors in Qdrant", len(vectors))

	// Group vectors by collection
	collections := make(map[string][]*VectorData)
	for _, v := range vectors {
		collection := v.Collection
		if collection == "" {
			collection = "default"
		}
		collections[collection] = append(collections[collection], v)
	}

	// Store in each collection
	for collection, vecs := range collections {
		if err := p.storeInCollection(ctx, collection, vecs); err != nil {
			return fmt.Errorf("failed to store in collection %s: %w", collection, err)
		}
	}

	return nil
}

// storeInCollection stores vectors in a specific collection
func (p *QdrantProvider) storeInCollection(ctx context.Context, collection string, vectors []*VectorData) error {
	// Ensure collection exists
	if err := p.ensureCollection(ctx, collection, vectors[0]); err != nil {
		return err
	}

	// Prepare data for Qdrant
	points := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		point := map[string]interface{}{
			"id":      v.ID,
			"vector":  v.Vector,
			"payload": v.Metadata,
		}
		points[i] = point
	}

	data := map[string]interface{}{
		"points": points,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s/points", p.config.URL, collection)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qdrant upsert failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ensureCollection ensures a collection exists
func (p *QdrantProvider) ensureCollection(ctx context.Context, name string, sampleVector *VectorData) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/collections/%s", p.config.URL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Collection exists
		return nil
	}

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to check collection existence: %s", string(body))
	}

	// Create collection
	createData := map[string]interface{}{
		"name": name,
		"vectors": map[string]interface{}{
			"size":     len(sampleVector.Vector),
			"distance": "Cosine",
		},
	}

	jsonData, err := json.Marshal(createData)
	if err != nil {
		return err
	}

	url = fmt.Sprintf("%s/collections/%s", p.config.URL, name)
	req, err = http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err = p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", string(body))
	}

	return nil
}

// Retrieve retrieves vectors by IDs
func (p *QdrantProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Retrieving %d vectors from Qdrant", len(ids))

	// Qdrant retrieve by IDs
	if len(ids) == 0 {
		return []*VectorData{}, nil
	}

	collection := "default"

	data := map[string]interface{}{
		"ids":          ids,
		"with_payload": true,
		"with_vectors": true,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/collections/%s/points", p.config.URL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Qdrant retrieve failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var qdrantResp struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Vector  []float64              `json:"vector"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &qdrantResp); err != nil {
		return nil, err
	}

	results := make([]*VectorData, len(qdrantResp.Result))
	for i, point := range qdrantResp.Result {
		id := fmt.Sprintf("%v", point.ID)
		results[i] = &VectorData{
			ID:         id,
			Vector:     point.Vector,
			Metadata:   point.Payload,
			Collection: collection,
			Timestamp:  time.Now(),
		}
	}

	return results, nil
}

// Update updates a vector
func (p *QdrantProvider) Update(ctx context.Context, id string, vector *VectorData) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	p.logger.Info("Updating vector %s in Qdrant", id)

	// Update is same as upsert in Qdrant
	return p.Store(ctx, []*VectorData{vector})
}

// Delete deletes vectors by IDs
func (p *QdrantProvider) Delete(ctx context.Context, ids []string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	p.logger.Info("Deleting %d vectors from Qdrant", len(ids))

	if len(ids) == 0 {
		return nil
	}

	collection := "default"

	data := map[string]interface{}{
		"points": ids,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", p.config.URL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qdrant delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Search performs vector similarity search
func (p *QdrantProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Searching vectors in Qdrant with top_k=%d", query.TopK)

	collection := query.Collection
	if collection == "" {
		collection = "default"
	}

	data := map[string]interface{}{
		"vector":       query.Vector,
		"limit":        query.TopK,
		"with_payload": true,
		"with_vectors": query.IncludeVector,
	}

	if query.Filters != nil {
		data["filter"] = query.Filters
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", p.config.URL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Qdrant search failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var qdrantResp struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float64                `json:"score"`
			Vector  []float64              `json:"vector,omitempty"`
			Payload map[string]interface{} `json:"payload,omitempty"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &qdrantResp); err != nil {
		return nil, err
	}

	results := make([]*VectorSearchResultItem, len(qdrantResp.Result))
	for i, hit := range qdrantResp.Result {
		id := fmt.Sprintf("%v", hit.ID)
		results[i] = &VectorSearchResultItem{
			ID:       id,
			Score:    hit.Score,
			Distance: 1.0 - hit.Score,
			Metadata: hit.Payload,
		}

		if query.IncludeVector && hit.Vector != nil {
			results[i].Vector = hit.Vector
		}
	}

	return &VectorSearchResult{
		Results: results,
		Total:   len(results),
		Query:   query,
	}, nil
}

// FindSimilar finds similar vectors
func (p *QdrantProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Finding %d similar vectors in Qdrant", k)

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
func (p *QdrantProvider) BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Batch finding similar vectors for %d queries in Qdrant", len(queries))

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

// CreateCollection creates a new collection
func (p *QdrantProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Creating collection %s in Qdrant", name)

	createData := map[string]interface{}{
		"name": name,
		"vectors": map[string]interface{}{
			"size":     config.Dimension,
			"distance": "Cosine",
		},
	}

	jsonData, err := json.Marshal(createData)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s", p.config.URL, name)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", string(body))
	}

	return nil
}

// DeleteCollection deletes a collection
func (p *QdrantProvider) DeleteCollection(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting collection %s from Qdrant", name)

	url := fmt.Sprintf("%s/collections/%s", p.config.URL, name)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qdrant delete collection failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListCollections lists all collections
func (p *QdrantProvider) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Listing collections in Qdrant")

	url := fmt.Sprintf("%s/collections", p.config.URL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Qdrant list collections failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var qdrantResp struct {
		Result []struct {
			Name string `json:"name"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &qdrantResp); err != nil {
		return nil, err
	}

	result := make([]*CollectionInfo, len(qdrantResp.Result))
	for i, coll := range qdrantResp.Result {
		result[i] = &CollectionInfo{Name: coll.Name}
	}

	return result, nil
}

// GetCollection gets collection information
func (p *QdrantProvider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting collection %s info from Qdrant", name)

	url := fmt.Sprintf("%s/collections/%s", p.config.URL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("collection %s not found", name)
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Qdrant get collection failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var qdrantResp struct {
		Result struct {
			Name string `json:"name"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &qdrantResp); err != nil {
		return nil, err
	}

	return &CollectionInfo{
		Name: qdrantResp.Result.Name,
	}, nil
}

// CreateIndex creates an index
func (p *QdrantProvider) CreateIndex(ctx context.Context, collection string, config *IndexConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Creating index %s in collection %s in Qdrant", config.Name, collection)

	// Qdrant handles indexing automatically
	return fmt.Errorf("Qdrant does not support manual index creation")
}

// DeleteIndex deletes an index
func (p *QdrantProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting index %s from collection %s in Qdrant", name, collection)

	// Qdrant handles indexing automatically
	return fmt.Errorf("Qdrant does not support manual index deletion")
}

// ListIndexes lists indexes in a collection
func (p *QdrantProvider) ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Listing indexes in collection %s in Qdrant", collection)

	// Qdrant uses automatic indexing
	return []*IndexInfo{
		{
			Name:  "default",
			Type:  "hnsw",
			State: "active",
		},
	}, nil
}

// AddMetadata adds metadata to a vector
func (p *QdrantProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Adding metadata to vector %s in Qdrant", id)

	// To add metadata, we need to update the point
	collection := "default"

	data := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      id,
				"payload": metadata,
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s/points/payload", p.config.URL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qdrant add metadata failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateMetadata updates metadata
func (p *QdrantProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Updating metadata for vector %s in Qdrant", id)

	// Update metadata directly
	collection := "default"

	data := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      id,
				"payload": metadata,
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s/points/payload", p.config.URL, collection)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qdrant update metadata failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetMetadata gets metadata for vectors
func (p *QdrantProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting metadata for %d vectors from Qdrant", len(ids))

	if len(ids) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	collection := "default"

	data := map[string]interface{}{
		"ids":          ids,
		"with_payload": true,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/collections/%s/points", p.config.URL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Qdrant get metadata failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var qdrantResp struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &qdrantResp); err != nil {
		return nil, err
	}

	result := make(map[string]map[string]interface{})
	for _, point := range qdrantResp.Result {
		id := fmt.Sprintf("%v", point.ID)
		result[id] = point.Payload
	}

	return result, nil
}

// DeleteMetadata deletes metadata
func (p *QdrantProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting metadata for %d vectors in Qdrant", len(ids))

	if len(ids) == 0 {
		return nil
	}

	collection := "default"

	data := map[string]interface{}{
		"points": ids,
		"keys":   keys,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s/points/payload/delete", p.config.URL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("api-key", p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qdrant delete metadata failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetStats returns provider statistics
func (p *QdrantProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting stats from Qdrant provider")

	collections, err := p.ListCollections(ctx)
	if err != nil {
		p.logger.Warn("Failed to list collections for stats: %v", err)
	}

	totalCollections := len(collections)
	// Qdrant doesn't provide easy way to get total vectors across all collections
	return &ProviderStats{
		Name:             p.GetName(),
		Type:             p.GetType(),
		Status:           "operational",
		TotalVectors:     0, // Would need to sum across collections
		TotalCollections: int64(totalCollections),
		TotalSize:        0,
		LastHealthCheck:  time.Now(),
	}, nil
}

// Optimize optimizes the provider
func (p *QdrantProvider) Optimize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Optimizing Qdrant provider")

	// Qdrant handles optimization automatically
	return nil
}

// Backup creates a backup
func (p *QdrantProvider) Backup(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Creating backup at %s for Qdrant provider", path)

	// Qdrant may have backup APIs but not implemented here
	return fmt.Errorf("Qdrant backup not implemented")
}

// Restore restores from backup
func (p *QdrantProvider) Restore(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Restoring from backup at %s for Qdrant provider", path)

	// Qdrant may have restore APIs but not implemented here
	return fmt.Errorf("Qdrant restore not implemented")
}

// Health checks provider health
func (p *QdrantProvider) Health(ctx context.Context) (*HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Checking health of Qdrant provider")

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
