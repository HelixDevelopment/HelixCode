package performance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPerformanceOptimizer tests the constructor
func TestNewPerformanceOptimizer(t *testing.T) {
	t.Run("NewPerformanceOptimizer_MinimalConfig", func(t *testing.T) {
		config := PerformanceConfig{}

		po, err := NewPerformanceOptimizer(config)

		assert.NoError(t, err)
		assert.NotNil(t, po)
		assert.Equal(t, config, po.config)
		assert.NotNil(t, po.metrics)
		assert.NotNil(t, po.optimizations)
		assert.False(t, po.running.Load())
	})

	t.Run("NewPerformanceOptimizer_FullConfig", func(t *testing.T) {
		config := PerformanceConfig{
			CPUOptimization:         true,
			MemoryOptimization:      true,
			GarbageCollection:       true,
			ConcurrencyOptimization: true,
			CacheOptimization:       true,
			NetworkOptimization:     true,
			DatabaseOptimization:    true,
			WorkerOptimization:      true,
			LLMOptimization:         true,
			TargetThroughput:        1000,
			TargetLatency:           "100ms",
			TargetCPUUtilization:    70.0,
			TargetMemoryUsage:       1024 * 1024 * 1024,
			MaxResponseTime:         "500ms",
			MinCacheHitRate:         0.95,
			MaxErrorRate:            0.01,
		}

		po, err := NewPerformanceOptimizer(config)

		assert.NoError(t, err)
		assert.NotNil(t, po)
		// Should have created optimizations for all enabled types
		assert.Greater(t, len(po.optimizations), 0)
	})
}

// TestPerformanceConfig tests the config structure
func TestPerformanceConfig(t *testing.T) {
	t.Run("PerformanceConfig_ZeroValues", func(t *testing.T) {
		config := PerformanceConfig{}

		assert.False(t, config.CPUOptimization)
		assert.False(t, config.MemoryOptimization)
		assert.Equal(t, 0, config.TargetThroughput)
		assert.Equal(t, 0.0, config.TargetCPUUtilization)
	})

	t.Run("PerformanceConfig_AllEnabled", func(t *testing.T) {
		config := PerformanceConfig{
			CPUOptimization:         true,
			MemoryOptimization:      true,
			GarbageCollection:       true,
			ConcurrencyOptimization: true,
			CacheOptimization:       true,
			NetworkOptimization:     true,
			DatabaseOptimization:    true,
			WorkerOptimization:      true,
			LLMOptimization:         true,
		}

		assert.True(t, config.CPUOptimization)
		assert.True(t, config.MemoryOptimization)
		assert.True(t, config.GarbageCollection)
		assert.True(t, config.ConcurrencyOptimization)
		assert.True(t, config.CacheOptimization)
		assert.True(t, config.NetworkOptimization)
		assert.True(t, config.DatabaseOptimization)
		assert.True(t, config.WorkerOptimization)
		assert.True(t, config.LLMOptimization)
	})
}

// TestPerformanceMetrics tests metrics structure
func TestPerformanceMetrics(t *testing.T) {
	t.Run("PerformanceMetrics_ZeroValues", func(t *testing.T) {
		metrics := PerformanceMetrics{}

		assert.True(t, metrics.Timestamp.IsZero())
		assert.Equal(t, 0.0, metrics.CPUUtilization)
		assert.Equal(t, int64(0), metrics.MemoryUsage)
		assert.Equal(t, 0, metrics.Throughput)
	})

	t.Run("PerformanceMetrics_WithValues", func(t *testing.T) {
		now := time.Now()
		metrics := PerformanceMetrics{
			Timestamp:         now,
			CPUUtilization:    75.5,
			MemoryUsage:       1024 * 1024 * 100, // 100 MB
			Throughput:        1000,
			AverageLatency:    50 * time.Millisecond,
			P95Latency:        100 * time.Millisecond,
			P99Latency:        200 * time.Millisecond,
			CacheHitRate:      0.95,
			ErrorRate:         0.01,
			WorkerUtilization: []float64{80.0, 75.0, 85.0},
		}

		assert.Equal(t, now, metrics.Timestamp)
		assert.Equal(t, 75.5, metrics.CPUUtilization)
		assert.Equal(t, int64(100*1024*1024), metrics.MemoryUsage)
		assert.Equal(t, 1000, metrics.Throughput)
		assert.Len(t, metrics.WorkerUtilization, 3)
	})
}

