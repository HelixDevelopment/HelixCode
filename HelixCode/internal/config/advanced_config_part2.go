package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Continued from previous file...

// Helper methods for transformation (continued)

func (t *ConfigurationTransformer) getValueAtPath(obj interface{}, path string) interface{} {
	if path == "" {
		return obj
	}

	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		switch val := current.(type) {
		case map[string]interface{}:
			current = val[part]
		default:
			// Use reflection for struct fields
			r := reflect.ValueOf(current)
			if r.Kind() == reflect.Ptr {
				r = r.Elem()
			}
			if r.Kind() == reflect.Struct {
				field := r.FieldByName(part)
				if field.IsValid() {
					current = field.Interface()
				} else {
					return nil
				}
			} else {
				return nil
			}
		}

		if current == nil {
			return nil
		}
	}

	return current
}

func (t *ConfigurationTransformer) setValueAtPath(obj interface{}, path string, value interface{}) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	parts := strings.Split(path, ".")
	current := obj

	// Navigate to parent
	for i, part := range parts {
		isLast := i == len(parts)-1

		switch val := current.(type) {
		case map[string]interface{}:
			if isLast {
				val[part] = value
			} else {
				if val[part] == nil {
					val[part] = make(map[string]interface{})
				}
				current = val[part]
			}
		default:
			// Use reflection for struct fields
			r := reflect.ValueOf(current)
			if r.Kind() == reflect.Ptr {
				r = r.Elem()
			}

			if r.Kind() == reflect.Struct {
				if isLast {
					field := r.FieldByName(part)
					if field.IsValid() && field.CanSet() {
						// Convert value to field type
						converted, err := t.convertValueForField(value, field.Type())
						if err != nil {
							return err
						}
						field.Set(reflect.ValueOf(converted))
					}
				} else {
					field := r.FieldByName(part)
					if field.IsValid() {
						if field.IsNil() {
							// Initialize pointer field
							field.Set(reflect.New(field.Type().Elem()))
						}
						current = field.Interface()
					} else {
						return fmt.Errorf("field not found: %s", part)
					}
				}
			} else {
				return fmt.Errorf("cannot set field on non-struct type")
			}
		}
	}

	return nil
}

func (t *ConfigurationTransformer) convertValue(value interface{}, parameters interface{}) (interface{}, error) {
	// Implementation depends on specific conversion type
	// This is a placeholder for more complex conversions

	switch conv := parameters.(type) {
	case string:
		switch conv {
		case "to_upper":
			if str, ok := value.(string); ok {
				return strings.ToUpper(str), nil
			}
		case "to_lower":
			if str, ok := value.(string); ok {
				return strings.ToLower(str), nil
			}
		case "to_int":
			if num, ok := t.getNumberValue(value); ok {
				return int(num), nil
			}
		}
	case map[string]interface{}:
		// Complex conversion with parameters
		if from, ok := conv["from"].(string); ok {
			if to, ok := conv["to"].(string); ok {
				return t.convertBetweenTypes(value, from, to)
			}
		}
	}

	return value, nil
}

func (t *ConfigurationTransformer) convertBetweenTypes(value interface{}, from, to string) (interface{}, error) {
	// Handle specific type conversions
	switch from {
	case "string":
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for conversion")
		}
		switch to {
		case "int":
			return strconv.Atoi(str)
		case "float":
			return strconv.ParseFloat(str, 64)
		case "bool":
			return t.parseBool(str), nil
		case "duration":
			return time.ParseDuration(str)
		}
	case "int":
		num, ok := value.(int)
		if !ok {
			return nil, fmt.Errorf("expected int for conversion")
		}
		switch to {
		case "string":
			return strconv.Itoa(num), nil
		case "float":
			return float64(num), nil
		}
	}

	return value, nil
}

func (t *ConfigurationTransformer) applyTemplate(value interface{}, parameters interface{}, variables map[string]interface{}) (interface{}, error) {
	// Simple template implementation
	// In real implementation, use text/template

	templateStr, ok := parameters.(string)
	if !ok {
		return value, nil
	}

	// Replace simple variables
	result := templateStr
	for key, val := range variables {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", val))
	}

	return result, nil
}

