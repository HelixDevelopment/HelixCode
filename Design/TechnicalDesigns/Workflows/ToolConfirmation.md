# Tool Confirmation System - Technical Design

## Overview

An interactive confirmation system that provides safety controls for tool execution through policy-based approval, user prompts, and audit logging. Supports different confirmation levels and batch mode for CI/CD environments.

## Architecture

### Component Diagram

```
┌────────────────────────────────────────────────────────────┐
│              ConfirmationCoordinator                        │
│  - Orchestrates confirmation workflow                       │
│  - Enforces policies                                        │
└────────────┬───────────────────────────────┬───────────────┘
             │                               │
             ▼                               ▼
┌────────────────────────┐      ┌────────────────────────┐
│    PolicyEngine        │      │   PromptManager        │
│  - Evaluate policies   │      │  - User prompts        │
│  - Rule matching       │      │  - Response handling   │
└──────────┬─────────────┘      └────────────────────────┘
           │
           ├──────────────┬────────────────┐
           ▼              ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│DangerDetector│  │ RuleEvaluator│  │AuditLogger   │
│- Risk assess │  │- Match rules │  │- Log actions │
└──────────────┘  └──────────────┘  └──────────────┘
```

### Core Components

#### 1. ConfirmationCoordinator

```go
package confirmation

import (
    "context"
    "sync"
)

// ConfirmationCoordinator manages confirmation workflow
type ConfirmationCoordinator struct {
    policyEngine  *PolicyEngine
    promptManager *PromptManager
    auditLogger   *AuditLogger
    dangerDetector *DangerDetector
    config        *Config

    mu            sync.RWMutex
    userChoices   map[string]Choice // Tool -> permanent choice
}

// NewConfirmationCoordinator creates a new coordinator
func NewConfirmationCoordinator(opts ...Option) *ConfirmationCoordinator {
    cc := &ConfirmationCoordinator{
        policyEngine:   NewPolicyEngine(),
        promptManager:  NewPromptManager(),
        auditLogger:    NewAuditLogger(),
        dangerDetector: NewDangerDetector(),
        config:         DefaultConfig(),
        userChoices:    make(map[string]Choice),
    }

    for _, opt := range opts {
        opt(cc)
    }

    return cc
}

// Confirm checks if tool execution should be allowed
func (cc *ConfirmationCoordinator) Confirm(ctx context.Context, req ConfirmationRequest) (*ConfirmationResult, error)

// GetPolicy retrieves the policy for a tool
func (cc *ConfirmationCoordinator) GetPolicy(toolName string) (*Policy, error)

// SetPolicy updates the policy for a tool
func (cc *ConfirmationCoordinator) SetPolicy(toolName string, policy *Policy) error

// ResetChoices clears all permanent user choices
func (cc *ConfirmationCoordinator) ResetChoices()
```

#### 2. ConfirmationRequest & Result

```go
// ConfirmationRequest requests confirmation for tool execution
type ConfirmationRequest struct {
    ToolName    string
    Operation   Operation
    Parameters  map[string]interface{}
    Context     ExecutionContext
    BatchMode   bool
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

// ExecutionContext provides context about execution
type ExecutionContext struct {
    User          string
    SessionID     string
    ConversationID string
    Timestamp     time.Time
    CI            bool // Running in CI/CD
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

// Choice represents user's decision
type Choice int

const (
    ChoiceAllow Choice = iota
    ChoiceDeny
    ChoiceAlways
    ChoiceNever
    ChoiceAsk
)
```

#### 3. PolicyEngine

