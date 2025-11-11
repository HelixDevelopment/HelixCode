package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestMockDatabaseExec tests the Exec method on MockDatabase
func TestMockDatabaseExec(t *testing.T) {
	t.Run("successful exec", func(t *testing.T) {
		mockDB := NewMockDatabase()
		mockDB.MockExecSuccess(1)

		ctx := context.Background()
		tag, err := mockDB.Exec(ctx, "INSERT INTO users VALUES ($1)", []interface{}{"test"})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), tag.RowsAffected())
		mockDB.AssertExpectations(t)
	})

	t.Run("exec with error", func(t *testing.T) {
		mockDB := NewMockDatabase()
		expectedErr := fmt.Errorf("database error")
		mockDB.MockExecError(expectedErr)

		ctx := context.Background()
		tag, err := mockDB.Exec(ctx, "INSERT INTO users VALUES ($1)", []interface{}{"test"})

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, int64(0), tag.RowsAffected())
		mockDB.AssertExpectations(t)
	})

	t.Run("exec with specific SQL", func(t *testing.T) {
		mockDB := NewMockDatabase()
		sql := "UPDATE users SET name = $1 WHERE id = $2"
		mockDB.MockExecWithSQL(sql, 1)

		ctx := context.Background()
		tag, err := mockDB.Exec(ctx, sql, []interface{}{"John", "123"})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), tag.RowsAffected())
		mockDB.AssertExpectations(t)
	})

	t.Run("exec called once", func(t *testing.T) {
		mockDB := NewMockDatabase()
		mockDB.MockExecSuccessOnce(1)

		ctx := context.Background()
		_, err := mockDB.Exec(ctx, "INSERT", []interface{}{})

		assert.NoError(t, err)
		mockDB.AssertExpectations(t)
	})
}

// TestMockDatabaseQueryRow tests the QueryRow method on MockDatabase
func TestMockDatabaseQueryRow(t *testing.T) {
	t.Run("successful query row", func(t *testing.T) {
		mockDB := NewMockDatabase()
		mockRow := NewMockRowWithValues("user-123", "John Doe", 30)
		mockDB.MockQueryRowSuccess(mockRow)

		ctx := context.Background()
		row := mockDB.QueryRow(ctx, "SELECT id, name, age FROM users WHERE id = $1", []interface{}{"user-123"})

		var id, name string
		var age int
		err := row.Scan(&id, &name, &age)

		assert.NoError(t, err)
		assert.Equal(t, "user-123", id)
		assert.Equal(t, "John Doe", name)
		assert.Equal(t, 30, age)
		mockDB.AssertExpectations(t)
	})

	t.Run("query row with error", func(t *testing.T) {
		mockDB := NewMockDatabase()
		mockDB.MockQueryRowError(pgx.ErrNoRows)

		ctx := context.Background()
		row := mockDB.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", []interface{}{"nonexistent"})

		var id string
		err := row.Scan(&id)

		assert.Error(t, err)
		assert.Equal(t, pgx.ErrNoRows, err)
		mockDB.AssertExpectations(t)
	})

	t.Run("query row with bool and int64", func(t *testing.T) {
		mockDB := NewMockDatabase()
		mockRow := NewMockRowWithValues("task-456", true, int64(12345))
		mockDB.MockQueryRowSuccess(mockRow)

		ctx := context.Background()
		row := mockDB.QueryRow(ctx, "SELECT id, active, count FROM tasks", []interface{}{})

		var id string
		var active bool
		var count int64
		err := row.Scan(&id, &active, &count)

		assert.NoError(t, err)
		assert.Equal(t, "task-456", id)
		assert.True(t, active)
		assert.Equal(t, int64(12345), count)
		mockDB.AssertExpectations(t)
	})
}

// TestMockDatabaseQuery tests the Query method on MockDatabase
func TestMockDatabaseQuery(t *testing.T) {
	t.Run("query returns nil rows", func(t *testing.T) {
		mockDB := NewMockDatabase()
		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, nil)

		ctx := context.Background()
		rows, err := mockDB.Query(ctx, "SELECT * FROM users", []interface{}{})

		assert.NoError(t, err)
		assert.Nil(t, rows)
		mockDB.AssertExpectations(t)
	})

	t.Run("query with error", func(t *testing.T) {
		mockDB := NewMockDatabase()
		expectedErr := fmt.Errorf("query failed")
		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, expectedErr)

		ctx := context.Background()
		rows, err := mockDB.Query(ctx, "SELECT * FROM users", []interface{}{})

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, rows)
		mockDB.AssertExpectations(t)
	})
}

