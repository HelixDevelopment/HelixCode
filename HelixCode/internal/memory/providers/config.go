package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/memory"
)

// ConfigManager manages provider configurations
type ConfigManager struct {
	configPath string
	config     *ProviderConfig
	logger     logging.Logger
}

// ProviderConfig contains all provider configurations
type ProviderConfig struct {
	DefaultProvider string                           `json:"default_provider"`
	Providers       map[string]*SingleProviderConfig `json:"providers"`
	Manager         *ManagerConfig                   `json:"manager"`
	Backup          *BackupConfig                    `json:"backup"`
	Monitoring      *MonitoringConfig                `json:"monitoring"`
	Security        *SecurityConfig                  `json:"security"`
	Performance     *PerformanceConfig               `json:"performance"`
}

// SingleProviderConfig contains configuration for a single provider
type SingleProviderConfig struct {
	Type      memory.ProviderType    `json:"type"`
	Enabled   bool                   `json:"enabled"`
	Config    map[string]interface{} `json:"config"`
	Priority  int                    `json:"priority"`
	Tags      []string               `json:"tags"`
	Resources *ResourceConfig        `json:"resources"`
	Health    *HealthConfig          `json:"health"`
	Backup    *ProviderBackupConfig  `json:"backup"`
}

// ResourceConfig contains resource configuration
type ResourceConfig struct {
	MaxMemory      int64   `json:"max_memory"`
	MaxCPU         float64 `json:"max_cpu"`
	MaxConnections int     `json:"max_connections"`
	Timeout        int64   `json:"timeout"`
	Retries        int     `json:"retries"`
}

// HealthConfig contains health check configuration
type HealthConfig struct {
	Enabled          bool            `json:"enabled"`
	Interval         int64           `json:"interval"`
	Timeout          int64           `json:"timeout"`
	FailureThreshold int             `json:"failure_threshold"`
	RetryAttempts    int             `json:"retry_attempts"`
	Alerting         *AlertingConfig `json:"alerting"`
}

// AlertingConfig contains alerting configuration
type AlertingConfig struct {
	Enabled    bool               `json:"enabled"`
	Channels   []string           `json:"channels"`
	Conditions []string           `json:"conditions"`
	Thresholds map[string]float64 `json:"thresholds"`
	Cooldown   int64              `json:"cooldown"`
}

// ProviderBackupConfig contains backup configuration for a provider
type ProviderBackupConfig struct {
	Enabled     bool     `json:"enabled"`
	Schedule    string   `json:"schedule"`
	Retention   int      `json:"retention"`
	Compression bool     `json:"compression"`
	Destination string   `json:"destination"`
	Include     []string `json:"include"`
	Exclude     []string `json:"exclude"`
}

