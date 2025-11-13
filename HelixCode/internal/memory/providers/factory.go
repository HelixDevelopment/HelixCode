package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/memory"
)

// ProviderFactory creates provider instances with validation and initialization
type ProviderFactory struct {
	registry *ProviderRegistry
	config   *FactoryConfig
}

// FactoryConfig contains factory configuration
type FactoryConfig struct {
	DefaultTimeout     int64                        `json:"default_timeout"`
	EnableValidation   bool                         `json:"enable_validation"`
	EnableAutoConfig   bool                         `json:"enable_auto_config"`
	PreferredProviders []ProviderType               `json:"preferred_providers"`
	CustomConfigs      map[ProviderType]interface{} `json:"custom_configs"`
	HealthCheckOnInit  bool                         `json:"health_check_on_init"`
	FailFastOnErrors   bool                         `json:"fail_fast_on_errors"`
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(config *FactoryConfig) *ProviderFactory {
	if config == nil {
		config = &FactoryConfig{
			DefaultTimeout:    30,
			EnableValidation:  true,
			EnableAutoConfig:  true,
			HealthCheckOnInit: true,
			FailFastOnErrors:  true,
		}
	}

	return &ProviderFactory{
		registry: GetRegistry(),
		config:   config,
	}
}

// CreateProvider creates a provider with enhanced error handling and validation
func (f *ProviderFactory) CreateProvider(providerType ProviderType, config map[string]interface{}) (VectorProvider, error) {
	// Validate provider type exists
	if err := f.validateProviderType(providerType); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	// Apply auto-configuration
	if f.config.EnableAutoConfig {
		config = f.applyAutoConfiguration(providerType, config)
	}

	// Validate configuration
	if f.config.EnableValidation {
		if err := f.validateConfiguration(providerType, config); err != nil {
			return nil, fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Create provider
	provider, err := f.registry.CreateProvider(providerType, config)
	if err != nil {
		return nil, fmt.Errorf("provider creation failed: %w", err)
	}

	// Wrap with monitoring if enabled
	provider = f.wrapWithMonitoring(provider)

	return provider, nil
}

// CreateProviderWithDefaults creates a provider with default configuration
func (f *ProviderFactory) CreateProviderWithDefaults(providerType ProviderType) (VectorProvider, error) {
	defaults := f.getDefaultConfiguration(providerType)
	return f.CreateProvider(providerType, defaults)
}

// CreateProviderChain creates a chain of providers for fallback scenarios
func (f *ProviderFactory) CreateProviderChain(providerTypes []ProviderType, configs []map[string]interface{}) (*ProviderChain, error) {
	if len(providerTypes) != len(configs) {
		return nil, fmt.Errorf("provider types and configs length mismatch")
	}

	var providers []VectorProvider
	for i, providerType := range providerTypes {
		provider, err := f.CreateProvider(providerType, configs[i])
		if err != nil {
			if f.config.FailFastOnErrors {
				return nil, fmt.Errorf("failed to create provider at index %d: %w", i, err)
			}
			continue
		}
		providers = append(providers, provider)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers could be created")
	}

	return NewProviderChain(providers), nil
}

// CreateHybridProvider creates a hybrid provider that uses multiple providers for different operations
func (f *ProviderFactory) CreateHybridProvider(config *HybridProviderConfig) (*HybridProvider, error) {
	providers := make(map[string]VectorProvider)

	for operation, providerConfig := range config.Providers {
		provider, err := f.CreateProvider(providerConfig.Type, providerConfig.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider for operation %s: %w", operation, err)
		}
		providers[operation] = provider
	}

	return NewHybridProvider(config.Strategy, providers), nil
}

// validateProviderType validates that a provider type exists and is supported
func (f *ProviderFactory) validateProviderType(providerType ProviderType) error {
	_, err := f.registry.GetProviderFactory(providerType)
	if err != nil {
		return fmt.Errorf("unknown provider type: %s", providerType)
	}

	return nil
}

// validateConfiguration validates provider configuration
func (f *ProviderFactory) validateConfiguration(providerType ProviderType, config map[string]interface{}) error {
	return f.registry.ValidateProviderConfig(providerType, config)
}

// applyAutoConfiguration applies automatic configuration based on provider type
func (f *ProviderFactory) applyAutoConfiguration(providerType ProviderType, config map[string]interface{}) map[string]interface{} {
	// Start with defaults
	defaults := f.getDefaultConfiguration(providerType)

	// Merge with provided config
	result := make(map[string]interface{})
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range config {
		result[k] = v
	}

	// Apply custom configs if available
	if customConfig, exists := f.config.CustomConfigs[providerType]; exists {
		if customMap, ok := customConfig.(map[string]interface{}); ok {
			for k, v := range customMap {
				result[k] = v
			}
		}
	}

	return result
}

// getDefaultConfiguration gets default configuration for a provider type
func (f *ProviderFactory) getDefaultConfiguration(providerType ProviderType) map[string]interface{} {
	return f.registry.GetDefaultConfig(providerType)
}

// wrapWithMonitoring wraps provider with monitoring capabilities
func (f *ProviderFactory) wrapWithMonitoring(provider VectorProvider) VectorProvider {
	// Create monitoring wrapper that tracks metrics
	return &MonitoredProvider{
		provider: provider,
		metrics: &ProviderMetrics{
			TotalOperations: 0,
			SuccessCount:    0,
			FailureCount:    0,
			TotalLatency:    0,
			LastOperation:   time.Now(),
		},
	}
}

// MonitoredProvider wraps a VectorProvider with monitoring capabilities
type MonitoredProvider struct {
	provider VectorProvider
	metrics  *ProviderMetrics
	mu       sync.RWMutex
}

// ProviderMetrics tracks provider performance metrics
type ProviderMetrics struct {
	TotalOperations int64
	SuccessCount    int64
	FailureCount    int64
	TotalLatency    time.Duration
	LastOperation   time.Time
}

// Helper method to record operation metrics
func (m *MonitoredProvider) recordOperation(start time.Time, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.TotalOperations++
	m.metrics.TotalLatency += time.Since(start)
	m.metrics.LastOperation = time.Now()

	if err != nil {
		m.metrics.FailureCount++
	} else {
		m.metrics.SuccessCount++
	}
}

// GetMetrics returns current provider metrics
func (m *MonitoredProvider) GetMetrics() *ProviderMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &ProviderMetrics{
		TotalOperations: m.metrics.TotalOperations,
		SuccessCount:    m.metrics.SuccessCount,
		FailureCount:    m.metrics.FailureCount,
		TotalLatency:    m.metrics.TotalLatency,
		LastOperation:   m.metrics.LastOperation,
	}
}

// Delegate all VectorProvider methods to wrapped provider with monitoring

func (m *MonitoredProvider) GetType() string {
	return m.provider.GetType()
}

func (m *MonitoredProvider) GetName() string {
	return m.provider.GetName()
}

func (m *MonitoredProvider) GetCapabilities() []string {
	return m.provider.GetCapabilities()
}

func (m *MonitoredProvider) GetConfiguration() interface{} {
	return m.provider.GetConfiguration()
}

func (m *MonitoredProvider) IsCloud() bool {
	return m.provider.IsCloud()
}

func (m *MonitoredProvider) Store(ctx context.Context, data []*VectorData) error {
	start := time.Now()
	err := m.provider.Store(ctx, data)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	start := time.Now()
	result, err := m.provider.Search(ctx, query)
	m.recordOperation(start, err)
	return result, err
}

func (m *MonitoredProvider) Delete(ctx context.Context, ids []string) error {
	start := time.Now()
	err := m.provider.Delete(ctx, ids)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	start := time.Now()
	stats, err := m.provider.GetStats(ctx)
	m.recordOperation(start, err)
	return stats, err
}

func (m *MonitoredProvider) Health(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	health, err := m.provider.Health(ctx)
	m.recordOperation(start, err)
	return health, err
}

func (m *MonitoredProvider) Close(ctx context.Context) error {
	start := time.Now()
	err := m.provider.Close(ctx)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	start := time.Now()
	err := m.provider.AddMetadata(ctx, id, metadata)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	start := time.Now()
	err := m.provider.UpdateMetadata(ctx, id, metadata)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	start := time.Now()
	metadata, err := m.provider.GetMetadata(ctx, ids)
	m.recordOperation(start, err)
	return metadata, err
}

func (m *MonitoredProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	start := time.Now()
	err := m.provider.DeleteMetadata(ctx, ids, keys)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	start := time.Now()
	data, err := m.provider.Retrieve(ctx, ids)
	m.recordOperation(start, err)
	return data, err
}

func (m *MonitoredProvider) Update(ctx context.Context, id string, vector *VectorData) error {
	start := time.Now()
	err := m.provider.Update(ctx, id, vector)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	start := time.Now()
	results, err := m.provider.FindSimilar(ctx, embedding, k, filters)
	m.recordOperation(start, err)
	return results, err
}

func (m *MonitoredProvider) BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error) {
	start := time.Now()
	results, err := m.provider.BatchFindSimilar(ctx, queries, k)
	m.recordOperation(start, err)
	return results, err
}

func (m *MonitoredProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	start := time.Now()
	err := m.provider.CreateCollection(ctx, name, config)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) DeleteCollection(ctx context.Context, name string) error {
	start := time.Now()
	err := m.provider.DeleteCollection(ctx, name)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	start := time.Now()
	collections, err := m.provider.ListCollections(ctx)
	m.recordOperation(start, err)
	return collections, err
}

func (m *MonitoredProvider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	start := time.Now()
	collection, err := m.provider.GetCollection(ctx, name)
	m.recordOperation(start, err)
	return collection, err
}

func (m *MonitoredProvider) CreateIndex(ctx context.Context, collection string, config *IndexConfig) error {
	start := time.Now()
	err := m.provider.CreateIndex(ctx, collection, config)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	start := time.Now()
	err := m.provider.DeleteIndex(ctx, collection, name)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error) {
	start := time.Now()
	indexes, err := m.provider.ListIndexes(ctx, collection)
	m.recordOperation(start, err)
	return indexes, err
}

func (m *MonitoredProvider) Optimize(ctx context.Context) error {
	start := time.Now()
	err := m.provider.Optimize(ctx)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) Backup(ctx context.Context, path string) error {
	start := time.Now()
	err := m.provider.Backup(ctx, path)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) Restore(ctx context.Context, path string) error {
	start := time.Now()
	err := m.provider.Restore(ctx, path)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) Initialize(ctx context.Context, config interface{}) error {
	start := time.Now()
	err := m.provider.Initialize(ctx, config)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) Start(ctx context.Context) error {
	start := time.Now()
	err := m.provider.Start(ctx)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) Stop(ctx context.Context) error {
	start := time.Now()
	err := m.provider.Close(ctx)
	m.recordOperation(start, err)
	return err
}

func (m *MonitoredProvider) GetCostInfo() *CostInfo {
	return m.provider.GetCostInfo()
}

// ProviderValidator interface for providers that can validate themselves
type ProviderValidator interface {
	Validate() error
}

// ProviderChain provides fallback capability across multiple providers
type ProviderChain struct {
	providers []VectorProvider
	current   int
}

// NewProviderChain creates a new provider chain
func NewProviderChain(providers []VectorProvider) *ProviderChain {
	return &ProviderChain{
		providers: providers,
		current:   0,
	}
}

// Initialize initializes all providers in the chain
func (pc *ProviderChain) Initialize(ctx context.Context, config interface{}) error {
	for _, provider := range pc.providers {
		if err := provider.Initialize(ctx, config); err != nil {
			return err
		}
	}
	return nil
}

// Start starts all providers in the chain
func (pc *ProviderChain) Start(ctx context.Context) error {
	for _, provider := range pc.providers {
		if err := provider.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Store stores vectors using the current provider, with fallback
func (pc *ProviderChain) Store(ctx context.Context, vectors []*memory.VectorData) error {
	providerVectors := convertMemoryVectorDataSliceToProvider(vectors)
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		err := provider.Store(ctx, providerVectors)
		if err == nil {
			return nil
		}
		// Try next provider
		pc.current = i + 1
	}
	return fmt.Errorf("all providers in chain failed to store vectors")
}

// Retrieve retrieves vectors using the current provider, with fallback
func (pc *ProviderChain) Retrieve(ctx context.Context, ids []string) ([]*memory.VectorData, error) {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		result, err := provider.Retrieve(ctx, ids)
		if err == nil {
			return convertProviderVectorDataSliceToMemory(result), nil
		}
		// Try next provider
		pc.current = i + 1
	}
	return nil, fmt.Errorf("all providers in chain failed to retrieve vectors")
}

// Search performs search using the current provider, with fallback
func (pc *ProviderChain) Search(ctx context.Context, query *memory.VectorQuery) (*memory.VectorSearchResult, error) {
	providerQuery := convertMemoryVectorQueryToProvider(query)
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		result, err := provider.Search(ctx, providerQuery)
		if err == nil {
			return convertProviderVectorSearchResultToMemory(result), nil
		}
		// Try next provider
		pc.current = i + 1
	}
	return nil, fmt.Errorf("all providers in chain failed to search")
}

