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

// ChromaDBProvider implements VectorProvider for ChromaDB
type ChromaDBProvider struct {
	config      *ChromaDBConfig
	logger      *logging.Logger
	httpClient  *http.Client
	mu          sync.RWMutex
	initialized bool
	started     bool
}

// ChromaDBConfig holds configuration for ChromaDB
type ChromaDBConfig struct {
	URL      string `json:"url"`
	APIKey   string `json:"api_key"`
	Tenant   string `json:"tenant"`
	Database string `json:"database"`
}

// NewChromaDBProvider creates a new ChromaDB provider
func NewChromaDBProvider(config map[string]interface{}) (VectorProvider, error) {
	cfg := &ChromaDBConfig{
		URL:      getStringConfig(config, "url", "http://localhost:8000"),
		APIKey:   getStringConfig(config, "api_key", ""),
		Tenant:   getStringConfig(config, "tenant", "default_tenant"),
		Database: getStringConfig(config, "database", "default_database"),
	}

	logger := logging.NewLoggerWithName("chromadb_provider")

	return &ChromaDBProvider{
		config: cfg,
		logger: logger,
	}, nil
}

// testConnection tests the connection to ChromaDB
func (p *ChromaDBProvider) testConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/heartbeat", p.config.URL)
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
		return fmt.Errorf("ChromaDB heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}

// Initialize initializes the ChromaDB provider
func (p *ChromaDBProvider) Initialize(ctx context.Context, config interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Initializing ChromaDB provider url=%s tenant=%s database=%s", p.config.URL, p.config.Tenant, p.config.Database)

	// Initialize HTTP client
	p.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Test connection to ChromaDB
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to ChromaDB: %w", err)
	}

	p.initialized = true
	p.logger.Info("ChromaDB provider initialized successfully")
	return nil
}

// Start starts the ChromaDB provider
func (p *ChromaDBProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("provider not initialized")
	}

	p.logger.Info("Starting ChromaDB provider")

	// Test connection before starting
	if err := p.testConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to ChromaDB during start: %w", err)
	}

	p.started = true
	p.logger.Info("ChromaDB provider started successfully")
	return nil
}

// Stop stops the ChromaDB provider
func (p *ChromaDBProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Stopping ChromaDB provider")

	// Close HTTP client connections
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}

	p.started = false
	p.logger.Info("ChromaDB provider stopped successfully")
	return nil
}

// GetName returns the provider name
func (p *ChromaDBProvider) GetName() string {
	return "chromadb"
}

// GetType returns the provider type
func (p *ChromaDBProvider) GetType() string {
	return string(ProviderTypeChroma)
}

// GetCapabilities returns provider capabilities
func (p *ChromaDBProvider) GetCapabilities() []string {
	return []string{"vector_storage", "similarity_search", "metadata_filtering"}
}

