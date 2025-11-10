package cognee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/hardware"
	"dev.helix.code/internal/logging"

	"github.com/google/uuid"
)

// CogneeManager manages Cognee.ai integration with HelixCode
type CogneeManager struct {
	// Configuration
	config    *CogneeConfig
	hwProfile *hardware.Profile
	logger    logging.Logger

	// Cognee process
	cogneeProcess *os.Process
	cogneeDir     string
	configPath    string

	// API client
	apiClient *http.Client
	baseURL   string
	apiKey    string

	// State
	initialized  bool
	running      bool
	enabled      bool
	healthStatus *HealthStatus

	// Integrations
	providerIntegrations map[string]*ProviderIntegration
	modelIntegrations    map[string]*ModelIntegration

	// Performance
	metrics   *CogneeMetrics
	cache     *CacheManager
	optimizer *PerformanceOptimizer

	// Synchronization
	mu       sync.RWMutex
	bgTasks  sync.WaitGroup
	stopChan chan struct{}
}

// CogneeConfig contains Cognee configuration
type CogneeConfig struct {
	Enabled       bool               `json:"enabled" yaml:"enabled"`
	AutoStart     bool               `json:"auto_start" yaml:"auto_start"`
	Host          string             `json:"host" yaml:"host"`
	Port          int                `json:"port" yaml:"port"`
	APIKey        string             `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	DynamicConfig bool               `json:"dynamic_config" yaml:"dynamic_config"`
	Source        string             `json:"source,omitempty" yaml:"source,omitempty"`
	Branch        string             `json:"branch,omitempty" yaml:"branch,omitempty"`
	BuildPath     string             `json:"build_path,omitempty" yaml:"build_path,omitempty"`
	Optimization  OptimizationConfig `json:"optimization" yaml:"optimization"`
	Features      FeatureConfig      `json:"features" yaml:"features"`
	Providers     ProviderConfigMap  `json:"providers" yaml:"providers"`
	API           APIConfig          `json:"api" yaml:"api"`
	Performance   PerformanceConfig  `json:"performance" yaml:"performance"`
	Cache         CacheConfig        `json:"cache" yaml:"cache"`
	Monitoring    MonitoringConfig   `json:"monitoring" yaml:"monitoring"`
}

// OptimizationConfig contains optimization settings
type OptimizationConfig struct {
	HostAware          bool                   `json:"host_aware" yaml:"host_aware"`
	CPUOptimization    bool                   `json:"cpu_optimization" yaml:"cpu_optimization"`
	GPUOptimization    bool                   `json:"gpu_optimization" yaml:"gpu_optimization"`
	MemoryOptimization bool                   `json:"memory_optimization" yaml:"memory_optimization"`
	HostSpecific       map[string]interface{} `json:"host_specific,omitempty" yaml:"host_specific,omitempty"`
}

// FeatureConfig contains feature settings
type FeatureConfig struct {
	KnowledgeGraph     bool `json:"knowledge_graph" yaml:"knowledge_graph"`
	SemanticSearch     bool `json:"semantic_search" yaml:"semantic_search"`
	RealTimeProcessing bool `json:"real_time_processing" yaml:"real_time_processing"`
	MultiModalSupport  bool `json:"multi_modal_support" yaml:"multi_modal_support"`
	GraphAnalytics     bool `json:"graph_analytics" yaml:"graph_analytics"`
	AdvancedInsights   bool `json:"advanced_insights" yaml:"advanced_insights"`
	AutoOptimization   bool `json:"auto_optimization" yaml:"auto_optimization"`
}

// ProviderConfig contains provider-specific Cognee settings
type ProviderConfig struct {
	Enabled      bool                   `json:"enabled" yaml:"enabled"`
	Integration  string                 `json:"integration" yaml:"integration"`
	Priority     int                    `json:"priority,omitempty" yaml:"priority,omitempty"`
	Features     []string               `json:"features,omitempty" yaml:"features,omitempty"`
	Optimization map[string]interface{} `json:"optimization,omitempty" yaml:"optimization,omitempty"`
}

// ProviderConfigMap is a map of provider configurations
type ProviderConfigMap map[string]ProviderConfig

// APIConfig contains API configuration
type APIConfig struct {
	Enabled        bool          `json:"enabled" yaml:"enabled"`
	Host           string        `json:"host" yaml:"host"`
	Port           int           `json:"port" yaml:"port"`
	AuthRequired   bool          `json:"auth_required" yaml:"auth_required"`
	RateLimit      int           `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
	CORS           bool          `json:"cors" yaml:"cors"`
	DocsEnabled    bool          `json:"docs_enabled" yaml:"docs_enabled"`
	Timeout        time.Duration `json:"timeout" yaml:"timeout"`
	MaxRequestSize int64         `json:"max_request_size,omitempty" yaml:"max_request_size,omitempty"`
}

