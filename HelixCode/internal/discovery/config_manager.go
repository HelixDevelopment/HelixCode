package discovery

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrInvalidConfig is returned when configuration is invalid
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrConfigLocked is returned when configuration cannot be updated
	ErrConfigLocked = errors.New("configuration is locked")
)

// ConfigUpdateCallback is called when configuration is updated
type ConfigUpdateCallback func(oldConfig, newConfig DiscoveryConfig) error

// DiscoveryConfig represents the complete discovery system configuration
type DiscoveryConfig struct {
	// Port allocation settings
	PortRanges         map[string]PortRange
	ReservedPorts      []int
	AllowEphemeral     bool
	EphemeralPortStart int
	EphemeralPortEnd   int

	// Registry settings
	DefaultTTL           time.Duration
	CleanupInterval      time.Duration
	EnableHealthChecks   bool
	HealthCheckInterval  time.Duration
	HealthCheckTimeout   time.Duration
	MaxRetries           int

	// Broadcast settings
	BroadcastEnabled         bool
	MulticastAddress         string
	AnnouncementInterval     time.Duration
	DiscoveryTimeout         time.Duration
	BroadcastTTL             int

	// Discovery client settings
	EnableRegistry      bool
	EnableBroadcast     bool
	EnableDNS           bool
	DefaultPorts        map[string]int
	PreferredStrategies []DiscoveryStrategy

	// General settings
	MaxServices int
	LogLevel    string
}

// DefaultDiscoveryConfig returns default discovery configuration
func DefaultDiscoveryConfig() DiscoveryConfig {
	return DiscoveryConfig{
		// Port allocation
		PortRanges: map[string]PortRange{
			"database": {Start: 5433, End: 5442},
			"cache":    {Start: 6380, End: 6389},
			"api":      {Start: 8081, End: 8099},
			"grpc":     {Start: 9091, End: 9099},
			"metrics":  {Start: 9101, End: 9109},
			"general":  {Start: 10000, End: 10999},
		},
		ReservedPorts:      []int{5432, 6379, 8080, 9090, 9100},
		AllowEphemeral:     false,
		EphemeralPortStart: 49152,
		EphemeralPortEnd:   65535,

		// Registry
		DefaultTTL:          30 * time.Second,
		CleanupInterval:     10 * time.Second,
		EnableHealthChecks:  true,
		HealthCheckInterval: 5 * time.Second,
		HealthCheckTimeout:  2 * time.Second,
		MaxRetries:          3,

		// Broadcast
		BroadcastEnabled:     false,
		MulticastAddress:     DefaultMulticastAddress,
		AnnouncementInterval: DefaultAnnouncementInterval,
		DiscoveryTimeout:     DefaultDiscoveryTimeout,
		BroadcastTTL:         2,

		// Discovery client
		EnableRegistry:  true,
		EnableBroadcast: false,
		EnableDNS:       true,
		DefaultPorts: map[string]int{
			"database": 5432,
			"cache":    6379,
			"api":      8080,
			"grpc":     9090,
			"metrics":  9100,
		},
		PreferredStrategies: []DiscoveryStrategy{
			StrategyDefaultPort,
			StrategyRegistry,
			StrategyDNS,
		},

		// General
		MaxServices: 1000,
		LogLevel:    "info",
	}
}

