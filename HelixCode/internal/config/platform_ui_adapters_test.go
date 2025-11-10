package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasePlatformAdapter tests base platform adapter functionality
func TestBasePlatformAdapter(t *testing.T) {
	adapter := NewBasePlatformAdapter("desktop")

	// Test platform type
	assert.Equal(t, "desktop", adapter.GetPlatformType())

	// Test features
	features := adapter.GetPlatformFeatures()
	assert.NotEmpty(t, features)
	assert.Contains(t, features, "native_menus")
	assert.Contains(t, features, "system_tray")
	assert.Contains(t, features, "file_dialogs")

	// Test themes
	themes := adapter.GetPlatformThemes()
	assert.NotEmpty(t, themes)
	assert.Contains(t, themes, "system")

	systemTheme := themes["system"]
	assert.Equal(t, "System", systemTheme.Name)
	assert.Equal(t, "Matches system appearance", systemTheme.Description)
}

// TestDesktopUIAdapter tests desktop UI adapter
func TestDesktopUIAdapter(t *testing.T) {
	adapter := NewDesktopUIAdapter()

	// Test platform type
	assert.Equal(t, "desktop", adapter.GetPlatformType())

	// Test features
	features := adapter.GetPlatformFeatures()
	expectedFeatures := []string{
		"native_menus",
		"system_tray",
		"file_dialogs",
		"native_fonts",
		"keyboard_shortcuts",
		"drag_drop",
		"context_menus",
		"system_notifications",
		"auto_update",
		"window_management",
	}

	for _, feature := range expectedFeatures {
		assert.Contains(t, features, feature)
	}

	// Test themes
	themes := adapter.GetPlatformThemes()
	assert.Contains(t, themes, "system")
	assert.Contains(t, themes, "dark")
	assert.Contains(t, themes, "light")

	// Test config form rendering
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.json"

	configUI, err := NewConfigUI(configPath)
	require.NoError(t, err)

	form, err := adapter.RenderConfigForm(configUI)
	require.NoError(t, err)

	// Verify form structure
	desktopForm, ok := form.(DesktopConfigForm)
	require.True(t, ok)

	assert.Equal(t, "helix_config_form", desktopForm.ID)
	assert.Equal(t, "HelixCode Configuration", desktopForm.Title)
	assert.Equal(t, "native_window", desktopForm.Type)
	assert.Equal(t, "tabs", desktopForm.Layout)
	assert.True(t, desktopForm.Modal)
	assert.True(t, desktopForm.Resizable)
	assert.Equal(t, 800, desktopForm.MinWidth)
	assert.Equal(t, 600, desktopForm.MinHeight)
	assert.Equal(t, 1200, desktopForm.DefaultWidth)
	assert.Equal(t, 800, desktopForm.DefaultHeight)
	assert.True(t, desktopForm.CenterScreen)

	// Verify sections
	assert.NotEmpty(t, desktopForm.Sections)
	assert.Greater(t, len(desktopForm.Sections), 0)

	// Find application section
	var appSection *DesktopConfigSection
	for _, section := range desktopForm.Sections {
		if section.ID == "application" {
			appSection = &section
			break
		}
	}

	require.NotNil(t, appSection)
	assert.Equal(t, "Application", appSection.Title)
	assert.Equal(t, "🚀", appSection.Icon)
	assert.Equal(t, "tab_page", appSection.Type)
	assert.True(t, appSection.Expanded)
	assert.NotEmpty(t, appSection.Fields)

	// Verify actions
	assert.NotEmpty(t, desktopForm.Actions)
	assert.Greater(t, len(desktopForm.Actions), 0)

	// Find save action
	var saveAction *DesktopConfigAction
	for _, action := range desktopForm.Actions {
		if action.ID == "save" {
			saveAction = &action
			break
		}
	}

	require.NotNil(t, saveAction)
	assert.Equal(t, "Save", saveAction.Label)
	assert.Equal(t, "primary", saveAction.Type)
	assert.Equal(t, "💾", saveAction.Icon)
	assert.Equal(t, "Ctrl+S", saveAction.Shortcut)
	assert.True(t, saveAction.Default)
	assert.False(t, saveAction.Cancel)
	assert.Equal(t, "right", saveAction.Position)
}

