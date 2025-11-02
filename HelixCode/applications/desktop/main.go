package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/notification"
	"dev.helix.code/internal/redis"
	"dev.helix.code/internal/server"
	"dev.helix.code/internal/task"
	"dev.helix.code/internal/worker"
)

// DesktopApp represents the desktop application
type DesktopApp struct {
	fyneApp            fyne.App
	mainWindow         fyne.Window
	config             *config.Config
	db                 *database.Database
	taskManager        *task.TaskManager
	workerManager      *worker.WorkerManager
	llmProvider        llm.Provider
	notificationEngine *notification.NotificationEngine
	server             *server.Server
	themeManager       *ThemeManager

	// UI Components
	tabs      *container.AppTabs
	statusBar *widget.Label
}

// NewDesktopApp creates a new desktop application
func NewDesktopApp() *DesktopApp {
	fyneApp := app.New()
	fyneApp.Settings().SetTheme(&CustomTheme{})

	return &DesktopApp{
		fyneApp: fyneApp,
	}
}

// Initialize sets up the desktop application
func (app *DesktopApp) Initialize() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}
	app.config = cfg

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	app.db = db

	// Initialize Redis
	rds, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %v", err)
	}

	// Initialize components
	app.taskManager = task.NewTaskManager(db, rds)
	app.workerManager = &worker.WorkerManager{} // Placeholder
	app.notificationEngine = notification.NewNotificationEngine()

	// Initialize server for API calls
	app.server = server.New(cfg, db, rds)

	// Initialize theme manager
	app.themeManager = NewThemeManager()

	// Setup UI
	app.setupUI()

	return nil
}

// setupUI initializes the user interface
func (app *DesktopApp) setupUI() {
	// Create main window
	app.mainWindow = app.fyneApp.NewWindow("HelixCode - Distributed AI Development Platform")
	app.mainWindow.SetMaster()
	app.mainWindow.Resize(fyne.NewSize(1200, 800))

	// Create tabs
	app.tabs = container.NewAppTabs(
		container.NewTabItem("Dashboard", app.createDashboardTab()),
		container.NewTabItem("Tasks", app.createTasksTab()),
		container.NewTabItem("Workers", app.createWorkersTab()),
		container.NewTabItem("Projects", app.createProjectsTab()),
		container.NewTabItem("Sessions", app.createSessionsTab()),
		container.NewTabItem("LLM", app.createLLMTab()),
		container.NewTabItem("Settings", app.createSettingsTab()),
	)

	// Create status bar
	app.statusBar = widget.NewLabel("Ready | User: Not logged in | Session: None")
	app.statusBar.Alignment = fyne.TextAlignCenter

	// Create main layout
	mainContent := container.NewBorder(nil, app.statusBar, nil, nil, app.tabs)

	app.mainWindow.SetContent(mainContent)
}

// createDashboardTab creates the dashboard tab
func (app *DesktopApp) createDashboardTab() fyne.CanvasObject {
	// Header with logo placeholder
	header := widget.NewLabel("🌀 HelixCode - Distributed AI Development Platform")
	header.Alignment = fyne.TextAlignCenter
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Stats cards
	workerCard := widget.NewCard("Workers", "", widget.NewLabel("Total: 0\nActive: 0\nHealthy: 0"))
	taskCard := widget.NewCard("Tasks", "", widget.NewLabel("Total: 0\nCompleted: 0\nRunning: 0"))
	systemCard := widget.NewCard("System", "", widget.NewLabel("Status: Operational\nUptime: 00:00:00"))

	statsContainer := container.NewGridWithColumns(3, workerCard, taskCard, systemCard)

	// Activity log
	activityLog := widget.NewMultiLineEntry()
	activityLog.SetText("• System initialized\n• Worker pool started\n• Task manager ready\n• LLM providers loaded")
	activityLog.Disable()

	activityCard := widget.NewCard("Recent Activity", "", activityLog)

	// Quick actions
	actionsCard := widget.NewCard("Quick Actions", "",
		container.NewVBox(
			widget.NewButton("New Task", func() {}),
			widget.NewButton("Add Worker", func() {}),
			widget.NewButton("LLM Chat", func() {}),
		),
	)

	bottomContainer := container.NewGridWithColumns(2, activityCard, actionsCard)

	return container.NewVBox(header, statsContainer, bottomContainer)
}

