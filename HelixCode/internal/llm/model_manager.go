package llm

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/hardware"
)

// ModelManager manages LLM models and their selection
type ModelManager struct {
	hardwareDetector *hardware.Detector
	providers        map[ProviderType]Provider
	modelRegistry    map[string]*ModelInfo
	mu               sync.RWMutex
}

// ModelSelectionCriteria defines criteria for model selection
type ModelSelectionCriteria struct {
	TaskType             string
	RequiredCapabilities []ModelCapability
	MaxTokens            int
	Budget               float64 // Cost budget in USD
	LatencyRequirement   time.Duration
	QualityPreference    string // "fast", "balanced", "quality"
}

// ModelScore represents a scored model for selection
type ModelScore struct {
	Model      *ModelInfo
	Score      float64
	Reason     string
	Confidence float64
}

// NewModelManager creates a new model manager
func NewModelManager() *ModelManager {
	return &ModelManager{
		hardwareDetector: hardware.NewDetector(),
		providers:        make(map[ProviderType]Provider),
		modelRegistry:    make(map[string]*ModelInfo),
	}
}

// RegisterProvider registers an LLM provider with the manager
func (m *ModelManager) RegisterProvider(provider Provider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	providerType := provider.GetType()
	if _, exists := m.providers[providerType]; exists {
		return fmt.Errorf("provider %s already registered", providerType)
	}

	m.providers[providerType] = provider

	// Register provider's models
	models := provider.GetModels()
	for i := range models {
		model := &models[i]
		modelKey := m.getModelKey(providerType, model.Name)
		m.modelRegistry[modelKey] = model
	}

	log.Printf("✅ Provider registered: %s with %d models", provider.GetName(), len(models))
	return nil
}

// SelectOptimalModel selects the best model for given criteria
func (m *ModelManager) SelectOptimalModel(criteria ModelSelectionCriteria) (*ModelInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get available models
	availableModels := m.getAvailableModels()
	if len(availableModels) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// Score models based on criteria
	scoredModels := m.scoreModels(availableModels, criteria)
	if len(scoredModels) == 0 {
		return nil, fmt.Errorf("no suitable models found for criteria")
	}

	// Sort by score (descending)
	sort.Slice(scoredModels, func(i, j int) bool {
		return scoredModels[i].Score > scoredModels[j].Score
	})

	bestModel := scoredModels[0]
	log.Printf("🎯 Selected model: %s (score: %.2f, reason: %s)",
		bestModel.Model.Name, bestModel.Score, bestModel.Reason)

	return bestModel.Model, nil
}

// GetAvailableModels returns all available models
func (m *ModelManager) GetAvailableModels() []*ModelInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getAvailableModels()
}

// GetModelsByCapability returns models that support specific capabilities
func (m *ModelManager) GetModelsByCapability(capabilities []ModelCapability) []*ModelInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matching []*ModelInfo
	for _, model := range m.modelRegistry {
		if m.hasAllCapabilities(model.Capabilities, capabilities) {
			matching = append(matching, model)
		}
	}

	return matching
}

// GetProviderForModel returns the provider for a specific model
func (m *ModelManager) GetProviderForModel(modelName string, providerType ProviderType) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	modelKey := m.getModelKey(providerType, modelName)
	if _, exists := m.modelRegistry[modelKey]; !exists {
		return nil, fmt.Errorf("model %s not found for provider %s", modelName, providerType)
	}

	provider, exists := m.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider %s not available", providerType)
	}

	return provider, nil
}

