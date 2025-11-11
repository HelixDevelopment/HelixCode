package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HelixConfigManager manages ~/.config/helix/helix.json configuration
type HelixConfigManager struct {
	mu         sync.RWMutex
	config     *HelixConfig
	configPath string
	watchers   []ConfigWatcher
	isWatching bool
	version    string
}

// ConfigWatcher represents a configuration change watcher
type ConfigWatcher interface {
	OnConfigChange(oldConfig, newConfig *HelixConfig) error
}

// NewHelixConfigManager creates a new configuration manager for ~/.config/helix/helix.json
func NewHelixConfigManager(configPath string) (*HelixConfigManager, error) {
	if configPath == "" {
		configPath = GetHelixConfigPath()
	}

	manager := &HelixConfigManager{
		configPath: configPath,
		watchers:   make([]ConfigWatcher, 0),
		version:    "1.0.0",
	}

	// Load existing config or create default
	if err := manager.Load(); err != nil {
		// If file doesn't exist, create default config
		if os.IsNotExist(err) {
			if err := manager.CreateDefault(); err != nil {
				return nil, fmt.Errorf("failed to create default config: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load config: %v", err)
		}
	}

	return manager, nil
}

// GetHelixConfigPath returns the standard configuration path: ~/.config/helix/helix.json
func GetHelixConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "./helix.json"
	}
	return filepath.Join(home, ".config", "helix", "helix.json")
}

// Load loads configuration from ~/.config/helix/helix.json
func (m *HelixConfigManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var config HelixConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Validate configuration
	if err := m.validateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	m.config = &config
	return nil
}

// Save saves configuration to ~/.config/helix/helix.json
func (m *HelixConfigManager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	// Update metadata
	m.config.LastUpdated = time.Now()
	m.config.Version = m.version

	// Validate before saving
	if err := m.validateConfig(m.config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// GetConfig returns a copy of current configuration
func (m *HelixConfigManager) GetConfig() *HelixConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	if m.config == nil {
		return nil
	}

	data, _ := json.Marshal(m.config)
	var copy HelixConfig
	json.Unmarshal(data, &copy)

	return &copy
}

// UpdateConfig updates configuration with atomic save and watcher notifications
func (m *HelixConfigManager) UpdateConfig(updater func(*HelixConfig)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldConfig := m.copyConfig()

	// Apply updates
	updater(m.config)

	// Validate new configuration
	if err := m.validateConfig(m.config); err != nil {
		// Restore old config
		m.config = oldConfig
		return fmt.Errorf("invalid configuration update: %v", err)
	}

	// Save to file
	if err := m.saveLocked(); err != nil {
		// Restore old config
		m.config = oldConfig
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	// Notify watchers
	m.notifyWatchers(oldConfig, m.config)

	return nil
}

// AddWatcher adds a configuration change watcher
func (m *HelixConfigManager) AddWatcher(watcher ConfigWatcher) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.watchers = append(m.watchers, watcher)
}

// CreateDefault creates a default configuration at ~/.config/helix/helix.json
func (m *HelixConfigManager) CreateDefault() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = m.getDefaultConfig()

	return m.saveLocked()
}

// GetDefaultConfig returns a default configuration
func (m *HelixConfigManager) GetDefaultConfig() *HelixConfig {
	return m.getDefaultConfig()
}

// ValidateConfig validates a configuration
func (m *HelixConfigManager) ValidateConfig(config *HelixConfig) error {
	return m.validateConfig(config)
}

// ExportConfig exports configuration to a specific file
func (m *HelixConfigManager) ExportConfig(exportPath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return fmt.Errorf("no configuration to export")
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(exportPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(exportPath, data, 0644)
}

// ImportConfig imports configuration from a file
func (m *HelixConfigManager) ImportConfig(importPath string) error {
	data, err := os.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %v", err)
	}

	var importedConfig HelixConfig
	if err := json.Unmarshal(data, &importedConfig); err != nil {
		return fmt.Errorf("failed to parse imported config: %v", err)
	}

	// Validate imported config
	if err := m.validateConfig(&importedConfig); err != nil {
		return fmt.Errorf("invalid imported configuration: %v", err)
	}

	return m.UpdateConfig(func(config *HelixConfig) {
		*config = importedConfig
	})
}

// ResetToDefaults resets configuration to defaults
func (m *HelixConfigManager) ResetToDefaults() error {
	return m.UpdateConfig(func(config *HelixConfig) {
		*config = *m.getDefaultConfig()
	})
}

// GetConfigPath returns the current configuration file path
func (m *HelixConfigManager) GetConfigPath() string {
	return m.configPath
}

// IsConfigPresent checks if configuration file exists
func (m *HelixConfigManager) IsConfigPresent() bool {
	_, err := os.Stat(m.configPath)
	return err == nil
}

