package notification

import (
	"context"
	"fmt"
	"log"

	"dev.helix.code/internal/event"
)

// EventNotificationHandler handles system events and sends notifications
type EventNotificationHandler struct {
	engine *NotificationEngine
}

// NewEventNotificationHandler creates a new event notification handler
func NewEventNotificationHandler(engine *NotificationEngine) *EventNotificationHandler {
	return &EventNotificationHandler{
		engine: engine,
	}
}

// HandleEvent processes an event and sends appropriate notifications
func (h *EventNotificationHandler) HandleEvent(ctx context.Context, evt event.Event) error {
	if h.engine == nil {
		return fmt.Errorf("notification engine not initialized")
	}

	notification := h.eventToNotification(evt)
	if notification == nil {
		// Not all events need notifications
		return nil
	}

	log.Printf("Sending notification for event: %s (ID: %s)", evt.Type, evt.ID)

	// Send notification through engine which will apply rules
	return h.engine.SendNotification(ctx, notification)
}

// eventToNotification converts an event to a notification
func (h *EventNotificationHandler) eventToNotification(evt event.Event) *Notification {
	// Determine notification type and priority based on event severity
	notifType := h.severityToNotificationType(evt.Severity)
	priority := h.severityToPriority(evt.Severity)

	// Create notification based on event type
	switch evt.Type {
	// Task events
	case event.EventTaskCompleted:
		return h.createTaskCompletedNotification(evt, notifType, priority)
	case event.EventTaskFailed:
		return h.createTaskFailedNotification(evt, notifType, priority)
	case event.EventTaskStarted:
		// Don't send notifications for task started (too noisy)
		return nil

	// Workflow events
	case event.EventWorkflowCompleted:
		return h.createWorkflowCompletedNotification(evt, notifType, priority)
	case event.EventWorkflowFailed:
		return h.createWorkflowFailedNotification(evt, notifType, priority)

	// Worker events
	case event.EventWorkerDisconnected:
		return h.createWorkerDisconnectedNotification(evt, notifType, priority)
	case event.EventWorkerHealthDegraded:
		return h.createWorkerHealthDegradedNotification(evt, notifType, priority)

	// System events
	case event.EventSystemError:
		return h.createSystemErrorNotification(evt, notifType, priority)
	case event.EventSystemStartup:
		return h.createSystemStartupNotification(evt, notifType, priority)
	case event.EventSystemShutdown:
		return h.createSystemShutdownNotification(evt, notifType, priority)

	default:
		// Event type doesn't require notification
		return nil
	}
}

// Task notification creators

func (h *EventNotificationHandler) createTaskCompletedNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	taskID := h.getDataString(evt.Data, "task_id", evt.TaskID)
	duration := h.getDataString(evt.Data, "duration", "")

	message := fmt.Sprintf("Task %s completed successfully", taskID)
	if duration != "" {
		message += fmt.Sprintf(" in %s", duration)
	}

	return &Notification{
		Title:    "Task Completed",
		Message:  message,
		Type:     NotificationTypeSuccess,
		Priority: NotificationPriorityLow,
		Metadata: map[string]interface{}{
			"event_id":   evt.ID,
			"event_type": string(evt.Type),
			"task_id":    taskID,
			"duration":   duration,
			"project_id": evt.ProjectID,
			"user_id":    evt.UserID,
		},
	}
}

func (h *EventNotificationHandler) createTaskFailedNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	taskID := h.getDataString(evt.Data, "task_id", evt.TaskID)
	errorMsg := h.getDataString(evt.Data, "error", "Unknown error")

	message := fmt.Sprintf("Task %s failed: %s", taskID, errorMsg)

	return &Notification{
		Title:    "Task Failed",
		Message:  message,
		Type:     NotificationTypeError,
		Priority: NotificationPriorityHigh,
		Metadata: map[string]interface{}{
			"event_id":   evt.ID,
			"event_type": string(evt.Type),
			"task_id":    taskID,
			"error":      errorMsg,
			"project_id": evt.ProjectID,
			"user_id":    evt.UserID,
		},
	}
}

// Workflow notification creators

func (h *EventNotificationHandler) createWorkflowCompletedNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	workflowID := h.getDataString(evt.Data, "workflow_id", "")
	workflowName := h.getDataString(evt.Data, "workflow_name", workflowID)

	return &Notification{
		Title:    "Workflow Completed",
		Message:  fmt.Sprintf("Workflow '%s' completed successfully", workflowName),
		Type:     NotificationTypeSuccess,
		Priority: NotificationPriorityMedium,
		Metadata: map[string]interface{}{
			"event_id":     evt.ID,
			"event_type":   string(evt.Type),
			"workflow_id":  workflowID,
			"workflow_name": workflowName,
			"project_id":   evt.ProjectID,
			"user_id":      evt.UserID,
		},
	}
}

func (h *EventNotificationHandler) createWorkflowFailedNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	workflowID := h.getDataString(evt.Data, "workflow_id", "")
	workflowName := h.getDataString(evt.Data, "workflow_name", workflowID)
	errorMsg := h.getDataString(evt.Data, "error", "Unknown error")

	return &Notification{
		Title:    "Workflow Failed",
		Message:  fmt.Sprintf("Workflow '%s' failed: %s", workflowName, errorMsg),
		Type:     NotificationTypeError,
		Priority: NotificationPriorityHigh,
		Metadata: map[string]interface{}{
			"event_id":      evt.ID,
			"event_type":    string(evt.Type),
			"workflow_id":   workflowID,
			"workflow_name": workflowName,
			"error":         errorMsg,
			"project_id":    evt.ProjectID,
			"user_id":       evt.UserID,
		},
	}
}