// PerformanceConfig contains performance settings
type PerformanceConfig struct {
	Workers           int           `json:"workers" yaml:"workers"`
	QueueSize         int           `json:"queue_size" yaml:"queue_size"`
	BatchSize         int           `json:"batch_size" yaml:"batch_size"`
	FlushInterval     time.Duration `json:"flush_interval" yaml:"flush_interval"`
	MaxMemory         int64         `json:"max_memory,omitempty" yaml:"max_memory,omitempty"`
	CacheSize         int64         `json:"cache_size,omitempty" yaml:"cache_size,omitempty"`
	OptimizationLevel string        `json:"optimization_level" yaml:"optimization_level"`
}

// CacheConfig contains cache configuration
type CacheConfig struct {
	Enabled     bool          `json:"enabled" yaml:"enabled"`
	Type        string        `json:"type" yaml:"type"`
	Host        string        `json:"host,omitempty" yaml:"host,omitempty"`
	Port        int           `json:"port,omitempty" yaml:"port,omitempty"`
	Database    int           `json:"database,omitempty" yaml:"database,omitempty"`
	TTL         time.Duration `json:"ttl" yaml:"ttl"`
	MaxSize     int64         `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	Compression bool          `json:"compression" yaml:"compression"`
}

// MonitoringConfig contains monitoring settings
type MonitoringConfig struct {
	Enabled      bool          `json:"enabled" yaml:"enabled"`
	MetricsPort  int           `json:"metrics_port" yaml:"metrics_port"`
	HealthCheck  time.Duration `json:"health_check" yaml:"health_check"`
	LogLevel     string        `json:"log_level" yaml:"log_level"`
	TraceEnabled bool          `json:"trace_enabled" yaml:"trace_enabled"`
	AlertWebhook string        `json:"alert_webhook,omitempty" yaml:"alert_webhook,omitempty"`
}

// HealthStatus contains Cognee health information
type HealthStatus struct {
	Status    string                 `json:"status"`
	Uptime    time.Duration          `json:"uptime"`
	Version   string                 `json:"version"`
	Memory    MemoryStatus           `json:"memory"`
	CPU       CPUStatus              `json:"cpu"`
	GPU       []GPUStatus            `json:"gpu"`
	Services  map[string]string      `json:"services"`
	LastCheck time.Time              `json:"last_check"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// MemoryStatus contains memory information
type MemoryStatus struct {
	Used      int64   `json:"used"`
	Total     int64   `json:"total"`
	Available int64   `json:"available"`
	Percent   float64 `json:"percent"`
	Processes int     `json:"processes"`
}

// CPUStatus contains CPU information
type CPUStatus struct {
	Usage       float64   `json:"usage"`
	Cores       int       `json:"cores"`
	Frequency   float64   `json:"frequency"`
	Temperature float64   `json:"temperature,omitempty"`
	Load        []float64 `json:"load"`
}

// GPUStatus contains GPU information
type GPUStatus struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	MemoryUsed  int64   `json:"memory_used"`
	MemoryTotal int64   `json:"memory_total"`
	Usage       float64 `json:"usage"`
	Temperature float64 `json:"temperature,omitempty"`
	PowerUsage  float64 `json:"power_usage,omitempty"`
}

// ProviderIntegration tracks provider integration with Cognee
type ProviderIntegration struct {
	Provider     string             `json:"provider"`
	Connected    bool               `json:"connected"`
	Config       ProviderConfig     `json:"config"`
	Features     []string           `json:"features"`
	Metrics      IntegrationMetrics `json:"metrics"`
	Status       string             `json:"status"`
	LastError    string             `json:"last_error,omitempty"`
	ConnectedAt  time.Time          `json:"connected_at"`
	LastActivity time.Time          `json:"last_activity"`
}

// ModelIntegration tracks model integration with Cognee
type ModelIntegration struct {
	Provider     string                 `json:"provider"`
	Model        string                 `json:"model"`
	Connected    bool                   `json:"connected"`
	Config       map[string]interface{} `json:"config"`
	Features     []string               `json:"features"`
	Metrics      IntegrationMetrics     `json:"metrics"`
	Status       string                 `json:"status"`
	LastError    string                 `json:"last_error,omitempty"`
	ConnectedAt  time.Time              `json:"connected_at"`
	LastActivity time.Time              `json:"last_activity"`
}

