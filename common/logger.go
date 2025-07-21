package common

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// Color struct to hold ANSI color codes
type Color struct {
	// Reset ANSI color code
	Reset string
	// Red ANSI color code
	Red string
	// Green ANSI color code
	Green string
	// Yellow ANSI color code
	Yellow string
	// Blue ANSI color code
	Blue string
	// Orange ANSI color code
	Orange string
	// Purple ANSI color code
	Purple string
	// Cyan ANSI color code
	Cyan string
}

// Colors instance to access color codes
var Colors = Color{
	Reset:  "\033[0m",
	Red:    "\033[31m",
	Green:  "\033[32m",
	Yellow: "\033[33m",
	Blue:   "\033[34m",
	Orange: "\033[38;5;208m",
	Purple: "\033[35m",
	Cyan:   "\033[36m",
}

// ColorizeString returns a string with the specified color
// Use the Colors struct to access the color codes
// Example: ColorizeString(Colors.Red, "This is red")
func ColorizeString(color, message string) string {
	return color + message + Colors.Reset
}

// TestLogger struct to hold the logger configuration
type TestLogger struct {
	logger          *log.Logger
	testName        string
	prefix          string
	includeDateTime bool
	quietMode       bool
}

// NewTestLogger initializes the custom logger
func NewTestLogger(testName string) *TestLogger {
	return NewTestLoggerWithPrefix(testName, "")
}

// NewTestLoggerWithPrefix initializes the custom logger with a prefix
func NewTestLoggerWithPrefix(testName, prefix string) *TestLogger {
	return &TestLogger{
		logger:   log.New(os.Stdout, "", 0), // No flags by default
		testName: testName,
		prefix:   prefix,
	}
}

// NewTestLoggerWithQuietMode initializes the custom logger with quiet mode setting
func NewTestLoggerWithQuietMode(testName string, quietMode bool) *TestLogger {
	logger := NewTestLogger(testName)
	logger.SetQuietMode(quietMode)
	return logger
}

// NewTestLoggerWithPrefixAndQuietMode initializes the custom logger with prefix and quiet mode setting
func NewTestLoggerWithPrefixAndQuietMode(testName, prefix string, quietMode bool) *TestLogger {
	logger := NewTestLoggerWithPrefix(testName, prefix)
	logger.SetQuietMode(quietMode)
	return logger
}

// NewTestLoggerFromParent creates a new logger that inherits quiet mode from parent logger
func NewTestLoggerFromParent(testName string, parentLogger *TestLogger) *TestLogger {
	logger := NewTestLogger(testName)
	if parentLogger != nil {
		logger.SetQuietMode(parentLogger.IsQuietMode())
		logger.EnableDateTime(parentLogger.includeDateTime)
	}
	return logger
}

// NewTestLoggerFromParentWithPrefix creates a new logger with prefix that inherits quiet mode from parent
func NewTestLoggerFromParentWithPrefix(testName, prefix string, parentLogger *TestLogger) *TestLogger {
	logger := NewTestLoggerWithPrefix(testName, prefix)
	if parentLogger != nil {
		logger.SetQuietMode(parentLogger.IsQuietMode())
		logger.EnableDateTime(parentLogger.includeDateTime)
	}
	return logger
}

// SetPrefix sets the prefix for the logger
func (t *TestLogger) SetPrefix(prefix string) {
	t.prefix = prefix
}

// EnableDateTime enables or disables the date and time stamp in the log
func (t *TestLogger) EnableDateTime(enable bool) {
	t.includeDateTime = enable
	if enable {
		t.logger.SetFlags(log.LstdFlags)
	} else {
		t.logger.SetFlags(0)
	}
}

// SetQuietMode enables or disables quiet mode for the logger
func (t *TestLogger) SetQuietMode(quiet bool) {
	t.quietMode = quiet
}

// IsQuietMode returns whether the logger is in quiet mode
func (t *TestLogger) IsQuietMode() bool {
	return t.quietMode
}

