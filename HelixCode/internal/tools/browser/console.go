package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// ConsoleMessageType defines console message types
type ConsoleMessageType int

const (
	ConsoleLog ConsoleMessageType = iota
	ConsoleInfo
	ConsoleWarning
	ConsoleError
	ConsoleDebug
)

// String returns the string representation of ConsoleMessageType
func (t ConsoleMessageType) String() string {
	switch t {
	case ConsoleLog:
		return "log"
	case ConsoleInfo:
		return "info"
	case ConsoleWarning:
		return "warning"
	case ConsoleError:
		return "error"
	case ConsoleDebug:
		return "debug"
	default:
		return "unknown"
	}
}

// ConsoleMessage represents a console message
type ConsoleMessage struct {
	Type       ConsoleMessageType
	Text       string
	URL        string
	Line       int
	Column     int
	Timestamp  time.Time
	Args       []interface{}
	StackTrace string
}

// ConsoleMonitor monitors browser console messages
type ConsoleMonitor struct {
	messages     chan *ConsoleMessage
	errors       chan *ConsoleMessage
	messageLog   []*ConsoleMessage
	errorLog     []*ConsoleMessage
	maxLogSize   int
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	filterErrors bool
}

// ConsoleMonitorOptions configures console monitoring
type ConsoleMonitorOptions struct {
	MaxLogSize   int  // Maximum number of messages to keep in memory
	FilterErrors bool // Whether to filter errors into separate channel
	BufferSize   int  // Channel buffer size
}

// DefaultConsoleMonitorOptions returns default console monitor options
func DefaultConsoleMonitorOptions() *ConsoleMonitorOptions {
	return &ConsoleMonitorOptions{
		MaxLogSize:   1000,
		FilterErrors: true,
		BufferSize:   100,
	}
}

// NewConsoleMonitor creates a new console monitor
func NewConsoleMonitor(opts *ConsoleMonitorOptions) *ConsoleMonitor {
	if opts == nil {
		opts = DefaultConsoleMonitorOptions()
	}

	return &ConsoleMonitor{
		messages:     make(chan *ConsoleMessage, opts.BufferSize),
		errors:       make(chan *ConsoleMessage, opts.BufferSize),
		messageLog:   make([]*ConsoleMessage, 0, opts.MaxLogSize),
		errorLog:     make([]*ConsoleMessage, 0, opts.MaxLogSize),
		maxLogSize:   opts.MaxLogSize,
		filterErrors: opts.FilterErrors,
	}
}

// Start starts monitoring console in the given browser context
func (cm *ConsoleMonitor) Start(browserCtx context.Context) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Create a cancellable context
	cm.ctx, cm.cancel = context.WithCancel(browserCtx)

	// Listen for console API calls
	chromedp.ListenTarget(cm.ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			cm.handleConsoleAPI(ev)

		case *runtime.EventExceptionThrown:
			cm.handleException(ev)
		}
	})
}

// Stop stops monitoring console
func (cm *ConsoleMonitor) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.cancel != nil {
		cm.cancel()
		cm.cancel = nil
	}

	// Close channels
	close(cm.messages)
	close(cm.errors)
}

// GetMessages returns the messages channel
func (cm *ConsoleMonitor) GetMessages() <-chan *ConsoleMessage {
	return cm.messages
}

// GetErrors returns the errors channel
func (cm *ConsoleMonitor) GetErrors() <-chan *ConsoleMessage {
	return cm.errors
}

// GetMessageLog returns all logged messages
func (cm *ConsoleMonitor) GetMessageLog() []*ConsoleMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy
	log := make([]*ConsoleMessage, len(cm.messageLog))
	copy(log, cm.messageLog)
	return log
}

// GetErrorLog returns all logged errors
func (cm *ConsoleMonitor) GetErrorLog() []*ConsoleMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy
	log := make([]*ConsoleMessage, len(cm.errorLog))
	copy(log, cm.errorLog)
	return log
}

// ClearLogs clears all logged messages and errors
func (cm *ConsoleMonitor) ClearLogs() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.messageLog = make([]*ConsoleMessage, 0, cm.maxLogSize)
	cm.errorLog = make([]*ConsoleMessage, 0, cm.maxLogSize)
}

// handleConsoleAPI handles console API events
func (cm *ConsoleMonitor) handleConsoleAPI(ev *runtime.EventConsoleAPICalled) {
	msg := &ConsoleMessage{
		Type:      cm.mapConsoleType(ev.Type),
		Timestamp: time.Now(),
		Args:      make([]interface{}, 0, len(ev.Args)),
	}

	// Extract arguments
	for _, arg := range ev.Args {
		if arg.Value != nil {
			msg.Args = append(msg.Args, arg.Value)
		} else if arg.Description != "" {
			msg.Args = append(msg.Args, arg.Description)
		}
	}

	// Build text from arguments
	if len(ev.Args) > 0 {
		if ev.Args[0].Description != "" {
			msg.Text = ev.Args[0].Description
		} else if ev.Args[0].Value != nil {
			msg.Text = fmt.Sprintf("%v", ev.Args[0].Value)
		}
	}

	// Get stack trace if available
	if ev.StackTrace != nil {
		msg.StackTrace = cm.formatStackTrace(ev.StackTrace)
	}

	cm.addMessage(msg)
}

