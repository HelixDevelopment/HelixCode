package shell

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Config contains shell execution configuration
type Config struct {
	Security       *SecurityConfig
	Sandbox        *SandboxConfig
	MaxConcurrent  int
	DefaultTimeout time.Duration
	MaxTimeout     time.Duration
	MaxOutputSize  int64
	WorkDir        string
	Env            map[string]string
	AuditLog       bool
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	AllowlistMode     AllowlistMode
	Allowlist         map[string]bool
	AllowlistPrefixes []string
	AllowlistPatterns []string
	Blocklist         map[string]bool
	BlocklistPatterns []string
}

// ShellExecutor is the main coordinator for shell execution
type ShellExecutor struct {
	executor *DefaultExecutor
	config   *Config
}

// NewShellExecutor creates a new shell executor with the given configuration
func NewShellExecutor(config *Config) *ShellExecutor {
	if config == nil {
		config = DefaultConfig()
	}

	return &ShellExecutor{
		executor: NewDefaultExecutor(config),
		config:   config,
	}
}

// Execute executes a command synchronously
func (se *ShellExecutor) Execute(ctx context.Context, cmd *Command) (*ExecutionResult, error) {
	// Apply default configuration if not set
	se.applyDefaults(cmd)

	return se.executor.Execute(ctx, cmd)
}

// ExecuteAsync executes a command asynchronously
func (se *ShellExecutor) ExecuteAsync(ctx context.Context, cmd *Command) (*AsyncExecution, error) {
	// Apply default configuration if not set
	se.applyDefaults(cmd)

	return se.executor.ExecuteAsync(ctx, cmd)
}

// ExecuteStream executes a command with real-time output streaming
func (se *ShellExecutor) ExecuteStream(ctx context.Context, cmd *Command) (*StreamingExecution, error) {
	// Apply default configuration if not set
	se.applyDefaults(cmd)

	return se.executor.ExecuteStream(ctx, cmd)
}

// ExecuteWithTimeout executes a command with a specific timeout
func (se *ShellExecutor) ExecuteWithTimeout(cmd *Command, timeout time.Duration) (*ExecutionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd.Timeout = timeout
	return se.Execute(ctx, cmd)
}

// Kill terminates a running command
func (se *ShellExecutor) Kill(executionID string, signal os.Signal) error {
	return se.executor.Kill(executionID, signal)
}

// GetStatus returns the status of a running command
func (se *ShellExecutor) GetStatus(executionID string) (*ExecutionStatus, error) {
	return se.executor.GetStatus(executionID)
}

// ListExecutions lists all running executions
func (se *ShellExecutor) ListExecutions() []*ExecutionStatus {
	return se.executor.ListExecutions()
}

// applyDefaults applies default configuration to a command
func (se *ShellExecutor) applyDefaults(cmd *Command) {
	if cmd.WorkDir == "" && se.config.WorkDir != "" {
		cmd.WorkDir = se.config.WorkDir
	}

	if cmd.Env == nil && se.config.Env != nil {
		cmd.Env = se.config.Env
	}

	if cmd.MaxOutputSize == 0 {
		cmd.MaxOutputSize = se.config.MaxOutputSize
	}

	if cmd.Timeout == 0 {
		cmd.Timeout = se.config.DefaultTimeout
	}

	if cmd.Sandbox == nil {
		cmd.Sandbox = se.config.Sandbox
	}
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Security: &SecurityConfig{
			AllowlistMode:     AllowlistDisabled, // Disabled by default, only blocklist active
			Allowlist:         DefaultAllowlist(),
			AllowlistPrefixes: []string{},
			AllowlistPatterns: []string{},
			Blocklist:         DefaultBlocklist(),
			BlocklistPatterns: DefaultBlocklistPatterns(),
		},
		Sandbox: &SandboxConfig{
			Enabled: true,
			Resources: ResourceLimits{
				MaxMemory:    500 * 1024 * 1024, // 500 MB
				MaxProcesses: 20,
				MaxFileSize:  100 * 1024 * 1024, // 100 MB
				MaxOpenFiles: 1024,
				Timeout:      5 * time.Minute,
			},
		},
		MaxConcurrent:  10,
		DefaultTimeout: 30 * time.Second,
		MaxTimeout:     10 * time.Minute,
		MaxOutputSize:  10 * 1024 * 1024, // 10 MB
		AuditLog:       true,
	}
}

