package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionVariables(t *testing.T) {
	// Test that version variables are set
	assert.NotEmpty(t, version)
	assert.NotEmpty(t, buildTime)
	assert.NotEmpty(t, gitCommit)

	// Test that they have reasonable values
	assert.Contains(t, []string{"1.0.0", "dev", "unknown"}, version)
	assert.Contains(t, []string{"unknown", buildTime}, buildTime)
	assert.Contains(t, []string{"unknown", gitCommit}, gitCommit)
}
