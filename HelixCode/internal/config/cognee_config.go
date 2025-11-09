package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CogneeConfig contains Cognee.ai configuration
type CogneeConfig struct {
	// Basic Settings
	Enabled    bool `json:"enabled" yaml:"enabled"`
	AutoStart  bool `json:"auto_start" yaml:"auto_start"`
	Host       string `json:"host" yaml:"host"`
	Port       int    `json:"port" yaml:"port"`
	
	// Dynamic Configuration
	DynamicConfig bool `json:"dynamic_config" yaml:"dynamic_config"`
	
	// Repository Settings
	Source    string `json:"source,omitempty" yaml:"source,omitempty"`
	Branch    string `json:"branch,omitempty" yaml:"branch,omitempty"`
	BuildPath string `json:"build_path,omitempty" yaml:"build_path,omitempty"`
	
	// Optimization Settings
	Optimization *CogneeOptimizationConfig `json:"optimization,omitempty" yaml:"optimization,omitempty"`
	
	// Feature Settings
	Features *CogneeFeatureConfig `json:"features,omitempty" yaml:"features,omitempty"`
	
	// Provider Integration
	Providers map[string]*CogneeProviderConfig `json:"providers,omitempty" yaml:"providers,omitempty"`
	
	// API Configuration
	API *CogneeAPIConfig `json:"api,omitempty" yaml:"api,omitempty"`
	
	// Performance Configuration
	Performance *CogneePerformanceConfig `json:"performance,omitempty" yaml:"performance,omitempty"`
	
	// Cache Configuration
	Cache *CogneeCacheConfig `json:"cache,omitempty" yaml:"cache,omitempty"`
	
	// Monitoring Configuration
	Monitoring *CogneeMonitoringConfig `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`
}

// CogneeOptimizationConfig contains optimization settings
type CogneeOptimizationConfig struct {
	HostAware         bool                   `json:"host_aware" yaml:"host_aware"`
	CPUOptimization   bool                   `json:"cpu_optimization" yaml:"cpu_optimization"`
	GPUOptimization   bool                   `json:"gpu_optimization" yaml:"gpu_optimization"`
	MemoryOptimization bool                   `json:"memory_optimization" yaml:"memory_optimization"`
	HostSpecific      map[string]interface{} `json:"host_specific,omitempty" yaml:"host_specific,omitempty"`
}

// CogneeFeatureConfig contains feature settings
type CogneeFeatureConfig struct {
	KnowledgeGraph     bool `json:"knowledge_graph" yaml:"knowledge_graph"`
	SemanticSearch     bool `json:"semantic_search" yaml:"semantic_search"`
	RealTimeProcessing  bool `json:"real_time_processing" yaml:"real_time_processing"`
	MultiModalSupport   bool `json:"multi_modal_support" yaml:"multi_modal_support"`
	GraphAnalytics      bool `json:"graph_analytics" yaml:"graph_analytics"`
	AdvancedInsights    bool `json:"advanced_insights" yaml:"advanced_insights"`
	AutoOptimization    bool `json:"auto_optimization" yaml:"auto_optimization"`
}

// CogneeProviderConfig contains provider-specific Cognee settings
type CogneeProviderConfig struct {
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	Integration string   `json:"integration" yaml:"integration"`
	Priority    int      `json:"priority,omitempty" yaml:"priority,omitempty"`
	Features    []string `json:"features,omitempty" yaml:"features,omitempty"`
}

