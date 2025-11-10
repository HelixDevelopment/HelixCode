package config

import (
	"os"
	"path/filepath"
	"time"
)

// HelixConfig represents the complete HelixCode configuration
// This is the main configuration structure that should be used across all applications
type HelixConfig struct {
	// Metadata
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
	UpdatedBy   string    `json:"updated_by"`

	// Core Application Settings
	Application ApplicationConfig `json:"application"`

	// Core Infrastructure
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	Auth     AuthConfig     `json:"auth"`
	Server   ServerConfig   `json:"server"`

	// Distributed Computing
	Workers WorkersConfig `json:"workers"`
	Tasks   TasksConfig   `json:"tasks"`

	// AI/LLM Configuration
	LLM LLMConfig `json:"llm"`

	// Tools & Features
	Tools     ToolsConfig     `json:"tools"`
	Workflows WorkflowsConfig `json:"workflows"`

	// User Interface
	UI UIConfig `json:"ui"`

	// Communication
	Notifications NotificationsConfig `json:"notifications"`

	// Security & Privacy
	Security SecurityConfig `json:"security"`

	// Development & Debugging
	Development DevelopmentConfig `json:"development"`

	// Platform-specific
	Platform PlatformConfig `json:"platform"`
}

// ApplicationConfig represents core application settings
type ApplicationConfig struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Version     string          `json:"version"`
	Environment string          `json:"environment"` // development, staging, production
	Workspace   WorkspaceConfig `json:"workspace"`
	Session     SessionConfig   `json:"session"`
	Logging     LoggingConfig   `json:"logging"`
	Telemetry   TelemetryConfig `json:"telemetry"`
}

// WorkspaceConfig represents workspace settings
type WorkspaceConfig struct {
	DefaultPath      string            `json:"default_path"`
	AutoSave         bool              `json:"auto_save"`
	AutoSaveInterval int               `json:"auto_save_interval"` // seconds
	BackupEnabled    bool              `json:"backup_enabled"`
	BackupLocation   string            `json:"backup_location"`
	BackupRetention  int               `json:"backup_retention"` // days
	RecentProjects   []string          `json:"recent_projects"`
	CustomPaths      map[string]string `json:"custom_paths"`
}

// SessionConfig represents session management
type SessionConfig struct {
	Timeout            int                      `json:"timeout"` // minutes
	PersistContext     bool                     `json:"persist_context"`
	ContextRetention   int                      `json:"context_retention"` // days
	MaxHistorySize     int                      `json:"max_history_size"`
	AutoResume         bool                     `json:"auto_resume"`
	SavedSessions      []string                 `json:"saved_sessions"`
	ContextCompression ContextCompressionConfig `json:"context_compression"`
}

// ContextCompressionConfig represents context compression settings
type ContextCompressionConfig struct {
	Enabled          bool    `json:"enabled"`
	Threshold        int     `json:"threshold"` // tokens
	Strategy         string  `json:"strategy"`  // semantic, chronological, hybrid
	CompressionRatio float64 `json:"compression_ratio"`
	RetentionPolicy  string  `json:"retention_policy"`
}

// TelemetryConfig represents telemetry and analytics
type TelemetryConfig struct {
	Enabled       bool     `json:"enabled"`
	AnalyticsID   string   `json:"analytics_id"`
	Endpoint      string   `json:"endpoint"`
	DataRetention int      `json:"data_retention"` // days
	Events        []string `json:"events"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type     string `json:"type"` // postgresql, mysql, sqlite
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"` // encrypted
	SSLMode  string `json:"ssl_mode"`

	// Connection pool settings
	MaxConnections     int `json:"max_connections"`
	MaxIdleConnections int `json:"max_idle_connections"`
	ConnectionLifetime int `json:"connection_lifetime"` // seconds

	// Performance settings
	EnableQueryCache bool          `json:"enable_query_cache"`
	QueryTimeout     time.Duration `json:"query_timeout"`

	// Backup and replication
	BackupEnabled bool   `json:"backup_enabled"`
	BackupPath    string `json:"backup_path"`
	Replication   bool   `json:"replication"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"` // encrypted
	Database int    `json:"database"`

	// Connection settings
	MaxConnections     int `json:"max_connections"`
	MaxIdleConnections int `json:"max_idle_connections"`

	// Performance settings
	PoolSize           int           `json:"pool_size"`
	MinIdleConnections int           `json:"min_idle_connections"`
	MaxRetries         int           `json:"max_retries"`
	DialTimeout        time.Duration `json:"dial_timeout"`
	ReadTimeout        time.Duration `json:"read_timeout"`
	WriteTimeout       time.Duration `json:"write_timeout"`

	// Cluster settings
	ClusterEnabled bool     `json:"cluster_enabled"`
	ClusterNodes   []string `json:"cluster_nodes"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	// JWT settings
	JWTSecret          string `json:"jwt_secret"`           // encrypted
	TokenExpiry        int    `json:"token_expiry"`         // seconds
	RefreshTokenExpiry int    `json:"refresh_token_expiry"` // seconds

	// Session settings
	SessionTimeout int  `json:"session_timeout"` // minutes
	SessionExpiry  int  `json:"session_expiry"`  // seconds
	RememberMe     bool `json:"remember_me"`

	// Security settings
	BcryptCost       int  `json:"bcrypt_cost"`
	Require2FA       bool `json:"require_2fa"`
	MaxLoginAttempts int  `json:"max_login_attempts"`
	LockoutDuration  int  `json:"lockout_duration"` // minutes

	// OAuth providers
	OAuthProviders map[string]OAuthProvider `json:"oauth_providers"`

	// RBAC settings
	RBACEnabled bool                  `json:"rbac_enabled"`
	Roles       map[string]RoleConfig `json:"roles"`
	Permissions map[string]Permission `json:"permissions"`
}

// OAuthProvider represents OAuth provider configuration
type OAuthProvider struct {
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"` // encrypted
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	UserInfoURL  string   `json:"user_info_url"`
}

// RoleConfig represents role configuration
type RoleConfig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Inherits    []string `json:"inherits"`
}

// Permission represents a permission
type Permission struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Address         string        `json:"address"`
	Port            int           `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// SSL/TLS
	SSLEnabled  bool   `json:"ssl_enabled"`
	SSLCertFile string `json:"ssl_cert_file"`
	SSLKeyFile  string `json:"ssl_key_file"`

	// CORS
	CORSAllowedOrigins []string `json:"cors_allowed_origins"`
	CORSAllowedMethods []string `json:"cors_allowed_methods"`
	CORSAllowedHeaders []string `json:"cors_allowed_headers"`

	// Rate limiting
	RateLimitEnabled bool   `json:"rate_limit_enabled"`
	RateLimitRate    string `json:"rate_limit_rate"` // e.g., "100/hour"

	// Proxy settings
	ProxyEnabled bool   `json:"proxy_enabled"`
	ProxyURL     string `json:"proxy_url"`
}

// WorkersConfig represents worker configuration
type WorkersConfig struct {
	// Health monitoring
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthTTL           time.Duration `json:"health_ttl"`
	MaxConcurrentTasks  int           `json:"max_concurrent_tasks"`

	// Auto-scaling
	AutoScaling        bool `json:"auto_scaling"`
	MinWorkers         int  `json:"min_workers"`
	MaxWorkers         int  `json:"max_workers"`
	ScaleUpThreshold   int  `json:"scale_up_threshold"`
	ScaleDownThreshold int  `json:"scale_down_threshold"`

	// Resource limits
	CPULimit    float64 `json:"cpu_limit"`    // percentage
	MemoryLimit int64   `json:"memory_limit"` // bytes
	DiskLimit   int64   `json:"disk_limit"`   // bytes

	// Security settings
	IsolationEnabled bool   `json:"isolation_enabled"`
	SandboxType      string `json:"sandbox_type"` // docker, vm, process

	// SSH settings
	DefaultSSHUser     string `json:"default_ssh_user"`
	DefaultSSHPort     int    `json:"default_ssh_port"`
	SSHKeyPath         string `json:"ssh_key_path"`
	SSHKeyPassphrase   string `json:"ssh_key_passphrase"` // encrypted
	AutoInstallEnabled bool   `json:"auto_install_enabled"`
}

// TasksConfig represents task configuration
type TasksConfig struct {
	// Queue settings
	QueueSize     int           `json:"queue_size"`
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	MaxRetryDelay time.Duration `json:"max_retry_delay"`

	// Checkpointing
	CheckpointInterval  time.Duration `json:"checkpoint_interval"`
	CheckpointRetention int           `json:"checkpoint_retention"` // days
	CheckpointStorage   string        `json:"checkpoint_storage"`   // local, s3, database

	// Priority settings
	PriorityLevels  int    `json:"priority_levels"`
	DefaultPriority string `json:"default_priority"`

	// Dependencies
	DependencyResolution bool `json:"dependency_resolution"`
	MaxDependencyDepth   int  `json:"max_dependency_depth"`

	// Cleanup
	CleanupInterval time.Duration `json:"cleanup_interval"`
	TaskRetention   int           `json:"task_retention"` // days
	LogRetention    int           `json:"log_retention"`  // days
}

// LLMConfig represents LLM configuration
type LLMConfig struct {
	// Default settings
	DefaultProvider string  `json:"default_provider"`
	DefaultModel    string  `json:"default_model"`
	MaxTokens       int     `json:"max_tokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"top_p"`

	// Provider configurations
	Providers map[string]LLMProviderConfig `json:"providers"`

	// Model selection
	ModelSelection ModelSelectionConfig `json:"model_selection"`

	// Features
	Features LLMFeaturesConfig `json:"features"`

	// Performance
	Performance LLMPerformanceConfig `json:"performance"`

	// Cost management
	CostManagement LLMCostConfig `json:"cost_management"`
}

