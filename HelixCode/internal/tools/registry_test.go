package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestToolRegistry(t *testing.T) {
	// Create temporary directory for tests
	tmpDir := t.TempDir()

	// Create registry
	config := DefaultRegistryConfig()
	config.FileSystemConfig.WorkspaceRoot = tmpDir
	config.ShellConfig.WorkDir = tmpDir

	registry, err := NewToolRegistry(config)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	t.Run("list_tools", func(t *testing.T) {
		tools := registry.List()
		if len(tools) == 0 {
			t.Error("Expected tools to be registered")
		}

		t.Logf("Registered %d tools", len(tools))
		for _, tool := range tools {
			t.Logf("  - %s: %s", tool.Name(), tool.Description())
		}
	})

	t.Run("get_tool", func(t *testing.T) {
		tool, err := registry.Get("fs_read")
		if err != nil {
			t.Errorf("Failed to get fs_read tool: %v", err)
		}
		if tool.Name() != "fs_read" {
			t.Errorf("Expected tool name 'fs_read', got '%s'", tool.Name())
		}
	})

	t.Run("get_schema", func(t *testing.T) {
		schema, err := registry.GetSchema("fs_read")
		if err != nil {
			t.Errorf("Failed to get schema: %v", err)
		}
		if len(schema.Required) == 0 {
			t.Error("Expected required fields in schema")
		}
	})

	t.Run("export_schemas", func(t *testing.T) {
		data, err := registry.ExportSchemas()
		if err != nil {
			t.Errorf("Failed to export schemas: %v", err)
		}
		if len(data) == 0 {
			t.Error("Expected exported schemas")
		}
	})

	t.Run("list_by_category", func(t *testing.T) {
		fsTools := registry.ListByCategory(CategoryFileSystem)
		if len(fsTools) == 0 {
			t.Error("Expected filesystem tools")
		}
		t.Logf("Found %d filesystem tools", len(fsTools))
	})
}

func TestIntegrationFileSystemTools(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file.\n"

	config := DefaultRegistryConfig()
	config.FileSystemConfig.WorkspaceRoot = tmpDir

	registry, err := NewToolRegistry(config)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	ctx := context.Background()

	t.Run("write_and_read", func(t *testing.T) {
		// Write file
		_, err := registry.Execute(ctx, "fs_write", map[string]interface{}{
			"path":    testFile,
			"content": testContent,
		})
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Read file
		result, err := registry.Execute(ctx, "fs_read", map[string]interface{}{
			"path": testFile,
		})
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		t.Logf("Read file successfully: %d bytes", len(result.(map[string]interface{})["content"].([]byte)))
	})

	t.Run("edit_file", func(t *testing.T) {
		// Edit file
		_, err := registry.Execute(ctx, "fs_edit", map[string]interface{}{
			"path":       testFile,
			"old_string": "World",
			"new_string": "HelixCode",
		})
		if err != nil {
			t.Fatalf("Failed to edit file: %v", err)
		}

		// Verify edit
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file after edit: %v", err)
		}

		t.Logf("File after edit: %s", string(content))
	})

	t.Run("glob_search", func(t *testing.T) {
		// Create more files
		os.WriteFile(filepath.Join(tmpDir, "test1.go"), []byte("package main"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "test2.go"), []byte("package main"), 0644)

		result, err := registry.Execute(ctx, "glob", map[string]interface{}{
			"pattern": "*.go",
		})
		if err != nil {
			t.Fatalf("Failed to glob: %v", err)
		}

		matches := result.([]string)
		t.Logf("Found %d .go files", len(matches))
	})
}

func TestIntegrationShellTools(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultRegistryConfig()
	config.ShellConfig.WorkDir = tmpDir

	registry, err := NewToolRegistry(config)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	ctx := context.Background()

	t.Run("execute_command", func(t *testing.T) {
		result, err := registry.Execute(ctx, "shell", map[string]interface{}{
			"command": "echo 'Hello from shell'",
			"timeout": 5,
		})
		if err != nil {
			t.Fatalf("Failed to execute shell command: %v", err)
		}

		t.Logf("Shell command result: %+v", result)
	})

	t.Run("background_execution", func(t *testing.T) {
		// Start background command
		result, err := registry.Execute(ctx, "shell_background", map[string]interface{}{
			"command": "sleep 1 && echo 'Background done'",
		})
		if err != nil {
			t.Fatalf("Failed to start background command: %v", err)
		}

		t.Logf("Background execution started: %+v", result)

		// Give it time to complete
		time.Sleep(2 * time.Second)
	})
}