// TestWebUIAdapter tests web UI adapter
func TestWebUIAdapter(t *testing.T) {
	adapter := NewWebUIAdapter()

	// Test platform type
	assert.Equal(t, "web", adapter.GetPlatformType())

	// Test features
	features := adapter.GetPlatformFeatures()
	expectedFeatures := []string{
		"responsive_design",
		"pwa",
		"offline_support",
		"websockets",
		"touch_support",
		"browser_storage",
		"push_notifications",
		"service_worker",
		"css_animations",
	}

	for _, feature := range expectedFeatures {
		assert.Contains(t, features, feature)
	}

	// Test themes
	themes := adapter.GetPlatformThemes()
	assert.Contains(t, themes, "dark")
	assert.Contains(t, themes, "light")

	// Test config form rendering
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.json"

	configUI, err := NewConfigUI(configPath)
	require.NoError(t, err)

	form, err := adapter.RenderConfigForm(configUI)
	require.NoError(t, err)

	// Verify form structure
	webForm, ok := form.(WebConfigForm)
	require.True(t, ok)

	assert.Equal(t, "helix_config_form", webForm.ID)
	assert.Equal(t, "HelixCode Configuration", webForm.Title)
	assert.Equal(t, "spa_component", webForm.Type)
	assert.Equal(t, "responsive_tabs", webForm.Layout)
	assert.True(t, webForm.Responsive)
	assert.NotEmpty(t, webForm.JavaScript)
	assert.NotEmpty(t, webForm.CSS)

	// Verify sections
	assert.NotEmpty(t, webForm.Sections)

	// Find application section
	var appSection *WebConfigSection
	for _, section := range webForm.Sections {
		if section.ID == "application" {
			appSection = &section
			break
		}
	}

	require.NotNil(t, appSection)
	assert.Equal(t, "Application", appSection.Title)
	assert.Equal(t, "🚀", appSection.Icon)
	assert.Equal(t, "tab", appSection.Type)
	assert.NotEmpty(t, appSection.Fields)

	// Verify fields
	require.NotEmpty(t, appSection.Fields)

	// Find app name field
	var nameField *WebConfigField
	for _, field := range appSection.Fields {
		if field.ID == "app_name" {
			nameField = &field
			break
		}
	}

	require.NotNil(t, nameField)
	assert.Equal(t, "text", nameField.Type)
	assert.Equal(t, "Application Name", nameField.Label)
	assert.Equal(t, "text_input", nameField.Type) // Transformed
	assert.Equal(t, "form-control", nameField.Class)
	assert.False(t, nameField.Disabled)
	assert.True(t, nameField.Visible)
}

// TestMobileUIAdapter tests mobile UI adapter
func TestMobileUIAdapter(t *testing.T) {
	adapter := NewMobileUIAdapter()

	// Test platform type
	assert.Equal(t, "mobile", adapter.GetPlatformType())

	// Test features
	features := adapter.GetPlatformFeatures()
	expectedFeatures := []string{
		"touch_gestures",
		"biometric_auth",
		"push_notifications",
		"offline_first",
		"app_lifecycle",
		"camera_access",
		"location_services",
		"device_orientation",
		"native_plugins",
	}

	for _, feature := range expectedFeatures {
		assert.Contains(t, features, feature)
	}

	// Test themes
	themes := adapter.GetPlatformThemes()
	assert.Contains(t, themes, "mobile_light")
	assert.Contains(t, themes, "mobile_dark")

	// Test config form rendering
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.json"

	configUI, err := NewConfigUI(configPath)
	require.NoError(t, err)

	form, err := adapter.RenderConfigForm(configUI)
	require.NoError(t, err)

	// Verify form structure
	mobileForm, ok := form.(MobileConfigForm)
	require.True(t, ok)

	assert.Equal(t, "helix_config_form", mobileForm.ID)
	assert.Equal(t, "HelixCode Configuration", mobileForm.Title)
	assert.Equal(t, "mobile_screens", mobileForm.Type)
	assert.Equal(t, "carousel", mobileForm.Layout)
	assert.True(t, mobileForm.Responsive)
	assert.NotEmpty(t, mobileForm.Gestures)

	// Verify gestures
	assert.Contains(t, mobileForm.Gestures, "swipe")
	assert.Contains(t, mobileForm.Gestures, "tap")
	assert.Contains(t, mobileForm.Gestures, "double_tap")
	assert.Contains(t, mobileForm.Gestures, "pinch")

	// Verify sections
	assert.NotEmpty(t, mobileForm.Sections)

	// Find application section
	var appSection *MobileConfigSection
	for _, section := range mobileForm.Sections {
		if section.ID == "application" {
			appSection = &section
			break
		}
	}

	require.NotNil(t, appSection)
	assert.Equal(t, "Application", appSection.Title)
	assert.Equal(t, "🚀", appSection.Icon)
	assert.Equal(t, "screen", appSection.Type)
	assert.NotEmpty(t, appSection.Fields)

	// Verify fields
	require.NotEmpty(t, appSection.Fields)

	// Find app name field
	var nameField *MobileConfigField
	for _, field := range appSection.Fields {
		if field.ID == "app_name" {
			nameField = &field
			break
		}
	}

	require.NotNil(t, nameField)
	assert.Equal(t, "text", nameField.Type)
	assert.Equal(t, "Application Name", nameField.Label)
	assert.Equal(t, "default", nameField.Keyboard)
	assert.False(t, nameField.Disabled)
}