// TestMockDatabasePing tests the Ping method on MockDatabase
func TestMockDatabasePing(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		mockDB := NewMockDatabase()
		mockDB.MockPingSuccess()

		err := mockDB.Ping(context.Background())

		assert.NoError(t, err)
		mockDB.AssertExpectations(t)
	})

	t.Run("ping with error", func(t *testing.T) {
		mockDB := NewMockDatabase()
		expectedErr := fmt.Errorf("connection lost")
		mockDB.MockPingError(expectedErr)

		err := mockDB.Ping(context.Background())

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockDB.AssertExpectations(t)
	})
}

// TestMockDatabaseClose tests the Close method on MockDatabase
func TestMockDatabaseClose(t *testing.T) {
	mockDB := NewMockDatabase()
	mockDB.MockClose()

	mockDB.Close()

	mockDB.AssertExpectations(t)
}

// TestMockRowScan tests the MockRow Scan functionality
func TestMockRowScan(t *testing.T) {
	t.Run("scan string values", func(t *testing.T) {
		row := NewMockRowWithValues("value1", "value2", "value3")

		var v1, v2, v3 string
		err := row.Scan(&v1, &v2, &v3)

		assert.NoError(t, err)
		assert.Equal(t, "value1", v1)
		assert.Equal(t, "value2", v2)
		assert.Equal(t, "value3", v3)
	})

	t.Run("scan mixed types", func(t *testing.T) {
		row := NewMockRowWithValues("text", 42, true, int64(99), 3.14)

		var s string
		var i int
		var b bool
		var i64 int64
		var f float64
		err := row.Scan(&s, &i, &b, &i64, &f)

		assert.NoError(t, err)
		assert.Equal(t, "text", s)
		assert.Equal(t, 42, i)
		assert.True(t, b)
		assert.Equal(t, int64(99), i64)
		assert.Equal(t, 3.14, f)
	})

	t.Run("scan with error", func(t *testing.T) {
		expectedErr := fmt.Errorf("scan error")
		row := NewMockRowWithError(expectedErr)

		var v string
		err := row.Scan(&v)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("scan mismatched destination count", func(t *testing.T) {
		row := NewMockRowWithValues("value1", "value2")

		var v1 string
		err := row.Scan(&v1) // Too few destinations

		assert.Error(t, err)
		assert.Equal(t, pgx.ErrNoRows, err)
	})

	t.Run("scan byte slice", func(t *testing.T) {
		data := []byte("binary data")
		row := NewMockRowWithValues(data)

		var result []byte
		err := row.Scan(&result)

		assert.NoError(t, err)
		assert.Equal(t, data, result)
	})
}

// TestDatabaseInterfaceImplementation verifies Database implements DatabaseInterface
func TestDatabaseInterfaceImplementation(t *testing.T) {
	// This test will fail to compile if Database doesn't implement DatabaseInterface
	var _ DatabaseInterface = (*Database)(nil)
	var _ DatabaseInterface = (*MockDatabase)(nil)

	// If we get here, the interface is properly implemented
	assert.True(t, true, "Database implements DatabaseInterface")
}

// TestMockDatabaseChaining tests that helper methods return mock.Call for chaining
func TestMockDatabaseChaining(t *testing.T) {
	mockDB := NewMockDatabase()

	// Test that methods return *mock.Call for chaining
	call := mockDB.MockExecSuccess(1)
	assert.NotNil(t, call, "MockExecSuccess should return *mock.Call")

	call = mockDB.MockExecError(fmt.Errorf("error"))
	assert.NotNil(t, call, "MockExecError should return *mock.Call")

	call = mockDB.MockPingSuccess()
	assert.NotNil(t, call, "MockPingSuccess should return *mock.Call")

	call = mockDB.MockPingError(fmt.Errorf("error"))
	assert.NotNil(t, call, "MockPingError should return *mock.Call")
}
