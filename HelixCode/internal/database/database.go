package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
)

// Database represents the database connection pool
type Database struct {
	Pool *pgxpool.Pool
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// New creates a new database connection pool
func New(config Config) (*Database, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %v", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %v", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	log.Println("✅ Database connection established successfully")

	return &Database{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		log.Println("✅ Database connection pool closed")
	}
}

// Exec executes a query without returning any rows.
// Implements DatabaseInterface.
func (db *Database) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return db.Pool.Exec(ctx, sql, arguments...)
}

// Query executes a query that returns rows.
// Implements DatabaseInterface.
func (db *Database) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.Pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row.
// Implements DatabaseInterface.
func (db *Database) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.Pool.QueryRow(ctx, sql, args...)
}

// Ping verifies the database connection.
// Implements DatabaseInterface.
func (db *Database) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// InitializeSchema creates the database schema if it doesn't exist
func (db *Database) InitializeSchema() error {
	ctx := context.Background()

	// Check if schema exists
	var schemaExists bool
	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'users'
		)
	`).Scan(&schemaExists)

	if err != nil {
		return fmt.Errorf("failed to check schema existence: %v", err)
	}

	if schemaExists {
		log.Println("✅ Database schema already exists")
		return nil
	}

	log.Println("🔧 Creating database schema...")

	// Execute schema creation
	_, err = db.Pool.Exec(ctx, createSchemaSQL)
	if err != nil {
		return fmt.Errorf("failed to create schema: %v", err)
	}

	log.Println("✅ Database schema created successfully")
	return nil
}

// GetDB returns a standard sql.DB for compatibility with other libraries
func (db *Database) GetDB() (*sql.DB, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	// Convert pgxpool.Pool to *sql.DB
	return stdlib.OpenDBFromPool(db.Pool), nil
}

// HealthCheck performs a health check on the database
func (db *Database) HealthCheck() error {
	if db.Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Pool.Ping(ctx)
}

// createSchemaSQL contains the complete database schema
const createSchemaSQL = `
-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================
-- 1. USERS & AUTHENTICATION
-- =============================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    avatar_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    mfa_secret VARCHAR(255),
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX users_email_idx ON users (email);
CREATE INDEX users_username_idx ON users (username);
CREATE INDEX users_created_at_idx ON users (created_at);

CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(512) UNIQUE NOT NULL,
    client_type VARCHAR(50) NOT NULL CHECK (client_type IN ('terminal_ui', 'cli', 'rest_api', 'mobile_ios', 'mobile_android')),
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX user_sessions_token_idx ON user_sessions (session_token);
CREATE INDEX user_sessions_user_id_idx ON user_sessions (user_id);
CREATE INDEX user_sessions_expires_at_idx ON user_sessions (expires_at);

-- =============================================
-- 2. WORKERS & DISTRIBUTED COMPUTING
-- =============================================

