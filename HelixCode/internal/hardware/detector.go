package hardware

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// HardwareInfo contains comprehensive hardware information
type HardwareInfo struct {
	CPU      CPUInfo      `json:"cpu"`
	GPU      GPUInfo      `json:"gpu"`
	Memory   MemoryInfo   `json:"memory"`
	Platform PlatformInfo `json:"platform"`
}

// CPUInfo contains CPU-specific information
type CPUInfo struct {
	Architecture string `json:"architecture"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	Cores        int    `json:"cores"`
	Threads      int    `json:"threads"`
}

// GPUInfo contains GPU-specific information
type GPUInfo struct {
	Vendor        string `json:"vendor"`
	Model         string `json:"model"`
	VRAM          string `json:"vram"`
	SupportsCUDA  bool   `json:"supports_cuda"`
	SupportsMetal bool   `json:"supports_metal"`
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	TotalRAM string `json:"total_ram"`
}

// PlatformInfo contains platform-specific information
type PlatformInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Hostname     string `json:"hostname"`
}

// Detector handles hardware detection
type Detector struct {
	info *HardwareInfo
}

// NewDetector creates a new hardware detector
func NewDetector() *Detector {
	return &Detector{
		info: &HardwareInfo{},
	}
}

// Detect performs comprehensive hardware detection
func (d *Detector) Detect() (*HardwareInfo, error) {
	log.Println("🔍 Starting hardware detection...")

	// Detect CPU information
	if err := d.detectCPU(); err != nil {
		log.Printf("Warning: CPU detection failed: %v", err)
	}

	// Detect GPU information
	if err := d.detectGPU(); err != nil {
		log.Printf("Warning: GPU detection failed: %v", err)
	}

	// Detect memory information
	if err := d.detectMemory(); err != nil {
		log.Printf("Warning: Memory detection failed: %v", err)
	}

	// Detect platform information
	if err := d.detectPlatform(); err != nil {
		log.Printf("Warning: Platform detection failed: %v", err)
	}

	log.Println("✅ Hardware detection completed")
	return d.info, nil
}

// GetOptimalModelSize calculates the optimal model size for the hardware
func (d *Detector) GetOptimalModelSize() string {
	// Calculate based on available VRAM and RAM
	var vramGB, ramGB int

	// Parse VRAM
	if strings.Contains(d.info.GPU.VRAM, "GB") {
		vramGB, _ = strconv.Atoi(strings.TrimSuffix(d.info.GPU.VRAM, "GB"))
	} else if strings.Contains(d.info.GPU.VRAM, "MB") {
		vramMB, _ := strconv.Atoi(strings.TrimSuffix(d.info.GPU.VRAM, "MB"))
		vramGB = vramMB / 1024
	}

	// Parse RAM
	if strings.Contains(d.info.Memory.TotalRAM, "GB") {
		ramGB, _ = strconv.Atoi(strings.TrimSuffix(d.info.Memory.TotalRAM, "GB"))
	}

	// Determine optimal model size based on available memory
	totalMemory := vramGB + (ramGB / 2) // Use half of RAM for model loading

	switch {
	case totalMemory >= 32:
		return "70B" // 70B parameter models
	case totalMemory >= 16:
		return "34B" // 34B parameter models
	case totalMemory >= 8:
		return "13B" // 13B parameter models
	case totalMemory >= 4:
		return "7B" // 7B parameter models
	default:
		return "3B" // 3B parameter models
	}
}

// CanRunModel checks if the hardware can run a specific model size
func (d *Detector) CanRunModel(modelSize string) bool {
	optimalSize := d.GetOptimalModelSize()

	// Model size comparison (larger numbers are better)
	sizeOrder := map[string]int{
		"3B":  1,
		"7B":  2,
		"13B": 3,
		"34B": 4,
		"70B": 5,
	}

	requestedOrder := sizeOrder[modelSize]
	optimalOrder := sizeOrder[optimalSize]

	return requestedOrder <= optimalOrder
}

// GetCompilationFlags returns hardware-specific compilation flags
func (d *Detector) GetCompilationFlags() []string {
	var flags []string

	// GPU acceleration flags
	if d.info.GPU.SupportsCUDA {
		flags = append(flags, "-DGGML_USE_CUBLAS")
	}
	if d.info.GPU.SupportsMetal {
		flags = append(flags, "-DGGML_USE_METAL")
	}

	return flags
}

// Private detection methods

func (d *Detector) detectCPU() error {
	d.info.CPU = CPUInfo{
		Architecture: runtime.GOARCH,
		Cores:        runtime.NumCPU(),
		Threads:      runtime.NumCPU(), // Simplified - same as cores
	}

	// Try to get more detailed CPU info
	switch runtime.GOOS {
	case "linux":
		return d.detectCPULinux()
	case "darwin":
		return d.detectCPUMacOS()
	default:
		return d.detectCPUGeneric()
	}
}

func (d *Detector) detectCPULinux() error {
	// Read /proc/cpuinfo for detailed CPU information
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				d.info.CPU.Model = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "vendor_id") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				d.info.CPU.Vendor = strings.TrimSpace(parts[1])
			}
		}
	}

	return nil
}

func (d *Detector) detectCPUMacOS() error {
	// Use sysctl to get CPU information on macOS
	cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
	if output, err := cmd.Output(); err == nil {
		d.info.CPU.Model = strings.TrimSpace(string(output))
	}

	cmd = exec.Command("sysctl", "-n", "machdep.cpu.vendor")
	if output, err := cmd.Output(); err == nil {
		d.info.CPU.Vendor = strings.TrimSpace(string(output))
	}

	return nil
}

func (d *Detector) detectCPUGeneric() error {
	// Generic fallback
	d.info.CPU.Model = "Unknown"
	d.info.CPU.Vendor = "Unknown"
	return nil
}

func (d *Detector) detectGPU() error {
	// Try different GPU detection methods based on platform
	switch runtime.GOOS {
	case "linux":
		return d.detectGPULinux()
	case "darwin":
		return d.detectGPUMacOS()
	default:
		return d.detectGPUGeneric()
	}
}

func (d *Detector) detectGPULinux() error {
	// Try to detect NVIDIA GPUs
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		return d.detectNVIDIA()
	}

	// Fallback to generic detection
	return d.detectGPUGeneric()
}

func (d *Detector) detectGPUMacOS() error {
	// macOS GPU detection
	cmd := exec.Command("system_profiler", "SPDisplaysDataType")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Chipset Model:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					d.info.GPU.Model = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(line, "VRAM") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					d.info.GPU.VRAM = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	d.info.GPU.Vendor = "Apple"
	d.info.GPU.SupportsMetal = true

	return nil
}

func (d *Detector) detectGPUGeneric() error {
	// Generic fallback
	d.info.GPU.Model = "Unknown"
	d.info.GPU.Vendor = "Unknown"
	return nil
}

func (d *Detector) detectNVIDIA() error {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			parts := strings.Split(lines[0], ", ")
			if len(parts) >= 2 {
				d.info.GPU.Model = strings.TrimSpace(parts[0])
				d.info.GPU.VRAM = strings.TrimSpace(parts[1])
			}
		}
	}

	d.info.GPU.Vendor = "NVIDIA"
	d.info.GPU.SupportsCUDA = true

	return nil
}

func (d *Detector) detectMemory() error {
	// Simplified memory detection
	// In a real implementation, this would use platform-specific methods
	d.info.Memory.TotalRAM = "16GB" // Default fallback
	return nil
}

func (d *Detector) detectPlatform() error {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	d.info.Platform = PlatformInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Hostname:     hostname,
	}

	return nil
}