// LLMProviderConfig represents LLM provider configuration
type LLMProviderConfig struct {
	Type       string `json:"type"` // anthropic, openai, gemini, etc.
	Enabled    bool   `json:"enabled"`
	Endpoint   string `json:"endpoint"`
	APIKey     string `json:"api_key"` // encrypted
	APIVersion string `json:"api_version"`
	Region     string `json:"region"`

	// Models
	Models       []LLMModel `json:"models"`
	DefaultModel string     `json:"default_model"`

	// Authentication
	AuthType   string            `json:"auth_type"` // api_key, oauth, aws_signature, etc.
	AuthConfig map[string]string `json:"auth_config"`

	// Connection settings
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`

	// Rate limiting
	RateLimitEnabled bool `json:"rate_limit_enabled"`
	RateLimitRPM     int  `json:"rate_limit_rpm"`
	RateLimitTPM     int  `json:"rate_limit_tpm"`

	// Features
	SupportsStreaming bool     `json:"supports_streaming"`
	SupportsVision    bool     `json:"supports_vision"`
	SupportsTools     bool     `json:"supports_tools"`
	SupportsCaching   bool     `json:"supports_caching"`
	Capabilities      []string `json:"capabilities"`

	// Custom parameters
	Parameters map[string]interface{} `json:"parameters"`
}

// LLMModel represents an LLM model
type LLMModel struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	MaxTokens   int     `json:"max_tokens"`
	InputCost   float64 `json:"input_cost"`  // per 1K tokens
	OutputCost  float64 `json:"output_cost"` // per 1K tokens

	// Capabilities
	SupportsStreaming bool     `json:"supports_streaming"`
	SupportsVision    bool     `json:"supports_vision"`
	SupportsTools     bool     `json:"supports_tools"`
	SupportsCaching   bool     `json:"supports_caching"`
	SupportsReasoning bool     `json:"supports_reasoning"`
	Capabilities      []string `json:"capabilities"`

	// Model-specific settings
	TemperatureRange [2]float64 `json:"temperature_range"`
	Recommended      bool       `json:"recommended"`
	Deprecated       bool       `json:"deprecated"`
	Beta             bool       `json:"beta"`
}

// ModelSelectionConfig represents model selection configuration
type ModelSelectionConfig struct {
	Strategy           string   `json:"strategy"` // performance, cost, availability, intelligent
	FallbackEnabled    bool     `json:"fallback_enabled"`
	FallbackChain      []string `json:"fallback_chain"`
	HealthCheck        bool     `json:"health_check"`
	LoadBalancing      bool     `json:"load_balancing"`
	AutoFailover       bool     `json:"auto_failover"`
	PerformanceMetrics bool     `json:"performance_metrics"`
}

// LLMFeaturesConfig represents LLM features configuration
type LLMFeaturesConfig struct {
	// Reasoning
	ReasoningEnabled bool     `json:"reasoning_enabled"`
	ReasoningModels  []string `json:"reasoning_models"`

	// Prompt caching
	CachingEnabled bool          `json:"caching_enabled"`
	CacheStrategy  string        `json:"cache_strategy"` // aggressive, conservative, custom
	CacheTTL       time.Duration `json:"cache_ttl"`
	CacheMaxSize   int64         `json:"cache_max_size"` // bytes

	// Tool calling
	ToolsEnabled       bool          `json:"tools_enabled"`
	ToolTimeout        time.Duration `json:"tool_timeout"`
	MaxConcurrentTools int           `json:"max_concurrent_tools"`

	// Vision
	VisionEnabled      bool     `json:"vision_enabled"`
	VisionMaxImageSize int64    `json:"vision_max_image_size"` // bytes
	VisionFormats      []string `json:"vision_formats"`        // png, jpg, webp, etc.

	// Streaming
	StreamingEnabled   bool          `json:"streaming_enabled"`
	StreamingChunkSize int           `json:"streaming_chunk_size"`
	StreamingTimeout   time.Duration `json:"streaming_timeout"`
}

// LLMPerformanceConfig represents LLM performance configuration
type LLMPerformanceConfig struct {
	// Connection pooling
	PoolEnabled        bool `json:"pool_enabled"`
	MaxConnections     int  `json:"max_connections"`
	MaxIdleConnections int  `json:"max_idle_connections"`

	// Request optimization
	BatchingEnabled bool          `json:"batching_enabled"`
	BatchSize       int           `json:"batch_size"`
	BatchTimeout    time.Duration `json:"batch_timeout"`

	// Caching
	ResponseCacheEnabled bool          `json:"response_cache_enabled"`
	ResponseCacheTTL     time.Duration `json:"response_cache_ttl"`
	ResponseCacheSize    int64         `json:"response_cache_size"` // bytes

	// Monitoring
	MetricsEnabled  bool          `json:"metrics_enabled"`
	MetricsEndpoint string        `json:"metrics_endpoint"`
	MetricsInterval time.Duration `json:"metrics_interval"`
}

// LLMCostConfig represents LLM cost management configuration
type LLMCostConfig struct {
	// Budget management
	BudgetEnabled bool    `json:"budget_enabled"`
	DailyBudget   float64 `json:"daily_budget"`   // USD
	WeeklyBudget  float64 `json:"weekly_budget"`  // USD
	MonthlyBudget float64 `json:"monthly_budget"` // USD

	// Cost tracking
	CostTrackingEnabled bool    `json:"cost_tracking_enabled"`
	CostAlertsEnabled   bool    `json:"cost_alerts_enabled"`
	CostAlertThreshold  float64 `json:"cost_alert_threshold"` // USD

	// Provider preferences
	CostOptimizationEnabled bool     `json:"cost_optimization_enabled"`
	PreferredProviders      []string `json:"preferred_providers"`
	CheapestProviders       []string `json:"cheapest_providers"`

	// Usage limits
	TokenLimitsEnabled bool `json:"token_limits_enabled"`
	DailyTokenLimit    int  `json:"daily_token_limit"`
	MonthlyTokenLimit  int  `json:"monthly_token_limit"`
}

// ToolsConfig represents tools configuration
type ToolsConfig struct {
	// File system tools
	FileSystem FileSystemToolsConfig `json:"file_system"`

	// Shell tools
	Shell ShellToolsConfig `json:"shell"`

	// Browser tools
	Browser BrowserToolsConfig `json:"browser"`

	// Web tools
	Web WebToolsConfig `json:"web"`

	// Voice tools
	Voice VoiceToolsConfig `json:"voice"`

	// Code analysis tools
	CodeAnalysis CodeAnalysisConfig `json:"code_analysis"`

	// Git tools
	Git GitToolsConfig `json:"git"`

	// Multi-file editing
	MultiEdit MultiEditConfig `json:"multi_edit"`

	// Confirmation system
	Confirmation ConfirmationConfig `json:"confirmation"`
}

// FileSystemToolsConfig represents file system tools configuration
type FileSystemToolsConfig struct {
	Enabled      bool     `json:"enabled"`
	AllowedPaths []string `json:"allowed_paths"`
	DeniedPaths  []string `json:"denied_paths"`

	// File operations
	ReadEnabled    bool `json:"read_enabled"`
	WriteEnabled   bool `json:"write_enabled"`
	DeleteEnabled  bool `json:"delete_enabled"`
	ExecuteEnabled bool `json:"execute_enabled"`

	// Limits
	MaxFileSize   int64 `json:"max_file_size"` // bytes
	MaxReadSize   int64 `json:"max_read_size"` // bytes
	MaxFilesPerOp int   `json:"max_files_per_op"`

	// Security
	PermissionChecks bool `json:"permission_checks"`
	SymlinkFollow    bool `json:"symlink_follow"`
	GitAware         bool `json:"git_aware"`
}

// ShellToolsConfig represents shell tools configuration
type ShellToolsConfig struct {
	Enabled         bool     `json:"enabled"`
	AllowedCommands []string `json:"allowed_commands"`
	DeniedCommands  []string `json:"denied_commands"`

	// Execution environment
	SandboxEnabled   bool              `json:"sandbox_enabled"`
	SandboxType      string            `json:"sandbox_type"` // docker, vm, process
	WorkingDirectory string            `json:"working_directory"`
	EnvironmentVars  map[string]string `json:"environment_vars"`

	// Limits
	MaxExecutionTime time.Duration `json:"max_execution_time"`
	MaxMemoryUsage   int64         `json:"max_memory_usage"` // bytes
	MaxProcesses     int           `json:"max_processes"`

	// Security
	RequireConfirmation bool     `json:"require_confirmation"`
	DangerousCommands   []string `json:"dangerous_commands"`
	SudoEnabled         bool     `json:"sudo_enabled"`

	// Logging
	LogEnabled bool   `json:"log_enabled"`
	LogLevel   string `json:"log_level"`
	LogOutput  string `json:"log_output"`
}

// BrowserToolsConfig represents browser tools configuration
type BrowserToolsConfig struct {
	Enabled        bool   `json:"enabled"`
	DefaultBrowser string `json:"default_browser"` // chrome, firefox, safari, edge

	// Browser settings
	Headless   bool       `json:"headless"`
	WindowSize [2]int     `json:"window_size"` // [width, height]
	UserAgent  string     `json:"user_agent"`
	Viewport   [2]float64 `json:"viewport"` // [width, height]

	// Security
	SandboxEnabled bool     `json:"sandbox_enabled"`
	AllowedDomains []string `json:"allowed_domains"`
	DeniedDomains  []string `json:"denied_domains"`
	BlockPopups    bool     `json:"block_popups"`

	// Performance
	Timeout           time.Duration `json:"timeout"`
	PageLoadTimeout   time.Duration `json:"page_load_timeout"`
	WaitTimeout       time.Duration `json:"wait_timeout"`
	MaxConcurrentTabs int           `json:"max_concurrent_tabs"`

	// Features
	ScreenshotEnabled     bool `json:"screenshot_enabled"`
	ConsoleLogEnabled     bool `json:"console_log_enabled"`
	NetworkLoggingEnabled bool `json:"network_logging_enabled"`
	CookieHandlingEnabled bool `json:"cookie_handling_enabled"`
}

// WebToolsConfig represents web tools configuration
type WebToolsConfig struct {
	Enabled       bool     `json:"enabled"`
	SearchEngines []string `json:"search_engines"` // google, bing, duckduckgo, etc.

	// HTTP settings
	UserAgent  string        `json:"user_agent"`
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`

	// Proxy settings
	ProxyEnabled bool   `json:"proxy_enabled"`
	ProxyURL     string `json:"proxy_url"`
	ProxyAuth    string `json:"proxy_auth"`

	// Caching
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`
	CacheSize    int64         `json:"cache_size"` // bytes

	// Rate limiting
	RateLimitEnabled bool `json:"rate_limit_enabled"`
	RateLimitRPS     int  `json:"rate_limit_rps"`

	// Content processing
	MaxContentSize int64    `json:"max_content_size"` // bytes
	AllowedTypes   []string `json:"allowed_types"`
	BlockedDomains []string `json:"blocked_domains"`
}

// VoiceToolsConfig represents voice tools configuration
type VoiceToolsConfig struct {
	Enabled       bool   `json:"enabled"`
	DefaultDevice string `json:"default_device"`

	// Recording settings
	SampleRate int    `json:"sample_rate"` // Hz
	Channels   int    `json:"channels"`    // 1 for mono, 2 for stereo
	BitDepth   int    `json:"bit_depth"`   // 16, 24, 32
	Format     string `json:"format"`      // wav, mp3, flac

	// Voice detection
	VoiceActivityDetection bool          `json:"voice_activity_detection"`
	SilenceThreshold       int           `json:"silence_threshold"` // dB
	MinRecordingDuration   time.Duration `json:"min_recording_duration"`
	MaxRecordingDuration   time.Duration `json:"max_recording_duration"`

	// Transcription
	TranscriptionEnabled  bool   `json:"transcription_enabled"`
	TranscriptionProvider string `json:"transcription_provider"` // openai, google, azure
	TranscriptionLanguage string `json:"transcription_language"`
	TranscriptionModel    string `json:"transcription_model"`

	// Language support
	SupportedLanguages []string `json:"supported_languages"`

	// Privacy
	LocalProcessing bool   `json:"local_processing"`
	StoreRecordings bool   `json:"store_recordings"`
	RecordingPath   string `json:"recording_path"`
}

// CodeAnalysisConfig represents code analysis tools configuration
type CodeAnalysisConfig struct {
	Enabled            bool     `json:"enabled"`
	SupportedLanguages []string `json:"supported_languages"`

	// Analysis features
	SyntaxAnalysis      bool `json:"syntax_analysis"`
	SemanticAnalysis    bool `json:"semantic_analysis"`
	DependencyAnalysis  bool `json:"dependency_analysis"`
	SecurityAnalysis    bool `json:"security_analysis"`
	PerformanceAnalysis bool `json:"performance_analysis"`

	// Tree-sitter settings
	TreeSitterEnabled bool   `json:"tree_sitter_enabled"`
	ParserCachePath   string `json:"parser_cache_path"`
	MaxCacheSize      int64  `json:"max_cache_size"` // bytes

	// Context building
	MaxContextFiles int    `json:"max_context_files"`
	MaxContextSize  int64  `json:"max_context_size"` // bytes
	ContextStrategy string `json:"context_strategy"` // ast, plain, hybrid

	// Indexing
	IndexEnabled        bool          `json:"index_enabled"`
	IndexPath           string        `json:"index_path"`
	IndexUpdateInterval time.Duration `json:"index_update_interval"`
}

// GitToolsConfig represents Git tools configuration
type GitToolsConfig struct {
	Enabled       bool   `json:"enabled"`
	DefaultBranch string `json:"default_branch"`

	// Auto-commit settings
	AutoCommitEnabled     bool   `json:"auto_commit_enabled"`
	AutoCommitMessage     bool   `json:"auto_commit_message"`
	CommitMessageProvider string `json:"commit_message_provider"` // local, llm

	// Staging
	AutoStageTracked bool `json:"auto_stage_tracked"`
	AutoStageNew     bool `json:"auto_stage_new"`

	// Branching
	CreateBranchOnCommit bool   `json:"create_branch_on_commit"`
	BranchNamingPattern  string `json:"branch_naming_pattern"`

	// Integration
	GitHubIntegration    bool `json:"github_integration"`
	GitLabIntegration    bool `json:"gitlab_integration"`
	BitbucketIntegration bool `json:"bitbucket_integration"`

	// Security
	SignedCommits bool   `json:"signed_commits"`
	GPGKeyPath    string `json:"gpg_key_path"`
}

// MultiEditConfig represents multi-file editing configuration
type MultiEditConfig struct {
	Enabled     bool  `json:"enabled"`
	MaxFiles    int   `json:"max_files"`
	MaxFileSize int64 `json:"max_file_size"` // bytes

	// Transaction settings
	Transactional   bool   `json:"transactional"`
	AutoBackup      bool   `json:"auto_backup"`
	BackupPath      string `json:"backup_path"`
	RollbackEnabled bool   `json:"rollback_enabled"`

	// Preview
	PreviewEnabled bool   `json:"preview_enabled"`
	DiffFormat     string `json:"diff_format"` // unified, context, html

	// Conflict handling
	ConflictStrategy string `json:"conflict_strategy"` // abort, merge, manual

	// Performance
	BatchSize    int           `json:"batch_size"`
	BatchTimeout time.Duration `json:"batch_timeout"`
}

// ConfirmationConfig represents confirmation system configuration
type ConfirmationConfig struct {
	Enabled         bool `json:"enabled"`
	InteractiveMode bool `json:"interactive_mode"`

	// Confirmation levels
	Levels map[string]ConfirmationLevel `json:"levels"`

	// Policies
	Policies map[string]ConfirmationPolicy `json:"policies"`

	// Audit
	AuditEnabled   bool   `json:"audit_enabled"`
	AuditLogPath   string `json:"audit_log_path"`
	AuditRetention int    `json:"audit_retention"` // days

	// UI settings
	ShowReason       bool `json:"show_reason"`
	ShowImpact       bool `json:"show_impact"`
	ShowAlternatives bool `json:"show_alternatives"`
}

// ConfirmationLevel represents a confirmation level
type ConfirmationLevel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	Required    bool   `json:"required"`
}

// ConfirmationPolicy represents a confirmation policy
type ConfirmationPolicy struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Pattern     string   `json:"pattern"` // regex or glob
	Level       string   `json:"level"`   // info, warning, danger
	Action      string   `json:"action"`  // prompt, auto-approve, auto-deny
	Exceptions  []string `json:"exceptions"`
}

// WorkflowsConfig represents workflows configuration
type WorkflowsConfig struct {
	Enabled     bool   `json:"enabled"`
	DefaultMode string `json:"default_mode"` // plan, act, auto

	// Plan mode
	PlanMode PlanModeConfig `json:"plan_mode"`

	// Autonomy modes
	Autonomy AutonomyConfig `json:"autonomy"`

	// Snapshots
	Snapshots SnapshotsConfig `json:"snapshots"`

	// Custom workflows
	Workflows map[string]WorkflowConfig `json:"workflows"`

	// Integration
	Integration WorkflowIntegrationConfig `json:"integration"`
}

// PlanModeConfig represents plan mode configuration
type PlanModeConfig struct {
	Enabled             bool `json:"enabled"`
	TwoPhase            bool `json:"two_phase"`
	ShowOptions         bool `json:"show_options"`
	MaxOptions          int  `json:"max_options"`
	RequireConfirmation bool `json:"require_confirmation"`

	// Planning strategy
	Strategy           string `json:"strategy"` // comprehensive, quick, iterative
	MaxPlanComplexity  int    `json:"max_plan_complexity"`
	TaskBreakdown      bool   `json:"task_breakdown"`
	DependencyAnalysis bool   `json:"dependency_analysis"`
}

// AutonomyConfig represents autonomy configuration
type AutonomyConfig struct {
	Enabled      bool   `json:"enabled"`
	DefaultLevel string `json:"default_level"` // full, semi, basic_plus, basic, none

	// Level definitions
	Levels map[string]AutonomyLevel `json:"levels"`

	// Context management
	AutoContext   bool           `json:"auto_context"`
	ContextLimits map[string]int `json:"context_limits"`

	// Safety
	SafetyChecks    bool `json:"safety_checks"`
	FailsafeEnabled bool `json:"failsafe_enabled"`
	EmergencyStop   bool `json:"emergency_stop"`
}

// AutonomyLevel represents an autonomy level
type AutonomyLevel struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Capabilities   []string `json:"capabilities"`
	Limitations    []string `json:"limitations"`
	AutoApprove    []string `json:"auto_approve"`
	RequireConfirm []string `json:"require_confirm"`
	Blocked        []string `json:"blocked"`
}

// SnapshotsConfig represents snapshots configuration
type SnapshotsConfig struct {
	Enabled         bool   `json:"enabled"`
	AutoSnapshot    bool   `json:"auto_snapshot"`
	StorageLocation string `json:"storage_location"`

	// Snapshot settings
	IncludeGitState     bool `json:"include_git_state"`
	IncludeDependencies bool `json:"include_dependencies"`
	IncludeEnvironment  bool `json:"include_environment"`
	IncludeConfig       bool `json:"include_config"`

	// Retention
	RetentionPolicy string `json:"retention_policy"` // count, time, smart
	MaxSnapshots    int    `json:"max_snapshots"`
	RetentionDays   int    `json:"retention_days"`

	// Comparison
	DiffTool     string `json:"diff_tool"`
	ShowChanges  bool   `json:"show_changes"`
	ShowMetadata bool   `json:"show_metadata"`
}

// WorkflowConfig represents a custom workflow
type WorkflowConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Steps       []WorkflowStep    `json:"steps"`
	Triggers    []WorkflowTrigger `json:"triggers"`
	Variables   map[string]string `json:"variables"`
	Enabled     bool              `json:"enabled"`
}

// WorkflowStep represents a workflow step
type WorkflowStep struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // task, condition, parallel, etc.
	Action       string                 `json:"action"`
	Parameters   map[string]interface{} `json:"parameters"`
	Dependencies []string               `json:"dependencies"`
	Timeout      time.Duration          `json:"timeout"`
	Retry        WorkflowRetry          `json:"retry"`
	OnFailure    string                 `json:"on_failure"`
	OnSuccess    string                 `json:"on_success"`
}

// WorkflowTrigger represents a workflow trigger
type WorkflowTrigger struct {
	Type       string                 `json:"type"` // event, schedule, manual
	Condition  string                 `json:"condition"`
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
}

// WorkflowRetry represents workflow retry configuration
type WorkflowRetry struct {
	Enabled     bool          `json:"enabled"`
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     string        `json:"backoff"` // fixed, exponential
}

// WorkflowIntegrationConfig represents workflow integration configuration
type WorkflowIntegrationConfig struct {
	// Git integration
	GitEnabled    bool   `json:"git_enabled"`
	GitAutoPush   bool   `json:"git_auto_push"`
	GitBranchName string `json:"git_branch_name"`

	// CI/CD integration
	CIIntegrationEnabled bool     `json:"ci_integration_enabled"`
	CISystems            []string `json:"ci_systems"` // jenkins, github-actions, gitlab-ci

	// External systems
	WebhooksEnabled bool     `json:"webhooks_enabled"`
	WebhookURLs     []string `json:"webhook_urls"`

	// Notification integration
	NotifyOnStart    bool `json:"notify_on_start"`
	NotifyOnComplete bool `json:"notify_on_complete"`
	NotifyOnFailure  bool `json:"notify_on_failure"`
}

// UIConfig represents user interface configuration
type UIConfig struct {
	// General settings
	Theme      string `json:"theme"`
	Language   string `json:"language"`
	FontFamily string `json:"font_family"`
	FontSize   int    `json:"font_size"`

	// Window settings
	WindowSettings WindowSettings `json:"window_settings"`

	// Editor settings
	Editor EditorSettings `json:"editor"`

	// Terminal settings
	Terminal TerminalSettings `json:"terminal"`

	// Accessibility
	Accessibility AccessibilitySettings `json:"accessibility"`

	// Platform-specific UI
	PlatformUI map[string]PlatformUIConfig `json:"platform_ui"`
}

// WindowSettings represents window settings
type WindowSettings struct {
	DefaultWidth     int    `json:"default_width"`
	DefaultHeight    int    `json:"default_height"`
	MinWidth         int    `json:"min_width"`
	MinHeight        int    `json:"min_height"`
	MaxWidth         int    `json:"max_width"`
	MaxHeight        int    `json:"max_height"`
	RememberSize     bool   `json:"remember_size"`
	RememberPosition bool   `json:"remember_position"`
	StartupPosition  string `json:"startup_position"` // center, last, custom
	DefaultPosition  [2]int `json:"default_position"` // [x, y]
}

// EditorSettings represents editor settings
type EditorSettings struct {
	TabSize        int  `json:"tab_size"`
	InsertSpaces   bool `json:"insert_spaces"`
	WordWrap       bool `json:"word_wrap"`
	LineNumbers    bool `json:"line_numbers"`
	HighlightLine  bool `json:"highlight_line"`
	AutoIndent     bool `json:"auto_indent"`
	ShowWhitespace bool `json:"show_whitespace"`
	ShowMinimap    bool `json:"show_minimap"`

	// Syntax highlighting
	SyntaxHighlighting bool   `json:"syntax_highlighting"`
	ColorScheme        string `json:"color_scheme"`

	// Auto-completion
	AutoCompletion bool `json:"auto_completion"`
	AutoSuggestion bool `json:"auto_suggestion"`
	SnippetEnabled bool `json:"snippet_enabled"`

	// Code folding
	CodeFolding bool `json:"code_folding"`

	// Settings
	AutoSave         bool `json:"auto_save"`
	AutoSaveInterval int  `json:"auto_save_interval"` // seconds

	// Search
	CaseSensitiveSearch bool `json:"case_sensitive_search"`
	RegexSearch         bool `json:"regex_search"`
	IncrementalSearch   bool `json:"incremental_search"`
}

// TerminalSettings represents terminal settings
type TerminalSettings struct {
	Shell           string `json:"shell"`
	ScrollbackLines int    `json:"scrollback_lines"`
	FontSize        int    `json:"font_size"`
	FontFamily      string `json:"font_family"`

	// Colors
	ForegroundColor string `json:"foreground_color"`
	BackgroundColor string `json:"background_color"`
	CursorColor     string `json:"cursor_color"`
	ColorScheme     string `json:"color_scheme"`

	// Features
	Transparency  float64 `json:"transparency"` // 0.0 to 1.0
	Blurriness    float64 `json:"blurriness"`   // 0.0 to 1.0
	AlwaysOnTop   bool    `json:"always_on_top"`
	HideScrollbar bool    `json:"hide_scrollbar"`
	EnableBell    bool    `json:"enable_bell"`

	// Copy/paste
	CopyOnSelect       bool `json:"copy_on_select"`
	PasteOnMiddleClick bool `json:"paste_on_middle_click"`
}

// AccessibilitySettings represents accessibility settings
type AccessibilitySettings struct {
	Enabled            bool `json:"enabled"`
	HighContrast       bool `json:"high_contrast"`
	LargeFonts         bool `json:"large_fonts"`
	ScreenReader       bool `json:"screen_reader"`
	KeyboardNavigation bool `json:"keyboard_navigation"`
	ReduceMotion       bool `json:"reduce_motion"`
	FocusVisible       bool `json:"focus_visible"`
}

// PlatformUIConfig represents platform-specific UI configuration
type PlatformUIConfig struct {
	MenuBar        bool     `json:"menu_bar"`
	ToolBar        bool     `json:"tool_bar"`
	StatusBar      bool     `json:"status_bar"`
	SideBar        bool     `json:"side_bar"`
	FullscreenMode bool     `json:"fullscreen_mode"`
	CompactMode    bool     `json:"compact_mode"`
	TouchOptimized bool     `json:"touch_optimized"`
	Features       []string `json:"features"`
}

// NotificationsConfig represents notifications configuration
type NotificationsConfig struct {
	Enabled bool `json:"enabled"`

	// Channels
	Channels map[string]NotificationChannel `json:"channels"`

	// Rules
	Rules []NotificationRule `json:"rules"`

	// General settings
	DefaultSound   string `json:"default_sound"`
	DefaultUrgency string `json:"default_urgency"` // low, normal, critical

	// Quiet hours
	QuietHoursEnabled bool     `json:"quiet_hours_enabled"`
	QuietHoursStart   string   `json:"quiet_hours_start"` // HH:MM
	QuietHoursEnd     string   `json:"quiet_hours_end"`   // HH:MM
	QuietHoursDays    []string `json:"quiet_hours_days"`  // monday, tuesday, etc.

	// Do not disturb
	DoNotDisturb      bool   `json:"do_not_disturb"`
	DoNotDisturbUntil string `json:"do_not_disturb_until"` // timestamp

	// Aggregation
	AggregateNotifications bool          `json:"aggregate_notifications"`
	MaxAggregatedItems     int           `json:"max_aggregated_items"`
	AggregationTimeout     time.Duration `json:"aggregation_timeout"`
}

// NotificationChannel represents a notification channel
type NotificationChannel struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // desktop, email, slack, telegram, discord
	Enabled  bool                   `json:"enabled"`
	Config   map[string]interface{} `json:"config"`
	Filter   NotificationFilter     `json:"filter"`
	Priority string                 `json:"priority"` // low, medium, high, critical
}

// NotificationFilter represents notification filtering
type NotificationFilter struct {
	IncludeTypes   []string `json:"include_types"`
	ExcludeTypes   []string `json:"exclude_types"`
	IncludeSources []string `json:"include_sources"`
	ExcludeSources []string `json:"exclude_sources"`
	MinPriority    string   `json:"min_priority"`
	MaxPriority    string   `json:"max_priority"`
}

// NotificationRule represents a notification rule
type NotificationRule struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Condition   string   `json:"condition"` // expression
	Channels    []string `json:"channels"`
	Priority    string   `json:"priority"`
	Enabled     bool     `json:"enabled"`
	Template    string   `json:"template"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	// General
	EncryptionEnabled bool   `json:"encryption_enabled"`
	EncryptionKey     string `json:"encryption_key"` // encrypted

	// Authentication
	Authentication AuthenticationConfig `json:"authentication"`

	// Authorization
	Authorization AuthorizationConfig `json:"authorization"`

	// Data protection
	DataProtection DataProtectionConfig `json:"data_protection"`

	// Network security
	Network NetworkSecurityConfig `json:"network"`

	// Auditing
	Audit AuditConfig `json:"audit"`

	// Privacy
	Privacy PrivacyConfig `json:"privacy"`
}

