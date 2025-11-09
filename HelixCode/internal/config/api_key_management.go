package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// APIKeyManager manages API keys with pools and fallback mechanisms
type APIKeyManager struct {
	config          *HelixConfig
	logger          Logger
	mu              sync.RWMutex
	keyPools        map[string]*APIKeyPool
	fallbackPools   map[string]*APIKeyPool
	usageStats      map[string]*APIKeyUsageStats
	rotationPolicies map[string]*RotationPolicy
	initialized     bool
}

// APIKeyPool represents a pool of API keys for load balancing
type APIKeyPool struct {
	Service        string              `json:"service"`
	Keys           []string            `json:"keys"`
	PriorityKeys   []string            `json:"priority_keys,omitempty"`
	CurrentIndex   int                 `json:"current_index"`
	Strategy       LoadBalancingStrategy `json:"strategy"`
	Weights        map[string]float64   `json:"weights,omitempty"`
	RateLimits     map[string]int64     `json:"rate_limits,omitempty"`
	RateLimitReset time.Time           `json:"rate_limit_reset"`
	Mu             sync.RWMutex         `json:"-"`
	Initialized    bool                 `json:"-"`
}

// FallbackPool represents fallback API key pools
type FallbackPool struct {
	Service        string                   `json:"service"`
	Pools          []string                 `json:"pools"`           // Pool identifiers
	Strategy       FallbackStrategy         `json:"strategy"`
	Timeout        time.Duration            `json:"timeout"`
	MaxRetries     int                      `json:"max_retries"`
	RetryDelay     time.Duration            `json:"retry_delay"`
	CircuitBreaker *CircuitBreakerConfig    `json:"circuit_breaker,omitempty"`
	Mu             sync.RWMutex             `json:"-"`
}

