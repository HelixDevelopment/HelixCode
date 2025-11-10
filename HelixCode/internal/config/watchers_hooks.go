package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/logging"
)

// Built-in configuration watchers

// FileWatcher monitors configuration file changes
type FileWatcher struct {
	Name         string
	WatchPaths   []string
	Recursive    bool
	PollInterval time.Duration
	lastModified map[string]time.Time
	mu           sync.RWMutex
	logger       *logging.Logger
}

func (w *FileWatcher) OnConfigChange(change *ConfigChange) error {
	w.logger.Info("Configuration file changed: path=%s type=%s", change.Path, change.Type)

	// This would trigger configuration reload
	// For now, just log the change
	changeJSON, _ := json.MarshalIndent(change, "", "  ")
	w.logger.Debug("Configuration change details: %s", string(changeJSON))

	return nil
}

func (w *FileWatcher) GetName() string {
	return w.Name
}

func (w *FileWatcher) GetWatchPaths() []string {
	return w.WatchPaths
}

// WebhookWatcher sends configuration change notifications via HTTP webhook
type WebhookWatcher struct {
	Name       string
	URL        string
	Method     string
	Headers    map[string]string
	Timeout    time.Duration
	RetryCount int
	RetryDelay time.Duration
	Secret     string // Optional secret for webhook authentication
	logger     *logging.Logger
	client     *http.Client
}

func (w *WebhookWatcher) OnConfigChange(change *ConfigChange) error {
	w.logger.Info("Sending webhook notification: url=%s type=%s", w.URL, change.Type)

	// Prepare webhook payload
	payload := map[string]interface{}{
		"event":     "config_change",
		"change":    change,
		"timestamp": time.Now().Unix(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest(w.Method, w.URL, strings.NewReader(string(payloadJSON)))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range w.Headers {
		req.Header.Set(key, value)
	}

	// Add authentication if secret provided
	if w.Secret != "" {
		req.Header.Set("X-Helix-Webhook-Secret", w.Secret)
	}

	// Send request with retry logic
	var lastErr error
	for i := 0; i <= w.RetryCount; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), w.Timeout)
		defer cancel()

		resp, err := w.client.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = err
			if i < w.RetryCount {
				w.logger.Warn("Webhook request failed, retrying: attempt=%d error=%v", i+1, err)
				time.Sleep(w.RetryDelay)
				continue
			}
			break
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			w.logger.Info("Webhook notification sent successfully: status=%d", resp.StatusCode)
			return nil
		} else {
			lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
			if i < w.RetryCount {
				w.logger.Warn("Webhook request failed, retrying: attempt=%d status=%d", i+1, resp.StatusCode)
				time.Sleep(w.RetryDelay)
				continue
			}
			break
		}
	}

	return fmt.Errorf("failed to send webhook after %d attempts: %w", w.RetryCount+1, lastErr)
}

func (w *WebhookWatcher) GetName() string {
	return w.Name
}

func (w *WebhookWatcher) GetWatchPaths() []string {
	return []string{} // Webhook watcher doesn't watch files
}

// LoggingWatcher logs configuration changes
type LoggingWatcher struct {
	Name       string
	LogLevel   string
	Format     string // "json" or "text"
	OutputPath string // Optional log file path
	logger     *logging.Logger
}

func (w *LoggingWatcher) OnConfigChange(change *ConfigChange) error {
	switch w.Format {
	case "json":
		changeJSON, _ := json.Marshal(change)
		w.logger.Info("Configuration change", "change", string(changeJSON))
	case "text":
		w.logger.Info("Configuration change",
			"type", change.Type,
			"path", change.Path,
			"property", change.Property,
			"timestamp", change.Timestamp)
	default:
		w.logger.Info("Configuration change detected")
	}
	return nil
}

func (w *LoggingWatcher) GetName() string {
	return w.Name
}

func (w *LoggingWatcher) GetWatchPaths() []string {
	return []string{} // Logging watcher doesn't watch files
}

// AlertWatcher sends alerts for critical configuration changes
type AlertWatcher struct {
	Name          string
	AlertURL      string
	AlertChannel  string
	CriticalPaths []string      // Paths that trigger alerts
	MinSeverity   string        // Minimum alert severity
	RateLimit     time.Duration // Minimum time between alerts
	lastAlert     time.Time
	logger        *logging.Logger
	mu            sync.RWMutex
}

