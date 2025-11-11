package database

import (
	"reflect"

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
func NewMockRowWithValues(values ...interface{}) *MockRow {
	return &MockRow{
		values: values,
		err:    nil,
		index:  0,
	}
}

// NewMockRowWithError creates a MockRow that returns an error when Scan() is called.
func NewMockRowWithError(err error) *MockRow {
	return &MockRow{
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

	for i, val := range m.values {
		if val == nil {
			// Skip nil values - leave destination at zero value
			continue
		}

		d := dest[i]
		switch v := d.(type) {
		case *string:
			if s, ok := val.(string); ok {
				*v = s
			}
		case **string:
			if s, ok := val.(string); ok {
				str := s
				*v = &str
			}
		case *int:
			if i, ok := val.(int); ok {
				*v = i
			}
		case *int64:
			if i, ok := val.(int64); ok {
				*v = i
			}
		case *bool:
			if b, ok := val.(bool); ok {
				*v = b
			}
		case *float64:
			if f, ok := val.(float64); ok {
				*v = f
			}
		case *[]byte:
			if b, ok := val.([]byte); ok {
				*v = b
			}
		default:
			// For complex types (UUID, maps, slices, time.Time, etc.),
			// use reflection to assign the value
			destValue := reflect.ValueOf(d)
			if destValue.Kind() == reflect.Ptr {
				srcValue := reflect.ValueOf(val)
				// Check if types match
				if destValue.Elem().Type() == srcValue.Type() {
					destValue.Elem().Set(srcValue)
				}
			}
		}
	}

	return nil
}

// Ensure MockRow implements pgx.Row at compile time.
var _ pgx.Row = (*MockRow)(nil)
