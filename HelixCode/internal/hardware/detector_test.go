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
