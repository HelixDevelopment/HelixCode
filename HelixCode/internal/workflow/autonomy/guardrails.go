package autonomy

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GuardrailsChecker enforces safety constraints
type GuardrailsChecker struct {
	mu         sync.RWMutex
	rules      []GuardrailRule
	violations *ViolationTracker
}

// GuardrailRule defines a safety constraint
type GuardrailRule struct {
	Name        string
	Description string
	Check       func(context.Context, *Action) (bool, string)
	Severity    RiskLevel
	Enabled     bool
}

// ViolationTracker records guardrail violations
type ViolationTracker struct {
	mu         sync.RWMutex
	violations []Violation
}

// Violation represents a guardrail breach
type Violation struct {
	Rule      string
	Action    *Action
	Timestamp time.Time
	Severity  RiskLevel
	Allowed   bool
	Reason    string
}

// NewGuardrailsChecker creates a checker with default rules
func NewGuardrailsChecker() *GuardrailsChecker {
	gc := &GuardrailsChecker{
		rules: make([]GuardrailRule, 0),
		violations: &ViolationTracker{
			violations: make([]Violation, 0),
		},
	}

	// Add default rules
	for _, rule := range DefaultGuardrails {
		gc.AddRule(rule)
	}

	return gc
}

// Check verifies action against all rules
func (g *GuardrailsChecker) Check(ctx context.Context, action *Action) (bool, []string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var failedRules []string
	allPassed := true

	for _, rule := range g.rules {
		if !rule.Enabled {
			continue
		}

		passed, reason := rule.Check(ctx, action)
		if !passed {
			allPassed = false
			failedRules = append(failedRules, fmt.Sprintf("%s: %s", rule.Name, reason))

			// Record violation
			g.violations.record(Violation{
				Rule:      rule.Name,
				Action:    action,
				Timestamp: time.Now(),
				Severity:  rule.Severity,
				Allowed:   false,
				Reason:    reason,
			})
		}
	}

	return allPassed, failedRules, nil
}

// AddRule adds a custom guardrail rule
func (g *GuardrailsChecker) AddRule(rule GuardrailRule) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rules = append(g.rules, rule)
}

// DisableRule disables a specific rule
func (g *GuardrailsChecker) DisableRule(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i := range g.rules {
		if g.rules[i].Name == name {
			g.rules[i].Enabled = false
			break
		}
	}
}

// EnableRule enables a specific rule
func (g *GuardrailsChecker) EnableRule(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i := range g.rules {
		if g.rules[i].Name == name {
			g.rules[i].Enabled = true
			break
		}
	}
}

// GetViolations returns recent violations
func (g *GuardrailsChecker) GetViolations() []Violation {
	return g.violations.getAll()
}

// GetRules returns all rules
func (g *GuardrailsChecker) GetRules() []GuardrailRule {
	g.mu.RLock()
	defer g.mu.RUnlock()

	rules := make([]GuardrailRule, len(g.rules))
	copy(rules, g.rules)
	return rules
}

// ViolationTracker methods

func (vt *ViolationTracker) record(v Violation) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.violations = append(vt.violations, v)

	// Keep only last 100 violations
	if len(vt.violations) > 100 {
		vt.violations = vt.violations[len(vt.violations)-100:]
	}
}

func (vt *ViolationTracker) getAll() []Violation {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	violations := make([]Violation, len(vt.violations))
	copy(violations, vt.violations)
	return violations
}

