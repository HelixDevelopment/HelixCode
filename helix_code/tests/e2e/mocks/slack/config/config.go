package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the configuration for the mock Slack service
type Config struct {
	Port            string
	ResponseDelay   time.Duration
	EnableLogging   bool
	StorageCapacity int
	WebhookSecret   string

	// AllowedOrigins is the CORS allowlist: the exact set of browser origins
	// permitted to make cross-origin requests to this service. It is read
	// from MOCK_SLACK_ALLOWED_ORIGINS (comma-separated) and defaults to
	// EMPTY, i.e. default-deny — no cross-origin browser access at all.
	//
	// CONST-045 / §11.4.28: no origin is hardcoded here. The allowlist is
	// configuration; adding an origin means setting the environment
	// variable, never editing this file.
	//
	// The empty default does not break the normal workflow: this service is
	// driven by Go/e2e HTTP clients, which are not browsers, send no Origin
	// header, and ignore CORS response headers entirely. Only a browser
	// front-end pointed at this mock needs the variable set, e.g.
	// MOCK_SLACK_ALLOWED_ORIGINS=http://localhost:3000
	AllowedOrigins []string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:            getEnv("MOCK_SLACK_PORT", "8091"),
		ResponseDelay:   time.Duration(getEnvInt("MOCK_SLACK_DELAY_MS", 50)) * time.Millisecond,
		EnableLogging:   getEnvBool("MOCK_SLACK_LOGGING", true),
		StorageCapacity: getEnvInt("MOCK_SLACK_STORAGE_CAPACITY", 1000),
		WebhookSecret:   getEnv("MOCK_SLACK_WEBHOOK_SECRET", "test-webhook-secret"),
		AllowedOrigins:  getEnvList("MOCK_SLACK_ALLOWED_ORIGINS"),
	}
}

// getEnvList reads a comma-separated environment variable into a slice,
// trimming surrounding whitespace and discarding empty entries. An unset or
// empty variable yields nil — for an allowlist that means default-deny.
func getEnvList(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}

	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
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
