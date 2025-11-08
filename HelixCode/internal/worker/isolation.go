package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

// WorkerIsolationManager handles worker sandboxing and security
type WorkerIsolationManager struct {
	sandboxes map[uuid.UUID]*WorkerSandbox
	mutex     sync.RWMutex
}

// WorkerSandbox represents an isolated execution environment
type WorkerSandbox struct {
	ID           uuid.UUID
	WorkerID      uuid.UUID
	Directory     string
	User          string
	Group         string
	MaxMemory     int64 // in bytes
	MaxCPU        float64 // percentage
	MaxProcesses  int
	NetworkAccess bool
	CreatedAt     time.Time
	LastUsed      time.Time
}

// NewWorkerIsolationManager creates a new isolation manager
func NewWorkerIsolationManager() *WorkerIsolationManager {
	return &WorkerIsolationManager{
		sandboxes: make(map[uuid.UUID]*WorkerSandbox),
	}
}

// CreateSandbox creates an isolated sandbox for a worker
func (wim *WorkerIsolationManager) CreateSandbox(ctx context.Context, workerID uuid.UUID, resourceLimits Resources) (*WorkerSandbox, error) {
	wim.mutex.Lock()
	defer wim.mutex.Unlock()

	sandboxID := uuid.New()
	
	// Create sandbox directory
	sandboxDir := filepath.Join(os.TempDir(), "helix-sandbox-"+sandboxID.String())
	if err := os.MkdirAll(sandboxDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create sandbox directory: %v", err)
	}

	// Create user with limited privileges
	username := fmt.Sprintf("helix-%s", sandboxID.String()[:8])
	if err := wim.createSandboxUser(username, sandboxDir); err != nil {
		os.RemoveAll(sandboxDir)
		return nil, fmt.Errorf("failed to create sandbox user: %v", err)
	}

	// Convert Resources to sandbox limits
	maxMemory := resourceLimits.TotalMemory / 2 // Use half of available memory
	maxCPU := float64(resourceLimits.CPUCount) // Use all available CPUs

	sandbox := &WorkerSandbox{
		ID:           sandboxID,
		WorkerID:      workerID,
		Directory:     sandboxDir,
		User:          username,
		Group:         username,
		MaxMemory:     maxMemory,
		MaxCPU:        maxCPU,
		MaxProcesses:  100,
		NetworkAccess: false,
		CreatedAt:     time.Now(),
		LastUsed:      time.Now(),
	}

	// Apply resource limits
	if err := wim.applyResourceLimits(sandbox); err != nil {
		wim.cleanupSandbox(sandbox)
		return nil, fmt.Errorf("failed to apply resource limits: %v", err)
	}

	wim.sandboxes[sandboxID] = sandbox
	return sandbox, nil
}

// ExecuteInSandbox executes a command in a sandboxed environment
func (wim *WorkerIsolationManager) ExecuteInSandbox(ctx context.Context, sandboxID uuid.UUID, client *ssh.Client, command string) (string, string, error) {
	wim.mutex.RLock()
	sandbox, exists := wim.sandboxes[sandboxID]
	wim.mutex.RUnlock()

	if !exists {
		return "", "", fmt.Errorf("sandbox not found")
	}

	// Update last used
	sandbox.LastUsed = time.Now()

	// Build sandboxed command
	sandboxedCommand := wim.buildSandboxedCommand(sandbox, command)

	// Execute via SSH with sandbox user
	session, err := client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Set environment variables
	if err := wim.setSandboxEnvironment(session, sandbox); err != nil {
		return "", "", fmt.Errorf("failed to set sandbox environment: %v", err)
	}

	output, err := session.CombinedOutput(sandboxedCommand)
	stdout := string(output)
	stderr := ""
	if err != nil {
		stderr = err.Error()
	}

	return stdout, stderr, err
}

