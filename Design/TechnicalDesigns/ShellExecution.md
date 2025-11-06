# ShellExecution - Technical Design Document

## Overview

The ShellExecution module provides secure, controlled, and efficient shell command execution for HelixCode. This design is inspired by Cline's shell execution and Aider's command execution, with enhanced security, real-time streaming, and sandboxing capabilities.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      ShellExecution                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Executor   │  │   Sandbox    │  │   Streamer   │         │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │
│         │                  │                  │                 │
│  ┌──────┴──────────────────┴──────────────────┴──────┐         │
│  │            Security Manager                        │         │
│  └──────┬──────────────┬──────────────┬──────────────┘         │
│         │              │              │                         │
│  ┌──────┴─────┐ ┌──────┴─────┐ ┌──────┴─────┐                 │
│  │ Allowlist  │ │  Timeout   │ │  Signal    │                 │
│  │ Manager    │ │  Manager   │ │  Handler   │                 │
│  └────────────┘ └────────────┘ └────────────┘                 │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐         │
│  │          Environment Manager                     │         │
│  └──────────────────────────────────────────────────┘         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           │
                           ▼
                    OS Shell (bash/sh)
```

## Core Interfaces

### CommandExecutor Interface

```go
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

// Command represents a shell command to execute
type Command struct {
    ID             string
    Command        string
    Args           []string
    WorkDir        string
    Env            map[string]string
    Timeout        time.Duration
    Shell          string // bash, sh, zsh, etc.
    CaptureOutput  bool
    StreamOutput   bool
    User           string // Run as specific user (requires elevated privileges)
    MaxOutputSize  int64
    Sandbox        *SandboxConfig
}

