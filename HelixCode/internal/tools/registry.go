package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"dev.helix.code/internal/tools/browser"
	"dev.helix.code/internal/tools/confirmation"
	"dev.helix.code/internal/tools/filesystem"
	"dev.helix.code/internal/tools/mapping"
	"dev.helix.code/internal/tools/multiedit"
	"dev.helix.code/internal/tools/shell"
	"dev.helix.code/internal/tools/web"
)

// Tool represents a unified interface for all tools
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns a brief description of what the tool does
	Description() string

	// Execute executes the tool with given parameters
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)

	// Schema returns the JSON schema for the tool's parameters
	Schema() ToolSchema

	// Category returns the tool category
	Category() ToolCategory

	// Validate validates the parameters before execution
	Validate(params map[string]interface{}) error
}

// ToolSchema defines the JSON schema for tool parameters
type ToolSchema struct {
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
	Required    []string               `json:"required"`
	Description string                 `json:"description"`
}

// ToolCategory represents the category of a tool
type ToolCategory string

const (
	CategoryFileSystem   ToolCategory = "filesystem"
	CategoryShell        ToolCategory = "shell"
	CategoryWeb          ToolCategory = "web"
	CategoryBrowser      ToolCategory = "browser"
	CategoryMapping      ToolCategory = "mapping"
	CategoryMultiEdit    ToolCategory = "multiedit"
	CategoryConfirmation ToolCategory = "confirmation"
	CategoryNotebook     ToolCategory = "notebook"
	CategoryInteractive  ToolCategory = "interactive"
)

// ToolRegistry manages all available tools
type ToolRegistry struct {
	tools    map[string]Tool
	aliases  map[string]string // alias -> tool name
	mu       sync.RWMutex

	// Component instances
	filesystem   *filesystem.FileSystemTools
	shell        *shell.ShellExecutor
	web          *web.WebTools
	browser      *browser.BrowserTools
	mapper       mapping.Mapper
	multiEdit    *multiedit.MultiFileEditor
	confirmation *confirmation.ConfirmationCoordinator
}

// RegistryConfig contains configuration for the tool registry
type RegistryConfig struct {
	FileSystemConfig   *filesystem.Config
	ShellConfig        *shell.Config
	WebConfig          *web.Config
	BrowserConfig      *browser.Config
	MappingWorkspace   string
	MultiEditConfig    *multiedit.Config
	ConfirmationConfig *confirmation.Config
}

// DefaultRegistryConfig returns default registry configuration
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		FileSystemConfig:   filesystem.DefaultConfig(),
		ShellConfig:        shell.DefaultConfig(),
		WebConfig:          web.DefaultConfig(),
		BrowserConfig:      browser.DefaultConfig(),
		MappingWorkspace:   "",
		MultiEditConfig:    multiedit.DefaultConfig(),
		ConfirmationConfig: confirmation.DefaultConfig(),
	}
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(config *RegistryConfig) (*ToolRegistry, error) {
	if config == nil {
		config = DefaultRegistryConfig()
	}

	registry := &ToolRegistry{
		tools:   make(map[string]Tool),
		aliases: make(map[string]string),
	}

	// Initialize components
	if err := registry.initializeComponents(config); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Register all tools
	if err := registry.registerAllTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return registry, nil
}

// initializeComponents initializes all tool components
func (r *ToolRegistry) initializeComponents(config *RegistryConfig) error {
	var err error

	// Initialize filesystem
	r.filesystem, err = filesystem.NewFileSystemTools(config.FileSystemConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize filesystem: %w", err)
	}

	// Initialize shell
	r.shell = shell.NewShellExecutor(config.ShellConfig)

	// Initialize web
	r.web, err = web.NewWebTools(config.WebConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize web: %w", err)
	}

	// Initialize browser
	r.browser = browser.NewBrowserTools(config.BrowserConfig)

	// Initialize mapper
	workspace := config.MappingWorkspace
	if workspace == "" {
		workspace = config.FileSystemConfig.WorkspaceRoot
	}
	r.mapper = mapping.NewMapper(workspace)

	// Initialize multi-edit
	r.multiEdit, err = multiedit.NewMultiFileEditor(
		multiedit.WithConfig(config.MultiEditConfig),
		multiedit.WithFileSystem(r.filesystem),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize multi-edit: %w", err)
	}

	// Initialize confirmation
	r.confirmation = confirmation.NewConfirmationCoordinator()

	return nil
}

