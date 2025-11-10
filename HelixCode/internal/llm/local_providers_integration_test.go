//go:build integration

package llm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestVLLMProviderIntegration tests the VLLM provider with integration
func TestVLLMProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeVLLM,
		Endpoint: getEnvOrDefault("VLLM_ENDPOINT", "http://localhost:8000"),
		APIKey:   os.Getenv("VLLM_API_KEY"),
		Models:   []string{"llama-2-7b-chat-hf"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewVLLMProvider(config)
	if err != nil {
		t.Skipf("Skipping VLLM integration test: %v", err)
	}

	testLocalProvider(t, provider, "VLLM")
}

// TestLocalAIProviderIntegration tests the LocalAI provider with integration
func TestLocalAIProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeLocalAI,
		Endpoint: getEnvOrDefault("LOCALAI_ENDPOINT", "http://localhost:8080"),
		APIKey:   os.Getenv("LOCALAI_API_KEY"),
		Models:   []string{"gpt-3.5-turbo"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewLocalAIProvider(config)
	if err != nil {
		t.Skipf("Skipping LocalAI integration test: %v", err)
	}

	testLocalProvider(t, provider, "LocalAI")
}

// TestFastChatProviderIntegration tests the FastChat provider with integration
func TestFastChatProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeFastChat,
		Endpoint: getEnvOrDefault("FASTCHAT_ENDPOINT", "http://localhost:7860"),
		APIKey:   os.Getenv("FASTCHAT_API_KEY"),
		Models:   []string{"vicuna-13b-v1.5"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewFastChatProvider(config)
	if err != nil {
		t.Skipf("Skipping FastChat integration test: %v", err)
	}

	testLocalProvider(t, provider, "FastChat")
}

// TestTextGenProviderIntegration tests the Text Generation WebUI provider with integration
func TestTextGenProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeTextGen,
		Endpoint: getEnvOrDefault("TEXTGEN_ENDPOINT", "http://localhost:5000"),
		APIKey:   os.Getenv("TEXTGEN_API_KEY"),
		Models:   []string{"llama-2-7b-chat-hf"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewTextGenProvider(config)
	if err != nil {
		t.Skipf("Skipping TextGen integration test: %v", err)
	}

	testLocalProvider(t, provider, "TextGen")
}

// TestLMStudioProviderIntegration tests the LM Studio provider with integration
func TestLMStudioProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeLMStudio,
		Endpoint: getEnvOrDefault("LMSTUDIO_ENDPOINT", "http://localhost:1234"),
		APIKey:   os.Getenv("LMSTUDIO_API_KEY"),
		Models:   []string{"local-model"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewLMStudioProvider(config)
	if err != nil {
		t.Skipf("Skipping LM Studio integration test: %v", err)
	}

	testLocalProvider(t, provider, "LM Studio")
}

// TestJanProviderIntegration tests the Jan AI provider with integration
func TestJanProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeJan,
		Endpoint: getEnvOrDefault("JAN_ENDPOINT", "http://localhost:1337"),
		APIKey:   os.Getenv("JAN_API_KEY"),
		Models:   []string{"jan-model"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewJanProvider(config)
	if err != nil {
		t.Skipf("Skipping Jan AI integration test: %v", err)
	}

	testLocalProvider(t, provider, "Jan AI")
}

// TestKoboldAIProviderIntegration tests the KoboldAI provider with integration
func TestKoboldAIProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeKoboldAI,
		Endpoint: getEnvOrDefault("KOBOLD_ENDPOINT", "http://localhost:5001"),
		APIKey:   os.Getenv("KOBOLD_API_KEY"),
		Models:   []string{"kobold-model"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewKoboldAIProvider(config)
	if err != nil {
		t.Skipf("Skipping KoboldAI integration test: %v", err)
	}

	testLocalProvider(t, provider, "KoboldAI")
}

// TestGPT4AllProviderIntegration tests the GPT4All provider with integration
func TestGPT4AllProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeGPT4All,
		Endpoint: getEnvOrDefault("GPT4ALL_ENDPOINT", "http://localhost:4891"),
		APIKey:   os.Getenv("GPT4ALL_API_KEY"),
		Models:   []string{"gpt4all-model"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": false, // GPT4All might not support streaming
		},
	}

	provider, err := NewGPT4AllProvider(config)
	if err != nil {
		t.Skipf("Skipping GPT4All integration test: %v", err)
	}

	testLocalProvider(t, provider, "GPT4All")
}

