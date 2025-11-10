package mocks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/memory/providers"
)

// MockVectorProvider mocks the VectorProvider interface
type MockVectorProvider struct {
	mock.Mock
	mu sync.Mutex

	// Internal state for testing
	store       map[string][]*memory.VectorData
	collections map[string]*memory.CollectionConfig
	indices     map[string]*memory.IndexInfo
	stats       *providers.ProviderStats
	healthy     bool
	initialized bool
	started     bool
}

// NewMockVectorProvider creates a new mock vector provider
func NewMockVectorProvider(t interface{}) *MockVectorProvider {
	return &MockVectorProvider{
		store:       make(map[string][]*memory.VectorData),
		collections: make(map[string]*memory.CollectionConfig),
		indices:     make(map[string]*memory.IndexInfo),
		stats: &providers.ProviderStats{
			TotalVectors:     0,
			TotalCollections: 0,
			TotalSize:        0,
			AverageLatency:   0,
			LastOperation:    time.Now(),
			ErrorCount:       0,
			Uptime:           0,
		},
		healthy:     true,
		initialized: false,
		started:     false,
	}
}

// Store mocks storing vectors
func (m *MockVectorProvider) Store(ctx context.Context, vectors []*memory.VectorData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update call count
	m.Called(ctx, vectors)

	if len(vectors) == 0 {
		return errors.New("no vectors to store")
	}

	// Store vectors
	for _, vector := range vectors {
		collection := vector.Collection
		if collection == "" {
			collection = "default"
		}

		m.store[collection] = append(m.store[collection], vector)
		m.stats.TotalVectors++
		m.stats.TotalSize += int64(len(vector.Vector) * 8) // Approximate size
	}

	m.stats.LastOperation = time.Now()
	return nil
}

