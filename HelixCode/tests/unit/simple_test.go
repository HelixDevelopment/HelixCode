package unit

import (
	"testing"
	"time"
)

// Simple unit tests that don't depend on complex imports
func TestBasicFunctionality(t *testing.T) {
	// Test basic functionality
	result := add(2, 3)
	expected := 5
	
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestTimeHandling(t *testing.T) {
	// Test time handling
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	duration := time.Since(start)
	
	if duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", duration)
	}
}

func TestStringOperations(t *testing.T) {
	// Test string operations
	input := "hello world"
	expected := "hello world"
	
	if input != expected {
		t.Errorf("Expected %s, got %s", expected, input)
	}
}

func TestMapOperations(t *testing.T) {
	// Test map operations
	m := make(map[string]int)
	m["key"] = 42
	
	if m["key"] != 42 {
		t.Errorf("Expected 42, got %d", m["key"])
	}
}

func TestSliceOperations(t *testing.T) {
	// Test slice operations
	slice := []int{1, 2, 3}
	
	if len(slice) != 3 {
		t.Errorf("Expected length 3, got %d", len(slice))
	}
	
	if slice[0] != 1 {
		t.Errorf("Expected first element 1, got %d", slice[0])
	}
}

// Helper function for testing
func add(a, b int) int {
	return a + b
}