package llm

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

// LlamaCPPProvider implements the LLM provider interface for Llama.cpp
type LlamaCPPProvider struct {
	config    LlamaConfig
	isRunning bool
}

// LlamaConfig holds configuration for Llama.cpp
type LlamaConfig struct {
	ModelPath     string        `json:"model_path"`
	ContextSize   int           `json:"context_size"`
	GPUEnabled    bool          `json:"gpu_enabled"`
	GPULayers     int           `json:"gpu_layers"`
	Threads       int           `json:"threads"`
	ServerHost    string        `json:"server_host"`
	ServerPort    int           `json:"server_port"`
	ServerTimeout time.Duration `json:"server_timeout"`
}

// NewLlamaCPPProvider creates a new Llama.cpp provider
func NewLlamaCPPProvider(config LlamaConfig) (*LlamaCPPProvider, error) {
	provider := &LlamaCPPProvider{
		config:    config,
		isRunning: true,
	}

	log.Printf("✅ Llama.cpp provider initialized with model: %s", config.ModelPath)
	return provider, nil
}

// GetType returns the provider type
func (p *LlamaCPPProvider) GetType() ProviderType {
	return ProviderTypeLocal
}

// GetName returns the provider name
func (p *LlamaCPPProvider) GetName() string {
	return "llama-cpp"
}

// GetModels returns available models
func (p *LlamaCPPProvider) GetModels() []ModelInfo {
	return []ModelInfo{
		{
			Name:           p.config.ModelPath,
			Provider:       ProviderTypeLocal,
			ContextSize:    p.config.ContextSize,
			Capabilities:   []ModelCapability{CapabilityTextGeneration, CapabilityCodeGeneration, CapabilityCodeAnalysis},
			MaxTokens:      p.config.ContextSize,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Local Llama.cpp model",
		},
	}
}

// GetCapabilities returns model capabilities
func (p *LlamaCPPProvider) GetCapabilities() []ModelCapability {
	return []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
	}
}

// Generate generates a response using Llama.cpp
func (p *LlamaCPPProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	if !p.isRunning {
		return nil, ErrProviderUnavailable
	}

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	return &LLMResponse{
		ID:        uuid.New(),
		RequestID: request.ID,
		Content:   "This is a simulated response from Llama.cpp provider",
		Usage: Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		ProcessingTime: 100 * time.Millisecond,
		CreatedAt:      time.Now(),
	}, nil
}

// GenerateStream generates a streaming response
func (p *LlamaCPPProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	if !p.isRunning {
		return ErrProviderUnavailable
	}

	// Simulate streaming response
	chunks := []string{"This", " is", " a", " streaming", " response"}
	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- LLMResponse{
			ID:        uuid.New(),
			RequestID: request.ID,
			Content:   chunk,
			CreatedAt: time.Now(),
		}:
			time.Sleep(50 * time.Millisecond)
		}
	}

	return nil
}

// IsAvailable checks if the provider is available
func (p *LlamaCPPProvider) IsAvailable(ctx context.Context) bool {
	return p.isRunning
}

// GetHealth returns provider health status
func (p *LlamaCPPProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	if !p.isRunning {
		return &ProviderHealth{
			Status:     "unhealthy",
			LastCheck:  time.Now(),
			ErrorCount: 1,
		}, nil
	}

	return &ProviderHealth{
		Status:     "healthy",
		LastCheck:  time.Now(),
		ErrorCount: 0,
		ModelCount: len(p.GetModels()),
	}, nil
}

// Close stops the Llama.cpp provider
func (p *LlamaCPPProvider) Close() error {
	p.isRunning = false
	log.Println("✅ Llama.cpp provider closed")
	return nil
}
