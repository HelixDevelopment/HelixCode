package template

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Type represents the template type
type Type string

const (
	TypeCode          Type = "code"          // Code generation template
	TypePrompt        Type = "prompt"        // LLM prompt template
	TypeWorkflow      Type = "workflow"      // Workflow definition template
	TypeDocumentation Type = "documentation" // Documentation template
	TypeEmail         Type = "email"         // Email template
	TypeCustom        Type = "custom"        // Custom template
)

// String returns string representation
func (t Type) String() string {
	return string(t)
}

// IsValid checks if type is valid
func (t Type) IsValid() bool {
	switch t {
	case TypeCode, TypePrompt, TypeWorkflow, TypeDocumentation, TypeEmail, TypeCustom:
		return true
	}
	return false
}

// Template represents a reusable template
type Template struct {
	ID          string            // Unique identifier
	Name        string            // Template name
	Description string            // Template description
	Type        Type              // Template type
	Content     string            // Template content with placeholders
	Variables   []Variable        // Expected variables
	Metadata    map[string]string // Additional metadata
	Tags        []string          // Tags for categorization
	Author      string            // Template author
	Version     string            // Template version
	CreatedAt   time.Time         // Creation time
	UpdatedAt   time.Time         // Last update time
}

// Variable represents a template variable
type Variable struct {
	Name         string // Variable name
	Description  string // Variable description
	Required     bool   // Whether variable is required
	DefaultValue string // Default value if not provided
	Type         string // Expected type (string, int, bool, etc.)
}

// NewTemplate creates a new template
func NewTemplate(name, description string, templateType Type) *Template {
	return &Template{
		ID:          generateTemplateID(),
		Name:        name,
		Description: description,
		Type:        templateType,
		Variables:   make([]Variable, 0),
		Metadata:    make(map[string]string),
		Tags:        make([]string, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     "1.0.0",
	}
}

// AddVariable adds a variable to the template
func (t *Template) AddVariable(variable Variable) {
	t.Variables = append(t.Variables, variable)
	t.UpdatedAt = time.Now()
}

// SetContent sets the template content
func (t *Template) SetContent(content string) {
	t.Content = content
	t.UpdatedAt = time.Now()
}

// Render renders the template with provided variables
func (t *Template) Render(vars map[string]interface{}) (string, error) {
	// Validate required variables
	if err := t.ValidateVariables(vars); err != nil {
		return "", err
	}

	// Apply defaults for missing optional variables
	mergedVars := t.applyDefaults(vars)

	// Replace placeholders
	result := t.Content
	for key, value := range mergedVars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}

	// Check for unreplaced placeholders
	if hasUnreplacedPlaceholders(result) {
		return "", fmt.Errorf("template has unreplaced placeholders")
	}

	return result, nil
}

// ValidateVariables validates that all required variables are provided
func (t *Template) ValidateVariables(vars map[string]interface{}) error {
	for _, variable := range t.Variables {
		if variable.Required {
			if _, exists := vars[variable.Name]; !exists {
				return fmt.Errorf("required variable '%s' is missing", variable.Name)
			}
		}
	}
	return nil
}

// applyDefaults applies default values for missing optional variables
func (t *Template) applyDefaults(vars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy provided variables
	for key, value := range vars {
		result[key] = value
	}

	// Apply defaults for missing variables
	for _, variable := range t.Variables {
		if _, exists := result[variable.Name]; !exists && variable.DefaultValue != "" {
			result[variable.Name] = variable.DefaultValue
		}
	}

	return result
}

// ExtractVariables extracts variable names from template content
func (t *Template) ExtractVariables() []string {
	re := regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)
	matches := re.FindAllStringSubmatch(t.Content, -1)

	seen := make(map[string]bool)
	variables := make([]string, 0)

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !seen[varName] {
				seen[varName] = true
				variables = append(variables, varName)
			}
		}
	}

	return variables
}

