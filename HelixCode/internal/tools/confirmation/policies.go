package confirmation

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Policy defines confirmation policy
type Policy struct {
	Name               string
	Description        string
	Rules              []Rule
	DefaultAction      Action
	BatchDefaultAction Action
	Enabled            bool
}

// Rule defines a policy rule
type Rule struct {
	Name      string
	Priority  int
	Condition Condition
	Action    Action
	Level     ConfirmationLevel
}

// Condition defines matching criteria
type Condition struct {
	ToolName      string
	OperationType []OperationType
	RiskLevel     []RiskLevel
	PathPattern   string
	Custom        func(ConfirmationRequest) bool
}

// Matches checks if condition matches request
func (c Condition) Matches(req ConfirmationRequest) bool {
	// Match tool name
	if c.ToolName != "" && c.ToolName != req.ToolName {
		return false
	}

	// Match operation type
	if len(c.OperationType) > 0 {
		matched := false
		for _, op := range c.OperationType {
			if op == req.Operation.Type {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Match risk level
	if len(c.RiskLevel) > 0 {
		matched := false
		for _, risk := range c.RiskLevel {
			if risk == req.Operation.Risk {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Match path pattern
	if c.PathPattern != "" {
		if matched, _ := filepath.Match(c.PathPattern, req.Operation.Target); !matched {
			return false
		}
	}

	// Custom condition
	if c.Custom != nil {
		return c.Custom(req)
	}

	return true
}

// PolicyDecision contains policy evaluation result
type PolicyDecision struct {
	Action    Action
	Rule      *Rule
	Policy    *Policy
	MatchedBy string
}

// PolicyEngine evaluates policies
type PolicyEngine struct {
	mu       sync.RWMutex
	policies map[string]*Policy
	defaults *Policy
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		policies: make(map[string]*Policy),
		defaults: DefaultPolicy(),
	}
}

// Evaluate evaluates a confirmation request against policies
func (pe *PolicyEngine) Evaluate(req ConfirmationRequest) (*PolicyDecision, error) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// Get policy for tool
	policy := pe.policies[req.ToolName]
	if policy == nil {
		policy = pe.defaults
	}

	if !policy.Enabled {
		return &PolicyDecision{
			Action:    ActionAllow,
			Policy:    policy,
			MatchedBy: "disabled",
		}, nil
	}

	// Sort rules by priority (highest first)
	rules := make([]Rule, len(policy.Rules))
	copy(rules, policy.Rules)
	sortRulesByPriority(rules)

	// Evaluate rules
	for _, rule := range rules {
		if rule.Condition.Matches(req) {
			return &PolicyDecision{
				Action:    rule.Action,
				Rule:      &rule,
				Policy:    policy,
				MatchedBy: rule.Name,
			}, nil
		}
	}

	// Default action
	action := policy.DefaultAction
	if req.BatchMode && action == ActionAsk {
		action = policy.BatchDefaultAction
	}

	return &PolicyDecision{
		Action:    action,
		Policy:    policy,
		MatchedBy: "default",
	}, nil
}

// SetPolicy sets a policy for a tool
func (pe *PolicyEngine) SetPolicy(toolName string, policy *Policy) error {
	if err := ValidatePolicy(policy); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.policies[toolName] = policy
	return nil
}

// GetPolicy retrieves a policy for a tool
func (pe *PolicyEngine) GetPolicy(toolName string) (*Policy, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	policy, ok := pe.policies[toolName]
	return policy, ok
}

// RemovePolicy removes a policy for a tool
func (pe *PolicyEngine) RemovePolicy(toolName string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	delete(pe.policies, toolName)
}

// DefaultPolicy returns the default policy
func DefaultPolicy() *Policy {
	return &Policy{
		Name:               "default",
		Description:        "Default confirmation policy",
		DefaultAction:      ActionAsk,
		BatchDefaultAction: ActionDeny,
		Enabled:            true,
		Rules: []Rule{
			{
				Name:     "allow_safe_reads",
				Priority: 10,
				Condition: Condition{
					OperationType: []OperationType{OpRead},
					RiskLevel:     []RiskLevel{RiskNone, RiskLow},
				},
				Action: ActionAllow,
				Level:  LevelInfo,
			},
			{
				Name:     "warn_writes",
				Priority: 9,
				Condition: Condition{
					OperationType: []OperationType{OpWrite},
				},
				Action: ActionAsk,
				Level:  LevelWarning,
			},
			{
				Name:     "danger_deletes",
				Priority: 8,
				Condition: Condition{
					OperationType: []OperationType{OpDelete},
				},
				Action: ActionAsk,
				Level:  LevelDanger,
			},
			{
				Name:     "critical_operations",
				Priority: 11,
				Condition: Condition{
					RiskLevel: []RiskLevel{RiskCritical},
				},
				Action: ActionAsk,
				Level:  LevelDanger,
			},
		},
	}
}

// ValidatePolicy ensures policy is safe
func ValidatePolicy(policy *Policy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	// Check for conflicting rules
	priorities := make(map[int]string)
	for _, rule := range policy.Rules {
		if existing, ok := priorities[rule.Priority]; ok {
			return fmt.Errorf("rules %s and %s have same priority %d", rule.Name, existing, rule.Priority)
		}
		priorities[rule.Priority] = rule.Name
	}

	// Ensure at least one rule or default action
	if len(policy.Rules) == 0 && policy.DefaultAction == 0 {
		return fmt.Errorf("policy must have rules or default action")
	}

	return nil
}

// sortRulesByPriority sorts rules by priority (highest first)
func sortRulesByPriority(rules []Rule) {
	for i := 0; i < len(rules); i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

// BashPolicy returns a policy for bash tool
func BashPolicy() *Policy {
	return &Policy{
		Name:               "bash",
		Description:        "Policy for bash tool execution",
		DefaultAction:      ActionAsk,
		BatchDefaultAction: ActionDeny,
		Enabled:            true,
		Rules: []Rule{
			{
				Name:     "block_system_paths",
				Priority: 15,
				Condition: Condition{
					Custom: func(req ConfirmationRequest) bool {
						systemPaths := []string{"/etc/", "/sys/", "/bin/", "/usr/bin/", "/sbin/"}
						for _, path := range systemPaths {
							if strings.HasPrefix(req.Operation.Target, path) {
								return true
							}
						}
						return false
					},
				},
				Action: ActionDeny,
				Level:  LevelDanger,
			},
			{
				Name:     "allow_safe_reads",
				Priority: 10,
				Condition: Condition{
					OperationType: []OperationType{OpRead},
					RiskLevel:     []RiskLevel{RiskNone, RiskLow},
				},
				Action: ActionAllow,
				Level:  LevelInfo,
			},
			{
				Name:     "warn_deletes",
				Priority: 12,
				Condition: Condition{
					OperationType: []OperationType{OpDelete},
				},
				Action: ActionAsk,
				Level:  LevelDanger,
			},
		},
	}
}

// GitPolicy returns a policy for git tool
func GitPolicy() *Policy {
	return &Policy{
		Name:               "git",
		Description:        "Policy for git operations",
		DefaultAction:      ActionAsk,
		BatchDefaultAction: ActionDeny,
		Enabled:            true,
		Rules: []Rule{
			{
				Name:     "warn_force_push",
				Priority: 15,
				Condition: Condition{
					Custom: func(req ConfirmationRequest) bool {
						if cmd, ok := req.Parameters["command"].(string); ok {
							return strings.Contains(cmd, "push") && (strings.Contains(cmd, "--force") || strings.Contains(cmd, "-f"))
						}
						return false
					},
				},
				Action: ActionAsk,
				Level:  LevelDanger,
			},
			{
				Name:     "warn_main_branch",
				Priority: 10,
				Condition: Condition{
					Custom: func(req ConfirmationRequest) bool {
						if branch, ok := req.Parameters["branch"].(string); ok {
							return branch == "main" || branch == "master"
						}
						return false
					},
				},
				Action: ActionAsk,
				Level:  LevelWarning,
			},
			{
				Name:     "allow_safe_operations",
				Priority: 5,
				Condition: Condition{
					OperationType: []OperationType{OpRead},
				},
				Action: ActionAllow,
				Level:  LevelInfo,
			},
		},
	}
}
