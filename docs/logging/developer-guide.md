# Developer Guide for Framework Developers

## For Framework Developers

This guide is for developers who need to understand, extend, or modify the logging framework implementation. It covers logger internals, architecture, and advanced usage patterns.

## Logger Architecture

### Interface Hierarchy

The logging system is built around a core `Logger` interface that provides universal compatibility:

```golang
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

    // Progress methods (bypass quiet mode)
    ProgressStage(stage string)
    ProgressSuccess(message string)
    ProgressInfo(message string)

    // Configuration methods
    SetPrefix(prefix string)
    EnableDateTime(enable bool)
    SetQuietMode(quiet bool)
    IsQuietMode() bool

    // Enhanced error methods (automatic buffer management)
    CriticalError(message string)
    ErrorWithContext(message string)
    FatalError(message string)

    // Manual buffering methods (no-op for TestLogger)
    FlushBuffer()
    FlushOnFailure()
    ClearBuffer()
    GetBufferSize() int

    // Enhanced error handling methods
    CriticalError(message string)     // Shows buffer context first, then prominent red-bordered error
    FatalError(message string)        // Immediate error display, bypasses buffering
    ErrorWithContext(message string)  // Shows buffer context with moderate error formatting

    // Immediate methods (bypass buffering and quiet mode)
    ImmediateShortError(message string)  // Force error output immediately
    ImmediateShortInfo(message string)   // Force info output immediately
    ImmediateShortWarn(message string)   // Force warning output immediately

    // Compatibility methods
    GetUnderlyingLogger() *TestLogger
}
```

### Implementation Hierarchy

```
Logger (interface)
├── TestLogger (basic implementation)
│   ├── Direct console output
│   ├── Color support
│   └── Quiet mode suppression
├── BufferedTestLogger (wraps TestLogger)
│   ├── Message buffering in quiet mode
│   ├── Conditional output on failure
│   └── Buffer management
├── SmartLogger (wraps any Logger)
│   ├── Automatic phase detection
│   ├── Pattern matching
│   └── Progress tracking
└── LoggerProxy (delegation wrapper)
    ├── Automatic buffering decisions
    ├── Context-aware wrapping
    └── Universal delegation
```

## Core Components

### 1. TestLogger - Foundation Implementation

The base logger that handles direct output:

```golang
type TestLogger struct {
    logger          *log.Logger      // Go standard logger
    testName        string           // Test identification
    prefix          string           // Message prefix
    includeDateTime bool             // Timestamp control
    quietMode       bool             // Output suppression
}

// Key behaviors:
// - Immediate output to console
// - Color-coded messages using ANSI codes
// - Quiet mode suppresses non-critical messages
// - File/line tracking with runtime.Caller()
```

#### Critical Implementation Details

**Caller Tracking**: Uses `runtime.Caller(3)` to get file/line information:
```golang
func (t *TestLogger) logWithCaller(level, message, color string) {
    _, file, line, ok := runtime.Caller(3) // Depth 3 due to wrapper layers
    if ok {
        file = file[strings.LastIndex(file, "/")+1:] // Extract filename only
    }
    // ... formatting and output
}
```

**Quiet Mode Bypass**: Critical messages always show:
```golang
func (t *TestLogger) logWithCallerForceOutput(level, message, color string) {
    // This bypasses quiet mode for errors and progress messages
}
```

### 2. BufferedTestLogger - Advanced Buffering

Wraps TestLogger to provide conditional output:

```golang
type BufferedTestLogger struct {
    testLogger *TestLogger        // Underlying logger
    buffer     []LogMessage       // Message buffer
    quietMode  bool              // Buffer control flag
    failed     bool              // Failure state
}

type LogMessage struct {
    Timestamp time.Time           // When logged
    Level     string             // Log level
    Message   string             // Actual message
    Color     string             // ANSI color code
    TestName  string             // Test identifier
    Prefix    string             // Message prefix
}
```

#### Buffer Management

**Buffering Logic**:
```golang
func (b *BufferedTestLogger) ShortInfo(message string) {
    if b.quietMode {
        b.addToBuffer("", message, Colors.Green) // Store for later
    } else {
        b.testLogger.ShortInfo(message)          // Immediate output
    }
}
```

**Conditional Flush**:
```golang
func (b *BufferedTestLogger) FlushOnFailure() {
    if b.failed {
        b.FlushBuffer() // Only flush if test marked as failed
    }
}
```

