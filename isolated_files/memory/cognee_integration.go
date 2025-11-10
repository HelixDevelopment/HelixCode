package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/cognee"
	"dev.helix.code/internal/hardware"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/provider"
)

// CogneeIntegration provides seamless integration between Cognee and memory management
type CogneeIntegration struct {
	// Core components
	memoryManager   *Manager
	cogneeManager  *cognee.CogneeManager
	providerManager *provider.ProviderManager
	aikeyManager    *config.APIKeyManager
	
	// Configuration
	config         *config.HelixConfig
	hwProfile      *hardware.Profile
	cogneeConfig   *config.CogneeConfig
	
	// State and synchronization
	mu             sync.RWMutex
	initialized     bool
	healthStatus    *HealthStatus
	
	// Optimization
	optimizer       *cognee.HostAwareOptimizer
	perfOptimizer   *cognee.ResearchBasedOptimizer
	
	// Monitoring and metrics
	logger          logging.Logger
	metrics         *CogneeMetrics
	performance     map[string]time.Duration
	
	// Context management
	contextManager  *ContextManager
	memoryStore     *MemoryStore
	
	// Provider-specific adapters
	providerAdapters map[string]*ProviderAdapter
}

// CogneeMetrics tracks Cognee integration metrics
type CogneeMetrics struct {
	TotalOperations       int64         `json:"total_operations"`
	SuccessfulOperations  int64         `json:"successful_operations"`
	FailedOperations      int64         `json:"failed_operations"`
	AverageLatency       time.Duration `json:"average_latency"`
	LastOperation        time.Time     `json:"last_operation"`
	MemoryEntries        int64         `json:"memory_entries"`
	ContextEntries       int64         `json:"context_entries"`
	ProviderUsage        map[string]int64 `json:"provider_usage"`
	ErrorTypes          map[string]int64 `json:"error_types"`
	ResourceUsage       *ResourceUsage `json:"resource_usage"`
	Performance         map[string]*ProviderPerformance `json:"performance"`
}

// ResourceUsage tracks resource utilization
type ResourceUsage struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    int64   `json:"memory_usage"`
	DiskUsage      int64   `json:"disk_usage"`
	NetworkUsage   int64   `json:"network_usage"`
	GPUUsage       float64 `json:"gpu_usage"`
	Timestamp      time.Time `json:"timestamp"`
}

// ProviderPerformance tracks per-provider performance metrics
type ProviderPerformance struct {
	ProviderName   string        `json:"provider_name"`
	TotalRequests  int64         `json:"total_requests"`
	SuccessfulRequests int64      `json:"successful_requests"`
	FailedRequests int64         `json:"failed_requests"`
	AverageLatency time.Duration `json:"average_latency"`
	LastRequest   time.Time     `json:"last_request"`
	ErrorRate     float64       `json:"error_rate"`
	Throughput    float64       `json:"throughput"`
	PerformanceScore float64     `json:"performance_score"`
}

// ContextManager manages conversation context across providers
type ContextManager struct {
	activeContexts map[string]*ConversationContext
	maxContextSize int
	maxContextAge  time.Duration
	mu              sync.RWMutex
	logger           logging.Logger
}

// ConversationContext represents the context of a conversation
type ConversationContext struct {
	ID              string                 `json:"id"`
	Provider        string                 `json:"provider"`
	Model           string                 `json:"model"`
	SessionID       string                 `json:"session_id"`
	ConversationID  string                 `json:"conversation_id"`
	Memory          map[string]interface{} `json:"memory"`
	Knowledge       []*KnowledgeItem       `json:"knowledge"`
	Documents       []*DocumentItem        `json:"documents"`
	LastUpdated     time.Time              `json:"last_updated"`
	Expiration      time.Time              `json:"expiration"`
}

// KnowledgeItem represents a piece of knowledge from Cognee
type KnowledgeItem struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Type         string                 `json:"type"`
	Confidence   float64               `json:"confidence"`
	Source       string                 `json:"source"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	AccessCount  int64                  `json:"access_count"`
}

// DocumentItem represents a document in the memory system
type DocumentItem struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Content      string                 `json:"content"`
	URL          string                 `json:"url"`
	Size         int64                  `json:"size"`
	Type         string                 `json:"type"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	ModifiedAt   time.Time              `json:"modified_at"`
	AccessCount  int64                  `json:"access_count"`
	Relevance    float64               `json:"relevance"`
}

// MemoryStore provides persistent storage for memory data
type MemoryStore struct {
	data           map[string]*MemoryEntry
	index          map[string][]string // Type -> IDs
	searchIndex    map[string][]string // Content -> IDs
	maxEntries     int64
	maxSize        int64
	currentSize    int64
	mu             sync.RWMutex
	logger         logging.Logger
}

