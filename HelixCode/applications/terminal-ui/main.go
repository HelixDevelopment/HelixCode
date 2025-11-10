package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/database"
	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/notification"
	"dev.helix.code/internal/redis"
	"dev.helix.code/internal/server"
	"dev.helix.code/internal/task"
	"dev.helix.code/internal/worker"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TerminalUI represents the main terminal user interface
type TerminalUI struct {
	app                *tview.Application
	config             *config.Config
	helixConfig        *config.HelixConfig
	db                 *database.Database
	taskManager        *task.TaskManager
	workerManager      *worker.WorkerManager
	llmProvider        llm.Provider
	notificationEngine *notification.NotificationEngine
	server             *server.Server
	themeManager       *ThemeManager

	// UI Components
	pages     *tview.Pages
	mainFlex  *tview.Flex
	sidebar   *tview.List
	content   *tview.Pages
	statusBar *tview.TextView

	// Current state
	currentUser    string
	currentSession string
}

// NewTerminalUI creates a new Terminal UI instance
func NewTerminalUI() *TerminalUI {
	return &TerminalUI{
		app: tview.NewApplication(),
	}
}

// Initialize sets up the Terminal UI with all dependencies
func (tui *TerminalUI) Initialize() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}
	tui.config = cfg

	// Load Helix configuration
	helixCfg, err := config.LoadHelixConfig()
	if err != nil {
		// If helix config doesn't exist, create default
		helixCfg, _ = config.LoadDefaultConfig()
	}
	tui.helixConfig = helixCfg

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	tui.db = db

	// Initialize Redis
	rds, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %v", err)
	}

	// Initialize components
	tui.taskManager = task.NewTaskManager(db, rds)
	// For now, create a simple worker manager - will be improved later
	tui.workerManager = &worker.WorkerManager{} // Placeholder
	tui.notificationEngine = notification.NewNotificationEngine()

	// Initialize server for API calls
	tui.server = server.New(cfg, db, rds)

	// Initialize theme manager
	tui.themeManager = NewThemeManager()

	// Setup UI components
	tui.setupUI()

	return nil
}

// setupUI initializes the user interface components
func (tui *TerminalUI) setupUI() {
	// Create main pages container
	tui.pages = tview.NewPages()

	// Create main layout
	tui.mainFlex = tview.NewFlex().SetDirection(tview.FlexColumn)

	// Create sidebar navigation
	tui.sidebar = tview.NewList().
		AddItem("Dashboard", "System overview and status", 'd', tui.showDashboard).
		AddItem("Tasks", "Manage distributed tasks", 't', tui.showTasks).
		AddItem("Workers", "Monitor worker nodes", 'w', tui.showWorkers).
		AddItem("Projects", "Project management", 'p', tui.showProjects).
		AddItem("Sessions", "Active development sessions", 's', tui.showSessions).
		AddItem("LLM", "AI model interaction", 'l', tui.showLLM).
		AddItem("Settings", "Configuration and preferences", 'c', tui.showSettings).
		ShowSecondaryText(false)

	tui.sidebar.SetBorder(true).SetTitle("HelixCode v1.0.0")

	// Create content area
	tui.content = tview.NewPages()

	// Create status bar
	tui.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[green]Ready[white] | User: [yellow]Not logged in[white] | Session: [yellow]None").
		SetTextAlign(tview.AlignCenter)
	tui.statusBar.SetBorder(true).SetTitle("Status")

	// Setup main layout
	tui.mainFlex.
		AddItem(tui.sidebar, 25, 0, true).
		AddItem(tui.content, 0, 1, false)

	// Create main flex with status bar
	mainContainer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tui.mainFlex, 0, 1, false).
		AddItem(tui.statusBar, 3, 0, false)

	// Add main page
	tui.pages.AddPage("main", mainContainer, true, true)

	// Show initial dashboard
	tui.showDashboard()
}