// APIKeyUsageStats tracks API key usage statistics
type APIKeyUsageStats struct {
	Service        string                    `json:"service"`
	KeyID          string                    `json:"key_id"`
	TotalRequests  int64                     `json:"total_requests"`
	SuccessRequests int64                    `json:"success_requests"`
	FailedRequests int64                     `json:"failed_requests"`
	LastError      *ErrorInfo                `json:"last_error,omitempty"`
	LastSuccess    time.Time                 `json:"last_success"`
	LastFailure    time.Time                 `json:"last_failure"`
	AverageLatency time.Duration             `json:"average_latency"`
	RateLimitHits  int64                     `json:"rate_limit_hits"`
	Disabled       bool                      `json:"disabled"`
	DisabledUntil  time.Time                 `json:"disabled_until,omitempty"`
	DisabledReason string                    `json:"disabled_reason,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code        string        `json:"code"`
	Message     string        `json:"message"`
	Timestamp   time.Time     `json:"timestamp"`
	Retryable   bool          `json:"retryable"`
	RateLimited bool          `json:"rate_limited"`
}

// RotationPolicy defines API key rotation policies
type RotationPolicy struct {
	Service         string        `json:"service"`
	Enabled         bool          `json:"enabled"`
	Interval        time.Duration `json:"interval"`
	MaxAge          time.Duration `json:"max_age"`
	MaxRequests     int64         `json:"max_requests"`
	MaxErrors       int64         `json:"max_errors"`
	AutoRotate      bool          `json:"auto_rotate"`
	RotationHook    string        `json:"rotation_hook,omitempty"`
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	Enabled           bool          `json:"enabled"`
	FailureThreshold  int           `json:"failure_threshold"`
	RecoveryTimeout   time.Duration `json:"recovery_timeout"`
	SuccessThreshold  int           `json:"success_threshold"`
	MonitoringPeriod  time.Duration `json:"monitoring_period"`
}

// APIKeyConfig contains comprehensive API key configuration
type APIKeyConfig struct {
	// Cognee Configuration
	Cognee *CogneeAPIConfig `json:"cognee,omitempty"`
	
	// Service API Keys
	OpenAI          *ServiceAPIKeyConfig `json:"openai,omitempty"`
	Anthropic       *ServiceAPIKeyConfig `json:"anthropic,omitempty"`
	Google          *ServiceAPIKeyConfig `json:"google,omitempty"`
	Cohere          *ServiceAPIKeyConfig `json:"cohere,omitempty"`
	Replicate       *ServiceAPIKeyConfig `json:"replicate,omitempty"`
	HuggingFace     *ServiceAPIKeyConfig `json:"huggingface,omitempty"`
	Together        *ServiceAPIKeyConfig `json:"together,omitempty"`
	Perplexity      *ServiceAPIKeyConfig `json:"perplexity,omitempty"`
	DeepL           *ServiceAPIKeyConfig `json:"deepl,omitempty"`
	StabilityAI     *ServiceAPIKeyConfig `json:"stability_ai,omitempty"`
	
	// Remote Services
	CogneeRemote    *CogneeRemoteConfig  `json:"cognee_remote,omitempty"`
	
	// Global Settings
	LoadBalancing   *LoadBalancingConfig `json:"load_balancing,omitempty"`
	Fallback        *FallbackConfig      `json:"fallback,omitempty"`
	Security        *SecurityConfig       `json:"security,omitempty"`
	Monitoring      *MonitoringConfig     `json:"monitoring,omitempty"`
}

// CogneeAPIConfig contains Cognee-specific API configuration
type CogneeAPIConfig struct {
	Enabled         bool                   `json:"enabled"`
	Mode            CogneeMode             `json:"mode"`                    // "local" or "remote"
	APIKeys         *CogneeAPIKeyConfig    `json:"api_keys,omitempty"`
	RemoteAPI       *CogneeRemoteConfig    `json:"remote_api,omitempty"`
	FallbackAPI     *CogneeFallbackConfig  `json:"fallback_api,omitempty"`
	LoadBalancing   *CogneeLoadBalanceConfig `json:"load_balancing,omitempty"`
}

// CogneeMode represents Cognee operation mode
type CogneeMode string

const (
	CogneeModeLocal  CogneeMode = "local"
	CogneeModeRemote CogneeMode = "remote"
	CogneeModeHybrid CogneeMode = "hybrid"
)

// CogneeAPIKeyConfig contains Cognee API key configuration
type CogneeAPIKeyConfig struct {
	PrimaryKeys     []string `json:"primary_keys,omitempty"`
	FallbackKeys    []string `json:"fallback_keys,omitempty"`
	ServiceEndpoint string   `json:"service_endpoint,omitempty"`
	APIVersion      string   `json:"api_version,omitempty"`
	Timeout         time.Duration `json:"timeout,omitempty"`
}

// CogneeRemoteConfig contains remote Cognee configuration
type CogneeRemoteConfig struct {
	Enabled         bool                   `json:"enabled"`
	ServiceEndpoint string                 `json:"service_endpoint"`
	APIVersion      string                 `json:"api_version"`
	APIKeys         []string               `json:"api_keys"`
	PriorityKeys    []string               `json:"priority_keys,omitempty"`
	LoadBalancing   *CogneeRemoteLBConfig `json:"load_balancing,omitempty"`
	CircuitBreaker  *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
	Timeout         time.Duration          `json:"timeout"`
	RateLimit       int64                  `json:"rate_limit,omitempty"`
	WebhookURL      string                 `json:"webhook_url,omitempty"`
}

// CogneeFallbackConfig contains fallback configuration
type CogneeFallbackConfig struct {
	Enabled           bool                   `json:"enabled"`
	FallbackTo        CogneeMode             `json:"fallback_to"`         // "local" or "remote"
	RemoteFallback    *CogneeRemoteConfig    `json:"remote_fallback,omitempty"`
	LocalFallback     *CogneeLocalConfig     `json:"local_fallback,omitempty"`
	RetryPolicy       *RetryPolicy           `json:"retry_policy,omitempty"`
	CircuitBreaker    *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
}

// CogneeLoadBalanceConfig contains load balancing configuration
type CogneeLoadBalanceConfig struct {
	Strategy       LoadBalancingStrategy `json:"strategy"`
	Weights        map[string]float64   `json:"weights,omitempty"`
	PriorityKeys   []string            `json:"priority_keys,omitempty"`
	HealthCheck    *HealthCheckConfig   `json:"health_check,omitempty"`
}

// CogneeRemoteLBConfig contains remote load balancing
type CogneeRemoteLBConfig struct {
	Strategy       LoadBalancingStrategy `json:"strategy"`
	Weights        map[string]float64   `json:"weights,omitempty"`
	HealthCheck    *HealthCheckConfig   `json:"health_check,omitempty"`
}

// ServiceAPIKeyConfig contains service-specific API key configuration
type ServiceAPIKeyConfig struct {
	Enabled         bool                   `json:"enabled"`
	PrimaryKeys     []string               `json:"primary_keys,omitempty"`
	FallbackKeys    []string               `json:"fallback_keys,omitempty"`
	ServiceEndpoint string                 `json:"service_endpoint,omitempty"`
	APIVersion      string                 `json:"api_version,omitempty"`
	LoadBalancing   *ServiceLBConfig      `json:"load_balancing,omitempty"`
	Fallback        *ServiceFallbackConfig `json:"fallback,omitempty"`
	RateLimit       *ServiceRateLimitConfig `json:"rate_limit,omitempty"`
}

// ServiceLBConfig contains service load balancing
type ServiceLBConfig struct {
	Strategy       LoadBalancingStrategy `json:"strategy"`
	Weights        map[string]float64   `json:"weights,omitempty"`
	PriorityKeys   []string            `json:"priority_keys,omitempty"`
	HealthCheck    *HealthCheckConfig   `json:"health_check,omitempty"`
}

// ServiceFallbackConfig contains service fallback
type ServiceFallbackConfig struct {
	Enabled        bool                 `json:"enabled"`
	Strategy       FallbackStrategy     `json:"strategy"`
	MaxRetries     int                  `json:"max_retries"`
	RetryDelay     time.Duration        `json:"retry_delay"`
	BackoffFactor  float64              `json:"backoff_factor,omitempty"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
}

