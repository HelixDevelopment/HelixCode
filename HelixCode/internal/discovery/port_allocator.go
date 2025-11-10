package discovery

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	// ErrNoPortsAvailable is returned when no ports are available in the configured range
	ErrNoPortsAvailable = errors.New("no ports available in configured range")

	// ErrInvalidPortRange is returned when the port range is invalid
	ErrInvalidPortRange = errors.New("invalid port range")

	// ErrPortAlreadyAllocated is returned when trying to allocate an already allocated port
	ErrPortAlreadyAllocated = errors.New("port already allocated")

	// ErrPortNotAllocated is returned when trying to release a port that wasn't allocated
	ErrPortNotAllocated = errors.New("port not allocated")
)

// PortRange defines a range of ports for a service type
type PortRange struct {
	Start int
	End   int
}

// PortAllocation represents an allocated port
type PortAllocation struct {
	Port        int
	ServiceName string
	AllocatedAt time.Time
}

// PortAllocatorConfig configures the port allocator
type PortAllocatorConfig struct {
	// AllowEphemeral allows allocation of ephemeral ports when ranges are exhausted
	AllowEphemeral bool

	// PortRanges defines port ranges for different service types
	PortRanges map[string]PortRange

	// ReservedPorts are ports that should never be allocated
	ReservedPorts []int
}

// DefaultPortAllocatorConfig returns default configuration
func DefaultPortAllocatorConfig() PortAllocatorConfig {
	return PortAllocatorConfig{
		AllowEphemeral: false,
		PortRanges: map[string]PortRange{
			"database":  {Start: 5433, End: 5442},
			"cache":     {Start: 6380, End: 6389},
			"api":       {Start: 8081, End: 8099},
			"grpc":      {Start: 9091, End: 9109},
			"metrics":   {Start: 9100, End: 9199},
			"websocket": {Start: 8001, End: 8020},
		},
		ReservedPorts: []int{22, 80, 443, 3306, 5432, 6379, 8080, 9090},
	}
}

// PortAllocator manages port allocation with fallback mechanisms
type PortAllocator struct {
	config      PortAllocatorConfig
	allocations map[int]*PortAllocation
	serviceMap  map[string]int
	mu          sync.RWMutex
}

// NewPortAllocator creates a new port allocator
func NewPortAllocator(config PortAllocatorConfig) *PortAllocator {
	return &PortAllocator{
		config:      config,
		allocations: make(map[int]*PortAllocation),
		serviceMap:  make(map[string]int),
	}
}

// NewDefaultPortAllocator creates a port allocator with default configuration
func NewDefaultPortAllocator() *PortAllocator {
	return NewPortAllocator(DefaultPortAllocatorConfig())
}

// AllocatePort allocates a port for a service, preferring the specified port
// If the preferred port is unavailable, it falls back to the range for the service type
func (pa *PortAllocator) AllocatePort(serviceName string, preferredPort int) (int, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// Check if service already has a port
	if existingPort, exists := pa.serviceMap[serviceName]; exists {
		return existingPort, nil
	}

	// Try preferred port first (if not reserved and available)
	if !pa.isReserved(preferredPort) && pa.isPortAvailableUnsafe(preferredPort) {
		return pa.reservePortUnsafe(preferredPort, serviceName)
	}

	// Get service type from service name
	serviceType := pa.getServiceType(serviceName)

	// Try fallback range
	portRange, exists := pa.config.PortRanges[serviceType]
	if exists {
		port, err := pa.allocateFromRangeUnsafe(serviceName, portRange)
		if err == nil {
			return port, nil
		}
	}

	// Try ephemeral if allowed
	if pa.config.AllowEphemeral {
		return pa.allocateEphemeralPortUnsafe(serviceName)
	}

	return 0, ErrNoPortsAvailable
}

// AllocatePortInRange allocates a port within a specific range
func (pa *PortAllocator) AllocatePortInRange(serviceName string, startPort, endPort int) (int, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	if startPort < 1 || endPort > 65535 || startPort > endPort {
		return 0, ErrInvalidPortRange
	}

	// Check if service already has a port
	if existingPort, exists := pa.serviceMap[serviceName]; exists {
		return existingPort, nil
	}

	return pa.allocateFromRangeUnsafe(serviceName, PortRange{Start: startPort, End: endPort})
}

