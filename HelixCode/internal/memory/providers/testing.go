package providers

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ProviderTestSuite provides comprehensive testing for providers
type ProviderTestSuite struct {
	provider VectorProvider
	config   map[string]interface{}
	ctx      context.Context
}

// NewProviderTestSuite creates a new test suite
func NewProviderTestSuite(provider VectorProvider, config map[string]interface{}) *ProviderTestSuite {
	return &ProviderTestSuite{
		provider: provider,
		config:   config,
		ctx:      context.Background(),
	}
}

// RunAllTests runs all tests for the provider
func (pts *ProviderTestSuite) RunAllTests(t *testing.T) {
	t.Run("Initialize", pts.TestInitialize)
	t.Run("Start", pts.TestStart)
	t.Run("Store", pts.TestStore)
	t.Run("Retrieve", pts.TestRetrieve)
	t.Run("Search", pts.TestSearch)
	t.Run("FindSimilar", pts.TestFindSimilar)
	t.Run("CollectionManagement", pts.TestCollectionManagement)
	t.Run("IndexManagement", pts.TestIndexManagement)
	t.Run("MetadataManagement", pts.TestMetadataManagement)
	t.Run("Stats", pts.TestStats)
	t.Run("Health", pts.TestHealth)
	t.Run("Optimize", pts.TestOptimize)
	t.Run("BackupRestore", pts.TestBackupRestore)
	t.Run("Stop", pts.TestStop)
}

// TestInitialize tests provider initialization
func (pts *ProviderTestSuite) TestInitialize(t *testing.T) {
	err := pts.provider.Initialize(pts.ctx, pts.config)
	if err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}
}

// TestStart tests provider start
func (pts *ProviderTestSuite) TestStart(t *testing.T) {
	err := pts.provider.Start(pts.ctx)
	if err != nil {
		t.Fatalf("Failed to start provider: %v", err)
	}
}

// TestStore tests vector storage
func (pts *ProviderTestSuite) TestStore(t *testing.T) {
	vectors := pts.generateTestVectors(100, 1536)

	start := time.Now()
	err := pts.provider.Store(pts.ctx, vectors)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to store vectors: %v", err)
	}

	t.Logf("Stored %d vectors in %v (%.2f vectors/sec)",
		len(vectors), duration, float64(len(vectors))/duration.Seconds())
}

// TestRetrieve tests vector retrieval
func (pts *ProviderTestSuite) TestRetrieve(t *testing.T) {
	// Store test vectors first
	vectors := pts.generateTestVectors(10, 1536)
	err := pts.provider.Store(pts.ctx, vectors)
	if err != nil {
		t.Fatalf("Failed to store test vectors: %v", err)
	}

	// Retrieve by IDs
	var ids []string
	for _, vector := range vectors {
		ids = append(ids, vector.ID)
	}

	start := time.Now()
	retrieved, err := pts.provider.Retrieve(pts.ctx, ids)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to retrieve vectors: %v", err)
	}

	if len(retrieved) != len(ids) {
		t.Fatalf("Expected %d vectors, got %d", len(ids), len(retrieved))
	}

	t.Logf("Retrieved %d vectors in %v", len(retrieved), duration)
}

// TestSearch tests vector similarity search
func (pts *ProviderTestSuite) TestSearch(t *testing.T) {
	// Store test vectors first
	vectors := pts.generateTestVectors(1000, 1536)
	err := pts.provider.Store(pts.ctx, vectors)
	if err != nil {
		t.Fatalf("Failed to store test vectors: %v", err)
	}

	// Create search query
	queryVector := pts.generateTestVector(1536)
	query := &VectorQuery{
		Vector:     queryVector,
		Collection: "test_collection",
		TopK:       10,
		Threshold:  0.7,
	}

	start := time.Now()
	result, err := pts.provider.Search(pts.ctx, query)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to search vectors: %v", err)
	}

	if len(result.Results) > query.TopK {
		t.Fatalf("Expected at most %d results, got %d", query.TopK, len(result.Results))
	}

	t.Logf("Search returned %d results in %v (%.2f results/sec)",
		len(result.Results), duration, float64(len(result.Results))/duration.Seconds())
}

