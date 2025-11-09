package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/helixcode/internal/llm"
	"github.com/helixcode/internal/hardware"
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
	
	// Model management commands
	localLLMCmd.AddCommand(modelsCmd)
	localLLMCmd.AddCommand(downloadModelCmd)
	localLLMCmd.AddCommand(convertModelCmd)
	localLLMCmd.AddCommand(listModelsCmd)
	localLLMCmd.AddCommand(searchModelsCmd)
	
	// Cross-provider model sharing commands
	localLLMCmd.AddCommand(shareModelCmd)
	localLLMCmd.AddCommand(downloadAllCmd)
	localLLMCmd.AddCommand(listSharedCmd)
	localLLMCmd.AddCommand(optimizeModelCmd)
	localLLMCmd.AddCommand(syncModelsCmd)
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

// Model management commands

// modelsCmd represents the models command group
var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage LLM models (download, convert, list)",
	Long: `Manage LLM models including downloading from various sources,
converting between formats, and listing available models.`,
}

// downloadModelCmd represents the model download command
var downloadModelCmd = &cobra.Command{
	Use:   "download [model-id]",
	Short: "Download a model from available sources",
	Long: `Download a model from various sources (HuggingFace, TheBloke, etc.)
and convert it to the desired format if needed.

Examples:
  helix local-llm models download llama-3-8b-instruct --format gguf --provider vllm
  helix local-llm models download mistral-7b-instruct --format gptq --provider localai`,
	RunE: runDownloadModel,
}

// convertModelCmd represents the model conversion command
var convertModelCmd = &cobra.Command{
	Use:   "convert [input-path]",
	Short: "Convert a model to a different format",
	Long: `Convert a model from one format to another using specialized tools.

Supported conversions:
  HF -> GGUF (llama.cpp)
  HF -> GPTQ (AutoGPTQ)
  HF -> AWQ (AutoAWQ)
  HF -> FP16/BF16 (transformers)

Examples:
  helix local-llm models convert ./model.hf --format gguf --quantize q4_k_m
  helix local-llm models convert ./model.gguf --format fp16`,
	RunE: runConvertModel,
}

// listModelsCmd represents the list models command
var listModelsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available models",
	Long: `List all available models from the model registry with their
information including available formats, sizes, and requirements.`,
	RunE: runListModels,
}

// searchModelsCmd represents the search models command
var searchModelsCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for models by name, description, or tags",
	Long: `Search for models in the registry by name, description, or tags.
This is useful for finding models for specific tasks or requirements.

Examples:
  helix local-llm models search "code"
  helix local-llm models search "instruct"
  helix local-llm models search "7b"`,
	RunE: runSearchModels,
}

// Model management flags
var (
	downloadFormat       string
	downloadProvider     string
	downloadTargetPath   string
	forceDownload        bool
	convertTargetFormat  string
	convertQuantMethod   string
	convertOptimizeFor   string
	convertTargetHardware string
	shareModelProvider   string
	optimizeProvider     string
	syncAllProviders    bool
)

func init() {
	// Model command flags
	downloadModelCmd.Flags().StringVar(&downloadFormat, "format", "gguf", "Target model format (gguf, gptq, awq, hf, fp16, bf16)")
	downloadModelCmd.Flags().StringVar(&downloadProvider, "provider", "", "Target provider for the model")
	downloadModelCmd.Flags().StringVar(&downloadTargetPath, "output", "", "Custom output path for the model")
	downloadModelCmd.Flags().BoolVar(&forceDownload, "force", false, "Force download even if model already exists")
	
	convertModelCmd.Flags().StringVar(&convertTargetFormat, "format", "", "Target format (required)")
	convertModelCmd.Flags().StringVar(&convertQuantMethod, "quantize", "", "Quantization method (q4_0, q4_k_m, q8_0, etc.)")
	convertModelCmd.Flags().StringVar(&convertOptimizeFor, "optimize", "", "Optimize for (cpu, gpu, mobile)")
	convertModelCmd.Flags().StringVar(&convertTargetHardware, "hardware", "", "Target hardware (nvidia, amd, apple, intel)")
	
	convertModelCmd.MarkFlagRequired("format")
	
	// Cross-provider command flags
	shareModelCmd.Flags().StringVar(&shareModelProvider, "provider", "", "Share with specific provider only (default: all compatible)")
	optimizeModelCmd.Flags().StringVar(&optimizeProvider, "provider", "", "Target provider for optimization (required)")
	syncModelsCmd.Flags().BoolVar(&syncAllProviders, "all", false, "Sync with all providers (default: compatible only)")
	
	optimizeModelCmd.MarkFlagRequired("provider")
}

