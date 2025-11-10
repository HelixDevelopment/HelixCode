package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory"
)

// ProviderManager manages multiple vector providers
type ProviderManager struct {
	mu              sync.RWMutex
	providers       map[string]VectorProvider
	defaultProvider string
	logger          *logging.Logger
	config          *ManagerConfig
	stats           *ManagerStats
}

// ManagerConfig contains manager configuration
type ManagerConfig struct {
	Providers             []ProviderConfig `json:"providers"`
	DefaultProvider       string           `json:"default_provider"`
	LoadBalancing         LoadBalanceType  `json:"load_balancing"`
	FailoverEnabled       bool             `json:"failover_enabled"`
	FailoverTimeout       time.Duration    `json:"failover_timeout"`
	RetryAttempts         int              `json:"retry_attempts"`
	RetryBackoff          time.Duration    `json:"retry_backoff"`
	HealthCheckInterval   time.Duration    `json:"health_check_interval"`
	PerformanceMonitoring bool             `json:"performance_monitoring"`
	CostTracking          bool             `json:"cost_tracking"`
	BackupEnabled         bool             `json:"backup_enabled"`
	BackupInterval        time.Duration    `json:"backup_interval"`
}

// LoadBalanceType defines load balancing strategy
type LoadBalanceType string

const (
	LoadBalanceRoundRobin LoadBalanceType = "round_robin"
	LoadBalanceLeastUsed  LoadBalanceType = "least_used"
	LoadBalanceWeighted   LoadBalanceType = "weighted"
	LoadBalanceSticky     LoadBalanceType = "sticky"
)

// ManagerStats contains manager statistics
type ManagerStats struct {
	TotalProviders       int                       `json:"total_providers"`
	ActiveProviders      int                       `json:"active_providers"`
	FailedProviders      int                       `json:"failed_providers"`
	TotalOperations      int64                     `json:"total_operations"`
	SuccessfulOperations int64                     `json:"successful_operations"`
	FailedOperations     int64                     `json:"failed_operations"`
	AverageLatency       time.Duration             `json:"average_latency"`
	TotalCost            float64                   `json:"total_cost"`
	ProviderStats        map[string]*ProviderStats `json:"provider_stats"`
	LastHealthCheck      time.Time                 `json:"last_health_check"`
	Uptime               time.Duration             `json:"uptime"`
}

// NewProviderManager creates a new provider manager
func NewProviderManager(config *ManagerConfig) (*ProviderManager, error) {
	logger := logging.NewLoggerWithName("provider_manager")

	manager := &ProviderManager{
		providers: make(map[string]VectorProvider),
		logger:    logger,
		config:    config,
		stats: &ManagerStats{
			ProviderStats: make(map[string]*ProviderStats),
		},
	}

	if err := manager.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	return manager, nil
}

// initializeProviders initializes all configured providers
func (m *ProviderManager) initializeProviders() error {
	for _, providerConfig := range m.config.Providers {
		if !providerConfig.Enabled {
			m.logger.Info("Skipping disabled provider", "name", providerConfig.Name)
			continue
		}

		provider, err := m.createProvider(providerConfig.Type, providerConfig.Config)
		if err != nil {
			m.logger.Error("Failed to create provider",
				"name", providerConfig.Name,
				"type", providerConfig.Type,
				"error", err)
			continue
		}

		// Initialize provider
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := provider.Initialize(ctx, providerConfig.Config); err != nil {
			m.logger.Error("Failed to initialize provider",
				"name", providerConfig.Name,
				"error", err)
			continue
		}

		// Start provider
		if err := provider.Start(ctx); err != nil {
			m.logger.Error("Failed to start provider",
				"name", providerConfig.Name,
				"error", err)
			continue
		}

		m.providers[providerConfig.Name] = provider
		m.stats.ProviderStats[providerConfig.Name] = &ProviderStats{
			Name:   providerConfig.Name,
			Type:   providerConfig.Type,
			Status: "active",
		}

		m.logger.Info("Provider initialized and started",
			"name", providerConfig.Name,
			"type", providerConfig.Type)

		// Set default provider if not already set
		if m.defaultProvider == "" || providerConfig.Priority > 0 {
			m.defaultProvider = providerConfig.Name
		}
	}

	m.stats.TotalProviders = len(m.config.Providers)
	m.stats.ActiveProviders = len(m.providers)
	m.stats.FailedProviders = m.stats.TotalProviders - m.stats.ActiveProviders

	if m.defaultProvider == "" && len(m.providers) > 0 {
		for name := range m.providers {
			m.defaultProvider = name
			break
		}
	}

	m.logger.Info("Provider manager initialized",
		"total_providers", m.stats.TotalProviders,
		"active_providers", m.stats.ActiveProviders,
		"default_provider", m.defaultProvider)

	return nil
}