// MemoryEntry represents an entry in the memory store
type MemoryEntry struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Content      string                 `json:"content"`
	Provider     string                 `json:"provider"`
	Model        string                 `json:"model"`
	SessionID    string                 `json:"session_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Expiration   *time.Time            `json:"expiration,omitempty"`
	AccessCount  int64                  `json:"access_count"`
	LastAccess   time.Time              `json:"last_access"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ProviderAdapter adapts provider-specific memory operations
type ProviderAdapter struct {
	Provider       string
	Model          string
	MaxTokens      int
	MaxContext     int
	SupportedTypes []string
	Capabilities   []string
}

// NewCogneeIntegration creates a new Cognee integration
func NewCogneeIntegration(
	memoryManager *Manager,
	providerManager *provider.ProviderManager,
	aikeyManager *config.APIKeyManager,
) *CogneeIntegration {
	logger := logging.NewLogger("cognee_integration")
	
	return &CogneeIntegration{
		memoryManager:    memoryManager,
		providerManager:  providerManager,
		aikeyManager:     aikeyManager,
		logger:           logger,
		performance:      make(map[string]time.Duration),
		providerAdapters: make(map[string]*ProviderAdapter),
		contextManager: &ContextManager{
			activeContexts: make(map[string]*ConversationContext),
			maxContextSize: 1000,
			maxContextAge:  24 * time.Hour,
			logger:         logging.NewLogger("context_manager"),
		},
		memoryStore: &MemoryStore{
			data:        make(map[string]*MemoryEntry),
			index:       make(map[string][]string),
			searchIndex: make(map[string][]string),
			maxEntries:  1000000,
			maxSize:     1024 * 1024 * 1024, // 1GB
			logger:      logging.NewLogger("memory_store"),
		},
		metrics: &CogneeMetrics{
			ProviderUsage:  make(map[string]int64),
			ErrorTypes:     make(map[string]int64),
			Performance:    make(map[string]*ProviderPerformance),
		},
	}
}

// Initialize initializes the Cognee integration
func (ci *CogneeIntegration) Initialize(ctx context.Context, config *config.HelixConfig) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	if ci.initialized {
		return nil
	}
	
	ci.logger.Info("Initializing Cognee Integration...")
	
	// Set configuration
	ci.config = config
	ci.cogneeConfig = config.Cognee
	
	// Create hardware profile
	var err error
	ci.hwProfile, err = hardware.GetProfile()
	if err != nil {
		ci.logger.Warn("Failed to get hardware profile", "error", err)
		// Create default profile
		ci.hwProfile = hardware.DefaultProfile()
	}
	
	// Initialize Cognee manager
	ci.cogneeManager, err = cognee.NewCogneeManager(ci.cogneeConfig, ci.hwProfile)
	if err != nil {
		return fmt.Errorf("failed to create Cognee manager: %w", err)
	}
	
	// Initialize Cognee
	if err := ci.cogneeManager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize Cognee manager: %w", err)
	}
	
	// Create optimizers
	ci.optimizer = cognee.NewHostAwareOptimizer(ci.hwProfile)
	ci.perfOptimizer = cognee.NewResearchBasedOptimizer(ci.cogneeConfig)
	
	// Initialize optimizers
	if err := ci.optimizer.Initialize(ctx); err != nil {
		ci.logger.Warn("Failed to initialize host optimizer", "error", err)
	}
	
	if err := ci.perfOptimizer.Initialize(ctx); err != nil {
		ci.logger.Warn("Failed to initialize performance optimizer", "error", err)
	}
	
	// Create provider adapters
	if err := ci.createProviderAdapters(); err != nil {
		return fmt.Errorf("failed to create provider adapters: %w", err)
	}
	
	// Start background tasks
	if err := ci.startBackgroundTasks(ctx); err != nil {
		return fmt.Errorf("failed to start background tasks: %w", err)
	}
	
	// Update health status
	ci.healthStatus = &HealthStatus{
		Status:       "healthy",
		LastCheck:    time.Now(),
		ResponseTime:  0,
		Metrics:       make(map[string]float64),
		Dependencies:  make(map[string]string),
	}
	
	ci.initialized = true
	ci.logger.Info("Cognee Integration initialized successfully")
	
	return nil
}

