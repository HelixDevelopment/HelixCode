package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CrossProviderRegistry manages model compatibility across different providers
type CrossProviderRegistry struct {
	baseDir        string
	registryPath   string
	compatibility  map[string]*ProviderCompatibility
	providers      map[string]*ProviderInfo
	downloadedModels map[string]*DownloadedModel
	mu            sync.RWMutex
}

// ProviderCompatibility describes format support for a provider
type ProviderCompatibility struct {
	Provider      string                `json:"provider"`
	SupportedFormats []ModelFormat       `json:"supported_formats"`
	PreferredFormats []ModelFormat       `json:"preferred_formats"`
	ConversionPaths map[string][]string   `json:"conversion_paths"` // source_format -> [conversion_methods]
	Requirements  ProviderRequirements   `json:"requirements"`
	Performance   ProviderPerformance    `json:"performance"`
	LastUpdated   time.Time             `json:"last_updated"`
}

// ProviderInfo contains metadata about a provider
type ProviderInfo struct {
	Name         string           `json:"name"`
	Type         string           `json:"type"` // "openai-compatible", "custom", "specialized"
	Endpoint     string           `json:"endpoint"`
	Version      string           `json:"version"`
	Repository   string           `json:"repository"`
	Description  string           `json:"description"`
	Website      string           `json:"website"`
	DefaultPort  int              `json:"default_port"`
	License      string           `json:"license"`
	Tags         []string         `json:"tags"`
}

// ProviderRequirements specifies hardware/software requirements
type ProviderRequirements struct {
	MinRAM         string   `json:"min_ram"`
	MinVRAM        string   `json:"min_vram,omitempty"`
	RecommendedVRAM string  `json:"recommended_vram,omitempty"`
	SupportedOS    []string `json:"supported_os"`
	GPURequired    bool     `json:"gpu_required"`
	CPUOnly        bool     `json:"cpu_only"`
	PythonVersion  string   `json:"python_version,omitempty"`
	GPULibraries  []string `json:"gpu_libraries,omitempty"`
}

// ProviderPerformance describes performance characteristics
type ProviderPerformance struct {
	Throughput     string   `json:"throughput"`     // "high", "medium", "low"
	Latency        string   `json:"latency"`        // "low", "medium", "high"
	MemoryUsage    string   `json:"memory_usage"`    // "low", "medium", "high"
	BatchSize      int      `json:"batch_size"`
	Parallelism    int      `json:"parallelism"`
	Optimizations  []string `json:"optimizations"`
}

// DownloadedModel represents a downloaded model and its metadata
type DownloadedModel struct {
	ModelID      string            `json:"model_id"`
	Provider     string            `json:"provider"`
	Format       ModelFormat        `json:"format"`
	Path         string            `json:"path"`
	Size         int64             `json:"size"`
	Checksum     string            `json:"checksum"`
	DownloadTime time.Time         `json:"download_time"`
	LastUsed     time.Time         `json:"last_used"`
	UseCount     int               `json:"use_count"`
	Tags         []string          `json:"tags"`
	Metadata     map[string]string `json:"metadata"`
	CompatibleProviders []string    `json:"compatible_providers"`
}

// ModelCompatibilityQuery represents a query for model compatibility
type ModelCompatibilityQuery struct {
	ModelID      string     `json:"model_id"`
	SourceFormat ModelFormat `json:"source_format"`
	TargetProvider string   `json:"target_provider"`
	TargetFormat ModelFormat `json:"target_format,omitempty"`
	Constraints  map[string]interface{} `json:"constraints,omitempty"`
}

// CompatibilityResult represents the result of a compatibility check
type CompatibilityResult struct {
	IsCompatible   bool              `json:"is_compatible"`
	Confidence     float64           `json:"confidence"`
	ConversionRequired bool           `json:"conversion_required"`
	ConversionPath []string          `json:"conversion_path,omitempty"`
	EstimatedTime  int64             `json:"estimated_time,omitempty"` // in minutes
	EstimatedSize  int64             `json:"estimated_size,omitempty"` // in bytes
	Warnings       []string          `json:"warnings,omitempty"`
	Recommendations []string         `json:"recommendations,omitempty"`
	AlternativeProviders []string     `json:"alternative_providers,omitempty"`
}