```go
// PolicyEngine evaluates policies
type PolicyEngine struct {
    mu       sync.RWMutex
    policies map[string]*Policy
    defaults *Policy
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

    // Evaluate rules
    for _, rule := range policy.Rules {
        if rule.Matches(req) {
            return &PolicyDecision{
                Action:    rule.Action,
                Rule:      rule,
                Policy:    policy,
                MatchedBy: rule.Name,
            }, nil
        }
    }

    // Default action
    return &PolicyDecision{
        Action:    policy.DefaultAction,
        Policy:    policy,
        MatchedBy: "default",
    }, nil
}

// Policy defines confirmation policy
type Policy struct {
    Name          string
    Description   string
    Rules         []Rule
    DefaultAction Action
    Enabled       bool
}

// Rule defines a policy rule
type Rule struct {
    Name        string
    Priority    int
    Condition   Condition
    Action      Action
    Level       ConfirmationLevel
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

// Action defines what to do
type Action int

const (
    ActionAllow Action = iota
    ActionDeny
    ActionAsk
)

// ConfirmationLevel defines urgency
type ConfirmationLevel int

const (
    LevelInfo ConfirmationLevel = iota
    LevelWarning
    LevelDanger
)

// PolicyDecision contains policy evaluation result
type PolicyDecision struct {
    Action    Action
    Rule      Rule
    Policy    *Policy
    MatchedBy string
}
```

#### 4. DangerDetector

```go
// DangerDetector identifies dangerous operations
type DangerDetector struct {
    patterns []DangerPattern
}

// Detect checks if operation is dangerous
func (dd *DangerDetector) Detect(req ConfirmationRequest) *DangerAssessment {
    assessment := &DangerAssessment{
        Risk:      RiskLow,
        Dangers:   []string{},
        Reversible: true,
    }

    for _, pattern := range dd.patterns {
        if pattern.Match(req) {
            assessment.Risk = maxRisk(assessment.Risk, pattern.Risk)
            assessment.Dangers = append(assessment.Dangers, pattern.Description)
            if !pattern.Reversible {
                assessment.Reversible = false
            }
        }
    }

    return assessment
}

// DangerPattern defines a dangerous pattern
type DangerPattern struct {
    Name        string
    Description string
    Risk        RiskLevel
    Reversible  bool
    Match       func(ConfirmationRequest) bool
}

// DangerAssessment contains risk assessment
type DangerAssessment struct {
    Risk       RiskLevel
    Dangers    []string
    Reversible bool
}

// Default danger patterns
var defaultDangerPatterns = []DangerPattern{
    {
        Name:        "delete_operation",
        Description: "Deleting files or data",
        Risk:        RiskHigh,
        Reversible:  false,
        Match: func(req ConfirmationRequest) bool {
            return req.Operation.Type == OpDelete
        },
    },
    {
        Name:        "system_files",
        Description: "Operating on system files",
        Risk:        RiskCritical,
        Reversible:  false,
        Match: func(req ConfirmationRequest) bool {
            systemPaths := []string{"/etc", "/sys", "/bin", "/usr"}
            for _, path := range systemPaths {
                if strings.HasPrefix(req.Operation.Target, path) {
                    return true
                }
            }
            return false
        },
    },
    {
        Name:        "git_force_push",
        Description: "Force pushing to git remote",
        Risk:        RiskHigh,
        Reversible:  false,
        Match: func(req ConfirmationRequest) bool {
            if req.ToolName == "git" {
                if cmd, ok := req.Parameters["command"].(string); ok {
                    return strings.Contains(cmd, "push") && strings.Contains(cmd, "--force")
                }
            }
            return false
        },
    },
    {
        Name:        "main_branch_operation",
        Description: "Operating on main/master branch",
        Risk:        RiskMedium,
        Reversible:  true,
        Match: func(req ConfirmationRequest) bool {
            if branch, ok := req.Parameters["branch"].(string); ok {
                return branch == "main" || branch == "master"
            }
            return false
        },
    },
    {
        Name:        "network_request",
        Description: "Making network requests",
        Risk:        RiskMedium,
        Reversible:  true,
        Match: func(req ConfirmationRequest) bool {
            return req.Operation.Type == OpNetwork
        },
    },
}

func maxRisk(a, b RiskLevel) RiskLevel {
    if a > b {
        return a
    }
    return b
}
```

#### 5. PromptManager

