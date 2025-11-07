package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	// DefaultMulticastAddress is the multicast group for service discovery
	DefaultMulticastAddress = "239.255.0.1:7001"

	// DefaultAnnouncementInterval is how often services announce themselves
	DefaultAnnouncementInterval = 5 * time.Second

	// DefaultDiscoveryTimeout is how long to wait for announcements
	DefaultDiscoveryTimeout = 3 * time.Second
)

var (
	// ErrBroadcastNotRunning is returned when attempting operations on a stopped broadcast service
	ErrBroadcastNotRunning = errors.New("broadcast service not running")

	// ErrInvalidMessage is returned when a malformed message is received
	ErrInvalidMessage = errors.New("invalid broadcast message")
)

// BroadcastMessage represents a service announcement message
type BroadcastMessage struct {
	Type        string                 `json:"type"`        // "announce", "query", "response"
	ServiceInfo ServiceInfo            `json:"service"`     // Service information
	Timestamp   time.Time              `json:"timestamp"`   // Message timestamp
	Metadata    map[string]interface{} `json:"metadata"`    // Additional metadata
}

// BroadcastConfig configures the broadcast service
type BroadcastConfig struct {
	// MulticastAddress is the multicast group address
	MulticastAddress string

	// AnnouncementInterval is how often to announce this service
	AnnouncementInterval time.Duration

	// DiscoveryTimeout is how long to wait for discovery responses
	DiscoveryTimeout time.Duration

	// Interface is the network interface to use (empty for default)
	Interface string

	// TTL is the multicast TTL (time-to-live/hop limit)
	TTL int
}

// DefaultBroadcastConfig returns default broadcast configuration
func DefaultBroadcastConfig() BroadcastConfig {
	return BroadcastConfig{
		MulticastAddress:     DefaultMulticastAddress,
		AnnouncementInterval: DefaultAnnouncementInterval,
		DiscoveryTimeout:     DefaultDiscoveryTimeout,
		Interface:            "",
		TTL:                  2, // Local network only
	}
}

// BroadcastService handles UDP multicast service discovery
type BroadcastService struct {
	config  BroadcastConfig
	conn    *net.UDPConn
	running bool
	mu      sync.RWMutex

	// Services discovered via broadcast
	discovered map[string]*ServiceInfo
	discoveryMu sync.RWMutex

	// Local service to announce
	localService *ServiceInfo

	// Channels for coordination
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewBroadcastService creates a new broadcast service
func NewBroadcastService(config BroadcastConfig) *BroadcastService {
	return &BroadcastService{
		config:     config,
		discovered: make(map[string]*ServiceInfo),
		stopChan:   make(chan struct{}),
	}
}

// Start starts the broadcast service
func (bs *BroadcastService) Start() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.running {
		return errors.New("broadcast service already running")
	}

	// Parse multicast address
	addr, err := net.ResolveUDPAddr("udp", bs.config.MulticastAddress)
	if err != nil {
		return fmt.Errorf("failed to resolve multicast address: %w", err)
	}

	// Listen on the multicast address
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to listen on multicast address: %w", err)
	}

	// Set read buffer size
	if err := conn.SetReadBuffer(65536); err != nil {
		conn.Close()
		return fmt.Errorf("failed to set read buffer: %w", err)
	}

	bs.conn = conn
	bs.running = true

	// Start listener goroutine
	bs.wg.Add(1)
	go bs.listen()

	// Start announcer goroutine if we have a local service
	if bs.localService != nil {
		bs.wg.Add(1)
		go bs.announce()
	}

	return nil
}

// Stop stops the broadcast service
func (bs *BroadcastService) Stop() error {
	bs.mu.Lock()
	if !bs.running {
		bs.mu.Unlock()
		return ErrBroadcastNotRunning
	}

	close(bs.stopChan)
	bs.running = false
	bs.mu.Unlock()

	// Close connection to unblock listener
	if bs.conn != nil {
		bs.conn.Close()
	}

	// Wait for goroutines to finish
	bs.wg.Wait()

	return nil
}

// SetLocalService sets the service to announce
func (bs *BroadcastService) SetLocalService(info ServiceInfo) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.localService = &info

	// If already running, start announcer
	if bs.running {
		bs.wg.Add(1)
		go bs.announce()
	}

	return nil
}