// StoreMemory stores memory data using Cognee
func (ci *CogneeIntegration) StoreMemory(ctx context.Context, data *MemoryData) error {
	if !ci.initialized {
		return fmt.Errorf("cognee integration not initialized")
	}
	
	start := time.Now()
	defer func() {
		ci.metrics.TotalOperations++
		ci.metrics.LastOperation = time.Now()
		
		if err := recover(); err != nil {
			ci.metrics.FailedOperations++
			ci.logger.Error("Memory store panic", "error", err)
		}
		
		latency := time.Since(start)
		ci.performance["store"] = latency
		ci.metrics.AverageLatency = time.Duration(
			(ci.metrics.AverageLatency + latency) / 2,
		)
	}()
	
	ci.logger.Debug("Storing memory data", "id", data.ID, "type", data.Type)
	
	// Determine provider and model from context
	provider, model := ci.getProviderFromContext(ctx)
	
	// Store in Cognee
	if err := ci.storeInCognee(ctx, data, provider, model); err != nil {
		ci.metrics.FailedOperations++
		ci.metrics.ErrorTypes["cognee_store"]++
		return fmt.Errorf("failed to store in Cognee: %w", err)
	}
	
	// Store in memory store
	if err := ci.storeInMemoryStore(data, provider, model); err != nil {
		ci.metrics.ErrorTypes["memory_store"]++
		ci.logger.Warn("Failed to store in memory store", "error", err)
	}
	
	// Update context
	if err := ci.updateContext(ctx, data, provider, model); err != nil {
		ci.logger.Warn("Failed to update context", "error", err)
	}
	
	// Update metrics
	ci.metrics.MemoryEntries++
	if provider != "" {
		ci.metrics.ProviderUsage[provider]++
	}
	
	ci.metrics.SuccessfulOperations++
	ci.logger.Debug("Memory data stored successfully", "id", data.ID)
	
	return nil
}

// RetrieveMemory retrieves memory data using Cognee
func (ci *CogneeIntegration) RetrieveMemory(ctx context.Context, query *MemoryQuery) (*MemoryResult, error) {
	if !ci.initialized {
		return nil, fmt.Errorf("cognee integration not initialized")
	}
	
	start := time.Now()
	defer func() {
		ci.metrics.TotalOperations++
		ci.metrics.LastOperation = time.Now()
		
		if err := recover(); err != nil {
			ci.metrics.FailedOperations++
			ci.logger.Error("Memory retrieve panic", "error", err)
		}
		
		latency := time.Since(start)
		ci.performance["retrieve"] = latency
		ci.metrics.AverageLatency = time.Duration(
			(ci.metrics.AverageLatency + latency) / 2,
		)
	}()
	
	ci.logger.Debug("Retrieving memory data", "query", query)
	
	// Determine provider and model from context
	provider, model := ci.getProviderFromContext(ctx)
	
	// Optimize query
	optimizedQuery, err := ci.optimizeQuery(ctx, query, provider, model)
	if err != nil {
		ci.logger.Warn("Failed to optimize query", "error", err)
		optimizedQuery = query
	}
	
	// Retrieve from Cognee
	cogneeResult, err := ci.retrieveFromCognee(ctx, optimizedQuery, provider, model)
	if err != nil {
		ci.metrics.FailedOperations++
		ci.metrics.ErrorTypes["cognee_retrieve"]++
		return nil, fmt.Errorf("failed to retrieve from Cognee: %w", err)
	}
	
	// Retrieve from memory store
	memoryResult, err := ci.retrieveFromMemoryStore(optimizedQuery, provider, model)
	if err != nil {
		ci.logger.Warn("Failed to retrieve from memory store", "error", err)
		memoryResult = &MemoryResult{Data: []*MemoryData{}}
	}
	
	// Merge results
	mergedResult := ci.mergeResults(cogneeResult, memoryResult, optimizedQuery)
	
	ci.metrics.SuccessfulOperations++
	ci.logger.Debug("Memory data retrieved successfully", "count", len(mergedResult.Data))
	
	return mergedResult, nil
}

