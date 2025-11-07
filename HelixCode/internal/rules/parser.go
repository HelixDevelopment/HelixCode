package rules

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Parser parses .clinerules files
type Parser struct {
	filePath string
}

// NewParser creates a new rules parser
func NewParser(filePath string) *Parser {
	return &Parser{
		filePath: filePath,
	}
}

// Parse parses the .clinerules file and returns a RuleSet
func (p *Parser) Parse() (*RuleSet, error) {
	file, err := os.Open(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rules file: %w", err)
	}
	defer file.Close()

	ruleSet := &RuleSet{
		Name:     filepath.Base(p.filePath),
		Rules:    make([]*Rule, 0),
		Metadata: make(map[string]string),
	}

	scanner := bufio.NewScanner(file)
	var currentRule *Rule
	var contentBuilder strings.Builder
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for rule start (pattern: [rule_name])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// Save previous rule if exists
			if currentRule != nil {
				currentRule.Content = strings.TrimSpace(contentBuilder.String())
				if err := ruleSet.AddRule(currentRule); err != nil {
					return nil, fmt.Errorf("line %d: %w", lineNum, err)
				}
				contentBuilder.Reset()
			}

			// Start new rule
			ruleName := strings.Trim(trimmed, "[]")
			currentRule = &Rule{
				ID:          generateRuleID(ruleName),
				Name:        ruleName,
				Pattern:     "*", // Default pattern
				PatternType: PatternTypeGlob,
				Priority:    1,
				Category:    RuleCategoryGeneral,
				Scope:       RuleScopeGlobal,
				Tags:        make([]string, 0),
				Metadata:    make(map[string]string),
			}
			continue
		}

		// Check for metadata (key: value)
		if strings.Contains(trimmed, ":") && currentRule != nil {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				value := strings.TrimSpace(parts[1])

				switch key {
				case "pattern":
					currentRule.Pattern = value
					// Auto-detect pattern type
					if strings.HasPrefix(value, "/") && strings.HasSuffix(value, "/") {
						currentRule.PatternType = PatternTypeRegex
						currentRule.Pattern = strings.Trim(value, "/")
					} else if value == "*" || value == "**" {
						currentRule.PatternType = PatternTypeAny
					} else if strings.Contains(value, "*") || strings.Contains(value, "?") {
						currentRule.PatternType = PatternTypeGlob
					} else {
						currentRule.PatternType = PatternTypeExact
					}

				case "description":
					currentRule.Description = value

				case "priority":
					if priority, err := parseint(value); err == nil {
						currentRule.Priority = priority
					}

				case "category":
					currentRule.Category = RuleCategory(value)

				case "scope":
					currentRule.Scope = RuleScope(value)

				case "tags":
					currentRule.Tags = parseTags(value)

				default:
					// Store as metadata
					currentRule.Metadata[key] = value
				}
				continue
			}
		}

		// Otherwise, it's rule content
		if currentRule != nil {
			if contentBuilder.Len() > 0 {
				contentBuilder.WriteString("\n")
			}
			contentBuilder.WriteString(line)
		}
	}

	// Save last rule
	if currentRule != nil {
		currentRule.Content = strings.TrimSpace(contentBuilder.String())
		if err := ruleSet.AddRule(currentRule); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return ruleSet, nil
}

