package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/helixcode/internal/llm"
)

// localLLMCmd represents the local-llm command
var localLLMCmd = &cobra.Command{
	Use:   "local-llm",
	Short: "Manage local LLM providers",
	Long: `Manage local LLM providers including VLLM, LocalAI, FastChat, TextGen,
LM Studio, Jan AI, KoboldAI, GPT4All, TabbyAPI, MLX, and MistralRS.

This command automatically clones, builds, configures, and manages all local
LLM providers with zero configuration required.`,
}

var (
	localLLMDir      string
	autoStart        bool
	healthInterval   int
	selectedProvider string
)

func init() {
	rootCmd.AddCommand(localLLMCmd)

	// Persistent flags
	localLLMCmd.PersistentFlags().StringVar(&localLLMDir, "dir", "", "Base directory for local LLM providers (default: ~/.helixcode/local-llm)")
	localLLMCmd.PersistentFlags().BoolVar(&autoStart, "auto-start", true, "Auto-start all providers after initialization")
	localLLMCmd.PersistentFlags().IntVar(&healthInterval, "health-interval", 30, "Health check interval in seconds")

	// Subcommands
	localLLMCmd.AddCommand(initCmd)
	localLLMCmd.AddCommand(startCmd)
	localLLMCmd.AddCommand(stopCmd)
	localLLMCmd.AddCommand(statusCmd)
	localLLMCmd.AddCommand(listCmd)
	localLLMCmd.AddCommand(cleanupCmd)
	localLLMCmd.AddCommand(updateCmd)
	localLLMCmd.AddCommand(logsCmd)
}

// initCmd represents the local-llm init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize and install all local LLM providers",
	Long: `Initialize and install all local LLM providers. This command will:
- Clone provider repositories
- Build and configure providers
- Create startup scripts
- Set up default configurations

This may take 10-30 minutes depending on your system and internet speed.`,
	RunE: runInit,
}

// startCmd represents the local-llm start command
var startCmd = &cobra.Command{
	Use:   "start [provider]",
	Short: "Start local LLM providers",
	Long: `Start local LLM providers. If no provider is specified, all providers will be started.
You can start individual providers by specifying their name.

Available providers: vllm, localai, fastchat, textgen, lmstudio, jan, koboldai, gpt4all, tabbyapi, mlx, mistralrs`,
	RunE: runStart,
}

// stopCmd represents the local-llm stop command
var stopCmd = &cobra.Command{
	Use:   "stop [provider]",
	Short: "Stop local LLM providers",
	Long: `Stop local LLM providers. If no provider is specified, all providers will be stopped.
You can stop individual providers by specifying their name.`,
	RunE: runStop,
}

// statusCmd represents the local-llm status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all local LLM providers",
	Long: `Show detailed status of all local LLM providers including:
- Installation status
- Running status
- Health check results
- Process information
- Last check timestamp`,
	RunE: runStatus,
}

// listCmd represents the local-llm list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available local LLM providers",
	Long: `List all available local LLM providers with their descriptions,
default ports, and current status.`,
	RunE: runList,
}

// cleanupCmd represents the local-llm cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Stop all providers and clean up resources",
	Long: `Stop all running local LLM providers and clean up temporary resources.
Downloaded models and configurations will be preserved.`,
	RunE: runCleanup,
}

// updateCmd represents the local-llm update command
var updateCmd = &cobra.Command{
	Use:   "update [provider]",
	Short: "Update local LLM providers",
	Long: `Update local LLM providers to their latest versions.
If no provider is specified, all providers will be updated.`,
	RunE: runUpdate,
}

// logsCmd represents the local-llm logs command
var logsCmd = &cobra.Command{
	Use:   "logs [provider]",
	Short: "Show logs for local LLM providers",
	Long: `Show logs for local LLM providers. If no provider is specified,
logs for all running providers will be displayed.`,
	RunE: runLogs,
}

// Command implementations

func runInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	fmt.Println("🚀 Initializing Local LLM Provider Manager...")
	
	// Create manager instance
	manager := llm.NewLocalLLMManager(localLLMDir)
	
	// Initialize (clone, build, configure)
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	fmt.Println("✅ Initialization complete!")
	
	// Auto-start if requested
	if autoStart {
		fmt.Println("\n🚀 Auto-starting all providers...")
		if err := manager.StartAllProviders(ctx); err != nil {
			return fmt.Errorf("failed to start providers: %w", err)
		}
		
		// Wait a bit for providers to start
		time.Sleep(5 * time.Second)
		
		// Show status
		return runStatus(cmd, args)
	}
	
	return nil
}