func (t *ConfigurationTransformer) calculateValue(value interface{}, parameters interface{}, variables map[string]interface{}) (interface{}, error) {
	// Simple calculation implementation
	// In real implementation, use expression parser

	calc, ok := parameters.(map[string]interface{})
	if !ok {
		return value, nil
	}

	operation, ok := calc["operation"].(string)
	if !ok {
		return value, nil
	}

	switch operation {
	case "add":
		operand, ok := calc["operand"].(float64)
		if !ok {
			return value, nil
		}
		if num, ok := t.getNumberValue(value); ok {
			return num + operand, nil
		}
	case "multiply":
		factor, ok := calc["factor"].(float64)
		if !ok {
			return value, nil
		}
		if num, ok := t.getNumberValue(value); ok {
			return num * factor, nil
		}
	case "format":
		format, ok := calc["format"].(string)
		if !ok {
			return value, nil
		}
		return fmt.Sprintf(format, value), nil
	}

	return value, nil
}

func (t *ConfigurationTransformer) findPatternMatches(config *HelixConfig, pattern string) []interface{} {
	// Simple pattern matching
	// In real implementation, use regex or glob patterns

	matches := make([]interface{}, 0)

	if pattern == "*" {
		// Return all fields (simplified)
		r := reflect.ValueOf(config)
		for i := 0; i < r.NumField(); i++ {
			field := r.Field(i)
			if field.CanInterface() {
				matches = append(matches, field.Interface())
			}
		}
	} else if strings.Contains(pattern, ".") {
		// Specific field path
		value := t.getValueAtPath(config, pattern)
		if value != nil {
			matches = append(matches, value)
		}
	}

	return matches
}

func (t *ConfigurationTransformer) deepCopy(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func (t *ConfigurationTransformer) deepCopyValue(value interface{}) interface{} {
	data, _ := json.Marshal(value)
	var result interface{}
	json.Unmarshal(data, &result)
	return result
}

// ConfigurationTemplateManager manages configuration templates
type ConfigurationTemplateManager struct {
	templates map[string]*ConfigurationTemplate
	paths     []string
}

// NewConfigurationTemplateManager creates a new template manager
func NewConfigurationTemplateManager(paths ...string) *ConfigurationTemplateManager {
	if len(paths) == 0 {
		paths = []string{
			"~/.helixcode/templates",
			"./templates",
			"/etc/helixcode/templates",
		}
	}

	manager := &ConfigurationTemplateManager{
		templates: make(map[string]*ConfigurationTemplate),
		paths:     paths,
	}

	manager.loadTemplates()

	return manager
}

// LoadTemplate loads a template from file
func (tm *ConfigurationTemplateManager) LoadTemplate(filePath string) (*ConfigurationTemplate, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var template ConfigurationTemplate
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, err
	}

	// Set file path as ID if not set
	if template.ID == "" {
		template.ID = filepath.Base(filePath)
	}

	// Set file modification time
	if info, err := os.Stat(filePath); err == nil {
		template.UpdatedAt = info.ModTime()
		template.CreatedAt = info.ModTime() // Use mod time as creation if not set
	}

	tm.templates[template.ID] = &template

	return &template, nil
}

// SaveTemplate saves a template to file
func (tm *ConfigurationTemplateManager) SaveTemplate(template *ConfigurationTemplate, filePath string) error {
	// Update timestamps
	template.UpdatedAt = time.Now()
	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now()
	}

	data, err := yaml.Marshal(template)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// GetTemplate gets a template by ID
func (tm *ConfigurationTemplateManager) GetTemplate(id string) (*ConfigurationTemplate, bool) {
	template, exists := tm.templates[id]
	return template, exists
}