// Validate validates the discovery configuration
func (c *DiscoveryConfig) Validate() error {
	// Validate port ranges
	for name, portRange := range c.PortRanges {
		if portRange.Start < 1 || portRange.Start > 65535 {
			return errors.New("invalid port range start: " + name)
		}
		if portRange.End < 1 || portRange.End > 65535 {
			return errors.New("invalid port range end: " + name)
		}
		if portRange.Start > portRange.End {
			return errors.New("invalid port range: start > end for " + name)
		}
	}

	// Validate ephemeral ports
	if c.AllowEphemeral {
		if c.EphemeralPortStart < 1024 || c.EphemeralPortStart > 65535 {
			return errors.New("invalid ephemeral port start")
		}
		if c.EphemeralPortEnd < 1024 || c.EphemeralPortEnd > 65535 {
			return errors.New("invalid ephemeral port end")
		}
		if c.EphemeralPortStart > c.EphemeralPortEnd {
			return errors.New("invalid ephemeral port range")
		}
	}

	// Validate timeouts
	if c.DefaultTTL < 0 {
		return errors.New("invalid default TTL")
	}
	if c.CleanupInterval < 0 {
		return errors.New("invalid cleanup interval")
	}
	if c.HealthCheckInterval < 0 {
		return errors.New("invalid health check interval")
	}
	if c.HealthCheckTimeout < 0 {
		return errors.New("invalid health check timeout")
	}
	if c.DiscoveryTimeout < 0 {
		return errors.New("invalid discovery timeout")
	}
	if c.AnnouncementInterval < 0 {
		return errors.New("invalid announcement interval")
	}

	// Validate max services
	if c.MaxServices < 1 {
		return errors.New("max services must be at least 1")
	}

	// Validate broadcast TTL
	if c.BroadcastTTL < 0 || c.BroadcastTTL > 255 {
		return errors.New("invalid broadcast TTL")
	}

	return nil
}

// ConfigManager manages discovery system configuration
type ConfigManager struct {
	config    DiscoveryConfig
	mu        sync.RWMutex
	locked    bool
	callbacks []ConfigUpdateCallback

	// Component references for applying config updates
	registry         *ServiceRegistry
	portAllocator    *PortAllocator
	broadcastService *BroadcastService
	discoveryClient  *DiscoveryClient
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(initialConfig DiscoveryConfig) (*ConfigManager, error) {
	if err := initialConfig.Validate(); err != nil {
		return nil, err
	}

	return &ConfigManager{
		config:    initialConfig,
		callbacks: make([]ConfigUpdateCallback, 0),
	}, nil
}

// GetConfig returns a copy of the current configuration
func (cm *ConfigManager) GetConfig() DiscoveryConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config
}

// UpdateConfig updates the configuration
func (cm *ConfigManager) UpdateConfig(newConfig DiscoveryConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.locked {
		return ErrConfigLocked
	}

	// Validate new configuration
	if err := newConfig.Validate(); err != nil {
		return err
	}

	oldConfig := cm.config

	// Notify callbacks
	for _, callback := range cm.callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			return err
		}
	}

	// Apply configuration
	cm.config = newConfig

	// Apply to components if registered
	if err := cm.applyToComponents(); err != nil {
		// Rollback on error
		cm.config = oldConfig
		return err
	}

	return nil
}

// UpdatePartial updates specific configuration fields
func (cm *ConfigManager) UpdatePartial(updateFn func(*DiscoveryConfig)) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.locked {
		return ErrConfigLocked
	}

	// Create a copy
	newConfig := cm.config

	// Apply updates
	updateFn(&newConfig)

	// Validate
	if err := newConfig.Validate(); err != nil {
		return err
	}

	oldConfig := cm.config

	// Notify callbacks
	for _, callback := range cm.callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			return err
		}
	}

	// Apply
	cm.config = newConfig

	// Apply to components
	if err := cm.applyToComponents(); err != nil {
		// Rollback on error
		cm.config = oldConfig
		return err
	}

	return nil
}

// Lock locks the configuration, preventing updates
func (cm *ConfigManager) Lock() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.locked = true
}

// Unlock unlocks the configuration, allowing updates
func (cm *ConfigManager) Unlock() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.locked = false
}

// IsLocked returns whether the configuration is locked
func (cm *ConfigManager) IsLocked() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.locked
}

// RegisterCallback registers a callback for configuration updates
func (cm *ConfigManager) RegisterCallback(callback ConfigUpdateCallback) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.callbacks = append(cm.callbacks, callback)
}

// RegisterComponents registers discovery components for configuration updates
func (cm *ConfigManager) RegisterComponents(
	registry *ServiceRegistry,
	allocator *PortAllocator,
	broadcast *BroadcastService,
	client *DiscoveryClient,
) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.registry = registry
	cm.portAllocator = allocator
	cm.broadcastService = broadcast
	cm.discoveryClient = client
}

