package focus

import (
	"fmt"
	"testing"
	"time"
)

// TestFocus tests basic focus functionality
func TestFocus(t *testing.T) {
	t.Run("create_focus", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "main.go")

		if focus.ID == "" {
			t.Error("focus ID should not be empty")
		}

		if focus.Type != FocusTypeFile {
			t.Errorf("expected type file, got %s", focus.Type)
		}

		if focus.Target != "main.go" {
			t.Errorf("expected target main.go, got %s", focus.Target)
		}

		if focus.Priority != PriorityNormal {
			t.Errorf("expected normal priority, got %d", focus.Priority)
		}
	})

	t.Run("create_with_priority", func(t *testing.T) {
		focus := NewFocusWithPriority(FocusTypeError, "bug-123", PriorityCritical)

		if focus.Priority != PriorityCritical {
			t.Errorf("expected critical priority, got %d", focus.Priority)
		}
	})

	t.Run("validate", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		if err := focus.Validate(); err != nil {
			t.Errorf("validation should pass: %v", err)
		}
	})

	t.Run("validate_empty_id", func(t *testing.T) {
		focus := &Focus{
			ID:     "",
			Type:   FocusTypeFile,
			Target: "test.go",
		}

		if err := focus.Validate(); err == nil {
			t.Error("validation should fail for empty ID")
		}
	})

	t.Run("validate_empty_type", func(t *testing.T) {
		focus := &Focus{
			ID:     "test-id",
			Type:   "",
			Target: "test.go",
		}

		if err := focus.Validate(); err == nil {
			t.Error("validation should fail for empty type")
		}
	})

	t.Run("validate_empty_target", func(t *testing.T) {
		focus := &Focus{
			ID:     "test-id",
			Type:   FocusTypeFile,
			Target: "",
		}

		if err := focus.Validate(); err == nil {
			t.Error("validation should fail for empty target")
		}
	})

	t.Run("validate_invalid_priority", func(t *testing.T) {
		focus := &Focus{
			ID:       "test-id",
			Type:     FocusTypeFile,
			Target:   "test.go",
			Priority: 100, // Too high
		}

		if err := focus.Validate(); err == nil {
			t.Error("validation should fail for invalid priority")
		}
	})
}

// TestFocusTags tests tag functionality
func TestFocusTags(t *testing.T) {
	t.Run("add_tag", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		focus.AddTag("important")

		if !focus.HasTag("important") {
			t.Error("focus should have 'important' tag")
		}
	})

	t.Run("add_duplicate_tag", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		focus.AddTag("test")
		focus.AddTag("test")

		count := 0
		for _, tag := range focus.Tags {
			if tag == "test" {
				count++
			}
		}

		if count != 1 {
			t.Errorf("duplicate tag should not be added, found %d occurrences", count)
		}
	})

	t.Run("has_tag", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		focus.AddTag("backend")

		if !focus.HasTag("backend") {
			t.Error("should have 'backend' tag")
		}

		if focus.HasTag("frontend") {
			t.Error("should not have 'frontend' tag")
		}
	})
}

// TestFocusContext tests context management
func TestFocusContext(t *testing.T) {
	t.Run("set_and_get_context", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		focus.SetContext("line", 42)

		value, ok := focus.GetContext("line")
		if !ok {
			t.Error("context value should exist")
		}

		if value != 42 {
			t.Errorf("expected context value 42, got %v", value)
		}
	})

	t.Run("get_missing_context", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		_, ok := focus.GetContext("missing")
		if ok {
			t.Error("missing context should return false")
		}
	})
}

// TestFocusMetadata tests metadata management
func TestFocusMetadata(t *testing.T) {
	t.Run("set_and_get_metadata", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		focus.SetMetadata("author", "test-user")

		value, ok := focus.GetMetadata("author")
		if !ok {
			t.Error("metadata value should exist")
		}

		if value != "test-user" {
			t.Errorf("expected metadata value 'test-user', got %s", value)
		}
	})
}