// NewCrossProviderRegistry creates a new cross-provider registry
func NewCrossProviderRegistry(baseDir string) *CrossProviderRegistry {
	registry := &CrossProviderRegistry{
		baseDir:        baseDir,
		registryPath:   filepath.Join(baseDir, "registry.json"),
		compatibility:  make(map[string]*ProviderCompatibility),
		providers:      make(map[string]*ProviderInfo),
		downloadedModels: make(map[string]*DownloadedModel),
	}
	
	// Ensure registry directory exists
	os.MkdirAll(baseDir, 0755)
	
	// Load existing registry
	registry.loadRegistry()
	
	// Initialize with default providers if empty
	if len(registry.compatibility) == 0 {
		registry.initializeDefaultProviders()
	}
	
	return registry
}

// GetCompatibleFormats returns formats compatible with a provider
func (r *CrossProviderRegistry) GetCompatibleFormats(provider string) ([]ModelFormat, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	compat, exists := r.compatibility[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found in registry", provider)
	}
	
	return compat.SupportedFormats, nil
}

// CheckCompatibility checks if a model is compatible with a provider
func (r *CrossProviderRegistry) CheckCompatibility(query ModelCompatibilityQuery) (*CompatibilityResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := &CompatibilityResult{
		IsCompatible:   false,
		Confidence:     0.0,
		ConversionRequired: false,
		Warnings:       []string{},
		Recommendations: []string{},
	}
	
	// Check if provider exists
	compat, exists := r.compatibility[query.TargetProvider]
	if !exists {
		return result, fmt.Errorf("provider %s not found", query.TargetProvider)
	}
	
	// Check if format is directly supported
	formatSupported := false
	for _, format := range compat.SupportedFormats {
		if format == query.TargetFormat || (query.TargetFormat == "" && format == query.SourceFormat) {
			formatSupported = true
			break
		}
	}
	
	if formatSupported {
		result.IsCompatible = true
		result.Confidence = 1.0
	} else {
		// Check if conversion is possible
		if conversionPath := r.findConversionPath(query.SourceFormat, query.TargetProvider, query.TargetFormat); conversionPath != nil {
			result.IsCompatible = true
			result.Confidence = 0.8
			result.ConversionRequired = true
			result.ConversionPath = conversionPath
			result.EstimatedTime = r.estimateConversionTime(query.SourceFormat, query.TargetFormat)
		}
	}
	
	// Add alternative providers if not compatible or low confidence
	if !result.IsCompatible || result.Confidence < 0.8 {
		result.AlternativeProviders = r.findAlternativeProviders(query.ModelID, query.SourceFormat)
	}
	
	// Add warnings and recommendations
	result.Warnings = append(result.Warnings, r.generateWarnings(query)...)
	result.Recommendations = append(result.Recommendations, r.generateRecommendations(query)...)
	
	return result, nil
}

// RegisterDownloadedModel registers a downloaded model in the registry
func (r *CrossProviderRegistry) RegisterDownloadedModel(model *DownloadedModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Generate unique key
	key := fmt.Sprintf("%s:%s:%s", model.Provider, model.ModelID, model.Format)
	
	// Update compatible providers
	model.CompatibleProviders = r.findCompatibleProvidersForModel(model.ModelID, model.Format)
	
	r.downloadedModels[key] = model
	
	// Save registry
	return r.saveRegistry()
}

// GetDownloadedModels returns all downloaded models
func (r *CrossProviderRegistry) GetDownloadedModels() []*DownloadedModel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var models []*DownloadedModel
	for _, model := range r.downloadedModels {
		models = append(models, model)
	}
	
	return models
}

// FindModelsForProvider returns models compatible with a specific provider
func (r *CrossProviderRegistry) FindModelsForProvider(provider string) ([]*DownloadedModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var compatibleModels []*DownloadedModel
	
	for _, model := range r.downloadedModels {
		for _, compatibleProvider := range model.CompatibleProviders {
			if compatibleProvider == provider {
				compatibleModels = append(compatibleModels, model)
				break
			}
		}
	}
	
	return compatibleModels, nil
}

// FindOptimalProvider finds the best provider for a given model format
func (r *CrossProviderRegistry) FindOptimalProvider(modelID string, format ModelFormat, constraints map[string]interface{}) (*ProviderInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var candidates []*ProviderScore
	
	for provider, compat := range r.compatibility {
		score := r.scoreProviderForModel(provider, compat, format, constraints)
		if score > 0 {
			providerInfo, exists := r.providers[provider]
			if exists {
				candidates = append(candidates, &ProviderScore{
					Provider: providerInfo,
					Score:    score,
					Reason:   r.getScoreReason(provider, compat, format),
				})
			}
		}
	}
	
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no compatible providers found for format %s", format)
	}
	
	// Return highest scoring provider
	best := candidates[0]
	for _, candidate := range candidates {
		if candidate.Score > best.Score {
			best = candidate
		}
	}
	
	return best.Provider, nil
}

