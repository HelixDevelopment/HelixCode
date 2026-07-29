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
	"sync"
	"time"

	"dev.helix.code/internal/providers/httpclient"
)

// LocalLLMManager manages all local LLM providers
type LocalLLMManager struct {
	// mu guards the providers map itself AND the mutable fields of the
	// *LocalLLMProvider records it holds (Status, LastCheck, Process), plus
	// isInitialized and skipProviderInstall — i.e. every mutable field on this
	// type. The provider fields that are assigned only during Initialize,
	// BEFORE the record is published into the map (Name, DefaultPort, DataPath,
	// ConfigPath, HealthURL, Environment, ...) are safe to read unguarded: the
	// mutex handoff in registerProvider/lookupProvider supplies the
	// happens-before edge.
	//
	// LOCK DISCIPLINE (HXC-203) — two rules, both load-bearing:
	//
	//  1. NEVER hold mu across blocking work: no network probe, no exec, no
	//     process Wait, no filesystem walk. A hung provider must not be able to
	//     block every other reader of the status map. Blocking work is always
	//     performed on values snapshotted out from under the lock.
	//  2. NEVER call another locking method while holding mu. Go mutexes are
	//     not reentrant, so a helper that locks must not be invoked from a
	//     critical section. The small accessors below each take and release
	//     mu themselves, which is what keeps that rule mechanical.
	mu                  sync.RWMutex
	baseDir             string
	binaryDir           string
	configDir           string
	dataDir             string
	providers           map[string]*LocalLLMProvider
	httpClient          *http.Client
	isInitialized       bool
	skipProviderInstall bool // Skip provider installation (for testing)
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
		baseDir:   baseDir,
		binaryDir: filepath.Join(baseDir, "bin"),
		configDir: filepath.Join(baseDir, "config"),
		dataDir:   filepath.Join(baseDir, "data"),
		providers: make(map[string]*LocalLLMProvider),
		// Shared tuned HTTP/2 transport (speed programme P1-T01,
		// R1 B03 / R3 §4.7) — connection pooling only; request
		// behaviour is unchanged.
		httpClient:    httpclient.NewHTTPClient(10 * time.Second),
		isInitialized: false,
	}

	return manager
}

// --- Shared-state accessors (HXC-203) ------------------------------------
//
// Every read or write of the providers map, of a provider's mutable fields,
// or of isInitialized goes through one of these. Each takes and releases mu
// itself and performs no blocking work, so callers can never accidentally
// hold the lock across a network probe or an exec, and can never deadlock by
// nesting two of them.

// registerProvider inserts a provider record under the write lock.
func (m *LocalLLMManager) registerProvider(name string, provider *LocalLLMProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = provider
}

// lookupProvider returns the live provider record for name.
func (m *LocalLLMManager) lookupProvider(name string) (*LocalLLMProvider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	provider, exists := m.providers[name]
	return provider, exists
}

// providerNames returns a snapshot of the registered provider names. Callers
// iterate this slice rather than the live map, so a concurrent registration
// cannot trigger Go's concurrent-map-iteration fatal error and the lock is not
// held while the loop body does real work.
func (m *LocalLLMManager) providerNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// providerCount returns the number of registered providers.
func (m *LocalLLMManager) providerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.providers)
}

// statusOf reads a provider's Status under the read lock.
func (m *LocalLLMManager) statusOf(provider *LocalLLMProvider) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return provider.Status
}

// setStatus writes a provider's Status under the write lock.
func (m *LocalLLMManager) setStatus(provider *LocalLLMProvider, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	provider.Status = status
}

// setStarted records the process handle, the "starting" status and the check
// timestamp in one critical section, so no reader can observe a provider that
// has a live process but no status (or the reverse).
func (m *LocalLLMManager) setStarted(provider *LocalLLMProvider, process *os.Process) {
	m.mu.Lock()
	defer m.mu.Unlock()
	provider.Process = process
	provider.Status = "starting"
	provider.LastCheck = time.Now()
}

// takeProcess clears and returns a provider's process handle. The caller then
// signals/kills/waits on the returned handle with no lock held, since those
// operations block for as long as the child takes to exit.
func (m *LocalLLMManager) takeProcess(provider *LocalLLMProvider) *os.Process {
	m.mu.Lock()
	defer m.mu.Unlock()
	process := provider.Process
	provider.Process = nil
	return process
}

// Initialize sets up the local LLM manager
func (m *LocalLLMManager) Initialize(ctx context.Context) error {
	m.mu.RLock()
	initialized := m.isInitialized
	m.mu.RUnlock()
	if initialized {
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

		m.registerProvider(name, provider)

		// Install provider (skip if in test mode). Deliberately performed with
		// no lock held — installProvider clones a repository and runs a build.
		if !m.shouldSkipProviderInstall() {
			if err := m.installProvider(ctx, provider); err != nil {
				log.Printf("⚠️  Failed to install %s: %v", name, err)
			}
		}
	}

	m.mu.Lock()
	m.isInitialized = true
	m.mu.Unlock()

	log.Printf("✅ Local LLM Manager initialized with %d providers", m.providerCount())

	return nil
}

