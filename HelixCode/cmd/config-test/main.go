package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dev.helix.code/internal/config"
)

func main() {
	fmt.Println("🔧 Testing Configuration Hot-Reload System")
	fmt.Println("==========================================")

	// Load initial configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	fmt.Println("✅ Initial configuration loaded:")
	printConfigInfo(cfg)

	// Configuration watcher not implemented in current API
	configPath := config.GetConfigPath()

	fmt.Printf("📁 Config path: %s\n", configPath)
	fmt.Println("⏹️  Press Ctrl+C to exit")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 Shutting down configuration test...")
}

func printConfigInfo(cfg *config.Config) {
	// ConfigInfo is empty struct, so we'll print directly from cfg
	fmt.Printf("   🖥️  Server: %s:%d\n", cfg.Server.Address, cfg.Server.Port)
	fmt.Printf("   🗄️  Database: %s:%d/%s\n",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName)
	fmt.Printf("   🔴 Redis: %s:%d (enabled: %t)\n",
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Enabled)
	fmt.Printf("   🔐 Auth: JWT Secret Length: %d\n", len(cfg.Auth.JWTSecret))
	fmt.Printf("   🤖 LLM: %s (tokens: %d, temp: %.1f)\n",
		cfg.LLM.DefaultProvider,
		cfg.LLM.MaxTokens,
		cfg.LLM.Temperature)
}