// loadASCIIArt loads the ASCII art logo from file
func (tui *TerminalUI) loadASCIIArt() string {
	// Try to load from assets
	asciiPath := "assets/images/logo-ascii.txt"
	if data, err := ioutil.ReadFile(asciiPath); err == nil {
		return string(data)
	}

	// Fallback to default ASCII art
	return `
                 :=+**##**+-:
              :+##%#######%%#+-
            :*############**+*#*-
          .+#############*++***##*.
         :#############*++****#####:
        =#############+++****#######:
       =#*##########*+++****#########.
      =#**#########*++*****#########%*
     -#****#######*+++**#############%-
    .#*******#####++******#############
    +#**********#*+*+:.    :+#########%=
   :#**********#**+:         :*%#######*
   **************+             *%######%:
  :#************+    :=++=-.    #######%=
  +*************.  :***####*-   :%#####%+
  ************#-  -#***+--+##-   *######*
 :#************  -#***.    :**   -%######
 =***********#=  ****.  :-. -#-  :%######
 +#**********#: -#*#-  +##*. #=  .#######
 =++++++++++++. +**#. :#**#- *+  .######*
                +***. -#*##. #-  :%####%+
                +**#. .#**. =#   =%####%-
                +#*#-  =##-+#:   #######.
                =#*#*   -+*=.   =%####%*
                :#**#=         -######%:
                 *####=       =######%+
                 -#*####=---+**+****#*
                  +###########**+++**.
                   *##############%*.
                    +###########%#=
                     -*#%%##%%%#+.
                       :-++*+=:
`
}

// showDashboard displays the main dashboard
func (tui *TerminalUI) showDashboard() {
	dashboard := tview.NewFlex().SetDirection(tview.FlexRow)

	// Header with ASCII logo
	asciiArt := tui.loadASCIIArt()
	header := tview.NewTextView().
		SetText(asciiArt).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	header.SetBorder(true).SetTitle("HelixCode")

	// Stats grid
	statsGrid := tview.NewGrid().SetRows(3, 3, 3).SetColumns(30, 30, 30).SetBorders(true)

	// Worker stats
	workerStats := tui.createWorkerStatsView()
	statsGrid.AddItem(workerStats, 0, 0, 1, 1, 0, 0, false)

	// Task stats
	taskStats := tui.createTaskStatsView()
	statsGrid.AddItem(taskStats, 0, 1, 1, 1, 0, 0, false)

	// System stats
	systemStats := tui.createSystemStatsView()
	statsGrid.AddItem(systemStats, 0, 2, 1, 1, 0, 0, false)

	// Recent activity
	activityView := tui.createActivityView()
	statsGrid.AddItem(activityView, 1, 0, 1, 3, 0, 0, false)

	// Quick actions
	quickActions := tui.createQuickActionsView()
	statsGrid.AddItem(quickActions, 2, 0, 1, 3, 0, 0, false)

	dashboard.
		AddItem(header, 15, 0, false).
		AddItem(statsGrid, 0, 1, false)

	tui.content.SwitchToPage("dashboard")
	tui.content.AddPage("dashboard", dashboard, true, true)
}

// createWorkerStatsView creates the worker statistics view
func (tui *TerminalUI) createWorkerStatsView() tview.Primitive {
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetBorder(true)
	view.SetTitle("Workers")

	// Get real worker stats - for now simulate with basic data
	// In a real implementation, this would connect to the worker manager
	totalWorkers := 0
	activeWorkers := 0
	healthyWorkers := 0

	// Simulate some basic stats for demonstration
	content := fmt.Sprintf("[green]Total: %d\n[white]Active: %d\n[yellow]Healthy: %d\n[red]Failed: %d",
		totalWorkers, activeWorkers, healthyWorkers, totalWorkers-healthyWorkers)

	view.SetText(content)
	return view
}

