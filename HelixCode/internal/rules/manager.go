package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Manager manages rules with inheritance support
type Manager struct {
	workspaceRules *RuleSet // Workspace-level rules
	projectRules   *RuleSet // Project-level rules
	fileRules      map[string]*RuleSet // File-specific rules
	mu             sync.RWMutex
}

// NewManager creates a new rule manager
func NewManager() *Manager {
	return &Manager{
		workspaceRules: &RuleSet{
			Name:  "workspace",
			Rules: make([]*Rule, 0),
		},
		projectRules: &RuleSet{
			Name:  "project",
			Rules: make([]*Rule, 0),
		},
		fileRules: make(map[string]*RuleSet),
	}
}

// LoadFromDirectory loads rules from standard locations
func (m *Manager) LoadFromDirectory(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to load workspace rules (.helix/.clinerules)
	workspacePath := filepath.Join(dir, ".helix", ".clinerules")
	if _, err := os.Stat(workspacePath); err == nil {
		parser := NewParser(workspacePath)
		ruleSet, err := parser.Parse()
		if err != nil {
			return fmt.Errorf("failed to load workspace rules: %w", err)
		}
		m.workspaceRules = ruleSet
		m.workspaceRules.Name = "workspace"
	}

	// Try to load project rules (.clinerules in project root)
	projectPath := filepath.Join(dir, ".clinerules")
	if _, err := os.Stat(projectPath); err == nil {
		parser := NewParser(projectPath)
		ruleSet, err := parser.Parse()
		if err != nil {
			return fmt.Errorf("failed to load project rules: %w", err)
		}
		m.projectRules = ruleSet
		m.projectRules.Name = "project"
	}

	// Scan for file-specific rules (*.clinerules)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			return filepath.SkipDir
		}

		// Check for .clinerules files (but not the main ones)
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".clinerules") {
			if path != workspacePath && path != projectPath {
				parser := NewParser(path)
				ruleSet, err := parser.Parse()
				if err != nil {
					// Log error but continue
					fmt.Printf("Warning: failed to load rules from %s: %v\n", path, err)
					return nil
				}

				// Store relative to project directory
				relPath, _ := filepath.Rel(dir, path)
				m.fileRules[relPath] = ruleSet
			}
		}

		return nil
	})

	return err
}

// LoadWorkspaceRules loads workspace-level rules
func (m *Manager) LoadWorkspaceRules(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	parser := NewParser(path)
	ruleSet, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to load workspace rules: %w", err)
	}

	m.workspaceRules = ruleSet
	m.workspaceRules.Name = "workspace"
	return nil
}

// LoadProjectRules loads project-level rules
func (m *Manager) LoadProjectRules(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	parser := NewParser(path)
	ruleSet, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to load project rules: %w", err)
	}

	m.projectRules = ruleSet
	m.projectRules.Name = "project"
	return nil
}

// LoadFileRules loads file-specific rules
func (m *Manager) LoadFileRules(fileKey, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	parser := NewParser(path)
	ruleSet, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to load file rules: %w", err)
	}

	m.fileRules[fileKey] = ruleSet
	return nil
}

// GetRulesForFile gets all applicable rules for a file with inheritance
// Priority order: file-specific > project > workspace
func (m *Manager) GetRulesForFile(filePath string) []*MatchResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allMatches := make([]*MatchResult, 0)

	// Collect matches from all levels
	// 1. Workspace rules (lowest priority)
	workspaceMatches := m.workspaceRules.GetMatchingRules(filePath)
	for _, match := range workspaceMatches {
		match.Score += 1000 // Base score for workspace
		allMatches = append(allMatches, match)
	}

	// 2. Project rules (medium priority)
	projectMatches := m.projectRules.GetMatchingRules(filePath)
	for _, match := range projectMatches {
		match.Score += 2000 // Higher score for project
		allMatches = append(allMatches, match)
	}

	// 3. File-specific rules (highest priority)
	for _, fileRuleSet := range m.fileRules {
		fileMatches := fileRuleSet.GetMatchingRules(filePath)
		for _, match := range fileMatches {
			match.Score += 3000 // Highest score for file-specific
			allMatches = append(allMatches, match)
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(allMatches)-1; i++ {
		for j := i + 1; j < len(allMatches); j++ {
			if allMatches[j].Score > allMatches[i].Score {
				allMatches[i], allMatches[j] = allMatches[j], allMatches[i]
			}
		}
	}

	return allMatches
}

// GetAllRules returns all rules from all levels
func (m *Manager) GetAllRules() []*Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allRules := make([]*Rule, 0)

	// Workspace rules
	allRules = append(allRules, m.workspaceRules.Rules...)

	// Project rules
	allRules = append(allRules, m.projectRules.Rules...)

	// File rules
	for _, ruleSet := range m.fileRules {
		allRules = append(allRules, ruleSet.Rules...)
	}

	return allRules
}