// GetBaseDir returns the base directory for the local LLM manager
func (m *LocalLLMManager) GetBaseDir() string {
	return m.baseDir
}

// SetSkipProviderInstall sets whether to skip provider installation (for testing)
func (m *LocalLLMManager) SetSkipProviderInstall(skip bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skipProviderInstall = skip
}

// shouldSkipProviderInstall reads the test-only install-skip flag under the
// read lock. Guarded for the same reason as every other mutable field on this
// type: Initialize reads it while a caller could still be setting it.
func (m *LocalLLMManager) shouldSkipProviderInstall() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.skipProviderInstall
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

	m.setStatus(provider, "installed")
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

		cmd := exec.CommandContext(ctx, "bash", "-c", provider.BuildScript) // nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command -- BuildScript sourced only from static providerDefinitions map literals; never deserialized/config-loaded (see docs/research/semgrep_exec_triage_20260622)
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
	provider, exists := m.lookupProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	if m.statusOf(provider) == "running" {
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

	m.setStarted(provider, cmd.Process)

	// Wait for provider to be ready. No lock held: this polls the provider's
	// health endpoint for up to 60s.
	if err := m.waitForProvider(ctx, provider); err != nil {
		m.setStatus(provider, "failed")
		return fmt.Errorf("provider failed to start: %w", err)
	}

	m.setStatus(provider, "running")
	log.Printf("✅ Successfully started %s on port %d", provider.Name, provider.DefaultPort)

	return nil
}

// StopProvider stops a specific local LLM provider
func (m *LocalLLMManager) StopProvider(ctx context.Context, providerName string) error {
	provider, exists := m.lookupProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	if m.statusOf(provider) != "running" {
		return fmt.Errorf("provider %s is not running", providerName)
	}

	log.Printf("🛑 Stopping %s...", provider.Name)

	// Claim the process handle under the lock and clear it in the same critical
	// section. Two concurrent stops therefore cannot both signal and Wait on
	// the same child (a double Wait reaps an already-reaped process and
	// errors); the loser simply sees nil and falls through to the status write.
	if process := m.takeProcess(provider); process != nil {
		// No lock held from here: Wait blocks until the child exits.
		// Try graceful shutdown first
		if err := process.Signal(os.Interrupt); err != nil {
			// Force kill if graceful fails
			process.Kill()
		}

		// Wait for process to exit
		if _, err := process.Wait(); err != nil {
			log.Printf("⚠️  Error waiting for process to exit: %v", err)
		}
	}

	m.setStatus(provider, "stopped")
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
	// DefaultPort is immutable after Initialize, so reading it here needs no
	// lock. The probe itself is delegated so the status refresh can call it
	// with a plain port value and touch no shared memory at all.
	return m.isEndpointHealthy(ctx, provider.DefaultPort)
}

// isEndpointHealthy probes a provider's health endpoint by port. It takes the
// port BY VALUE and reads no manager state, so it is safe — and required — to
// call it with no lock held: it performs network I/O that can hang for the
// duration of the HTTP client timeout.
func (m *LocalLLMManager) isEndpointHealthy(ctx context.Context, port int) bool {
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)

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

// refreshProviderHealth re-probes every provider that claims to be running and
// records the verdict. This is the MUTATING half of what used to live inside
// GetProviderStatus (HXC-203): a method named as a query wrote to shared state
// with no synchronisation, so two concurrent callers raced on Status and
// LastCheck.
//
// It runs in three phases so that the network probe never happens under the
// lock — holding mu across a hung provider's health check would stall every
// other reader of the status map for the full HTTP timeout.
func (m *LocalLLMManager) refreshProviderHealth(ctx context.Context) {
	type probeTarget struct {
		name string
		port int
	}

	// Phase 1 — snapshot the probe targets under the read lock. The port is
	// copied by value so phase 2 dereferences nothing shared.
	m.mu.RLock()
	targets := make([]probeTarget, 0, len(m.providers))
	for name, provider := range m.providers {
		if provider.Status == "running" {
			targets = append(targets, probeTarget{name: name, port: provider.DefaultPort})
		}
	}
	m.mu.RUnlock()

	// Phase 2 — probe with NO lock held. This is the blocking part.
	verdicts := make(map[string]bool, len(targets))
	for _, target := range targets {
		verdicts[target.name] = m.isEndpointHealthy(ctx, target.port)
	}

	// Phase 3 — apply the verdicts under the write lock.
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, healthy := range verdicts {
		provider, exists := m.providers[name]
		if !exists {
			continue
		}
		// Compare-and-set. The lock was released for the probe, so
		// StartProvider or StopProvider may have moved this provider on since
		// the verdict was formed. Only a provider that is STILL claiming
		// "running" may be judged by this probe — otherwise a stale verdict
		// would clobber newer, authoritative state (demoting a provider that
		// has since been deliberately stopped, or resurrecting one as
		// "unhealthy" after it was stopped cleanly).
		if provider.Status != "running" {
			continue
		}
		if healthy {
			provider.Status = "running"
		} else {
			provider.Status = "unhealthy"
		}
	}

	// LastCheck is stamped on every provider, not just the probed ones, which
	// preserves the pre-fix contract: `helixcode local-llm status` renders a
	// LAST CHECK column for every row.
	for _, provider := range m.providers {
		provider.LastCheck = now
	}
}

