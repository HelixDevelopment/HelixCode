package llm

import (
	"time"
)

// Provider constructors for specific local LLM services

// NewVLLMProvider creates a new VLLM provider
func NewVLLMProvider(config ProviderConfigEntry) (Provider, error) {
	vllmConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:8000"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "llama-2-7b-chat-hf"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("vllm", vllmConfig)
}

// NewLocalAIProvider creates a new LocalAI provider
func NewLocalAIProvider(config ProviderConfigEntry) (Provider, error) {
	localAIConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:8080"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "gpt-3.5-turbo"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("localai", localAIConfig)
}

// NewFastChatProvider creates a new FastChat provider
func NewFastChatProvider(config ProviderConfigEntry) (Provider, error) {
	fastChatConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:7860"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "vicuna-13b-v1.5"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("fastchat", fastChatConfig)
}

// NewTextGenProvider creates a new Text Generation WebUI provider
func NewTextGenProvider(config ProviderConfigEntry) (Provider, error) {
	textGenConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:5000"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "llama-2-7b-chat-hf"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("textgen", textGenConfig)
}

// NewLMStudioProvider creates a new LM Studio provider
func NewLMStudioProvider(config ProviderConfigEntry) (Provider, error) {
	lmStudioConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:1234"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "local-model"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("lmstudio", lmStudioConfig)
}

// NewJanProvider creates a new Jan AI provider
func NewJanProvider(config ProviderConfigEntry) (Provider, error) {
	janConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:1337"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "jan-model"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("jan", janConfig)
}

// NewGPT4AllProvider creates a new GPT4All provider
func NewGPT4AllProvider(config ProviderConfigEntry) (Provider, error) {
	gpt4AllConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:4891"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "gpt4all-model"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", false), // GPT4All might not support streaming
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("gpt4all", gpt4AllConfig)
}

// NewTabbyAPIProvider creates a new TabbyAPI provider
func NewTabbyAPIProvider(config ProviderConfigEntry) (Provider, error) {
	tabbyAPIConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:5000"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "tabby-model"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("tabbyapi", tabbyAPIConfig)
}

// NewMLXProvider creates a new MLX LLM provider
func NewMLXProvider(config ProviderConfigEntry) (Provider, error) {
	mlxConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:8080"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "mlx-model"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("mlx", mlxConfig)
}

// NewMistralRSProvider creates a new Mistral RS provider
func NewMistralRSProvider(config ProviderConfigEntry) (Provider, error) {
	mistralRSConfig := OpenAICompatibleConfig{
		BaseURL:          getEndpoint(config.Endpoint, "http://localhost:8080"),
		APIKey:           config.APIKey,
		DefaultModel:     getFirstModel(config.Models, "mistral-model"),
		Timeout:          getTimeout(config.Parameters, 30*time.Second),
		MaxRetries:       getIntParam(config.Parameters, "max_retries", 3),
		Headers:          getStringMapParam(config.Parameters, "headers"),
		StreamingSupport: getBoolParam(config.Parameters, "streaming_support", true),
		ModelEndpoint:    "/v1/models",
		ChatEndpoint:     "/v1/chat/completions",
	}

	return NewOpenAICompatibleProvider("mistralrs", mistralRSConfig)
}

// Helper functions for extracting configuration parameters

func getEndpoint(configEndpoint, defaultEndpoint string) string {
	if configEndpoint != "" {
		return configEndpoint
	}
	return defaultEndpoint
}

func getFirstModel(models []string, defaultModel string) string {
	if len(models) > 0 {
		return models[0]
	}
	return defaultModel
}

func getTimeout(params map[string]interface{}, defaultTimeout time.Duration) time.Duration {
	if timeout, ok := params["timeout"].(float64); ok {
		return time.Duration(timeout) * time.Second
	}
	if timeout, ok := params["timeout"].(int); ok {
		return time.Duration(timeout) * time.Second
	}
	return defaultTimeout
}

func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(float64); ok {
		return int(value)
	}
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := params[key].(bool); ok {
		return value
	}
	return defaultValue
}

func getStringMapParam(params map[string]interface{}, key string) map[string]string {
	if value, ok := params[key].(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range value {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
		return result
	}
	return make(map[string]string)
}
