package hardware

import (
	"runtime"
	"time"
)

// HardwareProfile represents the hardware capabilities of the system
type HardwareProfile struct {
	CPU     CPUInfo     `json:"cpu"`
	GPU     *GPUInfo    `json:"gpu,omitempty"`
	Memory  MemoryInfo  `json:"memory"`
	OS      OSInfo      `json:"os"`
	Network NetworkInfo `json:"network"`
}

// CPUInfo represents CPU information
type CPUInfo struct {
	Architecture string `json:"architecture"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	Cores        int    `json:"cores"`
	Threads      int    `json:"threads"`
	HasAVX       bool   `json:"has_avx"`
	HasAVX2      bool   `json:"has_avx2"`
	HasNEON      bool   `json:"has_neon"`
	Arch         string `json:"arch"`
	Frequency    int    `json:"frequency_mhz"`
	CacheSize    int    `json:"cache_size_kb"`
}

// GPUInfo represents GPU information
type GPUInfo struct {
	Name              string  `json:"name"`
	Type              GPUType `json:"type"`
	Vendor            string  `json:"vendor"`
	Model             string  `json:"model"`
	VRAM              string  `json:"vram"` // e.g., "8GB", "16GB"
	ComputeCapability float64 `json:"compute_capability"`
	CUDAVersion       string  `json:"cuda_version,omitempty"`
	SupportsCUDA      bool    `json:"supports_cuda"`
	SupportsMetal     bool    `json:"supports_metal"`
}

// GPUType represents the type of GPU
type GPUType string

const (
	GPUTypeNVIDIA GPUType = "nvidia"
	GPUTypeAMD    GPUType = "amd"
	GPUTypeApple  GPUType = "apple"
	GPUTypeIntel  GPUType = "intel"
)

// MemoryInfo represents memory information
type MemoryInfo struct {
	Total     int64  `json:"total_bytes"`
	Available int64  `json:"available_bytes"`
	SwapTotal int64  `json:"swap_total_bytes"`
	TotalRAM  string `json:"total_ram"`
}

// OSInfo represents OS information
type OSInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

// OSType represents the type of operating system
type OSType string

const (
	OSTypeLinux   OSType = "linux"
	OSTypeMacOS   OSType = "macos"
	OSTypeWindows OSType = "windows"
)

// Arch represents the system architecture
type Arch string

const (
	ArchX86_64 Arch = "x86_64"
	ArchARM64  Arch = "arm64"
	ArchARM32  Arch = "arm32"
)

// NetworkInfo represents network information
type NetworkInfo struct {
	HasInternet bool          `json:"has_internet"`
	Latency     time.Duration `json:"latency"`
	Bandwidth   int64         `json:"bandwidth_bps"`
}

// HardwareDetector detects hardware capabilities
type HardwareDetector struct{}

// NewHardwareDetector creates a new hardware detector
func NewHardwareDetector() *HardwareDetector {
	return &HardwareDetector{}
}

// GetProfile returns the current hardware profile
func (hd *HardwareDetector) GetProfile() *HardwareProfile {
	return &HardwareProfile{
		CPU: CPUInfo{
			Cores:     runtime.NumCPU(),
			Threads:   runtime.NumCPU(),
			HasAVX:    false, // Default to false
			HasAVX2:   false, // Default to false
			HasNEON:   false, // Default to false
			Arch:      runtime.GOARCH,
			Frequency: 0, // Unknown
			CacheSize: 0, // Unknown
		},
		Memory: MemoryInfo{
			Total:     8 * 1024 * 1024 * 1024, // 8GB default
			Available: 8 * 1024 * 1024 * 1024, // Assume all available
			SwapTotal: 0,
		},
		OS: OSInfo{
			Name:    runtime.GOOS,
			Version: "", // Unknown
			Arch:    runtime.GOARCH,
		},
		Network: NetworkInfo{
			HasInternet: true,
			Latency:     0,
			Bandwidth:   0,
		},
	}
}

// DefaultProfile returns a default hardware profile
func DefaultProfile() *HardwareProfile {
	detector := NewHardwareDetector()
	return detector.GetProfile()
}