// ExecutionResult contains the result of command execution
type ExecutionResult struct {
    ID          string
    Command     string
    ExitCode    int
    Stdout      string
    Stderr      string
    Duration    time.Duration
    StartTime   time.Time
    EndTime     time.Time
    Error       error
    Killed      bool
    TimedOut    bool
    OutputSize  int64
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
```

### ExecutionEnvironment Interface

```go
// ExecutionEnvironment manages the execution environment
type ExecutionEnvironment interface {
    // Prepare prepares the execution environment
    Prepare(ctx context.Context, cmd *Command) (*PreparedEnvironment, error)

    // Cleanup cleans up the execution environment
    Cleanup(env *PreparedEnvironment) error

    // Validate validates the command before execution
    Validate(cmd *Command) error
}

// PreparedEnvironment represents a prepared execution environment
type PreparedEnvironment struct {
    WorkDir     string
    Env         []string
    Shell       string
    TempFiles   []string
    Mounts      []Mount
    NetworkMode NetworkMode
}

// Mount represents a filesystem mount
type Mount struct {
    Source   string
    Target   string
    ReadOnly bool
}

// NetworkMode defines network access mode
type NetworkMode int

const (
    NetworkFull NetworkMode = iota // Full network access
    NetworkNone                     // No network access
    NetworkHost                     // Host network only
)
```

## Security Implementation

### Security Manager

```go
// SecurityManager manages command security
type SecurityManager struct {
    allowlist     *AllowlistManager
    blocklist     *BlocklistManager
    sandboxConfig *SandboxConfig
    auditLog      *AuditLog
}

// ValidateCommand validates a command against security policies
func (sm *SecurityManager) ValidateCommand(cmd *Command) error {
    // Check blocklist first (fastest rejection)
    if sm.blocklist.IsBlocked(cmd.Command) {
        return &SecurityError{
            Type:    "blocked_command",
            Message: "command is blocked",
            Command: cmd.Command,
        }
    }

    // Check allowlist
    if !sm.allowlist.IsAllowed(cmd.Command) {
        return &SecurityError{
            Type:    "not_allowed",
            Message: "command is not in allowlist",
            Command: cmd.Command,
        }
    }

    // Check for dangerous patterns
    if sm.containsDangerousPatterns(cmd.Command) {
        return &SecurityError{
            Type:    "dangerous_pattern",
            Message: "command contains dangerous patterns",
            Command: cmd.Command,
        }
    }

    // Check arguments
    for _, arg := range cmd.Args {
        if sm.containsDangerousPatterns(arg) {
            return &SecurityError{
                Type:    "dangerous_argument",
                Message: "argument contains dangerous patterns",
                Command: cmd.Command,
            }
        }
    }

    // Audit log
    sm.auditLog.LogCommandAttempt(cmd)

    return nil
}

// containsDangerousPatterns checks for dangerous command patterns
func (sm *SecurityManager) containsDangerousPatterns(s string) bool {
    dangerousPatterns := []string{
        "rm -rf /",
        ":(){ :|:& };:", // Fork bomb
        "> /dev/sda",
        "mkfs",
        "dd if=/dev/zero",
        "wget http://",
        "curl http://",
    }

    for _, pattern := range dangerousPatterns {
        if strings.Contains(s, pattern) {
            return true
        }
    }

    return false
}

// SecurityError represents a security-related error
type SecurityError struct {
    Type    string
    Message string
    Command string
}

func (e *SecurityError) Error() string {
    return fmt.Sprintf("security error [%s]: %s (command: %s)", e.Type, e.Message, e.Command)
}
```

### Allowlist/Blocklist Management

```go
// AllowlistManager manages allowed commands
type AllowlistManager struct {
    exactCommands  map[string]bool
    prefixCommands []string
    patterns       []*regexp.Regexp
    mode           AllowlistMode
}

// AllowlistMode defines the allowlist behavior
type AllowlistMode int

const (
    AllowlistStrict AllowlistMode = iota // Only exact matches allowed
    AllowlistPrefix                       // Prefix matches allowed
    AllowlistPattern                      // Pattern matches allowed
    AllowlistDisabled                     // Allowlist disabled (blocklist only)
)

// IsAllowed checks if a command is allowed
func (am *AllowlistManager) IsAllowed(command string) bool {
    if am.mode == AllowlistDisabled {
        return true
    }

    // Extract base command (first word)
    parts := strings.Fields(command)
    if len(parts) == 0 {
        return false
    }
    baseCmd := parts[0]

    // Check exact matches
    if am.exactCommands[baseCmd] {
        return true
    }

    // Check prefix matches
    if am.mode >= AllowlistPrefix {
        for _, prefix := range am.prefixCommands {
            if strings.HasPrefix(baseCmd, prefix) {
                return true
            }
        }
    }

    // Check pattern matches
    if am.mode >= AllowlistPattern {
        for _, pattern := range am.patterns {
            if pattern.MatchString(command) {
                return true
            }
        }
    }

    return false
}

// BlocklistManager manages blocked commands
type BlocklistManager struct {
    exactCommands  map[string]bool
    patterns       []*regexp.Regexp
}

// IsBlocked checks if a command is blocked
func (bm *BlocklistManager) IsBlocked(command string) bool {
    parts := strings.Fields(command)
    if len(parts) == 0 {
        return false
    }
    baseCmd := parts[0]

    // Check exact matches
    if bm.exactCommands[baseCmd] {
        return true
    }

    // Check pattern matches
    for _, pattern := range bm.patterns {
        if pattern.MatchString(command) {
            return true
        }
    }

    return false
}

// DefaultAllowlist returns a default allowlist
func DefaultAllowlist() *AllowlistManager {
    return &AllowlistManager{
        exactCommands: map[string]bool{
            "ls":     true,
            "cat":    true,
            "grep":   true,
            "find":   true,
            "git":    true,
            "npm":    true,
            "go":     true,
            "python": true,
            "node":   true,
            "make":   true,
            "cargo":  true,
            "docker": true,
            "kubectl": true,
        },
        mode: AllowlistPrefix,
    }
}

// DefaultBlocklist returns a default blocklist
func DefaultBlocklist() *BlocklistManager {
    return &BlocklistManager{
        exactCommands: map[string]bool{
            "rm":      true,
            "rmdir":   true,
            "dd":      true,
            "mkfs":    true,
            "fdisk":   true,
            "shutdown": true,
            "reboot":  true,
            "halt":    true,
        },
        patterns: []*regexp.Regexp{
            regexp.MustCompile(`rm\s+-rf\s+/`),
            regexp.MustCompile(`>\s*/dev/sd[a-z]`),
            regexp.MustCompile(`wget\s+http://`),
            regexp.MustCompile(`curl\s+http://`),
        },
    }
}
```

### Sandbox Configuration

```go
// SandboxConfig configures command sandboxing
type SandboxConfig struct {
    Enabled       bool
    Filesystem    FilesystemSandbox
    Network       NetworkSandbox
    Resources     ResourceLimits
    Capabilities  []string
}

// FilesystemSandbox configures filesystem access
type FilesystemSandbox struct {
    RootDir       string
    ReadOnlyPaths []string
    ReadWritePaths []string
    TempDir       string
    IsolateFS     bool // Use chroot or container
}

// NetworkSandbox configures network access
type NetworkSandbox struct {
    Mode          NetworkMode
    AllowedHosts  []string
    AllowedPorts  []int
    DNSServers    []string
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
    MaxMemory     int64
    MaxCPU        float64 // CPU cores
    MaxProcesses  int
    MaxFileSize   int64
    MaxOpenFiles  int
    Timeout       time.Duration
}

// Sandbox implements command sandboxing
type Sandbox struct {
    config *SandboxConfig
}

// Apply applies sandbox restrictions to a command
func (s *Sandbox) Apply(cmd *exec.Cmd) error {
    if !s.config.Enabled {
        return nil
    }

    // Set resource limits
    if err := s.applyResourceLimits(cmd); err != nil {
        return err
    }

    // Set filesystem restrictions
    if err := s.applyFilesystemRestrictions(cmd); err != nil {
        return err
    }

    // Set network restrictions
    if err := s.applyNetworkRestrictions(cmd); err != nil {
        return err
    }

    return nil
}

// applyResourceLimits applies resource limits using setrlimit
func (s *Sandbox) applyResourceLimits(cmd *exec.Cmd) error {
    // Implementation would use syscall.Setrlimit on Unix systems
    // This is a simplified example

    // Set process group for easier cleanup
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }

    return nil
}
```

## Real-Time Output Streaming

### Output Streamer

```go
// OutputStreamer streams command output in real-time
type OutputStreamer struct {
    stdout       io.Reader
    stderr       io.Reader
    stdoutChan   chan string
    stderrChan   chan string
    maxLineSize  int
    done         chan struct{}
}

