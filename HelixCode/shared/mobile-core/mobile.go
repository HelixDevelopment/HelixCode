// Package core provides shared functionality for HelixCode mobile clients
package core

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/notification"
	"dev.helix.code/internal/redis"
	"dev.helix.code/internal/server"
	"dev.helix.code/internal/task"
	"dev.helix.code/internal/worker"
)

// MobileCore represents the shared mobile core functionality
type MobileCore struct {
	config             *config.Config
	db                 *database.Database
	taskManager        *task.TaskManager
	workerManager      *worker.WorkerManager
	llmProvider        llm.Provider
	notificationEngine *notification.NotificationEngine
	server             *server.Server
	themeManager       *ThemeManager

	// Mobile-specific state
	isConnected  bool
	currentUser  string
	sessionToken string
	mu           sync.RWMutex
}

// Theme represents a UI theme for mobile
type Theme struct {
	Name       string
	IsDark     bool
	Primary    string
	Secondary  string
	Accent     string
	Text       string
	Background string
	Border     string
	Success    string
	Warning    string
	Error      string
	Info       string
}

// ThemeManager manages UI themes for mobile
type ThemeManager struct {
	currentTheme *Theme
	themes       map[string]*Theme
}

// NewThemeManager creates a new mobile theme manager
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		themes: make(map[string]*Theme),
	}

	// Register themes
	tm.themes["dark"] = &Theme{
		Name:       "Dark",
		IsDark:     true,
		Primary:    "#2E86AB",
		Secondary:  "#A23B72",
		Accent:     "#F18F01",
		Text:       "#FFFFFF",
		Background: "#1E1E1E",
		Border:     "#404040",
		Success:    "#4CAF50",
		Warning:    "#FF9800",
		Error:      "#F44336",
		Info:       "#2196F3",
	}

	tm.themes["light"] = &Theme{
		Name:       "Light",
		IsDark:     false,
		Primary:    "#1976D2",
		Secondary:  "#7B1FA2",
		Accent:     "#FF6F00",
		Text:       "#212121",
		Background: "#FFFFFF",
		Border:     "#BDBDBD",
		Success:    "#4CAF50",
		Warning:    "#FF9800",
		Error:      "#F44336",
		Info:       "#2196F3",
	}

	tm.themes["helix"] = &Theme{
		Name:       "Helix",
		IsDark:     true,
		Primary:    "#C2E95B",
		Secondary:  "#C0E853",
		Accent:     "#B8ECD7",
		Text:       "#2D3047",
		Background: "#1A1A1A",
		Border:     "#404040",
		Success:    "#4CAF50",
		Warning:    "#FF9800",
		Error:      "#F44336",
		Info:       "#2196F3",
	}

	// Set default theme
	tm.currentTheme = tm.themes["helix"]

	return tm
}

// GetCurrentTheme returns the current theme
func (tm *ThemeManager) GetCurrentTheme() *Theme {
	return tm.currentTheme
}

// SetTheme sets the current theme
func (tm *ThemeManager) SetTheme(themeName string) bool {
	if theme, exists := tm.themes[themeName]; exists {
		tm.currentTheme = theme
		return true
	}
	return false
}

// GetAvailableThemes returns available theme names
func (tm *ThemeManager) GetAvailableThemes() []string {
	names := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		names = append(names, name)
	}
	return names
}

// Exported functions for gomobile binding

// NewMobileCore creates a new mobile core instance
//
//export NewMobileCore
func NewMobileCore() *MobileCore {
	return &MobileCore{
		themeManager: NewThemeManager(),
		isConnected:  false,
	}
}

// Initialize initializes the mobile core
//
//export Initialize
func (mc *MobileCore) Initialize() error {
	return mc.initializeInternal()
}

// Connect establishes connection to the HelixCode server
//
//export Connect
func (mc *MobileCore) Connect(serverURL, username, password string) error {
	return mc.connectInternal(serverURL, username, password)
}

// Disconnect disconnects from the server
//
//export Disconnect
func (mc *MobileCore) Disconnect() error {
	return mc.disconnectInternal()
}

// IsConnected returns connection status
//
//export IsConnected
func (mc *MobileCore) IsConnected() bool {
	return mc.isConnectedInternal()
}

// GetCurrentUser returns the current user
//
//export GetCurrentUser
func (mc *MobileCore) GetCurrentUser() string {
	return mc.getCurrentUserInternal()
}

// GetDashboardData returns dashboard data as JSON
//
//export GetDashboardData
func (mc *MobileCore) GetDashboardData() string {
	return mc.getDashboardDataInternal()
}

// GetTasks returns tasks data as JSON
//
//export GetTasks
func (mc *MobileCore) GetTasks() string {
	return mc.getTasksInternal()
}

// GetWorkers returns workers data as JSON
//
//export GetWorkers
func (mc *MobileCore) GetWorkers() string {
	return mc.getWorkersInternal()
}

// CreateTask creates a new task
//
//export CreateTask
func (mc *MobileCore) CreateTask(name, description string) string {
	return mc.createTaskInternal(name, description)
}

