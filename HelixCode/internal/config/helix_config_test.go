package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHelixConfigManager tests configuration manager creation
func TestNewHelixConfigManager(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	// Test creating new manager with existing config
	defaultConfig := getDefaultConfig()
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, configPath, manager.GetConfigPath())

	// Test creating manager with non-existent config (should create default)
	configPath2 := filepath.Join(tempDir, "test_config2.json")
	manager2, err := NewHelixConfigManager(configPath2)
	require.NoError(t, err)
	assert.NotNil(t, manager2)
	assert.True(t, manager2.IsConfigPresent())
}

// TestHelixConfigLoadSave tests configuration loading and saving
func TestHelixConfigLoadSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	// Test default config creation
	config := manager.GetConfig()
	require.NotNil(t, config)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "HelixCode", config.Application.Name)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "local", config.LLM.DefaultProvider)

	// Modify config
	config.Application.Name = "Test Application"
	config.Server.Port = 9090
	config.LLM.MaxTokens = 8192

	// Save config
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Application.Name = "Test Application"
		c.Server.Port = 9090
		c.LLM.MaxTokens = 8192
	})
	require.NoError(t, err)

	// Load config into new manager
	manager2, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	config2 := manager2.GetConfig()
	assert.Equal(t, "Test Application", config2.Application.Name)
	assert.Equal(t, 9090, config2.Server.Port)
	assert.Equal(t, 8192, config2.LLM.MaxTokens)
}

// TestHelixConfigValidation tests configuration validation
func TestHelixConfigValidation(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	// Test invalid server port
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Server.Port = 70000 // Invalid port
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server port")

	// Test empty database host
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Server.Port = 8080 // Reset to valid
		c.Database.Host = ""
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database host is required")

	// Test invalid Redis port when enabled
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Database.Host = "localhost"
		c.Redis.Enabled = true
		c.Redis.Port = 70000 // Invalid port
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid redis port")

	// Test empty LLM provider
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Redis.Port = 6379 // Reset to valid
		c.LLM.DefaultProvider = ""
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default LLM provider is required")

	// Test invalid temperature
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.LLM.DefaultProvider = "local"
		c.LLM.Temperature = 3.0 // Invalid temperature (> 2)
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "temperature must be between 0 and 2")

	// Test invalid max tokens
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.LLM.Temperature = 0.7 // Reset to valid
		c.LLM.MaxTokens = 0     // Invalid tokens
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max tokens must be positive")
}

// TestHelixConfigWatchers tests configuration change watchers
func TestHelixConfigWatchers(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	// Create test watcher
	watcherCalled := false
	var oldConfig, newConfig *HelixConfig

	testWatcher := &TestConfigWatcher{
		OnChangeFunc: func(old, new *HelixConfig) error {
			watcherCalled = true
			oldConfig = old
			newConfig = new
			return nil
		},
	}

	// Add watcher
	manager.AddWatcher(testWatcher)

	// Update config
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Application.Name = "Watched Change"
	})
	require.NoError(t, err)

	// Verify watcher was called
	assert.True(t, watcherCalled)
	assert.NotNil(t, oldConfig)
	assert.NotNil(t, newConfig)
	assert.Equal(t, "HelixConfig", oldConfig.Application.Name)
	assert.Equal(t, "Watched Change", newConfig.Application.Name)
}

// TestHelixConfigExportImport tests configuration export and import
func TestHelixConfigExportImport(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")
	exportPath := filepath.Join(tempDir, "exported_config.json")

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	// Modify config
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Application.Name = "Export Test"
		c.Server.Port = 9090
	})
	require.NoError(t, err)

	// Export config
	err = manager.ExportConfig(exportPath)
	require.NoError(t, err)
	assert.FileExists(t, exportPath)

	// Verify exported content
	data, err := os.ReadFile(exportPath)
	require.NoError(t, err)

	var exportedConfig HelixConfig
	err = json.Unmarshal(data, &exportedConfig)
	require.NoError(t, err)
	assert.Equal(t, "Export Test", exportedConfig.Application.Name)
	assert.Equal(t, 9090, exportedConfig.Server.Port)

	// Create new manager and import
	manager2, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	err = manager2.ImportConfig(exportPath)
	require.NoError(t, err)

	config2 := manager2.GetConfig()
	assert.Equal(t, "Export Test", config2.Application.Name)
	assert.Equal(t, 9090, config2.Server.Port)
}

// TestHelixConfigBackup tests configuration backup
func TestHelixConfigBackup(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")
	backupPath := filepath.Join(tempDir, "backup_config.json")

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	// Modify config
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Application.Name = "Backup Test"
	})
	require.NoError(t, err)

	// Create backup
	err = manager.BackupConfig(backupPath)
	require.NoError(t, err)
	assert.FileExists(t, backupPath)

	// Verify backup content
	data, err := os.ReadFile(backupPath)
	require.NoError(t, err)

	var backupConfig HelixConfig
	err = json.Unmarshal(data, &backupConfig)
	require.NoError(t, err)
	assert.Equal(t, "Backup Test", backupConfig.Application.Name)

	// Test automatic backup path generation
	manager2, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	err = manager2.BackupConfig("")
	require.NoError(t, err)

	// Check for auto-generated backup file
	files, err := filepath.Glob(filepath.Join(tempDir, "helix_backup_*.json"))
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

// TestHelixConfigReset tests configuration reset to defaults
func TestHelixConfigReset(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	// Modify config
	err = manager.UpdateConfig(func(c *HelixConfig) {
		c.Application.Name = "Modified Config"
		c.Server.Port = 9090
		c.LLM.MaxTokens = 8192
	})
	require.NoError(t, err)

	// Verify modifications
	config := manager.GetConfig()
	assert.Equal(t, "Modified Config", config.Application.Name)
	assert.Equal(t, 9090, config.Server.Port)
	assert.Equal(t, 8192, config.LLM.MaxTokens)

	// Reset to defaults
	err = manager.ResetToDefaults()
	require.NoError(t, err)

	// Verify reset
	config = manager.GetConfig()
	assert.Equal(t, "HelixCode", config.Application.Name)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, 4096, config.LLM.MaxTokens)
}

// TestGetDefaultConfigPath tests default configuration path
func TestGetDefaultConfigPath(t *testing.T) {
	path := GetHelixConfigPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "helix.json")
	assert.Contains(t, path, ".config")
}

// TestLoadSaveHelixConfig tests global configuration functions
func TestLoadSaveHelixConfig(t *testing.T) {
	// Create temporary directory for test
	tempHome := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create config directory
	configDir := filepath.Join(tempHome, ".config", "helix")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Test creating default config
	err = CreateDefaultHelixConfig()
	require.NoError(t, err)
	assert.True(t, IsHelixConfigPresent())

	// Test loading config
	config, err := LoadHelixConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "HelixCode", config.Application.Name)

	// Test modifying and saving config
	config.Application.Name = "Global Test"
	err = SaveHelixConfig(config)
	require.NoError(t, err)

	// Test updating config
	err = UpdateHelixConfig(func(c *HelixConfig) {
		c.Server.Port = 9090
	})
	require.NoError(t, err)

	// Reload and verify changes
	config2, err := LoadHelixConfig()
	require.NoError(t, err)
	assert.Equal(t, "Global Test", config2.Application.Name)
	assert.Equal(t, 9090, config2.Server.Port)
}

