package providers

import (
	"fmt"
	"math"

	"dev.helix.code/internal/memory"
)

// vectorToString converts a vector to a string representation
func vectorToString(vector *memory.VectorData) string {
	return fmt.Sprintf("Vector ID: %s, Size: %d", vector.ID, len(vector.Vector))
}

// sqrt calculates the square root of a float64
func sqrt(x float64) float64 {
	return math.Sqrt(x)
}

// calculateCosineSimilarity calculates cosine similarity between two vectors
func calculateCosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// calculateSimilarity calculates similarity between two vectors (alias for cosine similarity)
func calculateSimilarity(a, b []float64) float64 {
	return calculateCosineSimilarity(a, b)
}

// boolToFloat64 converts a boolean to float64
func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// convertMemoryVectorDataToProvider converts memory.VectorData to VectorData
func convertMemoryVectorDataToProvider(mv *memory.VectorData) *VectorData {
	if mv == nil {
		return nil
	}
	return &VectorData{
		ID:         mv.ID,
		Vector:     mv.Vector,
		Metadata:   mv.Metadata,
		Collection: mv.Collection,
		Timestamp:  mv.Timestamp,
		TTL:        mv.TTL,
		Namespace:  mv.Namespace,
	}
}

// convertMemoryVectorDataSliceToProvider converts []*memory.VectorData to []*VectorData
func convertMemoryVectorDataSliceToProvider(mvs []*memory.VectorData) []*VectorData {
	if mvs == nil {
		return nil
	}
	result := make([]*VectorData, len(mvs))
	for i, mv := range mvs {
		result[i] = convertMemoryVectorDataToProvider(mv)
	}
	return result
}

// convertProviderVectorDataToMemory converts VectorData to memory.VectorData
func convertProviderVectorDataToMemory(pv *VectorData) *memory.VectorData {
	if pv == nil {
		return nil
	}
	return &memory.VectorData{
		ID:         pv.ID,
		Vector:     pv.Vector,
		Metadata:   pv.Metadata,
		Collection: pv.Collection,
		Timestamp:  pv.Timestamp,
		TTL:        pv.TTL,
		Namespace:  pv.Namespace,
	}
}

// convertProviderVectorDataSliceToMemory converts []*VectorData to []*memory.VectorData
func convertProviderVectorDataSliceToMemory(pvs []*VectorData) []*memory.VectorData {
	if pvs == nil {
		return nil
	}
	result := make([]*memory.VectorData, len(pvs))
	for i, pv := range pvs {
		result[i] = convertProviderVectorDataToMemory(pv)
	}
	return result
}

// convertMemoryVectorQueryToProvider converts memory.VectorQuery to VectorQuery
func convertMemoryVectorQueryToProvider(mq *memory.VectorQuery) *VectorQuery {
	if mq == nil {
		return nil
	}
	return &VectorQuery{
		Vector:        mq.Vector,
		Collection:    mq.Collection,
		Namespace:     mq.Namespace,
		TopK:          mq.TopK,
		Threshold:     mq.Threshold,
		IncludeVector: mq.IncludeVector,
		Filters:       mq.Filters,
	}
}

// convertProviderVectorQueryToMemory converts VectorQuery to memory.VectorQuery
func convertProviderVectorQueryToMemory(pq *VectorQuery) *memory.VectorQuery {
	if pq == nil {
		return nil
	}
	return &memory.VectorQuery{
		Vector:        pq.Vector,
		Collection:    pq.Collection,
		Namespace:     pq.Namespace,
		TopK:          pq.TopK,
		Threshold:     pq.Threshold,
		IncludeVector: pq.IncludeVector,
		Filters:       pq.Filters,
	}
}

// convertProviderVectorSearchResultToMemory converts VectorSearchResult to memory.VectorSearchResult
func convertProviderVectorSearchResultToMemory(psr *VectorSearchResult) *memory.VectorSearchResult {
	if psr == nil {
		return nil
	}
	// Convert VectorSearchResultItem slice to memory format
	results := make([]*memory.VectorSearchResultItem, len(psr.Results))
	for i, item := range psr.Results {
		results[i] = &memory.VectorSearchResultItem{
			ID:       item.ID,
			Vector:   item.Vector,
			Metadata: item.Metadata,
			Score:    item.Score,
			Distance: item.Distance,
		}
	}
	return &memory.VectorSearchResult{
		Results:   results,
		Total:     psr.Total,
		Query:     convertProviderVectorQueryToMemory(psr.Query),
		Duration:  psr.Duration,
		Namespace: psr.Namespace,
	}
}

