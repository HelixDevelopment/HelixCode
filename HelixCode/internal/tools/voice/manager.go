package voice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// VoiceConfig contains configuration for voice input
type VoiceConfig struct {
	// Audio settings
	SampleRate int         // Default: 16000 Hz
	Channels   int         // Default: 1 (mono)
	BitDepth   int         // Default: 16
	Format     AudioFormat // WAV or MP3

	// Recording settings
	MaxDuration      time.Duration // Default: 5 minutes
	SilenceTimeout   time.Duration // Default: 2 seconds
	SilenceThreshold float64       // dB threshold, default: -40.0

	// Transcription settings
	WhisperModel string  // Default: "whisper-1"
	Language     string  // Optional, auto-detect if empty
	Prompt       string  // Optional context for transcription
	Temperature  float64 // Default: 0.0 (deterministic)

	// Device settings
	DefaultDevice string // Device ID or name
	AutoSelect    bool   // Auto-select default device

	// Output settings
	OutputDirectory string // Directory for audio files
	APIKey          string // OpenAI API key
	BaseURL         string // Optional custom API endpoint
}

// VoiceInputManager orchestrates voice input operations
type VoiceInputManager struct {
	recorder    *AudioRecorder
	devices     *DeviceManager
	transcriber *Transcriber
	config      *VoiceConfig
}

// NewVoiceInputManager creates a new voice input manager
func NewVoiceInputManager(config *VoiceConfig) (*VoiceInputManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set defaults
	if config.SampleRate == 0 {
		config.SampleRate = 16000
	}
	if config.Channels == 0 {
		config.Channels = 1
	}
	if config.BitDepth == 0 {
		config.BitDepth = 16
	}
	if config.Format == "" {
		config.Format = FormatWAV
	}
	if config.MaxDuration == 0 {
		config.MaxDuration = 5 * time.Minute
	}
	if config.SilenceTimeout == 0 {
		config.SilenceTimeout = 2 * time.Second
	}
	if config.SilenceThreshold == 0 {
		config.SilenceThreshold = -40.0
	}
	if config.WhisperModel == "" {
		config.WhisperModel = "whisper-1"
	}
	if config.OutputDirectory == "" {
		homeDir, _ := os.UserHomeDir()
		config.OutputDirectory = filepath.Join(homeDir, ".helix", "voice", "recordings")
	}

	// Initialize device manager
	deviceManager, err := NewDeviceManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create device manager: %w", err)
	}

	// Select device
	var device *AudioDevice
	if config.DefaultDevice != "" {
		device, err = deviceManager.GetDevice(config.DefaultDevice)
		if err != nil {
			return nil, fmt.Errorf("failed to get specified device: %w", err)
		}
	} else if config.AutoSelect {
		device, err = deviceManager.GetDefaultDevice()
		if err != nil {
			return nil, fmt.Errorf("failed to get default device: %w", err)
		}
	} else {
		device, err = deviceManager.GetDefaultDevice()
		if err != nil {
			return nil, fmt.Errorf("failed to get default device: %w", err)
		}
	}

	// Create audio config
	audioConfig := &AudioConfig{
		SampleRate:      config.SampleRate,
		Channels:        config.Channels,
		BitDepth:        config.BitDepth,
		Format:          config.Format,
		OutputDirectory: config.OutputDirectory,
	}

	// Initialize recorder
	recorder, err := NewAudioRecorder(device, audioConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio recorder: %w", err)
	}

	// Initialize transcriber if API key is provided
	var transcriber *Transcriber
	if config.APIKey != "" {
		transcriptionConfig := &TranscriptionConfig{
			APIKey:      config.APIKey,
			Model:       config.WhisperModel,
			Language:    config.Language,
			Prompt:      config.Prompt,
			Temperature: config.Temperature,
			BaseURL:     config.BaseURL,
		}

		transcriber, err = NewTranscriber(transcriptionConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create transcriber: %w", err)
		}
	}

	return &VoiceInputManager{
		recorder:    recorder,
		devices:     deviceManager,
		transcriber: transcriber,
		config:      config,
	}, nil
}

