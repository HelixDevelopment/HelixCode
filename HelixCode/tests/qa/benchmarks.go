package qa_test

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/memory/providers"
	"dev.helix.code/internal/mocks"
)

// BenchmarkSuite runs comprehensive benchmarks for memory system
type BenchmarkSuite struct {
	ctx           context.Context
	logger        logging.Logger
	benchmarkData *BenchmarkData
}

// BenchmarkData contains data for benchmarking
type BenchmarkData struct {
	VectorSizes     []int
	BatchSizes      []int
	CollectionSizes []int
	QuerySizes      []int
	TestVectors     map[string][]*memory.VectorData
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	ctx := context.Background()
	logger := logging.NewTestLogger("benchmark")

	return &BenchmarkSuite{
		ctx:           ctx,
		logger:        logger,
		benchmarkData: createBenchmarkData(),
	}
}

// RunAllBenchmarks runs all benchmarks
func (bs *BenchmarkSuite) RunAllBenchmarks(b *testing.B) {
	b.Run("VectorStorage", bs.BenchmarkVectorStorage)
	b.Run("VectorSearch", bs.BenchmarkVectorSearch)
	b.Run("BatchOperations", bs.BenchmarkBatchOperations)
	b.Run("ConcurrentOperations", bs.BenchmarkConcurrentOperations)
	b.Run("MemoryUsage", bs.BenchmarkMemoryUsage)
	b.Run("ProviderSwitching", bs.BenchmarkProviderSwitching)
	b.Run("CogneeIntegration", bs.BenchmarkCogneeIntegration)
	b.Run("ConversationManagement", bs.BenchmarkConversationManagement)
}

// createBenchmarkData creates test data for benchmarks
func createBenchmarkData() *BenchmarkData {
	return &BenchmarkData{
		VectorSizes:     []int{256, 512, 1024, 1536, 2048, 4096},
		BatchSizes:      []int{1, 10, 50, 100, 500, 1000},
		CollectionSizes: []int{100, 1000, 10000, 100000},
		QuerySizes:      []int{1, 10, 50, 100, 500},
		TestVectors:     make(map[string][]*memory.VectorData),
	}
}

// generateTestVectors generates test vectors for benchmarking
func (bs *BenchmarkSuite) generateTestVectors(count, size int) []*memory.VectorData {
	vectors := make([]*memory.VectorData, count)
	for i := 0; i < count; i++ {
		vectors[i] = &memory.VectorData{
			ID:     fmt.Sprintf("bench_vector_%d_%d", size, i),
			Vector: generateRandomVector(size),
			Metadata: map[string]interface{}{
				"index":      i,
				"size":       size,
				"created_at": time.Now(),
				"category":   fmt.Sprintf("category_%d", i%10),
				"priority":   fmt.Sprintf("priority_%d", i%5),
			},
			Collection: fmt.Sprintf("benchmark_%d", size),
			Timestamp:  time.Now(),
		}
	}
	return vectors
}

// generateRandomVector generates a random vector for testing
func generateRandomVector(size int) []float64 {
	vector := make([]float64, size)
	for i := 0; i < size; i++ {
		vector[i] = rand.Float64()
	}
	// Normalize vector
	norm := 0.0
	for _, v := range vector {
		norm += v * v
	}
	norm = sqrt(norm)
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}
	return vector
}

// sqrt implements square root
func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// BenchmarkVectorStorage benchmarks vector storage operations
func (bs *BenchmarkSuite) BenchmarkVectorStorage(b *testing.B) {
	provider := mocks.NewMockVectorProvider(b)

	b.Run("SingleVector", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vector := bs.generateTestVectors(1, 1536)[0]
			_ = provider.Store(bs.ctx, []*memory.VectorData{vector})
		}
	})

	b.Run("BatchStorage", func(b *testing.B) {
		batchSizes := bs.benchmarkData.BatchSizes
		for _, batchSize := range batchSizes {
			b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
				vectors := bs.generateTestVectors(batchSize, 1536)

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = provider.Store(bs.ctx, vectors)
				}
			})
		}
	})

	b.Run("DifferentSizes", func(b *testing.B) {
		vectorSizes := bs.benchmarkData.VectorSizes
		for _, size := range vectorSizes {
			b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
				vector := bs.generateTestVectors(1, size)[0]

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = provider.Store(bs.ctx, []*memory.VectorData{vector})
				}
			})
		}
	})
}

