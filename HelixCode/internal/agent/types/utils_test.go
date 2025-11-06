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
			input:    "hello world",
			expected: 1,
		},
		{
			name:     "Single line with newline",
			input:    "hello world\n",
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
			name:     "Empty lines counted",
			input:    "line1\n\nline3",
			expected: 3,
		},
		{
			name:     "Only newlines",
			input:    "\n\n\n",
			expected: 4,
		},
		{
			name:     "Code with multiple newlines",
			input:    "func main() {\n\tprintln(\"hello\")\n}\n",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.input)
			assert.Equal(t, tt.expected, result, "Line count mismatch for input: %q", tt.input)
		})
	}
}

// TestCountLinesEdgeCases tests edge cases for countLines
func TestCountLinesEdgeCases(t *testing.T) {
	t.Run("Very long single line", func(t *testing.T) {
		longLine := ""
		for i := 0; i < 10000; i++ {
			longLine += "a"
		}
		assert.Equal(t, 1, countLines(longLine))
	})

	t.Run("Many short lines", func(t *testing.T) {
		manyLines := ""
		for i := 0; i < 1000; i++ {
			manyLines += "line\n"
		}
		assert.Equal(t, 1001, countLines(manyLines))
	})

	t.Run("Mixed line endings simulation", func(t *testing.T) {
		// Only testing \n since that's what the function checks for
		mixed := "line1\nline2\nline3\n"
		assert.Equal(t, 4, countLines(mixed))
	})

	t.Run("Unicode content", func(t *testing.T) {
		unicode := "Hello 世界\n你好\nПривет\n"
		assert.Equal(t, 4, countLines(unicode))
	})
}

// TestCountLinesWithCode tests countLines with actual code samples
func TestCountLinesWithCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "Simple Go function",
			code: `func add(a, b int) int {
	return a + b
}`,
			expected: 3,
		},
		{
			name: "Go struct",
			code: `type Person struct {
	Name string
	Age  int
}`,
			expected: 4,
		},
		{
			name: "Go function with comments",
			code: `// Add returns the sum of two integers
func add(a, b int) int {
	return a + b
}`,
			expected: 4,
		},
		{
			name: "Multi-line string",
			code: `func main() {
	msg := "line1
line2
line3"
	println(msg)
}`,
			expected: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}
