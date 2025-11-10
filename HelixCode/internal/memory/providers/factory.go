package providers

import (
	"context"
	"fmt"

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
	// TODO: Implement monitoring wrapper
	return provider
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

// TODO: Implement HybridProvider methods based on strategy