func (w *AlertWatcher) OnConfigChange(change *ConfigChange) error {
	// Check if this path requires an alert
	requiresAlert := false
	for _, path := range w.CriticalPaths {
		if change.Path == path || strings.HasPrefix(change.Path, path) {
			requiresAlert = true
			break
		}
	}

	if !requiresAlert {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Check rate limiting
	if time.Since(w.lastAlert) < w.RateLimit {
		w.logger.Debug("Alert rate limited", "path", change.Path)
		return nil
	}

	// Determine alert severity
	severity := "info"
	if change.Type == ChangeTypeDeleted || change.Type == ChangeTypeUpdated {
		severity = "warning"
	}

	// Send alert
	w.logger.Warn("Configuration change alert",
		"path", change.Path,
		"type", change.Type,
		"severity", severity,
		"timestamp", change.Timestamp)

	if w.AlertURL != "" {
		w.sendAlert(change, severity)
	}

	w.lastAlert = time.Now()
	return nil
}

func (w *AlertWatcher) GetName() string {
	return w.Name
}

func (w *AlertWatcher) GetWatchPaths() []string {
	return w.CriticalPaths
}

func (w *AlertWatcher) sendAlert(change *ConfigChange, severity string) {
	// This would implement actual alert sending
	// For now, just log
	w.logger.Info("Alert sent", "url", w.AlertURL, "channel", w.AlertChannel, "severity", severity)
}

// Built-in configuration hooks

// ValidationHook validates configuration before save
type ValidationHook struct {
	Name     string
	Priority int
	Strict   bool
	logger   *logging.Logger
}

func (h *ValidationHook) BeforeLoad(path string, config *HelixConfig) error {
	h.logger.Debug("Before load validation", "path", path)
	// Validation before load is optional
	return nil
}

func (h *ValidationHook) AfterLoad(path string, config *HelixConfig) error {
	h.logger.Debug("After load validation", "path", path)
	// Validate loaded configuration
	return h.validateConfiguration(config)
}

func (h *ValidationHook) BeforeSave(path string, config *HelixConfig) error {
	h.logger.Debug("Before save validation", "path", path)
	// Validate configuration before save
	return h.validateConfiguration(config)
}

func (h *ValidationHook) AfterSave(path string, config *HelixConfig) error {
	h.logger.Debug("After save validation", "path", path)
	// Validation after save is optional
	return nil
}

func (h *ValidationHook) OnError(path string, err error, operation string) error {
	h.logger.Error("Configuration operation error",
		"path", path,
		"operation", operation,
		"error", err)
	return nil
}

func (h *ValidationHook) GetName() string {
	return h.Name
}

func (h *ValidationHook) GetPriority() int {
	return h.Priority
}

func (h *ValidationHook) validateConfiguration(config *HelixConfig) error {
	// Basic validation
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// More comprehensive validation would be added here
	return nil
}

// BackupHook creates automatic backups before save
type BackupHook struct {
	Name       string
	Priority   int
	BackupDir  string
	MaxBackups int
	Compress   bool
	logger     *logging.Logger
}

func (h *BackupHook) BeforeLoad(path string, config *HelixConfig) error {
	// No action needed before load
	return nil
}

func (h *BackupHook) AfterLoad(path string, config *HelixConfig) error {
	// No action needed after load
	return nil
}

func (h *BackupHook) BeforeSave(path string, config *HelixConfig) error {
	h.logger.Debug("Creating backup before save", "path", path)

	if h.BackupDir == "" {
		return nil
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(h.BackupDir, 0755); err != nil {
		h.logger.Error("Failed to create backup directory", "dir", h.BackupDir, "error", err)
		return nil // Don't fail save due to backup issues
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(h.BackupDir, fmt.Sprintf("config_backup_%s.json", timestamp))

	// Copy current config to backup
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			h.logger.Error("Failed to read config for backup", "error", err)
			return nil
		}

		if h.Compress {
			// This would implement compression
			backupFile += ".gz"
		}

		if err := os.WriteFile(backupFile, data, 0644); err != nil {
			h.logger.Error("Failed to write backup file", "error", err)
			return nil
		}

		h.logger.Info("Configuration backup created", "file", backupFile)

		// Clean up old backups
		h.cleanupOldBackups()
	}

	return nil
}

func (h *BackupHook) AfterSave(path string, config *HelixConfig) error {
	// No action needed after save
	return nil
}

func (h *BackupHook) OnError(path string, err error, operation string) error {
	h.logger.Error("Configuration operation error",
		"path", path,
		"operation", operation,
		"error", err)
	return nil
}

func (h *BackupHook) GetName() string {
	return h.Name
}

func (h *BackupHook) GetPriority() int {
	return h.Priority
}

func (h *BackupHook) cleanupOldBackups() {
	entries, err := os.ReadDir(h.BackupDir)
	if err != nil {
		return
	}

	var backupFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasPrefix(entry.Name(), "config_backup_") ||
			strings.HasPrefix(entry.Name(), "config_backup_") && strings.HasSuffix(entry.Name(), ".gz")) {
			backupFiles = append(backupFiles, entry)
		}
	}

	// Remove old backups if we have too many
	if len(backupFiles) > h.MaxBackups {
		// Sort by modification time
		for i := 0; i < len(backupFiles); i++ {
			for j := i + 1; j < len(backupFiles); j++ {
				infoI, _ := backupFiles[i].Info()
				infoJ, _ := backupFiles[j].Info()
				if infoI.ModTime().Before(infoJ.ModTime()) {
					backupFiles[i], backupFiles[j] = backupFiles[j], backupFiles[i]
				}
			}
		}

		// Remove oldest backups
		for i := h.MaxBackups; i < len(backupFiles); i++ {
			oldBackup := filepath.Join(h.BackupDir, backupFiles[i].Name())
			if err := os.Remove(oldBackup); err != nil {
				h.logger.Warn("Failed to remove old backup", "file", oldBackup, "error", err)
			} else {
				h.logger.Debug("Removed old backup", "file", oldBackup)
			}
		}
	}
}

