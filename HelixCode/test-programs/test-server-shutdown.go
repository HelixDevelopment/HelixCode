//go:build testprograms

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
)

func main() {
	fmt.Println("🚀 Starting Minimal Test Server")
	fmt.Println("📝 This server will help diagnose the shutdown issue")
	fmt.Println("⏰ Server started at:", time.Now())

	// Create a simple HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello from test server! Time: %s\n", time.Now().Format(time.RFC3339))
		}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  300 * time.Second, // 5 minutes
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🌐 Starting HTTP server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Log every 10 seconds to track server status
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for t := range ticker.C {
			log.Printf("⏰ Server still running at: %s", t.Format(time.RFC3339))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")
	ticker.Stop()

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited properly at:", time.Now())
}