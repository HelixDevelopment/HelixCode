package llm

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProviderType represents different LLM provider types
type ProviderType string

const (
	ProviderTypeLocal      ProviderType = "local"
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeAnthropic  ProviderType = "anthropic"
	ProviderTypeGemini     ProviderType = "gemini"
	ProviderTypeVertexAI   ProviderType = "vertexai"
	ProviderTypeQwen       ProviderType = "qwen"
	ProviderTypeXAI        ProviderType = "xai"
	ProviderTypeOpenRouter ProviderType = "openrouter"
	ProviderTypeCopilot    ProviderType = "copilot"
	ProviderTypeBedrock    ProviderType = "bedrock"
	ProviderTypeAzure      ProviderType = "azure"
	ProviderTypeGroq       ProviderType = "groq"
	ProviderTypeCustom     ProviderType = "custom"
)

// ModelCapability represents what a model can do
type ModelCapability string

const (
	CapabilityTextGeneration ModelCapability = "text_generation"
	CapabilityCodeGeneration ModelCapability = "code_generation"
	CapabilityCodeAnalysis   ModelCapability = "code_analysis"
	CapabilityPlanning       ModelCapability = "planning"
	CapabilityDebugging      ModelCapability = "debugging"
	CapabilityRefactoring    ModelCapability = "refactoring"
	CapabilityTesting        ModelCapability = "testing"
	CapabilityVision         ModelCapability = "vision"
)

// LLMRequest represents a request to an LLM provider
type LLMRequest struct {
	ID           uuid.UUID         `json:"id"`
	ProviderType ProviderType      `json:"provider_type"`
	Model        string            `json:"model"`
	Messages     []Message         `json:"messages"`
	MaxTokens    int               `json:"max_tokens"`
	Temperature  float64           `json:"temperature"`
	TopP         float64           `json:"top_p"`
	Stream       bool              `json:"stream"`
	Tools        []Tool            `json:"tools"`
	ToolChoice   string            `json:"tool_choice"`
	Capabilities []ModelCapability `json:"capabilities"`
	CreatedAt    time.Time         `json:"created_at"`

	// Advanced features
	Reasoning      *ReasoningConfig `json:"reasoning,omitempty"`       // Reasoning/thinking configuration
	CacheConfig    *CacheConfig     `json:"cache_config,omitempty"`    // Prompt caching configuration
	TokenBudget    *TokenBudget     `json:"token_budget,omitempty"`    // Token budget limits
	ThinkingBudget int              `json:"thinking_budget,omitempty"` // Token budget for thinking
	SessionID      string           `json:"session_id,omitempty"`      // Session ID for tracking
}

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Tool represents a function/tool that the LLM can call
type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition defines a callable function
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// LLMResponse represents a response from an LLM provider
type LLMResponse struct {
	ID               uuid.UUID     `json:"id"`
	RequestID        uuid.UUID     `json:"request_id"`
	Content          string        `json:"content"`
	ToolCalls        []ToolCall    `json:"tool_calls"`
	FinishReason     string        `json:"finish_reason"`
	Usage            Usage         `json:"usage"`
	ProviderMetadata interface{}   `json:"provider_metadata"`
	ProcessingTime   time.Duration `json:"processing_time"`
	CreatedAt        time.Time     `json:"created_at"`
}

// ToolCall represents a tool call from the LLM
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction represents the function call in a tool call
type ToolCallFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Provider defines the interface for LLM providers
type Provider interface {
	// Basic provider information
	GetType() ProviderType
	GetName() string
	GetModels() []ModelInfo
	GetCapabilities() []ModelCapability

	// Core functionality
	Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
	GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error

	// Provider management
	IsAvailable(ctx context.Context) bool
	GetHealth(ctx context.Context) (*ProviderHealth, error)
	Close() error
}

// ModelInfo represents information about an available model
type ModelInfo struct {
	Name           string            `json:"name"`
	Provider       ProviderType      `json:"provider"`
	ContextSize    int               `json:"context_size"`
	Capabilities   []ModelCapability `json:"capabilities"`
	MaxTokens      int               `json:"max_tokens"`
	SupportsTools  bool              `json:"supports_tools"`
	SupportsVision bool              `json:"supports_vision"`
	Description    string            `json:"description"`
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	Status     string        `json:"status"`
	Latency    time.Duration `json:"latency"`
	LastCheck  time.Time     `json:"last_check"`
	ErrorCount int           `json:"error_count"`
	ModelCount int           `json:"model_count"`
}

// ProviderManager manages multiple LLM providers
type ProviderManager struct {
	providers    map[ProviderType]Provider
	config       ProviderConfig
	tokenTracker *TokenTracker // Track token usage across providers
	cacheMetrics *CacheMetrics // Track caching performance
	// Note: Context compaction framework exists in internal/llm/compression/
	// Full integration pending architectural refactor to avoid circular dependencies
}

