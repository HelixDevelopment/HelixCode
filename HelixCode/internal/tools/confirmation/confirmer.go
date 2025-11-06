package confirmation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConfirmationCoordinator manages confirmation workflow
type ConfirmationCoordinator struct {
	policyEngine   *PolicyEngine
	promptManager  *PromptManager
	auditLogger    *AuditLogger
	dangerDetector *DangerDetector
	config         *Config

	mu          sync.RWMutex
	userChoices map[string]Choice // Tool -> permanent choice
}

// NewConfirmationCoordinator creates a new coordinator
func NewConfirmationCoordinator(opts ...Option) *ConfirmationCoordinator {
	config := DefaultConfig()

	cc := &ConfirmationCoordinator{
		policyEngine:   NewPolicyEngine(),
		promptManager:  NewPromptManager(),
		auditLogger:    NewAuditLogger(config.AuditPath),
		dangerDetector: NewDangerDetector(),
		config:         config,
		userChoices:    make(map[string]Choice),
	}

	// Apply options
	for _, opt := range opts {
		opt(cc)
	}

	return cc
}

// Confirm checks if tool execution should be allowed
func (cc *ConfirmationCoordinator) Confirm(ctx context.Context, req ConfirmationRequest) (*ConfirmationResult, error) {
	// Check if disabled
	if !cc.config.Enabled {
		return &ConfirmationResult{
			Allowed:   true,
			Reason:    "confirmation disabled",
			Choice:    ChoiceAllow,
			Timestamp: time.Now(),
		}, nil
	}

	// Check for permanent user choice
	cc.mu.RLock()
	userChoice, hasUserChoice := cc.userChoices[req.ToolName]
	cc.mu.RUnlock()

	if hasUserChoice {
		allowed := userChoice == ChoiceAlways
		result := &ConfirmationResult{
			Allowed:   allowed,
			Reason:    fmt.Sprintf("user choice: %s", userChoice.String()),
			Choice:    userChoice,
			Timestamp: time.Now(),
			AuditID:   GenerateAuditID(),
		}

		// Log to audit
		_ = cc.logToAudit(ctx, req, result, nil, nil)

		return result, nil
	}

	// Assess danger
	dangerAssessment := cc.dangerDetector.Detect(req)
	if dangerAssessment.Risk > req.Operation.Risk {
		req.Operation.Risk = dangerAssessment.Risk
		req.Operation.Reversible = dangerAssessment.Reversible
	}

	// Evaluate policy
	decision, err := cc.policyEngine.Evaluate(req)
	if err != nil {
		return nil, fmt.Errorf("evaluate policy: %w", err)
	}

	// Handle batch mode
	if req.BatchMode || cc.config.BatchMode || req.Context.CI {
		return cc.handleBatchMode(ctx, req, decision, dangerAssessment)
	}

	// Handle based on policy action
	switch decision.Action {
	case ActionAllow:
		return cc.handleAllow(ctx, req, decision, dangerAssessment)

	case ActionDeny:
		return cc.handleDeny(ctx, req, decision, dangerAssessment)

	case ActionAsk:
		return cc.handleAsk(ctx, req, decision, dangerAssessment)

	default:
		return nil, fmt.Errorf("unknown action: %v", decision.Action)
	}
}

// handleAllow handles allow action
func (cc *ConfirmationCoordinator) handleAllow(ctx context.Context, req ConfirmationRequest, decision *PolicyDecision, danger *DangerAssessment) (*ConfirmationResult, error) {
	result := &ConfirmationResult{
		Allowed:   true,
		Reason:    fmt.Sprintf("policy: %s", decision.MatchedBy),
		Choice:    ChoiceAllow,
		Policy:    decision.Policy,
		Timestamp: time.Now(),
		AuditID:   GenerateAuditID(),
	}

	// Log to audit
	_ = cc.logToAudit(ctx, req, result, decision, danger)

	return result, nil
}

// handleDeny handles deny action
func (cc *ConfirmationCoordinator) handleDeny(ctx context.Context, req ConfirmationRequest, decision *PolicyDecision, danger *DangerAssessment) (*ConfirmationResult, error) {
	result := &ConfirmationResult{
		Allowed:   false,
		Reason:    fmt.Sprintf("policy: %s", decision.MatchedBy),
		Choice:    ChoiceDeny,
		Policy:    decision.Policy,
		Timestamp: time.Now(),
		AuditID:   GenerateAuditID(),
	}

	// Log to audit
	_ = cc.logToAudit(ctx, req, result, decision, danger)

	return result, nil
}

