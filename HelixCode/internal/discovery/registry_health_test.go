package discovery

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckServiceHealth_Dispatcher tests the health check dispatcher logic
func TestCheckServiceHealth_Dispatcher(t *testing.T) {
	registry := NewDefaultServiceRegistry()

	tests := []struct {
		name     string
		service  *ServiceInfo
		expected string // expected protocol to be checked
	}{
		{
			name: "HTTP protocol",
			service: &ServiceInfo{
				Name:     "http-service",
				Host:     "localhost",
				Port:     8080,
				Protocol: "http",
				Healthy:  true,
			},
			expected: "http",
		},
		{
			name: "HTTPS protocol",
			service: &ServiceInfo{
				Name:     "https-service",
				Host:     "localhost",
				Port:     8443,
				Protocol: "https",
				Healthy:  true,
			},
			expected: "https",
		},
		{
			name: "TCP protocol",
			service: &ServiceInfo{
				Name:     "tcp-service",
				Host:     "localhost",
				Port:     9000,
				Protocol: "tcp",
				Healthy:  true,
			},
			expected: "tcp",
		},
		{
			name: "unknown protocol falls back to current health",
			service: &ServiceInfo{
				Name:     "unknown-service",
				Host:     "localhost",
				Port:     10000,
				Protocol: "unknown",
				Healthy:  true,
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For unknown protocols, it should return the current healthy status
			if tt.expected == "unknown" {
				result := registry.checkServiceHealth(tt.service)
				assert.True(t, result, "unknown protocol should maintain current health status")
			}
		})
	}
}

// TestCheckHTTPHealth tests HTTP health checking
func TestCheckHTTPHealth(t *testing.T) {
	registry := NewDefaultServiceRegistry()

	tests := []struct {
		name            string
		serverResponse  int
		expectedHealthy bool
	}{
		{
			name:            "200 OK is healthy",
			serverResponse:  http.StatusOK,
			expectedHealthy: true,
		},
		{
			name:            "201 Created is healthy",
			serverResponse:  http.StatusCreated,
			expectedHealthy: true,
		},
		{
			name:            "204 No Content is healthy",
			serverResponse:  http.StatusNoContent,
			expectedHealthy: true,
		},
		{
			name:            "301 Redirect is healthy",
			serverResponse:  http.StatusMovedPermanently,
			expectedHealthy: true,
		},
		{
			name:            "400 Bad Request is unhealthy",
			serverResponse:  http.StatusBadRequest,
			expectedHealthy: false,
		},
		{
			name:            "404 Not Found is unhealthy",
			serverResponse:  http.StatusNotFound,
			expectedHealthy: false,
		},
		{
			name:            "500 Internal Server Error is unhealthy",
			serverResponse:  http.StatusInternalServerError,
			expectedHealthy: false,
		},
		{
			name:            "503 Service Unavailable is unhealthy",
			serverResponse:  http.StatusServiceUnavailable,
			expectedHealthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/health", r.URL.Path, "should call default health endpoint")
				w.WriteHeader(tt.serverResponse)
			}))
			defer server.Close()

			// Parse server URL to get host and port
			host := server.Listener.Addr().(*net.TCPAddr).IP.String()
			port := server.Listener.Addr().(*net.TCPAddr).Port

			service := &ServiceInfo{
				Name:     "test-service",
				Host:     host,
				Port:     port,
				Protocol: "http",
				Metadata: make(map[string]string),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			healthy := registry.checkHTTPHealth(ctx, service)
			assert.Equal(t, tt.expectedHealthy, healthy)
		})
	}
}