// Retrieve mocks retrieving vectors by ID
func (m *MockVectorProvider) Retrieve(ctx context.Context, ids []string) ([]*memory.VectorData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, ids)

	if result := args.Get(0); result != nil {
		if vectors, ok := result.([]*memory.VectorData); ok {
			return vectors, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	// Find vectors by ID
	var results []*memory.VectorData
	for _, collection := range m.store {
		for _, vector := range collection {
			for _, id := range ids {
				if vector.ID == id {
					results = append(results, vector)
					break
				}
			}
		}
	}

	m.stats.LastOperation = time.Now()
	return results, nil
}

// Search mocks vector similarity search
func (m *MockVectorProvider) Search(ctx context.Context, query *memory.VectorQuery) (*memory.VectorSearchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, query)

	if result := args.Get(0); result != nil {
		if searchResult, ok := result.(*memory.VectorSearchResult); ok {
			return searchResult, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	// Mock search results
	var results []*memory.VectorSearchResultItem
	collection := query.Collection
	if collection == "" {
		collection = "default"
	}

	if vectors, exists := m.store[collection]; exists {
		for i, vector := range vectors {
			if i >= query.TopK {
				break
			}

			// Mock similarity score
			score := 1.0 - float64(i)*0.1
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
	}

	m.stats.LastOperation = time.Now()
	return &memory.VectorSearchResult{
		Results:   results,
		Total:     len(results),
		Query:     query,
		Duration:  time.Since(m.stats.LastOperation),
		Namespace: query.Namespace,
	}, nil
}

// FindSimilar mocks finding similar vectors
func (m *MockVectorProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*memory.VectorSimilarityResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, embedding, k, filters)

	if result := args.Get(0); result != nil {
		if similarResults, ok := result.([]*memory.VectorSimilarityResult); ok {
			return similarResults, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	// Mock similar results
	var results []*memory.VectorSimilarityResult
	for _, collection := range m.store {
		for i, vector := range collection {
			if i >= k {
				break
			}

			// Mock similarity score
			score := 0.9 - float64(i)*0.1

			results = append(results, &memory.VectorSimilarityResult{
				ID:       vector.ID,
				Vector:   vector.Vector,
				Metadata: vector.Metadata,
				Score:    score,
				Distance: 1 - score,
			})
		}
	}

	return results, nil
}

// CreateCollection mocks creating a collection
func (m *MockVectorProvider) CreateCollection(ctx context.Context, name string, config *memory.CollectionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, name, config)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	if _, exists := m.collections[name]; exists {
		return fmt.Errorf("collection %s already exists", name)
	}

	m.collections[name] = config
	m.store[name] = make([]*memory.VectorData, 0)
	m.stats.TotalCollections++
	m.stats.LastOperation = time.Now()

	return nil
}

// DeleteCollection mocks deleting a collection
func (m *MockVectorProvider) DeleteCollection(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, name)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	if _, exists := m.collections[name]; !exists {
		return fmt.Errorf("collection %s not found", name)
	}

	delete(m.collections, name)
	delete(m.store, name)
	m.stats.TotalCollections--
	m.stats.LastOperation = time.Now()

	return nil
}

// ListCollections mocks listing collections
func (m *MockVectorProvider) ListCollections(ctx context.Context) ([]*memory.CollectionInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx)

	if result := args.Get(0); result != nil {
		if collections, ok := result.([]*memory.CollectionInfo); ok {
			return collections, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	var result []*memory.CollectionInfo
	for name, config := range m.collections {
		size := len(m.store[name])
		result = append(result, &memory.CollectionInfo{
			Name:         name,
			Dimension:    config.Dimension,
			Metric:       config.Metric,
			VectorsCount: int64(size),
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
	}

	return result, nil
}

// GetCollection mocks getting collection info
func (m *MockVectorProvider) GetCollection(ctx context.Context, name string) (*memory.CollectionInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, name)

	if result := args.Get(0); result != nil {
		if collection, ok := result.(*memory.CollectionInfo); ok {
			return collection, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	config, exists := m.collections[name]
	if !exists {
		return nil, fmt.Errorf("collection %s not found", name)
	}

	size := len(m.store[name])
	return &memory.CollectionInfo{
		Name:         name,
		Dimension:    config.Dimension,
		Metric:       config.Metric,
		VectorsCount: int64(size),
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// CreateIndex mocks creating an index
func (m *MockVectorProvider) CreateIndex(ctx context.Context, collection string, config *memory.IndexConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, collection, config)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	indexName := fmt.Sprintf("%s_%s", collection, config.Name)
	m.indices[indexName] = &memory.IndexInfo{
		Name:      config.Name,
		Type:      config.Type,
		Dimension: config.Dimension,
		Metric:    config.Metric,
		CreatedAt: time.Now(),
	}

	return nil
}

// DeleteIndex mocks deleting an index
func (m *MockVectorProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, collection, name)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	indexName := fmt.Sprintf("%s_%s", collection, name)
	delete(m.indices, indexName)

	return nil
}

// ListIndexes mocks listing indexes
func (m *MockVectorProvider) ListIndexes(ctx context.Context, collection string) ([]*memory.IndexInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, collection)

	if result := args.Get(0); result != nil {
		if indices, ok := result.([]*memory.IndexInfo); ok {
			return indices, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	var result []*memory.IndexInfo
	for name, index := range m.indices {
		if len(name) > len(collection) && name[:len(collection)] == collection {
			result = append(result, index)
		}
	}

	return result, nil
}

// AddMetadata mocks adding metadata to vectors
func (m *MockVectorProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, id, metadata)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	// Find vector and add metadata
	for _, collection := range m.store {
		for _, vector := range collection {
			if vector.ID == id {
				if vector.Metadata == nil {
					vector.Metadata = make(map[string]interface{})
				}
				for k, v := range metadata {
					vector.Metadata[k] = v
				}
				return nil
			}
		}
	}

	return fmt.Errorf("vector with ID %s not found", id)
}

// UpdateMetadata mocks updating vector metadata
func (m *MockVectorProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, id, metadata)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	// Find vector and update metadata
	for _, collection := range m.store {
		for _, vector := range collection {
			if vector.ID == id {
				if vector.Metadata == nil {
					vector.Metadata = make(map[string]interface{})
				}
				// Update existing metadata
				for k, v := range metadata {
					vector.Metadata[k] = v
				}
				return nil
			}
		}
	}

	return fmt.Errorf("vector with ID %s not found", id)
}

// GetMetadata mocks getting vector metadata
func (m *MockVectorProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, ids)

	if result := args.Get(0); result != nil {
		if metadata, ok := result.(map[string]map[string]interface{}); ok {
			return metadata, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	result := make(map[string]map[string]interface{})
	for _, collection := range m.store {
		for _, vector := range collection {
			for _, id := range ids {
				if vector.ID == id {
					result[id] = vector.Metadata
					break
				}
			}
		}
	}

	return result, nil
}

// DeleteMetadata mocks deleting vector metadata
func (m *MockVectorProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, ids, keys)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	// Delete metadata keys from specified vectors
	for _, collection := range m.store {
		for _, vector := range collection {
			for _, id := range ids {
				if vector.ID == id && vector.Metadata != nil {
					for _, key := range keys {
						delete(vector.Metadata, key)
					}
					break
				}
			}
		}
	}

	return nil
}

// GetStats mocks getting provider statistics
func (m *MockVectorProvider) GetStats(ctx context.Context) (*providers.ProviderStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx)

	if result := args.Get(0); result != nil {
		if stats, ok := result.(*providers.ProviderStats); ok {
			return stats, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	// Return copy of stats
	return &providers.ProviderStats{
		TotalVectors:     m.stats.TotalVectors,
		TotalCollections: m.stats.TotalCollections,
		TotalSize:        m.stats.TotalSize,
		AverageLatency:   m.stats.AverageLatency,
		LastOperation:    m.stats.LastOperation,
		ErrorCount:       m.stats.ErrorCount,
		Uptime:           m.stats.Uptime,
	}, nil
}

// Optimize mocks optimizing the provider
func (m *MockVectorProvider) Optimize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	m.stats.LastOperation = time.Now()
	return nil
}

// Backup mocks backing up the provider
func (m *MockVectorProvider) Backup(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, path)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	m.stats.LastOperation = time.Now()
	return nil
}

// Restore mocks restoring the provider
func (m *MockVectorProvider) Restore(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, path)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	m.stats.LastOperation = time.Now()
	return nil
}

// Initialize mocks initializing the provider
func (m *MockVectorProvider) Initialize(ctx context.Context, config interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, config)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	m.initialized = true
	m.stats.LastOperation = time.Now()
	return nil
}

// Start mocks starting the provider
func (m *MockVectorProvider) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	if !m.initialized {
		return errors.New("provider not initialized")
	}

	m.started = true
	m.stats.LastOperation = time.Now()
	m.stats.Uptime = time.Since(time.Now())

	return nil
}

// Stop mocks stopping the provider
func (m *MockVectorProvider) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	m.started = false
	m.stats.LastOperation = time.Now()

	return nil
}

