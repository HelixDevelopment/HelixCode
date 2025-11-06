// Package autonomy provides a five-level spectrum of AI autonomy modes for HelixCode.
//
// # Overview
//
// The autonomy system enables users to control how much the AI can do independently,
// balancing automation with user oversight. It provides five distinct levels ranging
// from complete manual control to fully autonomous operation.
//
// # Autonomy Modes
//
// The five autonomy modes from most manual to most automated:
//
//  1. None (Level 1) - Complete manual control
//     - No automatic actions
//     - User confirms everything
//     - Best for: Critical systems, auditing
//
//  2. Basic (Level 2) - Manual workflow
//     - Manual context gathering
//     - Manual file selection
//     - Step-by-step workflow
//     - Best for: Fine-grained control
//
//  3. Basic Plus (Level 3) - Smart semi-automation
//     - Context suggestions with manual selection
//     - Apply suggestions with confirmation
//     - Best for: Learning the tool
//
//  4. Semi Auto (Level 4) - Automated with approval [DEFAULT]
//     - Automatic context gathering
//     - Manual approval for changes
//     - One-click apply after review
//     - Best for: Most development workflows
//
//  5. Full Auto (Level 5) - Fully autonomous
//     - No user confirmation required
//     - Automatic context gathering
//     - Auto-apply all changes
//     - Continuous iteration until completion
//     - Best for: Trusted environments, simple tasks
//
// # Basic Usage
//
// Create a controller with default configuration:
//
//	controller, err := autonomy.NewAutonomyController(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get current mode
//	mode := controller.GetCurrentMode()
//	fmt.Printf("Current mode: %s\n", mode)
//
//	// Change mode
//	err = controller.SetMode(ctx, autonomy.ModeFullAuto)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Action Execution
//
// Execute actions with automatic permission checking:
//
//	// Create an action
//	action := autonomy.NewAction(
//	    autonomy.ActionLoadContext,
//	    "Load project context",
//	    autonomy.RiskNone,
//	)
//
//	// Execute with automatic permission check
//	result, err := controller.ExecuteAction(ctx, action)
//	if err != nil {
//	    log.Printf("Action failed: %v", err)
//	}
//
//	if result.Success {
//	    fmt.Printf("Action completed: %s\n", result.Output)
//	}
//
// # Permission Checking
//
// Check if an action is permitted before execution:
//
//	action := autonomy.NewAction(
//	    autonomy.ActionApplyChange,
//	    "Apply code changes",
//	    autonomy.RiskMedium,
//	)
//	action.Context = &autonomy.ActionContext{
//	    FilesAffected: []string{"main.go", "utils.go"},
//	}
//
//	perm, err := controller.RequestPermission(ctx, action)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if !perm.Granted {
//	    fmt.Printf("Permission denied: %s\n", perm.Reason)
//	    return
//	}
//
//	if perm.RequiresConfirm {
//	    fmt.Printf("Confirmation required: %s\n", perm.ConfirmPrompt)
//	    // Request user confirmation...
//	}
//
// # Mode Escalation
//
// Temporarily escalate to a higher mode:
//
//	// Escalate for 30 minutes
//	err := controller.RequestEscalation(
//	    ctx,
//	    "debugging critical issue",
//	    30*time.Minute,
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Mode will automatically revert after duration
//	// Or manually de-escalate:
//	err = controller.DeEscalate(ctx)
//
// # Guardrails
//
// Safety guardrails protect against dangerous operations:
//
//	// Add custom guardrail
//	controller.AddGuardrailRule(autonomy.GuardrailRule{
//	    Name:        "no_production_delete",
//	    Description: "Prevent deletions in production",
//	    Severity:    autonomy.RiskCritical,
//	    Enabled:     true,
//	    Check: func(ctx context.Context, action *autonomy.Action) (bool, string) {
//	        if action.Type == autonomy.ActionFileDelete {
//	            if strings.Contains(action.Description, "production") {
//	                return false, "cannot delete production files"
//	            }
//	        }
//	        return true, ""
//	    },
//	})
//
//	// Disable a guardrail
//	controller.DisableGuardrailRule("no_system_file_delete")
//
// # Configuration
//
// Configure the autonomy system:
//
//	config := autonomy.NewDefaultConfig()
//
//	// Mode settings
//	config.DefaultMode = autonomy.ModeSemiAuto
//	config.AllowModeSwitch = true
//	config.PersistMode = true
//
//	// Safety settings
//	config.EnableGuardrails = true
//	config.RiskThreshold = autonomy.RiskMedium
//	config.RequireReason = true
//
//	// Confirmation settings
//	config.ConfirmRisky = true
//	config.ConfirmBulk = true
//	config.BulkThreshold = 5
//
//	// Auto-debug settings
//	config.DebugEnabled = true
//	config.MaxRetries = 3
//	config.RetryDelay = 2 * time.Second
//
//	controller, err := autonomy.NewAutonomyController(config)
//
// # Metrics
//
// Track system performance:
//
//	metrics := controller.GetMetrics()
//	stats := metrics.GetStats()
//
//	fmt.Printf("Actions executed: %d\n", stats.ActionsExecuted)
//	fmt.Printf("Success rate: %.2f%%\n", stats.SuccessRate()*100)
//	fmt.Printf("Approval rate: %.2f%%\n", stats.ApprovalRate()*100)
//	fmt.Printf("Average execution time: %v\n", stats.AverageExecuteTime)
//
// # Mode History
//
// Track mode changes:
//
//	history := controller.GetModeHistory()
//	for _, change := range history.Changes {
//	    fmt.Printf("%s: %s -> %s (%s)\n",
//	        change.Timestamp.Format(time.RFC3339),
//	        change.From, change.To, change.Reason)
//	}
//
// # Graceful Shutdown
//
// Save state on shutdown:
//
//	defer func() {
//	    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	    defer cancel()
//
//	    if err := controller.Shutdown(ctx); err != nil {
//	        log.Printf("Shutdown error: %v", err)
//	    }
//	}()
//
// # Thread Safety
//
// All components are thread-safe and can be used concurrently:
//
//	// Multiple goroutines can safely use the controller
//	for i := 0; i < 10; i++ {
//	    go func(id int) {
//	        action := autonomy.NewAction(
//	            autonomy.ActionLoadContext,
//	            fmt.Sprintf("Task %d", id),
//	            autonomy.RiskNone,
//	        )
//	        result, _ := controller.ExecuteAction(ctx, action)
//	        fmt.Printf("Task %d: %v\n", id, result.Success)
//	    }(i)
//	}
//
// # References
//
// Based on patterns from:
//   - Plandex autonomy modes (plan_config.go)
//   - Agentic framework patterns (AutoGPT, BabyAGI)
//   - Human-in-the-loop ML design principles
//   - AI safety and alignment research
package autonomy
