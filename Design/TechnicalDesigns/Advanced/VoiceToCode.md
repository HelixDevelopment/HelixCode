# Voice to Code - Technical Design

## Overview

Voice-to-code functionality enables users to provide input through speech rather than typing. This feature integrates audio capture, device management, and speech-to-text transcription using OpenAI's Whisper API.

**References:**
- Aider's voice.py implementation
- OpenAI Whisper API documentation

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Voice System                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────────┐
        │         VoiceInputManager               │
        │  - Start/Stop Recording                 │
        │  - Device Selection                     │
        │  - Transcription Orchestration          │
        └─────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐     ┌──────────────┐
│AudioRecorder │    │DeviceManager │     │ Transcriber  │
│              │    │              │     │              │
│ • Capture    │    │ • List       │     │ • Whisper    │
│ • Format     │    │ • Select     │     │ • Language   │
│ • Levels     │    │ • Default    │     │ • Stream     │
│ • Silence    │    │ • Validate   │     │ • Batch      │
└──────────────┘    └──────────────┘     └──────────────┘
        │                                         │
        ▼                                         ▼
┌──────────────┐                        ┌──────────────┐
│ Audio File   │───────────────────────▶│ Transcribed  │
│ (WAV/MP3)    │                        │ Text Output  │
└──────────────┘                        └──────────────┘
```

---

## Component Interfaces

### VoiceInputManager

```go
package voice

import (
    "context"
    "io"
    "time"
)

// VoiceInputManager orchestrates voice input operations
type VoiceInputManager struct {
    recorder    *AudioRecorder
    devices     *DeviceManager
    transcriber *Transcriber
    config      *VoiceConfig
}

// VoiceConfig contains configuration for voice input
type VoiceConfig struct {
    // Audio settings
    SampleRate      int           // Default: 16000 Hz
    Channels        int           // Default: 1 (mono)
    BitDepth        int           // Default: 16
    Format          AudioFormat   // WAV or MP3

    // Recording settings
    MaxDuration     time.Duration // Default: 5 minutes
    SilenceTimeout  time.Duration // Default: 2 seconds
    SilenceThreshold float64      // dB threshold, default: -40.0

    // Transcription settings
    WhisperModel    string        // Default: "whisper-1"
    Language        string        // Optional, auto-detect if empty
    Prompt          string        // Optional context for transcription
    Temperature     float64       // Default: 0.0 (deterministic)

    // Device settings
    DefaultDevice   string        // Device ID or name
    AutoSelect      bool          // Auto-select default device
}

// NewVoiceInputManager creates a new voice input manager
func NewVoiceInputManager(config *VoiceConfig) (*VoiceInputManager, error)

// ListDevices returns available audio input devices
func (v *VoiceInputManager) ListDevices(ctx context.Context) ([]AudioDevice, error)

// SelectDevice sets the active audio input device
func (v *VoiceInputManager) SelectDevice(ctx context.Context, deviceID string) error

// StartRecording begins audio capture
func (v *VoiceInputManager) StartRecording(ctx context.Context) error

// StopRecording ends audio capture and returns the file path
func (v *VoiceInputManager) StopRecording(ctx context.Context) (string, error)

// GetAudioLevels returns real-time audio level information
func (v *VoiceInputManager) GetAudioLevels() *AudioLevels

// TranscribeRecording transcribes the most recent recording
func (v *VoiceInputManager) TranscribeRecording(ctx context.Context, audioPath string) (string, error)

// RecordAndTranscribe performs recording and transcription in one operation
func (v *VoiceInputManager) RecordAndTranscribe(ctx context.Context) (string, error)
```

### AudioRecorder

```go
package voice

import (
    "context"
    "time"
)

// AudioFormat represents audio file format
type AudioFormat string

const (
    FormatWAV AudioFormat = "wav"
    FormatMP3 AudioFormat = "mp3"
)

// AudioRecorder handles microphone input and recording
type AudioRecorder struct {
    device          *AudioDevice
    config          *AudioConfig
    recording       bool
    currentFile     string
    levelMonitor    *LevelMonitor
    silenceDetector *SilenceDetector
}

