package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRule tests basic rule functionality
func TestRule(t *testing.T) {
	t.Run("validate", func(t *testing.T) {
		rule := &Rule{
			Name:    "Test Rule",
			Pattern: "*.go",
			Content: "Test content",
		}

		if err := rule.Validate(); err != nil {
			t.Errorf("validation should pass: %v", err)
		}
	})

	t.Run("validate_empty_name", func(t *testing.T) {
		rule := &Rule{
			Name:    "",
			Pattern: "*.go",
			Content: "Test content",
		}

		if err := rule.Validate(); err == nil {
			t.Error("validation should fail for empty name")
		}
	})

	t.Run("validate_empty_pattern", func(t *testing.T) {
		rule := &Rule{
			Name:        "Test",
			Pattern:     "",
			PatternType: PatternTypeGlob,
			Content:     "Test content",
		}

		if err := rule.Validate(); err == nil {
			t.Error("validation should fail for empty pattern")
		}
	})

	t.Run("validate_empty_content", func(t *testing.T) {
		rule := &Rule{
			Name:    "Test",
			Pattern: "*.go",
			Content: "",
		}

		if err := rule.Validate(); err == nil {
			t.Error("validation should fail for empty content")
		}
	})

	t.Run("validate_invalid_regex", func(t *testing.T) {
		rule := &Rule{
			Name:        "Test",
			Pattern:     "[invalid(regex",
			PatternType: PatternTypeRegex,
			Content:     "Test content",
		}

		if err := rule.Validate(); err == nil {
			t.Error("validation should fail for invalid regex")
		}
	})
}

// TestRuleMatching tests pattern matching
func TestRuleMatching(t *testing.T) {
	t.Run("glob_match", func(t *testing.T) {
		rule := &Rule{
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
		}

		if !rule.Matches("main.go") {
			t.Error("should match main.go")
		}

		if !rule.Matches("test.go") {
			t.Error("should match test.go")
		}

		if rule.Matches("main.js") {
			t.Error("should not match main.js")
		}
	})

	t.Run("glob_path_match", func(t *testing.T) {
		rule := &Rule{
			Pattern:     "src/**/*.go",
			PatternType: PatternTypeGlob,
		}

		if !rule.Matches("src/main.go") {
			t.Error("should match src/main.go")
		}

		if !rule.Matches("src/utils/helper.go") {
			t.Error("should match src/utils/helper.go")
		}

		if rule.Matches("test/main.go") {
			t.Error("should not match test/main.go")
		}
	})

	t.Run("exact_match", func(t *testing.T) {
		rule := &Rule{
			Pattern:     "src/main.go",
			PatternType: PatternTypeExact,
		}

		if !rule.Matches("src/main.go") {
			t.Error("should match src/main.go")
		}

		if rule.Matches("src/test.go") {
			t.Error("should not match src/test.go")
		}
	})

	t.Run("regex_match", func(t *testing.T) {
		rule := &Rule{
			Pattern:     `.*\.go$`,
			PatternType: PatternTypeRegex,
		}

		if !rule.Matches("main.go") {
			t.Error("should match main.go")
		}

		if !rule.Matches("src/main.go") {
			t.Error("should match src/main.go")
		}

		if rule.Matches("main.js") {
			t.Error("should not match main.js")
		}
	})

	t.Run("any_match", func(t *testing.T) {
		rule := &Rule{
			PatternType: PatternTypeAny,
		}

		if !rule.Matches("any/file/path.txt") {
			t.Error("should match any file")
		}
	})
}