// Health mocks health check
func (m *MockVectorProvider) Health(ctx context.Context) (*providers.HealthStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx)

	if result := args.Get(0); result != nil {
		if health, ok := result.(*providers.HealthStatus); ok {
			return health, nil
		}
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	status := "healthy"
	if !m.healthy {
		status = "unhealthy"
	}

	return &providers.HealthStatus{
		Status:       status,
		LastCheck:    time.Now(),
		ResponseTime: 10 * time.Millisecond,
		Metrics:      make(map[string]float64),
		Dependencies: make(map[string]string),
	}, nil
}

// GetName mocks getting provider name
func (m *MockVectorProvider) GetName() string {
	args := m.Called()
	if name := args.Get(0); name != nil {
		return name.(string)
	}
	return "MockVectorProvider"
}

// GetType mocks getting provider type
func (m *MockVectorProvider) GetType() providers.ProviderType {
	args := m.Called()
	if ptype := args.Get(0); ptype != nil {
		return ptype.(providers.ProviderType)
	}
	return providers.ProviderTypeChromaDB
}

// GetCapabilities mocks getting provider capabilities
func (m *MockVectorProvider) GetCapabilities() []string {
	args := m.Called()
	if caps := args.Get(0); caps != nil {
		return caps.([]string)
	}
	return []string{
		"vector_storage",
		"vector_search",
		"metadata_filtering",
		"batch_operations",
		"collection_management",
	}
}