// TestOptimizationTypes tests the optimization type constants
func TestOptimizationTypes(t *testing.T) {
	assert.Equal(t, OptType("cpu"), CPUOpt)
	assert.Equal(t, OptType("memory"), MemoryOpt)
	assert.Equal(t, OptType("garbage_collection"), GCOpt)
	assert.Equal(t, OptType("concurrency"), ConcurrencyOpt)
	assert.Equal(t, OptType("cache"), CacheOpt)
	assert.Equal(t, OptType("network"), NetworkOpt)
	assert.Equal(t, OptType("database"), DatabaseOpt)
	assert.Equal(t, OptType("worker"), WorkerOpt)
	assert.Equal(t, OptType("llm"), LLMOpt)
	assert.Equal(t, OptType("comprehensive"), ComprehensiveOpt)
}

// TestInitializeOptimizations tests optimization initialization
func TestInitializeOptimizations(t *testing.T) {
	t.Run("InitializeOptimizations_CPUOnly", func(t *testing.T) {
		config := PerformanceConfig{CPUOptimization: true}
		po, err := NewPerformanceOptimizer(config)

		require.NoError(t, err)
		opts := po.getOptimizationsByType(CPUOpt)
		assert.Len(t, opts, 2)
		assert.Contains(t, po.optimizations, "cpu_goroutine_pool")
		assert.Contains(t, po.optimizations, "cpu_benchmark_optimization")
	})

	t.Run("InitializeOptimizations_MemoryOnly", func(t *testing.T) {
		config := PerformanceConfig{MemoryOptimization: true}
		po, err := NewPerformanceOptimizer(config)

		require.NoError(t, err)
		opts := po.getOptimizationsByType(MemoryOpt)
		assert.Len(t, opts, 2)
		assert.Contains(t, po.optimizations, "memory_pool_optimization")
		assert.Contains(t, po.optimizations, "memory_profiling_optimization")
	})

	t.Run("InitializeOptimizations_GCOnly", func(t *testing.T) {
		config := PerformanceConfig{GarbageCollection: true}
		po, err := NewPerformanceOptimizer(config)

		require.NoError(t, err)
		opts := po.getOptimizationsByType(GCOpt)
		assert.Len(t, opts, 2)
		assert.Contains(t, po.optimizations, "gc_tuning")
		assert.Contains(t, po.optimizations, "gc_monitoring")
	})

	t.Run("InitializeOptimizations_CacheOnly", func(t *testing.T) {
		config := PerformanceConfig{CacheOptimization: true}
		po, err := NewPerformanceOptimizer(config)

		require.NoError(t, err)
		opts := po.getOptimizationsByType(CacheOpt)
		assert.Len(t, opts, 2)
		assert.Contains(t, po.optimizations, "cache_strategy_optimization")
		assert.Contains(t, po.optimizations, "cache_warming")
	})

	t.Run("InitializeOptimizations_AllTypes", func(t *testing.T) {
		config := PerformanceConfig{
			CPUOptimization:         true,
			MemoryOptimization:      true,
			GarbageCollection:       true,
			ConcurrencyOptimization: true,
			CacheOptimization:       true,
			NetworkOptimization:     true,
			DatabaseOptimization:    true,
			WorkerOptimization:      true,
			LLMOptimization:         true,
		}
		po, err := NewPerformanceOptimizer(config)

		require.NoError(t, err)
		// Should have 2 optimizations per type (9 types * 2 = 18)
		assert.Equal(t, 18, len(po.optimizations))
	})
}

