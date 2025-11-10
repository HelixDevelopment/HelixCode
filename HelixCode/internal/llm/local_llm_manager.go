package llm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LocalLLMManager manages all local LLM providers
type LocalLLMManager struct {
	baseDir       string
	binaryDir     string
	configDir     string
	dataDir       string
	providers     map[string]*LocalLLMProvider
	httpClient    *http.Client
	isInitialized bool
}

// LocalLLMProvider represents a local LLM provider instance
type LocalLLMProvider struct {
	Name         string            `json:"name"`
	Repository   string            `json:"repository"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	DefaultPort  int               `json:"default_port"`
	BinaryPath   string            `json:"binary_path"`
	ConfigPath   string            `json:"config_path"`
	DataPath     string            `json:"data_path"`
	Status       string            `json:"status"`
	Process      *os.Process       `json:"-"`
	HealthURL    string            `json:"health_url"`
	Dependencies []string          `json:"dependencies"`
	BuildScript  string            `json:"build_script"`
	StartupCmd   []string          `json:"startup_cmd"`
	Environment  map[string]string `json:"environment"`
	LastCheck    time.Time         `json:"last_check"`
}

// Provider definitions
var providerDefinitions = map[string]*LocalLLMProvider{
	"vllm": {
		Name:         "VLLM",
		Repository:   "https://github.com/vllm-project/vllm.git",
		Version:      "main",
		Description:  "High-throughput inference engine for LLMs",
		DefaultPort:  8000,
		Dependencies: []string{"python3", "pip", "git"},
		BuildScript:  "python3 -m pip install -e .",
		StartupCmd:   []string{"python3", "-m", "vllm.entrypoints.api_server"},
		Environment: map[string]string{
			"VLLM_HOST": "127.0.0.1",
			"VLLM_PORT": "8000",
		},
	},
	"localai": {
		Name:         "LocalAI",
		Repository:   "https://github.com/mudler/LocalAI.git",
		Version:      "main",
		Description:  "Drop-in OpenAI replacement with extensive model support",
		DefaultPort:  8080,
		Dependencies: []string{"git", "make"},
		BuildScript:  "make build",
		StartupCmd:   []string{"./local-ai"},
		Environment: map[string]string{
			"WEB_UI":      "true",
			"GALLERIES":   "native",
			"MODELS_PATH": "./models",
			"ADDRESS":     "127.0.0.1:8080",
		},
	},
	"fastchat": {
		Name:         "FastChat",
		Repository:   "https://github.com/lm-sys/FastChat.git",
		Version:      "main",
		Description:  "Training and serving platform for large language models",
		DefaultPort:  7860,
		Dependencies: []string{"python3", "pip", "git"},
		BuildScript:  "pip install -e .",
		StartupCmd:   []string{"python3", "-m", "fastchat.serve.cli"},
		Environment: map[string]string{
			"HOST": "127.0.0.1",
			"PORT": "7860",
		},
	},
	"textgen": {
		Name:         "Text Generation WebUI",
		Repository:   "https://github.com/oobabooga/text-generation-webui.git",
		Version:      "main",
		Description:  "Popular Gradio-based interface with extensive features",
		DefaultPort:  5000,
		Dependencies: []string{"git", "python3", "pip"},
		BuildScript:  "pip install -r requirements.txt",
		StartupCmd:   []string{"python3", "server.py"},
		Environment: map[string]string{
			"LISTEN": "127.0.0.1:5000",
			"SHARE":  "false",
			"PUBLIC": "false",
		},
	},
	"lmstudio": {
		Name:         "LM Studio",
		Repository:   "https://github.com/lm-sys/FastChat.git", // LM Studio uses similar backend
		Version:      "main",
		Description:  "User-friendly desktop application with built-in model management",
		DefaultPort:  1234,
		Dependencies: []string{"git", "python3", "pip"},
		BuildScript:  "pip install -e .",
		StartupCmd:   []string{"python3", "-m", "fastchat.serve.cli"},
		Environment: map[string]string{
			"HOST": "127.0.0.1",
			"PORT": "1234",
		},
	},
	"jan": {
		Name:         "Jan AI",
		Repository:   "https://github.com/janhq/jan.git",
		Version:      "main",
		Description:  "Open-source local AI assistant with RAG capabilities",
		DefaultPort:  1337,
		Dependencies: []string{"git", "node", "npm"},
		BuildScript:  "npm install && npm run build",
		StartupCmd:   []string{"npm", "run", "start"},
		Environment: map[string]string{
			"PORT": "1337",
		},
	},
	"koboldai": {
		Name:         "KoboldAI",
		Repository:   "https://github.com/KoboldAI/KoboldAI-United.git",
		Version:      "main",
		Description:  "Writing-focused interface with creative assistance",
		DefaultPort:  5001,
		Dependencies: []string{"git", "python3", "pip"},
		BuildScript:  "pip install -r requirements.txt",
		StartupCmd:   []string{"python3", "server.py"},
		Environment: map[string]string{
			"HOST": "127.0.0.1",
			"PORT": "5001",
		},
	},
	"gpt4all": {
		Name:         "GPT4All",
		Repository:   "https://github.com/nomic-ai/gpt4all.git",
		Version:      "main",
		Description:  "CPU-focused inference for low-resource environments",
		DefaultPort:  4891,
		Dependencies: []string{"git", "cmake", "make"},
		BuildScript:  "mkdir -p build && cd build && cmake .. && make -j$(nproc)",
		StartupCmd:   []string{"./gpt4all-chat"},
		Environment: map[string]string{
			"HOST": "127.0.0.1",
			"PORT": "4891",
		},
	},
	"tabbyapi": {
		Name:         "TabbyAPI",
		Repository:   "https://github.com/theroyallab/tabbyAPI.git",
		Version:      "main",
		Description:  "High-performance inference server with advanced quantization",
		DefaultPort:  5000,
		Dependencies: []string{"git", "python3", "pip"},
		BuildScript:  "pip install -r requirements.txt",
		StartupCmd:   []string{"python3", "main.py"},
		Environment: map[string]string{
			"HOST": "127.0.0.1",
			"PORT": "5000",
		},
	},
	"mlx": {
		Name:         "MLX LLM",
		Repository:   "https://github.com/ml-explore/mlx-examples.git",
		Version:      "main",
		Description:  "Apple Silicon optimized inference framework",
		DefaultPort:  8080,
		Dependencies: []string{"git", "python3", "pip"},
		BuildScript:  "cd llms && pip install -e .",
		StartupCmd:   []string{"python3", "-m", "mlx_llm.serve"},
		Environment: map[string]string{
			"HOST": "127.0.0.1",
			"PORT": "8080",
		},
	},
	"mistralrs": {
		Name:         "Mistral RS",
		Repository:   "https://github.com/EricLBuehler/mistral.rs.git",
		Version:      "main",
		Description:  "High-performance Rust-based inference engine",
		DefaultPort:  8080,
		Dependencies: []string{"git", "cargo", "rustc"},
		BuildScript:  "cargo build --release",
		StartupCmd:   []string{"./target/release/mistralrs-server"},
		Environment: map[string]string{
			"HOST": "127.0.0.1",
			"PORT": "8080",
		},
	},
}

// NewLocalLLMManager creates a new local LLM manager
func NewLocalLLMManager(baseDir string) *LocalLLMManager {
	if baseDir == "" {
		homeDir, _ := os.UserHomeDir()
		baseDir = filepath.Join(homeDir, ".helixcode", "local-llm")
	}

	manager := &LocalLLMManager{
		baseDir:       baseDir,
		binaryDir:     filepath.Join(baseDir, "bin"),
		configDir:     filepath.Join(baseDir, "config"),
		dataDir:       filepath.Join(baseDir, "data"),
		providers:     make(map[string]*LocalLLMProvider),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		isInitialized: false,
	}

	return manager
}

// Initialize sets up the local LLM manager
func (m *LocalLLMManager) Initialize(ctx context.Context) error {
	if m.isInitialized {
		return nil
	}

	log.Printf("🔧 Initializing Local LLM Manager in %s", m.baseDir)

	// Create directories
	if err := m.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Check dependencies
	if err := m.checkDependencies(); err != nil {
		log.Printf("⚠️  Some dependencies missing: %v", err)
	}

	// Clone and build all providers
	for name, definition := range providerDefinitions {
		provider := &LocalLLMProvider{
			Name:         definition.Name,
			Repository:   definition.Repository,
			Version:      definition.Version,
			Description:  definition.Description,
			DefaultPort:  definition.DefaultPort,
			Dependencies: definition.Dependencies,
			BuildScript:  definition.BuildScript,
			StartupCmd:   definition.StartupCmd,
			Environment:  definition.Environment,
			Status:       "not_installed",
		}

		// Set paths
		provider.BinaryPath = filepath.Join(m.binaryDir, name)
		provider.ConfigPath = filepath.Join(m.configDir, name)
		provider.DataPath = filepath.Join(m.dataDir, name)
		provider.HealthURL = fmt.Sprintf("http://127.0.0.1:%d/health", provider.DefaultPort)

		// Create provider directories
		os.MkdirAll(provider.ConfigPath, 0755)
		os.MkdirAll(provider.DataPath, 0755)

		m.providers[name] = provider

		// Install provider
		if err := m.installProvider(ctx, provider); err != nil {
			log.Printf("⚠️  Failed to install %s: %v", name, err)
		}
	}

	m.isInitialized = true
	log.Printf("✅ Local LLM Manager initialized with %d providers", len(m.providers))

	return nil
}

// createDirectories creates necessary directories
func (m *LocalLLMManager) createDirectories() error {
	dirs := []string{m.baseDir, m.binaryDir, m.configDir, m.dataDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// checkDependencies verifies system dependencies
func (m *LocalLLMManager) checkDependencies() error {
	log.Println("🔍 Checking system dependencies...")

	// Common dependencies
	commonDeps := []string{"git", "curl", "wget"}
	missing := []string{}

	for _, dep := range commonDeps {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, dep)
		}
	}

	// Platform-specific dependencies
	switch runtime.GOOS {
	case "linux":
		linuxDeps := []string{"make", "cmake", "gcc", "g++"}
		for _, dep := range linuxDeps {
			if _, err := exec.LookPath(dep); err != nil {
				missing = append(missing, dep)
			}
		}
	case "darwin":
		darwinDeps := []string{"make", "cmake", "clang"}
		for _, dep := range darwinDeps {
			if _, err := exec.LookPath(dep); err != nil {
				missing = append(missing, dep)
			}
		}
	case "windows":
		windowsDeps := []string{"gcc.exe", "cmake.exe"}
		for _, dep := range windowsDeps {
			if _, err := exec.LookPath(dep); err != nil {
				missing = append(missing, dep)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing dependencies: %v. Please install them before continuing", missing)
	}

	log.Println("✅ All dependencies satisfied")
	return nil
}

// installProvider clones and builds a specific provider
func (m *LocalLLMManager) installProvider(ctx context.Context, provider *LocalLLMProvider) error {
	log.Printf("🔧 Installing %s (%s)...", provider.Name, provider.Version)

	providerDir := filepath.Join(m.dataDir, strings.ToLower(provider.Name))

	// Clone repository
	if err := m.cloneRepository(ctx, provider.Repository, providerDir, provider.Version); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Build provider
	if err := m.buildProvider(ctx, provider, providerDir); err != nil {
		return fmt.Errorf("failed to build provider: %w", err)
	}

	// Create startup script
	if err := m.createStartupScript(provider); err != nil {
		return fmt.Errorf("failed to create startup script: %w", err)
	}

	provider.Status = "installed"
	log.Printf("✅ Successfully installed %s", provider.Name)

	return nil
}

// cloneRepository clones a Git repository
func (m *LocalLLMManager) cloneRepository(ctx context.Context, repo, dir, version string) error {
	// Check if directory already exists
	if _, err := os.Stat(dir); err == nil {
		// Directory exists, pull latest changes
		cmd := exec.CommandContext(ctx, "git", "pull", "origin", version)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git pull failed: %s", string(output))
		}
	} else {
		// Clone fresh repository
		cmd := exec.CommandContext(ctx, "git", "clone", repo, dir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %s", string(output))
		}
	}

	// Checkout specific version
	cmd := exec.CommandContext(ctx, "git", "checkout", version)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %s", string(output))
	}

	return nil
}

// buildProvider builds the provider in its directory
func (m *LocalLLMManager) buildProvider(ctx context.Context, provider *LocalLLMProvider, dir string) error {
	log.Printf("🔨 Building %s...", provider.Name)

	// Check for provider-specific build script
	buildScript := filepath.Join(dir, "build.sh")
	if _, err := os.Stat(buildScript); err == nil {
		// Execute build script
		cmd := exec.CommandContext(ctx, "bash", "build.sh")
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build script failed: %s", string(output))
		}
		return nil
	}

	// Use generic build script
	if provider.BuildScript != "" {
		// Set environment variables
		env := os.Environ()
		for k, v := range provider.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}

		cmd := exec.CommandContext(ctx, "bash", "-c", provider.BuildScript)
		cmd.Dir = dir
		cmd.Env = env

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build failed: %s", string(output))
		}
	}

	return nil
}

// createStartupScript creates a startup script for the provider
func (m *LocalLLMManager) createStartupScript(provider *LocalLLMProvider) error {
	scriptPath := filepath.Join(m.binaryDir, strings.ToLower(provider.Name)+".sh")

	var script strings.Builder
	script.WriteString("#!/bin/bash\n")
	script.WriteString(fmt.Sprintf("# Auto-generated startup script for %s\n", provider.Name))
	script.WriteString("\n")

	// Set environment variables
	for k, v := range provider.Environment {
		script.WriteString(fmt.Sprintf("export %s=\"%s\"\n", k, v))
	}
	script.WriteString("\n")

	// Change to provider directory
	providerDir := filepath.Join(m.dataDir, strings.ToLower(provider.Name))
	script.WriteString(fmt.Sprintf("cd \"%s\"\n", providerDir))
	script.WriteString("\n")

	// Add startup command
	if len(provider.StartupCmd) > 0 {
		cmd := strings.Join(provider.StartupCmd, " ")
		script.WriteString(fmt.Sprintf("exec %s\n", cmd))
	}

	// Write script
	if err := os.WriteFile(scriptPath, []byte(script.String()), 0755); err != nil {
		return fmt.Errorf("failed to write startup script: %w", err)
	}

	return nil
}

// StartProvider starts a specific local LLM provider
func (m *LocalLLMManager) StartProvider(ctx context.Context, providerName string) error {
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	if provider.Status == "running" {
		return fmt.Errorf("provider %s is already running", providerName)
	}

	log.Printf("🚀 Starting %s...", provider.Name)

	// Start the provider process
	scriptPath := filepath.Join(m.binaryDir, strings.ToLower(providerName)+".sh")
	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = filepath.Join(m.dataDir, strings.ToLower(providerName))

	// Set environment variables
	env := os.Environ()
	for k, v := range provider.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start provider: %w", err)
	}

	provider.Process = cmd.Process
	provider.Status = "starting"
	provider.LastCheck = time.Now()

	// Wait for provider to be ready
	if err := m.waitForProvider(ctx, provider); err != nil {
		provider.Status = "failed"
		return fmt.Errorf("provider failed to start: %w", err)
	}

	provider.Status = "running"
	log.Printf("✅ Successfully started %s on port %d", provider.Name, provider.DefaultPort)

	return nil
}

// StopProvider stops a specific local LLM provider
func (m *LocalLLMManager) StopProvider(ctx context.Context, providerName string) error {
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	if provider.Status != "running" {
		return fmt.Errorf("provider %s is not running", providerName)
	}

	log.Printf("🛑 Stopping %s...", provider.Name)

	if provider.Process != nil {
		// Try graceful shutdown first
		if err := provider.Process.Signal(os.Interrupt); err != nil {
			// Force kill if graceful fails
			provider.Process.Kill()
		}

		// Wait for process to exit
		if _, err := provider.Process.Wait(); err != nil {
			log.Printf("⚠️  Error waiting for process to exit: %v", err)
		}

		provider.Process = nil
	}

	provider.Status = "stopped"
	log.Printf("✅ Successfully stopped %s", provider.Name)

	return nil
}

// waitForProvider waits for a provider to become healthy
func (m *LocalLLMManager) waitForProvider(ctx context.Context, provider *LocalLLMProvider) error {
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for provider to become healthy")
		case <-ticker.C:
			if m.isProviderHealthy(ctx, provider) {
				return nil
			}
		}
	}
}

// isProviderHealthy checks if a provider is healthy
func (m *LocalLLMManager) isProviderHealthy(ctx context.Context, provider *LocalLLMProvider) bool {
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", provider.DefaultPort)

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return false
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetProviderStatus returns the status of all providers
func (m *LocalLLMManager) GetProviderStatus(ctx context.Context) map[string]*LocalLLMProvider {
	for _, provider := range m.providers {
		if provider.Status == "running" {
			if m.isProviderHealthy(ctx, provider) {
				provider.Status = "running"
			} else {
				provider.Status = "unhealthy"
			}
		}
		provider.LastCheck = time.Now()
	}
	return m.providers
}

// GetRunningProviders returns a list of running provider endpoints
func (m *LocalLLMManager) GetRunningProviders(ctx context.Context) []string {
	var running []string
	status := m.GetProviderStatus(ctx)

	for _, provider := range status {
		if provider.Status == "running" {
			endpoint := fmt.Sprintf("http://127.0.0.1:%d", provider.DefaultPort)
			running = append(running, endpoint)
		}
	}

	return running
}

// StartAllProviders starts all available providers
func (m *LocalLLMManager) StartAllProviders(ctx context.Context) error {
	log.Println("🚀 Starting all local LLM providers...")

	for name := range m.providers {
		if err := m.StartProvider(ctx, name); err != nil {
			log.Printf("⚠️  Failed to start %s: %v", name, err)
		}
	}

	log.Println("✅ Started available providers")
	return nil
}

// StopAllProviders stops all running providers
func (m *LocalLLMManager) StopAllProviders(ctx context.Context) error {
	log.Println("🛑 Stopping all local LLM providers...")

	for name := range m.providers {
		if err := m.StopProvider(ctx, name); err != nil {
			log.Printf("⚠️  Failed to stop %s: %v", name, err)
		}
	}

	log.Println("✅ Stopped all providers")
	return nil
}

// Cleanup cleans up all provider resources
func (m *LocalLLMManager) Cleanup(ctx context.Context) error {
	log.Println("🧹 Cleaning up local LLM providers...")

	// Stop all providers first
	m.StopAllProviders(ctx)

	// Optionally remove data directories
	// (Commented out to preserve downloaded models and configs)
	// os.RemoveAll(m.dataDir)

	log.Println("✅ Cleanup completed")
	return nil
}

// UpdateProvider updates a specific provider to the latest version
func (m *LocalLLMManager) UpdateProvider(ctx context.Context, providerName string) error {
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	log.Printf("🔄 Updating %s...", provider.Name)

	// Stop provider if running
	if provider.Status == "running" {
		if err := m.StopProvider(ctx, providerName); err != nil {
			log.Printf("⚠️  Failed to stop provider for update: %v", err)
		}
	}

	// Pull latest changes
	providerDir := filepath.Join(m.dataDir, strings.ToLower(provider.Name))
	cmd := exec.CommandContext(ctx, "git", "pull", "origin", provider.Version)
	cmd.Dir = providerDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %s", string(output))
	}

	// Rebuild provider
	if err := m.buildProvider(ctx, provider, providerDir); err != nil {
		return fmt.Errorf("failed to rebuild provider: %w", err)
	}

	// Update startup script
	if err := m.createStartupScript(provider); err != nil {
		return fmt.Errorf("failed to update startup script: %w", err)
	}

	log.Printf("✅ Successfully updated %s", provider.Name)
	return nil
}

// ShareModelWithProviders shares a downloaded model with all compatible providers
func (m *LocalLLMManager) ShareModelWithProviders(ctx context.Context, modelPath string, modelName string) error {
	log.Printf("🔗 Sharing model %s with compatible providers...", modelName)

	// Detect model format
	format, err := m.detectModelFormat(modelPath)
	if err != nil {
		return fmt.Errorf("failed to detect model format: %w", err)
	}

	// Find compatible providers
	compatibleProviders := []string{}
	for name := range m.providers {
		if m.isFormatCompatibleWithProvider(format, name) {
			compatibleProviders = append(compatibleProviders, name)
		}
	}

	if len(compatibleProviders) == 0 {
		return fmt.Errorf("no providers found compatible with format %s", format)
	}

	// Create symlinks or copies for each compatible provider
	for _, providerName := range compatibleProviders {
		provider := m.providers[providerName]
		targetDir := filepath.Join(provider.DataPath, "models")
		os.MkdirAll(targetDir, 0755)

		targetPath := filepath.Join(targetDir, filepath.Base(modelPath))

		// Remove existing target if it exists
		os.Remove(targetPath)

		// Create symlink (or copy if symlink fails)
		err := os.Symlink(modelPath, targetPath)
		if err != nil {
			// Fallback to copy
			log.Printf("⚠️  Symlink failed for %s, copying instead: %v", providerName, err)
			if err := m.copyModel(modelPath, targetPath); err != nil {
				log.Printf("❌ Failed to copy model for %s: %v", providerName, err)
				continue
			}
		} else {
			log.Printf("✅ Linked model for %s", providerName)
		}
	}

	log.Printf("✅ Model shared with %d providers", len(compatibleProviders))
	return nil
}

// DownloadModelForAllProviders downloads a model and makes it available to all compatible providers
func (m *LocalLLMManager) DownloadModelForAllProviders(ctx context.Context, modelID string, sourceFormat ModelFormat) error {
	log.Printf("🌐 Downloading model %s for all providers...", modelID)

	// Initialize download manager
	downloadManager := NewModelDownloadManager(m.baseDir)

	// Get model info
	_, err := downloadManager.GetModelByID(modelID)
	if err != nil {
		return fmt.Errorf("model not found: %w", err)
	}

	// Find best format to download (most compatible)
	bestFormat := m.findMostCompatibleFormat(sourceFormat)

	// Download model in best format
	req := ModelDownloadRequest{
		ModelID:        modelID,
		Format:         bestFormat,
		TargetProvider: "", // Download to shared location
		ForceDownload:  false,
	}

	progressChan, err := downloadManager.DownloadModel(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}

	// Monitor download
	for progress := range progressChan {
		if progress.Error != "" {
			return fmt.Errorf("download failed: %s", progress.Error)
		}
		if progress.Progress == 1.0 {
			log.Printf("✅ Download completed for model %s", modelID)
			break
		}
	}

	// Get the downloaded model path
	downloadedPath := filepath.Join(m.baseDir, "shared", modelID, fmt.Sprintf("model.%s", bestFormat))

	// Share with all compatible providers
	return m.ShareModelWithProviders(ctx, downloadedPath, modelID)
}

// GetSharedModels returns list of models shared across providers
func (m *LocalLLMManager) GetSharedModels(ctx context.Context) (map[string][]string, error) {
	shared := make(map[string][]string)

	for name, provider := range m.providers {
		modelsDir := filepath.Join(provider.DataPath, "models")
		if _, err := os.Stat(modelsDir); err == nil {
			entries, err := os.ReadDir(modelsDir)
			if err != nil {
				continue
			}

			var models []string
			for _, entry := range entries {
				if !entry.IsDir() {
					models = append(models, entry.Name())
				}
			}

			if len(models) > 0 {
				shared[name] = models
			}
		}
	}

	return shared, nil
}

// OptimizeModelForProvider optimizes a model specifically for a provider
func (m *LocalLLMManager) OptimizeModelForProvider(ctx context.Context, modelPath string, targetProvider string) error {
	provider, exists := m.providers[targetProvider]
	if !exists {
		return fmt.Errorf("provider %s not found", targetProvider)
	}

	log.Printf("⚡ Optimizing model for %s...", provider.Name)

	// Detect current format
	currentFormat, err := m.detectModelFormat(modelPath)
	if err != nil {
		return fmt.Errorf("failed to detect current format: %w", err)
	}

	// Get optimal format for provider
	optimalFormat := m.getOptimalFormatForProvider(targetProvider)

	// If already in optimal format, just share it
	if currentFormat == optimalFormat {
		return m.ShareModelWithProviders(ctx, modelPath, filepath.Base(modelPath))
	}

	// Convert model
	converter := NewModelConverter(m.baseDir)
	config := ConversionConfig{
		SourcePath:   modelPath,
		SourceFormat: currentFormat,
		TargetFormat: optimalFormat,
		Optimization: &OptimizationConfig{
			OptimizeFor:    m.getOptimizationTarget(targetProvider),
			TargetHardware: m.getTargetHardware(targetProvider),
		},
		Timeout: 60,
	}

	job, err := converter.ConvertModel(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to start conversion: %w", err)
	}

	// Wait for conversion completion
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := converter.GetConversionStatus(job.ID)
			if err != nil {
				return fmt.Errorf("failed to get conversion status: %w", err)
			}

			switch status.Status {
			case StatusCompleted:
				log.Printf("✅ Model optimized for %s", provider.Name)
				return m.ShareModelWithProviders(ctx, status.TargetPath, filepath.Base(status.TargetPath))
			case StatusFailed:
				return fmt.Errorf("conversion failed: %s", status.Error)
			case StatusCancelled:
				return fmt.Errorf("conversion cancelled")
			}
		}
	}
}

// Helper methods for cross-provider functionality

func (m *LocalLLMManager) detectModelFormat(modelPath string) (ModelFormat, error) {
	ext := strings.ToLower(filepath.Ext(modelPath))
	switch ext {
	case ".gguf":
		return FormatGGUF, nil
	case ".pt", ".pth", ".safetensors":
		return FormatHF, nil
	case ".bin":
		return FormatGPTQ, nil
	default:
		return "", fmt.Errorf("unknown model format for extension: %s", ext)
	}
}

func (m *LocalLLMManager) isFormatCompatibleWithProvider(format ModelFormat, providerName string) bool {
	// Get supported formats for provider
	var supportedFormats []ModelFormat
	switch providerName {
	case "vllm":
		supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF, FormatFP16, FormatBF16}
	case "llamacpp":
		supportedFormats = []ModelFormat{FormatGGUF}
	case "ollama":
		supportedFormats = []ModelFormat{FormatGGUF}
	case "localai":
		supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF}
	case "fastchat":
		supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF}
	case "textgen":
		supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF}
	case "lmstudio":
		supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF}
	case "jan":
		supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF}
	case "koboldai":
		supportedFormats = []ModelFormat{FormatGGUF}
	case "gpt4all":
		supportedFormats = []ModelFormat{FormatGGUF}
	case "tabbyapi":
		supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF}
	case "mlx":
		supportedFormats = []ModelFormat{FormatGGUF, FormatHF}
	case "mistralrs":
		supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF, FormatBF16, FormatFP16}
	default:
		supportedFormats = []ModelFormat{FormatGGUF} // Most universal format
	}

	for _, supportedFormat := range supportedFormats {
		if supportedFormat == format {
			return true
		}
	}
	return false
}

func (m *LocalLLMManager) findMostCompatibleFormat(sourceFormat ModelFormat) ModelFormat {
	// Count how many providers support each format
	formatCounts := make(map[ModelFormat]int)

	for _, provider := range m.providers {
		var supportedFormats []ModelFormat
		switch provider.Name {
		case "VLLM":
			supportedFormats = []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF, FormatFP16, FormatBF16}
		case "Llama.cpp":
			supportedFormats = []ModelFormat{FormatGGUF}
		case "Ollama":
			supportedFormats = []ModelFormat{FormatGGUF}
		default:
			supportedFormats = []ModelFormat{FormatGGUF}
		}

		for _, format := range supportedFormats {
			formatCounts[format]++
		}
	}

	// Return format with highest compatibility
	maxCount := 0
	bestFormat := FormatGGUF // Default
	for format, count := range formatCounts {
		if count > maxCount {
			maxCount = count
			bestFormat = format
		}
	}

	return bestFormat
}

func (m *LocalLLMManager) getOptimalFormatForProvider(providerName string) ModelFormat {
	switch providerName {
	case "llamacpp":
		return FormatGGUF
	case "vllm":
		return FormatGGUF // Best performance/speed balance
	case "ollama":
		return FormatGGUF
	case "localai":
		return FormatGGUF
	case "mistralrs":
		return FormatGGUF
	default:
		return FormatGGUF
	}
}

func (m *LocalLLMManager) getOptimizationTarget(providerName string) string {
	switch providerName {
	case "vllm":
		return "gpu"
	case "llamacpp":
		return "cpu" // Can be GPU too, but CPU is more universal
	case "mlx":
		return "gpu" // Apple Silicon GPU
	case "mistralrs":
		return "gpu"
	default:
		return "cpu" // Most universal
	}
}

func (m *LocalLLMManager) getTargetHardware(providerName string) string {
	switch providerName {
	case "vllm":
		return "nvidia"
	case "mlx":
		return "apple"
	case "mistralrs":
		return "nvidia"
	default:
		return "cpu"
	}
}

func (m *LocalLLMManager) copyModel(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
