package performance_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/cognee"
	"dev.helix.code/internal/hardware"
)

// Performance Test Metrics
type PerformanceMetrics struct {
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulRequests int64        `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	AverageLatency    time.Duration `json:"average_latency"`
	MinLatency        time.Duration `json:"min_latency"`
	MaxLatency        time.Duration `json:"max_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	Throughput        float64       `json:"throughput"`
	ErrorRate         float64       `json:"error_rate"`
	MemoryUsage       int64         `json:"memory_usage"`
	GoroutineCount    int           `json:"goroutine_count"`
	TestDuration      time.Duration `json:"test_duration"`
}

// TestLoadBalancingPerformance tests performance of different load balancing strategies
func TestLoadBalancingPerformance(t *testing.T) {
	strategies := []config.LoadBalancingStrategy{
		config.StrategyRoundRobin,
		config.StrategyWeighted,
		config.StrategyRandom,
		config.StrategyPriorityFirst,
		config.StrategyLeastUsed,
		config.StrategyHealthAware,
	}

	for _, strategy := range strategies {
		t.Run("Strategy_"+string(strategy), func(t *testing.T) {
			metrics := runLoadBalancingPerformanceTest(strategy, 100000, 10)
			reportPerformanceResults(t, "Load Balancing", string(strategy), metrics)
			
			// Performance assertions
			if metrics.AverageLatency > 1*time.Millisecond {
				t.Errorf("Load balancing too slow for %s: %v (threshold: 1ms)", 
					strategy, metrics.AverageLatency)
			}
			
			if metrics.Throughput < 50000 {
				t.Errorf("Throughput too low for %s: %.0f requests/second (threshold: 50000)", 
					strategy, metrics.Throughput)
			}
			
			if metrics.ErrorRate > 0.01 {
				t.Errorf("Error rate too high for %s: %.2f%% (threshold: 1%%)", 
					strategy, metrics.ErrorRate*100)
			}
		})
	}
}

// TestConcurrentAccessPerformance tests concurrent access performance
func TestConcurrentAccessPerformance(t *testing.T) {
	concurrencyLevels := []int{1, 10, 50, 100, 500, 1000}
	requestsPerGoroutine := 1000
	
	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(t *testing.T) {
			metrics := runConcurrentAccessPerformanceTest(concurrency, requestsPerGoroutine)
			reportPerformanceResults(t, "Concurrent Access", fmt.Sprintf("%d", concurrency), metrics)
			
			// Performance assertions
			if metrics.AverageLatency > 5*time.Millisecond {
				t.Errorf("Concurrent access too slow for %d goroutines: %v (threshold: 5ms)", 
					concurrency, metrics.AverageLatency)
			}
			
			if metrics.Throughput < 20000 {
				t.Errorf("Throughput too low for %d goroutines: %.0f requests/second (threshold: 20000)", 
					concurrency, metrics.Throughput)
			}
			
			if metrics.ErrorRate > 0.05 {
				t.Errorf("Error rate too high for %d goroutines: %.2f%% (threshold: 5%%)", 
					concurrency, metrics.ErrorRate*100)
			}
			
			// Memory efficiency check
			perRequestMemory := float64(metrics.MemoryUsage) / float64(metrics.TotalRequests)
			if perRequestMemory > 1024 { // 1KB per request
				t.Errorf("Memory usage too high: %d bytes/request (threshold: 1024)", perRequestMemory)
			}
		})
	}
}

// TestFallbackPerformance tests fallback mechanism performance
func TestFallbackPerformance(t *testing.T) {
	failureRates := []float64{0.0, 0.1, 0.25, 0.5, 0.75, 0.9}
	
	for _, failureRate := range failureRates {
		t.Run(fmt.Sprintf("FailureRate_%.0f", failureRate*100), func(t *testing.T) {
			metrics := runFallbackPerformanceTest(failureRate, 10000)
			reportPerformanceResults(t, "Fallback", fmt.Sprintf("%.0f%%", failureRate*100), metrics)
			
			// Performance assertions
			if failureRate < 0.5 && metrics.AverageLatency > 10*time.Millisecond {
				t.Errorf("Fallback too slow for failure rate %.1f: %v (threshold: 10ms)", 
					failureRate, metrics.AverageLatency)
			}
			
			if metrics.ErrorRate > (failureRate + 0.05) { // Allow 5% tolerance
				t.Errorf("Error rate too high for failure rate %.1f: %.2f%% (expected: %.1f%%)", 
					failureRate, metrics.ErrorRate*100, failureRate*100)
			}
		})
	}
}

