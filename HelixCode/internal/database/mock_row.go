package database

import (
	"github.com/jackc/pgx/v5"
)

// MockRow is a simple implementation of pgx.Row for testing.
// It allows you to provide scan destinations or errors for QueryRow mocks.
//
// Usage Example:
//
//	// Create a mock row that will scan a value
//	row := database.NewMockRowWithValues("test-id", "test-name")
//	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(&row)
//
//	// Or create a mock row that returns an error
//	row := database.NewMockRowWithError(pgx.ErrNoRows)
//	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(&row)
type MockRow struct {
	values []interface{}
	err    error
	index  int
}

// NewMockRowWithValues creates a MockRow that will scan the provided values.
// Call Scan() on the row to populate your variables.
func NewMockRowWithValues(values ...interface{}) MockRow {
	return MockRow{
		values: values,
		err:    nil,
		index:  0,
	}
}

// NewMockRowWithError creates a MockRow that returns an error when Scan() is called.
func NewMockRowWithError(err error) MockRow {
	return MockRow{
		values: nil,
		err:    err,
		index:  0,
	}
}

// Scan implements pgx.Row.Scan().
// It populates the destination variables with the mocked values or returns the mocked error.
func (m *MockRow) Scan(dest ...interface{}) error {
	if m.err != nil {
		return m.err
	}

	if len(dest) != len(m.values) {
		return pgx.ErrNoRows // Return appropriate error if counts don't match
	}

	for i, d := range dest {
		switch v := d.(type) {
		case *string:
			*v = m.values[i].(string)
		case *int:
			*v = m.values[i].(int)
		case *int64:
			*v = m.values[i].(int64)
		case *bool:
			*v = m.values[i].(bool)
		case *float64:
			*v = m.values[i].(float64)
		case *[]byte:
			*v = m.values[i].([]byte)
		// Add more types as needed
		default:
			// For complex types, try direct assignment
			// This works for pointers to structs, etc.
			*v.(*interface{}) = m.values[i]
		}
	}

	return nil
}

// Ensure MockRow implements pgx.Row at compile time.
var _ pgx.Row = (*MockRow)(nil)
