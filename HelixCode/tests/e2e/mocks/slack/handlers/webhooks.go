package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dev.helix.code/tests/e2e/mocks/slack/config"
)

// WebhookPayload represents a webhook payload
type WebhookPayload struct {
	ID        string                 `json:"id"`
	Text      string                 `json:"text"`
	Channel   string                 `json:"channel,omitempty"`
	Username  string                 `json:"username,omitempty"`
	IconEmoji string                 `json:"icon_emoji,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// WebhooksHandler handles webhook-related requests
type WebhooksHandler struct {
	config   *config.Config
	webhooks []WebhookPayload
	mu       sync.RWMutex
}

// NewWebhooksHandler creates a new webhooks handler
func NewWebhooksHandler(cfg *config.Config) *WebhooksHandler {
	return &WebhooksHandler{
		config:   cfg,
		webhooks: make([]WebhookPayload, 0),
	}
}

// HandleWebhook handles POST /webhook/:id (incoming webhooks)
func (h *WebhooksHandler) HandleWebhook(c *gin.Context) {
	// Simulate response delay
	time.Sleep(h.config.ResponseDelay)

	webhookID := c.Param("id")
	if webhookID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "webhook_id_required",
		})
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid_payload",
		})
		return
	}

	// Extract common fields
	text, _ := payload["text"].(string)
	channel, _ := payload["channel"].(string)
	username, _ := payload["username"].(string)
	iconEmoji, _ := payload["icon_emoji"].(string)

	// Create webhook record
	webhook := WebhookPayload{
		ID:        uuid.New().String(),
		Text:      text,
		Channel:   channel,
		Username:  username,
		IconEmoji: iconEmoji,
		Timestamp: time.Now(),
		Extra:     payload,
	}

	// Store webhook
	h.mu.Lock()
	h.webhooks = append(h.webhooks, webhook)
	// Limit storage capacity
	if len(h.webhooks) > h.config.StorageCapacity {
		h.webhooks = h.webhooks[1:]
	}
	h.mu.Unlock()

	// Return success response (Slack format)
	c.String(http.StatusOK, "ok")
}

// HandleGetWebhooks handles GET /api/webhooks (custom endpoint for testing)
func (h *WebhooksHandler) HandleGetWebhooks(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"webhooks": h.webhooks,
		"count":    len(h.webhooks),
	})
}

// HandleClearWebhooks handles DELETE /api/webhooks (custom endpoint for testing)
func (h *WebhooksHandler) HandleClearWebhooks(c *gin.Context) {
	h.mu.Lock()
	h.webhooks = make([]WebhookPayload, 0)
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": "All webhooks cleared",
	})
}
