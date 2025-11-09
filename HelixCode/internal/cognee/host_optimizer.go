package cognee

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"dev.helix.code/internal/hardware"
)

// HostOptimizer implements host-aware optimization for Cognee
type HostOptimizer struct {
	Profile *hardware.Profile
}

// NewHostOptimizer creates a new host optimizer
func NewHostOptimizer(profile *hardware.Profile) *HostOptimizer {
	return &HostOptimizer{
		Profile: profile,
	}
}

// OptimizeConfig optimizes Cognee configuration based on host profile
func (ho *HostOptimizer) OptimizeConfig(config *CogneeConfig) *CogneeConfig {
	optimized := *config // Create a copy
	
	// Apply CPU optimizations
	optimized = ho.optimizeForCPU(&optimized)
	
	// Apply GPU optimizations
	optimized = ho.optimizeForGPU(&optimized)
	
	// Apply memory optimizations
	optimized = ho.optimizeForMemory(&optimized)
	
	// Apply OS-specific optimizations
	optimized = ho.optimizeForOS(&optimized)
	
	// Apply architecture optimizations
	optimized = ho.optimizeForArchitecture(&optimized)
	
	return &optimized
}

// optimizeForCPU optimizes configuration based on CPU characteristics
func (ho *HostOptimizer) optimizeForCPU(config *CogneeConfig) *CogneeConfig {
	if !config.Optimization.CPUOptimization {
		return config
	}
	
	cpu := ho.Profile.CPU
	
	// Adjust worker count based on CPU cores
	if cpu.Cores <= 2 {
		config.Performance.Workers = 2
		config.Performance.QueueSize = 100
		config.Performance.BatchSize = 8
		config.Performance.OptimizationLevel = "low"
	} else if cpu.Cores <= 4 {
		config.Performance.Workers = cpu.Cores
		config.Performance.QueueSize = 500
		config.Performance.BatchSize = 16
		config.Performance.OptimizationLevel = "medium"
	} else if cpu.Cores <= 8 {
		config.Performance.Workers = cpu.Cores * 2
		config.Performance.QueueSize = 1000
		config.Performance.BatchSize = 32
		config.Performance.OptimizationLevel = "high"
	} else {
		config.Performance.Workers = cpu.Cores * 3
		config.Performance.QueueSize = 2000
		config.Performance.BatchSize = 64
		config.Performance.OptimizationLevel = "maximum"
	}
	
	// Adjust based on CPU frequency
	if cpu.FrequencyGHz >= 3.5 {
		config.Performance.BatchSize *= 2
		config.Performance.FlushInterval = 2 * config.Performance.FlushInterval
	}
	
	// Adjust based on hyper-threading
	if cpu.Threads > cpu.Cores {
		config.Performance.Workers = cpu.Threads
	}
	
	// CPU-specific optimizations
	if ho.isHighPerformanceCPU() {
		config.Features.RealTimeProcessing = true
		config.Features.AutoOptimization = true
		config.Monitoring.TraceEnabled = true
	}
	
	return config
}

// optimizeForGPU optimizes configuration based on GPU characteristics
func (ho *HostOptimizer) optimizeForGPU(config *CogneeConfig) *CogneeConfig {
	if !config.Optimization.GPUOptimization {
		return config
	}
	
	gpus := ho.Profile.GPUs
	
	if len(gpus) == 0 {
		// CPU-only optimization
		config.Performance.MaxMemory = ho.Profile.Memory.Available / 2
		config.Cache.MaxSize = ho.Profile.Memory.Available / 4
		return config
	}
	
	// GPU optimizations
	for _, gpu := range gpus {
		switch gpu.Type {
		case hardware.GPUTypeNVIDIA:
			config = ho.optimizeForNVIDIAGPU(config, gpu)
		case hardware.GPUTypeApple:
			config = ho.optimizeForAppleGPU(config, gpu)
		case hardware.GPUTypeAMD:
			config = ho.optimizeForAMDGPU(config, gpu)
		case hardware.GPUTypeIntel:
			config = ho.optimizeForIntelGPU(config, gpu)
		}
	}
	
	// Multi-GPU optimizations
	if len(gpus) > 1 {
		config.Performance.Workers *= len(gpus)
		config.Performance.QueueSize *= len(gpus)
		config.Performance.BatchSize *= len(gpus)
		config.Features.MultiModalSupport = true
	}
	
	return config
}