// TestFocusExpiration tests expiration functionality
func TestFocusExpiration(t *testing.T) {
	t.Run("set_expiration", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		focus.SetExpiration(1 * time.Hour)

		if focus.ExpiresAt == nil {
			t.Error("expiration should be set")
		}
	})

	t.Run("is_expired", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		// Set expiration in the past
		focus.SetExpiration(-1 * time.Second)

		if !focus.IsExpired() {
			t.Error("focus should be expired")
		}
	})

	t.Run("not_expired", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		focus.SetExpiration(1 * time.Hour)

		if focus.IsExpired() {
			t.Error("focus should not be expired")
		}
	})

	t.Run("no_expiration", func(t *testing.T) {
		focus := NewFocus(FocusTypeFile, "test.go")

		if focus.IsExpired() {
			t.Error("focus without expiration should not be expired")
		}
	})
}

// TestFocusHierarchy tests hierarchical focus functionality
func TestFocusHierarchy(t *testing.T) {
	t.Run("add_child", func(t *testing.T) {
		parent := NewFocus(FocusTypeDirectory, "src")
		child := NewFocus(FocusTypeFile, "src/main.go")

		parent.AddChild(child)

		if len(parent.Children) != 1 {
			t.Errorf("expected 1 child, got %d", len(parent.Children))
		}

		if child.Parent != parent {
			t.Error("child's parent should be set")
		}
	})

	t.Run("remove_child", func(t *testing.T) {
		parent := NewFocus(FocusTypeDirectory, "src")
		child := NewFocus(FocusTypeFile, "src/main.go")

		parent.AddChild(child)

		if !parent.RemoveChild(child.ID) {
			t.Error("should successfully remove child")
		}

		if len(parent.Children) != 0 {
			t.Errorf("expected 0 children, got %d", len(parent.Children))
		}

		if child.Parent != nil {
			t.Error("child's parent should be nil after removal")
		}
	})

	t.Run("get_depth", func(t *testing.T) {
		root := NewFocus(FocusTypeProject, "myproject")
		dir := NewFocus(FocusTypeDirectory, "src")
		file := NewFocus(FocusTypeFile, "src/main.go")

		root.AddChild(dir)
		dir.AddChild(file)

		if root.GetDepth() != 0 {
			t.Errorf("root depth should be 0, got %d", root.GetDepth())
		}

		if dir.GetDepth() != 1 {
			t.Errorf("dir depth should be 1, got %d", dir.GetDepth())
		}

		if file.GetDepth() != 2 {
			t.Errorf("file depth should be 2, got %d", file.GetDepth())
		}
	})

	t.Run("get_root", func(t *testing.T) {
		root := NewFocus(FocusTypeProject, "myproject")
		dir := NewFocus(FocusTypeDirectory, "src")
		file := NewFocus(FocusTypeFile, "src/main.go")

		root.AddChild(dir)
		dir.AddChild(file)

		if file.GetRoot() != root {
			t.Error("file's root should be the project focus")
		}
	})

	t.Run("get_path", func(t *testing.T) {
		root := NewFocus(FocusTypeProject, "myproject")
		dir := NewFocus(FocusTypeDirectory, "src")
		file := NewFocus(FocusTypeFile, "src/main.go")

		root.AddChild(dir)
		dir.AddChild(file)

		path := file.GetPath()

		if len(path) != 3 {
			t.Errorf("expected path length 3, got %d", len(path))
		}

		if path[0] != root || path[1] != dir || path[2] != file {
			t.Error("path order is incorrect")
		}
	})

	t.Run("find_child", func(t *testing.T) {
		root := NewFocus(FocusTypeProject, "myproject")
		dir := NewFocus(FocusTypeDirectory, "src")
		file := NewFocus(FocusTypeFile, "src/main.go")

		root.AddChild(dir)
		dir.AddChild(file)

		found := root.FindChild(file.ID)
		if found != file {
			t.Error("should find file in hierarchy")
		}
	})

	t.Run("count_descendants", func(t *testing.T) {
		root := NewFocus(FocusTypeProject, "myproject")
		dir1 := NewFocus(FocusTypeDirectory, "src")
		dir2 := NewFocus(FocusTypeDirectory, "test")
		file1 := NewFocus(FocusTypeFile, "src/main.go")
		file2 := NewFocus(FocusTypeFile, "test/test.go")

		root.AddChild(dir1)
		root.AddChild(dir2)
		dir1.AddChild(file1)
		dir2.AddChild(file2)

		count := root.CountDescendants()
		if count != 4 {
			t.Errorf("expected 4 descendants, got %d", count)
		}
	})
}

