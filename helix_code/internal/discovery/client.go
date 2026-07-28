package discovery

import (
	"context"
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
		EnableRegistry:   true,
		EnableBroadcast:  false, // Phase 2
		EnableDNS:        true,
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

	return c.discoverTimeoutSelect(resultChan, errorChan, time.After(timeout), serviceName, timeout)
}

// discoverTimeoutSelect implements DiscoverWithTimeout's select decision.
// Extracted (identical behaviour) from DiscoverWithTimeout so the §11.4.115
// regression guard (client_racefix_test.go) can force the race
// deterministically -- pre-buffering a real result/error onto
// resultChan/errorChan AND pre-firing timeoutC BEFORE this function (and its
// internal select) is ever invoked, so both are provably ready at the exact
// instant the select is evaluated, on every single trial, with zero timing
// dependency.
//
// ANTI-BLUFF FIX (§11.4.115 unprioritized-select race, fourth confirmed
// instance in this codebase this session, after
// internal/persistence/store.go autoSaveTick, helix_agent's
// lazy_provider.go createProviderWithContext, and the applications/{desktop,
// harmony_os,aurora_os} background-update-loop teardown races): Go's select
// chooses UNIFORMLY AT RANDOM among ALL cases ready at the instant it is
// evaluated. A naked
// `select { case result := <-resultChan: ...; case err := <-errorChan: ...;
// case <-time.After(timeout): return timeout-error }`
// can therefore discard a REAL discovered result (or a real discovery error)
// in favour of a spurious timeout error, if the discovery goroutine's answer
// becomes ready at (scheduler-)nearly the same instant the timeout fires --
// a narrower, single-instant version of the loop-teardown races fixed
// elsewhere in this batch (one decision, not a repeated select), but the same
// random-pick hazard applies. The non-blocking priority pre-check below
// closes the wide window: it runs FIRST, before the blocking select, and
// returns the real answer immediately whenever it is already available.
//
// RESIDUAL WINDOW (closed by the inner re-check inside the timeout case):
// the pre-check only proves the channels were empty at the instant it ran.
// If the discovery goroutine's send lands in the gap BETWEEN the pre-check
// returning and the blocking select below being evaluated -- and the timeout
// has ALSO already fired -- both cases are ready when the blocking select
// runs, and Go's uniform-random pick can still choose the timeout branch.
// The inner re-check inside the timeout case catches this: any real
// result/error observed there is returned instead of the timeout error.
//
// HONESTY (§11.4.6): this is the strongest ordering guarantee a plain
// channel select can offer -- it is NOT a guarantee the race is closed to
// zero width. Under Go's async goroutine preemption (>=1.14) this goroutine
// can still be preempted between the final re-check and the `return` that
// follows it while, in principle, an alternative outcome becomes available a
// nanosecond later -- an unobservable, unavoidable race no select-based
// implementation can provably eliminate. What IS delivered: a real
// result/error observed at the final decision point ALWAYS wins over the
// timeout; a genuinely-not-yet-produced answer at that same instant
// legitimately times out.
func (c *DiscoveryClient) discoverTimeoutSelect(resultChan <-chan *DiscoveryResult, errorChan <-chan error, timeoutC <-chan time.Time, serviceName string, timeout time.Duration) (*DiscoveryResult, error) {
	// Priority pre-check: a real answer already sitting on either channel
	// MUST win over a timeout -- even one that has also already fired.
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	default:
	}

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-timeoutC:
		// Re-check: a result/error can become ready in the window between
		// the pre-check above and this blocking select being evaluated,
		// leaving the timeout branch "randomly" chosen even though a real
		// answer now exists. Catch it here, non-blocking, before discarding
		// it in favour of a spurious timeout error.
		select {
		case result := <-resultChan:
			return result, nil
		case err := <-errorChan:
			return nil, err
		default:
		}
		return nil, fmt.Errorf("%s", tr(context.Background(), "internal_discovery_timeout_after_for_service", map[string]any{"Timeout": timeout.String(), "ServiceName": serviceName}))
	}
}

// Register registers a service with the discovery system
func (c *DiscoveryClient) Register(info ServiceInfo) error {
	if !c.config.EnableRegistry || c.config.Registry == nil {
		return errors.New(tr(context.Background(), "internal_discovery_registry_not_enabled_or_not_configured", nil))
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
		return errors.New(tr(context.Background(), "internal_discovery_registry_not_enabled_or_not_configured", nil))
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
		return errors.New(tr(context.Background(), "internal_discovery_registry_not_enabled_or_not_configured", nil))
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
		return nil, errors.New(tr(context.Background(), "internal_discovery_no_default_port_configured", nil))
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

	return nil, errors.New(tr(context.Background(), "internal_discovery_default_port_not_reachable", nil))
}

func (c *DiscoveryClient) discoverByRegistry(serviceName string) (*DiscoveryResult, error) {
	if !c.config.EnableRegistry || c.config.Registry == nil {
		return nil, errors.New(tr(context.Background(), "internal_discovery_registry_not_enabled", nil))
	}

	serviceInfo, err := c.config.Registry.Get(serviceName)
	if err != nil {
		return nil, err
	}

	// Verify service is healthy and not expired
	if !serviceInfo.Healthy || serviceInfo.IsExpired() {
		return nil, errors.New(tr(context.Background(), "internal_discovery_service_unhealthy_or_expired", nil))
	}

	return &DiscoveryResult{
		ServiceInfo: serviceInfo,
		Strategy:    StrategyRegistry,
	}, nil
}

func (c *DiscoveryClient) discoverByBroadcast(serviceName string) (*DiscoveryResult, error) {
	if !c.config.EnableBroadcast {
		return nil, errors.New(tr(context.Background(), "internal_discovery_broadcast_discovery_not_enabled", nil))
	}

	if c.config.BroadcastService == nil {
		return nil, errors.New(tr(context.Background(), "internal_discovery_broadcast_service_not_configured", nil))
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
		return nil, errors.New(tr(context.Background(), "internal_discovery_dns_discovery_not_enabled", nil))
	}

	// Bound DNS lookup with a short context so a slow or unreachable resolver
	// cannot block past the caller's discovery budget (e.g. WaitForService's
	// maxWait). System DNS resolution can otherwise stall many seconds under
	// load or with mDNS/Avahi misconfiguration before NXDOMAIN propagates.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupHost(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed: %w", err)
	}

	if len(addresses) == 0 {
		return nil, errors.New(tr(context.Background(), "internal_discovery_no_addresses_found_in_dns", nil))
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
