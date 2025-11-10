package logging

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

// ========================================
// LogLevel Tests
// ========================================

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLogLevel_Constants(t *testing.T) {
	// Verify ordering
	if DEBUG >= INFO {
		t.Error("DEBUG should be less than INFO")
	}
	if INFO >= WARN {
		t.Error("INFO should be less than WARN")
	}
	if WARN >= ERROR {
		t.Error("WARN should be less than ERROR")
	}
	if ERROR >= FATAL {
		t.Error("ERROR should be less than FATAL")
	}
}

// ========================================
// Constructor Tests
// ========================================

func TestNewLogger(t *testing.T) {
	logger := NewLogger(DEBUG)
	if logger == nil {
		t.Fatal("Expected logger, got nil")
	}
	if logger.level != DEBUG {
		t.Errorf("Expected level DEBUG, got %v", logger.level)
	}
	if logger.logger == nil {
		t.Error("Expected internal logger to be initialized")
	}
}

func TestNewLogger_DifferentLevels(t *testing.T) {
	levels := []LogLevel{DEBUG, INFO, WARN, ERROR, FATAL}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			logger := NewLogger(level)
			if logger.level != level {
				t.Errorf("Expected level %v, got %v", level, logger.level)
			}
		})
	}
}

func TestNewLoggerWithName(t *testing.T) {
	logger := NewLoggerWithName("test-logger")
	if logger == nil {
		t.Fatal("Expected logger, got nil")
	}
	if logger.level != INFO {
		t.Errorf("Expected default level INFO, got %v", logger.level)
	}
	if logger.logger == nil {
		t.Error("Expected internal logger to be initialized")
	}
}

func TestDefaultLogger(t *testing.T) {
	logger := DefaultLogger()
	if logger == nil {
		t.Fatal("Expected logger, got nil")
	}
	if logger.level != INFO {
		t.Errorf("Expected level INFO, got %v", logger.level)
	}
}

func TestNewTestLogger(t *testing.T) {
	logger := NewTestLogger("test")
	if logger == nil {
		t.Fatal("Expected logger, got nil")
	}
	if logger.level != INFO {
		t.Errorf("Expected level INFO, got %v", logger.level)
	}
}

// ========================================
// Logging Method Tests
// ========================================

func captureOutput(f func()) string {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DEBUG,
		logger: log.New(&buf, "", 0),
	}

	logger.Debug("test debug message")

	output := buf.String()
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("Expected output to contain [DEBUG]")
	}
	if !strings.Contains(output, "test debug message") {
		t.Error("Expected output to contain message")
	}
}

func TestLogger_Debug_FilteredByLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  INFO, // INFO level should filter out DEBUG
		logger: log.New(&buf, "", 0),
	}

	logger.Debug("this should not appear")

	output := buf.String()
	if output != "" {
		t.Error("Expected no output when level filters debug")
	}
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  INFO,
		logger: log.New(&buf, "", 0),
	}

	logger.Info("test info message")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Error("Expected output to contain [INFO]")
	}
	if !strings.Contains(output, "test info message") {
		t.Error("Expected output to contain message")
	}
}

func TestLogger_Info_FilteredByLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  WARN, // WARN level should filter out INFO
		logger: log.New(&buf, "", 0),
	}

	logger.Info("this should not appear")

	output := buf.String()
	if output != "" {
		t.Error("Expected no output when level filters info")
	}
}

func TestLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  WARN,
		logger: log.New(&buf, "", 0),
	}

	logger.Warn("test warn message")

	output := buf.String()
	if !strings.Contains(output, "[WARN]") {
		t.Error("Expected output to contain [WARN]")
	}
	if !strings.Contains(output, "test warn message") {
		t.Error("Expected output to contain message")
	}
}

func TestLogger_Warn_FilteredByLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  ERROR, // ERROR level should filter out WARN
		logger: log.New(&buf, "", 0),
	}

	logger.Warn("this should not appear")

	output := buf.String()
	if output != "" {
		t.Error("Expected no output when level filters warn")
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  ERROR,
		logger: log.New(&buf, "", 0),
	}

	logger.Error("test error message")

	output := buf.String()
	if !strings.Contains(output, "[ERROR]") {
		t.Error("Expected output to contain [ERROR]")
	}
	if !strings.Contains(output, "test error message") {
		t.Error("Expected output to contain message")
	}
}

func TestLogger_Error_FilteredByLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  FATAL, // FATAL level should filter out ERROR
		logger: log.New(&buf, "", 0),
	}

	logger.Error("this should not appear")

	output := buf.String()
	if output != "" {
		t.Error("Expected no output when level filters error")
	}
}

// Note: We cannot easily test Fatal() as it calls os.Exit(1)
// which would terminate the test process. In a real scenario,
// you would use dependency injection or mocking to test this.

// ========================================
// Formatting Tests
// ========================================