// ServiceRateLimitConfig contains rate limiting
type ServiceRateLimitConfig struct {
	Enabled         bool          `json:"enabled"`
	RequestsPerMinute int64        `json:"requests_per_minute"`
	RequestsPerHour   int64        `json:"requests_per_hour"`
	RequestsPerDay    int64        `json:"requests_per_day"`
	BurstSize         int          `json:"burst_size,omitempty"`
}

// LoadBalancingConfig contains global load balancing settings
type LoadBalancingConfig struct {
	DefaultStrategy LoadBalancingStrategy `json:"default_strategy"`
	PriorityFirst   bool                   `json:"priority_first"`
	HealthCheck     *HealthCheckConfig      `json:"health_check,omitempty"`
	Metrics         *MetricsConfig         `json:"metrics,omitempty"`
}

// FallbackConfig contains global fallback settings
type FallbackConfig struct {
	Enabled         bool                    `json:"enabled"`
	DefaultStrategy FallbackStrategy        `json:"default_strategy"`
	MaxRetries      int                     `json:"max_retries"`
	RetryDelay      time.Duration           `json:"retry_delay"`
	BackoffFactor   float64                 `json:"backoff_factor,omitempty"`
	CircuitBreaker  *CircuitBreakerConfig   `json:"circuit_breaker,omitempty"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	EncryptionEnabled bool     `json:"encryption_enabled"`
	KeyRotation      bool     `json:"key_rotation"`
	AuditLogging     bool     `json:"audit_logging"`
	AccessControl    bool     `json:"access_control"`
	AllowedIPs       []string `json:"allowed_ips,omitempty"`
	BlockedIPs       []string `json:"blocked_ips,omitempty"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled         bool          `json:"enabled"`
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod  time.Duration `json:"retention_period"`
	MetricsTypes     []string      `json:"metrics_types,omitempty"`
}

// HealthCheckConfig contains health check configuration
type HealthCheckConfig struct {
	Enabled     bool          `json:"enabled"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	Endpoint    string        `json:"endpoint,omitempty"`
	Method      string        `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	ExpectedStatusCodes []int  `json:"expected_status_codes,omitempty"`
}

// RetryPolicy contains retry policy configuration
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	BackoffFactor float64       `json:"backoff_factor,omitempty"`
	MaxDelay      time.Duration `json:"max_delay,omitempty"`
	RetryableErrors []string    `json:"retryable_errors,omitempty"`
}

// Constants
type LoadBalancingStrategy string

const (
	StrategyRoundRobin    LoadBalancingStrategy = "round_robin"
	StrategyWeighted      LoadBalancingStrategy = "weighted"
	StrategyRandom        LoadBalancingStrategy = "random"
	StrategyPriorityFirst LoadBalancingStrategy = "priority_first"
	StrategyLeastUsed     LoadBalancingStrategy = "least_used"
	StrategyHealthAware   LoadBalancingStrategy = "health_aware"
)