// TestCheckHTTPHealth_CustomEndpoint tests custom health endpoint configuration
func TestCheckHTTPHealth_CustomEndpoint(t *testing.T) {
	registry := NewDefaultServiceRegistry()

	// Create a test server that only responds to custom endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/healthz" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host := server.Listener.Addr().(*net.TCPAddr).IP.String()
	port := server.Listener.Addr().(*net.TCPAddr).Port

	service := &ServiceInfo{
		Name:     "test-service",
		Host:     host,
		Port:     port,
		Protocol: "http",
		Metadata: map[string]string{
			"health_endpoint": "/api/healthz",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthy := registry.checkHTTPHealth(ctx, service)
	assert.True(t, healthy, "should use custom health endpoint from metadata")
}

// TestCheckHTTPHealth_Timeout tests HTTP health check timeout
func TestCheckHTTPHealth_Timeout(t *testing.T) {
	registry := NewDefaultServiceRegistry()

	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than our timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.Listener.Addr().(*net.TCPAddr).IP.String()
	port := server.Listener.Addr().(*net.TCPAddr).Port

	service := &ServiceInfo{
		Name:     "slow-service",
		Host:     host,
		Port:     port,
		Protocol: "http",
		Metadata: make(map[string]string),
	}

	// Use short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	healthy := registry.checkHTTPHealth(ctx, service)
	assert.False(t, healthy, "should fail on timeout")
}

// TestCheckHTTPHealth_UnreachableHost tests health check for unreachable service
func TestCheckHTTPHealth_UnreachableHost(t *testing.T) {
	registry := NewDefaultServiceRegistry()

	service := &ServiceInfo{
		Name:     "unreachable-service",
		Host:     "127.0.0.1",
		Port:     99999, // Invalid port
		Protocol: "http",
		Metadata: make(map[string]string),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	healthy := registry.checkHTTPHealth(ctx, service)
	assert.False(t, healthy, "should fail for unreachable host")
}

// TestCheckTCPHealth tests TCP connection health checking
func TestCheckTCPHealth(t *testing.T) {
	registry := NewDefaultServiceRegistry()

	// Create a TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	t.Run("successful TCP connection", func(t *testing.T) {
		service := &ServiceInfo{
			Name:     "tcp-service",
			Host:     "127.0.0.1",
			Port:     port,
			Protocol: "tcp",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		healthy := registry.checkTCPHealth(ctx, service)
		assert.True(t, healthy, "should successfully connect to TCP service")
	})

	t.Run("failed TCP connection", func(t *testing.T) {
		service := &ServiceInfo{
			Name:     "tcp-service-down",
			Host:     "127.0.0.1",
			Port:     99999, // Non-existent port
			Protocol: "tcp",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		healthy := registry.checkTCPHealth(ctx, service)
		assert.False(t, healthy, "should fail to connect to non-existent port")
	})
}

// TestPerformHealthChecks_Integration tests the integrated health checking process
func TestPerformHealthChecks_Integration(t *testing.T) {
	// Create registry with disabled background health checks
	config := RegistryConfig{
		DefaultTTL:             30 * time.Second,
		CleanupInterval:        10 * time.Second,
		EnableHealthChecks:     false, // Disable automatic checks
		HealthCheckInterval:    15 * time.Second,
	}
	registry := NewServiceRegistry(config)

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.Listener.Addr().(*net.TCPAddr).IP.String()
	port := server.Listener.Addr().(*net.TCPAddr).Port

	// Register an HTTP service
	httpService := ServiceInfo{
		Name:          "http-service",
		Host:          host,
		Port:          port,
		Protocol:      "http",
		TTL:           30 * time.Second,
		LastHeartbeat: time.Now(),
		Healthy:       false, // Start as unhealthy
		Metadata:      make(map[string]string),
	}

	err := registry.Register(httpService)
	require.NoError(t, err)

	// Create TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	tcpPort := listener.Addr().(*net.TCPAddr).Port

	// Register a TCP service
	tcpService := ServiceInfo{
		Name:          "tcp-service",
		Host:          "127.0.0.1",
		Port:          tcpPort,
		Protocol:      "tcp",
		TTL:           30 * time.Second,
		LastHeartbeat: time.Now(),
		Healthy:       false, // Start as unhealthy
	}

	err = registry.Register(tcpService)
	require.NoError(t, err)

	// Manually trigger health checks
	registry.performHealthChecks()

	// Verify HTTP service is now healthy
	retrieved, err := registry.Get("http-service")
	require.NoError(t, err)
	assert.True(t, retrieved.Healthy, "HTTP service should be healthy after check")

	// Verify TCP service is now healthy
	retrieved, err = registry.Get("tcp-service")
	require.NoError(t, err)
	assert.True(t, retrieved.Healthy, "TCP service should be healthy after check")
}

// TestPerformHealthChecks_HeartbeatCheck tests heartbeat-based health detection
func TestPerformHealthChecks_HeartbeatCheck(t *testing.T) {
	config := RegistryConfig{
		DefaultTTL:             10 * time.Second,
		CleanupInterval:        10 * time.Second,
		EnableHealthChecks:     false,
		HealthCheckInterval:    15 * time.Second,
	}
	registry := NewServiceRegistry(config)

	// Register a service with old heartbeat
	service := ServiceInfo{
		Name:          "stale-service",
		Host:          "localhost",
		Port:          8080,
		Protocol:      "http",
		TTL:           10 * time.Second,
		LastHeartbeat: time.Now().Add(-6 * time.Second), // More than TTL/2
		Healthy:       true,
	}

	err := registry.Register(service)
	require.NoError(t, err)

	// Perform health checks
	registry.performHealthChecks()

	// Service should be marked unhealthy due to stale heartbeat
	retrieved, err := registry.Get("stale-service")
	require.NoError(t, err)
	assert.False(t, retrieved.Healthy, "service with stale heartbeat should be marked unhealthy")
}

// TestPerformHealthChecks_UnknownProtocol tests handling of unknown protocols
func TestPerformHealthChecks_UnknownProtocol(t *testing.T) {
	registry := NewDefaultServiceRegistry()

	// Register service with unknown protocol
	service := ServiceInfo{
		Name:          "custom-protocol-service",
		Host:          "localhost",
		Port:          8080,
		Protocol:      "custom",
		TTL:           30 * time.Second,
		LastHeartbeat: time.Now(),
		Healthy:       true, // Start as healthy
	}

	err := registry.Register(service)
	require.NoError(t, err)

	// Perform health checks
	registry.performHealthChecks()

	// Service should maintain its healthy status
	retrieved, err := registry.Get("custom-protocol-service")
	require.NoError(t, err)
	assert.True(t, retrieved.Healthy, "unknown protocol should maintain current health status")
}

// BenchmarkHTTPHealthCheck benchmarks HTTP health check performance
func BenchmarkHTTPHealthCheck(b *testing.B) {
	registry := NewDefaultServiceRegistry()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.Listener.Addr().(*net.TCPAddr).IP.String()
	port := server.Listener.Addr().(*net.TCPAddr).Port

	service := &ServiceInfo{
		Name:     "bench-service",
		Host:     host,
		Port:     port,
		Protocol: "http",
		Metadata: make(map[string]string),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.checkHTTPHealth(ctx, service)
	}
}

// BenchmarkTCPHealthCheck benchmarks TCP health check performance
func BenchmarkTCPHealthCheck(b *testing.B) {
	registry := NewDefaultServiceRegistry()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	service := &ServiceInfo{
		Name:     "bench-tcp-service",
		Host:     "127.0.0.1",
		Port:     port,
		Protocol: "tcp",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.checkTCPHealth(ctx, service)
	}
}

// Example demonstrates protocol-specific health checking
func ExampleServiceRegistry_checkServiceHealth() {
	registry := NewDefaultServiceRegistry()

	// HTTP service
	httpService := &ServiceInfo{
		Name:     "api-server",
		Host:     "localhost",
		Port:     8080,
		Protocol: "http",
		Metadata: map[string]string{
			"health_endpoint": "/api/health",
		},
	}

	healthy := registry.checkServiceHealth(httpService)
	fmt.Printf("HTTP service healthy: %v\n", healthy)

	// TCP service
	tcpService := &ServiceInfo{
		Name:     "database",
		Host:     "localhost",
		Port:     5432,
		Protocol: "tcp",
	}

	healthy = registry.checkServiceHealth(tcpService)
	fmt.Printf("TCP service healthy: %v\n", healthy)
}