// GetProviderInfo returns information about a specific provider
func (r *CrossProviderRegistry) GetProviderInfo(provider string) (*ProviderInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providerInfo, exists := r.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", provider)
	}
	
	return providerInfo, nil
}

// ListProviders returns all registered providers
func (r *CrossProviderRegistry) ListProviders() []*ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var providers []*ProviderInfo
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	
	return providers
}

// Private helper methods

func (r *CrossProviderRegistry) loadRegistry() {
	data, err := os.ReadFile(r.registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return // Registry doesn't exist yet
		}
		fmt.Printf("Warning: failed to load registry: %v\n", err)
		return
	}
	
	var registry struct {
		Compatibility  map[string]*ProviderCompatibility `json:"compatibility"`
		Providers      map[string]*ProviderInfo          `json:"providers"`
		DownloadedModels map[string]*DownloadedModel    `json:"downloaded_models"`
	}
	
	if err := json.Unmarshal(data, &registry); err != nil {
		fmt.Printf("Warning: failed to parse registry: %v\n", err)
		return
	}
	
	r.compatibility = registry.Compatibility
	r.providers = registry.Providers
	r.downloadedModels = registry.DownloadedModels
	
	// Initialize maps if nil
	if r.compatibility == nil {
		r.compatibility = make(map[string]*ProviderCompatibility)
	}
	if r.providers == nil {
		r.providers = make(map[string]*ProviderInfo)
	}
	if r.downloadedModels == nil {
		r.downloadedModels = make(map[string]*DownloadedModel)
	}
}

func (r *CrossProviderRegistry) saveRegistry() error {
	registry := struct {
		Compatibility  map[string]*ProviderCompatibility `json:"compatibility"`
		Providers      map[string]*ProviderInfo          `json:"providers"`
		DownloadedModels map[string]*DownloadedModel    `json:"downloaded_models"`
	}{
		Compatibility:  r.compatibility,
		Providers:      r.providers,
		DownloadedModels: r.downloadedModels,
	}
	
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(r.registryPath, data, 0644)
}

