package responses

import (
	"encoding/json"
	"os"
	"strings"
)

// Fixtures holds the loaded response fixtures
type Fixtures struct {
	DefaultCompletion string            `json:"default_completion"`
	Patterns          map[string]string `json:"patterns"`
	Embeddings        struct {
		Dimension int    `json:"dimension"`
		Model     string `json:"model"`
	} `json:"embeddings"`
	Models []Model `json:"models"`
}

// Model represents a mock model
type Model struct {
	ID           string   `json:"id"`
	Object       string   `json:"object"`
	Created      int64    `json:"created"`
	OwnedBy      string   `json:"owned_by"`
	Capabilities []string `json:"capabilities"`
}

// LoadFixtures loads response fixtures from JSON file
func LoadFixtures(path string) (*Fixtures, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var fixtures Fixtures
	if err := json.Unmarshal(data, &fixtures); err != nil {
		return nil, err
	}

	return &fixtures, nil
}

// FindResponse finds the best response for a given prompt
func (f *Fixtures) FindResponse(prompt string) string {
	promptLower := strings.ToLower(prompt)

	// Check for pattern matches
	for pattern, response := range f.Patterns {
		if strings.Contains(promptLower, pattern) {
			return response
		}
	}

	// Return default if no pattern matches
	return f.DefaultCompletion
}

// GetModels returns all available models
func (f *Fixtures) GetModels() []Model {
	return f.Models
}

// GetModel returns a specific model by ID
func (f *Fixtures) GetModel(id string) *Model {
	for i := range f.Models {
		if f.Models[i].ID == id {
			return &f.Models[i]
		}
	}
	return nil
}