// createTaskStatsView creates the task statistics view
func (tui *TerminalUI) createTaskStatsView() tview.Primitive {
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetBorder(true)
	view.SetTitle("Tasks")

	// Get real task stats - for now simulate with basic data
	// In a real implementation, this would connect to the task manager
	totalTasks := 0
	completedTasks := 0
	runningTasks := 0
	failedTasks := 0

	content := fmt.Sprintf("[blue]Total: %d\n[green]Completed: %d\n[yellow]Running: %d\n[red]Failed: %d",
		totalTasks, completedTasks, runningTasks, failedTasks)
	view.SetText(content)
	return view
}

// createSystemStatsView creates the system statistics view
func (tui *TerminalUI) createSystemStatsView() tview.Primitive {
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetBorder(true)
	view.SetTitle("System")

	// Get real system stats
	status := "[green]Status: Operational"
	uptime := "[white]Uptime: Running"
	memory := "[yellow]Memory: N/A"
	cpu := "[blue]CPU: N/A"

	// In a real implementation, this would get actual system metrics
	content := fmt.Sprintf("%s\n%s\n%s\n%s", status, uptime, memory, cpu)
	view.SetText(content)
	return view
}

// createActivityView creates the recent activity view
func (tui *TerminalUI) createActivityView() tview.Primitive {
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetBorder(true)
	view.SetTitle("Recent Activity")

	content := "• System initialized\n• Worker pool started\n• Task manager ready\n• LLM providers loaded"
	view.SetText(content)
	return view
}

// createQuickActionsView creates the quick actions view
func (tui *TerminalUI) createQuickActionsView() *tview.List {
	list := tview.NewList().
		AddItem("New Task", "Create a new distributed task", 'n', nil).
		AddItem("Add Worker", "Register a new worker node", 'a', nil).
		AddItem("LLM Chat", "Start AI conversation", 'c', nil).
		AddItem("View Logs", "Check system logs", 'l', nil)

	list.SetBorder(true).SetTitle("Quick Actions")
	return list
}

// showTasks displays the task management interface
func (tui *TerminalUI) showTasks() {
	tasksView := tview.NewFlex().SetDirection(tview.FlexRow)

	header := tview.NewTextView()
	header.SetText("[::b]Task Management")
	header.SetTextAlign(tview.AlignCenter)
	header.SetDynamicColors(true)
	header.SetBorder(true)

	// Create task list using components
	components := NewUIComponents(tui)

	// Sample task data - will be replaced with real data
	taskItems := []ListItem{
		{MainText: "Code Generation Task", SecondaryText: "Generate REST API endpoints", Shortcut: '1'},
		{MainText: "Testing Task", SecondaryText: "Run unit tests", Shortcut: '2'},
		{MainText: "Build Task", SecondaryText: "Compile application", Shortcut: '3'},
	}

	taskList := components.CreateList("Tasks", taskItems)

	// Action buttons
	actions := tview.NewFlex().SetDirection(tview.FlexColumn)
	newTaskBtn := tview.NewButton("New Task").SetSelectedFunc(func() {
		// TODO: Implement new task form
	})
	actions.AddItem(newTaskBtn, 0, 1, false)

	tasksView.
		AddItem(header, 3, 0, false).
		AddItem(taskList, 0, 1, false).
		AddItem(actions, 3, 0, false)

	tui.content.SwitchToPage("tasks")
	tui.content.AddPage("tasks", tasksView, true, true)
}

// showWorkers displays the worker management interface
func (tui *TerminalUI) showWorkers() {
	workersView := tview.NewFlex().SetDirection(tview.FlexRow)

	header := tview.NewTextView().
		SetText("[::b]Worker Management").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	header.SetBorder(true)

	// Worker list will be implemented in next phase
	workerList := tview.NewTextView().
		SetText("Worker list implementation pending...").
		SetTextAlign(tview.AlignCenter)
	workerList.SetBorder(true).SetTitle("Workers")

	workersView.
		AddItem(header, 3, 0, false).
		AddItem(workerList, 0, 1, false)

	tui.content.SwitchToPage("workers")
	tui.content.AddPage("workers", workersView, true, true)
}

