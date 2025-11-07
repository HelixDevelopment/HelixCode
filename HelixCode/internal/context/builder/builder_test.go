package builder

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.helix.code/internal/focus"
	"dev.helix.code/internal/session"
)

func TestBuilder(t *testing.T) {
	t.Run("create_builder", func(t *testing.T) {
		builder := NewBuilder()
		assert.NotNil(t, builder)
		assert.Equal(t, 0, builder.Count())
		assert.Equal(t, 0, builder.TotalSize())
	})

	t.Run("add_text", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Test", "Hello World", PriorityNormal)

		assert.Equal(t, 1, builder.Count())
		assert.Greater(t, builder.TotalSize(), 0)
	})

	t.Run("add_item", func(t *testing.T) {
		builder := NewBuilder()
		item := &ContextItem{
			Type:     SourceCustom,
			Priority: PriorityHigh,
			Title:    "Test Item",
			Content:  "Test Content",
			Metadata: make(map[string]string),
		}

		builder.AddItem(item)
		assert.Equal(t, 1, builder.Count())
		assert.Greater(t, item.Size, 0) // Size calculated automatically
	})

	t.Run("build_context", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Title1", "Content1", PriorityNormal)
		builder.AddText("Title2", "Content2", PriorityNormal)

		context, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, context, "Title1")
		assert.Contains(t, context, "Content1")
		assert.Contains(t, context, "Title2")
		assert.Contains(t, context, "Content2")
	})

	t.Run("clear", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Test", "Content", PriorityNormal)
		assert.Equal(t, 1, builder.Count())

		builder.Clear()
		assert.Equal(t, 0, builder.Count())
	})
}

func TestPriority(t *testing.T) {
	t.Run("sort_by_priority", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Low", "Low priority", PriorityLow)
		builder.AddText("High", "High priority", PriorityHigh)
		builder.AddText("Normal", "Normal priority", PriorityNormal)
		builder.AddText("Critical", "Critical priority", PriorityCritical)

		context, err := builder.Build()
		require.NoError(t, err)

		// Check order: Critical, High, Normal, Low
		criticalPos := strings.Index(context, "Critical")
		highPos := strings.Index(context, "High priority")
		normalPos := strings.Index(context, "Normal priority")
		lowPos := strings.Index(context, "Low priority")

		assert.Less(t, criticalPos, highPos)
		assert.Less(t, highPos, normalPos)
		assert.Less(t, normalPos, lowPos)
	})

	t.Run("get_by_priority", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Low", "Content", PriorityLow)
		builder.AddText("High", "Content", PriorityHigh)
		builder.AddText("Normal", "Content", PriorityNormal)

		highPriority := builder.GetItemsByPriority(PriorityHigh)
		assert.Len(t, highPriority, 1) // Only high priority items
	})
}

func TestSizeLimits(t *testing.T) {
	t.Run("max_size_limit", func(t *testing.T) {
		builder := NewBuilder()
		builder.SetMaxSize(100) // Very small limit

		// Add large content
		builder.AddText("Large", strings.Repeat("x", 1000), PriorityNormal)

		context, err := builder.Build()
		require.NoError(t, err)
		assert.LessOrEqual(t, len(context), 150) // Should be truncated (with headers)
	})

	t.Run("max_tokens", func(t *testing.T) {
		builder := NewBuilder()
		builder.SetMaxTokens(10) // ~40 bytes

		builder.AddText("Test", strings.Repeat("x", 1000), PriorityNormal)

		context, err := builder.Build()
		require.NoError(t, err)
		assert.LessOrEqual(t, len(context), 60) // Should be limited
	})
}

func TestSessionIntegration(t *testing.T) {
	t.Run("add_session", func(t *testing.T) {
		builder := NewBuilder()
		sessionMgr := session.NewManager()

		sess, _ := sessionMgr.Create("proj-1", "Test Session", "Test", session.ModePlanning)
		sess.AddTag("critical")
		sess.SetMetadata("author", "alice")

		err := builder.AddSession(sess)
		require.NoError(t, err)

		context, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, context, "Test Session")
		assert.Contains(t, context, "planning")
		assert.Contains(t, context, "critical")
		assert.Contains(t, context, "alice")
	})

	t.Run("add_nil_session", func(t *testing.T) {
		builder := NewBuilder()
		err := builder.AddSession(nil)
		assert.Error(t, err)
	})
}

