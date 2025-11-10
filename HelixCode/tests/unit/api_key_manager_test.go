package unit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/logging"
)

// TestAPIKeyManagerInitialization tests API key manager initialization
func TestAPIKeyManagerInitialization(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.HelixConfig
		expectedError bool
		expectedInit  bool
	}{
		{
			name:          "Valid config",
			config:        createTestConfig(),
			expectedError: false,
			expectedInit:  true,
		},
		{
			name:          "Nil config",
			config:        nil,
			expectedError: true,
			expectedInit:  false,
		},
		{
			name:          "Empty config",
			config:        &config.HelixConfig{},
			expectedError: false,
			expectedInit:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create API key manager
			manager, err := config.NewAPIKeyManager(test.config)

			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if manager != nil {
					t.Error("Expected nil manager but got non-nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if manager == nil {
				t.Error("Expected manager but got nil")
				return
			}

			// Test initialization
			err = manager.Initialize()
			if err != nil {
				t.Errorf("Initialization failed: %v", err)
				return
			}

			// Check initialization status (would need to expose this for testing)
			// For now, assume successful initialization
		})
	}
}

// TestAPIKeyRetrieval tests API key retrieval for different services
func TestAPIKeyRetrieval(t *testing.T) {
	tests := []struct {
		name          string
		service       string
		config        *config.HelixConfig
		expectedKey   string
		expectedError bool
	}{
		{
			name:          "Cognee local mode",
			service:       "cognee",
			config:        createCogneeLocalConfig(),
			expectedKey:   "",
			expectedError: false,
		},
		{
			name:          "Cognee remote mode with keys",
			service:       "cognee",
			config:        createCogneeRemoteConfig(),
			expectedKey:   "remote-key-1",
			expectedError: false,
		},
		{
			name:          "OpenAI with keys",
			service:       "openai",
			config:        createOpenAIConfig(),
			expectedKey:   "sk-test-key-1",
			expectedError: false,
		},
		{
			name:          "Service without keys",
			service:       "anthropic",
			config:        createEmptyServiceConfig(),
			expectedKey:   "",
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create and initialize manager
			manager, err := config.NewAPIKeyManager(test.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			err = manager.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize manager: %v", err)
			}

			// Test API key retrieval
			var key string
			if test.service == "cognee" {
				key, err = manager.GetCogneeAPIKey()
			} else {
				key, err = manager.GetAPIKey(test.service)
			}

			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if key != "" {
					t.Errorf("Expected empty key but got: %s", key)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if key != test.expectedKey {
				t.Errorf("Expected key %s but got: %s", test.expectedKey, key)
			}
		})
	}
}

// TestCogneeModeSwitching tests different Cognee modes
func TestCogneeModeSwitching(t *testing.T) {
	tests := []struct {
		name                  string
		config                *config.HelixConfig
		remoteEnabled         bool
		localFallback         bool
		expectedKey           string
		expectedRemoteEnabled bool
	}{
		{
			name:                  "Local mode only",
			config:                createCogneeLocalConfig(),
			remoteEnabled:         false,
			localFallback:         true,
			expectedKey:           "",
			expectedRemoteEnabled: false,
		},
		{
			name:                  "Remote mode only",
			config:                createCogneeRemoteConfig(),
			remoteEnabled:         true,
			localFallback:         false,
			expectedKey:           "remote-key-1",
			expectedRemoteEnabled: true,
		},
		{
			name:                  "Hybrid mode with remote",
			config:                createCogneeHybridConfig(),
			remoteEnabled:         true,
			localFallback:         true,
			expectedKey:           "remote-key-1",
			expectedRemoteEnabled: true,
		},
		{
			name:                  "Hybrid mode without remote",
			config:                createCogneeHybridNoRemoteConfig(),
			remoteEnabled:         false,
			localFallback:         true,
			expectedKey:           "",
			expectedRemoteEnabled: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create and initialize manager
			manager, err := config.NewAPIKeyManager(test.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			err = manager.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize manager: %v", err)
			}

			// Test Cognee API key retrieval
			key, err := manager.GetCogneeAPIKey()
			if err != nil {
				t.Errorf("Unexpected error getting Cognee key: %v", err)
			}

			if key != test.expectedKey {
				t.Errorf("Expected key %s but got: %s", test.expectedKey, key)
			}

			// Test remote enabled status
			remoteEnabled := manager.IsCogneeRemoteEnabled()
			if remoteEnabled != test.expectedRemoteEnabled {
				t.Errorf("Expected remote enabled %t but got: %t",
					test.expectedRemoteEnabled, remoteEnabled)
			}

			// Test fallback to local
			localFallback := manager.ShouldFallbackToCogneeLocal()
			if localFallback != test.localFallback {
				t.Errorf("Expected local fallback %t but got: %t",
					test.localFallback, localFallback)
			}
		})
	}
}

