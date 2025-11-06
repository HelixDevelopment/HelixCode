package confirmation

import (
	"time"

	"github.com/google/uuid"
)

// OperationType categorizes operations
type OperationType string

const (
	OpRead       OperationType = "read"
	OpWrite      OperationType = "write"
	OpDelete     OperationType = "delete"
	OpExecute    OperationType = "execute"
	OpNetwork    OperationType = "network"
	OpFileSystem OperationType = "filesystem"
	OpGit        OperationType = "git"
)

// RiskLevel categorizes operation risk
type RiskLevel int

const (
	RiskNone RiskLevel = iota
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

// String returns string representation of RiskLevel
func (r RiskLevel) String() string {
	switch r {
	case RiskNone:
		return "none"
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Operation describes what the tool will do
type Operation struct {
	Type        OperationType
	Description string
	Target      string
	Risk        RiskLevel
	Reversible  bool
	Preview     string
}

// ExecutionContext provides context about execution
type ExecutionContext struct {
	User           string
	SessionID      string
	ConversationID string
	Timestamp      time.Time
	CI             bool // Running in CI/CD
}

// ConfirmationRequest requests confirmation for tool execution
type ConfirmationRequest struct {
	ToolName   string
	Operation  Operation
	Parameters map[string]interface{}
	Context    ExecutionContext
	BatchMode  bool
}

// Choice represents user's decision
type Choice int

const (
	ChoiceAllow Choice = iota
	ChoiceDeny
	ChoiceAlways
	ChoiceNever
	ChoiceAsk
)

// String returns string representation of Choice
func (c Choice) String() string {
	switch c {
	case ChoiceAllow:
		return "allow"
	case ChoiceDeny:
		return "deny"
	case ChoiceAlways:
		return "always"
	case ChoiceNever:
		return "never"
	case ChoiceAsk:
		return "ask"
	default:
		return "unknown"
	}
}

// ConfirmationResult contains the decision
type ConfirmationResult struct {
	Allowed   bool
	Reason    string
	Choice    Choice
	Policy    *Policy
	Timestamp time.Time
	AuditID   string
}

// Action defines what to do
type Action int

const (
	ActionAllow Action = iota
	ActionDeny
	ActionAsk
)

// String returns string representation of Action
func (a Action) String() string {
	switch a {
	case ActionAllow:
		return "allow"
	case ActionDeny:
		return "deny"
	case ActionAsk:
		return "ask"
	default:
		return "unknown"
	}
}

// ConfirmationLevel defines urgency
type ConfirmationLevel int

const (
	LevelInfo ConfirmationLevel = iota
	LevelWarning
	LevelDanger
)

// String returns string representation of ConfirmationLevel
func (l ConfirmationLevel) String() string {
	switch l {
	case LevelInfo:
		return "info"
	case LevelWarning:
		return "warning"
	case LevelDanger:
		return "danger"
	default:
		return "unknown"
	}
}

// AuditEntry represents a logged decision
type AuditEntry struct {
	ID             string
	Timestamp      time.Time
	User           string
	SessionID      string
	ConversationID string
	ToolName       string
	Operation      Operation
	Decision       Choice
	Policy         string
	Rule           string
	Reason         string
}

// AuditQuery filters audit entries
type AuditQuery struct {
	User      string
	Tool      string
	StartTime time.Time
	EndTime   time.Time
	Decision  *Choice
	Limit     int
}

// Config represents confirmation configuration
type Config struct {
	Enabled       bool
	DefaultPolicy *Policy
	AuditPath     string
	BatchMode     bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:       true,
		DefaultPolicy: DefaultPolicy(),
		AuditPath:     ".helix/audit/confirmations.jsonl",
		BatchMode:     false,
	}
}

// Option is a functional option for ConfirmationCoordinator
type Option func(*ConfirmationCoordinator)

// WithPrompter sets a custom prompter
func WithPrompter(p Prompter) Option {
	return func(cc *ConfirmationCoordinator) {
		cc.promptManager.prompter = p
	}
}

// WithAuditPath sets a custom audit log path
func WithAuditPath(path string) Option {
	return func(cc *ConfirmationCoordinator) {
		cc.config.AuditPath = path
	}
}

// WithBatchMode enables batch mode
func WithBatchMode(enabled bool) Option {
	return func(cc *ConfirmationCoordinator) {
		cc.config.BatchMode = enabled
	}
}

// WithConfig sets a custom config
func WithConfig(cfg *Config) Option {
	return func(cc *ConfirmationCoordinator) {
		cc.config = cfg
	}
}

// GenerateAuditID generates a unique audit ID
func GenerateAuditID() string {
	return uuid.New().String()
}