// TestHelixConfigDefaultValues tests all default configuration values
func TestHelixConfigDefaultValues(t *testing.T) {
	config := getDefaultConfig()
	require.NotNil(t, config)

	// Application section
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "system", config.UpdatedBy)
	assert.Equal(t, "HelixCode", config.Application.Name)
	assert.Equal(t, "Distributed AI Development Platform", config.Application.Description)
	assert.Equal(t, "development", config.Application.Environment)
	assert.Equal(t, "~/helixcode", config.Application.Workspace.DefaultPath)
	assert.True(t, config.Application.Workspace.AutoSave)
	assert.Equal(t, 300, config.Application.Workspace.AutoSaveInterval)
	assert.True(t, config.Application.Workspace.BackupEnabled)
	assert.Equal(t, "~/helixcode/backups", config.Application.Workspace.BackupLocation)
	assert.Equal(t, 30, config.Application.Workspace.BackupRetention)
	assert.Equal(t, 60, config.Application.Session.Timeout)
	assert.True(t, config.Application.Session.PersistContext)
	assert.Equal(t, 7, config.Application.Session.ContextRetention)
	assert.Equal(t, 1000, config.Application.Session.MaxHistorySize)
	assert.True(t, config.Application.Session.AutoResume)
	assert.True(t, config.Application.Session.ContextCompression.Enabled)
	assert.Equal(t, 10000, config.Application.Session.ContextCompression.Threshold)
	assert.Equal(t, "hybrid", config.Application.Session.ContextCompression.Strategy)
	assert.Equal(t, 0.5, config.Application.Session.ContextCompression.CompressionRatio)
	assert.Equal(t, "7days", config.Application.Session.ContextCompression.RetentionPolicy)
	assert.Equal(t, "info", config.Application.Logging.Level)
	assert.Equal(t, "text", config.Application.Logging.Format)
	assert.Equal(t, "stdout", config.Application.Logging.Output)
	assert.False(t, config.Application.Telemetry.Enabled)
	assert.Equal(t, 30, config.Application.Telemetry.DataRetention)

	// Database section
	assert.Equal(t, "postgresql", config.Database.Type)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "helixcode", config.Database.Database)
	assert.Equal(t, "helixcode", config.Database.Username)
	assert.Equal(t, "disable", config.Database.SSLMode)
	assert.Equal(t, 20, config.Database.MaxConnections)
	assert.Equal(t, 5, config.Database.MaxIdleConnections)
	assert.Equal(t, 3600*time.Second, config.Database.ConnectionLifetime)
	assert.True(t, config.Database.EnableQueryCache)
	assert.Equal(t, 30*time.Second, config.Database.QueryTimeout)
	assert.True(t, config.Database.BackupEnabled)
	assert.Equal(t, "~/helixcode/backups/database", config.Database.BackupPath)
	assert.False(t, config.Database.Replication)

	// Redis section
	assert.True(t, config.Redis.Enabled)
	assert.Equal(t, "localhost", config.Redis.Host)
	assert.Equal(t, 6379, config.Redis.Port)
	assert.Equal(t, 0, config.Redis.Database)
	assert.Equal(t, 20, config.Redis.MaxConnections)
	assert.Equal(t, 5, config.Redis.MaxIdleConnections)
	assert.Equal(t, 20, config.Redis.PoolSize)
	assert.Equal(t, 2, config.Redis.MinIdleConnections)
	assert.Equal(t, 3, config.Redis.MaxRetries)
	assert.Equal(t, 5*time.Second, config.Redis.DialTimeout)
	assert.Equal(t, 3*time.Second, config.Redis.ReadTimeout)
	assert.Equal(t, 3*time.Second, config.Redis.WriteTimeout)
	assert.False(t, config.Redis.ClusterEnabled)
	assert.Empty(t, config.Redis.ClusterNodes)

	// Auth section
	assert.NotEmpty(t, config.Auth.TokenExpiry)
	assert.NotEmpty(t, config.Auth.RefreshTokenExpiry)
	assert.Equal(t, 30, config.Auth.SessionTimeout)
	assert.True(t, config.Auth.RememberMe)
	assert.Equal(t, 12, config.Auth.BcryptCost)
	assert.False(t, config.Auth.Require2FA)
	assert.Equal(t, 5, config.Auth.MaxLoginAttempts)
	assert.Equal(t, 15, config.Auth.LockoutDuration)
	assert.True(t, config.Auth.RBACEnabled)
	assert.Empty(t, config.Auth.OAuthProviders)
	assert.Empty(t, config.Auth.Roles)
	assert.Empty(t, config.Auth.Permissions)

	// Server section
	assert.Equal(t, "0.0.0.0", config.Server.Address)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, 30*time.Second, config.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.Server.WriteTimeout)
	assert.Equal(t, 60*time.Second, config.Server.IdleTimeout)
	assert.Equal(t, 30*time.Second, config.Server.ShutdownTimeout)
	assert.False(t, config.Server.SSLEnabled)
	assert.Empty(t, config.Server.SSLCertFile)
	assert.Empty(t, config.Server.SSLKeyFile)
	assert.Equal(t, []string{"*"}, config.Server.CORSAllowedOrigins)
	assert.Equal(t, []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, config.Server.CORSAllowedMethods)
	assert.Equal(t, []string{"*"}, config.Server.CORSAllowedHeaders)
	assert.False(t, config.Server.RateLimitEnabled)
	assert.Equal(t, "100/hour", config.Server.RateLimitRate)
	assert.False(t, config.Server.ProxyEnabled)
	assert.Empty(t, config.Server.ProxyURL)

	// Workers section
	assert.Equal(t, 30*time.Second, config.Workers.HealthCheckInterval)
	assert.Equal(t, 120*time.Second, config.Workers.HealthTTL)
	assert.Equal(t, 10, config.Workers.MaxConcurrentTasks)
	assert.False(t, config.Workers.AutoScaling)
	assert.Equal(t, 1, config.Workers.MinWorkers)
	assert.Equal(t, 10, config.Workers.MaxWorkers)
	assert.Equal(t, 80, config.Workers.ScaleUpThreshold)
	assert.Equal(t, 20, config.Workers.ScaleDownThreshold)
	assert.Equal(t, 80.0, config.Workers.CPULimit)
	assert.Equal(t, int64(1024*1024*1024), config.Workers.MemoryLimit)
	assert.Equal(t, int64(10*1024*1024*1024), config.Workers.DiskLimit)
	assert.True(t, config.Workers.IsolationEnabled)
	assert.Equal(t, "docker", config.Workers.SandboxType)
	assert.Equal(t, "helix", config.Workers.DefaultSSHUser)
	assert.Equal(t, 22, config.Workers.DefaultSSHPort)
	assert.Equal(t, "~/.ssh/id_rsa", config.Workers.SSHKeyPath)
	assert.Empty(t, config.Workers.SSHKeyPassphrase)
	assert.True(t, config.Workers.AutoInstallEnabled)

	// Tasks section
	assert.Equal(t, 1000, config.Tasks.QueueSize)
	assert.Equal(t, 3, config.Tasks.MaxRetries)
	assert.Equal(t, 5*time.Second, config.Tasks.RetryDelay)
	assert.Equal(t, 60*time.Second, config.Tasks.MaxRetryDelay)
	assert.Equal(t, 300*time.Second, config.Tasks.CheckpointInterval)
	assert.Equal(t, 7, config.Tasks.CheckpointRetention)
	assert.Equal(t, "local", config.Tasks.CheckpointStorage)
	assert.Equal(t, 5, config.Tasks.PriorityLevels)
	assert.Equal(t, "normal", config.Tasks.DefaultPriority)
	assert.True(t, config.Tasks.DependencyResolution)
	assert.Equal(t, 10, config.Tasks.MaxDependencyDepth)
	assert.Equal(t, 3600*time.Second, config.Tasks.CleanupInterval)
	assert.Equal(t, 30, config.Tasks.TaskRetention)
	assert.Equal(t, 7, config.Tasks.LogRetention)

	// LLM section
	assert.Equal(t, "local", config.LLM.DefaultProvider)
	assert.Equal(t, "llama-3.2-3b", config.LLM.DefaultModel)
	assert.Equal(t, 4096, config.LLM.MaxTokens)
	assert.Equal(t, 0.7, config.LLM.Temperature)
	assert.Equal(t, 0.9, config.LLM.TopP)
	assert.Empty(t, config.LLM.Providers)
	assert.Equal(t, "performance", config.LLM.ModelSelection.Strategy)
	assert.True(t, config.LLM.ModelSelection.FallbackEnabled)
	assert.Empty(t, config.LLM.ModelSelection.FallbackChain)
	assert.True(t, config.LLM.ModelSelection.HealthCheck)
	assert.False(t, config.LLM.ModelSelection.LoadBalancing)
	assert.True(t, config.LLM.ModelSelection.AutoFailover)
	assert.True(t, config.LLM.ModelSelection.PerformanceMetrics)

	// LLM Features
	assert.True(t, config.LLM.Features.ReasoningEnabled)
	assert.Equal(t, []string{"o1-preview", "claude-4-sonnet"}, config.LLM.Features.ReasoningModels)
	assert.True(t, config.LLM.Features.CachingEnabled)
	assert.Equal(t, "conservative", config.LLM.Features.CacheStrategy)
	assert.Equal(t, 3600*time.Second, config.LLM.Features.CacheTTL)
	assert.Equal(t, int64(100*1024*1024), config.LLM.Features.CacheMaxSize)
	assert.True(t, config.LLM.Features.ToolsEnabled)
	assert.Equal(t, 30*time.Second, config.LLM.Features.ToolTimeout)
	assert.Equal(t, 5, config.LLM.Features.MaxConcurrentTools)
	assert.True(t, config.LLM.Features.VisionEnabled)
	assert.Equal(t, int64(10*1024*1024), config.LLM.Features.VisionMaxImageSize)
	assert.Equal(t, []string{"png", "jpg", "jpeg", "webp"}, config.LLM.Features.VisionFormats)
	assert.True(t, config.LLM.Features.StreamingEnabled)
	assert.Equal(t, 1024, config.LLM.Features.StreamingChunkSize)
	assert.Equal(t, 60*time.Second, config.LLM.Features.StreamingTimeout)

	// LLM Performance
	assert.True(t, config.LLM.Performance.PoolEnabled)
	assert.Equal(t, 20, config.LLM.Performance.MaxConnections)
	assert.Equal(t, 5, config.LLM.Performance.MaxIdleConnections)
	assert.False(t, config.LLM.Performance.BatchingEnabled)
	assert.Equal(t, 10, config.LLM.Performance.BatchSize)
	assert.Equal(t, 5*time.Second, config.LLM.Performance.BatchTimeout)
	assert.True(t, config.LLM.Performance.ResponseCacheEnabled)
	assert.Equal(t, 300*time.Second, config.LLM.Performance.ResponseCacheTTL)
	assert.Equal(t, int64(50*1024*1024), config.LLM.Performance.ResponseCacheSize)
	assert.True(t, config.LLM.Performance.MetricsEnabled)
	assert.Empty(t, config.LLM.Performance.MetricsEndpoint)
	assert.Equal(t, 60*time.Second, config.LLM.Performance.MetricsInterval)

	// LLM Cost Management
	assert.False(t, config.LLM.CostManagement.BudgetEnabled)
	assert.Equal(t, 10.0, config.LLM.CostManagement.DailyBudget)
	assert.Equal(t, 50.0, config.LLM.CostManagement.WeeklyBudget)
	assert.Equal(t, 200.0, config.LLM.CostManagement.MonthlyBudget)
	assert.True(t, config.LLM.CostManagement.CostTrackingEnabled)
	assert.True(t, config.LLM.CostManagement.CostAlertsEnabled)
	assert.Equal(t, 50.0, config.LLM.CostManagement.CostAlertThreshold)
	assert.True(t, config.LLM.CostManagement.CostOptimizationEnabled)
	assert.Empty(t, config.LLM.CostManagement.PreferredProviders)
	assert.Empty(t, config.LLM.CostManagement.CheapestProviders)
	assert.False(t, config.LLM.CostManagement.TokenLimitsEnabled)
	assert.Equal(t, 100000, config.LLM.CostManagement.DailyTokenLimit)
	assert.Equal(t, 3000000, config.LLM.CostManagement.MonthlyTokenLimit)

	// Tools section
	assert.True(t, config.Tools.FileSystem.Enabled)
	assert.Equal(t, []string{"~", "/tmp"}, config.Tools.FileSystem.AllowedPaths)
	assert.Equal(t, []string{"/etc", "/usr/bin", "/bin"}, config.Tools.FileSystem.DeniedPaths)
	assert.True(t, config.Tools.FileSystem.ReadEnabled)
	assert.True(t, config.Tools.FileSystem.WriteEnabled)
	assert.False(t, config.Tools.FileSystem.DeleteEnabled)
	assert.False(t, config.Tools.FileSystem.ExecuteEnabled)
	assert.Equal(t, int64(100*1024*1024), config.Tools.FileSystem.MaxFileSize)
	assert.Equal(t, int64(10*1024*1024), config.Tools.FileSystem.MaxReadSize)
	assert.Equal(t, 100, config.Tools.FileSystem.MaxFilesPerOp)
	assert.True(t, config.Tools.FileSystem.PermissionChecks)
	assert.False(t, config.Tools.FileSystem.SymlinkFollow)
	assert.True(t, config.Tools.FileSystem.GitAware)

	assert.True(t, config.Tools.Shell.Enabled)
	assert.Equal(t, []string{"ls", "cd", "pwd", "cat", "echo", "grep", "find"}, config.Tools.Shell.AllowedCommands)
	assert.Equal(t, []string{"rm", "sudo", "su", "chmod", "chown"}, config.Tools.Shell.DeniedCommands)
	assert.True(t, config.Tools.Shell.SandboxEnabled)
	assert.Equal(t, "docker", config.Tools.Shell.SandboxType)
	assert.Equal(t, "/tmp/helix", config.Tools.Shell.WorkingDirectory)
	assert.Empty(t, config.Tools.Shell.EnvironmentVars)
	assert.Equal(t, 300*time.Second, config.Tools.Shell.MaxExecutionTime)
	assert.Equal(t, int64(512*1024*1024), config.Tools.Shell.MaxMemoryUsage)
	assert.Equal(t, 10, config.Tools.Shell.MaxProcesses)
	assert.True(t, config.Tools.Shell.RequireConfirmation)
	assert.Equal(t, []string{"rm -rf", "sudo", "chmod 777"}, config.Tools.Shell.DangerousCommands)
	assert.False(t, config.Tools.Shell.SudoEnabled)
	assert.True(t, config.Tools.Shell.LogEnabled)
	assert.Equal(t, "info", config.Tools.Shell.LogLevel)
	assert.Equal(t, "file", config.Tools.Shell.LogOutput)

	assert.True(t, config.Tools.Browser.Enabled)
	assert.Equal(t, "chrome", config.Tools.Browser.DefaultBrowser)
	assert.True(t, config.Tools.Browser.Headless)
	assert.Equal(t, [2]int{1920, 1080}, config.Tools.Browser.WindowSize)
	assert.Equal(t, "HelixCode/1.0", config.Tools.Browser.UserAgent)
	assert.Equal(t, [2]float64{1920, 1080}, config.Tools.Browser.Viewport)
	assert.True(t, config.Tools.Browser.SandboxEnabled)
	assert.Empty(t, config.Tools.Browser.AllowedDomains)
	assert.Empty(t, config.Tools.Browser.DeniedDomains)
	assert.True(t, config.Tools.Browser.BlockPopups)
	assert.Equal(t, 30*time.Second, config.Tools.Browser.Timeout)
	assert.Equal(t, 30*time.Second, config.Tools.Browser.PageLoadTimeout)
	assert.Equal(t, 10*time.Second, config.Tools.Browser.WaitTimeout)
	assert.Equal(t, 5, config.Tools.Browser.MaxConcurrentTabs)
	assert.True(t, config.Tools.Browser.ScreenshotEnabled)
	assert.True(t, config.Tools.Browser.ConsoleLogEnabled)
	assert.False(t, config.Tools.Browser.NetworkLoggingEnabled)
	assert.True(t, config.Tools.Browser.CookieHandlingEnabled)

	assert.True(t, config.Tools.Web.Enabled)
	assert.Equal(t, []string{"google", "duckduckgo"}, config.Tools.Web.SearchEngines)
	assert.Equal(t, "HelixCode/1.0", config.Tools.Web.UserAgent)
	assert.Equal(t, 30*time.Second, config.Tools.Web.Timeout)
	assert.Equal(t, 3, config.Tools.Web.MaxRetries)
	assert.Equal(t, 5*time.Second, config.Tools.Web.RetryDelay)
	assert.False(t, config.Tools.Web.ProxyEnabled)
	assert.Empty(t, config.Tools.Web.ProxyURL)
	assert.Empty(t, config.Tools.Web.ProxyAuth)
	assert.True(t, config.Tools.Web.CacheEnabled)
	assert.Equal(t, 900*time.Second, config.Tools.Web.CacheTTL)
	assert.Equal(t, int64(100*1024*1024), config.Tools.Web.CacheSize)
	assert.True(t, config.Tools.Web.RateLimitEnabled)
	assert.Equal(t, 10, config.Tools.Web.RateLimitRPS)
	assert.Equal(t, int64(10*1024*1024), config.Tools.Web.MaxContentSize)
	assert.Equal(t, []string{"text/html", "application/json", "text/plain"}, config.Tools.Web.AllowedTypes)
	assert.Empty(t, config.Tools.Web.BlockedDomains)

	assert.False(t, config.Tools.Voice.Enabled)
	assert.Empty(t, config.Tools.Voice.DefaultDevice)
	assert.Equal(t, 16000, config.Tools.Voice.SampleRate)
	assert.Equal(t, 1, config.Tools.Voice.Channels)
	assert.Equal(t, 16, config.Tools.Voice.BitDepth)
	assert.Equal(t, "wav", config.Tools.Voice.Format)
	assert.True(t, config.Tools.Voice.VoiceActivityDetection)
	assert.Equal(t, -40, config.Tools.Voice.SilenceThreshold)
	assert.Equal(t, 1*time.Second, config.Tools.Voice.MinRecordingDuration)
	assert.Equal(t, 30*time.Second, config.Tools.Voice.MaxRecordingDuration)
	assert.True(t, config.Tools.Voice.TranscriptionEnabled)
	assert.Equal(t, "openai", config.Tools.Voice.TranscriptionProvider)
	assert.Equal(t, "en", config.Tools.Voice.TranscriptionLanguage)
	assert.Equal(t, "whisper-1", config.Tools.Voice.TranscriptionModel)
	assert.Equal(t, []string{"en", "es", "fr", "de", "it", "pt", "zh", "ja"}, config.Tools.Voice.SupportedLanguages)
	assert.False(t, config.Tools.Voice.LocalProcessing)
	assert.False(t, config.Tools.Voice.StoreRecordings)
	assert.Equal(t, "~/.helixcode/recordings", config.Tools.Voice.RecordingPath)

	assert.True(t, config.Tools.CodeAnalysis.Enabled)
	assert.Equal(t, []string{"go", "python", "javascript", "typescript", "java", "c++", "rust"}, config.Tools.CodeAnalysis.SupportedLanguages)
	assert.True(t, config.Tools.CodeAnalysis.SyntaxAnalysis)
	assert.True(t, config.Tools.CodeAnalysis.SemanticAnalysis)
	assert.True(t, config.Tools.CodeAnalysis.DependencyAnalysis)
	assert.False(t, config.Tools.CodeAnalysis.SecurityAnalysis)
	assert.False(t, config.Tools.CodeAnalysis.PerformanceAnalysis)
	assert.True(t, config.Tools.CodeAnalysis.TreeSitterEnabled)
	assert.Equal(t, "~/.helixcode/cache/parsers", config.Tools.CodeAnalysis.ParserCachePath)
	assert.Equal(t, int64(100*1024*1024), config.Tools.CodeAnalysis.MaxCacheSize)
	assert.Equal(t, 50, config.Tools.CodeAnalysis.MaxContextFiles)
	assert.Equal(t, int64(1024*1024), config.Tools.CodeAnalysis.MaxContextSize)
	assert.Equal(t, "hybrid", config.Tools.CodeAnalysis.ContextStrategy)
	assert.True(t, config.Tools.CodeAnalysis.IndexEnabled)
	assert.Equal(t, "~/.helixcode/index", config.Tools.CodeAnalysis.IndexPath)
	assert.Equal(t, 3600*time.Second, config.Tools.CodeAnalysis.IndexUpdateInterval)

	assert.True(t, config.Tools.Git.Enabled)
	assert.Equal(t, "main", config.Tools.Git.DefaultBranch)
	assert.True(t, config.Tools.Git.AutoCommitEnabled)
	assert.True(t, config.Tools.Git.AutoCommitMessage)
	assert.Equal(t, "llm", config.Tools.Git.CommitMessageProvider)
	assert.True(t, config.Tools.Git.AutoStageTracked)
	assert.False(t, config.Tools.Git.AutoStageNew)
	assert.False(t, config.Tools.Git.CreateBranchOnCommit)
	assert.Equal(t, "feature/{}", config.Tools.Git.BranchNamingPattern)
	assert.False(t, config.Tools.Git.GitHubIntegration)
	assert.False(t, config.Tools.Git.GitLabIntegration)
	assert.False(t, config.Tools.Git.BitbucketIntegration)
	assert.False(t, config.Tools.Git.SignedCommits)
	assert.Empty(t, config.Tools.Git.GPGKeyPath)

	assert.True(t, config.Tools.MultiEdit.Enabled)
	assert.Equal(t, 50, config.Tools.MultiEdit.MaxFiles)
	assert.Equal(t, int64(10*1024*1024), config.Tools.MultiEdit.MaxFileSize)
	assert.True(t, config.Tools.MultiEdit.Transactional)
	assert.True(t, config.Tools.MultiEdit.AutoBackup)
	assert.Equal(t, "~/.helixcode/backups/edits", config.Tools.MultiEdit.BackupPath)
	assert.True(t, config.Tools.MultiEdit.RollbackEnabled)
	assert.True(t, config.Tools.MultiEdit.PreviewEnabled)
	assert.Equal(t, "unified", config.Tools.MultiEdit.DiffFormat)
	assert.Equal(t, "manual", config.Tools.MultiEdit.ConflictStrategy)
	assert.Equal(t, 10, config.Tools.MultiEdit.BatchSize)
	assert.Equal(t, 30*time.Second, config.Tools.MultiEdit.BatchTimeout)

	assert.True(t, config.Tools.Confirmation.Enabled)
	assert.True(t, config.Tools.Confirmation.InteractiveMode)
	assert.Len(t, config.Tools.Confirmation.Levels, 3)
	assert.Contains(t, config.Tools.Confirmation.Levels, "info")
	assert.Contains(t, config.Tools.Confirmation.Levels, "warning")
	assert.Contains(t, config.Tools.Confirmation.Levels, "danger")
	assert.Len(t, config.Tools.Confirmation.Policies, 2)
	assert.Contains(t, config.Tools.Confirmation.Policies, "file_delete")
	assert.Contains(t, config.Tools.Confirmation.Policies, "shell_sudo")
	assert.True(t, config.Tools.Confirmation.AuditEnabled)
	assert.Equal(t, "~/.helixcode/logs/audit.log", config.Tools.Confirmation.AuditLogPath)
	assert.Equal(t, 30, config.Tools.Confirmation.AuditRetention)
	assert.True(t, config.Tools.Confirmation.ShowReason)
	assert.True(t, config.Tools.Confirmation.ShowImpact)
	assert.True(t, config.Tools.Confirmation.ShowAlternatives)

	// Workflows section
	assert.True(t, config.Workflows.Enabled)
	assert.Equal(t, "plan", config.Workflows.DefaultMode)
	assert.True(t, config.Workflows.PlanMode.Enabled)
	assert.True(t, config.Workflows.PlanMode.TwoPhase)
	assert.True(t, config.Workflows.PlanMode.ShowOptions)
	assert.Equal(t, 5, config.Workflows.PlanMode.MaxOptions)
	assert.True(t, config.Workflows.PlanMode.RequireConfirmation)
	assert.Equal(t, "comprehensive", config.Workflows.PlanMode.Strategy)
	assert.Equal(t, 10, config.Workflows.PlanMode.MaxPlanComplexity)
	assert.True(t, config.Workflows.PlanMode.TaskBreakdown)
	assert.True(t, config.Workflows.PlanMode.DependencyAnalysis)

	assert.True(t, config.Workflows.Autonomy.Enabled)
	assert.Equal(t, "basic_plus", config.Workflows.Autonomy.DefaultLevel)
	assert.Len(t, config.Workflows.Autonomy.Levels, 5)
	assert.Contains(t, config.Workflows.Autonomy.Levels, "full")
	assert.Contains(t, config.Workflows.Autonomy.Levels, "semi")
	assert.Contains(t, config.Workflows.Autonomy.Levels, "basic_plus")
	assert.Contains(t, config.Workflows.Autonomy.Levels, "basic")
	assert.Contains(t, config.Workflows.Autonomy.Levels, "none")
	assert.True(t, config.Workflows.Autonomy.AutoContext)
	assert.Equal(t, 8192, config.Workflows.Autonomy.ContextLimits["max_tokens"])
	assert.Equal(t, 20, config.Workflows.Autonomy.ContextLimits["max_files"])
	assert.True(t, config.Workflows.Autonomy.SafetyChecks)
	assert.True(t, config.Workflows.Autonomy.FailsafeEnabled)
	assert.True(t, config.Workflows.Autonomy.EmergencyStop)

	assert.True(t, config.Workflows.Snapshots.Enabled)
	assert.True(t, config.Workflows.Snapshots.AutoSnapshot)
	assert.Equal(t, "~/.helixcode/snapshots", config.Workflows.Snapshots.StorageLocation)
	assert.True(t, config.Workflows.Snapshots.IncludeGitState)
	assert.True(t, config.Workflows.Snapshots.IncludeDependencies)
	assert.True(t, config.Workflows.Snapshots.IncludeEnvironment)
	assert.True(t, config.Workflows.Snapshots.IncludeConfig)
	assert.Equal(t, "smart", config.Workflows.Snapshots.RetentionPolicy)
	assert.Equal(t, 100, config.Workflows.Snapshots.MaxSnapshots)
	assert.Equal(t, 30, config.Workflows.Snapshots.RetentionDays)
	assert.Equal(t, "git", config.Workflows.Snapshots.DiffTool)
	assert.True(t, config.Workflows.Snapshots.ShowChanges)
	assert.True(t, config.Workflows.Snapshots.ShowMetadata)

	assert.Empty(t, config.Workflows.Workflows)
	assert.True(t, config.Workflows.Integration.GitEnabled)
	assert.False(t, config.Workflows.Integration.GitAutoPush)
	assert.Equal(t, "helix-workflow", config.Workflows.Integration.GitBranchName)
	assert.False(t, config.Workflows.Integration.CIIntegrationEnabled)
	assert.Empty(t, config.Workflows.Integration.CISystems)
	assert.False(t, config.Workflows.Integration.WebhooksEnabled)
	assert.Empty(t, config.Workflows.Integration.WebhookURLs)
	assert.False(t, config.Workflows.Integration.NotifyOnStart)
	assert.False(t, config.Workflows.Integration.NotifyOnComplete)
	assert.False(t, config.Workflows.Integration.NotifyOnFailure)

	// UI section
	assert.Equal(t, "dark", config.UI.Theme)
	assert.Equal(t, "en", config.UI.Language)
	assert.Equal(t, "SF Mono", config.UI.FontFamily)
	assert.Equal(t, 14, config.UI.FontSize)

	// Window settings
	assert.Equal(t, 1200, config.UI.WindowSettings.DefaultWidth)
	assert.Equal(t, 800, config.UI.WindowSettings.DefaultHeight)
	assert.Equal(t, 800, config.UI.WindowSettings.MinWidth)
	assert.Equal(t, 600, config.UI.WindowSettings.MinHeight)
	assert.Equal(t, 0, config.UI.WindowSettings.MaxWidth)  // unlimited
	assert.Equal(t, 0, config.UI.WindowSettings.MaxHeight) // unlimited
	assert.True(t, config.UI.WindowSettings.RememberSize)
	assert.True(t, config.UI.WindowSettings.RememberPosition)
	assert.Equal(t, "center", config.UI.WindowSettings.StartupPosition)
	assert.Equal(t, [2]int{0, 0}, config.UI.WindowSettings.DefaultPosition)

	// Editor settings
	assert.Equal(t, 4, config.UI.Editor.TabSize)
	assert.True(t, config.UI.Editor.InsertSpaces)
	assert.True(t, config.UI.Editor.WordWrap)
	assert.True(t, config.UI.Editor.LineNumbers)
	assert.True(t, config.UI.Editor.HighlightLine)
	assert.True(t, config.UI.Editor.AutoIndent)
	assert.False(t, config.UI.Editor.ShowWhitespace)
	assert.False(t, config.UI.Editor.ShowMinimap)
	assert.True(t, config.UI.Editor.SyntaxHighlighting)
	assert.Equal(t, "dark", config.UI.Editor.ColorScheme)
	assert.True(t, config.UI.Editor.AutoCompletion)
	assert.True(t, config.UI.Editor.AutoSuggestion)
	assert.True(t, config.UI.Editor.SnippetEnabled)
	assert.False(t, config.UI.Editor.CodeFolding)
	assert.True(t, config.UI.Editor.AutoSave)
	assert.Equal(t, 300, config.UI.Editor.AutoSaveInterval)
	assert.False(t, config.UI.Editor.CaseSensitiveSearch)
	assert.True(t, config.UI.Editor.RegexSearch)
	assert.True(t, config.UI.Editor.IncrementalSearch)

	// Terminal settings
	assert.Equal(t, "/bin/bash", config.UI.Terminal.Shell)
	assert.Equal(t, 10000, config.UI.Terminal.ScrollbackLines)
	assert.Equal(t, 12, config.UI.Terminal.FontSize)
	assert.Equal(t, "SF Mono", config.UI.Terminal.FontFamily)
	assert.Equal(t, "#ffffff", config.UI.Terminal.ForegroundColor)
	assert.Equal(t, "#000000", config.UI.Terminal.BackgroundColor)
	assert.Equal(t, "#ffffff", config.UI.Terminal.CursorColor)
	assert.Equal(t, "dark", config.UI.Terminal.ColorScheme)
	assert.Equal(t, 0.0, config.UI.Terminal.Transparency)
	assert.Equal(t, 0.0, config.UI.Terminal.Blurriness)
	assert.False(t, config.UI.Terminal.AlwaysOnTop)
	assert.False(t, config.UI.Terminal.HideScrollbar)
	assert.True(t, config.UI.Terminal.EnableBell)
	assert.False(t, config.UI.Terminal.CopyOnSelect)
	assert.False(t, config.UI.Terminal.PasteOnMiddleClick)

	// Accessibility settings
	assert.False(t, config.UI.Accessibility.Enabled)
	assert.False(t, config.UI.Accessibility.HighContrast)
	assert.False(t, config.UI.Accessibility.LargeFonts)
	assert.False(t, config.UI.Accessibility.ScreenReader)
	assert.True(t, config.UI.Accessibility.KeyboardNavigation)
	assert.False(t, config.UI.Accessibility.ReduceMotion)
	assert.True(t, config.UI.Accessibility.FocusVisible)

	// Platform UI settings
	assert.Len(t, config.UI.PlatformUI, 4)
	assert.Contains(t, config.UI.PlatformUI, "desktop")
	assert.Contains(t, config.UI.PlatformUI, "web")
	assert.Contains(t, config.UI.PlatformUI, "mobile")
	assert.Contains(t, config.UI.PlatformUI, "tui")

	// Desktop UI
	assert.True(t, config.UI.PlatformUI["desktop"].MenuBar)
	assert.True(t, config.UI.PlatformUI["desktop"].ToolBar)
	assert.True(t, config.UI.PlatformUI["desktop"].StatusBar)
	assert.True(t, config.UI.PlatformUI["desktop"].SideBar)
	assert.False(t, config.UI.PlatformUI["desktop"].FullscreenMode)
	assert.False(t, config.UI.PlatformUI["desktop"].CompactMode)
	assert.False(t, config.UI.PlatformUI["desktop"].TouchOptimized)
	assert.Equal(t, []string{"file_browser", "terminal", "editor"}, config.UI.PlatformUI["desktop"].Features)

	// Web UI
	assert.True(t, config.UI.PlatformUI["web"].MenuBar)
	assert.True(t, config.UI.PlatformUI["web"].ToolBar)
	assert.True(t, config.UI.PlatformUI["web"].StatusBar)
	assert.True(t, config.UI.PlatformUI["web"].SideBar)
	assert.True(t, config.UI.PlatformUI["web"].FullscreenMode)
	assert.False(t, config.UI.PlatformUI["web"].CompactMode)
	assert.True(t, config.UI.PlatformUI["web"].TouchOptimized)
	assert.Equal(t, []string{"responsive_design", "pwa"}, config.UI.PlatformUI["web"].Features)

	// Mobile UI
	assert.False(t, config.UI.PlatformUI["mobile"].MenuBar)
	assert.True(t, config.UI.PlatformUI["mobile"].ToolBar)
	assert.True(t, config.UI.PlatformUI["mobile"].StatusBar)
	assert.False(t, config.UI.PlatformUI["mobile"].SideBar)
	assert.True(t, config.UI.PlatformUI["mobile"].FullscreenMode)
	assert.True(t, config.UI.PlatformUI["mobile"].CompactMode)
	assert.True(t, config.UI.PlatformUI["mobile"].TouchOptimized)
	assert.Equal(t, []string{"gestures", "offline_support"}, config.UI.PlatformUI["mobile"].Features)

	// TUI UI
	assert.False(t, config.UI.PlatformUI["tui"].MenuBar)
	assert.False(t, config.UI.PlatformUI["tui"].ToolBar)
	assert.True(t, config.UI.PlatformUI["tui"].StatusBar)
	assert.False(t, config.UI.PlatformUI["tui"].SideBar)
	assert.True(t, config.UI.PlatformUI["tui"].FullscreenMode)
	assert.True(t, config.UI.PlatformUI["tui"].CompactMode)
	assert.False(t, config.UI.PlatformUI["tui"].TouchOptimized)
	assert.Equal(t, []string{"keyboard_shortcuts", "mouse_support"}, config.UI.PlatformUI["tui"].Features)

	// Notifications section
	assert.True(t, config.Notifications.Enabled)
	assert.Len(t, config.Notifications.Channels, 3)
	assert.Contains(t, config.Notifications.Channels, "desktop")
	assert.Contains(t, config.Notifications.Channels, "email")
	assert.Contains(t, config.Notifications.Channels, "slack")
	assert.Len(t, config.Notifications.Rules, 2)
	assert.Equal(t, "default", config.Notifications.DefaultSound)
	assert.Equal(t, "normal", config.Notifications.DefaultUrgency)
	assert.False(t, config.Notifications.QuietHoursEnabled)
	assert.Equal(t, "22:00", config.Notifications.QuietHoursStart)
	assert.Equal(t, "08:00", config.Notifications.QuietHoursEnd)
	assert.Equal(t, []string{"saturday", "sunday"}, config.Notifications.QuietHoursDays)
	assert.False(t, config.Notifications.DoNotDisturb)
	assert.Empty(t, config.Notifications.DoNotDisturbUntil)
	assert.True(t, config.Notifications.AggregateNotifications)
	assert.Equal(t, 5, config.Notifications.MaxAggregatedItems)
	assert.Equal(t, 300*time.Second, config.Notifications.AggregationTimeout)

	// Security section
	assert.True(t, config.Security.EncryptionEnabled)
	assert.Empty(t, config.Security.EncryptionKey)
	assert.Equal(t, []string{"password", "2fa"}, config.Security.Authentication.Methods)
	assert.Equal(t, 8, config.Security.Authentication.PasswordPolicy.MinLength)
	assert.True(t, config.Security.Authentication.PasswordPolicy.RequireUppercase)
	assert.True(t, config.Security.Authentication.PasswordPolicy.RequireLowercase)
	assert.True(t, config.Security.Authentication.PasswordPolicy.RequireNumbers)
	assert.False(t, config.Security.Authentication.PasswordPolicy.RequireSymbols)
	assert.Equal(t, 90*24*time.Hour, config.Security.Authentication.PasswordPolicy.MaxAge)
	assert.Equal(t, 5, config.Security.Authentication.PasswordPolicy.HistoryCount)
	assert.False(t, config.Security.Authentication.TwoFactorAuth.Enabled)
	assert.Equal(t, []string{"totp", "sms"}, config.Security.Authentication.TwoFactorAuth.Methods)
	assert.True(t, config.Security.Authentication.TwoFactorAuth.BackupCodes)
	assert.True(t, config.Security.Authentication.TwoFactorAuth.RememberDevice)
	assert.Equal(t, 30*24*time.Hour, config.Security.Authentication.TwoFactorAuth.RememberDeviceTTL)
	assert.Equal(t, 30*time.Minute, config.Security.Authentication.SessionTimeout)
	assert.Equal(t, 3, config.Security.Authentication.MaxConcurrentSessions)
	assert.True(t, config.Security.Authentication.LockoutPolicy.Enabled)
	assert.Equal(t, 5, config.Security.Authentication.LockoutPolicy.MaxAttempts)
	assert.Equal(t, 15*time.Minute, config.Security.Authentication.LockoutPolicy.WindowDuration)
	assert.Equal(t, 15*time.Minute, config.Security.Authentication.LockoutPolicy.LockoutDuration)
	assert.True(t, config.Security.Authentication.LockoutPolicy.Progressive)

	assert.True(t, config.Security.Authorization.Enabled)
	assert.Equal(t, "deny", config.Security.Authorization.DefaultPolicy)
	assert.True(t, config.Security.Authorization.RBAC)
	assert.Empty(t, config.Security.Authorization.Roles)
	assert.Empty(t, config.Security.Authorization.Permissions)
	assert.Empty(t, config.Security.Authorization.Policies)

	assert.True(t, config.Security.DataProtection.EncryptionAtRest)
	assert.True(t, config.Security.DataProtection.EncryptionInTransit)
	assert.Equal(t, 90*24*time.Hour, config.Security.DataProtection.KeyRotation)
	assert.True(t, config.Security.DataProtection.RetentionPolicy.Enabled)
	assert.Equal(t, 365*24*time.Hour, config.Security.DataProtection.RetentionPolicy.DefaultRetention)
	assert.Empty(t, config.Security.DataProtection.RetentionPolicy.SpecificRetention)
	assert.False(t, config.Security.DataProtection.RetentionPolicy.AutoDelete)
	assert.Equal(t, 30*24*time.Hour, config.Security.DataProtection.RetentionPolicy.NotificationPeriod)
	assert.False(t, config.Security.DataProtection.MaskingEnabled)
	assert.Empty(t, config.Security.DataProtection.MaskedFields)
	assert.True(t, config.Security.DataProtection.BackupEncryption)
	assert.Equal(t, 90*24*time.Hour, config.Security.DataProtection.BackupKeyRotation)

	assert.False(t, config.Security.Network.FirewallEnabled)
	assert.Empty(t, config.Security.Network.AllowedIPs)
	assert.Empty(t, config.Security.Network.BlockedIPs)
	assert.Equal(t, []int{8080, 22}, config.Security.Network.AllowedPorts)
	assert.Empty(t, config.Security.Network.BlockedPorts)
	assert.False(t, config.Security.Network.TLSEnabled)
	assert.Equal(t, "1.3", config.Security.Network.TLSVersion)
	assert.Empty(t, config.Security.Network.CipherSuites)
	assert.False(t, config.Security.Network.VPNEnabled)
	assert.Empty(t, config.Security.Network.VPNProvider)
	assert.Empty(t, config.Security.Network.VPNConfig)

	assert.True(t, config.Security.Audit.Enabled)
	assert.Equal(t, "info", config.Security.Audit.LogLevel)
	assert.Equal(t, "~/.helixcode/logs/audit.log", config.Security.Audit.LogPath)
	assert.Equal(t, int64(100*1024*1024), config.Security.Audit.MaxLogSize)
	assert.Equal(t, 90, config.Security.Audit.LogRetention)
	assert.Equal(t, []string{"login", "logout", "config_change", "task_execution"}, config.Security.Audit.Events)
	assert.False(t, config.Security.Audit.RealTimeEnabled)
	assert.Empty(t, config.Security.Audit.AlertEndpoints)

	assert.False(t, config.Security.Privacy.DataCollectionEnabled)
	assert.False(t, config.Security.Privacy.AnalyticsEnabled)
	assert.True(t, config.Security.Privacy.ConsentRequired)
	assert.Equal(t, "1.0", config.Security.Privacy.ConsentVersion)
	assert.False(t, config.Security.Privacy.DataSharingEnabled)
	assert.Empty(t, config.Security.Privacy.SharedDataTypes)
	assert.True(t, config.Security.Privacy.AnonymizeData)
	assert.Equal(t, "hashing", config.Security.Privacy.AnonymizationMethod)
	assert.True(t, config.Security.Privacy.RightToDeletion)
	assert.Equal(t, "secure_erase", config.Security.Privacy.DataDeletionMethod)

	// Development section
	assert.False(t, config.Development.Enabled)
	assert.Equal(t, "development", config.Development.Environment)
	assert.False(t, config.Development.Debug.Enabled)
	assert.Equal(t, "debug", config.Development.Debug.Level)
	assert.False(t, config.Development.Debug.Verbose)
	assert.False(t, config.Development.Debug.TraceEnabled)
	assert.Empty(t, config.Development.Debug.Breakpoints)
	assert.True(t, config.Development.Debug.OutputToFile)
	assert.True(t, config.Development.Debug.OutputToConsole)
	assert.Equal(t, "~/.helixcode/logs/debug.log", config.Development.Debug.OutputPath)
	assert.Empty(t, config.Development.Debug.FeatureFlags)

	assert.True(t, config.Development.Testing.Enabled)
	assert.Equal(t, []string{"unit", "integration"}, config.Development.Testing.TestTypes)
	assert.True(t, config.Development.Testing.ParallelExecution)
	assert.Equal(t, 10, config.Development.Testing.MaxParallelTests)
	assert.Equal(t, 30*time.Minute, config.Development.Testing.Timeout)
	assert.Equal(t, "~/.helixcode/test_data", config.Development.Testing.TestDataPath)
	assert.True(t, config.Development.Testing.CleanupAfterTest)
	assert.True(t, config.Development.Testing.CoverageEnabled)
	assert.Equal(t, 80, config.Development.Testing.CoverageThreshold)
	assert.Equal(t, "~/.helixcode/coverage", config.Development.Testing.CoverageReportPath)
	assert.True(t, config.Development.Testing.MockingEnabled)
	assert.Empty(t, config.Development.Testing.MockProviders)

	assert.False(t, config.Development.Profiling.Enabled)
	assert.Equal(t, []string{"cpu", "memory"}, config.Development.Profiling.ProfileTypes)
	assert.Equal(t, "~/.helixcode/profiles", config.Development.Profiling.OutputPath)
	assert.Equal(t, "pprof", config.Development.Profiling.OutputFormat)
	assert.Equal(t, 100.0, config.Development.Profiling.SamplingRate)
	assert.Equal(t, 5*time.Minute, config.Development.Profiling.MaxDuration)
	assert.True(t, config.Development.Profiling.AutoAnalysis)
	assert.Equal(t, "go tool pprof", config.Development.Profiling.AnalysisTool)

	assert.True(t, config.Development.HotReload.Enabled)
	assert.Equal(t, []string{"./config", "./internal"}, config.Development.HotReload.WatchPaths)
	assert.Equal(t, []string{"*.tmp", "*.log"}, config.Development.HotReload.IgnorePatterns)
	assert.True(t, config.Development.HotReload.TriggerOnFileChange)
	assert.True(t, config.Development.HotReload.TriggerOnConfigChange)
	assert.True(t, config.Development.HotReload.RestartServer)
	assert.True(t, config.Development.HotReload.ReloadConfig)
	assert.True(t, config.Development.HotReload.RefreshUI)
	assert.Equal(t, 1*time.Second, config.Development.HotReload.DebounceDelay)
	assert.Equal(t, 5*time.Second, config.Development.HotReload.RestartDelay)

	assert.Equal(t, "debug", config.Development.Logging.Level)
	assert.Equal(t, "structured", config.Development.Logging.Format)
	assert.Equal(t, "both", config.Development.Logging.Output)
	assert.Empty(t, config.Development.Logging.ModuleLevels)
	assert.True(t, config.Development.Logging.StructuredLogging)
	assert.True(t, config.Development.Logging.CorrelationID)
	assert.True(t, config.Development.Logging.StackTraces)
	assert.True(t, config.Development.Logging.LogPerformance)
	assert.True(t, config.Development.Logging.LogSlowQueries)
	assert.Equal(t, 1*time.Second, config.Development.Logging.SlowQueryThreshold)

	// Platform section
	assert.Equal(t, "desktop", config.Platform.CurrentPlatform)

	// Desktop platform
	assert.True(t, config.Platform.Desktop.Enabled)
	assert.False(t, config.Platform.Desktop.AutoStart)
	assert.True(t, config.Platform.Desktop.MinimizeToTray)
	assert.True(t, config.Platform.Desktop.ShowInTaskbar)
	assert.Empty(t, config.Platform.Desktop.FileAssociations)
	assert.True(t, config.Platform.Desktop.ContextMenuEnabled)
	assert.True(t, config.Platform.Desktop.AutoUpdate)
	assert.True(t, config.Platform.Desktop.HardwareAcceleration)
	assert.True(t, config.Platform.Desktop.GPUAcceleration)
	assert.Equal(t, int64(2*1024*1024*1024), config.Platform.Desktop.MemoryLimit)
	assert.Equal(t, 1.0, config.Platform.Desktop.UIScale)
	assert.True(t, config.Platform.Desktop.HighDPI)

	// Web platform
	assert.True(t, config.Platform.Web.Enabled)
	assert.Equal(t, "localhost", config.Platform.Web.Host)
	assert.Equal(t, 3000, config.Platform.Web.Port)
	assert.Equal(t, "/", config.Platform.Web.BasePath)
	assert.Equal(t, "./static", config.Platform.Web.StaticPath)
	assert.True(t, config.Platform.Web.CacheEnabled)
	assert.Equal(t, 3600*time.Second, config.Platform.Web.CacheTTL)
	assert.True(t, config.Platform.Web.PWAEnabled)
	assert.True(t, config.Platform.Web.OfflineEnabled)
	assert.True(t, config.Platform.Web.CSProtection)
	assert.True(t, config.Platform.Web.XSSProtection)
	assert.True(t, config.Platform.Web.CompressionEnabled)
	assert.True(t, config.Platform.Web.MinifyEnabled)
	assert.True(t, config.Platform.Web.RealTimeUpdates)
	assert.True(t, config.Platform.Web.WebSocketEnabled)

	// Mobile platform
	assert.True(t, config.Platform.Mobile.Enabled)

	// iOS platform
	assert.True(t, config.Platform.Mobile.IOS.Enabled)
	assert.False(t, config.Platform.Mobile.IOS.AppStoreConnect)
	assert.False(t, config.Platform.Mobile.IOS.PushNotifications)
	assert.True(t, config.Platform.Mobile.IOS.BackgroundTasks)
	assert.False(t, config.Platform.Mobile.IOS.WatchKitApp)
	assert.Empty(t, config.Platform.Mobile.IOS.TeamID)
	assert.Equal(t, "dev.helix.code", config.Platform.Mobile.IOS.BundleID)
	assert.False(t, config.Platform.Mobile.IOS.DevelopmentCert)

	// Android platform
	assert.True(t, config.Platform.Mobile.Android.Enabled)
	assert.False(t, config.Platform.Mobile.Android.GooglePlayConsole)
	assert.False(t, config.Platform.Mobile.Android.PushNotifications)
	assert.True(t, config.Platform.Mobile.Android.BackgroundTasks)
	assert.False(t, config.Platform.Mobile.Android.WearOSApp)
	assert.Equal(t, "dev.helix.code", config.Platform.Mobile.Android.PackageName)
	assert.True(t, config.Platform.Mobile.Android.SigningEnabled)
	assert.True(t, config.Platform.Mobile.Android.DebugBuild)

	// Cross-platform mobile
	assert.Equal(t, "gomobile", config.Platform.Mobile.CrossPlatform.Framework)
	assert.True(t, config.Platform.Mobile.CrossPlatform.OfflineFirst)
	assert.True(t, config.Platform.Mobile.CrossPlatform.SyncEnabled)
	assert.True(t, config.Platform.Mobile.CrossPlatform.ImageOptimization)
	assert.True(t, config.Platform.Mobile.CrossPlatform.LazyLoading)
	assert.True(t, config.Platform.Mobile.CrossPlatform.BiometricAuth)
	assert.True(t, config.Platform.Mobile.CrossPlatform.DeviceEncryption)

	// TUI platform
	assert.True(t, config.Platform.TUI.Enabled)
	assert.Equal(t, "auto", config.Platform.TUI.CompatibilityMode)
	assert.Equal(t, "dark", config.Platform.TUI.ColorScheme)
	assert.True(t, config.Platform.TUI.TrueColor)
	assert.True(t, config.Platform.TUI.MouseEnabled)
	assert.Equal(t, 60, config.Platform.TUI.RenderFPS)
	assert.Equal(t, 10000, config.Platform.TUI.BufferSize)
	assert.True(t, config.Platform.TUI.StatusLine)
	assert.True(t, config.Platform.TUI.TabBar)
	assert.True(t, config.Platform.TUI.SplitScreen)

	// Aurora OS platform
	assert.False(t, config.Platform.AuroraOS.Enabled)
	assert.True(t, config.Platform.AuroraOS.SailfishIntegration)
	assert.False(t, config.Platform.AuroraOS.StoreIntegration)
	assert.Equal(t, "4.4", config.Platform.AuroraOS.SDKVersion)
	assert.True(t, config.Platform.AuroraOS.NativeUI)
	assert.True(t, config.Platform.AuroraOS.QtComponents)

	// Harmony OS platform
	assert.False(t, config.Platform.HarmonyOS.Enabled)
	assert.True(t, config.Platform.HarmonyOS.HarmonyServices)
	assert.False(t, config.Platform.HarmonyOS.AppGallery)
	assert.False(t, config.Platform.HarmonyOS.DevEcoStudio)
	assert.Equal(t, "5.0", config.Platform.HarmonyOS.SDKVersion)
	assert.True(t, config.Platform.HarmonyOS.ArkUI)

	// Cross-platform settings
	assert.True(t, config.Platform.CrossPlatform.ConsistentTheme)
	assert.True(t, config.Platform.CrossPlatform.SyncConfig)
	assert.True(t, config.Platform.CrossPlatform.SyncData)
	assert.True(t, config.Platform.CrossPlatform.UpdateAcrossPlatforms)
	assert.True(t, config.Platform.CrossPlatform.CommonFeatureSet)
	assert.True(t, config.Platform.CrossPlatform.PlatformOptimizations)
}

