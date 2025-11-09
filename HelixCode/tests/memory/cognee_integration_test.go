package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/memory/providers"
	mocks "dev.helix.code/internal/mocks"
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
)

// CogneeIntegrationTestSuite tests the Cognee integration functionality
type CogneeIntegrationTestSuite struct {
	suite.Suite
	ctx                context.Context
	logger             logging.Logger
	mockProvider       *mocks.MockVectorProvider
	mockAPIKeyManager  *mocks.MockAPIKeyManager
	config            *config.CogneeConfig
	cogneeIntegration  *memory.CogneeIntegration
}

// SetupSuite initializes the test suite
func (suite *CogneeIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.logger = logging.NewTestLogger("cognee_test")
	suite.mockProvider = mocks.NewMockVectorProvider(suite.T())
	suite.mockAPIKeyManager = mocks.NewMockAPIKeyManager(suite.T())
	
	suite.config = &config.CogneeConfig{
		Enabled: true,
		Mode:    config.CogneeModeLocal,
		Host:    "localhost",
		Port:    8000,
		Optimization: &config.OptimizationConfig{
			HostAware:      true,
			ResearchBased:  true,
			AutoTune:       true,
			TuneInterval:   time.Hour,
		},
		Fallback: &config.FallbackConfig{
			Enabled:    true,
			Strategy:   "sequential",
			Providers:  []string{"chromadb", "faiss"},
			Timeout:    10 * time.Second,
			RetryCount: 3,
		},
		Security: &config.SecurityConfig{
			Encryption:      true,
			Authentication: true,
			Authorization:  true,
		},
		Performance: &config.PerformanceConfig{
			BatchSize:       32,
			MaxConcurrency:  10,
			CacheSize:      1000,
			Prefetch:       true,
			AsyncProcessing: true,
		},
	}
	
	suite.cogneeIntegration = memory.NewCogneeIntegration(
		suite.mockProvider,
		suite.mockProvider, // Using same mock for simplicity
		suite.mockAPIKeyManager,
	)
}

// TearDownSuite cleans up the test suite
func (suite *CogneeIntegrationTestSuite) TearDownSuite() {
	suite.cogneeIntegration.Shutdown(suite.ctx)
}

