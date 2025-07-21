# Configuration Guide

This guide covers all configuration options available in the logging framework, from basic setup to advanced customization.

## Basic Configuration

### Logger Construction Options

#### TestLogger

```golang
// Basic logger
logger := common.NewTestLogger("test-name")

// With prefix
logger := common.NewTestLoggerWithPrefix("test-name", "prefix")

// With quiet mode
logger := common.NewTestLoggerWithQuietMode("test-name", true)

// With both prefix and quiet mode
logger := common.NewTestLoggerWithPrefixAndQuietMode("test-name", "prefix", true)

// Inherit from parent logger
parentLogger := common.NewTestLogger("parent")
logger := common.NewTestLoggerFromParent("child", parentLogger)

// Inherit with prefix
logger := common.NewTestLoggerFromParentWithPrefix("child", "prefix", parentLogger)
```

#### BufferedTestLogger

```golang
// Basic buffered logger
logger := common.NewBufferedTestLogger("test-name", true) // quietMode

// With prefix
logger := common.NewBufferedTestLoggerWithPrefix("test-name", "prefix", true)
```

#### SmartLogger

```golang
// Custom configuration
baseLogger := common.NewBufferedTestLogger("test-name", true)
config := common.SmartLoggerConfig{
    PhasePatterns: customPatterns,
}
logger := common.NewSmartLogger(baseLogger, config)
```

### Factory Functions

Simplified creation with predefined configurations:

```golang
// Auto-buffering loggers
logger := common.CreateAutoBufferingLogger("test-name", true)
logger := common.CreateAutoBufferingLoggerWithPrefix("test-name", "prefix", true)

// Smart auto-buffering loggers (uses AddonPhasePatterns)
logger := common.CreateSmartAutoBufferingLogger("test-name", true)
logger := common.CreateSmartAutoBufferingLoggerWithPrefix("test-name", "prefix", true)

// Specialized loggers
logger := common.CreateAddonLogger("test-name", true)      // Addon patterns
logger := common.CreateProjectLogger("test-name", true)   // Project patterns
logger := common.CreateHelperLogger("test-name", true)    // Terraform patterns
logger := common.CreateSchematicLogger("test-name", true) // Schematics patterns
```

## Runtime Configuration

### Common Settings

```golang
// Change prefix
logger.SetPrefix("new-prefix")

// Enable/disable timestamps
logger.EnableDateTime(true)  // Shows timestamps
logger.EnableDateTime(false) // Hides timestamps (default)

// Control quiet mode
logger.SetQuietMode(true)   // Suppress normal output
logger.SetQuietMode(false)  // Show all output

// Check quiet mode status
isQuiet := logger.IsQuietMode()
```

### Buffered Logger Settings

```golang
bufferedLogger := common.NewBufferedTestLogger("test-name", true)

// Mark test as failed (triggers buffer flush)
bufferedLogger.MarkFailed()

// Manual buffer operations
bufferedLogger.FlushBuffer()        // Always flush
bufferedLogger.FlushOnFailure()     // Flush only if marked as failed
bufferedLogger.ClearBuffer()        // Clear without flushing

// Buffer status
size := bufferedLogger.GetBufferSize()

// Enhanced error handling methods
bufferedLogger.CriticalError("Critical failure")     // Shows buffer context + red-bordered error
bufferedLogger.FatalError("Immediate failure")       // Bypasses buffering, immediate display
bufferedLogger.ErrorWithContext("Moderate error")    // Shows buffer context + yellow-bordered error
```

### Smart Logger Settings

```golang
smartLogger := common.NewSmartLogger(baseLogger, config)

// Batch mode - reduces repetitive progress messages
smartLogger.EnableBatchMode()
smartLogger.DisableBatchMode()
```

## Phase Pattern Configuration

### Understanding Phase Patterns

Phase patterns map log message substrings to progress stage messages:

```golang
type PhasePatterns map[string]string

patterns := common.PhasePatterns{
    "log message substring": "🔄 Progress stage message",
    "completion substring":  "✅ Completion message",
    "error substring":       "❌ Error message",
}
```

### Predefined Pattern Sets

#### AddonPhasePatterns

For IBM Cloud Projects addon testing:

```golang
var AddonPhasePatterns = common.PhasePatterns{
    "Getting offering details":           "🔄 Retrieving catalog information",
    "Getting offering version locator":   "🔄 Resolving version constraints",
    "Starting reference resolution":      "🔄 Resolving project references",
    "Attempting reference resolution":    "🔄 Validating dependencies",
    "Request completed":                  "✅ Operation completed",
    "Creating catalog":                   "🔄 Setting up catalog",
    "Importing offering":                 "🔄 Loading offering configuration",
    "Validating configuration":           "🔄 Validating inputs",
    "Processing configuration details":   "🔄 Processing configuration",
    "Building dependency graph":          "🔄 Analyzing dependencies",
}
```