// SearchMemory searches memory using Cognee
func (ci *CogneeIntegration) SearchMemory(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
	if !ci.initialized {
		return nil, fmt.Errorf("cognee integration not initialized")
	}
	
	start := time.Now()
	defer func() {
		ci.metrics.TotalOperations++
		ci.metrics.LastOperation = time.Now()
		
		if err := recover(); err != nil {
			ci.metrics.FailedOperations++
			ci.logger.Error("Memory search panic", "error", err)
		}
		
		latency := time.Since(start)
		ci.performance["search"] = latency
		ci.metrics.AverageLatency = time.Duration(
			(ci.metrics.AverageLatency + latency) / 2,
		)
	}()
	
	ci.logger.Debug("Searching memory", "query", query.Query)
	
	// Determine provider and model from context
	provider, model := ci.getProviderFromContext(ctx)
	
	// Search in Cognee
	cogneeResult, err := ci.searchInCognee(ctx, query, provider, model)
	if err != nil {
		ci.metrics.FailedOperations++
		ci.metrics.ErrorTypes["cognee_search"]++
		return nil, fmt.Errorf("failed to search in Cognee: %w", err)
	}
	
	// Search in memory store
	memoryResult, err := ci.searchInMemoryStore(query, provider, model)
	if err != nil {
		ci.logger.Warn("Failed to search in memory store", "error", err)
		memoryResult = &SearchResult{Results: []*SearchResultItem{}}
	}
	
	// Merge results
	mergedResult := ci.mergeSearchResults(cogneeResult, memoryResult, query)
	
	ci.metrics.SuccessfulOperations++
	ci.logger.Debug("Memory search completed successfully", "count", len(mergedResult.Results))
	
	return mergedResult, nil
}

// GetContext gets conversation context for a provider
func (ci *CogneeIntegration) GetContext(ctx context.Context, provider, model, sessionID string) (*ConversationContext, error) {
	if !ci.initialized {
		return nil, fmt.Errorf("cognee integration not initialized")
	}
	
	return ci.contextManager.GetContext(provider, model, sessionID)
}

// UpdateContext updates conversation context
func (ci *CogneeIntegration) UpdateContext(ctx context.Context, context *ConversationContext) error {
	if !ci.initialized {
		return fmt.Errorf("cognee integration not initialized")
	}
	
	return ci.contextManager.UpdateContext(context)
}

// GetProviderMemory gets memory for a specific provider
func (ci *CogneeIntegration) GetProviderMemory(ctx context.Context, provider string) ([]*MemoryData, error) {
	if !ci.initialized {
		return nil, fmt.Errorf("cognee integration not initialized")
	}
	
	query := &MemoryQuery{
		Sources: []string{provider},
		Limit:   1000,
		SortBy:  "timestamp",
		SortOrder: "desc",
	}
	
	result, err := ci.RetrieveMemory(ctx, query)
	if err != nil {
		return nil, err
	}
	
	return result.Data, nil
}

// GetMetrics returns integration metrics
func (ci *CogneeIntegration) GetMetrics() *CogneeMetrics {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	
	// Update resource usage
	ci.updateResourceUsage()
	
	// Return copy
	metricsCopy := *ci.metrics
	if metricsCopy.ProviderUsage == nil {
		metricsCopy.ProviderUsage = make(map[string]int64)
	}
	if metricsCopy.ErrorTypes == nil {
		metricsCopy.ErrorTypes = make(map[string]int64)
	}
	if metricsCopy.Performance == nil {
		metricsCopy.Performance = make(map[string]*ProviderPerformance)
	}
	
	return &metricsCopy
}

// GetHealth returns health status
func (ci *CogneeIntegration) GetHealth(ctx context.Context) *HealthStatus {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	// Update health status
	ci.healthStatus.LastCheck = time.Now()
	ci.healthStatus.Metrics = ci.calculateHealthMetrics()
	
	// Check dependencies
	ci.healthStatus.Dependencies = ci.checkDependencies(ctx)
	
	// Determine overall status
	if ci.metrics.FailedOperations > ci.metrics.SuccessfulOperations {
		ci.healthStatus.Status = "unhealthy"
	} else if ci.metrics.FailedOperations > 0 {
		ci.healthStatus.Status = "degraded"
	} else {
		ci.healthStatus.Status = "healthy"
	}
	
	return ci.healthStatus
}

// Optimize optimizes the integration for better performance
func (ci *CogneeIntegration) Optimize(ctx context.Context) error {
	if !ci.initialized {
		return fmt.Errorf("cognee integration not initialized")
	}
	
	ci.logger.Info("Optimizing Cognee integration...")
	
	// Optimize Cognee
	if ci.cogneeManager != nil {
		if err := ci.cogneeManager.Optimize(ctx); err != nil {
			ci.logger.Warn("Failed to optimize Cognee", "error", err)
		}
	}
	
	// Optimize memory store
	if err := ci.optimizeMemoryStore(); err != nil {
		ci.logger.Warn("Failed to optimize memory store", "error", err)
	}
	
	// Optimize context manager
	if err := ci.contextManager.Optimize(); err != nil {
		ci.logger.Warn("Failed to optimize context manager", "error", err)
	}
	
	ci.logger.Info("Cognee integration optimization completed")
	return nil
}

