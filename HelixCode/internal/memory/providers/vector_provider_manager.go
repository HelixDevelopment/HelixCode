package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
)

// VectorProviderManager manages vector database providers with seamless switching
type VectorProviderManager struct {
	providers map[string]VectorProvider
	active    string
	config    *config.VectorProviderConfig
	logger    logging.Logger
	mu        sync.RWMutex

	// Health monitoring
	healthStatus map[string]*ProviderHealth
	healthTicker *time.Ticker

	// Load balancing
	loadBalancer LoadBalancer

	// Fallback mechanism
	fallbackManager *FallbackManager

	// Performance monitoring
	performanceMonitor *PerformanceMonitor
}

// VectorProvider defines interface for vector database providers
type VectorProvider interface {
	// Core operations
	Store(ctx context.Context, vectors []*VectorData) error
	Retrieve(ctx context.Context, ids []string) ([]*VectorData, error)
	Update(ctx context.Context, id string, vector *VectorData) error
	Delete(ctx context.Context, ids []string) error

	// Search operations
	Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error)
	FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error)
	BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error)

	// Collection management
	CreateCollection(ctx context.Context, name string, config *CollectionConfig) error
	DeleteCollection(ctx context.Context, name string) error
	ListCollections(ctx context.Context) ([]*CollectionInfo, error)
	GetCollection(ctx context.Context, name string) (*CollectionInfo, error)

	// Index management
	CreateIndex(ctx context.Context, collection string, config *IndexConfig) error
	DeleteIndex(ctx context.Context, collection, name string) error
	ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error)

	// Metadata operations
	AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error
	UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error
	GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error)
	DeleteMetadata(ctx context.Context, ids []string, keys []string) error

	// Management
	GetStats(ctx context.Context) (*ProviderStats, error)
	Optimize(ctx context.Context) error
	Backup(ctx context.Context, path string) error
	Restore(ctx context.Context, path string) error

	// Lifecycle
	Initialize(ctx context.Context, config interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) (*HealthStatus, error)

	// Metadata
	GetName() string
	GetType() VectorProviderType
	GetCapabilities() []string
	GetConfiguration() interface{}
	IsCloud() bool
	GetCostInfo() *CostInfo
}

// VectorData represents a vector entry
type VectorData struct {
	ID         string                 `json:"id"`
	Vector     []float64              `json:"vector"`
	Metadata   map[string]interface{} `json:"metadata"`
	Collection string                 `json:"collection"`
	Timestamp  time.Time              `json:"timestamp"`
	TTL        *time.Duration         `json:"ttl,omitempty"`
	Namespace  string                 `json:"namespace,omitempty"`
}

// VectorQuery represents a vector search query
type VectorQuery struct {
	Vector        []float64              `json:"vector"`
	Collection    string                 `json:"collection"`
	Namespace     string                 `json:"namespace,omitempty"`
	TopK          int                    `json:"top_k"`
	Threshold     float64                `json:"threshold,omitempty"`
	IncludeVector bool                   `json:"include_vector"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
}

// VectorSearchResult represents the result of a vector search
type VectorSearchResult struct {
	Results   []*VectorSearchResultItem `json:"results"`
	Total     int                       `json:"total"`
	Query     *VectorQuery              `json:"query"`
	Duration  time.Duration             `json:"duration"`
	Namespace string                    `json:"namespace"`
}

// VectorSearchResultItem represents a single search result
type VectorSearchResultItem struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float64                `json:"score"`
	Distance float64                `json:"distance"`
}

// VectorSimilarityResult represents similarity search result
type VectorSimilarityResult struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float64                `json:"score"`
	Distance float64                `json:"distance"`
}

// CollectionConfig represents collection configuration
type CollectionConfig struct {
	Name        string                 `json:"name"`
	Dimension   int                    `json:"dimension"`
	Metric      string                 `json:"metric"`
	Metadata    map[string]interface{} `json:"metadata"`
	Description string                 `json:"description"`
	Replicas    int                    `json:"replicas"`
	Shards      int                    `json:"shards"`
}

// CollectionInfo represents collection information
type CollectionInfo struct {
	Name        string                 `json:"name"`
	Dimension   int                    `json:"dimension"`
	Metric      string                 `json:"metric"`
	Size        int64                  `json:"size"`
	VectorCount int64                  `json:"vector_count"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// IndexConfig represents index configuration
type IndexConfig struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Fields   []string               `json:"fields"`
	Metadata map[string]interface{} `json:"metadata"`
	Options  map[string]interface{} `json:"options"`
}

