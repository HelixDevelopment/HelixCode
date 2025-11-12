package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// MultiAgentChallengeSolution demonstrates HelixCode multi-agent API integration
type MultiAgentChallengeSolution struct {
	BaseURL     string
	AuthToken   string
	ProjectID   string
	Coordinator *AgentCoordinator
}

// AgentCoordinator manages multiple specialized agents
type AgentCoordinator struct {
	Agents      []Agent
	TaskManager *TaskManager
}

// Agent represents a specialized AI agent
type Agent interface {
	GetCapabilities() []string
	CanHandle(task Task) bool
	Execute(task Task) (TaskResult, error)
}

// PlanningAgent handles project analysis and task breakdown
type PlanningAgent struct {
	ID           string
	Capabilities []string
}

// BuildingAgent handles code generation and integration
type BuildingAgent struct {
	ID           string
	Capabilities []string
}

// TestingAgent handles test execution and validation
type TestingAgent struct {
	ID           string
	Capabilities []string
}

// Task represents a work unit in the system
type Task struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type"`
	Status       string                 `json:"status"`
	Priority     string                 `json:"priority"`
	Parameters   map[string]interface{} `json:"parameters"`
	Dependencies []string               `json:"dependencies"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TaskResult represents the outcome of task execution
type TaskResult struct {
	TaskID    string                 `json:"task_id"`
	Status    string                 `json:"status"`
	Output    map[string]interface{} `json:"output"`
	Errors    []string               `json:"errors"`
	Timestamp time.Time              `json:"timestamp"`
}

// TaskManager handles task operations via HelixCode API
type TaskManager struct {
	BaseURL    string
	AuthToken  string
	HTTPClient *http.Client
}

// Checkpoint represents task state for work preservation
type Checkpoint struct {
	TaskID    string                 `json:"task_id"`
	State     map[string]interface{} `json:"state"`
	Progress  float64                `json:"progress"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewMultiAgentChallengeSolution creates a new challenge solution instance
func NewMultiAgentChallengeSolution(baseURL string) *MultiAgentChallengeSolution {
	return &MultiAgentChallengeSolution{
		BaseURL: baseURL,
		Coordinator: &AgentCoordinator{
			Agents: []Agent{
				&PlanningAgent{
					ID:           "planning-agent-1",
					Capabilities: []string{"analysis", "planning", "task-breakdown"},
				},
				&BuildingAgent{
					ID:           "building-agent-1",
					Capabilities: []string{"code-generation", "integration", "build"},
				},
				&TestingAgent{
					ID:           "testing-agent-1",
					Capabilities: []string{"testing", "validation", "quality-check"},
				},
			},
			TaskManager: &TaskManager{
				BaseURL:    baseURL,
				HTTPClient: &http.Client{Timeout: 30 * time.Second},
			},
		},
	}
}

// Run executes the complete multi-agent challenge
func (s *MultiAgentChallengeSolution) Run() error {
	log.Println("🚀 Starting Multi-Agent API Challenge Solution")

	// Step 1: Authentication
	if err := s.authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	// Step 2: Create project
	if err := s.createProject(); err != nil {
		return fmt.Errorf("project creation failed: %v", err)
	}

	// Step 3: Execute multi-agent workflow
	if err := s.executeMultiAgentWorkflow(); err != nil {
		return fmt.Errorf("multi-agent workflow failed: %v", err)
	}

	log.Println("✅ Multi-Agent API Challenge completed successfully!")
	return nil
}

// authenticate handles user registration and login
func (s *MultiAgentChallengeSolution) authenticate() error {
	log.Println("🔐 Authenticating with HelixCode API...")

	// Register new user
	registerData := map[string]interface{}{
		"username":     "challenge_user",
		"email":        "challenge@example.com",
		"password":     "secure_password_123",
		"display_name": "Challenge User",
	}

	registerBody, _ := json.Marshal(registerData)
	resp, err := http.Post(s.BaseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(registerBody))
	if err != nil {
		return fmt.Errorf("registration request failed: %v", err)
	}
	defer resp.Body.Close()

	// Login to get auth token
	loginData := map[string]interface{}{
		"username": "challenge_user",
		"password": "secure_password_123",
	}

	loginBody, _ := json.Marshal(loginData)
	resp, err = http.Post(s.BaseURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(loginBody))
	if err != nil {
		return fmt.Errorf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse login response: %v", err)
	}

	if result["status"] != "success" {
		return fmt.Errorf("login failed: %v", result["message"])
	}

	userData := result["user"].(map[string]interface{})
	token := result["token"].(string)

	s.AuthToken = token
	s.Coordinator.TaskManager.AuthToken = token

	log.Printf("✅ Authenticated as user: %s", userData["username"])
	return nil
}