// TestCollectMetrics tests metrics collection
func TestCollectMetrics(t *testing.T) {
	t.Run("CollectMetrics_Success", func(t *testing.T) {
		config := PerformanceConfig{}
		po, err := NewPerformanceOptimizer(config)
		require.NoError(t, err)

		metrics, err := po.collectMetrics()

		assert.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.False(t, metrics.Timestamp.IsZero())
		assert.GreaterOrEqual(t, metrics.CPUUtilization, 0.0)
		assert.GreaterOrEqual(t, metrics.MemoryUsage, int64(0))
		assert.GreaterOrEqual(t, metrics.Throughput, 0)
	})

	t.Run("CollectMetrics_GCStats", func(t *testing.T) {
		config := PerformanceConfig{}
		po, err := NewPerformanceOptimizer(config)
		require.NoError(t, err)

		metrics, err := po.collectMetrics()

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, metrics.GCStats.NumGC, uint32(0))
		assert.GreaterOrEqual(t, metrics.GCStats.HeapAlloc, uint64(0))
		assert.GreaterOrEqual(t, metrics.GCStats.HeapSys, uint64(0))
	})
}

// TestGetOptimizationMetric tests getting specific metrics
func TestGetOptimizationMetric(t *testing.T) {
	config := PerformanceConfig{}
	po, err := NewPerformanceOptimizer(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		optType OptType
	}{
		{"CPU", CPUOpt},
		{"Memory", MemoryOpt},
		{"GC", GCOpt},
		{"Concurrency", ConcurrencyOpt},
		{"Cache", CacheOpt},
		{"Network", NetworkOpt},
		{"Database", DatabaseOpt},
		{"Worker", WorkerOpt},
		{"LLM", LLMOpt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := po.getOptimizationMetric(tt.optType)

			assert.NoError(t, err)
			assert.GreaterOrEqual(t, metric, 0.0)
		})
	}

	t.Run("UnsupportedType", func(t *testing.T) {
		metric, err := po.getOptimizationMetric(OptType("unsupported"))

		assert.Error(t, err)
		assert.Equal(t, 0.0, metric)
		assert.Contains(t, err.Error(), "unsupported optimization type")
	})
}

// TestSimulationFunctions tests all simulation functions
func TestSimulationFunctions(t *testing.T) {
	t.Run("simulateCPUUsage", func(t *testing.T) {
		cpu := simulateCPUUsage()
		assert.GreaterOrEqual(t, cpu, 0.0)
		assert.LessOrEqual(t, cpu, 100.0)
	})

	t.Run("simulateThroughput", func(t *testing.T) {
		throughput := simulateThroughput()
		assert.GreaterOrEqual(t, throughput, 1000)
		assert.LessOrEqual(t, throughput, 1500)
	})

	t.Run("simulateLatency", func(t *testing.T) {
		latency := simulateLatency()
		assert.GreaterOrEqual(t, latency, 50*time.Millisecond)
		assert.LessOrEqual(t, latency, 150*time.Millisecond)
	})

	t.Run("simulateP95Latency", func(t *testing.T) {
		p95 := simulateP95Latency()
		assert.GreaterOrEqual(t, p95, 100*time.Millisecond)
		assert.LessOrEqual(t, p95, 250*time.Millisecond)
	})

	t.Run("simulateP99Latency", func(t *testing.T) {
		p99 := simulateP99Latency()
		assert.GreaterOrEqual(t, p99, 200*time.Millisecond)
		assert.LessOrEqual(t, p99, 400*time.Millisecond)
	})

	t.Run("simulateCacheHitRate", func(t *testing.T) {
		hitRate := simulateCacheHitRate()
		assert.GreaterOrEqual(t, hitRate, 0.85)
		assert.LessOrEqual(t, hitRate, 0.95)
	})

	t.Run("simulateErrorRate", func(t *testing.T) {
		errorRate := simulateErrorRate()
		assert.GreaterOrEqual(t, errorRate, 0.01)
		assert.LessOrEqual(t, errorRate, 0.015)
	})

	t.Run("simulateWorkerUtilization", func(t *testing.T) {
		utilization := simulateWorkerUtilization()
		assert.Len(t, utilization, 10)
		for _, u := range utilization {
			assert.GreaterOrEqual(t, u, 60.0)
			assert.LessOrEqual(t, u, 80.0)
		}
	})

	t.Run("simulateLLMResponseTime", func(t *testing.T) {
		llmTime := simulateLLMResponseTime()
		assert.GreaterOrEqual(t, llmTime, 500*time.Millisecond)
		assert.LessOrEqual(t, llmTime, 1500*time.Millisecond)
	})

	t.Run("simulateDatabaseQueryTime", func(t *testing.T) {
		dbTime := simulateDatabaseQueryTime()
		assert.GreaterOrEqual(t, dbTime, 10*time.Millisecond)
		assert.LessOrEqual(t, dbTime, 60*time.Millisecond)
	})

	t.Run("simulateNetworkLatency", func(t *testing.T) {
		netLatency := simulateNetworkLatency()
		assert.GreaterOrEqual(t, netLatency, 5*time.Millisecond)
		assert.LessOrEqual(t, netLatency, 25*time.Millisecond)
	})
}