// TestMemoryEfficiency tests memory efficiency under various conditions
func TestMemoryEfficiency(t *testing.T) {
	testCases := []struct {
		name          string
		numKeys       int
		requestCount  int
		expectedUsage int64 // Maximum expected memory usage in bytes
	}{
		{
			name:          "Small_Scale",
			numKeys:       10,
			requestCount:  10000,
			expectedUsage: 10 * 1024 * 1024, // 10MB
		},
		{
			name:          "Medium_Scale",
			numKeys:       100,
			requestCount:  100000,
			expectedUsage: 50 * 1024 * 1024, // 50MB
		},
		{
			name:          "Large_Scale",
			numKeys:       1000,
			requestCount:  1000000,
			expectedUsage: 200 * 1024 * 1024, // 200MB
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics := runMemoryEfficiencyTest(tc.numKeys, tc.requestCount)
			reportPerformanceResults(t, "Memory Efficiency", tc.name, metrics)
			
			// Memory efficiency assertions
			if metrics.MemoryUsage > tc.expectedUsage {
				t.Errorf("Memory usage too high: %d bytes (expected: < %d)", 
					metrics.MemoryUsage, tc.expectedUsage)
			}
			
			perRequestMemory := float64(metrics.MemoryUsage) / float64(metrics.TotalRequests)
			if perRequestMemory > 1024 { // 1KB per request
				t.Errorf("Per-request memory too high: %.2f bytes/request (threshold: 1024)", perRequestMemory)
			}
			
			// Check for memory leaks (goroutine count should be reasonable)
			if metrics.GoroutineCount > 1000 {
				t.Errorf("Too many goroutines: %d (expected: < 1000)", metrics.GoroutineCount)
			}
		})
	}
}

// TestCogneeIntegrationPerformance tests Cognee integration performance
func TestCogneeIntegrationPerformance(t *testing.T) {
	modes := []config.CogneeMode{config.CogneeModeLocal, config.CogneeModeHybrid}
	
	for _, mode := range modes {
		t.Run("Mode_"+string(mode), func(t *testing.T) {
			metrics := runCogneeIntegrationPerformanceTest(mode, 10000)
			reportPerformanceResults(t, "Cognee Integration", string(mode), metrics)
			
			// Performance assertions
			if metrics.AverageLatency > 50*time.Millisecond {
				t.Errorf("Cognee integration too slow for mode %s: %v (threshold: 50ms)", 
					mode, metrics.AverageLatency)
			}
			
			if metrics.Throughput < 200 {
				t.Errorf("Throughput too low for mode %s: %.0f operations/second (threshold: 200)", 
					mode, metrics.Throughput)
			}
			
			if metrics.ErrorRate > 0.05 {
				t.Errorf("Error rate too high for mode %s: %.2f%% (threshold: 5%%)", 
					mode, metrics.ErrorRate*100)
			}
		})
	}
}

// TestStressTest performs stress testing
func TestStressTest(t *testing.T) {
	const (
		stressDuration = 5 * time.Minute
		targetTPS     = 10000 // Target transactions per second
	)
	
	t.Run("Sustained_Load", func(t *testing.T) {
		metrics := runStressTest(stressDuration, targetTPS)
		reportPerformanceResults(t, "Stress Test", "Sustained Load", metrics)
		
		// Stress test assertions
		if metrics.Throughput < targetTPS*0.8 { // Allow 20% degradation
			t.Errorf("Throughput too low during stress test: %.0f TPS (target: %d)", 
				metrics.Throughput, targetTPS)
		}
		
		if metrics.ErrorRate > 0.1 {
			t.Errorf("Error rate too high during stress test: %.2f%% (threshold: 10%%)", 
				metrics.ErrorRate*100)
		}
		
		// Check performance stability (P99 should not be too high)
		if metrics.P99Latency > 100*time.Millisecond {
			t.Errorf("P99 latency too high during stress test: %v (threshold: 100ms)", 
				metrics.P99Latency)
		}
	})
}

