package worker

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// dummyPublicKey implements ssh.PublicKey for testing
type dummyPublicKey struct {
	keyType string
	data    []byte
}

func (dp *dummyPublicKey) Type() string    { return dp.keyType }
func (dp *dummyPublicKey) Marshal() []byte { return dp.data }
func (dp *dummyPublicKey) Verify(data []byte, sig *ssh.Signature) error {
	return fmt.Errorf("dummy key - not verifiable")
}

// TestSSHSecurity_HostKeyVerification tests secure host key verification
func TestSSHSecurity_HostKeyVerification(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		knownHosts  map[string][]ssh.PublicKey
		testHost    string
		testKey     ssh.PublicKey
		strictMode  bool
	}{
		{
			name:        "Known host with correct key",
			expectError: false,
			knownHosts: map[string][]ssh.PublicKey{
				"testhost": {&dummyPublicKey{keyType: "ssh-rsa", data: []byte("known-key")}},
			},
			testHost:   "testhost",
			testKey:    &dummyPublicKey{keyType: "ssh-rsa", data: []byte("known-key")},
			strictMode: true,
		},
		{
			name:        "Unknown host in strict mode",
			expectError: true,
			knownHosts: map[string][]ssh.PublicKey{
				"otherhost": {&dummyPublicKey{keyType: "ssh-rsa", data: []byte("other-key")}},
			},
			testHost:   "unknownhost",
			testKey:    &dummyPublicKey{keyType: "ssh-rsa", data: []byte("unknown-key")},
			strictMode: true,
		},
		{
			name:        "Known host with mismatched key",
			expectError: true,
			knownHosts: map[string][]ssh.PublicKey{
				"testhost": {&dummyPublicKey{keyType: "ssh-rsa", data: []byte("known-key")}},
			},
			testHost:   "testhost",
			testKey:    &dummyPublicKey{keyType: "ssh-rsa", data: []byte("different-key")},
			strictMode: true,
		},
		{
			name:        "Unknown host in permissive mode",
			expectError: false,
			knownHosts:  map[string][]ssh.PublicKey{},
			testHost:    "unknownhost",
			testKey:     &dummyPublicKey{keyType: "ssh-rsa", data: []byte("unknown-key")},
			strictMode:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			knownHostsFile := filepath.Join(tempDir, "known_hosts")
			hkm := NewHostKeyManager(knownHostsFile)
			hkm.knownHosts = tt.knownHosts

			verifyCallback := hkm.VerifyHostKey()
			err := verifyCallback(tt.testHost, &net.TCPAddr{}, tt.testKey)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSSHSecurity_SandboxIsolation tests worker sandbox isolation
func TestSSHSecurity_SandboxIsolation(t *testing.T) {
	ctx := context.Background()
	wim := NewWorkerIsolationManager()
	workerID := uuid.New()
	resources := Resources{
		TotalMemory: 512 * 1024 * 1024,
		CPUCount:    4,
	}

	sandbox, err := wim.CreateSandbox(ctx, workerID, resources)
	require.NoError(t, err)
	require.NotNil(t, sandbox)

	assert.NotEmpty(t, sandbox.Directory)
	assert.Contains(t, sandbox.User, "helix-")
	assert.Equal(t, workerID, sandbox.WorkerID)
	assert.Equal(t, int64(512*1024*1024), sandbox.MaxMemory)
	assert.Equal(t, float64(4), sandbox.MaxCPU)

	err = wim.CleanupSandbox(ctx, sandbox.ID)
	assert.NoError(t, err)

	_, err = wim.GetSandbox(sandbox.ID)
	assert.Error(t, err)
}

// TestSSHSecurity_KnownHostsFileManagement tests known hosts file operations
func TestSSHSecurity_KnownHostsFileManagement(t *testing.T) {
	tempDir := t.TempDir()
	knownHostsFile := filepath.Join(tempDir, "known_hosts")
	hkm := NewHostKeyManager(knownHostsFile)

	err := hkm.LoadKnownHosts()
	assert.NoError(t, err)

	_, err = os.Stat(knownHostsFile)
	assert.NoError(t, err)

	testKey := &dummyPublicKey{keyType: "ssh-rsa", data: []byte("test-key")}
	err = hkm.AddHostKey("testhost.com", testKey)
	assert.NoError(t, err)

	content, err := os.ReadFile(knownHostsFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "testhost.com")
}

// TestSSHSecurity_Integration tests full integration
func TestSSHSecurity_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := NewSSHWorkerPool(true)
	workerConfig := SSHWorkerConfig{
		Host:                  "localhost",
		Port:                  2222,
		Username:              "testuser",
		KeyPath:               filepath.Join(os.TempDir(), "test_key"),
		StrictHostKeyChecking: true,
	}

	worker := &SSHWorker{
		ID:        uuid.New(),
		Hostname:  workerConfig.Host,
		SSHConfig: &workerConfig,
	}

	err := pool.AddWorker(context.Background(), worker)
	assert.Contains(t, err.Error(), "SSH connection failed")

	assert.NotNil(t, pool.hostKeys)
	assert.NotNil(t, pool.isolation)
}
