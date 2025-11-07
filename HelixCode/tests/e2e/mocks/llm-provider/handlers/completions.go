package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dev.helix.code/tests/e2e/mocks/llm-provider/config"
	"dev.helix.code/tests/e2e/mocks/llm-provider/responses"
)

// CompletionRequest represents a chat completion request
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse represents a chat completion response
type CompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionsHandler handles chat completion requests
type CompletionsHandler struct {
	config   *config.Config
	fixtures *responses.Fixtures
}

// NewCompletionsHandler creates a new completions handler
func NewCompletionsHandler(cfg *config.Config, fixtures *responses.Fixtures) *CompletionsHandler {
	return &CompletionsHandler{
		config:   cfg,
		fixtures: fixtures,
	}
}

// Handle processes a chat completion request
func (h *CompletionsHandler) Handle(c *gin.Context) {
	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Simulate processing delay
	time.Sleep(h.config.ResponseDelay)

	// Get the last user message
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	// Find appropriate response
	responseContent := h.fixtures.FindResponse(userMessage)

	// Calculate token usage
	promptTokens := estimateTokens(req.Messages)
	completionTokens := estimateTokens([]Message{{Content: responseContent}})

	response := CompletionResponse{
		ID:      "chatcmpl-" + uuid.New().String(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: responseContent,
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}

	c.JSON(http.StatusOK, response)
}

// estimateTokens estimates the number of tokens in messages
func estimateTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		// Rough estimation: ~4 characters per token
		total += len(msg.Content) / 4
	}
	return total
}

// extractLastUserMessage extracts the last user message from the conversation
func extractLastUserMessage(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.ToLower(messages[i].Role) == "user" {
			return messages[i].Content
		}
	}
	return ""
}