func (r *CrossProviderRegistry) initializeDefaultProviders() {
	providers := []struct{
		name string
		info *ProviderInfo
		compat *ProviderCompatibility
	}{
		{
			name: "vllm",
			info: &ProviderInfo{
				Name:        "VLLM",
				Type:        "openai-compatible",
				Endpoint:    "http://localhost:8000",
				Version:     "latest",
				Repository:  "https://github.com/vllm-project/vllm.git",
				Description: "High-throughput inference engine for LLMs",
				Website:     "https://vllm.ai",
				DefaultPort: 8000,
				License:     "Apache 2.0",
				Tags:        []string{"gpu", "high-performance", "openai-compatible"},
			},
			compat: &ProviderCompatibility{
				Provider: "vllm",
				SupportedFormats: []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF, FormatFP16, FormatBF16},
				PreferredFormats: []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ},
				ConversionPaths: map[string][]string{
					"hf": {"llamacpp", "autogptq", "autoawq"},
					"fp16": {"llamacpp", "autogptq", "autoawq"},
					"bf16": {"llamacpp", "autogptq", "autoawq"},
				},
				Requirements: ProviderRequirements{
					MinRAM:         "16GB",
					MinVRAM:        "8GB",
					RecommendedVRAM: "16GB",
					SupportedOS:    []string{"linux", "darwin", "windows"},
					GPURequired:    true,
					CPUOnly:        false,
					PythonVersion:  "3.8+",
					GPULibraries:  []string{"CUDA", "ROCm"},
				},
				Performance: ProviderPerformance{
					Throughput:    "high",
					Latency:       "low",
					MemoryUsage:   "medium",
					BatchSize:     100,
					Parallelism:   8,
					Optimizations: []string{"PagedAttention", "ContinuousBatching", "Quantization"},
				},
				LastUpdated: time.Now(),
			},
		},
		{
			name: "llamacpp",
			info: &ProviderInfo{
				Name:        "Llama.cpp",
				Type:        "custom",
				Endpoint:    "http://localhost:8080",
				Version:     "latest",
				Repository:  "https://github.com/ggerganov/llama.cpp.git",
				Description: "LLM inference in C/C++",
				Website:     "https://github.com/ggerganov/llama.cpp",
				DefaultPort:  8080,
				License:     "MIT",
				Tags:        []string{"cpu", "gpu", "lightweight", "gguf"},
			},
			compat: &ProviderCompatibility{
				Provider: "llamacpp",
				SupportedFormats: []ModelFormat{FormatGGUF},
				PreferredFormats: []ModelFormat{FormatGGUF},
				ConversionPaths: map[string][]string{
					"hf": {"llamacpp"},
					"fp16": {"llamacpp"},
					"bf16": {"llamacpp"},
					"gptq": {"gguf_to_gguf"},
					"awq": {"gguf_to_gguf"},
				},
				Requirements: ProviderRequirements{
					MinRAM:         "8GB",
					MinVRAM:        "0GB",
					RecommendedVRAM: "8GB",
					SupportedOS:    []string{"linux", "darwin", "windows", "android", "ios"},
					GPURequired:    false,
					CPUOnly:        true,
				},
				Performance: ProviderPerformance{
					Throughput:    "medium",
					Latency:       "medium",
					MemoryUsage:   "low",
					BatchSize:     1,
					Parallelism:   1,
					Optimizations: []string{"Quantization", "Metal", "OpenCL", "CUDA"},
				},
				LastUpdated: time.Now(),
			},
		},
		{
			name: "ollama",
			info: &ProviderInfo{
				Name:        "Ollama",
				Type:        "openai-compatible",
				Endpoint:    "http://localhost:11434",
				Version:     "latest",
				Repository:  "https://github.com/ollama/ollama.git",
				Description: "Get up and running with Llama 2 and other large language models locally",
				Website:     "https://ollama.ai",
				DefaultPort:  11434,
				License:     "MIT",
				Tags:        []string{"easy", "user-friendly", "multi-model", "cross-platform"},
			},
			compat: &ProviderCompatibility{
				Provider: "ollama",
				SupportedFormats: []ModelFormat{FormatGGUF},
				PreferredFormats: []ModelFormat{FormatGGUF},
				ConversionPaths: map[string][]string{
					"hf": {"ollama_convert"},
					"fp16": {"ollama_convert"},
					"bf16": {"ollama_convert"},
				},
				Requirements: ProviderRequirements{
					MinRAM:         "8GB",
					MinVRAM:        "0GB",
					RecommendedVRAM: "8GB",
					SupportedOS:    []string{"linux", "darwin", "windows"},
					GPURequired:    false,
					CPUOnly:        true,
				},
				Performance: ProviderPerformance{
					Throughput:    "medium",
					Latency:       "medium",
					MemoryUsage:   "medium",
					BatchSize:     1,
					Parallelism:   1,
					Optimizations: []string{"ModelManagement", "GPUAcceleration"},
				},
				LastUpdated: time.Now(),
			},
		},
	}
	
	for _, p := range providers {
		r.providers[p.name] = p.info
		r.compatibility[p.name] = p.compat
	}
	
	// Save initial registry
	r.saveRegistry()
}

func (r *CrossProviderRegistry) findConversionPath(sourceFormat ModelFormat, targetProvider string, targetFormat ModelFormat) []string {
	compat, exists := r.compatibility[targetProvider]
	if !exists {
		return nil
	}
	
	// If target format is specified, check direct conversion
	if targetFormat != "" {
		if paths, exists := compat.ConversionPaths[string(sourceFormat)]; exists {
			return paths
		}
	}
	
	// Check if any supported format can be converted to
	for range compat.SupportedFormats {
		if paths, exists := compat.ConversionPaths[string(sourceFormat)]; exists {
			return paths
		}
	}
	
	return nil
}

func (r *CrossProviderRegistry) findAlternativeProviders(modelID string, format ModelFormat) []string {
	var alternatives []string
	
	for provider, compat := range r.compatibility {
		for _, supportedFormat := range compat.SupportedFormats {
			if supportedFormat == format || r.findConversionPath(format, provider, "") != nil {
				alternatives = append(alternatives, provider)
				break
			}
		}
	}
	
	return alternatives
}

func (r *CrossProviderRegistry) findCompatibleProvidersForModel(modelID string, format ModelFormat) []string {
	var compatible []string
	
	for provider, compat := range r.compatibility {
		for _, supportedFormat := range compat.SupportedFormats {
			if supportedFormat == format {
				compatible = append(compatible, provider)
				break
			}
		}
	}
	
	return compatible
}

