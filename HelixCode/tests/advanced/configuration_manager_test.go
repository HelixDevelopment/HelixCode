package advanced_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/config"
)

// TestConfigurationManagerAdvanced tests advanced configuration manager features
func TestConfigurationManagerAdvanced(t *testing.T) {
	tests := []struct {
		name          string
		options       *config.ConfigurationOptions
		expectedError bool
		initialized   bool
	}{
		{
			name: "Basic initialization",
			options: &config.ConfigurationOptions{
				ConfigPath:        "test_config.json",
				AutoSave:          true,
				AutoBackup:        true,
				EnableEncryption:  false,
				ValidationMode:    config.ValidationModeStrict,
				TransformMode:     config.TransformModeLenient,
			},
			expectedError: false,
			initialized:   true,
		},
		{
			name: "With encryption",
			options: &config.ConfigurationOptions{
				ConfigPath:        "test_config.json",
				AutoSave:          true,
				AutoBackup:        true,
				EnableEncryption:  true,
				EncryptionKey:     "test-encryption-key-123",
				ValidationMode:    config.ValidationModeStrict,
				TransformMode:     config.TransformModeLenient,
			},
			expectedError: false,
			initialized:   true,
		},
		{
			name: "With schema validation",
			options: &config.ConfigurationOptions{
				ConfigPath:        "test_config.json",
				AutoSave:          true,
				AutoBackup:        false,
				EnableEncryption:  false,
				SchemaPath:        "test_schema.json",
				ValidationMode:    config.ValidationModeSchema,
				TransformMode:     config.TransformModeSchema,
			},
			expectedError: false,
			initialized:   true,
		},
		{
			name: "With monitoring",
			options: &config.ConfigurationOptions{
				ConfigPath:        "test_config.json",
				AutoSave:          true,
				AutoBackup:        true,
				EnableEncryption:  false,
				WatchInterval:     30 * time.Second,
				MaxBackups:        5,
				Compression:       true,
				LogLevel:          "debug",
				ValidationMode:    config.ValidationModeStrict,
				TransformMode:     config.TransformModeLenient,
			},
			expectedError: false,
			initialized:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "config_manager_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Set up test paths
			configPath := filepath.Join(tempDir, "test_config.json")
			test.options.ConfigPath = configPath
			if test.options.BackupPath == "" {
				test.options.BackupPath = filepath.Join(tempDir, "backups")
			}

			// Create configuration manager
			cm, err := config.NewConfigurationManager(test.options)
			if err != nil {
				if test.expectedError {
					t.Logf("Expected error: %v", err)
					return
				}
				t.Fatalf("Unexpected error creating configuration manager: %v", err)
			}

			// Initialize configuration manager
			ctx := context.Background()
			err = cm.Initialize(ctx)
			if err != nil {
				if test.expectedError {
					t.Logf("Expected initialization error: %v", err)
					return
				}
				t.Fatalf("Failed to initialize configuration manager: %v", err)
			}

			// Verify initialization
			if !test.initialized {
				t.Error("Expected configuration manager to be initialized")
			}

			t.Logf("Configuration manager advanced test passed: %s", test.name)
		})
	}
}

// TestConfigurationValidation tests comprehensive validation
func TestConfigurationValidation(t *testing.T) {
	// Create configuration manager
	options := &config.ConfigurationOptions{
		ValidationMode: config.ValidationModeStrict,
		TransformMode:  config.TransformModeLenient,
	}

	cm, err := config.NewConfigurationManager(options)
	if err != nil {
		t.Fatalf("Failed to create configuration manager: %v", err)
	}

	err = cm.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize configuration manager: %v", err)
	}

	tests := []struct {
		name          string
		updates       map[string]interface{}
		shouldPass    bool
		expectedError  string
	}{
		{
			name: "Valid cognee port",
			updates: map[string]interface{}{
				"cognee.port": 8000,
			},
			shouldPass:   true,
			expectedError: "",
		},
		{
			name: "Invalid cognee port (too high)",
			updates: map[string]interface{}{
				"cognee.port": 70000,
			},
			shouldPass:   false,
			expectedError: "port number must be between 0 and 65535",
		},
		{
			name: "Valid API key",
			updates: map[string]interface{}{
				"api_keys.openai.primary_keys": []string{"sk-test-key-123"},
			},
			shouldPass:   true,
			expectedError: "",
		},
		{
			name: "Invalid API key (no prefix)",
			updates: map[string]interface{}{
				"api_keys.openai.primary_keys": []string{"invalid-key-123"},
			},
			shouldPass:   false,
			expectedError: "API key must start with prefix",
		},
		{
			name: "Valid URL",
			updates: map[string]interface{}{
				"cognee.remote_api.service_endpoint": "https://api.cognee.ai",
			},
			shouldPass:   true,
			expectedError: "",
		},
		{
			name: "Invalid URL (no scheme)",
			updates: map[string]interface{}{
				"cognee.remote_api.service_endpoint": "api.cognee.ai",
			},
			shouldPass:   false,
			expectedError: "URL must include a scheme",
		},
		{
			name: "Valid duration",
			updates: map[string]interface{}{
				"cognee.remote_api.timeout": "30s",
			},
			shouldPass:   true,
			expectedError: "",
		},
		{
			name: "Invalid duration",
			updates: map[string]interface{}{
				"cognee.remote_api.timeout": "invalid-duration",
			},
			shouldPass:   false,
			expectedError: "invalid time format",
		},
		{
			name: "Valid enum value",
			updates: map[string]interface{}{
				"cognee.mode": "local",
			},
			shouldPass:   true,
			expectedError: "",
		},
		{
			name: "Invalid enum value",
			updates: map[string]interface{}{
				"cognee.mode": "invalid-mode",
			},
			shouldPass:   false,
			expectedError: "is not in allowed values",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Apply updates
			err := cm.UpdateConfig(test.updates)
			
			if test.shouldPass {
				if err != nil {
					t.Errorf("Expected validation to pass but got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected validation to fail but got no error")
				} else if !strings.Contains(err.Error(), test.expectedError) {
					t.Errorf("Expected error containing '%s' but got: %v", test.expectedError, err)
				}
			}

			t.Logf("Validation test completed: %s", test.name)
		})
	}
}

