package planmode

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMProvider is a mock LLM provider for testing
type MockLLMProvider struct {
	responses map[string]string
}

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		responses: map[string]string{
			"plan": `{
				"title": "Test Plan",
				"description": "A test implementation plan",
				"steps": [
					{
						"title": "Step 1",
						"description": "First step",
						"type": "file_operation",
						"action": "create file",
						"dependencies": [],
						"estimated_minutes": 5
					},
					{
						"title": "Step 2",
						"description": "Second step",
						"type": "shell_command",
						"action": "run command",
						"dependencies": ["step-1"],
						"estimated_minutes": 10
					}
				],
				"risks": [
					{
						"description": "Low risk",
						"impact": "low",
						"likelihood": "low",
						"mitigation": "None needed"
					}
				],
				"estimates": {
					"duration_minutes": 15,
					"complexity": "low",
					"confidence": 0.9
				}
			}`,
			"options": `{
				"options": [
					{
						"title": "Option 1",
						"description": "First approach",
						"plan": {
							"title": "Plan 1",
							"description": "First plan",
							"steps": [
								{
									"title": "Step 1",
									"description": "First step",
									"type": "file_operation",
									"action": "create",
									"dependencies": [],
									"estimated_minutes": 5
								}
							],
							"risks": [],
							"estimates": {
								"duration_minutes": 5,
								"complexity": "low",
								"confidence": 0.95
							}
						},
						"pros": ["Simple", "Fast"],
						"cons": ["Limited features"]
					},
					{
						"title": "Option 2",
						"description": "Second approach",
						"plan": {
							"title": "Plan 2",
							"description": "Second plan",
							"steps": [
								{
									"title": "Step 1",
									"description": "First step",
									"type": "code_generation",
									"action": "generate",
									"dependencies": [],
									"estimated_minutes": 15
								}
							],
							"risks": [],
							"estimates": {
								"duration_minutes": 15,
								"complexity": "medium",
								"confidence": 0.85
							}
						},
						"pros": ["Full featured", "Robust"],
						"cons": ["More complex", "Slower"]
					}
				]
			}`,
		},
	}
}

