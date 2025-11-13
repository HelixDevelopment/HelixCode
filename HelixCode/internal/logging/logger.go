package logging

import (
	"fmt"
	"log"
	"os"
)

// Logger provides structured logging functionality
type Logger struct {
	level  LogLevel
	logger *log.Logger
	name   string
}

// LogLevel represents the severity level of log messages
type LogLevel int

const (
	// DEBUG level for detailed debugging information
	DEBUG LogLevel = iota
	// INFO level for general information
	INFO
	// WARN level for warning messages
	WARN
	// ERROR level for error messages
	ERROR
	// FATAL level for fatal errors that cause program exit
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// NewLogger creates a new logger instance
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// NewLoggerWithName creates a new logger instance with a specific name
func NewLoggerWithName(name string) *Logger {
	return &Logger{
		level:  INFO,
		logger: log.New(os.Stdout, "["+name+"] ", log.LstdFlags),
		name:   name,
	}
}

// DefaultLogger returns a logger with INFO level
func DefaultLogger() *Logger {
	return NewLogger(INFO)
}

// NewTestLogger creates a new logger instance for testing
func NewTestLogger(name string) *Logger {
	return NewLoggerWithName(name)
}

// GetName returns the logger name
func (l *Logger) GetName() string {
	return l.name
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= DEBUG {
		l.log("DEBUG", format, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= INFO {
		l.log("INFO", format, args...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= WARN {
		l.log("WARN", format, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= ERROR {
		l.log("ERROR", format, args...)
	}
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(format string, args ...interface{}) {
	if l.level <= FATAL {
		l.log("FATAL", format, args...)
		os.Exit(1)
	}
}

// log is the internal logging method
func (l *Logger) log(level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s] %s", level, message)
}

// Global logger instance
var defaultLogger = DefaultLogger()

// Debug logs a debug message using the default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs an info message using the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs a warning message using the default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs a fatal message using the default logger and exits
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}