// AudioConfig contains audio recording configuration
type AudioConfig struct {
    SampleRate       int
    Channels         int
    BitDepth         int
    Format           AudioFormat
    OutputDirectory  string
}

// AudioLevels contains real-time audio level information
type AudioLevels struct {
    Peak      float64   // Peak level in dB
    RMS       float64   // RMS level in dB
    IsSilent  bool      // Whether current audio is silent
    Timestamp time.Time // Timestamp of measurement
}

// NewAudioRecorder creates a new audio recorder
func NewAudioRecorder(device *AudioDevice, config *AudioConfig) (*AudioRecorder, error)

// Start begins audio capture
func (a *AudioRecorder) Start(ctx context.Context) error

// Stop ends audio capture and finalizes the file
func (a *AudioRecorder) Stop(ctx context.Context) (string, error)

// IsRecording returns true if currently recording
func (a *AudioRecorder) IsRecording() bool

// GetLevels returns current audio levels
func (a *AudioRecorder) GetLevels() *AudioLevels

// SetDevice changes the active recording device
func (a *AudioRecorder) SetDevice(device *AudioDevice) error
```

### DeviceManager

```go
package voice

import (
    "context"
)

// AudioDevice represents an audio input device
type AudioDevice struct {
    ID           string   // System device ID
    Name         string   // Human-readable name
    IsDefault    bool     // Whether this is the system default
    SampleRates  []int    // Supported sample rates
    Channels     int      // Number of channels
    IsAvailable  bool     // Current availability status
    Driver       string   // Audio driver (CoreAudio, ALSA, etc.)
}

// DeviceManager handles audio device enumeration and selection
type DeviceManager struct {
    devices       []AudioDevice
    activeDevice  *AudioDevice
    refreshInterval time.Duration
}

// NewDeviceManager creates a new device manager
func NewDeviceManager() (*DeviceManager, error)

// ListDevices enumerates all available audio input devices
func (d *DeviceManager) ListDevices(ctx context.Context) ([]AudioDevice, error)

// GetDevice retrieves a specific device by ID
func (d *DeviceManager) GetDevice(deviceID string) (*AudioDevice, error)

// GetDefaultDevice returns the system default input device
func (d *DeviceManager) GetDefaultDevice() (*AudioDevice, error)

// SelectDevice sets the active device for recording
func (d *DeviceManager) SelectDevice(deviceID string) error

// GetActiveDevice returns the currently selected device
func (d *DeviceManager) GetActiveDevice() *AudioDevice

// RefreshDevices updates the device list
func (d *DeviceManager) RefreshDevices(ctx context.Context) error

// ValidateDevice checks if a device is available and properly configured
func (d *DeviceManager) ValidateDevice(device *AudioDevice) error
```

### Transcriber

```go
package voice

import (
    "context"
    "io"
)

// Transcriber handles speech-to-text conversion via Whisper API
type Transcriber struct {
    client  *WhisperClient
    config  *TranscriptionConfig
}

// TranscriptionConfig contains transcription settings
type TranscriptionConfig struct {
    APIKey      string  // OpenAI API key
    Model       string  // Whisper model version
    Language    string  // Optional language code (e.g., "en", "es")
    Prompt      string  // Optional context prompt
    Temperature float64 // Sampling temperature (0.0 - 1.0)
    BaseURL     string  // Optional custom API endpoint
}

// TranscriptionResult contains the transcription output
type TranscriptionResult struct {
    Text      string        // Transcribed text
    Language  string        // Detected language
    Duration  float64       // Audio duration in seconds
    Segments  []Segment     // Optional word-level segments
    Metadata  *Metadata     // Additional metadata
}

// Segment represents a timestamped segment of transcription
type Segment struct {
    ID               int
    Start            float64
    End              float64
    Text             string
    AvgLogProb       float64
    CompressionRatio float64
    NoSpeechProb     float64
}

// Metadata contains additional transcription information
type Metadata struct {
    Model         string
    RequestID     string
    ProcessingTime float64
}

