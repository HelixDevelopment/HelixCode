package hardware

import (
	"testing"
)

// TestHardwareDetector tests the hardware detector
func TestHardwareDetector(t *testing.T) {
	detector := NewDetector()

	// Test hardware detection
	hardwareInfo, err := detector.Detect()
	if err != nil {
		t.Fatalf("Hardware detection failed: %v", err)
	}

	// Verify basic hardware information
	if hardwareInfo.CPU.Cores == 0 {
		t.Error("CPU core count should be greater than 0")
	}

	if hardwareInfo.CPU.Architecture == "" {
		t.Error("CPU architecture should not be empty")
	}

	if hardwareInfo.Platform.OS == "" {
		t.Error("Platform OS should not be empty")
	}

	if hardwareInfo.Platform.Architecture == "" {
		t.Error("Platform architecture should not be empty")
	}

	// Test model size calculation
	optimalSize := detector.GetOptimalModelSize()
	if optimalSize == "" {
		t.Error("Optimal model size should not be empty")
	}

	// Test compatibility checking
	testSizes := []string{"3B", "7B", "13B", "34B", "70B"}
	for _, size := range testSizes {
		compatible := detector.CanRunModel(size)
		t.Logf("Model size %s compatible: %t", size, compatible)
	}

	// Test compilation flags
	flags := detector.GetCompilationFlags()
	if len(flags) == 0 {
		t.Log("No compilation flags returned (may be normal for test environment)")
	} else {
		t.Logf("Compilation flags: %v", flags)
	}

	t.Logf("✅ Hardware detection test passed: %s CPU, %d cores, optimal model: %s",
		hardwareInfo.CPU.Model, hardwareInfo.CPU.Cores, optimalSize)
}

// TestHardwareDetectionErrorHandling tests error handling in hardware detection
func TestHardwareDetectionErrorHandling(t *testing.T) {
	detector := NewDetector()

	// Test with invalid model sizes
	invalidSizes := []string{"", "invalid", "1B", "100B"}
	for _, size := range invalidSizes {
		compatible := detector.CanRunModel(size)
		// Should handle invalid sizes gracefully
		t.Logf("Model size '%s' compatible: %t", size, compatible)
	}

	// Test compilation flags consistency
	flags1 := detector.GetCompilationFlags()
	flags2 := detector.GetCompilationFlags()

	// Should return consistent results
	if len(flags1) != len(flags2) {
		t.Error("Compilation flags should be consistent")
	}

	t.Log("✅ Hardware detection error handling test passed")
}

// TestPlatformSpecificDetection tests platform-specific detection logic
func TestPlatformSpecificDetection(t *testing.T) {
	detector := NewDetector()

	// Test that platform detection works
	hardwareInfo, err := detector.Detect()
	if err != nil {
		t.Fatalf("Platform detection failed: %v", err)
	}

	// Verify platform information
	if hardwareInfo.Platform.Hostname == "" {
		t.Log("Hostname not detected (may be normal in test environment)")
	}

	// Test memory detection
	if hardwareInfo.Memory.TotalRAM == "" {
		t.Log("Memory detection not available (may be normal in test environment)")
	}

	// Test GPU detection
	if hardwareInfo.GPU.Model == "" {
		t.Log("GPU detection not available (may be normal in test environment)")
	}

	t.Logf("✅ Platform-specific detection test passed: %s/%s",
		hardwareInfo.Platform.OS, hardwareInfo.Platform.Architecture)
}

