package template

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplate(t *testing.T) {
	t.Run("create_template", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test template", TypeCode)
		assert.NotEmpty(t, tpl.ID)
		assert.Equal(t, "Test", tpl.Name)
		assert.Equal(t, "Test template", tpl.Description)
		assert.Equal(t, TypeCode, tpl.Type)
		assert.NotZero(t, tpl.CreatedAt)
	})

	t.Run("add_variable", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.AddVariable(Variable{
			Name:     "name",
			Required: true,
			Type:     "string",
		})

		assert.Len(t, tpl.Variables, 1)
		assert.Equal(t, "name", tpl.Variables[0].Name)
		assert.True(t, tpl.Variables[0].Required)
	})

	t.Run("set_content", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		content := "Hello {{name}}"
		tpl.SetContent(content)

		assert.Equal(t, content, tpl.Content)
	})

	t.Run("extract_variables", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}, you are {{age}} years old")

		vars := tpl.ExtractVariables()
		assert.Len(t, vars, 2)
		assert.Contains(t, vars, "name")
		assert.Contains(t, vars, "age")
	})

	t.Run("extract_variables_duplicates", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("{{name}} {{name}} {{name}}")

		vars := tpl.ExtractVariables()
		assert.Len(t, vars, 1)
		assert.Equal(t, "name", vars[0])
	})

	t.Run("validate_template", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})

		err := tpl.Validate()
		assert.NoError(t, err)
	})

	t.Run("validate_empty_name", func(t *testing.T) {
		tpl := NewTemplate("", "Test", TypeCode)
		tpl.SetContent("Hello")

		err := tpl.Validate()
		assert.Error(t, err)
	})

	t.Run("validate_empty_content", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)

		err := tpl.Validate()
		assert.Error(t, err)
	})

	t.Run("validate_undeclared_variable", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "other", Required: true})

		err := tpl.Validate()
		assert.Error(t, err)
	})

	t.Run("clone_template", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})
		tpl.AddTag("test")

		clone := tpl.Clone()
		assert.NotEqual(t, tpl.ID, clone.ID)
		assert.Contains(t, clone.Name, "Copy")
		assert.Equal(t, tpl.Content, clone.Content)
		assert.Len(t, clone.Variables, 1)
		assert.Len(t, clone.Tags, 1)
	})

	t.Run("metadata", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetMetadata("author", "John")

		value, ok := tpl.GetMetadata("author")
		assert.True(t, ok)
		assert.Equal(t, "John", value)

		_, ok = tpl.GetMetadata("nonexistent")
		assert.False(t, ok)
	})

	t.Run("tags", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.AddTag("go")
		tpl.AddTag("code")
		tpl.AddTag("go") // Duplicate

		assert.Len(t, tpl.Tags, 2)
		assert.True(t, tpl.HasTag("go"))
		assert.True(t, tpl.HasTag("code"))
		assert.False(t, tpl.HasTag("python"))
	})
}

func TestRender(t *testing.T) {
	t.Run("render_simple", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}, you are {{age}} years old")
		tpl.AddVariable(Variable{Name: "name", Required: true})
		tpl.AddVariable(Variable{Name: "age", Required: true})

		result, err := tpl.Render(map[string]interface{}{
			"name": "Alice",
			"age":  30,
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello Alice, you are 30 years old", result)
	})

	t.Run("render_missing_required", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})

		_, err := tpl.Render(map[string]interface{}{})
		assert.Error(t, err)
	})

	t.Run("render_with_defaults", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}, status: {{status}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})
		tpl.AddVariable(Variable{Name: "status", Required: false, DefaultValue: "active"})

		result, err := tpl.Render(map[string]interface{}{
			"name": "Alice",
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello Alice, status: active", result)
	})

	t.Run("render_unreplaced_placeholder", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}, {{missing}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})

		_, err := tpl.Render(map[string]interface{}{
			"name": "Alice",
		})

		assert.Error(t, err)
	})

	t.Run("render_simple_function", func(t *testing.T) {
		content := "Hello {{name}}"
		result := RenderSimple(content, map[string]interface{}{
			"name": "Alice",
		})

		assert.Equal(t, "Hello Alice", result)
	})
}

func TestParseTemplate(t *testing.T) {
	t.Run("parse_template", func(t *testing.T) {
		content := "Hello {{name}}, you are {{age}}"
		tpl, err := ParseTemplate("Greeting", content, TypePrompt)

		require.NoError(t, err)
		assert.Equal(t, "Greeting", tpl.Name)
		assert.Equal(t, TypePrompt, tpl.Type)
		assert.Len(t, tpl.Variables, 2)
	})
}

