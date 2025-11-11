package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"gopkg.in/yaml.v3"
)

// getRequestID extracts or generates a request ID from the HTTP request
func getRequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ConfigurationAPI provides RESTful API for configuration management
type ConfigurationAPI struct {
	server       *http.Server
	router       *mux.Router
	config       *HelixConfig
	manager      *HelixConfigManager
	validator    *ConfigurationValidator
	migrator     *ConfigurationMigrator
	templateMgr  *ConfigurationTemplateManager
	upgrader     websocket.Upgrader
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// APIResponse represents a standardized API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Code      string      `json:"code,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id"`
}

// ConfigurationEvent represents a configuration change event
type ConfigurationEvent struct {
	Type      string                 `json:"type"` // change, validation, migration
	Path      string                 `json:"path"` // Field path that changed
	OldValue  interface{}            `json:"old_value"`
	NewValue  interface{}            `json:"new_value"`
	Context   map[string]interface{} `json:"context"`
	Timestamp time.Time              `json:"timestamp"`
	User      string                 `json:"user,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// ConfigurationRequest represents a configuration update request
type ConfigurationRequest struct {
	Path      string                 `json:"path"`
	Value     interface{}            `json:"value"`
	Context   map[string]interface{} `json:"context,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	User      string                 `json:"user,omitempty"`
}

// ValidationRequest represents a validation request
type ValidationRequest struct {
	Config    *HelixConfig           `json:"config"`
	Path      string                 `json:"path,omitempty"`
	Rules     []ValidationRule       `json:"rules,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	User      string                 `json:"user,omitempty"`
}

// MigrationRequest represents a migration request
type MigrationRequest struct {
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	DryRun    bool                   `json:"dry_run"`
	Backup    bool                   `json:"backup"`
	Context   map[string]interface{} `json:"context,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	User      string                 `json:"user,omitempty"`
}

// TemplateRequest represents a template request
type TemplateRequest struct {
	TemplateID string                 `json:"template_id"`
	Variables  map[string]interface{} `json:"variables"`
	Context    map[string]interface{} `json:"context,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	User       string                 `json:"user,omitempty"`
}

// ConfigurationAPIServer represents configuration API server configuration
type ConfigurationAPIServer struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	BasePath     string        `json:"base_path"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
	TLS          TLSSettings   `json:"tls"`
	Auth         AuthSettings  `json:"auth"`
	CORS         CORSSettings  `json:"cors"`
	RateLimit    RateLimit     `json:"rate_limit"`
	Metrics      bool          `json:"metrics"`
	HealthCheck  bool          `json:"health_check"`
}

// TLSSettings represents TLS configuration
type TLSSettings struct {
	Enabled            bool     `json:"enabled"`
	CertFile           string   `json:"cert_file"`
	KeyFile            string   `json:"key_file"`
	CAFile             string   `json:"ca_file"`
	ClientAuth         bool     `json:"client_auth"`
	MinVersion         string   `json:"min_version"`
	MaxVersion         string   `json:"max_version"`
	CipherSuites       []string `json:"cipher_suites"`
	PreferServerCipher bool     `json:"prefer_server_cipher"`
}

// AuthSettings represents authentication configuration
type AuthSettings struct {
	Enabled    bool          `json:"enabled"`
	Type       string        `json:"type"` // jwt, basic, oauth, apikey
	Algorithm  string        `json:"algorithm"`
	Secret     string        `json:"secret"`
	Expiration time.Duration `json:"expiration"`
	Renewal    time.Duration `json:"renewal"`
	Issuer     string        `json:"issuer"`
	Audience   []string      `json:"audience"`
	Role       string        `json:"role"`
}

