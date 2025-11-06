package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ExecutionState represents the state of an execution
type ExecutionState int

const (
	StateQueued ExecutionState = iota
	StateRunning
	StateCompleted
	StateFailed
	StateKilled
	StateTimedOut
)

func (s ExecutionState) String() string {
	return [...]string{"Queued", "Running", "Completed", "Failed", "Killed", "TimedOut"}[s]
}

// Command represents a shell command to execute
type Command struct {
	ID            string
	Command       string
	Args          []string
	WorkDir       string
	Env           map[string]string
	Timeout       time.Duration
	Shell         string // bash, sh, zsh, etc.
	CaptureOutput bool
	StreamOutput  bool
	User          string // Run as specific user (requires elevated privileges)
	MaxOutputSize int64
	Sandbox       *SandboxConfig
}

// ExecutionResult contains the result of command execution
type ExecutionResult struct {
	ID         string
	Command    string
	ExitCode   int
	Stdout     string
	Stderr     string
	Duration   time.Duration
	StartTime  time.Time
	EndTime    time.Time
	Error      error
	Killed     bool
	TimedOut   bool
	OutputSize int64
}

// AsyncExecution represents an asynchronous command execution
type AsyncExecution struct {
	ID        string
	Command   string
	StartTime time.Time
	Done      <-chan *ExecutionResult
	Cancel    context.CancelFunc
}

// StreamingExecution provides real-time output streaming
type StreamingExecution struct {
	ID        string
	Command   string
	StartTime time.Time
	Stdout    <-chan string
	Stderr    <-chan string
	Done      <-chan *ExecutionResult
	Cancel    context.CancelFunc
}

// ExecutionStatus represents the current status of an execution
type ExecutionStatus struct {
	ID        string
	Command   string
	State     ExecutionState
	StartTime time.Time
	Duration  time.Duration
	PID       int
}

// CommandExecutor executes shell commands
type CommandExecutor interface {
	// Execute runs a command and waits for completion
	Execute(ctx context.Context, cmd *Command) (*ExecutionResult, error)

	// ExecuteAsync runs a command asynchronously
	ExecuteAsync(ctx context.Context, cmd *Command) (*AsyncExecution, error)

	// ExecuteStream runs a command with real-time output streaming
	ExecuteStream(ctx context.Context, cmd *Command) (*StreamingExecution, error)

	// Kill terminates a running command
	Kill(executionID string, signal os.Signal) error

	// GetStatus returns the status of a running command
	GetStatus(executionID string) (*ExecutionStatus, error)

	// ListExecutions lists all running executions
	ListExecutions() []*ExecutionStatus
}

// DefaultExecutor implements CommandExecutor
type DefaultExecutor struct {
	security       *SecurityManager
	sandbox        *Sandbox
	signalHandler  *SignalHandler
	timeoutManager *TimeoutManager
	executions     sync.Map // map[string]*ExecutionStatus
	maxConcurrent  int
	semaphore      chan struct{}
}

// NewDefaultExecutor creates a new default executor
func NewDefaultExecutor(config *Config) *DefaultExecutor {
	return &DefaultExecutor{
		security:       NewSecurityManager(config.Security),
		sandbox:        NewSandbox(config.Sandbox),
		signalHandler:  NewSignalHandler(),
		timeoutManager: NewTimeoutManager(config.DefaultTimeout, config.MaxTimeout),
		maxConcurrent:  config.MaxConcurrent,
		semaphore:      make(chan struct{}, config.MaxConcurrent),
	}
}

