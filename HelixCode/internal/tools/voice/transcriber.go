package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

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
	Text     string    // Transcribed text
	Language string    // Detected language
	Duration float64   // Audio duration in seconds
	Segments []Segment // Optional word-level segments
	Metadata *Metadata // Additional metadata
}

// Segment represents a timestamped segment of transcription
type Segment struct {
	ID               int     `json:"id"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	AvgLogProb       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

// Metadata contains additional transcription information
type Metadata struct {
	Model          string  `json:"model"`
	RequestID      string  `json:"request_id"`
	ProcessingTime float64 `json:"processing_time"`
}

// Transcriber handles speech-to-text conversion via Whisper API
type Transcriber struct {
	client *http.Client
	config *TranscriptionConfig
}

// whisperResponse represents the API response structure
type whisperResponse struct {
	Text     string    `json:"text"`
	Language string    `json:"language"`
	Duration float64   `json:"duration"`
	Segments []Segment `json:"segments"`
}

// NewTranscriber creates a new transcriber
func NewTranscriber(config *TranscriptionConfig) (*Transcriber, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, ErrInvalidAPIKey
	}

	if config.Model == "" {
		config.Model = "whisper-1"
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	return &Transcriber{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}, nil
}

// TranscribeFile transcribes an audio file
func (t *Transcriber) TranscribeFile(ctx context.Context, filePath string) (*TranscriptionResult, error) {
	// Validate file exists
	if err := t.ValidateAudioFile(filePath); err != nil {
		return nil, err
	}

	// Check file size (25MB limit for Whisper API)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	maxSize := int64(25 * 1024 * 1024) // 25MB
	if fileInfo.Size() > maxSize {
		return nil, ErrFileTooLarge
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Add model
	if err := writer.WriteField("model", t.config.Model); err != nil {
		return nil, fmt.Errorf("failed to write model field: %w", err)
	}

	// Add optional fields
	if t.config.Language != "" {
		if err := writer.WriteField("language", t.config.Language); err != nil {
			return nil, fmt.Errorf("failed to write language field: %w", err)
		}
	}

	if t.config.Prompt != "" {
		if err := writer.WriteField("prompt", t.config.Prompt); err != nil {
			return nil, fmt.Errorf("failed to write prompt field: %w", err)
		}
	}

	if t.config.Temperature > 0 {
		if err := writer.WriteField("temperature", fmt.Sprintf("%f", t.config.Temperature)); err != nil {
			return nil, fmt.Errorf("failed to write temperature field: %w", err)
		}
	}

	// Add response format to get detailed output
	if err := writer.WriteField("response_format", "verbose_json"); err != nil {
		return nil, fmt.Errorf("failed to write response_format field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/audio/transcriptions", t.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.config.APIKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	startTime := time.Now()
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var whisperResp whisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&whisperResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Build result
	result := &TranscriptionResult{
		Text:     whisperResp.Text,
		Language: whisperResp.Language,
		Duration: whisperResp.Duration,
		Segments: whisperResp.Segments,
		Metadata: &Metadata{
			Model:          t.config.Model,
			ProcessingTime: time.Since(startTime).Seconds(),
		},
	}

	return result, nil
}

// TranscribeStream transcribes audio from a stream
func (t *Transcriber) TranscribeStream(ctx context.Context, reader io.Reader, format AudioFormat) (*TranscriptionResult, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("voice_*.%s", format))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy stream to file
	if _, err := io.Copy(tmpFile, reader); err != nil {
		return nil, fmt.Errorf("failed to copy stream: %w", err)
	}

	// Transcribe the temp file
	return t.TranscribeFile(ctx, tmpFile.Name())
}

// ValidateAudioFile checks if the file is suitable for transcription
func (t *Transcriber) ValidateAudioFile(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return ErrFileNotFound
	}

	// Check file extension
	ext := filepath.Ext(filePath)
	validExtensions := map[string]bool{
		".wav":  true,
		".mp3":  true,
		".m4a":  true,
		".flac": true,
		".ogg":  true,
		".webm": true,
	}

	if !validExtensions[ext] {
		return ErrUnsupportedFormat
	}

	return nil
}

// MockTranscriber is a mock transcriber for testing
type MockTranscriber struct {
	config        *TranscriptionConfig
	mockResponses map[string]string
}

// NewMockTranscriber creates a new mock transcriber
func NewMockTranscriber(config *TranscriptionConfig) *MockTranscriber {
	return &MockTranscriber{
		config: config,
		mockResponses: map[string]string{
			"default": "This is a mock transcription of the audio file.",
		},
	}
}

// TranscribeFile returns a mock transcription
func (m *MockTranscriber) TranscribeFile(ctx context.Context, filePath string) (*TranscriptionResult, error) {
	// Validate file
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, ErrFileNotFound
	}

	// Return mock result
	text := m.mockResponses["default"]
	if customText, ok := m.mockResponses[filePath]; ok {
		text = customText
	}

	return &TranscriptionResult{
		Text:     text,
		Language: "en",
		Duration: 5.0,
		Segments: []Segment{
			{
				ID:               0,
				Start:            0.0,
				End:              5.0,
				Text:             text,
				AvgLogProb:       -0.5,
				CompressionRatio: 1.5,
				NoSpeechProb:     0.1,
			},
		},
		Metadata: &Metadata{
			Model:          "whisper-1-mock",
			ProcessingTime: 0.5,
		},
	}, nil
}

// SetMockResponse sets a custom response for a specific file
func (m *MockTranscriber) SetMockResponse(filePath, response string) {
	m.mockResponses[filePath] = response
}
