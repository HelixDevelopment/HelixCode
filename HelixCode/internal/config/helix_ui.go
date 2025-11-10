package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ConfigUI represents the configuration editing interface
type ConfigUI struct {
	manager          *HelixConfigManager
	currentPath      string
	history          []ConfigSnapshot
	validationErrors map[string]string
	themes           map[string]ThemeConfig
}

// ConfigSnapshot represents a configuration snapshot
type ConfigSnapshot struct {
	ID        uuid.UUID    `json:"id"`
	Name      string       `json:"name"`
	Timestamp time.Time    `json:"timestamp"`
	Config    *HelixConfig `json:"config"`
	CreatedBy string       `json:"created_by"`
	Notes     string       `json:"notes"`
}

// ThemeConfig represents a UI theme configuration
type ThemeConfig struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Colors      map[string]string      `json:"colors"`
	Fonts       map[string]string      `json:"fonts"`
	Icons       map[string]string      `json:"icons"`
	Animations  map[string]interface{} `json:"animations"`
}

// ConfigForm represents a configuration form structure
type ConfigForm struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Sections    []ConfigSection  `json:"sections"`
	Actions     []ConfigAction   `json:"actions"`
	Validation  ConfigValidation `json:"validation"`
	Layout      ConfigFormLayout `json:"layout"`
}

// ConfigSection represents a configuration section
type ConfigSection struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Icon        string             `json:"icon"`
	Fields      []ConfigField      `json:"fields"`
	Groups      []ConfigFieldGroup `json:"groups"`
	Visible     bool               `json:"visible"`
	Collapsed   bool               `json:"collapsed"`
	Priority    int                `json:"priority"`
}

// ConfigField represents a configuration field
type ConfigField struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"` // text, number, boolean, select, multiselect, file, directory, etc.
	Label        string            `json:"label"`
	Description  string            `json:"description"`
	Path         string            `json:"path"` // JSON path to the config value
	Default      interface{}       `json:"default"`
	Required     bool              `json:"required"`
	Validation   FieldValidation   `json:"validation"`
	UI           FieldUI           `json:"ui"`
	Dependencies []FieldDependency `json:"dependencies"`
}

// ConfigFieldGroup represents a group of related fields
type ConfigFieldGroup struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Fields      []string `json:"fields"` // Field IDs
	Layout      string   `json:"layout"` // horizontal, vertical, grid
	Visible     bool     `json:"visible"`
	Priority    int      `json:"priority"`
}

// FieldValidation represents field validation rules
type FieldValidation struct {
	Rules       []ValidationRuleConfig `json:"rules"`
	CustomRules []string               `json:"custom_rules"`
	Constraints map[string]interface{} `json:"constraints"`
}

// ValidationRule represents a validation rule
type ValidationRuleConfig struct {
	Type      string      `json:"type"` // required, min, max, pattern, custom
	Parameter interface{} `json:"parameter"`
	Message   string      `json:"message"`
	Severity  string      `json:"severity"` // error, warning, info
}

// FieldUI represents UI-specific field configuration
type FieldUI struct {
	Placeholder string                 `json:"placeholder"`
	HelpText    string                 `json:"help_text"`
	Icon        string                 `json:"icon"`
	Class       string                 `json:"class"`
	Style       map[string]string      `json:"style"`
	Attributes  map[string]interface{} `json:"attributes"`
	Options     []FieldOption          `json:"options"`
}

// FieldOption represents a field option (for select/multiselect)
type FieldOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Group       string `json:"group"`
	Disabled    bool   `json:"disabled"`
}

// FieldDependency represents field dependency conditions
type FieldDependency struct {
	Field     string      `json:"field"`     // Field ID
	Condition string      `json:"condition"` // equals, not_equals, contains, etc.
	Value     interface{} `json:"value"`
	Action    string      `json:"action"` // show, hide, enable, disable
}

// ConfigAction represents a configuration action button
type ConfigAction struct {
	ID           string             `json:"id"`
	Label        string             `json:"label"`
	Description  string             `json:"description"`
	Type         string             `json:"type"`   // primary, secondary, danger, etc.
	Action       string             `json:"action"` // save, reset, export, import, etc.
	Icon         string             `json:"icon"`
	Shortcut     string             `json:"shortcut"`
	Confirmation ActionConfirmation `json:"confirmation"`
	Visible      bool               `json:"visible"`
	Disabled     bool               `json:"disabled"`
}

// ActionConfirmation represents action confirmation requirements
type ActionConfirmation struct {
	Required    bool   `json:"required"`
	Message     string `json:"message"`
	Type        string `json:"type"` // modal, toast, inline
	Title       string `json:"title"`
	OKLabel     string `json:"ok_label"`
	CancelLabel string `json:"cancel_label"`
}

// ConfigValidation represents form validation
type ConfigValidation struct {
	Rules       []ValidationRuleConfig `json:"rules"`
	StrictMode  bool                   `json:"strict_mode"`
	RealTime    bool                   `json:"real_time"`
	CustomRules map[string]string      `json:"custom_rules"`
}

// ConfigFormLayout represents form layout configuration
type ConfigFormLayout struct {
	Type       string           `json:"type"` // tabs, accordion, wizard, single_page
	Columns    int              `json:"columns"`
	Gap        string           `json:"gap"`
	Responsive ResponsiveLayout `json:"responsive"`
	Sections   []string         `json:"sections"` // Section IDs in order
	CSS        string           `json:"css"`
}

// ResponsiveLayout represents responsive layout settings
type ResponsiveLayout struct {
	Breakpoints map[string]ResponsiveBreakpoint `json:"breakpoints"`
	HideOn      map[string][]string             `json:"hide_on"`
	ReorderOn   map[string][]string             `json:"reorder_on"`
}

// ResponsiveBreakpoint represents a responsive breakpoint
type ResponsiveBreakpoint struct {
	MaxWidth int    `json:"max_width"`
	Columns  int    `json:"columns"`
	Layout   string `json:"layout"`
}