// createProvider creates a provider instance
func (m *ProviderManager) createProvider(providerType memory.ProviderType, config map[string]interface{}) (VectorProvider, error) {
	switch providerType {
	case memory.ProviderTypePinecone:
		return NewPineconeProvider(config)
	case memory.ProviderTypeMilvus:
		return NewMilvusProvider(config)
	case memory.ProviderTypeWeaviate:
		return NewWeaviateProvider(config)
	case memory.ProviderTypeQdrant:
		return NewQdrantProvider(config)
	case memory.ProviderTypeRedis:
		return NewRedisProvider(config)
	case memory.ProviderTypeChroma:
		return NewChromaProvider(config)
	case memory.ProviderTypeOpenAI:
		return NewOpenAIProvider(config)
	case memory.ProviderTypeAnthropic:
		return NewAnthropicProvider(config)
	case memory.ProviderTypeCohere:
		return NewCohereProvider(config)
	case memory.ProviderTypeHuggingFace:
		return NewHuggingFaceProvider(config)
	case memory.ProviderTypeMistral:
		return NewMistralProvider(config)
	case memory.ProviderTypeGemini:
		return NewGeminiProvider(config)
	case memory.ProviderTypeVertexAI:
		return NewVertexAIProvider(config)
	case memory.ProviderTypeClickHouse:
		return NewClickHouseProvider(config)
	case memory.ProviderTypeSupabase:
		return NewSupabaseProvider(config)
	case memory.ProviderTypeDeepLake:
		return NewDeepLakeProvider(config)
	case memory.ProviderTypeFAISS:
		return NewFAISSProvider(config)
	case memory.ProviderTypeLlamaIndex:
		return NewLlamaIndexProvider(config)
	case memory.ProviderTypeMemGPT:
		return NewMemGPTProvider(config)
	case memory.ProviderTypeCrewAI:
		return NewCrewAIProvider(config)
	case memory.ProviderTypeCharacterAI:
		return NewCharacterAIProvider(config)
	case memory.ProviderTypeReplika:
		return NewReplikaProvider(config)
	case memory.ProviderTypeAnima:
		return NewAnimaProvider(config)
	case memory.ProviderTypeGemma:
		return NewGemmaProvider(config)
	case memory.ProviderTypeAgnostic:
		return NewProviderAgnosticProvider(config)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// Store stores vectors using the load balancing strategy
func (m *ProviderManager) Store(ctx context.Context, vectors []*memory.VectorData) error {
	start := time.Now()
	defer func() {
		m.updateOperationStats(time.Since(start), true)
	}()

	provider, err := m.selectProvider("store")
	if err != nil {
		m.updateOperationStats(time.Since(start), false)
		return err
	}

	return m.executeWithRetry(ctx, provider, func(ctx context.Context) error {
		return provider.Store(ctx, vectors)
	})
}

// Retrieve retrieves vectors using the load balancing strategy
func (m *ProviderManager) Retrieve(ctx context.Context, ids []string) ([]*memory.VectorData, error) {
	start := time.Now()
	defer func() {
		m.updateOperationStats(time.Since(start), true)
	}()

	provider, err := m.selectProvider("retrieve")
	if err != nil {
		m.updateOperationStats(time.Since(start), false)
		return nil, err
	}

	var result []*memory.VectorData
	err = m.executeWithRetry(ctx, provider, func(ctx context.Context) error {
		var err error
		result, err = provider.Retrieve(ctx, ids)
		return err
	})

	return result, err
}

// Search performs vector search using the load balancing strategy
func (m *ProviderManager) Search(ctx context.Context, query *memory.VectorQuery) (*memory.VectorSearchResult, error) {
	start := time.Now()
	defer func() {
		m.updateOperationStats(time.Since(start), true)
	}()

	provider, err := m.selectProvider("search")
	if err != nil {
		m.updateOperationStats(time.Since(start), false)
		return nil, err
	}

	var result *memory.VectorSearchResult
	err = m.executeWithRetry(ctx, provider, func(ctx context.Context) error {
		var err error
		result, err = provider.Search(ctx, query)
		return err
	})

	return result, err
}

// FindSimilar finds similar vectors using the load balancing strategy
func (m *ProviderManager) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*memory.VectorSimilarityResult, error) {
	start := time.Now()
	defer func() {
		m.updateOperationStats(time.Since(start), true)
	}()

	provider, err := m.selectProvider("find_similar")
	if err != nil {
		m.updateOperationStats(time.Since(start), false)
		return nil, err
	}

	var result []*memory.VectorSimilarityResult
	err = m.executeWithRetry(ctx, provider, func(ctx context.Context) error {
		var err error
		result, err = provider.FindSimilar(ctx, embedding, k, filters)
		return err
	})

	return result, err
}

// CreateCollection creates a collection using the specified provider
func (m *ProviderManager) CreateCollection(ctx context.Context, providerName, collectionName string, config *memory.CollectionConfig) error {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return err
	}

	return m.executeWithRetry(ctx, provider, func(ctx context.Context) error {
		return provider.CreateCollection(ctx, collectionName, config)
	})
}