// createTasksTab creates the tasks tab
func (app *DesktopApp) createTasksTab() fyne.CanvasObject {
	// Task list
	taskList := widget.NewList(
		func() int { return 3 }, // Number of items
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			tasks := []string{"Code Generation Task", "Testing Task", "Build Task"}
			obj.(*widget.Label).SetText(tasks[id])
		},
	)

	taskCard := widget.NewCard("Tasks", "", taskList)

	// Action buttons
	actions := container.NewVBox(
		widget.NewButton("New Task", func() {}),
		widget.NewButton("Refresh", func() {}),
	)

	return container.NewBorder(nil, nil, nil, actions, taskCard)
}

// createWorkersTab creates the workers tab
func (app *DesktopApp) createWorkersTab() fyne.CanvasObject {
	workerList := widget.NewList(
		func() int { return 0 }, // No workers for now
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(fmt.Sprintf("Worker %d", id+1))
		},
	)

	workerCard := widget.NewCard("Workers", "", workerList)

	actions := container.NewVBox(
		widget.NewButton("Add Worker", func() {}),
		widget.NewButton("Refresh", func() {}),
	)

	return container.NewBorder(nil, nil, nil, actions, workerCard)
}

// createProjectsTab creates the projects tab
func (app *DesktopApp) createProjectsTab() fyne.CanvasObject {
	return widget.NewCard("Projects", "Project management coming soon", widget.NewLabel("Implementation pending..."))
}

// createSessionsTab creates the sessions tab
func (app *DesktopApp) createSessionsTab() fyne.CanvasObject {
	return widget.NewCard("Sessions", "Session management coming soon", widget.NewLabel("Implementation pending..."))
}

// createLLMTab creates the LLM tab
func (app *DesktopApp) createLLMTab() fyne.CanvasObject {
	return widget.NewCard("AI Models", "LLM interaction coming soon", widget.NewLabel("Implementation pending..."))
}

// createSettingsTab creates the settings tab
func (app *DesktopApp) createSettingsTab() fyne.CanvasObject {
	// Theme selection
	themeSelect := widget.NewSelect(app.themeManager.GetAvailableThemes(), func(selected string) {
		app.themeManager.SetTheme(selected)
		// Apply theme change
	})
	themeSelect.SetSelected(app.themeManager.GetCurrentTheme().Name)

	themeCard := widget.NewCard("Theme", "Select application theme", themeSelect)

	// Current theme info
	currentTheme := app.themeManager.GetCurrentTheme()
	themeInfo := fmt.Sprintf("Name: %s\nDark: %t\nPrimary: %s\nSecondary: %s\nAccent: %s",
		currentTheme.Name, currentTheme.IsDark,
		currentTheme.Primary, currentTheme.Secondary, currentTheme.Accent)

	infoLabel := widget.NewLabel(themeInfo)
	infoCard := widget.NewCard("Current Theme", "", infoLabel)

	return container.NewVBox(themeCard, infoCard)
}

// Run starts the desktop application
func (app *DesktopApp) Run() {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Show window
	app.mainWindow.ShowAndRun()

	// Wait for shutdown signal
	<-sigChan
	app.fyneApp.Quit()
}

// Close cleans up resources
func (app *DesktopApp) Close() error {
	if app.db != nil {
		app.db.Close()
	}
	return nil
}

func main() {
	desktopApp := NewDesktopApp()

	if err := desktopApp.Initialize(); err != nil {
		log.Fatalf("Failed to initialize desktop app: %v", err)
	}
	defer desktopApp.Close()

	desktopApp.Run()
}