func (r *CrossProviderRegistry) estimateConversionTime(sourceFormat, targetFormat ModelFormat) int64 {
	// Base conversion times in minutes
	baseTimes := map[string]map[string]int64{
		"hf": {
			"gguf": 15,
			"gptq": 30,
			"awq":  20,
			"fp16": 5,
			"bf16": 5,
		},
		"fp16": {
			"gguf": 10,
			"gptq": 25,
			"awq":  15,
		},
		"bf16": {
			"gguf": 10,
			"gptq": 25,
			"awq":  15,
		},
	}
	
	if sourceTimes, exists := baseTimes[string(sourceFormat)]; exists {
		if time, exists := sourceTimes[string(targetFormat)]; exists {
			return time
		}
	}
	
	return 30 // Default 30 minutes
}

func (r *CrossProviderRegistry) generateWarnings(query ModelCompatibilityQuery) []string {
	var warnings []string
	
	// Check provider requirements
	if compat, exists := r.compatibility[query.TargetProvider]; exists {
		if compat.Requirements.GPURequired {
			warnings = append(warnings, "Provider requires GPU support")
		}
		
		if len(compat.Requirements.SupportedOS) > 0 {
			warnings = append(warnings, "Check OS compatibility")
		}
	}
	
	// Check format-specific warnings
	switch query.TargetFormat {
	case FormatGPTQ:
		warnings = append(warnings, "GPTQ format requires NVIDIA GPU for best performance")
	case FormatAWQ:
		warnings = append(warnings, "AWQ format may have limited CPU support")
	}
	
	return warnings
}

func (r *CrossProviderRegistry) generateRecommendations(query ModelCompatibilityQuery) []string {
	var recommendations []string
	
	// Format recommendations
	switch query.TargetFormat {
	case FormatGGUF:
		recommendations = append(recommendations, "GGUF is recommended for broad compatibility and CPU inference")
	case FormatGPTQ:
		recommendations = append(recommendations, "GPTQ provides excellent GPU performance with minimal quality loss")
	case FormatAWQ:
		recommendations = append(recommendations, "AWQ offers good balance of performance and quality")
	}
	
	// Provider recommendations
	if compat, exists := r.compatibility[query.TargetProvider]; exists {
		if compat.Performance.Throughput == "high" {
			recommendations = append(recommendations, "Provider optimized for high-throughput scenarios")
		}
		if compat.Performance.Latency == "low" {
			recommendations = append(recommendations, "Provider optimized for low-latency scenarios")
		}
	}
	
	return recommendations
}

func (r *CrossProviderRegistry) scoreProviderForModel(provider string, compat *ProviderCompatibility, format ModelFormat, constraints map[string]interface{}) float64 {
	var score float64
	
	// Check format compatibility
	formatSupported := false
	for _, supportedFormat := range compat.SupportedFormats {
		if supportedFormat == format {
			formatSupported = true
			score += 0.4
			break
		}
	}
	
	if !formatSupported {
		// Check if conversion is possible
		if r.findConversionPath(format, provider, "") != nil {
			score += 0.2 // Lower score for conversion required
		} else {
			return 0 // Not compatible
		}
	}
	
	// Preferred format bonus
	for _, preferredFormat := range compat.PreferredFormats {
		if preferredFormat == format {
			score += 0.2
			break
		}
	}
	
	// Performance scoring
	switch compat.Performance.Throughput {
	case "high":
		score += 0.2
	case "medium":
		score += 0.1
	}
	
	switch compat.Performance.Latency {
	case "low":
		score += 0.1
	case "medium":
		score += 0.05
	}
	
	// Apply constraints
	if constraints != nil {
		if gpuRequired, ok := constraints["gpu_required"].(bool); ok && gpuRequired && !compat.Requirements.GPURequired {
			score -= 0.3
		}
		
		if cpuOnly, ok := constraints["cpu_only"].(bool); ok && cpuOnly && compat.Requirements.GPURequired {
			score -= 0.3
		}
	}
	
	return score
}

func (r *CrossProviderRegistry) getScoreReason(provider string, compat *ProviderCompatibility, format ModelFormat) string {
	reasons := []string{}
	
	// Check format support
	for _, supportedFormat := range compat.SupportedFormats {
		if supportedFormat == format {
			reasons = append(reasons, "direct format support")
			break
		}
	}
	
	// Check performance characteristics
	if compat.Performance.Throughput == "high" {
		reasons = append(reasons, "high throughput")
	}
	if compat.Performance.Latency == "low" {
		reasons = append(reasons, "low latency")
	}
	
	return fmt.Sprintf("Selected for: %s", strings.Join(reasons, ", "))
}

// ProviderScore represents a scored provider for selection
type ProviderScore struct {
	Provider *ProviderInfo
	Score    float64
	Reason   string
}