// Package testinfra provides utilities for managing test infrastructure
// including Docker/Podman containers for integration and E2E tests.
package testinfra

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/netutil"
	"github.com/redis/go-redis/v9"
)

// Service represents a test infrastructure service
type Service string

const (
	// ServicePostgres is PostgreSQL test database
	ServicePostgres Service = "postgres-test"
	// ServiceRedis is Redis test cache
	ServiceRedis Service = "redis-test"
	// ServiceCognee is Cognee AI service
	ServiceCognee Service = "cognee-test"
	// ServiceChromaDB is ChromaDB vector store
	ServiceChromaDB Service = "chromadb-test"
	// ServiceQdrant is Qdrant vector database
	ServiceQdrant Service = "qdrant-test"
	// ServiceOllama is Ollama LLM service
	ServiceOllama Service = "ollama-test"
	// ServicePrometheus is Prometheus monitoring
	ServicePrometheus Service = "prometheus-test"
	// ServiceGrafana is Grafana dashboards
	ServiceGrafana Service = "grafana-test"
)

// Config holds test infrastructure configuration
type Config struct {
	// PostgreSQL
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// Cognee
	CogneeHost   string
	CogneePort   string
	CogneeAPIKey string

	// ChromaDB
	ChromaDBHost string
	ChromaDBPort string

	// Qdrant
	QdrantHost string
	QdrantPort string

	// Ollama
	OllamaHost string
	OllamaPort string

	// Prometheus
	PrometheusHost string
	PrometheusPort string

	// Grafana
	GrafanaHost     string
	GrafanaPort     string
	GrafanaUser     string
	GrafanaPassword string

	// Timeouts
	StartupTimeout time.Duration
	HealthTimeout  time.Duration
}

