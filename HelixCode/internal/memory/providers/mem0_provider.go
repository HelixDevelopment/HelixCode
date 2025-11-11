package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory"
)

// Mem0Provider implements memory operations using Mem0.ai
type Mem0Provider struct {
	config  map[string]interface{}
	client  *http.Client
	baseURL string
	apiKey  string
	logger  *logging.Logger
	userID  string
	agentID string
	runID   string
}

// NewMem0Provider creates a new Mem0 provider instance
func NewMem0Provider(config map[string]interface{}) (*Mem0Provider, error) {
	provider := &Mem0Provider{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logging.NewLoggerWithName("mem0_provider"),
	}

	// Extract configuration
	if baseURL, ok := config["base_url"].(string); ok {
		provider.baseURL = baseURL
	} else {
		provider.baseURL = "https://api.mem0.ai"
	}

	if apiKey, ok := config["api_key"].(string); ok {
		provider.apiKey = apiKey
	}

	if userID, ok := config["user_id"].(string); ok {
		provider.userID = userID
	}

	if agentID, ok := config["agent_id"].(string); ok {
		provider.agentID = agentID
	}

	if runID, ok := config["run_id"].(string); ok {
		provider.runID = runID
	}

	return provider, nil
}

// GetType returns the provider type
func (p *Mem0Provider) GetType() string {
	return string(memory.ProviderTypeMem0)
}

// GetName returns the provider name
func (p *Mem0Provider) GetName() string {
	return "Mem0"
}

// GetCapabilities returns provider capabilities
func (p *Mem0Provider) GetCapabilities() []string {
	return []string{
		"memory_storage",
		"memory_retrieval",
		"memory_search",
		"context_management",
		"graph_memory",
		"vector_memory",
	}
}

// GetConfiguration returns provider configuration
func (p *Mem0Provider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether this is a cloud provider
func (p *Mem0Provider) IsCloud() bool {
	return strings.Contains(p.baseURL, "mem0.ai") || strings.Contains(p.baseURL, "api.mem0")
}

// Store stores vectors in Mem0
func (p *Mem0Provider) Store(ctx context.Context, data []*VectorData) error {
	if len(data) == 0 {
		return nil
	}

	// Convert to Mem0 format
	memories := make([]map[string]interface{}, len(data))
	for i, item := range data {
		memories[i] = map[string]interface{}{
			"id":       item.ID,
			"text":     item.Metadata["content"],
			"user_id":  p.userID,
			"agent_id": p.agentID,
			"run_id":   p.runID,
			"metadata": item.Metadata,
		}
	}

	payload := map[string]interface{}{
		"memories": memories,
	}

	return p.makeRequest(ctx, "POST", "/memories/", payload, nil)
}

// Search searches for vectors in Mem0
func (p *Mem0Provider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	// Prepare search payload
	payload := map[string]interface{}{
		"query":    query.Text,
		"user_id":  p.userID,
		"agent_id": p.agentID,
		"run_id":   p.runID,
		"limit":    query.TopK,
	}

	if query.Filters != nil {
		payload["filters"] = query.Filters
	}

	var response map[string]interface{}
	err := p.makeRequest(ctx, "POST", "/search/", payload, &response)
	if err != nil {
		return nil, err
	}

	// Parse response
	results := []*VectorSearchResultItem{}
	if memories, ok := response["memories"].([]interface{}); ok {
		for _, mem := range memories {
			if memMap, ok := mem.(map[string]interface{}); ok {
				item := &VectorSearchResultItem{
					ID:       getStringValue(memMap, "id"),
					Score:    getFloatValue(memMap, "score"),
					Metadata: memMap,
				}
				results = append(results, item)
			}
		}
	}

	return &VectorSearchResult{
		Results: results,
	}, nil
}

// Delete deletes vectors from Mem0
func (p *Mem0Provider) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	payload := map[string]interface{}{
		"memory_ids": ids,
		"user_id":    p.userID,
		"agent_id":   p.agentID,
		"run_id":     p.runID,
	}

	return p.makeRequest(ctx, "DELETE", "/memories/", payload, nil)
}

// GetStats returns provider statistics
func (p *Mem0Provider) GetStats(ctx context.Context) (*ProviderStats, error) {
	var response map[string]interface{}
	err := p.makeRequest(ctx, "GET", "/stats/", nil, &response)
	if err != nil {
		return nil, err
	}

	return &ProviderStats{
		TotalVectors:   int64(getFloatValue(response, "total_memories")),
		TotalSize:      int64(getFloatValue(response, "total_size_bytes")),
		AverageLatency: time.Duration(getFloatValue(response, "avg_latency_ms")) * time.Millisecond,
	}, nil
}

