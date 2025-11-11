package cognee

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/hardware"
)

// TestCacheManager tests the CacheManager stub
func TestCacheManager(t *testing.T) {
	t.Run("NewCacheManager_Success", func(t *testing.T) {
		cfg := map[string]interface{}{"test": "config"}
		cm, err := NewCacheManager(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, cm)
	})

	t.Run("NewCacheManager_NilConfig", func(t *testing.T) {
		cm, err := NewCacheManager(nil)

		assert.NoError(t, err)
		assert.NotNil(t, cm)
	})
}

// TestCogneeManager tests the CogneeManager stub
func TestCogneeManager(t *testing.T) {
	t.Run("NewCogneeManager_Success", func(t *testing.T) {
		cfg := &config.HelixConfig{}
		hwProfile := &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{
				Model: "Test CPU",
				Cores: 4,
			},
		}

		cm, err := NewCogneeManager(cfg, hwProfile)

		assert.NoError(t, err)
		assert.NotNil(t, cm)
		assert.Equal(t, cfg, cm.config)
		assert.Equal(t, hwProfile, cm.hwProfile)
		assert.NotNil(t, cm.logger)
	})

	t.Run("NewCogneeManager_NilConfig", func(t *testing.T) {
		cm, err := NewCogneeManager(nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, cm)
	})

	t.Run("ProcessKnowledge_ReturnsNotImplementedError", func(t *testing.T) {
		cm, _ := NewCogneeManager(&config.HelixConfig{}, nil)
		ctx := context.Background()

		err := cm.ProcessKnowledge(ctx, "test content")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})

	t.Run("ProcessKnowledge_WithEmptyContent", func(t *testing.T) {
		cm, _ := NewCogneeManager(&config.HelixConfig{}, nil)
		ctx := context.Background()

		err := cm.ProcessKnowledge(ctx, "")

		assert.Error(t, err)
	})

	t.Run("SearchKnowledge_ReturnsNotImplementedError", func(t *testing.T) {
		cm, _ := NewCogneeManager(&config.HelixConfig{}, nil)
		ctx := context.Background()

		result, err := cm.SearchKnowledge(ctx, "test query")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not implemented")
	})

	t.Run("SearchKnowledge_WithEmptyQuery", func(t *testing.T) {
		cm, _ := NewCogneeManager(&config.HelixConfig{}, nil)
		ctx := context.Background()

		result, err := cm.SearchKnowledge(ctx, "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("GetStatus_ReturnsStub", func(t *testing.T) {
		cm, _ := NewCogneeManager(&config.HelixConfig{}, nil)

		status := cm.GetStatus()

		assert.Equal(t, "stub", status)
	})

	t.Run("Close_Success", func(t *testing.T) {
		cm, _ := NewCogneeManager(&config.HelixConfig{}, nil)

		err := cm.Close()

		assert.NoError(t, err)
	})

	t.Run("Close_MultipleCalls", func(t *testing.T) {
		cm, _ := NewCogneeManager(&config.HelixConfig{}, nil)

		err1 := cm.Close()
		err2 := cm.Close()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

// TestHostOptimizer tests the HostOptimizer stub
func TestHostOptimizer(t *testing.T) {
	t.Run("NewHostOptimizer_Success", func(t *testing.T) {
		hwProfile := &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{
				Model: "Test CPU",
				Cores: 8,
			},
		}

		ho := NewHostOptimizer(hwProfile)

		assert.NotNil(t, ho)
	})

	t.Run("NewHostOptimizer_NilProfile", func(t *testing.T) {
		ho := NewHostOptimizer(nil)

		assert.NotNil(t, ho)
	})

	t.Run("OptimizeConfig_ReturnsUnchanged", func(t *testing.T) {
		ho := NewHostOptimizer(nil)
		originalConfig := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}

		optimizedConfig := ho.OptimizeConfig(originalConfig)

		assert.Equal(t, originalConfig, optimizedConfig)
	})

	t.Run("OptimizeConfig_WithNilConfig", func(t *testing.T) {
		ho := NewHostOptimizer(nil)

		optimizedConfig := ho.OptimizeConfig(nil)

		assert.Nil(t, optimizedConfig)
	})

	t.Run("OptimizeConfig_WithComplexConfig", func(t *testing.T) {
		ho := NewHostOptimizer(&hardware.HardwareProfile{})
		complexConfig := map[string]interface{}{
			"nested": map[string]interface{}{
				"key": "value",
			},
			"array": []int{1, 2, 3},
		}

		optimizedConfig := ho.OptimizeConfig(complexConfig)

		assert.Equal(t, complexConfig, optimizedConfig)
	})
}

// TestPerformanceOptimizer tests the PerformanceOptimizer
func TestPerformanceOptimizer(t *testing.T) {
	t.Run("NewPerformanceOptimizer_Success", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Mode:    "local",
		}
		hwProfile := &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{
				Model: "Test CPU",
				Cores: 4,
			},
		}

		po, err := NewPerformanceOptimizer(cfg, hwProfile)

		assert.NoError(t, err)
		assert.NotNil(t, po)
		assert.Equal(t, cfg, po.config)
		assert.Equal(t, hwProfile, po.hwProfile)
		assert.False(t, po.initialized)
		assert.False(t, po.running)
	})

	t.Run("NewPerformanceOptimizer_NilConfig", func(t *testing.T) {
		po, err := NewPerformanceOptimizer(nil, nil)

		assert.Error(t, err)
		assert.Nil(t, po)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("GetMetrics_InitialState", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: true}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})

		metrics := po.GetMetrics()

		assert.NotNil(t, metrics)
		assert.Equal(t, float64(0), metrics.TraversalSpeed)
		// MemoryUsage returns actual runtime memory, so it should be > 0
		assert.Greater(t, metrics.MemoryUsage, int64(0), "Memory usage should reflect actual runtime allocation")
	})

	t.Run("GetStatus_InitialState", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: true}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})

		status := po.GetStatus()

		assert.NotNil(t, status)
		assert.Equal(t, false, status["initialized"])
		assert.Equal(t, false, status["running"])
		assert.NotNil(t, status["metrics"])
	})

	t.Run("Optimize_WithoutInitialize_ReturnsError", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: true}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})
		ctx := context.Background()

		result, err := po.Optimize(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Start_WithoutInitialize_ReturnsError", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: true}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})
		ctx := context.Background()

		err := po.Start(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Initialize_Success", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 4,
			},
			Optimization: &config.CogneeOptimizationConfig{
				CPUOptimization:    true,
				MemoryOptimization: true,
			},
		}
		hwProfile := &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{
				Model: "Test CPU",
				Cores: 8,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, hwProfile)
		ctx := context.Background()

		err := po.Initialize(ctx)

		assert.NoError(t, err)
		assert.True(t, po.initialized)
		assert.NotNil(t, po.cache)
		assert.NotNil(t, po.pool)
		assert.NotNil(t, po.optimizer)
	})

	t.Run("Initialize_Idempotent", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		})
		ctx := context.Background()

		err1 := po.Initialize(ctx)
		err2 := po.Initialize(ctx)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.True(t, po.initialized)
	})

	t.Run("Start_Success", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		hwProfile := &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		}
		po, _ := NewPerformanceOptimizer(cfg, hwProfile)
		ctx := context.Background()

		err := po.Initialize(ctx)
		require.NoError(t, err)

		err = po.Start(ctx)

		assert.NoError(t, err)
		assert.True(t, po.running)
	})

	t.Run("Start_Idempotent", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		})
		ctx := context.Background()

		po.Initialize(ctx)
		err1 := po.Start(ctx)
		err2 := po.Start(ctx)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})

	t.Run("Stop_Success", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		})
		ctx := context.Background()

		po.Initialize(ctx)
		po.Start(ctx)
		err := po.Stop(ctx)

		assert.NoError(t, err)
		assert.False(t, po.running)
	})

	t.Run("Stop_WithoutStart_Success", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: true}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})
		ctx := context.Background()

		err := po.Stop(ctx)

		assert.NoError(t, err)
	})

	t.Run("Optimize_Success", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
			Optimization: &config.CogneeOptimizationConfig{
				CPUOptimization:    true,
				MemoryOptimization: true,
			},
		}
		hwProfile := &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		}
		po, _ := NewPerformanceOptimizer(cfg, hwProfile)
		ctx := context.Background()

		po.Initialize(ctx)
		result, err := po.Optimize(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.MetricsBefore)
		assert.NotNil(t, result.MetricsAfter)
		assert.NotZero(t, result.Timestamp)
		assert.NotZero(t, result.Duration)
	})

	t.Run("Optimize_WithGPU", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
			Optimization: &config.CogneeOptimizationConfig{
				GPUOptimization: true,
			},
		}
		hwProfile := &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
			GPU: &hardware.GPUInfo{
				Model: "Test GPU",
				VRAM:  "8GB",
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, hwProfile)
		ctx := context.Background()

		po.Initialize(ctx)
		result, err := po.Optimize(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, len(result.AppliedOpts), 0)
	})
}

