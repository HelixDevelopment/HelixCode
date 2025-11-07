//go:build integration
// +build integration

package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_Integration tests database connection with real PostgreSQL
func TestNew_Integration(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify connection pool is configured
	assert.NotNil(t, db.Pool)

	// Test health check
	err = db.HealthCheck()
	assert.NoError(t, err)
}

// TestInitializeSchema_Integration tests schema initialization
func TestInitializeSchema_Integration(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Clean up existing schema if any
	_, err = db.Pool.Exec(context.Background(), `
		DROP SCHEMA IF EXISTS public CASCADE;
		CREATE SCHEMA public;
	`)
	require.NoError(t, err)

	// Initialize schema
	err = db.InitializeSchema()
	assert.NoError(t, err)

	// Verify tables were created
	var tableCount int
	err = db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
	`).Scan(&tableCount)
	require.NoError(t, err)
	assert.Greater(t, tableCount, 10, "Expected at least 10 tables to be created")

	// Verify specific tables exist
	tables := []string{
		"users", "user_sessions", "workers", "worker_metrics",
		"distributed_tasks", "task_checkpoints", "worker_connectivity_events",
		"projects", "sessions", "llm_providers", "llm_models",
		"mcp_servers", "tools", "notifications", "audit_logs",
	}

	for _, tableName := range tables {
		var exists bool
		err = db.Pool.QueryRow(context.Background(), `
			SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, tableName).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Table %s should exist", tableName)
	}
}

// TestInitializeSchema_AlreadyExists tests idempotent schema initialization
func TestInitializeSchema_AlreadyExists(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Initialize schema first time
	err = db.InitializeSchema()
	require.NoError(t, err)

	// Initialize schema second time (should not error)
	err = db.InitializeSchema()
	assert.NoError(t, err)
}

// TestClose_Integration tests closing database connection
func TestClose_Integration(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Close should not panic
	db.Close()

	// Pool should be closed
	err = db.HealthCheck()
	assert.Error(t, err, "Health check should fail after close")
}

// TestHealthCheck_Integration tests health check with real database
func TestHealthCheck_Integration(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Health check should pass
	err = db.HealthCheck()
	assert.NoError(t, err)

	// Test health check with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = db.Pool.Ping(ctx)
	assert.NoError(t, err)
}

// TestGetDB_Integration tests getting standard sql.DB
func TestGetDB_Integration(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Get standard sql.DB
	sqlDB, err := db.GetDB()
	require.NoError(t, err)
	assert.NotNil(t, sqlDB)

	// Verify it works
	err = sqlDB.Ping()
	assert.NoError(t, err)

	// Test a simple query
	var result int
	err = sqlDB.QueryRow("SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

// TestConnectionPool_Integration tests connection pool configuration
func TestConnectionPool_Integration(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify pool stats
	stats := db.Pool.Stat()
	assert.Greater(t, stats.MaxConns(), int32(0), "Max connections should be configured")
	assert.GreaterOrEqual(t, stats.MaxConns(), int32(5), "Max connections should be at least 5")
}

// TestNew_InvalidHost tests connection with invalid host
func TestNew_InvalidHost(t *testing.T) {
	config := Config{
		Host:     "invalid-host-that-does-not-exist",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to ping database")
}

// TestNew_InvalidCredentials tests connection with wrong credentials
func TestNew_InvalidCredentials(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "wrong_user",
		Password: "wrong_password",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	assert.Error(t, err)
	assert.Nil(t, db)
}

// TestCRUD_Integration tests basic CRUD operations
func TestCRUD_Integration(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Ensure schema exists
	err = db.InitializeSchema()
	require.NoError(t, err)

	ctx := context.Background()

	// Insert a test user
	var userID string
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, display_name, is_active, is_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "testuser", "test@example.com", "hash123", "Test User", true, false).Scan(&userID)
	require.NoError(t, err)
	assert.NotEmpty(t, userID)

	// Read the user
	var username, email string
	err = db.Pool.QueryRow(ctx, `
		SELECT username, email FROM users WHERE id = $1
	`, userID).Scan(&username, &email)
	require.NoError(t, err)
	assert.Equal(t, "testuser", username)
	assert.Equal(t, "test@example.com", email)

	// Update the user
	_, err = db.Pool.Exec(ctx, `
		UPDATE users SET display_name = $1 WHERE id = $2
	`, "Updated Name", userID)
	assert.NoError(t, err)

	// Delete the user
	_, err = db.Pool.Exec(ctx, `
		DELETE FROM users WHERE id = $1
	`, userID)
	assert.NoError(t, err)

	// Verify deletion
	var count int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE id = $1
	`, userID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestTransaction_Integration tests transaction support
func TestTransaction_Integration(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5433,
		User:     "helix_test",
		Password: "test_password_secure_123",
		DBName:   "helix_test",
		SSLMode:  "disable",
	}

	db, err := New(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Ensure schema exists
	err = db.InitializeSchema()
	require.NoError(t, err)

	ctx := context.Background()

	// Begin transaction
	tx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)

	// Insert user in transaction
	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, is_active, is_verified)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, "txuser", "tx@example.com", "hash123", true, false).Scan(&userID)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback(ctx)
	assert.NoError(t, err)

	// Verify user was not inserted
	var count int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE username = $1
	`, "txuser").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "User should not exist after rollback")
}
