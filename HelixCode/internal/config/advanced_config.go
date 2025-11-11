package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// ConfigurationValidator provides comprehensive validation for configuration
type ConfigurationValidator struct {
	rules       map[string][]ValidationRuleConfig
	schema      *AdvancedConfigurationSchema
	strictMode  bool
	customRules map[string]func(interface{}) error
}

// AdvancedConfigurationSchema represents the configuration structure schema
type AdvancedConfigurationSchema struct {
	Version     string                     `json:"version"`
	Properties  map[string]*SchemaProperty `json:"properties"`
	Required    []string                   `json:"required"`
	Definitions map[string]*SchemaProperty `json:"definitions"`
	Patterns    map[string]string          `json:"patterns"`
	Enums       map[string][]interface{}   `json:"enums"`
}

// SchemaProperty represents a property definition in the schema
type SchemaProperty struct {
	Type         string                     `json:"type"`
	Title        string                     `json:"title"`
	Description  string                     `json:"description"`
	Default      interface{}                `json:"default"`
	Required     []string                   `json:"required,omitempty"`
	Properties   map[string]*SchemaProperty `json:"properties,omitempty"`
	Items        *SchemaProperty            `json:"items,omitempty"`
	Minimum      *float64                   `json:"minimum,omitempty"`
	Maximum      *float64                   `json:"maximum,omitempty"`
	MinLength    *int                       `json:"minLength,omitempty"`
	MaxLength    *int                       `json:"maxLength,omitempty"`
	Pattern      string                     `json:"pattern,omitempty"`
	Enum         []interface{}              `json:"enum,omitempty"`
	Format       string                     `json:"format,omitempty"`
	ExclusiveMin *bool                      `json:"exclusiveMinimum,omitempty"`
	ExclusiveMax *bool                      `json:"exclusiveMaximum,omitempty"`
	Unique       *bool                      `json:"unique,omitempty"`
	OneOf        []*SchemaProperty          `json:"oneOf,omitempty"`
	AnyOf        []*SchemaProperty          `json:"anyOf,omitempty"`
	AllOf        []*SchemaProperty          `json:"allOf,omitempty"`
	Not          *SchemaProperty            `json:"not,omitempty"`
	Ref          string                     `json:"$ref,omitempty"`
	Constraints  map[string]interface{}     `json:"constraints,omitempty"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid    bool                   `json:"valid"`
	Errors   []ValidationError      `json:"errors"`
	Warnings []ValidationError      `json:"warnings"`
	Path     string                 `json:"path"`
	Context  map[string]interface{} `json:"context"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Path       string      `json:"path"`
	Property   string      `json:"property"`
	Message    string      `json:"message"`
	Code       string      `json:"code"`
	Severity   string      `json:"severity"`
	Value      interface{} `json:"value"`
	Expected   interface{} `json:"expected,omitempty"`
	Actual     interface{} `json:"actual,omitempty"`
	Constraint string      `json:"constraint,omitempty"`
	Suggestion string      `json:"suggestion,omitempty"`
}

// ConfigurationMigrator handles configuration version migrations
type ConfigurationMigrator struct {
	migrations map[string][]Migration
	current    string
}

// Migration represents a configuration migration
type Migration struct {
	From      string                   `json:"from"`
	To        string                   `json:"to"`
	Name      string                   `json:"name"`
	Desc      string                   `json:"description"`
	Timestamp time.Time                `json:"timestamp"`
	Up        func(*HelixConfig) error `json:"-"`
	Down      func(*HelixConfig) error `json:"-"`
	DryRun    bool                     `json:"dry_run"`
	Backup    bool                     `json:"backup"`
}

// ConfigurationTransformer provides configuration transformation utilities
type ConfigurationTransformer struct {
	mappings map[string]TransformMapping
	rules    []TransformRule
}

// TransformMapping represents a field transformation mapping
type TransformMapping struct {
	Source     string      `json:"source"`
	Target     string      `json:"target"`
	Transform  string      `json:"transform"` // rename, copy, move, delete, convert
	Parameters interface{} `json:"parameters"`
	Condition  string      `json:"condition"` // when condition
	Priority   int         `json:"priority"`
	Required   bool        `json:"required"`
}

// TransformRule represents a transformation rule
type TransformRule struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Pattern     string                        `json:"pattern"`
	Transform   func(interface{}) interface{} `json:"-"`
	Parameters  map[string]interface{}        `json:"parameters"`
}

