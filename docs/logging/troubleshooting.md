# Troubleshooting Guide

This guide covers common issues, solutions, and debugging techniques for the logging framework.

## Common Issues

### 1. No Log Output in Parallel Tests

**Problem**: Logs don't appear during parallel test execution.

**Symptoms**:
```
=== RUN   TestParallelOperation
=== PAUSE TestParallelOperation
=== CONT  TestParallelOperation
--- PASS: TestParallelOperation (10.25s)
```

**Cause**: Using `TestLogger` in quiet mode or not calling `MarkFailed()` with `BufferedTestLogger`.

**Solution**:
```golang
// ❌ Wrong - TestLogger suppresses output in quiet mode
logger := common.NewTestLogger(t.Name())
logger.SetQuietMode(true)

// ✅ Correct - BufferedTestLogger with failure marking
logger := common.NewBufferedTestLogger(t.Name(), true)
// ... test logic ...
if err != nil {
    logger.MarkFailed() // Essential!
    t.Fatalf("Test failed: %v", err)
}
```

### 2. Missing Debug Information on Test Failure

**Problem**: Test fails but no debug logs are shown.

**Symptoms**:
```
--- FAIL: TestComplexOperation (30.15s)
    test.go:45: Test failed: operation timeout
```

**Cause**: Forgetting to call `MarkFailed()` before test failure or using basic error handling.

**Solution**:
```golang
// ❌ Wrong - no debug logs shown
if err != nil {
    t.Fatalf("Test failed: %v", err)
}

// ✅ Better - enhanced error methods handle everything automatically
if err != nil {
    logger.CriticalError(fmt.Sprintf("Test failed: %v", err))
    return
}

// ✅ Also correct - manual approach
if err != nil {
    logger.MarkFailed()
    t.Fatalf("Test failed: %v", err)
}
```

### 3. Repetitive Progress Messages

**Problem**: Too many repetitive progress messages during batch operations.

**Symptoms**:
```
🔄 Retrieving catalog information
🔄 Retrieving catalog information
🔄 Retrieving catalog information
...
```

**Cause**: Not using batch mode for repetitive operations.

**Solution**:
```golang
// ✅ Enable batch mode for repetitive operations
logger := common.CreateAddonLogger(t.Name(), true)
smartLogger := logger.(*common.SmartLogger)
smartLogger.EnableBatchMode()

defer smartLogger.DisableBatchMode()

// Now repetitive messages are suppressed
for _, item := range items {
    logger.ShortInfo("Getting offering details") // Only shows once
    processItem(item)
}
```

### 4. Phase Detection Not Working

**Problem**: Smart logger doesn't detect phases from log messages.

**Symptoms**:
```
[test] Getting offering details
[test] Validating configuration
```
Expected:
```
🔄 Retrieving catalog information
🔄 Validating inputs
```

**Cause**: Using wrong pattern set or incorrect logger type.

**Solution**:
```golang
// ❌ Wrong - no phase detection
logger := common.NewBufferedTestLogger(t.Name(), true)

// ✅ Correct - with appropriate patterns
logger := common.CreateAddonLogger(t.Name(), true)

// Or custom patterns
baseLogger := common.NewBufferedTestLogger(t.Name(), true)
config := common.SmartLoggerConfig{
    PhasePatterns: common.AddonPhasePatterns,
}
logger := common.NewSmartLogger(baseLogger, config)
```

### 5. Logger Nil Pointer Panics

**Problem**: Runtime panic due to nil logger.

**Symptoms**:
```
panic: runtime error: invalid memory address or nil pointer dereference
    at logger.go:123
```

**Cause**: Logger not properly initialized or passed incorrectly.

**Solution**:
```golang
// ❌ Potential nil pointer
var logger common.Logger
logger.ShortInfo("This will panic")

// ✅ Proper initialization
logger := common.NewBufferedTestLogger(t.Name(), true)
if logger == nil {
    t.Fatalf("Failed to create logger")
}

// ✅ Defensive programming
func useLogger(logger common.Logger) {
    if logger == nil {
        log.Printf("Logger is nil, using fallback")
        return
    }
    logger.ShortInfo("Safe logging")
}
```

