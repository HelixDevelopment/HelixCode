package discovery

import (
	"errors"
	"fmt"
	"net"
	"time"
)

var (
	// ErrServiceUnavailable is returned when a service cannot be discovered
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrInvalidServiceName is returned when the service name is invalid
	ErrInvalidServiceName = errors.New("invalid service name")
)

// DiscoveryStrategy represents the strategy used to discover a service
type DiscoveryStrategy string

const (
	// StrategyDefaultPort tries the default/well-known port first
	StrategyDefaultPort DiscoveryStrategy = "default_port"

	// StrategyRegistry queries the service registry
	StrategyRegistry DiscoveryStrategy = "registry"

	// StrategyBroadcast uses UDP multicast discovery (Phase 2)
	StrategyBroadcast DiscoveryStrategy = "broadcast"

	// StrategyDNS falls back to DNS resolution
	StrategyDNS DiscoveryStrategy = "dns"
)

// DiscoveryResult contains information about a discovered service
type DiscoveryResult struct {
	ServiceInfo *ServiceInfo
	Strategy    DiscoveryStrategy
	Latency     time.Duration
}

// DiscoveryClientConfig configures the discovery client
type DiscoveryClientConfig struct {
	// Registry is the service registry to query
	Registry *ServiceRegistry

	// PortAllocator is the port allocator for managing ports
	PortAllocator *PortAllocator

	// BroadcastService is the broadcast service for UDP multicast discovery
	BroadcastService *BroadcastService

	// DefaultPorts maps service names to their default ports
	DefaultPorts map[string]int

	// EnableRegistry enables registry-based discovery
	EnableRegistry bool

	// EnableBroadcast enables broadcast-based discovery (Phase 2)
	EnableBroadcast bool

	// EnableDNS enables DNS fallback
	EnableDNS bool

	// DiscoveryTimeout is the timeout for discovery operations
	DiscoveryTimeout time.Duration

	// PreferredStrategies defines the order of discovery strategies
	PreferredStrategies []DiscoveryStrategy
}

// DefaultDiscoveryClientConfig returns default configuration
func DefaultDiscoveryClientConfig(registry *ServiceRegistry, allocator *PortAllocator) DiscoveryClientConfig {
	return DiscoveryClientConfig{
		Registry:      registry,
		PortAllocator: allocator,
		DefaultPorts: map[string]int{
			"database": 5432,
			"cache":    6379,
			"api":      8080,
			"grpc":     9090,
			"metrics":  9100,
		},
		EnableRegistry:  true,
		EnableBroadcast: false, // Phase 2
		EnableDNS:       true,
		DiscoveryTimeout: 5 * time.Second,
		PreferredStrategies: []DiscoveryStrategy{
			StrategyDefaultPort,
			StrategyRegistry,
			StrategyDNS,
		},
	}
}

// DiscoveryClient provides service discovery capabilities
type DiscoveryClient struct {
	config DiscoveryClientConfig
}

// NewDiscoveryClient creates a new discovery client
func NewDiscoveryClient(config DiscoveryClientConfig) *DiscoveryClient {
	return &DiscoveryClient{
		config: config,
	}
}

