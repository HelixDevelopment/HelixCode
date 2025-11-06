package voice

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AudioFormat represents audio file format
type AudioFormat string

const (
	FormatWAV AudioFormat = "wav"
	FormatMP3 AudioFormat = "mp3"
)

// AudioConfig contains audio recording configuration
type AudioConfig struct {
	SampleRate      int
	Channels        int
	BitDepth        int
	Format          AudioFormat
	OutputDirectory string
}

// AudioLevels contains real-time audio level information
type AudioLevels struct {
	Peak      float64   // Peak level in dB
	RMS       float64   // RMS level in dB
	IsSilent  bool      // Whether current audio is silent
	Timestamp time.Time // Timestamp of measurement
}

// AudioRecorder handles microphone input and recording
type AudioRecorder struct {
	device          *AudioDevice
	config          *AudioConfig
	recording       bool
	currentFile     string
	levelMonitor    *LevelMonitor
	silenceDetector *SilenceDetector
	samples         []int16
	mu              sync.RWMutex
	mockMode        bool
}

// NewAudioRecorder creates a new audio recorder
func NewAudioRecorder(device *AudioDevice, config *AudioConfig) (*AudioRecorder, error) {
	if device == nil {
		return nil, fmt.Errorf("device cannot be nil")
	}

	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create output directory if it doesn't exist
	if config.OutputDirectory != "" {
		if err := os.MkdirAll(config.OutputDirectory, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	return &AudioRecorder{
		device:          device,
		config:          config,
		recording:       false,
		levelMonitor:    NewLevelMonitor(100*time.Millisecond, 10*time.Millisecond),
		silenceDetector: NewSilenceDetector(-40.0, 2*time.Second),
		samples:         make([]int16, 0),
		mockMode:        device.Driver == "Mock",
	}, nil
}

// Start begins audio capture
func (a *AudioRecorder) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.recording {
		return ErrAlreadyRecording
	}

	// Generate output filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("recording_%s.%s", timestamp, a.config.Format)
	if a.config.OutputDirectory != "" {
		a.currentFile = filepath.Join(a.config.OutputDirectory, filename)
	} else {
		a.currentFile = filename
	}

	// Reset sample buffer
	a.samples = make([]int16, 0)
	a.recording = true

	// Start mock recording if in mock mode
	if a.mockMode {
		go a.recordMockAudio(ctx)
	} else {
		// In a real implementation, this would start the audio capture
		// using platform-specific APIs (CoreAudio, ALSA, WASAPI)
		go a.recordRealAudio(ctx)
	}

	return nil
}

// Stop ends audio capture and finalizes the file
func (a *AudioRecorder) Stop(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.recording {
		return "", ErrNotRecording
	}

	a.recording = false

	// Write audio file
	if err := a.writeAudioFile(); err != nil {
		return "", fmt.Errorf("failed to write audio file: %w", err)
	}

	return a.currentFile, nil
}

// IsRecording returns true if currently recording
func (a *AudioRecorder) IsRecording() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.recording
}

// GetLevels returns current audio levels
func (a *AudioRecorder) GetLevels() *AudioLevels {
	return a.levelMonitor.GetLevels()
}

// SetDevice changes the active recording device
func (a *AudioRecorder) SetDevice(device *AudioDevice) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.recording {
		return fmt.Errorf("cannot change device while recording")
	}

	if device == nil {
		return fmt.Errorf("device cannot be nil")
	}

	a.device = device
	a.mockMode = device.Driver == "Mock"

	return nil
}

// recordMockAudio generates mock audio data for testing
func (a *AudioRecorder) recordMockAudio(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	sampleCount := 0
	frequency := 440.0 // A4 note

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.RLock()
			if !a.recording {
				a.mu.RUnlock()
				return
			}
			a.mu.RUnlock()

			// Generate sine wave samples
			samplesPerTick := int(float64(a.config.SampleRate) * 0.01) // 10ms worth
			newSamples := make([]int16, samplesPerTick)

			for i := 0; i < samplesPerTick; i++ {
				t := float64(sampleCount+i) / float64(a.config.SampleRate)
				sample := math.Sin(2 * math.Pi * frequency * t)

				// Add some amplitude variation to simulate voice
				amplitude := 0.3 + 0.2*math.Sin(2*math.Pi*0.5*t)
				sample *= amplitude

				// Convert to 16-bit PCM
				newSamples[i] = int16(sample * 32767)
			}

			a.mu.Lock()
			a.samples = append(a.samples, newSamples...)
			a.mu.Unlock()

			// Update level monitor
			floatSamples := make([]float64, len(newSamples))
			for i, s := range newSamples {
				floatSamples[i] = float64(s) / 32768.0
			}
			a.levelMonitor.Update(floatSamples)

			sampleCount += samplesPerTick
		}
	}
}

// recordRealAudio would capture real audio from the device
func (a *AudioRecorder) recordRealAudio(ctx context.Context) {
	// In a real implementation, this would interface with platform-specific
	// audio APIs (CoreAudio, ALSA, WASAPI) to capture real audio
	// For now, fall back to mock audio
	a.recordMockAudio(ctx)
}