// Validate validates the template
func (t *Template) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	if t.Content == "" {
		return fmt.Errorf("template content cannot be empty")
	}

	if !t.Type.IsValid() {
		return fmt.Errorf("invalid template type: %s", t.Type)
	}

	// Check that all declared variables exist in content
	extractedVars := t.ExtractVariables()
	extractedSet := make(map[string]bool)
	for _, v := range extractedVars {
		extractedSet[v] = true
	}

	for _, variable := range t.Variables {
		if !extractedSet[variable.Name] {
			return fmt.Errorf("declared variable '%s' not found in template content", variable.Name)
		}
	}

	return nil
}

// Clone creates a copy of the template
func (t *Template) Clone() *Template {
	clone := &Template{
		ID:          generateTemplateID(),
		Name:        t.Name + " (Copy)",
		Description: t.Description,
		Type:        t.Type,
		Content:     t.Content,
		Variables:   make([]Variable, len(t.Variables)),
		Metadata:    make(map[string]string),
		Tags:        make([]string, len(t.Tags)),
		Author:      t.Author,
		Version:     t.Version,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Deep copy variables
	copy(clone.Variables, t.Variables)

	// Deep copy metadata
	for k, v := range t.Metadata {
		clone.Metadata[k] = v
	}

	// Deep copy tags
	copy(clone.Tags, t.Tags)

	return clone
}

// SetMetadata sets a metadata value
func (t *Template) SetMetadata(key, value string) {
	t.Metadata[key] = value
	t.UpdatedAt = time.Now()
}

// GetMetadata gets a metadata value
func (t *Template) GetMetadata(key string) (string, bool) {
	value, ok := t.Metadata[key]
	return value, ok
}

// AddTag adds a tag
func (t *Template) AddTag(tag string) {
	for _, existing := range t.Tags {
		if existing == tag {
			return
		}
	}
	t.Tags = append(t.Tags, tag)
	t.UpdatedAt = time.Now()
}

// HasTag checks if template has a tag
func (t *Template) HasTag(tag string) bool {
	for _, t := range t.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// String returns a string representation
func (t *Template) String() string {
	return fmt.Sprintf("Template[%s]: %s (%s)", t.ID, t.Name, t.Type)
}

// hasUnreplacedPlaceholders checks if string has unreplaced placeholders
func hasUnreplacedPlaceholders(s string) bool {
	re := regexp.MustCompile(`\{\{[a-zA-Z_][a-zA-Z0-9_]*\}\}`)
	return re.MatchString(s)
}

// generateTemplateID generates a unique template ID
func generateTemplateID() string {
	return fmt.Sprintf("tpl-%d", time.Now().UnixNano())
}

// ParseTemplate parses a template string and creates a Template
func ParseTemplate(name string, content string, templateType Type) (*Template, error) {
	tpl := NewTemplate(name, "", templateType)
	tpl.SetContent(content)

	// Auto-detect variables
	variables := tpl.ExtractVariables()
	for _, varName := range variables {
		tpl.AddVariable(Variable{
			Name:     varName,
			Required: true, // Assume required by default
			Type:     "string",
		})
	}

	return tpl, nil
}

// RenderSimple renders a template string with variables (no Template object needed)
func RenderSimple(content string, vars map[string]interface{}) string {
	result := content
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}
	return result
}

// VariableSet represents a set of variables for rendering
type VariableSet map[string]interface{}

// NewVariableSet creates a new variable set
func NewVariableSet() VariableSet {
	return make(map[string]interface{})
}

// Set sets a variable value
func (vs VariableSet) Set(name string, value interface{}) {
	vs[name] = value
}

// Get gets a variable value
func (vs VariableSet) Get(name string) (interface{}, bool) {
	value, ok := vs[name]
	return value, ok
}

// Merge merges another variable set into this one
func (vs VariableSet) Merge(other VariableSet) {
	for key, value := range other {
		vs[key] = value
	}
}

// TemplateSnapshot represents a template export
type TemplateSnapshot struct {
	Template   *Template `json:"template"`
	ExportedAt time.Time `json:"exported_at"`
}