// NewConfigUI creates a new configuration UI
func NewConfigUI(configPath string) (*ConfigUI, error) {
	manager, err := NewHelixConfigManager(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %v", err)
	}

	ui := &ConfigUI{
		manager:          manager,
		currentPath:      configPath,
		history:          make([]ConfigSnapshot, 0),
		validationErrors: make(map[string]string),
		themes:           make(map[string]ThemeConfig),
	}

	// Initialize themes
	ui.initializeThemes()

	return ui, nil
}

// initializeThemes initializes UI themes
func (ui *ConfigUI) initializeThemes() {
	ui.themes["dark"] = ThemeConfig{
		Name:        "Dark",
		Description: "Dark theme for low-light environments",
		Colors: map[string]string{
			"background": "#1e1e1e",
			"foreground": "#d4d4d4",
			"primary":    "#007acc",
			"secondary":  "#3c3c3c",
			"accent":     "#4fc3f7",
			"success":    "#4caf50",
			"warning":    "#ff9800",
			"error":      "#f44336",
			"border":     "#404040",
			"muted":      "#808080",
		},
		Fonts: map[string]string{
			"primary":   "SF Mono, Monaco, Consolas, monospace",
			"secondary": "SF Pro Text, Arial, sans-serif",
			"heading":   "SF Pro Display, Arial, sans-serif",
		},
		Icons: map[string]string{
			"folder":   "📁",
			"file":     "📄",
			"settings": "⚙️",
			"save":     "💾",
			"reset":    "🔄",
			"export":   "📤",
			"import":   "📥",
			"warning":  "⚠️",
			"error":    "❌",
			"success":  "✅",
		},
		Animations: map[string]interface{}{
			"duration": "0.2s",
			"easing":   "ease-in-out",
		},
	}

	ui.themes["light"] = ThemeConfig{
		Name:        "Light",
		Description: "Light theme for bright environments",
		Colors: map[string]string{
			"background": "#ffffff",
			"foreground": "#333333",
			"primary":    "#1976d2",
			"secondary":  "#f5f5f5",
			"accent":     "#03a9f4",
			"success":    "#4caf50",
			"warning":    "#ff9800",
			"error":      "#f44336",
			"border":     "#e0e0e0",
			"muted":      "#999999",
		},
		Fonts: map[string]string{
			"primary":   "SF Mono, Monaco, Consolas, monospace",
			"secondary": "SF Pro Text, Arial, sans-serif",
			"heading":   "SF Pro Display, Arial, sans-serif",
		},
		Icons: map[string]string{
			"folder":   "📁",
			"file":     "📄",
			"settings": "⚙️",
			"save":     "💾",
			"reset":    "🔄",
			"export":   "📤",
			"import":   "📥",
			"warning":  "⚠️",
			"error":    "❌",
			"success":  "✅",
		},
		Animations: map[string]interface{}{
			"duration": "0.2s",
			"easing":   "ease-in-out",
		},
	}
}

// GetConfigForm returns the complete configuration form
func (ui *ConfigUI) GetConfigForm() *ConfigForm {
	return &ConfigForm{
		ID:          "helix_config_form",
		Title:       "HelixCode Configuration",
		Description: "Configure all aspects of your HelixCode environment",
		Sections:    ui.getAllSections(),
		Actions:     ui.getAllActions(),
		Validation: ConfigValidation{
			Rules: []ValidationRuleConfig{
				{
					Type:      "custom",
					Parameter: "validate_all",
					Message:   "Configuration validation failed",
					Severity:  "error",
				},
			},
			StrictMode: true,
			RealTime:   true,
		},
		Layout: ConfigFormLayout{
			Type:    "tabs",
			Columns: 1,
			Gap:     "16px",
			Sections: []string{
				"application",
				"database",
				"redis",
				"auth",
				"server",
				"workers",
				"tasks",
				"llm",
				"tools",
				"workflows",
				"ui",
				"notifications",
				"security",
				"development",
				"platform",
			},
			Responsive: ResponsiveLayout{
				Breakpoints: map[string]ResponsiveBreakpoint{
					"mobile": {
						MaxWidth: 768,
						Columns:  1,
						Layout:   "accordion",
					},
					"tablet": {
						MaxWidth: 1024,
						Columns:  1,
						Layout:   "tabs",
					},
					"desktop": {
						MaxWidth: 9999,
						Columns:  2,
						Layout:   "tabs",
					},
				},
				HideOn: map[string][]string{
					"mobile": {"advanced_settings"},
				},
				ReorderOn: map[string][]string{
					"mobile": {"application", "llm", "tools", "ui"},
				},
			},
			CSS: `
				.config-form {
					max-width: 1200px;
					margin: 0 auto;
					padding: 20px;
				}
				.config-section {
					margin-bottom: 24px;
				}
				.config-field {
					margin-bottom: 16px;
				}
				.field-group {
					display: flex;
					gap: 16px;
					align-items: start;
				}
				.field-group.horizontal {
					flex-direction: row;
				}
				.field-group.vertical {
					flex-direction: column;
				}
				.field-group.grid {
					display: grid;
					grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
					gap: 16px;
				}
			`,
		},
	}
}

// getAllSections returns all configuration sections
func (ui *ConfigUI) getAllSections() []ConfigSection {
	return []ConfigSection{
		ui.getApplicationSection(),
		ui.getDatabaseSection(),
		ui.getRedisSection(),
		ui.getAuthSection(),
		ui.getServerSection(),
		ui.getWorkersSection(),
		ui.getTasksSection(),
		ui.getLLMSection(),
		ui.getToolsSection(),
		ui.getWorkflowsSection(),
		ui.getUISection(),
		ui.getNotificationsSection(),
		ui.getSecuritySection(),
		ui.getDevelopmentSection(),
		ui.getPlatformSection(),
	}
}