```go
// PromptManager handles user prompts
type PromptManager struct {
    prompter Prompter
    formatter *PromptFormatter
}

// Prompt prompts user for confirmation
func (pm *PromptManager) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
    // Format prompt
    prompt := pm.formatter.Format(req)

    // Show prompt to user
    response, err := pm.prompter.Prompt(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("prompt user: %w", err)
    }

    return response, nil
}

// PromptRequest describes what to prompt for
type PromptRequest struct {
    Tool        string
    Operation   Operation
    Level       ConfirmationLevel
    Danger      *DangerAssessment
    Preview     string
}

// PromptResponse contains user response
type PromptResponse struct {
    Choice    Choice
    Reason    string
    Timestamp time.Time
}

// Prompter interface for different prompt implementations
type Prompter interface {
    Prompt(ctx context.Context, prompt *FormattedPrompt) (*PromptResponse, error)
}

// FormattedPrompt contains formatted prompt
type FormattedPrompt struct {
    Title       string
    Message     string
    Details     []string
    Level       ConfirmationLevel
    Options     []PromptOption
    DefaultOpt  int
}

// PromptOption represents a choice option
type PromptOption struct {
    Label       string
    Description string
    Choice      Choice
    Shortcut    string
}

// PromptFormatter formats prompts for display
type PromptFormatter struct{}

// Format formats a prompt request
func (pf *PromptFormatter) Format(req PromptRequest) *FormattedPrompt {
    prompt := &FormattedPrompt{
        Title:   fmt.Sprintf("Confirm %s operation", req.Tool),
        Level:   req.Level,
        Options: defaultOptions(),
    }

    // Build message
    var msg strings.Builder
    msg.WriteString(fmt.Sprintf("Tool: %s\n", req.Tool))
    msg.WriteString(fmt.Sprintf("Operation: %s\n", req.Operation.Description))
    if req.Operation.Target != "" {
        msg.WriteString(fmt.Sprintf("Target: %s\n", req.Operation.Target))
    }

    prompt.Message = msg.String()

    // Add details
    if req.Danger != nil && len(req.Danger.Dangers) > 0 {
        prompt.Details = append(prompt.Details, "Warnings:")
        prompt.Details = append(prompt.Details, req.Danger.Dangers...)
        if !req.Danger.Reversible {
            prompt.Details = append(prompt.Details, "⚠️  This operation is NOT reversible!")
        }
    }

    // Add preview
    if req.Preview != "" {
        prompt.Details = append(prompt.Details, "")
        prompt.Details = append(prompt.Details, "Preview:")
        prompt.Details = append(prompt.Details, req.Preview)
    }

    return prompt
}

// Default prompt options
func defaultOptions() []PromptOption {
    return []PromptOption{
        {
            Label:       "Allow",
            Description: "Allow this operation",
            Choice:      ChoiceAllow,
            Shortcut:    "y",
        },
        {
            Label:       "Deny",
            Description: "Deny this operation",
            Choice:      ChoiceDeny,
            Shortcut:    "n",
        },
        {
            Label:       "Always",
            Description: "Always allow this tool",
            Choice:      ChoiceAlways,
            Shortcut:    "a",
        },
        {
            Label:       "Never",
            Description: "Never allow this tool",
            Choice:      ChoiceNever,
            Shortcut:    "N",
        },
    }
}

// InteractivePrompter prompts via terminal
type InteractivePrompter struct {
    input  io.Reader
    output io.Writer
}

// Prompt implements Prompter
func (ip *InteractivePrompter) Prompt(ctx context.Context, prompt *FormattedPrompt) (*PromptResponse, error) {
    // Display prompt
    ip.displayPrompt(prompt)

    // Read response
    reader := bufio.NewReader(ip.input)
    response, err := reader.ReadString('\n')
    if err != nil {
        return nil, err
    }

    response = strings.TrimSpace(response)

    // Parse response
    for _, opt := range prompt.Options {
        if response == opt.Shortcut || strings.EqualFold(response, opt.Label) {
            return &PromptResponse{
                Choice:    opt.Choice,
                Timestamp: time.Now(),
            }, nil
        }
    }

    return nil, fmt.Errorf("invalid choice: %s", response)
}

// displayPrompt displays formatted prompt
func (ip *InteractivePrompter) displayPrompt(prompt *FormattedPrompt) {
    // Display based on level
    switch prompt.Level {
    case LevelInfo:
        fmt.Fprintf(ip.output, "ℹ️  %s\n", prompt.Title)
    case LevelWarning:
        fmt.Fprintf(ip.output, "⚠️  %s\n", prompt.Title)
    case LevelDanger:
        fmt.Fprintf(ip.output, "🚨 %s\n", prompt.Title)
    }

    fmt.Fprintf(ip.output, "\n%s\n", prompt.Message)

    // Display details
    if len(prompt.Details) > 0 {
        fmt.Fprintln(ip.output)
        for _, detail := range prompt.Details {
            fmt.Fprintf(ip.output, "  %s\n", detail)
        }
    }

    // Display options
    fmt.Fprintln(ip.output, "\nOptions:")
    for _, opt := range prompt.Options {
        fmt.Fprintf(ip.output, "  [%s] %s - %s\n", opt.Shortcut, opt.Label, opt.Description)
    }

    fmt.Fprint(ip.output, "\nChoice: ")
}
```

