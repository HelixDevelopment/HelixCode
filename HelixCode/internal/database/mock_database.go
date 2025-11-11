package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
)

// MockDatabase is a mock implementation of DatabaseInterface for testing.
// It uses testify/mock to provide a flexible mocking framework.
//
// Usage Example:
//
//	mockDB := database.NewMockDatabase()
//	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
//		Return(pgconn.NewCommandTag("INSERT 0 1"), nil)
//
//	// Use mockDB in your tests
//	err := someFunction(mockDB)
//
//	// Verify expectations
//	mockDB.AssertExpectations(t)
type MockDatabase struct {
	mock.Mock
}

// NewMockDatabase creates a new mock database instance.
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{}
}

// Exec mocks the Exec method.
// Use mock.On() to set expectations and return values.
func (m *MockDatabase) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

// Query mocks the Query method.
// Use mock.On() to set expectations and return values.
func (m *MockDatabase) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	args := m.Called(ctx, sql, arguments)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(pgx.Rows), args.Error(1)
}

// QueryRow mocks the QueryRow method.
// Use mock.On() to set expectations and return values.
func (m *MockDatabase) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgx.Row)
}

// Ping mocks the Ping method.
// Use mock.On() to set expectations and return values.
func (m *MockDatabase) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Close mocks the Close method.
// Use mock.On() to set expectations.
func (m *MockDatabase) Close() {
	m.Called()
}

// Ensure MockDatabase implements DatabaseInterface at compile time.
var _ DatabaseInterface = (*MockDatabase)(nil)