func TestVariableSet(t *testing.T) {
	t.Run("create_variable_set", func(t *testing.T) {
		vs := NewVariableSet()
		assert.NotNil(t, vs)
	})

	t.Run("set_and_get", func(t *testing.T) {
		vs := NewVariableSet()
		vs.Set("name", "Alice")

		value, ok := vs.Get("name")
		assert.True(t, ok)
		assert.Equal(t, "Alice", value)
	})

	t.Run("merge_variable_sets", func(t *testing.T) {
		vs1 := NewVariableSet()
		vs1.Set("name", "Alice")

		vs2 := NewVariableSet()
		vs2.Set("age", 30)

		vs1.Merge(vs2)

		name, ok := vs1.Get("name")
		assert.True(t, ok)
		assert.Equal(t, "Alice", name)

		age, ok := vs1.Get("age")
		assert.True(t, ok)
		assert.Equal(t, 30, age)
	})
}

func TestType(t *testing.T) {
	t.Run("type_is_valid", func(t *testing.T) {
		assert.True(t, TypeCode.IsValid())
		assert.True(t, TypePrompt.IsValid())
		assert.True(t, TypeWorkflow.IsValid())
		assert.False(t, Type("invalid").IsValid())
	})

	t.Run("type_string", func(t *testing.T) {
		assert.Equal(t, "code", TypeCode.String())
		assert.Equal(t, "prompt", TypePrompt.String())
	})
}

func TestManager(t *testing.T) {
	t.Run("create_manager", func(t *testing.T) {
		mgr := NewManager()
		assert.NotNil(t, mgr)
		assert.Equal(t, 0, mgr.Count())
	})

	t.Run("register_template", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})

		err := mgr.Register(tpl)
		require.NoError(t, err)
		assert.Equal(t, 1, mgr.Count())
	})

	t.Run("register_duplicate_name", func(t *testing.T) {
		mgr := NewManager()

		tpl1 := NewTemplate("Test", "Test", TypeCode)
		tpl1.SetContent("Content 1")
		mgr.Register(tpl1)

		tpl2 := NewTemplate("Test", "Test", TypeCode)
		tpl2.SetContent("Content 2")
		err := mgr.Register(tpl2)

		assert.Error(t, err)
	})

	t.Run("get_template", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")
		mgr.Register(tpl)

		retrieved, err := mgr.Get(tpl.ID)
		require.NoError(t, err)
		assert.Equal(t, tpl.ID, retrieved.ID)
	})

	t.Run("get_by_name", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")
		mgr.Register(tpl)

		retrieved, err := mgr.GetByName("Test")
		require.NoError(t, err)
		assert.Equal(t, "Test", retrieved.Name)
	})

	t.Run("get_by_type", func(t *testing.T) {
		mgr := NewManager()

		tpl1 := NewTemplate("Code1", "Test", TypeCode)
		tpl1.SetContent("Code")
		mgr.Register(tpl1)

		tpl2 := NewTemplate("Code2", "Test", TypeCode)
		tpl2.SetContent("Code")
		mgr.Register(tpl2)

		tpl3 := NewTemplate("Prompt1", "Test", TypePrompt)
		tpl3.SetContent("Prompt")
		mgr.Register(tpl3)

		codeTemplates := mgr.GetByType(TypeCode)
		assert.Len(t, codeTemplates, 2)

		promptTemplates := mgr.GetByType(TypePrompt)
		assert.Len(t, promptTemplates, 1)
	})

	t.Run("delete_template", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")
		mgr.Register(tpl)

		err := mgr.Delete(tpl.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, mgr.Count())

		_, err = mgr.Get(tpl.ID)
		assert.Error(t, err)
	})

	t.Run("update_template", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")
		mgr.Register(tpl)

		err := mgr.Update(tpl.ID, func(t *Template) {
			t.Description = "Updated"
		})

		require.NoError(t, err)

		retrieved, _ := mgr.Get(tpl.ID)
		assert.Equal(t, "Updated", retrieved.Description)
	})

	t.Run("render_by_id", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})
		mgr.Register(tpl)

		result, err := mgr.Render(tpl.ID, map[string]interface{}{
			"name": "Alice",
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello Alice", result)
	})

	t.Run("render_by_name", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Greeting", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})
		mgr.Register(tpl)

		result, err := mgr.RenderByName("Greeting", map[string]interface{}{
			"name": "Alice",
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello Alice", result)
	})

	t.Run("search_templates", func(t *testing.T) {
		mgr := NewManager()

		tpl1 := NewTemplate("User Greeting", "Greet user", TypeCode)
		tpl1.SetContent("Hello")
		mgr.Register(tpl1)

		tpl2 := NewTemplate("Admin Panel", "Admin interface", TypeCode)
		tpl2.SetContent("Admin")
		mgr.Register(tpl2)

		results := mgr.Search("user")
		assert.Len(t, results, 1)
		assert.Equal(t, "User Greeting", results[0].Name)
	})

	t.Run("get_by_tag", func(t *testing.T) {
		mgr := NewManager()

		tpl1 := NewTemplate("Test1", "Test", TypeCode)
		tpl1.SetContent("Content")
		tpl1.AddTag("go")
		mgr.Register(tpl1)

		tpl2 := NewTemplate("Test2", "Test", TypeCode)
		tpl2.SetContent("Content")
		tpl2.AddTag("python")
		mgr.Register(tpl2)

		goTemplates := mgr.GetByTag("go")
		assert.Len(t, goTemplates, 1)
		assert.Equal(t, "Test1", goTemplates[0].Name)
	})

	t.Run("count_by_type", func(t *testing.T) {
		mgr := NewManager()

		tpl1 := NewTemplate("Code1", "Test", TypeCode)
		tpl1.SetContent("Code")
		mgr.Register(tpl1)

		tpl2 := NewTemplate("Code2", "Test", TypeCode)
		tpl2.SetContent("Code")
		mgr.Register(tpl2)

		tpl3 := NewTemplate("Prompt1", "Test", TypePrompt)
		tpl3.SetContent("Prompt")
		mgr.Register(tpl3)

		counts := mgr.CountByType()
		assert.Equal(t, 2, counts[TypeCode])
		assert.Equal(t, 1, counts[TypePrompt])
	})

	t.Run("clear_templates", func(t *testing.T) {
		mgr := NewManager()

		tpl1 := NewTemplate("Test1", "Test", TypeCode)
		tpl1.SetContent("Content")
		mgr.Register(tpl1)

		tpl2 := NewTemplate("Test2", "Test", TypeCode)
		tpl2.SetContent("Content")
		mgr.Register(tpl2)

		assert.Equal(t, 2, mgr.Count())

		mgr.Clear()
		assert.Equal(t, 0, mgr.Count())
	})
}