// TestTUIAdapter tests terminal UI adapter
func TestTUIAdapter(t *testing.T) {
	adapter := NewTUIAdapter()

	// Test platform type
	assert.Equal(t, "tui", adapter.GetPlatformType())

	// Test features
	features := adapter.GetPlatformFeatures()
	expectedFeatures := []string{
		"terminal_colors",
		"keyboard_navigation",
		"mouse_support",
		"terminal_fonts",
		"unicode_support",
		"screen_reader",
		"clipboard_access",
		"terminal_shortcuts",
		"resize_handling",
	}

	for _, feature := range expectedFeatures {
		assert.Contains(t, features, feature)
	}

	// Test themes
	themes := adapter.GetPlatformThemes()
	assert.Contains(t, themes, "terminal")

	// Test config form rendering
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.json"

	configUI, err := NewConfigUI(configPath)
	require.NoError(t, err)

	form, err := adapter.RenderConfigForm(configUI)
	require.NoError(t, err)

	// Verify form structure
	tuiForm, ok := form.(TUIConfigForm)
	require.True(t, ok)

	assert.Equal(t, "helix_config_form", tuiForm.ID)
	assert.Equal(t, "HelixCode Configuration", tuiForm.Title)
	assert.Equal(t, "tui_screens", tuiForm.Type)
	assert.Equal(t, "menu_driven", tuiForm.Layout)
	assert.Equal(t, "terminal", tuiForm.Theme)
	assert.True(t, tuiForm.Features != nil)
	assert.NotEmpty(t, tuiForm.KeyBindings)

	// Verify key bindings
	keyBindings := tuiForm.KeyBindings
	assert.Equal(t, "Ctrl+S", keyBindings["save"])
	assert.Equal(t, "Ctrl+R", keyBindings["reset"])
	assert.Equal(t, "Ctrl+Q", keyBindings["quit"])
	assert.Equal(t, "Tab", keyBindings["next_field"])
	assert.Equal(t, "Shift+Tab", keyBindings["prev_field"])
	assert.Equal(t, "Enter", keyBindings["select"])
	assert.Equal(t, "Esc", keyBindings["cancel"])

	// Verify sections
	assert.NotEmpty(t, tuiForm.Sections)

	// Find application section
	var appSection *TUIConfigSection
	for _, section := range tuiForm.Sections {
		if section.ID == "application" {
			appSection = &section
			break
		}
	}

	require.NotNil(t, appSection)
	assert.Equal(t, "Application", appSection.Title)
	assert.NotEmpty(t, appSection.Fields)

	// Verify fields
	require.NotEmpty(t, appSection.Fields)

	// Find app name field
	var nameField *TUIConfigField
	for _, field := range appSection.Fields {
		if field.ID == "app_name" {
			nameField = &field
			break
		}
	}

	require.NotNil(t, nameField)
	assert.Equal(t, "text", nameField.Type)
	assert.Equal(t, "Application Name", nameField.Label)
	assert.NotEmpty(t, nameField.HelpText)
}