// SendNotification sends a notification
//
//export SendNotification
func (mc *MobileCore) SendNotification(title, message, notificationType string) string {
	return mc.sendNotificationInternal(title, message, notificationType)
}

// GetTheme returns current theme data as JSON
//
//export GetTheme
func (mc *MobileCore) GetTheme() string {
	return mc.getThemeInternal()
}

// SetTheme sets the current theme
//
//export SetTheme
func (mc *MobileCore) SetTheme(themeName string) bool {
	return mc.setThemeInternal(themeName)
}

// GetAvailableThemes returns available themes as JSON
//
//export GetAvailableThemes
func (mc *MobileCore) GetAvailableThemes() string {
	return mc.getAvailableThemesInternal()
}

// Close cleans up resources
//
//export Close
func (mc *MobileCore) Close() error {
	return mc.closeInternal()
}

// Internal methods (renamed to avoid conflicts)
func (mc *MobileCore) initializeInternal() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}
	mc.config = cfg

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	mc.db = db

	// Initialize Redis
	rds, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %v", err)
	}

	// Initialize components
	mc.taskManager = task.NewTaskManager(db, rds)
	mc.workerManager = &worker.WorkerManager{} // Placeholder
	mc.notificationEngine = notification.NewNotificationEngine()

	// Initialize server for API calls
	mc.server = server.New(cfg, db, rds)

	log.Println("Mobile core initialized successfully")
	return nil
}

func (mc *MobileCore) connectInternal(serverURL, username, password string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// TODO: Implement actual authentication
	// For now, simulate connection
	mc.isConnected = true
	mc.currentUser = username

	log.Printf("Connected to server: %s as user: %s", serverURL, username)
	return nil
}

func (mc *MobileCore) disconnectInternal() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.isConnected = false
	mc.currentUser = ""
	mc.sessionToken = ""

	log.Println("Disconnected from server")
	return nil
}

func (mc *MobileCore) isConnectedInternal() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.isConnected
}

func (mc *MobileCore) getCurrentUserInternal() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.currentUser
}

func (mc *MobileCore) getDashboardDataInternal() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	data := map[string]interface{}{
		"isConnected": mc.isConnected,
		"user":        mc.currentUser,
		"theme":       mc.themeManager.GetCurrentTheme().Name,
		"stats": map[string]interface{}{
			"tasks":    0,
			"workers":  0,
			"projects": 0,
			"sessions": 0,
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	return string(jsonData)
}

func (mc *MobileCore) getTasksInternal() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// TODO: Get real tasks from task manager
	tasks := []map[string]interface{}{
		{
			"id":          "1",
			"name":        "Code Generation Task",
			"description": "Generate REST API endpoints",
			"status":      "running",
			"progress":    65,
		},
		{
			"id":          "2",
			"name":        "Testing Task",
			"description": "Run unit tests",
			"status":      "completed",
			"progress":    100,
		},
	}

	data := map[string]interface{}{
		"tasks": tasks,
		"total": len(tasks),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	return string(jsonData)
}

func (mc *MobileCore) getWorkersInternal() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// TODO: Get real workers from worker manager
	workers := []map[string]interface{}{
		{
			"id":       "worker-1",
			"hostname": "worker-1.local",
			"status":   "online",
			"cpu":      45,
			"memory":   67,
		},
	}

	data := map[string]interface{}{
		"workers": workers,
		"total":   len(workers),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	return string(jsonData)
}

func (mc *MobileCore) createTaskInternal(name, description string) string {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// TODO: Implement actual task creation
	result := map[string]interface{}{
		"success": true,
		"taskId":  "task-" + name,
		"message": "Task created successfully",
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	return string(jsonData)
}

func (mc *MobileCore) sendNotificationInternal(title, message, notificationType string) string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// TODO: Implement actual notification sending
	result := map[string]interface{}{
		"success": true,
		"message": "Notification sent",
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	return string(jsonData)
}

func (mc *MobileCore) getThemeInternal() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	theme := mc.themeManager.GetCurrentTheme()
	data := map[string]interface{}{
		"name":       theme.Name,
		"isDark":     theme.IsDark,
		"primary":    theme.Primary,
		"secondary":  theme.Secondary,
		"accent":     theme.Accent,
		"text":       theme.Text,
		"background": theme.Background,
		"border":     theme.Border,
		"success":    theme.Success,
		"warning":    theme.Warning,
		"error":      theme.Error,
		"info":       theme.Info,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	return string(jsonData)
}

func (mc *MobileCore) setThemeInternal(themeName string) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	return mc.themeManager.SetTheme(themeName)
}

func (mc *MobileCore) getAvailableThemesInternal() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	themes := mc.themeManager.GetAvailableThemes()
	data := map[string]interface{}{
		"themes": themes,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	return string(jsonData)
}

func (mc *MobileCore) closeInternal() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.db != nil {
		mc.db.Close()
	}

	log.Println("Mobile core closed")
	return nil
}