// TestFocusClone tests cloning functionality
func TestFocusClone(t *testing.T) {
	t.Run("clone_focus", func(t *testing.T) {
		original := NewFocus(FocusTypeFile, "test.go")
		original.Description = "Test file"
		original.Priority = PriorityHigh
		original.AddTag("important")
		original.SetContext("line", 42)
		original.SetMetadata("author", "test-user")

		clone := original.Clone()

		if clone.ID != original.ID {
			t.Error("clone should have same ID")
		}

		if clone.Type != original.Type {
			t.Error("clone should have same type")
		}

		if clone.Target != original.Target {
			t.Error("clone should have same target")
		}

		if clone.Description != original.Description {
			t.Error("clone should have same description")
		}

		if clone.Priority != original.Priority {
			t.Error("clone should have same priority")
		}

		// Check deep copy
		original.SetContext("line", 100)
		value, _ := clone.GetContext("line")
		if value == 100 {
			t.Error("clone should not be affected by original changes")
		}
	})

	t.Run("clone_with_children", func(t *testing.T) {
		parent := NewFocus(FocusTypeDirectory, "src")
		child := NewFocus(FocusTypeFile, "src/main.go")
		parent.AddChild(child)

		clone := parent.Clone()

		if len(clone.Children) != 1 {
			t.Errorf("clone should have 1 child, got %d", len(clone.Children))
		}

		// Check deep copy
		parent.AddChild(NewFocus(FocusTypeFile, "src/test.go"))
		if len(clone.Children) != 1 {
			t.Error("clone should not be affected by original changes")
		}
	})
}

// TestChain tests basic chain functionality
func TestChain(t *testing.T) {
	t.Run("create_chain", func(t *testing.T) {
		chain := NewChain("test-chain")

		if chain.ID == "" {
			t.Error("chain ID should not be empty")
		}

		if chain.Name != "test-chain" {
			t.Errorf("expected name 'test-chain', got %s", chain.Name)
		}

		if !chain.IsEmpty() {
			t.Error("new chain should be empty")
		}
	})

	t.Run("push_focus", func(t *testing.T) {
		chain := NewChain("test")
		focus := NewFocus(FocusTypeFile, "test.go")

		err := chain.Push(focus)
		if err != nil {
			t.Errorf("push should succeed: %v", err)
		}

		if chain.Size() != 1 {
			t.Errorf("expected size 1, got %d", chain.Size())
		}
	})

	t.Run("pop_focus", func(t *testing.T) {
		chain := NewChain("test")
		focus1 := NewFocus(FocusTypeFile, "test1.go")
		focus2 := NewFocus(FocusTypeFile, "test2.go")

		chain.Push(focus1)
		chain.Push(focus2)

		popped, err := chain.Pop()
		if err != nil {
			t.Errorf("pop should succeed: %v", err)
		}

		if popped != focus2 {
			t.Error("should pop focus2")
		}

		if chain.Size() != 1 {
			t.Errorf("expected size 1, got %d", chain.Size())
		}
	})

	t.Run("current_focus", func(t *testing.T) {
		chain := NewChain("test")
		focus := NewFocus(FocusTypeFile, "test.go")

		chain.Push(focus)

		current, err := chain.Current()
		if err != nil {
			t.Errorf("current should succeed: %v", err)
		}

		if current != focus {
			t.Error("current should return pushed focus")
		}
	})
}

