package config

import (
	"fmt"
)

// PlatformUIAdapter represents the interface for platform-specific UI adapters
type PlatformUIAdapter interface {
	GetPlatformType() string
	RenderConfigForm(configUI *ConfigUI) (interface{}, error)
	ShowConfigDialog(configUI *ConfigUI) (bool, error)
	HandleConfigChange(configUI *ConfigUI, fieldID string, value interface{}) error
	ValidateConfig(configUI *ConfigUI) (map[string]string, error)
	GetPlatformThemes() map[string]ThemeConfig
	GetPlatformFeatures() []string
}

// BasePlatformAdapter provides common functionality for all platform adapters
type BasePlatformAdapter struct {
	platformType string
	features     []string
	themes       map[string]ThemeConfig
}

// NewBasePlatformAdapter creates a new base platform adapter
func NewBasePlatformAdapter(platformType string) *BasePlatformAdapter {
	adapter := &BasePlatformAdapter{
		platformType: platformType,
		features:     make([]string, 0),
		themes:       make(map[string]ThemeConfig),
	}

	adapter.initializeFeatures()
	adapter.initializeThemes()

	return adapter
}

// GetPlatformType returns the platform type
func (a *BasePlatformAdapter) GetPlatformType() string {
	return a.platformType
}

// initializeFeatures initializes platform-specific features
func (a *BasePlatformAdapter) initializeFeatures() {
	switch a.platformType {
	case "desktop":
		a.features = []string{
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
	case "web":
		a.features = []string{
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
	case "mobile":
		a.features = []string{
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
	case "tui":
		a.features = []string{
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
	}
}

// initializeThemes initializes platform-specific themes
func (a *BasePlatformAdapter) initializeThemes() {
	switch a.platformType {
	case "desktop":
		a.themes["system"] = ThemeConfig{
			Name:        "System",
			Description: "Matches system appearance",
			Colors: map[string]string{
				"primary":    "system",
				"background": "system",
				"foreground": "system",
			},
		}
	case "mobile":
		a.themes["mobile_light"] = ThemeConfig{
			Name:        "Mobile Light",
			Description: "Light theme optimized for mobile displays",
			Colors: map[string]string{
				"background": "#fafafa",
				"foreground": "#212121",
				"primary":    "#2196f3",
				"secondary":  "#f5f5f5",
				"accent":     "#03a9f4",
			},
		}
		a.themes["mobile_dark"] = ThemeConfig{
			Name:        "Mobile Dark",
			Description: "Dark theme optimized for mobile displays",
			Colors: map[string]string{
				"background": "#121212",
				"foreground": "#ffffff",
				"primary":    "#1976d2",
				"secondary":  "#2d2d2d",
				"accent":     "#03a9f4",
			},
		}
	case "tui":
		a.themes["terminal"] = ThemeConfig{
			Name:        "Terminal",
			Description: "Standard terminal colors",
			Colors: map[string]string{
				"background": "#000000",
				"foreground": "#ffffff",
				"primary":    "#0000ff",
				"secondary":  "#808080",
				"accent":     "#00ff00",
			},
		}
	}
}

// GetPlatformThemes returns platform-specific themes
func (a *BasePlatformAdapter) GetPlatformThemes() map[string]ThemeConfig {
	return a.themes
}

// GetPlatformFeatures returns platform-specific features
func (a *BasePlatformAdapter) GetPlatformFeatures() []string {
	return a.features
}

// DesktopUIAdapter implements UI adapter for desktop applications
type DesktopUIAdapter struct {
	*BasePlatformAdapter
}

// NewDesktopUIAdapter creates a new desktop UI adapter
func NewDesktopUIAdapter() *DesktopUIAdapter {
	return &DesktopUIAdapter{
		BasePlatformAdapter: NewBasePlatformAdapter("desktop"),
	}
}

// RenderConfigForm renders configuration form for desktop
func (a *DesktopUIAdapter) RenderConfigForm(configUI *ConfigUI) (interface{}, error) {
	// form := configUI.GetConfigForm()
	_ = configUI // TODO: Use configUI

	// Transform form for desktop rendering
	// TODO: Define DesktopConfigForm type
	// desktopForm := DesktopConfigForm{
	// 	ID:            form.ID,
	// 	Title:         form.Title,
	// 	Description:   form.Description,
	// 	Type:          "native_window",
	// 	Modal:         true,
	// 	Resizable:     true,
	// 	MinWidth:      800,
	// 	MinHeight:     600,
	// 	DefaultWidth:  1200,
	// 	DefaultHeight: 800,
	// 	CenterScreen:  true,
	// 	Layout:        "tabs",
	// 	Sections:      a.transformSections(form.Sections),
	// 	Actions:       a.transformActions(form.Actions),
	// 	Theme:         "system",
	// 	Features:      append(a.GetPlatformFeatures(), "native_controls"),
	// }

	// return desktopForm, nil
	return nil, fmt.Errorf("desktop form rendering not implemented")
}

// ShowConfigDialog shows configuration dialog on desktop
func (a *DesktopUIAdapter) ShowConfigDialog(configUI *ConfigUI) (bool, error) {
	// Implementation would use platform-specific UI framework
	// For now, return simulated result
	fmt.Printf("Showing desktop configuration dialog for platform: %s\n", a.GetPlatformType())

	// In a real implementation, this would:
	// 1. Create native window/dialog
	// 2. Render form controls
	// 3. Handle user interactions
	// 4. Save changes when user clicks OK/Apply
	// 5. Return true if changes were made

	return true, nil
}

// HandleConfigChange handles configuration changes on desktop
func (a *DesktopUIAdapter) HandleConfigChange(configUI *ConfigUI, fieldID string, value interface{}) error {
	// In a real implementation, this would:
	// 1. Validate the field value
	// 2. Update the configuration
	// 3. Update UI state
	// 4. Trigger any dependent field updates

	return UpdateHelixConfig(func(config *HelixConfig) {
		// Apply field change based on field ID
		applyFieldChangeGeneric(config, fieldID, value)
	})
}

// ValidateConfig validates configuration on desktop
func (a *DesktopUIAdapter) ValidateConfig(configUI *ConfigUI) (map[string]string, error) {
	errors := configUI.ValidateConfig()

	// In a real implementation, this would:
	// 1. Show validation errors in the UI
	// 2. Highlight fields with errors
	// 3. Show error tooltips
	// 4. Disable save button if errors exist

	return errors, nil
}

// transformSections transforms sections for desktop rendering
func (a *DesktopUIAdapter) transformSections(sections []ConfigSection) []DesktopConfigSection {
	desktopSections := make([]DesktopConfigSection, len(sections))

	for i, section := range sections {
		desktopSections[i] = DesktopConfigSection{
			ID:          section.ID,
			Title:       section.Title,
			Description: section.Description,
			Icon:        section.Icon,
			Type:        "tab_page",
			Expanded:    !section.Collapsed,
			Fields:      a.transformFields(section.Fields),
			Groups:      a.transformFieldGroups(section.Groups),
		}
	}

	return desktopSections
}

// transformFields transforms fields for desktop rendering
func (a *DesktopUIAdapter) transformFields(fields []ConfigField) []DesktopConfigField {
	desktopFields := make([]DesktopConfigField, len(fields))

	for i, field := range fields {
		desktopFields[i] = DesktopConfigField{
			ID:          field.ID,
			Type:        a.getDesktopFieldType(field.Type),
			Label:       field.Label,
			Description: field.Description,
			Value:       field.Default,
			Required:    field.Required,
			Placeholder: field.UI.Placeholder,
			HelpText:    field.UI.HelpText,
			Tooltip:     field.Description,
			Width:       a.getDesktopFieldWidth(field.Type),
			Options:     field.UI.Options,
			Validation:  field.Validation,
			Disabled:    false,
			Visible:     true,
			TabIndex:    i + 1,
		}
	}

	return desktopFields
}

// transformFieldGroups transforms field groups for desktop rendering
func (a *DesktopUIAdapter) transformFieldGroups(groups []ConfigFieldGroup) []DesktopConfigFieldGroup {
	desktopGroups := make([]DesktopConfigFieldGroup, len(groups))

	for i, group := range groups {
		desktopGroups[i] = DesktopConfigFieldGroup{
			ID:          group.ID,
			Title:       group.Title,
			Description: group.Description,
			Type:        "group_box",
			Layout:      group.Layout,
			Border:      true,
			Fields:      group.Fields,
			Spacing:     8,
			Margins:     DesktopMargins{Top: 16, Right: 16, Bottom: 16, Left: 16},
		}
	}

	return desktopGroups
}

// transformActions transforms actions for desktop rendering
func (a *DesktopUIAdapter) transformActions(actions []ConfigAction) []DesktopConfigAction {
	desktopActions := make([]DesktopConfigAction, len(actions))

	for i, action := range actions {
		desktopActions[i] = DesktopConfigAction{
			ID:           action.ID,
			Label:        action.Label,
			Description:  action.Description,
			Type:         a.getDesktopActionType(action.Type),
			Icon:         action.Icon,
			Shortcut:     action.Shortcut,
			Default:      action.ID == "save",
			Cancel:       action.ID == "reset",
			Confirmation: action.Confirmation,
			Position:     a.getDesktopActionPosition(action.ID),
			Width:        100,
			Height:       32,
		}
	}

	return desktopActions
}

// getDesktopFieldType returns desktop-specific field type
func (a *DesktopUIAdapter) getDesktopFieldType(fieldType string) string {
	switch fieldType {
	case "text":
		return "text_input"
	case "textarea":
		return "text_area"
	case "number":
		return "number_input"
	case "boolean":
		return "checkbox"
	case "select":
		return "combo_box"
	case "multiselect":
		return "list_box"
	case "password":
		return "password_input"
	case "file":
		return "file_picker"
	case "directory":
		return "directory_picker"
	case "slider":
		return "slider"
	case "color":
		return "color_picker"
	default:
		return "text_input"
	}
}

// getDesktopFieldWidth returns appropriate width for field type
func (a *DesktopUIAdapter) getDesktopFieldWidth(fieldType string) int {
	switch fieldType {
	case "boolean":
		return 120
	case "number":
		return 150
	case "select":
		return 200
	case "color":
		return 100
	default:
		return 300
	}
}

// getDesktopActionType returns desktop-specific action type
func (a *DesktopUIAdapter) getDesktopActionType(actionType string) string {
	switch actionType {
	case "primary":
		return "default_button"
	case "secondary":
		return "normal_button"
	case "danger":
		return "cancel_button"
	default:
		return "normal_button"
	}
}

// getDesktopActionPosition returns action button position
func (a *DesktopUIAdapter) getDesktopActionPosition(actionID string) string {
	switch actionID {
	case "save":
		return "right"
	case "reset":
		return "left"
	default:
		return "center"
	}
}

// applyFieldChange applies a field value change to the configuration

// DesktopConfigSection represents desktop configuration section
type DesktopConfigSection struct {
	ID          string                    ` + targetTab + `
	Title       string                    ` + targetTab + `
	Description string                    ` + targetTab + `
	Icon        string                    ` + targetTab + `
	Type        string                    ` + targetTab + `
	Expanded    bool                      ` + targetTab + `
	Fields      []DesktopConfigField      ` + targetTab + `
	Groups      []DesktopConfigFieldGroup ` + targetTab + `
}

// DesktopConfigField represents desktop configuration field
type DesktopConfigField struct {
	ID          string          ` + targetTab + `
	Type        string          ` + targetTab + `
	Label       string          ` + targetTab + `
	Description string          ` + targetTab + `
	Value       interface{}     ` + targetTab + `
	Required    bool            ` + targetTab + `
	Placeholder string          ` + targetTab + `
	HelpText    string          ` + targetTab + `
	Tooltip     string          ` + targetTab + `
	Width       int             ` + targetTab + `
	Options     []FieldOption   ` + targetTab + `
	Validation  FieldValidation ` + targetTab + `
	Disabled    bool            ` + targetTab + `
	Visible     bool            ` + targetTab + `
	TabIndex    int             ` + targetTab + `
}

// DesktopConfigFieldGroup represents desktop field group
type DesktopConfigFieldGroup struct {
	ID          string         ` + targetTab + `
	Title       string         ` + targetTab + `
	Description string         ` + targetTab + `
	Type        string         ` + targetTab + `
	Layout      string         ` + targetTab + `
	Border      bool           ` + targetTab + `
	Fields      []string       ` + targetTab + `
	Spacing     int            ` + targetTab + `
	Margins     DesktopMargins ` + targetTab + `
}

// DesktopConfigAction represents desktop action button
type DesktopConfigAction struct {
	ID           string             ` + targetTab + `
	Label        string             ` + targetTab + `
	Description  string             ` + targetTab + `
	Type         string             ` + targetTab + `
	Icon         string             ` + targetTab + `
	Shortcut     string             ` + targetTab + `
	Default      bool               ` + targetTab + `
	Cancel       bool               ` + targetTab + `
	Confirmation ActionConfirmation ` + targetTab + `
	Position     string             ` + targetTab + `
	Width        int                ` + targetTab + `
	Height       int                ` + targetTab + `
}

// DesktopMargins represents margins
type DesktopMargins struct {
	Top    int ` + targetTab + `
	Right  int ` + targetTab + `
	Bottom int ` + targetTab + `
	Left   int ` + targetTab + `
}

// WebUIAdapter implements UI adapter for web applications
type WebUIAdapter struct {
	*BasePlatformAdapter
}

// NewWebUIAdapter creates a new web UI adapter
func NewWebUIAdapter() *WebUIAdapter {
	return &WebUIAdapter{
		BasePlatformAdapter: NewBasePlatformAdapter("web"),
	}
}

// RenderConfigForm renders configuration form for web
func (a *WebUIAdapter) RenderConfigForm(configUI *ConfigUI) (interface{}, error) {
	form := configUI.GetConfigForm()

	// Transform form for web rendering
	webForm := WebConfigForm{
		ID:          form.ID,
		Title:       form.Title,
		Description: form.Description,
		Type:        "spa_component",
		Layout:      "responsive_tabs",
		Sections:    a.transformWebSections(form.Sections),
		Actions:     a.transformWebActions(form.Actions),
		Theme:       "auto",
		Responsive:  true,
		Features:    append(a.GetPlatformFeatures(), "progressive_enhancement"),
		CSS:         form.Layout.CSS,
		JavaScript:  a.getWebJavaScript(),
	}

	return webForm, nil
}

// ShowConfigDialog shows configuration dialog on web
func (a *WebUIAdapter) ShowConfigDialog(configUI *ConfigUI) (bool, error) {
	fmt.Printf("Showing web configuration dialog for platform: %s\n", a.GetPlatformType())
	return true, nil
}

// HandleConfigChange handles configuration changes on web
func (a *WebUIAdapter) HandleConfigChange(configUI *ConfigUI, fieldID string, value interface{}) error {
	return UpdateHelixConfig(func(config *HelixConfig) {
		applyFieldChangeGeneric(config, fieldID, value)
	})
}

// ValidateConfig validates configuration on web
func (a *WebUIAdapter) ValidateConfig(configUI *ConfigUI) (map[string]string, error) {
	errors := configUI.ValidateConfig()
	return errors, nil
}

// MobileUIAdapter implements UI adapter for mobile applications
type MobileUIAdapter struct {
	*BasePlatformAdapter
}

// NewMobileUIAdapter creates a new mobile UI adapter
func NewMobileUIAdapter() *MobileUIAdapter {
	return &MobileUIAdapter{
		BasePlatformAdapter: NewBasePlatformAdapter("mobile"),
	}
}

// RenderConfigForm renders configuration form for mobile
func (a *MobileUIAdapter) RenderConfigForm(configUI *ConfigUI) (interface{}, error) {
	form := configUI.GetConfigForm()

	// Transform form for mobile rendering
	mobileForm := MobileConfigForm{
		ID:          form.ID,
		Title:       form.Title,
		Description: form.Description,
		Type:        "mobile_screens",
		Layout:      "carousel",
		Sections:    a.transformMobileSections(form.Sections),
		Actions:     a.transformMobileActions(form.Actions),
		Theme:       "mobile_auto",
		Responsive:  true,
		Features:    append(a.GetPlatformFeatures(), "touch_optimized"),
		Gestures:    a.getMobileGestures(),
	}

	return mobileForm, nil
}

// ShowConfigDialog shows configuration dialog on mobile
func (a *MobileUIAdapter) ShowConfigDialog(configUI *ConfigUI) (bool, error) {
	fmt.Printf("Showing mobile configuration dialog for platform: %s\n", a.GetPlatformType())
	return true, nil
}

// HandleConfigChange handles configuration changes on mobile
func (a *MobileUIAdapter) HandleConfigChange(configUI *ConfigUI, fieldID string, value interface{}) error {
	return UpdateHelixConfig(func(config *HelixConfig) {
		applyFieldChangeGeneric(config, fieldID, value)
	})
}

// ValidateConfig validates configuration on mobile
func (a *MobileUIAdapter) ValidateConfig(configUI *ConfigUI) (map[string]string, error) {
	errors := configUI.ValidateConfig()
	return errors, nil
}

// TUIAdapter implements UI adapter for terminal UI
type TUIAdapter struct {
	*BasePlatformAdapter
}

// NewTUIAdapter creates a new TUI adapter
func NewTUIAdapter() *TUIAdapter {
	return &TUIAdapter{
		BasePlatformAdapter: NewBasePlatformAdapter("tui"),
	}
}

// RenderConfigForm renders configuration form for TUI
func (a *TUIAdapter) RenderConfigForm(configUI *ConfigUI) (interface{}, error) {
	form := configUI.GetConfigForm()

	// Transform form for TUI rendering
	tuiForm := TUIConfigForm{
		ID:          form.ID,
		Title:       form.Title,
		Description: form.Description,
		Type:        "tui_screens",
		Layout:      "menu_driven",
		Sections:    a.transformTUISections(form.Sections),
		Actions:     a.transformTUIActions(form.Actions),
		Theme:       "terminal",
		Features:    append(a.GetPlatformFeatures(), "keyboard_first"),
		KeyBindings: a.getTUIKeyBindings(),
	}

	return tuiForm, nil
}

// ShowConfigDialog shows configuration dialog on TUI
func (a *TUIAdapter) ShowConfigDialog(configUI *ConfigUI) (bool, error) {
	fmt.Printf("Showing TUI configuration dialog for platform: %s\n", a.GetPlatformType())
	return true, nil
}

// HandleConfigChange handles configuration changes on TUI
func (a *TUIAdapter) HandleConfigChange(configUI *ConfigUI, fieldID string, value interface{}) error {
	return UpdateHelixConfig(func(config *HelixConfig) {
		applyFieldChangeGeneric(config, fieldID, value)
	})
}

// ValidateConfig validates configuration on TUI
func (a *TUIAdapter) ValidateConfig(configUI *ConfigUI) (map[string]string, error) {
	errors := configUI.ValidateConfig()
	return errors, nil
}

// Helper structures for web, mobile, and TUI adapters

// WebConfigForm represents web configuration form
type WebConfigForm struct {
	ID          string             ` + targetTab + `
	Title       string             ` + targetTab + `
	Description string             ` + targetTab + `
	Type        string             ` + targetTab + `
	Layout      string             ` + targetTab + `
	Sections    []WebConfigSection ` + targetTab + `
	Actions     []WebConfigAction  ` + targetTab + `
	Theme       string             ` + targetTab + `
	Responsive  bool               ` + targetTab + `
	Features    []string           ` + targetTab + `
	CSS         string             ` + targetTab + `
	JavaScript  string             ` + targetTab + `
}

// WebConfigSection represents web configuration section
type WebConfigSection struct {
	ID          string           ` + targetTab + `
	Title       string           ` + targetTab + `
	Description string           ` + targetTab + `
	Icon        string           ` + targetTab + `
	Type        string           ` + targetTab + `
	Fields      []WebConfigField ` + targetTab + `
	Collapsed   bool             ` + targetTab + `
}

// WebConfigField represents web configuration field
type WebConfigField struct {
	ID          string        ` + targetTab + `
	Type        string        ` + targetTab + `
	Label       string        ` + targetTab + `
	Description string        ` + targetTab + `
	Value       interface{}   ` + targetTab + `
	Required    bool          ` + targetTab + `
	Placeholder string        ` + targetTab + `
	Class       string        ` + targetTab + `
	Options     []FieldOption ` + targetTab + `
	Disabled    bool          ` + targetTab + `
}

// WebConfigAction represents web action button
type WebConfigAction struct {
	ID       string ` + targetTab + `
	Label    string ` + targetTab + `
	Type     string ` + targetTab + `
	Icon     string ` + targetTab + `
	Class    string ` + targetTab + `
	Disabled bool   ` + targetTab + `
}

// MobileConfigForm represents mobile configuration form
type MobileConfigForm struct {
	ID          string                 ` + targetTab + `
	Title       string                 ` + targetTab + `
	Description string                 ` + targetTab + `
	Type        string                 ` + targetTab + `
	Layout      string                 ` + targetTab + `
	Sections    []MobileConfigSection  ` + targetTab + `
	Actions     []MobileConfigAction   ` + targetTab + `
	Theme       string                 ` + targetTab + `
	Responsive  bool                   ` + targetTab + `
	Features    []string               ` + targetTab + `
	Gestures    map[string]interface{} ` + targetTab + `
}

// MobileConfigSection represents mobile configuration section
type MobileConfigSection struct {
	ID          string              ` + targetTab + `
	Title       string              ` + targetTab + `
	Description string              ` + targetTab + `
	Icon        string              ` + targetTab + `
	Type        string              ` + targetTab + `
	Fields      []MobileConfigField ` + targetTab + `
}

// MobileConfigField represents mobile configuration field
type MobileConfigField struct {
	ID          string        ` + targetTab + `
	Type        string        ` + targetTab + `
	Label       string        ` + targetTab + `
	Description string        ` + targetTab + `
	Value       interface{}   ` + targetTab + `
	Required    bool          ` + targetTab + `
	Placeholder string        ` + targetTab + `
	Keyboard    string        ` + targetTab + `
	Options     []FieldOption ` + targetTab + `
}

// MobileConfigAction represents mobile action button
type MobileConfigAction struct {
	ID       string ` + targetTab + `
	Label    string ` + targetTab + `
	Type     string ` + targetTab + `
	Icon     string ` + targetTab + `
	Color    string ` + targetTab + `
	Disabled bool   ` + targetTab + `
}

// TUIConfigForm represents TUI configuration form
type TUIConfigForm struct {
	ID          string             ` + targetTab + `
	Title       string             ` + targetTab + `
	Description string             ` + targetTab + `
	Type        string             ` + targetTab + `
	Layout      string             ` + targetTab + `
	Sections    []TUIConfigSection ` + targetTab + `
	Actions     []TUIConfigAction  ` + targetTab + `
	Theme       string             ` + targetTab + `
	Features    []string           ` + targetTab + `
	KeyBindings map[string]string  ` + targetTab + `
}

// TUIConfigSection represents TUI configuration section
type TUIConfigSection struct {
	ID          string           ` + targetTab + `
	Title       string           ` + targetTab + `
	Description string           ` + targetTab + `
	Fields      []TUIConfigField ` + targetTab + `
}

// TUIConfigField represents TUI configuration field
type TUIConfigField struct {
	ID          string        ` + targetTab + `
	Type        string        ` + targetTab + `
	Label       string        ` + targetTab + `
	Description string        ` + targetTab + `
	Value       interface{}   ` + targetTab + `
	Required    bool          ` + targetTab + `
	Placeholder string        ` + targetTab + `
	Options     []FieldOption ` + targetTab + `
	HelpText    string        ` + targetTab + `
}

// TUIConfigAction represents TUI action
type TUIConfigAction struct {
	ID       string ` + targetTab + `
	Label    string ` + targetTab + `
	Type     string ` + targetTab + `
	Shortcut string ` + targetTab + `
	Disabled bool   ` + targetTab + `
}

// Helper methods for transformation

func (a *WebUIAdapter) transformWebSections(sections []ConfigSection) []WebConfigSection {
	// Transform sections for web rendering
	webSections := make([]WebConfigSection, len(sections))
	for i, section := range sections {
		webSections[i] = WebConfigSection{
			ID:          section.ID,
			Title:       section.Title,
			Description: section.Description,
			Icon:        section.Icon,
			Type:        "tab",
			Fields:      a.transformWebFields(section.Fields),
			Collapsed:   section.Collapsed,
		}
	}
	return webSections
}

func (a *WebUIAdapter) transformWebFields(fields []ConfigField) []WebConfigField {
	// Transform fields for web rendering
	webFields := make([]WebConfigField, len(fields))
	for i, field := range fields {
		webFields[i] = WebConfigField{
			ID:          field.ID,
			Type:        field.Type,
			Label:       field.Label,
			Description: field.Description,
			Value:       field.Default,
			Required:    field.Required,
			Placeholder: field.UI.Placeholder,
			Class:       "form-control",
			Options:     field.UI.Options,
			Disabled:    false,
		}
	}
	return webFields
}

func (a *WebUIAdapter) transformWebActions(actions []ConfigAction) []WebConfigAction {
	// Transform actions for web rendering
	webActions := make([]WebConfigAction, len(actions))
	for i, action := range actions {
		webActions[i] = WebConfigAction{
			ID:       action.ID,
			Label:    action.Label,
			Type:     action.Type,
			Icon:     action.Icon,
			Class:    "btn btn-" + action.Type,
			Disabled: action.Disabled,
		}
	}
	return webActions
}

func (a *WebUIAdapter) getWebJavaScript() string {
	return `
	// Web configuration form JavaScript
	class HelixConfigForm {
		constructor(formElement) {
			this.form = formElement;
			this.setupEventListeners();
			this.setupValidation();
		}
		
		setupEventListeners() {
			// Handle form changes
			this.form.addEventListener('change', (e) => {
				this.handleFieldChange(e.target);
			});
			
			// Handle form submission
			this.form.addEventListener('submit', (e) => {
				e.preventDefault();
				this.saveConfig();
			});
			
			// Handle tabs
			this.setupTabs();
		}
		
		setupValidation() {
			// Real-time validation
			const fields = this.form.querySelectorAll('input, select, textarea');
			fields.forEach(field => {
				field.addEventListener('blur', () => {
					this.validateField(field);
				});
			});
		}
		
		handleFieldChange(field) {
			// Send field change to backend
			const data = {
				field_id: field.id,
				value: field.value
			};
			
			fetch('/api/config/update', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify(data)
			})
			.then(response => response.json())
			.then(result => {
				if (!result.success) {
					this.showError(field, result.error);
				}
			})
			.catch(error => {
				this.showError(field, error.message);
			});
		}
		
		validateField(field) {
			// Client-side validation
			const errors = this.validateFieldRule(field);
			if (errors.length > 0) {
				this.showFieldErrors(field, errors);
			} else {
				this.clearFieldErrors(field);
			}
		}
		
		saveConfig() {
			// Save entire configuration
			const formData = new FormData(this.form);
			const data = Object.fromEntries(formData);
			
			fetch('/api/config/save', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify(data)
			})
			.then(response => response.json())
			.then(result => {
				if (result.success) {
					this.showSuccess('Configuration saved successfully');
				} else {
					this.showError(null, result.error);
				}
			})
			.catch(error => {
				this.showError(null, error.message);
			});
		}
		
		setupTabs() {
			const tabs = this.form.querySelectorAll('[data-tab]');
			const panels = this.form.querySelectorAll('[data-panel]');
			
			tabs.forEach(tab => {
				tab.addEventListener('click', () => {
					const targetTab = tab.dataset.tab;
					
					// Update active states
					tabs.forEach(t => t.classList.remove('active'));
					panels.forEach(p => p.classList.remove('active'));
					
					// Activate selected tab
					tab.classList.add('active');
					this.form.querySelector('[data-panel="' + targetTab + '"]').classList.add('active');
				});
			});
		}
		
		// Helper methods
		showError(field, message) {
			// Show error message
			console.error(message);
		}
		
		showSuccess(message) {
			// Show success message
			console.log(message);
		}
		
		showFieldErrors(field, errors) {
			// Show field validation errors
		}
		
		clearFieldErrors(field) {
			// Clear field validation errors
		}
		
		validateFieldRule(field) {
			// Validate individual field
			return [];
		}
	}
	
	// Initialize form when DOM is ready
	document.addEventListener('DOMContentLoaded', () => {
		const form = document.getElementById('helix-config-form');
		if (form) {
			new HelixConfigForm(form);
		}
	});
	`
}

func (a *MobileUIAdapter) transformMobileSections(sections []ConfigSection) []MobileConfigSection {
	// Transform sections for mobile rendering
	mobileSections := make([]MobileConfigSection, len(sections))
	for i, section := range sections {
		mobileSections[i] = MobileConfigSection{
			ID:          section.ID,
			Title:       section.Title,
			Description: section.Description,
			Icon:        section.Icon,
			Type:        "screen",
			Fields:      a.transformMobileFields(section.Fields),
		}
	}
	return mobileSections
}

func (a *MobileUIAdapter) transformMobileFields(fields []ConfigField) []MobileConfigField {
	// Transform fields for mobile rendering
	mobileFields := make([]MobileConfigField, len(fields))
	for i, field := range fields {
		mobileFields[i] = MobileConfigField{
			ID:          field.ID,
			Type:        field.Type,
			Label:       field.Label,
			Description: field.Description,
			Value:       field.Default,
			Required:    field.Required,
			Placeholder: field.UI.Placeholder,
			Keyboard:    a.getMobileKeyboardType(field.Type),
			Options:     field.UI.Options,
		}
	}
	return mobileFields
}

func (a *MobileUIAdapter) transformMobileActions(actions []ConfigAction) []MobileConfigAction {
	// Transform actions for mobile rendering
	mobileActions := make([]MobileConfigAction, len(actions))
	for i, action := range actions {
		mobileActions[i] = MobileConfigAction{
			ID:       action.ID,
			Label:    action.Label,
			Type:     action.Type,
			Icon:     action.Icon,
			Color:    a.getMobileActionColor(action.Type),
			Disabled: action.Disabled,
		}
	}
	return mobileActions
}

func (a *MobileUIAdapter) getMobileKeyboardType(fieldType string) string {
	switch fieldType {
	case "number":
		return "numeric"
	case "email":
		return "email"
	case "password":
		return "password"
	case "url":
		return "url"
	case "phone":
		return "phone"
	default:
		return "default"
	}
}

func (a *MobileUIAdapter) getMobileActionColor(actionType string) string {
	switch actionType {
	case "primary":
		return "blue"
	case "secondary":
		return "gray"
	case "danger":
		return "red"
	default:
		return "blue"
	}
}

func (a *MobileUIAdapter) getMobileGestures() map[string]interface{} {
	return map[string]interface{}{
		"swipe":      "horizontal",
		"tap":        "select",
		"double_tap": "confirm",
		"pinch":      "zoom",
		"scroll":     "navigate",
	}
}

func (a *TUIAdapter) transformTUISections(sections []ConfigSection) []TUIConfigSection {
	// Transform sections for TUI rendering
	tuiSections := make([]TUIConfigSection, len(sections))
	for i, section := range sections {
		tuiSections[i] = TUIConfigSection{
			ID:          section.ID,
			Title:       section.Title,
			Description: section.Description,
			Fields:      a.transformTUIFields(section.Fields),
		}
	}
	return tuiSections
}

func (a *TUIAdapter) transformTUIFields(fields []ConfigField) []TUIConfigField {
	// Transform fields for TUI rendering
	tuiFields := make([]TUIConfigField, len(fields))
	for i, field := range fields {
		tuiFields[i] = TUIConfigField{
			ID:          field.ID,
			Type:        field.Type,
			Label:       field.Label,
			Description: field.Description,
			Value:       field.Default,
			Required:    field.Required,
			Placeholder: field.UI.Placeholder,
			Options:     field.UI.Options,
			HelpText:    field.UI.HelpText,
		}
	}
	return tuiFields
}

func (a *TUIAdapter) transformTUIActions(actions []ConfigAction) []TUIConfigAction {
	// Transform actions for TUI rendering
	tuiActions := make([]TUIConfigAction, len(actions))
	for i, action := range actions {
		tuiActions[i] = TUIConfigAction{
			ID:       action.ID,
			Label:    action.Label,
			Type:     action.Type,
			Shortcut: action.Shortcut,
			Disabled: action.Disabled,
		}
	}
	return tuiActions
}

func (a *TUIAdapter) getTUIKeyBindings() map[string]string {
	return map[string]string{
		"save":           "Ctrl+S",
		"reset":          "Ctrl+R",
		"quit":           "Ctrl+Q",
		"next_field":     "Tab",
		"prev_field":     "Shift+Tab",
		"next_tab":       "Ctrl+Right",
		"prev_tab":       "Ctrl+Left",
		"toggle_help":    "F1",
		"toggle_menu":    "F10",
		"navigate_up":    "Up",
		"navigate_down":  "Down",
		"navigate_left":  "Left",
		"navigate_right": "Right",
		"select":         "Enter",
		"cancel":         "Esc",
	}
}

// Shared field change application method
func applyFieldChangeGeneric(config *HelixConfig, fieldID string, value interface{}) {
	// This method is shared across all adapters
	// It applies field changes to the configuration

	switch fieldID {
	case "app_name":
		if strValue, ok := value.(string); ok {
			config.Application.Name = strValue
		}
	case "app_description":
		if strValue, ok := value.(string); ok {
			config.Application.Description = strValue
		}
	case "app_version":
		if strValue, ok := value.(string); ok {
			config.Application.Version = strValue
		}
	case "app_environment":
		if strValue, ok := value.(string); ok {
			config.Application.Environment = strValue
		}
	case "server_address":
		if strValue, ok := value.(string); ok {
			config.Server.Address = strValue
		}
	case "server_port":
		if intValue, ok := value.(int); ok {
			config.Server.Port = intValue
		} else if floatValue, ok := value.(float64); ok {
			config.Server.Port = int(floatValue)
		}
	case "llm_default_provider":
		if strValue, ok := value.(string); ok {
			config.LLM.DefaultProvider = strValue
		}
	case "llm_default_model":
		if strValue, ok := value.(string); ok {
			config.LLM.DefaultModel = strValue
		}
	case "llm_max_tokens":
		if intValue, ok := value.(int); ok {
			config.LLM.MaxTokens = intValue
		} else if floatValue, ok := value.(float64); ok {
			config.LLM.MaxTokens = int(floatValue)
		}
	case "llm_temperature":
		if floatValue, ok := value.(float64); ok {
			config.LLM.Temperature = floatValue
		}
	case "ui_theme":
		if strValue, ok := value.(string); ok {
			config.UI.Theme = strValue
		}
	case "ui_language":
		if strValue, ok := value.(string); ok {
			config.UI.Language = strValue
		}
	case "ui_font_size":
		if intValue, ok := value.(int); ok {
			config.UI.FontSize = intValue
		} else if floatValue, ok := value.(float64); ok {
			config.UI.FontSize = int(floatValue)
		}
	}
}

// Factory function to get appropriate UI adapter for platform
func GetPlatformUIAdapter(platformType string) PlatformUIAdapter {
	switch platformType {
	case "desktop":
		return NewDesktopUIAdapter()
	case "web":
		return NewWebUIAdapter()
	case "mobile":
		return NewMobileUIAdapter()
	case "tui":
		return NewTUIAdapter()
	default:
		return NewDesktopUIAdapter() // Default to desktop
	}
}

// GetPlatformUIAdapterForCurrentPlatform gets UI adapter for current platform
func GetPlatformUIAdapterForCurrentPlatform() PlatformUIAdapter {
	config, err := LoadHelixConfig()
	if err != nil {
		// Fallback to desktop if config loading fails
		return NewDesktopUIAdapter()
	}

	return GetPlatformUIAdapter(config.Platform.CurrentPlatform)
}