// Shutdown shuts down the Cognee integration
func (ci *CogneeIntegration) Shutdown(ctx context.Context) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	if !ci.initialized {
		return nil
	}
	
	ci.logger.Info("Shutting down Cognee Integration...")
	
	// Stop background tasks
	if err := ci.stopBackgroundTasks(ctx); err != nil {
		ci.logger.Warn("Failed to stop background tasks", "error", err)
	}
	
	// Shutdown Cognee
	if ci.cogneeManager != nil {
		if err := ci.cogneeManager.Shutdown(ctx); err != nil {
			ci.logger.Warn("Failed to shutdown Cognee manager", "error", err)
		}
	}
	
	// Clear data
	ci.providerAdapters = make(map[string]*ProviderAdapter)
	ci.performance = make(map[string]time.Duration)
	
	ci.initialized = false
	ci.logger.Info("Cognee Integration shut down successfully")
	
	return nil
}

// Private helper methods

func (ci *CogneeIntegration) createProviderAdapters() error {
	// Create adapters for all supported providers
	adapters := map[string]*ProviderAdapter{
		"openai": {
			Provider:      "openai",
			MaxTokens:    4096,
			MaxContext:   8192,
			SupportedTypes: []string{"conversation", "knowledge", "document"},
			Capabilities:  []string{"embedding", "completion", "memory"},
		},
		"anthropic": {
			Provider:      "anthropic",
			MaxTokens:    4096,
			MaxContext:   100000,
			SupportedTypes: []string{"conversation", "knowledge"},
			Capabilities:  []string{"completion", "memory"},
		},
		"google": {
			Provider:      "google",
			MaxTokens:    4096,
			MaxContext:   32768,
			SupportedTypes: []string{"conversation", "knowledge", "document"},
			Capabilities:  []string{"embedding", "completion", "memory"},
		},
		"cohere": {
			Provider:      "cohere",
			MaxTokens:    4096,
			MaxContext:   8192,
			SupportedTypes: []string{"conversation", "knowledge"},
			Capabilities:  []string{"embedding", "completion", "memory"},
		},
		"replicate": {
			Provider:      "replicate",
			MaxTokens:    2048,
			MaxContext:   4096,
			SupportedTypes: []string{"conversation", "image", "document"},
			Capabilities:  []string{"completion", "image", "memory"},
		},
		"huggingface": {
			Provider:      "huggingface",
			MaxTokens:    4096,
			MaxContext:   8192,
			SupportedTypes: []string{"conversation", "knowledge", "document"},
			Capabilities:  []string{"embedding", "completion", "memory"},
		},
		"vllm": {
			Provider:      "vllm",
			MaxTokens:    4096,
			MaxContext:   8192,
			SupportedTypes: []string{"conversation", "knowledge", "document"},
			Capabilities:  []string{"embedding", "completion", "memory"},
		},
	}
	
	ci.providerAdapters = adapters
	ci.logger.Info("Created provider adapters", "count", len(adapters))
	
	return nil
}

func (ci *CogneeIntegration) getProviderFromContext(ctx context.Context) (string, string) {
	// Extract provider and model from context
	if provider := ctx.Value("provider"); provider != nil {
		if providerStr, ok := provider.(string); ok {
			if model := ctx.Value("model"); model != nil {
				if modelStr, ok := model.(string); ok {
					return providerStr, modelStr
				}
			}
			return providerStr, ""
		}
	}
	
	return "", ""
}

func (ci *CogneeIntegration) storeInCognee(ctx context.Context, data *MemoryData, provider, model string) error {
	// Convert memory data to Cognee format
	cogneeData := &cognee.MemoryItem{
		ID:          data.ID,
		Content:     data.Content,
		Type:        string(data.Type),
		Provider:    provider,
		Model:       model,
		Metadata:    data.Metadata,
		Timestamp:   data.Timestamp,
		Tags:        data.Tags,
		Source:      data.Source,
		Importance:  data.Importance,
	}
	
	// Store in Cognee
	if err := ci.cogneeManager.StoreMemory(ctx, cogneeData); err != nil {
		return fmt.Errorf("Cognee store failed: %w", err)
	}
	
	return nil
}

