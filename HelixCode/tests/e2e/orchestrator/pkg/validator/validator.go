package validator

import (
	"fmt"
	"reflect"

	"dev.helix.code/tests/e2e/orchestrator/pkg"
)

// Validator provides assertion helpers for tests
type Validator struct {
	assertions []pkg.AssertionResult
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		assertions: make([]pkg.AssertionResult, 0),
	}
}

// GetAssertions returns all recorded assertions
func (v *Validator) GetAssertions() []pkg.AssertionResult {
	return v.assertions
}

// Assert evaluates a condition and records the result
func (v *Validator) Assert(condition bool, description string) error {
	result := pkg.AssertionResult{
		Description: description,
		Passed:      condition,
	}

	if !condition {
		result.Message = "Assertion failed"
	}

	v.assertions = append(v.assertions, result)

	if !condition {
		return fmt.Errorf("assertion failed: %s", description)
	}

	return nil
}

// AssertEqual checks if two values are equal
func (v *Validator) AssertEqual(expected, actual interface{}, description string) error {
	equal := reflect.DeepEqual(expected, actual)

	result := pkg.AssertionResult{
		Description: description,
		Passed:      equal,
		Expected:    fmt.Sprintf("%v", expected),
		Actual:      fmt.Sprintf("%v", actual),
	}

	if !equal {
		result.Message = fmt.Sprintf("Expected %v, got %v", expected, actual)
	}

	v.assertions = append(v.assertions, result)

	if !equal {
		return fmt.Errorf("%s: expected %v, got %v", description, expected, actual)
	}

	return nil
}

// AssertNotEqual checks if two values are not equal
func (v *Validator) AssertNotEqual(expected, actual interface{}, description string) error {
	equal := reflect.DeepEqual(expected, actual)

	result := pkg.AssertionResult{
		Description: description,
		Passed:      !equal,
		Expected:    fmt.Sprintf("not %v", expected),
		Actual:      fmt.Sprintf("%v", actual),
	}

	if equal {
		result.Message = fmt.Sprintf("Expected not %v, but got %v", expected, actual)
	}

	v.assertions = append(v.assertions, result)

	if equal {
		return fmt.Errorf("%s: expected not %v, but got %v", description, expected, actual)
	}

	return nil
}

// AssertNil checks if value is nil
func (v *Validator) AssertNil(actual interface{}, description string) error {
	isNil := actual == nil || (reflect.ValueOf(actual).Kind() == reflect.Ptr && reflect.ValueOf(actual).IsNil())

	result := pkg.AssertionResult{
		Description: description,
		Passed:      isNil,
		Expected:    "nil",
		Actual:      fmt.Sprintf("%v", actual),
	}

	if !isNil {
		result.Message = fmt.Sprintf("Expected nil, got %v", actual)
	}

	v.assertions = append(v.assertions, result)

	if !isNil {
		return fmt.Errorf("%s: expected nil, got %v", description, actual)
	}

	return nil
}

// AssertNotNil checks if value is not nil
func (v *Validator) AssertNotNil(actual interface{}, description string) error {
	isNil := actual == nil || (reflect.ValueOf(actual).Kind() == reflect.Ptr && reflect.ValueOf(actual).IsNil())

	result := pkg.AssertionResult{
		Description: description,
		Passed:      !isNil,
		Expected:    "not nil",
		Actual:      fmt.Sprintf("%v", actual),
	}

	if isNil {
		result.Message = "Expected not nil, got nil"
	}

	v.assertions = append(v.assertions, result)

	if isNil {
		return fmt.Errorf("%s: expected not nil, got nil", description)
	}

	return nil
}

// AssertTrue checks if condition is true
func (v *Validator) AssertTrue(condition bool, description string) error {
	return v.AssertEqual(true, condition, description)
}

// AssertFalse checks if condition is false
func (v *Validator) AssertFalse(condition bool, description string) error {
	return v.AssertEqual(false, condition, description)
}

// AssertContains checks if a string contains a substring
func (v *Validator) AssertContains(haystack, needle string, description string) error {
	contains := len(haystack) > 0 && len(needle) > 0
	if contains {
		for i := 0; i <= len(haystack)-len(needle); i++ {
			if haystack[i:i+len(needle)] == needle {
				return v.Assert(true, description)
			}
		}
	}
	return v.Assert(false, fmt.Sprintf("%s: '%s' does not contain '%s'", description, haystack, needle))
}