// TestCalculateImprovement tests the improvement calculation
func TestCalculateImprovement(t *testing.T) {
	t.Run("PositiveImprovement", func(t *testing.T) {
		improvement := calculateImprovement(100.0, 150.0)
		assert.Equal(t, 50.0, improvement)
	})

	t.Run("NegativeImprovement", func(t *testing.T) {
		improvement := calculateImprovement(150.0, 100.0)
		assert.Equal(t, -33.33333333333333, improvement)
	})

	t.Run("NoChange", func(t *testing.T) {
		improvement := calculateImprovement(100.0, 100.0)
		assert.Equal(t, 0.0, improvement)
	})

	t.Run("ZeroBefore", func(t *testing.T) {
		improvement := calculateImprovement(0.0, 100.0)
		assert.Equal(t, 0.0, improvement)
	})
}

// TestGetOptimizationsByType tests filtering optimizations by type
func TestGetOptimizationsByType(t *testing.T) {
	config := PerformanceConfig{
		CPUOptimization:    true,
		MemoryOptimization: true,
		CacheOptimization:  true,
	}
	po, err := NewPerformanceOptimizer(config)
	require.NoError(t, err)

	t.Run("GetCPUOptimizations", func(t *testing.T) {
		opts := po.getOptimizationsByType(CPUOpt)
		assert.Len(t, opts, 2)
		for _, opt := range opts {
			assert.Equal(t, CPUOpt, opt.Type)
		}
	})

	t.Run("GetMemoryOptimizations", func(t *testing.T) {
		opts := po.getOptimizationsByType(MemoryOpt)
		assert.Len(t, opts, 2)
		for _, opt := range opts {
			assert.Equal(t, MemoryOpt, opt.Type)
		}
	})

	t.Run("GetNonExistentType", func(t *testing.T) {
		opts := po.getOptimizationsByType(DatabaseOpt)
		assert.Len(t, opts, 0)
	})
}

// TestApplyOptimizations tests individual optimization application
func TestApplyOptimizations(t *testing.T) {
	config := PerformanceConfig{
		CPUOptimization:         true,
		MemoryOptimization:      true,
		GarbageCollection:       true,
		ConcurrencyOptimization: true,
		CacheOptimization:       true,
		NetworkOptimization:     true,
		DatabaseOptimization:    true,
		WorkerOptimization:      true,
		LLMOptimization:         true,
	}
	po, err := NewPerformanceOptimizer(config)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name     string
		optType  OptType
		applyFn  func(context.Context, *Optimization, float64) (*OptResult, error)
	}{
		{"CPU", CPUOpt, po.applyCPUOptimization},
		{"Memory", MemoryOpt, po.applyMemoryOptimization},
		{"GC", GCOpt, po.applyGCOptimization},
		{"Concurrency", ConcurrencyOpt, po.applyConcurrencyOptimization},
		{"Cache", CacheOpt, po.applyCacheOptimization},
		{"Network", NetworkOpt, po.applyNetworkOptimization},
		{"Database", DatabaseOpt, po.applyDatabaseOptimization},
		{"Worker", WorkerOpt, po.applyWorkerOptimization},
		{"LLM", LLMOpt, po.applyLLMOptimization},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := Optimization{
				Name:     "test-" + string(tt.optType),
				Type:     tt.optType,
				Enabled:  true,
				Priority: 1,
			}

			result, err := tt.applyFn(ctx, &opt, 100.0)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, result.Success)
			assert.NotZero(t, result.BeforeValue)
			// GC metrics can be 0, so just check >= 0
			assert.GreaterOrEqual(t, result.AfterValue, 0.0)
			assert.False(t, result.Timestamp.IsZero())
		})
	}
}

