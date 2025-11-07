package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the mock LLM provider configuration
type Config struct {
	Port            string
	ResponseDelay   time.Duration
	EnableLogging   bool
	FixturesPath    string
	DefaultModel    string
	SupportedModels []string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	port := getEnv("MOCK_LLM_PORT", "8090")
	delayMs := getEnvInt("MOCK_LLM_DELAY_MS", 100)
	enableLogging := getEnvBool("MOCK_LLM_LOGGING", true)
	fixturesPath := getEnv("MOCK_LLM_FIXTURES", "./responses/fixtures.json")
	defaultModel := getEnv("MOCK_LLM_DEFAULT_MODEL", "mock-gpt-4")

	return &Config{
		Port:          port,
		ResponseDelay: time.Duration(delayMs) * time.Millisecond,
		EnableLogging: enableLogging,
		FixturesPath:  fixturesPath,
		DefaultModel:  defaultModel,
		SupportedModels: []string{
			"mock-gpt-4",
			"mock-gpt-3.5-turbo",
			"mock-claude-3",
			"mock-llama-3-8b",
			"mock-mixtral-8x7b",
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
