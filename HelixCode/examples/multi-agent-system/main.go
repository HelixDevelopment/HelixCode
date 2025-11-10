package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"dev.helix.code/internal/agent"
	"dev.helix.code/internal/agent/task"
	"dev.helix.code/internal/agent/types"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/logging"
)

// MockLLMProvider is a simple mock for testing
type MockLLMProvider struct {
	models       []llm.ModelInfo
	generateFunc func(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error)
}

func (m *MockLLMProvider) GetType() llm.ProviderType {
	return llm.ProviderType("mock")
}

func (m *MockLLMProvider) GetName() string {
	return "mock"
}

func (m *MockLLMProvider) GetModels() []llm.ModelInfo {
	if m.models == nil {
		return []llm.ModelInfo{{Name: "test-model", Provider: "test"}}
	}
	return m.models
}

func (m *MockLLMProvider) GetCapabilities() []llm.ModelCapability {
	return []llm.ModelCapability{}
}

func (m *MockLLMProvider) Generate(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, request)
	}

	prompt := request.Messages[0].Content

	// Check if this is a planning request that needs JSON subtasks
	if strings.Contains(prompt, "Extract subtasks from the following plan") {
		return &llm.LLMResponse{Content: `[
  {
    "title": "Design User Model",
    "description": "Create user data structure with required fields",
    "type": "code_generation",
    "priority": 3,
    "estimated_duration_minutes": 30,
    "depends_on": [],
    "required_capabilities": ["code_generation"]
  },
  {
    "title": "Implement JWT Logic",
    "description": "Create JWT token generation and validation functions",
    "type": "code_generation",
    "priority": 3,
    "estimated_duration_minutes": 45,
    "depends_on": ["Design User Model"],
    "required_capabilities": ["code_generation"]
  },
  {
    "title": "Create Auth Endpoints",
    "description": "Implement login, logout, and refresh endpoints",
    "type": "code_generation",
    "priority": 2,
    "estimated_duration_minutes": 60,
    "depends_on": ["Implement JWT Logic"],
    "required_capabilities": ["code_generation"]
  },
  {
    "title": "Write Unit Tests",
    "description": "Create comprehensive unit tests for auth system",
    "type": "testing",
    "priority": 2,
    "estimated_duration_minutes": 40,
    "depends_on": ["Create Auth Endpoints"],
    "required_capabilities": ["test_generation"]
  },
  {
    "title": "Security Review",
    "description": "Audit code for security vulnerabilities",
    "type": "review",
    "priority": 1,
    "estimated_duration_minutes": 25,
    "depends_on": ["Write Unit Tests"],
    "required_capabilities": ["code_review"]
  }
]`}, nil
	}

	// For planning requests, return a structured plan
	if strings.Contains(prompt, "You are a software planning agent") {
		return &llm.LLMResponse{Content: `# User Authentication System Plan

## Analysis
The requirement is to implement secure user authentication with JWT tokens, password hashing, and session management. This needs to be production-ready, secure, and scalable.

## Key Technical Decisions
- Use bcrypt for password hashing
- JWT tokens with configurable expiration
- PostgreSQL for user storage
- Gin framework for API endpoints
- Comprehensive error handling and logging

## Subtasks Breakdown
1. Design User Model - Create user data structure
2. Implement JWT Logic - Token generation/validation
3. Create Auth Endpoints - Login/logout/refresh APIs
4. Write Unit Tests - Comprehensive test coverage
5. Security Review - Vulnerability assessment

## Risks and Mitigations
- Password security: Use strong hashing algorithms
- Token theft: Implement refresh token rotation
- Scalability: Design for horizontal scaling`}, nil
	}

	// For coding requests, return JSON with code and explanation
	if strings.Contains(prompt, "Generate code") || strings.Contains(prompt, "Create user model") || strings.Contains(prompt, "Implement JWT") || strings.Contains(prompt, "Create login, logout") {
		return &llm.LLMResponse{Content: `{
  "code": "// Generated authentication code\npackage auth\n\nimport (\n\t\"time\"\n\t\"github.com/golang-jwt/jwt/v4\"\n\t\"golang.org/x/crypto/bcrypt\"\n)\n\ntype User struct {\n\tID        int       ` + "`" + `json:\"id\"` + "`" + `\n\tEmail     string    ` + "`" + `json:\"email\"` + "`" + `\n\tPassword  string    ` + "`" + `json:\"password_hash\"` + "`" + `\n\tCreatedAt time.Time ` + "`" + `json:\"created_at\"` + "`" + `\n\tUpdatedAt time.Time ` + "`" + `json:\"updated_at\"` + "`" + `\n}\n\nfunc HashPassword(password string) (string, error) {\n\tbytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)\n\treturn string(bytes), err\n}\n\nfunc CheckPasswordHash(password, hash string) bool {\n\terr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))\n\treturn err == nil\n}\n\nfunc GenerateJWT(userID int, secret string) (string, error) {\n\ttoken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{\n\t\t\"user_id\": userID,\n\t\t\"exp\":     time.Now().Add(time.Hour * 24).Unix(),\n\t})\n\treturn token.SignedString([]byte(secret))\n}\n\nfunc ValidateJWT(tokenString, secret string) (*jwt.Token, error) {\n\treturn jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {\n\t\treturn []byte(secret), nil\n\t})\n}",
  "explanation": "Generated complete authentication system with user model, password hashing, and JWT token management"
}`}, nil
	}

	// For testing requests, return JSON with test code and cases
	if strings.Contains(prompt, "Write unit tests") || strings.Contains(prompt, "integration tests") || strings.Contains(prompt, "Generate comprehensive tests") {
		return &llm.LLMResponse{Content: `{
  "test_code": "// Generated test code\npackage auth\n\nimport (\n\t\"testing\"\n\t\"time\"\n)\n\nfunc TestHashPassword(t *testing.T) {\n\tpassword := \"testpassword\"\n\thash, err := HashPassword(password)\n\tif err != nil {\n\t\tt.Fatalf(\"Error hashing password: %v\", err)\n\t}\n\tif !CheckPasswordHash(password, hash) {\n\t\tt.Error(\"Password hash check failed\")\n\t}\n}\n\nfunc TestGenerateJWT(t *testing.T) {\n\tuserID := 1\n\tsecret := \"testsecret\"\n\ttoken, err := GenerateJWT(userID, secret)\n\tif err != nil {\n\t\tt.Fatalf(\"Error generating JWT: %v\", err)\n\t}\n\tif token == \"\" {\n\t\tt.Error(\"Generated token is empty\")\n\t}\n}\n\nfunc TestValidateJWT(t *testing.T) {\n\tuserID := 1\n\tsecret := \"testsecret\"\n\ttoken, err := GenerateJWT(userID, secret)\n\tif err != nil {\n\t\tt.Fatalf(\"Error generating JWT: %v\", err)\n\t}\n\n\tparsedToken, err := ValidateJWT(token, secret)\n\tif err != nil {\n\t\tt.Fatalf(\"Error validating JWT: %v\", err)\n\t}\n\tif !parsedToken.Valid {\n\t\tt.Error(\"Token is not valid\")\n\t}\n}",
  "test_cases": ["TestHashPassword", "TestGenerateJWT", "TestValidateJWT"]
}`}, nil
	}

	// For review requests, return JSON review report
	if strings.Contains(prompt, "Review code for security") || strings.Contains(prompt, "security vulnerabilities") || strings.Contains(prompt, "comprehensive code review") {
		return &llm.LLMResponse{Content: `{
  "review_summary": "Security and code quality review completed. Code is production-ready with minor recommendations.",
  "issues": [
    {
      "severity": "medium",
      "type": "security",
      "description": "JWT secret should be environment variable, not hardcoded",
      "line_number": 0,
      "recommendation": "Move JWT secret to environment configuration"
    },
    {
      "severity": "low",
      "type": "performance",
      "description": "Consider connection pooling for database queries",
      "line_number": 0,
      "recommendation": "Implement database connection pooling"
    }
  ],
  "suggestions": [
    "Add rate limiting to authentication endpoints",
    "Implement refresh token rotation",
    "Add comprehensive input sanitization",
    "Consider adding request ID tracing"
  ],
  "metrics": {
    "lines_reviewed": 45,
    "issues_found": 2,
    "security_score": 85,
    "maintainability_score": 90
  }
}`}, nil
	}

	return &llm.LLMResponse{Content: "Mock response: Task completed successfully"}, nil
}

