package config

import (
	"os"
	"strconv"
	"strings"
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

	// AllowedOrigins is the CORS allowlist: the exact set of browser origins
	// permitted to make cross-origin requests to this service. It is read
	// from MOCK_LLM_ALLOWED_ORIGINS (comma-separated) and defaults to EMPTY,
	// i.e. default-deny — no cross-origin browser access at all.
	//
	// CONST-045 / §11.4.28: no origin is hardcoded here. The allowlist is
	// configuration; adding an origin means setting the environment
	// variable, never editing this file.
	//
	// The empty default does not break the normal workflow: this service is
	// driven by Go/e2e HTTP clients, which are not browsers, send no Origin
	// header, and ignore CORS response headers entirely. Only a browser
	// front-end pointed at this mock needs the variable set, e.g.
	// MOCK_LLM_ALLOWED_ORIGINS=http://localhost:3000
	AllowedOrigins []string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	port := getEnv("MOCK_LLM_PORT", "8090")
	delayMs := getEnvInt("MOCK_LLM_DELAY_MS", 100)
	enableLogging := getEnvBool("MOCK_LLM_LOGGING", true)
	fixturesPath := getEnv("MOCK_LLM_FIXTURES", "./responses/fixtures.json")
	defaultModel := getEnv("MOCK_LLM_DEFAULT_MODEL", "mock-gpt-4")

	return &Config{
		Port:           port,
		ResponseDelay:  time.Duration(delayMs) * time.Millisecond,
		EnableLogging:  enableLogging,
		FixturesPath:   fixturesPath,
		DefaultModel:   defaultModel,
		AllowedOrigins: getEnvList("MOCK_LLM_ALLOWED_ORIGINS"),
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