// NewOutputStreamer creates a new output streamer
func NewOutputStreamer(stdout, stderr io.Reader) *OutputStreamer {
    return &OutputStreamer{
        stdout:      stdout,
        stderr:      stderr,
        stdoutChan:  make(chan string, 100),
        stderrChan:  make(chan string, 100),
        maxLineSize: 4096,
        done:        make(chan struct{}),
    }
}

// Start starts streaming output
func (os *OutputStreamer) Start() {
    var wg sync.WaitGroup

    wg.Add(2)
    go func() {
        defer wg.Done()
        os.streamOutput(os.stdout, os.stdoutChan)
    }()

    go func() {
        defer wg.Done()
        os.streamOutput(os.stderr, os.stderrChan)
    }()

    go func() {
        wg.Wait()
        close(os.stdoutChan)
        close(os.stderrChan)
        close(os.done)
    }()
}

// streamOutput streams output from a reader to a channel
func (os *OutputStreamer) streamOutput(reader io.Reader, ch chan<- string) {
    scanner := bufio.NewScanner(reader)
    scanner.Buffer(make([]byte, os.maxLineSize), os.maxLineSize)

    for scanner.Scan() {
        line := scanner.Text()
        select {
        case ch <- line:
        case <-os.done:
            return
        }
    }
}

// GetStdout returns the stdout channel
func (os *OutputStreamer) GetStdout() <-chan string {
    return os.stdoutChan
}

// GetStderr returns the stderr channel
func (os *OutputStreamer) GetStderr() <-chan string {
    return os.stderrChan
}