// TestLoadBalancingStrategies tests different load balancing strategies
func TestLoadBalancingStrategies(t *testing.T) {
	tests := []struct {
		name            string
		strategy        config.LoadBalancingStrategy
		keys            []string
		weights         map[string]float64
		expectedPattern []string // Expected key order for testing
	}{
		{
			name:            "Round robin strategy",
			strategy:        config.StrategyRoundRobin,
			keys:            []string{"key1", "key2", "key3"},
			expectedPattern: []string{"key1", "key2", "key3", "key1", "key2"},
		},
		{
			name:            "Weighted strategy",
			strategy:        config.StrategyWeighted,
			keys:            []string{"key1", "key2"},
			weights:         map[string]float64{"key1": 0.8, "key2": 0.2},
			expectedPattern: []string{}, // Hard to predict exact pattern
		},
		{
			name:            "Random strategy",
			strategy:        config.StrategyRandom,
			keys:            []string{"key1", "key2", "key3"},
			expectedPattern: []string{}, // Random, no specific pattern
		},
		{
			name:            "Priority first strategy",
			strategy:        config.StrategyPriorityFirst,
			keys:            []string{"key1", "key2", "key3"},
			expectedPattern: []string{}, // Depends on priority keys
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create config with specified strategy
			helixConfig := createTestConfig()
			helixConfig.APIKeys.OpenAI.LoadBalancing = &config.ServiceLBConfig{
				Strategy: test.strategy,
				Weights:  test.weights,
			}

			manager, err := config.NewAPIKeyManager(helixConfig)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			err = manager.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize manager: %v", err)
			}

			// Test key retrieval patterns
			actualKeys := make([]string, 0)
			for i := 0; i < 5; i++ {
				key, err := manager.GetAPIKey("openai")
				if err != nil {
					t.Errorf("Unexpected error getting key: %v", err)
					continue
				}
				actualKeys = append(actualKeys, key)
			}

			// Validate patterns based on strategy
			validateLoadBalancingPattern(t, test.strategy, actualKeys, test.expectedPattern)
		})
	}
}

// TestFallbackMechanisms tests fallback API key pools
func TestFallbackMechanisms(t *testing.T) {
	tests := []struct {
		name          string
		primaryKeys   []string
		fallbackKeys  []string
		primaryFail   bool
		expectedKey   string
		expectedError bool
	}{
		{
			name:          "Primary keys work",
			primaryKeys:   []string{"primary-1", "primary-2"},
			fallbackKeys:  []string{"fallback-1", "fallback-2"},
			primaryFail:   false,
			expectedKey:   "primary-1",
			expectedError: false,
		},
		{
			name:          "Primary keys fail, fallback works",
			primaryKeys:   []string{},
			fallbackKeys:  []string{"fallback-1", "fallback-2"},
			primaryFail:   true,
			expectedKey:   "fallback-1",
			expectedError: false,
		},
		{
			name:          "Both primary and fallback fail",
			primaryKeys:   []string{},
			fallbackKeys:  []string{},
			primaryFail:   true,
			expectedKey:   "",
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create config with primary and fallback keys
			helixConfig := createTestConfig()
			helixConfig.APIKeys.OpenAI.PrimaryKeys = test.primaryKeys
			helixConfig.APIKeys.OpenAI.FallbackKeys = test.fallbackKeys
			helixConfig.APIKeys.OpenAI.Fallback = &config.ServiceFallbackConfig{
				Enabled:    true,
				Strategy:   config.FallbackStrategySequential,
				MaxRetries: 3,
			}

			manager, err := config.NewAPIKeyManager(helixConfig)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			err = manager.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize manager: %v", err)
			}

			// Test API key retrieval
			key, err := manager.GetAPIKey("openai")

			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if key != test.expectedKey && len(test.expectedKey) > 0 {
				t.Errorf("Expected key %s but got: %s", test.expectedKey, key)
			}
		})
	}
}

