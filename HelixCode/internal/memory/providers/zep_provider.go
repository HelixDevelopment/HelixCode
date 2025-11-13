package providers

import (
	"context"
	"fmt"
	"time"

	"dev.helix.code/internal/logging"
	zep "github.com/getzep/zep-go/v3"
	zepclient "github.com/getzep/zep-go/v3/client"
	"github.com/getzep/zep-go/v3/option"
)

// ZepProvider implements memory operations using Zep.ai
type ZepProvider struct {
	config  map[string]interface{}
	client  *zepclient.Client
	logger  *logging.Logger
	userID  string
	apiKey  string
	baseURL string
}

// NewZepProvider creates a new Zep provider instance
func NewZepProvider(config map[string]interface{}) (*ZepProvider, error) {
	provider := &ZepProvider{
		config: config,
		logger: logging.NewLoggerWithName("zep_provider"),
	}

	// Extract configuration
	if apiKey, ok := config["api_key"].(string); ok {
		provider.apiKey = apiKey
	}

	if baseURL, ok := config["base_url"].(string); ok {
		provider.baseURL = baseURL
	}

	if userID, ok := config["user_id"].(string); ok {
		provider.userID = userID
	}

	// Initialize client
	clientOptions := []option.RequestOption{
		option.WithAPIKey(provider.apiKey),
	}

	if provider.baseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(provider.baseURL))
	}

	provider.client = zepclient.NewClient(clientOptions...)

	return provider, nil
}

// GetType returns the provider type
func (p *ZepProvider) GetType() string {
	return string(ProviderTypeZep)
}

// GetName returns the provider name
func (p *ZepProvider) GetName() string {
	return "Zep"
}

// GetCapabilities returns provider capabilities
func (p *ZepProvider) GetCapabilities() []string {
	return []string{
		"memory_storage",
		"memory_retrieval",
		"memory_search",
		"context_management",
		"graph_memory",
		"knowledge_graph",
		"user_management",
		"thread_management",
	}
}

// GetConfiguration returns provider configuration
func (p *ZepProvider) GetConfiguration() interface{} {
	return p.config
}

// IsCloud returns whether this is a cloud provider
func (p *ZepProvider) IsCloud() bool {
	return p.baseURL == "" || contains(p.baseURL, "zep.ai") || contains(p.baseURL, "getzep.com")
}

// Store stores memory data in Zep
func (p *ZepProvider) Store(ctx context.Context, data []*VectorData) error {
	if len(data) == 0 {
		return nil
	}

	// Create user if not exists
	if p.userID != "" {
		_, err := p.client.User.Add(ctx, &zep.CreateUserRequest{
			UserID: p.userID,
		})
		if err != nil {
			p.logger.Warn("Failed to create user: %v", err)
		}
	}

	// Create thread for storing messages
	threadID := generateThreadID()

	_, err := p.client.Thread.Create(ctx, &zep.CreateThreadRequest{
		ThreadID: threadID,
		UserID:   p.userID,
	})
	if err != nil {
		return fmt.Errorf("failed to create thread: %w", err)
	}

	// Convert data to messages
	var messages []*zep.Message
	for _, item := range data {
		content := item.Metadata["content"]
		if contentStr, ok := content.(string); ok {
			messages = append(messages, &zep.Message{
				Role:    "user",
				Content: contentStr,
			})
		}
	}

	if len(messages) > 0 {
		_, err = p.client.Thread.AddMessages(ctx, threadID, &zep.AddThreadMessagesRequest{
			Messages: messages,
		})
		if err != nil {
			return fmt.Errorf("failed to add messages: %w", err)
		}
	}

	return nil
}