// GetConfiguration mocks getting provider configuration
func (m *MockVectorProvider) GetConfiguration() interface{} {
	args := m.Called()
	if config := args.Get(0); config != nil {
		return config
	}
	return map[string]interface{}{
		"type":    "mock",
		"enabled": true,
	}
}

// IsCloud mocks checking if provider is cloud-based
func (m *MockVectorProvider) IsCloud() bool {
	args := m.Called()
	if isCloud := args.Get(0); isCloud != nil {
		return isCloud.(bool)
	}
	return false
}

// GetCostInfo mocks getting cost information
func (m *MockVectorProvider) GetCostInfo() *providers.CostInfo {
	args := m.Called()
	if costInfo := args.Get(0); costInfo != nil {
		return costInfo.(*providers.CostInfo)
	}
	return &providers.CostInfo{
		StorageCost:   0.0,
		ComputeCost:   0.0,
		TransferCost:  0.0,
		TotalCost:     0.0,
		Currency:      "USD",
		BillingPeriod: "monthly",
		FreeTierUsed:  false,
		FreeTierLimit: 0.0,
	}
}

// Helper methods for testing

// SetHealth sets the health status of the mock provider
func (m *MockVectorProvider) SetHealth(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

// AddTestData adds test data to the mock provider
func (m *MockVectorProvider) AddTestData(vectors []*memory.VectorData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, vector := range vectors {
		collection := vector.Collection
		if collection == "" {
			collection = "default"
		}

		m.store[collection] = append(m.store[collection], vector)
		m.stats.TotalVectors++
		m.stats.TotalSize += int64(len(vector.Vector) * 8)
	}
}

// ClearTestData clears all test data from the mock provider
func (m *MockVectorProvider) ClearTestData() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store = make(map[string][]*memory.VectorData)
	m.collections = make(map[string]*memory.CollectionConfig)
	m.indices = make(map[string]*memory.IndexInfo)
	m.stats.TotalVectors = 0
	m.stats.TotalCollections = 0
	m.stats.TotalSize = 0
}

// GetStoredVectors returns the stored vectors for testing
func (m *MockVectorProvider) GetStoredVectors(collection string) []*memory.VectorData {
	m.mu.Lock()
	defer m.mu.Unlock()

	if collection == "" {
		collection = "default"
	}

	vectors := make([]*memory.VectorData, len(m.store[collection]))
	copy(vectors, m.store[collection])
	return vectors
}

// MockVectorProviderManager mocks the VectorProviderManager
type MockVectorProviderManager struct {
	mock.Mock
	mu sync.Mutex

	providers map[string]providers.VectorProvider
	active    string
}

// NewMockVectorProviderManager creates a new mock vector provider manager
func NewMockVectorProviderManager(t interface{}) *MockVectorProviderManager {
	return &MockVectorProviderManager{
		providers: make(map[string]providers.VectorProvider),
		active:    "default",
	}
}

// Initialize mocks initializing the manager
func (m *MockVectorProviderManager) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Shutdown mocks shutting down the manager
func (m *MockVectorProviderManager) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Store mocks storing vectors
func (m *MockVectorProviderManager) Store(ctx context.Context, vectors []*memory.VectorData) error {
	args := m.Called(ctx, vectors)
	return args.Error(0)
}

// Retrieve mocks retrieving vectors
func (m *MockVectorProviderManager) Retrieve(ctx context.Context, ids []string) ([]*memory.VectorData, error) {
	args := m.Called(ctx, ids)
	if result := args.Get(0); result != nil {
		return result.([]*memory.VectorData)
	}
	return nil, args.Error(1)
}

// Search mocks searching vectors
func (m *MockVectorProviderManager) Search(ctx context.Context, query *memory.VectorQuery) (*memory.VectorSearchResult, error) {
	args := m.Called(ctx, query)
	if result := args.Get(0); result != nil {
		return result.(*memory.VectorSearchResult)
	}
	return nil, args.Error(1)
}

// GetActiveProvider mocks getting the active provider
func (m *MockVectorProviderManager) GetActiveProvider() string {
	args := m.Called()
	if provider := args.Get(0); provider != nil {
		return provider.(string)
	}
	return m.active
}

