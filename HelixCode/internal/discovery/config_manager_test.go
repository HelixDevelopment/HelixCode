package discovery

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigManager(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)

	require.NoError(t, err)
	assert.NotNil(t, cm)
	assert.False(t, cm.IsLocked())
}

func TestNewConfigManager_InvalidConfig(t *testing.T) {
	config := DefaultDiscoveryConfig()
	config.MaxServices = 0 // Invalid

	_, err := NewConfigManager(config)
	assert.Error(t, err)
}

func TestDefaultDiscoveryConfig(t *testing.T) {
	config := DefaultDiscoveryConfig()

	// Verify port ranges
	assert.NotNil(t, config.PortRanges)
	assert.Contains(t, config.PortRanges, "database")
	assert.Contains(t, config.PortRanges, "cache")
	assert.Contains(t, config.PortRanges, "api")

	// Verify defaults
	assert.False(t, config.AllowEphemeral)
	assert.Equal(t, 30*time.Second, config.DefaultTTL)
	assert.True(t, config.EnableHealthChecks)
	assert.False(t, config.BroadcastEnabled)
	assert.Equal(t, 1000, config.MaxServices)
}

func TestDiscoveryConfig_Validate(t *testing.T) {
	tests := []struct {
		name         string
		modifyConfig func(*DiscoveryConfig)
		expectError  bool
	}{
		{
			name: "valid config",
			modifyConfig: func(c *DiscoveryConfig) {
				// Default config is valid
			},
			expectError: false,
		},
		{
			name: "invalid port range start",
			modifyConfig: func(c *DiscoveryConfig) {
				c.PortRanges["test"] = PortRange{Start: 0, End: 100}
			},
			expectError: true,
		},
		{
			name: "invalid port range end",
			modifyConfig: func(c *DiscoveryConfig) {
				c.PortRanges["test"] = PortRange{Start: 1000, End: 70000}
			},
			expectError: true,
		},
		{
			name: "port range start > end",
			modifyConfig: func(c *DiscoveryConfig) {
				c.PortRanges["test"] = PortRange{Start: 2000, End: 1000}
			},
			expectError: true,
		},
		{
			name: "invalid max services",
			modifyConfig: func(c *DiscoveryConfig) {
				c.MaxServices = 0
			},
			expectError: true,
		},
		{
			name: "invalid broadcast TTL",
			modifyConfig: func(c *DiscoveryConfig) {
				c.BroadcastTTL = 300
			},
			expectError: true,
		},
		{
			name: "negative health check interval",
			modifyConfig: func(c *DiscoveryConfig) {
				c.HealthCheckInterval = -1 * time.Second
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDiscoveryConfig()
			tt.modifyConfig(&config)

			err := config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigManager_GetConfig(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	retrieved := cm.GetConfig()
	assert.Equal(t, config.MaxServices, retrieved.MaxServices)
	assert.Equal(t, config.DefaultTTL, retrieved.DefaultTTL)
}

func TestConfigManager_UpdateConfig(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	newConfig := DefaultDiscoveryConfig()
	newConfig.MaxServices = 2000
	newConfig.BroadcastEnabled = true

	err = cm.UpdateConfig(newConfig)
	require.NoError(t, err)

	retrieved := cm.GetConfig()
	assert.Equal(t, 2000, retrieved.MaxServices)
	assert.True(t, retrieved.BroadcastEnabled)
}

func TestConfigManager_UpdateConfig_Invalid(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	newConfig := DefaultDiscoveryConfig()
	newConfig.MaxServices = 0 // Invalid

	err = cm.UpdateConfig(newConfig)
	assert.Error(t, err)

	// Config should remain unchanged
	retrieved := cm.GetConfig()
	assert.Equal(t, 1000, retrieved.MaxServices)
}

func TestConfigManager_UpdateConfig_Locked(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	cm.Lock()

	newConfig := DefaultDiscoveryConfig()
	err = cm.UpdateConfig(newConfig)
	assert.ErrorIs(t, err, ErrConfigLocked)
}

func TestConfigManager_UpdatePartial(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	err = cm.UpdatePartial(func(c *DiscoveryConfig) {
		c.MaxServices = 500
		c.LogLevel = "debug"
	})
	require.NoError(t, err)

	retrieved := cm.GetConfig()
	assert.Equal(t, 500, retrieved.MaxServices)
	assert.Equal(t, "debug", retrieved.LogLevel)
}

func TestConfigManager_LockUnlock(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	assert.False(t, cm.IsLocked())

	cm.Lock()
	assert.True(t, cm.IsLocked())

	cm.Unlock()
	assert.False(t, cm.IsLocked())
}

func TestConfigManager_RegisterCallback(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	callbackCalled := false
	cm.RegisterCallback(func(oldConfig, newConfig DiscoveryConfig) error {
		callbackCalled = true
		assert.Equal(t, 1000, oldConfig.MaxServices)
		assert.Equal(t, 2000, newConfig.MaxServices)
		return nil
	})

	newConfig := DefaultDiscoveryConfig()
	newConfig.MaxServices = 2000

	err = cm.UpdateConfig(newConfig)
	require.NoError(t, err)
	assert.True(t, callbackCalled)
}

func TestConfigManager_RegisterCallback_Error(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	testErr := errors.New("callback error")
	cm.RegisterCallback(func(oldConfig, newConfig DiscoveryConfig) error {
		return testErr
	})

	newConfig := DefaultDiscoveryConfig()
	newConfig.MaxServices = 2000

	err = cm.UpdateConfig(newConfig)
	assert.ErrorIs(t, err, testErr)

	// Config should remain unchanged
	retrieved := cm.GetConfig()
	assert.Equal(t, 1000, retrieved.MaxServices)
}

func TestConfigManager_GetPortRange(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	portRange, exists := cm.GetPortRange("database")
	assert.True(t, exists)
	assert.Equal(t, 5433, portRange.Start)
	assert.Equal(t, 5442, portRange.End)

	_, exists = cm.GetPortRange("nonexistent")
	assert.False(t, exists)
}

func TestConfigManager_SetPortRange(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	newRange := PortRange{Start: 7000, End: 7100}
	err = cm.SetPortRange("custom", newRange)
	require.NoError(t, err)

	portRange, exists := cm.GetPortRange("custom")
	assert.True(t, exists)
	assert.Equal(t, 7000, portRange.Start)
	assert.Equal(t, 7100, portRange.End)
}

func TestConfigManager_EnableBroadcast(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	assert.False(t, cm.GetConfig().BroadcastEnabled)

	err = cm.EnableBroadcast(true)
	require.NoError(t, err)

	retrieved := cm.GetConfig()
	assert.True(t, retrieved.BroadcastEnabled)
	assert.True(t, retrieved.EnableBroadcast)
}

func TestConfigManager_SetHealthCheckInterval(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	err = cm.SetHealthCheckInterval(10 * time.Second)
	require.NoError(t, err)

	retrieved := cm.GetConfig()
	assert.Equal(t, 10*time.Second, retrieved.HealthCheckInterval)
}

func TestConfigManager_SetDiscoveryStrategies(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	newStrategies := []DiscoveryStrategy{
		StrategyBroadcast,
		StrategyRegistry,
		StrategyDNS,
	}

	err = cm.SetDiscoveryStrategies(newStrategies)
	require.NoError(t, err)

	retrieved := cm.GetConfig()
	assert.Equal(t, newStrategies, retrieved.PreferredStrategies)
}

func TestConfigManager_AddReservedPort(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	err = cm.AddReservedPort(3000)
	require.NoError(t, err)

	ports := cm.GetReservedPorts()
	assert.Contains(t, ports, 3000)
}

func TestConfigManager_AddReservedPort_Duplicate(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	err = cm.AddReservedPort(3000)
	require.NoError(t, err)

	err = cm.AddReservedPort(3000)
	require.NoError(t, err)

	ports := cm.GetReservedPorts()

	count := 0
	for _, p := range ports {
		if p == 3000 {
			count++
		}
	}
	assert.Equal(t, 1, count, "Port should only be added once")
}

func TestConfigManager_RemoveReservedPort(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	err = cm.AddReservedPort(3000)
	require.NoError(t, err)

	err = cm.RemoveReservedPort(3000)
	require.NoError(t, err)

	ports := cm.GetReservedPorts()
	assert.NotContains(t, ports, 3000)
}

func TestConfigManager_ExportConfig(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	exported := cm.ExportConfig()

	assert.NotNil(t, exported)
	assert.Contains(t, exported, "port_ranges")
	assert.Contains(t, exported, "max_services")
	assert.Contains(t, exported, "log_level")
	assert.Equal(t, 1000, exported["max_services"])
	assert.Equal(t, "info", exported["log_level"])
}

func TestConfigManager_ConcurrentAccess(t *testing.T) {
	config := DefaultDiscoveryConfig()
	cm, err := NewConfigManager(config)
	require.NoError(t, err)

	done := make(chan bool)

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cm.GetConfig()
				cm.GetPortRange("database")
				cm.GetReservedPorts()
			}
			done <- true
		}()
	}

	// Concurrent writers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				cm.UpdatePartial(func(c *DiscoveryConfig) {
					c.MaxServices = 1000 + id
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	// Should not panic or deadlock
	assert.NotNil(t, cm.GetConfig())
}