// Execute executes a command synchronously
func (e *DefaultExecutor) Execute(ctx context.Context, cmd *Command) (*ExecutionResult, error) {
	// Validate command
	if err := e.security.ValidateCommand(cmd); err != nil {
		return nil, err
	}

	// Acquire semaphore
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Prepare execution
	execCmd, err := e.prepareCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Set max output size
	maxOutputSize := cmd.MaxOutputSize
	if maxOutputSize <= 0 {
		maxOutputSize = 10 * 1024 * 1024 // 10 MB default
	}

	// Create output collector
	collector := NewOutputCollector(maxOutputSize)
	execCmd.Stdout = &writerAdapter{collector.WriteStdout}
	execCmd.Stderr = &writerAdapter{collector.WriteStderr}

	// Apply sandbox
	if err := e.sandbox.Apply(execCmd); err != nil {
		return nil, err
	}

	// Create execution result
	result := &ExecutionResult{
		ID:        cmd.ID,
		Command:   cmd.Command,
		StartTime: time.Now(),
	}

	// Set up timeout
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	timeout := cmd.Timeout
	if cmd.Sandbox != nil && cmd.Sandbox.Resources.Timeout > 0 {
		if timeout == 0 || cmd.Sandbox.Resources.Timeout < timeout {
			timeout = cmd.Sandbox.Resources.Timeout
		}
	}

	if timeout > 0 {
		e.timeoutManager.Start(cmd.ID, timeout, func() {
			result.TimedOut = true
			cancel()
		})
		defer e.timeoutManager.Cancel(cmd.ID)
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		result.Error = err
		return result, err
	}

	// Register for signal handling
	pid := execCmd.Process.Pid
	pgid := pid
	if execCmd.SysProcAttr != nil && execCmd.SysProcAttr.Setpgid {
		pgid = pid
	}
	e.signalHandler.Register(cmd.ID, pid, pgid, cmd.Command)
	defer e.signalHandler.Unregister(cmd.ID)

	// Register execution status
	e.executions.Store(cmd.ID, &ExecutionStatus{
		ID:        cmd.ID,
		Command:   cmd.Command,
		State:     StateRunning,
		StartTime: result.StartTime,
		PID:       pid,
	})
	defer e.executions.Delete(cmd.ID)

	// Wait for completion
	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()

	select {
	case err := <-done:
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
			} else {
				result.Error = err
			}
		} else {
			result.ExitCode = 0
		}

	case <-execCtx.Done():
		// Timeout or cancellation
		e.signalHandler.Send(cmd.ID, syscall.SIGKILL)
		result.Killed = true
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		<-done // Wait for process to actually exit
	}

	// Collect output
	stdout, stderr, truncated := collector.GetOutput()
	result.Stdout = stdout
	result.Stderr = stderr
	result.OutputSize = collector.Size()
	if truncated {
		result.Stdout += "\n[output truncated due to size limit]"
	}

	return result, nil
}

// ExecuteAsync executes a command asynchronously
func (e *DefaultExecutor) ExecuteAsync(ctx context.Context, cmd *Command) (*AsyncExecution, error) {
	// Validate command
	if err := e.security.ValidateCommand(cmd); err != nil {
		return nil, err
	}

	// Create execution context
	execCtx, cancel := context.WithCancel(ctx)

	// Create result channel
	done := make(chan *ExecutionResult, 1)

	// Start execution in background
	go func() {
		result, _ := e.Execute(execCtx, cmd)
		done <- result
	}()

	return &AsyncExecution{
		ID:        cmd.ID,
		Command:   cmd.Command,
		StartTime: time.Now(),
		Done:      done,
		Cancel:    cancel,
	}, nil
}