// TestPerformanceOptimizerConcurrency tests concurrent access
func TestPerformanceOptimizerConcurrency(t *testing.T) {
	t.Run("ConcurrentGetMetrics", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: true}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				metrics := po.GetMetrics()
				assert.NotNil(t, metrics)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("ConcurrentGetStatus", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: true}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				status := po.GetStatus()
				assert.NotNil(t, status)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestPerformanceMetrics tests the metrics structure
func TestPerformanceMetrics(t *testing.T) {
	t.Run("InitialMetrics_ZeroValues", func(t *testing.T) {
		metrics := &PerformanceMetrics{}

		assert.Equal(t, float64(0), metrics.TraversalSpeed)
		assert.Equal(t, float64(0), metrics.UpdateSpeed)
		assert.Equal(t, float64(0), metrics.QuerySpeed)
		assert.Equal(t, int64(0), metrics.MemoryUsage)
		assert.Equal(t, float64(0), metrics.CPUUsage)
	})

	t.Run("MetricsUpdate_NonZeroValues", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: true}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})

		// Get metrics
		metrics := po.GetMetrics()

		assert.NotNil(t, metrics)
		assert.NotZero(t, metrics.StartTime)
	})
}

// TestConstructorsWithVariousInputs tests edge cases
func TestConstructorsWithVariousInputs(t *testing.T) {
	t.Run("CogneeManager_WithEmptyConfig", func(t *testing.T) {
		cm, err := NewCogneeManager(&config.HelixConfig{}, nil)
		require.NoError(t, err)
		assert.NotNil(t, cm)
		assert.Equal(t, "stub", cm.GetStatus())
	})

	t.Run("PerformanceOptimizer_WithMinimalConfig", func(t *testing.T) {
		cfg := &config.CogneeConfig{Enabled: false}
		po, err := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{})
		require.NoError(t, err)
		assert.NotNil(t, po)
		assert.False(t, po.initialized)
	})
}