// TestRuleScoring tests match scoring
func TestRuleScoring(t *testing.T) {
	t.Run("exact_pattern_higher_score", func(t *testing.T) {
		exactRule := &Rule{
			Pattern:     "src/main.go",
			PatternType: PatternTypeExact,
			Priority:    1,
			Scope:       RuleScopeFile,
		}

		globRule := &Rule{
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Priority:    1,
			Scope:       RuleScopeGlobal,
		}

		exactScore := exactRule.MatchScore("src/main.go")
		globScore := globRule.MatchScore("src/main.go")

		if exactScore <= globScore {
			t.Errorf("exact pattern should have higher score: exact=%d, glob=%d", exactScore, globScore)
		}
	})

	t.Run("higher_priority_higher_score", func(t *testing.T) {
		highPriority := &Rule{
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Priority:    10,
			Scope:       RuleScopeGlobal,
		}

		lowPriority := &Rule{
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Priority:    1,
			Scope:       RuleScopeGlobal,
		}

		highScore := highPriority.MatchScore("main.go")
		lowScore := lowPriority.MatchScore("main.go")

		if highScore <= lowScore {
			t.Errorf("higher priority should have higher score: high=%d, low=%d", highScore, lowScore)
		}
	})

	t.Run("no_match_zero_score", func(t *testing.T) {
		rule := &Rule{
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Priority:    10,
		}

		score := rule.MatchScore("main.js")
		if score != 0 {
			t.Errorf("non-matching file should have zero score, got %d", score)
		}
	})
}

// TestRuleClone tests rule cloning
func TestRuleClone(t *testing.T) {
	original := &Rule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "Test description",
		Pattern:     "*.go",
		PatternType: PatternTypeGlob,
		Content:     "Test content",
		Priority:    5,
		Category:    RuleCategoryStyle,
		Scope:       RuleScopeGlobal,
		Tags:        []string{"tag1", "tag2"},
		Metadata:    map[string]string{"key": "value"},
	}

	clone := original.Clone()

	// Verify clone has same values
	if clone.ID != original.ID {
		t.Error("clone should have same ID")
	}
	if clone.Name != original.Name {
		t.Error("clone should have same name")
	}

	// Verify deep copy (modifying clone doesn't affect original)
	clone.Tags[0] = "modified"
	if original.Tags[0] == "modified" {
		t.Error("modifying clone should not affect original")
	}

	clone.Metadata["key"] = "modified"
	if original.Metadata["key"] == "modified" {
		t.Error("modifying clone metadata should not affect original")
	}
}

