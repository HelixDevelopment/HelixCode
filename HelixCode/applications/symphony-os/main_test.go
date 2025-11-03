package main

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSymphonyPerformanceMonitor(t *testing.T) {
	monitor := &SymphonyPerformanceMonitor{
		gcStats:       &runtime.MemStats{},
		optimizations: make([]string, 0),
		lastUpdate:    time.Now(),
	}

	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.gcStats)
	assert.NotNil(t, monitor.optimizations)
	assert.True(t, monitor.lastUpdate.After(time.Now().Add(-time.Second)))
}

func TestSymphonyResourceOptimizer(t *testing.T) {
	optimizer := &SymphonyResourceOptimizer{
		gcThreshold:  100 * 1024 * 1024, // 100MB
		cacheSize:    50,
		workerPool:   runtime.NumCPU(),
		adaptiveMode: true,
	}

	assert.NotNil(t, optimizer)
	assert.Equal(t, uint64(100*1024*1024), optimizer.gcThreshold)
	assert.Equal(t, 50, optimizer.cacheSize)
	assert.Equal(t, runtime.NumCPU(), optimizer.workerPool)
	assert.True(t, optimizer.adaptiveMode)
}

func TestSymphonyAdaptiveUI(t *testing.T) {
	adaptiveUI := &SymphonyAdaptiveUI{
		screenDensity: 1.0,
		fontScale:     1.0,
		themeVariant:  "default",
		accessibility: false,
	}

	assert.NotNil(t, adaptiveUI)
	assert.Equal(t, float32(1.0), adaptiveUI.screenDensity)
	assert.Equal(t, float32(1.0), adaptiveUI.fontScale)
	assert.Equal(t, "default", adaptiveUI.themeVariant)
	assert.False(t, adaptiveUI.accessibility)
}