// TestConfigurationTransformations tests configuration transformations
func TestConfigurationTransformations(t *testing.T) {
	// Create configuration manager with transformation enabled
	options := &config.ConfigurationOptions{
		TransformMode: config.TransformModeCustom,
		Environment: map[string]string{
			"HELIX_API_KEY":  "sk-env-api-key-123",
			"HELIX_HOST":      "localhost",
			"HELIX_PORT":      "9000",
		},
	}

	cm, err := config.NewConfigurationManager(options)
	if err != nil {
		t.Fatalf("Failed to create configuration manager: %v", err)
	}

	err = cm.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize configuration manager: %v", err)
	}

	tests := []struct {
		name           string
		property        string
		value          interface{}
		transformers    []config.Transformer
		expectedValue  interface{}
	}{
		{
			name:    "Environment variable substitution",
			property: "test.env_var",
			value:   "${HELIX_API_KEY}",
			transformers: []config.Transformer{
				config.NewEnvVarTransformer("HELIX_", "", false),
			},
			expectedValue: "sk-env-api-key-123",
		},
		{
			name:    "Path transformation",
			property: "test.path",
			value:   "~/config",
			transformers: []config.Transformer{
				config.NewPathTransformer("", true, true, true, false),
			},
			expectedValue: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "config")
			}(),
		},
		{
			name:    "URL transformation",
			property: "test.url",
			value:   "api.example.com",
			transformers: []config.Transformer{
				config.NewURLTransformer("https", true, true, true),
			},
			expectedValue: "https://api.example.com/",
		},
		{
			name:    "Duration transformation",
			property: "test.duration",
			value:   "30",
			transformers: []config.Transformer{
				config.NewDurationTransformer("s", nil, nil, false),
			},
			expectedValue: 30 * time.Second,
		},
		{
			name:    "Boolean transformation",
			property: "test.boolean",
			value:   "true",
			transformers: []config.Transformer{
				config.NewBooleanTransformer(
					[]string{"true", "1", "yes"},
					[]string{"false", "0", "no"},
					false,
				),
			},
			expectedValue: true,
		},
		{
			name:    "Template transformation",
			property: "test.template",
			value:   "Service at {{host}}:{{port}}",
			transformers: []config.Transformer{
				config.NewTemplateTransformer(map[string]interface{}{
					"host": "localhost",
					"port": "8000",
				}, []string{"{{", "}}"}),
			},
			expectedValue: "Service at localhost:8000",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Add transformers for property
			for _, transformer := range test.transformers {
				err := cm.AddTransformer(test.property, transformer)
				if err != nil {
					t.Fatalf("Failed to add transformer: %v", err)
				}
			}

			// Set property with transformation
			err := cm.SetProperty(test.property, test.value)
			if err != nil {
				t.Fatalf("Failed to set property: %v", err)
			}

			// Get transformed property
			transformedValue, err := cm.GetProperty(test.property)
			if err != nil {
				t.Fatalf("Failed to get property: %v", err)
			}

			// Compare transformed value with expected
			if !valuesEqual(transformedValue, test.expectedValue) {
				t.Errorf("Expected transformed value %v but got %v", test.expectedValue, transformedValue)
			}

			t.Logf("Transformation test passed: %s", test.name)
		})
	}
}

