package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/hardware"
)

// ModelDiscoveryEngine provides intelligent model recommendation and discovery
type ModelDiscoveryEngine struct {
	registry          *CrossProviderRegistry
	hardwareDetector   *hardware.Detector
	usageAnalytics    *UsageAnalytics
	modelRanker       *ModelRanker
	baseDir           string
	cacheDir          string
	mu                sync.RWMutex
}

// ModelRecommendation represents a recommended model
type ModelRecommendation struct {
	Model            *ModelInfo             `json:"model"`
	RecommendationScore float64               `json:"score"`
	Reasons          []string              `json:"reasons"`
	Providers         []string              `json:"providers"`
	EstimatedPerformance *PerformanceEstimate `json:"performance"`
	HardwareFit      *HardwareFit          `json:"hardware_fit"`
	UsageMatch        *UsageMatch           `json:"usage_match"`
}

// PerformanceEstimate provides performance predictions
type PerformanceEstimate struct {
	TokensPerSecond   float64 `json:"tokens_per_second"`
	MemoryUsage       int64   `json:"memory_usage_mb"`
	Latency          int64   `json:"latency_ms"`
	Throughput       int64   `json:"throughput_req_per_min"`
	CostPerMillion   float64 `json:"cost_per_million_tokens"`
	QualityScore     float64 `json:"quality_score"`
}

// HardwareFit evaluates model-hardware compatibility
type HardwareFit struct {
	CPUScore         float64 `json:"cpu_score"`
	GPUScore         float64 `json:"gpu_score"`
	MemoryScore      float64 `json:"memory_score"`
	OverallFit       float64 `json:"overall_fit"`
	WillRun          bool    `json:"will_run"`
	OptimalSettings  map[string]interface{} `json:"optimal_settings"`
	Warnings         []string `json:"warnings"`
	Recommendations  []string `json:"recommendations"`
}

// UsageMatch analyzes model fit for specific use cases
type UsageMatch struct {
	TaskType         string  `json:"task_type"`
	FitScore         float64 `json:"fit_score"`
	RecommendedFor   []string `json:"recommended_for"`
	NotRecommendedFor []string `json:"not_recommended_for"`
	Reasoning        []string `json:"reasoning"`
}

// UsageAnalytics tracks model usage patterns
type UsageAnalytics struct {
	modelUsageStats  map[string]*ModelUsageStats `json:"model_usage_stats"`
	taskPatterns     map[string]*TaskPattern      `json:"task_patterns"`
	userPreferences  map[string]*UserPreferences  `json:"user_preferences"`
	performanceHistory map[string]*PerformanceHistory `json:"performance_history"`
	mu               sync.RWMutex
}

// ModelUsageStats tracks usage statistics for models
type ModelUsageStats struct {
	ModelID          string    `json:"model_id"`
	TotalRequests    int64     `json:"total_requests"`
	AverageLatency   float64   `json:"average_latency_ms"`
	SuccessRate      float64   `json:"success_rate"`
	UserSatisfaction float64   `json:"user_satisfaction"`
	PreferredBy      []string  `json:"preferred_by"`
	CommonTasks     []string  `json:"common_tasks"`
	PerformanceIssues []string  `json:"performance_issues"`
	LastUsed        time.Time `json:"last_used"`
	UsageTrend      string    `json:"usage_trend"` // "increasing", "decreasing", "stable"
}

// TaskPattern represents usage patterns for specific tasks
type TaskPattern struct {
	TaskType         string    `json:"task_type"`
	CommonModels     []string  `json:"common_models"`
	AverageComplexity float64   `json:"average_complexity"`
	PeakHours        []string  `json:"peak_hours"`
	PerformanceRequirements map[string]float64 `json:"performance_requirements"`
	RecommendedModelSizes []string `json:"recommended_model_sizes"`
}

// UserPreferences tracks individual user preferences
type UserPreferences struct {
	UserID            string            `json:"user_id"`
	PreferredProviders []string          `json:"preferred_providers"`
	QualityPreference  string            `json:"quality_preference"` // "fast", "balanced", "quality"
	BudgetConstraints  map[string]float64 `json:"budget_constraints"`
	TaskFrequencies   map[string]int    `json:"task_frequencies"`
	HardwareConstraints map[string]bool   `json:"hardware_constraints"`
	PrivacyRequirements map[string]bool   `json:"privacy_requirements"`
}

// PerformanceHistory tracks performance over time
type PerformanceHistory struct {
	ModelID       string            `json:"model_id"`
	Provider      string            `json:"provider"`
	TimeSeries    []PerformanceDataPoint `json:"time_series"`
	AverageMetrics *PerformanceEstimate `json:"average_metrics"`
	OptimizationHistory []OptimizationRecord `json:"optimization_history"`
}

// PerformanceDataPoint represents a single performance measurement
type PerformanceDataPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	TokensPerSecond float64   `json:"tokens_per_second"`
	MemoryUsage     int64     `json:"memory_usage_mb"`
	Latency        int64     `json:"latency_ms"`
	SuccessRate    float64   `json:"success_rate"`
	UserRating     float64   `json:"user_rating"`
	HardwareConfig map[string]interface{} `json:"hardware_config"`
}