// getApplicationSection returns application configuration section
func (ui *ConfigUI) getApplicationSection() ConfigSection {
	return ConfigSection{
		ID:          "application",
		Title:       "Application",
		Description: "Core application settings",
		Icon:        "🚀",
		Visible:     true,
		Collapsed:   false,
		Priority:    1,
		Fields: []ConfigField{
			{
				ID:          "app_name",
				Type:        "text",
				Label:       "Application Name",
				Description: "Display name for the application",
				Path:        "application.name",
				Default:     "HelixCode",
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:     "required",
							Message:  "Application name is required",
							Severity: "error",
						},
						{
							Type:      "min",
							Parameter: 2,
							Message:   "Application name must be at least 2 characters",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 50,
							Message:   "Application name cannot exceed 50 characters",
							Severity:  "warning",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "Enter application name",
					Icon:        "📝",
				},
			},
			{
				ID:          "app_description",
				Type:        "textarea",
				Label:       "Description",
				Description: "Application description",
				Path:        "application.description",
				Default:     "Distributed AI Development Platform",
				Required:    false,
				UI: FieldUI{
					Placeholder: "Enter application description",
					Icon:        "📄",
				},
			},
			{
				ID:          "app_version",
				Type:        "text",
				Label:       "Version",
				Description: "Application version",
				Path:        "application.version",
				Default:     "1.0.0",
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:     "required",
							Message:  "Version is required",
							Severity: "error",
						},
						{
							Type:      "pattern",
							Parameter: `\d+\.\d+\.\d+`,
							Message:   "Version must follow semantic versioning (x.y.z)",
							Severity:  "error",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "1.0.0",
					Icon:        "🏷️",
				},
			},
			{
				ID:          "app_environment",
				Type:        "select",
				Label:       "Environment",
				Description: "Application environment",
				Path:        "application.environment",
				Default:     "development",
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "development",
							Label:       "Development",
							Description: "Development environment with debug features",
						},
						{
							Value:       "testing",
							Label:       "Testing",
							Description: "Testing environment for automated tests",
						},
						{
							Value:       "staging",
							Label:       "Staging",
							Description: "Pre-production environment",
						},
						{
							Value:       "production",
							Label:       "Production",
							Description: "Production environment",
						},
					},
				},
			},
		},
		Groups: []ConfigFieldGroup{
			{
				ID:          "workspace_group",
				Title:       "Workspace Settings",
				Description: "Configure workspace and file management",
				Fields:      []string{"workspace_path", "auto_save", "backup_enabled"},
				Layout:      "vertical",
				Visible:     true,
				Priority:    1,
			},
		},
	}
}

// getDatabaseSection returns database configuration section
func (ui *ConfigUI) getDatabaseSection() ConfigSection {
	return ConfigSection{
		ID:          "database",
		Title:       "Database",
		Description: "Database connection settings",
		Icon:        "🗄️",
		Visible:     true,
		Collapsed:   false,
		Priority:    2,
		Fields: []ConfigField{
			{
				ID:          "db_type",
				Type:        "select",
				Label:       "Database Type",
				Description: "Type of database to use",
				Path:        "database.type",
				Default:     "postgresql",
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "postgresql",
							Label:       "PostgreSQL",
							Description: "PostgreSQL database",
						},
						{
							Value:       "mysql",
							Label:       "MySQL",
							Description: "MySQL database",
						},
						{
							Value:       "sqlite",
							Label:       "SQLite",
							Description: "SQLite file database",
						},
					},
				},
			},
			{
				ID:          "db_host",
				Type:        "text",
				Label:       "Host",
				Description: "Database server hostname",
				Path:        "database.host",
				Default:     "localhost",
				Required:    true,
				UI: FieldUI{
					Placeholder: "localhost",
					Icon:        "🌐",
				},
				Dependencies: []FieldDependency{
					{
						Field:     "db_type",
						Condition: "equals",
						Value:     "sqlite",
						Action:    "hide",
					},
				},
			},
			{
				ID:          "db_port",
				Type:        "number",
				Label:       "Port",
				Description: "Database server port",
				Path:        "database.port",
				Default:     5432,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 1,
							Message:   "Port must be greater than 0",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 65535,
							Message:   "Port must be less than 65536",
							Severity:  "error",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "5432",
					Icon:        "🔌",
				},
				Dependencies: []FieldDependency{
					{
						Field:     "db_type",
						Condition: "equals",
						Value:     "sqlite",
						Action:    "hide",
					},
				},
			},
			{
				ID:          "db_database",
				Type:        "text",
				Label:       "Database Name",
				Description: "Name of the database",
				Path:        "database.database",
				Default:     "helixcode",
				Required:    true,
				UI: FieldUI{
					Placeholder: "helixcode",
					Icon:        "📚",
				},
			},
			{
				ID:          "db_username",
				Type:        "text",
				Label:       "Username",
				Description: "Database username",
				Path:        "database.username",
				Default:     "helixcode",
				Required:    true,
				UI: FieldUI{
					Placeholder: "helixcode",
					Icon:        "👤",
				},
				Dependencies: []FieldDependency{
					{
						Field:     "db_type",
						Condition: "equals",
						Value:     "sqlite",
						Action:    "hide",
					},
				},
			},
			{
				ID:          "db_password",
				Type:        "password",
				Label:       "Password",
				Description: "Database password",
				Path:        "database.password",
				Default:     "",
				Required:    false,
				UI: FieldUI{
					Placeholder: "Enter password",
					Icon:        "🔒",
				},
				Dependencies: []FieldDependency{
					{
						Field:     "db_type",
						Condition: "equals",
						Value:     "sqlite",
						Action:    "hide",
					},
				},
			},
			{
				ID:          "db_ssl_mode",
				Type:        "select",
				Label:       "SSL Mode",
				Description: "SSL connection mode",
				Path:        "database.ssl_mode",
				Default:     "disable",
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "disable",
							Label:       "Disabled",
							Description: "SSL disabled",
						},
						{
							Value:       "require",
							Label:       "Require",
							Description: "SSL required",
						},
						{
							Value:       "verify-ca",
							Label:       "Verify CA",
							Description: "SSL with CA verification",
						},
						{
							Value:       "verify-full",
							Label:       "Verify Full",
							Description: "SSL with full verification",
						},
					},
				},
				Dependencies: []FieldDependency{
					{
						Field:     "db_type",
						Condition: "equals",
						Value:     "sqlite",
						Action:    "hide",
					},
				},
			},
		},
		Groups: []ConfigFieldGroup{
			{
				ID:          "connection_pool",
				Title:       "Connection Pool",
				Description: "Database connection pool settings",
				Fields:      []string{"max_connections", "max_idle_connections", "connection_lifetime"},
				Layout:      "horizontal",
				Visible:     true,
				Priority:    2,
			},
		},
	}
}

