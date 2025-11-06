package shell

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// NetworkMode defines network access mode
type NetworkMode int

const (
	NetworkFull NetworkMode = iota // Full network access
	NetworkNone                     // No network access
	NetworkHost                     // Host network only
)

func (n NetworkMode) String() string {
	return [...]string{"Full", "None", "Host"}[n]
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	MaxMemory    int64         // Maximum memory in bytes
	MaxCPU       float64       // CPU cores (0 = unlimited)
	MaxProcesses int           // Maximum number of processes
	MaxFileSize  int64         // Maximum file size in bytes
	MaxOpenFiles int           // Maximum number of open files
	Timeout      time.Duration // Execution timeout
}

// FilesystemSandbox configures filesystem access
type FilesystemSandbox struct {
	RootDir        string   // Root directory for command execution
	ReadOnlyPaths  []string // Paths that are read-only
	ReadWritePaths []string // Paths that are read-write
	TempDir        string   // Temporary directory
	IsolateFS      bool     // Use chroot or container
}

// NetworkSandbox configures network access
type NetworkSandbox struct {
	Mode         NetworkMode // Network access mode
	AllowedHosts []string    // Allowed hostnames/IPs
	AllowedPorts []int       // Allowed ports
	DNSServers   []string    // DNS servers to use
}

// SandboxConfig configures command sandboxing
type SandboxConfig struct {
	Enabled    bool              // Enable sandboxing
	Filesystem FilesystemSandbox // Filesystem restrictions
	Network    NetworkSandbox    // Network restrictions
	Resources  ResourceLimits    // Resource limits
}

// Sandbox implements command sandboxing
type Sandbox struct {
	config *SandboxConfig
}

// NewSandbox creates a new sandbox
func NewSandbox(config *SandboxConfig) *Sandbox {
	if config == nil {
		config = DefaultSandboxConfig()
	}
	return &Sandbox{
		config: config,
	}
}

// Apply applies sandbox restrictions to a command
func (s *Sandbox) Apply(cmd *exec.Cmd) error {
	if !s.config.Enabled {
		return nil
	}

	// Apply resource limits
	if err := s.applyResourceLimits(cmd); err != nil {
		return err
	}

	// Apply filesystem restrictions
	if err := s.applyFilesystemRestrictions(cmd); err != nil {
		return err
	}

	// Apply network restrictions (platform-dependent)
	if err := s.applyNetworkRestrictions(cmd); err != nil {
		return err
	}

	return nil
}

// applyResourceLimits applies resource limits
func (s *Sandbox) applyResourceLimits(cmd *exec.Cmd) error {
	// Only apply on Unix-like systems
	if runtime.GOOS == "windows" {
		return nil
	}

	// Initialize SysProcAttr if needed
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Set process group for easier cleanup
	cmd.SysProcAttr.Setpgid = true

	// Apply resource limits on Unix systems
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if err := s.applyUnixResourceLimits(cmd); err != nil {
			return err
		}
	}

	return nil
}

// applyUnixResourceLimits applies Unix-specific resource limits
func (s *Sandbox) applyUnixResourceLimits(cmd *exec.Cmd) error {
	limits := s.config.Resources

	// Create rlimits slice
	var rlimits []syscall.Rlimit

	// Memory limit (RLIMIT_AS - address space)
	if limits.MaxMemory > 0 {
		rlimits = append(rlimits, syscall.Rlimit{
			Cur: uint64(limits.MaxMemory),
			Max: uint64(limits.MaxMemory),
		})
	}

	// Process limit (RLIMIT_NPROC)
	if limits.MaxProcesses > 0 {
		rlimits = append(rlimits, syscall.Rlimit{
			Cur: uint64(limits.MaxProcesses),
			Max: uint64(limits.MaxProcesses),
		})
	}

	// File size limit (RLIMIT_FSIZE)
	if limits.MaxFileSize > 0 {
		rlimits = append(rlimits, syscall.Rlimit{
			Cur: uint64(limits.MaxFileSize),
			Max: uint64(limits.MaxFileSize),
		})
	}

	// Open files limit (RLIMIT_NOFILE)
	if limits.MaxOpenFiles > 0 {
		rlimits = append(rlimits, syscall.Rlimit{
			Cur: uint64(limits.MaxOpenFiles),
			Max: uint64(limits.MaxOpenFiles),
		})
	}

	// Note: Direct rlimit setting via SysProcAttr is platform-specific
	// For full implementation, would need to use syscall.Setrlimit in a pre-exec hook
	// This is a simplified version showing the structure

	return nil
}

// applyFilesystemRestrictions applies filesystem restrictions
func (s *Sandbox) applyFilesystemRestrictions(cmd *exec.Cmd) error {
	fs := s.config.Filesystem

	// Set working directory if specified
	if fs.RootDir != "" {
		cmd.Dir = fs.RootDir
	}

	// For full filesystem isolation, would need:
	// - chroot (requires root)
	// - containers (Docker, containerd)
	// - namespaces (Linux only)
	// This is a simplified implementation

	return nil
}