func TestFocusIntegration(t *testing.T) {
	t.Run("add_focus_chain", func(t *testing.T) {
		builder := NewBuilder()
		focusMgr := focus.NewManager()

		chain, _ := focusMgr.CreateChain("test-chain", true)
		f1 := focus.NewFocus(focus.FocusTypeFile, "file1.go")
		f2 := focus.NewFocus(focus.FocusTypeFile, "file2.go")
		chain.Push(f1)
		chain.Push(f2)

		err := builder.AddFocusChain(chain, 10)
		require.NoError(t, err)

		context, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, context, "file1.go")
		assert.Contains(t, context, "file2.go")
	})

	t.Run("add_nil_chain", func(t *testing.T) {
		builder := NewBuilder()
		err := builder.AddFocusChain(nil, 10)
		assert.Error(t, err)
	})
}

func TestSources(t *testing.T) {
	t.Run("session_source", func(t *testing.T) {
		sessionMgr := session.NewManager()
		sess, _ := sessionMgr.Create("proj-1", "Test", "Desc", session.ModePlanning)
		sessionMgr.Start(sess.ID)

		source := NewSessionSource(sessionMgr)
		items, err := source.GetContext()
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, SourceSession, items[0].Type)
		assert.Contains(t, items[0].Content, "Test")
	})

	t.Run("focus_source", func(t *testing.T) {
		focusMgr := focus.NewManager()
		chain, _ := focusMgr.CreateChain("test", true)
		f := focus.NewFocus(focus.FocusTypeFile, "test.go")
		chain.Push(f)

		source := NewFocusSource(focusMgr, 10)
		items, err := source.GetContext()
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, SourceFocus, items[0].Type)
		assert.Contains(t, items[0].Content, "test.go")
	})

	t.Run("project_source", func(t *testing.T) {
		metadata := map[string]string{
			"language": "Go",
			"version":  "1.21",
		}
		source := NewProjectSource("MyProject", "A test project", metadata)

		items, err := source.GetContext()
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, SourceProject, items[0].Type)
		assert.Contains(t, items[0].Content, "MyProject")
		assert.Contains(t, items[0].Content, "Go")
	})

	t.Run("file_source", func(t *testing.T) {
		source := NewFileSource("test.go", "package main\n\nfunc main() {}", PriorityHigh)

		items, err := source.GetContext()
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, SourceFile, items[0].Type)
		assert.Contains(t, items[0].Content, "test.go")
		assert.Contains(t, items[0].Content, "package main")
	})

	t.Run("error_source", func(t *testing.T) {
		source := NewErrorSource()
		source.AddError("Null pointer exception", "handler.go", 42, "2025-01-01")
		source.AddError("Connection timeout", "client.go", 10, "2025-01-01")

		items, err := source.GetContext()
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, SourceError, items[0].Type)
		assert.Equal(t, PriorityCritical, items[0].Priority)
		assert.Contains(t, items[0].Content, "Null pointer")
		assert.Contains(t, items[0].Content, "handler.go:42")
	})

	t.Run("custom_source", func(t *testing.T) {
		source := NewCustomSource(SourceCustom, func() ([]*ContextItem, error) {
			return []*ContextItem{
				{
					Type:     SourceCustom,
					Priority: PriorityNormal,
					Title:    "Custom",
					Content:  "Custom content",
					Metadata: make(map[string]string),
					Size:     14,
				},
			}, nil
		})

		items, err := source.GetContext()
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Contains(t, items[0].Content, "Custom content")
	})
}

