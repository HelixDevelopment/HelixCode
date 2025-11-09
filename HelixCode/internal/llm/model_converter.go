package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ModelConverter handles format conversion between different model formats
type ModelConverter struct {
	baseDir         string
	conversionTools map[string]*ConversionTool
	tempDir         string
	httpClient      *HTTPClient
}

// ConversionJob represents an ongoing conversion
type ConversionJob struct {
	ID            string             `json:"id"`
	SourcePath    string             `json:"source_path"`
	TargetPath    string             `json:"target_path"`
	SourceFormat  ModelFormat        `json:"source_format"`
	TargetFormat  ModelFormat        `json:"target_format"`
	Progress      float64            `json:"progress"`
	Status        ConversionStatus    `json:"status"`
	StartTime     time.Time          `json:"start_time"`
	EndTime       *time.Time         `json:"end_time,omitempty"`
	Error         string             `json:"error,omitempty"`
	LogPath       string             `json:"log_path"`
	Command       string             `json:"command"`
	Args          []string           `json:"args"`
	EstimatedTime int64              `json:"estimated_time"` // in seconds
	CurrentStep   string             `json:"current_step"`
}

// ConversionStatus represents the status of a conversion job
type ConversionStatus string

const (
	StatusPending    ConversionStatus = "pending"
	StatusRunning    ConversionStatus = "running"
	StatusCompleted  ConversionStatus = "completed"
	StatusFailed     ConversionStatus = "failed"
	StatusCancelled  ConversionStatus = "cancelled"
)

