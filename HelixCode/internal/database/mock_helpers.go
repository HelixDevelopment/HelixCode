package database

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
)

// MockExecSuccess sets up a successful Exec expectation.
// Returns a CommandTag indicating the number of rows affected.
//
// Usage:
//
//	mockDB := NewMockDatabase()
//	mockDB.MockExecSuccess(1) // 1 row affected
//	err := someFunc(mockDB)
func (m *MockDatabase) MockExecSuccess(rowsAffected int64) *mock.Call {
	tag := pgconn.NewCommandTag(fmt.Sprintf("INSERT 0 %d", rowsAffected))
	return m.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(tag, nil)
}

// MockExecSuccessOnce sets up a single successful Exec expectation.
// Only applies to the next call, then removes the expectation.
func (m *MockDatabase) MockExecSuccessOnce(rowsAffected int64) *mock.Call {
	tag := pgconn.NewCommandTag(fmt.Sprintf("INSERT 0 %d", rowsAffected))
	return m.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(tag, nil).Once()
}

// MockExecError sets up a failed Exec expectation.
// Returns an error without affecting any rows.
//
// Usage:
//
//	mockDB := NewMockDatabase()
//	mockDB.MockExecError(fmt.Errorf("connection lost"))
func (m *MockDatabase) MockExecError(err error) *mock.Call {
	tag := pgconn.CommandTag{}
	return m.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(tag, err)
}

// MockExecErrorOnce sets up a single failed Exec expectation.
func (m *MockDatabase) MockExecErrorOnce(err error) *mock.Call {
	tag := pgconn.CommandTag{}
	return m.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(tag, err).Once()
}

// MockQueryRowSuccess sets up a successful QueryRow expectation.
// The provided row will be returned when QueryRow is called.
//
// Note: You'll need to create a mock row implementation for your specific use case.
// See MockRow in mock_row.go for a simple implementation.
func (m *MockDatabase) MockQueryRowSuccess(row MockRow) *mock.Call {
	return m.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(&row)
}

// MockQueryRowError sets up a failed QueryRow expectation.
// The provided error will be returned when the row is scanned.
func (m *MockDatabase) MockQueryRowError(err error) *mock.Call {
	row := NewMockRowWithError(err)
	return m.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(&row)
}

// MockPingSuccess sets up a successful Ping expectation.
func (m *MockDatabase) MockPingSuccess() *mock.Call {
	return m.On("Ping", mock.Anything).Return(nil)
}

// MockPingError sets up a failed Ping expectation.
func (m *MockDatabase) MockPingError(err error) *mock.Call {
	return m.On("Ping", mock.Anything).Return(err)
}

// MockClose sets up a Close expectation.
func (m *MockDatabase) MockClose() *mock.Call {
	return m.On("Close").Return()
}

// MockExecWithSQL sets up an Exec expectation that matches specific SQL.
// This is useful when you want to verify the exact SQL being executed.
func (m *MockDatabase) MockExecWithSQL(sql string, rowsAffected int64) *mock.Call {
	tag := pgconn.NewCommandTag(fmt.Sprintf("INSERT 0 %d", rowsAffected))
	return m.On("Exec", mock.Anything, sql, mock.Anything).Return(tag, nil)
}

// MockExecWithSQLAndArgs sets up an Exec expectation matching SQL and arguments.
func (m *MockDatabase) MockExecWithSQLAndArgs(sql string, args []interface{}, rowsAffected int64) *mock.Call {
	tag := pgconn.NewCommandTag(fmt.Sprintf("INSERT 0 %d", rowsAffected))
	return m.On("Exec", mock.Anything, sql, args).Return(tag, nil)
}