// Command implementations for model management

func runDownloadModel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	if len(args) == 0 {
		return fmt.Errorf("model ID is required")
	}
	
	modelID := args[0]
	manager := llm.NewModelDownloadManager(getLocalLLMBaseDir())
	
	// Check if model exists
	model, err := manager.GetModelByID(modelID)
	if err != nil {
		return fmt.Errorf("model not found: %w", err)
	}
	
	fmt.Printf("📥 Downloading model: %s\n", model.Name)
	fmt.Printf("📝 Description: %s\n", model.Description)
	fmt.Printf("📊 Model size: %s, Context: %d tokens\n", model.ModelSize, model.ContextSize)
	
	// Get compatible formats
	if downloadProvider == "" {
		fmt.Println("⚠️  No provider specified, showing compatible formats for all providers:")
		formats := map[string]bool{}
		for _, provider := range []string{"vllm", "localai", "ollama", "llamacpp"} {
			compatible, _ := manager.GetCompatibleFormats(provider, modelID)
			for _, format := range compatible {
				formats[string(format)] = true
			}
		}
		var formatList []string
		for format := range formats {
			formatList = append(formatList, format)
		}
		fmt.Printf("Available formats: %s\n", strings.Join(formatList, ", "))
	}
	
	// Validate format compatibility
	if downloadProvider != "" {
		compatible, err := manager.GetCompatibleFormats(downloadProvider, modelID)
		if err != nil {
			return fmt.Errorf("failed to get compatible formats: %w", err)
		}
		
		formatFound := false
		for _, format := range compatible {
			if string(format) == downloadFormat {
				formatFound = true
				break
			}
		}
		
		if !formatFound {
			return fmt.Errorf("format %s is not compatible with provider %s", downloadFormat, downloadProvider)
		}
	}
	
	// Create download request
	req := llm.ModelDownloadRequest{
		ModelID:        modelID,
		Format:         llm.ModelFormat(downloadFormat),
		TargetProvider: downloadProvider,
		TargetPath:     downloadTargetPath,
		ForceDownload:  forceDownload,
	}
	
	// Start download
	progressChan, err := manager.DownloadModel(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	
	// Monitor progress
	fmt.Println("\n🚀 Starting download...")
	lastProgress := -1.0
	
	for progress := range progressChan {
		if progress.Progress != lastProgress {
			fmt.Printf("\r⏳ Progress: %.1f%%", progress.Progress*100)
			if progress.Speed > 0 {
				fmt.Printf(" | Speed: %s/s", formatBytes(progress.Speed))
			}
			if progress.ETA > 0 {
				fmt.Printf(" | ETA: %ds", progress.ETA)
			}
			lastProgress = progress.Progress
		}
		
		if progress.Error != "" {
			fmt.Printf("\n❌ Error: %s\n", progress.Error)
			return fmt.Errorf("download failed: %s", progress.Error)
		}
	}
	
	fmt.Println("\n✅ Download completed successfully!")
	if downloadTargetPath != "" {
		fmt.Printf("📁 Model saved to: %s\n", downloadTargetPath)
	} else if downloadProvider != "" {
		fmt.Printf("📁 Model saved to provider directory: %s\n", filepath.Join(getLocalLLMBaseDir(), downloadProvider, modelID))
	}
	
	return nil
}