**Memory Management**: Buffer is cleared after flushing to prevent memory leaks and message duplication:
```golang
func (b *BufferedTestLogger) FlushBuffer() {
    if len(b.buffer) > 0 {
        b.testLogger.logWithoutCallerForceOutput("", "=== BUFFERED LOG OUTPUT ===", Colors.Yellow)
        for _, msg := range b.buffer {
            // ... output buffered messages ...
        }
        b.testLogger.logWithoutCallerForceOutput("", "=== END BUFFERED LOG OUTPUT ===", Colors.Yellow)
        b.buffer = make([]LogMessage, 0) // Clear buffer after flush to prevent duplication
    }
}
```

### 3. SmartLogger - Intelligent Phase Detection

Provides automatic progress tracking through pattern matching:

```golang
type SmartLogger struct {
    logger Logger                    // Wrapped logger (any type)
    config SmartLoggerConfig        // Phase patterns configuration
    activePhase string              // Current detected phase
    batchMode bool                  // Repetition suppression
    suppressedPhases map[string]bool // Batch tracking
}

type SmartLoggerConfig struct {
    PhasePatterns PhasePatterns     // Pattern to phase mapping
}

type PhasePatterns map[string]string // "log substring" -> "progress message"

// Example predefined pattern sets
var AddonPhasePatterns = PhasePatterns{
    "Getting offering details":           "🔄 Retrieving catalog information",
    "Getting offering version locator":   "🔄 Resolving version constraints",
    "Starting reference resolution":      "🔄 Resolving project references",
    "Attempting reference resolution":    "🔄 Validating dependencies",
    "Configuration deployed to project":  "✅ Configuration deployed to project",
}
```

#### Phase Detection Algorithm

**Pattern Matching**:
```golang
func (s *SmartLogger) detectPhaseFromMessage(message string) string {
    for pattern, phase := range s.config.PhasePatterns {
        if strings.Contains(message, pattern) {
            return phase // First match wins
        }
    }
    return "" // No phase detected
}
```

**Intelligent Display Logic**:
```golang
func (s *SmartLogger) ShortInfo(message string) {
    if phase := s.detectPhaseFromMessage(message); phase != "" {
        shouldShowPhase := false

        if s.batchMode {
            // Suppress repetitive completion phases
            if strings.HasPrefix(phase, "✅") {
                if !s.suppressedPhases[phase] {
                    s.suppressedPhases[phase] = true
                    shouldShowPhase = true
                }
            } else {
                // Allow progress phases if different
                shouldShowPhase = phase != s.activePhase
            }
        } else {
            // Normal mode: show completions, suppress duplicate progress
            shouldShowPhase = strings.HasPrefix(phase, "✅") || phase != s.activePhase
        }

        if shouldShowPhase {
            s.activePhase = phase
            s.logger.ProgressStage(strings.TrimPrefix(phase, "🔄 "))
        }
    }

    s.logger.ShortInfo(message) // Always log the original message
}
```

## Advanced Features

### Enhanced Error Handling Methods

The logger interface includes specialized error methods and immediate output methods:

```golang
// CriticalError: Shows buffer context first, then prominent bordered error
func (b *BufferedTestLogger) CriticalError(message string) {
    b.MarkFailed()
    b.FlushOnFailure()

    separator := strings.Repeat("=", 80)
    b.testLogger.logWithoutCallerForceOutput("", separator, Colors.Red)
    b.testLogger.logWithoutCallerForceOutput("", fmt.Sprintf("CRITICAL ERROR: %s", message), Colors.Red)
    b.testLogger.logWithoutCallerForceOutput("", separator, Colors.Red)
}

// FatalError: Immediate error display, bypasses all buffering
func (b *BufferedTestLogger) FatalError(message string) {
    b.testLogger.logWithoutCallerForceOutput("", fmt.Sprintf("FATAL ERROR: %s", message), Colors.Red)
}

// ErrorWithContext: Shows buffer context with moderate error formatting
func (b *BufferedTestLogger) ErrorWithContext(message string) {
    b.MarkFailed()
    b.FlushOnFailure()

    separator := strings.Repeat("-", 60)
    b.testLogger.logWithoutCallerForceOutput("", separator, Colors.Yellow)
    b.testLogger.logWithoutCallerForceOutput("", fmt.Sprintf("ERROR: %s", message), Colors.Red)
    b.testLogger.logWithoutCallerForceOutput("", separator, Colors.Yellow)
}

// ImmediateShortError: Force error output bypassing buffering and quiet mode
func (b *BufferedTestLogger) ImmediateShortError(message string) {
    b.testLogger.logWithCallerForceOutput("", message, Colors.Red)
}

// ImmediateShortInfo: Force info output bypassing buffering and quiet mode
func (b *BufferedTestLogger) ImmediateShortInfo(message string) {
    b.testLogger.logWithCallerForceOutput("", message, Colors.Green)
}

// ImmediateShortWarn: Force warning output bypassing buffering and quiet mode
func (b *BufferedTestLogger) ImmediateShortWarn(message string) {
    b.testLogger.logWithCallerForceOutput("", message, Colors.Yellow)
}
```