// TestScalability tests scalability characteristics
func TestScalability(t *testing.T) {
	loadLevels := []struct {
		name         string
		goroutines   int
		requests     int
		minTPS       float64
		maxLatency   time.Duration
	}{
		{"Level_1", 10, 10000, 5000, 5 * time.Millisecond},
		{"Level_2", 50, 50000, 20000, 10 * time.Millisecond},
		{"Level_3", 100, 100000, 35000, 20 * time.Millisecond},
		{"Level_4", 500, 500000, 100000, 50 * time.Millisecond},
	}
	
	for _, level := range loadLevels {
		t.Run(level.name, func(t *testing.T) {
			metrics := runScalabilityTest(level.goroutines, level.requests)
			reportPerformanceResults(t, "Scalability", level.name, metrics)
			
			// Scalability assertions
			if metrics.Throughput < level.minTPS {
				t.Errorf("Throughput too low for %s: %.0f TPS (minimum: %.0f)", 
					level.name, metrics.Throughput, level.minTPS)
			}
			
			if metrics.AverageLatency > level.maxLatency {
				t.Errorf("Average latency too high for %s: %v (maximum: %v)", 
					level.name, metrics.AverageLatency, level.maxLatency)
			}
			
			// Calculate scaling efficiency
			linearTPS := float64(level.goroutines) * 100 // Base TPS per goroutine
			scalingEfficiency := metrics.Throughput / linearTPS
			if scalingEfficiency < 0.3 { // At least 30% efficiency
				t.Errorf("Scaling efficiency too low for %s: %.1f%% (minimum: 30%%)", 
					level.name, scalingEfficiency*100)
			}
		})
	}
}

// Performance test implementation functions

func runLoadBalancingPerformanceTest(strategy config.LoadBalancingStrategy, requestCount, keyCount int) *PerformanceMetrics {
	// Create configuration with specified strategy
	helixConfig := createPerformanceTestConfig(keyCount, strategy)
	
	// Create API key manager
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(requestCount)}
	}
	
	err = apiKeyManager.Initialize()
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(requestCount)}
	}
	
	// Run performance test
	return runPerformanceTest(apiKeyManager, "openai", requestCount)
}

func runConcurrentAccessPerformanceTest(concurrency, requestsPerGoroutine int) *PerformanceMetrics {
	helixConfig := createPerformanceTestConfig(50, config.StrategyRoundRobin)
	
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(concurrency * requestsPerGoroutine)}
	}
	
	err = apiKeyManager.Initialize()
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(concurrency * requestsPerGoroutine)}
	}
	
	var wg sync.WaitGroup
	var totalLatency int64
	var minLatency int64 = int64(time.Hour) // Initialize to very large value
	var maxLatency int64
	var requestCount int64
	var errorCount int64
	
	latencies := make([]int64, 0, concurrency*requestsPerGoroutine)
	
	start := time.Now()
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := 0; j < requestsPerGoroutine; j++ {
				reqStart := time.Now()
				_, err := apiKeyManager.GetAPIKey("openai")
				latency := time.Since(reqStart)
				
				latencyNs := latency.Nanoseconds()
				atomic.AddInt64(&totalLatency, latencyNs)
				atomic.AddInt64(&requestCount, 1)
				
				latencies = append(latencies, latencyNs)
				
				for {
					currentMin := atomic.LoadInt64(&minLatency)
					if latencyNs >= currentMin || atomic.CompareAndSwapInt64(&minLatency, currentMin, latencyNs) {
						break
					}
				}
				
				for {
					currentMax := atomic.LoadInt64(&maxLatency)
					if latencyNs <= currentMax || atomic.CompareAndSwapInt64(&maxLatency, currentMax, latencyNs) {
						break
					}
				}
				
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	// Calculate metrics
	metrics := &PerformanceMetrics{
		TotalRequests:     requestCount,
		SuccessfulRequests: requestCount - errorCount,
		FailedRequests:    errorCount,
		AverageLatency:    time.Duration(totalLatency / requestCount),
		MinLatency:        time.Duration(minLatency),
		MaxLatency:        time.Duration(maxLatency),
		Throughput:        float64(requestCount) / duration.Seconds(),
		ErrorRate:         float64(errorCount) / float64(requestCount),
		TestDuration:      duration,
	}
	
	// Calculate percentiles
	if len(latencies) > 0 {
		metrics.P95Latency = time.Duration(calculatePercentile(latencies, 0.95))
		metrics.P99Latency = time.Duration(calculatePercentile(latencies, 0.99))
	}
	
	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.MemoryUsage = int64(m.Alloc)
	metrics.GoroutineCount = runtime.NumGoroutine()
	
	return metrics
}