// BenchmarkVectorSearch benchmarks vector search operations
func (bs *BenchmarkSuite) BenchmarkVectorSearch(b *testing.B) {
	provider := mocks.NewMockVectorProvider(b)

	// Store test data
	collectionSizes := bs.benchmarkData.CollectionSizes
	for _, collectionSize := range collectionSizes {
		vectors := bs.generateTestVectors(collectionSize, 1536)
		provider.Store(bs.ctx, vectors)

		b.Run(fmt.Sprintf("Collection_%d", collectionSize), func(b *testing.B) {
			queryVector := generateRandomVector(1536)
			query := &memory.VectorQuery{
				Vector:     queryVector,
				Collection: fmt.Sprintf("benchmark_%d", 1536),
				TopK:       10,
				Threshold:  0.7,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = provider.Search(bs.ctx, query)
			}
		})
	}

	b.Run("DifferentTopK", func(b *testing.B) {
		queryVector := generateRandomVector(1536)
		topKs := []int{1, 5, 10, 25, 50, 100}

		for _, topK := range topKs {
			b.Run(fmt.Sprintf("TopK_%d", topK), func(b *testing.B) {
				query := &memory.VectorQuery{
					Vector:     queryVector,
					Collection: "benchmark_1536",
					TopK:       topK,
					Threshold:  0.7,
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = provider.Search(bs.ctx, query)
				}
			})
		}
	})
}