// NewTranscriber creates a new transcriber
func NewTranscriber(config *TranscriptionConfig) (*Transcriber, error)

// TranscribeFile transcribes an audio file
func (t *Transcriber) TranscribeFile(ctx context.Context, filePath string) (*TranscriptionResult, error)

// TranscribeStream transcribes audio from a stream
func (t *Transcriber) TranscribeStream(ctx context.Context, reader io.Reader, format AudioFormat) (*TranscriptionResult, error)

// ValidateAudioFile checks if the file is suitable for transcription
func (t *Transcriber) ValidateAudioFile(filePath string) error
```

---

## State Machine

### Recording State Machine

```
                    ┌─────────┐
                    │  IDLE   │
                    └────┬────┘
                         │
                  Start()│
                         │
                         ▼
                    ┌─────────┐
            ┌──────▶│RECORDING│◀──────┐
            │       └────┬────┘       │
            │            │            │
            │   Stop()   │  Continue  │
            │   Timeout  │            │
            │   Silence  │   Active   │
            │            │   Audio    │
            │            ▼            │
            │       ┌─────────┐       │
            │       │ MONITOR │───────┘
            │       └────┬────┘
            │            │
            │   Silence  │
            │   Detected │
            │            ▼
            │       ┌─────────┐
            │       │STOPPING │
            │       └────┬────┘
            │            │
            │            ▼
            │       ┌─────────┐
            └───────│FINALIZING│
                    └────┬────┘
                         │
                         ▼
                    ┌─────────┐
                    │  DONE   │
                    └─────────┘
```

### Transcription State Machine

```
                    ┌─────────┐
                    │ PENDING │
                    └────┬────┘
                         │
                Transcribe()
                         │
                         ▼
                    ┌─────────┐
                    │UPLOADING│
                    └────┬────┘
                         │
                         ▼
                    ┌─────────┐
            ┌──────▶│PROCESSING│
            │       └────┬────┘
            │            │
            │   Error    │   Success
            │            │
  Retry (if │            ▼
  attempts  │       ┌─────────┐
  remain)   │       │ SUCCESS │
            │       └─────────┘
            │
            │
            │       ┌─────────┐
            └───────│  ERROR  │
                    └─────────┘
```

---

## Data Structures

### Audio File Storage

```go
// AudioFile represents a recorded audio file
type AudioFile struct {
    ID          string      // Unique identifier
    Path        string      // File system path
    Format      AudioFormat // WAV or MP3
    Duration    float64     // Duration in seconds
    Size        int64       // File size in bytes
    SampleRate  int         // Sample rate in Hz
    Channels    int         // Number of channels
    BitDepth    int         // Bit depth
    CreatedAt   time.Time   // Creation timestamp
    DeviceID    string      // Recording device ID
    Metadata    map[string]string // Additional metadata
}
```

### Level Monitoring

```go
// LevelMonitor tracks real-time audio levels
type LevelMonitor struct {
    buffer      []float64   // Circular buffer for samples
    bufferSize  int
    windowSize  time.Duration
    updateRate  time.Duration
}

// Update adds new samples to the monitor
func (l *LevelMonitor) Update(samples []float64)

// GetLevels calculates current audio levels
func (l *LevelMonitor) GetLevels() *AudioLevels
```

### Silence Detection

```go
// SilenceDetector identifies periods of silence
type SilenceDetector struct {
    threshold       float64       // dB threshold
    minDuration     time.Duration // Minimum silence duration
    silenceStart    time.Time     // When silence began
    isSilent        bool
}

// IsSilent checks if current audio is below threshold
func (s *SilenceDetector) IsSilent(levels *AudioLevels) bool

// SilenceDuration returns how long silence has persisted
func (s *SilenceDetector) SilenceDuration() time.Duration