CREATE TABLE workers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hostname VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    ssh_config JSONB NOT NULL,
    capabilities TEXT[] NOT NULL DEFAULT '{}',
    resources JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active' 
        CHECK (status IN ('active', 'inactive', 'maintenance', 'failed', 'offline')),
    health_status VARCHAR(50) NOT NULL DEFAULT 'healthy'
        CHECK (health_status IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
    last_heartbeat TIMESTAMPTZ,
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_percent DECIMAL(5,2),
    disk_usage_percent DECIMAL(5,2),
    current_tasks_count INTEGER NOT NULL DEFAULT 0,
    max_concurrent_tasks INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX workers_hostname_unique ON workers (hostname);
CREATE INDEX workers_status_idx ON workers (status);
CREATE INDEX workers_health_status_idx ON workers (health_status);
CREATE INDEX workers_last_heartbeat_idx ON workers (last_heartbeat);
CREATE INDEX workers_capabilities_idx ON workers USING GIN (capabilities);

CREATE TABLE worker_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    worker_id UUID NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_percent DECIMAL(5,2),
    disk_usage_percent DECIMAL(5,2),
    network_rx_bytes BIGINT,
    network_tx_bytes BIGINT,
    current_tasks_count INTEGER,
    temperature_celsius DECIMAL(5,2),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX worker_metrics_worker_id_idx ON worker_metrics (worker_id);
CREATE INDEX worker_metrics_recorded_at_idx ON worker_metrics (recorded_at);

-- =============================================
-- 3. WORK PRESERVATION & DISTRIBUTED TASKS
-- =============================================

CREATE TABLE distributed_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_type VARCHAR(100) NOT NULL,
    task_data JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'assigned', 'running', 'completed', 'failed', 'paused', 'waiting_for_worker')),
    priority INTEGER NOT NULL DEFAULT 5,
    criticality VARCHAR(20) NOT NULL DEFAULT 'normal'
        CHECK (criticality IN ('low', 'normal', 'high', 'critical')),
    assigned_worker_id UUID REFERENCES workers(id),
    original_worker_id UUID REFERENCES workers(id),
    dependencies UUID[] DEFAULT '{}',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    error_message TEXT,
    result_data JSONB,
    checkpoint_data JSONB,
    estimated_duration INTERVAL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX distributed_tasks_status_idx ON distributed_tasks (status);
CREATE INDEX distributed_tasks_criticality_idx ON distributed_tasks (criticality);
CREATE INDEX distributed_tasks_assigned_worker_idx ON distributed_tasks (assigned_worker_id);
CREATE INDEX distributed_tasks_priority_idx ON distributed_tasks (priority);
CREATE INDEX distributed_tasks_dependencies_idx ON distributed_tasks USING GIN (dependencies);
CREATE INDEX distributed_tasks_created_at_idx ON distributed_tasks (created_at);

CREATE TABLE task_checkpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES distributed_tasks(id) ON DELETE CASCADE,
    checkpoint_name VARCHAR(255) NOT NULL,
    checkpoint_data JSONB NOT NULL,
    worker_id UUID NOT NULL REFERENCES workers(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX task_checkpoints_task_id_idx ON task_checkpoints (task_id);
CREATE INDEX task_checkpoints_worker_id_idx ON task_checkpoints (worker_id);
CREATE INDEX task_checkpoints_created_at_idx ON task_checkpoints (created_at);

CREATE TABLE worker_connectivity_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    worker_id UUID NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('connected', 'disconnected', 'reconnected', 'heartbeat_missed')),
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX worker_connectivity_events_worker_id_idx ON worker_connectivity_events (worker_id);
CREATE INDEX worker_connectivity_events_event_type_idx ON worker_connectivity_events (event_type);
CREATE INDEX worker_connectivity_events_created_at_idx ON worker_connectivity_events (created_at);

