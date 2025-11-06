package tools

import (
	"context"
	"fmt"

	"dev.helix.code/internal/tools/mapping"
	"dev.helix.code/internal/tools/multiedit"
)

// CodebaseMapTool creates a codebase map
type CodebaseMapTool struct {
	registry *ToolRegistry
}

func (t *CodebaseMapTool) Name() string { return "codebase_map" }

func (t *CodebaseMapTool) Description() string {
	return "Create a map of the codebase structure and definitions"
}

func (t *CodebaseMapTool) Category() ToolCategory {
	return CategoryMapping
}

func (t *CodebaseMapTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"root": map[string]interface{}{
				"type":        "string",
				"description": "Root directory to map (default: workspace root)",
			},
			"languages": map[string]interface{}{
				"type":        "array",
				"description": "Languages to include (default: all supported)",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"use_cache": map[string]interface{}{
				"type":        "boolean",
				"description": "Use cached results if available (default: true)",
			},
		},
		Required:    []string{},
		Description: "Create a map of the codebase structure and definitions",
	}
}

func (t *CodebaseMapTool) Validate(params map[string]interface{}) error {
	return nil
}

func (t *CodebaseMapTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	opts := mapping.DefaultMapOptions()

	root := ""
	if r, ok := params["root"].(string); ok && r != "" {
		root = r
	}

	if languages, ok := params["languages"].([]interface{}); ok {
		opts.Languages = make([]string, len(languages))
		for i, lang := range languages {
			opts.Languages[i] = lang.(string)
		}
	}

	if useCache, ok := params["use_cache"].(bool); ok {
		opts.UseCache = useCache
	}

	// Use mapper's MapCodebase method
	return t.registry.mapper.MapCodebase(ctx, root, opts)
}

// FileDefinitionsTool gets definitions from a specific file
type FileDefinitionsTool struct {
	registry *ToolRegistry
}

func (t *FileDefinitionsTool) Name() string { return "file_definitions" }

func (t *FileDefinitionsTool) Description() string {
	return "Get all definitions (functions, classes, etc.) from a file"
}

func (t *FileDefinitionsTool) Category() ToolCategory {
	return CategoryMapping
}

func (t *FileDefinitionsTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file",
			},
		},
		Required:    []string{"path"},
		Description: "Get all definitions (functions, classes, etc.) from a file",
	}
}

func (t *FileDefinitionsTool) Validate(params map[string]interface{}) error {
	if _, ok := params["path"]; !ok {
		return fmt.Errorf("path is required")
	}
	return nil
}

func (t *FileDefinitionsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path := params["path"].(string)
	return t.registry.mapper.MapFile(ctx, path)
}

// MultiEditBeginTool starts a multi-file edit transaction
type MultiEditBeginTool struct {
	registry *ToolRegistry
}

func (t *MultiEditBeginTool) Name() string { return "multiedit_begin" }

func (t *MultiEditBeginTool) Description() string {
	return "Begin a multi-file edit transaction"
}

func (t *MultiEditBeginTool) Category() ToolCategory {
	return CategoryMultiEdit
}

func (t *MultiEditBeginTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Description of the edit operation",
			},
			"require_preview": map[string]interface{}{
				"type":        "boolean",
				"description": "Require preview before commit (default: true)",
			},
		},
		Required:    []string{"description"},
		Description: "Begin a multi-file edit transaction",
	}
}

func (t *MultiEditBeginTool) Validate(params map[string]interface{}) error {
	if _, ok := params["description"]; !ok {
		return fmt.Errorf("description is required")
	}
	return nil
}

func (t *MultiEditBeginTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	opts := multiedit.EditOptions{
		BackupEnabled: true,
		GitAware:      true,
	}

	// Store description in metadata (will be added to transaction after creation)
	description := params["description"].(string)
	_ = description // Use description for logging if needed

	return t.registry.multiEdit.BeginEdit(ctx, opts)
}

// MultiEditAddTool adds an edit to a transaction
type MultiEditAddTool struct {
	registry *ToolRegistry
}

func (t *MultiEditAddTool) Name() string { return "multiedit_add" }

