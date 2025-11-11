package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ModelFormat represents different model formats
type ModelFormat string

const (
	FormatGGUF ModelFormat = "gguf"
	FormatGPTQ ModelFormat = "gptq"
	FormatAWQ  ModelFormat = "awq"
	FormatBF16 ModelFormat = "bf16"
	FormatFP16 ModelFormat = "fp16"
	FormatINT8 ModelFormat = "int8"
	FormatINT4 ModelFormat = "int4"
	FormatHF   ModelFormat = "hf" // HuggingFace format
)

// ModelDownloadSource represents a source for downloading models
type ModelDownloadSource struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Formats     []ModelFormat     `json:"formats"`
	Headers     map[string]string `json:"headers"`
	Description string            `json:"description"`
	Priority    int               `json:"priority"` // Lower number = higher priority
}

// DownloadableModelInfo represents a downloadable model
type DownloadableModelInfo struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Provider         string                 `json:"provider"`
	AvailableFormats []ModelFormat          `json:"available_formats"`
	DefaultFormat    ModelFormat            `json:"default_format"`
	Sources          []ModelDownloadSource  `json:"sources"`
	ModelSize        string                 `json:"model_size"` // "7B", "13B", "34B", "70B"
	ContextSize      int                    `json:"context_size"`
	Requirements     ModelRequirements      `json:"requirements"`
	DownloadURLs     map[ModelFormat]string `json:"download_urls"`
	LastUpdated      time.Time              `json:"last_updated"`
	Tags             []string               `json:"tags"`
}

// ModelRequirements specifies hardware requirements
type ModelRequirements struct {
	MinRAM          string   `json:"min_ram"`
	MinVRAM         string   `json:"min_vram,omitempty"`
	RecommendedVRAM string   `json:"recommended_vram,omitempty"`
	SupportedOS     []string `json:"supported_os"`
	GPURequired     bool     `json:"gpu_required"`
	CPUOnly         bool     `json:"cpu_only"`
}

// ModelDownloadRequest represents a model download request
type ModelDownloadRequest struct {
	ModelID        string      `json:"model_id"`
	Format         ModelFormat `json:"format"`
	TargetProvider string      `json:"target_provider"`
	TargetPath     string      `json:"target_path,omitempty"`
	ForceDownload  bool        `json:"force_download"`
}

// ModelDownloadProgress represents download progress
type ModelDownloadProgress struct {
	ModelID   string      `json:"model_id"`
	Format    ModelFormat `json:"format"`
	Progress  float64     `json:"progress"` // 0.0 to 1.0
	Speed     int64       `json:"speed"`    // bytes per second
	ETA       int64       `json:"eta"`      // estimated time remaining in seconds
	Error     string      `json:"error,omitempty"`
	StartTime time.Time   `json:"start_time"`
}

// ModelDownloadManager handles model downloading and format conversion
type ModelDownloadManager struct {
	baseDir         string
	httpClient      *http.Client
	availableModels map[string]*DownloadableModelInfo
	sources         []ModelDownloadSource
	conversionTools map[ModelFormat]*ConversionTool
	downloads       map[string]*ModelDownloadProgress
}