// IntegrationMetrics contains integration metrics
type IntegrationMetrics struct {
	Requests            int64         `json:"requests"`
	Responses           int64         `json:"responses"`
	ErrorRate           float64       `json:"error_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastResponseTime    time.Time     `json:"last_response_time"`
	Throughput          float64       `json:"throughput"`
	CacheHits           int64         `json:"cache_hits"`
	CacheMisses         int64         `json:"cache_misses"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
}

// CogneeMetrics contains Cognee performance metrics
type CogneeMetrics struct {
	StartTime           time.Time             `json:"start_time"`
	Uptime              time.Duration         `json:"uptime"`
	TotalRequests       int64                 `json:"total_requests"`
	SuccessRequests     int64                 `json:"success_requests"`
	ErrorRequests       int64                 `json:"error_requests"`
	AverageResponseTime time.Duration         `json:"average_response_time"`
	MemoryUsage         int64                 `json:"memory_usage"`
	CPUUsage            float64               `json:"cpu_usage"`
	GPUUsage            float64               `json:"gpu_usage"`
	KnowledgeGraph      KnowledgeGraphMetrics `json:"knowledge_graph"`
	Cache               CacheMetrics          `json:"cache"`
	API                 APIMetrics            `json:"api"`
	BackgroundTasks     BackgroundTaskMetrics `json:"background_tasks"`
}

// KnowledgeGraphMetrics contains knowledge graph metrics
type KnowledgeGraphMetrics struct {
	Nodes      int64     `json:"nodes"`
	Edges      int64     `json:"edges"`
	Complexity float64   `json:"complexity"`
	LastUpdate time.Time `json:"last_update"`
	Updates    int64     `json:"updates"`
	Queries    int64     `json:"queries"`
	Inserts    int64     `json:"inserts"`
	Deletes    int64     `json:"deletes"`
}

// CacheMetrics contains cache metrics
type CacheMetrics struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
	Size       int64   `json:"size"`
	Evictions  int64   `json:"evictions"`
	Operations int64   `json:"operations"`
}

// APIMetrics contains API metrics
type APIMetrics struct {
	Requests      int64                    `json:"requests"`
	Responses     int64                    `json:"responses"`
	ErrorRate     float64                  `json:"error_rate"`
	ByEndpoint    map[string]int64         `json:"by_endpoint"`
	ByMethod      map[string]int64         `json:"by_method"`
	ByStatus      map[int]int64            `json:"by_status"`
	ResponseTimes map[string]time.Duration `json:"response_times"`
	LastRequest   time.Time                `json:"last_request"`
}

// BackgroundTaskMetrics contains background task metrics
type BackgroundTaskMetrics struct {
	Running         int              `json:"running"`
	Completed       int64            `json:"completed"`
	Failed          int64            `json:"failed"`
	AverageDuration time.Duration    `json:"average_duration"`
	Types           map[string]int64 `json:"types"`
}