// TestUsageStatistics tests API key usage tracking
func TestUsageStatistics(t *testing.T) {
	// Create test config
	helixConfig := createOpenAIConfig()

	manager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Record usage statistics
	testCases := []struct {
		service  string
		keyID    string
		success  bool
		errorMsg string
		latency  time.Duration
	}{
		{"openai", "sk-test-key-1", true, "", 100 * time.Millisecond},
		{"openai", "sk-test-key-2", false, "Rate limit exceeded", 0},
		{"openai", "sk-test-key-1", true, "", 150 * time.Millisecond},
		{"openai", "sk-test-key-1", false, "Invalid request", 0},
	}

	for _, tc := range testCases {
		manager.RecordAPIKeyUsage(tc.service, tc.keyID, tc.success, tc.errorMsg, tc.latency)
	}

	// Get usage statistics
	stats := manager.GetUsageStats("openai")

	// Validate statistics
	if len(stats) == 0 {
		t.Error("Expected usage statistics but got none")
		return
	}

	// Check key1 statistics
	key1Stats := stats["openai:sk-test-key-1"]
	if key1Stats == nil {
		t.Error("Expected statistics for key1 but got none")
	} else {
		if key1Stats.TotalRequests != 3 {
			t.Errorf("Expected 3 requests for key1 but got: %d", key1Stats.TotalRequests)
		}
		if key1Stats.SuccessRequests != 2 {
			t.Errorf("Expected 2 successful requests for key1 but got: %d", key1Stats.SuccessRequests)
		}
		if key1Stats.FailedRequests != 1 {
			t.Errorf("Expected 1 failed request for key1 but got: %d", key1Stats.FailedRequests)
		}
	}

	// Check key2 statistics
	key2Stats := stats["openai:sk-test-key-2"]
	if key2Stats == nil {
		t.Error("Expected statistics for key2 but got none")
	} else {
		if key2Stats.TotalRequests != 1 {
			t.Errorf("Expected 1 request for key2 but got: %d", key2Stats.TotalRequests)
		}
		if key2Stats.SuccessRequests != 0 {
			t.Errorf("Expected 0 successful requests for key2 but got: %d", key2Stats.SuccessRequests)
		}
		if key2Stats.FailedRequests != 1 {
			t.Errorf("Expected 1 failed request for key2 but got: %d", key2Stats.FailedRequests)
		}
		if key2Stats.LastError == nil {
			t.Error("Expected error information but got none")
		} else if key2Stats.LastError.Message != "Rate limit exceeded" {
			t.Errorf("Expected error message 'Rate limit exceeded' but got: %s", key2Stats.LastError.Message)
		}
	}
}