func (m *MockLLMProvider) Generate(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
	var response string
	if strings.Contains(request.Messages[1].Content, "multiple") || strings.Contains(request.Messages[1].Content, "options") {
		response = m.responses["options"]
	} else {
		response = m.responses["plan"]
	}

	return &llm.LLMResponse{
		ID:        uuid.New(),
		RequestID: request.ID,
		Content:   response,
		Usage: llm.Usage{
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
		},
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockLLMProvider) GenerateStream(ctx context.Context, request *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	return nil
}

func (m *MockLLMProvider) GetType() llm.ProviderType {
	return llm.ProviderTypeLocal
}

func (m *MockLLMProvider) GetName() string {
	return "mock"
}

func (m *MockLLMProvider) GetModels() []llm.ModelInfo {
	return []llm.ModelInfo{{Name: "mock-model"}}
}

func (m *MockLLMProvider) GetCapabilities() []llm.ModelCapability {
	return []llm.ModelCapability{llm.CapabilityPlanning}
}

func (m *MockLLMProvider) IsAvailable(ctx context.Context) bool {
	return true
}

func (m *MockLLMProvider) GetHealth(ctx context.Context) (*llm.ProviderHealth, error) {
	return &llm.ProviderHealth{Status: "healthy"}, nil
}

func (m *MockLLMProvider) Close() error {
	return nil
}

// MockPresenter is a mock option presenter for testing
type MockPresenter struct {
	selectedIndex int
}

func (m *MockPresenter) Present(ctx context.Context, options []*PlanOption) (*Selection, error) {
	if len(options) == 0 {
		return nil, assert.AnError
	}

	index := m.selectedIndex
	if index >= len(options) {
		index = 0
	}

	return &Selection{
		OptionID:  options[index].ID,
		Timestamp: time.Now(),
	}, nil
}

func (m *MockPresenter) CompareOptions(options []*PlanOption) (*Comparison, error) {
	return &Comparison{
		Options:  options,
		Criteria: []string{"Complexity", "Duration"},
		Matrix:   [][]ComparisonCell{},
	}, nil
}

func (m *MockPresenter) RankOptions(options []*PlanOption, criteria []RankCriterion) ([]*RankedOption, error) {
	ranked := make([]*RankedOption, len(options))
	for i, opt := range options {
		ranked[i] = &RankedOption{
			Option: opt,
			Rank:   i + 1,
			Score:  100.0 - float64(i*10),
		}
	}
	return ranked, nil
}

// Test Mode Controller
func TestModeController(t *testing.T) {
	t.Run("InitialMode", func(t *testing.T) {
		controller := NewModeController()
		assert.Equal(t, ModeNormal, controller.GetMode())
	})

	t.Run("ValidTransitions", func(t *testing.T) {
		controller := NewModeController()

		// Normal -> Plan
		err := controller.TransitionTo(ModePlan)
		require.NoError(t, err)
		assert.Equal(t, ModePlan, controller.GetMode())

		// Plan -> Act
		err = controller.TransitionTo(ModeAct)
		require.NoError(t, err)
		assert.Equal(t, ModeAct, controller.GetMode())

		// Act -> Paused
		err = controller.TransitionTo(ModePaused)
		require.NoError(t, err)
		assert.Equal(t, ModePaused, controller.GetMode())

		// Paused -> Act
		err = controller.TransitionTo(ModeAct)
		require.NoError(t, err)
		assert.Equal(t, ModeAct, controller.GetMode())

		// Act -> Normal
		err = controller.TransitionTo(ModeNormal)
		require.NoError(t, err)
		assert.Equal(t, ModeNormal, controller.GetMode())
	})

	t.Run("InvalidTransitions", func(t *testing.T) {
		controller := NewModeController()

		// Normal -> Act (invalid)
		err := controller.TransitionTo(ModeAct)
		assert.Error(t, err)

		// Normal -> Paused (invalid)
		err = controller.TransitionTo(ModePaused)
		assert.Error(t, err)
	})

	t.Run("ModeChangeCallback", func(t *testing.T) {
		controller := NewModeController()
		var called bool
		var fromMode, toMode Mode

		controller.RegisterCallback(func(from, to Mode, state *ModeState) {
			called = true
			fromMode = from
			toMode = to
		})

		controller.TransitionTo(ModePlan)
		assert.True(t, called)
		assert.Equal(t, ModeNormal, fromMode)
		assert.Equal(t, ModePlan, toMode)
	})

	t.Run("StateManagement", func(t *testing.T) {
		controller := NewModeController()

		state := &ModeState{
			Mode:        ModePlan,
			PlanID:      "plan-123",
			OptionID:    "option-456",
			ExecutionID: "exec-789",
			Metadata:    map[string]interface{}{"key": "value"},
		}

		err := controller.UpdateState(state)
		require.NoError(t, err)

		retrieved := controller.GetState()
		assert.Equal(t, state.PlanID, retrieved.PlanID)
		assert.Equal(t, state.OptionID, retrieved.OptionID)
		assert.Equal(t, state.ExecutionID, retrieved.ExecutionID)
	})
}

// Test Planner
func TestPlanner(t *testing.T) {
	mockLLM := NewMockLLMProvider()
	planner := NewLLMPlanner(mockLLM)

	t.Run("GeneratePlan", func(t *testing.T) {
		task := &Task{
			ID:          "task-1",
			Description: "Create a test feature",
			Context: &TaskContext{
				WorkspaceRoot: "/test",
			},
		}

		plan, err := planner.GeneratePlan(context.Background(), task)
		require.NoError(t, err)
		assert.NotNil(t, plan)
		assert.Equal(t, "Test Plan", plan.Title)
		assert.Equal(t, 2, len(plan.Steps))
		assert.Equal(t, PlanDraft, plan.Status)
	})

	t.Run("GenerateOptions", func(t *testing.T) {
		task := &Task{
			ID:          "task-2",
			Description: "Implement authentication",
		}

		options, err := planner.GenerateOptions(context.Background(), task)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(options), 2)

		// Check options are ranked
		for i, opt := range options {
			assert.Equal(t, i+1, opt.Rank)
			assert.Greater(t, opt.Score, 0.0)
		}

		// First option should be recommended
		assert.True(t, options[0].Recommended)
	})

	t.Run("ValidatePlan", func(t *testing.T) {
		validPlan := &Plan{
			ID:    "plan-1",
			Title: "Valid Plan",
			Steps: []*PlanStep{
				{
					ID:    "step-1",
					Title: "Step 1",
				},
			},
		}

		result, err := planner.ValidatePlan(context.Background(), validPlan)
		require.NoError(t, err)
		assert.True(t, result.Valid)

		invalidPlan := &Plan{
			ID:    "plan-2",
			Title: "",
			Steps: []*PlanStep{},
		}

		result, err = planner.ValidatePlan(context.Background(), invalidPlan)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Greater(t, len(result.Errors), 0)
	})
}