// Discover attempts to discover a service using configured strategies
func (c *DiscoveryClient) Discover(serviceName string) (*DiscoveryResult, error) {
	if serviceName == "" {
		return nil, ErrInvalidServiceName
	}

	startTime := time.Now()

	// Try each strategy in order
	for _, strategy := range c.config.PreferredStrategies {
		var result *DiscoveryResult
		var err error

		switch strategy {
		case StrategyDefaultPort:
			result, err = c.discoverByDefaultPort(serviceName)
		case StrategyRegistry:
			result, err = c.discoverByRegistry(serviceName)
		case StrategyBroadcast:
			result, err = c.discoverByBroadcast(serviceName)
		case StrategyDNS:
			result, err = c.discoverByDNS(serviceName)
		}

		if err == nil && result != nil {
			result.Latency = time.Since(startTime)
			return result, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrServiceUnavailable, serviceName)
}

// DiscoverWithTimeout attempts to discover a service with a timeout
func (c *DiscoveryClient) DiscoverWithTimeout(serviceName string, timeout time.Duration) (*DiscoveryResult, error) {
	resultChan := make(chan *DiscoveryResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := c.Discover(serviceName)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("discovery timeout after %v for service: %s", timeout, serviceName)
	}
}

// Register registers a service with the discovery system
func (c *DiscoveryClient) Register(info ServiceInfo) error {
	if !c.config.EnableRegistry || c.config.Registry == nil {
		return errors.New("registry not enabled or not configured")
	}

	// Allocate port if needed
	if info.Port == 0 {
		defaultPort := c.getDefaultPort(info.Name)
		allocatedPort, err := c.config.PortAllocator.AllocatePort(info.Name, defaultPort)
		if err != nil {
			return fmt.Errorf("failed to allocate port: %w", err)
		}
		info.Port = allocatedPort
	}

	// Register with registry
	return c.config.Registry.Register(info)
}

// Deregister removes a service from the discovery system
func (c *DiscoveryClient) Deregister(serviceName string) error {
	if !c.config.EnableRegistry || c.config.Registry == nil {
		return errors.New("registry not enabled or not configured")
	}

	// Release port
	if c.config.PortAllocator != nil {
		c.config.PortAllocator.ReleaseServicePort(serviceName)
	}

	// Deregister from registry
	return c.config.Registry.Deregister(serviceName)
}

// Heartbeat sends a heartbeat for a service
func (c *DiscoveryClient) Heartbeat(serviceName string) error {
	if !c.config.EnableRegistry || c.config.Registry == nil {
		return errors.New("registry not enabled or not configured")
	}

	return c.config.Registry.Heartbeat(serviceName)
}

// ListServices returns all registered services
func (c *DiscoveryClient) ListServices() []*ServiceInfo {
	if !c.config.EnableRegistry || c.config.Registry == nil {
		return []*ServiceInfo{}
	}

	return c.config.Registry.List()
}

// ListHealthyServices returns only healthy services
func (c *DiscoveryClient) ListHealthyServices() []*ServiceInfo {
	if !c.config.EnableRegistry || c.config.Registry == nil {
		return []*ServiceInfo{}
	}

	return c.config.Registry.ListHealthy()
}

// Discovery strategy implementations

func (c *DiscoveryClient) discoverByDefaultPort(serviceName string) (*DiscoveryResult, error) {
	defaultPort := c.getDefaultPort(serviceName)
	if defaultPort == 0 {
		return nil, errors.New("no default port configured")
	}

	// Check if the port is reachable
	address := fmt.Sprintf("localhost:%d", defaultPort)
	if c.isPortReachable(address, 100*time.Millisecond) {
		return &DiscoveryResult{
			ServiceInfo: &ServiceInfo{
				Name:     serviceName,
				Host:     "localhost",
				Port:     defaultPort,
				Protocol: "tcp",
				Healthy:  true,
			},
			Strategy: StrategyDefaultPort,
		}, nil
	}

	return nil, errors.New("default port not reachable")
}

func (c *DiscoveryClient) discoverByRegistry(serviceName string) (*DiscoveryResult, error) {
	if !c.config.EnableRegistry || c.config.Registry == nil {
		return nil, errors.New("registry not enabled")
	}

	serviceInfo, err := c.config.Registry.Get(serviceName)
	if err != nil {
		return nil, err
	}

	// Verify service is healthy and not expired
	if !serviceInfo.Healthy || serviceInfo.IsExpired() {
		return nil, errors.New("service unhealthy or expired")
	}

	return &DiscoveryResult{
		ServiceInfo: serviceInfo,
		Strategy:    StrategyRegistry,
	}, nil
}

func (c *DiscoveryClient) discoverByBroadcast(serviceName string) (*DiscoveryResult, error) {
	if !c.config.EnableBroadcast {
		return nil, errors.New("broadcast discovery not enabled")
	}

	if c.config.BroadcastService == nil {
		return nil, errors.New("broadcast service not configured")
	}

	// Ensure broadcast service is running
	if !c.config.BroadcastService.IsRunning() {
		if err := c.config.BroadcastService.Start(); err != nil {
			return nil, fmt.Errorf("failed to start broadcast service: %w", err)
		}
	}

	// Discover service via broadcast
	serviceInfo, err := c.config.BroadcastService.Discover(serviceName)
	if err != nil {
		return nil, err
	}

	return &DiscoveryResult{
		ServiceInfo: serviceInfo,
		Strategy:    StrategyBroadcast,
	}, nil
}

func (c *DiscoveryClient) discoverByDNS(serviceName string) (*DiscoveryResult, error) {
	if !c.config.EnableDNS {
		return nil, errors.New("DNS discovery not enabled")
	}

	// Try DNS lookup
	addresses, err := net.LookupHost(serviceName)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed: %w", err)
	}

	if len(addresses) == 0 {
		return nil, errors.New("no addresses found in DNS")
	}

	// Use first address and try to determine port
	host := addresses[0]
	port := c.getDefaultPort(serviceName)
	if port == 0 {
		port = 80 // Default to HTTP port
	}

	return &DiscoveryResult{
		ServiceInfo: &ServiceInfo{
			Name:     serviceName,
			Host:     host,
			Port:     port,
			Protocol: "tcp",
			Healthy:  true,
		},
		Strategy: StrategyDNS,
	}, nil
}

// Helper methods

func (c *DiscoveryClient) getDefaultPort(serviceName string) int {
	if port, exists := c.config.DefaultPorts[serviceName]; exists {
		return port
	}

	// Try to match by service type keywords
	if contains(serviceName, "postgres", "postgresql", "pg") {
		return 5432
	}
	if contains(serviceName, "redis", "cache") {
		return 6379
	}
	if contains(serviceName, "grpc") {
		return 9090
	}
	if contains(serviceName, "metrics", "prometheus") {
		return 9100
	}
	if contains(serviceName, "api", "http") {
		return 8080
	}

	return 0 // No default port
}

func (c *DiscoveryClient) isPortReachable(address string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetServiceAddress is a convenience method to get the full address of a service
func (c *DiscoveryClient) GetServiceAddress(serviceName string) (string, error) {
	result, err := c.Discover(serviceName)
	if err != nil {
		return "", err
	}

	return result.ServiceInfo.Address(), nil
}

// WaitForService waits for a service to become available
func (c *DiscoveryClient) WaitForService(serviceName string, maxWait time.Duration) (*DiscoveryResult, error) {
	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		result, err := c.Discover(serviceName)
		if err == nil {
			return result, nil
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for service %s after %v", serviceName, maxWait)
			}
		}
	}
}