// AuthenticationConfig represents authentication configuration
type AuthenticationConfig struct {
	Methods        []string            `json:"methods"` // password, 2fa, oauth, certificate
	PasswordPolicy PasswordPolicy      `json:"password_policy"`
	TwoFactorAuth  TwoFactorAuthConfig `json:"two_factor_auth"`

	// Session security
	SessionTimeout        time.Duration `json:"session_timeout"`
	MaxConcurrentSessions int           `json:"max_concurrent_sessions"`

	// Lockout policy
	LockoutPolicy LockoutPolicy `json:"lockout_policy"`
}

// PasswordPolicy represents password policy
type PasswordPolicy struct {
	MinLength        int           `json:"min_length"`
	RequireUppercase bool          `json:"require_uppercase"`
	RequireLowercase bool          `json:"require_lowercase"`
	RequireNumbers   bool          `json:"require_numbers"`
	RequireSymbols   bool          `json:"require_symbols"`
	MaxAge           time.Duration `json:"max_age"`
	HistoryCount     int           `json:"history_count"`
}

// TwoFactorAuthConfig represents 2FA configuration
type TwoFactorAuthConfig struct {
	Enabled           bool          `json:"enabled"`
	Methods           []string      `json:"methods"` // totp, sms, email, app
	BackupCodes       bool          `json:"backup_codes"`
	RememberDevice    bool          `json:"remember_device"`
	RememberDeviceTTL time.Duration `json:"remember_device_ttl"`
}