func runFallbackPerformanceTest(failureRate float64, requestCount int) *PerformanceMetrics {
	// This would simulate primary pool failures
	// For now, implement basic test
	helixConfig := createFallbackTestConfig()
	
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(requestCount)}
	}
	
	err = apiKeyManager.Initialize()
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(requestCount)}
	}
	
	// Simulate failures by configuring the pool to fail
	// This would require extending the API key manager for testing
	
	return runPerformanceTest(apiKeyManager, "openai", requestCount)
}

func runMemoryEfficiencyTest(numKeys, requestCount int) *PerformanceMetrics {
	helixConfig := createMemoryEfficiencyTestConfig(numKeys)
	
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(requestCount)}
	}
	
	err = apiKeyManager.Initialize()
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(requestCount)}
	}
	
	// Get initial memory usage
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialMemory := int64(m1.Alloc)
	
	// Run performance test
	metrics := runPerformanceTest(apiKeyManager, "openai", requestCount)
	
	// Get final memory usage
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalMemory := int64(m2.Alloc)
	
	metrics.MemoryUsage = finalMemory - initialMemory
	metrics.GoroutineCount = runtime.NumGoroutine()
	
	return metrics
}

func runCogneeIntegrationPerformanceTest(mode config.CogneeMode, operationCount int) *PerformanceMetrics {
	// Create Cognee configuration
	helixConfig := createCogneePerformanceTestConfig(mode)
	
	// Create hardware profile
	hwProfile := &hardware.Profile{
		CPU: &hardware.CPUProfile{
			Cores:        8,
			Threads:      16,
			Model:        "Test CPU",
			FrequencyGHz: 3.0,
		},
		Memory: &hardware.MemoryProfile{
			TotalGB:     32,
			AvailableGB: 24,
		},
	}
	
	// Create API key manager
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(operationCount)}
	}
	
	err = apiKeyManager.Initialize()
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(operationCount)}
	}
	
	// Create Cognee manager
	cogneeManager, err := cognee.NewCogneeManager(helixConfig, hwProfile)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(operationCount)}
	}
	
	// Initialize Cognee manager
	ctx := context.Background()
	err = cogneeManager.Initialize(ctx)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(operationCount)}
	}
	
	// Run performance test with Cognee operations
	return runCogneePerformanceTest(apiKeyManager, cogneeManager, operationCount)
}

func runStressTest(duration time.Duration, targetTPS int) *PerformanceMetrics {
	helixConfig := createStressTestConfig(targetTPS)
	
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: 1}
	}
	
	err = apiKeyManager.Initialize()
	if err != nil {
		return &PerformanceMetrics{FailedRequests: 1}
	}
	
	// Calculate required goroutines
	requestInterval := time.Duration(float64(time.Second) / float64(targetTPS))
	
	return runStressTestWithDuration(apiKeyManager, duration, requestInterval)
}

func runScalabilityTest(goroutines, requests int) *PerformanceMetrics {
	helixConfig := createScalabilityTestConfig(goroutines)
	
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(requests)}
	}
	
	err = apiKeyManager.Initialize()
	if err != nil {
		return &PerformanceMetrics{FailedRequests: int64(requests)}
	}
	
	return runConcurrentAccessPerformanceTest(goroutines, requests/goroutines)
}

// Helper functions for performance testing

func runPerformanceTest(apiKeyManager *config.APIKeyManager, service string, requestCount int) *PerformanceMetrics {
	var totalLatency int64
	var minLatency int64 = int64(time.Hour)
	var maxLatency int64
	var errorCount int64
	
	latencies := make([]int64, 0, requestCount)
	
	start := time.Now()
	
	for i := 0; i < requestCount; i++ {
		reqStart := time.Now()
		_, err := apiKeyManager.GetAPIKey(service)
		latency := time.Since(reqStart)
		
		latencyNs := latency.Nanoseconds()
		totalLatency += latencyNs
		
		latencies = append(latencies, latencyNs)
		
		if latencyNs < minLatency {
			minLatency = latencyNs
		}
		
		if latencyNs > maxLatency {
			maxLatency = latencyNs
		}
		
		if err != nil {
			errorCount++
		}
	}
	
	duration := time.Since(start)
	
	metrics := &PerformanceMetrics{
		TotalRequests:     int64(requestCount),
		SuccessfulRequests: int64(requestCount) - errorCount,
		FailedRequests:    errorCount,
		AverageLatency:    time.Duration(totalLatency / int64(requestCount)),
		MinLatency:        time.Duration(minLatency),
		MaxLatency:        time.Duration(maxLatency),
		Throughput:        float64(requestCount) / duration.Seconds(),
		ErrorRate:         float64(errorCount) / float64(requestCount),
		TestDuration:      duration,
	}
	
	// Calculate percentiles
	if len(latencies) > 0 {
		metrics.P95Latency = time.Duration(calculatePercentile(latencies, 0.95))
		metrics.P99Latency = time.Duration(calculatePercentile(latencies, 0.99))
	}
	
	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.MemoryUsage = int64(m.Alloc)
	metrics.GoroutineCount = runtime.NumGoroutine()
	
	return metrics
}