// logWithCaller logs a message with the caller's file and line number
func (t *TestLogger) logWithCaller(level, message, color string) {
	if t.quietMode {
		return // Suppress output in quiet mode
	}
	t.logWithCallerForceOutput(level, message, color)
}

// logWithCallerForceOutput logs a message with the caller's file and line number
// This bypasses quiet mode and always outputs the message (used for errors and critical messages)
func (t *TestLogger) logWithCallerForceOutput(level, message, color string) {
	_, file, line, ok := runtime.Caller(3) // Adjusted caller depth due to extra function layer
	if ok {
		// Extract only the file name from the full path
		file = file[strings.LastIndex(file, "/")+1:]
	}
	coloredLevel := ColorizeString(color, level)
	var coloredPrefix string
	if t.prefix != "" {
		coloredPrefix = ColorizeString(color, fmt.Sprintf("[%s - %s]", t.testName, t.prefix))
	} else {
		coloredPrefix = ColorizeString(color, fmt.Sprintf("[%s]", t.testName))
	}
	if ok {
		t.logger.Printf("%s: %s %s:%d %s", coloredLevel, coloredPrefix, file, line, message)
	} else {
		t.logger.Printf("%s: %s %s", coloredLevel, coloredPrefix, message)
	}
}

// logWithoutCaller logs a message without the caller's file and line number
func (t *TestLogger) logWithoutCaller(level, message, color string) {
	if t.quietMode {
		return // Suppress output in quiet mode
	}
	t.logWithoutCallerForceOutput(level, message, color)
}

// logWithoutCallerForceOutput logs a message without the caller's file and line number
// This bypasses quiet mode and always outputs the message (used for errors and critical messages)
func (t *TestLogger) logWithoutCallerForceOutput(level, message, color string) {
	coloredLevel := ColorizeString(color, level)
	var coloredPrefix string
	if t.prefix != "" {
		coloredPrefix = ColorizeString(color, fmt.Sprintf("[%s - %s]", t.testName, t.prefix))
	} else {
		coloredPrefix = ColorizeString(color, fmt.Sprintf("[%s]", t.testName))
	}
	if level == "" {
		t.logger.Printf("%s %s", coloredPrefix, message)
	} else {
		t.logger.Printf("%s: %s %s", coloredLevel, coloredPrefix, message)
	}
}

// ShortInfo logs a short info message without caller information or INFO prefix
func (t *TestLogger) ShortInfo(message string) {
	t.logWithoutCaller("", message, Colors.Green)
}

// Info logs an info message
func (t *TestLogger) Info(message string) {
	t.logWithCaller("INFO", message, Colors.Green)
}

// ShortError logs a short error message without caller information or ERROR prefix but message is red
// Error messages always show, even in quiet mode, to ensure debugging information is available
func (t *TestLogger) ShortError(message string) {
	t.logWithoutCallerForceOutput("", message, Colors.Red)
}

// Error logs an error message
// Error messages always show, even in quiet mode, to ensure debugging information is available
func (t *TestLogger) Error(message string) {
	t.logWithCallerForceOutput("ERROR", message, Colors.Red)
}

// ShortWarn logs a short warning message without caller information or WARN prefix
func (t *TestLogger) ShortWarn(message string) {
	t.logWithoutCaller("", message, Colors.Yellow)
}

// ProgressStage logs a progress stage message that bypasses quiet mode for high-level test feedback
func (t *TestLogger) ProgressStage(stage string) {
	// Progress messages bypass quiet mode to provide essential feedback during long-running operations
	savedQuietMode := t.quietMode
	t.quietMode = false
	t.logWithoutCaller("", fmt.Sprintf("🔄 %s", stage), Colors.Cyan)
	t.quietMode = savedQuietMode
}

