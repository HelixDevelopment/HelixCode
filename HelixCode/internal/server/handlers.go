package server

import (
	"net/http"
	"time"

	"dev.helix.code/internal/auth"
	"dev.helix.code/internal/project"
	"dev.helix.code/internal/workflow"
	"github.com/gin-gonic/gin"
)

// Project Handlers

func (s *Server) listProjects(c *gin.Context) {
	// For now, return empty list until we have user authentication
	// In production, this would use: projectManager := project.NewDatabaseManager(s.db)
	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"projects": []interface{}{},
	})
}

// Auth Handlers

func (s *Server) register(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		Email       string `json:"email" binding:"required"`
		Password    string `json:"password" binding:"required"`
		DisplayName string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request",
			"error":   err.Error(),
		})
		return
	}

	user, err := s.auth.Register(c.Request.Context(), req.Username, req.Email, req.Password, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Registration failed",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"user":   user,
	})
}

func (s *Server) login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request",
			"error":   err.Error(),
		})
		return
	}

	session, user, err := s.auth.Login(c.Request.Context(), req.Username, req.Password, "rest_api", c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Login failed",
			"error":   err.Error(),
		})
		return
	}

	// Generate JWT token
	token, err := s.auth.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to generate token",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"user":    user,
		"token":   token,
		"session": session,
	})
}

func (s *Server) logout(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) <= 7 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid authorization header",
		})
		return
	}

	token := authHeader[7:]

	// For JWT, we can't invalidate it server-side, but we can log the logout
	// In a production system, you might want to maintain a blacklist
	if err := s.auth.Logout(c.Request.Context(), token); err != nil {
		// Log error but don't fail the request
		c.Error(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logged out successfully",
	})
}

func (s *Server) refreshToken(c *gin.Context) {
	// Get current user from context (set by auth middleware)
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "User not authenticated",
		})
		return
	}

	user, ok := userValue.(*auth.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Invalid user context",
		})
		return
	}

	// Generate new JWT token
	token, err := s.auth.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to generate token",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"token":  token,
	})
}

func (s *Server) getCurrentUser(c *gin.Context) {
	// Get current user from context (set by auth middleware)
	userValue, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "User not authenticated",
		})
		return
	}

	user, ok := userValue.(*auth.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Invalid user context",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"user":   user,
	})
}

func (s *Server) createProject(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Path        string `json:"path" binding:"required"`
		Type        string `json:"type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request",
			"error":   err.Error(),
		})
		return
	}

	// For now, return placeholder until we have user authentication
	// In production, this would use: projectManager := project.NewDatabaseManager(s.db)
	proj := gin.H{
		"id":          "proj_placeholder",
		"name":        req.Name,
		"description": req.Description,
		"path":        req.Path,
		"type":        req.Type,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"project": proj,
	})
}

func (s *Server) getProject(c *gin.Context) {
	id := c.Param("id")

	// For now, return placeholder until we have user authentication
	// In production, this would use: projectManager := project.NewDatabaseManager(s.db)
	proj := gin.H{
		"id":          id,
		"name":        "Sample Project",
		"description": "This is a sample project",
		"path":        "/path/to/project",
		"type":        "go",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"project": proj,
	})
}

func (s *Server) updateProject(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request",
			"error":   err.Error(),
		})
		return
	}

	// For now, return placeholder until we have user authentication
	// In production, this would use: projectManager := project.NewDatabaseManager(s.db)
	proj := gin.H{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"path":        "/path/to/project",
		"type":        "go",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"project": proj,
	})
}

func (s *Server) deleteProject(c *gin.Context) {
	// For now, return success until we have user authentication
	// In production, this would use: projectManager := project.NewDatabaseManager(s.db)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Project deleted",
	})
}

// Task Handlers

func (s *Server) listTasks(c *gin.Context) {
	// Return empty list for now
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"tasks":  []interface{}{},
	})
}

func (s *Server) createTask(c *gin.Context) {
	var req struct {
		Name         string                 `json:"name" binding:"required"`
		Description  string                 `json:"description"`
		Type         string                 `json:"type" binding:"required"`
		Priority     string                 `json:"priority"`
		Parameters   map[string]interface{} `json:"parameters"`
		Dependencies []string               `json:"dependencies"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request",
			"error":   err.Error(),
		})
		return
	}

	// Return placeholder task
	task := gin.H{
		"id":          "task_placeholder",
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"status":      "pending",
		"created_at":  time.Now(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"task":   task,
	})
}