func TestSourceRegistration(t *testing.T) {
	t.Run("register_and_build_from_sources", func(t *testing.T) {
		builder := NewBuilder()

		// Register project source
		projectSource := NewProjectSource("Test", "Description", nil)
		builder.RegisterSource(projectSource)

		// Register custom source
		customSource := NewCustomSource(SourceCustom, func() ([]*ContextItem, error) {
			return []*ContextItem{
				{
					Type:     SourceCustom,
					Priority: PriorityNormal,
					Title:    "Custom",
					Content:  "Custom data",
					Metadata: make(map[string]string),
					Size:     11,
				},
			}, nil
		})
		builder.RegisterSource(customSource)

		context, err := builder.BuildFromSources()
		require.NoError(t, err)
		assert.Contains(t, context, "Test")
		assert.Contains(t, context, "Custom data")
	})
}

func TestTemplates(t *testing.T) {
	t.Run("register_template", func(t *testing.T) {
		builder := NewBuilder()
		template := &Template{
			Name:        "test",
			Description: "Test template",
			Sections: []*TemplateSection{
				{
					Title:    "Session",
					Types:    []SourceType{SourceSession},
					Priority: PriorityHigh,
					MaxItems: 1,
				},
			},
		}

		builder.RegisterTemplate(template)

		// Try to build with template
		_, err := builder.BuildWithTemplate("test")
		require.NoError(t, err) // Should not error even with no items
	})

	t.Run("build_with_nonexistent_template", func(t *testing.T) {
		builder := NewBuilder()
		_, err := builder.BuildWithTemplate("nonexistent")
		assert.Error(t, err)
	})

	t.Run("coding_template", func(t *testing.T) {
		builder := NewBuilder()
		template := GetCodingTemplate()
		builder.RegisterTemplate(template)

		// Add some items
		builder.AddText("Test", "Content", PriorityNormal)
		builder.AddItem(&ContextItem{
			Type:     SourceSession,
			Priority: PriorityHigh,
			Title:    "Session",
			Content:  "Session content",
			Metadata: make(map[string]string),
			Size:     15,
		})

		context, err := builder.BuildWithTemplate("coding")
		require.NoError(t, err)
		assert.Contains(t, context, "Current Session")
	})

	t.Run("register_default_templates", func(t *testing.T) {
		builder := NewBuilder()
		RegisterDefaultTemplates(builder)

		// Should be able to build with all default templates
		templates := []string{"coding", "debugging", "planning", "review", "refactoring"}
		for _, name := range templates {
			_, err := builder.BuildWithTemplate(name)
			require.NoError(t, err, "Template %s should work", name)
		}
	})
}

func TestQueries(t *testing.T) {
	t.Run("get_items", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Test1", "Content1", PriorityNormal)
		builder.AddText("Test2", "Content2", PriorityNormal)

		items := builder.GetItems()
		assert.Len(t, items, 2)
	})

	t.Run("get_by_type", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddItem(&ContextItem{
			Type:     SourceSession,
			Priority: PriorityNormal,
			Content:  "Session",
			Metadata: make(map[string]string),
			Size:     7,
		})
		builder.AddItem(&ContextItem{
			Type:     SourceFile,
			Priority: PriorityNormal,
			Content:  "File",
			Metadata: make(map[string]string),
			Size:     4,
		})

		sessionItems := builder.GetItemsByType(SourceSession)
		assert.Len(t, sessionItems, 1)

		fileItems := builder.GetItemsByType(SourceFile)
		assert.Len(t, fileItems, 1)
	})

	t.Run("count_and_size", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Test", "Content", PriorityNormal)

		assert.Equal(t, 1, builder.Count())
		assert.Greater(t, builder.TotalSize(), 0)
	})
}

func TestStatistics(t *testing.T) {
	t.Run("get_statistics", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddItem(&ContextItem{
			Type:     SourceSession,
			Priority: PriorityHigh,
			Content:  "Session",
			Metadata: make(map[string]string),
			Size:     100,
		})
		builder.AddItem(&ContextItem{
			Type:     SourceFile,
			Priority: PriorityNormal,
			Content:  "File",
			Metadata: make(map[string]string),
			Size:     200,
		})

		stats := builder.GetStatistics()
		assert.Equal(t, 2, stats.TotalItems)
		assert.Equal(t, 300, stats.TotalSize)
		assert.Equal(t, 1, stats.ByType[SourceSession])
		assert.Equal(t, 1, stats.ByType[SourceFile])
		assert.Equal(t, 1, stats.ByPriority[PriorityHigh])
		assert.Equal(t, 1, stats.ByPriority[PriorityNormal])
	})
}

