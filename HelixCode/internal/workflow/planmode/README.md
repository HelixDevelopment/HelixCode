# Plan Mode Implementation

This package implements the complete Plan Mode workflow for HelixCode based on the technical design specification.

## Overview

Plan Mode provides a two-phase workflow where the system first generates and presents multiple implementation options to the user, then executes the selected approach. This design is inspired by Cline's Plan Mode, enhanced with better option presentation, state management, and execution tracking.

## Package Structure

```
planmode/
├── doc.go               - Comprehensive package documentation
├── mode_controller.go   - Mode management (Normal/Plan/Act/Paused)
├── planner.go          - Planning engine with LLM integration
├── options.go          - Option management and presentation
├── executor.go         - Plan execution with progress tracking
├── planmode.go         - Main workflow orchestration
├── planmode_test.go    - Comprehensive test suite (15+ tests)
└── README.md           - This file
```

## Implementation Statistics

- **Total Lines**: 3,270 lines of Go code
- **Production Code**: 2,477 lines
- **Test Code**: 793 lines
- **Test Coverage**: 15+ test cases including:
  - Mode transition tests
  - Plan generation tests
  - Option ranking tests
  - Workflow execution tests
  - Error recovery tests
  - Benchmark tests

## Key Features Implemented

### 1. Mode Controller (193 lines)
- **Modes**: Normal, Plan, Act, Paused
- **Valid Transitions**:
  - Normal → Plan (start planning)
  - Plan → Act (start execution)
  - Plan → Normal (cancel)
  - Act → Paused (pause)
  - Act → Normal (complete)
  - Paused → Act (resume)
- **State Management**: Thread-safe state tracking with callbacks
- **Callbacks**: Mode change notifications

### 2. Planner (730 lines)
- **LLM Integration**: Full integration with HelixCode LLM provider system
- **Plan Generation**: Creates detailed implementation plans
- **Option Generation**: Generates 3-4 distinct implementation approaches
- **Plan Validation**: Validates plans for correctness
- **Plan Refinement**: Refines plans based on user feedback
- **Scoring System**: Ranks options based on:
  - Complexity (30%)
  - Confidence (30%)
  - Pros vs Cons
  - Risk assessment

### 3. Options Management (340 lines)
- **PlanOption**: Complete option structure with pros/cons
- **CLI Presenter**: Interactive command-line option presentation
- **Option Comparison**: Side-by-side comparison matrix
- **Ranking System**: Multi-criteria ranking with weights
- **Criterion Types**:
  - Speed
  - Safety
  - Simplicity
  - Maintainability
  - Performance
  - Cost

### 4. Executor (456 lines)
- **Step Execution**: Executes individual plan steps
- **Progress Tracking**: Real-time progress updates
- **Pause/Resume**: Control execution flow
- **Cancellation**: Graceful cancellation support
- **Error Handling**: Comprehensive error capture and reporting
- **Step Types**:
  - File Operations
  - Shell Commands
  - Code Generation
  - Code Analysis
  - Validation
  - Testing

### 5. Workflow Orchestration (499 lines)
- **State Manager**: Thread-safe state management
- **Two-Phase Workflow**: Plan → Execute
- **Progress Callbacks**: Real-time progress notifications
- **YOLO Mode**: Auto-select best option and execute
- **Configuration**: Flexible configuration system

## Usage Examples

### Basic Usage

```go
// Create components
llmProvider := llm.NewLocalProvider(config)
planner := planmode.NewLLMPlanner(llmProvider)
presenter := planmode.NewCLIOptionPresenter(os.Stdout, os.Stdin)
executor := planmode.NewDefaultExecutor("/workspace")
stateManager := planmode.NewStateManager()
controller := planmode.NewModeController()

// Create workflow
workflow := planmode.NewPlanModeWorkflow(
    planner, presenter, executor, stateManager, controller,
)

// Create plan mode instance
config := planmode.DefaultConfig()
pm := planmode.NewPlanMode(workflow, config)

// Execute workflow
task := &planmode.Task{
    ID:          "task-1",
    Description: "Implement user authentication",
    Requirements: []string{
        "Use JWT tokens",
        "Support OAuth2",
    },
}

result, err := pm.Run(context.Background(), task)
```

### With Progress Tracking

```go
result, err := pm.RunWithProgress(ctx, task,
    func(progress *planmode.WorkflowProgress) {
        fmt.Printf("[%s] %s (%d/%d steps)\n",
            progress.Phase,
            progress.Status,
            progress.CompletedSteps,
            progress.TotalSteps,
        )
    })
```