// LockoutPolicy represents lockout policy
type LockoutPolicy struct {
	Enabled         bool          `json:"enabled"`
	MaxAttempts     int           `json:"max_attempts"`
	WindowDuration  time.Duration `json:"window_duration"`
	LockoutDuration time.Duration `json:"lockout_duration"`
	Progressive     bool          `json:"progressive"`
}

// AuthorizationConfig represents authorization configuration
type AuthorizationConfig struct {
	Enabled       bool                  `json:"enabled"`
	DefaultPolicy string                `json:"default_policy"` // allow, deny
	RBAC          bool                  `json:"rbac"`
	Roles         map[string]Role       `json:"roles"`
	Permissions   map[string]Permission `json:"permissions"`
	Policies      []AccessPolicy        `json:"policies"`
}

// Role represents a role
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Inherits    []string `json:"inherits"`
}

// AccessPolicy represents an access policy
type AccessPolicy struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Effect      string `json:"effect"` // allow, deny
	Action      string `json:"action"`
	Resource    string `json:"resource"`
	Condition   string `json:"condition"`
	Priority    int    `json:"priority"`
}

// DataProtectionConfig represents data protection configuration
type DataProtectionConfig struct {
	EncryptionAtRest    bool          `json:"encryption_at_rest"`
	EncryptionInTransit bool          `json:"encryption_in_transit"`
	KeyRotation         time.Duration `json:"key_rotation"`

	// Data retention
	RetentionPolicy RetentionPolicy `json:"retention_policy"`

	// Data masking
	MaskingEnabled bool     `json:"masking_enabled"`
	MaskedFields   []string `json:"masked_fields"`

	// Backup encryption
	BackupEncryption  bool          `json:"backup_encryption"`
	BackupKeyRotation time.Duration `json:"backup_key_rotation"`
}

// RetentionPolicy represents data retention policy
type RetentionPolicy struct {
	Enabled            bool                     `json:"enabled"`
	DefaultRetention   time.Duration            `json:"default_retention"`
	SpecificRetention  map[string]time.Duration `json:"specific_retention"`
	AutoDelete         bool                     `json:"auto_delete"`
	NotificationPeriod time.Duration            `json:"notification_period"`
}

// NetworkSecurityConfig represents network security configuration
type NetworkSecurityConfig struct {
	// Firewall
	FirewallEnabled bool     `json:"firewall_enabled"`
	AllowedIPs      []string `json:"allowed_ips"`
	BlockedIPs      []string `json:"blocked_ips"`
	AllowedPorts    []int    `json:"allowed_ports"`
	BlockedPorts    []int    `json:"blocked_ports"`

	// SSL/TLS
	TLSEnabled   bool     `json:"tls_enabled"`
	TLSVersion   string   `json:"tls_version"`
	CipherSuites []string `json:"cipher_suites"`

	// VPN
	VPNEnabled  bool              `json:"vpn_enabled"`
	VPNProvider string            `json:"vpn_provider"`
	VPNConfig   map[string]string `json:"vpn_config"`
}

// AuditConfig represents audit configuration
type AuditConfig struct {
	Enabled      bool   `json:"enabled"`
	LogLevel     string `json:"log_level"`
	LogPath      string `json:"log_path"`
	MaxLogSize   int64  `json:"max_log_size"`  // bytes
	LogRetention int    `json:"log_retention"` // days

	// Events to audit
	Events []string `json:"events"`

	// Real-time monitoring
	RealTimeEnabled bool     `json:"real_time_enabled"`
	AlertEndpoints  []string `json:"alert_endpoints"`
}

// PrivacyConfig represents privacy configuration
type PrivacyConfig struct {
	// Data collection
	DataCollectionEnabled bool `json:"data_collection_enabled"`
	AnalyticsEnabled      bool `json:"analytics_enabled"`

	// User consent
	ConsentRequired bool   `json:"consent_required"`
	ConsentVersion  string `json:"consent_version"`

	// Data sharing
	DataSharingEnabled bool     `json:"data_sharing_enabled"`
	SharedDataTypes    []string `json:"shared_data_types"`

	// Anonymization
	AnonymizeData       bool   `json:"anonymize_data"`
	AnonymizationMethod string `json:"anonymization_method"`

	// Right to be forgotten
	RightToDeletion    bool   `json:"right_to_deletion"`
	DataDeletionMethod string `json:"data_deletion_method"`
}