// getRedisSection returns Redis configuration section
func (ui *ConfigUI) getRedisSection() ConfigSection {
	return ConfigSection{
		ID:          "redis",
		Title:       "Redis",
		Description: "Redis cache and session storage",
		Icon:        "🔴",
		Visible:     true,
		Collapsed:   false,
		Priority:    3,
		Fields: []ConfigField{
			{
				ID:          "redis_enabled",
				Type:        "boolean",
				Label:       "Enable Redis",
				Description: "Enable Redis for caching and sessions",
				Path:        "redis.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "⚡",
				},
			},
			{
				ID:          "redis_host",
				Type:        "text",
				Label:       "Host",
				Description: "Redis server hostname",
				Path:        "redis.host",
				Default:     "localhost",
				Required:    true,
				UI: FieldUI{
					Placeholder: "localhost",
					Icon:        "🌐",
				},
				Dependencies: []FieldDependency{
					{
						Field:     "redis_enabled",
						Condition: "equals",
						Value:     false,
						Action:    "disable",
					},
				},
			},
			{
				ID:          "redis_port",
				Type:        "number",
				Label:       "Port",
				Description: "Redis server port",
				Path:        "redis.port",
				Default:     6379,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 1,
							Message:   "Port must be greater than 0",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 65535,
							Message:   "Port must be less than 65536",
							Severity:  "error",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "6379",
					Icon:        "🔌",
				},
				Dependencies: []FieldDependency{
					{
						Field:     "redis_enabled",
						Condition: "equals",
						Value:     false,
						Action:    "disable",
					},
				},
			},
			{
				ID:          "redis_password",
				Type:        "password",
				Label:       "Password",
				Description: "Redis password (optional)",
				Path:        "redis.password",
				Default:     "",
				Required:    false,
				UI: FieldUI{
					Placeholder: "Enter Redis password",
					Icon:        "🔒",
				},
				Dependencies: []FieldDependency{
					{
						Field:     "redis_enabled",
						Condition: "equals",
						Value:     false,
						Action:    "disable",
					},
				},
			},
			{
				ID:          "redis_database",
				Type:        "number",
				Label:       "Database",
				Description: "Redis database number",
				Path:        "redis.database",
				Default:     0,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 0,
							Message:   "Database number must be 0 or greater",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 15,
							Message:   "Database number must be less than 16",
							Severity:  "error",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "0",
					Icon:        "📊",
				},
				Dependencies: []FieldDependency{
					{
						Field:     "redis_enabled",
						Condition: "equals",
						Value:     false,
						Action:    "disable",
					},
				},
			},
		},
	}
}

// getAuthSection returns authentication configuration section
func (ui *ConfigUI) getAuthSection() ConfigSection {
	return ConfigSection{
		ID:          "auth",
		Title:       "Authentication",
		Description: "Authentication and authorization settings",
		Icon:        "🔐",
		Visible:     true,
		Collapsed:   false,
		Priority:    4,
		Fields: []ConfigField{
			{
				ID:          "auth_jwt_secret",
				Type:        "password",
				Label:       "JWT Secret",
				Description: "Secret key for JWT tokens",
				Path:        "auth.jwt_secret",
				Default:     "",
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:     "required",
							Message:  "JWT secret is required",
							Severity: "error",
						},
						{
							Type:      "min",
							Parameter: 32,
							Message:   "JWT secret must be at least 32 characters",
							Severity:  "error",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "Enter JWT secret",
					Icon:        "🔑",
				},
			},
			{
				ID:          "auth_token_expiry",
				Type:        "number",
				Label:       "Token Expiry",
				Description: "JWT token expiry time in seconds",
				Path:        "auth.token_expiry",
				Default:     86400,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 300,
							Message:   "Token expiry must be at least 300 seconds (5 minutes)",
							Severity:  "warning",
						},
						{
							Type:      "max",
							Parameter: 604800,
							Message:   "Token expiry should not exceed 604800 seconds (7 days)",
							Severity:  "warning",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "86400",
					Icon:        "⏰",
				},
			},
			{
				ID:          "auth_session_timeout",
				Type:        "number",
				Label:       "Session Timeout",
				Description: "Session timeout in minutes",
				Path:        "auth.session_timeout",
				Default:     30,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 5,
							Message:   "Session timeout must be at least 5 minutes",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 1440,
							Message:   "Session timeout should not exceed 1440 minutes (24 hours)",
							Severity:  "warning",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "30",
					Icon:        "⏱️",
				},
			},
		},
	}
}

