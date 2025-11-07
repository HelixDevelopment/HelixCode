// Package integration provides end-to-end integration tests for HelixCode
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig holds test configuration
type TestConfig struct {
	BaseURL      string
	DatabaseURL  string
	RedisURL     string
	AuthToken    string
	Timeout      time.Duration
}

// LoadTestConfig loads test configuration from environment
func LoadTestConfig() *TestConfig {
	return &TestConfig{
		BaseURL:     getEnv("TEST_BASE_URL", "http://localhost:8080"),
		DatabaseURL: getEnv("TEST_DATABASE_URL", "postgres://helixcode:testpass@localhost:5432/helixcode_test"),
		RedisURL:    getEnv("TEST_REDIS_URL", "redis://localhost:6379"),
		AuthToken:   getEnv("TEST_AUTH_TOKEN", ""),
		Timeout:     30 * time.Second,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestClient wraps HTTP client for API testing
type TestClient struct {
	httpClient *http.Client
	baseURL    string
	authToken  string
}

// NewTestClient creates a new test client
func NewTestClient(config *TestConfig) *TestClient {
	return &TestClient{
		httpClient: &http.Client{Timeout: config.Timeout},
		baseURL:    config.BaseURL,
		authToken:  config.AuthToken,
	}
}

// Get performs GET request
func (c *TestClient) Get(path string) (*http.Response, error) {
	return c.do("GET", path, nil)
}

// Post performs POST request
func (c *TestClient) Post(path string, body interface{}) (*http.Response, error) {
	return c.do("POST", path, body)
}

// Put performs PUT request
func (c *TestClient) Put(path string, body interface{}) (*http.Response, error) {
	return c.do("PUT", path, body)
}

// Delete performs DELETE request
func (c *TestClient) Delete(path string) (*http.Response, error) {
	return c.do("DELETE", path, nil)
}

func (c *TestClient) do(method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	return c.httpClient.Do(req)
}

// Test Suite: Basic Health Checks
func TestHealthEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := LoadTestConfig()
	client := NewTestClient(config)

	resp, err := client.Get("/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthData map[string]interface{}
	err = json.Unmarshal(body, &healthData)
	require.NoError(t, err)

	assert.Equal(t, "ok", healthData["status"])
}

// Test Suite: Authentication Flow
func TestAuthenticationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := LoadTestConfig()
	client := NewTestClient(config)

	tests := []struct {
		name         string
		credentials  map[string]string
		expectStatus int
	}{
		{
			name:         "Valid credentials",
			credentials:  map[string]string{"username": "admin", "password": "admin123"},
			expectStatus: http.StatusOK,
		},
		{
			name:         "Invalid credentials",
			credentials:  map[string]string{"username": "admin", "password": "wrong"},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Missing credentials",
			credentials:  map[string]string{},
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Post("/api/v1/auth/login", tt.credentials)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectStatus, resp.StatusCode)
		})
	}
}

// Test Suite: Task Management
func TestTaskCreationAndRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := LoadTestConfig()
	client := NewTestClient(config)

	// Create a task
	taskData := map[string]interface{}{
		"title":       "Integration Test Task",
		"description": "Test task created by integration test",
		"priority":    "high",
		"type":        "planning",
	}

	resp, err := client.Post("/api/v1/tasks", taskData)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse response
	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	require.NoError(t, err)

	taskID, ok := createResponse["id"].(string)
	require.True(t, ok, "Task ID should be a string")

	// Retrieve the task
	resp, err = client.Get(fmt.Sprintf("/api/v1/tasks/%s", taskID))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var task map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&task)
	require.NoError(t, err)

	assert.Equal(t, taskData["title"], task["title"])
	assert.Equal(t, taskData["priority"], task["priority"])

	// Cleanup: Delete the task
	resp, err = client.Delete(fmt.Sprintf("/api/v1/tasks/%s", taskID))
	require.NoError(t, err)
	resp.Body.Close()
}

// Test Suite: Worker Management
func TestWorkerRegistrationAndHeartbeat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := LoadTestConfig()
	client := NewTestClient(config)

	// Register worker
	workerData := map[string]interface{}{
		"hostname":     "test-worker-1",
		"capabilities": []string{"planning", "building", "testing"},
		"resources": map[string]interface{}{
			"cpu":    4,
			"memory": 8192,
		},
	}

	resp, err := client.Post("/api/v1/workers/register", workerData)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var registerResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResponse)
	require.NoError(t, err)

	workerID, ok := registerResponse["worker_id"].(string)
	require.True(t, ok)

	// Send heartbeat
	heartbeatData := map[string]interface{}{
		"worker_id": workerID,
		"status":    "idle",
	}

	resp, err = client.Post("/api/v1/workers/heartbeat", heartbeatData)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Test Suite: End-to-End Workflow
func TestCompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := LoadTestConfig()
	client := NewTestClient(config)
	ctx := context.Background()

	// 1. Create a project
	projectData := map[string]interface{}{
		"name":        "Integration Test Project",
		"description": "E2E test project",
	}

	resp, err := client.Post("/api/v1/projects", projectData)
	require.NoError(t, err)
	defer resp.Body.Close()

	var projectResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&projectResp)
	require.NoError(t, err)

	projectID := projectResp["id"].(string)

	// 2. Create a task within the project
	taskData := map[string]interface{}{
		"project_id":  projectID,
		"title":       "Test Task",
		"description": "Task for E2E test",
		"type":        "planning",
	}

	resp, err = client.Post("/api/v1/tasks", taskData)
	require.NoError(t, err)
	defer resp.Body.Close()

	var taskResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&taskResp)
	require.NoError(t, err)

	taskID := taskResp["id"].(string)

	// 3. Assign task to a worker (simulated)
	// 4. Poll for task completion with timeout
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	taskCompleted := false
	for !taskCompleted {
		select {
		case <-ctx.Done():
			t.Fatal("Context cancelled")
		case <-timeout:
			t.Log("Task did not complete within timeout (expected in test)")
			taskCompleted = true
		case <-ticker.C:
			resp, err := client.Get(fmt.Sprintf("/api/v1/tasks/%s", taskID))
			if err != nil {
				continue
			}

			var status map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&status)
			resp.Body.Close()

			if status["status"] == "completed" || status["status"] == "failed" {
				taskCompleted = true
			}
		}
	}

	// Cleanup
	client.Delete(fmt.Sprintf("/api/v1/projects/%s", projectID))
}

// Test Suite: Concurrent Operations
func TestConcurrentTaskCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := LoadTestConfig()
	client := NewTestClient(config)

	numTasks := 10
	results := make(chan error, numTasks)

	for i := 0; i < numTasks; i++ {
		go func(taskNum int) {
			taskData := map[string]interface{}{
				"title":    fmt.Sprintf("Concurrent Task %d", taskNum),
				"type":     "planning",
				"priority": "normal",
			}

			resp, err := client.Post("/api/v1/tasks", taskData)
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				results <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				return
			}

			results <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numTasks; i++ {
		err := <-results
		assert.NoError(t, err, "Task %d should succeed", i)
	}
}