func runConvertModel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	if len(args) == 0 {
		return fmt.Errorf("input model path is required")
	}
	
	inputPath := args[0]
	
	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputPath)
	}
	
	// Detect source format
	sourceFormat, err := detectModelFormat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to detect source format: %w", err)
	}
	
	targetFormat := llm.ModelFormat(convertTargetFormat)
	
	fmt.Printf("🔄 Converting model: %s\n", inputPath)
	fmt.Printf("📝 Source format: %s\n", sourceFormat)
	fmt.Printf("🎯 Target format: %s\n", targetFormat)
	
	// Initialize converter
	converter := llm.NewModelConverter(getLocalLLMBaseDir())
	
	// Validate conversion
	result, err := converter.ValidateConversion(sourceFormat, targetFormat)
	if err != nil {
		return fmt.Errorf("conversion validation failed: %w", err)
	}
	
	if !result.IsPossible {
		fmt.Println("❌ Conversion is not possible")
		for _, warning := range result.Warnings {
			fmt.Printf("⚠️  %s\n", warning)
		}
		return fmt.Errorf("conversion not supported")
	}
	
	// Show warnings and recommendations
	for _, warning := range result.Warnings {
		fmt.Printf("⚠️  %s\n", warning)
	}
	for _, recommendation := range result.Recommendations {
		fmt.Printf("💡 %s\n", recommendation)
	}
	
	// Prepare conversion config
	config := llm.ConversionConfig{
		SourcePath:   inputPath,
		SourceFormat: sourceFormat,
		TargetFormat: targetFormat,
		Timeout:      60, // 60 minutes default
	}
	
	// Add quantization if specified
	if convertQuantMethod != "" {
		config.Quantization = &llm.QuantizationConfig{
			Method: convertQuantMethod,
		}
	}
	
	// Add optimization if specified
	if convertOptimizeFor != "" || convertTargetHardware != "" {
		config.Optimization = &llm.OptimizationConfig{
			OptimizeFor:    convertOptimizeFor,
			TargetHardware: convertTargetHardware,
		}
	}
	
	// Start conversion
	job, err := converter.ConvertModel(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to start conversion: %w", err)
	}
	
	fmt.Printf("🚀 Conversion started (Job ID: %s)\n", job.ID)
	fmt.Printf("📁 Output will be saved to: %s\n", job.TargetPath)
	fmt.Printf("📋 Logs available at: %s\n", job.LogPath)
	
	// Monitor conversion progress
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			status, err := converter.GetConversionStatus(job.ID)
			if err != nil {
				fmt.Printf("❌ Failed to get status: %v\n", err)
				break
			}
			
			switch status.Status {
			case llm.StatusRunning:
				fmt.Printf("\r⏳ Progress: %.1f%% | Step: %s", 
					status.Progress*100, status.CurrentStep)
			case llm.StatusCompleted:
				fmt.Printf("\n✅ Conversion completed successfully!\n")
				fmt.Printf("📁 Output saved to: %s\n", status.TargetPath)
				if status.EndTime != nil {
					duration := status.EndTime.Sub(status.StartTime)
					fmt.Printf("⏱️  Duration: %v\n", duration)
				}
				return nil
			case llm.StatusFailed:
				fmt.Printf("\n❌ Conversion failed: %s\n", status.Error)
				fmt.Printf("📋 Check logs for details: %s\n", status.LogPath)
				return fmt.Errorf("conversion failed")
			case llm.StatusCancelled:
				fmt.Println("\n🚫 Conversion was cancelled")
				return fmt.Errorf("conversion cancelled")
			}
		case <-time.After(time.Hour): // Timeout after 1 hour
			return fmt.Errorf("conversion timed out")
		}
	}
}

func runListModels(cmd *cobra.Command, args []string) error {
	manager := llm.NewModelDownloadManager(getLocalLLMBaseDir())
	models := manager.GetAvailableModels()
	
	if len(models) == 0 {
		fmt.Println("❌ No models available in registry")
		return nil
	}
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSIZE\tFORMATS\tCONTEXT\tTAGS")
	fmt.Fprintln(w, "--\t----\t----\t-------\t-------\t----")
	
	for _, model := range models {
		formats := make([]string, len(model.AvailableFormats))
		for i, f := range model.AvailableFormats {
			formats[i] = string(f)
		}
		
		tags := strings.Join(model.Tags[:min(len(model.Tags), 3)], ", ")
		if len(model.Tags) > 3 {
			tags += "..."
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			model.ID,
			truncateString(model.Name, 30),
			model.ModelSize,
			strings.Join(formats, ","),
			model.ContextSize,
			tags)
	}
	
	w.Flush()
	
	fmt.Printf("\n📊 Total models: %d\n", len(models))
	fmt.Println("💡 Use 'helix local-llm models search <query>' to find specific models")
	fmt.Println("💡 Use 'helix local-llm models download <model-id>' to download a model")
	
	return nil
}

func runSearchModels(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("search query is required")
	}
	
	query := args[0]
	manager := llm.NewModelDownloadManager(getLocalLLMBaseDir())
	results := manager.SearchModels(query)
	
	if len(results) == 0 {
		fmt.Printf("❌ No models found for query: %s\n", query)
		return nil
	}
	
	fmt.Printf("🔍 Search results for '%s' (%d models found):\n\n", query, len(results))
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSIZE\tDESCRIPTION\tTAGS")
	fmt.Fprintln(w, "--\t----\t----\t-----------\t----")
	
	for _, model := range results {
		tags := strings.Join(model.Tags[:min(len(model.Tags), 2)], ", ")
		if len(model.Tags) > 2 {
			tags += "..."
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			model.ID,
			truncateString(model.Name, 25),
			model.ModelSize,
			truncateString(model.Description, 40),
			tags)
	}
	
	w.Flush()
	
	fmt.Printf("\n💡 Use 'helix local-llm models download <model-id>' to download a model")
	
	return nil
}