// applyNetworkRestrictions applies network restrictions
func (s *Sandbox) applyNetworkRestrictions(cmd *exec.Cmd) error {
	// Network isolation requires:
	// - Network namespaces (Linux)
	// - Firewall rules
	// - Container networking
	// This is a placeholder for future implementation

	return nil
}

// Validate validates the sandbox configuration
func (s *Sandbox) Validate() error {
	if !s.config.Enabled {
		return nil
	}

	// Validate resource limits
	if s.config.Resources.MaxMemory < 0 {
		return fmt.Errorf("invalid max memory: %d", s.config.Resources.MaxMemory)
	}

	if s.config.Resources.MaxCPU < 0 {
		return fmt.Errorf("invalid max CPU: %f", s.config.Resources.MaxCPU)
	}

	if s.config.Resources.MaxProcesses < 0 {
		return fmt.Errorf("invalid max processes: %d", s.config.Resources.MaxProcesses)
	}

	if s.config.Resources.MaxFileSize < 0 {
		return fmt.Errorf("invalid max file size: %d", s.config.Resources.MaxFileSize)
	}

	if s.config.Resources.MaxOpenFiles < 0 {
		return fmt.Errorf("invalid max open files: %d", s.config.Resources.MaxOpenFiles)
	}

	if s.config.Resources.Timeout < 0 {
		return fmt.Errorf("invalid timeout: %v", s.config.Resources.Timeout)
	}

	return nil
}

// DefaultSandboxConfig returns default sandbox configuration
func DefaultSandboxConfig() *SandboxConfig {
	return &SandboxConfig{
		Enabled: true,
		Filesystem: FilesystemSandbox{
			RootDir:        "",
			ReadOnlyPaths:  []string{},
			ReadWritePaths: []string{},
			TempDir:        "",
			IsolateFS:      false,
		},
		Network: NetworkSandbox{
			Mode:         NetworkFull,
			AllowedHosts: []string{},
			AllowedPorts: []int{},
			DNSServers:   []string{},
		},
		Resources: ResourceLimits{
			MaxMemory:    500 * 1024 * 1024, // 500 MB
			MaxCPU:       0,                  // Unlimited
			MaxProcesses: 20,
			MaxFileSize:  100 * 1024 * 1024, // 100 MB
			MaxOpenFiles: 1024,
			Timeout:      5 * time.Minute,
		},
	}
}

// SignalHandler manages process signals
type SignalHandler struct {
	processes sync.Map // map[string]*ProcessInfo
}

// ProcessInfo contains process information
type ProcessInfo struct {
	PID     int
	PGID    int // Process group ID
	Command string
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler() *SignalHandler {
	return &SignalHandler{}
}

// Register registers a process for signal handling
func (sh *SignalHandler) Register(id string, pid int, pgid int, command string) {
	sh.processes.Store(id, &ProcessInfo{
		PID:     pid,
		PGID:    pgid,
		Command: command,
	})
}

// Unregister unregisters a process
func (sh *SignalHandler) Unregister(id string) {
	sh.processes.Delete(id)
}

// Send sends a signal to a process
func (sh *SignalHandler) Send(id string, sig syscall.Signal) error {
	val, ok := sh.processes.Load(id)
	if !ok {
		return fmt.Errorf("process not found: %s", id)
	}

	info := val.(*ProcessInfo)

	// Send signal to process group if available, otherwise to process
	pid := info.PID
	if info.PGID > 0 {
		pid = -info.PGID // Negative PID sends to process group
	}

	return syscall.Kill(pid, sig)
}

// KillAll kills all registered processes
func (sh *SignalHandler) KillAll() {
	sh.processes.Range(func(key, value interface{}) bool {
		info := value.(*ProcessInfo)
		pid := info.PID
		if info.PGID > 0 {
			pid = -info.PGID
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
		return true
	})
}

// GracefulShutdown attempts graceful shutdown with timeout
func (sh *SignalHandler) GracefulShutdown(id string, timeout time.Duration) error {
	val, ok := sh.processes.Load(id)
	if !ok {
		return fmt.Errorf("process not found: %s", id)
	}

	info := val.(*ProcessInfo)
	pid := info.PID
	if info.PGID > 0 {
		pid = -info.PGID
	}

	// Send SIGTERM
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return err
	}

	// Wait for process to exit with timeout
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Check if process still exists
		if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
			// Process no longer exists
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill if timeout exceeded
	return syscall.Kill(pid, syscall.SIGKILL)
}

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
	if timeout > tm.maxTimeout && tm.maxTimeout > 0 {
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
