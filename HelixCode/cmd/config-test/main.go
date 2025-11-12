package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Set up configuration watcher
	configPath, err := config.GetConfigPath()
	if err != nil {
		log.Fatalf("❌ Failed to get config path: %v", err)
	}

	watcher, err := config.NewConfigWatcher(configPath, func(newCfg *config.Config) {
		fmt.Printf("\n🔄 Configuration reloaded at %s\n", time.Now().Format("15:04:05"))
		printConfigInfo(newCfg)
	})
	if err != nil {
		log.Fatalf("❌ Failed to create config watcher: %v", err)
	}
	defer watcher.Stop()

	fmt.Printf("👀 Watching for changes in: %s\n", configPath)
	fmt.Println("💡 Try editing the config file and saving it...")
	fmt.Println("⏹️  Press Ctrl+C to exit")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 Shutting down configuration test...")
}

func printConfigInfo(cfg *config.Config) {
	info := config.GetConfigInfo(cfg)
	
	fmt.Printf("   🖥️  Server: %s:%d\n", info["server"].(map[string]interface{})["address"], info["server"].(map[string]interface{})["port"])
	fmt.Printf("   🗄️  Database: %s:%d/%s\n", 
		info["database"].(map[string]interface{})["host"],
		info["database"].(map[string]interface{})["port"],
		info["database"].(map[string]interface{})["database"])
	fmt.Printf("   🔴 Redis: %s:%d (enabled: %t)\n", 
		info["redis"].(map[string]interface{})["host"],
		info["redis"].(map[string]interface{})["port"],
		info["redis"].(map[string]interface{})["enabled"])
	fmt.Printf("   🔐 Auth: JWT Secret Length: %d\n", len(cfg.Auth.JWTSecret))
	fmt.Printf("   🤖 LLM: %s (tokens: %d, temp: %.1f)\n", 
		info["llm"].(map[string]interface{})["default_provider"],
		info["llm"].(map[string]interface{})["max_tokens"],
		info["llm"].(map[string]interface{})["temperature"])
}