// BenchmarkBatchOperations benchmarks batch operations
func (bs *BenchmarkSuite) BenchmarkBatchOperations(b *testing.B) {
	provider := mocks.NewMockVectorProvider(b)

	b.Run("BatchStore", func(b *testing.B) {
		batchSizes := bs.benchmarkData.BatchSizes
		for _, batchSize := range batchSizes {
			b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					vectors := bs.generateTestVectors(batchSize, 1536)
					_ = provider.Store(bs.ctx, vectors)
				}
			})
		}
	})

	b.Run("BatchRetrieve", func(b *testing.B) {
		// Store test data
		testVectors := bs.generateTestVectors(1000, 1536)
		provider.Store(bs.ctx, testVectors)

		batchSizes := bs.benchmarkData.BatchSizes
		for _, batchSize := range batchSizes {
			if batchSize > len(testVectors) {
				continue
			}

			b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
				ids := make([]string, batchSize)
				for i := 0; i < batchSize; i++ {
					ids[i] = testVectors[i].ID
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = provider.Retrieve(bs.ctx, ids)
				}
			})
		}
	})

	b.Run("BatchSearch", func(b *testing.B) {
		// Store test data
		testVectors := bs.generateTestVectors(1000, 1536)
		provider.Store(bs.ctx, testVectors)

		querySizes := bs.benchmarkData.QuerySizes
		for _, querySize := range querySizes {
			b.Run(fmt.Sprintf("QuerySize_%d", querySize), func(b *testing.B) {
				queryVectors := make([]float64, querySize*1536)
				for i := 0; i < querySize; i++ {
					copy(queryVectors[i*1536:(i+1)*1536], generateRandomVector(1536))
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for j := 0; j < querySize; j++ {
						query := &memory.VectorQuery{
							Vector:     queryVectors[j*1536 : (j+1)*1536],
							Collection: "benchmark_1536",
							TopK:       10,
						}
						_, _ = provider.Search(bs.ctx, query)
					}
				}
			})
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func (bs *BenchmarkSuite) BenchmarkConcurrentOperations(b *testing.B) {
	provider := mocks.NewMockVectorProvider(b)

	b.Run("ConcurrentStore", func(b *testing.B) {
		concurrencyLevels := []int{1, 2, 4, 8, 16, 32}

		for _, concurrency := range concurrencyLevels {
			b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
				b.SetParallelism(concurrency)
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						vector := bs.generateTestVectors(1, 1536)[0]
						provider.Store(bs.ctx, []*memory.VectorData{vector})
					}
				})
			})
		}
	})

	b.Run("ConcurrentSearch", func(b *testing.B) {
		// Store test data
		testVectors := bs.generateTestVectors(1000, 1536)
		provider.Store(bs.ctx, testVectors)

		concurrencyLevels := []int{1, 2, 4, 8, 16, 32}

		for _, concurrency := range concurrencyLevels {
			b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
				b.SetParallelism(concurrency)
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						queryVector := generateRandomVector(1536)
						query := &memory.VectorQuery{
							Vector:     queryVector,
							Collection: "benchmark_1536",
							TopK:       10,
						}
						provider.Search(bs.ctx, query)
					}
				})
			})
		}
	})

	b.Run("MixedOperations", func(b *testing.B) {
		concurrencyLevels := []int{1, 2, 4, 8, 16, 32}

		for _, concurrency := range concurrencyLevels {
			b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
				b.SetParallelism(concurrency)
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						// Randomly choose operation
						op := rand.Intn(3)
						switch op {
						case 0:
							// Store
							vector := bs.generateTestVectors(1, 1536)[0]
							provider.Store(bs.ctx, []*memory.VectorData{vector})
						case 1:
							// Search
							queryVector := generateRandomVector(1536)
							query := &memory.VectorQuery{
								Vector:     queryVector,
								Collection: "benchmark_1536",
								TopK:       10,
							}
							provider.Search(bs.ctx, query)
						case 2:
							// Retrieve
							ids := []string{"bench_vector_1536_0"}
							provider.Retrieve(bs.ctx, ids)
						}
					}
				})
			})
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage
func (bs *BenchmarkSuite) BenchmarkMemoryUsage(b *testing.B) {
	provider := mocks.NewMockVectorProvider(b)

	b.Run("StorageMemory", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Get initial memory usage
			var m1 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			// Store vectors
			vectors := bs.generateTestVectors(1000, 1536)
			provider.Store(bs.ctx, vectors)

			// Get final memory usage
			var m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m2)

			// Calculate memory used
			memoryUsed := m2.Alloc - m1.Alloc
			b.ReportMetric(float64(memoryUsed), "bytes/1000_vectors")
		}
	})

	b.Run("SearchMemory", func(b *testing.B) {
		// Store test data
		testVectors := bs.generateTestVectors(10000, 1536)
		provider.Store(bs.ctx, testVectors)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Get initial memory usage
			var m1 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			// Perform search
			queryVector := generateRandomVector(1536)
			query := &memory.VectorQuery{
				Vector:     queryVector,
				Collection: "benchmark_1536",
				TopK:       100,
			}
			provider.Search(bs.ctx, query)

			// Get final memory usage
			var m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m2)

			// Calculate memory used
			memoryUsed := m2.Alloc - m1.Alloc
			b.ReportMetric(float64(memoryUsed), "bytes/search")
		}
	})
}

// BenchmarkProviderSwitching benchmarks provider switching operations
func (bs *BenchmarkSuite) BenchmarkProviderSwitching(b *testing.B) {
	providerManager := mocks.NewMockVectorProviderManager(b)

	b.Run("ProviderSwitch", func(b *testing.B) {
		providers := []string{"chromadb", "pinecone", "faiss"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			provider := providers[i%len(providers)]
			_ = providerManager.SetActiveProvider(bs.ctx, provider)
		}
	})

	b.Run("ActiveProviderGet", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = providerManager.GetActiveProvider()
		}
	})

	b.Run("ProviderList", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = providerManager.ListProviders()
		}
	})
}