// NewCogneeManager creates a new Cognee manager
func NewCogneeManager(config *config.HelixConfig, hwProfile *hardware.Profile) (*CogneeManager, error) {
	// Default Cognee configuration
	cogneeConfig := &CogneeConfig{
		Enabled:       true,
		AutoStart:     true,
		Host:          "localhost",
		Port:          8000,
		DynamicConfig: true,
		Source:        "https://github.com/cognee-ai/cognee.git",
		Branch:        "main",
		BuildPath:     "external/cognee",
		Optimization: OptimizationConfig{
			HostAware:          true,
			CPUOptimization:    true,
			GPUOptimization:    true,
			MemoryOptimization: true,
			HostSpecific:       make(map[string]interface{}),
		},
		Features: FeatureConfig{
			KnowledgeGraph:     true,
			SemanticSearch:     true,
			RealTimeProcessing: true,
			MultiModalSupport:  true,
			GraphAnalytics:     true,
			AdvancedInsights:   true,
			AutoOptimization:   true,
		},
		Providers: make(ProviderConfigMap),
		API: APIConfig{
			Enabled:      true,
			Host:         "localhost",
			Port:         8000,
			AuthRequired: false,
			RateLimit:    1000,
			CORS:         true,
			DocsEnabled:  true,
			Timeout:      30 * time.Second,
		},
		Performance: PerformanceConfig{
			Workers:           4,
			QueueSize:         1000,
			BatchSize:         32,
			FlushInterval:     5 * time.Second,
			OptimizationLevel: "high",
		},
		Cache: CacheConfig{
			Enabled:     true,
			Type:        "redis",
			Host:        "localhost",
			Port:        6379,
			Database:    0,
			TTL:         1 * time.Hour,
			Compression: true,
		},
		Monitoring: MonitoringConfig{
			Enabled:      true,
			MetricsPort:  9090,
			HealthCheck:  30 * time.Second,
			LogLevel:     "info",
			TraceEnabled: true,
		},
	}

	// Load configuration from helix config if available
	if config.Cognee != nil {
		cogneeConfig.Enabled = config.Cognee.Enabled
		cogneeConfig.AutoStart = config.Cognee.AutoStart
		if config.Cognee.Host != "" {
			cogneeConfig.Host = config.Cognee.Host
		}
		if config.Cognee.Port > 0 {
			cogneeConfig.Port = config.Cognee.Port
		}
		cogneeConfig.DynamicConfig = config.Cognee.DynamicConfig

		// Load optimization config
		if config.Cognee.Optimization != nil {
			cogneeConfig.Optimization.HostAware = config.Cognee.Optimization.HostAware
			cogneeConfig.Optimization.CPUOptimization = config.Cognee.Optimization.CPUOptimization
			cogneeConfig.Optimization.GPUOptimization = config.Cognee.Optimization.GPUOptimization
			cogneeConfig.Optimization.MemoryOptimization = config.Cognee.Optimization.MemoryOptimization
		}

		// Load features config
		if config.Cognee.Features != nil {
			cogneeConfig.Features.KnowledgeGraph = config.Cognee.Features.KnowledgeGraph
			cogneeConfig.Features.SemanticSearch = config.Cognee.Features.SemanticSearch
			cogneeConfig.Features.RealTimeProcessing = config.Cognee.Features.RealTimeProcessing
			cogneeConfig.Features.MultiModalSupport = config.Cognee.Features.MultiModalSupport
			cogneeConfig.Features.GraphAnalytics = config.Cognee.Features.GraphAnalytics
			cogneeConfig.Features.AdvancedInsights = config.Cognee.Features.AdvancedInsights
			cogneeConfig.Features.AutoOptimization = config.Cognee.Features.AutoOptimization
		}

		// Load providers config
		if config.Cognee.Providers != nil {
			for name, provConfig := range config.Cognee.Providers {
				cogneeConfig.Providers[name] = ProviderConfig{
					Enabled:     provConfig.Enabled,
					Integration: provConfig.Integration,
					Priority:    provConfig.Priority,
					Features:    provConfig.Features,
				}
			}
		}
	}

	// Create logger
	logger := logging.NewLogger("cognee_manager")

	manager := &CogneeManager{
		config:               cogneeConfig,
		hwProfile:            hwProfile,
		logger:               logger,
		cogneeDir:            filepath.Join(os.Getenv("HOME"), ".helix", "cognee"),
		configPath:           filepath.Join(os.Getenv("HOME"), ".helix", "cognee", "config.yaml"),
		apiClient:            &http.Client{Timeout: 30 * time.Second},
		baseURL:              fmt.Sprintf("http://%s:%d", cogneeConfig.Host, cogneeConfig.Port),
		stopChan:             make(chan struct{}),
		providerIntegrations: make(map[string]*ProviderIntegration),
		modelIntegrations:    make(map[string]*ModelIntegration),
		metrics:              &CogneeMetrics{StartTime: time.Now()},
	}

	// Initialize performance optimizer
	optimizer, err := NewPerformanceOptimizer(cogneeConfig, hwProfile)
	if err != nil {
		logger.Warn("Failed to initialize performance optimizer", "error", err)
	} else {
		manager.optimizer = optimizer
	}

	// Initialize cache manager
	cacheManager, err := NewCacheManager(cogneeConfig.Cache)
	if err != nil {
		logger.Warn("Failed to initialize cache manager", "error", err)
	} else {
		manager.cache = cacheManager
	}

	return manager, nil
}

// Initialize sets up Cognee integration
func (cm *CogneeManager) Initialize(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.initialized {
		return nil
	}

	if !cm.config.Enabled {
		cm.logger.Info("Cognee is disabled in configuration")
		return nil
	}

	cm.logger.Info("Initializing Cognee integration...")

	// Create directories
	if err := cm.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Clone and build Cognee
	if err := cm.setupCognee(ctx); err != nil {
		return fmt.Errorf("failed to setup Cognee: %w", err)
	}

	// Apply dynamic configuration
	if cm.config.DynamicConfig {
		if err := cm.applyDynamicConfig(); err != nil {
			cm.logger.Warn("Failed to apply dynamic configuration", "error", err)
		}
	}

	// Initialize health status
	cm.healthStatus = &HealthStatus{
		Status:    "initializing",
		Services:  make(map[string]string),
		Details:   make(map[string]interface{}),
		LastCheck: time.Now(),
	}

	// Initialize provider integrations
	if err := cm.initializeProviderIntegrations(ctx); err != nil {
		return fmt.Errorf("failed to initialize provider integrations: %w", err)
	}

	cm.initialized = true
	cm.logger.Info("Cognee integration initialized successfully")

	// Auto-start if configured
	if cm.config.AutoStart {
		return cm.Start(ctx)
	}

	return nil
}