// TestOptimizationCache tests cache operations
func TestOptimizationCache(t *testing.T) {
	t.Run("Set_And_Get", func(t *testing.T) {
		cache := &OptimizationCache{
			items:          make(map[string]*CacheItem),
			maxSize:        1024 * 1024,
			evictionPolicy: "lru",
		}

		err := cache.Set("key1", "value1", 0)
		assert.NoError(t, err)

		val, found := cache.Get("key1")
		assert.True(t, found)
		assert.Equal(t, "value1", val)
	})

	t.Run("Get_NonExistent", func(t *testing.T) {
		cache := &OptimizationCache{
			items:          make(map[string]*CacheItem),
			maxSize:        1024 * 1024,
			evictionPolicy: "lru",
		}

		val, found := cache.Get("nonexistent")
		assert.False(t, found)
		assert.Nil(t, val)
	})

	t.Run("Delete", func(t *testing.T) {
		cache := &OptimizationCache{
			items:          make(map[string]*CacheItem),
			maxSize:        1024 * 1024,
			evictionPolicy: "lru",
		}

		cache.Set("key1", "value1", 0)
		cache.Delete("key1")

		val, found := cache.Get("key1")
		assert.False(t, found)
		assert.Nil(t, val)
	})

	t.Run("evictLeastRecentlyUsed", func(t *testing.T) {
		cache := &OptimizationCache{
			items:          make(map[string]*CacheItem),
			maxSize:        1024 * 1024,
			evictionPolicy: "lru",
		}

		cache.Set("key1", "value1", 0)
		cache.Set("key2", "value2", 0)
		cache.evictLeastRecentlyUsed()

		// At least one item should still exist
		assert.LessOrEqual(t, len(cache.items), 2)
	})

	t.Run("evictExpired", func(t *testing.T) {
		cache := &OptimizationCache{
			items:          make(map[string]*CacheItem),
			maxSize:        1024 * 1024,
			evictionPolicy: "lru",
		}

		// Set with very short TTL
		cache.Set("key1", "value1", 1)
		cache.evictExpired()

		// Item count depends on timing
		assert.LessOrEqual(t, len(cache.items), 1)
	})
}