func (m *MockLLMProvider) GenerateStream(ctx context.Context, request *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	return nil
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

// WorkflowExecutionResult represents the result of workflow execution
type WorkflowExecutionResult struct {
	Status       string
	StepResults  map[string]*task.Result
	SuccessCount int
	FailureCount int
}

// AgentStats represents agent performance statistics
type AgentStats struct {
	AgentPerformance map[string]AgentPerf
}

type AgentPerf struct {
	TasksCompleted int
	SuccessRate    float64
}

func main() {
	fmt.Println("🚀 HelixCode Multi-Agent System Demo")
	fmt.Println("====================================")

	// Initialize logging
	logger := logging.NewLogger(logging.INFO)

	// Create mock LLM provider
	mockProvider := &MockLLMProvider{}

	// Create mock tool registry with basic tools
	mockFSWrite := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"status": "success", "path": params["path"]}, nil
	}
	mockFSRead := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"content": "mock file content"}, nil
	}
	mockShell := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"status": "success", "output": "mock command output"}, nil
	}

	toolRegistry := types.CreateMockToolRegistry(mockFSRead, mockFSWrite, mockShell)
	convertedToolRegistry := types.ConvertToToolRegistry(toolRegistry)

	// Create agent registry
	registry := agent.NewAgentRegistry()

	// Create specialized agents
	ctx := context.Background()

	// Planning Agent
	planningAgent, err := types.NewPlanningAgent(&agent.AgentConfig{
		ID:           "planning-001",
		Type:         agent.AgentTypePlanning,
		Name:         "Strategic Planner",
		Capabilities: []agent.Capability{agent.CapabilityPlanning, agent.CapabilityCodeAnalysis},
	}, mockProvider)
	if err != nil {
		log.Fatalf("Failed to create planning agent: %v", err)
	}
	if err := planningAgent.Initialize(ctx, nil); err != nil {
		log.Fatalf("Failed to initialize planning agent: %v", err)
	}
	registry.Register(planningAgent)

	// Coding Agent
	codingAgent, err := types.NewCodingAgent(&agent.AgentConfig{
		ID:           "coding-001",
		Type:         agent.AgentTypeCoding,
		Name:         "Code Architect",
		Capabilities: []agent.Capability{agent.CapabilityCodeGeneration, agent.CapabilityCodeAnalysis},
	}, mockProvider, convertedToolRegistry)
	if err != nil {
		log.Fatalf("Failed to create coding agent: %v", err)
	}
	if err := codingAgent.Initialize(ctx, nil); err != nil {
		log.Fatalf("Failed to initialize coding agent: %v", err)
	}
	registry.Register(codingAgent)

	// Testing Agent
	testingAgent, err := types.NewTestingAgent(&agent.AgentConfig{
		ID:           "testing-001",
		Type:         agent.AgentTypeTesting,
		Name:         "Quality Assurance",
		Capabilities: []agent.Capability{agent.CapabilityTestGeneration, agent.CapabilityTestExecution},
	}, mockProvider, convertedToolRegistry)
	if err != nil {
		log.Fatalf("Failed to create testing agent: %v", err)
	}
	if err := testingAgent.Initialize(ctx, nil); err != nil {
		log.Fatalf("Failed to initialize testing agent: %v", err)
	}
	registry.Register(testingAgent)

	// Review Agent
	reviewAgent, err := types.NewReviewAgent(&agent.AgentConfig{
		ID:           "review-001",
		Type:         agent.AgentTypeReview,
		Name:         "Code Reviewer",
		Capabilities: []agent.Capability{agent.CapabilityCodeReview, agent.CapabilitySecurityAudit},
	}, mockProvider, convertedToolRegistry)
	if err != nil {
		log.Fatalf("Failed to create review agent: %v", err)
	}
	if err := reviewAgent.Initialize(ctx, nil); err != nil {
		log.Fatalf("Failed to initialize review agent: %v", err)
	}
	registry.Register(reviewAgent)

	fmt.Printf("✅ Created %d specialized agents\n", len(registry.List()))

	// Create coordinator
	coordinator := agent.NewCoordinator(&agent.CoordinatorConfig{
		MaxConcurrentTasks:  3,
		TaskTimeout:         10 * time.Minute,
		EnableCollaboration: true,
		EnableResilience:    true,
	})

	// Register agents with coordinator
	for _, ag := range registry.List() {
		coordinator.RegisterAgent(ag)
	}

	fmt.Println("✅ Coordinator initialized with agents")

	// Create a complex workflow: "Implement User Authentication System"
	workflow := agent.NewWorkflow(
		"User Authentication System",
		"Complete implementation of user authentication with JWT tokens",
	)

	// Step 1: Planning - Analyze requirements and create technical specification
	workflow.AddStep(&agent.WorkflowStep{
		ID:           "plan-auth-system",
		Name:         "Plan Authentication System",
		AgentType:    agent.AgentTypePlanning,
		RequiredCaps: []agent.Capability{agent.CapabilityPlanning},
		Input: map[string]interface{}{
			"requirements": "Implement secure user authentication with JWT tokens, password hashing, and session management",
			"constraints":  "Must be production-ready, secure, and scalable",
		},
	})

	// Step 2: Coding - Create user model and database schema
	workflow.AddStep(&agent.WorkflowStep{
		ID:           "implement-user-model",
		Name:         "Implement User Model",
		AgentType:    agent.AgentTypeCoding,
		RequiredCaps: []agent.Capability{agent.CapabilityCodeGeneration},
		Input: map[string]interface{}{
			"requirements": "Create user model with fields: id, email, password_hash, created_at, updated_at",
		},
		DependsOn: []string{"plan-auth-system"},
	})

	// Step 3: Coding - Implement JWT authentication logic
	workflow.AddStep(&agent.WorkflowStep{
		ID:           "implement-jwt-auth",
		Name:         "Implement JWT Authentication",
		AgentType:    agent.AgentTypeCoding,
		RequiredCaps: []agent.Capability{agent.CapabilityCodeGeneration},
		Input: map[string]interface{}{
			"requirements": "Implement JWT token generation, validation, and refresh logic",
		},
		DependsOn: []string{"implement-user-model"},
	})

	// Step 4: Coding - Create authentication endpoints
	workflow.AddStep(&agent.WorkflowStep{
		ID:           "implement-auth-endpoints",
		Name:         "Implement Auth Endpoints",
		AgentType:    agent.AgentTypeCoding,
		RequiredCaps: []agent.Capability{agent.CapabilityCodeGeneration},
		Input: map[string]interface{}{
			"requirements": "Create login, logout, and token refresh API endpoints",
		},
		DependsOn: []string{"implement-jwt-auth"},
	})

	// Step 5: Testing - Write comprehensive tests
	workflow.AddStep(&agent.WorkflowStep{
		ID:           "write-auth-tests",
		Name:         "Write Authentication Tests",
		AgentType:    agent.AgentTypeTesting,
		RequiredCaps: []agent.Capability{agent.CapabilityTestGeneration},
		Input: map[string]interface{}{
			"requirements": "Write unit tests for user model, JWT logic, and integration tests for endpoints",
		},
		DependsOn: []string{"implement-user-model", "implement-jwt-auth", "implement-auth-endpoints"},
	})

	// Step 6: Review - Security and code quality audit
	workflow.AddStep(&agent.WorkflowStep{
		ID:           "security-review",
		Name:         "Security Review",
		AgentType:    agent.AgentTypeReview,
		RequiredCaps: []agent.Capability{agent.CapabilityCodeReview},
		Input: map[string]interface{}{
			"requirements": "Review code for security vulnerabilities, performance issues, and best practices",
		},
		DependsOn: []string{"implement-auth-endpoints", "write-auth-tests"},
	})

	fmt.Printf("✅ Created workflow with %d steps\n", len(workflow.Steps))

	// Execute the workflow
	fmt.Println("\n🎯 Executing Multi-Agent Workflow...")
	fmt.Println("=====================================")

	executor := agent.NewWorkflowExecutor(coordinator)

	startTime := time.Now()
	err = executor.Execute(ctx, workflow)
	duration := time.Since(startTime)

	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Create execution result from workflow
	result := &WorkflowExecutionResult{
		Status:      string(workflow.Status),
		StepResults: workflow.Results,
	}

	result.SuccessCount = 0
	result.FailureCount = 0
	for _, stepResult := range result.StepResults {
		if stepResult.Success {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	}

	// Display results
	fmt.Printf("\n🎉 Workflow Completed in %v\n", duration)
	fmt.Println("================================")

	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Total Steps: %d\n", len(result.StepResults))
	fmt.Printf("Successful Steps: %d\n", result.SuccessCount)
	fmt.Printf("Failed Steps: %d\n", result.FailureCount)

	// Show detailed results for each step
	fmt.Println("\n📋 Step Results:")
	fmt.Println("================")

	for stepID, stepResult := range result.StepResults {
		status := "✅"
		if !stepResult.Success {
			status = "❌"
		}

		fmt.Printf("%s %s (%s)\n", status, stepID, stepResult.AgentID)

		if stepResult.Success {
			if output, ok := stepResult.Output["summary"].(string); ok && output != "" {
				fmt.Printf("   Summary: %s\n", output)
			}
		} else {
			fmt.Printf("   Error: %v\n", stepResult.Error)
		}
		fmt.Println()
	}

	// Show agent performance
	fmt.Println("🏆 Agent Performance:")
	fmt.Println("====================")

	// Create simple stats
	stats := &AgentStats{
		AgentPerformance: make(map[string]AgentPerf),
	}

	for _, ag := range registry.List() {
		health := ag.Health()
		stats.AgentPerformance[ag.ID()] = AgentPerf{
			TasksCompleted: health.TaskCount,
			SuccessRate:    1.0 - health.ErrorRate,
		}
	}

	for agentID, perf := range stats.AgentPerformance {
		fmt.Printf("Agent %s: %d tasks completed, %.1f%% success rate\n",
			agentID, perf.TasksCompleted, perf.SuccessRate*100)
	}

	// Cleanup
	fmt.Println("\n🧹 Cleaning up...")
	for _, ag := range registry.List() {
		if err := ag.Shutdown(ctx); err != nil {
			logger.Warn("Error shutting down agent", "agent", ag.ID(), "error", err)
		}
	}

	fmt.Println("✅ Multi-Agent System Demo Complete!")
	fmt.Println("\nThis demo showed:")
	fmt.Println("• 4 specialized agents working together")
	fmt.Println("• Complex workflow with dependencies")
	fmt.Println("• Parallel execution where possible")
	fmt.Println("• Result aggregation and reporting")
	fmt.Println("• Agent performance tracking")
}
