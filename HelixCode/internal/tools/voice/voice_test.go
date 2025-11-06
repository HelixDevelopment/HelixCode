package voice

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDeviceManager_ListDevices tests device enumeration
func TestDeviceManager_ListDevices(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Fatalf("NewDeviceManager() error = %v", err)
	}

	devices, err := dm.ListDevices(context.Background())
	if err != nil {
		t.Fatalf("ListDevices() error = %v", err)
	}

	if len(devices) == 0 {
		t.Error("expected at least one device")
	}

	// Verify device properties
	for _, device := range devices {
		if device.ID == "" {
			t.Error("device ID should not be empty")
		}
		if device.Name == "" {
			t.Error("device name should not be empty")
		}
		if len(device.SampleRates) == 0 {
			t.Error("device should have at least one sample rate")
		}
	}
}

// TestDeviceManager_GetDefaultDevice tests getting the default device
func TestDeviceManager_GetDefaultDevice(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Fatalf("NewDeviceManager() error = %v", err)
	}

	device, err := dm.GetDefaultDevice()
	if err != nil {
		t.Fatalf("GetDefaultDevice() error = %v", err)
	}

	if device == nil {
		t.Fatal("expected non-nil device")
	}

	if !device.IsDefault && len(device.ID) == 0 {
		t.Error("expected device to be marked as default or have an ID")
	}
}

// TestDeviceManager_SelectDevice tests device selection
func TestDeviceManager_SelectDevice(t *testing.T) {
	dm, err := NewDeviceManager()
	if err != nil {
		t.Fatalf("NewDeviceManager() error = %v", err)
	}

	devices, err := dm.ListDevices(context.Background())
	if err != nil || len(devices) == 0 {
		t.Skip("no devices available for testing")
	}

	// Select first device
	err = dm.SelectDevice(devices[0].ID)
	if err != nil {
		t.Fatalf("SelectDevice() error = %v", err)
	}

	// Verify selection
	activeDevice := dm.GetActiveDevice()
	if activeDevice == nil {
		t.Fatal("expected active device to be set")
	}

	if activeDevice.ID != devices[0].ID {
		t.Errorf("expected device ID %s, got %s", devices[0].ID, activeDevice.ID)
	}
}

// TestAudioRecorder_StartStop tests recording start and stop
func TestAudioRecorder_StartStop(t *testing.T) {
	// Create temporary directory for recordings
	tmpDir, err := os.MkdirTemp("", "voice_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mock device
	device := &AudioDevice{
		ID:          "test-device",
		Name:        "Test Microphone",
		IsDefault:   true,
		SampleRates: []int{16000, 44100},
		Channels:    1,
		IsAvailable: true,
		Driver:      "Mock",
	}

	config := &AudioConfig{
		SampleRate:      16000,
		Channels:        1,
		BitDepth:        16,
		Format:          FormatWAV,
		OutputDirectory: tmpDir,
	}

	recorder, err := NewAudioRecorder(device, config)
	if err != nil {
		t.Fatalf("NewAudioRecorder() error = %v", err)
	}

	ctx := context.Background()

	// Test starting recording
	if err := recorder.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !recorder.IsRecording() {
		t.Error("expected recorder to be recording")
	}

	// Simulate recording time
	time.Sleep(100 * time.Millisecond)

	// Test stopping recording
	filePath, err := recorder.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if filePath == "" {
		t.Error("expected non-empty file path")
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("recorded file does not exist: %s", filePath)
	}

	// Verify file has content
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Error("recorded file is empty")
	}
}

// TestAudioRecorder_DoubleStart tests error handling for double start
func TestAudioRecorder_DoubleStart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "voice_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	device := &AudioDevice{
		ID:          "test-device",
		Name:        "Test Microphone",
		IsDefault:   true,
		SampleRates: []int{16000},
		Channels:    1,
		IsAvailable: true,
		Driver:      "Mock",
	}

	config := &AudioConfig{
		SampleRate:      16000,
		Channels:        1,
		BitDepth:        16,
		Format:          FormatWAV,
		OutputDirectory: tmpDir,
	}

	recorder, err := NewAudioRecorder(device, config)
	if err != nil {
		t.Fatalf("NewAudioRecorder() error = %v", err)
	}

	ctx := context.Background()

	// Start recording
	if err := recorder.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Try to start again - should fail
	err = recorder.Start(ctx)
	if err != ErrAlreadyRecording {
		t.Errorf("expected ErrAlreadyRecording, got %v", err)
	}

	// Clean up
	recorder.Stop(ctx)
}

// TestLevelMonitor tests audio level monitoring
func TestLevelMonitor(t *testing.T) {
	monitor := NewLevelMonitor(100*time.Millisecond, 10*time.Millisecond)

	// Generate test samples (sine wave)
	samples := generateSineWave(1000, 440.0, 16000)
	monitor.Update(samples)

	levels := monitor.GetLevels()

	// Peak and RMS are in dB, so they should be > -100 (not silence)
	if levels.Peak <= -100.0 {
		t.Errorf("expected peak level > -100 dB, got %.2f", levels.Peak)
	}

	if levels.RMS <= -100.0 {
		t.Errorf("expected RMS level > -100 dB, got %.2f", levels.RMS)
	}

	if levels.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// Test with silence - need to flush the buffer with more silence samples
	// since the buffer size is 1024 and still contains sine wave samples
	silence := make([]float64, 2000)
	monitor.Update(silence)

	silentLevels := monitor.GetLevels()
	// After flushing with enough silence, RMS should be very low (< -40 dB)
	if silentLevels.RMS > -40.0 {
		t.Errorf("expected silent levels (< -40 dB), got %.2f dB", silentLevels.RMS)
	}
}