// DeleteCollection deletes a collection using the specified provider
func (m *ProviderManager) DeleteCollection(ctx context.Context, providerName, collectionName string) error {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return err
	}

	return m.executeWithRetry(ctx, provider, func(ctx context.Context) error {
		return provider.DeleteCollection(ctx, collectionName)
	})
}

// ListCollections lists collections from all active providers
func (m *ProviderManager) ListCollections(ctx context.Context) (map[string][]*memory.CollectionInfo, error) {
	result := make(map[string][]*memory.CollectionInfo)

	for name, provider := range m.providers {
		collections, err := provider.ListCollections(ctx)
		if err != nil {
			m.logger.Warn("Failed to list collections from provider",
				"provider", name,
				"error", err)
			continue
		}
		result[name] = collections
	}

	return result, nil
}

// GetCollection gets collection information from the specified provider
func (m *ProviderManager) GetCollection(ctx context.Context, providerName, collectionName string) (*memory.CollectionInfo, error) {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.GetCollection(ctx, collectionName)
}

// Health checks health of all providers
func (m *ProviderManager) Health(ctx context.Context) (map[string]*HealthStatus, error) {
	result := make(map[string]*HealthStatus)

	for name, provider := range m.providers {
		health, err := provider.Health(ctx)
		if err != nil {
			m.logger.Warn("Health check failed for provider",
				"provider", name,
				"error", err)
			result[name] = &HealthStatus{
				Status:       "unhealthy",
				LastCheck:    time.Now(),
				ResponseTime: 0,
			}
		} else {
			result[name] = health
		}

		// Update provider stats
		if stats, exists := m.stats.ProviderStats[name]; exists {
			stats.HealthCheckCount++
			if err == nil {
				stats.HealthCheckSuccesses++
			}
		}
	}

	m.stats.LastHealthCheck = time.Now()
	return result, nil
}

// GetStats returns manager statistics
func (m *ProviderManager) GetStats() *ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Update aggregate stats
	totalOps := int64(0)
	totalSuccesses := int64(0)
	totalFailures := int64(0)
	totalCost := 0.0
	totalLatency := time.Duration(0)
	latencyCount := 0

	for _, stats := range m.stats.ProviderStats {
		totalOps += stats.Operations
		totalSuccesses += stats.Successes
		totalFailures += stats.Failures
		totalCost += stats.Cost
		if stats.AverageLatency > 0 {
			totalLatency += stats.AverageLatency
			latencyCount++
		}
	}

	m.stats.TotalOperations = totalOps
	m.stats.SuccessfulOperations = totalSuccesses
	m.stats.FailedOperations = totalFailures
	m.stats.TotalCost = totalCost

	if latencyCount > 0 {
		m.stats.AverageLatency = totalLatency / time.Duration(latencyCount)
	}

	return m.stats
}

