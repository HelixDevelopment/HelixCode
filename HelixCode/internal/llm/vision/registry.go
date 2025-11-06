package vision

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Model represents a language model with its capabilities
type Model struct {
	ID           string
	Name         string
	Provider     string
	Capabilities *Capabilities
	Metadata     *ModelMetadata
}

// Capabilities defines model capabilities
type Capabilities struct {
	SupportsVision   bool
	SupportsAudio    bool
	SupportsVideo    bool
	MaxImageSize     int64
	MaxImages        int
	SupportedFormats []string
	ContextWindow    int
	OutputTokens     int
	FunctionCalling  bool
	StreamingSupport bool
}

// ModelMetadata contains additional model information
type ModelMetadata struct {
	Version      string
	Released     time.Time
	Deprecated   bool
	ReplacementID string
	Pricing      *Pricing
	RateLimits   *RateLimits
}

// Pricing contains model pricing information
type Pricing struct {
	InputCost  float64 // per 1M tokens
	OutputCost float64 // per 1M tokens
	ImageCost  float64 // per image (if applicable)
}

// RateLimits defines rate limiting constraints
type RateLimits struct {
	RequestsPerMinute int
	TokensPerMinute   int
	ImagesPerMinute   int
}

// VisionCapabilities contains vision-specific capabilities
type VisionCapabilities struct {
	MaxImageSize     int64
	MaxImages        int
	SupportedFormats []string
	MaxResolution    *Dimensions
	DetailLevels     []string // low, high, auto
}

// ModelPreferences defines model selection preferences
type ModelPreferences struct {
	Provider         string
	MinContextWindow int
	MaxCost          float64
	RequireStreaming bool
	PreferredModels  []string
}

// ModelRegistry maintains a registry of available models
type ModelRegistry struct {
	models map[string]*Model
	mu     sync.RWMutex
}

// ModelFilter filters model queries
type ModelFilter struct {
	Provider       string
	SupportsVision bool
	SupportsAudio  bool
	MinContext     int
	MaxCost        float64
}

// ModelUpdate contains fields to update
type ModelUpdate struct {
	Capabilities *Capabilities
	Metadata     *ModelMetadata
	Deprecated   *bool
}

// NewModelRegistry creates a new model registry with default vision models
func NewModelRegistry() *ModelRegistry {
	registry := &ModelRegistry{
		models: make(map[string]*Model),
	}

	// Register default vision-capable models
	for _, model := range DefaultVisionModels {
		registry.Register(model)
	}

	return registry
}

// Register registers a model
func (r *ModelRegistry) Register(model *Model) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if model.ID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}

	r.models[model.ID] = model
	return nil
}

// Get retrieves a model by ID
func (r *ModelRegistry) Get(modelID string) (*Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, exists := r.models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	return model, nil
}

// List returns all registered models matching the filter
func (r *ModelRegistry) List(filter *ModelFilter) ([]*Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []*Model
	for _, model := range r.models {
		if filter == nil || r.matchesFilter(model, filter) {
			models = append(models, model)
		}
	}

	return models, nil
}

// FindVisionModels returns all vision-capable models
func (r *ModelRegistry) FindVisionModels() ([]*Model, error) {
	filter := &ModelFilter{
		SupportsVision: true,
	}
	return r.List(filter)
}

// GetDefaultVisionModel returns the default vision model
func (r *ModelRegistry) GetDefaultVisionModel() (*Model, error) {
	// Try to get Claude 3.5 Sonnet as default
	model, err := r.Get("claude-3-5-sonnet-20241022")
	if err == nil {
		return model, nil
	}

	// Fallback to any vision model
	visionModels, err := r.FindVisionModels()
	if err != nil || len(visionModels) == 0 {
		return nil, fmt.Errorf("no vision-capable models available")
	}

	return visionModels[0], nil
}

// Update updates model information
func (r *ModelRegistry) Update(modelID string, updates *ModelUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	model, exists := r.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	if updates.Capabilities != nil {
		model.Capabilities = updates.Capabilities
	}

	if updates.Metadata != nil {
		model.Metadata = updates.Metadata
	}

	if updates.Deprecated != nil {
		if model.Metadata == nil {
			model.Metadata = &ModelMetadata{}
		}
		model.Metadata.Deprecated = *updates.Deprecated
	}

	return nil
}

