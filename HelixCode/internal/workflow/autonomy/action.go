package autonomy

import "time"

// Action represents an operation requiring permission
type Action struct {
	Type        ActionType
	Description string
	Risk        RiskLevel
	Context     *ActionContext
	Metadata    map[string]interface{}
}

// ActionType categorizes actions
type ActionType string

const (
	ActionLoadContext  ActionType = "load_context"
	ActionApplyChange  ActionType = "apply_change"
	ActionExecuteCmd   ActionType = "execute_command"
	ActionDebugRetry   ActionType = "debug_retry"
	ActionFileDelete   ActionType = "file_delete"
	ActionBulkEdit     ActionType = "bulk_edit"
	ActionNetworkCall  ActionType = "network_call"
	ActionSystemChange ActionType = "system_change"
	ActionIteration    ActionType = "iteration"
)

// RiskLevel categorizes action risk
type RiskLevel string

const (
	RiskNone     RiskLevel = "none"     // No risk
	RiskLow      RiskLevel = "low"      // Low risk, easily reversible
	RiskMedium   RiskLevel = "medium"   // Medium risk, may need effort to reverse
	RiskHigh     RiskLevel = "high"     // High risk, difficult to reverse
	RiskCritical RiskLevel = "critical" // Critical risk, potentially destructive
)

// ActionContext provides context for permission decisions
type ActionContext struct {
	TaskID          string
	StepNumber      int
	FilesAffected   []string
	CommandToRun    string
	ExpectedOutcome string
	Reversible      bool
	IterationCount  int
}

// Permission represents the result of a permission check
type Permission struct {
	Granted         bool
	Reason          string
	RequiresConfirm bool
	ConfirmPrompt   string
	Conditions      []Condition
	ExpiresAt       time.Time
}

// Condition is a requirement for permission
type Condition struct {
	Type        ConditionType
	Description string
	Met         bool
}

// ConditionType categorizes permission conditions
type ConditionType string

const (
	ConditionUserConfirm  ConditionType = "user_confirm"
	ConditionBackupExists ConditionType = "backup_exists"
	ConditionTestsPass    ConditionType = "tests_pass"
	ConditionReviewable   ConditionType = "reviewable"
)

// ActionResult contains execution results
type ActionResult struct {
	Success     bool
	Action      *Action
	Output      string
	Error       error
	Duration    time.Duration
	Retries     int
	Confirmed   bool
	Escalated   bool
	IterationNo int
}

// CodeChange represents a code modification
type CodeChange struct {
	FilePath    string
	OldContent  string
	NewContent  string
	Description string
	Reversible  bool
}

// NewAction creates a new action with the given type
func NewAction(actionType ActionType, description string, risk RiskLevel) *Action {
	return &Action{
		Type:        actionType,
		Description: description,
		Risk:        risk,
		Context:     &ActionContext{},
		Metadata:    make(map[string]interface{}),
	}
}

// WithContext adds context to the action
func (a *Action) WithContext(ctx *ActionContext) *Action {
	a.Context = ctx
	return a
}

// WithMetadata adds metadata to the action
func (a *Action) WithMetadata(key string, value interface{}) *Action {
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
	a.Metadata[key] = value
	return a
}

// IsRisky returns true if the action is considered risky
func (a *Action) IsRisky() bool {
	return a.Risk == RiskHigh || a.Risk == RiskCritical
}

// IsBulk returns true if the action affects multiple files
func (a *Action) IsBulk(threshold int) bool {
	if a.Context == nil {
		return false
	}
	return len(a.Context.FilesAffected) >= threshold
}

// IsDestructive returns true if the action is potentially destructive
func (a *Action) IsDestructive() bool {
	return a.Type == ActionFileDelete ||
		a.Type == ActionSystemChange ||
		(a.Risk == RiskCritical)
}