type FallbackStrategy string

const (
	FallbackStrategySequential FallbackStrategy = "sequential"
	FallbackStrategyRandom     FallbackStrategy = "random"
	FallbackStrategyPriority   FallbackStrategy = "priority"
	FallbackStrategyHealthAware FallbackStrategy = "health_aware"
)

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(config *HelixConfig) (*APIKeyManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	
	logger := NewLogger("api_key_manager")
	
	manager := &APIKeyManager{
		config:          config,
		logger:          logger,
		keyPools:        make(map[string]*APIKeyPool),
		fallbackPools:   make(map[string]*FallbackPool),
		usageStats:      make(map[string]*APIKeyUsageStats),
		rotationPolicies: make(map[string]*RotationPolicy),
	}
	
	return manager, nil
}

// Initialize initializes the API key manager
func (akm *APIKeyManager) Initialize() error {
	akm.mu.Lock()
	defer akm.mu.Unlock()
	
	if akm.initialized {
		return nil
	}
	
	akm.logger.Info("Initializing API Key Manager...")
	
	// Initialize Cognee API keys
	if err := akm.initializeCogneeKeys(); err != nil {
		return fmt.Errorf("failed to initialize Cognee keys: %w", err)
	}
	
	// Initialize service API keys
	if err := akm.initializeServiceKeys(); err != nil {
		return fmt.Errorf("failed to initialize service keys: %w", err)
	}
	
	// Initialize fallback pools
	if err := akm.initializeFallbackPools(); err != nil {
		return fmt.Errorf("failed to initialize fallback pools: %w", err)
	}
	
	// Initialize rotation policies
	if err := akm.initializeRotationPolicies(); err != nil {
		return fmt.Errorf("failed to initialize rotation policies: %w", err)
	}
	
	akm.initialized = true
	akm.logger.Info("API Key Manager initialized successfully")
	
	return nil
}

// GetAPIKey gets an API key for the specified service
func (akm *APIKeyManager) GetAPIKey(service string) (string, error) {
	if !akm.initialized {
		return "", fmt.Errorf("API key manager not initialized")
	}
	
	// Try primary pool first
	if pool, exists := akm.keyPools[service]; exists && pool.HasKeys() {
		key, err := pool.GetKey()
		if err == nil {
			akm.recordUsage(service, key, true, "")
			return key, nil
		}
		akm.logger.Warn("Primary pool failed, trying fallback", "service", service, "error", err)
	}
	
	// Try fallback pool
	if fallbackPool, exists := akm.fallbackPools[service]; exists {
		for _, poolID := range fallbackPool.Pools {
			if pool, exists := akm.keyPools[poolID]; exists && pool.HasKeys() {
				key, err := pool.GetKey()
				if err == nil {
					akm.logger.Info("Using fallback key", "service", service, "pool", poolID)
					akm.recordUsage(service, key, true, "")
					return key, nil
				}
				akm.logger.Warn("Fallback pool failed", "service", service, "pool", poolID, "error", err)
			}
		}
	}
	
	return "", fmt.Errorf("no API keys available for service: %s", service)
}

// GetCogneeAPIKey gets a Cognee API key based on mode
func (akm *APIKeyManager) GetCogneeAPIKey() (string, error) {
	if !akm.initialized {
		return "", fmt.Errorf("API key manager not initialized")
	}
	
	// Get Cognee configuration
	cogneeConfig := akm.config.APIKeys.Cognee
	if cogneeConfig == nil {
		return "", fmt.Errorf("Cognee configuration not found")
	}
	
	// Handle different modes
	switch cogneeConfig.Mode {
	case CogneeModeLocal:
		// Local mode - no API key needed
		return "", nil
		
	case CogneeModeRemote:
		// Remote mode - use remote API keys
		if cogneeConfig.RemoteAPI != nil && cogneeConfig.RemoteAPI.Enabled {
			return akm.GetAPIKey("cognee_remote")
		}
		return "", fmt.Errorf("remote Cognee API not configured")
		
	case CogneeModeHybrid:
		// Hybrid mode - try remote first, fallback to local
		if cogneeConfig.RemoteAPI != nil && cogneeConfig.RemoteAPI.Enabled {
			if key, err := akm.GetAPIKey("cognee_remote"); err == nil && key != "" {
				return key, nil
			}
			akm.logger.Info("Remote Cognee failed, using local mode")
		}
		return "", nil
		
	default:
		return "", fmt.Errorf("unknown Cognee mode: %s", cogneeConfig.Mode)
	}
}

