package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Manager manages templates
type Manager struct {
	templates map[string]*Template // Templates by ID
	byName    map[string]*Template // Templates by name (for quick lookup)
	byType    map[Type][]*Template // Templates by type
	mu        sync.RWMutex         // Thread-safety
	onCreate  []TemplateCallback   // Callbacks on template creation
	onUpdate  []TemplateCallback   // Callbacks on template update
	onDelete  []TemplateCallback   // Callbacks on template deletion
}

// TemplateCallback is called for template events
type TemplateCallback func(*Template)

// NewManager creates a new template manager
func NewManager() *Manager {
	return &Manager{
		templates: make(map[string]*Template),
		byName:    make(map[string]*Template),
		byType:    make(map[Type][]*Template),
		onCreate:  make([]TemplateCallback, 0),
		onUpdate:  make([]TemplateCallback, 0),
		onDelete:  make([]TemplateCallback, 0),
	}
}

// Register registers a template
func (m *Manager) Register(template *Template) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate template
	if err := template.Validate(); err != nil {
		return err
	}

	// Check for duplicate name
	if _, exists := m.byName[template.Name]; exists {
		return fmt.Errorf("template with name '%s' already exists", template.Name)
	}

	// Register
	m.templates[template.ID] = template
	m.byName[template.Name] = template

	// Add to type index
	if m.byType[template.Type] == nil {
		m.byType[template.Type] = make([]*Template, 0)
	}
	m.byType[template.Type] = append(m.byType[template.Type], template)

	// Trigger callbacks
	for _, callback := range m.onCreate {
		callback(template)
	}

	return nil
}

// Get gets a template by ID
func (m *Manager) Get(id string) (*Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	template, exists := m.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	return template, nil
}

// GetByName gets a template by name
func (m *Manager) GetByName(name string) (*Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	template, exists := m.byName[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	return template, nil
}

// GetByType gets templates by type
func (m *Manager) GetByType(templateType Type) []*Template {
	m.mu.RLock()
	defer m.mu.RUnlock()

	templates := m.byType[templateType]
	if templates == nil {
		return []*Template{}
	}

	// Return copy to prevent external modification
	result := make([]*Template, len(templates))
	copy(result, templates)
	return result
}

// GetAll returns all templates
func (m *Manager) GetAll() []*Template {
	m.mu.RLock()
	defer m.mu.RUnlock()

	templates := make([]*Template, 0, len(m.templates))
	for _, template := range m.templates {
		templates = append(templates, template)
	}

	return templates
}

// Update updates a template
func (m *Manager) Update(id string, updater func(*Template)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	template, exists := m.templates[id]
	if !exists {
		return fmt.Errorf("template not found: %s", id)
	}

	// Apply update
	updater(template)

	// Validate
	if err := template.Validate(); err != nil {
		return err
	}

	// Trigger callbacks
	for _, callback := range m.onUpdate {
		callback(template)
	}

	return nil
}

// Delete deletes a template
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	template, exists := m.templates[id]
	if !exists {
		return fmt.Errorf("template not found: %s", id)
	}

	// Remove from all indexes
	delete(m.templates, id)
	delete(m.byName, template.Name)

	// Remove from type index
	if typeTemplates, ok := m.byType[template.Type]; ok {
		for i, t := range typeTemplates {
			if t.ID == id {
				m.byType[template.Type] = append(typeTemplates[:i], typeTemplates[i+1:]...)
				break
			}
		}
	}

	// Trigger callbacks
	for _, callback := range m.onDelete {
		callback(template)
	}

	return nil
}

// Render renders a template with variables
func (m *Manager) Render(templateID string, vars map[string]interface{}) (string, error) {
	template, err := m.Get(templateID)
	if err != nil {
		return "", err
	}

	return template.Render(vars)
}

// RenderByName renders a template by name with variables
func (m *Manager) RenderByName(name string, vars map[string]interface{}) (string, error) {
	template, err := m.GetByName(name)
	if err != nil {
		return "", err
	}

	return template.Render(vars)
}

// Search searches templates by query (searches name, description, tags)
func (m *Manager) Search(query string) []*Template {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query = strings.ToLower(query)
	results := make([]*Template, 0)

	for _, template := range m.templates {
		// Search in name
		if strings.Contains(strings.ToLower(template.Name), query) {
			results = append(results, template)
			continue
		}

		// Search in description
		if strings.Contains(strings.ToLower(template.Description), query) {
			results = append(results, template)
			continue
		}

		// Search in tags
		for _, tag := range template.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, template)
				break
			}
		}
	}

	return results
}