// FindSimilar finds similar vectors using the current provider, with fallback
func (pc *ProviderChain) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*memory.VectorSimilarityResult, error) {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		result, err := provider.FindSimilar(ctx, embedding, k, filters)
		if err == nil {
			return convertProviderVectorSimilarityResultSliceToMemorySingle(result), nil
		}
		// Try next provider
		pc.current = i + 1
	}
	return nil, fmt.Errorf("all providers in chain failed to find similar vectors")
}

// Implement other required methods with fallback logic
func (pc *ProviderChain) CreateCollection(ctx context.Context, name string, config *memory.CollectionConfig) error {
	providerConfig := convertMemoryCollectionConfigToProvider(config)
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		err := provider.CreateCollection(ctx, name, providerConfig)
		if err == nil {
			return nil
		}
		pc.current = i + 1
	}
	return fmt.Errorf("all providers in chain failed to create collection")
}

func (pc *ProviderChain) DeleteCollection(ctx context.Context, name string) error {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		err := provider.DeleteCollection(ctx, name)
		if err == nil {
			return nil
		}
		pc.current = i + 1
	}
	return fmt.Errorf("all providers in chain failed to delete collection")
}

func (pc *ProviderChain) ListCollections(ctx context.Context) ([]*memory.CollectionInfo, error) {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		result, err := provider.ListCollections(ctx)
		if err == nil {
			return convertProviderCollectionInfoSliceToMemory(result), nil
		}
		pc.current = i + 1
	}
	return nil, fmt.Errorf("all providers in chain failed to list collections")
}