### Color System Implementation

ANSI color support with automatic reset:

```golang
type Color struct {
    Reset  string // "\033[0m"
    Red    string // "\033[31m"
    Green  string // "\033[32m"
    Yellow string // "\033[33m"
    Blue   string // "\033[34m"
    Orange string // "\033[38;5;208m"
    Purple string // "\033[35m"
    Cyan   string // "\033[36m"
}

func ColorizeString(color, message string) string {
    return color + message + Colors.Reset // Automatic reset prevents color bleed
}
```

### Factory Function Architecture

Simplified logger creation with sensible defaults:

```golang
// Auto-buffering based on quiet mode
func CreateAutoBufferingLogger(testName string, quietMode bool) Logger {
    baseLogger := NewTestLogger(testName)
    return WrapWithBufferingIfNeeded(baseLogger, quietMode)
}

// Specialized loggers with predefined patterns
func CreateAddonLogger(testName string, quietMode bool) Logger {
    baseLogger := CreateAutoBufferingLogger(testName, quietMode)
    config := SmartLoggerConfig{PhasePatterns: AddonPhasePatterns}
    return NewSmartLogger(baseLogger, config)
}

func CreateProjectLogger(testName string, quietMode bool) Logger {
    baseLogger := CreateAutoBufferingLogger(testName, quietMode)
    config := SmartLoggerConfig{PhasePatterns: ProjectPhasePatterns}
    return NewSmartLogger(baseLogger, config)
}

func CreateHelperLogger(testName string, quietMode bool) Logger {
    baseLogger := CreateAutoBufferingLogger(testName, quietMode)
    config := SmartLoggerConfig{PhasePatterns: HelperPhasePatterns}
    return NewSmartLogger(baseLogger, config)
}
```

## Extending the Framework

### Creating Custom Logger Types

Implement the `Logger` interface for specialized behavior:

```golang
type CustomLogger struct {
    underlying Logger
    // Custom fields
}

// Must implement all Logger interface methods
func (c *CustomLogger) ShortInfo(message string) {
    // Custom logic
    c.underlying.ShortInfo(message)
}

// ... implement remaining 20+ interface methods
```

### Adding New Phase Patterns

Define patterns for new test types:

```golang
var CustomPhasePatterns = PhasePatterns{
    "Starting data migration":     "🔄 Migrating data",
    "Validating data integrity":   "🔄 Validating integrity",
    "Updating database schema":    "🔄 Updating schema",
    "Migration completed":         "✅ Migration successful",
}

// Create specialized logger
func CreateCustomLogger(testName string, quietMode bool) Logger {
    baseLogger := CreateAutoBufferingLogger(testName, quietMode)
    config := SmartLoggerConfig{PhasePatterns: CustomPhasePatterns}
    return NewSmartLogger(baseLogger, config)
}
```

### Wrapper Integration

Integrate loggers with test wrapper packages:

```golang
type TestOptions struct {
    Testing           *testing.T
    Logger            *common.BufferedTestLogger  // Specific type required
    QuietMode         *bool
    VerboseOnFailure  bool
    // ... other fields
}

func TestOptionsDefault(inputOptions *TestOptions) *TestOptions {
    newOptions := &TestOptions{
        // ... set defaults ...
    }

    // Initialize logger if not provided
    if newOptions.Logger == nil {
        testName := "default-test"
        if newOptions.Testing != nil && newOptions.Testing.Name() != "" {
            testName = newOptions.Testing.Name()
        }

        quietMode := false
        if newOptions.QuietMode != nil {
            quietMode = *newOptions.QuietMode
        }

        newOptions.Logger = common.NewBufferedTestLogger(testName, quietMode)

        // Configure quiet mode if specified
        if newOptions.QuietMode != nil && *newOptions.QuietMode {
            newOptions.Logger.SetQuietMode(true)
        }
    }

    return newOptions
}
```

## Performance Considerations