// ListDevices returns available audio input devices
func (v *VoiceInputManager) ListDevices(ctx context.Context) ([]AudioDevice, error) {
	return v.devices.ListDevices(ctx)
}

// SelectDevice sets the active audio input device
func (v *VoiceInputManager) SelectDevice(ctx context.Context, deviceID string) error {
	device, err := v.devices.GetDevice(deviceID)
	if err != nil {
		return err
	}

	if err := v.recorder.SetDevice(device); err != nil {
		return fmt.Errorf("failed to set device on recorder: %w", err)
	}

	if err := v.devices.SelectDevice(deviceID); err != nil {
		return fmt.Errorf("failed to select device: %w", err)
	}

	return nil
}

// StartRecording begins audio capture
func (v *VoiceInputManager) StartRecording(ctx context.Context) error {
	if v.recorder.IsRecording() {
		return ErrAlreadyRecording
	}

	return v.recorder.Start(ctx)
}

// StopRecording ends audio capture and returns the file path
func (v *VoiceInputManager) StopRecording(ctx context.Context) (string, error) {
	if !v.recorder.IsRecording() {
		return "", ErrNotRecording
	}

	return v.recorder.Stop(ctx)
}

// GetAudioLevels returns real-time audio level information
func (v *VoiceInputManager) GetAudioLevels() *AudioLevels {
	return v.recorder.GetLevels()
}

// TranscribeRecording transcribes the most recent recording
func (v *VoiceInputManager) TranscribeRecording(ctx context.Context, audioPath string) (string, error) {
	if v.transcriber == nil {
		return "", fmt.Errorf("transcriber not initialized (API key not provided)")
	}

	result, err := v.transcriber.TranscribeFile(ctx, audioPath)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}

	return result.Text, nil
}

// RecordAndTranscribe performs recording and transcription in one operation
func (v *VoiceInputManager) RecordAndTranscribe(ctx context.Context) (string, error) {
	// Start recording
	if err := v.StartRecording(ctx); err != nil {
		return "", fmt.Errorf("failed to start recording: %w", err)
	}

	// Create a timeout context for max duration
	timeoutCtx, cancel := context.WithTimeout(ctx, v.config.MaxDuration)
	defer cancel()

	// Monitor for silence or timeout
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			// Timeout or cancellation
			audioPath, err := v.StopRecording(ctx)
			if err != nil {
				return "", fmt.Errorf("failed to stop recording: %w", err)
			}

			// Transcribe if we have audio
			if audioPath != "" && v.transcriber != nil {
				return v.TranscribeRecording(ctx, audioPath)
			}

			return "", fmt.Errorf("recording stopped without transcription")

		case <-ticker.C:
			// Check for silence
			levels := v.GetAudioLevels()
			if levels.IsSilent {
				// Check if silence duration exceeds threshold
				silenceDuration := v.recorder.silenceDetector.SilenceDuration()
				if silenceDuration >= v.config.SilenceTimeout {
					// Auto-stop on silence
					audioPath, err := v.StopRecording(ctx)
					if err != nil {
						return "", fmt.Errorf("failed to stop recording: %w", err)
					}

					// Transcribe
					if audioPath != "" && v.transcriber != nil {
						return v.TranscribeRecording(ctx, audioPath)
					}

					return "", nil
				}
			}
		}
	}
}

// GetRecorderStatus returns the current recording status
func (v *VoiceInputManager) GetRecorderStatus() bool {
	return v.recorder.IsRecording()
}

// GetActiveDevice returns the currently selected device
func (v *VoiceInputManager) GetActiveDevice() *AudioDevice {
	return v.devices.GetActiveDevice()
}

// SetTranscriber sets a custom transcriber (useful for testing)
func (v *VoiceInputManager) SetTranscriber(transcriber *Transcriber) {
	v.transcriber = transcriber
}
