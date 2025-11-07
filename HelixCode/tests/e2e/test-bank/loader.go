package testbank

import (
	"encoding/json"
	"fmt"
	"os"

	"dev.helix.code/tests/e2e/orchestrator/pkg"
	"dev.helix.code/tests/e2e/test-bank/core"
)

// TestMetadata represents the metadata for a test case
type TestMetadata struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Priority          string   `json:"priority"`
	Tags              []string `json:"tags"`
	EstimatedDuration string   `json:"estimated_duration"`
	Dependencies      []string `json:"dependencies"`
	Timeout           string   `json:"timeout"`
	RetryCount        int      `json:"retry_count"`
	Platforms         []string `json:"platforms"`
	Description       string   `json:"description"`
	Preconditions     []string `json:"preconditions"`
	Steps             []string `json:"steps"`
	ExpectedResults   []string `json:"expected_results"`
}

// TestBank manages all test cases and their metadata
type TestBank struct {
	metadata map[string]*TestMetadata
	tests    map[string]*pkg.TestCase
}

// NewTestBank creates a new test bank instance
func NewTestBank() *TestBank {
	return &TestBank{
		metadata: make(map[string]*TestMetadata),
		tests:    make(map[string]*pkg.TestCase),
	}
}

// LoadMetadata loads test metadata from JSON files
func (tb *TestBank) LoadMetadata(metadataFile string) error {
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadataList []*TestMetadata
	if err := json.Unmarshal(data, &metadataList); err != nil {
		return fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	for _, meta := range metadataList {
		tb.metadata[meta.ID] = meta
	}

	return nil
}

// LoadTests loads all test cases from different categories
func (tb *TestBank) LoadTests() error {
	// Load core tests
	coreTests := core.GetCoreTests()
	for _, test := range coreTests {
		tb.tests[test.ID] = test
	}

	// TODO: Load integration tests
	// TODO: Load distributed tests
	// TODO: Load platform tests

	return nil
}

// GetAllTests returns all loaded test cases
func (tb *TestBank) GetAllTests() []*pkg.TestCase {
	tests := make([]*pkg.TestCase, 0, len(tb.tests))
	for _, test := range tb.tests {
		tests = append(tests, test)
	}
	return tests
}

// GetTestsByCategory returns tests filtered by category
func (tb *TestBank) GetTestsByCategory(category string) []*pkg.TestCase {
	tests := make([]*pkg.TestCase, 0)
	for _, test := range tb.tests {
		// Check if test has the category tag
		for _, tag := range test.Tags {
			if tag == category {
				tests = append(tests, test)
				break
			}
		}
	}
	return tests
}

// GetTestsByTags returns tests that match all given tags
func (tb *TestBank) GetTestsByTags(tags []string) []*pkg.TestCase {
	tests := make([]*pkg.TestCase, 0)
	for _, test := range tb.tests {
		if tb.hasAllTags(test, tags) {
			tests = append(tests, test)
		}
	}
	return tests
}

// GetTestByID returns a specific test by ID
func (tb *TestBank) GetTestByID(id string) (*pkg.TestCase, bool) {
	test, found := tb.tests[id]
	return test, found
}

// GetMetadata returns metadata for a specific test
func (tb *TestBank) GetMetadata(id string) (*TestMetadata, bool) {
	meta, found := tb.metadata[id]
	return meta, found
}

// hasAllTags checks if a test has all the specified tags
func (tb *TestBank) hasAllTags(test *pkg.TestCase, tags []string) bool {
	testTagSet := make(map[string]bool)
	for _, tag := range test.Tags {
		testTagSet[tag] = true
	}

	for _, tag := range tags {
		if !testTagSet[tag] {
			return false
		}
	}

	return true
}

// GetTestSuite creates a test suite with all tests
func (tb *TestBank) GetTestSuite() *pkg.TestSuite {
	return &pkg.TestSuite{
		Name:        "HelixCode E2E Test Suite",
		Description: "Comprehensive end-to-end test suite for the HelixCode platform",
		Tests:       tb.GetAllTests(),
	}
}

// GetTestSuiteByCategory creates a test suite for a specific category
func (tb *TestBank) GetTestSuiteByCategory(category string) *pkg.TestSuite {
	return &pkg.TestSuite{
		Name:        fmt.Sprintf("HelixCode %s Tests", category),
		Description: fmt.Sprintf("%s test suite for the HelixCode platform", category),
		Tests:       tb.GetTestsByCategory(category),
	}
}