// BackupConfig contains global backup configuration
type BackupConfig struct {
	Enabled          bool   `json:"enabled"`
	DefaultSchedule  string `json:"default_schedule"`
	DefaultRetention int    `json:"default_retention"`
	CompressionType  string `json:"compression_type"`
	StorageLocation  string `json:"storage_location"`
	Encryption       bool   `json:"encryption"`
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	Enabled          bool            `json:"enabled"`
	MetricsInterval  int64           `json:"metrics_interval"`
	LogLevel         string          `json:"log_level"`
	MetricsEndpoint  string          `json:"metrics_endpoint"`
	TracingEnabled   bool            `json:"tracing_enabled"`
	ProfilingEnabled bool            `json:"profiling_enabled"`
	Alerting         *AlertingConfig `json:"alerting"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	EncryptionEnabled bool              `json:"encryption_enabled"`
	APIKeys           map[string]string `json:"api_keys"`
	Certificates      map[string]string `json:"certificates"`
	AllowedHosts      []string          `json:"allowed_hosts"`
	RateLimiting      *RateLimitConfig  `json:"rate_limiting"`
	Authentication    *AuthConfig       `json:"authentication"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled   bool  `json:"enabled"`
	Requests  int   `json:"requests"`
	Window    int64 `json:"window"`
	BurstSize int   `json:"burst_size"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Type           string                 `json:"type"`
	Providers      []string               `json:"providers"`
	TokenExpiry    int64                  `json:"token_expiry"`
	RefreshEnabled bool                   `json:"refresh_enabled"`
	Scope          []string               `json:"scope"`
	Claims         map[string]interface{} `json:"claims"`
}

// PerformanceConfig contains performance configuration
type PerformanceConfig struct {
	CacheEnabled   bool   `json:"cache_enabled"`
	CacheSize      int    `json:"cache_size"`
	CacheTTL       int64  `json:"cache_ttl"`
	BatchSize      int    `json:"batch_size"`
	MaxConcurrency int    `json:"max_concurrency"`
	Timeout        int64  `json:"timeout"`
	RetryPolicy    string `json:"retry_policy"`
}

// NewConfigManager creates a new config manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
		logger:     logging.NewLogger("config_manager"),
	}
}

// LoadConfig loads provider configuration from file
func (cm *ConfigManager) LoadConfig() error {
	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		cm.logger.Info("Config file not found, creating default config")
		return cm.createDefaultConfig()
	}

	// Read config file
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config
	config := &ProviderConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if err := cm.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cm.config = config
	cm.logger.Info("Configuration loaded successfully",
		"providers", len(config.Providers),
		"default_provider", config.DefaultProvider)

	return nil
}

// SaveConfig saves provider configuration to file
func (cm *ConfigManager) SaveConfig() error {
	if cm.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	// Validate config before saving
	if err := cm.validateConfig(cm.config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	cm.logger.Info("Configuration saved successfully", "path", cm.configPath)
	return nil
}

// createDefaultConfig creates a default configuration
func (cm *ConfigManager) createDefaultConfig() error {
	cm.config = &ProviderConfig{
		DefaultProvider: "pinecone",
		Providers: map[string]*SingleProviderConfig{
			"pinecone": {
				Type:     memory.ProviderTypePinecone,
				Enabled:  true,
				Priority: 1,
				Tags:     []string{"production", "primary"},
				Config: map[string]interface{}{
					"environment": "us-west1-gcp",
					"index_name":  "vectors",
					"dimension":   1536,
					"metric":      "cosine",
				},
				Resources: &ResourceConfig{
					MaxMemory:      1024 * 1024 * 1024, // 1GB
					MaxCPU:         0.8,
					MaxConnections: 100,
					Timeout:        30000,
					Retries:        3,
				},
				Health: &HealthConfig{
					Enabled:          true,
					Interval:         60000,
					Timeout:          10000,
					FailureThreshold: 3,
					RetryAttempts:    3,
					Alerting: &AlertingConfig{
						Enabled:    false,
						Channels:   []string{"email", "slack"},
						Conditions: []string{"unhealthy", "slow"},
						Thresholds: map[string]float64{
							"latency_ms": 1000,
							"error_rate": 0.05,
						},
						Cooldown: 300000,
					},
				},
				Backup: &ProviderBackupConfig{
					Enabled:     true,
					Schedule:    "0 2 * * *", // Daily at 2 AM
					Retention:   30,          // 30 days
					Compression: true,
					Destination: "s3://backups/",
					Include:     []string{"collections", "indexes"},
					Exclude:     []string{"cache", "logs"},
				},
			},
		},
		Manager: &ManagerConfig{
			Providers:             []ProviderConfig{},
			DefaultProvider:       "pinecone",
			LoadBalancing:         LoadBalanceRoundRobin,
			FailoverEnabled:       true,
			FailoverTimeout:       30000,
			RetryAttempts:         3,
			RetryBackoff:          1000,
			HealthCheckInterval:   60000,
			PerformanceMonitoring: true,
			CostTracking:          true,
			BackupEnabled:         true,
			BackupInterval:        86400000, // 24 hours
		},
		Backup: &BackupConfig{
			Enabled:          true,
			DefaultSchedule:  "0 2 * * *",
			DefaultRetention: 30,
			CompressionType:  "gzip",
			StorageLocation:  "/var/lib/helix/backups/",
			Encryption:       true,
		},
		Monitoring: &MonitoringConfig{
			Enabled:          true,
			MetricsInterval:  60000,
			LogLevel:         "INFO",
			MetricsEndpoint:  "http://localhost:8080/metrics",
			TracingEnabled:   false,
			ProfilingEnabled: false,
			Alerting: &AlertingConfig{
				Enabled:    false,
				Channels:   []string{"email"},
				Conditions: []string{"high_latency", "error_rate"},
				Thresholds: map[string]float64{
					"latency_ms": 2000,
					"error_rate": 0.1,
				},
				Cooldown: 600000,
			},
		},
		Security: &SecurityConfig{
			EncryptionEnabled: true,
			APIKeys: map[string]string{
				"pinecone": os.Getenv("PINECONE_API_KEY"),
				"openai":   os.Getenv("OPENAI_API_KEY"),
			},
			Certificates: map[string]string{},
			AllowedHosts: []string{"localhost", "api.pinecone.io"},
			RateLimiting: &RateLimitConfig{
				Enabled:   true,
				Requests:  1000,
				Window:    3600000,
				BurstSize: 100,
			},
			Authentication: &AuthConfig{
				Type:           "oauth2",
				Providers:      []string{"github", "google"},
				TokenExpiry:    3600,
				RefreshEnabled: true,
				Scope:          []string{"read", "write"},
				Claims:         map[string]interface{}{},
			},
		},
		Performance: &PerformanceConfig{
			CacheEnabled:   true,
			CacheSize:      1000,
			CacheTTL:       300000,
			BatchSize:      100,
			MaxConcurrency: 10,
			Timeout:        30000,
			RetryPolicy:    "exponential_backoff",
		},
	}

	return cm.SaveConfig()
}

// validateConfig validates the configuration
func (cm *ConfigManager) validateConfig(config *ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Validate default provider exists
	if config.DefaultProvider == "" {
		return fmt.Errorf("default provider is not specified")
	}

	if config.Providers == nil || len(config.Providers) == 0 {
		return fmt.Errorf("no providers configured")
	}

	if _, exists := config.Providers[config.DefaultProvider]; !exists {
		return fmt.Errorf("default provider %s is not configured", config.DefaultProvider)
	}

	// Validate each provider
	for name, providerConfig := range config.Providers {
		if err := cm.validateProviderConfig(name, providerConfig); err != nil {
			return fmt.Errorf("invalid config for provider %s: %w", name, err)
		}
	}

	return nil
}

// validateProviderConfig validates a single provider configuration
func (cm *ConfigManager) validateProviderConfig(name string, config *SingleProviderConfig) error {
	if config.Type == "" {
		return fmt.Errorf("provider type is not specified")
	}

	// Validate provider type exists
	registry := GetRegistry()
	providers := registry.ListProviders()

	validType := false
	for _, providerType := range providers {
		if providerType == config.Type {
			validType = true
			break
		}
	}

	if !validType {
		return fmt.Errorf("unknown provider type: %s", config.Type)
	}

	// Validate priority
	if config.Priority < 0 {
		return fmt.Errorf("priority cannot be negative")
	}

	// Validate resource config
	if config.Resources != nil {
		if config.Resources.MaxMemory <= 0 {
			return fmt.Errorf("max_memory must be positive")
		}
		if config.Resources.MaxCPU <= 0 || config.Resources.MaxCPU > 1 {
			return fmt.Errorf("max_cpu must be between 0 and 1")
		}
		if config.Resources.MaxConnections <= 0 {
			return fmt.Errorf("max_connections must be positive")
		}
	}

	// Validate health config
	if config.Health != nil {
		if config.Health.Interval <= 0 {
			return fmt.Errorf("health check interval must be positive")
		}
		if config.Health.Timeout <= 0 {
			return fmt.Errorf("health check timeout must be positive")
		}
	}

	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *ProviderConfig {
	return cm.config
}

// SetConfig sets the configuration
func (cm *ConfigManager) SetConfig(config *ProviderConfig) error {
	if err := cm.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cm.config = config
	return nil
}

// GetProviderConfig gets configuration for a specific provider
func (cm *ConfigManager) GetProviderConfig(name string) (*SingleProviderConfig, error) {
	if cm.config == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	providerConfig, exists := cm.config.Providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found in configuration", name)
	}

	return providerConfig, nil
}

// SetProviderConfig sets configuration for a specific provider
func (cm *ConfigManager) SetProviderConfig(name string, providerConfig *SingleProviderConfig) error {
	if cm.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	if err := cm.validateProviderConfig(name, providerConfig); err != nil {
		return fmt.Errorf("invalid provider configuration: %w", err)
	}

	if cm.config.Providers == nil {
		cm.config.Providers = make(map[string]*SingleProviderConfig)
	}

	cm.config.Providers[name] = providerConfig
	return cm.SaveConfig()
}

// EnableProvider enables a provider
func (cm *ConfigManager) EnableProvider(name string) error {
	return cm.setProviderEnabled(name, true)
}

// DisableProvider disables a provider
func (cm *ConfigManager) DisableProvider(name string) error {
	return cm.setProviderEnabled(name, false)
}

// setProviderEnabled sets provider enabled status
func (cm *ConfigManager) setProviderEnabled(name string, enabled bool) error {
	providerConfig, err := cm.GetProviderConfig(name)
	if err != nil {
		return err
	}

	providerConfig.Enabled = enabled
	return cm.SetProviderConfig(name, providerConfig)
}

// GetEnabledProviders returns list of enabled providers
func (cm *ConfigManager) GetEnabledProviders() []string {
	if cm.config == nil {
		return []string{}
	}

	var enabledProviders []string
	for name, providerConfig := range cm.config.Providers {
		if providerConfig.Enabled {
			enabledProviders = append(enabledProviders, name)
		}
	}

	return enabledProviders
}

// SetDefaultProvider sets the default provider
func (cm *ConfigManager) SetDefaultProvider(name string) error {
	if cm.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	_, exists := cm.config.Providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found in configuration", name)
	}

	cm.config.DefaultProvider = name
	return cm.SaveConfig()
}

// GetDefaultProvider returns the default provider name
func (cm *ConfigManager) GetDefaultProvider() string {
	if cm.config == nil {
		return ""
	}
	return cm.config.DefaultProvider
}

// ReloadConfig reloads configuration from file
func (cm *ConfigManager) ReloadConfig() error {
	cm.logger.Info("Reloading configuration")
	return cm.LoadConfig()
}

// ExportConfig exports configuration to JSON string
func (cm *ConfigManager) ExportConfig() (string, error) {
	if cm.config == nil {
		return "", fmt.Errorf("no configuration to export")
	}

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to export configuration: %w", err)
	}

	return string(data), nil
}

// ImportConfig imports configuration from JSON string
func (cm *ConfigManager) ImportConfig(configJSON string) error {
	config := &ProviderConfig{}
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		return fmt.Errorf("failed to import configuration: %w", err)
	}

	return cm.SetConfig(config)
}

// ValidateProviderConfigJSON validates provider configuration from JSON
func (cm *ConfigManager) ValidateProviderConfigJSON(configJSON string) error {
	providerConfig := &SingleProviderConfig{}
	if err := json.Unmarshal([]byte(configJSON), providerConfig); err != nil {
		return fmt.Errorf("failed to parse provider configuration: %w", err)
	}

	return cm.validateProviderConfig("test", providerConfig)
}

// MergeConfigs merges two configurations
func (cm *ConfigManager) MergeConfigs(base, override *ProviderConfig) *ProviderConfig {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := &ProviderConfig{
		DefaultProvider: override.DefaultProvider,
		Providers:       make(map[string]*SingleProviderConfig),
		Manager:         override.Manager,
		Backup:          override.Backup,
		Monitoring:      override.Monitoring,
		Security:        override.Security,
		Performance:     override.Performance,
	}

	// Merge providers
	for name, config := range base.Providers {
		result.Providers[name] = config
	}
	for name, config := range override.Providers {
		result.Providers[name] = config
	}

	// Use override values if they exist
	if override.DefaultProvider != "" {
		result.DefaultProvider = override.DefaultProvider
	}
	if override.Manager != nil {
		result.Manager = override.Manager
	}
	if override.Backup != nil {
		result.Backup = override.Backup
	}
	if override.Monitoring != nil {
		result.Monitoring = override.Monitoring
	}
	if override.Security != nil {
		result.Security = override.Security
	}
	if override.Performance != nil {
		result.Performance = override.Performance
	}

	return result
}

// CreateProviderTemplate creates a configuration template for a provider type
func (cm *ConfigManager) CreateProviderTemplate(providerType memory.ProviderType) (*SingleProviderConfig, error) {
	registry := GetRegistry()
	defaultConfig := registry.GetDefaultConfig(providerType)

	return &SingleProviderConfig{
		Type:     providerType,
		Enabled:  false,
		Priority: 0,
		Tags:     []string{},
		Config:   defaultConfig,
		Resources: &ResourceConfig{
			MaxMemory:      512 * 1024 * 1024, // 512MB
			MaxCPU:         0.5,
			MaxConnections: 50,
			Timeout:        30000,
			Retries:        3,
		},
		Health: &HealthConfig{
			Enabled:          true,
			Interval:         60000,
			Timeout:          10000,
			FailureThreshold: 3,
			RetryAttempts:    3,
		},
		Backup: &ProviderBackupConfig{
			Enabled:     false,
			Schedule:    "0 2 * * *",
			Retention:   7,
			Compression: true,
		},
	}, nil
}
