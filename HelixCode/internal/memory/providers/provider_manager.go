package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/logging"
)

// ProviderManager manages multiple vector providers with load balancing and failover
type ProviderManager struct {
	mu        sync.RWMutex
	providers map[string]VectorProvider
	active    string
	config    *ManagerConfig
	logger    *logging.Logger
	health    map[string]*HealthStatus
}

// ManagerConfig contains configuration for the provider manager
type ManagerConfig struct {
	Providers             map[string]*SingleProviderConfig `json:"providers"`
	DefaultProvider       string                           `json:"default_provider"`
	LoadBalancing         LoadBalanceType                  `json:"load_balancing"`
	FailoverEnabled       bool                             `json:"failover_enabled"`
	RetryAttempts         int                              `json:"retry_attempts"`
	RetryBackoff          int64                            `json:"retry_backoff"`
	HealthCheckInterval   int64                            `json:"health_check_interval"`
	PerformanceMonitoring bool                             `json:"performance_monitoring"`
	CostTracking          bool                             `json:"cost_tracking"`
	BackupEnabled         bool                             `json:"backup_enabled"`
}

// NewProviderManager creates a new provider manager
func NewProviderManager(config *ManagerConfig) (*ProviderManager, error) {
	if config == nil {
		return nil, fmt.Errorf("manager config cannot be nil")
	}

	pm := &ProviderManager{
		providers: make(map[string]VectorProvider),
		config:    config,
		logger:    logging.NewLoggerWithName("provider_manager"),
		health:    make(map[string]*HealthStatus),
	}

	// Initialize providers
	for name, providerConfig := range config.Providers {
		provider, err := GetRegistry().CreateProvider(providerConfig.Type, providerConfig.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", name, err)
		}

		pm.providers[name] = provider
		pm.health[name] = &HealthStatus{
			Status:    "unknown",
			Timestamp: time.Now(),
		}
	}

	// Set default active provider
	if config.DefaultProvider != "" {
		if _, exists := pm.providers[config.DefaultProvider]; exists {
			pm.active = config.DefaultProvider
		} else {
			pm.logger.Warn("Default provider %s not found, using first available", config.DefaultProvider)
			for name := range pm.providers {
				pm.active = name
				break
			}
		}
	} else {
		// Use first provider as default
		for name := range pm.providers {
			pm.active = name
			break
		}
	}

	pm.logger.Info("Provider manager initialized with %d providers, active: %s", len(pm.providers), pm.active)
	return pm, nil
}

// Initialize initializes the provider manager
func (pm *ProviderManager) Initialize(ctx context.Context, config interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, provider := range pm.providers {
		if err := provider.Initialize(ctx, config); err != nil {
			pm.logger.Error("Failed to initialize provider %s: %v", name, err)
			return fmt.Errorf("failed to initialize provider %s: %w", name, err)
		}
		pm.health[name] = &HealthStatus{
			Status:    "initialized",
			Timestamp: time.Now(),
		}
	}

	return nil
}

// Start starts the provider manager
func (pm *ProviderManager) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, provider := range pm.providers {
		if err := provider.Start(ctx); err != nil {
			pm.logger.Error("Failed to start provider %s: %v", name, err)
			return fmt.Errorf("failed to start provider %s: %w", name, err)
		}
		pm.health[name] = &HealthStatus{
			Status:    "running",
			Timestamp: time.Now(),
		}
	}

	return nil
}

// Stop stops the provider manager
func (pm *ProviderManager) Stop(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, provider := range pm.providers {
		if err := provider.Stop(ctx); err != nil {
			pm.logger.Error("Failed to stop provider %s: %v", name, err)
			// Continue stopping other providers
		}
		pm.health[name] = &HealthStatus{
			Status:    "stopped",
			Timestamp: time.Now(),
		}
	}

	return nil
}

// Store stores vectors using the active provider
func (pm *ProviderManager) Store(ctx context.Context, vectors []*VectorData) error {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("no active provider")
	}

	return provider.Store(ctx, convertToProviderVectors(vectors))
}

// Retrieve retrieves vectors using the active provider
func (pm *ProviderManager) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("no active provider")
	}

	results, err := provider.Retrieve(ctx, ids)
	if err != nil {
		return nil, err
	}

	return convertFromProviderVectors(results), nil
}

// Search performs a vector search using the active provider
func (pm *ProviderManager) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("no active provider")
	}

	return provider.Search(ctx, convertToProviderQuery(query))
}

// FindSimilar finds similar vectors using the active provider
func (pm *ProviderManager) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("no active provider")
	}

	return provider.FindSimilar(ctx, embedding, k, filters)
}

// CreateCollection creates a collection using the active provider
func (pm *ProviderManager) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("no active provider")
	}

	return provider.CreateCollection(ctx, name, config)
}

// DeleteCollection deletes a collection using the active provider
func (pm *ProviderManager) DeleteCollection(ctx context.Context, name string) error {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("no active provider")
	}

	return provider.DeleteCollection(ctx, name)
}

// ListCollections lists collections using the active provider
func (pm *ProviderManager) ListCollections(ctx context.Context) ([]*CollectionInfo, error) {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("no active provider")
	}

	return provider.ListCollections(ctx)
}

// GetStats gets statistics from the active provider
func (pm *ProviderManager) GetStats() (*ProviderStats, error) {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("no active provider")
	}

	return provider.GetStats(context.Background())
}

// Health gets health status of all providers
func (pm *ProviderManager) Health(ctx context.Context) (map[string]*HealthStatus, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	health := make(map[string]*HealthStatus)
	for name, provider := range pm.providers {
		status, err := provider.Health(ctx)
		if err != nil {
			health[name] = &HealthStatus{
				Status:    "error",
				Message:   err.Error(),
				Timestamp: time.Now(),
			}
		} else {
			health[name] = status
		}
	}

	return health, nil
}

// Optimize optimizes the active provider
func (pm *ProviderManager) Optimize(ctx context.Context) error {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("no active provider")
	}

	return provider.Optimize(ctx)
}

// Backup backs up the active provider
func (pm *ProviderManager) Backup(ctx context.Context, path string) error {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("no active provider")
	}

	return provider.Backup(ctx, path)
}

// Restore restores the active provider
func (pm *ProviderManager) Restore(ctx context.Context, path string) error {
	pm.mu.RLock()
	provider := pm.providers[pm.active]
	pm.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("no active provider")
	}

	return provider.Restore(ctx, path)
}

// Helper functions for type conversion
func convertToProviderVectors(vectors []*VectorData) []*VectorData {
	// For now, assume they are the same type
	return vectors
}

func convertFromProviderVectors(vectors []*VectorData) []*VectorData {
	// For now, assume they are the same type
	return vectors
}

func convertToProviderQuery(query *VectorQuery) *VectorQuery {
	// For now, assume they are the same type
	return query
}