// TestPlatformUIAdapterFactory tests factory function
func TestPlatformUIAdapterFactory(t *testing.T) {
	// Test desktop adapter
	desktopAdapter := GetPlatformUIAdapter("desktop")
	assert.IsType(t, &DesktopUIAdapter{}, desktopAdapter)
	assert.Equal(t, "desktop", desktopAdapter.GetPlatformType())

	// Test web adapter
	webAdapter := GetPlatformUIAdapter("web")
	assert.IsType(t, &WebUIAdapter{}, webAdapter)
	assert.Equal(t, "web", webAdapter.GetPlatformType())

	// Test mobile adapter
	mobileAdapter := GetPlatformUIAdapter("mobile")
	assert.IsType(t, &MobileUIAdapter{}, mobileAdapter)
	assert.Equal(t, "mobile", mobileAdapter.GetPlatformType())

	// Test TUI adapter
	tuiAdapter := GetPlatformUIAdapter("tui")
	assert.IsType(t, &TUIAdapter{}, tuiAdapter)
	assert.Equal(t, "tui", tuiAdapter.GetPlatformType())

	// Test default adapter (fallback to desktop)
	defaultAdapter := GetPlatformUIAdapter("unknown")
	assert.IsType(t, &DesktopUIAdapter{}, defaultAdapter)
	assert.Equal(t, "desktop", defaultAdapter.GetPlatformType())
}

// TestConfigFieldTransformation tests field transformation across platforms
func TestConfigFieldTransformation(t *testing.T) {
	// Create test config field
	field := ConfigField{
		ID:          "test_field",
		Type:        "text",
		Label:       "Test Field",
		Description: "Test description",
		Required:    true,
		UI: FieldUI{
			Placeholder: "Enter value",
			HelpText:    "Field help",
		},
	}

	// Test desktop transformation
	desktopAdapter := NewDesktopUIAdapter()
	desktopFields := desktopAdapter.transformFields([]ConfigField{field})
	require.Len(t, desktopFields, 1)

	desktopField := desktopFields[0]
	assert.Equal(t, "text_input", desktopField.Type)
	assert.Equal(t, 300, desktopField.Width)
	assert.Equal(t, 1, desktopField.TabIndex)
	assert.False(t, desktopField.Disabled)
	assert.True(t, desktopField.Visible)

	// Test web transformation
	webAdapter := NewWebUIAdapter()
	webFields := webAdapter.transformWebFields([]ConfigField{field})
	require.Len(t, webFields, 1)

	webField := webFields[0]
	assert.Equal(t, "text", webField.Type)
	assert.Equal(t, "form-control", webField.Class)
	assert.False(t, webField.Disabled)
	assert.True(t, webField.Visible)

	// Test mobile transformation
	mobileAdapter := NewMobileUIAdapter()
	mobileFields := mobileAdapter.transformMobileFields([]ConfigField{field})
	require.Len(t, mobileFields, 1)

	mobileField := mobileFields[0]
	assert.Equal(t, "text", mobileField.Type)
	assert.Equal(t, "default", mobileField.Keyboard)
	assert.False(t, mobileField.Disabled)

	// Test TUI transformation
	tuiAdapter := NewTUIAdapter()
	tuiFields := tuiAdapter.transformTUIFields([]ConfigField{field})
	require.Len(t, tuiFields, 1)

	tuiField := tuiFields[0]
	assert.Equal(t, "text", tuiField.Type)
	assert.NotEmpty(t, tuiField.HelpText)
}

// TestConfigActionTransformation tests action transformation across platforms
func TestConfigActionTransformation(t *testing.T) {
	// Create test action
	action := ConfigAction{
		ID:          "save",
		Label:       "Save",
		Description: "Save configuration",
		Type:        "primary",
		Icon:        "💾",
		Shortcut:    "Ctrl+S",
		Disabled:    false,
	}

	// Test desktop transformation
	desktopAdapter := NewDesktopUIAdapter()
	desktopActions := desktopAdapter.transformActions([]ConfigAction{action})
	require.Len(t, desktopActions, 1)

	desktopAction := desktopActions[0]
	assert.Equal(t, "default_button", desktopAction.Type)
	assert.True(t, desktopAction.Default)
	assert.False(t, desktopAction.Cancel)
	assert.Equal(t, "right", desktopAction.Position)

	// Test web transformation
	webAdapter := NewWebUIAdapter()
	webActions := webAdapter.transformWebActions([]ConfigAction{action})
	require.Len(t, webActions, 1)

	webAction := webActions[0]
	assert.Equal(t, "btn btn-primary", webAction.Class)
	assert.False(t, webAction.Disabled)

	// Test mobile transformation
	mobileAdapter := NewMobileUIAdapter()
	mobileActions := mobileAdapter.transformMobileActions([]ConfigAction{action})
	require.Len(t, mobileActions, 1)

	mobileAction := mobileActions[0]
	assert.Equal(t, "blue", mobileAction.Color)
	assert.False(t, mobileAction.Disabled)

	// Test TUI transformation
	tuiAdapter := NewTUIAdapter()
	tuiActions := tuiAdapter.transformTUIActions([]ConfigAction{action})
	require.Len(t, tuiActions, 1)

	tuiAction := tuiActions[0]
	assert.Equal(t, "Ctrl+S", tuiAction.Shortcut)
	assert.False(t, tuiAction.Disabled)
}