// ProgressSuccess logs a success progress message that bypasses quiet mode
func (t *TestLogger) ProgressSuccess(message string) {
	// Progress messages bypass quiet mode to provide essential feedback
	savedQuietMode := t.quietMode
	t.quietMode = false
	t.logWithoutCaller("", fmt.Sprintf("✅ %s", message), Colors.Green)
	t.quietMode = savedQuietMode
}

// ProgressInfo logs an informational progress message that bypasses quiet mode
func (t *TestLogger) ProgressInfo(message string) {
	// Progress messages bypass quiet mode to provide essential feedback
	savedQuietMode := t.quietMode
	t.quietMode = false
	t.logWithoutCaller("", fmt.Sprintf("ℹ️  %s", message), Colors.Blue)
	t.quietMode = savedQuietMode
}

// Warn logs a warning message
func (t *TestLogger) Warn(message string) {
	t.logWithCaller("WARN", message, Colors.Yellow)
}

// ShortDebug logs a short debug message without caller information or DEBUG prefix
func (t *TestLogger) ShortDebug(message string) {
	t.logWithoutCaller("", message, Colors.Blue)
}

// Debug logs a debug message
func (t *TestLogger) Debug(message string) {
	t.logWithCaller("DEBUG", message, Colors.Blue)
}

// Custom logs a message with a custom level name and color
func (t *TestLogger) Custom(level, message, color string) {
	t.logWithCaller(level, message, color)
}

// ShortCustom logs a message with a custom color without caller information or level prefix
func (t *TestLogger) ShortCustom(message, color string) {
	t.logWithoutCaller("", message, color)
}

// LogMessage represents a single log message with metadata
type LogMessage struct {
	Timestamp time.Time
	Level     string
	Message   string
	Color     string
	TestName  string
	Prefix    string
}

// BufferedTestLogger wraps TestLogger to buffer messages and output them conditionally
type BufferedTestLogger struct {
	testLogger *TestLogger
	buffer     []LogMessage
	quietMode  bool
	failed     bool
}

// NewBufferedTestLogger creates a new buffered logger
func NewBufferedTestLogger(testName string, quietMode bool) *BufferedTestLogger {
	return NewBufferedTestLoggerWithPrefix(testName, "", quietMode)
}

// NewBufferedTestLoggerWithPrefix creates a new buffered logger with a prefix
func NewBufferedTestLoggerWithPrefix(testName, prefix string, quietMode bool) *BufferedTestLogger {
	return &BufferedTestLogger{
		testLogger: NewTestLoggerWithPrefix(testName, prefix),
		buffer:     make([]LogMessage, 0),
		quietMode:  quietMode,
		failed:     false,
	}
}

// SetPrefix sets the prefix for the underlying logger
func (b *BufferedTestLogger) SetPrefix(prefix string) {
	b.testLogger.SetPrefix(prefix)
}

// EnableDateTime enables or disables the date and time stamp in the log
func (b *BufferedTestLogger) EnableDateTime(enable bool) {
	b.testLogger.EnableDateTime(enable)
}

// MarkFailed marks the test as failed, which will cause buffered logs to be output
func (b *BufferedTestLogger) MarkFailed() {
	b.failed = true
}

// IsQuietMode returns whether the logger is in quiet mode
func (b *BufferedTestLogger) IsQuietMode() bool {
	return b.quietMode
}

// SetQuietMode enables or disables quiet mode
func (b *BufferedTestLogger) SetQuietMode(quiet bool) {
	b.quietMode = quiet
}

// addToBuffer adds a message to the buffer
func (b *BufferedTestLogger) addToBuffer(level, message, color string) {
	logMsg := LogMessage{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Color:     color,
		TestName:  b.testLogger.testName,
		Prefix:    b.testLogger.prefix,
	}
	b.buffer = append(b.buffer, logMsg)
}