// TestChainNavigation tests chain navigation
func TestChainNavigation(t *testing.T) {
	t.Run("next_and_previous", func(t *testing.T) {
		chain := NewChain("test")
		focus1 := NewFocus(FocusTypeFile, "test1.go")
		focus2 := NewFocus(FocusTypeFile, "test2.go")
		focus3 := NewFocus(FocusTypeFile, "test3.go")

		chain.Push(focus1)
		chain.Push(focus2)
		chain.Push(focus3)

		// At focus3, go back
		prev, err := chain.Previous()
		if err != nil {
			t.Errorf("previous should succeed: %v", err)
		}
		if prev != focus2 {
			t.Error("previous should be focus2")
		}

		// At focus2, go back again
		prev, err = chain.Previous()
		if err != nil {
			t.Errorf("previous should succeed: %v", err)
		}
		if prev != focus1 {
			t.Error("previous should be focus1")
		}

		// At focus1, go forward
		next, err := chain.Next()
		if err != nil {
			t.Errorf("next should succeed: %v", err)
		}
		if next != focus2 {
			t.Error("next should be focus2")
		}
	})

	t.Run("first_and_last", func(t *testing.T) {
		chain := NewChain("test")
		focus1 := NewFocus(FocusTypeFile, "test1.go")
		focus2 := NewFocus(FocusTypeFile, "test2.go")

		chain.Push(focus1)
		chain.Push(focus2)

		first, _ := chain.First()
		if first != focus1 {
			t.Error("first should be focus1")
		}

		last, _ := chain.Last()
		if last != focus2 {
			t.Error("last should be focus2")
		}
	})

	t.Run("get_by_index", func(t *testing.T) {
		chain := NewChain("test")
		focus1 := NewFocus(FocusTypeFile, "test1.go")
		focus2 := NewFocus(FocusTypeFile, "test2.go")

		chain.Push(focus1)
		chain.Push(focus2)

		got, err := chain.Get(0)
		if err != nil {
			t.Errorf("get should succeed: %v", err)
		}
		if got != focus1 {
			t.Error("get(0) should return focus1")
		}

		got, err = chain.Get(1)
		if err != nil {
			t.Errorf("get should succeed: %v", err)
		}
		if got != focus2 {
			t.Error("get(1) should return focus2")
		}
	})

	t.Run("get_by_id", func(t *testing.T) {
		chain := NewChain("test")
		focus := NewFocus(FocusTypeFile, "test.go")

		chain.Push(focus)

		got, err := chain.GetByID(focus.ID)
		if err != nil {
			t.Errorf("get by ID should succeed: %v", err)
		}
		if got != focus {
			t.Error("should return correct focus")
		}
	})
}

// TestChainFiltering tests chain filtering
func TestChainFiltering(t *testing.T) {
	t.Run("get_by_type", func(t *testing.T) {
		chain := NewChain("test")
		file1 := NewFocus(FocusTypeFile, "test1.go")
		file2 := NewFocus(FocusTypeFile, "test2.go")
		task := NewFocus(FocusTypeTask, "implement-feature")

		chain.Push(file1)
		chain.Push(task)
		chain.Push(file2)

		files := chain.GetByType(FocusTypeFile)
		if len(files) != 2 {
			t.Errorf("expected 2 file focuses, got %d", len(files))
		}

		tasks := chain.GetByType(FocusTypeTask)
		if len(tasks) != 1 {
			t.Errorf("expected 1 task focus, got %d", len(tasks))
		}
	})

	t.Run("get_by_tag", func(t *testing.T) {
		chain := NewChain("test")
		focus1 := NewFocus(FocusTypeFile, "test1.go")
		focus1.AddTag("important")
		focus2 := NewFocus(FocusTypeFile, "test2.go")
		focus2.AddTag("important")
		focus3 := NewFocus(FocusTypeFile, "test3.go")

		chain.Push(focus1)
		chain.Push(focus2)
		chain.Push(focus3)

		important := chain.GetByTag("important")
		if len(important) != 2 {
			t.Errorf("expected 2 important focuses, got %d", len(important))
		}
	})

	t.Run("get_by_priority", func(t *testing.T) {
		chain := NewChain("test")
		low := NewFocusWithPriority(FocusTypeFile, "low.go", PriorityLow)
		high := NewFocusWithPriority(FocusTypeFile, "high.go", PriorityHigh)
		critical := NewFocusWithPriority(FocusTypeFile, "critical.go", PriorityCritical)

		chain.Push(low)
		chain.Push(high)
		chain.Push(critical)

		highPriority := chain.GetByPriority(PriorityHigh)
		if len(highPriority) != 2 {
			t.Errorf("expected 2 high+ priority focuses, got %d", len(highPriority))
		}
	})

	t.Run("get_recent", func(t *testing.T) {
		chain := NewChain("test")
		for i := 0; i < 5; i++ {
			chain.Push(NewFocus(FocusTypeFile, fmt.Sprintf("test%d.go", i)))
		}

		recent := chain.GetRecent(3)
		if len(recent) != 3 {
			t.Errorf("expected 3 recent focuses, got %d", len(recent))
		}

		// Check order (should be last 3)
		if recent[0].Target != "test2.go" {
			t.Error("recent[0] should be test2.go")
		}
		if recent[2].Target != "test4.go" {
			t.Error("recent[2] should be test4.go")
		}
	})
}