// GetByTag gets templates with a specific tag
func (m *Manager) GetByTag(tag string) []*Template {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*Template, 0)
	for _, template := range m.templates {
		if template.HasTag(tag) {
			results = append(results, template)
		}
	}

	return results
}

// Count returns the total number of templates
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.templates)
}

// CountByType returns the number of templates per type
func (m *Manager) CountByType() map[Type]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[Type]int)
	for templateType, templates := range m.byType {
		counts[templateType] = len(templates)
	}

	return counts
}

// Clear removes all templates
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.templates = make(map[string]*Template)
	m.byName = make(map[string]*Template)
	m.byType = make(map[Type][]*Template)
}

// LoadFromFile loads a template from a JSON file
func (m *Manager) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return err
	}

	return m.Register(&template)
}

// LoadFromDirectory loads all templates from a directory
func (m *Manager) LoadFromDirectory(dirPath string) (int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, err
	}

	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filename := filepath.Join(dirPath, entry.Name())
		if err := m.LoadFromFile(filename); err != nil {
			// Continue loading other templates even if one fails
			continue
		}

		loaded++
	}

	return loaded, nil
}

// SaveToFile saves a template to a JSON file
func (m *Manager) SaveToFile(templateID, filename string) error {
	template, err := m.Get(templateID)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// Export exports a template
func (m *Manager) Export(templateID string) (*TemplateSnapshot, error) {
	template, err := m.Get(templateID)
	if err != nil {
		return nil, err
	}

	// Create a proper clone for export (preserve original name)
	clone := &Template{
		ID:          template.ID,
		Name:        template.Name, // Preserve original name
		Description: template.Description,
		Type:        template.Type,
		Content:     template.Content,
		Variables:   make([]Variable, len(template.Variables)),
		Metadata:    make(map[string]string),
		Tags:        make([]string, len(template.Tags)),
		Author:      template.Author,
		Version:     template.Version,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}

	// Deep copy variables
	copy(clone.Variables, template.Variables)

	// Deep copy metadata
	for k, v := range template.Metadata {
		clone.Metadata[k] = v
	}

	// Deep copy tags
	copy(clone.Tags, template.Tags)

	return &TemplateSnapshot{
		Template:   clone,
		ExportedAt: template.UpdatedAt,
	}, nil
}

// Import imports a template
func (m *Manager) Import(snapshot *TemplateSnapshot) error {
	return m.Register(snapshot.Template)
}

// RegisterBuiltinTemplates registers built-in templates
func (m *Manager) RegisterBuiltinTemplates() error {
	builtins := getBuiltinTemplates()
	for _, template := range builtins {
		if err := m.Register(template); err != nil {
			return err
		}
	}
	return nil
}

// OnCreate registers a callback for template creation
func (m *Manager) OnCreate(callback TemplateCallback) {
	m.onCreate = append(m.onCreate, callback)
}

// OnUpdate registers a callback for template updates
func (m *Manager) OnUpdate(callback TemplateCallback) {
	m.onUpdate = append(m.onUpdate, callback)
}

// OnDelete registers a callback for template deletion
func (m *Manager) OnDelete(callback TemplateCallback) {
	m.onDelete = append(m.onDelete, callback)
}

// GetStatistics returns manager statistics
func (m *Manager) GetStatistics() *ManagerStatistics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ManagerStatistics{
		TotalTemplates: len(m.templates),
		ByType:         make(map[Type]int),
		TagCloud:       make(map[string]int),
	}

	for templateType, templates := range m.byType {
		stats.ByType[templateType] = len(templates)
	}

	for _, template := range m.templates {
		for _, tag := range template.Tags {
			stats.TagCloud[tag]++
		}
	}

	return stats
}

// ManagerStatistics contains manager statistics
type ManagerStatistics struct {
	TotalTemplates int            // Total number of templates
	ByType         map[Type]int   // Count by type
	TagCloud       map[string]int // Tag usage count
}

// String returns string representation
func (s *ManagerStatistics) String() string {
	return fmt.Sprintf("Templates: %d", s.TotalTemplates)
}