// Start starts Cognee service
func (cm *CogneeManager) Start(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.config.Enabled {
		return fmt.Errorf("Cognee is disabled")
	}

	if cm.running {
		return fmt.Errorf("Cognee is already running")
	}

	if !cm.initialized {
		return fmt.Errorf("Cognee not initialized")
	}

	cm.logger.Info("Starting Cognee service...")

	// Start Cognee server
	if err := cm.startCogneeServer(ctx); err != nil {
		return fmt.Errorf("failed to start Cognee server: %w", err)
	}

	// Wait for server to be ready
	if err := cm.waitForServer(ctx); err != nil {
		return fmt.Errorf("Cognee server failed to start: %w", err)
	}

	// Start background tasks
	cm.bgTasks.Add(1)
	go cm.healthCheckLoop(ctx)

	cm.bgTasks.Add(1)
	go cm.metricsCollectionLoop(ctx)

	cm.bgTasks.Add(1)
	go cm.optimizationLoop(ctx)

	cm.running = true
	cm.metrics.StartTime = time.Now()

	cm.logger.Info("Cognee service started successfully")
	return nil
}

// Stop stops Cognee service
func (cm *CogneeManager) Stop(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return nil
	}

	cm.logger.Info("Stopping Cognee service...")

	// Signal background tasks to stop
	close(cm.stopChan)

	// Wait for background tasks to complete
	done := make(chan struct{})
	go func() {
		cm.bgTasks.Wait()
		close(done)
	}()

	select {
	case <-done:
		cm.logger.Info("All background tasks stopped")
	case <-ctx.Done():
		return ctx.Err()
	}

	// Stop Cognee server
	if err := cm.stopCogneeServer(); err != nil {
		cm.logger.Warn("Failed to stop Cognee server gracefully", "error", err)
	}

	cm.running = false
	cm.logger.Info("Cognee service stopped")
	return nil
}

// IsEnabled returns whether Cognee is enabled
func (cm *CogneeManager) IsEnabled() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Enabled
}

// IsRunning returns whether Cognee is running
func (cm *CogneeManager) IsRunning() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.running
}

// GetConfig returns Cognee configuration
func (cm *CogneeManager) GetConfig() *CogneeConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent modifications
	configCopy := *cm.config
	return &configCopy
}

// GetHealth returns Cognee health status
func (cm *CogneeManager) GetHealth() *HealthStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.healthStatus == nil {
		return &HealthStatus{Status: "not_initialized"}
	}

	// Return a copy
	statusCopy := *cm.healthStatus
	return &statusCopy
}

// GetMetrics returns Cognee metrics
func (cm *CogneeManager) GetMetrics() *CogneeMetrics {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metrics := *cm.metrics
	metrics.Uptime = time.Since(metrics.StartTime)
	return &metrics
}

// UpdateConfig updates Cognee configuration
func (cm *CogneeManager) UpdateConfig(newConfig *CogneeConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldEnabled := cm.config.Enabled
	cm.config = newConfig

	// Handle enable/disable
	if !oldEnabled && newConfig.Enabled {
		// Cognee was enabled
		return cm.Initialize(context.Background())
	} else if oldEnabled && !newConfig.Enabled {
		// Cognee was disabled
		if cm.running {
			if err := cm.Stop(context.Background()); err != nil {
				return err
			}
		}
	}

	// Apply dynamic configuration
	if newConfig.DynamicConfig {
		if err := cm.applyDynamicConfig(); err != nil {
			cm.logger.Warn("Failed to apply updated dynamic configuration", "error", err)
		}
	}

	return nil
}

// Private helper methods