// IsCogneeRemoteEnabled checks if remote Cognee is enabled
func (akm *APIKeyManager) IsCogneeRemoteEnabled() bool {
	cogneeConfig := akm.config.APIKeys.Cognee
	if cogneeConfig == nil {
		return false
	}
	
	switch cogneeConfig.Mode {
	case CogneeModeRemote:
		return cogneeConfig.RemoteAPI != nil && cogneeConfig.RemoteAPI.Enabled
	case CogneeModeHybrid:
		return cogneeConfig.RemoteAPI != nil && cogneeConfig.RemoteAPI.Enabled
	default:
		return false
	}
}

// ShouldFallbackToCogneeLocal determines if fallback to local Cognee should occur
func (akm *APIKeyManager) ShouldFallbackToCogneeLocal() bool {
	cogneeConfig := akm.config.APIKeys.Cognee
	if cogneeConfig == nil {
		return true // Default to local
	}
	
	if cogneeConfig.FallbackAPI == nil || !cogneeConfig.FallbackAPI.Enabled {
		return false
	}
	
	return cogneeConfig.FallbackAPI.FallbackTo == CogneeModeLocal
}

// RecordAPIKeyUsage records usage statistics for an API key
func (akm *APIKeyManager) RecordAPIKeyUsage(service, keyID string, success bool, errorMsg string, latency time.Duration) {
	akm.mu.Lock()
	defer akm.mu.Unlock()
	
	stats := akm.usageStats[service+":"+keyID]
	if stats == nil {
		stats = &APIKeyUsageStats{
			Service: service,
			KeyID:   keyID,
		}
		akm.usageStats[service+":"+keyID] = stats
	}
	
	stats.TotalRequests++
	if success {
		stats.SuccessRequests++
		stats.LastSuccess = time.Now()
	} else {
		stats.FailedRequests++
		stats.LastFailure = time.Now()
		stats.LastError = &ErrorInfo{
			Code:      "api_error",
			Message:   errorMsg,
			Timestamp: time.Now(),
			Retryable: true,
		}
	}
	
	if latency > 0 {
		// Update average latency
		if stats.AverageLatency == 0 {
			stats.AverageLatency = latency
		} else {
			stats.AverageLatency = (stats.AverageLatency + latency) / 2
		}
	}
}

// GetUsageStats returns usage statistics for a service
func (akm *APIKeyManager) GetUsageStats(service string) map[string]*APIKeyUsageStats {
	akm.mu.RLock()
	defer akm.mu.RUnlock()
	
	result := make(map[string]*APIKeyUsageStats)
	for key, stats := range akm.usageStats {
		if strings.HasPrefix(key, service+":") {
			result[key] = stats
		}
	}
	
	return result
}

// GetKeyPoolStatus returns status of all key pools
func (akm *APIKeyManager) GetKeyPoolStatus() map[string]interface{} {
	akm.mu.RLock()
	defer akm.mu.RUnlock()
	
	status := make(map[string]interface{})
	
	// Primary pools
	primaryPools := make(map[string]interface{})
	for name, pool := range akm.keyPools {
		primaryPools[name] = pool.GetStatus()
	}
	status["primary_pools"] = primaryPools
	
	// Fallback pools
	fallbackPools := make(map[string]interface{})
	for name, pool := range akm.fallbackPools {
		fallbackPools[name] = pool.GetStatus()
	}
	status["fallback_pools"] = fallbackPools
	
	// Usage statistics
	status["usage_stats"] = akm.usageStats
	
	return status
}

// LoadAPIKeyConfig loads API key configuration from file
func LoadAPIKeyConfig(configPath string) (*APIKeyConfig, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultAPIKeyConfig(), nil
	}
	
	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read API key config: %w", err)
	}
	
	// Parse JSON
	var config APIKeyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse API key config: %w", err)
	}
	
	// Apply defaults
	config = config.applyDefaults()
	
	return &config, nil
}

// SaveAPIKeyConfig saves API key configuration to file
func SaveAPIKeyConfig(config *APIKeyConfig, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Convert to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API key config: %w", err)
	}
	
	// Write file
	if err := os.WriteFile(configPath, data, 0600); err != nil { // 0600 for security
		return fmt.Errorf("failed to write API key config: %w", err)
	}
	
	return nil
}