// CogneeAPIConfig contains API configuration
type CogneeAPIConfig struct {
	Enabled       bool          `json:"enabled" yaml:"enabled"`
	Host          string        `json:"host" yaml:"host"`
	Port          int           `json:"port" yaml:"port"`
	AuthRequired  bool          `json:"auth_required" yaml:"auth_required"`
	RateLimit     int           `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
	CORS          bool          `json:"cors" yaml:"cors"`
	DocsEnabled   bool          `json:"docs_enabled" yaml:"docs_enabled"`
	Timeout       time.Duration `json:"timeout" yaml:"timeout"`
	MaxRequestSize int64        `json:"max_request_size,omitempty" yaml:"max_request_size,omitempty"`
}

// CogneePerformanceConfig contains performance configuration
type CogneePerformanceConfig struct {
	Workers          int           `json:"workers" yaml:"workers"`
	QueueSize        int           `json:"queue_size" yaml:"queue_size"`
	BatchSize        int           `json:"batch_size" yaml:"batch_size"`
	FlushInterval    time.Duration `json:"flush_interval" yaml:"flush_interval"`
	MaxMemory        int64         `json:"max_memory,omitempty" yaml:"max_memory,omitempty"`
	CacheSize        int64         `json:"cache_size,omitempty" yaml:"cache_size,omitempty"`
	OptimizationLevel string        `json:"optimization_level" yaml:"optimization_level"`
}

// CogneeCacheConfig contains cache configuration
type CogneeCacheConfig struct {
	Enabled     bool          `json:"enabled" yaml:"enabled"`
	Type        string        `json:"type" yaml:"type"`
	Host        string        `json:"host,omitempty" yaml:"host,omitempty"`
	Port        int           `json:"port,omitempty" yaml:"port,omitempty"`
	Database    int           `json:"database,omitempty" yaml:"database,omitempty"`
	TTL         time.Duration `json:"ttl" yaml:"ttl"`
	MaxSize     int64         `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	Compression  bool          `json:"compression" yaml:"compression"`
}

// CogneeMonitoringConfig contains monitoring configuration
type CogneeMonitoringConfig struct {
	Enabled      bool          `json:"enabled" yaml:"enabled"`
	MetricsPort  int           `json:"metrics_port" yaml:"metrics_port"`
	HealthCheck  time.Duration `json:"health_check" yaml:"health_check"`
	LogLevel     string        `json:"log_level" yaml:"log_level"`
	TraceEnabled bool          `json:"trace_enabled" yaml:"trace_enabled"`
	AlertWebhook string        `json:"alert_webhook,omitempty" yaml:"alert_webhook,omitempty"`
}

// DefaultCogneeConfig returns default Cognee configuration
func DefaultCogneeConfig() *CogneeConfig {
	return &CogneeConfig{
		Enabled:       true,
		AutoStart:     true,
		Host:          "localhost",
		Port:          8000,
		DynamicConfig:  true,
		Source:        "https://github.com/cognee-ai/cognee.git",
		Branch:        "main",
		BuildPath:     "external/cognee",
		Optimization: &CogneeOptimizationConfig{
			HostAware:         true,
			CPUOptimization:   true,
			GPUOptimization:   true,
			MemoryOptimization: true,
			HostSpecific:      make(map[string]interface{}),
		},
		Features: &CogneeFeatureConfig{
			KnowledgeGraph:     true,
			SemanticSearch:     true,
			RealTimeProcessing:  true,
			MultiModalSupport:   true,
			GraphAnalytics:      true,
			AdvancedInsights:    true,
			AutoOptimization:    true,
		},
		Providers: make(map[string]*CogneeProviderConfig),
		API: &CogneeAPIConfig{
			Enabled:        true,
			Host:           "localhost",
			Port:           8000,
			AuthRequired:   false,
			RateLimit:      1000,
			CORS:           true,
			DocsEnabled:    true,
			Timeout:        30 * time.Second,
		},
		Performance: &CogneePerformanceConfig{
			Workers:           4,
			QueueSize:         1000,
			BatchSize:         32,
			FlushInterval:     5 * time.Second,
			OptimizationLevel: "high",
		},
		Cache: &CogneeCacheConfig{
			Enabled:    true,
			Type:       "redis",
			Host:       "localhost",
			Port:       6379,
			Database:   0,
			TTL:        1 * time.Hour,
			Compression: true,
		},
		Monitoring: &CogneeMonitoringConfig{
			Enabled:      true,
			MetricsPort:  9090,
			HealthCheck:  30 * time.Second,
			LogLevel:     "info",
			TraceEnabled: true,
		},
	}
}

// LoadCogneeConfig loads Cognee configuration from file
func LoadCogneeConfig(configPath string) (*CogneeConfig, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config
		return DefaultCogneeConfig(), nil
	}
	
	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Cognee config: %w", err)
	}
	
	// Parse JSON
	var config CogneeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Cognee config: %w", err)
	}
	
	// Apply defaults for missing fields
	config = config.applyDefaults()
	
	return &config, nil
}

// SaveCogneeConfig saves Cognee configuration to file
func SaveCogneeConfig(config *CogneeConfig, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Convert to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Cognee config: %w", err)
	}
	
	// Write file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write Cognee config: %w", err)
	}
	
	return nil
}