// TestRuleSet tests rule set functionality
func TestRuleSet(t *testing.T) {
	t.Run("add_rule", func(t *testing.T) {
		rs := &RuleSet{
			Rules: make([]*Rule, 0),
		}

		rule := &Rule{
			ID:      "test-1",
			Name:    "Test Rule",
			Pattern: "*.go",
			Content: "Test content",
		}

		if err := rs.AddRule(rule); err != nil {
			t.Errorf("failed to add rule: %v", err)
		}

		if rs.Count() != 1 {
			t.Errorf("expected 1 rule, got %d", rs.Count())
		}
	})

	t.Run("add_duplicate_id", func(t *testing.T) {
		rs := &RuleSet{
			Rules: make([]*Rule, 0),
		}

		rule1 := &Rule{
			ID:      "test-1",
			Name:    "Test Rule 1",
			Pattern: "*.go",
			Content: "Test content",
		}

		rule2 := &Rule{
			ID:      "test-1",
			Name:    "Test Rule 2",
			Pattern: "*.js",
			Content: "Test content",
		}

		_ = rs.AddRule(rule1)
		err := rs.AddRule(rule2)

		if err == nil {
			t.Error("should fail to add duplicate ID")
		}
	})

	t.Run("remove_rule", func(t *testing.T) {
		rs := &RuleSet{
			Rules: make([]*Rule, 0),
		}

		rule := &Rule{
			ID:      "test-1",
			Name:    "Test Rule",
			Pattern: "*.go",
			Content: "Test content",
		}

		_ = rs.AddRule(rule)

		if !rs.RemoveRule("test-1") {
			t.Error("should successfully remove rule")
		}

		if rs.Count() != 0 {
			t.Error("rule should be removed")
		}
	})

	t.Run("get_rule", func(t *testing.T) {
		rs := &RuleSet{
			Rules: make([]*Rule, 0),
		}

		rule := &Rule{
			ID:      "test-1",
			Name:    "Test Rule",
			Pattern: "*.go",
			Content: "Test content",
		}

		_ = rs.AddRule(rule)

		retrieved := rs.GetRule("test-1")
		if retrieved == nil {
			t.Error("should find rule")
		}

		if retrieved.Name != "Test Rule" {
			t.Errorf("expected 'Test Rule', got '%s'", retrieved.Name)
		}
	})

	t.Run("get_matching_rules", func(t *testing.T) {
		rs := &RuleSet{
			Rules: make([]*Rule, 0),
		}

		rule1 := &Rule{
			ID:          "test-1",
			Name:        "Go Rule",
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Content:     "Go content",
			Priority:    1,
		}

		rule2 := &Rule{
			ID:          "test-2",
			Name:        "JS Rule",
			Pattern:     "*.js",
			PatternType: PatternTypeGlob,
			Content:     "JS content",
			Priority:    1,
		}

		_ = rs.AddRule(rule1)
		_ = rs.AddRule(rule2)

		matches := rs.GetMatchingRules("main.go")

		if len(matches) != 1 {
			t.Errorf("expected 1 match, got %d", len(matches))
		}

		if matches[0].Rule.Name != "Go Rule" {
			t.Error("should match Go Rule")
		}
	})

	t.Run("get_rules_by_category", func(t *testing.T) {
		rs := &RuleSet{
			Rules: make([]*Rule, 0),
		}

		rule1 := &Rule{
			ID:       "test-1",
			Name:     "Style Rule",
			Pattern:  "*.go",
			Content:  "Style content",
			Category: RuleCategoryStyle,
		}

		rule2 := &Rule{
			ID:       "test-2",
			Name:     "Security Rule",
			Pattern:  "*.go",
			Content:  "Security content",
			Category: RuleCategorySecurity,
		}

		_ = rs.AddRule(rule1)
		_ = rs.AddRule(rule2)

		styleRules := rs.GetRulesByCategory(RuleCategoryStyle)

		if len(styleRules) != 1 {
			t.Errorf("expected 1 style rule, got %d", len(styleRules))
		}

		if styleRules[0].Name != "Style Rule" {
			t.Error("should find Style Rule")
		}
	})

	t.Run("get_rules_by_tag", func(t *testing.T) {
		rs := &RuleSet{
			Rules: make([]*Rule, 0),
		}

		rule1 := &Rule{
			ID:      "test-1",
			Name:    "Rule 1",
			Pattern: "*.go",
			Content: "Content",
			Tags:    []string{"important", "style"},
		}

		rule2 := &Rule{
			ID:      "test-2",
			Name:    "Rule 2",
			Pattern: "*.js",
			Content: "Content",
			Tags:    []string{"style"},
		}

		_ = rs.AddRule(rule1)
		_ = rs.AddRule(rule2)

		importantRules := rs.GetRulesByTag("important")

		if len(importantRules) != 1 {
			t.Errorf("expected 1 rule with 'important' tag, got %d", len(importantRules))
		}
	})
}

