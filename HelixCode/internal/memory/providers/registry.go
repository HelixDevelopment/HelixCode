package providers

import (
	"fmt"
	"sync"

	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory"
)

// ProviderRegistry manages provider registration and creation
type ProviderRegistry struct {
	mu          sync.RWMutex
	providers   map[memory.ProviderType]ProviderFactoryFunc
	logger      logging.Logger
	initialized bool
}

// ProviderFactoryFunc creates a provider instance
type ProviderFactoryFunc func(config map[string]interface{}) (VectorProvider, error)

var (
	// Global registry instance
	globalRegistry *ProviderRegistry
	once           sync.Once
)

// GetRegistry returns the global provider registry
func GetRegistry() *ProviderRegistry {
	once.Do(func() {
		globalRegistry = NewProviderRegistry()
	})
	return globalRegistry
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	registry := &ProviderRegistry{
		providers: make(map[memory.ProviderType]ProviderFactory),
		logger:    logging.NewLogger("provider_registry"),
	}

	// Register all built-in providers
	registry.registerBuiltInProviders()
	return registry
}

// registerBuiltInProviders registers all built-in provider factories
func (r *ProviderRegistry) registerBuiltInProviders() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Register all built-in providers
	r.providers[memory.ProviderPinecone] = NewPineconeProvider
	r.providers[memory.ProviderMilvus] = NewMilvusProvider
	r.providers[memory.ProviderWeaviate] = NewWeaviateProvider
	r.providers[memory.ProviderQdrant] = NewQdrantProvider
	r.providers[memory.ProviderRedis] = NewRedisProvider
	r.providers[memory.ProviderChroma] = NewChromaProvider

	r.providers[memory.ProviderOpenAI] = NewOpenAIProvider
	r.providers[memory.ProviderAnthropic] = NewAnthropicProvider
	r.providers[memory.ProviderCohere] = NewCohereProvider
	r.providers[memory.ProviderHuggingFace] = NewHuggingFaceProvider
	r.providers[memory.ProviderMistral] = NewMistralProvider
	r.providers[memory.ProviderGemini] = NewGeminiProvider
	r.providers[memory.ProviderGemma] = NewGemmaProvider
	r.providers[memory.ProviderLlamaIndex] = NewLlamaIndexProvider

	r.providers[memory.ProviderVertexAI] = NewVertexAIProvider
	r.providers[memory.ProviderClickHouse] = NewClickHouseProvider
	r.providers[memory.ProviderSupabase] = NewSupabaseProvider
	r.providers[memory.ProviderDeepLake] = NewDeepLakeProvider
	r.providers[memory.ProviderFAISS] = NewFAISSProvider

	r.providers[memory.ProviderMemGPT] = NewMemGPTProvider
	r.providers[memory.ProviderCrewAI] = NewCrewAIProvider
	r.providers[memory.ProviderCharacterAI] = NewCharacterAIProvider
	r.providers[memory.ProviderReplika] = NewReplikaProvider
	r.providers[memory.ProviderAnima] = NewAnimaProvider

	r.providers[memory.ProviderAgnostic] = NewProviderAgnosticProvider

	return nil

	r.initialized = true
	r.logger.Info("Provider registry initialized",
		"total_providers", len(r.providers))
}

// RegisterProvider registers a new provider factory
func (r *ProviderRegistry) RegisterProvider(providerType memory.ProviderType, factory ProviderFactoryFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[providerType]; exists {
		return fmt.Errorf("provider type %s already registered", providerType)
	}

	r.providers[providerType] = factory
	r.logger.Info("Provider registered", "type", providerType)
	return nil
}

// UnregisterProvider unregisters a provider factory
func (r *ProviderRegistry) UnregisterProvider(providerType memory.ProviderType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[providerType]; !exists {
		return fmt.Errorf("provider type %s not registered", providerType)
	}

	delete(r.providers, providerType)
	r.logger.Info("Provider unregistered", "type", providerType)
	return nil
}