// applyDefaults applies default values for missing fields
func (config *CogneeConfig) applyDefaults() CogneeConfig {
	defaults := DefaultCogneeConfig()
	
	// Apply defaults for missing fields
	if config.Optimization == nil {
		config.Optimization = defaults.Optimization
	}
	
	if config.Features == nil {
		config.Features = defaults.Features
	}
	
	if config.Providers == nil {
		config.Providers = make(map[string]*CogneeProviderConfig)
	}
	
	if config.API == nil {
		config.API = defaults.API
	}
	
	if config.Performance == nil {
		config.Performance = defaults.Performance
	}
	
	if config.Cache == nil {
		config.Cache = defaults.Cache
	}
	
	if config.Monitoring == nil {
		config.Monitoring = defaults.Monitoring
	}
	
	return *config
}

// Validate validates Cognee configuration
func (config *CogneeConfig) Validate() error {
	// Validate basic settings
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", config.Port)
	}
	
	if config.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	
	// Validate optimization settings
	if config.Optimization != nil {
		if config.Optimization.HostSpecific == nil {
			config.Optimization.HostSpecific = make(map[string]interface{})
		}
	}
	
	// Validate API settings
	if config.API != nil {
		if config.API.Port <= 0 || config.API.Port > 65535 {
			return fmt.Errorf("invalid API port: %d", config.API.Port)
		}
		
		if config.API.Timeout <= 0 {
			config.API.Timeout = 30 * time.Second
		}
	}
	
	// Validate performance settings
	if config.Performance != nil {
		if config.Performance.Workers <= 0 {
			config.Performance.Workers = 4
		}
		
		if config.Performance.QueueSize <= 0 {
			config.Performance.QueueSize = 1000
		}
		
		if config.Performance.BatchSize <= 0 {
			config.Performance.BatchSize = 32
		}
		
		if config.Performance.FlushInterval <= 0 {
			config.Performance.FlushInterval = 5 * time.Second
		}
	}
	
	// Validate cache settings
	if config.Cache != nil {
		if config.Cache.Port <= 0 || config.Cache.Port > 65535 {
			config.Cache.Port = 6379
		}
		
		if config.Cache.TTL <= 0 {
			config.Cache.TTL = 1 * time.Hour
		}
	}
	
	// Validate monitoring settings
	if config.Monitoring != nil {
		if config.Monitoring.MetricsPort <= 0 || config.Monitoring.MetricsPort > 65535 {
			config.Monitoring.MetricsPort = 9090
		}
		
		if config.Monitoring.HealthCheck <= 0 {
			config.Monitoring.HealthCheck = 30 * time.Second
		}
		
		validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
		logLevelValid := false
		for _, level := range validLogLevels {
			if config.Monitoring.LogLevel == level {
				logLevelValid = true
				break
			}
		}
		if !logLevelValid {
			config.Monitoring.LogLevel = "info"
		}
	}
	
	return nil
}

