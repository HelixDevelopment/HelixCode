package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/cognee"
	"dev.helix.code/internal/provider"
	"dev.helix.code/internal/hardware"
)

// TestCogneeAPIKeyIntegration tests integration between Cognee and API key management
func TestCogneeAPIKeyIntegration(t *testing.T) {
	tests := []struct {
		name          string
		cogneeConfig  *config.CogneeAPIConfig
		apiKeyConfig  *config.APIKeyConfig
		expectedMode  string
		expectedError bool
	}{
		{
			name: "Local Cognee mode",
			cogneeConfig: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeLocal,
			},
			apiKeyConfig: &config.APIKeyConfig{
				Cognee: &config.CogneeAPIConfig{
					Enabled: true,
					Mode:    config.CogneeModeLocal,
				},
			},
			expectedMode:  "local",
			expectedError: false,
		},
		{
			name: "Remote Cognee mode with keys",
			cogneeConfig: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeRemote,
				RemoteAPI: &config.CogneeRemoteConfig{
					Enabled:         true,
					ServiceEndpoint: "https://api.cognee.ai",
					APIVersion:      "v2",
					APIKeys:         []string{"cognee-key-1", "cognee-key-2"},
				},
			},
			apiKeyConfig: &config.APIKeyConfig{
				Cognee: &config.CogneeAPIConfig{
					Enabled: true,
					Mode:    config.CogneeModeRemote,
					RemoteAPI: &config.CogneeRemoteConfig{
						Enabled:         true,
						ServiceEndpoint: "https://api.cognee.ai",
						APIVersion:      "v2",
						APIKeys:         []string{"cognee-key-1", "cognee-key-2"},
					},
				},
			},
			expectedMode:  "remote",
			expectedError: false,
		},
		{
			name: "Hybrid Cognee mode with fallback",
			cogneeConfig: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeHybrid,
				RemoteAPI: &config.CogneeRemoteConfig{
					Enabled:         true,
					ServiceEndpoint: "https://api.cognee.ai",
					APIVersion:      "v2",
					APIKeys:         []string{"cognee-key-1"},
				},
				FallbackAPI: &config.CogneeFallbackConfig{
					Enabled:     true,
					FallbackTo:  config.CogneeModeLocal,
					RetryPolicy: &config.RetryPolicy{
						MaxRetries:    3,
						RetryDelay:    time.Second,
						BackoffFactor: 2.0,
					},
				},
			},
			apiKeyConfig: &config.APIKeyConfig{
				Cognee: &config.CogneeAPIConfig{
					Enabled: true,
					Mode:    config.CogneeModeHybrid,
					RemoteAPI: &config.CogneeRemoteConfig{
						Enabled:         true,
						ServiceEndpoint: "https://api.cognee.ai",
						APIVersion:      "v2",
						APIKeys:         []string{"cognee-key-1"},
					},
					FallbackAPI: &config.CogneeFallbackConfig{
						Enabled:     true,
						FallbackTo:  config.CogneeModeLocal,
					},
				},
			},
			expectedMode:  "hybrid",
			expectedError: false,
		},
		{
			name: "Remote Cognee mode without keys",
			cogneeConfig: &config.CogneeAPIConfig{
				Enabled: true,
				Mode:    config.CogneeModeRemote,
				RemoteAPI: &config.CogneeRemoteConfig{
					Enabled: true,
					// No API keys configured
				},
			},
			apiKeyConfig: &config.APIKeyConfig{
				Cognee: &config.CogneeAPIConfig{
					Enabled: true,
					Mode:    config.CogneeModeRemote,
					RemoteAPI: &config.CogneeRemoteConfig{
						Enabled: true,
						// No API keys configured
					},
				},
			},
			expectedMode:  "remote",
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "cognee_integration_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Save API key configuration
			apiKeyConfigPath := filepath.Join(tempDir, "api_keys.json")
			err = config.SaveAPIKeyConfig(test.apiKeyConfig, apiKeyConfigPath)
			if err != nil {
				t.Fatalf("Failed to save API key config: %v", err)
			}

			// Load API key configuration
			loadedAPIKeyConfig, err := config.LoadAPIKeyConfig(apiKeyConfigPath)
			if err != nil {
				t.Fatalf("Failed to load API key config: %v", err)
			}

			// Create test Helix configuration
			helixConfig := &config.HelixConfig{
				Cognee: test.cogneeConfig,
				APIKeys: loadedAPIKeyConfig,
			}

			// Create hardware profile
			hwProfile := &hardware.Profile{
				CPU: &hardware.CPUProfile{
					Cores:        8,
					Threads:      16,
					Model:        "Test CPU",
					FrequencyGHz: 3.0,
				},
				Memory: &hardware.MemoryProfile{
					TotalGB:     32,
					AvailableGB: 24,
				},
			}

			// Create API key manager
			apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
			if err != nil {
				t.Fatalf("Failed to create API key manager: %v", err)
			}

			// Initialize API key manager
			err = apiKeyManager.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize API key manager: %v", err)
			}

			// Test Cognee mode detection
			remoteEnabled := apiKeyManager.IsCogneeRemoteEnabled()
			shouldFallback := apiKeyManager.ShouldFallbackToCogneeLocal()

			// Get Cognee API key
			key, err := apiKeyManager.GetCogneeAPIKey()

			// Validate results
			switch test.expectedMode {
			case "local":
				if remoteEnabled {
					t.Error("Expected remote disabled for local mode")
				}
				if key != "" {
					t.Errorf("Expected empty key for local mode but got: %s", maskKey(key))
				}
			case "remote":
				if !remoteEnabled {
					t.Error("Expected remote enabled for remote mode")
				}
				if test.expectedError {
					if err == nil {
						t.Error("Expected error for remote mode without keys")
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error for remote mode: %v", err)
					}
					if key == "" {
						t.Error("Expected API key for remote mode but got empty")
					}
				}
			case "hybrid":
				if test.expectedError {
					if err == nil {
						t.Error("Expected error for hybrid mode without remote keys")
					}
				} else {
					// In hybrid mode, remote should be enabled if keys are available
					hasRemoteKeys := test.cogneeConfig.RemoteAPI != nil && len(test.cogneeConfig.RemoteAPI.APIKeys) > 0
					if remoteEnabled != hasRemoteKeys {
						t.Errorf("Expected remote enabled %t for hybrid mode but got: %t", 
							hasRemoteKeys, remoteEnabled)
					}
				}
			}

			// Test API key manager status
			status := apiKeyManager.GetKeyPoolStatus()
			if status == nil {
				t.Error("Expected status but got nil")
			} else {
				t.Logf("API Key Manager Status: %+v", status)
			}

			t.Logf("Integration test passed: %s, mode: %s, remote: %t, fallback: %t", 
				test.name, test.expectedMode, remoteEnabled, shouldFallback)
		})
	}
}