func (cm *CogneeManager) createDirectories() error {
	dirs := []string{
		cm.cogneeDir,
		filepath.Dir(cm.configPath),
		filepath.Join(cm.cogneeDir, "logs"),
		filepath.Join(cm.cogneeDir, "cache"),
		filepath.Join(cm.cogneeDir, "data"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (cm *CogneeManager) setupCognee(ctx context.Context) error {
	// Check if Cognee is already built
	if _, err := os.Stat(cm.config.BuildPath); err == nil {
		cm.logger.Info("Cognee already exists, checking for updates")
		return cm.updateCognee(ctx)
	}

	cm.logger.Info("Cloning Cognee repository...")

	// Clone Cognee
	cmd := exec.CommandContext(ctx, "git", "clone",
		"--branch", cm.config.Branch,
		cm.config.Source,
		cm.config.BuildPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone Cognee: %w", err)
	}

	return cm.buildCognee(ctx)
}

func (cm *CogneeManager) updateCognee(ctx context.Context) error {
	cm.logger.Info("Updating Cognee repository...")

	cmd := exec.CommandContext(ctx, "git", "-C", cm.config.BuildPath, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		cm.logger.Warn("Failed to update Cognee, using existing version", "error", err)
		return nil
	}

	return cm.buildCognee(ctx)
}

func (cm *CogneeManager) buildCognee(ctx context.Context) error {
	cm.logger.Info("Building Cognee...")

	// Install dependencies
	cmd := exec.CommandContext(ctx, "pip", "install", "-r", "requirements.txt")
	cmd.Dir = cm.config.BuildPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Cognee dependencies: %w", err)
	}

	// Build Cognee
	cmd = exec.CommandContext(ctx, "python", "setup.py", "build_ext", "--inplace")
	cmd.Dir = cm.config.BuildPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build Cognee: %w", err)
	}

	cm.logger.Info("Cognee built successfully")
	return nil
}

func (cm *CogneeManager) applyDynamicConfig() error {
	if !cm.config.DynamicConfig {
		return nil
	}

	cm.logger.Info("Applying dynamic configuration based on host profile...")

	// Apply host-specific optimizations
	if cm.config.Optimization.HostAware {
		optimizer := &HostOptimizer{Profile: cm.hwProfile}
		optimizedConfig := optimizer.OptimizeConfig(cm.config)
		cm.config.Optimization = optimizedConfig.Optimization
		cm.config.Performance = optimizedConfig.Performance
	}

	// Save configuration
	return cm.saveConfig()
}

func (cm *CogneeManager) saveConfig() error {
	configData, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.configPath, configData, 0644)
}

func (cm *CogneeManager) startCogneeServer(ctx context.Context) error {
	cm.logger.Info("Starting Cognee server...")

	// Prepare environment
	env := os.Environ()
	env = append(env, "COGNEE_HOST="+cm.config.Host)
	env = append(env, "COGNEE_PORT="+fmt.Sprintf("%d", cm.config.Port))
	env = append(env, "COGNEE_LOG_LEVEL="+cm.config.Monitoring.LogLevel)

	// Start Cognee server
	cmd := exec.CommandContext(ctx, "python", "-m", "cognee.server")
	cmd.Dir = cm.config.BuildPath
	cmd.Env = env

	// Redirect output to log files
	logFile, err := os.OpenFile(
		filepath.Join(cm.cogneeDir, "logs", "cognee.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Cognee server: %w", err)
	}

	cm.cogneeProcess = cmd.Process
	return nil
}

func (cm *CogneeManager) stopCogneeServer() error {
	if cm.cogneeProcess == nil {
		return nil
	}

	cm.logger.Info("Stopping Cognee server...")

	// Send SIGTERM for graceful shutdown
	if err := cm.cogneeProcess.Signal(os.Interrupt); err != nil {
		cm.logger.Warn("Failed to send SIGTERM to Cognee process", "error", err)
	}

	// Wait for graceful shutdown
	done := make(chan error, 1)
	go func() {
		_, err := cm.cogneeProcess.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			cm.logger.Warn("Cognee server exited with error", "error", err)
		}
	case <-time.After(30 * time.Second):
		cm.logger.Warn("Cognee server did not stop gracefully, force killing")
		cm.cogneeProcess.Kill()
	}

	cm.cogneeProcess = nil
	return nil
}

