package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	// ErrHealthCheckFailed is returned when a health check fails
	ErrHealthCheckFailed = errors.New("health check failed")

	// ErrHealthMonitorNotRunning is returned when monitor is not running
	ErrHealthMonitorNotRunning = errors.New("health monitor not running")
)

// HealthCheckStrategy defines the strategy for health checking
type HealthCheckStrategy string

const (
	// HealthCheckTCP checks if TCP port is reachable
	HealthCheckTCP HealthCheckStrategy = "tcp"

	// HealthCheckHTTP performs HTTP GET request
	HealthCheckHTTP HealthCheckStrategy = "http"

	// HealthCheckCustom uses custom health check function
	HealthCheckCustom HealthCheckStrategy = "custom"
)

// HealthCheckFunc is a custom health check function
type HealthCheckFunc func(info *ServiceInfo) error

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	ServiceName string
	Healthy     bool
	Timestamp   time.Time
	Latency     time.Duration
	Error       error
}

// HealthMonitorConfig configures the health monitor
type HealthMonitorConfig struct {
	// CheckInterval is how often to check service health
	CheckInterval time.Duration

	// CheckTimeout is the timeout for each health check
	CheckTimeout time.Duration

	// UnhealthyThreshold is how many failed checks before marking unhealthy
	UnhealthyThreshold int

	// HealthyThreshold is how many successful checks before marking healthy
	HealthyThreshold int

	// DefaultStrategy is the default health check strategy
	DefaultStrategy HealthCheckStrategy

	// EnableAutoRemoval removes unhealthy services after threshold
	EnableAutoRemoval bool

	// RemovalThreshold is how many consecutive failures before removal
	RemovalThreshold int
}

// DefaultHealthMonitorConfig returns default configuration
func DefaultHealthMonitorConfig() HealthMonitorConfig {
	return HealthMonitorConfig{
		CheckInterval:      5 * time.Second,
		CheckTimeout:       2 * time.Second,
		UnhealthyThreshold: 3,
		HealthyThreshold:   2,
		DefaultStrategy:    HealthCheckTCP,
		EnableAutoRemoval:  true,
		RemovalThreshold:   5,
	}
}

// HealthMonitor monitors service health
type HealthMonitor struct {
	config   HealthMonitorConfig
	registry *ServiceRegistry
	mu       sync.RWMutex

	// Health check state
	failureCounts   map[string]int
	successCounts   map[string]int
	lastResults     map[string]*HealthCheckResult
	customChecks    map[string]HealthCheckFunc
	serviceStrategy map[string]HealthCheckStrategy

	// Control
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(config HealthMonitorConfig, registry *ServiceRegistry) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &HealthMonitor{
		config:          config,
		registry:        registry,
		failureCounts:   make(map[string]int),
		successCounts:   make(map[string]int),
		lastResults:     make(map[string]*HealthCheckResult),
		customChecks:    make(map[string]HealthCheckFunc),
		serviceStrategy: make(map[string]HealthCheckStrategy),
		stopChan:        make(chan struct{}),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start starts the health monitor
func (hm *HealthMonitor) Start() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.running {
		return errors.New("health monitor already running")
	}

	if hm.registry == nil {
		return errors.New("registry not configured")
	}

	hm.running = true

	// Start monitoring goroutine
	hm.wg.Add(1)
	go hm.monitorLoop()

	return nil
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() error {
	hm.mu.Lock()
	if !hm.running {
		hm.mu.Unlock()
		return ErrHealthMonitorNotRunning
	}

	hm.running = false
	hm.mu.Unlock()

	// Signal stop
	hm.cancel()
	close(hm.stopChan)

	// Wait for goroutines
	hm.wg.Wait()

	return nil
}

// IsRunning returns whether the monitor is running
func (hm *HealthMonitor) IsRunning() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.running
}

// RegisterCustomCheck registers a custom health check for a service
func (hm *HealthMonitor) RegisterCustomCheck(serviceName string, checkFn HealthCheckFunc) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.customChecks[serviceName] = checkFn
	hm.serviceStrategy[serviceName] = HealthCheckCustom
}

// SetServiceStrategy sets the health check strategy for a service
func (hm *HealthMonitor) SetServiceStrategy(serviceName string, strategy HealthCheckStrategy) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.serviceStrategy[serviceName] = strategy
}

// GetLastResult returns the last health check result for a service
func (hm *HealthMonitor) GetLastResult(serviceName string) (*HealthCheckResult, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result, exists := hm.lastResults[serviceName]
	if !exists {
		return nil, false
	}

	// Return a copy
	resultCopy := *result
	return &resultCopy, true
}

// GetAllResults returns all health check results
func (hm *HealthMonitor) GetAllResults() map[string]*HealthCheckResult {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results := make(map[string]*HealthCheckResult, len(hm.lastResults))
	for name, result := range hm.lastResults {
		resultCopy := *result
		results[name] = &resultCopy
	}

	return results
}

// CheckServiceHealth performs an immediate health check on a service
func (hm *HealthMonitor) CheckServiceHealth(serviceName string) (*HealthCheckResult, error) {
	// Get service from registry
	serviceInfo, err := hm.registry.Get(serviceName)
	if err != nil {
		return nil, err
	}

	return hm.checkService(serviceInfo)
}