func (pc *ProviderChain) GetCollection(ctx context.Context, name string) (*memory.CollectionInfo, error) {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		result, err := provider.GetCollection(ctx, name)
		if err == nil {
			return convertProviderCollectionInfoToMemory(result), nil
		}
		pc.current = i + 1
	}
	return nil, fmt.Errorf("all providers in chain failed to get collection")
}

func (pc *ProviderChain) CreateIndex(ctx context.Context, collection string, config *memory.IndexConfig) error {
	providerConfig := convertMemoryIndexConfigToProvider(config)
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		err := provider.CreateIndex(ctx, collection, providerConfig)
		if err == nil {
			return nil
		}
		pc.current = i + 1
	}
	return fmt.Errorf("all providers in chain failed to create index")
}

func (pc *ProviderChain) DeleteIndex(ctx context.Context, collection, name string) error {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		err := provider.DeleteIndex(ctx, collection, name)
		if err == nil {
			return nil
		}
		pc.current = i + 1
	}
	return fmt.Errorf("all providers in chain failed to delete index")
}

func (pc *ProviderChain) ListIndexes(ctx context.Context, collection string) ([]*memory.IndexInfo, error) {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		result, err := provider.ListIndexes(ctx, collection)
		if err == nil {
			return convertProviderIndexInfoSliceToMemory(result), nil
		}
		pc.current = i + 1
	}
	return nil, fmt.Errorf("all providers in chain failed to list indexes")
}