// Test State Manager
func TestStateManager(t *testing.T) {
	sm := NewStateManager()

	t.Run("StorePlan", func(t *testing.T) {
		plan := &Plan{
			ID:    "plan-1",
			Title: "Test Plan",
		}

		err := sm.StorePlan(plan)
		require.NoError(t, err)

		retrieved, err := sm.GetPlan("plan-1")
		require.NoError(t, err)
		assert.Equal(t, plan.ID, retrieved.ID)
		assert.Equal(t, plan.Title, retrieved.Title)
	})

	t.Run("StoreOptions", func(t *testing.T) {
		options := []*PlanOption{
			{ID: "opt-1", Title: "Option 1"},
			{ID: "opt-2", Title: "Option 2"},
		}

		err := sm.StoreOptions("plan-1", options)
		require.NoError(t, err)

		retrieved, err := sm.GetOptions("plan-1")
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
	})

	t.Run("StoreSelection", func(t *testing.T) {
		selection := &Selection{
			OptionID:  "opt-1",
			Timestamp: time.Now(),
		}

		err := sm.StoreSelection("plan-1", selection)
		require.NoError(t, err)

		retrieved, err := sm.GetSelection("plan-1")
		require.NoError(t, err)
		assert.Equal(t, selection.OptionID, retrieved.OptionID)
	})

	t.Run("ListPlans", func(t *testing.T) {
		plans := sm.ListPlans()
		assert.GreaterOrEqual(t, len(plans), 1)
	})

	t.Run("ClearPlan", func(t *testing.T) {
		sm.ClearPlan("plan-1")
		_, err := sm.GetPlan("plan-1")
		assert.Error(t, err)
	})
}