// getServerSection returns server configuration section
func (ui *ConfigUI) getServerSection() ConfigSection {
	return ConfigSection{
		ID:          "server",
		Title:       "Server",
		Description: "HTTP server configuration",
		Icon:        "🌐",
		Visible:     true,
		Collapsed:   false,
		Priority:    5,
		Fields: []ConfigField{
			{
				ID:          "server_address",
				Type:        "text",
				Label:       "Address",
				Description: "Server bind address",
				Path:        "server.address",
				Default:     "0.0.0.0",
				Required:    true,
				UI: FieldUI{
					Placeholder: "0.0.0.0",
					Icon:        "📍",
				},
			},
			{
				ID:          "server_port",
				Type:        "number",
				Label:       "Port",
				Description: "HTTP server port",
				Path:        "server.port",
				Default:     8080,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 1,
							Message:   "Port must be greater than 0",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 65535,
							Message:   "Port must be less than 65536",
							Severity:  "error",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "8080",
					Icon:        "🔌",
				},
			},
			{
				ID:          "server_read_timeout",
				Type:        "number",
				Label:       "Read Timeout",
				Description: "Request read timeout in seconds",
				Path:        "server.read_timeout",
				Default:     30,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 1,
							Message:   "Read timeout must be at least 1 second",
							Severity:  "warning",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "30",
					Icon:        "📖",
				},
			},
			{
				ID:          "server_write_timeout",
				Type:        "number",
				Label:       "Write Timeout",
				Description: "Response write timeout in seconds",
				Path:        "server.write_timeout",
				Default:     30,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 1,
							Message:   "Write timeout must be at least 1 second",
							Severity:  "warning",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "30",
					Icon:        "✍️",
				},
			},
		},
	}
}

// getWorkersSection returns workers configuration section
func (ui *ConfigUI) getWorkersSection() ConfigSection {
	return ConfigSection{
		ID:          "workers",
		Title:       "Workers",
		Description: "Distributed worker configuration",
		Icon:        "👥",
		Visible:     true,
		Collapsed:   false,
		Priority:    6,
		Fields: []ConfigField{
			{
				ID:          "workers_health_check_interval",
				Type:        "number",
				Label:       "Health Check Interval",
				Description: "Worker health check interval in seconds",
				Path:        "workers.health_check_interval",
				Default:     30,
				Required:    true,
				UI: FieldUI{
					Placeholder: "30",
					Icon:        "💓",
				},
			},
			{
				ID:          "workers_max_concurrent_tasks",
				Type:        "number",
				Label:       "Max Concurrent Tasks",
				Description: "Maximum concurrent tasks per worker",
				Path:        "workers.max_concurrent_tasks",
				Default:     10,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 1,
							Message:   "Max concurrent tasks must be at least 1",
							Severity:  "error",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "10",
					Icon:        "⚙️",
				},
			},
			{
				ID:          "workers_auto_scaling",
				Type:        "boolean",
				Label:       "Auto Scaling",
				Description: "Enable automatic worker scaling",
				Path:        "workers.auto_scaling",
				Default:     false,
				Required:    true,
				UI: FieldUI{
					Icon: "📈",
				},
			},
		},
	}
}

// getTasksSection returns tasks configuration section
func (ui *ConfigUI) getTasksSection() ConfigSection {
	return ConfigSection{
		ID:          "tasks",
		Title:       "Tasks",
		Description: "Task queue and execution settings",
		Icon:        "📋",
		Visible:     true,
		Collapsed:   false,
		Priority:    7,
		Fields: []ConfigField{
			{
				ID:          "tasks_queue_size",
				Type:        "number",
				Label:       "Queue Size",
				Description: "Maximum task queue size",
				Path:        "tasks.queue_size",
				Default:     1000,
				Required:    true,
				UI: FieldUI{
					Placeholder: "1000",
					Icon:        "📊",
				},
			},
			{
				ID:          "tasks_max_retries",
				Type:        "number",
				Label:       "Max Retries",
				Description: "Maximum number of task retries",
				Path:        "tasks.max_retries",
				Default:     3,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 0,
							Message:   "Max retries cannot be negative",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 10,
							Message:   "Max retries should not exceed 10",
							Severity:  "warning",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "3",
					Icon:        "🔄",
				},
			},
			{
				ID:          "tasks_checkpoint_interval",
				Type:        "number",
				Label:       "Checkpoint Interval",
				Description: "Task checkpoint interval in seconds",
				Path:        "tasks.checkpoint_interval",
				Default:     300,
				Required:    true,
				UI: FieldUI{
					Placeholder: "300",
					Icon:        "💾",
				},
			},
		},
	}
}

// getLLMSection returns LLM configuration section
func (ui *ConfigUI) getLLMSection() ConfigSection {
	return ConfigSection{
		ID:          "llm",
		Title:       "LLM",
		Description: "Large Language Model configuration",
		Icon:        "🤖",
		Visible:     true,
		Collapsed:   false,
		Priority:    8,
		Fields: []ConfigField{
			{
				ID:          "llm_default_provider",
				Type:        "select",
				Label:       "Default Provider",
				Description: "Default LLM provider",
				Path:        "llm.default_provider",
				Default:     "local",
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "local",
							Label:       "Local",
							Description: "Local LLM (Ollama, Llama.cpp, etc.)",
						},
						{
							Value:       "openai",
							Label:       "OpenAI",
							Description: "OpenAI GPT models",
						},
						{
							Value:       "anthropic",
							Label:       "Anthropic",
							Description: "Anthropic Claude models",
						},
						{
							Value:       "gemini",
							Label:       "Google Gemini",
							Description: "Google Gemini models",
						},
						{
							Value:       "azure",
							Label:       "Azure OpenAI",
							Description: "Microsoft Azure OpenAI",
						},
						{
							Value:       "vertexai",
							Label:       "Vertex AI",
							Description: "Google Cloud Vertex AI",
						},
						{
							Value:       "bedrock",
							Label:       "AWS Bedrock",
							Description: "Amazon Bedrock",
						},
						{
							Value:       "qwen",
							Label:       "Qwen",
							Description: "Alibaba Qwen models",
						},
						{
							Value:       "xai",
							Label:       "xAI",
							Description: "xAI Grok models",
						},
						{
							Value:       "groq",
							Label:       "Groq",
							Description: "Groq fast inference",
						},
						{
							Value:       "openrouter",
							Label:       "OpenRouter",
							Description: "OpenRouter model marketplace",
						},
						{
							Value:       "copilot",
							Label:       "GitHub Copilot",
							Description: "GitHub Copilot models",
						},
					},
				},
			},
			{
				ID:          "llm_default_model",
				Type:        "text",
				Label:       "Default Model",
				Description: "Default LLM model name",
				Path:        "llm.default_model",
				Default:     "llama-3.2-3b",
				Required:    true,
				UI: FieldUI{
					Placeholder: "llama-3.2-3b",
					Icon:        "🧠",
				},
			},
			{
				ID:          "llm_max_tokens",
				Type:        "number",
				Label:       "Max Tokens",
				Description: "Maximum tokens per request",
				Path:        "llm.max_tokens",
				Default:     4096,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 1,
							Message:   "Max tokens must be greater than 0",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 32768,
							Message:   "Max tokens should not exceed 32768",
							Severity:  "warning",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "4096",
					Icon:        "📏",
				},
			},
			{
				ID:          "llm_temperature",
				Type:        "slider",
				Label:       "Temperature",
				Description: "Sampling temperature (0.0 to 2.0)",
				Path:        "llm.temperature",
				Default:     0.7,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 0.0,
							Message:   "Temperature must be between 0.0 and 2.0",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 2.0,
							Message:   "Temperature must be between 0.0 and 2.0",
							Severity:  "error",
						},
					},
				},
				UI: FieldUI{
					Icon: "🌡️",
				},
			},
		},
	}
}