// Worker notification creators

func (h *EventNotificationHandler) createWorkerDisconnectedNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	workerID := h.getDataString(evt.Data, "worker_id", evt.WorkerID)
	workerHost := h.getDataString(evt.Data, "host", "unknown")
	reason := h.getDataString(evt.Data, "reason", "Unknown reason")

	return &Notification{
		Title:    "Worker Disconnected",
		Message:  fmt.Sprintf("Worker %s (%s) disconnected: %s", workerID, workerHost, reason),
		Type:     NotificationTypeWarning,
		Priority: NotificationPriorityMedium,
		Metadata: map[string]interface{}{
			"event_id":  evt.ID,
			"event_type": string(evt.Type),
			"worker_id": workerID,
			"host":      workerHost,
			"reason":    reason,
		},
	}
}

func (h *EventNotificationHandler) createWorkerHealthDegradedNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	workerID := h.getDataString(evt.Data, "worker_id", evt.WorkerID)
	workerHost := h.getDataString(evt.Data, "host", "unknown")
	healthStatus := h.getDataString(evt.Data, "health_status", "degraded")

	return &Notification{
		Title:    "Worker Health Degraded",
		Message:  fmt.Sprintf("Worker %s (%s) health status: %s", workerID, workerHost, healthStatus),
		Type:     NotificationTypeWarning,
		Priority: NotificationPriorityMedium,
		Metadata: map[string]interface{}{
			"event_id":      evt.ID,
			"event_type":    string(evt.Type),
			"worker_id":     workerID,
			"host":          workerHost,
			"health_status": healthStatus,
		},
	}
}

// System notification creators

func (h *EventNotificationHandler) createSystemErrorNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	component := h.getDataString(evt.Data, "component", evt.Source)
	errorMsg := h.getDataString(evt.Data, "error", "Unknown error")

	return &Notification{
		Title:    "System Error",
		Message:  fmt.Sprintf("Error in %s: %s", component, errorMsg),
		Type:     NotificationTypeError,
		Priority: NotificationPriorityUrgent,
		Metadata: map[string]interface{}{
			"event_id":  evt.ID,
			"event_type": string(evt.Type),
			"component": component,
			"error":     errorMsg,
		},
	}
}

func (h *EventNotificationHandler) createSystemStartupNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	version := h.getDataString(evt.Data, "version", "")

	message := "HelixCode system started"
	if version != "" {
		message += fmt.Sprintf(" (version %s)", version)
	}

	return &Notification{
		Title:    "System Started",
		Message:  message,
		Type:     NotificationTypeInfo,
		Priority: NotificationPriorityLow,
		Metadata: map[string]interface{}{
			"event_id":  evt.ID,
			"event_type": string(evt.Type),
			"version":   version,
		},
	}
}

func (h *EventNotificationHandler) createSystemShutdownNotification(evt event.Event, notifType NotificationType, priority NotificationPriority) *Notification {
	reason := h.getDataString(evt.Data, "reason", "Normal shutdown")

	return &Notification{
		Title:    "System Shutdown",
		Message:  fmt.Sprintf("HelixCode system shutting down: %s", reason),
		Type:     NotificationTypeWarning,
		Priority: NotificationPriorityMedium,
		Metadata: map[string]interface{}{
			"event_id":  evt.ID,
			"event_type": string(evt.Type),
			"reason":    reason,
		},
	}
}

// Helper methods

func (h *EventNotificationHandler) severityToNotificationType(severity event.EventSeverity) NotificationType {
	switch severity {
	case event.SeverityCritical:
		return NotificationTypeAlert
	case event.SeverityError:
		return NotificationTypeError
	case event.SeverityWarning:
		return NotificationTypeWarning
	default:
		return NotificationTypeInfo
	}
}

func (h *EventNotificationHandler) severityToPriority(severity event.EventSeverity) NotificationPriority {
	switch severity {
	case event.SeverityCritical:
		return NotificationPriorityUrgent
	case event.SeverityError:
		return NotificationPriorityHigh
	case event.SeverityWarning:
		return NotificationPriorityMedium
	default:
		return NotificationPriorityLow
	}
}

func (h *EventNotificationHandler) getDataString(data map[string]interface{}, key string, defaultValue string) string {
	if data == nil {
		return defaultValue
	}
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// RegisterWithEventBus subscribes the handler to relevant events
func (h *EventNotificationHandler) RegisterWithEventBus(bus *event.EventBus) {
	// Task events
	bus.Subscribe(event.EventTaskCompleted, h.HandleEvent)
	bus.Subscribe(event.EventTaskFailed, h.HandleEvent)

	// Workflow events
	bus.Subscribe(event.EventWorkflowCompleted, h.HandleEvent)
	bus.Subscribe(event.EventWorkflowFailed, h.HandleEvent)

	// Worker events
	bus.Subscribe(event.EventWorkerDisconnected, h.HandleEvent)
	bus.Subscribe(event.EventWorkerHealthDegraded, h.HandleEvent)

	// System events
	bus.Subscribe(event.EventSystemError, h.HandleEvent)
	bus.Subscribe(event.EventSystemStartup, h.HandleEvent)
	bus.Subscribe(event.EventSystemShutdown, h.HandleEvent)

	log.Println("Notification event handler registered with event bus")
}