// Default guardrail rules
var DefaultGuardrails = []GuardrailRule{
	{
		Name:        "no_system_file_delete",
		Description: "Prevent deletion of system files",
		Severity:    RiskCritical,
		Enabled:     true,
		Check: func(ctx context.Context, action *Action) (bool, string) {
			if action.Type != ActionFileDelete {
				return true, ""
			}

			if action.Context == nil {
				return true, ""
			}

			// System directories and files that should not be deleted
			systemPaths := []string{
				"/etc", "/sys", "/proc", "/dev",
				"/bin", "/sbin", "/usr/bin", "/usr/sbin",
				"/boot", "/lib", "/lib64",
			}

			for _, file := range action.Context.FilesAffected {
				absPath, _ := filepath.Abs(file)
				for _, sysPath := range systemPaths {
					if strings.HasPrefix(absPath, sysPath) {
						return false, fmt.Sprintf("cannot delete system file: %s", file)
					}
				}
			}

			return true, ""
		},
	},
	{
		Name:        "no_bulk_unreviewed",
		Description: "Prevent bulk changes without review",
		Severity:    RiskHigh,
		Enabled:     true,
		Check: func(ctx context.Context, action *Action) (bool, string) {
			if action.Type != ActionBulkEdit {
				return true, ""
			}

			if action.Context == nil {
				return true, ""
			}

			bulkThreshold := 10
			if len(action.Context.FilesAffected) > bulkThreshold {
				return false, fmt.Sprintf("bulk operation affects %d files (threshold: %d)",
					len(action.Context.FilesAffected), bulkThreshold)
			}

			return true, ""
		},
	},
	{
		Name:        "no_destructive_commands",
		Description: "Prevent destructive commands",
		Severity:    RiskCritical,
		Enabled:     true,
		Check: func(ctx context.Context, action *Action) (bool, string) {
			if action.Type != ActionExecuteCmd {
				return true, ""
			}

			if action.Context == nil || action.Context.CommandToRun == "" {
				return true, ""
			}

			// Dangerous commands
			dangerous := []string{
				"rm -rf /",
				"dd if=/dev/zero",
				"mkfs.",
				"fdisk",
				":(){ :|:& };:", // Fork bomb
				"> /dev/sda",
			}

			cmd := action.Context.CommandToRun
			for _, pattern := range dangerous {
				if strings.Contains(cmd, pattern) {
					return false, fmt.Sprintf("destructive command detected: %s", pattern)
				}
			}

			return true, ""
		},
	},
	{
		Name:        "no_uncontrolled_network",
		Description: "Require approval for network operations",
		Severity:    RiskMedium,
		Enabled:     true,
		Check: func(ctx context.Context, action *Action) (bool, string) {
			if action.Type != ActionNetworkCall {
				return true, ""
			}

			// Network operations should have explicit approval in metadata
			if action.Metadata != nil {
				if approved, ok := action.Metadata["network_approved"].(bool); ok && approved {
					return true, ""
				}
			}

			return false, "network operation requires explicit approval"
		},
	},
	{
		Name:        "require_reversible_changes",
		Description: "Ensure changes can be reversed",
		Severity:    RiskMedium,
		Enabled:     true,
		Check: func(ctx context.Context, action *Action) (bool, string) {
			if action.Type != ActionApplyChange {
				return true, ""
			}

			if action.Context == nil {
				return true, ""
			}

			if !action.Context.Reversible && action.Risk != RiskNone && action.Risk != RiskLow {
				return false, "irreversible high-risk change requires manual approval"
			}

			return true, ""
		},
	},
	{
		Name:        "limit_iteration_depth",
		Description: "Prevent infinite iteration loops",
		Severity:    RiskMedium,
		Enabled:     true,
		Check: func(ctx context.Context, action *Action) (bool, string) {
			if action.Type != ActionIteration {
				return true, ""
			}

			if action.Context == nil {
				return true, ""
			}

			maxIterations := 50
			if action.Context.IterationCount > maxIterations {
				return false, fmt.Sprintf("iteration limit exceeded: %d > %d",
					action.Context.IterationCount, maxIterations)
			}

			return true, ""
		},
	},
	{
		Name:        "no_credential_exposure",
		Description: "Prevent exposure of credentials",
		Severity:    RiskCritical,
		Enabled:     true,
		Check: func(ctx context.Context, action *Action) (bool, string) {
			// Check if action might expose credentials
			sensitivePatterns := []string{
				"password", "token", "secret", "key", "credential",
				"api_key", "private_key", "ssh_key",
			}

			desc := strings.ToLower(action.Description)
			for _, pattern := range sensitivePatterns {
				if strings.Contains(desc, pattern) {
					// Check if it's explicitly marked as safe
					if action.Metadata != nil {
						if safe, ok := action.Metadata["credential_safe"].(bool); ok && safe {
							return true, ""
						}
					}
					return false, fmt.Sprintf("potential credential exposure: %s", pattern)
				}
			}

			return true, ""
		},
	},
}