func (ci *CogneeIntegration) retrieveFromCognee(ctx context.Context, query *MemoryQuery, provider, model string) (*MemoryResult, error) {
	// Convert query to Cognee format
	cogneeQuery := &cognee.Query{
		Types:       convertMemoryTypes(query.Types),
		Providers:   []string{provider},
		Models:      []string{model},
		TimeRange:    query.TimeRange,
		Limit:        query.Limit,
		Offset:       query.Offset,
		SortBy:       query.SortBy,
		SortOrder:    query.SortOrder,
		Metadata:     query.Metadata,
	}
	
	// Retrieve from Cognee
	cogneeResult, err := ci.cogneeManager.RetrieveMemory(ctx, cogneeQuery)
	if err != nil {
		return nil, fmt.Errorf("Cognee retrieve failed: %w", err)
	}
	
	// Convert result back to memory format
	result := &MemoryResult{
		Data:    convertCognneeToMemory(cogneeResult.Data),
		Total:   cogneeResult.Total,
		HasMore: cogneeResult.HasMore,
		Query:   query,
		Duration: cogneeResult.Duration,
		Metadata: cogneeResult.Metadata,
	}
	
	return result, nil
}

func (ci *CogneeIntegration) searchInCognee(ctx context.Context, query *SearchQuery, provider, model string) (*SearchResult, error) {
	// Convert query to Cognee format
	cogneeQuery := &cognee.SearchQuery{
		Query:     query.Query,
		Types:     convertMemoryTypes(convertSearchTypes(query.Types)),
		Threshold: query.Threshold,
		K:         query.K,
		Providers: []string{provider},
		Models:    []string{model},
		TimeRange: query.TimeRange,
		Metadata:  query.Metadata,
		IncludeText: query.IncludeText,
	}
	
	// Search in Cognee
	cogneeResult, err := ci.cogneeManager.SearchMemory(ctx, cogneeQuery)
	if err != nil {
		return nil, fmt.Errorf("Cognee search failed: %w", err)
	}
	
	// Convert result back to search format
	result := &SearchResult{
		Results:    convertCognneeToSearch(cogneeResult.Results),
		Total:      cogneeResult.Total,
		Query:      query,
		Duration:   cogneeResult.Duration,
		Confidence: cogneeResult.Confidence,
		Metadata:   cogneeResult.Metadata,
	}
	
	return result, nil
}

func (ci *CogneeIntegration) storeInMemoryStore(data *MemoryData, provider, model string) error {
	entry := &MemoryEntry{
		ID:          data.ID,
		Type:        string(data.Type),
		Content:     data.Content,
		Provider:    provider,
		Model:       model,
		SessionID:   getSessionIDFromData(data),
		Timestamp:   data.Timestamp,
		AccessCount: 0,
		LastAccess:  time.Now(),
		Tags:        data.Tags,
		Metadata:    data.Metadata,
	}
	
	return ci.memoryStore.Store(entry)
}

func (ci *CogneeIntegration) retrieveFromMemoryStore(query *MemoryQuery, provider, model string) (*MemoryResult, error) {
	entries, err := ci.memoryStore.Retrieve(query, provider, model)
	if err != nil {
		return nil, err
	}
	
	return &MemoryResult{
		Data: convertEntriesToMemory(entries),
	}, nil
}

func (ci *CogneeIntegration) searchInMemoryStore(query *SearchQuery, provider, model string) (*SearchResult, error) {
	entries, err := ci.memoryStore.Search(query, provider, model)
	if err != nil {
		return nil, err
	}
	
	return &SearchResult{
		Results: convertEntriesToSearch(entries),
	}, nil
}

func (ci *CogneeIntegration) mergeResults(cognee, memory *MemoryResult, query *MemoryQuery) *MemoryResult {
	// Simple merge - combine results and remove duplicates
	seenIDs := make(map[string]bool)
	merged := make([]*MemoryData, 0)
	
	// Add Cognee results
	for _, data := range cognee.Data {
		if !seenIDs[data.ID] {
			merged = append(merged, data)
			seenIDs[data.ID] = true
		}
	}
	
	// Add memory store results
	for _, data := range memory.Data {
		if !seenIDs[data.ID] {
			merged = append(merged, data)
			seenIDs[data.ID] = true
		}
	}
	
	return &MemoryResult{
		Data:     merged,
		Total:    len(merged),
		Query:    query,
		Metadata: map[string]interface{}{
			"cognee_count":   len(cognee.Data),
			"memory_count":   len(memory.Data),
			"merged_count":   len(merged),
			"sources":       []string{"cognee", "memory_store"},
		},
	}
}