// applyToComponents applies configuration to registered components
func (cm *ConfigManager) applyToComponents() error {
	// Note: This would require components to expose configuration update methods
	// For now, we just validate that components are compatible

	// In a full implementation, we would:
	// - Update port allocator ranges
	// - Update registry TTL and cleanup interval
	// - Update broadcast service settings
	// - Update discovery client strategies

	// Components would need methods like:
	// - portAllocator.UpdateConfig(config)
	// - registry.UpdateConfig(config)
	// - etc.

	return nil
}

// GetPortRange returns the port range for a service type
func (cm *ConfigManager) GetPortRange(serviceType string) (PortRange, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	portRange, exists := cm.config.PortRanges[serviceType]
	return portRange, exists
}

// SetPortRange sets the port range for a service type
func (cm *ConfigManager) SetPortRange(serviceType string, portRange PortRange) error {
	return cm.UpdatePartial(func(config *DiscoveryConfig) {
		if config.PortRanges == nil {
			config.PortRanges = make(map[string]PortRange)
		}
		config.PortRanges[serviceType] = portRange
	})
}

// EnableBroadcast enables or disables broadcast discovery
func (cm *ConfigManager) EnableBroadcast(enabled bool) error {
	return cm.UpdatePartial(func(config *DiscoveryConfig) {
		config.BroadcastEnabled = enabled
		config.EnableBroadcast = enabled
	})
}

// SetHealthCheckInterval sets the health check interval
func (cm *ConfigManager) SetHealthCheckInterval(interval time.Duration) error {
	return cm.UpdatePartial(func(config *DiscoveryConfig) {
		config.HealthCheckInterval = interval
	})
}

// SetDiscoveryStrategies sets the preferred discovery strategies
func (cm *ConfigManager) SetDiscoveryStrategies(strategies []DiscoveryStrategy) error {
	return cm.UpdatePartial(func(config *DiscoveryConfig) {
		config.PreferredStrategies = strategies
	})
}

// AddReservedPort adds a port to the reserved ports list
func (cm *ConfigManager) AddReservedPort(port int) error {
	return cm.UpdatePartial(func(config *DiscoveryConfig) {
		// Check if already reserved
		for _, p := range config.ReservedPorts {
			if p == port {
				return
			}
		}
		config.ReservedPorts = append(config.ReservedPorts, port)
	})
}

// RemoveReservedPort removes a port from the reserved ports list
func (cm *ConfigManager) RemoveReservedPort(port int) error {
	return cm.UpdatePartial(func(config *DiscoveryConfig) {
		newReserved := make([]int, 0, len(config.ReservedPorts))
		for _, p := range config.ReservedPorts {
			if p != port {
				newReserved = append(newReserved, p)
			}
		}
		config.ReservedPorts = newReserved
	})
}

// GetReservedPorts returns the list of reserved ports
func (cm *ConfigManager) GetReservedPorts() []int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy
	ports := make([]int, len(cm.config.ReservedPorts))
	copy(ports, cm.config.ReservedPorts)
	return ports
}

// ExportConfig exports the configuration as a map
func (cm *ConfigManager) ExportConfig() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"port_ranges":             cm.config.PortRanges,
		"reserved_ports":          cm.config.ReservedPorts,
		"allow_ephemeral":         cm.config.AllowEphemeral,
		"default_ttl":             cm.config.DefaultTTL.String(),
		"cleanup_interval":        cm.config.CleanupInterval.String(),
		"enable_health_checks":    cm.config.EnableHealthChecks,
		"health_check_interval":   cm.config.HealthCheckInterval.String(),
		"broadcast_enabled":       cm.config.BroadcastEnabled,
		"multicast_address":       cm.config.MulticastAddress,
		"enable_registry":         cm.config.EnableRegistry,
		"enable_broadcast":        cm.config.EnableBroadcast,
		"enable_dns":              cm.config.EnableDNS,
		"preferred_strategies":    cm.config.PreferredStrategies,
		"max_services":            cm.config.MaxServices,
		"log_level":               cm.config.LogLevel,
	}
}
