package llm

import (
	"fmt"
	"time"
)

// NewProvider creates a new provider instance based on the configuration
func NewProvider(config ProviderConfigEntry) (Provider, error) {
	switch config.Type {
	case ProviderTypeOpenAI:
		return NewOpenAIProvider(config)
	case ProviderTypeAnthropic:
		return NewAnthropicProvider(config)
	case ProviderTypeGemini:
		return NewGeminiProvider(config)
	case ProviderTypeOllama:
		ollamaConfig := OllamaConfig{
			BaseURL:       config.Endpoint,
			DefaultModel:  "llama2", // Default
			Timeout:       120 * time.Second,
			StreamEnabled: true,
		}
		if len(config.Models) > 0 {
			ollamaConfig.DefaultModel = config.Models[0]
		}
		// Map parameters if available
		if val, ok := config.Parameters["timeout"].(float64); ok {
			ollamaConfig.Timeout = time.Duration(val) * time.Second
		}
		return NewOllamaProvider(ollamaConfig)
	case ProviderTypeLlamaCpp:
		llamaConfig := LlamaConfig{
			Model:         "", // Needs to be set from config
			ContextSize:   4096,
			ServerTimeout: 120 * time.Second,
		}
		if len(config.Models) > 0 {
			llamaConfig.Model = config.Models[0]
		}
		// Map parameters
		if val, ok := config.Parameters["context_size"].(float64); ok {
			llamaConfig.ContextSize = int(val)
		}
		if val, ok := config.Parameters["gpu_enabled"].(bool); ok {
			llamaConfig.GPUEnabled = val
		}
		return NewLlamaCPPProvider(llamaConfig)
	case ProviderTypeQwen:
		return NewQwenProvider(config)
	case ProviderTypeXAI:
		return NewXAIProvider(config)
	case ProviderTypeOpenRouter:
		return NewOpenRouterProvider(config)
	case ProviderTypeCopilot:
		return NewCopilotProvider(config)
	case ProviderTypeAzure:
		return NewAzureProvider(config)
	case ProviderTypeBedrock:
		return NewBedrockProvider(config)
	case ProviderTypeVertexAI:
		return NewVertexAIProvider(config)
	case ProviderTypeGroq:
		return NewGroqProvider(config)
	case ProviderTypeVLLM:
		return newOpenAICompatibleFromConfig("vllm", config)
	case ProviderTypeLocalAI:
		return newOpenAICompatibleFromConfig("localai", config)
	case ProviderTypeFastChat:
		return newOpenAICompatibleFromConfig("fastchat", config)
	case ProviderTypeTextGen:
		return newOpenAICompatibleFromConfig("textgen", config)
	case ProviderTypeLMStudio:
		return newOpenAICompatibleFromConfig("lmstudio", config)
	case ProviderTypeJan:
		return newOpenAICompatibleFromConfig("jan", config)
	case ProviderTypeGPT4All:
		return newOpenAICompatibleFromConfig("gpt4all", config)
	case ProviderTypeTabbyAPI:
		return newOpenAICompatibleFromConfig("tabbyapi", config)
	case ProviderTypeMLX:
		return newOpenAICompatibleFromConfig("mlx", config)
	case ProviderTypeMistralRS:
		return newOpenAICompatibleFromConfig("mistralrs", config)
	case ProviderTypeKoboldAI:
		koboldConfig := KoboldAIConfig{
			BaseURL: config.Endpoint,
			APIKey:  config.APIKey,
			Timeout: 120 * time.Second,
		}
		if len(config.Models) > 0 {
			koboldConfig.DefaultModel = config.Models[0]
		}
		if val, ok := config.Parameters["timeout"].(float64); ok {
			koboldConfig.Timeout = time.Duration(val) * time.Second
		}
		return NewKoboldAIProvider(koboldConfig)
	case ProviderTypeXiaomi:
		return NewXiaomiProvider(config)
	case ProviderTypeReplicate:
		return NewReplicateProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// newOpenAICompatibleFromConfig creates an OpenAI-compatible provider from a generic config entry.
// This is used for local providers (VLLM, LocalAI, LMStudio, etc.) that implement the OpenAI API spec.
func newOpenAICompatibleFromConfig(name string, config ProviderConfigEntry) (Provider, error) {
	cfg := OpenAICompatibleConfig{
		BaseURL:          config.Endpoint,
		APIKey:           config.APIKey,
		DefaultModel:     "",
		Timeout:          120 * time.Second,
		MaxRetries:       3,
		StreamingSupport: true,
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}
	if len(config.Models) > 0 {
		cfg.DefaultModel = config.Models[0]
	}
	if val, ok := config.Parameters["timeout"].(float64); ok {
		cfg.Timeout = time.Duration(val) * time.Second
	}
	if val, ok := config.Parameters["streaming_support"].(bool); ok {
		cfg.StreamingSupport = val
	}
	if val, ok := config.Parameters["model_endpoint"].(string); ok {
		cfg.ModelEndpoint = val
	}
	if val, ok := config.Parameters["chat_endpoint"].(string); ok {
		cfg.ChatEndpoint = val
	}
	return NewOpenAICompatibleProvider(name, cfg)
}

// InitializeModelManager initializes a ModelManager with providers from configuration
func InitializeModelManager(configs []ProviderConfigEntry) (*ModelManager, error) {
	manager := NewModelManager()

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		provider, err := NewProvider(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", config.Type, err)
		}

		if err := manager.RegisterProvider(provider); err != nil {
			return nil, fmt.Errorf("failed to register provider %s: %w", config.Type, err)
		}
	}

	return manager, nil
}