// Reset resets the silence detector state
func (s *SilenceDetector) Reset()
```

---

## Configuration Schema

### YAML Configuration

```yaml
voice:
  # Audio recording settings
  audio:
    sample_rate: 16000        # Hz, standard for speech
    channels: 1               # Mono
    bit_depth: 16            # 16-bit audio
    format: wav              # wav or mp3
    output_dir: ~/.helix/voice/recordings

  # Recording behavior
  recording:
    max_duration: 300s       # 5 minutes maximum
    silence_timeout: 2s      # Auto-stop after 2s silence
    silence_threshold: -40.0 # dB
    auto_trim_silence: true

  # Transcription settings
  transcription:
    api_key: ${OPENAI_API_KEY}
    model: whisper-1
    language: ""             # Auto-detect
    prompt: ""               # Optional context
    temperature: 0.0         # Deterministic
    base_url: https://api.openai.com/v1

  # Device settings
  device:
    default_device: ""       # Auto-select if empty
    auto_select: true
    refresh_interval: 30s

  # UI settings
  ui:
    show_levels: true
    level_update_rate: 100ms
    show_waveform: false
```

### Go Configuration Struct

```go
type Config struct {
    Voice VoiceSettings `yaml:"voice"`
}

type VoiceSettings struct {
    Audio         AudioSettings         `yaml:"audio"`
    Recording     RecordingSettings     `yaml:"recording"`
    Transcription TranscriptionSettings `yaml:"transcription"`
    Device        DeviceSettings        `yaml:"device"`
    UI            UISettings            `yaml:"ui"`
}

type AudioSettings struct {
    SampleRate  int         `yaml:"sample_rate"`
    Channels    int         `yaml:"channels"`
    BitDepth    int         `yaml:"bit_depth"`
    Format      AudioFormat `yaml:"format"`
    OutputDir   string      `yaml:"output_dir"`
}

type RecordingSettings struct {
    MaxDuration      time.Duration `yaml:"max_duration"`
    SilenceTimeout   time.Duration `yaml:"silence_timeout"`
    SilenceThreshold float64       `yaml:"silence_threshold"`
    AutoTrimSilence  bool          `yaml:"auto_trim_silence"`
}

type TranscriptionSettings struct {
    APIKey      string  `yaml:"api_key"`
    Model       string  `yaml:"model"`
    Language    string  `yaml:"language"`
    Prompt      string  `yaml:"prompt"`
    Temperature float64 `yaml:"temperature"`
    BaseURL     string  `yaml:"base_url"`
}

type DeviceSettings struct {
    DefaultDevice   string        `yaml:"default_device"`
    AutoSelect      bool          `yaml:"auto_select"`
    RefreshInterval time.Duration `yaml:"refresh_interval"`
}

type UISettings struct {
    ShowLevels      bool          `yaml:"show_levels"`
    LevelUpdateRate time.Duration `yaml:"level_update_rate"`
    ShowWaveform    bool          `yaml:"show_waveform"`
}
```

---

## Error Handling

### Error Types

```go
package voice

import "errors"

var (
    // Device errors
    ErrNoDevicesFound     = errors.New("no audio input devices found")
    ErrDeviceNotFound     = errors.New("specified device not found")
    ErrDeviceUnavailable  = errors.New("device is not available")
    ErrDeviceInUse        = errors.New("device is already in use")

    // Recording errors
    ErrAlreadyRecording   = errors.New("recording already in progress")
    ErrNotRecording       = errors.New("no recording in progress")
    ErrRecordingTimeout   = errors.New("recording exceeded maximum duration")
    ErrAudioCaptureFailed = errors.New("failed to capture audio")
    ErrInvalidFormat      = errors.New("invalid audio format")

    // Transcription errors
    ErrTranscriptionFailed = errors.New("transcription failed")
    ErrInvalidAPIKey       = errors.New("invalid or missing API key")
    ErrFileTooLarge        = errors.New("audio file exceeds size limit")
    ErrUnsupportedFormat   = errors.New("unsupported audio format")
    ErrNoSpeechDetected    = errors.New("no speech detected in audio")

    // File errors
    ErrFileNotFound        = errors.New("audio file not found")
    ErrFileReadFailed      = errors.New("failed to read audio file")
    ErrFileWriteFailed     = errors.New("failed to write audio file")
)