func (ci *CogneeIntegration) mergeSearchResults(cognee, memory *SearchResult, query *SearchQuery) *SearchResult {
	// Simple merge - combine results and remove duplicates
	seenIDs := make(map[string]bool)
	merged := make([]*SearchResultItem, 0)
	
	// Add Cognee results
	for _, item := range cognee.Results {
		if !seenIDs[item.Data.ID] {
			merged = append(merged, item)
			seenIDs[item.Data.ID] = true
		}
	}
	
	// Add memory store results
	for _, item := range memory.Results {
		if !seenIDs[item.Data.ID] {
			merged = append(merged, item)
			seenIDs[item.Data.ID] = true
		}
	}
	
	return &SearchResult{
		Results:    merged,
		Total:      len(merged),
		Query:      query,
		Metadata: map[string]interface{}{
			"cognee_count":   len(cognee.Results),
			"memory_count":   len(memory.Results),
			"merged_count":   len(merged),
			"sources":       []string{"cognee", "memory_store"},
		},
	}
}

func (ci *CogneeIntegration) updateContext(ctx context.Context, data *MemoryData, provider, model string) error {
	sessionID := getSessionIDFromData(data)
	
	// Get or create context
	context, err := ci.contextManager.GetContext(provider, model, sessionID)
	if err != nil {
		// Create new context
		context = &ConversationContext{
			ID:             generateContextID(provider, model, sessionID),
			Provider:       provider,
			Model:          model,
			SessionID:      sessionID,
			Memory:         make(map[string]interface{}),
			Knowledge:      make([]*KnowledgeItem, 0),
			Documents:      make([]*DocumentItem, 0),
			LastUpdated:    time.Now(),
			Expiration:     time.Now().Add(24 * time.Hour),
		}
	}
	
	// Update context with new data
	switch data.Type {
	case MemoryTypeConversation:
		context.Memory["last_conversation"] = data.Content
	case MemoryTypeKnowledge:
		knowledge := &KnowledgeItem{
			ID:         data.ID,
			Content:    data.Content,
			Type:       "cognee_knowledge",
			Source:     data.Source,
			Tags:       data.Tags,
			Metadata:   data.Metadata,
			CreatedAt:  data.Timestamp,
			AccessCount: 0,
		}
		context.Knowledge = append(context.Knowledge, knowledge)
	case MemoryTypeDocument:
		document := &DocumentItem{
			ID:         data.ID,
			Title:      getTitleFromData(data),
			Content:    data.Content,
			Tags:       data.Tags,
			Metadata:   data.Metadata,
			CreatedAt:  data.Timestamp,
			ModifiedAt: data.Timestamp,
			AccessCount: 0,
			Relevance:  calculateRelevance(data),
		}
		context.Documents = append(context.Documents, document)
	}
	
	context.LastUpdated = time.Now()
	
	return ci.contextManager.UpdateContext(context)
}

func (ci *CogneeIntegration) optimizeQuery(ctx context.Context, query *MemoryQuery, provider, model string) (*MemoryQuery, error) {
	// Apply performance optimization
	if ci.perfOptimizer != nil {
		optimized, err := ci.perfOptimizer.OptimizeQuery(ctx, query, provider, model)
		if err != nil {
			ci.logger.Warn("Performance optimizer failed", "error", err)
		} else {
			return optimized, nil
		}
	}
	
	return query, nil
}

func (ci *CogneeIntegration) startBackgroundTasks(ctx context.Context) error {
	// Start context cleanup
	go ci.contextCleanupTask(ctx)
	
	// Start metrics collection
	go ci.metricsCollectionTask(ctx)
	
	// Start health monitoring
	go ci.healthMonitoringTask(ctx)
	
	return nil
}

func (ci *CogneeIntegration) stopBackgroundTasks(ctx context.Context) error {
	// Background tasks will stop when context is cancelled
	return nil
}

func (ci *CogneeIntegration) contextCleanupTask(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ci.contextManager.Cleanup(); err != nil {
				ci.logger.Warn("Context cleanup failed", "error", err)
			}
		}
	}
}

func (ci *CogneeIntegration) metricsCollectionTask(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ci.updateResourceUsage()
		}
	}
}

func (ci *CogneeIntegration) healthMonitoringTask(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ci.GetHealth(ctx)
		}
	}
}

func (ci *CogneeIntegration) updateResourceUsage() {
	// Get resource usage from hardware profile
	if ci.hwProfile != nil {
		cpuUsage := ci.hwProfile.GetCPUUsage()
		memoryUsage := ci.hwProfile.GetMemoryUsage()
		gpuUsage := ci.hwProfile.GetGPUUsage()
		
		ci.metrics.ResourceUsage = &ResourceUsage{
			CPUUsage:    cpuUsage,
			MemoryUsage:  memoryUsage,
			GPUUsage:     gpuUsage,
			Timestamp:    time.Now(),
		}
	}
}