// Discover discovers services via broadcast
func (bs *BroadcastService) Discover(serviceName string) (*ServiceInfo, error) {
	bs.mu.RLock()
	if !bs.running {
		bs.mu.RUnlock()
		return nil, ErrBroadcastNotRunning
	}
	bs.mu.RUnlock()

	// Send query message
	if err := bs.sendQuery(serviceName); err != nil {
		return nil, fmt.Errorf("failed to send query: %w", err)
	}

	// Wait for responses
	deadline := time.Now().Add(bs.config.DiscoveryTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if service has been discovered
			bs.discoveryMu.RLock()
			info, exists := bs.discovered[serviceName]
			bs.discoveryMu.RUnlock()

			if exists && info.Healthy {
				// Return a copy
				infoCopy := *info
				return &infoCopy, nil
			}

			if time.Now().After(deadline) {
				return nil, fmt.Errorf("discovery timeout: service %s not found", serviceName)
			}

		case <-bs.stopChan:
			return nil, ErrBroadcastNotRunning
		}
	}
}

// List returns all discovered services
func (bs *BroadcastService) List() []*ServiceInfo {
	bs.discoveryMu.RLock()
	defer bs.discoveryMu.RUnlock()

	services := make([]*ServiceInfo, 0, len(bs.discovered))
	for _, info := range bs.discovered {
		// Return copies
		infoCopy := *info
		services = append(services, &infoCopy)
	}

	return services
}

// IsRunning returns whether the broadcast service is running
func (bs *BroadcastService) IsRunning() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.running
}

// Internal methods

func (bs *BroadcastService) listen() {
	defer bs.wg.Done()

	buffer := make([]byte, 65536)

	for {
		select {
		case <-bs.stopChan:
			return
		default:
		}

		// Set read deadline to allow checking stop channel
		bs.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, addr, err := bs.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// Connection closed or other error
			return
		}

		// Parse message
		var msg BroadcastMessage
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue // Skip invalid messages
		}

		// Handle message
		bs.handleMessage(&msg, addr)
	}
}

func (bs *BroadcastService) announce() {
	defer bs.wg.Done()

	ticker := time.NewTicker(bs.config.AnnouncementInterval)
	defer ticker.Stop()

	// Announce immediately on start
	bs.sendAnnouncement()

	for {
		select {
		case <-ticker.C:
			bs.sendAnnouncement()

		case <-bs.stopChan:
			return
		}
	}
}

func (bs *BroadcastService) sendAnnouncement() {
	bs.mu.RLock()
	localService := bs.localService
	bs.mu.RUnlock()

	if localService == nil {
		return
	}

	msg := BroadcastMessage{
		Type:        "announce",
		ServiceInfo: *localService,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	bs.sendMessage(&msg)
}

func (bs *BroadcastService) sendQuery(serviceName string) error {
	msg := BroadcastMessage{
		Type: "query",
		ServiceInfo: ServiceInfo{
			Name: serviceName,
		},
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	return bs.sendMessage(&msg)
}

func (bs *BroadcastService) sendResponse(serviceName string) error {
	bs.mu.RLock()
	localService := bs.localService
	bs.mu.RUnlock()

	if localService == nil || localService.Name != serviceName {
		return nil // Not our service
	}

	msg := BroadcastMessage{
		Type:        "response",
		ServiceInfo: *localService,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	return bs.sendMessage(&msg)
}

func (bs *BroadcastService) sendMessage(msg *BroadcastMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Parse multicast address
	addr, err := net.ResolveUDPAddr("udp", bs.config.MulticastAddress)
	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	// Send message
	_, err = bs.conn.WriteToUDP(data, addr)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (bs *BroadcastService) handleMessage(msg *BroadcastMessage, addr *net.UDPAddr) {
	// Validate timestamp (reject messages older than 1 minute)
	if time.Since(msg.Timestamp) > 1*time.Minute {
		return
	}

	switch msg.Type {
	case "announce":
		bs.handleAnnouncement(&msg.ServiceInfo)

	case "query":
		// Send response if we have the requested service
		bs.sendResponse(msg.ServiceInfo.Name)

	case "response":
		bs.handleAnnouncement(&msg.ServiceInfo)
	}
}

func (bs *BroadcastService) handleAnnouncement(info *ServiceInfo) {
	bs.discoveryMu.Lock()
	defer bs.discoveryMu.Unlock()

	// Update last seen time
	info.LastHeartbeat = time.Now()

	// Store or update service
	bs.discovered[info.Name] = info
}

// CleanExpired removes services that haven't announced recently
func (bs *BroadcastService) CleanExpired() {
	bs.discoveryMu.Lock()
	defer bs.discoveryMu.Unlock()

	now := time.Now()
	for name, info := range bs.discovered {
		// Consider services expired if no announcement for 3x announcement interval
		if now.Sub(info.LastHeartbeat) > 3*bs.config.AnnouncementInterval {
			delete(bs.discovered, name)
		}
	}
}