// VoiceError wraps errors with additional context
type VoiceError struct {
    Op      string // Operation that failed
    Err     error  // Underlying error
    Context string // Additional context
}

func (e *VoiceError) Error() string {
    if e.Context != "" {
        return fmt.Sprintf("%s: %v (%s)", e.Op, e.Err, e.Context)
    }
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *VoiceError) Unwrap() error {
    return e.Err
}
```

### Error Recovery

```go
// RetryConfig defines retry behavior for transient failures
type RetryConfig struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

// DefaultRetryConfig provides sensible defaults
var DefaultRetryConfig = RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 1 * time.Second,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
}

// WithRetry wraps an operation with retry logic
func WithRetry(ctx context.Context, config RetryConfig, fn func() error) error {
    var lastErr error
    delay := config.InitialDelay

    for attempt := 0; attempt < config.MaxAttempts; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err

            // Don't retry non-transient errors
            if !isTransientError(err) {
                return err
            }

            if attempt < config.MaxAttempts-1 {
                select {
                case <-ctx.Done():
                    return ctx.Err()
                case <-time.After(delay):
                    delay = time.Duration(float64(delay) * config.Multiplier)
                    if delay > config.MaxDelay {
                        delay = config.MaxDelay
                    }
                }
            }
        }
    }

    return fmt.Errorf("operation failed after %d attempts: %w",
        config.MaxAttempts, lastErr)
}

func isTransientError(err error) bool {
    // Network errors, timeouts, rate limits are transient
    // File not found, invalid API key, etc. are not
    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Timeout() {
        return true
    }

    if errors.Is(err, ErrDeviceUnavailable) ||
       errors.Is(err, ErrDeviceInUse) {
        return true
    }

    return false
}
```

---

## Testing Strategy

### Unit Tests

```go
package voice_test

import (
    "context"
    "testing"
    "time"

    "github.com/yourusername/helix/internal/voice"
)