// ProviderConfig holds configuration for the provider manager
type ProviderConfig struct {
	DefaultProvider ProviderType                   `json:"default_provider"`
	Providers       map[string]ProviderConfigEntry `json:"providers"`
	Timeout         time.Duration                  `json:"timeout"`
	MaxRetries      int                            `json:"max_retries"`
}

// ProviderConfigEntry holds configuration for a specific provider
type ProviderConfigEntry struct {
	Type       ProviderType           `json:"type"`
	Endpoint   string                 `json:"endpoint"`
	APIKey     string                 `json:"api_key"`
	Models     []string               `json:"models"`
	Enabled    bool                   `json:"enabled"`
	Parameters map[string]interface{} `json:"parameters"`
}

// NewProviderManager creates a new provider manager
func NewProviderManager(config ProviderConfig) *ProviderManager {
	return &ProviderManager{
		providers:    make(map[ProviderType]Provider),
		config:       config,
		tokenTracker: NewTokenTracker(DefaultTokenBudget()),
		cacheMetrics: &CacheMetrics{},
	}
}

// NewProviderManagerWithBudget creates a provider manager with custom token budget
func NewProviderManagerWithBudget(config ProviderConfig, budget TokenBudget) *ProviderManager {
	return &ProviderManager{
		providers:    make(map[ProviderType]Provider),
		config:       config,
		tokenTracker: NewTokenTracker(budget),
		cacheMetrics: &CacheMetrics{},
	}
}

// RegisterProvider registers a new LLM provider
func (pm *ProviderManager) RegisterProvider(provider Provider) error {
	providerType := provider.GetType()

	if _, exists := pm.providers[providerType]; exists {
		return fmt.Errorf("provider %s already registered", providerType)
	}

	pm.providers[providerType] = provider
	log.Printf("✅ LLM Provider registered: %s (%s)", provider.GetName(), providerType)
	return nil
}

// GetProvider returns a provider by type
func (pm *ProviderManager) GetProvider(providerType ProviderType) (Provider, error) {
	provider, exists := pm.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerType)
	}

	if !provider.IsAvailable(context.Background()) {
		return nil, fmt.Errorf("provider %s is not available", providerType)
	}

	return provider, nil
}

// GetDefaultProvider returns the default provider
func (pm *ProviderManager) GetDefaultProvider() (Provider, error) {
	return pm.GetProvider(pm.config.DefaultProvider)
}

// Generate uses the appropriate provider to generate a response
func (pm *ProviderManager) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	var provider Provider
	var err error

	// Use specified provider or default
	if request.ProviderType != "" {
		provider, err = pm.GetProvider(request.ProviderType)
	} else {
		provider, err = pm.GetDefaultProvider()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	// Set request ID if not set
	if request.ID == uuid.Nil {
		request.ID = uuid.New()
	}
	request.CreatedAt = time.Now()

	// Apply default reasoning config if not set
	if request.Reasoning == nil && IsReasoningModelByName(request.Model) {
		isReasoning, modelType := IsReasoningModel(request.Model)
		if isReasoning {
			request.Reasoning = NewReasoningConfig(modelType)
		}
	}

	// Apply default cache config if not set
	if request.CacheConfig == nil {
		defaultCache := DefaultCacheConfig()
		request.CacheConfig = &defaultCache
	}

	// Check token budget if enabled
	if request.SessionID != "" && pm.tokenTracker != nil {
		estimatedTokens := EstimateTokens(request)
		estimatedCost := EstimateCost(request.Model, estimatedTokens, 0.01) // Default cost estimation

		if err := pm.tokenTracker.CheckBudget(ctx, request.SessionID, estimatedTokens, estimatedCost); err != nil {
			return nil, fmt.Errorf("budget check failed: %v", err)
		}
	}

	// Generate response
	response, err := provider.Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %v", err)
	}

	// Track token usage
	if request.SessionID != "" && pm.tokenTracker != nil {
		cost := EstimateCost(request.Model, response.Usage.TotalTokens, 0.01)
		pm.tokenTracker.TrackRequest(request.SessionID, request, response, cost)
	}

	// Track cache metrics if available
	if cacheStats, ok := response.ProviderMetadata.(map[string]interface{}); ok {
		if _, hasCacheData := cacheStats["cache_creation_tokens"]; hasCacheData {
			stats := CacheStats{
				CacheCreationInputTokens: getIntFromMap(cacheStats, "cache_creation_tokens"),
				CacheReadInputTokens:     getIntFromMap(cacheStats, "cache_read_tokens"),
				InputTokens:              response.Usage.PromptTokens,
				OutputTokens:             response.Usage.CompletionTokens,
			}
			savings := CalculateCacheSavings(stats, 0.01, 0.001) // Example pricing
			pm.cacheMetrics.UpdateMetrics(stats, savings)
		}
	}

	return response, nil
}