// getToolsSection returns tools configuration section
func (ui *ConfigUI) getToolsSection() ConfigSection {
	return ConfigSection{
		ID:          "tools",
		Title:       "Tools",
		Description: "AI tools and capabilities configuration",
		Icon:        "🛠️",
		Visible:     true,
		Collapsed:   false,
		Priority:    9,
		Fields: []ConfigField{
			{
				ID:          "tools_file_system_enabled",
				Type:        "boolean",
				Label:       "File System Tools",
				Description: "Enable file system operations",
				Path:        "tools.file_system.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "📁",
				},
			},
			{
				ID:          "tools_shell_enabled",
				Type:        "boolean",
				Label:       "Shell Tools",
				Description: "Enable shell command execution",
				Path:        "tools.shell.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "💻",
				},
			},
			{
				ID:          "tools_browser_enabled",
				Type:        "boolean",
				Label:       "Browser Tools",
				Description: "Enable web browser automation",
				Path:        "tools.browser.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "🌐",
				},
			},
			{
				ID:          "tools_voice_enabled",
				Type:        "boolean",
				Label:       "Voice Tools",
				Description: "Enable voice input and transcription",
				Path:        "tools.voice.enabled",
				Default:     false,
				Required:    true,
				UI: FieldUI{
					Icon: "🎤",
				},
			},
		},
	}
}

// getWorkflowsSection returns workflows configuration section
func (ui *ConfigUI) getWorkflowsSection() ConfigSection {
	return ConfigSection{
		ID:          "workflows",
		Title:       "Workflows",
		Description: "Workflow and automation settings",
		Icon:        "⚡",
		Visible:     true,
		Collapsed:   false,
		Priority:    10,
		Fields: []ConfigField{
			{
				ID:          "workflows_enabled",
				Type:        "boolean",
				Label:       "Enable Workflows",
				Description: "Enable workflow automation",
				Path:        "workflows.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "🔧",
				},
			},
			{
				ID:          "workflows_default_mode",
				Type:        "select",
				Label:       "Default Mode",
				Description: "Default workflow execution mode",
				Path:        "workflows.default_mode",
				Default:     "plan",
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "plan",
							Label:       "Plan Mode",
							Description: "Plan and review before execution",
						},
						{
							Value:       "act",
							Label:       "Act Mode",
							Description: "Execute without planning",
						},
						{
							Value:       "auto",
							Label:       "Auto Mode",
							Description: "Automatic execution with minimal supervision",
						},
					},
				},
			},
			{
				ID:          "workflows_plan_mode_enabled",
				Type:        "boolean",
				Label:       "Plan Mode",
				Description: "Enable two-phase planning mode",
				Path:        "workflows.plan_mode.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "📋",
				},
			},
		},
	}
}

// getUISection returns UI configuration section
func (ui *ConfigUI) getUISection() ConfigSection {
	return ConfigSection{
		ID:          "ui",
		Title:       "User Interface",
		Description: "UI appearance and behavior settings",
		Icon:        "🎨",
		Visible:     true,
		Collapsed:   false,
		Priority:    11,
		Fields: []ConfigField{
			{
				ID:          "ui_theme",
				Type:        "select",
				Label:       "Theme",
				Description: "UI theme",
				Path:        "ui.theme",
				Default:     "dark",
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "dark",
							Label:       "Dark",
							Description: "Dark theme for low-light environments",
						},
						{
							Value:       "light",
							Label:       "Light",
							Description: "Light theme for bright environments",
						},
						{
							Value:       "auto",
							Label:       "Auto",
							Description: "Automatically switch based on system preference",
						},
					},
				},
			},
			{
				ID:          "ui_language",
				Type:        "select",
				Label:       "Language",
				Description: "Interface language",
				Path:        "ui.language",
				Default:     "en",
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "en",
							Label:       "English",
							Description: "English language",
						},
						{
							Value:       "zh",
							Label:       "中文",
							Description: "Chinese language",
						},
						{
							Value:       "es",
							Label:       "Español",
							Description: "Spanish language",
						},
						{
							Value:       "fr",
							Label:       "Français",
							Description: "French language",
						},
						{
							Value:       "de",
							Label:       "Deutsch",
							Description: "German language",
						},
						{
							Value:       "ja",
							Label:       "日本語",
							Description: "Japanese language",
						},
					},
				},
			},
			{
				ID:          "ui_font_size",
				Type:        "number",
				Label:       "Font Size",
				Description: "UI font size in pixels",
				Path:        "ui.font_size",
				Default:     14,
				Required:    true,
				Validation: FieldValidation{
					Rules: []ValidationRuleConfig{
						{
							Type:      "min",
							Parameter: 8,
							Message:   "Font size must be at least 8 pixels",
							Severity:  "error",
						},
						{
							Type:      "max",
							Parameter: 32,
							Message:   "Font size should not exceed 32 pixels",
							Severity:  "warning",
						},
					},
				},
				UI: FieldUI{
					Placeholder: "14",
					Icon:        "📝",
				},
			},
		},
	}
}

