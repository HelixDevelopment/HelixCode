package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTerminalUI(t *testing.T) {
	tui := NewTerminalUI()
	assert.NotNil(t, tui)
	assert.NotNil(t, tui.app)
}

func TestNewThemeManager(t *testing.T) {
	tm := NewThemeManager()
	assert.NotNil(t, tm)
	assert.NotNil(t, tm.currentTheme)
	assert.NotEmpty(t, tm.themes)
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

func TestThemeManager_GetColor(t *testing.T) {
	tm := NewThemeManager()

	// Test valid color types
	assert.NotEmpty(t, tm.GetColor("primary"))
	assert.NotEmpty(t, tm.GetColor("text"))
	assert.NotEmpty(t, tm.GetColor("background"))

	// Test invalid color type (should return text color)
	assert.NotEmpty(t, tm.GetColor("invalid"))
}

func TestNewUIComponents(t *testing.T) {
	tui := NewTerminalUI()
	components := NewUIComponents(tui)
	assert.NotNil(t, components)
	assert.Equal(t, tui, components.tui)
}

func TestUIComponents_CreateForm(t *testing.T) {
	tui := NewTerminalUI()
	components := NewUIComponents(tui)

	fields := []FormField{
		{Type: "text", Label: "Name", DefaultValue: "test"},
	}

	form := components.CreateForm("Test Form", fields)
	assert.NotNil(t, form)
}

func TestUIComponents_CreateList(t *testing.T) {
	tui := NewTerminalUI()
	components := NewUIComponents(tui)

	items := []ListItem{
		{MainText: "Item 1", SecondaryText: "Description 1"},
		{MainText: "Item 2", SecondaryText: "Description 2"},
	}

	list := components.CreateList("Test List", items)
	assert.NotNil(t, list)
}

func TestUIComponents_CreateTable(t *testing.T) {
	tui := NewTerminalUI()
	components := NewUIComponents(tui)

	headers := []string{"Col1", "Col2"}
	data := [][]string{
		{"Row1Col1", "Row1Col2"},
		{"Row2Col1", "Row2Col2"},
	}

	table := components.CreateTable("Test Table", headers, data)
	assert.NotNil(t, table)
}

func TestUIComponents_CreateProgressBar(t *testing.T) {
	tui := NewTerminalUI()
	components := NewUIComponents(tui)

	progress := components.CreateProgressBar("Test Progress", 50, 100)
	assert.NotNil(t, progress)
}

func TestUIComponents_CreateModal(t *testing.T) {
	tui := NewTerminalUI()
	components := NewUIComponents(tui)

	buttons := []ModalButton{
		{Label: "OK", OnClick: func() {}},
		{Label: "Cancel", OnClick: func() {}},
	}

	modal := components.CreateModal("Test Modal", "Test message", buttons)
	assert.NotNil(t, modal)
}

func TestUIComponents_CreateStatusBar(t *testing.T) {
	tui := NewTerminalUI()
	components := NewUIComponents(tui)

	status := components.CreateStatusBar()
	assert.NotNil(t, status)
}

func TestUIComponents_CreateLogView(t *testing.T) {
	tui := NewTerminalUI()
	components := NewUIComponents(tui)

	logView := components.CreateLogView("Test Log")
	assert.NotNil(t, logView)
}