// TestCogneeInitialization tests Cognee integration initialization
func (suite *CogneeIntegrationTestSuite) TestCogneeInitialization() {
	tests := []struct {
		name    string
		config  *config.CogneeConfig
		wantErr bool
	}{
		{
			name:   "Valid local config",
			config: suite.config,
			wantErr: false,
		},
		{
			name: "Disabled config",
			config: &config.CogneeConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "Invalid config - nil",
			config: nil,
			wantErr: true,
		},
		{
			name: "Hybrid mode config",
			config: &config.CogneeConfig{
				Enabled: true,
				Mode:    config.CogneeModeHybrid,
				Host:    "localhost",
				Port:    8000,
			},
			wantErr: false,
		},
		{
			name: "Cloud mode config",
			config: &config.CogneeConfig{
				Enabled: true,
				Mode:    config.CogneeModeCloud,
				RemoteAPI: &config.RemoteAPIConfig{
					ServiceEndpoint: "https://api.cognee.ai",
					APIKey:          "test-key",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ci := memory.NewCogneeIntegration(
				suite.mockProvider,
				suite.mockProvider,
				suite.mockAPIKeyManager,
			)
			
			err := ci.Initialize(suite.ctx, tt.config)
			
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}

// TestStoreMemory tests memory storage with Cognee optimization
func (suite *CogneeIntegrationTestSuite) TestStoreMemory() {
	tests := []struct {
		name     string
		memory   *memory.MemoryData
		setupMock func()
		wantErr  bool
	}{
		{
			name: "Valid memory data",
			memory: &memory.MemoryData{
				ID:      "test_memory_1",
				Type:    memory.MemoryTypeConversation,
				Content: "Test conversation content",
				Source:  "test",
				Metadata: map[string]interface{}{
					"user_id": "user123",
					"session_id": "session456",
				},
				Timestamp: time.Now(),
			},
			setupMock: func() {
				suite.mockProvider.On("Store", suite.ctx, mock.AnythingOfType("[]*memory.VectorData")).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Memory data with embeddings",
			memory: &memory.MemoryData{
				ID:       "test_memory_2",
				Type:     memory.MemoryTypeKnowledge,
				Content:  "Test knowledge content",
				Source:   "test",
				Embedding: []float64{0.1, 0.2, 0.3, 0.4},
				Metadata: map[string]interface{}{
					"category": "AI",
					"tags":    []string{"machine-learning", "neural-networks"},
				},
				Timestamp: time.Now(),
			},
			setupMock: func() {
				suite.mockProvider.On("Store", suite.ctx, mock.AnythingOfType("[]*memory.VectorData")).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Invalid memory data - nil",
			memory: nil,
			setupMock: func() {
				// No mock setup expected
			},
			wantErr: true,
		},
		{
			name: "Provider store error",
			memory: &memory.MemoryData{
				ID:      "test_memory_3",
				Type:    memory.MemoryTypeDocument,
				Content: "Test document content",
				Source:  "test",
			},
			setupMock: func() {
				suite.mockProvider.On("Store", suite.ctx, mock.AnythingOfType("[]*memory.VectorData")).
					Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			
			err := suite.cogneeIntegration.StoreMemory(suite.ctx, tt.memory)
			
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			
			suite.mockProvider.AssertExpectations(suite.T())
		})
	}
}

// TestRetrieveMemory tests memory retrieval with Cognee optimization
func (suite *CogneeIntegrationTestSuite) TestRetrieveMemory() {
	tests := []struct {
		name     string
		query    *memory.RetrievalQuery
		setupMock func()
		want     *memory.RetrievalResult
		wantErr  bool
	}{
		{
			name: "Valid retrieval query",
			query: &memory.RetrievalQuery{
				Type:     memory.MemoryTypeConversation,
				Keywords: []string{"AI", "machine learning"},
				Limit:    10,
				Filters: map[string]interface{}{
					"user_id": "user123",
				},
			},
			setupMock: func() {
				mockResults := []*memory.VectorData{
					{
						ID:       "memory_1",
						Vector:   []float64{0.1, 0.2, 0.3},
						Metadata: map[string]interface{}{
							"content": "AI conversation",
							"type":    "conversation",
						},
					},
				}
				suite.mockProvider.On("Search", suite.ctx, mock.AnythingOfType("*memory.VectorQuery")).
					Return(&memory.VectorSearchResult{
						Results: []*memory.VectorSearchResultItem{
							{
								ID:     "memory_1",
								Vector: []float64{0.1, 0.2, 0.3},
								Score:  0.95,
								Metadata: map[string]interface{}{
									"content": "AI conversation",
									"type":    "conversation",
								},
							},
						},
					}, nil)
			},
			want: &memory.RetrievalResult{
				Items: []*memory.MemoryItem{
					{
						ID:       "memory_1",
						Type:     memory.MemoryTypeConversation,
						Content:  "AI conversation",
						Score:    0.95,
						Metadata: map[string]interface{}{
							"type": "conversation",
						},
					},
				},
				Total: 1,
			},
			wantErr: false,
		},
		{
			name: "Invalid retrieval query - nil",
			query: nil,
			setupMock: func() {
				// No mock setup expected
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Provider search error",
			query: &memory.RetrievalQuery{
				Type:  memory.MemoryTypeKnowledge,
				Limit: 5,
			},
			setupMock: func() {
				suite.mockProvider.On("Search", suite.ctx, mock.AnythingOfType("*memory.VectorQuery")).
					Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			
			result, err := suite.cogneeIntegration.RetrieveMemory(suite.ctx, tt.query)
			
			if tt.wantErr {
				suite.Error(err)
				suite.Nil(result)
			} else {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal(tt.want.Total, result.Total)
				if tt.want != nil && len(tt.want.Items) > 0 {
					suite.Len(result.Items, len(tt.want.Items))
					suite.Equal(tt.want.Items[0].ID, result.Items[0].ID)
				}
			}
			
			suite.mockProvider.AssertExpectations(suite.T())
		})
	}
}

// TestGetContext tests context retrieval with LLM provider
func (suite *CogneeIntegrationTestSuite) TestGetContext() {
	tests := []struct {
		name     string
		provider string
		model    string
		session  string
		setupMock func()
		want     *memory.ContextData
		wantErr  bool
	}{
		{
			name:     "Valid context request",
			provider: "openai",
			model:    "gpt-4",
			session:  "session_123",
			setupMock: func() {
				mockResults := []*memory.VectorData{
					{
						ID:       "context_1",
						Vector:   []float64{0.1, 0.2, 0.3},
						Metadata: map[string]interface{}{
							"content": "Previous conversation",
							"type":    "conversation",
						},
					},
				}
				suite.mockProvider.On("Search", suite.ctx, mock.AnythingOfType("*memory.VectorQuery")).
					Return(&memory.VectorSearchResult{
						Results: []*memory.VectorSearchResultItem{
							{
								ID:     "context_1",
								Vector: []float64{0.1, 0.2, 0.3},
								Score:  0.90,
								Metadata: map[string]interface{}{
									"content": "Previous conversation",
									"type":    "conversation",
								},
							},
						},
					}, nil)
			},
			want: &memory.ContextData{
				Context: "Previous conversation",
				Memory: []*memory.MemoryItem{
					{
						ID:      "context_1",
						Type:    memory.MemoryTypeConversation,
						Content: "Previous conversation",
						Score:   0.90,
						Metadata: map[string]interface{}{
							"type": "conversation",
						},
					},
				},
				LastUpdated: time.Now(),
			},
			wantErr: false,
		},
		{
			name:     "Empty session",
			provider: "openai",
			model:    "gpt-4",
			session:  "",
			setupMock: func() {
				suite.mockProvider.On("Search", suite.ctx, mock.AnythingOfType("*memory.VectorQuery")).
					Return(&memory.VectorSearchResult{Results: []*memory.VectorSearchResultItem{}}, nil)
			},
			want: &memory.ContextData{
				Context:    "",
				Memory:     []*memory.MemoryItem{},
				LastUpdated: time.Now(),
			},
			wantErr: false,
		},
		{
			name:     "Provider search error",
			provider: "openai",
			model:    "gpt-4",
			session:  "session_456",
			setupMock: func() {
				suite.mockProvider.On("Search", suite.ctx, mock.AnythingOfType("*memory.VectorQuery")).
					Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			
			result, err := suite.cogneeIntegration.GetContext(suite.ctx, tt.provider, tt.model, tt.session)
			
			if tt.wantErr {
				suite.Error(err)
				suite.Nil(result)
			} else {
				suite.NoError(err)
				suite.NotNil(result)
				suite.NotEmpty(result.Context)
			}
			
			suite.mockProvider.AssertExpectations(suite.T())
		})
	}
}

// TestSystemInfo tests system information retrieval
func (suite *CogneeIntegrationTestSuite) TestSystemInfo() {
	tests := []struct {
		name     string
		setupMock func()
		want     *memory.SystemInfo
		wantErr  bool
	}{
		{
			name: "Valid system info",
			setupMock: func() {
				suite.mockProvider.On("GetStats", suite.ctx).
					Return(&providers.ProviderStats{
						TotalVectors: 1000,
						TotalSize:    1024000,
						AverageLatency: 50 * time.Millisecond,
					}, nil)
			},
			want: &memory.SystemInfo{
				CPUCores:       4,
				TotalMemory:    8589934592, // 8GB in bytes
				DiskSpace:      107374182400, // 100GB in bytes
				GPUAvailable:   false,
				ActiveProvider: "mock",
				VectorCount:    1000,
				StorageSize:    1024000,
				AverageLatency: 50 * time.Millisecond,
			},
			wantErr: false,
		},
		{
			name: "Provider stats error",
			setupMock: func() {
				suite.mockProvider.On("GetStats", suite.ctx).
					Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			
			result, err := suite.cogneeIntegration.GetSystemInfo(suite.ctx)
			
			if tt.wantErr {
				suite.Error(err)
				suite.Nil(result)
			} else {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Greater(result.CPUCores, 0)
				suite.Greater(result.TotalMemory, uint64(0))
				suite.GreaterOrEqual(result.DiskSpace, uint64(0))
			}
			
			suite.mockProvider.AssertExpectations(suite.T())
		})
	}
}

// TestOptimizationRecommendations tests optimization recommendations
func (suite *CogneeIntegrationTestSuite) TestOptimizationRecommendations() {
	tests := []struct {
		name     string
		setupMock func()
		want     []*memory.OptimizationRecommendation
		wantErr  bool
	}{
		{
			name: "Valid recommendations",
			setupMock: func() {
				suite.mockProvider.On("GetStats", suite.ctx).
					Return(&providers.ProviderStats{
						TotalVectors:    10000,
						TotalSize:      10240000,
						AverageLatency: 200 * time.Millisecond,
						ErrorCount:     100,
						TotalOperations: 1000,
					}, nil)
			},
			want: []*memory.OptimizationRecommendation{
				{
					Category:        "performance",
					Recommendation:  "Increase batch size to improve throughput",
					Confidence:      0.85,
					ResearchPaper:   "Vector Database Optimization Studies",
				},
				{
					Category:        "storage",
					Recommendation:  "Enable compression to reduce storage costs",
					Confidence:      0.90,
					ResearchPaper:   "Efficient Vector Storage Techniques",
				},
				{
					Category:        "latency",
					Recommendation:  "Consider using GPU acceleration for faster processing",
					Confidence:      0.75,
					ResearchPaper:   "GPU-Based Vector Similarity Search",
				},
			},
			wantErr: false,
		},
		{
			name: "No recommendations needed",
			setupMock: func() {
				suite.mockProvider.On("GetStats", suite.ctx).
					Return(&providers.ProviderStats{
						TotalVectors:     1000,
						TotalSize:       1024000,
						AverageLatency:   50 * time.Millisecond,
						ErrorCount:      10,
						TotalOperations:  1000,
					}, nil)
			},
			want:    []*memory.OptimizationRecommendation{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			
			result, err := suite.cogneeIntegration.GetOptimizationRecommendations(suite.ctx)
			
			if tt.wantErr {
				suite.Error(err)
				suite.Nil(result)
			} else {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Len(result, len(tt.want))
			}
			
			suite.mockProvider.AssertExpectations(suite.T())
		})
	}
}

// TestApplyOptimizations tests applying optimization recommendations
func (suite *CogneeIntegrationTestSuite) TestApplyOptimizations() {
	recommendations := []*memory.OptimizationRecommendation{
		{
			Category:       "performance",
			Recommendation: "Increase batch size",
			Confidence:     0.85,
		},
	}

	tests := []struct {
		name          string
		recommendations []*memory.OptimizationRecommendation
		setupMock     func()
		wantErr       bool
	}{
		{
			name:          "Valid optimizations",
			recommendations: recommendations,
			setupMock: func() {
				suite.mockProvider.On("Optimize", suite.ctx).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "Empty recommendations",
			recommendations: []*memory.OptimizationRecommendation{},
			setupMock: func() {
				// No optimization expected
			},
			wantErr: false,
		},
		{
			name:          "Optimize error",
			recommendations: recommendations,
			setupMock: func() {
				suite.mockProvider.On("Optimize", suite.ctx).
					Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			
			err := suite.cogneeIntegration.ApplyOptimizations(suite.ctx, tt.recommendations)
			
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
			
			suite.mockProvider.AssertExpectations(suite.T())
		})
	}
}

// TestHealthCheck tests health check functionality
func (suite *CogneeIntegrationTestSuite) TestHealthCheck() {
	tests := []struct {
		name     string
		setupMock func()
		want     *memory.HealthStatus
		wantErr  bool
	}{
		{
			name: "Healthy status",
			setupMock: func() {
				suite.mockProvider.On("Health", suite.ctx).
					Return(&providers.HealthStatus{
						Status:      "healthy",
						LastCheck:   time.Now(),
						ResponseTime: 100 * time.Millisecond,
					}, nil)
			},
			want: &memory.HealthStatus{
				Status:       "healthy",
				LastCheck:    time.Now(),
				ResponseTime:  100 * time.Millisecond,
			},
			wantErr: false,
		},
		{
			name: "Unhealthy status",
			setupMock: func() {
				suite.mockProvider.On("Health", suite.ctx).
					Return(&providers.HealthStatus{
						Status:      "unhealthy",
						LastCheck:   time.Now(),
						ResponseTime: 1000 * time.Millisecond,
						Error:       "Connection failed",
					}, nil)
			},
			want: &memory.HealthStatus{
				Status:       "unhealthy",
				LastCheck:    time.Now(),
				ResponseTime:  1000 * time.Millisecond,
				Error:        "Connection failed",
			},
			wantErr: false,
		},
		{
			name: "Health check error",
			setupMock: func() {
				suite.mockProvider.On("Health", suite.ctx).
					Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			
			result, err := suite.cogneeIntegration.HealthCheck(suite.ctx)
			
			if tt.wantErr {
				suite.Error(err)
				suite.Nil(result)
			} else {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal(tt.want.Status, result.Status)
			}
			
			suite.mockProvider.AssertExpectations(suite.T())
		})
	}
}

// TestConcurrentOperations tests concurrent operations
func (suite *CogneeIntegrationTestSuite) TestConcurrentOperations() {
	const numOperations = 100
	
	suite.mockProvider.On("Store", suite.ctx, mock.AnythingOfType("[]*memory.VectorData")).
		Return(nil)
	
	// Test concurrent store operations
	done := make(chan bool, numOperations)
	
	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer func() { done <- true }()
			
			memory := &memory.MemoryData{
				ID:       fmt.Sprintf("concurrent_memory_%d", index),
				Type:     memory.MemoryTypeConversation,
				Content:  fmt.Sprintf("Concurrent test content %d", index),
				Source:   "test",
				Metadata: map[string]interface{}{
					"index": index,
				},
				Timestamp: time.Now(),
			}
			
			err := suite.cogneeIntegration.StoreMemory(suite.ctx, memory)
			suite.NoError(err)
		}(i)
	}
	
	// Wait for all operations to complete
	for i := 0; i < numOperations; i++ {
		<-done
	}
	
	// Verify all store operations were called
	suite.mockProvider.AssertNumberOfCalls(suite.T(), "Store", numOperations)
}

// TestShutdown tests graceful shutdown
func (suite *CogneeIntegrationTestSuite) TestShutdown() {
	// Initialize and then shutdown
	err := suite.cogneeIntegration.Initialize(suite.ctx, suite.config)
	suite.NoError(err)
	
	// Shutdown should not panic
	assert.NotPanics(suite.T(), func() {
		suite.cogneeIntegration.Shutdown(suite.ctx)
	})
}

// BenchmarkStoreMemory benchmarks memory storage
func (suite *CogneeIntegrationTestSuite) BenchmarkStoreMemory() {
	suite.mockProvider.On("Store", suite.ctx, mock.AnythingOfType("[]*memory.VectorData")).
		Return(nil)
	
	memory := &memory.MemoryData{
		ID:       "benchmark_memory",
		Type:     memory.MemoryTypeConversation,
		Content:  "Benchmark test content",
		Source:   "test",
		Metadata: map[string]interface{}{
			"benchmark": true,
		},
		Timestamp: time.Now(),
	}
	
	suite.b.ResetTimer()
	for i := 0; i < suite.b.N; i++ {
		err := suite.cogneeIntegration.StoreMemory(suite.ctx, memory)
		suite.NoError(err)
	}
}

// BenchmarkRetrieveMemory benchmarks memory retrieval
func (suite *CogneeIntegrationTestSuite) BenchmarkRetrieveMemory() {
	suite.mockProvider.On("Search", suite.ctx, mock.AnythingOfType("*memory.VectorQuery")).
		Return(&memory.VectorSearchResult{
			Results: []*memory.VectorSearchResultItem{},
		}, nil)
	
	query := &memory.RetrievalQuery{
		Type:     memory.MemoryTypeConversation,
		Keywords: []string{"test", "benchmark"},
		Limit:    10,
	}
	
	suite.b.ResetTimer()
	for i := 0; i < suite.b.N; i++ {
		_, err := suite.cogneeIntegration.RetrieveMemory(suite.ctx, query)
		suite.NoError(err)
	}
}

// Run the test suite
func TestCogneeIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CogneeIntegrationTestSuite))
}

// Individual test functions for isolated testing
func TestCogneeInitialization_Error(t *testing.T) {
	logger := logging.NewTestLogger("cognee_test")
	mockProvider := mocks.NewMockVectorProvider(t)
	mockAPIKeyManager := mocks.NewMockAPIKeyManager(t)
	
	ci := memory.NewCogneeIntegration(
		mockProvider,
		mockProvider,
		mockAPIKeyManager,
	)
	
	// Test initialization with nil config
	err := ci.Initialize(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration is required")
}

func TestCogneeStoreMemory_EmptyContent(t *testing.T) {
	logger := logging.NewTestLogger("cognee_test")
	mockProvider := mocks.NewMockVectorProvider(t)
	mockAPIKeyManager := mocks.NewMockAPIKeyManager(t)
	
	ci := memory.NewCogneeIntegration(
		mockProvider,
		mockProvider,
		mockAPIKeyManager,
	)
	
	// Initialize with valid config
	config := &config.CogneeConfig{
		Enabled: true,
		Mode:    config.CogneeModeLocal,
	}
	
	err := ci.Initialize(context.Background(), config)
	require.NoError(t, err)
	
	// Test storing memory with empty content
	memory := &memory.MemoryData{
		ID:      "test_empty",
		Type:    memory.MemoryTypeConversation,
		Content: "",
		Source:  "test",
	}
	
	err = ci.StoreMemory(context.Background(), memory)
	// Should not error - empty content is handled gracefully
	assert.NoError(t, err)
}

func TestCogneeRetrieveMemory_EmptyQuery(t *testing.T) {
	logger := logging.NewTestLogger("cognee_test")
	mockProvider := mocks.NewMockVectorProvider(t)
	mockAPIKeyManager := mocks.NewMockAPIKeyManager(t)
	
	ci := memory.NewCogneeIntegration(
		mockProvider,
		mockProvider,
		mockAPIKeyManager,
	)
	
	// Initialize with valid config
	config := &config.CogneeConfig{
		Enabled: true,
		Mode:    config.CogneeModeLocal,
	}
	
	err := ci.Initialize(context.Background(), config)
	require.NoError(t, err)
	
	// Test retrieval with empty query
	query := &memory.RetrievalQuery{}
	
	result, err := ci.RetrieveMemory(context.Background(), query)
	assert.NoError(t, err)
	assert.NotNil(result)
	assert.Equal(0, result.Total)
}

func TestCogneeIntegration_CancelledContext(t *testing.T) {
	logger := logging.NewTestLogger("cognee_test")
	mockProvider := mocks.NewMockVectorProvider(t)
	mockAPIKeyManager := mocks.NewMockAPIKeyManager(t)
	
	ci := memory.NewCogneeIntegration(
		mockProvider,
		mockProvider,
		mockAPIKeyManager,
	)
	
	// Initialize with valid config
	config := &config.CogneeConfig{
		Enabled: true,
		Mode:    config.CogneeModeLocal,
	}
	
	err := ci.Initialize(context.Background(), config)
	require.NoError(t, err)
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	// Test operations with cancelled context
	memory := &memory.MemoryData{
		ID:      "test_cancelled",
		Type:    memory.MemoryTypeConversation,
		Content: "Test content",
		Source:  "test",
	}
	
	err = ci.StoreMemory(ctx, memory)
	// Should return context cancelled error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// Edge case tests
func TestCogneeIntegration_MaximumDataSize(t *testing.T) {
	logger := logging.NewTestLogger("cognee_test")
	mockProvider := mocks.NewMockVectorProvider(t)
	mockAPIKeyManager := mocks.NewMockAPIKeyManager(t)
	
	ci := memory.NewCogneeIntegration(
		mockProvider,
		mockProvider,
		mockAPIKeyManager,
	)
	
	// Initialize with valid config
	config := &config.CogneeConfig{
		Enabled: true,
		Mode:    config.CogneeModeLocal,
	}
	
	err := ci.Initialize(context.Background(), config)
	require.NoError(t, err)
	
	// Create memory with very large content (1MB)
	largeContent := strings.Repeat("x", 1024*1024)
	memory := &memory.MemoryData{
		ID:      "test_large",
		Type:    memory.MemoryTypeDocument,
		Content: largeContent,
		Source:  "test",
	}
	
	// Should handle large content gracefully
	err = ci.StoreMemory(context.Background(), memory)
	// Depending on implementation, this might succeed or fail with appropriate error
	// For now, we just verify it doesn't panic
	assert.NotPanics(t, func() {
		ci.StoreMemory(context.Background(), memory)
	})
}

func TestCogneeIntegration_SpecialCharacters(t *testing.T) {
	logger := logging.NewTestLogger("cognee_test")
	mockProvider := mocks.NewMockVectorProvider(t)
	mockAPIKeyManager := mocks.NewMockAPIKeyManager(t)
	
	ci := memory.NewCogneeIntegration(
		mockProvider,
		mockProvider,
		mockAPIKeyManager,
	)
	
	// Initialize with valid config
	config := &config.CogneeConfig{
		Enabled: true,
		Mode:    config.CogneeModeLocal,
	}
	
	err := ci.Initialize(context.Background(), config)
	require.NoError(t, err)
	
	// Create memory with special characters
	specialContent := "Special chars: áéíóú 中文 العربية русский 한국어 日本語 🚀🎉"
	memory := &memory.MemoryData{
		ID:      "test_special",
		Type:    memory.MemoryTypeConversation,
		Content: specialContent,
		Source:  "test",
		Metadata: map[string]interface{}{
			"special_field": "special_value_áéíóú",
		},
	}
	
	err = ci.StoreMemory(context.Background(), memory)
	assert.NoError(t, err)
}