// TestDeviceManager tests device enumeration and selection
func TestDeviceManager(t *testing.T) {
    tests := []struct {
        name    string
        setup   func(*testing.T) *voice.DeviceManager
        test    func(*testing.T, *voice.DeviceManager)
        wantErr bool
    }{
        {
            name: "list devices",
            setup: func(t *testing.T) *voice.DeviceManager {
                return voice.NewDeviceManager()
            },
            test: func(t *testing.T, dm *voice.DeviceManager) {
                devices, err := dm.ListDevices(context.Background())
                if err != nil {
                    t.Fatalf("ListDevices() error = %v", err)
                }
                if len(devices) == 0 {
                    t.Error("expected at least one device")
                }
            },
        },
        {
            name: "get default device",
            setup: func(t *testing.T) *voice.DeviceManager {
                return voice.NewDeviceManager()
            },
            test: func(t *testing.T, dm *voice.DeviceManager) {
                device, err := dm.GetDefaultDevice()
                if err != nil {
                    t.Fatalf("GetDefaultDevice() error = %v", err)
                }
                if !device.IsDefault {
                    t.Error("device should be marked as default")
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dm := tt.setup(t)
            tt.test(t, dm)
        })
    }
}

// TestAudioRecorder tests audio recording with mock input
func TestAudioRecorder(t *testing.T) {
    // Create mock audio device
    device := &voice.AudioDevice{
        ID:       "mock-device",
        Name:     "Mock Microphone",
        IsDefault: true,
        SampleRates: []int{16000, 44100},
        Channels: 1,
        IsAvailable: true,
    }

    config := &voice.AudioConfig{
        SampleRate: 16000,
        Channels: 1,
        BitDepth: 16,
        Format: voice.FormatWAV,
        OutputDirectory: t.TempDir(),
    }

    recorder, err := voice.NewAudioRecorder(device, config)
    if err != nil {
        t.Fatalf("NewAudioRecorder() error = %v", err)
    }

    ctx := context.Background()

    // Start recording
    if err := recorder.Start(ctx); err != nil {
        t.Fatalf("Start() error = %v", err)
    }

    if !recorder.IsRecording() {
        t.Error("expected recorder to be recording")
    }

    // Simulate recording time
    time.Sleep(500 * time.Millisecond)

    // Stop recording
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
}

// TestLevelMonitor tests audio level monitoring
func TestLevelMonitor(t *testing.T) {
    monitor := voice.NewLevelMonitor(100*time.Millisecond, 10*time.Millisecond)

    // Generate test samples (sine wave)
    samples := generateSineWave(1000, 440.0, 16000)
    monitor.Update(samples)

    levels := monitor.GetLevels()

    if levels.Peak == 0 {
        t.Error("expected non-zero peak level")
    }

    if levels.RMS == 0 {
        t.Error("expected non-zero RMS level")
    }
}

// TestSilenceDetector tests silence detection
func TestSilenceDetector(t *testing.T) {
    detector := voice.NewSilenceDetector(-40.0, 1*time.Second)

    // Test with loud audio
    loudLevels := &voice.AudioLevels{
        Peak: -10.0,
        RMS:  -15.0,
        Timestamp: time.Now(),
    }

    if detector.IsSilent(loudLevels) {
        t.Error("loud audio incorrectly detected as silent")
    }

    // Test with quiet audio
    quietLevels := &voice.AudioLevels{
        Peak: -50.0,
        RMS:  -55.0,
        Timestamp: time.Now(),
    }

    if !detector.IsSilent(quietLevels) {
        t.Error("quiet audio not detected as silent")
    }
}

func generateSineWave(samples int, frequency, sampleRate float64) []float64 {
    wave := make([]float64, samples)
    for i := range wave {
        t := float64(i) / sampleRate
        wave[i] = math.Sin(2 * math.Pi * frequency * t)
    }
    return wave
}
```

### Integration Tests

```go
package voice_test

import (
    "context"
    "os"
    "testing"

    "github.com/yourusername/helix/internal/voice"
)

// TestTranscriptionIntegration tests the full transcription flow
func TestTranscriptionIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    config := &voice.TranscriptionConfig{
        APIKey:      apiKey,
        Model:       "whisper-1",
        Temperature: 0.0,
    }

    transcriber, err := voice.NewTranscriber(config)
    if err != nil {
        t.Fatalf("NewTranscriber() error = %v", err)
    }

    // Use a test audio file with known content
    testFile := "testdata/sample_speech.wav"

    ctx := context.Background()
    result, err := transcriber.TranscribeFile(ctx, testFile)
    if err != nil {
        t.Fatalf("TranscribeFile() error = %v", err)
    }

    if result.Text == "" {
        t.Error("expected non-empty transcription")
    }

    if result.Language == "" {
        t.Error("expected language detection")
    }

    t.Logf("Transcription: %s", result.Text)
    t.Logf("Language: %s", result.Language)
}

// TestVoiceInputManagerE2E tests the end-to-end flow
func TestVoiceInputManagerE2E(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }

    config := &voice.VoiceConfig{
        SampleRate:      16000,
        Channels:        1,
        BitDepth:        16,
        Format:          voice.FormatWAV,
        MaxDuration:     30 * time.Second,
        SilenceTimeout:  2 * time.Second,
        SilenceThreshold: -40.0,
        WhisperModel:    "whisper-1",
        AutoSelect:      true,
    }

    manager, err := voice.NewVoiceInputManager(config)
    if err != nil {
        t.Fatalf("NewVoiceInputManager() error = %v", err)
    }

    ctx := context.Background()

    // List devices
    devices, err := manager.ListDevices(ctx)
    if err != nil {
        t.Fatalf("ListDevices() error = %v", err)
    }

    if len(devices) == 0 {
        t.Skip("no audio devices available")
    }

    // Note: Actual recording would require user interaction
    // In automated tests, we would use mock audio input
    t.Log("E2E test would record and transcribe audio here")
}
```

### Mock Audio Input

```go
package voice

import (
    "context"
    "math"
    "time"
)

// MockAudioDevice simulates an audio input device for testing
type MockAudioDevice struct {
    *AudioDevice
    generator SampleGenerator
}