// handleException handles exception events
func (cm *ConsoleMonitor) handleException(ev *runtime.EventExceptionThrown) {
	msg := &ConsoleMessage{
		Type:      ConsoleError,
		Text:      ev.ExceptionDetails.Text,
		Timestamp: time.Now(),
	}

	if ev.ExceptionDetails.URL != "" {
		msg.URL = ev.ExceptionDetails.URL
	}
	msg.Line = int(ev.ExceptionDetails.LineNumber)
	msg.Column = int(ev.ExceptionDetails.ColumnNumber)

	// Get stack trace if available
	if ev.ExceptionDetails.StackTrace != nil {
		msg.StackTrace = cm.formatStackTrace(ev.ExceptionDetails.StackTrace)
	}

	// Add exception details to text
	if ev.ExceptionDetails.Exception != nil {
		if ev.ExceptionDetails.Exception.Description != "" {
			msg.Text = ev.ExceptionDetails.Exception.Description
		}
	}

	cm.addMessage(msg)
}

// addMessage adds a message to the log and channels
func (cm *ConsoleMonitor) addMessage(msg *ConsoleMessage) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Add to message log
	cm.messageLog = append(cm.messageLog, msg)
	if len(cm.messageLog) > cm.maxLogSize {
		// Remove oldest message
		cm.messageLog = cm.messageLog[1:]
	}

	// Add to error log if it's an error
	if msg.Type == ConsoleError {
		cm.errorLog = append(cm.errorLog, msg)
		if len(cm.errorLog) > cm.maxLogSize {
			// Remove oldest error
			cm.errorLog = cm.errorLog[1:]
		}
	}

	// Send to channels (non-blocking)
	select {
	case cm.messages <- msg:
	default:
		// Channel full, skip
	}

	if msg.Type == ConsoleError && cm.filterErrors {
		select {
		case cm.errors <- msg:
		default:
			// Channel full, skip
		}
	}
}

// mapConsoleType maps CDP console type to our type
func (cm *ConsoleMonitor) mapConsoleType(cdpType runtime.APIType) ConsoleMessageType {
	switch cdpType {
	case runtime.APITypeLog:
		return ConsoleLog
	case runtime.APITypeInfo:
		return ConsoleInfo
	case runtime.APITypeWarning:
		return ConsoleWarning
	case runtime.APITypeError:
		return ConsoleError
	case runtime.APITypeDebug:
		return ConsoleDebug
	default:
		return ConsoleLog
	}
}

// formatStackTrace formats a stack trace for display
func (cm *ConsoleMonitor) formatStackTrace(stackTrace *runtime.StackTrace) string {
	if stackTrace == nil || len(stackTrace.CallFrames) == 0 {
		return ""
	}

	var result string
	for i, frame := range stackTrace.CallFrames {
		if i > 0 {
			result += "\n"
		}
		result += fmt.Sprintf("  at %s (%s:%d:%d)",
			frame.FunctionName,
			frame.URL,
			frame.LineNumber,
			frame.ColumnNumber,
		)
	}

	return result
}

// ConsoleLogger provides a simple interface to log console messages
type ConsoleLogger struct {
	monitor *ConsoleMonitor
	prefix  string
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger(monitor *ConsoleMonitor, prefix string) *ConsoleLogger {
	return &ConsoleLogger{
		monitor: monitor,
		prefix:  prefix,
	}
}

// LogMessages starts logging messages to stdout
func (cl *ConsoleLogger) LogMessages(ctx context.Context) {
	go func() {
		for {
			select {
			case msg, ok := <-cl.monitor.GetMessages():
				if !ok {
					return
				}
				cl.printMessage(msg)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// LogErrors starts logging errors to stderr
func (cl *ConsoleLogger) LogErrors(ctx context.Context) {
	go func() {
		for {
			select {
			case err, ok := <-cl.monitor.GetErrors():
				if !ok {
					return
				}
				cl.printError(err)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// printMessage prints a console message
func (cl *ConsoleLogger) printMessage(msg *ConsoleMessage) {
	timestamp := msg.Timestamp.Format("15:04:05.000")
	prefix := cl.prefix
	if prefix != "" {
		prefix = prefix + " "
	}

	fmt.Printf("[%s] %s[%s] %s\n", timestamp, prefix, msg.Type, msg.Text)

	if msg.StackTrace != "" {
		fmt.Printf("%s\n", msg.StackTrace)
	}
}

// printError prints a console error
func (cl *ConsoleLogger) printError(msg *ConsoleMessage) {
	timestamp := msg.Timestamp.Format("15:04:05.000")
	prefix := cl.prefix
	if prefix != "" {
		prefix = prefix + " "
	}

	location := ""
	if msg.URL != "" {
		location = fmt.Sprintf(" at %s:%d:%d", msg.URL, msg.Line, msg.Column)
	}

	fmt.Printf("[%s] %s[ERROR]%s %s\n", timestamp, prefix, location, msg.Text)

	if msg.StackTrace != "" {
		fmt.Printf("%s\n", msg.StackTrace)
	}
}

// GetErrorCount returns the number of errors logged
func (cm *ConsoleMonitor) GetErrorCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.errorLog)
}

// GetMessageCount returns the number of messages logged
func (cm *ConsoleMonitor) GetMessageCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.messageLog)
}

// HasErrors returns true if any errors have been logged
func (cm *ConsoleMonitor) HasErrors() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.errorLog) > 0
}
