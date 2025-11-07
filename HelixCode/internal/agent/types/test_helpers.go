package types

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"dev.helix.code/internal/tools"
)

// MockTool implements the tools.Tool interface for testing
type MockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

func NewMockTool(name string, executeFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)) *MockTool {
	return &MockTool{
		name:        name,
		description: fmt.Sprintf("Mock %s tool", name),
		executeFunc: executeFunc,
	}
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type:        "object",
		Properties:  make(map[string]interface{}),
		Required:    []string{},
		Description: m.description,
	}
}

func (m *MockTool) Category() tools.ToolCategory {
	return tools.CategoryFileSystem
}

func (m *MockTool) Validate(params map[string]interface{}) error {
	return nil
}

// MockToolRegistry implements a simple tool registry for testing
// IMPORTANT: This struct must have the same memory layout as tools.ToolRegistry
// for unsafe.Pointer conversion to work correctly
type MockToolRegistry struct {
	tools   map[string]tools.Tool
	aliases map[string]string // Not used in tests but needed for memory layout compatibility
	mu      sync.RWMutex

	// These fields mirror tools.ToolRegistry to maintain memory layout compatibility
	// They are not used in tests but must be present for unsafe pointer conversion
	_ interface{} // filesystem
	_ interface{} // shell
	_ interface{} // web
	_ interface{} // browser
	_ interface{} // mapper
	_ interface{} // multiEdit
	_ interface{} // confirmation
}

func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		tools:   make(map[string]tools.Tool),
		aliases: make(map[string]string),
	}
}

func (m *MockToolRegistry) Register(tool tools.Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name()] = tool
}

func (m *MockToolRegistry) Get(name string) (tools.Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, exists := m.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	return tool, nil
}

// CreateMockToolRegistry creates a registry with common mock tools
func CreateMockToolRegistry(fsReadFunc, fsWriteFunc, shellFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)) *MockToolRegistry {
	registry := NewMockToolRegistry()

	if fsReadFunc != nil {
		registry.Register(NewMockTool("FSRead", fsReadFunc))
	}

	if fsWriteFunc != nil {
		registry.Register(NewMockTool("FSWrite", fsWriteFunc))
	}

	if shellFunc != nil {
		registry.Register(NewMockTool("Shell", shellFunc))
	}

	return registry
}

// ConvertToToolRegistry converts a MockToolRegistry to *tools.ToolRegistry using unsafe pointer conversion
// This is safe because both types have compatible Get methods with the same signature
func ConvertToToolRegistry(mock *MockToolRegistry) *tools.ToolRegistry {
	return (*tools.ToolRegistry)(unsafe.Pointer(mock))
}