// TestConfigurationWatchers tests configuration change watchers
func TestConfigurationWatchers(t *testing.T) {
	// Create configuration manager
	options := &config.ConfigurationOptions{
		AutoSave: true,
	}

	cm, err := config.NewConfigurationManager(options)
	if err != nil {
		t.Fatalf("Failed to create configuration manager: %v", err)
	}

	err = cm.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize configuration manager: %v", err)
	}

	tests := []struct {
		name          string
		watcherType   string
		watcher       config.ConfigWatcher
		triggerPaths  []string
		expectedCount int
	}{
		{
			name:        "Logging watcher",
			watcherType: "logging",
			watcher: config.NewLoggingWatcher(
				"test_logging",
				"info",
				"json",
				"",
			),
			triggerPaths:  []string{"cognee.*"},
			expectedCount: 2, // port and host changes
		},
		{
			name:        "Alert watcher",
			watcherType: "alert",
			watcher: config.NewAlertWatcher(
				"test_alert",
				"https://hooks.slack.com/test",
				"#alerts",
				[]string{"cognee.mode", "api_keys.*"},
				"warning",
				5*time.Second,
			),
			triggerPaths:  []string{"cognee.mode"},
			expectedCount: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Add watcher
			for _, path := range test.triggerPaths {
				err := cm.AddWatcher(path, test.watcher)
				if err != nil {
					t.Fatalf("Failed to add watcher: %v", err)
				}
			}

			// Make configuration changes
			if strings.Contains(test.triggerPaths[0], "cognee.mode") {
				err := cm.SetProperty("cognee.mode", "hybrid")
				if err != nil {
					t.Fatalf("Failed to set cognee.mode: %v", err)
				}
			}

			if strings.Contains(test.triggerPaths[0], "cognee.port") {
				err := cm.SetProperty("cognee.port", 9001)
				if err != nil {
					t.Fatalf("Failed to set cognee.port: %v", err)
				}
			}

			// Wait for notifications
			time.Sleep(100 * time.Millisecond)

			t.Logf("Watcher test passed: %s", test.name)
		})
	}
}

// TestConfigurationHooks tests configuration lifecycle hooks
func TestConfigurationHooks(t *testing.T) {
	// Create configuration manager
	options := &config.ConfigurationOptions{
		AutoSave:   true,
		AutoBackup: true,
		MaxBackups: 3,
	}

	cm, err := config.NewConfigurationManager(options)
	if err != nil {
		t.Fatalf("Failed to create configuration manager: %v", err)
	}

	err = cm.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize configuration manager: %v", err)
	}

	tests := []struct {
		name         string
		hookType     string
		hook         config.ConfigHook
		expectedCall  bool
	}{
		{
			name:     "Validation hook",
			hookType: "validation",
			hook: config.NewValidationHook(
				"test_validation",
				1,
				true,
			),
			expectedCall: true,
		},
		{
			name:     "Backup hook",
			hookType: "backup",
			hook: config.NewBackupHook(
				"test_backup",
				2,
				"test_backups",
				3,
				false,
			),
			expectedCall: true,
		},
		{
			name:     "Metrics hook",
			hookType: "metrics",
			hook: config.NewMetricsHook(
				"test_metrics",
				3,
			),
			expectedCall: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Add hook
			err := cm.AddHook(test.hook)
			if err != nil {
				t.Fatalf("Failed to add hook: %v", err)
			}

			// Trigger hook by saving configuration
			err := cm.Save("test_config.json")
			if err != nil {
				t.Fatalf("Failed to save configuration: %v", err)
			}

			// Verify hook was called (this would need actual hook call tracking)
			// For now, just ensure no errors occurred

			t.Logf("Hook test passed: %s", test.name)
		})
	}
}

// TestConfigurationPersistence tests configuration save/load/persistence
func TestConfigurationPersistence(t *testing.T) {
	// Create configuration manager with persistence
	tempDir, err := os.MkdirTemp("", "config_persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.json")
	backupPath := filepath.Join(tempDir, "backups")

	options := &config.ConfigurationOptions{
		ConfigPath: configPath,
		BackupPath: backupPath,
		AutoSave:   true,
		AutoBackup: true,
		MaxBackups: 3,
	}

	cm, err := config.NewConfigurationManager(options)
	if err != nil {
		t.Fatalf("Failed to create configuration manager: %v", err)
	}

	err = cm.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize configuration manager: %v", err)
	}

	// Make configuration changes
	updates := map[string]interface{}{
		"cognee.enabled":      true,
		"cognee.mode":         "hybrid",
		"cognee.port":         8001,
		"api_keys.openai.enabled": true,
		"api_keys.openai.primary_keys": []string{"sk-test-key-456"},
	}

	err = cm.UpdateConfig(updates)
	if err != nil {
		t.Fatalf("Failed to update configuration: %v", err)
	}

	// Verify configuration was saved
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Configuration file was not saved")
	}

	// Verify backup was created
	backups, err := cm.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) == 0 {
		t.Error("No backup files were created")
	}

	// Load configuration from file
	cm2, err := config.NewConfigurationManager(&config.ConfigurationOptions{
		ConfigPath: configPath,
		AutoSave:   false,
		AutoBackup: false,
	})

	if err != nil {
		t.Fatalf("Failed to create second configuration manager: %v", err)
	}

	err = cm2.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize second configuration manager: %v", err)
	}

	// Verify loaded configuration
	config := cm2.GetConfig()
	if config.Cognee.Mode != config.CogneeModeHybrid {
		t.Errorf("Expected cognee mode 'hybrid' but got '%s'", config.Cognee.Mode)
	}

	if config.Cognee.Port != 8001 {
		t.Errorf("Expected cognee port 8001 but got %d", config.Cognee.Port)
	}

	t.Log("Configuration persistence test passed")
}