// SetActiveProvider mocks setting the active provider
func (m *MockVectorProviderManager) SetActiveProvider(ctx context.Context, provider string) error {
	args := m.Called(ctx, provider)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.active = provider
		m.mu.Unlock()
	}
	return args.Error(0)
}

// ListProviders mocks listing providers
func (m *MockVectorProviderManager) ListProviders() map[string]providers.ProviderInfo {
	args := m.Called()
	if providers := args.Get(0); providers != nil {
		return providers.(map[string]providers.ProviderInfo)
	}
	return make(map[string]providers.ProviderInfo)
}

// GetProviderHealth mocks getting provider health
func (m *MockVectorProviderManager) GetProviderHealth(ctx context.Context) (map[string]*providers.HealthStatus, error) {
	args := m.Called(ctx)
	if health := args.Get(0); health != nil {
		return health.(map[string]*providers.HealthStatus)
	}
	return nil, args.Error(1)
}

// GetProviderPerformance mocks getting provider performance
func (m *MockVectorProviderManager) GetProviderPerformance() map[string]providers.ProviderPerformance {
	args := m.Called()
	if perf := args.Get(0); perf != nil {
		return perf.(map[string]providers.ProviderPerformance)
	}
	return make(map[string]providers.ProviderPerformance)
}

// MockAPIKeyManager mocks the APIKeyManager
type MockAPIKeyManager struct {
	mock.Mock
	mu sync.Mutex

	keys map[string]config.APIKeyConfig
}

// NewMockAPIKeyManager creates a new mock API key manager
func NewMockAPIKeyManager(t interface{}) *MockAPIKeyManager {
	return &MockAPIKeyManager{
		keys: make(map[string]config.APIKeyConfig),
	}
}

// GetAPIKey mocks getting an API key
func (m *MockAPIKeyManager) GetAPIKey(provider string) (string, error) {
	args := m.Called(provider)
	if key := args.Get(0); key != nil {
		return key.(string)
	}
	return "", args.Error(1)
}

// SetAPIKey mocks setting an API key
func (m *MockAPIKeyManager) SetAPIKey(provider, key string) error {
	args := m.Called(provider, key)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.keys[provider] = config.APIKeyConfig{
			PrimaryKeys: []string{key},
		}
		m.mu.Unlock()
	}
	return args.Error(0)
}

// RotateAPIKey mocks rotating an API key
func (m *MockAPIKeyManager) RotateAPIKey(provider string) error {
	args := m.Called(provider)
	return args.Error(0)
}

// MockMemoryManager mocks the MemoryManager
type MockMemoryManager struct {
	mock.Mock
}

// NewMockMemoryManager creates a new mock memory manager
func NewMockMemoryManager(t interface{}) *MockMemoryManager {
	return &MockMemoryManager{}
}

// Initialize mocks initializing the memory manager
func (m *MockMemoryManager) Initialize(ctx context.Context, config interface{}) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

// Shutdown mocks shutting down the memory manager
func (m *MockMemoryManager) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Store mocks storing memory data
func (m *MockMemoryManager) Store(ctx context.Context, data *memory.MemoryData) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

// Retrieve mocks retrieving memory data
func (m *MockMemoryManager) Retrieve(ctx context.Context, id string) (*memory.MemoryData, error) {
	args := m.Called(ctx, id)
	if data := args.Get(0); data != nil {
		return data.(*memory.MemoryData)
	}
	return nil, args.Error(1)
}

// Search mocks searching memory data
func (m *MockMemoryManager) Search(ctx context.Context, query *memory.SearchQuery) (*memory.SearchResult, error) {
	args := m.Called(ctx, query)
	if result := args.Get(0); result != nil {
		return result.(*memory.SearchResult)
	}
	return nil, args.Error(1)
}

// MockConversationManager mocks the ConversationManager
type MockConversationManager struct {
	mock.Mock
}

// NewMockConversationManager creates a new mock conversation manager
func NewMockConversationManager(t interface{}) *MockConversationManager {
	return &MockConversationManager{}
}

// AddMessage mocks adding a message to conversation
func (m *MockConversationManager) AddMessage(ctx context.Context, sessionID string, message *memory.ConversationMessage) error {
	args := m.Called(ctx, sessionID, message)
	return args.Error(0)
}