func runStart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	manager := llm.NewLocalLLMManager(localLLMDir)
	
	// Ensure manager is initialized
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	if len(args) == 0 {
		// Start all providers
		fmt.Println("🚀 Starting all local LLM providers...")
		return manager.StartAllProviders(ctx)
	}
	
	// Start specific provider
	providerName := args[0]
	fmt.Printf("🚀 Starting provider: %s\n", providerName)
	return manager.StartProvider(ctx, providerName)
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	manager := llm.NewLocalLLMManager(localLLMDir)
	
	if len(args) == 0 {
		// Stop all providers
		fmt.Println("🛑 Stopping all local LLM providers...")
		return manager.StopAllProviders(ctx)
	}
	
	// Stop specific provider
	providerName := args[0]
	fmt.Printf("🛑 Stopping provider: %s\n", providerName)
	return manager.StopProvider(ctx, providerName)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	manager := llm.NewLocalLLMManager(localLLMDir)
	
	// Get provider status
	status := manager.GetProviderStatus(ctx)
	
	if len(status) == 0 {
		fmt.Println("❌ No local LLM providers found. Run 'helix local-llm init' to install providers.")
		return nil
	}
	
	// Display status in tabular format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tSTATUS\tPORT\tLAST CHECK")
	fmt.Fprintln(w, "--------\t------\t----\t-----------")
	
	for name, provider := range status {
		statusIcon := getStatusIcon(provider.Status)
		fmt.Fprintf(w, "%s\t%s%s\t%d\t%s\n", 
			name, 
			statusIcon, 
			provider.Status, 
			provider.DefaultPort,
			provider.LastCheck.Format("15:04:05"))
	}
	
	w.Flush()
	
	// Show running endpoints
	running := manager.GetRunningProviders(ctx)
	if len(running) > 0 {
		fmt.Println("\n📡 Running Provider Endpoints:")
		for _, endpoint := range running {
			fmt.Printf("  • %s\n", endpoint)
		}
	}
	
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tDESCRIPTION\tPORT\tTYPE")
	fmt.Fprintln(w, "--------\t-----------\t----\t----")
	
	providers := []struct{
		name string
		desc string
		port int
		typ  string
	}{
		{"vllm", "High-throughput inference engine", 8000, "OpenAI-compatible"},
		{"localai", "Drop-in OpenAI replacement", 8080, "OpenAI-compatible"},
		{"fastchat", "Training and serving platform", 7860, "OpenAI-compatible"},
		{"textgen", "Popular Gradio interface", 5000, "OpenAI-compatible"},
		{"lmstudio", "User-friendly desktop app", 1234, "OpenAI-compatible"},
		{"jan", "Open-source AI assistant", 1337, "OpenAI-compatible"},
		{"koboldai", "Writing-focused interface", 5001, "Custom API"},
		{"gpt4all", "CPU-focused inference", 4891, "OpenAI-compatible"},
		{"tabbyapi", "High-performance server", 5000, "OpenAI-compatible"},
		{"mlx", "Apple Silicon optimized", 8080, "OpenAI-compatible"},
		{"mistralrs", "Rust-based inference", 8080, "OpenAI-compatible"},
	}
	
	for _, p := range providers {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", p.name, p.desc, p.port, p.typ)
	}
	
	w.Flush()
	return nil
}

func runCleanup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	manager := llm.NewLocalLLMManager(localLLMDir)
	
	fmt.Println("🧹 Cleaning up local LLM providers...")
	return manager.Cleanup(ctx)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	manager := llm.NewLocalLLMManager(localLLMDir)
	
	// Ensure manager is initialized
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	if len(args) == 0 {
		// Update all providers
		fmt.Println("🔄 Updating all local LLM providers...")
		status := manager.GetProviderStatus(ctx)
		for name := range status {
			if err := manager.UpdateProvider(ctx, name); err != nil {
				fmt.Printf("⚠️  Failed to update %s: %v\n", name, err)
			} else {
				fmt.Printf("✅ Updated %s\n", name)
			}
		}
	} else {
		// Update specific provider
		providerName := args[0]
		fmt.Printf("🔄 Updating provider: %s\n", providerName)
		return manager.UpdateProvider(ctx, providerName)
	}
	
	return nil
}

func runLogs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Show logs for all providers
		homeDir, _ := os.UserHomeDir()
		logsDir := fmt.Sprintf("%s/.helixcode/local-llm/logs", homeDir)
		fmt.Printf("📋 Log directory: %s\n", logsDir)
		return nil
	}
	
	providerName := args[0]
	homeDir, _ := os.UserHomeDir()
	logFile := fmt.Sprintf("%s/.helixcode/local-llm/logs/%s.log", homeDir, providerName)
	
	fmt.Printf("📋 Showing logs for %s:\n", providerName)
	fmt.Printf("Log file: %s\n\n", logFile)
	
	// Show last 50 lines of log
	cmd := exec.Command("tail", "-50", logFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Helper functions

func getStatusIcon(status string) string {
	switch status {
	case "running":
		return "🟢 "
	case "starting":
		return "🟡 "
	case "failed", "unhealthy":
		return "🔴 "
	case "stopped":
		return "⚪ "
	case "installed":
		return "🔵 "
	default:
		return "⚫ "
	}
}

// runMonitor starts the interactive monitoring mode
func runMonitor(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	manager := llm.NewLocalLLMManager(localLLMDir)
	
	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Initialize manager
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	// Start monitoring loop
	ticker := time.NewTicker(time.Duration(healthInterval) * time.Second)
	defer ticker.Stop()
	
	fmt.Println("🔍 Starting Local LLM Provider Monitoring...")
	fmt.Println("Press Ctrl+C to stop monitoring\n")
	
	for {
		select {
		case <-sigChan:
			fmt.Println("\n👋 Stopping monitoring...")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Clear screen and show status
			clearScreen()
			fmt.Printf("🔍 Local LLM Provider Status - %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
			
			if err := runStatus(cmd, args); err != nil {
				fmt.Printf("❌ Error getting status: %v\n", err)
			}
		}
	}
}

// runWatch starts the watch mode for real-time updates
func runWatch(cmd *cobra.Command, args []string) error {
	fmt.Println("👀 Starting watch mode for local LLM providers...")
	fmt.Println("Changes will be displayed in real-time. Press Ctrl+C to stop.\n")
	
	// This would implement file system watching for provider changes
	// For now, just call monitor
	return runMonitor(cmd, args)
}

func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}