// snapshotProviders returns a deep copy of the provider records.
//
// The pre-fix GetProviderStatus returned m.providers — the LIVE internal map —
// so every caller received pointers into manager state that it could mutate,
// and any range over the result raced with a concurrent registration. Callers
// now get records they own outright.
func (m *LocalLLMManager) snapshotProviders() map[string]*LocalLLMProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := make(map[string]*LocalLLMProvider, len(m.providers))
	for name, provider := range m.providers {
		clone := *provider

		// Never hand out the live OS process handle: a caller holding it could
		// signal or kill a provider this manager owns. No caller reads it.
		clone.Process = nil

		// Copy the reference-typed fields too, so a caller mutating the
		// returned record cannot reach back into manager state through them.
		if provider.Environment != nil {
			environment := make(map[string]string, len(provider.Environment))
			for key, value := range provider.Environment {
				environment[key] = value
			}
			clone.Environment = environment
		}
		clone.Dependencies = append([]string(nil), provider.Dependencies...)
		clone.StartupCmd = append([]string(nil), provider.StartupCmd...)

		snapshot[name] = &clone
	}

	return snapshot
}

// GetProviderStatus refreshes provider health and returns a snapshot of all
// providers. The returned map and the records in it are private copies owned
// by the caller; mutating them does not affect manager state.
func (m *LocalLLMManager) GetProviderStatus(ctx context.Context) map[string]*LocalLLMProvider {
	m.refreshProviderHealth(ctx)
	return m.snapshotProviders()
}

// GetRunningProviders returns a list of running provider endpoints
func (m *LocalLLMManager) GetRunningProviders(ctx context.Context) []string {
	running := make([]string, 0)
	// GetProviderStatus hands back a private snapshot, so reading Status off
	// these records needs no further synchronisation.
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

	// Iterate a name snapshot, never the live map: StartProvider locks
	// internally, so ranging m.providers under the lock would deadlock, and
	// ranging it unlocked would race with a concurrent registration.
	for _, name := range m.providerNames() {
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

	// Name snapshot, same reasoning as StartAllProviders.
	for _, name := range m.providerNames() {
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
	provider, exists := m.lookupProvider(providerName)
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	log.Printf("🔄 Updating %s...", provider.Name)

	// Stop provider if running
	if m.statusOf(provider) == "running" {
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
	for _, name := range m.providerNames() {
		if m.isFormatCompatibleWithProvider(format, name) {
			compatibleProviders = append(compatibleProviders, name)
		}
	}

	if len(compatibleProviders) == 0 {
		return fmt.Errorf("no providers found compatible with format %s", format)
	}

	// Create symlinks or copies for each compatible provider
	for _, providerName := range compatibleProviders {
		provider, exists := m.lookupProvider(providerName)
		if !exists {
			// Deregistered between the name snapshot and here.
			continue
		}
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

	// Snapshot (name, DataPath) under the lock, then walk the filesystem with
	// no lock held — a slow or stalled mount must not block status readers.
	type providerDir struct {
		name     string
		dataPath string
	}
	m.mu.RLock()
	providerDirs := make([]providerDir, 0, len(m.providers))
	for name, provider := range m.providers {
		providerDirs = append(providerDirs, providerDir{name: name, dataPath: provider.DataPath})
	}
	m.mu.RUnlock()

	for _, entry := range providerDirs {
		name := entry.name
		modelsDir := filepath.Join(entry.dataPath, "models")
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
	provider, exists := m.lookupProvider(targetProvider)
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

	// Snapshot the display names under the lock rather than ranging the live
	// map, which would race with a concurrent registration.
	m.mu.RLock()
	displayNames := make([]string, 0, len(m.providers))
	for _, provider := range m.providers {
		displayNames = append(displayNames, provider.Name)
	}
	m.mu.RUnlock()

	for _, displayName := range displayNames {
		var supportedFormats []ModelFormat
		switch displayName {
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
