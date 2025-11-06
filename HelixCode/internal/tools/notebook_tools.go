package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// NotebookReadTool reads Jupyter notebook files
type NotebookReadTool struct {
	registry *ToolRegistry
}

func (t *NotebookReadTool) Name() string { return "notebook_read" }

func (t *NotebookReadTool) Description() string {
	return "Read and parse a Jupyter notebook (.ipynb) file"
}

func (t *NotebookReadTool) Category() ToolCategory {
	return CategoryNotebook
}

func (t *NotebookReadTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the notebook file",
			},
			"include_outputs": map[string]interface{}{
				"type":        "boolean",
				"description": "Include cell outputs (default: true)",
			},
		},
		Required:    []string{"path"},
		Description: "Read and parse a Jupyter notebook (.ipynb) file",
	}
}

func (t *NotebookReadTool) Validate(params map[string]interface{}) error {
	if _, ok := params["path"]; !ok {
		return fmt.Errorf("path is required")
	}
	return nil
}

func (t *NotebookReadTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path := params["path"].(string)
	includeOutputs := true

	if val, ok := params["include_outputs"].(bool); ok {
		includeOutputs = val
	}

	// Read file
	content, err := t.registry.filesystem.Reader().Read(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read notebook: %w", err)
	}

	// Parse notebook
	var notebook Notebook
	if err := json.Unmarshal(content.Content, &notebook); err != nil {
		return nil, fmt.Errorf("failed to parse notebook: %w", err)
	}

	// Filter outputs if requested
	if !includeOutputs {
		for i := range notebook.Cells {
			notebook.Cells[i].Outputs = nil
		}
	}

	return &notebook, nil
}

// NotebookEditTool edits Jupyter notebook cells
type NotebookEditTool struct {
	registry *ToolRegistry
}

func (t *NotebookEditTool) Name() string { return "notebook_edit" }

func (t *NotebookEditTool) Description() string {
	return "Edit a cell in a Jupyter notebook"
}

func (t *NotebookEditTool) Category() ToolCategory {
	return CategoryNotebook
}

func (t *NotebookEditTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the notebook file",
			},
			"cell_index": map[string]interface{}{
				"type":        "integer",
				"description": "Index of the cell to edit (0-based)",
			},
			"cell_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the cell to edit (alternative to cell_index)",
			},
			"source": map[string]interface{}{
				"type":        "string",
				"description": "New source code/markdown for the cell",
			},
			"cell_type": map[string]interface{}{
				"type":        "string",
				"description": "Cell type: code or markdown",
			},
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "Operation: replace, insert, delete (default: replace)",
			},
		},
		Required:    []string{"path", "source"},
		Description: "Edit a cell in a Jupyter notebook",
	}
}

func (t *NotebookEditTool) Validate(params map[string]interface{}) error {
	if _, ok := params["path"]; !ok {
		return fmt.Errorf("path is required")
	}
	if _, ok := params["source"]; !ok {
		if op, ok := params["operation"].(string); !ok || op != "delete" {
			return fmt.Errorf("source is required for non-delete operations")
		}
	}
	return nil
}

func (t *NotebookEditTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path := params["path"].(string)
	operation := "replace"

	if op, ok := params["operation"].(string); ok {
		operation = op
	}

	// Read current notebook
	content, err := t.registry.filesystem.Reader().Read(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read notebook: %w", err)
	}

	var notebook Notebook
	if err := json.Unmarshal(content.Content, &notebook); err != nil {
		return nil, fmt.Errorf("failed to parse notebook: %w", err)
	}

	// Perform operation
	switch operation {
	case "replace":
		if err := t.replaceCell(&notebook, params); err != nil {
			return nil, err
		}
	case "insert":
		if err := t.insertCell(&notebook, params); err != nil {
			return nil, err
		}
	case "delete":
		if err := t.deleteCell(&notebook, params); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid operation: %s", operation)
	}

	// Write back to file
	newContent, err := json.MarshalIndent(&notebook, "", " ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notebook: %w", err)
	}

	if err := t.registry.filesystem.Writer().Write(ctx, path, newContent); err != nil {
		return nil, fmt.Errorf("failed to write notebook: %w", err)
	}

	return &notebook, nil
}

