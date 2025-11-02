package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

// SSHWorkerPool manages SSH-based distributed workers
type SSHWorkerPool struct {
	workers     map[uuid.UUID]*SSHWorker
	mutex       sync.RWMutex
	autoInstall bool
}

// SSHWorker represents an SSH-accessible worker node
type SSHWorker struct {
	ID           uuid.UUID
	Hostname     string
	DisplayName  string
	SSHConfig    *SSHWorkerConfig
	Capabilities []string
	Resources    Resources
	Status       WorkerStatus
	HealthStatus WorkerHealth
	LastCheck    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	client       *ssh.Client
}

// SSHWorkerConfig represents SSH connection configuration for worker pool
type SSHWorkerConfig struct {
	Host       string
	Port       int
	Username   string
	PrivateKey string
	Password   string
	KeyPath    string
}

// NewSSHWorkerPool creates a new SSH worker pool
func NewSSHWorkerPool(autoInstall bool) *SSHWorkerPool {
	return &SSHWorkerPool{
		workers:     make(map[uuid.UUID]*SSHWorker),
		autoInstall: autoInstall,
	}
}

// AddWorker adds a new worker to the pool
func (p *SSHWorkerPool) AddWorker(ctx context.Context, worker *SSHWorker) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if worker == nil {
		return fmt.Errorf("worker is nil")
	}

	// Validate SSH configuration
	if err := p.validateSSHConfig(worker.SSHConfig); err != nil {
		return fmt.Errorf("invalid SSH config: %v", err)
	}

	// Test SSH connection
	if err := p.testSSHConnection(worker.SSHConfig); err != nil {
		return fmt.Errorf("SSH connection failed: %v", err)
	}

	// Auto-install Helix CLI if enabled
	if p.autoInstall {
		if err := p.installHelixCLI(ctx, worker); err != nil {
			log.Printf("Warning: Failed to auto-install Helix CLI on %s: %v", worker.Hostname, err)
		}
	}

	// Detect worker capabilities and resources
	if err := p.detectWorkerCapabilities(ctx, worker); err != nil {
		log.Printf("Warning: Failed to detect capabilities on %s: %v", worker.Hostname, err)
	}

	worker.ID = uuid.New()
	worker.CreatedAt = time.Now()
	worker.UpdatedAt = time.Now()
	worker.Status = WorkerStatusActive
	worker.HealthStatus = WorkerHealthHealthy

	p.workers[worker.ID] = worker
	log.Printf("SSH Worker added: %s (%s)", worker.Hostname, worker.ID)
	return nil
}

// RemoveWorker removes a worker from the pool
func (p *SSHWorkerPool) RemoveWorker(ctx context.Context, workerID uuid.UUID) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	worker, exists := p.workers[workerID]
	if !exists {
		return fmt.Errorf("worker not found: %s", workerID)
	}

	// Close SSH connection
	if worker.client != nil {
		worker.client.Close()
	}

	delete(p.workers, workerID)
	log.Printf("SSH Worker removed: %s (%s)", worker.Hostname, workerID)
	return nil
}

// ExecuteCommand executes a command on a worker
func (p *SSHWorkerPool) ExecuteCommand(ctx context.Context, workerID uuid.UUID, command string) (string, error) {
	p.mutex.RLock()
	worker, exists := p.workers[workerID]
	p.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("worker not found: %s", workerID)
	}

	// Ensure SSH connection
	if err := p.ensureSSHConnection(worker); err != nil {
		return "", fmt.Errorf("SSH connection failed: %v", err)
	}

	// Create session
	session, err := worker.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Execute command
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(command); err != nil {
		return "", fmt.Errorf("command execution failed: %v, stderr: %s", err, stderr.String())
	}

	worker.LastCheck = time.Now()
	return stdout.String(), nil
}

// HealthCheck performs health checks on all workers
func (p *SSHWorkerPool) HealthCheck(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	for _, worker := range p.workers {
		// Test SSH connection
		if err := p.testSSHConnection(worker.SSHConfig); err != nil {
			worker.HealthStatus = WorkerHealthUnhealthy
			worker.Status = WorkerStatusOffline
			log.Printf("Worker %s is unhealthy: %v", worker.Hostname, err)
		} else {
			worker.HealthStatus = WorkerHealthHealthy
			worker.Status = WorkerStatusActive
		}
		worker.UpdatedAt = now
	}

	return nil
}

// GetWorkerStats returns statistics about the worker pool
func (p *SSHWorkerPool) GetWorkerStats(ctx context.Context) *SSHWorkerStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := &SSHWorkerStats{
		TotalWorkers:   len(p.workers),
		ActiveWorkers:  0,
		HealthyWorkers: 0,
		TotalCPU:       0,
		TotalMemory:    0,
		TotalGPU:       0,
	}

	for _, worker := range p.workers {
		if worker.Status == WorkerStatusActive {
			stats.ActiveWorkers++
		}
		if worker.HealthStatus == WorkerHealthHealthy {
			stats.HealthyWorkers++
		}
		stats.TotalCPU += worker.Resources.CPUCount
		stats.TotalMemory += worker.Resources.TotalMemory
		stats.TotalGPU += worker.Resources.GPUCount
	}

	return stats
}