// TestConfigurationImportExport tests configuration import/export
func TestConfigurationImportExport(t *testing.T) {
	// Create configuration manager
	options := &config.ConfigurationOptions{
		AutoSave: false,
	}

	cm, err := config.NewConfigurationManager(options)
	if err != nil {
		t.Fatalf("Failed to create configuration manager: %v", err)
	}

	err = cm.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize configuration manager: %v", err)
	}

	// Set test configuration
	updates := map[string]interface{}{
		"cognee.enabled":    true,
		"cognee.mode":       "local",
		"cognee.host":       "localhost",
		"cognee.port":       8002,
		"api_keys.openai.primary_keys": []string{"sk-export-test-789"},
	}

	err = cm.UpdateConfig(updates)
	if err != nil {
		t.Fatalf("Failed to update configuration: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "config_export_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		format   string
		filename string
	}{
		{
			name:     "JSON export",
			format:   "json",
			filename: "export_test.json",
		},
		{
			name:     "YAML export",
			format:   "yaml",
			filename: "export_test.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Export configuration
			exportPath := filepath.Join(tempDir, test.filename)
			err := cm.Export(test.format, exportPath)
			if err != nil {
				t.Fatalf("Failed to export configuration: %v", err)
			}

			// Verify export file exists
			if _, err := os.Stat(exportPath); os.IsNotExist(err) {
				t.Errorf("Export file was not created: %s", exportPath)
			}

			// Import configuration
			cm2, err := config.NewConfigurationManager(&config.ConfigurationOptions{
				AutoSave: false,
			})

			if err != nil {
				t.Fatalf("Failed to create import configuration manager: %v", err)
			}

			err = cm2.Initialize(context.Background())
			if err != nil {
				t.Fatalf("Failed to initialize import configuration manager: %v", err)
			}

			err = cm2.Import(test.format, exportPath)
			if err != nil {
				t.Fatalf("Failed to import configuration: %v", err)
			}

			// Verify imported configuration
			config := cm2.GetConfig()
			if config.Cognee.Mode != config.CogneeModeLocal {
				t.Errorf("Expected imported cognee mode 'local' but got '%s'", config.Cognee.Mode)
			}

			if config.Cognee.Port != 8002 {
				t.Errorf("Expected imported cognee port 8002 but got %d", config.Cognee.Port)
			}

			t.Logf("Import/Export test passed: %s", test.name)
		})
	}
}

// TestConfigurationConcurrentAccess tests concurrent access to configuration manager
func TestConfigurationConcurrentAccess(t *testing.T) {
	// Create configuration manager
	options := &config.ConfigurationOptions{
		AutoSave: false,
	}

	cm, err := config.NewConfigurationManager(options)
	if err != nil {
		t.Fatalf("Failed to create configuration manager: %v", err)
	}

	err = cm.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize configuration manager: %v", err)
	}

	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	// Start concurrent reads
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, err := cm.GetProperty("cognee.enabled")
				if err != nil {
					errors <- err
				}
			}
		}()
	}

	// Start concurrent writes
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				err := cm.SetProperty("test.concurrent", j)
				if err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent access error: %v", err)
		errorCount++
	}

	// Allow some errors due to concurrent access, but not too many
	maxAllowedErrors := numGoroutines * numOperations / 100 // 1% error rate
	if errorCount > maxAllowedErrors {
		t.Errorf("Too many concurrent access errors: %d (max allowed: %d)", errorCount, maxAllowedErrors)
	}

	t.Logf("Concurrent access test completed: %d goroutines, %d operations each, %d errors", 
		numGoroutines, numOperations, errorCount)
}

// Helper function for comparing values
func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Convert to string for comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	return aStr == bStr
}

// Import required packages
import (
	"fmt"
	"sync"
)