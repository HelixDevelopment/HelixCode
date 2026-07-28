package shell

import (
	"context"
	"errors"
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

	// Create parent-owned pipes for streaming. This deliberately does NOT use
	// execCmd.StdoutPipe()/StderrPipe() — see newStreamPipes for why.
	pipes, err := newStreamPipes(execCmd)
	if err != nil {
		<-e.semaphore
		return nil, err
	}

	// Create output streamer
	streamer := NewOutputStreamer(pipes.stdoutR, pipes.stderrR)

	// Apply sandbox
	if err := e.sandbox.Apply(execCmd); err != nil {
		pipes.closeAll()
		<-e.semaphore
		return nil, err
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		pipes.closeAll()
		<-e.semaphore
		return nil, err
	}

	// Drop the parent's copies of the write ends now that the child has
	// inherited them. Until every write end but the child's is closed the
	// readers never observe EOF, so the scanners would hang after the command
	// exits instead of finishing.
	pipes.closeWriteEnds()

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
			// Runs after the result has been published and after the scanners
			// have finished, so closing the read ends here cannot truncate
			// anything.
			pipes.closeAll()
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

			// The caller may have abandoned the output channels when it
			// cancelled. Release any scanner parked on a channel send so the
			// result below is still delivered.
			streamer.Stop()
		}

		// Publish the result only once the scanners have finished. Every write
		// end is closed by now (the child's on exit, ours right after Start),
		// so the scanners drain the pipes to EOF and terminate on their own.
		//
		// This ordering is what makes "Done fired" mean "all output has already
		// been delivered" instead of "output may still be in flight". It is the
		// ordering guarantee, combined with the parent-owned pipes above, that
		// replaces the previous race in which Cmd.Wait could tear the pipes
		// down before the scanners had read a single byte.
		<-streamer.Done()

		if streamErr := streamer.Err(); streamErr != nil &&
			result.Error == nil && !errors.Is(streamErr, ErrStreamStopped) {
			result.Error = fmt.Errorf("output streaming failed: %w", streamErr)
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

// streamPipes holds the parent-owned ends of the pipes wired to a command's
// stdout and stderr for streaming.
type streamPipes struct {
	stdoutR *os.File
	stderrR *os.File
	stdoutW *os.File
	stderrW *os.File
}

// newStreamPipes wires execCmd's stdout and stderr to pipes owned by US.
//
// It deliberately does NOT use execCmd.StdoutPipe()/StderrPipe(). Those hand
// the parent's read end to os/exec, which records it in Cmd.parentIOPipes
// (go1.26 os/exec/exec.go:1095) and closes it inside Cmd.Wait
// (exec.go:954, `closeDescriptors(c.parentIOPipes)`) the moment the process is
// reaped. The stdlib states the constraint outright at exec.go:1077-1079:
// "Cmd.Wait will close the pipe after seeing the command exit ... It is thus
// incorrect to call Wait before all reads from the pipe have completed."
//
// ExecuteStream calls Wait from a goroutine that runs CONCURRENTLY with the
// scanner goroutines, so nothing enforced that constraint. For a command that
// exits almost immediately, Wait frequently won the race and closed the read
// end before the scanner's first Read — the scanner then failed instantly, and
// because bufio.Scanner reports a read error the same way it reports EOF, the
// caller saw zero output alongside a perfectly correct exit code 0.
//
// Assigning an *os.File directly to Cmd.Stdout/Stderr removes the hazard
// structurally rather than re-ordering around it: Cmd.writerDescriptor returns
// an *os.File as-is (exec.go, `if f, ok := w.(*os.File); ok { return f, nil }`)
// and appends it to NEITHER parentIOPipes NOR childIOFiles, so os/exec never
// closes any of these four descriptors. Their lifetime is ours alone, and Wait
// can no longer interfere with a read in progress.
//
// The caller MUST call closeWriteEnds after Start, and closeAll once streaming
// has finished.
func newStreamPipes(execCmd *exec.Cmd) (*streamPipes, error) {
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	execCmd.Stdout = stdoutW
	execCmd.Stderr = stderrW

	return &streamPipes{
		stdoutR: stdoutR,
		stderrR: stderrR,
		stdoutW: stdoutW,
		stderrW: stderrW,
	}, nil
}

// closeWriteEnds drops the parent's copies of the write ends. It MUST be called
// after execCmd.Start(): while the parent still holds a write end open, the
// pipe always has a live writer, so the reader never observes EOF even after
// the child has exited.
func (p *streamPipes) closeWriteEnds() {
	if p.stdoutW != nil {
		p.stdoutW.Close()
		p.stdoutW = nil
	}
	if p.stderrW != nil {
		p.stderrW.Close()
		p.stderrW = nil
	}
}

// closeAll releases every descriptor still held. Because os/exec does not own
// these files, nothing else will. Safe to call more than once.
func (p *streamPipes) closeAll() {
	p.closeWriteEnds()
	if p.stdoutR != nil {
		p.stdoutR.Close()
		p.stdoutR = nil
	}
	if p.stderrR != nil {
		p.stderrR.Close()
		p.stderrR = nil
	}
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