// ReleasePort releases a previously allocated port
func (pa *PortAllocator) ReleasePort(port int) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	allocation, exists := pa.allocations[port]
	if !exists {
		return ErrPortNotAllocated
	}

	delete(pa.allocations, port)
	delete(pa.serviceMap, allocation.ServiceName)

	return nil
}

// ReleaseServicePort releases the port allocated to a service
func (pa *PortAllocator) ReleaseServicePort(serviceName string) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	port, exists := pa.serviceMap[serviceName]
	if !exists {
		return ErrPortNotAllocated
	}

	delete(pa.allocations, port)
	delete(pa.serviceMap, serviceName)

	return nil
}

// IsPortAvailable checks if a port is available for binding
func (pa *PortAllocator) IsPortAvailable(port int) bool {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	return pa.isPortAvailableUnsafe(port)
}

// GetPortForService returns the port allocated to a service
func (pa *PortAllocator) GetPortForService(serviceName string) (int, bool) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	port, exists := pa.serviceMap[serviceName]
	return port, exists
}

// GetAllocation returns allocation details for a port
func (pa *PortAllocator) GetAllocation(port int) (*PortAllocation, bool) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	allocation, exists := pa.allocations[port]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent modification
	allocationCopy := *allocation
	return &allocationCopy, true
}

// ListAllocations returns all current port allocations
func (pa *PortAllocator) ListAllocations() []*PortAllocation {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	allocations := make([]*PortAllocation, 0, len(pa.allocations))
	for _, allocation := range pa.allocations {
		allocationCopy := *allocation
		allocations = append(allocations, &allocationCopy)
	}

	return allocations
}

// Internal helper methods (must be called with lock held)

func (pa *PortAllocator) isPortAvailableUnsafe(port int) bool {
	// Check if already allocated
	if _, exists := pa.allocations[port]; exists {
		return false
	}

	// Check if reserved
	if pa.isReserved(port) {
		return false
	}

	// Try to bind to the port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()

	return true
}

func (pa *PortAllocator) reservePortUnsafe(port int, serviceName string) (int, error) {
	if _, exists := pa.allocations[port]; exists {
		return 0, ErrPortAlreadyAllocated
	}

	allocation := &PortAllocation{
		Port:        port,
		ServiceName: serviceName,
		AllocatedAt: time.Now(),
	}

	pa.allocations[port] = allocation
	pa.serviceMap[serviceName] = port

	return port, nil
}

func (pa *PortAllocator) allocateFromRangeUnsafe(serviceName string, portRange PortRange) (int, error) {
	for port := portRange.Start; port <= portRange.End; port++ {
		if pa.isPortAvailableUnsafe(port) {
			return pa.reservePortUnsafe(port, serviceName)
		}
	}

	return 0, ErrNoPortsAvailable
}

func (pa *PortAllocator) allocateEphemeralPortUnsafe(serviceName string) (int, error) {
	// Let the OS assign an ephemeral port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to allocate ephemeral port: %w", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	listener.Close()

	// Reserve the port
	return pa.reservePortUnsafe(port, serviceName)
}

func (pa *PortAllocator) isReserved(port int) bool {
	for _, reserved := range pa.config.ReservedPorts {
		if port == reserved {
			return true
		}
	}
	return false
}

func (pa *PortAllocator) getServiceType(serviceName string) string {
	// Simple heuristic to determine service type from name
	// This can be enhanced based on actual service naming conventions

	if contains(serviceName, "postgres", "pg", "database", "db") {
		return "database"
	}
	if contains(serviceName, "redis", "cache", "memcache") {
		return "cache"
	}
	if contains(serviceName, "grpc") {
		return "grpc"
	}
	if contains(serviceName, "metrics", "prometheus", "prom") {
		return "metrics"
	}
	if contains(serviceName, "websocket", "ws") {
		return "websocket"
	}

	// Default to api
	return "api"
}

func contains(s string, substrings ...string) bool {
	lower := s
	for _, sub := range substrings {
		if len(lower) >= len(sub) {
			for i := 0; i <= len(lower)-len(sub); i++ {
				match := true
				for j := 0; j < len(sub); j++ {
					if lower[i+j] != sub[j] && lower[i+j] != sub[j]-32 && lower[i+j] != sub[j]+32 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}