// DevelopmentConfig represents development configuration
type DevelopmentConfig struct {
	Enabled     bool   `json:"enabled"`
	Environment string `json:"environment"` // development, testing, staging

	// Debug settings
	Debug DebugConfig `json:"debug"`

	// Testing
	Testing TestingConfig `json:"testing"`

	// Profiling
	Profiling ProfilingConfig `json:"profiling"`

	// Hot reload
	HotReload HotReloadConfig `json:"hot_reload"`

	// Logging
	Logging DevelopmentLoggingConfig `json:"logging"`
}

// DebugConfig represents debug configuration
type DebugConfig struct {
	Enabled      bool     `json:"enabled"`
	Level        string   `json:"level"` // debug, info, warn, error
	Verbose      bool     `json:"verbose"`
	TraceEnabled bool     `json:"trace_enabled"`
	Breakpoints  []string `json:"breakpoints"`

	// Output
	OutputToFile    bool   `json:"output_to_file"`
	OutputToConsole bool   `json:"output_to_console"`
	OutputPath      string `json:"output_path"`

	// Features
	FeatureFlags map[string]bool `json:"feature_flags"`
}

// TestingConfig represents testing configuration
type TestingConfig struct {
	Enabled   bool     `json:"enabled"`
	TestTypes []string `json:"test_types"` // unit, integration, e2e, performance

	// Test execution
	ParallelExecution bool          `json:"parallel_execution"`
	MaxParallelTests  int           `json:"max_parallel_tests"`
	Timeout           time.Duration `json:"timeout"`

	// Test data
	TestDataPath     string `json:"test_data_path"`
	CleanupAfterTest bool   `json:"cleanup_after_test"`

	// Coverage
	CoverageEnabled    bool   `json:"coverage_enabled"`
	CoverageThreshold  int    `json:"coverage_threshold"`
	CoverageReportPath string `json:"coverage_report_path"`

	// Mocking
	MockingEnabled bool     `json:"mocking_enabled"`
	MockProviders  []string `json:"mock_providers"`
}

// ProfilingConfig represents profiling configuration
type ProfilingConfig struct {
	Enabled      bool     `json:"enabled"`
	ProfileTypes []string `json:"profile_types"` // cpu, memory, goroutine, block

	// Output
	OutputPath   string `json:"output_path"`
	OutputFormat string `json:"output_format"` // pprof, svg, pdf

	// Collection
	SamplingRate float64       `json:"sampling_rate"`
	MaxDuration  time.Duration `json:"max_duration"`

	// Analysis
	AutoAnalysis bool   `json:"auto_analysis"`
	AnalysisTool string `json:"analysis_tool"`
}

// HotReloadConfig represents hot reload configuration
type HotReloadConfig struct {
	Enabled        bool     `json:"enabled"`
	WatchPaths     []string `json:"watch_paths"`
	IgnorePatterns []string `json:"ignore_patterns"`

	// Triggers
	TriggerOnFileChange   bool `json:"trigger_on_file_change"`
	TriggerOnConfigChange bool `json:"trigger_on_config_change"`

	// Actions
	RestartServer bool `json:"restart_server"`
	ReloadConfig  bool `json:"reload_config"`
	RefreshUI     bool `json:"refresh_ui"`

	// Delays
	DebounceDelay time.Duration `json:"debounce_delay"`
	RestartDelay  time.Duration `json:"restart_delay"`
}

// DevelopmentLoggingConfig represents development logging configuration
type DevelopmentLoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"` // text, json, structured
	Output string `json:"output"` // console, file, both

	// Modules
	ModuleLevels map[string]string `json:"module_levels"`

	// Advanced
	StructuredLogging bool `json:"structured_logging"`
	CorrelationID     bool `json:"correlation_id"`
	StackTraces       bool `json:"stack_traces"`

	// Performance
	LogPerformance     bool          `json:"log_performance"`
	LogSlowQueries     bool          `json:"log_slow_queries"`
	SlowQueryThreshold time.Duration `json:"slow_query_threshold"`
}

// PlatformConfig represents platform-specific configuration
type PlatformConfig struct {
	// Current platform
	CurrentPlatform string `json:"current_platform"` // desktop, web, mobile, tui

	// Platform settings
	Desktop DesktopConfig `json:"desktop"`
	Web     WebConfig     `json:"web"`
	Mobile  MobileConfig  `json:"mobile"`
	TUI     TUIConfig     `json:"tui"`

	// Specialized platforms
	AuroraOS  AuroraOSConfig  `json:"aurora_os"`
	HarmonyOS HarmonyOSConfig `json:"harmony_os"`

	// Cross-platform
	CrossPlatform CrossPlatformConfig `json:"cross_platform"`
}

// DesktopConfig represents desktop application configuration
type DesktopConfig struct {
	Enabled bool `json:"enabled"`

	// Window management
	AutoStart      bool `json:"auto_start"`
	MinimizeToTray bool `json:"minimize_to_tray"`
	ShowInTaskbar  bool `json:"show_in_taskbar"`

	// System integration
	FileAssociations   map[string]string `json:"file_associations"`
	ContextMenuEnabled bool              `json:"context_menu_enabled"`
	AutoUpdate         bool              `json:"auto_update"`

	// Performance
	HardwareAcceleration bool  `json:"hardware_acceleration"`
	GPUAcceleration      bool  `json:"gpu_acceleration"`
	MemoryLimit          int64 `json:"memory_limit"` // bytes

	// UI scaling
	UIScale float64 `json:"ui_scale"`
	HighDPI bool    `json:"high_dpi"`
}