// FlushBuffer outputs all buffered messages using the underlying logger
func (b *BufferedTestLogger) FlushBuffer() {
	if len(b.buffer) == 0 {
		return
	}

	// Output a separator for clarity
	b.testLogger.ShortCustom("=== BUFFERED LOG OUTPUT ===", Colors.Cyan)

	// Create a temporary buffer to capture and replay the logs
	var buf bytes.Buffer
	tempLogger := log.New(&buf, "", 0)
	originalLogger := b.testLogger.logger
	b.testLogger.logger = tempLogger

	for _, logMsg := range b.buffer {
		// Restore the original timestamp context
		if logMsg.Level == "" {
			b.testLogger.logWithoutCaller("", logMsg.Message, logMsg.Color)
		} else {
			b.testLogger.logWithCaller(logMsg.Level, logMsg.Message, logMsg.Color)
		}
	}

	// Restore the original logger and output the buffered content
	b.testLogger.logger = originalLogger
	fmt.Print(buf.String())

	b.testLogger.ShortCustom("=== END BUFFERED LOG OUTPUT ===", Colors.Cyan)
}

// FlushOnFailure outputs buffered messages only if the test failed
func (b *BufferedTestLogger) FlushOnFailure() {
	if b.failed {
		b.FlushBuffer()
	}
}

// ClearBuffer clears the buffer without outputting
func (b *BufferedTestLogger) ClearBuffer() {
	b.buffer = make([]LogMessage, 0)
}

// GetBufferSize returns the number of buffered messages
func (b *BufferedTestLogger) GetBufferSize() int {
	return len(b.buffer)
}

// Immediate output methods (bypass buffering)
func (b *BufferedTestLogger) ImmediateShortInfo(message string) {
	b.testLogger.ShortInfo(message)
}

func (b *BufferedTestLogger) ImmediateShortError(message string) {
	b.testLogger.ShortError(message)
}

func (b *BufferedTestLogger) ImmediateShortWarn(message string) {
	b.testLogger.ShortWarn(message)
}

// Buffered logging methods
func (b *BufferedTestLogger) ShortInfo(message string) {
	if b.quietMode {
		b.addToBuffer("", message, Colors.Green)
	} else {
		b.testLogger.ShortInfo(message)
	}
}

func (b *BufferedTestLogger) Info(message string) {
	if b.quietMode {
		b.addToBuffer("INFO", message, Colors.Green)
	} else {
		b.testLogger.Info(message)
	}
}

func (b *BufferedTestLogger) ShortError(message string) {
	if b.quietMode {
		b.addToBuffer("", message, Colors.Red)
	} else {
		b.testLogger.ShortError(message)
	}
}

func (b *BufferedTestLogger) Error(message string) {
	if b.quietMode {
		b.addToBuffer("ERROR", message, Colors.Red)
	} else {
		b.testLogger.Error(message)
	}
}

func (b *BufferedTestLogger) ShortWarn(message string) {
	if b.quietMode {
		b.addToBuffer("", message, Colors.Yellow)
	} else {
		b.testLogger.ShortWarn(message)
	}
}

func (b *BufferedTestLogger) Warn(message string) {
	if b.quietMode {
		b.addToBuffer("WARN", message, Colors.Yellow)
	} else {
		b.testLogger.Warn(message)
	}
}

func (b *BufferedTestLogger) ShortDebug(message string) {
	if b.quietMode {
		b.addToBuffer("", message, Colors.Blue)
	} else {
		b.testLogger.ShortDebug(message)
	}
}

func (b *BufferedTestLogger) Debug(message string) {
	if b.quietMode {
		b.addToBuffer("DEBUG", message, Colors.Blue)
	} else {
		b.testLogger.Debug(message)
	}
}

func (b *BufferedTestLogger) Custom(level, message, color string) {
	if b.quietMode {
		b.addToBuffer(level, message, color)
	} else {
		b.testLogger.Custom(level, message, color)
	}
}

func (b *BufferedTestLogger) ShortCustom(message, color string) {
	if b.quietMode {
		b.addToBuffer("", message, color)
	} else {
		b.testLogger.ShortCustom(message, color)
	}
}