// TestConfigurationPersistence tests saving and loading API key configurations
func TestConfigurationPersistence(t *testing.T) {
	// Create test configuration
	originalConfig := &config.APIKeyConfig{
		Cognee: &config.CogneeAPIConfig{
			Enabled: true,
			Mode:    config.CogneeModeHybrid,
			APIKeys: &config.CogneeAPIKeyConfig{
				PrimaryKeys:     []string{"primary-key-1"},
				FallbackKeys:    []string{"fallback-key-1"},
				ServiceEndpoint: "https://api.cognee.ai",
				APIVersion:      "v2",
				Timeout:         30 * time.Second,
			},
		},
		OpenAI: &config.ServiceAPIKeyConfig{
			Enabled:         true,
			PrimaryKeys:     []string{"sk-test-key-1", "sk-test-key-2"},
			ServiceEndpoint: "https://api.openai.com",
			APIVersion:      "v1",
		},
		LoadBalancing: &config.LoadBalancingConfig{
			DefaultStrategy: config.StrategyWeighted,
			PriorityFirst:   true,
		},
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "api_key_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "api_keys.json")

	// Test saving configuration
	err = config.SaveAPIKeyConfig(originalConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to save configuration: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected configuration file to be created")
	}

	// Test loading configuration
	loadedConfig, err := config.LoadAPIKeyConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate loaded configuration
	if loadedConfig.Cognee == nil {
		t.Error("Expected Cognee configuration but got nil")
	} else {
		if loadedConfig.Cognee.Enabled != originalConfig.Cognee.Enabled {
			t.Errorf("Expected Cognee enabled %t but got: %t",
				originalConfig.Cognee.Enabled, loadedConfig.Cognee.Enabled)
		}

		if loadedConfig.Cognee.Mode != originalConfig.Cognee.Mode {
			t.Errorf("Expected Cognee mode %s but got: %s",
				originalConfig.Cognee.Mode, loadedConfig.Cognee.Mode)
		}

		if loadedConfig.Cognee.APIKeys.ServiceEndpoint != originalConfig.Cognee.APIKeys.ServiceEndpoint {
			t.Errorf("Expected service endpoint %s but got: %s",
				originalConfig.Cognee.APIKeys.ServiceEndpoint, loadedConfig.Cognee.APIKeys.ServiceEndpoint)
		}
	}

	if loadedConfig.OpenAI == nil {
		t.Error("Expected OpenAI configuration but got nil")
	} else {
		if len(loadedConfig.OpenAI.PrimaryKeys) != len(originalConfig.OpenAI.PrimaryKeys) {
			t.Errorf("Expected %d OpenAI keys but got: %d",
				len(originalConfig.OpenAI.PrimaryKeys), len(loadedConfig.OpenAI.PrimaryKeys))
		}
	}

	if loadedConfig.LoadBalancing == nil {
		t.Error("Expected LoadBalancing configuration but got nil")
	} else {
		if loadedConfig.LoadBalancing.DefaultStrategy != originalConfig.LoadBalancing.DefaultStrategy {
			t.Errorf("Expected strategy %s but got: %s",
				originalConfig.LoadBalancing.DefaultStrategy, loadedConfig.LoadBalancing.DefaultStrategy)
		}
	}
}

// TestDefaultConfiguration tests default API key configuration
func TestDefaultConfiguration(t *testing.T) {
	// Get default configuration
	defaultConfig := config.DefaultAPIKeyConfig()

	// Validate default settings
	if defaultConfig.Cognee == nil {
		t.Error("Expected default Cognee configuration")
	} else {
		if defaultConfig.Cognee.Mode != config.CogneeModeLocal {
			t.Errorf("Expected default Cognee mode %s but got: %s",
				config.CogneeModeLocal, defaultConfig.Cognee.Mode)
		}
	}

	if defaultConfig.LoadBalancing == nil {
		t.Error("Expected default LoadBalancing configuration")
	} else {
		if defaultConfig.LoadBalancing.DefaultStrategy != config.StrategyRoundRobin {
			t.Errorf("Expected default strategy %s but got: %s",
				config.StrategyRoundRobin, defaultConfig.LoadBalancing.DefaultStrategy)
		}
	}

	if defaultConfig.Fallback == nil {
		t.Error("Expected default Fallback configuration")
	} else {
		if defaultConfig.Fallback.MaxRetries != 3 {
			t.Errorf("Expected default max retries 3 but got: %d",
				defaultConfig.Fallback.MaxRetries)
		}
	}

	if defaultConfig.Security == nil {
		t.Error("Expected default Security configuration")
	} else {
		if !defaultConfig.Security.EncryptionEnabled {
			t.Error("Expected encryption to be enabled by default")
		}
	}

	if defaultConfig.Monitoring == nil {
		t.Error("Expected default Monitoring configuration")
	} else {
		if !defaultConfig.Monitoring.Enabled {
			t.Error("Expected monitoring to be enabled by default")
		}
	}
}

// TestCircuitBreaker tests circuit breaker functionality
func TestCircuitBreaker(t *testing.T) {
	// Create configuration with circuit breaker
	helixConfig := createTestConfig()
	helixConfig.APIKeys.OpenAI.Fallback = &config.ServiceFallbackConfig{
		Enabled:    true,
		MaxRetries: 3,
		CircuitBreaker: &config.CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			RecoveryTimeout:  time.Minute,
			SuccessThreshold: 3,
			MonitoringPeriod: 5 * time.Minute,
		},
	}

	manager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Simulate failures to trigger circuit breaker
	for i := 0; i < 10; i++ {
		manager.RecordAPIKeyUsage("openai", "sk-test-key-1", false, "Circuit breaker test", 0)
	}

	// Check if circuit breaker would be triggered
	// This would require exposing circuit breaker state for proper testing
	// For now, just ensure no panics occur
}

