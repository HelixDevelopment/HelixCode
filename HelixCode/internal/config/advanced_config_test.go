package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigurationValidator tests configuration validation
func TestConfigurationValidator(t *testing.T) {
	validator := NewConfigurationValidator(true)
	require.NotNil(t, validator)

	// Test valid configuration
	config := getDefaultConfig()
	result := validator.Validate(config)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Test invalid configuration
	invalidConfig := getDefaultConfig()
	invalidConfig.Server.Port = -1
	invalidConfig.LLM.Temperature = 5.0

	result = validator.Validate(invalidConfig)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)

	// Check for specific errors
	portErrorFound := false
	tempErrorFound := false

	for _, err := range result.Errors {
		if err.Property == "server.port" {
			portErrorFound = true
			assert.Equal(t, "server.port", err.Path)
			assert.Equal(t, "error", err.Severity)
		}
		if err.Property == "llm.temperature" {
			tempErrorFound = true
		}
	}

	assert.True(t, portErrorFound)
	assert.True(t, tempErrorFound)
}

// TestConfigurationValidatorCustomRules tests custom validation rules
func TestConfigurationValidatorCustomRules(t *testing.T) {
	validator := NewConfigurationValidator(true)

	// Add custom rule
	validator.AddCustomRule("application.name", func(value interface{}) error {
		if str, ok := value.(string); ok && str == "forbidden" {
			return assert.AnError
		}
		return nil
	})

	// Test valid value
	config := getDefaultConfig()
	config.Application.Name = "allowed"
	result := validator.Validate(config)
	assert.True(t, result.Valid)

	// Test invalid value
	config.Application.Name = "forbidden"
	result = validator.Validate(config)
	assert.False(t, result.Valid)

	// Check for custom rule error
	customErrorFound := false
	for _, err := range result.Errors {
		if err.Code == "CUSTOM_RULE_ERROR" && err.Property == "application.name" {
			customErrorFound = true
			break
		}
	}
	assert.True(t, customErrorFound)
}

// TestConfigurationValidatorFieldValidation tests field-specific validation
func TestConfigurationValidatorFieldValidation(t *testing.T) {
	validator := NewConfigurationValidator(true)

	// Test field validation
	config := getDefaultConfig()

	// Valid field
	result := validator.ValidateField(config, "server.port")
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Invalid field
	config.Server.Port = 70000
	result = validator.ValidateField(config, "server.port")
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Equal(t, "server.port", result.Path)
}

// TestConfigurationSchema tests configuration schema
func TestConfigurationSchema(t *testing.T) {
	validator := NewConfigurationValidator(true)
	schema := validator.createDefaultSchema()

	require.NotNil(t, schema)
	assert.Equal(t, "1.0", schema.Version)
	assert.NotEmpty(t, schema.Properties)
	assert.NotEmpty(t, schema.Required)

	// Check required properties
	assert.Contains(t, schema.Required, "version")
	assert.Contains(t, schema.Required, "application")
	assert.Contains(t, schema.Required, "server")

	// Check application properties
	appProp, exists := schema.Properties["application"]
	require.True(t, exists)
	assert.Equal(t, "object", appProp.Type)
	assert.NotEmpty(t, appProp.Properties)
	assert.NotEmpty(t, appProp.Required)

	// Check name property
	nameProp, exists := appProp.Properties["name"]
	require.True(t, exists)
	assert.Equal(t, "string", nameProp.Type)
	assert.NotNil(t, nameProp.MinLength)
	assert.NotNil(t, nameProp.MaxLength)
}

// TestConfigurationMigrator tests configuration migration
func TestConfigurationMigrator(t *testing.T) {
	migrator := NewConfigurationMigrator("1.0.0")
	require.NotNil(t, migrator)
	assert.Equal(t, "1.0.0", migrator.current)

	// Test available versions
	versions := migrator.GetAvailableVersions()
	assert.NotEmpty(t, versions)
	assert.Contains(t, versions, "1.0.0")
	assert.Contains(t, versions, "1.1.0")
	assert.Contains(t, versions, "1.2.0")

	// Test migration path
	config := &HelixConfig{
		Version: "1.0.0",
		Application: ApplicationConfig{
			Name: "Test App",
			Workspace: WorkspaceConfig{
				AutoSave: false, // Should be set to true by migration
			},
		},
	}

	err := migrator.Migrate(config, "1.2.0")
	assert.NoError(t, err)
	assert.Equal(t, "1.2.0", config.Version)
	assert.True(t, config.Application.Workspace.AutoSave)
}

