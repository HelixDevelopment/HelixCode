package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DatabaseInterface defines the database operations interface.
// This allows mocking for testing while maintaining pgx semantics.
// All methods mirror the pgxpool.Pool API for minimal friction.
type DatabaseInterface interface {
	// Exec executes a query without returning any rows.
	// Typically used for INSERT, UPDATE, DELETE, and DDL statements.
	// Returns CommandTag with information about the execution (rows affected, etc.)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)

	// Query executes a query that returns rows.
	// Typically used for SELECT statements.
	// Returns Rows which must be closed after use.
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)

	// QueryRow executes a query that is expected to return at most one row.
	// Typically used for SELECT statements with LIMIT 1 or aggregate functions.
	// Returns Row which handles the result or error.
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row

	// Ping verifies the database connection is alive.
	// Used for health checks and connection validation.
	Ping(ctx context.Context) error

	// Close closes all connections in the pool.
	// Should be called when the database is no longer needed.
	Close()
}

// Ensure Database implements DatabaseInterface at compile time.
// This will cause a compilation error if Database doesn't implement all interface methods.
var _ DatabaseInterface = (*Database)(nil)