// OptimizationRecord tracks optimization attempts
type OptimizationRecord struct {
	Timestamp       time.Time `json:"timestamp"`
	OptimizationType string   `json:"optimization_type"`
	BeforeMetrics   *PerformanceEstimate `json:"before_metrics"`
	AfterMetrics    *PerformanceEstimate `json:"after_metrics"`
	Improvement     float64   `json:"improvement_percentage"`
	Success        bool      `json:"success"`
	Method         string    `json:"method"`
}

// ModelRanker provides intelligent model ranking
type ModelRanker struct {
	weights           map[string]float64 `json:"weights"`
	scoringFactors    []string          `json:"scoring_factors"`
	customScorers     map[string]CustomScorer `json:"custom_scorers"`
	mu                sync.RWMutex
}

// CustomScorer allows custom scoring logic
type CustomScorer interface {
	Score(model *ModelInfo, context map[string]interface{}) (float64, []string)
	Name() string
	Weight() float64
}

// RecommendationRequest represents a request for model recommendations
type RecommendationRequest struct {
	TaskTypes         []string               `json:"task_types"`
	Constraints       map[string]interface{} `json:"constraints"`
	UserPreferences   *UserPreferences        `json:"user_preferences,omitempty"`
	HardwareProfile   *hardware.HardwareProfile `json:"hardware_profile,omitempty"`
	BudgetLimit       float64                `json:"budget_limit,omitempty"`
	QualityPreference  string                 `json:"quality_preference"` // "fast", "balanced", "quality"
	PrivacyLevel      string                 `json:"privacy_level"` // "local", "hybrid", "cloud"
	MaxRecommendations int                    `json:"max_recommendations"`
	ExcludeModels    []string               `json:"exclude_models,omitempty"`
	IncludeProviders []string               `json:"include_providers,omitempty"`
}

// RecommendationResponse contains model recommendations
type RecommendationResponse struct {
	Recommendations  []*ModelRecommendation `json:"recommendations"`
	TotalModels     int                  `json:"total_models"`
	SearchTime      time.Duration         `json:"search_time"`
	RelevanceScore  float64              `json:"relevance_score"`
	Alternatives     []*ModelRecommendation `json:"alternatives"`
	Insights        *RecommendationInsights `json:"insights"`
}

// RecommendationInsights provides insights about recommendations
type RecommendationInsights struct {
	MarketTrends     []string `json:"market_trends"`
	PerformanceComparisons map[string]float64 `json:"performance_comparisons"`
	CostAnalysis     map[string]float64 `json:"cost_analysis"`
	HardwareAnalysis map[string]string `json:"hardware_analysis"`
	RecommendationReasoning []string `json:"recommendation_reasoning"`
}

// NewModelDiscoveryEngine creates a new model discovery engine
func NewModelDiscoveryEngine(baseDir string) *ModelDiscoveryEngine {
	cacheDir := filepath.Join(baseDir, "cache", "discovery")
	os.MkdirAll(cacheDir, 0755)
	
	return &ModelDiscoveryEngine{
		registry:         NewCrossProviderRegistry(baseDir),
		hardwareDetector: hardware.NewDetector(),
		usageAnalytics:   NewUsageAnalytics(baseDir),
		modelRanker:      NewModelRanker(),
		baseDir:          baseDir,
		cacheDir:         cacheDir,
	}
}

// GetRecommendations returns intelligent model recommendations
func (e *ModelDiscoveryEngine) GetRecommendations(ctx context.Context, req *RecommendationRequest) (*RecommendationResponse, error) {
	startTime := time.Now()
	
	// Get available models
	models := e.getAvailableModels(ctx, req)
	
	// Score and rank models
	recommendations := e.scoreAndRankModels(models, req)
	
	// Generate alternatives
	alternatives := e.generateAlternatives(recommendations, req)
	
	// Generate insights
	insights := e.generateInsights(recommendations, alternatives, req)
	
	// Sort by score
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].RecommendationScore > recommendations[j].RecommendationScore
	})
	
	// Limit results
	if req.MaxRecommendations > 0 && len(recommendations) > req.MaxRecommendations {
		recommendations = recommendations[:req.MaxRecommendations]
	}
	
	return &RecommendationResponse{
		Recommendations: recommendations,
		TotalModels:     len(models),
		SearchTime:      time.Since(startTime),
		RelevanceScore:  e.calculateRelevanceScore(recommendations, req),
		Alternatives:     alternatives,
		Insights:        insights,
	}, nil
}