// StrictConfig returns a strict security configuration
func StrictConfig() *Config {
	config := DefaultConfig()
	config.Security.AllowlistMode = AllowlistStrict
	config.Security.Allowlist = map[string]bool{
		"ls":   true,
		"cat":  true,
		"echo": true,
		"pwd":  true,
	}
	config.Sandbox.Resources.MaxMemory = 100 * 1024 * 1024 // 100 MB
	config.Sandbox.Resources.MaxProcesses = 5
	config.Sandbox.Resources.Timeout = 1 * time.Minute
	config.MaxConcurrent = 3
	config.DefaultTimeout = 15 * time.Second
	config.MaxTimeout = 2 * time.Minute
	return config
}

// PermissiveConfig returns a permissive configuration (use with caution)
func PermissiveConfig() *Config {
	config := DefaultConfig()
	config.Security.AllowlistMode = AllowlistDisabled
	config.Sandbox.Enabled = false
	config.MaxConcurrent = 50
	config.DefaultTimeout = 5 * time.Minute
	config.MaxTimeout = 30 * time.Minute
	config.MaxOutputSize = 100 * 1024 * 1024 // 100 MB
	return config
}

// ValidateConfig validates a configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Security == nil {
		return fmt.Errorf("security config cannot be nil")
	}

	if config.Sandbox == nil {
		return fmt.Errorf("sandbox config cannot be nil")
	}

	if config.MaxConcurrent <= 0 {
		return fmt.Errorf("max concurrent must be positive: %d", config.MaxConcurrent)
	}

	if config.DefaultTimeout < 0 {
		return fmt.Errorf("default timeout cannot be negative: %v", config.DefaultTimeout)
	}

	if config.MaxTimeout < 0 {
		return fmt.Errorf("max timeout cannot be negative: %v", config.MaxTimeout)
	}

	if config.MaxTimeout > 0 && config.DefaultTimeout > config.MaxTimeout {
		return fmt.Errorf("default timeout (%v) cannot exceed max timeout (%v)", config.DefaultTimeout, config.MaxTimeout)
	}

	if config.MaxOutputSize < 0 {
		return fmt.Errorf("max output size cannot be negative: %d", config.MaxOutputSize)
	}

	// Validate sandbox configuration
	sandbox := NewSandbox(config.Sandbox)
	if err := sandbox.Validate(); err != nil {
		return fmt.Errorf("invalid sandbox config: %w", err)
	}

	return nil
}

// QuickExecute is a convenience function to quickly execute a command with default settings
func QuickExecute(command string) (*ExecutionResult, error) {
	executor := NewShellExecutor(DefaultConfig())
	cmd := &Command{
		ID:      fmt.Sprintf("quick-%d", time.Now().UnixNano()),
		Command: command,
	}
	return executor.Execute(context.Background(), cmd)
}

// QuickExecuteWithTimeout is a convenience function to quickly execute a command with a timeout
func QuickExecuteWithTimeout(command string, timeout time.Duration) (*ExecutionResult, error) {
	executor := NewShellExecutor(DefaultConfig())
	cmd := &Command{
		ID:      fmt.Sprintf("quick-%d", time.Now().UnixNano()),
		Command: command,
		Timeout: timeout,
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return executor.Execute(ctx, cmd)
}

// QuickStream is a convenience function to quickly stream a command's output
func QuickStream(command string) (*StreamingExecution, error) {
	executor := NewShellExecutor(DefaultConfig())
	cmd := &Command{
		ID:      fmt.Sprintf("stream-%d", time.Now().UnixNano()),
		Command: command,
	}
	return executor.ExecuteStream(context.Background(), cmd)
}
