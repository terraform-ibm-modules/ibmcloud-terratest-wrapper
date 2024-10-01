package common

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
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

// logWithCaller logs a message with the caller's file and line number
func (t *TestLogger) logWithCaller(level, message, color string) {
	_, file, line, ok := runtime.Caller(2)
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
func (t *TestLogger) ShortError(message string) {
	t.logWithoutCaller("", message, Colors.Red)
}

// Error logs an error message
func (t *TestLogger) Error(message string) {
	t.logWithCaller("ERROR", message, Colors.Red)
}

// ShortDebug logs a short debug message without caller information or DEBUG prefix
func (t *TestLogger) ShortDebug(message string) {
	t.logWithoutCaller("", message, Colors.Yellow)
}

// Debug logs a debug message
func (t *TestLogger) Debug(message string) {
	t.logWithCaller("DEBUG", message, Colors.Yellow)
}

// Custom logs a message with a custom level name and color
func (t *TestLogger) Custom(level, message, color string) {
	t.logWithCaller(level, message, color)
}

// ShortCustom logs a message with a custom color without caller information or level prefix
func (t *TestLogger) ShortCustom(message, color string) {
	t.logWithoutCaller("", message, color)
}
