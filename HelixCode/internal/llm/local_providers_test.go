package llm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewOpenAICompatibleProvider tests the creation of OpenAI-compatible providers
func TestNewOpenAICompatibleProvider(t *testing.T) {
	config := OpenAICompatibleConfig{
		BaseURL:         "http://localhost:8000",
		APIKey:          "test-key",
		DefaultModel:    "test-model",
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		StreamingSupport: true,
	}

	provider, err := NewOpenAICompatibleProvider("test-provider", config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, "test-provider", provider.GetName())
	assert.Equal(t, ProviderTypeLocal, provider.GetType()) // Falls back to local for unknown names
	assert.Equal(t, config.Timeout, provider.httpClient.Timeout)
	assert.True(t, provider.isRunning)

	// Test model capabilities
	capabilities := provider.GetCapabilities()
	assert.Contains(t, capabilities, CapabilityTextGeneration)
	assert.Contains(t, capabilities, CapabilityCodeGeneration)

	// Cleanup
	err = provider.Close()
	assert.NoError(t, err)
	assert.False(t, provider.isRunning)
}

// TestOpenAICompatibleProviderConfigDefaults tests configuration defaults
func TestOpenAICompatibleProviderConfigDefaults(t *testing.T) {
	config := OpenAICompatibleConfig{
		BaseURL:      "http://localhost:8000",
		DefaultModel: "test-model",
		Timeout:      30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("test", config)
	require.NoError(t, err)
	
	// Should set default endpoints
	assert.Equal(t, "/v1/models", config.ModelEndpoint)
	assert.Equal(t, "/v1/chat/completions", config.ChatEndpoint)
	
	provider.Close()
}

// TestProviderTypeMapping tests provider type mapping for different names
func TestProviderTypeMapping(t *testing.T) {
	tests := []struct {
		name     string
		expected ProviderType
	}{
		{"vllm", ProviderTypeVLLM},
		{"localai", ProviderTypeLocalAI},
		{"fastchat", ProviderTypeFastChat},
		{"textgen", ProviderTypeTextGen},
		{"lmstudio", ProviderTypeLMStudio},
		{"jan", ProviderTypeJan},
		{"koboldai", ProviderTypeKoboldAI},
		{"gpt4all", ProviderTypeGPT4All},
		{"tabbyapi", ProviderTypeTabbyAPI},
		{"mlx", ProviderTypeMLX},
		{"mistralrs", ProviderTypeMistralRS},
		{"unknown", ProviderTypeLocal}, // Falls back to local
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := OpenAICompatibleConfig{
				BaseURL:      "http://localhost:8000",
				DefaultModel: "test-model",
				Timeout:      30 * time.Second,
			}

			provider, err := NewOpenAICompatibleProvider(tt.name, config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, provider.GetType())
			
			provider.Close()
		})
	}
}

// TestContextSizeInference tests context size inference from model names
func TestContextSizeInference(t *testing.T) {
	tests := []struct {
		modelName     string
		expectedSize int
	}{
		{"llama-2-7b-32k", 32768},
		{"gpt-4-16k", 16384},
		{"claude-3-8k", 8192},
		{"gpt-4", 8192},
		{"claude-3-opus", 100000},
		{"llama-2-7b", 4096},
		{"unknown-model", 4096},
	}

	config := OpenAICompatibleConfig{
		BaseURL:      "http://localhost:8000",
		DefaultModel: "test-model",
		Timeout:      30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("test", config)
	require.NoError(t, err)
	defer provider.Close()

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			contextSize := provider.inferContextSize(tt.modelName)
			assert.Equal(t, tt.expectedSize, contextSize)
		})
	}
}

// TestVisionModelDetection tests vision model detection
func TestVisionModelDetection(t *testing.T) {
	tests := []struct {
		modelName string
		expected bool
	}{
		{"gpt-4-vision-preview", true},
		{"claude-3-vision", true},
		{"llava-1.5", true},
		{"multimodal-model", true},
		{"clip-model", true},
		{"gpt-4", false},
		{"claude-3-opus", false},
		{"llama-2-7b", false},
	}

	config := OpenAICompatibleConfig{
		BaseURL:      "http://localhost:8000",
		DefaultModel: "test-model",
		Timeout:      30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("test", config)
	require.NoError(t, err)
	defer provider.Close()

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			supportsVision := provider.supportsVisionModel(tt.modelName)
			assert.Equal(t, tt.expected, supportsVision)
		})
	}
}