// Done returns a channel that's closed when streaming is complete
func (os *OutputStreamer) Done() <-chan struct{} {
    return os.done
}
```

### Buffered Output Collection

```go
// OutputCollector collects command output with size limits
type OutputCollector struct {
    stdout       *bytes.Buffer
    stderr       *bytes.Buffer
    maxSize      int64
    currentSize  atomic.Int64
    truncated    atomic.Bool
    mu           sync.Mutex
}

// NewOutputCollector creates a new output collector
func NewOutputCollector(maxSize int64) *OutputCollector {
    return &OutputCollector{
        stdout:  &bytes.Buffer{},
        stderr:  &bytes.Buffer{},
        maxSize: maxSize,
    }
}

// WriteStdout writes to stdout buffer
func (oc *OutputCollector) WriteStdout(p []byte) (n int, err error) {
    return oc.write(oc.stdout, p)
}

// WriteStderr writes to stderr buffer
func (oc *OutputCollector) WriteStderr(p []byte) (n int, err error) {
    return oc.write(oc.stderr, p)
}

// write writes data to a buffer with size limit
func (oc *OutputCollector) write(buf *bytes.Buffer, p []byte) (int, error) {
    oc.mu.Lock()
    defer oc.mu.Unlock()

    if oc.truncated.Load() {
        return len(p), nil // Discard if already truncated
    }

    newSize := oc.currentSize.Load() + int64(len(p))
    if newSize > oc.maxSize {
        oc.truncated.Store(true)
        remaining := oc.maxSize - oc.currentSize.Load()
        if remaining > 0 {
            buf.Write(p[:remaining])
            buf.WriteString("\n... [output truncated] ...\n")
        }
        return len(p), nil
    }

    n, err := buf.Write(p)
    oc.currentSize.Add(int64(n))
    return n, err
}

// GetOutput returns collected output
func (oc *OutputCollector) GetOutput() (stdout, stderr string, truncated bool) {
    oc.mu.Lock()
    defer oc.mu.Unlock()
    return oc.stdout.String(), oc.stderr.String(), oc.truncated.Load()
}
```

## Signal Handling

### Signal Handler

```go
// SignalHandler manages process signals
type SignalHandler struct {
    processes sync.Map // map[string]*os.Process
}

// Register registers a process for signal handling
func (sh *SignalHandler) Register(id string, process *os.Process) {
    sh.processes.Store(id, process)
}

// Unregister unregisters a process
func (sh *SignalHandler) Unregister(id string) {
    sh.processes.Delete(id)
}

// Send sends a signal to a process
func (sh *SignalHandler) Send(id string, sig os.Signal) error {
    val, ok := sh.processes.Load(id)
    if !ok {
        return fmt.Errorf("process not found: %s", id)
    }

    process := val.(*os.Process)
    return process.Signal(sig)
}

// KillAll kills all registered processes
func (sh *SignalHandler) KillAll() {
    sh.processes.Range(func(key, value interface{}) bool {
        process := value.(*os.Process)
        _ = process.Kill()
        return true
    })
}