// ToJSON converts configuration to JSON
func (config *CogneeConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// ToYAML converts configuration to YAML (basic implementation)
func (config *CogneeConfig) ToYAML() ([]byte, error) {
	// This is a basic YAML implementation
	// In production, use a proper YAML library
	yaml := fmt.Sprintf(`# Cognee.ai Configuration

# Basic Settings
enabled: %t
auto_start: %t
host: %s
port: %d
dynamic_config: %t

# Repository Settings
source: %s
branch: %s
build_path: %s

# Optimization Settings
optimization:
  host_aware: %t
  cpu_optimization: %t
  gpu_optimization: %t
  memory_optimization: %t

# Feature Settings
features:
  knowledge_graph: %t
  semantic_search: %t
  real_time_processing: %t
  multi_modal_support: %t
  graph_analytics: %t
  advanced_insights: %t
  auto_optimization: %t

# API Settings
api:
  enabled: %t
  host: %s
  port: %d
  auth_required: %t
  rate_limit: %d
  cors: %t
  docs_enabled: %t
  timeout: %s
  max_request_size: %d

# Performance Settings
performance:
  workers: %d
  queue_size: %d
  batch_size: %d
  flush_interval: %s
  max_memory: %d
  cache_size: %d
  optimization_level: %s

# Cache Settings
cache:
  enabled: %t
  type: %s
  host: %s
  port: %d
  database: %d
  ttl: %s
  max_size: %d
  compression: %t

# Monitoring Settings
monitoring:
  enabled: %t
  metrics_port: %d
  health_check: %s
  log_level: %s
  trace_enabled: %t
  alert_webhook: %s

# Provider Settings
providers:`,
		config.Enabled,
		config.AutoStart,
		config.Host,
		config.Port,
		config.DynamicConfig,
		config.Source,
		config.Branch,
		config.BuildPath,
		config.Optimization.HostAware,
		config.Optimization.CPUOptimization,
		config.Optimization.GPUOptimization,
		config.Optimization.MemoryOptimization,
		config.Features.KnowledgeGraph,
		config.Features.SemanticSearch,
		config.Features.RealTimeProcessing,
		config.Features.MultiModalSupport,
		config.Features.GraphAnalytics,
		config.Features.AdvancedInsights,
		config.Features.AutoOptimization,
		config.API.Enabled,
		config.API.Host,
		config.API.Port,
		config.API.AuthRequired,
		config.API.RateLimit,
		config.API.CORS,
		config.API.DocsEnabled,
		config.API.Timeout,
		config.API.MaxRequestSize,
		config.Performance.Workers,
		config.Performance.QueueSize,
		config.Performance.BatchSize,
		config.Performance.FlushInterval,
		config.Performance.MaxMemory,
		config.Performance.CacheSize,
		config.Performance.OptimizationLevel,
		config.Cache.Enabled,
		config.Cache.Type,
		config.Cache.Host,
		config.Cache.Port,
		config.Cache.Database,
		config.Cache.TTL,
		config.Cache.MaxSize,
		config.Cache.Compression,
		config.Monitoring.Enabled,
		config.Monitoring.MetricsPort,
		config.Monitoring.HealthCheck,
		config.Monitoring.LogLevel,
		config.Monitoring.TraceEnabled,
		config.Monitoring.AlertWebhook,
	)
	
	// Add providers
	for name, provConfig := range config.Providers {
		yaml += fmt.Sprintf(`
  %s:
    enabled: %t
    integration: %s
    priority: %d
    features: [%s]`,
			name,
			provConfig.Enabled,
			provConfig.Integration,
			provConfig.Priority,
			`"` + strings.Join(provConfig.Features, `", "`) + `"`,
		)
	}
	
	return []byte(yaml), nil
}

// Clone creates a deep copy of the configuration
func (config *CogneeConfig) Clone() *CogneeConfig {
	// Convert to JSON and back for deep copy
	data, err := json.Marshal(config)
	if err != nil {
		return nil
	}
	
	var clone CogneeConfig
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil
	}
	
	return &clone
}

// Merge merges another configuration into this one
func (config *CogneeConfig) Merge(other *CogneeConfig) {
	if other.Enabled {
		config.Enabled = other.Enabled
	}
	
	if other.AutoStart {
		config.AutoStart = other.AutoStart
	}
	
	if other.Host != "" {
		config.Host = other.Host
	}
	
	if other.Port > 0 {
		config.Port = other.Port
	}
	
	if other.Source != "" {
		config.Source = other.Source
	}
	
	if other.Branch != "" {
		config.Branch = other.Branch
	}
	
	if other.BuildPath != "" {
		config.BuildPath = other.BuildPath
	}
	
	if other.Optimization != nil {
		if config.Optimization == nil {
			config.Optimization = &CogneeOptimizationConfig{}
		}
		mergeOptimizationConfig(config.Optimization, other.Optimization)
	}
	
	if other.Features != nil {
		if config.Features == nil {
			config.Features = &CogneeFeatureConfig{}
		}
		mergeFeatureConfig(config.Features, other.Features)
	}
	
	if other.Providers != nil {
		if config.Providers == nil {
			config.Providers = make(map[string]*CogneeProviderConfig)
		}
		for name, provConfig := range other.Providers {
			config.Providers[name] = provConfig
		}
	}
	
	if other.API != nil {
		if config.API == nil {
			config.API = &CogneeAPIConfig{}
		}
		mergeAPIConfig(config.API, other.API)
	}
	
	if other.Performance != nil {
		if config.Performance == nil {
			config.Performance = &CogneePerformanceConfig{}
		}
		mergePerformanceConfig(config.Performance, other.Performance)
	}
	
	if other.Cache != nil {
		if config.Cache == nil {
			config.Cache = &CogneeCacheConfig{}
		}
		mergeCacheConfig(config.Cache, other.Cache)
	}
	
	if other.Monitoring != nil {
		if config.Monitoring == nil {
			config.Monitoring = &CogneeMonitoringConfig{}
		}
		mergeMonitoringConfig(config.Monitoring, other.Monitoring)
	}
}