// optimizeForMemory optimizes configuration based on memory characteristics
func (ho *HostOptimizer) optimizeForMemory(config *CogneeConfig) *CogneeConfig {
	if !config.Optimization.MemoryOptimization {
		return config
	}
	
	mem := ho.Profile.Memory
	
	// Adjust memory limits based on available memory
	if mem.TotalGB <= 4 {
		// Low memory system
		config.Performance.MaxMemory = 1024 * 1024 * 1024 // 1GB
		config.Cache.MaxSize = 512 * 1024 * 1024    // 512MB
		config.Performance.QueueSize = 100
		config.Performance.BatchSize = 4
		config.Performance.OptimizationLevel = "low"
		config.Features.RealTimeProcessing = false
	} else if mem.TotalGB <= 8 {
		// Medium memory system
		config.Performance.MaxMemory = 2 * 1024 * 1024 * 1024 // 2GB
		config.Cache.MaxSize = 1 * 1024 * 1024 * 1024      // 1GB
		config.Performance.QueueSize = 500
		config.Performance.BatchSize = 8
		config.Performance.OptimizationLevel = "medium"
	} else if mem.TotalGB <= 16 {
		// High memory system
		config.Performance.MaxMemory = 4 * 1024 * 1024 * 1024 // 4GB
		config.Cache.MaxSize = 2 * 1024 * 1024 * 1024      // 2GB
		config.Performance.QueueSize = 1000
		config.Performance.BatchSize = 16
		config.Performance.OptimizationLevel = "high"
	} else if mem.TotalGB <= 32 {
		// Very high memory system
		config.Performance.MaxMemory = 8 * 1024 * 1024 * 1024 // 8GB
		config.Cache.MaxSize = 4 * 1024 * 1024 * 1024      // 4GB
		config.Performance.QueueSize = 2000
		config.Performance.BatchSize = 32
		config.Performance.OptimizationLevel = "maximum"
	} else {
		// Enterprise memory system
		config.Performance.MaxMemory = 16 * 1024 * 1024 * 1024 // 16GB
		config.Cache.MaxSize = 8 * 1024 * 1024 * 1024       // 8GB
		config.Performance.QueueSize = 4000
		config.Performance.BatchSize = 64
		config.Performance.OptimizationLevel = "enterprise"
		config.Features.MultiModalSupport = true
		config.Features.AdvancedInsights = true
	}
	
	// Adjust for available memory
	availableRatio := float64(mem.AvailableGB) / float64(mem.TotalGB)
	if availableRatio < 0.5 {
		// Low available memory, reduce usage
		config.Performance.MaxMemory = int64(float64(config.Performance.MaxMemory) * 0.7)
		config.Cache.MaxSize = int64(float64(config.Cache.MaxSize) * 0.6)
		config.Performance.BatchSize /= 2
	}
	
	return config
}

// optimizeForOS optimizes configuration based on operating system
func (ho *HostOptimizer) optimizeForOS(config *CogneeConfig) *CogneeConfig {
	os := ho.Profile.OS
	
	switch os.Type {
	case hardware.OSTypeMacOS:
		return ho.optimizeForMacOS(config)
	case hardware.OSTypeLinux:
		return ho.optimizeForLinux(config)
	case hardware.OSTypeWindows:
		return ho.optimizeForWindows(config)
	default:
		return config
	}
}

// optimizeForArchitecture optimizes configuration based on system architecture
func (ho *HostOptimizer) optimizeForArchitecture(config *CogneeConfig) *CogneeConfig {
	arch := ho.Profile.Architecture
	
	switch arch {
	case hardware.ArchARM64:
		return ho.optimizeForARM64(config)
	case hardware.ArchX86_64:
		return ho.optimizeForX86_64(config)
	case hardware.ArchARM32:
		return ho.optimizeForARM32(config)
	default:
		return config
	}
}

// GPU-specific optimizations

func (ho *HostOptimizer) optimizeForNVIDIAGPU(config *CogneeConfig, gpu hardware.GPU) *CogneeConfig {
	// NVIDIA GPU optimizations
	if gpu.VRAMGB >= 8 {
		config.Performance.Workers *= 2
		config.Performance.BatchSize *= 2
		config.Features.MultiModalSupport = true
	}
	
	// CUDA optimizations
	config.Optimization.HostSpecific["cuda_optimization"] = true
	config.Optimization.HostSpecific["tensor_cores"] = gpu.SupportsTensorCores()
	config.Optimization.HostSpecific["cuda_version"] = ho.getCUDAVersion()
	
	// VRAM-based cache sizing
	if gpu.VRAMGB >= 24 {
		config.Cache.MaxSize = 2 * 1024 * 1024 * 1024 // 2GB in GPU cache
	} else if gpu.VRAMGB >= 12 {
		config.Cache.MaxSize = 1 * 1024 * 1024 * 1024 // 1GB in GPU cache
	} else if gpu.VRAMGB >= 6 {
		config.Cache.MaxSize = 512 * 1024 * 1024 // 512MB in GPU cache
	}
	
	return config
}