// GracefulShutdown attempts graceful shutdown with timeout
func (sh *SignalHandler) GracefulShutdown(id string, timeout time.Duration) error {
    val, ok := sh.processes.Load(id)
    if !ok {
        return fmt.Errorf("process not found: %s", id)
    }

    process := val.(*os.Process)

    // Send SIGTERM
    if err := process.Signal(syscall.SIGTERM); err != nil {
        return err
    }

    // Wait for process to exit
    done := make(chan error, 1)
    go func() {
        _, err := process.Wait()
        done <- err
    }()

    select {
    case err := <-done:
        return err
    case <-time.After(timeout):
        // Force kill if timeout exceeded
        return process.Kill()
    }
}
```

## Timeout Management

### Timeout Manager

```go
// TimeoutManager manages command timeouts
type TimeoutManager struct {
    defaultTimeout time.Duration
    maxTimeout     time.Duration
    timers         sync.Map // map[string]*time.Timer
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager(defaultTimeout, maxTimeout time.Duration) *TimeoutManager {
    return &TimeoutManager{
        defaultTimeout: defaultTimeout,
        maxTimeout:     maxTimeout,
    }
}

// Start starts a timeout for an execution
func (tm *TimeoutManager) Start(id string, timeout time.Duration, onTimeout func()) {
    if timeout == 0 {
        timeout = tm.defaultTimeout
    }
    if timeout > tm.maxTimeout {
        timeout = tm.maxTimeout
    }

    timer := time.AfterFunc(timeout, func() {
        onTimeout()
        tm.timers.Delete(id)
    })

    tm.timers.Store(id, timer)
}

// Cancel cancels a timeout
func (tm *TimeoutManager) Cancel(id string) {
    if val, ok := tm.timers.LoadAndDelete(id); ok {
        timer := val.(*time.Timer)
        timer.Stop()
    }
}

// Extend extends a timeout
func (tm *TimeoutManager) Extend(id string, duration time.Duration) bool {
    val, ok := tm.timers.Load(id)
    if !ok {
        return false
    }

    timer := val.(*time.Timer)
    return timer.Reset(duration)
}
```

## Implementation Example

### Command Executor Implementation

```go
// DefaultExecutor implements CommandExecutor
type DefaultExecutor struct {
    security       *SecurityManager
    sandbox        *Sandbox
    signalHandler  *SignalHandler
    timeoutManager *TimeoutManager
    executions     sync.Map
    maxConcurrent  int
    semaphore      chan struct{}
}

// NewDefaultExecutor creates a new default executor
func NewDefaultExecutor(config *Config) *DefaultExecutor {
    return &DefaultExecutor{
        security:       NewSecurityManager(config.Security),
        sandbox:        NewSandbox(config.Sandbox),
        signalHandler:  &SignalHandler{},
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

    // Create output collector
    collector := NewOutputCollector(cmd.MaxOutputSize)
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

    if cmd.Timeout > 0 {
        e.timeoutManager.Start(cmd.ID, cmd.Timeout, func() {
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
    e.signalHandler.Register(cmd.ID, execCmd.Process)
    defer e.signalHandler.Unregister(cmd.ID)

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
    }

    // Collect output
    stdout, stderr, truncated := collector.GetOutput()
    result.Stdout = stdout
    result.Stderr = stderr
    if truncated {
        result.Stdout += "\n[output truncated due to size limit]"
    }

    return result, nil
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
    e.signalHandler.Register(cmd.ID, execCmd.Process)

    // Start streaming
    streamer.Start()

    // Create execution context
    execCtx, cancel := context.WithCancel(ctx)

    // Set up timeout
    if cmd.Timeout > 0 {
        e.timeoutManager.Start(cmd.ID, cmd.Timeout, func() {
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
        }()

        result := &ExecutionResult{
            ID:        cmd.ID,
            Command:   cmd.Command,
            StartTime: time.Now(),
        }

        err := execCmd.Wait()
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

        done <- result
    }()

    return &StreamingExecution{
        ID:        cmd.ID,
        Command:   cmd.Command,
        StartTime: time.Now(),
        Stdout:    streamer.GetStdout(),
        Stderr:    streamer.GetStderr(),
        Done:      done,
        Cancel:    cancel,
    }, nil
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
        env := os.Environ()
        for k, v := range cmd.Env {
            env = append(env, fmt.Sprintf("%s=%s", k, v))
        }
        execCmd.Env = env
    }

    return execCmd, nil
}

// writerAdapter adapts a write function to io.Writer
type writerAdapter struct {
    write func([]byte) (int, error)
}

func (w *writerAdapter) Write(p []byte) (int, error) {
    return w.write(p)
}
```

## Testing Strategy

### Unit Tests

```go
// TestCommandValidation tests command validation
func TestCommandValidation(t *testing.T) {
    security := NewSecurityManager(&SecurityConfig{
        AllowlistMode: AllowlistStrict,
        Allowlist:     []string{"ls", "cat", "echo"},
        Blocklist:     []string{"rm", "dd"},
    })

    tests := []struct {
        name    string
        command string
        wantErr bool
        errType string
    }{
        {
            name:    "allowed command",
            command: "ls -la",
            wantErr: false,
        },
        {
            name:    "blocked command",
            command: "rm -rf /",
            wantErr: true,
            errType: "blocked_command",
        },
        {
            name:    "not in allowlist",
            command: "python script.py",
            wantErr: true,
            errType: "not_allowed",
        },
        {
            name:    "dangerous pattern",
            command: "dd if=/dev/zero of=/dev/sda",
            wantErr: true,
            errType: "dangerous_pattern",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := &Command{Command: tt.command}
            err := security.ValidateCommand(cmd)

            if tt.wantErr {
                require.Error(t, err)
                var secErr *SecurityError
                require.ErrorAs(t, err, &secErr)
                assert.Equal(t, tt.errType, secErr.Type)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// TestCommandExecution tests command execution
func TestCommandExecution(t *testing.T) {
    executor := NewDefaultExecutor(&Config{
        MaxConcurrent:  5,
        DefaultTimeout: 30 * time.Second,
        MaxTimeout:     5 * time.Minute,
    })

    t.Run("simple command", func(t *testing.T) {
        cmd := &Command{
            ID:      "test-1",
            Command: "echo 'Hello, World!'",
        }

        result, err := executor.Execute(context.Background(), cmd)
        require.NoError(t, err)
        assert.Equal(t, 0, result.ExitCode)
        assert.Contains(t, result.Stdout, "Hello, World!")
    })

    t.Run("command with timeout", func(t *testing.T) {
        cmd := &Command{
            ID:      "test-2",
            Command: "sleep 10",
            Timeout: 1 * time.Second,
        }

        result, err := executor.Execute(context.Background(), cmd)
        assert.Error(t, err)
        assert.True(t, result.TimedOut)
    })

    t.Run("command with error", func(t *testing.T) {
        cmd := &Command{
            ID:      "test-3",
            Command: "exit 1",
        }

        result, err := executor.Execute(context.Background(), cmd)
        assert.NoError(t, err)
        assert.Equal(t, 1, result.ExitCode)
    })
}

// TestStreamingExecution tests streaming execution
func TestStreamingExecution(t *testing.T) {
    executor := NewDefaultExecutor(&Config{
        MaxConcurrent: 5,
    })

    cmd := &Command{
        ID:      "test-stream",
        Command: "for i in 1 2 3; do echo $i; sleep 0.1; done",
    }

    exec, err := executor.ExecuteStream(context.Background(), cmd)
    require.NoError(t, err)

    var lines []string
    for line := range exec.Stdout {
        lines = append(lines, line)
    }

    result := <-exec.Done
    assert.Equal(t, 0, result.ExitCode)
    assert.Equal(t, []string{"1", "2", "3"}, lines)
}
```

### Integration Tests

```go
// TestShellExecutionIntegration tests the full execution workflow
func TestShellExecutionIntegration(t *testing.T) {
    config := &Config{
        Security: &SecurityConfig{
            AllowlistMode: AllowlistDisabled,
            Blocklist:     []string{"rm", "dd", "mkfs"},
        },
        Sandbox: &SandboxConfig{
            Enabled: true,
            Resources: ResourceLimits{
                MaxMemory:    100 * 1024 * 1024, // 100 MB
                MaxProcesses: 10,
                Timeout:      1 * time.Minute,
            },
        },
        MaxConcurrent:  5,
        DefaultTimeout: 30 * time.Second,
    }

    executor := NewDefaultExecutor(config)

    t.Run("execute script with output", func(t *testing.T) {
        script := `#!/bin/sh
echo "Starting..."
for i in 1 2 3 4 5; do
    echo "Count: $i"
done
echo "Done!"
`
        tmpfile, err := os.CreateTemp("", "test-script-*.sh")
        require.NoError(t, err)
        defer os.Remove(tmpfile.Name())

        _, err = tmpfile.WriteString(script)
        require.NoError(t, err)
        tmpfile.Close()

        err = os.Chmod(tmpfile.Name(), 0755)
        require.NoError(t, err)

        cmd := &Command{
            ID:      "test-script",
            Command: tmpfile.Name(),
        }

        result, err := executor.Execute(context.Background(), cmd)
        require.NoError(t, err)
        assert.Equal(t, 0, result.ExitCode)
        assert.Contains(t, result.Stdout, "Starting...")
        assert.Contains(t, result.Stdout, "Done!")
    })

    t.Run("concurrent executions", func(t *testing.T) {
        var wg sync.WaitGroup
        results := make([]*ExecutionResult, 10)

        for i := 0; i < 10; i++ {
            wg.Add(1)
            go func(idx int) {
                defer wg.Done()

                cmd := &Command{
                    ID:      fmt.Sprintf("concurrent-%d", idx),
                    Command: fmt.Sprintf("echo 'Task %d'", idx),
                }

                result, err := executor.Execute(context.Background(), cmd)
                require.NoError(t, err)
                results[idx] = result
            }(i)
        }

        wg.Wait()

        for i, result := range results {
            assert.Equal(t, 0, result.ExitCode)
            assert.Contains(t, result.Stdout, fmt.Sprintf("Task %d", i))
        }
    })
}
```

### Security Tests

```go
// TestSecurityEnforcement tests security enforcement
func TestSecurityEnforcement(t *testing.T) {
    config := &Config{
        Security: &SecurityConfig{
            AllowlistMode: AllowlistStrict,
            Allowlist:     []string{"echo", "printf"},
            Blocklist:     []string{"rm", "dd"},
        },
    }

    executor := NewDefaultExecutor(config)

    t.Run("reject dangerous commands", func(t *testing.T) {
        dangerousCommands := []string{
            "rm -rf /",
            "dd if=/dev/zero of=/dev/sda",
            ":(){ :|:& };:",
            "mkfs.ext4 /dev/sda1",
        }

        for _, cmdStr := range dangerousCommands {
            cmd := &Command{
                ID:      "dangerous",
                Command: cmdStr,
            }

            _, err := executor.Execute(context.Background(), cmd)
            assert.Error(t, err, "should reject: %s", cmdStr)

            var secErr *SecurityError
            assert.ErrorAs(t, err, &secErr)
        }
    })

    t.Run("enforce allowlist", func(t *testing.T) {
        cmd := &Command{
            ID:      "not-allowed",
            Command: "ls -la",
        }

        _, err := executor.Execute(context.Background(), cmd)
        assert.Error(t, err)

        var secErr *SecurityError
        require.ErrorAs(t, err, &secErr)
        assert.Equal(t, "not_allowed", secErr.Type)
    })
}
```

## Configuration

```go
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

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
    return &Config{
        Security: &SecurityConfig{
            AllowlistMode: AllowlistPrefix,
            Allowlist:     DefaultAllowlist().exactCommands,
            Blocklist:     DefaultBlocklist().exactCommands,
        },
        Sandbox: &SandboxConfig{
            Enabled: true,
            Resources: ResourceLimits{
                MaxMemory:    500 * 1024 * 1024, // 500 MB
                MaxProcesses: 20,
                MaxFileSize:  100 * 1024 * 1024, // 100 MB
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
```

## References

### Cline's Shell Execution

- **Location**: `src/core/webview/WebviewProvider.ts`, `src/core/shell/ShellExecutor.ts`
- **Features**:
  - Real-time output streaming
  - Command history
  - Interactive shell support
  - Terminal emulation

### Aider's Command Execution

- **Location**: `aider/coders/base_coder.py`, command execution methods
- **Features**:
  - Safe command execution
  - Output capturing
  - Error handling
  - Git command integration

### Additional References

- Go standard library `os/exec`
- `github.com/creack/pty` for PTY support (future enhancement)
- Docker/containerd for advanced sandboxing

## Future Enhancements

1. **Interactive Shell**: Full terminal emulation with PTY
2. **Container Sandboxing**: Use Docker/containerd for isolation
3. **Command History**: Persistent command history with search
4. **Shell Sessions**: Long-lived shell sessions with state
5. **Progress Tracking**: Better progress indication for long-running commands
6. **Output Filtering**: Pattern-based output filtering
7. **Environment Profiles**: Pre-configured environment setups
8. **Command Templates**: Reusable command templates
9. **Execution Policies**: Fine-grained execution policies based on context
10. **Remote Execution**: Execute commands on remote hosts via SSH