// Helper functions

func getLocalLLMBaseDir() string {
	if localLLMDir != "" {
		return localLLMDir
	}
	
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".helixcode", "local-llm")
}

func detectModelFormat(path string) (llm.ModelFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".gguf":
		return llm.FormatGGUF, nil
	case ".pt", ".pth", ".safetensors":
		// Could be HF or GPTQ, check file content
		return llm.FormatHF, nil // Default to HF
	case ".bin":
		return llm.FormatGPTQ, nil
	default:
		return "", fmt.Errorf("unknown model format for extension: %s", ext)
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Cross-provider model sharing commands

// shareModelCmd represents the model sharing command
var shareModelCmd = &cobra.Command{
	Use:   "share [model-path]",
	Short: "Share a model with all compatible providers",
	Long: `Share a downloaded model with all compatible providers by creating
symlinks or copying to provider model directories.

This enables models downloaded for one provider to be used by
all other compatible providers without re-downloading.

Examples:
  helix local-llm share ./models/llama-3-8b.gguf
  helix local-llm share ./models/mistral-7b.gguf --provider vllm`,
	RunE: runShareModel,
}

// downloadAllCmd represents the download for all providers command
var downloadAllCmd = &cobra.Command{
	Use:   "download-all [model-id]",
	Short: "Download a model and make it available to all providers",
	Long: `Download a model in the most compatible format and automatically
share it with all compatible providers. This ensures the model can
be used by any provider without manual conversion.

Examples:
  helix local-llm download-all llama-3-8b-instruct
  helix local-llm download-all mistral-7b-instruct --format gguf`,
	RunE: runDownloadForAll,
}

// listSharedCmd represents the list shared models command
var listSharedCmd = &cobra.Command{
	Use:   "list-shared",
	Short: "List models shared across providers",
	Long: `List all models that are shared across multiple providers,
showing which providers have access to each model.

This helps you understand which models are available to
which providers without additional downloads.`,
	RunE: runListShared,
}

// optimizeModelCmd represents the model optimization command
var optimizeModelCmd = &cobra.Command{
	Use:   "optimize [model-path] --provider [provider]",
	Short: "Optimize a model for a specific provider",
	Long: `Optimize a model specifically for a target provider by converting
it to the optimal format and applying provider-specific optimizations.

This ensures maximum performance and compatibility for the target provider.

Examples:
  helix local-llm optimize ./model.gguf --provider vllm
  helix local-llm optimize ./model.hf --provider llamacpp`,
	RunE: runOptimizeModel,
}

// syncModelsCmd represents the model synchronization command
var syncModelsCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize models across all providers",
	Long: `Synchronize all downloaded models across providers by sharing
compatible models and converting when necessary. This ensures all
providers have access to all available models.

This command will:
- Scan all provider model directories
- Share compatible models across providers
- Convert models when needed for compatibility
- Report any incompatibilities or conversion failures`,
	RunE: runSyncModels,
}

// Command implementations for cross-provider model sharing

func runShareModel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	if len(args) == 0 {
		return fmt.Errorf("model path is required")
	}
	
	modelPath := args[0]
	
	// Check if model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file does not exist: %s", modelPath)
	}
	
	manager := llm.NewLocalLLMManager(getLocalLLMBaseDir())
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	modelName := filepath.Base(modelPath)
	fmt.Printf("🔗 Sharing model: %s\n", modelName)
	
	if shareModelProvider != "" {
		fmt.Printf("🎯 Target provider: %s\n", shareModelProvider)
	}
	
	// Share model
	if err := manager.ShareModelWithProviders(ctx, modelPath, modelName); err != nil {
		return fmt.Errorf("failed to share model: %w", err)
	}
	
	fmt.Println("✅ Model shared successfully!")
	return nil
}

func runDownloadForAll(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	if len(args) == 0 {
		return fmt.Errorf("model ID is required")
	}
	
	modelID := args[0]
	manager := llm.NewLocalLLMManager(getLocalLLMBaseDir())
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	fmt.Printf("🌐 Downloading model for all providers: %s\n", modelID)
	
	// Determine source format (default to GGUF for broad compatibility)
	sourceFormat := llm.ModelFormat(downloadFormat)
	if sourceFormat == "" {
		sourceFormat = llm.FormatGGUF
	}
	
	// Download for all providers
	if err := manager.DownloadModelForAllProviders(ctx, modelID, sourceFormat); err != nil {
		return fmt.Errorf("failed to download for all providers: %w", err)
	}
	
	fmt.Println("✅ Model downloaded and shared with all compatible providers!")
	return nil
}