// TestProviderAPIKeyIntegration tests integration between providers and API key management
func TestProviderAPIKeyIntegration(t *testing.T) {
	tests := []struct {
		name          string
		providerType  provider.ProviderType
		apiKeyConfig  *config.APIKeyConfig
		expectedKey   string
		expectedError bool
	}{
		{
			name:         "OpenAI provider with keys",
			providerType: provider.ProviderTypeOpenAI,
			apiKeyConfig: &config.APIKeyConfig{
				OpenAI: &config.ServiceAPIKeyConfig{
					Enabled:     true,
					PrimaryKeys: []string{"sk-openai-test-1", "sk-openai-test-2"},
					ServiceEndpoint: "https://api.openai.com",
					APIVersion:    "v1",
					LoadBalancing: &config.ServiceLBConfig{
						Strategy: config.StrategyRoundRobin,
					},
				},
			},
			expectedKey:   "sk-openai-test-1",
			expectedError: false,
		},
		{
			name:         "Anthropic provider with keys",
			providerType: provider.ProviderTypeAnthropic,
			apiKeyConfig: &config.APIKeyConfig{
				Anthropic: &config.ServiceAPIKeyConfig{
					Enabled:     true,
					PrimaryKeys: []string{"sk-ant-test-1", "sk-ant-test-2"],
					ServiceEndpoint: "https://api.anthropic.com",
					APIVersion:    "v1",
					LoadBalancing: &config.ServiceLBConfig{
						Strategy: config.StrategyWeighted,
						Weights: map[string]float64{
							"sk-ant-test-1": 0.7,
							"sk-ant-test-2": 0.3,
						},
					},
				},
			},
			expectedKey:   "sk-ant-test-1",
			expectedError: false,
		},
		{
			name:         "Google provider without keys",
			providerType: provider.ProviderTypeGoogle,
			apiKeyConfig: &config.APIKeyConfig{
				Google: &config.ServiceAPIKeyConfig{
					Enabled: true,
					// No keys configured
				},
			},
			expectedKey:   "",
			expectedError: true,
		},
		{
			name:         "VLLM provider (no API key needed)",
			providerType: provider.ProviderTypeVLLM,
			apiKeyConfig: &config.APIKeyConfig{
				// VLLM doesn't use API keys
			},
			expectedKey:   "",
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "provider_integration_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Save API key configuration
			apiKeyConfigPath := filepath.Join(tempDir, "api_keys.json")
			err = config.SaveAPIKeyConfig(test.apiKeyConfig, apiKeyConfigPath)
			if err != nil {
				t.Fatalf("Failed to save API key config: %v", err)
			}

			// Load API key configuration
			loadedAPIKeyConfig, err := config.LoadAPIKeyConfig(apiKeyConfigPath)
			if err != nil {
				t.Fatalf("Failed to load API key config: %v", err)
			}

			// Create test Helix configuration
			helixConfig := &config.HelixConfig{
				APIKeys: loadedAPIKeyConfig,
			}

			// Create hardware profile
			hwProfile := &hardware.Profile{
				CPU: &hardware.CPUProfile{
					Cores:        4,
					Threads:      8,
					Model:        "Test CPU",
					FrequencyGHz: 2.5,
				},
				Memory: &hardware.MemoryProfile{
					TotalGB:     16,
					AvailableGB: 12,
				},
			}

			// Create API key manager
			apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
			if err != nil {
				t.Fatalf("Failed to create API key manager: %v", err)
			}

			// Initialize API key manager
			err = apiKeyManager.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize API key manager: %v", err)
			}

			// Get provider name
			providerName := test.providerType.String()

			// Test API key retrieval
			key, err := apiKeyManager.GetAPIKey(providerName)

			// Validate results
			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if key != "" {
					t.Errorf("Expected empty key but got: %s", maskKey(key))
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				
				// Some providers don't use API keys (like VLLM)
				if test.expectedKey != "" {
					if key == "" {
						t.Errorf("Expected key but got empty")
					} else if key != test.expectedKey {
						// For load balancing, we might get different keys
						t.Logf("Got different key than expected (load balancing): %s", maskKey(key))
					}
				}
			}

			// Test usage statistics
			if key != "" {
				apiKeyManager.RecordAPIKeyUsage(providerName, key, true, "", 100*time.Millisecond)
				stats := apiKeyManager.GetUsageStats(providerName)
				if len(stats) == 0 {
					t.Error("Expected usage statistics but got none")
				} else {
					t.Logf("Usage statistics for %s: %+v", providerName, stats)
				}
			}

			t.Logf("Provider integration test passed: %s, key: %s", 
				test.name, maskKey(key))
		})
	}
}

