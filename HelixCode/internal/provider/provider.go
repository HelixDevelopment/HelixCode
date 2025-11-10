package provider

import (
	"context"

	"dev.helix.code/internal/hardware"
	"dev.helix.code/internal/llm"
)

// ProviderType represents the type of LLM provider
type ProviderType string

// Provider type constants
const (
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeAnthropic  ProviderType = "anthropic"
	ProviderTypeGemini     ProviderType = "gemini"
	ProviderTypeVertexAI   ProviderType = "vertexai"
	ProviderTypeAzure      ProviderType = "azure"
	ProviderTypeBedrock    ProviderType = "bedrock"
	ProviderTypeGroq       ProviderType = "groq"
	ProviderTypeQwen       ProviderType = "qwen"
	ProviderTypeCopilot    ProviderType = "copilot"
	ProviderTypeOpenRouter ProviderType = "openrouter"
	ProviderTypeXAI        ProviderType = "xai"
	ProviderTypeOllama     ProviderType = "ollama"
	ProviderTypeLocal      ProviderType = "local"
	ProviderTypeLlamaCpp   ProviderType = "llamacpp"
	ProviderTypeVLLM       ProviderType = "vllm"
	ProviderTypeLocalAI    ProviderType = "localai"
)

// Provider represents a generic LLM provider interface
type Provider interface {
	GetType() ProviderType
	GetName() string
	GetModels() []llm.ModelInfo
	GetCapabilities() []llm.ModelCapability
	Generate(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error)
	GenerateStream(ctx context.Context, request *llm.LLMRequest, ch chan<- llm.LLMResponse) error
	IsAvailable(ctx context.Context) bool
	GetHealth(ctx context.Context) (*llm.ProviderHealth, error)
	Close() error

	// Cognee integration methods
	SupportsCognee() bool
	InitializeCognee(config interface{}, options interface{}) error
	GetModelName() string
	GetModelInfo() *llm.ModelInfo
	GetHardwareProfile() *hardware.HardwareProfile
}

// String returns the string representation of ProviderType
func (pt ProviderType) String() string {
	return string(pt)
}