#### 6. AuditLogger

```go
// AuditLogger logs all confirmation decisions
type AuditLogger struct {
    logger  *slog.Logger
    storage AuditStorage
}

// Log logs a confirmation decision
func (al *AuditLogger) Log(ctx context.Context, entry AuditEntry) error {
    // Log to structured logger
    al.logger.InfoContext(ctx, "tool confirmation",
        "tool", entry.ToolName,
        "operation", entry.Operation.Type,
        "decision", entry.Decision,
        "user", entry.User,
        "session", entry.SessionID,
    )

    // Store in audit storage
    if err := al.storage.Store(ctx, entry); err != nil {
        return fmt.Errorf("store audit entry: %w", err)
    }

    return nil
}

// Query queries audit log
func (al *AuditLogger) Query(ctx context.Context, query AuditQuery) ([]AuditEntry, error)

// AuditEntry represents a logged decision
type AuditEntry struct {
    ID            string
    Timestamp     time.Time
    User          string
    SessionID     string
    ConversationID string
    ToolName      string
    Operation     Operation
    Decision      Choice
    Policy        string
    Rule          string
    Reason        string
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

// AuditStorage interface for audit storage
type AuditStorage interface {
    Store(ctx context.Context, entry AuditEntry) error
    Query(ctx context.Context, query AuditQuery) ([]AuditEntry, error)
    Clear(ctx context.Context) error
}

// FileAuditStorage stores audit logs in files
type FileAuditStorage struct {
    path string
    mu   sync.Mutex
}

// Store implements AuditStorage
func (fas *FileAuditStorage) Store(ctx context.Context, entry AuditEntry) error {
    fas.mu.Lock()
    defer fas.mu.Unlock()

    f, err := os.OpenFile(fas.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }

    _, err = f.Write(append(data, '\n'))
    return err
}

// Query implements AuditStorage
func (fas *FileAuditStorage) Query(ctx context.Context, query AuditQuery) ([]AuditEntry, error) {
    fas.mu.Lock()
    defer fas.mu.Unlock()

    f, err := os.Open(fas.path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    var entries []AuditEntry
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        var entry AuditEntry
        if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
            continue
        }

        if matchesQuery(entry, query) {
            entries = append(entries, entry)
            if query.Limit > 0 && len(entries) >= query.Limit {
                break
            }
        }
    }

    return entries, scanner.Err()
}

func matchesQuery(entry AuditEntry, query AuditQuery) bool {
    if query.User != "" && entry.User != query.User {
        return false
    }
    if query.Tool != "" && entry.ToolName != query.Tool {
        return false
    }
    if !query.StartTime.IsZero() && entry.Timestamp.Before(query.StartTime) {
        return false
    }
    if !query.EndTime.IsZero() && entry.Timestamp.After(query.EndTime) {
        return false
    }
    if query.Decision != nil && entry.Decision != *query.Decision {
        return false
    }
    return true
}
```

### State Machine

