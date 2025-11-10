package discovery

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var (
	// ErrServiceNotFound is returned when a service is not found in the registry
	ErrServiceNotFound = errors.New("service not found")

	// ErrServiceAlreadyRegistered is returned when trying to register a service that already exists
	ErrServiceAlreadyRegistered = errors.New("service already registered")

	// ErrInvalidServiceInfo is returned when service information is invalid
	ErrInvalidServiceInfo = errors.New("invalid service information")
)

// ServiceInfo represents information about a registered service
type ServiceInfo struct {
	Name          string            `json:"name"`
	Host          string            `json:"host"`
	Port          int               `json:"port"`
	Protocol      string            `json:"protocol"` // tcp, udp, http, https, grpc
	Version       string            `json:"version"`
	Metadata      map[string]string `json:"metadata"`
	RegisteredAt  time.Time         `json:"registered_at"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	TTL           time.Duration     `json:"ttl"`
	Healthy       bool              `json:"healthy"`
}

// Address returns the full address of the service
func (s *ServiceInfo) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// IsExpired checks if the service registration has expired based on TTL
func (s *ServiceInfo) IsExpired() bool {
	if s.TTL == 0 {
		return false // No TTL means never expires
	}
	return time.Since(s.LastHeartbeat) > s.TTL
}

// RegistryConfig configures the service registry
type RegistryConfig struct {
	// DefaultTTL is the default TTL for service registrations
	DefaultTTL time.Duration

	// CleanupInterval is how often to clean up expired services
	CleanupInterval time.Duration

	// EnableHealthChecks enables automatic health checking
	EnableHealthChecks bool

	// HealthCheckInterval is how often to health check services
	HealthCheckInterval time.Duration
}

// DefaultRegistryConfig returns default registry configuration
func DefaultRegistryConfig() RegistryConfig {
	return RegistryConfig{
		DefaultTTL:          30 * time.Second,
		CleanupInterval:     10 * time.Second,
		EnableHealthChecks:  true,
		HealthCheckInterval: 15 * time.Second,
	}
}

// ServiceRegistry manages service registration and discovery
type ServiceRegistry struct {
	config    RegistryConfig
	services  map[string]*ServiceInfo // key: service name
	mu        sync.RWMutex
	stopChan  chan struct{}
	cleanupWg sync.WaitGroup
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(config RegistryConfig) *ServiceRegistry {
	return &ServiceRegistry{
		config:   config,
		services: make(map[string]*ServiceInfo),
		stopChan: make(chan struct{}),
	}
}

// NewDefaultServiceRegistry creates a service registry with default configuration
func NewDefaultServiceRegistry() *ServiceRegistry {
	return NewServiceRegistry(DefaultRegistryConfig())
}

// Start starts the registry background tasks (cleanup, health checks)
func (r *ServiceRegistry) Start() {
	r.cleanupWg.Add(1)
	go r.cleanupLoop()

	if r.config.EnableHealthChecks {
		r.cleanupWg.Add(1)
		go r.healthCheckLoop()
	}
}

// Stop stops the registry background tasks
func (r *ServiceRegistry) Stop() {
	close(r.stopChan)
	r.cleanupWg.Wait()
}

// Register registers a new service with the registry
func (r *ServiceRegistry) Register(info ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate service info
	if err := r.validateServiceInfo(&info); err != nil {
		return err
	}

	// Check if service already exists
	if _, exists := r.services[info.Name]; exists {
		return ErrServiceAlreadyRegistered
	}

	// Set timestamps
	now := time.Now()
	info.RegisteredAt = now
	info.LastHeartbeat = now
	info.Healthy = true

	// Set default TTL if not specified
	if info.TTL == 0 {
		info.TTL = r.config.DefaultTTL
	}

	// Store service
	r.services[info.Name] = &info

	return nil
}

// Deregister removes a service from the registry
func (r *ServiceRegistry) Deregister(serviceName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[serviceName]; !exists {
		return ErrServiceNotFound
	}

	delete(r.services, serviceName)
	return nil
}

// Update updates an existing service's information
func (r *ServiceRegistry) Update(serviceName string, info ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.services[serviceName]
	if !exists {
		return ErrServiceNotFound
	}

	// Validate new info
	if err := r.validateServiceInfo(&info); err != nil {
		return err
	}

	// Preserve original registration time
	info.RegisteredAt = existing.RegisteredAt
	info.LastHeartbeat = time.Now()

	// Set default TTL if not specified
	if info.TTL == 0 {
		info.TTL = r.config.DefaultTTL
	}

	r.services[serviceName] = &info
	return nil
}

// Heartbeat updates the last heartbeat time for a service
func (r *ServiceRegistry) Heartbeat(serviceName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	service, exists := r.services[serviceName]
	if !exists {
		return ErrServiceNotFound
	}

	service.LastHeartbeat = time.Now()
	return nil
}

// Get retrieves service information by name
func (r *ServiceRegistry) Get(serviceName string) (*ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[serviceName]
	if !exists {
		return nil, ErrServiceNotFound
	}

	// Return a copy to prevent external modification
	serviceCopy := *service
	return &serviceCopy, nil
}

// List returns all registered services
func (r *ServiceRegistry) List() []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*ServiceInfo, 0, len(r.services))
	for _, service := range r.services {
		serviceCopy := *service
		services = append(services, &serviceCopy)
	}

	return services
}

// ListByProtocol returns all services using a specific protocol
func (r *ServiceRegistry) ListByProtocol(protocol string) []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*ServiceInfo, 0)
	for _, service := range r.services {
		if service.Protocol == protocol {
			serviceCopy := *service
			services = append(services, &serviceCopy)
		}
	}

	return services
}

// ListHealthy returns all healthy services
func (r *ServiceRegistry) ListHealthy() []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*ServiceInfo, 0)
	for _, service := range r.services {
		if service.Healthy && !service.IsExpired() {
			serviceCopy := *service
			services = append(services, &serviceCopy)
		}
	}

	return services
}

// UpdateHealth updates the health status of a service
func (r *ServiceRegistry) UpdateHealth(serviceName string, healthy bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	service, exists := r.services[serviceName]
	if !exists {
		return ErrServiceNotFound
	}

	service.Healthy = healthy
	return nil
}

// Internal helper methods

func (r *ServiceRegistry) validateServiceInfo(info *ServiceInfo) error {
	if info.Name == "" {
		return fmt.Errorf("%w: service name is required", ErrInvalidServiceInfo)
	}
	if info.Host == "" {
		return fmt.Errorf("%w: service host is required", ErrInvalidServiceInfo)
	}
	if info.Port < 1 || info.Port > 65535 {
		return fmt.Errorf("%w: invalid port %d", ErrInvalidServiceInfo, info.Port)
	}
	if info.Protocol == "" {
		info.Protocol = "tcp" // Default protocol
	}
	return nil
}

func (r *ServiceRegistry) cleanupLoop() {
	defer r.cleanupWg.Done()

	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanupExpired()
		case <-r.stopChan:
			return
		}
	}
}

func (r *ServiceRegistry) cleanupExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, service := range r.services {
		if service.IsExpired() {
			delete(r.services, name)
		}
	}
}

func (r *ServiceRegistry) healthCheckLoop() {
	defer r.cleanupWg.Done()

	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.performHealthChecks()
		case <-r.stopChan:
			return
		}
	}
}

func (r *ServiceRegistry) performHealthChecks() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, service := range r.services {
		// First check heartbeat-based health
		if service.TTL > 0 && time.Since(service.LastHeartbeat) > service.TTL/2 {
			service.Healthy = false
			continue
		}

		// Perform protocol-specific health check
		healthy := r.checkServiceHealth(service)
		service.Healthy = healthy
	}
}

// checkServiceHealth performs protocol-specific health check
func (r *ServiceRegistry) checkServiceHealth(service *ServiceInfo) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch service.Protocol {
	case "http", "https":
		return r.checkHTTPHealth(ctx, service)
	case "grpc":
		return r.checkGRPCHealth(ctx, service)
	case "tcp", "udp":
		return r.checkTCPHealth(ctx, service)
	default:
		// For unknown protocols, rely on heartbeat only
		return service.Healthy
	}
}

// checkHTTPHealth performs HTTP/HTTPS health check
func (r *ServiceRegistry) checkHTTPHealth(ctx context.Context, service *ServiceInfo) bool {
	// Check for custom health endpoint in metadata
	healthPath := service.Metadata["health_endpoint"]
	if healthPath == "" {
		healthPath = "/health"
	}

	scheme := service.Protocol
	if scheme == "" {
		scheme = "http"
	}

	url := fmt.Sprintf("%s://%s:%d%s", scheme, service.Host, service.Port, healthPath)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // For testing/internal services
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx status codes as healthy
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// checkGRPCHealth performs gRPC health check using standard health checking protocol
func (r *ServiceRegistry) checkGRPCHealth(ctx context.Context, service *ServiceInfo) bool {
	address := fmt.Sprintf("%s:%d", service.Host, service.Port)

	// Create gRPC connection with timeout
	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Use standard gRPC health checking protocol
	healthClient := grpc_health_v1.NewHealthClient(conn)

	// Check the service specified in metadata, or empty string for server health
	serviceName := service.Metadata["grpc_service_name"]

	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: serviceName,
	})
	if err != nil {
		return false
	}

	return resp.Status == grpc_health_v1.HealthCheckResponse_SERVING
}

// checkTCPHealth performs TCP connection health check
func (r *ServiceRegistry) checkTCPHealth(ctx context.Context, service *ServiceInfo) bool {
	address := fmt.Sprintf("%s:%d", service.Host, service.Port)

	// Attempt to establish TCP connection
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Successfully connected, service is healthy
	return true
}