// getAvailableModels retrieves models based on request constraints
func (e *ModelDiscoveryEngine) getAvailableModels(ctx context.Context, req *RecommendationRequest) []*ModelInfo {
	// Get models from cross-provider registry
	downloadedModels := e.registry.GetDownloadedModels()
	
	// Convert to ModelInfo
	var models []*ModelInfo
	modelMap := make(map[string]*ModelInfo)
	
	for _, downloadedModel := range downloadedModels {
		model := &ModelInfo{
			ID:            downloadedModel.ModelID,
			Name:          strings.Title(strings.ReplaceAll(downloadedModel.ModelID, "-", " ")),
			Format:        downloadedModel.Format,
			Size:          downloadedModel.Size,
			ContextSize:   4096, // Default, should be from metadata
			Provider:      ProviderLocalLLM,
			Capabilities:  e.inferCapabilities(downloadedModel.ModelID),
			Metadata:      downloadedModel.Metadata,
		}
		
		models = append(models, model)
		modelMap[downloadedModel.ModelID] = model
	}
	
	// Add models from external sources (HuggingFace, etc.)
	externalModels := e.fetchExternalModels(ctx, req)
	for _, model := range externalModels {
		if modelMap[model.ID] == nil {
			models = append(models, model)
		}
	}
	
	// Filter by constraints
	filtered := e.filterModelsByConstraints(models, req)
	
	// Exclude specified models
	if len(req.ExcludeModels) > 0 {
		excludeMap := make(map[string]bool)
		for _, exclude := range req.ExcludeModels {
			excludeMap[exclude] = true
		}
		
		var result []*ModelInfo
		for _, model := range filtered {
			if !excludeMap[model.ID] {
				result = append(result, model)
			}
		}
		filtered = result
	}
	
	return filtered
}

// scoreAndRankModels scores and ranks models based on request
func (e *ModelDiscoveryEngine) scoreAndRankModels(models []*ModelInfo, req *RecommendationRequest) []*ModelRecommendation {
	var recommendations []*ModelRecommendation
	
	for _, model := range models {
		recommendation := e.scoreModel(model, req)
		recommendations = append(recommendations, recommendation)
	}
	
	return recommendations
}

// scoreModel scores a single model based on request criteria
func (e *ModelDiscoveryEngine) scoreModel(model *ModelInfo, req *RecommendationRequest) *ModelRecommendation {
	score := 0.0
	reasons := []string{}
	
	// Task compatibility scoring
	taskScore := e.scoreTaskCompatibility(model, req.TaskTypes)
	score += taskScore * 0.3
	if taskScore > 0.7 {
		reasons = append(reasons, "Excellent task compatibility")
	}
	
	// Hardware compatibility scoring
	hardwareScore := e.scoreHardwareCompatibility(model, req.HardwareProfile)
	score += hardwareScore * 0.25
	if hardwareScore > 0.8 {
		reasons = append(reasons, "Perfect hardware fit")
	}
	
	// Performance scoring
	performanceScore := e.scorePerformance(model, req.QualityPreference)
	score += performanceScore * 0.2
	if performanceScore > 0.8 {
		reasons = append(reasons, "High performance expected")
	}
	
	// Cost scoring
	costScore := e.scoreCost(model, req.BudgetLimit)
	score += costScore * 0.15
	if costScore > 0.8 {
		reasons = append(reasons, "Cost-effective")
	}
	
	// Privacy scoring
	privacyScore := e.scorePrivacy(model, req.PrivacyLevel)
	score += privacyScore * 0.1
	if privacyScore > 0.9 {
		reasons = append(reasons, "Privacy-compliant")
	}
	
	// Get compatible providers
	providers := e.getCompatibleProviders(model)
	
	// Estimate performance
	performanceEstimate := e.estimatePerformance(model, req.HardwareProfile)
	
	// Evaluate hardware fit
	hardwareFit := e.evaluateHardwareFit(model, req.HardwareProfile)
	
	// Analyze usage match
	usageMatch := e.analyzeUsageMatch(model, req.TaskTypes)
	
	return &ModelRecommendation{
		Model:               model,
		RecommendationScore:  score,
		Reasons:             reasons,
		Providers:           providers,
		EstimatedPerformance: performanceEstimate,
		HardwareFit:         hardwareFit,
		UsageMatch:          usageMatch,
	}
}

// scoreTaskCompatibility scores model compatibility with task types
func (e *ModelDiscoveryEngine) scoreTaskCompatibility(model *ModelInfo, taskTypes []string) float64 {
	if len(taskTypes) == 0 {
		return 0.5 // Neutral score
	}
	
	totalScore := 0.0
	for _, taskType := range taskTypes {
		taskScore := e.scoreModelForTask(model, taskType)
		totalScore += taskScore
	}
	
	return totalScore / float64(len(taskTypes))
}

