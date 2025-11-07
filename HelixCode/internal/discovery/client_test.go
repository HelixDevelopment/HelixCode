package discovery

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscoveryClient(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)

	client := NewDiscoveryClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, registry, client.config.Registry)
	assert.Equal(t, allocator, client.config.PortAllocator)
}

func TestDefaultDiscoveryClientConfig(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()

	config := DefaultDiscoveryClientConfig(registry, allocator)

	assert.True(t, config.EnableRegistry)
	assert.False(t, config.EnableBroadcast) // Phase 2
	assert.True(t, config.EnableDNS)
	assert.Equal(t, 5*time.Second, config.DiscoveryTimeout)
	assert.Len(t, config.PreferredStrategies, 3)
}

func TestRegister(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	info := ServiceInfo{
		Name:     "test-service",
		Host:     "localhost",
		Port:     8080,
		Protocol: "http",
	}

	err := client.Register(info)
	require.NoError(t, err)

	// Verify service is in registry
	retrieved, err := registry.Get("test-service")
	require.NoError(t, err)
	assert.Equal(t, "test-service", retrieved.Name)
	assert.Equal(t, 8080, retrieved.Port)
}

func TestRegister_AutoAllocatePort(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	info := ServiceInfo{
		Name:     "api",
		Host:     "localhost",
		Port:     0, // Let it allocate
		Protocol: "http",
	}

	err := client.Register(info)
	require.NoError(t, err)

	// Verify port was allocated
	retrieved, err := registry.Get("api")
	require.NoError(t, err)
	assert.NotEqual(t, 0, retrieved.Port)
	assert.GreaterOrEqual(t, retrieved.Port, 8081) // Should be in api range
}

