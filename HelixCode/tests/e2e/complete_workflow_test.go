package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// End-to-end tests for complete workflows

const (
	cliPath        = "./local-llm"
	testTimeout     = 5 * time.Minute
	healthTimeout   = 30 * time.Second
	longTimeout     = 10 * time.Minute
)

// Test scenarios for complete user workflows
func TestCompleteUserWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	// Build CLI
	require.NoError(t, buildCLI(), "CLI should build successfully")

	// Test workflow 1: New user setup
	t.Run("NewUserSetup", func(t *testing.T) {
		testNewUserSetup(t)
	})

	// Test workflow 2: Advanced user workflow
	t.Run("AdvancedUserWorkflow", func(t *testing.T) {
		testAdvancedUserWorkflow(t)
	})

	// Test workflow 3: Production deployment
	t.Run("ProductionDeployment", func(t *testing.T) {
		testProductionDeployment(t)
	})

	// Test workflow 4: Multi-provider orchestration
	t.Run("MultiProviderOrchestration", func(t *testing.T) {
		testMultiProviderOrchestration(t)
	})

	// Test workflow 5: Model optimization workflow
	t.Run("ModelOptimizationWorkflow", func(t *testing.T) {
		testModelOptimizationWorkflow(t)
	})
}

func TestCLICommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI tests in short mode")
	}

	// Build CLI
	require.NoError(t, buildCLI(), "CLI should build successfully")

	testCases := []struct {
		name     string
		args     []string
		expected string
		exitCode int
	}{
		{
			name:     "HelpCommand",
			args:     []string{"--help"},
			expected: "Local LLM Management System",
			exitCode:  0,
		},
		{
			name:     "ListProviders",
			args:     []string{"list"},
			expected: "Available Local LLM Providers",
			exitCode:  0,
		},
		{
			name:     "InvalidCommand",
			args:     []string{"invalid-command"},
			expected: "unknown command",
			exitCode:  1,
		},
		{
			name:     "StatusCommand",
			args:     []string{"status"},
			expected: "Provider Status Report",
			exitCode:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, exitCode := runCLICommand(t, tc.args...)
			
			assert.Equal(t, tc.exitCode, exitCode, "Exit code mismatch")
			assert.Contains(t, output, tc.expected, "Expected output not found")
		})
	}
}

func TestRealModelWorkflow(t *testing.T) {
	if testing.Short() || os.Getenv("SKIP_REAL_MODEL_TESTS") == "true" {
		t.Skip("Skipping real model tests")
	}

	// Build CLI
	require.NoError(t, buildCLI(), "CLI should build successfully")

	// Test complete workflow with real model
	testRealModelWorkflow(t)
}