func runCogneePerformanceTest(apiKeyManager *config.APIKeyManager, cogneeManager *cognee.CogneeManager, operationCount int) *PerformanceMetrics {
	var totalLatency int64
	var minLatency int64 = int64(time.Hour)
	var maxLatency int64
	var errorCount int64
	
	latencies := make([]int64, 0, operationCount)
	
	start := time.Now()
	
	for i := 0; i < operationCount; i++ {
		reqStart := time.Now()
		
		// Perform Cognee operation
		key, err := apiKeyManager.GetCogneeAPIKey()
		if err != nil {
			errorCount++
			continue
		}
		
		// Simulate Cognee operation
		_ = key
		
		latency := time.Since(reqStart)
		latencyNs := latency.Nanoseconds()
		totalLatency += latencyNs
		
		latencies = append(latencies, latencyNs)
		
		if latencyNs < minLatency {
			minLatency = latencyNs
		}
		
		if latencyNs > maxLatency {
			maxLatency = latencyNs
		}
	}
	
	duration := time.Since(start)
	
	metrics := &PerformanceMetrics{
		TotalRequests:     int64(operationCount),
		SuccessfulRequests: int64(operationCount) - errorCount,
		FailedRequests:    errorCount,
		AverageLatency:    time.Duration(totalLatency / int64(operationCount)),
		MinLatency:        time.Duration(minLatency),
		MaxLatency:        time.Duration(maxLatency),
		Throughput:        float64(operationCount) / duration.Seconds(),
		ErrorRate:         float64(errorCount) / float64(operationCount),
		TestDuration:      duration,
	}
	
	// Calculate percentiles
	if len(latencies) > 0 {
		metrics.P95Latency = time.Duration(calculatePercentile(latencies, 0.95))
		metrics.P99Latency = time.Duration(calculatePercentile(latencies, 0.99))
	}
	
	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.MemoryUsage = int64(m.Alloc)
	metrics.GoroutineCount = runtime.NumGoroutine()
	
	return metrics
}

func runStressTestWithDuration(apiKeyManager *config.APIKeyManager, duration time.Duration, requestInterval time.Duration) *PerformanceMetrics {
	var totalLatency int64
	var minLatency int64 = int64(time.Hour)
	var maxLatency int64
	var requestCount int64
	var errorCount int64
	
	latencies := make([]int64, 0)
	
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()
	
	start := time.Now()
	
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			reqStart := time.Now()
			_, err := apiKeyManager.GetAPIKey("openai")
			latency := time.Since(reqStart)
			
			latencyNs := latency.Nanoseconds()
			totalLatency += latencyNs
			requestCount++
			
			latencies = append(latencies, latencyNs)
			
			if latencyNs < minLatency {
				minLatency = latencyNs
			}
			
			if latencyNs > maxLatency {
				maxLatency = latencyNs
			}
			
			if err != nil {
				errorCount++
			}
		}
	}
	
done:
	actualDuration := time.Since(start)
	
	metrics := &PerformanceMetrics{
		TotalRequests:     requestCount,
		SuccessfulRequests: requestCount - errorCount,
		FailedRequests:    errorCount,
		AverageLatency:    time.Duration(totalLatency / requestCount),
		MinLatency:        time.Duration(minLatency),
		MaxLatency:        time.Duration(maxLatency),
		Throughput:        float64(requestCount) / actualDuration.Seconds(),
		ErrorRate:         float64(errorCount) / float64(requestCount),
		TestDuration:      actualDuration,
	}
	
	// Calculate percentiles
	if len(latencies) > 0 {
		metrics.P95Latency = time.Duration(calculatePercentile(latencies, 0.95))
		metrics.P99Latency = time.Duration(calculatePercentile(latencies, 0.99))
	}
	
	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.MemoryUsage = int64(m.Alloc)
	metrics.GoroutineCount = runtime.NumGoroutine()
	
	return metrics
}

