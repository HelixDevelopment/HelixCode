package handlers

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"dev.helix.code/tests/e2e/mocks/llm-provider/config"
	"dev.helix.code/tests/e2e/mocks/llm-provider/responses"
)

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Object string           `json:"object"`
	Data   []EmbeddingData  `json:"data"`
	Model  string           `json:"model"`
	Usage  EmbeddingUsage   `json:"usage"`
}

// EmbeddingData represents a single embedding
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage represents token usage for embeddings
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingsHandler handles embedding requests
type EmbeddingsHandler struct {
	config   *config.Config
	fixtures *responses.Fixtures
	rand     *rand.Rand
}

// NewEmbeddingsHandler creates a new embeddings handler
func NewEmbeddingsHandler(cfg *config.Config, fixtures *responses.Fixtures) *EmbeddingsHandler {
	return &EmbeddingsHandler{
		config:   cfg,
		fixtures: fixtures,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Handle processes an embedding request
func (h *EmbeddingsHandler) Handle(c *gin.Context) {
	var req EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Simulate processing delay
	time.Sleep(h.config.ResponseDelay)

	// Generate mock embeddings
	data := make([]EmbeddingData, len(req.Input))
	totalTokens := 0

	for i, input := range req.Input {
		embedding := h.generateMockEmbedding(h.fixtures.Embeddings.Dimension)
		data[i] = EmbeddingData{
			Object:    "embedding",
			Embedding: embedding,
			Index:     i,
		}
		// Estimate tokens (roughly 1 token per 4 characters)
		totalTokens += len(input) / 4
	}

	response := EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  req.Model,
		Usage: EmbeddingUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}

	c.JSON(http.StatusOK, response)
}

// generateMockEmbedding generates a random embedding vector
func (h *EmbeddingsHandler) generateMockEmbedding(dimension int) []float64 {
	embedding := make([]float64, dimension)
	for i := range embedding {
		// Generate values between -1 and 1
		embedding[i] = h.rand.Float64()*2 - 1
	}
	return embedding
}
