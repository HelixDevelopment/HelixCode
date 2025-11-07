package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dev.helix.code/tests/e2e/mocks/slack/config"
)

// Message represents a Slack message
type Message struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	Text      string    `json:"text"`
	Username  string    `json:"username,omitempty"`
	IconEmoji string    `json:"icon_emoji,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	ThreadTS  string    `json:"thread_ts,omitempty"`
}

// PostMessageRequest represents a Slack message posting request
type PostMessageRequest struct {
	Channel   string `json:"channel" binding:"required"`
	Text      string `json:"text" binding:"required"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	ThreadTS  string `json:"thread_ts,omitempty"`
}

// PostMessageResponse represents a Slack message posting response
type PostMessageResponse struct {
	OK      bool   `json:"ok"`
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	Message Message `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// MessagesHandler handles message-related requests
type MessagesHandler struct {
	config   *config.Config
	messages []Message
	mu       sync.RWMutex
}

// NewMessagesHandler creates a new messages handler
func NewMessagesHandler(cfg *config.Config) *MessagesHandler {
	return &MessagesHandler{
		config:   cfg,
		messages: make([]Message, 0),
	}
}

// HandlePostMessage handles POST /api/chat.postMessage
func (h *MessagesHandler) HandlePostMessage(c *gin.Context) {
	// Simulate response delay
	time.Sleep(h.config.ResponseDelay)

	var req PostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PostMessageResponse{
			OK:    false,
			Error: "invalid_request",
		})
		return
	}

	// Validate channel
	if req.Channel == "" {
		c.JSON(http.StatusBadRequest, PostMessageResponse{
			OK:    false,
			Error: "channel_not_found",
		})
		return
	}

	// Create message
	message := Message{
		ID:        uuid.New().String(),
		Channel:   req.Channel,
		Text:      req.Text,
		Username:  req.Username,
		IconEmoji: req.IconEmoji,
		Timestamp: time.Now(),
		ThreadTS:  req.ThreadTS,
	}

	// Store message
	h.mu.Lock()
	h.messages = append(h.messages, message)
	// Limit storage capacity
	if len(h.messages) > h.config.StorageCapacity {
		h.messages = h.messages[1:]
	}
	h.mu.Unlock()

	// Return success response
	c.JSON(http.StatusOK, PostMessageResponse{
		OK:      true,
		Channel: message.Channel,
		TS:      message.ID,
		Message: message,
	})
}

// HandleGetMessages handles GET /api/messages (custom endpoint for testing)
func (h *MessagesHandler) HandleGetMessages(c *gin.Context) {
	channel := c.Query("channel")

	h.mu.RLock()
	defer h.mu.RUnlock()

	if channel == "" {
		// Return all messages
		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"messages": h.messages,
			"count":    len(h.messages),
		})
		return
	}

	// Filter by channel
	filtered := make([]Message, 0)
	for _, msg := range h.messages {
		if msg.Channel == channel {
			filtered = append(filtered, msg)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"channel":  channel,
		"messages": filtered,
		"count":    len(filtered),
	})
}

// HandleClearMessages handles DELETE /api/messages (custom endpoint for testing)
func (h *MessagesHandler) HandleClearMessages(c *gin.Context) {
	h.mu.Lock()
	h.messages = make([]Message, 0)
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": "All messages cleared",
	})
}
