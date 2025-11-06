// Package planmode implements the Plan Mode workflow for HelixCode.
//
// Plan Mode provides a two-phase workflow where the system first generates
// and presents multiple implementation options to the user, then executes
// the selected approach. This design is inspired by Cline's Plan Mode,
// enhanced with better option presentation, state management, and execution tracking.
//
// # Architecture
//
// The package is organized into several key components:
//
//   - ModeController: Manages operational modes (Normal, Plan, Act, Paused)
//   - Planner: Generates implementation plans using LLM integration
//   - OptionPresenter: Presents options to users and handles selection
//   - Executor: Executes plans step-by-step with progress tracking
//   - StateManager: Manages state across the workflow
//   - PlanModeWorkflow: Orchestrates the complete workflow
//
// # Workflow Phases
//
// ## Phase 1: Planning
//
//  1. System enters Plan mode
//  2. LLM generates 3-4 implementation options
//  3. Options are ranked by score (complexity, confidence, risks, pros/cons)
//  4. Options are presented to the user
//  5. User selects preferred option
//
// ## Phase 2: Execution
//
//  1. System enters Act mode
//  2. Selected plan is executed step-by-step
//  3. Progress is tracked and reported
//  4. Results are stored
//  5. System returns to Normal mode
//
// # Usage Examples
//
// ## Basic Usage
//
//	// Create components
//	llmProvider := llm.NewLocalProvider(config)
//	planner := planmode.NewLLMPlanner(llmProvider)
//	presenter := planmode.NewCLIOptionPresenter(os.Stdout, os.Stdin)
//	executor := planmode.NewDefaultExecutor("/workspace")
//	stateManager := planmode.NewStateManager()
//	controller := planmode.NewModeController()
//
//	// Create workflow
//	workflow := planmode.NewPlanModeWorkflow(
//	    planner,
//	    presenter,
//	    executor,
//	    stateManager,
//	    controller,
//	)
//
//	// Create plan mode instance
//	config := planmode.DefaultConfig()
//	pm := planmode.NewPlanMode(workflow, config)
//
//	// Execute workflow
//	task := &planmode.Task{
//	    ID:          "task-1",
//	    Description: "Implement user authentication",
//	    Requirements: []string{
//	        "Use JWT tokens",
//	        "Support OAuth2",
//	    },
//	}
//
//	result, err := pm.Run(context.Background(), task)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Execution completed: %v\n", result.Success)
//
// ## With Progress Tracking
//
//	result, err := pm.RunWithProgress(ctx, task, func(progress *planmode.WorkflowProgress) {
//	    fmt.Printf("[%s] %s (%d/%d steps)\n",
//	        progress.Phase,
//	        progress.Status,
//	        progress.CompletedSteps,
//	        progress.TotalSteps,
//	    )
//	})
//
// ## YOLO Mode (Auto-Select Best Option)
//
//	config := planmode.DefaultConfig()
//	config.AutoSelectBest = true
//	pm := planmode.NewPlanMode(workflow, config)
//
//	result, err := pm.Run(ctx, task)
//
// # Mode Transitions
//
// Valid mode transitions:
//
//	Normal -> Plan   (start planning)
//	Plan   -> Act    (start execution)
//	Plan   -> Normal (cancel planning)
//	Act    -> Paused (pause execution)
//	Act    -> Normal (complete execution)
//	Paused -> Act    (resume execution)
//	Paused -> Normal (cancel execution)
//
// # Plan Structure
//
// A Plan consists of:
//
//   - Title and description
//   - Ordered steps with dependencies
//   - Risk assessment (impact, likelihood, mitigation)
//   - Time and complexity estimates
//   - Resource requirements
//
// ## Step Types
//
//   - FileOperation: File creation, modification, deletion
//   - ShellCommand: Shell command execution
//   - CodeGeneration: LLM-based code generation
//   - CodeAnalysis: Code analysis and review
//   - Validation: Validation checks
//   - Testing: Test execution
//
// # Option Ranking
//
// Options are scored (0-100) based on:
//
//   - Complexity (30%): Lower is better
//   - Confidence (30%): Higher is better
//   - Pros vs Cons: More pros, fewer cons is better
//   - Risks: Lower risk impact and likelihood is better
//
// Custom ranking criteria can be provided:
//
//	criteria := []planmode.RankCriterion{
//	    {Name: "Speed", Weight: 1.0, Type: planmode.CriterionSpeed},
//	    {Name: "Safety", Weight: 0.8, Type: planmode.CriterionSafety},
//	    {Name: "Simplicity", Weight: 0.6, Type: planmode.CriterionSimplicity},
//	}
//
//	ranked, err := presenter.RankOptions(options, criteria)
//
// # State Management
//
// The StateManager maintains:
//
//   - Plans: All generated plans
//   - Options: Options for each plan
//   - Selections: User selections
//   - Executions: Execution results
//
// State is thread-safe and can be persisted for recovery.
//
// # Error Handling
//
// The package provides robust error handling:
//
//   - Plan validation before execution
//   - Step dependency checking
//   - Execution error capture and reporting
//   - Graceful degradation on failures
//
// Errors are collected in ExecutionResult.Errors for analysis.
//
// # Progress Tracking
//
// Progress is tracked at multiple levels:
//
//   - Workflow level: Planning vs Execution phases
//   - Execution level: Current step, completed/failed counts
//   - Step level: Individual step status and results
//
// Progress callbacks enable real-time UI updates:
//
//	pm.RunWithProgress(ctx, task, func(progress *WorkflowProgress) {
//	    updateUI(progress)
//	})
//
// # LLM Integration
//
// The Planner integrates with the LLM provider system:
//
//   - Uses structured prompts for plan generation
//   - Parses JSON responses into Plan structures
//   - Supports plan refinement based on feedback
//   - Validates generated plans
//
// Custom prompt templates can be provided via PromptBuilder.
//
// # Execution Control
//
// Execution can be controlled during runtime:
//
//	// Pause execution
//	err := pm.Pause(executionID)
//
//	// Resume execution
//	err := pm.Resume(executionID)
//
//	// Cancel execution
//	err := pm.Cancel(executionID)
//
//	// Get progress
//	progress, err := pm.GetProgress(executionID)
//
// # Configuration
//
// Plan Mode can be configured via Config:
//
//   - DefaultOptionCount: Number of options to generate (default: 3)
//   - MaxOptionCount: Maximum options allowed (default: 5)
//   - AutoSelectBest: Enable YOLO mode (default: false)
//   - ShowComparison: Show option comparison (default: true)
//   - EnableProgressBar: Enable progress visualization (default: true)
//   - ConfidenceThreshold: Minimum confidence for plans (default: 0.7)
//   - MaxPlanComplexity: Maximum allowed complexity (default: High)
//
// # Testing
//
// The package includes comprehensive tests:
//
//   - Unit tests for all components
//   - Integration tests for workflow
//   - Mock implementations for testing
//   - Benchmark tests for performance
//
// Run tests with:
//
//	go test -v ./internal/workflow/planmode
//	go test -bench=. ./internal/workflow/planmode
//
// # References
//
// This implementation is inspired by:
//
//   - Cline's Plan Mode: Two-phase workflow pattern
//   - YOLO Mode: Automatic best-option selection
//   - Enterprise planning patterns: Multi-option presentation
//
// # Future Enhancements
//
// Planned enhancements include:
//
//   - Interactive plan editing before execution
//   - Plan templates for common tasks
//   - Plan versioning and history
//   - Rollback support for failed executions
//   - Cost estimation (time, resources, API calls)
//   - Parallel execution of independent steps
//   - Conditional step execution
//   - Plan visualization (flowcharts, diagrams)
//   - Multi-user collaboration and approval
//   - Machine learning from past executions
package planmode
