package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/llm/compressioniface"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory/providers"
)

// AIIntegration provides unified interface for AI systems integration
type AIIntegration struct {
	mu              sync.RWMutex
	registry        *providers.ProviderRegistry
	manager         *providers.ProviderManager
	vector          *VectorIntegration
	memory          *MemoryIntegration
	logger          *logging.Logger
	config          *AIConfig
	providers       map[string]AIProvider
	conversationMgr *ConversationManager
	personalityMgr  *PersonalityManager
}

// AIConfig contains AI integration configuration
type AIConfig struct {
	DefaultLLM       string                       `json:"default_llm"`
	DefaultMemory    string                       `json:"default_memory"`
	Providers        map[string]*AIProviderConfig `json:"providers"`
	VectorConfig     *VectorConfig                `json:"vector_config"`
	MemoryConfig     *MemoryConfig                `json:"memory_config"`
	CacheEnabled     bool                         `json:"cache_enabled"`
	CacheSize        int                          `json:"cache_size"`
	CacheTTL         int64                        `json:"cache_ttl"`
	ProfilingEnabled bool                         `json:"profiling_enabled"`
}

// AIProviderConfig contains configuration for AI provider
type AIProviderConfig struct {
	Type             providers.ProviderType `json:"type"`
	Enabled          bool                   `json:"enabled"`
	Config           map[string]interface{} `json:"config"`
	Model            string                 `json:"model"`
	MaxTokens        int                    `json:"max_tokens"`
	Temperature      float64                `json:"temperature"`
	TopP             float64                `json:"top_p"`
	FrequencyPenalty float64                `json:"frequency_penalty"`
	PresencePenalty  float64                `json:"presence_penalty"`
}

// AIProvider defines interface for AI providers
type AIProvider interface {
	GenerateText(ctx context.Context, prompt string, options *GenerationOptions) (*GenerationResult, error)
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	GenerateChat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*ChatResult, error)
	ClassifyText(ctx context.Context, text string, categories []string) (*ClassificationResult, error)
	ExtractEntities(ctx context.Context, text string) ([]*Entity, error)
	GetCapabilities() []string
	GetCostInfo() *CostInfo
}

// GenerationOptions contains options for text generation
type GenerationOptions struct {
	MaxTokens        int                `json:"max_tokens"`
	Temperature      float64            `json:"temperature"`
	TopP             float64            `json:"top_p"`
	FrequencyPenalty float64            `json:"frequency_penalty"`
	PresencePenalty  float64            `json:"presence_penalty"`
	Stop             []string           `json:"stop"`
	Stream           bool               `json:"stream"`
	Callback         func(string) error `json:"callback"`
}

// GenerationResult contains result of text generation
type GenerationResult struct {
	Text         string                 `json:"text"`
	Tokens       int                    `json:"tokens"`
	FinishReason string                 `json:"finish_reason"`
	Metadata     map[string]interface{} `json:"metadata"`
	Cost         float64                `json:"cost"`
	Duration     time.Duration          `json:"duration"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role     string                 `json:"role"`
	Content  string                 `json:"content"`
	Name     string                 `json:"name,omitempty"`
	Tokens   int                    `json:"tokens"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ChatOptions contains options for chat generation
type ChatOptions struct {
	Model            string   `json:"model"`
	MaxTokens        int      `json:"max_tokens"`
	Temperature      float64  `json:"temperature"`
	TopP             float64  `json:"top_p"`
	FrequencyPenalty float64  `json:"frequency_penalty"`
	PresencePenalty  float64  `json:"presence_penalty"`
	Stop             []string `json:"stop"`
	Stream           bool     `json:"stream"`
	SystemPrompt     string   `json:"system_prompt"`
	Tools            []string `json:"tools"`
}

// ChatResult contains result of chat generation
type ChatResult struct {
	Message      *ChatMessage           `json:"message"`
	Messages     []*ChatMessage         `json:"messages"`
	Tokens       int                    `json:"tokens"`
	FinishReason string                 `json:"finish_reason"`
	Metadata     map[string]interface{} `json:"metadata"`
	Cost         float64                `json:"cost"`
	Duration     time.Duration          `json:"duration"`
}

// ClassificationResult contains result of text classification
type ClassificationResult struct {
	Category      string           `json:"category"`
	Confidence    float64          `json:"confidence"`
	AllCategories []*CategoryScore `json:"all_categories"`
	Tokens        int              `json:"tokens"`
	Cost          float64          `json:"cost"`
	Duration      time.Duration    `json:"duration"`
}

// CategoryScore contains category with score
type CategoryScore struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
}