### 6. Memory Issues with Long-Running Tests

**Problem**: Memory usage grows continuously during long tests.

**Symptoms**:
- Increasing memory consumption
- Slow test execution over time
- Out of memory errors

**Cause**: Buffer accumulation in `BufferedTestLogger` without periodic clearing.

**Solution**:
```golang
logger := common.NewBufferedTestLogger(t.Name(), true)

for i := 0; i < 10000; i++ {
    logger.ShortDebug("Processing item %d", i)

    // Periodically clear buffer to prevent memory growth
    if i%1000 == 0 {
        logger.ClearBuffer()
        logger.ProgressInfo("Processed %d items", i)
    }
}
```

### 7. Colors Not Displaying Properly

**Problem**: ANSI color codes appear as text instead of colors.

**Symptoms**:
```
[0m[31mERROR[0m: Something failed
```

**Cause**: Terminal doesn't support ANSI colors or colors disabled.

**Solution**:
```golang
// Check if terminal supports colors
func supportsColor() bool {
    return os.Getenv("TERM") != "dumb" &&
           (os.Getenv("FORCE_COLOR") == "true" || isatty.IsTerminal(os.Stdout.Fd()))
}

// Conditional color usage
if supportsColor() {
    logger.ShortCustom("Colored message", common.Colors.Green)
} else {
    logger.ShortInfo("Plain message")
}
```

## Debugging Techniques

### 1. Enable Verbose Logging

```golang
func TestWithVerboseLogging(t *testing.T) {
    // Temporarily disable quiet mode for debugging
    logger := common.CreateAddonLogger(t.Name(), false) // verbose mode
    logger.EnableDateTime(true) // Add timestamps

    logger.ShortInfo("Debug mode enabled")
    // Your test logic here
}
```

### 2. Check Buffer Status

```golang
bufferedLogger := common.NewBufferedTestLogger(t.Name(), true)

// Check buffer size during test
logger.ShortInfo("Buffer size: %d", bufferedLogger.GetBufferSize())

// Manually flush for debugging
bufferedLogger.FlushBuffer()
```

### 3. Test Logger Configuration

```golang
func TestLoggerConfiguration(t *testing.T) {
    logger := common.CreateAddonLogger(t.Name(), true)

    t.Logf("Logger quiet mode: %v", logger.IsQuietMode())
    t.Logf("Underlying logger: %T", logger.GetUnderlyingLogger())

    // Test different log levels
    logger.ShortInfo("Info message")
    logger.ShortWarn("Warning message")
    logger.ShortError("Error message") // Should always show
    logger.ProgressStage("Progress message") // Should always show
}
```

### 4. Verify Phase Pattern Matching

```golang
func TestPhasePatterns(t *testing.T) {
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

## Environment-Specific Issues

### CI/CD Environments

**Issue**: Different behavior in CI vs. local development.

**Solution**:
```golang
func createEnvironmentAwareLogger(testName string) common.Logger {
    isCI := os.Getenv("CI") == "true" ||
            os.Getenv("GITHUB_ACTIONS") == "true" ||
            os.Getenv("JENKINS_URL") != ""

    logger := common.CreateAddonLogger(testName, isCI)

    if isCI {
        // CI-specific configuration
        logger.EnableDateTime(true) // Timestamps useful in CI logs
    }

    return logger
}
```

### Docker Environments

**Issue**: Colors or interactive features not working in containers.

**Solution**:
```golang
func createDockerLogger(testName string) common.Logger {
    inDocker := os.Getenv("CONTAINER") == "true" ||
               fileExists("/.dockerenv")

    logger := common.CreateAddonLogger(testName, true)

    if inDocker {
        // Docker-specific adjustments
        logger.EnableDateTime(true)
        // Colors usually work in Docker, but can be disabled if needed
    }

    return logger
}
```

## Performance Issues

### 1. Slow Logger Creation

**Problem**: Logger creation takes too long.

**Cause**: Complex configuration or pattern matching.

**Solution**:
```golang
// Cache logger instances for reuse
var loggerCache = make(map[string]common.Logger)
var loggerMutex sync.RWMutex