// registerAllTools registers all available tools
func (r *ToolRegistry) registerAllTools() error {
	// File System Tools
	r.Register(&FSReadTool{registry: r})
	r.Register(&FSWriteTool{registry: r})
	r.Register(&FSEditTool{registry: r})
	r.Register(&GlobTool{registry: r})
	r.Register(&GrepTool{registry: r})

	// Shell Tools
	r.Register(&ShellTool{registry: r})
	r.Register(&ShellBackgroundTool{registry: r})
	r.Register(&ShellOutputTool{registry: r})
	r.Register(&ShellKillTool{registry: r})

	// Web Tools
	r.Register(&WebFetchTool{registry: r})
	r.Register(&WebSearchTool{registry: r})

	// Browser Tools
	r.Register(&BrowserLaunchTool{registry: r})
	r.Register(&BrowserNavigateTool{registry: r})
	r.Register(&BrowserScreenshotTool{registry: r})
	r.Register(&BrowserCloseTool{registry: r})

	// Mapping Tools
	r.Register(&CodebaseMapTool{registry: r})
	r.Register(&FileDefinitionsTool{registry: r})

	// Multi-Edit Tools
	r.Register(&MultiEditBeginTool{registry: r})
	r.Register(&MultiEditAddTool{registry: r})
	r.Register(&MultiEditPreviewTool{registry: r})
	r.Register(&MultiEditCommitTool{registry: r})

	// Interactive Tools
	r.Register(&AskUserTool{registry: r})
	r.Register(&TaskTrackerTool{registry: r})

	// Notebook Tools
	r.Register(&NotebookReadTool{registry: r})
	r.Register(&NotebookEditTool{registry: r})

	return nil
}

// Register registers a tool
func (r *ToolRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// RegisterAlias registers an alias for a tool
func (r *ToolRegistry) RegisterAlias(alias, toolName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[toolName]; !exists {
		return fmt.Errorf("tool %s not found", toolName)
	}

	r.aliases[alias] = toolName
	return nil
}

// Get retrieves a tool by name or alias
func (r *ToolRegistry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check aliases first
	if actualName, ok := r.aliases[name]; ok {
		name = actualName
	}

	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tool, nil
}

// Execute executes a tool by name with given parameters
func (r *ToolRegistry) Execute(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// Validate parameters
	if err := tool.Validate(params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Execute tool
	return tool.Execute(ctx, params)
}

// List returns all registered tools
func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// ListByCategory returns all tools in a category
func (r *ToolRegistry) ListByCategory(category ToolCategory) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []Tool
	for _, tool := range r.tools {
		if tool.Category() == category {
			tools = append(tools, tool)
		}
	}

	return tools
}

// GetSchema returns the schema for a tool
func (r *ToolRegistry) GetSchema(name string) (*ToolSchema, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	schema := tool.Schema()
	return &schema, nil
}

// GetAllSchemas returns schemas for all tools
func (r *ToolRegistry) GetAllSchemas() map[string]ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make(map[string]ToolSchema)
	for name, tool := range r.tools {
		schemas[name] = tool.Schema()
	}

	return schemas
}

// ExportSchemas exports all tool schemas as JSON
func (r *ToolRegistry) ExportSchemas() ([]byte, error) {
	schemas := r.GetAllSchemas()
	return json.MarshalIndent(schemas, "", "  ")
}

// Close closes the registry and releases all resources
func (r *ToolRegistry) Close() error {
	if r.web != nil {
		if err := r.web.Close(); err != nil {
			return fmt.Errorf("failed to close web tools: %w", err)
		}
	}

	if r.browser != nil {
		if err := r.browser.CloseAllBrowsers(); err != nil {
			return fmt.Errorf("failed to close browsers: %w", err)
		}
	}

	return nil
}

// GetFileSystem returns the filesystem tools instance
func (r *ToolRegistry) GetFileSystem() *filesystem.FileSystemTools {
	return r.filesystem
}

// GetShell returns the shell executor instance
func (r *ToolRegistry) GetShell() *shell.ShellExecutor {
	return r.shell
}

// GetWeb returns the web tools instance
func (r *ToolRegistry) GetWeb() *web.WebTools {
	return r.web
}

// GetBrowser returns the browser tools instance
func (r *ToolRegistry) GetBrowser() *browser.BrowserTools {
	return r.browser
}

// GetMapper returns the mapper instance
func (r *ToolRegistry) GetMapper() mapping.Mapper {
	return r.mapper
}

// GetMultiEdit returns the multi-edit instance
func (r *ToolRegistry) GetMultiEdit() *multiedit.MultiFileEditor {
	return r.multiEdit
}

// GetConfirmation returns the confirmation instance
func (r *ToolRegistry) GetConfirmation() *confirmation.ConfirmationCoordinator {
	return r.confirmation
}