// SampleGenerator generates test audio samples
type SampleGenerator interface {
    Generate(count int) []float64
}

// SineWaveGenerator generates sine wave samples
type SineWaveGenerator struct {
    Frequency  float64
    SampleRate float64
    Phase      float64
    Amplitude  float64
}

func (g *SineWaveGenerator) Generate(count int) []float64 {
    samples := make([]float64, count)
    phaseIncrement := 2 * math.Pi * g.Frequency / g.SampleRate

    for i := range samples {
        samples[i] = g.Amplitude * math.Sin(g.Phase)
        g.Phase += phaseIncrement
        if g.Phase > 2*math.Pi {
            g.Phase -= 2 * math.Pi
        }
    }

    return samples
}

// SilenceGenerator generates silent samples
type SilenceGenerator struct{}

func (g *SilenceGenerator) Generate(count int) []float64 {
    return make([]float64, count)
}

// NoiseGenerator generates white noise
type NoiseGenerator struct {
    Amplitude float64
    rand      *rand.Rand
}

func (g *NoiseGenerator) Generate(count int) []float64 {
    samples := make([]float64, count)
    for i := range samples {
        samples[i] = (g.rand.Float64()*2 - 1) * g.Amplitude
    }
    return samples
}
```

---

## Performance Considerations

### Audio Buffer Management

```go
// BufferPool manages reusable audio buffers
type BufferPool struct {
    pool sync.Pool
    size int
}

func NewBufferPool(bufferSize int) *BufferPool {
    return &BufferPool{
        pool: sync.Pool{
            New: func() interface{} {
                return make([]float64, bufferSize)
            },
        },
        size: bufferSize,
    }
}

func (p *BufferPool) Get() []float64 {
    return p.pool.Get().([]float64)
}

func (p *BufferPool) Put(buf []float64) {
    if len(buf) != p.size {
        return // Don't pool wrong-sized buffers
    }
    p.pool.Put(buf)
}
```

### Optimization Guidelines

1. **Memory Management**
   - Use buffer pools for audio samples
   - Limit maximum recording duration
   - Stream large files rather than loading entirely

2. **Audio Processing**
   - Process audio in chunks (e.g., 1024 samples)
   - Use goroutines for concurrent level monitoring
   - Implement circular buffers for real-time processing

3. **API Calls**
   - Cache transcription results
   - Implement request rate limiting
   - Use compression for large audio files

4. **File I/O**
   - Use buffered I/O for audio files
   - Implement atomic file writes
   - Clean up temporary files promptly

### Performance Metrics

```go
// Metrics tracks voice system performance
type Metrics struct {
    RecordingsStarted    atomic.Int64
    RecordingsCompleted  atomic.Int64
    RecordingsFailed     atomic.Int64
    TranscriptionCount   atomic.Int64
    TranscriptionErrors  atomic.Int64
    AverageRecordingTime atomic.Int64 // milliseconds
    AverageAPILatency    atomic.Int64 // milliseconds
    TotalAudioDuration   atomic.Int64 // seconds
}

func (m *Metrics) RecordRecording(duration time.Duration, success bool) {
    m.RecordingsStarted.Add(1)
    if success {
        m.RecordingsCompleted.Add(1)

        // Update average using exponential moving average
        current := m.AverageRecordingTime.Load()
        newAvg := (current*9 + duration.Milliseconds()) / 10
        m.AverageRecordingTime.Store(newAvg)
    } else {
        m.RecordingsFailed.Add(1)
    }
}

func (m *Metrics) RecordTranscription(latency time.Duration, success bool) {
    if success {
        m.TranscriptionCount.Add(1)

        current := m.AverageAPILatency.Load()
        newAvg := (current*9 + latency.Milliseconds()) / 10
        m.AverageAPILatency.Store(newAvg)
    } else {
        m.TranscriptionErrors.Add(1)
    }
}
```

---

## User Experience Flow

### Voice Input Flow

```
User Action                    System Response
────────────────────────────────────────────────────────────

1. /voice                      → List available devices
                               → Show current device