// WebConfig represents web application configuration
type WebConfig struct {
	Enabled bool `json:"enabled"`

	// Server settings
	Host     string `json:"host"`
	Port     int    `json:"port"`
	BasePath string `json:"base_path"`

	// Static assets
	StaticPath   string        `json:"static_path"`
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`

	// PWA
	PWAEnabled     bool `json:"pwa_enabled"`
	OfflineEnabled bool `json:"offline_enabled"`

	// Security
	CSProtection  bool `json:"csp_protection"`
	XSSProtection bool `json:"xss_protection"`

	// Performance
	CompressionEnabled bool `json:"compression_enabled"`
	MinifyEnabled      bool `json:"minify_enabled"`

	// Features
	RealTimeUpdates  bool `json:"real_time_updates"`
	WebSocketEnabled bool `json:"websocket_enabled"`
}

// MobileConfig represents mobile application configuration
type MobileConfig struct {
	Enabled bool `json:"enabled"`

	// iOS settings
	IOS iOSConfig `json:"ios"`

	// Android settings
	Android AndroidConfig `json:"android"`

	// Cross-platform mobile
	CrossPlatform MobileCrossPlatformConfig `json:"cross_platform"`
}

// iOSConfig represents iOS-specific configuration
type iOSConfig struct {
	Enabled bool `json:"enabled"`

	// App store
	AppStoreConnect bool `json:"app_store_connect"`

	// Features
	PushNotifications bool `json:"push_notifications"`
	BackgroundTasks   bool `json:"background_tasks"`
	WatchKitApp       bool `json:"watchkit_app"`

	// Development
	TeamID          string `json:"team_id"`
	BundleID        string `json:"bundle_id"`
	DevelopmentCert bool   `json:"development_cert"`
}

// AndroidConfig represents Android-specific configuration
type AndroidConfig struct {
	Enabled bool `json:"enabled"`

	// Play store
	GooglePlayConsole bool `json:"google_play_console"`

	// Features
	PushNotifications bool `json:"push_notifications"`
	BackgroundTasks   bool `json:"background_tasks"`
	WearOSApp         bool `json:"wearos_app"`

	// Development
	PackageName    string `json:"package_name"`
	SigningEnabled bool   `json:"signing_enabled"`
	DebugBuild     bool   `json:"debug_build"`
}

// MobileCrossPlatformConfig represents cross-platform mobile configuration
type MobileCrossPlatformConfig struct {
	// Framework
	Framework string `json:"framework"` // react_native, flutter, ionic, gomobile

	// Features
	OfflineFirst bool `json:"offline_first"`
	SyncEnabled  bool `json:"sync_enabled"`

	// Performance
	ImageOptimization bool `json:"image_optimization"`
	LazyLoading       bool `json:"lazy_loading"`

	// Security
	BiometricAuth    bool `json:"biometric_auth"`
	DeviceEncryption bool `json:"device_encryption"`
}

// TUIConfig represents terminal UI configuration
type TUIConfig struct {
	Enabled bool `json:"enabled"`

	// Terminal compatibility
	CompatibilityMode string `json:"compatibility_mode"` // auto, modern, legacy

	// Colors
	ColorScheme string `json:"color_scheme"`
	TrueColor   bool   `json:"true_color"`

	// Mouse support
	MouseEnabled bool `json:"mouse_enabled"`

	// Performance
	RenderFPS  int `json:"render_fps"`
	BufferSize int `json:"buffer_size"`

	// Features
	StatusLine  bool `json:"status_line"`
	TabBar      bool `json:"tab_bar"`
	SplitScreen bool `json:"split_screen"`
}

// AuroraOSConfig represents Aurora OS configuration
type AuroraOSConfig struct {
	Enabled bool `json:"enabled"`

	// Platform-specific features
	SailfishIntegration bool `json:"sailfish_integration"`

	// Store
	StoreIntegration bool `json:"store_integration"`

	// Development
	SDKVersion string `json:"sdk_version"`

	// UI
	NativeUI     bool `json:"native_ui"`
	QtComponents bool `json:"qt_components"`
}

// HarmonyOSConfig represents Harmony OS configuration
type HarmonyOSConfig struct {
	Enabled bool `json:"enabled"`

	// Platform-specific features
	HarmonyServices bool `json:"harmony_services"`

	// Store
	AppGallery bool `json:"app_gallery"`

	// Development
	DevEcoStudio bool   `json:"dev_eco_studio"`
	SDKVersion   string `json:"sdk_version"`

	// UI
	ArkUI bool `json:"ark_ui"`
}

// CrossPlatformConfig represents cross-platform configuration
type CrossPlatformConfig struct {
	// Theme consistency
	ConsistentTheme bool `json:"consistent_theme"`

	// Data synchronization
	SyncConfig bool `json:"sync_config"`
	SyncData   bool `json:"sync_data"`

	// Updates
	UpdateAcrossPlatforms bool `json:"update_across_platforms"`

	// Features
	CommonFeatureSet      bool `json:"common_feature_set"`
	PlatformOptimizations bool `json:"platform_optimizations"`
}

// getDefaultConfig returns a default configuration
func (m *HelixConfigManager) getDefaultConfig() *HelixConfig {
	now := time.Now()

	return &HelixConfig{
		Version:     m.version,
		LastUpdated: now,
		UpdatedBy:   "system",

		Application: ApplicationConfig{
			Name:        "HelixCode",
			Description: "Distributed AI Development Platform",
			Version:     "1.0.0",
			Environment: "development",
			Workspace: WorkspaceConfig{
				DefaultPath:      "~/helixcode",
				AutoSave:         true,
				AutoSaveInterval: 300, // 5 minutes
				BackupEnabled:    true,
				BackupLocation:   "~/helixcode/backups",
				BackupRetention:  30, // days
				RecentProjects:   []string{},
				CustomPaths:      make(map[string]string),
			},
			Session: SessionConfig{
				Timeout:          60, // minutes
				PersistContext:   true,
				ContextRetention: 7, // days
				MaxHistorySize:   1000,
				AutoResume:       true,
				SavedSessions:    []string{},
				ContextCompression: ContextCompressionConfig{
					Enabled:          true,
					Threshold:        10000, // tokens
					Strategy:         "hybrid",
					CompressionRatio: 0.5,
					RetentionPolicy:  "7days",
				},
			},
			Logging: LoggingConfig{
				Level:  "info",
				Format: "text",
				Output: "stdout",
			},
			Telemetry: TelemetryConfig{
				Enabled:       false,
				AnalyticsID:   "",
				Endpoint:      "",
				DataRetention: 30, // days
				Events:        []string{},
			},
		},

		Database: DatabaseConfig{
			Type:               "postgresql",
			Host:               "localhost",
			Port:               5432,
			Database:           "helixcode",
			Username:           "helixcode",
			Password:           "",
			SSLMode:            "disable",
			MaxConnections:     20,
			MaxIdleConnections: 5,
			ConnectionLifetime: 3600, // 1 hour
			EnableQueryCache:   true,
			QueryTimeout:       30 * time.Second,
			BackupEnabled:      true,
			BackupPath:         "~/helixcode/backups/database",
			Replication:        false,
		},

		Redis: RedisConfig{
			Enabled:            true,
			Host:               "localhost",
			Port:               6379,
			Password:           "",
			Database:           0,
			MaxConnections:     20,
			MaxIdleConnections: 5,
			PoolSize:           20,
			MinIdleConnections: 2,
			MaxRetries:         3,
			DialTimeout:        5 * time.Second,
			ReadTimeout:        3 * time.Second,
			WriteTimeout:       3 * time.Second,
			ClusterEnabled:     false,
			ClusterNodes:       []string{},
		},

		Auth: AuthConfig{
			JWTSecret:          "",
			TokenExpiry:        86400,  // 24 hours
			RefreshTokenExpiry: 604800, // 7 days
			SessionTimeout:     30,     // minutes
			SessionExpiry:      604800, // 7 days
			RememberMe:         true,
			BcryptCost:         12,
			Require2FA:         false,
			MaxLoginAttempts:   5,
			LockoutDuration:    15, // minutes
			OAuthProviders:     make(map[string]OAuthProvider),
			RBACEnabled:        true,
			Roles:              make(map[string]RoleConfig),
			Permissions:        make(map[string]Permission),
		},

		Server: ServerConfig{
			Address:            "0.0.0.0",
			Port:               8080,
			ReadTimeout:        30 * time.Second,
			WriteTimeout:       30 * time.Second,
			IdleTimeout:        60 * time.Second,
			ShutdownTimeout:    30 * time.Second,
			SSLEnabled:         false,
			SSLCertFile:        "",
			SSLKeyFile:         "",
			CORSAllowedOrigins: []string{"*"},
			CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			CORSAllowedHeaders: []string{"*"},
			RateLimitEnabled:   false,
			RateLimitRate:      "100/hour",
			ProxyEnabled:       false,
			ProxyURL:           "",
		},

		Workers: WorkersConfig{
			HealthCheckInterval: 30 * time.Second,
			HealthTTL:           120 * time.Second,
			MaxConcurrentTasks:  10,
			AutoScaling:         false,
			MinWorkers:          1,
			MaxWorkers:          10,
			ScaleUpThreshold:    80,                      // percentage
			ScaleDownThreshold:  20,                      // percentage
			CPULimit:            80,                      // percentage
			MemoryLimit:         1024 * 1024 * 1024,      // 1GB
			DiskLimit:           10 * 1024 * 1024 * 1024, // 10GB
			IsolationEnabled:    true,
			SandboxType:         "docker",
			DefaultSSHUser:      "helix",
			DefaultSSHPort:      22,
			SSHKeyPath:          "~/.ssh/id_rsa",
			SSHKeyPassphrase:    "",
			AutoInstallEnabled:  true,
		},

		Tasks: TasksConfig{
			QueueSize:            1000,
			MaxRetries:           3,
			RetryDelay:           5 * time.Second,
			MaxRetryDelay:        60 * time.Second,
			CheckpointInterval:   300 * time.Second, // 5 minutes
			CheckpointRetention:  7,                 // days
			CheckpointStorage:    "local",
			PriorityLevels:       5,
			DefaultPriority:      "normal",
			DependencyResolution: true,
			MaxDependencyDepth:   10,
			CleanupInterval:      3600 * time.Second, // 1 hour
			TaskRetention:        30,                 // days
			LogRetention:         7,                  // days
		},

		LLM: LLMConfig{
			DefaultProvider: "local",
			DefaultModel:    "llama-3.2-3b",
			MaxTokens:       4096,
			Temperature:     0.7,
			TopP:            0.9,
			Providers:       make(map[string]LLMProviderConfig),
			ModelSelection: ModelSelectionConfig{
				Strategy:           "performance",
				FallbackEnabled:    true,
				FallbackChain:      []string{},
				HealthCheck:        true,
				LoadBalancing:      false,
				AutoFailover:       true,
				PerformanceMetrics: true,
			},
			Features: LLMFeaturesConfig{
				ReasoningEnabled:   true,
				ReasoningModels:    []string{"o1-preview", "claude-4-sonnet"},
				CachingEnabled:     true,
				CacheStrategy:      "conservative",
				CacheTTL:           3600 * time.Second, // 1 hour
				CacheMaxSize:       100 * 1024 * 1024,  // 100MB
				ToolsEnabled:       true,
				ToolTimeout:        30 * time.Second,
				MaxConcurrentTools: 5,
				VisionEnabled:      true,
				VisionMaxImageSize: 10 * 1024 * 1024, // 10MB
				VisionFormats:      []string{"png", "jpg", "jpeg", "webp"},
				StreamingEnabled:   true,
				StreamingChunkSize: 1024,
				StreamingTimeout:   60 * time.Second,
			},
			Performance: LLMPerformanceConfig{
				PoolEnabled:          true,
				MaxConnections:       20,
				MaxIdleConnections:   5,
				BatchingEnabled:      false,
				BatchSize:            10,
				BatchTimeout:         5 * time.Second,
				ResponseCacheEnabled: true,
				ResponseCacheTTL:     300 * time.Second, // 5 minutes
				ResponseCacheSize:    50 * 1024 * 1024,  // 50MB
				MetricsEnabled:       true,
				MetricsEndpoint:      "",
				MetricsInterval:      60 * time.Second,
			},
			CostManagement: LLMCostConfig{
				BudgetEnabled:           false,
				DailyBudget:             10.0,  // $10
				WeeklyBudget:            50.0,  // $50
				MonthlyBudget:           200.0, // $200
				CostTrackingEnabled:     true,
				CostAlertsEnabled:       true,
				CostAlertThreshold:      50.0, // $50
				CostOptimizationEnabled: true,
				PreferredProviders:      []string{},
				CheapestProviders:       []string{},
				TokenLimitsEnabled:      false,
				DailyTokenLimit:         100000,
				MonthlyTokenLimit:       3000000,
			},
		},

		Tools: ToolsConfig{
			FileSystem: FileSystemToolsConfig{
				Enabled:          true,
				AllowedPaths:     []string{"~", "/tmp"},
				DeniedPaths:      []string{"/etc", "/usr/bin", "/bin"},
				ReadEnabled:      true,
				WriteEnabled:     true,
				DeleteEnabled:    false,
				ExecuteEnabled:   false,
				MaxFileSize:      100 * 1024 * 1024, // 100MB
				MaxReadSize:      10 * 1024 * 1024,  // 10MB
				MaxFilesPerOp:    100,
				PermissionChecks: true,
				SymlinkFollow:    false,
				GitAware:         true,
			},
			Shell: ShellToolsConfig{
				Enabled:             true,
				AllowedCommands:     []string{"ls", "cd", "pwd", "cat", "echo", "grep", "find"},
				DeniedCommands:      []string{"rm", "sudo", "su", "chmod", "chown"},
				SandboxEnabled:      true,
				SandboxType:         "docker",
				WorkingDirectory:    "/tmp/helix",
				EnvironmentVars:     make(map[string]string),
				MaxExecutionTime:    300 * time.Second, // 5 minutes
				MaxMemoryUsage:      512 * 1024 * 1024, // 512MB
				MaxProcesses:        10,
				RequireConfirmation: true,
				DangerousCommands:   []string{"rm -rf", "sudo", "chmod 777"},
				SudoEnabled:         false,
				LogEnabled:          true,
				LogLevel:            "info",
				LogOutput:           "file",
			},
			Browser: BrowserToolsConfig{
				Enabled:               true,
				DefaultBrowser:        "chrome",
				Headless:              true,
				WindowSize:            [2]int{1920, 1080},
				UserAgent:             "HelixCode/1.0",
				Viewport:              [2]float64{1920, 1080},
				SandboxEnabled:        true,
				AllowedDomains:        []string{},
				DeniedDomains:         []string{},
				BlockPopups:           true,
				Timeout:               30 * time.Second,
				PageLoadTimeout:       30 * time.Second,
				WaitTimeout:           10 * time.Second,
				MaxConcurrentTabs:     5,
				ScreenshotEnabled:     true,
				ConsoleLogEnabled:     true,
				NetworkLoggingEnabled: false,
				CookieHandlingEnabled: true,
			},
			Web: WebToolsConfig{
				Enabled:          true,
				SearchEngines:    []string{"google", "duckduckgo"},
				UserAgent:        "HelixCode/1.0",
				Timeout:          30 * time.Second,
				MaxRetries:       3,
				RetryDelay:       5 * time.Second,
				ProxyEnabled:     false,
				ProxyURL:         "",
				ProxyAuth:        "",
				CacheEnabled:     true,
				CacheTTL:         900 * time.Second, // 15 minutes
				CacheSize:        100 * 1024 * 1024, // 100MB
				RateLimitEnabled: true,
				RateLimitRPS:     10,
				MaxContentSize:   10 * 1024 * 1024, // 10MB
				AllowedTypes:     []string{"text/html", "application/json", "text/plain"},
				BlockedDomains:   []string{},
			},
			Voice: VoiceToolsConfig{
				Enabled:                false,
				DefaultDevice:          "",
				SampleRate:             16000,
				Channels:               1,
				BitDepth:               16,
				Format:                 "wav",
				VoiceActivityDetection: true,
				SilenceThreshold:       -40, // dB
				MinRecordingDuration:   1 * time.Second,
				MaxRecordingDuration:   30 * time.Second,
				TranscriptionEnabled:   true,
				TranscriptionProvider:  "openai",
				TranscriptionLanguage:  "en",
				TranscriptionModel:     "whisper-1",
				SupportedLanguages:     []string{"en", "es", "fr", "de", "it", "pt", "zh", "ja"},
				LocalProcessing:        false,
				StoreRecordings:        false,
				RecordingPath:          "~/.helixcode/recordings",
			},
			CodeAnalysis: CodeAnalysisConfig{
				Enabled:             true,
				SupportedLanguages:  []string{"go", "python", "javascript", "typescript", "java", "c++", "rust"},
				SyntaxAnalysis:      true,
				SemanticAnalysis:    true,
				DependencyAnalysis:  true,
				SecurityAnalysis:    false,
				PerformanceAnalysis: false,
				TreeSitterEnabled:   true,
				ParserCachePath:     "~/.helixcode/cache/parsers",
				MaxCacheSize:        100 * 1024 * 1024, // 100MB
				MaxContextFiles:     50,
				MaxContextSize:      1024 * 1024, // 1MB
				ContextStrategy:     "hybrid",
				IndexEnabled:        true,
				IndexPath:           "~/.helixcode/index",
				IndexUpdateInterval: 3600 * time.Second, // 1 hour
			},
			Git: GitToolsConfig{
				Enabled:               true,
				DefaultBranch:         "main",
				AutoCommitEnabled:     true,
				AutoCommitMessage:     true,
				CommitMessageProvider: "llm",
				AutoStageTracked:      true,
				AutoStageNew:          false,
				CreateBranchOnCommit:  false,
				BranchNamingPattern:   "feature/{}",
				GitHubIntegration:     false,
				GitLabIntegration:     false,
				BitbucketIntegration:  false,
				SignedCommits:         false,
				GPGKeyPath:            "",
			},
			MultiEdit: MultiEditConfig{
				Enabled:          true,
				MaxFiles:         50,
				MaxFileSize:      10 * 1024 * 1024, // 10MB
				Transactional:    true,
				AutoBackup:       true,
				BackupPath:       "~/.helixcode/backups/edits",
				RollbackEnabled:  true,
				PreviewEnabled:   true,
				DiffFormat:       "unified",
				ConflictStrategy: "manual",
				BatchSize:        10,
				BatchTimeout:     30 * time.Second,
			},
			Confirmation: ConfirmationConfig{
				Enabled:         true,
				InteractiveMode: true,
				Levels: map[string]ConfirmationLevel{
					"info": {
						Name:        "Info",
						Description: "Informational operations",
						Color:       "blue",
						Icon:        "ℹ️",
						Required:    false,
					},
					"warning": {
						Name:        "Warning",
						Description: "Potentially risky operations",
						Color:       "yellow",
						Icon:        "⚠️",
						Required:    true,
					},
					"danger": {
						Name:        "Danger",
						Description: "Dangerous operations",
						Color:       "red",
						Icon:        "🚨",
						Required:    true,
					},
				},
				Policies: map[string]ConfirmationPolicy{
					"file_delete": {
						Name:        "File Deletion",
						Description: "Confirm file deletion",
						Pattern:     "**/*",
						Level:       "danger",
						Action:      "prompt",
						Exceptions:  []string{"*.tmp", "*.cache"},
					},
					"shell_sudo": {
						Name:        "Sudo Commands",
						Description: "Confirm sudo usage",
						Pattern:     "sudo *",
						Level:       "danger",
						Action:      "prompt",
						Exceptions:  []string{},
					},
				},
				AuditEnabled:     true,
				AuditLogPath:     "~/.helixcode/logs/audit.log",
				AuditRetention:   30, // days
				ShowReason:       true,
				ShowImpact:       true,
				ShowAlternatives: true,
			},
		},

		Workflows: WorkflowsConfig{
			Enabled:     true,
			DefaultMode: "plan",
			PlanMode: PlanModeConfig{
				Enabled:             true,
				TwoPhase:            true,
				ShowOptions:         true,
				MaxOptions:          5,
				RequireConfirmation: true,
				Strategy:            "comprehensive",
				MaxPlanComplexity:   10,
				TaskBreakdown:       true,
				DependencyAnalysis:  true,
			},
			Autonomy: AutonomyConfig{
				Enabled:      true,
				DefaultLevel: "basic_plus",
				Levels: map[string]AutonomyLevel{
					"full": {
						Name:           "Full Auto",
						Description:    "Complete automation",
						Capabilities:   []string{"auto_execute", "auto_approve", "auto_commit"},
						Limitations:    []string{"no_human_review"},
						AutoApprove:    []string{"file_edit", "tool_execution"},
						RequireConfirm: []string{"system_change", "security_sensitive"},
						Blocked:        []string{"data_destruction"},
					},
					"semi": {
						Name:           "Semi Auto",
						Description:    "Balanced automation",
						Capabilities:   []string{"auto_context", "auto_plan"},
						Limitations:    []string{"manual_apply"},
						AutoApprove:    []string{"context_building", "analysis"},
						RequireConfirm: []string{"file_edit", "system_change"},
						Blocked:        []string{},
					},
					"basic_plus": {
						Name:           "Basic Plus",
						Description:    "Smart semi-automation",
						Capabilities:   []string{"smart_suggestions", "auto_complete"},
						Limitations:    []string{},
						AutoApprove:    []string{"safe_operations"},
						RequireConfirm: []string{"file_modification", "system_changes"},
						Blocked:        []string{},
					},
					"basic": {
						Name:           "Basic",
						Description:    "Manual workflow",
						Capabilities:   []string{},
						Limitations:    []string{"manual_confirmation_required"},
						AutoApprove:    []string{},
						RequireConfirm: []string{"all_operations"},
						Blocked:        []string{},
					},
					"none": {
						Name:           "None",
						Description:    "Step-by-step control",
						Capabilities:   []string{},
						Limitations:    []string{"step_by_step_only"},
						AutoApprove:    []string{},
						RequireConfirm: []string{"every_step"},
						Blocked:        []string{},
					},
				},
				AutoContext: true,
				ContextLimits: map[string]int{
					"max_tokens": 8192,
					"max_files":  20,
				},
				SafetyChecks:    true,
				FailsafeEnabled: true,
				EmergencyStop:   true,
			},
			Snapshots: SnapshotsConfig{
				Enabled:             true,
				AutoSnapshot:        true,
				StorageLocation:     "~/.helixcode/snapshots",
				IncludeGitState:     true,
				IncludeDependencies: true,
				IncludeEnvironment:  true,
				IncludeConfig:       true,
				RetentionPolicy:     "smart",
				MaxSnapshots:        100,
				RetentionDays:       30,
				DiffTool:            "git",
				ShowChanges:         true,
				ShowMetadata:        true,
			},
			Workflows: make(map[string]WorkflowConfig),
			Integration: WorkflowIntegrationConfig{
				GitEnabled:           true,
				GitAutoPush:          false,
				GitBranchName:        "helix-workflow",
				CIIntegrationEnabled: false,
				CISystems:            []string{},
				WebhooksEnabled:      false,
				WebhookURLs:          []string{},
				NotifyOnStart:        false,
				NotifyOnComplete:     false,
				NotifyOnFailure:      false,
			},
		},

		UI: UIConfig{
			Theme:      "dark",
			Language:   "en",
			FontFamily: "SF Mono",
			FontSize:   14,
			WindowSettings: WindowSettings{
				DefaultWidth:     1200,
				DefaultHeight:    800,
				MinWidth:         800,
				MinHeight:        600,
				MaxWidth:         0, // 0 = unlimited
				MaxHeight:        0, // 0 = unlimited
				RememberSize:     true,
				RememberPosition: true,
				StartupPosition:  "center",
				DefaultPosition:  [2]int{0, 0},
			},
			Editor: EditorSettings{
				TabSize:             4,
				InsertSpaces:        true,
				WordWrap:            true,
				LineNumbers:         true,
				HighlightLine:       true,
				AutoIndent:          true,
				ShowWhitespace:      false,
				ShowMinimap:         false,
				SyntaxHighlighting:  true,
				ColorScheme:         "dark",
				AutoCompletion:      true,
				AutoSuggestion:      true,
				SnippetEnabled:      true,
				CodeFolding:         false,
				AutoSave:            true,
				AutoSaveInterval:    300, // 5 minutes
				CaseSensitiveSearch: false,
				RegexSearch:         true,
				IncrementalSearch:   true,
			},
			Terminal: TerminalSettings{
				Shell:              "/bin/bash",
				ScrollbackLines:    10000,
				FontSize:           12,
				FontFamily:         "SF Mono",
				ForegroundColor:    "#ffffff",
				BackgroundColor:    "#000000",
				CursorColor:        "#ffffff",
				ColorScheme:        "dark",
				Transparency:       0.0,
				Blurriness:         0.0,
				AlwaysOnTop:        false,
				HideScrollbar:      false,
				EnableBell:         true,
				CopyOnSelect:       false,
				PasteOnMiddleClick: false,
			},
			Accessibility: AccessibilitySettings{
				Enabled:            false,
				HighContrast:       false,
				LargeFonts:         false,
				ScreenReader:       false,
				KeyboardNavigation: true,
				ReduceMotion:       false,
				FocusVisible:       true,
			},
			PlatformUI: map[string]PlatformUIConfig{
				"desktop": {
					MenuBar:        true,
					ToolBar:        true,
					StatusBar:      true,
					SideBar:        true,
					FullscreenMode: false,
					CompactMode:    false,
					TouchOptimized: false,
					Features:       []string{"file_browser", "terminal", "editor"},
				},
				"web": {
					MenuBar:        true,
					ToolBar:        true,
					StatusBar:      true,
					SideBar:        true,
					FullscreenMode: true,
					CompactMode:    false,
					TouchOptimized: true,
					Features:       []string{"responsive_design", "pwa"},
				},
				"mobile": {
					MenuBar:        false,
					ToolBar:        true,
					StatusBar:      true,
					SideBar:        false,
					FullscreenMode: true,
					CompactMode:    true,
					TouchOptimized: true,
					Features:       []string{"gestures", "offline_support"},
				},
				"tui": {
					MenuBar:        false,
					ToolBar:        false,
					StatusBar:      true,
					SideBar:        false,
					FullscreenMode: true,
					CompactMode:    true,
					TouchOptimized: false,
					Features:       []string{"keyboard_shortcuts", "mouse_support"},
				},
			},
		},

		Notifications: NotificationsConfig{
			Enabled: true,
			Channels: map[string]NotificationChannel{
				"desktop": {
					Name:    "Desktop Notifications",
					Type:    "desktop",
					Enabled: true,
					Config:  map[string]interface{}{},
					Filter: NotificationFilter{
						IncludeTypes:   []string{},
						ExcludeTypes:   []string{},
						IncludeSources: []string{},
						ExcludeSources: []string{},
						MinPriority:    "low",
						MaxPriority:    "critical",
					},
					Priority: "normal",
				},
				"email": {
					Name:    "Email Notifications",
					Type:    "email",
					Enabled: false,
					Config: map[string]interface{}{
						"smtp_server": "",
						"smtp_port":   587,
						"username":    "",
						"password":    "",
						"from":        "",
						"to":          []string{},
					},
					Filter: NotificationFilter{
						IncludeTypes:   []string{},
						ExcludeTypes:   []string{},
						IncludeSources: []string{},
						ExcludeSources: []string{},
						MinPriority:    "medium",
						MaxPriority:    "critical",
					},
					Priority: "medium",
				},
				"slack": {
					Name:    "Slack Notifications",
					Type:    "slack",
					Enabled: false,
					Config: map[string]interface{}{
						"webhook_url": "",
						"channel":     "#general",
						"username":    "HelixCode",
					},
					Filter: NotificationFilter{
						IncludeTypes:   []string{},
						ExcludeTypes:   []string{},
						IncludeSources: []string{},
						ExcludeSources: []string{},
						MinPriority:    "medium",
						MaxPriority:    "critical",
					},
					Priority: "medium",
				},
			},
			Rules: []NotificationRule{
				{
					Name:        "Critical Errors",
					Description: "Notify on critical errors",
					Condition:   "priority == 'critical' && type == 'error'",
					Channels:    []string{"desktop", "email"},
					Priority:    "critical",
					Enabled:     true,
					Template:    "Critical Error: {{.Message}}",
				},
				{
					Name:        "Task Completion",
					Description: "Notify when tasks complete",
					Condition:   "type == 'task_complete'",
					Channels:    []string{"desktop"},
					Priority:    "normal",
					Enabled:     true,
					Template:    "Task Completed: {{.TaskName}}",
				},
			},
			DefaultSound:           "default",
			DefaultUrgency:         "normal",
			QuietHoursEnabled:      false,
			QuietHoursStart:        "22:00",
			QuietHoursEnd:          "08:00",
			QuietHoursDays:         []string{"saturday", "sunday"},
			DoNotDisturb:           false,
			DoNotDisturbUntil:      "",
			AggregateNotifications: true,
			MaxAggregatedItems:     5,
			AggregationTimeout:     300 * time.Second, // 5 minutes
		},

		Security: SecurityConfig{
			EncryptionEnabled: true,
			EncryptionKey:     "",
			Authentication: AuthenticationConfig{
				Methods: []string{"password", "2fa"},
				PasswordPolicy: PasswordPolicy{
					MinLength:        8,
					RequireUppercase: true,
					RequireLowercase: true,
					RequireNumbers:   true,
					RequireSymbols:   false,
					MaxAge:           90 * 24 * time.Hour, // 90 days
					HistoryCount:     5,
				},
				TwoFactorAuth: TwoFactorAuthConfig{
					Enabled:           false,
					Methods:           []string{"totp", "sms"},
					BackupCodes:       true,
					RememberDevice:    true,
					RememberDeviceTTL: 30 * 24 * time.Hour, // 30 days
				},
				SessionTimeout:        30 * time.Minute,
				MaxConcurrentSessions: 3,
				LockoutPolicy: LockoutPolicy{
					Enabled:         true,
					MaxAttempts:     5,
					WindowDuration:  15 * time.Minute,
					LockoutDuration: 15 * time.Minute,
					Progressive:     true,
				},
			},
			Authorization: AuthorizationConfig{
				Enabled:       true,
				DefaultPolicy: "deny",
				RBAC:          true,
				Roles:         make(map[string]Role),
				Permissions:   make(map[string]Permission),
				Policies:      []AccessPolicy{},
			},
			DataProtection: DataProtectionConfig{
				EncryptionAtRest:    true,
				EncryptionInTransit: true,
				KeyRotation:         90 * 24 * time.Hour, // 90 days
				RetentionPolicy: RetentionPolicy{
					Enabled:            true,
					DefaultRetention:   365 * 24 * time.Hour, // 1 year
					SpecificRetention:  make(map[string]time.Duration),
					AutoDelete:         false,
					NotificationPeriod: 30 * 24 * time.Hour, // 30 days
				},
				MaskingEnabled:    false,
				MaskedFields:      []string{},
				BackupEncryption:  true,
				BackupKeyRotation: 90 * 24 * time.Hour, // 90 days
			},
			Network: NetworkSecurityConfig{
				FirewallEnabled: false,
				AllowedIPs:      []string{},
				BlockedIPs:      []string{},
				AllowedPorts:    []int{8080, 22},
				BlockedPorts:    []int{},
				TLSEnabled:      false,
				TLSVersion:      "1.3",
				CipherSuites:    []string{},
				VPNEnabled:      false,
				VPNProvider:     "",
				VPNConfig:       make(map[string]string),
			},
			Audit: AuditConfig{
				Enabled:         true,
				LogLevel:        "info",
				LogPath:         "~/.helixcode/logs/audit.log",
				MaxLogSize:      100 * 1024 * 1024, // 100MB
				LogRetention:    90,                // days
				Events:          []string{"login", "logout", "config_change", "task_execution"},
				RealTimeEnabled: false,
				AlertEndpoints:  []string{},
			},
			Privacy: PrivacyConfig{
				DataCollectionEnabled: false,
				AnalyticsEnabled:      false,
				ConsentRequired:       true,
				ConsentVersion:        "1.0",
				DataSharingEnabled:    false,
				SharedDataTypes:       []string{},
				AnonymizeData:         true,
				AnonymizationMethod:   "hashing",
				RightToDeletion:       true,
				DataDeletionMethod:    "secure_erase",
			},
		},

		Development: DevelopmentConfig{
			Enabled:     false,
			Environment: "development",
			Debug: DebugConfig{
				Enabled:         false,
				Level:           "debug",
				Verbose:         false,
				TraceEnabled:    false,
				Breakpoints:     []string{},
				OutputToFile:    true,
				OutputToConsole: true,
				OutputPath:      "~/.helixcode/logs/debug.log",
				FeatureFlags:    make(map[string]bool),
			},
			Testing: TestingConfig{
				Enabled:            true,
				TestTypes:          []string{"unit", "integration"},
				ParallelExecution:  true,
				MaxParallelTests:   10,
				Timeout:            30 * time.Minute,
				TestDataPath:       "~/.helixcode/test_data",
				CleanupAfterTest:   true,
				CoverageEnabled:    true,
				CoverageThreshold:  80,
				CoverageReportPath: "~/.helixcode/coverage",
				MockingEnabled:     true,
				MockProviders:      []string{},
			},
			Profiling: ProfilingConfig{
				Enabled:      false,
				ProfileTypes: []string{"cpu", "memory"},
				OutputPath:   "~/.helixcode/profiles",
				OutputFormat: "pprof",
				SamplingRate: 100, // Hz
				MaxDuration:  5 * time.Minute,
				AutoAnalysis: true,
				AnalysisTool: "go tool pprof",
			},
			HotReload: HotReloadConfig{
				Enabled:               true,
				WatchPaths:            []string{"./config", "./internal"},
				IgnorePatterns:        []string{"*.tmp", "*.log"},
				TriggerOnFileChange:   true,
				TriggerOnConfigChange: true,
				RestartServer:         true,
				ReloadConfig:          true,
				RefreshUI:             true,
				DebounceDelay:         1 * time.Second,
				RestartDelay:          5 * time.Second,
			},
			Logging: DevelopmentLoggingConfig{
				Level:              "debug",
				Format:             "structured",
				Output:             "both",
				ModuleLevels:       make(map[string]string),
				StructuredLogging:  true,
				CorrelationID:      true,
				StackTraces:        true,
				LogPerformance:     true,
				LogSlowQueries:     true,
				SlowQueryThreshold: 1 * time.Second,
			},
		},

		Platform: PlatformConfig{
			CurrentPlatform: "desktop",
			Desktop: DesktopConfig{
				Enabled:              true,
				AutoStart:            false,
				MinimizeToTray:       true,
				ShowInTaskbar:        true,
				FileAssociations:     map[string]string{},
				ContextMenuEnabled:   true,
				AutoUpdate:           true,
				HardwareAcceleration: true,
				GPUAcceleration:      true,
				MemoryLimit:          2 * 1024 * 1024 * 1024, // 2GB
				UIScale:              1.0,
				HighDPI:              true,
			},
			Web: WebConfig{
				Enabled:            true,
				Host:               "localhost",
				Port:               3000,
				BasePath:           "/",
				StaticPath:         "./static",
				CacheEnabled:       true,
				CacheTTL:           3600 * time.Second, // 1 hour
				PWAEnabled:         true,
				OfflineEnabled:     true,
				CSProtection:       true,
				XSSProtection:      true,
				CompressionEnabled: true,
				MinifyEnabled:      true,
				RealTimeUpdates:    true,
				WebSocketEnabled:   true,
			},
			Mobile: MobileConfig{
				Enabled: true,
				IOS: iOSConfig{
					Enabled:           true,
					AppStoreConnect:   false,
					PushNotifications: false,
					BackgroundTasks:   true,
					WatchKitApp:       false,
					TeamID:            "",
					BundleID:          "dev.helix.code",
					DevelopmentCert:   false,
				},
				Android: AndroidConfig{
					Enabled:           true,
					GooglePlayConsole: false,
					PushNotifications: false,
					BackgroundTasks:   true,
					WearOSApp:         false,
					PackageName:       "dev.helix.code",
					SigningEnabled:    true,
					DebugBuild:        true,
				},
				CrossPlatform: MobileCrossPlatformConfig{
					Framework:         "gomobile",
					OfflineFirst:      true,
					SyncEnabled:       true,
					ImageOptimization: true,
					LazyLoading:       true,
					BiometricAuth:     true,
					DeviceEncryption:  true,
				},
			},
			TUI: TUIConfig{
				Enabled:           true,
				CompatibilityMode: "auto",
				ColorScheme:       "dark",
				TrueColor:         true,
				MouseEnabled:      true,
				RenderFPS:         60,
				BufferSize:        10000,
				StatusLine:        true,
				TabBar:            true,
				SplitScreen:       true,
			},
			AuroraOS: AuroraOSConfig{
				Enabled:             false,
				SailfishIntegration: true,
				StoreIntegration:    false,
				SDKVersion:          "4.4",
				NativeUI:            true,
				QtComponents:        true,
			},
			HarmonyOS: HarmonyOSConfig{
				Enabled:         false,
				HarmonyServices: true,
				AppGallery:      false,
				DevEcoStudio:    false,
				SDKVersion:      "5.0",
				ArkUI:           true,
			},
			CrossPlatform: CrossPlatformConfig{
				ConsistentTheme:       true,
				SyncConfig:            true,
				SyncData:              true,
				UpdateAcrossPlatforms: true,
				CommonFeatureSet:      true,
				PlatformOptimizations: true,
			},
		},
	}
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "helixcode", "helix.json")
}

// LoadDefaultConfig loads the default configuration
func LoadDefaultConfig() (*HelixConfig, error) {
	manager, err := NewHelixConfigManager(GetDefaultConfigPath())
	if err != nil {
		return nil, err
	}

	return manager.GetConfig(), nil
}

// SaveDefaultConfig saves a configuration to the default path
func SaveDefaultConfig(config *HelixConfig) error {
	manager, err := NewHelixConfigManager(GetDefaultConfigPath())
	if err != nil {
		return err
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.config = config
	return manager.saveLocked()
}