// DefaultConfig returns the default test configuration
func DefaultConfig() *Config {
	return &Config{
		PostgresHost:     getEnv("HELIX_TEST_DB_HOST", "localhost"),
		PostgresPort:     getEnv("HELIX_TEST_DB_PORT", "5433"),
		PostgresDB:       getEnv("HELIX_TEST_DB_NAME", "helix_test"),
		PostgresUser:     getEnv("HELIX_TEST_DB_USER", "helix_test"),
		PostgresPassword: getEnv("HELIX_TEST_DB_PASSWORD", "test_password_secure_123"),

		RedisHost:     getEnv("HELIX_TEST_REDIS_HOST", "localhost"),
		RedisPort:     getEnv("HELIX_TEST_REDIS_PORT", "6380"),
		RedisPassword: getEnv("HELIX_TEST_REDIS_PASSWORD", "test_redis_password_123"),

		CogneeHost:   getEnv("HELIX_TEST_COGNEE_HOST", "localhost"),
		CogneePort:   getEnv("HELIX_TEST_COGNEE_PORT", "8001"),
		CogneeAPIKey: getEnv("HELIX_TEST_COGNEE_API_KEY", "test_cognee_key_123"),

		ChromaDBHost: getEnv("HELIX_TEST_CHROMADB_HOST", "localhost"),
		ChromaDBPort: getEnv("HELIX_TEST_CHROMADB_PORT", "8002"),

		QdrantHost: getEnv("HELIX_TEST_QDRANT_HOST", "localhost"),
		QdrantPort: getEnv("HELIX_TEST_QDRANT_PORT", "6333"),

		OllamaHost: getEnv("HELIX_TEST_OLLAMA_HOST", "localhost"),
		OllamaPort: getEnv("HELIX_TEST_OLLAMA_PORT", "11434"),

		PrometheusHost: getEnv("HELIX_TEST_PROMETHEUS_HOST", "localhost"),
		PrometheusPort: getEnv("HELIX_TEST_PROMETHEUS_PORT", "9091"),

		GrafanaHost:     getEnv("HELIX_TEST_GRAFANA_HOST", "localhost"),
		GrafanaPort:     getEnv("HELIX_TEST_GRAFANA_PORT", "3001"),
		GrafanaUser:     getEnv("HELIX_TEST_GRAFANA_USER", "admin"),
		GrafanaPassword: getEnv("HELIX_TEST_GRAFANA_PASSWORD", "admin123"),

		StartupTimeout: 5 * time.Minute,
		HealthTimeout:  30 * time.Second,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// PostgresURL returns the PostgreSQL connection URL
func (c *Config) PostgresURL() string {
	// HXC-185: bracket IPv6 literals in the authority (RFC 3986 §3.2.2).
	return fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable",
		c.PostgresUser, c.PostgresPassword,
		netutil.JoinHostPortStr(c.PostgresHost, c.PostgresPort), c.PostgresDB)
}

// RedisURL returns the Redis connection URL
func (c *Config) RedisURL() string {
	// HXC-185: bracket IPv6 literals in the authority (RFC 3986 §3.2.2).
	return fmt.Sprintf("redis://:%s@%s", c.RedisPassword,
		netutil.JoinHostPortStr(c.RedisHost, c.RedisPort))
}

// CogneeURL returns the Cognee API URL
func (c *Config) CogneeURL() string {
	return "http://" + netutil.JoinHostPortStr(c.CogneeHost, c.CogneePort)
}

// ChromaDBURL returns the ChromaDB API URL
func (c *Config) ChromaDBURL() string {
	return "http://" + netutil.JoinHostPortStr(c.ChromaDBHost, c.ChromaDBPort)
}

// QdrantURL returns the Qdrant API URL
func (c *Config) QdrantURL() string {
	return "http://" + netutil.JoinHostPortStr(c.QdrantHost, c.QdrantPort)
}

// OllamaURL returns the Ollama API URL
func (c *Config) OllamaURL() string {
	return "http://" + netutil.JoinHostPortStr(c.OllamaHost, c.OllamaPort)
}

// Infrastructure manages test infrastructure services
type Infrastructure struct {
	config       *Config
	scriptPath   string
	started      bool
	services     []Service
	mu           sync.Mutex
	httpClient   *http.Client
	postgresDB   *sql.DB
	redisClient  *redis.Client
	cleanupFuncs []func()
}

// New creates a new Infrastructure manager
func New(config *Config) *Infrastructure {
	if config == nil {
		config = DefaultConfig()
	}

	return &Infrastructure{
		config:     config,
		scriptPath: findScriptPath(),
		httpClient: &http.Client{
			Timeout: config.HealthTimeout,
		},
	}
}

func findScriptPath() string {
	// Try common locations
	paths := []string{
		"./scripts/test-infra.sh",
		"../scripts/test-infra.sh",
		"../../scripts/test-infra.sh",
		"../../../scripts/test-infra.sh",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Try using GOPATH or working directory
	wd, _ := os.Getwd()
	return wd + "/scripts/test-infra.sh"
}

// Start starts the test infrastructure
func (i *Infrastructure) Start(ctx context.Context, services ...Service) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.started {
		return nil
	}

	i.services = services

	// Check if infrastructure is already running
	if i.isHealthy(ctx) {
		i.started = true
		return nil
	}

	// Start using script
	if err := i.runScript(ctx, "start"); err != nil {
		return fmt.Errorf("failed to start infrastructure: %w", err)
	}

	// Wait for services to be ready
	if err := i.waitForHealth(ctx); err != nil {
		return fmt.Errorf("infrastructure did not become healthy: %w", err)
	}

	i.started = true
	return nil
}

// Stop stops the test infrastructure
func (i *Infrastructure) Stop(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Run cleanup functions
	for _, cleanup := range i.cleanupFuncs {
		cleanup()
	}
	i.cleanupFuncs = nil

	// Close connections
	if i.postgresDB != nil {
		i.postgresDB.Close()
		i.postgresDB = nil
	}

	if i.redisClient != nil {
		i.redisClient.Close()
		i.redisClient = nil
	}

	i.started = false
	return nil
}

// StopAndClean stops infrastructure and removes all data
func (i *Infrastructure) StopAndClean(ctx context.Context) error {
	if err := i.Stop(ctx); err != nil {
		return err
	}

	return i.runScript(ctx, "clean")
}

func (i *Infrastructure) runScript(ctx context.Context, args ...string) error {
	if _, err := os.Stat(i.scriptPath); err != nil {
		return fmt.Errorf("script not found: %s", i.scriptPath)
	}

	cmd := exec.CommandContext(ctx, "bash", append([]string{i.scriptPath}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script failed: %w\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}

	return nil
}

func (i *Infrastructure) isHealthy(ctx context.Context) bool {
	// Check if core services are responding
	services := map[string]string{
		"chromadb": "http://" + netutil.JoinHostPortStr(i.config.ChromaDBHost, i.config.ChromaDBPort) + "/api/v1/heartbeat",
		"qdrant":   "http://" + netutil.JoinHostPortStr(i.config.QdrantHost, i.config.QdrantPort) + "/healthz",
	}

	for name, url := range services {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := i.httpClient.Do(req)
		if err != nil {
			return false
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false
		}

		_ = name // used for debugging
	}

	return true
}

func (i *Infrastructure) waitForHealth(ctx context.Context) error {
	timeout := time.After(i.config.StartupTimeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for infrastructure to be healthy")
		case <-ticker.C:
			if i.isHealthy(ctx) {
				return nil
			}
		}
	}
}

// GetPostgresDB returns a PostgreSQL database connection
func (i *Infrastructure) GetPostgresDB(ctx context.Context) (*sql.DB, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.postgresDB != nil {
		return i.postgresDB, nil
	}

	db, err := sql.Open("pgx", i.config.PostgresURL())
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	i.postgresDB = db
	return db, nil
}

// GetRedisClient returns a Redis client
func (i *Infrastructure) GetRedisClient(ctx context.Context) (*redis.Client, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.redisClient != nil {
		return i.redisClient, nil
	}

	client := redis.NewClient(&redis.Options{
		// HXC-185: bracket IPv6 literals so the resolver accepts the address.
		Addr:     netutil.JoinHostPortStr(i.config.RedisHost, i.config.RedisPort),
		Password: i.config.RedisPassword,
		DB:       0,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	i.redisClient = client
	return client, nil
}

// CheckService checks if a specific service is healthy
func (i *Infrastructure) CheckService(ctx context.Context, service Service) error {
	var url string
	switch service {
	case ServicePostgres:
		db, err := i.GetPostgresDB(ctx)
		if err != nil {
			return err
		}
		return db.PingContext(ctx)

	case ServiceRedis:
		client, err := i.GetRedisClient(ctx)
		if err != nil {
			return err
		}
		return client.Ping(ctx).Err()

	case ServiceCognee:
		url = fmt.Sprintf("%s/health", i.config.CogneeURL())

	case ServiceChromaDB:
		url = fmt.Sprintf("%s/api/v1/heartbeat", i.config.ChromaDBURL())

	case ServiceQdrant:
		url = fmt.Sprintf("%s/healthz", i.config.QdrantURL())

	case ServiceOllama:
		url = fmt.Sprintf("%s/api/tags", i.config.OllamaURL())

	case ServicePrometheus:
		url = "http://" + netutil.JoinHostPortStr(i.config.PrometheusHost, i.config.PrometheusPort) + "/-/healthy"

	case ServiceGrafana:
		url = "http://" + netutil.JoinHostPortStr(i.config.GrafanaHost, i.config.GrafanaPort) + "/api/health"

	default:
		return fmt.Errorf("unknown service: %s", service)
	}

	if url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		resp, err := i.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("service %s returned status %d", service, resp.StatusCode)
		}
	}

	return nil
}

// RegisterCleanup registers a cleanup function to be called on Stop
func (i *Infrastructure) RegisterCleanup(fn func()) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.cleanupFuncs = append(i.cleanupFuncs, fn)
}