// Entity represents an extracted entity
type Entity struct {
	Type       string                 `json:"type"`
	Text       string                 `json:"text"`
	Confidence float64                `json:"confidence"`
	Start      int                    `json:"start"`
	End        int                    `json:"end"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// CostInfo contains cost information
type CostInfo struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	Cost         float64 `json:"cost"`
	Currency     string  `json:"currency"`
}

// NewAIIntegration creates a new AI integration
func NewAIIntegration(config *AIConfig) *AIIntegration {
	if config == nil {
		config = &AIConfig{
			DefaultLLM:    "openai",
			DefaultMemory: "memgpt",
			Providers: map[string]*AIProviderConfig{
				"openai": {
					Type:             providers.ProviderTypeOpenAI,
					Enabled:          true,
					Model:            "gpt-4",
					MaxTokens:        4096,
					Temperature:      0.7,
					TopP:             1.0,
					FrequencyPenalty: 0.0,
					PresencePenalty:  0.0,
					Config: map[string]interface{}{
						"api_key": "",
					},
				},
				"anthropic": {
					Type:             providers.ProviderTypeAnthropic,
					Enabled:          true,
					Model:            "claude-3-haiku-20240307",
					MaxTokens:        4096,
					Temperature:      0.7,
					TopP:             1.0,
					FrequencyPenalty: 0.0,
					PresencePenalty:  0.0,
					Config: map[string]interface{}{
						"api_key": "",
					},
				},
				"memgpt": {
					Type:        providers.ProviderTypeMemGPT,
					Enabled:     true,
					Model:       "memgpt-1.0",
					MaxTokens:   4096,
					Temperature: 0.7,
					Config: map[string]interface{}{
						"base_url": "https://api.memgpt.ai",
					},
				},
			},
			CacheEnabled:     true,
			CacheSize:        1000,
			CacheTTL:         300000, // 5 minutes
			ProfilingEnabled: false,
		}
	}

	ai := &AIIntegration{
		registry:  providers.GetRegistry(),
		logger:    logging.NewLogger(logging.INFO),
		config:    config,
		providers: make(map[string]AIProvider),
	}

	// Initialize vector integration
	ai.vector = NewVectorIntegration(config.VectorConfig)

	// Initialize memory integration
	ai.memory = NewMemoryIntegration(config.MemoryConfig)

	return ai
}

// Initialize initializes AI integration
func (ai *AIIntegration) Initialize(ctx context.Context) error {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.logger.Info("Initializing AI integration: default_llm=%s, default_memory=%s, providers_count=%d", ai.config.DefaultLLM, ai.config.DefaultMemory, len(ai.config.Providers))

	// Initialize vector integration
	if err := ai.vector.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize vector integration: %w", err)
	}

	// Initialize memory integration
	if err := ai.memory.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize memory integration: %w", err)
	}

	// Initialize AI providers
	for name, providerConfig := range ai.config.Providers {
		if !providerConfig.Enabled {
			ai.logger.Info("Skipping disabled AI provider: name=%s", name)
			continue
		}

		provider, err := ai.createAIProvider(providerConfig)
		if err != nil {
			ai.logger.Error("Failed to create AI provider: name=%s, error=%v", name, err)
			continue
		}

		ai.providers[name] = provider
		ai.logger.Info("AI provider created: name=%s, type=%s, model=%s", name, providerConfig.Type, providerConfig.Model)
	}

	// Initialize conversation manager
	ai.conversationMgr = NewConversationManager(ai, ai.config)

	// Initialize personality manager
	ai.personalityMgr = NewPersonalityManager(ai, ai.config)

	ai.logger.Info("AI integration initialized successfully")
	return nil
}

// createAIProvider creates an AI provider instance
func (ai *AIIntegration) createAIProvider(config *AIProviderConfig) (AIProvider, error) {
	switch config.Type {
	case providers.ProviderTypeOpenAI:
		return NewOpenAIProvider(config), nil
	case providers.ProviderTypeAnthropic:
		return NewAnthropicProvider(config), nil
	case providers.ProviderTypeCohere:
		return NewCohereProvider(config), nil
	case providers.ProviderTypeHuggingFace:
		return NewHuggingFaceProvider(config), nil
	case providers.ProviderTypeMistral:
		return NewMistralProvider(config), nil
	case providers.ProviderTypeGemini:
		return NewGeminiProvider(config), nil
	case providers.ProviderTypeGemma:
		return NewGemmaProvider(config), nil
	case providers.ProviderTypeLlamaIndex:
		return NewLlamaIndexProvider(config), nil
	case providers.ProviderTypeMemGPT:
		return NewMemGPTAIProvider(config), nil
	case providers.ProviderTypeCrewAI:
		return NewCrewAIProvider(config), nil
	case providers.ProviderTypeCharacterAI:
		return NewCharacterAIProvider(config), nil
	case providers.ProviderTypeReplika:
		return NewReplikaAIProvider(config), nil
	case providers.ProviderTypeAnima:
		return NewAnimaAIProvider(config), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider type: %s", config.Type)
	}
}

// GenerateText generates text using default LLM
func (ai *AIIntegration) GenerateText(ctx context.Context, prompt string, options *GenerationOptions) (*GenerationResult, error) {
	return ai.GenerateTextWithProvider(ctx, ai.config.DefaultLLM, prompt, options)
}

// GenerateTextWithProvider generates text using specific provider
func (ai *AIIntegration) GenerateTextWithProvider(ctx context.Context, providerName string, prompt string, options *GenerationOptions) (*GenerationResult, error) {
	ai.mu.RLock()
	provider, exists := ai.providers[providerName]
	ai.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("AI provider not found: %s", providerName)
	}

	start := time.Now()
	defer func() {
		ai.logger.Debug("Text generation completed: provider=%s, duration=%v", providerName, time.Since(start))
	}()

	result, err := provider.GenerateText(ctx, prompt, options)
	if err != nil {
		ai.logger.Error("Text generation failed: provider=%s, error=%v", providerName, err)
		return nil, err
	}

	// Store generation in memory
	if ai.config.DefaultMemory != "" {
		ai.memory.StoreGeneration(ctx, providerName, prompt, result)
	}

	return result, nil
}

// GenerateChat generates chat response using default LLM
func (ai *AIIntegration) GenerateChat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*ChatResult, error) {
	return ai.GenerateChatWithProvider(ctx, ai.config.DefaultLLM, messages, options)
}

// GenerateChatWithProvider generates chat response using specific provider
func (ai *AIIntegration) GenerateChatWithProvider(ctx context.Context, providerName string, messages []*ChatMessage, options *ChatOptions) (*ChatResult, error) {
	ai.mu.RLock()
	provider, exists := ai.providers[providerName]
	ai.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("AI provider not found: %s", providerName)
	}

	start := time.Now()
	defer func() {
		ai.logger.Debug("Chat generation completed: provider=%s, duration=%v", providerName, time.Since(start))
	}()

	result, err := provider.GenerateChat(ctx, messages, options)
	if err != nil {
		ai.logger.Error("Chat generation failed: provider=%s, error=%v", providerName, err)
		return nil, err
	}

	// Store conversation in memory
	if ai.config.DefaultMemory != "" {
		ai.memory.StoreConversation(ctx, providerName, messages, result)
	}

	return result, nil
}

// GenerateEmbedding generates embedding using default LLM
func (ai *AIIntegration) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return ai.GenerateEmbeddingWithProvider(ctx, ai.config.DefaultLLM, text)
}

// GenerateEmbeddingWithProvider generates embedding using specific provider
func (ai *AIIntegration) GenerateEmbeddingWithProvider(ctx context.Context, providerName string, text string) ([]float64, error) {
	ai.mu.RLock()
	provider, exists := ai.providers[providerName]
	ai.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("AI provider not found: %s", providerName)
	}

	start := time.Now()
	defer func() {
		ai.logger.Debug("Embedding generation completed: provider=%s, duration=%v", providerName, time.Since(start))
	}()

	embedding, err := provider.GenerateEmbedding(ctx, text)
	if err != nil {
		ai.logger.Error("Embedding generation failed: provider=%s, error=%v", providerName, err)
		return nil, err
	}

	// Store embedding in vector database
	if ai.vector != nil {
		vectorData := &VectorData{
			ID:        fmt.Sprintf("embed_%s_%d", providerName, time.Now().UnixNano()),
			Embedding: embedding,
			Metadata: map[string]interface{}{
				"provider":    providerName,
				"text":        text,
				"text_length": len(text),
				"created_at":  time.Now(),
			},
			IndexName: "text_embeddings",
			CreatedAt: time.Now(),
		}

		if err := ai.vector.StoreVector(ctx, vectorData); err != nil {
			ai.logger.Warn("Failed to store embedding in vector database: %v", err)
		}
	}

	return embedding, nil
}

// ClassifyText classifies text using default LLM
func (ai *AIIntegration) ClassifyText(ctx context.Context, text string, categories []string) (*ClassificationResult, error) {
	return ai.ClassifyTextWithProvider(ctx, ai.config.DefaultLLM, text, categories)
}

// ClassifyTextWithProvider classifies text using specific provider
func (ai *AIIntegration) ClassifyTextWithProvider(ctx context.Context, providerName string, text string, categories []string) (*ClassificationResult, error) {
	ai.mu.RLock()
	provider, exists := ai.providers[providerName]
	ai.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("AI provider not found: %s", providerName)
	}

	start := time.Now()
	defer func() {
		ai.logger.Debug("Text classification completed: provider=%s, duration=%v", providerName, time.Since(start))
	}()

	result, err := provider.ClassifyText(ctx, text, categories)
	if err != nil {
		ai.logger.Error("Text classification failed: provider=%s, error=%v", providerName, err)
		return nil, err
	}

	return result, nil
}

// ExtractEntities extracts entities using default LLM
func (ai *AIIntegration) ExtractEntities(ctx context.Context, text string) ([]*Entity, error) {
	return ai.ExtractEntitiesWithProvider(ctx, ai.config.DefaultLLM, text)
}

// ExtractEntitiesWithProvider extracts entities using specific provider
func (ai *AIIntegration) ExtractEntitiesWithProvider(ctx context.Context, providerName string, text string) ([]*Entity, error) {
	ai.mu.RLock()
	provider, exists := ai.providers[providerName]
	ai.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("AI provider not found: %s", providerName)
	}

	start := time.Now()
	defer func() {
		ai.logger.Debug("Entity extraction completed: provider=%s, duration=%v", providerName, time.Since(start))
	}()

	entities, err := provider.ExtractEntities(ctx, text)
	if err != nil {
		ai.logger.Error("Entity extraction failed: provider=%s, error=%v", providerName, err)
		return nil, err
	}

	return entities, nil
}

// GetConversation returns conversation manager
func (ai *AIIntegration) GetConversation() *ConversationManager {
	return ai.conversationMgr
}

// GetPersonality returns personality manager
func (ai *AIIntegration) GetPersonality() *PersonalityManager {
	return ai.personalityMgr
}

// GetVector returns vector integration
func (ai *AIIntegration) GetVector() *VectorIntegration {
	return ai.vector
}

// GetMemory returns memory integration
func (ai *AIIntegration) GetMemory() *MemoryIntegration {
	return ai.memory
}

// GetProvider returns specific AI provider
func (ai *AIIntegration) GetProvider(name string) (AIProvider, error) {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	provider, exists := ai.providers[name]
	if !exists {
		return nil, fmt.Errorf("AI provider not found: %s", name)
	}

	return provider, nil
}

// ListProviders returns list of available AI providers
func (ai *AIIntegration) ListProviders() []string {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	var providers []string
	for name := range ai.providers {
		providers = append(providers, name)
	}

	return providers
}

// GetStats returns statistics about AI integration
func (ai *AIIntegration) GetStats(ctx context.Context) (*AIStats, error) {
	stats := &AIStats{
		Providers: make(map[string]*AIProviderStats),
	}

	// Get stats from each provider
	for name, provider := range ai.providers {
		if aiProvider, ok := provider.(AIStatsProvider); ok {
			providerStats, err := aiProvider.GetStats()
			if err == nil {
				stats.Providers[name] = providerStats
			}
		}
	}

	// Get vector stats
	if ai.vector != nil {
		vectorStats, err := ai.vector.GetVectorStats(ctx)
		if err == nil {
			stats.VectorStats = vectorStats
		}
	}

	// Get memory stats
	if ai.memory != nil {
		memoryStats, err := ai.memory.GetMemoryStats(ctx)
		if err == nil {
			stats.MemoryStats = memoryStats
		}
	}

	return stats, nil
}

// HealthCheck performs health check on all AI providers
func (ai *AIIntegration) HealthCheck(ctx context.Context) (*AIHealthStatus, error) {
	status := &AIHealthStatus{
		ProviderStatuses: make(map[string]string),
	}

	healthyCount := 0
	totalCount := 0

	for name, provider := range ai.providers {
		totalCount++
		// Simple health check - try to generate a small text
		result, err := provider.GenerateText(ctx, "test", &GenerationOptions{MaxTokens: 10})
		if err == nil && result.Text != "" {
			status.ProviderStatuses[name] = "healthy"
			healthyCount++
		} else {
			status.ProviderStatuses[name] = "unhealthy"
		}
	}

	if healthyCount == totalCount {
		status.Status = "healthy"
	} else if healthyCount > 0 {
		status.Status = "degraded"
	} else {
		status.Status = "unhealthy"
	}

	status.TotalProviders = totalCount
	status.HealthyProviders = healthyCount
	status.LastCheck = time.Now()

	return status, nil
}

// Stop stops AI integration
func (ai *AIIntegration) Stop(ctx context.Context) error {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.logger.Info("Stopping AI integration")

	// Stop vector integration
	if ai.vector != nil {
		if err := ai.vector.Stop(ctx); err != nil {
			ai.logger.Warn("Failed to stop vector integration: %v", err)
		}
	}

	// Stop memory integration
	if ai.memory != nil {
		if err := ai.memory.Stop(ctx); err != nil {
			ai.logger.Warn("Failed to stop memory integration: %v", err)
		}
	}

	// Stop conversation manager
	if ai.conversationMgr != nil {
		if err := ai.conversationMgr.Stop(ctx); err != nil {
			ai.logger.Warn("Failed to stop conversation manager: %v", err)
		}
	}

	// Stop personality manager
	if ai.personalityMgr != nil {
		if err := ai.personalityMgr.Stop(ctx); err != nil {
			ai.logger.Warn("Failed to stop personality manager: %v", err)
		}
	}

	ai.logger.Info("AI integration stopped")
	return nil
}

// AIStats contains statistics about AI integration
type AIStats struct {
	Providers   map[string]*AIProviderStats `json:"providers"`
	VectorStats *VectorStats                `json:"vector_stats"`
	MemoryStats *MemoryStats                `json:"memory_stats"`
	TotalCost   float64                     `json:"total_cost"`
	TotalTokens int                         `json:"total_tokens"`
	LastUpdate  time.Time                   `json:"last_update"`
}

// AIProviderStats contains statistics for AI provider
type AIProviderStats struct {
	Name           string        `json:"name"`
	Type           string        `json:"type"`
	Requests       int64         `json:"requests"`
	Successes      int64         `json:"successes"`
	Failures       int64         `json:"failures"`
	AverageLatency time.Duration `json:"average_latency"`
	TotalCost      float64       `json:"total_cost"`
	TotalTokens    int           `json:"total_tokens"`
	LastRequest    time.Time     `json:"last_request"`
}

// AIHealthStatus contains health status of AI integration
type AIHealthStatus struct {
	Status           string            `json:"status"`
	TotalProviders   int               `json:"total_providers"`
	HealthyProviders int               `json:"healthy_providers"`
	ProviderStatuses map[string]string `json:"provider_statuses"`
	LastCheck        time.Time         `json:"last_check"`
}

// AIStatsProvider interface for providers that can provide statistics
type AIStatsProvider interface {
	GetStats() (*AIProviderStats, error)
}

// Placeholder implementations for missing types and functions
type MemoryIntegration struct {
	// TODO: Add memory integration fields
}

func NewMemoryIntegration(config *MemoryConfig) *MemoryIntegration {
	return &MemoryIntegration{}
}

func (mi *MemoryIntegration) Initialize(ctx context.Context) error {
	return nil
}

func (mi *MemoryIntegration) StoreGeneration(ctx context.Context, providerName, prompt string, generation *GenerationResult) error {
	return nil
}

func (mi *MemoryIntegration) StoreConversation(ctx context.Context, providerName string, messages []*ChatMessage, result *ChatResult) error {
	return nil
}

func (mi *MemoryIntegration) GetMemoryStats(ctx context.Context) (*MemoryStats, error) {
	return &MemoryStats{}, nil
}

func (mi *MemoryIntegration) Stop(ctx context.Context) error {
	return nil
}

type MemoryConfig struct{}
type ConversationManager struct {
	compressionCoordinator compressioniface.CompressionCoordinator
}

func NewConversationManager(ai *AIIntegration, config *AIConfig) *ConversationManager {
	// Initialize compression coordinator
	// TODO: Pass actual LLM provider when available
	compressionConfig := &compressioniface.Config{
		Enabled:              true,
		DefaultStrategy:      compressioniface.StrategyHybrid,
		TokenBudget:          200000,
		WarningThreshold:     150000,
		CompressionThreshold: 180000,
		AutoCompressEnabled:  true,
		AutoCompressInterval: 5 * time.Minute,
	}

	compressionCoordinator, err := compressioniface.NewCoordinatorFactory(nil, compressionConfig)
	if err != nil {
		// Log error but don't fail initialization
		ai.logger.Warn("Failed to initialize compression coordinator: %v", err)
		compressionCoordinator = nil
	}

	return &ConversationManager{
		compressionCoordinator: compressionCoordinator,
	}
}
func (cm *ConversationManager) Stop(ctx context.Context) error { return nil }

type PersonalityManager struct {
	// TODO: Add personality management fields
}

func NewPersonalityManager(ai *AIIntegration, config *AIConfig) *PersonalityManager {
	return &PersonalityManager{}
}
func (pm *PersonalityManager) Stop(ctx context.Context) error { return nil }

type MemoryStats struct{}

// Placeholder provider implementations
func NewOpenAIProvider(config *AIProviderConfig) AIProvider      { return &MockAIProvider{} }
func NewAnthropicProvider(config *AIProviderConfig) AIProvider   { return &MockAIProvider{} }
func NewCohereProvider(config *AIProviderConfig) AIProvider      { return &MockAIProvider{} }
func NewHuggingFaceProvider(config *AIProviderConfig) AIProvider { return &MockAIProvider{} }
func NewMistralProvider(config *AIProviderConfig) AIProvider     { return &MockAIProvider{} }
func NewGeminiProvider(config *AIProviderConfig) AIProvider      { return &MockAIProvider{} }
func NewGemmaProvider(config *AIProviderConfig) AIProvider       { return &MockAIProvider{} }
func NewLlamaIndexProvider(config *AIProviderConfig) AIProvider  { return &MockAIProvider{} }
func NewMemGPTAIProvider(config *AIProviderConfig) AIProvider    { return &MockAIProvider{} }
func NewCrewAIProvider(config *AIProviderConfig) AIProvider      { return &MockAIProvider{} }
func NewCharacterAIProvider(config *AIProviderConfig) AIProvider { return &MockAIProvider{} }
func NewReplikaAIProvider(config *AIProviderConfig) AIProvider   { return &MockAIProvider{} }
func NewAnimaAIProvider(config *AIProviderConfig) AIProvider     { return &MockAIProvider{} }

// MockAIProvider provides mock implementation
type MockAIProvider struct{}

func (m *MockAIProvider) GenerateText(ctx context.Context, prompt string, options *GenerationOptions) (*GenerationResult, error) {
	return &GenerationResult{
		Text:         "Mock generated text",
		Tokens:       20,
		FinishReason: "stop",
		Metadata:     map[string]interface{}{"mock": true},
		Cost:         0.001,
		Duration:     time.Millisecond * 100,
	}, nil
}

func (m *MockAIProvider) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	embedding := make([]float64, 1536)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

func (m *MockAIProvider) GenerateChat(ctx context.Context, messages []*ChatMessage, options *ChatOptions) (*ChatResult, error) {
	return &ChatResult{
		Message: &ChatMessage{
			Role:    "assistant",
			Content: "Mock chat response",
			Tokens:  15,
		},
		Tokens:       25,
		FinishReason: "stop",
		Metadata:     map[string]interface{}{"mock": true},
		Cost:         0.002,
		Duration:     time.Millisecond * 150,
	}, nil
}

func (m *MockAIProvider) ClassifyText(ctx context.Context, text string, categories []string) (*ClassificationResult, error) {
	return &ClassificationResult{
		Category:   categories[0],
		Confidence: 0.8,
		AllCategories: []*CategoryScore{
			{Category: categories[0], Confidence: 0.8},
			{Category: categories[1], Confidence: 0.2},
		},
		Tokens:   10,
		Cost:     0.001,
		Duration: time.Millisecond * 50,
	}, nil
}

func (m *MockAIProvider) ExtractEntities(ctx context.Context, text string) ([]*Entity, error) {
	return []*Entity{
		{
			Type:       "PERSON",
			Text:       "John Doe",
			Confidence: 0.9,
			Start:      0,
			End:        8,
			Metadata:   map[string]interface{}{"mock": true},
		},
	}, nil
}

func (m *MockAIProvider) GetCapabilities() []string {
	return []string{"text_generation", "chat", "embedding", "classification", "entity_extraction"}
}

func (m *MockAIProvider) GetCostInfo() *CostInfo {
	return &CostInfo{
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
		Cost:         0.001,
		Currency:     "USD",
	}
}
