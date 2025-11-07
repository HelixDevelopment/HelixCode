package rules

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Rule represents a project rule
type Rule struct {
	ID          string            // Unique identifier
	Name        string            // Human-readable name
	Description string            // Rule description
	Pattern     string            // File pattern (glob or regex)
	PatternType PatternType       // Type of pattern
	Content     string            // Rule content/guidelines
	Priority    int               // Priority (higher = more important)
	Category    RuleCategory      // Rule category
	Scope       RuleScope         // Where rule applies
	Tags        []string          // Tags for categorization
	Metadata    map[string]string // Additional metadata
}

// PatternType defines the type of pattern matching
type PatternType string

const (
	PatternTypeGlob  PatternType = "glob"  // Glob pattern (*.go, src/**/*.js)
	PatternTypeRegex PatternType = "regex" // Regular expression
	PatternTypeExact PatternType = "exact" // Exact file path match
	PatternTypeAny   PatternType = "any"   // Matches any file
)

// RuleCategory categorizes rules by purpose
type RuleCategory string

const (
	RuleCategoryStyle       RuleCategory = "style"       // Code style rules
	RuleCategoryArchitecture RuleCategory = "architecture" // Architecture guidelines
	RuleCategorySecurity    RuleCategory = "security"    // Security requirements
	RuleCategoryTesting     RuleCategory = "testing"     // Testing guidelines
	RuleCategoryDocumentation RuleCategory = "documentation" // Documentation rules
	RuleCategoryPerformance RuleCategory = "performance" // Performance guidelines
	RuleCategoryGeneral     RuleCategory = "general"     // General guidelines
)

// RuleScope defines where a rule applies
type RuleScope string

const (
	RuleScopeGlobal    RuleScope = "global"    // Applies to entire project
	RuleScopeDirectory RuleScope = "directory" // Applies to specific directory
	RuleScopeFile      RuleScope = "file"      // Applies to specific file
)

// MatchResult represents the result of matching a rule against a file
type MatchResult struct {
	Matches  bool   // Whether the rule matches
	Score    int    // Match score (for prioritization)
	Rule     *Rule  // The matched rule
	FilePath string // The file path that was matched
}

// Matches checks if this rule matches the given file path
func (r *Rule) Matches(filePath string) bool {
	switch r.PatternType {
	case PatternTypeAny:
		return true

	case PatternTypeExact:
		return r.Pattern == filePath

	case PatternTypeGlob:
		matched, err := filepath.Match(r.Pattern, filepath.Base(filePath))
		if err != nil {
			return false
		}
		if matched {
			return true
		}

		// Try matching full path for patterns like "src/**/*.go"
		return matchGlobPath(r.Pattern, filePath)

	case PatternTypeRegex:
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return false
		}
		return re.MatchString(filePath)

	default:
		return false
	}
}

// matchGlobPath matches glob patterns with directory wildcards
func matchGlobPath(pattern, path string) bool {
	// Convert glob pattern to regex
	// ** = match any directory depth (zero or more)
	// * = match any characters except /
	// ? = match single character

	regexPattern := regexp.QuoteMeta(pattern)

	// Replace **/ with regex for zero or more directory levels
	// Use a placeholder to avoid conflicts with single *
	regexPattern = strings.ReplaceAll(regexPattern, `\*\*/`, `<<<DOUBLESTAR>>>`)

	// Replace * with regex for any non-slash characters
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, `[^/]*`)

	// Now replace the placeholder with the correct regex for **/
	// (?:.*/)? matches an optional path with trailing slash (zero or more dirs)
	regexPattern = strings.ReplaceAll(regexPattern, `<<<DOUBLESTAR>>>`, `(?:.*/)?`)

	// Replace ? with regex for single character
	regexPattern = strings.ReplaceAll(regexPattern, `\?`, `.`)

	// Anchor the pattern
	regexPattern = "^" + regexPattern + "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	return re.MatchString(path)
}