// TestToolSupportDetection tests tool support detection
func TestToolSupportDetection(t *testing.T) {
	tests := []struct {
		modelName string
		expected bool
	}{
		{"gpt-4-turbo", true},
		{"gpt-4", true},
		{"claude-3-opus", true},
		{"claude-3-sonnet", true},
		{"llama-3-8b-instruct", true},
		{"llama-2-7b-chat", false},
		{"mistral-7b", true},
		{"vicuna-13b", false},
	}

	config := OpenAICompatibleConfig{
		BaseURL:      "http://localhost:8000",
		DefaultModel: "test-model",
		Timeout:      30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("test", config)
	require.NoError(t, err)
	defer provider.Close()

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			supportsTools := provider.supportsTools(tt.modelName)
			assert.Equal(t, tt.expected, supportsTools)
		})
	}
}

// TestGetModelName tests model name resolution
func TestGetModelName(t *testing.T) {
	tests := []struct {
		requestedModel    string
		defaultModel     string
		availableModels   []ModelInfo
		expectedResult   string
	}{
		{"", "default-model", []ModelInfo{}, "default-model"},
		{"", "", []ModelInfo{{Name: "first-model"}}, "first-model"},
		{"", "", []ModelInfo{}, "gpt-3.5-turbo"}, // fallback
		{"requested-model", "default-model", []ModelInfo{}, "requested-model"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			config := OpenAICompatibleConfig{
				BaseURL:      "http://localhost:8000",
				DefaultModel: tt.defaultModel,
				Timeout:      30 * time.Second,
			}

			provider, err := NewOpenAICompatibleProvider("test", config)
			require.NoError(t, err)
			
			// Set models manually for testing
			provider.models = tt.availableModels
			
			result := provider.getModelName(tt.requestedModel)
			assert.Equal(t, tt.expectedResult, result)
			
			provider.Close()
		})
	}
}

// TestAPIURLGeneration tests API URL generation for different providers
func TestAPIURLGeneration(t *testing.T) {
	tests := []struct {
		name           string
		baseURL        string
		endpoint       string
		expectedResult string
	}{
		{"vllm", "http://localhost:8000", "/v1/models", "http://localhost:8000/v1/models"},
		{"vllm", "http://localhost:8000/", "/v1/models", "http://localhost:8000/v1/models"},
		{"textgen", "", "/v1/chat/completions", "http://localhost:5000/v1/chat/completions"},
		{"lmstudio", "http://localhost:1234", "/v1/models", "http://localhost:1234/v1/models"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := OpenAICompatibleConfig{
				BaseURL:   tt.baseURL,
				Timeout:   30 * time.Second,
			}

			provider, err := NewOpenAICompatibleProvider(tt.name, config)
			require.NoError(t, err)
			
			result := provider.getAPIURL(tt.endpoint)
			assert.Equal(t, tt.expectedResult, result)
			
			provider.Close()
		})
	}
}