func (ho *HostOptimizer) optimizeForAppleGPU(config *CogneeConfig, gpu hardware.GPU) *CogneeConfig {
	// Apple Silicon GPU optimizations
	config.Optimization.HostSpecific["metal_optimization"] = true
	config.Optimization.HostSpecific["unified_memory"] = true
	config.Optimization.HostSpecific["mlx_optimization"] = true
	
	// Apple Silicon specific features
	if strings.Contains(gpu.Name, "M2") || strings.Contains(gpu.Name, "M3") {
		config.Performance.Workers *= 2
		config.Features.MultiModalSupport = true
		config.Features.RealTimeProcessing = true
	}
	
	// Unified memory optimization
	config.Performance.MaxMemory = ho.Profile.Memory.Available / 3
	
	return config
}

func (ho *HostOptimizer) optimizeForAMDGPU(config *CogneeConfig, gpu hardware.GPU) *CogneeConfig {
	// AMD GPU optimizations
	config.Optimization.HostSpecific["rocm_optimization"] = true
	config.Optimization.HostSpecific["vulkan_optimization"] = true
	
	// ROCm-specific optimizations
	if gpu.VRAMGB >= 8 {
		config.Performance.Workers = config.Performance.Workers * 3 / 2
	}
	
	return config
}

func (ho *HostOptimizer) optimizeForIntelGPU(config *CogneeConfig, gpu hardware.GPU) *CogneeConfig {
	// Intel GPU optimizations
	config.Optimization.HostSpecific["opencl_optimization"] = true
	config.Optimization.HostSpecific["level_zero_optimization"] = true
	
	// Intel integrated GPU optimizations
	config.Performance.Workers = max(config.Performance.Workers/2, 2)
	config.Performance.BatchSize = max(config.Performance.BatchSize/2, 4)
	
	return config
}

// OS-specific optimizations

func (ho *HostOptimizer) optimizeForMacOS(config *CogneeConfig) *CogneeConfig {
	// macOS optimizations
	config.Cache.Type = "memory" // Use memory cache for better performance
	config.Monitoring.MetricsPort = 9090
	
	// macOS-specific features
	config.Optimization.HostSpecific["macos_optimization"] = true
	config.Optimization.HostSpecific["metal_performance"] = true
	
	// Process management
	config.Performance.Workers = ho.Profile.CPU.Cores
	
	return config
}

func (ho *HostOptimizer) optimizeForLinux(config *CogneeConfig) *CogneeConfig {
	// Linux optimizations
	config.Cache.Type = "redis" // Use Redis cache for better performance
	config.Monitoring.MetricsPort = 9091
	
	// Linux-specific features
	config.Optimization.HostSpecific["linux_optimization"] = true
	config.Optimization.HostSpecific["systemd_integration"] = true
	
	// System resource limits
	config.Performance.MaxMemory = ho.Profile.Memory.Available / 2
	
	// File system optimizations
	config.Optimization.HostSpecific["linux_filesystem"] = true
	config.Optimization.HostSpecific["io_uring"] = ho.supportsIOUring()
	
	return config
}

func (ho *HostOptimizer) optimizeForWindows(config *CogneeConfig) *CogneeConfig {
	// Windows optimizations
	config.Cache.Type = "memory" // Use memory cache for simplicity
	config.Monitoring.MetricsPort = 9092
	
	// Windows-specific features
	config.Optimization.HostSpecific["windows_optimization"] = true
	config.Optimization.HostSpecific["win32_integration"] = true
	
	// Process management
	config.Performance.Workers = ho.Profile.CPU.Cores / 2
	
	return config
}

// Architecture-specific optimizations

func (ho *HostOptimizer) optimizeForARM64(config *CogneeConfig) *CogneeConfig {
	// ARM64 optimizations
	config.Optimization.HostSpecific["arm64_optimization"] = true
	config.Optimization.HostSpecific["neon_instructions"] = true
	
	// ARM64 specific performance tuning
	config.Performance.BatchSize = config.Performance.BatchSize / 2
	config.Performance.FlushInterval = config.Performance.FlushInterval * 2
	
	// ARM64 features
	if strings.Contains(ho.Profile.CPU.Model, "Apple") {
		config.Features.MultiModalSupport = true
		config.Features.RealTimeProcessing = true
	}
	
	return config
}

