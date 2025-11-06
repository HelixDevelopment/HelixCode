package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCountLines tests the countLines utility function
func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "Single line no newline",
			input:    "hello",
			expected: 1,
		},
		{
			name:     "Single line with newline",
			input:    "hello\n",
			expected: 2,
		},
		{
			name:     "Multiple lines",
			input:    "line1\nline2\nline3",
			expected: 3,
		},
		{
			name:     "Multiple lines with trailing newline",
			input:    "line1\nline2\nline3\n",
			expected: 4,
		},
		{
			name:     "Only newlines",
			input:    "\n\n\n",
			expected: 4,
		},
		{
			name:     "Mixed content",
			input:    "func main() {\n\tfmt.Println(\"Hello\")\n}\n",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