// TestConfigWatcher implementation for testing
type TestConfigWatcher struct {
	OnChangeFunc func(old, new *HelixConfig) error
}

func (t *TestConfigWatcher) OnConfigChange(old, new *HelixConfig) error {
	if t.OnChangeFunc != nil {
		return t.OnChangeFunc(old, new)
	}
	return nil
}

// BenchmarkHelixConfigManager benchmarks configuration operations
func BenchmarkHelixConfigManager(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "bench_config.json")

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(b, err)

	b.Run("Load", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := NewHelixConfigManager(configPath)
			require.NoError(b, err)
		}
	})

	b.Run("Save", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := manager.UpdateConfig(func(c *HelixConfig) {
				c.Application.Name = fmt.Sprintf("Benchmark %d", i)
			})
			require.NoError(b, err)
		}
	})

	b.Run("GetConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = manager.GetConfig()
		}
	})
}

// TestHelixConfigConcurrentAccess tests concurrent configuration access
func TestHelixConfigConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "concurrent_config.json")

	manager, err := NewHelixConfigManager(configPath)
	require.NoError(t, err)

	// Test concurrent reads
	const numReaders = 10
	readDone := make(chan bool, numReaders)

	for i := 0; i < numReaders; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				config := manager.GetConfig()
				assert.NotNil(t, config)
			}
			readDone <- true
		}()
	}

	// Test concurrent writes
	const numWriters = 5
	writeDone := make(chan bool, numWriters)

	for i := 0; i < numWriters; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				err := manager.UpdateConfig(func(c *HelixConfig) {
					c.Application.Name = fmt.Sprintf("Concurrent %d-%d", id, j)
				})
				assert.NoError(t, err)
			}
			writeDone <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < numReaders; i++ {
		<-readDone
	}
	for i := 0; i < numWriters; i++ {
		<-writeDone
	}

	// Verify final state is consistent
	finalConfig := manager.GetConfig()
	assert.NotEmpty(t, finalConfig.Application.Name)
}

// TestHelixConfigErrorHandling tests error handling in configuration operations
func TestHelixConfigErrorHandling(t *testing.T) {
	// Test with invalid JSON path
	_, err := NewHelixConfigManager("/invalid/path/test.json")
	assert.Error(t, err)

	// Test with invalid JSON content
	tempDir := t.TempDir()
	invalidConfigPath := filepath.Join(tempDir, "invalid.json")

	err = os.WriteFile(invalidConfigPath, []byte("invalid json content"), 0644)
	require.NoError(t, err)

	_, err = NewHelixConfigManager(invalidConfigPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}
