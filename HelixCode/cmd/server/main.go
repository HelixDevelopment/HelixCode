package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/redis"
	"dev.helix.code/internal/server"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	fmt.Printf("🚀 Starting HelixCode Server v%s\n", version)
	fmt.Printf("📅 Build: %s\n", buildTime)
	fmt.Printf("🔧 Commit: %s\n", gitCommit)
	fmt.Println()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize database schema
	if err := db.InitializeSchema(); err != nil {
		log.Fatalf("❌ Failed to initialize database schema: %v", err)
	}

	// Initialize Redis
	rds, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Redis: %v", err)
	}
	defer rds.Close()

	// Create HTTP server
	srv := server.New(cfg, db, rds)

	// Start server in a goroutine
	go func() {
		log.Printf("🌐 Starting HTTP server on %s", cfg.Server.Address)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited properly")
}