func (cm *CogneeManager) waitForServer(ctx context.Context) error {
	cm.logger.Info("Waiting for Cognee server to be ready...")

	client := &http.Client{Timeout: 2 * time.Second}
	healthURL := fmt.Sprintf("%s/health", cm.baseURL)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
			if err != nil {
				continue
			}

			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				cm.logger.Info("Cognee server is ready")
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

func (cm *CogneeManager) initializeProviderIntegrations(ctx context.Context) error {
	cm.logger.Info("Initializing provider integrations...")

	for providerName, providerConfig := range cm.config.Providers {
		if !providerConfig.Enabled {
			continue
		}

		integration := &ProviderIntegration{
			Provider:    providerName,
			Connected:   false,
			Config:      providerConfig,
			Features:    providerConfig.Features,
			Metrics:     IntegrationMetrics{},
			Status:      "initializing",
			ConnectedAt: time.Now(),
		}

		cm.providerIntegrations[providerName] = integration

		// Initialize integration in background
		cm.bgTasks.Add(1)
		go cm.initializeProviderIntegration(ctx, providerName, integration)
	}

	return nil
}

func (cm *CogneeManager) initializeProviderIntegration(ctx context.Context,
	providerName string, integration *ProviderIntegration) {
	defer cm.bgTasks.Done()

	cm.logger.Info("Initializing provider integration", "provider", providerName)

	// Get provider instance
	prov := provider.GetProvider(providerName)
	if prov == nil {
		cm.logger.Warn("Provider not found", "provider", providerName)
		integration.Status = "provider_not_found"
		integration.LastError = "Provider not found in registry"
		return
	}

	// Check if provider supports Cognee integration
	if !prov.SupportsCognee() {
		cm.logger.Info("Provider does not support Cognee integration", "provider", providerName)
		integration.Status = "not_supported"
		return
	}

	// Initialize Cognee integration
	if err := prov.InitializeCognee(cm.config, cm.hwProfile); err != nil {
		cm.logger.Error("Failed to initialize Cognee integration",
			"provider", providerName, "error", err)
		integration.Status = "initialization_failed"
		integration.LastError = err.Error()
		return
	}

	integration.Connected = true
	integration.Status = "connected"
	integration.LastActivity = time.Now()

	cm.logger.Info("Provider integration initialized", "provider", providerName)
}

func (cm *CogneeManager) healthCheckLoop(ctx context.Context) {
	defer cm.bgTasks.Done()

	ticker := time.NewTicker(cm.config.Monitoring.HealthCheck)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopChan:
			return
		case <-ticker.C:
			cm.performHealthCheck()
		}
	}
}

func (cm *CogneeManager) performHealthCheck() {
	if !cm.running {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check Cognee server health
	client := &http.Client{Timeout: 5 * time.Second}
	healthURL := fmt.Sprintf("%s/health", cm.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		cm.updateHealthStatus("unhealthy", "health_check_error", err.Error())
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		cm.updateHealthStatus("unhealthy", "health_check_error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		cm.updateHealthStatus("unhealthy", "health_check_failed",
			fmt.Sprintf("HTTP %d", resp.StatusCode))
		return
	}

	// Parse health response
	var health HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err == nil {
		cm.healthStatus = &health
	} else {
		cm.healthStatus.Status = "healthy"
	}

	cm.healthStatus.LastCheck = time.Now()

	// Update provider integration health
	cm.updateProviderHealth()
}

func (cm *CogneeManager) updateHealthStatus(status, reason, details string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.healthStatus != nil {
		cm.healthStatus.Status = status
		cm.healthStatus.Details = map[string]interface{}{
			"reason":  reason,
			"details": details,
		}
		cm.healthStatus.LastCheck = time.Now()
	}
}

func (cm *CogneeManager) updateProviderHealth() {
	for providerName, integration := range cm.providerIntegrations {
		if !integration.Connected {
			continue
		}

		// Check provider health
		prov := provider.GetProvider(providerName)
		if prov == nil {
			integration.Status = "provider_unavailable"
			integration.Connected = false
			continue
		}

		healthy := prov.IsCogneeHealthy()
		if healthy {
			integration.Status = "healthy"
		} else {
			integration.Status = "unhealthy"
		}

		integration.LastActivity = time.Now()
	}
}

func (cm *CogneeManager) metricsCollectionLoop(ctx context.Context) {
	defer cm.bgTasks.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopChan:
			return
		case <-ticker.C:
			cm.collectMetrics()
		}
	}
}

func (cm *CogneeManager) collectMetrics() {
	if !cm.running {
		return
	}

	// Update system metrics
	cm.metrics.MemoryUsage = cm.getMemoryUsage()
	cm.metrics.CPUUsage = cm.getCPUUsage()
	cm.metrics.GPUUsage = cm.getGPUUsage()

	// Update API metrics
	cm.updateAPIMetrics()

	// Update cache metrics
	if cm.cache != nil {
		cm.metrics.Cache = cm.cache.GetMetrics()
	}

	// Update integration metrics
	cm.updateIntegrationMetrics()
}

func (cm *CogneeManager) getMemoryUsage() int64 {
	// Implementation would get actual memory usage
	return 0 // Placeholder
}

func (cm *CogneeManager) getCPUUsage() float64 {
	// Implementation would get actual CPU usage
	return 0.0 // Placeholder
}