// GetSummary mocks getting conversation summary
func (m *MockConversationManager) GetSummary(ctx context.Context, sessionID string) (*memory.ConversationSummary, error) {
	args := m.Called(ctx, sessionID)
	if summary := args.Get(0); summary != nil {
		return summary.(*memory.ConversationSummary)
	}
	return nil, args.Error(1)
}

// GetContextWindow mocks getting context window
func (m *MockConversationManager) GetContextWindow(ctx context.Context, sessionID string, limit int) ([]*memory.ConversationMessage, error) {
	args := m.Called(ctx, sessionID, limit)
	if messages := args.Get(0); messages != nil {
		return messages.([]*memory.ConversationMessage)
	}
	return nil, args.Error(1)
}

// Test utilities for creating test data

// CreateTestVector creates a test vector for testing
func CreateTestVector(id, collection string, size int) *memory.VectorData {
	return &memory.VectorData{
		ID:     id,
		Vector: makeTestFloat64Slice(size),
		Metadata: map[string]interface{}{
			"test":    true,
			"created": time.Now(),
		},
		Collection: collection,
		Timestamp:  time.Now(),
	}
}

// CreateTestVectors creates multiple test vectors
func CreateTestVectors(count int, collection string, size int) []*memory.VectorData {
	vectors := make([]*memory.VectorData, count)
	for i := 0; i < count; i++ {
		vectors[i] = CreateTestVector(
			fmt.Sprintf("test_vector_%d", i),
			collection,
			size,
		)
	}
	return vectors
}

// CreateTestMemory creates test memory data
func CreateTestMemory(id, memType, content string) *memory.MemoryData {
	return &memory.MemoryData{
		ID:      id,
		Type:    memory.MemoryType(memType),
		Content: content,
		Source:  "test",
		Metadata: map[string]interface{}{
			"test":    true,
			"created": time.Now(),
		},
		Timestamp: time.Now(),
	}
}

// CreateTestConversationMessage creates a test conversation message
func CreateTestConversationMessage(id, role, content string) *memory.ConversationMessage {
	return &memory.ConversationMessage{
		ID:        id,
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"test": true,
		},
	}
}

// createTestFloat64Slice creates a test float64 slice
func createTestFloat64Slice(size int) []float64 {
	slice := make([]float64, size)
	for i := 0; i < size; i++ {
		slice[i] = float64(i) / float64(size)
	}
	return slice
}

// MockSuite is a test suite that includes mock providers
type MockSuite struct {
	suite.Suite
	ctx                 context.Context
	logger              logging.Logger
	mockProvider        *MockVectorProvider
	mockProviderManager *MockVectorProviderManager
	mockAPIKeyManager   *MockAPIKeyManager
	mockMemoryManager   *MockMemoryManager
	mockConvManager     *MockConversationManager
}

// SetupSuite sets up the test suite with mocks
func (s *MockSuite) SetupSuite() {
	s.ctx = context.Background()
	s.logger = logging.NewTestLogger("mock_test")
	s.mockProvider = NewMockVectorProvider(s.T())
	s.mockProviderManager = NewMockVectorProviderManager(s.T())
	s.mockAPIKeyManager = NewMockAPIKeyManager(s.T())
	s.mockMemoryManager = NewMockMemoryManager(s.T())
	s.mockConvManager = NewMockConversationManager(s.T())
}

// TearDownSuite tears down the test suite
func (s *MockSuite) TearDownSuite() {
	// Cleanup if needed
}

// SetupTest sets up each individual test
func (s *MockSuite) SetupTest() {
	s.mockProvider.ClearTestData()
	s.mockProvider.SetHealth(true)
}

// CreateMockVectorProvider creates a mock vector provider with test data
func CreateMockVectorProvider(t interface{}, vectorCount int) *MockVectorProvider {
	provider := NewMockVectorProvider(t)

	// Add test data
	for i := 0; i < vectorCount; i++ {
		vector := CreateTestVector(
			fmt.Sprintf("test_%d", i),
			"test_collection",
			1536,
		)
		provider.AddTestData([]*memory.VectorData{vector})
	}

	return provider
}