// TestParser tests the .clinerules parser
func TestParser(t *testing.T) {
	t.Run("parse_simple", func(t *testing.T) {
		content := `
[Go Style]
pattern: *.go
Use proper Go formatting

[JavaScript Style]
pattern: *.js
Use ES6+ features
`

		ruleSet, err := ParseString(content)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if ruleSet.Count() != 2 {
			t.Errorf("expected 2 rules, got %d", ruleSet.Count())
		}

		goRule := ruleSet.GetRule("go-style")
		if goRule == nil {
			t.Error("should find 'go-style' rule")
		}

		if goRule.Pattern != "*.go" {
			t.Errorf("expected pattern '*.go', got '%s'", goRule.Pattern)
		}
	})

	t.Run("parse_with_metadata", func(t *testing.T) {
		content := `
[Security Rule]
pattern: **/*.go
description: Security best practices
priority: 10
category: security
tags: important, security, go

Never store passwords in plain text
`

		ruleSet, err := ParseString(content)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		rule := ruleSet.GetRule("security-rule")
		if rule == nil {
			t.Fatal("should find security rule")
		}

		if rule.Description != "Security best practices" {
			t.Error("should have description")
		}

		if rule.Priority != 10 {
			t.Errorf("expected priority 10, got %d", rule.Priority)
		}

		if rule.Category != RuleCategorySecurity {
			t.Error("should have security category")
		}

		if len(rule.Tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(rule.Tags))
		}
	})

	t.Run("parse_regex_pattern", func(t *testing.T) {
		content := `
[Test Files]
pattern: /.*_test\.go$/
Test files should have proper coverage
`

		ruleSet, err := ParseString(content)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		rule := ruleSet.GetRule("test-files")
		if rule == nil {
			t.Fatal("should find test files rule")
		}

		if rule.PatternType != PatternTypeRegex {
			t.Error("should detect regex pattern")
		}

		if rule.Pattern != `.*_test\.go$` {
			t.Errorf("expected regex without slashes, got '%s'", rule.Pattern)
		}
	})

	t.Run("parse_with_comments", func(t *testing.T) {
		content := `
# This is a comment
[Rule 1]
pattern: *.go
# Another comment
Content here
`

		ruleSet, err := ParseString(content)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}

		if ruleSet.Count() != 1 {
			t.Errorf("comments should be ignored, expected 1 rule, got %d", ruleSet.Count())
		}
	})
}

// TestFormat tests formatting rules back to .clinerules format
func TestFormat(t *testing.T) {
	ruleSet := &RuleSet{
		Name:  "test",
		Rules: make([]*Rule, 0),
	}

	rule := &Rule{
		ID:          "go-style",
		Name:        "Go Style",
		Description: "Go formatting rules",
		Pattern:     "*.go",
		PatternType: PatternTypeGlob,
		Content:     "Use gofmt",
		Priority:    5,
		Category:    RuleCategoryStyle,
		Tags:        []string{"go", "style"},
	}

	_ = ruleSet.AddRule(rule)

	formatted := Format(ruleSet)

	// Parse it back to verify round-trip
	parsed, err := ParseString(formatted)
	if err != nil {
		t.Fatalf("failed to parse formatted output: %v", err)
	}

	if parsed.Count() != 1 {
		t.Errorf("expected 1 rule after round-trip, got %d", parsed.Count())
	}

	parsedRule := parsed.GetRule("go-style")
	if parsedRule == nil {
		t.Fatal("should find rule after round-trip")
	}

	if parsedRule.Description != rule.Description {
		t.Error("description should match after round-trip")
	}
}

