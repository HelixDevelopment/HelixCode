package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"dev.helix.code/tests/e2e/mocks/llm-provider/config"
	"dev.helix.code/tests/e2e/mocks/llm-provider/handlers"
	"dev.helix.code/tests/e2e/mocks/llm-provider/responses"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Load fixtures
	fixtures, err := responses.LoadFixtures(cfg.FixturesPath)
	if err != nil {
		log.Fatalf("Failed to load fixtures: %v", err)
	}

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
	completionsHandler := handlers.NewCompletionsHandler(cfg, fixtures)
	embeddingsHandler := handlers.NewEmbeddingsHandler(cfg, fixtures)
	modelsHandler := handlers.NewModelsHandler(cfg, fixtures)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "mock-llm-provider",
		})
	})

	// API routes - OpenAI-compatible
	v1 := router.Group("/v1")
	{
		// Chat completions endpoint
		v1.POST("/chat/completions", completionsHandler.Handle)

		// Embeddings endpoint
		v1.POST("/embeddings", embeddingsHandler.Handle)

		// Models endpoints
		v1.GET("/models", modelsHandler.HandleList)
		v1.GET("/models/:model", modelsHandler.HandleGet)
	}

	// Alternative routes for different provider formats
	router.POST("/api/chat", completionsHandler.Handle) // Anthropic-style
	router.POST("/api/generate", completionsHandler.Handle) // Ollama-style
	router.POST("/completions", completionsHandler.Handle) // Simple style

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Mock LLM Provider starting on %s", addr)
	log.Printf("Default model: %s", cfg.DefaultModel)
	log.Printf("Response delay: %v", cfg.ResponseDelay)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