// Helper function to extract int from map
func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if intVal, ok := val.(int); ok {
			return intVal
		}
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
	}
	return 0
}

// IsReasoningModelByName checks if model name suggests reasoning capability
func IsReasoningModelByName(modelName string) bool {
	isReasoning, _ := IsReasoningModel(modelName)
	return isReasoning
}

// GetAvailableProviders returns all available providers
func (pm *ProviderManager) GetAvailableProviders() []Provider {
	var available []Provider

	for _, provider := range pm.providers {
		if provider.IsAvailable(context.Background()) {
			available = append(available, provider)
		}
	}

	return available
}

// GetProviderHealth returns health status for all providers
func (pm *ProviderManager) GetProviderHealth(ctx context.Context) map[ProviderType]*ProviderHealth {
	health := make(map[ProviderType]*ProviderHealth)

	for providerType, provider := range pm.providers {
		if healthStatus, err := provider.GetHealth(ctx); err == nil {
			health[providerType] = healthStatus
		} else {
			health[providerType] = &ProviderHealth{
				Status:     "unhealthy",
				LastCheck:  time.Now(),
				ErrorCount: 1,
			}
		}
	}

	return health
}

// FindProviderForCapabilities finds providers that support specific capabilities
func (pm *ProviderManager) FindProviderForCapabilities(capabilities []ModelCapability) []Provider {
	var matching []Provider

	for _, provider := range pm.GetAvailableProviders() {
		providerCaps := provider.GetCapabilities()
		if hasAllCapabilities(providerCaps, capabilities) {
			matching = append(matching, provider)
		}
	}

	return matching
}

// GetTokenTracker returns the token tracker for budget management
func (pm *ProviderManager) GetTokenTracker() *TokenTracker {
	return pm.tokenTracker
}

// GetCacheMetrics returns cache performance metrics
func (pm *ProviderManager) GetCacheMetrics() *CacheMetrics {
	return pm.cacheMetrics
}

// GetSessionUsage returns token usage for a session
func (pm *ProviderManager) GetSessionUsage(sessionID string) (*SessionUsage, error) {
	if pm.tokenTracker == nil {
		return nil, fmt.Errorf("token tracker not initialized")
	}
	return pm.tokenTracker.GetSessionUsage(sessionID)
}

// GetBudgetStatus returns budget status for a session
func (pm *ProviderManager) GetBudgetStatus(sessionID string) *BudgetStatus {
	if pm.tokenTracker == nil {
		return nil
	}
	return pm.tokenTracker.GetBudgetStatus(sessionID)
}

// ResetSession clears usage data for a session
func (pm *ProviderManager) ResetSession(sessionID string) {
	if pm.tokenTracker != nil {
		pm.tokenTracker.ResetSession(sessionID)
	}
}

// Close closes all providers
func (pm *ProviderManager) Close() error {
	var errors []string

	for _, provider := range pm.providers {
		if err := provider.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", provider.GetType(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing providers: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Helper functions

func hasAllCapabilities(available []ModelCapability, required []ModelCapability) bool {
	availableMap := make(map[ModelCapability]bool)
	for _, cap := range available {
		availableMap[cap] = true
	}

	for _, req := range required {
		if !availableMap[req] {
			return false
		}
	}

	return true
}

// Common errors
var (
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrModelNotFound       = errors.New("model not found")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrRateLimited         = errors.New("rate limited")
	ErrContextTooLong      = errors.New("context too long")
)

// ProviderFactory creates providers based on configuration
type ProviderFactory struct{}

// CreateProvider creates a provider from configuration
func (pf *ProviderFactory) CreateProvider(config ProviderConfigEntry) (Provider, error) {
	switch config.Type {
	case ProviderTypeLocal:
		return NewLocalProvider(config)
	case ProviderTypeOpenAI:
		return NewOpenAIProvider(config)
	case ProviderTypeAnthropic:
		return NewAnthropicProvider(config)
	case ProviderTypeGemini:
		return NewGeminiProvider(config)
	case ProviderTypeVertexAI:
		return NewVertexAIProvider(config)
	case ProviderTypeQwen:
		return NewQwenProvider(config)
	case ProviderTypeXAI:
		return NewXAIProvider(config)
	case ProviderTypeOpenRouter:
		return NewOpenRouterProvider(config)
	case ProviderTypeCopilot:
		return NewCopilotProvider(config)
	case ProviderTypeBedrock:
		return NewBedrockProvider(config)
	case ProviderTypeAzure:
		return NewAzureProvider(config)
	case ProviderTypeGroq:
		return NewGroqProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}