// scoreModelForTask scores a model for a specific task
func (e *ModelDiscoveryEngine) scoreModelForTask(model *ModelInfo, taskType string) float64 {
	taskRequirements := map[string]map[ModelCapability]float64{
		"code_generation": {
			CapabilityCodeGeneration: 1.0,
			CapabilityReasoning:       0.8,
			CapabilityDebugging:       0.7,
		},
		"planning": {
			CapabilityPlanning:        1.0,
			CapabilityReasoning:       0.9,
			CapabilityAnalysis:       0.8,
		},
		"debugging": {
			CapabilityDebugging:       1.0,
			CapabilityCodeGeneration: 0.8,
			CapabilityReasoning:       0.7,
		},
		"testing": {
			CapabilityTesting:         1.0,
			CapabilityCodeGeneration: 0.7,
			CapabilityAnalysis:       0.6,
		},
		"refactoring": {
			CapabilityRefactoring:    1.0,
			CapabilityCodeGeneration: 0.8,
			CapabilityAnalysis:       0.7,
		},
		"documentation": {
			CapabilityDocumentation: 1.0,
			CapabilityReasoning:       0.6,
			CapabilityWriting:        0.8,
		},
		"analysis": {
			CapabilityAnalysis:       1.0,
			CapabilityReasoning:       0.8,
			CapabilityPlanning:        0.6,
		},
	}
	
	requirements, exists := taskRequirements[taskType]
	if !exists {
		return 0.5 // Unknown task, neutral score
	}
	
	totalScore := 0.0
	totalWeight := 0.0
	for capability, weight := range requirements {
		if e.hasCapability(model.Capabilities, capability) {
			totalScore += weight
		}
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return 0.0
	}
	
	return totalScore / totalWeight
}

// scoreHardwareCompatibility scores model compatibility with hardware
func (e *ModelDiscoveryEngine) scoreHardwareCompatibility(model *ModelInfo, profile *hardware.HardwareProfile) float64 {
	if profile == nil {
		// Use detected hardware if no profile provided
		var err error
		profile, err = e.hardwareDetector.Detect()
		if err != nil {
			return 0.5 // Unknown hardware, neutral score
		}
	}
	
	// Estimate model memory requirements
	modelMemory := e.estimateModelMemoryRequirements(model)
	
	// Score based on available memory
	if profile.TotalRAM > 0 {
		memoryScore := math.Min(1.0, float64(profile.TotalRAM)/float64(modelMemory))
		if memoryScore < 0.5 {
			return memoryScore // Not enough memory
		}
	}
	
	// GPU compatibility
	if profile.GPU != nil {
		gpuScore := e.scoreGPUCompatibility(model, profile.GPU)
		return gpuScore
	}
	
	// CPU-only scoring
	cpuScore := e.scoreCPUCompatibility(model, profile.CPU)
	return cpuScore
}

// scoreGPUCompatibility scores model compatibility with GPU
func (e *ModelDiscoveryEngine) scoreGPUCompatibility(model *ModelInfo, gpu *hardware.GPUInfo) float64 {
	// Check VRAM requirements
	modelVRAM := e.estimateModelVRAMRequirements(model)
	if gpu.VRAM > 0 && modelVRAM > 0 {
		vramScore := math.Min(1.0, float64(gpu.VRAM)/float64(modelVRAM))
		if vramScore < 0.5 {
			return vramScore
		}
	}
	
	// Check compute capability
	if gpu.ComputeCapability > 0 {
		// Certain models require specific compute capabilities
		computeScore := 1.0
		if strings.Contains(model.ID, "llama") && gpu.ComputeCapability < 6.0 {
			computeScore = 0.7
		}
		return computeScore
	}
	
	return 0.8 // Unknown GPU capability, reasonably compatible
}

// scoreCPUCompatibility scores model compatibility with CPU
func (e *ModelDiscoveryEngine) scoreCPUCompatibility(model *ModelInfo, cpu *hardware.CPUInfo) float64 {
	// Check for AVX/AVX2 support (important for many models)
	cpuScore := 0.7 // Base score
	
	if cpu.HasAVX {
		cpuScore += 0.1
	}
	if cpu.HasAVX2 {
		cpuScore += 0.1
	}
	if cpu.HasNEON {
		cpuScore += 0.1
	}
	
	// Model size considerations for CPU
	modelSize := e.estimateModelSize(model.ID)
	if modelSize == "70B" || modelSize == "34B" {
		if cpu.Cores < 8 {
			cpuScore *= 0.6
		}
	}
	
	return math.Min(1.0, cpuScore)
}

// scorePerformance scores model based on quality preference
func (e *ModelDiscoveryEngine) scorePerformance(model *ModelInfo, qualityPreference string) float64 {
	modelSize := e.estimateModelSize(model.ID)
	
	switch qualityPreference {
	case "fast":
		// Prefer smaller, faster models
		switch modelSize {
		case "3B":
			return 1.0
		case "7B":
			return 0.8
		case "13B":
			return 0.6
		case "34B":
			return 0.4
		case "70B":
			return 0.2
		default:
			return 0.5
		}
	case "quality":
		// Prefer larger, higher quality models
		switch modelSize {
		case "3B":
			return 0.3
		case "7B":
			return 0.5
		case "13B":
			return 0.7
		case "34B":
			return 0.9
		case "70B":
			return 1.0
		default:
			return 0.5
		}
	case "balanced":
		// Prefer medium-sized models
		switch modelSize {
		case "3B":
			return 0.4
		case "7B":
			return 0.8
		case "13B":
			return 0.9
		case "34B":
			return 0.6
		case "70B":
			return 0.3
		default:
			return 0.5
		}
	default:
		return 0.5 // Neutral preference
	}
}