-- =============================================
-- 4. PROJECTS & SESSIONS
-- =============================================

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL REFERENCES users(id),
    workspace_path TEXT,
    git_repository_url TEXT,
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'archived', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX projects_owner_id_idx ON projects (owner_id);
CREATE INDEX projects_status_idx ON projects (status);
CREATE INDEX projects_created_at_idx ON projects (created_at);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    session_type VARCHAR(50) NOT NULL
        CHECK (session_type IN ('planning', 'building', 'testing', 'refactoring', 'debugging')),
    status VARCHAR(50) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'paused', 'completed', 'failed', 'waiting_for_worker')),
    context_data JSONB NOT NULL DEFAULT '{}',
    token_count INTEGER NOT NULL DEFAULT 0,
    current_task_id UUID REFERENCES distributed_tasks(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX sessions_project_id_idx ON sessions (project_id);
CREATE INDEX sessions_status_idx ON sessions (status);
CREATE INDEX sessions_session_type_idx ON sessions (session_type);
CREATE INDEX sessions_current_task_id_idx ON sessions (current_task_id);

-- =============================================
-- 5. LLM PROVIDERS & MODELS
-- =============================================

CREATE TABLE llm_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    provider_type VARCHAR(50) NOT NULL
        CHECK (provider_type IN ('local', 'openai', 'anthropic', 'gemini', 'qwen', 'xai', 'openrouter', 'copilot', 'custom')),
    api_key TEXT,
    api_endpoint TEXT,
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'failed')),
    health_status VARCHAR(50) NOT NULL DEFAULT 'unknown'
        CHECK (health_status IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
    last_health_check TIMESTAMPTZ,
    error_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX llm_providers_name_idx ON llm_providers (name);
CREATE INDEX llm_providers_type_idx ON llm_providers (provider_type);
CREATE INDEX llm_providers_status_idx ON llm_providers (status);

CREATE TABLE llm_models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES llm_providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    capabilities TEXT[] NOT NULL DEFAULT '{}',
    context_length INTEGER NOT NULL DEFAULT 4096,
    max_tokens INTEGER NOT NULL DEFAULT 2048,
    supports_tools BOOLEAN NOT NULL DEFAULT false,
    supports_vision BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'deprecated')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX llm_models_provider_id_idx ON llm_models (provider_id);
CREATE INDEX llm_models_name_idx ON llm_models (name);
CREATE INDEX llm_models_model_id_idx ON llm_models (model_id);
CREATE INDEX llm_models_status_idx ON llm_models (status);
CREATE INDEX llm_models_capabilities_idx ON llm_models USING GIN (capabilities);

-- =============================================
-- 6. MCP INTEGRATION
-- =============================================

CREATE TABLE mcp_servers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    transport_type VARCHAR(50) NOT NULL
        CHECK (transport_type IN ('stdio', 'sse', 'http', 'websocket')),
    command TEXT,
    args TEXT[],
    url TEXT,
    env_vars JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'failed')),
    last_health_check TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX mcp_servers_name_idx ON mcp_servers (name);
CREATE INDEX mcp_servers_transport_idx ON mcp_servers (transport_type);
CREATE INDEX mcp_servers_status_idx ON mcp_servers (status);

CREATE TABLE tools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    mcp_server_id UUID REFERENCES mcp_servers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    parameters JSONB NOT NULL DEFAULT '{}',
    permissions TEXT[] NOT NULL DEFAULT '{}',
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX tools_mcp_server_id_idx ON tools (mcp_server_id);
CREATE INDEX tools_name_idx ON tools (name);
CREATE INDEX tools_is_enabled_idx ON tools (is_enabled);

-- =============================================
-- 7. NOTIFICATIONS
-- =============================================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    notification_type VARCHAR(50) NOT NULL
        CHECK (notification_type IN ('info', 'warning', 'error', 'success', 'alert')),
    priority VARCHAR(50) NOT NULL DEFAULT 'medium'
        CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    channels TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed', 'cancelled')),
    metadata JSONB NOT NULL DEFAULT '{}',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX notifications_user_id_idx ON notifications (user_id);
CREATE INDEX notifications_type_idx ON notifications (notification_type);
CREATE INDEX notifications_priority_idx ON notifications (priority);
CREATE INDEX notifications_status_idx ON notifications (status);
CREATE INDEX notifications_created_at_idx ON notifications (created_at);

-- =============================================
-- 8. AUDIT LOGS
-- =============================================

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    details JSONB NOT NULL DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(50) NOT NULL
        CHECK (status IN ('success', 'failure', 'error')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX audit_logs_user_id_idx ON audit_logs (user_id);
CREATE INDEX audit_logs_action_idx ON audit_logs (action);
CREATE INDEX audit_logs_resource_type_idx ON audit_logs (resource_type);
CREATE INDEX audit_logs_resource_id_idx ON audit_logs (resource_id);
CREATE INDEX audit_logs_created_at_idx ON audit_logs (created_at);
CREATE INDEX audit_logs_status_idx ON audit_logs (status);
`
