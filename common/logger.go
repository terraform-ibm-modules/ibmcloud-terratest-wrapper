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

// Logger interface defines the common logging methods that both TestLogger and BufferedTestLogger implement
// This allows for universal logger usage without type-specific code
type Logger interface {
	// Basic logging methods
	ShortInfo(message string)
	Info(message string)
	ShortError(message string)
	Error(message string)
	ShortWarn(message string)
	Warn(message string)
	ShortDebug(message string)
	Debug(message string)

	// Custom logging methods
	Custom(level, message, color string)
	ShortCustom(message, color string)

	// Progress methods
	ProgressStage(stage string)
	ProgressSuccess(message string)
	ProgressInfo(message string)

	// Configuration methods
	SetPrefix(prefix string)
	EnableDateTime(enable bool)
	SetQuietMode(quiet bool)
	IsQuietMode() bool

	// BufferedTestLogger specific methods (no-op for TestLogger)
	MarkFailed()
	FlushBuffer()
	FlushOnFailure()
	ClearBuffer()
	GetBufferSize() int

	// Enhanced error handling methods
	CriticalError(message string)    // Shows buffer context first, then prominent red-bordered error
	FatalError(message string)       // Immediate error display, bypasses buffering
	ErrorWithContext(message string) // Shows buffer context with moderate error formatting

	// Compatibility methods
	GetUnderlyingLogger() *TestLogger
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

// BufferedTestLogger-specific methods (no-op implementations for TestLogger)

// MarkFailed is a no-op for TestLogger (compatibility with Logger interface)
func (t *TestLogger) MarkFailed() {
	// No-op: TestLogger doesn't have buffering, so no marking needed
}

// FlushBuffer is a no-op for TestLogger (compatibility with Logger interface)
func (t *TestLogger) FlushBuffer() {
	// No-op: TestLogger doesn't buffer, so nothing to flush
}

// FlushOnFailure is a no-op for TestLogger (compatibility with Logger interface)
func (t *TestLogger) FlushOnFailure() {
	// No-op: TestLogger doesn't buffer, so nothing to flush
}

// ClearBuffer is a no-op for TestLogger (compatibility with Logger interface)
func (t *TestLogger) ClearBuffer() {
	// No-op: TestLogger doesn't buffer, so nothing to clear
}

// GetBufferSize returns 0 for TestLogger (compatibility with Logger interface)
func (t *TestLogger) GetBufferSize() int {
	return 0 // TestLogger doesn't buffer, so size is always 0
}

// GetUnderlyingLogger returns itself for TestLogger (compatibility with Logger interface)
func (t *TestLogger) GetUnderlyingLogger() *TestLogger {
	return t
}

// Enhanced error handling methods for TestLogger

// CriticalError shows buffer context first, then prominent red-bordered error (no-op for TestLogger since no buffering)
func (t *TestLogger) CriticalError(message string) {
	separator := strings.Repeat("=", 80)
	t.logWithoutCallerForceOutput("", separator, Colors.Red)
	t.logWithoutCallerForceOutput("", fmt.Sprintf("CRITICAL ERROR: %s", message), Colors.Red)
	t.logWithoutCallerForceOutput("", separator, Colors.Red)
}

// FatalError shows immediate error, bypasses buffering
func (t *TestLogger) FatalError(message string) {
	t.logWithoutCallerForceOutput("", fmt.Sprintf("FATAL ERROR: %s", message), Colors.Red)
}

// ErrorWithContext shows buffer context with moderate error formatting (no-op for TestLogger since no buffering)
func (t *TestLogger) ErrorWithContext(message string) {
	separator := strings.Repeat("-", 60)
	t.logWithoutCallerForceOutput("", separator, Colors.Yellow)
	t.logWithoutCallerForceOutput("", fmt.Sprintf("ERROR: %s", message), Colors.Red)
	t.logWithoutCallerForceOutput("", separator, Colors.Yellow)
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

	// Clear the buffer after flushing to avoid re-displaying messages
	b.buffer = make([]LogMessage, 0)
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

// Progress methods - these always output immediately and bypass quiet mode to provide essential feedback
func (b *BufferedTestLogger) ProgressStage(stage string) {
	b.testLogger.ProgressStage(stage)
}

func (b *BufferedTestLogger) ProgressSuccess(message string) {
	b.testLogger.ProgressSuccess(message)
}

func (b *BufferedTestLogger) ProgressInfo(message string) {
	b.testLogger.ProgressInfo(message)
}

// GetUnderlyingLogger returns the internal TestLogger for compatibility with APIs that expect *TestLogger
func (b *BufferedTestLogger) GetUnderlyingLogger() *TestLogger {
	return b.testLogger
}

// Enhanced error handling methods for BufferedTestLogger

// CriticalError shows buffer context first, then prominent red-bordered error
func (b *BufferedTestLogger) CriticalError(message string) {
	b.MarkFailed()
	b.FlushOnFailure()

	separator := strings.Repeat("=", 80)
	b.testLogger.logWithoutCallerForceOutput("", separator, Colors.Red)
	b.testLogger.logWithoutCallerForceOutput("", fmt.Sprintf("CRITICAL ERROR: %s", message), Colors.Red)
	b.testLogger.logWithoutCallerForceOutput("", separator, Colors.Red)
}

// FatalError shows immediate error, bypasses buffering
func (b *BufferedTestLogger) FatalError(message string) {
	b.testLogger.logWithoutCallerForceOutput("", fmt.Sprintf("FATAL ERROR: %s", message), Colors.Red)
}

// ErrorWithContext shows buffer context with moderate error formatting
func (b *BufferedTestLogger) ErrorWithContext(message string) {
	b.MarkFailed()
	b.FlushOnFailure()

	separator := strings.Repeat("-", 60)
	b.testLogger.logWithoutCallerForceOutput("", separator, Colors.Yellow)
	b.testLogger.logWithoutCallerForceOutput("", fmt.Sprintf("ERROR: %s", message), Colors.Red)
	b.testLogger.logWithoutCallerForceOutput("", separator, Colors.Yellow)
}

// LoggerProxy provides automatic smart wrapping of loggers with buffering capabilities
// It analyzes the context and automatically applies buffering when appropriate
type LoggerProxy struct {
	logger Logger
}

// NewLoggerProxy creates a new LoggerProxy that automatically handles buffering based on context
func NewLoggerProxy(logger Logger) *LoggerProxy {
	return &LoggerProxy{logger: logger}
}

// WrapWithBufferingIfNeeded automatically wraps a TestLogger with BufferedTestLogger if quiet mode should be enabled
// This provides seamless auto-buffering without requiring code changes throughout the system
func WrapWithBufferingIfNeeded(logger *TestLogger, quietMode bool) Logger {
	if logger == nil {
		return nil
	}

	if quietMode {
		// Automatically wrap with buffering for quiet mode
		buffered := &BufferedTestLogger{
			testLogger: logger,
			buffer:     make([]LogMessage, 0),
			quietMode:  true,
			failed:     false,
		}
		// Inherit existing settings
		buffered.testLogger.SetQuietMode(false) // BufferedTestLogger handles quiet mode logic
		return buffered
	}

	// Return as-is for non-quiet mode
	return logger
}

// CreateAutoBufferingLogger creates a logger that automatically handles buffering based on quiet mode
func CreateAutoBufferingLogger(testName string, quietMode bool) Logger {
	baseLogger := NewTestLogger(testName)
	return WrapWithBufferingIfNeeded(baseLogger, quietMode)
}

// CreateAutoBufferingLoggerWithPrefix creates a logger with prefix that automatically handles buffering
func CreateAutoBufferingLoggerWithPrefix(testName, prefix string, quietMode bool) Logger {
	baseLogger := NewTestLoggerWithPrefix(testName, prefix)
	return WrapWithBufferingIfNeeded(baseLogger, quietMode)
}

// Delegate all Logger interface methods to the wrapped logger
func (p *LoggerProxy) ShortInfo(message string) {
	p.logger.ShortInfo(message)
}

func (p *LoggerProxy) Info(message string) {
	p.logger.Info(message)
}

func (p *LoggerProxy) ShortError(message string) {
	p.logger.ShortError(message)
}

func (p *LoggerProxy) Error(message string) {
	p.logger.Error(message)
}

func (p *LoggerProxy) ShortWarn(message string) {
	p.logger.ShortWarn(message)
}

func (p *LoggerProxy) Warn(message string) {
	p.logger.Warn(message)
}

func (p *LoggerProxy) ShortDebug(message string) {
	p.logger.ShortDebug(message)
}

func (p *LoggerProxy) Debug(message string) {
	p.logger.Debug(message)
}

func (p *LoggerProxy) Custom(level, message, color string) {
	p.logger.Custom(level, message, color)
}

func (p *LoggerProxy) ShortCustom(message, color string) {
	p.logger.ShortCustom(message, color)
}

func (p *LoggerProxy) ProgressStage(stage string) {
	p.logger.ProgressStage(stage)
}

func (p *LoggerProxy) ProgressSuccess(message string) {
	p.logger.ProgressSuccess(message)
}

func (p *LoggerProxy) ProgressInfo(message string) {
	p.logger.ProgressInfo(message)
}

func (p *LoggerProxy) SetPrefix(prefix string) {
	p.logger.SetPrefix(prefix)
}

func (p *LoggerProxy) EnableDateTime(enable bool) {
	p.logger.EnableDateTime(enable)
}

func (p *LoggerProxy) SetQuietMode(quiet bool) {
	p.logger.SetQuietMode(quiet)
}

func (p *LoggerProxy) IsQuietMode() bool {
	return p.logger.IsQuietMode()
}

func (p *LoggerProxy) MarkFailed() {
	p.logger.MarkFailed()
}

func (p *LoggerProxy) FlushBuffer() {
	p.logger.FlushBuffer()
}

func (p *LoggerProxy) FlushOnFailure() {
	p.logger.FlushOnFailure()
}

func (p *LoggerProxy) ClearBuffer() {
	p.logger.ClearBuffer()
}

func (p *LoggerProxy) GetBufferSize() int {
	return p.logger.GetBufferSize()
}

func (p *LoggerProxy) GetUnderlyingLogger() *TestLogger {
	return p.logger.GetUnderlyingLogger()
}

// Enhanced error handling methods for LoggerProxy

func (p *LoggerProxy) CriticalError(message string) {
	p.logger.CriticalError(message)
}

func (p *LoggerProxy) FatalError(message string) {
	p.logger.FatalError(message)
}

func (p *LoggerProxy) ErrorWithContext(message string) {
	p.logger.ErrorWithContext(message)
}

// PhasePatterns maps log message patterns to progress stage messages
type PhasePatterns map[string]string

// SmartLoggerConfig configures SmartLogger behavior
type SmartLoggerConfig struct {
	PhasePatterns PhasePatterns
}

// SmartLogger provides intelligent automatic phase detection and logging
// It analyzes log messages and automatically provides progress tracking for common operations
type SmartLogger struct {
	logger           Logger
	config           SmartLoggerConfig
	activePhase      string
	batchMode        bool
	suppressedPhases map[string]bool
}

// NewSmartLogger creates a logger with the provided configuration
func NewSmartLogger(logger Logger, config SmartLoggerConfig) *SmartLogger {
	return &SmartLogger{
		logger:           logger,
		config:           config,
		suppressedPhases: make(map[string]bool),
	}
}

// detectPhaseFromMessage analyzes a log message and determines if it represents a known operation phase
func (s *SmartLogger) detectPhaseFromMessage(message string) string {
	for pattern, phase := range s.config.PhasePatterns {
		if strings.Contains(message, pattern) {
			return phase
		}
	}
	return "" // No phase detected
}

// Predefined phase pattern configurations for different test types

// AddonPhasePatterns contains patterns for addon testing
var AddonPhasePatterns = PhasePatterns{
	"Getting offering details":          "🔄 Retrieving catalog information",
	"Getting offering version locator":  "🔄 Resolving version constraints",
	"Starting reference resolution":     "🔄 Resolving project references",
	"Attempting reference resolution":   "🔄 Validating dependencies",
	"Configuration deployed to project": "✅ Configuration deployed to project",
	"Creating catalog":                  "🔄 Setting up catalog",
	"Importing offering":                "🔄 Loading offering configuration",
	"Validating configuration":          "🔄 Validating inputs",
	"Processing configuration details":  "🔄 Processing configuration",
	"Building dependency graph":         "🔄 Analyzing dependencies",
}

// ProjectPhasePatterns contains patterns for project testing
var ProjectPhasePatterns = PhasePatterns{
	"Configuring Test Stack":        "🔄 Configuring stack",
	"Triggering Deploy":             "🔄 Triggering deployment",
	"Deploy Triggered Successfully": "✅ Deployment triggered",
	"Checking Stack Deploy Status":  "🔄 Checking deployment status",
	"Stack Deployed Successfully":   "✅ Stack deployed",
	"Stack Deploy Failed":           "✗ Stack deployment failed",
}

// HelperPhasePatterns contains patterns for basic terraform testing
var HelperPhasePatterns = PhasePatterns{
	"Running Terraform Init":     "🔄 Initializing Terraform",
	"Running Terraform Plan":     "🔄 Planning infrastructure",
	"Running Terraform Apply":    "🔄 Applying infrastructure",
	"Running Terraform Destroy":  "🔄 Destroying infrastructure",
	"Terraform Apply Complete":   "✅ Infrastructure applied",
	"Terraform Destroy Complete": "✅ Infrastructure destroyed",
}

// SchematicPhasePatterns contains patterns for schematics testing
var SchematicPhasePatterns = PhasePatterns{
	"Creating Workspace":        "🔄 Creating workspace",
	"Uploading Template":        "🔄 Uploading template",
	"Generating Plan":           "🔄 Generating plan",
	"Applying Plan":             "🔄 Applying plan",
	"Destroying Resources":      "🔄 Destroying resources",
	"Workspace Created":         "✅ Workspace created",
	"Plan Applied Successfully": "✅ Plan applied",
	"Resources Destroyed":       "✅ Resources destroyed",
}

// EnableBatchMode enables batch-aware logging to reduce repetitive progress messages
func (s *SmartLogger) EnableBatchMode() {
	s.batchMode = true
}

// DisableBatchMode disables batch-aware logging and clears suppressed phases
func (s *SmartLogger) DisableBatchMode() {
	s.batchMode = false
	s.suppressedPhases = make(map[string]bool) // Clear suppressed phases
	s.activePhase = ""                         // Clear active phase to allow re-showing phases
}

// ShortInfo with automatic phase detection
func (s *SmartLogger) ShortInfo(message string) {
	// Check if this message represents a new phase
	if phase := s.detectPhaseFromMessage(message); phase != "" {
		shouldShowPhase := false

		if s.batchMode {
			// In batch mode, suppress repetitive completion phases but allow progress phases
			if strings.HasPrefix(phase, "✅") {
				if !s.suppressedPhases[phase] {
					s.suppressedPhases[phase] = true
					shouldShowPhase = true
				}
			} else {
				// Allow progress phases if they're different from active phase
				shouldShowPhase = phase != s.activePhase
			}
		} else {
			// In normal mode, allow all completion phases but suppress duplicate progress phases
			if strings.HasPrefix(phase, "✅") {
				shouldShowPhase = true // Always show completion phases in normal mode
			} else {
				shouldShowPhase = phase != s.activePhase
			}
		}

		if shouldShowPhase {
			if !strings.HasPrefix(phase, "✅") {
				// Only update activePhase for progress phases, not completion phases
				s.activePhase = phase
			}
			// Show progress stage instead of raw debug message in quiet mode
			if s.logger.IsQuietMode() {
				s.logger.ProgressStage(strings.TrimPrefix(phase, "🔄 "))
				return // Don't log the raw message in quiet mode
			} else {
				s.logger.ProgressStage(strings.TrimPrefix(phase, "🔄 "))
			}
		} else if s.logger.IsQuietMode() {
			// In quiet mode and phase is suppressed, don't log the raw message either
			return
		}
	}

	// Log the actual message (will be buffered in quiet mode)
	s.logger.ShortInfo(message)
}

// Delegate all other Logger interface methods to the wrapped logger
func (s *SmartLogger) Info(message string) {
	// In quiet mode, suppress regular Info messages unless they're essential
	if s.logger.IsQuietMode() {
		// Only allow essential/progress messages through in quiet mode
		// For now, suppress all Info() calls in quiet mode since they're typically debug/verbose output
		return
	}
	s.logger.Info(message)
}

func (s *SmartLogger) ShortError(message string) {
	s.logger.ShortError(message)
}

func (s *SmartLogger) Error(message string) {
	s.logger.Error(message)
}

func (s *SmartLogger) ShortWarn(message string) {
	s.logger.ShortWarn(message)
}

func (s *SmartLogger) Warn(message string) {
	s.logger.Warn(message)
}

func (s *SmartLogger) ShortDebug(message string) {
	s.logger.ShortDebug(message)
}

func (s *SmartLogger) Debug(message string) {
	s.logger.Debug(message)
}

func (s *SmartLogger) Custom(level, message, color string) {
	s.logger.Custom(level, message, color)
}

func (s *SmartLogger) ShortCustom(message, color string) {
	s.logger.ShortCustom(message, color)
}

func (s *SmartLogger) ProgressStage(stage string) {
	s.activePhase = "🔄 " + stage
	s.logger.ProgressStage(stage)
}

func (s *SmartLogger) ProgressSuccess(message string) {
	s.logger.ProgressSuccess(message)
}

func (s *SmartLogger) ProgressInfo(message string) {
	s.logger.ProgressInfo(message)
}

func (s *SmartLogger) SetPrefix(prefix string) {
	s.logger.SetPrefix(prefix)
}

func (s *SmartLogger) EnableDateTime(enable bool) {
	s.logger.EnableDateTime(enable)
}

func (s *SmartLogger) SetQuietMode(quiet bool) {
	s.logger.SetQuietMode(quiet)
}

func (s *SmartLogger) IsQuietMode() bool {
	return s.logger.IsQuietMode()
}

func (s *SmartLogger) MarkFailed() {
	s.logger.MarkFailed()
}

func (s *SmartLogger) FlushBuffer() {
	s.logger.FlushBuffer()
}

func (s *SmartLogger) FlushOnFailure() {
	s.logger.FlushOnFailure()
}

func (s *SmartLogger) ClearBuffer() {
	s.logger.ClearBuffer()
}

func (s *SmartLogger) GetBufferSize() int {
	return s.logger.GetBufferSize()
}

func (s *SmartLogger) GetUnderlyingLogger() *TestLogger {
	return s.logger.GetUnderlyingLogger()
}

// Enhanced error handling methods for SmartLogger

func (s *SmartLogger) CriticalError(message string) {
	s.logger.CriticalError(message)
}

func (s *SmartLogger) FatalError(message string) {
	s.logger.FatalError(message)
}

func (s *SmartLogger) ErrorWithContext(message string) {
	s.logger.ErrorWithContext(message)
}

// CreateSmartAutoBufferingLogger creates a logger with both auto-buffering and auto-phase detection
// Uses AddonPhasePatterns by default for backward compatibility with existing addon tests
func CreateSmartAutoBufferingLogger(testName string, quietMode bool) Logger {
	baseLogger := CreateAutoBufferingLogger(testName, quietMode)
	config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
	return NewSmartLogger(baseLogger, config)
}

// CreateSmartAutoBufferingLoggerWithPrefix creates a logger with prefix, auto-buffering and auto-phase detection
// Uses AddonPhasePatterns by default for backward compatibility with existing addon tests
func CreateSmartAutoBufferingLoggerWithPrefix(testName, prefix string, quietMode bool) Logger {
	baseLogger := CreateAutoBufferingLoggerWithPrefix(testName, prefix, quietMode)
	config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
	return NewSmartLogger(baseLogger, config)
}

// Convenient factory functions for different test types

// CreateAddonLogger creates a logger configured for addon testing
func CreateAddonLogger(testName string, quietMode bool) Logger {
	baseLogger := CreateAutoBufferingLogger(testName, quietMode)
	config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
	return NewSmartLogger(baseLogger, config)
}

// CreateProjectLogger creates a logger configured for project testing
func CreateProjectLogger(testName string, quietMode bool) Logger {
	baseLogger := CreateAutoBufferingLogger(testName, quietMode)
	config := SmartLoggerConfig{PhasePatterns: ProjectPhasePatterns}
	return NewSmartLogger(baseLogger, config)
}

// CreateHelperLogger creates a logger configured for terraform helper testing
func CreateHelperLogger(testName string, quietMode bool) Logger {
	baseLogger := CreateAutoBufferingLogger(testName, quietMode)
	config := SmartLoggerConfig{PhasePatterns: HelperPhasePatterns}
	return NewSmartLogger(baseLogger, config)
}

// CreateSchematicLogger creates a logger configured for schematics testing
func CreateSchematicLogger(testName string, quietMode bool) Logger {
	baseLogger := CreateAutoBufferingLogger(testName, quietMode)
	config := SmartLoggerConfig{PhasePatterns: SchematicPhasePatterns}
	return NewSmartLogger(baseLogger, config)
}
