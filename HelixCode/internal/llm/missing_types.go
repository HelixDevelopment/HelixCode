package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Missing types for command line interface

// ModelCapability represents the capabilities of an LLM model
type ModelCapability string

// Model capability constants
const (
	CapabilityTextGeneration ModelCapability = "text_generation"
	CapabilityCodeGeneration ModelCapability = "code_generation"
	CapabilityCodeAnalysis   ModelCapability = "code_analysis"
	CapabilityPlanning       ModelCapability = "planning"
	CapabilityDebugging      ModelCapability = "debugging"
	CapabilityRefactoring    ModelCapability = "refactoring"
	CapabilityTesting        ModelCapability = "testing"
	CapabilityVision         ModelCapability = "vision"
	CapabilityReasoning      ModelCapability = "reasoning"
	CapabilityAnalysis       ModelCapability = "analysis"
	CapabilityWriting        ModelCapability = "writing"
	CapabilityDocumentation  ModelCapability = "documentation"
)

// ProviderType represents the type of LLM provider
type ProviderType string

// Provider type constants
const (
	ProviderTypeOpenAI      ProviderType = "openai"
	ProviderTypeAnthropic   ProviderType = "anthropic"
	ProviderTypeGemini      ProviderType = "gemini"
	ProviderTypeVertexAI    ProviderType = "vertexai"
	ProviderTypeAzure       ProviderType = "azure"
	ProviderTypeBedrock     ProviderType = "bedrock"
	ProviderTypeGroq        ProviderType = "groq"
	ProviderTypeQwen        ProviderType = "qwen"
	ProviderTypeCopilot     ProviderType = "copilot"
	ProviderTypeOpenRouter  ProviderType = "openrouter"
	ProviderTypeXAI         ProviderType = "xai"
	ProviderTypeOllama      ProviderType = "ollama"
	ProviderTypeLocal       ProviderType = "local"
	ProviderTypeLlamaCpp    ProviderType = "llamacpp"
	ProviderTypeVLLM        ProviderType = "vllm"
	ProviderTypeLocalAI     ProviderType = "localai"
	ProviderTypeFastChat    ProviderType = "fastchat"
	ProviderTypeTextGen     ProviderType = "textgen"
	ProviderTypeLMStudio    ProviderType = "lmstudio"
	ProviderTypeJan         ProviderType = "jan"
	ProviderTypeKoboldAI    ProviderType = "koboldai"
	ProviderTypeGPT4All     ProviderType = "gpt4all"
	ProviderTypeTabbyAPI    ProviderType = "tabbyapi"
	ProviderTypeMLX         ProviderType = "mlx"
	ProviderTypeMistralRS   ProviderType = "mistralrs"
	ProviderTypeMemGPT      ProviderType = "memgpt"
	ProviderTypeCrewAI      ProviderType = "crewai"
	ProviderTypeCharacterAI ProviderType = "characterai"
	ProviderTypeReplika     ProviderType = "replika"
	ProviderTypeAnima       ProviderType = "anima"
	ProviderTypeGemma       ProviderType = "gemma"
	ProviderTypeLlamaIndex  ProviderType = "llamaindex"
	ProviderTypeCohere      ProviderType = "cohere"
	ProviderTypeHuggingFace ProviderType = "huggingface"
	ProviderTypeMistral     ProviderType = "mistral"
	ProviderTypeClickHouse  ProviderType = "clickhouse"
	ProviderTypeSupabase    ProviderType = "supabase"
	ProviderTypeDeepLake    ProviderType = "deeplake"
	ProviderTypeChroma      ProviderType = "chroma"
	ProviderTypeAgnostic    ProviderType = "agnostic"
)

type ModelInfo struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Provider       ProviderType      `json:"provider"`
	ContextSize    int               `json:"context_size"`
	MaxTokens      int               `json:"max_tokens"`
	Capabilities   []ModelCapability `json:"capabilities"`
	SupportsTools  bool              `json:"supports_tools"`
	SupportsVision bool              `json:"supports_vision"`
	Description    string            `json:"description"`
}

type ProviderConfigEntry struct {
	Type       string                 `json:"type"`
	Endpoint   string                 `json:"endpoint"`
	APIKey     string                 `json:"api_key"`
	Models     []string               `json:"models"`
	Enabled    bool                   `json:"enabled"`
	Parameters map[string]interface{} `json:"parameters"`
}

type ProviderHealth struct {
	Status     string        `json:"status"`
	LastCheck  time.Time     `json:"last_check"`
	Latency    time.Duration `json:"latency"`
	ModelCount int           `json:"model_count"`
	ErrorCount int           `json:"error_count"`
	Message    string        `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolCallFunc `json:"function"`
}

type ToolCallFunc struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type LLMRequest struct {
	ID               uuid.UUID              `json:"id"`
	Model            string                 `json:"model"`
	Messages         []Message              `json:"messages"`
	MaxTokens        int                    `json:"max_tokens"`
	Temperature      float64                `json:"temperature"`
	TopP             float64                `json:"top_p"`
	Stream           bool                   `json:"stream"`
	Tools            []Tool                 `json:"tools"`
	ToolChoice       interface{}            `json:"tool_choice"`
	Stop             []string               `json:"stop"`
	ThinkingBudget   int                    `json:"thinking_budget"`
	CacheConfig      *CacheConfig           `json:"cache_config"`
	Reasoning        *ReasoningConfig       `json:"reasoning"`
	ProviderMetadata map[string]interface{} `json:"provider_metadata"`
}

type LLMResponse struct {
	ID               uuid.UUID              `json:"id"`
	RequestID        uuid.UUID              `json:"request_id"`
	Content          string                 `json:"content"`
	ToolCalls        []ToolCall             `json:"tool_calls"`
	Usage            Usage                  `json:"usage"`
	FinishReason     string                 `json:"finish_reason"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	CreatedAt        time.Time              `json:"created_at"`
	ProviderMetadata map[string]interface{} `json:"provider_metadata"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Common errors
var (
	ErrModelNotFound       = fmt.Errorf("model not found")
	ErrInvalidRequest      = fmt.Errorf("invalid request")
	ErrRateLimited         = fmt.Errorf("rate limited")
	ErrContextTooLong      = fmt.Errorf("context too long")
	ErrProviderUnavailable = fmt.Errorf("provider unavailable")
)

// Provider interface
type Provider interface {
	GetType() ProviderType
	GetName() string
	GetModels() []ModelInfo
	GetCapabilities() []ModelCapability
	Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
	GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error
	IsAvailable(ctx context.Context) bool
	GetHealth(ctx context.Context) (*ProviderHealth, error)
	Close() error
}