func (cm *CogneeManager) getGPUUsage() float64 {
	// Implementation would get actual GPU usage
	return 0.0 // Placeholder
}

func (cm *CogneeManager) updateAPIMetrics() {
	// Implementation would update API metrics from Cognee server
}

func (cm *CogneeManager) updateIntegrationMetrics() {
	// Implementation would update integration metrics from providers
}

func (cm *CogneeManager) optimizationLoop(ctx context.Context) {
	defer cm.bgTasks.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopChan:
			return
		case <-ticker.C:
			cm.performOptimization()
		}
	}
}

func (cm *CogneeManager) performOptimization() {
	if !cm.config.Features.AutoOptimization {
		return
	}

	cm.logger.Debug("Performing automatic optimization...")

	if cm.optimizer != nil {
		// Apply performance optimizations
		if err := cm.optimizer.Optimize(); err != nil {
			cm.logger.Warn("Failed to apply optimizations", "error", err)
		}
	}
}

// Integration methods for providers

// AddKnowledge adds knowledge to Cognee from a provider
func (cm *CogneeManager) AddKnowledge(ctx context.Context, providerName,
	modelName string, data interface{}, metadata map[string]interface{}) ([]string, error) {

	if !cm.running {
		return nil, fmt.Errorf("Cognee is not running")
	}

	cm.logger.Debug("Adding knowledge to Cognee",
		"provider", providerName, "model", modelName)

	// Prepare request
	request := map[string]interface{}{
		"data":     data,
		"metadata": metadata,
		"source": map[string]string{
			"provider": providerName,
			"model":    modelName,
		},
		"timestamp": time.Now().Unix(),
	}

	// Send to Cognee API
	reqData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/knowledge", cm.baseURL),
		strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := cm.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Cognee API returned status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Success bool     `json:"success"`
		Nodes   []string `json:"nodes"`
		Error   string   `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("Cognee API error: %s", response.Error)
	}

	// Update metrics
	cm.metrics.TotalRequests++
	cm.metrics.SuccessRequests++
	cm.metrics.KnowledgeGraph.Inserts += int64(len(response.Nodes))

	return response.Nodes, nil
}

// SearchKnowledge searches knowledge in Cognee
func (cm *CogneeManager) SearchKnowledge(ctx context.Context, query string,
	filters map[string]interface{}, limit int) ([]map[string]interface{}, error) {

	if !cm.running {
		return nil, fmt.Errorf("Cognee is not running")
	}

	cm.logger.Debug("Searching knowledge in Cognee", "query", query)

	// Check cache first
	cacheKey := fmt.Sprintf("search:%s:%s:%d", query,
		hashMap(filters), limit)

	if cm.cache != nil {
		if result, err := cm.cache.Get(cacheKey); err == nil {
			cm.metrics.KnowledgeGraph.CacheHits++
			return result.([]map[string]interface{}), nil
		}
	}

	// Prepare request
	request := map[string]interface{}{
		"query":   query,
		"filters": filters,
		"limit":   limit,
	}

	reqData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/search", cm.baseURL),
		strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := cm.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Cognee API returned status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Success bool                     `json:"success"`
		Results []map[string]interface{} `json:"results"`
		Error   string                   `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("Cognee API error: %s", response.Error)
	}

	// Update cache
	if cm.cache != nil {
		cm.cache.Set(cacheKey, response.Results, cm.config.Cache.TTL)
	}

	// Update metrics
	cm.metrics.TotalRequests++
	cm.metrics.SuccessRequests++
	cm.metrics.KnowledgeGraph.Queries++
	cm.metrics.KnowledgeGraph.CacheMisses++

	return response.Results, nil
}

// GetInsights gets insights from Cognee
func (cm *CogneeManager) GetInsights(ctx context.Context, analysisType string,
	parameters map[string]interface{}) (map[string]interface{}, error) {

	if !cm.running {
		return nil, fmt.Errorf("Cognee is not running")
	}

	cm.logger.Debug("Getting insights from Cognee", "type", analysisType)

	// Prepare request
	request := map[string]interface{}{
		"analysis_type": analysisType,
		"parameters":    parameters,
	}

	reqData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/insights", cm.baseURL),
		strings.NewReader(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := cm.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Cognee API returned status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Success  bool                   `json:"success"`
		Insights map[string]interface{} `json:"insights"`
		Error    string                 `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("Cognee API error: %s", response.Error)
	}

	// Update metrics
	cm.metrics.TotalRequests++
	cm.metrics.SuccessRequests++

	return response.Insights, nil
}

// Helper functions

func hashMap(m map[string]interface{}) string {
	data, _ := json.Marshal(m)
	return string(data)
}