func (ho *HostOptimizer) optimizeForX86_64(config *CogneeConfig) *CogneeConfig {
	// x86_64 optimizations
	config.Optimization.HostSpecific["x86_64_optimization"] = true
	config.Optimization.HostSpecific["avx_instructions"] = true
	
	// x86_64 specific performance tuning
	if strings.Contains(ho.Profile.CPU.Model, "Intel") {
		config.Performance.Workers *= 2
	}
	
	// Instruction set optimizations
	config.Optimization.HostSpecific["avx2_support"] = ho.supportsAVX2()
	config.Optimization.HostSpecific["avx512_support"] = ho.supportsAVX512()
	
	return config
}

func (ho *HostOptimizer) optimizeForARM32(config *CogneeConfig) *CogneeConfig {
	// ARM32 optimizations
	config.Optimization.HostSpecific["arm32_optimization"] = true
	config.Performance.Workers = max(config.Performance.Workers/2, 1)
	config.Performance.BatchSize = max(config.Performance.BatchSize/2, 2)
	
	return config
}

// Helper functions

func (ho *HostOptimizer) isHighPerformanceCPU() bool {
	cpu := ho.Profile.CPU
	
	// High performance indicators
	highPerfIndicators := []string{
		"Intel Core i9",
		"Intel Core i7",
		"AMD Ryzen 9",
		"AMD Ryzen 7",
		"Apple M",
		"Intel Xeon",
		"AMD EPYC",
	}
	
	for _, indicator := range highPerfIndicators {
		if strings.Contains(cpu.Model, indicator) {
			return true
		}
	}
	
	// High frequency
	if cpu.FrequencyGHz >= 3.5 {
		return true
	}
	
	// High core count
	if cpu.Cores >= 8 {
		return true
	}
	
	return false
}

func (ho *HostOptimizer) getCUDAVersion() string {
	// This would detect CUDA version
	// For now, return a placeholder
	return "12.0"
}

func (ho *HostOptimizer) supportsIOUring() bool {
	// This would check for io_uring support
	// For now, assume modern Linux supports it
	return ho.Profile.OS.Type == hardware.OSTypeLinux
}

func (ho *HostOptimizer) supportsAVX2() bool {
	// This would check for AVX2 support
	// For now, assume modern x86_64 supports it
	return ho.Profile.Architecture == hardware.ArchX86_64
}

func (ho *HostOptimizer) supportsAVX512() bool {
	// This would check for AVX512 support
	// For now, check CPU model
	model := strings.ToLower(ho.Profile.CPU.Model)
	return strings.Contains(model, "intel") && 
		   (strings.Contains(model, "xeon") || strings.Contains(model, "core i9"))
}

// Performance Prediction

func (ho *HostOptimizer) PredictPerformance() map[string]interface{} {
	predictions := make(map[string]interface{})
	
	// Calculate performance score
	cpuScore := ho.calculateCPUScore()
	gpuScore := ho.calculateGPUScore()
	memoryScore := ho.calculateMemoryScore()
	
	overallScore := (cpuScore + gpuScore + memoryScore) / 3
	
	predictions["cpu_score"] = cpuScore
	predictions["gpu_score"] = gpuScore
	predictions["memory_score"] = memoryScore
	predictions["overall_score"] = overallScore
	
	// Performance tier
	var tier string
	if overallScore >= 90 {
		tier = "enterprise"
	} else if overallScore >= 70 {
		tier = "high"
	} else if overallScore >= 50 {
		tier = "medium"
	} else {
		tier = "low"
	}
	predictions["performance_tier"] = tier
	
	// Recommended configuration
	predictions["recommended_workers"] = ho.recommendWorkers()
	predictions["recommended_batch_size"] = ho.recommendBatchSize()
	predictions["recommended_cache_size"] = ho.recommendCacheSize()
	
	// Bottleneck analysis
	predictions["bottleneck"] = ho.identifyBottleneck(cpuScore, gpuScore, memoryScore)
	
	return predictions
}