// scoreCost scores model based on budget constraints
func (e *ModelDiscoveryEngine) scoreCost(model *ModelInfo, budgetLimit float64) float64 {
	if budgetLimit <= 0 {
		return 0.5 // No budget constraint
	}
	
	// Estimate cost per million tokens (simplified)
	modelSize := e.estimateModelSize(model.ID)
	costPerMillion := e.estimateCostPerMillion(modelSize, model.Format)
	
	if costPerMillion > budgetLimit {
		return 0.0 // Over budget
	}
	
	// Score based on how much under budget
	budgetRatio := costPerMillion / budgetLimit
	return math.Max(0.0, 1.0-budgetRatio*2.0) // Penalize as we approach budget
}

// scorePrivacy scores model based on privacy requirements
func (e *ModelDiscoveryEngine) scorePrivacy(model *ModelInfo, privacyLevel string) float64 {
	switch privacyLevel {
	case "local":
		// All local models get perfect score
		return 1.0
	case "hybrid":
		// Some models may have cloud components
		if strings.Contains(model.ID, "api") || strings.Contains(model.ID, "cloud") {
			return 0.5
		}
		return 1.0
	case "cloud":
		// All models are acceptable
		return 1.0
	default:
		return 0.8 // Unknown privacy level, reasonably private
	}
}

// Helper methods

func (e *ModelDiscoveryEngine) inferCapabilities(modelID string) []ModelCapability {
	capabilities := []ModelCapability{}
	
	modelID = strings.ToLower(modelID)
	
	// Infer capabilities from model name
	if strings.Contains(modelID, "code") || strings.Contains(modelID, "instruct") {
		capabilities = append(capabilities, CapabilityCodeGeneration)
	}
	if strings.Contains(modelID, "reason") || strings.Contains(modelID, "analysis") {
		capabilities = append(capabilities, CapabilityReasoning)
	}
	if strings.Contains(modelID, "debug") {
		capabilities = append(capabilities, CapabilityDebugging)
	}
	if strings.Contains(modelID, "test") {
		capabilities = append(capabilities, CapabilityTesting)
	}
	if strings.Contains(modelID, "refactor") {
		capabilities = append(capabilities, CapabilityRefactoring)
	}
	if strings.Contains(modelID, "plan") {
		capabilities = append(capabilities, CapabilityPlanning)
	}
	if strings.Contains(modelID, "doc") || strings.Contains(modelID, "write") {
		capabilities = append(capabilities, CapabilityDocumentation, CapabilityWriting)
	}
	
	// Default capabilities for all models
	if len(capabilities) == 0 {
		capabilities = append(capabilities, CapabilityTextGeneration, CapabilityReasoning)
	}
	
	return capabilities
}

func (e *ModelDiscoveryEngine) fetchExternalModels(ctx context.Context, req *RecommendationRequest) []*ModelInfo {
	// In a real implementation, this would fetch from external APIs
	// For now, return some popular models
	return []*ModelInfo{
		{
			ID:           "llama-3-8b-instruct",
			Name:         "Llama 3 8B Instruct",
			Format:       FormatGGUF,
			Size:         4700000000, // ~4.7GB
			ContextSize:  8192,
			Provider:     ProviderLocalLLM,
			Capabilities: []ModelCapability{CapabilityCodeGeneration, CapabilityReasoning, CapabilityDebugging},
		},
		{
			ID:           "mistral-7b-instruct",
			Name:         "Mistral 7B Instruct",
			Format:       FormatGGUF,
			Size:         4100000000, // ~4.1GB
			ContextSize:  32768,
			Provider:     ProviderLocalLLM,
			Capabilities: []ModelCapability{CapabilityCodeGeneration, CapabilityReasoning, CapabilityAnalysis},
		},
		{
			ID:           "codellama-7b-instruct",
			Name:         "CodeLlama 7B Instruct",
			Format:       FormatGGUF,
			Size:         3800000000, // ~3.8GB
			ContextSize:  16384,
			Provider:     ProviderLocalLLM,
			Capabilities: []ModelCapability{CapabilityCodeGeneration, CapabilityDebugging, CapabilityTesting},
		},
	}
}

func (e *ModelDiscoveryEngine) filterModelsByConstraints(models []*ModelInfo, req *RecommendationRequest) []*ModelInfo {
	var filtered []*ModelInfo
	
	for _, model := range models {
		if e.modelSatisfiesConstraints(model, req) {
			filtered = append(filtered, model)
		}
	}
	
	return filtered
}