// createProject creates a new project for the challenge
func (s *MultiAgentChallengeSolution) createProject() error {
	log.Println("📁 Creating challenge project...")

	projectData := map[string]interface{}{
		"name":        "Multi-Agent API Challenge",
		"description": "Demonstration of HelixCode multi-agent capabilities",
		"path":        "/challenges/multi-agent",
		"type":        "go",
	}

	body, _ := json.Marshal(projectData)
	req, err := http.NewRequest("POST", s.BaseURL+"/api/v1/projects", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Coordinator.TaskManager.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result["status"] != "success" {
		return fmt.Errorf("project creation failed: %v", result["message"])
	}

	project := result["project"].(map[string]interface{})
	s.ProjectID = project["id"].(string)

	log.Printf("✅ Project created with ID: %s", s.ProjectID)
	return nil
}

// executeMultiAgentWorkflow demonstrates multi-agent coordination
func (s *MultiAgentChallengeSolution) executeMultiAgentWorkflow() error {
	log.Println("🤖 Executing multi-agent workflow...")

	// Create tasks for different agents
	tasks := []Task{
		{
			Name:        "Requirements Analysis",
			Description: "Analyze project requirements and create task breakdown",
			Type:        "planning",
			Priority:    "high",
			Parameters: map[string]interface{}{
				"requirements": "Create a distributed multi-agent system with task coordination",
			},
		},
		{
			Name:        "System Architecture",
			Description: "Design system architecture and component interfaces",
			Type:        "planning",
			Priority:    "high",
			Parameters: map[string]interface{}{
				"components": []string{"coordinator", "planning-agent", "building-agent", "testing-agent"},
			},
		},
		{
			Name:        "Code Implementation",
			Description: "Implement core system components and agent logic",
			Type:        "building",
			Priority:    "normal",
			Dependencies: []string{"Requirements Analysis", "System Architecture"},
		},
		{
			Name:        "System Testing",
			Description: "Test multi-agent coordination and task execution",
			Type:        "testing",
			Priority:    "normal",
			Dependencies: []string{"Code Implementation"},
		},
	}

	// Execute tasks through coordinator
	for _, task := range tasks {
		log.Printf("📋 Processing task: %s", task.Name)

		// Create task via API
		createdTask, err := s.Coordinator.TaskManager.CreateTask(task)
		if err != nil {
			return fmt.Errorf("failed to create task %s: %v", task.Name, err)
		}

		// Assign to appropriate agent
		result, err := s.Coordinator.AssignTask(createdTask)
		if err != nil {
			return fmt.Errorf("failed to assign task %s: %v", task.Name, err)
		}

		log.Printf("✅ Task %s completed by agent: %s", task.Name, result.Status)

		// Simulate checkpointing for long-running tasks
		if task.Type == "building" || task.Type == "testing" {
			checkpoint := Checkpoint{
				TaskID:    createdTask.ID,
				State:     result.Output,
				Progress:  1.0,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"agent":       "challenge-solution",
					"workflow":    "multi-agent",
					"checkpointed": true,
				},
			}
			s.Coordinator.TaskManager.SaveCheckpoint(checkpoint)
		}

		// Small delay to simulate real processing
		time.Sleep(500 * time.Millisecond)
	}

	// Execute planning workflow
	log.Println("🔄 Executing planning workflow...")
	if err := s.executeWorkflow("planning"); err != nil {
		return fmt.Errorf("planning workflow failed: %v", err)
	}

	// Execute building workflow
	log.Println("🔄 Executing building workflow...")
	if err := s.executeWorkflow("building"); err != nil {
		return fmt.Errorf("building workflow failed: %v", err)
	}

	return nil
}

// executeWorkflow triggers a specific workflow type
func (s *MultiAgentChallengeSolution) executeWorkflow(workflowType string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/projects/%s/workflows/%s", s.BaseURL, s.ProjectID, workflowType), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.AuthToken)

	resp, err := s.Coordinator.TaskManager.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result["status"] != "success" {
		return fmt.Errorf("workflow execution failed: %v", result["message"])
	}

	log.Printf("✅ %s workflow executed successfully", workflowType)
	return nil
}

// AgentCoordinator methods

