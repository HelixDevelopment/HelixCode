package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuroraAppCreation(t *testing.T) {
	// Test that we can create the basic structs without panicking
	// Note: Full GUI testing would require a display environment

	aurora := &AuroraIntegration{
		nativeServices: make(map[string]interface{}),
	}

	assert.NotNil(t, aurora)
	assert.NotNil(t, aurora.nativeServices)
}

func TestAuroraSystemMonitor(t *testing.T) {
	monitor := &AuroraSystemMonitor{
		networkStats: make(map[string]interface{}),
	}

	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.networkStats)
}

func TestAuroraSecurityManager(t *testing.T) {
	security := &AuroraSecurityManager{
		accessControl: make(map[string][]string),
		auditLog:      make([]string, 0),
	}

	assert.NotNil(t, security)
	assert.NotNil(t, security.accessControl)
	assert.NotNil(t, security.auditLog)
}