// IndexInfo represents index information
type IndexInfo struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Fields    []string               `json:"fields"`
	State     string                 `json:"state"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// VectorProviderType represents the type of vector provider
type VectorProviderType string

const (
	VectorProviderTypeChromaDB VectorProviderType = "chromadb"
	VectorProviderTypePinecone VectorProviderType = "pinecone"
	VectorProviderTypeFAISS    VectorProviderType = "faiss"
	VectorProviderTypeQdrant   VectorProviderType = "qdrant"
	VectorProviderTypeMilvus   VectorProviderType = "milvus"
	VectorProviderTypeWeaviate VectorProviderType = "weaviate"
	VectorProviderTypeRedis    VectorProviderType = "redis"
)

// ProviderHealth represents provider health status
type ProviderHealth struct {
	Status       string        `json:"status"`
	LastCheck    time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorCount   int           `json:"error_count"`
	ErrorMessage string        `json:"error_message,omitempty"`
	IsHealthy    bool          `json:"is_healthy"`
	Uptime       time.Duration `json:"uptime"`
}

// LoadBalancer interface for load balancing strategies
type LoadBalancer interface {
	SelectProvider(providers []string, operation string) string
	UpdateProviderPerformance(provider string, latency time.Duration, success bool)
	GetProviderStats() map[string]*ProviderPerformanceStats
}

// FallbackManager manages fallback between providers
type FallbackManager interface {
	SetFallbackOrder(providers []string)
	ExecuteWithFallback(ctx context.Context, operation string, fn func(string) error) error
	GetFallbackHistory() []*FallbackAttempt
}

// PerformanceMonitor tracks provider performance
type PerformanceMonitor interface {
	RecordOperation(provider, operation string, duration time.Duration, success bool)
	GetProviderPerformance(provider string) *ProviderPerformanceStats
	GetAllProviderPerformance() map[string]*ProviderPerformanceStats
	SetAlertThresholds(thresholds map[string]float64)
}

// ProviderPerformanceStats represents provider performance statistics
type ProviderPerformanceStats struct {
	Provider         string             `json:"provider"`
	TotalOperations  int64              `json:"total_operations"`
	SuccessfulOps    int64              `json:"successful_operations"`
	FailedOps        int64              `json:"failed_operations"`
	AverageLatency   time.Duration      `json:"average_latency"`
	MinLatency       time.Duration      `json:"min_latency"`
	MaxLatency       time.Duration      `json:"max_latency"`
	LastOperation    time.Time          `json:"last_operation"`
	ErrorRate        float64            `json:"error_rate"`
	Throughput       float64            `json:"throughput"`
	CostPerOperation float64            `json:"cost_per_operation"`
	CustomMetrics    map[string]float64 `json:"custom_metrics"`
}

// CostInfo represents cost information for cloud providers
type CostInfo struct {
	StorageCost   float64 `json:"storage_cost"`
	ComputeCost   float64 `json:"compute_cost"`
	TransferCost  float64 `json:"transfer_cost"`
	TotalCost     float64 `json:"total_cost"`
	Currency      string  `json:"currency"`
	BillingPeriod string  `json:"billing_period"`
	FreeTierUsed  bool    `json:"free_tier_used"`
	FreeTierLimit float64 `json:"free_tier_limit"`
}

// FallbackAttempt represents a fallback operation attempt
type FallbackAttempt struct {
	Timestamp    time.Time     `json:"timestamp"`
	Operation    string        `json:"operation"`
	Providers    []string      `json:"providers"`
	SuccessIndex int           `json:"success_index"`
	Errors       []string      `json:"errors"`
	Duration     time.Duration `json:"duration"`
}

// NewVectorProviderManager creates a new vector provider manager
func NewVectorProviderManager(config *config.VectorProviderConfig, logger logging.Logger) *VectorProviderManager {
	if config == nil {
		config = config.DefaultVectorProviderConfig()
	}

	if logger == nil {
		logger = logging.NewLogger("vector_provider_manager")
	}

	return &VectorProviderManager{
		providers:          make(map[string]VectorProvider),
		config:             config,
		logger:             logger,
		healthStatus:       make(map[string]*ProviderHealth),
		loadBalancer:       NewRoundRobinLoadBalancer(),
		fallbackManager:    NewSequentialFallbackManager(),
		performanceMonitor: NewDefaultPerformanceMonitor(),
	}
}

// Initialize initializes the vector provider manager
func (vpm *VectorProviderManager) Initialize(ctx context.Context) error {
	vpm.mu.Lock()
	defer vpm.mu.Unlock()

	vpm.logger.Info("Initializing Vector Provider Manager...")

	// Initialize configured providers
	for name, providerConfig := range vpm.config.Providers {
		if err := vpm.initializeProvider(ctx, name, providerConfig); err != nil {
			vpm.logger.Error("Failed to initialize provider", "name", name, "error", err)
			return fmt.Errorf("failed to initialize provider %s: %w", name, err)
		}
	}

	// Set active provider
	if vpm.config.ActiveProvider != "" {
		if err := vpm.SetActiveProvider(ctx, vpm.config.ActiveProvider); err != nil {
			vpm.logger.Warn("Failed to set active provider", "provider", vpm.config.ActiveProvider, "error", err)
		}
	} else {
		// Select best available provider
		if err := vpm.selectBestProvider(ctx); err != nil {
			return fmt.Errorf("failed to select best provider: %w", err)
		}
	}

	// Start health monitoring
	if vpm.config.HealthMonitoring.Enabled {
		vpm.startHealthMonitoring(ctx)
	}

	// Setup fallback order
	if len(vpm.config.Fallback.Providers) > 0 {
		vpm.fallbackManager.SetFallbackOrder(vpm.config.Fallback.Providers)
	}

	vpm.logger.Info("Vector Provider Manager initialized successfully")
	return nil
}

// Store stores vectors using the active provider
func (vpm *VectorProviderManager) Store(ctx context.Context, vectors []*VectorData) error {
	provider, err := vpm.getProvider(ctx)
	if err != nil {
		return err
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		vpm.performanceMonitor.RecordOperation(provider.GetName(), "store", duration, err == nil)
	}()

	vpm.logger.Debug("Storing vectors", "count", len(vectors), "provider", provider.GetName())

	return provider.Store(ctx, vectors)
}

// Retrieve retrieves vectors using the active provider
func (vpm *VectorProviderManager) Retrieve(ctx context.Context, ids []string) ([]*VectorData, error) {
	provider, err := vpm.getProvider(ctx)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		vpm.performanceMonitor.RecordOperation(provider.GetName(), "retrieve", duration, err == nil)
	}()

	vpm.logger.Debug("Retrieving vectors", "count", len(ids), "provider", provider.GetName())

	return provider.Retrieve(ctx, ids)
}

// Search performs vector search using the active provider
func (vpm *VectorProviderManager) Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error) {
	provider, err := vpm.getProvider(ctx)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		vpm.performanceMonitor.RecordOperation(provider.GetName(), "search", duration, err == nil)
	}()

	vpm.logger.Debug("Searching vectors", "top_k", query.TopK, "provider", provider.GetName())

	return provider.Search(ctx, query)
}

// FindSimilar finds similar vectors using the active provider
func (vpm *VectorProviderManager) FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error) {
	provider, err := vpm.getProvider(ctx)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		vpm.performanceMonitor.RecordOperation(provider.GetName(), "find_similar", duration, err == nil)
	}()

	vpm.logger.Debug("Finding similar vectors", "k", k, "provider", provider.GetName())

	return provider.FindSimilar(ctx, embedding, k, filters)
}

// CreateCollection creates a collection using the active provider
func (vpm *VectorProviderManager) CreateCollection(ctx context.Context, name string, config *CollectionConfig) error {
	provider, err := vpm.getProvider(ctx)
	if err != nil {
		return err
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		vpm.performanceMonitor.RecordOperation(provider.GetName(), "create_collection", duration, err == nil)
	}()

	vpm.logger.Debug("Creating collection", "name", name, "provider", provider.GetName())

	return provider.CreateCollection(ctx, name, config)
}

// SwitchProvider switches to a different provider
func (vpm *VectorProviderManager) SwitchProvider(ctx context.Context, providerName string) error {
	vpm.mu.Lock()
	defer vpm.mu.Unlock()

	provider, exists := vpm.providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	// Check provider health
	health, err := provider.Health(ctx)
	if err != nil {
		return fmt.Errorf("provider %s is not healthy: %w", providerName, err)
	}

	if health.Status != "healthy" {
		return fmt.Errorf("provider %s is not healthy: %s", providerName, health.Status)
	}

	// Switch active provider
	oldProvider := vpm.active
	vpm.active = providerName

	vpm.logger.Info("Switched vector provider",
		"from", oldProvider,
		"to", providerName,
		"type", provider.GetType(),
		"is_cloud", provider.IsCloud())

	return nil
}

// SetActiveProvider sets the active provider
func (vpm *VectorProviderManager) SetActiveProvider(ctx context.Context, providerName string) error {
	return vpm.SwitchProvider(ctx, providerName)
}

// GetActiveProvider returns the active provider name
func (vpm *VectorProviderManager) GetActiveProvider() string {
	vpm.mu.RLock()
	defer vpm.mu.RUnlock()

	return vpm.active
}

// ListProviders returns a list of all providers
func (vpm *VectorProviderManager) ListProviders() map[string]*ProviderInfo {
	vpm.mu.RLock()
	defer vpm.mu.RUnlock()

	providers := make(map[string]*ProviderInfo)
	for name, provider := range vpm.providers {
		providers[name] = &ProviderInfo{
			Name:         provider.GetName(),
			Type:         string(provider.GetType()),
			Capabilities: provider.GetCapabilities(),
			IsCloud:      provider.IsCloud(),
			IsActive:     name == vpm.active,
			IsHealthy:    vpm.isProviderHealthy(name),
			CostInfo:     provider.GetCostInfo(),
		}
	}

	return providers
}

// GetProvider returns a specific provider
func (vpm *VectorProviderManager) GetProvider(name string) (VectorProvider, error) {
	vpm.mu.RLock()
	defer vpm.mu.RUnlock()

	provider, exists := vpm.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// GetProviderHealth returns health status of all providers
func (vpm *VectorProviderManager) GetProviderHealth(ctx context.Context) (map[string]*ProviderHealth, error) {
	vpm.mu.RLock()
	defer vpm.mu.RUnlock()

	// Update health status
	for name, provider := range vpm.providers {
		health, err := provider.Health(ctx)
		if err != nil {
			vpm.logger.Warn("Failed to get provider health", "name", name, "error", err)
			if vpm.healthStatus[name] == nil {
				vpm.healthStatus[name] = &ProviderHealth{}
			}
			vpm.healthStatus[name].Status = "unhealthy"
			vpm.healthStatus[name].ErrorMessage = err.Error()
			vpm.healthStatus[name].IsHealthy = false
			vpm.healthStatus[name].ErrorCount++
		} else {
			if vpm.healthStatus[name] == nil {
				vpm.healthStatus[name] = &ProviderHealth{}
			}

			vpm.healthStatus[name].Status = health.Status
			vpm.healthStatus[name].ResponseTime = health.ResponseTime
			vpm.healthStatus[name].LastCheck = health.LastCheck
			vpm.healthStatus[name].IsHealthy = health.Status == "healthy"
		}
	}

	// Return copy
	healthCopy := make(map[string]*ProviderHealth)
	for name, health := range vpm.healthStatus {
		healthCopy[name] = &ProviderHealth{
			Status:       health.Status,
			LastCheck:    health.LastCheck,
			ResponseTime: health.ResponseTime,
			ErrorCount:   health.ErrorCount,
			ErrorMessage: health.ErrorMessage,
			IsHealthy:    health.IsHealthy,
			Uptime:       health.Uptime,
		}
	}

	return healthCopy, nil
}

// GetProviderPerformance returns performance statistics
func (vpm *VectorProviderManager) GetProviderPerformance() map[string]*ProviderPerformanceStats {
	return vpm.performanceMonitor.GetAllProviderPerformance()
}

// OptimizeProviders optimizes all providers
func (vpm *VectorProviderManager) OptimizeProviders(ctx context.Context) error {
	vpm.mu.RLock()
	providers := make(map[string]VectorProvider)
	for name, provider := range vpm.providers {
		providers[name] = provider
	}
	vpm.mu.RUnlock()

	for name, provider := range providers {
		vpm.logger.Debug("Optimizing provider", "name", name)

		if err := provider.Optimize(ctx); err != nil {
			vpm.logger.Warn("Failed to optimize provider", "name", name, "error", err)
		}
	}

	return nil
}

// Shutdown shuts down the vector provider manager
func (vpm *VectorProviderManager) Shutdown(ctx context.Context) error {
	vpm.mu.Lock()
	defer vpm.mu.Unlock()

	// Stop health monitoring
	if vpm.healthTicker != nil {
		vpm.healthTicker.Stop()
	}

	// Shutdown all providers
	for name, provider := range vpm.providers {
		vpm.logger.Debug("Shutting down provider", "name", name)

		if err := provider.Stop(ctx); err != nil {
			vpm.logger.Warn("Failed to shutdown provider", "name", name, "error", err)
		}
	}

	vpm.providers = make(map[string]VectorProvider)
	vpm.active = ""

	vpm.logger.Info("Vector Provider Manager shut down successfully")
	return nil
}

// ProviderInfo represents information about a provider
type ProviderInfo struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Capabilities []string  `json:"capabilities"`
	IsCloud      bool      `json:"is_cloud"`
	IsActive     bool      `json:"is_active"`
	IsHealthy    bool      `json:"is_healthy"`
	CostInfo     *CostInfo `json:"cost_info,omitempty"`
}

// Private helper methods

func (vpm *VectorProviderManager) getProvider(ctx context.Context) (VectorProvider, error) {
	vpm.mu.RLock()
	defer vpm.mu.RUnlock()

	if vpm.active == "" {
		return nil, fmt.Errorf("no active vector provider configured")
	}

	provider, exists := vpm.providers[vpm.active]
	if !exists {
		return nil, fmt.Errorf("active provider %s not found", vpm.active)
	}

	return provider, nil
}

func (vpm *VectorProviderManager) initializeProvider(ctx context.Context, name string, config *config.ProviderConfig) error {
	// Create provider based on type
	var provider VectorProvider
	var err error

	switch config.Type {
	case "chromadb":
		provider, err = vpm.createChromaDBProvider(config)
	case "pinecone":
		provider, err = vpm.createPineconeProvider(config)
	case "faiss":
		provider, err = vpm.createFAISSProvider(config)
	case "qdrant":
		provider, err = vpm.createQdrantProvider(config)
	case "milvus":
		provider, err = vpm.createMilvusProvider(config)
	case "weaviate":
		provider, err = vpm.createWeaviateProvider(config)
	case "redis":
		provider, err = vpm.createRedisProvider(config)
	default:
		return fmt.Errorf("unsupported vector provider type: %s", config.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create provider %s: %w", name, err)
	}

	// Initialize provider
	if err := provider.Initialize(ctx, config.Configuration); err != nil {
		return fmt.Errorf("failed to initialize provider %s: %w", name, err)
	}

	// Start provider
	if err := provider.Start(ctx); err != nil {
		return fmt.Errorf("failed to start provider %s: %w", name, err)
	}

	// Add provider
	vpm.providers[name] = provider

	vpm.logger.Info("Vector provider initialized",
		"name", name,
		"type", config.Type,
		"capabilities", provider.GetCapabilities(),
		"is_cloud", provider.IsCloud())

	return nil
}

func (vpm *VectorProviderManager) selectBestProvider(ctx context.Context) error {
	// If no providers configured, return error
	if len(vpm.providers) == 0 {
		return fmt.Errorf("no vector providers configured")
	}

	// Prioritize non-cloud providers for local development
	var preferredProviders []string
	var cloudProviders []string

	for name, provider := range vpm.providers {
		if provider.IsCloud() {
			cloudProviders = append(cloudProviders, name)
		} else {
			preferredProviders = append(preferredProviders, name)
		}
	}

	// Try preferred providers first
	candidates := append(preferredProviders, cloudProviders...)

	for _, name := range candidates {
		provider := vpm.providers[name]

		// Check health
		health, err := provider.Health(ctx)
		if err != nil {
			continue
		}

		if health.Status == "healthy" {
			vpm.active = name
			vpm.logger.Info("Selected best provider", "name", name, "type", provider.GetType())
			return nil
		}
	}

	// If no healthy provider found, use first available
	if len(candidates) > 0 {
		vpm.active = candidates[0]
		vpm.logger.Warn("Selected first available provider (no healthy providers)", "name", vpm.active)
		return nil
	}

	return fmt.Errorf("no available vector providers")
}

func (vpm *VectorProviderManager) isProviderHealthy(name string) bool {
	health, exists := vpm.healthStatus[name]
	if !exists {
		return false
	}
	return health.IsHealthy
}

func (vpm *VectorProviderManager) startHealthMonitoring(ctx context.Context) {
	interval := vpm.config.HealthMonitoring.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}

	vpm.healthTicker = time.NewTicker(interval)

	go func() {
		defer vpm.healthTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-vpm.healthTicker.C:
				vpm.updateHealthStatus(ctx)
			}
		}
	}()
}

func (vpm *VectorProviderManager) updateHealthStatus(ctx context.Context) {
	vpm.mu.RLock()
	providers := make(map[string]VectorProvider)
	for name, provider := range vpm.providers {
		providers[name] = provider
	}
	vpm.mu.RUnlock()

	for name, provider := range providers {
		health, err := provider.Health(ctx)
		if err != nil {
			vpm.logger.Warn("Health check failed", "provider", name, "error", err)
			continue
		}

		if vpm.healthStatus[name] == nil {
			vpm.healthStatus[name] = &ProviderHealth{}
		}

		vpm.healthStatus[name].Status = health.Status
		vpm.healthStatus[name].ResponseTime = health.ResponseTime
		vpm.healthStatus[name].LastCheck = health.LastCheck
		vpm.healthStatus[name].IsHealthy = health.Status == "healthy"
	}
}

// Provider creation methods (to be implemented)

func (vpm *VectorProviderManager) createChromaDBProvider(config *config.ProviderConfig) (VectorProvider, error) {
	return nil, fmt.Errorf("ChromaDB provider not yet implemented")
}

func (vpm *VectorProviderManager) createPineconeProvider(config *config.ProviderConfig) (VectorProvider, error) {
	return NewPineconeProvider(config)
}

func (vpm *VectorProviderManager) createFAISSProvider(config *config.ProviderConfig) (VectorProvider, error) {
	return nil, fmt.Errorf("FAISS provider not yet implemented")
}

func (vpm *VectorProviderManager) createQdrantProvider(config *config.ProviderConfig) (VectorProvider, error) {
	return nil, fmt.Errorf("Qdrant provider not yet implemented")
}

func (vpm *VectorProviderManager) createMilvusProvider(config *config.ProviderConfig) (VectorProvider, error) {
	return nil, fmt.Errorf("Milvus provider not yet implemented")
}

func (vpm *VectorProviderManager) createWeaviateProvider(config *config.ProviderConfig) (VectorProvider, error) {
	return nil, fmt.Errorf("Weaviate provider not yet implemented")
}

func (vpm *VectorProviderManager) createRedisProvider(config *config.ProviderConfig) (VectorProvider, error) {
	return nil, fmt.Errorf("Redis provider not yet implemented")
}

// Load balancer and fallback manager implementations (simplified)

func NewRoundRobinLoadBalancer() LoadBalancer {
	return &roundRobinLoadBalancer{
		currentIndex: make(map[string]int),
	}
}

type roundRobinLoadBalancer struct {
	mu           sync.Mutex
	currentIndex map[string]int
}

func (r *roundRobinLoadBalancer) SelectProvider(providers []string, operation string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(providers) == 0 {
		return ""
	}

	index := r.currentIndex[operation]
	provider := providers[index%len(providers)]
	r.currentIndex[operation] = index + 1

	return provider
}

func (r *roundRobinLoadBalancer) UpdateProviderPerformance(provider string, latency time.Duration, success bool) {
	// Implementation for updating performance stats
}

func (r *roundRobinLoadBalancer) GetProviderStats() map[string]*ProviderPerformanceStats {
	return make(map[string]*ProviderPerformanceStats)
}

func NewSequentialFallbackManager() FallbackManager {
	return &sequentialFallbackManager{
		fallbackOrder: []string{},
		attempts:      []*FallbackAttempt{},
	}
}

type sequentialFallbackManager struct {
	mu            sync.Mutex
	fallbackOrder []string
	attempts      []*FallbackAttempt
}

func (s *sequentialFallbackManager) SetFallbackOrder(providers []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallbackOrder = providers
}

func (s *sequentialFallbackManager) ExecuteWithFallback(ctx context.Context, operation string, fn func(string) error) error {
	s.mu.Lock()
	providers := make([]string, len(s.fallbackOrder))
	copy(providers, s.fallbackOrder)
	s.mu.Unlock()

	var errors []string
	var attempt *FallbackAttempt

	for i, provider := range providers {
		err := fn(provider)
		if err == nil {
			if attempt != nil {
				attempt.SuccessIndex = i
				attempt.Duration = time.Since(attempt.Timestamp)
				s.recordAttempt(attempt)
			}
			return nil
		}

		errors = append(errors, fmt.Sprintf("%s: %v", provider, err))

		if attempt == nil {
			attempt = &FallbackAttempt{
				Timestamp: time.Now(),
				Operation: operation,
				Providers: providers,
				Errors:    errors,
			}
		}
	}

	if attempt != nil {
		attempt.Duration = time.Since(attempt.Timestamp)
		s.recordAttempt(attempt)
	}

	return fmt.Errorf("all fallback providers failed: %v", errors)
}

func (s *sequentialFallbackManager) GetFallbackHistory() []*FallbackAttempt {
	s.mu.Lock()
	defer s.mu.Unlock()

	attempts := make([]*FallbackAttempt, len(s.attempts))
	copy(attempts, s.attempts)
	return attempts
}

func (s *sequentialFallbackManager) recordAttempt(attempt *FallbackAttempt) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts = append(s.attempts, attempt)

	// Keep only last 100 attempts
	if len(s.attempts) > 100 {
		s.attempts = s.attempts[1:]
	}
}

func NewDefaultPerformanceMonitor() PerformanceMonitor {
	return &defaultPerformanceMonitor{
		stats: make(map[string]*ProviderPerformanceStats),
	}
}

type defaultPerformanceMonitor struct {
	mu     sync.Mutex
	stats  map[string]*ProviderPerformanceStats
	alerts map[string]float64
}

func (d *defaultPerformanceMonitor) RecordOperation(provider, operation string, duration time.Duration, success bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	stats, exists := d.stats[provider]
	if !exists {
		stats = &ProviderPerformanceStats{
			Provider:      provider,
			MinLatency:    duration,
			MaxLatency:    duration,
			CustomMetrics: make(map[string]float64),
		}
		d.stats[provider] = stats
	}

	stats.TotalOperations++
	if success {
		stats.SuccessfulOps++
	} else {
		stats.FailedOps++
	}

	// Update latency stats
	if duration < stats.MinLatency {
		stats.MinLatency = duration
	}
	if duration > stats.MaxLatency {
		stats.MaxLatency = duration
	}

	// Calculate average latency
	totalOps := float64(stats.TotalOperations)
	currentAvg := float64(stats.AverageLatency)
	newAvg := (currentAvg*(totalOps-1) + float64(duration)) / totalOps
	stats.AverageLatency = time.Duration(newAvg)

	stats.LastOperation = time.Now()

	// Calculate error rate
	if stats.TotalOperations > 0 {
		stats.ErrorRate = float64(stats.FailedOps) / float64(stats.TotalOperations)
	}

	// Calculate throughput (operations per second)
	if stats.LastOperation.Sub(stats.LastOperation.Add(-time.Hour)) > 0 {
		stats.Throughput = float64(stats.TotalOperations) / stats.LastOperation.Sub(stats.LastOperation.Add(-time.Hour)).Seconds()
	}
}

func (d *defaultPerformanceMonitor) GetProviderPerformance(provider string) *ProviderPerformanceStats {
	d.mu.Lock()
	defer d.mu.Unlock()

	stats, exists := d.stats[provider]
	if !exists {
		return nil
	}

	// Return copy
	return &ProviderPerformanceStats{
		Provider:         stats.Provider,
		TotalOperations:  stats.TotalOperations,
		SuccessfulOps:    stats.SuccessfulOps,
		FailedOps:        stats.FailedOps,
		AverageLatency:   stats.AverageLatency,
		MinLatency:       stats.MinLatency,
		MaxLatency:       stats.MaxLatency,
		LastOperation:    stats.LastOperation,
		ErrorRate:        stats.ErrorRate,
		Throughput:       stats.Throughput,
		CostPerOperation: stats.CostPerOperation,
		CustomMetrics:    make(map[string]float64),
	}
}

func (d *defaultPerformanceMonitor) GetAllProviderPerformance() map[string]*ProviderPerformanceStats {
	d.mu.Lock()
	defer d.mu.Unlock()

	result := make(map[string]*ProviderPerformanceStats)
	for provider, stats := range d.stats {
		result[provider] = d.GetProviderPerformance(provider)
	}

	return result
}

func (d *defaultPerformanceMonitor) SetAlertThresholds(thresholds map[string]float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.alerts = make(map[string]float64)
	for metric, threshold := range thresholds {
		d.alerts[metric] = threshold
	}
}