// DefaultAPIKeyConfig returns default API key configuration
func DefaultAPIKeyConfig() *APIKeyConfig {
	return &APIKeyConfig{
		Cognee: &CogneeAPIConfig{
			Enabled: true,
			Mode:    CogneeModeLocal, // Default to local mode
			FallbackAPI: &CogneeFallbackConfig{
				Enabled:     true,
				FallbackTo:  CogneeModeLocal,
				RetryPolicy: &RetryPolicy{
					MaxRetries:    3,
					RetryDelay:    time.Second,
					BackoffFactor: 2.0,
					MaxDelay:      30 * time.Second,
				},
			},
			LoadBalancing: &CogneeLoadBalanceConfig{
				Strategy: StrategyRoundRobin,
				HealthCheck: &HealthCheckConfig{
					Enabled:  true,
					Interval: time.Minute,
					Timeout:  10 * time.Second,
				},
			},
		},
		LoadBalancing: &LoadBalancingConfig{
			DefaultStrategy: StrategyRoundRobin,
			PriorityFirst:   true,
			HealthCheck: &HealthCheckConfig{
				Enabled:  true,
				Interval: time.Minute,
				Timeout:  10 * time.Second,
			},
		},
		Fallback: &FallbackConfig{
			Enabled:         true,
			DefaultStrategy:  FallbackStrategySequential,
			MaxRetries:       3,
			RetryDelay:       time.Second,
			BackoffFactor:    2.0,
			CircuitBreaker: &CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				RecoveryTimeout:  time.Minute,
				SuccessThreshold: 3,
				MonitoringPeriod: 5 * time.Minute,
			},
		},
		Security: &SecurityConfig{
			EncryptionEnabled: true,
			KeyRotation:      true,
			AuditLogging:     true,
			AccessControl:    true,
		},
		Monitoring: &MetricsConfig{
			Enabled:            true,
			CollectionInterval: time.Minute,
			RetentionPeriod:    24 * time.Hour,
			MetricsTypes:       []string{"usage", "performance", "errors"},
		},
	}
}

// Private helper methods

func (akm *APIKeyManager) initializeCogneeKeys() error {
	cogneeConfig := akm.config.APIKeys.Cognee
	if cogneeConfig == nil {
		return nil
	}
	
	// Initialize remote API keys
	if cogneeConfig.RemoteAPI != nil && cogneeConfig.RemoteAPI.Enabled {
		pool := &APIKeyPool{
			Service:      "cognee_remote",
			Keys:         cogneeConfig.RemoteAPI.APIKeys,
			PriorityKeys: cogneeConfig.RemoteAPI.PriorityKeys,
			Strategy:     StrategyRoundRobin,
			Initialized:  true,
		}
		
		if cogneeConfig.RemoteAPI.LoadBalancing != nil {
			pool.Strategy = cogneeConfig.RemoteAPI.LoadBalancing.Strategy
			pool.Weights = cogneeConfig.RemoteAPI.LoadBalancing.Weights
			pool.PriorityKeys = cogneeConfig.RemoteAPI.LoadBalancing.PriorityKeys
		}
		
		akm.keyPools["cognee_remote"] = pool
	}
	
	return nil
}