2. Select device (optional)    → Confirm device selection
                               → Validate device

3. Start recording             → Initialize audio capture
   - Press hotkey or           → Show recording indicator
   - Type "start"              → Display audio levels

4. Speak into microphone       → Monitor audio levels
                               → Detect silence periods
                               → Update duration counter

5. Stop recording              → Finalize audio file
   - Press hotkey or           → Show file size/duration
   - Type "stop" or            → Begin transcription
   - Auto-stop on silence

6. Transcription               → Upload to Whisper API
                               → Show progress indicator
                               → Receive transcribed text

7. Review & edit (optional)    → Display transcription
                               → Allow edits before submit

8. Submit                      → Process as user input
                               → Continue conversation
```

### CLI Interface

```bash
# List devices
$ helix voice devices
Available audio input devices:
  1. [DEFAULT] Built-in Microphone (ID: default)
  2. USB Microphone (ID: usb-0)
  3. Virtual Audio Device (ID: virtual-0)

Current device: Built-in Microphone

# Select device
$ helix voice device --select usb-0
Selected device: USB Microphone

# Start interactive voice session
$ helix voice
Recording... (Press Ctrl+D to stop, or wait for auto-stop)

🎤 [====---] 5s | Peak: -12dB | RMS: -18dB

Stopped. Transcribing...

Transcription:
"Create a new function to calculate the Fibonacci sequence up to n terms"

[E]dit, [S]ubmit, or [C]ancel? s

Processing request...
```

### TUI Interface

```
┌─────────────────────────────────────────────────────────────┐
│ Voice Input                                                 │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ Device: Built-in Microphone                    [Change]    │
│                                                             │
│ ┌─────────────────────────────────────────────────────┐   │
│ │ 🎤 Recording: 00:05                                 │   │
│ │                                                     │   │
│ │ Peak Level:  [████████████░░░░░░░░] -12 dB        │   │
│ │ RMS Level:   [█████████░░░░░░░░░░░] -18 dB        │   │
│ │                                                     │   │
│ │ Silence detected: 0s / 2s                          │   │
│ └─────────────────────────────────────────────────────┘   │
│                                                             │
│ Controls:                                                   │
│   Ctrl+D  - Stop recording                                 │
│   Ctrl+C  - Cancel                                         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 1: Core Audio (Week 1-2)
- [ ] Implement DeviceManager
- [ ] Implement AudioRecorder with WAV support
- [ ] Add basic level monitoring
- [ ] Unit tests for audio components

### Phase 2: Transcription (Week 2-3)
- [ ] Implement Whisper API client
- [ ] Add file upload and transcription
- [ ] Error handling and retries
- [ ] Integration tests with API

### Phase 3: Advanced Features (Week 3-4)
- [ ] Silence detection
- [ ] Auto-stop on silence
- [ ] MP3 format support
- [ ] Real-time level visualization

### Phase 4: Integration (Week 4-5)
- [ ] CLI commands
- [ ] TUI interface
- [ ] Configuration management
- [ ] End-to-end testing

### Phase 5: Polish (Week 5-6)
- [ ] Performance optimization
- [ ] Error message improvements
- [ ] Documentation
- [ ] User acceptance testing

---

## Security Considerations

1. **API Key Management**
   - Store API keys in secure configuration
   - Support environment variables
   - Never log API keys
   - Validate key format before use

2. **File Permissions**
   - Restrict recording directory permissions (0700)
   - Clean up temporary files
   - Validate file paths to prevent traversal

3. **Audio Privacy**
   - Optional local-only transcription
   - Configurable data retention
   - Clear audio file cleanup policy
   - User consent for cloud processing

4. **Network Security**
   - Use HTTPS for API calls
   - Validate SSL certificates
   - Support proxy configuration
   - Implement request timeouts

---

## References

- **Aider voice.py**: Audio recording and device management implementation
- **OpenAI Whisper API**: https://platform.openai.com/docs/guides/speech-to-text
- **PortAudio**: Cross-platform audio I/O library
- **FFmpeg**: Audio format conversion and processing