// CreateProvider creates a provider instance
func (r *ProviderRegistry) CreateProvider(providerType memory.ProviderType, config map[string]interface{}) (VectorProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}

	provider, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerType, err)
	}

	r.logger.Info("Provider created", "type", providerType)
	return provider, nil
}

// GetProviderFactory gets the factory for a provider type
func (r *ProviderRegistry) GetProviderFactory(providerType memory.ProviderType) (ProviderFactoryFunc, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}

	return factory, nil
}

// ListProviders returns a list of all registered provider types
func (r *ProviderRegistry) ListProviders() []memory.ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []memory.ProviderType
	for providerType := range r.providers {
		providers = append(providers, providerType)
	}

	return providers
}

// GetProviderInfo returns information about a provider type
func (r *ProviderRegistry) GetProviderInfo(providerType memory.ProviderType) (*ProviderInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}

	// Create a temporary provider to get info
	tempProvider, err := factory(make(map[string]interface{}))
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary provider: %w", err)
	}

	return &ProviderInfo{
		Type:          providerType,
		Name:          tempProvider.GetName(),
		Capabilities:  tempProvider.GetCapabilities(),
		IsCloud:       tempProvider.IsCloud(),
		Configuration: tempProvider.GetConfiguration(),
	}, nil
}

// GetProviderInfoMap returns information about all providers
func (r *ProviderRegistry) GetProviderInfoMap() map[memory.ProviderType]*ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infoMap := make(map[memory.ProviderType]*ProviderInfo)

	for providerType := range r.providers {
		if info, err := r.GetProviderInfo(providerType); err == nil {
			infoMap[providerType] = info
		}
	}

	return infoMap
}

// ProviderInfo contains information about a provider
type ProviderInfo struct {
	Type          memory.ProviderType `json:"type"`
	Name          string              `json:"name"`
	Capabilities  []string            `json:"capabilities"`
	IsCloud       bool                `json:"is_cloud"`
	Configuration interface{}         `json:"configuration"`
	CostInfo      *memory.CostInfo    `json:"cost_info,omitempty"`
}

// ValidateProviderConfig validates a provider configuration
func (r *ProviderRegistry) ValidateProviderConfig(providerType memory.ProviderType, config map[string]interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.providers[providerType]
	if !exists {
		return fmt.Errorf("unknown provider type: %s", providerType)
	}

	// Create a temporary provider to validate config
	provider, err := factory(config)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Additional validation if needed
	if validator, ok := provider.(ProviderConfigValidator); ok {
		if err := validator.ValidateConfig(config); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return nil
}

// ProviderConfigValidator interface for providers that can validate their configuration
type ProviderConfigValidator interface {
	ValidateConfig(config map[string]interface{}) error
}

// GetCompatibleProviders returns providers compatible with the given requirements
func (r *ProviderRegistry) GetCompatibleProviders(requirements *ProviderRequirements) []memory.ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var compatible []memory.ProviderType

	for providerType := range r.providers {
		if r.isCompatible(providerType, requirements) {
			compatible = append(compatible, providerType)
		}
	}

	return compatible
}