func (akm *APIKeyManager) initializeServiceKeys() error {
	// Initialize service API keys for each supported service
	services := map[string]*ServiceAPIKeyConfig{
		"openai":          akm.config.APIKeys.OpenAI,
		"anthropic":       akm.config.APIKeys.Anthropic,
		"google":          akm.config.APIKeys.Google,
		"cohere":          akm.config.APIKeys.Cohere,
		"replicate":       akm.config.APIKeys.Replicate,
		"huggingface":     akm.config.APIKeys.HuggingFace,
		"together":        akm.config.APIKeys.Together,
		"perplexity":      akm.config.APIKeys.Perplexity,
		"deepl":           akm.config.APIKeys.DeepL,
		"stability_ai":     akm.config.APIKeys.StabilityAI,
	}
	
	for serviceName, serviceConfig := range services {
		if serviceConfig != nil && serviceConfig.Enabled {
			if len(serviceConfig.PrimaryKeys) > 0 {
				pool := &APIKeyPool{
					Service:      serviceName,
					Keys:         serviceConfig.PrimaryKeys,
					PriorityKeys: serviceConfig.PriorityKeys,
					Strategy:     StrategyRoundRobin,
					Initialized:  true,
				}
				
				if serviceConfig.LoadBalancing != nil {
					pool.Strategy = serviceConfig.LoadBalancing.Strategy
					pool.Weights = serviceConfig.LoadBalancing.Weights
					pool.PriorityKeys = serviceConfig.LoadBalancing.PriorityKeys
				}
				
				akm.keyPools[serviceName] = pool
				
				// Initialize fallback keys if available
				if len(serviceConfig.FallbackKeys) > 0 {
					fallbackPool := &APIKeyPool{
						Service:      serviceName + "_fallback",
						Keys:         serviceConfig.FallbackKeys,
						Strategy:     StrategyRoundRobin,
						Initialized:  true,
					}
					akm.keyPools[serviceName+"_fallback"] = fallbackPool
					
					// Create fallback pool configuration
					fallbackConfig := &FallbackPool{
						Service:    serviceName,
						Pools:      []string{serviceName, serviceName + "_fallback"},
						Strategy:   FallbackStrategySequential,
						MaxRetries: 3,
						RetryDelay: time.Second,
					}
					
					if serviceConfig.Fallback != nil {
						fallbackConfig.Strategy = serviceConfig.Fallback.Strategy
						fallbackConfig.MaxRetries = serviceConfig.Fallback.MaxRetries
						fallbackConfig.RetryDelay = serviceConfig.Fallback.RetryDelay
						fallbackConfig.BackoffFactor = serviceConfig.Fallback.BackoffFactor
						fallbackConfig.CircuitBreaker = serviceConfig.Fallback.CircuitBreaker
					}
					
					akm.fallbackPools[serviceName] = fallbackConfig
				}
			}
		}
	}
	
	return nil
}

func (akm *APIKeyManager) initializeFallbackPools() error {
	// Initialize global fallback configuration
	globalFallback := akm.config.APIKeys.Fallback
	if globalFallback != nil && globalFallback.Enabled {
		// Apply global settings to existing fallback pools
		for _, pool := range akm.fallbackPools {
			pool.Strategy = globalFallback.DefaultStrategy
			pool.MaxRetries = globalFallback.MaxRetries
			pool.RetryDelay = globalFallback.RetryDelay
			pool.BackoffFactor = globalFallback.BackoffFactor
			pool.CircuitBreaker = globalFallback.CircuitBreaker
		}
	}
	
	return nil
}

func (akm *APIKeyManager) initializeRotationPolicies() error {
	// Initialize key rotation policies
	// This would load rotation policies from configuration
	return nil
}

func (akm *APIKeyManager) recordUsage(service, keyID string, success bool, errorMsg string) {
	stats := akm.usageStats[service+":"+keyID]
	if stats == nil {
		stats = &APIKeyUsageStats{
			Service: service,
			KeyID:   keyID,
		}
		akm.usageStats[service+":"+keyID] = stats
	}
	
	stats.TotalRequests++
	if success {
		stats.SuccessRequests++
		stats.LastSuccess = time.Now()
	} else {
		stats.FailedRequests++
		stats.LastFailure = time.Now()
		if errorMsg != "" {
			stats.LastError = &ErrorInfo{
				Code:      "api_error",
				Message:   errorMsg,
				Timestamp: time.Now(),
				Retryable: true,
			}
		}
	}
}

func (config *APIKeyConfig) applyDefaults() *APIKeyConfig {
	if config.Cognee == nil {
		config.Cognee = &CogneeAPIConfig{
			Enabled: true,
			Mode:    CogneeModeLocal,
		}
	}
	
	if config.LoadBalancing == nil {
		config.LoadBalancing = &LoadBalancingConfig{
			DefaultStrategy: StrategyRoundRobin,
			PriorityFirst:   true,
		}
	}
	
	if config.Fallback == nil {
		config.Fallback = &FallbackConfig{
			Enabled:         true,
			DefaultStrategy:  FallbackStrategySequential,
			MaxRetries:       3,
			RetryDelay:       time.Second,
			BackoffFactor:    2.0,
		}
	}
	
	if config.Security == nil {
		config.Security = &SecurityConfig{
			EncryptionEnabled: true,
			KeyRotation:      true,
			AuditLogging:     true,
			AccessControl:    true,
		}
	}
	
	if config.Monitoring == nil {
		config.Monitoring = &MetricsConfig{
			Enabled:            true,
			CollectionInterval: time.Minute,
			RetentionPeriod:    24 * time.Hour,
		}
	}
	
	return config
}