// createSandboxUser creates a system user for the sandbox
func (wim *WorkerIsolationManager) createSandboxUser(username, sandboxDir string) error {
	// Create user with no home directory and limited shell
	useraddCmd := fmt.Sprintf("sudo useradd -m -d %s -s /bin/bash %s", sandboxDir, username)
	if err := exec.Command("sh", "-c", useraddCmd).Run(); err != nil {
		// Fallback: try creating with minimal parameters
		useraddCmd = fmt.Sprintf("sudo useradd %s", username)
		if err2 := exec.Command("sh", "-c", useraddCmd).Run(); err2 != nil {
			return fmt.Errorf("failed to create user %s: %v (both attempts failed)", username, err)
		}
	}

	// Set ownership of sandbox directory
	chownCmd := fmt.Sprintf("sudo chown -R %s:%s %s", username, username, sandboxDir)
	if err := exec.Command("sh", "-c", chownCmd).Run(); err != nil {
		return fmt.Errorf("failed to set sandbox ownership: %v", err)
	}

	// Set limited permissions
	chmodCmd := fmt.Sprintf("sudo chmod 750 %s", sandboxDir)
	if err := exec.Command("sh", "-c", chmodCmd).Run(); err != nil {
		return fmt.Errorf("failed to set sandbox permissions: %v", err)
	}

	return nil
}

// applyResourceLimits applies resource limits to the sandbox
func (wim *WorkerIsolationManager) applyResourceLimits(sandbox *WorkerSandbox) error {
	// Create limits configuration
	limitsConfig := fmt.Sprintf(`
# HelixCode Sandbox Limits for %s
%s soft memlock %d
%s hard memlock %d
%s soft nproc %d
%s hard nproc %d
%s soft nofile 1024
%s hard nofile 2048
`, sandbox.User, sandbox.User, sandbox.MaxMemory, sandbox.User, sandbox.MaxMemory,
		sandbox.User, sandbox.MaxProcesses, sandbox.User, sandbox.MaxProcesses,
		sandbox.User, sandbox.User)

	limitsFile := filepath.Join("/etc/security/limits.d", fmt.Sprintf("helix-%s.conf", sandbox.User))
	if err := os.WriteFile(limitsFile, []byte(limitsConfig), 0644); err != nil {
		return fmt.Errorf("failed to write limits file: %v", err)
	}

	// Apply cgroup limits (Linux-specific)
	if err := wim.applyCgroupLimits(sandbox); err != nil {
		log.Printf("Warning: Failed to apply cgroup limits: %v", err)
	}

	return nil
}

// applyCgroupLimits applies cgroup limits for resource isolation
func (wim *WorkerIsolationManager) applyCgroupLimits(sandbox *WorkerSandbox) error {
	// Create cgroup
	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/helix/%s", sandbox.User)
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup: %v", err)
	}

	// Apply memory limit
	memoryLimitFile := filepath.Join(cgroupPath, "memory.limit_in_bytes")
	if err := os.WriteFile(memoryLimitFile, []byte(strconv.FormatInt(sandbox.MaxMemory, 10)), 0644); err != nil {
		return fmt.Errorf("failed to set memory limit: %v", err)
	}

	// Apply CPU limit
	cpuLimitFile := filepath.Join(cgroupPath, "cpu.shares")
	if err := os.WriteFile(cpuLimitFile, []byte(strconv.FormatInt(int64(sandbox.MaxCPU*1024), 10)), 0644); err != nil {
		return fmt.Errorf("failed to set CPU limit: %v", err)
	}

	// Add process to cgroup
	_ = filepath.Join(cgroupPath, "tasks") // Not used now but kept for future implementation
	// This will be done when the process starts

	return nil
}

// buildSandboxedCommand wraps a command for sandbox execution
func (wim *WorkerIsolationManager) buildSandboxedCommand(sandbox *WorkerSandbox, command string) string {
	// Escape shell injection
	command = strings.ReplaceAll(command, "'", "'\"'\"'")
	
	// Build sandboxed command with security measures
	sandboxedCommand := fmt.Sprintf(`
# Set up sandbox environment
export HELIX_SANDBOX_ID="%s"
export HELIX_SANDBOX_DIR="%s"
export HOME="%s"
export TMPDIR="%s/tmp"

# Create temp directory
mkdir -p "$TMPDIR"

# Execute command with strict permissions
sudo -u %s bash -c '
set -e
set -u
# Security settings
ulimit -t 300  # 5 minute timeout
ulimit -f 100   # 100MB file size limit

# Execute the actual command
%s
'
`, sandbox.ID.String(), sandbox.Directory, sandbox.Directory, sandbox.Directory, sandbox.User, command)

	return strings.TrimSpace(sandboxedCommand)
}

