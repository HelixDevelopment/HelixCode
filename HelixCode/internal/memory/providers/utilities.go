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