// TestHealthStatusUpdate tests health status updating
func TestHealthStatusUpdate(t *testing.T) {
	config := OpenAICompatibleConfig{
		BaseURL: "http://localhost:8000",
		Timeout: 30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("test", config)
	require.NoError(t, err)
	defer provider.Close()

	// Test initial health status
	assert.Equal(t, "unknown", provider.lastHealth.Status)
	assert.Equal(t, 0, provider.lastHealth.ErrorCount)

	// Update health status
	provider.updateHealth("healthy", 100*time.Millisecond, 0)
	assert.Equal(t, "healthy", provider.lastHealth.Status)
	assert.Equal(t, 100*time.Millisecond, provider.lastHealth.Latency)
	assert.Equal(t, 0, provider.lastHealth.ErrorCount)

	// Update with error
	provider.updateHealth("unhealthy", 200*time.Millisecond, 1)
	assert.Equal(t, "unhealthy", provider.lastHealth.Status)
	assert.Equal(t, 200*time.Millisecond, provider.lastHealth.Latency)
	assert.Equal(t, 1, provider.lastHealth.ErrorCount)
}

// TestConvertToOpenAIRequest tests request conversion
func TestConvertToOpenAIRequest(t *testing.T) {
	config := OpenAICompatibleConfig{
		BaseURL:      "http://localhost:8000",
		DefaultModel: "test-model",
		Timeout:      30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("test", config)
	require.NoError(t, err)
	defer provider.Close()

	request := &LLMRequest{
		Model:       "specific-model",
		Messages:    []Message{{Role: "user", Content: "Hello"}},
		MaxTokens:   100,
		Temperature: 0.5,
		TopP:        0.9,
		Stream:      true,
		Tools:       []Tool{{Type: "function"}},
		ToolChoice:  "auto",
	}

	apiRequest := provider.convertToOpenAIRequest(request)

	assert.Equal(t, "specific-model", apiRequest.Model)
	assert.Equal(t, request.Messages, apiRequest.Messages)
	assert.Equal(t, 100, apiRequest.MaxTokens)
	assert.Equal(t, 0.5, apiRequest.Temperature)
	assert.Equal(t, 0.9, apiRequest.TopP)
	assert.True(t, apiRequest.Stream)
	assert.Equal(t, request.Tools, apiRequest.Tools)
	assert.Equal(t, "auto", apiRequest.ToolChoice)
}

// TestConvertFromOpenAIResponse tests response conversion
func TestConvertFromOpenAIResponse(t *testing.T) {
	config := OpenAICompatibleConfig{
		BaseURL: "http://localhost:8000",
		Timeout: 30 * time.Second,
	}

	provider, err := NewOpenAICompatibleProvider("test", config)
	require.NoError(t, err)
	defer provider.Close()

	response := &OpenAICompatibleResponse{
		ID:      "resp-id",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test-model",
		Choices: []OpenAICompatibleChoice{
			{
				Index: 0,
				Message: OpenAICompatibleMessage{
					Role:    "assistant",
					Content: "Hello world",
					ToolCalls: []ToolCall{
						{
							ID:   "tool-call-id",
							Type: "function",
							Function: ToolCallFunction{
								Name:      "test_function",
								Arguments: map[string]interface{}{"arg1": "value1"},
							},
						},
					},
				},
				FinishReason: "stop",
			},
		},
		Usage: OpenAICompatibleUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	llmResponse := provider.convertFromOpenAIResponse(response, uuid.New(), 100*time.Millisecond)

	assert.NotEqual(t, uuid.Nil, llmResponse.ID)
	assert.Equal(t, "Hello world", llmResponse.Content)
	assert.Equal(t, "stop", llmResponse.FinishReason)
	assert.Len(t, llmResponse.ToolCalls, 1)
	assert.Equal(t, "tool-call-id", llmResponse.ToolCalls[0].ID)
	assert.Equal(t, 10, llmResponse.Usage.PromptTokens)
	assert.Equal(t, 5, llmResponse.Usage.CompletionTokens)
	assert.Equal(t, 15, llmResponse.Usage.TotalTokens)
	assert.Equal(t, 100*time.Millisecond, llmResponse.ProcessingTime)
}

// TestKoboldAIProviderCreation tests KoboldAI provider creation
func TestKoboldAIProviderCreation(t *testing.T) {
	config := KoboldAIConfig{
		BaseURL:         "http://localhost:5001",
		APIKey:          "test-key",
		DefaultModel:    "kobold-model",
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		StreamingSupport: true,
	}

	provider, err := NewKoboldAIProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, "koboldai", provider.GetName())
	assert.Equal(t, ProviderTypeKoboldAI, provider.GetType())
	assert.Equal(t, config.Timeout, provider.httpClient.Timeout)
	assert.True(t, provider.isRunning)

	// Test model capabilities
	capabilities := provider.GetCapabilities()
	assert.Contains(t, capabilities, CapabilityTextGeneration)
	assert.Contains(t, capabilities, CapabilityCodeGeneration)

	// Cleanup
	err = provider.Close()
	assert.NoError(t, err)
	assert.False(t, provider.isRunning)
}

// TestKoboldAIMessageConversion tests message to prompt conversion for KoboldAI
func TestKoboldAIMessageConversion(t *testing.T) {
	config := KoboldAIConfig{
		BaseURL: "http://localhost:5001",
		Timeout: 30 * time.Second,
	}

	provider, err := NewKoboldAIProvider(config)
	require.NoError(t, err)
	defer provider.Close()

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	prompt := provider.convertMessagesToPrompt(messages)
	
	expected := `System: You are a helpful assistant

User: Hello

Assistant: Hi there!

User: How are you?

Assistant: `
	
	assert.Equal(t, expected, prompt)
}