func runListShared(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	manager := llm.NewLocalLLMManager(getLocalLLMBaseDir())
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	shared, err := manager.GetSharedModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to get shared models: %w", err)
	}
	
	if len(shared) == 0 {
		fmt.Println("❌ No shared models found")
		fmt.Println("💡 Use 'helix local-llm share <model-path>' to share models across providers")
		return nil
	}
	
	fmt.Printf("🔗 Shared Models (%d providers):\n\n", len(shared))
	
	for provider, models := range shared {
		fmt.Printf("📦 %s:\n", provider)
		for _, model := range models {
			fmt.Printf("  • %s\n", model)
		}
		fmt.Println()
	}
	
	return nil
}

func runOptimizeModel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	if len(args) == 0 {
		return fmt.Errorf("model path is required")
	}
	
	modelPath := args[0]
	
	// Check if model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file does not exist: %s", modelPath)
	}
	
	manager := llm.NewLocalLLMManager(getLocalLLMBaseDir())
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	modelName := filepath.Base(modelPath)
	fmt.Printf("⚡ Optimizing model: %s\n", modelName)
	fmt.Printf("🎯 Target provider: %s\n", optimizeProvider)
	
	// Optimize model
	if err := manager.OptimizeModelForProvider(ctx, modelPath, optimizeProvider); err != nil {
		return fmt.Errorf("failed to optimize model: %w", err)
	}
	
	fmt.Printf("✅ Model optimized and shared for %s!\n", optimizeProvider)
	return nil
}

func runSyncModels(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	manager := llm.NewLocalLLMManager(getLocalLLMBaseDir())
	if err := manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}
	
	fmt.Println("🔄 Synchronizing models across all providers...")
	
	// Get cross-provider registry
	registry := llm.NewCrossProviderRegistry(filepath.Join(getLocalLLMBaseDir(), "registry"))
	
	// Get all downloaded models
	downloadedModels := registry.GetDownloadedModels()
	
	if len(downloadedModels) == 0 {
		fmt.Println("❌ No downloaded models found")
		fmt.Println("💡 Use 'helix local-llm models download <model-id>' to download models first")
		return nil
	}
	
	fmt.Printf("📊 Found %d downloaded models\n", len(downloadedModels))
	
	// Process each model
	syncedCount := 0
	errorCount := 0
	
	for _, model := range downloadedModels {
		fmt.Printf("🔗 Processing: %s (%s)\n", model.ModelID, model.Format)
		
		// Check if model needs optimization for any provider
		providers := []string{"vllm", "llamacpp", "ollama", "localai", "fastchat", "textgen", "lmstudio", "jan", "koboldai", "gpt4all", "tabbyapi", "mlx", "mistralrs"}
		
		for _, provider := range providers {
			// Check compatibility
			query := llm.ModelCompatibilityQuery{
				ModelID:       model.ModelID,
				SourceFormat:  model.Format,
				TargetProvider: provider,
			}
			
			result, err := registry.CheckCompatibility(query)
			if err != nil {
				fmt.Printf("  ⚠️  %s: failed to check compatibility - %v\n", provider, err)
				errorCount++
				continue
			}
			
			if !result.IsCompatible {
				fmt.Printf("  ❌ %s: not compatible\n", provider)
				continue
			}
			
			if result.ConversionRequired {
				fmt.Printf("  🔄 %s: conversion required (%.0f min est.)\n", provider, result.EstimatedTime)
				// Perform optimization/conversion
				if err := manager.OptimizeModelForProvider(ctx, model.Path, provider); err != nil {
					fmt.Printf("    ❌ Conversion failed: %v\n", err)
					errorCount++
				} else {
					fmt.Printf("    ✅ Converted successfully\n")
					syncedCount++
				}
			} else {
				fmt.Printf("  ✅ %s: already compatible\n", provider)
				// Share directly
				if err := manager.ShareModelWithProviders(ctx, model.Path, filepath.Base(model.Path)); err != nil {
					fmt.Printf("    ❌ Failed to share: %v\n", err)
					errorCount++
				} else {
					syncedCount++
				}
			}
		}
	}
	
	fmt.Printf("\n📊 Sync completed: %d successful, %d errors\n", syncedCount, errorCount)
	if errorCount == 0 {
		fmt.Println("✅ All models synchronized successfully!")
	} else {
		fmt.Printf("⚠️  %d models failed to sync. Check logs for details.\n", errorCount)
	}
	
	return nil
}