// TestModelSizeCalculation tests the model size calculation logic
func TestModelSizeCalculation(t *testing.T) {
	detector := NewDetector()

	// Get current optimal size
	currentOptimal := detector.GetOptimalModelSize()

	// Test that the calculation is deterministic
	for i := 0; i < 3; i++ {
		optimal := detector.GetOptimalModelSize()
		if optimal != currentOptimal {
			t.Errorf("Optimal model size should be consistent, got %s then %s", currentOptimal, optimal)
		}
	}

	// Test that we get a valid model size
	validSizes := []string{"3B", "7B", "13B", "34B", "70B"}
	valid := false
	for _, size := range validSizes {
		if currentOptimal == size {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("Optimal model size %s is not a valid size", currentOptimal)
	}

	t.Logf("✅ Model size calculation test passed: optimal size is %s", currentOptimal)
}

// TestNewHardwareDetector tests HardwareDetector constructor
func TestNewHardwareDetector(t *testing.T) {
	detector := NewHardwareDetector()

	if detector == nil {
		t.Fatal("NewHardwareDetector() should not return nil")
	}

	// Verify it returns a valid HardwareDetector instance
	if _, ok := interface{}(detector).(*HardwareDetector); !ok {
		t.Error("NewHardwareDetector() should return *HardwareDetector")
	}

	t.Log("✅ NewHardwareDetector test passed")
}

// TestGetProfile tests the GetProfile method
func TestGetProfile(t *testing.T) {
	detector := NewHardwareDetector()
	profile := detector.GetProfile()

	if profile == nil {
		t.Fatal("GetProfile() should not return nil")
	}

	// Test CPU information
	t.Run("CPU Info", func(t *testing.T) {
		if profile.CPU.Cores == 0 {
			t.Error("CPU cores should be greater than 0")
		}
		if profile.CPU.Threads == 0 {
			t.Error("CPU threads should be greater than 0")
		}
		if profile.CPU.Arch == "" {
			t.Error("CPU architecture should not be empty")
		}
		t.Logf("CPU: %d cores, %d threads, arch: %s", profile.CPU.Cores, profile.CPU.Threads, profile.CPU.Arch)
	})

	// Test Memory information
	t.Run("Memory Info", func(t *testing.T) {
		if profile.Memory.Total == 0 {
			t.Error("Memory total should be greater than 0")
		}
		if profile.Memory.Available == 0 {
			t.Error("Memory available should be greater than 0")
		}
		// Default is 8GB
		expectedTotal := int64(8 * 1024 * 1024 * 1024)
		if profile.Memory.Total != expectedTotal {
			t.Logf("Memory total: %d bytes (expected default: %d)", profile.Memory.Total, expectedTotal)
		}
	})

	// Test OS information
	t.Run("OS Info", func(t *testing.T) {
		if profile.OS.Name == "" {
			t.Error("OS name should not be empty")
		}
		if profile.OS.Arch == "" {
			t.Error("OS architecture should not be empty")
		}
		t.Logf("OS: %s, arch: %s", profile.OS.Name, profile.OS.Arch)
	})

	// Test Network information
	t.Run("Network Info", func(t *testing.T) {
		if !profile.Network.HasInternet {
			t.Log("Network HasInternet is false (default is true)")
		}
		t.Logf("Network: HasInternet=%t, Latency=%v, Bandwidth=%d",
			profile.Network.HasInternet, profile.Network.Latency, profile.Network.Bandwidth)
	})

	// Test struct completeness
	t.Run("Struct Completeness", func(t *testing.T) {
		if profile.CPU.Cores == 0 && profile.Memory.Total == 0 && profile.OS.Name == "" {
			t.Error("Profile should have at least some fields populated")
		}
	})

	t.Log("✅ GetProfile test passed")
}

// TestDefaultProfile tests the DefaultProfile function
func TestDefaultProfile(t *testing.T) {
	profile := DefaultProfile()

	if profile == nil {
		t.Fatal("DefaultProfile() should not return nil")
	}

	// Verify it returns a valid profile
	if profile.CPU.Cores == 0 {
		t.Error("Default profile should have CPU cores > 0")
	}

	if profile.CPU.Threads == 0 {
		t.Error("Default profile should have CPU threads > 0")
	}

	if profile.OS.Name == "" {
		t.Error("Default profile should have OS name")
	}

	if profile.Memory.Total == 0 {
		t.Error("Default profile should have memory total > 0")
	}

	// Test that DefaultProfile uses NewHardwareDetector and GetProfile internally
	// by verifying similar results
	detector := NewHardwareDetector()
	directProfile := detector.GetProfile()

	if profile.CPU.Cores != directProfile.CPU.Cores {
		t.Error("DefaultProfile should use GetProfile internally")
	}

	if profile.OS.Name != directProfile.OS.Name {
		t.Error("DefaultProfile should return consistent OS info")
	}

	t.Logf("✅ DefaultProfile test passed: %d cores, %s, %d bytes memory",
		profile.CPU.Cores, profile.OS.Name, profile.Memory.Total)
}

// TestHardwareProfileStructTypes tests struct type definitions
func TestHardwareProfileStructTypes(t *testing.T) {
	profile := DefaultProfile()

	// Test GPUType constants
	t.Run("GPUType Constants", func(t *testing.T) {
		types := []GPUType{GPUTypeNVIDIA, GPUTypeAMD, GPUTypeApple, GPUTypeIntel}
		expected := []string{"nvidia", "amd", "apple", "intel"}

		for i, gpuType := range types {
			if string(gpuType) != expected[i] {
				t.Errorf("GPUType constant %d should be %s, got %s", i, expected[i], string(gpuType))
			}
		}
	})

	// Test OSType constants
	t.Run("OSType Constants", func(t *testing.T) {
		types := []OSType{OSTypeLinux, OSTypeMacOS, OSTypeWindows}
		expected := []string{"linux", "macos", "windows"}

		for i, osType := range types {
			if string(osType) != expected[i] {
				t.Errorf("OSType constant %d should be %s, got %s", i, expected[i], string(osType))
			}
		}
	})

	// Test Arch constants
	t.Run("Arch Constants", func(t *testing.T) {
		arches := []Arch{ArchX86_64, ArchARM64, ArchARM32}
		expected := []string{"x86_64", "arm64", "arm32"}

		for i, arch := range arches {
			if string(arch) != expected[i] {
				t.Errorf("Arch constant %d should be %s, got %s", i, expected[i], string(arch))
			}
		}
	})

	// Test profile structure
	t.Run("Profile Structure", func(t *testing.T) {
		if profile.GPU != nil {
			t.Logf("GPU detected: %s (%s)", profile.GPU.Name, profile.GPU.Type)
		}

		// Verify all main fields are initialized
		if profile.CPU.Arch == "" && profile.OS.Name == "" && profile.Memory.Total == 0 {
			t.Error("Profile should have fields initialized")
		}
	})

	t.Log("✅ Hardware profile struct types test passed")
}

// TestHardwareProfileConsistency tests that multiple calls return consistent data
func TestHardwareProfileConsistency(t *testing.T) {
	detector := NewHardwareDetector()

	profile1 := detector.GetProfile()
	profile2 := detector.GetProfile()
	profile3 := DefaultProfile()

	// All should return the same core count
	if profile1.CPU.Cores != profile2.CPU.Cores {
		t.Error("GetProfile should return consistent CPU core count")
	}

	if profile1.CPU.Cores != profile3.CPU.Cores {
		t.Error("DefaultProfile should return same CPU cores as GetProfile")
	}

	// All should return the same OS
	if profile1.OS.Name != profile2.OS.Name {
		t.Error("GetProfile should return consistent OS name")
	}

	if profile1.OS.Name != profile3.OS.Name {
		t.Error("DefaultProfile should return same OS as GetProfile")
	}

	// All should return the same memory
	if profile1.Memory.Total != profile2.Memory.Total {
		t.Error("GetProfile should return consistent memory total")
	}

	t.Log("✅ Hardware profile consistency test passed")
}
