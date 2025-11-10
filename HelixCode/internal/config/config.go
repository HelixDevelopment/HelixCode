package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"dev.helix.code/internal/database"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  database.Config `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Workers   WorkersConfig   `mapstructure:"workers"`
	Tasks     TasksConfig     `mapstructure:"tasks"`
	LLM       LLMConfig       `mapstructure:"llm"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Logging   LoggingConfig   `mapstructure:"logging"`
}

// ProvidersConfig represents provider configurations
type ProvidersConfig struct {
	Mem0    Mem0Config    `mapstructure:"mem0"`
	Zep     ZepConfig     `mapstructure:"zep"`
	Memonto MemontoConfig `mapstructure:"memonto"`
	BaseAI  BaseAIConfig  `mapstructure:"baseai"`
}

// Mem0Config represents Mem0 provider configuration
type Mem0Config struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// ZepConfig represents Zep provider configuration
type ZepConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// MemontoConfig represents Memonto provider configuration
type MemontoConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// BaseAIConfig represents BaseAI provider configuration
type BaseAIConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set default values
	setDefaults()

	// Find config file
	configPath := findConfigFile()
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Use default config locations
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./config/")
		viper.AddConfigPath("./")
		viper.AddConfigPath("$HOME/.config/helixcode/")
		viper.AddConfigPath("/etc/helixcode/")
	}

	// Read in environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("HELIX")

	// Explicitly bind environment variables for critical settings
	viper.BindEnv("auth.jwt_secret", "HELIX_AUTH_JWT_SECRET")
	viper.BindEnv("database.password", "HELIX_DATABASE_PASSWORD")
	viper.BindEnv("database.host", "HELIX_DATABASE_HOST")
	viper.BindEnv("database.port", "HELIX_DATABASE_PORT")
	viper.BindEnv("database.user", "HELIX_DATABASE_USER")
	viper.BindEnv("database.dbname", "HELIX_DATABASE_NAME")
	viper.BindEnv("redis.password", "HELIX_REDIS_PASSWORD")
	viper.BindEnv("redis.host", "HELIX_REDIS_HOST")
	viper.BindEnv("redis.port", "HELIX_REDIS_PORT")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
		// Config file not found, but we can continue with defaults
		fmt.Println("⚠️  No config file found, using defaults and environment variables")
	} else {
		fmt.Printf("📁 Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Unmarshal config
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Validate config
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.address", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 60)
	viper.SetDefault("server.shutdown_timeout", 30)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "helixcode")
	viper.SetDefault("database.dbname", "helixcode")
	viper.SetDefault("database.sslmode", "disable")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.enabled", true)

	// Auth defaults
	viper.SetDefault("auth.jwt_secret", "default-secret-change-in-production")
	viper.SetDefault("auth.token_expiry", 86400)    // 24 hours
	viper.SetDefault("auth.session_expiry", 604800) // 7 days
	viper.SetDefault("auth.bcrypt_cost", 12)

	// Workers defaults
	viper.SetDefault("workers.health_check_interval", 30)
	viper.SetDefault("workers.health_ttl", 120)
	viper.SetDefault("workers.max_concurrent_tasks", 10)

	// Tasks defaults
	viper.SetDefault("tasks.max_retries", 3)
	viper.SetDefault("tasks.checkpoint_interval", 300)
	viper.SetDefault("tasks.cleanup_interval", 3600)

	// LLM defaults
	viper.SetDefault("llm.default_provider", "local")
	viper.SetDefault("llm.max_tokens", 4096)
	viper.SetDefault("llm.temperature", 0.7)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.output", "stdout")
}

// findConfigFile searches for config file in various locations
func findConfigFile() string {
	// Check environment variable first
	if configPath := os.Getenv("HELIX_CONFIG"); configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check common locations
	locations := []string{
		"./config/config.yaml",
		"./config.yaml",
		"$HOME/.config/helixcode/config.yaml",
		"/etc/helixcode/config.yaml",
	}

	for _, location := range locations {
		if expanded := os.ExpandEnv(location); expanded != location {
			if _, err := os.Stat(expanded); err == nil {
				return expanded
			}
		}
	}

	return ""
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Server validation
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	// Database validation
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	// Redis validation
	if cfg.Redis.Enabled {
		if cfg.Redis.Host == "" {
			return fmt.Errorf("redis host is required when redis is enabled")
		}
		if cfg.Redis.Port < 1 || cfg.Redis.Port > 65535 {
			return fmt.Errorf("redis port must be between 1 and 65535")
		}
	}

	// Auth validation
	if cfg.Auth.JWTSecret == "" || cfg.Auth.JWTSecret == "default-secret-change-in-production" {
		return fmt.Errorf("JWT secret must be set and not use default value")
	}

	// Workers validation
	if cfg.Workers.HealthCheckInterval < 1 {
		return fmt.Errorf("health check interval must be positive")
	}
	if cfg.Workers.MaxConcurrentTasks < 1 {
		return fmt.Errorf("max concurrent tasks must be positive")
	}

	// Tasks validation
	if cfg.Tasks.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	// LLM validation
	if cfg.LLM.MaxTokens < 1 {
		return fmt.Errorf("max tokens must be positive")
	}
	if cfg.LLM.Temperature < 0 || cfg.LLM.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	return nil
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Create default config content
	configContent := `# HelixCode Server Configuration

server:
  address: "0.0.0.0"
  port: 8080
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 60
  shutdown_timeout: 30

database:
  host: "localhost"
  port: 5432
  user: "helixcode"
  password: "" # Set via HELIX_DATABASE_PASSWORD environment variable
  dbname: "helixcode"
  sslmode: "disable"

redis:
  host: "localhost"
  port: 6379
  password: "" # Set via HELIX_REDIS_PASSWORD environment variable
  db: 0
  enabled: true

auth:
  jwt_secret: "" # Set via HELIX_AUTH_JWT_SECRET environment variable
  token_expiry: 86400
  session_expiry: 604800
  bcrypt_cost: 12

workers:
  health_check_interval: 30
  health_ttl: 120
  max_concurrent_tasks: 10

tasks:
  max_retries: 3
  checkpoint_interval: 300
  cleanup_interval: 3600

llm:
  default_provider: "local"
  providers:
    local: "http://localhost:11434"
    openai: "" # Set API key via environment variable
  max_tokens: 4096
  temperature: 0.7

logging:
  level: "info"
  format: "text"
  output: "stdout"
`

	// Write config file
	if err := os.WriteFile(path, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// GetEnvOrDefault gets an environment variable with a default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvIntOrDefault gets an environment variable as integer with a default value
func GetEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