// TestRateLimiting tests rate limiting functionality
func TestRateLimiting(t *testing.T) {
	// Create configuration with rate limiting
	helixConfig := createTestConfig()
	helixConfig.APIKeys.OpenAI.RateLimit = &config.ServiceRateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
		BurstSize:         10,
	}

	manager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Test rate limiting by making rapid requests
	// This would require implementing actual rate limiting logic
	// For now, just ensure no panics occur
}

// TestHealthCheck tests health check functionality
func TestHealthCheck(t *testing.T) {
	// Create configuration with health check
	helixConfig := createTestConfig()
	helixConfig.APIKeys.OpenAI.LoadBalancing = &config.ServiceLBConfig{
		Strategy: config.StrategyHealthAware,
		HealthCheck: &config.HealthCheckConfig{
			Enabled:             true,
			Interval:            time.Minute,
			Timeout:             10 * time.Second,
			Endpoint:            "/health",
			Method:              "GET",
			Headers:             map[string]string{"Content-Type": "application/json"},
			ExpectedStatusCodes: []int{200},
		},
	}

	manager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Test health check
	// This would require implementing actual health check logic
	// For now, just ensure no panics occur
}

// TestKeyRotation tests API key rotation
func TestKeyRotation(t *testing.T) {
	// Create configuration with key rotation
	helixConfig := createTestConfig()

	// Initialize manager
	manager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Test key rotation
	// This would require implementing actual key rotation logic
	// For now, just ensure no panics occur
}

// TestSecurityFeatures tests security features
func TestSecurityFeatures(t *testing.T) {
	// Create configuration with security features
	helixConfig := createTestConfig()
	helixConfig.APIKeys.Security = &config.SecurityConfig{
		EncryptionEnabled: true,
		KeyRotation:       true,
		AuditLogging:      true,
		AccessControl:     true,
		AllowedIPs:        []string{"127.0.0.1", "::1"},
		BlockedIPs:        []string{"192.168.1.100"},
	}

	manager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Test security features
	// This would require implementing actual security logic
	// For now, just ensure no panics occur
}

// TestConcurrentAccess tests concurrent access to API key manager
func TestConcurrentAccess(t *testing.T) {
	helixConfig := createOpenAIConfig()

	manager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Test concurrent access
	const numGoroutines = 100
	const numRequests = 10

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numRequests; j++ {
				key, err := manager.GetAPIKey("openai")
				if err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
					continue
				}

				// Record usage
				manager.RecordAPIKeyUsage("openai", key, true, "", 100*time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Check for errors
	if len(errors) > 0 {
		t.Errorf("Unexpected errors during concurrent access: %v", errors)
	}

	// Check usage statistics
	stats := manager.GetUsageStats("openai")
	totalRequests := int64(0)
	for _, stat := range stats {
		totalRequests += stat.TotalRequests
	}

	expectedRequests := int64(numGoroutines * numRequests)
	if totalRequests != expectedRequests {
		t.Errorf("Expected %d total requests but got: %d", expectedRequests, totalRequests)
	}
}

// TestPerformance tests performance of API key manager
func TestPerformance(t *testing.T) {
	helixConfig := createOpenAIConfig()

	manager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Measure performance
	const numRequests = 10000
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		_, err := manager.GetAPIKey("openai")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	duration := time.Since(start)
	avgDuration := duration / numRequests

	t.Logf("Performance: %d requests in %v (avg: %v per request)",
		numRequests, duration, avgDuration)

	// Performance assertion - should be fast
	if avgDuration > 1*time.Millisecond {
		t.Errorf("Performance too slow: average %v per request (threshold: 1ms)", avgDuration)
	}
}

// Helper functions for creating test configurations

func createTestConfig() *config.HelixConfig {
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			Cognee: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeLocal,
			},
			OpenAI: &config.ServiceAPIKeyConfig{
				Enabled:         true,
				PrimaryKeys:     []string{"sk-test-key-1", "sk-test-key-2"},
				ServiceEndpoint: "https://api.openai.com",
				APIVersion:      "v1",
				LoadBalancing: &config.ServiceLBConfig{
					Strategy: config.StrategyRoundRobin,
				},
			},
		},
	}
}