```
┌─────────┐
│ Pending │  (Tool wants to execute)
└────┬────┘
     │
     ▼
┌────────────┐
│ Evaluating │
│ - Check pol│
│ - Assess   │
└─────┬──────┘
      │
      ├──Policy: Allow──┐
      │                 ▼
      │          ┌───────────┐
      │          │  Allowed  │
      │          └───────────┘
      │
      ├──Policy: Deny───┐
      │                 ▼
      │          ┌───────────┐
      │          │  Denied   │
      │          └───────────┘
      │
      └──Policy: Ask────┐
                        ▼
                 ┌────────────┐
                 │ Prompting  │
                 │ - Show UI  │
                 │ - Wait usr │
                 └─────┬──────┘
                       │
                       ├──Allow/Always──┐
                       │                ▼
                       │         ┌───────────┐
                       │         │  Allowed  │
                       │         └───────────┘
                       │
                       └──Deny/Never────┐
                                        ▼
                                 ┌───────────┐
                                 │  Denied   │
                                 └───────────┘

All states log to:
┌─────────────┐
│ Audit Log   │
└─────────────┘
```

## Batch Mode

```go
// BatchModeHandler handles batch mode execution
type BatchModeHandler struct {
    policy *Policy
}

// Handle handles confirmation in batch mode
func (bmh *BatchModeHandler) Handle(req ConfirmationRequest) (*ConfirmationResult, error) {
    // In batch mode, follow policy strictly without prompting
    decision, err := bmh.policy.Evaluate(req)
    if err != nil {
        return nil, err
    }

    // If policy says "ask", use default action in batch mode
    if decision.Action == ActionAsk {
        decision.Action = bmh.policy.BatchDefaultAction
    }

    return &ConfirmationResult{
        Allowed: decision.Action == ActionAllow,
        Reason:  fmt.Sprintf("batch mode: %s", decision.MatchedBy),
        Policy:  bmh.policy,
    }, nil
}

// Policy extension for batch mode
type Policy struct {
    // ... existing fields ...
    BatchDefaultAction Action // Default action in batch mode
}
```

## Configuration Schema

```yaml
# tool_confirmation.yaml

confirmation:
  # Enable confirmation system
  enabled: true

  # Default policy
  default_policy:
    enabled: true
    default_action: ask  # allow, deny, ask

    # Batch mode default (when policy says "ask" in CI)
    batch_default_action: deny

  # Tool-specific policies
  policies:
    bash:
      default_action: ask
      rules:
        - name: allow_safe_reads
          priority: 10
          condition:
            operation_type: [read]
          action: allow
          level: info

        - name: warn_writes
          priority: 9
          condition:
            operation_type: [write]
          action: ask
          level: warning

        - name: danger_deletes
          priority: 8
          condition:
            operation_type: [delete]
          action: ask
          level: danger

        - name: block_system_paths
          priority: 11
          condition:
            path_pattern: "/etc/**"
          action: deny
          level: danger

    git:
      default_action: ask
      rules:
        - name: warn_force_push
          priority: 10
          condition:
            operation_type: [git]
            custom: is_force_push
          action: ask
          level: danger

        - name: warn_main_branch
          priority: 9
          condition:
            custom: is_main_branch
          action: ask
          level: warning

  # Danger detection
  danger_detection:
    enabled: true
    patterns:
      - name: delete_operation
        risk: high
        reversible: false

      - name: system_files
        risk: critical
        reversible: false

      - name: git_force_push
        risk: high
        reversible: false

  # Prompting
  prompting:
    # Prompt type
    type: interactive  # interactive, gui, api

    # Timeout for user response
    timeout: 5m

    # Show preview
    show_preview: true
    preview_lines: 20

  # Audit logging
  audit:
    enabled: true
    storage: file  # file, database, cloud
    path: .helix/audit/confirmations.jsonl
    retention: 90d

    # What to log
    log_allowed: true
    log_denied: true
    log_policy_changes: true

  # Batch mode
  batch:
    # Detect CI environment
    auto_detect_ci: true

    # CI environment variables to check
    ci_env_vars:
      - CI
      - CONTINUOUS_INTEGRATION
      - GITHUB_ACTIONS
      - GITLAB_CI

    # Allow batch override
    allow_batch_override: false
```