func (ho *HostOptimizer) calculateCPUScore() float64 {
	cpu := ho.Profile.CPU
	
	// Base score from cores
	score := float64(cpu.Cores) * 10
	
	// Bonus for hyper-threading
	if cpu.Threads > cpu.Cores {
		score *= 1.5
	}
	
	// Bonus for frequency
	score += (cpu.FrequencyGHz - 2.0) * 10
	
	// Bonus for high-performance CPU
	if ho.isHighPerformanceCPU() {
		score *= 1.3
	}
	
	// Architecture bonus
	if ho.Profile.Architecture == hardware.ArchX86_64 {
		score *= 1.1
	} else if ho.Profile.Architecture == hardware.ArchARM64 {
		score *= 1.2
	}
	
	// Clamp to 0-100
	if score > 100 {
		score = 100
	}
	
	return score
}

func (ho *HostOptimizer) calculateGPUScore() float64 {
	gpus := ho.Profile.GPUs
	
	if len(gpus) == 0 {
		return 0
	}
	
	var totalScore float64
	
	for _, gpu := range gpus {
		score := 0.0
		
		// Base score from VRAM
		score += float64(gpu.VRAMGB) * 2
		
		// Bonus for modern GPUs
		if strings.Contains(gpu.Name, "RTX") || strings.Contains(gpu.Name, "Radeon") {
			score += 20
		}
		
		// Type-specific bonuses
		switch gpu.Type {
		case hardware.GPUTypeNVIDIA:
			score *= 1.2
		case hardware.GPUTypeApple:
			score *= 1.1
		case hardware.GPUTypeAMD:
			score *= 1.1
		}
		
		totalScore += score
	}
	
	// Average across GPUs
	avgScore := totalScore / float64(len(gpus))
	
	// Multi-GPU bonus
	if len(gpus) > 1 {
		avgScore *= 1.2
	}
	
	// Clamp to 0-100
	if avgScore > 100 {
		avgScore = 100
	}
	
	return avgScore
}

func (ho *HostOptimizer) calculateMemoryScore() float64 {
	mem := ho.Profile.Memory
	
	// Base score from total memory
	score := float64(mem.TotalGB) * 2.5
	
	// Bonus for available memory ratio
	availableRatio := float64(mem.AvailableGB) / float64(mem.TotalGB)
	score += availableRatio * 20
	
	// Clamp to 0-100
	if score > 100 {
		score = 100
	}
	
	return score
}

func (ho *HostOptimizer) recommendWorkers() int {
	cpu := ho.Profile.CPU
	
	// Base on CPU threads
	workers := cpu.Threads
	
	// Adjust for GPU
	if len(ho.Profile.GPUs) > 0 {
		workers *= 2
	}
	
	// Adjust for memory
	if ho.Profile.Memory.TotalGB < 8 {
		workers = min(workers, 4)
	} else if ho.Profile.Memory.TotalGB < 16 {
		workers = min(workers, 8)
	}
	
	return max(workers, 2)
}

func (ho *HostOptimizer) recommendBatchSize() int {
	cpu := ho.Profile.CPU
	
	// Base on CPU cores
	batchSize := cpu.Cores * 4
	
	// Adjust for memory
	if ho.Profile.Memory.TotalGB < 8 {
		batchSize = min(batchSize, 8)
	} else if ho.Profile.Memory.TotalGB < 16 {
		batchSize = min(batchSize, 16)
	} else {
		batchSize = min(batchSize, 32)
	}
	
	// Adjust for GPU
	if len(ho.Profile.GPUs) > 0 {
		batchSize *= 2
	}
	
	return max(batchSize, 4)
}

func (ho *HostOptimizer) recommendCacheSize() int64 {
	mem := ho.Profile.Memory
	
	// Recommend 25% of available memory
	cacheSize := int64(float64(mem.AvailableGB) * 1024 * 1024 * 1024 * 0.25)
	
	// Minimum 100MB
	if cacheSize < 100*1024*1024 {
		cacheSize = 100 * 1024 * 1024
	}
	
	// Maximum 4GB
	if cacheSize > 4*1024*1024*1024 {
		cacheSize = 4 * 1024 * 1024 * 1024
	}
	
	return cacheSize
}

func (ho *HostOptimizer) identifyBottleneck(cpuScore, gpuScore, memoryScore float64) string {
	scores := map[string]float64{
		"cpu":    cpuScore,
		"gpu":    gpuScore,
		"memory": memoryScore,
	}
	
	minScore := 100.0
	bottleneck := "balanced"
	
	for component, score := range scores {
		if score < minScore {
			minScore = score
			bottleneck = component
		}
	}
	
	// If all scores are close, it's balanced
	if cpuScore-gpuScore < 20 && gpuScore-memoryScore < 20 && cpuScore-memoryScore < 20 {
		return "balanced"
	}
	
	return bottleneck
}

// Utility functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}