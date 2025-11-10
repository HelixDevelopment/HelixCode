package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/logging"
	"gopkg.in/yaml.v3"
)

// ConfigurationManager manages configuration with advanced features
type ConfigurationManager struct {
	config           *HelixConfig
	logger           logging.Logger
	mu               sync.RWMutex
	configPath       string
	backupPath       string
	lastModified     time.Time
	version          string
	schemas          map[string]*ConfigurationSchema
	validators       map[string][]ValidationRule
	transformers     map[string][]Transformer
	watchers         map[string][]ConfigFileWatcher
	hooks            map[string][]ConfigHook
	encryptionKey    []byte
	initialized      bool
	autoSave         bool
	autoBackup       bool
	enableEncryption bool
}

// ConfigurationSchema defines the structure and validation rules for configuration
type ConfigurationSchema struct {
	Version              string                     `json:"version"`
	Properties           map[string]*PropertySchema `json:"properties"`
	Required             []string                   `json:"required"`
	AdditionalProperties bool                       `json:"additionalProperties"`
	Description          string                     `json:"description"`
	Examples             []interface{}              `json:"examples"`
}

// PropertySchema defines individual property validation
type PropertySchema struct {
	Type            string                     `json:"type"`
	Description     string                     `json:"description"`
	Required        bool                       `json:"required"`
	Default         interface{}                `json:"default"`
	Enum            []interface{}              `json:"enum,omitempty"`
	Minimum         *float64                   `json:"minimum,omitempty"`
	Maximum         *float64                   `json:"maximum,omitempty"`
	MinLength       *int                       `json:"minLength,omitempty"`
	MaxLength       *int                       `json:"maxLength,omitempty"`
	Pattern         string                     `json:"pattern,omitempty"`
	Format          string                     `json:"format,omitempty"`
	Items           *PropertySchema            `json:"items,omitempty"`
	Properties      map[string]*PropertySchema `json:"properties,omitempty"`
	ValidationRules []string                   `json:"validationRules,omitempty"`
	Transformations []string                   `json:"transformations,omitempty"`
	Sensitive       bool                       `json:"sensitive,omitempty"`
}

// ValidationRule defines custom validation logic
type ValidationRule interface {
	Validate(value interface{}, context *ValidationContext) error
	GetName() string
	GetDescription() string
}

// Transformer defines value transformation logic
type Transformer interface {
	Transform(value interface{}, context *TransformContext) (interface{}, error)
	GetName() string
	GetDescription() string
}

// ValidationContext provides context for validation
type ValidationContext struct {
	Property    string
	FullPath    string
	Schema      *PropertySchema
	Config      *HelixConfig
	Environment map[string]string
	DateTime    time.Time
	User        string
	Session     string
}

// TransformContext provides context for transformation
type TransformContext struct {
	Property     string
	FullPath     string
	Schema       *PropertySchema
	Config       *HelixConfig
	Environment  map[string]string
	DateTime     time.Time
	User         string
	Session      string
	Transformers map[string]Transformer
}

// ConfigFileWatcher defines configuration file change notification
type ConfigFileWatcher interface {
	OnConfigChange(change *ConfigChange) error
	GetName() string
	GetWatchPaths() []string
}

// ConfigHook defines configuration lifecycle hooks
type ConfigHook interface {
	BeforeLoad(path string, config *HelixConfig) error
	AfterLoad(path string, config *HelixConfig) error
	BeforeSave(path string, config *HelixConfig) error
	AfterSave(path string, config *HelixConfig) error
	OnError(path string, err error, operation string) error
	GetName() string
	GetPriority() int
}