// BenchmarkCogneeIntegration benchmarks Cognee integration
func (bs *BenchmarkSuite) BenchmarkCogneeIntegration(b *testing.B) {
	mockProvider := mocks.NewMockVectorProvider(b)
	mockAPIKeyManager := mocks.NewMockAPIKeyManager(b)

	cogneeIntegration := memory.NewCogneeIntegration(
		mockProvider,
		mockProvider,
		mockAPIKeyManager,
	)

	config := &config.CogneeConfig{
		Enabled: true,
		Mode:    config.CogneeModeLocal,
		Optimization: &config.OptimizationConfig{
			HostAware:     true,
			ResearchBased: true,
			AutoTune:      true,
		},
	}

	_ = cogneeIntegration.Initialize(bs.ctx, config)

	b.Run("CogneeStore", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mem := &memory.MemoryData{
				ID:      fmt.Sprintf("cognee_mem_%d", i),
				Type:    memory.MemoryTypeConversation,
				Content: fmt.Sprintf("Test content %d", i),
				Source:  "cognee_benchmark",
				Metadata: map[string]interface{}{
					"index": i,
				},
				Timestamp: time.Now(),
			}
			_ = cogneeIntegration.StoreMemory(bs.ctx, mem)
		}
	})

	b.Run("CogneeRetrieve", func(b *testing.B) {
		// Store test data
		for i := 0; i < 1000; i++ {
			mem := &memory.MemoryData{
				ID:      fmt.Sprintf("cognee_retrieve_%d", i),
				Type:    memory.MemoryTypeKnowledge,
				Content: fmt.Sprintf("Knowledge content %d", i),
				Source:  "cognee_benchmark",
			}
			_ = cogneeIntegration.StoreMemory(bs.ctx, mem)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := &memory.RetrievalQuery{
				Type:     memory.MemoryTypeKnowledge,
				Keywords: []string{"knowledge"},
				Limit:    10,
			}
			_, _ = cogneeIntegration.RetrieveMemory(bs.ctx, query)
		}
	})

	b.Run("CogneeContext", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cogneeIntegration.GetContext(bs.ctx, "openai", "gpt-4", fmt.Sprintf("session_%d", i))
		}
	})
}

// BenchmarkConversationManagement benchmarks conversation management
func (bs *BenchmarkSuite) BenchmarkConversationManagement(b *testing.B) {
	mockProvider := mocks.NewMockVectorProvider(b)
	conversationManager := memory.NewConversationManager(mockProvider, bs.logger)

	b.Run("AddMessage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			message := &memory.ConversationMessage{
				ID:        fmt.Sprintf("msg_%d", i),
				Role:      "user",
				Content:   fmt.Sprintf("Test message %d", i),
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"index": i,
				},
			}
			_ = conversationManager.AddMessage(bs.ctx, fmt.Sprintf("session_%d", i%100), message)
		}
	})

	b.Run("GetContextWindow", func(b *testing.B) {
		// Add test messages
		sessionID := "benchmark_session"
		for i := 0; i < 1000; i++ {
			message := &memory.ConversationMessage{
				ID:        fmt.Sprintf("msg_%d", i),
				Role:      "user",
				Content:   fmt.Sprintf("Test message %d", i),
				Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			}
			_ = conversationManager.AddMessage(bs.ctx, sessionID, message)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = conversationManager.GetContextWindow(bs.ctx, sessionID, 10)
		}
	})

	b.Run("GetSummary", func(b *testing.B) {
		// Add test messages
		sessionID := "summary_session"
		for i := 0; i < 100; i++ {
			message := &memory.ConversationMessage{
				ID:        fmt.Sprintf("msg_%d", i),
				Role:      []string{"user", "assistant"}[i%2],
				Content:   fmt.Sprintf("Test message %d", i),
				Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			}
			_ = conversationManager.AddMessage(bs.ctx, sessionID, message)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = conversationManager.GetSummary(bs.ctx, sessionID)
		}
	})
}

// BenchmarkSuite runs all QA benchmarks
func BenchmarkSuite(b *testing.B) {
	suite := NewBenchmarkSuite()
	suite.RunAllBenchmarks(b)
}