// Internal methods

func (hm *HealthMonitor) monitorLoop() {
	defer hm.wg.Done()

	ticker := time.NewTicker(hm.config.CheckInterval)
	defer ticker.Stop()

	// Perform initial check
	hm.checkAllServices()

	for {
		select {
		case <-ticker.C:
			hm.checkAllServices()

		case <-hm.stopChan:
			return

		case <-hm.ctx.Done():
			return
		}
	}
}

func (hm *HealthMonitor) checkAllServices() {
	services := hm.registry.List()

	for _, serviceInfo := range services {
		result, err := hm.checkService(serviceInfo)
		if err != nil {
			continue
		}

		hm.processResult(result)
	}
}

func (hm *HealthMonitor) checkService(serviceInfo *ServiceInfo) (*HealthCheckResult, error) {
	startTime := time.Now()

	// Determine strategy
	strategy := hm.getStrategy(serviceInfo.Name)

	var err error
	switch strategy {
	case HealthCheckTCP:
		err = hm.checkTCP(serviceInfo)
	case HealthCheckHTTP:
		err = hm.checkHTTP(serviceInfo)
	case HealthCheckCustom:
		err = hm.checkCustom(serviceInfo)
	default:
		err = hm.checkTCP(serviceInfo)
	}

	latency := time.Since(startTime)

	result := &HealthCheckResult{
		ServiceName: serviceInfo.Name,
		Healthy:     err == nil,
		Timestamp:   time.Now(),
		Latency:     latency,
		Error:       err,
	}

	return result, nil
}

func (hm *HealthMonitor) getStrategy(serviceName string) HealthCheckStrategy {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if strategy, exists := hm.serviceStrategy[serviceName]; exists {
		return strategy
	}

	return hm.config.DefaultStrategy
}

func (hm *HealthMonitor) checkTCP(serviceInfo *ServiceInfo) error {
	address := fmt.Sprintf("%s:%d", serviceInfo.Host, serviceInfo.Port)

	conn, err := net.DialTimeout("tcp", address, hm.config.CheckTimeout)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHealthCheckFailed, err)
	}
	conn.Close()

	return nil
}

func (hm *HealthMonitor) checkHTTP(serviceInfo *ServiceInfo) error {
	url := fmt.Sprintf("http://%s:%d/health", serviceInfo.Host, serviceInfo.Port)

	client := &http.Client{
		Timeout: hm.config.CheckTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHealthCheckFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrHealthCheckFailed, resp.StatusCode)
	}

	return nil
}

func (hm *HealthMonitor) checkCustom(serviceInfo *ServiceInfo) error {
	hm.mu.RLock()
	checkFn, exists := hm.customChecks[serviceInfo.Name]
	hm.mu.RUnlock()

	if !exists {
		return errors.New("no custom health check registered")
	}

	return checkFn(serviceInfo)
}

func (hm *HealthMonitor) processResult(result *HealthCheckResult) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Store result
	hm.lastResults[result.ServiceName] = result

	if result.Healthy {
		// Increment success count
		hm.successCounts[result.ServiceName]++
		hm.failureCounts[result.ServiceName] = 0

		// Check if we should mark as healthy
		if hm.successCounts[result.ServiceName] >= hm.config.HealthyThreshold {
			hm.registry.UpdateHealth(result.ServiceName, true)
		}
	} else {
		// Increment failure count
		hm.failureCounts[result.ServiceName]++
		hm.successCounts[result.ServiceName] = 0

		// Check if we should mark as unhealthy
		if hm.failureCounts[result.ServiceName] >= hm.config.UnhealthyThreshold {
			hm.registry.UpdateHealth(result.ServiceName, false)
		}

		// Check if we should remove the service
		if hm.config.EnableAutoRemoval &&
			hm.failureCounts[result.ServiceName] >= hm.config.RemovalThreshold {
			hm.registry.Deregister(result.ServiceName)
			delete(hm.failureCounts, result.ServiceName)
			delete(hm.successCounts, result.ServiceName)
			delete(hm.lastResults, result.ServiceName)
		}
	}
}

// GetFailureCount returns the current failure count for a service
func (hm *HealthMonitor) GetFailureCount(serviceName string) int {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.failureCounts[serviceName]
}

// GetSuccessCount returns the current success count for a service
func (hm *HealthMonitor) GetSuccessCount(serviceName string) int {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.successCounts[serviceName]
}

// ResetCounts resets the failure and success counts for a service
func (hm *HealthMonitor) ResetCounts(serviceName string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.failureCounts, serviceName)
	delete(hm.successCounts, serviceName)
}

// GetHealthyServices returns all currently healthy services
func (hm *HealthMonitor) GetHealthyServices() []*ServiceInfo {
	return hm.registry.ListHealthy()
}

// GetUnhealthyServices returns all currently unhealthy services
func (hm *HealthMonitor) GetUnhealthyServices() []*ServiceInfo {
	allServices := hm.registry.List()
	unhealthy := make([]*ServiceInfo, 0)

	for _, service := range allServices {
		if !service.Healthy {
			unhealthy = append(unhealthy, service)
		}
	}

	return unhealthy
}