// TestConfigurationMigratorPathFinding tests migration path finding
func TestConfigurationMigratorPathFinding(t *testing.T) {
	migrator := NewConfigurationMigrator("1.0.0")

	// Test direct path
	path := migrator.findMigrationPath("1.0.0", "1.1.0")
	assert.Len(t, path, 1)

	// Test multi-step path
	path = migrator.findMigrationPath("1.0.0", "1.2.0")
	assert.Len(t, path, 2) // 1.0.0 -> 1.1.0 -> 1.2.0

	// Test no path
	path = migrator.findMigrationPath("1.0.0", "9.9.9")
	assert.Nil(t, path)
}

// TestConfigurationTransformer tests configuration transformation
func TestConfigurationTransformer(t *testing.T) {
	transformer := NewConfigurationTransformer()
	require.NotNil(t, transformer)

	// Test simple transformation
	config := getDefaultConfig()
	variables := map[string]interface{}{
		"server_port": 9090,
		"app_name":    "Transformed App",
	}

	// Add mappings
	transformer.AddMapping(TransformMapping{
		Source:    "server.port",
		Target:    "server.port",
		Transform: "copy",
		Priority:  1,
	})

	transformer.AddMapping(TransformMapping{
		Source:    "application.name",
		Target:    "application.name",
		Transform: "copy",
		Priority:  2,
	})

	result, err := transformer.Transform(config, variables)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify transformations
	assert.Equal(t, 9090, result.Server.Port)
	assert.Equal(t, "Transformed App", result.Application.Name)
}

// TestConfigurationTransformerTypes tests type transformations
func TestConfigurationTransformerTypes(t *testing.T) {
	transformer := NewConfigurationTransformer()

	// Test string to int conversion
	mapping := TransformMapping{
		Source:    "test_value",
		Target:    "converted_value",
		Transform: "convert",
		Parameters: map[string]interface{}{
			"from": "string",
			"to":   "int",
		},
	}

	transformer.AddMapping(mapping)

	// Test conversion
	config := &struct {
		TestValue      string
		ConvertedValue int
	}{}

	variables := map[string]interface{}{
		"test_value": "123",
	}

	err := transformer.applyMapping(config, mapping, variables)
	assert.NoError(t, err)
}