// ParseString parses rules from a string
func ParseString(content string) (*RuleSet, error) {
	ruleSet := &RuleSet{
		Name:     "inline",
		Rules:    make([]*Rule, 0),
		Metadata: make(map[string]string),
	}

	var currentRule *Rule
	var contentBuilder strings.Builder
	lineNum := 0

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		lineNum++
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for rule start
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// Save previous rule
			if currentRule != nil {
				currentRule.Content = strings.TrimSpace(contentBuilder.String())
				if err := ruleSet.AddRule(currentRule); err != nil {
					return nil, fmt.Errorf("line %d: %w", lineNum, err)
				}
				contentBuilder.Reset()
			}

			// Start new rule
			ruleName := strings.Trim(trimmed, "[]")
			currentRule = &Rule{
				ID:          generateRuleID(ruleName),
				Name:        ruleName,
				Pattern:     "*",
				PatternType: PatternTypeGlob,
				Priority:    1,
				Category:    RuleCategoryGeneral,
				Scope:       RuleScopeGlobal,
				Tags:        make([]string, 0),
				Metadata:    make(map[string]string),
			}
			continue
		}

		// Check for metadata
		if strings.Contains(trimmed, ":") && currentRule != nil {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				value := strings.TrimSpace(parts[1])

				switch key {
				case "pattern":
					currentRule.Pattern = value
					if strings.HasPrefix(value, "/") && strings.HasSuffix(value, "/") {
						currentRule.PatternType = PatternTypeRegex
						currentRule.Pattern = strings.Trim(value, "/")
					} else if value == "*" || value == "**" {
						currentRule.PatternType = PatternTypeAny
					} else if strings.Contains(value, "*") || strings.Contains(value, "?") {
						currentRule.PatternType = PatternTypeGlob
					} else {
						currentRule.PatternType = PatternTypeExact
					}

				case "description":
					currentRule.Description = value

				case "priority":
					if priority, err := parseint(value); err == nil {
						currentRule.Priority = priority
					}

				case "category":
					currentRule.Category = RuleCategory(value)

				case "scope":
					currentRule.Scope = RuleScope(value)

				case "tags":
					currentRule.Tags = parseTags(value)

				default:
					currentRule.Metadata[key] = value
				}
				continue
			}
		}

		// Rule content
		if currentRule != nil {
			if contentBuilder.Len() > 0 {
				contentBuilder.WriteString("\n")
			}
			contentBuilder.WriteString(line)
		}
	}

	// Save last rule
	if currentRule != nil {
		currentRule.Content = strings.TrimSpace(contentBuilder.String())
		if err := ruleSet.AddRule(currentRule); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}

	return ruleSet, nil
}

// generateRuleID generates a unique ID from a rule name
func generateRuleID(name string) string {
	// Convert to lowercase and replace spaces with hyphens
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove non-alphanumeric characters (except hyphens)
	var builder strings.Builder
	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			builder.WriteRune(ch)
		}
	}
	return builder.String()
}

// parseTags parses a comma-separated list of tags
func parseTags(tagString string) []string {
	if tagString == "" {
		return []string{}
	}

	tags := strings.Split(tagString, ",")
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseint parses an integer from a string
func parseint(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// Format formats a RuleSet back to .clinerules format
func Format(ruleSet *RuleSet) string {
	var builder strings.Builder

	// Write header comment
	if ruleSet.Description != "" {
		builder.WriteString("# ")
		builder.WriteString(ruleSet.Description)
		builder.WriteString("\n\n")
	}

	// Write each rule
	for i, rule := range ruleSet.Rules {
		if i > 0 {
			builder.WriteString("\n")
		}

		// Rule name
		builder.WriteString("[")
		builder.WriteString(rule.Name)
		builder.WriteString("]\n")

		// Metadata
		if rule.Pattern != "*" {
			builder.WriteString("pattern: ")
			// Add regex delimiters if regex pattern
			if rule.PatternType == PatternTypeRegex {
				builder.WriteString("/")
				builder.WriteString(rule.Pattern)
				builder.WriteString("/")
			} else {
				builder.WriteString(rule.Pattern)
			}
			builder.WriteString("\n")
		}

		if rule.Description != "" {
			builder.WriteString("description: ")
			builder.WriteString(rule.Description)
			builder.WriteString("\n")
		}

		if rule.Priority != 1 {
			builder.WriteString(fmt.Sprintf("priority: %d\n", rule.Priority))
		}

		if rule.Category != RuleCategoryGeneral {
			builder.WriteString("category: ")
			builder.WriteString(string(rule.Category))
			builder.WriteString("\n")
		}

		if rule.Scope != RuleScopeGlobal {
			builder.WriteString("scope: ")
			builder.WriteString(string(rule.Scope))
			builder.WriteString("\n")
		}

		if len(rule.Tags) > 0 {
			builder.WriteString("tags: ")
			builder.WriteString(strings.Join(rule.Tags, ", "))
			builder.WriteString("\n")
		}

		// Custom metadata
		for key, value := range rule.Metadata {
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(value)
			builder.WriteString("\n")
		}

		// Rule content
		if rule.Content != "" {
			builder.WriteString("\n")
			builder.WriteString(rule.Content)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}