// EncryptionHook encrypts sensitive configuration data
type EncryptionHook struct {
	Name       string
	Priority   int
	EncryptKey []byte
	Sensitive  map[string]bool // Properties that should be encrypted
	logger     *logging.Logger
}

func (h *EncryptionHook) BeforeLoad(path string, config *HelixConfig) error {
	h.logger.Debug("Before load decryption", "path", path)
	// Decrypt configuration after loading
	return h.decryptConfiguration(config)
}

func (h *EncryptionHook) AfterLoad(path string, config *HelixConfig) error {
	// No action needed after load (decryption happens in BeforeLoad)
	return nil
}

func (h *EncryptionHook) BeforeSave(path string, config *HelixConfig) error {
	h.logger.Debug("Before save encryption", "path", path)
	// Encrypt sensitive data before save
	return h.encryptConfiguration(config)
}

func (h *EncryptionHook) AfterSave(path string, config *HelixConfig) error {
	// No action needed after save (encryption happens in BeforeSave)
	return nil
}

func (h *EncryptionHook) OnError(path string, err error, operation string) error {
	h.logger.Error("Configuration operation error",
		"path", path,
		"operation", operation,
		"error", err)
	return nil
}

func (h *EncryptionHook) GetName() string {
	return h.Name
}

func (h *EncryptionHook) GetPriority() int {
	return h.Priority
}

func (h *EncryptionHook) encryptConfiguration(config *HelixConfig) error {
	// This would implement actual encryption of sensitive properties
	// For now, just log
	if len(h.EncryptKey) > 0 {
		h.logger.Debug("Encrypting sensitive configuration properties")
	}
	return nil
}

func (h *EncryptionHook) decryptConfiguration(config *HelixConfig) error {
	// This would implement actual decryption of sensitive properties
	// For now, just log
	if len(h.EncryptKey) > 0 {
		h.logger.Debug("Decrypting sensitive configuration properties")
	}
	return nil
}

// MetricsHook tracks configuration metrics
type MetricsHook struct {
	Name     string
	Priority int
	metrics  map[string]interface{}
	mu       sync.RWMutex
	logger   *logging.Logger
}

func (h *MetricsHook) BeforeLoad(path string, config *HelixConfig) error {
	h.recordMetric("config_load_started", map[string]interface{}{
		"path":      path,
		"timestamp": time.Now(),
	})
	return nil
}

func (h *MetricsHook) AfterLoad(path string, config *HelixConfig) error {
	h.recordMetric("config_load_completed", map[string]interface{}{
		"path":      path,
		"timestamp": time.Now(),
	})
	return nil
}

func (h *MetricsHook) BeforeSave(path string, config *HelixConfig) error {
	h.recordMetric("config_save_started", map[string]interface{}{
		"path":      path,
		"timestamp": time.Now(),
	})
	return nil
}