// CORSSettings represents CORS configuration
type CORSSettings struct {
	Enabled          bool     `json:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

// RateLimit represents rate limiting configuration
type RateLimit struct {
	Enabled bool          `json:"enabled"`
	Rate    int           `json:"rate"`  // Requests per second
	Burst   int           `json:"burst"` // Burst size
	Window  time.Duration `json:"window"`
	Methods []string      `json:"methods"`
	Paths   []string      `json:"paths"`
}

// NewConfigurationAPI creates a new configuration API server
func NewConfigurationAPI(config *HelixConfig, apiConfig *ConfigurationAPIServer) (*ConfigurationAPI, error) {
	ctx, cancel := context.WithCancel(context.Background())

	api := &ConfigurationAPI{
		config:      config,
		manager:     globalConfigManager,
		validator:   NewConfigurationValidator(true),
		migrator:    NewConfigurationMigrator(config.Version),
		templateMgr: NewConfigurationTemplateManager(),
		clients:     make(map[*websocket.Conn]bool),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize API server configuration
	if apiConfig == nil {
		apiConfig = getDefaultAPIServerConfig()
	}

	// Setup HTTP server
	api.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", apiConfig.Host, apiConfig.Port),
		Handler:      api.setupRouter(apiConfig),
		ReadTimeout:  apiConfig.ReadTimeout,
		WriteTimeout: apiConfig.WriteTimeout,
		IdleTimeout:  apiConfig.IdleTimeout,
	}

	// Setup WebSocket upgrader
	api.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Configure based on CORS settings
		},
	}

	return api, nil
}

// Start starts the configuration API server
func (api *ConfigurationAPI) Start() error {
	// Start configuration watching
	go api.watchConfigurationChanges()

	// Start WebSocket cleanup
	go api.cleanupWebSockets()

	// Start HTTP server
	return api.server.ListenAndServe()
}

// StartTLS starts the configuration API server with TLS
func (api *ConfigurationAPI) StartTLS() error {
	return api.server.ListenAndServeTLS("", "") // Use configured cert/key files
}

// Stop stops the configuration API server
func (api *ConfigurationAPI) Stop(ctx context.Context) error {
	// Cancel context
	api.cancel()

	// Close all WebSocket connections
	api.clientsMutex.Lock()
	for client := range api.clients {
		client.Close()
	}
	api.clientsMutex.Unlock()

	// Shutdown HTTP server
	return api.server.Shutdown(ctx)
}

// setupRouter sets up the HTTP router
func (api *ConfigurationAPI) setupRouter(config *ConfigurationAPIServer) http.Handler {
	router := mux.NewRouter()

	// Apply CORS middleware
	if config.CORS.Enabled {
		corsMiddleware := cors.New(cors.Options{
			AllowedOrigins:   config.CORS.AllowedOrigins,
			AllowedMethods:   config.CORS.AllowedMethods,
			AllowedHeaders:   config.CORS.AllowedHeaders,
			ExposedHeaders:   config.CORS.ExposedHeaders,
			AllowCredentials: config.CORS.AllowCredentials,
			MaxAge:           config.CORS.MaxAge,
		})
		router.Use(corsMiddleware.Handler)
	}

	// Apply authentication middleware
	// if config.Auth.Enabled {
	// 	router.Use(api.authMiddleware(config.Auth))
	// }

	// Apply rate limiting middleware
	// if config.RateLimit.Enabled {
	// 	router.Use(api.rateLimitMiddleware(config.RateLimit))
	// }

	// Setup routes
	// api.setupRoutes(router, config)

	// Setup health check
	// if config.HealthCheck {
	// 	router.HandleFunc("/health", api.handleHealth).Methods("GET")
	// }

	// Setup metrics
	// if config.Metrics {
	// 	router.HandleFunc("/metrics", api.handleMetrics).Methods("GET")
	// }

	return router
}

// setupRoutes sets up API routes
func (api *ConfigurationAPI) setupRoutes(router *mux.Router, config *ConfigurationAPIServer) {
	// Configuration CRUD operations
	router.HandleFunc("/api/v1/config", api.handleGetConfig).Methods("GET")
	router.HandleFunc("/api/v1/config", api.handleUpdateConfig).Methods("PUT", "PATCH")
	router.HandleFunc("/api/v1/config/validate", api.handleValidateConfig).Methods("POST")

	// Configuration import/export
	router.HandleFunc("/api/v1/config/export", api.handleExportConfig).Methods("GET")
	router.HandleFunc("/api/v1/config/import", api.handleImportConfig).Methods("POST")

	// Configuration management
	router.HandleFunc("/api/v1/config/backup", api.handleBackupConfig).Methods("POST")
	router.HandleFunc("/api/v1/config/restore", api.handleRestoreConfig).Methods("POST")
	router.HandleFunc("/api/v1/config/reset", api.handleResetConfig).Methods("POST")
	router.HandleFunc("/api/v1/config/reload", api.handleReloadConfig).Methods("POST")

	// Field-specific operations
	router.HandleFunc("/api/v1/config/field/{path:.*}", api.handleGetField).Methods("GET")
	router.HandleFunc("/api/v1/config/field/{path:.*}", api.handleUpdateField).Methods("PUT", "PATCH")
	router.HandleFunc("/api/v1/config/field/{path:.*}", api.handleDeleteField).Methods("DELETE")

	// WebSocket endpoints for real-time updates
	router.HandleFunc("/api/v1/config/ws", api.handleWebSocket)
	router.HandleFunc("/api/v1/config/ws/field/{path:.*}", api.handleWebSocketField)

	// Health and status endpoints
	router.HandleFunc("/api/v1/config/health", api.handleHealth).Methods("GET")
	router.HandleFunc("/api/v1/config/status", api.handleStatus).Methods("GET")
}

// API Route Handlers

func (api *ConfigurationAPI) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success:   true,
		Data:      api.config,
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Config  *HelixConfig           `json:"config"`
		Context map[string]interface{} `json:"context,omitempty"`
		User    string                 `json:"user,omitempty"`
	}

	if err := api.readJSONRequest(r, &request); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Validate new configuration
	result := api.validator.Validate(request.Config)
	if !result.Valid {
		api.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Configuration validation failed")
		return
	}

	// Apply configuration update
	if err := api.manager.UpdateConfig(func(config *HelixConfig) {
		*config = *request.Config
	}); err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	// Broadcast configuration change event
	api.broadcastEvent(ConfigurationEvent{
		Type:      "change",
		OldValue:  api.config,
		NewValue:  request.Config,
		Context:   request.Context,
		Timestamp: time.Now(),
		User:      request.User,
		SessionID: api.getSessionID(r),
	})

	// Update local config reference
	api.config = request.Config

	response := APIResponse{
		Success:   true,
		Message:   "Configuration updated successfully",
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	var request ValidationRequest

	if err := api.readJSONRequest(r, &request); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	var result *ValidationResult
	if request.Path != "" {
		result = api.validator.ValidateField(request.Config, request.Path)
	} else {
		result = api.validator.Validate(request.Config)
	}

	response := APIResponse{
		Success:   result.Valid,
		Data:      result,
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	var data []byte
	var contentType string
	var filename string

	switch format {
	case "json":
		data, _ = json.MarshalIndent(api.config, "", "  ")
		contentType = "application/json"
		filename = "helix_config.json"
	case "yaml":
		data, _ = yaml.Marshal(api.config)
		contentType = "application/x-yaml"
		filename = "helix_config.yaml"
	case "toml":
		// Implement TOML export if needed
		api.writeErrorResponse(w, http.StatusBadRequest, "UNSUPPORTED_FORMAT", "TOML format not supported")
		return
	default:
		api.writeErrorResponse(w, http.StatusBadRequest, "INVALID_FORMAT", "Unsupported export format")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(data)
}

func (api *ConfigurationAPI) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("config")
	if err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "NO_FILE", "No configuration file provided")
		return
	}
	defer file.Close()

	// Determine file format
	format := "json"
	if filepath.Ext(header.Filename) == ".yaml" || filepath.Ext(header.Filename) == ".yml" {
		format = "yaml"
	}

	// Read and parse configuration
	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "READ_ERROR", err.Error())
		return
	}

	var newConfig *HelixConfig
	switch format {
	case "json":
		if err := json.Unmarshal(data, &newConfig); err != nil {
			api.writeErrorResponse(w, http.StatusBadRequest, "PARSE_ERROR", err.Error())
			return
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &newConfig); err != nil {
			api.writeErrorResponse(w, http.StatusBadRequest, "PARSE_ERROR", err.Error())
			return
		}
	}

	// Validate configuration
	result := api.validator.Validate(newConfig)
	if !result.Valid {
		api.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Imported configuration validation failed")
		return
	}

	// Apply imported configuration
	if err := api.manager.UpdateConfig(func(config *HelixConfig) {
		*config = *newConfig
	}); err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, "IMPORT_FAILED", err.Error())
		return
	}

	response := APIResponse{
		Success:   true,
		Message:   "Configuration imported successfully",
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleBackupConfig(w http.ResponseWriter, r *http.Request) {
	backupPath := r.URL.Query().Get("path")
	if backupPath == "" {
		backupPath = filepath.Join(os.TempDir(), fmt.Sprintf("helix_config_backup_%s.json", time.Now().Format("20060102_150405")))
	}

	if err := api.manager.BackupConfig(backupPath); err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, "BACKUP_FAILED", err.Error())
		return
	}

	response := APIResponse{
		Success:   true,
		Message:   "Configuration backed up successfully",
		Data:      map[string]string{"path": backupPath},
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleRestoreConfig(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Path    string                 `json:"path"`
		Context map[string]interface{} `json:"context,omitempty"`
		User    string                 `json:"user,omitempty"`
	}

	if err := api.readJSONRequest(r, &request); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Restore configuration from backup
	if err := api.manager.RestoreConfig(request.Path); err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, "RESTORE_FAILED", err.Error())
		return
	}

	// Broadcast configuration change event
	api.broadcastEvent(ConfigurationEvent{
		Type:      "restore",
		Context:   request.Context,
		User:      request.User,
		Timestamp: time.Now(),
	})

	response := APIResponse{
		Success:   true,
		Message:   "Configuration restored successfully",
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleResetConfig(w http.ResponseWriter, r *http.Request) {
	if err := api.manager.ResetToDefaults(); err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, "RESET_FAILED", err.Error())
		return
	}

	response := APIResponse{
		Success:   true,
		Message:   "Configuration reset to defaults",
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	// Capture old config for event
	oldConfig := api.config

	// Reload configuration from disk
	if err := api.manager.ReloadConfig(); err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, "RELOAD_FAILED", err.Error())
		return
	}

	// Update API config reference
	api.config = api.manager.GetConfig()

	// Broadcast configuration change event
	api.broadcastEvent(ConfigurationEvent{
		Type:      "reload",
		OldValue:  oldConfig,
		NewValue:  api.config,
		Timestamp: time.Now(),
	})

	response := APIResponse{
		Success:   true,
		Message:   "Configuration reloaded successfully",
		Data:      api.config,
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

// Field-specific handlers

func (api *ConfigurationAPI) handleGetField(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	value := api.getValueAtPath(api.config, path)
	if value == nil {
		api.writeErrorResponse(w, http.StatusNotFound, "FIELD_NOT_FOUND", fmt.Sprintf("Field not found: %s", path))
		return
	}

	response := APIResponse{
		Success:   true,
		Data:      value,
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleUpdateField(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	var request ConfigurationRequest
	if err := api.readJSONRequest(r, &request); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Get current value
	oldValue := api.getValueAtPath(api.config, path)

	// Validate field update
	result := api.validator.ValidateField(api.config, path)
	if !result.Valid {
		api.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Field validation failed")
		return
	}

	// Apply field update
	if err := api.manager.UpdateConfig(func(config *HelixConfig) {
		api.setValueAtPath(config, path, request.Value)
	}); err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	// Broadcast field change event
	api.broadcastEvent(ConfigurationEvent{
		Type:      "field_change",
		Path:      path,
		OldValue:  oldValue,
		NewValue:  request.Value,
		Context:   request.Context,
		Timestamp: time.Now(),
		User:      request.User,
		SessionID: request.SessionID,
	})

	response := APIResponse{
		Success:   true,
		Message:   "Field updated successfully",
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleDeleteField(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	// Get current value
	oldValue := api.getValueAtPath(api.config, path)

	// Apply field deletion (set to default value)
	if err := api.manager.UpdateConfig(func(config *HelixConfig) {
		api.setValueAtPath(config, path, nil)
	}); err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}

	// Broadcast field change event
	api.broadcastEvent(ConfigurationEvent{
		Type:      "field_delete",
		Path:      path,
		OldValue:  oldValue,
		NewValue:  nil,
		Timestamp: time.Now(),
		SessionID: api.getSessionID(r),
	})

	response := APIResponse{
		Success:   true,
		Message:   "Field deleted successfully",
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

// Helper methods

func (api *ConfigurationAPI) getValueAtPath(obj interface{}, path string) interface{} {
	// Use reflection to navigate path
	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		switch val := current.(type) {
		case map[string]interface{}:
			current = val[part]
		default:
			// Use reflection for struct fields
			r := reflect.ValueOf(current)
			if r.Kind() == reflect.Ptr {
				r = r.Elem()
			}
			if r.Kind() == reflect.Struct {
				field := r.FieldByName(part)
				if field.IsValid() {
					current = field.Interface()
				} else {
					return nil
				}
			} else {
				return nil
			}
		}

		if current == nil {
			return nil
		}
	}

	return current
}

func (api *ConfigurationAPI) setValueAtPath(obj interface{}, path string, value interface{}) error {
	// Use reflection to set value at path
	parts := strings.Split(path, ".")
	current := obj

	// Navigate to parent
	for i, part := range parts {
		isLast := i == len(parts)-1

		switch val := current.(type) {
		case map[string]interface{}:
			if isLast {
				val[part] = value
			} else {
				if val[part] == nil {
					val[part] = make(map[string]interface{})
				}
				current = val[part]
			}
		default:
			// Use reflection for struct fields
			r := reflect.ValueOf(current)
			if r.Kind() == reflect.Ptr {
				r = r.Elem()
			}

			if r.Kind() == reflect.Struct {
				if isLast {
					field := r.FieldByName(part)
					if field.IsValid() && field.CanSet() {
						// Convert value to field type
						converted, err := api.convertValueForField(value, field.Type())
						if err != nil {
							return err
						}
						field.Set(reflect.ValueOf(converted))
					}
				} else {
					field := r.FieldByName(part)
					if field.IsValid() {
						if field.IsNil() {
							// Initialize pointer field
							field.Set(reflect.New(field.Type().Elem()))
						}
						current = field.Interface()
					} else {
						return fmt.Errorf("field not found: %s", part)
					}
				}
			} else {
				return fmt.Errorf("cannot set field on non-struct type")
			}
		}
	}

	return nil
}

func (api *ConfigurationAPI) convertValueForField(value interface{}, targetType reflect.Type) (interface{}, error) {
	if value == nil {
		return reflect.Zero(targetType).Interface(), nil
	}

	sourceType := reflect.TypeOf(value)

	// If types match, return as-is
	if sourceType == targetType {
		return value, nil
	}

	// Handle pointer types
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// Convert based on target type
	switch targetType.Kind() {
	case reflect.String:
		return fmt.Sprintf("%v", value), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if num, ok := api.getNumberValue(value); ok {
			return api.convertToInt(num, targetType), nil
		}
		return nil, fmt.Errorf("cannot convert %v to %v", value, targetType)
	case reflect.Float32, reflect.Float64:
		if num, ok := api.getNumberValue(value); ok {
			return float64(num), nil
		}
		return nil, fmt.Errorf("cannot convert %v to %v", value, targetType)
	case reflect.Bool:
		if str, ok := value.(string); ok {
			return api.parseBool(str), nil
		}
		if b, ok := value.(bool); ok {
			return b, nil
		}
		return nil, fmt.Errorf("cannot convert %v to bool", value)
	default:
		// For complex types, use JSON marshaling
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}

		target := reflect.New(targetType).Interface()
		return target, json.Unmarshal(data, &target)
	}
}

func (api *ConfigurationAPI) getNumberValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

func (api *ConfigurationAPI) convertToInt(value float64, targetType reflect.Type) interface{} {
	switch targetType.Kind() {
	case reflect.Int:
		return int(value)
	case reflect.Int8:
		return int8(value)
	case reflect.Int16:
		return int16(value)
	case reflect.Int32:
		return int32(value)
	case reflect.Int64:
		return int64(value)
	default:
		return int(value)
	}
}

// Health and Status handlers

func (api *ConfigurationAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success:   true,
		Message:   "Configuration API is healthy",
		Data: map[string]interface{}{
			"status": "ok",
			"uptime": time.Since(time.Now()).String(), // Would track actual start time in production
		},
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *ConfigurationAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	api.clientsMutex.RLock()
	connectedClients := len(api.clients)
	api.clientsMutex.RUnlock()

	response := APIResponse{
		Success:   true,
		Data: map[string]interface{}{
			"config_loaded":       api.config != nil,
			"config_path":         api.manager.GetConfigPath(),
			"websocket_clients":   connectedClients,
			"server_running":      true,
		},
		Timestamp: time.Now(),
		RequestID: getRequestID(r),
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

// Utility methods

func (api *ConfigurationAPI) parseBool(s string) bool {
	return strings.ToLower(s) == "true" || s == "1" || s == "yes" || s == "on"
}

func (api *ConfigurationAPI) readJSONRequest(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func (api *ConfigurationAPI) writeJSONResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func (api *ConfigurationAPI) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	response := APIResponse{
		Success:   false,
		Error:     code,
		Message:   message,
		Code:      code,
		Timestamp: time.Now(),
		RequestID: "",
	}

	api.writeJSONResponse(w, statusCode, response)
}

func (api *ConfigurationAPI) getRequestID(r *http.Request) string {
	// Generate or extract request ID
	return r.Header.Get("X-Request-ID")
}

func (api *ConfigurationAPI) getSessionID(r *http.Request) string {
	// Extract session ID from request
	return r.Header.Get("X-Session-ID")
}

// WebSocket handlers and methods

func (api *ConfigurationAPI) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := api.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Add client
	api.clientsMutex.Lock()
	api.clients[conn] = true
	api.clientsMutex.Unlock()

	// Handle WebSocket messages
	for {
		var event ConfigurationEvent
		if err := conn.ReadJSON(&event); err != nil {
			break
		}

		// Process WebSocket message
		api.handleWebSocketMessage(conn, &event)
	}

	// Remove client
	api.clientsMutex.Lock()
	delete(api.clients, conn)
	api.clientsMutex.Unlock()
}

func (api *ConfigurationAPI) handleWebSocketField(w http.ResponseWriter, r *http.Request) {
	// Handle field-specific WebSocket connections
	vars := mux.Vars(r)
	fieldPath := vars["path"]
	_ = fieldPath // TODO: Implement field-specific WebSocket handler

	// Create field-specific WebSocket handler
	// Implementation depends on requirements
}

func (api *ConfigurationAPI) broadcastEvent(event ConfigurationEvent) {
	api.clientsMutex.RLock()
	defer api.clientsMutex.RUnlock()

	for client := range api.clients {
		if err := client.WriteJSON(event); err != nil {
			client.Close()
			delete(api.clients, client)
		}
	}
}

func (api *ConfigurationAPI) handleWebSocketMessage(conn *websocket.Conn, event *ConfigurationEvent) {
	// Process incoming WebSocket messages
	switch event.Type {
	case "subscribe":
		// Handle subscription
	case "unsubscribe":
		// Handle unsubscription
	case "ping":
		// Handle ping
	}
}

func (api *ConfigurationAPI) cleanupWebSockets() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Cleanup inactive connections
			api.clientsMutex.Lock()
			for client := range api.clients {
				if err := client.WriteMessage(websocket.PingMessage, nil); err != nil {
					client.Close()
					delete(api.clients, client)
				}
			}
			api.clientsMutex.Unlock()
		case <-api.ctx.Done():
			return
		}
	}
}

func (api *ConfigurationAPI) watchConfigurationChanges() {
	// Watch for configuration changes and broadcast events
	// Implementation depends on configuration manager
}

// Default configurations

func getDefaultAPIServerConfig() *ConfigurationAPIServer {
	return &ConfigurationAPIServer{
		Host:         "localhost",
		Port:         8081,
		BasePath:     "/api/v1",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		TLS: TLSSettings{
			Enabled: false,
		},
		Auth: AuthSettings{
			Enabled: false,
		},
		CORS: CORSSettings{
			Enabled:          true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
			MaxAge:           86400,
		},
		RateLimit: RateLimit{
			Enabled: false,
		},
		Metrics:     true,
		HealthCheck: true,
	}
}