// GetConfiguration returns the current configuration
func (p *ChromaDBProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether this is a cloud provider
func (p *ChromaDBProvider) IsCloud() bool {
	return false // ChromaDB can be self-hosted or cloud
}

// GetCostInfo returns cost information
func (p *ChromaDBProvider) GetCostInfo() *CostInfo {
	return &CostInfo{
		Currency:      "USD",
		ComputeCost:   0.0,
		TransferCost:  0.0,
		StorageCost:   0.0,
		TotalCost:     0.0,
		BillingPeriod: "monthly",
	}
}

// Store stores vectors in ChromaDB
func (p *ChromaDBProvider) Store(ctx context.Context, vectors []*VectorData) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	if len(vectors) == 0 {
		return nil
	}

	p.logger.Info("Storing %d vectors in ChromaDB", len(vectors))

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
func (p *ChromaDBProvider) storeInCollection(ctx context.Context, collection string, vectors []*VectorData) error {
	// Ensure collection exists
	if err := p.ensureCollection(ctx, collection, vectors[0]); err != nil {
		return err
	}

	// Prepare data for ChromaDB
	ids := make([]string, len(vectors))
	embeddings := make([][]float64, len(vectors))
	metadatas := make([]map[string]interface{}, len(vectors))

	for i, v := range vectors {
		ids[i] = v.ID
		embeddings[i] = v.Vector
		metadatas[i] = v.Metadata
		if metadatas[i] == nil {
			metadatas[i] = make(map[string]interface{})
		}
		// Add timestamp if not present
		if _, ok := metadatas[i]["timestamp"]; !ok {
			metadatas[i]["timestamp"] = v.Timestamp.Unix()
		}
	}

	data := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"metadatas":  metadatas,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/add", p.config.URL, collection)
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
		return fmt.Errorf("ChromaDB add failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ensureCollection ensures a collection exists
func (p *ChromaDBProvider) ensureCollection(ctx context.Context, name string, sampleVector *VectorData) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/api/v1/collections/%s", p.config.URL, name)
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
		"metadata": map[string]interface{}{
			"dimension": len(sampleVector.Vector),
		},
	}

	jsonData, err := json.Marshal(createData)
	if err != nil {
		return err
	}

	url = fmt.Sprintf("%s/api/v1/collections", p.config.URL)
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
		return fmt.Errorf("failed to create collection: %s", string(body))
	}

	return nil
}