// LoadTest runs load testing for memory system
func LoadTest(b *testing.B) {
	ctx := context.Background()
	logger := logging.NewTestLogger("load_test")

	provider := mocks.NewMockVectorProvider(b)
	providerManager := mocks.NewMockVectorProviderManager(b)
	memoryManager := mocks.NewMockMemoryManager(b)

	// Initialize
	provider.Initialize(ctx, nil)
	provider.Start(ctx)

	// Load test parameters
	const (
		duration         = 30 * time.Second
		operationsPerSec = 1000
		vectorSize       = 1536
		batchSize        = 100
	)

	operationTicker := time.NewTicker(time.Second / time.Duration(operationsPerSec))
	defer operationTicker.Stop()

	durationTimer := time.NewTimer(duration)
	defer durationTimer.Stop()

	var operations uint64
	var errors uint64
	var mu sync.Mutex

	b.ResetTimer()

Loop:
	for {
		select {
		case <-operationTicker.C:
			// Perform operation in goroutine
			go func() {
				defer func() {
					mu.Lock()
					operations++
					mu.Unlock()
				}()

				// Random operation
				switch rand.Intn(3) {
				case 0:
					// Store
					vectors := make([]*memory.VectorData, batchSize)
					for i := 0; i < batchSize; i++ {
						vectors[i] = &memory.VectorData{
							ID:         fmt.Sprintf("load_test_%d_%d", time.Now().UnixNano(), i),
							Vector:     generateRandomVector(vectorSize),
							Collection: "load_test_collection",
						}
					}
					if err := provider.Store(ctx, vectors); err != nil {
						mu.Lock()
						errors++
						mu.Unlock()
					}
				case 1:
					// Search
					queryVector := generateRandomVector(vectorSize)
					query := &memory.VectorQuery{
						Vector:     queryVector,
						Collection: "load_test_collection",
						TopK:       10,
					}
					if _, err := provider.Search(ctx, query); err != nil {
						mu.Lock()
						errors++
						mu.Unlock()
					}
				case 2:
					// Retrieve
					ids := []string{fmt.Sprintf("load_test_%d", rand.Intn(1000))}
					if _, err := provider.Retrieve(ctx, ids); err != nil {
						mu.Lock()
						errors++
						mu.Unlock()
					}
				}
			}()

		case <-durationTimer.C:
			break Loop
		}
	}

	// Wait for all operations to complete
	time.Sleep(1 * time.Second)

	mu.Lock()
	totalOps := operations
	totalErrs := errors
	mu.Unlock()

	// Report results
	b.ReportMetric(float64(totalOps), "total_operations")
	b.ReportMetric(float64(totalErrs), "total_errors")
	b.ReportMetric(float64(totalOps)/duration.Seconds(), "operations_per_second")
	b.ReportMetric(float64(totalErrs)/float64(totalOps)*100, "error_rate_percent")

	// Ensure load test performance
	performanceOpsPerSec := float64(totalOps) / duration.Seconds()
	expectedOpsPerSec := float64(operationsPerSec)

	require.Greater(b, performanceOpsPerSec, expectedOpsPerSec*0.8,
		"Load test performance should be at least 80%% of expected")

	errorRate := float64(totalErrs) / float64(totalOps)
	require.Less(b, errorRate, 0.01,
		"Error rate should be less than 1%%")
}