// APIKeyPool methods

func (pool *APIKeyPool) HasKeys() bool {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	return len(pool.Keys) > 0
}

func (pool *APIKeyPool) GetKey() (string, error) {
	if !pool.Initialized {
		return "", fmt.Errorf("pool not initialized")
	}
	
	pool.mu.Lock()
	defer pool.mu.Unlock()
	
	if len(pool.Keys) == 0 {
		return "", fmt.Errorf("no keys available in pool")
	}
	
	// Check rate limits
	if pool.isRateLimited() {
		return "", fmt.Errorf("pool rate limited until %v", pool.RateLimitReset)
	}
	
	var key string
	var err error
	
	// Try priority keys first
	if len(pool.PriorityKeys) > 0 {
		key, err = pool.getKeyFromList(pool.PriorityKeys)
		if err == nil {
			pool.recordKeyUsage(key)
			return key, nil
		}
	}
	
	// Use load balancing strategy
	switch pool.Strategy {
	case StrategyRoundRobin:
		key = pool.Keys[pool.CurrentIndex%len(pool.Keys)]
		pool.CurrentIndex++
	case StrategyWeighted:
		key = pool.getWeightedKey()
	case StrategyRandom:
		key = pool.Keys[time.Now().UnixNano()%int64(len(pool.Keys))]
	case StrategyPriorityFirst:
		// Already tried priority keys above
		fallthrough
	case StrategyLeastUsed:
		key = pool.getLeastUsedKey()
	case StrategyHealthAware:
		key = pool.getHealthAwareKey()
	default:
		key = pool.Keys[pool.CurrentIndex%len(pool.Keys)]
		pool.CurrentIndex++
	}
	
	if key == "" {
		return "", fmt.Errorf("no suitable key found")
	}
	
	pool.recordKeyUsage(key)
	return key, nil
}

func (pool *APIKeyPool) GetStatus() map[string]interface{} {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	
	return map[string]interface{}{
		"service":      pool.Service,
		"total_keys":   len(pool.Keys),
		"current_index": pool.CurrentIndex,
		"strategy":     pool.Strategy,
		"initialized":  pool.Initialized,
		"rate_limited": pool.isRateLimited(),
		"rate_limit_reset": pool.RateLimitReset,
	}
}

func (pool *APIKeyPool) isRateLimited() bool {
	return time.Now().Before(pool.RateLimitReset)
}

func (pool *APIKeyPool) getKeyFromList(keys []string) (string, error) {
	if len(keys) == 0 {
		return "", fmt.Errorf("no keys in list")
	}
	return keys[time.Now().UnixNano()%int64(len(keys))], nil
}

func (pool *APIKeyPool) getWeightedKey() string {
	// Implementation would use weights to select key
	// For now, use round robin
	if len(pool.Keys) == 0 {
		return ""
	}
	return pool.Keys[pool.CurrentIndex%len(pool.Keys)]
}

func (pool *APIKeyPool) getLeastUsedKey() string {
	// Implementation would track usage and select least used
	// For now, use round robin
	if len(pool.Keys) == 0 {
		return ""
	}
	return pool.Keys[pool.CurrentIndex%len(pool.Keys)]
}

func (pool *APIKeyPool) getHealthAwareKey() string {
	// Implementation would check health and select healthy key
	// For now, use round robin
	if len(pool.Keys) == 0 {
		return ""
	}
	return pool.Keys[pool.CurrentIndex%len(pool.Keys)]
}

func (pool *APIKeyPool) recordKeyUsage(key string) {
	// Implementation would record key usage
	// For now, just log
}

// FallbackPool methods

func (fp *FallbackPool) GetStatus() map[string]interface{} {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	
	return map[string]interface{}{
		"service":       fp.Service,
		"pools":         fp.Pools,
		"strategy":      fp.Strategy,
		"max_retries":   fp.MaxRetries,
		"retry_delay":   fp.RetryDelay,
		"backoff_factor": fp.BackoffFactor,
		"circuit_breaker": fp.CircuitBreaker,
	}
}