func (t *MultiEditAddTool) Description() string {
	return "Add a file edit to an open transaction"
}

func (t *MultiEditAddTool) Category() ToolCategory {
	return CategoryMultiEdit
}

func (t *MultiEditAddTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"transaction_id": map[string]interface{}{
				"type":        "string",
				"description": "Transaction ID from multiedit_begin",
			},
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to edit",
			},
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "Operation type: create, update, delete",
			},
			"new_content": map[string]interface{}{
				"type":        "string",
				"description": "New content for the file (for create/update)",
			},
		},
		Required:    []string{"transaction_id", "file_path", "operation"},
		Description: "Add a file edit to an open transaction",
	}
}

func (t *MultiEditAddTool) Validate(params map[string]interface{}) error {
	if _, ok := params["transaction_id"]; !ok {
		return fmt.Errorf("transaction_id is required")
	}
	if _, ok := params["file_path"]; !ok {
		return fmt.Errorf("file_path is required")
	}
	if _, ok := params["operation"]; !ok {
		return fmt.Errorf("operation is required")
	}
	return nil
}

func (t *MultiEditAddTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	txID := params["transaction_id"].(string)
	tx, err := t.registry.multiEdit.GetTransaction(ctx, txID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	edit := &multiedit.FileEdit{
		FilePath: params["file_path"].(string),
	}

	opStr := params["operation"].(string)
	switch opStr {
	case "create":
		edit.Operation = multiedit.OpCreate
	case "update":
		edit.Operation = multiedit.OpUpdate
	case "delete":
		edit.Operation = multiedit.OpDelete
	default:
		return nil, fmt.Errorf("invalid operation: %s", opStr)
	}

	if newContent, ok := params["new_content"].(string); ok {
		edit.NewContent = []byte(newContent)
	}

	return nil, t.registry.multiEdit.AddEdit(ctx, tx, edit)
}

// MultiEditPreviewTool previews changes in a transaction
type MultiEditPreviewTool struct {
	registry *ToolRegistry
}

func (t *MultiEditPreviewTool) Name() string { return "multiedit_preview" }

func (t *MultiEditPreviewTool) Description() string {
	return "Preview changes in a multi-file edit transaction"
}

func (t *MultiEditPreviewTool) Category() ToolCategory {
	return CategoryMultiEdit
}

func (t *MultiEditPreviewTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"transaction_id": map[string]interface{}{
				"type":        "string",
				"description": "Transaction ID from multiedit_begin",
			},
		},
		Required:    []string{"transaction_id"},
		Description: "Preview changes in a multi-file edit transaction",
	}
}

func (t *MultiEditPreviewTool) Validate(params map[string]interface{}) error {
	if _, ok := params["transaction_id"]; !ok {
		return fmt.Errorf("transaction_id is required")
	}
	return nil
}

func (t *MultiEditPreviewTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	txID := params["transaction_id"].(string)
	tx, err := t.registry.multiEdit.GetTransaction(ctx, txID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	return t.registry.multiEdit.Preview(ctx, tx)
}

// MultiEditCommitTool commits a transaction
type MultiEditCommitTool struct {
	registry *ToolRegistry
}

func (t *MultiEditCommitTool) Name() string { return "multiedit_commit" }

func (t *MultiEditCommitTool) Description() string {
	return "Commit a multi-file edit transaction"
}

func (t *MultiEditCommitTool) Category() ToolCategory {
	return CategoryMultiEdit
}

func (t *MultiEditCommitTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"transaction_id": map[string]interface{}{
				"type":        "string",
				"description": "Transaction ID from multiedit_begin",
			},
		},
		Required:    []string{"transaction_id"},
		Description: "Commit a multi-file edit transaction",
	}
}

func (t *MultiEditCommitTool) Validate(params map[string]interface{}) error {
	if _, ok := params["transaction_id"]; !ok {
		return fmt.Errorf("transaction_id is required")
	}
	return nil
}

func (t *MultiEditCommitTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	txID := params["transaction_id"].(string)
	tx, err := t.registry.multiEdit.GetTransaction(ctx, txID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	return nil, t.registry.multiEdit.Commit(ctx, tx)
}