// ListTemplates lists all available templates
func (tm *ConfigurationTemplateManager) ListTemplates() []*ConfigurationTemplate {
	templates := make([]*ConfigurationTemplate, 0, len(tm.templates))
	for _, template := range tm.templates {
		templates = append(templates, template)
	}
	return templates
}

// SearchTemplates searches templates by criteria
func (tm *ConfigurationTemplateManager) SearchTemplates(query string) []*ConfigurationTemplate {
	results := make([]*ConfigurationTemplate, 0)

	for _, template := range tm.templates {
		if tm.matchesTemplate(template, query) {
			results = append(results, template)
		}
	}

	return results
}

// ApplyTemplate applies a template with variables
func (tm *ConfigurationTemplateManager) ApplyTemplate(templateID string, variables map[string]interface{}) (*HelixConfig, error) {
	template, exists := tm.GetTemplate(templateID)
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	return tm.processTemplate(template, variables)
}

// CreateTemplateFromConfig creates a template from existing configuration
func (tm *ConfigurationTemplateManager) CreateTemplateFromConfig(config *HelixConfig, name, description string, variables map[string]*TemplateVariable) (*ConfigurationTemplate, error) {
	template := &ConfigurationTemplate{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Category:    "custom",
		Author:      "system",
		Version:     "1.0.0",
		Config:      config,
		Variables:   variables,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tm.templates[template.ID] = template

	return template, nil
}

// loadTemplates loads all templates from configured paths
func (tm *ConfigurationTemplateManager) loadTemplates() {
	for _, path := range tm.paths {
		// Expand ~ to home directory
		if strings.HasPrefix(path, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				path = filepath.Join(home, path[2:])
			}
		}

		// Check if path exists
		if _, err := os.Stat(path); err != nil {
			continue
		}

		// Load all .yaml and .yml files
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && (strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml")) {
				tm.LoadTemplate(filePath)
			}

			return nil
		})

		if err != nil {
			// Log error but continue loading other paths
			continue
		}
	}
}

// matchesTemplate checks if a template matches search query
func (tm *ConfigurationTemplateManager) matchesTemplate(template *ConfigurationTemplate, query string) bool {
	query = strings.ToLower(query)

	// Check name
	if strings.Contains(strings.ToLower(template.Name), query) {
		return true
	}

	// Check description
	if strings.Contains(strings.ToLower(template.Description), query) {
		return true
	}

	// Check category
	if strings.Contains(strings.ToLower(template.Category), query) {
		return true
	}

	// Check tags
	for _, tag := range template.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}

	return false
}

// processTemplate processes a template with variables
func (tm *ConfigurationTemplateManager) processTemplate(template *ConfigurationTemplate, variables map[string]interface{}) (*HelixConfig, error) {
	// Start with template config
	config := &HelixConfig{}
	if err := tm.deepCopy(template.Config, config); err != nil {
		return nil, err
	}

	// Apply variable substitutions
	if err := tm.applyVariables(config, template.Variables, variables); err != nil {
		return nil, err
	}

	// Validate processed configuration
	if err := tm.validateTemplateVariables(template, variables); err != nil {
		return nil, err
	}

	return config, nil
}

// applyVariables applies template variables to configuration
func (tm *ConfigurationTemplateManager) applyVariables(config *HelixConfig, templateVars map[string]*TemplateVariable, variables map[string]interface{}) error {
	// Validate required variables
	for name, templateVar := range templateVars {
		value, exists := variables[name]
		if !exists && templateVar.Required {
			return fmt.Errorf("required variable not provided: %s", name)
		}

		if exists {
			// Validate variable type and constraints
			if err := tm.validateVariableValue(name, value, templateVar); err != nil {
				return err
			}
		}
	}

	// Apply substitutions using reflection
	return tm.applyVariableSubstitutions(config, variables)
}