// getNotificationsSection returns notifications configuration section
func (ui *ConfigUI) getNotificationsSection() ConfigSection {
	return ConfigSection{
		ID:          "notifications",
		Title:       "Notifications",
		Description: "Notification settings and channels",
		Icon:        "🔔",
		Visible:     true,
		Collapsed:   false,
		Priority:    12,
		Fields: []ConfigField{
			{
				ID:          "notifications_enabled",
				Type:        "boolean",
				Label:       "Enable Notifications",
				Description: "Enable system notifications",
				Path:        "notifications.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "🔕",
				},
			},
			{
				ID:          "notifications_desktop_enabled",
				Type:        "boolean",
				Label:       "Desktop Notifications",
				Description: "Enable desktop notification popups",
				Path:        "notifications.channels.desktop.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "💻",
				},
			},
		},
	}
}

// getSecuritySection returns security configuration section
func (ui *ConfigUI) getSecuritySection() ConfigSection {
	return ConfigSection{
		ID:          "security",
		Title:       "Security",
		Description: "Security and privacy settings",
		Icon:        "🔒",
		Visible:     true,
		Collapsed:   false,
		Priority:    13,
		Fields: []ConfigField{
			{
				ID:          "security_encryption_enabled",
				Type:        "boolean",
				Label:       "Encryption",
				Description: "Enable data encryption",
				Path:        "security.encryption_enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "🔐",
				},
			},
			{
				ID:          "security_auth_methods",
				Type:        "multiselect",
				Label:       "Authentication Methods",
				Description: "Allowed authentication methods",
				Path:        "security.authentication.methods",
				Default:     []string{"password", "2fa"},
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "password",
							Label:       "Password",
							Description: "Username and password authentication",
						},
						{
							Value:       "2fa",
							Label:       "Two-Factor Auth",
							Description: "Two-factor authentication",
						},
						{
							Value:       "oauth",
							Label:       "OAuth",
							Description: "OAuth authentication",
						},
						{
							Value:       "certificate",
							Label:       "Certificate",
							Description: "Client certificate authentication",
						},
					},
				},
			},
		},
	}
}

// getDevelopmentSection returns development configuration section
func (ui *ConfigUI) getDevelopmentSection() ConfigSection {
	return ConfigSection{
		ID:          "development",
		Title:       "Development",
		Description: "Development and debugging settings",
		Icon:        "🔧",
		Visible:     true,
		Collapsed:   false,
		Priority:    14,
		Fields: []ConfigField{
			{
				ID:          "development_enabled",
				Type:        "boolean",
				Label:       "Enable Development",
				Description: "Enable development features",
				Path:        "development.enabled",
				Default:     false,
				Required:    true,
				UI: FieldUI{
					Icon: "🛠️",
				},
			},
			{
				ID:          "development_debug_enabled",
				Type:        "boolean",
				Label:       "Debug Mode",
				Description: "Enable debug logging and features",
				Path:        "development.debug.enabled",
				Default:     false,
				Required:    true,
				UI: FieldUI{
					Icon: "🐛",
				},
			},
			{
				ID:          "development_testing_enabled",
				Type:        "boolean",
				Label:       "Testing Features",
				Description: "Enable testing features and tools",
				Path:        "development.testing.enabled",
				Default:     true,
				Required:    true,
				UI: FieldUI{
					Icon: "🧪",
				},
			},
		},
	}
}

// getPlatformSection returns platform configuration section
func (ui *ConfigUI) getPlatformSection() ConfigSection {
	return ConfigSection{
		ID:          "platform",
		Title:       "Platform",
		Description: "Platform-specific settings",
		Icon:        "📱",
		Visible:     true,
		Collapsed:   false,
		Priority:    15,
		Fields: []ConfigField{
			{
				ID:          "platform_current_platform",
				Type:        "select",
				Label:       "Current Platform",
				Description: "Currently active platform",
				Path:        "platform.current_platform",
				Default:     "desktop",
				Required:    true,
				UI: FieldUI{
					Options: []FieldOption{
						{
							Value:       "desktop",
							Label:       "Desktop",
							Description: "Desktop application",
						},
						{
							Value:       "web",
							Label:       "Web",
							Description: "Web application",
						},
						{
							Value:       "mobile",
							Label:       "Mobile",
							Description: "Mobile application",
						},
						{
							Value:       "tui",
							Label:       "Terminal UI",
							Description: "Terminal user interface",
						},
					},
				},
			},
		},
	}
}

