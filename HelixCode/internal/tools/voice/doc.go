// Package voice provides voice-to-code functionality for HelixCode.
//
// This package enables users to provide input through speech rather than typing,
// integrating audio capture, device management, and speech-to-text transcription
// using OpenAI's Whisper API.
//
// # Features
//
//   - Audio recording from microphone with configurable sample rates and formats
//   - Device selection and enumeration across platforms (macOS, Linux, Windows)
//   - Whisper transcription integration with OpenAI API
//   - Language support with auto-detection or explicit specification
//   - Volume level detection and real-time monitoring
//   - WAV/MP3 format support
//   - Silence detection for automatic recording termination
//   - Mock audio support for testing without hardware
//
// # Basic Usage
//
// Create a voice input manager and start recording:
//
//	config := &voice.VoiceConfig{
//	    SampleRate:      16000,
//	    Channels:        1,
//	    Format:          voice.FormatWAV,
//	    SilenceTimeout:  2 * time.Second,
//	    APIKey:          os.Getenv("OPENAI_API_KEY"),
//	    AutoSelect:      true,
//	}
//
//	manager, err := voice.NewVoiceInputManager(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start recording
//	ctx := context.Background()
//	if err := manager.StartRecording(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Wait for user to speak...
//	time.Sleep(5 * time.Second)
//
//	// Stop and get the audio file
//	audioPath, err := manager.StopRecording(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Transcribe the recording
//	text, err := manager.TranscribeRecording(ctx, audioPath)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Transcribed: %s\n", text)
//
// # One-Shot Recording and Transcription
//
// For convenience, you can record and transcribe in a single operation:
//
//	text, err := manager.RecordAndTranscribe(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("You said: %s\n", text)
//
// This method automatically stops recording after detecting silence
// for the configured duration (default: 2 seconds).
//
// # Device Management
//
// List and select audio input devices:
//
//	// List all available devices
//	devices, err := manager.ListDevices(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, device := range devices {
//	    fmt.Printf("Device: %s (%s)\n", device.Name, device.ID)
//	    fmt.Printf("  Default: %v, Available: %v\n",
//	        device.IsDefault, device.IsAvailable)
//	    fmt.Printf("  Sample rates: %v\n", device.SampleRates)
//	}
//
//	// Select a specific device
//	if err := manager.SelectDevice(ctx, "device-id"); err != nil {
//	    log.Fatal(err)
//	}
//
// # Audio Level Monitoring
//
// Monitor real-time audio levels during recording:
//
//	if err := manager.StartRecording(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	ticker := time.NewTicker(100 * time.Millisecond)
//	defer ticker.Stop()
//
//	for {
//	    select {
//	    case <-ticker.C:
//	        levels := manager.GetAudioLevels()
//	        fmt.Printf("Peak: %.1f dB, RMS: %.1f dB, Silent: %v\n",
//	            levels.Peak, levels.RMS, levels.IsSilent)
//	    }
//	}
//
// # Testing with Mock Audio
//
// The package includes mock implementations for testing without audio hardware:
//
//	// Create a mock transcriber
//	mockTranscriber := voice.NewMockTranscriber(&voice.TranscriptionConfig{
//	    APIKey: "test-key",
//	    Model:  "whisper-1",
//	})
//
//	// Set custom responses for specific files
//	mockTranscriber.SetMockResponse("/path/to/test.wav",
//	    "This is a test transcription")
//
//	// Use the mock transcriber
//	result, err := mockTranscriber.TranscribeFile(ctx, "/path/to/test.wav")
//
// Mock audio devices are automatically created when real audio hardware
// is not available, enabling tests to run in CI/CD environments.
//
// # Error Handling
//
// The package defines specific error types for different failure scenarios:
//
//   - ErrNoDevicesFound: No audio input devices available
//   - ErrDeviceNotFound: Specified device does not exist
//   - ErrAlreadyRecording: Recording already in progress
//   - ErrNotRecording: No recording to stop
//   - ErrTranscriptionFailed: Whisper API transcription failed
//   - ErrInvalidAPIKey: OpenAI API key is invalid or missing
//   - ErrFileTooLarge: Audio file exceeds 25MB Whisper API limit
//
// All errors can be checked using errors.Is():
//
//	if errors.Is(err, voice.ErrAlreadyRecording) {
//	    // Handle duplicate recording attempt
//	}
//
// # Platform Support
//
// The package supports audio capture on multiple platforms:
//
//   - macOS: CoreAudio driver
//   - Linux: ALSA/PulseAudio driver
//   - Windows: WASAPI driver
//   - Other: Mock driver for testing
//
// Platform-specific audio APIs are used when available, with automatic
// fallback to mock implementations for unsupported platforms or testing.
//
// # Configuration
//
// VoiceConfig provides comprehensive configuration options:
//
//	type VoiceConfig struct {
//	    // Audio settings
//	    SampleRate int         // Default: 16000 Hz (Whisper requirement)
//	    Channels   int         // Default: 1 (mono)
//	    BitDepth   int         // Default: 16
//	    Format     AudioFormat // WAV or MP3
//
//	    // Recording settings
//	    MaxDuration      time.Duration // Default: 5 minutes
//	    SilenceTimeout   time.Duration // Default: 2 seconds
//	    SilenceThreshold float64       // Default: -40.0 dB
//
//	    // Transcription settings
//	    WhisperModel string  // Default: "whisper-1"
//	    Language     string  // Optional, auto-detect if empty
//	    Prompt       string  // Optional context for transcription
//	    Temperature  float64 // Default: 0.0 (deterministic)
//	    APIKey       string  // OpenAI API key (required for transcription)
//
//	    // Device settings
//	    DefaultDevice string // Device ID or name
//	    AutoSelect    bool   // Auto-select default device
//
//	    // Output settings
//	    OutputDirectory string // Default: ~/.helix/voice/recordings
//	    BaseURL         string // Optional custom Whisper API endpoint
//	}
//
// # Thread Safety
//
// All components in this package are thread-safe and can be safely used
// from multiple goroutines. Internal state is protected by mutexes.
//
// # Performance Considerations
//
// - Audio is buffered in memory during recording
// - Level monitoring runs in a separate goroutine with configurable update rate
// - Whisper API calls have a 30-second timeout
// - Audio files larger than 25MB are rejected before upload
// - Mock audio generates sine waves at 440Hz (A4 note) for testing
//
// # References
//
//   - OpenAI Whisper API: https://platform.openai.com/docs/guides/speech-to-text
//   - WAV file format: http://soundfile.sapp.org/doc/WaveFormat/
//   - Audio level measurement: https://en.wikipedia.org/wiki/DBFS
package voice