// ConversionConfig represents conversion configuration
type ConversionConfig struct {
	SourcePath    string            `json:"source_path"`
	SourceFormat  ModelFormat       `json:"source_format"`
	TargetFormat  ModelFormat       `json:"target_format"`
	TargetPath    string            `json:"target_path,omitempty"`
	Quantization  *QuantizationConfig `json:"quantization,omitempty"`
	Optimization  *OptimizationConfig `json:"optimization,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	Timeout       int               `json:"timeout,omitempty"` // in minutes
}

// QuantizationConfig represents quantization options
type QuantizationConfig struct {
	Method     string `json:"method"`     // "q4_k_m", "q8_0", "q4_0", etc.
	Bits       int    `json:"bits"`       // 4, 8, 16
	UseExllama bool   `json:"use_exllama"`
	UseFp16    bool   `json:"use_fp16"`
}

// OptimizationConfig represents optimization options
type OptimizationConfig struct	{
	RemoveUnusedLayers bool     `json:"remove_unused_layers"`
	FuseOperations      bool     `json:"fuse_operations"`
	OptimizeFor         string   `json:"optimize_for"` // "cpu", "gpu", "mobile"
	TargetHardware      string   `json:"target_hardware"` // "nvidia", "amd", "apple", "intel"
	DeviceMap           string   `json:"device_map"` // "auto", "cpu", "cuda:0"
}

// NewModelConverter creates a new model converter
func NewModelConverter(baseDir string) *ModelConverter {
	tempDir := filepath.Join(baseDir, "temp")
	os.MkdirAll(tempDir, 0755)
	
	return &ModelConverter{
		baseDir:         baseDir,
		conversionTools: initializeAllConversionTools(),
		tempDir:         tempDir,
		httpClient:      &HTTPClient{Timeout: 30 * time.Second},
	}
}

// ConvertModel converts a model from one format to another
func (c *ModelConverter) ConvertModel(ctx context.Context, config ConversionConfig) (*ConversionJob, error) {
	// Validate conversion path
	tool, err := c.getConversionTool(config.SourceFormat, config.TargetFormat)
	if err != nil {
		return nil, err
	}
	
	// Set default target path
	if config.TargetPath == "" {
		config.TargetPath = c.generateTargetPath(config.SourcePath, config.TargetFormat)
	}
	
	// Create conversion job
	jobID := generateJobID()
	job := &ConversionJob{
		ID:           jobID,
		SourcePath:   config.SourcePath,
		TargetPath:   config.TargetPath,
		SourceFormat: config.SourceFormat,
		TargetFormat: config.TargetFormat,
		Status:       StatusPending,
		StartTime:    time.Now(),
		LogPath:      filepath.Join(c.baseDir, "logs", fmt.Sprintf("conversion_%s.log", jobID)),
		Command:      tool.Command,
		Args:         c.buildCommand(tool, config),
	}
	
	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(config.TargetPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}
	
	// Estimate conversion time
	job.EstimatedTime = c.estimateConversionTime(config)
	
	// Start conversion in background
	go c.runConversion(ctx, job, config, tool)
	
	return job, nil
}

// GetConversionStatus returns the status of a conversion job
func (c *ModelConverter) GetConversionStatus(jobID string) (*ConversionJob, error) {
	jobPath := filepath.Join(c.baseDir, "jobs", fmt.Sprintf("%s.json", jobID))
	
	data, err := os.ReadFile(jobPath)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	
	var job ConversionJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to read job status: %w", err)
	}
	
	return &job, nil
}

// ListConversionJobs returns all conversion jobs
func (c *ModelConverter) ListConversionJobs() ([]*ConversionJob, error) {
	jobDir := filepath.Join(c.baseDir, "jobs")
	files, err := os.ReadDir(jobDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ConversionJob{}, nil
		}
		return nil, err
	}
	
	var jobs []*ConversionJob
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			jobPath := filepath.Join(jobDir, file.Name())
			data, err := os.ReadFile(jobPath)
			if err != nil {
				continue
			}
			
			var job ConversionJob
			if err := json.Unmarshal(data, &job); err != nil {
				continue
			}
			
			jobs = append(jobs, &job)
		}
	}
	
	return jobs, nil
}

// CancelConversion cancels an ongoing conversion
func (c *ModelConverter) CancelConversion(jobID string) error {
	job, err := c.GetConversionStatus(jobID)
	if err != nil {
		return err
	}
	
	if job.Status != StatusRunning && job.Status != StatusPending {
		return fmt.Errorf("job %s cannot be cancelled (current status: %s)", jobID, job.Status)
	}
	
	job.Status = StatusCancelled
	if job.EndTime == nil {
		now := time.Now()
		job.EndTime = &now
	}
	
	return c.saveJobStatus(job)
}

// GetSupportedConversions returns all supported format conversions
func (c *ModelConverter) GetSupportedConversions() map[string][]string {
	conversions := make(map[string][]string)
	
	for _, tool := range c.conversionTools {
		for _, sourceFormat := range tool.SourceFormats {
			sourceKey := string(sourceFormat)
			if _, exists := conversions[sourceKey]; !exists {
				conversions[sourceKey] = []string{}
			}
			
			targetKey := string(tool.TargetFormat)
			alreadyExists := false
			for _, existing := range conversions[sourceKey] {
				if existing == targetKey {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				conversions[sourceKey] = append(conversions[sourceKey], targetKey)
			}
		}
	}
	
	return conversions
}

// Private methods

func (c *ModelConverter) runConversion(ctx context.Context, job *ConversionJob, config ConversionConfig, tool *ConversionTool) {
	// Save initial job status
	c.saveJobStatus(job)
	
	// Update status to running
	job.Status = StatusRunning
	job.CurrentStep = "Starting conversion"
	c.saveJobStatus(job)
	
	// Create log file
	logFile, err := os.Create(job.LogPath)
	if err != nil {
		job.Status = StatusFailed
		job.Error = fmt.Sprintf("failed to create log file: %v", err)
		c.saveJobStatus(job)
		return
	}
	defer logFile.Close()
	
	// Prepare command
	cmd := exec.CommandContext(ctx, tool.Command, job.Args...)
	
	// Set environment variables
	if len(tool.EnvVars) > 0 {
		cmd.Env = append(os.Environ(), c.flattenEnvVars(tool.EnvVars)...)
	}
	
	// Add user environment
	if len(config.Environment) > 0 {
		cmd.Env = append(cmd.Env, c.flattenEnvVars(config.Environment)...)
	}
	
	// Redirect output
	cmd.Stdout = io.MultiWriter(logFile, os.Stdout)
	cmd.Stderr = io.MultiWriter(logFile, os.Stderr)
	
	// Set working directory
	cmd.Dir = c.tempDir
	
	// Update progress
	job.CurrentStep = "Running conversion command"
	job.Progress = 0.1
	c.saveJobStatus(job)
	
	// Run command
	err = cmd.Run()
	if err != nil {
		job.Status = StatusFailed
		job.Error = fmt.Sprintf("conversion failed: %v", err)
		job.Progress = 0.0
	} else {
		job.Status = StatusCompleted
		job.Progress = 1.0
		job.CurrentStep = "Conversion completed successfully"
	}
	
	// Update end time
	now := time.Now()
	job.EndTime = &now
	
	// Save final status
	c.saveJobStatus(job)
	
	// Clean up temporary files
	c.cleanupTempFiles(job.ID)
}

func (c *ModelConverter) getConversionTool(sourceFormat, targetFormat ModelFormat) (*ConversionTool, error) {
	for _, tool := range c.conversionTools {
		if tool.TargetFormat == targetFormat {
			for _, sf := range tool.SourceFormats {
				if sf == sourceFormat {
					return tool, nil
				}
			}
		}
	}
	
	return nil, fmt.Errorf("no conversion tool found for %s -> %s", sourceFormat, targetFormat)
}

func (c *ModelConverter) buildCommand(tool *ConversionTool, config ConversionConfig) []string {
	args := make([]string, len(tool.Args))
	copy(args, tool.Args)
	
	// Add source and target paths
	args = append(args, "--input", config.SourcePath)
	args = append(args, "--output", config.TargetPath)
	
	// Add quantization options
	if config.Quantization != nil {
		if config.Quantization.Method != "" {
			args = append(args, "--quantize", config.Quantization.Method)
		}
		if config.Quantization.Bits > 0 {
			args = append(args, "--bits", fmt.Sprintf("%d", config.Quantization.Bits))
		}
		if config.Quantization.UseExllama {
			args = append(args, "--exllama")
		}
		if config.Quantization.UseFp16 {
			args = append(args, "--fp16")
		}
	}
	
	// Add optimization options
	if config.Optimization != nil {
		if config.Optimization.RemoveUnusedLayers {
			args = append(args, "--prune")
		}
		if config.Optimization.FuseOperations {
			args = append(args, "--fuse")
		}
		if config.Optimization.OptimizeFor != "" {
			args = append(args, "--optimize-for", config.Optimization.OptimizeFor)
		}
		if config.Optimization.TargetHardware != "" {
			args = append(args, "--target-hardware", config.Optimization.TargetHardware)
		}
		if config.Optimization.DeviceMap != "" {
			args = append(args, "--device-map", config.Optimization.DeviceMap)
		}
	}
	
	return args
}

func (c *ModelConverter) generateTargetPath(sourcePath string, targetFormat ModelFormat) string {
	dir := filepath.Dir(sourcePath)
	name := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	return filepath.Join(dir, fmt.Sprintf("%s.%s", name, targetFormat))
}

func (c *ModelConverter) estimateConversionTime(config ConversionConfig) int64 {
	// Base estimation in minutes
	baseTime := 10
	
	// Adjust based on format complexity
	switch config.TargetFormat {
	case FormatGGUF:
		baseTime += 5
	case FormatGPTQ:
		baseTime += 15
	case FormatAWQ:
		baseTime += 10
	}
	
	// Adjust based on quantization
	if config.Quantization != nil {
		if config.Quantization.Bits < 8 {
			baseTime += 10 // Lower bit quantization takes longer
		}
	}
	
	// Adjust for optimizations
	if config.Optimization != nil {
		if config.Optimization.RemoveUnusedLayers {
			baseTime += 5
		}
		if config.Optimization.FuseOperations {
			baseTime += 5
		}
	}
	
	return int64(baseTime)
}

func (c *ModelConverter) saveJobStatus(job *ConversionJob) error {
	jobDir := filepath.Join(c.baseDir, "jobs")
	os.MkdirAll(jobDir, 0755)
	
	jobPath := filepath.Join(jobDir, fmt.Sprintf("%s.json", job.ID))
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(jobPath, data, 0644)
}

func (c *ModelConverter) cleanupTempFiles(jobID string) {
	tempPath := filepath.Join(c.tempDir, jobID)
	os.RemoveAll(tempPath)
}

func (c *ModelConverter) flattenEnvVars(envVars map[string]string) []string {
	var flattened []string
	for k, v := range envVars {
		flattened = append(flattened, fmt.Sprintf("%s=%s", k, v))
	}
	return flattened
}

func generateJobID() string {
	return fmt.Sprintf("conv_%d", time.Now().UnixNano())
}

// Conversion tool implementations

func initializeAllConversionTools() map[string]*ConversionTool {
	tools := make(map[string]*ConversionTool)
	
	// GGUF conversion (llama.cpp)
	tools["llamacpp_gguf"] = &ConversionTool{
		Name:         "llama.cpp GGUF Converter",
		Command:      "python",
		Args:         []string{"-c", "import sys; sys.path.append('..'); from llama_cpp.convert import convert; convert()"},
		SourceFormats: []ModelFormat{FormatHF, FormatFP16, FormatBF16},
		TargetFormat: FormatGGUF,
		EnvVars: map[string]string{
			"PYTHONPATH": "/usr/local/lib/python3.*/site-packages",
			"HF_HUB_DISABLE_TELEMETRY": "1",
		},
	}
	
	// GPTQ conversion (AutoGPTQ)
	tools["autogptq_gptq"] = &ConversionTool{
		Name:         "AutoGPTQ Converter",
		Command:      "python",
		Args:         []string{"-m", "auto_gptq.cli.convert"},
		SourceFormats: []ModelFormat{FormatHF, FormatFP16, FormatBF16},
		TargetFormat: FormatGPTQ,
		EnvVars: map[string]string{
			"HF_HUB_DISABLE_TELEMETRY": "1",
			"TOKENIZERS_PARALLELISM": "false",
		},
	}
	
	// AWQ conversion (AutoAWQ)
	tools["autoawq_awq"] = &ConversionTool{
		Name:         "AutoAWQ Converter",
		Command:      "python",
		Args:         []string{"-m", "awq.cli.convert"},
		SourceFormats: []ModelFormat{FormatHF, FormatFP16, FormatBF16},
		TargetFormat: FormatAWQ,
		EnvVars: map[string]string{
			"HF_HUB_DISABLE_TELEMETRY": "1",
			"CUDA_VISIBLE_DEVICES": "0",
		},
	}
	
	// BF16/FP16 conversion (transformers)
	tools["transformers_fp16"] = &ConversionTool{
		Name:         "Transformers FP16/BF16 Converter",
		Command:      "python",
		Args:         []string{"-c", "from transformers import AutoModelForCausalLM; AutoModelForCausalLM.from_pretrained('$INPUT').save_pretrained('$OUTPUT')"},
		SourceFormats: []ModelFormat{FormatHF, FormatGGUF},
		TargetFormat: FormatFP16,
		EnvVars: map[string]string{
			"HF_HUB_DISABLE_TELEMETRY": "1",
			"TORCH_DTYPE": "float16",
		},
	}
	
	// MLX conversion (Apple Silicon)
	tools["mlx_convert"] = &ConversionTool{
		Name:         "MLX Converter",
		Command:      "python",
		Args:         []string{"-m", "mlx_lm.convert", "--hf-path", "$INPUT", "--mlx-path", "$OUTPUT"},
		SourceFormats: []ModelFormat{FormatHF, FormatFP16, FormatGGUF},
		TargetFormat: FormatHF, // MLX uses HF format but optimized
		EnvVars: map[string]string{
			"MLX_PATH": "/usr/local/lib/python3.*/site-packages/mlx",
		},
	}
	
	return tools
}

// Helper type for HTTP client
type HTTPClient struct {
	Timeout time.Duration
}

// GetInstalledConversionTools returns which conversion tools are available on the system
func (c *ModelConverter) GetInstalledConversionTools() map[string]bool {
	installed := make(map[string]bool)
	
	for name, tool := range c.conversionTools {
		// Check if command exists
		_, err := exec.LookPath(tool.Command)
		if err == nil {
			installed[name] = true
		} else {
			installed[name] = false
		}
	}
	
	return installed
}

// ValidateConversion checks if a conversion is possible and returns recommendations
func (c *ModelConverter) ValidateConversion(sourceFormat, targetFormat ModelFormat) (*ValidationResult, error) {
	result := &ValidationResult{
		IsPossible: false,
		Warnings:   []string{},
		Recommendations: []string{},
	}
	
	// Check if conversion is supported
	tool, err := c.getConversionTool(sourceFormat, targetFormat)
	if err != nil {
		return result, err
	}
	
	result.IsPossible = true
	
	// Check if tool is installed
	_, err = exec.LookPath(tool.Command)
	if err != nil {
		result.IsPossible = false
		result.Warnings = append(result.Warnings, fmt.Sprintf("Conversion tool '%s' is not installed", tool.Name))
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Install %s to enable this conversion", tool.Name))
	}
	
	// Add format-specific recommendations
	switch targetFormat {
	case FormatGGUF:
		result.Recommendations = append(result.Recommendations, "GGUF is recommended for CPU inference and broad compatibility")
	case FormatGPTQ:
		result.Recommendations = append(result.Recommendations, "GPTQ provides excellent GPU performance with minimal quality loss")
	case FormatAWQ:
		result.Recommendations = append(result.Recommendations, "AWQ offers good balance of performance and quality")
	}
	
	return result, nil
}

// ValidationResult represents the result of conversion validation
type ValidationResult struct {
	IsPossible     bool     `json:"is_possible"`
	Warnings       []string `json:"warnings"`
	Recommendations []string `json:"recommendations"`
	EstimatedTime  int64    `json:"estimated_time"` // in minutes
	RequiredSpace  string   `json:"required_space"` // human-readable size
}

// GetConversionHistory returns conversion history with statistics
func (c *ModelConverter) GetConversionHistory() (*ConversionHistory, error) {
	jobs, err := c.ListConversionJobs()
	if err != nil {
		return nil, err
	}
	
	history := &ConversionHistory{
		TotalConversions: len(jobs),
		SuccessfulConversions: 0,
		FailedConversions: 0,
		AverageConversionTime: 0,
		RecentConversions: []*ConversionJob{},
	}
	
	var totalTime int64
	for _, job := range jobs {
		if job.Status == StatusCompleted {
			history.SuccessfulConversions++
			if job.EndTime != nil {
				totalTime += job.EndTime.Sub(job.StartTime).Milliseconds()
			}
		} else if job.Status == StatusFailed {
			history.FailedConversions++
		}
	}
	
	if history.SuccessfulConversions > 0 {
		history.AverageConversionTime = totalTime / int64(history.SuccessfulConversions) / 1000 / 60 // Convert to minutes
	}
	
	// Get recent conversions (last 10)
	if len(jobs) > 10 {
		history.RecentConversions = jobs[len(jobs)-10:]
	} else {
		history.RecentConversions = jobs
	}
	
	return history, nil
}

// ConversionHistory represents conversion statistics and history
type ConversionHistory struct {
	TotalConversions        int               `json:"total_conversions"`
	SuccessfulConversions   int               `json:"successful_conversions"`
	FailedConversions       int               `json:"failed_conversions"`
	AverageConversionTime   int64             `json:"average_conversion_time"` // in minutes
	RecentConversions        []*ConversionJob  `json:"recent_conversions"`
}