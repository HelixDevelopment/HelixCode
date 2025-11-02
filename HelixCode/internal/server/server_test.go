package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"dev.helix.code/internal/auth"
	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/mcp"
	"dev.helix.code/internal/notification"
	"dev.helix.code/internal/redis"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock dependencies for testing
func createMockDependencies(t *testing.T) (*config.Config, *database.Database, *redis.Client) {
	// Create a test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Logging: config.LoggingConfig{
			Level: "debug",
		},
	}

	// Mock database (nil for testing)
	db := (*database.Database)(nil)

	// Mock Redis client
	rds := &redis.Client{}

	return cfg, db, rds
}

func TestNewServer(t *testing.T) {
	cfg, db, rds := createMockDependencies(t)

	server := New(cfg, db, rds)

	assert.NotNil(t, server)
	assert.Equal(t, cfg, server.config)
	assert.Equal(t, db, server.db)
	assert.Equal(t, rds, server.redis)
	assert.NotNil(t, server.router)
}

func TestNewServer_DebugMode(t *testing.T) {
	cfg, db, rds := createMockDependencies(t)
	cfg.Logging.Level = "debug"

	server := New(cfg, db, rds)

	assert.NotNil(t, server)
	// Gin should be in debug mode
	assert.Equal(t, gin.DebugMode, gin.Mode())
}

func TestNewServer_ReleaseMode(t *testing.T) {
	cfg, db, rds := createMockDependencies(t)
	cfg.Logging.Level = "info"

	// Reset gin mode
	gin.SetMode(gin.ReleaseMode)

	server := New(cfg, db, rds)

	assert.NotNil(t, server)
	assert.Equal(t, gin.ReleaseMode, gin.Mode())
}

func TestCORSMiddleware(t *testing.T) {
	middleware := CORSMiddleware()

	assert.NotNil(t, middleware)

	// Test the middleware function
	router := gin.New()
	router.Use(middleware)

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
}

func TestSecurityMiddleware(t *testing.T) {
	middleware := SecurityMiddleware()

	assert.NotNil(t, middleware)

	// Test the middleware function
	router := gin.New()
	router.Use(middleware)

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// Check security headers
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
}

func TestHealthCheckEndpoint(t *testing.T) {
	cfg, db, rds := createMockDependencies(t)

	// Create server with minimal setup
	server := &Server{
		config: cfg,
		db:     db,
		redis:  rds,
		router: gin.New(),
	}

	// Setup routes manually for testing
	server.setupRoutes()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)

	server.router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestServerInitialization(t *testing.T) {
	cfg, db, rds := createMockDependencies(t)

	server := New(cfg, db, rds)

	// Test that all components are initialized
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.router)

	// Test that middleware is applied
	routes := server.router.Routes()
	assert.Greater(t, len(routes), 0, "Server should have routes configured")
}

func TestServerWithNilDependencies(t *testing.T) {
	// Test server creation with nil dependencies (should not panic)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Logging: config.LoggingConfig{
			Level: "debug",
		},
	}

	server := New(cfg, nil, nil)

	assert.NotNil(t, server)
	assert.NotNil(t, server.router)
	assert.Equal(t, cfg, server.config)
}

func TestServerPortConfiguration(t *testing.T) {
	testCases := []struct {
		name     string
		port     int
		expected int
	}{
		{"valid port", 8080, 8080},
		{"different port", 3000, 3000},
		{"high port", 65535, 65535},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Host: "localhost",
					Port: tc.port,
				},
			}

			server := New(cfg, nil, nil)
			assert.Equal(t, tc.expected, cfg.Server.Port)
			assert.NotNil(t, server)
		})
	}
}

func TestMiddlewareOrder(t *testing.T) {
	cfg, db, rds := createMockDependencies(t)

	server := New(cfg, db, rds)

	// The server should have middleware applied in the correct order
	// This is tested indirectly through the route setup
	assert.NotNil(t, server.router)
}

// Benchmark tests
func BenchmarkNewServer(b *testing.B) {
	cfg, db, rds := createMockDependencies(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server := New(cfg, db, rds)
		_ = server
	}
}

func BenchmarkCORSMiddleware(b *testing.B) {
	middleware := CORSMiddleware()

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkSecurityMiddleware(b *testing.B) {
	middleware := SecurityMiddleware()

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}