// ProviderRequirements defines requirements for provider selection
type ProviderRequirements struct {
	Capabilities     []string `json:"capabilities"`
	IsCloud          *bool    `json:"is_cloud,omitempty"`
	MaxCost          float64  `json:"max_cost,omitempty"`
	MinPerformance   float64  `json:"min_performance,omitempty"`
	SupportedMetrics []string `json:"supported_metrics,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

// isCompatible checks if a provider meets the requirements
func (r *ProviderRegistry) isCompatible(providerType memory.ProviderType, requirements *ProviderRequirements) bool {
	info, err := r.GetProviderInfo(providerType)
	if err != nil {
		return false
	}

	// Check capabilities
	if len(requirements.Capabilities) > 0 {
		providerCaps := make(map[string]bool)
		for _, cap := range info.Capabilities {
			providerCaps[cap] = true
		}

		for _, reqCap := range requirements.Capabilities {
			if !providerCaps[reqCap] {
				return false
			}
		}
	}

	// Check cloud requirement
	if requirements.IsCloud != nil {
		if info.IsCloud != *requirements.IsCloud {
			return false
		}
	}

	// Check cost requirement
	if requirements.MaxCost > 0 && info.CostInfo != nil {
		if info.CostInfo.TotalCost > requirements.MaxCost {
			return false
		}
	}

	return true
}

// CreateProviderWithDefaults creates a provider with default configuration
func (r *ProviderRegistry) CreateProviderWithDefaults(providerType memory.ProviderType) (VectorProvider, error) {
	defaults := r.GetDefaultConfig(providerType)
	return r.CreateProvider(providerType, defaults)
}

// GetDefaultConfig returns the default configuration for a provider type
func (r *ProviderRegistry) GetDefaultConfig(providerType memory.ProviderType) map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch providerType {
	case ProviderTypePinecone:
		return map[string]interface{}{
			"environment": "us-west1-gcp",
			"index_name":  "vectors",
			"dimension":   1536,
			"metric":      "cosine",
		}
	case memory.ProviderMilvus:
		return map[string]interface{}{
			"host":        "localhost",
			"port":        19530,
			"database":    "default",
			"index_type":  "IVF_FLAT",
			"metric_type": "L2",
		}
	case memory.ProviderOpenAI:
		return map[string]interface{}{
			"model":       "text-embedding-3-small",
			"timeout":     30,
			"max_retries": 3,
		}
	case memory.ProviderAnthropic:
		return map[string]interface{}{
			"model":       "claude-3-haiku-20240307",
			"timeout":     30,
			"max_retries": 3,
		}
	case memory.ProviderRedis:
		return map[string]interface{}{
			"addr":          "localhost:6379",
			"db":            0,
			"enable_search": true,
			"compression":   true,
		}
	case memory.ProviderChroma:
		return map[string]interface{}{
			"host": "localhost",
			"port": 8000,
			"path": "./chroma_db",
		}
	case memory.ProviderQdrant:
		return map[string]interface{}{
			"host":       "localhost",
			"port":       6333,
			"api_key":    "",
			"collection": "vectors",
		}
	case memory.ProviderWeaviate:
		return map[string]interface{}{
			"url":        "http://localhost:8080",
			"api_key":    "",
			"batch_size": 100,
		}
	case memory.ProviderMemGPT:
		return map[string]interface{}{
			"base_url":   "https://api.memgpt.ai",
			"model":      "memgpt-1.0",
			"max_tokens": 4096,
		}
	case memory.ProviderCrewAI:
		return map[string]interface{}{
			"base_url":           "https://api.crewai.ai",
			"max_agents":         10,
			"parallel_execution": true,
		}
	case memory.ProviderCharacterAI:
		return map[string]interface{}{
			"base_url":            "https://api.character.ai",
			"max_characters":      1000,
			"relationship_memory": true,
		}
	case memory.ProviderReplika:
		return map[string]interface{}{
			"base_url":          "https://api.replika.ai",
			"max_personalities": 1000,
			"emotional_memory":  true,
		}
	case memory.ProviderAnima:
		return map[string]interface{}{
			"base_url":           "https://api.anima.ai",
			"max_avatars":        1000,
			"emotional_tracking": true,
		}
	case memory.ProviderGemma:
		return map[string]interface{}{
			"base_url":            "https://api.gemma.ai",
			"model":               "gemma-7b",
			"embedding_dimension": 4096,
			"gpu_enabled":         true,
		}
	case memory.ProviderLlamaIndex:
		return map[string]interface{}{
			"storage_type": "local",
			"persist_dir":  "./llama_index",
			"chunk_size":   1024,
		}
	case memory.ProviderCohere:
		return map[string]interface{}{
			"model":       "embed-english-v3.0",
			"timeout":     30,
			"max_retries": 3,
		}
	case memory.ProviderHuggingFace:
		return map[string]interface{}{
			"model":   "sentence-transformers/all-MiniLM-L6-v2",
			"task":    "feature-extraction",
			"timeout": 30,
		}
	case memory.ProviderMistral:
		return map[string]interface{}{
			"model":       "mistral-embed",
			"timeout":     30,
			"max_retries": 3,
		}
	case memory.ProviderGemini:
		return map[string]interface{}{
			"model":       "text-embedding-004",
			"timeout":     30,
			"max_retries": 3,
		}
	case memory.ProviderVertexAI:
		return map[string]interface{}{
			"project_id": "",
			"location":   "us-central1",
			"index_name": "vectors",
		}
	case memory.ProviderClickHouse:
		return map[string]interface{}{
			"host":     "localhost",
			"port":     9000,
			"database": "vectors",
			"table":    "embeddings",
		}
	case memory.ProviderSupabase:
		return map[string]interface{}{
			"url":   "",
			"key":   "",
			"table": "vectors",
		}
	case memory.ProviderDeepLake:
		return map[string]interface{}{
			"path":               "./deeplake",
			"embedding_function": "text-embedding-ada-002",
		}
	case memory.ProviderFAISS:
		return map[string]interface{}{
			"index_type": "IVF",
			"dimension":  1536,
			"nlist":      100,
			"metric":     "cosine",
		}
	case memory.ProviderAgnostic:
		return map[string]interface{}{
			"storage_type":       "memory",
			"enable_persistence": false,
		}
	default:
		return make(map[string]interface{})
	}
}

// GetProviderStatistics returns statistics about the registry
func (r *ProviderRegistry) GetProviderStatistics() *RegistryStatistics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cloudProviders := 0
	localProviders := 0
	providersByType := make(map[string]int)

	for providerType := range r.providers {
		info, err := r.GetProviderInfo(providerType)
		if err == nil {
			if info.IsCloud {
				cloudProviders++
			} else {
				localProviders++
			}

			category := r.getProviderCategory(providerType)
			providersByType[category]++
		}
	}

	return &RegistryStatistics{
		TotalProviders:  len(r.providers),
		CloudProviders:  cloudProviders,
		LocalProviders:  localProviders,
		ProvidersByType: providersByType,
		Initialized:     r.initialized,
	}
}

// RegistryStatistics contains statistics about the provider registry
type RegistryStatistics struct {
	TotalProviders  int            `json:"total_providers"`
	CloudProviders  int            `json:"cloud_providers"`
	LocalProviders  int            `json:"local_providers"`
	ProvidersByType map[string]int `json:"providers_by_type"`
	Initialized     bool           `json:"initialized"`
}

// getProviderCategory returns the category of a provider type
func (r *ProviderRegistry) getProviderCategory(providerType memory.ProviderType) string {
	switch {
	case providerType == ProviderTypePinecone ||
		providerType == ProviderTypeMilvus ||
		providerType == ProviderTypeWeaviate ||
		providerType == ProviderTypeQdrant ||
		providerType == ProviderTypeRedis ||
		providerType == ProviderTypeChroma ||
		providerType == ProviderTypeFAISS ||
		providerType == ProviderTypeDeepLake ||
		providerType == ProviderTypeClickHouse ||
		providerType == ProviderTypeSupabase ||
		providerType == ProviderTypeVertexAI:
		return "vector_database"
	case providerType == memory.ProviderMemGPT ||
		providerType == memory.ProviderCrewAI ||
		providerType == memory.ProviderCharacterAI ||
		providerType == memory.ProviderReplika ||
		providerType == memory.ProviderAnima:
		return "ai_memory"
	case providerType == ProviderTypeAgnostic:
		return "utility"
	default:
		return "unknown"
	}
}