// TestConfigChangeHandling tests configuration change handling
func TestConfigChangeHandling(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.json"

	// Create initial config
	configUI, err := NewConfigUI(configPath)
	require.NoError(t, err)

	// Test desktop adapter
	desktopAdapter := NewDesktopUIAdapter()

	// Handle field change
	err = desktopAdapter.HandleConfigChange(configUI, "app_name", "Test App")
	require.NoError(t, err)

	// Verify change
	config := configUI.GetConfigManager().GetConfig()
	assert.Equal(t, "Test App", config.Application.Name)

	// Test number change
	err = desktopAdapter.HandleConfigChange(configUI, "server_port", 9090)
	require.NoError(t, err)

	config = configUI.GetConfigManager().GetConfig()
	assert.Equal(t, 9090, config.Server.Port)

	// Test float to int conversion
	err = desktopAdapter.HandleConfigChange(configUI, "server_port", float64(8080))
	require.NoError(t, err)

	config = configUI.GetConfigManager().GetConfig()
	assert.Equal(t, 8080, config.Server.Port)

	// Test temperature change
	err = desktopAdapter.HandleConfigChange(configUI, "llm_temperature", 0.8)
	require.NoError(t, err)

	config = configUI.GetConfigManager().GetConfig()
	assert.Equal(t, 0.8, config.LLM.Temperature)

	// Test web adapter
	webAdapter := NewWebUIAdapter()

	err = webAdapter.HandleConfigChange(configUI, "app_description", "Web Test Description")
	require.NoError(t, err)

	config = configUI.GetConfigManager().GetConfig()
	assert.Equal(t, "Web Test Description", config.Application.Description)

	// Test mobile adapter
	mobileAdapter := NewMobileUIAdapter()

	err = mobileAdapter.HandleConfigChange(configUI, "ui_theme", "mobile_light")
	require.NoError(t, err)

	config = configUI.GetConfigManager().GetConfig()
	assert.Equal(t, "mobile_light", config.UI.Theme)

	// Test TUI adapter
	tuiAdapter := NewTUIAdapter()

	err = tuiAdapter.HandleConfigChange(configUI, "ui_font_size", 16)
	require.NoError(t, err)

	config = configUI.GetConfigManager().GetConfig()
	assert.Equal(t, 16, config.UI.FontSize)
}

// TestPlatformValidation tests platform-specific validation
func TestPlatformValidation(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.json"

	configUI, err := NewConfigUI(configPath)
	require.NoError(t, err)

	// Test all adapters
	adapters := []PlatformUIAdapter{
		NewDesktopUIAdapter(),
		NewWebUIAdapter(),
		NewMobileUIAdapter(),
		NewTUIAdapter(),
	}

	for _, adapter := range adapters {
		errors, err := adapter.ValidateConfig(configUI)
		require.NoError(t, err)
		assert.NotNil(t, errors)
		// Should have no errors for default config
		assert.Empty(t, errors)
	}
}

// TestPlatformSpecificFeatures tests platform-specific features
func TestPlatformSpecificFeatures(t *testing.T) {
	// Test desktop features
	desktopAdapter := NewDesktopUIAdapter()
	desktopFeatures := desktopAdapter.GetPlatformFeatures()
	assert.Contains(t, desktopFeatures, "native_menus")
	assert.Contains(t, desktopFeatures, "system_tray")
	assert.Contains(t, desktopFeatures, "file_dialogs")

	// Test web features
	webAdapter := NewWebUIAdapter()
	webFeatures := webAdapter.GetPlatformFeatures()
	assert.Contains(t, webFeatures, "responsive_design")
	assert.Contains(t, webFeatures, "pwa")
	assert.Contains(t, webFeatures, "websockets")

	// Test mobile features
	mobileAdapter := NewMobileUIAdapter()
	mobileFeatures := mobileAdapter.GetPlatformFeatures()
	assert.Contains(t, mobileFeatures, "touch_gestures")
	assert.Contains(t, mobileFeatures, "biometric_auth")
	assert.Contains(t, mobileFeatures, "push_notifications")

	// Test TUI features
	tuiAdapter := NewTUIAdapter()
	tuiFeatures := tuiAdapter.GetPlatformFeatures()
	assert.Contains(t, tuiFeatures, "terminal_colors")
	assert.Contains(t, tuiFeatures, "keyboard_navigation")
	assert.Contains(t, tuiFeatures, "unicode_support")
}