// TestStartProductionOptimization tests the main optimization workflow
func TestStartProductionOptimization(t *testing.T) {
	t.Run("StartProductionOptimization_MinimalConfig", func(t *testing.T) {
		config := PerformanceConfig{
			CPUOptimization:    true,
			MemoryOptimization: true,
			TargetThroughput:   1000,
		}
		po, err := NewPerformanceOptimizer(config)
		require.NoError(t, err)

		ctx := context.Background()
		result, err := po.StartProductionOptimization(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.TotalApplied, 0)
		assert.Greater(t, result.Successful, 0)
		assert.NotNil(t, result.Baseline)
		assert.NotNil(t, result.PostOptimization)
		assert.NotNil(t, result.OverallImprovement)
		assert.False(t, result.StartTime.IsZero())
		assert.False(t, result.EndTime.IsZero())
		assert.True(t, result.EndTime.After(result.StartTime))
	})

	t.Run("StartProductionOptimization_AlreadyRunning", func(t *testing.T) {
		config := PerformanceConfig{CPUOptimization: true}
		po, err := NewPerformanceOptimizer(config)
		require.NoError(t, err)

		po.running.Store(true)
		ctx := context.Background()

		result, err := po.StartProductionOptimization(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "already running")
	})

	t.Run("StartProductionOptimization_AllOptimizations", func(t *testing.T) {
		config := PerformanceConfig{
			CPUOptimization:         true,
			MemoryOptimization:      true,
			GarbageCollection:       true,
			ConcurrencyOptimization: true,
			CacheOptimization:       true,
			NetworkOptimization:     true,
			DatabaseOptimization:    true,
			WorkerOptimization:      true,
			LLMOptimization:         true,
			TargetThroughput:        1500,
			TargetCPUUtilization:    70.0,
			TargetMemoryUsage:       1024 * 1024 * 512,
			MinCacheHitRate:         0.90,
			MaxErrorRate:            0.02,
		}
		po, err := NewPerformanceOptimizer(config)
		require.NoError(t, err)

		ctx := context.Background()
		result, err := po.StartProductionOptimization(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 18, result.TotalApplied) // 9 types * 2 optimizations each
		assert.Greater(t, result.Successful, 0)
		assert.NotNil(t, result.OverallImprovement)
		assert.NotZero(t, result.OverallImprovement.OverallScore)
	})
}

// TestOptimizationResult tests the optimization result structure
func TestOptimizationResult(t *testing.T) {
	t.Run("OptimizationResult_Structure", func(t *testing.T) {
		startTime := time.Now()
		endTime := startTime.Add(5 * time.Minute)

		result := OptimizationResult{
			StartTime:      startTime,
			EndTime:        endTime,
			Duration:       5 * time.Minute,
			TotalApplied:   10,
			Successful:     8,
			Failed:         2,
			Baseline:       &PerformanceMetrics{Throughput: 1000},
			PostOptimization: &PerformanceMetrics{Throughput: 1500},
			Optimizations:  make(map[string]Optimization),
			OverallImprovement: &OverallImprovement{
				ThroughputImprovement: 50.0,
				LatencyImprovement:    20.0,
				MemoryImprovement:     10.0,
				CPUImprovement:        15.0,
				OverallScore:          23.75,
			},
		}

		assert.Equal(t, 10, result.TotalApplied)
		assert.Equal(t, 8, result.Successful)
		assert.Equal(t, 2, result.Failed)
		assert.Equal(t, 5*time.Minute, result.Duration)
		assert.Equal(t, 50.0, result.OverallImprovement.ThroughputImprovement)
	})
}