func (pc *ProviderChain) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		err := provider.AddMetadata(ctx, id, metadata)
		if err == nil {
			return nil
		}
		pc.current = i + 1
	}
	return fmt.Errorf("all providers in chain failed to add metadata")
}

func (pc *ProviderChain) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		err := provider.UpdateMetadata(ctx, id, metadata)
		if err == nil {
			return nil
		}
		pc.current = i + 1
	}
	return fmt.Errorf("all providers in chain failed to update metadata")
}

func (pc *ProviderChain) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		result, err := provider.GetMetadata(ctx, ids)
		if err == nil {
			return result, nil
		}
		pc.current = i + 1
	}
	return nil, fmt.Errorf("all providers in chain failed to get metadata")
}

func (pc *ProviderChain) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	for i := pc.current; i < len(pc.providers); i++ {
		provider := pc.providers[i]
		err := provider.DeleteMetadata(ctx, ids, keys)
		if err == nil {
			return nil
		}
		pc.current = i + 1
	}
	return fmt.Errorf("all providers in chain failed to delete metadata")
}

func (pc *ProviderChain) GetStats(ctx context.Context) (*memory.ProviderStats, error) {
	// Return stats from current provider
	if pc.current < len(pc.providers) {
		stats, err := pc.providers[pc.current].GetStats(ctx)
		if err != nil {
			return nil, err
		}
		return convertProviderStatsToMemory(stats), nil
	}
	return nil, fmt.Errorf("no active providers")
}