// MatchScore calculates a match score for prioritization
func (r *Rule) MatchScore(filePath string) int {
	if !r.Matches(filePath) {
		return 0
	}

	score := r.Priority * 100

	// Add bonus for more specific patterns
	switch r.PatternType {
	case PatternTypeExact:
		score += 50 // Exact match is most specific
	case PatternTypeGlob:
		// More specific globs get higher scores
		if !strings.Contains(r.Pattern, "*") {
			score += 40
		} else if strings.Count(r.Pattern, "*") == 1 {
			score += 30
		} else {
			score += 20
		}
	case PatternTypeRegex:
		score += 25
	case PatternTypeAny:
		score += 0 // Least specific
	}

	// Add bonus for more specific scopes
	switch r.Scope {
	case RuleScopeFile:
		score += 30
	case RuleScopeDirectory:
		score += 20
	case RuleScopeGlobal:
		score += 10
	}

	return score
}

// Validate validates the rule
func (r *Rule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}

	if r.Pattern == "" && r.PatternType != PatternTypeAny {
		return fmt.Errorf("rule pattern cannot be empty")
	}

	if r.Content == "" {
		return fmt.Errorf("rule content cannot be empty")
	}

	// Validate pattern based on type
	switch r.PatternType {
	case PatternTypeRegex:
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	case PatternTypeGlob:
		// Basic validation - check for valid glob syntax
		if strings.Contains(r.Pattern, "***") {
			return fmt.Errorf("invalid glob pattern: *** is not valid")
		}
	}

	return nil
}

// String returns a string representation of the rule
func (r *Rule) String() string {
	return fmt.Sprintf("%s (%s): %s", r.Name, r.Pattern, r.Description)
}

// Clone creates a deep copy of the rule
func (r *Rule) Clone() *Rule {
	tags := make([]string, len(r.Tags))
	copy(tags, r.Tags)

	metadata := make(map[string]string)
	for k, v := range r.Metadata {
		metadata[k] = v
	}

	return &Rule{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Pattern:     r.Pattern,
		PatternType: r.PatternType,
		Content:     r.Content,
		Priority:    r.Priority,
		Category:    r.Category,
		Scope:       r.Scope,
		Tags:        tags,
		Metadata:    metadata,
	}
}

// RuleSet represents a collection of rules
type RuleSet struct {
	Name        string            // Name of the rule set
	Description string            // Description
	Rules       []*Rule           // List of rules
	Metadata    map[string]string // Additional metadata
}

// AddRule adds a rule to the rule set
func (rs *RuleSet) AddRule(rule *Rule) error {
	if err := rule.Validate(); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// Check for duplicate IDs
	for _, r := range rs.Rules {
		if r.ID == rule.ID {
			return fmt.Errorf("rule with ID '%s' already exists", rule.ID)
		}
	}

	rs.Rules = append(rs.Rules, rule)
	return nil
}

// RemoveRule removes a rule by ID
func (rs *RuleSet) RemoveRule(id string) bool {
	for i, rule := range rs.Rules {
		if rule.ID == id {
			rs.Rules = append(rs.Rules[:i], rs.Rules[i+1:]...)
			return true
		}
	}
	return false
}

// GetRule retrieves a rule by ID
func (rs *RuleSet) GetRule(id string) *Rule {
	for _, rule := range rs.Rules {
		if rule.ID == id {
			return rule
		}
	}
	return nil
}

// GetMatchingRules returns all rules that match the given file path
func (rs *RuleSet) GetMatchingRules(filePath string) []*MatchResult {
	results := make([]*MatchResult, 0)

	for _, rule := range rs.Rules {
		if rule.Matches(filePath) {
			results = append(results, &MatchResult{
				Matches:  true,
				Score:    rule.MatchScore(filePath),
				Rule:     rule,
				FilePath: filePath,
			})
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// GetRulesByCategory returns all rules in a category
func (rs *RuleSet) GetRulesByCategory(category RuleCategory) []*Rule {
	results := make([]*Rule, 0)
	for _, rule := range rs.Rules {
		if rule.Category == category {
			results = append(results, rule)
		}
	}
	return results
}

// GetRulesByTag returns all rules with a specific tag
func (rs *RuleSet) GetRulesByTag(tag string) []*Rule {
	results := make([]*Rule, 0)
	for _, rule := range rs.Rules {
		for _, t := range rule.Tags {
			if t == tag {
				results = append(results, rule)
				break
			}
		}
	}
	return results
}

// Count returns the number of rules
func (rs *RuleSet) Count() int {
	return len(rs.Rules)
}

// Clear removes all rules
func (rs *RuleSet) Clear() {
	rs.Rules = make([]*Rule, 0)
}
