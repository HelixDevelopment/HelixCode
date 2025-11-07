package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHarmonyApp(t *testing.T) {
	app := NewHarmonyApp()

	require.NotNil(t, app, "HarmonyApp should not be nil")
	assert.NotNil(t, app.fyneApp, "Fyne app should be initialized")
	assert.NotNil(t, app.themeManager, "Theme manager should be initialized")
}

func TestHarmonyIntegration(t *testing.T) {
	app := NewHarmonyApp()
	app.initializeHarmonyComponents()

	require.NotNil(t, app.harmonyIntegration, "Harmony integration should be initialized")
	assert.NotNil(t, app.harmonyIntegration.systemAPI, "System API should be initialized")
	assert.NotNil(t, app.harmonyIntegration.distributedEngine, "Distributed engine should be initialized")

	// Test system API
	assert.Equal(t, "HarmonyOS 4.0", app.harmonyIntegration.systemAPI.systemVersion)
	assert.Equal(t, "Harmony", app.harmonyIntegration.systemAPI.deviceInfo["ecosystem"])
	assert.Contains(t, app.harmonyIntegration.systemAPI.capabilities, "distributed_computing")
	assert.Contains(t, app.harmonyIntegration.systemAPI.capabilities, "ai_acceleration")
}

func TestHarmonyDistributedEngine(t *testing.T) {
	app := NewHarmonyApp()
	app.initializeHarmonyComponents()

	engine := app.harmonyIntegration.distributedEngine

	require.NotNil(t, engine, "Distributed engine should not be nil")
	assert.NotNil(t, engine.taskScheduler, "Task scheduler should be initialized")
	assert.NotNil(t, engine.dataSync, "Data sync should be initialized")

	// Test task scheduler
	assert.Equal(t, "balanced", engine.taskScheduler.schedulingPolicy)
	assert.Equal(t, 4, engine.taskScheduler.priorityLevels["critical"])
	assert.Equal(t, 1, engine.taskScheduler.priorityLevels["low"])

	// Test data sync
	assert.True(t, engine.dataSync.syncEnabled, "Data sync should be enabled")
	assert.Equal(t, 30*time.Second, engine.dataSync.syncInterval)
}

func TestHarmonySystemMonitor(t *testing.T) {
	app := NewHarmonyApp()
	app.initializeHarmonyComponents()

	monitor := app.systemMonitor

	require.NotNil(t, monitor, "System monitor should not be nil")
	assert.True(t, monitor.monitoring, "Monitoring should be enabled")
	assert.Equal(t, 5*time.Second, monitor.updateInterval)

	// Test metrics update
	app.updateSystemMetrics()

	assert.Greater(t, monitor.cpuUsage, 0.0, "CPU usage should be set")
	assert.Greater(t, monitor.memoryUsage, 0.0, "Memory usage should be set")
	assert.GreaterOrEqual(t, monitor.gpuUsage, 0.0, "GPU usage should be set")
	assert.Greater(t, monitor.temperature, 0.0, "Temperature should be set")
	assert.Greater(t, monitor.powerUsage, 0.0, "Power usage should be set")
}

func TestHarmonyResourceManager(t *testing.T) {
	app := NewHarmonyApp()
	app.initializeHarmonyComponents()

	resourceMgr := app.resourceManager

	require.NotNil(t, resourceMgr, "Resource manager should not be nil")
	assert.True(t, resourceMgr.optimization, "Optimization should be enabled")
	assert.True(t, resourceMgr.autoTuning, "Auto-tuning should be enabled")

	// Test resource policies
	assert.Equal(t, "balanced", resourceMgr.resourcePolicies["cpu"])
	assert.Equal(t, "optimized", resourceMgr.resourcePolicies["memory"])
	assert.Equal(t, "efficient", resourceMgr.resourcePolicies["power"])
}

func TestHarmonyServiceCoordinator(t *testing.T) {
	app := NewHarmonyApp()
	app.initializeHarmonyComponents()

	coordinator := app.serviceCoordinator

	require.NotNil(t, coordinator, "Service coordinator should not be nil")
	assert.NotNil(t, coordinator.services, "Services map should be initialized")
	assert.NotNil(t, coordinator.serviceRegistry, "Service registry should be initialized")
	assert.NotNil(t, coordinator.coordinator, "Coordinator should be initialized")
	assert.True(t, coordinator.coordinator.failoverEnabled, "Failover should be enabled")
}