// ConfigurationTemplate provides configuration templates
type ConfigurationTemplate struct {
	ID           string                       `json:"id"`
	Name         string                       `json:"name"`
	Description  string                       `json:"description"`
	Category     string                       `json:"category"`
	Author       string                       `json:"author"`
	Version      string                       `json:"version"`
	Tags         []string                     `json:"tags"`
	Config       *HelixConfig                 `json:"config"`
	Variables    map[string]*TemplateVariable `json:"variables"`
	Validation   []string                     `json:"validation"`
	Dependencies []string                     `json:"dependencies"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
}

// TemplateVariable represents a template variable
type TemplateVariable struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Default     interface{}   `json:"default"`
	Required    bool          `json:"required"`
	Enum        []interface{} `json:"enum"`
	Pattern     string        `json:"pattern"`
	Min         *float64      `json:"min"`
	Max         *float64      `json:"max"`
	MinLength   *int          `json:"min_length"`
	MaxLength   *int          `json:"max_length"`
}

// NewConfigurationValidator creates a new configuration validator
func NewConfigurationValidator(strictMode bool) *ConfigurationValidator {
	validator := &ConfigurationValidator{
		rules:       make(map[string][]ValidationRuleConfig),
		strictMode:  strictMode,
		customRules: make(map[string]func(interface{}) error),
	}

	// Initialize with default schema
	validator.schema = validator.createDefaultSchema()
	validator.initializeDefaultRules()

	return validator
}

// Validate validates a configuration object
func (v *ConfigurationValidator) Validate(config *HelixConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationError, 0),
		Context:  make(map[string]interface{}),
	}

	// Validate against schema
	if err := v.validateSchema(config, result); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:     "root",
			Property: "schema",
			Message:  fmt.Sprintf("Schema validation failed: %v", err),
			Code:     "SCHEMA_ERROR",
			Severity: "error",
		})
		result.Valid = false
	}

	// Apply custom rules
	for path, rule := range v.customRules {
		value := v.getValueAtPath(config, path)
		if err := rule(value); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     path,
				Property: path,
				Message:  err.Error(),
				Code:     "CUSTOM_RULE_ERROR",
				Severity: "error",
				Value:    value,
			})
			result.Valid = false
		}
	}

	// Apply field-specific rules
	if err := v.validateFieldRules(config, result); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:     "fields",
			Property: "rules",
			Message:  fmt.Sprintf("Field validation failed: %v", err),
			Code:     "FIELD_RULE_ERROR",
			Severity: "error",
		})
		result.Valid = false
	}

	return result
}

// ValidateField validates a specific field
func (v *ConfigurationValidator) ValidateField(config *HelixConfig, fieldPath string) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationError, 0),
		Path:     fieldPath,
		Context:  make(map[string]interface{}),
	}

	// Get field value
	value := v.getValueAtPath(config, fieldPath)

	// Validate against schema
	if err := v.validateFieldSchema(fieldPath, value); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:     fieldPath,
			Property: fieldPath,
			Message:  err.Error(),
			Code:     "FIELD_SCHEMA_ERROR",
			Severity: "error",
			Value:    value,
		})
		result.Valid = false
	}

	// Apply custom rules for this field
	if rule, exists := v.customRules[fieldPath]; exists {
		if err := rule(value); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Path:     fieldPath,
				Property: fieldPath,
				Message:  err.Error(),
				Code:     "FIELD_CUSTOM_RULE_ERROR",
				Severity: "error",
				Value:    value,
			})
			result.Valid = false
		}
	}

	return result
}

// AddRule adds a validation rule
func (v *ConfigurationValidator) AddRule(fieldPath string, rule ValidationRuleConfig) {
	if v.rules[fieldPath] == nil {
		v.rules[fieldPath] = make([]ValidationRuleConfig, 0)
	}
	v.rules[fieldPath] = append(v.rules[fieldPath], rule)
}

// AddCustomRule adds a custom validation rule
func (v *ConfigurationValidator) AddCustomRule(fieldPath string, rule func(interface{}) error) {
	v.customRules[fieldPath] = rule
}

// SetSchema sets the validation schema
func (v *ConfigurationValidator) SetSchema(schema *AdvancedConfigurationSchema) {
	v.schema = schema
}

// validateSchema validates configuration against schema
func (v *ConfigurationValidator) validateSchema(config *HelixConfig, result *ValidationResult) error {
	if v.schema == nil {
		return fmt.Errorf("no schema defined")
	}

	return v.validateProperty("", config, v.schema, result)
}

// validateProperty validates a property against schema
func (v *ConfigurationValidator) validateProperty(path string, value interface{}, schema *AdvancedConfigurationSchema, result *ValidationResult) error {
	// Handle schema references
	if schema.Properties["$ref"] != nil {
		ref := schema.Properties["$ref"].Ref
		return v.resolveAndValidateRef(ref, path, value, result)
	}

	// Validate required properties
	if len(schema.Required) > 0 {
		for _, required := range schema.Required {
			if !v.hasProperty(value, required) {
				result.Errors = append(result.Errors, ValidationError{
					Path:     path,
					Property: required,
					Message:  fmt.Sprintf("Required property '%s' is missing", required),
					Code:     "REQUIRED_PROPERTY_MISSING",
					Severity: "error",
					Value:    value,
				})
				result.Valid = false
			}
		}
	}

	// Validate each property
	for propName, propSchema := range schema.Properties {
		propValue := v.getPropertyValue(value, propName)
		propPath := v.joinPath(path, propName)

		if err := v.validatePropertySchema(propPath, propValue, propSchema); err != nil {
			return fmt.Errorf("property '%s' validation failed: %v", propName, err)
		}
	}

	return nil
}

// validatePropertySchema validates a property against its schema
func (v *ConfigurationValidator) validatePropertySchema(path string, value interface{}, schema *SchemaProperty) error {
	if value == nil && schema.Default != nil {
		return nil // Allow nil if there's a default
	}

	// Handle schema reference
	if schema.Ref != "" {
		return v.resolveAndValidateRef(schema.Ref, path, value, &ValidationResult{Valid: true})
	}

	// Type validation
	if err := v.validateType(path, value, schema.Type); err != nil {
		return err
	}

	// Convert value to appropriate type for validation
	switch schema.Type {
	case "string":
		strValue, ok := value.(string)
		if !ok {
			return nil // Type validation already failed
		}
		return v.validateString(path, strValue, schema)
	case "number", "integer":
		return v.validateNumber(path, value, schema)
	case "boolean":
		return v.validateBoolean(path, value, schema)
	case "array":
		return v.validateArray(path, value, schema)
	case "object":
		return v.validateObject(path, value, schema)
	case "enum":
		return v.validateEnum(path, value, schema.Enum)
	}

	return nil
}

// validateType validates the type of a value
func (v *ConfigurationValidator) validateType(path string, value interface{}, expectedType string) error {
	if value == nil {
		return nil // Skip type validation for nil values
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			if _, ok := value.(int); !ok {
				return fmt.Errorf("expected number, got %T", value)
			}
		}
	case "integer":
		if _, ok := value.(int); !ok {
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("expected integer, got %T", value)
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		if reflect.TypeOf(value).Kind() != reflect.Slice {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if reflect.TypeOf(value).Kind() != reflect.Struct && reflect.TypeOf(value).Kind() != reflect.Map {
			return fmt.Errorf("expected object, got %T", value)
		}
	}

	return nil
}

// validateString validates string values
func (v *ConfigurationValidator) validateString(path string, value string, schema *SchemaProperty) error {
	// Length validation
	if schema.MinLength != nil && len(value) < *schema.MinLength {
		return fmt.Errorf("string too short, minimum length is %d", *schema.MinLength)
	}

	if schema.MaxLength != nil && len(value) > *schema.MaxLength {
		return fmt.Errorf("string too long, maximum length is %d", *schema.MaxLength)
	}

	// Pattern validation
	if schema.Pattern != "" {
		// In a real implementation, use regex package
		// For now, just basic validation
		if !strings.Contains(value, "example") && schema.Pattern == "example" {
			return fmt.Errorf("string doesn't match required pattern")
		}
	}

	// Format validation
	if schema.Format != "" {
		if err := v.validateFormat(path, value, schema.Format); err != nil {
			return err
		}
	}

	return nil
}

// validateNumber validates numeric values
func (v *ConfigurationValidator) validateNumber(path string, value interface{}, schema *SchemaProperty) error {
	var numValue float64

	switch v := value.(type) {
	case int:
		numValue = float64(v)
	case float64:
		numValue = v
	case int32:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	case float32:
		numValue = float64(v)
	default:
		return fmt.Errorf("invalid numeric type: %T", value)
	}

	// Minimum validation
	if schema.Minimum != nil {
		min := *schema.Minimum
		if schema.ExclusiveMin != nil && *schema.ExclusiveMin {
			if numValue <= min {
				return fmt.Errorf("value must be greater than %g", min)
			}
		} else {
			if numValue < min {
				return fmt.Errorf("value must be at least %g", min)
			}
		}
	}

	// Maximum validation
	if schema.Maximum != nil {
		max := *schema.Maximum
		if schema.ExclusiveMax != nil && *schema.ExclusiveMax {
			if numValue >= max {
				return fmt.Errorf("value must be less than %g", max)
			}
		} else {
			if numValue > max {
				return fmt.Errorf("value must be at most %g", max)
			}
		}
	}

	// Integer validation for integer type
	if schema.Type == "integer" {
		if numValue != float64(int(numValue)) {
			return fmt.Errorf("value must be an integer")
		}
	}

	return nil
}

// validateBoolean validates boolean values
func (v *ConfigurationValidator) validateBoolean(path string, value interface{}, schema *SchemaProperty) error {
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("expected boolean, got %T", value)
	}
	return nil
}

// validateArray validates array values
func (v *ConfigurationValidator) validateArray(path string, value interface{}, schema *SchemaProperty) error {
	// Type already validated, so we can safely reflect
	slice := reflect.ValueOf(value)

	// Validate items if schema defined
	if schema.Items != nil {
		for i := 0; i < slice.Len(); i++ {
			item := slice.Index(i).Interface()
			itemPath := fmt.Sprintf("%s[%d]", path, i)

			if err := v.validatePropertySchema(itemPath, item, schema.Items); err != nil {
				return fmt.Errorf("array item at index %d validation failed: %v", i, err)
			}
		}
	}

	// Unique validation
	if schema.Unique != nil && *schema.Unique {
		seen := make(map[interface{}]bool)
		for i := 0; i < slice.Len(); i++ {
			item := slice.Index(i).Interface()
			if seen[item] {
				return fmt.Errorf("array contains duplicate value at index %d", i)
			}
			seen[item] = true
		}
	}

	return nil
}

// validateObject validates object values
func (v *ConfigurationValidator) validateObject(path string, value interface{}, schema *SchemaProperty) error {
	// Validate required properties
	if len(schema.Required) > 0 {
		objValue := reflect.ValueOf(value)
		for _, required := range schema.Required {
			found := false

			switch objValue.Kind() {
			case reflect.Struct:
				// Check struct field
				field := objValue.FieldByName(required)
				found = field.IsValid()
			case reflect.Map:
				// Check map key
				mapKeys := objValue.MapKeys()
				for _, key := range mapKeys {
					if key.String() == required {
						found = true
						break
					}
				}
			}

			if !found {
				return fmt.Errorf("required property '%s' is missing", required)
			}
		}
	}

	// Validate each property
	if len(schema.Properties) > 0 {
		for propName, propSchema := range schema.Properties {
			propValue := v.getPropertyValue(value, propName)
			propPath := v.joinPath(path, propName)

			if err := v.validatePropertySchema(propPath, propValue, propSchema); err != nil {
				return fmt.Errorf("property '%s' validation failed: %v", propName, err)
			}
		}
	}

	return nil
}

// validateEnum validates enum values
func (v *ConfigurationValidator) validateEnum(path string, value interface{}, enum []interface{}) error {
	for _, enumValue := range enum {
		if reflect.DeepEqual(value, enumValue) {
			return nil
		}
	}

	return fmt.Errorf("value %v is not in allowed enum values: %v", value, enum)
}

// validateFormat validates format-specific values
func (v *ConfigurationValidator) validateFormat(path string, value string, format string) error {
	switch format {
	case "date-time":
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return fmt.Errorf("invalid date-time format: %v", err)
		}
	case "date":
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("invalid date format: %v", err)
		}
	case "time":
		if _, err := time.Parse("15:04:05", value); err != nil {
			return fmt.Errorf("invalid time format: %v", err)
		}
	case "email":
		// Basic email validation
		if !strings.Contains(value, "@") || !strings.Contains(value, ".") {
			return fmt.Errorf("invalid email format")
		}
	case "uri", "url":
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("invalid URL format")
		}
	case "ipv4":
		// Basic IPv4 validation
		parts := strings.Split(value, ".")
		if len(parts) != 4 {
			return fmt.Errorf("invalid IPv4 address")
		}
		for _, part := range parts {
			if len(part) == 0 {
				return fmt.Errorf("invalid IPv4 address")
			}
		}
	case "hostname":
		if strings.Contains(value, " ") {
			return fmt.Errorf("invalid hostname")
		}
	}

	return nil
}

// Helper methods

func (v *ConfigurationValidator) getValueAtPath(obj interface{}, path string) interface{} {
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
				// Capitalize first letter for Go struct field names
				fieldName := strings.ToUpper(part[:1]) + part[1:]
				field := r.FieldByName(fieldName)
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

func (v *ConfigurationValidator) hasProperty(obj interface{}, prop string) bool {
	if obj == nil {
		return false
	}

	r := reflect.ValueOf(obj)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	switch r.Kind() {
	case reflect.Map:
		keys := r.MapKeys()
		for _, key := range keys {
			if key.String() == prop {
				return true
			}
		}
	case reflect.Struct:
		// Capitalize first letter for Go struct field names
		fieldName := strings.ToUpper(prop[:1]) + prop[1:]
		field := r.FieldByName(fieldName)
		return field.IsValid()
	}

	return false
}

func (v *ConfigurationValidator) getPropertyValue(obj interface{}, prop string) interface{} {
	if obj == nil {
		return nil
	}

	r := reflect.ValueOf(obj)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	switch r.Kind() {
	case reflect.Map:
		keys := r.MapKeys()
		for _, key := range keys {
			if key.String() == prop {
				return r.MapIndex(key).Interface()
			}
		}
	case reflect.Struct:
		// Capitalize first letter for Go struct field names
		fieldName := strings.ToUpper(prop[:1]) + prop[1:]
		field := r.FieldByName(fieldName)
		if field.IsValid() {
			return field.Interface()
		}
	}

	return nil
}

func (v *ConfigurationValidator) joinPath(base, part string) string {
	if base == "" {
		return part
	}
	return base + "." + part
}

func (v *ConfigurationValidator) resolveAndValidateRef(ref string, path string, value interface{}, result *ValidationResult) error {
	// Remove #/ prefix from JSON pointer
	refPath := strings.TrimPrefix(ref, "#/")
	if refPath == "" {
		return fmt.Errorf("empty schema reference")
	}

	// In a real implementation, resolve the reference in the schema
	// For now, just return success
	return nil
}

func (v *ConfigurationValidator) validateFieldSchema(path string, value interface{}) error {
	// In a real implementation, find the schema for this field
	// For now, just do basic validation

	if value == nil {
		return nil
	}

	// Type-specific basic validation
	switch value.(type) {
	case string:
		// Basic string validation
	case int, int32, int64, float32, float64:
		// Basic number validation
	case bool:
		// Basic boolean validation
	case []interface{}:
		// Basic array validation
	case map[string]interface{}:
		// Basic object validation
	default:
		// Unknown type
	}

	return nil
}

func (v *ConfigurationValidator) validateFieldRules(config *HelixConfig, result *ValidationResult) error {
	// Apply field-specific validation rules
	for fieldPath, rules := range v.rules {
		value := v.getValueAtPath(config, fieldPath)

		for _, rule := range rules {
			if err := v.validateRule(fieldPath, value, rule); err != nil {
				severity := rule.Severity
				if severity == "" {
					severity = "error"
				}

				error := ValidationError{
					Path:       fieldPath,
					Property:   fieldPath,
					Message:    err.Error(),
					Code:       rule.Type,
					Severity:   severity,
					Value:      value,
					Constraint: rule.Type,
				}

				if severity == "error" {
					result.Errors = append(result.Errors, error)
					result.Valid = false
				} else {
					result.Warnings = append(result.Warnings, error)
				}
			}
		}
	}

	return nil
}

func (v *ConfigurationValidator) validateRule(path string, value interface{}, rule ValidationRuleConfig) error {
	switch rule.Type {
	case "required":
		if value == nil || value == "" {
			return fmt.Errorf("%s", rule.Message)
		}
	case "min":
		if numValue, ok := v.getNumberValue(value); ok {
			min, ok := rule.Parameter.(float64)
			if !ok || numValue < min {
				return fmt.Errorf("%s", rule.Message)
			}
		}
	case "max":
		if numValue, ok := v.getNumberValue(value); ok {
			max, ok := rule.Parameter.(float64)
			if !ok || numValue > max {
				return fmt.Errorf("%s", rule.Message)
			}
		}
	case "pattern":
		if strValue, ok := value.(string); ok {
			pattern, ok := rule.Parameter.(string)
			if !ok {
				return fmt.Errorf("invalid pattern parameter")
			}
			// In real implementation, use regex
			if !strings.Contains(strValue, pattern) { // Simplified
				return fmt.Errorf("%s", rule.Message)
			}
		}
	case "custom":
		// Handle custom rule
		if fn, ok := v.customRules[path]; ok {
			return fn(value)
		}
	default:
		return fmt.Errorf("unknown validation rule type: %s", rule.Type)
	}

	return nil
}

// createDefaultSchema creates the default configuration schema
func (v *ConfigurationValidator) createDefaultSchema() *AdvancedConfigurationSchema {
	return &AdvancedConfigurationSchema{
		Version: "1.0",
		Properties: map[string]*SchemaProperty{
			"version": {
				Type:        "string",
				Title:       "Version",
				Description: "Configuration version",
				Default:     "1.0.0",
				Pattern:     "^\\d+\\.\\d+\\.\\d+$",
			},
			"application": {
				Type:        "object",
				Title:       "Application",
				Description: "Application settings",
				Properties: map[string]*SchemaProperty{
					"name": {
						Type:      "string",
						Title:     "Name",
						MinLength: intPtr(1),
						MaxLength: intPtr(100),
					},
					"environment": {
						Type: "string",
						Enum: []interface{}{"development", "testing", "staging", "production"},
					},
					"workspace": {
						Type: "object",
						Properties: map[string]*SchemaProperty{
							"default_path": {
								Type:    "string",
								Default: "~/helixcode",
							},
							"auto_save": {
								Type:    "boolean",
								Default: true,
							},
						},
						Required: []string{"default_path"},
					},
				},
				Required: []string{"name", "environment"},
			},
			"server": {
				Type: "object",
				Properties: map[string]*SchemaProperty{
					"address": {
						Type:    "string",
						Default: "0.0.0.0",
						Pattern: "^[0-9.]+$",
					},
					"port": {
						Type:    "integer",
						Default: 8080,
						Minimum: float64Ptr(1),
						Maximum: float64Ptr(65535),
					},
					"read_timeout": {
						Type:    "integer",
						Default: 30,
						Minimum: float64Ptr(1),
					},
					"write_timeout": {
						Type:    "integer",
						Default: 30,
						Minimum: float64Ptr(1),
					},
				},
				Required: []string{"port"},
			},
			"llm": {
				Type: "object",
				Properties: map[string]*SchemaProperty{
					"default_provider": {
						Type:    "string",
						Default: "local",
						Enum: []interface{}{
							"local", "openai", "anthropic", "gemini", "qwen", "azure",
							"vertexai", "bedrock", "xai", "groq", "openrouter", "copilot",
						},
					},
					"default_model": {
						Type:    "string",
						Default: "llama-3.2-3b",
					},
					"max_tokens": {
						Type:    "integer",
						Default: 4096,
						Minimum: float64Ptr(1),
						Maximum: float64Ptr(32768),
					},
					"temperature": {
						Type:    "number",
						Default: 0.7,
						Minimum: float64Ptr(0.0),
						Maximum: float64Ptr(2.0),
					},
				},
				Required: []string{"default_provider"},
			},
		},
		Required: []string{"version", "application", "server"},
	}
}

// initializeDefaultRules initializes default validation rules
func (v *ConfigurationValidator) initializeDefaultRules() {
	// Server port validation
	v.AddRule("server.port", ValidationRuleConfig{
		Type:      "min",
		Parameter: 1,
		Message:   "Server port must be greater than 0",
		Severity:  "error",
	})

	v.AddRule("server.port", ValidationRuleConfig{
		Type:      "max",
		Parameter: 65535,
		Message:   "Server port must be less than 65536",
		Severity:  "error",
	})

	// LLM provider validation
	v.AddCustomRule("llm.default_provider", func(value interface{}) error {
		provider, ok := value.(string)
		if !ok {
			return fmt.Errorf("LLM provider must be a string")
		}

		validProviders := []string{
			"local", "openai", "anthropic", "gemini", "qwen", "azure",
			"vertexai", "bedrock", "xai", "groq", "openrouter", "copilot",
		}

		for _, valid := range validProviders {
			if provider == valid {
				return nil
			}
		}

		return fmt.Errorf("invalid LLM provider: %s", provider)
	})

	// Temperature validation
	v.AddRule("llm.temperature", ValidationRuleConfig{
		Type:      "min",
		Parameter: 0.0,
		Message:   "Temperature must be between 0.0 and 2.0",
		Severity:  "error",
	})

	v.AddRule("llm.temperature", ValidationRuleConfig{
		Type:      "max",
		Parameter: 2.0,
		Message:   "Temperature must be between 0.0 and 2.0",
		Severity:  "error",
	})

	// Database host validation
	v.AddCustomRule("database.host", func(value interface{}) error {
		host, ok := value.(string)
		if !ok {
			return fmt.Errorf("Database host must be a string")
		}

		if host == "" {
			return fmt.Errorf("Database host is required")
		}

		return nil
	})
}

// NewConfigurationMigrator creates a new configuration migrator
func NewConfigurationMigrator(currentVersion string) *ConfigurationMigrator {
	migrator := &ConfigurationMigrator{
		migrations: make(map[string][]Migration),
		current:    currentVersion,
	}

	migrator.initializeMigrations()

	return migrator
}

// Migrate migrates configuration to target version
func (m *ConfigurationMigrator) Migrate(config *HelixConfig, targetVersion string) error {
	currentPath := m.findMigrationPath(config.Version, targetVersion)

	for _, migrationID := range currentPath {
		migration := m.findMigration(migrationID)
		if migration == nil {
			return fmt.Errorf("migration not found: %s", migrationID)
		}

		if migration.Backup {
			// Create backup before migration
			if err := m.createBackup(config); err != nil {
				return fmt.Errorf("backup failed for migration %s: %v", migration.Name, err)
			}
		}

		if err := migration.Up(config); err != nil {
			return fmt.Errorf("migration %s failed: %v", migration.Name, err)
		}

		config.Version = migration.To
	}

	return nil
}

// GetAvailableVersions returns all available versions
func (m *ConfigurationMigrator) GetAvailableVersions() []string {
	versions := make([]string, 0)
	versionSet := make(map[string]bool)

	for _, migrations := range m.migrations {
		for _, migration := range migrations {
			if !versionSet[migration.From] {
				versions = append(versions, migration.From)
				versionSet[migration.From] = true
			}
			if !versionSet[migration.To] {
				versions = append(versions, migration.To)
				versionSet[migration.To] = true
			}
		}
	}

	return versions
}

// initializeMigrations initializes default migrations
func (m *ConfigurationMigrator) initializeMigrations() {
	// Example migration from 1.0.0 to 1.1.0
	m.migrations["1.0.0"] = append(m.migrations["1.0.0"], Migration{
		From:      "1.0.0",
		To:        "1.1.0",
		Name:      "add_workspace_auto_save",
		Desc:      "Add workspace auto_save setting",
		Timestamp: time.Now(),
		Up: func(config *HelixConfig) error {
			if config.Application.Workspace.AutoSave == false {
				// Default to true for new installations
				config.Application.Workspace.AutoSave = true
			}
			return nil
		},
		Down: func(config *HelixConfig) error {
			// Remove auto_save setting
			return nil
		},
		DryRun: false,
		Backup: true,
	})

	// Example migration from 1.1.0 to 1.2.0
	m.migrations["1.1.0"] = append(m.migrations["1.1.0"], Migration{
		From:      "1.1.0",
		To:        "1.2.0",
		Name:      "add_llm_reasoning",
		Desc:      "Add LLM reasoning support",
		Timestamp: time.Now(),
		Up: func(config *HelixConfig) error {
			// Enable reasoning by default
			config.LLM.Features.ReasoningEnabled = true
			return nil
		},
		Down: func(config *HelixConfig) error {
			// Disable reasoning
			return nil
		},
		DryRun: false,
		Backup: true,
	})
}

// findMigrationPath finds migration path between versions
func (m *ConfigurationMigrator) findMigrationPath(from, to string) []string {
	// Simplified implementation
	// In real implementation, use graph algorithms to find shortest path

	if from == to {
		return []string{}
	}

	path := make([]string, 0)
	current := from

	for current != to {
		migrations, exists := m.migrations[current]
		if !exists || len(migrations) == 0 {
			return nil // No path found
		}

		// Take first available migration
		nextMigration := migrations[0]
		path = append(path, m.getMigrationID(&nextMigration))
		current = nextMigration.To

		// Prevent infinite loops
		if len(path) > 100 {
			return nil
		}
	}

	return path
}

// findMigration finds migration by ID
func (m *ConfigurationMigrator) findMigration(migrationID string) *Migration {
	for _, migrations := range m.migrations {
		for _, migration := range migrations {
			if m.getMigrationID(&migration) == migrationID {
				return &migration
			}
		}
	}
	return nil
}

// getMigrationID generates migration ID
func (m *ConfigurationMigrator) getMigrationID(migration *Migration) string {
	return fmt.Sprintf("%s_to_%s", migration.From, migration.To)
}

// createBackup creates configuration backup
func (m *ConfigurationMigrator) createBackup(config *HelixConfig) error {
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("helix_config_backup_%s.json", time.Now().Format("20060102_150405")))

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

// NewConfigurationTransformer creates a new configuration transformer
func NewConfigurationTransformer() *ConfigurationTransformer {
	return &ConfigurationTransformer{
		mappings: make(map[string]TransformMapping),
		rules:    make([]TransformRule, 0),
	}
}

// Transform transforms configuration according to mappings and rules
func (t *ConfigurationTransformer) Transform(config *HelixConfig, variables map[string]interface{}) (*HelixConfig, error) {
	// Create a copy to avoid modifying original
	result := &HelixConfig{}
	if err := t.deepCopy(config, result); err != nil {
		return nil, err
	}

	// Apply transformations in priority order
	if err := t.applyMappings(result, variables); err != nil {
		return nil, err
	}

	// Apply transformation rules
	if err := t.applyRules(result, variables); err != nil {
		return nil, err
	}

	return result, nil
}

// AddMapping adds a transformation mapping
func (t *ConfigurationTransformer) AddMapping(mapping TransformMapping) {
	t.mappings[mapping.Source] = mapping
}

// AddRule adds a transformation rule
func (t *ConfigurationTransformer) AddRule(rule TransformRule) {
	t.rules = append(t.rules, rule)
}

// applyMappings applies transformation mappings
func (t *ConfigurationTransformer) applyMappings(config *HelixConfig, variables map[string]interface{}) error {
	for _, mapping := range t.mappings {
		if err := t.applyMapping(config, mapping, variables); err != nil {
			return err
		}
	}
	return nil
}

// applyMapping applies a single transformation mapping
func (t *ConfigurationTransformer) applyMapping(config *HelixConfig, mapping TransformMapping, variables map[string]interface{}) error {
	// Check condition
	if mapping.Condition != "" {
		if !t.evaluateCondition(mapping.Condition, config, variables) {
			return nil // Skip this mapping
		}
	}

	// Get source value
	sourceValue := t.getValueAtPath(config, mapping.Source)
	if sourceValue == nil && mapping.Required {
		return fmt.Errorf("required source field not found: %s", mapping.Source)
	}

	// Apply transformation
	transformedValue, err := t.applyTransformation(sourceValue, mapping, variables)
	if err != nil {
		return err
	}

	// Set target value
	return t.setValueAtPath(config, mapping.Target, transformedValue)
}

// applyTransformation applies transformation to a value
func (t *ConfigurationTransformer) applyTransformation(value interface{}, mapping TransformMapping, variables map[string]interface{}) (interface{}, error) {
	switch mapping.Transform {
	case "rename":
		// Just move the value
		return value, nil
	case "copy":
		// Return a copy
		return t.deepCopyValue(value), nil
	case "convert":
		// Convert value type
		return t.convertValue(value, mapping.Parameters)
	case "template":
		// Apply template
		return t.applyTemplate(value, mapping.Parameters, variables)
	case "calculate":
		// Calculate value
		return t.calculateValue(value, mapping.Parameters, variables)
	default:
		return value, nil
	}
}

// applyRules applies transformation rules
func (t *ConfigurationTransformer) applyRules(config *HelixConfig, variables map[string]interface{}) error {
	for _, rule := range t.rules {
		if err := t.applyRule(config, rule, variables); err != nil {
			return err
		}
	}
	return nil
}

// applyRule applies a transformation rule
func (t *ConfigurationTransformer) applyRule(config *HelixConfig, rule TransformRule, variables map[string]interface{}) error {
	// Find matching fields
	matches := t.findPatternMatches(config, rule.Pattern)

	// Apply transformation to matches
	for _, match := range matches {
		_ = rule.Transform(match)
		// Update config with transformed value
		// Implementation depends on specific use case
	}

	return nil
}

// Helper methods for transformation

func (t *ConfigurationTransformer) evaluateCondition(condition string, config *HelixConfig, variables map[string]interface{}) bool {
	// Simplified condition evaluation
	// In real implementation, use proper expression parser

	if condition == "always" {
		return true
	}

	if condition == "development" {
		return config.Application.Environment == "development"
	}

	return false
}
