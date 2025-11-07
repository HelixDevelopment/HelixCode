package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for the mock Slack service
type Config struct {
	Port            string
	ResponseDelay   time.Duration
	EnableLogging   bool
	StorageCapacity int
	WebhookSecret   string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:            getEnv("MOCK_SLACK_PORT", "8091"),
		ResponseDelay:   time.Duration(getEnvInt("MOCK_SLACK_DELAY_MS", 50)) * time.Millisecond,
		EnableLogging:   getEnvBool("MOCK_SLACK_LOGGING", true),
		StorageCapacity: getEnvInt("MOCK_SLACK_STORAGE_CAPACITY", 1000),
		WebhookSecret:   getEnv("MOCK_SLACK_WEBHOOK_SECRET", "test-webhook-secret"),
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