#### ProjectPhasePatterns

For IBM Cloud Projects stack testing:

```golang
var ProjectPhasePatterns = common.PhasePatterns{
    "Configuring Test Stack":         "🔄 Configuring stack",
    "Triggering Deploy":              "🔄 Triggering deployment",
    "Deploy Triggered Successfully":  "✅ Deployment triggered",
    "Checking Stack Deploy Status":   "🔄 Checking deployment status",
    "Stack Deployed Successfully":    "✅ Stack deployed",
    "Stack Deploy Failed":            "❌ Stack deployment failed",
}
```

#### HelperPhasePatterns

For basic Terraform operations:

```golang
var HelperPhasePatterns = common.PhasePatterns{
    "Running Terraform Init":     "🔄 Initializing Terraform",
    "Running Terraform Plan":     "🔄 Planning infrastructure",
    "Running Terraform Apply":    "🔄 Applying infrastructure",
    "Running Terraform Destroy":  "🔄 Destroying infrastructure",
    "Terraform Apply Complete":   "✅ Infrastructure applied",
    "Terraform Destroy Complete": "✅ Infrastructure destroyed",
}
```

#### SchematicPhasePatterns

For IBM Cloud Schematics operations:

```golang
var SchematicPhasePatterns = common.PhasePatterns{
    "Creating Workspace":         "🔄 Creating workspace",
    "Uploading Template":         "🔄 Uploading template",
    "Generating Plan":            "🔄 Generating plan",
    "Applying Plan":              "🔄 Applying plan",
    "Destroying Resources":       "🔄 Destroying resources",
    "Workspace Created":          "✅ Workspace created",
    "Plan Applied Successfully":  "✅ Plan applied",
    "Resources Destroyed":        "✅ Resources destroyed",
}
```

### Custom Phase Patterns

Create your own patterns for specific use cases:

```golang
customPatterns := common.PhasePatterns{
    // Progress phases (🔄)
    "Starting data migration":     "🔄 Migrating data",
    "Validating data integrity":   "🔄 Validating integrity",
    "Updating database schema":    "🔄 Updating schema",
    "Connecting to service":       "🔄 Establishing connection",

    // Success phases (✅)
    "Migration completed":         "✅ Migration successful",
    "Validation passed":           "✅ Data integrity confirmed",
    "Schema updated":              "✅ Schema update successful",
    "Connection established":      "✅ Connected to service",

    // Error phases (❌)
    "Migration failed":            "❌ Migration failed",
    "Validation failed":           "❌ Data integrity check failed",
    "Schema update failed":        "❌ Schema update failed",
    "Connection timeout":          "❌ Connection failed",
}

config := common.SmartLoggerConfig{PhasePatterns: customPatterns}
logger := common.NewSmartLogger(baseLogger, config)
```

### Combining Pattern Sets

Merge multiple pattern sets:

```golang
combinedPatterns := make(common.PhasePatterns)

// Add addon patterns
for k, v := range common.AddonPhasePatterns {
    combinedPatterns[k] = v
}

// Add custom patterns
customPatterns := common.PhasePatterns{
    "Custom operation": "🔄 Custom phase",
}
for k, v := range customPatterns {
    combinedPatterns[k] = v
}

config := common.SmartLoggerConfig{PhasePatterns: combinedPatterns}
```

## Color Configuration

### Available Colors

```golang
// Predefined colors
common.Colors.Reset   // "\033[0m"
common.Colors.Red     // "\033[31m"     - Errors
common.Colors.Green   // "\033[32m"     - Success/Info
common.Colors.Yellow  // "\033[33m"     - Warnings
common.Colors.Blue    // "\033[34m"     - Debug
common.Colors.Orange  // "\033[38;5;208m" - Custom
common.Colors.Purple  // "\033[35m"     - Custom
common.Colors.Cyan    // "\033[36m"     - Progress/Custom
```

### Using Colors

```golang
// In custom logging
logger.Custom("CUSTOM", "Custom message", common.Colors.Purple)
logger.ShortCustom("Short custom message", common.Colors.Orange)

// Creating colored strings
coloredText := common.ColorizeString(common.Colors.Cyan, "This is cyan")
logger.ShortInfo(coloredText)
```

### Custom Color Codes

```golang
// Define custom ANSI colors
brightMagenta := "\033[95m"
logger.ShortCustom("Bright magenta message", brightMagenta)
```

## SmartLogger Configuration

### SmartLoggerConfig Structure

```golang
type SmartLoggerConfig struct {
    PhasePatterns PhasePatterns
}
```

### Configuration Options

```golang
config := common.SmartLoggerConfig{
    PhasePatterns: common.AddonPhasePatterns,
}

logger := common.NewSmartLogger(baseLogger, config)
```