// applyVariableSubstitutions applies variable substitutions to configuration
func (tm *ConfigurationTemplateManager) applyVariableSubstitutions(config interface{}, variables map[string]interface{}) error {
	// In a real implementation, this would parse and replace template variables
	// For now, use a simple string substitution approach

	r := reflect.ValueOf(config)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	// Iterate through all string fields and replace variables
	for i := 0; i < r.NumField(); i++ {
		field := r.Field(i)
		if !field.CanInterface() {
			continue
		}

		fieldType := field.Type()
		if fieldType.Kind() == reflect.String {
			str := field.String()
			// Replace {{variable}} patterns
			for key, value := range variables {
				placeholder := fmt.Sprintf("{{%s}}", key)
				if strings.Contains(str, placeholder) {
					replacement := fmt.Sprintf("%v", value)
					str = strings.ReplaceAll(str, placeholder, replacement)
					field.SetString(str)
				}
			}
		} else if fieldType.Kind() == reflect.Struct {
			// Recursively process nested structs
			nested := field.Addr().Interface()
			if err := tm.applyVariableSubstitutions(nested, variables); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateVariableValue validates a variable value against template definition
func (tm *ConfigurationTemplateManager) validateVariableValue(name string, value interface{}, templateVar *TemplateVariable) error {
	// Type validation
	switch templateVar.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("variable '%s' must be a string", name)
		}
		str := value.(string)

		// Length validation
		if templateVar.MinLength != nil && len(str) < *templateVar.MinLength {
			return fmt.Errorf("variable '%s' too short, minimum length is %d", name, *templateVar.MinLength)
		}
		if templateVar.MaxLength != nil && len(str) > *templateVar.MaxLength {
			return fmt.Errorf("variable '%s' too long, maximum length is %d", name, *templateVar.MaxLength)
		}

		// Pattern validation
		if templateVar.Pattern != "" {
			if matched, err := regexp.MatchString(templateVar.Pattern, str); err != nil {
				return fmt.Errorf("invalid pattern for variable '%s': %v", name, err)
			} else if !matched {
				return fmt.Errorf("variable '%s' doesn't match required pattern", name)
			}
		}

	case "number":
		num, ok := tm.getNumberValue(value)
		if !ok {
			return fmt.Errorf("variable '%s' must be a number", name)
		}

		// Range validation
		if templateVar.Min != nil && num < *templateVar.Min {
			return fmt.Errorf("variable '%s' too small, minimum value is %g", name, *templateVar.Min)
		}
		if templateVar.Max != nil && num > *templateVar.Max {
			return fmt.Errorf("variable '%s' too large, maximum value is %g", name, *templateVar.Max)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("variable '%s' must be a boolean", name)
		}
	case "array":
		if reflect.TypeOf(value).Kind() != reflect.Slice {
			return fmt.Errorf("variable '%s' must be an array", name)
		}
	}

	// Enum validation
	if len(templateVar.Enum) > 0 {
		found := false
		for _, enumValue := range templateVar.Enum {
			if reflect.DeepEqual(value, enumValue) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("variable '%s' must be one of: %v", name, templateVar.Enum)
		}
	}

	return nil
}

// validateTemplateVariables validates that all provided variables are valid for the template
func (tm *ConfigurationTemplateManager) validateTemplateVariables(template *ConfigurationTemplate, variables map[string]interface{}) error {
	// Check for unknown variables
	for varName := range variables {
		if _, exists := template.Variables[varName]; !exists {
			return fmt.Errorf("unknown variable: %s", varName)
		}
	}

	return nil
}

// getNumberValue gets numeric value from interface

// CreateDefaultTemplates creates default configuration templates
func CreateDefaultTemplates() map[string]*ConfigurationTemplate {
	templates := make(map[string]*ConfigurationTemplate)

	// Development template
	templates["development"] = &ConfigurationTemplate{
		ID:          "development",
		Name:        "Development Environment",
		Description: "Configuration optimized for development",
		Category:    "environment",
		Author:      "system",
		Version:     "1.0.0",
		Tags:        []string{"development", "debug"},
		Config:      createDevelopmentTemplateConfig(),
		Variables: map[string]*TemplateVariable{
			"workspace_path": {
				Name:     "Workspace Path",
				Type:     "string",
				Default:  "~/development/helixcode",
				Required: true,
				Pattern:  "^[^/].*",
			},
			"debug_enabled": {
				Name:     "Debug Enabled",
				Type:     "boolean",
				Default:  true,
				Required: false,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Production template
	templates["production"] = &ConfigurationTemplate{
		ID:          "production",
		Name:        "Production Environment",
		Description: "Configuration optimized for production",
		Category:    "environment",
		Author:      "system",
		Version:     "1.0.0",
		Tags:        []string{"production", "security"},
		Config:      createProductionTemplateConfig(),
		Variables: map[string]*TemplateVariable{
			"server_port": {
				Name:     "Server Port",
				Type:     "number",
				Default:  443,
				Required: true,
				Min:      float64Ptr(1),
				Max:      float64Ptr(65535),
			},
			"ssl_enabled": {
				Name:     "SSL Enabled",
				Type:     "boolean",
				Default:  true,
				Required: false,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Testing template
	templates["testing"] = &ConfigurationTemplate{
		ID:          "testing",
		Name:        "Testing Environment",
		Description: "Configuration optimized for testing",
		Category:    "environment",
		Author:      "system",
		Version:     "1.0.0",
		Tags:        []string{"testing", "ci"},
		Config:      createTestingTemplateConfig(),
		Variables: map[string]*TemplateVariable{
			"auto_test": {
				Name:     "Auto Test",
				Type:     "boolean",
				Default:  true,
				Required: false,
			},
			"test_timeout": {
				Name:     "Test Timeout",
				Type:     "number",
				Default:  30,
				Required: false,
				Min:      float64Ptr(1),
				Max:      float64Ptr(300),
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return templates
}

// getDefaultConfig returns a default configuration
func getDefaultConfig() *HelixConfig {
	now := time.Now()

	return &HelixConfig{
		Version:     "1.0.0",
		LastUpdated: now,
		UpdatedBy:   "system",

		Application: ApplicationConfig{
			Name:        "HelixCode",
			Description: "Distributed AI Development Platform",
			Version:     "1.0.0",
			Environment: "development",
		},

		Server: ServerConfig{
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},

		LLM: LLMConfig{
			Temperature: 0.7,
			MaxTokens:   4096,
		},
	}
}

// createDevelopmentTemplateConfig creates development template config
func createDevelopmentTemplateConfig() *HelixConfig {
	config := getDefaultConfig()
	config.Application.Environment = "development"
	config.Development.Enabled = true
	config.Development.Debug.Enabled = true
	config.Development.Debug.Level = "debug"
	config.Development.HotReload.Enabled = true
	config.Application.Logging.Level = "debug"
	config.Server.Port = 8080
	config.Server.SSLEnabled = false
	config.Security.EncryptionEnabled = false
	return config
}

// createProductionTemplateConfig creates production template config
func createProductionTemplateConfig() *HelixConfig {
	config := getDefaultConfig()
	config.Application.Environment = "production"
	config.Development.Enabled = false
	config.Application.Logging.Level = "error"
	config.Server.Port = 443
	config.Server.SSLEnabled = true
	config.Security.EncryptionEnabled = true
	config.Security.Authorization.Enabled = true
	config.Security.Audit.Enabled = true
	config.Workflows.Autonomy.DefaultLevel = "basic"
	config.LLM.CostManagement.BudgetEnabled = true
	return config
}

// createTestingTemplateConfig creates testing template config
func createTestingTemplateConfig() *HelixConfig {
	config := getDefaultConfig()
	config.Application.Environment = "testing"
	config.Development.Enabled = true
	config.Development.Testing.Enabled = true
	config.Application.Logging.Level = "warn"
	config.Server.Port = 0 // Random port for testing
	config.Server.SSLEnabled = false
	config.Security.EncryptionEnabled = false
	config.Workers.MaxConcurrentTasks = 5 // Reduce for testing
	config.LLM.MaxTokens = 2048           // Reduce for testing
	return config
}