// SSHWorkerStats represents statistics about SSH workers
type SSHWorkerStats struct {
	TotalWorkers   int
	ActiveWorkers  int
	HealthyWorkers int
	TotalCPU       int
	TotalMemory    int64
	TotalGPU       int
}

// Helper methods

func (p *SSHWorkerPool) validateSSHConfig(config *SSHWorkerConfig) error {
	if config == nil {
		return fmt.Errorf("SSH config is required")
	}
	if config.Host == "" {
		return fmt.Errorf("host is required")
	}
	if config.Username == "" {
		return fmt.Errorf("username is required")
	}
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}
	return nil
}

func (p *SSHWorkerPool) testSSHConnection(config *SSHWorkerConfig) error {
	client, err := p.createSSHClient(config)
	if err != nil {
		return err
	}
	defer client.Close()

	// Test with a simple command
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return session.Run("echo 'SSH connection test successful'")
}

func (p *SSHWorkerPool) createSSHClient(config *SSHWorkerConfig) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// Add private key authentication
	if config.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(config.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Add key file authentication
	if config.KeyPath != "" {
		keyBytes, err := os.ReadFile(config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key file: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Add password authentication
	if config.Password != "" {
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication methods provided")
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
		Timeout:         30 * time.Second,
	}

	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), sshConfig)
}

func (p *SSHWorkerPool) ensureSSHConnection(worker *SSHWorker) error {
	if worker == nil {
		return fmt.Errorf("worker is nil")
	}

	if worker.client != nil {
		// Test if connection is still alive
		_, _, err := worker.client.SendRequest("keepalive@golang.org", true, nil)
		if err == nil {
			return nil
		}
		worker.client.Close()
	}

	client, err := p.createSSHClient(worker.SSHConfig)
	if err != nil {
		return err
	}
	worker.client = client
	return nil
}

func (p *SSHWorkerPool) installHelixCLI(ctx context.Context, worker *SSHWorker) error {
	// Check if Helix CLI is already installed
	output, err := p.ExecuteCommand(ctx, worker.ID, "which helix")
	if err == nil && output != "" {
		log.Printf("Helix CLI already installed on %s", worker.Hostname)
		return nil
	}

	// Installation script for Helix CLI
	installScript := `#!/bin/bash
set -e

# Download and install Helix CLI
curl -L https://github.com/helixdev/helix-cli/releases/latest/download/helix-linux-amd64 -o /tmp/helix
chmod +x /tmp/helix
sudo mv /tmp/helix /usr/local/bin/

# Verify installation
helix --version
`

	_, err = p.ExecuteCommand(ctx, worker.ID, installScript)
	if err != nil {
		return fmt.Errorf("failed to install Helix CLI: %v", err)
	}

	log.Printf("Helix CLI installed on %s", worker.Hostname)
	return nil
}

func (p *SSHWorkerPool) detectWorkerCapabilities(ctx context.Context, worker *SSHWorker) error {
	if worker == nil {
		return fmt.Errorf("worker is nil")
	}

	// Establish SSH connection for capability detection
	client, err := p.createSSHClient(worker.SSHConfig)
	if err != nil {
		return fmt.Errorf("failed to detect capabilities: %v", err)
	}
	defer client.Close()

	// Helper function to execute command on the client
	executeCommand := func(command string) (string, error) {
		session, err := client.NewSession()
		if err != nil {
			return "", err
		}
		defer session.Close()

		var stdout, stderr bytes.Buffer
		session.Stdout = &stdout
		session.Stderr = &stderr

		if err := session.Run(command); err != nil {
			return "", fmt.Errorf("command failed: %v, stderr: %s", err, stderr.String())
		}

		return stdout.String(), nil
	}

	// Detect CPU information
	cpuInfo, err := executeCommand("nproc")
	if err == nil && cpuInfo != "" {
		var cpuCount int
		fmt.Sscanf(cpuInfo, "%d", &cpuCount)
		worker.Resources.CPUCount = cpuCount
	}

	// Detect memory information
	memInfo, err := executeCommand("free -b | awk 'NR==2{print $2}'")
	if err == nil && memInfo != "" {
		var totalMemory int64
		fmt.Sscanf(memInfo, "%d", &totalMemory)
		worker.Resources.TotalMemory = totalMemory
	}

	// Detect GPU information
	gpuInfo, err := executeCommand("lspci | grep -i nvidia | wc -l")
	if err == nil && gpuInfo != "" {
		var gpuCount int
		fmt.Sscanf(gpuInfo, "%d", &gpuCount)
		worker.Resources.GPUCount = gpuCount
	}

	// Detect capabilities based on available tools
	capabilities := []string{"ssh-execution", "remote-computation"}

	// Check for LLM capabilities
	if _, err := executeCommand("which python3"); err == nil {
		capabilities = append(capabilities, "python-execution")
	}

	// Check for Docker
	if _, err := executeCommand("which docker"); err == nil {
		capabilities = append(capabilities, "docker-execution")
	}

	// Check for CUDA
	if _, err := executeCommand("which nvcc"); err == nil {
		capabilities = append(capabilities, "cuda-computation")
	}

	worker.Capabilities = capabilities
	return nil
}