// showProjects displays the project management interface
func (tui *TerminalUI) showProjects() {
	projectsView := tview.NewTextView().
		SetText("[::b]Project Management\n\nImplementation pending...").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	projectsView.SetBorder(true)

	tui.content.SwitchToPage("projects")
	tui.content.AddPage("projects", projectsView, true, true)
}

// showSessions displays the session management interface
func (tui *TerminalUI) showSessions() {
	sessionsView := tview.NewTextView().
		SetText("[::b]Session Management\n\nImplementation pending...").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	sessionsView.SetBorder(true)

	tui.content.SwitchToPage("sessions")
	tui.content.AddPage("sessions", sessionsView, true, true)
}

// showLLM displays the LLM interaction interface
func (tui *TerminalUI) showLLM() {
	llmView := tview.NewTextView().
		SetText("[::b]AI Model Interaction\n\nImplementation pending...").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	llmView.SetBorder(true)

	tui.content.SwitchToPage("llm")
	tui.content.AddPage("llm", llmView, true, true)
}

// showSettings displays the settings interface
func (tui *TerminalUI) showSettings() {
	settingsView := tview.NewFlex().SetDirection(tview.FlexRow)

	header := tview.NewTextView()
	header.SetText("[::b]Settings & Configuration")
	header.SetTextAlign(tview.AlignCenter)
	header.SetDynamicColors(true)
	header.SetBorder(true)

	// Create tabs for different settings categories
	tabs := tview.NewTextView()
	tabs.SetText("[::b]1. Theme  2. Cognee  3. System")
	tabs.SetTextAlign(tview.AlignCenter)
	tabs.SetDynamicColors(true)
	tabs.SetBorder(true)
	tabs.SetTitle("Settings Categories")

	// Theme settings
	themeView := tui.createThemeSettingsView()

	// Cognee settings
	cogneeView := tui.createCogneeSettingsView()

	// System settings
	systemView := tui.createSystemSettingsView()

	// Start with theme view
	contentArea := tview.NewPages()
	contentArea.AddPage("theme", themeView, true, true)
	contentArea.AddPage("cognee", cogneeView, true, true)
	contentArea.AddPage("system", systemView, true, true)
	contentArea.SwitchToPage("theme")

	// Handle tab navigation
	tabs.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1, tcell.KeyCtrlT:
			contentArea.SwitchToPage("theme")
			return nil
		case tcell.KeyF2, tcell.KeyCtrlC:
			contentArea.SwitchToPage("cognee")
			return nil
		case tcell.KeyF3, tcell.KeyCtrlS:
			contentArea.SwitchToPage("system")
			return nil
		}
		return event
	})

	settingsView.
		AddItem(header, 3, 0, false).
		AddItem(tabs, 3, 0, false).
		AddItem(contentArea, 0, 1, false)

	tui.content.SwitchToPage("settings")
	tui.content.AddPage("settings", settingsView, true, true)
}

// createThemeSettingsView creates the theme settings view
func (tui *TerminalUI) createThemeSettingsView() tview.Primitive {
	view := tview.NewFlex().SetDirection(tview.FlexRow)

	// Theme selection
	themeList := tview.NewList()
	themeList.SetBorder(true)
	themeList.SetTitle("Theme Selection")
	themeList.SetTitleAlign(tview.AlignLeft)

	themes := tui.themeManager.GetAvailableThemes()
	for _, themeName := range themes {
		themeList.AddItem(themeName, "", 0, func() {
			// This will be called when theme is selected
			// For now, just show a message
		})
	}

	// Current theme info
	currentTheme := tui.themeManager.GetCurrentTheme()
	themeInfo := tview.NewTextView()
	themeInfo.SetBorder(true)
	themeInfo.SetTitle("Current Theme")
	themeInfo.SetTitleAlign(tview.AlignLeft)
	themeInfo.SetText(fmt.Sprintf("Name: %s\nDark: %t\nPrimary: %s\nSecondary: %s\nAccent: %s",
		currentTheme.Name, currentTheme.IsDark,
		currentTheme.Primary, currentTheme.Secondary, currentTheme.Accent))

	view.
		AddItem(themeList, 0, 1, false).
		AddItem(themeInfo, 0, 1, false)

	return view
}

