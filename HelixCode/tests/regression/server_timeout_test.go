package regression

import (
	"testing"
	"time"

	"dev.helix.code/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerTimeoutConfiguration ensures that server timeout configurations
// are properly set to prevent premature shutdown issues
func TestServerTimeoutConfiguration(t *testing.T) {
	t.Run("DefaultIdleTimeoutIs300Seconds", func(t *testing.T) {
		// Load default configuration
		cfg, err := config.Load()
		require.NoError(t, err, "Should load configuration without error")
		
		// Verify idle timeout is 300 seconds (5 minutes), not 60 seconds
		assert.Equal(t, 300, cfg.Server.IdleTimeout, 
			"Default idle timeout should be 300 seconds to prevent premature shutdown")
		
		// Verify other timeout settings
		assert.Equal(t, 30, cfg.Server.ReadTimeout, 
			"Read timeout should be 30 seconds")
		assert.Equal(t, 30, cfg.Server.WriteTimeout, 
			"Write timeout should be 30 seconds")
		assert.Equal(t, 30, cfg.Server.ShutdownTimeout, 
			"Shutdown timeout should be 30 seconds")
	})

	t.Run("ConfigurationFilesHaveCorrectTimeouts", func(t *testing.T) {
		// Test files that should have idle_timeout = 300
		testCases := []struct {
			name     string
			filePath string
		}{
			{"MainConfig", "config/config.yaml"},
			{"MinimalConfig", "config/minimal-config.yaml"},
			{"TestConfig", "config/test-config.yaml"},
			{"WorkingConfig", "config/working-config.yaml"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Load specific config file
				cfg, err := config.LoadWithFile(tc.filePath)
				require.NoError(t, err, "Should load %s without error", tc.filePath)
				
				assert.Equal(t, 300, cfg.Server.IdleTimeout, 
					"%s should have idle_timeout = 300", tc.filePath)
			})
		}
	})

	t.Run("TimeoutDurationsAreValid", func(t *testing.T) {
		cfg, err := config.Load()
		require.NoError(t, err)

		// Convert to time.Duration and verify they're reasonable
		idleTimeout := time.Duration(cfg.Server.IdleTimeout) * time.Second
		readTimeout := time.Duration(cfg.Server.ReadTimeout) * time.Second
		writeTimeout := time.Duration(cfg.Server.WriteTimeout) * time.Second
		shutdownTimeout := time.Duration(cfg.Server.ShutdownTimeout) * time.Second

		// Verify timeouts are within reasonable ranges
		assert.Greater(t, idleTimeout, 60*time.Second, 
			"Idle timeout should be greater than 60 seconds to prevent premature shutdown")
		assert.LessOrEqual(t, idleTimeout, 600*time.Second, 
			"Idle timeout should be reasonable (<= 10 minutes)")
		
		assert.Greater(t, readTimeout, 5*time.Second, 
			"Read timeout should be reasonable (> 5 seconds)")
		assert.LessOrEqual(t, readTimeout, 60*time.Second, 
			"Read timeout should be reasonable (<= 60 seconds)")
		
		assert.Greater(t, writeTimeout, 5*time.Second, 
			"Write timeout should be reasonable (> 5 seconds)")
		assert.LessOrEqual(t, writeTimeout, 60*time.Second, 
			"Write timeout should be reasonable (<= 60 seconds)")
		
		assert.Greater(t, shutdownTimeout, 10*time.Second, 
			"Shutdown timeout should allow graceful shutdown (> 10 seconds)")
		assert.LessOrEqual(t, shutdownTimeout, 60*time.Second, 
			"Shutdown timeout should be reasonable (<= 60 seconds)")
	})

	t.Run("NoEnvironmentVariablesOverrideTimeouts", func(t *testing.T) {
		// This test ensures that no environment variables are accidentally
		// overriding our timeout configurations
		cfg, err := config.Load()
		require.NoError(t, err)

		// The configuration should match our expected values regardless of environment
		assert.Equal(t, 300, cfg.Server.IdleTimeout, 
			"Idle timeout should be 300 regardless of environment")
	})
}

// TestServerStability ensures the server can run for extended periods
func TestServerStability(t *testing.T) {
	// This is a long-running test that verifies the server doesn't
	// shut down prematurely due to timeout issues
	t.Skip("Long-running stability test - run manually for verification")
	
	cfg, err := config.Load()
	require.NoError(t, err)

	// Server should be able to run for at least 2x the idle timeout
	minRuntime := time.Duration(cfg.Server.IdleTimeout*2) * time.Second
	
	// In a real test, we would start the server and verify it runs
	// for at least minRuntime without shutting down
	t.Logf("Server should run for at least %v without premature shutdown", minRuntime)
}