package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"dev.helix.code/internal/auth"
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/mcp"
	"dev.helix.code/internal/notification"
	"dev.helix.code/internal/redis"
	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server
type Server struct {
	config       *config.Config
	db           *database.Database
	redis        *redis.Client
	auth         *auth.AuthService
	llm          *llm.Provider
	mcp          *mcp.MCPServer
	notification *notification.NotificationEngine
	server       *http.Server
	router       *gin.Engine
}

// New creates a new HTTP server
func New(cfg *config.Config, db *database.Database, rds *redis.Client) *Server {
	// Set Gin mode
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())
	router.Use(SecurityMiddleware())

	// Initialize auth service
	var authService *auth.AuthService
	if db != nil {
		authConfig := auth.AuthConfig{
			JWTSecret:     cfg.Auth.JWTSecret,
			TokenExpiry:   time.Duration(cfg.Auth.TokenExpiry) * time.Second,
			SessionExpiry: time.Duration(cfg.Auth.SessionExpiry) * time.Second,
			BcryptCost:    cfg.Auth.BcryptCost,
		}
		authDB := auth.NewAuthDB(db.Pool)
		authService = auth.NewAuthService(authConfig, authDB)
	}

	// Initialize MCP server
	mcpServer := mcp.NewMCPServer()

	// Initialize notification engine
	notificationEngine := notification.NewNotificationEngine()

	// Initialize LLM provider (basic setup - would be configured based on config)
	// For now, we'll skip LLM initialization as it requires more complex setup

	server := &Server{
		config:       cfg,
		db:           db,
		redis:        rds,
		auth:         authService,
		mcp:          mcpServer,
		notification: notificationEngine,
		router:       router,
	}

	// Setup routes
	server.setupRoutes()

	// Create HTTP server
	server.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("🚀 Starting HelixCode server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// setupRoutes sets up all HTTP routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)

	// API routes
	api := s.router.Group("/api/v1")
	{
		// Authentication routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", s.register)
			auth.POST("/login", s.login)
			auth.POST("/logout", s.logout)
			auth.POST("/refresh", s.refreshToken)
		}

		// User routes
		users := api.Group("/users")
		users.Use(s.authMiddleware())
		{
			users.GET("/me", s.getCurrentUser)
			users.PUT("/me", s.notImplemented)
			users.DELETE("/me", s.notImplemented)
		}

		// Worker routes
		workers := api.Group("/workers")
		workers.Use(s.authMiddleware())
		{
			workers.GET("", s.listWorkers)
			workers.POST("", s.notImplemented)
			workers.GET("/:id", s.getWorker)
			workers.PUT("/:id", s.notImplemented)
			workers.DELETE("/:id", s.notImplemented)
			workers.POST("/:id/heartbeat", s.notImplemented)
			workers.GET("/:id/metrics", s.notImplemented)
		}

		// Task routes
		tasks := api.Group("/tasks")
		tasks.Use(s.authMiddleware())
		{
			tasks.GET("", s.listTasks)
			tasks.POST("", s.createTask)
			tasks.GET("/:id", s.getTask)
			tasks.PUT("/:id", s.updateTask)
			tasks.DELETE("/:id", s.deleteTask)
			tasks.POST("/:id/assign", s.notImplemented)
			tasks.POST("/:id/start", s.notImplemented)
			tasks.POST("/:id/complete", s.notImplemented)
			tasks.POST("/:id/fail", s.notImplemented)
			tasks.POST("/:id/checkpoint", s.notImplemented)
			tasks.GET("/:id/checkpoints", s.notImplemented)
			tasks.POST("/:id/retry", s.notImplemented)
		}

		// Project routes
		projects := api.Group("/projects")
		projects.Use(s.authMiddleware())
		{
			projects.GET("", s.listProjects)
			projects.POST("", s.createProject)
			projects.GET("/:id", s.getProject)
			projects.PUT("/:id", s.updateProject)
			projects.DELETE("/:id", s.deleteProject)
			projects.GET("/:id/sessions", s.notImplemented)

			// Workflow routes
			projects.POST("/:projectId/workflows/planning", s.executePlanningWorkflow)
			projects.POST("/:projectId/workflows/building", s.executeBuildingWorkflow)
			projects.POST("/:projectId/workflows/testing", s.executeTestingWorkflow)
			projects.POST("/:projectId/workflows/refactoring", s.executeRefactoringWorkflow)
		}

		// Session routes
		sessions := api.Group("/sessions")
		sessions.Use(s.authMiddleware())
		{
			sessions.GET("", s.notImplemented)
			sessions.POST("", s.notImplemented)
			sessions.GET("/:id", s.notImplemented)
			sessions.PUT("/:id", s.notImplemented)
			sessions.DELETE("/:id", s.notImplemented)
		}

		// System routes
		system := api.Group("/system")
		system.Use(s.authMiddleware())
		{
			system.GET("/stats", s.getSystemStats)
			system.GET("/status", s.getSystemStatus)
		}
	}

	// WebSocket routes
	s.router.GET("/ws", s.handleWebSocket)

	// Static file serving for web interface
	s.router.Static("/static", "./web/frontend/static")
	s.router.StaticFile("/", "./web/frontend/index.html")
	s.router.StaticFile("/favicon.ico", "./assets/icons/icon-32x32.png")
}

// Handler methods

func (s *Server) healthCheck(c *gin.Context) {
	// Check database connection
	if s.db != nil {
		if err := s.db.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "Database connection failed",
				"error":   err.Error(),
			})
			return
		}
	}

	// Check Redis connection if enabled
	if s.redis.IsEnabled() {
		if _, err := s.redis.Get(c.Request.Context(), "health_check"); err != nil && err.Error() != "redis: nil" {
			// Try to ping Redis
			if s.redis.GetClient().Ping(c.Request.Context()).Err() != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status":  "error",
					"message": "Redis connection failed",
					"error":   err.Error(),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC(),
	})
}

func (s *Server) notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"status":  "error",
		"message": "Not implemented yet",
	})
}

// Middleware

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Check for Bearer token
		const bearerPrefix = "Bearer "
		if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := authHeader[len(bearerPrefix):]

		// Verify JWT token
		user, err := s.auth.VerifyJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid or expired token",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Next()
	}
}

// handleWebSocket handles WebSocket connections for MCP
func (s *Server) handleWebSocket(c *gin.Context) {
	s.mcp.HandleWebSocket(c.Writer, c.Request)
}

// CORSMiddleware provides CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityMiddleware provides security headers
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}