// TestChainOperations tests chain operations
func TestChainOperations(t *testing.T) {
	t.Run("remove_focus", func(t *testing.T) {
		chain := NewChain("test")
		focus1 := NewFocus(FocusTypeFile, "test1.go")
		focus2 := NewFocus(FocusTypeFile, "test2.go")

		chain.Push(focus1)
		chain.Push(focus2)

		err := chain.Remove(focus1.ID)
		if err != nil {
			t.Errorf("remove should succeed: %v", err)
		}

		if chain.Size() != 1 {
			t.Errorf("expected size 1, got %d", chain.Size())
		}
	})

	t.Run("clear_chain", func(t *testing.T) {
		chain := NewChain("test")
		chain.Push(NewFocus(FocusTypeFile, "test1.go"))
		chain.Push(NewFocus(FocusTypeFile, "test2.go"))

		chain.Clear()

		if !chain.IsEmpty() {
			t.Error("chain should be empty after clear")
		}
	})

	t.Run("clean_expired", func(t *testing.T) {
		chain := NewChain("test")
		focus1 := NewFocus(FocusTypeFile, "expired.go")
		focus2 := NewFocus(FocusTypeFile, "active.go")

		// Push both focuses first (not expired yet)
		chain.Push(focus1)
		chain.Push(focus2)

		// Now expire the first focus
		expiredTime := time.Now().Add(-1 * time.Hour)
		focus1.ExpiresAt = &expiredTime

		// Clean expired
		removed := chain.CleanExpired()

		if removed != 1 {
			t.Errorf("expected 1 removed, got %d", removed)
		}

		if chain.Size() != 1 {
			t.Errorf("expected size 1, got %d", chain.Size())
		}
	})

	t.Run("merge_chains", func(t *testing.T) {
		chain1 := NewChain("chain1")
		chain2 := NewChain("chain2")

		chain1.Push(NewFocus(FocusTypeFile, "test1.go"))
		chain2.Push(NewFocus(FocusTypeFile, "test2.go"))
		chain2.Push(NewFocus(FocusTypeFile, "test3.go"))

		err := chain1.Merge(chain2)
		if err != nil {
			t.Errorf("merge should succeed: %v", err)
		}

		if chain1.Size() != 3 {
			t.Errorf("expected size 3 after merge, got %d", chain1.Size())
		}
	})

	t.Run("split_chain", func(t *testing.T) {
		chain := NewChain("test")
		chain.Push(NewFocus(FocusTypeFile, "test1.go"))
		chain.Push(NewFocus(FocusTypeFile, "test2.go"))
		chain.Push(NewFocus(FocusTypeFile, "test3.go"))

		newChain, err := chain.Split(2)
		if err != nil {
			t.Errorf("split should succeed: %v", err)
		}

		if chain.Size() != 2 {
			t.Errorf("original chain should have size 2, got %d", chain.Size())
		}

		if newChain.Size() != 1 {
			t.Errorf("new chain should have size 1, got %d", newChain.Size())
		}
	})

	t.Run("reverse_chain", func(t *testing.T) {
		chain := NewChain("test")
		focus1 := NewFocus(FocusTypeFile, "test1.go")
		focus2 := NewFocus(FocusTypeFile, "test2.go")
		focus3 := NewFocus(FocusTypeFile, "test3.go")

		chain.Push(focus1)
		chain.Push(focus2)
		chain.Push(focus3)

		chain.Reverse()

		first, _ := chain.First()
		if first != focus3 {
			t.Error("first should be focus3 after reverse")
		}

		last, _ := chain.Last()
		if last != focus1 {
			t.Error("last should be focus1 after reverse")
		}
	})
}