// AssignTask finds the best agent for a task and executes it
func (c *AgentCoordinator) AssignTask(task Task) (TaskResult, error) {
	for _, agent := range c.Agents {
		if agent.CanHandle(task) {
			log.Printf("🤖 Assigning task '%s' to agent %T", task.Name, agent)
			return agent.Execute(task)
		}
	}
	return TaskResult{}, fmt.Errorf("no suitable agent found for task: %s", task.Name)
}

// PlanningAgent implementation

func (a *PlanningAgent) GetCapabilities() []string {
	return a.Capabilities
}

func (a *PlanningAgent) CanHandle(task Task) bool {
	return task.Type == "planning"
}

func (a *PlanningAgent) Execute(task Task) (TaskResult, error) {
	log.Printf("📊 PlanningAgent executing: %s", task.Name)

	// Simulate planning work
	time.Sleep(1 * time.Second)

	result := TaskResult{
		TaskID:    task.ID,
		Status:    "completed",
		Output:    map[string]interface{}{"plan": "detailed_task_breakdown", "estimated_time": "2 hours"},
		Timestamp: time.Now(),
	}

	return result, nil
}

// BuildingAgent implementation

func (a *BuildingAgent) GetCapabilities() []string {
	return a.Capabilities
}

func (a *BuildingAgent) CanHandle(task Task) bool {
	return task.Type == "building"
}

func (a *BuildingAgent) Execute(task Task) (TaskResult, error) {
	log.Printf("🔨 BuildingAgent executing: %s", task.Name)

	// Simulate building work
	time.Sleep(2 * time.Second)

	result := TaskResult{
		TaskID:    task.ID,
		Status:    "completed",
		Output:    map[string]interface{}{"code_generated": true, "files_created": 5, "integration_complete": true},
		Timestamp: time.Now(),
	}

	return result, nil
}

// TestingAgent implementation

func (a *TestingAgent) GetCapabilities() []string {
	return a.Capabilities
}

func (a *TestingAgent) CanHandle(task Task) bool {
	return task.Type == "testing"
}

func (a *TestingAgent) Execute(task Task) (TaskResult, error) {
	log.Printf("🧪 TestingAgent executing: %s", task.Name)

	// Simulate testing work
	time.Sleep(1 * time.Second)

	result := TaskResult{
		TaskID:    task.ID,
		Status:    "completed",
		Output:    map[string]interface{}{"tests_passed": 15, "tests_failed": 0, "coverage": "85%"},
		Timestamp: time.Now(),
	}

	return result, nil
}

// TaskManager methods

// CreateTask creates a new task via HelixCode API
func (tm *TaskManager) CreateTask(task Task) (Task, error) {
	taskData := map[string]interface{}{
		"name":         task.Name,
		"description":  task.Description,
		"type":         task.Type,
		"priority":     task.Priority,
		"parameters":   task.Parameters,
		"dependencies": task.Dependencies,
	}

	body, _ := json.Marshal(taskData)
	req, err := http.NewRequest("POST", tm.BaseURL+"/api/v1/tasks", bytes.NewBuffer(body))
	if err != nil {
		return Task{}, err
	}

	req.Header.Set("Authorization", "Bearer "+tm.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := tm.HTTPClient.Do(req)
	if err != nil {
		return Task{}, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Task{}, err
	}

	if result["status"] != "success" {
		return Task{}, fmt.Errorf("task creation failed: %v", result["message"])
	}

	taskResult := result["task"].(map[string]interface{})
	createdTask := Task{
		ID:          taskResult["id"].(string),
		Name:        taskResult["name"].(string),
		Description: taskResult["description"].(string),
		Type:        taskResult["type"].(string),
		Status:      taskResult["status"].(string),
		CreatedAt:   time.Now(),
	}

	return createdTask, nil
}

// SaveCheckpoint simulates checkpoint saving (would use database in real implementation)
func (tm *TaskManager) SaveCheckpoint(checkpoint Checkpoint) error {
	log.Printf("💾 Saving checkpoint for task %s (progress: %.1f%%)", checkpoint.TaskID, checkpoint.Progress*100)
	// In a real implementation, this would save to the database
	return nil
}

// Main function to run the challenge solution
func main() {
	// Create and run the challenge solution
	solution := NewMultiAgentChallengeSolution("http://localhost:8080")
	
	if err := solution.Run(); err != nil {
		log.Fatalf("❌ Challenge solution failed: %v", err)
	}
	
	log.Println("🎉 Multi-Agent API Challenge Solution completed successfully!")
}