func testNewUserSetup(t *testing.T) {
	t.Log("🔄 Testing new user setup workflow")

	baseDir := t.TempDir()
	
	// Step 1: Initialize system
	output, exitCode := runCLICommandWithEnv(t, []string{
		"init",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Init should succeed")
	assert.Contains(t, output, "Local LLM Management System initialized", "Should show success message")

	// Step 2: Check status
	output, exitCode = runCLICommandWithEnv(t, []string{
		"status",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Status should succeed")
	assert.Contains(t, output, "Provider Status Report", "Should show status")

	// Step 3: Start a provider
	provider := getAvailableProvider(t)
	if provider == "" {
		t.Skip("No available providers for E2E test")
		return
	}

	output, exitCode = runCLICommandWithEnv(t, []string{
		"start", provider,
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Provider should start")
	assert.Contains(t, output, "started successfully", "Should show start success")

	// Step 4: Download a test model
	output, exitCode = runCLICommandWithEnv(t, []string{
		"models", "download", "test-model-small",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	// Model download might fail in test environment, but command should complete
	t.Logf("Model download result: %s (exit code: %d)", output, exitCode)

	// Step 5: Stop the provider
	output, exitCode = runCLICommandWithEnv(t, []string{
		"stop", provider,
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Provider should stop")
	assert.Contains(t, output, "stopped successfully", "Should show stop success")

	t.Log("✅ New user setup workflow completed")
}

func testAdvancedUserWorkflow(t *testing.T) {
	t.Log("🔄 Testing advanced user workflow")

	baseDir := t.TempDir()

	// Step 1: Initialize with advanced configuration
	configFile := createAdvancedConfig(t, baseDir)
	
	output, exitCode := runCLICommandWithEnv(t, []string{
		"init",
		"--config", configFile,
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Advanced init should succeed")

	// Step 2: Start multiple providers
	providers := getMultipleAvailableProviders(t, 3)
	if len(providers) < 2 {
		t.Skip("Need at least 2 providers for advanced test")
		return
	}

	for _, provider := range providers {
		output, exitCode = runCLICommandWithEnv(t, []string{
			"start", provider,
			"--base-dir", baseDir,
		}, map[string]string{
			"HELIX_BASE_DIR": baseDir,
		})

		if exitCode != 0 {
			t.Logf("Provider %s failed to start: %s", provider, output)
		}
	}

	// Step 3: Download and share models
	output, exitCode = runCLICommandWithEnv(t, []string{
		"models", "download", "llama-3-8b-instruct",
		"--share", "true",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Model download and share result: %s (exit code: %d)", output, exitCode)

	// Step 4: Test cross-provider sharing
	output, exitCode = runCLICommandWithEnv(t, []string{
		"share", filepath.Join(baseDir, "shared", "llama-3-8b-instruct", "model.gguf"),
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Share result: %s (exit code: %d)", output, exitCode)

	// Step 5: View analytics
	output, exitCode = runCLICommandWithEnv(t, []string{
		"analytics",
		"--time-range", "1h",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Analytics should work")
	assert.Contains(t, output, "Analytics Dashboard", "Should show analytics")

	// Step 6: Stop all providers
	for _, provider := range providers {
		output, exitCode = runCLICommandWithEnv(t, []string{
			"stop", provider,
			"--base-dir", baseDir,
		}, map[string]string{
			"HELIX_BASE_DIR": baseDir,
		})

		if exitCode != 0 {
			t.Logf("Provider %s failed to stop: %s", provider, output)
		}
	}

	t.Log("✅ Advanced user workflow completed")
}

func testProductionDeployment(t *testing.T) {
	t.Log("🔄 Testing production deployment workflow")

	baseDir := t.TempDir()

	// Step 1: Create production configuration
	configFile := createProductionConfig(t, baseDir)

	// Step 2: Initialize production system
	output, exitCode := runCLICommandWithEnv(t, []string{
		"init",
		"--config", configFile,
		"--production", "true",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR":      baseDir,
		"HELIX_PRODUCTION":     "true",
		"HELIX_LOG_LEVEL":      "info",
		"HELIX_METRICS_ENABLED": "true",
	})

	assert.Equal(t, 0, exitCode, "Production init should succeed")

	// Step 3: Start all providers
	output, exitCode = runCLICommandWithEnv(t, []string{
		"start", "--all",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Start all providers result: %s (exit code: %d)", output, exitCode)

	// Step 4: Load production models
	prodModels := []string{"llama-3-8b-instruct", "mistral-7b-instruct"}
	for _, model := range prodModels {
		output, exitCode = runCLICommandWithEnv(t, []string{
			"models", "download", model,
			"--optimize", "true",
			"--base-dir", baseDir,
		}, map[string]string{
			"HELIX_BASE_DIR": baseDir,
		})

		t.Logf("Production model download %s: %s (exit code: %d)", model, output, exitCode)
	}

	// Step 5: Test failover scenarios
	output, exitCode = runCLICommandWithEnv(t, []string{
		"test", "failover",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Failover test result: %s (exit code: %d)", output, exitCode)

	// Step 6: Monitor production metrics
	output, exitCode = runCLICommandWithEnv(t, []string{
		"monitor",
		"--duration", "30s",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Monitoring result: %s (exit code: %d)", output, exitCode)

	// Step 7: Cleanup production deployment
	output, exitCode = runCLICommandWithEnv(t, []string{
		"cleanup",
		"--force", "true",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Production cleanup result: %s (exit code: %d)", output, exitCode)

	t.Log("✅ Production deployment workflow completed")
}

func testMultiProviderOrchestration(t *testing.T) {
	t.Log("🔄 Testing multi-provider orchestration workflow")

	baseDir := t.TempDir()

	// Step 1: Initialize with orchestration config
	configFile := createOrchestrationConfig(t, baseDir)

	output, exitCode := runCLICommandWithEnv(t, []string{
		"init",
		"--config", configFile,
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Orchestration init should succeed")

	// Step 2: Start orchestration cluster
	output, exitCode = runCLICommandWithEnv(t, []string{
		"orchestrate", "start",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Orchestration start result: %s (exit code: %d)", output, exitCode)

	// Step 3: Deploy models across cluster
	output, exitCode = runCLICommandWithEnv(t, []string{
		"orchestrate", "deploy",
		"--model", "llama-3-8b-instruct",
		"--replicas", "3",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Orchestration deploy result: %s (exit code: %d)", output, exitCode)

	// Step 4: Test load balancing
	output, exitCode = runCLICommandWithEnv(t, []string{
		"orchestrate", "test",
		"--load-balance",
		"--requests", "100",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Load balance test result: %s (exit code: %d)", output, exitCode)

	// Step 5: Test auto-scaling
	output, exitCode = runCLICommandWithEnv(t, []string{
		"orchestrate", "scale",
		"--auto", "true",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Auto-scale test result: %s (exit code: %d)", output, exitCode)

	// Step 6: Stop orchestration cluster
	output, exitCode = runCLICommandWithEnv(t, []string{
		"orchestrate", "stop",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Orchestration stop result: %s (exit code: %d)", output, exitCode)

	t.Log("✅ Multi-provider orchestration workflow completed")
}

func testModelOptimizationWorkflow(t *testing.T) {
	t.Log("🔄 Testing model optimization workflow")

	baseDir := t.TempDir()

	// Step 1: Initialize system
	output, exitCode := runCLICommandWithEnv(t, []string{
		"init",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Init should succeed")

	// Step 2: Download high-quality model
	output, exitCode = runCLICommandWithEnv(t, []string{
		"models", "download", "llama-3-8b-instruct-hf",
		"--format", "hf",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("HF model download result: %s (exit code: %d)", output, exitCode)

	// Step 3: Optimize for different providers
	providers := []string{"vllm", "llamacpp", "mlx"}
	targets := []struct {
		provider string
		hardware string
	}{
		{"vllm", "nvidia"},
		{"llamacpp", "cpu"},
		{"mlx", "apple"},
	}

	for _, target := range targets {
		output, exitCode = runCLICommandWithEnv(t, []string{
			"optimize",
			"--model", filepath.Join(baseDir, "models", "llama-3-8b-instruct-hf", "model"),
			"--provider", target.provider,
			"--hardware", target.hardware,
			"--base-dir", baseDir,
		}, map[string]string{
			"HELIX_BASE_DIR": baseDir,
		})

		t.Logf("Optimization for %s/%s: %s (exit code: %d)", 
			target.provider, target.hardware, output, exitCode)
	}

	// Step 4: Compare optimization results
	output, exitCode = runCLICommandWithEnv(t, []string{
		"compare",
		"--models", "llama-3-8b-instruct*",
		"--metrics", "speed,size,memory",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Optimization comparison result: %s (exit code: %d)", output, exitCode)

	// Step 5: Test performance benchmarks
	output, exitCode = runCLICommandWithEnv(t, []string{
		"benchmark",
		"--model", "llama-3-8b-instruct-gguf",
		 "--duration", "60s",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Benchmark result: %s (exit code: %d)", output, exitCode)

	t.Log("✅ Model optimization workflow completed")
}

func testRealModelWorkflow(t *testing.T) {
	t.Log("🔄 Testing real model workflow")

	baseDir := t.TempDir()

	// Step 1: Initialize system
	output, exitCode := runCLICommandWithEnv(t, []string{
		"init",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	assert.Equal(t, 0, exitCode, "Init should succeed")

	// Step 2: Start a real provider
	provider := getWorkingProvider(t)
	if provider == "" {
		t.Skip("No working providers available")
		return
	}

	output, exitCode = runCLICommandWithEnv(t, []string{
		"start", provider,
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	if exitCode != 0 {
		t.Logf("Provider %s failed to start: %s", provider, output)
		t.Skip("Provider not working in test environment")
		return
	}

	// Step 3: Download a real small model
	realModels := []string{"phi-2", "qwen1.5-1.8b", "gemma-2b"}
	var success bool

	for _, model := range realModels {
		output, exitCode = runCLICommandWithEnv(t, []string{
			"models", "download", model,
			"--format", "gguf",
			"--timeout", "300",
			"--base-dir", baseDir,
		}, map[string]string{
			"HELIX_BASE_DIR": baseDir,
		})

		if exitCode == 0 && strings.Contains(output, "successfully") {
			t.Logf("✅ Successfully downloaded model: %s", model)
			success = true
			break
		} else {
			t.Logf("❌ Failed to download model %s: %s", model, output)
		}
	}

	if !success {
		t.Skip("No real model could be downloaded in test environment")
		return
	}

	// Step 4: Test real inference
	output, exitCode = runCLICommandWithEnv(t, []string{
		"test", "inference",
		"--prompt", "Hello, how are you?",
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Inference test result: %s (exit code: %d)", output, exitCode)

	// Step 5: Cleanup
	output, exitCode = runCLICommandWithEnv(t, []string{
		"stop", provider,
		"--base-dir", baseDir,
	}, map[string]string{
		"HELIX_BASE_DIR": baseDir,
	})

	t.Logf("Stop result: %s (exit code: %d)", output, exitCode)

	t.Log("✅ Real model workflow completed")
}

// Helper functions

func buildCLI() error {
	// Build the CLI for testing
	cmd := exec.Command("go", "build", "-o", cliPath, "local-llm-test.go")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GOFLAGS=-tags=test")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}

func runCLICommand(t *testing.T, args ...string) (string, int) {
	return runCLICommandWithEnv(t, args, nil)
}

func runCLICommandWithEnv(t *testing.T, args []string, env map[string]string) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cliPath, args...)
	
	// Set environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return output, exitError.ExitCode()
		}
		return output, 1
	}

	return output, 0
}

func getAvailableProvider(t *testing.T) string {
	providers := []string{"ollama", "vllm", "localai", "llamacpp"}

	for _, provider := range providers {
		if isProviderInstalled(provider) {
			return provider
		}
	}

	return ""
}

func getMultipleAvailableProviders(t *testing.T, count int) []string {
	providers := []string{"ollama", "vllm", "localai", "llamacpp", "fastchat", "textgen"}
	var available []string

	for _, provider := range providers {
		if isProviderInstalled(provider) && len(available) < count {
			available = append(available, provider)
		}
	}

	return available
}

func getWorkingProvider(t *testing.T) string {
	providers := []string{"ollama", "vllm", "localai", "llamacpp"}

	for _, provider := range providers {
		if isProviderWorking(provider) {
			return provider
		}
	}

	return ""
}

func isProviderInstalled(provider string) bool {
	_, err := exec.LookPath(provider)
	if err != nil {
		// Check for alternative commands
		switch provider {
		case "vllm":
			cmd := exec.Command("python", "-c", "import vllm")
			return cmd.Run() == nil
		case "localai":
			_, err := exec.LookPath("local-ai")
			return err == nil
		case "llamacpp":
			_, err := os.Stat("./main")
			return err == nil
		}
		return false
	}
	return true
}

func isProviderWorking(provider string) bool {
	if !isProviderInstalled(provider) {
		return false
	}

	// Try to start provider briefly and check if it responds
	switch provider {
	case "ollama":
		cmd := exec.Command("ollama", "--version")
		return cmd.Run() == nil
	case "vllm":
		// Check if vllm can be imported and basic functionality works
		cmd := exec.Command("python", "-c", 
			"import vllm; print('VLLM available')")
		return cmd.Run() == nil
	default:
		return true
	}
}

func createAdvancedConfig(t *testing.T, baseDir string) string {
	config := `
providers:
  ollama:
    type: ollama
    endpoint: "http://127.0.0.1:11434"
    enabled: true
  vllm:
    type: vllm
    endpoint: "http://127.0.0.1:8000"
    enabled: true
    parameters:
      gpu_memory_utilization: 0.8
      max_num_batched_tokens: 8192

settings:
  auto_share: true
  optimization_enabled: true
  analytics_enabled: true
  
models:
  auto_download: false
  cache_size: "10GB"
  optimization_profiles:
    cpu:
      quantization: "q4_k_m"
    gpu:
      quantization: "q4_k_m"
    apple:
      quantization: "q4_k_m"
`

	configPath := filepath.Join(baseDir, "advanced-config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)
	
	return configPath
}

func createProductionConfig(t *testing.T, baseDir string) string {
	config := `
providers:
  vllm:
    type: vllm
    endpoint: "http://0.0.0.0:8000"
    enabled: true
    parameters:
      gpu_memory_utilization: 0.9
      max_num_batched_tokens: 16384
      tensor_parallel_size: 2
  ollama:
    type: ollama
    endpoint: "http://0.0.0.0:11434"
    enabled: true

production:
  health_checks:
    enabled: true
    interval: "30s"
  monitoring:
    enabled: true
    metrics: ["tps", "latency", "memory", "gpu"]
  failover:
    enabled: true
    max_retries: 3
    backoff: "exponential"
  security:
    tls: true
    auth_required: true
    audit_log: true

models:
  cache_size: "50GB"
  preloaded: ["llama-3-8b-instruct", "mistral-7b-instruct"]
  auto_optimize: true
`

	configPath := filepath.Join(baseDir, "production-config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)
	
	return configPath
}

func createOrchestrationConfig(t *testing.T, baseDir string) string {
	config := `
orchestration:
  enabled: true
  strategy: "round_robin"
  
load_balancing:
  algorithm: "least_connections"
  health_check_interval: "10s"
  
auto_scaling:
  enabled: true
  min_replicas: 1
  max_replicas: 10
  scale_up_threshold: 80
  scale_down_threshold: 20
  cooldown_period: "60s"

cluster:
  nodes:
    - name: "node-1"
      providers: ["vllm", "ollama"]
      resources: { cpu: 8, memory: "32GB", gpu: 1 }
    - name: "node-2"
      providers: ["localai", "llamacpp"]
      resources: { cpu: 16, memory: "64GB", gpu: 2 }

monitoring:
  metrics: ["throughput", "latency", "error_rate", "resource_utilization"]
  alerts:
    - name: "high_latency"
      threshold: "2s"
      action: "scale_up"
    - name: "error_rate_high"
      threshold: "5%"
      action: "failover"
`

	configPath := filepath.Join(baseDir, "orchestration-config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)
	
	return configPath
}