// Helper functions for merging

func mergeOptimizationConfig(base, other *CogneeOptimizationConfig) {
	if other.HostAware {
		base.HostAware = other.HostAware
	}
	if other.CPUOptimization {
		base.CPUOptimization = other.CPUOptimization
	}
	if other.GPUOptimization {
		base.GPUOptimization = other.GPUOptimization
	}
	if other.MemoryOptimization {
		base.MemoryOptimization = other.MemoryOptimization
	}
	if other.HostSpecific != nil {
		if base.HostSpecific == nil {
			base.HostSpecific = make(map[string]interface{})
		}
		for k, v := range other.HostSpecific {
			base.HostSpecific[k] = v
		}
	}
}

func mergeFeatureConfig(base, other *CogneeFeatureConfig) {
	if other.KnowledgeGraph {
		base.KnowledgeGraph = other.KnowledgeGraph
	}
	if other.SemanticSearch {
		base.SemanticSearch = other.SemanticSearch
	}
	if other.RealTimeProcessing {
		base.RealTimeProcessing = other.RealTimeProcessing
	}
	if other.MultiModalSupport {
		base.MultiModalSupport = other.MultiModalSupport
	}
	if other.GraphAnalytics {
		base.GraphAnalytics = other.GraphAnalytics
	}
	if other.AdvancedInsights {
		base.AdvancedInsights = other.AdvancedInsights
	}
	if other.AutoOptimization {
		base.AutoOptimization = other.AutoOptimization
	}
}

func mergeAPIConfig(base, other *CogneeAPIConfig) {
	if other.Enabled {
		base.Enabled = other.Enabled
	}
	if other.Host != "" {
		base.Host = other.Host
	}
	if other.Port > 0 {
		base.Port = other.Port
	}
	if other.AuthRequired {
		base.AuthRequired = other.AuthRequired
	}
	if other.RateLimit > 0 {
		base.RateLimit = other.RateLimit
	}
	if other.CORS {
		base.CORS = other.CORS
	}
	if other.DocsEnabled {
		base.DocsEnabled = other.DocsEnabled
	}
	if other.Timeout > 0 {
		base.Timeout = other.Timeout
	}
	if other.MaxRequestSize > 0 {
		base.MaxRequestSize = other.MaxRequestSize
	}
}

func mergePerformanceConfig(base, other *CogneePerformanceConfig) {
	if other.Workers > 0 {
		base.Workers = other.Workers
	}
	if other.QueueSize > 0 {
		base.QueueSize = other.QueueSize
	}
	if other.BatchSize > 0 {
		base.BatchSize = other.BatchSize
	}
	if other.FlushInterval > 0 {
		base.FlushInterval = other.FlushInterval
	}
	if other.MaxMemory > 0 {
		base.MaxMemory = other.MaxMemory
	}
	if other.CacheSize > 0 {
		base.CacheSize = other.CacheSize
	}
	if other.OptimizationLevel != "" {
		base.OptimizationLevel = other.OptimizationLevel
	}
}

func mergeCacheConfig(base, other *CogneeCacheConfig) {
	if other.Enabled {
		base.Enabled = other.Enabled
	}
	if other.Type != "" {
		base.Type = other.Type
	}
	if other.Host != "" {
		base.Host = other.Host
	}
	if other.Port > 0 {
		base.Port = other.Port
	}
	if other.Database > 0 {
		base.Database = other.Database
	}
	if other.TTL > 0 {
		base.TTL = other.TTL
	}
	if other.MaxSize > 0 {
		base.MaxSize = other.MaxSize
	}
	if other.Compression {
		base.Compression = other.Compression
	}
}

func mergeMonitoringConfig(base, other *CogneeMonitoringConfig) {
	if other.Enabled {
		base.Enabled = other.Enabled
	}
	if other.MetricsPort > 0 {
		base.MetricsPort = other.MetricsPort
	}
	if other.HealthCheck > 0 {
		base.HealthCheck = other.HealthCheck
	}
	if other.LogLevel != "" {
		base.LogLevel = other.LogLevel
	}
	if other.TraceEnabled {
		base.TraceEnabled = other.TraceEnabled
	}
	if other.AlertWebhook != "" {
		base.AlertWebhook = other.AlertWebhook
	}
}