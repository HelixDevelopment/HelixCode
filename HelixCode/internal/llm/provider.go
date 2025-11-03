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
	ProviderTypeQwen       ProviderType = "qwen"
	ProviderTypeXAI        ProviderType = "xai"
	ProviderTypeOpenRouter ProviderType = "openrouter"
	ProviderTypeCopilot    ProviderType = "copilot"
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
	providers map[ProviderType]Provider
	config    ProviderConfig
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
		providers: make(map[ProviderType]Provider),
		config:    config,
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

	// Generate response
	response, err := provider.Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %v", err)
	}

	return response, nil
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
	case ProviderTypeQwen:
		return NewQwenProvider(config)
	case ProviderTypeXAI:
		return NewXAIProvider(config)
	case ProviderTypeOpenRouter:
		return NewOpenRouterProvider(config)
	case ProviderTypeCopilot:
		return NewCopilotProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}