// getBuiltinTemplates returns built-in templates
func getBuiltinTemplates() []*Template {
	templates := make([]*Template, 0)

	// Code template: Function
	funcTpl := NewTemplate("Function", "Generate a function", TypeCode)
	funcTpl.SetContent(`func {{function_name}}({{parameters}}) {{return_type}} {
	{{body}}
}`)
	funcTpl.AddVariable(Variable{Name: "function_name", Required: true, Type: "string"})
	funcTpl.AddVariable(Variable{Name: "parameters", Required: false, DefaultValue: "", Type: "string"})
	funcTpl.AddVariable(Variable{Name: "return_type", Required: false, DefaultValue: "", Type: "string"})
	funcTpl.AddVariable(Variable{Name: "body", Required: true, Type: "string"})
	funcTpl.AddTag("code")
	funcTpl.AddTag("go")
	templates = append(templates, funcTpl)

	// Prompt template: Code Review
	reviewTpl := NewTemplate("Code Review", "Code review prompt", TypePrompt)
	reviewTpl.SetContent(`Review the following {{language}} code and provide feedback on:
- Code quality
- Best practices
- Potential issues
- Suggestions for improvement

Code:
{{code}}

Focus on: {{focus_areas}}`)
	reviewTpl.AddVariable(Variable{Name: "language", Required: true, Type: "string"})
	reviewTpl.AddVariable(Variable{Name: "code", Required: true, Type: "string"})
	reviewTpl.AddVariable(Variable{Name: "focus_areas", Required: false, DefaultValue: "all aspects", Type: "string"})
	reviewTpl.AddTag("prompt")
	reviewTpl.AddTag("review")
	templates = append(templates, reviewTpl)

	// Prompt template: Bug Fix
	bugFixTpl := NewTemplate("Bug Fix", "Bug fix assistance prompt", TypePrompt)
	bugFixTpl.SetContent(`Help me fix the following bug in {{language}}:

Error: {{error_message}}

Code:
{{code}}

Expected behavior: {{expected_behavior}}
Actual behavior: {{actual_behavior}}

Please provide a fix and explanation.`)
	bugFixTpl.AddVariable(Variable{Name: "language", Required: true, Type: "string"})
	bugFixTpl.AddVariable(Variable{Name: "error_message", Required: true, Type: "string"})
	bugFixTpl.AddVariable(Variable{Name: "code", Required: true, Type: "string"})
	bugFixTpl.AddVariable(Variable{Name: "expected_behavior", Required: true, Type: "string"})
	bugFixTpl.AddVariable(Variable{Name: "actual_behavior", Required: true, Type: "string"})
	bugFixTpl.AddTag("prompt")
	bugFixTpl.AddTag("debug")
	templates = append(templates, bugFixTpl)

	// Documentation template: Function Doc
	funcDocTpl := NewTemplate("Function Documentation", "Document a function", TypeDocumentation)
	funcDocTpl.SetContent(`// {{function_name}} {{description}}
//
// Parameters:
{{parameters_doc}}
//
// Returns:
{{returns_doc}}
//
// Example:
//   {{example}}`)
	funcDocTpl.AddVariable(Variable{Name: "function_name", Required: true, Type: "string"})
	funcDocTpl.AddVariable(Variable{Name: "description", Required: true, Type: "string"})
	funcDocTpl.AddVariable(Variable{Name: "parameters_doc", Required: false, DefaultValue: "//   None", Type: "string"})
	funcDocTpl.AddVariable(Variable{Name: "returns_doc", Required: false, DefaultValue: "//   None", Type: "string"})
	funcDocTpl.AddVariable(Variable{Name: "example", Required: false, DefaultValue: "N/A", Type: "string"})
	funcDocTpl.AddTag("documentation")
	funcDocTpl.AddTag("function")
	templates = append(templates, funcDocTpl)

	// Email template: Status Update
	statusTpl := NewTemplate("Status Update Email", "Project status update", TypeEmail)
	statusTpl.SetContent(`Subject: {{project_name}} - Status Update

Hi {{recipient_name}},

Here's the latest update on {{project_name}}:

Progress: {{progress}}%
Status: {{status}}

Completed:
{{completed_items}}

In Progress:
{{in_progress_items}}

Next Steps:
{{next_steps}}

Best regards,
{{sender_name}}`)
	statusTpl.AddVariable(Variable{Name: "project_name", Required: true, Type: "string"})
	statusTpl.AddVariable(Variable{Name: "recipient_name", Required: true, Type: "string"})
	statusTpl.AddVariable(Variable{Name: "progress", Required: true, Type: "int"})
	statusTpl.AddVariable(Variable{Name: "status", Required: true, Type: "string"})
	statusTpl.AddVariable(Variable{Name: "completed_items", Required: true, Type: "string"})
	statusTpl.AddVariable(Variable{Name: "in_progress_items", Required: true, Type: "string"})
	statusTpl.AddVariable(Variable{Name: "next_steps", Required: true, Type: "string"})
	statusTpl.AddVariable(Variable{Name: "sender_name", Required: true, Type: "string"})
	statusTpl.AddTag("email")
	statusTpl.AddTag("status")
	templates = append(templates, statusTpl)

	return templates
}