// TestConfigurationTransformerConditions tests conditional transformations
func TestConfigurationTransformerConditions(t *testing.T) {
	transformer := NewConfigurationTransformer()

	// Test conditional mapping
	mapping := TransformMapping{
		Source:    "llm.temperature",
		Target:    "llm.temperature",
		Transform: "copy",
		Condition: "development",
		Priority:  1,
	}

	transformer.AddMapping(mapping)

	// Test with development config
	devConfig := getDefaultConfig()
	devConfig.Application.Environment = "development"
	variables := map[string]interface{}{}

	result, err := transformer.Transform(devConfig, variables)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test with production config (should not apply)
	prodConfig := getDefaultConfig()
	prodConfig.Application.Environment = "production"

	result, err = transformer.Transform(prodConfig, variables)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestConfigurationTemplateManager tests template management
func TestConfigurationTemplateManager(t *testing.T) {
	tempDir := t.TempDir()
	templateDir := tempDir + "/templates"

	// Create template directory
	err := os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	manager := NewConfigurationTemplateManager(templateDir)
	require.NotNil(t, manager)

	// Test creating template from config
	config := getDefaultConfig()
	variables := map[string]*TemplateVariable{
		"app_name": {
			Name:      "Application Name",
			Type:      "string",
			Default:   "My App",
			Required:  true,
			MinLength: intPtr(1),
		},
		"port": {
			Name:    "Server Port",
			Type:    "number",
			Default: 8080,
			Min:     float64Ptr(1),
			Max:     float64Ptr(65535),
		},
	}

	template, err := manager.CreateTemplateFromConfig(config, "Test Template", "A test template", variables)
	require.NoError(t, err)
	require.NotNil(t, template)

	assert.Equal(t, "Test Template", template.Name)
	assert.Equal(t, "A test template", template.Description)
	assert.Equal(t, "custom", template.Category)
	assert.NotEmpty(t, template.ID)
	assert.NotEmpty(t, template.Variables)

	// Test saving template
	templatePath := tempDir + "/test_template.yaml"
	err = manager.SaveTemplate(template, templatePath)
	require.NoError(t, err)

	// Test loading template
	loadedTemplate, err := manager.LoadTemplate(templatePath)
	require.NoError(t, err)
	require.NotNil(t, loadedTemplate)

	assert.Equal(t, template.ID, loadedTemplate.ID)
	assert.Equal(t, template.Name, loadedTemplate.Name)
	assert.Equal(t, template.Description, loadedTemplate.Description)
}

// TestConfigurationTemplateManagerApply tests template application
func TestConfigurationTemplateManagerApply(t *testing.T) {
	// Create default templates
	templates := CreateDefaultTemplates()
	require.NotEmpty(t, templates)

	manager := NewConfigurationTemplateManager()

	// Add template
	devTemplate := templates["development"]
	manager.templates[devTemplate.ID] = devTemplate

	// Test applying template
	variables := map[string]interface{}{
		"workspace_path": "~/test_workspace",
		"debug_enabled":  true,
	}

	config, err := manager.ApplyTemplate(devTemplate.ID, variables)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "development", config.Application.Environment)
	assert.True(t, config.Development.Enabled)
	assert.True(t, config.Development.Debug.Enabled)
	assert.Equal(t, "debug", config.Development.Debug.Level)
}

// TestConfigurationTemplateManagerSearch tests template searching
func TestConfigurationTemplateManagerSearch(t *testing.T) {
	manager := NewConfigurationTemplateManager()

	// Add some test templates
	templates := CreateDefaultTemplates()
	for _, template := range templates {
		manager.templates[template.ID] = template
	}

	// Test search by name
	results := manager.SearchTemplates("development")
	assert.Len(t, results, 1)
	assert.Equal(t, "development", results[0].ID)

	// Test search by description
	results = manager.SearchTemplates("optimized")
	assert.Len(t, results, 3) // development, production, testing

	// Test search by category
	results = manager.SearchTemplates("environment")
	assert.Len(t, results, 3)

	// Test search by tag
	results = manager.SearchTemplates("debug")
	assert.Len(t, results, 1)
	assert.Equal(t, "development", results[0].ID)
}