### Memory Management

**Buffer Size Monitoring**:
```golang
func monitorBufferSize(logger common.Logger) {
    if buffered, ok := logger.(*common.BufferedTestLogger); ok {
        size := buffered.GetBufferSize()
        if size > 1000 { // Threshold for cleanup
            buffered.ClearBuffer()
            buffered.ProgressInfo("Cleared buffer due to size: %d", size)
        }
    }
}
```

**Periodic Buffer Clearing**:
```golang
for i := 0; i < 10000; i++ {
    logger.ShortDebug("Processing item %d", i)

    if i%1000 == 0 {
        logger.ClearBuffer() // Prevent memory growth
        logger.ProgressInfo("Processed %d items", i)
    }
}
```

### Pattern Matching Optimization

Phase pattern matching is O(n) where n is the number of patterns. For performance-critical scenarios:

```golang
// Cache compiled patterns for repeated use
type OptimizedSmartLogger struct {
    *SmartLogger
    compiledPatterns map[*regexp.Regexp]string
}

func (o *OptimizedSmartLogger) detectPhaseFromMessage(message string) string {
    for regex, phase := range o.compiledPatterns {
        if regex.MatchString(message) {
            return phase
        }
    }
    return ""
}
```

## Testing the Logger Framework

### Unit Test Patterns

```golang
func TestLoggerBehavior(t *testing.T) {
    var buf bytes.Buffer
    baseLogger := &TestLogger{
        logger:   log.New(&buf, "", 0),
        testName: "test-logger",
    }

    logger := WrapWithBufferingIfNeeded(baseLogger, true)

    logger.ShortInfo("Test message")
    assert.Equal(t, 0, len(buf.String())) // Should be buffered

    logger.CriticalError("Test failed")
    assert.Contains(t, buf.String(), "Test message") // Should be flushed
    assert.Contains(t, buf.String(), "CRITICAL ERROR") // Should show error
}
```

### Integration Test Patterns

```golang
func TestPhaseDetection(t *testing.T) {
    patterns := common.AddonPhasePatterns
    testMessages := []string{
        "Getting offering details",
        "Validating configuration",
        "Unknown message",
    }

    for _, msg := range testMessages {
        found := false
        for pattern, phase := range patterns {
            if strings.Contains(msg, pattern) {
                t.Logf("Message '%s' matches pattern '%s' -> '%s'", msg, pattern, phase)
                found = true
                break
            }
        }
        if !found {
            t.Logf("Message '%s' has no matching pattern", msg)
        }
    }
}
```

## Migration and Compatibility

### Backward Compatibility

The framework maintains compatibility through interface conformance:

```golang
// Old code using specific types still works
var logger *common.BufferedTestLogger = common.NewBufferedTestLogger("test", true)

// New code can use interface for flexibility
var logger common.Logger = common.CreateAddonLogger("test", true)

// Both work with existing APIs
someFunction(logger.GetUnderlyingLogger()) // Returns *TestLogger
```

### Migration Path

When adding new features:

1. **Extend interface**: Add new methods with no-op defaults
2. **Implement in all types**: Maintain compatibility
3. **Update factory functions**: Provide new functionality
4. **Document breaking changes**: Clear migration guidance

## Debugging and Diagnostics

### Logger Type Detection

```golang
func diagnoseLogger(logger common.Logger) {
    fmt.Printf("Logger type: %T\n", logger)
    fmt.Printf("Quiet mode: %v\n", logger.IsQuietMode())
    fmt.Printf("Buffer size: %d\n", logger.GetBufferSize())

    underlying := logger.GetUnderlyingLogger()
    fmt.Printf("Underlying logger: %T\n", underlying)

    // Type-specific diagnostics
    if smartLogger, ok := logger.(*common.SmartLogger); ok {
        fmt.Printf("Smart logger batch mode: %v\n", smartLogger.batchMode)
    }
}
```

### Performance Profiling

```golang
func profileLoggerPerformance() {
    logger := common.CreateAddonLogger("perf-test", true)

    start := time.Now()
    for i := 0; i < 10000; i++ {
        logger.ShortInfo("Test message %d", i)
    }
    elapsed := time.Since(start)

    fmt.Printf("10000 messages in %v (%v per message)\n", elapsed, elapsed/10000)
    fmt.Printf("Buffer size: %d\n", logger.GetBufferSize())
}
```

This developer guide provides the technical foundation for understanding and extending the logging framework. For user-focused configuration guidance, see the [User Guide](user-guide.md).