func TestManagerFileOperations(t *testing.T) {
	t.Run("load_from_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewManager()

		// Create template
		tpl := NewTemplate("Test", "Test template", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})

		// Save to file
		filename := filepath.Join(tmpDir, "template.json")
		err := mgr.Register(tpl)
		require.NoError(t, err)
		err = mgr.SaveToFile(tpl.ID, filename)
		require.NoError(t, err)

		// Create new manager and load
		newMgr := NewManager()
		err = newMgr.LoadFromFile(filename)
		require.NoError(t, err)

		// Verify
		loaded, err := newMgr.GetByName("Test")
		require.NoError(t, err)
		assert.Equal(t, "Test", loaded.Name)
	})

	t.Run("load_from_directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewManager()

		// Create templates
		tpl1 := NewTemplate("Test1", "Test", TypeCode)
		tpl1.SetContent("Content 1")
		mgr.Register(tpl1)

		tpl2 := NewTemplate("Test2", "Test", TypeCode)
		tpl2.SetContent("Content 2")
		mgr.Register(tpl2)

		// Save to files
		mgr.SaveToFile(tpl1.ID, filepath.Join(tmpDir, "tpl1.json"))
		mgr.SaveToFile(tpl2.ID, filepath.Join(tmpDir, "tpl2.json"))

		// Create new manager and load directory
		newMgr := NewManager()
		count, err := newMgr.LoadFromDirectory(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Equal(t, 2, newMgr.Count())
	})

	t.Run("load_nonexistent_directory", func(t *testing.T) {
		mgr := NewManager()
		_, err := mgr.LoadFromDirectory("/nonexistent/path")
		assert.Error(t, err)
	})
}

func TestManagerCallbacks(t *testing.T) {
	t.Run("on_create_callback", func(t *testing.T) {
		mgr := NewManager()

		var called bool
		var capturedTemplate *Template

		mgr.OnCreate(func(tpl *Template) {
			called = true
			capturedTemplate = tpl
		})

		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")
		mgr.Register(tpl)

		assert.True(t, called)
		assert.NotNil(t, capturedTemplate)
		assert.Equal(t, "Test", capturedTemplate.Name)
	})

	t.Run("on_update_callback", func(t *testing.T) {
		mgr := NewManager()

		var called bool

		mgr.OnUpdate(func(tpl *Template) {
			called = true
		})

		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")
		mgr.Register(tpl)

		mgr.Update(tpl.ID, func(t *Template) {
			t.Description = "Updated"
		})

		assert.True(t, called)
	})

	t.Run("on_delete_callback", func(t *testing.T) {
		mgr := NewManager()

		var called bool

		mgr.OnDelete(func(tpl *Template) {
			called = true
		})

		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")
		mgr.Register(tpl)

		mgr.Delete(tpl.ID)

		assert.True(t, called)
	})
}