func (t *NotebookEditTool) replaceCell(notebook *Notebook, params map[string]interface{}) error {
	cellIndex := t.findCellIndex(notebook, params)
	if cellIndex < 0 {
		return fmt.Errorf("cell not found")
	}

	source := params["source"].(string)
	notebook.Cells[cellIndex].Source = splitSource(source)

	if cellType, ok := params["cell_type"].(string); ok {
		notebook.Cells[cellIndex].CellType = cellType
	}

	return nil
}

func (t *NotebookEditTool) insertCell(notebook *Notebook, params map[string]interface{}) error {
	cellIndex := 0
	if idx, ok := params["cell_index"].(int); ok {
		cellIndex = idx
	}

	if cellIndex < 0 || cellIndex > len(notebook.Cells) {
		return fmt.Errorf("invalid cell index: %d", cellIndex)
	}

	source := params["source"].(string)
	cellType := "code"
	if ct, ok := params["cell_type"].(string); ok {
		cellType = ct
	}

	newCell := NotebookCell{
		CellType: cellType,
		Source:   splitSource(source),
		Metadata: make(map[string]interface{}),
	}

	// Insert cell at index
	notebook.Cells = append(notebook.Cells[:cellIndex], append([]NotebookCell{newCell}, notebook.Cells[cellIndex:]...)...)

	return nil
}

func (t *NotebookEditTool) deleteCell(notebook *Notebook, params map[string]interface{}) error {
	cellIndex := t.findCellIndex(notebook, params)
	if cellIndex < 0 {
		return fmt.Errorf("cell not found")
	}

	// Delete cell
	notebook.Cells = append(notebook.Cells[:cellIndex], notebook.Cells[cellIndex+1:]...)

	return nil
}

func (t *NotebookEditTool) findCellIndex(notebook *Notebook, params map[string]interface{}) int {
	// Try cell index first
	if idx, ok := params["cell_index"].(int); ok {
		if idx >= 0 && idx < len(notebook.Cells) {
			return idx
		}
	}

	// Try cell ID
	if cellID, ok := params["cell_id"].(string); ok {
		for i, cell := range notebook.Cells {
			if cell.ID == cellID {
				return i
			}
		}
	}

	return -1
}

// Notebook represents a Jupyter notebook structure
type Notebook struct {
	Cells        []NotebookCell         `json:"cells"`
	Metadata     map[string]interface{} `json:"metadata"`
	NBFormat     int                    `json:"nbformat"`
	NBFormatMinor int                   `json:"nbformat_minor"`
}

// NotebookCell represents a cell in a Jupyter notebook
type NotebookCell struct {
	ID           string                   `json:"id,omitempty"`
	CellType     string                   `json:"cell_type"`
	Source       []string                 `json:"source"`
	Metadata     map[string]interface{}   `json:"metadata"`
	Outputs      []map[string]interface{} `json:"outputs,omitempty"`
	ExecutionCount interface{}            `json:"execution_count,omitempty"`
}

// splitSource splits source code into lines as Jupyter expects
func splitSource(source string) []string {
	if source == "" {
		return []string{}
	}

	lines := []string{}
	current := ""
	for _, char := range source {
		current += string(char)
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		}
	}
	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// ReadNotebook reads a notebook file
func ReadNotebook(path string) (*Notebook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var notebook Notebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		return nil, err
	}

	return &notebook, nil
}

// WriteNotebook writes a notebook to file
func WriteNotebook(path string, notebook *Notebook) error {
	data, err := json.MarshalIndent(notebook, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
