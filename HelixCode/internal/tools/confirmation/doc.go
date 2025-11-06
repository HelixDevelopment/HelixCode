/*
Package confirmation provides an interactive confirmation system for tool execution
with policy-based approval, user prompts, and audit logging.

# Overview

The confirmation package implements a safety control system that ensures dangerous
or sensitive operations require explicit user approval. It supports multiple
confirmation levels, policy-based auto-approval, and comprehensive audit logging.

# Core Components

The package consists of several key components:

  - ConfirmationCoordinator: Orchestrates the confirmation workflow
  - PolicyEngine: Evaluates policies and rules
  - DangerDetector: Identifies dangerous operations
  - PromptManager: Handles user interaction
  - AuditLogger: Logs all confirmation decisions

# Basic Usage

Create a confirmation coordinator and use it to confirm operations:

	coordinator := confirmation.NewConfirmationCoordinator()

	req := confirmation.ConfirmationRequest{
		ToolName: "bash",
		Operation: confirmation.Operation{
			Type:        confirmation.OpWrite,
			Description: "Write configuration file",
			Target:      "/etc/app/config.yaml",
			Risk:        confirmation.RiskMedium,
		},
		Context: confirmation.ExecutionContext{
			User: "admin",
		},
	}

	result, err := coordinator.Confirm(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	if result.Allowed {
		// Execute the operation
	}

# Policies

Policies define how operations should be handled. You can create custom policies:

	policy := &confirmation.Policy{
		Name:          "custom",
		DefaultAction: confirmation.ActionAsk,
		Rules: []confirmation.Rule{
			{
				Name:     "allow_safe_reads",
				Priority: 10,
				Condition: confirmation.Condition{
					OperationType: []confirmation.OperationType{confirmation.OpRead},
					RiskLevel:     []confirmation.RiskLevel{confirmation.RiskLow},
				},
				Action: confirmation.ActionAllow,
				Level:  confirmation.LevelInfo,
			},
		},
	}

	coordinator.SetPolicy("bash", policy)

# Danger Detection

The danger detector automatically identifies risky operations:

  - Delete operations (rm -rf, etc.)
  - System file modifications (/etc, /sys, etc.)
  - Git force pushes
  - Elevated privilege commands (sudo)
  - Database operations (DROP TABLE, TRUNCATE)
  - Package publishing (npm publish, twine upload)

# Batch Mode

For CI/CD environments, batch mode allows operations without user prompts:

	req := confirmation.ConfirmationRequest{
		ToolName:  "bash",
		BatchMode: true,
		Context: confirmation.ExecutionContext{
			CI: true,
		},
	}

	result, _ := coordinator.Confirm(context.Background(), req)

# Audit Logging

All confirmation decisions are logged for audit purposes:

	entries, err := coordinator.QueryAudit(context.Background(),
		confirmation.AuditQuery{
			User: "admin",
			Tool: "bash",
		})

# User Choices

Users can set permanent choices (always/never) for specific tools:

	coordinator.SetUserChoice("bash", confirmation.ChoiceAlways)

# Risk Levels

Operations are categorized by risk level:

  - RiskNone: No risk (read-only operations)
  - RiskLow: Minimal risk (safe writes)
  - RiskMedium: Moderate risk (network operations)
  - RiskHigh: High risk (deletions, force operations)
  - RiskCritical: Critical risk (system files, irreversible operations)

# Confirmation Levels

Prompts are displayed with different urgency levels:

  - LevelInfo: Informational (safe operations)
  - LevelWarning: Warning (potentially risky)
  - LevelDanger: Danger (high-risk operations)

# Actions

Policies can specify three types of actions:

  - ActionAllow: Automatically allow the operation
  - ActionDeny: Automatically deny the operation
  - ActionAsk: Prompt the user for confirmation

# Examples

Example 1: Custom policy for git operations

	gitPolicy := &confirmation.Policy{
		Name:          "git",
		DefaultAction: confirmation.ActionAsk,
		Rules: []confirmation.Rule{
			{
				Name:     "warn_force_push",
				Priority: 15,
				Condition: confirmation.Condition{
					Custom: func(req confirmation.ConfirmationRequest) bool {
						if cmd, ok := req.Parameters["command"].(string); ok {
							return strings.Contains(cmd, "push --force")
						}
						return false
					},
				},
				Action: confirmation.ActionAsk,
				Level:  confirmation.LevelDanger,
			},
		},
	}

Example 2: Querying audit logs

	startTime := time.Now().Add(-24 * time.Hour)
	entries, err := coordinator.QueryAudit(ctx, confirmation.AuditQuery{
		StartTime: startTime,
		Decision:  &confirmation.ChoiceDeny,
		Limit:     100,
	})

Example 3: Custom prompter for testing

	mockPrompter := &confirmation.MockPrompter{
		Response: &confirmation.PromptResponse{
			Choice: confirmation.ChoiceAllow,
		},
	}

	coordinator := confirmation.NewConfirmationCoordinator(
		confirmation.WithPrompter(mockPrompter),
	)

# Thread Safety

All components are thread-safe and can be used concurrently:

  - PolicyEngine uses RWMutex for policy storage
  - ConfirmationCoordinator uses RWMutex for user choices
  - AuditLogger uses Mutex for file operations

# Configuration

Configure the coordinator with functional options:

	coordinator := confirmation.NewConfirmationCoordinator(
		confirmation.WithBatchMode(true),
		confirmation.WithAuditPath("/var/log/helix/audit.jsonl"),
		confirmation.WithConfig(&confirmation.Config{
			Enabled: true,
		}),
	)

# Testing

The package includes mock implementations for testing:

  - MockPrompter: Simulates user responses
  - MemoryAuditStorage: In-memory audit log for tests

# Performance

The package is designed for minimal overhead:

  - Policy evaluation is O(n) where n is the number of rules
  - Audit logging is asynchronous
  - File operations use buffered I/O

# Security

Security considerations:

  - All policies are validated before use
  - Audit logs are append-only
  - User choices are stored in memory (not persisted)
  - Batch mode respects policy defaults

# Integration

Integrate with tool execution:

	func ExecuteTool(toolName string, operation Operation) error {
		req := confirmation.ConfirmationRequest{
			ToolName:  toolName,
			Operation: operation,
			Context: confirmation.ExecutionContext{
				User: getCurrentUser(),
			},
		}

		result, err := confirmationCoordinator.Confirm(context.Background(), req)
		if err != nil {
			return err
		}

		if !result.Allowed {
			return fmt.Errorf("operation denied: %s", result.Reason)
		}

		// Execute the tool
		return doExecute(toolName, operation)
	}
*/
package confirmation