// Config returns the infrastructure configuration
func (i *Infrastructure) Config() *Config {
	return i.config
}

// RequireService is a test helper that skips if a service is unavailable
func RequireService(t interface{ Skip(...interface{}) }, ctx context.Context, infra *Infrastructure, service Service) {
	if err := infra.CheckService(ctx, service); err != nil {
		t.Skip(fmt.Sprintf("Service %s not available: %v", service, err))  // SKIP-OK: #legacy-untriaged
	}
}

// SkipIfNoInfrastructure skips the test if infrastructure is not available
func SkipIfNoInfrastructure(t interface{ Skip(...interface{}) }) {
	if os.Getenv("HELIX_TEST_INFRA") != "true" {
		t.Skip("Test infrastructure not available (set HELIX_TEST_INFRA=true)")  // SKIP-OK: #legacy-untriaged
	}
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t interface{ Skip(...interface{}); Short() bool }) {
	if t.Short() {
		t.Skip("Skipping in short mode")  // SKIP-OK: #short-mode
	}
}

// MustStartInfrastructure starts infrastructure or skips the test
func MustStartInfrastructure(t interface {
	Skip(...interface{})
	Fatalf(string, ...interface{})
	Cleanup(func())
}, ctx context.Context) *Infrastructure {
	SkipIfNoInfrastructure(t)

	infra := New(nil)
	if err := infra.Start(ctx); err != nil {
		// Check if services are already running externally
		if infra.isHealthy(ctx) {
			t.Cleanup(func() {
				infra.Stop(ctx)
			})
			return infra
		}
		t.Fatalf("Failed to start infrastructure: %v", err)
	}

	t.Cleanup(func() {
		infra.Stop(ctx)
	})

	return infra
}

// HTTPClient returns an HTTP client configured for test infrastructure
type HTTPClient struct {
	baseURL string
	client  *http.Client
	headers map[string]string
}

// NewHTTPClient creates a new HTTP client for a service
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: timeout,
		},
		headers: make(map[string]string),
	}
}

// SetHeader sets a default header for all requests
func (c *HTTPClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// Get performs a GET request
func (c *HTTPClient) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.client.Do(req)
}

// Post performs a POST request
func (c *HTTPClient) Post(ctx context.Context, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.client.Do(req)
}

// Delete performs a DELETE request
func (c *HTTPClient) Delete(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.client.Do(req)
}