// TestSilenceDetector tests silence detection
func TestSilenceDetector(t *testing.T) {
	detector := NewSilenceDetector(-40.0, 100*time.Millisecond)

	// Test with loud audio
	loudLevels := &AudioLevels{
		Peak:      -10.0,
		RMS:       -15.0,
		IsSilent:  false,
		Timestamp: time.Now(),
	}

	if detector.IsSilent(loudLevels) {
		t.Error("loud audio incorrectly detected as silent")
	}

	// Test with quiet audio
	quietLevels := &AudioLevels{
		Peak:      -50.0,
		RMS:       -55.0,
		IsSilent:  true,
		Timestamp: time.Now(),
	}

	// First check - not yet past minimum duration
	if detector.IsSilent(quietLevels) {
		t.Error("silence detected before minimum duration")
	}

	// Wait for minimum duration
	time.Sleep(150 * time.Millisecond)

	if !detector.IsSilent(quietLevels) {
		t.Error("silence not detected after minimum duration")
	}

	// Test reset
	detector.Reset()
	if detector.IsSilent(quietLevels) {
		t.Error("silence detected immediately after reset")
	}
}

// TestMockTranscriber tests the mock transcriber
func TestMockTranscriber(t *testing.T) {
	// Create temporary audio file
	tmpDir, err := os.MkdirTemp("", "voice_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	audioFile := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(audioFile, []byte("fake audio data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := &TranscriptionConfig{
		APIKey: "test-key",
		Model:  "whisper-1",
	}

	transcriber := NewMockTranscriber(config)

	ctx := context.Background()
	result, err := transcriber.TranscribeFile(ctx, audioFile)
	if err != nil {
		t.Fatalf("TranscribeFile() error = %v", err)
	}

	if result.Text == "" {
		t.Error("expected non-empty transcription text")
	}

	if result.Language == "" {
		t.Error("expected language to be detected")
	}

	if len(result.Segments) == 0 {
		t.Error("expected at least one segment")
	}
}

// TestMockTranscriber_CustomResponse tests custom mock responses
func TestMockTranscriber_CustomResponse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "voice_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	audioFile := filepath.Join(tmpDir, "custom.wav")
	if err := os.WriteFile(audioFile, []byte("fake audio data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := &TranscriptionConfig{
		APIKey: "test-key",
		Model:  "whisper-1",
	}

	transcriber := NewMockTranscriber(config)
	customText := "This is a custom transcription response"
	transcriber.SetMockResponse(audioFile, customText)

	ctx := context.Background()
	result, err := transcriber.TranscribeFile(ctx, audioFile)
	if err != nil {
		t.Fatalf("TranscribeFile() error = %v", err)
	}

	if result.Text != customText {
		t.Errorf("expected text %q, got %q", customText, result.Text)
	}
}

// TestVoiceInputManager_Integration tests the full integration
func TestVoiceInputManager_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "voice_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &VoiceConfig{
		SampleRate:       16000,
		Channels:         1,
		BitDepth:         16,
		Format:           FormatWAV,
		MaxDuration:      30 * time.Second,
		SilenceTimeout:   2 * time.Second,
		SilenceThreshold: -40.0,
		WhisperModel:     "whisper-1",
		AutoSelect:       true,
		OutputDirectory:  tmpDir,
	}

	manager, err := NewVoiceInputManager(config)
	if err != nil {
		t.Fatalf("NewVoiceInputManager() error = %v", err)
	}

	ctx := context.Background()

	// Test device listing
	devices, err := manager.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices() error = %v", err)
	}

	if len(devices) == 0 {
		t.Error("expected at least one device")
	}

	// Test recording
	if err := manager.StartRecording(ctx); err != nil {
		t.Fatalf("StartRecording() error = %v", err)
	}

	if !manager.GetRecorderStatus() {
		t.Error("expected recorder to be active")
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Get audio levels
	levels := manager.GetAudioLevels()
	if levels == nil {
		t.Error("expected non-nil audio levels")
	}

	// Stop recording
	audioPath, err := manager.StopRecording(ctx)
	if err != nil {
		t.Fatalf("StopRecording() error = %v", err)
	}

	if audioPath == "" {
		t.Error("expected non-empty audio path")
	}

	// Verify file exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Errorf("audio file does not exist: %s", audioPath)
	}
}

// TestVoiceError tests error wrapping
func TestVoiceError(t *testing.T) {
	originalErr := ErrDeviceNotFound
	wrappedErr := NewVoiceError("SelectDevice", originalErr, "device-123")

	if wrappedErr == nil {
		t.Fatal("expected non-nil error")
	}

	errorMsg := wrappedErr.Error()
	if errorMsg == "" {
		t.Error("expected non-empty error message")
	}

	// Check if original error is preserved
	var voiceErr *VoiceError
	if !os.IsExist(wrappedErr) {
		// Try to unwrap
		if ve, ok := wrappedErr.(*VoiceError); ok {
			voiceErr = ve
		}
	}

	if voiceErr != nil && voiceErr.Unwrap() != originalErr {
		t.Error("expected wrapped error to preserve original error")
	}
}

// Helper function to generate sine wave samples
func generateSineWave(samples int, frequency, sampleRate float64) []float64 {
	wave := make([]float64, samples)
	for i := range wave {
		t := float64(i) / sampleRate
		wave[i] = math.Sin(2 * math.Pi * frequency * t)
	}
	return wave
}