// getAllActions returns all configuration form actions
func (ui *ConfigUI) getAllActions() []ConfigAction {
	return []ConfigAction{
		{
			ID:          "save",
			Label:       "Save",
			Description: "Save configuration changes",
			Type:        "primary",
			Action:      "save",
			Icon:        "💾",
			Shortcut:    "Ctrl+S",
			Confirmation: ActionConfirmation{
				Required: false,
				Message:  "",
			},
			Visible:  true,
			Disabled: false,
		},
		{
			ID:          "reset",
			Label:       "Reset",
			Description: "Reset to default values",
			Type:        "secondary",
			Action:      "reset",
			Icon:        "🔄",
			Shortcut:    "Ctrl+R",
			Confirmation: ActionConfirmation{
				Required:    true,
				Message:     "Are you sure you want to reset all settings to their defaults? This action cannot be undone.",
				Type:        "modal",
				Title:       "Reset Configuration",
				OKLabel:     "Reset",
				CancelLabel: "Cancel",
			},
			Visible:  true,
			Disabled: false,
		},
		{
			ID:          "export",
			Label:       "Export",
			Description: "Export configuration to file",
			Type:        "secondary",
			Action:      "export",
			Icon:        "📤",
			Shortcut:    "Ctrl+E",
			Confirmation: ActionConfirmation{
				Required: false,
				Message:  "",
			},
			Visible:  true,
			Disabled: false,
		},
		{
			ID:          "import",
			Label:       "Import",
			Description: "Import configuration from file",
			Type:        "secondary",
			Action:      "import",
			Icon:        "📥",
			Shortcut:    "Ctrl+I",
			Confirmation: ActionConfirmation{
				Required: false,
				Message:  "",
			},
			Visible:  true,
			Disabled: false,
		},
		{
			ID:          "backup",
			Label:       "Backup",
			Description: "Create configuration backup",
			Type:        "secondary",
			Action:      "backup",
			Icon:        "📦",
			Shortcut:    "Ctrl+B",
			Confirmation: ActionConfirmation{
				Required: false,
				Message:  "",
			},
			Visible:  true,
			Disabled: false,
		},
		{
			ID:          "validate",
			Label:       "Validate",
			Description: "Validate current configuration",
			Type:        "secondary",
			Action:      "validate",
			Icon:        "✅",
			Shortcut:    "Ctrl+V",
			Confirmation: ActionConfirmation{
				Required: false,
				Message:  "",
			},
			Visible:  true,
			Disabled: false,
		},
	}
}

// SaveConfig saves the current configuration
func (ui *ConfigUI) SaveConfig() error {
	return ui.manager.Save()
}

// ResetConfig resets configuration to defaults
func (ui *ConfigUI) ResetConfig() error {
	return ui.manager.ResetToDefaults()
}

// ExportConfig exports configuration to a file
func (ui *ConfigUI) ExportConfig(filePath string) error {
	return ui.manager.ExportConfig(filePath)
}

// ImportConfig imports configuration from a file
func (ui *ConfigUI) ImportConfig(filePath string) error {
	return ui.manager.ImportConfig(filePath)
}

// BackupConfig creates a backup of the current configuration
func (ui *ConfigUI) BackupConfig(backupPath string) error {
	return ui.manager.BackupConfig(backupPath)
}

// ValidateConfig validates the current configuration
func (ui *ConfigUI) ValidateConfig() map[string]string {
	config := ui.manager.GetConfig()
	return ui.validateConfig(config)
}

// validateConfig validates configuration and returns errors
func (ui *ConfigUI) validateConfig(config *HelixConfig) map[string]string {
	errors := make(map[string]string)

	// Basic validation
	if config.Application.Name == "" {
		errors["application.name"] = "Application name is required"
	}

	if config.Server.Port < 1 || config.Server.Port > 65535 {
		errors["server.port"] = "Server port must be between 1 and 65535"
	}

	if config.Database.Host == "" {
		errors["database.host"] = "Database host is required"
	}

	if config.LLM.DefaultProvider == "" {
		errors["llm.default_provider"] = "Default LLM provider is required"
	}

	if config.LLM.MaxTokens < 1 {
		errors["llm.max_tokens"] = "Max tokens must be positive"
	}

	if config.LLM.Temperature < 0 || config.LLM.Temperature > 2 {
		errors["llm.temperature"] = "Temperature must be between 0 and 2"
	}

	// Validate using manager
	if err := ui.manager.ValidateConfig(config); err != nil {
		errors["general"] = err.Error()
	}

	ui.validationErrors = errors
	return errors
}

// CreateSnapshot creates a configuration snapshot
func (ui *ConfigUI) CreateSnapshot(name, notes string) (*ConfigSnapshot, error) {
	config := ui.manager.GetConfig()

	snapshot := ConfigSnapshot{
		ID:        uuid.New(),
		Name:      name,
		Timestamp: time.Now(),
		Config:    config,
		CreatedBy: "user",
		Notes:     notes,
	}

	ui.history = append(ui.history, snapshot)

	return &snapshot, nil
}

// GetSnapshots returns all configuration snapshots
func (ui *ConfigUI) GetSnapshots() []ConfigSnapshot {
	return ui.history
}

// RestoreSnapshot restores a configuration snapshot
func (ui *ConfigUI) RestoreSnapshot(snapshotID uuid.UUID) error {
	for _, snapshot := range ui.history {
		if snapshot.ID == snapshotID {
			return ui.manager.UpdateConfig(func(config *HelixConfig) {
				*config = *snapshot.Config
			})
		}
	}

	return fmt.Errorf("snapshot not found")
}

// GetTheme returns a theme configuration
func (ui *ConfigUI) GetTheme(themeName string) (ThemeConfig, bool) {
	theme, exists := ui.themes[themeName]
	return theme, exists
}

// GetThemes returns all available themes
func (ui *ConfigUI) GetThemes() map[string]ThemeConfig {
	return ui.themes
}

// SetTheme applies a theme to the configuration
func (ui *ConfigUI) SetTheme(themeName string) error {
	if _, exists := ui.themes[themeName]; !exists {
		return fmt.Errorf("theme not found: %s", themeName)
	}

	return ui.manager.UpdateConfig(func(config *HelixConfig) {
		config.UI.Theme = themeName
	})
}

// GetConfigJSON returns the current configuration as JSON
func (ui *ConfigUI) GetConfigJSON() ([]byte, error) {
	config := ui.manager.GetConfig()
	return json.MarshalIndent(config, "", "  ")
}

// SetConfigJSON sets configuration from JSON
func (ui *ConfigUI) SetConfigJSON(data []byte) error {
	var config HelixConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse configuration JSON: %v", err)
	}

	return ui.manager.UpdateConfig(func(current *HelixConfig) {
		*current = config
	})
}

// GetConfigManager returns the underlying configuration manager
func (ui *ConfigUI) GetConfigManager() *HelixConfigManager {
	return ui.manager
}