func getCachedLogger(testName string) common.Logger {
    loggerMutex.RLock()
    if logger, exists := loggerCache[testName]; exists {
        loggerMutex.RUnlock()
        return logger
    }
    loggerMutex.RUnlock()

    loggerMutex.Lock()
    defer loggerMutex.Unlock()

    // Double-check after acquiring write lock
    if logger, exists := loggerCache[testName]; exists {
        return logger
    }

    logger := common.CreateAddonLogger(testName, true)
    loggerCache[testName] = logger
    return logger
}
```

### 2. High Memory Usage

**Problem**: Logger consuming too much memory.

**Solution**:
```golang
// Monitor buffer size
func monitorBufferSize(logger common.Logger) {
    if buffered, ok := logger.(*common.BufferedTestLogger); ok {
        size := buffered.GetBufferSize()
        if size > 1000 { // Threshold
            buffered.ClearBuffer()
            buffered.ProgressInfo("Cleared buffer due to size: %d", size)
        }
    }
}
```

## Diagnostic Tools

### 1. Logger Type Detection

```golang
func diagnoseLogger(logger common.Logger) {
    fmt.Printf("Logger type: %T\n", logger)
    fmt.Printf("Quiet mode: %v\n", logger.IsQuietMode())
    fmt.Printf("Buffer size: %d\n", logger.GetBufferSize())

    underlying := logger.GetUnderlyingLogger()
    fmt.Printf("Underlying logger: %T\n", underlying)
}
```

### 2. Pattern Matching Test

```golang
func testPatternMatching(patterns common.PhasePatterns, message string) {
    fmt.Printf("Testing message: %s\n", message)

    for pattern, phase := range patterns {
        if strings.Contains(message, pattern) {
            fmt.Printf("  Matches pattern '%s' -> '%s'\n", pattern, phase)
        }
    }
}
```

### 3. Buffer Content Inspection

```golang
// For debugging buffer contents (requires custom modification)
func inspectBuffer(logger *common.BufferedTestLogger) {
    // This would require exposing buffer contents in the API
    // For now, use GetBufferSize() and strategic FlushBuffer() calls
    fmt.Printf("Buffer size: %d\n", logger.GetBufferSize())

    // Temporary flush to see contents
    logger.FlushBuffer()
    logger.ClearBuffer()
}
```

## Best Practices for Troubleshooting

1. **Always check logger type** when debugging issues
2. **Use temporary verbose mode** to diagnose problems
3. **Test with simple logger first** before using complex configurations
4. **Check environment variables** that might affect behavior
5. **Use buffer size monitoring** for long-running tests
6. **Test phase patterns** independently before using in smart loggers
7. **Enable timestamps** when debugging timing-related issues
8. **Use manual flush** to verify buffer contents during development

## Getting Help

When reporting issues:

1. **Include logger configuration** used
2. **Provide minimal reproduction** case
3. **Specify environment details** (CI, local, Docker)
4. **Include relevant log output** showing the problem
5. **Mention expected vs. actual behavior**

Example issue report:
```golang
// Configuration used
logger := common.CreateAddonLogger(t.Name(), true)

// Environment
// - Go 1.19
// - Running in GitHub Actions
// - Parallel tests enabled

// Expected: Progress messages should show during test
// Actual: No output until test completes

// Minimal reproduction
func TestIssueReproduction(t *testing.T) {
    t.Parallel()
    logger := common.CreateAddonLogger(t.Name(), true)
    logger.ShortInfo("Getting offering details") // Should show progress
    time.Sleep(5 * time.Second)
    logger.ShortInfo("Request completed")
}
```

For additional help, see [Configuration](configuration.md) for detailed setup options and [Examples](examples.md) for working code samples.