### Batch Mode Configuration

Batch mode reduces repetitive progress messages during bulk operations:

```golang
smartLogger := logger.(*common.SmartLogger)

// Enable batch mode
smartLogger.EnableBatchMode()

// Process multiple items (repetitive progress messages suppressed)
for _, item := range items {
    logger.ShortInfo("Getting offering details") // Only shows first time
    processItem(item)
    logger.ShortInfo("Request completed") // Shows each completion
}

// Disable batch mode
smartLogger.DisableBatchMode()
```

## Advanced Configuration Patterns

### Conditional Logger Configuration

```golang
func createLogger(testName string, isCI bool, verboseMode bool) common.Logger {
    quietMode := isCI && !verboseMode

    if verboseMode {
        // Verbose mode: show everything immediately
        return common.NewTestLogger(testName)
    } else if isCI {
        // CI mode: quiet with smart phase detection
        return common.CreateAddonLogger(testName, true)
    } else {
        // Local development: buffered
        return common.NewBufferedTestLogger(testName, false)
    }
}
```

### Environment-Based Configuration

```golang
func createConfiguredLogger(testName string) common.Logger {
    quietMode := os.Getenv("QUIET_MODE") == "true"
    logLevel := os.Getenv("LOG_LEVEL") // debug, info, warn, error
    testType := os.Getenv("TEST_TYPE")  // addon, project, helper, schematic

    var logger common.Logger

    switch testType {
    case "addon":
        logger = common.CreateAddonLogger(testName, quietMode)
    case "project":
        logger = common.CreateProjectLogger(testName, quietMode)
    case "helper":
        logger = common.CreateHelperLogger(testName, quietMode)
    case "schematic":
        logger = common.CreateSchematicLogger(testName, quietMode)
    default:
        logger = common.CreateAutoBufferingLogger(testName, quietMode)
    }

    // Configure based on log level
    switch logLevel {
    case "error":
        // Custom configuration to show only errors
        logger.SetQuietMode(true)
    case "debug":
        // Enable timestamps for debug mode
        logger.EnableDateTime(true)
    }

    return logger
}
```

### Dynamic Configuration Updates

```golang
func TestWithDynamicConfig(t *testing.T) {
    t.Parallel()

    logger := common.CreateAddonLogger(t.Name(), false)

    // Start with verbose logging
    logger.ShortInfo("Starting with verbose output")

    // Switch to quiet mode for bulk operations
    logger.SetQuietMode(true)
    for i := 0; i < 100; i++ {
        logger.ShortDebug("Processing item %d", i) // Suppressed
    }

    // Switch back to verbose for final steps
    logger.SetQuietMode(false)
    logger.ShortInfo("Bulk processing completed")

    // Add prefix for final validation
    logger.SetPrefix("validation")
    logger.ShortInfo("Running final validation")
}
```

## Best Practices

1. **Use factory functions** when possible for consistent configuration
2. **Choose appropriate quiet mode** based on test execution context (parallel vs. sequential)
3. **Use predefined patterns** before creating custom ones
4. **Combine pattern sets** when you need multiple types of phase detection
5. **Enable batch mode** for repetitive operations
6. **Configure based on environment** (CI vs. local development)
7. **Use progress methods** for user-facing status updates that should always show

## Configuration Reference

### Constructor Functions

| Function | Purpose | Parameters |
|----------|---------|------------|
| `NewTestLogger` | Basic logger | `testName` |
| `NewBufferedTestLogger` | Buffered logger | `testName, quietMode` |
| `NewSmartLogger` | Smart phase detection | `logger, config` |
| `CreateAddonLogger` | Addon-optimized | `testName, quietMode` |
| `CreateProjectLogger` | Project-optimized | `testName, quietMode` |
| `CreateHelperLogger` | Terraform-optimized | `testName, quietMode` |
| `CreateSchematicLogger` | Schematics-optimized | `testName, quietMode` |

### Configuration Methods

| Method | Purpose | Parameters |
|--------|---------|------------|
| `SetPrefix` | Set log prefix | `prefix` |
| `EnableDateTime` | Toggle timestamps | `enable` |
| `SetQuietMode` | Toggle quiet mode | `quiet` |
| `MarkFailed` | Mark for buffer flush | none |
| `EnableBatchMode` | Reduce repetitive messages | none |
| `DisableBatchMode` | Re-enable all messages | none |

### Enhanced Error Methods

| Method | Purpose | Behavior |
|--------|---------|----------|
| `CriticalError` | Severe test failures | Shows buffer context + red-bordered error |
| `FatalError` | Immediate failures | Bypasses buffering, immediate display |
| `ErrorWithContext` | Moderate failures | Shows buffer context + yellow-bordered error |

See [Examples](examples.md) for practical usage scenarios and [Testing Integration](testing-integration.md) for framework-specific guidance.