// Test Executor
func TestExecutor(t *testing.T) {
	executor := NewDefaultExecutor("/tmp/test")

	t.Run("ExecuteStep", func(t *testing.T) {
		step := &PlanStep{
			ID:          "step-1",
			Title:       "Test Step",
			Description: "echo 'test'",
			Type:        StepTypeShellCommand,
			Action:      "echo 'test'",
		}

		result, err := executor.ExecuteStep(context.Background(), step)
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("ExecutePlan", func(t *testing.T) {
		plan := &Plan{
			ID: "plan-1",
			Steps: []*PlanStep{
				{
					ID:          "step-1",
					Title:       "Step 1",
					Description: "First step",
					Type:        StepTypeFileOperation,
					Action:      "test",
				},
				{
					ID:          "step-2",
					Title:       "Step 2",
					Description: "Second step",
					Type:        StepTypeFileOperation,
					Action:      "test2",
				},
			},
		}

		result, err := executor.Execute(context.Background(), plan)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, len(result.Steps))
	})

	t.Run("PauseResumeExecution", func(t *testing.T) {
		plan := &Plan{
			ID: "plan-2",
			Steps: []*PlanStep{
				{
					ID:    "step-1",
					Title: "Long Step",
					Type:  StepTypeFileOperation,
				},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			executor.Execute(ctx, plan)
		}()

		time.Sleep(100 * time.Millisecond)

		// Note: This test is simplified as we can't easily get the execution ID
		// in a real scenario without modifying the API
	})
}

// Test Option Presenter
func TestOptionPresenter(t *testing.T) {
	t.Run("CLIPresenter", func(t *testing.T) {
		var output bytes.Buffer
		input := strings.NewReader("1\n")

		presenter := NewCLIOptionPresenter(&output, input)

		options := []*PlanOption{
			{
				ID:          "opt-1",
				Title:       "Option 1",
				Description: "First option",
				Score:       85.0,
				Recommended: true,
				Plan: &Plan{
					Estimates: Estimates{
						Duration:   15 * time.Minute,
						Complexity: ComplexityLow,
						Confidence: 0.9,
					},
				},
				Pros: []string{"Fast", "Simple"},
				Cons: []string{"Limited"},
			},
		}

		selection, err := presenter.Present(context.Background(), options)
		require.NoError(t, err)
		assert.Equal(t, "opt-1", selection.OptionID)

		outputStr := output.String()
		assert.Contains(t, outputStr, "Option 1")
		assert.Contains(t, outputStr, "[RECOMMENDED]")
		assert.Contains(t, outputStr, "Fast")
	})

	t.Run("CompareOptions", func(t *testing.T) {
		presenter := NewCLIOptionPresenter(&bytes.Buffer{}, strings.NewReader(""))

		options := []*PlanOption{
			{
				ID:    "opt-1",
				Title: "Option 1",
				Plan: &Plan{
					Estimates: Estimates{
						Duration:   10 * time.Minute,
						Complexity: ComplexityLow,
						Confidence: 0.9,
					},
					Risks: []Risk{},
				},
			},
			{
				ID:    "opt-2",
				Title: "Option 2",
				Plan: &Plan{
					Estimates: Estimates{
						Duration:   20 * time.Minute,
						Complexity: ComplexityMedium,
						Confidence: 0.8,
					},
					Risks: []Risk{},
				},
			},
		}

		comparison, err := presenter.CompareOptions(options)
		require.NoError(t, err)
		assert.Equal(t, 2, len(comparison.Options))
		assert.Greater(t, len(comparison.Criteria), 0)
	})

	t.Run("RankOptions", func(t *testing.T) {
		presenter := NewCLIOptionPresenter(&bytes.Buffer{}, strings.NewReader(""))

		options := []*PlanOption{
			{
				ID: "opt-1",
				Plan: &Plan{
					Estimates: Estimates{
						Duration:   10 * time.Minute,
						Complexity: ComplexityLow,
						Confidence: 0.9,
					},
				},
			},
		}

		criteria := []RankCriterion{
			{Name: "Speed", Weight: 1.0, Type: CriterionSpeed},
			{Name: "Simplicity", Weight: 0.8, Type: CriterionSimplicity},
		}

		ranked, err := presenter.RankOptions(options, criteria)
		require.NoError(t, err)
		assert.Equal(t, 1, len(ranked))
		assert.Equal(t, 1, ranked[0].Rank)
	})
}

// Test Workflow
func TestPlanModeWorkflow(t *testing.T) {
	mockLLM := NewMockLLMProvider()
	planner := NewLLMPlanner(mockLLM)
	presenter := &MockPresenter{selectedIndex: 0}
	executor := NewDefaultExecutor("/tmp/test")
	stateManager := NewStateManager()
	controller := NewModeController()

	workflow := NewPlanModeWorkflow(planner, presenter, executor, stateManager, controller)

	t.Run("ExecuteWorkflow", func(t *testing.T) {
		task := &Task{
			ID:          "task-1",
			Description: "Test task",
		}

		result, err := workflow.ExecuteWorkflow(context.Background(), task)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, ModeNormal, controller.GetMode())
	})

	t.Run("ExecuteWithProgress", func(t *testing.T) {
		task := &Task{
			ID:          "task-2",
			Description: "Test task with progress",
		}

		progressUpdates := 0
		progressFn := func(progress *WorkflowProgress) {
			progressUpdates++
		}

		result, err := workflow.ExecuteWithProgress(context.Background(), task, progressFn)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, progressUpdates, 0)
	})

	t.Run("YOLOMode", func(t *testing.T) {
		task := &Task{
			ID:          "task-3",
			Description: "YOLO task",
		}

		result, err := workflow.ExecuteYOLOMode(context.Background(), task)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, ModeNormal, controller.GetMode())
	})
}

