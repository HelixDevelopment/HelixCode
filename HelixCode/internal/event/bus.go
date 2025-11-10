package event

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of system event
type EventType string

const (
	// Task events
	EventTaskCreated   EventType = "task.created"
	EventTaskAssigned  EventType = "task.assigned"
	EventTaskStarted   EventType = "task.started"
	EventTaskCompleted EventType = "task.completed"
	EventTaskFailed    EventType = "task.failed"
	EventTaskPaused    EventType = "task.paused"
	EventTaskResumed   EventType = "task.resumed"
	EventTaskCancelled EventType = "task.cancelled"

	// Workflow events
	EventWorkflowStarted   EventType = "workflow.started"
	EventWorkflowCompleted EventType = "workflow.completed"
	EventWorkflowFailed    EventType = "workflow.failed"
	EventStepCompleted     EventType = "step.completed"
	EventStepFailed        EventType = "step.failed"

	// Worker events
	EventWorkerConnected       EventType = "worker.connected"
	EventWorkerDisconnected    EventType = "worker.disconnected"
	EventWorkerHealthDegraded  EventType = "worker.health_degraded"
	EventWorkerHeartbeatMissed EventType = "worker.heartbeat_missed"
	EventWorkerTaskAssigned    EventType = "worker.task_assigned"
	EventWorkerTaskCompleted   EventType = "worker.task_completed"

	// API events
	EventUserRegistered EventType = "user.registered"
	EventUserLogin      EventType = "user.login"
	EventUserLogout     EventType = "user.logout"
	EventProjectCreated EventType = "project.created"
	EventProjectDeleted EventType = "project.deleted"
	EventAuthFailure    EventType = "auth.failure"

	// System events
	EventSystemStartup  EventType = "system.startup"
	EventSystemShutdown EventType = "system.shutdown"
	EventSystemError    EventType = "system.error"
)

// EventSeverity indicates the importance of an event
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityError    EventSeverity = "error"
	SeverityCritical EventSeverity = "critical"
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Severity  EventSeverity          `json:"severity"`
	Data      map[string]interface{} `json:"data"`
	UserID    string                 `json:"user_id,omitempty"`
	ProjectID string                 `json:"project_id,omitempty"`
	TaskID    string                 `json:"task_id,omitempty"`
	WorkerID  string                 `json:"worker_id,omitempty"`
}

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event Event) error

// EventBus manages event subscriptions and publishing
type EventBus struct {
	subscribers map[EventType][]EventHandler
	mutex       sync.RWMutex
	async       bool
	errorLog    []error
	errorMutex  sync.Mutex
}

// NewEventBus creates a new event bus
func NewEventBus(async bool) *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		async:       async,
		errorLog:    make([]error, 0),
	}
}

// Subscribe registers a handler for a specific event type
func (bus *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	if _, exists := bus.subscribers[eventType]; !exists {
		bus.subscribers[eventType] = make([]EventHandler, 0)
	}

	bus.subscribers[eventType] = append(bus.subscribers[eventType], handler)
	log.Printf("Event handler subscribed to: %s (total: %d handlers)", eventType, len(bus.subscribers[eventType]))
}

// SubscribeMultiple subscribes a handler to multiple event types
func (bus *EventBus) SubscribeMultiple(eventTypes []EventType, handler EventHandler) {
	for _, eventType := range eventTypes {
		bus.Subscribe(eventType, handler)
	}
}

// Unsubscribe removes all handlers for an event type
func (bus *EventBus) Unsubscribe(eventType EventType) {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	delete(bus.subscribers, eventType)
	log.Printf("All handlers unsubscribed from: %s", eventType)
}