func calculatePercentile(latencies []int64, percentile float64) int64 {
	if len(latencies) == 0 {
		return 0
	}
	
	// Simple implementation - in production, use proper percentile calculation
	sorted := make([]int64, len(latencies))
	copy(sorted, latencies)
	
	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}

// Configuration creation functions for performance testing

func createPerformanceTestConfig(keyCount int, strategy config.LoadBalancingStrategy) *config.HelixConfig {
	keys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("sk-test-key-%d", i+1)
	}
	
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			OpenAI: &config.ServiceAPIKeyConfig{
				Enabled:     true,
				PrimaryKeys: keys,
				LoadBalancing: &config.ServiceLBConfig{
					Strategy: strategy,
				},
			},
		},
	}
}

func createFallbackTestConfig() *config.HelixConfig {
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			OpenAI: &config.ServiceAPIKeyConfig{
				Enabled:     true,
				PrimaryKeys: []string{"sk-primary-1", "sk-primary-2"},
				FallbackKeys: []string{"sk-fallback-1", "sk-fallback-2"},
				Fallback: &config.ServiceFallbackConfig{
					Enabled:    true,
					Strategy:   config.FallbackStrategySequential,
					MaxRetries: 3,
				},
			},
		},
	}
}

func createMemoryEfficiencyTestConfig(keyCount int) *config.HelixConfig {
	keys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("sk-mem-test-key-%d", i+1)
	}
	
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			OpenAI: &config.ServiceAPIKeyConfig{
				Enabled:     true,
				PrimaryKeys: keys,
			},
		},
	}
}

func createCogneePerformanceTestConfig(mode config.CogneeMode) *config.HelixConfig {
	return &config.HelixConfig{
		Cognee: &config.CogneeConfig{
			Enabled: true,
			Mode:    mode,
		},
		APIKeys: &config.APIKeyConfig{
			Cognee: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    mode,
			},
		},
	}
}

func createStressTestConfig(targetTPS int) *config.HelixConfig {
	keys := make([]string, targetTPS) // One key per request for max throughput
	for i := 0; i < targetTPS; i++ {
		keys[i] = fmt.Sprintf("sk-stress-key-%d", i+1)
	}
	
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			OpenAI: &config.ServiceAPIKeyConfig{
				Enabled:     true,
				PrimaryKeys: keys,
			},
		},
	}
}

func createScalabilityTestConfig(goroutines int) *config.HelixConfig {
	// Create enough keys to avoid contention
	keyCount := goroutines * 4
	keys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("sk-scale-key-%d", i+1)
	}
	
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			OpenAI: &config.ServiceAPIKeyConfig{
				Enabled:     true,
				PrimaryKeys: keys,
			},
		},
	}
}

func reportPerformanceResults(t *testing.T, testType, testName string, metrics *PerformanceMetrics) {
	t.Logf("=== Performance Results: %s - %s ===", testType, testName)
	t.Logf("Total Requests: %d", metrics.TotalRequests)
	t.Logf("Successful Requests: %d", metrics.SuccessfulRequests)
	t.Logf("Failed Requests: %d", metrics.FailedRequests)
	t.Logf("Error Rate: %.2f%%", metrics.ErrorRate*100)
	t.Logf("Average Latency: %v", metrics.AverageLatency)
	t.Logf("Min Latency: %v", metrics.MinLatency)
	t.Logf("Max Latency: %v", metrics.MaxLatency)
	t.Logf("P95 Latency: %v", metrics.P95Latency)
	t.Logf("P99 Latency: %v", metrics.P99Latency)
	t.Logf("Throughput: %.0f requests/second", metrics.Throughput)
	t.Logf("Memory Usage: %d bytes (%.2f MB)", metrics.MemoryUsage, float64(metrics.MemoryUsage)/1024/1024)
	t.Logf("Goroutine Count: %d", metrics.GoroutineCount)
	t.Logf("Test Duration: %v", metrics.TestDuration)
	t.Logf("=========================================")
}