// TestOptResult tests optimization result structure
func TestOptResult(t *testing.T) {
	t.Run("OptResult_Success", func(t *testing.T) {
		result := OptResult{
			Timestamp:     time.Now(),
			Success:       true,
			Improvement:   25.5,
			BeforeValue:   100.0,
			AfterValue:    125.5,
			MetricsChange: "CPU optimized",
		}

		assert.True(t, result.Success)
		assert.Equal(t, 25.5, result.Improvement)
		assert.Equal(t, 100.0, result.BeforeValue)
		assert.Equal(t, 125.5, result.AfterValue)
		assert.Empty(t, result.ErrorMessage)
	})

	t.Run("OptResult_Failure", func(t *testing.T) {
		result := OptResult{
			Timestamp:    time.Now(),
			Success:      false,
			ErrorMessage: "optimization failed",
		}

		assert.False(t, result.Success)
		assert.NotEmpty(t, result.ErrorMessage)
	})
}

// TestOptimization tests optimization structure
func TestOptimization(t *testing.T) {
	t.Run("Optimization_Creation", func(t *testing.T) {
		opt := Optimization{
			Name:        "Test Optimization",
			Type:        CPUOpt,
			Description: "Test description",
			Priority:    1,
			Enabled:     true,
			Config: map[string]interface{}{
				"param1": "value1",
				"param2": 123,
			},
		}

		assert.Equal(t, "Test Optimization", opt.Name)
		assert.Equal(t, CPUOpt, opt.Type)
		assert.Equal(t, 1, opt.Priority)
		assert.True(t, opt.Enabled)
		assert.NotNil(t, opt.Config)
		assert.Nil(t, opt.Results)
	})

	t.Run("Optimization_WithResults", func(t *testing.T) {
		opt := Optimization{
			Name:    "Test",
			Type:    MemoryOpt,
			Enabled: true,
			Results: &OptResult{
				Success:     true,
				Improvement: 15.0,
			},
		}

		assert.NotNil(t, opt.Results)
		assert.True(t, opt.Results.Success)
		assert.Equal(t, 15.0, opt.Results.Improvement)
	})
}

// TestGCStats tests GC stats structure
func TestGCStats(t *testing.T) {
	t.Run("GCStats_Structure", func(t *testing.T) {
		stats := GCStats{
			NumGC:        100,
			TotalGC:      5 * time.Second,
			PauseTotalNs: 5000000000,
			HeapAlloc:    1024 * 1024 * 50,
			HeapSys:      1024 * 1024 * 100,
			HeapIdle:     1024 * 1024 * 30,
			HeapInuse:    1024 * 1024 * 70,
			StackInuse:   1024 * 1024 * 5,
		}

		assert.Equal(t, uint32(100), stats.NumGC)
		assert.Equal(t, 5*time.Second, stats.TotalGC)
		assert.Equal(t, uint64(5000000000), stats.PauseTotalNs)
		assert.Greater(t, stats.HeapAlloc, uint64(0))
	})
}