// TestCogneeManagerAPIKeyIntegration tests Cognee manager with API key management
func TestCogneeManagerAPIKeyIntegration(t *testing.T) {
	tests := []struct {
		name          string
		helixConfig   *config.HelixConfig
		hwProfile     *hardware.Profile
		expectedInit  bool
		expectedError bool
	}{
		{
			name: "Local Cognee with API keys",
			helixConfig: &config.HelixConfig{
				Cognee: &config.CogneeConfig{
					Enabled:        true,
					AutoStart:      true,
					Host:           "localhost",
					Port:           8000,
					DynamicConfig:  true,
				},
				APIKeys: &config.APIKeyConfig{
					Cognee: &config.CogneeAPIConfig{
						Enabled: true,
						Mode:    config.CogneeModeLocal,
					},
					OpenAI: &config.ServiceAPIKeyConfig{
						Enabled:     true,
						PrimaryKeys: []string{"sk-test-key-1"},
					},
				},
			},
			hwProfile: &hardware.Profile{
				CPU: &hardware.CPUProfile{
					Cores:        8,
					Threads:      16,
					Model:        "Apple M2 Pro",
					FrequencyGHz: 3.5,
				},
				Memory: &hardware.MemoryProfile{
					TotalGB:     32,
					AvailableGB: 24,
				},
				GPUs: []*hardware.GPUProfile{
					{
						Type:      hardware.GPUTypeApple,
						Name:      "Apple M2 Pro GPU",
						VRAMGB:    19,
						Available: true,
					},
				},
			},
			expectedInit:  true,
			expectedError: false,
		},
		{
			name: "Remote Cognee with API keys",
			helixConfig: &config.HelixConfig{
				Cognee: &config.CogneeConfig{
					Enabled:        true,
					AutoStart:      false, // Don't start remote service in test
					Host:           "localhost",
					Port:           8000,
					DynamicConfig:  true,
				},
				APIKeys: &config.APIKeyConfig{
					Cognee: &config.CogneeAPIConfig{
						Enabled: true,
						Mode:    config.CogneeModeRemote,
						RemoteAPI: &config.CogneeRemoteConfig{
							Enabled:         true,
							ServiceEndpoint: "https://api.cognee.ai",
							APIVersion:      "v2",
							APIKeys:         []string{"cognee-remote-1", "cognee-remote-2"},
							LoadBalancing: &config.CogneeRemoteLBConfig{
								Strategy: config.StrategyRoundRobin,
							},
						},
					},
				},
			},
			hwProfile: &hardware.Profile{
				CPU: &hardware.CPUProfile{
					Cores:        16,
					Threads:      32,
					Model:        "Intel Xeon",
					FrequencyGHz: 3.8,
				},
				Memory: &hardware.MemoryProfile{
					TotalGB:     64,
					AvailableGB: 48,
				},
				GPUs: []*hardware.GPUProfile{
					{
						Type:      hardware.GPUTypeNVIDIA,
						Name:      "NVIDIA A100",
						VRAMGB:    80,
						Available: true,
					},
				},
			},
			expectedInit:  true,
			expectedError: false,
		},
		{
			name: "Hybrid Cognee with fallback",
			helixConfig: &config.HelixConfig{
				Cognee: &config.CogneeConfig{
					Enabled:        true,
					AutoStart:      false,
					Host:           "localhost",
					Port:           8000,
					DynamicConfig:  true,
				},
				APIKeys: &config.APIKeyConfig{
					Cognee: &config.CogneeAPIConfig{
						Enabled: true,
						Mode:    config.CogneeModeHybrid,
						RemoteAPI: &config.CogneeRemoteConfig{
							Enabled:         true,
							ServiceEndpoint: "https://api.cognee.ai",
							APIVersion:      "v2",
							APIKeys:         []string{"cognee-hybrid-1"},
						},
						FallbackAPI: &config.CogneeFallbackConfig{
							Enabled:     true,
							FallbackTo:  config.CogneeModeLocal,
							RetryPolicy: &config.RetryPolicy{
								MaxRetries:    3,
								RetryDelay:    time.Second,
								BackoffFactor: 2.0,
							},
						},
					},
				},
			},
			hwProfile: &hardware.Profile{
				CPU: &hardware.CPUProfile{
					Cores:        12,
					Threads:      24,
					Model:        "AMD Ryzen 9",
					FrequencyGHz: 4.2,
				},
				Memory: &hardware.MemoryProfile{
					TotalGB:     32,
					AvailableGB: 20,
				},
				GPUs: []*hardware.GPUProfile{
					{
						Type:      hardware.GPUTypeAMD,
						Name:      "AMD Radeon RX 6800 XT",
						VRAMGB:    16,
						Available: true,
					},
				},
			},
			expectedInit:  true,
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create API key manager
			apiKeyManager, err := config.NewAPIKeyManager(test.helixConfig)
			if err != nil {
				t.Fatalf("Failed to create API key manager: %v", err)
			}

			// Initialize API key manager
			err = apiKeyManager.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize API key manager: %v", err)
			}

			// Create Cognee manager
			cogneeManager, err := cognee.NewCogneeManager(test.helixConfig, test.hwProfile)
			if err != nil {
				t.Fatalf("Failed to create Cognee manager: %v", err)
			}

			// Initialize Cognee manager
			ctx := context.Background()
			err = cogneeManager.Initialize(ctx)
			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
			}

			// Test that Cognee manager can access API keys
			if test.expectedInit {
				// Test getting Cognee API key
				key, err := apiKeyManager.GetCogneeAPIKey()
				if err != nil {
					t.Errorf("Failed to get Cognee API key: %v", err)
				} else {
					t.Logf("Cognee API key retrieved: %s", maskKey(key))
				}

				// Test Cognee manager status
				status := cogneeManager.GetStatus()
				if status == nil {
					t.Error("Expected status but got nil")
				} else {
					t.Logf("Cognee Manager Status: %+v", status)
				}

				// Test API key manager status
				keyStatus := apiKeyManager.GetKeyPoolStatus()
				if keyStatus == nil {
					t.Error("Expected key status but got nil")
				} else {
					t.Logf("API Key Manager Status: %+v", keyStatus)
				}
			}

			// Test provider integration
			if len(test.helixConfig.APIKeys.OpenAI.PrimaryKeys) > 0 {
				openaiKey, err := apiKeyManager.GetAPIKey("openai")
				if err != nil {
					t.Errorf("Failed to get OpenAI key: %v", err)
				} else {
					t.Logf("OpenAI API key retrieved: %s", maskKey(openaiKey))
				}
			}

			t.Logf("Cognee manager integration test passed: %s", test.name)
		})
	}
}