func TestManagerExportImport(t *testing.T) {
	t.Run("export_template", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")
		mgr.Register(tpl)

		snapshot, err := mgr.Export(tpl.ID)
		require.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, "Test", snapshot.Template.Name)
	})

	t.Run("import_template", func(t *testing.T) {
		mgr := NewManager()
		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello")

		snapshot := &TemplateSnapshot{
			Template:   tpl,
			ExportedAt: tpl.UpdatedAt,
		}

		err := mgr.Import(snapshot)
		require.NoError(t, err)
		assert.Equal(t, 1, mgr.Count())
	})
}

func TestBuiltinTemplates(t *testing.T) {
	t.Run("register_builtin_templates", func(t *testing.T) {
		mgr := NewManager()
		err := mgr.RegisterBuiltinTemplates()
		require.NoError(t, err)
		assert.Greater(t, mgr.Count(), 0)
	})

	t.Run("builtin_function_template", func(t *testing.T) {
		mgr := NewManager()
		mgr.RegisterBuiltinTemplates()

		result, err := mgr.RenderByName("Function", map[string]interface{}{
			"function_name": "add",
			"parameters":    "a, b int",
			"return_type":   "int",
			"body":          "return a + b",
		})

		require.NoError(t, err)
		assert.Contains(t, result, "func add")
		assert.Contains(t, result, "return a + b")
	})

	t.Run("builtin_code_review_template", func(t *testing.T) {
		mgr := NewManager()
		mgr.RegisterBuiltinTemplates()

		result, err := mgr.RenderByName("Code Review", map[string]interface{}{
			"language": "Go",
			"code":     "func main() {}",
		})

		require.NoError(t, err)
		assert.Contains(t, result, "Go")
		assert.Contains(t, result, "func main() {}")
	})
}

func TestStatistics(t *testing.T) {
	t.Run("get_statistics", func(t *testing.T) {
		mgr := NewManager()

		tpl1 := NewTemplate("Code1", "Test", TypeCode)
		tpl1.SetContent("Code")
		tpl1.AddTag("go")
		mgr.Register(tpl1)

		tpl2 := NewTemplate("Code2", "Test", TypeCode)
		tpl2.SetContent("Code")
		tpl2.AddTag("go")
		tpl2.AddTag("test")
		mgr.Register(tpl2)

		tpl3 := NewTemplate("Prompt1", "Test", TypePrompt)
		tpl3.SetContent("Prompt")
		mgr.Register(tpl3)

		stats := mgr.GetStatistics()
		assert.Equal(t, 3, stats.TotalTemplates)
		assert.Equal(t, 2, stats.ByType[TypeCode])
		assert.Equal(t, 1, stats.ByType[TypePrompt])
		assert.Equal(t, 2, stats.TagCloud["go"])
		assert.Equal(t, 1, stats.TagCloud["test"])
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent_register", func(t *testing.T) {
		mgr := NewManager()
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				tpl := NewTemplate(fmt.Sprintf("Test%d", idx), "Test", TypeCode)
				tpl.SetContent("Content")
				mgr.Register(tpl)
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 10, mgr.Count())
	})

	t.Run("concurrent_read", func(t *testing.T) {
		mgr := NewManager()

		tpl := NewTemplate("Test", "Test", TypeCode)
		tpl.SetContent("Hello {{name}}")
		tpl.AddVariable(Variable{Name: "name", Required: true})
		mgr.Register(tpl)

		var wg sync.WaitGroup
		errors := make([]error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_, errors[idx] = mgr.Render(tpl.ID, map[string]interface{}{
					"name": "Alice",
				})
			}(i)
		}

		wg.Wait()

		for _, err := range errors {
			assert.NoError(t, err)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty_content", func(t *testing.T) {
		tpl := NewTemplate("Test", "Test", TypeCode)
		err := tpl.Validate()
		assert.Error(t, err)
	})

	t.Run("get_nonexistent_template", func(t *testing.T) {
		mgr := NewManager()
		_, err := mgr.Get("nonexistent")
		assert.Error(t, err)
	})

	t.Run("delete_nonexistent_template", func(t *testing.T) {
		mgr := NewManager()
		err := mgr.Delete("nonexistent")
		assert.Error(t, err)
	})

	t.Run("update_nonexistent_template", func(t *testing.T) {
		mgr := NewManager()
		err := mgr.Update("nonexistent", func(t *Template) {})
		assert.Error(t, err)
	})

	t.Run("get_all_empty_manager", func(t *testing.T) {
		mgr := NewManager()
		all := mgr.GetAll()
		assert.Empty(t, all)
	})

	t.Run("get_by_type_empty", func(t *testing.T) {
		mgr := NewManager()
		templates := mgr.GetByType(TypeCode)
		assert.Empty(t, templates)
	})

	t.Run("search_empty_manager", func(t *testing.T) {
		mgr := NewManager()
		results := mgr.Search("test")
		assert.Empty(t, results)
	})
}