func TestCaching(t *testing.T) {
	t.Run("cache_basic", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Test", "Content", PriorityNormal)

		// First build - cache miss
		context1, _ := builder.Build()
		stats1 := builder.GetStatistics()
		assert.Equal(t, 0, stats1.CacheHits)
		assert.Equal(t, 1, stats1.CacheMisses)

		// Second build - cache hit
		context2, _ := builder.Build()
		assert.Equal(t, context1, context2)
		stats2 := builder.GetStatistics()
		assert.Equal(t, 1, stats2.CacheHits)
	})

	t.Run("cache_invalidation", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Test", "Content", PriorityNormal)

		// Build and cache
		builder.Build()

		// Add new item - should invalidate cache
		builder.AddText("New", "New content", PriorityNormal)

		// Next build should be cache miss
		builder.Build()
		stats := builder.GetStatistics()
		assert.Equal(t, 2, stats.CacheMisses) // One for each build
	})

	t.Run("cache_enable_disable", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Test", "Content", PriorityNormal)

		// Disable cache
		builder.EnableCache(false)

		builder.Build()
		builder.Build()

		stats := builder.GetStatistics()
		assert.Equal(t, 0, stats.CacheHits) // No caching
	})

	t.Run("manual_invalidation", func(t *testing.T) {
		builder := NewBuilder()
		builder.AddText("Test", "Content", PriorityNormal)

		builder.Build()
		builder.InvalidateCache()
		builder.Build()

		stats := builder.GetStatistics()
		assert.Equal(t, 2, stats.CacheMisses)
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent_add", func(t *testing.T) {
		builder := NewBuilder()
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				builder.AddText("Test", "Content", PriorityNormal)
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 10, builder.Count())
	})

	t.Run("concurrent_build_and_add", func(t *testing.T) {
		builder := NewBuilder()
		var wg sync.WaitGroup

		// Add initial item
		builder.AddText("Initial", "Content", PriorityNormal)

		// Concurrent builds
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				builder.Build()
			}()
		}

		// Concurrent adds
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				builder.AddText("Test", "Content", PriorityNormal)
			}(i)
		}

		wg.Wait()
		assert.Greater(t, builder.Count(), 0)
	})

	t.Run("concurrent_source_registration", func(t *testing.T) {
		builder := NewBuilder()
		var wg sync.WaitGroup

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				source := NewProjectSource("Project", "Desc", nil)
				builder.RegisterSource(source)
			}(i)
		}

		wg.Wait()
		// Should not panic or error
	})
}

func TestBuilderWithManagers(t *testing.T) {
	t.Run("create_with_managers", func(t *testing.T) {
		sessionMgr := session.NewManager()
		focusMgr := focus.NewManager()

		builder := NewBuilderWithManagers(sessionMgr, focusMgr)
		assert.NotNil(t, builder)
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty_builder", func(t *testing.T) {
		builder := NewBuilder()
		context, err := builder.Build()
		require.NoError(t, err)
		assert.Empty(t, context)
	})

	t.Run("large_content", func(t *testing.T) {
		builder := NewBuilder()
		largeContent := strings.Repeat("x", 1000000) // 1MB
		builder.AddText("Large", largeContent, PriorityNormal)

		context, err := builder.Build()
		require.NoError(t, err)
		// Should be limited by maxSize
		assert.Less(t, len(context), 200000) // Well under 1MB
	})

	t.Run("zero_size_item", func(t *testing.T) {
		builder := NewBuilder()
		item := &ContextItem{
			Type:     SourceCustom,
			Priority: PriorityNormal,
			Content:  "Test",
			Metadata: make(map[string]string),
			Size:     0, // Explicitly zero
		}
		builder.AddItem(item)

		// Size should be calculated
		assert.Greater(t, item.Size, 0)
	})
}
