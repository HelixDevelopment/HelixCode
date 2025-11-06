package voice

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// AudioDevice represents an audio input device
type AudioDevice struct {
	ID          string   // System device ID
	Name        string   // Human-readable name
	IsDefault   bool     // Whether this is the system default
	SampleRates []int    // Supported sample rates
	Channels    int      // Number of channels
	IsAvailable bool     // Current availability status
	Driver      string   // Audio driver (CoreAudio, ALSA, etc.)
}

// DeviceManager handles audio device enumeration and selection
type DeviceManager struct {
	devices         []AudioDevice
	activeDevice    *AudioDevice
	refreshInterval time.Duration
	mu              sync.RWMutex
}

// NewDeviceManager creates a new device manager
func NewDeviceManager() (*DeviceManager, error) {
	dm := &DeviceManager{
		devices:         []AudioDevice{},
		refreshInterval: 30 * time.Second,
	}

	// Initialize device list
	if err := dm.RefreshDevices(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize devices: %w", err)
	}

	return dm, nil
}

// ListDevices enumerates all available audio input devices
func (d *DeviceManager) ListDevices(ctx context.Context) ([]AudioDevice, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.devices) == 0 {
		return nil, ErrNoDevicesFound
	}

	// Return a copy to prevent external modifications
	devices := make([]AudioDevice, len(d.devices))
	copy(devices, d.devices)

	return devices, nil
}

// GetDevice retrieves a specific device by ID
func (d *DeviceManager) GetDevice(deviceID string) (*AudioDevice, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, device := range d.devices {
		if device.ID == deviceID {
			deviceCopy := device
			return &deviceCopy, nil
		}
	}

	return nil, ErrDeviceNotFound
}

// GetDefaultDevice returns the system default input device
func (d *DeviceManager) GetDefaultDevice() (*AudioDevice, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, device := range d.devices {
		if device.IsDefault {
			deviceCopy := device
			return &deviceCopy, nil
		}
	}

	// If no default found, return first available
	if len(d.devices) > 0 {
		deviceCopy := d.devices[0]
		return &deviceCopy, nil
	}

	return nil, ErrNoDevicesFound
}

// SelectDevice sets the active device for recording
func (d *DeviceManager) SelectDevice(deviceID string) error {
	device, err := d.GetDevice(deviceID)
	if err != nil {
		return err
	}

	if !device.IsAvailable {
		return ErrDeviceUnavailable
	}

	d.mu.Lock()
	d.activeDevice = device
	d.mu.Unlock()

	return nil
}

// GetActiveDevice returns the currently selected device
func (d *DeviceManager) GetActiveDevice() *AudioDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.activeDevice != nil {
		deviceCopy := *d.activeDevice
		return &deviceCopy
	}

	return nil
}

// RefreshDevices updates the device list
func (d *DeviceManager) RefreshDevices(ctx context.Context) error {
	devices, err := d.enumerateDevices()
	if err != nil {
		return err
	}

	d.mu.Lock()
	d.devices = devices
	d.mu.Unlock()

	return nil
}

// ValidateDevice checks if a device is available and properly configured
func (d *DeviceManager) ValidateDevice(device *AudioDevice) error {
	if device == nil {
		return fmt.Errorf("device is nil")
	}

	if !device.IsAvailable {
		return ErrDeviceUnavailable
	}

	if device.Channels < 1 {
		return fmt.Errorf("device has no input channels")
	}

	if len(device.SampleRates) == 0 {
		return fmt.Errorf("device has no supported sample rates")
	}

	return nil
}

// enumerateDevices performs platform-specific device enumeration
func (d *DeviceManager) enumerateDevices() ([]AudioDevice, error) {
	// Platform-specific implementation
	switch runtime.GOOS {
	case "darwin":
		return d.enumerateMacOSDevices()
	case "linux":
		return d.enumerateLinuxDevices()
	case "windows":
		return d.enumerateWindowsDevices()
	default:
		// Return mock device for unsupported platforms
		return d.enumerateMockDevices()
	}
}

// enumerateMacOSDevices enumerates devices on macOS using CoreAudio
func (d *DeviceManager) enumerateMacOSDevices() ([]AudioDevice, error) {
	// In a real implementation, this would use CoreAudio APIs
	// For now, return a mock device for testing
	return []AudioDevice{
		{
			ID:          "default",
			Name:        "Built-in Microphone",
			IsDefault:   true,
			SampleRates: []int{8000, 16000, 44100, 48000},
			Channels:    1,
			IsAvailable: true,
			Driver:      "CoreAudio",
		},
	}, nil
}

// enumerateLinuxDevices enumerates devices on Linux using ALSA/PulseAudio
func (d *DeviceManager) enumerateLinuxDevices() ([]AudioDevice, error) {
	// In a real implementation, this would use ALSA or PulseAudio APIs
	return []AudioDevice{
		{
			ID:          "default",
			Name:        "Default Audio Device",
			IsDefault:   true,
			SampleRates: []int{8000, 16000, 44100, 48000},
			Channels:    1,
			IsAvailable: true,
			Driver:      "ALSA",
		},
	}, nil
}

// enumerateWindowsDevices enumerates devices on Windows using WASAPI
func (d *DeviceManager) enumerateWindowsDevices() ([]AudioDevice, error) {
	// In a real implementation, this would use WASAPI APIs
	return []AudioDevice{
		{
			ID:          "default",
			Name:        "Default Audio Device",
			IsDefault:   true,
			SampleRates: []int{8000, 16000, 44100, 48000},
			Channels:    1,
			IsAvailable: true,
			Driver:      "WASAPI",
		},
	}, nil
}

// enumerateMockDevices returns mock devices for testing
func (d *DeviceManager) enumerateMockDevices() ([]AudioDevice, error) {
	return []AudioDevice{
		{
			ID:          "mock-default",
			Name:        "Mock Default Microphone",
			IsDefault:   true,
			SampleRates: []int{8000, 16000, 44100, 48000},
			Channels:    1,
			IsAvailable: true,
			Driver:      "Mock",
		},
		{
			ID:          "mock-usb",
			Name:        "Mock USB Microphone",
			IsDefault:   false,
			SampleRates: []int{16000, 44100, 48000},
			Channels:    1,
			IsAvailable: true,
			Driver:      "Mock",
		},
	}, nil
}