// matchesFilter checks if a model matches the filter
func (r *ModelRegistry) matchesFilter(model *Model, filter *ModelFilter) bool {
	if filter.Provider != "" && !strings.EqualFold(model.Provider, filter.Provider) {
		return false
	}

	if filter.SupportsVision && !model.Capabilities.SupportsVision {
		return false
	}

	if filter.SupportsAudio && !model.Capabilities.SupportsAudio {
		return false
	}

	if filter.MinContext > 0 && model.Capabilities.ContextWindow < filter.MinContext {
		return false
	}

	if filter.MaxCost > 0 && model.Metadata != nil && model.Metadata.Pricing != nil {
		if model.Metadata.Pricing.InputCost > filter.MaxCost {
			return false
		}
	}

	return true
}

// CapabilityChecker checks model capabilities
type CapabilityChecker struct {
	registry *ModelRegistry
}

// NewCapabilityChecker creates a capability checker
func NewCapabilityChecker(registry *ModelRegistry) *CapabilityChecker {
	return &CapabilityChecker{
		registry: registry,
	}
}

// SupportsVision checks if a model supports vision
func (c *CapabilityChecker) SupportsVision(ctx context.Context, modelID string) (bool, error) {
	model, err := c.registry.Get(modelID)
	if err != nil {
		return false, err
	}

	return model.Capabilities.SupportsVision, nil
}

// GetVisionCapabilities returns vision-specific capabilities
func (c *CapabilityChecker) GetVisionCapabilities(modelID string) (*VisionCapabilities, error) {
	model, err := c.registry.Get(modelID)
	if err != nil {
		return nil, err
	}

	if !model.Capabilities.SupportsVision {
		return nil, fmt.Errorf("model %s does not support vision", modelID)
	}

	return &VisionCapabilities{
		MaxImageSize:     model.Capabilities.MaxImageSize,
		MaxImages:        model.Capabilities.MaxImages,
		SupportedFormats: model.Capabilities.SupportedFormats,
		DetailLevels:     []string{"low", "high", "auto"},
	}, nil
}

// FindBestVisionModel finds the best vision-capable model based on preferences
func (c *CapabilityChecker) FindBestVisionModel(ctx context.Context, preferences *ModelPreferences) (*Model, error) {
	visionModels, err := c.registry.FindVisionModels()
	if err != nil {
		return nil, err
	}

	if len(visionModels) == 0 {
		return nil, fmt.Errorf("no vision-capable models available")
	}

	// If preferred models specified, try them first
	if preferences != nil && len(preferences.PreferredModels) > 0 {
		for _, preferredID := range preferences.PreferredModels {
			for _, model := range visionModels {
				if model.ID == preferredID {
					return model, nil
				}
			}
		}
	}

	// Filter by provider if specified
	if preferences != nil && preferences.Provider != "" {
		for _, model := range visionModels {
			if strings.EqualFold(model.Provider, preferences.Provider) {
				return model, nil
			}
		}
	}

	// Return first available vision model
	return visionModels[0], nil
}