// writeAudioFile writes the recorded samples to a WAV file
func (a *AudioRecorder) writeAudioFile() error {
	file, err := os.Create(a.currentFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// WAV file format implementation
	numSamples := len(a.samples)
	dataSize := numSamples * 2 // 16-bit samples

	// Write RIFF header
	file.WriteString("RIFF")
	binary.Write(file, binary.LittleEndian, uint32(36+dataSize))
	file.WriteString("WAVE")

	// Write fmt chunk
	file.WriteString("fmt ")
	binary.Write(file, binary.LittleEndian, uint32(16))           // Chunk size
	binary.Write(file, binary.LittleEndian, uint16(1))            // PCM format
	binary.Write(file, binary.LittleEndian, uint16(a.config.Channels))
	binary.Write(file, binary.LittleEndian, uint32(a.config.SampleRate))
	binary.Write(file, binary.LittleEndian, uint32(a.config.SampleRate*a.config.Channels*2))
	binary.Write(file, binary.LittleEndian, uint16(a.config.Channels*2))
	binary.Write(file, binary.LittleEndian, uint16(16))          // Bits per sample

	// Write data chunk
	file.WriteString("data")
	binary.Write(file, binary.LittleEndian, uint32(dataSize))

	for _, sample := range a.samples {
		binary.Write(file, binary.LittleEndian, sample)
	}

	return nil
}

// LevelMonitor tracks real-time audio levels
type LevelMonitor struct {
	buffer      []float64
	bufferSize  int
	windowSize  time.Duration
	updateRate  time.Duration
	currentPeak float64
	currentRMS  float64
	mu          sync.RWMutex
}

// NewLevelMonitor creates a new level monitor
func NewLevelMonitor(windowSize, updateRate time.Duration) *LevelMonitor {
	return &LevelMonitor{
		buffer:      make([]float64, 0),
		bufferSize:  1024,
		windowSize:  windowSize,
		updateRate:  updateRate,
		currentPeak: -100.0,
		currentRMS:  -100.0,
	}
}

// Update adds new samples to the monitor
func (l *LevelMonitor) Update(samples []float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Add samples to buffer
	l.buffer = append(l.buffer, samples...)

	// Limit buffer size
	if len(l.buffer) > l.bufferSize {
		l.buffer = l.buffer[len(l.buffer)-l.bufferSize:]
	}

	// Calculate levels
	if len(l.buffer) > 0 {
		l.calculateLevels()
	}
}

// calculateLevels calculates peak and RMS levels
func (l *LevelMonitor) calculateLevels() {
	if len(l.buffer) == 0 {
		return
	}

	// Calculate peak
	peak := 0.0
	for _, sample := range l.buffer {
		abs := math.Abs(sample)
		if abs > peak {
			peak = abs
		}
	}

	// Calculate RMS
	sumSquares := 0.0
	for _, sample := range l.buffer {
		sumSquares += sample * sample
	}
	rms := math.Sqrt(sumSquares / float64(len(l.buffer)))

	// Convert to dB
	l.currentPeak = amplitudeToDb(peak)
	l.currentRMS = amplitudeToDb(rms)
}

// GetLevels returns current audio levels
func (l *LevelMonitor) GetLevels() *AudioLevels {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &AudioLevels{
		Peak:      l.currentPeak,
		RMS:       l.currentRMS,
		IsSilent:  l.currentRMS < -40.0,
		Timestamp: time.Now(),
	}
}

// SilenceDetector identifies periods of silence
type SilenceDetector struct {
	threshold    float64
	minDuration  time.Duration
	silenceStart time.Time
	isSilent     bool
	mu           sync.RWMutex
}

// NewSilenceDetector creates a new silence detector
func NewSilenceDetector(threshold float64, minDuration time.Duration) *SilenceDetector {
	return &SilenceDetector{
		threshold:   threshold,
		minDuration: minDuration,
		isSilent:    false,
	}
}

// IsSilent checks if current audio is below threshold
func (s *SilenceDetector) IsSilent(levels *AudioLevels) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	isCurrentlySilent := levels.RMS < s.threshold

	if isCurrentlySilent && !s.isSilent {
		// Started being silent
		s.silenceStart = time.Now()
		s.isSilent = true
	} else if !isCurrentlySilent && s.isSilent {
		// No longer silent
		s.isSilent = false
		s.silenceStart = time.Time{}
	}

	return s.isSilent && time.Since(s.silenceStart) >= s.minDuration
}

// SilenceDuration returns how long silence has persisted
func (s *SilenceDetector) SilenceDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isSilent {
		return 0
	}

	return time.Since(s.silenceStart)
}

// Reset resets the silence detector state
func (s *SilenceDetector) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isSilent = false
	s.silenceStart = time.Time{}
}

// amplitudeToDb converts amplitude to decibels
func amplitudeToDb(amplitude float64) float64 {
	if amplitude <= 0 {
		return -100.0 // Silence threshold
	}
	return 20 * math.Log10(amplitude)
}