// ConfigChange represents configuration changes
type ConfigChange struct {
	Type      ChangeType             `json:"type"`
	Path      string                 `json:"path"`
	Property  string                 `json:"property"`
	OldValue  interface{}            `json:"oldValue,omitempty"`
	NewValue  interface{}            `json:"newValue,omitempty"`
	OldConfig *HelixConfig           `json:"oldConfig,omitempty"`
	NewConfig *HelixConfig           `json:"newConfig,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	User      string                 `json:"user,omitempty"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ChangeType represents the type of configuration change
type ChangeType string

const (
	ChangeTypeCreated ChangeType = "created"
	ChangeTypeUpdated ChangeType = "updated"
	ChangeTypeDeleted ChangeType = "deleted"
	ChangeTypeMoved   ChangeType = "moved"
	ChangeTypeCopied  ChangeType = "copied"
)

// ConfigurationOptions provides options for configuration manager
type ConfigurationOptions struct {
	ConfigPath       string                 `json:"configPath"`
	BackupPath       string                 `json:"backupPath"`
	AutoSave         bool                   `json:"autoSave"`
	AutoBackup       bool                   `json:"autoBackup"`
	EnableEncryption bool                   `json:"enableEncryption"`
	EncryptionKey    string                 `json:"encryptionKey,omitempty"`
	SchemaPath       string                 `json:"schemaPath,omitempty"`
	WatchInterval    time.Duration          `json:"watchInterval"`
	MaxBackups       int                    `json:"maxBackups"`
	Compression      bool                   `json:"compression"`
	LogLevel         string                 `json:"logLevel"`
	Environment      map[string]string      `json:"environment"`
	ValidationMode   ValidationMode         `json:"validationMode"`
	TransformMode    TransformMode          `json:"transformMode"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationMode defines validation behavior
type ValidationMode string

const (
	ValidationModeStrict   ValidationMode = "strict"
	ValidationModeLenient  ValidationMode = "lenient"
	ValidationModeDisabled ValidationMode = "disabled"
	ValidationModeSchema   ValidationMode = "schema"
	ValidationModeCustom   ValidationMode = "custom"
)

// TransformMode defines transformation behavior
type TransformMode string

const (
	TransformModeStrict   TransformMode = "strict"
	TransformModeLenient  TransformMode = "lenient"
	TransformModeDisabled TransformMode = "disabled"
	TransformModeSchema   TransformMode = "schema"
	TransformModeCustom   TransformMode = "custom"
)

// NewConfigurationManager creates a new configuration manager
func NewConfigurationManager(options *ConfigurationOptions) (*ConfigurationManager, error) {
	if options == nil {
		options = &ConfigurationOptions{
			AutoSave:         true,
			AutoBackup:       true,
			EnableEncryption: false,
			ValidationMode:   ValidationModeStrict,
			TransformMode:    TransformModeLenient,
			MaxBackups:       10,
			Compression:      true,
			LogLevel:         "info",
		}
	}

	logger := logging.NewLogger("configuration_manager")

	manager := &ConfigurationManager{
		logger:           logger,
		configPath:       options.ConfigPath,
		backupPath:       options.BackupPath,
		version:          "1.0.0",
		schemas:          make(map[string]*ConfigurationSchema),
		validators:       make(map[string][]ValidationRule),
		transformers:     make(map[string][]Transformer),
		watchers:         make(map[string][]ConfigFileWatcher),
		hooks:            make(map[string][]ConfigHook),
		autoSave:         options.AutoSave,
		autoBackup:       options.AutoBackup,
		enableEncryption: options.EnableEncryption,
		initialized:      false,
	}

	// Set encryption key if provided
	if options.EnableEncryption && options.EncryptionKey != "" {
		manager.encryptionKey = []byte(options.EncryptionKey)
	}

	// Initialize with defaults
	config := DefaultHelixConfig()
	manager.config = config

	return manager, nil
}

// Initialize initializes the configuration manager
func (cm *ConfigurationManager) Initialize(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.initialized {
		return nil
	}

	cm.logger.Info("Initializing Configuration Manager...")

	// Load schemas
	if err := cm.loadSchemas(); err != nil {
		return fmt.Errorf("failed to load schemas: %w", err)
	}

	// Load configuration
	if cm.configPath != "" {
		if err := cm.loadConfiguration(cm.configPath); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	// Apply default values
	if err := cm.applyDefaults(); err != nil {
		return fmt.Errorf("failed to apply defaults: %w", err)
	}

	// Validate configuration
	if err := cm.validateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Apply transformations
	if err := cm.applyTransformations(); err != nil {
		return fmt.Errorf("failed to apply transformations: %w", err)
	}

	// Set up file watching
	if err := cm.setupFileWatching(); err != nil {
		return fmt.Errorf("failed to setup file watching: %w", err)
	}

	// Create backup directory if needed
	if cm.autoBackup && cm.backupPath != "" {
		if err := os.MkdirAll(cm.backupPath, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}
	}

	cm.initialized = true
	cm.lastModified = time.Now()

	cm.logger.Info("Configuration Manager initialized successfully")
	return nil
}

// Load loads configuration from the specified path
func (cm *ConfigurationManager) Load(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.loadConfiguration(path)
}

// Save saves configuration to the specified path
func (cm *ConfigurationManager) Save(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.saveConfiguration(path)
}

// GetConfig returns the current configuration
func (cm *ConfigurationManager) GetConfig() *HelixConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a deep copy to prevent modification
	return cm.deepCopyConfig(cm.config)
}

// UpdateConfig updates the configuration with the provided changes
func (cm *ConfigurationManager) UpdateConfig(updates map[string]interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Create old config for comparison
	oldConfig := cm.deepCopyConfig(cm.config)

	// Apply updates
	if err := cm.applyUpdates(updates); err != nil {
		return fmt.Errorf("failed to apply updates: %w", err)
	}

	// Validate updated configuration
	if err := cm.validateConfiguration(); err != nil {
		// Revert to old config on validation failure
		cm.config = oldConfig
		return fmt.Errorf("validation failed after updates: %w", err)
	}

	// Apply transformations
	if err := cm.applyTransformations(); err != nil {
		// Revert to old config on transformation failure
		cm.config = oldConfig
		return fmt.Errorf("transformation failed after updates: %w", err)
	}

	// Create change notification
	change := &ConfigChange{
		Type:      ChangeTypeUpdated,
		NewConfig: cm.deepCopyConfig(cm.config),
		Timestamp: time.Now(),
		Source:    "UpdateConfig",
		Metadata: map[string]interface{}{
			"updates": updates,
		},
	}

	// Notify watchers
	if err := cm.notifyWatchers(change); err != nil {
		cm.logger.Warn("Failed to notify configuration watchers", "error", err)
	}

	// Auto-save if enabled
	if cm.autoSave {
		if err := cm.saveConfiguration(cm.configPath); err != nil {
			cm.logger.Warn("Failed to auto-save configuration", "error", err)
		}
	}

	cm.lastModified = time.Now()
	cm.logger.Info("Configuration updated successfully", "changes", len(updates))

	return nil
}

// GetProperty returns a specific configuration property
func (cm *ConfigurationManager) GetProperty(path string) (interface{}, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.getPropertyValue(path, cm.config)
}

// SetProperty sets a specific configuration property
func (cm *ConfigurationManager) SetProperty(path string, value interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Get old value
	oldValue, err := cm.getPropertyValue(path, cm.config)
	if err != nil {
		return fmt.Errorf("failed to get current property value: %w", err)
	}

	// Validate property
	if err := cm.validateProperty(path, value); err != nil {
		return fmt.Errorf("property validation failed: %w", err)
	}

	// Transform property
	transformedValue, err := cm.transformProperty(path, value)
	if err != nil {
		return fmt.Errorf("property transformation failed: %w", err)
	}

	// Set property
	if err := cm.setPropertyValue(path, transformedValue, cm.config); err != nil {
		return fmt.Errorf("failed to set property value: %w", err)
	}

	// Create change notification
	change := &ConfigChange{
		Type:      ChangeTypeUpdated,
		Path:      path,
		Property:  filepath.Base(path),
		OldValue:  oldValue,
		NewValue:  transformedValue,
		Timestamp: time.Now(),
		Source:    "SetProperty",
	}

	// Notify watchers
	if err := cm.notifyWatchers(change); err != nil {
		cm.logger.Warn("Failed to notify configuration watchers", "error", err)
	}

	// Auto-save if enabled
	if cm.autoSave {
		if err := cm.saveConfiguration(cm.configPath); err != nil {
			cm.logger.Warn("Failed to auto-save configuration", "error", err)
		}
	}

	cm.lastModified = time.Now()
	cm.logger.Debug("Property set successfully", "path", path)

	return nil
}

// AddSchema adds a configuration schema
func (cm *ConfigurationManager) AddSchema(name string, schema *ConfigurationSchema) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.schemas[name] = schema
	cm.logger.Debug("Schema added", "name", name, "version", schema.Version)

	return nil
}

// AddValidator adds a validation rule for a property
func (cm *ConfigurationManager) AddValidator(property string, rule ValidationRule) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.validators[property] == nil {
		cm.validators[property] = make([]ValidationRule, 0)
	}

	cm.validators[property] = append(cm.validators[property], rule)
	cm.logger.Debug("Validator added", "property", property, "rule", rule.GetName())

	return nil
}

// AddTransformer adds a transformer for a property
func (cm *ConfigurationManager) AddTransformer(property string, transformer Transformer) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.transformers[property] == nil {
		cm.transformers[property] = make([]Transformer, 0)
	}

	cm.transformers[property] = append(cm.transformers[property], transformer)
	cm.logger.Debug("Transformer added", "property", property, "transformer", transformer.GetName())

	return nil
}

// AddWatcher adds a configuration change watcher
func (cm *ConfigurationManager) AddWatcher(property string, watcher ConfigFileWatcher) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.watchers[property] == nil {
		cm.watchers[property] = make([]ConfigFileWatcher, 0)
	}

	cm.watchers[property] = append(cm.watchers[property], watcher)
	cm.logger.Debug("Watcher added", "property", property, "watcher", watcher.GetName())

	return nil
}

// AddHook adds a configuration lifecycle hook
func (cm *ConfigurationManager) AddHook(hook ConfigHook) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	hookType := reflect.TypeOf(hook).String()
	if cm.hooks[hookType] == nil {
		cm.hooks[hookType] = make([]ConfigHook, 0)
	}

	cm.hooks[hookType] = append(cm.hooks[hookType], hook)
	cm.logger.Debug("Hook added", "type", hookType, "hook", hook.GetName())

	return nil
}

// CreateBackup creates a backup of the current configuration
func (cm *ConfigurationManager) CreateBackup() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.backupPath == "" {
		return fmt.Errorf("backup path not configured")
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(cm.backupPath, fmt.Sprintf("helix_config_%s.json", timestamp))

	// Save configuration to backup file
	if err := cm.saveConfigurationToFile(backupFile, cm.config); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Clean up old backups
	if err := cm.cleanupOldBackups(); err != nil {
		cm.logger.Warn("Failed to cleanup old backups", "error", err)
	}

	cm.logger.Info("Backup created successfully", "file", backupFile)
	return nil
}

// RestoreBackup restores configuration from a backup file
func (cm *ConfigurationManager) RestoreBackup(backupFile string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !filepath.IsAbs(backupFile) {
		backupFile = filepath.Join(cm.backupPath, backupFile)
	}

	// Check if backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	// Load backup configuration
	backupConfig := &HelixConfig{}
	if err := cm.loadConfigurationFromFile(backupFile, backupConfig); err != nil {
		return fmt.Errorf("failed to load backup configuration: %w", err)
	}

	// Validate backup configuration
	if err := cm.validateConfigObject(backupConfig); err != nil {
		return fmt.Errorf("backup configuration validation failed: %w", err)
	}

	// Replace current configuration with backup
	cm.config = backupConfig

	// Save restored configuration
	if cm.autoSave {
		if err := cm.saveConfiguration(cm.configPath); err != nil {
			return fmt.Errorf("failed to save restored configuration: %w", err)
		}
	}

	cm.lastModified = time.Now()
	cm.logger.Info("Configuration restored from backup", "file", backupFile)

	return nil
}

// ListBackups lists available backup files
func (cm *ConfigurationManager) ListBackups() ([]string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.backupPath == "" {
		return nil, fmt.Errorf("backup path not configured")
	}

	// Read backup directory
	entries, err := os.ReadDir(cm.backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter and sort backup files
	backups := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "helix_config_") && strings.HasSuffix(entry.Name(), ".json") {
			backups = append(backups, entry.Name())
		}
	}

	return backups, nil
}

// GetVersion returns the configuration version
func (cm *ConfigurationManager) GetVersion() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.version
}

// GetLastModified returns the last modification time
func (cm *ConfigurationManager) GetLastModified() time.Time {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.lastModified
}

// Export exports configuration to the specified format
func (cm *ConfigurationManager) Export(format string, path string) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(cm.config, "", "  ")
	case "yaml":
		data, err = yaml.Marshal(cm.config)
	case "toml":
		// TOML support would require additional library
		return fmt.Errorf("TOML export not yet supported")
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	cm.logger.Info("Configuration exported successfully", "format", format, "path", path)
	return nil
}

// Import imports configuration from the specified format
func (cm *ConfigurationManager) Import(format string, path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	config := &HelixConfig{}

	switch strings.ToLower(format) {
	case "json":
		err = json.Unmarshal(data, config)
	case "yaml":
		err = yaml.Unmarshal(data, config)
	case "toml":
		// TOML support would require additional library
		return fmt.Errorf("TOML import not yet supported")
	default:
		return fmt.Errorf("unsupported import format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate imported configuration
	if err := cm.validateConfigObject(config); err != nil {
		return fmt.Errorf("imported configuration validation failed: %w", err)
	}

	// Replace current configuration
	cm.config = config
	cm.lastModified = time.Now()

	// Auto-save if enabled
	if cm.autoSave {
		if err := cm.saveConfiguration(cm.configPath); err != nil {
			cm.logger.Warn("Failed to auto-save imported configuration", "error", err)
		}
	}

	cm.logger.Info("Configuration imported successfully", "format", format, "path", path)
	return nil
}

// Private helper methods

func (cm *ConfigurationManager) loadConfiguration(path string) error {
	// Execute before load hooks
	if err := cm.executeHooks("before_load", path, cm.config); err != nil {
		return fmt.Errorf("before load hooks failed: %w", err)
	}

	// Load configuration from file
	config := &HelixConfig{}
	if err := cm.loadConfigurationFromFile(path, config); err != nil {
		return fmt.Errorf("failed to load configuration from file: %w", err)
	}

	// Execute after load hooks
	if err := cm.executeHooks("after_load", path, config); err != nil {
		return fmt.Errorf("after load hooks failed: %w", err)
	}

	cm.config = config
	cm.configPath = path
	cm.lastModified = time.Now()

	return nil
}

func (cm *ConfigurationManager) saveConfiguration(path string) error {
	// Execute before save hooks
	if err := cm.executeHooks("before_save", path, cm.config); err != nil {
		return fmt.Errorf("before save hooks failed: %w", err)
	}

	// Create backup if enabled
	if cm.autoBackup {
		if err := cm.CreateBackup(); err != nil {
			cm.logger.Warn("Failed to create backup before save", "error", err)
		}
	}

	// Save configuration to file
	if err := cm.saveConfigurationToFile(path, cm.config); err != nil {
		// Execute error hooks
		_ = cm.executeHooks("error", path, err, "save")
		return fmt.Errorf("failed to save configuration to file: %w", err)
	}

	// Execute after save hooks
	if err := cm.executeHooks("after_save", path, cm.config); err != nil {
		return fmt.Errorf("after save hooks failed: %w", err)
	}

	cm.configPath = path
	cm.lastModified = time.Now()

	return nil
}

func (cm *ConfigurationManager) loadConfigurationFromFile(path string, config *HelixConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Decrypt if encryption is enabled
	if cm.enableEncryption && len(cm.encryptionKey) > 0 {
		decryptedData, err := cm.decryptData(data)
		if err != nil {
			return fmt.Errorf("failed to decrypt configuration: %w", err)
		}
		data = decryptedData
	}

	// Determine format from file extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		err = json.Unmarshal(data, config)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, config)
	default:
		// Try JSON as default
		err = json.Unmarshal(data, config)
	}

	if err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return nil
}

func (cm *ConfigurationManager) saveConfigurationToFile(path string, config *HelixConfig) error {
	// Determine format from file extension
	ext := strings.ToLower(filepath.Ext(path))
	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
	default:
		// Default to JSON
		data, err = json.MarshalIndent(config, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Encrypt if encryption is enabled
	if cm.enableEncryption && len(cm.encryptionKey) > 0 {
		encryptedData, err := cm.encryptData(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt configuration: %w", err)
		}
		data = encryptedData
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

func (cm *ConfigurationManager) validateConfiguration() error {
	return cm.validateConfigObject(cm.config)
}

func (cm *ConfigurationManager) validateConfigObject(config *HelixConfig) error {
	// Schema validation
	for name, schema := range cm.schemas {
		if err := cm.validateAgainstSchema(schema, config); err != nil {
			return fmt.Errorf("schema validation failed for %s: %w", name, err)
		}
	}

	// Custom validation
	for property, rules := range cm.validators {
		value, err := cm.getPropertyValue(property, config)
		if err != nil {
			return fmt.Errorf("failed to get property %s for validation: %w", property, err)
		}

		context := &ValidationContext{
			Property:    property,
			FullPath:    property,
			Config:      config,
			Environment: cm.getEnvironment(),
			DateTime:    time.Now(),
		}

		for _, rule := range rules {
			if err := rule.Validate(value, context); err != nil {
				return fmt.Errorf("validation rule %s failed for property %s: %w", rule.GetName(), property, err)
			}
		}
	}

	return nil
}

func (cm *ConfigurationManager) validateProperty(path string, value interface{}) error {
	// Get schema for property
	schema := cm.getPropertySchema(path)
	if schema != nil {
		context := &ValidationContext{
			Property:    filepath.Base(path),
			FullPath:    path,
			Schema:      schema,
			Config:      cm.config,
			Environment: cm.getEnvironment(),
			DateTime:    time.Now(),
		}

		if err := cm.validateValueAgainstSchema(value, schema, context); err != nil {
			return fmt.Errorf("property %s validation failed: %w", path, err)
		}
	}

	// Custom validation rules
	if rules, exists := cm.validators[path]; exists {
		context := &ValidationContext{
			Property:    filepath.Base(path),
			FullPath:    path,
			Config:      cm.config,
			Environment: cm.getEnvironment(),
			DateTime:    time.Now(),
		}

		for _, rule := range rules {
			if err := rule.Validate(value, context); err != nil {
				return fmt.Errorf("validation rule %s failed for property %s: %w", rule.GetName(), path, err)
			}
		}
	}

	return nil
}

func (cm *ConfigurationManager) applyTransformations() error {
	// Apply schema-based transformations
	for name, schema := range cm.schemas {
		if err := cm.applySchemaTransformations(schema, cm.config); err != nil {
			return fmt.Errorf("schema transformations failed for %s: %w", name, err)
		}
	}

	// Apply custom transformations
	for property, transformers := range cm.transformers {
		value, err := cm.getPropertyValue(property, cm.config)
		if err != nil {
			return fmt.Errorf("failed to get property %s for transformation: %w", property, err)
		}

		context := &TransformContext{
			Property:     filepath.Base(property),
			FullPath:     property,
			Config:       cm.config,
			Environment:  cm.getEnvironment(),
			DateTime:     time.Now(),
			Transformers: cm.makeTransformerMap(),
		}

		for _, transformer := range transformers {
			transformedValue, err := transformer.Transform(value, context)
			if err != nil {
				return fmt.Errorf("transformation %s failed for property %s: %w", transformer.GetName(), property, err)
			}

			value = transformedValue
		}

		// Set transformed value back
		if err := cm.setPropertyValue(property, value, cm.config); err != nil {
			return fmt.Errorf("failed to set transformed property %s: %w", property, err)
		}
	}

	return nil
}

func (cm *ConfigurationManager) transformProperty(path string, value interface{}) (interface{}, error) {
	// Apply schema-based transformations
	schema := cm.getPropertySchema(path)
	if schema != nil {
		context := &TransformContext{
			Property:     filepath.Base(path),
			FullPath:     path,
			Schema:       schema,
			Config:       cm.config,
			Environment:  cm.getEnvironment(),
			DateTime:     time.Now(),
			Transformers: cm.makeTransformerMap(),
		}

		if transformedValue, err := cm.applyValueTransformations(value, schema, context); err == nil {
			value = transformedValue
		}
	}

	// Apply custom transformations
	if transformers, exists := cm.transformers[path]; exists {
		context := &TransformContext{
			Property:     filepath.Base(path),
			FullPath:     path,
			Config:       cm.config,
			Environment:  cm.getEnvironment(),
			DateTime:     time.Now(),
			Transformers: cm.makeTransformerMap(),
		}

		for _, transformer := range transformers {
			transformedValue, err := transformer.Transform(value, context)
			if err != nil {
				return nil, fmt.Errorf("transformation %s failed for property %s: %w", transformer.GetName(), path, err)
			}

			value = transformedValue
		}
	}

	return value, nil
}

func (cm *ConfigurationManager) applyDefaults() error {
	for name, schema := range cm.schemas {
		if err := cm.applySchemaDefaults(schema, cm.config); err != nil {
			return fmt.Errorf("failed to apply defaults for schema %s: %w", name, err)
		}
	}

	return nil
}

func (cm *ConfigurationManager) applyUpdates(updates map[string]interface{}) error {
	for path, value := range updates {
		if err := cm.setPropertyValue(path, value, cm.config); err != nil {
			return fmt.Errorf("failed to apply update for %s: %w", path, err)
		}
	}

	return nil
}

func (cm *ConfigurationManager) getPropertyValue(path string, config *HelixConfig) (interface{}, error) {
	// Navigate to the property using the path
	parts := strings.Split(path, ".")
	current := reflect.ValueOf(config).Elem()

	for i, part := range parts {
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}

		if current.Kind() == reflect.Struct {
			field := current.FieldByName(part)
			if !field.IsValid() {
				return nil, fmt.Errorf("invalid property path: %s (field %s not found)", path, part)
			}

			if i == len(parts)-1 {
				return field.Interface(), nil
			}

			current = field
		} else {
			return nil, fmt.Errorf("invalid property path: %s (not a struct at %s)", path, part)
		}
	}

	return nil, fmt.Errorf("invalid property path: %s", path)
}

func (cm *ConfigurationManager) setPropertyValue(path string, value interface{}, config *HelixConfig) error {
	// Navigate to the parent of the property
	parts := strings.Split(path, ".")
	current := reflect.ValueOf(config).Elem()

	for i, part := range parts {
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}

		if current.Kind() == reflect.Struct {
			field := current.FieldByName(part)
			if !field.IsValid() {
				return fmt.Errorf("invalid property path: %s (field %s not found)", path, part)
			}

			if i == len(parts)-1 {
				// Set the final field value
				valueType := reflect.TypeOf(value)
				if field.Type() != valueType {
					// Try to convert the value to the expected type
					convertedValue := reflect.New(field.Type()).Elem()
					convertedValue.Set(reflect.ValueOf(value))
					field.Set(convertedValue)
				} else {
					field.Set(reflect.ValueOf(value))
				}
				return nil
			}

			current = field
		} else {
			return fmt.Errorf("invalid property path: %s (not a struct at %s)", path, part)
		}
	}

	return fmt.Errorf("invalid property path: %s", path)
}

func (cm *ConfigurationManager) deepCopyConfig(config *HelixConfig) *HelixConfig {
	// Use JSON marshaling for deep copy
	data, _ := json.Marshal(config)
	copied := &HelixConfig{}
	json.Unmarshal(data, copied)
	return copied
}

func (cm *ConfigurationManager) getEnvironment() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

func (cm *ConfigurationManager) makeTransformerMap() map[string]Transformer {
	transformers := make(map[string]Transformer)
	for _, transformerList := range cm.transformers {
		for _, transformer := range transformerList {
			transformers[transformer.GetName()] = transformer
		}
	}
	return transformers
}

func (cm *ConfigurationManager) executeHooks(hookType, path string, args ...interface{}) error {
	hooks := cm.hooks[hookType]
	if hooks == nil {
		return nil
	}

	// Sort hooks by priority
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].GetPriority() < hooks[j].GetPriority()
	})

	for _, hook := range hooks {
		var err error
		switch hookType {
		case "before_load":
			err = hook.BeforeLoad(path, args[0].(*HelixConfig))
		case "after_load":
			err = hook.AfterLoad(path, args[0].(*HelixConfig))
		case "before_save":
			err = hook.BeforeSave(path, args[0].(*HelixConfig))
		case "after_save":
			err = hook.AfterSave(path, args[0].(*HelixConfig))
		case "error":
			err = hook.OnError(path, args[0].(error), args[1].(string))
		}

		if err != nil {
			return fmt.Errorf("hook %s failed: %w", hook.GetName(), err)
		}
	}

	return nil
}

func (cm *ConfigurationManager) notifyWatchers(change *ConfigChange) error {
	for property, watchers := range cm.watchers {
		// Check if this watcher should be notified
		if strings.HasPrefix(change.Path, property) || property == "*" {
			for _, watcher := range watchers {
				if err := watcher.OnConfigChange(change); err != nil {
					cm.logger.Warn("Watcher notification failed", "watcher", watcher.GetName(), "error", err)
				}
			}
		}
	}

	return nil
}

func (cm *ConfigurationManager) cleanupOldBackups() error {
	if cm.backupPath == "" {
		return nil
	}

	// Get list of backup files
	backupFiles, err := os.ReadDir(cm.backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter and sort backup files
	var backups []os.DirEntry
	for _, entry := range backupFiles {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "helix_config_") && strings.HasSuffix(entry.Name(), ".json") {
			backups = append(backups, entry)
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		infoI, _ := backups[i].Info()
		infoJ, _ := backups[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Remove old backups if we have too many
	maxBackups := 10 // Default
	if len(backups) > maxBackups {
		for i := maxBackups; i < len(backups); i++ {
			backupFile := filepath.Join(cm.backupPath, backups[i].Name())
			if err := os.Remove(backupFile); err != nil {
				cm.logger.Warn("Failed to remove old backup", "file", backupFile, "error", err)
			}
		}
	}

	return nil
}

func (cm *ConfigurationManager) setupFileWatching() error {
	// This would implement file system watching for auto-reload
	// For now, it's a placeholder
	return nil
}

func (cm *ConfigurationManager) loadSchemas() error {
	// This would load schemas from schema files
	// For now, add default schemas
	defaultSchema := &ConfigurationSchema{
		Version: "1.0.0",
		Properties: map[string]*PropertySchema{
			"cognee": {
				Type:        "object",
				Description: "Cognee configuration",
				Required:    false,
			},
			"api_keys": {
				Type:        "object",
				Description: "API keys configuration",
				Required:    false,
			},
		},
		Required:             []string{},
		AdditionalProperties: true,
		Description:          "HelixCode configuration schema",
	}

	cm.schemas["default"] = defaultSchema
	return nil
}

// Placeholder implementations for advanced features

func (cm *ConfigurationManager) encryptData(data []byte) ([]byte, error) {
	// This would implement actual encryption
	return data, nil
}

func (cm *ConfigurationManager) decryptData(data []byte) ([]byte, error) {
	// This would implement actual decryption
	return data, nil
}

func (cm *ConfigurationManager) getPropertySchema(path string) *PropertySchema {
	// This would get the schema for a specific property
	return nil
}

func (cm *ConfigurationManager) validateAgainstSchema(schema *ConfigurationSchema, config *HelixConfig) error {
	// This would validate configuration against schema
	return nil
}

func (cm *ConfigurationManager) validateValueAgainstSchema(value interface{}, schema *PropertySchema, context *ValidationContext) error {
	// This would validate a value against schema
	return nil
}

func (cm *ConfigurationManager) applySchemaTransformations(schema *ConfigurationSchema, config *HelixConfig) error {
	// This would apply schema-based transformations
	return nil
}

func (cm *ConfigurationManager) applyValueTransformations(value interface{}, schema *PropertySchema, context *TransformContext) (interface{}, error) {
	// This would apply transformations to a value
	return value, nil
}

func (cm *ConfigurationManager) applySchemaDefaults(schema *ConfigurationSchema, config *HelixConfig) error {
	// This would apply schema-based defaults
	return nil
}