func TestThemeManager(t *testing.T) {
	tm := NewThemeManager()

	require.NotNil(t, tm, "Theme manager should not be nil")
	assert.Equal(t, "Harmony", tm.currentTheme, "Default theme should be Harmony")

	// Test theme switching
	tm.SetTheme("Dark")
	assert.Equal(t, "Dark", tm.currentTheme)

	tm.SetTheme("Light")
	assert.Equal(t, "Light", tm.currentTheme)

	tm.SetTheme("Helix")
	assert.Equal(t, "Helix", tm.currentTheme)

	tm.SetTheme("Harmony")
	assert.Equal(t, "Harmony", tm.currentTheme)

	// Test getting available themes
	themes := tm.GetAvailableThemes()
	assert.Contains(t, themes, "Dark")
	assert.Contains(t, themes, "Light")
	assert.Contains(t, themes, "Helix")
	assert.Contains(t, themes, "Harmony")
}

func TestHarmonyTheme(t *testing.T) {
	assert.Equal(t, "Harmony", HarmonyTheme.Name)
	assert.True(t, HarmonyTheme.IsDark)
	assert.Equal(t, "#FF6B35", HarmonyTheme.Primary)
	assert.Equal(t, "#F7931E", HarmonyTheme.Secondary)
	assert.Equal(t, "#FDB462", HarmonyTheme.Accent)
	assert.Equal(t, "#FFFFFF", HarmonyTheme.Text)
	assert.Equal(t, "#1A1512", HarmonyTheme.Background)
}

func TestCustomTheme(t *testing.T) {
	ct := NewCustomTheme(&HarmonyTheme)

	require.NotNil(t, ct, "Custom theme should not be nil")
	assert.NotNil(t, ct.currentTheme, "Current theme should be set")
	assert.Equal(t, "Harmony", ct.currentTheme.Name)
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name     string
		hexColor string
		wantR    uint8
		wantG    uint8
		wantB    uint8
		wantA    uint8
	}{
		{
			name:     "Valid hex with #",
			hexColor: "#FF6B35",
			wantR:    255,
			wantG:    107,
			wantB:    53,
			wantA:    255,
		},
		{
			name:     "Valid hex without #",
			hexColor: "F7931E",
			wantR:    247,
			wantG:    147,
			wantB:    30,
			wantA:    255,
		},
		{
			name:     "White color",
			hexColor: "#FFFFFF",
			wantR:    255,
			wantG:    255,
			wantB:    255,
			wantA:    255,
		},
		{
			name:     "Black color",
			hexColor: "#000000",
			wantR:    0,
			wantG:    0,
			wantB:    0,
			wantA:    255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := parseHexColor(tt.hexColor)
			r, g, b, a := c.RGBA()

			// Convert from 16-bit to 8-bit
			assert.Equal(t, tt.wantR, uint8(r>>8), "Red component mismatch")
			assert.Equal(t, tt.wantG, uint8(g>>8), "Green component mismatch")
			assert.Equal(t, tt.wantB, uint8(b>>8), "Blue component mismatch")
			assert.Equal(t, tt.wantA, uint8(a>>8), "Alpha component mismatch")
		})
	}
}

func TestAddRemoveTheme(t *testing.T) {
	tm := NewThemeManager()

	// Add custom theme
	customTheme := &Theme{
		Name:       "Custom",
		IsDark:     true,
		Primary:    "#FF0000",
		Secondary:  "#00FF00",
		Accent:     "#0000FF",
		Text:       "#FFFFFF",
		Background: "#000000",
		Border:     "#808080",
		Success:    "#00FF00",
		Warning:    "#FFFF00",
		Error:      "#FF0000",
		Info:       "#0000FF",
	}

	tm.AddTheme("Custom", customTheme)
	themes := tm.GetAvailableThemes()
	assert.Contains(t, themes, "Custom")

	// Test switching to custom theme
	tm.SetTheme("Custom")
	assert.Equal(t, "Custom", tm.currentTheme)

	// Remove custom theme
	tm.RemoveTheme("Custom")
	themes = tm.GetAvailableThemes()
	assert.NotContains(t, themes, "Custom")

	// Test that built-in themes cannot be removed
	tm.RemoveTheme("Harmony")
	themes = tm.GetAvailableThemes()
	assert.Contains(t, themes, "Harmony")
}

func TestCleanup(t *testing.T) {
	app := NewHarmonyApp()
	app.initializeHarmonyComponents()

	// Cleanup should not panic
	assert.NotPanics(t, func() {
		app.Cleanup()
	})

	// Verify monitoring is stopped
	assert.False(t, app.systemMonitor.monitoring)
}
