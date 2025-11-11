package database

import (
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MockRows implements pgx.Rows for testing Query operations.
type MockRows struct {
	rows    [][]interface{}
	current int
	err     error
	closed  bool
}

// NewMockRows creates a new MockRows with the given data rows.
// Each row is a slice of interface{} representing the column values.
//
// Usage:
//
//	rows := NewMockRows([][]interface{}{
//	    {uuid.New(), "task1", "pending"},
//	    {uuid.New(), "task2", "running"},
//	})
func NewMockRows(rows [][]interface{}) *MockRows {
	return &MockRows{
		rows:    rows,
		current: -1,
		err:     nil,
		closed:  false,
	}
}

// NewMockRowsWithError creates a MockRows that will return an error.
func NewMockRowsWithError(err error) *MockRows {
	return &MockRows{
		rows:    nil,
		current: -1,
		err:     err,
		closed:  false,
	}
}

// Close closes the rows, making it unusable for any subsequent operations.
func (m *MockRows) Close() {
	m.closed = true
}

// Err returns any error that occurred during iteration.
func (m *MockRows) Err() error {
	return m.err
}

// CommandTag returns the command tag from the SQL command that created the rows.
// This is not applicable for SELECT queries, so we return an empty tag.
func (m *MockRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

// FieldDescriptions returns the field descriptions for the columns.
// For testing purposes, we return an empty slice.
func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription {
	return []pgconn.FieldDescription{}
}

// Next advances the rows to the next row.
// Returns true if there is another row and false if no more rows or an error occurred.
func (m *MockRows) Next() bool {
	if m.closed {
		return false
	}
	if m.err != nil {
		return false
	}
	m.current++
	return m.current < len(m.rows)
}

// Scan reads the values from the current row into dest.
func (m *MockRows) Scan(dest ...interface{}) error {
	if m.err != nil {
		return m.err
	}
	if m.current < 0 || m.current >= len(m.rows) {
		return pgx.ErrNoRows
	}

	row := m.rows[m.current]
	if len(dest) != len(row) {
		return pgx.ErrNoRows
	}

	for i, val := range row {
		if val == nil {
			// Skip nil values - leave destination at zero value
			continue
		}

		// Use reflection to handle all types
		destVal := dest[i]
		switch d := destVal.(type) {
		case *string:
			if v, ok := val.(string); ok {
				*d = v
			}
		case **string:
			if v, ok := val.(string); ok {
				s := v
				*d = &s
			}
		case *int:
			if v, ok := val.(int); ok {
				*d = v
			}
		case *int64:
			if v, ok := val.(int64); ok {
				*d = v
			}
		case *bool:
			if v, ok := val.(bool); ok {
				*d = v
			}
		default:
			// For complex types (UUID, maps, slices, time.Time, etc.),
			// use direct assignment via reflection
			destValue := reflect.ValueOf(destVal)
			if destValue.Kind() == reflect.Ptr {
				srcValue := reflect.ValueOf(val)
				if destValue.Elem().Type() == srcValue.Type() {
					destValue.Elem().Set(srcValue)
				}
			}
		}
	}

	return nil
}

// Values returns the decoded row values.
func (m *MockRows) Values() ([]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.current < 0 || m.current >= len(m.rows) {
		return nil, pgx.ErrNoRows
	}
	return m.rows[m.current], nil
}

// RawValues returns the unparsed bytes of the row values.
// For testing purposes, we return nil.
func (m *MockRows) RawValues() [][]byte {
	return nil
}

// Conn returns the underlying connection (not applicable for mocks).
func (m *MockRows) Conn() *pgx.Conn {
	return nil
}

// Compile-time check that MockRows implements pgx.Rows
var _ pgx.Rows = (*MockRows)(nil)
