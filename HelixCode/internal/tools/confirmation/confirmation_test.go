package confirmation

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Test 1: PolicyEngine Evaluate - Allow reads
func TestPolicyEngine_EvaluateAllowReads(t *testing.T) {
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
		},
		DefaultAction: ActionAsk,
		Enabled:       true,
	}

	if err := pe.SetPolicy("test", policy); err != nil {
		t.Fatalf("SetPolicy failed: %v", err)
	}

	req := ConfirmationRequest{
		ToolName: "test",
		Operation: Operation{
			Type: OpRead,
		},
	}

	decision, err := pe.Evaluate(req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Action != ActionAllow {
		t.Errorf("Expected ActionAllow, got %v", decision.Action)
	}
}

// Test 2: PolicyEngine Evaluate - Deny deletes
func TestPolicyEngine_EvaluateDenyDeletes(t *testing.T) {
	pe := NewPolicyEngine()

	policy := &Policy{
		Rules: []Rule{
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
		Enabled:       true,
	}

	if err := pe.SetPolicy("test", policy); err != nil {
		t.Fatalf("SetPolicy failed: %v", err)
	}

	req := ConfirmationRequest{
		ToolName: "test",
		Operation: Operation{
			Type: OpDelete,
		},
	}

	decision, err := pe.Evaluate(req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Action != ActionDeny {
		t.Errorf("Expected ActionDeny, got %v", decision.Action)
	}
}

// Test 3: PolicyEngine Evaluate - Default action for unmatched
func TestPolicyEngine_EvaluateDefaultAction(t *testing.T) {
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
		},
		DefaultAction: ActionAsk,
		Enabled:       true,
	}

	if err := pe.SetPolicy("test", policy); err != nil {
		t.Fatalf("SetPolicy failed: %v", err)
	}

	req := ConfirmationRequest{
		ToolName: "test",
		Operation: Operation{
			Type: OpWrite,
		},
	}

	decision, err := pe.Evaluate(req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Action != ActionAsk {
		t.Errorf("Expected ActionAsk, got %v", decision.Action)
	}
}

// Test 4: DangerDetector - Detect delete operation
func TestDangerDetector_DetectDeleteOperation(t *testing.T) {
	dd := NewDangerDetector()

	req := ConfirmationRequest{
		Operation: Operation{
			Type: OpDelete,
		},
	}

	assessment := dd.Detect(req)

	if assessment.Risk < RiskHigh {
		t.Errorf("Expected RiskHigh or higher, got %v", assessment.Risk)
	}

	if assessment.Reversible {
		t.Error("Delete operation should not be reversible")
	}

	if len(assessment.Dangers) == 0 {
		t.Error("Expected at least one danger description")
	}
}

// Test 5: DangerDetector - Detect system file operation
func TestDangerDetector_DetectSystemFiles(t *testing.T) {
	dd := NewDangerDetector()

	req := ConfirmationRequest{
		Operation: Operation{
			Type:   OpWrite,
			Target: "/etc/config",
		},
	}

	assessment := dd.Detect(req)

	if assessment.Risk != RiskCritical {
		t.Errorf("Expected RiskCritical, got %v", assessment.Risk)
	}

	if assessment.Reversible {
		t.Error("System file operation should not be reversible")
	}
}

// Test 6: DangerDetector - Detect git force push
func TestDangerDetector_DetectGitForcePush(t *testing.T) {
	dd := NewDangerDetector()

	req := ConfirmationRequest{
		ToolName: "git",
		Parameters: map[string]interface{}{
			"command": "git push --force origin main",
		},
	}

	assessment := dd.Detect(req)

	if assessment.Risk < RiskHigh {
		t.Errorf("Expected RiskHigh or higher, got %v", assessment.Risk)
	}

	if assessment.Reversible {
		t.Error("Force push should not be reversible")
	}
}

// Test 7: DangerDetector - Detect sudo command
func TestDangerDetector_DetectSudoCommand(t *testing.T) {
	dd := NewDangerDetector()

	req := ConfirmationRequest{
		ToolName: "bash",
		Parameters: map[string]interface{}{
			"command": "sudo rm -rf /tmp/test",
		},
	}

	assessment := dd.Detect(req)

	if assessment.Risk < RiskHigh {
		t.Errorf("Expected RiskHigh or higher, got %v", assessment.Risk)
	}
}

// Test 8: PromptFormatter - Format prompt
func TestPromptFormatter_Format(t *testing.T) {
	pf := &PromptFormatter{}

	req := PromptRequest{
		Tool: "bash",
		Operation: Operation{
			Type:        OpDelete,
			Description: "Delete file",
			Target:      "/tmp/test.txt",
			Risk:        RiskHigh,
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

	if prompt.Level != LevelDanger {
		t.Errorf("Expected LevelDanger, got %v", prompt.Level)
	}

	if !strings.Contains(prompt.Message, "bash") {
		t.Error("Prompt message should contain tool name")
	}

	if !strings.Contains(prompt.Message, "Delete file") {
		t.Error("Prompt message should contain operation description")
	}

	if len(prompt.Details) == 0 {
		t.Error("Expected danger details")
	}
}

// Test 9: AuditLogger - Log and query
func TestAuditLogger_LogAndQuery(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger := NewAuditLoggerWithStorage(storage)

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

	ctx := context.Background()
	err := logger.Log(ctx, entry)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	entries, err := storage.Query(ctx, AuditQuery{
		Tool: "bash",
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].ID != "test-1" {
		t.Errorf("Expected ID test-1, got %s", entries[0].ID)
	}
}

// Test 10: AuditLogger - Query filtering
func TestAuditLogger_QueryFiltering(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger := NewAuditLoggerWithStorage(storage)

	ctx := context.Background()

	// Add multiple entries
	entries := []AuditEntry{
		{
			ID:        "test-1",
			Timestamp: time.Now(),
			User:      "user1",
			ToolName:  "bash",
			Decision:  ChoiceAllow,
		},
		{
			ID:        "test-2",
			Timestamp: time.Now(),
			User:      "user2",
			ToolName:  "git",
			Decision:  ChoiceDeny,
		},
		{
			ID:        "test-3",
			Timestamp: time.Now(),
			User:      "user1",
			ToolName:  "bash",
			Decision:  ChoiceDeny,
		},
	}

	for _, entry := range entries {
		if err := logger.Log(ctx, entry); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Query by user
	results, err := storage.Query(ctx, AuditQuery{
		User: "user1",
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 entries for user1, got %d", len(results))
	}

	// Query by tool
	results, err = storage.Query(ctx, AuditQuery{
		Tool: "git",
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 entry for git, got %d", len(results))
	}

	// Query by decision
	denyDecision := ChoiceDeny
	results, err = storage.Query(ctx, AuditQuery{
		Decision: &denyDecision,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 deny entries, got %d", len(results))
	}
}

// Test 11: ConfirmationCoordinator - End to end with allow
func TestConfirmationCoordinator_EndToEndAllow(t *testing.T) {
	mockPrompter := &MockPrompter{
		Response: &PromptResponse{
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
	if err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected operation to be allowed")
	}

	if result.AuditID == "" {
		t.Error("Expected audit ID to be set")
	}
}

// Test 12: ConfirmationCoordinator - Batch mode
func TestConfirmationCoordinator_BatchMode(t *testing.T) {
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
	if err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	if result.Reason == "" {
		t.Error("Expected reason to be set")
	}

	if !strings.Contains(result.Reason, "batch") {
		t.Errorf("Expected batch mode reason, got: %s", result.Reason)
	}
}

// Test 13: ConfirmationCoordinator - User choice persistence
func TestConfirmationCoordinator_UserChoicePersistence(t *testing.T) {
	mockPrompter := &MockPrompter{
		Response: &PromptResponse{
			Choice: ChoiceAlways,
		},
	}

	coordinator := NewConfirmationCoordinator(
		WithPrompter(mockPrompter),
	)

	req := ConfirmationRequest{
		ToolName: "bash",
		Operation: Operation{
			Type: OpWrite,
		},
		Context: ExecutionContext{
			User: "test-user",
		},
	}

	// First confirmation - should prompt
	result1, err := coordinator.Confirm(context.Background(), req)
	if err != nil {
		t.Fatalf("First Confirm failed: %v", err)
	}

	if result1.Choice != ChoiceAlways {
		t.Errorf("Expected ChoiceAlways, got %v", result1.Choice)
	}

	// Second confirmation - should use cached choice
	result2, err := coordinator.Confirm(context.Background(), req)
	if err != nil {
		t.Fatalf("Second Confirm failed: %v", err)
	}

	if !result2.Allowed {
		t.Error("Expected operation to be allowed based on cached choice")
	}

	if !strings.Contains(result2.Reason, "user choice") {
		t.Errorf("Expected user choice reason, got: %s", result2.Reason)
	}
}

// Test 14: ConfirmationCoordinator - Reset choices
func TestConfirmationCoordinator_ResetChoices(t *testing.T) {
	coordinator := NewConfirmationCoordinator()

	coordinator.SetUserChoice("bash", ChoiceAlways)

	choice, ok := coordinator.GetUserChoice("bash")
	if !ok || choice != ChoiceAlways {
		t.Error("Expected user choice to be set")
	}

	coordinator.ResetChoices()

	_, ok = coordinator.GetUserChoice("bash")
	if ok {
		t.Error("Expected user choice to be cleared")
	}
}

// Test 15: Policy validation - Conflicting priorities
func TestPolicyValidation_ConflictingPriorities(t *testing.T) {
	policy := &Policy{
		Rules: []Rule{
			{
				Name:     "rule1",
				Priority: 10,
			},
			{
				Name:     "rule2",
				Priority: 10,
			},
		},
	}

	err := ValidatePolicy(policy)
	if err == nil {
		t.Error("Expected validation error for conflicting priorities")
	}

	if !strings.Contains(err.Error(), "same priority") {
		t.Errorf("Expected error about same priority, got: %v", err)
	}
}

// Test 16: Condition matching - Path pattern
func TestCondition_MatchesPathPattern(t *testing.T) {
	condition := Condition{
		PathPattern: "/tmp/*",
	}

	req := ConfirmationRequest{
		Operation: Operation{
			Target: "/tmp/test.txt",
		},
	}

	if !condition.Matches(req) {
		t.Error("Expected condition to match /tmp/test.txt")
	}

	req.Operation.Target = "/home/test.txt"
	if condition.Matches(req) {
		t.Error("Expected condition not to match /home/test.txt")
	}
}

// Test 17: Condition matching - Risk level
func TestCondition_MatchesRiskLevel(t *testing.T) {
	condition := Condition{
		RiskLevel: []RiskLevel{RiskHigh, RiskCritical},
	}

	req := ConfirmationRequest{
		Operation: Operation{
			Risk: RiskHigh,
		},
	}

	if !condition.Matches(req) {
		t.Error("Expected condition to match RiskHigh")
	}

	req.Operation.Risk = RiskLow
	if condition.Matches(req) {
		t.Error("Expected condition not to match RiskLow")
	}
}

// Test 18: BashPolicy - Block system paths
func TestBashPolicy_BlockSystemPaths(t *testing.T) {
	pe := NewPolicyEngine()
	policy := BashPolicy()

	if err := pe.SetPolicy("bash", policy); err != nil {
		t.Fatalf("SetPolicy failed: %v", err)
	}

	req := ConfirmationRequest{
		ToolName: "bash",
		Operation: Operation{
			Type:   OpWrite,
			Target: "/etc/passwd",
		},
	}

	decision, err := pe.Evaluate(req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Action != ActionDeny {
		t.Errorf("Expected ActionDeny for system path, got %v", decision.Action)
	}
}

// Test 19: GitPolicy - Warn force push
func TestGitPolicy_WarnForcePush(t *testing.T) {
	pe := NewPolicyEngine()
	policy := GitPolicy()

	if err := pe.SetPolicy("git", policy); err != nil {
		t.Fatalf("SetPolicy failed: %v", err)
	}

	req := ConfirmationRequest{
		ToolName: "git",
		Operation: Operation{
			Type: OpGit,
		},
		Parameters: map[string]interface{}{
			"command": "git push --force origin main",
		},
	}

	decision, err := pe.Evaluate(req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Action != ActionAsk {
		t.Errorf("Expected ActionAsk for force push, got %v", decision.Action)
	}
}

// Test 20: Audit storage - Clear
func TestAuditStorage_Clear(t *testing.T) {
	storage := NewMemoryAuditStorage()

	ctx := context.Background()

	// Add entries
	entry := AuditEntry{
		ID:       "test-1",
		ToolName: "bash",
	}

	if err := storage.Store(ctx, entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Clear
	if err := storage.Clear(ctx); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify empty
	entries, err := storage.Query(ctx, AuditQuery{})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}
}