// Retrieve retrieves vectors by IDs
func (p *ChromaDBProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Retrieving %d vectors from ChromaDB", len(ids))

	if len(ids) == 0 {
		return []*VectorData{}, nil
	}

	// Group by collection - for now assume all from default, but we need to handle multiple collections
	// This is a simplification; in practice, we'd need to know which collection each ID belongs to
	collection := "default"

	data := map[string]interface{}{
		"ids":     ids,
		"include": []string{"metadatas", "embeddings"},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/get", p.config.URL, collection)
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
		return nil, fmt.Errorf("ChromaDB get failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse ChromaDB response
	var chromaResp struct {
		Ids        []string                 `json:"ids"`
		Embeddings [][]float64              `json:"embeddings"`
		Metadatas  []map[string]interface{} `json:"metadatas"`
	}

	if err := json.Unmarshal(body, &chromaResp); err != nil {
		return nil, err
	}

	results := make([]*VectorData, len(chromaResp.Ids))
	for i, id := range chromaResp.Ids {
		results[i] = &VectorData{
			ID:         id,
			Vector:     chromaResp.Embeddings[i],
			Metadata:   chromaResp.Metadatas[i],
			Collection: collection,
			Timestamp:  time.Now(), // We don't get timestamp from ChromaDB, so use current time
		}
	}

	return results, nil
}

// Update updates a vector
func (p *ChromaDBProvider) Update(ctx context.Context, id string, vector *VectorData) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	p.logger.Info("Updating vector %s in ChromaDB", id)

	collection := vector.Collection
	if collection == "" {
		collection = "default"
	}

	data := map[string]interface{}{
		"ids":        []string{id},
		"embeddings": [][]float64{vector.Vector},
		"metadatas":  []map[string]interface{}{vector.Metadata},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/update", p.config.URL, collection)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ChromaDB update failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Delete deletes vectors by IDs
func (p *ChromaDBProvider) Delete(ctx context.Context, ids []string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return fmt.Errorf("provider not started")
	}

	p.logger.Info("Deleting %d vectors from ChromaDB", len(ids))

	if len(ids) == 0 {
		return nil
	}

	// Group by collection - simplification, assume all from default
	collection := "default"

	data := map[string]interface{}{
		"ids": ids,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", p.config.URL, collection)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ChromaDB delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Search performs vector similarity search
func (p *ChromaDBProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Searching vectors in ChromaDB with top_k=%d", query.TopK)

	collection := query.Collection
	if collection == "" {
		collection = "default"
	}

	// Prepare query data
	queryData := map[string]interface{}{
		"query_embeddings": [][]float64{query.Vector},
		"n_results":        query.TopK,
		"include":          []string{"metadatas", "documents", "distances"},
	}

	if query.IncludeVector {
		queryData["include"] = []string{"metadatas", "documents", "distances", "embeddings"}
	}

	jsonData, err := json.Marshal(queryData)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", p.config.URL, collection)
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
		return nil, fmt.Errorf("ChromaDB query failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse ChromaDB response
	var chromaResp struct {
		Ids        [][]string                 `json:"ids"`
		Distances  [][]float64                `json:"distances"`
		Metadatas  [][]map[string]interface{} `json:"metadatas"`
		Documents  [][]string                 `json:"documents"`
		Embeddings [][][]float64              `json:"embeddings,omitempty"`
	}

	if err := json.Unmarshal(body, &chromaResp); err != nil {
		return nil, err
	}

	// Convert to our format
	results := []*VectorSearchResultItem{}
	if len(chromaResp.Ids) > 0 && len(chromaResp.Ids[0]) > 0 {
		for i, id := range chromaResp.Ids[0] {
			item := &VectorSearchResultItem{
				ID:       id,
				Metadata: make(map[string]interface{}),
			}

			if len(chromaResp.Distances) > 0 && len(chromaResp.Distances[0]) > i {
				item.Distance = chromaResp.Distances[0][i]
				item.Score = 1.0 - item.Distance // Convert distance to similarity score
			}

			if len(chromaResp.Metadatas) > 0 && len(chromaResp.Metadatas[0]) > i && chromaResp.Metadatas[0][i] != nil {
				item.Metadata = chromaResp.Metadatas[0][i]
			}

			if query.IncludeVector && len(chromaResp.Embeddings) > 0 && len(chromaResp.Embeddings[0]) > i {
				item.Vector = chromaResp.Embeddings[0][i]
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

// FindSimilar finds similar vectors
func (p *ChromaDBProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Finding %d similar vectors in ChromaDB", k)

	collection := "default" // Simplification

	queryData := map[string]interface{}{
		"query_embeddings": [][]float64{embedding},
		"n_results":        k,
		"include":          []string{"metadatas", "distances"},
	}

	if filters != nil {
		queryData["where"] = filters
	}

	jsonData, err := json.Marshal(queryData)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", p.config.URL, collection)
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
		return nil, fmt.Errorf("ChromaDB query failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chromaResp struct {
		Ids       [][]string                 `json:"ids"`
		Distances [][]float64                `json:"distances"`
		Metadatas [][]map[string]interface{} `json:"metadatas"`
	}

	if err := json.Unmarshal(body, &chromaResp); err != nil {
		return nil, err
	}

	results := []*VectorSimilarityResult{}
	if len(chromaResp.Ids) > 0 && len(chromaResp.Ids[0]) > 0 {
		for i, id := range chromaResp.Ids[0] {
			result := &VectorSimilarityResult{
				ID:    id,
				Score: 0.0,
			}

			if len(chromaResp.Distances) > 0 && len(chromaResp.Distances[0]) > i {
				result.Score = 1.0 - chromaResp.Distances[0][i] // Convert distance to similarity
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// BatchFindSimilar performs batch similarity search
func (p *ChromaDBProvider) BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return nil, fmt.Errorf("provider not started")
	}

	p.logger.Info("Batch finding similar vectors for %d queries in ChromaDB", len(queries))

	if len(queries) == 0 {
		return [][]*VectorSimilarityResult{}, nil
	}

	collection := "default"

	queryData := map[string]interface{}{
		"query_embeddings": queries,
		"n_results":        k,
		"include":          []string{"distances"},
	}

	jsonData, err := json.Marshal(queryData)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", p.config.URL, collection)
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
		return nil, fmt.Errorf("ChromaDB batch query failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chromaResp struct {
		Ids       [][][]string  `json:"ids"`
		Distances [][][]float64 `json:"distances"`
	}

	if err := json.Unmarshal(body, &chromaResp); err != nil {
		return nil, err
	}

	results := make([][]*VectorSimilarityResult, len(queries))
	for qIdx := range queries {
		results[qIdx] = []*VectorSimilarityResult{}
		if len(chromaResp.Ids) > qIdx && len(chromaResp.Ids[qIdx]) > 0 && len(chromaResp.Ids[qIdx][0]) > 0 {
			for i, id := range chromaResp.Ids[qIdx][0] {
				result := &VectorSimilarityResult{
					ID:    id,
					Score: 0.0,
				}

				if len(chromaResp.Distances) > qIdx && len(chromaResp.Distances[qIdx]) > 0 && len(chromaResp.Distances[qIdx][0]) > i {
					result.Score = 1.0 - chromaResp.Distances[qIdx][0][i]
				}

				results[qIdx] = append(results[qIdx], result)
			}
		}
	}

	return results, nil
}

// CreateCollection creates a new collection
func (p *ChromaDBProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Creating collection %s in ChromaDB", name)

	createData := map[string]interface{}{
		"name": name,
		"metadata": map[string]interface{}{
			"dimension": config.Dimension,
		},
	}

	jsonData, err := json.Marshal(createData)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/collections", p.config.URL)
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
		return fmt.Errorf("failed to create collection: %s", string(body))
	}

	return nil
}

// DeleteCollection deletes a collection
func (p *ChromaDBProvider) DeleteCollection(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting collection %s from ChromaDB", name)

	url := fmt.Sprintf("%s/api/v1/collections/%s", p.config.URL, name)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ChromaDB delete collection failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListCollections lists all collections
func (p *ChromaDBProvider) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Listing collections in ChromaDB")

	url := fmt.Sprintf("%s/api/v1/collections", p.config.URL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("ChromaDB list collections failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse ChromaDB response - assuming it returns a list of collection names
	var collections []string
	if err := json.Unmarshal(body, &collections); err != nil {
		return nil, err
	}

	result := make([]*CollectionInfo, len(collections))
	for i, name := range collections {
		result[i] = &CollectionInfo{Name: name}
	}

	return result, nil
}

// GetCollection gets collection information
func (p *ChromaDBProvider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting collection %s info from ChromaDB", name)

	url := fmt.Sprintf("%s/api/v1/collections/%s", p.config.URL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
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
		return nil, fmt.Errorf("ChromaDB get collection failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse ChromaDB collection response
	var collection struct {
		Name     string                 `json:"name"`
		ID       string                 `json:"id"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.Unmarshal(body, &collection); err != nil {
		return nil, err
	}

	return &CollectionInfo{
		Name:     collection.Name,
		Metadata: collection.Metadata,
	}, nil
}

// CreateIndex creates an index
func (p *ChromaDBProvider) CreateIndex(ctx context.Context, collection string, config *IndexConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Creating index %s in collection %s in ChromaDB", config.Name, collection)

	// ChromaDB handles indexing automatically, no manual index creation needed
	return fmt.Errorf("ChromaDB does not support manual index creation")
}

// DeleteIndex deletes an index
func (p *ChromaDBProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting index %s from collection %s in ChromaDB", name, collection)

	// ChromaDB handles indexing automatically, no manual index deletion needed
	return fmt.Errorf("ChromaDB does not support manual index deletion")
}

// ListIndexes lists indexes in a collection
func (p *ChromaDBProvider) ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Listing indexes in collection %s in ChromaDB", collection)

	// ChromaDB uses automatic indexing, return default index info
	return []*IndexInfo{
		{
			Name:  "default",
			Type:  "hnsw",
			State: "active",
		},
	}, nil
}

// AddMetadata adds metadata to a vector
func (p *ChromaDBProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Adding metadata to vector %s in ChromaDB", id)

	// First get current metadata
	current, err := p.GetMetadata(ctx, []string{id})
	if err != nil {
		return err
	}

	if len(current) == 0 {
		return fmt.Errorf("vector %s not found", id)
	}

	// Merge metadata
	for k, v := range metadata {
		current[id][k] = v
	}

	// Update the vector with new metadata
	collection := "default"
	data := map[string]interface{}{
		"ids":       []string{id},
		"metadatas": []map[string]interface{}{current[id]},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/update", p.config.URL, collection)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ChromaDB update metadata failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateMetadata updates metadata
func (p *ChromaDBProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Updating metadata for vector %s in ChromaDB", id)

	// Update metadata directly
	collection := "default"
	data := map[string]interface{}{
		"ids":       []string{id},
		"metadatas": []map[string]interface{}{metadata},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/update", p.config.URL, collection)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ChromaDB update metadata failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetMetadata gets metadata for vectors
func (p *ChromaDBProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting metadata for %d vectors from ChromaDB", len(ids))

	if len(ids) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	collection := "default"
	data := map[string]interface{}{
		"ids":     ids,
		"include": []string{"metadatas"},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/get", p.config.URL, collection)
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
		return nil, fmt.Errorf("ChromaDB get metadata failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chromaResp struct {
		Ids       []string                 `json:"ids"`
		Metadatas []map[string]interface{} `json:"metadatas"`
	}

	if err := json.Unmarshal(body, &chromaResp); err != nil {
		return nil, err
	}

	result := make(map[string]map[string]interface{})
	for i, id := range chromaResp.Ids {
		if i < len(chromaResp.Metadatas) {
			result[id] = chromaResp.Metadatas[i]
		}
	}

	return result, nil
}

// DeleteMetadata deletes metadata
func (p *ChromaDBProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Deleting metadata for %d vectors in ChromaDB", len(ids))

	if len(ids) == 0 {
		return nil
	}

	// Get current metadata
	current, err := p.GetMetadata(ctx, ids)
	if err != nil {
		return err
	}

	// Remove specified keys
	for _, id := range ids {
		if meta, exists := current[id]; exists {
			for _, key := range keys {
				delete(meta, key)
			}
		}
	}

	// Update metadata
	collection := "default"
	metadatas := make([]map[string]interface{}, len(ids))
	for i, id := range ids {
		if meta, exists := current[id]; exists {
			metadatas[i] = meta
		} else {
			metadatas[i] = make(map[string]interface{})
		}
	}

	data := map[string]interface{}{
		"ids":       ids,
		"metadatas": metadatas,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/update", p.config.URL, collection)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ChromaDB delete metadata failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetStats returns provider statistics
func (p *ChromaDBProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Getting stats from ChromaDB provider")

	collections, err := p.ListCollections(ctx)
	if err != nil {
		p.logger.Warn("Failed to list collections for stats: %v", err)
	}

	totalCollections := len(collections)
	totalVectors := 0

	// For each collection, we could get count, but ChromaDB doesn't have a direct count API
	// For now, return basic stats
	return &ProviderStats{
		Name:             p.GetName(),
		Type:             p.GetType(),
		Status:           "operational",
		TotalVectors:     int64(totalVectors),
		TotalCollections: int64(totalCollections),
		TotalSize:        0, // Not available from ChromaDB API
		LastHealthCheck:  time.Now(),
	}, nil
}

// Optimize optimizes the provider
func (p *ChromaDBProvider) Optimize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Optimizing ChromaDB provider")

	// ChromaDB handles optimization automatically
	return nil
}

// Backup creates a backup
func (p *ChromaDBProvider) Backup(ctx context.Context, path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Creating backup at %s for ChromaDB provider", path)

	// ChromaDB does not provide built-in backup functionality
	return fmt.Errorf("ChromaDB does not support backup operations")
}

// Restore restores from backup
func (p *ChromaDBProvider) Restore(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Restoring from backup at %s for ChromaDB provider", path)

	// ChromaDB does not provide built-in restore functionality
	return fmt.Errorf("ChromaDB does not support restore operations")
}

// Health checks provider health
func (p *ChromaDBProvider) Health(ctx context.Context) (*HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Info("Checking health of ChromaDB provider")

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

// Close closes the provider
func (p *ChromaDBProvider) Close(ctx context.Context) error {
	// Cleanup resources if needed
	return nil
}