// GetRulesByCategory returns rules from all levels for a category
func (m *Manager) GetRulesByCategory(category RuleCategory) []*Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*Rule, 0)

	// Collect from all levels
	rules = append(rules, m.workspaceRules.GetRulesByCategory(category)...)
	rules = append(rules, m.projectRules.GetRulesByCategory(category)...)

	for _, ruleSet := range m.fileRules {
		rules = append(rules, ruleSet.GetRulesByCategory(category)...)
	}

	return rules
}

// GetRulesByTag returns rules from all levels with a specific tag
func (m *Manager) GetRulesByTag(tag string) []*Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*Rule, 0)

	rules = append(rules, m.workspaceRules.GetRulesByTag(tag)...)
	rules = append(rules, m.projectRules.GetRulesByTag(tag)...)

	for _, ruleSet := range m.fileRules {
		rules = append(rules, ruleSet.GetRulesByTag(tag)...)
	}

	return rules
}

// FormatRulesForFile formats all applicable rules for a file as a prompt
func (m *Manager) FormatRulesForFile(filePath string) string {
	matches := m.GetRulesForFile(filePath)

	if len(matches) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("# Project Rules\n\n")
	builder.WriteString("Please follow these rules when making changes:\n\n")

	for i, match := range matches {
		builder.WriteString(fmt.Sprintf("## Rule %d: %s\n", i+1, match.Rule.Name))

		if match.Rule.Description != "" {
			builder.WriteString(fmt.Sprintf("**Description:** %s\n", match.Rule.Description))
		}

		if match.Rule.Pattern != "*" {
			builder.WriteString(fmt.Sprintf("**Applies to:** `%s`\n", match.Rule.Pattern))
		}

		builder.WriteString("\n")
		builder.WriteString(match.Rule.Content)
		builder.WriteString("\n\n")
		builder.WriteString("---\n\n")
	}

	return builder.String()
}

// AddWorkspaceRule adds a rule to workspace level
func (m *Manager) AddWorkspaceRule(rule *Rule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.workspaceRules.AddRule(rule)
}

// AddProjectRule adds a rule to project level
func (m *Manager) AddProjectRule(rule *Rule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.projectRules.AddRule(rule)
}

// RemoveRule removes a rule by ID from all levels
func (m *Manager) RemoveRule(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try workspace
	if m.workspaceRules.RemoveRule(id) {
		return true
	}

	// Try project
	if m.projectRules.RemoveRule(id) {
		return true
	}

	// Try file rules
	for _, ruleSet := range m.fileRules {
		if ruleSet.RemoveRule(id) {
			return true
		}
	}

	return false
}

// Clear clears all rules
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workspaceRules.Clear()
	m.projectRules.Clear()
	m.fileRules = make(map[string]*RuleSet)
}

// Count returns total number of rules across all levels
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := m.workspaceRules.Count() + m.projectRules.Count()
	for _, ruleSet := range m.fileRules {
		count += ruleSet.Count()
	}

	return count
}

// Export exports all rules to a single RuleSet
func (m *Manager) Export() *RuleSet {
	m.mu.RLock()
	defer m.mu.RUnlock()

	combined := &RuleSet{
		Name:     "combined",
		Rules:    make([]*Rule, 0),
		Metadata: make(map[string]string),
	}

	// Add all rules
	combined.Rules = append(combined.Rules, m.workspaceRules.Rules...)
	combined.Rules = append(combined.Rules, m.projectRules.Rules...)

	for _, ruleSet := range m.fileRules {
		combined.Rules = append(combined.Rules, ruleSet.Rules...)
	}

	return combined
}

// SaveWorkspaceRules saves workspace rules to a file
func (m *Manager) SaveWorkspaceRules(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	content := Format(m.workspaceRules)
	return os.WriteFile(path, []byte(content), 0644)
}

// SaveProjectRules saves project rules to a file
func (m *Manager) SaveProjectRules(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	content := Format(m.projectRules)
	return os.WriteFile(path, []byte(content), 0644)
}