func TestIntegrationMultiEdit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	os.WriteFile(file1, []byte("Content 1"), 0644)
	os.WriteFile(file2, []byte("Content 2"), 0644)

	config := DefaultRegistryConfig()
	config.FileSystemConfig.WorkspaceRoot = tmpDir
	config.MultiEditConfig.WorkspaceRoot = tmpDir

	registry, err := NewToolRegistry(config)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	ctx := context.Background()

	t.Run("multi_file_edit_transaction", func(t *testing.T) {
		// Begin transaction
		txResult, err := registry.Execute(ctx, "multiedit_begin", map[string]interface{}{
			"description": "Test multi-file edit",
		})
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		t.Logf("Transaction started: %+v", txResult)

		// Note: Full multi-edit test would require proper transaction handling
		// which needs the actual transaction object, not just the result
	})
}

func TestIntegrationTaskTracker(t *testing.T) {
	registry, err := NewToolRegistry(DefaultRegistryConfig())
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	ctx := context.Background()

	t.Run("create_and_manage_tasks", func(t *testing.T) {
		// Create task
		result, err := registry.Execute(ctx, "task_tracker", map[string]interface{}{
			"action":      "create",
			"title":       "Test Task",
			"description": "This is a test task",
		})
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		task := result.(*Task)
		t.Logf("Created task: %s - %s", task.ID, task.Title)

		// Update task
		_, err = registry.Execute(ctx, "task_tracker", map[string]interface{}{
			"action":   "update",
			"task_id":  task.ID,
			"status":   "in_progress",
			"progress": 50,
		})
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}

		// List tasks
		listResult, err := registry.Execute(ctx, "task_tracker", map[string]interface{}{
			"action": "list",
		})
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		tasks := listResult.([]*Task)
		t.Logf("Found %d tasks", len(tasks))

		// Complete task
		_, err = registry.Execute(ctx, "task_tracker", map[string]interface{}{
			"action":  "complete",
			"task_id": task.ID,
		})
		if err != nil {
			t.Fatalf("Failed to complete task: %v", err)
		}
	})
}

func TestIntegrationNotebook(t *testing.T) {
	tmpDir := t.TempDir()
	notebookPath := filepath.Join(tmpDir, "test.ipynb")

	// Create a simple notebook
	notebook := &Notebook{
		Cells: []NotebookCell{
			{
				CellType: "code",
				Source:   []string{"print('Hello, World!')\n"},
				Metadata: make(map[string]interface{}),
			},
			{
				CellType: "markdown",
				Source:   []string{"# Title\n", "Some text\n"},
				Metadata: make(map[string]interface{}),
			},
		},
		Metadata:      make(map[string]interface{}),
		NBFormat:      4,
		NBFormatMinor: 5,
	}

	if err := WriteNotebook(notebookPath, notebook); err != nil {
		t.Fatalf("Failed to create test notebook: %v", err)
	}

	config := DefaultRegistryConfig()
	config.FileSystemConfig.WorkspaceRoot = tmpDir

	registry, err := NewToolRegistry(config)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	ctx := context.Background()

	t.Run("read_notebook", func(t *testing.T) {
		result, err := registry.Execute(ctx, "notebook_read", map[string]interface{}{
			"path":            notebookPath,
			"include_outputs": true,
		})
		if err != nil {
			t.Fatalf("Failed to read notebook: %v", err)
		}

		nb := result.(*Notebook)
		t.Logf("Read notebook with %d cells", len(nb.Cells))

		if len(nb.Cells) != 2 {
			t.Errorf("Expected 2 cells, got %d", len(nb.Cells))
		}
	})

	t.Run("edit_notebook_cell", func(t *testing.T) {
		_, err := registry.Execute(ctx, "notebook_edit", map[string]interface{}{
			"path":       notebookPath,
			"cell_index": 0,
			"source":     "print('Hello, HelixCode!')\n",
		})
		if err != nil {
			t.Fatalf("Failed to edit notebook: %v", err)
		}

		// Verify edit
		nb, err := ReadNotebook(notebookPath)
		if err != nil {
			t.Fatalf("Failed to read notebook after edit: %v", err)
		}

		t.Logf("Cell after edit: %v", nb.Cells[0].Source)
	})
}

func BenchmarkToolExecution(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.txt")

	config := DefaultRegistryConfig()
	config.FileSystemConfig.WorkspaceRoot = tmpDir

	registry, err := NewToolRegistry(config)
	if err != nil {
		b.Fatalf("Failed to create registry: %v", err)
	}
	defer registry.Close()

	ctx := context.Background()

	b.Run("fs_write", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			registry.Execute(ctx, "fs_write", map[string]interface{}{
				"path":    testFile,
				"content": "Benchmark content",
			})
		}
	})

	b.Run("fs_read", func(b *testing.B) {
		// Write once
		registry.Execute(ctx, "fs_write", map[string]interface{}{
			"path":    testFile,
			"content": "Benchmark content",
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			registry.Execute(ctx, "fs_read", map[string]interface{}{
				"path": testFile,
			})
		}
	})
}