// StressTest runs stress testing for memory system
func StressTest(b *testing.B) {
	ctx := context.Background()
	logger := logging.NewTestLogger("stress_test")

	providers := make([]providers.VectorProvider, 5)
	for i := 0; i < 5; i++ {
		providers[i] = mocks.NewMockVectorProvider(b)
		providers[i].Initialize(ctx, nil)
		providers[i].Start(ctx)
	}

	// Stress test parameters
	const (
		duration      = 60 * time.Second
		maxConcurrent = 100
		vectorSize    = 1536
	)

	var wg sync.WaitGroup
	var operations uint64
	var errors uint64
	var mu sync.Mutex

	// Start concurrent goroutines
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			operationTimer := time.NewTicker(time.Millisecond * 10)
			defer operationTimer.Stop()

			durationTimer := time.NewTimer(duration)
			defer durationTimer.Stop()

			for {
				select {
				case <-operationTimer.C:
					// Perform operation
					provider := providers[workerID%len(providers)]
					vector := &memory.VectorData{
						ID:         fmt.Sprintf("stress_%d_%d", workerID, time.Now().UnixNano()),
						Vector:     generateRandomVector(vectorSize),
						Collection: "stress_collection",
					}

					if err := provider.Store(ctx, []*memory.VectorData{vector}); err != nil {
						mu.Lock()
						errors++
						mu.Unlock()
					} else {
						mu.Lock()
						operations++
						mu.Unlock()
					}

				case <-durationTimer.C:
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Report results
	mu.Lock()
	totalOps := operations
	totalErrs := errors
	mu.Unlock()

	b.ReportMetric(float64(totalOps), "total_operations")
	b.ReportMetric(float64(totalErrs), "total_errors")
	b.ReportMetric(float64(totalOps)/duration.Seconds(), "operations_per_second")
	b.ReportMetric(float64(totalErrs)/float64(totalOps)*100, "error_rate_percent")

	// Ensure stress test performance
	performanceOpsPerSec := float64(totalOps) / duration.Seconds()
	require.Greater(b, performanceOpsPerSec, 100.0,
		"Stress test should handle at least 100 ops/sec")

	errorRate := float64(totalErrs) / float64(totalOps)
	require.Less(b, errorRate, 0.05,
		"Error rate should be less than 5%% under stress")
}

// EnduranceTest runs endurance testing for memory system
func EnduranceTest(b *testing.B) {
	ctx := context.Background()
	logger := logging.NewTestLogger("endurance_test")

	provider := mocks.NewMockVectorProvider(b)
	provider.Initialize(ctx, nil)
	provider.Start(ctx)

	// Endurance test parameters
	const (
		duration         = 5 * time.Minute
		operationsPerSec = 100
		vectorSize       = 1536
		reportInterval   = 30 * time.Second
	)

	var wg sync.WaitGroup
	var totalOperations uint64
	var totalErrors uint64
	var mu sync.Mutex

	// Report goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()

		reportTimer := time.NewTicker(reportInterval)
		defer reportTimer.Stop()

		durationTimer := time.NewTimer(duration)
		defer durationTimer.Stop()

		startTime := time.Now()

		for {
			select {
			case <-reportTimer.C:
				mu.Lock()
				ops := totalOperations
				errs := totalErrors
				mu.Unlock()

				elapsed := time.Since(startTime).Seconds()
				opsPerSec := float64(ops) / elapsed
				errRate := float64(errs) / float64(ops) * 100

				b.Logf("Endurance Test Report: Time=%.1fs, Ops=%d, Errs=%d, Ops/sec=%.2f, ErrRate=%.2f%%",
					elapsed, ops, errs, opsPerSec, errRate)

			case <-durationTimer.C:
				return
			}
		}
	}()

	// Operation goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()

		operationTicker := time.NewTicker(time.Second / time.Duration(operationsPerSec))
		defer operationTicker.Stop()

		durationTimer := time.NewTimer(duration)
		defer durationTimer.Stop()

		for {
			select {
			case <-operationTicker.C:
				vector := &memory.VectorData{
					ID:         fmt.Sprintf("endurance_%d", time.Now().UnixNano()),
					Vector:     generateRandomVector(vectorSize),
					Collection: "endurance_collection",
				}

				if err := provider.Store(ctx, []*memory.VectorData{vector}); err != nil {
					mu.Lock()
					totalErrors++
					mu.Unlock()
				} else {
					mu.Lock()
					totalOperations++
					mu.Unlock()
				}

			case <-durationTimer.C:
				return
			}
		}
	}()

	// Wait for completion
	wg.Wait()

	// Final report
	performanceOpsPerSec := float64(totalOperations) / duration.Seconds()
	errorRate := float64(totalErrors) / float64(totalOperations) * 100

	b.Logf("Endurance Test Final Results:")
	b.Logf("  Duration: %v", duration)
	b.Logf("  Total Operations: %d", totalOperations)
	b.Logf("  Total Errors: %d", totalErrors)
	b.Logf("  Operations/Second: %.2f", performanceOpsPerSec)
	b.Logf("  Error Rate: %.2f%%", errorRate)

	// Ensure endurance test requirements
	require.Greater(b, performanceOpsPerSec, 50.0,
		"Endurance test should maintain at least 50 ops/sec")
	require.Less(b, errorRate, 1.0,
		"Error rate should be less than 1%% over long duration")
}