func (s *Server) getTask(c *gin.Context) {
	id := c.Param("id")

	// Return placeholder task
	task := gin.H{
		"id":          id,
		"name":        "Sample Task",
		"description": "This is a sample task",
		"type":        "generic",
		"status":      "pending",
		"created_at":  time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"task":   task,
	})
}

func (s *Server) updateTask(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request",
			"error":   err.Error(),
		})
		return
	}

	// Return updated placeholder task
	task := gin.H{
		"id":          id,
		"name":        "Sample Task",
		"description": "This is a sample task",
		"type":        "generic",
		"status":      req.Status,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"task":   task,
	})
}

func (s *Server) deleteTask(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Task deleted",
	})
}

// Worker Handlers

func (s *Server) listWorkers(c *gin.Context) {
	// Return empty list for now
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"workers": []interface{}{},
	})
}

func (s *Server) getWorker(c *gin.Context) {
	id := c.Param("id")

	// Return placeholder worker
	worker := gin.H{
		"id":           id,
		"hostname":     "localhost",
		"status":       "active",
		"capabilities": []string{"build", "test"},
		"created_at":   time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"worker": worker,
	})
}

// System Handlers

func (s *Server) getSystemStats(c *gin.Context) {
	// Initialize counters
	var (
		totalTasks     = 0
		pendingTasks   = 0
		runningTasks   = 0
		completedTasks = 0
		failedTasks    = 0
		totalWorkers   = 0
		activeWorkers  = 0
	)

	// Get task statistics if task manager is available
	if s.taskManager != nil {
		tasks, err := s.taskManager.ListTasks(c.Request.Context())
		if err == nil {
			totalTasks = len(tasks)
			// Count tasks by status
			for _, t := range tasks {
				switch string(t.Status) {
				case "pending":
					pendingTasks++
				case "running":
					runningTasks++
				case "completed":
					completedTasks++
				case "failed":
					failedTasks++
				}
			}
		}
	}

	// Get worker statistics if worker manager is available
	if s.workerManager != nil {
		workers, err := s.workerManager.ListWorkers(c.Request.Context())
		if err == nil {
			totalWorkers = len(workers)
			// Count active workers
			for _, w := range workers {
				if string(w.Status) == "active" {
					activeWorkers++
				}
			}
		}
	}

	// Calculate uptime
	uptime := time.Since(s.startTime)

	stats := gin.H{
		"tasks": gin.H{
			"total":     totalTasks,
			"pending":   pendingTasks,
			"running":   runningTasks,
			"completed": completedTasks,
			"failed":    failedTasks,
		},
		"workers": gin.H{
			"total":  totalWorkers,
			"active": activeWorkers,
		},
		"system": gin.H{
			"uptime": uptime.String(),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"stats":  stats,
	})
}

func (s *Server) getSystemStatus(c *gin.Context) {
	// Check database connection
	dbStatus := "healthy"
	if err := s.db.HealthCheck(); err != nil {
		dbStatus = "unhealthy"
	}

	status := gin.H{
		"database": dbStatus,
		"api":      "healthy",
		"version":  "1.0.0",
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"system": status,
	})
}

// Workflow Handlers

func (s *Server) executePlanningWorkflow(c *gin.Context) {
	projectID := c.Param("projectId")

	projectManager := project.NewManager()
	workflowExecutor := workflow.NewExecutor(projectManager)

	wf, err := workflowExecutor.ExecutePlanningWorkflow(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to execute planning workflow",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"workflow": wf,
	})
}

func (s *Server) executeBuildingWorkflow(c *gin.Context) {
	projectID := c.Param("projectId")

	projectManager := project.NewManager()
	workflowExecutor := workflow.NewExecutor(projectManager)

	wf, err := workflowExecutor.ExecuteBuildingWorkflow(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to execute building workflow",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"workflow": wf,
	})
}

func (s *Server) executeTestingWorkflow(c *gin.Context) {
	projectID := c.Param("projectId")

	projectManager := project.NewManager()
	workflowExecutor := workflow.NewExecutor(projectManager)

	wf, err := workflowExecutor.ExecuteTestingWorkflow(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to execute testing workflow",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"workflow": wf,
	})
}

func (s *Server) executeRefactoringWorkflow(c *gin.Context) {
	projectID := c.Param("projectId")

	projectManager := project.NewManager()
	workflowExecutor := workflow.NewExecutor(projectManager)

	wf, err := workflowExecutor.ExecuteRefactoringWorkflow(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to execute refactoring workflow",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"workflow": wf,
	})
}
