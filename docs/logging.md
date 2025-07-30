# Logging Configuration Guide

## For Test Writers

This guide shows you how to configure logging behavior when writing tests using the IBM Cloud Terratest Wrapper. You'll control logging through test options - no need to create or manage loggers directly.

## Quick Start

Most tests should use quiet mode to keep output clean during parallel execution, but show detailed logs when tests fail:

```golang
package test

import (
    "testing"
    "github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testaddons"
)

func TestMyAddon(t *testing.T) {
    t.Parallel()

    // Configure logging behavior through test options
    quietMode := true
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        Prefix:            "my-test",
        ResourceGroup:     "test-rg",
        QuietMode:         &quietMode,          // Key setting: suppress logs except on failure
        VerboseOnFailure:  true,               // Show debug info when test fails
    })

    // The framework handles all logging automatically
    output, err := options.RunAddonTest()
    if err != nil {
        t.Fatalf("Test failed: %v", err) // Debug logs will be shown
    }
}
```

## Test Logging Options

### Core Settings

| Option | Type | Purpose | Recommended |
|--------|------|---------|-------------|
| `QuietMode` | `*bool` | Suppress logs except on failure | `true` for parallel tests |
| `VerboseOnFailure` | `bool` | Show detailed debug info when test fails | `true` |

### When to Use Each Setting

#### Parallel Tests (Recommended)
```golang
quietMode := true
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    QuietMode:         &quietMode,  // Clean output during parallel execution
    VerboseOnFailure:  true,        // See details on failure
    // ... other options
})
```

#### Local Development/Debugging
```golang
quietMode := false
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    QuietMode:         &quietMode,  // See all output immediately
    VerboseOnFailure:  true,        // Additional context on failure
    // ... other options
})
```

#### CI/CD Environments
```golang
// Automatically detect CI environment
isCI := os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
quietMode := isCI

options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    QuietMode:         &quietMode,  // Quiet in CI, verbose locally
    VerboseOnFailure:  true,        // Always show failure details
    // ... other options
})
```

## Documentation Structure

### For Test Writers
- [User Guide](logging/user-guide.md) - When and how to configure logging options
- [Examples](logging/examples.md) - Real-world testing scenarios
- [Troubleshooting](logging/troubleshooting.md) - Common configuration issues

### For Framework Developers
- [Developer Guide](logging/developer-guide.md) - Logger implementation and architecture
- [Configuration Reference](logging/configuration.md) - Complete API reference

## Package-Specific Configuration

### testaddons Package

```golang
options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
    Testing:           t,
    QuietMode:         &[]bool{true}[0],     // Recommended for addon tests
    VerboseOnFailure:  true,
    // ... addon-specific options
})
```

### testhelper Package

```golang
options := testhelper.TestOptionsDefault(&testhelper.TestOptions{
    Testing:           t,
    TerraformDir:      "examples/basic",
    QuietMode:         &[]bool{true}[0],     // Clean parallel execution
    VerboseOnFailure:  true,
    // ... terraform-specific options
})
```

### testprojects Package

```golang
options := testprojects.TestProjectsOptionsDefault(&testprojects.TestProjectsOptions{
    Testing:           t,
    QuietMode:         &[]bool{true}[0],     // Recommended for project tests
    VerboseOnFailure:  true,
    // ... project-specific options
})
```

### testschematic Package

```golang
options := testschematic.TestSchematicOptionsDefault(&testschematic.TestSchematicOptions{
    Testing:           t,
    QuietMode:         &[]bool{true}[0],     // Clean schematics output
    VerboseOnFailure:  true,
    // ... schematics-specific options
})
```

## What You Get

When you configure logging options correctly:

- ✅ **Clean parallel test execution** - No output noise during successful tests
- ✅ **Automatic progress tracking** - See high-level phases like "🔄 Deploying infrastructure"
- ✅ **Detailed failure diagnostics** - Full debug logs when tests fail with enhanced error handling
- ✅ **CI-friendly output** - Appropriate verbosity for automated environments
- ✅ **Color-coded messages** - Easy to scan success/warning/error indicators
- ✅ **Smart phase detection** - Automatic recognition of test phases with configurable patterns
- ✅ **Enhanced error methods** - CriticalError, FatalError, and ErrorWithContext for different failure scenarios
- ✅ **Framework-specific optimization** - Specialized loggers for addon, project, terraform, and schematics testing

## Common Patterns

### Standard Parallel Test
```golang
func TestStandardPattern(t *testing.T) {
    t.Parallel()
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        QuietMode:         &[]bool{true}[0],
        VerboseOnFailure:  true,
        // ... test configuration
    })
}
```

### Debug-Friendly Test
```golang
func TestWithDebugging(t *testing.T) {
    // Omit t.Parallel() for debugging
    options := testaddons.TestAddonOptionsDefault(&testaddons.TestAddonOptions{
        Testing:           t,
        QuietMode:         &[]bool{false}[0], // See all output immediately
        VerboseOnFailure:  true,
        // ... test configuration
    })
}
```

See the [User Guide](logging/user-guide.md) for detailed guidance and [Examples](logging/examples.md) for complete test scenarios.
