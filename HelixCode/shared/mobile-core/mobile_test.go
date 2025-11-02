package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMobileCore(t *testing.T) {
	core := NewMobileCore()
	assert.NotNil(t, core)
	assert.NotNil(t, core.themeManager)
}

func TestThemeManager_GetAvailableThemes(t *testing.T) {
	tm := NewThemeManager()
	themes := tm.GetAvailableThemes()
	assert.Contains(t, themes, "dark")
	assert.Contains(t, themes, "light")
	assert.Contains(t, themes, "helix")
}

func TestThemeManager_SetTheme(t *testing.T) {
	tm := NewThemeManager()

	// Test valid theme
	assert.True(t, tm.SetTheme("light"))
	assert.Equal(t, "Light", tm.GetCurrentTheme().Name)

	// Test invalid theme
	assert.False(t, tm.SetTheme("invalid"))
}

func TestMobileCore_GetDashboardData(t *testing.T) {
	core := NewMobileCore()
	data := core.GetDashboardData()
	assert.NotEmpty(t, data)
	assert.Contains(t, data, "isConnected")
}

func TestMobileCore_GetTasks(t *testing.T) {
	core := NewMobileCore()
	tasks := core.GetTasks()
	assert.NotEmpty(t, tasks)
	// Should contain tasks array
	assert.Contains(t, tasks, `"tasks":`)
}

func TestMobileCore_GetWorkers(t *testing.T) {
	core := NewMobileCore()
	workers := core.GetWorkers()
	assert.NotEmpty(t, workers)
	// Should contain workers array
	assert.Contains(t, workers, `"workers":`)
}

func TestMobileCore_CreateTask(t *testing.T) {
	core := NewMobileCore()
	result := core.CreateTask("Test Task", "Test Description")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "success")
}

func TestMobileCore_GetTheme(t *testing.T) {
	core := NewMobileCore()
	theme := core.GetTheme()
	assert.NotEmpty(t, theme)
	assert.Contains(t, theme, "name")
}

func TestMobileCore_SetTheme(t *testing.T) {
	core := NewMobileCore()

	// Test valid theme
	assert.True(t, core.SetTheme("dark"))

	// Test invalid theme
	assert.False(t, core.SetTheme("invalid"))
}

func TestMobileCore_GetAvailableThemes(t *testing.T) {
	core := NewMobileCore()
	themes := core.GetAvailableThemes()
	assert.NotEmpty(t, themes)
	assert.Contains(t, themes, "dark")
	assert.Contains(t, themes, "light")
	assert.Contains(t, themes, "helix")
}
