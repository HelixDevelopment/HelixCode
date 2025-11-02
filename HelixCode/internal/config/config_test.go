package config

import (
	"os"
	"path/filepath"
	"testing"

	"dev.helix.code/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  address: "0.0.0.0"
  port: 8080
auth:
  jwt_secret: "test-jwt-secret-for-testing"
database:
  host: "localhost"
  dbname: "test"
redis:
  enabled: false
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set config path environment variable
	oldConfig := os.Getenv("HELIX_CONFIG")
	defer os.Setenv("HELIX_CONFIG", oldConfig)
	os.Setenv("HELIX_CONFIG", configPath)

	// Test loading config
	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "0.0.0.0", cfg.Server.Address)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "test-jwt-secret-for-testing", cfg.Auth.JWTSecret)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Server: ServerConfig{Port: 8080},
				Database: database.Config{
					Host:   "localhost",
					DBName: "test",
				},
				Redis: RedisConfig{
					Host:    "localhost",
					Port:    6379,
					Enabled: true,
				},
				Auth: AuthConfig{
					JWTSecret: "test-secret",
				},
				Workers: WorkersConfig{
					HealthCheckInterval: 30,
					MaxConcurrentTasks:  10,
				},
				Tasks: TasksConfig{
					MaxRetries: 3,
				},
				LLM: LLMConfig{
					MaxTokens:   4096,
					Temperature: 0.7,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid server port",
			config: Config{
				Server: ServerConfig{Port: 99999},
			},
			wantErr: true,
		},
		{
			name: "missing database host",
			config: Config{
				Server: ServerConfig{Port: 8080},
				Database: database.Config{
					DBName: "test",
				},
			},
			wantErr: true,
		},
		{
			name: "default JWT secret",
			config: Config{
				Server: ServerConfig{Port: 8080},
				Database: database.Config{
					Host:   "localhost",
					DBName: "test",
				},
				Auth: AuthConfig{
					JWTSecret: "default-secret-change-in-production",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFindConfigFile(t *testing.T) {
	// Test with environment variable
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")
	err := os.WriteFile(configPath, []byte("test: content"), 0644)
	require.NoError(t, err)

	oldValue := os.Getenv("HELIX_CONFIG")
	defer os.Setenv("HELIX_CONFIG", oldValue)

	os.Setenv("HELIX_CONFIG", configPath)
	found := findConfigFile()
	assert.Equal(t, configPath, found)
}

func TestCreateDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	err := CreateDefaultConfig(configPath)
	assert.NoError(t, err)

	// Check if file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Check content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "server:")
	assert.Contains(t, string(content), "database:")
	assert.Contains(t, string(content), "redis:")
}

func TestGetEnvOrDefault(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	assert.Equal(t, "test_value", GetEnvOrDefault("TEST_VAR", "default"))

	// Test with non-existing env var
	assert.Equal(t, "default", GetEnvOrDefault("NON_EXISTING_VAR", "default"))
}

func TestGetEnvIntOrDefault(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_INT_VAR", "42")
	defer os.Unsetenv("TEST_INT_VAR")

	assert.Equal(t, 42, GetEnvIntOrDefault("TEST_INT_VAR", 10))

	// Test with non-existing env var
	assert.Equal(t, 10, GetEnvIntOrDefault("NON_EXISTING_INT_VAR", 10))

	// Test with invalid value
	os.Setenv("TEST_INVALID_INT", "not_a_number")
	defer os.Unsetenv("TEST_INVALID_INT")

	assert.Equal(t, 10, GetEnvIntOrDefault("TEST_INVALID_INT", 10))
}