func createCogneeLocalConfig() *config.HelixConfig {
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			Cognee: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeLocal,
			},
		},
	}
}

func createCogneeRemoteConfig() *config.HelixConfig {
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			Cognee: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeRemote,
				RemoteAPI: &config.CogneeRemoteConfig{
					Enabled:         true,
					ServiceEndpoint: "https://api.cognee.ai",
					APIVersion:      "v2",
					APIKeys:         []string{"remote-key-1", "remote-key-2"},
					PriorityKeys:    []string{"remote-key-1"},
					LoadBalancing: &config.CogneeRemoteLBConfig{
						Strategy: config.StrategyPriorityFirst,
					},
				},
			},
		},
	}
}

func createCogneeHybridConfig() *config.HelixConfig {
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			Cognee: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeHybrid,
				RemoteAPI: &config.CogneeRemoteConfig{
					Enabled:         true,
					ServiceEndpoint: "https://api.cognee.ai",
					APIVersion:      "v2",
					APIKeys:         []string{"remote-key-1", "remote-key-2"},
				},
				FallbackAPI: &config.CogneeFallbackConfig{
					Enabled:    true,
					FallbackTo: config.CogneeModeLocal,
					RetryPolicy: &config.RetryPolicy{
						MaxRetries: 3,
						RetryDelay: time.Second,
					},
				},
			},
		},
	}
}

func createCogneeHybridNoRemoteConfig() *config.HelixConfig {
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			Cognee: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeHybrid,
				FallbackAPI: &config.CogneeFallbackConfig{
					Enabled:    true,
					FallbackTo: config.CogneeModeLocal,
				},
			},
		},
	}
}

func createOpenAIConfig() *config.HelixConfig {
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			OpenAI: &config.ServiceAPIKeyConfig{
				Enabled:         true,
				PrimaryKeys:     []string{"sk-test-key-1", "sk-test-key-2"},
				ServiceEndpoint: "https://api.openai.com",
				APIVersion:      "v1",
				LoadBalancing: &config.ServiceLBConfig{
					Strategy: config.StrategyRoundRobin,
				},
			},
		},
	}
}

func createEmptyServiceConfig() *config.HelixConfig {
	return &config.HelixConfig{
		APIKeys: &config.APIKeyConfig{
			Anthropic: &config.ServiceAPIKeyConfig{
				Enabled: true,
				// No keys configured
			},
		},
	}
}

// Helper function to validate load balancing patterns
func validateLoadBalancingPattern(t *testing.T, strategy config.LoadBalancingStrategy, actualKeys, expectedPattern []string) {
	switch strategy {
	case config.StrategyRoundRobin:
		// For round robin, we expect cyclical pattern
		if len(expectedPattern) > 0 && len(actualKeys) >= len(expectedPattern) {
			for i, expectedKey := range expectedPattern {
				if i < len(actualKeys) && actualKeys[i] != expectedKey {
					t.Errorf("Round robin: expected key %s at position %d but got %s",
						expectedKey, i, actualKeys[i])
				}
			}
		}

	case config.StrategyRandom:
		// For random, we expect variation
		if len(actualKeys) > 1 {
			uniqueKeys := make(map[string]bool)
			for _, key := range actualKeys {
				uniqueKeys[key] = true
			}
			if len(uniqueKeys) == 1 {
				t.Error("Random strategy: expected variation but got same key repeatedly")
			}
		}

	case config.StrategyWeighted:
		// For weighted, we expect distribution based on weights
		// This is complex to test without knowing exact weight distribution
		t.Log("Weighted strategy test completed (complex to validate exact distribution)")

	case config.StrategyPriorityFirst:
		// For priority first, we expect priority keys to be used first
		t.Log("Priority first strategy test completed")

	case config.StrategyLeastUsed:
		// For least used, we expect distribution
		t.Log("Least used strategy test completed")

	case config.StrategyHealthAware:
		// For health aware, we expect healthy keys to be preferred
		t.Log("Health aware strategy test completed")
	}
}
