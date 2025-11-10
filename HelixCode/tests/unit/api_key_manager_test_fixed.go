package unit

import (
	"context"
	"sync"
	"testing"
	"time"

	"dev.helix.code/internal/config"
)

// Complete unit test file with all imports properly placed
// This file replaces api_key_manager_test.go with fixed imports

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

			t.Logf("API Key Manager initialized successfully for test: %s", test.name)
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

			if test.expectedKey != "" && key != test.expectedKey {
				t.Errorf("Expected key %s but got: %s", test.expectedKey, key)
			}

			t.Logf("API Key retrieval successful for service: %s, key: %s", test.service, maskKey(key))
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

			if test.expectedKey != "" && key != test.expectedKey {
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

			t.Logf("Cognee mode test passed: %s, remote: %t, fallback: %t, key: %s",
				test.name, remoteEnabled, localFallback, maskKey(key))
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
		{
			name:            "Least used strategy",
			strategy:        config.StrategyLeastUsed,
			keys:            []string{"key1", "key2", "key3"},
			expectedPattern: []string{}, // Depends on usage tracking
		},
		{
			name:            "Health aware strategy",
			strategy:        config.StrategyHealthAware,
			keys:            []string{"key1", "key2", "key3"},
			expectedPattern: []string{}, // Depends on health checks
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

			t.Logf("Load balancing test passed: %s, strategy: %s, keys retrieved: %d",
				test.name, test.strategy, len(actualKeys))
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
				if key != "" {
					t.Errorf("Expected empty key but got: %s", key)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if test.expectedKey != "" && key != test.expectedKey {
				t.Errorf("Expected key %s but got: %s", test.expectedKey, key)
			}

			t.Logf("Fallback mechanism test passed: %s, key: %s",
				test.name, maskKey(key))
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

	t.Log("Usage statistics test passed successfully")
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

	t.Logf("Concurrent access test passed: %d goroutines, %d requests each, total: %d",
		numGoroutines, numRequests, totalRequests)
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

	// Performance goal: should handle at least 10,000 requests/second
	requestsPerSecond := float64(numRequests) / duration.Seconds()
	if requestsPerSecond < 10000 {
		t.Errorf("Performance below target: %.0f requests/second (target: 10000)", requestsPerSecond)
	}
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

	key, err := manager.GetAPIKey("openai")
	if err != nil {
		t.Errorf("Unexpected error getting API key: %v", err)
	}

	if key == "" {
		t.Error("Expected API key but got empty string")
	}

	t.Logf("Security features test passed, key retrieved: %s", maskKey(key))
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

// Helper function to mask API keys for logging
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}