### YOLO Mode

```go
config := planmode.DefaultConfig()
config.AutoSelectBest = true
pm := planmode.NewPlanMode(workflow, config)

result, err := pm.Run(ctx, task)
```

## Testing

The package includes comprehensive tests covering:

1. **Mode Controller Tests**
   - Initial mode verification
   - Valid transitions
   - Invalid transitions
   - Mode change callbacks
   - State management

2. **Planner Tests**
   - Plan generation
   - Option generation
   - Plan validation
   - Option ranking

3. **State Manager Tests**
   - Plan storage/retrieval
   - Option storage/retrieval
   - Selection tracking
   - Execution results

4. **Executor Tests**
   - Step execution
   - Plan execution
   - Pause/resume
   - Progress tracking

5. **Workflow Tests**
   - Full workflow execution
   - Progress tracking
   - YOLO mode
   - Error recovery

6. **Benchmark Tests**
   - Plan generation performance
   - Option ranking performance

### Running Tests

```bash
# Run all tests
go test -v ./internal/workflow/planmode

# Run with coverage
go test -cover ./internal/workflow/planmode

# Run benchmarks
go test -bench=. ./internal/workflow/planmode
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         PlanMode                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   ModeController                         │  │
│  │  (Manages mode transitions: Normal ↔ Plan ↔ Act)       │  │
│  └──────────────┬───────────────────────────┬───────────────┘  │
│                 │                           │                   │
│         ┌───────▼────────┐         ┌────────▼────────┐         │
│         │                │         │                 │         │
│    ┌────┴────┐      ┌────┴────┐   │    Executor     │         │
│    │ Planner │      │ Option  │   │                 │         │
│    │         │──────│Presenter│   │                 │         │
│    └────┬────┘      └────┬────┘   └────────┬────────┘         │
│         │                │                  │                   │
│    ┌────▼────────────────▼──────────────────▼────────┐         │
│    │              StateManager                        │         │
│    │  (Plan, Options, Selection, Execution State)    │         │
│    └──────────────────────────────────────────────────┘         │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐          │
│  │            ProgressTracker                       │          │
│  │  (Tracks execution progress and status)         │          │
│  └──────────────────────────────────────────────────┘          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
   FileSystem       ShellExecution      LLM Provider
```

## Configuration Options

```go
type Config struct {
    DefaultOptionCount  int     // Default: 3
    MaxOptionCount      int     // Default: 5
    AutoSelectBest      bool    // Default: false (YOLO mode)
    ShowComparison      bool    // Default: true
    EnableProgressBar   bool    // Default: true
    ConfidenceThreshold float64 // Default: 0.7
    MaxPlanComplexity   Complexity // Default: High
}
```

## Plan Structure

A Plan consists of:
- **Title and Description**: High-level overview
- **Steps**: Ordered steps with dependencies
- **Risks**: Risk assessment with impact/likelihood/mitigation
- **Estimates**: Time, complexity, and confidence estimates
- **Resources**: Required resources
- **Status**: Current plan status

## Step Types

1. **FileOperation**: File creation, modification, deletion
2. **ShellCommand**: Shell command execution
3. **CodeGeneration**: LLM-based code generation
4. **CodeAnalysis**: Code analysis and review
5. **Validation**: Validation checks
6. **Testing**: Test execution

## Error Handling

The package provides robust error handling:
- Plan validation before execution
- Step dependency checking
- Execution error capture and reporting
- Graceful degradation on failures
- Comprehensive error collection in ExecutionResult

## Future Enhancements

Planned enhancements (as documented in the design):

1. Interactive plan editing before execution
2. Plan templates for common tasks
3. Plan versioning and history
4. Rollback support for failed executions
5. Cost estimation (time, resources, API calls)
6. Parallel execution of independent steps
7. Conditional step execution
8. Plan visualization (flowcharts, diagrams)
9. Multi-user collaboration and approval
10. Machine learning from past executions

## Integration with HelixCode

This package integrates seamlessly with HelixCode's existing systems:

- **LLM Providers**: Uses `dev.helix.code/internal/llm` for all LLM operations
- **Project Context**: Can integrate with `dev.helix.code/internal/project`
- **Workflow System**: Extends `dev.helix.code/internal/workflow`
- **Task Management**: Compatible with task management system

## License

Part of the HelixCode project. See main project LICENSE for details.