// TestReportFunctions tests report generation functions
func TestReportFunctions(t *testing.T) {
	config := PerformanceConfig{
		CPUOptimization:      true,
		MemoryOptimization:   true,
		TargetThroughput:     1000,
		TargetCPUUtilization: 70.0,
		MinCacheHitRate:      0.95,
	}
	po, err := NewPerformanceOptimizer(config)
	require.NoError(t, err)

	t.Run("FormatOptimizationDetails", func(t *testing.T) {
		details := po.formatOptimizationDetails(po.optimizations)

		assert.NotEmpty(t, details)
		assert.Contains(t, details, "Type:")
		assert.Contains(t, details, "Priority:")
	})

	t.Run("GenerateRecommendations_LowImprovement", func(t *testing.T) {
		result := &OptimizationResult{
			OverallImprovement: &OverallImprovement{
				ThroughputImprovement: 5.0,
				LatencyImprovement:    5.0,
				MemoryImprovement:     5.0,
			},
			PostOptimization: &PerformanceMetrics{
				CacheHitRate: 0.85,
			},
		}

		recommendations := po.generateRecommendations(result)

		assert.NotEmpty(t, recommendations)
		assert.Contains(t, recommendations, "CPU optimizations")
		assert.Contains(t, recommendations, "caching strategies")
		assert.Contains(t, recommendations, "memory pool")
	})

	t.Run("GenerateRecommendations_HighImprovement", func(t *testing.T) {
		result := &OptimizationResult{
			OverallImprovement: &OverallImprovement{
				ThroughputImprovement: 25.0,
				LatencyImprovement:    25.0,
				MemoryImprovement:     25.0,
				OverallScore:          25.0,
			},
			PostOptimization: &PerformanceMetrics{
				CacheHitRate: 0.96,
			},
		}

		recommendations := po.generateRecommendations(result)

		assert.NotEmpty(t, recommendations)
		assert.Contains(t, recommendations, "Excellent")
	})

	t.Run("GenerateKeyAchievements", func(t *testing.T) {
		result := &OptimizationResult{
			Successful:   8,
			TotalApplied: 10,
			OverallImprovement: &OverallImprovement{
				ThroughputImprovement: 20.0,
				LatencyImprovement:    20.0,
				MemoryImprovement:     20.0,
				OverallScore:          25.0,
			},
		}

		achievements := po.generateKeyAchievements(result)

		assert.NotEmpty(t, achievements)
		assert.Contains(t, achievements, "throughput improvement")
		assert.Contains(t, achievements, "latency reduction")
		assert.Contains(t, achievements, "memory optimization")
		assert.Contains(t, achievements, "8/10")
	})

	t.Run("EvaluateProductionReadiness_Ready", func(t *testing.T) {
		result := &OptimizationResult{
			PostOptimization: &PerformanceMetrics{
				Throughput:      1500,
				CPUUtilization:  65.0,
				MemoryUsage:     400 * 1024 * 1024,
				CacheHitRate:    0.96,
				ErrorRate:       0.005,
			},
		}

		readiness := po.evaluateProductionReadiness(result)

		assert.Contains(t, readiness, "PRODUCTION READY")
	})

	t.Run("EvaluateProductionReadiness_NotReady", func(t *testing.T) {
		config := PerformanceConfig{
			TargetThroughput:     2000,
			TargetCPUUtilization: 50.0,
			MinCacheHitRate:      0.95,
		}
		po, _ := NewPerformanceOptimizer(config)

		result := &OptimizationResult{
			PostOptimization: &PerformanceMetrics{
				Throughput:      1000, // Below target
				CPUUtilization:  80.0, // Above target
				CacheHitRate:    0.90, // Below target
			},
		}

		readiness := po.evaluateProductionReadiness(result)

		assert.Contains(t, readiness, "OPTIMIZATION NEEDED")
	})
}

// TestConcurrentOptimizations tests thread safety
func TestConcurrentOptimizations(t *testing.T) {
	t.Run("ConcurrentMetricsCollection", func(t *testing.T) {
		config := PerformanceConfig{}
		po, err := NewPerformanceOptimizer(config)
		require.NoError(t, err)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, err := po.collectMetrics()
				assert.NoError(t, err)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("ConcurrentGetOptimizationsByType", func(t *testing.T) {
		config := PerformanceConfig{CPUOptimization: true, MemoryOptimization: true}
		po, err := NewPerformanceOptimizer(config)
		require.NoError(t, err)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				opts := po.getOptimizationsByType(CPUOpt)
				assert.Len(t, opts, 2)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