// ConversionTool represents a format conversion tool
type ConversionTool struct {
	Name          string            `json:"name"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	SourceFormats []ModelFormat     `json:"source_formats"`
	TargetFormat  ModelFormat       `json:"target_format"`
	EnvVars       map[string]string `json:"env_vars"`
}

// NewModelDownloadManager creates a new model download manager
func NewModelDownloadManager(baseDir string) *ModelDownloadManager {
	m := &ModelDownloadManager{
		baseDir:         baseDir,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		availableModels: make(map[string]*DownloadableModelInfo),
		sources:         initializeDownloadSources(),
		conversionTools: initializeConversionTools(),
		downloads:       make(map[string]*ModelDownloadProgress),
	}

	// Ensure base directory exists
	os.MkdirAll(baseDir, 0755)

	// Load model registry
	m.loadModelRegistry()

	return m
}

// GetAvailableModels returns all available models
func (m *ModelDownloadManager) GetAvailableModels() []*DownloadableModelInfo {
	var models []*DownloadableModelInfo
	for _, model := range m.availableModels {
		models = append(models, model)
	}
	return models
}

// SearchModels searches for models by name, description, or tags
func (m *ModelDownloadManager) SearchModels(query string) []*DownloadableModelInfo {
	query = strings.ToLower(query)
	var results []*DownloadableModelInfo

	for _, model := range m.availableModels {
		if strings.Contains(strings.ToLower(model.Name), query) ||
			strings.Contains(strings.ToLower(model.Description), query) {
			results = append(results, model)
			continue
		}

		for _, tag := range model.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, model)
				break
			}
		}
	}

	return results
}

// GetModelByID returns model information by ID
func (m *ModelDownloadManager) GetModelByID(modelID string) (*DownloadableModelInfo, error) {
	model, exists := m.availableModels[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}
	return model, nil
}

// GetCompatibleModels returns models compatible with a specific provider
func (m *ModelDownloadManager) GetCompatibleFormats(provider string, modelID string) ([]ModelFormat, error) {
	model, err := m.GetModelByID(modelID)
	if err != nil {
		return nil, err
	}

	// Get supported formats for provider
	supportedFormats := m.getProviderSupportedFormats(provider)

	// Find intersection
	var compatible []ModelFormat
	for _, format := range model.AvailableFormats {
		for _, supported := range supportedFormats {
			if format == supported {
				compatible = append(compatible, format)
			}
		}
	}

	return compatible, nil
}

// DownloadModel downloads a model in the specified format
func (m *ModelDownloadManager) DownloadModel(ctx context.Context, req ModelDownloadRequest) (<-chan ModelDownloadProgress, error) {
	model, err := m.GetModelByID(req.ModelID)
	if err != nil {
		return nil, err
	}

	// Check if format is available
	formatAvailable := false
	for _, format := range model.AvailableFormats {
		if format == req.Format {
			formatAvailable = true
			break
		}
	}

	if !formatAvailable {
		// Check if we can convert from another format
		canConvert := false
		for _, tool := range m.conversionTools {
			if tool.TargetFormat == req.Format {
				for _, sourceFormat := range model.AvailableFormats {
					for _, sf := range tool.SourceFormats {
						if sf == sourceFormat {
							canConvert = true
							break
						}
					}
					if canConvert {
						break
					}
				}
			}
			if canConvert {
				break
			}
		}

		if !canConvert {
			return nil, fmt.Errorf("format %s not available for model %s and no conversion path found", req.Format, req.ModelID)
		}
	}

	// Create progress channel
	progressChan := make(chan ModelDownloadProgress)

	// Start download
	go m.downloadModelWithProgress(ctx, model, req, progressChan)

	return progressChan, nil
}

// downloadModelWithProgress handles the actual download with progress reporting
func (m *ModelDownloadManager) downloadModelWithProgress(ctx context.Context, model *DownloadableModelInfo, req ModelDownloadRequest, progressChan chan<- ModelDownloadProgress) {
	defer close(progressChan)

	progressKey := fmt.Sprintf("%s:%s", req.ModelID, req.Format)

	// Initialize progress
	progress := &ModelDownloadProgress{
		ModelID:   req.ModelID,
		Format:    req.Format,
		Progress:  0.0,
		StartTime: time.Now(),
	}
	m.downloads[progressKey] = progress

	// Send initial progress
	progressChan <- *progress

	// Find download source
	downloadURL, err := m.getDownloadURL(model, req.Format)
	if err != nil {
		progress.Error = err.Error()
		progressChan <- *progress
		return
	}

	// Determine target path
	targetPath := req.TargetPath
	if targetPath == "" {
		targetPath = m.getModelPath(req.TargetProvider, req.ModelID, req.Format)
	}

	// Ensure directory exists
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		progress.Error = fmt.Sprintf("failed to create directory: %v", err)
		progressChan <- *progress
		return
	}

	// Download file
	if err := m.downloadFile(ctx, downloadURL, targetPath, progress); err != nil {
		progress.Error = err.Error()
		progressChan <- *progress
		return
	}

	// Convert if necessary
	if !m.isFormatAvailableDirectly(model, req.Format) {
		conversionProgress := m.convertModel(targetPath, req.Format, progressChan)
		if conversionProgress.Error != "" {
			progress.Error = conversionProgress.Error
			progressChan <- *progress
			return
		}
	}

	// Complete
	progress.Progress = 1.0
	progressChan <- *progress

	log.Printf("✅ Successfully downloaded model %s in %s format", req.ModelID, req.Format)
}

// downloadFile downloads a file with progress tracking
func (m *ModelDownloadManager) downloadFile(ctx context.Context, url, targetPath string, progress *ModelDownloadProgress) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Get file size
	totalSize := resp.ContentLength
	if totalSize <= 0 {
		totalSize = 0 // Unknown size
	}

	// Create file
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Download with progress
	var downloaded int64 = 0
	buffer := make([]byte, 32*1024) // 32KB buffer
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				wrote, err := file.Write(buffer[:n])
				if err != nil {
					return fmt.Errorf("failed to write file: %w", err)
				}
				downloaded += int64(wrote)

				// Update progress
				if totalSize > 0 {
					progress.Progress = float64(downloaded) / float64(totalSize)
				}

				// Calculate speed and ETA
				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					progress.Speed = int64(float64(downloaded) / elapsed)
					if totalSize > 0 && progress.Speed > 0 {
						progress.ETA = int64((float64(totalSize-downloaded) / float64(progress.Speed)))
					}
				}
			}

			if err == io.EOF {
				return nil
			}

			if err != nil {
				return fmt.Errorf("download error: %w", err)
			}
		}
	}
}

// convertModel converts a model to the target format
func (m *ModelDownloadManager) convertModel(inputPath string, targetFormat ModelFormat, progressChan chan<- ModelDownloadProgress) *ModelDownloadProgress {
	progress := &ModelDownloadProgress{}

	// Find conversion tool
	var tool *ConversionTool
	for _, t := range m.conversionTools {
		if t.TargetFormat == targetFormat {
			tool = t
			break
		}
	}

	if tool == nil {
		progress.Error = fmt.Sprintf("no conversion tool found for format %s", targetFormat)
		return progress
	}

	// Validate input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		progress.Error = fmt.Sprintf("input file not found: %s", inputPath)
		return progress
	}

	log.Printf("Converting model to %s format using %s", targetFormat, tool.Name)

	// Prepare output path
	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "." + string(targetFormat)

	// Build conversion command with arguments
	args := make([]string, 0, len(tool.Args)+2)
	for _, arg := range tool.Args {
		// Replace placeholders in arguments
		arg = strings.ReplaceAll(arg, "{input}", inputPath)
		arg = strings.ReplaceAll(arg, "{output}", outputPath)
		arg = strings.ReplaceAll(arg, "{format}", string(targetFormat))
		args = append(args, arg)
	}

	// Update progress
	progress.Progress = 0.1
	log.Printf("Preparing conversion to %s", targetFormat)
	progressChan <- *progress

	// Execute conversion command
	cmd := exec.Command(tool.Command, args...)

	// Set environment variables if specified
	if len(tool.EnvVars) > 0 {
		cmd.Env = os.Environ()
		for key, value := range tool.EnvVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Capture output for error reporting
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Start the conversion process
	progress.Progress = 0.2
	log.Printf("Running conversion command: %s %v", tool.Command, args)
	progressChan <- *progress

	if err := cmd.Start(); err != nil {
		progress.Error = fmt.Sprintf("failed to start conversion: %v", err)
		return progress
	}

	// Monitor conversion progress (simplified - in production would parse tool output)
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Simulate incremental progress updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	currentProgress := 0.2
	for {
		select {
		case err := <-done:
			if err != nil {
				progress.Error = fmt.Sprintf("conversion failed: %v\nStderr: %s", err, stderr.String())
				return progress
			}

			// Verify output file was created
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				progress.Error = fmt.Sprintf("conversion completed but output file not found: %s", outputPath)
				return progress
			}

			// Log completion
			if stat, err := os.Stat(outputPath); err == nil {
				log.Printf("Conversion complete: output file size: %d bytes", stat.Size())
			}

			progress.Progress = 1.0
			log.Printf("Model conversion to %s complete: %s", targetFormat, outputPath)
			progressChan <- *progress
			return progress

		case <-ticker.C:
			// Increment progress gradually (would ideally parse tool output for real progress)
			if currentProgress < 0.9 {
				currentProgress += 0.1
				progress.Progress = currentProgress
				log.Printf("Converting to %s format... %.0f%% complete", targetFormat, currentProgress*100)
				progressChan <- *progress
			}
		}
	}
}

// Helper functions

func (m *ModelDownloadManager) getDownloadURL(model *DownloadableModelInfo, format ModelFormat) (string, error) {
	// Check if we have direct download URL
	if url, exists := model.DownloadURLs[format]; exists {
		return url, nil
	}

	// Find source that provides this format
	for _, source := range model.Sources {
		for _, f := range source.Formats {
			if f == format {
				// Construct URL - this is simplified, real implementation would be more sophisticated
				return fmt.Sprintf("%s/%s/%s", source.URL, model.ID, string(format)), nil
			}
		}
	}

	return "", fmt.Errorf("no download source found for format %s", format)
}

func (m *ModelDownloadManager) isFormatAvailableDirectly(model *DownloadableModelInfo, format ModelFormat) bool {
	for _, f := range model.AvailableFormats {
		if f == format {
			return true
		}
	}
	return false
}

func (m *ModelDownloadManager) getModelPath(provider, modelID string, format ModelFormat) string {
	return filepath.Join(m.baseDir, provider, modelID, fmt.Sprintf("model.%s", format))
}

func (m *ModelDownloadManager) getProviderSupportedFormats(provider string) []ModelFormat {
	// This would be based on provider capabilities
	switch provider {
	case "llamacpp", "ollama", "localai", "vllm":
		return []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF}
	case "textgen":
		return []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF}
	case "lmstudio", "jan", "tabbyapi":
		return []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF}
	case "mlx":
		return []ModelFormat{FormatGGUF, FormatHF} // MLX specific support
	case "mistralrs":
		return []ModelFormat{FormatGGUF, FormatGPTQ, FormatHF, FormatBF16, FormatFP16}
	case "koboldai", "gpt4all":
		return []ModelFormat{FormatGGUF} // CPU-focused, GGUF is best
	default:
		return []ModelFormat{FormatGGUF} // Most universal format
	}
}

func (m *ModelDownloadManager) loadModelRegistry() {
	// In a real implementation, this would load from a JSON file or API
	// For now, we'll add some popular models

	m.availableModels["llama-3-8b-instruct"] = &DownloadableModelInfo{
		ID:               "llama-3-8b-instruct",
		Name:             "Llama 3 8B Instruct",
		Description:      "Meta's Llama 3 8B instruction-tuned model",
		Provider:         "meta",
		AvailableFormats: []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF},
		DefaultFormat:    FormatGGUF,
		ModelSize:        "8B",
		ContextSize:      8192,
		Tags:             []string{"instruct", "chat", "general", "english"},
		DownloadURLs: map[ModelFormat]string{
			FormatGGUF: "https://huggingface.co/bartowski/Meta-Llama-3-8B-Instruct-GGUF/resolve/main/Meta-Llama-3-8B-Instruct-Q4_K_M.gguf",
		},
		LastUpdated: time.Now(),
		Requirements: ModelRequirements{
			MinRAM:          "8GB",
			MinVRAM:         "6GB",
			RecommendedVRAM: "8GB",
			SupportedOS:     []string{"linux", "darwin", "windows"},
			GPURequired:     false,
			CPUOnly:         true,
		},
	}

	m.availableModels["mistral-7b-instruct"] = &DownloadableModelInfo{
		ID:               "mistral-7b-instruct",
		Name:             "Mistral 7B Instruct",
		Description:      "Mistral AI's 7B instruction-tuned model",
		Provider:         "mistral",
		AvailableFormats: []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF},
		DefaultFormat:    FormatGGUF,
		ModelSize:        "7B",
		ContextSize:      32768,
		Tags:             []string{"instruct", "chat", "general", "multilingual"},
		DownloadURLs: map[ModelFormat]string{
			FormatGGUF: "https://huggingface.co/bartowski/Mistral-7B-Instruct-v0.2-GGUF/resolve/main/Mistral-7B-Instruct-v0.2-Q4_K_M.gguf",
		},
		LastUpdated: time.Now(),
		Requirements: ModelRequirements{
			MinRAM:          "8GB",
			MinVRAM:         "6GB",
			RecommendedVRAM: "8GB",
			SupportedOS:     []string{"linux", "darwin", "windows"},
			GPURequired:     false,
			CPUOnly:         true,
		},
	}

	m.availableModels["codellama-7b-instruct"] = &DownloadableModelInfo{
		ID:               "codellama-7b-instruct",
		Name:             "CodeLlama 7B Instruct",
		Description:      "Meta's CodeLlama 7B instruction-tuned model for code generation",
		Provider:         "meta",
		AvailableFormats: []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF},
		DefaultFormat:    FormatGGUF,
		ModelSize:        "7B",
		ContextSize:      16384,
		Tags:             []string{"instruct", "code", "programming", "development"},
		DownloadURLs: map[ModelFormat]string{
			FormatGGUF: "https://huggingface.co/bartowski/CodeLlama-7B-Instruct-GGUF/resolve/main/codellama-7b-instruct.Q4_K_M.gguf",
		},
		LastUpdated: time.Now(),
		Requirements: ModelRequirements{
			MinRAM:          "8GB",
			MinVRAM:         "6GB",
			RecommendedVRAM: "8GB",
			SupportedOS:     []string{"linux", "darwin", "windows"},
			GPURequired:     false,
			CPUOnly:         true,
		},
	}
}

func initializeDownloadSources() []ModelDownloadSource {
	return []ModelDownloadSource{
		{
			Name:        "HuggingFace",
			URL:         "https://huggingface.co",
			Formats:     []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ, FormatHF, FormatBF16, FormatFP16},
			Description: "Main HuggingFace model hub",
			Priority:    1,
		},
		{
			Name:        "TheBloke",
			URL:         "https://huggingface.co/TheBloke",
			Formats:     []ModelFormat{FormatGGUF, FormatGPTQ, FormatAWQ},
			Description: "High-quality quantized models by TheBloke",
			Priority:    2,
		},
		{
			Name:        "Bartowski",
			URL:         "https://huggingface.co/bartowski",
			Formats:     []ModelFormat{FormatGGUF},
			Description: "GGUF quantized models",
			Priority:    3,
		},
	}
}

func initializeConversionTools() map[ModelFormat]*ConversionTool {
	return map[ModelFormat]*ConversionTool{
		FormatGGUF: {
			Name:          "llama.cpp",
			Command:       "python",
			Args:          []string{"-m", "llama_cpp.convert"},
			SourceFormats: []ModelFormat{FormatHF, FormatFP16, FormatBF16},
			TargetFormat:  FormatGGUF,
			EnvVars:       map[string]string{"HF_HUB_DISABLE_TELEMETRY": "1"},
		},
		FormatGPTQ: {
			Name:          "AutoGPTQ",
			Command:       "python",
			Args:          []string{"-m", "auto_gptq"},
			SourceFormats: []ModelFormat{FormatHF, FormatFP16, FormatBF16},
			TargetFormat:  FormatGPTQ,
			EnvVars:       map[string]string{"HF_HUB_DISABLE_TELEMETRY": "1"},
		},
		FormatAWQ: {
			Name:          "AutoAWQ",
			Command:       "python",
			Args:          []string{"-m", "awq"},
			SourceFormats: []ModelFormat{FormatHF, FormatFP16, FormatBF16},
			TargetFormat:  FormatAWQ,
			EnvVars:       map[string]string{"HF_HUB_DISABLE_TELEMETRY": "1"},
		},
	}
}