// createCogneeSettingsView creates the Cognee settings view
func (tui *TerminalUI) createCogneeSettingsView() tview.Primitive {
	view := tview.NewFlex().SetDirection(tview.FlexRow)

	// Cognee status and controls
	statusView := tview.NewTextView()
	statusView.SetBorder(true)
	statusView.SetTitle("Cognee Status")
	statusView.SetTitleAlign(tview.AlignLeft)
	statusView.SetDynamicColors(true)

	// Get Cognee config from helix config
	cogneeEnabled := "Disabled"
	cogneeMode := "N/A"
	if tui.helixConfig != nil && tui.helixConfig.Cognee.Enabled {
		cogneeEnabled = "[green]Enabled"
		cogneeMode = tui.helixConfig.Cognee.Mode
	}

	statusView.SetText(fmt.Sprintf("Status: %s\nMode: %s\nHost: %s\nPort: %d",
		cogneeEnabled, cogneeMode,
		tui.helixConfig.Cognee.Host, tui.helixConfig.Cognee.Port))

	// Control buttons
	controls := tview.NewFlex().SetDirection(tview.FlexColumn)

	enableBtn := tview.NewButton("Enable Cognee").SetSelectedFunc(func() {
		// TODO: Enable Cognee
		statusView.SetText("Status: [green]Enabled\nMode: local\nHost: localhost\nPort: 8000")
	})
	controls.AddItem(enableBtn, 0, 1, false)

	disableBtn := tview.NewButton("Disable Cognee").SetSelectedFunc(func() {
		// TODO: Disable Cognee
		statusView.SetText("Status: [red]Disabled\nMode: N/A\nHost: N/A\nPort: N/A")
	})
	controls.AddItem(disableBtn, 0, 1, false)

	// Configuration options
	configView := tview.NewTextView()
	configView.SetBorder(true)
	configView.SetTitle("Configuration Options")
	configView.SetTitleAlign(tview.AlignLeft)
	configView.SetText(`[::b]Basic Settings:
• Auto Start: Enabled
• Mode: Local/Remote
• Host: localhost
• Port: 8000

[::b]Features:
• Knowledge Graph: Enabled
• Semantic Search: Enabled
• Real-time Processing: Enabled
• Multi-modal Support: Enabled

[::b]Performance:
• Workers: 4
• Cache: Redis
• Optimization: High`)

	view.
		AddItem(statusView, 6, 0, false).
		AddItem(controls, 3, 0, false).
		AddItem(configView, 0, 1, false)

	return view
}

// createSystemSettingsView creates the system settings view
func (tui *TerminalUI) createSystemSettingsView() tview.Primitive {
	view := tview.NewTextView()
	view.SetBorder(true)
	view.SetTitle("System Settings")
	view.SetTitleAlign(tview.AlignLeft)
	view.SetText(`[::b]System Configuration:

Database: PostgreSQL
Redis: Enabled
Workers: 4 active
Tasks: 0 running

[::b]Performance:
CPU Usage: 15%
Memory: 2.1GB / 8GB
Disk: 45GB free

[::b]Network:
Port: 8080
SSL: Disabled
CORS: Enabled`)

	return view
}

// Run starts the Terminal UI application
func (tui *TerminalUI) Run() error {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the application in a goroutine
	go func() {
		if err := tui.app.SetRoot(tui.pages, true).Run(); err != nil {
			log.Printf("TUI application error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	tui.app.Stop()

	return nil
}

// Close cleans up resources
func (tui *TerminalUI) Close() error {
	if tui.db != nil {
		tui.db.Close()
	}
	return nil
}

func main() {
	tui := NewTerminalUI()

	if err := tui.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Terminal UI: %v", err)
	}
	defer tui.Close()

	if err := tui.Run(); err != nil {
		log.Fatalf("Terminal UI error: %v", err)
	}
}