// Search searches for memory in Zep
func (p *ZepProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	// Use graph search for Zep
	searchResults, err := p.client.Graph.Search(ctx, &zep.GraphSearchQuery{
		UserID: zep.String(p.userID),
		Query:  query.Text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search graph: %w", err)
	}

	// Convert results
	results := []*VectorSearchResultItem{}
	for _, edge := range searchResults.Edges {
		results = append(results, &VectorSearchResultItem{
			ID: edge.UUID,
			Metadata: map[string]interface{}{
				"fact":     edge.Fact,
				"type":     "edge",
				"valid_at": edge.ValidAt,
			},
			Score: 1.0, // Zep doesn't provide scores in this format
		})
	}

	for _, node := range searchResults.Nodes {
		results = append(results, &VectorSearchResultItem{
			ID: node.UUID,
			Metadata: map[string]interface{}{
				"name":    node.Name,
				"type":    "node",
				"summary": node.Summary,
			},
			Score: 1.0,
		})
	}

	return &VectorSearchResult{
		Results: results,
	}, nil
}

// Retrieve retrieves vectors by IDs from Zep
func (p *ZepProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	// Zep doesn't have direct retrieve by ID, this is a stub
	p.logger.Warn("Retrieve operation not fully supported in Zep")
	return []*VectorData{}, nil
}

// Update updates a vector in Zep
func (p *ZepProvider) Update(ctx context.Context, id string, vector *VectorData) error {
	// Zep doesn't have direct update by ID, this is a stub
	p.logger.Warn("Update operation not fully supported in Zep")
	return nil
}

// Delete deletes memory from Zep
func (p *ZepProvider) Delete(ctx context.Context, ids []string) error {
	// Zep doesn't have direct delete by ID, this is a stub
	p.logger.Warn("Delete operation not fully supported in Zep")
	return nil
}

// FindSimilar finds similar vectors in Zep
func (p *ZepProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	// Use graph search for similarity
	searchResults, err := p.client.Graph.Search(ctx, &zep.GraphSearchQuery{
		UserID: zep.String(p.userID),
		Query:  fmt.Sprintf("embedding:%v", embedding), // Placeholder, Zep may not support direct embedding search
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search graph: %w", err)
	}

	results := []*VectorSimilarityResult{}
	for _, edge := range searchResults.Edges {
		results = append(results, &VectorSimilarityResult{
			ID:       edge.UUID,
			Score:    1.0,
			Metadata: map[string]interface{}{"fact": edge.Fact},
		})
		if len(results) >= k {
			break
		}
	}

	return results, nil
}

// BatchFindSimilar finds similar vectors for multiple queries in Zep
func (p *ZepProvider) BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error) {
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

// CreateCollection creates a collection in Zep
func (p *ZepProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	// Zep doesn't have explicit collections, this is a stub
	p.logger.Warn("CreateCollection not supported in Zep")
	return nil
}

// DeleteCollection deletes a collection in Zep
func (p *ZepProvider) DeleteCollection(ctx context.Context, name string) error {
	// Zep doesn't have explicit collections, this is a stub
	p.logger.Warn("DeleteCollection not supported in Zep")
	return nil
}

// ListCollections lists collections in Zep
func (p *ZepProvider) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	// Zep doesn't have explicit collections, return empty
	return []*CollectionInfo{}, nil
}

// GetCollection gets collection info in Zep
func (p *ZepProvider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	// Zep doesn't have explicit collections, this is a stub
	p.logger.Warn("GetCollection not supported in Zep")
	return nil, fmt.Errorf("collection not found")
}

// CreateIndex creates an index in Zep
func (p *ZepProvider) CreateIndex(ctx context.Context, collection string, config *IndexConfig) error {
	// Zep doesn't have explicit indexes, this is a stub
	p.logger.Warn("CreateIndex not supported in Zep")
	return nil
}

// DeleteIndex deletes an index in Zep
func (p *ZepProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	// Zep doesn't have explicit indexes, this is a stub
	p.logger.Warn("DeleteIndex not supported in Zep")
	return nil
}

// ListIndexes lists indexes in Zep
func (p *ZepProvider) ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error) {
	// Zep doesn't have explicit indexes, return empty
	return []*IndexInfo{}, nil
}

// AddMetadata adds metadata to a vector in Zep
func (p *ZepProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	// Zep doesn't have direct metadata operations, this is a stub
	p.logger.Warn("AddMetadata not supported in Zep")
	return nil
}

// UpdateMetadata updates metadata for a vector in Zep
func (p *ZepProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	// Zep doesn't have direct metadata operations, this is a stub
	p.logger.Warn("UpdateMetadata not supported in Zep")
	return nil
}

// GetMetadata gets metadata for vectors in Zep
func (p *ZepProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	// Zep doesn't have direct metadata operations, return empty
	return map[string]map[string]interface{}{}, nil
}

// DeleteMetadata deletes metadata from vectors in Zep
func (p *ZepProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	// Zep doesn't have direct metadata operations, this is a stub
	p.logger.Warn("DeleteMetadata not supported in Zep")
	return nil
}

// Optimize optimizes the Zep provider
func (p *ZepProvider) Optimize(ctx context.Context) error {
	// Zep doesn't have explicit optimization, this is a stub
	p.logger.Warn("Optimize not supported in Zep")
	return nil
}

// Backup backs up data in Zep
func (p *ZepProvider) Backup(ctx context.Context, path string) error {
	// Zep doesn't have explicit backup, this is a stub
	p.logger.Warn("Backup not supported in Zep")
	return nil
}

// Restore restores data in Zep
func (p *ZepProvider) Restore(ctx context.Context, path string) error {
	// Zep doesn't have explicit restore, this is a stub
	p.logger.Warn("Restore not supported in Zep")
	return nil
}

// Initialize initializes the Zep provider
func (p *ZepProvider) Initialize(ctx context.Context, config interface{}) error {
	// Already initialized in NewZepProvider
	return nil
}

// Start starts the Zep provider
func (p *ZepProvider) Start(ctx context.Context) error {
	// Zep client is already started
	return nil
}

// Stop stops the Zep provider
func (p *ZepProvider) Stop(ctx context.Context) error {
	// Close client if needed
	return p.Close(ctx)
}

// GetCostInfo returns cost information for Zep
func (p *ZepProvider) GetCostInfo() *CostInfo {
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

// GetStats returns provider statistics
func (p *ZepProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	// Get user info as basic stats
	stats := &ProviderStats{
		Name:             "Zep",
		Type:             "zep",
		Status:           "active",
		TotalOperations:  0,
		SuccessfulOps:    0,
		FailedOps:        0,
		AverageLatency:   0,
		TotalVectors:     0,
		TotalCollections: 0,
		TotalSize:        0,
		LastHealthCheck:  time.Now(),
	}

	return stats, nil
}

// Health checks provider health
func (p *ZepProvider) Health(ctx context.Context) (*HealthStatus, error) {
	// Simple health check by trying to get user info
	_, err := p.client.User.Get(ctx, p.userID)
	if err != nil {
		return &HealthStatus{
			Status:    "unhealthy",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	return &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
	}, nil
}

// Close closes the provider
func (p *ZepProvider) Close(ctx context.Context) error {
	// Cleanup if needed
	return nil
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func generateThreadID() string {
	return fmt.Sprintf("thread-%d", time.Now().UnixNano())
}