// convertProviderVectorSimilarityResultSliceToMemory converts []*VectorSimilarityResult to []*memory.VectorSimilarityResult
func convertProviderVectorSimilarityResultSliceToMemorySingle(psrs []*VectorSimilarityResult) []*memory.VectorSimilarityResult {
	if psrs == nil {
		return nil
	}
	result := make([]*memory.VectorSimilarityResult, len(psrs))
	for i, r := range psrs {
		result[i] = &memory.VectorSimilarityResult{
			ID:    r.ID,
			Score: r.Score,
		}
	}
	return result
}

// convertProviderVectorSimilarityResultSliceToMemory converts [][]*VectorSimilarityResult to [][]*memory.VectorSimilarityResult
func convertProviderVectorSimilarityResultSliceToMemory(psrs [][]*VectorSimilarityResult) [][]*memory.VectorSimilarityResult {
	if psrs == nil {
		return nil
	}
	result := make([][]*memory.VectorSimilarityResult, len(psrs))
	for i, psr := range psrs {
		result[i] = make([]*memory.VectorSimilarityResult, len(psr))
		for j, r := range psr {
			result[i][j] = &memory.VectorSimilarityResult{
				ID:    r.ID,
				Score: r.Score,
			}
		}
	}
	return result
}

// convertMemoryCollectionConfigToProvider converts memory.CollectionConfig to CollectionConfig
func convertMemoryCollectionConfigToProvider(mcc *memory.CollectionConfig) *CollectionConfig {
	if mcc == nil {
		return nil
	}
	return &CollectionConfig{
		Name:      mcc.Name,
		Dimension: mcc.Dimension,
		Metric:    mcc.Metric,
		Replicas:  mcc.Replicas,
		Shards:    mcc.Shards,
		Metadata:  mcc.Properties, // Map Properties to Metadata
	}
}

// convertProviderCollectionInfoSliceToMemory converts []*CollectionInfo to []*memory.CollectionInfo
func convertProviderCollectionInfoSliceToMemory(pcis []*CollectionInfo) []*memory.CollectionInfo {
	if pcis == nil {
		return nil
	}
	result := make([]*memory.CollectionInfo, len(pcis))
	for i, pci := range pcis {
		result[i] = &memory.CollectionInfo{
			Name:         pci.Name,
			Dimension:    pci.Dimension,
			Metric:       pci.Metric,
			VectorsCount: pci.Size, // Map Size to VectorsCount
			CreatedAt:    pci.CreatedAt,
			Metadata:     pci.Metadata,
		}
	}
	return result
}

// convertProviderCollectionInfoToMemory converts CollectionInfo to memory.CollectionInfo
func convertProviderCollectionInfoToMemory(pci *CollectionInfo) *memory.CollectionInfo {
	if pci == nil {
		return nil
	}
	return &memory.CollectionInfo{
		Name:         pci.Name,
		Dimension:    pci.Dimension,
		Metric:       pci.Metric,
		VectorsCount: pci.Size, // Map Size to VectorsCount
		CreatedAt:    pci.CreatedAt,
		Metadata:     pci.Metadata,
	}
}

// convertMemoryIndexConfigToProvider converts memory.IndexConfig to IndexConfig
func convertMemoryIndexConfigToProvider(mic *memory.IndexConfig) *IndexConfig {
	if mic == nil {
		return nil
	}
	return &IndexConfig{
		Name:     mic.Name,
		Type:     mic.Type,
		Options:  mic.Parameters, // Map Parameters to Options
		Metadata: make(map[string]interface{}),
	}
}

// convertProviderIndexInfoSliceToMemory converts []*IndexInfo to []*memory.IndexInfo
func convertProviderIndexInfoSliceToMemory(piis []*IndexInfo) []*memory.IndexInfo {
	if piis == nil {
		return nil
	}
	result := make([]*memory.IndexInfo, len(piis))
	for i, pii := range piis {
		result[i] = &memory.IndexInfo{
			Name:      pii.Name,
			Type:      pii.Type,
			Status:    pii.State, // Map State to Status
			CreatedAt: pii.CreatedAt,
			Metadata:  pii.Metadata,
		}
	}
	return result
}

// convertProviderStatsToMemory converts ProviderStats to memory.ProviderStats
func convertProviderStatsToMemory(ps *ProviderStats) *memory.ProviderStats {
	if ps == nil {
		return nil
	}
	return &memory.ProviderStats{
		TotalVectors:     ps.TotalVectors,
		TotalCollections: ps.TotalCollections,
		StorageSize:      ps.TotalSize,
		AvgResponseTime:  ps.AverageLatency,
	}
}

// convertProviderCostInfoToMemory converts CostInfo to memory.CostInfo
func convertProviderCostInfoToMemory(ci *CostInfo) *memory.CostInfo {
	if ci == nil {
		return nil
	}
	return &memory.CostInfo{
		Currency:    ci.Currency,
		ReadCost:    ci.ComputeCost,  // Map ComputeCost to ReadCost
		WriteCost:   ci.TransferCost, // Map TransferCost to WriteCost
		StorageCost: ci.StorageCost,
	}
}