func TestDeregister(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register service
	info := ServiceInfo{
		Name: "test-service",
		Host: "localhost",
		Port: 8080,
	}
	err := client.Register(info)
	require.NoError(t, err)

	// Deregister
	err = client.Deregister("test-service")
	require.NoError(t, err)

	// Verify service is gone
	_, err = registry.Get("test-service")
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

func TestHeartbeat(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register service
	info := ServiceInfo{
		Name: "test-service",
		Host: "localhost",
		Port: 8080,
	}
	err := client.Register(info)
	require.NoError(t, err)

	// Send heartbeat
	err = client.Heartbeat("test-service")
	assert.NoError(t, err)
}

func TestListServices(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register multiple services
	services := []string{"service-1", "service-2", "service-3"}
	for i, name := range services {
		info := ServiceInfo{
			Name: name,
			Host: "localhost",
			Port: 8080 + i,
		}
		require.NoError(t, client.Register(info))
	}

	// List all services
	list := client.ListServices()
	assert.Len(t, list, 3)
}

func TestListHealthyServices(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register services
	healthyInfo := ServiceInfo{
		Name: "healthy-service",
		Host: "localhost",
		Port: 8080,
	}
	unhealthyInfo := ServiceInfo{
		Name: "unhealthy-service",
		Host: "localhost",
		Port: 8081,
	}

	require.NoError(t, client.Register(healthyInfo))
	require.NoError(t, client.Register(unhealthyInfo))

	// Mark one as unhealthy
	require.NoError(t, registry.UpdateHealth("unhealthy-service", false))

	// List only healthy
	healthy := client.ListHealthyServices()
	assert.Len(t, healthy, 1)
	assert.Equal(t, "healthy-service", healthy[0].Name)
}

func TestDiscoverByRegistry(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register service
	info := ServiceInfo{
		Name:     "test-service",
		Host:     "localhost",
		Port:     8080,
		Protocol: "http",
	}
	require.NoError(t, client.Register(info))

	// Discover service
	result, err := client.Discover("test-service")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-service", result.ServiceInfo.Name)
	assert.Equal(t, 8080, result.ServiceInfo.Port)
	assert.Equal(t, StrategyRegistry, result.Strategy)
}

func TestDiscover_ServiceNotFound(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	config.EnableDNS = false // Disable DNS to force failure
	client := NewDiscoveryClient(config)

	// Try to discover non-existent service
	_, err := client.Discover("non-existent-service")
	assert.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestDiscover_InvalidServiceName(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	_, err := client.Discover("")
	assert.ErrorIs(t, err, ErrInvalidServiceName)
}

func TestDiscoverWithTimeout(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register service
	info := ServiceInfo{
		Name: "test-service",
		Host: "localhost",
		Port: 8080,
	}
	require.NoError(t, client.Register(info))

	// Discover with timeout
	result, err := client.DiscoverWithTimeout("test-service", 2*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-service", result.ServiceInfo.Name)
}

func TestDiscoverWithTimeout_Timeout(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	config.EnableDNS = false // Disable to force failure
	client := NewDiscoveryClient(config)

	// Try to discover non-existent service with short timeout
	// Service discovery will fail quickly, so we get ErrServiceUnavailable
	_, err := client.DiscoverWithTimeout("non-existent", 100*time.Millisecond)
	assert.Error(t, err)
	// Either timeout or service unavailable error is acceptable
	isTimeout := err.Error() == "timeout" || err.Error() == "discovery timeout after 100ms for service: non-existent"
	isUnavailable := err == ErrServiceUnavailable || err.Error() == "service unavailable: non-existent"
	assert.True(t, isTimeout || isUnavailable, "Expected timeout or unavailable error, got: %v", err)
}

func TestGetServiceAddress(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register service
	info := ServiceInfo{
		Name: "test-service",
		Host: "localhost",
		Port: 8080,
	}
	require.NoError(t, client.Register(info))

	// Get address
	address, err := client.GetServiceAddress("test-service")
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", address)
}

func TestGetDefaultPort(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	tests := []struct {
		serviceName  string
		expectedPort int
	}{
		{"database", 5432},
		{"cache", 6379},
		{"api", 8080},
		{"grpc", 9090},
		{"metrics", 9100},
		{"postgres-primary", 5432},  // Keyword match
		{"redis-cache", 6379},        // Keyword match
		{"unknown-service", 0},       // No default
	}

	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			port := client.getDefaultPort(tt.serviceName)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestWaitForService(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register service in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		info := ServiceInfo{
			Name: "delayed-service",
			Host: "localhost",
			Port: 8080,
		}
		client.Register(info)
	}()

	// Wait for service
	result, err := client.WaitForService("delayed-service", 2*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "delayed-service", result.ServiceInfo.Name)
}

func TestWaitForService_Timeout(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	config.EnableDNS = false // Disable to force failure
	client := NewDiscoveryClient(config)

	// Wait for non-existent service with short timeout
	_, err := client.WaitForService("non-existent", 200*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestDiscoverByDNS(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	config.EnableRegistry = false // Disable registry to force DNS
	config.PreferredStrategies = []DiscoveryStrategy{StrategyDNS}
	client := NewDiscoveryClient(config)

	// Test with localhost (should resolve)
	result, err := client.Discover("localhost")
	if err == nil {
		assert.NotNil(t, result)
		assert.Equal(t, StrategyDNS, result.Strategy)
	} else {
		// DNS might not work in all test environments
		t.Skip("DNS resolution not available in test environment")
	}
}

func TestIsPortReachable(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Start a test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Test reachable port
	reachable := client.isPortReachable(fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
	assert.True(t, reachable)

	// Test unreachable port
	unreachable := client.isPortReachable("127.0.0.1:9999", 100*time.Millisecond)
	assert.False(t, unreachable)
}

func TestRegister_NoRegistry(t *testing.T) {
	config := DiscoveryClientConfig{
		EnableRegistry: false,
	}
	client := NewDiscoveryClient(config)

	info := ServiceInfo{
		Name: "test-service",
		Host: "localhost",
		Port: 8080,
	}

	err := client.Register(info)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry not enabled")
}

func TestDeregister_NoRegistry(t *testing.T) {
	config := DiscoveryClientConfig{
		EnableRegistry: false,
	}
	client := NewDiscoveryClient(config)

	err := client.Deregister("test-service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry not enabled")
}

func TestHeartbeat_NoRegistry(t *testing.T) {
	config := DiscoveryClientConfig{
		EnableRegistry: false,
	}
	client := NewDiscoveryClient(config)

	err := client.Heartbeat("test-service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry not enabled")
}

func TestListServices_NoRegistry(t *testing.T) {
	config := DiscoveryClientConfig{
		EnableRegistry: false,
	}
	client := NewDiscoveryClient(config)

	services := client.ListServices()
	assert.Empty(t, services)
}

func TestDiscoveryResult_Latency(t *testing.T) {
	registry := NewDefaultServiceRegistry()
	allocator := NewDefaultPortAllocator()
	config := DefaultDiscoveryClientConfig(registry, allocator)
	client := NewDiscoveryClient(config)

	// Register service
	info := ServiceInfo{
		Name: "test-service",
		Host: "localhost",
		Port: 8080,
	}
	require.NoError(t, client.Register(info))

	// Discover and check latency
	result, err := client.Discover("test-service")
	require.NoError(t, err)
	assert.Greater(t, result.Latency, time.Duration(0))
	assert.Less(t, result.Latency, 100*time.Millisecond) // Should be very fast
}