// TestManager tests manager functionality
func TestManager(t *testing.T) {
	t.Run("create_manager", func(t *testing.T) {
		manager := NewManager()

		if manager.Count() != 0 {
			t.Error("new manager should have 0 chains")
		}
	})

	t.Run("create_chain", func(t *testing.T) {
		manager := NewManager()

		chain, err := manager.CreateChain("test", true)
		if err != nil {
			t.Errorf("create chain should succeed: %v", err)
		}

		if chain == nil {
			t.Error("chain should not be nil")
		}

		if manager.Count() != 1 {
			t.Errorf("expected 1 chain, got %d", manager.Count())
		}
	})

	t.Run("get_chain", func(t *testing.T) {
		manager := NewManager()
		chain, _ := manager.CreateChain("test", false)

		got, err := manager.GetChain(chain.ID)
		if err != nil {
			t.Errorf("get chain should succeed: %v", err)
		}

		if got != chain {
			t.Error("should return correct chain")
		}
	})

	t.Run("set_active_chain", func(t *testing.T) {
		manager := NewManager()
		chain, _ := manager.CreateChain("test", false)

		err := manager.SetActiveChain(chain.ID)
		if err != nil {
			t.Errorf("set active should succeed: %v", err)
		}

		active, _ := manager.GetActiveChain()
		if active != chain {
			t.Error("active chain should be the created chain")
		}
	})

	t.Run("delete_chain", func(t *testing.T) {
		manager := NewManager()
		chain, _ := manager.CreateChain("test", false)

		err := manager.DeleteChain(chain.ID)
		if err != nil {
			t.Errorf("delete chain should succeed: %v", err)
		}

		if manager.Count() != 0 {
			t.Errorf("expected 0 chains, got %d", manager.Count())
		}
	})

	t.Run("push_to_active", func(t *testing.T) {
		manager := NewManager()
		manager.CreateChain("test", true)
		focus := NewFocus(FocusTypeFile, "test.go")

		err := manager.PushToActive(focus)
		if err != nil {
			t.Errorf("push to active should succeed: %v", err)
		}

		current, _ := manager.GetCurrentFocus()
		if current != focus {
			t.Error("current focus should be pushed focus")
		}
	})

	t.Run("get_statistics", func(t *testing.T) {
		manager := NewManager()
		chain1, _ := manager.CreateChain("test1", false)
		chain2, _ := manager.CreateChain("test2", false)

		chain1.Push(NewFocus(FocusTypeFile, "test1.go"))
		chain1.Push(NewFocus(FocusTypeFile, "test2.go"))
		chain2.Push(NewFocus(FocusTypeFile, "test3.go"))

		stats := manager.GetStatistics()

		if stats.TotalChains != 2 {
			t.Errorf("expected 2 total chains, got %d", stats.TotalChains)
		}

		if stats.TotalFocuses != 3 {
			t.Errorf("expected 3 total focuses, got %d", stats.TotalFocuses)
		}

		expectedAvg := 1.5
		if stats.AverageFocusesPerChain != expectedAvg {
			t.Errorf("expected average %.1f, got %.1f", expectedAvg, stats.AverageFocusesPerChain)
		}
	})

	t.Run("export_and_import", func(t *testing.T) {
		manager1 := NewManager()
		chain, _ := manager1.CreateChain("test", false)
		chain.Push(NewFocus(FocusTypeFile, "test.go"))

		snapshot, err := manager1.ExportChain(chain.ID)
		if err != nil {
			t.Errorf("export should succeed: %v", err)
		}

		manager2 := NewManager()
		err = manager2.ImportChain(snapshot, true)
		if err != nil {
			t.Errorf("import should succeed: %v", err)
		}

		active, _ := manager2.GetActiveChain()
		if active.Size() != 1 {
			t.Errorf("imported chain should have 1 focus, got %d", active.Size())
		}
	})
}