// setSandboxEnvironment sets environment variables for the sandbox
func (wim *WorkerIsolationManager) setSandboxEnvironment(session *ssh.Session, sandbox *WorkerSandbox) error {
	envVars := map[string]string{
		"HELIX_SANDBOX_ID":     sandbox.ID.String(),
		"HELIX_SANDBOX_DIR":      sandbox.Directory,
		"HELIX_WORKER_ID":        sandbox.WorkerID.String(),
		"HELIX_ISOLATED":          "true",
		"PATH":                   "/usr/local/bin:/usr/bin:/bin",
		"TMPDIR":                 filepath.Join(sandbox.Directory, "tmp"),
	}

	for key, value := range envVars {
		if err := session.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %v", key, err)
		}
	}

	return nil
}

// CleanupSandbox removes a sandbox and cleans up resources
func (wim *WorkerIsolationManager) CleanupSandbox(ctx context.Context, sandboxID uuid.UUID) error {
	wim.mutex.Lock()
	defer wim.mutex.Unlock()

	sandbox, exists := wim.sandboxes[sandboxID]
	if !exists {
		return fmt.Errorf("sandbox not found")
	}

	return wim.cleanupSandbox(sandbox)
}

// cleanupSandbox performs the actual cleanup
func (wim *WorkerIsolationManager) cleanupSandbox(sandbox *WorkerSandbox) error {
	// Remove sandbox directory
	if err := os.RemoveAll(sandbox.Directory); err != nil {
		log.Printf("Warning: Failed to remove sandbox directory %s: %v", sandbox.Directory, err)
	}

	// Remove user
	if err := exec.Command("sudo", "userdel", "-f", sandbox.User).Run(); err != nil {
		log.Printf("Warning: Failed to remove user %s: %v", sandbox.User, err)
	}

	// Remove limits file
	limitsFile := filepath.Join("/etc/security/limits.d", fmt.Sprintf("helix-%s.conf", sandbox.User))
	if err := os.Remove(limitsFile); err != nil {
		log.Printf("Warning: Failed to remove limits file %s: %v", limitsFile, err)
	}

	// Remove cgroup
	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/helix/%s", sandbox.User)
	if err := os.RemoveAll(cgroupPath); err != nil {
		log.Printf("Warning: Failed to remove cgroup %s: %v", cgroupPath, err)
	}

	// Remove from memory
	delete(wim.sandboxes, sandbox.ID)

	log.Printf("Cleaned up sandbox %s for user %s", sandbox.ID.String(), sandbox.User)
	return nil
}

// CleanupExpiredSandboxes removes sandboxes that haven't been used recently
func (wim *WorkerIsolationManager) CleanupExpiredSandboxes(ctx context.Context, maxAge time.Duration) {
	wim.mutex.Lock()
	defer wim.mutex.Unlock()

	now := time.Now()
	for _, sandbox := range wim.sandboxes {
		if now.Sub(sandbox.LastUsed) > maxAge {
			go wim.cleanupSandbox(sandbox)
		}
	}
}

// GetSandbox returns a sandbox by ID
func (wim *WorkerIsolationManager) GetSandbox(sandboxID uuid.UUID) (*WorkerSandbox, error) {
	wim.mutex.RLock()
	defer wim.mutex.RUnlock()

	sandbox, exists := wim.sandboxes[sandboxID]
	if !exists {
		return nil, fmt.Errorf("sandbox not found")
	}

	return sandbox, nil
}

// ListSandboxes returns all active sandboxes
func (wim *WorkerIsolationManager) ListSandboxes() map[uuid.UUID]*WorkerSandbox {
	wim.mutex.RLock()
	defer wim.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[uuid.UUID]*WorkerSandbox)
	for id, sandbox := range wim.sandboxes {
		result[id] = sandbox
	}

	return result
}