// TestConfigurationPersistence tests configuration persistence with API keys
func TestConfigurationPersistence(t *testing.T) {
	// Create comprehensive API key configuration
	originalConfig := &config.APIKeyConfig{
		Cognee: &config.CogneeAPIConfig{
			Enabled: true,
			Mode:    config.CogneeModeHybrid,
			RemoteAPI: &config.CogneeRemoteConfig{
				Enabled:         true,
				ServiceEndpoint: "https://api.cognee.ai",
				APIVersion:      "v2",
				APIKeys:         []string{"cognee-1", "cognee-2"},
				PriorityKeys:    []string{"cognee-1"},
				LoadBalancing: &config.CogneeRemoteLBConfig{
					Strategy: config.StrategyPriorityFirst,
					Weights: map[string]float64{
						"cognee-1": 0.8,
						"cognee-2": 0.2,
					},
				},
				CircuitBreaker: &config.CircuitBreakerConfig{
					Enabled:          true,
					FailureThreshold: 5,
					RecoveryTimeout:  time.Minute,
				},
			},
			FallbackAPI: &config.CogneeFallbackConfig{
				Enabled:     true,
				FallbackTo:  config.CogneeModeLocal,
				RetryPolicy: &config.RetryPolicy{
					MaxRetries:    3,
					RetryDelay:    time.Second,
					BackoffFactor: 2.0,
				},
			},
		},
		OpenAI: &config.ServiceAPIKeyConfig{
			Enabled:     true,
			PrimaryKeys: []string{"sk-openai-1", "sk-openai-2"},
			FallbackKeys: []string{"sk-openai-fallback"},
			ServiceEndpoint: "https://api.openai.com",
			APIVersion:    "v1",
			LoadBalancing: &config.ServiceLBConfig{
				Strategy:     config.StrategyWeighted,
				Weights:      map[string]float64{"sk-openai-1": 0.7, "sk-openai-2": 0.3},
				PriorityKeys: []string{"sk-openai-1"},
			},
			Fallback: &config.ServiceFallbackConfig{
				Enabled:        true,
				Strategy:       config.FallbackStrategySequential,
				MaxRetries:     3,
				RetryDelay:     time.Second,
				BackoffFactor:  2.0,
				CircuitBreaker: &config.CircuitBreakerConfig{
					Enabled:          true,
					FailureThreshold: 3,
					RecoveryTimeout:  time.Minute * 2,
				},
			},
			RateLimit: &config.ServiceRateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 60,
				RequestsPerHour:   1000,
				RequestsPerDay:    10000,
				BurstSize:         10,
			},
		},
		Anthropic: &config.ServiceAPIKeyConfig{
			Enabled:     true,
			PrimaryKeys: []string{"sk-ant-1", "sk-ant-2"},
			ServiceEndpoint: "https://api.anthropic.com",
			APIVersion:    "v1",
			LoadBalancing: &config.ServiceLBConfig{
				Strategy: config.StrategyRoundRobin,
			},
		},
		LoadBalancing: &config.LoadBalancingConfig{
			DefaultStrategy: config.StrategyWeighted,
			PriorityFirst:   true,
			HealthCheck: &config.HealthCheckConfig{
				Enabled:  true,
				Interval: time.Minute,
				Timeout:  10 * time.Second,
			},
		},
		Fallback: &config.FallbackConfig{
			Enabled:         true,
			DefaultStrategy:  config.FallbackStrategySequential,
			MaxRetries:      3,
			RetryDelay:      time.Second,
			BackoffFactor:   2.0,
			CircuitBreaker: &config.CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				RecoveryTimeout:  time.Minute,
				SuccessThreshold: 3,
				MonitoringPeriod: 5 * time.Minute,
			},
		},
		Security: &config.SecurityConfig{
			EncryptionEnabled: true,
			KeyRotation:      true,
			AuditLogging:     true,
			AccessControl:    true,
			AllowedIPs:       []string{"127.0.0.1", "::1", "10.0.0.0/8"},
			BlockedIPs:       []string{"192.168.1.100"},
		},
		Monitoring: &config.MonitoringConfig{
			Enabled:            true,
			CollectionInterval: time.Minute,
			RetentionPeriod:    24 * time.Hour,
			MetricsTypes:       []string{"usage", "performance", "errors", "health"},
		},
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "config_persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test saving configuration
	configPath := filepath.Join(tempDir, "test_api_keys.json")
	err = config.SaveAPIKeyConfig(originalConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to save configuration: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Configuration file was not created")
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
		if loadedConfig.Cognee.Mode != originalConfig.Cognee.Mode {
			t.Errorf("Expected Cognee mode %s but got: %s", 
				originalConfig.Cognee.Mode, loadedConfig.Cognee.Mode)
		}

		if loadedConfig.Cognee.RemoteAPI == nil {
			t.Error("Expected RemoteAPI configuration but got nil")
		} else {
			if len(loadedConfig.Cognee.RemoteAPI.APIKeys) != len(originalConfig.Cognee.RemoteAPI.APIKeys) {
				t.Errorf("Expected %d remote API keys but got: %d", 
					len(originalConfig.Cognee.RemoteAPI.APIKeys), 
					len(loadedConfig.Cognee.RemoteAPI.APIKeys))
			}
		}

		if loadedConfig.Cognee.FallbackAPI == nil {
			t.Error("Expected FallbackAPI configuration but got nil")
		}
	}

	if loadedConfig.OpenAI == nil {
		t.Error("Expected OpenAI configuration but got nil")
	} else {
		if len(loadedConfig.OpenAI.PrimaryKeys) != len(originalConfig.OpenAI.PrimaryKeys) {
			t.Errorf("Expected %d OpenAI primary keys but got: %d", 
				len(originalConfig.OpenAI.PrimaryKeys), 
				len(loadedConfig.OpenAI.PrimaryKeys))
		}

		if len(loadedConfig.OpenAI.FallbackKeys) != len(originalConfig.OpenAI.FallbackKeys) {
			t.Errorf("Expected %d OpenAI fallback keys but got: %d", 
				len(originalConfig.OpenAI.FallbackKeys), 
				len(loadedConfig.OpenAI.FallbackKeys))
		}
	}

	// Test configuration round-trip
	helixConfig := &config.HelixConfig{
		APIKeys: loadedConfig,
	}

	// Create API key manager with loaded configuration
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create API key manager: %v", err)
	}

	// Initialize API key manager
	err = apiKeyManager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize API key manager: %v", err)
	}

	// Test API key retrieval with loaded configuration
	key, err := apiKeyManager.GetAPIKey("openai")
	if err != nil {
		t.Errorf("Failed to get OpenAI key: %v", err)
	} else {
		t.Logf("OpenAI key retrieved from loaded config: %s", maskKey(key))
	}

	cogneeKey, err := apiKeyManager.GetCogneeAPIKey()
	if err != nil {
		t.Errorf("Failed to get Cognee key: %v", err)
	} else {
		t.Logf("Cognee key retrieved from loaded config: %s", maskKey(cogneeKey))
	}

	// Test configuration JSON serialization
	configJSON, err := json.MarshalIndent(loadedConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal configuration: %v", err)
	}

	// Verify JSON is valid
	var unmarshaledConfig config.APIKeyConfig
	err = json.Unmarshal(configJSON, &unmarshaledConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal configuration: %v", err)
	}

	t.Logf("Configuration persistence test passed, JSON size: %d bytes", len(configJSON))
}