// Default vision-capable models
var DefaultVisionModels = []*Model{
	{
		ID:       "claude-3-5-sonnet-20241022",
		Name:     "Claude 3.5 Sonnet",
		Provider: "anthropic",
		Capabilities: &Capabilities{
			SupportsVision:   true,
			MaxImageSize:     10 * 1024 * 1024, // 10MB
			MaxImages:        20,
			SupportedFormats: []string{"jpg", "jpeg", "png", "gif", "webp"},
			ContextWindow:    200000,
			OutputTokens:     8192,
			FunctionCalling:  true,
			StreamingSupport: true,
		},
		Metadata: &ModelMetadata{
			Version:  "2024-10-22",
			Released: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			Pricing: &Pricing{
				InputCost:  3.0,
				OutputCost: 15.0,
			},
		},
	},
	{
		ID:       "gpt-4o",
		Name:     "GPT-4o",
		Provider: "openai",
		Capabilities: &Capabilities{
			SupportsVision:   true,
			MaxImageSize:     20 * 1024 * 1024, // 20MB
			MaxImages:        10,
			SupportedFormats: []string{"jpg", "jpeg", "png", "gif", "webp"},
			ContextWindow:    128000,
			OutputTokens:     4096,
			FunctionCalling:  true,
			StreamingSupport: true,
		},
		Metadata: &ModelMetadata{
			Version:  "2024-08-06",
			Released: time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC),
			Pricing: &Pricing{
				InputCost:  2.5,
				OutputCost: 10.0,
			},
		},
	},
	{
		ID:       "gpt-4-turbo",
		Name:     "GPT-4 Turbo",
		Provider: "openai",
		Capabilities: &Capabilities{
			SupportsVision:   true,
			MaxImageSize:     20 * 1024 * 1024, // 20MB
			MaxImages:        10,
			SupportedFormats: []string{"jpg", "jpeg", "png", "gif", "webp"},
			ContextWindow:    128000,
			OutputTokens:     4096,
			FunctionCalling:  true,
			StreamingSupport: true,
		},
		Metadata: &ModelMetadata{
			Version:  "2024-04-09",
			Released: time.Date(2024, 4, 9, 0, 0, 0, 0, time.UTC),
			Pricing: &Pricing{
				InputCost:  10.0,
				OutputCost: 30.0,
			},
		},
	},
	{
		ID:       "gpt-4-vision-preview",
		Name:     "GPT-4 Vision Preview",
		Provider: "openai",
		Capabilities: &Capabilities{
			SupportsVision:   true,
			MaxImageSize:     20 * 1024 * 1024, // 20MB
			MaxImages:        10,
			SupportedFormats: []string{"jpg", "jpeg", "png", "gif", "webp"},
			ContextWindow:    128000,
			OutputTokens:     4096,
			FunctionCalling:  true,
			StreamingSupport: true,
		},
		Metadata: &ModelMetadata{
			Version:    "preview",
			Released:   time.Date(2023, 11, 6, 0, 0, 0, 0, time.UTC),
			Deprecated: true,
			Pricing: &Pricing{
				InputCost:  10.0,
				OutputCost: 30.0,
			},
		},
	},
	{
		ID:       "gemini-2.0-flash",
		Name:     "Gemini 2.0 Flash",
		Provider: "google",
		Capabilities: &Capabilities{
			SupportsVision:   true,
			MaxImageSize:     4 * 1024 * 1024, // 4MB
			MaxImages:        16,
			SupportedFormats: []string{"jpg", "jpeg", "png", "webp"},
			ContextWindow:    1000000,
			OutputTokens:     8192,
			FunctionCalling:  true,
			StreamingSupport: true,
		},
		Metadata: &ModelMetadata{
			Version:  "2.0",
			Released: time.Date(2024, 12, 11, 0, 0, 0, 0, time.UTC),
			Pricing: &Pricing{
				InputCost:  0.075,
				OutputCost: 0.30,
			},
		},
	},
	{
		ID:       "gemini-pro-vision",
		Name:     "Gemini Pro Vision",
		Provider: "google",
		Capabilities: &Capabilities{
			SupportsVision:   true,
			MaxImageSize:     4 * 1024 * 1024, // 4MB
			MaxImages:        16,
			SupportedFormats: []string{"jpg", "jpeg", "png", "webp"},
			ContextWindow:    32760,
			OutputTokens:     2048,
			FunctionCalling:  true,
			StreamingSupport: true,
		},
		Metadata: &ModelMetadata{
			Version:  "1.0",
			Released: time.Date(2023, 12, 13, 0, 0, 0, 0, time.UTC),
			Pricing: &Pricing{
				InputCost:  0.125,
				OutputCost: 0.375,
			},
		},
	},
	{
		ID:       "qwen-vl-plus",
		Name:     "Qwen VL Plus",
		Provider: "qwen",
		Capabilities: &Capabilities{
			SupportsVision:   true,
			MaxImageSize:     10 * 1024 * 1024, // 10MB
			MaxImages:        8,
			SupportedFormats: []string{"jpg", "jpeg", "png"},
			ContextWindow:    32000,
			OutputTokens:     2048,
			FunctionCalling:  false,
			StreamingSupport: true,
		},
		Metadata: &ModelMetadata{
			Version:  "1.0",
			Released: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	},
	{
		ID:       "qwen-vl-max",
		Name:     "Qwen VL Max",
		Provider: "qwen",
		Capabilities: &Capabilities{
			SupportsVision:   true,
			MaxImageSize:     10 * 1024 * 1024, // 10MB
			MaxImages:        8,
			SupportedFormats: []string{"jpg", "jpeg", "png"},
			ContextWindow:    32000,
			OutputTokens:     8192,
			FunctionCalling:  false,
			StreamingSupport: true,
		},
		Metadata: &ModelMetadata{
			Version:  "1.0",
			Released: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	},
}
