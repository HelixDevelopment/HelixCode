-- HelixCode Test Database Initialization
-- Complete schema with sample data for distributed testing

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

CREATE INDEX users_email_idx ON users(email);
CREATE INDEX users_username_idx ON users(username);
CREATE INDEX users_created_at_idx ON users(created_at);

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

CREATE INDEX user_sessions_token_idx ON user_sessions(session_token);
CREATE INDEX user_sessions_user_id_idx ON user_sessions(user_id);
CREATE INDEX user_sessions_expires_at_idx ON user_sessions(expires_at);

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

ALTER TABLE workers ADD CONSTRAINT workers_hostname_unique UNIQUE (hostname);
CREATE INDEX workers_status_idx ON workers(status);
CREATE INDEX workers_health_status_idx ON workers(health_status);
CREATE INDEX workers_last_heartbeat_idx ON workers(last_heartbeat);
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

CREATE INDEX worker_metrics_worker_id_idx ON worker_metrics(worker_id);
CREATE INDEX worker_metrics_recorded_at_idx ON worker_metrics(recorded_at);

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

CREATE INDEX distributed_tasks_status_idx ON distributed_tasks(status);
CREATE INDEX distributed_tasks_criticality_idx ON distributed_tasks(criticality);
CREATE INDEX distributed_tasks_assigned_worker_idx ON distributed_tasks(assigned_worker_id);
CREATE INDEX distributed_tasks_priority_idx ON distributed_tasks(priority);
CREATE INDEX distributed_tasks_dependencies_idx ON distributed_tasks USING GIN (dependencies);
CREATE INDEX distributed_tasks_created_at_idx ON distributed_tasks(created_at);

CREATE TABLE task_checkpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES distributed_tasks(id) ON DELETE CASCADE,
    checkpoint_name VARCHAR(255) NOT NULL,
    checkpoint_data JSONB NOT NULL,
    worker_id UUID NOT NULL REFERENCES workers(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX task_checkpoints_task_id_idx ON task_checkpoints(task_id);
CREATE INDEX task_checkpoints_worker_id_idx ON task_checkpoints(worker_id);
CREATE INDEX task_checkpoints_created_at_idx ON task_checkpoints(created_at);

CREATE TABLE worker_connectivity_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    worker_id UUID NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('connected', 'disconnected', 'reconnected', 'heartbeat_missed')),
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX worker_connectivity_events_worker_id_idx ON worker_connectivity_events(worker_id);
CREATE INDEX worker_connectivity_events_event_type_idx ON worker_connectivity_events(event_type);
CREATE INDEX worker_connectivity_events_created_at_idx ON worker_connectivity_events(created_at);

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

CREATE INDEX projects_owner_id_idx ON projects(owner_id);
CREATE INDEX projects_status_idx ON projects(status);
CREATE INDEX projects_created_at_idx ON projects(created_at);

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

CREATE INDEX sessions_project_id_idx ON sessions(project_id);
CREATE INDEX sessions_status_idx ON sessions(status);
CREATE INDEX sessions_session_type_idx ON sessions(session_type);
CREATE INDEX sessions_current_task_id_idx ON sessions(current_task_id);

-- =============================================
-- SAMPLE DATA FOR TESTING
-- =============================================

-- Test user
INSERT INTO users (username, email, password_hash, display_name, is_active, is_verified)
VALUES ('testuser', 'test@example.com', '$2a$10$example.hash.here', 'Test User', true, true);

-- Test workers
INSERT INTO workers (hostname, display_name, ssh_config, capabilities, resources, status, health_status, max_concurrent_tasks)
VALUES
('worker1.local', 'Worker Node 1', '{"host": "worker1", "port": 22, "user": "root"}', ARRAY['golang', 'testing', 'building'], '{"cpu_cores": 4, "memory_gb": 8, "disk_gb": 50}', 'active', 'healthy', 5),
('worker2.local', 'Worker Node 2', '{"host": "worker2", "port": 22, "user": "root"}', ARRAY['golang', 'testing', 'building'], '{"cpu_cores": 4, "memory_gb": 8, "disk_gb": 50}', 'active', 'healthy', 5),
('worker3.local', 'Worker Node 3', '{"host": "worker3", "port": 22, "user": "root"}', ARRAY['golang', 'testing', 'building'], '{"cpu_cores": 4, "memory_gb": 8, "disk_gb": 50}', 'active', 'healthy', 5);

-- Test project
INSERT INTO projects (name, description, owner_id, workspace_path, status)
SELECT 'Test Project', 'A test project for distributed computing validation', id, '/tmp/test-project', 'active'
FROM users WHERE username = 'testuser';

-- Sample distributed tasks
INSERT INTO distributed_tasks (task_type, task_data, status, priority, criticality)
VALUES
('build', '{"command": "go build ./cmd/server", "working_dir": "/app"}', 'pending', 5, 'normal'),
('test', '{"command": "go test ./...", "working_dir": "/app"}', 'pending', 4, 'normal'),
('lint', '{"command": "golangci-lint run", "working_dir": "/app"}', 'pending', 3, 'low');

-- Test session
INSERT INTO sessions (project_id, name, description, session_type, status, context_data)
SELECT p.id, 'Test Session', 'Testing distributed task execution', 'testing', 'active', '{"test_mode": true}'
FROM projects p WHERE name = 'Test Project';