// BackupConfig creates a backup of the current configuration
func (m *HelixConfigManager) BackupConfig(backupPath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return fmt.Errorf("no configuration to backup")
	}

	if backupPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		backupPath = filepath.Join(filepath.Dir(m.configPath), fmt.Sprintf("helix_backup_%s.json", timestamp))
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(backupPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

// RestoreConfig restores configuration from a backup file
func (m *HelixConfigManager) RestoreConfig(backupPath string) error {
	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Parse backup configuration
	var backupConfig HelixConfig
	if err := json.Unmarshal(data, &backupConfig); err != nil {
		return fmt.Errorf("failed to parse backup configuration: %w", err)
	}

	// Validate backup configuration
	if err := m.validateConfig(&backupConfig); err != nil {
		return fmt.Errorf("backup configuration validation failed: %w", err)
	}

	// Apply restored configuration
	m.mu.Lock()
	defer m.mu.Unlock()

	oldConfig := m.copyConfig()
	m.config = &backupConfig

	// Save to current config path
	if err := m.saveLocked(); err != nil {
		// Rollback on save failure
		m.config = oldConfig
		return fmt.Errorf("failed to save restored configuration: %w", err)
	}

	// Notify watchers
	m.notifyWatchers(oldConfig, m.config)

	return nil
}

// ReloadConfig reloads configuration from disk
func (m *HelixConfigManager) ReloadConfig() error {
	// Read current config file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse configuration
	var newConfig HelixConfig
	if err := json.Unmarshal(data, &newConfig); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate configuration
	if err := m.validateConfig(&newConfig); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Apply reloaded configuration
	m.mu.Lock()
	defer m.mu.Unlock()

	oldConfig := m.copyConfig()
	m.config = &newConfig

	// Notify watchers
	m.notifyWatchers(oldConfig, m.config)

	return nil
}

// Helper methods for configuration management

func (m *HelixConfigManager) saveLocked() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

func (m *HelixConfigManager) copyConfig() *HelixConfig {
	if m.config == nil {
		return nil
	}

	data, _ := json.Marshal(m.config)
	var copy HelixConfig
	json.Unmarshal(data, &copy)

	return &copy
}

func (m *HelixConfigManager) notifyWatchers(oldConfig, newConfig *HelixConfig) {
	for _, watcher := range m.watchers {
		if err := watcher.OnConfigChange(oldConfig, newConfig); err != nil {
			// Log error but continue notifying other watchers
			fmt.Printf("Config watcher error: %v\n", err)
		}
	}
}

func (m *HelixConfigManager) validateConfig(config *HelixConfig) error {
	// Basic validation
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	// Validate server configuration
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate database configuration
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if config.Database.Port < 1 || config.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", config.Database.Port)
	}

	// Validate Redis configuration if enabled
	if config.Redis.Enabled {
		if config.Redis.Host == "" {
			return fmt.Errorf("redis host is required when redis is enabled")
		}

		if config.Redis.Port < 1 || config.Redis.Port > 65535 {
			return fmt.Errorf("invalid redis port: %d", config.Redis.Port)
		}
	}

	// Validate LLM configuration
	if config.LLM.DefaultProvider == "" {
		return fmt.Errorf("default LLM provider is required")
	}

	if config.LLM.MaxTokens < 1 {
		return fmt.Errorf("max tokens must be positive")
	}

	if config.LLM.Temperature < 0 || config.LLM.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	// Validate workspace configuration
	if config.Application.Workspace.DefaultPath == "" {
		return fmt.Errorf("default workspace path is required")
	}

	// More validation can be added here...

	return nil
}

// Global configuration manager instance
var globalConfigManager *HelixConfigManager
var globalConfigManagerOnce sync.Once

// GetGlobalConfigManager returns the global configuration manager instance
func GetGlobalConfigManager() *HelixConfigManager {
	globalConfigManagerOnce.Do(func() {
		manager, err := NewHelixConfigManager("")
		if err != nil {
			// Create a temporary manager with default config if loading fails
			manager, _ = NewHelixConfigManager("./helix_temp.json")
			manager.config = manager.getDefaultConfig()
		}
		globalConfigManager = manager
	})

	return globalConfigManager
}

// LoadHelixConfig loads the helix.json configuration from ~/.config/helix/helix.json
func LoadHelixConfig() (*HelixConfig, error) {
	manager := GetGlobalConfigManager()
	return manager.GetConfig(), nil
}

// SaveHelixConfig saves the helix.json configuration to ~/.config/helix/helix.json
func SaveHelixConfig(config *HelixConfig) error {
	manager := GetGlobalConfigManager()
	return manager.UpdateConfig(func(current *HelixConfig) {
		*current = *config
	})
}

// UpdateHelixConfig updates the helix.json configuration atomically
func UpdateHelixConfig(updater func(*HelixConfig)) error {
	manager := GetGlobalConfigManager()
	return manager.UpdateConfig(updater)
}

// IsHelixConfigPresent checks if the helix.json configuration file exists
func IsHelixConfigPresent() bool {
	_, err := os.Stat(GetHelixConfigPath())
	return err == nil
}

// CreateDefaultHelixConfig creates a default helix.json configuration
func CreateDefaultHelixConfig() error {
	manager, err := NewHelixConfigManager("")
	if err != nil {
		return err
	}

	return manager.CreateDefault()
}

// BackupHelixConfig creates a backup of the current helix.json configuration
func BackupHelixConfig(backupPath string) error {
	if backupPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		home, _ := os.UserHomeDir()
		backupPath = filepath.Join(home, ".config", "helix", fmt.Sprintf("helix_backup_%s.json", timestamp))
	}

	manager := GetGlobalConfigManager()
	return manager.BackupConfig(backupPath)
}

// ResetHelixConfigToDefaults resets configuration to defaults
func ResetHelixConfigToDefaults() error {
	manager := GetGlobalConfigManager()
	return manager.ResetToDefaults()
}

// AddHelixConfigWatcher adds a configuration change watcher
func AddHelixConfigWatcher(watcher ConfigWatcher) {
	manager := GetGlobalConfigManager()
	manager.AddWatcher(watcher)
}