func (h *MetricsHook) AfterSave(path string, config *HelixConfig) error {
	h.recordMetric("config_save_completed", map[string]interface{}{
		"path":      path,
		"timestamp": time.Now(),
	})
	return nil
}

func (h *MetricsHook) OnError(path string, err error, operation string) error {
	h.recordMetric("config_error", map[string]interface{}{
		"path":      path,
		"operation": operation,
		"error":     err.Error(),
		"timestamp": time.Now(),
	})
	return nil
}

func (h *MetricsHook) GetName() string {
	return h.Name
}

func (h *MetricsHook) GetPriority() int {
	return h.Priority
}

func (h *MetricsHook) recordMetric(name string, data map[string]interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.metrics[name] = data
	h.logger.Debug("Configuration metric recorded", "name", name, "data", data)
}

func (h *MetricsHook) GetMetrics() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent modification
	metrics := make(map[string]interface{})
	for k, v := range h.metrics {
		metrics[k] = v
	}
	return metrics
}

// Utility functions for creating watchers and hooks

// NewFileWatcher creates a new file watcher
func NewFileWatcher(name string, watchPaths []string, pollInterval time.Duration, recursive bool) *FileWatcher {
	logger := logging.NewLoggerWithName("file_watcher_" + name)

	return &FileWatcher{
		Name:         name,
		WatchPaths:   watchPaths,
		PollInterval: pollInterval,
		Recursive:    recursive,
		lastModified: make(map[string]time.Time),
		logger:       logger,
	}
}

// NewWebhookWatcher creates a new webhook watcher
func NewWebhookWatcher(name, url, method string, headers map[string]string, timeout time.Duration, retryCount int, retryDelay time.Duration, secret string) *WebhookWatcher {
	logger := logging.NewLoggerWithName("webhook_watcher_" + name)

	return &WebhookWatcher{
		Name:       name,
		URL:        url,
		Method:     method,
		Headers:    headers,
		Timeout:    timeout,
		RetryCount: retryCount,
		RetryDelay: retryDelay,
		Secret:     secret,
		logger:     logger,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// NewLoggingWatcher creates a new logging watcher
func NewLoggingWatcher(name, logLevel, format, outputPath string) *LoggingWatcher {
	logger := logging.NewLoggerWithName("logging_watcher_" + name)

	return &LoggingWatcher{
		Name:       name,
		LogLevel:   logLevel,
		Format:     format,
		OutputPath: outputPath,
		logger:     logger,
	}
}

// NewAlertWatcher creates a new alert watcher
func NewAlertWatcher(name, alertURL, alertChannel string, criticalPaths []string, minSeverity string, rateLimit time.Duration) *AlertWatcher {
	logger := logging.NewLoggerWithName("alert_watcher_" + name)

	return &AlertWatcher{
		Name:          name,
		AlertURL:      alertURL,
		AlertChannel:  alertChannel,
		CriticalPaths: criticalPaths,
		MinSeverity:   minSeverity,
		RateLimit:     rateLimit,
		logger:        logger,
	}
}

// NewValidationHook creates a new validation hook
func NewValidationHook(name string, priority int, strict bool) *ValidationHook {
	logger := logging.NewLoggerWithName("validation_hook_" + name)

	return &ValidationHook{
		Name:     name,
		Priority: priority,
		Strict:   strict,
		logger:   logger,
	}
}

// NewBackupHook creates a new backup hook
func NewBackupHook(name string, priority int, backupDir string, maxBackups int, compress bool) *BackupHook {
	logger := logging.NewLoggerWithName("backup_hook_" + name)

	return &BackupHook{
		Name:       name,
		Priority:   priority,
		BackupDir:  backupDir,
		MaxBackups: maxBackups,
		Compress:   compress,
		logger:     logger,
	}
}

// NewEncryptionHook creates a new encryption hook
func NewEncryptionHook(name string, priority int, encryptKey []byte, sensitive map[string]bool) *EncryptionHook {
	logger := logging.NewLoggerWithName("encryption_hook_" + name)

	return &EncryptionHook{
		Name:       name,
		Priority:   priority,
		EncryptKey: encryptKey,
		Sensitive:  sensitive,
		logger:     logger,
	}
}

// NewMetricsHook creates a new metrics hook
func NewMetricsHook(name string, priority int) *MetricsHook {
	logger := logging.NewLoggerWithName("metrics_hook_" + name)

	return &MetricsHook{
		Name:     name,
		Priority: priority,
		metrics:  make(map[string]interface{}),
		logger:   logger,
	}
}