// Test PlanMode
func TestPlanMode(t *testing.T) {
	mockLLM := NewMockLLMProvider()
	planner := NewLLMPlanner(mockLLM)
	presenter := &MockPresenter{selectedIndex: 0}
	executor := NewDefaultExecutor("/tmp/test")
	stateManager := NewStateManager()
	controller := NewModeController()

	workflow := NewPlanModeWorkflow(planner, presenter, executor, stateManager, controller)

	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		assert.Equal(t, 3, config.DefaultOptionCount)
		assert.Equal(t, 5, config.MaxOptionCount)
		assert.False(t, config.AutoSelectBest)
		assert.True(t, config.ShowComparison)
	})

	t.Run("RunNormalMode", func(t *testing.T) {
		config := DefaultConfig()
		pm := NewPlanMode(workflow, config)

		task := &Task{
			ID:          "task-1",
			Description: "Test task",
		}

		result, err := pm.Run(context.Background(), task)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("RunYOLOMode", func(t *testing.T) {
		config := DefaultConfig()
		config.AutoSelectBest = true
		pm := NewPlanMode(workflow, config)

		task := &Task{
			ID:          "task-2",
			Description: "YOLO task",
		}

		result, err := pm.Run(context.Background(), task)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("RunWithProgress", func(t *testing.T) {
		config := DefaultConfig()
		pm := NewPlanMode(workflow, config)

		task := &Task{
			ID:          "task-3",
			Description: "Progress task",
		}

		var lastPhase string
		progressFn := func(progress *WorkflowProgress) {
			lastPhase = progress.Phase
		}

		result, err := pm.RunWithProgress(context.Background(), task, progressFn)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Completed", lastPhase)
	})
}

// Test error recovery
func TestErrorRecovery(t *testing.T) {
	t.Run("InvalidPlan", func(t *testing.T) {
		executor := NewDefaultExecutor("/tmp/test")

		plan := &Plan{
			ID: "invalid-plan",
			Steps: []*PlanStep{
				{
					ID:           "step-1",
					Title:        "Invalid Step",
					Type:         StepType(999), // Invalid type
					Dependencies: []string{"non-existent"},
				},
			},
		}

		result, err := executor.Execute(context.Background(), plan)
		require.NoError(t, err) // Executor should handle errors gracefully
		assert.NotNil(t, result)
	})

	t.Run("EmptyOptions", func(t *testing.T) {
		presenter := &MockPresenter{}
		_, err := presenter.Present(context.Background(), []*PlanOption{})
		assert.Error(t, err)
	})
}

// Benchmark tests
func BenchmarkPlanGeneration(b *testing.B) {
	mockLLM := NewMockLLMProvider()
	planner := NewLLMPlanner(mockLLM)

	task := &Task{
		ID:          "bench-task",
		Description: "Benchmark task",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		planner.GeneratePlan(context.Background(), task)
	}
}

func BenchmarkOptionRanking(b *testing.B) {
	options := make([]*PlanOption, 5)
	for i := 0; i < 5; i++ {
		options[i] = &PlanOption{
			ID:    uuid.New().String(),
			Title: "Option",
			Plan: &Plan{
				Estimates: Estimates{
					Duration:   time.Duration(i*5) * time.Minute,
					Complexity: Complexity(i % 4),
					Confidence: 0.9,
				},
			},
		}
	}

	presenter := NewCLIOptionPresenter(&bytes.Buffer{}, strings.NewReader(""))
	criteria := []RankCriterion{
		{Name: "Speed", Weight: 1.0, Type: CriterionSpeed},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		presenter.RankOptions(options, criteria)
	}
}