// Publish sends an event to all registered handlers
func (bus *EventBus) Publish(ctx context.Context, event Event) error {
	// Ensure event has ID and timestamp
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	bus.mutex.RLock()
	handlers, exists := bus.subscribers[event.Type]
	bus.mutex.RUnlock()

	if !exists || len(handlers) == 0 {
		log.Printf("No subscribers for event type: %s", event.Type)
		return nil
	}

	log.Printf("Publishing event: %s (ID: %s) to %d handlers", event.Type, event.ID, len(handlers))

	if bus.async {
		// Async: fire and forget
		for _, handler := range handlers {
			h := handler // capture for goroutine
			go func() {
				if err := h(ctx, event); err != nil {
					bus.logError(fmt.Errorf("async handler error for event %s: %w", event.Type, err))
					log.Printf("Error handling event %s: %v", event.Type, err)
				}
			}()
		}
		return nil
	}

	// Sync: wait for all handlers
	var errors []string
	for i, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			errorMsg := fmt.Sprintf("handler %d failed: %v", i, err)
			errors = append(errors, errorMsg)
			bus.logError(fmt.Errorf("sync handler error for event %s: %w", event.Type, err))
			log.Printf("Error handling event %s (handler %d): %v", event.Type, i, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("event handling errors: %v", errors)
	}
	return nil
}

// PublishAndWait publishes an event and waits for all handlers to complete (async mode)
func (bus *EventBus) PublishAndWait(ctx context.Context, event Event) error {
	if !bus.async {
		// If sync mode, just use regular Publish
		return bus.Publish(ctx, event)
	}

	// Ensure event has ID and timestamp
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	bus.mutex.RLock()
	handlers, exists := bus.subscribers[event.Type]
	bus.mutex.RUnlock()

	if !exists || len(handlers) == 0 {
		log.Printf("No subscribers for event type: %s", event.Type)
		return nil
	}

	log.Printf("Publishing event (wait): %s (ID: %s) to %d handlers", event.Type, event.ID, len(handlers))

	var wg sync.WaitGroup
	errorsChan := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		h := handler // capture for goroutine
		go func() {
			defer wg.Done()
			if err := h(ctx, event); err != nil {
				errorsChan <- fmt.Errorf("handler error for event %s: %w", event.Type, err)
			}
		}()
	}

	wg.Wait()
	close(errorsChan)

	var errors []string
	for err := range errorsChan {
		errors = append(errors, err.Error())
		bus.logError(err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("event handling errors: %v", errors)
	}
	return nil
}

// GetSubscriberCount returns the number of subscribers for an event type
func (bus *EventBus) GetSubscriberCount(eventType EventType) int {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()
	return len(bus.subscribers[eventType])
}

// GetTotalSubscribers returns total number of subscriptions
func (bus *EventBus) GetTotalSubscribers() int {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()

	total := 0
	for _, handlers := range bus.subscribers {
		total += len(handlers)
	}
	return total
}

// GetSubscribedEvents returns all event types with subscribers
func (bus *EventBus) GetSubscribedEvents() []EventType {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()

	events := make([]EventType, 0, len(bus.subscribers))
	for eventType := range bus.subscribers {
		events = append(events, eventType)
	}
	return events
}

// logError stores errors for debugging
func (bus *EventBus) logError(err error) {
	bus.errorMutex.Lock()
	defer bus.errorMutex.Unlock()
	bus.errorLog = append(bus.errorLog, err)

	// Keep only last 100 errors
	if len(bus.errorLog) > 100 {
		bus.errorLog = bus.errorLog[len(bus.errorLog)-100:]
	}
}

// GetErrors returns recent errors
func (bus *EventBus) GetErrors() []error {
	bus.errorMutex.Lock()
	defer bus.errorMutex.Unlock()
	return append([]error{}, bus.errorLog...)
}

// ClearErrors clears the error log
func (bus *EventBus) ClearErrors() {
	bus.errorMutex.Lock()
	defer bus.errorMutex.Unlock()
	bus.errorLog = make([]error, 0)
}

// IsAsync returns whether the bus is in async mode
func (bus *EventBus) IsAsync() bool {
	return bus.async
}