// handleAsk handles ask action (prompts user)
func (cc *ConfirmationCoordinator) handleAsk(ctx context.Context, req ConfirmationRequest, decision *PolicyDecision, danger *DangerAssessment) (*ConfirmationResult, error) {
	// Determine confirmation level
	level := LevelInfo
	if decision.Rule != nil {
		level = decision.Rule.Level
	}

	// Upgrade level based on risk
	if req.Operation.Risk >= RiskHigh {
		level = LevelDanger
	} else if req.Operation.Risk >= RiskMedium {
		level = LevelWarning
	}

	// Create prompt request
	promptReq := PromptRequest{
		Tool:      req.ToolName,
		Operation: req.Operation,
		Level:     level,
		Danger:    danger,
		Preview:   req.Operation.Preview,
	}

	// Prompt user
	response, err := cc.promptManager.Prompt(ctx, promptReq)
	if err != nil {
		return nil, fmt.Errorf("prompt user: %w", err)
	}

	// Handle permanent choices
	if response.Choice == ChoiceAlways || response.Choice == ChoiceNever {
		cc.mu.Lock()
		cc.userChoices[req.ToolName] = response.Choice
		cc.mu.Unlock()
	}

	// Determine if allowed
	allowed := response.Choice == ChoiceAllow || response.Choice == ChoiceAlways

	result := &ConfirmationResult{
		Allowed:   allowed,
		Reason:    fmt.Sprintf("user: %s", response.Choice.String()),
		Choice:    response.Choice,
		Policy:    decision.Policy,
		Timestamp: time.Now(),
		AuditID:   GenerateAuditID(),
	}

	// Log to audit
	_ = cc.logToAudit(ctx, req, result, decision, danger)

	return result, nil
}

// handleBatchMode handles batch mode execution
func (cc *ConfirmationCoordinator) handleBatchMode(ctx context.Context, req ConfirmationRequest, decision *PolicyDecision, danger *DangerAssessment) (*ConfirmationResult, error) {
	action := decision.Action
	if action == ActionAsk {
		action = decision.Policy.BatchDefaultAction
	}

	allowed := action == ActionAllow

	result := &ConfirmationResult{
		Allowed:   allowed,
		Reason:    fmt.Sprintf("batch mode: %s", decision.MatchedBy),
		Choice:    ChoiceAllow,
		Policy:    decision.Policy,
		Timestamp: time.Now(),
		AuditID:   GenerateAuditID(),
	}

	if !allowed {
		result.Choice = ChoiceDeny
	}

	// Log to audit
	_ = cc.logToAudit(ctx, req, result, decision, danger)

	return result, nil
}

// logToAudit logs the confirmation decision to audit log
func (cc *ConfirmationCoordinator) logToAudit(ctx context.Context, req ConfirmationRequest, result *ConfirmationResult, decision *PolicyDecision, danger *DangerAssessment) error {
	entry := AuditEntry{
		ID:             result.AuditID,
		Timestamp:      result.Timestamp,
		User:           req.Context.User,
		SessionID:      req.Context.SessionID,
		ConversationID: req.Context.ConversationID,
		ToolName:       req.ToolName,
		Operation:      req.Operation,
		Decision:       result.Choice,
		Reason:         result.Reason,
	}

	if decision != nil {
		if decision.Policy != nil {
			entry.Policy = decision.Policy.Name
		}
		if decision.Rule != nil {
			entry.Rule = decision.Rule.Name
		}
	}

	return cc.auditLogger.Log(ctx, entry)
}

// GetPolicy retrieves the policy for a tool
func (cc *ConfirmationCoordinator) GetPolicy(toolName string) (*Policy, error) {
	policy, ok := cc.policyEngine.GetPolicy(toolName)
	if !ok {
		return cc.config.DefaultPolicy, nil
	}
	return policy, nil
}

// SetPolicy updates the policy for a tool
func (cc *ConfirmationCoordinator) SetPolicy(toolName string, policy *Policy) error {
	return cc.policyEngine.SetPolicy(toolName, policy)
}

// ResetChoices clears all permanent user choices
func (cc *ConfirmationCoordinator) ResetChoices() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.userChoices = make(map[string]Choice)
}

// GetUserChoice retrieves a permanent user choice for a tool
func (cc *ConfirmationCoordinator) GetUserChoice(toolName string) (Choice, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	choice, ok := cc.userChoices[toolName]
	return choice, ok
}

// SetUserChoice sets a permanent user choice for a tool
func (cc *ConfirmationCoordinator) SetUserChoice(toolName string, choice Choice) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.userChoices[toolName] = choice
}

// QueryAudit queries the audit log
func (cc *ConfirmationCoordinator) QueryAudit(ctx context.Context, query AuditQuery) ([]AuditEntry, error) {
	return cc.auditLogger.Query(ctx, query)
}

// ClearAudit clears the audit log
func (cc *ConfirmationCoordinator) ClearAudit(ctx context.Context) error {
	return cc.auditLogger.Clear(ctx)
}
