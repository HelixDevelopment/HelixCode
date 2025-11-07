package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"dev.helix.code/tests/e2e/mocks/llm-provider/config"
	"dev.helix.code/tests/e2e/mocks/llm-provider/responses"
)

// ModelsResponse represents a list of models response
type ModelsResponse struct {
	Object string             `json:"object"`
	Data   []responses.Model  `json:"data"`
}

// ModelsHandler handles model listing requests
type ModelsHandler struct {
	config   *config.Config
	fixtures *responses.Fixtures
}

// NewModelsHandler creates a new models handler
func NewModelsHandler(cfg *config.Config, fixtures *responses.Fixtures) *ModelsHandler {
	return &ModelsHandler{
		config:   cfg,
		fixtures: fixtures,
	}
}

// HandleList returns a list of all available models
func (h *ModelsHandler) HandleList(c *gin.Context) {
	response := ModelsResponse{
		Object: "list",
		Data:   h.fixtures.GetModels(),
	}

	c.JSON(http.StatusOK, response)
}

// HandleGet returns details for a specific model
func (h *ModelsHandler) HandleGet(c *gin.Context) {
	modelID := c.Param("model")

	model := h.fixtures.GetModel(modelID)
	if model == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "Model not found",
				"type":    "invalid_request_error",
				"param":   "model",
				"code":    "model_not_found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, model)
}