// TestFindSimilar tests finding similar vectors
func (pts *ProviderTestSuite) TestFindSimilar(t *testing.T) {
	// Store test vectors first
	vectors := pts.generateTestVectors(500, 1536)
	err := pts.provider.Store(pts.ctx, vectors)
	if err != nil {
		t.Fatalf("Failed to store test vectors: %v", err)
	}

	// Find similar vectors
	embedding := pts.generateTestVector(1536)
	k := 5
	filters := map[string]interface{}{
		"category": "test",
	}

	start := time.Now()
	results, err := pts.provider.FindSimilar(pts.ctx, embedding, k, filters)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to find similar vectors: %v", err)
	}

	if len(results) > k {
		t.Fatalf("Expected at most %d results, got %d", k, len(results))
	}

	t.Logf("FindSimilar returned %d results in %v", len(results), duration)
}

// TestCollectionManagement tests collection operations
func (pts *ProviderTestSuite) TestCollectionManagement(t *testing.T) {
	collectionName := "test_collection"
	config := &CollectionConfig{
		Dimension:   1536,
		Metric:      "cosine",
		Description: "Test collection",
	}

	// Create collection
	err := pts.provider.CreateCollection(pts.ctx, collectionName, config)
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// List collections
	collections, err := pts.provider.ListCollections(pts.ctx)
	if err != nil {
		t.Fatalf("Failed to list collections: %v", err)
	}

	found := false
	for _, collection := range collections {
		if collection.Name == collectionName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Collection %s not found in list", collectionName)
	}

	// Get collection info
	collectionInfo, err := pts.provider.GetCollection(pts.ctx, collectionName)
	if err != nil {
		t.Fatalf("Failed to get collection info: %v", err)
	}
	if collectionInfo.Name != collectionName {
		t.Fatalf("Expected collection name %s, got %s", collectionName, collectionInfo.Name)
	}

	// Delete collection
	err = pts.provider.DeleteCollection(pts.ctx, collectionName)
	if err != nil {
		t.Fatalf("Failed to delete collection: %v", err)
	}

	t.Logf("Collection management test passed")
}

// TestIndexManagement tests index operations
func (pts *ProviderTestSuite) TestIndexManagement(t *testing.T) {
	collectionName := "test_index_collection"
	config := &CollectionConfig{
		Dimension:   1536,
		Metric:      "cosine",
		Description: "Test index collection",
	}

	// Create collection first
	err := pts.provider.CreateCollection(pts.ctx, collectionName, config)
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	// Create index
	indexConfig := &IndexConfig{
		Name:   "test_index",
		Type:   "IVF_FLAT",
		Metric: "cosine",
	}

	err = pts.provider.CreateIndex(pts.ctx, collectionName, indexConfig)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Some providers don't support explicit index listing

	// Delete index
	err = pts.provider.DeleteIndex(pts.ctx, collectionName, indexConfig.Name)
	if err != nil {
		t.Fatalf("Failed to delete index: %v", err)
	}

	// Clean up collection
	err = pts.provider.DeleteCollection(pts.ctx, collectionName)
	if err != nil {
		t.Fatalf("Failed to delete collection: %v", err)
	}

	t.Logf("Index management test passed")
}