```go
// Config represents confirmation configuration
type Config struct {
    Enabled       bool              `yaml:"enabled"`
    DefaultPolicy PolicyConfig      `yaml:"default_policy"`
    Policies      map[string]PolicyConfig `yaml:"policies"`
    Danger        DangerConfig      `yaml:"danger_detection"`
    Prompting     PromptingConfig   `yaml:"prompting"`
    Audit         AuditConfig       `yaml:"audit"`
    Batch         BatchConfig       `yaml:"batch"`
}

// PolicyConfig configures a policy
type PolicyConfig struct {
    Enabled            bool         `yaml:"enabled"`
    DefaultAction      Action       `yaml:"default_action"`
    BatchDefaultAction Action       `yaml:"batch_default_action"`
    Rules              []RuleConfig `yaml:"rules"`
}

// RuleConfig configures a rule
type RuleConfig struct {
    Name      string          `yaml:"name"`
    Priority  int             `yaml:"priority"`
    Condition ConditionConfig `yaml:"condition"`
    Action    Action          `yaml:"action"`
    Level     ConfirmationLevel `yaml:"level"`
}

// ConditionConfig configures a condition
type ConditionConfig struct {
    OperationType []OperationType `yaml:"operation_type"`
    PathPattern   string          `yaml:"path_pattern"`
    Custom        string          `yaml:"custom"`
}
```

## Testing Strategy

### Unit Tests

```go
func TestPolicyEngine_Evaluate(t *testing.T) {
    pe := NewPolicyEngine()

    policy := &Policy{
        Rules: []Rule{
            {
                Name:     "allow_reads",
                Priority: 10,
                Condition: Condition{
                    OperationType: []OperationType{OpRead},
                },
                Action: ActionAllow,
            },
            {
                Name:     "deny_deletes",
                Priority: 9,
                Condition: Condition{
                    OperationType: []OperationType{OpDelete},
                },
                Action: ActionDeny,
            },
        },
        DefaultAction: ActionAsk,
    }

    pe.SetPolicy("test", policy)

    tests := []struct {
        name    string
        req     ConfirmationRequest
        want    Action
    }{
        {
            name: "allow read",
            req: ConfirmationRequest{
                ToolName: "test",
                Operation: Operation{
                    Type: OpRead,
                },
            },
            want: ActionAllow,
        },
        {
            name: "deny delete",
            req: ConfirmationRequest{
                ToolName: "test",
                Operation: Operation{
                    Type: OpDelete,
                },
            },
            want: ActionDeny,
        },
        {
            name: "ask for write",
            req: ConfirmationRequest{
                ToolName: "test",
                Operation: Operation{
                    Type: OpWrite,
                },
            },
            want: ActionAsk,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            decision, err := pe.Evaluate(tt.req)
            require.NoError(t, err)
            assert.Equal(t, tt.want, decision.Action)
        })
    }
}

func TestDangerDetector_Detect(t *testing.T) {
    dd := NewDangerDetector()

    tests := []struct {
        name     string
        req      ConfirmationRequest
        wantRisk RiskLevel
    }{
        {
            name: "delete operation",
            req: ConfirmationRequest{
                Operation: Operation{
                    Type: OpDelete,
                },
            },
            wantRisk: RiskHigh,
        },
        {
            name: "system file",
            req: ConfirmationRequest{
                Operation: Operation{
                    Type:   OpWrite,
                    Target: "/etc/config",
                },
            },
            wantRisk: RiskCritical,
        },
        {
            name: "normal operation",
            req: ConfirmationRequest{
                Operation: Operation{
                    Type:   OpRead,
                    Target: "/home/user/file.txt",
                },
            },
            wantRisk: RiskLow,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assessment := dd.Detect(tt.req)
            assert.Equal(t, tt.wantRisk, assessment.Risk)
        })
    }
}

func TestPromptFormatter_Format(t *testing.T) {
    pf := &PromptFormatter{}

    req := PromptRequest{
        Tool: "bash",
        Operation: Operation{
            Type:        OpDelete,
            Description: "Delete file",
            Target:      "/tmp/test.txt",
        },
        Level: LevelDanger,
        Danger: &DangerAssessment{
            Risk: RiskHigh,
            Dangers: []string{
                "Deleting files or data",
            },
            Reversible: false,
        },
    }

    prompt := pf.Format(req)
    assert.Equal(t, LevelDanger, prompt.Level)
    assert.Contains(t, prompt.Message, "bash")
    assert.Contains(t, prompt.Message, "Delete file")
    assert.Contains(t, prompt.Details[1], "Deleting files")
}

func TestAuditLogger_Log(t *testing.T) {
    tmpDir := t.TempDir()
    storage := &FileAuditStorage{
        path: filepath.Join(tmpDir, "audit.jsonl"),
    }

    logger := &AuditLogger{
        logger:  slog.Default(),
        storage: storage,
    }

    entry := AuditEntry{
        ID:        "test-1",
        Timestamp: time.Now(),
        User:      "test-user",
        ToolName:  "bash",
        Operation: Operation{
            Type: OpRead,
        },
        Decision: ChoiceAllow,
    }

    err := logger.Log(context.Background(), entry)
    require.NoError(t, err)

    // Query
    entries, err := storage.Query(context.Background(), AuditQuery{
        Tool: "bash",
    })
    require.NoError(t, err)
    assert.Len(t, entries, 1)
    assert.Equal(t, "test-1", entries[0].ID)
}
```