// TestManager tests the rule manager
func TestManager(t *testing.T) {
	t.Run("add_workspace_rule", func(t *testing.T) {
		manager := NewManager()

		rule := &Rule{
			ID:      "test-1",
			Name:    "Test Rule",
			Pattern: "*.go",
			Content: "Test content",
		}

		if err := manager.AddWorkspaceRule(rule); err != nil {
			t.Errorf("failed to add workspace rule: %v", err)
		}

		if manager.Count() != 1 {
			t.Errorf("expected 1 rule, got %d", manager.Count())
		}
	})

	t.Run("add_project_rule", func(t *testing.T) {
		manager := NewManager()

		rule := &Rule{
			ID:      "test-1",
			Name:    "Test Rule",
			Pattern: "*.go",
			Content: "Test content",
		}

		if err := manager.AddProjectRule(rule); err != nil {
			t.Errorf("failed to add project rule: %v", err)
		}

		if manager.Count() != 1 {
			t.Errorf("expected 1 rule, got %d", manager.Count())
		}
	})

	t.Run("inheritance_priority", func(t *testing.T) {
		manager := NewManager()

		// Add workspace rule (lowest priority)
		workspaceRule := &Rule{
			ID:          "style-1",
			Name:        "Workspace Style",
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Content:     "Workspace style guidelines",
			Priority:    1,
			Scope:       RuleScopeGlobal,
		}
		_ = manager.AddWorkspaceRule(workspaceRule)

		// Add project rule (higher priority)
		projectRule := &Rule{
			ID:          "style-2",
			Name:        "Project Style",
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Content:     "Project-specific style guidelines",
			Priority:    1,
			Scope:       RuleScopeGlobal,
		}
		_ = manager.AddProjectRule(projectRule)

		// Get rules for a Go file
		matches := manager.GetRulesForFile("main.go")

		if len(matches) != 2 {
			t.Errorf("expected 2 matches, got %d", len(matches))
		}

		// Project rule should come first (higher score)
		if matches[0].Rule.Name != "Project Style" {
			t.Errorf("project rule should have higher priority, got '%s'", matches[0].Rule.Name)
		}
	})

	t.Run("get_rules_by_category", func(t *testing.T) {
		manager := NewManager()

		rule1 := &Rule{
			ID:          "style-1",
			Name:        "Style Rule",
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Content:     "Content",
			Category:    RuleCategoryStyle,
		}

		rule2 := &Rule{
			ID:          "security-1",
			Name:        "Security Rule",
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Content:     "Content",
			Category:    RuleCategorySecurity,
		}

		_ = manager.AddWorkspaceRule(rule1)
		_ = manager.AddProjectRule(rule2)

		styleRules := manager.GetRulesByCategory(RuleCategoryStyle)

		if len(styleRules) != 1 {
			t.Errorf("expected 1 style rule, got %d", len(styleRules))
		}
	})

	t.Run("format_rules_for_file", func(t *testing.T) {
		manager := NewManager()

		rule := &Rule{
			ID:          "test-1",
			Name:        "Test Rule",
			Description: "Test description",
			Pattern:     "*.go",
			PatternType: PatternTypeGlob,
			Content:     "Test guidelines",
			Priority:    1,
		}

		_ = manager.AddWorkspaceRule(rule)

		formatted := manager.FormatRulesForFile("main.go")

		if formatted == "" {
			t.Error("should format rules for matching file")
		}

		if !strings.Contains(formatted, "Test Rule") {
			t.Error("formatted output should contain rule name")
		}

		if !strings.Contains(formatted, "Test guidelines") {
			t.Error("formatted output should contain rule content")
		}
	})

	t.Run("export_all_rules", func(t *testing.T) {
		manager := NewManager()

		rule1 := &Rule{
			ID:      "test-1",
			Name:    "Workspace Rule",
			Pattern: "*.go",
			Content: "Content",
		}

		rule2 := &Rule{
			ID:      "test-2",
			Name:    "Project Rule",
			Pattern: "*.js",
			Content: "Content",
		}

		_ = manager.AddWorkspaceRule(rule1)
		_ = manager.AddProjectRule(rule2)

		combined := manager.Export()

		if combined.Count() != 2 {
			t.Errorf("expected 2 rules in export, got %d", combined.Count())
		}
	})
}

// TestManagerFileOperations tests loading and saving rules
func TestManagerFileOperations(t *testing.T) {
	t.Run("save_and_load_workspace_rules", func(t *testing.T) {
		manager := NewManager()

		rule := &Rule{
			ID:      "test-1",
			Name:    "Test Rule",
			Pattern: "*.go",
			Content: "Test content",
		}

		_ = manager.AddWorkspaceRule(rule)

		// Save to temp file
		tmpFile := filepath.Join(os.TempDir(), "test-workspace.clinerules")
		defer os.Remove(tmpFile)

		if err := manager.SaveWorkspaceRules(tmpFile); err != nil {
			t.Fatalf("failed to save rules: %v", err)
		}

		// Load into new manager
		manager2 := NewManager()
		if err := manager2.LoadWorkspaceRules(tmpFile); err != nil {
			t.Fatalf("failed to load rules: %v", err)
		}

		if manager2.Count() != 1 {
			t.Errorf("expected 1 rule after load, got %d", manager2.Count())
		}
	})
}