// Health checks provider health
func (p *Mem0Provider) Health(ctx context.Context) (*HealthStatus, error) {
	var response map[string]interface{}
	err := p.makeRequest(ctx, "GET", "/health/", nil, &response)
	if err != nil {
		return &HealthStatus{
			Status:    "unhealthy",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	status := "healthy"
	if healthStatus, ok := response["status"].(string); ok && healthStatus != "ok" {
		status = "unhealthy"
	}

	return &HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
	}, nil
}

// Close closes the provider
func (p *Mem0Provider) Close() error {
	// Cleanup resources if needed
	return nil
}

// AddMetadata adds metadata to a vector
func (p *Mem0Provider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	// Mem0 doesn't have direct metadata addition, update the memory instead
	return p.UpdateMetadata(ctx, id, metadata)
}

// UpdateMetadata updates metadata for a vector
func (p *Mem0Provider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	// For Mem0, this would require updating the memory entry
	// This is a simplified implementation
	payload := map[string]interface{}{
		"memory_id": id,
		"metadata":  metadata,
		"user_id":   p.userID,
		"agent_id":  p.agentID,
		"run_id":    p.runID,
	}

	return p.makeRequest(ctx, "PUT", "/memories/"+id+"/", payload, nil)
}

// GetMetadata retrieves metadata for vectors
func (p *Mem0Provider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	// Mem0 doesn't have direct metadata retrieval, this is a stub
	result := make(map[string]map[string]interface{})
	for _, id := range ids {
		result[id] = make(map[string]interface{})
	}
	return result, nil
}

// DeleteMetadata deletes metadata from vectors
func (p *Mem0Provider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	// Stub implementation
	return nil
}

// Retrieve retrieves vectors by IDs
func (p *Mem0Provider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	// Mem0 doesn't have direct retrieval by ID, this is a stub
	return []*VectorData{}, nil
}

// Update updates a vector
func (p *Mem0Provider) Update(ctx context.Context, id string, vector *VectorData) error {
	// For Mem0, update the memory
	return p.UpdateMetadata(ctx, id, vector.Metadata)
}

// FindSimilar finds similar vectors
func (p *Mem0Provider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	// Use search with vector
	query := &VectorQuery{
		Vector:  embedding,
		TopK:    k,
		Filters: filters,
	}
	result, err := p.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	similar := make([]*VectorSimilarityResult, len(result.Results))
	for i, item := range result.Results {
		similar[i] = &VectorSimilarityResult{
			ID:       item.ID,
			Score:    item.Score,
			Distance: 1.0 - item.Score, // Convert score to distance
			Metadata: item.Metadata,
		}
	}
	return similar, nil
}

// BatchFindSimilar performs batch similarity search
func (p *Mem0Provider) BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error) {
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
func (p *Mem0Provider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	// Mem0 doesn't have collections in the same way, stub
	return nil
}

// DeleteCollection deletes a collection
func (p *Mem0Provider) DeleteCollection(ctx context.Context, name string) error {
	// Stub
	return nil
}

// ListCollections lists all collections
func (p *Mem0Provider) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	// Stub
	return []*CollectionInfo{}, nil
}

// GetCollection gets collection information
func (p *Mem0Provider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	// Stub
	return nil, fmt.Errorf("collection not found")
}

// CreateIndex creates an index
func (p *Mem0Provider) CreateIndex(ctx context.Context, collection string, config *IndexConfig) error {
	// Stub
	return nil
}

// DeleteIndex deletes an index
func (p *Mem0Provider) DeleteIndex(ctx context.Context, collection, name string) error {
	// Stub
	return nil
}

// ListIndexes lists indexes
func (p *Mem0Provider) ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error) {
	// Stub
	return []*IndexInfo{}, nil
}

// Optimize optimizes the provider
func (p *Mem0Provider) Optimize(ctx context.Context) error {
	// Stub
	return nil
}

// Backup creates a backup
func (p *Mem0Provider) Backup(ctx context.Context, path string) error {
	// Stub
	return nil
}

// Restore restores from backup
func (p *Mem0Provider) Restore(ctx context.Context, path string) error {
	// Stub
	return nil
}

// Initialize initializes the provider
func (p *Mem0Provider) Initialize(ctx context.Context, config interface{}) error {
	// Already initialized in NewMem0Provider
	return nil
}

// Start starts the provider
func (p *Mem0Provider) Start(ctx context.Context) error {
	// HTTP client is ready
	return nil
}

// Stop stops the provider
func (p *Mem0Provider) Stop(ctx context.Context) error {
	return p.Close()
}

// GetCostInfo returns cost information
func (p *Mem0Provider) GetCostInfo() *CostInfo {
	return &CostInfo{
		Currency:      "USD",
		ComputeCost:   0.0,
		TransferCost:  0.0,
		StorageCost:   0.0,
		TotalCost:     0.0,
		BillingPeriod: "monthly",
		FreeTierUsed:  0.0,
		FreeTierLimit: 0.0,
	}
}

// makeRequest makes an HTTP request to Mem0 API
func (p *Mem0Provider) makeRequest(ctx context.Context, method, endpoint string, payload interface{}, response interface{}) error {
	url := p.baseURL + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response if needed
	if response != nil {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Helper functions
func getStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloatValue(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok {
		if num, ok := val.(float64); ok {
			return num
		}
		if num, ok := val.(int); ok {
			return float64(num)
		}
	}
	return 0.0
}
