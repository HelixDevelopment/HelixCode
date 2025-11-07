package discovery

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthMonitor(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()

	hm := NewHealthMonitor(config, registry)

	assert.NotNil(t, hm)
	assert.False(t, hm.IsRunning())
	assert.Equal(t, config.CheckInterval, hm.config.CheckInterval)
}

func TestDefaultHealthMonitorConfig(t *testing.T) {
	config := DefaultHealthMonitorConfig()

	assert.Equal(t, 5*time.Second, config.CheckInterval)
	assert.Equal(t, 2*time.Second, config.CheckTimeout)
	assert.Equal(t, 3, config.UnhealthyThreshold)
	assert.Equal(t, 2, config.HealthyThreshold)
	assert.Equal(t, HealthCheckTCP, config.DefaultStrategy)
	assert.True(t, config.EnableAutoRemoval)
	assert.Equal(t, 5, config.RemovalThreshold)
}

func TestHealthMonitor_StartStop(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Start
	err := hm.Start()
	require.NoError(t, err)
	assert.True(t, hm.IsRunning())

	// Wait a moment
	time.Sleep(100 * time.Millisecond)

	// Stop
	err = hm.Stop()
	require.NoError(t, err)
	assert.False(t, hm.IsRunning())
}

func TestHealthMonitor_StartAlreadyRunning(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	err := hm.Start()
	require.NoError(t, err)
	defer hm.Stop()

	err = hm.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestHealthMonitor_StopNotRunning(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	err := hm.Stop()
	assert.ErrorIs(t, err, ErrHealthMonitorNotRunning)
}

func TestHealthMonitor_CheckServiceHealth_TCP(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Start a test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Register service
	info := ServiceInfo{
		Name: "test-service",
		Host: "127.0.0.1",
		Port: port,
	}
	err = registry.Register(info)
	require.NoError(t, err)

	// Check health
	result, err := hm.CheckServiceHealth("test-service")
	require.NoError(t, err)
	assert.True(t, result.Healthy)
	assert.Equal(t, "test-service", result.ServiceName)
}

func TestHealthMonitor_CheckServiceHealth_Unhealthy(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Register service on unreachable port
	info := ServiceInfo{
		Name: "unreachable-service",
		Host: "127.0.0.1",
		Port: 19999, // Unlikely to be in use
	}
	err := registry.Register(info)
	require.NoError(t, err)

	// Check health
	result, err := hm.CheckServiceHealth("unreachable-service")
	require.NoError(t, err)
	assert.False(t, result.Healthy)
	assert.NotNil(t, result.Error)
}

func TestHealthMonitor_RegisterCustomCheck(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Register custom check
	customCheckCalled := false
	hm.RegisterCustomCheck("custom-service", func(info *ServiceInfo) error {
		customCheckCalled = true
		return nil
	})

	// Register service
	info := ServiceInfo{
		Name: "custom-service",
		Host: "localhost",
		Port: 8080,
	}
	err := registry.Register(info)
	require.NoError(t, err)

	// Check health
	result, err := hm.CheckServiceHealth("custom-service")
	require.NoError(t, err)
	assert.True(t, result.Healthy)
	assert.True(t, customCheckCalled)
}

func TestHealthMonitor_CustomCheckFailure(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Register custom check that fails
	testErr := errors.New("custom check failed")
	hm.RegisterCustomCheck("failing-service", func(info *ServiceInfo) error {
		return testErr
	})

	// Register service
	info := ServiceInfo{
		Name: "failing-service",
		Host: "localhost",
		Port: 8080,
	}
	err := registry.Register(info)
	require.NoError(t, err)

	// Check health
	result, err := hm.CheckServiceHealth("failing-service")
	require.NoError(t, err)
	assert.False(t, result.Healthy)
	assert.Equal(t, testErr, result.Error)
}

func TestHealthMonitor_SetServiceStrategy(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	hm.SetServiceStrategy("test-service", HealthCheckHTTP)

	strategy := hm.getStrategy("test-service")
	assert.Equal(t, HealthCheckHTTP, strategy)
}

func TestHealthMonitor_GetLastResult(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Initially no result
	_, exists := hm.GetLastResult("test-service")
	assert.False(t, exists)

	// Store a result
	result := &HealthCheckResult{
		ServiceName: "test-service",
		Healthy:     true,
		Timestamp:   time.Now(),
		Latency:     10 * time.Millisecond,
	}

	hm.mu.Lock()
	hm.lastResults["test-service"] = result
	hm.mu.Unlock()

	// Retrieve result
	retrieved, exists := hm.GetLastResult("test-service")
	assert.True(t, exists)
	assert.Equal(t, "test-service", retrieved.ServiceName)
	assert.True(t, retrieved.Healthy)
}

func TestHealthMonitor_GetAllResults(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Store multiple results
	results := []*HealthCheckResult{
		{ServiceName: "service-1", Healthy: true},
		{ServiceName: "service-2", Healthy: false},
		{ServiceName: "service-3", Healthy: true},
	}

	hm.mu.Lock()
	for _, r := range results {
		hm.lastResults[r.ServiceName] = r
	}
	hm.mu.Unlock()

	// Retrieve all
	allResults := hm.GetAllResults()
	assert.Len(t, allResults, 3)
	assert.Contains(t, allResults, "service-1")
	assert.Contains(t, allResults, "service-2")
	assert.Contains(t, allResults, "service-3")
}

func TestHealthMonitor_FailureAndSuccessCounts(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	serviceName := "test-service"

	// Initially zero
	assert.Equal(t, 0, hm.GetFailureCount(serviceName))
	assert.Equal(t, 0, hm.GetSuccessCount(serviceName))

	// Set counts
	hm.mu.Lock()
	hm.failureCounts[serviceName] = 3
	hm.successCounts[serviceName] = 2
	hm.mu.Unlock()

	assert.Equal(t, 3, hm.GetFailureCount(serviceName))
	assert.Equal(t, 2, hm.GetSuccessCount(serviceName))
}

func TestHealthMonitor_ResetCounts(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	serviceName := "test-service"

	// Set counts
	hm.mu.Lock()
	hm.failureCounts[serviceName] = 5
	hm.successCounts[serviceName] = 3
	hm.mu.Unlock()

	// Reset
	hm.ResetCounts(serviceName)

	assert.Equal(t, 0, hm.GetFailureCount(serviceName))
	assert.Equal(t, 0, hm.GetSuccessCount(serviceName))
}

func TestHealthMonitor_ProcessResult_Success(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.HealthyThreshold = 2
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Register service
	info := ServiceInfo{
		Name: "test-service",
		Host: "localhost",
		Port: 8080,
	}
	err := registry.Register(info)
	require.NoError(t, err)

	// Process successful results
	for i := 0; i < 2; i++ {
		result := &HealthCheckResult{
			ServiceName: "test-service",
			Healthy:     true,
		}
		hm.processResult(result)
	}

	// Service should be marked healthy
	service, err := registry.Get("test-service")
	require.NoError(t, err)
	assert.True(t, service.Healthy)
}

func TestHealthMonitor_ProcessResult_Failure(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.UnhealthyThreshold = 2
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Register service
	info := ServiceInfo{
		Name: "test-service",
		Host: "localhost",
		Port: 8080,
	}
	err := registry.Register(info)
	require.NoError(t, err)

	// Process failed results
	for i := 0; i < 2; i++ {
		result := &HealthCheckResult{
			ServiceName: "test-service",
			Healthy:     false,
			Error:       errors.New("check failed"),
		}
		hm.processResult(result)
	}

	// Service should be marked unhealthy
	service, err := registry.Get("test-service")
	require.NoError(t, err)
	assert.False(t, service.Healthy)
}

func TestHealthMonitor_AutoRemoval(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.EnableAutoRemoval = true
	config.RemovalThreshold = 3
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Register service
	info := ServiceInfo{
		Name: "failing-service",
		Host: "localhost",
		Port: 8080,
	}
	err := registry.Register(info)
	require.NoError(t, err)

	// Process enough failures to trigger removal
	for i := 0; i < 3; i++ {
		result := &HealthCheckResult{
			ServiceName: "failing-service",
			Healthy:     false,
			Error:       errors.New("check failed"),
		}
		hm.processResult(result)
	}

	// Service should be removed
	_, err = registry.Get("failing-service")
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

func TestHealthMonitor_GetHealthyServices(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

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

	require.NoError(t, registry.Register(healthyInfo))
	require.NoError(t, registry.Register(unhealthyInfo))
	require.NoError(t, registry.UpdateHealth("unhealthy-service", false))

	// Get healthy services
	healthy := hm.GetHealthyServices()
	assert.Len(t, healthy, 1)
	assert.Equal(t, "healthy-service", healthy[0].Name)
}

func TestHealthMonitor_GetUnhealthyServices(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

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

	require.NoError(t, registry.Register(healthyInfo))
	require.NoError(t, registry.Register(unhealthyInfo))
	require.NoError(t, registry.UpdateHealth("unhealthy-service", false))

	// Get unhealthy services
	unhealthy := hm.GetUnhealthyServices()
	assert.Len(t, unhealthy, 1)
	assert.Equal(t, "unhealthy-service", unhealthy[0].Name)
}

func TestHealthMonitor_ConcurrentAccess(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 50 * time.Millisecond
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Register service with listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	info := ServiceInfo{
		Name: "concurrent-service",
		Host: "127.0.0.1",
		Port: port,
	}
	err = registry.Register(info)
	require.NoError(t, err)

	// Start monitor
	err = hm.Start()
	require.NoError(t, err)
	defer hm.Stop()

	done := make(chan bool)

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				hm.GetLastResult("concurrent-service")
				hm.GetAllResults()
				hm.GetFailureCount("concurrent-service")
				hm.GetSuccessCount("concurrent-service")
				hm.GetHealthyServices()
				time.Sleep(1 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or deadlock
	assert.True(t, hm.IsRunning())
}

func TestHealthMonitor_MonitorLoop(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 100 * time.Millisecond
	config.HealthyThreshold = 2
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Start test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Register service
	info := ServiceInfo{
		Name: "monitored-service",
		Host: "127.0.0.1",
		Port: port,
	}
	err = registry.Register(info)
	require.NoError(t, err)

	// Start monitoring
	err = hm.Start()
	require.NoError(t, err)

	// Wait for checks to run
	time.Sleep(300 * time.Millisecond)

	// Stop monitoring
	err = hm.Stop()
	require.NoError(t, err)

	// Check that results were recorded
	result, exists := hm.GetLastResult("monitored-service")
	assert.True(t, exists)
	assert.NotNil(t, result)

	// Service should be healthy
	service, err := registry.Get("monitored-service")
	require.NoError(t, err)
	assert.True(t, service.Healthy)
}

func TestHealthMonitor_MultipleServices(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	registry := NewDefaultServiceRegistry()
	hm := NewHealthMonitor(config, registry)

	// Start multiple test servers
	listeners := make([]net.Listener, 3)
	for i := 0; i < 3; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()
		listeners[i] = listener

		port := listener.Addr().(*net.TCPAddr).Port
		info := ServiceInfo{
			Name: fmt.Sprintf("service-%d", i),
			Host: "127.0.0.1",
			Port: port,
		}
		err = registry.Register(info)
		require.NoError(t, err)
	}

	// Check all services and manually store results
	for i := 0; i < 3; i++ {
		result, err := hm.CheckServiceHealth(fmt.Sprintf("service-%d", i))
		require.NoError(t, err)
		assert.True(t, result.Healthy)

		// Manually store result (since CheckServiceHealth doesn't auto-store)
		hm.mu.Lock()
		hm.lastResults[result.ServiceName] = result
		hm.mu.Unlock()
	}

	// All results should be stored
	allResults := hm.GetAllResults()
	assert.Len(t, allResults, 3)
}