func (pc *ProviderChain) Optimize(ctx context.Context) error {
	for _, provider := range pc.providers {
		if err := provider.Optimize(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (pc *ProviderChain) Backup(ctx context.Context, path string) error {
	for _, provider := range pc.providers {
		if err := provider.Backup(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

func (pc *ProviderChain) Restore(ctx context.Context, path string) error {
	for _, provider := range pc.providers {
		if err := provider.Restore(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

func (pc *ProviderChain) Health(ctx context.Context) (*HealthStatus, error) {
	// Return health from current provider
	if pc.current < len(pc.providers) {
		return pc.providers[pc.current].Health(ctx)
	}
	return nil, fmt.Errorf("no active providers")
}

func (pc *ProviderChain) GetName() string {
	return "provider_chain"
}

func (pc *ProviderChain) GetType() ProviderType {
	return ProviderTypeAgnostic
}

func (pc *ProviderChain) GetCapabilities() []string {
	capabilities := make(map[string]bool)
	for _, provider := range pc.providers {
		for _, cap := range provider.GetCapabilities() {
			capabilities[cap] = true
		}
	}

	var result []string
	for cap := range capabilities {
		result = append(result, cap)
	}
	return result
}

func (pc *ProviderChain) GetConfiguration() interface{} {
	// Return configuration of current provider
	if pc.current < len(pc.providers) {
		return pc.providers[pc.current].GetConfiguration()
	}
	return nil
}

func (pc *ProviderChain) IsCloud() bool {
	// Return cloud status of current provider
	if pc.current < len(pc.providers) {
		return pc.providers[pc.current].IsCloud()
	}
	return false
}

func (pc *ProviderChain) GetCostInfo() *memory.CostInfo {
	// Return cost info from current provider
	if pc.current < len(pc.providers) {
		costInfo := pc.providers[pc.current].GetCostInfo()
		return convertProviderCostInfoToMemory(costInfo)
	}
	return nil
}

func (pc *ProviderChain) Stop(ctx context.Context) error {
	for _, provider := range pc.providers {
		if err := provider.Stop(ctx); err != nil {
			return err
		}
	}
	return nil
}

// HybridProviderConfig contains configuration for hybrid provider
type HybridProviderConfig struct {
	Strategy  HybridStrategy         `json:"strategy"`
	Providers map[string]ProviderRef `json:"providers"`
}

// ProviderRef contains reference to a provider
type ProviderRef struct {
	Type   ProviderType           `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// HybridStrategy defines hybrid provider strategy
type HybridStrategy string

const (
	HybridStrategyFailover       HybridStrategy = "failover"
	HybridStrategyRoundRobin     HybridStrategy = "round_robin"
	HybridStrategyLoadBalance    HybridStrategy = "load_balance"
	HybridStrategyOperationBased HybridStrategy = "operation_based"
)

// HybridProvider routes operations to different providers based on strategy
type HybridProvider struct {
	strategy   HybridStrategy
	providers  map[string]VectorProvider
	roundRobin int
}

// NewHybridProvider creates a new hybrid provider
func NewHybridProvider(strategy HybridStrategy, providers map[string]VectorProvider) *HybridProvider {
	return &HybridProvider{
		strategy:  strategy,
		providers: providers,
	}
}

// HybridProvider methods implementing VectorProvider interface based on strategy

func (hp *HybridProvider) selectProvider(operation string) VectorProvider {
	switch hp.strategy {
	case HybridStrategyFailover:
		// Use first available provider
		for _, provider := range hp.providers {
			return provider
		}
	case HybridStrategyRoundRobin:
		// Rotate through providers
		providers := make([]VectorProvider, 0, len(hp.providers))
		for _, p := range hp.providers {
			providers = append(providers, p)
		}
		if len(providers) > 0 {
			selected := providers[hp.roundRobin%len(providers)]
			hp.roundRobin++
			return selected
		}
	case HybridStrategyLoadBalance:
		// Simple load balancing - could be enhanced with metrics
		return hp.selectProvider("roundrobin")
	}
	// Default: return first provider
	for _, provider := range hp.providers {
		return provider
	}
	return nil
}

func (hp *HybridProvider) GetType() string {
	return "hybrid"
}

func (hp *HybridProvider) GetName() string {
	return fmt.Sprintf("Hybrid(%s)", hp.strategy)
}

func (hp *HybridProvider) GetCapabilities() []string {
	// Aggregate capabilities from all providers
	capMap := make(map[string]bool)
	for _, provider := range hp.providers {
		for _, cap := range provider.GetCapabilities() {
			capMap[cap] = true
		}
	}
	caps := make([]string, 0, len(capMap))
	for cap := range capMap {
		caps = append(caps, cap)
	}
	return caps
}

func (hp *HybridProvider) GetConfiguration() interface{} {
	configs := make(map[string]interface{})
	for name, provider := range hp.providers {
		configs[name] = provider.GetConfiguration()
	}
	return map[string]interface{}{
		"strategy":  hp.strategy,
		"providers": configs,
	}
}

func (hp *HybridProvider) IsCloud() bool {
	// If any provider is cloud-based, return true
	for _, provider := range hp.providers {
		if provider.IsCloud() {
			return true
		}
	}
	return false
}

func (hp *HybridProvider) Store(ctx context.Context, data []*VectorData) error {
	provider := hp.selectProvider("store")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.Store(ctx, data)
}

func (hp *HybridProvider) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	provider := hp.selectProvider("search")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.Search(ctx, query)
}

func (hp *HybridProvider) Delete(ctx context.Context, ids []string) error {
	provider := hp.selectProvider("delete")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.Delete(ctx, ids)
}

func (hp *HybridProvider) GetStats(ctx context.Context) (*ProviderStats, error) {
	provider := hp.selectProvider("stats")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.GetStats(ctx)
}

func (hp *HybridProvider) Health(ctx context.Context) (*HealthStatus, error) {
	// Aggregate health from all providers
	allHealthy := true
	messages := []string{}
	for name, provider := range hp.providers {
		health, err := provider.Health(ctx)
		if err != nil || (health != nil && health.Status != "healthy") {
			allHealthy = false
			msg := fmt.Sprintf("%s: unhealthy", name)
			if err != nil {
				msg = fmt.Sprintf("%s (%v)", msg, err)
			}
			messages = append(messages, msg)
		}
	}

	status := "healthy"
	message := "All providers healthy"
	if !allHealthy {
		status = "degraded"
		message = strings.Join(messages, "; ")
	}

	return &HealthStatus{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}, nil
}

func (hp *HybridProvider) Close(ctx context.Context) error {
	// Close all providers
	var errs []string
	for name, provider := range hp.providers {
		if err := provider.Close(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (hp *HybridProvider) AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	provider := hp.selectProvider("metadata")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.AddMetadata(ctx, id, metadata)
}

func (hp *HybridProvider) UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	provider := hp.selectProvider("metadata")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.UpdateMetadata(ctx, id, metadata)
}

func (hp *HybridProvider) GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	provider := hp.selectProvider("metadata")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.GetMetadata(ctx, ids)
}

func (hp *HybridProvider) DeleteMetadata(ctx context.Context, ids []string, keys []string) error {
	provider := hp.selectProvider("metadata")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.DeleteMetadata(ctx, ids, keys)
}

func (hp *HybridProvider) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	provider := hp.selectProvider("retrieve")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.Retrieve(ctx, ids)
}

func (hp *HybridProvider) Update(ctx context.Context, id string, vector *VectorData) error {
	provider := hp.selectProvider("update")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.Update(ctx, id, vector)
}

func (hp *HybridProvider) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	provider := hp.selectProvider("similar")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.FindSimilar(ctx, embedding, k, filters)
}

func (hp *HybridProvider) BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error) {
	provider := hp.selectProvider("batch_similar")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.BatchFindSimilar(ctx, queries, k)
}

func (hp *HybridProvider) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	provider := hp.selectProvider("collection")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.CreateCollection(ctx, name, config)
}

func (hp *HybridProvider) DeleteCollection(ctx context.Context, name string) error {
	provider := hp.selectProvider("collection")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.DeleteCollection(ctx, name)
}

func (hp *HybridProvider) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	provider := hp.selectProvider("collection")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.ListCollections(ctx)
}

func (hp *HybridProvider) GetCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	provider := hp.selectProvider("collection")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.GetCollection(ctx, name)
}

func (hp *HybridProvider) CreateIndex(ctx context.Context, collection string, config *IndexConfig) error {
	provider := hp.selectProvider("index")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.CreateIndex(ctx, collection, config)
}

func (hp *HybridProvider) DeleteIndex(ctx context.Context, collection, name string) error {
	provider := hp.selectProvider("index")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.DeleteIndex(ctx, collection, name)
}

func (hp *HybridProvider) ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error) {
	provider := hp.selectProvider("index")
	if provider == nil {
		return nil, fmt.Errorf("no provider available")
	}
	return provider.ListIndexes(ctx, collection)
}

func (hp *HybridProvider) Optimize(ctx context.Context) error {
	provider := hp.selectProvider("optimize")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.Optimize(ctx)
}

func (hp *HybridProvider) Backup(ctx context.Context, path string) error {
	provider := hp.selectProvider("backup")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.Backup(ctx, path)
}

func (hp *HybridProvider) Restore(ctx context.Context, path string) error {
	provider := hp.selectProvider("restore")
	if provider == nil {
		return fmt.Errorf("no provider available")
	}
	return provider.Restore(ctx, path)
}

func (hp *HybridProvider) Initialize(ctx context.Context, config interface{}) error {
	// Initialize all providers
	var errs []string
	for name, provider := range hp.providers {
		if err := provider.Initialize(ctx, config); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors initializing providers: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (hp *HybridProvider) Start(ctx context.Context) error {
	// Start all providers
	var errs []string
	for name, provider := range hp.providers {
		if err := provider.Start(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors starting providers: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (hp *HybridProvider) Stop(ctx context.Context) error {
	// Stop all providers
	var errs []string
	for name, provider := range hp.providers {
		if err := provider.Stop(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors stopping providers: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (hp *HybridProvider) GetCostInfo() *CostInfo {
	// Aggregate cost from all providers
	totalCost := &CostInfo{
		Currency:      "USD",
		ComputeCost:   0,
		TransferCost:  0,
		StorageCost:   0,
		TotalCost:     0,
		BillingPeriod: "monthly",
		FreeTierUsed:  0,
		FreeTierLimit: 0,
	}

	for _, provider := range hp.providers {
		cost := provider.GetCostInfo()
		if cost != nil {
			totalCost.ComputeCost += cost.ComputeCost
			totalCost.TransferCost += cost.TransferCost
			totalCost.StorageCost += cost.StorageCost
			totalCost.TotalCost += cost.TotalCost
			totalCost.FreeTierUsed += cost.FreeTierUsed
			totalCost.FreeTierLimit += cost.FreeTierLimit
		}
	}

	return totalCost
}