// TestPlatformSpecificThemes tests platform-specific themes
func TestPlatformSpecificThemes(t *testing.T) {
	// Test desktop themes
	desktopAdapter := NewDesktopUIAdapter()
	desktopThemes := desktopAdapter.GetPlatformThemes()
	assert.Contains(t, desktopThemes, "system")

	systemTheme := desktopThemes["system"]
	assert.Equal(t, "system", systemTheme.Colors["primary"])
	assert.Equal(t, "system", systemTheme.Colors["background"])

	// Test mobile themes
	mobileAdapter := NewMobileUIAdapter()
	mobileThemes := mobileAdapter.GetPlatformThemes()
	assert.Contains(t, mobileThemes, "mobile_light")
	assert.Contains(t, mobileThemes, "mobile_dark")

	mobileLightTheme := mobileThemes["mobile_light"]
	assert.Equal(t, "#fafafa", mobileLightTheme.Colors["background"])
	assert.Equal(t, "#212121", mobileLightTheme.Colors["foreground"])
	assert.Equal(t, "#2196f3", mobileLightTheme.Colors["primary"])

	// Test TUI themes
	tuiAdapter := NewTUIAdapter()
	tuiThemes := tuiAdapter.GetPlatformThemes()
	assert.Contains(t, tuiThemes, "terminal")

	terminalTheme := tuiThemes["terminal"]
	assert.Equal(t, "#000000", terminalTheme.Colors["background"])
	assert.Equal(t, "#ffffff", terminalTheme.Colors["foreground"])
	assert.Equal(t, "#0000ff", terminalTheme.Colors["primary"])
}

// TestFieldTransformationByType tests field transformation for different types
func TestFieldTransformationByType(t *testing.T) {
	fieldTypes := []string{
		"text", "textarea", "number", "boolean", "select",
		"multiselect", "password", "file", "directory", "slider", "color",
	}

	// Test desktop adapter
	desktopAdapter := NewDesktopUIAdapter()

	for _, fieldType := range fieldTypes {
		field := ConfigField{
			ID:    "test_" + fieldType,
			Type:  fieldType,
			Label: "Test " + fieldType,
			UI:    FieldUI{},
		}

		desktopFields := desktopAdapter.transformFields([]ConfigField{field})
		require.Len(t, desktopFields, 1)

		transformedType := desktopAdapter.getDesktopFieldType(fieldType)
		assert.NotEmpty(t, transformedType)

		width := desktopAdapter.getDesktopFieldWidth(fieldType)
		assert.Greater(t, width, 0)
	}

	// Test action transformation
	actionTypes := []string{"primary", "secondary", "danger"}

	for _, actionType := range actionTypes {
		action := ConfigAction{
			ID:    "test_" + actionType,
			Type:  actionType,
			Label: "Test " + actionType,
		}

		desktopActions := desktopAdapter.transformActions([]ConfigAction{action})
		require.Len(t, desktopActions, 1)

		transformedType := desktopAdapter.getDesktopActionType(actionType)
		assert.NotEmpty(t, transformedType)

		position := desktopAdapter.getDesktopActionPosition(action.ID)
		assert.NotEmpty(t, position)
	}
}

// TestShowConfigDialog tests config dialog showing (simulated)
func TestShowConfigDialog(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.json"

	configUI, err := NewConfigUI(configPath)
	require.NoError(t, err)

	// Test all adapters
	adapters := []PlatformUIAdapter{
		NewDesktopUIAdapter(),
		NewWebUIAdapter(),
		NewMobileUIAdapter(),
		NewTUIAdapter(),
	}

	for _, adapter := range adapters {
		changesMade, err := adapter.ShowConfigDialog(configUI)
		require.NoError(t, err)
		assert.True(t, changesMade) // Simulated success
	}
}