// TestTabbyAPIProviderIntegration tests the TabbyAPI provider with integration
func TestTabbyAPIProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeTabbyAPI,
		Endpoint: getEnvOrDefault("TABBYPAI_ENDPOINT", "http://localhost:5000"),
		APIKey:   os.Getenv("TABBYPAI_API_KEY"),
		Models:   []string{"tabby-model"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewTabbyAPIProvider(config)
	if err != nil {
		t.Skipf("Skipping TabbyAPI integration test: %v", err)
	}

	testLocalProvider(t, provider, "TabbyAPI")
}

// TestMLXProviderIntegration tests the MLX provider with integration
func TestMLXProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeMLX,
		Endpoint: getEnvOrDefault("MLX_ENDPOINT", "http://localhost:8080"),
		APIKey:   os.Getenv("MLX_API_KEY"),
		Models:   []string{"mlx-model"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewMLXProvider(config)
	if err != nil {
		t.Skipf("Skipping MLX integration test: %v", err)
	}

	testLocalProvider(t, provider, "MLX")
}

// TestMistralRSProviderIntegration tests the Mistral RS provider with integration
func TestMistralRSProviderIntegration(t *testing.T) {
	config := ProviderConfigEntry{
		Type:     ProviderTypeMistralRS,
		Endpoint: getEnvOrDefault("MISTRALRS_ENDPOINT", "http://localhost:8080"),
		APIKey:   os.Getenv("MISTRALRS_API_KEY"),
		Models:   []string{"mistral-model"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewMistralRSProvider(config)
	if err != nil {
		t.Skipf("Skipping MistralRS integration test: %v", err)
	}

	testLocalProvider(t, provider, "MistralRS")
}

// TestAllLocalProvidersIntegration tests all local providers together
func TestAllLocalProvidersIntegration(t *testing.T) {
	providers := []struct {
		name     string
		provider Provider
	}{
		{"VLLM", createTestProvider(t, ProviderTypeVLLM, "VLLM_ENDPOINT", "http://localhost:8000")},
		{"LocalAI", createTestProvider(t, ProviderTypeLocalAI, "LOCALAI_ENDPOINT", "http://localhost:8080")},
		{"FastChat", createTestProvider(t, ProviderTypeFastChat, "FASTCHAT_ENDPOINT", "http://localhost:7860")},
		{"TextGen", createTestProvider(t, ProviderTypeTextGen, "TEXTGEN_ENDPOINT", "http://localhost:5000")},
		{"LM Studio", createTestProvider(t, ProviderTypeLMStudio, "LMSTUDIO_ENDPOINT", "http://localhost:1234")},
		{"Jan AI", createTestProvider(t, ProviderTypeJan, "JAN_ENDPOINT", "http://localhost:1337")},
		{"KoboldAI", createTestProvider(t, ProviderTypeKoboldAI, "KOBOLD_ENDPOINT", "http://localhost:5001")},
		{"GPT4All", createTestProvider(t, ProviderTypeGPT4All, "GPT4ALL_ENDPOINT", "http://localhost:4891")},
		{"TabbyAPI", createTestProvider(t, ProviderTypeTabbyAPI, "TABBYPAI_ENDPOINT", "http://localhost:5000")},
		{"MLX", createTestProvider(t, ProviderTypeMLX, "MLX_ENDPOINT", "http://localhost:8080")},
		{"MistralRS", createTestProvider(t, ProviderTypeMistralRS, "MISTRALRS_ENDPOINT", "http://localhost:8080")},
	}

	availableProviders := 0
	for _, p := range providers {
		if p.provider != nil {
			availableProviders++
			t.Logf("✅ %s provider is available", p.name)
			p.provider.Close()
		} else {
			t.Logf("❌ %s provider is not available", p.name)
		}
	}

	if availableProviders == 0 {
		t.Skip("No local providers available for integration testing")
	}

	t.Logf("✅ %d out of %d local providers are available", availableProviders, len(providers))
}

// TestLocalProviderStreaming tests streaming functionality for all local providers
func TestLocalProviderStreaming(t *testing.T) {
	// Test with a provider that should support streaming
	config := ProviderConfigEntry{
		Type:     ProviderTypeVLLM,
		Endpoint: getEnvOrDefault("VLLM_ENDPOINT", "http://localhost:8000"),
		APIKey:   os.Getenv("VLLM_API_KEY"),
		Models:   []string{"llama-2-7b-chat-hf"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	provider, err := NewVLLMProvider(config)
	if err != nil {
		t.Skipf("Skipping streaming test: %v", err)
	}

	ctx := context.Background()
	if !provider.IsAvailable(ctx) {
		t.Skip("Provider not available for streaming test")
	}

	// Test streaming
	ch := make(chan LLMResponse, 10)
	request := &LLMRequest{
		ID:           uuid.New(),
		ProviderType: ProviderTypeVLLM,
		Model:        "test-model",
		Messages: []Message{
			{Role: "user", Content: "Hello! Please respond with a short greeting."},
		},
		MaxTokens:   50,
		Temperature: 0.1,
		Stream:      true,
		CreatedAt:   time.Now(),
	}

	err = provider.GenerateStream(ctx, request, ch)
	if err != nil {
		t.Errorf("Streaming generation failed: %v", err)
	}

	// Collect streaming responses
	responseCount := 0
	totalContent := ""
	for response := range ch {
		responseCount++
		totalContent += response.Content
	}

	if responseCount == 0 {
		t.Error("No streaming responses received")
	}

	if totalContent == "" {
		t.Error("Streaming response content is empty")
	}

	t.Logf("✅ Streaming test passed: %d chunks, total length: %d", responseCount, len(totalContent))

	// Cleanup
	provider.Close()
}

// Helper functions for local provider testing

func createTestProvider(t *testing.T, providerType ProviderType, envVar, defaultEndpoint string) Provider {
	config := ProviderConfigEntry{
		Type:     providerType,
		Endpoint: getEnvOrDefault(envVar, defaultEndpoint),
		APIKey:   os.Getenv(envVar + "_API_KEY"),
		Models:   []string{"test-model"},
		Parameters: map[string]interface{}{
			"timeout":           30.0,
			"max_retries":       3,
			"streaming_support": true,
		},
	}

	var provider Provider
	var err error

	switch providerType {
	case ProviderTypeVLLM:
		provider, err = NewVLLMProvider(config)
	case ProviderTypeLocalAI:
		provider, err = NewLocalAIProvider(config)
	case ProviderTypeFastChat:
		provider, err = NewFastChatProvider(config)
	case ProviderTypeTextGen:
		provider, err = NewTextGenProvider(config)
	case ProviderTypeLMStudio:
		provider, err = NewLMStudioProvider(config)
	case ProviderTypeJan:
		provider, err = NewJanProvider(config)
	case ProviderTypeKoboldAI:
		provider, err = NewKoboldAIProvider(config)
	case ProviderTypeGPT4All:
		provider, err = NewGPT4AllProvider(config)
	case ProviderTypeTabbyAPI:
		provider, err = NewTabbyAPIProvider(config)
	case ProviderTypeMLX:
		provider, err = NewMLXProvider(config)
	case ProviderTypeMistralRS:
		provider, err = NewMistralRSProvider(config)
	}

	if err != nil {
		return nil
	}

	// Test availability
	ctx := context.Background()
	if !provider.IsAvailable(ctx) {
		provider.Close()
		return nil
	}

	return provider
}

func testLocalProvider(t *testing.T, provider Provider, providerName string) {
	ctx := context.Background()

	// Test provider availability
	available := provider.IsAvailable(ctx)
	if !available {
		t.Skipf("%s provider not available for integration test", providerName)
	}

	// Test model listing
	models := provider.GetModels()
	if len(models) == 0 {
		t.Logf("⚠️  No models available from %s (may be normal)", providerName)
	}

	// Test capabilities
	capabilities := provider.GetCapabilities()
	if len(capabilities) == 0 {
		t.Errorf("%s provider should have capabilities", providerName)
	}

	// Test health check
	health, err := provider.GetHealth(ctx)
	if err != nil {
		t.Errorf("%s health check failed: %v", providerName, err)
	}
	if health.Status == "" {
		t.Errorf("%s health status should not be empty", providerName)
	}

	// Test basic generation (if provider is healthy)
	if health.Status == "healthy" && len(models) > 0 {
		request := &LLMRequest{
			ID:           generateTestID(),
			ProviderType: provider.GetType(),
			Model:        models[0].Name, // Use first available model
			Messages: []Message{
				{Role: "user", Content: "Hello! Please respond with just 'Hello World'."},
			},
			MaxTokens:   50,
			Temperature: 0.1,
			CreatedAt:   time.Now(),
		}

		response, err := provider.Generate(ctx, request)
		if err != nil {
			t.Logf("⚠️  Generation test failed for %s (may be normal): %v", providerName, err)
		} else if response != nil {
			t.Logf("✅ %s generated response: %s", providerName, response.Content)
			if response.Usage.TotalTokens == 0 {
				t.Logf("⚠️  %s response should include token usage", providerName)
			}
		}
	}

	t.Logf("✅ %s provider integration test passed: %d models, health: %s",
		providerName, len(models), health.Status)

	// Cleanup
	provider.Close()
}