// TestMetadataManagement tests metadata operations
func (pts *ProviderTestSuite) TestMetadataManagement(t *testing.T) {
	// Store test vectors first
	vectors := pts.generateTestVectors(5, 1536)
	err := pts.provider.Store(pts.ctx, vectors)
	if err != nil {
		t.Fatalf("Failed to store test vectors: %v", err)
	}

	// Add metadata
	vectorID := vectors[0].ID
	metadata := map[string]interface{}{
		"category": "test",
		"priority": 1,
		"tags":     []string{"test", "metadata"},
	}

	err = pts.provider.AddMetadata(pts.ctx, vectorID, metadata)
	if err != nil {
		t.Fatalf("Failed to add metadata: %v", err)
	}

	// Get metadata
	var ids []string
	for _, vector := range vectors {
		ids = append(ids, vector.ID)
	}

	retrievedMetadata, err := pts.provider.GetMetadata(pts.ctx, ids)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if vectorMetadata, exists := retrievedMetadata[vectorID]; exists {
		if vectorMetadata["category"] != metadata["category"] {
			t.Fatalf("Metadata category mismatch: expected %v, got %v",
				metadata["category"], vectorMetadata["category"])
		}
	}

	// Update metadata
	updatedMetadata := map[string]interface{}{
		"category": "updated",
		"priority": 2,
	}

	err = pts.provider.UpdateMetadata(pts.ctx, vectorID, updatedMetadata)
	if err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}

	// Delete metadata
	err = pts.provider.DeleteMetadata(pts.ctx, []string{vectorID}, []string{"priority"})
	if err != nil {
		t.Fatalf("Failed to delete metadata: %v", err)
	}

	t.Logf("Metadata management test passed")
}

// TestStats tests provider statistics
func (pts *ProviderTestSuite) TestStats(t *testing.T) {
	stats, err := pts.provider.GetStats(pts.ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	t.Logf("Provider stats: TotalVectors=%d, TotalCollections=%d, TotalSize=%d",
		stats.TotalVectors, stats.TotalCollections, stats.TotalSize)

	// Test cost info
	costInfo := pts.provider.GetCostInfo()
	t.Logf("Cost info: TotalCost=%.2f, Currency=%s, BillingPeriod=%s",
		costInfo.TotalCost, costInfo.Currency, costInfo.BillingPeriod)
}

// TestHealth tests provider health
func (pts *ProviderTestSuite) TestHealth(t *testing.T) {
	health, err := pts.provider.Health(pts.ctx)
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}

	t.Logf("Provider health: Status=%s, ResponseTime=%v",
		health.Status, health.ResponseTime)

	if health.Status != "healthy" && health.Status != "not_started" {
		t.Fatalf("Unexpected health status: %s", health.Status)
	}
}

// TestOptimize tests provider optimization
func (pts *ProviderTestSuite) TestOptimize(t *testing.T) {
	err := pts.provider.Optimize(pts.ctx)
	if err != nil {
		t.Logf("Optimize failed (may not be supported): %v", err)
	} else {
		t.Logf("Optimization completed successfully")
	}
}

// TestBackupRestore tests backup and restore operations
func (pts *ProviderTestSuite) TestBackupRestore(t *testing.T) {
	backupPath := "/tmp/test_backup"

	// Test backup
	err := pts.provider.Backup(pts.ctx, backupPath)
	if err != nil {
		t.Logf("Backup failed (may not be supported): %v", err)
	} else {
		t.Logf("Backup completed successfully")
	}

	// Test restore
	err = pts.provider.Restore(pts.ctx, backupPath)
	if err != nil {
		t.Logf("Restore failed (may not be supported): %v", err)
	} else {
		t.Logf("Restore completed successfully")
	}
}

// TestStop tests provider stop
func (pts *ProviderTestSuite) TestStop(t *testing.T) {
	err := pts.provider.Stop(pts.ctx)
	if err != nil {
		t.Fatalf("Failed to stop provider: %v", err)
	}
}

// generateTestVectors generates test vectors
func (pts *ProviderTestSuite) generateTestVectors(count, dimension int) []*VectorData {
	var vectors []*VectorData

	for i := 0; i < count; i++ {
		vectors = append(vectors, pts.generateTestVectorWithID(dimension, i))
	}

	return vectors
}

// generateTestVector generates a single test vector
func (pts *ProviderTestSuite) generateTestVector(dimension int) []float64 {
	vectorData := pts.generateTestVectorWithID(dimension, 0)
	return vectorData.Vector
}