// TestConfigurationTemplateValidation tests template variable validation
func TestConfigurationTemplateValidation(t *testing.T) {
	manager := NewConfigurationTemplateManager()

	// Test with required variable missing
	template := &ConfigurationTemplate{
		ID:          "test_template",
		Name:        "Test Template",
		Description: "Test template",
		Category:    "test",
		Variables: map[string]*TemplateVariable{
			"required_var": {
				Name:     "Required Variable",
				Type:     "string",
				Required: true,
			},
		},
		Config: &HelixConfig{},
	}

	// Test with missing required variable
	variables := map[string]interface{}{
		"other_var": "value",
	}

	_, err := manager.processTemplate(template, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required variable not provided: required_var")

	// Test with invalid variable type
	variables = map[string]interface{}{
		"required_var": 123, // string expected
	}

	_, err = manager.processTemplate(template, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

// TestConfigurationTemplateVariableConstraints tests variable constraint validation
func TestConfigurationTemplateVariableConstraints(t *testing.T) {
	manager := NewConfigurationTemplateManager()

	template := &ConfigurationTemplate{
		ID:          "test_template",
		Name:        "Test Template",
		Description: "Test template",
		Variables: map[string]*TemplateVariable{
			"constrained_var": {
				Name:      "Constrained Variable",
				Type:      "string",
				MinLength: intPtr(5),
				MaxLength: intPtr(10),
				Pattern:   "^[a-z]+$",
			},
		},
		Config: &HelixConfig{},
	}

	// Test with too short value
	variables := map[string]interface{}{
		"constrained_var": "abc",
	}

	_, err := manager.processTemplate(template, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")

	// Test with too long value
	variables = map[string]interface{}{
		"constrained_var": "abcdefghijklmnopqrstuvwxyz",
	}

	_, err = manager.processTemplate(template, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too long")

	// Test with invalid pattern
	variables = map[string]interface{}{
		"constrained_var": "ABC123",
	}

	_, err = manager.processTemplate(template, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't match required pattern")

	// Test with valid value
	variables = map[string]interface{}{
		"constrained_var": "abcde",
	}

	config, err := manager.processTemplate(template, variables)
	assert.NoError(t, err)
	assert.NotNil(t, config)
}

// TestDefaultTemplates tests default template creation
func TestDefaultTemplates(t *testing.T) {
	templates := CreateDefaultTemplates()
	require.NotEmpty(t, templates)

	// Test development template
	devTemplate, exists := templates["development"]
	require.True(t, exists)
	assert.Equal(t, "Development Environment", devTemplate.Name)
	assert.Equal(t, "development", devTemplate.Config.Application.Environment)
	assert.True(t, devTemplate.Config.Development.Enabled)
	assert.True(t, devTemplate.Config.Development.Debug.Enabled)
	assert.Equal(t, "debug", devTemplate.Config.Development.Debug.Level)
	assert.Equal(t, 8080, devTemplate.Config.Server.Port)
	assert.False(t, devTemplate.Config.Server.SSLEnabled)

	// Test production template
	prodTemplate, exists := templates["production"]
	require.True(t, exists)
	assert.Equal(t, "Production Environment", prodTemplate.Name)
	assert.Equal(t, "production", prodTemplate.Config.Application.Environment)
	assert.False(t, prodTemplate.Config.Development.Enabled)
	assert.Equal(t, "error", prodTemplate.Config.Application.Logging.Level)
	assert.Equal(t, 443, prodTemplate.Config.Server.Port)
	assert.True(t, prodTemplate.Config.Server.SSLEnabled)
	assert.True(t, prodTemplate.Config.Security.EncryptionEnabled)

	// Test testing template
	testTemplate, exists := templates["testing"]
	require.True(t, exists)
	assert.Equal(t, "Testing Environment", testTemplate.Name)
	assert.Equal(t, "testing", testTemplate.Config.Application.Environment)
	assert.True(t, testTemplate.Config.Development.Testing.Enabled)
	assert.Equal(t, 0, testTemplate.Config.Server.Port) // Random port
	assert.False(t, testTemplate.Config.Server.SSLEnabled)
	assert.Equal(t, 5, testTemplate.Config.Workers.MaxConcurrentTasks)
}

// TestTemplateVariableSubstitution tests template variable substitution
func TestTemplateVariableSubstitution(t *testing.T) {
	manager := NewConfigurationTemplateManager()

	// Create template with string variables
	template := &ConfigurationTemplate{
		ID:          "test_template",
		Name:        "Test Template",
		Description: "Test template",
		Config: &HelixConfig{
			Application: ApplicationConfig{
				Name:        "{{app_name}}",
				Description: "{{app_desc}}",
				Version:     "1.0.0",
			},
		},
		Variables: map[string]*TemplateVariable{
			"app_name": {
				Name:     "App Name",
				Type:     "string",
				Default:  "Default App",
				Required: false,
			},
			"app_desc": {
				Name:     "App Description",
				Type:     "string",
				Default:  "Default Description",
				Required: false,
			},
		},
	}

	variables := map[string]interface{}{
		"app_name": "My Custom App",
		"app_desc": "My custom description",
	}

	config, err := manager.processTemplate(template, variables)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "My Custom App", config.Application.Name)
	assert.Equal(t, "My custom description", config.Application.Description)
	assert.Equal(t, "1.0.0", config.Application.Version) // Unchanged
}

// TestConfigurationValidationIntegration tests validation integration
func TestConfigurationValidationIntegration(t *testing.T) {
	validator := NewConfigurationValidator(true)

	// Test comprehensive validation
	config := &HelixConfig{
		Version: "1.0.0",
		Application: ApplicationConfig{
			Name:        "Test App",
			Description: "Test Description",
			Version:     "1.0.0",
			Environment: "invalid_env", // Should be in enum
			Workspace: WorkspaceConfig{
				DefaultPath: "~/test",
				AutoSave:    true,
			},
		},
		Server: ServerConfig{
			Address:      "localhost",
			Port:         -1, // Should be valid port
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		LLM: LLMConfig{
			DefaultProvider: "invalid_provider", // Should be in enum
			DefaultModel:    "test-model",
			MaxTokens:       -1,  // Should be positive
			Temperature:     3.0, // Should be <= 2.0
		},
	}

	result := validator.Validate(config)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)

	// Check for specific validation errors
	errorCodes := make(map[string]bool)
	for _, err := range result.Errors {
		errorCodes[err.Code] = true
	}

	// Should have errors for:
	// - Invalid environment
	// - Invalid port
	// - Invalid provider
	// - Invalid max tokens
	// - Invalid temperature
	assert.True(t, errorCodes["FIELD_SCHEMA_ERROR"])
	assert.True(t, errorCodes["CUSTOM_RULE_ERROR"])
}

// TestConfigurationMigrationBackup tests migration backup functionality
func TestConfigurationMigrationBackup(t *testing.T) {
	migrator := NewConfigurationMigrator("1.0.0")

	config := &HelixConfig{
		Version: "1.0.0",
		Application: ApplicationConfig{
			Name: "Test App",
		},
	}

	// Test migration with backup
	migration := Migration{
		From:   "1.0.0",
		To:     "1.1.0",
		Name:   "test_migration",
		Desc:   "Test migration",
		Backup: true,
		Up: func(config *HelixConfig) error {
			config.Application.Description = "Migrated"
			return nil
		},
		Down: func(config *HelixConfig) error {
			return nil
		},
	}

	migrator.migrations["1.0.0"] = []Migration{migration}

	err := migrator.Migrate(config, "1.1.0")
	assert.NoError(t, err)
	assert.Equal(t, "1.1.0", config.Version)
	assert.Equal(t, "Migrated", config.Application.Description)

	// Check that backup was created (in temp dir)
	tempDir := os.TempDir()
	backupFiles, err := filepath.Glob(filepath.Join(tempDir, "helix_config_backup_*.json"))
	assert.NoError(t, err)
	assert.NotEmpty(t, backupFiles)
}

// TestAdvancedConfigurationPerformance tests performance of advanced config operations
func TestAdvancedConfigurationPerformance(t *testing.T) {
	// Test validation performance
	validator := NewConfigurationValidator(true)
	config := getDefaultConfig()

	start := time.Now()
	for i := 0; i < 1000; i++ {
		validator.Validate(config)
	}
	duration := time.Since(start)

	// Should complete 1000 validations in reasonable time
	assert.Less(t, duration, 1*time.Second, "Validation should be fast")

	// Test transformation performance
	transformer := NewConfigurationTransformer()
	variables := map[string]interface{}{
		"test_var": "test_value",
	}

	start = time.Now()
	for i := 0; i < 1000; i++ {
		transformer.Transform(config, variables)
	}
	duration = time.Since(start)

	// Should complete 1000 transformations in reasonable time
	assert.Less(t, duration, 500*time.Millisecond, "Transformation should be fast")
}

// BenchmarkAdvancedConfiguration benchmarks advanced configuration operations
func BenchmarkAdvancedConfiguration(b *testing.B) {
	validator := NewConfigurationValidator(true)
	config := getDefaultConfig()

	b.Run("Validation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validator.Validate(config)
		}
	})

	transformer := NewConfigurationTransformer()
	variables := map[string]interface{}{
		"test_var": "test_value",
	}

	b.Run("Transformation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			transformer.Transform(config, variables)
		}
	})

	manager := NewConfigurationTemplateManager()
	templates := CreateDefaultTemplates()

	b.Run("TemplateApplication", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager.ApplyTemplate("development", map[string]interface{}{})
		}
	})
}