### Integration Tests

```go
func TestConfirmationCoordinator_EndToEnd(t *testing.T) {
    // Create mock prompter that always allows
    mockPrompter := &MockPrompter{
        response: &PromptResponse{
            Choice: ChoiceAllow,
        },
    }

    coordinator := NewConfirmationCoordinator(
        WithPrompter(mockPrompter),
    )

    req := ConfirmationRequest{
        ToolName: "bash",
        Operation: Operation{
            Type:        OpWrite,
            Description: "Write file",
            Target:      "/tmp/test.txt",
            Risk:        RiskLow,
        },
        Context: ExecutionContext{
            User: "test-user",
        },
    }

    result, err := coordinator.Confirm(context.Background(), req)
    require.NoError(t, err)
    assert.True(t, result.Allowed)
    assert.NotEmpty(t, result.AuditID)
}

func TestBatchMode(t *testing.T) {
    coordinator := NewConfirmationCoordinator()

    req := ConfirmationRequest{
        ToolName: "bash",
        Operation: Operation{
            Type: OpRead,
        },
        BatchMode: true,
        Context: ExecutionContext{
            CI: true,
        },
    }

    result, err := coordinator.Confirm(context.Background(), req)
    require.NoError(t, err)
    // Should use batch default action without prompting
    assert.NotEmpty(t, result.Reason)
}
```

### Policy Tests

```go
func TestPolicyConfiguration(t *testing.T) {
    tests := []struct {
        name   string
        policy *Policy
        req    ConfirmationRequest
        want   Action
    }{
        {
            name: "high priority rule wins",
            policy: &Policy{
                Rules: []Rule{
                    {
                        Priority: 5,
                        Condition: Condition{
                            OperationType: []OperationType{OpWrite},
                        },
                        Action: ActionDeny,
                    },
                    {
                        Priority: 10,
                        Condition: Condition{
                            PathPattern: "/tmp/**",
                        },
                        Action: ActionAllow,
                    },
                },
            },
            req: ConfirmationRequest{
                Operation: Operation{
                    Type:   OpWrite,
                    Target: "/tmp/test.txt",
                },
            },
            want: ActionAllow, // Higher priority wins
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            pe := &PolicyEngine{
                policies: map[string]*Policy{
                    "test": tt.policy,
                },
            }

            tt.req.ToolName = "test"
            decision, err := pe.Evaluate(tt.req)
            require.NoError(t, err)
            assert.Equal(t, tt.want, decision.Action)
        })
    }
}
```

## Performance Considerations

### Caching Decisions