// generateTestVectorWithID generates a test vector with specific ID
func (pts *ProviderTestSuite) generateTestVectorWithID(dimension, id int) *VectorData {
	vector := make([]float64, dimension)
	for i := 0; i < dimension; i++ {
		// Generate pseudo-random values based on ID and position
		vector[i] = float64((id*7+i*13)%100) / 100.0
	}

	return &VectorData{
		ID:     fmt.Sprintf("test_vector_%d", id),
		Vector: vector,
		Metadata: map[string]interface{}{
			"category": "test",
			"id":       id,
			"created":  time.Now(),
		},
		Collection: "test_collection",
		Timestamp:  time.Now(),
	}
}

// BenchmarkStore benchmarks vector storage
func (pts *ProviderTestSuite) BenchmarkStore(b *testing.B) {
	pts.provider.Initialize(context.Background(), pts.config)
	pts.provider.Start(context.Background())
	defer pts.provider.Stop(context.Background())

	vectors := pts.generateTestVectors(1000, 1536)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := vectors[:min(len(vectors), 100)]
		pts.provider.Store(context.Background(), batch)
	}
}

// BenchmarkSearch benchmarks vector search
func (pts *ProviderTestSuite) BenchmarkSearch(b *testing.B) {
	pts.provider.Initialize(context.Background(), pts.config)
	pts.provider.Start(context.Background())
	defer pts.provider.Stop(context.Background())

	// Store test vectors first
	vectors := pts.generateTestVectors(10000, 1536)
	pts.provider.Store(context.Background(), vectors)

	queryVector := pts.generateTestVector(1536)
	query := &VectorQuery{
		Vector:     queryVector,
		Collection: "test_collection",
		TopK:       10,
		Threshold:  0.7,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pts.provider.Search(context.Background(), query)
	}
}

// BenchmarkProviderPerformance runs performance benchmarks
func BenchmarkProviderPerformance(b *testing.B, provider VectorProvider, config map[string]interface{}) {
	suite := NewProviderTestSuite(provider, config)

	b.Run("BenchmarkStore", suite.BenchmarkStore)
	b.Run("BenchmarkSearch", suite.BenchmarkSearch)
}

// min returns minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CompareProviders compares performance between multiple providers
func CompareProviders(t *testing.T, providers map[string]VectorProvider, configs map[string]map[string]interface{}) {
	results := make(map[string]ProviderTestResult)

	for name, provider := range providers {
		config := configs[name]
		suite := NewProviderTestSuite(provider, config)

		result := ProviderTestResult{
			Name: name,
		}

		// Run store test
		start := time.Now()
		vectors := suite.generateTestVectors(1000, 1536)
		err := provider.Initialize(context.Background(), config)
		if err != nil {
			result.Error = err.Error()
			results[name] = result
			continue
		}

		err = provider.Start(context.Background())
		if err != nil {
			result.Error = err.Error()
			results[name] = result
			continue
		}

		err = provider.Store(context.Background(), vectors)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.StoreDuration = time.Since(start)
		}

		// Run search test
		queryVector := suite.generateTestVector(1536)
		query := &VectorQuery{
			Vector:     queryVector,
			Collection: "test_collection",
			TopK:       10,
			Threshold:  0.7,
		}

		start = time.Now()
		_, err = provider.Search(context.Background(), query)
		if err == nil {
			result.SearchDuration = time.Since(start)
		}

		provider.Stop(context.Background())
		results[name] = result
	}

	// Print comparison results
	t.Log("Provider Performance Comparison:")
	t.Logf("%-20s %-15s %-15s %-10s", "Provider", "Store Duration", "Search Duration", "Status")
	t.Logf("%-20s %-15s %-15s %-10s", "--------", "--------------", "---------------", "------")

	for name, result := range results {
		status := "OK"
		if result.Error != "" {
			status = "ERROR"
		}

		t.Logf("%-20s %-15s %-15s %-10s",
			name,
			result.StoreDuration.String(),
			result.SearchDuration.String(),
			status)
	}
}

// ProviderTestResult contains test results for a provider
type ProviderTestResult struct {
	Name           string        `json:"name"`
	StoreDuration  time.Duration `json:"store_duration"`
	SearchDuration time.Duration `json:"search_duration"`
	Error          string        `json:"error,omitempty"`
}