func (e *ModelDiscoveryEngine) modelSatisfiesConstraints(model *ModelInfo, req *RecommendationRequest) bool {
	// Check provider constraints
	if len(req.IncludeProviders) > 0 {
		providers := e.getCompatibleProviders(model)
		found := false
		for _, includedProvider := range req.IncludeProviders {
			for _, provider := range providers {
				if provider == includedProvider {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}
	
	// Check size constraints
	if maxMemory, ok := req.Constraints["max_memory_mb"].(float64); ok {
		modelMemory := float64(e.estimateModelMemoryRequirements(model))
		if modelMemory > maxMemory {
			return false
		}
	}
	
	// Check privacy constraints
	if privacyLevel, ok := req.Constraints["privacy_level"].(string); ok {
		privacyScore := e.scorePrivacy(model, privacyLevel)
		if privacyScore < 0.5 {
			return false
		}
	}
	
	return true
}

func (e *ModelDiscoveryEngine) getCompatibleProviders(model *ModelInfo) []string {
	providers := []string{}
	
	allProviders := []string{"vllm", "llamacpp", "ollama", "localai", "fastchat", "textgen"}
	for _, provider := range allProviders {
		if e.isModelCompatibleWithProvider(model, provider) {
			providers = append(providers, provider)
		}
	}
	
	return providers
}

func (e *ModelDiscoveryEngine) isModelCompatibleWithProvider(model *ModelInfo, provider string) bool {
	// Simplified compatibility check
	switch provider {
	case "llamacpp":
		return model.Format == FormatGGUF
	case "vllm":
		return model.Format == FormatGGUF || model.Format == FormatGPTQ || model.Format == FormatHF
	case "ollama":
		return model.Format == FormatGGUF
	default:
		return true // Assume compatible for unknown providers
	}
}

func (e *ModelDiscoveryEngine) estimatePerformance(model *ModelInfo, profile *hardware.HardwareProfile) *PerformanceEstimate {
	modelSize := e.estimateModelSize(model.ID)
	baseTPS := e.estimateBaseTokensPerSecond(modelSize, profile)
	
	return &PerformanceEstimate{
		TokensPerSecond: baseTPS,
		MemoryUsage:     int64(e.estimateModelMemoryRequirements(model)),
		Latency:        int64(1000.0 / baseTPS), // Approximate latency
		Throughput:     int64(baseTPS * 60),      // Requests per minute
		CostPerMillion: e.estimateCostPerMillion(modelSize, model.Format),
		QualityScore:   e.estimateQualityScore(modelSize),
	}
}

func (e *ModelDiscoveryEngine) evaluateHardwareFit(model *ModelInfo, profile *hardware.HardwareProfile) *HardwareFit {
	if profile == nil {
		var err error
		profile, err = e.hardwareDetector.Detect()
		if err != nil {
			return &HardwareFit{
				OverallFit: 0.5,
				WillRun:    true,
				Warnings:    []string{"Hardware detection failed"},
			}
		}
	}
	
	modelMemory := e.estimateModelMemoryRequirements(model)
	modelVRAM := e.estimateModelVRAMRequirements(model)
	
	// CPU scoring
	cpuScore := 0.7 // Base score
	if profile.CPU.HasAVX2 {
		cpuScore += 0.2
	}
	if profile.CPU.Cores >= 8 {
		cpuScore += 0.1
	}
	
	// GPU scoring
	gpuScore := 0.0
	if profile.GPU != nil {
		gpuScore = 0.7 // Base score
		if profile.GPU.VRAM >= modelVRAM {
			gpuScore += 0.3
		}
	}
	
	// Memory scoring
	memoryScore := 0.0
	if profile.TotalRAM >= modelMemory {
		memoryScore = 1.0
	} else if profile.TotalRAM >= modelMemory/2 {
		memoryScore = 0.7
	} else {
		memoryScore = 0.3
	}
	
	// Overall fit
	overallFit := (cpuScore + gpuScore + memoryScore) / 3.0
	willRun := profile.TotalRAM >= modelMemory/2 // Minimum requirements
	
	warnings := []string{}
	recommendations := []string{}
	optimalSettings := map[string]interface{}{}
	
	if profile.TotalRAM < modelMemory {
		warnings = append(warnings, "Insufficient RAM for optimal performance")
		recommendations = append(recommendations, "Consider using a smaller model or upgrading RAM")
	}
	
	if profile.GPU != nil && profile.GPU.VRAM < modelVRAM {
		warnings = append(warnings, "Insufficient VRAM for GPU acceleration")
		recommendations = append(recommendations, "GPU fallback to CPU or consider quantization")
		optimalSettings["use_gpu"] = false
	} else if profile.GPU != nil {
		optimalSettings["use_gpu"] = true
		optimalSettings["gpu_layers"] = e.calculateOptimalGPULayers(model, profile.GPU)
	}
	
	return &HardwareFit{
		CPUScore:        cpuScore,
		GPUScore:        gpuScore,
		MemoryScore:     memoryScore,
		OverallFit:      overallFit,
		WillRun:         willRun,
		OptimalSettings: optimalSettings,
		Warnings:        warnings,
		Recommendations: recommendations,
	}
}

func (e *ModelDiscoveryEngine) analyzeUsageMatch(model *ModelInfo, taskTypes []string) *UsageMatch {
	if len(taskTypes) == 0 {
		return &UsageMatch{
			TaskType:   "general",
			FitScore:   0.5,
			Reasoning:  []string{"No specific tasks provided"},
		}
	}
	
	totalFitScore := 0.0
	allRecommended := []string{}
	allNotRecommended := []string{}
	allReasoning := []string{}
	
	for _, taskType := range taskTypes {
		fitScore := e.scoreModelForTask(model, taskType)
		totalFitScore += fitScore
		
		if fitScore > 0.8 {
			allRecommended = append(allRecommended, taskType)
		} else if fitScore < 0.4 {
			allNotRecommended = append(allNotRecommended, taskType)
		}
		
		if fitScore > 0.7 {
			allReasoning = append(allReasoning, fmt.Sprintf("Well-suited for %s", taskType))
		} else if fitScore < 0.5 {
			allReasoning = append(allReasoning, fmt.Sprintf("Not ideal for %s", taskType))
		}
	}
	
	averageFitScore := totalFitScore / float64(len(taskTypes))
	
	return &UsageMatch{
		TaskType:           strings.Join(taskTypes, ", "),
		FitScore:           averageFitScore,
		RecommendedFor:     allRecommended,
		NotRecommendedFor:  allNotRecommended,
		Reasoning:          allReasoning,
	}
}

// Additional estimation methods

func (e *ModelDiscoveryEngine) estimateModelSize(modelID string) string {
	modelID = strings.ToLower(modelID)
	
	if strings.Contains(modelID, "70b") {
		return "70B"
	}
	if strings.Contains(modelID, "34b") {
		return "34B"
	}
	if strings.Contains(modelID, "13b") {
		return "13B"
	}
	if strings.Contains(modelID, "8b") {
		return "8B"
	}
	if strings.Contains(modelID, "7b") {
		return "7B"
	}
	if strings.Contains(modelID, "3b") {
		return "3B"
	}
	
	return "7B" // Default estimate
}

func (e *ModelDiscoveryEngine) estimateModelMemoryRequirements(model *ModelInfo) int64 {
	modelSize := e.estimateModelSize(model.ID)
	// Rough estimation: model size in GB * 2 (for loading) * 1024 MB/GB
	sizeInGB := map[string]float64{
		"3B":  2.0,
		"7B":  4.5,
		"8B":  5.0,
		"13B": 8.0,
		"34B": 20.0,
		"70B": 40.0,
	}
	
	if gb, exists := sizeInGB[modelSize]; exists {
		return int64(gb * 1024)
	}
	
	return int64(4.5 * 1024) // Default for 7B
}

func (e *ModelDiscoveryEngine) estimateModelVRAMRequirements(model *ModelInfo) int64 {
	// VRAM requirements are typically lower than RAM due to quantization
	ramRequirement := e.estimateModelMemoryRequirements(model)
	return int64(float64(ramRequirement) * 0.7) // 70% of RAM requirement
}

func (e *ModelDiscoveryEngine) estimateBaseTokensPerSecond(modelSize string, profile *hardware.HardwareProfile) float64 {
	baseTPS := map[string]float64{
		"3B":  50.0,
		"7B":  25.0,
		"8B":  20.0,
		"13B": 12.0,
		"34B": 4.0,
		"70B": 2.0,
	}
	
	tps, exists := baseTPS[modelSize]
	if !exists {
		tps = 25.0 // Default
	}
	
	// Adjust based on hardware
	if profile != nil {
		if profile.GPU != nil {
			tps *= 2.0 // GPU acceleration
			if profile.GPU.VRAM > 16000 { // >16GB
				tps *= 1.5 // High-end GPU
			}
		}
		
		if profile.CPU.Cores >= 16 {
			tps *= 1.3 // High-end CPU
		}
	}
	
	return tps
}

func (e *ModelDiscoveryEngine) estimateCostPerMillion(modelSize string, format ModelFormat) float64 {
	// Local models have negligible cost (just electricity)
	// This is for comparison with cloud models
	sizeMultiplier := map[string]float64{
		"3B":  0.1,
		"7B":  0.2,
		"8B":  0.25,
		"13B": 0.4,
		"34B": 1.0,
		"70B": 2.0,
	}
	
	if multiplier, exists := sizeMultiplier[modelSize]; exists {
		return multiplier
	}
	
	return 0.2 // Default
}

func (e *ModelDiscoveryEngine) estimateQualityScore(modelSize string) float64 {
	// Quality score based on model size (simplified)
	sizeScore := map[string]float64{
		"3B":  0.6,
		"7B":  0.7,
		"8B":  0.75,
		"13B": 0.8,
		"34B": 0.9,
		"70B": 0.95,
	}
	
	if score, exists := sizeScore[modelSize]; exists {
		return score
	}
	
	return 0.7 // Default
}

func (e *ModelDiscoveryEngine) calculateOptimalGPULayers(model *ModelInfo, gpu *hardware.GPUInfo) int {
	// Simplified calculation for optimal GPU layers
	modelSize := e.estimateModelSize(model.ID)
	baseLayers := map[string]int{
		"3B":  20,
		"7B":  32,
		"8B":  35,
		"13B": 40,
		"34B": 45,
		"70B": 48,
	}
	
	if layers, exists := baseLayers[modelSize]; exists {
		// Adjust based on VRAM
		vramRatio := float64(gpu.VRAM) / 8000.0 // Base on 8GB
		adjustedLayers := int(float64(layers) * vramRatio)
		return int(math.Min(float64(layers), float64(adjustedLayers)))
	}
	
	return 32 // Default
}

func (e *ModelDiscoveryEngine) generateAlternatives(recommendations []*ModelRecommendation, req *RecommendationRequest) []*ModelRecommendation {
	// Generate alternative recommendations for variety
	alternatives := []*ModelRecommendation{}
	
	// For each top recommendation, suggest alternatives
	for _, rec := range recommendations[:min(3, len(recommendations))] {
		// Find similar models with different characteristics
		alternatives = append(alternatives, e.findAlternativeModels(rec.Model, req)...)
	}
	
	return alternatives
}

func (e *ModelDiscoveryEngine) findAlternativeModels(model *ModelInfo, req *RecommendationRequest) []*ModelRecommendation {
	// Find models with similar capabilities but different sizes/formats
	alternatives := []*ModelRecommendation{}
	
	// This would typically query external APIs
	// For now, return some hardcoded alternatives
	alternativeMap := map[string][]string{
		"llama-3-8b-instruct": {"mistral-7b-instruct", "codellama-7b-instruct"},
		"mistral-7b-instruct":  {"llama-3-8b-instruct", "zephyr-7b-beta"},
		"codellama-7b-instruct": {"starcoder-7b", "deepseek-coder-6.7b"},
	}
	
	if altModels, exists := alternativeMap[model.ID]; exists {
		for _, altModelID := range altModels {
			// Create mock alternative model
			altModel := &ModelInfo{
				ID:           altModelID,
				Name:         strings.Title(strings.ReplaceAll(altModelID, "-", " ")),
				Format:       FormatGGUF,
				Capabilities: e.inferCapabilities(altModelID),
			}
			
			altRecommendation := e.scoreModel(altModel, req)
			alternatives = append(alternatives, altRecommendation)
		}
	}
	
	return alternatives
}

func (e *ModelDiscoveryEngine) generateInsights(recommendations []*ModelRecommendation, alternatives []*ModelRecommendation, req *RecommendationRequest) *RecommendationInsights {
	insights := &RecommendationInsights{
		MarketTrends:     []string{},
		PerformanceComparisons: map[string]float64{},
		CostAnalysis:     map[string]float64{},
		HardwareAnalysis: map[string]string{},
		RecommendationReasoning: []string{},
	}
	
	// Analyze market trends
	if len(recommendations) > 0 {
		topRec := recommendations[0]
		if strings.Contains(topRec.Model.ID, "llama") {
			insights.MarketTrends = append(insights.MarketTrends, "Llama models are currently market leaders")
		}
		if strings.Contains(topRec.Model.ID, "mistral") {
			insights.MarketTrends = append(insights.MarketTrends, "Mistral models offer excellent performance")
		}
	}
	
	// Performance comparisons
	for i, rec := range recommendations {
		if rec.EstimatedPerformance != nil {
			insights.PerformanceComparisons[rec.Model.ID] = rec.EstimatedPerformance.TokensPerSecond
		}
		if i == 0 && len(recommendations) > 1 {
			nextRec := recommendations[1]
			if rec.EstimatedPerformance != nil && nextRec.EstimatedPerformance != nil {
				ratio := rec.EstimatedPerformance.TokensPerSecond / nextRec.EstimatedPerformance.TokensPerSecond
				if ratio > 1.5 {
					insights.RecommendationReasoning = append(insights.RecommendationReasoning, 
						fmt.Sprintf("%s is %.1fx faster than alternatives", rec.Model.ID, ratio))
				}
			}
		}
	}
	
	// Cost analysis
	for _, rec := range recommendations {
		if rec.EstimatedPerformance != nil {
			insights.CostAnalysis[rec.Model.ID] = rec.EstimatedPerformance.CostPerMillion
		}
	}
	
	// Hardware analysis
	for _, rec := range recommendations {
		if rec.HardwareFit != nil {
			if rec.HardwareFit.OverallFit > 0.8 {
				insights.HardwareAnalysis[rec.Model.ID] = "Excellent hardware match"
			} else if rec.HardwareFit.OverallFit > 0.6 {
				insights.HardwareAnalysis[rec.Model.ID] = "Good hardware compatibility"
			} else {
				insights.HardwareAnalysis[rec.Model.ID] = "May require hardware optimization"
			}
		}
	}
	
	return insights
}

func (e *ModelDiscoveryEngine) calculateRelevanceScore(recommendations []*ModelRecommendation, req *RecommendationRequest) float64 {
	if len(recommendations) == 0 {
		return 0.0
	}
	
	// Calculate average score
	totalScore := 0.0
	for _, rec := range recommendations {
		totalScore += rec.RecommendationScore
	}
	
	averageScore := totalScore / float64(len(recommendations))
	
	// Adjust based on how well recommendations match request
	relevanceMultiplier := 1.0
	if len(req.TaskTypes) > 0 {
		// Check if recommendations are well-suited for requested tasks
		taskMatch := 0.0
		for _, rec := range recommendations {
			if rec.UsageMatch != nil {
				taskMatch += rec.UsageMatch.FitScore
			}
		}
		taskMatch /= float64(len(recommendations))
		relevanceMultiplier *= taskMatch
	}
	
	return averageScore * relevanceMultiplier
}

func (e *ModelDiscoveryEngine) hasCapability(capabilities []ModelCapability, capability ModelCapability) bool {
	for _, cap := range capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}