// TestComplexFieldTypes tests complex field types and transformations
func TestComplexFieldTypes(t *testing.T) {
	// Create complex config field with options
	field := ConfigField{
		ID:    "llm_provider",
		Type:  "select",
		Label: "LLM Provider",
		UI: FieldUI{
			Options: []FieldOption{
				{
					Value:       "openai",
					Label:       "OpenAI",
					Description: "OpenAI GPT models",
					Icon:        "🤖",
					Group:       "cloud",
				},
				{
					Value:       "anthropic",
					Label:       "Anthropic",
					Description: "Anthropic Claude models",
					Icon:        "🧠",
					Group:       "cloud",
				},
				{
					Value:       "local",
					Label:       "Local",
					Description: "Local LLM models",
					Icon:        "💻",
					Group:       "local",
				},
			},
		},
	}

	// Test desktop transformation
	desktopAdapter := NewDesktopUIAdapter()
	desktopFields := desktopAdapter.transformFields([]ConfigField{field})
	require.Len(t, desktopFields, 1)

	desktopField := desktopFields[0]
	assert.Equal(t, "combo_box", desktopField.Type)
	assert.NotEmpty(t, desktopField.Options)
	assert.Equal(t, 200, desktopField.Width) // Select field width

	// Test web transformation
	webAdapter := NewWebUIAdapter()
	webFields := webAdapter.transformWebFields([]ConfigField{field})
	require.Len(t, webFields, 1)

	webField := webFields[0]
	assert.Equal(t, "select", webField.Type)
	assert.NotEmpty(t, webField.Options)

	// Test mobile transformation
	mobileAdapter := NewMobileUIAdapter()
	mobileFields := mobileAdapter.transformMobileFields([]ConfigField{field})
	require.Len(t, mobileFields, 1)

	mobileField := mobileFields[0]
	assert.Equal(t, "select", mobileField.Type)
	assert.NotEmpty(t, mobileField.Options)
	assert.Equal(t, "default", mobileField.Keyboard) // Select keyboard
}

// TestValidationWithPlatformAdapter tests validation with platform adapters
func TestValidationWithPlatformAdapter(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/test_config.json"

	configUI, err := NewConfigUI(configPath)
	require.NoError(t, err)

	// Create invalid configuration by making changes
	desktopAdapter := NewDesktopUIAdapter()

	// Invalid port
	err = desktopAdapter.HandleConfigChange(configUI, "server_port", 70000)
	require.NoError(t, err)

	// Invalid temperature
	err = desktopAdapter.HandleConfigChange(configUI, "llm_temperature", 3.0)
	require.NoError(t, err)

	// Validate with adapter
	errors, err := desktopAdapter.ValidateConfig(configUI)
	require.NoError(t, err)

	// Should have validation errors
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors, "server.port")
	assert.Contains(t, errors, "llm.temperature")
}

// TestGetPlatformUIAdapterForCurrentPlatform tests getting adapter for current platform
func TestGetPlatformUIAdapterForCurrentPlatform(t *testing.T) {
	// This test simulates getting adapter for current platform configuration
	adapter := GetPlatformUIAdapterForCurrentPlatform()
	assert.NotNil(t, adapter)

	// Should be a desktop adapter by default
	assert.IsType(t, &DesktopUIAdapter{}, adapter)
	assert.Equal(t, "desktop", adapter.GetPlatformType())
}

// BenchmarkPlatformUIAdapter benchmarks platform UI adapter operations
func BenchmarkPlatformUIAdapter(b *testing.B) {
	tempDir := b.TempDir()
	configPath := tempDir + "/bench_config.json"

	configUI, err := NewConfigUI(configPath)
	require.NoError(b, err)

	// Test desktop adapter rendering
	desktopAdapter := NewDesktopUIAdapter()

	b.Run("Desktop_RenderForm", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := desktopAdapter.RenderConfigForm(configUI)
			require.NoError(b, err)
		}
	})

	// Test field transformation
	field := ConfigField{
		ID:    "test_field",
		Type:  "text",
		Label: "Test Field",
		UI:    FieldUI{},
	}

	b.Run("Desktop_TransformField", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fields := desktopAdapter.transformFields([]ConfigField{field})
			require.Len(b, fields, 1)
		}
	})

	// Test config change handling
	b.Run("Desktop_HandleConfigChange", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := desktopAdapter.HandleConfigChange(configUI, "app_name", "Bench Test")
			require.NoError(b, err)
		}
	})
}
