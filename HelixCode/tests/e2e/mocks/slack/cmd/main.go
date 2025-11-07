package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"dev.helix.code/tests/e2e/mocks/slack/config"
	"dev.helix.code/tests/e2e/mocks/slack/handlers"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	if !cfg.EnableLogging {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Create handlers
	messagesHandler := handlers.NewMessagesHandler(cfg)
	webhooksHandler := handlers.NewWebhooksHandler(cfg)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "mock-slack",
		})
	})

	// Slack API endpoints
	api := router.Group("/api")
	{
		// Message posting endpoint (Slack-compatible)
		api.POST("/chat.postMessage", messagesHandler.HandlePostMessage)

		// Custom testing endpoints
		api.GET("/messages", messagesHandler.HandleGetMessages)
		api.DELETE("/messages", messagesHandler.HandleClearMessages)

		// Webhook inspection endpoints
		api.GET("/webhooks", webhooksHandler.HandleGetWebhooks)
		api.DELETE("/webhooks", webhooksHandler.HandleClearWebhooks)
	}

	// Webhook endpoints (Slack-compatible incoming webhooks)
	router.POST("/webhook/:id", webhooksHandler.HandleWebhook)
	router.POST("/services/:service/:key/:id", webhooksHandler.HandleWebhook) // Slack webhook format

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Mock Slack Service starting on %s", addr)
	log.Printf("Response delay: %v", cfg.ResponseDelay)
	log.Printf("Storage capacity: %d messages", cfg.StorageCapacity)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