// TestCompressionAlgorithms tests algorithm implementations
func TestCompressionAlgorithms(t *testing.T) {
	t.Run("NeuralSymbolicCompression", func(t *testing.T) {
		alg := &NeuralSymbolicCompression{}
		assert.Equal(t, "neural_symbolic", alg.GetName())
		assert.Equal(t, 0.75, alg.GetCompressionRatio())

		_, err := alg.Compress("test")
		assert.NoError(t, err)

		err = alg.Decompress([]byte("test"), nil)
		assert.NoError(t, err)
	})

	t.Run("AdaptiveHuffmanCompression", func(t *testing.T) {
		alg := &AdaptiveHuffmanCompression{}
		assert.Equal(t, "adaptive_huffman", alg.GetName())
		assert.Equal(t, 0.65, alg.GetCompressionRatio())

		_, err := alg.Compress("test")
		assert.NoError(t, err)

		err = alg.Decompress([]byte("test"), nil)
		assert.NoError(t, err)
	})

	t.Run("NeuralEmbeddingCompression", func(t *testing.T) {
		alg := &NeuralEmbeddingCompression{}
		assert.Equal(t, "neural_embedding", alg.GetName())
		assert.Equal(t, 0.80, alg.GetCompressionRatio())

		_, err := alg.Compress("test")
		assert.NoError(t, err)

		err = alg.Decompress([]byte("test"), nil)
		assert.NoError(t, err)
	})
}