func (ci *CogneeIntegration) calculateHealthMetrics() map[string]float64 {
	metrics := make(map[string]float64)
	
	if ci.metrics.TotalOperations > 0 {
		metrics["success_rate"] = float64(ci.metrics.SuccessfulOperations) / float64(ci.metrics.TotalOperations)
		metrics["error_rate"] = float64(ci.metrics.FailedOperations) / float64(ci.metrics.TotalOperations)
	}
	
	metrics["memory_entries"] = float64(ci.metrics.MemoryEntries)
	metrics["provider_count"] = float64(len(ci.providerAdapters))
	
	return metrics
}

func (ci *CogneeIntegration) checkDependencies(ctx context.Context) map[string]string {
	dependencies := make(map[string]string)
	
	// Check Cognee
	if ci.cogneeManager != nil {
		health := ci.cogneeManager.GetHealth(ctx)
		if health != nil {
			dependencies["cognee"] = health.Status
		} else {
			dependencies["cognee"] = "unknown"
		}
	}
	
	// Check provider manager
	if ci.providerManager != nil {
		if ci.providerManager.IsHealthy() {
			dependencies["provider_manager"] = "healthy"
		} else {
			dependencies["provider_manager"] = "unhealthy"
		}
	}
	
	return dependencies
}

// Utility functions

func convertMemoryTypes(types []MemoryType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func convertSearchTypes(types []MemoryType) []string {
	return convertMemoryTypes(types)
}

func convertCognneeToMemory(cogneeData []*cognee.MemoryItem) []*MemoryData {
	result := make([]*MemoryData, len(cogneeData))
	for i, item := range cogneeData {
		result[i] = &MemoryData{
			ID:         item.ID,
			Content:    item.Content,
			Type:       MemoryType(item.Type),
			Metadata:   item.Metadata,
			Timestamp:  item.Timestamp,
			Tags:       item.Tags,
			Source:     item.Source,
			Importance: item.Importance,
		}
	}
	return result
}

func convertCognneeToSearch(cogneeResults []*cognee.SearchResultItem) []*SearchResultItem {
	result := make([]*SearchResultItem, len(cogneeResults))
	for i, item := range cogneeResults {
		result[i] = &SearchResultItem{
			Data:        &MemoryData{
				ID:        item.Data.ID,
				Content:   item.Data.Content,
				Type:      MemoryType(item.Data.Type),
				Metadata:  item.Data.Metadata,
				Timestamp: item.Data.Timestamp,
				Tags:      item.Data.Tags,
				Source:    item.Data.Source,
			},
			Score:       item.Score,
			Distance:    item.Distance,
			Explanation: item.Explanation,
		}
	}
	return result
}

func convertEntriesToMemory(entries []*MemoryEntry) []*MemoryData {
	result := make([]*MemoryData, len(entries))
	for i, entry := range entries {
		result[i] = &MemoryData{
			ID:        entry.ID,
			Content:   entry.Content,
			Type:      MemoryType(entry.Type),
			Metadata:  entry.Metadata,
			Timestamp: entry.Timestamp,
			Tags:      entry.Tags,
			Source:    entry.Provider,
		}
	}
	return result
}

func convertEntriesToSearch(entries []*MemoryEntry) []*SearchResultItem {
	result := make([]*SearchResultItem, len(entries))
	for i, entry := range entries {
		result[i] = &SearchResultItem{
			Data: &MemoryData{
				ID:        entry.ID,
				Content:   entry.Content,
				Type:      MemoryType(entry.Type),
				Metadata:  entry.Metadata,
				Timestamp: entry.Timestamp,
				Tags:      entry.Tags,
				Source:    entry.Provider,
			},
			Score: 1.0, // Default score for memory store entries
		}
	}
	return result
}

func getSessionIDFromData(data *MemoryData) string {
	if data.Metadata != nil {
		if sessionID, exists := data.Metadata["session_id"]; exists {
			if sessionIDStr, ok := sessionID.(string); ok {
				return sessionIDStr
			}
		}
	}
	return "default"
}

func getTitleFromData(data *MemoryData) string {
	if data.Metadata != nil {
		if title, exists := data.Metadata["title"]; exists {
			if titleStr, ok := title.(string); ok {
				return titleStr
			}
		}
	}
	return "Untitled"
}

func calculateRelevance(data *MemoryData) float64 {
	// Simple relevance calculation based on importance and recency
	relevance := data.Importance
	
	// Add recency factor
	recencyHours := time.Since(data.Timestamp).Hours()
	recencyFactor := 1.0 / (1.0 + recencyHours/24.0) // Decay over days
	
	return relevance * recencyFactor
}

func generateContextID(provider, model, sessionID string) string {
	return fmt.Sprintf("ctx_%s_%s_%s_%d", provider, model, sessionID, time.Now().Unix())
}