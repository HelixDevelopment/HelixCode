// Package shell provides secure, controlled, and efficient shell command execution
// for HelixCode. This package implements the ShellExecution module designed to
// execute shell commands with enhanced security, real-time streaming, and sandboxing
// capabilities.
//
// # Architecture
//
// The package is organized into several key components:
//
//   - Executor: Command execution with timeout and signal handling
//   - Sandbox: Resource limits and command restrictions
//   - OutputStreamer: Real-time output streaming to channels
//   - SecurityManager: Command validation and dangerous pattern detection
//   - SignalHandler: Process signal management (SIGINT, SIGTERM)
//
// # Security Features
//
// The shell package implements a security-first design:
//
//   - Command allowlist/blocklist validation
//   - Dangerous pattern detection (fork bombs, disk wiping, etc.)
//   - Path and environment sanitization
//   - Resource limits (CPU, memory, time)
//   - Network isolation (optional)
//   - Filesystem restrictions
//
// # Example Usage
//
// Basic command execution:
//
//	executor := shell.NewDefaultExecutor(shell.DefaultConfig())
//	cmd := &shell.Command{
//	    ID:      "example-1",
//	    Command: "echo 'Hello, World!'",
//	    Timeout: 30 * time.Second,
//	}
//	result, err := executor.Execute(context.Background(), cmd)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Stdout)
//
// Streaming execution:
//
//	cmd := &shell.Command{
//	    ID:      "example-stream",
//	    Command: "for i in 1 2 3; do echo $i; sleep 1; done",
//	}
//	exec, err := executor.ExecuteStream(context.Background(), cmd)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for line := range exec.Stdout {
//	    fmt.Println(line)
//	}
//	result := <-exec.Done
//
// Custom security configuration:
//
//	config := &shell.Config{
//	    Security: &shell.SecurityConfig{
//	        AllowlistMode: shell.AllowlistStrict,
//	        Allowlist:     map[string]bool{"ls": true, "cat": true},
//	        Blocklist:     map[string]bool{"rm": true, "dd": true},
//	    },
//	    Sandbox: &shell.SandboxConfig{
//	        Enabled: true,
//	        Resources: shell.ResourceLimits{
//	            MaxMemory:    100 * 1024 * 1024, // 100 MB
//	            MaxProcesses: 10,
//	            Timeout:      1 * time.Minute,
//	        },
//	    },
//	}
//	executor := shell.NewDefaultExecutor(config)
//
// # Design Inspiration
//
// This implementation is inspired by:
//
//   - Cline's shell execution: Real-time output streaming and terminal emulation
//   - Aider's command execution: Safe command execution and error handling
//   - Docker/containerd: Advanced sandboxing and resource management
//
// # References
//
// Technical Design: /Design/TechnicalDesigns/ShellExecution.md
package shell