// TestRealWorldScenario tests a real-world scenario with multiple providers and API keys
func TestRealWorldScenario(t *testing.T) {
	// Create real-world configuration with multiple providers
	realWorldConfig := &config.APIKeyConfig{
		Cognee: &config.CogneeAPIConfig{
			Enabled: true,
			Mode:    config.CogneeModeHybrid,
			RemoteAPI: &config.CogneeRemoteConfig{
				Enabled:         true,
				ServiceEndpoint: "https://api.cognee.ai",
				APIVersion:      "v2",
				APIKeys:         []string{"cognee-prod-1", "cognee-prod-2", "cognee-prod-3"},
				PriorityKeys:    []string{"cognee-prod-1"},
				LoadBalancing: &config.CogneeRemoteLBConfig{
					Strategy: config.StrategyPriorityFirst,
				},
				CircuitBreaker: &config.CircuitBreakerConfig{
					Enabled:          true,
					FailureThreshold: 5,
					RecoveryTimeout:  time.Minute,
				},
				Timeout: 30 * time.Second,
				RateLimit: 1000,
			},
			FallbackAPI: &config.CogneeFallbackConfig{
				Enabled:     true,
				FallbackTo:  config.CogneeModeLocal,
				RetryPolicy: &config.RetryPolicy{
					MaxRetries:    5,
					RetryDelay:    2 * time.Second,
					BackoffFactor: 2.0,
					MaxDelay:      60 * time.Second,
				},
			},
		},
		OpenAI: &config.ServiceAPIKeyConfig{
			Enabled:     true,
			PrimaryKeys: []string{"sk-openai-prod-1", "sk-openai-prod-2"},
			FallbackKeys: []string{"sk-openai-backup-1"},
			ServiceEndpoint: "https://api.openai.com",
			APIVersion:    "v1",
			LoadBalancing: &config.ServiceLBConfig{
				Strategy:     config.StrategyWeighted,
				Weights:      map[string]float64{"sk-openai-prod-1": 0.8, "sk-openai-prod-2": 0.2},
				PriorityKeys: []string{"sk-openai-prod-1"},
			},
			Fallback: &config.ServiceFallbackConfig{
				Enabled:        true,
				Strategy:       config.FallbackStrategySequential,
				MaxRetries:     5,
				RetryDelay:     time.Second,
				BackoffFactor:  2.0,
			},
			RateLimit: &config.ServiceRateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 500,
				RequestsPerHour:   10000,
				RequestsPerDay:    100000,
				BurstSize:         20,
			},
		},
		Anthropic: &config.ServiceAPIKeyConfig{
			Enabled:     true,
			PrimaryKeys: []string{"sk-ant-prod-1", "sk-ant-prod-2"},
			ServiceEndpoint: "https://api.anthropic.com",
			APIVersion:    "v1",
			LoadBalancing: &config.ServiceLBConfig{
				Strategy: config.StrategyRoundRobin,
			},
		},
		Google: &config.ServiceAPIKeyConfig{
			Enabled:     true,
			PrimaryKeys: []string{"google-prod-key-1"},
			ServiceEndpoint: "https://generativelanguage.googleapis.com",
			APIVersion:    "v1",
			LoadBalancing: &config.ServiceLBConfig{
				Strategy: config.StrategyRoundRobin,
			},
		},
		Cohere: &config.ServiceAPIKeyConfig{
			Enabled:     true,
			PrimaryKeys: []string{"cohere-prod-1", "cohere-prod-2"},
			ServiceEndpoint: "https://api.cohere.ai",
			APIVersion:    "v1",
			LoadBalancing: &config.ServiceLBConfig{
				Strategy: config.StrategyRandom,
			},
		},
		Replicate: &config.ServiceAPIKeyConfig{
			Enabled:     true,
			PrimaryKeys: []string{"r8-prod-key-1"},
			ServiceEndpoint: "https://api.replicate.com",
			APIVersion:    "v1",
		},
		HuggingFace: &config.ServiceAPIKeyConfig{
			Enabled:     true,
			PrimaryKeys: []string{"hf-prod-token-1"},
			ServiceEndpoint: "https://huggingface.co",
		},
		LoadBalancing: &config.LoadBalancingConfig{
			DefaultStrategy: config.StrategyPriorityFirst,
			PriorityFirst:   true,
			HealthCheck: &config.HealthCheckConfig{
				Enabled:   true,
				Interval:  30 * time.Second,
				Timeout:   5 * time.Second,
				Endpoint:  "/health",
				Method:    "GET",
				Headers:   map[string]string{"User-Agent": "HelixCode/1.0"},
				ExpectedStatusCodes: []int{200, 201, 202},
			},
		},
		Fallback: &config.FallbackConfig{
			Enabled:         true,
			DefaultStrategy:  config.FallbackStrategySequential,
			MaxRetries:      5,
			RetryDelay:      time.Second,
			BackoffFactor:   2.0,
			CircuitBreaker: &config.CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 10,
				RecoveryTimeout:  5 * time.Minute,
				SuccessThreshold: 5,
				MonitoringPeriod: 10 * time.Minute,
			},
		},
		Security: &config.SecurityConfig{
			EncryptionEnabled: true,
			KeyRotation:      true,
			AuditLogging:     true,
			AccessControl:    true,
			AllowedIPs:       []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		},
		Monitoring: &config.MonitoringConfig{
			Enabled:            true,
			CollectionInterval: 30 * time.Second,
			RetentionPeriod:    7 * 24 * time.Hour, // 7 days
			MetricsTypes:       []string{"usage", "performance", "errors", "health", "latency"},
		},
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "real_world_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save configuration
	configPath := filepath.Join(tempDir, "real_world_api_keys.json")
	err = config.SaveAPIKeyConfig(realWorldConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to save configuration: %v", err)
	}

	// Load configuration
	loadedConfig, err := config.LoadAPIKeyConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Create Helix configuration
	helixConfig := &config.HelixConfig{
		APIKeys: loadedConfig,
	}

	// Create hardware profile
	hwProfile := &hardware.Profile{
		CPU: &hardware.CPUProfile{
			Cores:        16,
			Threads:      32,
			Model:        "AMD EPYC 7742",
			FrequencyGHz: 2.25,
		},
		Memory: &hardware.MemoryProfile{
			TotalGB:     128,
			AvailableGB: 100,
		},
		GPUs: []*hardware.GPUProfile{
			{
				Type:      hardware.GPUTypeNVIDIA,
				Name:      "NVIDIA A100",
				VRAMGB:    80,
				Available: true,
			},
			{
				Type:      hardware.GPUTypeNVIDIA,
				Name:      "NVIDIA A100",
				VRAMGB:    80,
				Available: true,
			},
		},
	}

	// Create API key manager
	apiKeyManager, err := config.NewAPIKeyManager(helixConfig)
	if err != nil {
		t.Fatalf("Failed to create API key manager: %v", err)
	}

	// Initialize API key manager
	err = apiKeyManager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize API key manager: %v", err)
	}

	// Test all configured services
	services := []string{"cognee", "openai", "anthropic", "google", "cohere", "replicate", "huggingface"}
	
	for _, service := range services {
		t.Run("Service_"+service, func(t *testing.T) {
			var key string
			var err error
			
			if service == "cognee" {
				key, err = apiKeyManager.GetCogneeAPIKey()
			} else {
				key, err = apiKeyManager.GetAPIKey(service)
			}
			
			if err != nil {
				t.Errorf("Failed to get API key for %s: %v", service, err)
				return
			}
			
			// Some services might not use API keys (like local providers)
			t.Logf("API key for %s: %s", service, maskKey(key))
			
			// Record usage
			if key != "" {
				apiKeyManager.RecordAPIKeyUsage(service, key, true, "", 150*time.Millisecond)
			}
		})
	}

	// Test load balancing with multiple requests
	t.Run("LoadBalancing", func(t *testing.T) {
		const numRequests = 20
		
		// Test OpenAI load balancing
		openaiKeys := make([]string, 0)
		for i := 0; i < numRequests; i++ {
			key, err := apiKeyManager.GetAPIKey("openai")
			if err != nil {
				t.Errorf("Failed to get OpenAI key on request %d: %v", i, err)
				continue
			}
			openaiKeys = append(openaiKeys, key)
			apiKeyManager.RecordAPIKeyUsage("openai", key, true, "", 100*time.Millisecond)
		}
		
		// Validate load balancing
		keyCounts := make(map[string]int)
		for _, key := range openaiKeys {
			keyCounts[key]++
		}
		
		t.Logf("OpenAI key distribution: %+v", keyCounts)
		
		// Should have used multiple keys due to load balancing
		if len(keyCounts) < 2 {
			t.Error("Expected load balancing to use multiple keys")
		}
	})

	// Test usage statistics
	t.Run("UsageStatistics", func(t *testing.T) {
		stats := apiKeyManager.GetUsageStats("openai")
		if len(stats) == 0 {
			t.Error("Expected usage statistics for OpenAI")
		} else {
			totalRequests := int64(0)
			for _, stat := range stats {
				totalRequests += stat.TotalRequests
			}
			t.Logf("OpenAI usage statistics: total requests = %d", totalRequests)
		}
	})

	// Test configuration status
	t.Run("ConfigurationStatus", func(t *testing.T) {
		status := apiKeyManager.GetKeyPoolStatus()
		if status == nil {
			t.Error("Expected status but got nil")
		} else {
			t.Logf("Configuration status: %+v", status)
			
			// Should have primary pools
			primaryPools, ok := status["primary_pools"].(map[string]interface{})
			if !ok {
				t.Error("Expected primary_pools in status")
			} else {
				t.Logf("Primary pools count: %d", len(primaryPools))
			}
			
			// Should have usage statistics
			usageStats, ok := status["usage_stats"].(map[string]*config.APIKeyUsageStats)
			if !ok {
				t.Error("Expected usage_stats in status")
			} else {
				t.Logf("Usage statistics services: %d", len(usageStats))
			}
		}
	})

	t.Log("Real-world scenario test completed successfully")
}

// Helper function to mask API keys for logging
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}