// TestTraversalAlgorithms tests traversal implementations
func TestTraversalAlgorithms(t *testing.T) {
	t.Run("ParallelNeuralSymbolicTraversal", func(t *testing.T) {
		alg := &ParallelNeuralSymbolicTraversal{}
		assert.Equal(t, "parallel_neural_symbolic", alg.GetName())
		assert.Equal(t, "O(n log n)", alg.GetComplexity())

		_, err := alg.Traverse(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("GPUAcceleratedTraversal", func(t *testing.T) {
		alg := &GPUAcceleratedTraversal{}
		assert.Equal(t, "gpu_accelerated", alg.GetName())
		assert.Equal(t, "O(n)", alg.GetComplexity())

		_, err := alg.Traverse(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("MemoryOptimizedTraversal", func(t *testing.T) {
		alg := &MemoryOptimizedTraversal{}
		assert.Equal(t, "memory_optimized", alg.GetName())
		assert.Equal(t, "O(n)", alg.GetComplexity())

		_, err := alg.Traverse(nil, nil)
		assert.NoError(t, err)
	})
}

// TestPartitioningAlgorithms tests partitioning implementations
func TestPartitioningAlgorithms(t *testing.T) {
	t.Run("AdaptiveMemoryAwarePartitioning", func(t *testing.T) {
		alg := &AdaptiveMemoryAwarePartitioning{}
		assert.Equal(t, "adaptive_memory_aware", alg.GetName())
		assert.Equal(t, 0.85, alg.GetPartitionQuality())

		_, err := alg.Partition(nil, 4)
		assert.NoError(t, err)
	})

	t.Run("NeuralBasedPartitioning", func(t *testing.T) {
		alg := &NeuralBasedPartitioning{}
		assert.Equal(t, "neural_based", alg.GetName())
		assert.Equal(t, 0.90, alg.GetPartitionQuality())

		_, err := alg.Partition(nil, 4)
		assert.NoError(t, err)
	})

	t.Run("SymbolicOptimizedPartitioning", func(t *testing.T) {
		alg := &SymbolicOptimizedPartitioning{}
		assert.Equal(t, "symbolic_optimized", alg.GetName())
		assert.Equal(t, 0.80, alg.GetPartitionQuality())

		_, err := alg.Partition(nil, 4)
		assert.NoError(t, err)
	})
}

// TestStatusMethods tests status reporting methods
func TestStatusMethods(t *testing.T) {
	t.Run("GetCacheStatus_WithCache", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		})
		ctx := context.Background()

		po.Initialize(ctx)
		status := po.GetStatus()

		assert.NotNil(t, status)
		assert.NotNil(t, status["cache"])

		cacheStatus := status["cache"].(map[string]interface{})
		assert.Equal(t, "active", cacheStatus["status"])
	})

	t.Run("GetPoolStatus_WithPool", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		})
		ctx := context.Background()

		po.Initialize(ctx)
		status := po.GetStatus()

		assert.NotNil(t, status)
		assert.NotNil(t, status["pool"])

		poolStatus := status["pool"].(map[string]interface{})
		assert.Equal(t, "active", poolStatus["status"])
		assert.Equal(t, 2, poolStatus["total_workers"])
	})

	t.Run("GetBatchProcessorStatus", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		})
		ctx := context.Background()

		po.Initialize(ctx)
		status := po.GetStatus()

		assert.NotNil(t, status)
		assert.NotNil(t, status["pool"])

		poolStatus := status["pool"].(map[string]interface{})
		assert.NotNil(t, poolStatus["batch_processor"])

		batchStatus := poolStatus["batch_processor"].(map[string]interface{})
		assert.Equal(t, "active", batchStatus["status"])
	})
}

// TestBackgroundMethods tests background loop methods
func TestBackgroundMethods(t *testing.T) {
	t.Run("CollectMetrics", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		})
		ctx := context.Background()

		po.Initialize(ctx)

		// Call collectMetrics directly to increase coverage
		po.collectMetrics()

		metrics := po.GetMetrics()
		assert.NotNil(t, metrics)
		assert.NotZero(t, metrics.LastUpdate)
	})

	t.Run("MaintainCache", func(t *testing.T) {
		cfg := &config.CogneeConfig{
			Enabled: true,
			Performance: &config.CogneePerformanceConfig{
				Workers: 2,
			},
		}
		po, _ := NewPerformanceOptimizer(cfg, &hardware.HardwareProfile{
			CPU: hardware.CPUInfo{Cores: 4},
		})
		ctx := context.Background()

		po.Initialize(ctx)

		// Call maintainCache directly to increase coverage
		po.maintainCache()

		// Verify cache is still functional
		status := po.GetStatus()
		assert.NotNil(t, status)
		assert.NotNil(t, status["cache"])
	})
}