// HealthCheck performs health checks on all providers
func (m *ModelManager) HealthCheck(ctx context.Context) map[ProviderType]*ProviderHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[ProviderType]*ProviderHealth)
	for providerType, provider := range m.providers {
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

// Private helper methods

func (m *ModelManager) getAvailableModels() []*ModelInfo {
	var models []*ModelInfo
	for _, model := range m.modelRegistry {
		models = append(models, model)
	}
	return models
}

func (m *ModelManager) scoreModels(models []*ModelInfo, criteria ModelSelectionCriteria) []ModelScore {
	var scored []ModelScore

	for _, model := range models {
		score := m.calculateModelScore(model, criteria)
		if score.Score > 0 {
			scored = append(scored, score)
		}
	}

	return scored
}

func (m *ModelManager) calculateModelScore(model *ModelInfo, criteria ModelSelectionCriteria) ModelScore {
	var score float64

	// Base score
	baseScore := 1.0

	// Capability matching
	capabilityScore := m.calculateCapabilityScore(model.Capabilities, criteria.RequiredCapabilities)
	if capabilityScore == 0 {
		return ModelScore{Model: model, Score: 0, Reason: "missing required capabilities"}
	}
	baseScore *= capabilityScore

	// Context size adequacy
	if criteria.MaxTokens > 0 && model.ContextSize < criteria.MaxTokens {
		return ModelScore{Model: model, Score: 0, Reason: "insufficient context size"}
	}
	contextScore := float64(model.ContextSize) / float64(criteria.MaxTokens)
	if contextScore > 2.0 {
		contextScore = 2.0 // Cap at 2x requirement
	}
	baseScore *= contextScore

	// Task type suitability
	taskScore := m.calculateTaskSuitability(model, criteria.TaskType)
	baseScore *= taskScore

	// Hardware compatibility
	hardwareScore := m.calculateHardwareCompatibility(model)
	baseScore *= hardwareScore

	// Quality preference
	qualityScore := m.calculateQualityScore(model, criteria.QualityPreference)
	baseScore *= qualityScore

	// Provider availability
	provider, exists := m.providers[model.Provider]
	if !exists || !provider.IsAvailable(context.Background()) {
		return ModelScore{Model: model, Score: 0, Reason: "provider unavailable"}
	}

	score = baseScore
	reason := fmt.Sprintf("capabilities:%.2f, task:%.2f, hardware:%.2f, quality:%.2f",
		capabilityScore, taskScore, hardwareScore, qualityScore)

	return ModelScore{
		Model:      model,
		Score:      score,
		Reason:     reason,
		Confidence: m.calculateConfidence(model, criteria),
	}
}

func (m *ModelManager) calculateCapabilityScore(available, required []ModelCapability) float64 {
	if len(required) == 0 {
		return 1.0
	}

	availableMap := make(map[ModelCapability]bool)
	for _, cap := range available {
		availableMap[cap] = true
	}

	matched := 0
	for _, req := range required {
		if availableMap[req] {
			matched++
		}
	}

	return float64(matched) / float64(len(required))
}

func (m *ModelManager) calculateTaskSuitability(model *ModelInfo, taskType string) float64 {
	// Task-specific scoring based on model capabilities
	switch taskType {
	case "planning":
		// Planning tasks benefit from strong reasoning capabilities
		if m.hasCapability(model.Capabilities, CapabilityPlanning) {
			return 1.2
		}
	case "code_generation":
		// Code generation benefits from code-specific training
		if m.hasCapability(model.Capabilities, CapabilityCodeGeneration) {
			return 1.3
		}
	case "debugging":
		// Debugging benefits from analytical capabilities
		if m.hasCapability(model.Capabilities, CapabilityDebugging) {
			return 1.2
		}
	case "testing":
		// Testing benefits from structured output
		if m.hasCapability(model.Capabilities, CapabilityTesting) {
			return 1.1
		}
	case "refactoring":
		// Refactoring benefits from code analysis
		if m.hasCapability(model.Capabilities, CapabilityRefactoring) {
			return 1.2
		}
	}

	return 1.0 // Default suitability
}

func (m *ModelManager) calculateHardwareCompatibility(model *ModelInfo) float64 {
	// Check if model can run on current hardware
	_, err := m.hardwareDetector.Detect()
	if err != nil {
		log.Printf("Warning: Hardware detection failed: %v", err)
		return 0.8 // Assume compatibility with penalty
	}

	// Estimate model size from name
	modelSize := m.estimateModelSize(model.Name)
	if modelSize != "" {
		if !m.hardwareDetector.CanRunModel(modelSize) {
			return 0.0 // Cannot run this model
		}
	}

	return 1.0 // Hardware compatible
}

func (m *ModelManager) calculateQualityScore(model *ModelInfo, preference string) float64 {
	// Estimate model quality from name and capabilities
	var qualityEstimate float64

	// Larger models generally produce higher quality output
	modelSize := m.estimateModelSize(model.Name)
	switch modelSize {
	case "70B":
		qualityEstimate = 1.3
	case "34B":
		qualityEstimate = 1.2
	case "13B":
		qualityEstimate = 1.1
	case "7B":
		qualityEstimate = 1.0
	case "3B":
		qualityEstimate = 0.9
	default:
		qualityEstimate = 1.0
	}

	// Apply preference
	switch preference {
	case "quality":
		return qualityEstimate * 1.2
	case "fast":
		return 1.0 / qualityEstimate // Faster models for speed preference
	case "balanced":
		fallthrough
	default:
		return 1.0
	}
}

func (m *ModelManager) calculateConfidence(model *ModelInfo, criteria ModelSelectionCriteria) float64 {
	// Calculate confidence in model selection
	confidence := 0.5 // Base confidence

	// Increase confidence for exact capability matches
	if len(criteria.RequiredCapabilities) > 0 {
		capabilityMatch := m.calculateCapabilityScore(model.Capabilities, criteria.RequiredCapabilities)
		confidence += capabilityMatch * 0.3
	}

	// Increase confidence for task-specific models
	if criteria.TaskType != "" {
		taskSuitability := m.calculateTaskSuitability(model, criteria.TaskType)
		confidence += (taskSuitability - 1.0) * 0.2
	}

	// Cap confidence at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (m *ModelManager) estimateModelSize(modelName string) string {
	name := strings.ToLower(modelName)

	if strings.Contains(name, "70b") || strings.Contains(name, "70b") {
		return "70B"
	}
	if strings.Contains(name, "34b") || strings.Contains(name, "34b") {
		return "34B"
	}
	if strings.Contains(name, "13b") || strings.Contains(name, "13b") {
		return "13B"
	}
	if strings.Contains(name, "7b") || strings.Contains(name, "7b") {
		return "7B"
	}
	if strings.Contains(name, "3b") || strings.Contains(name, "3b") {
		return "3B"
	}

	return ""
}

func (m *ModelManager) hasAllCapabilities(available, required []ModelCapability) bool {
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

func (m *ModelManager) hasCapability(capabilities []ModelCapability, capability ModelCapability) bool {
	for _, cap := range capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

func (m *ModelManager) getModelKey(providerType ProviderType, modelName string) string {
	return string(providerType) + "::" + modelName
}