// Optimize optimizes all providers
func (m *ProviderManager) Optimize(ctx context.Context) error {
	var errors []error

	for name, provider := range m.providers {
		if err := provider.Optimize(ctx); err != nil {
			m.logger.Warn("Failed to optimize provider",
				"provider", name,
				"error", err)
			errors = append(errors, err)
		} else {
			m.logger.Info("Provider optimized", "name", name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("optimization failed for %d providers", len(errors))
	}
	return nil
}

// Backup performs backup on all providers that support it
func (m *ProviderManager) Backup(ctx context.Context, path string) error {
	var errors []error

	for name, provider := range m.providers {
		if err := provider.Backup(ctx, path+"/"+name); err != nil {
			m.logger.Warn("Failed to backup provider",
				"provider", name,
				"error", err)
			errors = append(errors, err)
		} else {
			m.logger.Info("Provider backed up", "name", name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("backup failed for %d providers", len(errors))
	}
	return nil
}

// selectProvider selects a provider based on the load balancing strategy
func (m *ProviderManager) selectProvider(operation string) (VectorProvider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.providers) == 0 {
		return nil, fmt.Errorf("no active providers available")
	}

	// Use default provider for now
	// TODO: Implement different load balancing strategies
	if defaultProvider, exists := m.providers[m.defaultProvider]; exists {
		return defaultProvider, nil
	}

	// Return any available provider
	for _, provider := range m.providers {
		return provider, nil
	}

	return nil, fmt.Errorf("no available providers")
}

// getProvider gets a specific provider by name
func (m *ProviderManager) getProvider(name string) (VectorProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// executeWithRetry executes an operation with retry logic
func (m *ProviderManager) executeWithRetry(ctx context.Context, provider VectorProvider, operation func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt <= m.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait with exponential backoff
			waitTime := time.Duration(attempt) * m.config.RetryBackoff
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := operation(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		m.logger.Warn("Operation failed, retrying",
			"attempt", attempt+1,
			"max_attempts", m.config.RetryAttempts+1,
			"error", err)

		// Check if error is retryable
		if !m.isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", m.config.RetryAttempts+1, lastErr)
}

// isRetryableError checks if an error is retryable
func (m *ProviderManager) isRetryableError(err error) bool {
	// TODO: Implement retryable error detection
	return true
}

// updateOperationStats updates operation statistics
func (m *ProviderManager) updateOperationStats(duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalOperations++
	m.stats.Uptime += duration

	if success {
		m.stats.SuccessfulOperations++
	} else {
		m.stats.FailedOperations++
	}

	// Update average latency
	if m.stats.AverageLatency == 0 {
		m.stats.AverageLatency = duration
	} else {
		m.stats.AverageLatency = (m.stats.AverageLatency + duration) / 2
	}
}

// Start starts the provider manager
func (m *ProviderManager) Start(ctx context.Context) error {
	// Start health monitoring if enabled
	if m.config.HealthCheckInterval > 0 {
		go m.healthMonitor(ctx)
	}

	// Start backup scheduler if enabled
	if m.config.BackupEnabled && m.config.BackupInterval > 0 {
		go m.backupScheduler(ctx)
	}

	return nil
}

// Stop stops the provider manager and all providers
func (m *ProviderManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error

	for name, provider := range m.providers {
		if err := provider.Stop(ctx); err != nil {
			m.logger.Error("Failed to stop provider",
				"name", name,
				"error", err)
			errors = append(errors, err)
		} else {
			m.logger.Info("Provider stopped", "name", name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop %d providers", len(errors))
	}

	return nil
}

// healthMonitor runs periodic health checks
func (m *ProviderManager) healthMonitor(ctx context.Context) {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			m.Health(healthCtx)
			cancel()
		}
	}
}

// backupScheduler runs periodic backups
func (m *ProviderManager) backupScheduler(ctx context.Context) {
	ticker := time.NewTicker(m.config.BackupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			backupPath := fmt.Sprintf("backup_%s", time.Now().Format("20060102_150405"))
			if err := m.Backup(ctx, backupPath); err != nil {
				m.logger.Error("Scheduled backup failed", "error", err)
			} else {
				m.logger.Info("Scheduled backup completed", "path", backupPath)
			}
		}
	}
}