// ExecuteStream executes a command with streaming output
func (e *DefaultExecutor) ExecuteStream(ctx context.Context, cmd *Command) (*StreamingExecution, error) {
	// Validate command
	if err := e.security.ValidateCommand(cmd); err != nil {
		return nil, err
	}

	// Acquire semaphore
	select {
	case e.semaphore <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Prepare execution
	execCmd, err := e.prepareCommand(cmd)
	if err != nil {
		<-e.semaphore
		return nil, err
	}

	// Create pipes for streaming
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		<-e.semaphore
		return nil, err
	}

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		<-e.semaphore
		return nil, err
	}

	// Create output streamer
	streamer := NewOutputStreamer(stdoutPipe, stderrPipe)

	// Apply sandbox
	if err := e.sandbox.Apply(execCmd); err != nil {
		<-e.semaphore
		return nil, err
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		<-e.semaphore
		return nil, err
	}

	// Register for signal handling
	pid := execCmd.Process.Pid
	pgid := pid
	if execCmd.SysProcAttr != nil && execCmd.SysProcAttr.Setpgid {
		pgid = pid
	}
	e.signalHandler.Register(cmd.ID, pid, pgid, cmd.Command)

	// Register execution status
	startTime := time.Now()
	e.executions.Store(cmd.ID, &ExecutionStatus{
		ID:        cmd.ID,
		Command:   cmd.Command,
		State:     StateRunning,
		StartTime: startTime,
		PID:       pid,
	})

	// Start streaming
	streamer.Start()

	// Create execution context
	execCtx, cancel := context.WithCancel(ctx)

	// Set up timeout
	timeout := cmd.Timeout
	if cmd.Sandbox != nil && cmd.Sandbox.Resources.Timeout > 0 {
		if timeout == 0 || cmd.Sandbox.Resources.Timeout < timeout {
			timeout = cmd.Sandbox.Resources.Timeout
		}
	}

	if timeout > 0 {
		e.timeoutManager.Start(cmd.ID, timeout, func() {
			cancel()
		})
	}

	// Create result channel
	done := make(chan *ExecutionResult, 1)
	go func() {
		defer func() {
			<-e.semaphore
			e.signalHandler.Unregister(cmd.ID)
			e.timeoutManager.Cancel(cmd.ID)
			e.executions.Delete(cmd.ID)
		}()

		result := &ExecutionResult{
			ID:        cmd.ID,
			Command:   cmd.Command,
			StartTime: startTime,
		}

		// Wait for completion
		waitDone := make(chan error, 1)
		go func() {
			waitDone <- execCmd.Wait()
		}()

		select {
		case err := <-waitDone:
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)

			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					result.ExitCode = exitErr.ExitCode()
				} else {
					result.Error = err
				}
			} else {
				result.ExitCode = 0
			}

		case <-execCtx.Done():
			// Timeout or cancellation
			result.TimedOut = ctx.Err() == context.DeadlineExceeded
			result.Killed = true
			e.signalHandler.Send(cmd.ID, syscall.SIGKILL)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			<-waitDone // Wait for process to actually exit
		}

		done <- result
	}()

	return &StreamingExecution{
		ID:        cmd.ID,
		Command:   cmd.Command,
		StartTime: startTime,
		Stdout:    streamer.GetStdout(),
		Stderr:    streamer.GetStderr(),
		Done:      done,
		Cancel:    cancel,
	}, nil
}

// Kill terminates a running command
func (e *DefaultExecutor) Kill(executionID string, signal os.Signal) error {
	// Convert os.Signal to syscall.Signal
	sig, ok := signal.(syscall.Signal)
	if !ok {
		sig = syscall.SIGKILL
	}

	return e.signalHandler.Send(executionID, sig)
}

// GetStatus returns the status of a running command
func (e *DefaultExecutor) GetStatus(executionID string) (*ExecutionStatus, error) {
	val, ok := e.executions.Load(executionID)
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	status := val.(*ExecutionStatus)
	// Update duration
	status.Duration = time.Since(status.StartTime)
	return status, nil
}

// ListExecutions lists all running executions
func (e *DefaultExecutor) ListExecutions() []*ExecutionStatus {
	var executions []*ExecutionStatus
	e.executions.Range(func(key, value interface{}) bool {
		status := value.(*ExecutionStatus)
		// Update duration
		status.Duration = time.Since(status.StartTime)
		executions = append(executions, status)
		return true
	})
	return executions
}

// prepareCommand prepares an exec.Cmd from a Command
func (e *DefaultExecutor) prepareCommand(cmd *Command) (*exec.Cmd, error) {
	shell := cmd.Shell
	if shell == "" {
		shell = "/bin/sh"
	}

	var execCmd *exec.Cmd
	if len(cmd.Args) > 0 {
		execCmd = exec.Command(shell, append([]string{"-c", cmd.Command}, cmd.Args...)...)
	} else {
		execCmd = exec.Command(shell, "-c", cmd.Command)
	}

	if cmd.WorkDir != "" {
		execCmd.Dir = cmd.WorkDir
	}

	if len(cmd.Env) > 0 {
		// Start with current environment
		env := os.Environ()
		// Add custom environment variables
		sanitizedEnv := SanitizeEnv(cmd.Env)
		for k, v := range sanitizedEnv {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		execCmd.Env = env
	}

	return execCmd, nil
}