```go
// DecisionCache caches policy decisions
type DecisionCache struct {
    mu    sync.RWMutex
    cache map[string]*CachedDecision
    ttl   time.Duration
}

type CachedDecision struct {
    Decision  *PolicyDecision
    Timestamp time.Time
}

// Get retrieves cached decision
func (dc *DecisionCache) Get(key string) (*PolicyDecision, bool) {
    dc.mu.RLock()
    defer dc.mu.RUnlock()

    cached, ok := dc.cache[key]
    if !ok {
        return nil, false
    }

    if time.Since(cached.Timestamp) > dc.ttl {
        return nil, false
    }

    return cached.Decision, true
}

// Set stores decision in cache
func (dc *DecisionCache) Set(key string, decision *PolicyDecision) {
    dc.mu.Lock()
    defer dc.mu.Unlock()

    dc.cache[key] = &CachedDecision{
        Decision:  decision,
        Timestamp: time.Now(),
    }
}

// generateCacheKey creates cache key from request
func generateCacheKey(req ConfirmationRequest) string {
    h := sha256.New()
    h.Write([]byte(req.ToolName))
    h.Write([]byte(req.Operation.Type))
    h.Write([]byte(req.Operation.Target))
    return fmt.Sprintf("%x", h.Sum(nil))
}
```

## Security Considerations

### Policy Validation

```go
// ValidatePolicy ensures policy is safe
func ValidatePolicy(policy *Policy) error {
    if policy == nil {
        return fmt.Errorf("policy cannot be nil")
    }

    // Check for conflicting rules
    for i, r1 := range policy.Rules {
        for j, r2 := range policy.Rules {
            if i != j && r1.Priority == r2.Priority {
                return fmt.Errorf("rules %s and %s have same priority", r1.Name, r2.Name)
            }
        }
    }

    // Ensure at least one rule or default action
    if len(policy.Rules) == 0 && policy.DefaultAction == 0 {
        return fmt.Errorf("policy must have rules or default action")
    }

    return nil
}
```

## References

### Cline Confirmation System

- **Repository**: `src/core/confirmation-manager.ts`
- **Features**:
  - Interactive prompts for tool execution
  - Always/Never options
  - Risk assessment
  - Audit logging

### Gemini CLI Policy Engine

- **Feature**: Policy-based tool execution control
- **Implementation**: Rule-based policy evaluation
- **Policies**: Allow, deny, ask with conditions

### Key Insights

1. **Safety First**: Default to asking for dangerous operations
2. **User Control**: Allow users to set permanent choices
3. **Transparency**: Always show what will happen
4. **Audit Trail**: Log all decisions for accountability
5. **Batch Support**: Handle CI/CD without prompts

## Usage Examples

```go
// Example 1: Basic confirmation
func ExampleBasicConfirmation() {
    coordinator := NewConfirmationCoordinator()

    result, _ := coordinator.Confirm(context.Background(), ConfirmationRequest{
        ToolName: "bash",
        Operation: Operation{
            Type:        OpWrite,
            Description: "Write configuration file",
            Target:      "/etc/app/config.yaml",
        },
    })

    if result.Allowed {
        // Execute operation
    }
}

// Example 2: With custom policy
func ExampleCustomPolicy() {
    coordinator := NewConfirmationCoordinator()

    policy := &Policy{
        DefaultAction: ActionAsk,
        Rules: []Rule{
            {
                Name:     "allow_tmp",
                Priority: 10,
                Condition: Condition{
                    PathPattern: "/tmp/**",
                },
                Action: ActionAllow,
            },
        },
    }

    coordinator.SetPolicy("bash", policy)
}

// Example 3: Batch mode
func ExampleBatchMode() {
    coordinator := NewConfirmationCoordinator()

    result, _ := coordinator.Confirm(context.Background(), ConfirmationRequest{
        ToolName:  "bash",
        BatchMode: true,
        Context: ExecutionContext{
            CI: true,
        },
    })

    // Will use batch default action without prompting
}
```

## Future Enhancements

1. **ML-Based Risk Assessment**: Learn from user decisions
2. **Context-Aware Policies**: Policies based on conversation context
3. **Remote Policy Management**: Centralized policy server
4. **Rollback Support**: Automatic rollback on failure
5. **Notification Integration**: Slack/email notifications for critical operations