func TestLogger_InfoWithFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  INFO,
		logger: log.New(&buf, "", 0),
	}

	logger.Info("formatted message: %s = %d", "value", 42)

	output := buf.String()
	if !strings.Contains(output, "formatted message: value = 42") {
		t.Errorf("Expected formatted message, got: %s", output)
	}
}

func TestLogger_DebugWithMultipleArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DEBUG,
		logger: log.New(&buf, "", 0),
	}

	logger.Debug("test %s %d %v", "string", 123, true)

	output := buf.String()
	if !strings.Contains(output, "test string 123 true") {
		t.Errorf("Expected formatted message, got: %s", output)
	}
}

// ========================================
// Global Logger Function Tests
// ========================================

func TestGlobalDebug(t *testing.T) {
	// Reset global logger to DEBUG level
	var buf bytes.Buffer
	defaultLogger = &Logger{
		level:  DEBUG,
		logger: log.New(&buf, "", 0),
	}

	Debug("global debug test")

	output := buf.String()
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("Expected global debug to work")
	}
	if !strings.Contains(output, "global debug test") {
		t.Error("Expected message in output")
	}
}

func TestGlobalInfo(t *testing.T) {
	var buf bytes.Buffer
	defaultLogger = &Logger{
		level:  INFO,
		logger: log.New(&buf, "", 0),
	}

	Info("global info test")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Error("Expected global info to work")
	}
	if !strings.Contains(output, "global info test") {
		t.Error("Expected message in output")
	}
}

func TestGlobalWarn(t *testing.T) {
	var buf bytes.Buffer
	defaultLogger = &Logger{
		level:  WARN,
		logger: log.New(&buf, "", 0),
	}

	Warn("global warn test")

	output := buf.String()
	if !strings.Contains(output, "[WARN]") {
		t.Error("Expected global warn to work")
	}
	if !strings.Contains(output, "global warn test") {
		t.Error("Expected message in output")
	}
}

func TestGlobalError(t *testing.T) {
	var buf bytes.Buffer
	defaultLogger = &Logger{
		level:  ERROR,
		logger: log.New(&buf, "", 0),
	}

	Error("global error test")

	output := buf.String()
	if !strings.Contains(output, "[ERROR]") {
		t.Error("Expected global error to work")
	}
	if !strings.Contains(output, "global error test") {
		t.Error("Expected message in output")
	}
}

// ========================================
// Level Filtering Tests
// ========================================

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name         string
		loggerLevel  LogLevel
		shouldLog    []string
		shouldNotLog []string
	}{
		{
			name:         "DEBUG level logs everything",
			loggerLevel:  DEBUG,
			shouldLog:    []string{"DEBUG", "INFO", "WARN", "ERROR"},
			shouldNotLog: []string{},
		},
		{
			name:         "INFO level filters DEBUG",
			loggerLevel:  INFO,
			shouldLog:    []string{"INFO", "WARN", "ERROR"},
			shouldNotLog: []string{"DEBUG"},
		},
		{
			name:         "WARN level filters DEBUG and INFO",
			loggerLevel:  WARN,
			shouldLog:    []string{"WARN", "ERROR"},
			shouldNotLog: []string{"DEBUG", "INFO"},
		},
		{
			name:         "ERROR level filters DEBUG, INFO, WARN",
			loggerLevel:  ERROR,
			shouldLog:    []string{"ERROR"},
			shouldNotLog: []string{"DEBUG", "INFO", "WARN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{
				level:  tt.loggerLevel,
				logger: log.New(&buf, "", 0),
			}

			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			output := buf.String()

			for _, level := range tt.shouldLog {
				if !strings.Contains(output, "["+level+"]") {
					t.Errorf("Expected output to contain [%s]", level)
				}
			}

			for _, level := range tt.shouldNotLog {
				if strings.Contains(output, "["+level+"]") {
					t.Errorf("Expected output to NOT contain [%s]", level)
				}
			}
		})
	}
}

// ========================================
// Edge Cases
// ========================================

func TestLogger_EmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  INFO,
		logger: log.New(&buf, "", 0),
	}

	logger.Info("")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Error("Expected [INFO] even with empty message")
	}
}

func TestLogger_NoFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  INFO,
		logger: log.New(&buf, "", 0),
	}

	logger.Info("no format specifiers")

	output := buf.String()
	if !strings.Contains(output, "no format specifiers") {
		t.Error("Expected plain message without format args")
	}
}

func TestLogger_SpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  INFO,
		logger: log.New(&buf, "", 0),
	}

	logger.Info("message with special chars: !@#$&*()")

	output := buf.String()
	if !strings.Contains(output, "!@#$&*()") {
		t.Error("Expected special characters to be preserved")
	}
}

func TestLogger_Newlines(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  INFO,
		logger: log.New(&buf, "", 0),
	}

	logger.Info("message\nwith\nnewlines")

	output := buf.String()
	if !strings.Contains(output, "message\nwith\nnewlines") {
		t.Error("Expected newlines to be preserved")
	}
}
